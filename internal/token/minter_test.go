package token

import (
	"testing"
	"time"

	"github.com/robohub/auth-service/internal/types"
)

func TestMinter_Mint(t *testing.T) {
	minter := NewMinter("test-secret", 10*time.Minute)

	claims := &types.VerifiedClaims{
		Repository: "owner/repo",
		Ref:        "refs/heads/main",
		Actor:      "testuser",
		RunID:      "123456789",
		Workflow:   ".github/workflows/test.yml@refs/heads/main",
		IssuedAt:   time.Now(),
		ExpiresAt:  time.Now().Add(1 * time.Hour),
	}

	tokenString, exp, err := minter.Mint(claims)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tokenString == "" {
		t.Error("expected non-empty token string")
	}

	if exp.IsZero() {
		t.Error("expected non-zero expiration time")
	}

	// Verify the token is valid
	parsed, err := minter.Validate(tokenString)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}

	if parsed.Issuer != "robohub-auth" {
		t.Errorf("expected issuer robohub-auth, got %s", parsed.Issuer)
	}

	if parsed.Subject != "repo:owner/repo" {
		t.Errorf("expected subject repo:owner/repo, got %s", parsed.Subject)
	}

	if parsed.Audience != "robohub-api" {
		t.Errorf("expected audience robohub-api, got %s", parsed.Audience)
	}

	if parsed.Repo != "owner/repo" {
		t.Errorf("expected repo owner/repo, got %s", parsed.Repo)
	}

	if parsed.Ref != "refs/heads/main" {
		t.Errorf("expected ref refs/heads/main, got %s", parsed.Ref)
	}

	if parsed.Actor != "testuser" {
		t.Errorf("expected actor testuser, got %s", parsed.Actor)
	}

	if parsed.RunID != "123456789" {
		t.Errorf("expected run_id 123456789, got %s", parsed.RunID)
	}

	if len(parsed.Scopes) != 1 || parsed.Scopes[0] != "ingest:build" {
		t.Errorf("expected scopes [ingest:build], got %v", parsed.Scopes)
	}

	if parsed.JTI == "" {
		t.Error("expected non-empty JTI")
	}
}

func TestMinter_Validate(t *testing.T) {
	minter := NewMinter("test-secret", 10*time.Minute)

	claims := &types.VerifiedClaims{
		Repository: "owner/repo",
		Ref:        "refs/heads/main",
		Actor:      "testuser",
		RunID:      "123456789",
		Workflow:   ".github/workflows/test.yml@refs/heads/main",
		IssuedAt:   time.Now(),
		ExpiresAt:  time.Now().Add(1 * time.Hour),
	}

	tokenString, _, err := minter.Mint(claims)
	if err != nil {
		t.Fatalf("failed to mint token: %v", err)
	}

	t.Run("valid token", func(t *testing.T) {
		parsed, err := minter.Validate(tokenString)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if parsed == nil {
			t.Fatal("expected non-nil claims")
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		_, err := minter.Validate("invalid.token.string")
		if err == nil {
			t.Error("expected error for invalid token")
		}
	})

	t.Run("wrong secret", func(t *testing.T) {
		wrongMinter := NewMinter("wrong-secret", 10*time.Minute)
		_, err := wrongMinter.Validate(tokenString)
		if err == nil {
			t.Error("expected error for wrong secret")
		}
	})

	t.Run("expired token", func(t *testing.T) {
		shortMinter := NewMinter("test-secret", 1*time.Nanosecond)
		expiredToken, _, err := shortMinter.Mint(claims)
		if err != nil {
			t.Fatalf("failed to mint token: %v", err)
		}

		time.Sleep(10 * time.Millisecond)

		_, err = shortMinter.Validate(expiredToken)
		if err == nil {
			t.Error("expected error for expired token")
		}
	})
}

func TestMinter_TTL(t *testing.T) {
	ttl := 5 * time.Minute
	minter := NewMinter("test-secret", ttl)

	claims := &types.VerifiedClaims{
		Repository: "owner/repo",
		Ref:        "refs/heads/main",
		Actor:      "testuser",
		RunID:      "123456789",
		Workflow:   ".github/workflows/test.yml@refs/heads/main",
		IssuedAt:   time.Now(),
		ExpiresAt:  time.Now().Add(1 * time.Hour),
	}

	before := time.Now()
	_, exp, err := minter.Mint(claims)
	after := time.Now()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedExp := before.Add(ttl)
	if exp.Before(expectedExp) || exp.After(after.Add(ttl)) {
		t.Errorf("expiration time out of expected range: got %v, expected around %v", exp, expectedExp)
	}
}
