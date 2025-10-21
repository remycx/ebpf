package auth

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// KeyStore represents persistent API key storage
type KeyStore struct {
	Keys []*APIKey `yaml:"keys"`
}

// LoadKeysFromFile loads API keys from a YAML file
func LoadKeysFromFile(filename string) ([]*APIKey, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return []*APIKey{}, nil // Empty list if file doesn't exist
		}
		return nil, fmt.Errorf("failed to read keys file: %w", err)
	}

	var store KeyStore
	if err := yaml.Unmarshal(data, &store); err != nil {
		return nil, fmt.Errorf("failed to parse keys file: %w", err)
	}

	return store.Keys, nil
}

// SaveKeysToFile saves API keys to a YAML file
func SaveKeysToFile(filename string, keys []*APIKey) error {
	store := KeyStore{Keys: keys}

	data, err := yaml.Marshal(&store)
	if err != nil {
		return fmt.Errorf("failed to marshal keys: %w", err)
	}

	if err := os.WriteFile(filename, data, 0600); err != nil {
		return fmt.Errorf("failed to write keys file: %w", err)
	}

	return nil
}

// CreateBootstrapKey creates an initial API key for bootstrapping
func CreateBootstrapKey() *APIKey {
	now := time.Now()
	return &APIKey{
		ID:        "bootstrap",
		KeyHash:   hashKey("sentinel-bootstrap-key-change-me"),
		Name:      "Bootstrap Key",
		CreatedAt: now,
		Active:    true,
		Metadata: map[string]string{
			"purpose": "Initial bootstrap key - change immediately",
		},
	}
}
