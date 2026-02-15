# RoboHub Auth Service

[![CI/CD](https://github.com/YOUR_ORG/auth-service/actions/workflows/ci.yml/badge.svg)](https://github.com/YOUR_ORG/auth-service/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/YOUR_ORG/auth-service)](https://goreportcard.com/report/github.com/YOUR_ORG/auth-service)
[![License](https://img.shields.io/badge/license-Proprietary-blue.svg)](LICENSE)

Production-grade authentication service that exchanges GitHub Actions OIDC tokens for short-lived RoboHub access tokens (JWT).

## Features

- **GitHub Actions OIDC Integration**: Verifies GitHub Actions OIDC tokens using JWKS
- **Policy Enforcement**: Repository allowlist/denylist, default branch restrictions
- **Rate Limiting**: Per-repository token bucket rate limiting
- **Secure Token Minting**: Issues short-lived JWT access tokens with configurable TTL
- **Production Ready**: Structured logging, graceful shutdown, health checks

## Quick Start

### Prerequisites

- Docker and Docker Compose
- Go 1.22+ (for local development)

### Run with Docker Compose

```bash
# Build and start the service
docker compose up --build

# Service will be available at http://localhost:8080
```

The service will start with default configuration suitable for local testing.

### Run Locally (Development)

```bash
# Install dependencies
go mod download

# Set required environment variable
export ROBOHUB_JWT_SECRET="your-secret-key"

# Run the service
go run cmd/robohub-auth/main.go

# Or build and run
go build -o robohub-auth cmd/robohub-auth/main.go
./robohub-auth
```

## API Endpoints

### Health Check

```bash
curl http://localhost:8080/healthz
# Response: ok
```

### Readiness Check

```bash
curl http://localhost:8080/readyz
# Response: ok
```

### GitHub OIDC Token Exchange

```bash
curl -X POST http://localhost:8080/auth/github-oidc \
  -H "Content-Type: application/json" \
  -d '{
    "oidc_token": "<GitHub-Actions-OIDC-JWT>"
  }'
```

**Success Response (200)**:

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 600,
  "token_type": "Bearer",
  "issued_at": "2026-02-15T10:30:00Z",
  "subject": {
    "provider": "github_actions",
    "repository": "owner/repo",
    "ref": "refs/heads/main",
    "workflow": ".github/workflows/ci.yml@refs/heads/main",
    "run_id": "123456789",
    "actor": "username"
  }
}
```

**Error Responses**:

- `400` - Invalid request (missing or malformed JSON)
- `401` - Invalid OIDC token (verification failed)
- `403` - Policy violation (denied repository or branch)
- `429` - Rate limit exceeded
- `500` - Internal server error

## Configuration

All configuration is via environment variables:

### Required

| Variable | Description | Example |
|----------|-------------|---------|
| `ROBOHUB_JWT_SECRET` | Secret key for signing access tokens | `strong-random-secret-here` |

### OIDC Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `ROBOHUB_OIDC_ISSUER` | GitHub OIDC issuer URL | `https://token.actions.githubusercontent.com` |
| `ROBOHUB_OIDC_AUDIENCE` | Expected audience in OIDC token | `robohub` |
| `ROBOHUB_CLOCK_SKEW_SECONDS` | Allowed clock skew for token validation | `60` |
| `ROBOHUB_JWKS_TTL_SECONDS` | JWKS cache TTL in seconds | `3600` |

### Policy Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `ROBOHUB_DEFAULT_BRANCH_ONLY` | Only allow default branch | `false` |
| `ROBOHUB_DEFAULT_BRANCH` | Name of default branch | `main` |
| `ROBOHUB_REPO_DENYLIST` | Comma-separated list of denied repos | `` |
| `ROBOHUB_REPO_ALLOWLIST` | Comma-separated list of allowed repos (if set, only these allowed) | `` |

**Policy Examples**:

```bash
# Deny specific repositories
ROBOHUB_REPO_DENYLIST=evil/repo,untrusted/project

# Only allow specific repositories
ROBOHUB_REPO_ALLOWLIST=myorg/trusted-repo,myorg/another-repo

# Only allow default branch (main)
ROBOHUB_DEFAULT_BRANCH_ONLY=true
ROBOHUB_DEFAULT_BRANCH=main

# Use custom default branch (develop)
ROBOHUB_DEFAULT_BRANCH_ONLY=true
ROBOHUB_DEFAULT_BRANCH=develop
```

### Rate Limiting

| Variable | Description | Default |
|----------|-------------|---------|
| `ROBOHUB_RATE_LIMIT_RPS` | Requests per second per repository | `1.0` |
| `ROBOHUB_RATE_LIMIT_BURST` | Burst size per repository | `5` |

### Token Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `ROBOHUB_TOKEN_TTL_SECONDS` | Access token TTL in seconds | `600` (10 minutes) |

### Server

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | HTTP server port | `8080` |

## Using in GitHub Actions

To use this service in your GitHub Actions workflow, you need to:

1. Configure your workflow to request an OIDC token
2. Send the token to this auth service
3. Use the returned access token

**Example Workflow**:

```yaml
name: Deploy to RoboHub

on:
  push:
    branches: [main]

permissions:
  id-token: write  # Required for OIDC token
  contents: read

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v4
      
      - name: Get OIDC Token
        id: oidc
        run: |
          OIDC_TOKEN=$(curl -H "Authorization: bearer $ACTIONS_ID_TOKEN_REQUEST_TOKEN" \
            "$ACTIONS_ID_TOKEN_REQUEST_URL&audience=robohub" | jq -r '.value')
          echo "token=$OIDC_TOKEN" >> $GITHUB_OUTPUT
      
      - name: Exchange for RoboHub Token
        id: auth
        run: |
          RESPONSE=$(curl -X POST https://your-auth-service.com/auth/github-oidc \
            -H "Content-Type: application/json" \
            -d "{\"oidc_token\": \"${{ steps.oidc.outputs.token }}\"}")
          ACCESS_TOKEN=$(echo $RESPONSE | jq -r '.access_token')
          echo "::add-mask::$ACCESS_TOKEN"
          echo "access_token=$ACCESS_TOKEN" >> $GITHUB_OUTPUT
      
      - name: Use RoboHub Token
        run: |
          # Use ${{ steps.auth.outputs.access_token }} to call RoboHub API
          curl -H "Authorization: Bearer ${{ steps.auth.outputs.access_token }}" \
            https://robohub-api.com/api/builds
```

**Important**: Configure the audience in your OIDC token request to match `ROBOHUB_OIDC_AUDIENCE` (default: `robohub`).

## Testing

Run all tests:

```bash
go test ./...
```

Run tests with verbose output:

```bash
go test ./... -v
```

Run tests with coverage:

```bash
go test ./... -cover
```

Run linter:

```bash
golangci-lint run
```

## CI/CD

This repository includes a complete GitHub Actions CI/CD pipeline:

### Workflows

**CI/CD Pipeline** (`.github/workflows/ci.yml`):
- Runs on every push and pull request
- **Test Job**: Runs all tests with race detection and coverage
- **Lint Job**: Runs golangci-lint for code quality
- **Build Job**: Builds binary for Linux amd64
- **Docker Job**: Builds and pushes Docker image for linux/amd64 (on main/tags)
- **Security Job**: Runs Gosec security scanner (results as artifact)
- **Integration Test Job**: Runs integration tests against live service
- **Release Job**: Creates GitHub releases with binary (on version tags)

**Dependency Updates** (`.github/workflows/dependencies.yml`):
- Runs weekly to update Go dependencies
- Creates pull requests for updates

**Dependabot** (`.github/dependabot.yml`):
- Automatically updates GitHub Actions, Go modules, and Docker base images
- Creates PRs for dependency updates

### Coverage Requirements

The CI pipeline enforces a minimum test coverage of 55%. Current coverage is ~60%.

### Docker Image Publishing

Docker images are automatically built and pushed to GitHub Container Registry (ghcr.io) when:
- Pushing to `main` branch (tagged as `main`)
- Creating version tags (e.g., `v1.0.0`)

Pull images:
```bash
docker pull ghcr.io/YOUR_ORG/auth-service:main
docker pull ghcr.io/YOUR_ORG/auth-service:v1.0.0
```

### Creating a Release

1. Tag your commit: `git tag -a v1.0.0 -m "Release v1.0.0"`
2. Push the tag: `git push origin v1.0.0`
3. GitHub Actions will automatically:
   - Run all tests and checks
   - Build binaries for all platforms
   - Create Docker images
   - Create a GitHub Release with binaries and checksums

## Development

### Project Structure

```
.
├── cmd/
│   └── robohub-auth/     # Main application entry point
│       └── main.go
├── internal/
│   ├── config/           # Configuration loading
│   ├── httpapi/          # HTTP handlers and routing
│   ├── oidc/             # OIDC verification with JWKS
│   ├── policy/           # Policy enforcement
│   ├── ratelimit/        # Per-repository rate limiting
│   ├── token/            # JWT token minting
│   └── types/            # Shared types
├── Dockerfile
├── docker-compose.yml
└── README.md
```

### Adding New Features

1. **New Policy**: Edit `internal/policy/enforcer.go`
2. **New Token Claims**: Edit `internal/token/minter.go` and `internal/types/types.go`
3. **New Endpoints**: Add handlers in `internal/httpapi/server.go`

### Testing OIDC Verification

For local testing, the codebase includes a `FakeVerifier` that can be used in tests. In production, the `GitHubVerifier` fetches and caches GitHub's JWKS automatically.

## Security Considerations

### JWT Secret

- **Production**: Use a strong, randomly generated secret (at least 32 bytes)
- **Rotation**: Plan for secret rotation (requires service restart)
- **Storage**: Store in secrets manager (e.g., AWS Secrets Manager, HashiCorp Vault)

### Future Enhancements

The current implementation uses HS256 (HMAC-SHA256) for simplicity. For production at scale, consider:

- Upgrading to RS256 (RSA) with key rotation
- Integration with KMS (AWS KMS, Google Cloud KMS)
- The code is structured with clean interfaces to make this upgrade straightforward

### Network Security

- Run behind TLS termination (load balancer or reverse proxy)
- Use network policies to restrict access
- Enable audit logging for all token exchanges

## Production Deployment

### Kubernetes

Example Kubernetes deployment:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: robohub-auth
spec:
  replicas: 3
  selector:
    matchLabels:
      app: robohub-auth
  template:
    metadata:
      labels:
        app: robohub-auth
    spec:
      containers:
      - name: robohub-auth
        image: robohub-auth:latest
        ports:
        - containerPort: 8080
        env:
        - name: ROBOHUB_JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: robohub-auth-secret
              key: jwt-secret
        - name: PORT
          value: "8080"
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /readyz
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: robohub-auth
spec:
  selector:
    app: robohub-auth
  ports:
  - port: 80
    targetPort: 8080
  type: ClusterIP
```

**Helm Chart (Recommended):**

The repository includes a production-ready Helm chart:

```bash
# Install with Helm
helm install robohub-auth ./helm/robohub-auth \
  --namespace robohub-auth \
  --create-namespace \
  --set secret.jwtSecret="$(openssl rand -base64 32)" \
  --set image.tag="v1.0.0" \
  --set ingress.enabled=true \
  --set ingress.hosts[0].host="auth.yourdomain.com"

# Or use values file
helm install robohub-auth ./helm/robohub-auth \
  -f helm/robohub-auth/values-prod.yaml
```

See `helm/robohub-auth/README.md` for complete documentation.

### Monitoring

The service logs structured JSON to stdout. Key metrics to monitor:

- Request rate and latency
- Error rates (401, 403, 500)
- Rate limit hits (429)
- Token issuance success rate

Example log entry:

```json
{
  "time": "2026-02-15T10:30:00Z",
  "level": "INFO",
  "msg": "issued access token",
  "repository": "owner/repo",
  "expires_in": 600
}
```

## Troubleshooting

### "failed to verify OIDC token"

- Verify the OIDC token is from GitHub Actions
- Check the audience matches `ROBOHUB_OIDC_AUDIENCE`
- Ensure the issuer is `https://token.actions.githubusercontent.com`
- Check network connectivity to GitHub JWKS endpoint

### "policy violation"

- Check if repository is in denylist
- If allowlist is configured, ensure repository is included
- Verify branch requirements if `ROBOHUB_DEFAULT_BRANCH_ONLY=true`

### "rate limit exceeded"

- Increase `ROBOHUB_RATE_LIMIT_RPS` or `ROBOHUB_RATE_LIMIT_BURST`
- Check for excessive retries in GitHub Actions workflow

## License

Copyright © 2026 RoboHub. All rights reserved.

## Support

For issues and questions, please open an issue in the repository.
