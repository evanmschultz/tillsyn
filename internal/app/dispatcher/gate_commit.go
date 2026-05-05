// Package dispatcher gate_commit.go ships the F.7-CORE F.7.13 commit gate:
// a path-scoped post-build commit step that consumes F.7.12's CommitAgent for
// the message and runs `git add` + `git commit` against the project worktree
// under the dispatcher's commit-cadence rules.
//
// F.7.13 is structurally distinct from the Drop 4b Wave A gates (mage_ci,
// mage_test_pkg) because the commit gate has side effects — it creates a real
// commit and mutates the action item's EndCommit field. The existing gateFunc
// signature ((ctx, item, project) GateResult) cannot carry a catalog handle,
// an auth bundle, or a pointer to mutate the action item in place; the
// CommitGateRunner exposes a richer signature dedicated to this gate. The
// Drop 4c follow-up wiring droplet adapts CommitGateRunner.Run to the
// gateRunner's gateFunc interface so the closed-enum dispatch in gates.go can
// route a templates.GateKindCommit entry to this runner without breaking the
// Wave A contract.
//
// Toggle behavior: F.7.15's project metadata DispatcherCommitEnabled toggle
// gates execution. When the toggle is unset OR explicitly false, Run is a
// pure no-op that returns nil — the gate did NOT run, and that is NOT an
// error. The dispatcher proceeds to the next gate in the sequence as if this
// gate had never been declared. This polarity matches the F.7.15 default-OFF
// rule: a project must opt IN before the dispatcher takes commit
// responsibility from the dev.
//
// Path scoping: `git add` is invoked with the action item's Paths slice
// verbatim. The gate NEVER passes `-A`, `--all`, or `.` — narrowing scope to
// the planner-declared write-set is a project-level rule (CLAUDE.md "Git
// Management (Pre-Cascade)") and the gate enforces it by construction. An
// empty Paths slice is a hard failure (ErrCommitGateNoPaths) rather than a
// silent no-op: an action item that produced no path declarations either
// (a) is wired to the wrong gate, or (b) has a planner bug that omitted the
// paths. Either way, the dev needs the failure visible.
//
// EndCommit mutation: on success, the gate writes the new HEAD hash returned
// by `git rev-parse HEAD` into item.EndCommit. The action item is passed by
// pointer specifically so this mutation is observable to the caller —
// downstream gates (the future push gate, the F.7.18 context aggregator)
// read item.EndCommit to drive their own behavior.
package dispatcher

import (
	"context"
	"errors"
	"fmt"

	"github.com/evanmschultz/tillsyn/internal/domain"
	"github.com/evanmschultz/tillsyn/internal/templates"
)

// ErrCommitGateDisabled is returned when callers explicitly want to detect
// the toggle-off branch. Run itself does NOT return this sentinel — the
// toggle-off path is a successful no-op (returns nil) per the F.7.15
// default-OFF contract. The sentinel is exported for symmetry with the rest
// of the gate's error vocabulary AND for future call sites (e.g. a CLI
// `till dispatcher commit-gate --explain` that wants to distinguish
// disabled-by-toggle from disabled-by-template).
//
// Detect via errors.Is. The sentinel exists ONLY as a future-safe label —
// today's code paths never produce it.
var ErrCommitGateDisabled = errors.New("dispatcher: commit gate disabled by project toggle")

// ErrCommitGateNoPaths is returned by CommitGateRunner.Run when the action
// item's Paths slice is empty. Empty paths is a hard failure per F.7.13: a
// path-scoped commit gate cannot operate without a path scope. Detect via
// errors.Is.
//
// The disposition is fail-loud rather than fail-silent because an empty
// Paths set in the build-then-commit pipeline indicates either (a) the
// planner omitted the paths declaration (a planner bug), or (b) the gate is
// bound to a kind that does not declare paths (a template bug). Silent
// no-op would mask both bugs — the gate was supposed to commit something
// and committed nothing.
var ErrCommitGateNoPaths = errors.New("dispatcher: commit gate: action item declares no paths")

// ErrCommitGateAddFailed wraps `git add` failures. Detect via errors.Is; the
// underlying *exec.ExitError or os/exec error is wrapped after the sentinel
// via fmt.Errorf("%w: %w") so callers can route both axes (sentinel-class
// and underlying-cause) without manual unwrapping.
//
// Common causes the sentinel indicates: a path that does not exist in the
// worktree (typo in the action item's Paths), a path outside the worktree
// (relative-path resolution misconfiguration), a worktree that is not a git
// repository (project misconfiguration), or `git` not on PATH (host
// environment misconfiguration).
var ErrCommitGateAddFailed = errors.New("dispatcher: commit gate: git add failed")

// ErrCommitGateCommitFailed wraps `git commit` failures. Detect via
// errors.Is. Common underlying causes: nothing to commit (the planner
// declared paths but did not actually modify them), pre-commit hook
// rejection, a worktree where commits are forbidden (detached HEAD on a
// protected ref), or a git config error (no user.name / user.email).
//
// "Nothing to commit" surfaces here, not at the `git add` step, because
// `git add` succeeds idempotently on unchanged files — the empty staging
// area is detected by `git commit` itself.
var ErrCommitGateCommitFailed = errors.New("dispatcher: commit gate: git commit failed")

// ErrCommitGateRevParseFailed wraps `git rev-parse HEAD` failures. Detect
// via errors.Is. The rev-parse step runs AFTER a successful commit, so the
// underlying causes are narrow: a worktree that lost its .git directory
// between `git commit` and `git rev-parse` (concurrent-process interference),
// or `git` rejecting the HEAD ref for an exotic reason (corrupted refs/HEAD).
// Either way the gate cannot populate item.EndCommit and the caller MUST
// route the action item to a manual-recovery state.
var ErrCommitGateRevParseFailed = errors.New("dispatcher: commit gate: git rev-parse HEAD failed")

// GitAddFunc is the test seam for `git add <paths>` in the project worktree.
// Production wiring shells `git add -- <paths...>` (the `--` separator
// rejects path-as-flag injection); tests inject deterministic stubs.
//
// Implementations MUST treat the paths slice verbatim — no `-A`, no `.`, no
// glob expansion. Any normalization (e.g. trimming whitespace, dropping
// empty entries) is the planner's responsibility upstream. The repoPath
// argument is the absolute path to the project worktree root passed through
// from project.RepoPrimaryWorktree.
//
// Returning a non-nil error halts the gate; CommitGateRunner.Run wraps the
// error with ErrCommitGateAddFailed.
type GitAddFunc func(ctx context.Context, repoPath string, paths []string) error

// GitCommitFunc is the test seam for `git commit -m <message>` in the
// project worktree. Production wiring shells `git commit -m <message>`;
// tests inject deterministic stubs.
//
// The message argument is the single-line conventional-commit body produced
// by F.7.12's CommitAgent.GenerateMessage. Implementations MUST NOT mutate
// the message (no automatic prefix, no automatic Signed-off-by trailer).
// Returning a non-nil error halts the gate; CommitGateRunner.Run wraps the
// error with ErrCommitGateCommitFailed.
type GitCommitFunc func(ctx context.Context, repoPath string, message string) error

// GitRevParseFunc is the test seam for `git rev-parse HEAD` in the project
// worktree. Production wiring shells `git rev-parse HEAD` and trims the
// trailing newline; tests inject deterministic stubs.
//
// The returned hash MUST be the full 40-character SHA-1 (or 64-character
// SHA-256 on repos using the SHA-256 object format) with NO trailing
// newline or whitespace. Trimming is the implementation's responsibility,
// not the gate's: the gate writes the returned string verbatim into
// item.EndCommit.
//
// Returning a non-nil error halts the gate; CommitGateRunner.Run wraps the
// error with ErrCommitGateRevParseFailed.
type GitRevParseFunc func(ctx context.Context, repoPath string) (string, error)

// CommitGateRunner orchestrates the F.7.13 commit gate: toggle check, paths
// guard, commit-message generation via F.7.12, path-scoped `git add` +
// `git commit`, and `git rev-parse HEAD` to capture the resulting commit
// hash on item.EndCommit.
//
// Production wiring assigns:
//
//	commitGate := &CommitGateRunner{
//	    CommitAgent:     commitAgent, // F.7.12 instance
//	    GitAdd:          adapters.GitAdd,
//	    GitCommit:       adapters.GitCommit,
//	    GitRevParseHead: adapters.GitRevParseHead,
//	}
//
// All four fields MUST be non-nil; nil-field detection happens lazily inside
// Run (a nil GitAdd / GitCommit / GitRevParseHead causes Run to wrap the
// nil-deref panic in the matching sentinel error). Production code MUST
// pass non-nil fields; nil-field tolerance is a defense-in-depth concern
// rather than a documented contract.
//
// Concurrency: a single CommitGateRunner value services concurrent Run
// calls when the underlying GitAdd / GitCommit / GitRevParseFunc
// implementations are themselves safe for concurrent use against distinct
// repoPaths. Production adapters today shell `os/exec` per-call and are
// safe under that constraint; the gate value itself holds no mutable state.
type CommitGateRunner struct {
	// CommitAgent is the F.7.12 message-generation surface. MUST be non-nil.
	// Run calls CommitAgent.GenerateMessage(ctx, *item, project, catalog,
	// auth) to obtain the single-line commit message before invoking GitAdd.
	CommitAgent *CommitAgent

	// GitAdd shells `git add -- <paths...>` in project.RepoPrimaryWorktree.
	// MUST be non-nil. Production wiring binds an os/exec-backed adapter;
	// tests inject deterministic stubs.
	GitAdd GitAddFunc

	// GitCommit shells `git commit -m <message>` in project.RepoPrimaryWorktree.
	// MUST be non-nil. Production wiring binds an os/exec-backed adapter;
	// tests inject deterministic stubs.
	GitCommit GitCommitFunc

	// GitRevParseHead shells `git rev-parse HEAD` in project.RepoPrimaryWorktree
	// and returns the trimmed commit hash. MUST be non-nil. Production
	// wiring binds an os/exec-backed adapter; tests inject deterministic
	// stubs.
	GitRevParseHead GitRevParseFunc
}

// Run executes the F.7.13 commit gate algorithm:
//
//  1. If project.Metadata.IsDispatcherCommitEnabled() returns false, return
//     nil. The gate is disabled-by-toggle and that is a successful no-op,
//     not an error. NO git commands are invoked, NO commit-message agent
//     is spawned, item.EndCommit is unchanged.
//  2. If item.Paths is empty, return ErrCommitGateNoPaths. Path-scoped
//     commit cannot operate without a path scope.
//  3. Call CommitAgent.GenerateMessage(ctx, *item, project, catalog, auth)
//     to obtain the single-line conventional-commit message. F.7.12 owns
//     length / multi-line validation. Failures propagate verbatim (wrapped
//     with a "commit gate: " prefix so the gate-name shows up in the error
//     chain alongside F.7.12's own ErrCommitMessageTooLong /
//     ErrCommitSpawnNoTerminal sentinels).
//  4. Call GitAdd(ctx, project.RepoPrimaryWorktree, item.Paths). On error,
//     wrap with ErrCommitGateAddFailed.
//  5. Call GitCommit(ctx, project.RepoPrimaryWorktree, message). On error,
//     wrap with ErrCommitGateCommitFailed.
//  6. Call GitRevParseHead(ctx, project.RepoPrimaryWorktree). On error,
//     wrap with ErrCommitGateRevParseFailed. Empty (whitespace-only) hash
//     is treated as a rev-parse failure — the gate cannot populate
//     item.EndCommit with an empty value because downstream gates use
//     non-empty as the "commit happened" signal.
//  7. Set item.EndCommit = newHash. Return nil.
//
// Mutation: item is passed by pointer specifically so step 7's EndCommit
// write is observable to the caller. Run does not mutate any other field.
//
// Idempotency: the gate is NOT idempotent. A second Run on the same item
// will spawn the commit-message agent again, run `git add` against the
// (now-clean) tree, and fail at GitCommit with "nothing to commit" wrapped
// as ErrCommitGateCommitFailed. Callers must not retry Run on success.
func (r *CommitGateRunner) Run(
	ctx context.Context,
	item *domain.ActionItem,
	project domain.Project,
	catalog templates.KindCatalog,
	auth AuthBundle,
) error {
	if r == nil {
		return errors.New("dispatcher: commit gate: nil CommitGateRunner receiver")
	}
	if item == nil {
		return errors.New("dispatcher: commit gate: nil action item")
	}

	// Step 1: toggle gate. Default OFF (DispatcherCommitEnabled == nil) →
	// no-op. Explicit false → no-op. Only explicit true proceeds.
	if !project.Metadata.IsDispatcherCommitEnabled() {
		return nil
	}

	// Step 2: paths guard. An empty Paths slice is a hard failure rather
	// than a silent no-op so a misconfigured planner / template surface is
	// visible in the failure-loop instinct.
	if len(item.Paths) == 0 {
		return fmt.Errorf("%w: action_item=%q", ErrCommitGateNoPaths, item.ID)
	}

	// Step 3: generate the commit message via F.7.12. F.7.12 owns its own
	// validation (length cap, multi-line rejection, missing diff anchors);
	// failures propagate with a "commit gate: " prefix so the gate-name
	// shows up in the error chain.
	if r.CommitAgent == nil {
		return errors.New("dispatcher: commit gate: nil CommitAgent (production wiring bug)")
	}
	message, err := r.CommitAgent.GenerateMessage(ctx, *item, project, catalog, auth)
	if err != nil {
		return fmt.Errorf("dispatcher: commit gate: generate message: %w", err)
	}

	// Step 4: `git add -- <paths>` in the project worktree. Path-scoped per
	// the gate's hard constraint — no `-A`.
	if r.GitAdd == nil {
		return errors.New("dispatcher: commit gate: nil GitAdd (production wiring bug)")
	}
	if err := r.GitAdd(ctx, project.RepoPrimaryWorktree, item.Paths); err != nil {
		return fmt.Errorf("%w: %w", ErrCommitGateAddFailed, err)
	}

	// Step 5: `git commit -m <message>`. The message is taken verbatim from
	// the F.7.12 agent — no automatic prefix, no automatic Signed-off-by.
	if r.GitCommit == nil {
		return errors.New("dispatcher: commit gate: nil GitCommit (production wiring bug)")
	}
	if err := r.GitCommit(ctx, project.RepoPrimaryWorktree, message); err != nil {
		return fmt.Errorf("%w: %w", ErrCommitGateCommitFailed, err)
	}

	// Step 6: capture the new HEAD hash. An empty (whitespace-trimmed) hash
	// is treated as a rev-parse failure — downstream gates use non-empty
	// as the "commit happened" signal and an empty value would silently
	// poison that read.
	if r.GitRevParseHead == nil {
		return errors.New("dispatcher: commit gate: nil GitRevParseHead (production wiring bug)")
	}
	newHash, err := r.GitRevParseHead(ctx, project.RepoPrimaryWorktree)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrCommitGateRevParseFailed, err)
	}
	if newHash == "" {
		return fmt.Errorf("%w: empty hash returned", ErrCommitGateRevParseFailed)
	}

	// Step 7: mutate item.EndCommit with the new HEAD hash. Observable to
	// the caller via the *domain.ActionItem pointer.
	item.EndCommit = newHash
	return nil
}
