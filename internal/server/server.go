package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/vijayvenkatj/relayd/internal/config"
	"github.com/vijayvenkatj/relayd/internal/proxy"
	"github.com/vijayvenkatj/relayd/internal/router"
)

const (
	reloadDrainTimeout = 5 * time.Second
	drainPollInterval  = 50 * time.Millisecond
)

type Server struct {
	ConfigPath string
	Router     atomic.Pointer[router.Router]

	HTTPServer   *http.Server
	Transport    *http.Transport
	CancelCtx    context.CancelFunc
	ReloadCancel context.CancelFunc
}

func (server *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	router := server.Router.Load()
	if router == nil {
		http.Error(w, "router unavailable", http.StatusInternalServerError)
		return
	}
	router.ServeHTTP(w, req)
}

func (server *Server) ListenAndServe() {

	configData, err := os.ReadFile(server.ConfigPath)
	if err != nil {
		log.Fatal("error loading config file: ", err.Error())
		return
	}

	cfg, err := config.Parse(configData)
	if err != nil {
		log.Fatal("error parsing config file: ", err.Error())
		return
	}

	addr := fmt.Sprintf(":%s", cfg.Server.Port)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal("error starting up a listener: ", err.Error())
		return
	}

	lifecycleCtx, lifecycleCancel := context.WithCancel(context.Background())
	server.CancelCtx = lifecycleCancel
	reloadCtx, reloadCancel := context.WithCancel(lifecycleCtx)
	server.ReloadCancel = reloadCancel

	// Shared transport to be injected
	transport := proxy.NewTransport()
	server.Transport = transport

	// Shared HTTP client to be injected
	httpClient := proxy.NewHTTPClient(transport)

	// Backend Groups from the User defined config
	backendGrps := cfg.BackendGroups(reloadCtx, transport, httpClient)

	// Router to handle Handlers
	rtr := router.NewRouter()
	rtr.Load(backendGrps)

	server.Router.Store(rtr)

	httpServer := &http.Server{
		Handler: server,

		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	server.HTTPServer = httpServer

	// Setup signal for config reloads
	signalChan := make(chan os.Signal, 1)
	shutdownDone := make(chan struct{})
	var shutdownStarted atomic.Bool
	signal.Notify(signalChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signalChan)

	go func() {
		for sig := range signalChan {
			switch sig {
			case syscall.SIGHUP:
				log.Println("reloading config")

				configData, err := os.ReadFile(server.ConfigPath)
				if err != nil {
					log.Println("error loading config file: ", err.Error())
					continue
				}

				cfg, err := config.Parse(configData)
				if err != nil {
					log.Println("error parsing config file: ", err.Error())
					continue
				}

				newReloadCtx, newReloadCancel := context.WithCancel(lifecycleCtx)
				backendGrps := cfg.BackendGroups(newReloadCtx, transport, httpClient)

				newRouter := router.NewRouter()
				newRouter.Load(backendGrps)

				oldRouter := server.Router.Load()
				oldReloadCancel := server.ReloadCancel

				server.Router.Store(newRouter)
				server.ReloadCancel = newReloadCancel

				if oldReloadCancel != nil {
					oldReloadCancel()
				}
				retireRouter(oldRouter, reloadDrainTimeout)
				if server.Transport != nil {
					server.Transport.CloseIdleConnections()
				}

				log.Println("reloaded config")
			case syscall.SIGINT, syscall.SIGTERM:
				if shutdownStarted.CompareAndSwap(false, true) {
					log.Println("shutdown signal received")
					server.GracefulShutdown()
					close(shutdownDone)
				}
				return
			}
		}
	}()

	log.Println("server listening on port ", addr)
	if err := server.HTTPServer.Serve(listener); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			if shutdownStarted.Load() {
				<-shutdownDone
			}
			return
		}
		log.Fatal("error listening: ", err.Error())
	}

}

func (server *Server) GracefulShutdown() {
	timeoutCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if server.HTTPServer != nil {
		if err := server.HTTPServer.Shutdown(timeoutCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Println("error during graceful shutdown: ", err.Error())
		}
	}
	log.Println("http server shutdown!")

	if server.ReloadCancel != nil {
		server.ReloadCancel()
	}
	if server.CancelCtx != nil {
		server.CancelCtx()
	}
	if server.Transport != nil {
		server.Transport.CloseIdleConnections()
	}

	retireRouter(server.Router.Load(), 0)

	log.Println("server shutdown gracedfully!")
}

func retireRouter(r *router.Router, timeout time.Duration) {
	if r == nil {
		return
	}

	if timeout > 0 {
		deadline := time.Now().Add(timeout)
		for activeRequests(r) > 0 && time.Now().Before(deadline) {
			time.Sleep(drainPollInterval)
		}
	}

	closeRouterBackends(r)
	log.Println("router retired!")
}

func activeRequests(r *router.Router) int32 {
	if r == nil {
		return 0
	}

	var total int32
	for _, grp := range r.BackendGroup {
		for _, backend := range grp.Backends {
			total += backend.Requests.Load()
		}
	}
	return total
}

func closeRouterBackends(r *router.Router) {
	if r == nil {
		return
	}

	for _, grp := range r.BackendGroup {
		for _, backend := range grp.Backends {
			backend.Close()
		}
	}
}
