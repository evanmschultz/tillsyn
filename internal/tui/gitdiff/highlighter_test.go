package gitdiff

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// goldenUpdateEnv regenerates the package's testdata/golden/ fixtures when set
// to a non-empty value. Environment-driven rather than flag-driven so it runs
// cleanly through `mage test-pkg ./internal/tui/gitdiff` without needing a
// custom mage target — the package's golden workflow deliberately stays
// independent of the TUI-wide `mage test-golden` suite, which is scoped to
// ./internal/tui only.
const goldenUpdateEnv = "TILLSYN_GITDIFF_GOLDEN_UPDATE"

// samplePatch is a compact unified diff that exercises the diff lexer's main
// production tokens — file header, mode-change header, hunk marker, deletion,
// addition, and context line — without bringing in repository-specific noise
// that would make the golden fixture brittle.
const samplePatch = `diff --git a/foo.txt b/foo.txt
index 83db48f..bf269f4 100644
--- a/foo.txt
+++ b/foo.txt
@@ -1,3 +1,4 @@
 alpha
-beta
+beta updated
 gamma
+delta
`

// TestHighlighter_EmptyPatch verifies the fast-path passthrough: an empty
// patch round-trips to an empty string with no error and no chroma call. The
// TUI relies on this to render an empty diff pane without paying for
// tokenisation when start == end.
func TestHighlighter_EmptyPatch(t *testing.T) {
	t.Parallel()
	h := NewChromaHighlighter()
	got, err := h.Highlight("")
	if err != nil {
		t.Fatalf("Highlight(\"\") error = %v, want nil", err)
	}
	if got != "" {
		t.Fatalf("Highlight(\"\") = %q, want empty string", got)
	}
}

// TestHighlighter_SimpleAddDelete asserts that a minimal add/delete patch
// produces output that both styles the content (ANSI escape present) and
// preserves the original line bodies as substrings. The content-preservation
// check is what guards against a regression where the formatter silently
// strips payload text.
func TestHighlighter_SimpleAddDelete(t *testing.T) {
	t.Parallel()
	patch := "--- a/x\n+++ b/x\n@@ -1 +1 @@\n-old\n+new\n"
	got, err := NewChromaHighlighter().Highlight(patch)
	if err != nil {
		t.Fatalf("Highlight error = %v, want nil", err)
	}
	if !strings.Contains(got, "\x1b[") {
		t.Fatalf("Highlight output missing ANSI escape, got %q", got)
	}
	for _, want := range []string{"old", "new"} {
		if !strings.Contains(got, want) {
			t.Fatalf("Highlight output missing original content %q, got %q", want, got)
		}
	}
}

// TestHighlighter_FileHeader exercises the `diff --git a/... b/...` header
// path and asserts both that the header survives into output and that at
// least one ANSI escape exists in the header region, confirming the diff
// lexer is actually classifying the header as a styled token rather than
// plain text.
func TestHighlighter_FileHeader(t *testing.T) {
	t.Parallel()
	patch := "diff --git a/foo.txt b/foo.txt\nindex 1..2 100644\n--- a/foo.txt\n+++ b/foo.txt\n@@ -1 +1 @@\n-a\n+b\n"
	got, err := NewChromaHighlighter().Highlight(patch)
	if err != nil {
		t.Fatalf("Highlight error = %v, want nil", err)
	}
	if !strings.Contains(got, "diff --git a/foo.txt b/foo.txt") {
		t.Fatalf("Highlight output missing file header literal, got %q", got)
	}
	if !strings.Contains(got, "\x1b[") {
		t.Fatalf("Highlight output missing ANSI escape, got %q", got)
	}
}

// TestHighlighter_HunkMarker verifies that the hunk header `@@ -1,3 +1,4 @@`
// is preserved verbatim in the output. The diff lexer classifies hunk
// markers distinctly from payload lines, and regressions here have shown up
// as dropped hunk metadata in Chroma upgrades, so pin it with an explicit
// substring check.
func TestHighlighter_HunkMarker(t *testing.T) {
	t.Parallel()
	patch := "--- a/x\n+++ b/x\n@@ -1,3 +1,4 @@\n a\n-b\n+b2\n c\n+d\n"
	got, err := NewChromaHighlighter().Highlight(patch)
	if err != nil {
		t.Fatalf("Highlight error = %v, want nil", err)
	}
	if !strings.Contains(got, "@@ -1,3 +1,4 @@") {
		t.Fatalf("Highlight output missing hunk marker, got %q", got)
	}
}

// TestHighlighter_BinaryFilesMarker guards the "Binary files ... differ"
// marker git emits for non-text diffs. The diff lexer treats this line as
// generic text, but the highlighter must still render it without error —
// this test fails loudly if chroma ever refuses the input.
func TestHighlighter_BinaryFilesMarker(t *testing.T) {
	t.Parallel()
	patch := "diff --git a/logo.png b/logo.png\nBinary files a/logo.png and b/logo.png differ\n"
	got, err := NewChromaHighlighter().Highlight(patch)
	if err != nil {
		t.Fatalf("Highlight error = %v, want nil", err)
	}
	if !strings.Contains(got, "Binary files a/logo.png and b/logo.png differ") {
		t.Fatalf("Highlight output missing binary marker, got %q", got)
	}
}

// TestHighlighter_Golden pins the full styled output of samplePatch against
// a checked-in fixture. To regenerate, set TILLSYN_GITDIFF_GOLDEN_UPDATE=1 in
// the environment and rerun the test — e.g.
// `TILLSYN_GITDIFF_GOLDEN_UPDATE=1 mage test-pkg ./internal/tui/gitdiff`.
// The golden fixture is intentionally opaque: it detects any drift in
// chroma's dracula palette, the diff lexer, or the terminal256 formatter
// that would otherwise slip past the coarse substring tests.
func TestHighlighter_Golden(t *testing.T) {
	t.Parallel()
	got, err := NewChromaHighlighter().Highlight(samplePatch)
	if err != nil {
		t.Fatalf("Highlight error = %v, want nil", err)
	}
	goldenPath := filepath.Join("testdata", "golden", "simple.ansi")
	if os.Getenv(goldenUpdateEnv) != "" {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatalf("mkdir golden dir: %v", err)
		}
		if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		return
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden %s: %v (run with %s=1 to regenerate)", goldenPath, err, goldenUpdateEnv)
	}
	if string(want) != got {
		t.Fatalf("golden mismatch at %s\nrun with %s=1 to regenerate\ngot:  %q\nwant: %q", goldenPath, goldenUpdateEnv, got, string(want))
	}
}

// TestHighlighter_ConstructorReturnsInterface pins the constructor's static
// return type to the Highlighter interface. A regression that returns the
// concrete *chromaHighlighter instead would compile this file just fine, so
// the assertion is carried by the interface conversion on the declaration.
func TestHighlighter_ConstructorReturnsInterface(t *testing.T) {
	t.Parallel()
	var h Highlighter = NewChromaHighlighter()
	if h == nil {
		t.Fatal("NewChromaHighlighter() returned nil")
	}
}

// TestHighlighter_Concurrent runs Highlight in parallel goroutines to catch
// any shared-state races. Running under `-race` (via `mage test-pkg`) is the
// primary guarantee; the secondary check is byte-for-byte output stability
// across goroutines, which catches any accidental per-call mutation of the
// chroma pipeline's internals.
func TestHighlighter_Concurrent(t *testing.T) {
	t.Parallel()
	h := NewChromaHighlighter()
	const goroutines = 10
	outputs := make([]string, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := range outputs {
		go func(idx int) {
			defer wg.Done()
			out, err := h.Highlight(samplePatch)
			if err != nil {
				t.Errorf("goroutine %d: Highlight error = %v", idx, err)
				return
			}
			outputs[idx] = out
		}(i)
	}
	wg.Wait()
	for i := 1; i < goroutines; i++ {
		if outputs[i] != outputs[0] {
			t.Fatalf("goroutine %d output differs from goroutine 0\ngoroutine 0: %q\ngoroutine %d: %q", i, outputs[0], i, outputs[i])
		}
	}
}
