package dispatcher

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// RequiredPluginsForProject is the package-level hook BuildSpawnCommand
// invokes to resolve the per-project required-plugin list before the
// pre-dispatch pre-flight check fires. A nil hook (the default) means
// "no required plugins" and CheckRequiredPlugins short-circuits before
// invoking the lister; callers wire the hook at process boot once the
// data feed is plumbed.
//
// Today's wiring story: KindCatalog (the per-project baked snapshot fed
// to BuildSpawnCommand) does NOT carry the [tillsyn] globals — only
// Kinds + AgentBindings. A future droplet that extends KindCatalog with
// a `RequiresPlugins []string` field will populate the hook from
// catalog.RequiresPlugins; until then, the hook is a deliberate seam
// adopters populate at process boot via direct assignment, e.g.:
//
//	dispatcher.RequiredPluginsForProject = func(p domain.Project) []string {
//	    // resolve from project metadata, separate config, etc.
//	    return …
//	}
//
// Concurrency: the hook is read once per BuildSpawnCommand call. Callers
// MUST set it before the first spawn; reassigning under load is unsafe.
var RequiredPluginsForProject func(domain.Project) []string

// pluginPreflightTimeout caps the wall-clock time the production lister
// allows `claude plugin list --json` to run. Per Drop 4c F.7-CORE F.7.6
// acceptance criteria the typical execution is <50ms; a 5-second cap
// surfaces a hung Claude binary or runaway daemon as a structured timeout
// rather than a stalled spawn pipeline.
const pluginPreflightTimeout = 5 * time.Second

// ClaudePluginListEntry is one row of the `claude plugin list --json`
// output. Each installed plugin contributes one entry; missing or
// unrecognized JSON fields decode as the zero value of their type without
// failing the parse.
//
// The four field names mirror the canonical claude plugin-list shape per
// Drop 4c spawn-architecture memory §1 Path B: `id`, `marketplace`,
// `version`, `installPath`. Future claude versions that add fields
// continue to decode cleanly because the dispatcher uses a non-strict
// json.Unmarshal — unknown keys are silently ignored.
type ClaudePluginListEntry struct {
	// ID is the plugin identifier (e.g. "context7", "gopls-lsp"). Required
	// for matching; an entry with an empty ID is skipped by the matcher.
	ID string `json:"id"`

	// Marketplace identifies the catalog source the plugin was installed
	// from (e.g. "claude-plugins-official"). The matcher compares this
	// against the marketplace segment of `<name>@<marketplace>` required
	// entries; bare-name `<name>` requirements ignore Marketplace
	// entirely.
	Marketplace string `json:"marketplace"`

	// Version is the installed plugin version string (e.g. "0.4.1"). The
	// pre-flight check does NOT enforce version constraints today —
	// out-of-scope per F.7.6 acceptance criteria — but the field is
	// captured for forward-compat and dev diagnostics.
	Version string `json:"version"`

	// InstallPath is the absolute filesystem path the plugin was
	// materialized to. Captured for forward-compat and dev diagnostics;
	// not consumed by the matcher.
	InstallPath string `json:"installPath"`
}

// ClaudePluginLister is the package-private test seam between
// CheckRequiredPlugins and the underlying `claude plugin list --json`
// invocation. Production code wires execClaudePluginLister which shells
// out via exec.CommandContext; tests inject a fake that returns a canned
// []ClaudePluginListEntry slice (or a canned error).
//
// The interface is intentionally small — one method, one return shape —
// so test fakes are trivial to author and the production implementation
// has nowhere to hide subtle behavior the seam couldn't reproduce.
type ClaudePluginLister interface {
	// List returns the set of plugins the local `claude` binary reports
	// as installed. Returns a non-nil error when the binary is missing
	// from PATH, the JSON parse fails, or the context expires. An empty
	// slice with nil error is a legitimate "claude installed but no
	// plugins" response and CheckRequiredPlugins treats it that way.
	List(ctx context.Context) ([]ClaudePluginListEntry, error)
}

// ErrMissingRequiredPlugins is the sentinel returned by
// CheckRequiredPlugins when one or more entries from `required` are absent
// from the `claude plugin list --json` output. The wrapped message lists
// every missing entry — NOT only the first — alongside the
// `claude plugin install <missing>` command the dev should run for each.
//
// Callers detect this via errors.Is(err, ErrMissingRequiredPlugins) to
// route the failure to a structured "fix your install" UX without
// reaching for substring scrapes of the error text.
var ErrMissingRequiredPlugins = errors.New("dispatcher: required claude plugins missing")

// ErrClaudeBinaryMissing is the sentinel returned by the production lister
// when the local environment has no `claude` executable on PATH. The
// wrapped message points the dev at the claude install instructions; the
// dispatcher cannot proceed without claude, so this surfaces as a hard
// preflight failure.
//
// Drop 4c F.7-CORE F.7.6 acceptance criteria require the failure to be
// distinguishable from "plugin missing" so callers can branch on the
// remediation path. Callers detect this via errors.Is.
var ErrClaudeBinaryMissing = errors.New("dispatcher: claude binary not found on PATH")

// ErrPluginListUnparseable is the sentinel returned by the production
// lister when `claude plugin list --json` exits cleanly but its stdout
// fails to decode as a JSON array of ClaudePluginListEntry values. The
// wrapped message names the underlying json.Unmarshal error so the dev
// can reason about whether claude's output schema drifted. Callers detect
// this via errors.Is.
var ErrPluginListUnparseable = errors.New("dispatcher: claude plugin list output unparseable")

// CheckRequiredPlugins verifies that every entry in `required` corresponds
// to an installed plugin reported by `lister`. Returns nil when all
// entries are matched (or when `required` is nil/empty — the no-op early
// return); returns ErrMissingRequiredPlugins wrapped with the missing list
// otherwise. Lister-side failures (claude binary missing, JSON parse
// error, context expiry) propagate verbatim.
//
// Matching rules per Drop 4c F.7-CORE F.7.6 entry contract:
//
//   - `<name>` (bare): matches any installed entry whose ID == name.
//     Marketplace is ignored on the installed-entry side.
//   - `<name>@<marketplace>`: matches only an installed entry whose ID ==
//     name AND Marketplace == marketplace. Both segments are required to
//     match exactly (case-sensitive); a partial match (right name, wrong
//     marketplace) does NOT satisfy the requirement and the entry is
//     reported as missing.
//
// The matcher iterates `required` in declaration order and aggregates ALL
// missing entries before returning. This avoids the "fix one, run again,
// see the next" treadmill — the dev sees the full list of plugins to
// install in one error message.
//
// CheckRequiredPlugins skips invoking lister.List entirely when `required`
// is empty. Adopters who depend on no plugins pay no exec cost per spawn.
func CheckRequiredPlugins(ctx context.Context, lister ClaudePluginLister, required []string) error {
	if len(required) == 0 {
		return nil
	}
	if lister == nil {
		return fmt.Errorf("%w: lister is nil", ErrInvalidSpawnInput)
	}
	installed, err := lister.List(ctx)
	if err != nil {
		return fmt.Errorf("dispatcher: list claude plugins: %w", err)
	}
	missing := findMissingPlugins(required, installed)
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("%w: %s", ErrMissingRequiredPlugins, formatMissingPlugins(missing))
}

// findMissingPlugins returns the subset of `required` entries that have no
// corresponding entry in `installed` per the F.7.6 matching rules. The
// return slice preserves the input order of `required` so the failure
// message lists missing entries in the order the template author wrote
// them.
func findMissingPlugins(required []string, installed []ClaudePluginListEntry) []string {
	missing := make([]string, 0, len(required))
	for _, entry := range required {
		if pluginIsInstalled(entry, installed) {
			continue
		}
		missing = append(missing, entry)
	}
	return missing
}

// pluginIsInstalled reports whether `entry` (in `<name>` or
// `<name>@<marketplace>` form) is satisfied by any element of `installed`.
// Bare-name requirements match on ID alone; scoped requirements match on
// (ID, Marketplace) exactly.
func pluginIsInstalled(entry string, installed []ClaudePluginListEntry) bool {
	name, marketplace, scoped := splitPluginEntry(entry)
	for _, row := range installed {
		if row.ID == "" {
			continue
		}
		if row.ID != name {
			continue
		}
		if scoped && row.Marketplace != marketplace {
			continue
		}
		return true
	}
	return false
}

// splitPluginEntry parses `entry` into its name + marketplace + scoped
// flag. The validator at template Load time enforces the well-formedness
// rules (single `@`, both segments non-empty when present), so this
// function does not re-validate — it splits structurally.
func splitPluginEntry(entry string) (name, marketplace string, scoped bool) {
	at := strings.IndexByte(entry, '@')
	if at < 0 {
		return entry, "", false
	}
	return entry[:at], entry[at+1:], true
}

// formatMissingPlugins renders the wrapped message body for
// ErrMissingRequiredPlugins. Each missing entry is rendered as a
// semicolon-separated `<entry> (run: claude plugin install <entry>)`
// fragment so the dev can copy-paste the install command directly. The
// rendering preserves input order.
func formatMissingPlugins(missing []string) string {
	parts := make([]string, 0, len(missing))
	for _, entry := range missing {
		parts = append(parts, fmt.Sprintf("%s (run: claude plugin install %s)", entry, entry))
	}
	return strings.Join(parts, "; ")
}

// execClaudePluginLister is the production ClaudePluginLister
// implementation. It shells out to `claude plugin list --json` via
// exec.CommandContext, captures stdout into a bounded bytes.Buffer, and
// decodes the result as a JSON array of ClaudePluginListEntry values.
//
// Production wiring: a package-private singleton (defaultClaudePluginLister
// below) is the seam tests swap via t.Cleanup-restored assignment. The
// dispatcher RuntimeOptions / cmd/till boot path consumes
// defaultClaudePluginLister at process start; subsequent calls use the
// same instance.
//
// Error handling per Drop 4c F.7-CORE F.7.6 acceptance criteria:
//
//   - claude binary missing on PATH → ErrClaudeBinaryMissing wrapping the
//     underlying exec.ErrNotFound. Distinguishable from
//     ErrMissingRequiredPlugins so dev sees "install claude" rather than
//     "install plugin X."
//   - Process-start or wait failure (other than NotFound) → wrapped raw
//     err so the dev sees the underlying syscall / signal context.
//   - Non-zero exit code → wrapped error naming the exit code; stderr is
//     captured separately and not currently surfaced (kept as a future
//     refinement).
//   - Stdout fails JSON decode → ErrPluginListUnparseable wrapping the
//     json.Unmarshal error for diagnostic fidelity.
//   - Context expiry → wrapped ctx.Err() so callers can route on
//     context.DeadlineExceeded vs context.Canceled.
type execClaudePluginLister struct{}

// List implements ClaudePluginLister by shelling out to
// `claude plugin list --json`. The context wraps a 5-second wall-clock
// timeout (pluginPreflightTimeout) on top of any outer ctx the caller
// supplies — whichever fires first cancels the child.
func (execClaudePluginLister) List(ctx context.Context) ([]ClaudePluginListEntry, error) {
	bounded, cancel := context.WithTimeout(ctx, pluginPreflightTimeout)
	defer cancel()

	cmd := exec.CommandContext(bounded, "claude", "plugin", "list", "--json")

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return nil, fmt.Errorf("%w: install claude per https://docs.claude.com/en/docs/claude-code (underlying: %v)",
				ErrClaudeBinaryMissing, err)
		}
		return nil, fmt.Errorf("dispatcher: start claude plugin list: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		if cerr := bounded.Err(); cerr != nil {
			return nil, fmt.Errorf("dispatcher: claude plugin list canceled: %w", cerr)
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("dispatcher: claude plugin list exited with code %d (stderr: %s)",
				exitErr.ExitCode(), strings.TrimSpace(stderrBuf.String()))
		}
		return nil, fmt.Errorf("dispatcher: claude plugin list wait: %w", err)
	}

	entries, err := parseClaudePluginList(stdoutBuf.Bytes())
	if err != nil {
		return nil, err
	}
	return entries, nil
}

// parseClaudePluginList decodes the stdout of `claude plugin list --json`
// into a []ClaudePluginListEntry. The expected shape is a top-level JSON
// array; an empty array is a legal "no plugins installed" response.
//
// Forward-compat: unknown JSON fields are silently ignored
// (encoding/json's default behavior with non-strict decoding). A future
// claude version that adds fields continues to parse cleanly without
// requiring this code to evolve in lockstep.
func parseClaudePluginList(stdout []byte) ([]ClaudePluginListEntry, error) {
	trimmed := bytes.TrimSpace(stdout)
	if len(trimmed) == 0 {
		// `claude plugin list --json` with no installed plugins emits an
		// empty array `[]`; an empty stdout is still treated as
		// "no plugins" rather than a parse error to keep the no-plugins
		// case ergonomic for adopters who don't depend on any.
		return nil, nil
	}
	var entries []ClaudePluginListEntry
	if err := json.Unmarshal(trimmed, &entries); err != nil {
		return nil, fmt.Errorf("%w: %v (stdout: %q)", ErrPluginListUnparseable, err, string(trimmed))
	}
	return entries, nil
}

// defaultClaudePluginLister is the production lister singleton. Tests
// override this var via t.Cleanup-restored assignment to inject a fake.
// Production code never reassigns it.
var defaultClaudePluginLister ClaudePluginLister = execClaudePluginLister{}
