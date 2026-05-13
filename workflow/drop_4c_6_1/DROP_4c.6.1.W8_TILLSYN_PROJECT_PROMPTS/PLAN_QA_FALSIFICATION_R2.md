# W8 Plan-QA Falsification Verdict — Round 2

**Verdict:** PASS
**Author:** go-qa-falsification-agent (round 2, background-mode)
**Date:** 2026-05-12
**Plan reviewed:** `workflow/drop_4c_6_1/DROP_4c.6.1.W8_TILLSYN_PROJECT_PROMPTS/PLAN.md` (round-2 absorption)
**Sibling _BLOCKERS reviewed:** `workflow/drop_4c_6_1/DROP_4c.6.1.W8_TILLSYN_PROJECT_PROMPTS/_BLOCKERS.toml`
**L1 cross-ref:** `workflow/drop_4c_6_1/PLAN.md` lines 706-893, W8 directive + R3-NIT5 GUIDANCE.
**Sibling round-2 plans:** W1 D3 (resolver subdir-per-group), W5 D6 (loader nested-schema), W6 D9 (palette.ts nested-schema).

---

## Attack Surface Coverage

Every attack vector from the spawn prompt was attempted:

| # | Attack | Result |
|---|---|---|
| 1 | `model: orchestrator-managed` parsing | REFUTED-WITH-NIT — see NIT1 below |
| 2 | Cross-wave bindings.json schema consistency (W5/W6) | REFUTED — nested schema match exact across W5/W6/W8 (verified) |
| 3 | D21 fixture path vs W1 D3 post-change resolver signature | REFUTED — fixture path `<tmpdir>/.tillsyn/agents/go/builder-agent.md` matches W1's post-change `filepath.Join(worktree, projectAgentsSubdir, group, basename)` (verified at render.go:881 pre-change + W1 D3 KindPayload shape_hint) |
| 4 | 22-droplet count across all surfaces | REFUTED — narrative L8, D-list L1161, Wave A+C enumeration, _BLOCKERS.toml all = 22 |
| 5 | Plan-QA vs Build-QA differentiation (D3↔D5, D4↔D6, D13↔D15, D14↔D16) | REFUTED — each pair declares distinct Evidence Sources (PLAN.md/REVISION_BRIEF/SKETCH for plan-QA vs Go source/git diff/test output for build-QA) and distinct What-To-Check / Attack-Vectors sections per SKETCH §3.1 |
| 6 | From-scratch citation sufficiency (D8/D9/D10/D18/D19/D20) | REFUTED — each cites multiple concrete source sections (CLAUDE.md sections, WORKFLOW.md phase, memory entries) sufficient for ≥1000-char prompts |
| 7 | `.gitignore` re-include ordering correctness | REFUTED — parent `.tillsyn/*` (wildcard children exclusion); `!.tillsyn/agents/` (dir re-include) precedes `!.tillsyn/agents/**/*.md` (glob); AcceptanceCriteria note explicitly captures the ordering invariant |
| 8 | bindings.json ID collision with stil baseline tillsyn ns | REFUTED — local IDs {`dispatch`, `plan`, `archive`, `settings`, `help`} disjoint from stil baseline tillsyn-ns IDs {`new-drop`, `complete-drop`, `handoff`, `comment`} (verified from stil/main/src/bindings/baseline.json L100-108); `archive` also exists in `ro-email` ns but that's a separate product namespace — no collision |
| 9 | `binding.AgentName` drives rendered file path | REFUTED — verified at render.go:327 `filepath.Join(bundle.Paths.Root, pluginSubdir, agentsSubdir, binding.AgentName+".md")`; D21 round-2 added explicit bullet `binding.AgentName = "builder-agent"` |
| 10 | Hylla in tools list vs Hylla-OFF directive | REFUTED — Hylla-OFF applies to the current orchestration spawn cycle; authored prompts govern future dogfood when Hylla is operational per L1 PLAN.md line 835. Round-2 framing correct |
| 11 | Round-2 Changes block completeness | REFUTED — all 4 round-1 FFs (Proof FF1.1/FF1.2 + Fals FF1/FF2) absorbed; 4 NITs ABSORBED (Proof NIT1/NIT2/NIT3/NIT4 = Fals NIT1/NIT2); 4 NITs DEFERRED (Fals NIT3/NIT4/NIT5/NIT6) with reasons |
| 12 | D21 `blocked_by` enumeration | REFUTED — 22 entries (D0 + D1-D20 + 4c.6.1.W1) — 21 internal nodes + 1 cross-wave = 22 |
| 13 | bindings.json schema_version alignment | REFUTED — W8 uses `schema_version: 1`; stil baseline uses `schema_version: 1`; W5/W6 loaders consume `product_extensions.tillsyn.commands` from JSON and don't gate on schema_version explicitly — documentation-grade for v1 |

---

## Findings

### FF — none

No CONFIRMED FFs in this round. All round-1 FFs were ABSORBED into the round-2 plan correctly. Counter-attacks against the absorption fail to land.

---

## NITs (each gets ABSORB or DEFERRED-AS-NIT with reason)

### NIT1 — `model: orchestrator-managed` is a sentinel value, not a model alias

**Disposition:** DEFERRED-AS-NIT.

Claude Code's `model:` frontmatter field is documented to accept bare names (`sonnet`/`opus`/`haiku`) or full version IDs. `orchestrator-managed` is NEITHER a model name NOR a version ID — it's a sentinel value indicating the kind is handled by the orchestrator and never spawned as a subagent. This invention is new in W8 round-2 (no precedent in `~/.claude/agents/` — none of the 10 system agent files name `orchestrator-managed` as a model, per round-1 PROOF verdict's empirical check).

Three observations soften the concern:

1. **Post-render validator does not check `model:` field validity.** `validateAgentBodyShape` at render.go:354 checks frontmatter delimiters + `name:` field + body length + content markers (`# PLACEHOLDER` / `# Section 0` / `## Role`). The `model:` value is opaque to the validator.
2. **These prompts are never spawned as subagents.** The 4 orchestrator-managed kinds (`closeout`, `refinement`, `discussion`, `human-verify`) are handled by the orchestrator directly per CLAUDE.md cascade table. The prompt files document orchestrator behavior; the dispatcher does NOT use them as agent spawn targets.
3. **The round-2 plan explicitly documents this** in line 111: "`model: orchestrator-managed` is a string value indicating orchestrator-managed scope — matches Tillsyn's orchestrator-managed-role convention (R10-D4)."

**Reason for deferral:** the field is inert in the current dispatcher path. The choice is semantically clear and documented. Promoting to ABSORB would either (a) replace `orchestrator-managed` with a placeholder like `sonnet` that's *also* inert but less semantic, or (b) drop the `model:` field entirely, which violates frontmatter shape uniformity. Either alternative trades one minor concern for another. If a future Claude Code version validates `model:` at agent-load time (vs spawn-time), this would resurface as a real FF — track as a future refinement (`MODEL-SENTINEL-R1`).

### NIT2 — D9/D19 commit-message tools list omits Hylla but L1 directive line 835 does not enumerate commit-message

**Disposition:** DEFERRED-AS-NIT.

Round-2 plan's commit-message-agent gets `tools: Read` only (D9 line 559, D19 line 917). L1 PLAN.md line 835 enumerates tool defaults for `qa-proof/qa-falsification/research/planning`, `builder`, and `closeout/orchestrator-managed` — but NOT for commit-message. The W8 plan's choice (`Read` only) is defensible — commit-message is mechanical — but isn't explicitly L1-sanctioned.

**Reason for deferral:** the commit-message role's purpose (form a one-line conventional commit from a diff) does not require Hylla; adding it would be expand-without-cause. L1 silence is acceptable for a haiku-model role whose toolset is naturally narrow. If the cascade methodology adds a commit-message tools-row to L1 directives in a future refinement, surface as `COMMIT-MSG-TOOLS-R1`.

### NIT3 — D8/D18 `tools: Read, Edit, Write, Grep, Glob` table cell uses prose paraphrase but per-droplet AC uses comma list — minor formatting drift

**Disposition:** DEFERRED-AS-NIT.

PLAN.md table line 107: `closeout-agent | orchestrator-managed | (orchestrator-managed — same as builder scope)`. The tools cell is prose, NOT a comma-separated list. Per-droplet AC at line 513 (D8) and 882 (D18) correctly enumerate: `tools: Read, Edit, Write, Grep, Glob`. The table cell's prose form would not pass a strict frontmatter-conformance grep (`tools: Read, Edit, Write, Grep, Glob` is the actual frontmatter value).

**Reason for deferral:** the table is documentation; the per-droplet AC is the authoritative tools list builders consult. The prose cell is for human-readability, not machine parsing. Builders authoring D8/D18 will use the AC bullet, not the table cell. Promoting to ABSORB would force a minor table touch-up with no behavior change.

### NIT4 — `model: orchestrator-managed` collides namewise with `name: orchestrator-managed` in D10/D20

**Disposition:** DEFERRED-AS-NIT.

D10 (line 598) and D20 (line 952) both set `name: orchestrator-managed` AND `model: orchestrator-managed`. The agent file name "orchestrator-managed" carrying a model value "orchestrator-managed" is internally consistent but reads ambiguously. A reader scanning the frontmatter might mistakenly conflate name-field and model-field semantics.

**Reason for deferral:** the file naming follows L1 PLAN.md scope decree (".tillsyn/agents/go/orchestrator-managed.md"). The model sentinel `orchestrator-managed` is the same string by independent convention. Renaming either would break the L1-locked decision. Acceptable as authored.

---

## Cross-Wave Concerns

These warrant orchestrator escalation (not L2-fixable):

### CW1 — D21 W1 dependency on exact post-change `readProjectTierAgent` signature

Per round-1 fals CW2 (re-verified 2026-05-12): current `readProjectTierAgent(projectWorktree, basename string)` at render.go:877. W1 D3 (verified in W1 round-2 PLAN.md line 278) changes to `(projectWorktree, group, basename string)` with path `filepath.Join(worktree, projectAgentsSubdir, group, basename)`.

D21 builder must verify W1's post-change signature at build time before authoring the test. PLAN.md U2 already acknowledges this. **No re-escalation needed** — the dependency is encoded as `blocked_by 4c.6.1.W1` on D21.

### CW2 — `extends_path` field is documentation-only for v1 (W5/W6 loaders don't consume it)

W5 D6 loader uses `LoadBindings(baselineJSON []byte, localPath string)` with baseline embedded; W6 D9 palette.ts uses `fetch('/stil-baseline.json')` + `fetch('/.tillsyn-bindings.json')` via Astro vite proxy. Neither consumes the `extends_path` field. The field exists in the W8-authored file as documentary intent only.

The round-2 PLAN.md ContextBlocks at line 195 already documents the loader-CWD invariant. No FF. Tracked as deferred NIT5 (fals round-1) for future loader-extension hardening.

---

## Verdict Summary

- **Verdict line:** **PASS**.
- **FF count:** 0 CONFIRMED. All round-1 FFs absorbed correctly into round-2.
- **NIT count:** 4 total, all DEFERRED-AS-NIT with reason. None of the NITs blocks builder spawn.
- **Cross-wave concerns:** 2 (CW1 D21↔W1 sig, CW2 extends_path) — both already encoded in PLAN.md (CW1 as `blocked_by`, CW2 as documentary intent + NIT5 deferral).
- **Attack-vector exhaustion:** 13 prompt-specified attacks all attempted; 12 REFUTED, 1 REFUTED-WITH-NIT.

**The round-2 plan correctly absorbed every round-1 finding, the cross-wave bindings.json schema lines up exactly with W5 and W6 round-2 consumers, the D21 fixture path matches the post-W1 resolver shape, and the 22-droplet count is internally consistent across all 5 representations (narrative, Wave A enumeration, KindPayload children, D-list, _BLOCKERS.toml mirror).** No counterexample lands. Recommend orchestrator proceed to L2 build dispatch (D0 first, then D1-D20 parallel after D0 commit, then D21 after D0-D20 + W1 all complete).

---

## Hylla Feedback

N/A — action item touched non-Go files only (PLAN.md, _BLOCKERS.toml, JSON, MD prompts). The lone Go file referenced (`render.go`) was read for symbol-shape verification at lines 320-340 (validateBundle, validateAgentBodyShape) and 870-890 (readProjectTierAgent), not Hylla-queried. Hylla is OFF per spawn directive.

---

## Section 0 — Orchestrator-Facing

# Section 0 — SEMI-FORMAL REASONING

## Proposal

- **Premises:** W8 round-2 L2 plan absorbs round-1 PROOF (2 FFs + 4 NITs) + FALSIFICATION (2 FFs + 6 NITs) findings. Round-2 plan must be (a) internally consistent (droplet count, blocked_by graph, model/tools tables), (b) externally consistent with sibling round-2 plans (W1 D3 resolver, W5 D6 loader, W6 D9 palette.ts), (c) free of shipped-but-not-wired drift, (d) free of new FFs introduced by the absorption itself.
- **Evidence:** Direct reads of W8 round-2 PLAN.md (1171 lines), `_BLOCKERS.toml` (122 lines), round-1 PROOF + FALS verdicts, L1 PLAN.md lines 700-895, stil/main/src/bindings/baseline.json (117 lines), W1 round-2 PLAN.md lines 180-310, W5 round-2 PLAN.md lines 1-220, W6 round-2 PLAN.md lines 1-100 + 540-760, render.go lines 320-337 (validateBundle, agentPath at line 327) + 870-890 (readProjectTierAgent), current `.gitignore` (40 lines).
- **Trace or cases:** 13 attack vectors enumerated; each landed (CONFIRMED with repro) or bounced (REFUTED with reason). Trace verified: droplet count 22 matches across 5 representations; cross-wave nested schema matches W5/W6/W8; D21 fixture path matches post-W1 resolver; `.gitignore` re-include ordering correct.
- **Conclusion:** PASS. No counterexample lands.
- **Unknowns:** Cannot read system-agent files in `~/.claude/agents/` directly (sandboxed). Round-1 PROOF's empirical claim that they use bare `model: sonnet`/`opus` aliases is taken as given.

## QA Proof

- **Premises:** Every round-1 finding (4 FFs + 10 NITs) absorbed or deferred-with-reason in round-2 Changes block.
- **Evidence:** Round 2 Changes block at PLAN.md lines 10-27 enumerates every absorption + every deferral; cross-walk against round-1 PROOF (FF1.1, FF1.2, NIT1.3, NIT1.4, NIT1.5, NIT1.6) and round-1 FALS (FF1, FF2, NIT1, NIT2, NIT3, NIT4, NIT5, NIT6) confirms 1:1 coverage.
- **Trace or cases:** Each round-1 finding mapped to its round-2 absorption clause; deferrals all have explicit reason text.
- **Conclusion:** Evidence completeness confirmed.
- **Unknowns:** None.

## QA Falsification

- **Premises:** Try to break the PASS verdict.
- **Evidence + attack vectors attempted:**
  - *Attack — `model: orchestrator-managed` is unparseable*: REFUTED-WITH-NIT (post-render validator doesn't check model field; agents never spawned as subagents).
  - *Attack — cross-wave nested schema drift between W8/W5/W6*: REFUTED (W5 line 207, W6 line 663, W8 line 154-176 all use the same nested schema with 5-command local file).
  - *Attack — D21 fixture path mismatch*: REFUTED (W1 D3 KindPayload shape_hint at W1 round-2 PLAN.md line 278 explicitly states `filepath.Join(worktree, projectAgentsSubdir, group, basename)`; W8 D21 fixture at `<tmpdir>/.tillsyn/agents/go/builder-agent.md` matches).
  - *Attack — D21 missing `binding.AgentName`*: REFUTED (round-2 PLAN line 1000 explicitly added the AC bullet).
  - *Attack — from-scratch droplets insufficient citation*: REFUTED (each cites 4+ concrete sources).
  - *Attack — `.gitignore` re-include ordering broken*: REFUTED (per gitignore docs, `.tillsyn/*` excludes children, not the dir; `!.tillsyn/agents/` re-include precedes the glob; AC explicitly captures the invariant).
  - *Attack — bindings.json ID collision*: REFUTED (verified disjoint in tillsyn ns).
  - *Attack — Hylla in tools list vs Hylla-OFF directive*: REFUTED (Hylla-OFF applies to spawn cycle, not authored prompt content).
  - *Attack — silent drops in round-2 Changes block*: REFUTED (all 14 round-1 findings accounted for).
  - *Attack — schema_version drift*: REFUTED (all 1).
- **Conclusion:** No unmitigated counterexample. The PASS verdict survives adversarial review.
- **Unknowns:** None within the falsification scope.

## Convergence

- (a) QA Falsification produced no unmitigated counterexample to the PASS verdict.
- (b) QA Proof confirmed every round-1 finding was absorbed or deferred-with-reason.
- (c) Remaining unknowns are routed: `model: orchestrator-managed` sentinel design recorded as NIT1 with MODEL-SENTINEL-R1 future-refinement pointer; commit-message tools-list silence recorded as NIT2 with COMMIT-MSG-TOOLS-R1 pointer; system-agent file reads sandboxed (round-1 empirical claim accepted as given); D21↔W1 signature sync encoded as `blocked_by 4c.6.1.W1`.

Converged.
