# Build-QA-Falsification — W3.D2 (project delete/archive/restore/rename CLIs)

Reviewer role: build-qa-falsification (Go)
Droplet: W3.D2
Authoritative spec: `workflow/drop_4c_6_1/DROP_4c.6.1.W3_CLI_SURFACE/PLAN.md` line 282
Builder claim: 4 lifecycle subcommands wired; --confirm required for delete; rename uses read-then-merge UpdateProject; 10 new tests; 370/370 PASS.
Hylla: OFF (per spawn directive). Tooling used: Read / Grep / git diff / mage test-pkg / mage test-func.

## Round 1 — 2026-05-14

### Attack hypotheses (from spawn prompt) and disposition

1. **--confirm bool vs positional** — EXHAUSTED, no counterexample. `projectDeleteCommandOptions.confirm bool` (project_cli.go:312-315); `runProjectDelete` (project_cli.go:342) checks `!opts.confirm` and returns clear error. Test `TestRunProjectDelete_RequiresConfirm` (project_cli_test.go:1108) confirms missing-confirm error string contains both `--confirm` and `hard delete is irreversible`. Cobra wiring is W3.D7's scope; not in this droplet.

2. **rename clobber risk — full UpdateProjectInput field preservation** — REFUTED. `UpdateProjectInput` is defined at `internal/app/service.go:855-870` with 14 fields. The rename implementation (project_cli.go:403-414) passes 9 of them: ProjectID, Name, Description, Metadata, HyllaArtifactRef, RepoBareRoot, RepoPrimaryWorktree, Language, BuildTool, DevMcpServerName. The 4 omitted (Kind, UpdatedBy, UpdatedByName, UpdatedType) are non-clobbering by design:
   - `Kind`: `UpdateProject` service (service.go:872-914) does NOT pass `in.Kind` to `project.UpdateDetails`; the project's existing kind is preserved because `UpdateDetails` (project.go:323-367) never touches `p.Kind`. Therefore `Kind` is a dead field on `UpdateProjectInput` for the rename code-path. (Also dead for `runProjectUpdate` — both follow the same convention.) This is a refinement-class observation, not a counterexample.
   - `UpdatedBy / UpdatedByName / UpdatedType`: propagated via `cliMutationContext(ctx, cfg)` (project_cli.go:402) before the service call. Parity with `runProjectUpdate` (project_cli.go:188 + 244-255), which also omits them from `UpdateProjectInput`. The audit context flows through ctx, not the input struct.
   - `Slug` is recomputed inside `UpdateDetails` (project.go:352) from the new Name — correct rename behavior.
   - `CapabilityPolicy` lives inside `ProjectMetadata` (project.go:127) which is preserved via `Metadata: existing.Metadata` (project_cli.go:407).
   - `KindCatalogJSON` is set once at project creation and frozen for the project's lifetime per project.go:81-85 doc-comment — not on `UpdateProjectInput`, so no preservation risk.

   Test `TestRunProjectRename_PreservesAllOtherFields` (project_cli_test.go:1286-1350) seeds a fully-populated project (Description, HyllaArtifactRef, RepoBareRoot, RepoPrimaryWorktree, Language, BuildTool, DevMcpServerName, Metadata.Owner), renames it, and asserts each preserved.

3. **delete missing --confirm: error message format match** — REFUTED. Test 1120-1124 asserts substring match on `--confirm` and `hard delete is irreversible`. Production string at project_cli.go:343: `"till project delete requires --confirm flag; hard delete is irreversible"`. Both substrings present. Aligned with PLAN.md ContextBlock line 332.

4. **archive on already-archived: idempotent or error?** — EXHAUSTED. Not specified in PLAN.md AcceptanceCriteria. `Service.ArchiveProject` (service.go:917-) delegates to `project.Archive(s.clock())` which is the domain's decision. CLI is correctly thin — it forwards the call. Whatever idempotency semantics domain.Project.Archive() implements is preserved by the CLI. No counterexample.

5. **restore on non-archived: idempotent or error?** — EXHAUSTED. Same shape as #4. CLI is thin forwarder. No counterexample.

6. **rename empty `--name ""`: clear error?** — REFUTED. `runProjectRename` (project_cli.go:395-397) trims `opts.newName` and checks empty; returns `"project rename requires --name <new-name>; new name cannot be empty"`. Test `TestRunProjectRename_MissingNameReturnsError` (project_cli_test.go:1250-1265) asserts error contains `--name`. Note: the test only checks `--name` substring but the production string explicitly states `cannot be empty` — stronger than the assertion. PASS.

7. **delete idempotency: delete twice — second call errors?** — EXHAUSTED, no counterexample. Second delete returns the repo's `GetProject` not-found error wrapped as `"delete project: <err>"`. Standard idempotency-via-error pattern; acceptable. Not specified in PLAN.md AcceptanceCriteria.

8. **`cliMutationContext` called in all 4 mutating functions?** — REFUTED. Verified inline:
   - `runProjectDelete` line 345.
   - `runProjectArchive` line 361.
   - `runProjectRestore` line 377.
   - `runProjectRename` line 402.
   All four call `cliMutationContext(ctx, cfg)` before the (*Service) mutation. Note: rename calls it AFTER `locateProjectForCLI` — this is correct because the read path (`locateProjectForCLI`) is a query, not a mutation, and is consistent with `runProjectUpdate` (project_cli.go:183-188).

9. **writeProjectDetail output: archive/restore/rename surface full fields per W3.D1 absorption?** — REFUTED. `writeProjectDetail` (project_cli.go:544-566) emits 18 KV rows including root_path, bare_root, language, build_tool, dev_mcp_server_name, hylla_artifact_ref, groups — all the W3.D1 absorption fields. archive/restore/rename use this helper directly with titles "Archived Project" / "Restored Project" / "Renamed Project".

10. **YAGNI: anything beyond spec?** — REFUTED. Each runFn is the minimal shape: validate inputs, gate svc nil, set mutation ctx, call service, write detail. No speculative abstractions, no unused options struct fields, no helper-helper layers.

11. **Cross-droplet bleed: edits confined to project_cli.go + project_cli_test.go?** — REFUTED. `git diff --name-only` returns exactly:
    - `cmd/till/project_cli.go`
    - `cmd/till/project_cli_test.go`
   Matches declared paths.

### Additional attack families

12. **Plan-spec vs final-code drift** — REFUTED. `KindPayload.changes` (PLAN.md lines 339-352) lists 4 options structs + 4 run functions + 4 test families. All 8 production symbols and 4 test families exist on disk with the named signatures.

13. **DecisionLog adequacy** — EXHAUSTED. No DecisionLog file requested; PLAN.md scope is the spec. No design decisions in this droplet warranted a separate log.

14. **Concurrency / goroutine leaks / interface misuse / error swallowing** — REFUTED. No goroutines spawned. No type assertions. All errors wrapped with `fmt.Errorf("... %w", err)` at the service-call boundary (project_cli.go:347, 364, 380, 416). No `_ = err`.

15. **Raw go commands / mage install in code or scripts** — REFUTED. No shell invocations in production. Tests use `mage test-pkg` (verified via my own run, see Tests below).

16. **`mage ci` smoke** — `mage test-pkg ./cmd/till` returned 370/370 PASS in 9.04s (also confirmed by builder's claim). I did not run `mage ci` separately (W3.D2 is mid-build; per WORKFLOW Phase 4 the per-droplet gate is `mage test-pkg`, not `mage ci`, which is drop-end).

### Tests run

- `mage test-pkg ./cmd/till` → 370/370 PASS (matches builder's 370/370 claim).
- `mage test-func ./cmd/till TestRunProjectDelete_SuccessPath` → 1/1 PASS (9.10s with -race).
- `mage test-func ./cmd/till TestRunProjectRename_PreservesAllOtherFields` → 1/1 PASS (9.04s with -race).

### Counterexamples — CONFIRMED

None.

### NIT — refinement-class observations

- **NIT-1 (informational, NOT a counterexample, NOT a blocker):** `UpdateProjectInput.Kind` (service.go:859) is a dead field on the rename code-path (and on `runProjectUpdate`'s code-path too). Service's `UpdateProject` does not pass `in.Kind` to `project.UpdateDetails`. Either the field should be wired through `UpdateDetails`, or removed from `UpdateProjectInput`. This is upstream-to-W3.D2: W3.D2 inherited the behavior. File against a future refinement drop, not against this builder.
- **NIT-2 (informational):** The rename missing-name test (`TestRunProjectRename_MissingNameReturnsError`, line 1250) only asserts `--name` substring; the production error string is stronger (`"till project rename requires --name <new-name>; new name cannot be empty"`). Tightening the assertion to include `cannot be empty` would catch future regressions where someone weakens the message. NIT-class only — current test still passes the contract.
- **NIT-3 (informational):** Archive/restore idempotency semantics are not exercised by tests, and are not specified in PLAN.md AcceptanceCriteria. If the dev wants a documented contract (idempotent vs error), file as a separate refinement against the domain layer — not against W3.D2.

### Unknowns

- W3.D7 (cobra wiring) is a downstream droplet; whether `--confirm` is wired as a bool flag at the cobra layer cannot be verified at W3.D2 boundary. This is correctly out-of-scope for W3.D2 build-QA.

### Verdict

**PASS — falsification produced no concrete counterexample.**

Builder's claim survives attack: 4 lifecycle subcommands correctly wired, --confirm required and surfaced as clear error, rename preserves all UpdateDetails-relevant fields via read-then-merge, audit context flows via `cliMutationContext` in all 4 mutators, no cross-droplet bleed, 370/370 PASS verified.

3 NITs are informational/refinement-class; none block W3.D2 close-out.
