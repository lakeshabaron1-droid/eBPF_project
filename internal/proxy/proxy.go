package proxy

import (
	"crypto/tls"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"ebpf-gateway/internal/config"
)

type Gateway struct {
	config *config.Config
	router http.Handler
	server *http.Server
}

func NewGateway(cfg *config.Config, router http.Handler) *Gateway {
	return &Gateway{
		config: cfg,
		router: router,
	}
}

func (g *Gateway) Start() error {
	g.server = &http.Server{
		Addr:         g.config.Listen.Address,
		Handler:      g.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if g.config.Listen.TlsCert != "" && g.config.Listen.TlsKey != "" {
		g.server.TLSConfig = &tls.Config{
			MinVersion:               tls.VersionTLS12,
			PreferServerCipherSuites: true,
		}
		return g.server.ListenAndServeTLS(g.config.Listen.TlsCert, g.config.Listen.TlsKey)
	}

	return g.server.ListenAndServe()
}

func (g *Gateway) Stop() error {
	if g.server != nil {
		return g.server.Close()
	}
	return nil
}

func NewReverseProxy(targetStr string) (*httputil.ReverseProxy, error) {
	target, err := url.Parse(targetStr)
	if err != nil {
		return nil, err
	}

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
		},
		ModifyResponse: func(res *http.Response) error {
			res.Header.Set("X-Proxy", "ebpf-gateway")
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, req *http.Request, err error) {
			log.Printf("Proxy error to %s: %v", targetStr, err)
			w.WriteHeader(http.StatusBadGateway)
		},
	}
	return proxy, nil
}
