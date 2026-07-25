package ratelimit

import (
	"encoding/json"
	"net/http"
	"strings"

	"ebpf-gateway/internal/config"
	bpf "ebpf-gateway/internal/ebpf"
)

type Controller struct {
	maps   *bpf.MapManager
	routes []config.RouteConfig
}

func NewController(maps *bpf.MapManager, routes []config.RouteConfig) *Controller {
	return &Controller{
		maps:   maps,
		routes: routes,
	}
}

func (c *Controller) HandleBlockIP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		IP string `json:"ip"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := c.maps.BlockIP(req.IP); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "blocked", "ip": req.IP})
}

func (c *Controller) HandleUnblockIP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "IP required in path", http.StatusBadRequest)
		return
	}
	ip := parts[len(parts)-1]

	if err := c.maps.UnblockIP(ip); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "unblocked", "ip": ip})
}

func (c *Controller) HandleUpdateRateLimit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Threshold uint32 `json:"threshold"`
		WindowMs  uint32 `json:"window_ms"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := c.maps.UpdateConfig(req.Threshold, req.WindowMs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "updated",
		"threshold": req.Threshold,
		"window_ms": req.WindowMs,
	})
}

func (c *Controller) HandleGetRoutes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(c.routes)
}

func (c *Controller) HandleGetBlockedIPs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ips, err := c.maps.ListBlockedIPs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"blocked_ips": ips})
}
