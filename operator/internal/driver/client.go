package driver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// httpClient implements DriverClient over the HTTP/JSON driver API.
type httpClient struct {
	baseURL string
	http    *http.Client
}

// NewHTTPClient returns a DriverClient that talks to the driver at baseURL.
func NewHTTPClient(baseURL string) DriverClient {
	return &httpClient{
		baseURL: baseURL,
		http: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *httpClient) Connect(ctx context.Context, endpoint string, params map[string]string) (*ConnectResponse, error) {
	req := ConnectRequest{Endpoint: endpoint, Params: params}
	var resp ConnectResponse
	if err := c.post(ctx, "/api/v1/connect", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *httpClient) Read(ctx context.Context, tags []TagAddress) ([]TagValue, error) {
	body := struct {
		Tags []TagAddress `json:"tags"`
	}{Tags: tags}
	var resp struct {
		Values []TagValue `json:"values"`
		Error  string     `json:"error,omitempty"`
	}
	if err := c.post(ctx, "/api/v1/read", body, &resp); err != nil {
		return nil, err
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("driver read error: %s", resp.Error)
	}
	return resp.Values, nil
}

func (c *httpClient) Write(ctx context.Context, values []TagValue) error {
	body := struct {
		Values []TagValue `json:"values"`
	}{Values: values}
	var resp WriteResponse
	if err := c.post(ctx, "/api/v1/write", body, &resp); err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf("driver write error: %s", resp.Error)
	}
	return nil
}

func (c *httpClient) Health(ctx context.Context) (*HealthResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/health", nil)
	if err != nil {
		return nil, err
	}
	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	var resp HealthResponse
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *httpClient) post(ctx context.Context, path string, body, out interface{}) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		msg, _ := io.ReadAll(res.Body)
		return fmt.Errorf("driver returned HTTP %d: %s", res.StatusCode, string(msg))
	}
	return json.NewDecoder(res.Body).Decode(out)
}
