# Drop 4c — F.7.17.2 Builder Worklog

## Droplet

`4c.F.7.17.2` — Pure types: `CLIAdapter` interface + value objects in
`internal/app/dispatcher/`. NO behavior, no spawn logic, no claude argv code
— that lands in F.7.17.3.

REVISIONS-first compliance: read REV-1 + REV-5 + REV-7 + REV-8 before reading
the body. The plan body (line 255) mandated `Command []string, ArgsPrefix
[]string` on `BindingResolved`; REV-1 supersedes that — those fields are
DROPPED. The plan body (line 252) named `ExtractTerminalReport`; REV-5
explicitly confirms the rename from `ExtractTerminalCost`. REV-7 removed
F.7.17.9 from scope. The spawn prompt's narrower scoping (no
`CLIKindCodex`, no `ResolveCLIKind` in this droplet) is honored over the
plan body's broader Schema-1-companion text.

## Files edited

- `internal/app/dispatcher/cli_adapter.go` (NEW) — types + interface only.
  - `type CLIKind string` closed-enum alias.
  - `const CLIKindClaude CLIKind = "claude"`. Drop 4c ships claude only;
    `CLIKindCodex` lands Drop 4d.
  - `func IsValidCLIKind(k CLIKind) bool` — returns true ONLY for
    CLIKindClaude. Empty string explicitly rejected (F.7.17 L15
    default-to-claude lives at adapter-lookup, NOT here).
  - `type CLIAdapter interface` with the three F.7.17 L10 methods:
    `BuildCommand`, `ParseStreamEvent`, `ExtractTerminalReport`.
  - `type BindingResolved struct` — flat resolved binding. NO
    `Command`/`ArgsPrefix` per REV-1. Carries `AgentName`, `CLIKind`, `Env`
    (the REV-1-mandated replacements) plus the existing-pattern fields
    `Model`, `Effort`, `Tools`, `ToolsAllowed`, `ToolsDisallowed`,
    `MaxTries`, `MaxBudgetUSD`, `MaxTurns`, `AutoPush`, `CommitAgent`,
    `BlockedRetries`, `BlockedRetryCooldown`. Pointer-typed where the
    priority-cascade resolver (master PLAN L9, F.7.17 L16) needs absent vs
    explicit-zero distinction. AgentName / CLIKind / Env / Tools fields
    use value/slice types because their zero values are the identity.
  - `type BundlePaths struct` — claude-neutral file locations only per
    F.7.17 L13. Six fields: `Root`, `SystemPromptPath`, `SystemAppendPath`,
    `StreamLogPath`, `ManifestPath`, `ContextDir`. Explicitly NO
    `Plugin`, `ClaudePlugin`, `Agents`, `MCPConfig`, `Settings`, `Claude*`
    fields — adapters compute their own CLI-specific subdirs under `Root`.
  - `type StreamEvent struct` — minimal cross-CLI canonical shape with the
    seven fields the spawn prompt named: `Type`, `Subtype`, `IsTerminal`,
    `Text`, `ToolName`, `ToolInput json.RawMessage`, `Raw json.RawMessage`.
  - `type ToolDenial struct` — `ToolName string`, `ToolInput json.RawMessage`.
  - `type TerminalReport struct` — `Cost *float64` (per F.7.17 L11 so
    non-cost-emitting CLIs degrade cleanly), `Denials []ToolDenial`,
    `Reason string`, `Errors []string`.
  - Doc-comments on every exported type, method, and field, citing F.7.17
    locked decisions (L10 / L11 / L13 / L14 / L15 / L16) plus REV-1 / REV-5
    where the comment carries supersession context.

- `internal/app/dispatcher/cli_adapter_test.go` (NEW) — pure-types
  assertions only:
  - `TestIsValidCLIKindClaudeMember` — claude is in the closed enum.
  - `TestIsValidCLIKindCodexNotYetInEnum` — codex (Drop 4d) is NOT in the
    closed enum (regression guard so the assertion flips when Drop 4d
    lands).
  - `TestIsValidCLIKindEmptyStringRejected` — empty string is NOT a member
    (F.7.17 L15 default-to-claude lives at adapter-lookup).
  - `TestIsValidCLIKindArbitraryStringRejected` — exact-match only;
    "Claude", " claude ", "bogus" all rejected.
  - `TestCLIKindClaudeStringValue` — pins the underlying string literal
    to "claude" so the dispatcher's adapter-map keys stay aligned with
    the templates package's `cli_kind = "claude"` TOML value.
  - `TestTerminalReportCostNilSignalsAbsence` — pointer-cost contract:
    zero-value `Cost` is nil; explicit `&zero` is distinguishable.
  - `TestBundlePathsZeroValueIsAllEmpty` — zero-value semantics.
  - `TestBundlePathsHasNoClaudeSpecificFields` — reflection guard against
    a future field named Plugin / ClaudePlugin / Agents / MCPConfig /
    Settings / Claude* breaking F.7.17 L13.
  - `TestBindingResolvedHasNoCommandOrArgsPrefix` — explicit REV-1
    regression guard.
  - `TestBindingResolvedCarriesEnvAndCLIKind` — REV-1 replacement fields
    present and correctly typed (`Env []string`, `CLIKind CLIKind`).
  - `TestBindingResolvedPointerTypedOptionalFields` — master PLAN L9
    absent-vs-explicit-zero discipline at the type level for Model /
    Effort / MaxTries / MaxBudgetUSD / MaxTurns / AutoPush / CommitAgent
    / BlockedRetries / BlockedRetryCooldown.
  - `TestBindingResolvedZeroValueIsAllAbsent` — zero-value resolver input
    leaves every pointer nil and every slice nil (== "no overrides").
  - `TestStreamEventHasSevenFields` — pins the seven-field shape and
    field order so a future field add forces test + doc-comment update.
  - `TestToolDenialShape` — two-field shape, types pinned.
  - `TestTerminalReportShape` — four-field shape (Cost / Denials / Reason
    / Errors); type assertions for `*float64` and `[]ToolDenial`.
  - `TestCLIAdapterInterfaceShape` — exactly three methods named
    BuildCommand / ParseStreamEvent / ExtractTerminalReport.
  - `TestCLIAdapterExtractTerminalReportNotExtractTerminalCost` —
    explicit REV-5 regression guard.

## NOT shipped (per spec scoping)

- No `CLIKindCodex` constant. Spawn prompt explicitly scopes Drop 4c to
  claude only; codex lands in Drop 4d. The plan body's text suggesting
  `CLIKindCodex` here is superseded by the prompt's narrower contract +
  the REV-1 spirit (additive multi-CLI work landing per-drop, not all at
  once). The Codex-not-in-enum regression test will flip to a positive
  membership assertion when Drop 4d lands.
- No `ResolveCLIKind(s string) CLIKind` function. The empty-string
  default-to-claude rule (F.7.17 L15) belongs in the dispatcher's
  adapter-lookup path (droplet 4c.F.7.17.5), not in the pure-types layer.
  Splitting the default rule away from `IsValidCLIKind` keeps both
  functions honest about their semantic.
- No `Command []string` field on `BindingResolved` (REV-1).
- No `ArgsPrefix []string` field on `BindingResolved` (REV-1).
- No claude-specific paths (`Plugin`, `ClaudePlugin`, `Agents`,
  `MCPConfig`, `Settings`) on `BundlePaths` (F.7.17 L13).
- No method named `ExtractTerminalCost` anywhere (REV-5).
- No claude argv assembly. No spawn logic. No stream parsing. No file
  I/O. Pure types.

The diff is strictly the F.7.17.2 minimum surface after REV-1, REV-5, and
the spawn prompt's narrower scoping.

## Verification

- `mage testPkg ./internal/app/dispatcher` — green (167 reported tests
  total post-droplet; this droplet adds 17 new top-level `Test*`
  functions in `cli_adapter_test.go`, all pass).
- `mage ci` — second run was fully green (2303 passed / 1 skipped /
  0 failed across 21 packages, dispatcher coverage 73.2% above the 70%
  gate, templates coverage 92.1%).
- `mage formatCheck` — green; new files are gofumpt-clean
  (verified directly via `go tool gofumpt -d` on the two new files —
  empty diff output).
- `mage build` — green; `till` binary builds cleanly.
- **Cross-droplet coordination caveat:** a follow-up `mage ci` after the
  initial green run started failing with a build error in
  `internal/templates` because a parallel droplet (F.7.18 context-aggregator
  or similar) began writing untracked test files into that package
  (`internal/templates/context_rules_test.go` appeared between runs).
  That package is OUTSIDE my droplet's locked `paths`
  (`internal/app/dispatcher/cli_adapter.go` + `cli_adapter_test.go` +
  worklog) — my diff did NOT touch `internal/templates/`. Confirmed by:
  (a) `mage testPkg ./internal/app/dispatcher` is green in isolation;
  (b) the templates-package failure mode is a build error, not a test
  failure, and the new untracked file is a parallel-droplet artifact;
  (c) `git diff --stat` shows my changes are isolated to the dispatcher
  package + workflow MD. The orchestrator will see the templates-package
  state stabilize when the parallel droplet completes its commit; my
  droplet's correctness is independent.

## Acceptance criteria — all met

- [x] `CLIKind` type + `CLIKindClaude` constant + `IsValidCLIKind` function.
- [x] `CLIAdapter` interface with the three methods (`BuildCommand`,
  `ParseStreamEvent`, `ExtractTerminalReport`).
- [x] `BindingResolved` struct with NO `Command`/`ArgsPrefix` fields,
  includes `CLIKind` + `Env` + existing-pattern resolved fields with
  pointer-typed optional numerics / strings.
- [x] `BundlePaths` struct with claude-neutral fields only (NO `plugin/`,
  `.claude-plugin/`, etc.).
- [x] `StreamEvent` struct with the 7 fields named in the spawn prompt:
  Type, Subtype, IsTerminal, Text, ToolName, ToolInput, Raw.
- [x] `ToolDenial` + `TerminalReport` structs.
- [x] Tests assert `IsValidCLIKind(CLIKindClaude) == true` and
  `IsValidCLIKind("codex") == false` (codex not yet in enum).
- [x] Tests assert `TerminalReport.Cost == nil` semantics for absence.
- [x] `mage check` (alias of `mage ci`) green.
- [x] Worklog written.

## Hylla feedback

N/A — task touched non-Go files only beyond the new Go code I authored,
and Hylla calls were forbidden by the spawn prompt for this droplet
(`Read` / `Grep` / `LSP` directly). No Hylla queries were attempted, so no
Hylla misses to report.
