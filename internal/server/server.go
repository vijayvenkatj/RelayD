package server

import (
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

type Server struct {
	ConfigPath string
	Router     atomic.Pointer[router.Router]
	HTTPServer *http.Server
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

	// Shared transport to be injected
	transport := proxy.NewTransport()

	// Shared HTTP client to be injected
	httpClient := proxy.NewHTTPClient(transport)

	// Backend Groups from the User defined config
	backendGrps := cfg.BackendGroups(transport, httpClient)

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
	signal.Notify(signalChan, syscall.SIGHUP)

	go func() {
		for range signalChan {
			log.Println("reloading config")

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

			backendGrps := cfg.BackendGroups(transport, httpClient)

			newRouter := router.NewRouter()
			newRouter.Load(backendGrps)

			server.Router.Store(newRouter)

			log.Println("reloaded config")
		}
	}()

	log.Println("server listening on port ", addr)
	if err := server.HTTPServer.Serve(listener); err != nil {
		log.Fatal("error listening: ", err.Error())
	}

}
