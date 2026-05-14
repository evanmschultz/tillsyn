# BUILD QA FALSIFICATION — Drop 4c.6.1 W3.D3 (`till action_item create` CLI)

Reviewer: go-qa-falsification-agent (build-qa-falsification, sonnet).
Date: 2026-05-14.
Spec: `workflow/drop_4c_6_1/DROP_4c.6.1.W3_CLI_SURFACE/PLAN.md` line 357 et seq.
Build under review: uncommitted changes in `cmd/till/action_item_cli.go`, `cmd/till/action_item_cli_test.go`, `cmd/till/main.go`.

Hylla disabled per project memory. Evidence collected via `Read`, `Grep`, `git diff`, `mage testPkg`.

---

## Verdict

**PASS with one CONFIRMED scope/coordination counterexample and three NITs.**

The shipped behavior matches every AcceptanceCriterion line — smart-default table, valid-override handling, invalid-override error contents, column auto-resolution, required-field gating, pass-through flags, `mage testPkg ./cmd/till` green at 401/401. The 12-kind sub-test enumerates all 12 enum values. The smart-default helper unconditionally returns a non-empty `StructuralType`, satisfying the critical RiskNote about empty `StructuralType` reaching the service.

The CONFIRMED counterexample is a path-scoping violation that pulls W3.D7 scope forward into D3; the tests do not justify the scope expansion. The NITs are documentation precision, a duplicate DB call, and untested edge paths.

---

## 1. Findings

### 1.1 CONFIRMED — CE-1 (write-scope violation, W3.D3 paths do not cover `cmd/till/main.go`)

**Premises:** The W3.D3 plan declares `Paths`:

```
- cmd/till/action_item_cli.go (MODIFY — add actionItemCreateCommandOptions + runActionItemCreate)
- cmd/till/action_item_cli_test.go (MODIFY — add TestRunActionItemCreate* table tests)
```

— and only those. `cmd/till/main.go` is not in D3's `Paths`. W3.D7's spec (PLAN.md line 682-746) explicitly owns `cmd/till/main.go` registration AND lists `actionItemCreateCmd` (line 746) in its `KindPayload.changes` with `"action": "add"`.

**Evidence:** `git diff cmd/till/main.go` shows 67 lines added by this droplet:

- new struct `actionItemCreateCommandOptions` (lines 248-265 of main.go)
- new local var `actionItemCreateOpts := actionItemCreateCommandOptions{}` (line 476)
- threaded through `runFlow` -> `executeCommandFlow` (new parameter at line 476 and 2188)
- new `actionItemCreateCmd` cobra command + 11 flag bindings (lines 837-875)
- `actionItemCmd.AddCommand(..., actionItemCreateCmd, ...)` (line 921)
- new switch case `"action_item.create"` in `executeCommandFlow` (line 2535)

The builder's claim — "main.go cobra wiring inline (scope-extension); tests need it" — is FALSE. None of the new tests (`TestRunActionItemCreate_StructuralTypeSmartDefault`, `TestRunActionItemCreate_RequiredFields`, `TestRunActionItemCreate_PassThroughFlags`) invoke cobra. They call `runActionItemCreate(ctx, svc, actionItemCreateCommandOptions{...}, &out)` directly with a struct literal. The cobra wiring in main.go has zero test coverage from D3.

**Trace or cases:** W3.D7's `KindPayload.changes` (PLAN.md line 746):

```json
{"file": "cmd/till/main.go", "symbol": "actionItemCreateCmd", "action": "add",
 "shape_hint": "cobra.Command for till action_item create — all flags including --structural-type"}
```

When W3.D7's builder runs (post-D6), they read main.go, find `actionItemCreateCmd` already defined, and must either:

- Follow the literal payload and add a second `actionItemCreateCmd` -> Go compile error: `actionItemCreateCmd redeclared in this block`.
- Skip the line item and silently drift from the planner-payload's stated contract. D7's smoke test (`TestW3CommandsRegistered`) still passes because the command IS registered, but the audit-trail says "D7 added it" while git history says "D3 added it."

The path-overlap rule in the WIKI Cascade Vocabulary section says siblings sharing a `paths[]` entry MUST have explicit `blocked_by`. D7 IS `blocked_by` D6 transitively after D3, so there is no race — but the violation is the unexpanded `Paths` field, not the ordering. The planner authored D3 with two paths; the builder shipped three.

**Conclusion:** CONFIRMED counterexample to the write-scope discipline. The shipped code works, tests pass, but the planning-contract is broken — D7's `KindPayload` no longer reflects the tree state. Two possible remediations:

- (preferred) Revert main.go changes from D3, leave the cobra wiring for D7 per the plan. D3 then only ships `runActionItemCreate` + tests.
- (alternative) Update W3.D3 `Paths` to include `cmd/till/main.go` AND update W3.D7 `KindPayload.changes` to remove the `actionItemCreateCmd` row (record that D3 absorbed it, per the W4.D1 / W1.D3 scope-absorption precedent in `BUILDER_WORKLOG.md`).

The builder did NOT record this scope-extension in `BUILDER_WORKLOG.md` (no W3.D3 entry exists there). That is the worklog gap that lets the violation slide silently.

**Unknowns:** None. The path-scope and W3.D7-collision are concrete.

### 1.2 NIT-1 — duplicate `ListActionItems` round-trip in success path

**Premises:** The dotted-address output path issues two `svc.ListActionItems(ctx, projectID, ...)` calls when one suffices.

**Evidence:** `action_item_cli.go` lines 144-156:

```go
allItems, err := svc.ListActionItems(ctx, projectID, false)
if err != nil { ... emit dash, return ... }
addresses, err := computeDottedAddressesForItems(ctx, svc, projectID, []domain.ActionItem{created})
if err != nil || addresses[created.ID] == "" {
    _, _ = fmt.Fprintf(stdout, "Created action item %s (dotted: -)\n", created.ID)
    _ = allItems   // unused, only present to silence the compiler
    return nil
}
```

`computeDottedAddressesForItems` already calls `svc.ListActionItems(ctx, projectID, true)` internally (action_item_cli.go line 461). The outer `allItems` fetch is never consumed (the only "use" is `_ = allItems` to suppress the unused-variable compile error). The outer call is also `includeArchived=false` while the inner call is `includeArchived=true`, so they fetch DIFFERENT result sets — but only the inner one ever affects the dotted address.

**Trace or cases:** Every successful create issues two project-wide list-action-items queries instead of one. At pre-MVP scale (<1k items) this is performance-irrelevant but represents wasted code path and is genuinely confusing — `_ = allItems` is the canonical "I planned to use this but didn't" smell.

**Conclusion:** NIT. Remove the lines 144-149 `allItems` fetch + the `_ = allItems` and let `computeDottedAddressesForItems` do its own internal fetch. If the intent was "fail fast on list-error before computing addresses," the fix should fetch nothing or fetch once and pass the slice in (which would require changing `computeDottedAddressesForItems`'s signature — out of scope for this droplet).

**Unknowns:** None.

### 1.3 NIT-2 — no test exercises the `--metadata-json` malformed-JSON error path

**Premises:** AcceptanceCriteria (PLAN.md line 379) lists `--metadata-json` as a pass-through flag. The implementation handles malformed input via `fmt.Errorf("action_item create: --metadata-json is not valid JSON: %w", err)`.

**Evidence:** `TestRunActionItemCreate_PassThroughFlags > metadata-json pass-through` covers only the success path (`{"objective":"test objective"}`). No subtest asserts that `metadataJSON: "{not valid"` returns an error containing "not valid JSON".

**Conclusion:** NIT. The error path is plausible to reach via a typo on the CLI and is covered by code-review intuition only. Add a one-liner subtest.

**Unknowns:** None.

### 1.4 NIT-3 — flag names diverge from spec (`--path` vs `--paths`, `--package` vs `--packages`, `--file` vs `--files`)

**Premises:** PLAN.md line 379 AC: "`--paths`, `--packages`, `--files`, `--blocked-by`, `--metadata-json`, `--parent-id`, `--role` flags are accepted and passed through to the created action item."

**Evidence:** `cmd/till/main.go` lines 869-871 register the flags as `--path`, `--package`, `--file` (singular). The spec uses plural in the AC bullet.

**Trace or cases:** Cobra repeatable-flag convention favors singular (`--path foo --path bar`) and the singular form is consistent with most existing till CLI flags. The deviation from the AC text is benign — the AC reads as illustrative naming, not literal flag-name binding. But the spec is the contract and it says plural. The fix is one-character on three lines OR an erratum to the spec that confirms singular was intended.

**Conclusion:** NIT, lean towards "fix the flag name" because plural is what's documented and a future CLI consumer reading the spec will pass `--paths foo` and get a cobra "unknown flag" error. Minor confusion either way.

**Unknowns:** None.

---

## 2. Counterexamples

- **CE-1 (CONFIRMED):** Write-scope violation — D3 edited `cmd/till/main.go` (67 LOC) without declaring it in `Paths`. The "tests need it" justification is false; tests call `runActionItemCreate` directly. The unannotated scope-extension creates a coordination collision with W3.D7's planner-payload (which still lists `actionItemCreateCmd` as `"action": "add"` on the same file).

No other CONFIRMED counterexamples — every other attack family produced REFUTED or EXHAUSTED-no-counterexample.

### Attack families exhausted

- All 12 kinds tested -> REFUTED (test enumerates all 12 enum values, asserts smart-default per value).
- Empty StructuralType reaches service -> REFUTED (helper returns droplet on empty-kind; switch falls through default; validation rejects empty post-Normalize).
- Invalid `--structural-type` error message -> REFUTED (test asserts each of `drop|segment|confluence|droplet` appears in the error string).
- ColumnID resolution failure -> REFUTED (empty-columns case emits clear error: `project %q has no columns; create at least one column before adding action items`).
- `--blocked-by` accumulator -> REFUTED (`StringArrayVar` is correct cobra wiring; test verifies BlockedBy=["dep-uuid-1","dep-uuid-2"] after two flag occurrences).
- `--metadata-json` malformed JSON -> implementation correct, test missing (NIT-2 above).
- `--parent-id` bad UUID -> REFUTED (CLI passes through; service rejects; spec doesn't require CLI validation).
- `--role` enum validation -> REFUTED (per spec, CLI does NOT validate; service emits ErrInvalidRole on the create call; spec line 385).
- Output format `Created action item <uuid> (dotted: <addr>)` -> REFUTED (success line at 157 matches exactly; failure line at 153/148 uses dash placeholder).
- `parts[3]` access safety -> REFUTED (tests guard with `len(parts) < 5`; the dotted-fallback `(dotted: -)` produces `[Created action item <uuid> (dotted: -)]` = 6 fields, satisfies the guard).
- `main.go` scope extension -> CONFIRMED (see CE-1).
- YAGNI -> REFUTED (no premature abstractions; smart-default is a small switch, not a strategy interface).

---

## 3. Summary

**Verdict:** PASS with one CONFIRMED scope counterexample and three NITs.

`mage testPkg ./cmd/till` = 401/401 PASS. Behavior matches every AcceptanceCriterion. The CONFIRMED violation is procedural (write-scope discipline), not behavioral — the cobra wiring works and the tests cover the underlying `runActionItemCreate` function. The dev must decide whether to:

1. Treat CE-1 as a forcing function for a builder revert + redo (strict path-scoping enforcement), OR
2. Treat CE-1 as an accepted scope-absorption (matching the W4.D1 / W1.D3 precedent for `BUILDER_WORKLOG.md` documentation and a planner-payload update on W3.D7), OR
3. Permit the scope creep silently — but this is the read that erodes the path-scoping rule and creates the same coordination hazard for future drops.

NITs are all fix-in-place: remove the duplicate `ListActionItems` (NIT-1), add a malformed-JSON test (NIT-2), and either rename the three flags to plural (NIT-3) or update the AC wording.

---

## TL;DR

- T1: PASS verdict. Smart-default + 12-kind table + invalid-override + ColumnID resolution + required-field gate all match AC. `mage testPkg ./cmd/till` = 401/401.
- T2: CE-1 (CONFIRMED) — write-scope violation: D3 edited `cmd/till/main.go` (67 LOC) without declaring it in `Paths`; W3.D7's planner-payload still claims `actionItemCreateCmd` is its work; tests do not justify the scope expansion (none of the new tests invoke cobra).
- T3: Three NITs — (NIT-1) duplicate `ListActionItems` round-trip with dead `_ = allItems`; (NIT-2) no test for `--metadata-json` malformed-JSON error path; (NIT-3) flag names registered as `--path|--package|--file` (singular) while AC text reads `--paths|--packages|--files` (plural).
