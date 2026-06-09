package controller

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	scadav1alpha1 "github.com/rfu10/scada/operator/api/v1alpha1"
)

// AlarmPolicyReconciler evaluates alarm conditions against current Tag statuses.
// It does not call the driver directly — it reads Tag.Status.Value which the
// TagReconciler keeps up to date.
type AlarmPolicyReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=scada.io,resources=alarmpolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=scada.io,resources=alarmpolicies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=scada.io,resources=tags,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *AlarmPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var alarm scadav1alpha1.AlarmPolicy
	if err := r.Get(ctx, req.NamespacedName, &alarm); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	wasActive := alarm.Status.Active
	allTrue := true

	for _, cond := range alarm.Spec.Conditions {
		var tag scadav1alpha1.Tag
		if err := r.Get(ctx, client.ObjectKey{Name: cond.TagRef.Name, Namespace: alarm.Namespace}, &tag); err != nil {
			allTrue = false
			break
		}
		if tag.Status.Quality != "GOOD" || tag.Status.Value == "" {
			allTrue = false
			break
		}
		if !evalCondition(tag.Status.Value, cond.Operator, cond.Threshold) {
			allTrue = false
			break
		}
	}

	now := metav1.Now()

	if allTrue && !wasActive {
		alarm.Status.Active = true
		alarm.Status.LastTriggered = &now
		alarm.Status.TriggerCount++
		meta.SetStatusCondition(&alarm.Status.Conditions, metav1.Condition{
			Type:    "Active",
			Status:  metav1.ConditionTrue,
			Reason:  "ConditionMet",
			Message: alarmMessage(&alarm),
		})
		r.Recorder.Eventf(&alarm, "Warning", fmt.Sprintf("Alarm%s", titleCase(alarm.Spec.Severity)),
			"[%s] %s", alarm.Spec.Severity, alarmMessage(&alarm))
	} else if !allTrue && wasActive {
		alarm.Status.Active = false
		alarm.Status.LastCleared = &now
		meta.SetStatusCondition(&alarm.Status.Conditions, metav1.Condition{
			Type:    "Active",
			Status:  metav1.ConditionFalse,
			Reason:  "ConditionCleared",
			Message: "alarm condition no longer met",
		})
		r.Recorder.Eventf(&alarm, "Normal", "AlarmCleared", "alarm cleared")
	}

	if err := r.Status().Update(ctx, &alarm); err != nil {
		return ctrl.Result{}, err
	}
	// Poll at 1s by default; alarm conditions are evaluated on every reconcile.
	return ctrl.Result{RequeueAfter: time.Second}, nil
}

func (r *AlarmPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&scadav1alpha1.AlarmPolicy{}).
		Complete(r)
}

// evalCondition compares the string tag value against the threshold using op.
func evalCondition(rawValue, op, threshold string) bool {
	vf, vErr := strconv.ParseFloat(rawValue, 64)
	tf, tErr := strconv.ParseFloat(threshold, 64)

	if vErr == nil && tErr == nil {
		switch op {
		case "gt":
			return vf > tf
		case "gte":
			return vf >= tf
		case "lt":
			return vf < tf
		case "lte":
			return vf <= tf
		case "eq":
			return vf == tf
		case "ne":
			return vf != tf
		}
	}
	// Fall back to string comparison for eq/ne.
	switch op {
	case "eq":
		return rawValue == threshold
	case "ne":
		return rawValue != threshold
	}
	return false
}

func alarmMessage(alarm *scadav1alpha1.AlarmPolicy) string {
	if alarm.Spec.Message != "" {
		return alarm.Spec.Message
	}
	return fmt.Sprintf("alarm %s/%s triggered", alarm.Namespace, alarm.Name)
}

func titleCase(s string) string {
	if len(s) == 0 {
		return s
	}
	return string(s[0]-32) + s[1:]
}
