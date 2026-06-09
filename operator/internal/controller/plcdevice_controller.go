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

// PLCDeviceReconciler reconciles a PLCDevice object.
type PLCDeviceReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Registry *driver.Registry
}

// +kubebuilder:rbac:groups=scada.io,resources=plcdevices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=scada.io,resources=plcdevices/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=scada.io,resources=plcdevices/finalizers,verbs=update

func (r *PLCDeviceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var device scadav1alpha1.PLCDevice
	if err := r.Get(ctx, req.NamespacedName, &device); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	driverURL := r.resolveDriverURL(&device)
	device.Status.DriverEndpoint = driverURL

	dc := r.Registry.GetOrCreate(driverURL)

	health, err := dc.Health(ctx)
	if err == nil && health.Status == driver.StatusConnected {
		device.Status.Connected = true
		meta.SetStatusCondition(&device.Status.Conditions, metav1.Condition{
			Type:    "Connected",
			Status:  metav1.ConditionTrue,
			Reason:  "Connected",
			Message: fmt.Sprintf("driver reports connected to %s", device.Spec.Endpoint),
		})
		if err := r.Status().Update(ctx, &device); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: parseDuration(device.Spec.ScanInterval, 30*time.Second)}, nil
	}

	// Not connected — attempt Connect.
	log.Info("connecting to PLC", "endpoint", device.Spec.Endpoint, "driver", driverURL)
	resp, err := dc.Connect(ctx, device.Spec.Endpoint, device.Spec.Params)
	if err != nil || !resp.Success {
		msg := "connection failed"
		if err != nil {
			msg = err.Error()
		} else if resp.Error != "" {
			msg = resp.Error
		}
		device.Status.Connected = false
		meta.SetStatusCondition(&device.Status.Conditions, metav1.Condition{
			Type:    "Connected",
			Status:  metav1.ConditionFalse,
			Reason:  "ConnectionFailed",
			Message: msg,
		})
		if err := r.Status().Update(ctx, &device); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	now := metav1.Now()
	device.Status.Connected = true
	device.Status.LastConnected = &now
	device.Status.DeviceInfo = resp.DeviceInfo
	meta.SetStatusCondition(&device.Status.Conditions, metav1.Condition{
		Type:    "Connected",
		Status:  metav1.ConditionTrue,
		Reason:  "Connected",
		Message: fmt.Sprintf("connected to %s via %s", device.Spec.Endpoint, device.Spec.Protocol),
	})

	if err := r.Status().Update(ctx, &device); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: parseDuration(device.Spec.ScanInterval, 30*time.Second)}, nil
}

func (r *PLCDeviceReconciler) resolveDriverURL(device *scadav1alpha1.PLCDevice) string {
	svc := fmt.Sprintf("scada-driver-%s", device.Spec.Protocol)
	ns := device.Namespace
	port := int32(8080)

	if ref := device.Spec.DriverRef; ref != nil {
		if ref.ServiceName != "" {
			svc = ref.ServiceName
		}
		if ref.Namespace != "" {
			ns = ref.Namespace
		}
		if ref.Port != 0 {
			port = ref.Port
		}
	}
	return fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", svc, ns, port)
}

func (r *PLCDeviceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&scadav1alpha1.PLCDevice{}).
		Complete(r)
}

func parseDuration(s string, def time.Duration) time.Duration {
	if s == "" {
		return def
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return def
	}
	return d
}
