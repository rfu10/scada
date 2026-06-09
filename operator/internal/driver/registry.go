package driver

import "sync"

// Registry caches DriverClient instances keyed by driver base URL so controllers
// share one HTTP client per driver pod rather than creating a new one per reconcile.
type Registry struct {
	mu      sync.RWMutex
	clients map[string]DriverClient
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{clients: make(map[string]DriverClient)}
}

// GetOrCreate returns the cached client for url, creating one if absent.
func (r *Registry) GetOrCreate(url string) DriverClient {
	r.mu.RLock()
	if c, ok := r.clients[url]; ok {
		r.mu.RUnlock()
		return c
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()
	if c, ok := r.clients[url]; ok {
		return c
	}
	c := NewHTTPClient(url)
	r.clients[url] = c
	return c
}
