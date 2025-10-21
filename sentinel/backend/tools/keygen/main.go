package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/sentinel/backend/pkg/auth"
)

var (
	name       = flag.String("name", "", "Name/description for the API key (required)")
	expiresIn  = flag.String("expires", "", "Expiration duration (e.g., 30d, 1y, or empty for no expiration)")
	output     = flag.String("output", "api-keys.yaml", "Output file path")
	append     = flag.Bool("append", false, "Append to existing file instead of creating new")
)

func main() {
	flag.Parse()

	if *name == "" {
		fmt.Println("Error: -name flag is required")
		flag.Usage()
		os.Exit(1)
	}

	// Parse expiration duration
	var expiresInDuration *time.Duration
	if *expiresIn != "" {
		duration, err := parseDuration(*expiresIn)
		if err != nil {
			log.Fatalf("Invalid expiration duration: %v", err)
		}
		expiresInDuration = &duration
	}

	// Create authentication manager
	authManager := auth.NewAuthManager()

	// Load existing keys if appending
	var existingKeys []*auth.APIKey
	if *append {
		keys, err := auth.LoadKeysFromFile(*output)
		if err != nil && !os.IsNotExist(err) {
			log.Printf("Warning: failed to load existing keys: %v", err)
		} else {
			existingKeys = keys
		}
	}

	// Generate new API key
	metadata := map[string]string{
		"generated_at": time.Now().Format(time.RFC3339),
		"generated_by": "keygen tool",
	}

	apiKey, err := authManager.GenerateAPIKey(*name, metadata, expiresInDuration)
	if err != nil {
		log.Fatalf("Failed to generate API key: %v", err)
	}

	// Combine with existing keys
	allKeys := append(existingKeys, apiKey)

	// Save to file
	if err := auth.SaveKeysToFile(*output, allKeys); err != nil {
		log.Fatalf("Failed to save API keys: %v", err)
	}

	// Print key information
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Println("  API Key Generated Successfully")
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Printf("  ID:         %s\n", apiKey.ID)
	fmt.Printf("  Name:       %s\n", apiKey.Name)
	fmt.Printf("  Created:    %s\n", apiKey.CreatedAt.Format(time.RFC3339))
	if apiKey.ExpiresAt != nil {
		fmt.Printf("  Expires:    %s\n", apiKey.ExpiresAt.Format(time.RFC3339))
	} else {
		fmt.Println("  Expires:    Never")
	}
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("  ⚠️  IMPORTANT: Copy this API key now!")
	fmt.Println("  It will not be shown again.")
	fmt.Println()
	fmt.Printf("  API Key: %s\n", apiKey.Key)
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("  Add this to your agent configuration:")
	fmt.Println()
	fmt.Println("  server:")
	fmt.Printf("    api_key: \"%s\"\n", apiKey.Key)
	fmt.Println()
	fmt.Printf("  Keys saved to: %s\n", *output)
	if *append {
		fmt.Printf("  Total keys in file: %d\n", len(allKeys))
	}
	fmt.Println()
}

func parseDuration(s string) (time.Duration, error) {
	// Handle common duration formats: 30d, 1y, etc.
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration format")
	}

	value := s[:len(s)-1]
	unit := s[len(s)-1:]

	var multiplier time.Duration
	switch unit {
	case "h":
		multiplier = time.Hour
	case "d":
		multiplier = 24 * time.Hour
	case "w":
		multiplier = 7 * 24 * time.Hour
	case "m":
		multiplier = 30 * 24 * time.Hour // Approximate month
	case "y":
		multiplier = 365 * 24 * time.Hour // Approximate year
	default:
		// Try parsing as standard Go duration
		return time.ParseDuration(s)
	}

	var count int
	if _, err := fmt.Sscanf(value, "%d", &count); err != nil {
		return 0, fmt.Errorf("invalid duration value: %w", err)
	}

	return time.Duration(count) * multiplier, nil
}
