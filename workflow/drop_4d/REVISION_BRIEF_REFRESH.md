# Drop 4d — REVISION_BRIEF Refresh

**Author:** research-agent (orch-driven).
**Date:** 2026-05-19.
**Brief age:** authored 2026-05-06, last touched in commit `ef2e85f` (Drop 4c era).
**HEAD at refresh:** `752cb94`. Hylla snapshot still pinned at `2124d2c` — post-`2124d2c` files verified via `Read` / `git diff` only.
**Pristine:** `workflow/drop_4d/REVISION_BRIEF.md` not edited; this file is the orch's input for a deliberate brief update.

## 1. Drift Summary

| Area | Status | Notes |
| --- | --- | --- |
| `CLIAdapter` interface shape | UNCHANGED | Three methods, same signatures. Codex slot still cleanly available. |
| `CLIKindClaude` + `IsValidCLIKind` | UNCHANGED | Brief's seam edits (4d.1) apply verbatim. |
| `BindingResolved` struct | EXTENDED | New field `SystemPromptTemplatePath` (`cli_adapter.go:107-131`) landed during Drop 4c.6 render work. Drop 4d codex adapter must consume it the same way cli_claude does — see §4. |
| `RegisterAdapter` cite | MOVED | Brief says `cli_adapter.go:33-49`; actually lives at `spawn.go:263-267`. Adapter map at `spawn.go:251`. |
| `spawn.go:240` example cite | SHIFTED | Brief's "registration pattern at line 240+" — actual register-comment now at `spawn.go:232-250`; the registration call example is at `spawn.go:240` inside a doc-comment. ACCURATE-ENOUGH; cite to line 240+ still lands in the right neighborhood. |
| `cmd/till/main.go:31` blank import | SHIFTED ±1 | cli_claude side-effect import now at line 32 (preceded by L29-31 comment). Cite to `cmd/till/main.go:32`. |
| `cmd/till/main.go:2704` `dispatcherTemplateResolver` | ACCURATE | Verified — exact line. NEW SYMBOL since brief authoring (introduced in commit `0360dc6`). Brief does not mention it as a context-cite — see §5 for whether codex adapter must thread through it. |
| `mock_adapter_test.go:555-578` contract test | UNCHANGED | `TestCLIAdapterContractTableDriven` still at line 555; table still single-row at 568. Brief's 4d.8 extension applies verbatim. |
| `binding_resolved.go:127-130` L15 default-to-claude | SHIFTED | Actual default block now at `binding_resolved.go:129-134` (the `if resolved.CLIKind == "" { resolved.CLIKind = CLIKindClaude }` block). |
| `cli_claude/render/` | MAJORLY EXTENDED | `render.go` grew from ~? to ~707-line diff (+707 lines), `render_test.go` +1176. Codex render mirror is now a larger build than brief implied — see §3. |
| `plugin_preflight.go` | EXTENDED | Brief cites `claude plugin list --json` invocation. Verified intact at `plugin_preflight.go:46-90` (`ClaudePluginListEntry`, `ClaudePluginLister`, `ErrClaudeBinaryMissing`). Routing 4d.2 still applies. |
| NEW: `hook_preflight.go` | NEW FILE | 284 lines, lands during Drop 4c.6/4c.6.1. Validates rendered hook scripts under `.claude/hooks/`. Codex has no `.claude/hooks/` equivalent — see §4 for routing implication. |
| NEW: `dispatcher_e2e_test.go` | NEW FILE | 713-line e2e suite. Codex contract row must extend whatever cross-CLI tables this file uses. See §6. |
| `monitor.go applyCrashTransition` (Q2 R-A.4-1) | STILL BUGGY | Brief flags ordering issue at `monitor.go:344-369`; actual location now `monitor.go:389-434`. Bug confirmed at HEAD — `MoveActionItem` on line 412 fires BEFORE metadata update on line 427. Brief's Q2 routing recommendation (land as 4d.0) STILL valid. |
| `dispatcher.go transitionToFailed` (Q2 R-A.4-1) | NEEDS REVERIFY | Brief cites `dispatcher.go:631-639`. Dispatcher grew 84 lines net since brief; line may have moved. Not loaded in this research pass — flag for planner to reverify. |

## 2. Hard Prereqs — Resolved Status

- **Drop 4c.5 closed + merged.** RESOLVED. Closing commit `f81f4a7 docs(drop-4c.5): mark drop done` dated 2026-05-09 — 10 days before this refresh. Brief's "HARD PRECONDITION: Drop 4c.5 closed and merged before Drop 4d builders fire" is SATISFIED.
- **`testdata/codex_stream_minimal.jsonl` fixture (Q1 Path A).** STILL PENDING. No `codex_stream*.jsonl` file exists anywhere in repo (verified via filesystem inspection of `internal/app/dispatcher/cli_claude/testdata/` — only `claude_stream_minimal.jsonl` and `mock_stream_minimal.jsonl` present). Brief's Q1 Path A still required before planner spawn.
- **Codex CLI on dev `$PATH`.** UNVERIFIED in this pass (cannot probe `$PATH` from research-agent sandbox). Dev-environment assumption; brief's "verified-once-by-dev" carve-out still applies.
- **`cli_codex/` package on disk.** ABSENT. No `internal/app/dispatcher/cli_codex/` directory exists. Builders create from scratch per brief §3.2.

## 3. Surfaces That Grew Since Brief Authoring (Affecting Scope)

Between commits `ef2e85f` (brief authored) and `752cb94` (HEAD), `internal/app/dispatcher/` accumulated 4,447 insertions / 168 deletions across 20 files. The Drop 4d-relevant changes:

- **`cli_claude/render/render.go` + `render_test.go`** — +707 / +1176 lines respectively. The render layer is now responsible for materialising agent files via a 3-tier walk (project → user → embedded), stripping cross-CLI sections, injecting permission grants, validating outputs. Brief §3.2 droplet 4d.7 says "Mirrors `cli_claude/render/` shape — `Render(ctx, ...) error` writes per-spawn artifacts." That description is now thin. **Codex render is a meaningfully bigger droplet than the brief suggests** because:
  - cli_claude/render now has post-render validators (`monitor.go` and render docs reference a validator at render exit; see Drop 4c.6 W3.D5 commit `5bfd242`).
  - There's a `defense-in-depth env vars in cli claude` commit (`01f4a22`, Drop 4c.6 W3.D4) that touches render → adapter wiring.
  - The 3-tier resolver (`02a3709` "w1.d3 subdir-per-group project-tier resolver") needs a codex-side counterpart OR explicit confirmation that codex bundles inherit the same resolution.
  - **Recommended brief update**: 4d.7 droplet split into 4d.7a (render skeleton + config.toml + AGENTS.md write) and 4d.7b (post-render validator + 3-tier-resolver wiring) OR explicit out-of-scope note that codex deliberately ships without the validator/3-tier extensions until Drop 5 surfaces the need.

- **`hook_preflight.go`** — NEW 284-line file (commit `fb0b3ec` "require hook artifacts when agent declares preToolUse"). The dispatcher now blocks spawns when a bound agent declares a PreToolUse hook but `<worktreePath>/.claude/hooks/validate-action-item-paths.sh` is missing or stale. **Codex has no `.claude/hooks/` equivalent** (it reads `~/.codex/` or `$CODEX_HOME`). The dispatcher's `CheckHookArtifacts` must be routed around for codex spawns the same way `plugin_preflight.go` is being routed in 4d.2. **NEW droplet recommended**: 4d.2b (hook_preflight skip for codex) — same shape as 4d.2 (Option A hardcoded skip vs Option B per-adapter `HookPreflight` method).

- **`dispatcher_e2e_test.go`** — NEW 713-line file (commit `006ff57` "add end-to-end auto-dispatch + gate-runner integration"). Brief's 4d.8 cites `mock_adapter_test.go:568` as the only extension site. The e2e test may also need a codex row OR an explicit "claude-only e2e" comment carve-out. **Reverify at planner spawn**: read `dispatcher_e2e_test.go` to confirm whether it's adapter-agnostic (codex inherits coverage for free) or codex needs its own e2e fixture.

- **Gate pipeline (`gates.go`, `gate_mage_ci.go`, `gate_mage_test_pkg.go`, `gate_commit.go`, `gate_push.go`)** — Drop 4b shipped these between brief authoring and HEAD. Brief makes no claim about gates touching the codex path. **Verify**: the gate runner takes a `templateResolver` (`monitor.go:441-461`) — codex spawns must thread through the same resolver. Probably zero codex-specific work, but planner should confirm in 4d.6 spec.

- **`monitor.go applyCleanExitTransition`** — NEW path (`monitor.go:436-476`). Clean-exit transitions now run gates. Codex spawns will hit this path the same as claude spawns. **Verify**: gate sequence is CLI-agnostic — gates run against the worktree, not against the spawn's CLI binary. Probably no codex-specific change needed.

## 4. New Prereqs / Dependencies Since Brief

- **Hook-preflight routing for codex.** New work item (see §3 above). Insert as 4d.2b OR fold into 4d.2's option pair.
- **Render-layer feature drift.** The codex render droplet is closer to ~150-200 LOC than ~50-80 LOC as brief implied. Update §10 droplet-size estimate from "~5-6 new files mirroring cli_claude" to "~6-8 new files plus possible render-validator + 3-tier-resolver wiring."
- **`SystemPromptTemplatePath` field on `BindingResolved`.** New `cli_adapter.go:107-131` field carries per-binding agent-file path source. cli_claude consumes it in render's 3-tier walk. Codex adapter must either:
  - Wire equivalent 3-tier resolution into cli_codex/render/ (mirrors cli_claude).
  - Document explicit deferral (codex render reads embedded default only for Drop 4d; 3-tier walk lands in Drop 5 multi-CLI dogfood prep).
  - **Recommendation**: defer. The 3-tier walk is project-customization-driven; Drop 4d's job is seam validation, not feature parity with claude's render maturity. Brief should add this to §5 Locked Architectural Decisions as `L-codex-6`.

## 5. Cross-Cuts From Sibling-Drop Work

- **`drop_4b_test_cleanup` (in flight, has its own plan-QA artifacts).** Latest commit `752cb94 test(dispatcher): r7.1 r7.2 r7.3 broker-chain e2e enrichment` lands new test infrastructure in `dispatcher_e2e_test.go`. The Drop 4d planner should READ that file before authoring 4d.8 — the e2e suite shape may have changed since the brief assumed `mock_adapter_test.go` was the sole contract surface.
- **`drop_fe_1_bootstrap` (in flight).** UI work in `ui/` subtree + `cmd/till` UI hooks. No dispatcher touch. ZERO collision with Drop 4d.
- **`feat(mcp): r8 register till.action_item operation supersede` (commits `89b4292`, `133d4c0`).** MCP-side `supersede` op added; non-dispatcher. ZERO collision.
- **`refactor(adapters): w7.d3 delete HTTP residue` (`5fb3f15`) + `w7.d2 extract mcp packages` (`5be5322`).** MCP adapter package layout shuffled. cmd/till imports updated; dispatcher untouched. ZERO collision.
- **`feat(cli): till dispatcher serve` (`731522e`) and co-host (`c8c16c2`).** The `till dispatcher` standalone subcommand was removed; dispatcher is now co-hosted inside `till mcp`. **Possible cite drift**: brief mentions "cascade dispatcher" generally but does not cite the standalone `till dispatcher run` CLI. If REVISION_BRIEF anywhere references invoking `till dispatcher run --action-item <id>` for testing codex spawns, that path is now `till mcp` co-hosted. SPOT CHECK during brief update.

## 6. Updated File:Line Cites Table

| Brief location | Cite as written | Cite as exists at HEAD | Action |
| --- | --- | --- | --- |
| §3.1 4d.1 | `cli_adapter.go:43-48` (IsValidCLIKind switch) | `cli_adapter.go:42-49` | Update inclusive range |
| §3.1 4d.1 | `cli_adapter.go:108-111` (doc-comment "Drop 4c only") | `cli_adapter.go:30-34` (`CLIKindClaude` const doc-comment) AND `cli_adapter.go:35-49` (IsValidCLIKind doc-comment) — the "Drop 4c only" phrasing appears in the const doc-comment | Update line range |
| §1 Hard Prereqs | "`cli_adapter.go:33-49`" RegisterAdapter | `spawn.go:263-267` (definition) + `spawn.go:251` (adapter map) | **MAJOR FIX**: RegisterAdapter lives in spawn.go, not cli_adapter.go. The full "spawn-side registry" mental model is in `spawn.go:232-277`. |
| §1 Hard Prereqs | "`spawn.go:240` example" | Still in same neighborhood; line 240 is inside the adaptersMap doc-comment showing the registration example | KEEP — accurate-enough |
| §3.3 4d.8 | "`cmd/till/main.go:31`" blank import line | `cmd/till/main.go:32` | Update to line 32 |
| §11 References | `cmd/till/main.go:31` | Same fix | Update to line 32 |
| §11 References | `internal/app/dispatcher/binding_resolved.go:127-130` (L15 default-to-claude) | `binding_resolved.go:129-134` | Update line range |
| §3.4 4d.0 | `monitor.go applyCrashTransition` at `line 344-369` | `monitor.go:389-434` | Update line range — bug still present |
| §3.4 4d.0 | `dispatcher.go transitionToFailed` at `line 631-639` | UNVERIFIED in this pass — likely shifted | Planner should reverify at spawn time |
| §11 References | `mock_adapter_test.go` TestCLIAdapterContractTableDriven `at line 555` | `mock_adapter_test.go:555` | UNCHANGED |
| §11 References | `mock_adapter_test.go:568` (table location for 4d.8 extension) | `mock_adapter_test.go:568` | UNCHANGED |
| §11 References | `permission_grants_repo_test.go:57` | UNVERIFIED in this pass | Brief's QA Proof Round 1 §1.9 verified this on 2026-05-06; assume still ACCURATE |
| §11 References | `internal/templates/schema.go:475-488` (AgentBinding.CLIKind) | UNVERIFIED in this pass | Likely shifted by Drop 4c.6 template refactors; planner should reverify |

## 7. Recommended Additions to the Brief

- **Add a §1 line: "Drop 4c.5 closed `f81f4a7` on 2026-05-09 — hard prereq SATISFIED."** Remove the "in flight in `workflow/drop_4c_5/`" qualifier.
- **Add §1 prereq: "`hook_preflight.go` (`internal/app/dispatcher/hook_preflight.go`) lands a new claude-specific preflight that codex must be routed around — see new droplet 4d.2b."**
- **Add §1 reference to the SystemPromptTemplatePath field** introduced post-brief — codex adapter consumes it identically (or defers per `L-codex-6`).
- **Insert droplet 4d.2b (hook_preflight skip for codex).** Mirrors 4d.2 (plugin_preflight skip) Option A/B pair. Blocked_by: 4d.1.
- **Insert `L-codex-6` in §5: "Drop 4d codex render writes from embedded defaults only; 3-tier project/user/embedded walk lands in Drop 5 multi-CLI dogfood prep if surfaced."**
- **Update §10 droplet count from 7-9 to 8-10** (add 4d.2b; possibly split 4d.7 into 4d.7a + 4d.7b for render-validator wiring).
- **Update §10 risk-adjusted estimate from "2-3 days" to "3-4 days"** to account for render-layer drift since brief authoring.
- **Add §13 R7: "Render layer grew during Drop 4c.6 (3-tier resolver, post-render validators, defense-in-depth env vars). Codex render mirror lands without these extensions; mitigation: explicit `L-codex-6` carve-out + Drop 5 surfaces need."**
- **Add §11 reference to `dispatcher_e2e_test.go`** as a possible 4d.8 extension site to verify.
- **Add §11 reference to `hook_preflight.go`** for the 4d.2b routing target.

## 8. Recommended Deletions / Re-Scopes

- **§1 hard prereq: "F.7 spawn pipeline shipped — ..."** — list is now stale. Trim to "F.7 spawn pipeline shipped per Drop 4c; further extensions landed in Drop 4c.6 (render 3-tier resolver, hook preflight, post-render validator). Codex adapter consumes the same seam." Avoids enumerating individual files that have shifted.
- **§12 swap mechanics checklist** — items still applicable, but the `f81f4a7 mark drop done` PLAN.md update should be verified (run `Read` on `PLAN.md` during brief update to confirm 4c.5 row is closed and 4d row is next).

## 9. New Prereqs Emerged Since Brief

1. **Hook-preflight routing.** `hook_preflight.go` did not exist when brief was written. Codex spawns will hit `CheckHookArtifacts` and need explicit skip routing.
2. **`SystemPromptTemplatePath` consumption decision.** Codex adapter must explicitly decide on the field (consume via 3-tier walk vs ignore + use embedded only).
3. **Co-hosted dispatcher.** `till dispatcher run` standalone CLI was retired in `c8c16c2`. Codex spawn testing happens via `till mcp` co-host OR via direct unit-test-level adapter calls. Confirm in planner's testing-strategy section.

## 10. Recommended Next Step

**PROCEED WITH PLANNER ROUND 1, but with a BRIEF UPDATE PASS FIRST.** The brief is broadly accurate but accumulates ~12 cite shifts and 2-3 scope additions (4d.2b, render droplet split, L-codex-6) that the planner would surface as PLAN_QA findings anyway. A focused 30-minute brief update by the orch (apply §6 cite table + §7 additions + §8 deletions) is cheaper than letting the planner discover them in round 1.

**Specifically:**

1. **Brief update pass (orch-direct edit).** Apply §6 cite table + §7 + §8 to REVISION_BRIEF.md. Single commit.
2. **Q1 fixture capture (dev-direct).** Dev runs `codex exec --json --skip-git-repo-check -C /tmp "say hi"` and commits as `internal/app/dispatcher/cli_codex/testdata/codex_stream_minimal.jsonl`. Single commit. ~30 seconds dev time.
3. **Planner spawn (round 1).** Single planner per brief §9 Q6 recommendation. Reads updated REVISION_BRIEF + Q1 fixture. Produces PLAN.md round 1.

**Defer-rework triggers (would push to scope-rework instead of proceed):**

- If the planner finds `CLIAdapter` interface changes are needed (unlikely; QA Proof Round 1 §1.4 confirmed argv divergence is non-interface-breaking).
- If Q1 fixture probe reveals codex emits SSE-fragment streams instead of strict JSONL (per brief §13 R1 mitigation). Then Drop 4d gains a 2-3 droplet interface-rewrite phase up front.
- If dev decides to absorb Drop 4d into Drop 5 (single multi-CLI drop instead of seam-validation drop + dogfood drop). Brief is sized for seam-validation-only; absorption changes Phase 3 + adds Drop 5's prior scope.

## 11. Confidence Statement

This refresh covers what changed in `internal/app/dispatcher/` between `ef2e85f` and `752cb94` per `git diff --stat` and targeted Read passes. NOT exhaustively reverified:

- `dispatcher.go:631-639` `transitionToFailed` line drift (file grew 84 lines net; cite likely shifted).
- `internal/templates/schema.go:475-488` `AgentBinding.CLIKind` cite (Drop 4c.6 template refactors touched schema.go).
- `permission_grants_repo_test.go:57` cite (Drop 4c.6 may have touched).
- Codex CLI binary availability on dev `$PATH` (cannot probe from research sandbox).
- `dispatcher_e2e_test.go` adapter-row shape (NEW file; planner should read at round 1).

All other §6 cites verified via direct Read of the cited file at HEAD `752cb94`.

## Hylla Feedback

N/A — research touched only files modified after Hylla's pinned snapshot `2124d2c`. Per the system reminder about mid-reingest, all evidence gathered via direct `Read` + `git diff` + `git log` rather than Hylla queries. No Hylla calls attempted; no Hylla misses to report.
