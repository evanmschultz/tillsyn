package domain

import (
	"errors"
	"testing"
)

// TestIsValidRole exercises the closed 9-value Role enum membership check,
// including the explicit rejection of the empty string at this validator
// level (callers that want to permit an unset role short-circuit on
// emptiness themselves).
func TestIsValidRole(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		role Role
		want bool
	}{
		{name: "builder", role: RoleBuilder, want: true},
		{name: "qa-proof", role: RoleQAProof, want: true},
		{name: "qa-falsification", role: RoleQAFalsification, want: true},
		{name: "qa-a11y", role: RoleQAA11y, want: true},
		{name: "qa-visual", role: RoleQAVisual, want: true},
		{name: "design", role: RoleDesign, want: true},
		{name: "commit", role: RoleCommit, want: true},
		{name: "planner", role: RolePlanner, want: true},
		{name: "research", role: RoleResearch, want: true},
		{name: "empty string is invalid", role: Role(""), want: false},
		{name: "unknown value", role: Role("foobar"), want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := IsValidRole(tc.role); got != tc.want {
				t.Fatalf("IsValidRole(%q) = %v, want %v", tc.role, got, tc.want)
			}
		})
	}
}

// TestNormalizeRole verifies trim + lowercase behavior across the canonical
// edge cases: surrounding whitespace, uppercase, and empty input pass-through.
func TestNormalizeRole(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   Role
		want Role
	}{
		{name: "trims surrounding whitespace", in: Role("  builder  "), want: RoleBuilder},
		{name: "lowercases uppercase input", in: Role("BUILDER"), want: RoleBuilder},
		{name: "empty stays empty", in: Role(""), want: Role("")},
		{name: "mixed case + whitespace", in: Role("  QA-Proof  "), want: RoleQAProof},
		{name: "whitespace-only normalizes to empty", in: Role("   "), want: Role("")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := NormalizeRole(tc.in); got != tc.want {
				t.Fatalf("NormalizeRole(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestParseRoleFromDescription covers every contract path of the
// description-scanning parser: no Role line, first-match wins, line-anchor
// strictness, whitespace tolerance, case sensitivity, unknown values, and
// successful round-trip into typed Role constants.
func TestParseRoleFromDescription(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		desc    string
		want    Role
		wantErr error
	}{
		{
			name:    "empty description returns empty role and no error",
			desc:    "",
			want:    Role(""),
			wantErr: nil,
		},
		{
			name:    "description with no Role line returns empty role and no error",
			desc:    "Some unrelated paragraph.\nNothing here matches.",
			want:    Role(""),
			wantErr: nil,
		},
		{
			name:    "Role token mid-paragraph is ignored due to start-of-line anchor",
			desc:    "intro paragraph mentioning Role: builder inline\nbut not anchored",
			want:    Role(""),
			wantErr: nil,
		},
		{
			name:    "two Role lines — first wins",
			desc:    "Role: builder\nRole: planner",
			want:    RoleBuilder,
			wantErr: nil,
		},
		{
			name:    "trailing whitespace inside the line is tolerated",
			desc:    "Role:  builder  ",
			want:    RoleBuilder,
			wantErr: nil,
		},
		{
			name:    "unknown role value returns ErrInvalidRole",
			desc:    "Role: foobar",
			want:    Role(""),
			wantErr: ErrInvalidRole,
		},
		{
			name:    "capitalized value fails to match (regex captures [a-z-]+ only)",
			desc:    "Role: Builder",
			want:    Role(""),
			wantErr: nil,
		},
		{
			name:    "qa-proof round-trips into RoleQAProof",
			desc:    "Role: qa-proof",
			want:    RoleQAProof,
			wantErr: nil,
		},
		{
			name:    "builder canonical line",
			desc:    "Role: builder",
			want:    RoleBuilder,
			wantErr: nil,
		},
		{
			name:    "qa-falsification canonical line",
			desc:    "Role: qa-falsification",
			want:    RoleQAFalsification,
			wantErr: nil,
		},
		{
			name:    "qa-a11y canonical line",
			desc:    "Role: qa-a11y",
			want:    RoleQAA11y,
			wantErr: nil,
		},
		{
			name:    "qa-visual canonical line",
			desc:    "Role: qa-visual",
			want:    RoleQAVisual,
			wantErr: nil,
		},
		{
			name:    "design canonical line",
			desc:    "Role: design",
			want:    RoleDesign,
			wantErr: nil,
		},
		{
			name:    "commit canonical line",
			desc:    "Role: commit",
			want:    RoleCommit,
			wantErr: nil,
		},
		{
			name:    "planner canonical line",
			desc:    "Role: planner",
			want:    RolePlanner,
			wantErr: nil,
		},
		{
			name:    "research canonical line",
			desc:    "Role: research",
			want:    RoleResearch,
			wantErr: nil,
		},
		{
			name:    "Role line embedded in larger description",
			desc:    "Title line\nSome prose\nRole: builder\nMore prose",
			want:    RoleBuilder,
			wantErr: nil,
		},
		{
			name:    "hyphen-only captured value fails enum membership",
			desc:    "Role: -",
			want:    Role(""),
			wantErr: ErrInvalidRole,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseRoleFromDescription(tc.desc)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("ParseRoleFromDescription(%q) err = %v, want %v", tc.desc, err, tc.wantErr)
			}
			if got != tc.want {
				t.Fatalf("ParseRoleFromDescription(%q) = %q, want %q", tc.desc, got, tc.want)
			}
		})
	}
}
