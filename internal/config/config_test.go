package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadFromEnv(t *testing.T) {
	// Save original env
	originalEnv := make(map[string]string)
	for _, key := range []string{
		"PORT", "ROBOHUB_JWT_SECRET", "ROBOHUB_OIDC_ISSUER", "ROBOHUB_OIDC_AUDIENCE",
		"ROBOHUB_CLOCK_SKEW_SECONDS", "ROBOHUB_JWKS_TTL_SECONDS", "ROBOHUB_DEFAULT_BRANCH_ONLY",
		"ROBOHUB_DEFAULT_BRANCH", "ROBOHUB_REPO_DENYLIST", "ROBOHUB_REPO_ALLOWLIST",
		"ROBOHUB_RATE_LIMIT_RPS", "ROBOHUB_RATE_LIMIT_BURST", "ROBOHUB_TOKEN_TTL_SECONDS",
	} {
		originalEnv[key] = os.Getenv(key)
	}

	// Restore env after test
	defer func() {
		for key, value := range originalEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	t.Run("missing JWT secret", func(t *testing.T) {
		os.Clearenv()
		_, err := LoadFromEnv()
		if err == nil {
			t.Error("expected error when JWT secret is missing")
		}
	})

	t.Run("defaults", func(t *testing.T) {
		os.Clearenv()
		os.Setenv("ROBOHUB_JWT_SECRET", "test-secret")

		cfg, err := LoadFromEnv()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cfg.Port != "8080" {
			t.Errorf("expected port 8080, got %s", cfg.Port)
		}
		if cfg.OIDCIssuer != "https://token.actions.githubusercontent.com" {
			t.Errorf("unexpected issuer: %s", cfg.OIDCIssuer)
		}
		if cfg.OIDCAudience != "robohub" {
			t.Errorf("unexpected audience: %s", cfg.OIDCAudience)
		}
		if cfg.ClockSkew != 60*time.Second {
			t.Errorf("unexpected clock skew: %v", cfg.ClockSkew)
		}
		if cfg.DefaultBranch != "main" {
			t.Errorf("unexpected default branch: %s", cfg.DefaultBranch)
		}
		if cfg.TokenTTL != 600*time.Second {
			t.Errorf("unexpected token TTL: %v", cfg.TokenTTL)
		}
	})

	t.Run("custom values", func(t *testing.T) {
		os.Clearenv()
		os.Setenv("ROBOHUB_JWT_SECRET", "custom-secret")
		os.Setenv("PORT", "9090")
		os.Setenv("ROBOHUB_DEFAULT_BRANCH_ONLY", "true")
		os.Setenv("ROBOHUB_DEFAULT_BRANCH", "develop")
		os.Setenv("ROBOHUB_REPO_DENYLIST", "evil/repo,bad/actor")
		os.Setenv("ROBOHUB_REPO_ALLOWLIST", "good/repo")
		os.Setenv("ROBOHUB_RATE_LIMIT_RPS", "2.5")
		os.Setenv("ROBOHUB_RATE_LIMIT_BURST", "10")
		os.Setenv("ROBOHUB_TOKEN_TTL_SECONDS", "300")

		cfg, err := LoadFromEnv()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cfg.Port != "9090" {
			t.Errorf("expected port 9090, got %s", cfg.Port)
		}
		if !cfg.DefaultBranchOnly {
			t.Error("expected DefaultBranchOnly to be true")
		}
		if cfg.DefaultBranch != "develop" {
			t.Errorf("unexpected default branch: %s", cfg.DefaultBranch)
		}
		if len(cfg.RepoDenyList) != 2 {
			t.Errorf("expected 2 denied repos, got %d", len(cfg.RepoDenyList))
		}
		if len(cfg.RepoAllowList) != 1 {
			t.Errorf("expected 1 allowed repo, got %d", len(cfg.RepoAllowList))
		}
		if cfg.RateLimitRPS != 2.5 {
			t.Errorf("unexpected rate limit RPS: %f", cfg.RateLimitRPS)
		}
		if cfg.RateLimitBurst != 10 {
			t.Errorf("unexpected rate limit burst: %d", cfg.RateLimitBurst)
		}
		if cfg.TokenTTL != 300*time.Second {
			t.Errorf("unexpected token TTL: %v", cfg.TokenTTL)
		}
	})
}

func TestParseCommaSeparated(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"empty", "", []string{}},
		{"single", "repo/a", []string{"repo/a"}},
		{"multiple", "repo/a,repo/b,repo/c", []string{"repo/a", "repo/b", "repo/c"}},
		{"with spaces", "repo/a, repo/b , repo/c", []string{"repo/a", "repo/b", "repo/c"}},
		{"trailing comma", "repo/a,repo/b,", []string{"repo/a", "repo/b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCommaSeparated(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d items, got %d", len(tt.expected), len(result))
				return
			}
			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("expected %s at index %d, got %s", expected, i, result[i])
				}
			}
		})
	}
}
