package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/robohub/auth-service/internal/config"
	"github.com/robohub/auth-service/internal/httpapi"
	"github.com/robohub/auth-service/internal/oidc"
	"github.com/robohub/auth-service/internal/policy"
	"github.com/robohub/auth-service/internal/ratelimit"
	"github.com/robohub/auth-service/internal/token"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Setup logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("starting robohub-auth service")

	// Load configuration
	cfg, err := config.LoadFromEnv()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	logger.Info("configuration loaded",
		"port", cfg.Port,
		"oidc_issuer", cfg.OIDCIssuer,
		"oidc_audience", cfg.OIDCAudience,
		"default_branch_only", cfg.DefaultBranchOnly,
		"default_branch", cfg.DefaultBranch,
		"token_ttl", cfg.TokenTTL,
		"rate_limit_rps", cfg.RateLimitRPS,
		"rate_limit_burst", cfg.RateLimitBurst,
	)

	// Initialize components
	verifier := oidc.NewGitHubVerifier(
		cfg.OIDCIssuer,
		cfg.OIDCAudience,
		cfg.ClockSkew,
		time.Duration(cfg.JWKSTTLSeconds)*time.Second,
	)

	policyEnforcer := policy.NewEnforcer(
		cfg.DefaultBranchOnly,
		cfg.DefaultBranch,
		cfg.RepoAllowList,
		cfg.RepoDenyList,
	)

	limiter := ratelimit.NewLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst)

	minter := token.NewMinter(cfg.JWTSecret, cfg.TokenTTL)

	// Create HTTP server
	apiServer := httpapi.NewServer(logger, verifier, policyEnforcer, limiter, minter)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      apiServer.Handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("server listening", "address", server.Addr)
		serverErrors <- server.ListenAndServe()
	}()

	// Wait for interrupt signal or server error
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)
	case sig := <-shutdown:
		logger.Info("shutdown signal received", "signal", sig)

		// Create context with timeout for shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		// Attempt graceful shutdown
		if err := server.Shutdown(ctx); err != nil {
			logger.Error("graceful shutdown failed", "error", err)
			if err := server.Close(); err != nil {
				return fmt.Errorf("failed to close server: %w", err)
			}
		}

		logger.Info("server stopped gracefully")
	}

	return nil
}
