package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"


)

type userResponse struct {

	Service    string   `json:"service"`
	Users      []string `json:"users"`


	UserID     string   `json:"user_id"`

	UserScopes string   `json:"user_scopes"`
	AuthMethod string   `json:"auth_method"`

	RequestID  string   `json:"request_id"`
	Timestamp  string   `json:"timestamp"`
}

type adminResponse struct {
	Service   string `json:"service"`
	Admin     bool   `json:"admin"`
	UserID    string `json:"user_id"`

	RequestID string `json:"request_id"`
}


func main() {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	http.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
		logHeaders(r)

		resp := userResponse{
			Service:    "service_a",
			Users:      []string{"alice", "bob", "charlie"},
			UserID:     r.Header.Get("X-User-ID"),
			UserScopes: r.Header.Get("X-User-Scopes"),
			AuthMethod: r.Header.Get("X-Auth-Method"),
			RequestID:  r.Header.Get("X-Request-ID"),
			Timestamp:  time.Now().Format(time.RFC3339),

		}

		w.Header().Set("Content-Type", "application/json")

		json.NewEncoder(w).Encode(resp)
	})


	http.HandleFunc("/api/admin", func(w http.ResponseWriter, r *http.Request) {
		logHeaders(r)

		resp := adminResponse{


			Service:   "service_a",
			Admin:     true,
			UserID:    r.Header.Get("X-User-ID"),
			RequestID: r.Header.Get("X-Request-ID"),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})


	fmt.Println("Mock Service A listening on :9001")
	log.Fatal(http.ListenAndServe(":9001", nil))
}

func logHeaders(r *http.Request) {
	fmt.Printf("[service_a] %s %s | X-User-ID=%s X-User-Scopes=%s X-Auth-Method=%s X-Request-ID=%s\n",
		r.Method,

		r.URL.Path,

		r.Header.Get("X-User-ID"),
		r.Header.Get("X-User-Scopes"),
		r.Header.Get("X-Auth-Method"),
