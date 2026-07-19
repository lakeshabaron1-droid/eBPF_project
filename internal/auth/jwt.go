package auth

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"io"
	"math/big"

	"net/http"
	"strings"
	"sync"
	"time"

	"ebpf-gateway/internal/config"
	"ebpf-gateway/internal/proxy"
	"github.com/golang-jwt/jwt/v5"
)

type JWTValidator struct {
	algorithm    string
	secret       []byte
	issuer       string
	jwksURL      string
	jwksCacheTTL time.Duration
	jwksCache    map[string]*rsa.PublicKey
	jwksMu       sync.RWMutex
	jwksLastFetch time.Time
}

func NewJWTValidator(cfg config.JwtConfig) *JWTValidator {


	return &JWTValidator{
		algorithm:    cfg.Algorithm,


		secret:       []byte(cfg.Secret),
		issuer:       cfg.Issuer,
		jwksURL:      cfg.JwksUrl,
		jwksCacheTTL: time.Duration(cfg.JwksCacheTtl) * time.Second,
		jwksCache:    make(map[string]*rsa.PublicKey),

	}
}

type JWTClaims struct {
	Sub    string   `json:"sub"`

	Scopes []string `json:"scopes"`
	jwt.RegisteredClaims

}

func (v *JWTValidator) Validate(tokenStr string) (*JWTClaims, error) {
	var claims JWTClaims

	token, err := jwt.ParseWithClaims(tokenStr, &claims, func(token *jwt.Token) (interface{}, error) {
		switch v.algorithm {

		case "HS256":
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return v.secret, nil
		case "RS256":
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, errors.New("unexpected signing method")
			}
			kid, _ := token.Header["kid"].(string)
			key, err := v.getPublicKey(kid)
			if err != nil {
				return nil, err
			}
			return key, nil
		default:
			return nil, errors.New("unsupported algorithm")
		}

	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	if v.issuer != "" && claims.Issuer != v.issuer {
		return nil, errors.New("invalid issuer")
	}

	return &claims, nil
}

func (v *JWTValidator) getPublicKey(kid string) (*rsa.PublicKey, error) {
	v.jwksMu.RLock()
	if key, ok := v.jwksCache[kid]; ok && time.Since(v.jwksLastFetch) < v.jwksCacheTTL {
		v.jwksMu.RUnlock()
		return key, nil
	}
	v.jwksMu.RUnlock()

	return v.fetchJWKS(kid)
}

type jwksResponse struct {
	Keys []jwkKey `json:"keys"`
}

type jwkKey struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	N   string `json:"n"`
	E   string `json:"e"`
}


func (v *JWTValidator) fetchJWKS(kid string) (*rsa.PublicKey, error) {
	v.jwksMu.Lock()
	defer v.jwksMu.Unlock()

	if v.jwksURL == "" {
		return nil, errors.New("no JWKS URL configured")
	}



	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(v.jwksURL)
	if err != nil {

		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var jwks jwksResponse
	if err := json.Unmarshal(body, &jwks); err != nil {
		return nil, err
	}



	v.jwksCache = make(map[string]*rsa.PublicKey)
	v.jwksLastFetch = time.Now()


	for _, k := range jwks.Keys {
		if k.Kty != "RSA" {
			continue
		}

		nBytes, err := jwt.NewParser().DecodeSegment(k.N)
		if err != nil {
			continue
		}
		eBytes, err := jwt.NewParser().DecodeSegment(k.E)
		if err != nil {
			continue
		}

		pubKey := &rsa.PublicKey{
			N: new(big.Int).SetBytes(nBytes),
			E: int(new(big.Int).SetBytes(eBytes).Int64()),
		}
		v.jwksCache[k.Kid] = pubKey
	}

	key, ok := v.jwksCache[kid]
	if !ok {
		return nil, errors.New("key not found in JWKS")
	}
	return key, nil
}

func (v *JWTValidator) Middleware() proxy.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
