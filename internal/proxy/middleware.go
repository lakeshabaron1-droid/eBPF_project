package proxy


import (

	"crypto/rand"
	"encoding/json"


	"fmt"

	"net/http"
	"os"

	"time"
)

type Middleware func(http.Handler) http.Handler

func Chain(middlewares ...Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {


			next = middlewares[i](next)
		}
		return next
	}
}



func generateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40

	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])






}

func RequestID() Middleware {
	return func(next http.Handler) http.Handler {


		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get("X-Request-ID")
			if id == "" {

				id = generateUUID()
				r.Header.Set("X-Request-ID", id)
			}
			w.Header().Set("X-Request-ID", id)
			next.ServeHTTP(w, r)

		})
	}

}

func CORS() Middleware {

	return func(next http.Handler) http.Handler {


		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", origin)

				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

				w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-API-Key")
			}

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return

			}
			next.ServeHTTP(w, r)

		})
	}


}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int


}
