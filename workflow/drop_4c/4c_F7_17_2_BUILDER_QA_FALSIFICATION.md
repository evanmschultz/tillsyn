# Drop 4c — F.7.17.2 Builder QA Falsification (Round 1)

**Reviewer:** go-qa-falsification-agent (subagent).
**Date:** 2026-05-04.
**Scope:** Read-only adversarial review of the F.7.17.2 builder output —
`internal/app/dispatcher/cli_adapter.go` (NEW, 307 lines) +
`internal/app/dispatcher/cli_adapter_test.go` (NEW, 17 tests) +
`workflow/drop_4c/4c_F7_17_2_BUILDER_WORKLOG.md`.

REVISIONS-first compliance: REV-1 (drop `Command`/`ArgsPrefix`), REV-5
(rename `ExtractTerminalCost` → `ExtractTerminalReport`), REV-7 (F.7.17.9
merged into F.7-CORE F.7.4 — not in this droplet's scope), and REV-8
(REVISIONS-first reading procedural rule) all read before the body.

---

## Per-attack verdicts

### A1 — Interface signature drift — REFUTED

Plan L10 (PLAN line 65 + REV-5) requires:

- `BuildCommand(ctx, BindingResolved, BundlePaths) (*exec.Cmd, error)`
- `ParseStreamEvent(line []byte) (StreamEvent, error)`
- `ExtractTerminalReport(StreamEvent) (TerminalReport, bool)`

`internal/app/dispatcher/cli_adapter.go` lines 65 / 72 / 82:

```go
BuildCommand(ctx context.Context, binding BindingResolved, paths BundlePaths) (*exec.Cmd, error)
ParseStreamEvent(line []byte) (StreamEvent, error)
ExtractTerminalReport(ev StreamEvent) (TerminalReport, bool)
```

All three signatures match the spec exactly. The bool comes second in
`ExtractTerminalReport`'s return tuple as `(TerminalReport, bool)`, not
the inverted `(StreamEvent, *TerminalReport)` form the attack proposed.
`TestCLIAdapterInterfaceShape` (test file lines 347-370) asserts NumMethod
== 3 and pins each name; a future drift fails the test.

### A2 — REV-5 rename airtight — REFUTED

`rg ExtractTerminalCost internal/app/dispatcher/` returns SEVEN matches,
all of which are legitimate (zero leftover symbol references):

- `cli_adapter.go:17` — supersession header doc-comment (REV-5 trace).
- `cli_adapter.go:81` — method's own doc-comment ("Renamed from
  ExtractTerminalCost per Drop 4c F.7.17 REV-5").
- `cli_adapter_test.go:372` / `:373` / `:374` — `TestCLIAdapterExtract
  TerminalReportNotExtractTerminalCost` doc-comment + function name.
- `cli_adapter_test.go:379` / `:380` — the regression-guard string
  literal `"ExtractTerminalCost"` and the failure message.

Zero hits where `ExtractTerminalCost` is used as an actual symbol /
method declaration / interface member. The rename is airtight; the only
mentions are documentation (REV-5 audit trail) and a reflection-based
regression guard that ASSERTS the rename held.

### A3 — REV-1 absence airtight — REFUTED

`rg "Command\s+\[\]string|ArgsPrefix" internal/app/dispatcher/` returns
EIGHT matches, all legitimate:

- `cli_adapter.go:16` / `:91-94` — supersession header + struct-level
  doc-comment ("BindingResolved does NOT carry Command []string or
  ArgsPrefix []string. The wrapper-interop knob is gone from Tillsyn …").
- `cli_adapter_test.go:135-148` —
  `TestBindingResolvedHasNoCommandOrArgsPrefix` — explicit reflection
  guard that fails if either field re-appears.

Zero hits where `Command` or `ArgsPrefix` is declared as a struct field.
The negative regression guard (`if name == "Command" || name == "Args
Prefix"` in the test) is the load-bearing assertion; doc-comments are
context only.

### A4 — `BundlePaths` claude-leakage — NIT (false positive)

`rg -i "plugin|mcp|settings\.json|agents/" cli_adapter.go` returns three
matches; ALL are in the `BundlePaths` doc-comment block (lines 184-186)
and are NEGATIVE-form ("CLI-specific subdirs (claude's plugin/,
.claude-plugin/, agents/, .mcp.json, settings.json) are NOT here.
Adapters materialize their own subdirs under Root themselves so the seam
stays narrow.").

The mentions are documentation explaining what is INTENTIONALLY OMITTED.
Field names themselves are claude-neutral: `Root`, `SystemPromptPath`,
`SystemAppendPath`, `StreamLogPath`, `ManifestPath`, `ContextDir`. None
carry `Claude`, `Plugin`, `MCPConfig`, `Settings`, or `Agents` tokens.
`TestBundlePathsHasNoClaudeSpecificFields` (test lines 111-133)
reflectively rejects 14 forbidden field names.

NIT (informational, not a counterexample): the doc-comment listing
claude-specific directories by name does technically anchor Claude
vocabulary INTO the file's text. A defensive reviewer might prefer the
omission-list be in a separate spec doc rather than the `BundlePaths`
godoc. This is a wording preference, not a contract violation — the
field set is conformant, the test guard is correct.

### A5 — Pointer-typed-optional discipline — REFUTED

`BindingResolved` field-by-field walk (cli_adapter.go lines 102-179):

| Field                  | Type             | Pointer? | Justification                                            |
|------------------------|------------------|----------|----------------------------------------------------------|
| AgentName              | string           | No       | Always populated by resolver (no "absent" form)         |
| CLIKind                | CLIKind          | No       | Default-to-claude per L15; resolver populates           |
| Env                    | []string         | No       | Nil slice IS the identity ("no forwarded env names")    |
| Model                  | *string          | YES      | Master PLAN L9 absent-vs-explicit-empty                 |
| Effort                 | *string          | YES      | Master PLAN L9                                           |
| Tools                  | []string         | No       | Nil distinct from empty per doc-comment line 132-133    |
| ToolsAllowed           | []string         | No       | Same as Tools                                            |
| ToolsDisallowed        | []string         | No       | Same as Tools                                            |
| MaxTries               | *int             | YES      | Zero is invalid value; nil means "use dispatcher default"|
| MaxBudgetUSD           | *float64         | YES      | Explicit-zero ("no spend") distinct from absent         |
| MaxTurns               | *int             | YES      | Absent vs explicit-zero                                 |
| AutoPush               | *bool            | YES      | Explicit-false distinct from absent                     |
| CommitAgent            | *string          | YES      | Absent vs explicit empty-string                         |
| BlockedRetries         | *int             | YES      | Absent vs explicit-zero                                 |
| BlockedRetryCooldown   | *time.Duration   | YES      | Absent vs explicit-zero                                 |

`TestBindingResolvedPointerTypedOptionalFields` (test lines 185-215)
reflectively asserts the 9 pointer fields. `TestBindingResolvedZeroValue
IsAllAbsent` (test lines 221-253) asserts the zero-value form. The
pointer-vs-value choice for slice fields (Env, Tools, ToolsAllowed,
ToolsDisallowed) is principled and documented per-field.

One borderline: `Tools` doc-comment (line 132-133) says "Nil means 'use
CLI default' — distinct from an explicit empty slice which would mean
'deny all tools' if the CLI supports that semantic." This means the
nil/empty distinction IS load-bearing for `Tools` — and Go nil-slice vs
empty-slice IS distinguishable via reflection / `len() == 0 && != nil`.
This is borderline but defensible — slices already encode the
distinction natively. NOT a counterexample.

### A6 — `CLIKind` enum closure — REFUTED

`rg "CLIKindCodex|Codex" cli_adapter.go` returns 2 matches, both in
doc-comments forecasting Drop 4d (lines 21, 38). No `const CLIKindCodex
CLIKind = "codex"` declaration. `IsValidCLIKind(CLIKind("codex"))`
returns false (test lines 22-28). `TestIsValidCLIKindCodexNotYetInEnum`
is a regression guard that flips to a positive-membership case when
Drop 4d lands. The closure is correct.

### A7 — `StreamEvent` field set — NIT (substantive, low-severity)

The plan body (PLAN line 257) specifies `StreamEvent struct (per L14):
minimal cross-CLI shape — { Type string; IsTerminal bool; RawJSON
[]byte }` — THREE fields.

The builder shipped SEVEN fields: `Type, Subtype, IsTerminal, Text,
ToolName, ToolInput json.RawMessage, Raw json.RawMessage`.

The worklog (line 44) cites the spawn prompt's narrower contract as
authority: "the spawn prompt's narrower scoping … is honored over the
plan body's broader Schema-1-companion text" — but this only covered
REV-1 (Command/ArgsPrefix) and `CLIKindCodex`. The spawn prompt's seven
StreamEvent fields are not addressed by any REV in the plan; they are
expansion-not-supersession.

**NIT (not CONFIRMED counterexample) because:**
1. The orchestrator authored the spawn prompt with the seven-field
   shape; the builder followed orchestrator instruction.
2. None of the REV entries covers StreamEvent — but neither does any
   REV explicitly LIMIT it to 3 fields. The plan body's field count is
   not a locked decision (L14 says "minimal cross-CLI canonical shape"
   without nailing the exact field set).
3. The added fields (Subtype, Text, ToolName, ToolInput) are defensive
   over-collection. Adapters that don't surface tool details leave
   them empty.
4. `TestStreamEventHasSevenFields` pins the seven-field shape so future
   drift is caught.

**The substantive concern:** plan-body L13/L14 specified narrow shapes;
builder expanded both. If the future planner thinks plan body is
authoritative, builder did expand-by-1 on `BundlePaths` (3→6) and by-4
on `StreamEvent` (3→7). This is a process trace concern, not a code
correctness concern. Recommend: orchestrator records this as a
plan-body/spawn-prompt drift to surface during F.7.17.3 + F.7.17.4
planning so consumers know the canonical shape is the seven-field one.

### A8 — `TerminalReport.Cost *float64` semantics — REFUTED

Cost field declared `*float64` on line 291. `TestTerminalReportCostNil
SignalsAbsence` (test lines 70-86) asserts:
- zero-value `TerminalReport{}.Cost == nil`.
- `TerminalReport{Cost: &zero}.Cost != nil`.
- `*TerminalReport{Cost: &zero}.Cost == 0.0`.

The pointer-cost contract is correctly typed and tested.
`TestTerminalReportShape` (test lines 305-341) reflectively asserts
`Cost type = *float64`. Both tests exist.

### A9 — `time.Duration` vs `templates.Duration` — REFUTED

`BlockedRetryCooldown *time.Duration` (cli_adapter.go line 178). Import
block (line 7) imports `"time"` (the stdlib package), NOT
`"github.com/evanmschultz/tillsyn/internal/templates"` or any wrapper.
`time.Duration` is correct: `BindingResolved` is the post-resolution
flat shape (master PLAN L9), so the TOML-decoding `templates.Duration`
TextUnmarshaler wrapper is the wrong type at this layer (it would force
adapters to call `.Duration()` on a wrapper). `time.Duration` is the
correct concrete type for an already-resolved duration value.

### A10 — `json.RawMessage` import — REFUTED

`encoding/json` imported (line 4). `StreamEvent.ToolInput json.RawMessage`
(line 258), `StreamEvent.Raw json.RawMessage` (line 264), `ToolDenial.
ToolInput json.RawMessage` (line 279). All three usages are correct
type references — `json.RawMessage` is `[]byte` under the hood and
defers JSON parsing until adapter-private decoding runs. `TestStreamEvent
HasSevenFields` (line 259-274) and `TestToolDenialShape` (line 278-301)
reflectively pin the type. `TestToolDenialShape` line 297-300 uses the
underlying `[]byte` representation: `if have["ToolInput"].Kind() !=
reflect.Slice || have["ToolInput"].Elem().Kind() != reflect.Uint8` —
correct given json.RawMessage's underlying type.

### A11 — `*exec.Cmd` import + portability — REFUTED

`os/exec` imported (line 6). `BuildCommand` returns `(*exec.Cmd, error)`
(line 65), the pointer form, NOT the value form `exec.Cmd`. This is
correct Go idiom — `exec.Cmd` is a heavy struct; `*exec.Cmd` is what
`exec.Command()` returns and what callers expect. The interface signature
mirrors stdlib convention.

### A12 — 17 tests robustness sample — REFUTED

Sampled 5 tests for substance:

1. `TestIsValidCLIKindArbitraryStringRejected` (line 45-53) — table-
   driven with 5 inputs (`"bogus"`, `"Claude"`, `"CLAUDE"`, `" claude "`,
   `"claude "`). Each asserts `IsValidCLIKind` returns false. Catches
   case-folding bugs, leading/trailing whitespace bugs. Substantive.
2. `TestTerminalReportCostNilSignalsAbsence` (line 70-86) — three
   distinct assertions (zero-value nil; explicit `&zero` non-nil; deref
   matches 0.0). Substantive.
3. `TestBundlePathsHasNoClaudeSpecificFields` (line 111-133) — 14-name
   forbidden-list reflection scan. Catches future addition of any
   field name in the forbidden list. Substantive.
4. `TestBindingResolvedZeroValueIsAllAbsent` (line 221-253) — explicit
   per-field nil/empty assertions for 13 fields. Substantive.
5. `TestStreamEventHasSevenFields` (line 259-274) — pins NumField == 7
   AND field names AND field ORDER. Substantive (the order pinning is
   the strict form).

Boilerplate "struct exists" assertions absent. Each test asserts
something specific that fails on a real regression.

### A13 — Cross-droplet compile-fail risk — REFUTED with caveat

`BindingResolved.AgentName string` is a value type, not pointer-typed.
The doc-comment (line 105-106) says "The resolver always populates this
— it has no sensible 'absent' form." This is correct: every spawn has
an agent name; a missing AgentName is a programming error, not a
priority-cascade absence. F.7.17.3 (claudeAdapter) consumes
`br.AgentName` directly into `--agent <name>` argv flag. F.7.17.5
(dispatcher wiring) reads `br.AgentName` to look up the agent spec.
Neither needs a nil-vs-empty distinction.

`BindingResolved.CLIKind CLIKind` is a value type. F.7.17 L15 default-
to-claude lives at the resolver layer (droplet 8), so by the time
adapters see `BindingResolved`, the CLIKind is always non-empty. Same
analysis as AgentName — value type is correct here.

CAVEAT: F.7.17.5 (dispatcher wiring) selects an adapter via
`adapterMap[br.CLIKind]`. If the resolver fails to apply the L15
default before populating `BindingResolved`, the lookup returns the
zero-value adapter (likely nil), and F.7.17.5 must defend with a
nil-check. This is droplet 8's contract obligation, not droplet 2's.
Documented adequately in the CLIKind doc-comment (lines 22-23).

### A14 — Memory rules cross-check — REFUTED

- `feedback_no_migration_logic_pre_mvp.md`: pure types in dispatcher
  package, no migration code, no SQL. CLEAN.
- `feedback_subagents_short_contexts.md`: 307 lines of types + 17 tests.
  This is a single-surface task — pure types for one logical seam. The
  size is bounded by the spec, not over-bundled. Builder did not bundle
  unrelated work. CLEAN.
- `feedback_orchestrator_no_build.md`: orchestrator did NOT edit Go;
  builder did. Confirmed by the worklog's narrative + the diff stat.
  CLEAN.
- `feedback_section_0_required.md`: builder worklog has section
  structure (Droplet / Files edited / NOT shipped / Verification /
  Acceptance criteria) but no Section 0 5-pass certificate. This is
  expected — builder worklogs document what was done, not orchestrator-
  facing reasoning. The Section 0 directive is for Claude responses,
  not MD artifacts. CLEAN.

### A15 — Cross-droplet test pollution — REFUTED

The worklog (lines 129-143) attributes the second `mage ci` failure to
a parallel droplet writing `internal/templates/context_rules_test.go`
into the templates package. Verified independently:
- `git status --porcelain internal/app/dispatcher/` shows ONLY the two
  new files (`cli_adapter.go` + `cli_adapter_test.go`); no diff bleeds
  into other packages.
- The `templates/context_rules_test.go` file does exist on disk
  (parallel droplet artifact) but is NOT in this droplet's diff.
- F.7.17.2's diff makes ZERO references to the templates package or
  `context_rules_test.go`.

The cross-droplet collision was NOT caused by F.7.17.2. The builder
correctly identified the parallel-droplet root cause and called it
out in the worklog. CLEAN.

---

## Forward-collision NIT (A-extra) — `cli_adapter_test.go` filename

The plan body line 242 specifies the test file name as
`cli_adapter_types_test.go` for droplet 4c.F.7.17.2. Builder shipped
`cli_adapter_test.go` instead.

The plan body line 349 reserves `cli_adapter_test.go` for droplet
4c.F.7.17.4 (MockAdapter + contract conformance test).

Risk: when 4c.F.7.17.4 lands, the MockAdapter test code must EITHER
edit the existing 4c.F.7.17.2 file OR be put in a different file
(e.g., `cli_adapter_mock_test.go`). If a future builder follows the
plan body literally and creates `cli_adapter_test.go` again, Go will
treat both as one package compilation unit (same package, same dir,
same filename = compile error if the second is `Write`-style overwrite,
or merge if it's `Edit`).

Severity: NIT — there is no current-state defect because 4c.F.7.17.4
hasn't run yet. The orchestrator dispatching F.7.17.4 should pre-route
the file naming choice (likely to `cli_adapter_mock_test.go` or by
appending to the existing file). Recommend the orchestrator capture
this as a forward-routing note before spawning F.7.17.4.

---

## Plan-body deviations summary (NITs only)

The builder cited the spawn prompt as authority over plan body in three
places:

1. **`BindingResolved` Command/ArgsPrefix removal** — covered by REV-1
   explicitly. CLEAN.
2. **`BundlePaths` field count** — plan body L13 says 3 fields (`Root`,
   `StreamLog`, `Manifest`); builder shipped 6 (`Root`, `SystemPrompt
   Path`, `SystemAppendPath`, `StreamLogPath`, `ManifestPath`, `Context
   Dir`). NOT covered by any REV. NIT — plan-body/spawn-prompt drift.
3. **`StreamEvent` field count** — plan body L14 says 3 fields (`Type`,
   `IsTerminal`, `RawJSON`); builder shipped 7 (`Type`, `Subtype`,
   `IsTerminal`, `Text`, `ToolName`, `ToolInput`, `Raw`). NOT covered
   by any REV. NIT — plan-body/spawn-prompt drift.
4. **`StreamEvent.RawJSON` → `Raw`** — field renamed from `RawJSON
   []byte` to `Raw json.RawMessage`. The type change `[]byte` →
   `json.RawMessage` is sub-typing (both are `[]byte` underlying); the
   name change is unmotivated by any REV. Substantive but cosmetic.
   NIT.
5. **Test filename** — plan body says `cli_adapter_types_test.go`;
   builder shipped `cli_adapter_test.go`. NIT (forward-collision risk
   noted above).

None of these is a CONFIRMED counterexample — the builder followed the
spawn prompt, which the orchestrator authored after applying REV-1 +
REV-5 + REV-7 + REV-8. The drift between plan-body and spawn-prompt is
the orchestrator's record-keeping concern, not the builder's correctness
concern. Builder did the right thing under "spawn prompt is authoritative"
discipline.

---

## Final verdict — PASS-WITH-NITS

**Counterexamples found:** 0.
**Refuted attacks:** 12 (A1, A2, A3, A5, A6, A8, A9, A10, A11, A12,
A14, A15).
**NITs (informational, non-blocking):** 4 (A4 wording preference, A7
plan-body field-count drift, A13 caveat documented adequately, A-extra
forward-collision risk on test filename).

The droplet 4c.F.7.17.2 build is correct against the spawn prompt,
correctly applies REV-1 + REV-5, correctly excludes REV-7's removed
scope, and correctly follows REV-8's REVISIONS-first reading procedure
(documented in the worklog opening paragraph).

Recommend orchestrator address the forward-collision NIT (test
filename) when spawning F.7.17.4, and record the plan-body/spawn-prompt
field-count drifts (BundlePaths 3→6, StreamEvent 3→7) so downstream
planners know the spawn-prompt shapes are canonical.

No rework required. Sibling QA-Proof verdict + this PASS-WITH-NITS
together clear F.7.17.2 for marking complete.

---

## Hylla Feedback

N/A — Hylla calls were forbidden by the spawn prompt's hard constraints
("No Hylla calls. No code edits."). All evidence-gathering used
`Read` + `Bash rg`. No Hylla queries attempted, so no Hylla misses to
report.
