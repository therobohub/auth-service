# GitHub Actions Setup Guide

This guide helps you configure GitHub Actions for the RoboHub Auth Service repository.

## Initial Setup

### 1. Repository Settings

Navigate to **Settings → Actions → General**:

#### Workflow Permissions
- ✅ **Read and write permissions**
- ✅ Allow GitHub Actions to create and approve pull requests

#### Actions Permissions
- ✅ Allow all actions and reusable workflows

#### Artifact and log retention
- Recommended: 90 days

### 2. Secrets Configuration

No secrets are required for the basic CI/CD pipeline! The workflow uses `GITHUB_TOKEN` which is automatically provided.

Optional: If you want to customize notifications or integrate with external services, add secrets at **Settings → Secrets and variables → Actions**.

### 3. Branch Protection Rules

Navigate to **Settings → Branches → Add rule**:

For branch: `main`
- ✅ Require a pull request before merging
  - ✅ Require approvals (1+)
- ✅ Require status checks to pass before merging
  - ✅ Require branches to be up to date before merging
  - Select these status checks:
    - Test
    - Lint
    - Security
    - Build
    - Integration Test
- ✅ Require conversation resolution before merging
- ✅ Do not allow bypassing the above settings

### 4. Enable Dependabot

Navigate to **Settings → Code security and analysis**:

- ✅ Enable Dependabot alerts
- ✅ Enable Dependabot security updates
- ✅ Enable Dependabot version updates

### 5. Code Scanning

Navigate to **Settings → Code security and analysis**:

- ✅ Enable code scanning
- ✅ Set up CodeQL analysis (optional, we use Gosec)

## Docker Image Publishing

### GitHub Container Registry (ghcr.io)

Docker images are automatically published to GitHub Container Registry. No additional configuration needed!

**After first publish:**
1. Go to **Packages** section in your repository
2. Click on the `auth-service` package
3. Click **Package settings**
4. Set visibility (Public/Private)
5. Optional: Link to repository

**Pull images:**
```bash
# Latest from main branch
docker pull ghcr.io/YOUR_ORG/auth-service:main

# Specific version
docker pull ghcr.io/YOUR_ORG/auth-service:v1.0.0
```

## First Commit

After pushing the workflows, the CI will run automatically:

```bash
git add .
git commit -m "ci: add GitHub Actions workflows"
git push origin main
```

**Check progress:**
1. Go to **Actions** tab
2. Click on the running workflow
3. Monitor job progress

## Creating Your First Release

```bash
# Create and push a version tag
git tag -a v1.0.0 -m "Initial release"
git push origin v1.0.0

# Wait for CI to complete (~5-10 minutes)
# Check Actions tab for progress
```

**What happens:**
1. All tests run
2. Binary built for Linux amd64
3. Docker image built and pushed
4. GitHub Release created with:
   - Binary for Linux amd64
   - SHA256 checksum
   - Auto-generated release notes

## Monitoring

### Actions Dashboard

**View workflow runs:**
- Navigate to **Actions** tab
- Click on specific workflow to see runs
- Click on run to see job details

**Common issues:**
- ❌ Test failures → Check test logs
- ❌ Lint failures → Run `make lint` locally
- ❌ Coverage too low → Add more tests
- ❌ Docker build fails → Check Dockerfile

### Status Badges

Update these in your README.md:

```markdown
[![CI/CD](https://github.com/YOUR_ORG/auth-service/actions/workflows/ci.yml/badge.svg)](https://github.com/YOUR_ORG/auth-service/actions/workflows/ci.yml)
```

Replace `YOUR_ORG` with your GitHub organization/username.

## Troubleshooting

### Workflow Not Running

1. **Check Actions are enabled:**
   - Settings → Actions → General → Actions permissions

2. **Check branch protection:**
   - Settings → Branches → View rules

3. **Check workflow file syntax:**
   ```bash
   # Validate YAML locally
   yamllint .github/workflows/ci.yml
   ```

### Docker Push Fails

1. **Check permissions:**
   - Settings → Actions → General → Workflow permissions
   - Must be "Read and write"

2. **Check package permissions:**
   - Go to Package settings
   - Ensure Actions has write access

### Coverage Check Fails

Current minimum coverage: 80%

**Fix:**
```bash
# Check current coverage
go test ./... -cover

# Generate detailed report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
open coverage.html

# Add tests to increase coverage
```

### Dependabot PRs

**Review process:**
1. Check the PR description
2. Review changelog for breaking changes
3. Verify CI passes
4. Merge if tests pass

**If tests fail:**
1. Check if API changes in dependency
2. Update code to match new API
3. Push fixes to Dependabot's branch

## Advanced Configuration

### Custom Linter Rules

Edit `.golangci.yml`:
```yaml
linters:
  enable:
    - newlinter
  disable:
    - oldlinter
```

### Skip CI for Commits

Add to commit message:
```bash
git commit -m "docs: update readme [skip ci]"
```

### Manual Workflow Trigger

Some workflows support manual trigger:
1. Go to **Actions** tab
2. Select workflow
3. Click "Run workflow"
4. Select branch
5. Click "Run workflow" button

### Custom Docker Tags

Edit `.github/workflows/ci.yml` in the `docker` job:

```yaml
tags: |
  type=ref,event=branch
  type=semver,pattern={{version}}
  type=sha
```

## Best Practices

### Pull Requests
- ✅ Wait for CI before requesting review
- ✅ Fix all linter issues
- ✅ Ensure coverage doesn't decrease
- ✅ Update tests for new code

### Releases
- ✅ Use semantic versioning (v1.2.3)
- ✅ Update CHANGELOG before tagging
- ✅ Test locally before creating tag
- ✅ Write meaningful release notes

### Maintenance
- ✅ Review Dependabot PRs weekly
- ✅ Keep workflows up to date
- ✅ Monitor failed workflow runs
- ✅ Update branch protection rules as needed

## Getting Help

### GitHub Actions Documentation
- [GitHub Actions Docs](https://docs.github.com/en/actions)
- [Workflow Syntax](https://docs.github.com/en/actions/reference/workflow-syntax-for-github-actions)
- [GitHub Packages](https://docs.github.com/en/packages)

### Project Documentation
- [CONTRIBUTING.md](../CONTRIBUTING.md) - Contribution guidelines
- [CI_ARCHITECTURE.md](CI_ARCHITECTURE.md) - CI/CD architecture
- [README.md](../README.md) - Project documentation

### Support
- Open an issue with label `ci/cd`
- Check workflow run logs for errors
- Review similar successful runs

## Checklist

Before going live, ensure:

- [ ] Repository settings configured
- [ ] Branch protection rules enabled
- [ ] First commit pushed successfully
- [ ] CI/CD pipeline ran successfully
- [ ] Docker images published
- [ ] Status badges updated in README
- [ ] Team members have appropriate permissions
- [ ] Dependabot enabled
- [ ] Code scanning enabled
- [ ] First release created and tested

## Success!

Your CI/CD pipeline is now set up! Every push will trigger automated testing, linting, and security scanning. Releases will be created automatically with binaries and Docker images.

🚀 Happy coding!
