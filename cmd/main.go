package main

import "github.com/vijayvenkatj/relayd/internal/server"

func main() {

	srv := &server.Server{
		ConfigPath: "./tmp/config.yml",
	}

	srv.ListenAndServe()
}
