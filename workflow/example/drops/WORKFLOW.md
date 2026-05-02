# `<PROJECT>` — Per-Drop Workflow (Cascade, MD-Only)

Canonical lifecycle doc for one drop, start to finish. **`<PROJECT>/CLAUDE.md` and `<PROJECT>/PLAN.md` link here rather than duplicate these rules.** Authoritative source for: drop directory shape, file lifecycles, phase order, subagent spawn cadence, restart recovery.

If you change a phase, change it here once. Do not copy the rules into `CLAUDE.md` or `PLAN.md`.

The cascade concept source (droplets, planner-calls-planner, package-level automated gates, planner-level LLM QA, ancestor re-QA on blocker failure) is `AGENT_CASCADE_DESIGN.md` at the project root. This file describes **operational mechanics** — how the cascade is executed with plain Markdown files and off-the-shelf Claude subagents, without a coordination runtime.

## Drop Directory Layout

Every drop is a **directory** under `drops/`. The directory name mirrors the `PLAN.md` container row title (`ALL_UPPERCASE_WITH_UNDERSCORES`).

```
drops/
├── WORKFLOW.md                     # this file
├── _TEMPLATE/                      # orch copies this for each new drop
│   ├── PLAN.md
│   ├── BUILDER_WORKLOG.md
│   └── CLOSEOUT.md
├── DROP_N_<NAME>/
│   ├── PLAN.md                     # durable
│   ├── _BLOCKERS.toml              # durable — sibling blocked_by ledger (only at dirs with >1 immediate child)
│   ├── PLAN_QA_PROOF.md            # round 1; round 2+ → PLAN_QA_PROOF_R2.md, _R3.md, ...
│   ├── PLAN_QA_FALSIFICATION.md    # round 1; round 2+ → PLAN_QA_FALSIFICATION_R2.md, ...
│   ├── BUILDER_WORKLOG.md          # durable
│   ├── BUILDER_QA_PROOF.md         # durable
│   ├── BUILDER_QA_FALSIFICATION.md # durable
│   └── CLOSEOUT.md                 # durable
└── DROP_N+1_<NAME>/
    └── …
```

### File Lifecycle

| File | Lifecycle | Owner |
|---|---|---|
| `PLAN.md` | **durable** — refined across plan-QA rounds; final at close | planner subagent edits, orch + builder + QA read |
| `_BLOCKERS.toml` | **durable** — present only at dirs with >1 immediate child; mirrors inline `Blocked by:` bullets from `PLAN.md` | planner subagent writes, orch + builder + QA read |
| `PLAN_QA_PROOF.md` | **durable** — round 1; round 2+ writes `PLAN_QA_PROOF_R2.md` / `PLAN_QA_PROOF_R3.md` / etc. Never `git rm`d. | qa-proof subagent writes |
| `PLAN_QA_FALSIFICATION.md` | **durable** — round 1; round 2+ writes `PLAN_QA_FALSIFICATION_R2.md` / etc. Never `git rm`d. | qa-falsification subagent writes |
| `BUILDER_WORKLOG.md` | **durable** — append `## Droplet N.M — Round K` per build attempt | builder subagent appends |
| `BUILDER_QA_PROOF.md` | **durable** — append `## Droplet N.M — Round K` per QA attempt | qa-proof subagent appends |
| `BUILDER_QA_FALSIFICATION.md` | **durable** — append `## Droplet N.M — Round K` per QA attempt | qa-falsification subagent appends |
| `CLOSEOUT.md` | **durable** — written once at drop close | orch (or `Role: commit` builder) writes |

**Durable, never `git rm`d.** Every QA round, every builder round, every plan revision stays in the tree. Two persistence patterns are used depending on the file:

- **Round-suffix files (plan-QA):** round 1 writes `PLAN_QA_PROOF.md` / `PLAN_QA_FALSIFICATION.md`; round 2 writes `PLAN_QA_PROOF_R2.md` / `PLAN_QA_FALSIFICATION_R2.md`; round 3 writes `_R3.md`; and so on. One file per round; every round visible in tree.
- **Append-round files (builder + build-QA):** `BUILDER_WORKLOG.md`, `BUILDER_QA_PROOF.md`, `BUILDER_QA_FALSIFICATION.md` are single files where each round appends a `## Droplet N.M — Round K` heading (e.g. `## Droplet 1.3 — Round 2`). Plan-phase findings use `## Plan — Round K` instead of a droplet number.

**Audit trail integrity is load-bearing.** Never `git rm` a QA worklog or builder worklog — every adversarial review and every build attempt must remain in the tree, not buried in `git log` archaeology.

### Sub-Drops (Cascade Recursion)

When a planner decides a droplet is not yet atomic (too many files, too many packages, too much acceptance to verify), it emits a **sub-drop** — a child `DROP_N.M_<NAME>/` directory nested under the parent. The sub-drop has its own `PLAN.md`, `BUILDER_WORKLOG.md`, QA files, and `CLOSEOUT.md`. A sub-drop's container row lives in the parent drop's `PLAN.md` Planner section (not the project-root `PLAN.md`).

Naming: dotted levels (`DROP_1.2.3_<NAME>`). Maximum nesting: unlimited in principle; 3–4 levels in practice before the planner should reconsider whether the parent drop's scope is right-sized.

A parent drop cannot close while any sub-drop is incomplete.

### `_BLOCKERS.toml` — Sibling Blocker Ledger

Every dir with more than one immediate child (sub-drops OR droplets) carries a `_BLOCKERS.toml` file. Scope: **immediate children of this dir only** — cross-level blockers ride on the parent-close-waits-for-child rule, not on this file.

Shape:

```toml
# _BLOCKERS.toml — drops/DROP_N_<NAME>/
# Immediate-children sibling blocker ledger.

[[blockers]]
node = "1.3"                 # droplet ID (or sub-drop dir name)
blocked_by = ["1.1", "1.2"]
reason = "file foo.go written by 1.1, test harness set up by 1.2"
```

**Today this file is a coordination hint.** The planner writes it; orch and build-QA read it to check the dispatch graph at a glance; humans and LLMs enforce the discipline. The file mirrors the inline `Blocked by:` bullets in each droplet's `PLAN.md` row — if the two disagree, `PLAN.md` is truth and `_BLOCKERS.toml` is stale (regenerate from `PLAN.md`).

**In MD-folder-only mode (no coordination runtime), `_BLOCKERS.toml` is always a coordination hint.** Discipline is human/LLM-enforced — the planner populates it, orch and build-QA consult it, dev adjudicates conflicts. No runtime blocks `in_progress` transitions for you.

**When this project sits alongside a coordination runtime** (in the tillsyn-native flavor: Tillsyn Drop 1 promotes `paths` / `packages` to first-class `ActionItem` domain fields; Tillsyn Drop 4 ships the dispatcher that hard-refuses `in_progress` transitions while any `blocked_by` entry is still unresolved), `_BLOCKERS.toml` becomes the one-shot migration source at migration time: each `[[blockers]]` row loads into the runtime `blocked_by` field. **Migration is two-step:** (1) regenerate the TOML from `PLAN.md` inline bullets so the two are guaranteed in sync, (2) then load into the runtime. Never import the TOML directly without the sync step — stale TOML could install a regression the moment runtime enforcement turns on.

**Format choice.** TOML (not YAML, not MD): strict parser catches typos early, `[[blockers]]` array-of-tables is readable without indent anxiety, and — for projects that sit alongside a Go coordination runtime — the `pelletier/go-toml/v2` dependency is already in the stack.

**Cross-subtree leaf-level ordering is NOT expressible in `_BLOCKERS.toml`.** When a leaf in sub-tree A needs to wait on a specific leaf in sub-tree B, the common-ancestor `_BLOCKERS.toml` can only name immediate children (the sub-trees themselves, not leaves inside). When this arises, the planner escalates the dependency to the nearest-common-ancestor-immediate-children pair — a coarser block with a larger blast radius — or refactors package boundaries so the leaf-level dependency becomes same-subtree. YAGNI for rare cross-subtree leaf cases; if they show up often, that's signal the decomposition wants reshaping.

## Phase Order

Plan → Plan QA → Discuss + Cleanup → (loop until plan good) → Build (per droplet) → Build QA (per droplet) → (loop until droplet good) → Verify → Closeout → next drop.

**Follow these phases in order, exactly as written.** No skipped phases. No reordered phases. No shortcut paths. If a phase looks redundant for a particular drop (e.g. plan-QA on a one-droplet cleanup drop), return the question to the dev — do not unilaterally drop the phase.

Phase exits gate the next phase: build cannot start while plan-QA findings are open; closeout cannot start while any droplet has open build-QA rounds; the next drop cannot start until the current drop's `CLOSEOUT.md` is written and its container row in the project-root `PLAN.md` is flipped to `done`.

---

## Agent Spawn Contract

The agents in `~/.claude/agents/` (`go-builder-agent`, `go-planning-agent`, `go-qa-proof-agent`, `go-qa-falsification-agent`, and FE variants) are **global** — shared with projects that may use a coordination runtime. Their agent definitions reference runtime tools (`till_*`, capability leases, attention items, handoffs) that this project does not use. **This project does not use any of that.** Every spawn from this project's orchestrator overrides those instructions.

This section is the **single canonical source** for the override preamble. Every Phase 1–7 spawn pulls its preamble from here. Do not inline the preamble inside the phase sections — they all link back here. If the override needs to change, change it here once.

### Required preamble (paste verbatim into every spawn)

```
Paradigm override: this project does NOT use any coordination runtime (no
Tillsyn action items, no MCP dispatcher, no capability leases). Drop
coordination lives in drops/DROP_N_<NAME>/ with the file lifecycle described
in drops/WORKFLOW.md. Ignore any instructions in your agent definition that
refer to till_*, capture_state, attention_item, handoff, capability_lease, or
auth_request. Read drops/WORKFLOW.md before acting. Edit only the files your
phase owns (see WORKFLOW.md "File Lifecycle" table).

Render your response beginning with a `# Section 0 — SEMI-FORMAL REASONING`
block containing `## Planner`, `## Builder`, `## QA Proof`, `## QA
Falsification`, and `## Convergence` passes (or the 4-pass subagent variant
`## Proposal / ## QA Proof / ## QA Falsification / ## Convergence` per
~/.claude/output-styles/tillsyn-flow.md § "Section 0 — SEMI-FORMAL REASONING
(Pre-Body Block)") before your final output.
Each pass uses the 5-field certificate (Premises / Evidence / Trace or cases
/ Conclusion / Unknowns) where applicable. Convergence must declare (a) QA
Falsification found no unmitigated counterexample, (b) QA Proof confirmed
evidence completeness, (c) remaining Unknowns are routed back to the
orchestrator. If any fail, loop back before Convergence.

Section 0 reasoning stays in the orchestrator-facing response only — do NOT
write Section 0 into PLAN.md, BUILDER_WORKLOG.md, BUILDER_QA_*.md,
PLAN_QA_*.md, CLOSEOUT.md, or any other durable drop artifact.
```

### Per-role appendix (concatenated after the preamble)

Each phase below specifies the per-role appendix it adds (drop's `PLAN.md` path, target output file, droplet ID, round number, etc.). Phase sections do not repeat the preamble — they reference this section.

### When the global agents update

If `~/.claude/agents/*.md` change in a way that conflicts with the override (e.g. they start *requiring* a coordination-runtime call rather than just suggesting one), update the preamble above and bump a `Last reviewed against ~/.claude/agents/*.md:` date footnote at the bottom of this section.

---

## Phase 1 — Plan

**Goal:** turn the `PLAN.md` row into atomic droplets with paths, packages, acceptance criteria, and `blocked_by` ordering. If the drop is large, the planner decomposes into sub-drops and spawns sub-planners (cascade recursion).

1. Orch copies `drops/_TEMPLATE/` → `drops/DROP_N_<NAME>/`. Sets `PLAN.md` header `state: planning`. Commits (`docs(drop-N): scaffold drop dir from template`).
2. Orch spawns a planner subagent (`go-planning-agent` or FE equivalent) with the spawn preamble from § "Agent Spawn Contract" + the planner appendix from § "Per-Role Spawn Appendices". The planner reads `PLAN.md` (project-root), the drop's `PLAN.md`, `CLAUDE.md`, `WIKI.md`, this file, and — if they exist — `AGENT_CASCADE_DESIGN.md` for droplet sizing rules.
3. Planner decides: decompose into **droplets** directly OR decompose into **sub-drops** that will themselves be planned.
   - **Droplets**: fills `## Planner` section in the drop's `PLAN.md` with droplets (`N.1`, `N.2`, …), each with `paths`, `packages`, `acceptance`, `blocked_by`, `state: todo`.
   - **Sub-drops**: fills `## Planner` section with sub-drop container rows (`N.1`, `N.2`, …), each with a stub directory reference. Orch then loops Phase 1 for each sub-drop, spawning a sub-planner per sub-drop. Sub-planners run in parallel when their scope doesn't overlap.
4. Planner returns control. Orch commits the plan (`docs(drop-N): planner decompose into K droplets` or `... into K sub-drops`). **If the planner emitted ≤1 immediate child at this dir** (single droplet, or single sub-drop), **orch `git rm`s the template-stamped `_BLOCKERS.toml`** in the same commit — the sibling-blocker ledger is present only at dirs with >1 immediate child (see § "`_BLOCKERS.toml` — Sibling Blocker Ledger"). Multi-child dirs: orch leaves the file in place for the planner to populate (or planner has already populated it inline with the plan).
5. **Droplets sharing a package MUST have explicit `blocked_by`** between them. A package is one compile unit; parallel builders on the same package trip over each other's test runs. Plan QA (Phase 2) attacks missing blockers.
6. Move to Phase 2.

## Phase 2 — Plan QA

**Goal:** independent proof + falsification review of the planner's decomposition. Fires at every planner node (package level and above), not at droplets directly.

1. Orch spawns a QA proof agent and a QA falsification agent **in parallel** with the preamble from § "Agent Spawn Contract" + plan-QA appendix from § "Per-Role Spawn Appendices". Each reads the drop's `PLAN.md`, `CLAUDE.md`, project-root `PLAN.md`, this file. Each writes its own file:
   - QA proof → `PLAN_QA_PROOF.md` with verdict (`pass` / `fail`) + findings
   - QA falsification → `PLAN_QA_FALSIFICATION.md` with verdict + counterexamples. Required attacks include: missing `blocked_by` between droplets sharing a file or package; cycles in `blocked_by`; `_BLOCKERS.toml` / `PLAN.md` drift — every `Blocked by:` bullet in `PLAN.md` must have a matching `[[blockers]]` row and vice-versa (`PLAN.md` is truth; stale TOML must regenerate from `PLAN.md`).
2. Disjoint files, no merge race. Both subagents return verdicts to orch via the `Agent` tool result.
3. Orch commits both QA outputs (`docs(drop-N): plan qa round K`).
4. **Global L1 sweep** (cascade depth ≥ 3): when a deep planner tree closes plan-QA at its local node, the level-1 planner is re-QA'd with the full descendant tree in scope. This catches cross-subtree contradictions (e.g. two sub-planners under different L1 siblings both claiming ownership of the same file). If global sweep fails, the L1 plan is revised — downstream planners may or may not need re-running depending on whether the revision touches their scope.

## Phase 3 — Discuss + Cleanup

**Goal:** synthesize QA findings with dev, hand a planner brief back, clean working tree.

1. Orch summarizes both QA mds for dev (one short numbered list per file). Dev decides accept / reject / defer per finding.
2. Orch synthesizes accepted findings into a planner brief (in-conversation; no scratch file).
3. **Orch leaves the round-K plan-QA files in tree** (`PLAN_QA_PROOF.md` / `PLAN_QA_FALSIFICATION.md` for round 1, `PLAN_QA_PROOF_R2.md` / `PLAN_QA_FALSIFICATION_R2.md` for round 2, etc.). Never `git rm`. Round K+1's plan-QA spawn writes the next round-suffix file. Every adversarial round stays visible in the tree for audit.
4. Orch re-spawns the planner (preamble + planner appendix again, plus the brief). Planner edits `PLAN.md` (revises droplets, adjusts `blocked_by`, sharpens acceptance). Heading convention: append `## Planner — Round K` if the round count matters for postmortem; otherwise edit in place and let `git log -- drops/DROP_N_<NAME>/PLAN.md` carry the audit. Default to in-place edit; bump round headings only when reviewers explicitly request the prior version stay visible.
5. Loop back to Phase 2. Exit when both plan-QA pass without dev-rejected findings.
6. On exit: orch flips drop's `PLAN.md` header `state: building`, commits (`docs(drop-N): plan accepted, advance to building`). Move to Phase 4.

## Phase 4 — Build (per droplet)

**Goal:** implement one atomic droplet cleanly.

1. Orch picks the next eligible droplet (`state: todo` and `blocked_by` empty or all `done`).
2. Orch spawns a builder subagent against droplet `N.M` with the preamble from § "Agent Spawn Contract" + builder appendix from § "Per-Role Spawn Appendices".
3. Builder edits droplet's `state` in `PLAN.md` to `in_progress` immediately. Implements. Edits `state` to `done` at end. Appends `## Droplet N.M — Round 1` section to `BUILDER_WORKLOG.md`: files touched, build-tool targets run, design notes, `## Hylla Feedback` (or equivalent index-feedback) subsection if any miss forced a fallback.
4. Orch commits (`feat(<scope>): <droplet-summary>` per `CLAUDE.md` § "Git Commit Format"). Move to Phase 5 for this droplet.

## Phase 5 — Build QA (per droplet / per package)

**Goal:** independent proof + falsification review of the droplet's implementation. LLM QA fires at package level and above; the package's automated gate (`mage ci` or equivalent) fires once per package and covers every droplet that touched it.

1. **Automated package-level gate first.** Before LLM QA runs, the builder (last in the package) triggers the project's package gate (`mage test-pkg <pkg>` or language equivalent). Gate green is a precondition for LLM QA. Gate red → back to Phase 4 for whichever droplet owns the breakage.
2. Orch spawns QA proof + QA falsification agents **in parallel** against droplet `N.M` with the preamble from § "Agent Spawn Contract" + build-QA appendix from § "Per-Role Spawn Appendices". Each reads the drop's `PLAN.md`, `BUILDER_WORKLOG.md`, the actual changed code (index / `git diff` / Read), `CLAUDE.md`, this file.
3. Each appends `## Droplet N.M — Round K` section to its own durable md (`BUILDER_QA_PROOF.md`, `BUILDER_QA_FALSIFICATION.md`) with verdict + findings.
4. **Append-not-overwrite.** Two parallel subagents writing to different files — no merge race. If both write to the same file via append-edit and one fails because the other won the write, orch retries the loser.
5. Pass + pass → orch commits (`docs(drop-N): droplet N.M qa green`). Next droplet (back to Phase 4 with `N.M+1`).
6. Fail (either) → orch summarizes findings to dev, dev decides direction, orch respawns the builder for the same droplet, builder appends `## Droplet N.M — Round 2` to `BUILDER_WORKLOG.md`, QA appends `## Droplet N.M — Round 2` to its files. Loop until both pass.
7. **Ancestor re-QA on blocker failure.** If a droplet's failure is rooted in a planner-above's decision (e.g. wrong package boundaries, missing `blocked_by`, unachievable acceptance), escalate to the planner level — the planner-above is re-QA'd and may need to revise. This prevents local fixes that mask planner-level flaws.

## Phase 6 — Verify

**Goal:** machine-checkable confirmation the drop's surface area still builds, tests, and lints clean at drop scope.

**Per-droplet verification** (during Phase 5, before declaring a droplet pass): the package-level gate runs. QA mds note the targets run + result.

**Drop-end verification** (after all droplets have passed Phase 5):
1. Orch (or builder, by spawn) runs the project's full CI target (`mage ci`, `just ci`, `npm run ci`, etc.) from the primary worktree. Must pass clean.
2. `git push` once the CI target is green.
3. `gh run watch --exit-status` until remote CI green.
4. If any step fails, treat as build-QA fail on whichever droplet owns the breakage — back to Phase 5 for that droplet.

## Phase 7 — Closeout

**Goal:** durable record of the drop, propagate findings, advance `PLAN.md`.

1. Orch (or `Role: commit` builder, spawned for this) writes `CLOSEOUT.md`:
   - Aggregate `## Hylla Feedback` subsections (or equivalent index-feedback) from `BUILDER_WORKLOG.md` → append entry to `HYLLA_FEEDBACK.md` at project root.
   - Aggregate usage findings → append entry to `REFINEMENTS.md` (or an index-specific refinements md if the finding is index-specific).
   - Append entry to `LEDGER.md`.
   - Append one-liner to `WIKI_CHANGELOG.md`.
   - Trigger code-understanding index reingest (full enrichment, from remote, **only after CI green**). Record result. Go projects: `hylla_ingest` with `enrichment_mode=full_enrichment`. Non-Go: substitute your index's reingest command or skip if none.
   - If anything in the drop changed best practice: update relevant section(s) of `WIKI.md` **in place** (no `2026-XX-XX update:` notes — git history is the audit).
2. Flip drop's `PLAN.md` header `state: done`. Flip the drop's row in project-root `PLAN.md` to `state: done`. Commit both in one commit (`docs(drop-N): closeout, advance plan`).
3. Move to next drop (back to Phase 1 for `DROP_N+1`).

---

## Per-Role Spawn Appendices

The preamble lives in § "Agent Spawn Contract" above. Each role appends the fields below after the preamble:

- **Planner**: drop's `PLAN.md` path, the `PLAN.md` container row excerpt, scope sentence from dev, round number if Phase 3 re-spawn, sub-planner depth (L1, L2, L3, …) if spawned by a parent planner.
- **Plan QA (proof / falsification)**: drop's `PLAN.md` path, target output path (`PLAN_QA_PROOF.md` or `PLAN_QA_FALSIFICATION.md`), round number, global-sweep flag if Phase 2 step 4 re-QA.
- **Builder**: droplet ID (e.g. `1.3`), droplet row excerpt from drop's `PLAN.md`, drop's `BUILDER_WORKLOG.md` path, round number, working dir.
- **Build QA (proof / falsification)**: droplet ID, drop's `PLAN.md` + `BUILDER_WORKLOG.md` paths, target append file (`BUILDER_QA_PROOF.md` or `BUILDER_QA_FALSIFICATION.md`), round number.

## Recovery After Restart

No coordination-runtime calls. Recovery is filesystem + git:

1. `git status` — uncommitted work.
2. `git log --oneline -20` — recent commits.
3. Read project-root `PLAN.md` — container states (which drop is `in_progress`).
4. List `drops/*/PLAN.md` headers — per-drop phase state (`planning` / `building` / `done` / `blocked`).
5. Per active drop: presence of `PLAN_QA_*.md` files = mid-plan-QA loop (Phase 2 or 3); absence + `BUILDER_WORKLOG.md` exists = mid-build (Phase 4 or 5); `CLOSEOUT.md` exists with `state: done` = drop closed.
6. Per active droplet: scan latest `## Droplet N.M — Round K` heading in `BUILDER_WORKLOG.md` + both `BUILDER_QA_*.md` to figure out whether build, build-QA, or fix is next.
7. Per active sub-drop tree: recurse — read the nested drop dir, same rules.

## File State Diagrams

### Drop's `PLAN.md` header `state`

```
planning ──(plan-QA both pass)──▶ building ──(all droplets done + CI green)──▶ done
   │                                  │
   │                                  └──(blocker discovered)──▶ blocked
   │
   └──(blocker discovered)──▶ blocked
```

`blocked` is orthogonal — entered from any state when a discovered blocker stops forward progress; exited back to whichever state was active.

### Per-droplet `state` (inside `PLAN.md` Planner section)

```
todo ──(builder claims)──▶ in_progress ──(package gate + both QA pass)──▶ done
                              │
                              └──(blocker)──▶ blocked
```

## Notes on Heading Conventions

- Plan-QA fresh files per round → no round suffix needed in headings.
- Builder + build-QA append rounds → `## Droplet N.M — Round K`.
- Planner re-runs default to in-place edit; add `## Planner — Round K` only when reviewers explicitly want the prior version preserved in the file (rare — git log usually suffices).
- Sub-drop Planner sections use dotted depth (e.g. `## Droplet 1.2.3 — Round 1` inside a sub-drop's `BUILDER_WORKLOG.md`).
