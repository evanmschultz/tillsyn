package dispatcher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/evanmschultz/tillsyn/internal/domain"
	"github.com/evanmschultz/tillsyn/internal/templates"
)

// stubActionItemReader is a deterministic test stub for the dispatcher's
// action-item lookup dependency. The test suite injects one of these into a
// dispatcher constructed via the package-internal struct literal so RunOnce
// scenarios can be exercised without a full Service + Repository graph.
type stubActionItemReader struct {
	wantID string
	item   domain.ActionItem
	err    error
	calls  int
}

// GetActionItem records the call and returns the configured fixture.
func (s *stubActionItemReader) GetActionItem(_ context.Context, actionItemID string) (domain.ActionItem, error) {
	s.calls++
	s.wantID = actionItemID
	if s.err != nil {
		return domain.ActionItem{}, s.err
	}
	return s.item, nil
}

// stubProjectReader returns app.ErrNotFound by default so the dispatcher's
// stage-1 project resolution short-circuits to ResultSkipped. Tests that need
// a populated project assign one to the project field.
type stubProjectReader struct {
	project domain.Project
	err     error
}

func (s *stubProjectReader) GetProject(_ context.Context, _ string) (domain.Project, error) {
	if s.err != nil {
		return domain.Project{}, s.err
	}
	return s.project, nil
}

// stubListingService satisfies the dispatcher's listingService interface with
// empty results — sufficient for the skip-path RunOnce scenarios that never
// reach the walker / conflict-detector stages.
type stubListingService struct{}

func (stubListingService) ListColumns(_ context.Context, _ string, _ bool) ([]domain.Column, error) {
	return nil, nil
}

func (stubListingService) ListActionItems(_ context.Context, _ string, _ bool) ([]domain.ActionItem, error) {
	return nil, nil
}

// newServiceForConstructorTest returns one *app.Service that is shape-valid
// for NewDispatcher's nil-check. The service is never invoked through the
// dispatcher in constructor tests (those exercise validation only); RunOnce
// tests use the dispatcher{} struct-literal path with a stub reader.
func newServiceForConstructorTest() *app.Service {
	return app.NewService(nil, nil, nil, app.ServiceConfig{})
}

// newBrokerForTest returns one in-process broker for tests.
func newBrokerForTest() app.LiveWaitBroker {
	return app.NewInProcessLiveWaitBroker()
}

// TestNewDispatcherRejectsNilService asserts the constructor wraps
// ErrInvalidDispatcherConfig when svc is nil.
func TestNewDispatcherRejectsNilService(t *testing.T) {
	t.Parallel()

	d, err := NewDispatcher(nil, newBrokerForTest(), Options{})
	if err == nil {
		t.Fatalf("NewDispatcher(nil svc) error = nil, want non-nil")
	}
	if !errors.Is(err, ErrInvalidDispatcherConfig) {
		t.Fatalf("NewDispatcher(nil svc) error = %v, want errors.Is(ErrInvalidDispatcherConfig)", err)
	}
	if d != nil {
		t.Fatalf("NewDispatcher(nil svc) dispatcher = %v, want nil", d)
	}
}

// TestNewDispatcherRejectsNilBroker asserts the constructor wraps
// ErrInvalidDispatcherConfig when broker is nil.
func TestNewDispatcherRejectsNilBroker(t *testing.T) {
	t.Parallel()

	d, err := NewDispatcher(newServiceForConstructorTest(), nil, Options{})
	if err == nil {
		t.Fatalf("NewDispatcher(nil broker) error = nil, want non-nil")
	}
	if !errors.Is(err, ErrInvalidDispatcherConfig) {
		t.Fatalf("NewDispatcher(nil broker) error = %v, want errors.Is(ErrInvalidDispatcherConfig)", err)
	}
	if d != nil {
		t.Fatalf("NewDispatcher(nil broker) dispatcher = %v, want nil", d)
	}
}

// TestNewDispatcherSucceedsWithValidArgs asserts the constructor returns a
// non-nil dispatcher when both dependencies are non-nil.
func TestNewDispatcherSucceedsWithValidArgs(t *testing.T) {
	t.Parallel()

	d, err := NewDispatcher(newServiceForConstructorTest(), newBrokerForTest(), Options{})
	if err != nil {
		t.Fatalf("NewDispatcher() error = %v, want nil", err)
	}
	if d == nil {
		t.Fatalf("NewDispatcher() dispatcher = nil, want non-nil")
	}
	// Compile-time assertion below in dispatcher.go also catches this; the
	// runtime check guards against future refactors that drop the
	// interface-satisfaction guarantee.
	var _ Dispatcher = d
}

// TestRunOnceSkipsEmptyActionItemID asserts that an empty/whitespace ID
// returns ResultSkipped without consulting the service.
func TestRunOnceSkipsEmptyActionItemID(t *testing.T) {
	t.Parallel()

	stub := &stubActionItemReader{}
	d := newDispatcherForTest(stub)

	outcome, err := d.RunOnce(context.Background(), "   ", "")
	if err != nil {
		t.Fatalf("RunOnce() error = %v, want nil", err)
	}
	if outcome.Result != ResultSkipped {
		t.Fatalf("RunOnce() Result = %q, want %q", outcome.Result, ResultSkipped)
	}
	if outcome.ActionItemID != "" {
		t.Fatalf("RunOnce() ActionItemID = %q, want empty", outcome.ActionItemID)
	}
	if stub.calls != 0 {
		t.Fatalf("stub.calls = %d, want 0 (empty ID short-circuits)", stub.calls)
	}
}

// TestRunOnceSkipsNonExistentActionItem asserts that ErrNotFound from the
// service surfaces as ResultSkipped.
func TestRunOnceSkipsNonExistentActionItem(t *testing.T) {
	t.Parallel()

	stub := &stubActionItemReader{err: app.ErrNotFound}
	d := newDispatcherForTest(stub)

	outcome, err := d.RunOnce(context.Background(), "missing-id", "")
	if err != nil {
		t.Fatalf("RunOnce() error = %v, want nil", err)
	}
	if outcome.Result != ResultSkipped {
		t.Fatalf("RunOnce() Result = %q, want %q", outcome.Result, ResultSkipped)
	}
	if outcome.ActionItemID != "missing-id" {
		t.Fatalf("RunOnce() ActionItemID = %q, want %q", outcome.ActionItemID, "missing-id")
	}
	if stub.calls != 1 {
		t.Fatalf("stub.calls = %d, want 1", stub.calls)
	}
	if stub.wantID != "missing-id" {
		t.Fatalf("stub forwarded ID = %q, want %q", stub.wantID, "missing-id")
	}
}

// TestRunOnceSkipsNonTodoActionItem asserts that an action item in any
// non-todo lifecycle state surfaces as ResultSkipped.
func TestRunOnceSkipsNonTodoActionItem(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		state domain.LifecycleState
	}{
		{name: "in_progress", state: domain.StateInProgress},
		{name: "complete", state: domain.StateComplete},
		{name: "failed", state: domain.StateFailed},
		{name: "archived", state: domain.StateArchived},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			stub := &stubActionItemReader{
				item: domain.ActionItem{
					ID:             "ai-1",
					LifecycleState: tc.state,
				},
			}
			d := newDispatcherForTest(stub)

			outcome, err := d.RunOnce(context.Background(), "ai-1", "")
			if err != nil {
				t.Fatalf("RunOnce() error = %v, want nil", err)
			}
			if outcome.Result != ResultSkipped {
				t.Fatalf("RunOnce() Result = %q, want %q (state=%q)", outcome.Result, ResultSkipped, tc.state)
			}
			if outcome.ActionItemID != "ai-1" {
				t.Fatalf("RunOnce() ActionItemID = %q, want %q", outcome.ActionItemID, "ai-1")
			}
		})
	}
}

// TestRunOnceTodoActionItemSkipsWhenProjectMissing asserts that a todo item
// whose project lookup returns app.ErrNotFound surfaces as ResultSkipped
// rather than a hard error. Replaces the Wave 2.1 "skeleton always skips"
// pin per droplet 4a.23: RunOnce now walks past the non-todo gate, and the
// project-resolution stage is the first skip path the test reaches. The
// helper's default stub project reader returns ErrNotFound so this test
// pins that path explicitly.
func TestRunOnceTodoActionItemSkipsWhenProjectMissing(t *testing.T) {
	t.Parallel()

	stub := &stubActionItemReader{
		item: domain.ActionItem{
			ID:             "ai-todo",
			ProjectID:      "proj-missing",
			LifecycleState: domain.StateTodo,
		},
	}
	d := newDispatcherForTest(stub)

	outcome, err := d.RunOnce(context.Background(), "ai-todo", "")
	if err != nil {
		t.Fatalf("RunOnce() error = %v, want nil", err)
	}
	if outcome.Result != ResultSkipped {
		t.Fatalf("RunOnce() Result = %q, want %q (project missing -> skip)", outcome.Result, ResultSkipped)
	}
	if outcome.Reason == "" {
		t.Errorf("RunOnce() Reason is empty; want a human-readable skip explanation")
	}
}

// TestRunOncePropagatesUnexpectedServiceError asserts that errors other than
// app.ErrNotFound bubble up to the caller wrapped with context.
func TestRunOncePropagatesUnexpectedServiceError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("database closed")
	stub := &stubActionItemReader{err: wantErr}
	d := newDispatcherForTest(stub)

	_, err := d.RunOnce(context.Background(), "ai-x", "")
	if err == nil {
		t.Fatalf("RunOnce() error = nil, want %v", wantErr)
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("RunOnce() error = %v, want errors.Is(%v)", err, wantErr)
	}
}

// TestRunOnceTrimsActionItemIDWhitespace asserts the dispatcher trims
// whitespace before the lookup, so " ai-1 " and "ai-1" share one path.
func TestRunOnceTrimsActionItemIDWhitespace(t *testing.T) {
	t.Parallel()

	stub := &stubActionItemReader{
		item: domain.ActionItem{
			ID:             "ai-trim",
			LifecycleState: domain.StateInProgress,
		},
	}
	d := newDispatcherForTest(stub)

	outcome, err := d.RunOnce(context.Background(), "  ai-trim  ", "")
	if err != nil {
		t.Fatalf("RunOnce() error = %v, want nil", err)
	}
	if outcome.ActionItemID != "ai-trim" {
		t.Fatalf("RunOnce() ActionItemID = %q, want %q", outcome.ActionItemID, "ai-trim")
	}
	if stub.wantID != "ai-trim" {
		t.Fatalf("stub forwarded ID = %q, want %q (whitespace must be trimmed)", stub.wantID, "ai-trim")
	}
}

// TestRunOnceUsesInjectedClock asserts that a dispatcher with a fixed clock
// emits SpawnedAt aligned with that clock — important so the future
// continuous-mode loop can deterministically order outcomes.
func TestRunOnceUsesInjectedClock(t *testing.T) {
	t.Parallel()

	fixed := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	stub := &stubActionItemReader{
		item: domain.ActionItem{
			ID:             "ai-clock",
			LifecycleState: domain.StateComplete,
		},
	}
	d := &dispatcher{
		svc:    stub,
		broker: newBrokerForTest(),
		clock:  func() time.Time { return fixed },
	}

	outcome, err := d.RunOnce(context.Background(), "ai-clock", "")
	if err != nil {
		t.Fatalf("RunOnce() error = %v, want nil", err)
	}
	if !outcome.SpawnedAt.Equal(fixed) {
		t.Fatalf("RunOnce() SpawnedAt = %v, want %v", outcome.SpawnedAt, fixed)
	}
}

// TestRunOnceNilDispatcherReturnsConfigError asserts a nil-receiver call
// surfaces as ErrInvalidDispatcherConfig rather than panicking. This is
// defense-in-depth: NewDispatcher never returns nil with nil error, but
// future refactors might construct dispatcher values through other paths.
func TestRunOnceNilDispatcherReturnsConfigError(t *testing.T) {
	t.Parallel()

	var d *dispatcher
	_, err := d.RunOnce(context.Background(), "ai-1", "")
	if err == nil {
		t.Fatalf("nil dispatcher RunOnce() error = nil, want non-nil")
	}
	if !errors.Is(err, ErrInvalidDispatcherConfig) {
		t.Fatalf("nil dispatcher RunOnce() error = %v, want errors.Is(ErrInvalidDispatcherConfig)", err)
	}
}

// TestStartRequiresWalker asserts the Drop 4b.7 Start path rejects a
// dispatcher missing the walker dependency with ErrInvalidDispatcherConfig.
// Replaces the pre-Drop-4b.7 ErrNotImplemented stub assertion. The
// newDispatcherForTest helper does not wire walker / projectsLister so this
// test lands on the first nil-check.
func TestStartRequiresWalker(t *testing.T) {
	t.Parallel()

	d := newDispatcherForTest(&stubActionItemReader{})
	err := d.Start(context.Background())
	if err == nil {
		t.Fatalf("Start() error = nil, want ErrInvalidDispatcherConfig")
	}
	if !errors.Is(err, ErrInvalidDispatcherConfig) {
		t.Fatalf("Start() error = %v, want errors.Is(ErrInvalidDispatcherConfig)", err)
	}
}

// TestStopOnUnstartedDispatcherReturnsNil asserts the Drop 4b.7 Stop path
// returns nil when called on a dispatcher that was never started — Stop is
// idempotent across the unstarted / started / stopped lifecycle states.
// Replaces the pre-Drop-4b.7 ErrNotImplemented stub assertion.
func TestStopOnUnstartedDispatcherReturnsNil(t *testing.T) {
	t.Parallel()

	d := newDispatcherForTest(&stubActionItemReader{})
	if err := d.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() on unstarted dispatcher error = %v, want nil", err)
	}
}

// TestResultEnumValuesAreStable pins the closed Result enum's wire values so
// downstream consumers (CLI output, logs, dashboards) detect any future
// rename as a deliberate edit rather than a silent regression.
func TestResultEnumValuesAreStable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		got  Result
		want string
	}{
		{ResultSpawned, "spawned"},
		{ResultSkipped, "skipped"},
		{ResultBlocked, "blocked"},
		{ResultFailed, "failed"},
	}
	for _, tc := range cases {
		if string(tc.got) != tc.want {
			t.Errorf("Result %q = %q, want %q", tc.want, string(tc.got), tc.want)
		}
	}
}

// TestErrInvalidDispatcherConfigWraps confirms the sentinel formats with
// fmt.Errorf-style %w wrapping for callers that pattern-match on the wrapped
// reason string.
func TestErrInvalidDispatcherConfigWraps(t *testing.T) {
	t.Parallel()

	wrapped := fmt.Errorf("%w: extra context", ErrInvalidDispatcherConfig)
	if !errors.Is(wrapped, ErrInvalidDispatcherConfig) {
		t.Fatalf("errors.Is(wrapped, ErrInvalidDispatcherConfig) = false, want true")
	}
}

// newDispatcherForTest constructs one dispatcher via the unexported struct
// literal so tests can swap in a stub actionItemReader. Production code
// always goes through NewDispatcher. The helper attaches default stubs for
// projects + listing so the RunOnce stages past the non-todo gate exit on
// the project-not-found / no-catalog skip paths rather than nil-deref.
func newDispatcherForTest(reader actionItemReader) *dispatcher {
	return &dispatcher{
		svc:      reader,
		projects: &stubProjectReader{err: app.ErrNotFound},
		listing:  stubListingService{},
		broker:   newBrokerForTest(),
		clock:    time.Now,
	}
}

// TestRunOnceRejectsProjectMismatch asserts the authoritative-override path
// (4a.23 QA-Falsification §2.2 fix): a non-empty projectIDOverride that
// does NOT match the action item's own ProjectID returns
// ErrProjectMismatch. The dispatcher MUST NOT silently fall through to the
// item-derived project. Callers detect via errors.Is.
func TestRunOnceRejectsProjectMismatch(t *testing.T) {
	t.Parallel()

	stub := &stubActionItemReader{
		item: domain.ActionItem{
			ID:             "ai-mismatch",
			ProjectID:      "proj-actual",
			LifecycleState: domain.StateTodo,
		},
	}
	d := newDispatcherForTest(stub)

	_, err := d.RunOnce(context.Background(), "ai-mismatch", "proj-other")
	if err == nil {
		t.Fatalf("RunOnce(override mismatch) error = nil, want ErrProjectMismatch")
	}
	if !errors.Is(err, ErrProjectMismatch) {
		t.Fatalf("RunOnce(override mismatch) error = %v, want errors.Is(ErrProjectMismatch)", err)
	}
}

// TestRunOnceHonorsMatchingProjectOverride asserts the authoritative-override
// path proceeds when override matches the action item's own ProjectID. The
// stub project returns ErrNotFound so the run still skips downstream — the
// pin here is that the override-match gate did NOT short-circuit before
// project resolution.
func TestRunOnceHonorsMatchingProjectOverride(t *testing.T) {
	t.Parallel()

	stub := &stubActionItemReader{
		item: domain.ActionItem{
			ID:             "ai-match",
			ProjectID:      "proj-shared",
			LifecycleState: domain.StateTodo,
		},
	}
	d := newDispatcherForTest(stub)

	outcome, err := d.RunOnce(context.Background(), "ai-match", "proj-shared")
	if err != nil {
		t.Fatalf("RunOnce(override match) error = %v, want nil", err)
	}
	// Default fixture has projects -> ErrNotFound, so we expect skip with a
	// project-resolution reason, not the mismatch error.
	if outcome.Result != ResultSkipped {
		t.Fatalf("RunOnce(override match) Result = %q, want %q", outcome.Result, ResultSkipped)
	}
	if outcome.Reason == "" {
		t.Errorf("RunOnce(override match) Reason is empty; expected project-not-found skip reason")
	}
}

// richDispatchService is the unified test stub used by the rollback /
// lock-conflict tests below. It combines the actionItemReader, projectReader,
// listingService, walkerService, conflictDetectorService, and
// failureTransitioner / monitorService surfaces so a single instance drives
// the dispatcher through all 8 RunOnce stages.
//
// The stub records every mutating call so the rollback assertions can pin
// (a) walker.Promote moved the item to in_progress, and (b) the Stage 8
// rollback transitioned it to failed with metadata.
type richDispatchService struct {
	mu sync.Mutex

	item    domain.ActionItem
	project domain.Project
	columns []domain.Column

	// movedTo records every column ID the dispatcher moved the item to,
	// in call order. The first entry is in_progress (from walker.Promote);
	// the second (when present) is failed (from Stage 8 rollback).
	movedTo []string
	// updateCalls captures every metadata-update payload (used by the
	// failure-transition write).
	updateCalls []app.UpdateActionItemInput
}

func (s *richDispatchService) GetActionItem(_ context.Context, _ string) (domain.ActionItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.item, nil
}

func (s *richDispatchService) GetProject(_ context.Context, _ string) (domain.Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.project, nil
}

func (s *richDispatchService) ListColumns(_ context.Context, _ string, _ bool) ([]domain.Column, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]domain.Column(nil), s.columns...), nil
}

func (s *richDispatchService) ListActionItems(_ context.Context, _ string, _ bool) ([]domain.ActionItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return []domain.ActionItem{s.item}, nil
}

func (s *richDispatchService) MoveActionItem(_ context.Context, actionItemID, toColumnID string, _ int) (domain.ActionItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.movedTo = append(s.movedTo, toColumnID)
	switch toColumnID {
	case "col-inprogress":
		s.item.LifecycleState = domain.StateInProgress
	case "col-failed":
		s.item.LifecycleState = domain.StateFailed
	}
	return s.item, nil
}

func (s *richDispatchService) UpdateActionItem(_ context.Context, in app.UpdateActionItemInput) (domain.ActionItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updateCalls = append(s.updateCalls, in)
	if in.Metadata != nil {
		s.item.Metadata = *in.Metadata
	}
	return s.item, nil
}

// dispatcherSvcAdapter adapts richDispatchService to the dispatcher's narrow
// updateActionItemInput shape. The production wiring goes through
// monitorServiceAdapter; tests reuse the same shape so the rollback path
// exercises the same translation seam.
type dispatcherSvcAdapter struct {
	inner *richDispatchService
}

func (a dispatcherSvcAdapter) ListColumns(ctx context.Context, projectID string, includeArchived bool) ([]domain.Column, error) {
	return a.inner.ListColumns(ctx, projectID, includeArchived)
}

func (a dispatcherSvcAdapter) MoveActionItem(ctx context.Context, actionItemID, toColumnID string, position int) (domain.ActionItem, error) {
	return a.inner.MoveActionItem(ctx, actionItemID, toColumnID, position)
}

func (a dispatcherSvcAdapter) UpdateActionItem(ctx context.Context, in updateActionItemInput) (domain.ActionItem, error) {
	wide := app.UpdateActionItemInput{
		ActionItemID: in.ActionItemID,
		Metadata:     in.Metadata,
		UpdatedType:  domain.ActorTypeSystem,
	}
	return a.inner.UpdateActionItem(ctx, wide)
}

// noopAuthRevoker satisfies the cleanup hook's auth-revoke seam for
// dispatcher tests that do not exercise the auth-revoke path. The richer
// auth-revoke assertions live in cleanup_test.go's stubAuthRevoker; here
// the no-op shape keeps the rich-dispatcher fixture focused on walker /
// conflict / monitor wiring without depending on a Service stack.
type noopAuthRevoker struct{}

// RevokeSessionForActionItem returns nil — the rich-dispatcher tests do not
// exercise the auth-revoke step.
func (noopAuthRevoker) RevokeSessionForActionItem(_ context.Context, _ string) error {
	return nil
}

// newRichDispatcherForTest wires a dispatcher with a real walker, conflict
// detector, file/package lock managers, monitor, and cleanup hook bound to
// rich. The mutator seam is wired so transitionToFailed has somewhere to go.
func newRichDispatcherForTest(t *testing.T, rich *richDispatchService) *dispatcher {
	t.Helper()
	walker := newTreeWalker(rich)
	conflict := newConflictDetector(&stubConflictService{})
	fileLocks := newFileLockManager()
	pkgLocks := newPackageLockManager()
	monitor := newProcessMonitor(richMonitorAdapter{inner: rich}, nil)
	cleanup, err := newCleanupHook(fileLocks, pkgLocks, monitor, noopAuthRevoker{})
	if err != nil {
		t.Fatalf("newCleanupHook() error = %v", err)
	}
	return &dispatcher{
		svc:       rich,
		projects:  rich,
		listing:   rich,
		mutator:   dispatcherSvcAdapter{inner: rich},
		broker:    newBrokerForTest(),
		walker:    walker,
		conflict:  conflict,
		fileLocks: fileLocks,
		pkgLocks:  pkgLocks,
		monitor:   monitor,
		cleanup:   cleanup,
		clock:     time.Now,
	}
}

// richMonitorAdapter satisfies the monitorService interface on
// richDispatchService. Independent from monitorServiceAdapter because the
// inner type differs (rich vs *app.Service); the field set we translate is
// identical.
type richMonitorAdapter struct {
	inner *richDispatchService
}

func (a richMonitorAdapter) GetActionItem(ctx context.Context, actionItemID string) (domain.ActionItem, error) {
	return a.inner.GetActionItem(ctx, actionItemID)
}

func (a richMonitorAdapter) ListColumns(ctx context.Context, projectID string, includeArchived bool) ([]domain.Column, error) {
	return a.inner.ListColumns(ctx, projectID, includeArchived)
}

func (a richMonitorAdapter) MoveActionItem(ctx context.Context, actionItemID, toColumnID string, position int) (domain.ActionItem, error) {
	return a.inner.MoveActionItem(ctx, actionItemID, toColumnID, position)
}

func (a richMonitorAdapter) UpdateActionItem(ctx context.Context, in updateActionItemInput) (domain.ActionItem, error) {
	wide := app.UpdateActionItemInput{
		ActionItemID: in.ActionItemID,
		Metadata:     in.Metadata,
		UpdatedType:  domain.ActorTypeSystem,
	}
	return a.inner.UpdateActionItem(ctx, wide)
}

// buildRichFixture seeds a richDispatchService with a single eligible action
// item, a project carrying a baked KindCatalog (kind=build → go-builder-agent),
// and the four canonical columns the walker/monitor resolve on.
func buildRichFixture(t *testing.T) *richDispatchService {
	t.Helper()
	tpl := templates.Template{
		SchemaVersion: templates.SchemaVersionV1,
		Kinds: map[domain.Kind]templates.KindRule{
			domain.KindBuild: {StructuralType: domain.StructuralTypeDroplet},
		},
		AgentBindings: map[domain.Kind]templates.AgentBinding{
			domain.KindBuild: {
				AgentName:    "go-builder-agent",
				Model:        "opus",
				MaxTries:     1,
				MaxBudgetUSD: 5,
				MaxTurns:     20,
			},
		},
	}
	encoded, err := json.Marshal(templates.Bake(tpl))
	if err != nil {
		t.Fatalf("json.Marshal(catalog) error = %v", err)
	}
	worktree := t.TempDir()
	return &richDispatchService{
		item: domain.ActionItem{
			ID:             "ai-stage-test",
			ProjectID:      "proj-rich",
			Kind:           domain.KindBuild,
			LifecycleState: domain.StateTodo,
			Position:       0,
		},
		project: domain.Project{
			ID:                  "proj-rich",
			Name:                "Rich Test",
			RepoPrimaryWorktree: worktree,
			HyllaArtifactRef:    "github.com/evanmschultz/tillsyn@main",
			KindCatalogJSON:     encoded,
		},
		columns: []domain.Column{
			{ID: "col-todo", ProjectID: "proj-rich", Name: "To Do", Position: 0},
			{ID: "col-inprogress", ProjectID: "proj-rich", Name: "In Progress", Position: 1},
			{ID: "col-complete", ProjectID: "proj-rich", Name: "Complete", Position: 2},
			{ID: "col-failed", ProjectID: "proj-rich", Name: "Failed", Position: 3},
		},
	}
}

// TestRunOnceStage8FailureMarksActionItemFailed pins the §2.1 fix: when
// Stage 8 (monitor.Track) fails to spawn the subprocess, the dispatcher MUST
// transition the action item to failed (with metadata) and fire the cleanup
// hook so the locks are released. Pre-fix path released locks but left the
// item phantom-in_progress, requiring manual DB recovery.
//
// Reproduction: spawn.go hardcodes argv[0]="claude" → cmd.Start fails with
// ENOENT when "claude" is not on PATH. We point PATH at an empty dir to
// guarantee the failure.
func TestRunOnceStage8FailureMarksActionItemFailed(t *testing.T) {
	// Cannot t.Parallel: t.Setenv is incompatible.
	rich := buildRichFixture(t)
	d := newRichDispatcherForTest(t, rich)

	// Force exec.Command("claude", ...) to fail with ENOENT by pointing
	// PATH at an empty directory. cmd.Start returns "executable file not
	// found in $PATH"; monitor.Track wraps it as ErrMonitorNotStarted.
	t.Setenv("PATH", t.TempDir())

	_, err := d.RunOnce(context.Background(), rich.item.ID, "")
	if err == nil {
		t.Fatalf("RunOnce() error = nil, want monitor-not-started error")
	}
	if !errors.Is(err, ErrMonitorNotStarted) {
		t.Fatalf("RunOnce() error = %v, want errors.Is(ErrMonitorNotStarted)", err)
	}

	// Action item MUST end in failed state, not in_progress.
	rich.mu.Lock()
	defer rich.mu.Unlock()
	if rich.item.LifecycleState != domain.StateFailed {
		t.Errorf("action item LifecycleState = %q, want %q (Stage 8 rollback should transition to failed)",
			rich.item.LifecycleState, domain.StateFailed)
	}
	// movedTo MUST contain BOTH in_progress (from walker.Promote) AND
	// failed (from transitionToFailed). The order pins the rollback
	// happened AFTER the promote, not instead of it.
	if len(rich.movedTo) != 2 {
		t.Fatalf("movedTo = %v, want 2 entries (in_progress + failed)", rich.movedTo)
	}
	if rich.movedTo[0] != "col-inprogress" {
		t.Errorf("movedTo[0] = %q, want col-inprogress (walker.Promote)", rich.movedTo[0])
	}
	if rich.movedTo[1] != "col-failed" {
		t.Errorf("movedTo[1] = %q, want col-failed (Stage 8 rollback)", rich.movedTo[1])
	}
	// Metadata MUST carry Outcome=failure and a BlockedReason populated by
	// the dispatcher's failure-reason builder.
	if rich.item.Metadata.Outcome != "failure" {
		t.Errorf("Metadata.Outcome = %q, want %q", rich.item.Metadata.Outcome, "failure")
	}
	if rich.item.Metadata.BlockedReason == "" {
		t.Errorf("Metadata.BlockedReason is empty; want dispatcher-prefixed reason")
	}
	if len(rich.updateCalls) != 1 {
		t.Errorf("updateCalls = %d, want 1 (transitionToFailed metadata write)", len(rich.updateCalls))
	}
}

// TestRunOnceStage5LockConflictReturnsBlocked pins the §2.4 fix: the
// dispatcher's Stage 5 (file/package lock acquire) MUST surface lock
// conflicts as ResultBlocked (not error, not phantom in_progress). Pre-fix
// state: this path was untested. We pre-acquire a file lock on the
// dispatcher's fileLockManager from a DIFFERENT action-item ID, then run
// the candidate; the candidate's Acquire returns conflicts → Stage 5
// short-circuits with Result=Blocked.
func TestRunOnceStage5LockConflictReturnsBlocked(t *testing.T) {
	t.Parallel()

	rich := buildRichFixture(t)
	rich.item.Paths = []string{"internal/app/dispatcher/dispatcher.go"}
	d := newRichDispatcherForTest(t, rich)

	// Pre-acquire the path under a different action-item ID. The
	// candidate's Acquire will report fileConflicts→ ResultBlocked.
	if _, conflicts, err := d.fileLocks.Acquire("ai-other-holder", rich.item.Paths); err != nil {
		t.Fatalf("pre-acquire fileLocks error = %v", err)
	} else if len(conflicts) != 0 {
		t.Fatalf("pre-acquire reported conflicts: %v", conflicts)
	}

	outcome, err := d.RunOnce(context.Background(), rich.item.ID, "")
	if err != nil {
		t.Fatalf("RunOnce() error = %v, want nil (lock conflict is a Blocked outcome, not an error)", err)
	}
	if outcome.Result != ResultBlocked {
		t.Fatalf("RunOnce() Result = %q, want %q", outcome.Result, ResultBlocked)
	}
	if outcome.Reason == "" {
		t.Errorf("RunOnce() Reason is empty; want lock-conflict reason")
	}
	// Walker.Promote MUST NOT have fired — the lock conflict short-circuits
	// at Stage 5, before Stage 7's promote.
	rich.mu.Lock()
	defer rich.mu.Unlock()
	if len(rich.movedTo) != 0 {
		t.Errorf("movedTo = %v, want empty (Stage 5 conflict precedes Stage 7 promote)", rich.movedTo)
	}
	if rich.item.LifecycleState != domain.StateTodo {
		t.Errorf("LifecycleState = %q, want %q (no promotion on lock conflict)",
			rich.item.LifecycleState, domain.StateTodo)
	}
}

// silenceUnused keeps the imports for exec/json/templates referenced in the
// rich-fixture helpers from triggering unused-import errors when the file
// is read in isolation. The functions above use all three.
var _ = exec.Command
