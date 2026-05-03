package domain

import (
	"errors"
	"testing"
)

// TestIsValidStructuralType exercises the closed 4-value StructuralType enum
// membership check, including the explicit rejection of the empty string at
// this validator level (callers that want to permit an unset structural type
// short-circuit on emptiness themselves).
func TestIsValidStructuralType(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		st   StructuralType
		want bool
	}{
		{name: "drop", st: StructuralTypeDrop, want: true},
		{name: "segment", st: StructuralTypeSegment, want: true},
		{name: "confluence", st: StructuralTypeConfluence, want: true},
		{name: "droplet", st: StructuralTypeDroplet, want: true},
		{name: "empty string is invalid", st: StructuralType(""), want: false},
		{name: "unknown value", st: StructuralType("foobar"), want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := IsValidStructuralType(tc.st); got != tc.want {
				t.Fatalf("IsValidStructuralType(%q) = %v, want %v", tc.st, got, tc.want)
			}
		})
	}
}

// TestNormalizeStructuralType verifies trim + lowercase behavior across the
// canonical edge cases: surrounding whitespace, uppercase, and empty input
// pass-through.
func TestNormalizeStructuralType(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   StructuralType
		want StructuralType
	}{
		{name: "trims surrounding whitespace", in: StructuralType("  drop  "), want: StructuralTypeDrop},
		{name: "lowercases uppercase input", in: StructuralType("DROP"), want: StructuralTypeDrop},
		{name: "empty stays empty", in: StructuralType(""), want: StructuralType("")},
		{name: "mixed case + whitespace", in: StructuralType("  Confluence  "), want: StructuralTypeConfluence},
		{name: "whitespace-only normalizes to empty", in: StructuralType("   "), want: StructuralType("")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := NormalizeStructuralType(tc.in); got != tc.want {
				t.Fatalf("NormalizeStructuralType(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestParseStructuralTypeFromDescription covers every contract path of the
// description-scanning parser: no StructuralType line, first-match wins,
// line-anchor strictness, whitespace tolerance, case sensitivity, unknown
// values, and successful round-trip into typed StructuralType constants.
func TestParseStructuralTypeFromDescription(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		desc    string
		want    StructuralType
		wantErr error
	}{
		{
			name:    "empty description returns empty structural type and no error",
			desc:    "",
			want:    StructuralType(""),
			wantErr: nil,
		},
		{
			name:    "description with no StructuralType line returns empty and no error",
			desc:    "Some unrelated paragraph.\nNothing here matches.",
			want:    StructuralType(""),
			wantErr: nil,
		},
		{
			name:    "StructuralType token mid-paragraph is ignored due to start-of-line anchor",
			desc:    "intro paragraph mentioning StructuralType: drop inline\nbut not anchored",
			want:    StructuralType(""),
			wantErr: nil,
		},
		{
			name:    "two StructuralType lines — first wins",
			desc:    "StructuralType: drop\nStructuralType: segment",
			want:    StructuralTypeDrop,
			wantErr: nil,
		},
		{
			name:    "trailing whitespace inside the line is tolerated",
			desc:    "StructuralType:  drop  ",
			want:    StructuralTypeDrop,
			wantErr: nil,
		},
		{
			name:    "unknown structural-type value returns ErrInvalidStructuralType",
			desc:    "StructuralType: foobar",
			want:    StructuralType(""),
			wantErr: ErrInvalidStructuralType,
		},
		{
			name:    "capitalized value fails to match (regex captures [a-z]+ only)",
			desc:    "StructuralType: Drop",
			want:    StructuralType(""),
			wantErr: nil,
		},
		{
			name:    "hyphenated value fails to match (regex captures [a-z]+ only, no hyphens)",
			desc:    "StructuralType: drop-let",
			want:    StructuralType(""),
			wantErr: nil,
		},
		{
			name:    "drop canonical line",
			desc:    "StructuralType: drop",
			want:    StructuralTypeDrop,
			wantErr: nil,
		},
		{
			name:    "segment canonical line",
			desc:    "StructuralType: segment",
			want:    StructuralTypeSegment,
			wantErr: nil,
		},
		{
			name:    "confluence canonical line",
			desc:    "StructuralType: confluence",
			want:    StructuralTypeConfluence,
			wantErr: nil,
		},
		{
			name:    "droplet canonical line",
			desc:    "StructuralType: droplet",
			want:    StructuralTypeDroplet,
			wantErr: nil,
		},
		{
			name:    "StructuralType line embedded in larger description",
			desc:    "Title line\nSome prose\nStructuralType: droplet\nMore prose",
			want:    StructuralTypeDroplet,
			wantErr: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseStructuralTypeFromDescription(tc.desc)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("ParseStructuralTypeFromDescription(%q) err = %v, want %v", tc.desc, err, tc.wantErr)
			}
			if got != tc.want {
				t.Fatalf("ParseStructuralTypeFromDescription(%q) = %q, want %q", tc.desc, got, tc.want)
			}
		})
	}
}
