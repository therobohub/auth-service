package oidc

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/robohub/auth-service/internal/types"
)

// Verifier defines the interface for verifying OIDC tokens
type Verifier interface {
	Verify(ctx context.Context, token string) (*types.VerifiedClaims, error)
}

// GitHubVerifier verifies GitHub Actions OIDC tokens
type GitHubVerifier struct {
	issuer    string
	audience  string
	clockSkew time.Duration
	jwksCache *JWKSCache
}

// NewGitHubVerifier creates a new GitHub OIDC verifier
func NewGitHubVerifier(issuer, audience string, clockSkew time.Duration, jwksTTL time.Duration) *GitHubVerifier {
	return &GitHubVerifier{
		issuer:    issuer,
		audience:  audience,
		clockSkew: clockSkew,
		jwksCache: NewJWKSCache(issuer+"/.well-known/jwks", jwksTTL),
	}
}

// Verify verifies a GitHub Actions OIDC token
func (v *GitHubVerifier) Verify(ctx context.Context, tokenString string) (*types.VerifiedClaims, error) {
	// Parse token to get kid from header
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Get kid from header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid kid in token header")
		}

		// Get public key from JWKS
		publicKey, err := v.jwksCache.GetKey(ctx, kid)
		if err != nil {
			return nil, fmt.Errorf("failed to get public key: %w", err)
		}

		return publicKey, nil
	}, jwt.WithLeeway(v.clockSkew))

	if err != nil {
		return nil, fmt.Errorf("failed to verify token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("token is invalid")
	}

	// Extract and validate claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims format")
	}

	// Validate issuer
	iss, ok := claims["iss"].(string)
	if !ok || iss != v.issuer {
		return nil, fmt.Errorf("invalid issuer: expected %s, got %s", v.issuer, iss)
	}

	// Validate audience
	aud, err := v.extractAudience(claims)
	if err != nil {
		return nil, fmt.Errorf("invalid audience: %w", err)
	}
	if !v.containsAudience(aud, v.audience) {
		return nil, fmt.Errorf("audience does not match: expected %s", v.audience)
	}

	// Extract required claims
	repository, ok := claims["repository"].(string)
	if !ok || repository == "" {
		return nil, fmt.Errorf("missing or invalid repository claim")
	}

	ref, ok := claims["ref"].(string)
	if !ok || ref == "" {
		return nil, fmt.Errorf("missing or invalid ref claim")
	}

	actor, ok := claims["actor"].(string)
	if !ok || actor == "" {
		return nil, fmt.Errorf("missing or invalid actor claim")
	}

	// Extract run_id (can be string or number)
	runID := v.extractRunID(claims)
	if runID == "" {
		return nil, fmt.Errorf("missing or invalid run_id claim")
	}

	// Extract workflow (try workflow_ref first, then job_workflow_ref)
	workflow := ""
	if wf, ok := claims["workflow_ref"].(string); ok {
		workflow = wf
	} else if jwf, ok := claims["job_workflow_ref"].(string); ok {
		workflow = jwf
	}
	if workflow == "" {
		return nil, fmt.Errorf("missing workflow_ref or job_workflow_ref claim")
	}

	// Extract timestamps
	iat := v.extractTimestamp(claims, "iat")
	exp := v.extractTimestamp(claims, "exp")

	return &types.VerifiedClaims{
		Repository: repository,
		Ref:        ref,
		Actor:      actor,
		RunID:      runID,
		Workflow:   workflow,
		IssuedAt:   iat,
		ExpiresAt:  exp,
	}, nil
}

func (v *GitHubVerifier) extractAudience(claims jwt.MapClaims) ([]string, error) {
	aud := claims["aud"]
	switch a := aud.(type) {
	case string:
		return []string{a}, nil
	case []interface{}:
		result := make([]string, 0, len(a))
		for _, item := range a {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result, nil
	default:
		return nil, fmt.Errorf("invalid audience type")
	}
}

func (v *GitHubVerifier) containsAudience(audiences []string, expected string) bool {
	for _, aud := range audiences {
		if aud == expected {
			return true
		}
	}
	return false
}

func (v *GitHubVerifier) extractRunID(claims jwt.MapClaims) string {
	if runID, ok := claims["run_id"].(string); ok {
		return runID
	}
	if runID, ok := claims["run_id"].(float64); ok {
		return fmt.Sprintf("%.0f", runID)
	}
	return ""
}

func (v *GitHubVerifier) extractTimestamp(claims jwt.MapClaims, key string) time.Time {
	if val, ok := claims[key].(float64); ok {
		return time.Unix(int64(val), 0)
	}
	return time.Time{}
}

// JWKSCache caches JWKS keys
type JWKSCache struct {
	url        string
	ttl        time.Duration
	mu         sync.RWMutex
	keys       map[string]*rsa.PublicKey
	fetchedAt  time.Time
	httpClient *http.Client
}

// NewJWKSCache creates a new JWKS cache
func NewJWKSCache(url string, ttl time.Duration) *JWKSCache {
	return &JWKSCache{
		url:        url,
		ttl:        ttl,
		keys:       make(map[string]*rsa.PublicKey),
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// GetKey retrieves a public key by kid
func (c *JWKSCache) GetKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	// Check cache first
	c.mu.RLock()
	if key, exists := c.keys[kid]; exists && time.Since(c.fetchedAt) < c.ttl {
		c.mu.RUnlock()
		return key, nil
	}
	c.mu.RUnlock()

	// Fetch JWKS
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if key, exists := c.keys[kid]; exists && time.Since(c.fetchedAt) < c.ttl {
		return key, nil
	}

	// Fetch from remote
	if err := c.fetchJWKS(ctx); err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}

	key, exists := c.keys[kid]
	if !exists {
		return nil, fmt.Errorf("key with kid %s not found in JWKS", kid)
	}

	return key, nil
}

func (c *JWKSCache) fetchJWKS(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	var jwks struct {
		Keys []struct {
			Kid string `json:"kid"`
			Kty string `json:"kty"`
			Use string `json:"use"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}

	if err := json.Unmarshal(body, &jwks); err != nil {
		return fmt.Errorf("failed to unmarshal JWKS: %w", err)
	}

	// Parse and cache keys
	newKeys := make(map[string]*rsa.PublicKey)
	for _, key := range jwks.Keys {
		if key.Kty != "RSA" {
			continue
		}

		pubKey, err := parseRSAPublicKey(key.N, key.E)
		if err != nil {
			continue // Skip invalid keys
		}

		newKeys[key.Kid] = pubKey
	}

	c.keys = newKeys
	c.fetchedAt = time.Now()

	return nil
}

func parseRSAPublicKey(nStr, eStr string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode n: %w", err)
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(eStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode e: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := 0
	for _, b := range eBytes {
		e = e*256 + int(b)
	}

	return &rsa.PublicKey{
		N: n,
		E: e,
	}, nil
}
