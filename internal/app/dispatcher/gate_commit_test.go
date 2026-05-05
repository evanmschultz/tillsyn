package dispatcher

import (
	"context"
	"errors"
	"testing"

	"github.com/evanmschultz/tillsyn/internal/domain"
	"github.com/evanmschultz/tillsyn/internal/templates"
)

// gate_commit_test.go ships the F.7-CORE F.7.13 unit-test suite for
// CommitGateRunner.Run. Every test injects mocks for CommitAgent + the three
// git seams so the algorithm path is tested without depending on a real
// spawn pipeline, real git, or a real commit-message agent.
//
// The 8 documented spec scenarios are:
//
//  1. Happy path — toggle on, paths populated, all deps succeed → EndCommit set.
//  2. Toggle off — IsDispatcherCommitEnabled() == false → no-op, no git invoked.
//  3. Empty paths — item.Paths == nil → ErrCommitGateNoPaths.
//  4. CommitAgent fails — GenerateMessage returns error → wrapped error.
//  5. git add fails — GitAdd returns error → ErrCommitGateAddFailed-wrapped.
//  6. git commit fails — GitCommit error → ErrCommitGateCommitFailed-wrapped.
//  7. git rev-parse fails — GitRevParseHead error → ErrCommitGateRevParseFailed-wrapped.
//  8. EndCommit set correctly — returned hash from rev-parse → item.EndCommit verbatim.

// recordingGitFns captures every git-shim invocation so tests can assert
// (a) the gate did or did not invoke each shim, and (b) the gate forwarded
// the expected (repoPath, paths, message) tuple. The closures returned from
// the constructor close over the same struct so a single recordingGitFns
// instance services all three shims.
type recordingGitFns struct {
	addCalls    int
	addPaths    []string
	addRepoPath string

	commitCalls   int
	commitMessage string

	revParseCalls    int
	revParseRepoPath string

	addErr      error
	commitErr   error
	revParseErr error

	revParseHash string // returned by GitRevParseHead on success
}

func (r *recordingGitFns) gitAdd() GitAddFunc {
	return func(_ context.Context, repoPath string, paths []string) error {
		r.addCalls++
		r.addRepoPath = repoPath
		r.addPaths = append([]string{}, paths...)
		return r.addErr
	}
}

func (r *recordingGitFns) gitCommit() GitCommitFunc {
	return func(_ context.Context, _ string, message string) error {
		r.commitCalls++
		r.commitMessage = message
		return r.commitErr
	}
}

func (r *recordingGitFns) gitRevParseHead() GitRevParseFunc {
	return func(_ context.Context, repoPath string) (string, error) {
		r.revParseCalls++
		r.revParseRepoPath = repoPath
		if r.revParseErr != nil {
			return "", r.revParseErr
		}
		return r.revParseHash, nil
	}
}

// commitGateBuildItem returns a build-kind action item with non-empty
// Paths + commits set. Tests that exercise empty-paths or empty-commits
// branches pass overrides directly.
func commitGateBuildItem() *domain.ActionItem {
	return &domain.ActionItem{
		ID:          "ai-commit-gate-1",
		ProjectID:   "proj-1",
		Kind:        domain.KindBuild,
		Paths:       []string{"internal/app/dispatcher/gate_commit.go"},
		StartCommit: "a1b2c3",
		EndCommit:   "f4e5d6",
		Title:       "build droplet under commit gate",
	}
}

// commitGateProjectToggleOn returns a project with DispatcherCommitEnabled
// = true so the gate proceeds past step 1.
func commitGateProjectToggleOn() domain.Project {
	on := true
	return domain.Project{
		ID:                  "proj-1",
		RepoPrimaryWorktree: "/tmp/proj-commit-gate",
		Metadata: domain.ProjectMetadata{
			DispatcherCommitEnabled: &on,
		},
	}
}

// commitGateProjectToggleOff returns a project with the toggle UNSET
// (nil pointer → IsDispatcherCommitEnabled() returns false). Default state
// for new projects pre-Drop-4c-dogfood.
func commitGateProjectToggleOff() domain.Project {
	return domain.Project{
		ID:                  "proj-1",
		RepoPrimaryWorktree: "/tmp/proj-commit-gate",
		// Metadata.DispatcherCommitEnabled left nil → toggle off.
	}
}

// commitGateProjectToggleExplicitFalse returns a project with the toggle
// explicitly set to false. The three-state pointer-bool design (nil vs
// false vs true) collapses nil and false to "disabled" today; the test
// pins that both forms are no-op'd identically.
func commitGateProjectToggleExplicitFalse() domain.Project {
	off := false
	return domain.Project{
		ID:                  "proj-1",
		RepoPrimaryWorktree: "/tmp/proj-commit-gate",
		Metadata: domain.ProjectMetadata{
			DispatcherCommitEnabled: &off,
		},
	}
}

// commitGateCommitAgent returns a CommitAgent wired with stubs that produce
// the supplied message + nil err on GenerateMessage. The agent is
// constructed with the same shape commit_agent_test.go uses (stubGitDiff +
// fakeSpawnBuilder + fakeMonitor) so the F.7.12 algorithm runs end-to-end
// inside the gate test.
//
// tmp is the test's t.TempDir() so the spawn bundle's context dir resolves
// to a writable location.
//
// agentErr (when non-nil) replaces the success path: GenerateMessage will
// return ("", agentErr). Used by the CommitAgent-fails scenario.
func commitGateCommitAgent(t *testing.T, tmp string, message string, agentErr error) *CommitAgent {
	t.Helper()
	if agentErr != nil {
		// When the agent should fail, wire BuildSpawnCommand to return the
		// error so the wrap-and-return branch fires inside the F.7.12
		// algorithm. fakeSpawnBuilder wraps BuildSpawnCommand failures
		// verbatim — F.7.12 returns "dispatcher: commit-agent build spawn:
		// %w" on the err path.
		c := &CommitAgent{
			GitDiff:           &stubGitDiff{},
			BuildSpawnCommand: fakeSpawnBuilder(tmp, agentErr),
			Monitor:           fakeMonitor(TerminalReport{}, nil),
		}
		return c
	}
	c := &CommitAgent{
		GitDiff:           &stubGitDiff{diff: []byte("diff --git a/x b/x\n+hello\n")},
		BuildSpawnCommand: fakeSpawnBuilder(tmp, nil),
		Monitor:           fakeMonitor(TerminalReport{Reason: message}, nil),
	}
	withMockAdapter(c)
	withNopRunCmd(c)
	withStreamReader(c, "")
	return c
}

// TestCommitGateRunHappyPath asserts the canonical happy path: toggle on,
// paths populated, every dependency succeeds → EndCommit is set to the
// rev-parse hash, all three git shims fire, and Run returns nil.
func TestCommitGateRunHappyPath(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	wantMsg := "feat(dispatcher): add commit gate"
	wantHash := "abcd1234567890ef1234567890abcd1234567890"

	rec := &recordingGitFns{revParseHash: wantHash}
	runner := &CommitGateRunner{
		CommitAgent:     commitGateCommitAgent(t, tmp, wantMsg, nil),
		GitAdd:          rec.gitAdd(),
		GitCommit:       rec.gitCommit(),
		GitRevParseHead: rec.gitRevParseHead(),
	}

	item := commitGateBuildItem()
	project := commitGateProjectToggleOn()

	err := runner.Run(context.Background(), item, project, commitCatalog(), AuthBundle{})
	if err != nil {
		t.Fatalf("Run err = %v; want nil", err)
	}

	if item.EndCommit != wantHash {
		t.Errorf("item.EndCommit = %q; want %q (rev-parse output)", item.EndCommit, wantHash)
	}
	if rec.addCalls != 1 {
		t.Errorf("git add calls = %d; want 1", rec.addCalls)
	}
	if rec.addRepoPath != project.RepoPrimaryWorktree {
		t.Errorf("git add repoPath = %q; want %q", rec.addRepoPath, project.RepoPrimaryWorktree)
	}
	if len(rec.addPaths) != 1 || rec.addPaths[0] != item.Paths[0] {
		t.Errorf("git add paths = %v; want %v", rec.addPaths, item.Paths)
	}
	if rec.commitCalls != 1 {
		t.Errorf("git commit calls = %d; want 1", rec.commitCalls)
	}
	if rec.commitMessage != wantMsg {
		t.Errorf("git commit message = %q; want %q", rec.commitMessage, wantMsg)
	}
	if rec.revParseCalls != 1 {
		t.Errorf("git rev-parse calls = %d; want 1", rec.revParseCalls)
	}
}

// TestCommitGateRunToggleOff asserts that IsDispatcherCommitEnabled() == false
// (nil pointer state, the default) short-circuits Run to a successful no-op:
// no git command fires, the CommitAgent is not invoked, item.EndCommit is
// unchanged, Run returns nil.
func TestCommitGateRunToggleOff(t *testing.T) {
	t.Parallel()

	rec := &recordingGitFns{}
	// CommitAgent intentionally has zero-value fields — invoking it would
	// nil-deref. The toggle-off path MUST short-circuit before touching
	// CommitAgent at all.
	runner := &CommitGateRunner{
		CommitAgent:     &CommitAgent{},
		GitAdd:          rec.gitAdd(),
		GitCommit:       rec.gitCommit(),
		GitRevParseHead: rec.gitRevParseHead(),
	}

	item := commitGateBuildItem()
	preEndCommit := item.EndCommit

	err := runner.Run(context.Background(), item, commitGateProjectToggleOff(), commitCatalog(), AuthBundle{})
	if err != nil {
		t.Fatalf("Run err = %v; want nil (toggle-off no-op)", err)
	}
	if item.EndCommit != preEndCommit {
		t.Errorf("item.EndCommit mutated = %q; want unchanged %q", item.EndCommit, preEndCommit)
	}
	if rec.addCalls != 0 {
		t.Errorf("git add calls = %d; want 0 on toggle-off", rec.addCalls)
	}
	if rec.commitCalls != 0 {
		t.Errorf("git commit calls = %d; want 0 on toggle-off", rec.commitCalls)
	}
	if rec.revParseCalls != 0 {
		t.Errorf("git rev-parse calls = %d; want 0 on toggle-off", rec.revParseCalls)
	}
}

// TestCommitGateRunToggleExplicitFalse pins that DispatcherCommitEnabled
// explicitly set to *false (vs nil) is treated identically to the default
// nil case. The three-state pointer-bool reserves the shape for future
// nil-vs-false divergence; today both forms collapse to "disabled."
func TestCommitGateRunToggleExplicitFalse(t *testing.T) {
	t.Parallel()

	rec := &recordingGitFns{}
	runner := &CommitGateRunner{
		CommitAgent:     &CommitAgent{},
		GitAdd:          rec.gitAdd(),
		GitCommit:       rec.gitCommit(),
		GitRevParseHead: rec.gitRevParseHead(),
	}

	item := commitGateBuildItem()
	err := runner.Run(context.Background(), item, commitGateProjectToggleExplicitFalse(), commitCatalog(), AuthBundle{})
	if err != nil {
		t.Fatalf("Run err = %v; want nil (explicit-false no-op)", err)
	}
	if rec.addCalls != 0 || rec.commitCalls != 0 || rec.revParseCalls != 0 {
		t.Errorf("git shims fired despite explicit-false toggle: add=%d commit=%d revParse=%d",
			rec.addCalls, rec.commitCalls, rec.revParseCalls)
	}
}

// TestCommitGateRunEmptyPaths asserts that an empty Paths slice (nil OR
// zero-length) triggers ErrCommitGateNoPaths BEFORE any git command runs.
// Empty paths is a hard failure per F.7.13 — silent no-op would mask a
// planner bug.
func TestCommitGateRunEmptyPaths(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		paths []string
	}{
		{"nil_paths", nil},
		{"zero_length_paths", []string{}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rec := &recordingGitFns{}
			runner := &CommitGateRunner{
				CommitAgent:     &CommitAgent{},
				GitAdd:          rec.gitAdd(),
				GitCommit:       rec.gitCommit(),
				GitRevParseHead: rec.gitRevParseHead(),
			}

			item := commitGateBuildItem()
			item.Paths = tc.paths

			err := runner.Run(context.Background(), item, commitGateProjectToggleOn(), commitCatalog(), AuthBundle{})
			if !errors.Is(err, ErrCommitGateNoPaths) {
				t.Errorf("Run err = %v; want errors.Is ErrCommitGateNoPaths", err)
			}
			if rec.addCalls != 0 || rec.commitCalls != 0 || rec.revParseCalls != 0 {
				t.Errorf("git shims fired despite empty paths: add=%d commit=%d revParse=%d",
					rec.addCalls, rec.commitCalls, rec.revParseCalls)
			}
		})
	}
}

// TestCommitGateRunCommitAgentFails asserts that a CommitAgent.GenerateMessage
// failure propagates as a wrapped error (sentinel from F.7.12 still
// reachable via errors.Is) and NO git command runs.
func TestCommitGateRunCommitAgentFails(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	agentErr := errors.New("synthetic build-spawn failure")

	rec := &recordingGitFns{}
	runner := &CommitGateRunner{
		CommitAgent:     commitGateCommitAgent(t, tmp, "", agentErr),
		GitAdd:          rec.gitAdd(),
		GitCommit:       rec.gitCommit(),
		GitRevParseHead: rec.gitRevParseHead(),
	}

	err := runner.Run(context.Background(), commitGateBuildItem(), commitGateProjectToggleOn(), commitCatalog(), AuthBundle{})
	if err == nil {
		t.Fatal("Run err = nil; want wrapped agent error")
	}
	if !errors.Is(err, agentErr) {
		t.Errorf("Run err = %v; want errors.Is agentErr", err)
	}
	if rec.addCalls != 0 || rec.commitCalls != 0 || rec.revParseCalls != 0 {
		t.Errorf("git shims fired despite agent failure: add=%d commit=%d revParse=%d",
			rec.addCalls, rec.commitCalls, rec.revParseCalls)
	}
}

// TestCommitGateRunGitAddFails asserts that a GitAdd error is wrapped with
// ErrCommitGateAddFailed (errors.Is) and the underlying error is also
// reachable via errors.Is. GitCommit + GitRevParseHead MUST NOT be called.
func TestCommitGateRunGitAddFails(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	addErr := errors.New("synthetic git add failure: pathspec did not match")

	rec := &recordingGitFns{addErr: addErr}
	runner := &CommitGateRunner{
		CommitAgent:     commitGateCommitAgent(t, tmp, "feat: x", nil),
		GitAdd:          rec.gitAdd(),
		GitCommit:       rec.gitCommit(),
		GitRevParseHead: rec.gitRevParseHead(),
	}

	item := commitGateBuildItem()
	preEndCommit := item.EndCommit

	err := runner.Run(context.Background(), item, commitGateProjectToggleOn(), commitCatalog(), AuthBundle{})
	if !errors.Is(err, ErrCommitGateAddFailed) {
		t.Errorf("Run err = %v; want errors.Is ErrCommitGateAddFailed", err)
	}
	if !errors.Is(err, addErr) {
		t.Errorf("Run err = %v; want errors.Is addErr (underlying cause)", err)
	}
	if rec.addCalls != 1 {
		t.Errorf("git add calls = %d; want 1", rec.addCalls)
	}
	if rec.commitCalls != 0 {
		t.Errorf("git commit calls = %d; want 0 after add failure", rec.commitCalls)
	}
	if rec.revParseCalls != 0 {
		t.Errorf("git rev-parse calls = %d; want 0 after add failure", rec.revParseCalls)
	}
	if item.EndCommit != preEndCommit {
		t.Errorf("item.EndCommit mutated = %q; want unchanged %q on add failure", item.EndCommit, preEndCommit)
	}
}

// TestCommitGateRunGitCommitFails asserts that a GitCommit error is
// wrapped with ErrCommitGateCommitFailed (errors.Is) and the underlying
// error is also reachable. GitRevParseHead MUST NOT be called.
func TestCommitGateRunGitCommitFails(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	commitErr := errors.New("synthetic git commit failure: nothing to commit")

	rec := &recordingGitFns{commitErr: commitErr}
	runner := &CommitGateRunner{
		CommitAgent:     commitGateCommitAgent(t, tmp, "feat: x", nil),
		GitAdd:          rec.gitAdd(),
		GitCommit:       rec.gitCommit(),
		GitRevParseHead: rec.gitRevParseHead(),
	}

	item := commitGateBuildItem()
	preEndCommit := item.EndCommit

	err := runner.Run(context.Background(), item, commitGateProjectToggleOn(), commitCatalog(), AuthBundle{})
	if !errors.Is(err, ErrCommitGateCommitFailed) {
		t.Errorf("Run err = %v; want errors.Is ErrCommitGateCommitFailed", err)
	}
	if !errors.Is(err, commitErr) {
		t.Errorf("Run err = %v; want errors.Is commitErr (underlying cause)", err)
	}
	if rec.addCalls != 1 || rec.commitCalls != 1 {
		t.Errorf("expected add=1 commit=1; got add=%d commit=%d", rec.addCalls, rec.commitCalls)
	}
	if rec.revParseCalls != 0 {
		t.Errorf("git rev-parse calls = %d; want 0 after commit failure", rec.revParseCalls)
	}
	if item.EndCommit != preEndCommit {
		t.Errorf("item.EndCommit mutated = %q; want unchanged %q on commit failure", item.EndCommit, preEndCommit)
	}
}

// TestCommitGateRunGitRevParseFails asserts that a GitRevParseHead error is
// wrapped with ErrCommitGateRevParseFailed (errors.Is) and the underlying
// error is reachable. The commit + add already succeeded but the gate
// cannot populate item.EndCommit so it MUST surface the failure rather
// than silently leaving EndCommit empty.
func TestCommitGateRunGitRevParseFails(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	revParseErr := errors.New("synthetic rev-parse failure: refs/HEAD corrupt")

	rec := &recordingGitFns{revParseErr: revParseErr}
	runner := &CommitGateRunner{
		CommitAgent:     commitGateCommitAgent(t, tmp, "feat: x", nil),
		GitAdd:          rec.gitAdd(),
		GitCommit:       rec.gitCommit(),
		GitRevParseHead: rec.gitRevParseHead(),
	}

	item := commitGateBuildItem()
	preEndCommit := item.EndCommit

	err := runner.Run(context.Background(), item, commitGateProjectToggleOn(), commitCatalog(), AuthBundle{})
	if !errors.Is(err, ErrCommitGateRevParseFailed) {
		t.Errorf("Run err = %v; want errors.Is ErrCommitGateRevParseFailed", err)
	}
	if !errors.Is(err, revParseErr) {
		t.Errorf("Run err = %v; want errors.Is revParseErr (underlying cause)", err)
	}
	if rec.addCalls != 1 || rec.commitCalls != 1 || rec.revParseCalls != 1 {
		t.Errorf("expected add=1 commit=1 revParse=1; got add=%d commit=%d revParse=%d",
			rec.addCalls, rec.commitCalls, rec.revParseCalls)
	}
	if item.EndCommit != preEndCommit {
		t.Errorf("item.EndCommit mutated = %q; want unchanged %q on rev-parse failure", item.EndCommit, preEndCommit)
	}
}

// TestCommitGateRunGitRevParseEmpty asserts that a GitRevParseHead returning
// an empty (whitespace-stripped) string is treated as a rev-parse failure,
// not silently leaving item.EndCommit empty. Downstream gates use non-empty
// EndCommit as the "commit happened" signal; an empty value would poison
// that read.
func TestCommitGateRunGitRevParseEmpty(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	rec := &recordingGitFns{revParseHash: ""}
	runner := &CommitGateRunner{
		CommitAgent:     commitGateCommitAgent(t, tmp, "feat: x", nil),
		GitAdd:          rec.gitAdd(),
		GitCommit:       rec.gitCommit(),
		GitRevParseHead: rec.gitRevParseHead(),
	}

	item := commitGateBuildItem()
	preEndCommit := item.EndCommit

	err := runner.Run(context.Background(), item, commitGateProjectToggleOn(), commitCatalog(), AuthBundle{})
	if !errors.Is(err, ErrCommitGateRevParseFailed) {
		t.Errorf("Run err = %v; want errors.Is ErrCommitGateRevParseFailed on empty hash", err)
	}
	if item.EndCommit != preEndCommit {
		t.Errorf("item.EndCommit mutated = %q; want unchanged %q on empty rev-parse output", item.EndCommit, preEndCommit)
	}
}

// TestCommitGateRunEndCommitSetCorrectly asserts that the rev-parse return
// value flows verbatim into item.EndCommit. Spec scenario 8 — explicitly
// independent of the happy-path test even though the assertion overlaps,
// because the spec calls it out as a separate acceptance line.
func TestCommitGateRunEndCommitSetCorrectly(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	wantHash := "0123456789abcdef0123456789abcdef01234567"

	rec := &recordingGitFns{revParseHash: wantHash}
	runner := &CommitGateRunner{
		CommitAgent:     commitGateCommitAgent(t, tmp, "feat: x", nil),
		GitAdd:          rec.gitAdd(),
		GitCommit:       rec.gitCommit(),
		GitRevParseHead: rec.gitRevParseHead(),
	}

	item := commitGateBuildItem()
	item.EndCommit = "OLD-VALUE-MUST-BE-OVERWRITTEN"

	err := runner.Run(context.Background(), item, commitGateProjectToggleOn(), commitCatalog(), AuthBundle{})
	if err != nil {
		t.Fatalf("Run err = %v; want nil", err)
	}
	if item.EndCommit != wantHash {
		t.Errorf("item.EndCommit = %q; want %q (verbatim from rev-parse)", item.EndCommit, wantHash)
	}
}

// TestCommitGateRunNilReceiver asserts a nil *CommitGateRunner returns a
// loud error rather than nil-derefing. Defense-in-depth — production
// wiring should never produce a nil runner, but the failure mode is
// observable rather than panic-driven.
func TestCommitGateRunNilReceiver(t *testing.T) {
	t.Parallel()

	var runner *CommitGateRunner
	err := runner.Run(context.Background(), commitGateBuildItem(), commitGateProjectToggleOn(), commitCatalog(), AuthBundle{})
	if err == nil {
		t.Fatal("Run on nil receiver err = nil; want loud error")
	}
}

// TestCommitGateRunNilItem asserts a nil *domain.ActionItem returns a
// loud error rather than nil-derefing. Defense-in-depth.
func TestCommitGateRunNilItem(t *testing.T) {
	t.Parallel()

	runner := &CommitGateRunner{
		CommitAgent:     &CommitAgent{},
		GitAdd:          (&recordingGitFns{}).gitAdd(),
		GitCommit:       (&recordingGitFns{}).gitCommit(),
		GitRevParseHead: (&recordingGitFns{}).gitRevParseHead(),
	}

	err := runner.Run(context.Background(), nil, commitGateProjectToggleOn(), commitCatalog(), AuthBundle{})
	if err == nil {
		t.Fatal("Run on nil item err = nil; want loud error")
	}
}

// TestCommitGateRunNilCommitAgent asserts that a nil CommitAgent field
// (production wiring bug) returns a clear error rather than nil-derefing
// on GenerateMessage call. Triggered AFTER the toggle / paths guards so
// the failure surface is "CommitAgent missing during execution," not
// "CommitAgent missing for a no-op."
func TestCommitGateRunNilCommitAgent(t *testing.T) {
	t.Parallel()

	rec := &recordingGitFns{}
	runner := &CommitGateRunner{
		CommitAgent:     nil, // production wiring bug
		GitAdd:          rec.gitAdd(),
		GitCommit:       rec.gitCommit(),
		GitRevParseHead: rec.gitRevParseHead(),
	}

	err := runner.Run(context.Background(), commitGateBuildItem(), commitGateProjectToggleOn(), commitCatalog(), AuthBundle{})
	if err == nil {
		t.Fatal("Run with nil CommitAgent err = nil; want loud error")
	}
	if rec.addCalls != 0 {
		t.Errorf("git add fired despite nil CommitAgent: calls = %d", rec.addCalls)
	}
}

// TestGateKindCommitRegistered cross-checks that the templates package's
// closed GateKind enum accepts "commit" via IsValidGateKind. Belt-and-
// suspenders against accidental enum churn that would otherwise let a
// template author bind GateKindCommit only for the gate to silently no-op
// at template-load time.
func TestGateKindCommitRegistered(t *testing.T) {
	t.Parallel()

	if !templates.IsValidGateKind(templates.GateKindCommit) {
		t.Fatalf("IsValidGateKind(%q) = false; want true after F.7.13", templates.GateKindCommit)
	}
	if string(templates.GateKindCommit) != "commit" {
		t.Errorf("GateKindCommit = %q; want %q (canonical gate-name string)", templates.GateKindCommit, "commit")
	}
}
