package dispatcher

import (
	"reflect"
	"testing"
)

// TestIsValidCLIKindClaudeMember asserts the closed enum's only Drop 4c
// member, CLIKindClaude, passes the membership check.
func TestIsValidCLIKindClaudeMember(t *testing.T) {
	t.Parallel()

	if !IsValidCLIKind(CLIKindClaude) {
		t.Fatalf("IsValidCLIKind(CLIKindClaude) = false; want true")
	}
}

// TestIsValidCLIKindCodexNotYetInEnum asserts that "codex" — the kind Drop 4d
// will add — is NOT a member of the closed enum shipped in Drop 4c. The test
// is a regression guard: when Drop 4d lands, this test moves into the
// positive-membership case below.
func TestIsValidCLIKindCodexNotYetInEnum(t *testing.T) {
	t.Parallel()

	if IsValidCLIKind(CLIKind("codex")) {
		t.Fatalf("IsValidCLIKind(\"codex\") = true; want false (codex lands in Drop 4d)")
	}
}

// TestIsValidCLIKindEmptyStringRejected asserts the empty-string sentinel is
// NOT a member of the closed enum. Empty-string default-to-claude semantics
// (F.7.17 locked decision L15) live in the dispatcher's adapter-lookup path,
// NOT in IsValidCLIKind — callers who need the closed-set check must apply
// the L15 default before calling IsValidCLIKind.
func TestIsValidCLIKindEmptyStringRejected(t *testing.T) {
	t.Parallel()

	if IsValidCLIKind(CLIKind("")) {
		t.Fatalf("IsValidCLIKind(\"\") = true; want false (empty-string default lives at adapter-lookup, not here)")
	}
}

// TestIsValidCLIKindArbitraryStringRejected asserts an arbitrary string that
// is neither claude nor a future-known kind fails the membership check.
func TestIsValidCLIKindArbitraryStringRejected(t *testing.T) {
	t.Parallel()

	for _, k := range []CLIKind{"bogus", "Claude", "CLAUDE", " claude ", "claude "} {
		if IsValidCLIKind(k) {
			t.Fatalf("IsValidCLIKind(%q) = true; want false (closed enum is exact-match only)", k)
		}
	}
}

// TestCLIKindClaudeStringValue asserts the const's underlying string is
// exactly "claude" — the value templates.AgentBinding.CLIKind carries on the
// TOML side and the value the dispatcher's adapter-map keys on.
func TestCLIKindClaudeStringValue(t *testing.T) {
	t.Parallel()

	if got, want := string(CLIKindClaude), "claude"; got != want {
		t.Fatalf("string(CLIKindClaude) = %q; want %q", got, want)
	}
}

// TestTerminalReportCostNilSignalsAbsence pins the F.7.17 locked-decision-L11
// contract: a zero-value TerminalReport has Cost == nil, and that nil is the
// canonical "this CLI did not emit cost telemetry" signal — distinct from a
// non-nil pointer to 0.0.
func TestTerminalReportCostNilSignalsAbsence(t *testing.T) {
	t.Parallel()

	var tr TerminalReport
	if tr.Cost != nil {
		t.Fatalf("zero-value TerminalReport.Cost = %v; want nil", tr.Cost)
	}

	zero := 0.0
	withZeroCost := TerminalReport{Cost: &zero}
	if withZeroCost.Cost == nil {
		t.Fatalf("TerminalReport{Cost: &0.0}.Cost = nil; want non-nil pointer to 0")
	}
	if *withZeroCost.Cost != 0.0 {
		t.Fatalf("*TerminalReport{Cost: &0.0}.Cost = %v; want 0.0", *withZeroCost.Cost)
	}
}

// TestBundlePathsZeroValueIsAllEmpty asserts the BundlePaths zero value is the
// empty struct — every field is the zero string. Adapter implementations that
// receive an unpopulated BundlePaths can detect "no bundle" by checking
// Root == "" and reject early.
func TestBundlePathsZeroValueIsAllEmpty(t *testing.T) {
	t.Parallel()

	var bp BundlePaths
	if bp != (BundlePaths{}) {
		t.Fatalf("zero-value BundlePaths is not the empty struct: %+v", bp)
	}
	if bp.Root != "" || bp.SystemPromptPath != "" || bp.SystemAppendPath != "" ||
		bp.StreamLogPath != "" || bp.ManifestPath != "" || bp.ContextDir != "" {
		t.Fatalf("zero-value BundlePaths has non-empty field: %+v", bp)
	}
}

// TestBundlePathsHasNoClaudeSpecificFields asserts BundlePaths carries ONLY
// claude-neutral fields per F.7.17 locked decision L13. If a future change
// adds a field whose name suggests claude-internal layout (plugin,
// claude_plugin, agents, mcp_config, settings, claude), this test fails so
// the reviewer notices the layering violation. Adapters compute their own
// CLI-specific subdirs under BundlePaths.Root.
func TestBundlePathsHasNoClaudeSpecificFields(t *testing.T) {
	t.Parallel()

	forbidden := []string{
		"Plugin", "PluginDir", "PluginPath",
		"ClaudePlugin", "ClaudePluginDir", "ClaudePluginPath",
		"Agents", "AgentsDir", "AgentsPath",
		"MCPConfig", "MCPConfigPath", "McpConfig", "McpConfigPath",
		"Settings", "SettingsPath",
		"Claude", "ClaudeDir", "ClaudePath",
	}

	t_ := reflect.TypeOf(BundlePaths{})
	have := make(map[string]bool, t_.NumField())
	for i := range t_.NumField() {
		have[t_.Field(i).Name] = true
	}
	for _, name := range forbidden {
		if have[name] {
			t.Fatalf("BundlePaths gained forbidden claude-specific field %q (per F.7.17 L13 must stay claude-neutral)", name)
		}
	}
}

// TestBindingResolvedHasNoCommandOrArgsPrefix asserts the REV-1 supersession
// is honored: BindingResolved must NOT carry Command or ArgsPrefix fields.
// The wrapper-interop knob is GONE from Tillsyn; adapters invoke their CLI
// binary directly.
func TestBindingResolvedHasNoCommandOrArgsPrefix(t *testing.T) {
	t.Parallel()

	t_ := reflect.TypeOf(BindingResolved{})
	for i := range t_.NumField() {
		name := t_.Field(i).Name
		if name == "Command" || name == "ArgsPrefix" {
			t.Fatalf("BindingResolved carries forbidden field %q (REV-1 dropped Command and ArgsPrefix)", name)
		}
	}
}

// TestBindingResolvedCarriesEnvAndCLIKind asserts the REV-1-mandated
// replacement fields (Env, CLIKind) are present.
func TestBindingResolvedCarriesEnvAndCLIKind(t *testing.T) {
	t.Parallel()

	t_ := reflect.TypeOf(BindingResolved{})
	have := make(map[string]reflect.Type, t_.NumField())
	for i := range t_.NumField() {
		f := t_.Field(i)
		have[f.Name] = f.Type
	}

	envField, ok := have["Env"]
	if !ok {
		t.Fatalf("BindingResolved missing Env field (REV-1 mandates Env []string)")
	}
	if envField.Kind() != reflect.Slice || envField.Elem().Kind() != reflect.String {
		t.Fatalf("BindingResolved.Env type = %v; want []string", envField)
	}

	cliKindField, ok := have["CLIKind"]
	if !ok {
		t.Fatalf("BindingResolved missing CLIKind field (REV-1 mandates CLIKind CLIKind)")
	}
	if cliKindField != reflect.TypeOf(CLIKind("")) {
		t.Fatalf("BindingResolved.CLIKind type = %v; want CLIKind", cliKindField)
	}
}

// TestBindingResolvedPointerTypedOptionalFields asserts the priority-cascade
// resolver's "absent vs explicit-zero" requirement is honored at the type
// level: optional numeric / boolean / string fields are pointer-typed so the
// resolver can express "lower-priority layer is authoritative" by leaving
// them nil. Per master PLAN L9.
func TestBindingResolvedPointerTypedOptionalFields(t *testing.T) {
	t.Parallel()

	t_ := reflect.TypeOf(BindingResolved{})
	have := make(map[string]reflect.Type, t_.NumField())
	for i := range t_.NumField() {
		f := t_.Field(i)
		have[f.Name] = f.Type
	}

	pointerFields := []string{
		"Model",
		"Effort",
		"MaxTries",
		"MaxBudgetUSD",
		"MaxTurns",
		"AutoPush",
		"CommitAgent",
		"BlockedRetries",
		"BlockedRetryCooldown",
	}
	for _, name := range pointerFields {
		ft, ok := have[name]
		if !ok {
			t.Fatalf("BindingResolved missing optional field %q (priority-cascade resolver needs it pointer-typed)", name)
		}
		if ft.Kind() != reflect.Pointer {
			t.Fatalf("BindingResolved.%s kind = %v; want pointer (absent vs explicit-zero per master PLAN L9)", name, ft.Kind())
		}
	}
}

// TestBindingResolvedZeroValueIsAllAbsent asserts the zero-value
// BindingResolved leaves every pointer field nil and every slice nil. The
// resolver's contract is that an unpopulated input means "no overrides" —
// the adapter falls back to its CLI's defaults.
func TestBindingResolvedZeroValueIsAllAbsent(t *testing.T) {
	t.Parallel()

	var br BindingResolved

	if br.AgentName != "" {
		t.Fatalf("zero-value BindingResolved.AgentName = %q; want empty", br.AgentName)
	}
	if br.CLIKind != "" {
		t.Fatalf("zero-value BindingResolved.CLIKind = %q; want empty", br.CLIKind)
	}
	if br.Env != nil {
		t.Fatalf("zero-value BindingResolved.Env = %v; want nil", br.Env)
	}
	if br.Tools != nil || br.ToolsAllowed != nil || br.ToolsDisallowed != nil {
		t.Fatalf("zero-value BindingResolved Tools fields not nil: %+v / %+v / %+v", br.Tools, br.ToolsAllowed, br.ToolsDisallowed)
	}
	if br.Model != nil || br.Effort != nil || br.CommitAgent != nil {
		t.Fatalf("zero-value BindingResolved string-pointer fields not nil")
	}
	if br.MaxTries != nil || br.MaxTurns != nil || br.BlockedRetries != nil {
		t.Fatalf("zero-value BindingResolved int-pointer fields not nil")
	}
	if br.MaxBudgetUSD != nil {
		t.Fatalf("zero-value BindingResolved.MaxBudgetUSD not nil")
	}
	if br.AutoPush != nil {
		t.Fatalf("zero-value BindingResolved.AutoPush not nil")
	}
	if br.BlockedRetryCooldown != nil {
		t.Fatalf("zero-value BindingResolved.BlockedRetryCooldown not nil")
	}
}

// TestStreamEventHasSevenFields pins the StreamEvent shape spelled out in the
// spawn prompt: Type, Subtype, IsTerminal, Text, ToolName, ToolInput, Raw —
// seven fields. A future change that adds or drops a field forces the
// reviewer to update this assertion AND the doc comments together.
func TestStreamEventHasSevenFields(t *testing.T) {
	t.Parallel()

	t_ := reflect.TypeOf(StreamEvent{})
	if got, want := t_.NumField(), 7; got != want {
		t.Fatalf("StreamEvent has %d fields; want %d", got, want)
	}

	want := []string{"Type", "Subtype", "IsTerminal", "Text", "ToolName", "ToolInput", "Raw"}
	for i, name := range want {
		got := t_.Field(i).Name
		if got != name {
			t.Fatalf("StreamEvent.Field(%d) = %q; want %q (order matters — adapters and tests reference fields positionally in fixtures)", i, got, name)
		}
	}
}

// TestToolDenialShape asserts ToolDenial's two fields are present and typed
// correctly. The struct is the unit of denial-attribution in TerminalReport.
func TestToolDenialShape(t *testing.T) {
	t.Parallel()

	t_ := reflect.TypeOf(ToolDenial{})
	if got, want := t_.NumField(), 2; got != want {
		t.Fatalf("ToolDenial has %d fields; want %d", got, want)
	}

	have := make(map[string]reflect.Type, t_.NumField())
	for i := range t_.NumField() {
		f := t_.Field(i)
		have[f.Name] = f.Type
	}
	if have["ToolName"] == nil || have["ToolName"].Kind() != reflect.String {
		t.Fatalf("ToolDenial.ToolName missing or not string (got %v)", have["ToolName"])
	}
	if have["ToolInput"] == nil {
		t.Fatalf("ToolDenial.ToolInput missing")
	}
	// json.RawMessage is []byte under the hood.
	if have["ToolInput"].Kind() != reflect.Slice || have["ToolInput"].Elem().Kind() != reflect.Uint8 {
		t.Fatalf("ToolDenial.ToolInput type = %v; want json.RawMessage ([]byte)", have["ToolInput"])
	}
}

// TestTerminalReportShape asserts the four fields Cost / Denials / Reason /
// Errors and their types.
func TestTerminalReportShape(t *testing.T) {
	t.Parallel()

	t_ := reflect.TypeOf(TerminalReport{})
	if got, want := t_.NumField(), 4; got != want {
		t.Fatalf("TerminalReport has %d fields; want %d", got, want)
	}

	have := make(map[string]reflect.Type, t_.NumField())
	for i := range t_.NumField() {
		f := t_.Field(i)
		have[f.Name] = f.Type
	}

	costType, ok := have["Cost"]
	if !ok || costType.Kind() != reflect.Pointer || costType.Elem().Kind() != reflect.Float64 {
		t.Fatalf("TerminalReport.Cost type = %v; want *float64 (per F.7.17 L11)", costType)
	}

	denialsType, ok := have["Denials"]
	if !ok || denialsType.Kind() != reflect.Slice {
		t.Fatalf("TerminalReport.Denials type = %v; want []ToolDenial", denialsType)
	}
	if denialsType.Elem() != reflect.TypeOf(ToolDenial{}) {
		t.Fatalf("TerminalReport.Denials element type = %v; want ToolDenial", denialsType.Elem())
	}

	reasonType, ok := have["Reason"]
	if !ok || reasonType.Kind() != reflect.String {
		t.Fatalf("TerminalReport.Reason type = %v; want string", reasonType)
	}

	errorsType, ok := have["Errors"]
	if !ok || errorsType.Kind() != reflect.Slice || errorsType.Elem().Kind() != reflect.String {
		t.Fatalf("TerminalReport.Errors type = %v; want []string", errorsType)
	}
}

// TestCLIAdapterInterfaceShape asserts the CLIAdapter interface declares
// exactly the three methods named in F.7.17 L10 + REV-5: BuildCommand,
// ParseStreamEvent, ExtractTerminalReport. A future change that adds a
// fourth method (or renames one) fails this assertion.
func TestCLIAdapterInterfaceShape(t *testing.T) {
	t.Parallel()

	iface := reflect.TypeOf((*CLIAdapter)(nil)).Elem()
	if got, want := iface.NumMethod(), 3; got != want {
		t.Fatalf("CLIAdapter has %d methods; want %d", got, want)
	}

	want := map[string]bool{
		"BuildCommand":          true,
		"ParseStreamEvent":      true,
		"ExtractTerminalReport": true,
	}
	for i := range iface.NumMethod() {
		name := iface.Method(i).Name
		if !want[name] {
			t.Fatalf("CLIAdapter has unexpected method %q", name)
		}
		delete(want, name)
	}
	if len(want) > 0 {
		t.Fatalf("CLIAdapter missing methods: %v", want)
	}
}

// TestCLIAdapterExtractTerminalReportNotExtractTerminalCost is the explicit
// REV-5 regression guard. The method MUST NOT be named ExtractTerminalCost.
func TestCLIAdapterExtractTerminalReportNotExtractTerminalCost(t *testing.T) {
	t.Parallel()

	iface := reflect.TypeOf((*CLIAdapter)(nil)).Elem()
	for i := range iface.NumMethod() {
		if iface.Method(i).Name == "ExtractTerminalCost" {
			t.Fatalf("CLIAdapter declares forbidden method ExtractTerminalCost (REV-5 renamed it to ExtractTerminalReport)")
		}
	}
}
