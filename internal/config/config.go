package config

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

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
func (config *Config) BackendGroups(reverseProxy *httputil.ReverseProxy, httpClient *http.Client) map[string]*upstream.BackendGroup {

	backendGroups := make(map[string]*upstream.BackendGroup)

	for _, route := range config.Routes {

		backends := config.Backends(route.Upstreams, reverseProxy, httpClient)
		backendGroup := upstream.NewBackendGroup(backends)

		backendGroups[route.Route] = backendGroup
	}

	return backendGroups
}

// Get the Backends from Config and Inject HTTP client.
func (config *Config) Backends(upstreams []string, reverseProxy *httputil.ReverseProxy, httpClient *http.Client) []*upstream.Backend {
	backends := []*upstream.Backend{}
	for _, host := range upstreams {

		hostUrl, err := url.Parse(host)
		if err != nil {
			log.Println("host invalid: ", hostUrl)
			continue
		}

		backend := upstream.NewBackend(hostUrl, reverseProxy, httpClient)
		backends = append(backends, backend)
	}

	return backends
}
