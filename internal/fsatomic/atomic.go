// Package fsatomic provides minimal atomic file-write helpers built on the
// textbook write-temp-in-same-dir + sync + rename pattern.
//
// Design.
//
// WriteFile writes data to a temp file in the *same directory* as the target,
// fsyncs the temp file, then renames it over the target. The same-directory
// requirement is load-bearing: os.Rename is only atomic on POSIX when source
// and destination live on the same filesystem, and placing the temp inside
// the target's parent directory guarantees that condition without probing
// mount points.
//
// On any failure between create and rename, the partially-written temp file
// is removed via a deferred cleanup guarded by a success flag, so the parent
// directory never accumulates ".tmp-*" residue from interrupted writes. After
// a successful rename, the temp filename no longer points at the file and
// cleanup is a no-op.
//
// Scope.
//
// This package owns the smallest surface W2.D5 needs. No staged-write struct,
// no parent-dir fsync helper, no rename-only helper — add those when a
// concrete caller in this repo needs them. Future migration to a shared
// hylla-utility location (see SKETCH.md §9.6) lifts this implementation
// directly without contract change.
package fsatomic

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteFile atomically writes data to the named file with the given
// permissions. The write is performed via a temporary file in the same
// directory followed by os.Rename, so the target is either fully present
// with the new contents or untouched — never observed half-written by a
// concurrent reader on POSIX filesystems.
//
// Permissions are applied to the temp file before rename so the final mode
// is correct at the moment the file becomes visible at the target path.
//
// On error, no temp file is left behind. WriteFile overwrites an existing
// target by default (matching os.WriteFile's contract); callers that want
// skip-if-present semantics must check os.Stat first.
func WriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	base := filepath.Base(path)

	f, err := os.CreateTemp(dir, base+".tmp-*")
	if err != nil {
		return fmt.Errorf("fsatomic: create temp in %s: %w", dir, err)
	}

	// Track whether we successfully renamed; on any early return before that
	// point the deferred cleanup removes the temp file. After rename the
	// filename no longer points at a file, so cleanup is a no-op.
	success := false
	tmpName := f.Name()
	defer func() {
		if !success {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return fmt.Errorf("fsatomic: write %s: %w", tmpName, err)
	}

	if err := f.Sync(); err != nil {
		_ = f.Close()
		return fmt.Errorf("fsatomic: sync %s: %w", tmpName, err)
	}

	if err := f.Chmod(perm); err != nil {
		_ = f.Close()
		return fmt.Errorf("fsatomic: chmod %s: %w", tmpName, err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("fsatomic: close %s: %w", tmpName, err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("fsatomic: rename %s -> %s: %w", tmpName, path, err)
	}

	success = true
	return nil
}
