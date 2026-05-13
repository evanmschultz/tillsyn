# DROP_4c.6.1.W3_CLI_SURFACE — L2 Plan

**Drop:** 4c.6.1.W3 — CLI Surface
**Kind:** plan (sub-plan container)
**State:** planning
**L2 Planner:** go-planning-agent (round 2)
**Blocked by (wave):** 4c.6.1.W2 (cmd/till package compile lock), 4c.6.1.W1 (HOME-tier path contract)
**Blocks:** nothing downstream (Wave D — last wave)
**Directory:** `workflow/drop_4c_6_1/DROP_4c.6.1.W3_CLI_SURFACE/`
**Hylla artifact ref:** `github.com/evanmschultz/tillsyn@main`

---

## Round 2 Changes (L2 plan-QA round-1 absorption — 2026-05-12)

Round-1 plan-QA (proof FAIL: 1 FF + 6 NITs; falsification FAIL: 3 FFs + 6 NITs) returned 4 unique FFs + ~8 unique NITs. R10 cross-cutting decisions also applied. All absorbed below:

- **Proof FF1 / Fals FF1 (`--add-group`/`--remove-group` missing)**: ABSORB per R10-D2. W1.D2 ships typed `ProjectMetadata.Groups []string` BEFORE W3 dispatches. D1 acceptance + KindPayload + ContextBlocks now include both flags. The "if absent, omit these flags" RiskNote hedge is REMOVED. Wave-level AC1 updated.
- **Fals FF2 (ColumnID omit → domain rejects)**: ABSORB. D3 ContextBlock now has a `constraint severity=critical` block: CLI MUST resolve a default ColumnID via `(*Service).ListColumns` → first column → `ColumnID: column.ID`. "ColumnID (omit)" is replaced. Test row added.
- **Fals FF3 / Proof NIT1 (R3-NIT6 verbatim trailing-period drift)**: ABSORB. All 4 occurrences updated (AC8, D6 AcceptanceCriteria bullet, D6 ContextBlock, D6 test shape_hint) to match L1-authoritative form — no trailing period after the closing backtick on `` `--force` ``.
- **Proof NIT2 (smart-default coverage — only 4 of 12 kinds)**: ABSORB. D3 `TestRunActionItemCreate_StructuralTypeSmartDefault` shape_hint now enumerates all 12 kinds.
- **Proof NIT3 (D6 fan-out test missing 2 of 4 destinations)**: ABSORB. D6 `TestRunAgentsBootstrap_QAFanOut` shape_hint now enumerates all 4 destination files. D6 AcceptanceCriteria fan-out bullet updated to name all 4.
- **Proof NIT4 (pass-through flags untested)**: ABSORB. D3 KindPayload adds `TestRunActionItemCreate_PassThroughFlags` test family.
- **Proof NIT5 (service.go line numbers)**: DEFERRED-AS-NIT with reason: per direct Read of `internal/app/service.go`, the round-1 line citations (625, 669, 689, 709, 1035) are correct — they cite the `func` keyword lines. The proof-agent cited comment lines (624, 668, etc.) as "actual." Standard Go practice cites the func signature line. No change to service.go line citations.
- **Proof NIT6 / Fals NIT1c (`--role pre-Drop-2` qualifier stale)**: ABSORB. D3 RiskNote drops "pre-Drop-2" qualifier. `domain.Role` is a closed 9-value enum today; service doc-comment at `service.go:742-746` confirms "Empty string is permitted" unconditionally.
- **Fals NIT1 (`--blocked-by` routing — open-ended "verify" language)**: ABSORB. D3 ContextBlock now explicitly states: `BlockedBy` is wired via `CreateActionItemInput.Metadata.BlockedBy []string` — no post-create UpdateActionItem needed.
- **Fals NIT2 (D7 "163K LOC" factual error)**: ABSORB. Corrected to "~4,069 LOC (163KB file size)" — `wc -l cmd/till/main.go` = 4069.
- **Fals NIT3 (CONSUMER-TIE wording self-contradiction)**: ABSORB. Top-level ContextBlock second sentence replaced to eliminate the `run(ctx, args, &out, io.Discard)` / `runXxx` contradiction.
- **Fals NIT4 (D7 LOC budget conditional split not acknowledged)**: ABSORB. D7 RiskNotes adds conditional D7a/D7b split note.
- **Fals NIT5 (`cliMutationContext` implicit in D2)**: ABSORB. D2 ContextBlocks adds a `reference` entry for the `cliMutationContext` pattern.
- **Fals NIT6 (D4/D5 embedded FS API ambiguity)**: ABSORB. D4 and D5 RiskNotes tightened to specify `templates.DefaultTemplateFS.ReadFile(path)` for raw-bytes access, with concrete path forms.

---

## Objective

Wire 15 new CLI subcommands plus `till agents bootstrap` into the `cmd/till` package.
No new service methods — every command calls an existing `(*Service)` method or performs
direct OS-level file I/O to `~/.tillsyn/templates/<group>.toml` and
`~/.tillsyn/agents/<group>/`. All commands follow the CONSUMER-TIE TEST CONTRACT
(`runXxx(ctx, svc, opts, stdout)` end-to-end) inherited from Drop 4c.6 W2.

**Scope recap:**

- `till project update` — 1 subcommand (calls `(*Service).UpdateProject`; includes `--add-group`/`--remove-group` for `Metadata.Groups`)
- `till project delete/archive/restore/rename` — 4 subcommands (call existing service methods)
- `till action_item create` — 1 subcommand with FF4 smart-default `--structural-type` + ColumnID resolution
- `till template save/list/show/diff/restore` — 5 subcommands, file I/O only
- `till agents save/list/show/diff` — 4 subcommands, file I/O only
- `till agents bootstrap` — 1 subcommand with 2-into-4 QA fan-out
- `main.go` registration — wire all new cobra commands

---

## AcceptanceCriteria (Wave Level)

- AC1: `till project update --project-id <id> [flags]` updates the project's metadata fields (root-path, bare-root, language, description, hylla-artifact-ref, build-tool, dev-mcp-server-name, owner, icon, color, homepage, tags); `--add-group <name>` appends to `Metadata.Groups` (dedup); `--remove-group <name>` filters out; returns the updated project as JSON.
- AC2: `till project delete/archive/restore/rename` all execute end-to-end; delete requires `--confirm`; rename requires `--name`.
- AC3: `till action_item create` creates an action item and returns UUID + dotted address; `--structural-type` smart-defaults (plan→segment, refinement→segment, all others→droplet); explicit `--structural-type invalid` fails with clear error + valid-values list; ColumnID is auto-resolved from the project's first column.
- AC4: All 5 `till template` subcommands (save/list/show/diff/restore) execute without errors against a test HOME dir.
- AC5: All 4 `till agents` subcommands (save/list/show/diff) execute without errors against a test HOME dir.
- AC6: `till agents bootstrap --from <path> --dry-run` prints the copy plan with no file writes.
- AC7: `till agents bootstrap --from <path>` copies files with 2-into-4 QA fan-out (producing `plan-qa-proof-agent.md`, `build-qa-proof-agent.md`, `plan-qa-falsification-agent.md`, `build-qa-falsification-agent.md`), reports missing files, generates `orchestrator-managed.md` starter.
- AC8: `--force` flag on `till agents bootstrap` help text EXPLICITLY warns: "Overwrites destination files; any post-bootstrap customization is lost. Use `till agents save --from-project <id>` to push customization back to HOME tier before re-running bootstrap with `--force`"
- AC9: All new commands registered in `main.go`; `till --help` shows them.
- AC10: `mage test-pkg ./cmd/till/...` passes. `mage ci` green.

---

## ValidationPlan

- Per-droplet: `mage test-pkg ./cmd/till/...` after each droplet merges.
- Drop-end gate: `mage ci` after D7 merges.
- AC3 smart-default: `TestRunActionItemCreate_StructuralTypeSmartDefault` table test — covers all 12 kinds × {explicit-override-valid, explicit-override-invalid, default}.
- AC3 ColumnID: `TestRunActionItemCreate_PassThroughFlags` verifies created action item lands on the project's first column when no `--column-id` flag passed.
- AC6/AC7/AC8: `TestRunAgentsBootstrap_*` table tests cover dry-run (no writes), actual copy (2-into-4 fan-out with all 4 destinations), force-flag, missing-file reporting.
- AC4/AC5: `TestRunTemplate*` and `TestRunAgents*` tests use `testing/fstest.MapFS` or `os.MkdirTemp` for HOME dir isolation.

---

## RiskNotes

- **R1 — D2 size ceiling:** 4 subcommands (delete/archive/restore/rename) at ~25 LOC each = ~100 LOC. Within the 120 LOC ceiling but tight. If builder exceeds, split into D2a (delete/archive) + D2b (restore/rename) with D2b blocked_by D2a. Builder MUST NOT exceed 120 LOC production code per droplet.
- **R2 — D4 size ceiling:** 5 template subcommands with file I/O. Per-subcommand helper ~20 LOC + formatting = ~120 LOC. At the limit. Builder may split D4a (save/list/show) + D4b (diff/restore) with D4b blocked_by D4a if needed.
- **R3 — D6 complexity:** bootstrap fan-out logic has 4 distinct code blocks: (1) parse source dir, (2) compute fan-out copy plan, (3) execute copies (skip on dry-run), (4) report missing + generate starter. Builder MUST stay within 120 LOC. If exceeds, split D6a (scan + plan) + D6b (execute + report) with D6b blocked_by D6a.
- **R4 — No RenameProject service method:** `domain.Project.Rename()` exists but no `(*Service).RenameProject`. The `till project rename` CLI MUST call `(*Service).UpdateProject` with the new name and all other fields read from the existing project first. Builder must NOT add a new service method.
- **R5 — StructuralType required by domain:** `domain.NewActionItem` returns `ErrInvalidStructuralType` on empty StructuralType. The `till action_item create` CLI MUST always derive a non-empty StructuralType before calling `(*Service).CreateActionItem` — either from the `--structural-type` flag or from the smart-default map. Builder must never pass empty.
- **R6 — Package compile lock:** All D1–D7 touch the `cmd/till` package. The linear `blocked_by` chain (D1→D2→D3→D4→D5→D6→D7) is MANDATORY. Do not attempt to parallelize siblings within this wave.

---

## ContextBlocks

```
[constraint severity=critical]
All droplets share cmd/till package. The linear blocked_by chain D1→D2→D3→D4→D5→D6→D7
is a hard structural invariant. No sibling within W3 may run in parallel with another
because they share the cmd/till package compile unit. Pre-cascade dispatch: orchestrator
enforces this manually. Post-cascade: package lock manager serializes automatically.

[constraint severity=high]
No new service methods. Every new CLI command wires to an existing (*Service) method
or performs direct OS-level file I/O. Builder MUST NOT add any method to internal/app/
or internal/domain/.

[constraint severity=high]
StructuralType is mandatory at domain.NewActionItem — empty rejects with
ErrInvalidStructuralType. till action_item create MUST supply a non-empty value
(via smart-default or explicit override) before calling CreateActionItem.

[constraint severity=high]
CONSUMER-TIE TEST CONTRACT: matches L1 PLAN.md + W2 round-2's `run(ctx, args, &out, io.Discard)` end-to-end pattern. Each command supports an internal `runXxx(ctx, svc, cfg, opts, stdout) error` factored helper for unit-test reuse, but the PRIMARY test invocation is end-to-end through `run(ctx, args, &out, io.Discard)` — this exercises the cobra flag-binding layer that L1 acceptance demands. Supplementary `runXxx`-direct tests with constructed options structs are acceptable for narrow unit coverage, but every droplet MUST also have at least one end-to-end `run(ctx, args, &out, io.Discard)` invocation per acceptance bullet that asserts new behavior. This restores the round-1 fals NIT3 reconciliation direction (round-2 fals FF1: W3 picked the wrong reconciliation, killed `run(args)` end-to-end instead of the contradiction; restored here per orchestrator-direct edit 2026-05-12).

[reference severity=normal]
Service methods confirmed existing at internal/app/service.go:
  - UpdateProject (line 625)
  - ArchiveProject (line 669)
  - RestoreProject (line 689)
  - DeleteProject (line 709)
  - CreateActionItem (line 1035)
  - ListColumns (line 1870) — public API for default-column resolution
domain.StructuralType closed enum: drop|segment|confluence|droplet (structural_type.go:18-31)
domain.Role closed enum: 9 values (role.go:14-23)

[reference severity=normal]
HOME-tier paths (finalized by W1 before W3 dispatches):
  ~/.tillsyn/templates/<group>.toml
  ~/.tillsyn/agents/<group>/<name>.md

[decision severity=normal]
till project rename calls (*Service).UpdateProject — reads existing project first,
passes new name + all other existing fields unchanged. Pattern mirrors existing
project update flow. domain.Project.Rename() domain method exists at project.go:289
but no service-level RenameProject; UpdateProject via UpdateDetails handles rename+reslug.

[decision severity=normal]
FF4 smart-default structural-type mapping (REVISION_BRIEF §2.9):
  plan → segment
  refinement → segment
  all other 10 kinds → droplet
Override via --structural-type <val> validates against closed enum; invalid fails loud
with valid-values list in the error message.

[warning severity=critical]
R3-NIT6 (from L1 PLAN.md): --force flag help text on till agents bootstrap MUST
include VERBATIM: "Overwrites destination files; any post-bootstrap customization is
lost. Use `till agents save --from-project <id>` to push customization back to HOME
tier before re-running bootstrap with `--force`"
This is an acceptance criterion, not a suggestion. Builder cannot omit it. The string
ends with the closing backtick — NO trailing period after the backtick.

[note severity=normal]
2-into-4 QA fan-out: source <group>-qa-proof-agent.md seeds BOTH
plan-qa-proof-agent.md AND build-qa-proof-agent.md at destination. Same for
qa-falsification. QA-SPLIT-R1 tracks proper per-role differentiation in Drop 4c.8;
for now, both destination files get identical content.
```

---

## KindPayload (plan-level child preview)

```json
{
  "children": [
    {"id": "W3.D1", "kind": "build", "title": "W3.D1 — till project update CLI", "blocked_by": []},
    {"id": "W3.D2", "kind": "build", "title": "W3.D2 — till project delete/archive/restore/rename CLIs", "blocked_by": ["W3.D1"]},
    {"id": "W3.D3", "kind": "build", "title": "W3.D3 — till action_item create CLI (FF4)", "blocked_by": ["W3.D2"]},
    {"id": "W3.D4", "kind": "build", "title": "W3.D4 — till template save/list/show/diff/restore CLIs", "blocked_by": ["W3.D3"]},
    {"id": "W3.D5", "kind": "build", "title": "W3.D5 — till agents save/list/show/diff CLIs", "blocked_by": ["W3.D4"]},
    {"id": "W3.D6", "kind": "build", "title": "W3.D6 — till agents bootstrap CLI (2-into-4 QA fan-out)", "blocked_by": ["W3.D5"]},
    {"id": "W3.D7", "kind": "build", "title": "W3.D7 — main.go registration for all W3 commands", "blocked_by": ["W3.D6"]}
  ]
}
```

---

## CompletionContract

**StartCriteria:**
- 4c.6.1.W2 is in `complete` state (cmd/till package compile lock released).
- 4c.6.1.W1 is in `complete` state (HOME-tier path contract finalized: `~/.tillsyn/templates/<group>.toml` + `~/.tillsyn/agents/<group>/` paths are stable; `ProjectMetadata.Groups []string` typed field shipped in W1.D2).
- Builder has read this PLAN.md + REVISION_BRIEF §2.7–2.10 + §2.17 + SKETCH §10.

**CompletionCriteria:**
- All 7 droplets (D1–D7) in `complete` state with their build-QA twins green.
- `mage ci` green on the whole drop (post-D7).
- All 15 + 1 subcommands registered and reachable via `till --help`.

**CompletionChecklist:**
- [ ] D1 complete + build-QA green
- [ ] D2 complete + build-QA green
- [ ] D3 complete + build-QA green
- [ ] D4 complete + build-QA green
- [ ] D5 complete + build-QA green
- [ ] D6 complete + build-QA green
- [ ] D7 complete + build-QA green
- [ ] `mage ci` green post-D7

---

## Droplet Decomposition

### W3.D1 — `till project update` CLI

- **State:** todo
- **Kind:** build (atomic droplet; `Irreducible: true`)
- **Paths:**
  - `cmd/till/project_cli.go` (MODIFY — add `projectUpdateCommandOptions` struct + `runProjectUpdate`)
  - `cmd/till/project_cli_test.go` (MODIFY — add `TestRunProjectUpdate*` table tests)
- **Packages:** `cmd/till`
- **Blocked by:** none (first in W3 chain; W3 itself is blocked_by W2 and W1 at the wave level)

**Specify:**

Objective: Wire `till project update` as a new subcommand under `till project`. The command accepts flag-driven partial updates to a project's first-class fields and metadata including group membership. It reads the existing project first (to preserve all unset fields), merges explicit flag values, then calls `(*Service).UpdateProject`.

AcceptanceCriteria:
- `till project update --project-id <id> --root-path /abs/path` updates `RepoPrimaryWorktree` on the project record and returns the updated project as a JSON blob on stdout.
- `till project update --project-id <id> --bare-root /abs/path` updates `RepoBareRoot`.
- `till project update --project-id <id> --language go` updates `Language`; `--language invalid` fails with `ErrInvalidLanguage` surfaced as a clear error message.
- `till project update --project-id <id> --hylla-artifact-ref github.com/org/repo@main` updates `HyllaArtifactRef`.
- `till project update --project-id <id> --description "new desc"` updates `Description`.
- `till project update --project-id <id> --build-tool mage` updates `BuildTool`.
- `till project update --project-id <id> --dev-mcp-server-name tillsyn-dev` updates `DevMcpServerName`.
- `till project update --project-id <id> --owner "Evan"` updates `Metadata.Owner`.
- `till project update --project-id <id> --add-group fe` appends `"fe"` to `Metadata.Groups` (dedup — no-op if already present).
- `till project update --project-id <id> --remove-group go` removes `"go"` from `Metadata.Groups` (no-op if not present).
- `--add-group` and `--remove-group` may be repeated (e.g. `--add-group go --add-group fe`).
- Missing `--project-id` fails with the canonical project-discovery error.
- `mage test-pkg ./cmd/till/...` passes.

RiskNotes:
- `(*Service).UpdateProject` is value-typed (no pointer sentinels): caller must supply ALL fields including unchanged ones. Builder MUST read the existing project via `locateProjectForCLI` (or equivalent) BEFORE constructing `UpdateProjectInput`, then overwrite only the flag-provided fields.
- `--add-group`/`--remove-group` operate on `ProjectMetadata.Groups []string`. W1.D2 ships this typed field on `domain.ProjectMetadata` BEFORE W3 dispatches. Builder MUST verify via LSP `goToDefinition` on `domain.ProjectMetadata` that the `Groups []string` field exists before writing. If (unexpectedly) absent, STOP and return to orchestrator rather than adding a TODO stub.

ContextBlocks:
```
[constraint severity=high]
UpdateProjectInput is value-typed. Builder MUST read the existing project first and copy
ALL fields into the input struct, then overwrite only those supplied by flags. Passing
zero values for unset fields would silently clobber existing data.

[constraint severity=high]
--add-group and --remove-group operate on Metadata.Groups []string (shipped by W1.D2).
Builder MUST:
  1. Read existing project to get current Metadata.Groups.
  2. For --add-group: append if not present (dedup check via linear scan or sort.SearchStrings).
  3. For --remove-group: filter out the named value.
  4. Pass the mutated Metadata (ALL fields preserved) to UpdateProjectInput.Metadata.
Group values must be in the allowed set (go, fe, gen) per W2's allowedInitGroups.

[reference severity=normal]
UpdateProjectInput fields (service.go:607-622): ProjectID, Name, Description,
HyllaArtifactRef, RepoBareRoot, RepoPrimaryWorktree, Language, BuildTool,
DevMcpServerName, UpdatedBy, UpdatedByName, UpdatedType. Plus Metadata (domain.ProjectMetadata).

[reference severity=normal]
Existing pattern to follow: runProjectCreate (project_cli.go:133) + buildProjectMetadata
(project_cli.go:211). The update flow mirrors create but reads-then-merges.

[reference severity=normal]
cliMutationContext helper (project_cli.go:142 — confirmed by Read). Must be
called to populate the UpdatedBy/UpdatedByName/UpdatedType fields from the active
CLI identity config: ctx = cliMutationContext(ctx, cfg).
```

KindPayload:
```json
{
  "changes": [
    {"file": "cmd/till/project_cli.go", "symbol": "projectUpdateCommandOptions", "action": "add", "shape_hint": "struct { projectID string; name string; description string; rootPath string; bareRoot string; language string; hyllaArtifactRef string; buildTool string; devMcpServerName string; owner string; icon string; color string; homepage string; tags []string; addGroups []string; removeGroups []string }"},
    {"file": "cmd/till/project_cli.go", "symbol": "runProjectUpdate", "action": "add", "shape_hint": "func runProjectUpdate(ctx context.Context, svc *app.Service, cfg config.Config, opts projectUpdateCommandOptions, stdout io.Writer) error"},
    {"file": "cmd/till/project_cli_test.go", "symbol": "TestRunProjectUpdate_*", "action": "add", "shape_hint": "table-driven tests covering each flag + missing --project-id error + language validation + add-group dedup + remove-group no-op"}
  ]
}
```

---

### W3.D2 — `till project delete/archive/restore/rename` CLIs

- **State:** todo
- **Kind:** build (atomic droplet; `Irreducible: true`)
- **Paths:**
  - `cmd/till/project_cli.go` (MODIFY — add 4 options structs + 4 run functions)
  - `cmd/till/project_cli_test.go` (MODIFY — add tests for all 4 subcommands)
- **Packages:** `cmd/till`
- **Blocked by:** W3.D1 (shares `project_cli.go` and `project_cli_test.go`)

**Specify:**

Objective: Wire four project lifecycle subcommands: `delete`, `archive`, `restore`, `rename`. Each calls an existing service method. `delete` requires an explicit `--confirm` flag. `rename` requires `--name <new-name>`.

AcceptanceCriteria:
- `till project delete --project-id <id> --confirm` calls `(*Service).DeleteProject` and writes a confirmation line on stdout.
- `till project delete --project-id <id>` (missing `--confirm`) fails with a clear error explaining `--confirm` is required for hard-delete.
- `till project archive --project-id <id>` calls `(*Service).ArchiveProject` and returns the archived project detail.
- `till project restore --project-id <id>` calls `(*Service).RestoreProject` and returns the restored project detail.
- `till project rename --project-id <id> --name <new-name>` calls `(*Service).UpdateProject` with the new name (preserving all other existing fields) and returns the renamed project detail. New name must be non-empty.
- `till project rename --project-id <id>` (missing `--name`) fails with a clear error.
- `mage test-pkg ./cmd/till/...` passes.

RiskNotes:
- **No `RenameProject` service method** — `till project rename` MUST call `(*Service).UpdateProject`. Builder must: (1) call `locateProjectForCLI` or `s.repo.GetProject` to fetch the existing project, (2) construct `UpdateProjectInput` copying ALL existing fields, (3) set `in.Name = opts.newName`. The domain's `Project.Rename()` method (domain/project.go:289) is NOT used directly — it is called internally by `UpdateDetails` inside `UpdateProject`.
- D2 has 4 subcommands at ~25 LOC each = ~100 LOC total. If the builder finds the test file pushes total over 120 LOC for the production file alone, split: D2a covers delete+archive, D2b covers restore+rename (D2b blocked_by D2a).

ContextBlocks:
```
[constraint severity=critical]
till project rename MUST call (*Service).UpdateProject (NOT a nonexistent RenameProject).
Pattern: fetch existing project → copy all fields to UpdateProjectInput → set Name to new name.
This is the only correct approach. A new service method is out of scope for W3.

[reference severity=normal]
(*Service).DeleteProject signature: func (s *Service) DeleteProject(ctx context.Context, projectID string) error
(*Service).ArchiveProject signature: func (s *Service) ArchiveProject(ctx context.Context, projectID string) (domain.Project, error)
(*Service).RestoreProject signature: func (s *Service) RestoreProject(ctx context.Context, projectID string) (domain.Project, error)
(*Service).UpdateProject signature: func (s *Service) UpdateProject(ctx context.Context, in UpdateProjectInput) (domain.Project, error)
All verified at internal/app/service.go lines 625, 669, 689, 709.

[reference severity=normal]
Every mutating subcommand (delete/archive/restore/rename) MUST call
cliMutationContext(ctx, cfg) before any (*Service) call — pattern at
project_cli.go:142. Resulting ctx carries the active CLI identity for
UpdatedBy/UpdatedByName/UpdatedType audit fields.

[warning severity=high]
--confirm flag for delete: must be a cobra bool flag (e.g. --confirm), NOT positional.
Without it the command must print a clear error: "till project delete requires --confirm
flag; hard delete is irreversible."
```

KindPayload:
```json
{
  "changes": [
    {"file": "cmd/till/project_cli.go", "symbol": "projectDeleteCommandOptions", "action": "add", "shape_hint": "struct { projectID string; confirm bool }"},
    {"file": "cmd/till/project_cli.go", "symbol": "runProjectDelete", "action": "add", "shape_hint": "func runProjectDelete(ctx, svc, cfg, opts, stdout) error — requires opts.confirm"},
    {"file": "cmd/till/project_cli.go", "symbol": "projectArchiveCommandOptions", "action": "add", "shape_hint": "struct { projectID string }"},
    {"file": "cmd/till/project_cli.go", "symbol": "runProjectArchive", "action": "add", "shape_hint": "func runProjectArchive(ctx, svc, cfg, opts, stdout) error"},
    {"file": "cmd/till/project_cli.go", "symbol": "projectRestoreCommandOptions", "action": "add", "shape_hint": "struct { projectID string }"},
    {"file": "cmd/till/project_cli.go", "symbol": "runProjectRestore", "action": "add", "shape_hint": "func runProjectRestore(ctx, svc, cfg, opts, stdout) error"},
    {"file": "cmd/till/project_cli.go", "symbol": "projectRenameCommandOptions", "action": "add", "shape_hint": "struct { projectID string; newName string }"},
    {"file": "cmd/till/project_cli.go", "symbol": "runProjectRename", "action": "add", "shape_hint": "func runProjectRename(ctx, svc, cfg, opts, stdout) error — calls UpdateProject with new Name, all other fields from existing project"},
    {"file": "cmd/till/project_cli_test.go", "symbol": "TestRunProjectDelete_*", "action": "add", "shape_hint": "table tests: confirm required; success path"},
    {"file": "cmd/till/project_cli_test.go", "symbol": "TestRunProjectArchive_*", "action": "add"},
    {"file": "cmd/till/project_cli_test.go", "symbol": "TestRunProjectRestore_*", "action": "add"},
    {"file": "cmd/till/project_cli_test.go", "symbol": "TestRunProjectRename_*", "action": "add", "shape_hint": "table tests: missing name; success; verify all other fields preserved"}
  ]
}
```

---

### W3.D3 — `till action_item create` CLI (FF4 smart-default structural-type)

- **State:** todo
- **Kind:** build (atomic droplet; `Irreducible: true`)
- **Paths:**
  - `cmd/till/action_item_cli.go` (MODIFY — add `actionItemCreateCommandOptions` struct + `runActionItemCreate`)
  - `cmd/till/action_item_cli_test.go` (MODIFY — add `TestRunActionItemCreate*` table tests)
- **Packages:** `cmd/till`
- **Blocked by:** W3.D2 (same cmd/till package compile; D2 must settle before D3 extends a different file)

**Specify:**

Objective: Wire `till action_item create` with FF4 smart-default `--structural-type` per kind. The command calls `(*Service).CreateActionItem`. When `--structural-type` is omitted, the CLI derives the default from `--kind`. When supplied, it validates against the closed 4-value enum and fails loud on invalid values. The CLI also resolves a default ColumnID via `(*Service).ListColumns` before calling `(*Service).CreateActionItem`.

AcceptanceCriteria:
- `till action_item create --project-id <id> --kind plan --title "T" --description "D"` creates the action item with `StructuralType=segment` (smart-default) and returns UUID + dotted address on stdout.
- `till action_item create --project-id <id> --kind build --title "T" --description "D"` creates with `StructuralType=droplet`.
- `till action_item create --project-id <id> --kind refinement --title "T" --description "D"` creates with `StructuralType=segment`.
- All other 9 kinds (research, plan-qa-proof, plan-qa-falsification, build-qa-proof, build-qa-falsification, closeout, commit, discussion, human-verify) default to `StructuralType=droplet`.
- `till action_item create ... --structural-type confluence` uses the explicit override (valid).
- `till action_item create ... --structural-type invalid` fails with a clear error listing valid values (drop|segment|confluence|droplet).
- The created action item is placed on the project's first column (ColumnID auto-resolved; no `--column-id` flag required).
- `--paths`, `--packages`, `--files`, `--blocked-by`, `--metadata-json`, `--parent-id`, `--role` flags are accepted and passed through to the created action item.
- Missing `--project-id`, `--kind`, `--title`, or `--description` fails with a clear required-field error.
- `mage test-pkg ./cmd/till/...` passes.

RiskNotes:
- `domain.NewActionItem` returns `ErrInvalidStructuralType` on empty StructuralType — the smart-default MUST fire unconditionally BEFORE `(*Service).CreateActionItem` is called. Never pass an empty `StructuralType` to the service.
- `--role` is optional (closed enum; empty is valid). If non-empty, pass as `domain.Role(opts.role)` — the service validates it; the CLI does NOT validate the role enum independently (service's ErrInvalidRole surfaces the error).
- `--blocked-by` is a repeated flag (`[]string`), wired into `CreateActionItemInput.Metadata.BlockedBy []string` directly — no post-create UpdateActionItem is needed.

ContextBlocks:
```
[constraint severity=critical]
FF4 smart-default table (from REVISION_BRIEF §2.9 + SKETCH §10):
  plan → segment
  refinement → segment
  all other 10 kinds → droplet
Builder MUST implement this as a switch or map BEFORE constructing CreateActionItemInput.
Never pass empty StructuralType to the service.

[constraint severity=critical]
ColumnID is REQUIRED by domain.NewActionItem — empty ColumnID rejects with ErrInvalidColumnID.
till action_item create MUST resolve a default ColumnID BEFORE calling (*Service).CreateActionItem.
Canonical pattern (auto_generate_steward.go:123-158 + service.go:1870):
  columns, err := svc.ListColumns(ctx, projectID, false)
  if err != nil { return err }
  if len(columns) == 0 { return fmt.Errorf("project has no columns") }
  columnID := columns[0].ID   // ListColumns returns sorted by position ascending

[constraint severity=high]
Valid override values for --structural-type: drop | segment | confluence | droplet
(domain.StructuralType constants verified at internal/domain/structural_type.go:18-31).
Invalid value: fail loud with message including the valid list.

[reference severity=normal]
BlockedBy is wired via CreateActionItemInput.Metadata.BlockedBy []string
(the ActionItemMetadata struct field at internal/domain/workitem.go:195, reachable
via CreateActionItemInput.Metadata). The --blocked-by flag accumulates []string values
into opts.blockedBy; builder merges them into the constructed domain.ActionItemMetadata.BlockedBy
before calling (*Service).CreateActionItem. No post-create UpdateActionItem is needed.

[reference severity=normal]
CreateActionItemInput fields that the CLI wires: ProjectID, ParentID, Kind, Role,
StructuralType, Title, Description, Paths, Packages, Files, ColumnID (required — resolved
from ListColumns), StartCommit (omit), EndCommit (omit), Priority (omit), DueAt (omit),
Labels (omit), Metadata (pass --metadata-json if supplied; also set Metadata.BlockedBy
from --blocked-by accumulator).

[reference severity=normal]
Output format: write the created action item's ID + dotted address to stdout.
Follow the pattern of writeActionItemJSON (action_item_cli.go) or add a simpler
"Created action item <id> (dotted: <addr>)" one-liner per existing CLI output style.
```

KindPayload:
```json
{
  "changes": [
    {"file": "cmd/till/action_item_cli.go", "symbol": "actionItemCreateCommandOptions", "action": "add", "shape_hint": "struct { projectID string; parentID string; kind string; title string; description string; paths []string; packages []string; files []string; blockedBy []string; metadataJSON string; structuralType string; role string }"},
    {"file": "cmd/till/action_item_cli.go", "symbol": "runActionItemCreate", "action": "add", "shape_hint": "func runActionItemCreate(ctx, svc, cfg, opts, stdout) error — smart-default StructuralType, resolve ColumnID via ListColumns, then CreateActionItem"},
    {"file": "cmd/till/action_item_cli.go", "symbol": "structuralTypeSmartDefault", "action": "add", "shape_hint": "func structuralTypeSmartDefault(kind string) domain.StructuralType — switch over FF4 table; new, not yet in tree"},
    {"file": "cmd/till/action_item_cli_test.go", "symbol": "TestRunActionItemCreate_StructuralTypeSmartDefault", "action": "add", "shape_hint": "table test covering ALL 12 kinds: plan→segment, refinement→segment, build→droplet, research→droplet, plan-qa-proof→droplet, plan-qa-falsification→droplet, build-qa-proof→droplet, build-qa-falsification→droplet, closeout→droplet, commit→droplet, discussion→droplet, human-verify→droplet; plus explicit-override-valid (confluence), explicit-override-invalid (error with valid-values list)"},
    {"file": "cmd/till/action_item_cli_test.go", "symbol": "TestRunActionItemCreate_RequiredFields", "action": "add", "shape_hint": "table test: missing project-id, missing kind, missing title, missing description"},
    {"file": "cmd/till/action_item_cli_test.go", "symbol": "TestRunActionItemCreate_PassThroughFlags", "action": "add", "shape_hint": "verifies each pass-through flag (paths, packages, files, blocked-by, metadata-json, parent-id, role) makes it onto the created domain.ActionItem; especially --blocked-by sets Metadata.BlockedBy; ColumnID is auto-resolved to first column; new, not yet in tree"}
  ]
}
```

---

### W3.D4 — `till template save/list/show/diff/restore` CLIs

- **State:** todo
- **Kind:** build (atomic droplet; `Irreducible: true`)
- **Paths:**
  - `cmd/till/template_cli.go` (NEW — create with 5 subcommand run functions + options structs)
  - `cmd/till/template_cli_test.go` (NEW — create with table tests for all 5 subcommands)
- **Packages:** `cmd/till`
- **Blocked by:** W3.D3 (same cmd/till package compile)

**Specify:**

Objective: Wire 5 `till template` subcommands. All perform direct OS-level file I/O to `~/.tillsyn/templates/<group>.toml` (HOME tier) and project local `<project>/.tillsyn/template.toml`. No service calls. `save` reads from project dir, writes to HOME. `list` shows both tiers. `show` prints one tier's content. `diff` diffs HOME vs embedded. `restore` copies embedded default back to HOME with confirm.

AcceptanceCriteria:
- `till template save --from-project <id> --group <group>` reads `<project-worktree>/.tillsyn/template.toml`'s `[<group>]` section and writes to `~/.tillsyn/templates/<group>.toml`. Confirms with user before overwrite (or `--force` to skip confirm).
- `till template list` shows HOME templates found under `~/.tillsyn/templates/` and embedded defaults, formatted as a table (group + source + present/absent).
- `till template show --group <group> --source home` prints the content of `~/.tillsyn/templates/<group>.toml`; `--source embedded` prints the embedded default.
- `till template diff --group <group>` shows diff between HOME tier and embedded default (use `fmt.Sprintf` or simple line-by-line diff; no external diff binary).
- `till template restore --group <group>` copies the embedded default to `~/.tillsyn/templates/<group>.toml`; asks for confirm unless `--force`.
- All commands surface clear errors when the requested group or file is not found.
- Tests use `os.MkdirTemp` to isolate HOME dir operations; no real `~/.tillsyn` touched.
- `mage test-pkg ./cmd/till/...` passes.

RiskNotes:
- 5 subcommands in one file is at the upper end of atomic-droplet sizing. If the builder finds the production file exceeds 120 LOC, split: D4a covers `save` + `list` + `show` (file = template_cli.go), D4b covers `diff` + `restore` (same file, blocked_by D4a since they share template_cli.go).
- The `save` and `restore` subcommands require user confirmation before overwriting. Implement a simple `promptConfirm(stdout, io.Reader, message string) bool` helper (or reuse an existing confirm helper if W2 or init_cmd.go already defines one — check via LSP before adding a duplicate).
- HOME dir path: use `os.UserHomeDir()` + `.tillsyn/templates/` as the base. Do NOT hardcode `~` — use the Go standard library.
- Embedded defaults: use `templates.DefaultTemplateFS.ReadFile(path)` for raw-bytes access (path form: `builtin/till-<group>.toml`). Do NOT use `templates.LoadDefaultTemplateForLanguage` — that returns a parsed Template struct, not raw bytes; `show --source embedded` and `diff` need raw bytes for content display.

ContextBlocks:
```
[constraint severity=high]
File I/O only — no service calls. These 5 subcommands operate entirely on
~/.tillsyn/templates/<group>.toml (HOME tier) and the embedded FS.

[reference severity=normal]
HOME tier path formula: filepath.Join(os.UserHomeDir(), ".tillsyn", "templates", group+".toml")
This matches the W1-finalized contract. W1 shipped before W3; use this exact path.

[reference severity=normal]
Embedded FS raw-bytes access: templates.DefaultTemplateFS.ReadFile("builtin/till-<group>.toml")
This returns the raw TOML bytes suitable for content display and diff.
Do NOT use LoadDefaultTemplateForLanguage — that returns a parsed Template struct.
Verify the actual path forms in internal/templates/embed.go before writing.

[warning severity=high]
Do NOT call mage install. Do NOT call mage test directly; use mage test-pkg ./cmd/till/...

[note severity=normal]
Diff implementation: simple line-by-line comparison or unified diff via Go's
bufio.Scanner is acceptable. No exec to external `diff` binary — CLI must work
on Windows too (pre-cascade cross-platform concern).
```

KindPayload:
```json
{
  "changes": [
    {"file": "cmd/till/template_cli.go", "symbol": "templateSaveCommandOptions", "action": "add", "shape_hint": "struct { fromProjectID string; group string; force bool }"},
    {"file": "cmd/till/template_cli.go", "symbol": "runTemplateSave", "action": "add", "shape_hint": "func runTemplateSave(ctx, svc, cfg, opts, stdout) error — file I/O, confirm on overwrite"},
    {"file": "cmd/till/template_cli.go", "symbol": "runTemplateList", "action": "add", "shape_hint": "func runTemplateList(homeDir string, stdout io.Writer) error — scan HOME dir, list embedded defaults"},
    {"file": "cmd/till/template_cli.go", "symbol": "runTemplateShow", "action": "add", "shape_hint": "func runTemplateShow(homeDir, group, source string, stdout io.Writer) error — print file content"},
    {"file": "cmd/till/template_cli.go", "symbol": "runTemplateDiff", "action": "add", "shape_hint": "func runTemplateDiff(homeDir, group string, stdout io.Writer) error — diff HOME vs embedded"},
    {"file": "cmd/till/template_cli.go", "symbol": "runTemplateRestore", "action": "add", "shape_hint": "func runTemplateRestore(homeDir, group string, force bool, stdout io.Writer) error — copy embedded to HOME with confirm"},
    {"file": "cmd/till/template_cli_test.go", "symbol": "TestRunTemplateSave_*", "action": "add"},
    {"file": "cmd/till/template_cli_test.go", "symbol": "TestRunTemplateList_*", "action": "add"},
    {"file": "cmd/till/template_cli_test.go", "symbol": "TestRunTemplateShow_*", "action": "add"},
    {"file": "cmd/till/template_cli_test.go", "symbol": "TestRunTemplateDiff_*", "action": "add"},
    {"file": "cmd/till/template_cli_test.go", "symbol": "TestRunTemplateRestore_*", "action": "add"}
  ]
}
```

---

### W3.D5 — `till agents save/list/show/diff` CLIs

- **State:** todo
- **Kind:** build (atomic droplet; `Irreducible: true`)
- **Paths:**
  - `cmd/till/agents_cli.go` (NEW — create with 4 subcommand run functions + options structs)
  - `cmd/till/agents_cli_test.go` (NEW — create with table tests for all 4 subcommands)
- **Packages:** `cmd/till`
- **Blocked by:** W3.D4 (same cmd/till package compile)

**Specify:**

Objective: Wire 4 `till agents` subcommands (excluding bootstrap, which is D6). All perform direct OS-level file I/O to `~/.tillsyn/agents/<group>/` (HOME tier) and project-local `<project>/.tillsyn/agents/<group>/`. No service calls.

AcceptanceCriteria:
- `till agents save --from-project <id> --group <group>` reads all `*.md` files from `<project-worktree>/.tillsyn/agents/<group>/` and writes them to `~/.tillsyn/agents/<group>/`. Confirms before overwrite (or `--force`). Returns a one-line summary of files written.
- `till agents list` shows groups and agent files found in `~/.tillsyn/agents/` (HOME tier) + embedded defaults, formatted as a table (group + agent name + present in home / embedded).
- `till agents show --group <group> --agent <name> --source home` prints the content of `~/.tillsyn/agents/<group>/<name>.md`; `--source embedded` prints the embedded default.
- `till agents diff --group <group> --agent <name>` shows a diff between HOME and embedded agent content.
- Clear errors when group dir or agent file is not found at the requested source.
- Tests use `os.MkdirTemp` to isolate HOME dir; no real `~/.tillsyn` touched.
- `mage test-pkg ./cmd/till/...` passes.

RiskNotes:
- HOME tier path: `filepath.Join(os.UserHomeDir(), ".tillsyn", "agents", group, agentName+".md")`.
- Subdir-per-group is the ONLY layout (FF2 from REVISION_BRIEF §2.3). No FLAT support.
- Embedded agents: use `templates.DefaultTemplateFS.ReadFile("builtin/agents/<group>/<name>.md")` for raw-bytes access. Do NOT use `LoadDefaultTemplateForLanguage` — that returns a parsed Template struct, not raw agent bytes. Verify the actual path forms in `internal/templates/embed.go` before writing.
- Reuse the `promptConfirm` helper from D4 if it was added there; otherwise add it to `agents_cli.go`.

ContextBlocks:
```
[constraint severity=high]
File I/O only — no service calls. Subdir-per-group is the ONLY agent layout.
HOME tier: ~/.tillsyn/agents/<group>/<name>.md
Project tier: <project-worktree>/.tillsyn/agents/<group>/<name>.md

[reference severity=normal]
Embedded agent raw-bytes access: templates.DefaultTemplateFS.ReadFile("builtin/agents/<group>/<name>.md")
Path form uses canonical group names (go, fe, gen — no till- prefix per R10-D1).
Do NOT use LoadDefaultTemplateForLanguage. Verify path forms in internal/templates/embed.go.

[note severity=normal]
D5 creates agents_cli.go. D6 will ADD bootstrap to the same file. D5 MUST leave
room for D6 to append — do not structure the file in a way that blocks additions.
Exported run functions are the correct unit; cobra registration happens in D7.
```

KindPayload:
```json
{
  "changes": [
    {"file": "cmd/till/agents_cli.go", "symbol": "agentsSaveCommandOptions", "action": "add", "shape_hint": "struct { fromProjectID string; group string; force bool }"},
    {"file": "cmd/till/agents_cli.go", "symbol": "runAgentsSave", "action": "add", "shape_hint": "func runAgentsSave(ctx, svc, cfg, opts, stdout) error"},
    {"file": "cmd/till/agents_cli.go", "symbol": "runAgentsList", "action": "add", "shape_hint": "func runAgentsList(homeDir string, stdout io.Writer) error"},
    {"file": "cmd/till/agents_cli.go", "symbol": "agentsShowCommandOptions", "action": "add", "shape_hint": "struct { group string; agentName string; source string }"},
    {"file": "cmd/till/agents_cli.go", "symbol": "runAgentsShow", "action": "add", "shape_hint": "func runAgentsShow(homeDir, group, agentName, source string, stdout io.Writer) error"},
    {"file": "cmd/till/agents_cli.go", "symbol": "agentsDiffCommandOptions", "action": "add", "shape_hint": "struct { group string; agentName string }"},
    {"file": "cmd/till/agents_cli.go", "symbol": "runAgentsDiff", "action": "add", "shape_hint": "func runAgentsDiff(homeDir, group, agentName string, stdout io.Writer) error"},
    {"file": "cmd/till/agents_cli_test.go", "symbol": "TestRunAgentsSave_*", "action": "add"},
    {"file": "cmd/till/agents_cli_test.go", "symbol": "TestRunAgentsList_*", "action": "add"},
    {"file": "cmd/till/agents_cli_test.go", "symbol": "TestRunAgentsShow_*", "action": "add"},
    {"file": "cmd/till/agents_cli_test.go", "symbol": "TestRunAgentsDiff_*", "action": "add"}
  ]
}
```

---

### W3.D6 — `till agents bootstrap` CLI (2-into-4 QA fan-out)

- **State:** todo
- **Kind:** build (atomic droplet; `Irreducible: true`)
- **Paths:**
  - `cmd/till/agents_cli.go` (MODIFY — add `agentsBootstrapCommandOptions` + `runAgentsBootstrap` + helpers)
  - `cmd/till/agents_cli_test.go` (MODIFY — add `TestRunAgentsBootstrap*` table tests)
- **Packages:** `cmd/till`
- **Blocked by:** W3.D5 (shares `agents_cli.go` and `agents_cli_test.go`)

**Specify:**

Objective: Wire `till agents bootstrap` — the onboarding CLI that maps `~/.claude/agents/<group>-<role>-agent.md` files to `~/.tillsyn/agents/<group>/<role>-agent.md`. Implements 2-into-4 QA fan-out: `<group>-qa-proof-agent.md` seeds BOTH `<group>/plan-qa-proof-agent.md` AND `<group>/build-qa-proof-agent.md`. Same for qa-falsification. Supports `--dry-run`, `--force`, `--to` override. Reports missing files, generates `orchestrator-managed.md` starter when absent from source.

AcceptanceCriteria:
- `till agents bootstrap --from ~/.claude/agents --dry-run` prints a copy plan (table of source→dest mappings) without writing any files.
- `till agents bootstrap --from ~/.claude/agents` copies agent files to `~/.tillsyn/agents/<group>/` per the mapping rules; writes each dest file; reports how many files written.
- 2-into-4 QA fan-out: `go-qa-proof-agent.md` in source produces BOTH `go/plan-qa-proof-agent.md` AND `go/build-qa-proof-agent.md` at dest with identical content.
- 2-into-4 fan-out applies equally to `qa-falsification`: `go-qa-falsification-agent.md` in source produces BOTH `go/plan-qa-falsification-agent.md` AND `go/build-qa-falsification-agent.md` at dest with identical content.
- Group-agnostic agent files (no `<group>-` prefix, e.g. `closeout-agent.md`) are copied to each known group's dir (`go/`, `fe/`, `gen/`).
- Missing files are reported to stdout (e.g. "Missing: go-planning-agent.md (not found in source)").
- `orchestrator-managed.md` is absent from `~/.claude/agents/` — bootstrap writes a 1-paragraph starter at each known group's `orchestrator-managed.md`.
- `--force` overwrites existing destination files without confirmation.
- `--force` flag Long help text MUST contain VERBATIM: "Overwrites destination files; any post-bootstrap customization is lost. Use `till agents save --from-project <id>` to push customization back to HOME tier before re-running bootstrap with `--force`"
- Without `--force`, bootstrap SKIPS existing destination files and reports the skip.
- `--to <path>` overrides the default `~/.tillsyn/agents/` destination base.
- Tests cover: dry-run (assert no files written), actual copy (assert files present at dest), 2-into-4 fan-out (verify all 4 QA destinations written: plan-qa-proof-agent.md, build-qa-proof-agent.md, plan-qa-falsification-agent.md, build-qa-falsification-agent.md), force-overwrite (verify existing file replaced), skip-without-force (verify existing file untouched), missing-file reporting, orchestrator-managed.md starter generation.
- `mage test-pkg ./cmd/till/...` passes.

RiskNotes:
- **R3-NIT6 (hard rule from L1 PLAN.md):** the `--force` help text MUST include the verbatim warning quoted above. The string ends with the closing backtick on `` `--force` `` — NO trailing period. Build-QA will verify character-exactness. Do not paraphrase.
- The bootstrap logic has 4 code blocks. Stay within 120 LOC production code (excluding test file). If bootstrap's `runAgentsBootstrap` alone exceeds 80 LOC, extract helpers: `buildBootstrapPlan(sourceDir string) ([]copyOp, []string, error)` and `executeBootstrapPlan(ops []copyOp, destBase string, force, dryRun bool, stdout io.Writer) error`.
- Known groups for fan-out: `[]string{"go", "fe", "gen"}`. Hard-coded in this droplet; future-extendable via a separate config. YAGNI prevents config abstraction now.
- `orchestrator-managed.md` starter content: a 1-paragraph stub. Suggested: `"# Orchestrator-Managed Roles\n\nThis agent handles closeout, refinement, discussion, and human-verify kinds.\nSee ORCH-MANAGED-R1 for the planned split in Drop 4c.8.\n"`. Builder may adjust phrasing but must keep it a single-paragraph stub.

ContextBlocks:
```
[constraint severity=critical]
--force flag help text MUST include VERBATIM (exact string match, case-sensitive):
  "Overwrites destination files; any post-bootstrap customization is lost. Use
   `till agents save --from-project <id>` to push customization back to HOME tier
   before re-running bootstrap with `--force`"
The string ends with the closing backtick — NO trailing period after the backtick.
Build-QA will check for this exact string.

[constraint severity=high]
2-into-4 QA fan-out rule:
  Source: <group>-qa-proof-agent.md  →  Dest: <group>/plan-qa-proof-agent.md
                                              <group>/build-qa-proof-agent.md  (same content)
  Source: <group>-qa-falsification-agent.md  →  Dest: <group>/plan-qa-falsification-agent.md
                                                       <group>/build-qa-falsification-agent.md  (same content)
QA-SPLIT-R1 tracks proper per-role differentiation for Drop 4c.8; for now, both
destination files receive identical content.

[constraint severity=high]
Source file naming pattern: <group>-<role>-agent.md where group ∈ {go, fe, gen} and
role maps to the dest filename. Non-prefixed files (closeout-agent.md, etc.) are
group-agnostic and are copied to ALL known group dirs.

[reference severity=normal]
Known groups for bootstrap: []string{"go", "fe", "gen"} — hard-coded per YAGNI.

[note severity=normal]
orchestrator-managed.md starter: if source dir does NOT contain a file that maps to
orchestrator-managed.md for a group, bootstrap writes a 1-paragraph stub at dest.
Stub content is a single short paragraph explaining the placeholder role.
```

KindPayload:
```json
{
  "changes": [
    {"file": "cmd/till/agents_cli.go", "symbol": "agentsBootstrapCommandOptions", "action": "add", "shape_hint": "struct { from string; to string; dryRun bool; force bool }"},
    {"file": "cmd/till/agents_cli.go", "symbol": "runAgentsBootstrap", "action": "add", "shape_hint": "func runAgentsBootstrap(opts agentsBootstrapCommandOptions, stdout io.Writer) error"},
    {"file": "cmd/till/agents_cli.go", "symbol": "buildBootstrapPlan", "action": "add", "shape_hint": "func buildBootstrapPlan(sourceDir string) ([]bootstrapCopyOp, []string, error) — new, not yet in tree"},
    {"file": "cmd/till/agents_cli.go", "symbol": "bootstrapCopyOp", "action": "add", "shape_hint": "struct { srcPath string; destPath string; isFanOut bool } — new, not yet in tree"},
    {"file": "cmd/till/agents_cli_test.go", "symbol": "TestRunAgentsBootstrap_DryRun", "action": "add"},
    {"file": "cmd/till/agents_cli_test.go", "symbol": "TestRunAgentsBootstrap_ActualCopy", "action": "add"},
    {"file": "cmd/till/agents_cli_test.go", "symbol": "TestRunAgentsBootstrap_QAFanOut", "action": "add", "shape_hint": "verifies all 4 fan-out destination files written with identical-to-source content: <group>/plan-qa-proof-agent.md, <group>/build-qa-proof-agent.md, <group>/plan-qa-falsification-agent.md, <group>/build-qa-falsification-agent.md"},
    {"file": "cmd/till/agents_cli_test.go", "symbol": "TestRunAgentsBootstrap_ForceOverwrite", "action": "add"},
    {"file": "cmd/till/agents_cli_test.go", "symbol": "TestRunAgentsBootstrap_SkipWithoutForce", "action": "add"},
    {"file": "cmd/till/agents_cli_test.go", "symbol": "TestRunAgentsBootstrap_MissingFileReport", "action": "add"},
    {"file": "cmd/till/agents_cli_test.go", "symbol": "TestRunAgentsBootstrap_OrchestratorManagedStarter", "action": "add"},
    {"file": "cmd/till/agents_cli_test.go", "symbol": "TestRunAgentsBootstrap_ForceHelpTextContainsWarning", "action": "add", "shape_hint": "verifies --force cobra flag help text contains the R3-NIT6 verbatim string (strings.Contains check, no trailing period after closing backtick): 'Overwrites destination files; any post-bootstrap customization is lost. Use `till agents save --from-project <id>` to push customization back to HOME tier before re-running bootstrap with `--force`'"}
  ]
}
```

---

### W3.D7 — `main.go` command registration for all W3 commands

- **State:** todo
- **Kind:** build (atomic droplet; `Irreducible: true`)
- **Paths:**
  - `cmd/till/main.go` (MODIFY — add cobra command trees for all W3 subcommands)
  - `cmd/till/main_test.go` (MODIFY — add smoke tests verifying new commands are registered)
- **Packages:** `cmd/till`
- **Blocked by:** W3.D6 (all W3 command files must exist before main.go wires them; D6 is last)

**Specify:**

Objective: Register all W3 cobra commands in `main.go`. This is pure wiring — creates cobra `Command` structs, binds flags, calls the corresponding `runXxx` functions from D1–D6. No logic beyond cobra plumbing. Adds `projectCmd.AddCommand` calls for update/delete/archive/restore/rename, `actionItemCmd.AddCommand` for create, and new top-level `templateCmd` + `agentsCmd` trees.

AcceptanceCriteria:
- `till project update --help` shows correct flags (--project-id, --root-path, --bare-root, --language, --description, --hylla-artifact-ref, --build-tool, --dev-mcp-server-name, --owner, --icon, --color, --homepage, --tags, --add-group, --remove-group).
- `till project delete --help` shows --project-id, --confirm.
- `till project archive --help`, `till project restore --help`, `till project rename --help` each show their flags.
- `till action_item create --help` shows all flags including --structural-type with smart-default documented.
- `till template --help` lists save/list/show/diff/restore.
- `till agents --help` lists save/list/show/diff/bootstrap.
- `till agents bootstrap --help` shows --from, --to, --dry-run, --force with the R3-NIT6 verbatim warning (no trailing period) in the --force description.
- `mage test-pkg ./cmd/till/...` passes. `mage ci` green.

RiskNotes:
- `main.go` is ~4,069 LOC (163KB file size — the largest file in cmd/till). Use LSP `goToDefinition` on existing `projectCmd` / `actionItemCmd` / `rootCmd` to locate insertion points. Do NOT grep-and-guess. Do NOT restructure existing cobra trees.
- The new `templateCmd` and `agentsCmd` top-level commands must be registered under the root cobra command. Find the root command's `AddCommand` calls via LSP or search for `rootCmd.AddCommand`.
- D7 is registration-only. If the builder finds a bug in D1–D6's run functions while writing D7, do NOT fix it here — file a note in BUILDER_WORKLOG.md and route to the orchestrator.
- `run()` in `main.go` is the top-level entry point (`func run(ctx context.Context, args []string, stdout, stderr io.Writer) error`). All cobra commands are wired inside `run()`.
- If D7's cobra registrations + test exceed 120 LOC production code, split: D7a covers project + action_item subcommand registrations; D7b covers template + agents subcommand registrations + smoke test. D7b blocked_by D7a since both touch `main.go`.

ContextBlocks:
```
[constraint severity=high]
main.go is ~4,069 LOC (163KB). Use LSP (goToDefinition, workspace symbols) to locate:
  - existing projectCmd cobra command (to AddCommand update/delete/archive/restore/rename)
  - existing actionItemCmd cobra command (to AddCommand create)
  - rootCmd (to AddCommand templateCmd + agentsCmd)
Do NOT grep-and-guess. Do NOT restructure existing cobra trees.

[constraint severity=high]
D7 is wiring-only. No logic. If a bug surfaces in D1–D6's run functions, document
it in BUILDER_WORKLOG.md and return to orchestrator. Do NOT fix in D7.

[reference severity=normal]
Existing registration pattern to follow: find how projectCmd wires existing
subcommands (list, create, show, discover) — copy that exact pattern for
update/delete/archive/restore/rename. Same for actionItemCmd for create.

[note severity=normal]
templateCmd and agentsCmd are NEW top-level commands (not sub-commands of an
existing command). They register directly under rootCmd. Find rootCmd.AddCommand
calls in main.go to locate the insertion point.
```

KindPayload:
```json
{
  "changes": [
    {"file": "cmd/till/main.go", "symbol": "projectUpdateCmd", "action": "add", "shape_hint": "cobra.Command for till project update — flags bound including --add-group/--remove-group, Use: project update"},
    {"file": "cmd/till/main.go", "symbol": "projectDeleteCmd", "action": "add", "shape_hint": "cobra.Command for till project delete — --confirm flag"},
    {"file": "cmd/till/main.go", "symbol": "projectArchiveCmd", "action": "add"},
    {"file": "cmd/till/main.go", "symbol": "projectRestoreCmd", "action": "add"},
    {"file": "cmd/till/main.go", "symbol": "projectRenameCmd", "action": "add", "shape_hint": "cobra.Command for till project rename — --name flag"},
    {"file": "cmd/till/main.go", "symbol": "actionItemCreateCmd", "action": "add", "shape_hint": "cobra.Command for till action_item create — all flags including --structural-type"},
    {"file": "cmd/till/main.go", "symbol": "templateCmd", "action": "add", "shape_hint": "top-level cobra.Command for till template — groups save/list/show/diff/restore"},
    {"file": "cmd/till/main.go", "symbol": "agentsCmd", "action": "add", "shape_hint": "top-level cobra.Command for till agents — groups save/list/show/diff/bootstrap"},
    {"file": "cmd/till/main_test.go", "symbol": "TestW3CommandsRegistered", "action": "add", "shape_hint": "smoke test: run(['till', '--help'], ...) and check all new command names appear; new, not yet in tree"}
  ]
}
```

---

## Blocked_by Summary

```
W3.D1  blocked_by:  (none within W3; W3 wave itself blocked_by W2 + W1)
W3.D2  blocked_by:  W3.D1   (shares project_cli.go)
W3.D3  blocked_by:  W3.D2   (same cmd/till package compile; D2 must settle before D3 extends action_item_cli.go)
W3.D4  blocked_by:  W3.D3   (same cmd/till package compile; D4 creates template_cli.go)
W3.D5  blocked_by:  W3.D4   (same cmd/till package compile; D5 creates agents_cli.go)
W3.D6  blocked_by:  W3.D5   (shares agents_cli.go which D5 creates)
W3.D7  blocked_by:  W3.D6   (all command files must exist before main.go registers them)
```

Execution order: D1 → D2 → D3 → D4 → D5 → D6 → D7 (fully serial; no parallelism within W3).

---

## _BLOCKERS.toml companion

Per WORKFLOW.md: a `_BLOCKERS.toml` is created when the dir has >1 immediate child.
This dir has 7 children (D1–D7), so `_BLOCKERS.toml` is required. No changes to
`_BLOCKERS.toml` from round 1 — the blocker graph is structurally sound; round-2
changes are all spec-content absorptions, not structural.

```toml
# _BLOCKERS.toml — DROP_4c.6.1.W3_CLI_SURFACE/
# Immediate-children sibling blocker ledger.
# Mirrors inline Blocked by: bullets in PLAN.md; PLAN.md is truth.

[[blockers]]
node = "W3.D2"
blocked_by = ["W3.D1"]
reason = "shares cmd/till/project_cli.go"

[[blockers]]
node = "W3.D3"
blocked_by = ["W3.D2"]
reason = "same cmd/till package compile; D2 must settle before D3 extends action_item_cli.go"

[[blockers]]
node = "W3.D4"
blocked_by = ["W3.D3"]
reason = "same cmd/till package compile; D4 creates template_cli.go"

[[blockers]]
node = "W3.D5"
blocked_by = ["W3.D4"]
reason = "same cmd/till package compile; D5 creates agents_cli.go"

[[blockers]]
node = "W3.D6"
blocked_by = ["W3.D5"]
reason = "shares agents_cli.go which D5 creates"

[[blockers]]
node = "W3.D7"
blocked_by = ["W3.D6"]
reason = "all W3 command files must exist before main.go registers them"
```
