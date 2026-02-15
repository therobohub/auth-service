package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration
type Config struct {
	// Server
	Port string

	// JWT Secret for signing RoboHub tokens
	JWTSecret string

	// OIDC Configuration
	OIDCIssuer     string
	OIDCAudience   string
	ClockSkew      time.Duration
	JWKSTTLSeconds int

	// Policy Configuration
	DefaultBranchOnly bool
	DefaultBranch     string
	RepoDenyList      []string
	RepoAllowList     []string

	// Rate Limiting
	RateLimitRPS   float64
	RateLimitBurst int

	// Token Configuration
	TokenTTL time.Duration
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv() (*Config, error) {
	cfg := &Config{
		Port:              getEnv("PORT", "8080"),
		JWTSecret:         os.Getenv("ROBOHUB_JWT_SECRET"),
		OIDCIssuer:        getEnv("ROBOHUB_OIDC_ISSUER", "https://token.actions.githubusercontent.com"),
		OIDCAudience:      getEnv("ROBOHUB_OIDC_AUDIENCE", "robohub"),
		ClockSkew:         time.Duration(getEnvInt("ROBOHUB_CLOCK_SKEW_SECONDS", 60)) * time.Second,
		JWKSTTLSeconds:    getEnvInt("ROBOHUB_JWKS_TTL_SECONDS", 3600),
		DefaultBranchOnly: getEnvBool("ROBOHUB_DEFAULT_BRANCH_ONLY", false),
		DefaultBranch:     getEnv("ROBOHUB_DEFAULT_BRANCH", "main"),
		RepoDenyList:      parseCommaSeparated(getEnv("ROBOHUB_REPO_DENYLIST", "")),
		RepoAllowList:     parseCommaSeparated(getEnv("ROBOHUB_REPO_ALLOWLIST", "")),
		RateLimitRPS:      getEnvFloat("ROBOHUB_RATE_LIMIT_RPS", 1.0),
		RateLimitBurst:    getEnvInt("ROBOHUB_RATE_LIMIT_BURST", 5),
		TokenTTL:          time.Duration(getEnvInt("ROBOHUB_TOKEN_TTL_SECONDS", 600)) * time.Second,
	}

	// Validate required fields
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("ROBOHUB_JWT_SECRET is required")
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			return floatVal
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

func parseCommaSeparated(value string) []string {
	if value == "" {
		return []string{}
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
