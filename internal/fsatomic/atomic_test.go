// Package fsatomic — atomic file-write tests.
//
// Each test runs against a fresh t.TempDir() to keep the cases hermetic and
// independent of any filesystem state outside the test.
package fsatomic

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestWriteFile_FreshWrite verifies that WriteFile writes new content to a
// non-existent target and the resulting file holds the exact bytes passed in.
func TestWriteFile_FreshWrite(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "fresh.txt")
	want := []byte("hello fsatomic")

	if err := WriteFile(target, want, 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile after WriteFile: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("content mismatch: got %q, want %q", got, want)
	}
}

// TestWriteFile_OverwritesExisting verifies that WriteFile replaces the
// contents of a pre-existing target without leaving stale bytes behind.
func TestWriteFile_OverwritesExisting(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "overwrite.txt")

	// Seed with old content using stdlib so the test isolates the
	// behavior-under-test from the function-under-test.
	if err := os.WriteFile(target, []byte("OLD CONTENT — much longer than the replacement"), 0o644); err != nil {
		t.Fatalf("seed write: %v", err)
	}

	want := []byte("new")
	if err := WriteFile(target, want, 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile after WriteFile: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("content mismatch: got %q, want %q", got, want)
	}
}

// TestWriteFile_CleansUpTempOnError verifies that when WriteFile fails — here,
// because the target's parent directory does not exist — no .tmp-* residue is
// left behind in the parent tree. The parent dir we observe is the *real*
// existing TempDir; the failing call points one level deeper into a
// nonexistent child.
func TestWriteFile_CleansUpTempOnError(t *testing.T) {
	parent := t.TempDir()
	// Point WriteFile at a path whose immediate parent ("missing") does NOT
	// exist. os.CreateTemp("<parent>/missing", ...) fails with ENOENT.
	target := filepath.Join(parent, "missing", "file.txt")

	err := WriteFile(target, []byte("doesn't matter"), 0o644)
	if err == nil {
		t.Fatal("expected error when parent dir missing, got nil")
	}

	// Verify the existing TempDir has no leaked .tmp- residue. (The
	// nonexistent subdir trivially has none.)
	entries, readErr := os.ReadDir(parent)
	if readErr != nil {
		t.Fatalf("ReadDir(%s): %v", parent, readErr)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Errorf("leaked temp file in parent dir: %s", e.Name())
		}
	}

	// And confirm the failing target itself does not exist.
	if _, statErr := os.Stat(target); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("target should not exist after failed WriteFile, got stat err: %v", statErr)
	}
}

// TestWriteFile_PreservesPermissions verifies that the requested mode lands on
// the final target file. The check uses Mode().Perm() to mask off any OS bits
// the stdlib might attach.
func TestWriteFile_PreservesPermissions(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "perm.txt")
	const want = os.FileMode(0o600)

	if err := WriteFile(target, []byte("perms"), want); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("Stat after WriteFile: %v", err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("permission mismatch: got %o, want %o", got, want)
	}
}

// TestWriteFile_RenameFailsWhenTargetIsDirectory exercises the rename-failure
// branch and the deferred cleanup body together. We pre-create a directory at
// the target path; os.CreateTemp + Write + Sync + Chmod + Close all succeed on
// the sibling temp file, then os.Rename(tmp, dir) fails with EISDIR/EEXIST on
// POSIX (rename(2) refuses to replace a directory with a regular file). The
// failure return runs before success=true, so the deferred os.Remove(tmpName)
// fires and the temp file must be gone from the parent directory.
//
// This is the ONE error branch reliably triggerable from pure Go without
// filesystem-injection scaffolding (the Write / Sync / Chmod / Close branches
// require a broken *os.File which the stdlib does not expose), so this test
// alone covers the post-CreateTemp deferred-cleanup path plus the rename-error
// wrapper at atomic.go:85-87.
func TestWriteFile_RenameFailsWhenTargetIsDirectory(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "blocker")

	// Pre-create a directory at the target path. Putting a child file inside
	// guarantees non-empty so platforms that special-case empty-dir rename
	// still surface the error.
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("seed mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "child"), []byte("x"), 0o644); err != nil {
		t.Fatalf("seed child write: %v", err)
	}

	err := WriteFile(target, []byte("payload"), 0o644)
	if err == nil {
		t.Fatal("expected error when target is a directory, got nil")
	}

	// The deferred cleanup must have removed the temp file. Read the parent
	// dir; the only entry should be the blocker directory itself, plus its
	// child file is read separately. No .tmp-* residue.
	entries, readErr := os.ReadDir(dir)
	if readErr != nil {
		t.Fatalf("ReadDir(%s): %v", dir, readErr)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Errorf("leaked temp file in parent dir: %s", e.Name())
		}
	}

	// And confirm the blocker directory itself is intact — rename(2) MUST
	// have left it untouched on failure (atomic-or-nothing invariant).
	info, statErr := os.Stat(target)
	if statErr != nil {
		t.Fatalf("Stat(blocker) after failed WriteFile: %v", statErr)
	}
	if !info.IsDir() {
		t.Errorf("blocker should still be a directory after failed WriteFile, got mode %v", info.Mode())
	}
}
