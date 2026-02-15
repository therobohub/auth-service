package token

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/robohub/auth-service/internal/types"
)

// Minter creates RoboHub access tokens
type Minter struct {
	secret []byte
	ttl    time.Duration
}

// NewMinter creates a new token minter
func NewMinter(secret string, ttl time.Duration) *Minter {
	return &Minter{
		secret: []byte(secret),
		ttl:    ttl,
	}
}

// Mint creates a new RoboHub access token
func (m *Minter) Mint(claims *types.VerifiedClaims) (string, time.Time, error) {
	now := time.Now()
	exp := now.Add(m.ttl)

	tokenClaims := jwt.MapClaims{
		"iss":    "robohub-auth",
		"sub":    fmt.Sprintf("repo:%s", claims.Repository),
		"aud":    "robohub-api",
		"iat":    now.Unix(),
		"exp":    exp.Unix(),
		"jti":    uuid.New().String(),
		"repo":   claims.Repository,
		"ref":    claims.Ref,
		"actor":  claims.Actor,
		"run_id": claims.RunID,
		"scopes": []string{"ingest:build"},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenClaims)
	tokenString, err := token.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, exp, nil
}

// Validate validates and parses a RoboHub access token
func (m *Minter) Validate(tokenString string) (*types.RoboHubClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims format")
	}

	// Extract claims
	robohubClaims := &types.RoboHubClaims{}

	if iss, ok := claims["iss"].(string); ok {
		robohubClaims.Issuer = iss
	}
	if sub, ok := claims["sub"].(string); ok {
		robohubClaims.Subject = sub
	}
	if aud, ok := claims["aud"].(string); ok {
		robohubClaims.Audience = aud
	}
	if iat, ok := claims["iat"].(float64); ok {
		robohubClaims.IssuedAt = int64(iat)
	}
	if exp, ok := claims["exp"].(float64); ok {
		robohubClaims.ExpiresAt = int64(exp)
	}
	if jti, ok := claims["jti"].(string); ok {
		robohubClaims.JTI = jti
	}
	if repo, ok := claims["repo"].(string); ok {
		robohubClaims.Repo = repo
	}
	if ref, ok := claims["ref"].(string); ok {
		robohubClaims.Ref = ref
	}
	if actor, ok := claims["actor"].(string); ok {
		robohubClaims.Actor = actor
	}
	if runID, ok := claims["run_id"].(string); ok {
		robohubClaims.RunID = runID
	}
	if scopes, ok := claims["scopes"].([]interface{}); ok {
		robohubClaims.Scopes = make([]string, 0, len(scopes))
		for _, scope := range scopes {
			if s, ok := scope.(string); ok {
				robohubClaims.Scopes = append(robohubClaims.Scopes, s)
			}
		}
	}

	return robohubClaims, nil
}
