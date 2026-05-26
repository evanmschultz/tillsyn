package cli_codex

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderExecpolicyRules(t *testing.T) {
	tests := []struct {
		name      string
		bashDeny  []string
		wantVerbs int
		wantLines func(lines []string) bool
	}{
		{
			name:      "nil bashDeny produces 28-verb floor",
			bashDeny:  nil,
			wantVerbs: 28,
			wantLines: func(lines []string) bool {
				if len(lines) != 28 {
					return false
				}
				// Verify first verb is "commit".
				if !strings.Contains(lines[0], `["git", "commit"]`) {
					return false
				}
				// Verify fetch, pull, remote, apply are present (should be at end).
				hasAll := true
				for _, verb := range []string{"fetch", "pull", "remote", "apply"} {
					found := false
					for _, line := range lines {
						if strings.Contains(line, `"git", "`+verb+`"`) {
							found = true
							break
						}
					}
					hasAll = hasAll && found
				}
				return hasAll
			},
		},
		{
			name:      "empty bashDeny produces 28-verb floor",
			bashDeny:  []string{},
			wantVerbs: 28,
			wantLines: func(lines []string) bool { return len(lines) == 28 },
		},
		{
			name:      "bashDeny with non-git pattern adds one rule",
			bashDeny:  []string{"mage install"},
			wantVerbs: 29,
			wantLines: func(lines []string) bool {
				if len(lines) != 29 {
					return false
				}
				// Last line should be the mage rule.
				return strings.Contains(lines[28], `pattern=["mage", "install"]`)
			},
		},
		{
			name:      "bashDeny with git-prefixed pattern skipped",
			bashDeny:  []string{"git commit"},
			wantVerbs: 28,
			wantLines: func(lines []string) bool { return len(lines) == 28 },
		},
		{
			name:      "bashDeny with multiple tokens tokenized correctly",
			bashDeny:  []string{"go get -u"},
			wantVerbs: 29,
			wantLines: func(lines []string) bool {
				if len(lines) != 29 {
					return false
				}
				// Last line should contain three quoted tokens.
				return strings.Contains(lines[28], `"go"`) &&
					strings.Contains(lines[28], `"get"`) &&
					strings.Contains(lines[28], `"-u"`)
			},
		},
		{
			name:      "bashDeny with whitespace-only entry skipped",
			bashDeny:  []string{"   "},
			wantVerbs: 28,
			wantLines: func(lines []string) bool { return len(lines) == 28 },
		},
		{
			name:      "bashDeny with mixed git and non-git entries",
			bashDeny:  []string{"mage install", "git reset", "go mod tidy"},
			wantVerbs: 30,
			wantLines: func(lines []string) bool {
				if len(lines) != 30 {
					return false
				}
				// Should have 28 git verbs + 2 non-git rules (git reset skipped).
				hasMage := false
				hasGoMod := false
				for i := 28; i < 30; i++ {
					if strings.Contains(lines[i], "mage") {
						hasMage = true
					}
					if strings.Contains(lines[i], "go") && strings.Contains(lines[i], "mod") {
						hasGoMod = true
					}
				}
				return hasMage && hasGoMod
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := renderExecpolicyRules(tt.bashDeny)
			lines := strings.Split(strings.TrimSpace(out), "\n")

			if len(lines) != tt.wantVerbs {
				t.Errorf("renderExecpolicyRules(%v) produced %d lines, want %d",
					tt.bashDeny, len(lines), tt.wantVerbs)
			}

			if !tt.wantLines(lines) {
				t.Errorf("renderExecpolicyRules(%v) output failed validation:\n%s",
					tt.bashDeny, out)
			}
		})
	}
}

func TestWriteExecpolicyRules(t *testing.T) {
	tmpDir := t.TempDir()
	codexHome := filepath.Join(tmpDir, "codex-home")

	// Create codex home dir (simulating what newHermeticCodexHome does).
	if err := os.Mkdir(codexHome, 0o755); err != nil {
		t.Fatalf("mkdir codex home: %v", err)
	}

	bashDeny := []string{"mage install", "go get"}
	if err := writeExecpolicyRules(codexHome, bashDeny); err != nil {
		t.Fatalf("writeExecpolicyRules() error = %v", err)
	}

	// Verify file exists.
	rulesFile := filepath.Join(codexHome, "rules", "default.rules")
	data, err := os.ReadFile(rulesFile)
	if err != nil {
		t.Fatalf("read rules file: %v", err)
	}

	content := string(data)
	lines := strings.Split(strings.TrimSpace(content), "\n")

	// Should have 28 verbs + 2 bashDeny rules.
	if len(lines) != 30 {
		t.Errorf("writeExecpolicyRules wrote %d lines, want 30", len(lines))
	}

	// Verify it matches what renderExecpolicyRules produces.
	expected := renderExecpolicyRules(bashDeny)
	if content != expected {
		t.Errorf("file content mismatch:\ngot:\n%s\n\nwant:\n%s", content, expected)
	}
}
