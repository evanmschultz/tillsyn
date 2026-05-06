// Package dispatcher gate_push.go ships the F.7-CORE F.7.14 push gate: a
// repo-level post-commit step that runs `git push origin <branch>` against
// the project's primary worktree under the dispatcher's push-cadence rules.
//
// F.7.14 mirrors F.7.13's CommitGateRunner shape: a struct with injected
// test seams, a Run(ctx, *item, project, catalog, auth) entry point, and a
// closed-enum sentinel-error vocabulary. The match is deliberate — a future
// wiring droplet (F.7-CORE REV-13) adapts both gates to the gateRunner's
// gateFunc interface uniformly, so they MUST present the same call surface
// even when their argument slots (catalog, auth) are not consumed by the
// gate's internal algorithm.
//
// Toggle behavior: F.7.15's project metadata DispatcherPushEnabled toggle
// gates execution. When the toggle is unset OR explicitly false, Run is a
// pure no-op that returns nil — the gate did NOT run, and that is NOT an
// error. The dispatcher proceeds to the next gate in the sequence as if this
// gate had never been declared. Same default-OFF polarity as F.7.13's commit
// gate: a project must opt IN before the dispatcher takes push responsibility
// from the dev.
//
// Branch resolution: the gate reads the current worktree branch via the
// GitCurrentBranch seam (production wiring shells `git symbolic-ref --short
// HEAD`). Neither domain.Project nor domain.ActionItem carries a branch
// field today, and hardcoding "main" would silently break drop-branch
// worktrees where every dispatched build runs against `drop/N`. Empty
// returned branch (whitespace-only string) is treated as
// ErrPushGateBranchMissing — the gate cannot know where to push when HEAD
// is detached or the seam cannot resolve a symbolic ref.
//
// No auto-rollback: per F.7.14 spec, a failed push leaves the local commit
// from F.7.13 in place. The dispatcher routes the action item to `failed`
// with metadata.BlockedReason populated; the dev decides via TUI attention
// item whether to amend, force-push, drop the commit, or pivot. The gate
// itself never invokes `git reset` / `git revert` / `git push --force`.
package dispatcher

import (
	"context"
	"errors"
	"fmt"

	"github.com/evanmschultz/tillsyn/internal/domain"
	"github.com/evanmschultz/tillsyn/internal/templates"
)

// ErrPushGateDisabled is the symmetry-only sentinel for the toggle-off
// branch. Run itself does NOT return this sentinel — the toggle-off path
// is a successful no-op (returns nil) per the F.7.15 default-OFF contract.
// Exported for parity with F.7.13's ErrCommitGateDisabled and for future
// call sites (e.g. a CLI `till dispatcher push-gate --explain` that wants
// to distinguish disabled-by-toggle from disabled-by-template).
//
// Detect via errors.Is. The sentinel exists ONLY as a future-safe label —
// today's code paths never produce it.
var ErrPushGateDisabled = errors.New("dispatcher: push gate disabled by project toggle")

// ErrPushGateBranchMissing is returned by PushGateRunner.Run when the
// GitCurrentBranch seam resolves to an empty string OR returns a non-nil
// error. Both shapes collapse to one sentinel because both signal the same
// observable condition: the gate cannot determine which branch to push.
//
// Common underlying causes: detached HEAD (CI checked out a tag or a raw
// commit hash), the worktree is not a git repository (project
// misconfiguration), `git symbolic-ref` rejected the ref for an exotic
// reason, or `git` not on PATH (host environment misconfiguration).
//
// Detect via errors.Is. When the underlying cause is a non-nil seam error
// it is also reachable via errors.Is — the wrapping uses fmt.Errorf("%w:
// %w") so callers can route on both axes.
var ErrPushGateBranchMissing = errors.New("dispatcher: push gate: branch not resolvable")

// ErrPushGatePushFailed wraps `git push` failures. Detect via errors.Is;
// the underlying *exec.ExitError or os/exec error is wrapped after the
// sentinel via fmt.Errorf("%w: %w") so callers can route both axes
// (sentinel-class and underlying-cause) without manual unwrapping.
//
// Common causes the sentinel indicates: authentication failure (no
// credentials, expired token, SSH key missing), non-fast-forward rejection
// (remote has commits the local does not), network error (offline, DNS
// failure, remote unreachable), branch protection rejection (push to
// protected ref forbidden), or `git` not on PATH.
//
// Per F.7.14 spec, a non-nil error here does NOT trigger automatic local
// commit rollback — the gate surfaces the failure and the action item moves
// to `failed` with metadata.BlockedReason populated by the gate runner's
// caller (route deferred to F.7.5b/wiring-droplet).
var ErrPushGatePushFailed = errors.New("dispatcher: push gate: git push failed")

// GitPushFunc is the test seam for `git push origin <branch>` in the
// project worktree. Production wiring shells `git push origin <branch>`;
// tests inject deterministic stubs.
//
// The repoPath argument is the absolute path to the project worktree root
// passed through from project.RepoPrimaryWorktree. The branch argument is
// the symbolic ref name (e.g. "main", "drop/4c") returned by
// GitCurrentBranchFunc — implementations MUST treat the value verbatim
// (no automatic prefix, no automatic refspec rewriting).
//
// Returning a non-nil error halts the gate; PushGateRunner.Run wraps the
// error with ErrPushGatePushFailed.
type GitPushFunc func(ctx context.Context, repoPath string, branch string) error

// GitCurrentBranchFunc is the test seam for `git symbolic-ref --short HEAD`
// in the project worktree. Production wiring shells the command and trims
// the trailing newline; tests inject deterministic stubs.
//
// The returned branch MUST be the bare symbolic ref name with NO trailing
// newline or whitespace. Trimming is the implementation's responsibility,
// not the gate's.
//
// An empty (whitespace-trimmed) return value is treated as
// ErrPushGateBranchMissing by the gate. A non-nil error is also wrapped
// with ErrPushGateBranchMissing — both shapes collapse to one sentinel
// because both signal "branch not resolvable" to downstream consumers.
type GitCurrentBranchFunc func(ctx context.Context, repoPath string) (string, error)

// PushGateRunner orchestrates the F.7.14 push gate: toggle check, branch
// resolution via GitCurrentBranch, and `git push origin <branch>` against
// the project's primary worktree.
//
// Production wiring assigns:
//
//	pushGate := &PushGateRunner{
//	    GitCurrentBranch: adapters.GitCurrentBranch,
//	    GitPush:          adapters.GitPush,
//	}
//
// Both fields MUST be non-nil; nil-field detection happens lazily inside
// Run (a nil GitCurrentBranch / GitPush causes Run to return a loud error
// rather than nil-derefing). Production code MUST pass non-nil fields;
// nil-field tolerance is a defense-in-depth concern rather than a
// documented contract.
//
// Concurrency: a single PushGateRunner value services concurrent Run calls
// when the underlying GitCurrentBranch / GitPushFunc implementations are
// themselves safe for concurrent use against distinct repoPaths. Production
// adapters today shell `os/exec` per-call and are safe under that
// constraint; the gate value itself holds no mutable state.
type PushGateRunner struct {
	// GitCurrentBranch shells `git symbolic-ref --short HEAD` in
	// project.RepoPrimaryWorktree and returns the trimmed branch name.
	// MUST be non-nil. Production wiring binds an os/exec-backed adapter;
	// tests inject deterministic stubs.
	GitCurrentBranch GitCurrentBranchFunc

	// GitPush shells `git push origin <branch>` in
	// project.RepoPrimaryWorktree. MUST be non-nil. Production wiring binds
	// an os/exec-backed adapter; tests inject deterministic stubs.
	GitPush GitPushFunc
}

// Run executes the F.7.14 push gate algorithm:
//
//  1. If project.Metadata.IsDispatcherPushEnabled() returns false, return
//     nil. The gate is disabled-by-toggle and that is a successful no-op,
//     not an error. NO git commands are invoked.
//  2. Call GitCurrentBranch(ctx, project.RepoPrimaryWorktree). On error OR
//     empty return, wrap with ErrPushGateBranchMissing.
//  3. Call GitPush(ctx, project.RepoPrimaryWorktree, branch). On error,
//     wrap with ErrPushGatePushFailed.
//  4. Return nil.
//
// Mutation: Run does NOT mutate item — push is a repo-level operation with
// no per-action-item field to update (compare to F.7.13's EndCommit
// mutation, which captures the new HEAD hash). The *domain.ActionItem
// pointer is accepted for signature symmetry with CommitGateRunner.Run so
// the wiring droplet (F.7-CORE REV-13) treats both gates uniformly.
//
// Idempotency: the gate is conditionally idempotent. A second Run on the
// same already-pushed commit will succeed-with-no-op at the git layer
// (git's "Everything up-to-date" exit code 0). A second Run after the
// remote has accepted further commits will fail with non-fast-forward,
// wrapped as ErrPushGatePushFailed.
//
// Auth: the auth and catalog parameters are accepted for signature
// symmetry with CommitGateRunner.Run. The push gate itself does not spawn
// an LLM-backed agent and does not consult the catalog; the parameters
// are inert for this gate.
func (r *PushGateRunner) Run(
	ctx context.Context,
	item *domain.ActionItem,
	project domain.Project,
	catalog templates.KindCatalog,
	auth AuthBundle,
) error {
	if r == nil {
		return errors.New("dispatcher: push gate: nil PushGateRunner receiver")
	}
	if item == nil {
		return errors.New("dispatcher: push gate: nil action item")
	}

	// Step 1: toggle gate. Default OFF (DispatcherPushEnabled == nil) →
	// no-op. Explicit false → no-op. Only explicit true proceeds.
	if !project.Metadata.IsDispatcherPushEnabled() {
		return nil
	}

	// Step 2: resolve current branch. Empty return OR non-nil error both
	// collapse to ErrPushGateBranchMissing — the gate cannot determine
	// where to push and surfacing both shapes through one sentinel keeps
	// the failure-routing logic on the caller side simpler.
	if r.GitCurrentBranch == nil {
		return errors.New("dispatcher: push gate: nil GitCurrentBranch (production wiring bug)")
	}
	branch, err := r.GitCurrentBranch(ctx, project.RepoPrimaryWorktree)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrPushGateBranchMissing, err)
	}
	if branch == "" {
		return fmt.Errorf("%w: empty branch returned", ErrPushGateBranchMissing)
	}

	// Step 3: `git push origin <branch>` in the project worktree. The
	// branch name is taken verbatim from GitCurrentBranch — no automatic
	// prefix, no automatic refspec rewriting, no force-push.
	if r.GitPush == nil {
		return errors.New("dispatcher: push gate: nil GitPush (production wiring bug)")
	}
	if err := r.GitPush(ctx, project.RepoPrimaryWorktree, branch); err != nil {
		return fmt.Errorf("%w: %w", ErrPushGatePushFailed, err)
	}

	return nil
}

// Compile-time interface symmetry check. The push gate accepts the same
// (catalog, auth) tail-args as the commit gate so the future
// gateRunner-adapter (F.7-CORE REV-13) can treat both uniformly. Reference
// the unused params via _ assignment so go vet does not flag them and so
// dropping a parameter from one gate without the other surfaces as a
// compile error here.
var _ = func(r *PushGateRunner, ctx context.Context, item *domain.ActionItem, project domain.Project, catalog templates.KindCatalog, auth AuthBundle) error {
	_ = catalog
	_ = auth
	return r.Run(ctx, item, project, catalog, auth)
}
