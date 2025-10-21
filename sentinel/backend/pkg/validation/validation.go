package validation

import (
	"encoding/json"
	"fmt"
	"regexp"
)

// ValidateEventData validates incoming event data from agents
func ValidateEventData(data []byte) error {
	// Check size limits
	if len(data) == 0 {
		return fmt.Errorf("empty event data")
	}

	if len(data) > 1024*1024 { // 1MB limit
		return fmt.Errorf("event data too large: %d bytes", len(data))
	}

	// Validate JSON structure
	var batch map[string]interface{}
	if err := json.Unmarshal(data, &batch); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Validate required fields
	if _, ok := batch["events"]; !ok {
		return fmt.Errorf("missing 'events' field")
	}

	if _, ok := batch["timestamp"]; !ok {
		return fmt.Errorf("missing 'timestamp' field")
	}

	// Validate events is an array
	events, ok := batch["events"].([]interface{})
	if !ok {
		return fmt.Errorf("'events' must be an array")
	}

	// Limit number of events per batch
	if len(events) > 1000 {
		return fmt.Errorf("too many events in batch: %d (max 1000)", len(events))
	}

	// Validate each event
	for i, rawEvent := range events {
		eventMap, ok := rawEvent.(map[string]interface{})
		if !ok {
			return fmt.Errorf("event %d is not an object", i)
		}

		if err := validateEvent(eventMap); err != nil {
			return fmt.Errorf("event %d: %w", i, err)
		}
	}

	return nil
}

func validateEvent(event map[string]interface{}) error {
	// Validate required fields
	requiredFields := []string{"type", "hostname"}
	for _, field := range requiredFields {
		if _, ok := event[field]; !ok {
			return fmt.Errorf("missing required field: %s", field)
		}
	}

	// Validate type
	eventType, ok := event["type"].(string)
	if !ok {
		return fmt.Errorf("'type' must be a string")
	}

	if eventType != "network" && eventType != "process" {
		return fmt.Errorf("invalid event type: %s (must be 'network' or 'process')", eventType)
	}

	// Validate hostname
	hostname, ok := event["hostname"].(string)
	if !ok {
		return fmt.Errorf("'hostname' must be a string")
	}

	if len(hostname) == 0 || len(hostname) > 255 {
		return fmt.Errorf("invalid hostname length: %d", len(hostname))
	}

	// Validate hostname format (RFC 1123)
	hostnameRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$`)
	if !hostnameRegex.MatchString(hostname) {
		return fmt.Errorf("invalid hostname format: %s", hostname)
	}

	// Type-specific validation
	if eventType == "network" {
		if err := validateNetworkEvent(event); err != nil {
			return err
		}
	} else if eventType == "process" {
		if err := validateProcessEvent(event); err != nil {
			return err
		}
	}

	return nil
}

func validateNetworkEvent(event map[string]interface{}) error {
	// Validate event_type
	eventType, ok := event["event_type"].(string)
	if !ok {
		return fmt.Errorf("'event_type' must be a string")
	}

	validEventTypes := map[string]bool{
		"connect": true,
		"accept":  true,
		"close":   true,
	}

	if !validEventTypes[eventType] {
		return fmt.Errorf("invalid network event_type: %s", eventType)
	}

	// Validate numeric fields
	numericFields := []string{"pid", "uid", "gid", "src_port", "dst_port"}
	for _, field := range numericFields {
		if val, ok := event[field]; ok {
			if _, isFloat := val.(float64); !isFloat {
				return fmt.Errorf("'%s' must be a number", field)
			}
		}
	}

	return nil
}

func validateProcessEvent(event map[string]interface{}) error {
	// Validate event_type
	eventType, ok := event["event_type"].(string)
	if !ok {
		return fmt.Errorf("'event_type' must be a string")
	}

	validEventTypes := map[string]bool{
		"exec": true,
		"exit": true,
	}

	if !validEventTypes[eventType] {
		return fmt.Errorf("invalid process event_type: %s", eventType)
	}

	// Validate numeric fields
	numericFields := []string{"pid", "ppid", "uid", "gid"}
	for _, field := range numericFields {
		if val, ok := event[field]; ok {
			if _, isFloat := val.(float64); !isFloat {
				return fmt.Errorf("'%s' must be a number", field)
			}
		}
	}

	// Validate string length limits
	if comm, ok := event["comm"].(string); ok {
		if len(comm) > 255 {
			return fmt.Errorf("'comm' too long: %d characters", len(comm))
		}
	}

	if username, ok := event["username"].(string); ok {
		if len(username) > 255 {
			return fmt.Errorf("'username' too long: %d characters", len(username))
		}
	}

	return nil
}
