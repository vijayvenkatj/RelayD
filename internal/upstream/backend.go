package upstream

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
	"time"
)

type Backend struct {
	URL          *url.URL
	ReverseProxy *httputil.ReverseProxy

	Requests atomic.Int32
	Alive    atomic.Bool
	Closed   atomic.Bool

	httpClient *http.Client
}

func NewBackend(ctx context.Context, hostUrl *url.URL, reverseProxy *httputil.ReverseProxy, httpClient *http.Client) *Backend {
	backend := &Backend{
		URL:          hostUrl,
		ReverseProxy: reverseProxy,
		httpClient:   httpClient,
	}
	// Optimistic initialisation
	backend.Alive.Store(true)

	go backend.HealthCheck(ctx)

	return backend
}

func (backend *Backend) Close() {
	if !backend.Closed.CompareAndSwap(false, true) {
		return
	}
	backend.Alive.Store(false)
}

// HealthCheck periodically checks the endpoint for health
func (backend *Backend) HealthCheck(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if backend.Closed.Load() {
				return
			}

			client := backend.httpClient
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, backend.URL.String(), nil)
			if err != nil {
				if !backend.Closed.Load() {
					backend.Alive.Store(false)
				}
				continue
			}

			resp, err := client.Do(req)
			if err != nil {
				if !backend.Closed.Load() {
					backend.Alive.Store(false)
				}
				continue
			}
			resp.Body.Close()

			state := resp.StatusCode >= 200 && resp.StatusCode < 500
			if !backend.Closed.Load() {
				backend.Alive.Store(state)
			}
		}
	}
}
