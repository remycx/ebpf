package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server ServerConfig `yaml:"server"`
	Storage StorageConfig `yaml:"storage"`
}

type ServerConfig struct {
	Listen string `yaml:"listen"`
}

type StorageConfig struct {
	Type           string `yaml:"type"`
	RetentionHours int    `yaml:"retention_hours"`
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
	}
}
