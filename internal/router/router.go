package router

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/vijayvenkatj/relayd/internal/upstream"
)

type Router struct {
	BackendGroup []*upstream.BackendGroup
}

func NewRouter() *Router {

	groups := []*upstream.BackendGroup{}
	router := &Router{
		BackendGroup: groups,
	}

	return router
}
func (r *Router) Load(groups []*upstream.BackendGroup) {
	r.BackendGroup = groups
}

var (
	ErrRouteNotFound = errors.New("route not found")
	ErrNoHealthyHost = errors.New("no healthy host found")
)

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	path := req.URL.Path

	backend, err := r.GetBackend(path)
	if err != nil {
		if err == ErrNoHealthyHost {
			http.Error(w, "no healthy host", http.StatusBadGateway)
			return
		}

		http.NotFound(w, req)
		return
	}

	backend.ReverseProxy.ServeHTTP(w, req)
}

// GetBackend gets the backend responsible for the current request
func (r *Router) GetBackend(route string) (*upstream.Backend, error) {

	var matchedRoute *upstream.BackendGroup = nil

	for _, backendGroup := range r.BackendGroup {
		if strings.HasPrefix(backendGroup.Route, route) {
			if matchedRoute != nil && len(backendGroup.Route) > len(matchedRoute.Route) {
				continue
			}
			matchedRoute = backendGroup
		}
	}

	if matchedRoute == nil {
		log.Println("no matching route:", route)
		return nil, ErrRouteNotFound
	}

	backend := matchedRoute.RoundRobin()

	if backend == nil {
		log.Println("no healthy upstreams for route:", matchedRoute.Route)
		return nil, ErrNoHealthyHost
	}

	return backend, nil
}
