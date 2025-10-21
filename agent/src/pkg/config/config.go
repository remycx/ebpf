package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Agent  AgentConfig  `yaml:"agent"`
	Server ServerConfig `yaml:"server"`
}

type AgentConfig struct {
	Hostname string `yaml:"hostname"`
	Tags     map[string]string `yaml:"tags"`
}

type ServerConfig struct {
	URL             string `yaml:"url"`
	ReconnectDelay  int    `yaml:"reconnect_delay"`
	MaxRetries      int    `yaml:"max_retries"`
	InsecureSkipTLS bool   `yaml:"insecure_skip_tls"`
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
	if cfg.Agent.Hostname == "" {
		hostname, _ := os.Hostname()
		cfg.Agent.Hostname = hostname
	}

	if cfg.Server.ReconnectDelay == 0 {
		cfg.Server.ReconnectDelay = 5
	}

	if cfg.Server.MaxRetries == 0 {
		cfg.Server.MaxRetries = -1 // Infinite retries
	}

	return &cfg, nil
}

func defaultConfig() *Config {
	hostname, _ := os.Hostname()
	return &Config{
		Agent: AgentConfig{
			Hostname: hostname,
			Tags:     make(map[string]string),
		},
		Server: ServerConfig{
			URL:            "ws://localhost:8080/api/v1/events",
			ReconnectDelay: 5,
			MaxRetries:     -1,
		},
	}
}
