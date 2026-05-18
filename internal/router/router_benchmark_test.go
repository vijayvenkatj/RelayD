package router

import (
	"fmt"
	"testing"

	"github.com/vijayvenkatj/relayd/internal/upstream"
)

func BenchmarkGetBackend(b *testing.B) {
	for _, routeCount := range []int{10, 100, 1000} {
		b.Run(fmt.Sprintf("%d-routes", routeCount), func(b *testing.B) {
			r := benchmarkRouter(routeCount)
			requestPath := fmt.Sprintf("/service-%04d/orders", routeCount-1)

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				backend, err := r.GetBackend(requestPath)
				if err != nil {
					b.Fatalf("GetBackend returned error: %v", err)
				}
				if backend == nil {
					b.Fatal("GetBackend returned nil backend")
				}
			}
		})
	}
}

func benchmarkRouter(routeCount int) *Router {
	groups := make([]*upstream.BackendGroup, 0, routeCount)

	for i := 0; i < routeCount; i++ {
		backend := &upstream.Backend{}
		backend.Alive.Store(true)

		groups = append(groups, upstream.NewBackendGroup(
			fmt.Sprintf("/service-%04d/*", i),
			[]*upstream.Backend{backend},
		))
	}

	r := NewRouter()
	r.Load(groups)
	return r
}
