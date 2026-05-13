package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

func NewReverseProxy(hostURL *url.URL, transport *http.Transport) *httputil.ReverseProxy {

	// Making a reverse proxy for a HOST
	proxy := httputil.NewSingleHostReverseProxy(hostURL)

	// Injecting shared Transport to improve connection pooling
	proxy.Transport = transport

	return proxy
}
