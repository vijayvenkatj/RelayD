package server

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/vijayvenkatj/relayd/internal/config"
	"github.com/vijayvenkatj/relayd/internal/proxy"
	"github.com/vijayvenkatj/relayd/internal/router"
)

type Server struct {
	ConfigPath string
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

	httpServer := &http.Server{
		Handler: rtr,

		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Println("server listening on port ", addr)
	if err := httpServer.Serve(listener); err != nil {
		log.Fatal("error listening: ", err.Error())
	}

}
