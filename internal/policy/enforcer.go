package policy

import (
	"fmt"
	"strings"
)

// Enforcer enforces repository and branch policies
type Enforcer struct {
	defaultBranchOnly bool
	defaultBranch     string
	allowList         map[string]bool
	denyList          map[string]bool
}

// NewEnforcer creates a new policy enforcer
func NewEnforcer(defaultBranchOnly bool, defaultBranch string, allowList, denyList []string) *Enforcer {
	e := &Enforcer{
		defaultBranchOnly: defaultBranchOnly,
		defaultBranch:     defaultBranch,
		allowList:         make(map[string]bool),
		denyList:          make(map[string]bool),
	}

	for _, repo := range allowList {
		e.allowList[repo] = true
	}

	for _, repo := range denyList {
		e.denyList[repo] = true
	}

	return e
}

// Evaluate checks if the repository and ref are allowed by policy
func (e *Enforcer) Evaluate(repository, ref string) error {
	// Check denylist first
	if e.denyList[repository] {
		return fmt.Errorf("repository %s is denied by policy", repository)
	}

	// Check allowlist if configured
	if len(e.allowList) > 0 && !e.allowList[repository] {
		return fmt.Errorf("repository %s is not in allowlist", repository)
	}

	// Check default branch requirement
	if e.defaultBranchOnly {
		expectedRef := "refs/heads/" + e.defaultBranch
		if ref != expectedRef {
			return fmt.Errorf("only default branch %s is allowed, got %s", expectedRef, ref)
		}
	}

	return nil
}

// IsDefaultBranch checks if the given ref is the default branch
func (e *Enforcer) IsDefaultBranch(ref string) bool {
	expectedRef := "refs/heads/" + e.defaultBranch
	return ref == expectedRef
}

// ExtractBranch extracts the branch name from a ref
func ExtractBranch(ref string) string {
	if strings.HasPrefix(ref, "refs/heads/") {
		return strings.TrimPrefix(ref, "refs/heads/")
	}
	return ref
}
