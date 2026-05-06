# Drop 4c.5 Master PLAN — QA Falsification Round 2

**Reviewer:** go-qa-falsification-agent
**Date:** 2026-05-05
**Verdict:** NEEDS-REWORK

Round-2 finds round-1's F1+F2 closure is correct as far as it goes (A.1 ↔ B.1 ↔ C.1 collision now serialized via Chain 1 + Chain 5). F3 + F4 also closed cleanly. **However, the Chain-5 callout missed a fourth participant: A.4 also touches `app_service_adapter_mcp.go` per `THEME_A_PLAN.md` line 201** (comment-only, but still a `paths`-level edit). Per CLAUDE.md "File- and package-level blocking", file overlap mandates an explicit `blocked_by`. Chain 1's serial ordering already does the right thing transitively (A.1 → A.4 → B.1 → C.1) so this is **traceability-grade, not correctness-grade**, but the master plan now claims Chain 5 has "1 droplet, plus A.1 + B.1 cross-chain" when the real shape is "1 droplet, plus A.1 + A.4 + B.1 cross-chain." Fixable with a one-line edit.

## 1. Round-1 Findings Closure

- **1.1 — F1 (A.1 ↔ C.1 collision).** **CLOSED.** Master Chain 5 now serializes C.1 after B.1, which serializes after A.1 via Chain 1. C.1 row's `blocked_by` cell explicitly says `B.1 (transitively A.1 via Chain 1)`. File order on `app_service_adapter_mcp.go` lands A.1 → B.1 → C.1, satisfying CLAUDE.md "Paths and Packages."
  - Citation: PLAN.md lines 97-103 (Chain 5 section) + line 117 (cross-theme justification block).

- **1.2 — F2 (B.1 ↔ C.1 collision).** **CLOSED.** Same fix as 1.1 — `C.1 blocked_by: B.1` is the direct edge, and Chain 5's narrative names B.1's `SupersedeActionItem` passthrough as the surface that needs to land before C.1's gate-extension edit lands. Wave E listing also moves C.1 out of any earlier wave (PLAN.md line 159).

- **1.3 — F3 (F.2.4 missing direct `F.1.3` edge).** **CLOSED.** Master Chain 1 row for F.2.4 now reads `F.1.3, F.2.1, F.2.2 (transitively F.1.2 via Chain 1)`. This matches Theme F line 277 verbatim plus an explicit transitive-via-package-lock annotation. PLAN.md line 62 + line 123.

- **1.4 — F4 (no spawn-prompt template).** **CLOSED.** A new "Builder Spawn-Prompt Template" subsection (PLAN.md lines 187-223) ships a verbatim prompt covering: (a) opus model directive (line 211), (b) "DO NOT commit" + "DO NOT push" directive with F.7-CORE REV-13 reference (line 212), (c) "NEVER raw `go test` / `go build` / `go vet` / `mage install`" (line 213), (d) Section 0 SEMI-FORMAL REASONING directive (line 218), (e) BUILDER_WORKLOG.md append directive with worklog-section template (line 215), (f) state-mutation directive — droplet `state: in_progress` at start, `state: done` at end (line 216), (g) required pre-work reading (lines 196-200): master PLAN, source THEME PLAN, REVISION_BRIEF section, CLAUDE.md, (h) Tillsyn-flow output style (line 220), (i) QA-prompt mirror note (line 223). Covers everything REVISION_BRIEF §6 + WORKFLOW.md spawn contract require.

## 2. New Counterexamples

### F5 — Chain 5 callout omits A.4 as a fourth participant on `app_service_adapter_mcp.go`

- **What breaks.** PLAN.md line 97 reads "Chain 5 ... 1 droplet, plus A.1 + B.1 cross-chain" and line 99 says "A.1 and B.1 (already in Chain 1) ALSO edit `internal/adapters/server/common/app_service_adapter_mcp.go` as secondary file." This narrative is incomplete: per `THEME_A_PLAN.md` line 201, **A.4 also edits `app_service_adapter_mcp.go`** ("`validateMetadataOutcome` stays unchanged ... add a comment cross-referencing the new service-level invariant for `failed`"). Comment-only is still a `paths`-level edit; CLAUDE.md "Paths and Packages" treats sibling builds sharing a file as requiring an explicit `blocked_by` regardless of edit-size.
- **Reproduction trace.** `rg -n "app_service_adapter_mcp" workflow/drop_4c_5/THEME_A_PLAN.md` returns three hits — line 46 (A.1's adapter mapping), line 155 (A.3 says "nothing changes here"), and **line 201 (A.4 adds a cross-reference comment).** The Chain 5 callout in PLAN.md missed line 201.
- **Severity.** **TRACEABILITY-GRADE, not CORRECTNESS-GRADE.** Chain 1 already serializes A.1 → A.4 → B.1, so the actual file-touch order on `app_service_adapter_mcp.go` is A.1 → A.4 → B.1 → C.1. The DAG is consistent; only the Chain-5 narrative undercount is wrong.
- **Recommended fix.** Update PLAN.md Chain 5 narrative (line 97 + line 99) to read "1 droplet, plus A.1 + A.4 + B.1 cross-chain" and "A.1, A.4, and B.1 (all in Chain 1) ALSO edit `internal/adapters/server/common/app_service_adapter_mcp.go` as secondary file." Also extend the cross-theme justification at line 117 to mention A.4's adapter-comment edit. One-line edits; no DAG change.

### F6 — Wave E claims C.1 lands "must follow B.1's adapter-file edits" but does not name A.4

- **What breaks.** PLAN.md line 159 says "C.1 (Chain 5; blocked_by: B.1 — must follow B.1's adapter-file edits)". Per F5 above, C.1 must also follow A.4's adapter-file edits. The Wave E description is consistent with the master Chain 5 row (which only names B.1 directly), but combined with F5 it underspecifies the file dependency graph.
- **Severity.** **TRACEABILITY-GRADE.** Same reasoning as F5 — Chain 1's serial ordering does the right thing transitively. But a future plan-orch reading the Wave E description without cross-checking THEME_A_PLAN.md will not realize A.4 also touches the file.
- **Recommended fix.** Update Wave E line 159 to add "(transitively follows A.4 via Chain 1)" alongside the existing B.1 callout.

## 3. Mitigated Attacks

- **Cycle in revised DAG.** Walked all `blocked_by` edges across Chain 1-5 + Independent group. Graph: A.1 → A.4 → B.1 → B.2 → C.2 → C.3 → E.8 → F.6.1 → F.1.1 → F.1.2 → F.2.4 → E.9 (Chain 1 serial). A.2 → A.1, A.3 → A.2, E.5 → A.3, F.3.1 → E.5/F.2.1/F.2.2/F.1.2, F.3.2 → F.3.1/F.5.1, F.3.3 → F.3.2/F.1.2 (Chain 2). E.1 → E.2 → E.3 → E.4 → E.7 (Chain 3 serial). F.2.1 → F.2.2 → F.1.3 → E.6 → F.5.1 → F.5.2 (Chain 4 serial). Cross-chain: F.1.1 → F.6.1 + F.2.1, F.1.2 → F.1.1 + F.1.3, F.2.4 → F.1.3 + F.2.1 + F.2.2, E.6 → F.1.3, F.5.1 → E.6 (already serial), F.5.1 → F.2.1 (transitive). C.1 → B.1 (Chain 5). D.2 → D.1. F.2.3 → F.2.1. **No cycle.**

- **Wave A integrity.** Verified each Wave A member has `blocked_by: —` in its master row: A.1 (line 52 — yes), C.4 (line 109 — yes), D.1 (line 110 — yes), E.1 (line 80 — yes), F.2.1 (line 90 — yes). **Five droplets, all true Wave A heads.** F.2.3 correctly removed from Wave A — appears in Wave B (line 143). The round-1 inconsistency is closed.

- **Spot-check 5 random droplets for master ↔ theme drift.**
  - **A.4** — Master row: `blocked_by: A.1`. Theme A line 235: "Blocked by: A.1 (same-package compile lock on `internal/app`)." ✓ Match.
  - **B.2** — Master row: `blocked_by: B.1`. Theme BD line 121: "Blocked by: `B.1`." ✓ Match.
  - **C.3** — Master row: `blocked_by: C.2`. Theme CE line 98: "**Blocked by:** **C.2**." ✓ Match.
  - **E.6** — Master row: `blocked_by: F.1.3`. Theme CE chain summary line 449: "E.6 (alone in internal/templates)." However, Theme F Note 8 line 561 says "F.5.1 ... blocks: F.2.1" but does not name E.6 directly. Master correctly inserts E.6 in Chain 4 between F.1.3 and F.5.1 (per cross-cutting decision line 40). ✓ Master cross-theme justification at line 118 names this. Match.
  - **F.5.2** — Master row: `blocked_by: F.5.1`. Theme F line 356: "**Blocked by:** F.5.1 (shares load.go editing surface — package-level lock)." ✓ Match.

- **Hidden dependencies via shared types (round-1 1.8).** Re-checked `UpdateActionItemInput` ripple. Round-1 noted `internal/tui/model.go` × 4 sites, `thread_mode.go` × 1 site as compile-error-guarded. Master plan does not explicitly add `internal/tui` to A.1's primary `paths`, but Theme A line 51 lists `internal/tui/` under "Files / paths to modify" with the caveat "any TUI call site that constructs `UpdateActionItemInput` directly (audit via grep)." Builder will see compile errors via `mage ci` — Go type system enforces it. Mitigated.

- **Spawn-prompt template completeness.** Verified against REVISION_BRIEF §6 and WORKFLOW.md spawn contract. All required directives present (see F4 closure above). One minor nit: the template's "REQUIRED PRE-WORK READING" lists `<RELEVANT_SECTION>` of REVISION_BRIEF — the orchestrator must remember to substitute the correct section number per droplet (e.g. §3.1 for Theme A droplets, §3.5 for Theme E droplets). Not a falsification finding because the placeholder `<RELEVANT_SECTION>` is explicit. Mitigated.

- **F.2.4 row's transitive-via-Chain-1 annotation.** Master row line 62: `F.1.3, F.2.1, F.2.2 (transitively F.1.2 via Chain 1)`. Verified Theme F Note 8 line 559: "F.2.4 (caller audit) — blocks: F.1.3, F.2.1, F.2.2." Master adds the F.1.2 transitive note for plan-orch traceability. Mitigated.

## 4. Conclusion

**Verdict: NEEDS-REWORK (traceability-grade only).**

The round-1 BLOCKER findings (F1, F2, F3, F4) are all closed. The DAG is acyclic. Wave A is correctly 5 droplets (no F.2.3 misclassification). The new Builder Spawn-Prompt Template covers every required directive.

Two new traceability findings landed:

- **F5** — Chain 5 narrative miscounts participants (says "A.1 + B.1 cross-chain" but A.4 also edits `app_service_adapter_mcp.go` per THEME_A_PLAN.md line 201). Chain 1 already serializes correctly so this is narrative-only.
- **F6** — Wave E description does not mention A.4's transitive contribution to the adapter-file ordering.

**Required fixes (one-line edits each):**

1. PLAN.md line 97: change "1 droplet, plus A.1 + B.1 cross-chain" → "1 droplet, plus A.1 + A.4 + B.1 cross-chain."
2. PLAN.md line 99: extend "A.1 and B.1 (already in Chain 1) ALSO edit ..." → "A.1, A.4, and B.1 (all in Chain 1) ALSO edit ...".
3. PLAN.md line 117 (cross-theme justification): mention A.4's `validateMetadataOutcome` cross-reference comment edit alongside A.1 (UpdateActionItem mapping) and B.1 (SupersedeActionItem passthrough).
4. PLAN.md line 159 (Wave E): add "(transitively follows A.4 via Chain 1)" alongside B.1 reference.

After these fixes, falsification round 3 should be a quick re-run focused on confirming the four edits land cleanly with no new collisions or DAG drift.
