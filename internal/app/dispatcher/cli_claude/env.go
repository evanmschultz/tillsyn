package cli_claude

import (
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/evanmschultz/tillsyn/internal/app/dispatcher"
)

// ErrMissingRequiredEnv is returned by assembleEnv when a name listed in
// BindingResolved.Env has no value in the orchestrator's environment
// (os.Getenv returns ""). Per F.7.17 P5 missing-env failures route to
// pre-lock so the dispatcher does not acquire any spawn lock against a
// doomed binding. Callers detect via errors.Is.
var ErrMissingRequiredEnv = errors.New("cli_claude: required env var unset in orchestrator process")

// closedBaselineEnvNames is the F.7.17 L6 + REV-2 closed POSIX baseline of
// environment-variable names every claude spawn inherits. The list is two
// concatenated groups:
//
//   - process basics: PATH, HOME, USER, LANG, LC_ALL, TZ, TMPDIR,
//     XDG_CONFIG_HOME, XDG_CACHE_HOME (9 names, F.7.17 L6).
//   - network conventions: HTTP_PROXY, HTTPS_PROXY, NO_PROXY, http_proxy,
//     https_proxy, no_proxy, SSL_CERT_FILE, SSL_CERT_DIR, CURL_CA_BUNDLE
//     (9 names, REV-2 expansion).
//
// Both upper- and lower-case proxy variants are listed because POSIX
// software conventions split: curl honors lowercase; many language SDKs
// honor uppercase. Listing both keeps adapter-side env handling
// SDK-agnostic.
//
// Names whose os.Getenv lookup is empty are OMITTED from the resulting
// cmd.Env (we do not emit `NAME=`); only the binding's own required Env
// fails loud on absence.
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
// returned slice is the COMPLETE environment passed to the spawned claude
// process — os.Environ() is NOT inherited. Tests assert orchestrator-only
// env vars (e.g. AWS_ACCESS_KEY_ID, sentinel values) do NOT leak through.
//
// The order is deterministic for testability: closed baseline names in
// declaration order first, then binding.Env names sorted for stable
// snapshotting. Duplicate baseline + binding entries collapse to one
// emission (binding.Env values WIN if a name appears in both — the
// binding's allow-list is per-binding-explicit, so the binding's Getenv
// value is authoritative).
//
// Returns ErrMissingRequiredEnv (wrapped with the offending name) when any
// binding.Env name has no value in the orchestrator process. Does NOT
// fail when a closed-baseline name is unset — those degrade gracefully
// (omitted from cmd.Env).
func assembleEnv(binding dispatcher.BindingResolved) ([]string, error) {
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
	emitted := make(map[string]string, len(closedBaselineEnvNames)+len(binding.Env))

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

	// Resolve closed baseline. Empty values (name unset) are skipped — we
	// do not emit `NAME=` for absent baseline vars. Binding-supplied
	// names take precedence: if the binding already populated the name,
	// don't overwrite (covers the case where adopter declared PATH in
	// binding.Env explicitly).
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
	// then any binding-only names (those NOT in the baseline) in sorted
	// order. The sort is purely for test determinism — exec consumes the
	// slice as an unordered set.
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
