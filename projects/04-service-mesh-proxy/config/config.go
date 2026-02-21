// Package config loads YAML proxy configuration.
package config

import (
	"fmt"
	"os"
	"time"
)

// Config is the top-level proxy configuration.
type Config struct {
	ListenAddr     string        `yaml:"listen_addr"`
	AdminAddr      string        `yaml:"admin_addr"`
	Upstreams      []UpstreamCfg `yaml:"upstreams"`
	TLS            TLSConfig     `yaml:"tls"`
	CircuitBreaker CBConfig      `yaml:"circuit_breaker"`
	Timeout        time.Duration `yaml:"timeout"`
	RetryPolicy    RetryConfig   `yaml:"retry"`
}

// UpstreamCfg describes one upstream endpoint.
type UpstreamCfg struct {
	Name    string `yaml:"name"`
	Addr    string `yaml:"addr"`
	Weight  int    `yaml:"weight"`   // for weighted round-robin
	MaxConn int    `yaml:"max_conn"` // connection pool limit
}

// TLSConfig holds mTLS material paths.
type TLSConfig struct {
	CACert     string `yaml:"ca_cert"`
	ServerCert string `yaml:"server_cert"`
	ServerKey  string `yaml:"server_key"`
}

// CBConfig is the circuit-breaker configuration per upstream.
type CBConfig struct {
	MaxFailures     int           `yaml:"max_failures"`
	ResetTimeout    time.Duration `yaml:"reset_timeout"`
	HalfOpenMaxReqs int           `yaml:"half_open_max_reqs"`
}

// RetryConfig controls retry behaviour.
type RetryConfig struct {
	MaxAttempts int           `yaml:"max_attempts"`
	Backoff     time.Duration `yaml:"backoff"`
}

// Defaults are applied when fields are zero.
var defaults = Config{
	ListenAddr: ":8080",
	AdminAddr:  ":9090",
	Timeout:    30 * time.Second,
	CircuitBreaker: CBConfig{
		MaxFailures:     5,
		ResetTimeout:    10 * time.Second,
		HalfOpenMaxReqs: 2,
	},
	RetryPolicy: RetryConfig{
		MaxAttempts: 3,
		Backoff:     200 * time.Millisecond,
	},
}

// Load reads a YAML config file and applies defaults for missing fields.
// Falls back to an in-memory default config if path doesn't exist.
func Load(path string) (*Config, error) {
	cfg := defaults

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return defaults with a loopback upstream for development.
			cfg.Upstreams = []UpstreamCfg{{Name: "default", Addr: "127.0.0.1:8081", Weight: 1}}
			return &cfg, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	// TODO: parse YAML into cfg (use gopkg.in/yaml.v3 or encode/json after converting)
	// For stdlib-only: write a minimal YAML parser or use JSON format instead.
	_ = data
	return &cfg, fmt.Errorf("YAML parsing not yet implemented — drop a config.json and swap Load()")
}
