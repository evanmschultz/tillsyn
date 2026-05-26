package pretoolgate

import (
	"regexp"
	"strings"
)

// gitSubcommand parses a segment of shell tokens (after `strings.Fields`)
// to extract the git subcommand verb if one is present. It skips leading
// VAR=val environment assignments, then walks tokens to find `git` (by
// basename, so /usr/bin/git matches). Once at git, it consumes global
// options (e.g. -C, --git-dir, -c) that take arguments, skips other flags,
// and returns the first non-flag token as the verb. Returns ("", false) if
// no git subcommand is found.
//
// This defeats evasions like:
//   - `git -C /repo commit` (global flag with arg)
//   - `/usr/bin/git push` (path-prefixed)
//   - `FOO=1 git commit` (env-assignment prefix)
//   - `git --git-dir=x commit` (inline flag assignment)
func gitSubcommand(segTokens []string) (string, bool) {
	globalOptsWithArg := map[string]struct{}{
		"-C":             {},
		"--git-dir":      {},
		"--work-tree":    {},
		"--namespace":    {},
		"-c":             {},
		"--exec-path":    {},
		"--super-prefix": {},
	}

	n := len(segTokens)
	i := 0

	// Skip leading VAR=val env assignments.
	envPat := regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)
	for i < n && envPat.MatchString(segTokens[i]) {
		i++
	}

	// Find git token (by basename).
	for i < n {
		parts := strings.Split(segTokens[i], "/")
		if parts[len(parts)-1] == "git" {
			j := i + 1
			// Consume global options that take arguments, skip other flags.
			for j < n {
				tk := segTokens[j]
				if _, ok := globalOptsWithArg[tk]; ok {
					j += 2 // option + arg
					continue
				}
				if strings.HasPrefix(tk, "-") {
					j++ // other flag (including --git-dir=... inline)
					continue
				}
				return tk, true // first non-flag is the subcommand
			}
			return "", false // no subcommand after git
		}
		i++
	}
	return "", false // git not found
}

// gitVerbsFromDeny extracts git subcommand verbs from deny patterns by
// matching the pattern `^git\s+(\S+)$` and collecting the captured verb.
// Non-git patterns (e.g. "mage install", "go get") are excluded.
// Returns a map (set-like) of the extracted verbs.
func gitVerbsFromDeny(denyPatterns []string) map[string]struct{} {
	verbs := make(map[string]struct{})
	patPat := regexp.MustCompile(`^git\s+(\S+)$`)
	for _, pat := range denyPatterns {
		m := patPat.FindStringSubmatch(strings.TrimSpace(pat))
		if m != nil {
			verbs[m[1]] = struct{}{}
		}
	}
	return verbs
}

// gitMutation is the hardcoded, ORCHESTRATOR-INDEPENDENT git-mutation floor.
// It checks whether a command attempts any git mutation (commit, push, etc.),
// returning (verb_name, true) if a mutation is detected, ("", false) otherwise.
// This fires REGARDLESS of whether the orchestrator listed git in bash_deny,
// preventing the "forgetful orchestrator" failure mode.
//
// The frozenset includes read-ish operations (fetch, clone, pull, remote)
// because the dev signed off on them as mutations (they modify .git/config, etc.).
// Read-only operations (diff, status, log, show, blame, rev-parse) are NOT
// in the set and remain allowed.
//
// The function splits the command on shell separators [;&|\n]+ (to handle
// chained commands like `git diff | git push`) and applies gitSubcommand
// per segment to extract and check the verb against the hardcoded set.
func gitMutation(command string) (string, bool) {
	segPat := regexp.MustCompile(`[;&|\n]+`)
	segments := segPat.Split(command, -1)

	for _, seg := range segments {
		tokens := strings.Fields(seg)
		if len(tokens) == 0 {
			continue
		}
		verb, found := gitSubcommand(tokens)
		if found && verb != "" {
			if _, isMutation := gitMutationVerbs[verb]; isMutation {
				return "git " + verb, true
			}
		}
	}
	return "", false
}

// gitMutationVerbs is the hardcoded, ORCHESTRATOR-INDEPENDENT baseline of
// git subcommands that are forbidden to scoped agents. This set is ported
// verbatim from the proven hook (ta_action_gate.py, fix #5), ensuring agents
// can NEVER commit/push/etc. regardless of whether the orchestrator remembered
// to list git in bash_deny. The set is frozen at 28 verbs.
//
// Verbs: commit, push, add, reset, rebase, merge, checkout, branch, tag,
// stash, restore, cherry-pick, am, clean, switch, rm, mv, update-ref, gc,
// prune, worktree, submodule, init, clone, fetch, pull, remote, apply.
var gitMutationVerbs = map[string]struct{}{
	"commit":      {},
	"push":        {},
	"add":         {},
	"reset":       {},
	"rebase":      {},
	"merge":       {},
	"checkout":    {},
	"branch":      {},
	"tag":         {},
	"stash":       {},
	"restore":     {},
	"cherry-pick": {},
	"am":          {},
	"clean":       {},
	"switch":      {},
	"rm":          {},
	"mv":          {},
	"update-ref":  {},
	"gc":          {},
	"prune":       {},
	"worktree":    {},
	"submodule":   {},
	"init":        {},
	"clone":       {},
	"fetch":       {},
	"pull":        {},
	"remote":      {},
	"apply":       {},
}
