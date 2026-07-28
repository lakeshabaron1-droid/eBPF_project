package proxy

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"

	"ebpf-gateway/internal/config"
)

type Router struct {
	mu          sync.RWMutex
	routes      []*Route
	healthCfg   config.HealthCheckConfig
	configWatch func() (*config.Config, error)
}

type Route struct {
	cfg   config.RouteConfig
	pool  *UpstreamPool
	proxy http.Handler
}

func NewRouter(cfg *config.Config, watch func() (*config.Config, error)) (*Router, error) {
	r := &Router{
		healthCfg:   cfg.HealthCheck,
		configWatch: watch,
	}

	if err := r.build(cfg.Routes); err != nil {
		return nil, err
	}

	if watch != nil {
		go r.watchSIGHUP()
	}

	return r, nil
}

func (r *Router) build(routes []config.RouteConfig) error {
	var newRoutes []*Route

	for _, rc := range routes {
		pool, err := NewUpstreamPool(r.healthCfg, strings.Split(rc.Upstream, ","))
		if err != nil {
			return err
		}

		targetURL, _ := url.Parse(strings.Split(rc.Upstream, ",")[0])

		proxy, err := NewReverseProxy(targetURL.String())
		if err != nil {
			return err
		}

		proxy.Transport = pool.RoundTripper()

		newRoutes = append(newRoutes, &Route{
			cfg:   rc,
			pool:  pool,
			proxy: proxy,
		})
	}

	sort.Slice(newRoutes, func(i, j int) bool {
		return len(newRoutes[i].cfg.Path) > len(newRoutes[j].cfg.Path)
	})

	r.mu.Lock()
	r.routes = newRoutes
	r.mu.Unlock()

	return nil
}

func (r *Router) watchSIGHUP() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP)
	for range sig {
		if cfg, err := r.configWatch(); err == nil {
			r.build(cfg.Routes)
		}
	}
}

type ctxKey string

const RouteContextKey ctxKey = "routeConfig"

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mu.RLock()
	routes := r.routes
	r.mu.RUnlock()

	for _, route := range routes {
		if strings.HasPrefix(req.URL.Path, route.cfg.Path) {
			ctx := context.WithValue(req.Context(), RouteContextKey, route.cfg)
			route.proxy.ServeHTTP(w, req.WithContext(ctx))
			return
		}
	}

	http.NotFound(w, req)
}
