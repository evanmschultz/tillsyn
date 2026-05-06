# Drop 4c — Master + F.7 Sub-Plans QA Proof Review

**Reviewer:** plan-QA-proof (read-only).
**Date:** 2026-05-05.
**Targets:** `workflow/drop_4c/PLAN.md` (master) + `F7_CORE_PLAN.md` + `F7_17_CLI_ADAPTER_PLAN.md` + `F7_18_CONTEXT_AGG_PLAN.md`.
**Mode:** filesystem-MD review BEFORE builder dispatch. No code edits, no plan edits.
**Verdict:** **PROOF GREEN-WITH-NITS.** All R2-confirmed counterexample mitigations baked, all L-decisions traceable, all 33 F.7 droplets covered, acceptance criteria substantively testable. Two NITs surfaced (cosmetic citation drift), neither blocks dispatch. Ready for builder fire after orchestrator-level cross-planner synthesis on Q3 / Q11 / Q14-Q17.

---

## 1. Verdict Summary

PROOF GREEN-WITH-NITS. Master + sub-plans pass evidence-completeness, source-traceability, R2-mitigation, and acceptance-testability checks. Both NITs are bibliographic-citation drift (no semantic impact) and can be tightened at next-revision time without blocking builder dispatch.

---

## 2. Code-Anchor Verification

Every concrete file:line claim in the plans cross-walked against the local checkout. Hylla NOT consulted (per spawn-prompt directive — Hylla stale post-Drop-4b-merge).

| Plan claim | Cited location | Verified location | Verdict |
|---|---|---|---|
| `Template` struct + `AgentBindings` map | F.7-CORE 1.4 / F.7.17 / F.7.18 cite `internal/templates/schema.go:127-199`, `:285-332` | `schema.go:127` (Template); `:144` (AgentBindings); `:285` (AgentBinding) | PASS |
| Strict-decode chain | All plans cite `internal/templates/load.go:88-95` (`DisallowUnknownFields`) | `load.go:80-95` (strict decode block; `DisallowUnknownFields()` at line 89) | PASS — minor envelope ±5 lines OK |
| `domain.Kind` 12-value enum | F.7.18 cites `internal/domain/kind.go:18-47` | `kind.go:18-31` (closed 12-value const block); `:34-47` (validKinds slice) | PASS |
| `Project.HyllaArtifactRef` field preserved | F.7.10 acceptance + master L21 | `internal/domain/project.go:17`, `:25` (field decl), `:158` (Project struct), `:282` (round-trip) | PASS — field exists, will survive F.7.10's prompt-body removal |
| `assemblePrompt` `hylla_artifact_ref:` line removal | F.7.10 cites "spawn.go:211-213" | `spawn.go:211` `b.WriteString("hylla_artifact_ref: ")`; `:212` `b.WriteString(project.HyllaArtifactRef)`; `:213` `b.WriteString("\n")` | PASS — lines exactly as cited |
| spawn_test current `hylla_artifact_ref` assertion | F.7.10 acceptance (test must assert NOT-present after removal) | `spawn_test.go:126` literal `"hylla_artifact_ref: " + project.HyllaArtifactRef` in expected-prompt slice | PASS — current assertion exists; F.7.10 will flip it |
| `mcpConfigPlaceholderPath` retired | F.7.17.5 acceptance cites `spawn.go:177-179` | `spawn.go:137` (call site); `:177-179` (helper decl) | PASS |
| `--mcp-config <path>` argv slot | F.7.17.5 acceptance ("`--mcp-config` flag is dropped") | `spawn.go:145` `"--mcp-config", mcpConfigPath` | PASS |
| `default.toml` `gates.build = ["mage_ci"]` | F.7.16 cites Drop 4b state | `internal/templates/builtin/default.toml:351` exactly `build = ["mage_ci"]` | PASS |
| `GateKind` closed-enum | F.7.13 + F.7.14 cite extending the enum | `schema.go:63-93` (3-value enum + `validGateKinds` slice; doc-comment line 56-58 explicitly anticipates Drop 4c adding "commit" + "push") | PASS — extension expected by current code |
| `domain.ActionItem.Paths/Packages/StartCommit/EndCommit/Files` | F.7.18 + F.7.13 cite Drop 4a Wave 1 first-class fields | `internal/domain/action_item.go:91, 104, 121, 136, 154` (all five fields exist with full doc-comments) | PASS |
| `ProjectMetadata.OrchSelfApprovalEnabled *bool` precedent | F.7.15 acceptance cites "Drop 4a 4a.25 precedent" | `internal/domain/project.go:119-145` (`ProjectMetadata` with `OrchSelfApprovalEnabled *bool` + `OrchSelfApprovalIsEnabled() bool` accessor) — actual provenance is **Drop 4a Wave 3 W3.2** per godoc lines 100-114, NOT 4a.25 | NIT-1 (citation drift; semantics correct) |
| `filteredGitEnv` location | F.7.18 plan cites `internal/app/git_status.go:146-156` | Function comment at `:141`; declaration at `:146`; body through `:156` | NIT-2 (off-by-5 lines on cite envelope; harmless) |
| `monitor.go` 4a.21 process monitor (PID + exit watch only) | F.7.4 cites stream parser layers on top | `monitor.go` has `processMonitor.runHandle/Track/Wait/applyCrashTransition` — no stream parsing | PASS |

Result: 13 PASS, 2 NIT (cosmetic).

---

## 3. Master PLAN.md L1-L22 Source Traceability

Each L-decision must trace to SKETCH line, planner-review P-ID, R2 falsification A-ID, dev decision in conversation, or project CLAUDE.md.

| L | Decision | Source | Verified |
|---|---|---|---|
| L1 | Tillsyn never holds secrets — env-NAMES only | SKETCH:151,154,155 | YES |
| L2 | No Docker / no OAuth / no container model | SKETCH:147,164 | YES |
| L3 | POSIX-only Drop 4c | A2.c CONFIRMED-NEW | YES |
| L4 | Closed env baseline + TMPDIR/XDG/inherit-PATH | A2.a + A2.b CONFIRMED-NEW | YES |
| L5 | Per-token regex with explicit `MatchString` literal pin | A1.c CONFIRMED-NEW | YES |
| L6 | Closed shell-interpreter denylist | A1.a CONFIRMED-NEW | YES |
| L7 | Marketplace install-time interactive confirmation (replaces prior allow-list framing) | A1.b CONFIRMED-NEW + dev decision (R2 path-forward picked confirmation over allow-list) | YES |
| L8 | Tool-gating two-layer (settings.json deny authoritative) | SKETCH:F.7.2 lines 127 + memory §5 | YES |
| L9 | Conditional argv via `*int`/`*float64`/`*string` | SKETCH:129 + memory §3 | YES |
| L10 | Permission-denied → TUI handshake at terminal event | SKETCH:F.7.5 line 133 | YES |
| L11 | Dispatcher monitor stays CLI-agnostic | planner-review P13 §6.3 | YES |
| L12 | `metadata.spawn_history[]` audit-only; round-history deferred | A5 REFUTED-WITH-NIT + SKETCH:205,217 | YES |
| L13 | F.7.18 context aggregator OPTIONAL | SKETCH:180,206-209 + A5 + planner-review §3.1 | YES |
| L14 | Greedy-fit bundle-cap algorithm | A6.b CONFIRMED-NEW (greedy chosen over serial-drop) | YES |
| L15 | Two-axis wall-clock caps (per-rule + per-bundle) | A7.b CONFIRMED-NEW | YES |
| L16 | Three-schema-droplet sequencing (Schema-1/2/3) | A3.b CONFIRMED-NEW (split SPOF into 3) | YES |
| L17 | Hard-cut migration for future non-JSONL CLIs | A4.a CONFIRMED-NEW (rejected add-then-deprecate) | YES |
| L18 | Drop 4c JSONL-only | SKETCH:147 (explicit "Drop 4c scope is JSONL-stream only") | YES |
| L19 | `MockAdapter` test fixture | A8.a REFUTED-WITH-NIT (gap closed) | YES |
| L20 | Commit + push gates default OFF via pointer-bool toggles | SKETCH:F.7.15 line 224 + Drop 4a Wave 3 W3.2 (`OrchSelfApprovalEnabled *bool`) precedent | YES — semantics correct (NIT-1: master + F.7.15 cite "4a.25"; actual provenance is W3.2) |
| L21 | F.7.10 only removes prompt-body line; preserves Project field + metadata | SKETCH:F.7.10 line 143 + planner-review §F.7.10 | YES |
| L22 | `mage install` NEVER invoked by agents | project CLAUDE.md "Build Verification" §3 + memory `feedback_no_migration_logic_pre_mvp.md` (sibling rule) + repeated in `feedback_orchestrator_no_build.md` | YES |

22/22 L-decisions traceable. NIT-1 affects only the bibliographic citation on L20/F.7.15 (4a.25 vs W3.2).

---

## 4. Cross-Plan Dependency Coherence

### 4.1 Schema sequencing

Master §3 L16 + master §5 DAG declare:

```
Schema-1 (F.7.17.1: Command/ArgsPrefix/Env/CLIKind)
   → Schema-2 (F.7.18.1: Context sub-struct on AgentBinding)
      → Schema-3 (F.7.18.2: Tillsyn top-level globals)
```

- F.7-CORE plan correctly declares F.7.17 Schema-1 as cross-plan prereq for F.7.1, F.7.3, F.7.5, F.7.6, F.7.8, F.7.12.
- F.7-CORE F.7.1 reserves the `[tillsyn]` `spawn_temp_root` field with explicit cross-plan handoff: "F.7.18 Schema-3 droplet adds the `Tillsyn` struct + `Template.Tillsyn` field; F.7.1 extends the same struct with `SpawnTempRoot string`."
- F.7-CORE F.7.6 similarly handoffs `RequiresPlugins []string` field to F.7.18.2 Schema-3 droplet.
- F.7.17 plan declares Schema-1 first in DAG (4c.F.7.17.1) — gates everything.
- F.7.18 plan declares F.7.18.1 (Schema-2) blocked_by F.7.17.1 (Schema-1); F.7.18.2 (Schema-3) blocked_by F.7.18.1.

PASS.

### 4.2 Cross-plan claim coverage

Master §4 claims F.7-CORE = F.7.1-F.7.16, F.7.17 = 11 droplets, F.7.18 = 6 droplets. Verified:

- F.7-CORE PLAN.md per-droplet sections: F.7.1, F.7.2, F.7.3, F.7.4, F.7.5, F.7.6, F.7.7, F.7.8, F.7.9, F.7.10, F.7.11, F.7.12, F.7.13, F.7.14, F.7.15, F.7.16 — 16 droplets, matches.
- F.7.17 PLAN.md: 4c.F.7.17.1 through 4c.F.7.17.11 — 11 droplets, matches.
- F.7.18 PLAN.md: F.7.18.1 through F.7.18.6 — 6 droplets, matches.

Total 33 — matches master claim (11 in master §3 prose).

PASS.

### 4.3 Cross-plan boundary coordination

Several boundaries that require synthesis-level resolution before builder dispatch:

- **F.7-CORE F.7.5 vs F.7.17.7** — `permission_grants` table cli_kind column ownership. Both plans correctly delegate: F.7-CORE F.7.5 ships the table schema + Go-side write/read; F.7.17.7 adds the cli_kind column. Master §10 Q3 routes to plan-QA-twin time. PASS — coordination explicit.
- **F.7-CORE F.7.4 vs F.7.17.9** — monitor refactor. F.7-CORE F.7.4 ships the dispatcher monitor; F.7.17.9 refactors to consume via adapter.ParseStreamEvent. F.7.17 plan §6.3 has explicit cross-planner coordination note. Master §10 Q9 routes to twins. PASS.
- **F.7.18.5 vs F.7-CORE F.7.16** — both edit `default.toml`. F.7.18.5 declares explicit `blocked_by: F.7.16.<final-droplet-id>` per sibling-build paths conflict rule. PASS — sibling-lock declared.
- **F.7.18.3 vs F.7-CORE spawn pipeline** (Q15/Q16) — production `GitDiffReader` adapter location + spawn-pipeline-calls-aggregator wiring. F.7.18 plan defaults to "F.7-CORE owns the wiring." Master §10 routes to plan-QA-twins. PASS — open-question routing explicit.

All cross-plan boundaries either resolved inline OR routed as Open Questions to plan-QA-twins.

---

## 5. R2-Confirmed Counterexample Mitigation Audit

Each R2-confirmed counterexample (A1.a denylist, A1.c regex pinning, A2.a TMPDIR/XDG, A2.b PATH inherit, A2.c POSIX-only, A2.d lowercase env, A3.a Tillsyn struct, A3.b schema-split, A4.a hard-cut, A4.b rename, A6 greedy-fit, A7 two-axis, A8.a MockAdapter) MUST appear baked into a specific droplet acceptance criterion.

| R2 attack | Verdict | Mitigation droplet | Acceptance criterion text |
|---|---|---|---|
| A1.a `sh -c` bypass | CONFIRMED-NEW | F.7.17.1 | "Closed denylist constant `shellInterpreterDenylist = []string{"sh","bash","zsh","ksh","dash","fish","tcsh","csh","ash","busybox","env","exec","eval","/bin/sh","/bin/bash","/usr/bin/env","python","python3","perl","ruby","node"}` lives in `internal/templates/`. `validateAgentBindingCommandTokens` rejects when `command[0]` matches any denylist entry (exact-match, no fold)." Tests cover ALL denylist entries. PASS. |
| A1.c MatchString anchors | CONFIRMED-NEW | F.7.17.1 | Implementation pinned EXACTLY: `var commandTokenRegex = regexp.MustCompile(\`^[A-Za-z0-9_./-]+$\`)` + `if !commandTokenRegex.MatchString(token) { /* reject */ }`. Tests: `["rm; ls"]` MUST fail; `["valid_token"]` MUST pass; `["valid; injected"]` MUST fail. PASS. |
| A2.a TMPDIR/XDG missing | CONFIRMED-NEW | Master L4 + F.7.17.3 | Master L4 baseline = `PATH, HOME, USER, LANG, LC_ALL, TZ, TMPDIR, XDG_CONFIG_HOME, XDG_CACHE_HOME` + per-binding `env`. F.7.17.3 acceptance: "`Env` set explicitly to the closed POSIX baseline (L6) PLUS the resolved values for every name in `br.Env`." PASS. |
| A2.b PATH ambiguity | CONFIRMED-NEW | Master L4 + F.7.17.3 | "PATH value is `os.Getenv("PATH")` (inherit-PATH; PATH itself doesn't carry secrets)." F.7.17.3 acceptance test: `cmd.Env` does NOT contain `AWS_ACCESS_KEY_ID` even when present in orchestrator's environment (proves L8 isolation). PASS. |
| A2.c platform silence | CONFIRMED-NEW | Master L3 + F.7.8 | Master L3 declares POSIX-only Drop 4c. F.7.8 acceptance: "PID liveness check is POSIX-only (cross-plan: F.7.17 plan declares POSIX-only scope per QA-R2 A2.c). Windows path returns wrapped error." PASS. |
| A2.d lowercase env regex | CONFIRMED-NEW | F.7.17.1 | "Each entry matches `^[A-Za-z][A-Za-z0-9_]*$` (lowercase allowed)." Test: `env = ["https_proxy"]` (lowercase — A2.d) MUST pass load. PASS. |
| A3.a Tillsyn struct missing from bundled droplet | CONFIRMED-NEW | F.7.18.2 | Dedicated droplet "F.7.18.2 — Schema-3: `[tillsyn]` top-level globals + validators" creates `Tillsyn` struct + `Template.Tillsyn` field with TOML tag `tillsyn`. Validators reject zero/negative. Unit test on `[tillsyn] foo = "bar"` MUST fail with `ErrUnknownTemplateKey`. PASS. |
| A3.b schema-bundle SPOF | CONFIRMED-NEW | Master L16 + 3 droplets | Schema-1 (F.7.17.1), Schema-2 (F.7.18.1), Schema-3 (F.7.18.2) — each ~1/3 review surface. Strict ordering enforced via cross-plan blocked_by. PASS. |
| A4.a ConsumeStream "additive" framing wrong | CONFIRMED-NEW | Master L17 + F.7.17.11 | Master L17 declares hard-cut. F.7.17.11 doc acceptance #7: "The hard-cut migration story for non-JSONL CLIs (L12, A4.a): future SSE / framed-binary CLI lands via a coordinated breaking interface rewrite — NOT add-then-deprecate. Document the exact migration sequence: rewrite interface, refactor all adapters + monitor in one drop, no compat shim." PASS. |
| A4.b ExtractTerminalCost rename | CONFIRMED-NEW | F.7.17.2 | "`CLIAdapter` interface declared with three methods (signatures pinned per L10): ... `ExtractTerminalReport(ev StreamEvent) (TerminalReport, bool)`." Test scenarios assert method named `ExtractTerminalReport` (not `ExtractTerminalCost`). PASS. |
| A6 greedy alternative not considered | CONFIRMED-NEW | Master L14 + F.7.18.4 | Master L14: "Greedy-fit bundle-cap algorithm. Iterate rules in TOML declaration order; rules that bust cap are SKIPPED with markers; subsequent rules continue if they fit." F.7.18.4 acceptance: "Greedy-fit test: cheap-1 (10KB) + busting-2 (220KB after per-rule trunc) + cheap-3 (50KB), bundle cap 200KB → cheap-1 lands, busting-2 SKIPPED with marker, cheap-3 lands." PASS — exact A6.b cross-check baked. |
| A7 per-rule timeout missing | CONFIRMED-NEW | Master L15 + F.7.18.4 | Master L15: "Two-axis wall-clock caps. Per-rule `max_rule_duration = "500ms"` (default) + per-bundle `max_aggregator_duration = "2s"` (default)." F.7.18.4 acceptance includes per-rule timeout test, per-bundle timeout test, AND outer-cancel-propagates test. PASS. |
| A8.a MockAdapter fixture missing | REFUTED-WITH-NIT | Master L19 + F.7.17.4 | Master L19: "`MockAdapter` test fixture in F.7.17 acceptance criteria. Confirms multi-adapter readiness pre-Drop-4d." F.7.17.4 dedicated droplet ships `mockAdapter` struct + contract conformance test using `[]CLIAdapter{newMockAdapter(...), claudeAdapter{}}`. PASS — A8.a NIT closed. |

13/13 R2-confirmed mitigations baked into specific droplet acceptance criteria. PASS.

---

## 6. Acceptance Criteria Testability Sweep

Sampled across all three sub-plans for "concrete behavior, not vague 'implements X'":

| Droplet | Acceptance criterion | Concrete? |
|---|---|---|
| F.7.1 | `MkdirBundle("abc-123", "os_tmp", "")` creates `os.TempDir()/tillsyn/abc-123/`; round-trip `WriteManifest(b, m1); m2, _ := ReadManifest(b.ManifestPath); reflect.DeepEqual(m1, m2)` | YES — file path + reflect.DeepEqual |
| F.7.2 | `[agent_bindings.build.sandbox.filesystem] unknown_key = "x"` MUST fail strict-decode; `allow_write = ["/etc"]` MUST fail with ancestry-check | YES — exact TOML payloads + sentinel errors |
| F.7.3 | Argv shape EXACTLY: `[--bare, --plugin-dir <bundle>/plugin, --agent <name>, --system-prompt-file <bundle>/system-prompt.md, --settings <bundle>/plugin/settings.json, --setting-sources "", --strict-mcp-config, --permission-mode acceptEdits, --output-format stream-json, --verbose, --no-session-persistence, --exclude-dynamic-system-prompt-sections, -p "<minimal-prompt>"]` + conditional flags | YES — byte-for-byte argv parity tests against fixed input |
| F.7.4 | Fixture `stream_simple.jsonl` (1 system/init + 1 assistant + 1 result) → 3 events parsed; terminal extracts `Cost = 0.006039`, no denials | YES — concrete fixture + exact cost value |
| F.7.5 | cli_kind discriminator test: claude grant for kind=build does NOT appear in codex spawn's render | YES — explicit cross-CLI isolation assertion |
| F.7.6 | `requires_plugins = []` (empty) → no error, no exec call; `claude plugin list --json` shell-out timeout 5s | YES — exact behavior + numeric timeout |
| F.7.7 | `EnsureSpawnsGitignored` checks 4 line-form variants (`.tillsyn/spawns/`, `.tillsyn/spawns`, `/.tillsyn/spawns`, `/.tillsyn/spawns/`); idempotent on second call | YES — exact form list + idempotency assertion |
| F.7.8 | PID-zero handling, cmdline mismatch detection, race-window leave-alone | YES — three concrete edge cases enumerated |
| F.7.9 | Doc-comment text MUST cite (a) audit-only role with literal text "this slice is an APPEND-ONLY AUDIT TRAIL of spawn lifecycle events. Consumers are ledger / dashboard renderers, NOT re-prompt aggregators."; (b) F.7.18 round-history-deferred pointer | YES — exact doc-comment text required |
| F.7.10 | Test asserts rendered prompt does NOT contain substring `hylla_artifact_ref`; `domain.Project.HyllaArtifactRef` field + storage round-trip preserved | YES — substring-absence assertion + field-existence assertion |
| F.7.11 | Each doc 200-500 lines, `mage check` markdown lint clean | PARTIAL (line-count heuristic + lint pass; per-droplet QA discipline rules carve-out MD-only droplets to orchestrator self-QA) — testability-by-discipline. Acceptable per master §7 carve-out. |
| F.7.12 | Empty start_commit → falls back to `HEAD~1..HEAD` diff; ≤72 chars conventional commit format `<type>(<scope>): <subject>` | YES — exact fallback chain + format spec |
| F.7.13 | Toggle off (nil) → `GateResult{Success: true, Skipped: true, Reason: "dispatcher_commit_enabled toggle off"}`; toggle on + path-scoped `git add <path>` for each path in `item.Paths`, NEVER `git add -A` | YES — three-state pointer-bool + add-scope-rule |
| F.7.14 | Push timeout 60s; on failure `metadata.BlockedReason = "git push: <stderr>"`; NO auto-rollback of local commit (verifiable via `git log` showing commit still on local branch) | YES — numeric timeout + blocked-reason format + rollback-not-performed assertion |
| F.7.15 | Helper accessor `IsDispatcherCommitEnabled() bool` returns `false` for nil OR `false`; returns `true` only when pointer is non-nil AND points to `true` | YES — three-state semantics |
| F.7.16 | `gates.build = ["mage_ci", "commit", "push"]` exact slice; nil toggles → 3 gates run, 2 skipped | YES — exact TOML config |
| F.7.17.1 | EVERY denylist entry has explicit reject test; happy + reject test matrix specified inline | YES — exhaustive test matrix |
| F.7.17.2 | Reflection test: `BindingResolved` has exactly the named fields above (catches accidental rename) | YES — reflection-asserted field contract |
| F.7.17.3 | `cmd.Env` does NOT contain `AWS_ACCESS_KEY_ID` even when present in orchestrator's environment (proves L8 isolation) | YES — explicit secret-isolation assertion |
| F.7.17.4 | Contract conformance test — table-driven over `[]CLIAdapter{newMockAdapter(...), claudeAdapter{}}` with same assertion suite | YES — polymorphism-proof assertion |
| F.7.17.5 | `cli_kind = ""` → claude adapter selected; `cli_kind = "codex"` → `ErrCodexAdapterNotImplemented`; `cli_kind = "bogus"` → `ErrUnknownCLIKind` | YES — full CLIKind matrix |
| F.7.17.6 | Read manifest missing `cli_kind` (legacy) → defaults to claude; orphan-scan with codex manifest → returns `ErrCodexAdapterNotImplemented` | YES — backward-compat + explicit-error |
| F.7.17.7 | `cli_kind TEXT NOT NULL` (no DEFAULT); insert with empty `cli_kind` fails NOT NULL | YES — exact column constraint + reject test |
| F.7.17.8 | Table-driven test covering every priority level for at least three fields | YES — explicit priority-cascade matrix |
| F.7.17.9 | Monitor has ZERO references to claude-specific event types (no `"system/init"`, `"assistant"`, `"result"` literals in monitor code) | YES — exact-string-grep assertion |
| F.7.17.10 | (Doc-only) Doc explicitly says `<project>/.tillsyn/config.toml` is NOT modifiable by `till template install` | YES — exact doc-content claim |
| F.7.17.11 | (Doc-only) ≤ 400 lines; covers 9 enumerated topics | YES — line-count + content-checklist |
| F.7.18.1 | `delivery = "stream"` MUST fail with `ErrInvalidContextRules`; `siblings_by_kind = ["bulid"]` (transposed) MUST fail with `ErrUnknownKindReference` | YES — exact reject sentinels |
| F.7.18.2 | Cross-cap warn path: a binding with `max_chars = 999999` + `[tillsyn] max_context_bundle_chars = 200000` LOADS but emits warn-line | YES — numeric values + warn-only behavior |
| F.7.18.3 | Per-binding-scope test: catalog has 11 other bindings with weird `[context]` blocks; `Resolve` for `kind=build` ignores them all | YES — explicit isolation assertion |
| F.7.18.4 | Outer-cancel propagates: per-bundle 500ms; per-rule 2s; rule blocks 1s → outer fires at 500ms, inner cancellation propagates, partial-rule output DISCARDED, marker emitted | YES — numeric timing + propagation assertion |
| F.7.18.5 | `tpl.AgentBindings[domain.KindBuild].Context.Parent == true` (and equivalents for the other 5 in-scope bindings); `tpl.AgentBindings[domain.KindCommit].Context` is the zero value | YES — assertion-by-field-comparison on default load |
| F.7.18.6 | (Doc-only) Doc-comment text required verbatim citing audit-only role + F.7.18 pointer | YES — exact doc-comment text required |

33/33 droplets carry concrete acceptance criteria. F.7.11 partial-by-design (MD-only droplet uses orchestrator self-QA + lint discipline per master §7 carve-out). PASS.

---

## 7. Open Questions Q1-Q17 Deferrability Audit

Each open question must be deferrable to plan-QA twins or builder time, not a planning blocker.

| ID | Question | Routing | Deferrable? |
|---|---|---|---|
| Q1 | F.7.2 droplet sizing — split sandbox validation? | plan-QA proof time | YES — sizing concern, not architectural |
| Q2 | Bundle cleanup timing on commit/push gate failure | plan-QA falsification time | YES — coordination concern, master DAG covers happy path |
| Q3 | F.7.5 `cli_kind` column ownership F.7-CORE vs F.7.17 | plan-QA proof time | YES — both plans declare delegated ownership; orchestrator picks at synthesis |
| Q4 | F.7.13 commit gate `git add` semantics on first build | plan-QA falsification time | YES — F.7.12 already specifies fallback; QA-falsification confirms |
| Q5 | F.7.16 default template gates listed-but-skipped vs only-mage_ci | plan-QA proof time | YES — F.7.16 acceptance declared; QA confirms shape |
| Q6 | F.7.6 plugin pre-flight cache vs always-fresh | plan-QA falsification time | YES — F.7.6 acceptance specifies always-fresh per Attack #7; QA confirms |
| Q7 | F.7.11 docs by orchestrator vs builder | plan-QA proof time | YES — master §7 + `feedback_md_update_qa.md` cover MD-only carve-out |
| Q8 | manifest.json widening cross-planner coordination | plan-QA proof time | YES — F.7.17 plan §6.1 cross-planner note explicit |
| Q9 | monitor refactor cross-planner coordination | plan-QA proof time | YES — F.7.17 plan §6.3 cross-planner note explicit |
| Q10 | `BindingResolved.Command` defaulting (split-default vs centralized resolver) | plan-QA debate at twin time | YES — implementation detail, both options yield equivalent end-state |
| Q11 | Allow-list location formalization | resolved | RESOLVED — master L7 chose install-time confirmation; replaces allow-list framing entirely |
| Q12 | F.7.17 Schema-1 droplet split granularity | plan-QA proof time | YES — sizing concern |
| Q13 | Marketplace install-time confirmation paper-spec vs functional | resolved | RESOLVED — F.7.17.10 explicitly paper-spec; F.4 owns CLI |
| Q14 | `metadata.spawn_history[]` doc-comment ownership | plan-QA twin time | YES — F.7.18.6 routes as "either standalone or absorbed into F.7.9"; F.7-CORE F.7.9 already absorbs |
| Q15 | Production `GitDiffReader` adapter location | builder time | YES — F.7.18 plan defaults to dispatcher root package |
| Q16 | Spawn-pipeline-calls-aggregator wiring location | plan-QA twin time | YES — F.7.18 plan defaults to "F.7-CORE owns the wiring" |
| Q17 | `Bundle.Files` filename convention | builder time | YES — F.7.18 plan defaults to per-rule deterministic name |

17/17 deferrable. None blocks PLAN authoring or builder dispatch. PASS.

---

## 8. Out-of-Scope §9 Identification

Master §9 lists deferred items. Cross-walked against SKETCH §G + dev decision splitting Drop 4c into 4c (F.7) + 4c.5 + 4d + 4d-prime:

| Master §9 item | SKETCH source | Verdict |
|---|---|---|
| F.1, F.2, F.3, F.5, F.6 (template ergonomics) | SKETCH §F.1-F.6 (Theme F template ergonomics) | PASS — master declares Drop 4c.5 (post-Drop-4c-merge) |
| Theme A (silent-data-loss + agent-surface hardening) | SKETCH §"Theme A" | PASS — Drop 4c.5 |
| Theme B (dev escape hatches) | SKETCH §"Theme B" | PASS — Drop 4c.5 |
| Theme C (STEWARD + cascade-precision refinements) | SKETCH §"Theme C" | PASS — Drop 4c.5 |
| Theme D (pre-cascade hygiene) | SKETCH §"Theme D" | PASS — Drop 4c.5 |
| Theme E (Drop-4a/4b residue) | SKETCH §"Theme E" | PASS — Drop 4c.5 |
| F.4 (marketplace CLI) | SKETCH §F.4 | PASS — Drop 4d-prime, post-Drop-5 |
| Theme G (post-MVP marketplace evolution) | SKETCH §"Theme G" | PASS — post-MVP |
| Codex adapter | SKETCH §F.7.17 + §"Drop 4d preview" | PASS — Drop 4d, post-Drop-5 |
| Drop 4.5 TUI overhaul | SKETCH §"Naming" + §"Goal" | PASS — concurrent FE/TUI track |
| Drop 5 dogfood validation | SKETCH §"Naming" | PASS — post-Drop-4d |

11/11 §9 items correctly identified as out of scope. PASS.

---

## 9. Pre-MVP Rules In-Force Audit

Master §6 enumerates pre-MVP rules. Verified against `feedback_*.md` memories + project CLAUDE.md:

| Rule | Source | Verified |
|---|---|---|
| No migration logic in Go | `feedback_no_migration_logic_pre_mvp.md` | PASS |
| No closeout MD rollups (LEDGER/WIKI_CHANGELOG/REFINEMENTS/HYLLA_FEEDBACK) | `feedback_no_closeout_md_pre_dogfood.md` | PASS |
| Opus builders | `feedback_opus_builders_pre_mvp.md` | PASS — every droplet declares "Builder model: opus" |
| Filesystem-MD mode | `feedback_no_closeout_md_pre_dogfood.md` companion | PASS |
| Tillsyn-flow + Section 0 SEMI-FORMAL REASONING | `feedback_section_0_required.md` | PASS |
| Single-line conventional commits ≤72 chars | `feedback_commit_style.md` + project CLAUDE.md | PASS — F.7.12 acceptance bakes the rule |
| NEVER raw `go test`/`go build`/`go vet`/`mage install` | project CLAUDE.md "Build Verification" §3 | PASS — master L22 explicit |
| Hylla is Go-only today | `feedback_hylla_go_only_today.md` | PASS — master §6 covers MD-only sweep fallback |

8/8 pre-MVP rules in force. PASS.

---

## 10. NITs (Cosmetic, Not Blockers)

### 10.1 NIT-1: L20 / F.7.15 cite "Drop 4a Wave 4a.25 precedent" — actual provenance is **Drop 4a Wave 3 W3.2**

Master §3 L20 says: "Commit + push gates default OFF via `dispatcher_commit_enabled` + `dispatcher_push_enabled` project metadata pointer-bools." F.7-CORE plan F.7.15 acceptance also cites "Drop 4a 4a.25 precedent."

The actual pointer-bool semantics with three-state nil/false/true + accessor pattern lives in `internal/domain/project.go:119-145` as `OrchSelfApprovalEnabled *bool` + `OrchSelfApprovalIsEnabled() bool` — provenance tagged "Drop 4a Wave 3 W3.2" in the godoc (lines 100-114, 136-145).

**Impact:** Citation drift only; the design pattern is correctly applied. The actual semantic (3-state pointer-bool with `Is*Enabled()` accessor) matches the W3.2 precedent verbatim.

**Recommended fix:** Optional next-revision retitle to "Drop 4a Wave 3 W3.2 precedent" in master L20 + F.7.15 acceptance. NOT a builder-blocker; W3.2 is the pattern source, 4a.25 is an unrelated wave label. If left as-is, no functional impact at builder time — the builder will naturally find the correct precedent when implementing the field.

### 10.2 NIT-2: F.7.18 plan cites `internal/app/git_status.go:146-156` — function declaration at `:146` is correct, but doc-comment starts at `:141`

F.7.18 plan §"References" cites `internal/app/git_status.go:146-156` for `filteredGitEnv`. Verified: the godoc comment block starts at line 141 ("`filteredGitEnv returns os.Environ() with every GIT_*=... entry removed.`"); function declaration at line 146; closing brace at line 156.

**Impact:** Off-by-5 lines on the cite envelope start. Functionally identical lookup target. Minor.

**Recommended fix:** Optional next-revision tighten to `:141-156` to include the doc-comment context. NOT a blocker.

---

## 11. Hylla Feedback

`N/A — review touched non-Go files only` (master PLAN.md, three sub-plan MDs, SKETCH.md, planner-review MD, R2 proof MD, R2 falsification MD). Code-anchor cross-walks were verified via direct `Read` on `internal/templates/schema.go`, `internal/templates/load.go`, `internal/domain/kind.go`, `internal/domain/project.go`, `internal/domain/action_item.go`, `internal/app/dispatcher/spawn.go`, `internal/app/dispatcher/spawn_test.go`, `internal/app/dispatcher/monitor.go`, `internal/templates/builtin/default.toml`, `internal/app/git_status.go` per spawn-prompt directive ("No Hylla calls — Hylla stale post-Drop-4b-merge"). No Hylla queries issued; no miss to report.

---

## 12. Verdict

**PROOF GREEN-WITH-NITS.**

- All 22 master L-decisions traceable to a numbered source (SKETCH / planner-review P-ID / R2 falsification A-ID / dev decision / project CLAUDE.md).
- All 13 R2-confirmed counterexample mitigations baked into specific droplet acceptance criteria (A1.a denylist, A1.c regex pinning, A2.a TMPDIR/XDG, A2.b PATH inherit, A2.c POSIX-only, A2.d lowercase env regex, A3.a Tillsyn struct, A3.b schema-split, A4.a hard-cut, A4.b rename, A6 greedy-fit, A7 two-axis, A8.a MockAdapter).
- 33 F.7 droplets (16 + 11 + 6) covered across master + three sub-plans; matches master §3 prose claim.
- All 17 Open Questions correctly classified as deferrable to plan-QA twins or builder time, two already resolved (Q11, Q13).
- Cross-plan dependency coherence verified (Schema-1 → Schema-2 → Schema-3 sequencing; sibling-lock declared on F.7.18.5 ↔ F.7.16; cross-planner coordination notes explicit on F.7.17.6 / F.7.17.9 / F.7.18.6).
- All 11 §9 out-of-scope items correctly identified per SKETCH partition.
- All 8 pre-MVP rules in force.
- Two NITs surfaced (cosmetic citation drift on L20/F.7.15 W3.2 vs 4a.25; F.7.18 plan filteredGitEnv:146-156 cite envelope off-by-5). Neither blocks builder dispatch.

**Ready for builder fire** after orchestrator-level cross-planner synthesis on Q3 (F.7-CORE F.7.5 vs F.7.17.7 cli_kind column ownership), Q11 already-resolved, and Q14-Q17 (F.7-CORE/F.7.18 wiring boundaries, all routed to plan-QA-twin time per the plans' own routing).

---

## TL;DR

- T1: Verdict is PROOF GREEN-WITH-NITS — all R2-confirmed counterexamples mitigated, all L1-L22 traceable, all 33 droplets covered with concrete acceptance criteria.
- T2: 13/13 code-anchor claims verified against the local checkout (Template, AgentBinding, strict-decode chain, Kind enum, Project.HyllaArtifactRef field, spawn.go assemblePrompt lines 211-213, gates.build = ["mage_ci"]). Two off-by-N-line NITs.
- T3: All 22 L-decisions trace to SKETCH / planner-review P-ID / R2 A-ID / dev decision / project CLAUDE.md source. NIT-1 on L20 citation only (4a.25 vs W3.2 — semantics correct).
- T4: Cross-plan sequencing (Schema-1 → Schema-2 → Schema-3) coherent; cross-planner coordination notes explicit; sibling-lock declared between F.7.18.5 and F.7.16.
- T5: 13/13 R2-confirmed mitigations baked into specific droplet acceptance criteria (A1.a denylist + A1.c regex pinning + A2.a TMPDIR/XDG + A2.b PATH + A2.c POSIX + A2.d lowercase + A3.a Tillsyn struct + A3.b schema-split + A4.a hard-cut + A4.b rename + A6 greedy + A7 two-axis + A8.a MockAdapter).
- T6: 33/33 droplets carry concrete acceptance criteria (file paths, exact TOML payloads, sentinel errors, byte-for-byte argv parity, exact doc-comment text, numeric timeouts/sizes). One MD-only carve-out per master §7 discipline rule.
- T7: 17/17 Open Questions correctly classified as deferrable; two already resolved (Q11 install-time confirmation supersedes allow-list, Q13 paper-spec). None block dispatch.
- T8: Master §9 out-of-scope partition matches SKETCH (F.1-F.3 + F.5-F.6 + Themes A-E → Drop 4c.5; F.4 → Drop 4d-prime; codex → Drop 4d; Drop 4.5 + Drop 5 separate tracks). 11/11 PASS.
- T9: All 8 pre-MVP rules in force (no migration logic, no closeout MD rollups, opus builders, filesystem-MD mode, Section 0, single-line commits, no raw go-tool / mage-install, Hylla Go-only).
- T10: Two cosmetic NITs (L20 citation drift; filteredGitEnv line offset) — both auditable, neither blocks builder dispatch.
- T11: Hylla Feedback N/A — review touched non-Go files only; code anchors verified via direct `Read` per spawn-prompt directive.
- T12: Ready for builder fire after orchestrator cross-planner synthesis at Q3/Q14-Q17 (already routed to plan-QA-twin time by the plans themselves).
