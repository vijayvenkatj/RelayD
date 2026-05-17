package upstream

import (
	"math/rand/v2"
	"sync/atomic"
)

type BackendGroup struct {
	Route    string
	Backends []*Backend

	requestCounter atomic.Int32
}

func NewBackendGroup(path string, backends []*Backend) *BackendGroup {
	backendGroup := &BackendGroup{
		Route:    path,
		Backends: backends,
	}
	backendGroup.requestCounter.Store(0)

	return backendGroup
}

func (group *BackendGroup) RoundRobin() *Backend {

	backendCount := len(group.Backends)
	if backendCount == 0 {
		return nil
	}

	for i := 0; i < backendCount; i++ {

		next := group.requestCounter.Add(1)
		idx := int(next % int32(backendCount))
		backend := group.Backends[idx]

		if backend.Alive.Load() && !backend.Closed.Load() {
			return backend
		}
	}

	return nil
}

func (group *BackendGroup) Random() *Backend {

	backendCount := len(group.Backends)
	if backendCount == 0 {
		return nil
	}

	for i := 0; i < backendCount; i++ {

		group.requestCounter.Add(1)
		idx := rand.IntN(len(group.Backends))
		backend := group.Backends[idx]

		if backend.Alive.Load() && !backend.Closed.Load() {
			return backend
		}
	}

	return nil
}
