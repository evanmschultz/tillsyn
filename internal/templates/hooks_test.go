package templates

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"regexp"
	"testing"
)

// TestHookTemplateEmbedded verifies that the validate-action-item-paths hook
// template is reachable via DefaultTemplateFS and contains the expected
// structural markers: a shebang line and the tillsyn-hook-hash header.
func TestHookTemplateEmbedded(t *testing.T) {
	t.Parallel()

	const path = "builtin/hooks/validate-action-item-paths.sh.tmpl"

	f, err := DefaultTemplateFS.Open(path)
	if err != nil {
		t.Fatalf("DefaultTemplateFS.Open(%q): unexpected error: %v", path, err)
	}
	defer f.Close()

	body, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("read %q: %v", path, err)
	}
	if len(body) == 0 {
		t.Fatalf("read %q: body is empty", path)
	}

	content := string(body)

	// Must contain a shebang line.
	if !containsAny(content, "#!/bin/bash", "#!/usr/bin/env bash") {
		t.Errorf("%q: missing shebang line (expected #!/bin/bash or #!/usr/bin/env bash)", path)
	}

	// Must contain the tillsyn-hook-hash header placeholder.
	if !containsSubstring(content, "# tillsyn-hook-hash:") {
		t.Errorf("%q: missing '# tillsyn-hook-hash:' header line", path)
	}
}

// TestComputeHookHash_Deterministic verifies that two successive calls to
// ComputeHookHash return the same value. The embed.FS is documented
// thread-safe for concurrent reads; this test proves the output is stable.
func TestComputeHookHash_Deterministic(t *testing.T) {
	t.Parallel()

	h1, err := ComputeHookHash()
	if err != nil {
		t.Fatalf("ComputeHookHash() first call: %v", err)
	}
	h2, err := ComputeHookHash()
	if err != nil {
		t.Fatalf("ComputeHookHash() second call: %v", err)
	}
	if h1 != h2 {
		t.Errorf("ComputeHookHash() not deterministic: %q != %q", h1, h2)
	}
}

// TestComputeHookHash_Format verifies the hash is 64 lowercase hex characters.
func TestComputeHookHash_Format(t *testing.T) {
	t.Parallel()

	h, err := ComputeHookHash()
	if err != nil {
		t.Fatalf("ComputeHookHash(): %v", err)
	}
	if len(h) != 64 {
		t.Errorf("ComputeHookHash() length = %d; want 64", len(h))
	}
	matched, _ := regexp.MatchString(`^[0-9a-f]{64}$`, h)
	if !matched {
		t.Errorf("ComputeHookHash() = %q; want 64 lowercase hex chars matching ^[0-9a-f]{64}$", h)
	}
}

// TestComputeHookHash_MatchesContent verifies that ComputeHookHash returns
// the sha256 of the embedded template bytes, computed independently.
func TestComputeHookHash_MatchesContent(t *testing.T) {
	t.Parallel()

	const path = "builtin/hooks/validate-action-item-paths.sh.tmpl"

	raw, err := DefaultTemplateFS.ReadFile(path)
	if err != nil {
		t.Fatalf("DefaultTemplateFS.ReadFile(%q): %v", path, err)
	}

	sum := sha256.Sum256(raw)
	want := hex.EncodeToString(sum[:])

	got, err := ComputeHookHash()
	if err != nil {
		t.Fatalf("ComputeHookHash(): %v", err)
	}
	if got != want {
		t.Errorf("ComputeHookHash() = %q; want %q (sha256 of embedded bytes)", got, want)
	}
}

// containsSubstring reports whether s contains substr.
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

// containsAny reports whether s contains at least one of the given substrings.
func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if containsSubstring(s, sub) {
			return true
		}
	}
	return false
}

// indexOf returns the index of the first occurrence of substr in s, or -1.
func indexOf(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
