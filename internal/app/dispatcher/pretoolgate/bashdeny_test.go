package pretoolgate

import (
	"testing"
)

func TestBashForbidden(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		denyPattern []string
		wantPattern string
		ok          bool
	}{
		// Case 11 from gate_test.sh: mage install (bash_deny pattern).
		{
			name:        "mage install (deny pattern)",
			command:     "mage install",
			denyPattern: []string{"git commit", "git push", "git add", "mage install", "go get"},
			wantPattern: "mage install",
			ok:          true,
		},
		// Case 12 from gate_test.sh: git diff (read, allowed).
		{
			name:        "git diff (read, allowed)",
			command:     "git diff --stat",
			denyPattern: []string{"git commit", "git push", "git add", "mage install"},
			wantPattern: "",
			ok:          false,
		},
		// Case 13 from gate_test.sh: mage ci (allowed).
		{
			name:        "mage ci (allowed)",
			command:     "mage ci",
			denyPattern: []string{"git commit", "git push", "git add", "mage install", "go get"},
			wantPattern: "",
			ok:          false,
		},
		// Additional: git -C /repo commit (evasion with global flag).
		{
			name:        "git -C /repo commit (evasion: global flag)",
			command:     "git -C /repo commit -m wip",
			denyPattern: []string{"git commit"},
			wantPattern: "git commit",
			ok:          true,
		},
		// Additional: go get (deny pattern).
		{
			name:        "go get (deny pattern)",
			command:     "go get github.com/some/pkg",
			denyPattern: []string{"go get", "go mod"},
			wantPattern: "go get",
			ok:          true,
		},
		// Additional: go doc (not denied).
		{
			name:        "go doc (not denied)",
			command:     "go doc fmt",
			denyPattern: []string{"go get", "go mod"},
			wantPattern: "",
			ok:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := bashForbidden(tt.command, tt.denyPattern)
			if got != tt.wantPattern || ok != tt.ok {
				t.Errorf("bashForbidden(%q, %v) = (%q, %v); want (%q, %v)",
					tt.command, tt.denyPattern, got, ok, tt.wantPattern, tt.ok)
			}
		})
	}
}

func TestBashWriteVector(t *testing.T) {
	tests := []struct {
		name    string
		command string
		wantDesc string
		ok      bool
	}{
		// Case 8: echo > file (output redirection).
		{
			name:     "echo > file (output redirection)",
			command:  "echo pwned > /repo/b.go",
			wantDesc: "output redirection (> / >>)",
			ok:       true,
		},
		// Case 9: sed -i (in-place edit).
		{
			name:     "sed -i s/a/b/ file",
			command:  "sed -i s/a/b/ /repo/a.go",
			wantDesc: "sed -i (in-place edit)",
			ok:       true,
		},
		// Case 10: python3 -c (interpreter).
		{
			name:     "python3 -c (interpreter)",
			command:  `python3 -c "open(0,1)"`,
			wantDesc: "interpreter (can write files)",
			ok:       true,
		},
		// NEGATIVE: > /dev/null (output redirection to /dev/null is excluded).
		{
			name:     "echo > /dev/null (excluded)",
			command:  "echo test > /dev/null",
			wantDesc: "",
			ok:       false,
		},
		// NEGATIVE: >& stderr (stderr redirect is excluded).
		{
			name:     "cmd >& /tmp/err (excluded)",
			command:  "cmd >& /tmp/err",
			wantDesc: "",
			ok:       false,
		},
		// NEGATIVE: git diff (read-only, no write vector).
		{
			name:     "git diff (read-only)",
			command:  "git diff --stat",
			wantDesc: "",
			ok:       false,
		},
		// NEGATIVE: mage ci (build tool, no write vector).
		{
			name:     "mage ci (build, allowed)",
			command:  "mage ci",
			wantDesc: "",
			ok:       false,
		},
		// tee (write vector).
		{
			name:     "cat file | tee output.txt",
			command:  "cat file | tee output.txt",
			wantDesc: "tee",
			ok:       true,
		},
		// cp (file-mutating command).
		{
			name:     "cp /src /dst",
			command:  "cp /src /dst",
			wantDesc: "file-mutating command",
			ok:       true,
		},
		// rm (file-mutating command).
		{
			name:     "rm /repo/old.go",
			command:  "rm /repo/old.go",
			wantDesc: "file-mutating command",
			ok:       true,
		},
		// perl -i (in-place edit).
		{
			name:     "perl -i -pe 's/a/b/' file.pl",
			command:  "perl -i -pe 's/a/b/' file.pl",
			wantDesc: "perl/ruby -i (in-place edit)",
			ok:       true,
		},
		// python (interpreter).
		{
			name:     "python -c code",
			command:  `python -c "import os; os.remove('x')"`,
			wantDesc: "interpreter (can write files)",
			ok:       true,
		},
		// node (interpreter).
		{
			name:     "node -e script",
			command:  "node -e \"require('fs').unlink('x')\"",
			wantDesc: "interpreter (can write files)",
			ok:       true,
		},
		// dd with of= (write vector).
		{
			name:     "dd if=/src of=/dst",
			command:  "dd if=/src of=/dst",
			wantDesc: "dd of=",
			ok:       true,
		},
		// touch (file-mutating command).
		{
			name:     "touch /repo/new.go",
			command:  "touch /repo/new.go",
			wantDesc: "file-mutating command",
			ok:       true,
		},
		// mkdir (file-mutating command).
		{
			name:     "mkdir /repo/newdir",
			command:  "mkdir /repo/newdir",
			wantDesc: "file-mutating command",
			ok:       true,
		},
		// chmod (file-mutating command).
		{
			name:     "chmod 755 /repo/script.sh",
			command:  "chmod 755 /repo/script.sh",
			wantDesc: "file-mutating command",
			ok:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := bashWriteVector(tt.command)
			if got != tt.wantDesc || ok != tt.ok {
				t.Errorf("bashWriteVector(%q) = (%q, %v); want (%q, %v)",
					tt.command, got, ok, tt.wantDesc, tt.ok)
			}
		})
	}
}
