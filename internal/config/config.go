package config

import (
	"log"
	"net/http"
	"net/url"

	"github.com/vijayvenkatj/relayd/internal/proxy"
	"github.com/vijayvenkatj/relayd/internal/upstream"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server ServerConfig  `yaml:"server"`
	Routes []RouteConfig `yaml:"routes"`
}

type ServerConfig struct {
	Port string `yaml:"port"`
}

type RouteConfig struct {
	Route     string   `yaml:"path"`
	Upstreams []string `yaml:"upstreams"`
}

func Parse(data []byte) (*Config, error) {

	var config Config
	err := yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// Get the BackendGroups from config
func (config *Config) BackendGroups(transport *http.Transport, httpClient *http.Client) []*upstream.BackendGroup {

	backendGroups := []*upstream.BackendGroup{}

	for _, route := range config.Routes {

		backends := config.Backends(route.Upstreams, transport, httpClient)
		backendGroup := upstream.NewBackendGroup(route.Route, backends)

		backendGroups = append(backendGroups, backendGroup)
	}

	return backendGroups
}

// Get the Backends from Config and Inject HTTP client.
func (config *Config) Backends(upstreams []string, transport *http.Transport, httpClient *http.Client) []*upstream.Backend {
	backends := []*upstream.Backend{}
	for _, host := range upstreams {

		hostUrl, err := url.Parse(host)
		if err != nil {
			log.Println("host invalid: ", hostUrl)
			continue
		}

		reverseProxy := proxy.NewReverseProxy(hostUrl, transport)

		backend := upstream.NewBackend(hostUrl, reverseProxy, httpClient)
		backends = append(backends, backend)
	}

	return backends
}
