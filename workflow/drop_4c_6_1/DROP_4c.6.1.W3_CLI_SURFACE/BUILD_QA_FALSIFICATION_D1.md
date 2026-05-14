# W3.D1 - BUILD-QA-FALSIFICATION Verdict

**Date:** 2026-05-14
**Reviewer:** go-qa-falsification-agent (opus)
**Overall verdict:** PASS WITH NITS

## Attack Hypotheses Tested

### A1 - Read-then-merge clobber [REFUTED, NIT-class test gap]

The implementation reads `existing` via `locateProjectForCLI`, then copies ALL first-class fields into locals (`name`, `description`, `hyllaArtifactRef`, `repoBareRoot`, `repoPrimaryWorktree`, `language`, `buildTool`, `devMcpServerName`) and overwrites only when `strings.TrimSpace(opts.X) != ""`. `metadata := existing.Metadata` is a struct value-copy, preserving Owner / Icon / Color / Homepage / Tags / StandardsMarkdown / KindPayload / CapabilityPolicy / OrchSelfApprovalEnabled / DispatcherCommitEnabled / DispatcherPushEnabled. Only `Owner` and `Groups` get mutated when flags are supplied.

NIT: existing tests verify success via substring match on `"Updated Project"` but do NOT explicitly assert that other fields survive a single-flag update. A direct `ListProjects` -> read-back check after a description-only update, asserting `Name`, `Language`, `BuildTool`, etc. unchanged, would close the proof gap.

### A2 - `groups[:0:0]` semantics on nil [REFUTED]

Per Go spec, `nil[:0:0]` is a valid zero-length zero-capacity slice expression (no panic). Subsequent `append` reallocates. The implementation pre-copies via `groups := append([]string(nil), existing...)` so even if `existing.Metadata.Groups` is nil the working slice is well-defined.

### A3 - `--add-group go --remove-group go` same call [REFUTED]

`applyGroupMutations` processes adds first, then removes. So `--add-group go --remove-group go` ends in "go removed". Order is deterministic and consistent regardless of CLI flag order on the command line. Spec does not pin order; the implemented order is sensible and documented in the function's leading comment.

### A4 - `--add-group GO --add-group go` case sensitivity [REFUTED]

`isAllowedProjectGroup` uses exact-string equality (`g == allowed`) against `allowedInitGroups = ["gen", "go", "fe"]`. `"GO"` is rejected with `unknown group "GO"` before any merge. Test `TestRunProjectUpdate_AddGroupRejectsUnknownGroup` covers the unknown-group reject path. Lowercase canonical preserved; uppercase rejected, not silently deduped.

### A5 - `--remove-group nonexistent` [REFUTED]

`removeGroups` are NOT validated against `allowedInitGroups`; they are filtered out via map lookup. Unknown values silently no-op. Test `TestRunProjectUpdate_RemoveGroupFiltersAndIsNoopWhenAbsent` covers the absent-group no-op. Spec line 229 explicitly says "no-op if not present" so behavior matches.

### A6 - `isAllowedProjectGroup` coverage [REFUTED]

The loop over `allowedInitGroups` rejects every value not in `["gen", "go", "fe"]`. Validation happens BEFORE the existing-project read, so a bad `--add-group` aborts cleanly without touching state. Test confirms.

### A7 - Empty `--owner ""` [REFUTED, behavior consistent with read-then-merge]

`if strings.TrimSpace(opts.owner) != "" { metadata.Owner = opts.owner }` -- empty owner is SKIPPED, NOT clobbered. Consistent with the spec's read-then-merge philosophy ("overwrite only flag-provided fields"). Side effect: there is no way to clear Owner back to empty via this CLI -- but that is not in AcceptanceCriteria. Documented design choice.

### A8 - Mutation context (UpdatedBy / UpdatedByName / UpdatedType) [REFUTED]

`ctx = cliMutationContext(ctx, cfg)` attaches `MutationActor{ActorID, ActorName, ActorType}` to context (cmd/till/main.go:3028). `Service.UpdateProject` then calls `withResolvedMutationActor(ctx, in.UpdatedBy, in.UpdatedByName, in.UpdatedType)` (service.go:878) which falls back to the context-attached actor when input fields are empty (service.go:3347-3358). Builder leaves `in.UpdatedBy / in.UpdatedByName / in.UpdatedType` at zero, which is fine because context carries the actor. Pattern matches `runProjectCreate` and other CLI mutation flows.

### A9 - Idempotent re-run [REFUTED, expected behavior]

Re-running the same update always re-reads, re-merges, re-writes, and refreshes `UpdatedAt` via `s.clock()`. Always-overwrites, not no-op-on-equal. Spec does not require no-op-on-equal idempotency; read-merge-write semantics are the contract.

### A10 - `--language=""` explicit empty [REFUTED]

Same gate as A7: `strings.TrimSpace("") != ""` is false, so existing language is preserved. Test case `empty language accepted` verifies no error returned. Documented behavior matches read-then-merge.

### A11 - YAGNI / spec drift [NIT, raise as refinement]

KindPayload `shape_hint` (PLAN.md line 273) enumerates these fields on `projectUpdateCommandOptions`: `projectID, name, description, rootPath, bareRoot, language, hyllaArtifactRef, buildTool, devMcpServerName, owner, icon, color, homepage, tags, addGroups, removeGroups`.

Builder's actual options struct (cmd/till/main.go diff) has: `projectID, description, rootPath, bareRoot, language, hyllaArtifactRef, buildTool, devMcpServerName, owner, addGroups, removeGroups`.

Missing: `name, icon, color, homepage, tags`.

`name` is correctly deferred -- W3.D2 owns `rename` (which takes `--name <new-name>`). The other four (`icon`, `color`, `homepage`, `tags`) are NOT in AcceptanceCriteria bullets but ARE in KindPayload shape_hint. This is spec ambiguity, not builder failure. AcceptanceCriteria is the load-bearing contract per PLAN convention, so the implementation is spec-compliant on the load-bearing surface. NIT: either tighten KindPayload to match AcceptanceCriteria, or add the four missing flags. The dev should choose; either is defensible.

### A12 - `writeProjectDetail` output completeness [NIT, spec phrasing drift]

Spec line 220: `... returns the updated project as a JSON blob on stdout.`

Builder uses `writeProjectDetail(stdout, project, "Updated Project")` which renders a `laslig.KV` human-readable key/value block, NOT a JSON blob.

The same pattern is used by `runProjectCreate` (project_cli.go:151 `writeProjectDetail(stdout, project, "Created Project")`), so builder is consistent with the established CLI convention. Only one AcceptanceCriteria bullet says "JSON blob" (`--root-path`); the others just say "updates X" without specifying format.

Additionally, `writeProjectDetail` (project_cli.go:414-430) renders these rows: `name, id, slug, owner, icon, color, homepage, tags, archived, description, standards_markdown`. It does NOT surface `RepoPrimaryWorktree, RepoBareRoot, Language, BuildTool, DevMcpServerName, HyllaArtifactRef, Groups`. So the user cannot visually confirm a `--root-path` or `--language` update from the CLI output -- the change is silent except for the title.

This is spec-vs-implementation drift but the spec is internally inconsistent. The pattern aligns with `runProjectCreate`. Raise as a refinement: either extend `writeProjectDetail` to show the six post-Drop-4a first-class fields + Groups, or carve a `writeProjectDetailJSON` for update/create.

## Additional probes (beyond the 12 hypotheses)

### A13 - Whitespace trim policy [NIT]

`runProjectUpdate` validates `opts.addGroups` against `isAllowedProjectGroup` BEFORE any trim (line 175). `applyGroupMutations` THEN trims each value (line 268). Result: `--add-group "  go  "` is rejected by validation, not accepted-after-trim. Over-rejection is safe but inconsistent. Spec does not pin trim behavior. NIT: either trim BEFORE validation OR drop the trim in `applyGroupMutations` to make policy uniform.

### A14 - Slice aliasing bug [REFUTED]

`applyGroupMutations(metadata.Groups, ...)`: the first line is `groups := append([]string(nil), existing...)` which always allocates a fresh backing array (cap == len). All subsequent appends to `groups` are on the fresh allocation, not the caller's `metadata.Groups`. `filtered := groups[:0:0]` reuses `groups`'s backing array but only after the dedup pass has completed; the original `existing.Metadata.Groups` (the project's persisted slice) is never reachable from `filtered`. No aliasing bug.

### A15 - Empty-flag-set idempotent path [REFUTED]

When BOTH `opts.addGroups` and `opts.removeGroups` are empty: `applyGroupMutations` returns a fresh slice copy of `existing` (or nil if existing was nil). `metadata.Groups = groups`. Then `UpdateDetails -> normalizeProjectMetadata` does NOT touch `meta.Groups` (verified at internal/domain/project.go:456-468). Result: Groups unchanged on no-flag invocations. OK.

### A16 - Pre-existing single-update race [REFUTED, out of scope]

`Service.UpdateProject` does GET -> mutate-in-place -> UPDATE without an explicit write lock. Two concurrent updates could lose one. Pre-existing pattern -- not introduced by this droplet.

## Unmitigated Counterexamples

None. All attacks either REFUTED or downgraded to NIT-class.

## NITs

1. **NIT-D1-1 (spec drift, JSON vs human-readable output)** -- PLAN.md line 220 says "returns the updated project as a JSON blob on stdout" but builder follows the `runProjectCreate` precedent and writes a `laslig.KV` block. Decide one direction: drop the "JSON blob" phrasing from PLAN.md, or carve `writeProjectDetailJSON` for update.

2. **NIT-D1-2 (writeProjectDetail completeness)** -- `writeProjectDetail` does not surface `RepoPrimaryWorktree`, `RepoBareRoot`, `Language`, `BuildTool`, `DevMcpServerName`, `HyllaArtifactRef`, or `Groups`, so the user cannot visually confirm a `--root-path`/`--language`/etc. update from CLI output. Extend `writeProjectDetail` rows or add a separate detail view that includes the Drop 4a L4 first-class fields and Groups.

3. **NIT-D1-3 (KindPayload shape_hint vs AcceptanceCriteria drift)** -- `shape_hint` enumerates `icon, color, homepage, tags` fields on `projectUpdateCommandOptions` but AcceptanceCriteria does not require corresponding `--icon`/`--color`/`--homepage`/`--tags` flags. Builder omitted them. Either add the four flags (with read-then-merge gates symmetric to `--owner`) OR tighten the shape_hint to match AcceptanceCriteria.

4. **NIT-D1-4 (whitespace trim policy inconsistency)** -- Validation in `runProjectUpdate` uses raw `g`; application in `applyGroupMutations` trims. `--add-group "  go  "` is rejected by validation; consistent trim before validation OR no trim in mutations would be more uniform. Low impact; over-rejection is safe.

5. **NIT-D1-5 (clobber-safety test depth)** -- Existing tests verify each flag updates the targeted field, but do NOT explicitly assert that other first-class fields survive a single-flag update unchanged. A round-trip read-back assertion (e.g., set Language=go via service directly, then run update with `--description "x"`, then read back and assert Language is still "go") would close the proof gap on the read-then-merge contract.

6. **NIT-D1-6 (Owner is sticky)** -- There is no flag to clear `--owner` back to empty. Once set, it can only be replaced with another non-empty value. Spec does not require clear-to-empty support; documented design choice. If a clear path is desired, introduce `--clear-owner` flag pattern.

## Verdict rationale

The implementation correctly enforces the read-then-merge contract on all 8 first-class fields plus Metadata. The `applyGroupMutations` helper is well-behaved on nil, empty, and populated input; case-sensitive validation against `allowedInitGroups` matches the closed enum; `--remove-group` is non-validating and silently no-ops on unknown values per spec. Mutation context propagates via `cliMutationContext`. The Kind field on `UpdateProjectInput` is unused by `Service.UpdateProject` (UpdateDetails never reads `in.Kind`), so the builder's zero-value pass-through is safe.

The six NITs above are spec-drift and test-depth observations, none of which break the AcceptanceCriteria bullets at PLAN.md lines 219-232. All 351 `cmd/till` tests pass under `mage testPkg ./cmd/till`. `mage ci` claimed green by builder; not re-verified by this falsification pass per QA-no-mutation discipline.

Recommend orchestrator: merge D1, raise NITs as refinements (NIT-D1-1 + NIT-D1-2 + NIT-D1-3 are spec-level for planner; NIT-D1-4 + NIT-D1-5 + NIT-D1-6 are NIT-absorption candidates for a follow-up builder).
