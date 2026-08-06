package main

import (

	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type orderResponse struct {
	Service    string   `json:"service"`
	Orders     []string `json:"orders"`
	UserID     string   `json:"user_id"`
	UserScopes string   `json:"user_scopes"`

	AuthMethod string   `json:"auth_method"`
	RequestID  string   `json:"request_id"`
	Timestamp  string   `json:"timestamp"`
}

func main() {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		w.Write([]byte("OK"))
	})


	http.HandleFunc("/api/orders", func(w http.ResponseWriter, r *http.Request) {
		logHeaders(r)

		resp := orderResponse{

			Service:    "service_b",
			Orders:     []string{"order_101", "order_102", "order_103"},
			UserID:     r.Header.Get("X-User-ID"),
			UserScopes: r.Header.Get("X-User-Scopes"),
			AuthMethod: r.Header.Get("X-Auth-Method"),
			RequestID:  r.Header.Get("X-Request-ID"),
			Timestamp:  time.Now().Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})


	fmt.Println("Mock Service B listening on :9002")
	log.Fatal(http.ListenAndServe(":9002", nil))
}

func logHeaders(r *http.Request) {
	fmt.Printf("[service_b] %s %s | X-User-ID=%s X-User-Scopes=%s X-Auth-Method=%s X-Request-ID=%s\n",
		r.Method,
		r.URL.Path,
		r.Header.Get("X-User-ID"),
		r.Header.Get("X-User-Scopes"),
		r.Header.Get("X-Auth-Method"),
