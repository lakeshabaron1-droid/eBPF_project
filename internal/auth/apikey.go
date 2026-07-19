package auth






import (

	"context"
	"net/http"
	"strings"


	"ebpf-gateway/internal/config"

	"ebpf-gateway/internal/proxy"
)

type ApiKeyInfo struct {
	Name   string

	Scopes []string
}

type APIKeyValidator struct {
	keys map[string]ApiKeyInfo
}

func NewAPIKeyValidator(cfg []config.ApiKeyConfig) *APIKeyValidator {
	v := &APIKeyValidator{


		keys: make(map[string]ApiKeyInfo),
	}
	for _, k := range cfg {

		v.keys[k.Key] = ApiKeyInfo{
			Name:   k.Name,


			Scopes: k.Scopes,
		}
	}
	return v

}

func (v *APIKeyValidator) Validate(req *http.Request) (*ApiKeyInfo, bool) {

	key := req.Header.Get("X-API-Key")

	if key == "" {
		authHeader := req.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "ApiKey ") {
			key = strings.TrimPrefix(authHeader, "ApiKey ")


		} else if strings.HasPrefix(authHeader, "Bearer ") {
			key = strings.TrimPrefix(authHeader, "Bearer ")
		}


	}



	if key == "" {
		return nil, false

	}


	info, exists := v.keys[key]
	if !exists {

