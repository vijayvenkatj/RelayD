package router

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/vijayvenkatj/relayd/internal/proxy"
	"github.com/vijayvenkatj/relayd/internal/upstream"
)

func BenchmarkServeHTTP(b *testing.B) {
	r := benchmarkProxyRouter()
	req, err := http.NewRequest(http.MethodGet, "http://relay.local/api/orders", nil)
	if err != nil {
		b.Fatalf("failed to create request: %v", err)
	}
	w := newDiscardResponseWriter()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w.reset()
		r.ServeHTTP(w, req)
		if w.status != http.StatusOK {
			b.Fatalf("unexpected status code: got %d, want %d", w.status, http.StatusOK)
		}
	}
}

func benchmarkProxyRouter() *Router {
	targetURL, err := url.Parse("http://upstream.local")
	if err != nil {
		panic(err)
	}

	reverseProxy := proxy.NewReverseProxy(targetURL, proxy.NewTransport())
	reverseProxy.Transport = roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode:    http.StatusOK,
			Header:        make(http.Header),
			Body:          io.NopCloser(strings.NewReader("ok")),
			ContentLength: 2,
		}, nil
	})

	backend := &upstream.Backend{
		URL:          targetURL,
		ReverseProxy: reverseProxy,
	}
	backend.Alive.Store(true)

	group := upstream.NewBackendGroup("/api/*", []*upstream.Backend{backend})

	r := NewRouter()
	r.Load([]*upstream.BackendGroup{group})
	return r
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type discardResponseWriter struct {
	header http.Header
	status int
}

func newDiscardResponseWriter() *discardResponseWriter {
	return &discardResponseWriter{
		header: make(http.Header),
	}
}

func (w *discardResponseWriter) Header() http.Header {
	return w.header
}

func (w *discardResponseWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return len(data), nil
}

func (w *discardResponseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
}

func (w *discardResponseWriter) Flush() {}

func (w *discardResponseWriter) reset() {
	clear(w.header)
	w.status = 0
}
