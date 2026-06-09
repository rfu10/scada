package controller

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	scadav1alpha1 "github.com/rfu10/scada/operator/api/v1alpha1"
	"github.com/rfu10/scada/operator/internal/driver"
)

// SetpointReconciler reconciles a Setpoint object.
type SetpointReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Registry *driver.Registry
}

// +kubebuilder:rbac:groups=scada.io,resources=setpoints,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=scada.io,resources=setpoints/status,verbs=get;update;patch

func (r *SetpointReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var sp scadav1alpha1.Setpoint
	if err := r.Get(ctx, req.NamespacedName, &sp); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// For mode=once, skip if already applied and spec hasn't changed (generation unchanged).
	if sp.Spec.Mode == "once" && sp.Status.Applied {
		return ctrl.Result{}, nil
	}

	var tag scadav1alpha1.Tag
	if err := r.Get(ctx, client.ObjectKey{Name: sp.Spec.TagRef.Name, Namespace: sp.Namespace}, &tag); err != nil {
		return r.failSP(ctx, &sp, "TagNotFound", err.Error())
	}

	var device scadav1alpha1.PLCDevice
	if err := r.Get(ctx, client.ObjectKey{Name: tag.Spec.DeviceRef.Name, Namespace: tag.Namespace}, &device); err != nil {
		return r.failSP(ctx, &sp, "DeviceNotFound", err.Error())
	}

	if !device.Status.Connected {
		return r.failSP(ctx, &sp, "DeviceNotConnected",
			fmt.Sprintf("PLCDevice %s is not connected", device.Name))
	}

	dc := r.Registry.GetOrCreate(device.Status.DriverEndpoint)

	log.Info("writing setpoint", "tag", tag.Spec.Address, "value", sp.Spec.Value)
	if err := dc.Write(ctx, []driver.TagValue{
		{Address: tag.Spec.Address, Value: sp.Spec.Value, Quality: "GOOD"},
	}); err != nil {
		return r.failSP(ctx, &sp, "WriteError", err.Error())
	}

	now := metav1.Now()
	sp.Status.Applied = true
	sp.Status.LastApplied = &now
	sp.Status.ActualValue = tag.Status.Value
	meta.SetStatusCondition(&sp.Status.Conditions, metav1.Condition{
		Type:    "Applied",
		Status:  metav1.ConditionTrue,
		Reason:  "WriteSuccess",
		Message: fmt.Sprintf("wrote %s to %s", sp.Spec.Value, tag.Spec.Address),
	})

	if err := r.Status().Update(ctx, &sp); err != nil {
		return ctrl.Result{}, err
	}

	if sp.Spec.Mode == "continuous" {
		return ctrl.Result{RequeueAfter: scanRate(tag.Spec.ScanRate, device.Spec.ScanInterval)}, nil
	}
	return ctrl.Result{}, nil
}

func (r *SetpointReconciler) failSP(ctx context.Context, sp *scadav1alpha1.Setpoint, reason, msg string) (ctrl.Result, error) {
	meta.SetStatusCondition(&sp.Status.Conditions, metav1.Condition{
		Type:    "Applied",
		Status:  metav1.ConditionFalse,
		Reason:  reason,
		Message: msg,
	})
	_ = r.Status().Update(ctx, sp)
	return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
}

func (r *SetpointReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&scadav1alpha1.Setpoint{}).
		Complete(r)
}
