package cli_codex

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// renderExecpolicyRules generates the execpolicy rules content as a string.
// It emits the 28-verb git-mutation floor (orchestrator is sole committer)
// plus tokenized non-git bash-deny patterns from the gate spec.
//
// Format: one prefix_rule per line, pattern array quoted, decision="forbidden".
//
// Git verbs (28, per dev ruling 2026-05-26):
//   - Oracle's 24: commit, push, add, reset, rebase, merge, checkout, branch, tag,
//     stash, restore, cherry-pick, am, clean, switch, rm, mv, update-ref, gc,
//     prune, worktree, submodule, init, clone
//   - Extended 4: fetch, pull, remote, apply
//
// Non-git patterns: whitespace-tokenized, skip entries prefixed with "git".
func renderExecpolicyRules(bashDeny []string) string {
	var out strings.Builder

	// 28-verb git floor in oracle order (24 + 4).
	gitVerbs := []string{
		"commit", "push", "add", "reset", "rebase", "merge", "checkout",
		"branch", "tag", "stash", "restore", "cherry-pick", "am", "clean",
		"switch", "rm", "mv", "update-ref", "gc", "prune", "worktree",
		"submodule", "init", "clone", "fetch", "pull", "remote", "apply",
	}

	for _, verb := range gitVerbs {
		fmt.Fprintf(&out, `prefix_rule(pattern=["git", "%s"], decision="forbidden")`+"\n", verb)
	}

	// Non-git bash-deny patterns: tokenize on whitespace, skip git-prefixed.
	for _, pat := range bashDeny {
		pat = strings.TrimSpace(pat)
		if pat == "" {
			continue
		}
		// Skip patterns that start with "git" (covered by floor).
		if strings.HasPrefix(pat, "git") && (len(pat) == 3 || pat[3] == ' ') {
			continue
		}

		// Tokenize on whitespace.
		toks := strings.Fields(pat)
		if len(toks) == 0 {
			continue
		}

		// Build quoted token list.
		quotedToks := make([]string, len(toks))
		for i, tok := range toks {
			quotedToks[i] = fmt.Sprintf("%q", tok)
		}
		tokStr := strings.Join(quotedToks, ", ")

		fmt.Fprintf(&out, "prefix_rule(pattern=[%s], decision=\"forbidden\")\n", tokStr)
	}

	return out.String()
}

// writeExecpolicyRules writes the rendered rules to $codexHome/rules/default.rules.
// It creates the rules directory if it does not exist.
func writeExecpolicyRules(codexHome string, bashDeny []string) error {
	rulesDir := filepath.Join(codexHome, "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		return fmt.Errorf("cli_codex: create rules dir: %w", err)
	}

	rulesFile := filepath.Join(rulesDir, "default.rules")
	content := renderExecpolicyRules(bashDeny)

	if err := os.WriteFile(rulesFile, []byte(content), 0o644); err != nil {
		return fmt.Errorf("cli_codex: write rules file: %w", err)
	}

	return nil
}
