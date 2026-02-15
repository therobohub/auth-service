package types

import "time"

// AuthRequest represents the incoming OIDC token exchange request
type AuthRequest struct {
	OIDCToken string `json:"oidc_token"`
}

// AuthResponse represents the successful token exchange response
type AuthResponse struct {
	AccessToken string         `json:"access_token"`
	ExpiresIn   int            `json:"expires_in"`
	TokenType   string         `json:"token_type"`
	IssuedAt    string         `json:"issued_at"`
	Subject     SubjectDetails `json:"subject"`
}

// SubjectDetails contains the GitHub Actions context
type SubjectDetails struct {
	Provider   string `json:"provider"`
	Repository string `json:"repository"`
	Ref        string `json:"ref"`
	Workflow   string `json:"workflow"`
	RunID      string `json:"run_id"`
	Actor      string `json:"actor"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// GitHubOIDCClaims represents the claims extracted from a GitHub Actions OIDC token
type GitHubOIDCClaims struct {
	Issuer         string `json:"iss"`
	Subject        string `json:"sub"`
	Audience       string `json:"aud"`
	ExpiresAt      int64  `json:"exp"`
	NotBefore      int64  `json:"nbf"`
	IssuedAt       int64  `json:"iat"`
	Repository     string `json:"repository"`
	Ref            string `json:"ref"`
	Actor          string `json:"actor"`
	RunID          string `json:"run_id"`
	WorkflowRef    string `json:"workflow_ref"`
	JobWorkflowRef string `json:"job_workflow_ref"`
}

// RoboHubClaims represents the claims in a RoboHub access token
type RoboHubClaims struct {
	Issuer    string   `json:"iss"`
	Subject   string   `json:"sub"`
	Audience  string   `json:"aud"`
	IssuedAt  int64    `json:"iat"`
	ExpiresAt int64    `json:"exp"`
	JTI       string   `json:"jti"`
	Repo      string   `json:"repo"`
	Ref       string   `json:"ref"`
	Actor     string   `json:"actor"`
	RunID     string   `json:"run_id"`
	Scopes    []string `json:"scopes"`
}

// VerifiedClaims represents verified OIDC claims
type VerifiedClaims struct {
	Repository string
	Ref        string
	Actor      string
	RunID      string
	Workflow   string
	IssuedAt   time.Time
	ExpiresAt  time.Time
}
