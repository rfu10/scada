package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AlarmCondition is a single threshold expression evaluated against a Tag.
type AlarmCondition struct {
	// TagRef is the Tag whose value is compared.
	TagRef corev1.LocalObjectReference `json:"tagRef"`

	// Operator is the comparison: gt ≥ lt ≤ eq ne.
	// +kubebuilder:validation:Enum=gt;gte;lt;lte;eq;ne
	Operator string `json:"operator"`

	// Threshold is the right-hand side of the comparison, as a string.
	Threshold string `json:"threshold"`
}

// AlarmPolicySpec defines when an alarm fires and how it is classified.
type AlarmPolicySpec struct {
	// Conditions are ANDed together; all must be true to trigger the alarm.
	// +kubebuilder:validation:MinItems=1
	Conditions []AlarmCondition `json:"conditions"`

	// Severity classifies the alarm for HMI filtering and notification routing.
	// +kubebuilder:validation:Enum=critical;major;minor;warning;info
	// +kubebuilder:default=warning
	Severity string `json:"severity,omitempty"`

	// Message is a human-readable description shown in the HMI alarm banner.
	// +optional
	Message string `json:"message,omitempty"`

	// Deadband suppresses re-trigger flapping.
	// A numeric value; the condition must recover past (threshold ± deadband) before clearing.
	// +optional
	// +kubebuilder:default="0"
	Deadband string `json:"deadband,omitempty"`
}

// AlarmPolicyStatus records the current and historical alarm state.
type AlarmPolicyStatus struct {
	// Active is true when all conditions are currently satisfied.
	Active bool `json:"active,omitempty"`

	// LastTriggered is when the alarm most recently became active.
	// +optional
	LastTriggered *metav1.Time `json:"lastTriggered,omitempty"`

	// LastCleared is when the alarm most recently became inactive.
	// +optional
	LastCleared *metav1.Time `json:"lastCleared,omitempty"`

	// TriggerCount is the lifetime number of activations.
	TriggerCount int64 `json:"triggerCount,omitempty"`

	// Conditions: Evaluating, Active.
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=alarm,categories=scada
// +kubebuilder:printcolumn:name="Severity",type=string,JSONPath=".spec.severity"
// +kubebuilder:printcolumn:name="Active",type=boolean,JSONPath=".status.active"
// +kubebuilder:printcolumn:name="Triggers",type=integer,JSONPath=".status.triggerCount"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"

// AlarmPolicy monitors Tag values and fires a K8s Event when conditions are met.
type AlarmPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AlarmPolicySpec   `json:"spec,omitempty"`
	Status AlarmPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AlarmPolicyList contains a list of AlarmPolicy.
type AlarmPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AlarmPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AlarmPolicy{}, &AlarmPolicyList{})
}
