package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"

	"strings"
	"time"


	"ebpf-gateway/internal/config"

	"ebpf-gateway/internal/proxy"
)

type ZeroTrustEnforcer struct {
	apiKeyValidator *APIKeyValidator
	jwtValidator    *JWTValidator
	mode            string

	routes          []config.RouteConfig

}

func NewZeroTrustEnforcer(cfg config.AuthConfig, routes []config.RouteConfig) *ZeroTrustEnforcer {
	sortedRoutes := make([]config.RouteConfig, len(routes))

	copy(sortedRoutes, routes)
	sort.Slice(sortedRoutes, func(i, j int) bool {
		return len(sortedRoutes[i].Path) > len(sortedRoutes[j].Path)
	})

	e := &ZeroTrustEnforcer{
		mode:   cfg.Mode,
		routes: sortedRoutes,
	}


	if cfg.Mode == "apikey" || cfg.Mode == "both" {
		e.apiKeyValidator = NewAPIKeyValidator(cfg.ApiKeys)
	}


	if cfg.Mode == "jwt" || cfg.Mode == "both" {
		e.jwtValidator = NewJWTValidator(cfg.Jwt)
	}

	return e
}

func (e *ZeroTrustEnforcer) matchRoute(path string) (config.RouteConfig, bool) {
	for _, r := range e.routes {
		if strings.HasPrefix(path, r.Path) {
			return r, true
		}
	}
	return config.RouteConfig{}, false
}


func (e *ZeroTrustEnforcer) Middleware() proxy.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			routeCfg, matched := e.matchRoute(r.URL.Path)
			if !matched || !routeCfg.AuthRequired {
				next.ServeHTTP(w, r)
				return
			}

			var authenticated bool
			var userID string
			var userScopes []string
			var authMethod string


			if e.apiKeyValidator != nil {

				info, valid := e.apiKeyValidator.Validate(r)
				if valid {
					authenticated = true
					userID = info.Name
					userScopes = info.Scopes
					authMethod = "apikey"

				}
			}

			if !authenticated && e.jwtValidator != nil {
				authHeader := r.Header.Get("Authorization")
				if strings.HasPrefix(authHeader, "Bearer ") {
					tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
					claims, err := e.jwtValidator.Validate(tokenStr)
					if err == nil {
						authenticated = true
						userID = claims.Sub
						userScopes = claims.Scopes
						authMethod = "jwt"
					}
				}
			}

			if !authenticated {
				e.auditLog(r, "", "denied", "no valid credentials")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if len(routeCfg.RequiredScopes) > 0 {

				if !hasRequiredScopes(userScopes, routeCfg.RequiredScopes) {

					e.auditLog(r, userID, "denied", "insufficient scopes")
					http.Error(w, "Forbidden", http.StatusForbidden)

					return
				}
			}

			r.Header.Set("X-User-ID", userID)
			r.Header.Set("X-User-Scopes", strings.Join(userScopes, ","))
			r.Header.Set("X-Auth-Method", authMethod)

			e.auditLog(r, userID, "allowed", authMethod)

			next.ServeHTTP(w, r)
		})
	}
}

func hasRequiredScopes(userScopes []string, required []string) bool {
	scopeSet := make(map[string]bool)
	for _, s := range userScopes {
		scopeSet[s] = true
	}
	for _, r := range required {

		if !scopeSet[r] {
			return false

		}
	}
	return true
}

func (e *ZeroTrustEnforcer) auditLog(r *http.Request, userID, decision, detail string) {
	entry := map[string]interface{}{
		"time":       time.Now().Format(time.RFC3339),
		"type":       "auth_audit",
		"request_id": r.Header.Get("X-Request-ID"),
		"method":     r.Method,
		"path":       r.URL.Path,
		"remote_ip":  r.RemoteAddr,
		"user_id":    userID,
		"decision":   decision,
		"detail":     detail,
	}
	jsonLog, _ := json.Marshal(entry)
	fmt.Fprintln(os.Stdout, string(jsonLog))
}

