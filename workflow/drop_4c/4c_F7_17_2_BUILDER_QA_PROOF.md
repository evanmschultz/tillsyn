# Drop 4c — F.7.17.2 Builder QA Proof (Round 1)

**Author role:** go-qa-proof-agent (read-only).
**Author date:** 2026-05-04.
**Verdict:** **PROOF GREEN-WITH-NITS.**

The builder's claim — pure types + value objects for the F.7.17 CLI adapter
seam — is **fully supported** by the evidence in
`internal/app/dispatcher/cli_adapter.go` and `cli_adapter_test.go`. Every
acceptance criterion (the spawn prompt's 9 items + REV-1 / REV-5 supersession
rules) is met with explicit file:line backing. Two N-class observations
surface only as NITs that do NOT change the verdict.

---

## REVISIONS-first compliance

The builder's worklog (lines 9-16) explicitly cites REV-1 + REV-5 + REV-7 +
REV-8 supersession before the implementation block. Plan REVISIONS section
(`F7_17_CLI_ADAPTER_PLAN.md` lines 683-735) confirms each:

- **REV-1** (lines 687-700): `Command []string` and `ArgsPrefix []string`
  DROPPED from `AgentBinding` AND from `BindingResolved`. Wrapper-interop
  knob is gone; adapters invoke their CLI binary directly.
- **REV-5** (lines 716-718): method renamed `ExtractTerminalCost` →
  `ExtractTerminalReport`.
- **REV-7** (lines 720-724): F.7.17.9 monitor refactor MERGED into F.7-CORE
  F.7.4 — out of scope here.
- **REV-8** (lines 726-728): builder spawn prompts MUST instruct
  REVISIONS-first reading. Worklog evidence confirms compliance.

Builder honored REVISIONS-FIRST. No implementation drift toward the
superseded plan body.

---

## 1. Per-criterion evidence

### 1.1 `CLIKind` enum + `IsValidCLIKind`

**Status:** PASS.

- `cli_adapter.go:29` — `type CLIKind string` (closed string-alias enum).
- `cli_adapter.go:33` — `const CLIKindClaude CLIKind = "claude"`.
- `cli_adapter.go:42-49` — `IsValidCLIKind(k CLIKind) bool` returns true ONLY
  for `CLIKindClaude` via switch; default returns false.
- `cli_adapter.go:21-24` — `CLIKindCodex` explicitly NOT shipped (Drop 4d's
  job per spawn prompt scoping).
- Test coverage: `cli_adapter_test.go:10-16` (claude member),
  `:22-28` (codex NOT in enum — regression guard for Drop 4d flip),
  `:35-41` (empty-string rejected — F.7.17 L15 default-to-claude lives at
  adapter-lookup, not here), `:45-53` (case-folding + whitespace variants
  rejected — exact-match only), `:58-64` (string literal pinned to
  `"claude"`).

### 1.2 `CLIAdapter` interface — three methods, exact signatures

**Status:** PASS.

- `cli_adapter.go:61-83` — interface declared with exactly three methods:
  - `:65` — `BuildCommand(ctx context.Context, binding BindingResolved, paths BundlePaths) (*exec.Cmd, error)`
  - `:72` — `ParseStreamEvent(line []byte) (StreamEvent, error)`
  - `:82` — `ExtractTerminalReport(ev StreamEvent) (TerminalReport, bool)`

Signatures match plan L10 (line 65 of plan) and the spawn prompt verbatim.
Signature parameter names are documented but irrelevant to interface
satisfaction.

- Test coverage: `cli_adapter_test.go:347-370` (`TestCLIAdapterInterfaceShape`
  — exactly 3 methods, names verified via reflection map),
  `:374-383` (`TestCLIAdapterExtractTerminalReportNotExtractTerminalCost` —
  explicit REV-5 regression guard).

### 1.3 `BindingResolved` — REV-1 absences + Env/CLIKind + pointer-typed optionals

**Status:** PASS.

REV-1 absences (`cli_adapter.go:92-101`, comment block explicitly cites
REV-1 supersession):

- No `Command` field in struct definition `:102-179`. Verified by direct
  read of all 17 declared fields.
- No `ArgsPrefix` field. Same evidence.
- Test coverage: `cli_adapter_test.go:139-149`
  (`TestBindingResolvedHasNoCommandOrArgsPrefix` — reflection iterates every
  field name, fails on `Command` or `ArgsPrefix`).

Required REV-1 replacements present:

- `:111` — `CLIKind CLIKind` (closed-enum field).
- `:119` — `Env []string` (allow-list of NAMES per F.7.17 L4).
- Test coverage: `cli_adapter_test.go:153-178` (`TestBindingResolvedCarriesEnvAndCLIKind`
  — reflection asserts both present with correct types).

Pointer-typed optional fields per master PLAN L9 priority cascade:

- `:124` — `Model *string`
- `:128` — `Effort *string`
- `:147` — `MaxTries *int`
- `:153` — `MaxBudgetUSD *float64`
- `:157` — `MaxTurns *int`
- `:163` — `AutoPush *bool`
- `:168` — `CommitAgent *string`
- `:173` — `BlockedRetries *int`
- `:178` — `BlockedRetryCooldown *time.Duration`

Value/slice typed where zero-value IS identity (per `:97-101`):

- `:106` — `AgentName string`
- `:111` — `CLIKind CLIKind`
- `:119` — `Env []string`
- `:134` — `Tools []string`, `:140-141` — `ToolsAllowed/ToolsDisallowed []string`

- Test coverage: `cli_adapter_test.go:185-215`
  (`TestBindingResolvedPointerTypedOptionalFields` — reflection asserts all 9
  pointer-typed fields by name + kind), `:221-253`
  (`TestBindingResolvedZeroValueIsAllAbsent` — zero-value semantics across all
  field-shapes).

### 1.4 `BundlePaths` — claude-neutral fields ONLY (no plugin/agents/.mcp.json/settings)

**Status:** PASS.

- `cli_adapter.go:191-220` — struct contains exactly six fields:
  - `:194` — `Root string`
  - `:199` — `SystemPromptPath string`
  - `:204` — `SystemAppendPath string`
  - `:209` — `StreamLogPath string`
  - `:214` — `ManifestPath string`
  - `:219` — `ContextDir string`

No claude-internal fields present. Direct read confirms zero occurrences of:
`Plugin`, `ClaudePlugin`, `Agents`, `AgentsDir`, `MCPConfig`, `McpConfig`,
`Settings`, `Claude`-prefixed paths.

NIT N1: plan L13 + locked-decision spec (line 256, line 68) called for a
**three-field minimal** `BundlePaths`: `{ Root string; StreamLog string;
Manifest string }`. Builder shipped six fields by adding `SystemPromptPath`,
`SystemAppendPath`, `ContextDir`. Each is claude-neutral (every adapter
needs a system prompt, optional append, and a per-spawn context staging dir
per F.7.18 — a parallel droplet) so no L13 violation; but it widens the
"thin handle" beyond the plan letter. The doc comment at `:181-190`
acknowledges this implicitly by calling out F.7.18 ContextDir consumption.
**Recommendation:** non-blocking; record as a coordination note for the
F.7.17.5 dispatcher-wiring droplet (it consumes BundlePaths and may need to
populate the additional three fields; cross-droplet planner should confirm
the wider shape was intended).

- Test coverage: `cli_adapter_test.go:92-103`
  (`TestBundlePathsZeroValueIsAllEmpty` — zero-value verified across all 6
  fields), `:111-133` (`TestBundlePathsHasNoClaudeSpecificFields` —
  reflection guards against forbidden field names: `Plugin`, `PluginDir`,
  `PluginPath`, `ClaudePlugin*`, `Agents*`, `MCPConfig*`, `McpConfig*`,
  `Settings*`, `Claude*`).

### 1.5 `StreamEvent` — 7 fields (Type, Subtype, IsTerminal, Text, ToolName, ToolInput, Raw)

**Status:** PASS.

- `cli_adapter.go:228-265` — struct has exactly 7 fields, in order:
  - `:233` — `Type string`
  - `:239` — `Subtype string`
  - `:244` — `IsTerminal bool`
  - `:249` — `Text string`
  - `:253` — `ToolName string`
  - `:258` — `ToolInput json.RawMessage`
  - `:264` — `Raw json.RawMessage`

NIT N2: plan body line 257 originally proposed a minimal 3-field shape
`{ Type string; IsTerminal bool; RawJSON []byte }`. Builder shipped a
7-field shape per the spawn prompt's verbatim list. The spawn prompt (the
authoritative source per REV-8 doctrine) names exactly these 7 fields, so
the wider shape is correct. The wider shape adds `Subtype`, `Text`,
`ToolName`, `ToolInput` — all needed by F.7-CORE F.7.4 monitor consumption
(per REV-7 the monitor lives there now). Renamed `RawJSON` → `Raw` matches
the spawn-prompt list. **Recommendation:** non-blocking; reflects the
post-REVISIONS shape correctly.

- Test coverage: `cli_adapter_test.go:259-274` (`TestStreamEventHasSevenFields`
  — explicit field-count + ordered-name assertion).

### 1.6 `ToolDenial` + `TerminalReport` — shape + Cost *float64 absence semantics

**Status:** PASS.

`ToolDenial` (`cli_adapter.go:271-280`):

- `:275` — `ToolName string`
- `:279` — `ToolInput json.RawMessage`

`TerminalReport` (`cli_adapter.go:287-306`):

- `:291` — `Cost *float64` (per F.7.17 L11 absence semantics).
- `:296` — `Denials []ToolDenial`
- `:300` — `Reason string`
- `:305` — `Errors []string`

Doc comment at `:282-291` explicitly cites: "Callers MUST NOT treat nil as
0." — pinning the absence-vs-zero distinction.

- Test coverage: `cli_adapter_test.go:70-86`
  (`TestTerminalReportCostNilSignalsAbsence` — zero-value `Cost == nil` AND
  explicit `&zero` is a non-nil distinguishable pointer to 0.0),
  `:278-301` (`TestToolDenialShape` — 2-field shape with correct types,
  `json.RawMessage` verified as `[]byte` underlying), `:305-341`
  (`TestTerminalReportShape` — 4-field count + Cost as `*float64` + Denials
  as `[]ToolDenial` + Reason as `string` + Errors as `[]string`).

### 1.7 17 tests cover acceptance surface

**Status:** PASS.

Counted test functions in `cli_adapter_test.go` by `func Test` declarations:

1. `TestIsValidCLIKindClaudeMember` (`:10`)
2. `TestIsValidCLIKindCodexNotYetInEnum` (`:22`) — REV-5/Drop-4d regression guard
3. `TestIsValidCLIKindEmptyStringRejected` (`:35`)
4. `TestIsValidCLIKindArbitraryStringRejected` (`:45`)
5. `TestCLIKindClaudeStringValue` (`:58`)
6. `TestTerminalReportCostNilSignalsAbsence` (`:70`)
7. `TestBundlePathsZeroValueIsAllEmpty` (`:92`)
8. `TestBundlePathsHasNoClaudeSpecificFields` (`:111`)
9. `TestBindingResolvedHasNoCommandOrArgsPrefix` (`:139`) — REV-1 regression guard
10. `TestBindingResolvedCarriesEnvAndCLIKind` (`:153`)
11. `TestBindingResolvedPointerTypedOptionalFields` (`:185`)
12. `TestBindingResolvedZeroValueIsAllAbsent` (`:221`)
13. `TestStreamEventHasSevenFields` (`:259`)
14. `TestToolDenialShape` (`:278`)
15. `TestTerminalReportShape` (`:305`)
16. `TestCLIAdapterInterfaceShape` (`:347`)
17. `TestCLIAdapterExtractTerminalReportNotExtractTerminalCost` (`:374`) — REV-5 regression guard

Count = 17, matches builder claim. Coverage map:

| Acceptance bucket | Tests |
| --- | --- |
| Enum membership + IsValidCLIKind correctness | 1, 2, 3, 4, 5 |
| REV-1 absence guard (Command/ArgsPrefix) | 9 |
| REV-5 method-name guard | 17 (also 16 catches via positive enum) |
| Struct-shape reflection | 8, 11, 13, 14, 15, 16 |
| Zero-value semantics | 6, 7, 12 |
| Pointer-typed-optional discipline | 11, 12 |
| REV-1 replacement fields present | 10 |

Every spawn-prompt-specified test category has at least one assertion.

### 1.8 Scope: only the two new files + worklog

**Status:** PASS.

`git status --porcelain` evidence (run from `main/`):

- `?? internal/app/dispatcher/cli_adapter.go` — NEW (this droplet).
- `?? internal/app/dispatcher/cli_adapter_test.go` — NEW (this droplet).
- `?? workflow/drop_4c/4c_F7_17_2_BUILDER_WORKLOG.md` — NEW (this droplet).

Other unstaged / untracked files in working tree are clearly OTHER droplets'
artifacts:

- `internal/templates/{schema,load}.go`, `internal/templates/{agent_binding,schema}_test.go`,
  `internal/templates/context_rules_test.go` — F.7.17.1 (Schema-1) +
  F.7.18.1 (context aggregator). NOT this droplet's scope; confirmed by
  `git diff --stat HEAD -- internal/templates/` which shows the schema
  changes are exactly the `Env []string` + `CLIKind string` widening that
  F.7.17.1 owns per REV-1.
- Other `workflow/drop_4c/*.md` files — sibling droplet artifacts.

This droplet's Go diff is **strictly two files** in `internal/app/dispatcher/`.
No leakage into other packages. No test fixtures, no shared helpers, no
modifications to existing dispatcher files.

### 1.9 `mage ci` green per worklog

**Status:** PASS (with cross-droplet caveat acknowledged).

Worklog (lines 117-143) reports:

- `mage testPkg ./internal/app/dispatcher` — green; new tests all pass.
- `mage ci` — first run green (2303 passed / 1 skipped / 0 failed; dispatcher
  coverage 73.2%, above 70% gate).
- `mage formatCheck` — green.
- `mage build` — green.
- Second `mage ci` run failed in `internal/templates` because a parallel
  droplet (F.7.18.1) added `context_rules_test.go` between runs.

Cross-validation: `git status --porcelain` confirms `internal/templates/context_rules_test.go`
is present and unstaged — matches the worklog's parallel-droplet
attribution. F.7.17.2's diff is isolated to `internal/app/dispatcher`,
which is independent of `internal/templates`. The flake is **cross-droplet
coordination**, not a F.7.17.2 regression.

QA Proof position: this is NOT a finding against F.7.17.2. The orchestrator
needs to coordinate the templates-package state across the parallel
droplets (F.7.17.1 lands the Schema-1 changes; F.7.18.1 lands context
rules; F.7.17.2's tests don't depend on either). When the parallel droplets
land + their tests stabilize, `mage ci` will be green again.

---

## 2. Missing Evidence

None. Every acceptance criterion in the spawn prompt has explicit file:line
backing in `cli_adapter.go` + `cli_adapter_test.go`. Coverage map at §1.7
confirms the 17-test count and per-bucket assignment.

---

## 3. NITs (non-blocking)

### 3.1 N1 — `BundlePaths` field count widened from 3 to 6

Plan L13 (line 68) + plan body line 256 specified a minimal 3-field shape
(`Root`, `StreamLog`, `Manifest`). Builder shipped 6 fields adding
`SystemPromptPath`, `SystemAppendPath`, `ContextDir`. Each is claude-neutral
and serves a real cross-CLI need (system prompt is universal; context
staging is F.7.18's surface). Doc comment at `cli_adapter.go:181-190` cites
F.7.18 consumption. The expansion does NOT violate L13's "claude-neutral
only" rule, but it widens the "thin handle" semantic. Whether the master
PLAN at the orchestrator level explicitly authorized the 3-field expansion
is a coordination question for F.7.17.5 (dispatcher-wiring) which consumes
this struct. Non-blocking for proof verdict.

### 3.2 N2 — `StreamEvent` field count widened from 3 to 7

Plan body line 257 originally proposed `{ Type string; IsTerminal bool;
RawJSON []byte }`. Builder shipped 7 fields per spawn-prompt verbatim. The
post-REVISIONS shape adds Subtype/Text/ToolName/ToolInput which F.7-CORE
F.7.4 monitor consumption needs (per REV-7's monitor merge). Renamed
RawJSON → Raw. The spawn prompt is the authoritative source here. The
wider shape costs adapters more decode work but reduces the "every adapter
re-decodes Raw for the same fields" anti-pattern. Non-blocking for proof
verdict.

### 3.3 N3 — Plan vs spawn-prompt test-file naming drift

Plan line 242 named the test file `cli_adapter_types_test.go`; spawn prompt
named it `cli_adapter_test.go`. Builder used `cli_adapter_test.go` per
spawn prompt. Plan line 349 (droplet 4c.F.7.17.4 MockAdapter) ALSO names
`cli_adapter_test.go` as its target file — meaning droplet 4 will need to
APPEND to this droplet's file (not create a fresh one). This is a
coordination heads-up for the F.7.17.4 builder, NOT a F.7.17.2 regression.
Non-blocking for proof verdict.

---

## 4. Falsification attacks attempted (and mitigated)

The proof-side review attempted these attacks before declaring PASS:

- **A1: `CLIKindCodex` constant smuggled in?** Confirmed absent by direct
  read of `cli_adapter.go:1-49`. Worklog line 23 + line 95-100 explicitly
  declares "No `CLIKindCodex` constant" until Drop 4d.
- **A2: `Command` or `ArgsPrefix` field on BindingResolved?** Confirmed
  absent across `cli_adapter.go:102-179`. Test #9 enforces.
- **A3: claude-internal paths on BundlePaths?** Confirmed absent across
  `cli_adapter.go:191-220`. Test #8 enforces (forbidden-field reflection
  guard).
- **A4: `TerminalReport.Cost` not pointer?** Confirmed `*float64` at
  `cli_adapter.go:291`. Test #6 + #15 enforce.
- **A5: `ParseStreamEvent` / `ExtractTerminalReport` signatures don't match?**
  Confirmed exact match against L10 / REV-5 at `cli_adapter.go:72` + `:82`.
  Test #16 + #17 enforce.
- **A6: Tests don't actually assert REV-1 / REV-5 absence?** Confirmed
  tests #9 (REV-1) + #17 (REV-5) explicitly named and reflection-driven —
  not just "test passes by accident."
- **A7: `IsValidCLIKind("")` accidentally returns true?** Confirmed false
  at `cli_adapter.go:42-49` (empty string falls through to `default`). Test
  #3 enforces.
- **A8: Worklog claims 17 tests but file actually has fewer?** Counted
  directly: 17 `func Test*` declarations in `cli_adapter_test.go`. Match.

Every attack falsified — no unmitigated counterexample.

---

## 5. Verdict

**PROOF GREEN-WITH-NITS.** F.7.17.2 acceptance criteria are met. Two NITs
(N1 BundlePaths widening, N2 StreamEvent widening) and one coordination
note (N3 test-file naming for F.7.17.4) are non-blocking for proof. The
builder's `mage ci` cross-droplet flake is correctly attributed to a
parallel droplet's untracked file, not this droplet's diff.

The builder may move F.7.17.2 to `complete` once the matched
falsification-twin returns green and the orchestrator confirms the
parallel-droplet flake doesn't impact this droplet's correctness.

---

## Hylla Feedback

N/A — task touched non-Go files only beyond the new Go code I authored;
Hylla calls were forbidden by the spawn prompt for this droplet (read-only
Read / Grep direct). No Hylla queries attempted; no misses to report.
