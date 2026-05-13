# Drop 4c.6.1 Builder Worklog

## W7.D2 — EXTRACT EVERYTHING-NOT-HTTP (2026-05-12)

**Droplet**: 4c.6.1.W7.D2 — Extract all non-HTTP-residue from `internal/adapters/server/` to new packages

**Files changed**:

New packages (created via `git mv` + new file):
- `internal/adapters/mcp_common/` — moved from `internal/adapters/server/common/`; package renamed `common` → `mcpcommon`; new file `server_config.go` adds `Config`, `Dependencies`, `NormalizeConfig`, `NormalizeEndpoint`
- `internal/adapters/mcp_rpc/` — moved from `internal/adapters/server/mcpapi/`; package renamed `mcpapi` → `mcprpc`; all intra-package `common.` imports updated to `mcpcommon.`
- `internal/adapters/mcp_stdio/stdio.go` — NEW file; `RunStdio` thin wrapper over `mcprpc.ServeStdio`

Modified:
- `internal/adapters/server/server.go` — stripped to HTTP-residue only (`Run`, `NewHandler`, `writeHealthStatus`); imports `mcp_common` for `Config`/`Dependencies`/`NormalizeConfig`; imports `mcp_rpc` for `mcprpc.NewHandler`
- `internal/adapters/server/httpapi/handler.go` — updated import from `server/common` → `mcp_common`; `common.` → `mcpcommon.`
- `internal/adapters/server/httpapi/handler_test.go` — same import update
- `internal/adapters/server/httpapi/handler_integration_test.go` — import alias updated `servercommon → mcp_common`
- `cmd/till/main.go` — `servercommon` import replaced with `mcpcommon "...mcp_common"` + `mcpstdio "...mcp_stdio"`; `serveradapter.Config/Dependencies` → `mcpcommon.Config/Dependencies`; `serveradapter.RunStdio` → `mcpstdio.RunStdio`; `serveradapter.Run` stays
- `cmd/till/main_test.go` — `servercommon` replaced with `mcpcommon`; `serveradapter.Config/Dependencies` → `mcpcommon.Config/Dependencies`; `serveradapter` unused import removed

**File-split decisions**:
- `server.go` — split at function level: `RunStdio`/`normalizeConfig`/`normalizeEndpoint`/`Config`/`Dependencies` extracted; HTTP-only (`Run`, `NewHandler`, `writeHealthStatus`, `defaultShutdownTimeout`) remain
- `mcpapi/handler.go` (now `mcp_rpc/handler.go`) — extracted whole; `NewHandler`/`Handler` HTTP wrapper stays in `mcp_rpc/` for W7.D3 deletion per inventory §5 recommendation
- `Config`/`Dependencies` placed in `mcp_common/server_config.go` (not `mcp_stdio/`) so HTTP residue can import them without a back-reference; `NormalizeConfig`/`NormalizeEndpoint` exported there

**Test results**:
- `mage test-pkg ./internal/adapters/mcp_common`: 165 passed
- `mage test-pkg ./internal/adapters/mcp_rpc`: 226 passed
- `mage test-pkg ./internal/adapters/mcp_stdio`: 0 tests (compiles clean)
- `mage test-pkg ./cmd/till`: 281 passed
- `mage ci`: 3164/3164 passed, 30 packages, coverage all >= 70%

## W0 — Config Decoder Multi-Group (2026-05-12)

**Droplet**: 4c.6.1.W0 — Update `internal/config/agents.go` for multi-group schema

**Files changed**:
- `internal/config/agents.go` — full rewrite of multi-group decoder
- `internal/config/agents_test.go` — full rewrite of tests for new multi-group API
- `internal/config/testdata/agents/multigroup_single.toml` — new golden fixture
- `internal/config/testdata/agents/multigroup_go_fe.toml` — new golden fixture
- `internal/config/testdata/agents/multigroup_kind_override.toml` — new golden fixture
- `internal/config/testdata/agents/multigroup_local_override.toml` — new golden fixture
- `internal/config/testdata/agents/multigroup_tools_deny_rejected.toml` — new golden fixture

**What was done**:

Rewrote `internal/config/agents.go` from the old single-group `[agents]` / `[agents.<kind>]` schema to the new multi-group `[<group>]` / `[<group>.<kind>]` schema.

Key type changes:
- Added `GroupConfig` struct: `{Default Preset; Kinds map[domain.Kind]Override}`
- Changed `AgentsRegistry` to `map[string]GroupConfig` (was `{Preset, Overrides map[domain.Kind]Override, Path}`)
- Removed `AgentRuntime` type (Resolve now returns `Preset` directly — same fields, simpler)
- Removed old `LoadRegistry` (single-group `[agents]` format loader)
- Added `LoadMultiGroupRegistry(path string) (AgentsRegistry, error)` — new multi-group loader
- Changed `Resolve` signature from `Resolve(registry *AgentsRegistry, kind domain.Kind) (AgentRuntime, error)` to `Resolve(registry AgentsRegistry, group, kind string) (Preset, error)`
- Changed `MergeLocal` to `Merge(local, project AgentsRegistry) (AgentsRegistry, error)`

TOML decode strategy: reused the existing `agentsTOMLGroupBlock` struct (renamed from `agentsTOMLBlock`) as the value type in a `map[string]agentsTOMLGroupBlock` root decode. This lets `DisallowUnknownFields()` reject typos in kind names within each group block, while map keys (group names) are user-defined and not validated.

`Merge` parameter order: `Merge(local, project)` — local wins. The tools_deny rejection logic moved into `rejectLocalToolsDeny(local AgentsRegistry)` helper which iterates all groups deterministically (sorted group names + deterministicKindOrder for per-kind iteration) before any merge work.

**Test results**: `mage test-pkg ./internal/config` — 77/77 PASS

**Pre-existing failures in other packages**: The render package (`internal/app/dispatcher/cli_claude/render`) has 19 pre-existing test failures due to W4.D1's template restructuring (till-go/go-*.md files deleted, till-go/ → go/ rename). These failures pre-date W0 and are out of scope.

**No production callers updated**: `Resolve` and `MergeLocal`/`Merge` had no production callers in any package outside `internal/config`. The render package uses only `config.StripFrontmatterKeys` (in `frontmatter.go` — unaffected).

---

## W4.D1 — Restructure embedded agent dirs (subdir rename + orphan deletion + qa split + fe group + embed.go) (2026-05-12)

**Droplet**: 4c.6.1.W4.D1 — Embedded agent dir restructure (Wave A head, no blockers)

**Files changed** (primary scope `internal/templates`):
- `internal/templates/builtin/agents/till-go/` → `go/` via `git mv` (history preserved)
- `internal/templates/builtin/agents/till-gen/` → `gen/` via `git mv` (history preserved)
- `internal/templates/builtin/agents/go/go-builder-agent.md` — DELETED via `git rm -f`
- `internal/templates/builtin/agents/go/go-planning-agent.md` — DELETED via `git rm -f`
- `internal/templates/builtin/agents/go/go-qa-proof-agent.md` — DELETED via `git rm -f`
- `internal/templates/builtin/agents/go/go-qa-falsification-agent.md` — DELETED via `git rm -f`
- `internal/templates/builtin/agents/go/go-research-agent.md` — DELETED via `git rm -f`
- `internal/templates/builtin/agents/go/qa-proof-agent.md` — DELETED via `git rm -f` (replaced by split)
- `internal/templates/builtin/agents/go/qa-falsification-agent.md` — DELETED via `git rm -f` (replaced by split)
- `internal/templates/builtin/agents/gen/qa-proof-agent.md` — DELETED via `git rm -f` (replaced by split)
- `internal/templates/builtin/agents/gen/qa-falsification-agent.md` — DELETED via `git rm -f` (replaced by split)
- `internal/templates/builtin/agents/go/plan-qa-proof-agent.md` — NEW (split from qa-proof-agent.md)
- `internal/templates/builtin/agents/go/build-qa-proof-agent.md` — NEW
- `internal/templates/builtin/agents/go/plan-qa-falsification-agent.md` — NEW
- `internal/templates/builtin/agents/go/build-qa-falsification-agent.md` — NEW
- `internal/templates/builtin/agents/go/orchestrator-managed.md` — NEW (was only in gen/)
- `internal/templates/builtin/agents/gen/plan-qa-proof-agent.md` — NEW
- `internal/templates/builtin/agents/gen/build-qa-proof-agent.md` — NEW
- `internal/templates/builtin/agents/gen/plan-qa-falsification-agent.md` — NEW
- `internal/templates/builtin/agents/gen/build-qa-falsification-agent.md` — NEW
- `internal/templates/builtin/agents/fe/` — NEW directory with 10 placeholder files
- `internal/templates/embed.go` — updated `//go:embed` explicit per-file list (new canonical paths)
- `internal/templates/embed_test.go` — updated FS-introspection test to use new 10-file group standard
- `internal/templates/load.go` — updated `embeddedAgentGroups` from `{till-gen, till-go, till-gdd}` to `{gen, go, fe, till-gdd}` + doc-comment updates
- `internal/templates/load_test.go` — updated inline TOML fixtures `go-builder-agent` → `builder-agent`, `go-planning-agent` → `planning-agent`; updated `TestLoadValidatesAgentBindingNamesDefaultLookupPermissivePreW1D1` body per LOUD WARNING; updated warn-test assertions
- `internal/templates/context_rules_test.go` — same agent name replacement
- `internal/templates/testdata/valid_minimal.toml` — `go-builder-agent` → `builder-agent`

**Scope expansion (necessary to satisfy `mage ci` green acceptance criterion)**:
- `internal/app/dispatcher/cli_claude/render/render.go` — updated `agentBodyDefaultGroup` `"till-go"` → `"go"` and `agentBodyFallbackGroup` `"till-gen"` → `"gen"` (W4.D1's rename makes the old constants immediately incorrect)
- `internal/app/dispatcher/cli_claude/render/render_test.go` — updated agent name fixtures, group paths, cross-group fallback test to use canonical names
- `internal/app/dispatcher/dispatcher_test.go` — updated `buildRichFixture` AgentName fixture
- `internal/app/dispatcher/spawn_test.go` — updated `goBuilderBinding()` and assertions
- `internal/app/template_service_test.go` — updated `minimalValidTemplateTOML` agent_name
- `cmd/till/init_cmd.go` — updated `allowedInitGroups` and `initTUIGroupRows` from `till-gen/till-go` to `gen/go`
- `cmd/till/init_cmd_test.go` — updated all `"till-go"`/`"till-gen"` group references in JSON payloads, assertions, and TUI test expectations
- `cmd/till/dispatcher_cli_test.go` — updated `goBuilderBinding()` agent name fixtures and assertions

**What was done**:

1. `git mv internal/templates/builtin/agents/till-go internal/templates/builtin/agents/go`
2. `git mv internal/templates/builtin/agents/till-gen internal/templates/builtin/agents/gen`
3. `git rm -f` on 5 orphan `go-*` files and 4 old monolithic QA files (2 per group)
4. Created 10 new placeholder files (4 QA split + 1 orchestrator-managed) per group (go/ + gen/)
5. Created `fe/` with 10 placeholder files (same 10 standard names)
6. Updated `embed.go` `//go:embed` explicit list for all canonical paths (gen/ + go/ + fe/ + till-gdd/)
7. Updated `embed_test.go` with new `w4d1StandardAgentNames` (10 names) and `w4d1CanonicalGroups` (gen/go/fe)
8. Updated `load.go`'s `embeddedAgentGroups` = `{"gen", "go", "fe", "till-gdd"}` so `validateAgentBindingNames` resolves against new paths
9. Updated `render.go`'s `agentBodyDefaultGroup` = `"go"` and `agentBodyFallbackGroup` = `"gen"`
10. Updated all downstream test fixtures using `"go-builder-agent"` → `"builder-agent"`, `"till-go"` group paths → `"go"`
11. Updated `init_cmd.go`'s `allowedInitGroups` = `["gen", "go"]` and `initTUIGroupRows` = `{gen, go, till-gdd}`

**Test results**:
- `mage test-pkg ./internal/templates` — 474/474 PASS
- `mage ci` — GREEN (3155/3155 PASS; all coverage thresholds met)

**Acceptance criteria verified**:
- `agents/` contains exactly `go/`, `gen/`, `fe/`, `till-gdd/` (4 subdirs). No `till-go/`. No `till-gen/`. ✓
- `go/` = 10 files (10 canonical names, no `go-*` orphans, no old `qa-proof-agent.md`). ✓
- `gen/` = 10 files (same 10 names). `orchestrator-managed.md` kept. ✓
- `fe/` = 10 placeholder files (same 10 names). ✓
- `till-gdd/` = 7 files (unchanged). ✓
- All new placeholder files have `name:` + `description:` frontmatter ONLY. ✓
- `//go:embed` explicit per-file (NOT glob). ✓
- `mage ci` green. ✓

**Scope expansion rationale**: W1.D3 was planned to update `agentBodyDefaultGroup`/`agentBodyFallbackGroup` constants in `render.go` (`blocked_by W4.D1`). Since W4.D1's rename removes `till-go/` and `till-gen/` from the embedded FS immediately, `render.go`'s old constants cause `mage ci` failures. Updating them here (W4.D1) rather than deferring to W1.D3 satisfies the acceptance criterion `mage ci` green without requiring W1.D3 to unblock. The orchestrator should mark W1.D3's constant-update scope as absorbed by W4.D1.

---

## W1.D3 — Subdir-Per-Group Project-Tier Resolver + Constant Renames (`render.go`) (2026-05-12)

**Droplet**: 4c.6.1.W1.D3 — `readProjectTierAgent` signature + path change, `assembleAgentFileBody` call site, render_test.go fixture updates

**Files changed**:
- `internal/app/dispatcher/cli_claude/render/render.go` — signature change + path change for `readProjectTierAgent`; call site update in `assembleAgentFileBody`; doc-comment updates
- `internal/app/dispatcher/cli_claude/render/render_test.go` — `agentTierProjectFixture` helper updated to group-subdir layout; 6 callers updated; 2 stale comments updated; `TestReadProjectTierAgent_SubdirPerGroup` added

**What was done**:

1. Verified W4.D1 pre-condition: `agentBodyDefaultGroup = "go"` and `agentBodyFallbackGroup = "gen"` were ALREADY set by W4.D1 (confirmed at lines 189 + 199 of render.go). Constant scope absorbed — no constant changes needed.

2. Changed `readProjectTierAgent` signature from `(projectWorktree, basename string)` to `(projectWorktree, group, basename string)`. Path construction changed from `filepath.Join(projectWorktree, projectAgentsSubdir, basename)` (flat) to `filepath.Join(projectWorktree, projectAgentsSubdir, group, basename)` (subdir-per-group). Added drop reference in doc-comment. Updated doc to note old flat layout no longer resolves.

3. Updated `assembleAgentFileBody` at the single call site (line 676): `readProjectTierAgent(project.RepoPrimaryWorktree, basename)` → `readProjectTierAgent(project.RepoPrimaryWorktree, group, basename)`. The `group` variable was already resolved at line 673 via `resolveAgentGroup(binding)` — no new logic needed.

4. Updated `userAgentsSubdir` doc comment (now both tiers are group-scoped; removed false "project tier is not" claim).

5. Updated `ErrAgentBodyNotFound` doc comment: `till-gen` → `gen` in two places.

6. Updated `agentTierProjectFixture` helper: added `group string` parameter; changed `dir` construction to `filepath.Join(projectDir, ".tillsyn", "agents", group)`.

7. Updated all 6 callers of `agentTierProjectFixture` to pass `"go"` group (all use `fixtureBinding()` with empty `SystemPromptTemplatePath`, so `resolveAgentGroup` returns `"go"`):
   - `TestAssembleAgentFileBody_ProjectOverride`
   - `TestRenderValidatorFailsOnTooShortBody`
   - `TestRenderValidatorFailsOnMissingFrontmatter`
   - `TestRenderValidatorFailsOnMissingMarker`
   - `TestRenderValidatorPassesOnSubstantiveBody`
   - `TestRenderValidatorAcceptsAllEmbeddedPlaceholders` (also added group derivation from embed path + `SystemPromptTemplatePath` on binding so `till-gdd`-group agents resolve correctly)

8. Updated 2 stale comments: `"till-go"` → `"go"` in the section header comment; `"till-go"` / `"till-gen"` → `"go"` / `"gen"` in `TestAssembleAgentFileBody_CrossGroupFallbackMissesBothGroups`.

9. Added `TestReadProjectTierAgent_SubdirPerGroup` with 2 subtests:
   - `flat_layout_is_miss`: seeds `<project>/.tillsyn/agents/builder-agent.md` (flat); verifies resolver misses (embedded tier fires instead)
   - `subdir_layout_is_hit`: seeds `<project>/.tillsyn/agents/go/builder-agent.md` (subdir); verifies resolver hits (project-tier sentinel wins over embedded)

**TDD cycle**:
- RED: updated `agentTierProjectFixture` and added new test before production change → 7 failures (correct RED)
- GREEN: updated `readProjectTierAgent` + `assembleAgentFileBody` → 83/83 PASS

**Test results**:
- `mage test-pkg ./internal/app/dispatcher/cli_claude/render` — 83/83 PASS (was 76 pre-D3 after Wave A)
- `mage test-func ./internal/app/dispatcher/cli_claude/render TestReadProjectTierAgent_SubdirPerGroup` — 3/3 PASS (parent + 2 subtests)
- `mage ci` — 2601/2601 test PASS; 4 PKG FAIL (`cmd/till`, `internal/adapters/mcp_rpc`, `internal/adapters/server`, `internal/adapters/server/httpapi`) are pre-existing coverage gate failures from other Wave A builders' uncommitted changes in those packages (confirmed: `0 test failures and 0 build errors across 29 packages`)

**Scope note**: Constant renames (`agentBodyDefaultGroup`, `agentBodyFallbackGroup`) were absorbed by W4.D1 per that builder's scope expansion. This droplet only executed the path-shape change + test updates.
