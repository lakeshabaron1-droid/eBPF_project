package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ebpf-gateway/internal/auth"
	"ebpf-gateway/internal/config"
	"ebpf-gateway/internal/ebpf"
	"ebpf-gateway/internal/metrics"
	"ebpf-gateway/internal/proxy"

	"ebpf-gateway/internal/ratelimit"
)

func main() {
	configPath := flag.String("config", "configs/gateway.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	bpfManager := ebpf.NewManager()
	if err := bpfManager.LoadPrograms(cfg.Ebpf.Interface); err != nil {
		log.Printf("Warning: failed to load eBPF programs: %v", err)
	} else {
		if err := bpfManager.AttachXDP(cfg.Ebpf.XdpMode); err != nil {
			log.Printf("Warning: failed to attach XDP: %v", err)
		}


		if err := bpfManager.AttachTC(); err != nil {

			log.Printf("Warning: failed to attach TC: %v", err)
		}

		defer bpfManager.Close()
	}


	mapManager := ebpf.NewMapManager(bpfManager)
	if err := mapManager.UpdateConfig(cfg.Ebpf.RateLimit.Threshold, cfg.Ebpf.RateLimit.WindowMs); err != nil {
		log.Printf("Warning: failed to update rate limit map: %v", err)
	}

	collector := metrics.NewCollector(mapManager)
	if err := collector.Start(); err != nil {
		log.Printf("Warning: failed to start metrics collector: %v", err)
	}
	defer collector.Stop()

	aggregator := metrics.NewAggregator()
	collector.Subscribe(aggregator.InputChannel())
	aggregator.Start()
	defer aggregator.Stop()


	broadcaster := metrics.NewSSEBroker()
	broadcaster.Start()
	defer broadcaster.Stop()

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			snap := aggregator.Snapshot()
			select {
			case broadcaster.InputChannel() <- snap:
			default:
			}
		}
	}()

	router, err := proxy.NewRouter(cfg, func() (*config.Config, error) {


		return config.Load(*configPath)
	})
	if err != nil {
		log.Fatalf("Failed to create router: %v", err)
	}

	enforcer := auth.NewZeroTrustEnforcer(cfg.Auth, cfg.Routes)

	handler := proxy.Chain(
		proxy.RequestID(),
		proxy.CORS(),
		proxy.Logging(),
		enforcer.Middleware(),
	)(router)

	gateway := proxy.NewGateway(cfg, handler)

	ctrl := ratelimit.NewController(mapManager, cfg.Routes)
	apiMux := http.NewServeMux()
	apiMux.Handle("/events", broadcaster)
	apiMux.HandleFunc("/api/block", ctrl.HandleBlockIP)
	apiMux.HandleFunc("/api/block/", ctrl.HandleUnblockIP)
	apiMux.HandleFunc("/api/config/ratelimit", ctrl.HandleUpdateRateLimit)
	apiMux.HandleFunc("/api/routes", ctrl.HandleGetRoutes)
	apiMux.HandleFunc("/api/blocked", ctrl.HandleGetBlockedIPs)

	apiServer := &http.Server{
		Addr:    cfg.Dashboard.ApiAddress,

		Handler: apiMux,
	}

	go func() {
		log.Printf("Starting API/SSE server on %s", cfg.Dashboard.ApiAddress)
		if err := apiServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("API server error: %v", err)
		}
	}()

	go func() {
		log.Printf("Starting Gateway proxy on %s", cfg.Listen.Address)
		if err := gateway.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Gateway server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down gateway gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := apiServer.Shutdown(ctx); err != nil {
		log.Printf("API server shutdown error: %v", err)
	}


	if err := gateway.Stop(); err != nil {
		log.Printf("Gateway shutdown error: %v", err)
	}

	log.Println("Gateway stopped.")
}


