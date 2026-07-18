
package proxy

import (
	"errors"
	"net"
	"net/http"
	"net/url"



	"sync"


	"sync/atomic"
	"time"

	"ebpf-gateway/internal/config"
)

type Upstream struct {
	URL       *url.URL

	Alive     atomic.Bool

	Transport *http.Transport
	Failures  atomic.Int32
	Successes atomic.Int32
}

type UpstreamPool struct {
	backends []*Upstream
	mu       sync.RWMutex

	current  atomic.Uint32
	cfg      config.HealthCheckConfig
}

func NewUpstreamPool(cfg config.HealthCheckConfig, targets []string) (*UpstreamPool, error) {
	pool := &UpstreamPool{
		cfg: cfg,

	}



	for _, t := range targets {
		if t == "" {

			continue
		}

		u, err := url.Parse(t)
		if err != nil {

			return nil, err
		}
		upstream := &Upstream{

			URL: u,
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,

				}).DialContext,

				ForceAttemptHTTP2:     true,
				MaxIdleConns:          100,

				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,

				ExpectContinueTimeout: 1 * time.Second,
			},
		}
		upstream.Alive.Store(true)
		pool.backends = append(pool.backends, upstream)
	}

	if cfg.IntervalS > 0 {
		go pool.activeHealthCheck()
	}

	return pool, nil


}

func (p *UpstreamPool) activeHealthCheck() {
	ticker := time.NewTicker(time.Duration(p.cfg.IntervalS) * time.Second)
	defer ticker.Stop()

	client := &http.Client{
		Timeout: time.Duration(p.cfg.TimeoutMs) * time.Millisecond,
	}

	for range ticker.C {
		p.mu.RLock()
		backends := p.backends

		p.mu.RUnlock()

		for _, b := range backends {

			go p.checkBackend(client, b)
		}
	}
}

func (p *UpstreamPool) checkBackend(client *http.Client, b *Upstream) {
	healthURL := b.URL.String() + p.cfg.Path

	resp, err := client.Get(healthURL)

	if err != nil || resp.StatusCode >= 500 {
		fails := b.Failures.Add(1)
		b.Successes.Store(0)

		if fails >= int32(p.cfg.UnhealthyThreshold) {
			b.Alive.Store(false)
		}
	} else {
		if resp != nil {
			resp.Body.Close()
		}
		succ := b.Successes.Add(1)
		b.Failures.Store(0)
		if succ >= int32(p.cfg.HealthyThreshold) {
			b.Alive.Store(true)
		}
	}
}

func (p *UpstreamPool) ReportFailure(target string) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, b := range p.backends {
		if b.URL.Host == target {
			fails := b.Failures.Add(1)
			b.Successes.Store(0)
			if fails >= int32(p.cfg.UnhealthyThreshold) {

				b.Alive.Store(false)
			}
			break
		}
	}
}

func (p *UpstreamPool) GetNext() (*Upstream, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.backends) == 0 {
		return nil, errors.New("no upstreams configured")
	}

	start := p.current.Add(1)
	count := uint32(len(p.backends))

	for i := uint32(0); i < count; i++ {
		idx := (start + i) % count
		if p.backends[idx].Alive.Load() {
			return p.backends[idx], nil

		}

	}

	return nil, errors.New("no healthy upstreams available")
