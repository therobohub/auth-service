package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/robohub/auth-service/internal/oidc"
	"github.com/robohub/auth-service/internal/policy"
	"github.com/robohub/auth-service/internal/ratelimit"
	"github.com/robohub/auth-service/internal/token"
	"github.com/robohub/auth-service/internal/types"
)

// Server holds the HTTP API server
type Server struct {
	router    chi.Router
	logger    *slog.Logger
	verifier  oidc.Verifier
	policy    *policy.Enforcer
	limiter   *ratelimit.Limiter
	minter    *token.Minter
}

// NewServer creates a new HTTP API server
func NewServer(
	logger *slog.Logger,
	verifier oidc.Verifier,
	policyEnforcer *policy.Enforcer,
	limiter *ratelimit.Limiter,
	minter *token.Minter,
) *Server {
	s := &Server{
		logger:   logger,
		verifier: verifier,
		policy:   policyEnforcer,
		limiter:  limiter,
		minter:   minter,
	}

	s.router = s.setupRouter()
	return s
}

func (s *Server) setupRouter() chi.Router {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(s.loggingMiddleware)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	// Routes
	r.Get("/healthz", s.handleHealthz)
	r.Get("/readyz", s.handleReadyz)
	r.Post("/auth/github-oidc", s.handleGitHubOIDC)

	return r
}

// Handler returns the HTTP handler
func (s *Server) Handler() http.Handler {
	return s.router
}

// handleHealthz handles health check requests
func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// handleReadyz handles readiness check requests
func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// handleGitHubOIDC handles GitHub OIDC token exchange
func (s *Server) handleGitHubOIDC(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request
	var req types.AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logger.WarnContext(ctx, "invalid request body", "error", err)
		s.respondError(w, http.StatusBadRequest, "invalid_request", "invalid JSON in request body")
		return
	}

	if req.OIDCToken == "" {
		s.logger.WarnContext(ctx, "missing oidc_token")
		s.respondError(w, http.StatusBadRequest, "invalid_request", "missing oidc_token field")
		return
	}

	// Verify OIDC token
	claims, err := s.verifier.Verify(ctx, req.OIDCToken)
	if err != nil {
		s.logger.WarnContext(ctx, "failed to verify OIDC token", "error", err)
		s.respondError(w, http.StatusUnauthorized, "invalid_token", "failed to verify OIDC token")
		return
	}

	s.logger.InfoContext(ctx, "verified OIDC token",
		"repository", claims.Repository,
		"ref", claims.Ref,
		"actor", claims.Actor,
		"run_id", claims.RunID,
	)

	// Check rate limit
	if !s.limiter.Allow(claims.Repository) {
		s.logger.WarnContext(ctx, "rate limit exceeded",
			"repository", claims.Repository,
		)
		s.respondError(w, http.StatusTooManyRequests, "rate_limited", "rate limit exceeded for repository")
		return
	}

	// Check policy
	if err := s.policy.Evaluate(claims.Repository, claims.Ref); err != nil {
		s.logger.WarnContext(ctx, "policy violation",
			"repository", claims.Repository,
			"ref", claims.Ref,
			"error", err,
		)
		s.respondError(w, http.StatusForbidden, "policy_violation", err.Error())
		return
	}

	// Mint access token
	accessToken, expiresAt, err := s.minter.Mint(claims)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to mint token", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "failed to create access token")
		return
	}

	expiresIn := int(time.Until(expiresAt).Seconds())

	resp := types.AuthResponse{
		AccessToken: accessToken,
		ExpiresIn:   expiresIn,
		TokenType:   "Bearer",
		IssuedAt:    time.Now().Format(time.RFC3339),
		Subject: types.SubjectDetails{
			Provider:   "github_actions",
			Repository: claims.Repository,
			Ref:        claims.Ref,
			Workflow:   claims.Workflow,
			RunID:      claims.RunID,
			Actor:      claims.Actor,
		},
	}

	s.logger.InfoContext(ctx, "issued access token",
		"repository", claims.Repository,
		"expires_in", expiresIn,
	)

	s.respondJSON(w, http.StatusOK, resp)
}

func (s *Server) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *Server) respondError(w http.ResponseWriter, status int, errorCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(types.ErrorResponse{
		Error:   errorCode,
		Message: message,
	})
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		s.logger.InfoContext(r.Context(), "request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
			"remote_addr", r.RemoteAddr,
		)
	})
}
