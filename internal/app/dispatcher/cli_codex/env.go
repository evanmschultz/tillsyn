package cli_codex

import (
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/evanmschultz/tillsyn/internal/app/dispatcher"
)

// ErrMissingRequiredEnv is returned by assembleEnv when a name listed in
// BindingResolved.Env has no value in the orchestrator's environment
// (os.LookupEnv returns ok=false). Per F.7.17 P5 missing-env failures route
// to pre-lock so the dispatcher does not acquire any spawn lock against a
// doomed binding. Callers detect via errors.Is.
var ErrMissingRequiredEnv = errors.New("cli_codex: required env var unset in orchestrator process")

// defenseInDepthEnvLiterals is the set of literal (name, value) pairs
// injected unconditionally into every codex spawn's cmd.Env. Unlike
// closedBaselineEnvNames (which is a name-only allowlist resolved via
// os.LookupEnv from the orchestrator process), these entries carry their
// values inline — they are emitted regardless of whether the orchestrator
// has them set, unset, or set to a different value.
//
// For codex we emit only a minimal set: the claude-specific
// CLAUDE_CODE_* / DISABLE_AUTOUPDATER flags from cli_claude are not
// applicable to codex. DISABLE_TELEMETRY is a reasonable hygiene flag for
// ephemeral spawned processes.
//
// Override semantics: binding.Env wins over these literals. The injection
// loop in assembleEnv runs AFTER binding.Env resolution and respects the
// same `alreadySet` skip pattern used at the closed-baseline loop — so a
// binding that explicitly names one of these env vars takes precedence.
// Net precedence: binding.Env > defense-in-depth literals > closed-baseline.
var defenseInDepthEnvLiterals = []struct {
	Name  string
	Value string
}{
	// Privacy + reproducibility for spawned processes.
	{Name: "DISABLE_TELEMETRY", Value: "1"},
}

// closedBaselineEnvNames is the F.7.17 L6 + REV-2 closed POSIX baseline of
// environment-variable names every codex spawn inherits. The list mirrors
// cli_claude's baseline exactly — it is the process-operation + network-
// convention set applicable to any CLI spawn, not claude-specific.
//
// Two concatenated groups:
//
//   - process basics: PATH, HOME, USER, LANG, LC_ALL, TZ, TMPDIR,
//     XDG_CONFIG_HOME, XDG_CACHE_HOME (9 names, F.7.17 L6).
//   - network conventions: HTTP_PROXY, HTTPS_PROXY, NO_PROXY, http_proxy,
//     https_proxy, no_proxy, SSL_CERT_FILE, SSL_CERT_DIR, CURL_CA_BUNDLE
//     (9 names, REV-2 expansion).
//
// Codex uses OPENAI_API_KEY (not ANTHROPIC_API_KEY). The API key is NOT in
// this baseline — it is in binding.Env for the bindings that need it, so it
// fails loud on absence per F.7.17 P5. This is the same pattern cli_claude
// uses for ANTHROPIC_API_KEY.
//
// Names whose os.LookupEnv lookup returns ok=false are OMITTED from the
// resulting cmd.Env (we do not emit `NAME=`); only the binding's own required
// Env fails loud on absence.
var closedBaselineEnvNames = []string{
	// Process basics (L6).
	"PATH",
	"HOME",
	"USER",
	"LANG",
	"LC_ALL",
	"TZ",
	"TMPDIR",
	"XDG_CONFIG_HOME",
	"XDG_CACHE_HOME",
	// Network conventions (REV-2).
	"HTTP_PROXY",
	"HTTPS_PROXY",
	"NO_PROXY",
	"http_proxy",
	"https_proxy",
	"no_proxy",
	"SSL_CERT_FILE",
	"SSL_CERT_DIR",
	"CURL_CA_BUNDLE",
}

// assembleEnv builds the cmd.Env slice for one spawn. Per F.7.17 L8 the
// returned slice is the COMPLETE environment passed to the spawned codex
// process — os.Environ() is NOT inherited. Tests assert orchestrator-only
// env vars do NOT leak through.
//
// Precedence (full contract): binding.Env (via os.LookupEnv) > envSetLiterals >
// defense-in-depth > closed POSIX baseline. On key collision, the earliest
// source in this chain wins via the alreadySet skip-guard pattern.
//
// The output slice order is deterministic for testability: closed baseline
// names in declaration order first, then defense-in-depth literals in
// declaration order, then envSetLiterals keys sorted, then binding-only names
// sorted.
//
// envSetLiterals (optional, may be nil) carries name-value pairs to inject
// unconditionally as literal values. Unlike binding.Env (resolved via
// os.LookupEnv), these carry their values inline and do not fail if the
// orchestrator process lacks a shell variable — they are emitted directly.
//
// Returns ErrMissingRequiredEnv (wrapped with the offending name) when any
// binding.Env name has no value in the orchestrator process. Does NOT fail
// when a closed-baseline name is unset — those degrade gracefully (omitted
// from cmd.Env).
func assembleEnv(binding dispatcher.BindingResolved, envSetLiterals map[string]string) ([]string, error) {
	// Track which names came from the binding's allow-list so we can route
	// missing values to ErrMissingRequiredEnv (closed-baseline absent values
	// silently omit, per docs above).
	bindingNames := make(map[string]struct{}, len(binding.Env))
	for _, name := range binding.Env {
		bindingNames[name] = struct{}{}
	}

	// emitted tracks names already added to env so duplicates collapse to
	// one entry. The string-key form (NAME=value) is what cmd.Env wants
	// directly; the map's purpose is dedup-by-name only.
	capacity := len(closedBaselineEnvNames) + len(binding.Env)
	if envSetLiterals != nil {
		capacity += len(envSetLiterals)
	}
	emitted := make(map[string]string, capacity)

	// Resolve every binding.Env name first so the missing-required error
	// surfaces before we do baseline work. Missing required env is the
	// load-bearing failure mode the dispatcher routes to pre-lock per P5;
	// surfacing it eagerly avoids any chance of partial state leaking
	// into the returned slice on the error path.
	for _, name := range binding.Env {
		val, ok := os.LookupEnv(name)
		if !ok {
			return nil, fmt.Errorf("%w: name=%q", ErrMissingRequiredEnv, name)
		}
		emitted[name] = val
	}

	// Inject envSetLiterals from binding.EnvSet (option D flow).
	// These carry their values inline — no os.LookupEnv. They are emitted
	// unconditionally when present. Precedence: binding.Env wins (explicit
	// per-binding allow-list is most authoritative); envSetLiterals win
	// over defense-in-depth + closed-baseline. Net precedence chain:
	// binding.Env > envSetLiterals > defense-in-depth > closed-baseline.
	if envSetLiterals != nil {
		for name, val := range envSetLiterals {
			if _, alreadySet := emitted[name]; alreadySet {
				continue
			}
			emitted[name] = val
		}
	}

	// Inject defense-in-depth literals. Unlike the closed-baseline loop below,
	// these carry their values inline — no os.LookupEnv. They are emitted
	// unconditionally. Precedence: binding.Env (above) wins over defense-in-
	// depth, and envSetLiterals (above) also win over defense-in-depth, so the
	// `alreadySet` skip mirrors the closed-baseline pattern. Net precedence
	// chain: binding.Env > envSetLiterals > defense-in-depth > closed-baseline.
	for _, lit := range defenseInDepthEnvLiterals {
		if _, alreadySet := emitted[lit.Name]; alreadySet {
			continue
		}
		emitted[lit.Name] = lit.Value
	}

	// Resolve closed baseline. Empty values (name unset) are skipped — we
	// do not emit `NAME=` for absent baseline vars. Binding-supplied
	// names + envSetLiterals + defense-in-depth literals take precedence:
	// if any already populated the name, don't overwrite (covers the case
	// where adopter declared PATH in binding.Env explicitly, or a
	// defense-in-depth name collides with a baseline name — though today's
	// baseline and defense-in-depth sets are disjoint).
	for _, name := range closedBaselineEnvNames {
		if _, alreadySet := emitted[name]; alreadySet {
			continue
		}
		val, ok := os.LookupEnv(name)
		if !ok {
			continue
		}
		emitted[name] = val
	}

	// Build the ordered slice: baseline names in their declared order,
	// then defense-in-depth literals in declaration order, then
	// envSetLiterals in sorted order, then any binding-only names (those
	// NOT in any prior set) in sorted order. The sort is purely for test
	// determinism — exec consumes the slice as an unordered set.
	out := make([]string, 0, len(emitted))
	seen := make(map[string]struct{}, len(emitted))
	for _, name := range closedBaselineEnvNames {
		val, ok := emitted[name]
		if !ok {
			continue
		}
		out = append(out, name+"="+val)
		seen[name] = struct{}{}
	}
	for _, lit := range defenseInDepthEnvLiterals {
		if _, alreadyEmitted := seen[lit.Name]; alreadyEmitted {
			continue
		}
		val, ok := emitted[lit.Name]
		if !ok {
			continue
		}
		out = append(out, lit.Name+"="+val)
		seen[lit.Name] = struct{}{}
	}

	// Emit envSetLiterals in sorted order for deterministic test output.
	envSetKeys := make([]string, 0, len(envSetLiterals))
	for name := range envSetLiterals {
		if _, alreadyEmitted := seen[name]; alreadyEmitted {
			continue
		}
		envSetKeys = append(envSetKeys, name)
	}
	sort.Strings(envSetKeys)
	for _, name := range envSetKeys {
		out = append(out, name+"="+emitted[name])
		seen[name] = struct{}{}
	}

	bindingOnly := make([]string, 0, len(binding.Env))
	for name := range bindingNames {
		if _, alreadyEmitted := seen[name]; alreadyEmitted {
			continue
		}
		bindingOnly = append(bindingOnly, name)
	}
	sort.Strings(bindingOnly)
	for _, name := range bindingOnly {
		out = append(out, name+"="+emitted[name])
	}

	return out, nil
}
