package upstream

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
	"time"
)

type Backend struct {
	URL          *url.URL
	ReverseProxy *httputil.ReverseProxy
	Alive        atomic.Bool

	httpClient *http.Client
}

func NewBackend(hostUrl *url.URL, reverseProxy *httputil.ReverseProxy, httpClient *http.Client) *Backend {
	backend := &Backend{
		URL:          hostUrl,
		ReverseProxy: reverseProxy,
		httpClient:   httpClient,
	}
	// Optimistic initialisation
	backend.Alive.Store(true)

	go backend.HealthCheck()

	return backend
}

// HealthCheck periodically checks the endpoint for health
func (backend *Backend) HealthCheck() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		client := backend.httpClient

		resp, err := client.Get(backend.URL.String())
		if err != nil {
			backend.Alive.Store(false)
			continue
		}
		resp.Body.Close()

		state := resp.StatusCode >= 200 && resp.StatusCode < 500
		backend.Alive.Store(state)
	}
}
