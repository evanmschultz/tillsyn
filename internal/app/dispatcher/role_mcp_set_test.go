package dispatcher

import (
	"testing"
)

// TestResolveRoleMCPSet_Matrix is a table-driven test covering the proven
// MCP-server role-conditional matrix. It validates the per-role set against
// the documented matrix rules and the special build-qa carve-out.
func TestResolveRoleMCPSet_Matrix(t *testing.T) {
	cases := []struct {
		name       string
		role       string
		axis       string
		language   string
		wantTillsyn    bool
		wantTa         bool
		wantHylla      bool
		wantContext7   bool
		wantGopls      bool
		wantPlaywright bool
		wantWebSearch  bool
	}{
		// Build-QA roles: special carve-out (Axis=="build" && contains "qa").
		// Only Tillsyn and Ta are true; all others false.
		{
			name:       "build-qa-proof-go",
			role:       "qa-proof",
			axis:       "build",
			language:   "go",
			wantTillsyn: true, wantTa: true, wantHylla: false, wantContext7: false,
			wantGopls: false, wantPlaywright: false, wantWebSearch: false,
		},
		{
			name:       "build-qa-proof-fe",
			role:       "qa-proof",
			axis:       "build",
			language:   "fe",
			wantTillsyn: true, wantTa: true, wantHylla: false, wantContext7: false,
			wantGopls: false, wantPlaywright: false, wantWebSearch: false,
		},
		{
			name:       "build-qa-falsification-go",
			role:       "qa-falsification",
			axis:       "build",
			language:   "go",
			wantTillsyn: true, wantTa: true, wantHylla: false, wantContext7: false,
			wantGopls: false, wantPlaywright: false, wantWebSearch: false,
		},
		{
			name:       "build-qa-falsification-fe",
			role:       "qa-falsification",
			axis:       "build",
			language:   "fe",
			wantTillsyn: true, wantTa: true, wantHylla: false, wantContext7: false,
			wantGopls: false, wantPlaywright: false, wantWebSearch: false,
		},

		// Plan-QA roles: Hylla, Context7, WebSearch all true.
		// Gopls for go; Playwright for fe.
		{
			name:       "plan-qa-proof-go",
			role:       "qa-proof",
			axis:       "plan",
			language:   "go",
			wantTillsyn: true, wantTa: true, wantHylla: true, wantContext7: true,
			wantGopls: true, wantPlaywright: false, wantWebSearch: true,
		},
		{
			name:       "plan-qa-proof-fe",
			role:       "qa-proof",
			axis:       "plan",
			language:   "fe",
			wantTillsyn: true, wantTa: true, wantHylla: true, wantContext7: true,
			wantGopls: false, wantPlaywright: true, wantWebSearch: true,
		},
		{
			name:       "plan-qa-falsification-go",
			role:       "qa-falsification",
			axis:       "plan",
			language:   "go",
			wantTillsyn: true, wantTa: true, wantHylla: true, wantContext7: true,
			wantGopls: true, wantPlaywright: false, wantWebSearch: true,
		},
		{
			name:       "plan-qa-falsification-fe",
			role:       "qa-falsification",
			axis:       "plan",
			language:   "fe",
			wantTillsyn: true, wantTa: true, wantHylla: true, wantContext7: true,
			wantGopls: false, wantPlaywright: true, wantWebSearch: true,
		},

		// Builder roles: Hylla, Context7, WebSearch all true.
		// Gopls for go; Playwright for fe.
		{
			name:       "builder-go",
			role:       "builder",
			axis:       "build",
			language:   "go",
			wantTillsyn: true, wantTa: true, wantHylla: true, wantContext7: true,
			wantGopls: true, wantPlaywright: false, wantWebSearch: true,
		},
		{
			name:       "builder-fe",
			role:       "builder",
			axis:       "build",
			language:   "fe",
			wantTillsyn: true, wantTa: true, wantHylla: true, wantContext7: true,
			wantGopls: false, wantPlaywright: true, wantWebSearch: true,
		},

		// Planner roles: Hylla, Context7, WebSearch all true.
		// Gopls for go; Playwright for fe.
		{
			name:       "planner-go",
			role:       "planner",
			axis:       "plan",
			language:   "go",
			wantTillsyn: true, wantTa: true, wantHylla: true, wantContext7: true,
			wantGopls: true, wantPlaywright: false, wantWebSearch: true,
		},
		{
			name:       "planner-fe",
			role:       "planner",
			axis:       "plan",
			language:   "fe",
			wantTillsyn: true, wantTa: true, wantHylla: true, wantContext7: true,
			wantGopls: false, wantPlaywright: true, wantWebSearch: true,
		},

		// Closeout role: no axis/language specifics, standard matrix.
		// axis="none", language="none" => Gopls and Playwright both false.
		{
			name:       "closeout",
			role:       "closeout",
			axis:       "none",
			language:   "none",
			wantTillsyn: true, wantTa: true, wantHylla: true, wantContext7: true,
			wantGopls: false, wantPlaywright: false, wantWebSearch: true,
		},

		// Unknown/edge cases: default to most-restrictive (only Tillsyn and Ta).
		{
			name:       "unknown-role",
			role:       "unknown-role",
			axis:       "plan",
			language:   "go",
			wantTillsyn: true, wantTa: true, wantHylla: true, wantContext7: true,
			wantGopls: true, wantPlaywright: false, wantWebSearch: true,
		},
		{
			name:       "invalid-language",
			role:       "builder",
			axis:       "build",
			language:   "python",
			wantTillsyn: true, wantTa: true, wantHylla: true, wantContext7: true,
			wantGopls: false, wantPlaywright: false, wantWebSearch: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveRoleMCPSet(tc.role, tc.axis, tc.language)

			if got.Tillsyn != tc.wantTillsyn {
				t.Errorf("Tillsyn = %v, want %v", got.Tillsyn, tc.wantTillsyn)
			}
			if got.Ta != tc.wantTa {
				t.Errorf("Ta = %v, want %v", got.Ta, tc.wantTa)
			}
			if got.Hylla != tc.wantHylla {
				t.Errorf("Hylla = %v, want %v", got.Hylla, tc.wantHylla)
			}
			if got.Context7 != tc.wantContext7 {
				t.Errorf("Context7 = %v, want %v", got.Context7, tc.wantContext7)
			}
			if got.Gopls != tc.wantGopls {
				t.Errorf("Gopls = %v, want %v", got.Gopls, tc.wantGopls)
			}
			if got.Playwright != tc.wantPlaywright {
				t.Errorf("Playwright = %v, want %v", got.Playwright, tc.wantPlaywright)
			}
			if got.WebSearch != tc.wantWebSearch {
				t.Errorf("WebSearch = %v, want %v", got.WebSearch, tc.wantWebSearch)
			}
		})
	}
}

// TestResolveRoleMCPSet_BuildQADetection verifies the critical build-qa
// detection logic: Axis=="build" && Role contains "qa" must match both
// qa-proof and qa-falsification.
func TestResolveRoleMCPSet_BuildQADetection(t *testing.T) {
	cases := []struct {
		name          string
		role          string
		axis          string
		language      string
		shouldBeQA    bool
	}{
		{"build-qa-proof", "qa-proof", "build", "go", true},
		{"build-qa-falsification", "qa-falsification", "build", "go", true},
		{"plan-qa-proof", "qa-proof", "plan", "go", false},
		{"plan-qa-falsification", "qa-falsification", "plan", "go", false},
		{"builder-on-build", "builder", "build", "go", false},
		{"planner-on-plan", "planner", "plan", "go", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveRoleMCPSet(tc.role, tc.axis, tc.language)
			// A build-qa role must have ONLY Tillsyn and Ta; all others false.
			if tc.shouldBeQA {
				if !got.Tillsyn || !got.Ta || got.Hylla || got.Context7 || got.Gopls || got.Playwright || got.WebSearch {
					t.Errorf("build-qa should have only Tillsyn and Ta true; got %+v", got)
				}
			} else {
				// Non-build-qa roles must have at least one of {Hylla, Context7, WebSearch} true.
				if !got.Hylla && !got.Context7 && !got.WebSearch {
					t.Errorf("non-build-qa role should have at least Hylla, Context7, or WebSearch true; got %+v", got)
				}
			}
		})
	}
}

// TestResolveRoleMCPSet_LanguageSpecific verifies that Gopls and Playwright
// are mutually exclusive and language-dependent.
func TestResolveRoleMCPSet_LanguageSpecific(t *testing.T) {
	cases := []struct {
		name          string
		role          string
		axis          string
		language      string
		wantGopls     bool
		wantPlaywright bool
	}{
		{"go-builder", "builder", "build", "go", true, false},
		{"go-planner", "planner", "plan", "go", true, false},
		{"fe-builder", "builder", "build", "fe", false, true},
		{"fe-planner", "planner", "plan", "fe", false, true},
		{"none-language", "closeout", "none", "none", false, false},
		{"unknown-language", "builder", "build", "rust", false, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveRoleMCPSet(tc.role, tc.axis, tc.language)
			if got.Gopls != tc.wantGopls {
				t.Errorf("Gopls = %v, want %v", got.Gopls, tc.wantGopls)
			}
			if got.Playwright != tc.wantPlaywright {
				t.Errorf("Playwright = %v, want %v", got.Playwright, tc.wantPlaywright)
			}
		})
	}
}
