# W1.D3 — BUILD-QA-PROOF Verdict

**Date:** 2026-05-12
**Reviewer:** go-qa-proof-agent (opus)
**Overall verdict:** PASS

## Acceptance Bullet Coverage

### AC1 — `readProjectTierAgent` signature + path change

> `readProjectTierAgent` signature changes from `(projectWorktree, basename string)` to `(projectWorktree, group, basename string)`. Function body: path becomes `filepath.Join(projectWorktree, projectAgentsSubdir, group, basename)`.

**Evidence:**
- `internal/app/dispatcher/cli_claude/render/render.go:892` — new signature `func readProjectTierAgent(projectWorktree, group, basename string) (string, bool, error)`.
- `internal/app/dispatcher/cli_claude/render/render.go:896` — `p := filepath.Join(projectWorktree, projectAgentsSubdir, group, basename)`.
- `git diff` confirms `-func readProjectTierAgent(projectWorktree, basename string)` → `+func readProjectTierAgent(projectWorktree, group, basename string)` and `-filepath.Join(projectWorktree, projectAgentsSubdir, basename)` → `+filepath.Join(projectWorktree, projectAgentsSubdir, group, basename)`.

**Verdict:** PASS.

### AC2 — `assembleAgentFileBody` threads `group` to `readProjectTierAgent`

> `assembleAgentFileBody` passes `group` (already resolved via `resolveAgentGroup` at line 663) to `readProjectTierAgent`.

**Evidence:**
- `internal/app/dispatcher/cli_claude/render/render.go:673` — `group := resolveAgentGroup(binding)` resolves once.
- `internal/app/dispatcher/cli_claude/render/render.go:676` — call site updated: `body, found, err := readProjectTierAgent(project.RepoPrimaryWorktree, group, basename)`.
- `git diff` confirms the call-site edit.
- `rg -n readProjectTierAgent internal/` returns only this one call site (plus the decl + doc-comment), confirming no other caller required updating.

**Verdict:** PASS.

### AC3 — `agentBodyDefaultGroup = "go"` (PASS-via-W4.D1)

> `agentBodyDefaultGroup` constant value changes from `"till-go"` to `"go"` (R10-D1).

**Evidence:**
- `internal/app/dispatcher/cli_claude/render/render.go:189` — `const agentBodyDefaultGroup = "go"`.
- `git log -- internal/app/dispatcher/cli_claude/render/render.go` shows commit `6f41bc8 refactor(templates): rename agent subdirs and split qa into 4 files` as the latest tip predating Wave B's uncommitted edits.
- `git diff` shows NO change to the constant line in D3's uncommitted delta — the value `"go"` was already present in the indexed (post-W4.D1) base, so D3's reduced scope correctly leaves it alone.

**Verdict:** PASS (constant already at `"go"` via W4.D1 `6f41bc8`; not D3's responsibility post-scope-reduction).

### AC4 — `agentBodyFallbackGroup = "gen"` (PASS-via-W4.D1)

> `agentBodyFallbackGroup` constant value changes from `"till-gen"` to `"gen"` (R10-D1).

**Evidence:**
- `internal/app/dispatcher/cli_claude/render/render.go:199` — `const agentBodyFallbackGroup = "gen"`.
- `git diff` shows NO constant-value change in D3's delta (only adjacent doc-comment edits for stale `till-gen` text on lines 95–101 — see NIT-cleanup observation below).
- Both constant renames landed in W4.D1 `6f41bc8` before Wave B kicked off.

**Verdict:** PASS (constant already at `"gen"` via W4.D1 `6f41bc8`; not D3's responsibility post-scope-reduction).

### AC5 — Flat layout MISS / subdir layout HIT

> A project with a flat `<project>/.tillsyn/agents/builder-agent.md` (old layout) results in a MISS at the project tier (falling through to user/embedded tier). A project with `<project>/.tillsyn/agents/go/builder-agent.md` (new layout) results in a HIT.

**Evidence:**
- `internal/app/dispatcher/cli_claude/render/render_test.go:1699` — new `TestReadProjectTierAgent_SubdirPerGroup`.
- Subtest `flat_layout_is_miss` (line 1700): seeds the OLD flat path, sets `binding.SystemPromptTemplatePath=""` (group defaults to `"go"`), asserts the rendered body does NOT contain `SENTINEL_FLAT_LAYOUT_MUST_NOT_WIN` AND DOES contain the embedded `# PLACEHOLDER` marker (proves fall-through).
- Subtest `subdir_layout_is_hit` (line 1744): seeds the NEW subdir path via `agentTierProjectFixture(t, project.RepoPrimaryWorktree, "go", "builder-agent.md", subdirBody)`, asserts the rendered body DOES contain `SENTINEL_SUBDIR_LAYOUT_WINS`.
- `mage test-func ./internal/app/dispatcher/cli_claude/render TestReadProjectTierAgent_SubdirPerGroup`: 3/3 PASS (parent + 2 subtests).

**Verdict:** PASS.

### AC6 — `resolveAgentGroup` returns `"go"` for empty `SystemPromptTemplatePath`; project-tier path becomes `.tillsyn/agents/go/<basename>`

> `resolveAgentGroup` continues to return `agentBodyDefaultGroup` (now `"go"`) when `binding.SystemPromptTemplatePath` is empty — project-tier path for the LOCKED default branch now resolves to `<project>/.tillsyn/agents/go/<basename>`...

**Evidence:**
- `internal/app/dispatcher/cli_claude/render/render.go:189` — `agentBodyDefaultGroup = "go"`.
- The new `subdir_layout_is_hit` subtest (test_render.go:1744) exercises exactly this path: empty `SystemPromptTemplatePath` → group resolves to `"go"` → seeded file at `.tillsyn/agents/go/builder-agent.md` is found → sentinel appears in output.
- Inline comment on line 1763: `// Empty SystemPromptTemplatePath → group = "go" (agentBodyDefaultGroup).` ties the behavior to the constant explicitly.

**Verdict:** PASS.

### AC7 — Cross-group fallback to `gen` group preserved in embedded tier

> `readEmbeddedTierAgent`'s cross-group fallback to `agentBodyFallbackGroup` (now `"gen"`) reads from `builtin/agents/gen/<basename>` — correct after W4.D1's `git mv till-gen → gen`. Existing cross-group fallback tests pass after updating fixture group name references.

**Evidence:**
- `TestAssembleAgentFileBody_CrossGroupFallbackMissesBothGroups` (render_test.go:~1022) — comments updated from `"till-go"`/`"till-gen"` to `"go"`/`"gen"`; test still asserts `ErrAgentBodyNotFound` when neither subdir holds the basename.
- `mage test-pkg ./internal/app/dispatcher/cli_claude/render`: 83/83 PASS — all existing fallback tests green post-rename.
- Stale-comment fix at render.go:95–101 (visible in diff) — doc-comment updated to reference `gen` (no behavior change; consistency hygiene).

**Verdict:** PASS.

### AC8 — All existing render tests updated to bare group names + subdir layout

> ALL existing render tests that reference `"till-go"` / `"till-gen"` as group name literals and ALL tests that set up fake project worktrees with flat agent files are updated to use bare group names (`go`, `gen`) and subdir-per-group layout.

**Evidence:**
- `agentTierProjectFixture` (render_test.go:819) — signature now `(t, projectDir, group, basename, content)`; body uses `filepath.Join(projectDir, ".tillsyn", "agents", group)`.
- 6 existing callers updated to pass `"go"` explicitly (render_test.go lines 942, 1481, 1519, 1547, 1579, 1767).
- 1 caller in `TestRenderValidatorAcceptsAllEmbeddedPlaceholders` (line 1662) updated to derive `group := path.Dir(relPath)` from the embed path and pass it through; this also sets `binding.SystemPromptTemplatePath = relPath` so the resolver picks up the correct group for non-default groups like `till-gdd` (preserves test correctness across renames).
- Stale comment fixes at render_test.go:794 (`"till-go"` → `"go"`) and ~1022 (`till-go`/`till-gen` → `go`/`gen`).
- `rg -n agentTierProjectFixture` confirms exactly 7 call sites + 1 helper decl + 1 doc-comment line, all consistent.

**Verdict:** PASS.

### AC9 — `mage test-pkg` green + new test added

> `mage test-pkg ./internal/app/dispatcher/cli_claude/render` passes with no regressions. New test `TestReadProjectTierAgent_SubdirPerGroup` added.

**Evidence:**
- `mage test-pkg ./internal/app/dispatcher/cli_claude/render`: `tests: 83 / passed: 83 / failed: 0` (live re-run during this QA pass).
- `mage test-func ./internal/app/dispatcher/cli_claude/render TestReadProjectTierAgent_SubdirPerGroup`: `tests: 3 / passed: 3` (parent + 2 subtests).
- Builder reports `mage ci` 3164/3164 PASS — not re-run by this reviewer (in-scope target was the per-package suite, per ValidationPlan); orchestrator-side gate-runner separately confirms.

**Verdict:** PASS.

---

## NITs

None blocking. Observations:

1. (informational, severity: low) The two subtests in `TestReadProjectTierAgent_SubdirPerGroup` cannot run `t.Parallel()` because they use `t.Setenv("HOME", ...)`. This is correctly noted in inline comments. Per Go testing semantics, that is the safe pattern; flagging only because the rest of the file generally parallelizes table-driven tests. No action needed.

2. (informational, severity: low) The doc-comment cleanups at render.go:95–101 and the comment edits at render_test.go:~794, ~1022 are technically outside D3's KindPayload, but they fix stale references to `till-gen`/`till-go` that would otherwise mislead readers post-W4.D1. Reviewer treats this as in-scope hygiene with the constant-rename absorption that W4.D1 already shipped. Not a defect.

---

## Verdict rationale

D3's reduced scope is `readProjectTierAgent` signature + path change, call-site update, fixture-helper extension, and test refactors. Every authoritative AC has on-disk evidence:

- **Signature + path (AC1, AC2):** correct on both render.go:892 (decl) and render.go:676 (call site); only one caller exists, confirmed via `rg`.
- **Constants (AC3, AC4):** already in tree from W4.D1 `6f41bc8`. D3's diff correctly does not re-rename them.
- **New test (AC5, AC9):** `TestReadProjectTierAgent_SubdirPerGroup` exercises both the flat-miss and subdir-hit cases with sentinel-based assertions; `mage test-func` green.
- **Default-group wiring (AC6):** subdir-hit subtest exercises the empty-`SystemPromptTemplatePath` → group `"go"` → `.tillsyn/agents/go/` path live.
- **Cross-group fallback (AC7):** existing fallback tests + 83/83 package pass.
- **Test fixture sweep (AC8):** 6 existing `agentTierProjectFixture` callers passed `"go"`; the more involved `TestRenderValidatorAcceptsAllEmbeddedPlaceholders` site correctly derives the group from the embed path and threads `SystemPromptTemplatePath` so non-default groups (like `till-gdd`) keep resolving via the path-based group derivation rather than misrouting to `"go"`.
- **Package green (AC9):** 83/83 pass live; new test 3/3 pass live.

Overall: PASS. Builder's claim is supported by on-disk evidence with no missing acceptance coverage.
