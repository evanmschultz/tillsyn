# Drop 4c — F.7.10 Builder Worklog

**Droplet:** F.7.10 — Drop hardcoded `hylla_artifact_ref` from `spawn.go` prompt body
**Builder model:** opus
**Date:** 2026-05-04
**Mode:** Filesystem-MD (no Tillsyn action items)

## Goal

Remove the hardcoded `hylla_artifact_ref:` line from `internal/app/dispatcher/spawn.go`'s `assemblePrompt` function. Hylla is dev-local, NOT part of Tillsyn's shipped cascade. Per locked architectural decision **L21** (master `PLAN.md` line 57) and F7-CORE plan decision **#9** (line 24), only the prompt-body emission is removed; `domain.Project.HyllaArtifactRef` and `project.metadata.hylla_artifact_ref` are preserved for adopter-local templates that opt into Hylla MCP.

## Files Changed

### `internal/app/dispatcher/spawn.go`

Three edits, all inside the dispatcher package, all in scope:

1. **`SpawnDescriptor.Prompt` doc-comment (lines 62-66)** — removed `hylla_artifact_ref` from the list of "structural fields the promptAssembler guarantees." The doc-comment was advertising a contract the function no longer provides; not updating would have been a documentation lie.

2. **`assemblePrompt` doc-comment (lines 192-206)** — removed `hylla_artifact_ref` from the structural-fields list and added a multi-line note explaining the F.7.10 deletion: Hylla is dev-local; adopters who opt into Hylla MCP can still surface the field via their own `system_prompt_template_path` (cross-reference to F.7.2); the data field + project metadata stay.

3. **`assemblePrompt` body (formerly lines 211-213)** — deleted the three `b.WriteString(...)` calls that emitted:

   ```
   hylla_artifact_ref: <value>\n
   ```

   The block sat between `project_dir:` and `kind:`. Removing it leaves the surrounding strings intact, no stray newlines.

### `internal/app/dispatcher/spawn_test.go`

Two edits:

1. **`TestBuildSpawnCommandAssemblesArgvForGoBuilder` assertion block (lines ~121-145)** — flipped per acceptance criterion. The `wantTokens` slice no longer contains `"hylla_artifact_ref: " + project.HyllaArtifactRef`. A new explicit negative-substring check follows the positive-tokens loop:

   ```go
   if strings.Contains(descriptor.Prompt, "hylla_artifact_ref") {
       t.Errorf(...)
   }
   ```

   The negative-substring check is unconditional — even if a future builder reintroduces a synonym (`hylla_ref:` etc.) the original `hylla_artifact_ref` substring stays banned.

2. **`fixtureProject` doc-comment (lines 39-46)** — updated to reflect post-F.7.10 reality. The pre-edit comment claimed `BuildSpawnCommand` reads `HyllaArtifactRef` + `Language`; post-edit it does not. Comment now records that `HyllaArtifactRef` is set in the fixture specifically to exercise the negative-substring assertion (the value MUST NOT leak into the prompt body even when populated). The struct literal itself is unchanged — keeping the field set hardens the negative test.

## Files NOT changed (verified untouched)

- `internal/domain/project.go` — `HyllaArtifactRef` field declarations at lines 25, 158, etc. all preserved (verified via `rg "HyllaArtifactRef" internal/`).
- `internal/adapters/storage/sqlite/repo.go` — `hylla_artifact_ref` SQLite column DDL + INSERT/UPDATE/SELECT statements untouched.
- `internal/adapters/server/mcpapi/extended_tools.go` — MCP tool surface for `hylla_artifact_ref` argument untouched.
- `internal/app/snapshot.go` / `internal/app/service.go` — service-layer DTO field `HyllaArtifactRef` untouched.
- `internal/tui/model_test.go` — TUI field-coverage tests untouched.
- All other test files referencing `HyllaArtifactRef` (domain, sqlite repo, snapshot, mcpapi extended_tools tests) — untouched.

## Acceptance Criteria Verification

- [x] `assemblePrompt` no longer emits `hylla_artifact_ref:` line. (post-edit `spawn.go:207-231`, no `hylla_artifact_ref` literal remains in the function body)
- [x] `domain.Project.HyllaArtifactRef` field PRESERVED. (`internal/domain/project.go:25` still declares the field; rg confirms zero deletions across the project field set)
- [x] `internal/app/dispatcher/spawn_test.go` flipped from "MUST contain" to "MUST NOT contain". (positive token list excludes the line; new explicit `strings.Contains` negative check added)
- [x] `mage check` passes. (21/21 packages green, all above 70% coverage, build green)
- [x] `mage ci` passes. (21/21 packages green, race + coverage thresholds clean, build green)
- [x] Worklog written. (this file)

## Verification Output Summary

`mage ci` final summary:

```
tests: 2265
passed: 2264
failed: 0
skipped: 1   (TestStewardIntegrationDropOrchSupersedeRejected — pre-existing skip)
packages: 21
pkg passed: 21
pkg failed: 0

Coverage:
  internal/app/dispatcher                              | 73.1%   (above 70% gate)
  ... all other packages also above 70%
  Minimum package coverage: 70.0% — SUCCESS

Build: SUCCESS
```

## Deviations From Spec

None of substance. Two scope-edge clarifications worth noting:

- **Doc-comment updates inside `spawn.go`:** the spec listed `spawn.go` and `spawn_test.go` as the only edit targets. Within `spawn.go` I updated TWO doc-comments in addition to the function body — one on `SpawnDescriptor.Prompt` (line 62-66) and one on `assemblePrompt` (line 192-206). Both previously cited `hylla_artifact_ref` as a "structural field the function guarantees." Leaving them stale would have made the doc-comments lie about post-edit behavior. The edits stay within the locked package (`internal/app/dispatcher`) and within the listed file.
- **`fixtureProject` doc-comment update inside `spawn_test.go`:** same rationale — the pre-edit comment claimed BuildSpawnCommand reads `HyllaArtifactRef`; post-edit it does not. Comment updated for accuracy. Struct literal unchanged.

## Hylla Feedback

`N/A — task touched non-Go-only files only via tool routing` is wrong-shape here. Correct shape: **`None — Hylla answered everything needed`**, except no Hylla queries were issued at all. Per project CLAUDE.md "Hylla is stale post-Drop-4b-merge" + the spawn-prompt directive ("NO Hylla calls — use `Read` / `Grep` / `LSP` directly"), this droplet routed all Go code lookups through `Read` and `rg`. No Hylla miss to record because no Hylla query was attempted; the routing was deliberate per the spawn-prompt directive.
