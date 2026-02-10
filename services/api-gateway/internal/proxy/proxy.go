package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"go.uber.org/zap"
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
		// Set X-Forwarded headers
		// X-Forwarded-For
		clientIP := req.RemoteAddr
		if ip := req.Header.Get("X-Forwarded-For"); ip != "" {
			req.Header.Set("X-Forwarded-For", ip+", "+clientIP)
		} else {
			req.Header.Set("X-Forwarded-For", clientIP)
		}
		// X-Forwarded-Host
		req.Header.Set("X-Forwarded-Host", req.Host)
		// X-Forwarded-Proto
		proto := "http"
		if req.TLS != nil {
			proto = "https"
		}
		req.Header.Set("X-Forwarded-Proto", proto)
		req.Host = url.Host
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		zap.L().Error("Proxy error", zap.Error(err))
		http.Error(w, "Service Unavailable", http.StatusBadGateway)
	}

	return proxy, nil
}
