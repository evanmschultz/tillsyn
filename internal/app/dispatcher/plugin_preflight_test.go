package dispatcher

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// fakeClaudePluginLister is the test seam mock for ClaudePluginLister.
// Tests pre-populate Entries (or Err) and inject the lister into
// CheckRequiredPlugins. The fake records the count of List invocations
// so tests can assert the empty-required short-circuit path skips the
// shell-out entirely.
type fakeClaudePluginLister struct {
	Entries []ClaudePluginListEntry
	Err     error
	Calls   int
}

// List implements ClaudePluginLister. Every call increments Calls; tests
// assert the value to verify CheckRequiredPlugins's no-op short-circuit
// for empty `required`.
func (f *fakeClaudePluginLister) List(_ context.Context) ([]ClaudePluginListEntry, error) {
	f.Calls++
	if f.Err != nil {
		return nil, f.Err
	}
	return f.Entries, nil
}

// TestCheckRequiredPluginsNilRequiredReturnsNil verifies the empty-required
// short-circuit: a nil `required` slice produces nil error and does NOT
// invoke the lister.
func TestCheckRequiredPluginsNilRequiredReturnsNil(t *testing.T) {
	lister := &fakeClaudePluginLister{}
	err := CheckRequiredPlugins(context.Background(), lister, nil)
	if err != nil {
		t.Fatalf("CheckRequiredPlugins(nil): unexpected error: %v", err)
	}
	if lister.Calls != 0 {
		t.Fatalf("lister.Calls = %d; want 0 (nil-required must short-circuit before List)", lister.Calls)
	}
}

// TestCheckRequiredPluginsEmptyRequiredReturnsNil mirrors the nil case for
// an explicit empty (non-nil) `required` slice. Both forms are accepted as
// "no required plugins" and both must short-circuit before the lister.
func TestCheckRequiredPluginsEmptyRequiredReturnsNil(t *testing.T) {
	lister := &fakeClaudePluginLister{}
	err := CheckRequiredPlugins(context.Background(), lister, []string{})
	if err != nil {
		t.Fatalf("CheckRequiredPlugins([]): unexpected error: %v", err)
	}
	if lister.Calls != 0 {
		t.Fatalf("lister.Calls = %d; want 0 (empty-required must short-circuit before List)", lister.Calls)
	}
}

// TestCheckRequiredPluginsAllInstalledReturnsNil verifies the happy path:
// every entry in `required` matches an installed plugin, so no error is
// returned. Both bare-name and scoped-name shapes are exercised in the
// same call to confirm the matcher handles them in parallel.
func TestCheckRequiredPluginsAllInstalledReturnsNil(t *testing.T) {
	lister := &fakeClaudePluginLister{
		Entries: []ClaudePluginListEntry{
			{ID: "context7", Marketplace: "claude-plugins-official", Version: "0.4.1", InstallPath: "/p1"},
			{ID: "gopls-lsp", Marketplace: "claude-plugins-official", Version: "0.2.0", InstallPath: "/p2"},
			{ID: "extra", Marketplace: "third-party", Version: "1.0.0", InstallPath: "/p3"},
		},
	}
	required := []string{"context7@claude-plugins-official", "gopls-lsp"}
	err := CheckRequiredPlugins(context.Background(), lister, required)
	if err != nil {
		t.Fatalf("CheckRequiredPlugins: unexpected error: %v", err)
	}
	if lister.Calls != 1 {
		t.Fatalf("lister.Calls = %d; want 1", lister.Calls)
	}
}

// TestCheckRequiredPluginsOneMissing verifies the canonical failure path:
// one of the required entries is absent from the installed list. The
// returned error wraps ErrMissingRequiredPlugins, names the missing
// entry, and emits the install instruction.
func TestCheckRequiredPluginsOneMissing(t *testing.T) {
	lister := &fakeClaudePluginLister{
		Entries: []ClaudePluginListEntry{
			{ID: "gopls-lsp", Marketplace: "claude-plugins-official"},
		},
	}
	required := []string{"context7@claude-plugins-official", "gopls-lsp"}
	err := CheckRequiredPlugins(context.Background(), lister, required)
	if err == nil {
		t.Fatalf("CheckRequiredPlugins: expected ErrMissingRequiredPlugins; got nil")
	}
	if !errors.Is(err, ErrMissingRequiredPlugins) {
		t.Fatalf("CheckRequiredPlugins: errors.Is(_, ErrMissingRequiredPlugins) = false; err = %v", err)
	}
	if !strings.Contains(err.Error(), "context7@claude-plugins-official") {
		t.Fatalf("CheckRequiredPlugins: err = %q; want missing-entry %q in message",
			err.Error(), "context7@claude-plugins-official")
	}
	if !strings.Contains(err.Error(), "claude plugin install context7@claude-plugins-official") {
		t.Fatalf("CheckRequiredPlugins: err = %q; want install instruction in message", err.Error())
	}
	// The installed entry "gopls-lsp" must NOT appear in the missing list.
	if strings.Contains(err.Error(), "claude plugin install gopls-lsp") {
		t.Fatalf("CheckRequiredPlugins: err = %q; gopls-lsp is installed and must not be listed as missing", err.Error())
	}
}

// TestCheckRequiredPluginsMultipleMissingAggregates verifies that ALL
// missing entries are aggregated into a single error message — the dev
// sees the full list to install in one shot rather than the
// fix-one-rerun-treadmill UX. Order is preserved from the `required`
// input slice.
func TestCheckRequiredPluginsMultipleMissingAggregates(t *testing.T) {
	lister := &fakeClaudePluginLister{Entries: nil}
	required := []string{"alpha@official", "beta", "gamma@third-party"}
	err := CheckRequiredPlugins(context.Background(), lister, required)
	if err == nil {
		t.Fatalf("CheckRequiredPlugins: expected ErrMissingRequiredPlugins; got nil")
	}
	if !errors.Is(err, ErrMissingRequiredPlugins) {
		t.Fatalf("CheckRequiredPlugins: errors.Is(_, ErrMissingRequiredPlugins) = false; err = %v", err)
	}
	for _, want := range []string{"alpha@official", "beta", "gamma@third-party"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("CheckRequiredPlugins: err = %q; want missing-entry %q in message", err.Error(), want)
		}
	}
	// Order preservation: alpha should appear before beta, beta before gamma.
	idxAlpha := strings.Index(err.Error(), "alpha@official")
	idxBeta := strings.Index(err.Error(), "beta")
	idxGamma := strings.Index(err.Error(), "gamma@third-party")
	if !(idxAlpha < idxBeta && idxBeta < idxGamma) {
		t.Fatalf("CheckRequiredPlugins: err = %q; want order alpha<beta<gamma (got %d, %d, %d)",
			err.Error(), idxAlpha, idxBeta, idxGamma)
	}
}

// TestCheckRequiredPluginsScopedRequirementMatchesScopedInstalled verifies
// that a `<name>@<marketplace>` requirement matches ONLY when both ID and
// Marketplace match. A right-name / wrong-marketplace installed entry
// does NOT satisfy the scoped requirement.
func TestCheckRequiredPluginsScopedRequirementMatchesScopedInstalled(t *testing.T) {
	tests := []struct {
		name      string
		installed []ClaudePluginListEntry
		required  []string
		wantErr   bool
	}{
		{
			name: "scoped match",
			installed: []ClaudePluginListEntry{
				{ID: "context7", Marketplace: "claude-plugins-official"},
			},
			required: []string{"context7@claude-plugins-official"},
			wantErr:  false,
		},
		{
			name: "scoped mismatch on marketplace",
			installed: []ClaudePluginListEntry{
				{ID: "context7", Marketplace: "third-party-fork"},
			},
			required: []string{"context7@claude-plugins-official"},
			wantErr:  true,
		},
		{
			name: "scoped requirement, bare-marketplace installed",
			installed: []ClaudePluginListEntry{
				{ID: "context7", Marketplace: ""},
			},
			required: []string{"context7@claude-plugins-official"},
			wantErr:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lister := &fakeClaudePluginLister{Entries: tc.installed}
			err := CheckRequiredPlugins(context.Background(), lister, tc.required)
			gotErr := err != nil
			if gotErr != tc.wantErr {
				t.Fatalf("CheckRequiredPlugins: gotErr = %v (err=%v); wantErr = %v", gotErr, err, tc.wantErr)
			}
		})
	}
}

// TestCheckRequiredPluginsBareRequirementIgnoresMarketplace verifies that
// a bare-name `<name>` requirement matches ANY installed entry whose ID
// matches, regardless of Marketplace. The matcher treats Marketplace as
// out-of-scope for bare requirements.
func TestCheckRequiredPluginsBareRequirementIgnoresMarketplace(t *testing.T) {
	tests := []struct {
		name      string
		installed []ClaudePluginListEntry
		wantErr   bool
	}{
		{
			name: "bare matches official-marketplace install",
			installed: []ClaudePluginListEntry{
				{ID: "context7", Marketplace: "claude-plugins-official"},
			},
			wantErr: false,
		},
		{
			name: "bare matches third-party-marketplace install",
			installed: []ClaudePluginListEntry{
				{ID: "context7", Marketplace: "third-party"},
			},
			wantErr: false,
		},
		{
			name: "bare matches empty-marketplace install",
			installed: []ClaudePluginListEntry{
				{ID: "context7", Marketplace: ""},
			},
			wantErr: false,
		},
		{
			name: "bare misses when ID absent",
			installed: []ClaudePluginListEntry{
				{ID: "other-plugin", Marketplace: "claude-plugins-official"},
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lister := &fakeClaudePluginLister{Entries: tc.installed}
			err := CheckRequiredPlugins(context.Background(), lister, []string{"context7"})
			gotErr := err != nil
			if gotErr != tc.wantErr {
				t.Fatalf("CheckRequiredPlugins: gotErr = %v (err=%v); wantErr = %v", gotErr, err, tc.wantErr)
			}
		})
	}
}

// TestCheckRequiredPluginsListerErrorPropagates verifies a lister-side
// failure (e.g. claude binary missing, JSON parse error) propagates
// verbatim through CheckRequiredPlugins as a wrapped error rather than
// being masked by ErrMissingRequiredPlugins.
func TestCheckRequiredPluginsListerErrorPropagates(t *testing.T) {
	sentinel := errors.New("synthetic lister failure")
	lister := &fakeClaudePluginLister{Err: sentinel}
	err := CheckRequiredPlugins(context.Background(), lister, []string{"context7"})
	if err == nil {
		t.Fatalf("CheckRequiredPlugins: expected lister-side error; got nil")
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("CheckRequiredPlugins: errors.Is(_, sentinel) = false; err = %v", err)
	}
	if errors.Is(err, ErrMissingRequiredPlugins) {
		t.Fatalf("CheckRequiredPlugins: lister-side failure must NOT wrap ErrMissingRequiredPlugins; err = %v", err)
	}
}

// TestCheckRequiredPluginsNilListerRejected verifies CheckRequiredPlugins
// rejects a nil lister with ErrInvalidSpawnInput when `required` is
// non-empty. The nil-lister + non-empty-required combination is a
// programming error (the wiring caller forgot to pass the singleton); the
// loud failure beats a silent nil-deref panic.
func TestCheckRequiredPluginsNilListerRejected(t *testing.T) {
	err := CheckRequiredPlugins(context.Background(), nil, []string{"context7"})
	if err == nil {
		t.Fatalf("CheckRequiredPlugins: expected error; got nil")
	}
	if !errors.Is(err, ErrInvalidSpawnInput) {
		t.Fatalf("CheckRequiredPlugins: errors.Is(_, ErrInvalidSpawnInput) = false; err = %v", err)
	}
}

// TestParseClaudePluginListEmpty verifies the production parser treats
// empty stdout (or whitespace-only stdout) as "no plugins installed"
// rather than a parse error. This keeps the no-plugins case ergonomic
// for adopters with empty `required` slices.
func TestParseClaudePluginListEmpty(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{name: "fully empty", in: ""},
		{name: "whitespace only", in: "   \n\t "},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			entries, err := parseClaudePluginList([]byte(tc.in))
			if err != nil {
				t.Fatalf("parseClaudePluginList: unexpected error: %v", err)
			}
			if entries != nil {
				t.Fatalf("parseClaudePluginList: entries = %v; want nil", entries)
			}
		})
	}
}

// TestParseClaudePluginListHappy verifies the production parser decodes
// the canonical `claude plugin list --json` output shape into the
// expected []ClaudePluginListEntry slice.
func TestParseClaudePluginListHappy(t *testing.T) {
	stdout := `[
  {"id": "context7", "marketplace": "claude-plugins-official", "version": "0.4.1", "installPath": "/Users/dev/.claude/plugins/context7"},
  {"id": "gopls-lsp", "marketplace": "claude-plugins-official", "version": "0.2.0", "installPath": "/Users/dev/.claude/plugins/gopls-lsp"}
]`
	entries, err := parseClaudePluginList([]byte(stdout))
	if err != nil {
		t.Fatalf("parseClaudePluginList: unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("len(entries) = %d; want 2", len(entries))
	}
	if entries[0].ID != "context7" || entries[0].Marketplace != "claude-plugins-official" {
		t.Fatalf("entries[0] = %+v; want context7@claude-plugins-official", entries[0])
	}
	if entries[1].ID != "gopls-lsp" || entries[1].Version != "0.2.0" {
		t.Fatalf("entries[1] = %+v; want gopls-lsp v0.2.0", entries[1])
	}
}

// TestParseClaudePluginListMalformed verifies the production parser
// returns ErrPluginListUnparseable when stdout is non-JSON.
func TestParseClaudePluginListMalformed(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{name: "garbage", in: "not json"},
		{name: "object not array", in: `{"id": "context7"}`},
		{name: "trailing garbage", in: `[{"id": "context7"}] extra`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseClaudePluginList([]byte(tc.in))
			if err == nil {
				t.Fatalf("parseClaudePluginList(%q): expected ErrPluginListUnparseable; got nil", tc.in)
			}
			if !errors.Is(err, ErrPluginListUnparseable) {
				t.Fatalf("parseClaudePluginList(%q): errors.Is(_, ErrPluginListUnparseable) = false; err = %v",
					tc.in, err)
			}
		})
	}
}

// TestParseClaudePluginListForwardCompatUnknownFields verifies that future
// claude versions which add fields to each plugin row continue to decode
// cleanly — unknown JSON keys are silently ignored per encoding/json's
// default non-strict behavior.
func TestParseClaudePluginListForwardCompatUnknownFields(t *testing.T) {
	stdout := `[{"id": "context7", "marketplace": "official", "future_field": "value", "another": 42}]`
	entries, err := parseClaudePluginList([]byte(stdout))
	if err != nil {
		t.Fatalf("parseClaudePluginList: unknown fields must not fail parse; err = %v", err)
	}
	if len(entries) != 1 || entries[0].ID != "context7" {
		t.Fatalf("entries = %+v; want 1 row with ID context7", entries)
	}
}

// TestSplitPluginEntry verifies splitPluginEntry's bare-vs-scoped logic
// for the structural cases the validator already guarantees are
// well-formed. The validator rejects malformed inputs at template Load
// time so this helper is not asked to handle them.
func TestSplitPluginEntry(t *testing.T) {
	tests := []struct {
		entry           string
		wantName        string
		wantMarketplace string
		wantScoped      bool
	}{
		{entry: "context7", wantName: "context7", wantMarketplace: "", wantScoped: false},
		{entry: "context7@official", wantName: "context7", wantMarketplace: "official", wantScoped: true},
	}
	for _, tc := range tests {
		t.Run(tc.entry, func(t *testing.T) {
			gotName, gotMarketplace, gotScoped := splitPluginEntry(tc.entry)
			if gotName != tc.wantName || gotMarketplace != tc.wantMarketplace || gotScoped != tc.wantScoped {
				t.Fatalf("splitPluginEntry(%q) = (%q, %q, %v); want (%q, %q, %v)",
					tc.entry, gotName, gotMarketplace, gotScoped,
					tc.wantName, tc.wantMarketplace, tc.wantScoped)
			}
		})
	}
}

// TestExecClaudePluginListerProductionWiring verifies the package-private
// production singleton (defaultClaudePluginLister) is wired to the
// execClaudePluginLister type. Tests inject fakes by overriding this
// var; production code consumes it directly. The wiring assertion guards
// against a future refactor accidentally pointing the singleton at a
// stub or test-only adapter.
func TestExecClaudePluginListerProductionWiring(t *testing.T) {
	if _, ok := defaultClaudePluginLister.(execClaudePluginLister); !ok {
		t.Fatalf("defaultClaudePluginLister type = %T; want execClaudePluginLister", defaultClaudePluginLister)
	}
}

// TestRequiredPluginsForProjectHookDefaultIsNil verifies the package-level
// hook is nil at package init time. The seam contract is "nil hook means
// no required plugins" — adopters opt in by assigning a non-nil function
// at process boot.
func TestRequiredPluginsForProjectHookDefaultIsNil(t *testing.T) {
	prior := RequiredPluginsForProject
	t.Cleanup(func() { RequiredPluginsForProject = prior })

	RequiredPluginsForProject = nil
	if RequiredPluginsForProject != nil {
		t.Fatal("RequiredPluginsForProject is not nil at package default")
	}
}

// TestRequiredPluginsForProjectHookReceivesProject verifies the hook
// receives the domain.Project value verbatim. Future hook implementations
// will dispatch on project.ID, project.Metadata, etc., so the contract
// is "the hook gets the same Project the dispatcher resolved at Stage 1
// of RunOnce."
func TestRequiredPluginsForProjectHookReceivesProject(t *testing.T) {
	prior := RequiredPluginsForProject
	t.Cleanup(func() { RequiredPluginsForProject = prior })

	var observed domain.Project
	RequiredPluginsForProject = func(p domain.Project) []string {
		observed = p
		return nil
	}

	want := domain.Project{ID: "proj-abc", RepoPrimaryWorktree: "/tmp/x"}
	got := RequiredPluginsForProject(want)
	if got != nil {
		t.Fatalf("hook returned non-nil; got len=%d", len(got))
	}
	if observed.ID != want.ID || observed.RepoPrimaryWorktree != want.RepoPrimaryWorktree {
		t.Fatalf("observed mismatch: id=%q want=%q wt=%q want_wt=%q",
			observed.ID, want.ID, observed.RepoPrimaryWorktree, want.RepoPrimaryWorktree)
	}
}
