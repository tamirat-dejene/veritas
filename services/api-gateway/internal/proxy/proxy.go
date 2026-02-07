package proxy

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// NewProxy creates a new ReverseProxy that forwards requests to the target URL.
func NewProxy(target string) (*httputil.ReverseProxy, error) {
	url, err := url.Parse(target)
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(url)

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		// Set X-Forwarded-Host headers etc if needed
		req.Header.Set("X-Forwarded-Host", req.Host)
		req.Host = url.Host // Important for some backends to receive correct Host header
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Proxy error: %v", err)
		http.Error(w, "Service Unavailable", http.StatusBadGateway)
	}

	return proxy, nil
}
