# Drop 4c.6.1 Builder Worklog

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
