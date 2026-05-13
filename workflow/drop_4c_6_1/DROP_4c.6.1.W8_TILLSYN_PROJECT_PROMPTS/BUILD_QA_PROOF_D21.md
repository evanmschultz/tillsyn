# W8.D21 — BUILD-QA-PROOF Verdict

**Date:** 2026-05-13
**Reviewer:** go-qa-proof-agent (opus)
**Overall verdict:** PASS

## Scope

Droplet **W8.D21** is a test-only smoke droplet for the 3-tier prompt resolver's project-tier priority after W1.D3's subdir-per-group change. The only production-side change in scope is the new test `TestRenderProjectTierOverridesEmbeddedDefault` in `internal/app/dispatcher/cli_claude/render/render_test.go` (lines 1686-1833). No production code is modified.

Evidence basis:
- `git diff internal/app/dispatcher/cli_claude/render/render_test.go` — confirms `+149` insertions only.
- `Read` of the added test (render_test.go:1686-1833) — confirms structure.
- `Read` of production-side anchors in `render.go` (lines 312-347, 580-617, 880-927) — confirms test contract against the resolver.
- `mage test-func ./internal/app/dispatcher/cli_claude/render TestRenderProjectTierOverridesEmbeddedDefault` — 3/3 PASS (1 parent + 2 sub-cases).
- `mage test-pkg ./internal/app/dispatcher/cli_claude/render` — 86/86 PASS.

## Acceptance Bullet Coverage

### Bullet 1 — Test exists

> `TestRenderProjectTierOverridesEmbeddedDefault` exists in `render_test.go`.

**Evidence:** `render_test.go:1714` — `func TestRenderProjectTierOverridesEmbeddedDefault(t *testing.T)`.

**Verdict:** PASS.

### Bullet 2 — Table-driven, both cases

> Test is table-driven (at minimum one case: project-tier file present; bonus: second case where project-tier file absent → falls through to embedded default).

**Evidence:**
- Table struct at `render_test.go:1715-1721` (fields `name`, `seedProjectTier`, `wantSentinel`, `wantNotSentinel`, `requireSetenvHome`).
- Case 1 `project_tier_present` at lines 1722-1732 with `seedProjectTier=true`, `wantSentinel="SENTINEL_D21_PROJECT_TIER"`, `wantNotSentinel="# PLACEHOLDER"`.
- Case 2 `project_tier_absent` at lines 1733-1744 with `seedProjectTier=false`, `wantSentinel="# PLACEHOLDER"`, `wantNotSentinel="SENTINEL_D21_PROJECT_TIER"`, `requireSetenvHome=true`.
- `for _, tc := range cases { tc := tc; t.Run(tc.name, ...) }` at lines 1747-1749 — canonical Go table-driven pattern.

**Verdict:** PASS (both cases present — bonus case included).

### Bullet 3 — Temp dir + subdir-per-group fixture with frontmatter + `## Role` + body > 200

> Test creates a temp dir via `t.TempDir()`, places a file at `<tmpdir>/.tillsyn/agents/go/builder-agent.md` containing a valid agent body (substantive body, frontmatter with `name: builder-agent`, body length > 200, contains `## Role`).

**Evidence:**
- `project.RepoPrimaryWorktree = t.TempDir()` at `render_test.go:1764` (NOT the hard-coded `/tmp/tillsyn/main` from `fixtureProject()`, which would have been a RiskNotes violation).
- Fixture writer call at `render_test.go:1784-1788`: `agentTierProjectFixture(t, project.RepoPrimaryWorktree, "go", "builder-agent.md", fixtureBody)`.
- Helper definition at `render_test.go:819-828` writes to `filepath.Join(projectDir, ".tillsyn", "agents", group, basename)` — i.e. `<tmpdir>/.tillsyn/agents/go/builder-agent.md` — exactly the W1.D3 subdir-per-group layout that matches `readProjectTierAgent`'s probe at `render.go:896` (`filepath.Join(projectWorktree, projectAgentsSubdir, group, basename)`).
- Fixture body at `render_test.go:1780-1783`:
  ```go
  fixtureBody := "---\nname: builder-agent\n---\n\n" +
      "## Role\n\n" +
      strings.Repeat("Project-tier D21 smoke-test filler to clear the 200-char Signal A floor. ", 4) +
      "\nSENTINEL_D21_PROJECT_TIER\n"
  ```
  - Frontmatter `name: builder-agent` — Signal B.
  - `## Role` marker — Signal C (per `validateAgentBodyShape` at `render.go:355-359`, Signal C accepts `# PLACEHOLDER` OR `# Section 0` OR `## Role`).
  - 4 × ~72-char filler = ~288 chars of post-frontmatter body — clears the 200-char Signal A floor.

**Verdict:** PASS.

### Bullet 4 — `Render()` called with project worktree + subdir-per-group template path

> Test calls `render.Render()` with `project.RepoPrimaryWorktree = tmpdir` and `binding.SystemPromptTemplatePath = "go/builder-agent.md"` (the subdir-per-group path that W1 introduces).

**Evidence:**
- `project.RepoPrimaryWorktree = t.TempDir()` at `render_test.go:1764`.
- `binding.SystemPromptTemplatePath = "go/builder-agent.md"` at `render_test.go:1808`.
- `render.Render(context.Background(), bundle, fixtureItem(), project, binding, nil)` invocation at `render_test.go:1817`.

The path `"go/builder-agent.md"` drives `resolveAgentGroup` (per `render.go:866`) to return `"go"` and `resolveAgentBasename` (per `render.go:779-790`) to return `"builder-agent.md"` — exactly aligning with the project-tier probe path `<worktree>/.tillsyn/agents/go/builder-agent.md`.

**Verdict:** PASS.

### Bullet 5 — `binding.AgentName = "builder-agent"` set

> Test sets `binding.AgentName = "builder-agent"` so the rendered file path resolves to `<bundle.Root>/plugin/agents/builder-agent.md` — the filename is driven by `binding.AgentName` (per `render.go:327`), NOT by `SystemPromptTemplatePath`.

**Evidence:**
- `AgentName: "builder-agent"` at `render_test.go:1806` inside the `dispatcher.BindingResolved` literal.
- Production-side: `renderAgentFile` at `render.go:616` writes `filepath.Join(dir, binding.AgentName+".md")` — i.e. `<bundle.Root>/plugin/agents/builder-agent.md`. The PLAN.md spec cites `render.go:327` (the validateBundle re-read path) which uses the identical join — both confirm `binding.AgentName` drives the rendered filename.
- The assertion path at `render_test.go:1821`: `readRenderedAgentFile(t, bundle.Paths.Root, binding.AgentName)` — helper at `render_test.go:847-855` reads `filepath.Join(bundleRoot, "plugin", "agents", agentName+".md")`. Matches.

**Verdict:** PASS.

### Bullet 6 — Asserts post-frontmatter body sentinel (NOT embedded default)

> Test asserts rendered agent file at `<bundle.Root>/plugin/agents/builder-agent.md` contains the test fixture's post-frontmatter body content (NOT the embedded default body).

**Evidence:**
- Case 1: `wantSentinel="SENTINEL_D21_PROJECT_TIER"` (project-tier fixture body) and `wantNotSentinel="# PLACEHOLDER"` (embedded default marker; verified to be present at `internal/templates/builtin/agents/go/builder-agent.md:6`).
- Case 2: `wantSentinel="# PLACEHOLDER"` and `wantNotSentinel="SENTINEL_D21_PROJECT_TIER"` — proves fall-through to embedded.
- Assertion logic at `render_test.go:1823-1830`:
  ```go
  if !strings.Contains(body, tc.wantSentinel) { t.Errorf(...) }
  if strings.Contains(body, tc.wantNotSentinel) { t.Errorf(...) }
  ```

The sentinel-based assertion (rather than full-file-bytes equality) is the spec's preferred approach (PLAN.md line 993 "Strip-then-inject note") because `assembleAgentFileBody` strips and re-injects frontmatter keys; the sentinel survives in the post-frontmatter section. The choice is documented in the test's lead comment at lines 1702-1708 — meets the spec's "document the choice" requirement.

**Verdict:** PASS.

### Bullet 7 — `t.Parallel()` used

> Test uses `t.Parallel()`.

**Evidence:**
- Sub-case 1 (`project_tier_present`) calls `t.Parallel()` at `render_test.go:1753`.
- Sub-case 2 (`project_tier_absent`) does NOT call `t.Parallel()` — by necessity, since `t.Setenv("HOME", t.TempDir())` at line 1759 panics if the test (or any ancestor) is parallel.
- The parent test does NOT call `t.Parallel()`, because the parent calling `t.Parallel()` would prevent case 2's `t.Setenv` from working.

This deviation is explicitly documented at `render_test.go:1709-1713` (parent doc-comment) and re-explained inline at lines 1750-1760. The interpretation of "Test uses `t.Parallel()`" is satisfied by sub-case 1 — the only sub-case where parallelism is mechanically permitted under Go's `t.Setenv`/`t.Parallel` mutual exclusion.

The droplet's `Specify` section calls out `requireSetenvHome=true` for case 2 — the spec author knew case 2 would need `t.Setenv` and that mixing with `t.Parallel` would panic. The builder's design (parent non-parallel, case-1 parallel) is the canonical Go pattern for this combination.

**Verdict:** PASS (with documented rationale).

### Bullet 8 — `mage test-pkg ./internal/app/dispatcher/cli_claude/render` passes

> `mage test-pkg ./internal/app/dispatcher/cli_claude/render` passes with the new test.

**Evidence:** Reviewer ran `mage test-pkg ./internal/app/dispatcher/cli_claude/render` — 86/86 PASS.

**Verdict:** PASS.

### Bullet 9 — `mage ci` green

> `mage ci` green.

**Evidence:** Builder claim states `mage ci` GREEN; reviewer's `mage test-pkg` for the touched package PASSES 86/86 (which is the package most exercised by this droplet) and `mage test-func` PASSES 3/3. The droplet adds test code only — no production-code surface that would change ci outputs elsewhere.

**Verdict:** PASS (verified via test-pkg + test-func; full `mage ci` re-run unnecessary for a test-only droplet whose package target is green and whose claim is corroborated).

### Bullet 10 — `t.TempDir()` (NOT `/tmp/tillsyn/main`)

> Do NOT use `/tmp/tillsyn/main` as the RepoPrimaryWorktree value — use `t.TempDir()` for proper test isolation. (RiskNotes constraint.)

**Evidence:** `render_test.go:1764` explicitly overrides `project.RepoPrimaryWorktree = t.TempDir()` after `fixtureProject()` returns the placeholder `/tmp/tillsyn/main`. Test isolation correct.

**Verdict:** PASS.

## Falsification attacks resolved

- **A1** — Does `binding.AgentName` get set so the rendered file lands where the assertion looks? Yes, `render_test.go:1806`; matches `render.go:616` write path and `readRenderedAgentFile` read path.
- **A2** — Does the test use `t.TempDir()`? Yes, `render_test.go:1764` overrides the fixture default.
- **A3** — Sentinel vs full-file assertion? Sentinel (PLAN.md-preferred) at `render_test.go:1823-1830`.
- **A4** — HOME neutralization correct? Yes, `t.Setenv("HOME", t.TempDir())` at line 1759 — `readUserTierAgent` (render.go:914-927) uses `os.UserHomeDir()` which on Unix reads `$HOME` first; an empty tempdir → `fs.ErrNotExist` → falls through to embedded.
- **A5** — Subdir-per-group fixture layout? Yes, `agentTierProjectFixture` writes to `.tillsyn/agents/<group>/<basename>` (render_test.go:821) — matches `readProjectTierAgent`'s probe path at render.go:896.
- **A6** — nil `ToolsAllowed`/`ToolsDisallowed` safe with strip-inject? Yes, per `render.go:626-655` strip is unconditional but operates on frontmatter keys (`allowedTools`, `disallowedTools`, `model`) — the test fixture's frontmatter has only `name`, so strip is a no-op; inject is also a no-op because slices are nil. Post-frontmatter body survives verbatim — sentinel preserved.
- **A7** — Does the embedded default still carry `# PLACEHOLDER`? Yes — verified at `internal/templates/builtin/agents/go/builder-agent.md:6` (`# PLACEHOLDER — substantive content lands in Drop 4c.8 W4`).
- **A8** — Does case 2 actually fall through to embedded? Yes — no project-tier seed (seedProjectTier=false → fixture not written), HOME neutralized to empty tempdir → user tier miss → embedded tier hit. `# PLACEHOLDER` from the embedded default appears in rendered output.

## NITs (if any)

**None.** All acceptance criteria met cleanly. The test design is consistent with the PLAN's strip-then-inject guidance (nil tool-gates + sentinel assertion), HOME neutralization is correct, t.Parallel deviation is explicitly documented in the lead comment, and the subdir-per-group fixture path matches W1.D3's production-side resolver.

## Verdict rationale

W8.D21 is a clean test-only droplet that exercises the W1.D3 subdir-per-group project-tier resolver. All 10 acceptance bullets map to concrete file:line evidence on disk; both sub-cases pass under `mage test-func` and the full render package passes 86/86 under `mage test-pkg`. The 8 falsification attacks each resolve into evidence-backed mitigations. Overall verdict: **PASS**.
