# BUILD-QA-PROOF — Drop 4c.6.1 W3.D3 `till action_item create` CLI

Verdict: **PASS** (with 4 NITs).

## Acceptance Criteria (per `workflow/drop_4c_6_1/DROP_4c.6.1.W3_CLI_SURFACE/PLAN.md` lines 371-381)

### AC1 — `till action_item create --kind plan ...` creates `StructuralType=segment` (smart-default)

Quote (spec): "`till action_item create --project-id <id> --kind plan --title \"T\" --description \"D\"` creates the action item with `StructuralType=segment` (smart-default) and returns UUID + dotted address on stdout."

Evidence:
- `cmd/till/action_item_cli.go:34-41` — `structuralTypeSmartDefault` returns `domain.StructuralTypeSegment` for `domain.KindPlan` via the switch on `strings.TrimSpace(strings.ToLower(kind))`.
- `cmd/till/action_item_cli.go:80-90` — `runActionItemCreate` calls `structuralTypeSmartDefault(opts.kind)` when `opts.structuralType` is empty.
- Test `cmd/till/action_item_cli_test.go:750-779` (table sub-test `plan`) asserts `structuralTypeSmartDefault("plan") == StructuralTypeSegment`.
- Test `cmd/till/action_item_cli_test.go:821-837` (sub-test `smart-default plan creates segment without explicit flag`) end-to-end creates a plan and confirms `Created action item ... (dotted: ...)` output.
- Output line at `action_item_cli.go:148`, `:153`, `:157` emits `Created action item <id> (dotted: <addr>)`.

Verdict: **PASS**.

### AC2 — `till action_item create --kind build ...` creates `StructuralType=droplet`

Evidence:
- `action_item_cli.go:38` — default branch returns `domain.StructuralTypeDroplet`.
- Test table sub-test `build` at `action_item_cli_test.go:758`.
- Test `smart-default build creates droplet without explicit flag` at `action_item_cli_test.go:839-855`.

Verdict: **PASS**.

### AC3 — `till action_item create --kind refinement ...` creates `StructuralType=segment`

Evidence:
- `action_item_cli.go:36` — switch case `string(domain.KindRefinement)` returns `StructuralTypeSegment`.
- Test table sub-test `refinement` at `action_item_cli_test.go:757` asserts `StructuralTypeSegment`.

Verdict: **PASS**.

### AC4 — All other 9 kinds default to `StructuralType=droplet`

Quote (spec): "research, plan-qa-proof, plan-qa-falsification, build-qa-proof, build-qa-falsification, closeout, commit, discussion, human-verify".

Evidence:
- `action_item_cli.go:34-41` — switch only special-cases `KindPlan` + `KindRefinement`; every other kind falls through to `default: return domain.StructuralTypeDroplet`.
- Test `cmd/till/action_item_cli_test.go:752-779` iterates ALL 12 kinds in the table. Verified entries: `research`, `plan-qa-proof`, `plan-qa-falsification`, `build-qa-proof`, `build-qa-falsification`, `closeout`, `commit`, `discussion`, `human-verify`, plus `build` — all assert `StructuralTypeDroplet`.

Verdict: **PASS** (all 9 other-kinds covered + `build`).

### AC5 — `--structural-type confluence` uses the explicit override (valid)

Evidence:
- `action_item_cli.go:83-90` — when `rawST` non-empty, normalizes via `domain.NormalizeStructuralType` and gates with `domain.IsValidStructuralType`; on pass, uses the normalized value.
- `domain.NormalizeStructuralType` (`internal/domain/structural_type.go:64-70`) lowercases + trims; `IsValidStructuralType` (`:56-58`) checks membership against `validStructuralTypes` (drop|segment|confluence|droplet).
- Test `explicit valid override accepted` at `action_item_cli_test.go:781-798` exercises `structuralType: "confluence"` end-to-end against a real `*app.Service` + SQLite — verifies success path.

Verdict: **PASS**.

### AC6 — `--structural-type invalid` fails with a clear error listing valid values

Evidence:
- `action_item_cli.go:85-88` — invalid value returns `fmt.Errorf("action_item create: --structural-type %q is invalid (valid values: %s)", rawST, strings.Join(validStructuralTypeValues, "|"))`.
- `validStructuralTypeValues` at `action_item_cli.go:23` is `[]string{"drop", "segment", "confluence", "droplet"}` (declaration-order match with the enum).
- Test `explicit invalid structural-type rejects with valid list` at `action_item_cli_test.go:800-819` passes `"invalid-value"` and asserts the error contains each of `"drop"`, `"segment"`, `"confluence"`, `"droplet"`.

Verdict: **PASS**.

### AC7 — Created action item is placed on the project's first column (ColumnID auto-resolved; no `--column-id` flag)

Evidence:
- `action_item_cli.go:96-103` — calls `svc.ListColumns(ctx, projectID, false)`, errors when zero columns, otherwise picks `columns[0].ID`.
- `Service.ListColumns` at `internal/app/service.go:2118-2127` sorts by `Position` ascending before returning, so `[0]` is the first column.
- No `--column-id` flag registered in `cmd/till/main.go` for `actionItemCreateCmd` (lines 864-875) — confirmed absent.
- Test `column auto-resolved to first column sorted by position` at `action_item_cli_test.go:1042-1069` confirms the created item carries a non-empty `ColumnID`.

Verdict: **PASS**.

### AC8 — Pass-through flags accepted (paths, packages, files, blocked-by, metadata-json, parent-id, role)

Evidence:
- `action_item_cli.go:122-136` — `CreateActionItemInput` wires `ProjectID`, `ParentID`, `Kind`, `Scope`, `Role`, `StructuralType`, `ColumnID`, `Title`, `Description`, `Paths`, `Packages`, `Files`, `Metadata`.
- `action_item_cli.go:115-117` — `--blocked-by` accumulator overwrites `Metadata.BlockedBy` (explicit-flag-wins; documented at lines 107-108).
- `action_item_cli.go:110-114` — `--metadata-json` parses into `domain.ActionItemMetadata`.
- `main.go:864-875` — all 7 flags registered: `--project-id`, `--parent-id`, `--kind`, `--title`, `--description`, `--path` (repeatable StringArray), `--package` (repeatable), `--file` (repeatable), `--blocked-by` (repeatable), `--metadata-json`, `--structural-type`, `--role`.
- Tests in `TestRunActionItemCreate_PassThroughFlags`:
  - `blocked-by sets Metadata.BlockedBy without post-create update` (`action_item_cli_test.go:912-949`) — asserts `item.Metadata.BlockedBy` == `[dep-uuid-1, dep-uuid-2]`.
  - `paths and packages pass through` (`:951-982`) — asserts `item.Paths` + `item.Packages`.
  - `role pass-through` (`:984-1011`) — asserts `item.Role == domain.Role("builder")`.
  - `metadata-json pass-through` (`:1013-1040`) — asserts `item.Metadata.Objective == "test objective"`.

Verdict: **PASS** (with NIT N1 on flag-name singular vs spec-text plural).

### AC9 — Missing `--project-id` / `--kind` / `--title` / `--description` fails with a clear required-field error

Evidence:
- `action_item_cli.go:60-71` — four sequential `strings.TrimSpace(...) == ""` checks return `fmt.Errorf("action_item create: --<field> is required")` for each missing field BEFORE the `svc == nil` check.
- Test `TestRunActionItemCreate_RequiredFields` at `action_item_cli_test.go:860-904` iterates all four missing cases; each asserts the error mentions the flag name (`--project-id`, `--kind`, `--title`, `--description`). Tests pass `svc=nil` to prove the gate fires before any service call.

Verdict: **PASS**.

### AC10 — `mage test-pkg ./cmd/till/...` passes

Evidence (this review):
- `mage test-pkg ./cmd/till/...` → 401 tests passed across 1 package; 0 failed.
- `mage ci` → all packages above 70% coverage; cmd/till at 77.5%; build target succeeded.

Verdict: **PASS**.

## RiskNotes compliance

- `domain.NewActionItem` rejects empty StructuralType. `action_item_cli.go:80-90` ensures StructuralType is set before the `svc.CreateActionItem` call (smart-default OR validated override). Confirmed at `internal/domain/action_item.go:354-359` — empty StructuralType returns `ErrInvalidStructuralType`. **Held.**
- `--role` optional: `action_item_cli.go:120` constructs `domain.Role(strings.TrimSpace(opts.role))`; empty string is permitted (service-side check at `internal/domain/action_item.go:350-353` short-circuits on empty Role). **Held.**
- `--blocked-by` repeated flag wired via `CreateActionItemInput.Metadata.BlockedBy []string`: `main.go:872` uses `StringArrayVar` so multiple `--blocked-by` flags accumulate; `action_item_cli.go:115-117` assigns into `Metadata.BlockedBy` before the service call. No post-create UpdateActionItem. **Held.**

## ContextBlocks compliance

- **FF4 smart-default table**: switch covers `KindPlan`+`KindRefinement` → segment, default → droplet. Verified at `action_item_cli.go:34-41`. **Held.**
- **ColumnID auto-resolution**: `svc.ListColumns(ctx, projectID, false)` + `columns[0].ID` pattern matches the canonical reference at `auto_generate_steward.go:123-158`. Empty-columns case explicitly errored at `action_item_cli.go:100-102`. **Held.**
- **Valid override values**: `validStructuralTypeValues = []string{"drop", "segment", "confluence", "droplet"}` at `action_item_cli.go:23` matches `internal/domain/structural_type.go:18-31`. **Held.**
- **CreateActionItemInput fields wired**: `ProjectID, ParentID, Kind, Role, StructuralType, Title, Description, Paths, Packages, Files, ColumnID, Metadata, Scope` — verified at `action_item_cli.go:122-136`. `Priority`, `Labels`, `DueAt`, `StartCommit`, `EndCommit` correctly omitted per spec. **Held.**
- **Output format**: `Created action item <id> (dotted: <addr>)\n` at `action_item_cli.go:148/153/157`. **Held.**

## Findings

### Axis: acceptance-criteria-coverage

All 10 AcceptanceCriteria PASS with backing evidence.

### Axis: spec-conformance

- N1 [Axis: spec-conformance] [severity: low] Spec AcceptanceCriteria #8 text writes `--paths`, `--packages`, `--files` (plural). Implementation registers `--path`, `--package`, `--file` (singular) per cobra `StringArrayVar` convention → `main.go:869-871` → fix_hint: either accept the cobra-convention singulars as in-spirit compliance (no code change) or add plural aliases via `cobra.Flags().StringArrayVar(&opts.paths, "paths", nil, ...)` for spec-text symmetry. Tests use `paths` etc. directly on the struct, so no test impact.

- N2 [Axis: spec-conformance] [severity: low] `--blocked-by` overwrites rather than merges with `--metadata-json`'s `blocked_by` field. The inline doc-comment at `action_item_cli.go:107-108` calls this out ("explicit flag wins, unusual but correct") but spec ContextBlocks line 416-417 says "merges them into the constructed domain.ActionItemMetadata.BlockedBy" — "merges" is ambiguous between concatenate and overwrite. → fix_hint: document the precedence rule one level higher (e.g. in the cobra `Long:` text) so operators know `--blocked-by` wins; alternatively concat `append(metadata.BlockedBy, opts.blockedBy...)` and de-duplicate.

### Axis: completion-checklist-audit

- N3 [Axis: completion-checklist-audit] [severity: low] `allItems` at `action_item_cli.go:144` is fetched via `svc.ListActionItems(ctx, projectID, false)` then never used in the success path — `computeDottedAddressesForItems` does its own `ListActionItems(includeArchived=true)` round-trip internally (`action_item_cli.go:461`). The outer fetch is a wasted repo call; only its error is consumed as the "give up on dotted address" signal. → fix_hint: drop the outer `ListActionItems` call entirely and rely on `computeDottedAddressesForItems`'s err-return to drive the fallback "dotted: -" path. Saves one round-trip per `create` invocation.

### Axis: decision-log-review

- N4 [Axis: decision-log-review] [severity: low] The cobra command's `Long:` text describes `--structural-type` valid values as `drop|segment|confluence|droplet` but omits the FF4 smart-default table from the help. Operators running `till action_item create --help` will not see which kinds default to segment vs droplet without reading source. → fix_hint: add a one-line "Defaults: plan and refinement → segment; all other kinds → droplet." to the `Long:` text at `main.go:842-851`.

## Shipped-but-not-wired check

All four steps present:
1. **Schema/struct** — `actionItemCreateCommandOptions` at `main.go:248-265` (per diff).
2. **Resolver/run-function** — `runActionItemCreate` at `action_item_cli.go:56-159`; switch case `action_item.create` in `executeCommandFlow` at `main.go:2536-2540` (per diff).
3. **Consumer/cobra wiring** — `actionItemCreateCmd` at `main.go:835-875` + registered as subcommand at `main.go:919` (per diff: `actionItemCreateCmd,` line added to the subcommand list).
4. **Integration tests** — 4 `TestRunActionItemCreate_*` test groups in `action_item_cli_test.go`, exercising smart-default (all 12 kinds + override valid + override invalid), required-field gate, pass-through flags, ColumnID auto-resolution, nil-service rejection.

Verdict: **shipped + wired**.

## Summary

PASS / FAIL: **PASS** with 4 low-severity NITs (N1 flag-naming, N2 blocked-by merge-vs-overwrite doc clarity, N3 wasted ListActionItems round-trip, N4 help-text smart-default table).
