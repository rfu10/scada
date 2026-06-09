package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SetpointSpec declares the desired value for a Tag — the "spec.replicas" of SCADA.
type SetpointSpec struct {
	// TagRef is the Tag this setpoint drives.
	TagRef corev1.LocalObjectReference `json:"tagRef"`

	// Value is the desired value as a string.
	// Numbers, booleans ("true"/"false"), and quoted strings are all valid.
	Value string `json:"value"`

	// Mode controls application behaviour.
	// "once"       — write on creation or spec change only.
	// "continuous" — re-write on every reconcile loop (holds the value against local overrides).
	// +kubebuilder:validation:Enum=once;continuous
	// +kubebuilder:default=once
	Mode string `json:"mode,omitempty"`
}

// SetpointStatus is the last observed write outcome.
type SetpointStatus struct {
	// Applied is true when Value was last written successfully.
	Applied bool `json:"applied,omitempty"`

	// LastApplied is when the most recent write completed.
	// +optional
	LastApplied *metav1.Time `json:"lastApplied,omitempty"`

	// ActualValue is the tag value read back immediately after the write.
	// +optional
	ActualValue string `json:"actualValue,omitempty"`

	// Conditions: Applied.
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=sp,categories=scada
// +kubebuilder:printcolumn:name="Tag",type=string,JSONPath=".spec.tagRef.name"
// +kubebuilder:printcolumn:name="Value",type=string,JSONPath=".spec.value"
// +kubebuilder:printcolumn:name="Mode",type=string,JSONPath=".spec.mode"
// +kubebuilder:printcolumn:name="Applied",type=boolean,JSONPath=".status.applied"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"

// Setpoint declares the desired value for a Tag and drives it onto the PLC.
type Setpoint struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SetpointSpec   `json:"spec,omitempty"`
	Status SetpointStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SetpointList contains a list of Setpoint.
type SetpointList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Setpoint `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Setpoint{}, &SetpointList{})
}
