

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


