package pretoolgate

import (
	"regexp"
	"strings"
)

// bashForbidden checks whether a command matches any of the deny patterns,
// returning (pattern_that_matched, true) if a hit is found, ("", false) otherwise.
// It ports the proven hook's _bash_forbidden (ta_action_gate.py lines 204-227).
//
// The logic is two-pass:
//  1. Git-subcommand pass: extract git verbs from patterns (e.g. "git commit" -> "commit"),
//     then split the command on shell separators [;&|\n]+ and check each segment's git verb
//     against the extracted verbs. Defeats evasions like `git -C dir commit`.
//  2. Generic word-boundary pass: for remaining (non-git) patterns like "mage install",
//     match with word-boundary checks: the pattern must NOT be preceded/followed by word chars
//     or hyphens (Go regexp has no lookbehind/lookahead, so we use FindAllStringIndex
//     and manually verify the surrounding characters).
func bashForbidden(command string, denyPatterns []string) (string, bool) {
	cmd := strings.TrimSpace(command)

	// Pass 1: Extract git verbs from deny patterns and check the command.
	gitVerbs := gitVerbsFromDeny(denyPatterns)
	if len(gitVerbs) > 0 {
		segPat := regexp.MustCompile(`[;&|\n]+`)
		segments := segPat.Split(cmd, -1)
		for _, seg := range segments {
			tokens := strings.Fields(seg)
			if len(tokens) == 0 {
				continue
			}
			verb, found := gitSubcommand(tokens)
			if found && verb != "" {
				if _, ok := gitVerbs[verb]; ok {
					return "git " + verb, true
				}
			}
		}
	}

	// Pass 2: Generic word-boundary pass for non-git patterns.
	// RE2 has no lookbehind/lookahead, so we use FindAllStringIndex to locate
	// the pattern and then manually verify the surrounding characters are not word chars.
	isWordChar := func(ch byte) bool {
		return (ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '_' || ch == '-'
	}

	for _, pat := range denyPatterns {
		pat = strings.TrimSpace(pat)
		if pat == "" {
			continue
		}
		// Skip git patterns (they were handled in pass 1).
		if strings.HasPrefix(pat, "git ") {
			continue
		}
		// Build a regex that matches the escaped pattern.
		escapedPat := regexp.QuoteMeta(pat)
		searchPat := regexp.MustCompile(escapedPat)
		matches := searchPat.FindAllStringIndex(cmd, -1)
		for _, matchIdx := range matches {
			// matchIdx = [start, end]
			start, end := matchIdx[0], matchIdx[1]
			// Check char before start — must NOT be a word char.
			if start > 0 && isWordChar(cmd[start-1]) {
				continue // boundary violated, skip this match.
			}
			// Check char after end — must NOT be a word char.
			if end < len(cmd) && isWordChar(cmd[end]) {
				continue // boundary violated, skip this match.
			}
			// Valid match: pattern surrounded by non-word-chars (or boundaries).
			return pat, true
		}
	}

	return "", false
}

// bashWriteVector checks whether a command contains any shell file-write or
// file-mutation vector, returning (description, true) if a hit is found,
// ("", false) otherwise. It ports the proven hook's _bash_write_vector
// (ta_action_gate.py lines 230-257).
//
// A write vector is any shell command or redirection that can mutate files,
// including: output redirection (> >>), tee, sed -i, perl/ruby -i, dd of=,
// interpreters (python3, node, etc.), and file-mutating commands (cp, mv, rm, etc.).
//
// Note: output redirection to /dev/null or >& (stderr redirect) are NOT counted
// as write vectors, since they don't mutate files on disk.
func bashWriteVector(command string) (string, bool) {
	cmd := strings.TrimSpace(command)
	if cmd == "" {
		return "", false
	}

	// Helper: check if a pattern matches with word-boundary checks.
	// RE2 has no lookahead/lookbehind, so we match and manually verify boundaries.
	// Returns the index of the first valid match, or -1 if none.
	findWithBoundary := func(pat string) int {
		escapedPat := regexp.QuoteMeta(pat)
		searchPat := regexp.MustCompile(escapedPat)
		matches := searchPat.FindAllStringIndex(cmd, -1)
		isWordChar := func(ch byte) bool {
			return (ch >= 'a' && ch <= 'z') ||
				(ch >= 'A' && ch <= 'Z') ||
				(ch >= '0' && ch <= '9') ||
				ch == '_' || ch == '-'
		}
		for _, matchIdx := range matches {
			start, end := matchIdx[0], matchIdx[1]
			// Check leading boundary.
			if start > 0 && isWordChar(cmd[start-1]) {
				continue
			}
			// Check trailing boundary.
			if end < len(cmd) && isWordChar(cmd[end]) {
				continue
			}
			return start
		}
		return -1
	}

	// Helper: check if index is preceded by a command-start marker.
	isCmdStart := func(idx int) bool {
		if idx < 0 {
			return false
		}
		if idx == 0 {
			return true
		}
		prevCh := cmd[idx-1]
		return prevCh == ' ' || prevCh == '\t' || prevCh == '/' || prevCh == '|' ||
			prevCh == ';' || prevCh == '&' || prevCh == '(' || prevCh == '`'
	}

	// Vector 1: Output redirection (> or >>) that is NOT to /dev/null or >&.
	// Python regex: >>?\s*(?!&|/dev/null\b)\S
	// Port: match >>? + whitespace, then check that the next token is not & or /dev/null.
	outputRedirPat := regexp.MustCompile(`>>?\s*`)
	matches := outputRedirPat.FindAllStringIndex(cmd, -1)
	for _, matchIdx := range matches {
		end := matchIdx[1]
		if end >= len(cmd) {
			continue // redirect with nothing after; skip.
		}
		// Check if the next char is & (stderr redirect) — exclude it.
		if cmd[end] == '&' {
			continue
		}
		// Check if the next token starts with /dev/null — exclude it.
		nextToken := strings.FieldsFunc(cmd[end:], func(r rune) bool {
			return r == ' ' || r == '\t' || r == '|' || r == ';' || r == '&' || r == '\n'
		})
		if len(nextToken) > 0 && strings.HasPrefix(nextToken[0], "/dev/null") {
			continue
		}
		// Valid output redirection hit.
		return "output redirection (> / >>)", true
	}

	// Vector 2: tee
	if idx := findWithBoundary("tee"); idx >= 0 && isCmdStart(idx) {
		return "tee", true
	}

	// Vector 3: sed -i
	// Pattern: sed followed by optional args, then -i flag before pipe/semicolon.
	// Simplified: just look for "sed " followed by "-i" somewhere before a pipe.
	if idx := strings.Index(cmd, "sed "); idx >= 0 && isCmdStart(idx) {
		// Found sed; check if -i appears before the next pipe/semicolon/ampersand.
		afterSed := cmd[idx+4:] // skip "sed "
		for i, ch := range afterSed {
			if ch == '|' || ch == ';' || ch == '&' {
				break // stop at segment boundary
			}
			if strings.HasPrefix(afterSed[i:], "-i") {
				return "sed -i (in-place edit)", true
			}
		}
	}

	// Vector 4: perl/ruby -i
	// Pattern: similar to sed but for perl/ruby.
	for _, pref := range []string{"perl ", "ruby "} {
		if idx := strings.Index(cmd, pref); idx >= 0 && isCmdStart(idx) {
			// Found perl/ruby; check if -i appears before the next pipe/semicolon/ampersand.
			afterCmd := cmd[idx+len(pref):]
			for i, ch := range afterCmd {
				if ch == '|' || ch == ';' || ch == '&' {
					break // stop at segment boundary
				}
				if strings.HasPrefix(afterCmd[i:], "-i") {
					return "perl/ruby -i (in-place edit)", true
				}
			}
		}
	}

	// Vector 5: dd of=
	ddOf := regexp.MustCompile(`dd\s+[^|;&\n]*\bof=`)
	if m := ddOf.FindStringIndex(cmd); m != nil && isCmdStart(m[0]) {
		return "dd of=", true
	}

	// Vector 6: interpreters (python3, python, node, deno, bun, ruby, perl, osascript, php)
	interpreters := []string{"python3", "python", "node", "deno", "bun", "ruby", "perl", "osascript", "php"}
	for _, interp := range interpreters {
		if idx := findWithBoundary(interp); idx >= 0 && isCmdStart(idx) {
			return "interpreter (can write files)", true
		}
	}

	// Vector 7: file-mutating commands
	mutators := []string{"cp", "mv", "install", "ln", "truncate", "touch", "mkdir", "rmdir", "rm", "chmod", "chown", "dd"}
	for _, mut := range mutators {
		if idx := findWithBoundary(mut); idx >= 0 && isCmdStart(idx) {
			return "file-mutating command", true
		}
	}

	return "", false
}
