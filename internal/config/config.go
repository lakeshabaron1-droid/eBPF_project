package config

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {

	Listen      ListenConfig      `yaml:"listen"`
	Ebpf        EbpfConfig        `yaml:"ebpf"`
	Auth        AuthConfig        `yaml:"auth"`

	Routes      []RouteConfig     `yaml:"routes"`
	HealthCheck HealthCheckConfig `yaml:"health_check"`


	Dashboard   DashboardConfig   `yaml:"dashboard"`
	Logging     LoggingConfig     `yaml:"logging"`
}

type ListenConfig struct {

	Address string `yaml:"address"`
	TlsCert string `yaml:"tls_cert"`



	TlsKey  string `yaml:"tls_key"`


}


type EbpfConfig struct {
	Interface      string          `yaml:"interface"`
	XdpMode        string          `yaml:"xdp_mode"`
	RateLimit      RateLimitConfig `yaml:"rate_limit"`
	RingBufferSize int             `yaml:"ring_buffer_size"`


}

type RateLimitConfig struct {
	Threshold uint32 `yaml:"threshold"`
	WindowMs  uint32 `yaml:"window_ms"`
}


type AuthConfig struct {

	Mode    string         `yaml:"mode"`

	ApiKeys []ApiKeyConfig `yaml:"api_keys"`

	Jwt     JwtConfig      `yaml:"jwt"`

}

type ApiKeyConfig struct {

	Key    string   `yaml:"key"`
	Name   string   `yaml:"name"`
	Scopes []string `yaml:"scopes"`
}

type JwtConfig struct {
	Algorithm    string `yaml:"algorithm"`


	Secret       string `yaml:"secret"`
	JwksUrl      string `yaml:"jwks_url"`
	Issuer       string `yaml:"issuer"`
	JwksCacheTtl int    `yaml:"jwks_cache_ttl"`

}


type RouteConfig struct {
	Path           string   `yaml:"path"`
	Upstream       string   `yaml:"upstream"`
	AuthRequired   bool     `yaml:"auth_required"`
	RequiredScopes []string `yaml:"required_scopes"`
	TimeoutMs      int      `yaml:"timeout_ms"`
}


type HealthCheckConfig struct {
	IntervalS          int    `yaml:"interval_s"`
	TimeoutMs          int    `yaml:"timeout_ms"`

	Path               string `yaml:"path"`
	UnhealthyThreshold int    `yaml:"unhealthy_threshold"`

	HealthyThreshold   int    `yaml:"healthy_threshold"`
}


type DashboardConfig struct {
	Enabled    bool   `yaml:"enabled"`
	ApiAddress string `yaml:"api_address"`
	CorsOrigin string `yaml:"cors_origin"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`


	Format string `yaml:"format"`
