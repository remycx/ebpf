package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// APIKey represents an agent API key
type APIKey struct {
	ID        string            `json:"id"`
	Key       string            `json:"-"` // Never expose in JSON
	KeyHash   string            `json:"key_hash"`
	Name      string            `json:"name"`
	CreatedAt time.Time         `json:"created_at"`
	ExpiresAt *time.Time        `json:"expires_at,omitempty"`
	LastUsed  *time.Time        `json:"last_used,omitempty"`
	Active    bool              `json:"active"`
	Metadata  map[string]string `json:"metadata"`
}

// AuthManager manages API keys for agent authentication
type AuthManager struct {
	keys map[string]*APIKey // key hash -> APIKey
	mu   sync.RWMutex
}

// NewAuthManager creates a new authentication manager
func NewAuthManager() *AuthManager {
	return &AuthManager{
		keys: make(map[string]*APIKey),
	}
}

// GenerateAPIKey creates a new API key
func (am *AuthManager) GenerateAPIKey(name string, metadata map[string]string, expiresIn *time.Duration) (*APIKey, error) {
	// Generate random key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random key: %w", err)
	}

	key := base64.URLEncoding.EncodeToString(keyBytes)
	keyHash := hashKey(key)

	apiKey := &APIKey{
		ID:        generateID(),
		Key:       key,
		KeyHash:   keyHash,
		Name:      name,
		CreatedAt: time.Now(),
		Active:    true,
		Metadata:  metadata,
	}

	if expiresIn != nil {
		expires := time.Now().Add(*expiresIn)
		apiKey.ExpiresAt = &expires
	}

	am.mu.Lock()
	am.keys[keyHash] = apiKey
	am.mu.Unlock()

	return apiKey, nil
}

// ValidateKey checks if a key is valid and updates last used time
func (am *AuthManager) ValidateKey(key string) (*APIKey, error) {
	keyHash := hashKey(key)

	am.mu.Lock()
	defer am.mu.Unlock()

	apiKey, exists := am.keys[keyHash]
	if !exists {
		return nil, fmt.Errorf("invalid API key")
	}

	if !apiKey.Active {
		return nil, fmt.Errorf("API key is disabled")
	}

	if apiKey.ExpiresAt != nil && time.Now().After(*apiKey.ExpiresAt) {
		return nil, fmt.Errorf("API key has expired")
	}

	// Update last used time
	now := time.Now()
	apiKey.LastUsed = &now

	return apiKey, nil
}

// RevokeKey disables an API key
func (am *AuthManager) RevokeKey(keyHash string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	apiKey, exists := am.keys[keyHash]
	if !exists {
		return fmt.Errorf("API key not found")
	}

	apiKey.Active = false
	return nil
}

// ListKeys returns all API keys (without the actual key value)
func (am *AuthManager) ListKeys() []*APIKey {
	am.mu.RLock()
	defer am.mu.RUnlock()

	keys := make([]*APIKey, 0, len(am.keys))
	for _, key := range am.keys {
		keyCopy := *key
		keyCopy.Key = "" // Never expose the actual key
		keys = append(keys, &keyCopy)
	}
	return keys
}

// LoadKey adds a pre-existing key to the manager (for bootstrapping)
func (am *AuthManager) LoadKey(apiKey *APIKey) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.keys[apiKey.KeyHash] = apiKey
}

// Helper functions

func hashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

func generateID() string {
	idBytes := make([]byte, 16)
	rand.Read(idBytes)
	return hex.EncodeToString(idBytes)
}
