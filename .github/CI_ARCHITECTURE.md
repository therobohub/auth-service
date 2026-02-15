# CI/CD Pipeline Architecture

## Workflow Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                         TRIGGER EVENTS                          │
├─────────────────────────────────────────────────────────────────┤
│  • Push to main/develop                                         │
│  • Pull Request to main/develop                                 │
│  • Version Tag (v1.0.0, v1.2.3, etc.)                          │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      PARALLEL EXECUTION                         │
└─────────────────────────────────────────────────────────────────┘
        │                    │                    │
        ▼                    ▼                    ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│     TEST     │    │     LINT     │    │   SECURITY   │
├──────────────┤    ├──────────────┤    ├──────────────┤
│ • Go 1.22    │    │ • golangci   │    │ • gosec      │
│ • go vet     │    │ • gofmt      │    │ • SARIF      │
│ • go fmt     │    │ • goimports  │    │ • Upload     │
│ • Tests      │    │ • 10+ rules  │    │              │
│ • Race det.  │    │              │    │              │
│ • Coverage   │    │              │    │              │
│   (80% min)  │    │              │    │              │
└──────────────┘    └──────────────┘    └──────────────┘
        │                    │                    │
        └────────────┬───────┴────────────────────┘
                     │ All checks pass
                     ▼
        ┌────────────────────────────┐
        │          BUILD             │
        ├────────────────────────────┤
        │ • Linux amd64             │
        │ • Upload artifact         │
        └────────────────────────────┘
                     │
                     ▼
        ┌────────────────────────────┐
        │   INTEGRATION TEST         │
        ├────────────────────────────┤
        │ • Start service            │
        │ • Run test.sh             │
        │ • Health checks           │
        │ • API validation          │
        └────────────────────────────┘
                     │
        ┌────────────┴────────────┐
        │                         │
        ▼                         ▼
┌─────────────────┐    ┌─────────────────┐
│  DOCKER BUILD   │    │    RELEASE      │
│  (main/tags)    │    │  (tags only)    │
├─────────────────┤    ├─────────────────┤
│ • linux/amd64   │    │ • Create GH     │
│ • ghcr.io push  │    │   Release       │
│ • Tag versions  │    │ • Attach binary │
│ • Cache layers  │    │ • Checksum      │
└─────────────────┘    └─────────────────┘
```

## Job Dependencies

```
Test ────┐
         ├──→ Build ──→ Integration Test ──┐
Lint ────┤                                  ├──→ Docker (if main/tag)
         │                                  │
Security ┘                                  └──→ Release (if tag)
```

## Trigger Matrix

| Event         | Test | Lint | Security | Build | Docker | Release |
|---------------|------|------|----------|-------|--------|---------|
| Push (branch) | ✓    | ✓    | ✓        | ✓     | ✗      | ✗       |
| Push (main)   | ✓    | ✓    | ✓        | ✓     | ✓      | ✗       |
| Pull Request  | ✓    | ✓    | ✓        | ✓     | ✗      | ✗       |
| Tag (v*)      | ✓    | ✓    | ✓        | ✓     | ✓      | ✓       |

## Artifact Flow

```
┌──────────┐
│  Source  │
└────┬─────┘
     │
     ├──→ coverage.out ────→ Artifacts
     ├──→ coverage.html ───→ Artifacts
     ├──→ binaries ────────→ Artifacts ─────→ Release (if tag)
     ├──→ docker images ───→ ghcr.io
     └──→ gosec.sarif ─────→ Code Scanning
```

## Docker Image Tags

```
Push to main:
  ghcr.io/org/auth-service:main
  ghcr.io/org/auth-service:main-sha-abc123

Tag v1.2.3:
  ghcr.io/org/auth-service:v1.2.3
  ghcr.io/org/auth-service:1.2.3
  ghcr.io/org/auth-service:1.2
  ghcr.io/org/auth-service:1
  ghcr.io/org/auth-service:latest
```

## Automated Updates

```
┌─────────────────────┐
│  Weekly Schedule    │
│  (Sundays 00:00)    │
└──────────┬──────────┘
           │
           ├──→ Dependabot
           │    ├─→ GitHub Actions updates
           │    ├─→ Go module updates
           │    └─→ Docker base image updates
           │
           └──→ Dependencies Workflow
                └─→ go get -u ./...
                └─→ Create PR
```

## Success Criteria

### Pull Request Merge Requirements
- ✓ All tests pass
- ✓ Linter passes
- ✓ Security scan passes
- ✓ Coverage ≥ 80%
- ✓ Code review approved
- ✓ No merge conflicts

### Release Requirements
- ✓ All PR requirements
- ✓ Integration tests pass
- ✓ Builds successful for all platforms
- ✓ Docker images built and pushed
- ✓ Tagged with semantic version

## Notifications

```
┌─────────────┐
│   GitHub    │
│   Actions   │
└──────┬──────┘
       │
       ├──→ Status Checks (PR)
       ├──→ Email (failures)
       ├──→ Actions Tab (logs)
       └──→ Badges (README)
```

## Caching Strategy

```
┌─────────────────┐
│   Go Modules    │──→ Cache Key: go.sum
└─────────────────┘

┌─────────────────┐
│  Docker Layers  │──→ Cache: GitHub Actions Cache
└─────────────────┘

┌─────────────────┐
│   Go Build      │──→ Cache Key: OS + Go version
└─────────────────┘
```

## Security Scanning

```
┌──────────────┐
│    Source    │
└──────┬───────┘
       │
       ├──→ Gosec ───→ SARIF ───→ Code Scanning
       │
       ├──→ Dependabot ─────────→ Vulnerability Alerts
       │
       └──→ go mod verify ──────→ Supply Chain
```

## Release Process

```
Developer                    GitHub Actions                 Users
    │                              │                          │
    │─── git tag v1.0.0            │                          │
    │─── git push --tags ─────────→│                          │
    │                              │                          │
    │                              │ Run Tests                │
    │                              │ Run Linter               │
    │                              │ Security Scan            │
    │                              │ Build Binaries           │
    │                              │ Build Docker             │
    │                              │                          │
    │                              │ Create Release           │
    │                              │ Attach Binaries          │
    │                              │ Generate Notes           │
    │                              │                          │
    │                              │ Push Docker Images ──────→│
    │                              │                          │
    │←──── Release Created ────────│                          │
    │                              │                          │
    │                              │                          │
    │                              │←── docker pull ──────────│
    │                              │                          │
```

## Monitoring

```
┌────────────────────────────────────────┐
│        GitHub Actions Dashboard         │
├────────────────────────────────────────┤
│                                        │
│  Workflow Runs                         │
│  ├─ Success Rate                       │
│  ├─ Average Duration                   │
│  └─ Failed Jobs                        │
│                                        │
│  Code Scanning                         │
│  ├─ Security Alerts                    │
│  ├─ Code Quality                       │
│  └─ Vulnerabilities                    │
│                                        │
│  Dependabot                            │
│  ├─ Open PRs                           │
│  ├─ Security Updates                   │
│  └─ Compatibility                      │
│                                        │
└────────────────────────────────────────┘
```
