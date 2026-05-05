# Drop 4c — F.7.17.8 BindingResolved Resolver — QA Falsification Round 1

**Reviewer:** go-qa-falsification-agent (read-only, adversarial pass).
**Target:** droplet `4c.F.7.17.8` — `BindingResolved` priority cascade resolver.
**Files reviewed:**

- `internal/app/dispatcher/binding_resolved.go` (NEW, 245 lines).
- `internal/app/dispatcher/binding_resolved_test.go` (NEW, 12 tests).
- `workflow/drop_4c/4c_F7_17_8_BUILDER_WORKLOG.md`.

**Cross-evidence consulted:**

- `internal/app/dispatcher/cli_adapter.go` (BindingResolved struct from F.7.17.2).
- `internal/templates/schema.go` (templates.AgentBinding shape).
- `workflow/drop_4c/F7_17_CLI_ADAPTER_PLAN.md` lines 525-565 (droplet 8 spec) + lines 683-735 (REVISIONS POST-AUTHORING REV-1 through REV-8).

# QA Falsification Review

## 1. Findings

### 1.1 A1 — Priority cascade walk direction — **REFUTED**

`binding_resolved.go:140-150` constructs the cascade by walking `overrides`
slice from index 0 forward. The doc-comment (lines 27-38) is explicit:
"Priority cascade (highest → lowest)" and "ResolveBinding merges them in
order, picking the first non-nil value." First-element-wins == highest-first.

Caller convention: `ResolveBinding(raw, cliOverrides, mcpOverrides, tuiOverrides)`
yields CLI > MCP > TUI > raw, matching master PLAN L9 / L16.

`TestResolveBindingMultiLayerPriority` (lines 125-138) asserts
`ResolveBinding(raw, highest, mid, low)` returns `highest`'s Model. Confirms
implementation matches documented direction.

No counterexample.

### 1.2 A2 — `BindingOverrides` ↔ `BindingResolved` type alignment — **REFUTED**

Field-by-field type comparison:

| Field                  | BindingOverrides   | BindingResolved    | Match |
| ---------------------- | ------------------ | ------------------ | ----- |
| `Model`                | `*string`          | `*string`          | yes   |
| `Effort`               | `*string`          | `*string`          | yes   |
| `MaxTries`             | `*int`             | `*int`             | yes   |
| `MaxBudgetUSD`         | `*float64`         | `*float64`         | yes   |
| `MaxTurns`             | `*int`             | `*int`             | yes   |
| `AutoPush`             | `*bool`            | `*bool`            | yes   |
| `BlockedRetries`       | `*int`             | `*int`             | yes   |
| `BlockedRetryCooldown` | `*time.Duration`   | `*time.Duration`   | yes   |

Cross-checked at `cli_adapter.go:102-179` and `binding_resolved.go:44-79`.
All eight overridable scalars are pointer-typed on both sides. No type
divergence. `time.Duration` conversion from `templates.Duration` happens
once at `binding_resolved.go:150` before feeding the resolver helper, so
the canonical stdlib type lands on the resolved struct.

No counterexample.

### 1.3 A3 — Defensive-copy correctness — **REFUTED**

`cloneStringSlice` (lines 237-244):

```go
func cloneStringSlice(in []string) []string {
    if in == nil {
        return nil
    }
    out := make([]string, len(in))
    copy(out, in)
    return out
}
```

`make` allocates a fresh backing array; `copy` copies elements. The output
slice does NOT share the input's underlying array. Caller mutating the
returned slice cannot leak into the rawBinding source.

`TestResolveBindingPureFunctionDoesNotMutateRaw` (lines 303-316) writes
`gotResolved.Tools[0] = "MUTATED"` and asserts `raw.Tools[0] == "Read"` —
confirms the clone is genuinely independent.

For pointer fields, `resolveStringPtr` / `resolveIntPtr` / etc. (lines
160-232) each take `vCopy := *v; return &vCopy` — the returned pointer
is to a fresh copy, NOT the override's internal pointer.
`TestResolveBindingPointerOverridesPreservedAsCopy` (lines 266-281) mutates
the source `model` variable AFTER `ResolveBinding` returns and asserts the
resolved struct holds the pre-mutation value.

No counterexample.

### 1.4 A4 — `CommitAgent` empty-string promotion to nil — **REFUTED**

Type cross-check:

- `templates.AgentBinding.CommitAgent string` (`schema.go:361`).
- `BindingResolved.CommitAgent *string` (`cli_adapter.go:168`).

Builder's claim ("CommitAgent empty string promotes to nil pointer
(preserves the absent-vs-explicit distinction)") is consistent because
the SOURCE is `string` and the TARGET is `*string`. The promotion
preserves a meaningful distinction at the resolved-struct boundary.

Code at `binding_resolved.go:135-138`:

```go
if rawBinding.CommitAgent != "" {
    v := rawBinding.CommitAgent
    resolved.CommitAgent = &v
}
```

Non-empty → pointer to copy. Empty string → resolved.CommitAgent stays
nil (struct zero value).

`TestResolveBindingCommitAgentEmptyToNil` (lines 249-260) and
`TestResolveBindingNoOverrides` (lines 84-86) jointly cover both branches.

Spawn-prompt sub-claim "non-pointer field on BindingResolved per F.7.17.2
builder's report" is INCORRECT — `cli_adapter.go:168` shows `CommitAgent
*string`. The builder's design honors the actual F.7.17.2 type. No bug.

No counterexample.

### 1.5 A5 — `time.Duration` zero-value semantics — **REFUTED with NIT**

`*time.Duration` correctly distinguishes nil (absent) from
`time.Duration(0)` (explicit-zero "retry immediately"). Doc-comment lines
75-78 documents the distinction.

`TestResolveBindingNoOverrides` (line 81) asserts `30*time.Second`
round-trip. `TestResolveBindingDurationOverride` (lines 285-296) asserts
override-supplied 5-minute value lands. Two of the three semantically
distinct cases are covered.

**NIT N1:** No test fixture exercises `rawBinding.BlockedRetryCooldown = 0`
+ no override → resolved `*time.Duration` is non-nil pointer to
`time.Duration(0)`. The implementation's fall-through (`v := rawValue;
return &v` in `resolveDurationPtr`) handles this correctly because
`time.Duration(0)` is still copied to a pointer. But the explicit-zero
distinction the doc-comment promotes is not test-pinned. Minor coverage
gap; no functional bug.

No counterexample.

### 1.6 A6 — Multi-layer priority correctness — **REFUTED**

`TestResolveBindingMultiLayerPriority` (lines 125-138):

```
raw.Model = "opus"
highest = {Model: "haiku"}
mid     = {Model: "sonnet"}
low     = {Model: "opus-mid-tier"}
ResolveBinding(raw, highest, mid, low) → resolved.Model = "haiku"
```

`resolveStringPtr`'s for-loop hits `highest` first, returns immediately.
Implementation matches assertion.

No counterexample.

### 1.7 A7 — Mixed-field overrides correctness — **REFUTED**

`TestResolveBindingMixedFieldOverrides` (lines 143-165):

```
highest = {Model: "haiku"}            // Model only
low     = {MaxTurns: 99}              // MaxTurns only
ResolveBinding(raw, highest, low) → resolved.Model = "haiku", MaxTurns = 99
```

The resolver does NOT short-circuit on first-layer-encountered. Each
pointer field has its own dedicated `resolveNNNPtr` call (lines 143-150)
with its own independent `for _, o := range overrides` loop. Field
walks are independent — Model's walk hits `highest`'s non-nil Model and
returns; MaxTurns' walk skips `highest` (its MaxTurns is nil), hits
`low`'s non-nil MaxTurns, returns.

No short-circuit risk. No counterexample.

### 1.8 A8 — Empty overrides slice — **REFUTED**

`TestResolveBindingEmptyOverridesSlice` (lines 191-207) calls
`ResolveBinding(raw, emptyOverrides...)` with `emptyOverrides :=
[]*BindingOverrides{}` (length 0).

Each `resolveNNNPtr` helper's `for _, o := range overrides` loop runs
zero iterations on an empty slice; falls through to the rawBinding
fallback path (`v := rawValue; return &v`). Behavior is identical to
calling `ResolveBinding(raw)` with no varargs.

The variadic-empty case is asserted explicitly with a different call
shape than scenario 1.

No counterexample.

### 1.9 A9 — Nil-vs-empty slice preservation — **REFUTED with NIT**

`cloneStringSlice` (lines 237-244) preserves the distinction:

- `nil` input → `nil` output (early return).
- `[]string{}` (empty non-nil) input → `make([]string, 0)` + `copy` →
  empty non-nil slice output.
- `[]string{"a"}` input → `make([]string, 1)` + `copy` → non-nil one-elem
  slice.

The implementation is correct.

**NIT N2:** No test fixture exercises `rawBinding.Tools = nil` →
`resolved.Tools == nil` OR `rawBinding.Tools = []string{}` →
`resolved.Tools` len 0 but non-nil. The fixture (`rawBindingFixture`)
sets non-empty Tools so the nil-preservation branch is exercised in
production but not test-asserted. Minor coverage gap; the doc-comment
on `cloneStringSlice` documents the nil-preservation invariant but no
test pins it.

No counterexample.

### 1.10 A10 — `AgentName` always from raw — **REFUTED**

`BindingOverrides` struct (lines 44-79) has NO `AgentName` field.
Resolver line 118 unconditionally sets
`AgentName: rawBinding.AgentName`. No override path.

`TestResolveBindingNoOverrides` (line 54-56) asserts the value passes
through; no test exercises an override pathway because the surface does
not exist.

No counterexample.

### 1.11 A11 — Tools / Env / CLIKind override absence — **REFUTED**

`BindingOverrides` struct fields enumerated (lines 47-79):

1. Model
2. Effort
3. MaxTries
4. MaxBudgetUSD
5. MaxTurns
6. AutoPush
7. BlockedRetries
8. BlockedRetryCooldown

NO Tools, ToolsAllowed, ToolsDisallowed, Env, AgentName, CLIKind,
CommitAgent fields. Doc-comment lines 40-43 explicitly name the
"NOT yet plumbed for override" set, matching master PLAN's "override
plumbing not yet wired" stance.

When the dispatcher grows CLI/MCP/TUI surfaces for those fields, the
struct extends naturally. No premature abstraction.

No counterexample.

### 1.12 A12 — No-commit per REV-13 — **REFUTED**

Worklog § "Acceptance criteria checklist" line 144:
`[x] **NO commit by builder** — orchestrator drives commits after the
QA pair returns green.`

`git status` confirms three new files untracked
(`binding_resolved.go`, `binding_resolved_test.go`,
`4c_F7_17_8_BUILDER_WORKLOG.md`).

(Note: REV-13 is not present in this plan's REVISIONS section — the
section runs REV-1 through REV-8. The "REV-13" reference in the spawn
prompt likely points to a global / cross-plan REVISION; either way the
no-commit invariant is honored at file-system reality.)

No counterexample.

### 1.13 A13 — Memory-rule conflicts — **REFUTED**

`feedback_subagents_short_contexts.md` — the droplet is single-surface
(one new package file + one new test file in `internal/app/dispatcher/`,
no cross-package edits, ~245 + ~316 LOC of new content). Single-task
scope, no parallelization need. No conflict.

`feedback_no_migration_logic_pre_mvp.md` — N/A (no schema work).
`feedback_orchestrator_no_build.md` — N/A (builder is a subagent).
`feedback_subagents_background_default.md` — orchestrator-side rule, not
the builder's concern.

No counterexample.

### 1.14 Additional self-attacks (out-of-list)

**A14 — REV-1 compliance.** Worklog explicitly claims `BindingResolved`
does NOT carry `Command` or `ArgsPrefix`. Verified at
`cli_adapter.go:102-179` — neither field exists. The droplet only
populates / cascades existing fields per F.7.17.2. **REFUTED.**

**A15 — Pure-function claim.** `ResolveBinding` performs no I/O, no
goroutines, no global-state reads. The single non-trivial conversion
(`time.Duration(rawBinding.BlockedRetryCooldown)` at line 150) is a
free type cast, not a side effect. Doc-comment lines 84-85 promises
"Pure function: no I/O, no global state, no side effects" — code matches.
**REFUTED.**

**A16 — Variadic-nil safety.** `ResolveBinding(raw, nil)` passes a
single `nil *BindingOverrides` entry. `TestResolveBindingNilLayerSkipped`
(lines 170-186) asserts `[nil, override, nil]` does not panic and
override's values land. Each `resolveNNNPtr` helper checks `if o == nil
{ continue }` (lines 161-163, 176-178, 191-193, 206-208, 221-223) — nil
entries safely skipped. **REFUTED.**

**A17 — `ResolveBinding(raw)` with NO varargs at all.** Equivalent to
empty slice case (Go variadic semantics: `overrides == nil` slice with
`len(overrides) == 0`). The for-loops iterate zero times. Test
`TestResolveBindingNoOverrides` (line 52) calls this shape directly.
**REFUTED.**

**A18 — Filename deviation from plan.** Master PLAN line 534 specifies
`binding_resolver.go` / `binding_resolver_test.go`. Builder shipped
`binding_resolved.go` / `binding_resolved_test.go`. The shipped names
align with the resolved-binding-noun pattern (`BindingResolved` is the
output type) but differ from the verb-form the plan suggests. NIT only
— filename is not a load-bearing acceptance criterion; the function
name (`ResolveBinding`), exported types (`BindingOverrides`,
`BindingResolved`), and package (`dispatcher`) all match the plan.
**NIT N3 — cosmetic.**

**A19 — Pointer copy on raw fall-through path.** When no override
sets a field, the resolver falls through to `v := rawValue; return &v`
(e.g. line 170-171 for strings). This intentionally creates a fresh
pointer on every call — distinct address but same value. Adapters
mutating through the resolved pointer cannot leak into the rawBinding
struct (rawBinding fields are scalars passed by value into
`resolveNNNPtr`, so even the pointer's pointee is a stack-copy of the
caller's scalar). No aliasing risk. **REFUTED.**

**A20 — `BlockedRetryCooldown` raw-zero round-trip.** When
`rawBinding.BlockedRetryCooldown == 0` (templates.Duration(0)) and no
override, `resolveDurationPtr` returns a `*time.Duration` pointing to
`time.Duration(0)`. This preserves the explicit-zero ("retry
immediately") semantic — adapters distinguish via pointer-non-nil + value
== 0. Behavior is correct; coverage gap noted in NIT N1.

## 2. Counterexamples

None. All thirteen attack vectors plus seven additional self-attacks
returned REFUTED or REFUTED-with-NIT. No CONFIRMED counterexamples
constructed.

## 3. Summary

**Verdict: PASS WITH NITS.**

The priority-cascade resolver implements master PLAN L9 / L16 correctly:

- Highest-priority-first walk via slice-index ordering (caller convention
  documented, test-asserted).
- Pointer-typed cascade preserves the absent-vs-explicit-zero distinction
  at every overridable field.
- Defensive copy at every output boundary — slice clone + scalar copy on
  pointer fall-through + dereference-and-copy on override-supplied
  pointers. Caller mutation cannot leak.
- Pure function — no I/O, no globals, no side effects.
- REV-1 compliance — no Command / ArgsPrefix references.
- No commit by builder — confirmed by file-system state.

Three minor coverage gaps (NIT N1, NIT N2, NIT N3) — none functional,
none blocking. Recommend optional follow-on test fixtures in a future
hardening pass:

- **NIT N1:** assert `rawBinding.BlockedRetryCooldown = 0` round-trips
  to `*time.Duration` non-nil pointer at value zero (explicit-zero
  preservation).
- **NIT N2:** assert `rawBinding.Tools = nil` → `resolved.Tools == nil`
  and `rawBinding.Tools = []string{}` → `resolved.Tools` len-0 non-nil
  (nil-vs-empty preservation pinned).
- **NIT N3:** filename is `binding_resolved.go` rather than master PLAN's
  `binding_resolver.go`. Cosmetic; orchestrator may rename or accept as-is.

No rework required. Orchestrator can promote the droplet to QA-PASS and
proceed to commit + downstream sequencing.

## Hylla Feedback

N/A — action item touched non-Go-code-search work. The review used
`Read` on Go source + plan markdown, no Hylla calls were attempted (the
droplet's surface is two new files + a worklog markdown — Hylla would
not yet have indexed the new files anyway, and the existing
`BindingResolved` struct was already discoverable via direct `Read` on
the file path the worklog cited).

## TL;DR

- T1. Thirteen spawn-prompt attack vectors plus seven self-attacks all
  REFUTED — no CONFIRMED counterexamples constructed.
- T2. None — no counterexamples produced.
- T3. PASS WITH NITS — three minor test-coverage gaps (NIT N1: zero-
  duration round-trip not pinned; NIT N2: nil-vs-empty slice preservation
  not pinned; NIT N3: filename `binding_resolved.go` rather than plan's
  `binding_resolver.go`). All cosmetic; no rework required.
