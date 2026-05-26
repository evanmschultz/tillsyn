package pretoolgate

import (
	"path"
	"path/filepath"
)

// normPath normalizes a file path relative to a working directory.
// Ports Python _norm (lines 151-156 of ta_action_gate.py):
//   - empty string → empty string
//   - relative path → joined with cwd, then cleaned
//   - absolute path → cleaned directly
//
// Note on Go stdlib mapping:
//   - Go filepath.Join + filepath.Clean ≈ Python os.path.join + os.path.normpath
//   - We use filepath (not path) for normalization because filepath is OS-aware:
//     it correctly handles absolute-path detection and cross-platform separators.
//     On Linux/macOS (the test substrate), filepath.Clean matches POSIX semantics.
func normPath(p, cwd string) string {
	if p == "" {
		return ""
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(cwd, p)
	}
	return filepath.Clean(p)
}

// editAllowed reports whether filePath is in the allowed list.
// Ports Python _edit_allowed (lines 159-169 of ta_action_gate.py).
//
// For each entry in allowed:
//  1. Normalize both target and entry (relative to cwd)
//  2. Return true if exact match (normalized or raw)
//  3. Return true if either glob pattern matches
//
// Go stdlib mapping for glob:
//   - Go path.Match is the functional equivalent of Python fnmatch.fnmatch.
//     fnmatch is //-agnostic glob matching; path.Match provides the same semantics.
//     (Note: path.Match is for forward-slash paths, but on Linux/macOS it matches
//     filepath behavior because both use / as the separator.)
//   - If path.Match reports an error (e.g., malformed pattern), treat it as no-match.
//
// Returns false if allowed is empty or no entry matches. Empty allowed denotes
// a deny-all gate (e.g., read-only roles).
func editAllowed(filePath string, allowed []string, cwd string) bool {
	if len(allowed) == 0 {
		return false
	}

	target := normPath(filePath, cwd)

	for _, entry := range allowed {
		normEntry := normPath(entry, cwd)

		// Exact match on normalized paths
		if target == normEntry {
			return true
		}

		// Glob match on normalized paths: normEntry as pattern, target as candidate
		if m, _ := path.Match(normEntry, target); m {
			return true
		}

		// Glob match on raw forms: entry as pattern, filePath as candidate
		if m, _ := path.Match(entry, filePath); m {
			return true
		}
	}

	return false
}
