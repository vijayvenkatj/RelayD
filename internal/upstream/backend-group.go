package upstream

import (
	"math/rand/v2"
	"sync/atomic"
)

type BackendGroup struct {
	Backends       []*Backend
	requestCounter atomic.Int32
}

func NewBackendGroup(backends []*Backend) *BackendGroup {
	backendGroup := &BackendGroup{
		Backends: backends,
	}
	backendGroup.requestCounter.Store(0)

	return backendGroup
}

func (group *BackendGroup) RoundRobin() *Backend {

	if len(group.Backends) == 0 {
		return nil
	}

	next := group.requestCounter.Add(1)
	idx := int(next) % len(group.Backends)
	return group.Backends[idx]
}

func (group *BackendGroup) Random() *Backend {
	if len(group.Backends) == 0 {
		return nil
	}

	idx := rand.IntN(len(group.Backends))
	return group.Backends[idx]
}
