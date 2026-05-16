package main

import (
	"flag"

	"github.com/vijayvenkatj/relayd/internal/server"
)

var configPath string

func init() {
	flag.StringVar(&configPath, "config", "./tmp/config.yml", "path to the config file")
	flag.Parse()
}

func main() {

	srv := &server.Server{
		ConfigPath: configPath,
	}

	srv.ListenAndServe()
}
