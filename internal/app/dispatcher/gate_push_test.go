package dispatcher

import (
	"context"
	"errors"
	"testing"

	"github.com/evanmschultz/tillsyn/internal/domain"
	"github.com/evanmschultz/tillsyn/internal/templates"
)

// gate_push_test.go ships the F.7-CORE F.7.14 unit-test suite for
// PushGateRunner.Run. Every test injects mocks for the GitCurrentBranch +
// GitPush seams so the algorithm path is tested without depending on a real
// git invocation, a real remote, or a real worktree.
//
// The 6 prompt-mandated spec scenarios are:
//
//  1. Toggle off — IsDispatcherPushEnabled() == false → no-op, no git invoked.
//  2. Toggle on, push succeeds → returns nil (happy path).
//  3. Toggle on, push fails → ErrPushGatePushFailed-wrapped (errors.Is unwraps).
//  4. Branch missing (empty return) → ErrPushGateBranchMissing.
//  5. Nil receiver → loud error, no git invoked.
//  6. GateKindPush in valid enum cross-check.
//
// Adjacent defense-in-depth scenarios mirror gate_commit_test.go's pattern:
// nil item, nil GitCurrentBranch field, nil GitPush field, GitCurrentBranch
// returning a non-nil error (vs the empty-string shape), and the explicit
// false-toggle path (vs the default nil-pointer toggle).

// recordingPushFns captures every push-shim invocation so tests can assert
// (a) the gate did or did not invoke each shim, and (b) the gate forwarded
// the expected (repoPath, branch) tuple. The closures returned from the
// constructors close over the same struct so a single recordingPushFns
// services both shims.
type recordingPushFns struct {
	branchCalls    int
	branchRepoPath string

	pushCalls    int
	pushRepoPath string
	pushBranch   string

	branchErr error
	pushErr   error

	branchReturn string // returned by GitCurrentBranch on success
}

func (r *recordingPushFns) gitCurrentBranch() GitCurrentBranchFunc {
	return func(_ context.Context, repoPath string) (string, error) {
		r.branchCalls++
		r.branchRepoPath = repoPath
		if r.branchErr != nil {
			return "", r.branchErr
		}
		return r.branchReturn, nil
	}
}

func (r *recordingPushFns) gitPush() GitPushFunc {
	return func(_ context.Context, repoPath string, branch string) error {
		r.pushCalls++
		r.pushRepoPath = repoPath
		r.pushBranch = branch
		return r.pushErr
	}
}

// pushGateBuildItem returns a build-kind action item suitable for push-gate
// scenarios. The push gate does NOT mutate item (per gate_push.go Run docs),
// so the field surface here is intentionally minimal.
func pushGateBuildItem() *domain.ActionItem {
	return &domain.ActionItem{
		ID:        "ai-push-gate-1",
		ProjectID: "proj-1",
		Kind:      domain.KindBuild,
		Title:     "build droplet under push gate",
	}
}

// pushGateProjectToggleOn returns a project with DispatcherPushEnabled = true
// so the gate proceeds past step 1.
func pushGateProjectToggleOn() domain.Project {
	on := true
	return domain.Project{
		ID:                  "proj-1",
		RepoPrimaryWorktree: "/tmp/proj-push-gate",
		Metadata: domain.ProjectMetadata{
			DispatcherPushEnabled: &on,
		},
	}
}

// pushGateProjectToggleOff returns a project with the toggle UNSET (nil
// pointer → IsDispatcherPushEnabled() returns false). Default state for new
// projects pre-Drop-4c-dogfood.
func pushGateProjectToggleOff() domain.Project {
	return domain.Project{
		ID:                  "proj-1",
		RepoPrimaryWorktree: "/tmp/proj-push-gate",
		// Metadata.DispatcherPushEnabled left nil → toggle off.
	}
}

// pushGateProjectToggleExplicitFalse returns a project with the toggle
// explicitly set to false. The three-state pointer-bool design (nil vs false
// vs true) collapses nil and false to "disabled" today; the test pins that
// both forms are no-op'd identically.
func pushGateProjectToggleExplicitFalse() domain.Project {
	off := false
	return domain.Project{
		ID:                  "proj-1",
		RepoPrimaryWorktree: "/tmp/proj-push-gate",
		Metadata: domain.ProjectMetadata{
			DispatcherPushEnabled: &off,
		},
	}
}

// TestPushGateRunHappyPath asserts the canonical happy path: toggle on,
// branch resolves to a non-empty value, push succeeds → Run returns nil and
// the (repoPath, branch) tuple flows verbatim from
// project.RepoPrimaryWorktree + GitCurrentBranch's return into GitPush.
func TestPushGateRunHappyPath(t *testing.T) {
	t.Parallel()

	wantBranch := "drop/4c"
	rec := &recordingPushFns{branchReturn: wantBranch}
	runner := &PushGateRunner{
		GitCurrentBranch: rec.gitCurrentBranch(),
		GitPush:          rec.gitPush(),
	}

	project := pushGateProjectToggleOn()
	err := runner.Run(context.Background(), pushGateBuildItem(), project, templates.KindCatalog{}, AuthBundle{})
	if err != nil {
		t.Fatalf("Run err = %v; want nil", err)
	}

	if rec.branchCalls != 1 {
		t.Errorf("git symbolic-ref calls = %d; want 1", rec.branchCalls)
	}
	if rec.branchRepoPath != project.RepoPrimaryWorktree {
		t.Errorf("git symbolic-ref repoPath = %q; want %q", rec.branchRepoPath, project.RepoPrimaryWorktree)
	}
	if rec.pushCalls != 1 {
		t.Errorf("git push calls = %d; want 1", rec.pushCalls)
	}
	if rec.pushRepoPath != project.RepoPrimaryWorktree {
		t.Errorf("git push repoPath = %q; want %q", rec.pushRepoPath, project.RepoPrimaryWorktree)
	}
	if rec.pushBranch != wantBranch {
		t.Errorf("git push branch = %q; want %q (verbatim from GitCurrentBranch)", rec.pushBranch, wantBranch)
	}
}

// TestPushGateRunToggleOff asserts that IsDispatcherPushEnabled() == false
// (nil pointer state, the default) short-circuits Run to a successful no-op:
// neither git seam fires, Run returns nil. The toggle-off path MUST NOT
// surface ErrPushGateDisabled — the sentinel exists only as a future-safe
// label per gate_push.go's doc comment.
func TestPushGateRunToggleOff(t *testing.T) {
	t.Parallel()

	rec := &recordingPushFns{}
	runner := &PushGateRunner{
		GitCurrentBranch: rec.gitCurrentBranch(),
		GitPush:          rec.gitPush(),
	}

	err := runner.Run(context.Background(), pushGateBuildItem(), pushGateProjectToggleOff(), templates.KindCatalog{}, AuthBundle{})
	if err != nil {
		t.Fatalf("Run err = %v; want nil (toggle-off no-op)", err)
	}
	// Ensure the symmetry-only sentinel did not slip onto the no-op path.
	if errors.Is(err, ErrPushGateDisabled) {
		t.Errorf("toggle-off path returned ErrPushGateDisabled = %v; gate_push.go contract: nil on no-op", err)
	}
	if rec.branchCalls != 0 {
		t.Errorf("git symbolic-ref calls = %d; want 0 on toggle-off", rec.branchCalls)
	}
	if rec.pushCalls != 0 {
		t.Errorf("git push calls = %d; want 0 on toggle-off", rec.pushCalls)
	}
}

// TestPushGateRunToggleExplicitFalse pins that DispatcherPushEnabled
// explicitly set to *false (vs nil) is treated identically to the default
// nil case. Three-state pointer-bool collapses nil and false to "disabled."
func TestPushGateRunToggleExplicitFalse(t *testing.T) {
	t.Parallel()

	rec := &recordingPushFns{}
	runner := &PushGateRunner{
		GitCurrentBranch: rec.gitCurrentBranch(),
		GitPush:          rec.gitPush(),
	}

	err := runner.Run(context.Background(), pushGateBuildItem(), pushGateProjectToggleExplicitFalse(), templates.KindCatalog{}, AuthBundle{})
	if err != nil {
		t.Fatalf("Run err = %v; want nil (explicit-false no-op)", err)
	}
	if rec.branchCalls != 0 || rec.pushCalls != 0 {
		t.Errorf("git seams fired despite explicit-false toggle: branch=%d push=%d",
			rec.branchCalls, rec.pushCalls)
	}
}

// TestPushGateRunPushFails asserts that a GitPush error is wrapped with
// ErrPushGatePushFailed (errors.Is) and the underlying error is also
// reachable via errors.Is. GitCurrentBranch fired exactly once; GitPush
// fired exactly once.
func TestPushGateRunPushFails(t *testing.T) {
	t.Parallel()

	pushErr := errors.New("synthetic git push failure: rejected non-fast-forward")

	rec := &recordingPushFns{
		branchReturn: "main",
		pushErr:      pushErr,
	}
	runner := &PushGateRunner{
		GitCurrentBranch: rec.gitCurrentBranch(),
		GitPush:          rec.gitPush(),
	}

	err := runner.Run(context.Background(), pushGateBuildItem(), pushGateProjectToggleOn(), templates.KindCatalog{}, AuthBundle{})
	if err == nil {
		t.Fatal("Run err = nil; want wrapped push error")
	}
	if !errors.Is(err, ErrPushGatePushFailed) {
		t.Errorf("Run err = %v; want errors.Is ErrPushGatePushFailed", err)
	}
	if !errors.Is(err, pushErr) {
		t.Errorf("Run err = %v; want errors.Is pushErr (underlying cause)", err)
	}
	if rec.branchCalls != 1 {
		t.Errorf("git symbolic-ref calls = %d; want 1", rec.branchCalls)
	}
	if rec.pushCalls != 1 {
		t.Errorf("git push calls = %d; want 1 (push attempted before failure)", rec.pushCalls)
	}
}

// TestPushGateRunBranchMissingEmpty asserts that GitCurrentBranch returning
// an empty string (whitespace-trimmed) collapses to ErrPushGateBranchMissing.
// GitPush MUST NOT fire — the gate cannot determine where to push.
func TestPushGateRunBranchMissingEmpty(t *testing.T) {
	t.Parallel()

	rec := &recordingPushFns{branchReturn: ""}
	runner := &PushGateRunner{
		GitCurrentBranch: rec.gitCurrentBranch(),
		GitPush:          rec.gitPush(),
	}

	err := runner.Run(context.Background(), pushGateBuildItem(), pushGateProjectToggleOn(), templates.KindCatalog{}, AuthBundle{})
	if !errors.Is(err, ErrPushGateBranchMissing) {
		t.Errorf("Run err = %v; want errors.Is ErrPushGateBranchMissing on empty branch", err)
	}
	if rec.branchCalls != 1 {
		t.Errorf("git symbolic-ref calls = %d; want 1", rec.branchCalls)
	}
	if rec.pushCalls != 0 {
		t.Errorf("git push calls = %d; want 0 on missing branch", rec.pushCalls)
	}
}

// TestPushGateRunBranchMissingError asserts that GitCurrentBranch returning a
// non-nil error also collapses to ErrPushGateBranchMissing AND the underlying
// error is reachable via errors.Is. The two shapes (empty string OR error)
// share a single sentinel per gate_push.go's documented contract.
func TestPushGateRunBranchMissingError(t *testing.T) {
	t.Parallel()

	branchErr := errors.New("synthetic symbolic-ref failure: HEAD detached")

	rec := &recordingPushFns{branchErr: branchErr}
	runner := &PushGateRunner{
		GitCurrentBranch: rec.gitCurrentBranch(),
		GitPush:          rec.gitPush(),
	}

	err := runner.Run(context.Background(), pushGateBuildItem(), pushGateProjectToggleOn(), templates.KindCatalog{}, AuthBundle{})
	if !errors.Is(err, ErrPushGateBranchMissing) {
		t.Errorf("Run err = %v; want errors.Is ErrPushGateBranchMissing on branch-resolve error", err)
	}
	if !errors.Is(err, branchErr) {
		t.Errorf("Run err = %v; want errors.Is branchErr (underlying cause)", err)
	}
	if rec.branchCalls != 1 {
		t.Errorf("git symbolic-ref calls = %d; want 1", rec.branchCalls)
	}
	if rec.pushCalls != 0 {
		t.Errorf("git push calls = %d; want 0 on branch-resolve error", rec.pushCalls)
	}
}

// TestPushGateRunNilReceiver asserts a nil *PushGateRunner returns a loud
// error rather than nil-derefing. Defense-in-depth — production wiring should
// never produce a nil runner, but the failure mode is observable rather than
// panic-driven.
func TestPushGateRunNilReceiver(t *testing.T) {
	t.Parallel()

	var runner *PushGateRunner
	err := runner.Run(context.Background(), pushGateBuildItem(), pushGateProjectToggleOn(), templates.KindCatalog{}, AuthBundle{})
	if err == nil {
		t.Fatal("Run on nil receiver err = nil; want loud error")
	}
}

// TestPushGateRunNilItem asserts a nil *domain.ActionItem returns a loud
// error rather than nil-derefing. Defense-in-depth — symmetric with
// TestCommitGateRunNilItem.
func TestPushGateRunNilItem(t *testing.T) {
	t.Parallel()

	rec := &recordingPushFns{}
	runner := &PushGateRunner{
		GitCurrentBranch: rec.gitCurrentBranch(),
		GitPush:          rec.gitPush(),
	}

	err := runner.Run(context.Background(), nil, pushGateProjectToggleOn(), templates.KindCatalog{}, AuthBundle{})
	if err == nil {
		t.Fatal("Run on nil item err = nil; want loud error")
	}
	if rec.branchCalls != 0 || rec.pushCalls != 0 {
		t.Errorf("git seams fired despite nil item: branch=%d push=%d",
			rec.branchCalls, rec.pushCalls)
	}
}

// TestPushGateRunNilGitCurrentBranchField asserts that a nil GitCurrentBranch
// field (production wiring bug) returns a clear error rather than nil-derefing
// the function value. Triggered AFTER the toggle guard so the failure surface
// is "GitCurrentBranch missing during execution," not "missing for a no-op."
func TestPushGateRunNilGitCurrentBranchField(t *testing.T) {
	t.Parallel()

	rec := &recordingPushFns{}
	runner := &PushGateRunner{
		GitCurrentBranch: nil, // production wiring bug
		GitPush:          rec.gitPush(),
	}

	err := runner.Run(context.Background(), pushGateBuildItem(), pushGateProjectToggleOn(), templates.KindCatalog{}, AuthBundle{})
	if err == nil {
		t.Fatal("Run with nil GitCurrentBranch err = nil; want loud error")
	}
	if rec.pushCalls != 0 {
		t.Errorf("git push fired despite nil GitCurrentBranch: calls = %d", rec.pushCalls)
	}
}

// TestPushGateRunNilGitPushField asserts that a nil GitPush field (production
// wiring bug) returns a clear error rather than nil-derefing the function
// value. Triggered AFTER GitCurrentBranch resolves a non-empty branch so the
// failure surface is "GitPush missing during execution."
func TestPushGateRunNilGitPushField(t *testing.T) {
	t.Parallel()

	rec := &recordingPushFns{branchReturn: "main"}
	runner := &PushGateRunner{
		GitCurrentBranch: rec.gitCurrentBranch(),
		GitPush:          nil, // production wiring bug
	}

	err := runner.Run(context.Background(), pushGateBuildItem(), pushGateProjectToggleOn(), templates.KindCatalog{}, AuthBundle{})
	if err == nil {
		t.Fatal("Run with nil GitPush err = nil; want loud error")
	}
	if rec.branchCalls != 1 {
		t.Errorf("git symbolic-ref calls = %d; want 1 (branch resolved before nil-push trip)", rec.branchCalls)
	}
}

// TestGateKindPushRegistered cross-checks that the templates package's closed
// GateKind enum accepts "push" via IsValidGateKind. Belt-and-suspenders
// against accidental enum churn that would otherwise let a template author
// bind GateKindPush only for the gate to silently no-op at template-load
// time.
func TestGateKindPushRegistered(t *testing.T) {
	t.Parallel()

	if !templates.IsValidGateKind(templates.GateKindPush) {
		t.Fatalf("IsValidGateKind(%q) = false; want true after F.7.14", templates.GateKindPush)
	}
	if string(templates.GateKindPush) != "push" {
		t.Errorf("GateKindPush = %q; want %q (canonical gate-name string)", templates.GateKindPush, "push")
	}
}
