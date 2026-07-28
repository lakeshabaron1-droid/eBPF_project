package main



import (
	"encoding/json"

	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})


	http.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("X-User-ID")
		userScopes := r.Header.Get("X-User-Scopes")
		authMethod := r.Header.Get("X-Auth-Method")


		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"service":     "service_a",
			"users":       []string{"alice", "bob"},

			"user_id":     userID,
			"user_scopes": userScopes,
			"auth_method": authMethod,

		})
	})




	http.HandleFunc("/api/admin", func(w http.ResponseWriter, r *http.Request) {

		userID := r.Header.Get("X-User-ID")
