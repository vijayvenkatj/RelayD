package upstream

import (
	"fmt"
	"testing"
)

func BenchmarkRoundRobin(b *testing.B) {
	testCases := []struct {
		name           string
		backendCount   int
		unhealthyEvery int
	}{
		{name: "1-all-healthy", backendCount: 1},
		{name: "10-all-healthy", backendCount: 10},
		{name: "100-all-healthy", backendCount: 100},
		{name: "10-half-unhealthy", backendCount: 10, unhealthyEvery: 2},
		{name: "100-half-unhealthy", backendCount: 100, unhealthyEvery: 2},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			group := benchmarkBackendGroup(tc.backendCount, tc.unhealthyEvery)

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				backend := group.RoundRobin()
				if backend == nil {
					b.Fatal("RoundRobin returned nil backend")
				}
			}
		})
	}
}

func benchmarkBackendGroup(backendCount int, unhealthyEvery int) *BackendGroup {
	backends := make([]*Backend, 0, backendCount)

	for i := 0; i < backendCount; i++ {
		backend := &Backend{}
		backend.Alive.Store(true)

		if unhealthyEvery > 0 && i%unhealthyEvery == 0 {
			backend.Alive.Store(false)
		}

		backends = append(backends, backend)
	}

	return NewBackendGroup(fmt.Sprintf("/bench-%d/*", backendCount), backends)
}
