package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/robohub/auth-service/internal/oidc"
	"github.com/robohub/auth-service/internal/policy"
	"github.com/robohub/auth-service/internal/ratelimit"
	"github.com/robohub/auth-service/internal/token"
	"github.com/robohub/auth-service/internal/types"
)

func TestHandleHealthz(t *testing.T) {
	server := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	server.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if w.Body.String() != "ok" {
		t.Errorf("expected body 'ok', got %s", w.Body.String())
	}
}

func TestHandleReadyz(t *testing.T) {
	server := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	server.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if w.Body.String() != "ok" {
		t.Errorf("expected body 'ok', got %s", w.Body.String())
	}
}

func TestHandleGitHubOIDC(t *testing.T) {
	t.Run("missing oidc_token", func(t *testing.T) {
		server := newTestServer()

		body := bytes.NewBufferString(`{}`)
		req := httptest.NewRequest(http.MethodPost, "/auth/github-oidc", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}

		var errResp types.ErrorResponse
		json.NewDecoder(w.Body).Decode(&errResp)
		if errResp.Error != "invalid_request" {
			t.Errorf("expected error 'invalid_request', got %s", errResp.Error)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		server := newTestServer()

		body := bytes.NewBufferString(`{invalid json}`)
		req := httptest.NewRequest(http.MethodPost, "/auth/github-oidc", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("successful token exchange", func(t *testing.T) {
		server := newTestServer()

		body := bytes.NewBufferString(`{"oidc_token": "valid-token"}`)
		req := httptest.NewRequest(http.MethodPost, "/auth/github-oidc", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp types.AuthResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.AccessToken == "" {
			t.Error("expected non-empty access_token")
		}

		if resp.TokenType != "Bearer" {
			t.Errorf("expected token_type 'Bearer', got %s", resp.TokenType)
		}

		if resp.ExpiresIn <= 0 {
			t.Errorf("expected positive expires_in, got %d", resp.ExpiresIn)
		}

		if resp.Subject.Provider != "github_actions" {
			t.Errorf("expected provider 'github_actions', got %s", resp.Subject.Provider)
		}

		if resp.Subject.Repository != "test/repo" {
			t.Errorf("expected repository 'test/repo', got %s", resp.Subject.Repository)
		}
	})

	t.Run("policy denied", func(t *testing.T) {
		// Create server with deny policy
		policyEnforcer := policy.NewEnforcer(false, "main", nil, []string{"test/repo"})
		server := &Server{
			logger:   slog.New(slog.NewTextHandler(os.Stderr, nil)),
			verifier: &oidc.FakeVerifier{},
			policy:   policyEnforcer,
			limiter:  ratelimit.NewLimiter(10.0, 10),
			minter:   token.NewMinter("test-secret", 10*time.Minute),
		}
		server.router = server.setupRouter()

		body := bytes.NewBufferString(`{"oidc_token": "valid-token"}`)
		req := httptest.NewRequest(http.MethodPost, "/auth/github-oidc", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected status 403, got %d", w.Code)
		}

		var errResp types.ErrorResponse
		json.NewDecoder(w.Body).Decode(&errResp)
		if errResp.Error != "policy_violation" {
			t.Errorf("expected error 'policy_violation', got %s", errResp.Error)
		}
	})

	t.Run("rate limited", func(t *testing.T) {
		// Create server with very restrictive rate limit
		limiter := ratelimit.NewLimiter(1.0, 1)
		server := &Server{
			logger:   slog.New(slog.NewTextHandler(os.Stderr, nil)),
			verifier: &oidc.FakeVerifier{},
			policy:   policy.NewEnforcer(false, "main", nil, nil),
			limiter:  limiter,
			minter:   token.NewMinter("test-secret", 10*time.Minute),
		}
		server.router = server.setupRouter()

		// First request should succeed
		body1 := bytes.NewBufferString(`{"oidc_token": "valid-token"}`)
		req1 := httptest.NewRequest(http.MethodPost, "/auth/github-oidc", body1)
		req1.Header.Set("Content-Type", "application/json")
		w1 := httptest.NewRecorder()
		server.Handler().ServeHTTP(w1, req1)

		if w1.Code != http.StatusOK {
			t.Errorf("expected first request to succeed with status 200, got %d", w1.Code)
		}

		// Second request should be rate limited
		body2 := bytes.NewBufferString(`{"oidc_token": "valid-token"}`)
		req2 := httptest.NewRequest(http.MethodPost, "/auth/github-oidc", body2)
		req2.Header.Set("Content-Type", "application/json")
		w2 := httptest.NewRecorder()
		server.Handler().ServeHTTP(w2, req2)

		if w2.Code != http.StatusTooManyRequests {
			t.Errorf("expected status 429, got %d", w2.Code)
		}

		var errResp types.ErrorResponse
		json.NewDecoder(w2.Body).Decode(&errResp)
		if errResp.Error != "rate_limited" {
			t.Errorf("expected error 'rate_limited', got %s", errResp.Error)
		}
	})

	t.Run("verification failure", func(t *testing.T) {
		// Create server with failing verifier
		failingVerifier := &oidc.FakeVerifier{
			VerifyFunc: func(ctx context.Context, token string) (*types.VerifiedClaims, error) {
				return nil, fmt.Errorf("verification failed")
			},
		}
		server := &Server{
			logger:   slog.New(slog.NewTextHandler(os.Stderr, nil)),
			verifier: failingVerifier,
			policy:   policy.NewEnforcer(false, "main", nil, nil),
			limiter:  ratelimit.NewLimiter(10.0, 10),
			minter:   token.NewMinter("test-secret", 10*time.Minute),
		}
		server.router = server.setupRouter()

		body := bytes.NewBufferString(`{"oidc_token": "invalid-token"}`)
		req := httptest.NewRequest(http.MethodPost, "/auth/github-oidc", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", w.Code)
		}

		var errResp types.ErrorResponse
		json.NewDecoder(w.Body).Decode(&errResp)
		if errResp.Error != "invalid_token" {
			t.Errorf("expected error 'invalid_token', got %s", errResp.Error)
		}
	})

	t.Run("default branch enforcement", func(t *testing.T) {
		// Create server with default branch enforcement
		policyEnforcer := policy.NewEnforcer(true, "main", nil, nil)
		server := &Server{
			logger: slog.New(slog.NewTextHandler(os.Stderr, nil)),
			verifier: &oidc.FakeVerifier{
				VerifyFunc: func(ctx context.Context, token string) (*types.VerifiedClaims, error) {
					return &types.VerifiedClaims{
						Repository: "test/repo",
						Ref:        "refs/heads/develop", // Not the default branch
						Actor:      "testuser",
						RunID:      "123456789",
						Workflow:   ".github/workflows/test.yml@refs/heads/develop",
						IssuedAt:   time.Now(),
						ExpiresAt:  time.Now().Add(1 * time.Hour),
					}, nil
				},
			},
			policy:  policyEnforcer,
			limiter: ratelimit.NewLimiter(10.0, 10),
			minter:  token.NewMinter("test-secret", 10*time.Minute),
		}
		server.router = server.setupRouter()

		body := bytes.NewBufferString(`{"oidc_token": "valid-token"}`)
		req := httptest.NewRequest(http.MethodPost, "/auth/github-oidc", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected status 403, got %d", w.Code)
		}
	})
}

func newTestServer() *Server {
	s := &Server{
		logger:   slog.New(slog.NewTextHandler(os.Stderr, nil)),
		verifier: &oidc.FakeVerifier{},
		policy:   policy.NewEnforcer(false, "main", nil, nil),
		limiter:  ratelimit.NewLimiter(10.0, 10),
		minter:   token.NewMinter("test-secret", 10*time.Minute),
	}
	s.router = s.setupRouter()
	return s
}

func (s *Server) withRouter() *Server {
	s.router = s.setupRouter()
	return s
}
