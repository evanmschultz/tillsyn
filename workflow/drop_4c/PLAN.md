# Drop 4c — Spawn Pipeline Redesign (F.7 Only)

**Status:** PLAN-authoring 2026-05-05.
**Author:** orchestrator (post-Drop-4b-merge synthesis from three parallel sub-planners).
**Drop scope (LOCKED):** Theme F.7 ONLY — spawn pipeline redesign, replaces 4a.19 stub wholesale. Splits the broader Drop 4c sketch into focused drops:
- **Drop 4c (this drop):** F.7 spawn pipeline (~33 droplets).
- **Drop 4c.5 (next drop, post-Drop-4c-merge):** F.1-F.3 + F.5-F.6 + Themes A/B/C/D + Theme E (~25 droplets) — template ergonomics + audit-debt.
- **Drop 4d (post-Drop-5 dogfood):** codex adapter on F.7.17 seam (~7-10 droplets).
- **Drop 4d-prime (post-Drop-5 dogfood):** F.4 marketplace CLI (~5 droplets) — no adopter consumes it pre-dogfood.

**Why split:** original SKETCH estimated ~50 droplets total; three parallel F.7 planners returned ~33 droplets for F.7 alone, putting full-Drop-4c at ~58-62. Splitting keeps drop-cycle time bounded and lets Drop 5 dogfood land sooner. F.7 is dogfood-BLOCKING (cascade-on-itself needs spawn pipeline); the rest is not.

## 1. Hard Prereqs

- Drop 4a on `main` (dispatcher core, 32 droplets, merged at `618c7d2` series).
- Drop 4b on `main` (gate execution + auth auto-revoke + git-status pre-check + auto-promotion subscriber + publishers, 8 droplets, merged at `86fba6f`).
- `internal/app/dispatcher/` package + `Dispatcher` interface + `RunOnce` (4a).
- `paths` / `packages` / `start_commit` / `end_commit` first-class on `ActionItem` (4a Wave 1).
- LiveWaitBroker + ActionItemChanged event (4a.15).
- Tree walker + auto-promotion (4a.18).
- Auth auto-revoke (4b.5), git-status pre-check (4b.6), auto-promotion subscriber (4b.7), publishers (4b.8).
- Gate framework (4b.1-4b.4): `mage_ci`, `mage_test_pkg`.

## 2. Goal

Replace the 4a.19 spawn.go stub wholesale with a production-grade spawn pipeline grounded in 2026-05-04 verbatim claude CLI probes. Architect the CLI adapter seam so Drop 4d (codex adapter) is purely additive. Ship the optional declarative context aggregator with FLEXIBILITY-not-PRESCRIPTION framing — both bounded-mode and agentic-mode are first-class.

End state:
- Per-spawn temp bundle lifecycle with `os.MkdirTemp` + `defer RemoveAll`.
- Per-binding TOML fields `env []string` / `cli_kind string`. Tillsyn never holds secrets; closed POSIX env baseline incl. proxy + TLS-cert vars (no `os.Environ()` inheritance).
- No `command` override field — adapter invokes the CLI binary directly (`claude` / `codex`); OS-level wrappers handle isolation.
- `CLIAdapter` interface (`BuildCommand` / `ParseStreamEvent` / `ExtractTerminalReport`) with `claudeAdapter` + `MockAdapter` test fixture.
- Stream-JSON monitor parses claude's terminal events; permission-denial → TUI handshake via SQLite `permission_grants(project_id, kind, rule, granted_by, granted_at, cli_kind)`.
- Optional `[agent_bindings.<kind>.context]` declarative aggregator with greedy-fit bundle cap + per-rule + per-bundle wall-clock caps.
- Commit + push gates via the new pipeline (default OFF until dogfood proves them safe).
- Drop hardcoded `hylla_artifact_ref` from prompt body.

## 3. Locked Architectural Decisions (drawn from SKETCH + planner-review + R2 falsification + dev decisions)

- **L1 — Tillsyn never holds secrets.** TOML refers to env-var NAMES only; `os.Getenv` at spawn; fail-loud on missing.
- **L2 — No Docker awareness, no OAuth registry, no container model, no wrapper-interop knob in Tillsyn core.** Adopters who want process isolation use OS-level mechanisms (PATH-shadowed binary shim, symlink replacement, container wrapping the entire Tillsyn binary, sandbox-exec). Tillsyn does NOT surface a `command` override field; the adapter calls its CLI binary directly (`claude` for claude adapter, `codex` for codex adapter). Removing the wrapper-interop knob entirely closes the marketplace-RCE vector — no shell-interpreter denylist needed, no per-token argv regex needed, no install-time argv-list confirmation needed.
- **L3 — POSIX-only.** Drop 4c spawn pipeline targets macOS / Linux only; Windows deferred to post-MVP refinement.
- **L4 — Closed env baseline.** Process basics: `PATH` (value `os.Getenv("PATH")` — inherit-PATH so the spawn finds `claude` / `codex` on the user's normal PATH), `HOME`, `USER`, `LANG`, `LC_ALL`, `TZ`, `TMPDIR`, `XDG_CONFIG_HOME`, `XDG_CACHE_HOME`. Network conventions (corporate-network adopters get a working spawn out-of-box; none carry secrets): `HTTP_PROXY`, `HTTPS_PROXY`, `NO_PROXY`, `http_proxy`, `https_proxy`, `no_proxy`, `SSL_CERT_FILE`, `SSL_CERT_DIR`, `CURL_CA_BUNDLE`. PLUS per-binding `env` allow-list. `os.Environ()` NOT inherited; closed-baseline purpose is to block `AWS_*` / `STRIPE_*` direnv-style secret-bearing names, not to break network connectivity for corporate adopters.
- **L8 — Tool-gating two-layer.** `settings.json` deny rules AUTHORITATIVE; agent-frontmatter mirrors for human readability; CLI flags skipped for typical kinds.
- **L9 — Conditional argv flag emission via `*int`/`*float64`/`*string`.** Priority cascade: `CLI > MCP > TUI > TOML > absent`.
- **L10 — Permission-denied → TUI handshake AT TERMINAL EVENT.** Real-time mid-stream variant out of scope.
- **L11 — Dispatcher monitor stays CLI-agnostic.** Stream parser lives inside the claude adapter package; dispatcher consumes adapter-returned `StreamEvent` values.
- **L12 — `metadata.spawn_history[]` is audit-only.** Round-history aggregation DEFERRED. Future need addressed via `prior_round_*` rules (worklog / gate output / QA findings), not raw stream-json.
- **L13 — F.7.18 context aggregator is OPTIONAL.** Templates that omit `[context]` entirely use full agentic exploration. Both modes first-class.
- **L14 — Greedy-fit bundle-cap algorithm.** Iterate rules in TOML declaration order; rules that bust cap are SKIPPED with markers; subsequent rules continue if they fit.
- **L15 — Two-axis wall-clock caps.** Per-rule `max_rule_duration = "500ms"` (default) + per-bundle `max_aggregator_duration = "2s"` (default).
- **L16 — Three-schema-droplet sequencing.** Schema-1 (F.7.17 per-binding fields) → Schema-2 (F.7.18 Context sub-struct) → Schema-3 (F.7.18 Tillsyn globals). Each ~1/3 the review surface; independent failure domains.
- **L17 — Hard-cut migration for future non-JSONL CLIs.** No backward-compat shim. ALL adapters + dispatcher monitor refactored in one drop when `ConsumeStream` push-model lands.
- **L18 — Drop 4c JSONL-only.** Only claude (4c) + codex (4d) — both JSONL. Non-JSONL extensibility is a roadmap concern.
- **L19 — `MockAdapter` test fixture in F.7.17 acceptance criteria.** Confirms multi-adapter readiness pre-Drop-4d.
- **L20 — Commit + push gates default OFF** via `dispatcher_commit_enabled` + `dispatcher_push_enabled` project metadata pointer-bools. Default template ships gates listed-but-toggle-disabled until dogfood proves them safe.
- **L21 — F.7.10 only removes the prompt-body `hylla_artifact_ref` line.** `domain.Project.HyllaArtifactRef` and project metadata preserved (adopter-local templates may opt into Hylla MCP).
- **L22 — `mage install` NEVER invoked by agents.** Dev-only target. Verification gates use `mage check` + `mage ci`.

## 4. Drop Structure

Theme F.7 decomposes into three sub-plans (each authored by a parallel planner):

### F.7-CORE — Spawn-Pipeline-Core (16 droplets)
Source: `workflow/drop_4c/F7_CORE_PLAN.md`.
Covers F.7.1-F.7.16: per-spawn bundle, tool-gating schema, headless argv, stream monitor, permission handshake, plugin pre-flight, gitignore, orphan scan, action-item metadata, hylla_artifact_ref removal, docs, commit-agent, commit gate, push gate, project-metadata toggles, default-template gate-list expansion.

### F.7.17 — CLI Adapter Seam (9 droplets after REVISIONS REV-1 + REV-7; was 11)
Source: `workflow/drop_4c/F7_17_CLI_ADAPTER_PLAN.md` — REVISIONS POST-AUTHORING section at the end supersedes affected body text. Builders MUST read REVISIONS first (per REV-8).
Removed: F.7.17.10 marketplace install paper-spec (no `command[0]` to confirm). F.7.17.9 monitor refactor (already covered by F.7-CORE F.7.4 from inception).
Covers Schema-1 (slim) + adapter scaffold + `claudeAdapter` + MockAdapter + dispatcher wiring + manifest cli_kind + permission_grants cli_kind column + BindingResolved resolver + adapter-authoring docs.

### F.7.18 — Context Aggregator (6 droplets)
Source: `workflow/drop_4c/F7_18_CONTEXT_AGG_PLAN.md`.
Covers Schema-2 + Schema-3 + aggregator engine + greedy-fit + two-axis timeouts + default-template `[context]` seeds + spawn_history doc-comment.

## 5. Cross-Plan Sequencing (DAG)

**Canonical droplet ID mapping (resolves Falsification #2 mismatch):**

| Conceptual label | Canonical droplet ID | Owner sub-plan | Ships |
|---|---|---|---|
| Schema-1 | F.7.17.1 | F.7.17 | `Env []string` + `CLIKind string` on `AgentBinding` + `validateAgentBindingEnvNames` |
| Adapter scaffold + types | F.7.17.2 | F.7.17 | `CLIKind` enum, `CLIAdapter` interface, `BindingResolved`, `BundlePaths`, `StreamEvent`, `ToolDenial`, `TerminalReport` |
| `claudeAdapter` | F.7.17.3 | F.7.17 | claude binary invocation (no `command` override; adapter hardcodes `claude`) |
| MockAdapter fixture | F.7.17.4 | F.7.17 | test fixture |
| Dispatcher wiring | F.7.17.5 | F.7.17 | `BuildSpawnCommand` looks up `CLIKind` → adapter |
| `Manifest.CLIKind` | F.7.17.6 | F.7.17 | SOLE OWNER. F.7-CORE F.7.1 ships `Manifest` WITHOUT `CLIKind`; this droplet adds it |
| `permission_grants.cli_kind` | F.7.17.7 | F.7.17 | SQLite column |
| `BindingResolved` resolver | F.7.17.8 | F.7.17 | priority cascade |
| CLI-agnostic monitor refactor | F.7.17.9 | F.7.17 | F.7.4 retro-edit |
| Adapter-authoring docs | F.7.17.10 | F.7.17 | docs (was F.7.17.11; renumbered after marketplace-install droplet removed) |
| Schema-2 | F.7.18.1 | F.7.18 | `Context` sub-struct on `AgentBinding` |
| Schema-3 | F.7.18.2 | F.7.18 | `Tillsyn` top-level struct (initial declaration with `MaxContextBundleChars` + `MaxAggregatorDuration`) |
| Aggregator engine | F.7.18.3 | F.7.18 | `internal/app/dispatcher/context/` package |
| Greedy-fit + caps | F.7.18.4 | F.7.18 | algorithm + two-axis timeouts |
| Default-template seeds | F.7.18.5 | F.7.18 | `default-go.toml` `[context]` blocks |
| `spawn_history[]` doc-comment | F.7.18.6 | F.7.18 | audit-only role pinned in godoc |

**Tillsyn struct extension policy:** F.7.18.2 owns the initial `Tillsyn` struct. F.7-CORE F.7.1 extends it with `SpawnTempRoot string`; F.7-CORE F.7.6 extends it with `RequiresPlugins []string`. Each extending droplet's acceptance criteria explicitly says "extends `Tillsyn` struct (initially declared in F.7.18.2)" and adds ONLY its named field with a unit test asserting strict-decode rejects unknown keys on the extended struct.

**Additional explicit blocked_by edges (resolves Falsification R2 #B4 + #C2 file-overlap concerns):**

- `F.7-CORE F.7.8 blocked_by F.7.17.6` — orphan scan reads `manifest.CLIKind`, which is added by F.7.17.6 only.
- `F.7-CORE F.7.1 blocked_by F.7.18.2` — `Tillsyn` struct's `SpawnTempRoot` extension requires the initial struct to exist.
- `F.7-CORE F.7.6 blocked_by F.7.18.2` — `Tillsyn` struct's `RequiresPlugins` extension requires the initial struct to exist.
- `F.7.17.9 (CLI-agnostic monitor refactor) blocked_by F.7-CORE F.7.4 (initial monitor implementation in claude adapter)` — F.7.4 lays the inline claude logic in `monitor.go`; F.7.17.9 then refactors it to dispatch via `adapter.ParseStreamEvent`. Sequential file-overlap; explicit edge.
- `F.7-CORE F.7.5c (settings.json grant injection) blocked_by F.7-CORE F.7.3b (bundle render — settings.json renderer)` — F.7.5c injects grants into settings.json renderer's permission entries.


```
[Drop 4b merged]
       │
       ├──→ F.7.10 (drop hylla_artifact_ref) — INDEPENDENT, can land first
       ├──→ F.7.9  (action-item metadata fields) — INDEPENDENT, can land first
       │
       ▼
[Schema-1: F.7.17 per-binding fields]  (FIRST schema droplet of F.7 wave)
       │
       ▼
[F.7.17 adapter scaffold + types]
       │
       ▼
[Schema-2: F.7.18 Context sub-struct]
       │
       ▼
[Schema-3: F.7.18 Tillsyn globals]
       │
       ├──→ F.7.18 aggregator engine
       │           │
       │           ▼
       │     F.7.18 default-template seeds
       │
       ├──→ F.7.17 claudeAdapter struct (consumes Schema-1)
       │           │
       │           ▼
       │     F.7.17 MockAdapter fixture
       │           │
       │           ▼
       │     F.7.17 dispatcher wiring
       │           │
       │           ▼
       │     F.7.17 manifest cli_kind + orphan-scan routing
       │           │
       │           ▼
       │     F.7.17 BindingResolved resolver
       │           │
       │           ▼
       │     F.7.17 CLI-agnostic monitor refactor
       │
       ├──→ F.7.1-F.7.6 spawn-pipeline-core (consume Schema-1 + adapter scaffold)
       │           │
       │           ▼
       │     F.7.7 (gitignore), F.7.8 (orphan scan), F.7.11 (docs)
       │
       └──→ F.7.12-F.7.16 commit/push gates + project-metadata toggles + default-template expansion
                          │
                          ▼
                  Drop 4c close: push + gh run watch + STEWARD post-merge consolidation
```

**Sequencing rule:** Schema-1 MUST land first (everything consumes it). Schema-2 + Schema-3 must land before F.7.18 engine. F.7.17 adapter scaffold must land before any droplet that consumes the `CLIAdapter` interface.

**Pre-dispatch reconciliation pass (resolves Falsification #2 cross-plan declaration mismatch):** before any builder fires, the orchestrator authors a follow-up reconciliation note in `workflow/drop_4c/PLAN_RECONCILIATION.md` that:

- Renames every "Schema-1" / "Schema-2" / "Schema-3" reference in F.7-CORE prereqs to its actual droplet ID in F.7.17 / F.7.18 sub-plans.
- Picks single-owner droplets for shared struct extensions:
  - `Tillsyn` top-level struct: F.7.18.2 owns initial declaration; F.7-CORE F.7.1 + F.7.6 add fields via "extends Tillsyn struct" markers in their droplet headers. Each extender droplet adds its own field with explicit acceptance criterion.
  - `Manifest.CLIKind`: F.7.17.6 is the SOLE owner; F.7-CORE F.7.1 ships `Manifest` WITHOUT `CLIKind`, F.7.17.6 adds the field afterwards.
- Converts every prose cross-plan handoff into machine-readable blocked_by chain (Schema-1 droplet ID → Schema-2 droplet ID → Schema-3 droplet ID → consumers).

**Droplet-sizing splits (resolves Falsification #11):**

- F.7.3 (originally ~600-800 LOC, 8+ files) splits into:
  - **F.7.3a — claude argv emission**: builds the headless argv slice from `BindingResolved` + bundle paths.
  - **F.7.3b — bundle render**: writes `system-prompt.md`, `system-append.md`, `settings.json`, `agents/<name>.md`, `.claude-plugin/plugin.json`, `.mcp.json` per the spawn architecture memory §2.
- F.7.5 (originally ~500 LOC, 5+ files) splits into:
  - **F.7.5a — `permission_grants` table + storage**: SQLite DDL + storage adapter ports.
  - **F.7.5b — TUI handshake**: parses `permission_denials[]` from terminal event → posts attention-item.
  - **F.7.5c — settings.json grant injection**: reads stored grants per-spawn → renders into `settings.json` `permissions.allow` entries.

Net droplet count: F.7-CORE 19 (16 + F.7.3 split +1 + F.7.5 split +2) + F.7.17 9 (11 -1 marketplace -1 monitor-refactor-merged) + F.7.18 6 = **34 droplets total in Drop 4c**.

**Acceptance criteria tightening (resolves Falsification #14):**

- Every droplet's acceptance criteria MUST cite concrete file:line pairs OR exact TOML / Go fixtures OR `grep -L`-style assertions. Vague phrasing like "implements X" is rejected at plan-QA time. Builders are expected to flag any droplet whose acceptance criteria they cannot mechanically verify.

**Metadata strategy lock (resolves Falsification #15):**

- F.7-CORE F.7.9 metadata fields (`spawn_bundle_path`, `spawn_history[]`, `actual_cost_usd`) ship as **JSON-encoded blob** inside `domain.ActionItem.Metadata`, NOT as new SQLite columns. Honors `feedback_no_migration_logic_pre_mvp.md` — no per-droplet column adds, no migration script, dev-fresh-DB pattern preserved.
- F.7.5a `permission_grants` table is the EXCEPTION (it's a new entity, not action-item metadata) and ships its DDL inline in the storage init path with explicit dev-fresh-DB callout in droplet acceptance criteria.

## 6. Pre-MVP Rules In Force

- **No migration logic in Go.** Schema additions (`AgentBinding.Command []string`, `Context` sub-struct, `Tillsyn` top-level struct, `permission_grants.cli_kind` column) ship inline in the storage init path; dev fresh-DBs `~/.tillsyn/tillsyn.db` BEFORE `mage ci` after each schema droplet.
- **No closeout MD rollups** (LEDGER / WIKI_CHANGELOG / REFINEMENTS / HYLLA_FEEDBACK) — pre-dogfood. Each droplet writes a per-droplet worklog only.
- **Opus builders.** Every builder spawn carries `model: opus`.
- **Filesystem-MD mode.** No Tillsyn-runtime per-droplet plan items.
- **Tillsyn-flow output style + Section 0 SEMI-FORMAL REASONING** in every subagent response.
- **Single-line conventional commits.** ≤72 chars.
- **NEVER raw `go test` / `go build` / `go vet` / `mage install`.** Always `mage <target>`.
- **Hylla is Go-only today.** Markdown sweeps fall back to `Read` / `rg` without logging Hylla misses.

## 6.5 Builder Spawn-Prompt Discipline (REVISIONS-first reading)

Sub-plans have REVISIONS POST-AUTHORING sections at the bottom that SUPERSEDE conflicting body text. Every F.7-touching builder spawn prompt MUST begin with:

> "Before reading the body of `<sub-plan>.md`, read the REVISIONS POST-AUTHORING section at the bottom of the file — it supersedes any conflicting body text. If a droplet body says one thing and REVISIONS says another, REVISIONS wins."

Orchestrator-procedural; does not require sub-plan rewrites.

## 7. Per-Droplet QA Discipline

Every Go-code-changing droplet receives the standard QA pair after the builder returns:
- **`go-qa-proof-agent`** verifies evidence completeness, reasoning coherence, trace coverage, claim support.
- **`go-qa-falsification-agent`** attacks the proposal: counterexamples, hidden dependencies, contract mismatches, YAGNI pressure, memory-rule conflicts.
- Both run in parallel with `run_in_background: true`.
- If either returns non-green, dispatch fix-builder, re-run QA twins, repeat until green.

MD-only droplets (F.7.11 docs, the round-history doc-comment droplet) get orchestrator self-QA per `feedback_md_update_qa.md`, no subagent QA.

## 8. Verification Gates

- **Per-droplet:** `mage check` (fast: format + vet + build) + `mage ci` (full: format + vet + build + test + race + coverage).
- **Drop-end:** local `mage ci` clean → push → `gh run watch --exit-status` until green.
- **Coverage:** below 70% is a hard failure (per project CLAUDE.md).

## 9. Out Of Scope

- F.1, F.2, F.3, F.5, F.6 (template ergonomics — moved to Drop 4c.5).
- Theme A (silent-data-loss + agent-surface hardening — Drop 4c.5).
- Theme B (dev escape hatches — Drop 4c.5).
- Theme C (STEWARD + cascade-precision refinements — Drop 4c.5).
- Theme D (pre-cascade hygiene — Drop 4c.5).
- Theme E (Drop-4a/4b residue — Drop 4c.5).
- F.4 (marketplace CLI — Drop 4d-prime, post-Drop-5).
- Theme G (post-MVP marketplace evolution — post-MVP).
- Codex adapter (Drop 4d, post-Drop-5).
- Drop 4.5 TUI overhaul (concurrent FE/TUI track).
- Drop 5 dogfood validation.

## 10. Open Questions Surfaced By Sub-Planners (resolve at plan-QA-twin time)

Aggregating from the three sub-plans:

**F.7-CORE Q1-Q7:**
- Q1: F.7.2 droplet sizing (split sandbox validation if too large?)
- Q2: Bundle cleanup timing on commit/push gate failure
- Q3: cli_kind column boundary between F.7-CORE F.7.5 and F.7.17 (which plan owns the table DDL?)
- Q4: `git add` semantics on first build with empty `start_commit`
- Q5: Default template gates listed-but-skipped vs only-mage_ci-until-toggle-flip
- Q6: Plugin pre-flight cache vs always-fresh per spawn
- Q7: F.7.11 docs authored by orchestrator vs builder subagent

**F.7.17 Q1-Q6:**
- Q8: Cross-planner coordination on `manifest.json` widening (F.7.1 vs F.7.17)
- Q9: Cross-planner coordination on monitor refactor (F.7.4 vs F.7.17)
- Q10: `BindingResolved.Command` defaulting (split-default vs centralized resolver)
- Q11: Allow-list location formalization (sub-plan flagged this — addressed in master PLAN §3 L7)
- Q12: Schema-1 droplet split (F.7.17 already split into multi-droplet; verify granular enough)
- Q13: Marketplace install-time confirmation lands as paper-spec OR as functional droplet in 4c

**F.7.18 Q1-Q4:**
- Q14: `metadata.spawn_history[]` doc-comment ownership (F.7-CORE F.7.9 vs F.7.18 standalone droplet)
- Q15: Production `GitDiffReader` adapter location (dispatcher root vs context package)
- Q16: Spawn-pipeline-calls-aggregator wiring (F.7-CORE vs F.7.18 ownership)
- Q17: `Bundle.Files` filename convention

## 11. References

- `workflow/drop_4c/SKETCH.md` (post-R2-rework, 2026-05-05).
- `workflow/drop_4c/F7_CORE_PLAN.md` (F.7.1-F.7.16 sub-plan).
- `workflow/drop_4c/F7_17_CLI_ADAPTER_PLAN.md` (F.7.17 sub-plan).
- `workflow/drop_4c/F7_18_CONTEXT_AGG_PLAN.md` (F.7.18 sub-plan).
- `workflow/drop_4c/4c_F7_EXT_PLANNER_REVIEW.md` (round-1 planner-grade review).
- `workflow/drop_4c/4c_F7_EXT_QA_PROOF_R2.md` (round-2 proof — GREEN-WITH-NITS).
- `workflow/drop_4c/4c_F7_EXT_QA_FALSIFICATION_R2.md` (round-2 falsification — closed via dev decisions).
- `~/.claude/projects/-Users-evanschultz-Documents-Code-hylla-tillsyn/memory/project_drop_4c_spawn_architecture.md` (canonical 13-section spawn architecture).
- `WIKI.md`, `PLAN.md`, `CLAUDE.md` — project-level discipline.
- `workflow/example/drops/WORKFLOW.md` — drop lifecycle phases.
