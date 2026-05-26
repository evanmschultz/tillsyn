package pretoolgate

import (
	"testing"
)

func TestGitSubcommand(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
		ok   bool
	}{
		// Basic cases.
		{
			name: "simple git commit",
			args: []string{"git", "commit", "-m", "wip"},
			want: "commit",
			ok:   true,
		},
		{
			name: "simple git push",
			args: []string{"git", "push", "origin", "main"},
			want: "push",
			ok:   true,
		},
		{
			name: "git with no subcommand",
			args: []string{"git"},
			want: "",
			ok:   false,
		},

		// Evasion: path-prefixed git.
		{
			name: "absolute path /usr/bin/git",
			args: []string{"/usr/bin/git", "commit", "-m", "x"},
			want: "commit",
			ok:   true,
		},
		{
			name: "relative path ./git",
			args: []string{"./git", "push"},
			want: "push",
			ok:   true,
		},

		// Evasion: env-prefix assignments.
		{
			name: "env-prefix FOO=1 git commit",
			args: []string{"FOO=1", "git", "commit"},
			want: "commit",
			ok:   true,
		},
		{
			name: "multiple env-prefix assignments",
			args: []string{"FOO=1", "BAR=val", "git", "push"},
			want: "push",
			ok:   true,
		},
		{
			name: "env-prefix + path-prefixed git",
			args: []string{"FOO=1", "/usr/bin/git", "push", "origin", "main"},
			want: "push",
			ok:   true,
		},

		// Evasion: global options with args.
		{
			name: "git -C /repo commit",
			args: []string{"git", "-C", "/repo", "commit", "-m", "x"},
			want: "commit",
			ok:   true,
		},
		{
			name: "git --git-dir=/path commit",
			args: []string{"git", "--git-dir=/path", "commit"},
			want: "commit",
			ok:   true,
		},
		{
			name: "git -c core.editor=vim commit",
			args: []string{"git", "-c", "core.editor=vim", "commit"},
			want: "commit",
			ok:   true,
		},
		{
			name: "git --work-tree=/path add",
			args: []string{"git", "--work-tree=/path", "add", "file.go"},
			want: "add",
			ok:   true,
		},

		// Evasion: inline flag assignment.
		{
			name: "git --git-dir=x commit (inline =)",
			args: []string{"git", "--git-dir=x", "commit"},
			want: "commit",
			ok:   true,
		},

		// Read-only operations (should be recognized but not blocked by gitMutation).
		{
			name: "git diff --stat",
			args: []string{"git", "diff", "--stat"},
			want: "diff",
			ok:   true,
		},
		{
			name: "git status",
			args: []string{"git", "status"},
			want: "status",
			ok:   true,
		},
		{
			name: "git log",
			args: []string{"git", "log"},
			want: "log",
			ok:   true,
		},
		{
			name: "git show",
			args: []string{"git", "show", "HEAD"},
			want: "show",
			ok:   true,
		},

		// No git at all.
		{
			name: "no git token",
			args: []string{"mage", "ci"},
			want: "",
			ok:   false,
		},
		{
			name: "empty input",
			args: []string{},
			want: "",
			ok:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := gitSubcommand(tt.args)
			if got != tt.want || ok != tt.ok {
				t.Errorf("gitSubcommand(%v) = (%q, %v); want (%q, %v)",
					tt.args, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestGitVerbsFromDeny(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		want     map[string]struct{}
	}{
		{
			name:     "empty patterns",
			patterns: []string{},
			want:     map[string]struct{}{},
		},
		{
			name:     "single git pattern",
			patterns: []string{"git commit"},
			want:     map[string]struct{}{"commit": {}},
		},
		{
			name:     "multiple git patterns",
			patterns: []string{"git commit", "git push", "git add"},
			want: map[string]struct{}{
				"commit": {},
				"push":   {},
				"add":    {},
			},
		},
		{
			name:     "mixed git and non-git patterns",
			patterns: []string{"git commit", "mage install", "git push", "go get"},
			want: map[string]struct{}{
				"commit": {},
				"push":   {},
			},
		},
		{
			name:     "non-git patterns only",
			patterns: []string{"mage install", "go get", "go mod"},
			want:     map[string]struct{}{},
		},
		{
			name:     "pattern with leading/trailing whitespace",
			patterns: []string{"  git commit  ", "git push"},
			want: map[string]struct{}{
				"commit": {},
				"push":   {},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gitVerbsFromDeny(tt.patterns)
			if !mapsEqual(got, tt.want) {
				t.Errorf("gitVerbsFromDeny(%v) = %v; want %v",
					tt.patterns, got, tt.want)
			}
		})
	}
}

func TestGitMutation(t *testing.T) {
	tests := []struct {
		name    string
		command string
		wantVrb string
		ok      bool
	}{
		// Mutation cases (should be blocked).
		{
			name:    "git commit -m x",
			command: "git commit -m wip",
			wantVrb: "git commit",
			ok:      true,
		},
		{
			name:    "git push origin main",
			command: "git push origin main",
			wantVrb: "git push",
			ok:      true,
		},
		{
			name:    "git add .",
			command: "git add .",
			wantVrb: "git add",
			ok:      true,
		},

		// Evasion cases (should still be blocked).
		{
			name:    "git -C /repo commit (evasion: global flag)",
			command: "git -C /repo commit -m wip",
			wantVrb: "git commit",
			ok:      true,
		},
		{
			name:    "FOO=1 /usr/bin/git push (evasion: env + path)",
			command: "FOO=1 /usr/bin/git push origin main",
			wantVrb: "git push",
			ok:      true,
		},
		{
			name:    "git --git-dir=x commit (evasion: inline flag)",
			command: "git --git-dir=x commit",
			wantVrb: "git commit",
			ok:      true,
		},

		// Shell chaining (should catch the mutation, not the read).
		{
			name:    "git diff | git push (chained: first read, second mutate)",
			command: "git diff | git push",
			wantVrb: "git push",
			ok:      true,
		},
		{
			name:    "git status && git commit (semicolon chain)",
			command: "git status; git commit -m x",
			wantVrb: "git commit",
			ok:      true,
		},
		{
			name:    "git log & git push (ampersand chain)",
			command: "git log & git push",
			wantVrb: "git push",
			ok:      true,
		},

		// Read-only cases (should NOT be blocked).
		{
			name:    "git diff --stat",
			command: "git diff --stat",
			wantVrb: "",
			ok:      false,
		},
		{
			name:    "git status",
			command: "git status",
			wantVrb: "",
			ok:      false,
		},
		{
			name:    "git log --oneline",
			command: "git log --oneline",
			wantVrb: "",
			ok:      false,
		},
		{
			name:    "git show HEAD",
			command: "git show HEAD",
			wantVrb: "",
			ok:      false,
		},
		{
			name:    "git rev-parse HEAD",
			command: "git rev-parse HEAD",
			wantVrb: "",
			ok:      false,
		},

		// Non-git cases (should NOT be blocked).
		{
			name:    "mage ci",
			command: "mage ci",
			wantVrb: "",
			ok:      false,
		},
		{
			name:    "go doc fmt",
			command: "go doc fmt",
			wantVrb: "",
			ok:      false,
		},
		{
			name:    "empty command",
			command: "",
			wantVrb: "",
			ok:      false,
		},

		// Mutation verbs in the hardcoded set.
		{
			name:    "git fetch (mutation in set)",
			command: "git fetch",
			wantVrb: "git fetch",
			ok:      true,
		},
		{
			name:    "git pull (mutation in set)",
			command: "git pull",
			wantVrb: "git pull",
			ok:      true,
		},
		{
			name:    "git clone (mutation in set)",
			command: "git clone https://example.com/repo.git",
			wantVrb: "git clone",
			ok:      true,
		},
		{
			name:    "git merge (mutation in set)",
			command: "git merge feature",
			wantVrb: "git merge",
			ok:      true,
		},
		{
			name:    "git reset (mutation in set)",
			command: "git reset --hard",
			wantVrb: "git reset",
			ok:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := gitMutation(tt.command)
			if got != tt.wantVrb || ok != tt.ok {
				t.Errorf("gitMutation(%q) = (%q, %v); want (%q, %v)",
					tt.command, got, ok, tt.wantVrb, tt.ok)
			}
		})
	}
}

// TestGitMutationVerbsSize asserts the hardcoded frozenset has exactly 28 verbs.
func TestGitMutationVerbsSize(t *testing.T) {
	if len(gitMutationVerbs) != 28 {
		t.Errorf("gitMutationVerbs has %d verbs; want exactly 28", len(gitMutationVerbs))
	}
}

// TestGitMutationVerbsContent verifies the frozenset contains the expected verbs.
func TestGitMutationVerbsContent(t *testing.T) {
	expected := []string{
		"commit", "push", "add", "reset", "rebase", "merge", "checkout",
		"branch", "tag", "stash", "restore", "cherry-pick", "am", "clean",
		"switch", "rm", "mv", "update-ref", "gc", "prune", "worktree",
		"submodule", "init", "clone", "fetch", "pull", "remote", "apply",
	}
	if len(expected) != 28 {
		t.Fatalf("test bug: expected list has %d verbs, not 28", len(expected))
	}

	for _, verb := range expected {
		if _, ok := gitMutationVerbs[verb]; !ok {
			t.Errorf("gitMutationVerbs missing verb: %s", verb)
		}
	}

	if len(gitMutationVerbs) > len(expected) {
		t.Errorf("gitMutationVerbs has extra verbs beyond the expected %d", len(expected))
	}
}

// Helper: compare two map[string]struct{} for equality.
func mapsEqual(a, b map[string]struct{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			return false
		}
	}
	return true
}
