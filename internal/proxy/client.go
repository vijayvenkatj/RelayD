package proxy

import (
	"net/http"
	"time"
)

// NewHTTPClient makes a http client that uses the system's shared transport
func NewHTTPClient(transport *http.Transport) *http.Client {
	return &http.Client{
		Transport: transport,
		Timeout:   2 * time.Second,
	}
}
