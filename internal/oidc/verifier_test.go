package oidc

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/robohub/auth-service/internal/types"
)

func TestGitHubVerifier_extractAudience(t *testing.T) {
	v := &GitHubVerifier{}

	tests := []struct {
		name      string
		claims    map[string]interface{}
		wantAud   []string
		wantError bool
	}{
		{
			name:      "string audience",
			claims:    map[string]interface{}{"aud": "robohub"},
			wantAud:   []string{"robohub"},
			wantError: false,
		},
		{
			name:      "array audience",
			claims:    map[string]interface{}{"aud": []interface{}{"robohub", "other"}},
			wantAud:   []string{"robohub", "other"},
			wantError: false,
		},
		{
			name:      "invalid audience type",
			claims:    map[string]interface{}{"aud": 123},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aud, err := v.extractAudience(tt.claims)
			if (err != nil) != tt.wantError {
				t.Errorf("expected error=%v, got error=%v", tt.wantError, err)
			}
			if !tt.wantError && len(aud) != len(tt.wantAud) {
				t.Errorf("expected %v, got %v", tt.wantAud, aud)
			}
		})
	}
}

func TestGitHubVerifier_containsAudience(t *testing.T) {
	v := &GitHubVerifier{}

	tests := []struct {
		name      string
		audiences []string
		expected  string
		want      bool
	}{
		{"contains", []string{"robohub", "other"}, "robohub", true},
		{"not contains", []string{"other"}, "robohub", false},
		{"empty", []string{}, "robohub", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := v.containsAudience(tt.audiences, tt.expected); got != tt.want {
				t.Errorf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestGitHubVerifier_extractRunID(t *testing.T) {
	v := &GitHubVerifier{}

	tests := []struct {
		name   string
		claims map[string]interface{}
		want   string
	}{
		{"string run_id", map[string]interface{}{"run_id": "123456789"}, "123456789"},
		{"number run_id", map[string]interface{}{"run_id": 123456789.0}, "123456789"},
		{"missing run_id", map[string]interface{}{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := v.extractRunID(tt.claims); got != tt.want {
				t.Errorf("expected %s, got %s", tt.want, got)
			}
		})
	}
}

func TestGitHubVerifier_extractTimestamp(t *testing.T) {
	v := &GitHubVerifier{}

	now := time.Now().Unix()
	claims := map[string]interface{}{"iat": float64(now)}

	result := v.extractTimestamp(claims, "iat")
	if result.Unix() != now {
		t.Errorf("expected %d, got %d", now, result.Unix())
	}

	// Missing timestamp should return zero time
	result = v.extractTimestamp(map[string]interface{}{}, "missing")
	if !result.IsZero() {
		t.Errorf("expected zero time for missing timestamp")
	}
}

func TestFakeVerifier(t *testing.T) {
	ctx := context.Background()

	t.Run("default behavior", func(t *testing.T) {
		fake := &FakeVerifier{}
		claims, err := fake.Verify(ctx, "dummy-token")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if claims.Repository != "test/repo" {
			t.Errorf("unexpected repository: %s", claims.Repository)
		}
	})

	t.Run("custom behavior", func(t *testing.T) {
		fake := &FakeVerifier{
			VerifyFunc: func(ctx context.Context, token string) (*types.VerifiedClaims, error) {
				return nil, fmt.Errorf("custom error")
			},
		}
		_, err := fake.Verify(ctx, "dummy-token")
		if err == nil {
			t.Error("expected error")
		}
		if err.Error() != "custom error" {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestParseRSAPublicKey(t *testing.T) {
	// Test with valid RSA key components (example from GitHub's JWKS)
	// These are base64url encoded modulus and exponent
	n := "xjlCRBqkQRiii6JJzkKNlLYNrwqqCRsf3a0g6s7dTbZSJmNvL0gVKfT_2GqM2cPbhGqJqJL9lFXJ5gZnMgSvVFCLkEYpQY3rR-pJQzkJFM1lLqJFd7QJIxJQlJQJpJGnJn9LjQQUKB6LQJ9n-2MnQQNnQmJMJJnQnQJMQQJnQQnQ"
	e := "AQAB"

	key, err := parseRSAPublicKey(n, e)
	if err != nil {
		t.Fatalf("expected valid key, got error: %v", err)
	}
	if key == nil {
		t.Fatal("expected non-nil key")
	}
	if key.E != 65537 { // AQAB is standard exponent 65537
		t.Errorf("expected exponent 65537, got %d", key.E)
	}
}

func TestJWKSCache(t *testing.T) {
	// Basic cache test - we can't test real JWKS fetching without a mock server
	// but we can test the cache structure
	cache := NewJWKSCache("https://example.com/.well-known/jwks", 1*time.Hour)
	if cache == nil {
		t.Fatal("expected non-nil cache")
	}
	if cache.url != "https://example.com/.well-known/jwks" {
		t.Errorf("unexpected URL: %s", cache.url)
	}
	if cache.ttl != 1*time.Hour {
		t.Errorf("unexpected TTL: %v", cache.ttl)
	}
}
