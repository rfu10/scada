package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TagSpec defines a single named data point on a PLCDevice.
type TagSpec struct {
	// DeviceRef is the PLCDevice that owns this tag.
	DeviceRef corev1.LocalObjectReference `json:"deviceRef"`

	// Address is the tag's location on the PLC.
	// EtherNet/IP:  "Program:MainProgram.Motor1.Speed"
	// OPC-UA:       "ns=2;s=PLC1.Motor1.Speed"
	// Modbus:       "holding:40001"
	Address string `json:"address"`

	// DataType is the PLC-native data type.
	// +kubebuilder:validation:Enum=BOOL;SINT;INT;DINT;LINT;REAL;LREAL;STRING;WORD;DWORD;LWORD
	DataType string `json:"dataType"`

	// ScanRate overrides the device's ScanInterval for this tag.
	// +optional
	ScanRate string `json:"scanRate,omitempty"`

	// EngineeringUnit is the unit of measure displayed in the HMI ("RPM", "degC", "bar").
	// +optional
	EngineeringUnit string `json:"engineeringUnit,omitempty"`

	// Description is a human-readable label for HMI display.
	// +optional
	Description string `json:"description,omitempty"`

	// HistorianEnabled sends every read value to the configured HistorianSink.
	// +optional
	// +kubebuilder:default=false
	HistorianEnabled bool `json:"historianEnabled,omitempty"`
}

// TagStatus is the last observed value for a Tag.
type TagStatus struct {
	// Value is the most recent read, serialised to string.
	// +optional
	Value string `json:"value,omitempty"`

	// Quality follows OPC-UA conventions: GOOD, BAD, UNCERTAIN, UNKNOWN.
	// +optional
	// +kubebuilder:validation:Enum=GOOD;BAD;UNCERTAIN;UNKNOWN
	Quality string `json:"quality,omitempty"`

	// Timestamp is when Value was last updated.
	// +optional
	Timestamp *metav1.Time `json:"timestamp,omitempty"`

	// Conditions: Ready.
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=stag,categories=scada
// +kubebuilder:printcolumn:name="Device",type=string,JSONPath=".spec.deviceRef.name"
// +kubebuilder:printcolumn:name="Address",type=string,JSONPath=".spec.address"
// +kubebuilder:printcolumn:name="Value",type=string,JSONPath=".status.value"
// +kubebuilder:printcolumn:name="Quality",type=string,JSONPath=".status.quality"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"

// Tag represents a named data point on a PLCDevice.
type Tag struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TagSpec   `json:"spec,omitempty"`
	Status TagStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TagList contains a list of Tag.
type TagList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Tag `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Tag{}, &TagList{})
}
