# Drop 4c Plan-QA Falsification — Round 4 (Final Verification)

**Reviewer role:** plan-qa-falsification (round 4 — final verification).
**Round date:** 2026-05-04.
**Inputs reviewed:**
- `workflow/drop_4c/PLAN.md` (master, post-R3 fixes).
- `workflow/drop_4c/F7_17_CLI_ADAPTER_PLAN.md` (REVISIONS body + STATUS callouts on F.7.17.9 / F.7.17.10).
- `workflow/drop_4c/F7_CORE_PLAN.md` (REVISIONS unchanged from R3; spot-check on F.7.4 acceptance for absorption coverage).
- `workflow/drop_4c/F7_18_CONTEXT_AGG_PLAN.md` (NEW REVISIONS section appended).
- `workflow/drop_4c/PLAN_QA_FALSIFICATION_R3.md` (anchor).

**Mode:** read-only. No code edits, no plan edits.
**Hard constraint:** no Hylla calls.

---

## 1. Round-3 Blocker Verifications (V1–V5)

### 1.1 V1 — Residual `command` / `args_prefix` / shell-interpreter scaffolding

**Search executed:** `rg -n "Command \[\]string|ArgsPrefix|args_prefix|validateAgentBindingCommandTokens|shellInterpreterDenylist"` across all four plan files.

**Hits found:**

- **F.7.17 sub-plan**: 11 hits (lines 15, 16, 161, 176, 178, 185, 186, 255, 304, 316, 410). REVISIONS REV-1 (lines 687–700) explicitly supersedes lines 15–17, the F.7.17.1 acceptance block, the BindingResolved field list at line 255, and the F.7.17.5 `SpawnDescriptor` widening at line 410. STATUS callout at line 614 supersedes F.7.17.10. **REV-1 coverage is comprehensive — every body hit is named in the supersession list.** REV-8 + the prepended REVISIONS-first directive ensure builders read REV-1 before the stale body. **VERDICT: REFUTED** (verified-closed).
- **F.7.18 sub-plan**: 4 hits (lines 26, 34, 68, 499–501). REVISIONS REV-1 (line 499) explicitly names lines 26, 34, 68 as superseded. **VERDICT: REFUTED** (R3 V1 closed).
- **F.7-CORE sub-plan**: 9 hits (lines 39, 64, 83, 252, 267, 698, 1000, 1002, 1006–1007). REVISIONS REV-1 (line 1000) explicitly enumerates F.7.3 hard prereqs (line 252 / 267), F.7.3 argv emission (line 39), and the cross-plan reference at line 698. The DAG diagram tokens at lines 64 + 83 are inside fenced ASCII diagrams; REV-1 doesn't enumerate them line-by-line, but the prose at REV-1 lines 1006–1007 is unambiguous about the new shape. **VERDICT: REFUTED-WITH-NIT** (DAG ASCII diagrams contain stale tokens not enumerated, but REV-1's prose makes the new shape clear; combined with the REVISIONS-first directive a builder will not implement the dropped fields).
- **Master PLAN.md**: 1 hit at line 195 — `**No migration logic in Go.** Schema additions (\`AgentBinding.Command []string\`, \`Context\` sub-struct, ...)`. **The master PLAN has NO REVISIONS section** — it is the canonical orchestration doc and was supposed to be rewritten in-place during R4. Line 195 still claims `AgentBinding.Command []string` is one of the schema additions in scope. Master PLAN §3 L2 (line 41) explicitly contradicts this with "Tillsyn does NOT surface a `command` override field." A careful reader sees the contradiction; a skim-reader of §6 alone does not. **VERDICT: NIT** — minor residual in master PLAN line 195. Recommend strike-through or rewrite to `(Context sub-struct, Tillsyn top-level struct, permission_grants.cli_kind column)`. Not blocking — §3 L2 is unambiguous and §6.5 directs builders to read REVISIONS first; master PLAN §6 is procedural/secondary.

**Aggregate verdict for V1: REFUTED-WITH-NITS.** R3's flagged sub-plan REVISIONS gaps closed. One residual master-PLAN line 195 stale-listing. Not a blocker.

### 1.2 V2 — F.7.17.10 marketplace droplet body removed

**Evidence:** F.7.17.10 section header at line 614:

```
### 4c.F.7.17.10 — Marketplace install-time confirmation (paper-spec) — **REMOVED per REVISIONS REV-4**

**STATUS: REMOVED.** This droplet is superseded by REVISIONS POST-AUTHORING REV-4 below. The `command` and `args_prefix` fields were dropped from the design; without `command[0]` to confirm at install time, this droplet has no scope. Builders MUST NOT implement this droplet. Acceptance criteria, test scenarios, and other body text below are NULL — kept only as a marker so historical references resolve.

(Original body deleted; refer to REVISIONS REV-4 for context.)
```

The body content is GONE. No acceptance criteria remain. Section ends at line 619 (the divider `---` precedes line 622 F.7.17.11). No leakage.

**Verdict: REFUTED.** R3 V1 (F.7.17.10 marketplace droplet) closed.

### 1.3 V3 — F.7.17.9 monitor-refactor MERGED into F.7-CORE F.7.4

**STATUS callout evidence:** F.7.17 sub-plan line 568:

```
### 4c.F.7.17.9 — CLI-agnostic monitor refactor — **MERGED INTO F.7-CORE F.7.4 per REVISIONS REV-7**

**STATUS: MERGED.** F.7-CORE F.7.4 already builds the CLI-agnostic monitor (per F.7-CORE plan acceptance line 356 — monitor consumes via `adapter.ParseStreamEvent` from inception, not as a refactor). This droplet is redundant. Builders MUST NOT implement this droplet as a separate unit. F.7-CORE F.7.4 absorbs every acceptance criterion below into its own acceptance.
```

**Absorption check (load-bearing test):** F.7-CORE F.7.4 acceptance criteria at lines 345–357 of `F7_CORE_PLAN.md`:

- ✅ `claudeAdapter.ParseStreamEvent` — present (line 346).
- ✅ `claudeAdapter.ExtractTerminalReport` — present (line 347).
- ✅ `TerminalReport` populated from result event — present (lines 348–352).
- ✅ Malformed JSON line returns wrapped error, monitor logs+skips — present (line 353).
- ✅ Truncated stream → `stream_unavailable` — present (line 354).
- ✅ Empty stream.jsonl → `stream_unavailable` — present (line 355).
- ✅ **Dispatcher monitor stays CLI-agnostic — consumes `StreamEvent` from `adapter.ParseStreamEvent`, does NOT branch on `cli_kind`. Adapter selection via `adapterRegistry.Get(cliKind)`** — present (line 356).
- ✅ `metadata.actual_cost_usd` written from `TerminalReport.Cost` on terminal-state — present (line 357).

**What was load-bearing in F.7.17.9 acceptance and is at risk of loss:**

- **MockAdapter polymorphism test against the monitor** (F.7.17.9 line 590: "Test using MockAdapter: monitor processes mock-emitted lines correctly (proves polymorphism)"). **NOT explicitly present in F.7.4 acceptance.** F.7.4's test scenarios use claude fixtures only.
- **"ZERO references to claude-specific event types in monitor code (no `system/init`, `assistant`, `result` literals)"** — F.7.17.9 line 589 was an explicit code-grep assertion. F.7.4's "monitor stays CLI-agnostic" (line 356) covers the spirit but not the grep-able assertion form.
- **Recorded claude-stream-trace regression test** — F.7.17.9 line 591/595 named "recorded `testdata/claude_stream_minimal.jsonl` processes identically to pre-refactor F.7.4 baseline." Since F.7.4 builds CLI-agnostic from inception, there's no "pre-refactor baseline" to regress against; F.7.4's own happy-path fixture tests (test scenarios at line 360–365) cover the equivalent. The regression-against-prior-baseline framing is moot.

**Net assessment of absorption:** F.7-CORE F.7.4 covers 7-of-8 of F.7.17.9's load-bearing acceptance criteria. The missing item is **explicit MockAdapter+monitor polymorphism test**. F.7.17.4 (MockAdapter fixture) ships the adapter and a contract conformance test; F.7.17.5 (dispatcher wiring) covers mock-injection at the spawn-command layer. But the **monitor's** polymorphism — that `monitor.go` actually consumes a Mock adapter without branching on cli_kind — is not asserted by any F.7-CORE or F.7.17 droplet's acceptance criteria post-merge.

This is the load-bearing absorption gap N3 explicitly flagged.

**Verdict: REFUTED-WITH-NIT.** STATUS callout is correctly placed; F.7-CORE F.7.4 absorbs 7-of-8 acceptance items. The remaining item — **monitor-side MockAdapter polymorphism test** — should either (a) be added as an explicit acceptance criterion to F.7-CORE F.7.4 ("test scenarios include a MockAdapter-driven monitor run asserting polymorphism"), or (b) be added as an explicit acceptance criterion to F.7.17.4 ("MockAdapter fixture is exercised against `monitor.go` to assert no cli_kind branching"). Builder dispatching F.7.4 today might write only claude-fixture tests and call the droplet done. Recommend resolving before F.7.4 dispatches.

### 1.4 V4 — F.7.18 REVISIONS section

**Evidence:** F.7.18 sub-plan lines 495–512:

- **REV-1 (line 499)** — explicitly names lines 26, 34, 68 as the body references to "per-binding `command`, `args_prefix`, `env`, `cli_kind`" being SUPERSEDED. Replacement framing: "F.7.17 Schema-1 (F.7.17.1) now ships ONLY `Env []string` + `CLIKind string` on `AgentBinding`." ✅
- **REV-2 (line 505)** — L4 closed env baseline expansion. ✅
- **REV-3 (line 509)** — `Tillsyn` struct extension policy: F.7.18.2 owns initial declaration; F.7-CORE F.7.1 + F.7.6 add fields; F.7.18.2 acceptance MUST include strict-decode unknown-key test. ✅

**Verdict: REFUTED.** R3 V1 (F.7.18 missing REVISIONS) closed.

### 1.5 V5 — Master PLAN §6.5 REVISIONS-first builder discipline

**Evidence:** Master PLAN lines 204–210:

```
## 6.5 Builder Spawn-Prompt Discipline (REVISIONS-first reading)

Sub-plans have REVISIONS POST-AUTHORING sections at the bottom that SUPERSEDE conflicting body text. Every F.7-touching builder spawn prompt MUST begin with:

> "Before reading the body of `<sub-plan>.md`, read the REVISIONS POST-AUTHORING section at the bottom of the file — it supersedes any conflicting body text. If a droplet body says one thing and REVISIONS says another, REVISIONS wins."

Orchestrator-procedural; does not require sub-plan rewrites.
```

The directive is verbatim quoted. §6.5 is positioned just before §7 Per-Droplet QA Discipline. Visible to any orchestrator reading the master PLAN top-to-bottom before dispatching builders.

**Verdict: REFUTED.** R3 N1 (REVISIONS reader-order) closed.

---

## 2. New Round-4 Attacks (N1–N4)

### 2.1 N1 — STATUS callouts mid-document: skim-failure attack

**Attack scenario:** A builder (or dispatcher) lands on F.7.17.9 or F.7.17.10 by line-number reference (e.g. master PLAN line 91 still names "CLI-agnostic monitor refactor | F.7.17.9 | F.7.17"; F.7.17 sub-plan line 130 still has F.7.17.9 in the DAG diagram). They jump to the F.7.17.9 / F.7.17.10 section header. They see a long body. Will they notice the STATUS callout?

**Evidence:**

- F.7.17.9 section header (line 568) — `### 4c.F.7.17.9 — CLI-agnostic monitor refactor — **MERGED INTO F.7-CORE F.7.4 per REVISIONS REV-7**` — the section header itself carries the **MERGED** marker in bold. A grep-skim sees it. ✅
- F.7.17.9 line 570 — `**STATUS: MERGED.**` in bold callout, first line of body, includes `Builders MUST NOT implement this droplet as a separate unit.` ✅
- F.7.17.10 section header (line 614) — `### 4c.F.7.17.10 — Marketplace install-time confirmation (paper-spec) — **REMOVED per REVISIONS REV-4**` — bold REMOVED in header. ✅
- F.7.17.10 line 616 — `**STATUS: REMOVED.**` first line of body, `Builders MUST NOT implement this droplet.` ✅

**Skim-failure analysis:** The bold STATUS callouts are placed at the FIRST line of the section body (immediately after the section header) AND the section header itself carries the bold MERGED/REMOVED marker. A builder cannot read the body without reading the callout — it's the very first prose. Combined with §6.5's REVISIONS-first directive, a builder is doubly-guarded: (a) the spawn prompt directs them to REVISIONS first; (b) if they skip directly to the body, the STATUS callout is the first thing they hit.

**Strength of mitigation:** Strong. The only failure mode is a builder who (i) ignores §6.5 directive, (ii) ignores the bold section-header MERGED/REMOVED marker, (iii) ignores the bold first-line STATUS callout, AND (iv) reads only the historical body content. That's four sequential failures of mechanical eye-tracking; not a realistic builder profile.

**Verdict: REFUTED.** Mitigation strength is sufficient.

### 2.2 N2 — F.7.17 droplet count = 9 verification

**Counting evidence (F.7.17 sub-plan section headers):**

| ID | Header line | STATUS |
|----|------|--------|
| 4c.F.7.17.1 | 159 | active |
| 4c.F.7.17.2 | 232 | active |
| 4c.F.7.17.3 | 286 | active |
| 4c.F.7.17.4 | 340 | active |
| 4c.F.7.17.5 | 380 | active |
| 4c.F.7.17.6 | 436 | active |
| 4c.F.7.17.7 | 482 | active |
| 4c.F.7.17.8 | 525 | active |
| 4c.F.7.17.9 | 568 | **MERGED** (per REV-7) |
| 4c.F.7.17.10 | 614 | **REMOVED** (per REV-4) |
| 4c.F.7.17.11 | 622 | active (renumbered to F.7.17.10 per REV-1, then renumbered again to F.7.17.9 per REV-7's claim — but the section header at line 622 still says "renumbered to 4c.F.7.17.10 per REVISIONS REV-1") |

Active count: 8 (F.7.17.1 through F.7.17.8) + 1 (F.7.17.11 alias) = **9 droplets**. Master PLAN line 68 + line 182 say 9. Match.

**However**, F.7-CORE REV-10 (line 1066–1068) says: **"F.7.17 (10 after marketplace droplet removal)"** + total **35**. This is stale — REV-10 wasn't updated when REV-7 (monitor merge) landed. F.7.17 sub-plan REV-7 (line 724) correctly says 9. Master PLAN line 182 correctly says 34 total. **F.7-CORE REV-10 has an arithmetic drift.**

Additional drift surfaces:

- **Master PLAN line 91** canonical-mapping table still lists `| CLI-agnostic monitor refactor | F.7.17.9 | F.7.17 | F.7.4 retro-edit |` as a live row — should be marked MERGED or removed.
- **Master PLAN line 92** says `| Adapter-authoring docs | F.7.17.10 | F.7.17 |`. After REV-7 the adapter-authoring docs final ID should be F.7.17.9 per F.7.17 sub-plan REV-7 line 724 ("renumbered F.7.17.11 → F.7.17.9 adapter docs"). But the F.7.17 sub-plan section header at line 622 still says "renumbered to 4c.F.7.17.10 per REVISIONS REV-1" without picking up REV-7's renumber. **Master PLAN and F.7.17 sub-plan agree at F.7.17.10 for adapter docs**; F.7.17 REV-7 is the outlier.
- **Master PLAN line 107** still has the line `F.7.17.9 (CLI-agnostic monitor refactor) blocked_by F.7-CORE F.7.4 (initial monitor implementation in claude adapter)` — F.7.17.9 doesn't exist as a droplet anymore.
- **F.7.17 sub-plan line 153** hard-constraint still says `4c.F.7.17.9 (monitor refactor) MUST land after 4c.F.7.17.5`.
- **F.7.17 sub-plan DAG diagram at lines 130–139** still shows F.7.17.9 + F.7.17.10 as live nodes.

**Verdict: NIT (multi-locus arithmetic + canonical-ID drift).** The droplet count of 9 is correct; master PLAN's 34 total is correct. But several stale references survive:

1. F.7-CORE REV-10 says F.7.17 = 10 + total 35. Should be 9 + 34.
2. F.7.17 final ID for adapter-authoring docs is ambiguous: F.7.17.10 per master PLAN + F.7.17 line 622, vs F.7.17.9 per F.7.17 REV-7 line 724.
3. Master PLAN line 91, 92, 107 + F.7.17 line 130–139, 153 carry stale F.7.17.9 / F.7.17.10 references that should be marked MERGED/REMOVED in-place or struck.

**Recommendation:** orchestrator picks ONE final ID for the adapter-authoring docs droplet (F.7.17.9 OR F.7.17.10 — pick consistent) and rewrites master PLAN lines 91/92/107 + F.7-CORE REV-10 + F.7.17 lines 130/139/153 + F.7.17 line 622 in one sweep. Not a blocker for dispatch — REVISIONS callouts make the intent unambiguous — but the canonical ID drift compounds future maintainability cost.

### 2.3 N3 — F.7-CORE F.7.4 absorption coverage

**Already analyzed in §1.3.** Net: F.7.4 covers 7-of-8 of F.7.17.9's load-bearing acceptance. The missing item is **monitor-side MockAdapter polymorphism test** — F.7.4's test scenarios use only claude fixtures.

**Why this matters concretely:** F.7.17.4 ships MockAdapter and a CLIAdapter contract conformance test (which exercises `BuildCommand` / `ParseStreamEvent` / `ExtractTerminalReport` against both adapters). F.7.4 ships claude monitor wiring + claude fixtures. NEITHER droplet asserts that **`monitor.go` runs end-to-end against a Mock adapter without claude-specific branching**. A builder could implement F.7.4's monitor wiring such that it secretly assumes `cli_kind == "claude"` somewhere (e.g. hardcoded fixture path resolution) and the F.7.4 test suite would pass green.

**Verdict: NIT (absorption gap).** Recommend ONE of:

- (a) Add to F.7-CORE F.7.4 acceptance: "Test scenarios include a MockAdapter-driven monitor run that asserts: monitor processes mock-emitted lines correctly without any cli_kind branching in monitor.go (verified via `grep -L 'system/init\\|assistant\\|result' internal/app/dispatcher/monitor.go` returning the file)."
- (b) Add to F.7.17.4 acceptance: "MockAdapter contract test is exercised against `monitor.go` (not just adapter-method-level), proving end-to-end polymorphism."

Recommend (a) — F.7.4 is the monitor's owner. Either path closes the gap. Not blocking — CLI-agnostic monitor (line 356) covers the spirit; the absent grep-able assertion just lets the implementation drift.

### 2.4 N4 — REVISIONS-first directive enforceability

**Attack scenario:** §6.5 says "Every F.7-touching builder spawn prompt MUST begin with..." — it's a procedural rule the orchestrator must remember to apply at every dispatch. If the orchestrator forgets, the directive is silently dropped and the spawn prompt looks normal. The builder reads the body top-down, hits the conflicting prose, and implements the wrong thing.

**Verification mechanisms available today:**

- None automatic. Pre-cascade today, orchestrator+dev manually craft each spawn prompt. The §6.5 quoted line is text-in-CLAUDE.md for the orchestrator — it's a memory-aid, not a runtime gate.
- Post-Drop-4 dispatcher: in principle the dispatcher could template the spawn prompt and ensure §6.5's directive is always prepended. But Drop 4a/4b shipped manual-trigger; the spawn-prompt-template piece is part of F.7 itself. So §6.5 is a self-bootstrapping rule — F.7-touching builders need it before F.7 ships.
- Plan-QA on each builder spawn prompt: not part of the cascade today (no pre-spawn-prompt-review droplet exists).

**Failure mode:** orchestrator dispatches the F.7.17.1 builder using a normal go-builder-agent spawn prompt; builder reads F.7.17 body top-down; lands on lines 15–16 (Command + ArgsPrefix + validateAgentBindingCommandTokens); implements all four fields plus the validator. QA-proof / QA-falsification spotting this requires the QA agents to also have the REVISIONS-first directive in their spawn prompts.

**Mitigation strength:** Procedural-only. Easy to forget. The recommendation already says "Orchestrator-procedural; does not require sub-plan rewrites" — which acknowledges the failure mode without fixing it.

**Verdict: NIT.** Recommend one of:

- (a) Make §6.5's quoted line a hard-required block in any spawn prompt for `kind=build` whose paths intersect F.7.* — the orchestrator self-checks before dispatch. (Manual today; encoded as a checklist item.)
- (b) Rewrite each affected sub-plan's body in-place to avoid the supersession problem entirely. F.7.17 has the most surface; F.7.18 is small enough to in-place edit; master PLAN line 195 is a one-line touch. This is the bulletproof fix but adds round-5 plan churn.
- (c) Defer to a Drop 4c.5 "plan-doc cleanup" droplet that does the in-place rewrites once F.7 lands; rely on §6.5 + REVISIONS callouts during F.7 dispatch.

Recommend (a) — checklist-item discipline, no rewrite churn. Not blocking. The orchestrator is responsible for spawn prompt construction; §6.5 names the rule clearly enough.

---

## 3. R3 NIT Regression Check

R3's PASS / PASS-WITH-NIT items inspected for regression:

- **V2 (F.7.18 REV-1 lines 26/34/68 supersession)** — PASS in R3 because absent; now present and correct in R4. No regression.
- **V3 (F.7-CORE REV references)** — PASS in R3; F.7-CORE REVISIONS unchanged in R4. No regression.
- **V4 (F.7.17 REVISIONS REV-1 / REV-4 / REV-5)** — PASS in R3; REV-7 + REV-8 added in R4 without altering REV-1/REV-4/REV-5. No regression.
- **V5 (master PLAN §3 / §5)** — PASS in R3; master PLAN §3 (line 41 L2 — no `command` override) and §5 (canonical-mapping table) unchanged in R4 EXCEPT lines 91/92/107 still carry stale F.7.17.9 references — flagged in N2 NIT, not regression because those lines were stale in R3 too (R3 didn't catch them because F.7.17.9 was still a live droplet then).
- **N2 (F.7.18 NITs from R3)** — N/A (R3 N2-N6 covered different ground).
- **N3 (Q3 spawn-aggregator wiring open)** — Q3 still open, F.7.18 sub-plan line 469 still flags it. F.7-CORE has no droplet absorbing the wiring. **Pre-existing open question, not introduced by R4.** Recommend filing as a Drop 4c open question or absorbing into F.7-CORE F.7.3b's bundle-render droplet (it's the natural seam — system-append.md is the inline-context destination per F.7-CORE line 294). Not blocking — engine + bundle exist; wiring is a thin call-site addition.
- **N4 (default-template seed drift)** — F.7.18.5 acceptance unchanged in R4. No regression.
- **N6 (bundle filename collision Q4)** — F.7.18 Q4 still open. No regression.

**Aggregate:** No R3-NIT regressions. Two pre-existing R3 open questions (Q3 spawn-pipeline-aggregator wiring, Q4 filename convention) remain — both are open-for-builder-dispatch-time-resolution, not blockers.

---

## 4. Summary

### 4.1 R3 blockers verified closed

- V1 (residual command/args_prefix in F.7.18) — REFUTED via REV-1 explicit-line-naming.
- V1 (F.7.17.10 marketplace droplet body) — REFUTED via STATUS REMOVED + body deletion.
- N5 (F.7.17.9 redundant) — REFUTED via STATUS MERGED + F.7-CORE F.7.4 absorbing 7-of-8 acceptance items.
- N1 (REVISIONS-first reader-order) — REFUTED via master PLAN §6.5.

### 4.2 New R4 NITs

- **N1 (STATUS callouts skim)** — REFUTED. Bold callouts at section-header + first-line; mitigation strong.
- **N2 (droplet count + canonical ID drift)** — NIT. F.7-CORE REV-10 says 35 should say 34; F.7.17 REV-7 vs sub-plan section header disagree on adapter-docs final ID (F.7.17.9 vs F.7.17.10); master PLAN lines 91/92/107 + F.7.17 lines 130/139/153 + F.7.17 line 622 carry stale F.7.17.9/F.7.17.10 references.
- **N3 (F.7-CORE F.7.4 absorption gap)** — NIT. 7-of-8 acceptance items absorbed; missing is monitor-side MockAdapter polymorphism test. Recommend adding to F.7.4 acceptance.
- **N4 (REVISIONS-first directive enforceability)** — NIT. Procedural rule with no runtime gate. Recommend orchestrator-side checklist discipline before dispatch.

### 4.3 Pre-existing residuals (carried from earlier rounds)

- **Master PLAN line 195** still names `AgentBinding.Command []string` as a schema addition. Master PLAN §3 L2 contradicts; not blocking but should be rewritten.
- **F.7.18 Q3 (spawn-pipeline-aggregator wiring ownership)** — open, no F.7-CORE droplet absorbs it. Recommend folding into F.7-CORE F.7.3b's bundle-render acceptance.
- **F.7.18 Q4 (filename collision convention)** — open, low-risk resolve-at-dispatch.

### 4.4 Counterexamples produced

None CONFIRMED. All R3 blockers verified-closed. Four R4 NITs surface; none rise to NEEDS-REWORK because:

1. Each affected droplet's CONCRETE acceptance criteria is unambiguous when read with §6.5's REVISIONS-first directive.
2. The droplet count drift + canonical ID drift is cosmetic; no builder is mis-routed by it.
3. The absorption gap (N3) is a single missing acceptance criterion that orchestrator can append at F.7.4 dispatch time without re-running plan-QA.
4. The §6.5 enforceability concern (N4) is procedural and well-known; the orchestrator owns spawn-prompt construction.

---

## 5. Final Verdict

**PASS-WITH-NITS.**

The plan is dispatch-ready. Builders may fire on F.7.10 / F.7.9 (independent leaves), F.7.17.1 (Schema-1, blocking), F.7.18.2 (Schema-3, blocking) per master PLAN §5 sequencing.

**Recommended pre-dispatch micro-touches (non-blocking, orchestrator-time):**

1. **F.7.4 acceptance addition (N3):** orchestrator appends one acceptance criterion to F.7-CORE F.7.4 at dispatch: "Test scenarios include a MockAdapter-driven monitor run; assert no claude-specific event-type literals in `monitor.go` (verified via `grep -L 'system/init\\|assistant\\|result' internal/app/dispatcher/monitor.go` returning the file)."
2. **Spawn-prompt checklist (N4):** orchestrator confirms every F.7-touching builder spawn prompt prepends the §6.5 REVISIONS-first quote.
3. **Stale-ID cleanup (N2 + V1 master-PLAN):** when a low-friction window opens (drop pause, planning down-time), rewrite master PLAN lines 91, 92, 107, 195; F.7.17 lines 130, 139, 153, 622; F.7-CORE REV-10. Optional; not blocking.

If the orchestrator applies (1) and (2), F.7 is fully dispatch-ready. (3) is hygiene.

---

## Hylla Feedback

`N/A — review touched non-Go files only.` All evidence gathered via direct `Read` on plan MDs and `rg` line-search across the four plan files. No Hylla queries issued. No miss to record.
