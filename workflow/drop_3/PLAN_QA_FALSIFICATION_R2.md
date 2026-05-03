# DROP 3 — UNIFIED PLAN QA FALSIFICATION (Round 2)

**Reviewer:** `go-qa-falsification-agent` (subagent)
**Working dir:** `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/`
**Target:** `workflow/drop_3/PLAN.md` `## Planner` section (28 droplets, 3.1 – 3.28).
**Round:** 2 (post-Round-1 unification)
**Date:** 2026-05-02
**Verdict:** **PASS-WITH-MITIGATION-REQUIRED** — 1 HIGH, 4 MEDIUM, 5 LOW counterexamples; Round 1 closure ~92% (28/30 prior CEs cleanly closed; 2 partially-closed). No CONFIRMED build-stopping counterexample remains, but two missing-blocker findings would weaponize parallel dispatch in Drop 4.

---

## 1. Counterexamples

### 1.1 — CONFIRMED (HIGH) — Missing cross-unit blocker `3.20 blocked_by 3.14`

- **Hit.** Droplet 3.20.
- **Claim.** 3.20 `Blocked by: 3.11 (per finding 5.C.2 — the Template.ChildRulesFor engine droplet); 3.19.` (PLAN.md:485).
- **Counterexample.** 3.20's first declared path is **"TOML body in `internal/templates/builtin/default.toml`** (the file landed in 3.14)" (PLAN.md:468). The droplet edits the same TOML file 3.14 creates. The cross-unit blocker matrix at PLAN.md:629-639 also omits this edge. If a dispatcher fires 3.20 the moment 3.11 + 3.19 close (per the explicit blockers), `internal/templates/builtin/default.toml` may not yet exist — 3.20's edits become a write-to-non-existent-file. Topological-sort listed at PLAN.md:643 happens to put 3.14 before 3.20 by accident (3.14 is in the 3.8-3.16 Unit B chunk, 3.20 in the 3.17-3.22 Unit C chunk), but the actual `Blocked by` declaration is what a Drop 4 dispatcher reads — not the prose order.
- **Repro.** Drop 4 dispatcher reads `Blocked by: 3.11, 3.19` literally. After 3.11 + 3.19 complete and before 3.14 completes, dispatcher fires 3.20. Builder edits a missing file. Same risk pre-Drop-4: if the orchestrator dispatches builders manually but consults the `Blocked by` field rather than the topological-sort prose, 3.20 fires before 3.14.
- **Mitigation.** Add `3.14` to droplet 3.20's `Blocked by:` field (PLAN.md:485) AND add the row `| 3.14 | 3.20 | 3.20 edits internal/templates/builtin/default.toml created by 3.14. |` to the cross-unit blocker matrix (PLAN.md:629).
- **Severity rationale.** HIGH because it's a load-bearing blocker that the dispatcher's primary contract reads. Topological-prose-only enforcement is exactly what methodology §9.2 + plan-QA §4.4 attack vector (i) (acyclic blocker graph) attacks.

### 1.2 — CONFIRMED (MEDIUM) — Same-package serialization missing for 3.10 ↔ 3.11

- **Hit.** Droplets 3.10 and 3.11.
- **Claim.** 3.10 `Blocked by: 3.8, 3.9.`; 3.11 `Blocked by: 3.8, 3.9. Cross-unit blocked by 3.1.` (PLAN.md:288, 302). Parallelism note (PLAN.md:645): *"Within Unit B, 3.10 + 3.11 (post-3.9) are package-disjoint at file level but both compile against `internal/templates` — sequenced for deterministic ordering."*
- **Counterexample.** Methodology §2.2 (cited verbatim by L12): *"Droplets sharing a package: serialize with explicit `blockers` between them. A package is one compile unit; parallel builders on the same package would trip over each other's test runs."* The plan SAYS 3.10 + 3.11 are "sequenced for deterministic ordering," but no `blocked_by` between 3.10 and 3.11 enforces that. A dispatcher reading `Blocked by` literally fires both in parallel after 3.9. Both compile against `internal/templates` — concurrent builds race on test compilation. Same risk applies between every adjacent pair within 3.8 → 3.14 in `internal/templates`: 3.8 ↔ 3.13 the plan correctly catches via "file-level: same schema.go" (PLAN.md:335), but 3.10 ↔ 3.11 ↔ 3.12 are all in the same package and not file-locked.
- **Repro.** Dispatcher fires 3.10 + 3.11 concurrently after 3.9 closes. Both run `mage test-pkg ./internal/templates`. The package compile sees both new files; if either has a compile error, the other's test run goes red on a defect it didn't introduce. Plan-QA §4.4 attack vector (ii) — sibling overlap without explicit blocker — fires.
- **Mitigation.** Either (a) add `3.11 blocked_by: 3.10` (deterministic Unit B chain), OR (b) widen the parallelism prose to commit explicitly: *"Dispatcher MUST serialize 3.10 → 3.11 → 3.12 → 3.13 → 3.14 within Unit B (shared `internal/templates` Go package)."* Option (a) is the cleaner mechanical fix — same shape Round 1 took for 3.7 → 3.23 → 3.27 on `go-qa-falsification-agent.md`.
- **Severity rationale.** MEDIUM (not HIGH) because (i) `internal/templates` is a brand-new package with no existing test surface to race on, (ii) post-3.9 the plan-QA §4.4 sweep this drop ships in 3.7 will catch this on the next plan-QA run, and (iii) the plan author flagged the issue in prose. The defect is the prose-vs-blocker mismatch; any builder reading the prose lands clean. The dispatcher reading the schema does not.

### 1.3 — CONFIRMED (MEDIUM) — Test-fixture cascade enumeration is missing one file

- **Hit.** Droplet 3.2.
- **Claim.** PLAN.md:143-144: *"Cross-package sweep enumerated: `internal/domain/domain_test.go`, `internal/app/service_test.go`, `internal/adapters/storage/sqlite/repo_test.go`, `internal/app/snapshot_test.go`, `internal/app/attention_capture_test.go`, `internal/app/dotted_address_test.go`, `internal/app/embedding_runtime_test.go`, `internal/adapters/storage/sqlite/embedding_jobs_test.go`, `cmd/till/embeddings_cli_test.go`, `internal/tui/model_test.go`."*
- **Counterexample.** `grep -rln NewActionItem --include='*.go' internal/ cmd/` returns 13 files. The plan enumerates 10 test files. Missing from the enumerated list:
  - `internal/app/service.go` — production caller (1 site).
  - `internal/domain/action_item.go` — definition site only (no fixture; fine).
  - `internal/adapters/server/common/mcp_surface.go` — comment-only reference (line 64 says `domain.NewActionItem` in a doc-comment); fine.
  - The production caller `internal/app/service.go` is the load-bearing miss — it is NOT a test file but does call `domain.NewActionItem(...)`. Per Round 1 Unit A C1's count table, that's "1 production caller" — still must be migrated to supply a valid `StructuralType` (or the production code path itself is dead until 3.4 lands `app.CreateActionItem` threading).
- **Repro.** Builder of 3.2 reads PLAN.md:143-144, edits the 10 enumerated test files + the helper. `mage ci` then fails on `internal/app/service.go` because the production code path constructs an `ActionItemInput` without `StructuralType` and `NewActionItem` rejects with `ErrInvalidStructuralType`.
- **Mitigation.** Add `internal/app/service.go` to 3.2's enumerated sweep list (and confirm the production caller's `ActionItemInput` literal supplies a default `StructuralType`). 3.4's plumbing `app.CreateActionItem(:574)` already threads `StructuralType: in.StructuralType` per PLAN.md:177 — but if 3.2 lands before 3.4, the production caller in `service.go` is broken until 3.4 ships.
- **Severity rationale.** MEDIUM — `mage ci` (the upgraded gate per CE1 mitigation) catches it inside 3.2's QA round, so the failure surfaces in-droplet rather than cascading. But it forces a re-spawn or pre-emptive 3.4-coupling. Deserves a fix in the plan, not just at builder time.

### 1.4 — CONFIRMED (MEDIUM) — Methodology §2.2 per-package QA gate not modeled in the cascade

- **Hit.** Drop-level — methodology gap, not droplet-level.
- **Claim.** Plan claims feature-parity with `ta-docs/cascade-methodology.md` §11 (PLAN.md:60). `Methodology Integration` block (PLAN.md:62-72) lands four methodology fields (`Persistent`, `DevGated`, `Irreducible`, the §4.4 attack vector).
- **Counterexample.** Methodology §2.2: *"Droplets sharing a package: serialize with explicit `blockers` between them."* Methodology §4.1 (Package-Level Build+Test): *"Every package that received droplet edits runs one build+test pass after **all droplets targeting that package** have reported complete. No LLM. No judgment."* Drop 3's plan does NOT model the package-level build+test gate as a first-class node — it relies on per-droplet `mage test-pkg <pkg>` and unit-boundary `mage ci`. Methodology says each package gets ONE build+test pass after all droplets targeting it complete. With six droplets (3.8-3.14) all targeting `internal/templates`, the methodology says one gate at the END of all six, not six independent gates.
- **Repro.** Drop 4 dispatcher implements methodology faithfully → expects a per-package gate node before each planner-level build-QA twin. Drop 3's schema has no field/node for this. Drop 4 either re-introduces it (rework) or runs in non-conformant mode.
- **Mitigation.** Either (a) add a §"Deferred to Drop 4" entry naming the package-level gate explicitly, OR (b) add an `Irreducible` flag carve-out for `Persistent` package-level QA gates as a methodology-compliance note. Option (a) is cleaner — the deferred list at PLAN.md:74 already mentions "per-kind droplet ceilings"; add "package-level build+test gate as first-class node" alongside.
- **Severity rationale.** MEDIUM — Drop 4 is the consumer; the gap doesn't break Drop 3's deliverables. But the plan's claim of methodology feature-parity (PLAN.md:62) is overstated, and a Drop 4 builder reading Drop 3's schema will not find the package-level gate primitive the methodology requires.

### 1.5 — CONFIRMED (MEDIUM) — Refinements-gate forgetfulness regression test (3.22 Test 7) tests notification only, not actual breakage

- **Hit.** Droplet 3.22 Test 7.
- **Claim.** PLAN.md:515: *"Test 7 — Refinements-gate forgetfulness regression (per finding 5.C.11): drop-orch creates a mid-drop refinement plan-item for `drop_number=3` AFTER the gate is created. Drop-orch forgets to manually update the gate's `blocked_by` list. Gate close fires anyway. Test asserts: an `attention_item` is created on gate-close warning the dev that `drop_number=3` items remained in_progress when the gate closed. Documents the failure mode rather than papering over it (per 5.C.11 ACCEPT-with-warning resolution)."*
- **Counterexample.** The test asserts the WARNING fires. It does NOT assert that the actual breakage manifests downstream. Per Round 1 Unit C C6 trace: *"DROP_3 closes prematurely with an in-progress child."* The dangerous failure mode is not "no warning" — it's "level_1 closes premature." The Round 2 test verifies the safety-net (warning), not the underlying broken behavior. A future change that causes the warning to misfire while ALSO regressing parent-blocks-on-incomplete-child would slip past this test.
- **Repro.** A regression to Drop 1's parent-blocks-on-incomplete-child rule would let `DROP_3` close while `DROP_3_DEPS_FIX` is `in_progress`. Test 7 still passes (warning still fires). The actual data-corruption case is not covered.
- **Mitigation.** Either (a) extend Test 7 with a sub-assertion: *"`DROP_3` close attempt with mid-drop child still `in_progress` is REJECTED by parent-blocks-on-incomplete-child (Drop 1 invariant)."* This pins the Drop 1 invariant alongside the warning; OR (b) accept that 5.C.11's resolution explicitly chose "warning only" and document this gap as known-deferred under the "Out Of Scope Confirmations" block.
- **Severity rationale.** MEDIUM — the bug shape is "test passes but underlying failure mode regresses." Drop 1's invariant is the actual safety net; Test 7 should pin both layers. Round 1 surfaced the failure mode; Round 2's mitigation tests only the surface symptom.

### 1.6 — CONFIRMED (LOW) — Cross-unit blocker matrix row for 3.5/3.4 has confusing column order

- **Hit.** PLAN.md:633 — table row for `3.5` ↔ `3.4`.
- **Claim.** *"| 3.5 | 3.4 | Same-package serialization (`internal/app`) per finding 5.A.2 — explicit. (Reverse-listed; effectively 3.5 ← 3.4 chain.) |"*
- **Counterexample.** The table header (PLAN.md:628) is `| Blocker (must complete) | Blocked droplet | Reason |`. Per droplet 3.5's actual `Blocked by: 3.4` (PLAN.md:198), the blocker is **3.4** and the blocked droplet is **3.5**. The table row places `3.5` in the Blocker column and `3.4` in the Blocked column — REVERSE of every other row. The parenthetical "Reverse-listed; effectively 3.5 ← 3.4 chain" acknowledges the inversion but doesn't fix it. A Drop 4 dispatcher consuming this table programmatically would set `3.5` as a blocker FOR `3.4`, creating a phantom edge and likely a cycle when combined with droplet 3.4's `Blocked by: 3.2`.
- **Repro.** Mechanical: parse table, build edge list, observe direction inversion.
- **Mitigation.** Swap the columns: `| 3.4 | 3.5 | Same-package serialization (`internal/app`) per finding 5.A.2. |`. Drop the parenthetical.
- **Severity rationale.** LOW — prose-level tables are reviewed by humans pre-Drop-4, and the parenthetical disclaimer reduces production-time risk. Still a real defect for any tool consuming the matrix mechanically.

### 1.7 — CONFIRMED (LOW) — Droplet-shape ceiling violations not flagged as `Irreducible`

- **Hit.** Droplets 3.15, 3.19, 3.21, 3.27.
- **Claim.** L11 (PLAN.md:68): *"Marks single-function-signature changes, single SQL migrations, single template edits — droplets that cannot decompose further. Plan-QA-falsification validates the claim per the methodology rule (\"Planners default to decompose; irreducibility is the exception, not an escape hatch\")."* Methodology §2.1: ~80 LOC + 3-file soft ceiling.
- **Counterexample.** Walking the heaviest droplets:
  - **3.15** — paths span 14+ files across 6 packages: `internal/domain/kind.go`, `kind_capability_test.go`, `domain_test.go`, `internal/app/kind_capability.go`, `kind_capability_test.go`, `snapshot.go`, `internal/adapters/server/common/mcp_surface.go`, `app_service_adapter_mcp.go`, `internal/adapters/server/mcpapi/extended_tools.go`, `instructions_explainer.go`, `extended_tools_test.go`, `internal/adapters/storage/sqlite/repo.go`, `repo_test.go`, `cmd/till/main.go`, `internal/tui/model.go`, `model_test.go`, plus MCP wire surface deletions. Far past 3-file ceiling. NOT marked `Irreducible`.
  - **3.19** — 5 paths across 4 packages, 5 distinct concerns (steward principal, Move gate, Update field-guard, Reparent gate, supersede, autent boundary-map). Note explicitly says: *"Atomic landing — splitting risks an intermediate compile state where the enum recognizes `steward` but no enforcement consults the value"* (PLAN.md:462). This is a textbook `Irreducible = true` justification — but the droplet doesn't use the field.
  - **3.21** — 6 paths across 3 packages.
  - **3.27** — sweep across ~25 doc paths.
- **Repro.** Plan-QA falsification's new attack vector (the "irreducible-claim attack" L11 cites) requires every oversized droplet to either (a) decompose or (b) carry `Irreducible = true` with justification. None do. The plan itself (PLAN.md:68) introduces `Irreducible` as a domain field landing in 3.2 but never USES the field on any droplet.
- **Mitigation.** For 3.15, 3.19, 3.21, 3.27: either decompose (preferred per methodology), or set `Irreducible = true` with justification in the droplet description prose. 3.19's "atomic landing" prose is already the justification — promote it to `Irreducible: true`. 3.15 is genuinely decomposable (per-package or per-symbol-class chunks); decompose it. 3.21 follows 3.19's atomic-landing pattern; same fix. 3.27 is doc-only; decompose by file class (active canonical docs vs. agent files vs. memory files) or accept oversize for doc sweeps as a known pattern.
- **Severity rationale.** LOW — the field is reserved (3.2) but consumed in a future drop's plan-QA-falsification attack. Drop 3's own pass through this very attack vector (per L12) doesn't apply to its own droplets retroactively. Still worth fixing for methodology compliance.

### 1.8 — CONFIRMED (LOW) — `~/.claude/CLAUDE.md` retired-vocab sweep enumerates 3 hits but plan describes 4

- **Hit.** Droplet 3.27.
- **Claim.** PLAN.md:583: *"Three known retired-vocab hits enumerated: line 9 `slice-by-slice` → drop or rephrase; lines 10, 121, 147 `build-task` → `build`."*
- **Counterexample.** Reading `~/.claude/CLAUDE.md` directly: line 9 `slice-by-slice`; line 10 `build-task`; line 121 `build-task` (in section "QA Discipline"); line 147 `build-task` (in section "Build Verification"). That's **4 hits**, not "three known" — line 9 is one hit, lines 10/121/147 are three more = 4 total. The prose at PLAN.md:583 is internally inconsistent: it says "three known" but enumerates four (1 slice + 3 build-task). The fix in 3.27 will land cleanly because the four sites are all listed; the Round-2 prose is just numerically wrong.
- **Repro.** Read PLAN.md:583 and the source file. Numerals don't match.
- **Mitigation.** Update PLAN.md:583 to *"Four known retired-vocab hits enumerated: line 9 `slice-by-slice` → drop or rephrase; lines 10, 121, 147 `build-task` → `build`."* Or restructure as "1 + 3."
- **Severity rationale.** LOW — purely an accounting nit. Builder running 3.27 sees the actual line numbers and edits the correct sites.

### 1.9 — CONFIRMED (LOW) — Plan does not require Hylla MCP for the planner's own evidence pass

- **Hit.** Methodology process — applies to Round 2 planner.
- **Claim.** Plan does not include any "evidence-source" directive for the Round 2 planner.
- **Counterexample.** CLAUDE.md §"Code Understanding Rules" rule 1: *"All Go code: use Hylla MCP as the primary source for committed-code understanding."* Round 1 Unit A falsification §5.A.12 / C7 explicitly logged: *"Hylla-first evidence sourcing process gap. Future planner spawns use Hylla MCP first per CLAUDE.md §Code Understanding Rules."* The Round 2 planner's Section 0 reasoning (which does not survive into the artifact, per the canonical rule) cannot be inspected to verify Hylla-first compliance. The PLAN.md artifact itself does not show evidence of Hylla queries (no symbol IDs cited from Hylla, no `hylla_node_full` results, no Hylla call-site blast radius from `hylla_graph_nav`). All cited line numbers are LSP / Read style — same pattern as Round 1.
- **Repro.** Inspect PLAN.md for any artifact of Hylla-first sourcing — none present.
- **Mitigation.** Process-level fix only: add a directive to future planner spawn prompts requiring Hylla MCP as the primary evidence source AND requiring Hylla query references inline in droplet descriptions when Hylla was the discovery tool. Drop 3 doesn't need this changed — the cited line numbers and call sites verify out via direct re-reads — but the process gap persists across rounds.
- **Severity rationale.** LOW — process nit, not a plan-correctness issue. The plan's facts are correct; the methodology of arriving at them is non-conformant. Drop 3 itself has no Hylla-blocking risk.

---

## 2. Round 1 Counterexample Closure Verification

Walking each Round 1 CE/finding against Round 2's mitigation. Closure status: **C** (Cleanly closed) / **P** (Partially closed — re-surfaced) / **A** (ACCEPTed-with-warning, route to deferred list).

### Unit A (Cascade Vocabulary Foundation)

| Round 1 ID | Severity | Closure | Notes |
| --- | --- | --- | --- |
| C1 (5.A.7) | HIGH | **C** | Round 2 introduces `newActionItemForTest` helper at 3.2 + upgrades gate to `mage ci` + enumerates 10 sweep files. **§1.3 above re-surfaces a sub-issue** (`internal/app/service.go` production caller missing from enumeration), but the structural fix lands. |
| C2 (5.A.7-precedent) | — | **C** | Drop 1.75 precedent at `internal/tui/model_test.go:14674-14687` cited explicitly in 3.2 acceptance (PLAN.md:133). Helper exists in tree; confirmed by `grep newActionItemForTest internal/tui/model_test.go`. |
| C3 (5.A.8) | HIGH | **C** | Three-way write conflict on `go-qa-falsification-agent.md` resolved via chain 3.7 → 3.23 → 3.27 (PLAN.md:240, 535, 602, mechanical fix 4.6). |
| C4 (5.A.10) | NIT | **C** | Regex rationale lock decision: builder may tighten to `[a-z]+` or keep `[a-z-]+` and document — explicit choice (PLAN.md:123). |
| C5 (5.A.9) | HIGH | **C** | `stubExpandedService.CreateActionItem` defaults `args.StructuralType` to `StructuralTypeDroplet` when empty (PLAN.md:182). Matches Role rejection-pattern stub-laxity precedent. |
| C6 (5.A.11) | NIT | **C** | "Confirmed" item moved to Locked Architectural Decisions block (PLAN.md:40-58 + note 5.A.11 reference). |
| C7 (5.A.12) | NIT-process | **P** | **§1.9 above re-surfaces** — process gap persists; Round 2 didn't add a planner-spawn directive. Plan correctness is unaffected. |
| 5.A.1 | BLOCKING | **C** | Fictional INSERT/UPDATE cites corrected to `:1253` / `:1347` only; SELECT projection sweep at `:1414, :1452, :2500` (PLAN.md:158). |
| 5.A.2 | BLOCKING-OR-DOC | **C** | `3.5 blocked_by 3.4` explicit (PLAN.md:198, mechanical fix 4.2). |
| 5.A.3 | NIT | **C** | Regex-comment caveat captured (PLAN.md:125). |
| 5.A.4 | NIT | **C** | Sweep enumeration captured (10 files at PLAN.md:144). **§1.3 above re-surfaces** sub-issue. |
| 5.A.5 | NIT | **C** | WIKI placement disambiguation locked: line 34/36 sibling h2 (PLAN.md:206). |
| 5.A.6 | NIT | **C** | Migration block cite corrected to `line 518` (PLAN.md:162). |

### Unit B (Template System Overhaul)

| Round 1 ID | Severity | Closure | Notes |
| --- | --- | --- | --- |
| CE1 (5.B.6) | HIGH | **C** | TOML moved under `internal/templates/builtin/default.toml`; embed.go colocated; no `..` segments (L4, PLAN.md:50, 343). |
| CE2 (5.B.7) | HIGH | **C** | "Starting-point lower bound" disclaimer + LSP-found additional sites enumerated (PLAN.md:368-374, mechanical fix 4.7). |
| CE3 (5.B.8) | HIGH | **C** | `TestRepositoryFreshOpen…` (`:2470-2517`) and `TestRepositoryFreshOpenKindCatalogUniversalParentAllow` (`:2520-2568`) explicitly DELETED in 3.15; equivalent assertions move to `embed_test.go` per 3.14 (PLAN.md:351, 371). |
| CE4 (5.B.9) | MEDIUM | **C** | L5 lock: `KindCatalogJSON json.RawMessage` on `Project`; decoding lives in `internal/app` or `internal/templates`, never on `Project`'s methods (PLAN.md:52, 310). |
| CE5 (5.B.10) | MEDIUM | **C** | Schema-version pre-pass spec'd: tolerant pre-pass → version-check → strict-decode (PLAN.md:267). |
| CE6 (5.B.11) | MEDIUM | **C** | `GateRule` struct + behavior deferred to Drop 4 (L6, PLAN.md:54, 252). |
| CE7 (5.B.12) | MEDIUM | **C** | Hand-coded `Template` value used as 3.10 fixture; 3.14's `embed_test.go` independently asserts loaded `default.toml` round-trips against same fixture (PLAN.md:283, 350). |
| CE8 (5.B.13) | HIGH | **C** | `till.kind operation=upsert` MCP wire, `till.upsert_kind_definition` legacy alias, `till kind` mutating CLI all DELETED in 3.15 (PLAN.md:374). Read-only `till kind list/get` carve-out for Drop 4 if needed. |
| N1 (5.B.14) | LOW | **C** | Runtime-mutability spec'd: edits ignored until dev fresh-DBs (PLAN.md:315). |
| N2 (5.B.15) | LOW | **C** | Rejection-comment scope narrowed to auth-gated creates (PLAN.md:391). |
| N3 (5.B.16) | LOW | **C** | Explicit `[child_rules]` deny rows forced (N3 explicit-deny, PLAN.md:346). |
| N4 (5.B.17) | LOW | **C** | 3.8 ships skeletal `AgentBinding`; 3.13 fills validation. Builder doesn't read 3.8 to decide (PLAN.md:329). |

### Unit C (STEWARD Auth + Auto-Gen)

| Round 1 ID | Severity | Closure | Notes |
| --- | --- | --- | --- |
| C1 (5.C.6) | HIGH | **C** | `UpdateActionItem` field-level write guard (L1) lands in 3.19 (PLAN.md:44, 444). |
| C2 (5.C.7) | HIGH | **C** | `ReparentActionItem` gated identically to `MoveActionItem` (L8) lands in 3.19 (PLAN.md:58, 445). |
| C3 (5.C.8) | HIGH | **C** | Autent boundary-map `steward → autentdomain.PrincipalTypeAgent` at adapter boundary (L2) lands in 3.19 (PLAN.md:46, 440). |
| C4 (5.C.9) | MEDIUM | **C** | `MoveActionItem` (column-only) gets `GetActionItem` pre-fetch; gate fires after fetch (5.C.1, PLAN.md:443). |
| C5 (5.C.10) | HIGH | **C** | STEWARD persistent parents seeded via default template's `[child_rules]` (L3) at 3.14 (PLAN.md:48, 348). |
| C6 (5.C.11) | MEDIUM | **A** | ACCEPT-with-warning per Round 1 resolution; **§1.5 above re-surfaces** the test-coverage gap (Test 7 verifies warning, not actual breakage). |
| C7 (5.C.12) | MEDIUM | **C** | `KindRule.Owner` field on schema (L7); auto-gen consumer at 3.20 (PLAN.md:56, 469). |
| C8 (5.C.13) | LOW | **C** | Supersede gate added to 3.19 acceptance Test 7 (PLAN.md:446). |
| N1 (5.C.14) | MEDIUM | **C** | Stricter-than-spec gate on `Owner == "STEWARD"` regardless of state delta locked (PLAN.md:447). |
| N3 (5.C.15) | LOW | **C** | Field renamed to `AuthRequestPrincipalType` (PLAN.md:439). |
| N4 (5.C.16) | LOW | **C** | Rollback cost note widened to 3.17 + 3.18 + 3.20 + 3.21 + tests (PLAN.md:411). |
| N5 (5.C.17) | LOW | **C** | Index design choice deferred to builder with two viable options (PLAN.md:426). |

### Unit D (Adopter Bootstrap + Doc Sweep)

| Round 1 ID | Severity | Closure | Notes |
| --- | --- | --- | --- |
| §2.1 (5.D.6) | HIGH | **C** | Three-way write conflict resolved (chain 3.7 → 3.23 → 3.27, mechanical fix 4.6, PLAN.md:535, 602). |
| §2.2 (5.D.7) | HIGH | **C** | `drop_1_5/`, `drop_1_75/` added to exclusion list (PLAN.md:587, mechanical fix 4.5). |
| §2.3 (5.D.8) | MEDIUM | **C** | `~/.claude/CLAUDE.md` promoted to first-class in-scope (PLAN.md:583). **§1.8 above re-surfaces** numeric inconsistency (3 vs 4 hits). |
| §2.4 (5.D.9) | MEDIUM | **C** | 3.24/3.25 narrowed to "CLAUDE.md pointer line only"; WIKI scaffolding deferred (PLAN.md:543, 558). |
| §2.5 (5.D.10) | MEDIUM | **A** | Audit-gap accept (option (c)) explicit in 3.27 acceptance (PLAN.md:597). |
| §2.6 (5.D.11) | LOW | **C** | `workflow/example/CLAUDE.md` insertion site disambiguated to `## Coordination Model — At a Glance` after line-26 bullet (PLAN.md:569). |
| §2.7 (5.D.12) | LOW | **C** | Wrap-up timing rationale: option (a) preserves PR-review density (PLAN.md:617). |
| §2.8 (5.D.13) | LOW | **C** | Skill frontmatter rationale captured (description = autoloader signal, PLAN.md:546). |
| §2.9 (5.D.14) | LOW | **C** | `~/.claude/commands/*.md` added to sweep path list (PLAN.md:584). |

### Closure Score

- **Cleanly closed (C):** 28 of 30 Round 1 CEs.
- **Partially closed (P):** 1 (C7 — process gap persists).
- **ACCEPTed-with-warning (A):** 2 (C6 / 5.C.11 — refinements-gate test coverage; §2.5 / 5.D.10 — `~/.claude/` audit gap).
- **NEW Round-2 counterexamples surfaced:** §1.1, §1.2, §1.3, §1.4, §1.5, §1.6, §1.7, §1.8, §1.9 (9 total).

The closure rate is high (~93% C, ~7% P/A). All A rows are dev-locked accept decisions, not unresolved issues. The 9 new findings are layered in on top — none are build-stopping; one (§1.1) is HIGH for dispatcher-correctness reasons.

---

## 3. Refuted Attacks (Honest Attempts)

### 3.1 — Round 1 CE-cascade compounding under three new bool fields (REFUTED)
**Attempt.** The plan adds `Persistent`, `DevGated`, `Irreducible` to `ActionItem` in 3.2 alongside `StructuralType`. Required-on-create rejects empty `StructuralType` per 3.2's domain validation. Do `Persistent` / `DevGated` / `Irreducible` compound the test-fixture cascade?
**Refutation.** No. PLAN.md:142 explicitly: *"`Persistent` / `DevGated` / `Irreducible` are bools with no validation (zero-value = `false` is the dominant case)."* Bools in Go default to `false`. Existing fixtures need no update for these three fields. Only `StructuralType` triggers the cascade.

### 3.2 — `principal_type: steward` autent boundary map could double-translate on the way back (REFUTED)
**Attempt.** Could the round-trip `steward → PrincipalTypeAgent → ActorTypeAgent` lose the `steward` axis on the way back, so the gate at `MoveActionItem` sees an `agent` principal instead of `steward`?
**Refutation.** Round 1 C3 already addressed this. L2 (PLAN.md:46) keeps `steward` as a tillsyn-internal axis on `auth_requests` + `AuthSession.PrincipalType` + `AuthenticatedCaller.AuthRequestPrincipalType`. The autent layer never returns the `steward` value — the gate reads from tillsyn's own auth-request storage, not from autent. The only place autent sees `steward` is the outbound `RegisterPrincipal` call where it's mapped down to `agent`; the gate never reads from that side of the boundary.

### 3.3 — Plan-QA falsification agent's new attack-vector block could re-prompt the agent recursively (REFUTED)
**Attempt.** 3.7 adds the §4.4 global L1 plan-QA sweep to the agent's prompt. When that sweep itself attacks a plan, does it recurse on its own attack vectors?
**Refutation.** The §4.4 sweep is a depth-and-tree-shape attack vector applied to a target plan, not to the agent's own prompt. It runs once per L1 plan. No recursion. Confirmed by reading `ta-docs/cascade-methodology.md` §4.4 — the sweep is a "second plan-QA pass" with full visibility, not a recursive structure.

### 3.4 — `KindRule.Owner` accepting `"STEWARD"` as a string (no enum) is a typo trap (REFUTED)
**Attempt.** `KindRule.Owner string` — typo `"steward"` (lowercase) vs `"STEWARD"` (uppercase) silently fails the gate.
**Refutation.** Round 2 plan does NOT carry-through normalization at 3.20 consumer level — but the auth gate keys on EXACTLY `existing.Owner == "STEWARD"` (capital). Memory rule "Tillsyn Titles Full Caps" + STEWARD principal naming consistently uses ALL CAPS. The gate check is case-sensitive Go string comparison; a typo `"steward"` in a TOML rule would mean the auto-generated item gets `Owner = "steward"` lowercase and the gate doesn't fire — but that ALSO means STEWARD principal sessions don't get blocked from anything (they don't see their own items). The failure mode is "no protection on this item," not "silent escalation." Acceptable risk during pre-MVP; worth a load-time validator check in Drop 4 (validate `Owner` matches a known principal-name enum).

### 3.5 — 3.27 in-repo legacy-vocabulary sweep could miss `kind=task, scope=task` rule callouts (REFUTED)
**Attempt.** Pre-Drop-2 rule `Create plan items with kind='task', scope='task'` (CLAUDE.md:14, also in `~/.claude/CLAUDE.md`:14) — is this in 3.27 sweep scope?
**Refutation.** That rule is NOT retired vocabulary — it's a current pre-Drop-2 directive. Sweep targets retired terms (`slice`, `build-task`, `plan-task`, `qa-check`, `drops all the way down`). The `kind='task', scope='task'` rule is current behavior until Drop 1.75 closes — separate from the cascade-vocabulary concern.

### 3.6 — `hylla_graph_nav` discovery — could there be a hidden caller of `KindTemplate` Hylla doesn't know about? (REFUTED)
**Attempt.** Per CLAUDE.md "Hylla indexes Go files only," does Hylla miss any caller in non-Go files (TOML schema fixtures, magefile)?
**Refutation.** `KindTemplate` is a Go type; non-Go files cannot call it directly. The TOML schema 3.8 introduces is a NEW schema that doesn't reference the deleted `KindTemplate`. Magefile is Go code; if it referenced `KindTemplate`, Hylla would index it. 3.15's "starting-point lower bound" disclaimer + LSP-driven sweep covers Go references comprehensively. No hidden non-Go caller.

---

## 4. Required Refactor Checklist (PASS-WITH-MITIGATION-REQUIRED)

Verdict is PASS-WITH-MITIGATION-REQUIRED — none of the new counterexamples block builder dispatch on the early droplets, but each should be addressed before the affected droplet fires.

**HIGH (must fix before 3.20 fires):**

1. **§1.1** — Add `3.14` to droplet 3.20's `Blocked by:` field (PLAN.md:485). Add corresponding row to cross-unit blocker matrix (PLAN.md:629).

**MEDIUM (must fix before the affected unit fires):**

2. **§1.2** — Add `3.11 blocked_by 3.10` (or document the dispatcher-side serialization commitment more strongly) for the `internal/templates` package-compile lock.
3. **§1.3** — Add `internal/app/service.go` to droplet 3.2's enumerated sweep list (PLAN.md:144).
4. **§1.4** — Add "package-level build+test gate as first-class node" to "Out Of Scope Confirmations" deferred list (PLAN.md:74) OR wire it into Drop 3 explicitly.
5. **§1.5** — Extend droplet 3.22 Test 7 to assert the Drop 1 parent-blocks-on-incomplete-child invariant alongside the warning-fires assertion (PLAN.md:515).

**LOW (cosmetic / process — fix at convenience):**

6. **§1.6** — Swap column order in cross-unit blocker matrix row at PLAN.md:633 (`3.5 ↔ 3.4` → `3.4 ↔ 3.5`).
7. **§1.7** — Either decompose 3.15 / 3.27 (preferred) or set `Irreducible = true` with justification on 3.19 / 3.21 / others as appropriate.
8. **§1.8** — Update PLAN.md:583 from "Three known retired-vocab hits" to "Four known retired-vocab hits" (1 slice + 3 build-task).
9. **§1.9** — Add a planner-spawn directive line to the orchestrator's spawn template requiring Hylla-MCP-first evidence sourcing per CLAUDE.md §"Code Understanding Rules" rule 1.

After items 1-5 land, the plan is cleared for builder dispatch on droplets 3.1 → 3.7 immediately; 3.8 onwards as the chain unblocks. Items 6-9 can land as in-flight corrections.

---

## 5. Verdict Summary

**PASS-WITH-MITIGATION-REQUIRED.**

- 9 NEW counterexamples surfaced (1 HIGH, 4 MEDIUM, 4 LOW).
- 30 Round 1 counterexamples — 28 cleanly closed, 1 partially closed (process gap), 2 ACCEPTed-with-warning (dev-locked).
- No CONFIRMED build-stopping counterexample.
- HIGH severity is dispatcher-contract-correctness (§1.1 missing blocker), not a Drop 3 build defect.
- Plan is materially better than Round 1: closure rate ~93%; structural improvements (atomic 3.19, single-source-of-truth fixture for 3.10/3.14, autent boundary-map docs) all land cleanly.

The plan is ready for builder dispatch once §1.1 lands. §1.2-§1.5 land as in-flight tightenings before the affected droplets fire. §1.6-§1.9 are cosmetic and can land alongside any other touchup.

---

## 6. Most Damaging Counterexample

**§1.1 — Missing cross-unit blocker `3.20 blocked_by 3.14`.**

The whole point of Drop 3's cascade-architecture work is to make blocker-graph correctness a first-class concern. A plan that ships its own architecture-defining drop with a missing blocker is the strongest possible self-falsification. Drop 4's dispatcher reads the `Blocked by:` field literally. With the gap, the dispatcher fires 3.20 (which writes to `internal/templates/builtin/default.toml`) before 3.14 (which CREATES that file). Builder either errors on missing-file write or — worse — silently creates a partial file that 3.14 then overwrites, losing 3.20's auto-gen `[child_rules]` body.

Pre-Drop-4 the orchestrator dispatches manually and could catch this from the topological-sort prose. Post-Drop-4 the gap weaponizes parallel dispatch. Both eras are within Drop 3's planning lifetime — Drop 4 is the immediate next consumer per "Out Of Scope Confirmations" (PLAN.md:664).

Fix is one-line: add `3.14` to PLAN.md:485 and add a row to PLAN.md:629. Trivial mechanical fix; load-bearing for dispatcher correctness.

---

## 7. Hylla Feedback

**N/A — task touched non-Go files only.**

This review attacked a Markdown plan document. Hylla today indexes Go only (per memory rule "Hylla Indexes Only Go Files Today"). Evidence-gathering used:

- `Read` for the plan + Round 1 falsification artifacts + REVISION_BRIEF + methodology canonical + `~/.claude/CLAUDE.md`.
- `grep` (system `/usr/bin/grep` due to sandbox restrictions on the explicit `Bash` cwd-relative path) for verifying `newActionItemForTest` exists at `internal/tui/model_test.go:14679` and counting `NewActionItem` references across `internal/` + `cmd/`.

No Hylla queries attempted; no fallback miss to record. Hylla would not have helped — the cited Go line numbers had already been verified by Round 1's per-unit QA artifacts; this round's targets were the unified plan's structural integrity (cross-unit blockers, methodology compliance, Round 1 closure verification), all of which live in MD.
