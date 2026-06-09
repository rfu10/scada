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

// TagReconciler reconciles a Tag object.
type TagReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Registry  *driver.Registry
	Historian HistorianSink
}

// +kubebuilder:rbac:groups=scada.io,resources=tags,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=scada.io,resources=tags/status,verbs=get;update;patch

func (r *TagReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var tag scadav1alpha1.Tag
	if err := r.Get(ctx, req.NamespacedName, &tag); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var device scadav1alpha1.PLCDevice
	if err := r.Get(ctx, client.ObjectKey{Name: tag.Spec.DeviceRef.Name, Namespace: tag.Namespace}, &device); err != nil {
		return r.failTag(ctx, &tag, "DeviceNotFound", err.Error())
	}

	if !device.Status.Connected {
		return r.failTag(ctx, &tag, "DeviceNotConnected",
			fmt.Sprintf("PLCDevice %s is not connected", device.Name))
	}

	if device.Status.DriverEndpoint == "" {
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	dc := r.Registry.GetOrCreate(device.Status.DriverEndpoint)

	values, err := dc.Read(ctx, []driver.TagAddress{
		{Address: tag.Spec.Address, DataType: tag.Spec.DataType},
	})
	if err != nil {
		log.Error(err, "read failed", "address", tag.Spec.Address)
		tag.Status.Quality = "BAD"
		meta.SetStatusCondition(&tag.Status.Conditions, metav1.Condition{
			Type:    "Ready",
			Status:  metav1.ConditionFalse,
			Reason:  "ReadError",
			Message: err.Error(),
		})
		_ = r.Status().Update(ctx, &tag)
		return ctrl.Result{RequeueAfter: scanRate(tag.Spec.ScanRate, device.Spec.ScanInterval)}, nil
	}

	if len(values) > 0 {
		v := values[0]
		now := metav1.Now()
		tag.Status.Value = fmt.Sprintf("%v", v.Value)
		tag.Status.Quality = v.Quality
		tag.Status.Timestamp = &now

		if tag.Spec.HistorianEnabled && r.Historian != nil {
			_ = r.Historian.Record(ctx, "tags", tag.Name, v.Value, v.Quality, now.Time)
		}
	}

	meta.SetStatusCondition(&tag.Status.Conditions, metav1.Condition{
		Type:   "Ready",
		Status: metav1.ConditionTrue,
		Reason: "ReadSuccess",
	})

	if err := r.Status().Update(ctx, &tag); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: scanRate(tag.Spec.ScanRate, device.Spec.ScanInterval)}, nil
}

func (r *TagReconciler) failTag(ctx context.Context, tag *scadav1alpha1.Tag, reason, msg string) (ctrl.Result, error) {
	meta.SetStatusCondition(&tag.Status.Conditions, metav1.Condition{
		Type:    "Ready",
		Status:  metav1.ConditionFalse,
		Reason:  reason,
		Message: msg,
	})
	_ = r.Status().Update(ctx, tag)
	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

func (r *TagReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&scadav1alpha1.Tag{}).
		Complete(r)
}

func scanRate(tagRate, deviceRate string) time.Duration {
	if tagRate != "" {
		return parseDuration(tagRate, time.Second)
	}
	return parseDuration(deviceRate, time.Second)
}
