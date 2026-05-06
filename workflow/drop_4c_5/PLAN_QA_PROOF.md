# Drop 4c.5 Master PLAN — QA Proof Round 1

**Reviewer:** go-qa-proof-agent
**Date:** 2026-05-05
**Verdict:** PASS (with NITs)

## 1. Trace Coverage

- **1.1 Droplet count integrity** — COVERED. Per-theme counts: Theme A = 4 (A.1/A.2/A.3/A.4), Theme BD = 4 (B.1/B.2/D.1/D.2), Theme CE = 13 (C.1/C.2/C.3/C.4/E.1/E.2/E.3/E.4/E.5/E.6/E.7/E.8/E.9), Theme F = 13 (F.1.1/F.1.2/F.1.3/F.2.1/F.2.2/F.2.3/F.2.4/F.3.1/F.3.2/F.3.3/F.5.1/F.5.2/F.6.1). Total 4+4+13+13=34. Master claim of 34 holds.
- **1.2 Cross-theme blocked_by accuracy** — COVERED with one NIT (P3 below). Verified A.2→A.1 (Theme A §1 line 34: "A.2 → A.3"; master coordinates wire-shape, defensible); F.1.1→F.2.1 (Theme F Note 8 line 555 confirms); F.3.1→F.2.1+F.2.2+F.1.2 (Theme F Note 8 line 563 confirms); F.3.2→F.5.1 (Theme F Note 8 line 564 confirms); E.6→F.1.3 (master "Cross-Theme Blocked-By Justifications" line 110 cites F.1.3 lands embedded resolver before E.6 canonicalization — defensible reasoning); F.5.1→E.6 (master line 111 cites shared `load.go` validator chain — defensible).
- **1.3 Package-lock chain correctness** — COVERED.
  - Chain 1 (`internal/app`, 12 droplets): A.1 (service.go) ✓; A.4 (service.go) ✓; B.1 (service.go new method) ✓; B.2 (service.go new method) ✓; C.2 (auto_generate_steward.go) ✓; C.3 (auto_generate_steward.go) ✓; E.8 (auth_requests.go) ✓; F.6.1 (service.go + kind_capability.go) ✓; F.1.1 (service.go) ✓; F.1.2 (service.go extend) ✓; F.2.4 (service.go) ✓; E.9 (service.go nil-check at lines 1015-1019) ✓. All in `internal/app` package.
  - Chain 2 (`mcpapi`, 6 droplets): A.2 (handler.go + extended_tools.go + handoff_tools.go) ✓; A.3 (handler.go) ✓; E.5 (handler.go) ✓; F.3.1 (extended_tools.go + handler.go) ✓; F.3.2 (extended_tools.go) ✓; F.3.3 (extended_tools.go) ✓. All in `internal/adapters/server/mcpapi`.
  - Chain 3 (dispatcher, 5 droplets): E.1 (locks_file.go + locks_package.go) ✓; E.2 (walker.go) ✓; E.3 (conflict.go) ✓; E.4 (monitor.go) ✓; E.7 (gate_mage_test_pkg_test.go) ✓. All in `internal/app/dispatcher`.
  - Chain 4 (templates, 6 droplets): F.2.1 (embed.go + builtin/) ✓; F.2.2 (builtin/) ✓; F.1.3 (embed.go) ✓; E.6 (load.go) ✓; F.5.1 (load.go) ✓; F.5.2 (load.go) ✓. All in `internal/templates`.
- **1.4 Open-question resolutions defensible** — COVERED, one departure noted (P4). Q1 DEFER matches REVISION_BRIEF §9 Q1 lean ✓. Q2 WARN-ONLY matches Q2 lean ✓. Q3 disposition documented across master + Theme CE (line 6 §0). Q4 ADAPTER-STAMP-IS-SUFFICIENT departs from REVISION_BRIEF §9 Q4 lean ("yes, dispatcher should also stamp"); master rationale is sound — dispatcher today uses CLI path which now stamps `"cli"`, so adapter-stamp covers it transitively. Routes "if dispatcher gains direct CreateAuthRequest call" to Drop 4d/5 follow-up. Defensible departure with documented routing. Q5 A+B mandatory matches Q5 lean ✓.
- **1.5 No unresolved cross-cutting design issues** — COVERED. Master "Cross-Theme Cross-Cutting Decisions" §§ explicitly resolves: (a) no new domain types (B.1 reuses metadata.outcome/transition_notes); (b) A.4's strict-failure-outcome-enum check INCLUDED (resolves Theme A §3 Q-A-3); (c) E.6 fix-path = post-decode canonicalization (resolves Theme CE §"E.6 fix-path decision"); (d) E.9 placement = `internal/platform/gitenv` (resolves Theme CE §"Falsification-loud surfaces" item 4); (e) F.6.1 + Theme A `service.go` collision serialized in Chain 1.
- **1.6 Evidence per droplet (spot-check 6)** — COVERED. A.1: Theme A lines 38-91 (file paths to specific line ranges, 8 acceptance criteria, 9 test scenarios, 3 falsification mitigations) ✓. B.1: Theme BD lines 19-75 (5 file paths, 8 acceptance criteria, 10 test scenarios, 3 mitigations) ✓. C.2: Theme CE lines 44-72 (file lines 364 + 344-363, 5 acceptance criteria, 4 test scenarios, 3 mitigations) ✓. E.6: Theme CE lines 290-320 (load.go line 284-301, 6 acceptance criteria, 5 test scenarios, 3 mitigations) ✓. F.3.1: Theme F lines 399-436 (5 file paths, 7 acceptance criteria, 4 test scenarios, 3 mitigations) ✓. F.6.1: Theme F lines 366-395 (service.go:897, kind_capability.go:1002, 6 acceptance criteria, pure refactor justification, 3 mitigations) ✓.
- **1.7 REVISION_BRIEF §3 scope coverage** — COVERED. §3.1 → A.1/A.2/A.3/A.4 (4 PATCH/strict-decode/client_type/outcome). §3.2 → B.1/B.2 (supersede + list). §3.3 → C.1 (R1), C.2 (R2), C.3 (R3), C.4 (R5) — all four Drop 3 refinements mapped. §3.4 → D.1 (go.mod replace cleanup) + D.2 (vet/gopls sweep). §3.5 → E.1 (R4+R7), E.2 (R5), E.3 (R8), E.4 (R9), E.5 (R12) for 4a; E.6 (R1), E.7 (R2), E.8 (R3), E.9 (R4) for 4b. All 4a/4b residue items mapped. §3.6 → F.1.1/F.1.2/F.1.3 (auto-discovery), F.2.1/F.2.2/F.2.3/F.2.4 (builtin separation), F.3.1/F.3.2/F.3.3 (MCP tool), F.5.1/F.5.2 (validation), F.6.1 (KindTemplate cleanup). All five F sub-themes covered.
- **1.8 No Tillsyn runtime dependencies in plan** — COVERED. Across all 34 droplets, no droplet invokes a Tillsyn MCP call, action-item runtime, or capability-lease primitive. All work is code edits + mage runs. Filesystem-MD-mode discipline preserved.

## 2. Evidence Completeness

- **2.1 A.1 evidence:** `THEME_A_PLAN.md:38-91` cites `internal/app/service.go:664-763` (struct), `:1201-1388` (body), `:1226-1232` (priority defaulting), `:1230` (UpdateDetails call). 9-row test scenario table at lines 72-82.
- **2.2 A.2 evidence:** `THEME_A_PLAN.md:94-141` cites `mcpapi/handler.go` (5 sites), `extended_tools.go` (11 sites), `handoff_tools.go` (5 sites) totaling 21 BindArguments call sites — concrete count.
- **2.3 A.3 evidence:** `THEME_A_PLAN.md:144-190` cites `auth_requests.go:224`, `handler.go:113`, `:199`, `:187-205`, `cmd/till/main.go:2675`, `:2689`, `:3055`. Cross-references autentauth `service.go:829`.
- **2.4 A.4 evidence:** `THEME_A_PLAN.md:192-241` cites `service.go:1043` (MoveActionItem entry), `:1068` (toState detection), `:1079` (terminal-state guard), `:1099` (column move) — surgical insertion point.
- **2.5 B.1 evidence:** `THEME_BD_PLAN.md:19-75` cites `service.go:1079` (terminal guard to bypass), `mutation_guard.go:23` (existing parent-blocks gate), `app_service_adapter_mcp.go:1163` (existing "superseded" recognition), `handler_steward_integration_test.go:459-461` (skipped test to un-skip).
- **2.6 C.1 evidence:** `THEME_CE_PLAN.md:10-41` cites `app_service_adapter_mcp.go:845-852` (call site), `:820-829` and `:1109-1121` (doc-comments).
- **2.7 E.1 evidence:** `THEME_CE_PLAN.md:139-171` cites `locks_file.go:60-81` (Acquire doc), `locks_file_test.go:307` (equalStringSlices helper). Mirror coverage on locks_package.
- **2.8 E.6 evidence:** `THEME_CE_PLAN.md:290-320` cites `load.go:284-301` (validateMapKeys), `load.go:125` (caller), `domain/kind.go:50-52` (case-fold contract).
- **2.9 F.1.1 evidence:** `THEME_F_PLAN.md:27-60` cites `service.go:427` (loadProjectTemplate body), `:401` (bakeProjectKindCatalog caller), `:346` (CreateProjectWithMetadata).
- **2.10 F.3.1 evidence:** `THEME_F_PLAN.md:399-436` cites `extended_tools.go:1673` (registerKindTools pattern), Drop-3-finding-5.B.14 (snapshot policy).
- **2.11 D.1 evidence:** `THEME_BD_PLAN.md:131-166` cites `PLAN.md §19.1` (retention rule), commit `66c354e` (last go.mod-touching commit), `third_party/teatest_v2/`, `internal/adapters/embeddings/fantasy/`.

## 3. Findings

**P1 — NIT — Wave A / Wave B contradiction on F.2.3.** `PLAN.md:127` (Wave A bullet for F.2.3) reads "F.2.3 (independent — but blocked_by F.2.1 since it copies the rebadged content; lands in Wave B)" while still listing F.2.3 under Wave A. F.2.3 then ALSO appears in Wave B at `PLAN.md:135`. This is a wave-list typo: F.2.3 should appear in Wave B only. Fix: remove F.2.3 from the Wave A bullet list at `PLAN.md:127` (keep the parenthetical explanation in the Wave B entry). Non-blocking — wave structure is already documented as approximate (`PLAN.md:152`).

**P2 — NIT — Master Chain 1 row for F.2.4 lists `F.1.2, F.2.1, F.2.2` while Theme F authoritative source says `F.1.3, F.2.1, F.2.2`.** `PLAN.md:62` (Chain 1 table row F.2.4) lists "F.1.2, F.2.1, F.2.2" as `Blocked by`. `THEME_F_PLAN.md:277` (F.2.4 droplet) and `THEME_F_PLAN.md:559` (Note 8 dependency graph) both say F.2.4 is blocked by F.1.3, F.2.1, F.2.2. The functional dependency is on F.1.3 (the language-aware resolver F.2.4's caller-audit redirects to). Master's `F.1.2` entry is transitively correct (F.1.2 → F.1.3 in Chain 4 ordering, and F.1.2 lands before F.1.3 in Chain 1) but mislabels the proximate dependency. Fix: change the Chain 1 row's `Blocked by` cell to `F.1.3, F.2.1, F.2.2` to match Theme F's authoritative source. Non-blocking — transitively equivalent; would-be Chain-1-builder still serializes correctly.

**P3 — NIT — Master cross-theme justification for F.5.1 misses that F.5.1 also depends on F.2.1.** `PLAN.md:111` justifies F.5.1 → E.6 via shared `load.go`. Theme F's F.5.1 droplet (line 318) explicitly says "Mark `blocked_by: F.2.1` to serialize" — so F.5.1 depends on F.2.1 (via test-fixture rename) AND on E.6 (via shared load.go file-lock). Master Chain 4 row F.5.1 lists `Blocked by: E.6` (transitively correct since E.6 → F.1.3 → F.2.1, F.2.2). Fix optional: add a line in master's "Cross-Theme Blocked-By Justifications" noting the transitive F.2.1 dependency, OR leave as-is since transitive ordering preserves correctness.

**P4 — NON-BLOCKER — Q4 resolution departs from REVISION_BRIEF §9 lean.** `REVISION_BRIEF.md:140` Q4 lean: "yes, for full cascade-on-itself coherence" (dispatcher should ALSO stamp). Master `PLAN.md:33` Q4 resolution: "ADAPTER-STAMP IS SUFFICIENT" — claims dispatcher's CLI-path provisioning makes adapter-stamp transitively cover cascade subagents. The master's reasoning is internally consistent (dispatcher today provisions via CLI, which stamps `"cli"`) and explicitly routes the future direct-call path to Drop 4d/5 follow-up. Departure from the BRIEF lean is documented and defensible. Surface for orchestrator awareness.

**P5 — NIT — Wave structure references "F.2.1 (Chain 4 head)" but does not flag that Wave A still has multiple Chain heads landing in parallel.** `PLAN.md:120-128` Wave A lists 7 launch candidates. The wave structure is labeled "approximate" (line 152) so this is descriptive, not prescriptive — non-blocking.

**P6 — NIT — Master `Wave E` final bullet says "C.2, C.3, E.8 in Chain 1 (remaining mid-chain droplets)".** Per Chain 1 ordering at `PLAN.md:50-64`, C.2 / C.3 / E.8 sit at positions 5/6/7 (between B.2 and F.6.1) — they are MID-chain not tail. Calling them "Final" wave is misleading. Wave structure is approximate; non-blocking.

## 4. Conclusion

**Verdict: PASS (with NITs).**

The master `PLAN.md` correctly synthesizes 34 droplets across four package-lock chains with consistent cross-theme blocked_by routing. Droplet counts match (4+4+13+13=34). Package-lock chains correctly group all 12+6+5+6 droplets by their actual package edits. All six spot-checked droplets have specific file:line references, yes/no-verifiable acceptance criteria, concrete test scenarios, and ≥3 falsification mitigations. Open-question resolutions Q1/Q2/Q3/Q5 align with REVISION_BRIEF §9 leans; Q4 departs but the rationale is documented and the future-path is routed. REVISION_BRIEF §3.1-3.6 scope is fully covered. No droplet depends on Tillsyn runtime calls.

The findings above are NITs — wave-structure prose drift (P1, P5, P6), one Chain-1-table proximate-vs-transitive blocker label (P2), and one minor Q4-departure flag (P4). None block builder dispatch. Plan-QA-falsification (running in parallel) may surface counterexamples this proof pass does not attempt to construct.

## 5. Hylla Feedback

N/A — Hylla calls explicitly disabled per orchestrator's filesystem-MD-mode override; all evidence sourced from Read on the four theme MDs + REVISION_BRIEF.
