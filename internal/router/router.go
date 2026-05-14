package router

import (
	"log"
	"net/http"

	"github.com/vijayvenkatj/relayd/internal/upstream"
)

type Router struct {
	BackendGroup map[string]*upstream.BackendGroup
}

func NewRouter() *Router {

	groups := make(map[string]*upstream.BackendGroup)
	router := &Router{
		BackendGroup: groups,
	}

	return router
}
func (router *Router) Load(groups map[string]*upstream.BackendGroup) {
	router.BackendGroup = groups
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Handle this later
}

// GetBackend gets the backend responsible for the current request
func (router *Router) GetBackend(route string) *upstream.Backend {

	if _, exists := router.BackendGroup[route]; !exists {
		log.Println("Invalid route")
		return nil
	}

	backendGroup := router.BackendGroup[route]
	backend := backendGroup.RoundRobin()

	return backend
}
