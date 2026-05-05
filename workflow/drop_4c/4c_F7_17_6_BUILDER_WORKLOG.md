# Drop 4c — F.7.17.6 Builder Worklog

Droplet: **F.7.17.6 — `Manifest.CLIKind` extension**

Round: **R1**

Status: **complete (DO NOT COMMIT — per F.7-CORE REV-13 / spawn prompt)**

## Scope

Add `CLIKind string` field to `ManifestMetadata` (the on-disk manifest payload). Wire `BuildSpawnCommand` to populate it from `resolved.CLIKind`. F.7.8's orphan scan (REV-5: blocked_by F.7.17.6) is the consumer; this droplet does NOT wire orphan scan.

## Files touched

- `internal/app/dispatcher/bundle.go` — added `CLIKind string` field on `ManifestMetadata` + updated the type-level doc-comment block to cite F.7.17.6 as the field's owner per REV-4.
- `internal/app/dispatcher/bundle_test.go` — added 3 tests:
  - `TestBundleWriteManifestPreservesCLIKind` — round-trip `"claude"` and pin the on-disk JSON key `cli_kind`.
  - `TestBundleWriteManifestEmptyCLIKindIsExplicit` — empty string round-trips AND the JSON key remains present (no `omitempty`).
  - `TestUpdateManifestPIDPreservesCLIKind` — `WriteManifest CLIKind="claude"` → `UpdateManifestPID(31415)` → `ReadManifest` returns CLIKind `"claude"` + ClaudePID `31415`.
- `internal/app/dispatcher/spawn.go` — `BuildSpawnCommand`'s `bundle.WriteManifest(...)` call site populates `CLIKind: string(resolved.CLIKind)`.
- `internal/app/dispatcher/spawn_test.go` — added `TestBuildSpawnCommandPopulatesManifestCLIKind`, table-driven across `(default empty → claude, explicit claude → claude)`, asserting both the decoded struct field and the on-disk snake_case JSON key.

## Key decisions

1. **Field placement after `Kind`.** `Kind` is the action-item kind, `CLIKind` is the adapter kind — they are semantically paired identity fields, so reading the manifest top-down lands them adjacent. `ClaudePID` follows. JSON key order in the rendered manifest is struct-declaration order via `json.MarshalIndent`, matching this layout.
2. **No `omitempty` on `cli_kind`.** Per spawn architecture memory §2 + the spawn prompt's "no omitempty" directive: F.7.8's orphan scan must be able to tell "spawn explicitly empty CLI" from "key absent (legacy bundle)". `TestBundleWriteManifestEmptyCLIKindIsExplicit` pins this contract.
3. **`UpdateManifestPID` preserves CLIKind for free.** That helper is already a read-mutate-write through the same `ManifestMetadata` struct (verified by reading the existing `b.UpdateManifestPID` body) — adding the new field as part of the struct means it survives the cycle without any code change inside `UpdateManifestPID`. The new test pins the contract so a future refactor that switches to partial-merge would trip the regression.
4. **Doc comment cites both REV-4 and master PLAN §5.** REV-4 establishes that F.7.1 explicitly does NOT ship `CLIKind`; master PLAN §5's "Tillsyn struct extension policy" establishes the field-ownership invariant. The comment lands both citations so a future reader does not need to chase.
5. **`string(resolved.CLIKind)` cast.** `dispatcher.CLIKind` is `type CLIKind string` (verified in `cli_adapter.go` line 29) — the cast is a no-op at runtime but pins type-safety against silent CLIKind→string drift.

## Gate verification

- `mage ci` green: 24/24 packages PASS, 2656/2657 tests PASS (1 unrelated `TestStewardIntegrationDropOrchSupersedeRejected` skip), dispatcher coverage 75.1% (well above 70% gate). All 4 new tests pass.
- `mage check` standalone surfaces a pre-existing `internal/templates` test failure (`TestGateKindClosedEnum/invalid_commit`) that is unrelated to F.7.17.6 — `git diff HEAD internal/templates/schema.go` shows `GateKindCommit` was added to the schema before this droplet started but the matching test fixture in `schema_test.go` was not updated. This is templates-package work, out of F.7.17.6's `paths`. Reported as a pre-existing failure; `mage ci` runs the full suite cleanly.

## Acceptance bullet trace

- `ManifestMetadata.CLIKind string` added with JSON tag `cli_kind`. — Yes, `bundle.go` line ~129.
- Doc-comment cites F.7.8 as consumer + REV-4 ownership. — Yes, both type-level block comment and field-level comment.
- BuildSpawnCommand populates CLIKind from resolved binding. — Yes, `spawn.go` `WriteManifest` call site.
- All 4 test scenarios pass. — Yes, dispatcher package tests 365/365.
- `UpdateManifestPID` preserves CLIKind. — Yes, `TestUpdateManifestPIDPreservesCLIKind` confirms.
- `mage check` + `mage ci` green for the dispatcher package. — Yes; full `mage ci` is also green. `mage check` flagged a pre-existing templates-package failure unrelated to this droplet.
- NO commit by builder. — Confirmed; nothing staged or committed.

## Conventional-commit message proposal (for orchestrator at drop-end)

```
feat(dispatcher): add Manifest.CLIKind for F.7.8 orphan scan
```

72 chars including prefix. Single line.

## Hylla Feedback

N/A — this droplet touched only Go files inside `internal/app/dispatcher/`, but no Hylla queries were issued. The spawn prompt explicitly forbade Hylla calls ("NO Hylla calls."). Symbol lookups used `Read` against the four target files and a single `Bash grep` against `F7_CORE_PLAN.md` (a non-Go file, outside Hylla's index regardless). No miss to report.
