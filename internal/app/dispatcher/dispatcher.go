package dispatcher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/evanmschultz/tillsyn/internal/domain"
	"github.com/evanmschultz/tillsyn/internal/templates"
)

// Result is a closed enum classifying one RunOnce outcome.
type Result string

// Result constants enumerate every RunOnce verdict the dispatcher can emit.
// The set is intentionally small; later Wave 2 droplets refine which path
// each one takes (e.g. ResultBlocked is emitted by the conflict detector in
// 2.7) but never add new values.
const (
	// ResultSpawned reports that the dispatcher launched one agent subprocess
	// for the action item. Wave 2.10 (droplet 4a.23) wires the actual spawn;
	// pre-2.10 droplets never returned this value.
	ResultSpawned Result = "spawned"
	// ResultSkipped reports that the action item is ineligible for spawn at
	// this moment (not in todo, missing, or otherwise filtered). Skipping is
	// not a failure: the dispatcher will reconsider the item on the next
	// state-change event.
	ResultSkipped Result = "skipped"
	// ResultBlocked reports that a runtime blocker was inserted (sibling
	// overlap on paths or packages, per Wave 2.7) and the action item is
	// waiting on a sibling to complete. The orchestrator receives an
	// attention item alongside this result.
	ResultBlocked Result = "blocked"
	// ResultFailed reports that the dispatcher tried to spawn the action
	// item but the spawn or its monitored process produced an error. The
	// action item is moved to the failed lifecycle state by the monitor
	// (Wave 2.8).
	ResultFailed Result = "failed"
)

// DispatchOutcome captures one RunOnce result for logging, the manual-trigger
// CLI (Wave 2.10), and downstream dashboards.
type DispatchOutcome struct {
	// ActionItemID is the input ID, trimmed and validated.
	ActionItemID string
	// AgentName is the resolved agent variant (e.g. go-builder-agent). Empty
	// when Result is Skipped or the action item has no kind binding.
	AgentName string
	// SpawnedAt is the wall-clock time the dispatcher recorded the outcome.
	// For ResultSkipped this is the moment the skip decision was reached;
	// for ResultSpawned it is the moment immediately before the subprocess
	// is launched.
	SpawnedAt time.Time
	// Result is the closed-enum verdict. See Result constants.
	Result Result
	// Reason is a short human-readable explanation accompanying the verdict.
	// Set on ResultSkipped (e.g. "not in todo", "no agent binding"),
	// ResultBlocked (e.g. "sibling overlap on internal/app/foo.go"), and
	// ResultFailed. Empty on ResultSpawned. Populated for the CLI's
	// human-readable line so callers do not have to derive the reason from
	// the surrounding logging.
	Reason string
	// Handle is the per-process tracking record for the spawned agent. Set
	// on ResultSpawned only; nil otherwise. The CLI surfaces a one-line
	// "spawned <agent> for <id>" message and returns immediately — the
	// caller is responsible for not blocking on Handle.Wait unless they
	// explicitly want to.
	Handle *Handle
}

// Options carries dispatcher configuration. The struct is intentionally open
// for extension: Wave 2.6, 2.8, and 2.10 each add fields, and Drop 4b adds
// gate-runner / commit-agent fields. Add new fields rather than introduce a
// new constructor variant. Zero-value Options is valid in Wave 2.1.
type Options struct {
	// _ is a placeholder so gofumpt-formatted struct literals using positional
	// composition produce a clear compile error if a caller relies on the
	// field shape. Later droplets replace this with concrete fields.
	_ struct{}
}

// Dispatcher is the cascade dispatcher contract consumed by the manual-trigger
// CLI (Wave 2.10) and Drop 4b's daemon mode. RunOnce is the only method
// fully wired in Wave 2; Start/Stop are stubs that return ErrNotImplemented
// until Drop 4b lands the continuous-mode loop.
type Dispatcher interface {
	// RunOnce evaluates one action item and either skips it, inserts a
	// blocker, or spawns an agent. It is the manual-trigger entry point.
	// RunOnce returns a non-nil error only for unexpected failures
	// (database errors, malformed configuration). Eligibility decisions
	// (todo-only, blocker-cleared) surface as Result values, not errors.
	//
	// projectIDOverride, when non-empty, is the authoritative project_id
	// the dispatcher MUST resolve. The dispatcher returns ErrProjectMismatch
	// when the override does not match item.ProjectID. An empty override
	// preserves the historical behaviour of resolving the project from the
	// action item itself.
	RunOnce(ctx context.Context, actionItemID, projectIDOverride string) (DispatchOutcome, error)
	// Start begins the continuous-mode dispatcher loop that subscribes to
	// LiveWaitBroker action-item-changed events and walks the tree on each
	// wakeup. Drop 4b implements this; Wave 2 returns ErrNotImplemented.
	Start(ctx context.Context) error
	// Stop tears down the continuous-mode loop started by Start. Drop 4b
	// implements this; Wave 2 returns ErrNotImplemented.
	Stop(ctx context.Context) error
}

// Sentinel errors exposed by the dispatcher package.
var (
	// ErrNotImplemented was returned by Start and Stop pre-Drop-4b.7 while
	// the continuous-mode loop was a stub. Drop 4b.7 wires real Start/Stop
	// bodies; the sentinel is preserved as a deprecated alias for backward
	// compat with callers that still detect it via errors.Is.
	//
	// Deprecated: Drop 4b.7 wired Start/Stop. Use ErrAlreadyStarted to detect
	// duplicate Start calls; clean Stop returns nil.
	ErrNotImplemented = errors.New("dispatcher: continuous-mode not implemented in wave 2")
	// ErrAlreadyStarted is returned by Start when the dispatcher's
	// continuous-mode loop has already been started (and not yet stopped),
	// or when the dispatcher has been stopped previously and a re-start is
	// attempted. Callers detect with errors.Is so the till serve flow
	// surfaces "dispatcher already running" cleanly rather than spawning
	// duplicate subscriber goroutines.
	ErrAlreadyStarted = errors.New("dispatcher: continuous-mode already started")
	// ErrInvalidDispatcherConfig is returned by NewDispatcher when a required
	// dependency is nil. Callers detect this with errors.Is to give the
	// dev a precise misconfiguration message.
	ErrInvalidDispatcherConfig = errors.New("dispatcher: invalid configuration")
	// ErrProjectMismatch is returned by RunOnce / PreviewSpawn when the
	// caller-supplied authoritative project_id does not match the action
	// item's own ProjectID. Callers detect this with errors.Is so the
	// manual-trigger CLI surfaces a precise "wrong project" message rather
	// than a silent skip. Counterexample fix from 4a.23 QA-Falsification
	// §2.2: the --project flag was previously plumbed but never read.
	ErrProjectMismatch = errors.New("dispatcher: action item belongs to a different project")
)

// actionItemReader is the narrow consumer-side view the dispatcher uses to
// fetch action items. *app.Service satisfies this interface; the indirection
// lets the test suite inject deterministic stubs without standing up a full
// service + repository graph.
type actionItemReader interface {
	GetActionItem(ctx context.Context, actionItemID string) (domain.ActionItem, error)
}

// projectReader is the narrow consumer-side view the dispatcher uses to load
// the project a spawn targets. *app.Service satisfies this interface (via
// the Service.GetProject method added alongside the cascade-dispatcher wiring
// in droplet 4a.23). The wave-2 RunOnce path needs the project's
// RepoPrimaryWorktree (cmd.Dir for the spawn), KindCatalogJSON (resolves
// AgentBinding for the action item's kind), and HyllaArtifactRef (prompt
// structural field).
type projectReader interface {
	GetProject(ctx context.Context, projectID string) (domain.Project, error)
}

// projectLister is the narrow consumer-side view the continuous-mode loop
// (Drop 4b.7 Start) uses to enumerate projects at startup. The subscriber
// spins one goroutine per project ID returned by ListProjects; projects
// added after Start are NOT picked up automatically — that is a Drop 4c /
// Drop 5 dogfood refinement (would require a LiveWaitEventProjectCreated
// event the broker does not yet publish).
type projectLister interface {
	ListProjects(ctx context.Context, includeArchived bool) ([]domain.Project, error)
}

// listingService is the narrow view the dispatcher uses to read sibling
// snapshots for the conflict detector and to resolve column IDs for promotion.
type listingService interface {
	ListColumns(ctx context.Context, projectID string, includeArchived bool) ([]domain.Column, error)
	ListActionItems(ctx context.Context, projectID string, includeArchived bool) ([]domain.ActionItem, error)
}

// failureTransitioner is the narrow consumer-side view RunOnce uses to
// transition an action item to StateFailed when Stage 8 (monitor.Track)
// fails after Stage 7 has already promoted to in_progress. The shape mirrors
// the failure half of monitor.applyCrashTransition (monitor.go:326-369) so
// the rollback writes the same Outcome/BlockedReason metadata shape the
// crash path writes — keeping a single canonical "agent failure" surface
// regardless of whether the cmd died at start (Stage 8 here) or during
// execution (the monitor's runHandle goroutine). *app.Service satisfies
// this interface via monitorServiceAdapter; the test suite injects a
// stub for deterministic Stage 8 rollback assertions.
type failureTransitioner interface {
	ListColumns(ctx context.Context, projectID string, includeArchived bool) ([]domain.Column, error)
	MoveActionItem(ctx context.Context, actionItemID, toColumnID string, position int) (domain.ActionItem, error)
	UpdateActionItem(ctx context.Context, in updateActionItemInput) (domain.ActionItem, error)
}

// dispatcher is the concrete implementation. The struct is intentionally
// unexported: callers depend on the Dispatcher interface, and NewDispatcher
// returns *dispatcher so Wave 2 droplets can add methods on the concrete
// type without breaking interface conformance.
type dispatcher struct {
	// svc is the application-service action-item reader. Stored as
	// actionItemReader for test-injection symmetry; the production
	// constructor wires *app.Service through.
	svc actionItemReader
	// projects loads project rows for spawn-site context. Production
	// constructor wires *app.Service; the unexported struct-literal path
	// (used by the test suite) injects a stub.
	projects projectReader
	// listing reads columns + action items for the walker / conflict
	// detector / promotion path. Production constructor wires *app.Service.
	listing listingService
	// mutator drives the Stage 8 failure-rollback path: when monitor.Track
	// fails after walker.Promote already moved the item to in_progress,
	// the dispatcher transitions the item to failed (with metadata) and
	// fires the cleanup hook. Wired via monitorServiceAdapter in production;
	// nil in older test fixtures that do not exercise the rollback path —
	// nil-safety is enforced inside transitionToFailed.
	mutator failureTransitioner
	// broker is the live-wait broker the continuous-mode loop will subscribe
	// to in Drop 4b. RunOnce does not consume it directly today, but the
	// constructor validates non-nil so misconfiguration surfaces at
	// startup rather than on first state-change event.
	broker app.LiveWaitBroker
	// walker is the auto-promotion engine (4a.18). RunOnce calls
	// EligibleForPromotion to drive the project-scoped sibling list and
	// Promote to move the candidate to in_progress before the spawn.
	walker *treeWalker
	// conflict is the sibling-overlap detector (4a.20). RunOnce calls
	// DetectSiblingOverlap before lock acquisition and InsertRuntimeBlockedBy
	// when an overlap exists without an explicit blocked_by edge.
	conflict *conflictDetector
	// fileLocks serializes spawns whose action items declare overlapping
	// paths (4a.16). RunOnce calls Acquire before promotion and the cleanup
	// hook calls Release on terminal state.
	fileLocks *fileLockManager
	// pkgLocks serializes spawns whose action items declare overlapping Go
	// packages (4a.17). Same lifecycle as fileLocks.
	pkgLocks *packageLockManager
	// monitor tracks the spawned subprocess (4a.21) and drives the
	// crash-path transition to StateFailed.
	monitor *processMonitor
	// cleanup releases locks + monitor entries on terminal state (4a.22).
	cleanup *cleanupHook
	// projectsLister enumerates projects at Start time so the continuous-mode
	// loop (Drop 4b.7) can spin one subscriber goroutine per project.
	// Production wiring assigns *app.Service; older tests that pre-date
	// Drop 4b.7 leave this nil and never call Start.
	projectsLister projectLister
	// opts carries forward-compatible configuration. Wave 2 leaves it at
	// zero value; later droplets read concrete fields.
	opts Options
	// clock returns the current time for outcome timestamps. Tests inject a
	// fake clock through a non-exported helper (see dispatcher_test.go);
	// production callers get time.Now via the constructor default.
	clock func() time.Time

	// Continuous-mode lifecycle state (Drop 4b.7). All four fields below are
	// read/written under subMu; subMu must NOT be held while invoking any
	// external surface (broker, walker, RunOnce) because those calls block on
	// the subscriber's own goroutines.
	subMu     sync.Mutex
	started   bool
	stopped   bool
	subCancel context.CancelFunc
	subWG     sync.WaitGroup
}

// NewDispatcher constructs a dispatcher wired with the full Wave-2 component
// graph: walker, spawn helper, conflict detector, file + package lock
// managers, process monitor, and terminal-state cleanup hook.
//
// svc and broker MUST be non-nil; opts is forward-compatible (zero value is
// valid). Returns ErrInvalidDispatcherConfig wrapped with the offending
// dependency name on validation failure.
//
// Internal component construction is hermetic: the lock managers, walker,
// conflict detector, monitor, and cleanup hook all close over svc + broker
// without taking additional caller-side arguments. Drop 4b's daemon variant
// will accept a richer Options shape (gate runner, commit agent) but the
// constructor signature stays the same — new fields land on Options, not on
// new function parameters.
func NewDispatcher(svc *app.Service, broker app.LiveWaitBroker, opts Options) (*dispatcher, error) {
	if svc == nil {
		return nil, fmt.Errorf("%w: svc is nil", ErrInvalidDispatcherConfig)
	}
	if broker == nil {
		return nil, fmt.Errorf("%w: broker is nil", ErrInvalidDispatcherConfig)
	}

	walker := newTreeWalker(svc)
	conflict := newConflictDetector(svc)
	fileLocks := newFileLockManager()
	pkgLocks := newPackageLockManager()
	adapter := monitorServiceAdapter{svc: svc}
	monitor := newProcessMonitor(adapter, nil)
	cleanup, err := newCleanupHook(fileLocks, pkgLocks, monitor, svc)
	if err != nil {
		// newCleanupHook only errors when one of its inputs is nil — defense
		// in depth; the lock managers + monitor above are always non-nil at
		// this point. Wrap with context anyway so a future regression
		// surfaces a clear message rather than a bare ErrInvalidDispatcherConfig.
		return nil, fmt.Errorf("dispatcher: wire cleanup hook: %w", err)
	}

	return &dispatcher{
		svc:            svc,
		projects:       svc,
		listing:        svc,
		mutator:        adapter,
		broker:         broker,
		walker:         walker,
		conflict:       conflict,
		fileLocks:      fileLocks,
		pkgLocks:       pkgLocks,
		monitor:        monitor,
		cleanup:        cleanup,
		projectsLister: svc,
		opts:           opts,
		clock:          time.Now,
	}, nil
}

// RunOnce evaluates one action item.
//
// Skip paths (return ResultSkipped, no error):
//   - Empty / whitespace ID
//   - app.ErrNotFound on the action-item lookup
//   - LifecycleState != StateTodo
//   - No agent binding for item.Kind in the project's KindCatalog
//   - Project worktree fields are empty (RepoPrimaryWorktree)
//   - The walker's eligibility predicate fails (blockers not clear, parent
//     not in_progress, etc.)
//
// Block path (return ResultBlocked, no error):
//   - Sibling overlap on paths or packages WITHOUT an explicit blocked_by
//     edge — the conflict detector inserts a runtime blocked_by + raises
//     an attention row, and RunOnce returns immediately without spawning.
//   - File-lock or package-lock conflict with another in-flight holder —
//     also returns Blocked, no spawn.
//
// Spawn path (return ResultSpawned, no error):
//   - Item is eligible, has a valid binding, no overlap, locks acquired.
//   - The action item is promoted to in_progress, BuildSpawnCommand
//     constructs the *exec.Cmd, and the monitor Track() launches the
//     subprocess. The Handle is returned on the outcome; CLI callers do
//     NOT block on Handle.Wait — the manual-trigger milestone exits as
//     soon as the spawn is observed.
//
// Error path (return DispatchOutcome{}, non-nil error):
//   - Database / transport errors during the read or promotion paths.
//   - Eligibility-walk infrastructure failures (ListActionItems,
//     ListColumns).
//   - Spawn-side construction errors that are NOT ErrNoAgentBinding
//     (which is treated as a skip — see above). Examples:
//     ErrInvalidSpawnInput from a corrupted-in-memory binding.
//   - Monitor.Track failures (the cmd.Start syscall) — when this happens
//     after Stage 7's promote, the dispatcher transitions the item to
//     StateFailed (with metadata) and fires the cleanup hook so the action
//     item does NOT linger in_progress. The originating Track error is
//     still wrapped and returned to the caller.
//
// projectIDOverride (4a.23 QA-Falsification §2.2 fix): when non-empty, the
// dispatcher MUST resolve the supplied project_id rather than the action
// item's own ProjectID. If the override mismatches item.ProjectID the
// dispatcher returns ErrProjectMismatch — callers detect via errors.Is.
// An empty override preserves the historical behaviour.
func (d *dispatcher) RunOnce(ctx context.Context, actionItemID, projectIDOverride string) (DispatchOutcome, error) {
	if d == nil {
		return DispatchOutcome{}, fmt.Errorf("%w: dispatcher is nil", ErrInvalidDispatcherConfig)
	}
	trimmed := strings.TrimSpace(actionItemID)
	overrideID := strings.TrimSpace(projectIDOverride)
	now := d.now()
	outcome := DispatchOutcome{
		ActionItemID: trimmed,
		SpawnedAt:    now,
		Result:       ResultSkipped,
	}

	if trimmed == "" {
		outcome.Reason = "empty action item id"
		return outcome, nil
	}

	item, err := d.svc.GetActionItem(ctx, trimmed)
	if err != nil {
		if errors.Is(err, app.ErrNotFound) {
			outcome.Reason = "action item not found"
			return outcome, nil
		}
		return DispatchOutcome{}, fmt.Errorf("dispatcher: get action item %q: %w", trimmed, err)
	}

	if item.LifecycleState != domain.StateTodo {
		outcome.Reason = "not in todo (state=" + string(item.LifecycleState) + ")"
		return outcome, nil
	}

	// --project authoritative-override validation (4a.23 §2.2 fix). When
	// the caller supplies a project_id that does NOT match the action
	// item's own ProjectID, fail loudly rather than silently fall through
	// to the action-item-derived project. The empty-override path skips
	// this gate.
	if overrideID != "" && overrideID != strings.TrimSpace(item.ProjectID) {
		return DispatchOutcome{}, fmt.Errorf("%w: action item %q project_id=%q, override=%q",
			ErrProjectMismatch, item.ID, item.ProjectID, overrideID)
	}

	// Wave 2.10 promotes the dispatcher from skeleton to full RunOnce. The
	// flow below is intentionally linear: each stage either short-circuits
	// (skip / block / error) or proceeds. There are no branches that bypass
	// downstream stages — the only way to reach the spawn is to have passed
	// every prior gate.

	// Stage 1: project resolution. The project carries the spawn-site
	// fields (RepoPrimaryWorktree, KindCatalogJSON, HyllaArtifactRef) and
	// the conflict-detector / walker need ProjectID to scope their sibling
	// reads. Skip when the project lookup fails — the dispatcher does not
	// own project lifecycle and a missing project is a planner-side issue.
	if d.projects == nil {
		return DispatchOutcome{}, fmt.Errorf("%w: project reader is nil", ErrInvalidDispatcherConfig)
	}
	project, err := d.projects.GetProject(ctx, item.ProjectID)
	if err != nil {
		if errors.Is(err, app.ErrNotFound) {
			outcome.Reason = "project not found"
			return outcome, nil
		}
		return DispatchOutcome{}, fmt.Errorf("dispatcher: get project %q: %w", item.ProjectID, err)
	}
	if strings.TrimSpace(project.RepoPrimaryWorktree) == "" {
		outcome.Reason = "project has empty repo_primary_worktree"
		return outcome, nil
	}

	// Stage 2: catalog decode + binding lookup. A missing binding is a
	// skip, not a hard error — the planner may have created a kind the
	// project's catalog does not yet bind, in which case the dispatcher
	// has nothing to spawn. The skip path mirrors the spec's "no agent
	// configured" failure mode.
	//
	// decodeProjectCatalog never returns a non-nil error today (both empty
	// and malformed-JSON paths surface as soft "no catalog" per droplet
	// 3.12). The error return is reserved for future hard-failure paths
	// (e.g. catalog-version pinning); we plumb it through anyway so the
	// signature stays stable.
	catalog, ok, _ := decodeProjectCatalog(project)
	if !ok {
		outcome.Reason = "project has no kind catalog"
		return outcome, nil
	}
	if _, hasBinding := catalog.LookupAgentBinding(item.Kind); !hasBinding {
		outcome.Reason = "no agent binding for kind " + string(item.Kind)
		return outcome, nil
	}

	// Stage 3: walker eligibility predicate. The walker reads the
	// project's full action-item set so the per-item check has the by-ID
	// index it needs (blocked_by, parent state, etc.). A non-eligible
	// item is a skip — the walker enforces the conservative "missing
	// reference treats as not-clear" rule.
	if d.walker == nil {
		return DispatchOutcome{}, fmt.Errorf("%w: walker is nil", ErrInvalidDispatcherConfig)
	}
	siblings, err := d.listing.ListActionItems(ctx, project.ID, false)
	if err != nil {
		return DispatchOutcome{}, fmt.Errorf("dispatcher: list action items for project %q: %w", project.ID, err)
	}
	byID := make(map[string]domain.ActionItem, len(siblings))
	for _, sib := range siblings {
		byID[sib.ID] = sib
	}
	if !d.walker.isEligible(item, byID) {
		outcome.Reason = "walker eligibility predicate not satisfied"
		return outcome, nil
	}

	// Stage 4: sibling-overlap conflict detection. Same-parent overlap on
	// paths or packages WITHOUT an explicit planner-side blocked_by edge
	// inserts a runtime blocked_by + raises an attention row. This
	// produces ResultBlocked and short-circuits the spawn — the next
	// dispatcher tick (after the holder completes) will revisit the
	// candidate.
	parentSiblings := siblingsUnderParent(siblings, item)
	if d.conflict == nil {
		return DispatchOutcome{}, fmt.Errorf("%w: conflict detector is nil", ErrInvalidDispatcherConfig)
	}
	overlaps, err := d.conflict.DetectSiblingOverlap(ctx, item, parentSiblings)
	if err != nil {
		return DispatchOutcome{}, fmt.Errorf("dispatcher: detect sibling overlap for %q: %w", item.ID, err)
	}
	for _, ov := range overlaps {
		if ov.HasExplicitBlockedBy {
			continue
		}
		// Insert a runtime blocked_by pointing at the sibling. The
		// conflict detector handles tie-break + attention-raise; the
		// dispatcher just surfaces ResultBlocked.
		reason := fmt.Sprintf("sibling overlap on %s %q", ov.OverlapKind, ov.OverlapValue)
		if err := d.conflict.InsertRuntimeBlockedBy(ctx, item, ov.SiblingID, reason); err != nil {
			return DispatchOutcome{}, fmt.Errorf("dispatcher: insert runtime blocked_by for %q: %w", item.ID, err)
		}
		outcome.Result = ResultBlocked
		outcome.Reason = reason
		return outcome, nil
	}

	// Stage 5: file + package lock acquisition. Even when sibling-overlap
	// is clean, a cross-subtree holder of the same path/package is the
	// canonical case the locks guard against. A conflict here is also
	// ResultBlocked — the holder is a different action item the walker
	// already promoted; we wait for its terminal state.
	_, fileConflicts, err := d.fileLocks.Acquire(item.ID, item.Paths)
	if err != nil {
		return DispatchOutcome{}, fmt.Errorf("dispatcher: acquire file locks for %q: %w", item.ID, err)
	}
	if len(fileConflicts) > 0 {
		// Roll back the partial acquire so a future tick can retry cleanly.
		d.fileLocks.Release(item.ID)
		outcome.Result = ResultBlocked
		outcome.Reason = "file lock held by another action item"
		return outcome, nil
	}
	_, pkgConflicts, err := d.pkgLocks.Acquire(item.ID, item.Packages)
	if err != nil {
		// Release the file locks we already acquired to keep the manager
		// state symmetric with the action item's terminal state.
		d.fileLocks.Release(item.ID)
		return DispatchOutcome{}, fmt.Errorf("dispatcher: acquire package locks for %q: %w", item.ID, err)
	}
	if len(pkgConflicts) > 0 {
		d.fileLocks.Release(item.ID)
		d.pkgLocks.Release(item.ID)
		outcome.Result = ResultBlocked
		outcome.Reason = "package lock held by another action item"
		return outcome, nil
	}

	// Stage 6: build the spawn command. ErrNoAgentBinding here is
	// defensive — Stage 2 already filtered that case — but a corrupted
	// in-memory binding (Validate trip) can still surface
	// ErrInvalidSpawnInput here. Treat it as a hard error so the dev sees
	// the misconfiguration rather than a silent skip. Stage 6's
	// BuildSpawnCommand re-resolves the binding from catalog internally;
	// no caller-side binding handle is needed here.
	cmd, descriptor, err := BuildSpawnCommand(item, project, catalog, AuthBundle{})
	if err != nil {
		d.fileLocks.Release(item.ID)
		d.pkgLocks.Release(item.ID)
		if errors.Is(err, ErrNoAgentBinding) {
			// Belt-and-suspenders: stage 2 already filtered this; if it
			// re-surfaces, log via Reason and skip rather than fail.
			outcome.Reason = "no agent binding (stage 6 fallback)"
			return outcome, nil
		}
		return DispatchOutcome{}, fmt.Errorf("dispatcher: build spawn command for %q: %w", item.ID, err)
	}

	// Stage 7: promote the action item to in_progress. The walker's
	// Promote method translates ErrTransitionBlocked into
	// ErrPromotionBlocked so callers can detect the planner-side
	// blocker condition with errors.Is. Treat it as ResultBlocked rather
	// than a hard error.
	if _, err := d.walker.Promote(ctx, item); err != nil {
		d.fileLocks.Release(item.ID)
		d.pkgLocks.Release(item.ID)
		if errors.Is(err, ErrPromotionBlocked) {
			outcome.Result = ResultBlocked
			outcome.Reason = "promotion to in_progress rejected by service"
			return outcome, nil
		}
		return DispatchOutcome{}, fmt.Errorf("dispatcher: promote action item %q: %w", item.ID, err)
	}

	// Stage 8: monitor.Track launches the subprocess. The monitor owns
	// cmd.Start lifecycle from this point — callers MUST NOT block on
	// Handle.Wait unless they want to. The CLI returns immediately; the
	// monitor goroutine drives the crash-path transition to StateFailed
	// asynchronously.
	//
	// 4a.23 QA-Falsification §2.1 fix: when monitor.Track fails AFTER
	// Stage 7 already promoted the action item to in_progress, we MUST
	// transition the item to failed (with metadata) and fire the cleanup
	// hook. The pre-fix path released the locks but left the item in
	// in_progress — the walker then skipped it on every subsequent tick,
	// requiring manual DB recovery. Treat a Track failure as a real
	// failure event so the dev sees it and can re-dispatch via supersede
	// later.
	handle, err := d.monitor.Track(ctx, item.ID, cmd)
	if err != nil {
		// Order: transition first (so cleanup fires on a terminal state),
		// then cleanup (releases locks + scrubs monitor map). Cleanup
		// itself releases locks; the explicit lock-release on the original
		// path is preserved as a defensive belt-and-suspenders for the
		// branch where transitionToFailed itself failed and cleanup did
		// not run.
		trackErr := err
		failureReason := "agent process failed to start: " + trackErr.Error()
		if transitionErr := d.transitionToFailed(ctx, item, failureReason); transitionErr != nil {
			// Fail-open: best-effort lock release if the transition
			// itself broke. The action item may still linger in
			// in_progress in this case (the underlying service or DB is
			// degraded), but the dev sees both errors aggregated.
			d.fileLocks.Release(item.ID)
			d.pkgLocks.Release(item.ID)
			return DispatchOutcome{}, fmt.Errorf("dispatcher: track spawn for %q: %w (additionally, failed to transition action item to failed: %v)",
				item.ID, trackErr, transitionErr)
		}
		// Cleanup releases file + package locks AND scrubs the monitor's
		// tracked-PID map. The hook's idempotency-set guards against
		// re-entry from any future state-change observer.
		if d.cleanup != nil {
			failed := item
			failed.LifecycleState = domain.StateFailed
			_ = d.cleanup.OnTerminalState(ctx, failed)
		} else {
			// Defensive: if cleanup is unwired (older test fixtures),
			// release locks directly so the manager state stays symmetric
			// with the now-failed action item.
			d.fileLocks.Release(item.ID)
			d.pkgLocks.Release(item.ID)
		}
		return DispatchOutcome{}, fmt.Errorf("dispatcher: track spawn for %q: %w", item.ID, trackErr)
	}

	outcome.Result = ResultSpawned
	outcome.AgentName = descriptor.AgentName
	outcome.Handle = handle
	return outcome, nil
}

// transitionToFailed moves item to its project's failed column and writes
// the metadata shape the monitor's crash-path writes (Outcome="failure",
// BlockedReason=reason). Used by RunOnce's Stage 8 rollback to make the
// "failed-to-start" event symmetric with the monitor's "crashed-mid-run"
// event — same canonical row shape, same cleanup-hook trigger.
//
// Returns nil when the mutator seam is unwired (older test fixtures that
// never reach Stage 8); production wiring always supplies the seam.
func (d *dispatcher) transitionToFailed(ctx context.Context, item domain.ActionItem, reason string) error {
	if d == nil || d.mutator == nil {
		return nil
	}
	columns, err := d.mutator.ListColumns(ctx, item.ProjectID, true)
	if err != nil {
		return fmt.Errorf("transitionToFailed: list columns for project %q: %w", item.ProjectID, err)
	}
	failedColumnID := columnIDForLifecycleState(columns, domain.StateFailed)
	if failedColumnID == "" {
		return fmt.Errorf("transitionToFailed: project %q has no failed column", item.ProjectID)
	}
	if _, err := d.mutator.MoveActionItem(ctx, item.ID, failedColumnID, item.Position); err != nil {
		return fmt.Errorf("transitionToFailed: move action item %q to failed: %w", item.ID, err)
	}
	updated := item.Metadata
	updated.Outcome = "failure"
	updated.BlockedReason = "dispatcher: " + reason
	if _, err := d.mutator.UpdateActionItem(ctx, updateActionItemInput{
		ActionItemID: item.ID,
		Metadata:     &updated,
	}); err != nil {
		return fmt.Errorf("transitionToFailed: update action item %q metadata: %w", item.ID, err)
	}
	return nil
}

// SpawnPreview is the rich --dry-run result returned by PreviewSpawn. It
// extends the bare SpawnDescriptor with the gates RunOnce evaluates between
// catalog-resolution and spawn — eligibility predicate, sibling-overlap
// detector, and lock-availability check — so the dev sees ahead-of-time
// whether the live RunOnce would have been ResultSpawned, ResultBlocked, or
// ResultSkipped. 4a.23 QA-Falsification §2.3 fix: pre-fix PreviewSpawn
// stopped at BuildSpawnCommand, hiding ineligibility / overlap / lock
// blockers behind a successful descriptor JSON.
//
// Eligible reflects the walker's predicate. When false, Reason names the
// gate that closed (parent-not-in-progress, blocker-not-clear, etc.) and
// the descriptor is still constructed so dev scripts can inspect the
// would-have-been argv shape.
//
// Overlaps is the detector's same-parent overlap report; entries with
// HasExplicitBlockedBy=true are informational (the planner already wired
// the dependency). Entries with HasExplicitBlockedBy=false would trigger
// runtime-blocker insertion in the live RunOnce path.
//
// FileLockConflicts / PackageLockConflicts are non-mutating snapshots from
// fileLockManager.WouldConflict / packageLockManager.WouldConflict — they
// reflect the in-process lock state at the moment of the dry-run only
// (no reservation made; another spawn could acquire between dry-run and
// live RunOnce).
type SpawnPreview struct {
	Descriptor           SpawnDescriptor
	Eligible             bool
	Reason               string
	Overlaps             []SiblingOverlap
	FileLockConflicts    map[string]string
	PackageLockConflicts map[string]string
}

// PreviewSpawn returns a SpawnPreview describing what the dispatcher would
// do for the supplied action item without executing the spawn or mutating
// any state. It is the manual-trigger CLI's --dry-run entry point: walk
// the same project + catalog + binding + eligibility + conflict +
// lock-availability checks as RunOnce, build the spawn command, and return
// the full snapshot. No locks acquired, no walker promote, no monitor
// track.
//
// 4a.23 QA-Falsification §2.3 fix: PreviewSpawn now walks Stages 1-5 of
// the live pipeline read-only (Stage 5's lock acquire is replaced with the
// non-mutating WouldConflict variant) before constructing the descriptor.
// The result's Eligible / Overlaps / FileLockConflicts /
// PackageLockConflicts fields surface block-state information the
// pre-fix shape silently dropped.
//
// PreviewSpawn returns the same skip / error vocabulary as RunOnce on the
// resolution path so the CLI surfaces a consistent message regardless of
// whether the dev ran it dry or live. Returns the same wrapped errors
// (errors.Is(err, app.ErrNotFound), errors.Is(err, ErrNoAgentBinding),
// errors.Is(err, ErrProjectMismatch)) so callers can distinguish skips
// from infrastructure failures.
//
// projectIDOverride matches the RunOnce override semantics: a non-empty
// value MUST match item.ProjectID; mismatch returns ErrProjectMismatch.
func (d *dispatcher) PreviewSpawn(ctx context.Context, actionItemID, projectIDOverride string) (SpawnPreview, domain.ActionItem, domain.Project, error) {
	if d == nil {
		return SpawnPreview{}, domain.ActionItem{}, domain.Project{}, fmt.Errorf("%w: dispatcher is nil", ErrInvalidDispatcherConfig)
	}
	trimmed := strings.TrimSpace(actionItemID)
	overrideID := strings.TrimSpace(projectIDOverride)
	if trimmed == "" {
		return SpawnPreview{}, domain.ActionItem{}, domain.Project{}, fmt.Errorf("%w: action item id is empty", ErrInvalidSpawnInput)
	}
	item, err := d.svc.GetActionItem(ctx, trimmed)
	if err != nil {
		return SpawnPreview{}, domain.ActionItem{}, domain.Project{}, fmt.Errorf("dispatcher: get action item %q: %w", trimmed, err)
	}
	if overrideID != "" && overrideID != strings.TrimSpace(item.ProjectID) {
		return SpawnPreview{}, item, domain.Project{}, fmt.Errorf("%w: action item %q project_id=%q, override=%q",
			ErrProjectMismatch, item.ID, item.ProjectID, overrideID)
	}
	project, err := d.projects.GetProject(ctx, item.ProjectID)
	if err != nil {
		return SpawnPreview{}, item, domain.Project{}, fmt.Errorf("dispatcher: get project %q: %w", item.ProjectID, err)
	}
	catalog, ok, _ := decodeProjectCatalog(project)
	if !ok {
		return SpawnPreview{}, item, project, fmt.Errorf("%w: project %q has no kind catalog", ErrNoAgentBinding, project.ID)
	}
	_, descriptor, err := BuildSpawnCommand(item, project, catalog, AuthBundle{})
	if err != nil {
		return SpawnPreview{}, item, project, err
	}

	preview := SpawnPreview{
		Descriptor:           descriptor,
		Eligible:             true,
		FileLockConflicts:    map[string]string{},
		PackageLockConflicts: map[string]string{},
	}

	// Stage 3 — walker eligibility. Read-only; same byID index RunOnce
	// builds. A non-todo state (e.g. already in_progress / complete) is
	// reflected here: walker.isEligible filters non-todo at the entry
	// gate.
	if d.walker != nil && d.listing != nil {
		siblings, listErr := d.listing.ListActionItems(ctx, project.ID, false)
		if listErr != nil {
			return preview, item, project, fmt.Errorf("dispatcher: list action items for project %q: %w", project.ID, listErr)
		}
		byID := make(map[string]domain.ActionItem, len(siblings))
		for _, sib := range siblings {
			byID[sib.ID] = sib
		}
		if !d.walker.isEligible(item, byID) {
			preview.Eligible = false
			preview.Reason = walkerIneligibilityReason(item, byID)
		}

		// Stage 4 — sibling-overlap detector. Read-only; never inserts a
		// runtime blocker in the dry-run path.
		if d.conflict != nil {
			overlaps, conflictErr := d.conflict.DetectSiblingOverlap(ctx, item, siblingsUnderParent(siblings, item))
			if conflictErr != nil {
				return preview, item, project, fmt.Errorf("dispatcher: detect sibling overlap for %q: %w", item.ID, conflictErr)
			}
			preview.Overlaps = overlaps
			if preview.Eligible {
				for _, ov := range overlaps {
					if ov.HasExplicitBlockedBy {
						continue
					}
					preview.Eligible = false
					preview.Reason = fmt.Sprintf("sibling overlap on %s %q", ov.OverlapKind, ov.OverlapValue)
					break
				}
			}
		}
	}

	// Stage 5 — lock-availability snapshot. WouldConflict on each manager
	// is the non-mutating variant of Acquire.
	if d.fileLocks != nil {
		preview.FileLockConflicts = d.fileLocks.WouldConflict(item.ID, item.Paths)
		if preview.Eligible && len(preview.FileLockConflicts) > 0 {
			preview.Eligible = false
			preview.Reason = "file lock held by another action item"
		}
	}
	if d.pkgLocks != nil {
		preview.PackageLockConflicts = d.pkgLocks.WouldConflict(item.ID, item.Packages)
		if preview.Eligible && len(preview.PackageLockConflicts) > 0 {
			preview.Eligible = false
			preview.Reason = "package lock held by another action item"
		}
	}

	return preview, item, project, nil
}

// walkerIneligibilityReason renders a human-readable cause for the walker's
// negative isEligible verdict. Mirrors the walker's predicate order so the
// reason names the gate that closed (non-todo state, missing/incomplete
// blocker, parent not-in-progress).
func walkerIneligibilityReason(item domain.ActionItem, byID map[string]domain.ActionItem) string {
	if item.LifecycleState != domain.StateTodo {
		return "not in todo (state=" + string(item.LifecycleState) + ")"
	}
	for _, blockerID := range item.Metadata.BlockedBy {
		bid := strings.TrimSpace(blockerID)
		blocker, ok := byID[bid]
		if !ok {
			return "blocked_by references unknown action item " + bid
		}
		if blocker.LifecycleState != domain.StateComplete {
			return "blocked_by " + bid + " is " + string(blocker.LifecycleState) + " (need complete)"
		}
	}
	parentID := strings.TrimSpace(item.ParentID)
	if parentID == "" {
		return "walker eligibility predicate not satisfied"
	}
	parent, ok := byID[parentID]
	if !ok {
		return "parent " + parentID + " missing from project tree"
	}
	if !parent.Persistent && parent.LifecycleState != domain.StateInProgress {
		return "parent " + parentID + " is " + string(parent.LifecycleState) + " (need in_progress)"
	}
	return "walker eligibility predicate not satisfied"
}

// Start / Stop bodies live in subscriber.go (Drop 4b.7). They replace the
// pre-Drop-4b.7 ErrNotImplemented stubs with real continuous-mode wiring.

// now returns the dispatcher's current time, defaulting to time.Now when no
// clock has been injected.
func (d *dispatcher) now() time.Time {
	if d == nil || d.clock == nil {
		return time.Now()
	}
	return d.clock()
}

// decodeProjectCatalog unmarshals the project's KindCatalogJSON envelope
// into a templates.KindCatalog. Returns (zero, false, nil) when the envelope
// is empty or malformed — mirroring the soft-fallback rule per Drop 3
// droplet 3.12: a malformed envelope must not brick the dispatcher path.
// Callers treat the false return as "no catalog available, skip the spawn".
func decodeProjectCatalog(project domain.Project) (templates.KindCatalog, bool, error) {
	if len(project.KindCatalogJSON) == 0 {
		return templates.KindCatalog{}, false, nil
	}
	var catalog templates.KindCatalog
	if err := json.Unmarshal(project.KindCatalogJSON, &catalog); err != nil {
		// Soft fallback per droplet 3.12: a malformed envelope is treated
		// as "no catalog" rather than a hard failure. The dev sees the
		// "no kind catalog" skip reason; the underlying decode error is
		// not surfaced because the action item is not the right place to
		// log it.
		return templates.KindCatalog{}, false, nil
	}
	return catalog, true, nil
}

// siblingsUnderParent filters siblings to the same-parent set (per the
// conflict detector's same-parent contract) and excludes the candidate
// itself. The detector applies the same filter defensively, but pre-filtering
// here keeps the slice small and self-documenting at the call site.
func siblingsUnderParent(all []domain.ActionItem, item domain.ActionItem) []domain.ActionItem {
	out := make([]domain.ActionItem, 0, len(all))
	for _, sib := range all {
		if sib.ID == item.ID {
			continue
		}
		if sib.ParentID != item.ParentID {
			continue
		}
		out = append(out, sib)
	}
	return out
}

// Compile-time assertion that *dispatcher satisfies the Dispatcher interface.
var _ Dispatcher = (*dispatcher)(nil)
