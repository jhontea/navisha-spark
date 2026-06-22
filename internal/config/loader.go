package config

import (
	"fmt"
	"os"
)

// GetEnv gets environment variable with fallback.
func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// GetEnvInt gets environment variable as int with fallback.
func GetEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		var result int
		if _, err := fmt.Sscanf(value, "%d", &result); err == nil {
			return result
		}
	}
	return fallback
}

// GetEnvBool gets environment variable as bool with fallback.
func GetEnvBool(key string, fallback bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		if value == "true" || value == "1" || value == "yes" {
			return true
		}
		return false
	}
	return fallback
}

// MustConfig loads config or panics.
func MustConfig(configPath string) *Config {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	return cfg
}
