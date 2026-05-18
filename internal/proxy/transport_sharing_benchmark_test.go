package proxy

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
)

func BenchmarkTransportSharing(b *testing.B) {
	stats := &upstreamConnStats{}
	upstream := newBenchmarkUpstream(stats)
	defer upstream.Close()

	targetURL, err := url.Parse(upstream.URL)
	if err != nil {
		b.Fatalf("failed to parse upstream url: %v", err)
	}

	const proxyCount = 64

	b.Run("shared-transport", func(b *testing.B) {
		runTransportBenchmark(b, targetURL, proxyCount, true, stats)
	})

	b.Run("transport-per-proxy", func(b *testing.B) {
		runTransportBenchmark(b, targetURL, proxyCount, false, stats)
	})
}

func runTransportBenchmark(
	b *testing.B,
	targetURL *url.URL,
	proxyCount int,
	shared bool,
	stats *upstreamConnStats,
) {
	transports := make([]*http.Transport, 0, proxyCount)
	proxies := make([]http.Handler, 0, proxyCount)

	if shared {
		transport := NewTransport()
		transports = append(transports, transport)
		for i := 0; i < proxyCount; i++ {
			proxies = append(proxies, NewReverseProxy(targetURL, transport))
		}
	} else {
		for i := 0; i < proxyCount; i++ {
			transport := NewTransport()
			transports = append(transports, transport)
			proxies = append(proxies, NewReverseProxy(targetURL, transport))
		}
	}

	var nextProxy atomic.Uint64
	var badStatus atomic.Int64
	before := stats.Load()

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		w := newBenchmarkResponseWriter()
		for pb.Next() {
			req := httptest.NewRequest(http.MethodGet, "http://relay.local/bench", nil)
			idx := nextProxy.Add(1) % uint64(proxyCount)

			w.reset()
			proxies[idx].ServeHTTP(w, req)
			if w.status != http.StatusOK {
				badStatus.Add(1)
			}
		}
	})

	b.StopTimer()
	if badStatus.Load() != 0 {
		b.Fatalf("unexpected status codes observed: %d", badStatus.Load())
	}
	after := stats.Load()
	b.ReportMetric(float64(after-before)/float64(b.N), "upstream_conns/op")

	for _, transport := range transports {
		transport.CloseIdleConnections()
	}
}

func newBenchmarkUpstream(stats *upstreamConnStats) *httptest.Server {
	upstream := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "ok")
	}))
	upstream.Config.ConnState = stats.OnConnState
	upstream.Start()
	return upstream
}

type upstreamConnStats struct {
	newConnections atomic.Int64
}

func (s *upstreamConnStats) OnConnState(_ net.Conn, state http.ConnState) {
	if state == http.StateNew {
		s.newConnections.Add(1)
	}
}

func (s *upstreamConnStats) Load() int64 {
	return s.newConnections.Load()
}

type benchmarkResponseWriter struct {
	header http.Header
	status int
}

func newBenchmarkResponseWriter() *benchmarkResponseWriter {
	return &benchmarkResponseWriter{header: make(http.Header)}
}

func (w *benchmarkResponseWriter) Header() http.Header {
	return w.header
}

func (w *benchmarkResponseWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return len(data), nil
}

func (w *benchmarkResponseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
}

func (w *benchmarkResponseWriter) Flush() {}

func (w *benchmarkResponseWriter) reset() {
	clear(w.header)
	w.status = 0
}
