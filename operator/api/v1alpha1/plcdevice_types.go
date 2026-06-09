package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PLCDeviceSpec defines the desired state of a PLCDevice.
type PLCDeviceSpec struct {
	// Protocol is the industrial protocol spoken by this device.
	// +kubebuilder:validation:Enum=ethernet-ip;opc-ua;modbus-tcp;modbus-rtu;dnp3
	Protocol string `json:"protocol"`

	// Endpoint is the network address of the PLC (host:port).
	// EtherNet/IP default port: 44818. OPC-UA default: 4840.
	Endpoint string `json:"endpoint"`

	// DriverRef locates the driver pod that speaks Protocol.
	// Defaults to service "scada-driver-{protocol}" in the same namespace on port 8080.
	// +optional
	DriverRef *DriverRef `json:"driverRef,omitempty"`

	// Params contains protocol-specific connection parameters.
	// EtherNet/IP: slot="0", path="1,0"
	// OPC-UA: security_policy="None", application_uri="urn:…"
	// +optional
	Params map[string]string `json:"params,omitempty"`

	// ScanInterval is the default poll rate for Tags on this device.
	// +optional
	// +kubebuilder:default="1s"
	ScanInterval string `json:"scanInterval,omitempty"`
}

// DriverRef locates the HTTP driver service for a given PLCDevice.
type DriverRef struct {
	// ServiceName overrides the default "scada-driver-{protocol}" service name.
	// +optional
	ServiceName string `json:"serviceName,omitempty"`

	// Namespace overrides the device's own namespace for cross-namespace drivers.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Port defaults to 8080.
	// +optional
	// +kubebuilder:default=8080
	Port int32 `json:"port,omitempty"`
}

// PLCDeviceStatus defines the observed state of a PLCDevice.
type PLCDeviceStatus struct {
	// Connected is true when the driver has an active session with the device.
	Connected bool `json:"connected,omitempty"`

	// LastConnected is the timestamp of the most recent successful Connect call.
	// +optional
	LastConnected *metav1.Time `json:"lastConnected,omitempty"`

	// DeviceInfo is the vendor/product string returned by the driver on connect.
	// +optional
	DeviceInfo string `json:"deviceInfo,omitempty"`

	// DriverEndpoint is the resolved HTTP base URL of the driver service.
	// +optional
	DriverEndpoint string `json:"driverEndpoint,omitempty"`

	// Conditions use standard K8s condition conventions.
	// Types: Connected, Ready.
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=plc,categories=scada
// +kubebuilder:printcolumn:name="Protocol",type=string,JSONPath=".spec.protocol"
// +kubebuilder:printcolumn:name="Endpoint",type=string,JSONPath=".spec.endpoint"
// +kubebuilder:printcolumn:name="Connected",type=boolean,JSONPath=".status.connected"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"

// PLCDevice represents a physical programmable logic controller.
type PLCDevice struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PLCDeviceSpec   `json:"spec,omitempty"`
	Status PLCDeviceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PLCDeviceList contains a list of PLCDevice.
type PLCDeviceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PLCDevice `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PLCDevice{}, &PLCDeviceList{})
}
