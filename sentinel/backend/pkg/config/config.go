package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Storage  StorageConfig  `yaml:"storage"`
	Security SecurityConfig `yaml:"security"`
}

type ServerConfig struct {
	Listen   string     `yaml:"listen"`
	TLS      TLSConfig  `yaml:"tls"`
}

type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

type StorageConfig struct {
	Type           string `yaml:"type"`
	RetentionHours int    `yaml:"retention_hours"`
}

type SecurityConfig struct {
	RequireAuth     bool    `yaml:"require_auth"`
	APIKeysFile     string  `yaml:"api_keys_file"`
	RateLimit       float64 `yaml:"rate_limit"`       // requests per second
	RateLimitBurst  int     `yaml:"rate_limit_burst"` // burst size
	TrustedProxies  []string `yaml:"trusted_proxies"`
}

func Load(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		// Return default config if file doesn't exist
		if os.IsNotExist(err) {
			return defaultConfig(), nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply defaults
	if cfg.Server.Listen == "" {
		cfg.Server.Listen = ":8080"
	}

	if cfg.Storage.Type == "" {
		cfg.Storage.Type = "memory"
	}

	if cfg.Storage.RetentionHours == 0 {
		cfg.Storage.RetentionHours = 24
	}

	if cfg.Security.RateLimit == 0 {
		cfg.Security.RateLimit = 10.0 // 10 requests per second
	}

	if cfg.Security.RateLimitBurst == 0 {
		cfg.Security.RateLimitBurst = 20
	}

	if cfg.Security.APIKeysFile == "" {
		cfg.Security.APIKeysFile = "api-keys.yaml"
	}

	return &cfg, nil
}

func defaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Listen: ":8080",
		},
		Storage: StorageConfig{
			Type:           "memory",
			RetentionHours: 24,
		},
		Security: SecurityConfig{
			RequireAuth:    false,
			APIKeysFile:    "api-keys.yaml",
			RateLimit:      10.0,
			RateLimitBurst: 20,
		},
	}
}
