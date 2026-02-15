package policy

import (
	"testing"
)

func TestEnforcer_Evaluate(t *testing.T) {
	tests := []struct {
		name              string
		defaultBranchOnly bool
		defaultBranch     string
		allowList         []string
		denyList          []string
		repository        string
		ref               string
		wantError         bool
		errorContains     string
	}{
		{
			name:          "allowed repo and ref",
			repository:    "owner/repo",
			ref:           "refs/heads/main",
			wantError:     false,
		},
		{
			name:          "denied repo",
			denyList:      []string{"evil/repo"},
			repository:    "evil/repo",
			ref:           "refs/heads/main",
			wantError:     true,
			errorContains: "denied by policy",
		},
		{
			name:          "not in allowlist",
			allowList:     []string{"good/repo"},
			repository:    "other/repo",
			ref:           "refs/heads/main",
			wantError:     true,
			errorContains: "not in allowlist",
		},
		{
			name:          "in allowlist",
			allowList:     []string{"good/repo"},
			repository:    "good/repo",
			ref:           "refs/heads/main",
			wantError:     false,
		},
		{
			name:              "default branch only - valid",
			defaultBranchOnly: true,
			defaultBranch:     "main",
			repository:        "owner/repo",
			ref:               "refs/heads/main",
			wantError:         false,
		},
		{
			name:              "default branch only - invalid",
			defaultBranchOnly: true,
			defaultBranch:     "main",
			repository:        "owner/repo",
			ref:               "refs/heads/develop",
			wantError:         true,
			errorContains:     "only default branch",
		},
		{
			name:              "custom default branch",
			defaultBranchOnly: true,
			defaultBranch:     "develop",
			repository:        "owner/repo",
			ref:               "refs/heads/develop",
			wantError:         false,
		},
		{
			name:              "custom default branch - invalid",
			defaultBranchOnly: true,
			defaultBranch:     "develop",
			repository:        "owner/repo",
			ref:               "refs/heads/main",
			wantError:         true,
			errorContains:     "only default branch",
		},
		{
			name:          "denylist takes precedence over allowlist",
			allowList:     []string{"conflicted/repo"},
			denyList:      []string{"conflicted/repo"},
			repository:    "conflicted/repo",
			ref:           "refs/heads/main",
			wantError:     true,
			errorContains: "denied by policy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEnforcer(tt.defaultBranchOnly, tt.defaultBranch, tt.allowList, tt.denyList)
			err := e.Evaluate(tt.repository, tt.ref)

			if (err != nil) != tt.wantError {
				t.Errorf("expected error=%v, got error=%v", tt.wantError, err)
			}

			if tt.wantError && tt.errorContains != "" {
				if err == nil || !contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error to contain %q, got %v", tt.errorContains, err)
				}
			}
		})
	}
}

func TestEnforcer_IsDefaultBranch(t *testing.T) {
	tests := []struct {
		name          string
		defaultBranch string
		ref           string
		want          bool
	}{
		{"main is default", "main", "refs/heads/main", true},
		{"develop is not default", "main", "refs/heads/develop", false},
		{"custom default branch", "develop", "refs/heads/develop", true},
		{"tag ref", "main", "refs/tags/v1.0.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEnforcer(false, tt.defaultBranch, nil, nil)
			if got := e.IsDefaultBranch(tt.ref); got != tt.want {
				t.Errorf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestExtractBranch(t *testing.T) {
	tests := []struct {
		name string
		ref  string
		want string
	}{
		{"branch ref", "refs/heads/main", "main"},
		{"branch ref develop", "refs/heads/develop", "develop"},
		{"tag ref", "refs/tags/v1.0.0", "refs/tags/v1.0.0"},
		{"plain string", "main", "main"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractBranch(tt.ref); got != tt.want {
				t.Errorf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > 0 && len(substr) > 0 && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
