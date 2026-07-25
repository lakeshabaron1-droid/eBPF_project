

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

	enforcer := auth.NewZeroTrustEnforcer(cfg.Auth)



	handler := proxy.Chain(

