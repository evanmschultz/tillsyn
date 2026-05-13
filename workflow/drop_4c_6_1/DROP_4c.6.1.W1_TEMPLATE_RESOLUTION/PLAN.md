# Drop 4c.6.1.W1 — Template Resolution (HOME Tier + Group-Aware Resolver)

**State:** planning
**Wave:** B (blocked by W4.D1)
**Blocks:** W2, W3
**Directory:** `workflow/drop_4c_6_1/DROP_4c.6.1.W1_TEMPLATE_RESOLUTION/`
**Packages:** `internal/app`, `internal/domain`, `internal/app/dispatcher/cli_claude/render`
**Paths:** `internal/app/service.go`, `internal/domain/project.go`, `internal/app/dispatcher/cli_claude/render/render.go`

---

## Round 2 Changes (round-2 planner absorption — 2026-05-12)

Absorbed all round-1 plan-QA findings per spawn directive. Changes by finding:

- **Proof FF1** (orphan "per U1" in `_BLOCKERS.toml`): RESOLVED pre-round-2 by orchestrator. No further action.
- **Proof FF2** (D1/D2 testability-seam contradiction): ABSORBED. Pinned ONE seam: D1 ships `loadProjectTemplateWithHome(project *domain.Project, homeDir, group string)` (new, not yet in tree) — accepts explicit homeDir and group. `loadProjectTemplate` calls it with `os.UserHomeDir()` + `project.Language`. D2 calls it per group from `Groups`. D1's KindPayload + acceptance + RiskNotes rewritten to use only this design. D2's RiskNotes updated to reference only this design. Terminology unified across both droplets.
- **Proof NIT1** (D2 test file hedge): ABSORBED. Pinned `internal/domain/project_test.go` (confirmed to exist, 11.6K).
- **Proof NIT2** (`mergeTemplates` per-field semantics): ABSORBED. All 8 `templates.Template` fields enumerated in D2's KindPayload + acceptance (see D2). Handled together with Fals FF3.
- **Proof NIT3** (terminology unification): ABSORBED — resolved by FF2 fix. Single term `loadProjectTemplateWithHome` used throughout D1 and D2.
- **Proof NIT4** (summary table `*_test.go` glob): ABSORBED. Summary table rows spell out explicit filenames.
- **Fals FF1** (till-prefix drift / `agentBodyDefaultGroup`): ABSORBED per R10-D1. D3 scope EXPANDS to update `agentBodyDefaultGroup` from `"till-go"` → `"go"` AND `agentBodyFallbackGroup` from `"till-gen"` → `"gen"` in `render.go`. D3's KindPayload, acceptance, and RiskNotes updated. D3's `blocked_by W4.D1` already captures the ordering constraint (W4.D1 renames embedded FS dirs first, then D3's constant updates are safe).
- **Fals FF2** (W2.D7/W3.D1 stale on Groups): D2 acceptance reaffirmed with explicit routing note. W2/W3 PLAN.md updates are out of W1 scope — their round-2 planners absorb this.
- **Fals FF3** (`mergeTemplates` under-specified): ABSORBED. D2 acceptance + KindPayload enumerate per-field merge strategy for all 9 `templates.Template` fields. Refinement `MERGE-FIELD-AXIS-R1` raised for future revisit on non-AgentBindings fields.
- **Fals NIT1** (D2 signature contradiction `loadProjectTemplatesForGroups`): ABSORBED — resolved by FF2 fix.
- **Fals NIT2** (D2 "calls `loadProjectTemplate` per group" implicit signature change): ABSORBED — D2 calls `loadProjectTemplateWithHome` (D1 seam) directly, no implicit signature change.
- **Fals NIT3** (`omitempty` for `[]string`): ABSORBED — doc-comment note added to D2's field acceptance: `omitempty` on `[]string` omits BOTH `nil` AND empty-non-nil slices (per Go `encoding/json` semantics — `len(s) == 0` is treated as empty for slices). Only non-empty slices like `["go","fe"]` appear in marshaled output.
- **Fals NIT4** ("same author" language in D2 RiskNotes): DEFERRED-AS-NIT — `blocked_by` is the structural enforcement; author-identity prose is informal. Non-load-bearing; doesn't change builder behavior.
- **Fals NIT5** (render_test.go size vague): ABSORBED — updated D3 RiskNote from "~64K" to "1661 lines" (confirmed via `ls`).
- **Fals NIT6** (orphan "per U1"): RESOLVED — same as Proof FF1.
- **Fals NIT7** (D3 KindPayload missing `agentBodyDefaultGroup` rename): ABSORBED — folded into FF1 resolution. D3's KindPayload now includes both constant renames.
- **Fals NIT8** (D2 "additive field" claim lacks round-trip test): ABSORBED — added explicit round-trip marshal/unmarshal test bullet to D2 acceptance.
- **Fals NIT9** (whitespace-only Language): DEFERRED-AS-NIT — D1 acceptance already specifies `strings.TrimSpace(project.Language)`; whitespace-only is handled. Concern refuted.
- **Fals NIT10** (D3 "one call site" not a test gate): DEFERRED-AS-NIT — D3 RiskNote instructs builder to use LSP findReferences. Process note, not a code defect.
- **R10-D1 expansion**: D3 acceptance explicitly enumerates `agentBodyDefaultGroup` + `agentBodyFallbackGroup` constant renames and test fixture updates. Added warning ContextBlock: D3 must dispatch AFTER W4.D1 renames embedded FS dirs.
- **R10-D2 reaffirmation**: D2 acceptance reaffirmed as shipping typed `ProjectMetadata.Groups []string` field; downstream W2/W3 consume the typed field directly.
- **Empty-group guard**: added to D2's RiskNotes — coordinator must skip or error on empty string in `Groups` slice.

---

## Planner Section

### Scope

This wave adds the user HOME tier to the 3-tier template resolution chain in
`bakeProjectKindCatalog` and updates the group-aware agent body resolver's
project-tier lookup from flat to subdir-per-group.

**What ships in W1:**

1. **D1** — HOME-tier walk for single-group projects. `loadProjectTemplate` gains a
   tier-3 candidate: `~/.tillsyn/templates/<group>.toml`. Derived via
   `os.UserHomeDir()` + `filepath.Join(home, ".tillsyn", "templates", group+".toml")`.
   Group is `strings.TrimSpace(project.Language)` for single-group projects. D1 also
   ships `loadProjectTemplateWithHome(project *domain.Project, homeDir, group string)`
   as a package-private helper enabling test injection of fake home + explicit group.

2. **D2** — Multi-group aggregation. `domain.ProjectMetadata` gains a
   `Groups []string` field. `bakeProjectKindCatalog` + `loadProjectTemplate` gain a
   multi-group coordinator `loadProjectTemplatesForGroups`: when
   `project.Metadata.Groups` is non-empty, the HOME tier walks each declared group
   and aggregates bindings + child_rules via `mergeTemplates`. D2 serializes
   after D1 (shares `service.go` + adds `internal/domain/project.go`).

3. **D3** — `render.go:assembleAgentFileBody` tier-1 update. `readProjectTierAgent`
   changes from flat `<project>/.tillsyn/agents/<basename>` to subdir-per-group
   `<project>/.tillsyn/agents/<group>/<basename>`. D3 also updates
   `agentBodyDefaultGroup` from `"till-go"` to `"go"` and `agentBodyFallbackGroup`
   from `"till-gen"` to `"gen"` (R10-D1). D3 runs parallel to D1+D2 since it
   touches a different package (`internal/app/dispatcher/cli_claude/render`).
   D3 dispatches only after W4.D1 renames embedded FS dirs.

**What does NOT ship in W1:**

- No `till template save` / `till template list` CLIs (W3).
- No changes to `till init` (W2).
- No `agents.toml` schema changes (W0).
- No `agents.local.toml` merge (W0).

---

### Specify Block

**Objective:**
Extend the template resolution chain in `bakeProjectKindCatalog` from 3-tier to 4-tier
by inserting a user HOME tier (`~/.tillsyn/templates/<group>.toml`) between the project
worktree tier and the embedded fallback. For multi-group projects, walk the HOME tier for
each declared group and aggregate bindings + child_rules. Update the agent body resolver's
project-tier lookup from flat to subdir-per-group so dispatched agents read from
`<project>/.tillsyn/agents/<group>/<basename>` rather than
`<project>/.tillsyn/agents/<basename>`. Update `agentBodyDefaultGroup` from `"till-go"`
to `"go"` and `agentBodyFallbackGroup` from `"till-gen"` to `"gen"` (R10-D1).

**AcceptanceCriteria:**

1. `loadProjectTemplate` walks the 4-tier chain: bare-root → primary-worktree →
   `~/.tillsyn/templates/<group>.toml` → embedded. First-candidate-wins semantics
   preserved. Error on candidate-load failure propagates; does NOT fall through to next
   tier.

2. When a `~/.tillsyn/templates/<group>.toml` file exists, `bakeProjectKindCatalog`
   uses it in preference to the embedded default (tier-3 wins over tier-4).

3. `domain.ProjectMetadata` carries a `Groups []string` field with JSON tag
   `"groups,omitempty"`. When non-empty, `bakeProjectKindCatalog` walks the HOME tier
   for each group in `Groups` and aggregates bindings + child_rules. W2.D7 and W3.D1
   MUST consume this typed field directly (not `KindPayload` JSON). The orchestrator
   must update W2.D7 + W3.D1 PLAN.md before dispatching those droplets.

4. `readProjectTierAgent` looks up `<project>/.tillsyn/agents/<group>/<basename>`,
   NOT `<project>/.tillsyn/agents/<basename>` (flat removed). `assembleAgentFileBody`
   passes `group` to `readProjectTierAgent`.

5. `agentBodyDefaultGroup` = `"go"` and `agentBodyFallbackGroup` = `"gen"` in `render.go`.
   Existing tests that reference `"till-go"` / `"till-gen"` as group names in test
   fixtures are updated to `"go"` / `"gen"`. This makes the dispatched agent's
   project-tier path match `<project>/.tillsyn/agents/go/<basename>` when
   `binding.SystemPromptTemplatePath` is empty (the W3-FF5 LOCKED default branch).

6. Cross-group fallback to `gen` group is preserved in the embedded tier
   (`readEmbeddedTierAgent` — reads `builtin/agents/gen/<basename>` after W4.D1
   renames the embedded dir; untouched by W1 beyond the constant rename).

7. `mage test-pkg ./internal/app` passes (≥70% coverage on new code paths).
   `mage test-pkg ./internal/domain` passes.
   `mage test-pkg ./internal/app/dispatcher/cli_claude/render` passes.
   `mage ci` green.

**ValidationPlan:**

- AC1–AC2: `mage test-pkg ./internal/app` — new table-driven tests in `service_test.go`
  cover: (a) HOME-tier file exists → used; (b) HOME-tier file absent → embedded fallback;
  (c) HOME-tier file malformed → error propagates; (d) empty `RepoPrimaryWorktree` +
  empty `RepoBareRoot` + no HOME file → embedded default.
- AC3: `mage test-pkg ./internal/app` — multi-group test: project with
  `Groups = ["go", "fe"]` walks HOME tier for both; aggregated bindings contain entries
  from both groups. `mage test-pkg ./internal/domain` — `ProjectMetadata` marshal/unmarshal
  round-trip confirms `Groups` field round-trips correctly.
- AC4: `mage test-pkg ./internal/app/dispatcher/cli_claude/render` — existing render
  tests updated to expect subdir-per-group path; new test: project tier returns miss on
  flat file but hit on `<group>/<basename>` subdir layout.
- AC5: updated render tests verify `agentBodyDefaultGroup = "go"` and
  `agentBodyFallbackGroup = "gen"` via path-resolution smoke.
- AC6: existing `readEmbeddedTierAgent` tests pass after constant renames + W4.D1's
  embedded dir rename (D3 dispatches only after W4.D1 complete).
- AC7: `mage ci` full run after D1 + D2 + D3 all land.

**RiskNotes:**

- **Aggregation shape for multi-group.** `templates.Template` has 9 fields. Per-field
  merge strategy for `mergeTemplates(base, overlay templates.Template) templates.Template`
  (new, not yet in tree — D2 ships this in `service.go`):
  - `SchemaVersion string`: last-group-wins (overlay overwrites base).
  - `Kinds map[domain.Kind]KindRule`: per-key last-group-wins.
  - `ChildRules []ChildRule`: append (concat base + overlay); dedup on
    `(WhenParentKind, CreateChildKind)` tuple, overlay entry wins on collision.
  - `AgentBindings map[domain.Kind]AgentBinding`: per-key last-group-wins (primary use).
  - `Agents map[domain.Kind]AgentRuntime`: per-key last-group-wins.
  - `Gates map[domain.Kind][]GateKind`: per-key last-group-wins (overlay slice replaces
    base slice for same kind; NOT concat).
  - `GateRulesRaw map[string]any`: per-key shallow merge, last-group-wins on collision.
  - `Tillsyn Tillsyn` struct: whole-struct last-group-wins; overlay `Tillsyn` replaces
    base entirely if overlay is non-zero (any of `MaxContextBundleChars != 0 ||
    MaxAggregatorDuration != 0 || SpawnTempRoot != ""`).
  - `StewardSeeds []StewardSeed`: append (concat base + overlay, no dedup).
  - Refinement `MERGE-FIELD-AXIS-R1` raised: revisit per-field semantics for
    `Tillsyn`, `StewardSeeds`, `Gates`, `GateRulesRaw`, `ChildRules`, `Kinds`,
    `Agents` when multi-group projects start exercising these fields in dogfood.
- **`loadProjectTemplateWithHome` seam (D1/D2 interface contract).** D1 ships
  `loadProjectTemplateWithHome(project *domain.Project, homeDir, group string)
  (templates.Template, bool, error)` — a package-private helper that accepts explicit
  homeDir and group. `loadProjectTemplate` calls it with `os.UserHomeDir()` result
  (skip HOME tier if `os.UserHomeDir()` fails) and `strings.TrimSpace(project.Language)`
  as group. D2 calls it per element in `project.Metadata.Groups`. This is the SINGLE
  testability seam for both droplets. Symbol `loadProjectTemplateWithHome` is new,
  not yet in tree.
- **Empty group guard.** `loadProjectTemplatesForGroups` must guard against empty-string
  entries in `project.Metadata.Groups` (e.g., skip entries where
  `strings.TrimSpace(group) == ""`). Bare empty string would produce a malformed HOME
  tier path (`~/.tillsyn/templates/.toml`).
- **`domain.ProjectMetadata.Groups` field addition.** Adding a new JSON field to
  `ProjectMetadata` is additive (zero-value `[]string` is nil → omitempty skips it on
  marshal). Note: Per Go `encoding/json` semantics, `omitempty` on `[]string` omits
  BOTH `nil` AND empty-non-nil slices (`len(s) == 0` is treated as empty for slices).
  So both `Groups: nil` AND `Groups: []string{}` are omitted; only non-empty slices
  like `Groups: ["go","fe"]` appear in marshaled output. The zero value of `Groups`
  in Go is nil, so freshly created projects have nil Groups, which marshal correctly.
  No migration needed. Builder searches callers via LSP `findReferences` on
  `ProjectMetadata` before wiring.
- **`readProjectTierAgent` signature change.** Adding `group` parameter changes the
  function signature. It is package-private; confirmed one call site today
  (`assembleAgentFileBody` at render.go:666). Builder uses LSP `findReferences` on
  `readProjectTierAgent` to confirm before editing.
- **`agentBodyFallbackGroup` rename dependency.** After D3 sets
  `agentBodyFallbackGroup = "gen"`, `readEmbeddedTierAgent` reads from
  `builtin/agents/gen/<basename>` in the embedded FS. This ONLY works after W4.D1
  renames `till-gen/` → `gen/` in the embedded FS via `git mv`. The `blocked_by W4.D1`
  constraint already serializes D3 after W4.D1. Builder must not dispatch D3 before
  W4.D1 is complete.
- **HOME path when `os.UserHomeDir()` fails.** Consistent with the `readUserTierAgent`
  pattern: if `os.UserHomeDir()` fails, skip the HOME tier silently. Builder mirrors
  the exact pattern from `readUserTierAgent` (render.go:899–900).
- **W4.D1 dependency.** W1 is `blocked_by` W4.D1 because canonical group names
  (`go`, `fe`, `gen` without `till-` prefix) must be confirmed before D1/D2/D3
  embed them as path segments or update constants. Tests MUST use confirmed group
  names from W4.D1 in fixture paths.

**ContextBlocks:**

- `constraint` (critical): W4.D1 MUST be `complete` before D1, D2, or D3 are
  dispatched. The HOME tier path is `~/.tillsyn/templates/<group>.toml` where
  `<group>` is derived from `project.Language` or `project.Metadata.Groups`. D3's
  constant renames depend on W4.D1's embedded FS dir renames being in place.
  Tests must use the W4.D1-confirmed bare names (`go`, `gen`, `fe`).
- `constraint` (high): D1 and D2 share `internal/app/service.go`. D2 is `blocked_by`
  D1. D3 shares no files with D1/D2 and runs in parallel after W4.D1.
- `constraint` (high): D2 adds `internal/domain/project.go` (adding `Groups []string`
  to `ProjectMetadata`). This adds `internal/domain` as a W1 package. D2's packages
  declaration includes `internal/domain`. No other W1 droplet touches `internal/domain`.
- `decision` (normal): HOME-tier path is derived via `os.UserHomeDir()` (same pattern
  as `readUserTierAgent`), NOT via `platform.DefaultPathsWithOptions`. The `platform`
  package's `Paths` struct has no `TemplatesDir` field. Consistent with the agent-body
  resolver's approach.
- `decision` (normal): Multi-group aggregation uses per-field merge semantics as
  enumerated in RiskNotes above. `AgentBindings` is the primary use case; other fields
  use last-group-wins or append. Builder documents the merge strategy in
  `mergeTemplates`'s doc-comment.
- `decision` (normal): `domain.ProjectMetadata.Groups` is `[]string` with
  `json:"groups,omitempty"`. No TOML tag needed (confirmed: `ProjectMetadata` uses
  `json:` tags primarily; some fields have both `json:` and `toml:` tags — builder
  follows existing struct convention).
- `decision` (normal): `loadProjectTemplateWithHome(project, homeDir, group string)`
  is the canonical testability seam for the HOME-tier extension. Both D1 and D2 use
  this symbol. D1 ships it; D2 calls it per group. This decision supersedes both
  the "walk-loop fake path" approach (D1 round-1 RiskNotes) and the two-helper
  approach (D2 round-1 RiskNotes).
- `warning` (critical): D3 updates `agentBodyFallbackGroup` from `"till-gen"` to
  `"gen"`. This is safe ONLY after W4.D1 renames `builtin/agents/till-gen/` →
  `builtin/agents/gen/` in the embedded FS. Dispatching D3 before W4.D1 completes
  will break `readEmbeddedTierAgent`'s cross-group fallback for ALL dispatches.
  `blocked_by W4.D1` is the guard.
- `warning` (high): ALL existing render tests that set up fake project worktrees with
  flat agent files AND tests that reference `"till-go"` / `"till-gen"` as group name
  literals MUST be updated in D3. Builder reads `render_test.go` (1661 lines) in full
  before editing to enumerate all affected test fixtures.
- `reference` (normal): `readUserTierAgent` in `render.go` (line ~898) — HOME-tier
  pattern to mirror in `loadProjectTemplate`. `assembleAgentFileBody` (line ~646) —
  call site that threads `group` to `readProjectTierAgent`.
- `note` (low): The aggregated `templates.Template` returned by multi-group coordinator
  is still a single `templates.Template` (same return type). Aggregation merges per
  field as documented. Later templates in the group list win on key collision for
  map-type fields.

**KindPayload (D1):**
```json
{"changes":[
  {"file":"internal/app/service.go","symbol":"loadProjectTemplate","action":"modify","shape_hint":"extend walk-loop to add HOME-tier candidate: filepath.Join(homeDir, '.tillsyn', 'templates', group+'.toml'); homeDir obtained from os.UserHomeDir() + skip on error; delegates to loadProjectTemplateWithHome"},
  {"file":"internal/app/service.go","symbol":"loadProjectTemplateWithHome","action":"add","shape_hint":"new, not yet in tree — package-private; signature: (project *domain.Project, homeDir, group string) (templates.Template, bool, error); runs 4-tier walk: bare-root + primary-worktree candidates (existing), then HOME-tier filepath.Join(homeDir, '.tillsyn', 'templates', group+'.toml'), then embedded fallback via templates.LoadDefaultTemplateForLanguage; loadProjectTemplate calls it with os.UserHomeDir() result and strings.TrimSpace(project.Language)"},
  {"file":"internal/app/service_test.go","symbol":"TestLoadProjectTemplate_HomeTier","action":"add","shape_hint":"new, not yet in tree — table-driven: HOME file exists→used; HOME file absent→embedded fallback; HOME file malformed→error propagates; empty-worktree-paths+no HOME file→embedded default; uses t.TempDir() as fake homeDir passed to loadProjectTemplateWithHome directly"}
]}
```

**KindPayload (D2):**
```json
{"changes":[
  {"file":"internal/domain/project.go","symbol":"ProjectMetadata.Groups","action":"modify","shape_hint":"add Groups []string field with json tag 'groups,omitempty'; doc-comment: per Go encoding/json semantics, omitempty omits BOTH nil AND empty-non-nil slices (len(s)==0 is empty for slices); zero value is nil"},
  {"file":"internal/app/service.go","symbol":"loadProjectTemplatesForGroups","action":"add","shape_hint":"new, not yet in tree — package-private; signature: (project *domain.Project, homeDir string) (templates.Template, bool, error); iterates project.Metadata.Groups, calls loadProjectTemplateWithHome per group (guarding against empty-string groups), merges results via mergeTemplates; bakeProjectKindCatalog calls this when Groups non-empty"},
  {"file":"internal/app/service.go","symbol":"mergeTemplates","action":"add","shape_hint":"new, not yet in tree — package-private; signature: (base, overlay templates.Template) templates.Template; per-field merge: SchemaVersion=last-wins; Kinds=per-key last-wins; ChildRules=append+dedup on (WhenParentKind,CreateChildKind); AgentBindings=per-key last-wins; Agents=per-key last-wins; Gates=per-key last-wins (slice replaces, no concat); GateRulesRaw=per-key shallow merge; Tillsyn=whole-struct last-wins if overlay non-zero; StewardSeeds=append no-dedup; doc-comment enumerates all 9 fields"},
  {"file":"internal/app/service.go","symbol":"bakeProjectKindCatalog","action":"modify","shape_hint":"when project.Metadata.Groups non-empty, call loadProjectTemplatesForGroups(project, homeDir); else call loadProjectTemplate(project) as before; homeDir obtained from os.UserHomeDir() at bakeProjectKindCatalog call site"},
  {"file":"internal/domain/project_test.go","symbol":"TestProjectMetadata_Groups_RoundTrip","action":"add","shape_hint":"new, not yet in tree — marshal/unmarshal round-trip: Groups=nil→omitted; Groups=['go','fe']→present; Groups=[]string{}→present (empty non-nil NOT omitted); verify JSON output matches expected"},
  {"file":"internal/app/service_test.go","symbol":"TestBakeProjectKindCatalog_MultiGroup","action":"add","shape_hint":"new, not yet in tree — 2 groups, both HOME files present→aggregated bindings contain entries from both groups; 1 group HOME absent→absent group uses embedded fallback; collision on same kind key→last group wins; empty-string in Groups→skipped without error"}
]}
```

**KindPayload (D3):**
```json
{"changes":[
  {"file":"internal/app/dispatcher/cli_claude/render/render.go","symbol":"agentBodyDefaultGroup","action":"modify","shape_hint":"change constant value from 'till-go' to 'go' (R10-D1); safe after W4.D1 renames embedded builtin/agents/till-go/ → builtin/agents/go/"},
  {"file":"internal/app/dispatcher/cli_claude/render/render.go","symbol":"agentBodyFallbackGroup","action":"modify","shape_hint":"change constant value from 'till-gen' to 'gen' (R10-D1); safe after W4.D1 renames embedded builtin/agents/till-gen/ → builtin/agents/gen/"},
  {"file":"internal/app/dispatcher/cli_claude/render/render.go","symbol":"readProjectTierAgent","action":"modify","shape_hint":"add group string parameter; signature changes from (projectWorktree, basename string) to (projectWorktree, group, basename string); path changes from filepath.Join(worktree, projectAgentsSubdir, basename) to filepath.Join(worktree, projectAgentsSubdir, group, basename)"},
  {"file":"internal/app/dispatcher/cli_claude/render/render.go","symbol":"assembleAgentFileBody","action":"modify","shape_hint":"pass already-resolved group variable to readProjectTierAgent call (group is resolved at line 663 via resolveAgentGroup; thread it through to readProjectTierAgent)"},
  {"file":"internal/app/dispatcher/cli_claude/render/render_test.go","symbol":"TestReadProjectTierAgent_SubdirPerGroup","action":"add","shape_hint":"new test: project tier returns miss on flat file but hit on <group>/<basename> subdir layout; update ALL existing tests that reference 'till-go'/'till-gen' group name literals and ALL tests that set up fake project worktrees with flat agent files to use subdir-per-group layout and bare group names (go, gen, fe)"}
]}
```

**CompletionContract:**

- StartCriteria: W4.D1 is `complete` (confirmed group names, embedded FS dirs renamed,
  subdir-per-group layout established).
- CompletionCriteria:
  - D1 complete: `loadProjectTemplate` HOME tier implemented, `loadProjectTemplateWithHome`
    seam added, tests pass.
  - D2 complete: `ProjectMetadata.Groups` added, multi-group coordinator + `mergeTemplates`
    with all-8-field spec implemented, tests pass; `internal/domain` + `internal/app`
    both green.
  - D3 complete: `readProjectTierAgent` subdir-per-group, call site updated,
    `agentBodyDefaultGroup` + `agentBodyFallbackGroup` constants updated, all affected
    tests updated, tests pass.
  - All three complete: `mage ci` green.
- CompletionChecklist:
  - [ ] D1 `mage test-pkg ./internal/app` green
  - [ ] D2 `mage test-pkg ./internal/domain` green
  - [ ] D2 `mage test-pkg ./internal/app` green (multi-group path covered)
  - [ ] D3 `mage test-pkg ./internal/app/dispatcher/cli_claude/render` green
  - [ ] `mage ci` green post-D1+D2+D3

---

### Droplets

#### D1 — HOME Tier in `loadProjectTemplate` (Single-Group)

- **Kind:** `build`
- **Irreducible:** true
- **State:** todo
- **Blocked by:** W4.D1 (group names must be confirmed before test fixtures and HOME
  tier path segments use them)
- **Paths:**
  - `internal/app/service.go`
  - `internal/app/service_test.go`
- **Packages:** `internal/app`
- **Acceptance:**
  1. `loadProjectTemplate` has a 4-tier resolution: bare-root → primary-worktree →
     HOME (`~/.tillsyn/templates/<group>.toml`) → embedded default.
  2. `group` for HOME tier = `strings.TrimSpace(project.Language)` when non-empty;
     empty Language skips the HOME tier candidate (no path to read).
  3. First-candidate-wins: if HOME file exists + parses OK, embedded is not consulted.
  4. HOME file exists but `templates.Load` errors → error propagates (same contract as
     existing tier-1/tier-2 error propagation).
  5. `os.UserHomeDir()` failure → HOME tier silently skipped (consistent with
     `readUserTierAgent` pattern in render.go:899–900).
  6. `loadProjectTemplateWithHome(project *domain.Project, homeDir, group string)` is
     added as a package-private helper that accepts explicit homeDir and group for test
     injection. `loadProjectTemplate` calls it with `os.UserHomeDir()` result and
     `strings.TrimSpace(project.Language)`.
  7. `mage test-pkg ./internal/app` passes. New `TestLoadProjectTemplate_HomeTier`
     covers all four cases listed in KindPayload: (a) HOME file exists; (b) HOME file
     absent; (c) HOME file malformed; (d) empty-worktree-paths + no HOME file.
- **Specify:**
  - **Objective:** Insert `~/.tillsyn/templates/<group>.toml` as tier-3 candidate in
    `loadProjectTemplate`. Single-group only (uses `project.Language` as group). Add
    `loadProjectTemplateWithHome` as the testability seam D2 will consume.
  - **RiskNotes:**
    - `loadProjectTemplateWithHome` is the testability seam: tests call it directly
      with a `t.TempDir()` fake homeDir + the fixture group name. The published
      `loadProjectTemplate` delegates to it with `os.UserHomeDir()`. Do NOT add the
      `platform` package import to `service.go` — `platform.Paths` has no
      `TemplatesDir` field. Direct `os.UserHomeDir()` is the canonical approach.
    - Symbol `loadProjectTemplateWithHome` is new, not yet in tree — D1 authors it.
      D2 consumes it. D2 is `blocked_by` D1.
  - **ContextBlocks:**
    - `constraint` (high): D2 is `blocked_by` D1. D1 must be `complete` (merged,
      test-passing) before D2 is dispatched.
    - `reference` (normal): `loadProjectTemplateCandidate` (service.go ~581) — the
      existing per-candidate reader; D1 adds a third call to this helper in
      `loadProjectTemplateWithHome`'s walk loop.
    - `reference` (normal): `readUserTierAgent` (render.go ~898) — the HOME-tier
      pattern to mirror: call `os.UserHomeDir()`, skip on error, derive path, read.
  - **ValidationPlan:** `mage test-pkg ./internal/app`

---

#### D2 — Multi-Group Aggregation in `bakeProjectKindCatalog`

- **Kind:** `build`
- **Irreducible:** true
- **State:** todo
- **Blocked by:** D1 (shares `service.go`; D1 must land first to avoid compile
  conflict; also D2 consumes `loadProjectTemplateWithHome` which D1 ships)
- **Paths:**
  - `internal/domain/project.go`
  - `internal/domain/project_test.go`
  - `internal/app/service.go`
  - `internal/app/service_test.go`
- **Packages:** `internal/domain`, `internal/app`
- **Acceptance:**
  1. `domain.ProjectMetadata.Groups` field exists: `Groups []string` with JSON tag
     `json:"groups,omitempty"`. Per Go `encoding/json`, `omitempty` on `[]string`
     omits BOTH `nil` AND empty-non-nil slices (`len(s) == 0` is treated as empty
     for slices). Only non-empty slices appear in marshaled output. Zero value is
     nil. Additive field — existing `ProjectMetadata` marshal/unmarshal round-trips
     are unaffected. Builder confirms no full-struct-literal comparisons on
     `ProjectMetadata` break via LSP `findReferences`.
  2. `ProjectMetadata.Groups` marshal/unmarshal round-trip test in `project_test.go`
     confirms: `Groups=nil`→omitted; `Groups=["go","fe"]`→present; `Groups=[]string{}`→
     omitted (both nil and len==0 slices are omitted per Go encoding/json semantics).
  3. New `loadProjectTemplatesForGroups(project *domain.Project, homeDir string)`
     helper in `service.go` (package-private). Calls `loadProjectTemplateWithHome`
     (shipped by D1) with each non-empty group in `project.Metadata.Groups`. Guards
     against empty-string entries in `Groups` (skips entries where
     `strings.TrimSpace(group) == ""`). Merges resulting `templates.Template` values
     via `mergeTemplates`.
  4. New `mergeTemplates(base, overlay templates.Template) templates.Template` helper
     in `service.go` (package-private). Per-field merge strategy:
     - `SchemaVersion`: last-group-wins (overlay overwrites base).
     - `Kinds`: per-key last-group-wins.
     - `ChildRules`: append base + overlay; dedup on `(WhenParentKind, CreateChildKind)`
       tuple, overlay entry wins on collision.
     - `AgentBindings`: per-key last-group-wins (primary multi-group use case).
     - `Agents`: per-key last-group-wins.
     - `Gates`: per-key last-group-wins (overlay slice replaces base slice for same kind;
       NOT concat).
     - `GateRulesRaw`: per-key shallow merge, last-group-wins on collision.
     - `Tillsyn`: whole-struct last-group-wins; overlay `Tillsyn` replaces base if
       overlay is non-zero (`MaxContextBundleChars != 0 || MaxAggregatorDuration != 0 ||
       SpawnTempRoot != ""`).
     - `StewardSeeds`: append base + overlay (no dedup; seeds are project-unique).
     - Doc-comment on `mergeTemplates` enumerates all 9 fields.
  5. `bakeProjectKindCatalog` branches: if `project.Metadata.Groups` is non-empty, call
     `loadProjectTemplatesForGroups`; else call `loadProjectTemplate` (existing path).
  6. W2.D7 and W3.D1 MUST consume `project.Metadata.Groups` typed field directly.
     They MUST NOT use `KindPayload` JSON stopgap. The orchestrator updates W2 + W3
     PLAN.md before dispatching those droplets. (W2-GROUPS-R1 refinement RESOLVED
     inline by this droplet.)
  7. `mage test-pkg ./internal/domain` passes.
  8. `mage test-pkg ./internal/app` passes. New `TestBakeProjectKindCatalog_MultiGroup`
     covers: (a) 2 groups, both HOME files present → aggregated bindings contain entries
     from both groups; (b) 2 groups, one HOME file absent → absent group uses embedded
     fallback; (c) collision on same kind key → last group wins; (d) empty-string entry
     in `Groups` → skipped without error.
- **Specify:**
  - **Objective:** Add `Groups []string` to `domain.ProjectMetadata` and wire a
    multi-group aggregator in `bakeProjectKindCatalog` that walks the HOME tier per
    group and merges templates with explicit per-field semantics for all 8
    `templates.Template` fields.
  - **RiskNotes:**
    - Template merge strategy: `templates.Template` has 9 fields (confirmed via
      `internal/templates/schema.go`). Builder implements `mergeTemplates` per the
      per-field semantics enumerated in acceptance #4 above. No `templates.Merge`
      function exists today (searched schema.go — not present). Symbol `mergeTemplates`
      is new, not yet in tree.
    - `loadProjectTemplatesForGroups` calls `loadProjectTemplateWithHome` (D1-shipped
      seam) per group. The `homeDir` param enables test injection of fake home (same
      approach as D1). Symbol `loadProjectTemplatesForGroups` is new, not yet in tree.
    - Empty-group guard: coordinator skips entries where
      `strings.TrimSpace(group) == ""` to avoid malformed HOME tier paths
      (`~/.tillsyn/templates/.toml`).
    - Adding `Groups []string` to `domain.ProjectMetadata` is additive (nil = unset).
      Builder searches `ProjectMetadata` callers via LSP `findReferences` to confirm no
      full-struct-literal comparisons break. Known safe: `ProjectMetadata` is JSON-
      decoded from DB; field-literal comparisons in test code use named fields.
    - D2 RiskNotes describe `loadProjectTemplateWithHome` as the D1-shipped seam.
      Builder MUST NOT dispatch D2 before D1 is complete. The `blocked_by D1` entry
      is the structural guard.
    - Refinement `MERGE-FIELD-AXIS-R1`: revisit per-field semantics for `Tillsyn`,
      `StewardSeeds`, `Gates`, `GateRulesRaw`, `ChildRules`, `Kinds`, `Agents` when
      multi-group projects start exercising these fields in dogfood. Pre-MVP,
      last-group-wins / append is the pragmatic default.
  - **ContextBlocks:**
    - `constraint` (high): D2 must not dispatch until D1 is `complete`. They share
      `service.go`; parallel execution would create a compile conflict. D2 also
      consumes `loadProjectTemplateWithHome` which D1 ships.
    - `constraint` (high): D2 adds `internal/domain` as a package in this wave. D2's
      `blocked_by` includes D1. No other droplet in W1 touches `internal/domain`.
    - `decision` (normal): last-group-wins for `AgentBindings` key collisions. This is
      an intentional choice (later groups in the selection list override earlier ones).
      Builder documents this in the `mergeTemplates` doc-comment.
    - `reference` (normal): `templates.Template` struct shape in `schema.go` (lines
      150–248) — builder reads this before writing the merge logic. 9 fields confirmed.
    - `note` (low): W2.D7 was previously planned to write `groups` into
      `Metadata.KindPayload` as JSON (pre-typed-field state). After D2 ships, W2.D7
      MUST write `Metadata.Groups = payload.Groups` directly. W3.D1 MUST wire
      `--add-group/--remove-group` against `Metadata.Groups`. Orchestrator must update
      W2 + W3 before dispatch.
  - **ValidationPlan:** `mage test-pkg ./internal/domain` + `mage test-pkg ./internal/app`

---

#### D3 — Subdir-Per-Group Project-Tier Resolver + Constant Renames (`render.go`)

- **Kind:** `build`
- **Irreducible:** true
- **State:** todo
- **Blocked by:** W4.D1 (confirmed subdir-per-group layout must exist in the embedded
  agent tree AND embedded dir names renamed before the resolver's path shape + constant
  values are final)
- **Paths:**
  - `internal/app/dispatcher/cli_claude/render/render.go`
  - `internal/app/dispatcher/cli_claude/render/render_test.go`
- **Packages:** `internal/app/dispatcher/cli_claude/render`
- **Acceptance:**
  1. `readProjectTierAgent` signature changes from `(projectWorktree, basename string)`
     to `(projectWorktree, group, basename string)`. Function body: path becomes
     `filepath.Join(projectWorktree, projectAgentsSubdir, group, basename)`.
  2. `assembleAgentFileBody` passes `group` (already resolved via `resolveAgentGroup`
     at line 663) to `readProjectTierAgent`.
  3. `agentBodyDefaultGroup` constant value changes from `"till-go"` to `"go"` (R10-D1).
  4. `agentBodyFallbackGroup` constant value changes from `"till-gen"` to `"gen"` (R10-D1).
  5. A project with a flat `<project>/.tillsyn/agents/builder-agent.md` (old layout)
     results in a MISS at the project tier (falling through to user/embedded tier).
     A project with `<project>/.tillsyn/agents/go/builder-agent.md` (new layout)
     results in a HIT.
  6. `resolveAgentGroup` continues to return `agentBodyDefaultGroup` (now `"go"`) when
     `binding.SystemPromptTemplatePath` is empty — project-tier path for the LOCKED
     default branch now resolves to `<project>/.tillsyn/agents/go/<basename>`, which
     matches `till init --group go` output (W2.D5 produces this layout).
  7. `readEmbeddedTierAgent`'s cross-group fallback to `agentBodyFallbackGroup` (now
     `"gen"`) reads from `builtin/agents/gen/<basename>` — correct after W4.D1's
     `git mv till-gen → gen`. Existing cross-group fallback tests pass after updating
     fixture group name references.
  8. ALL existing render tests that reference `"till-go"` / `"till-gen"` as group name
     literals and ALL tests that set up fake project worktrees with flat agent files are
     updated to use bare group names (`go`, `gen`) and subdir-per-group layout.
  9. `mage test-pkg ./internal/app/dispatcher/cli_claude/render` passes with no
     regressions. New test `TestReadProjectTierAgent_SubdirPerGroup` added.
- **Specify:**
  - **Objective:** Three changes to `render.go` in one atomic droplet: (1) change
    `readProjectTierAgent` from flat project-agent lookup to subdir-per-group lookup;
    (2) update `agentBodyDefaultGroup` to `"go"`; (3) update `agentBodyFallbackGroup`
    to `"gen"`. Update `render_test.go` accordingly.
  - **RiskNotes:**
    - `readProjectTierAgent` is package-private; confirmed one call site today
      (`assembleAgentFileBody` at render.go:666). Builder uses LSP `findReferences`
      on `readProjectTierAgent` to confirm before editing.
    - `agentBodyFallbackGroup` rename to `"gen"` is safe ONLY after W4.D1 completes
      `git mv till-gen → gen` in the embedded FS. D3 `blocked_by W4.D1` is the guard.
      Builder MUST verify W4.D1 is in `complete` state before proceeding.
    - `group` is already resolved at the `assembleAgentFileBody` level via
      `resolveAgentGroup(binding)` (render.go:863). Builder simply passes that already-
      resolved `group` variable down to `readProjectTierAgent`. No new group-resolution
      logic needed in D3.
    - Existing render tests (1661 lines in `render_test.go`) use fake project worktrees
      with flat agent file layouts AND reference `"till-go"` / `"till-gen"` string
      literals. ALL must be updated to subdir-per-group layout and bare group names.
      Builder reads `render_test.go` in full before editing to enumerate all affected
      fixtures. This is the highest test-churn risk in D3.
    - `const projectAgentsSubdir = ".tillsyn/agents"` stays unchanged. The group
      segment is appended between `projectAgentsSubdir` and `basename` by the call to
      `filepath.Join(worktree, projectAgentsSubdir, group, basename)`.
  - **ContextBlocks:**
    - `constraint` (critical): D3 must not dispatch until W4.D1 is `complete`. Both
      the subdir-per-group path shape AND the constant renames depend on W4.D1's
      embedded FS renames being in place.
    - `constraint` (high): D3 runs in parallel with D1 and D2 (different package,
      no shared files). The only ordering constraint is `blocked_by W4.D1`.
    - `decision` (normal): `const projectAgentsSubdir` stays `".tillsyn/agents"` — the
      group subdir is interpolated at call time, not in the constant. Matches the
      user-tier pattern where `userAgentsSubdir` is `".tillsyn/agents"` and `group` is
      appended separately.
    - `decision` (normal): Three logically related changes (subdir-per-group path,
      `agentBodyDefaultGroup` rename, `agentBodyFallbackGroup` rename) are bundled in
      one droplet because they share a single dispatch dependency (W4.D1) and a single
      test-update pass in `render_test.go`. Splitting would require two sequential
      D3a/D3b droplets with the same `blocked_by W4.D1` and overlapping test files —
      no net benefit.
    - `reference` (normal): `readUserTierAgent` (render.go ~898) uses `group` as a
      segment already: `filepath.Join(home, userAgentsSubdir, group, basename)`. D3
      makes the project tier consistent with the user tier.
    - `warning` (critical): ALL existing render tests that set up fake project
      worktrees with flat agent files WILL FAIL after D3. Builder must update every
      test fixture that populates `<tmpdir>/.tillsyn/agents/<basename>` to instead
      populate `<tmpdir>/.tillsyn/agents/<group>/<basename>`. Tests referencing
      `"till-go"` or `"till-gen"` string literals must change to `"go"` or `"gen"`.
      Full `render_test.go` read (1661 lines) is mandatory before editing.
  - **ValidationPlan:** `mage test-pkg ./internal/app/dispatcher/cli_claude/render`

---

### Blocked-by Graph

```
W4.D1 (Wave A)
   │
   ├──→ D1 (HOME tier, single-group) ─→ D2 (multi-group aggregation)
   │
   └──→ D3 (subdir-per-group resolver + constant renames)  [parallel to D1+D2]
```

- D1: `blocked_by W4.D1`
- D2: `blocked_by D1` (shared `service.go` compile lock; D2 consumes D1's seam)
- D3: `blocked_by W4.D1` (test fixtures use confirmed group names; constant renames
  depend on embedded FS dir renames)
- D3 runs parallel to D1 and D2 (different package, no shared files).

**Build order:** W4.D1 completes → D1 and D3 dispatch in parallel → D2 dispatches after D1 completes.

---

### Summary

| Droplet | Paths | Packages | Blocked By | Parallel With |
|---------|-------|----------|-----------|---------------|
| D1 | `service.go`, `service_test.go` | `internal/app` | W4.D1 | D3 |
| D2 | `project.go`, `project_test.go`, `service.go`, `service_test.go` | `internal/domain`, `internal/app` | D1 | — |
| D3 | `render.go`, `render_test.go` | `internal/app/dispatcher/cli_claude/render` | W4.D1 | D1 |

Total: **3 atomic droplets**.

---

### Mage Verification Targets

| Droplet | Per-droplet target | Post-wave target |
|---------|--------------------|-----------------|
| D1 | `mage test-pkg ./internal/app` | — |
| D2 | `mage test-pkg ./internal/domain` + `mage test-pkg ./internal/app` | — |
| D3 | `mage test-pkg ./internal/app/dispatcher/cli_claude/render` | `mage ci` |

`mage ci` runs after all three droplets are `done`.

---

### Build Discipline (Builders Read This)

- Never `go test`, `go build`, `go vet`, `go run`, `mage install`. Always `mage <target>`.
- Single-line conventional commits ≤72 chars. No body. No co-authored-by trailers.
- Spawn subagents with `run_in_background: true` by default.
- Do NOT call `hylla_ingest`. Reingest is drop-end only, orchestrator-run.
