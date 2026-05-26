package pretoolgate

import (
	"testing"
)

func TestNormPath(t *testing.T) {
	tests := []struct {
		name string
		p    string
		cwd  string
		want string
	}{
		{
			name: "empty path",
			p:    "",
			cwd:  "/repo",
			want: "",
		},
		{
			name: "absolute path",
			p:    "/repo/a.go",
			cwd:  "/somewhere",
			want: "/repo/a.go",
		},
		{
			name: "relative path joined and cleaned",
			p:    "file.go",
			cwd:  "/repo",
			want: "/repo/file.go",
		},
		{
			name: "relative with directory components",
			p:    "cmd/main.go",
			cwd:  "/repo",
			want: "/repo/cmd/main.go",
		},
		{
			name: "path normalization removes ..",
			p:    "/repo/cmd/../file.go",
			cwd:  "/somewhere",
			want: "/repo/file.go",
		},
		{
			name: "relative path with ..",
			p:    "cmd/../file.go",
			cwd:  "/repo",
			want: "/repo/file.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normPath(tt.p, tt.cwd)
			if got != tt.want {
				t.Errorf("normPath(%q, %q) = %q, want %q", tt.p, tt.cwd, got, tt.want)
			}
		})
	}
}

func TestEditAllowed(t *testing.T) {
	const cwd = "/repo"

	tests := []struct {
		name    string
		file    string
		allowed []string
		cwd     string
		want    bool
	}{
		// Gate_test.sh case 2: in-scope file
		{
			name:    "case 2: exact match absolute",
			file:    "/repo/a.go",
			allowed: []string{"/repo/a.go"},
			cwd:     cwd,
			want:    true,
		},
		// Gate_test.sh case 3: off-scope file
		{
			name:    "case 3: no match off-scope",
			file:    "/repo/b.go",
			allowed: []string{"/repo/a.go"},
			cwd:     cwd,
			want:    false,
		},
		// Empty allowlist (deny-all, e.g., read-only role)
		{
			name:    "case 4: empty allowed list",
			file:    "/repo/a.go",
			allowed: []string{},
			cwd:     cwd,
			want:    false,
		},
		// Relative file path joined against cwd
		{
			name:    "relative file path",
			file:    "a.go",
			allowed: []string{"/repo/a.go"},
			cwd:     cwd,
			want:    true,
		},
		// Glob pattern (wildcard)
		{
			name:    "glob on normalized paths",
			file:    "/repo/cmd/main.go",
			allowed: []string{"/repo/cmd/*.go"},
			cwd:     cwd,
			want:    true,
		},
		// Glob pattern with relative allowed entry
		{
			name:    "glob with relative allowed entry",
			file:    "/repo/cmd/main.go",
			allowed: []string{"cmd/*.go"},
			cwd:     cwd,
			want:    true,
		},
		// Glob pattern on raw forms (entry as pattern)
		{
			name:    "glob on raw forms",
			file:    "a.go",
			allowed: []string{"*.go"},
			cwd:     cwd,
			want:    true,
		},
		// Multiple entries, first doesn't match
		{
			name:    "multiple entries, match second",
			file:    "/repo/b.go",
			allowed: []string{"/repo/a.go", "/repo/b.go"},
			cwd:     cwd,
			want:    true,
		},
		// Multiple entries, no match
		{
			name:    "multiple entries, no match",
			file:    "/repo/c.go",
			allowed: []string{"/repo/a.go", "/repo/b.go"},
			cwd:     cwd,
			want:    false,
		},
		// Glob doesn't match
		{
			name:    "glob no match",
			file:    "/repo/cmd/main.go",
			allowed: []string{"/repo/*.go"},
			cwd:     cwd,
			want:    false,
		},
		// Glob pattern with */
		{
			name:    "one-level glob pattern",
			file:    "/repo/internal/pkg/foo.go",
			allowed: []string{"/repo/internal/*/*.go"},
			cwd:     cwd,
			want:    true, // path.Match(/repo/internal/*/*.go, /repo/internal/pkg/foo.go) = true
		},
		// Path with normalization (..)
		{
			name:    "path normalization in allowed",
			file:    "/repo/a.go",
			allowed: []string{"/repo/cmd/../a.go"},
			cwd:     cwd,
			want:    true,
		},
		// Relative path in both file and allowed
		{
			name:    "relative file and allowed",
			file:    "a.go",
			allowed: []string{"a.go"},
			cwd:     cwd,
			want:    true,
		},
		// Case-sensitive match (on case-sensitive filesystem)
		{
			name:    "case-sensitive mismatch",
			file:    "/repo/A.go",
			allowed: []string{"/repo/a.go"},
			cwd:     cwd,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := editAllowed(tt.file, tt.allowed, tt.cwd)
			if got != tt.want {
				t.Errorf("editAllowed(%q, %v, %q) = %v, want %v",
					tt.file, tt.allowed, tt.cwd, got, tt.want)
			}
		})
	}
}
