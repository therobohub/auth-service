package oidc

import (
	"context"
	"time"

	"github.com/robohub/auth-service/internal/types"
)

// FakeVerifier is a test implementation of Verifier
type FakeVerifier struct {
	VerifyFunc func(ctx context.Context, token string) (*types.VerifiedClaims, error)
}

// Verify implements the Verifier interface
func (f *FakeVerifier) Verify(ctx context.Context, token string) (*types.VerifiedClaims, error) {
	if f.VerifyFunc != nil {
		return f.VerifyFunc(ctx, token)
	}
	// Default successful verification
	return &types.VerifiedClaims{
		Repository: "test/repo",
		Ref:        "refs/heads/main",
		Actor:      "testuser",
		RunID:      "123456789",
		Workflow:   ".github/workflows/test.yml@refs/heads/main",
		IssuedAt:   time.Now(),
		ExpiresAt:  time.Now().Add(1 * time.Hour),
	}, nil
}
