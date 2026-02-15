# Quick Reference - RoboHub Auth Service

## Start the Service

```bash
# Using Docker Compose (recommended for local testing)
docker compose up --build

# Using Go directly
export ROBOHUB_JWT_SECRET="your-secret-here"
go run cmd/robohub-auth/main.go

# Using pre-built binary
export ROBOHUB_JWT_SECRET="your-secret-here"
./robohub-auth
```

## Test the Service

```bash
# Run all unit tests
go test ./...

# Run with coverage
go test ./... -cover

# Run integration tests (requires service running)
./test.sh

# Health check
curl http://localhost:8080/healthz

# Readiness check
curl http://localhost:8080/readyz
```

## API Usage

### Token Exchange
```bash
curl -X POST http://localhost:8080/auth/github-oidc \
  -H "Content-Type: application/json" \
  -d '{"oidc_token": "YOUR_GITHUB_OIDC_TOKEN"}'
```

**Success Response (200):**
```json
{
  "access_token": "eyJhbGc...",
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

**Error Responses:**
- `400` - Invalid request
- `401` - Invalid OIDC token
- `403` - Policy violation
- `429` - Rate limited
- `500` - Internal error

## Environment Variables

### Required
```bash
ROBOHUB_JWT_SECRET="strong-random-secret"
```

### Common Configuration
```bash
# OIDC Settings
ROBOHUB_OIDC_AUDIENCE="robohub"
ROBOHUB_OIDC_ISSUER="https://token.actions.githubusercontent.com"

# Policy Settings
ROBOHUB_DEFAULT_BRANCH_ONLY="false"
ROBOHUB_DEFAULT_BRANCH="main"
ROBOHUB_REPO_DENYLIST="evil/repo,bad/actor"
ROBOHUB_REPO_ALLOWLIST="trusted/org"

# Rate Limiting
ROBOHUB_RATE_LIMIT_RPS="10.0"
ROBOHUB_RATE_LIMIT_BURST="20"

# Token Settings
ROBOHUB_TOKEN_TTL_SECONDS="600"

# Server
PORT="8080"
```

## GitHub Actions Integration

### Minimal Workflow
```yaml
permissions:
  id-token: write
  contents: read

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - name: Get OIDC Token
        id: oidc
        run: |
          OIDC_TOKEN=$(curl -H "Authorization: bearer $ACTIONS_ID_TOKEN_REQUEST_TOKEN" \
            "$ACTIONS_ID_TOKEN_REQUEST_URL&audience=robohub" | jq -r '.value')
          echo "::add-mask::$OIDC_TOKEN"
          echo "token=$OIDC_TOKEN" >> $GITHUB_OUTPUT
      
      - name: Get RoboHub Token
        id: auth
        run: |
          curl -X POST https://auth.example.com/auth/github-oidc \
            -H "Content-Type: application/json" \
            -d "{\"oidc_token\": \"${{ steps.oidc.outputs.token }}\"}" \
            | jq -r '.access_token'
```

## Common Commands

```bash
# Build
make build              # Build binary
make docker-build       # Build Docker image

# Test
make test              # Run tests
make test-coverage     # Run tests with coverage

# Run
make run               # Run locally (needs ROBOHUB_JWT_SECRET)
make docker-up         # Run with docker-compose
make docker-down       # Stop docker-compose

# Clean
make clean             # Remove build artifacts

# Format
make fmt               # Format code
make tidy              # Tidy dependencies
```

## Deployment

### Kubernetes
```bash
# Apply manifests
kubectl apply -f examples/kubernetes.yml

# Update secret
kubectl create secret generic robohub-auth-secret \
  -n robohub-auth \
  --from-literal=jwt-secret="YOUR-SECRET" \
  --dry-run=client -o yaml | kubectl apply -f -
```

### Docker
```bash
# Build and tag
docker build -t robohub-auth:v1.0.0 .

# Run
docker run -p 8080:8080 \
  -e ROBOHUB_JWT_SECRET="your-secret" \
  robohub-auth:v1.0.0
```

## Troubleshooting

### "failed to verify OIDC token"
- Check token is from GitHub Actions
- Verify audience matches `ROBOHUB_OIDC_AUDIENCE`
- Ensure network access to GitHub JWKS endpoint

### "policy violation"
- Check if repo is in denylist: `ROBOHUB_REPO_DENYLIST`
- If allowlist set, ensure repo included: `ROBOHUB_REPO_ALLOWLIST`
- Verify branch if `ROBOHUB_DEFAULT_BRANCH_ONLY=true`

### "rate limit exceeded"
- Increase `ROBOHUB_RATE_LIMIT_RPS` or `ROBOHUB_RATE_LIMIT_BURST`
- Check for retry loops in workflow

### Service won't start
- Verify `ROBOHUB_JWT_SECRET` is set
- Check logs for configuration errors
- Ensure port 8080 is available

## Architecture

```
Request Flow:
1. GitHub Actions → Generate OIDC token (audience: robohub)
2. GitHub Actions → POST /auth/github-oidc with OIDC token
3. Auth Service → Verify OIDC token signature (JWKS)
4. Auth Service → Validate claims (issuer, audience, exp)
5. Auth Service → Check policy (allowlist/denylist/branch)
6. Auth Service → Check rate limit (per-repository)
7. Auth Service → Mint RoboHub JWT (HS256)
8. Auth Service → Return access token
9. GitHub Actions → Use token to call RoboHub API
```

## Security Best Practices

1. **JWT Secret**: Use strong random secret (32+ bytes)
2. **Network**: Run behind TLS (HTTPS)
3. **Secrets**: Store in secrets manager, not environment files
4. **Rotation**: Plan for secret rotation
5. **Monitoring**: Alert on 401/403/500 errors
6. **Logging**: Enable structured logs, export to SIEM
7. **Updates**: Keep dependencies updated
8. **Access**: Restrict network access to service

## Support

- **Documentation**: See README.md for full details
- **Examples**: Check examples/ directory
- **Tests**: Run `go test ./... -v` for detailed output
- **Logs**: Service logs to stdout in JSON format
