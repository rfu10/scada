// Package driver defines the DriverClient interface and wire types shared between
// the Go operator and the HTTP driver contract.  Controllers import only this
// package; the HTTP transport is in client.go.
package driver

import "context"

// TagAddress identifies a single tag to read or subscribe to.
type TagAddress struct {
	Address  string `json:"address"`
	DataType string `json:"dataType"`
}

// TagValue is a timestamped reading from (or value written to) a tag.
type TagValue struct {
	Address     string      `json:"address"`
	Value       interface{} `json:"value"`
	Quality     string      `json:"quality"`     // GOOD | BAD | UNCERTAIN
	TimestampNs int64       `json:"timestampNs"` // Unix nanoseconds; 0 = driver did not timestamp
}

// ConnectRequest is sent to POST /api/v1/connect.
type ConnectRequest struct {
	Endpoint string            `json:"endpoint"`
	Params   map[string]string `json:"params,omitempty"`
}

// ConnectResponse is the reply from POST /api/v1/connect.
type ConnectResponse struct {
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
	DeviceInfo string `json:"deviceInfo,omitempty"`
}

// WriteResponse is the reply from POST /api/v1/write.
type WriteResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// HealthStatus mirrors driver.v1.HealthResponse.Status.
type HealthStatus string

const (
	StatusConnected    HealthStatus = "CONNECTED"
	StatusDisconnected HealthStatus = "DISCONNECTED"
	StatusError        HealthStatus = "ERROR"
	StatusUnknown      HealthStatus = "UNKNOWN"
)

// HealthResponse is the reply from GET /api/v1/health.
type HealthResponse struct {
	Status  HealthStatus `json:"status"`
	Message string       `json:"message,omitempty"`
}

// DriverClient is the interface all controllers use to talk to a protocol driver.
// The concrete HTTP implementation lives in client.go; tests may substitute a fake.
type DriverClient interface {
	Connect(ctx context.Context, endpoint string, params map[string]string) (*ConnectResponse, error)
	Read(ctx context.Context, tags []TagAddress) ([]TagValue, error)
	Write(ctx context.Context, values []TagValue) error
	Health(ctx context.Context) (*HealthResponse, error)
}
