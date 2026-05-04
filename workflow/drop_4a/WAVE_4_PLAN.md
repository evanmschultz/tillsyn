# DROP_4A — WAVE 4 — CLOSEOUT (DOC SWEEP FOR DISPATCHER + DOMAIN FIELDS + AUTH-FLOW LANDING)

**State:** planning
**Wave position:** Wave 4 (lands LAST in Drop 4a sequence; depends on Wave 1, Wave 2, Wave 3 having all landed so docs reflect actual code state).
**Paths (expected, in-repo):** `main/CLAUDE.md`, `main/WIKI.md`, `main/STEWARD_ORCH_PROMPT.md`
**Paths (expected, outside-repo, audit-gap accept):** `~/.claude/agents/go-builder-agent.md`, `~/.claude/agents/go-planning-agent.md`, `~/.claude/agents/go-qa-proof-agent.md`, `~/.claude/agents/go-qa-falsification-agent.md`, `~/.claude/CLAUDE.md`, `~/.claude/projects/-Users-evanschultz-Documents-Code-hylla-tillsyn/memory/feedback_steward_spawn_drop_orch_flow.md`, `~/.claude/projects/-Users-evanschultz-Documents-Code-hylla-tillsyn/memory/project_steward_auth_bootstrap.md`
**Packages:** — (docs-only wave; zero Go code)
**REVISION_BRIEF ref:** `workflow/drop_4a/REVISION_BRIEF.md` § "Wave 4 — Closeout (~2 droplets)"
**Started:** 2026-05-03
**Closed:** —

## Wave Purpose

Wave 4 sweeps the canonical docs (`main/CLAUDE.md`, `main/WIKI.md`, `main/STEWARD_ORCH_PROMPT.md`) plus the outside-repo agent files / global CLAUDE.md / memory files so they reflect the post-Drop-4a runtime state:

1. **Dispatcher landing** — pre-cascade language ("orchestrator IS the dispatcher", "pre-cascade orchestrator manually spawns") survives only as historical context; current canonical pointers to dispatcher package + manual-trigger CLI + `mage` invocation. Pre-cascade pre-dogfood content stays where the pre-dogfood window genuinely persists; dispatcher-aware vocabulary lands wherever the new code path is the canonical answer.
2. **Domain-field landing** — `paths` / `packages` / `files` / `start_commit` / `end_commit` are first-class on `ActionItem` (Wave 1); project-node fields (`hylla_artifact_ref`, `repo_bare_root`, `repo_primary_worktree`, `language`, `build_tool`, `dev_mcp_server_name`) are first-class on `Project` (Wave 1). Any doc text saying "metadata.paths" or "in metadata" or "stored on metadata" for these fields is corrected to first-class.
3. **Auth-flow landing** — orch-self-approves-non-orch-subagent-in-subtree is canonical (Wave 3). The `S2 fallback (if approve is rejected today)` paragraph in `STEWARD_ORCH_PROMPT.md` §8.1 retires; the corresponding paragraphs in the four go-* agent auth sections retire; the `Pre-fix vs post-fix state` paragraph in `WIKI.md § "Auth Approval Cascade"` (line 242) retires; memory file caveats retire.
4. **Always-on parent-blocks-on-failed-child** — Wave 1 made the invariant unconditional. Any doc saying "Always-on parent-blocks-on-failed-child arrives in Drop 1" or "becomes always-on in Drop 1" is corrected to past tense.
5. **Drop 4b kickoff** — every doc that names "Drop 4" without disambiguation is reviewed; references to gate execution / commit pipeline / push / Hylla reingest automation point at Drop 4b explicitly.

This wave is **docs-only**. Zero Go code. Zero test changes. The wave's correctness gate is `mage ci` green (smoke check; no Go touched) plus self-QA + dev-approval gate before each commit, mirroring drop_3/PLAN.md droplets 3.27 + 3.28.

## Wave Decomposition

Four droplets. Three in-repo (one file per droplet — same-file-lock rule from `CLAUDE.md § "File- and package-level blocking"`). One outside-repo bundle (audit-gap acceptance per drop_3/PLAN.md droplet 3.27 finding 5.D.10). All four are blocked transitively on Wave 1 / Wave 2 / Wave 3 closure — docs cannot lock in until the code state they describe lands.

| ID    | Title                                                | In-repo? | Blocked by                                    |
| ----- | ---------------------------------------------------- | -------- | --------------------------------------------- |
| W4.1  | `MAIN/CLAUDE.MD` DISPATCHER-AWARE SWEEP              | yes      | Wave 1 close, Wave 2 close, Wave 3 close      |
| W4.2  | `MAIN/WIKI.MD` POST-DISPATCHER COHERENCE CHECK       | yes      | Wave 1 close, Wave 2 close, Wave 3 close      |
| W4.3  | `MAIN/STEWARD_ORCH_PROMPT.MD` §8.1 S2 FALLBACK SWEEP | yes      | Wave 3 close                                  |
| W4.4  | OUTSIDE-REPO AGENT + GLOBAL CLAUDE + MEMORY SWEEP    | no       | Wave 3 close, W4.1 (cross-doc consistency)    |

Total: 4 droplets.

**Sequencing rationale:** W4.1 / W4.2 / W4.3 are file-disjoint and could run in parallel from a same-file-lock perspective, but the orchestrator (per `feedback_md_update_qa.md`) does each MD edit inline as a single self-QA + dev-approval cycle. W4.4 explicitly blocks on W4.1 so the outside-repo agent files can quote / mirror any pointer-style updates W4.1 introduces in `main/CLAUDE.md` (preventing drift between in-repo canonical and outside-repo subagent prompts).

**Decomposition discipline:** each droplet does one canonical doc (or one outside-repo bundle) end-to-end — no half-sweep, no second-pass cleanup. Splitting any droplet further produces a half-swept doc with mixed pre-Drop-4a / post-Drop-4a vocabulary, which is exactly the failure mode drop_3/PLAN.md droplet 3.27 documents (`Irreducible: true` rationale). Marking W4.1 / W4.2 / W4.3 / W4.4 each `Irreducible: true` for the same reason.

## Droplet Decomposition

### W4.1 — `MAIN/CLAUDE.MD` DISPATCHER-AWARE SWEEP

- **State:** todo
- **Paths:** `main/CLAUDE.md`
- **Packages:** —
- **Acceptance:**
  - **In-scope sections** (every section listed below is reviewed; only sections whose content drifted from post-Drop-4a runtime get edited; pure pre-cascade history paragraphs survive untouched):
    - `## Cascade Plan` (line 25-27) — verify language. Currently says *"That plan is the source of truth for cascade architecture, drop ordering, and hard prerequisites. This `CLAUDE.md` documents the **current pre-cascade workflow** the orchestrator uses today."* After Drop 4a, the manual-trigger dispatcher landed; the "current pre-cascade workflow" text needs to acknowledge the partial dispatcher (Drop 4a delivered manual-trigger spawn + lock management + auto-promotion + auth provisioning; Drop 4b will deliver gate execution + commit pipeline). Append a single sentence: *"Drop 4a landed the manual-trigger dispatcher (`internal/app/dispatcher/`); orchestrator-IS-dispatcher remains the documented pre-cascade fallback while dogfooding ramps. Drop 4b adds gate execution + post-build pipeline."*
    - `## Cascade Tree Structure` § "State-Trigger Dispatch" (line 137-139) — currently says *"Pre-cascade, the orchestrator IS the dispatcher — it reads the kind, picks the binding above, moves the item to `in_progress`, and spawns the subagent via the `Agent` tool with Tillsyn auth credentials and Hylla artifact ref in the prompt."* After Drop 4a this paragraph keeps the pre-cascade sentence (still load-bearing during dogfood) but adds a sibling sentence: *"Drop 4a delivered the manual-trigger dispatcher: `till dispatcher run --action-item <id>` (`cmd/till`) reads the same template `agent_bindings`, acquires file/package locks via the lock manager, spawns the subagent via `claude --agent <name>`, and provisions auth via Wave-3's orch-self-approval flow. Per-drop work currently dogfoods both paths until Drop 4b's gate runner ships."*
    - `## Cascade Tree Structure` § "Required Children (Auto-Create Rules)" (line 90-99) — verify still accurate. Manual-creation today is correct. **No edit unless** drop_3/PLAN.md template `child_rules` enforcement landed in a way that displaces "Manual today" — verify against `internal/templates/builtin/default.toml` post-Drop-3 actual state. Likely no edit needed; record verification in the worklog.
    - `## Cascade Tree Structure` § "Agent Bindings" (line 101-117) — verify still accurate post-dispatcher. The table headers + agent IDs are stable. The wrapper paragraph at line 102 (*"Pre-cascade: orchestrator spawns these manually via the `Agent` tool using Tillsyn auth credentials in the prompt. Post-Drop-3: the template binds kinds → agents; the dispatcher spawns them on `in_progress` transitions."*) is correct as a Drop-3 description; append (or substitute) a sentence: *"Post-Drop-4a: Wave 2 delivered the dispatcher loop; manual-trigger today, automatic on `in_progress` post-Drop-4b."*
    - `## Cascade Tree Structure` § "Post-Build Gates" (line 119-129) — currently says *"Pre-cascade: orchestrator + dev do this manually (see Git Management (Pre-Cascade) below)."* Wave 0 added the `.githooks/pre-commit` + `.githooks/pre-push` gates locally; Wave 2's dispatcher does NOT yet run gates (deferred to Drop 4b per locked decision L7). Update gate-1 ("`mage ci`") to add: *"Wave 0 of Drop 4a wired `mage ci` into `.githooks/pre-push`, so a clean push is itself the smoke check."* Leave gates 2-4 ("Commit", "Push", "Hylla reingest") with their pre-cascade wording — Drop 4b is the consumer.
    - `## Action-Item Lifecycle (Current HEAD)` (line 261-272) — currently says *"A fourth state `failed` lands in Drop 1 of the cascade plan."* Drop 1 already landed. Edit the second sentence to past tense: *"`failed` is a real terminal state post-Drop-1."* Drop the *"Until then"* parenthetical and the *"Currently the action item stays in `in_progress` with a failure-flavored outcome; Drop 1 adds the real `failed` transition."* clause. Verify against actual code state via `LSP` workspace symbol search for `StateFailed` / `StatusFailed` enum values before rewriting (description-symbol verification rule). Also: line 271 *"No parent can move to terminal-success if any child is in a failure/blocked state — enforcement becomes always-on in Drop 1."* — Wave 1 of Drop 4a removed `RequireChildrenComplete` policy bit per locked decision L2; rewrite as past-tense canonical: *"No parent can move to terminal-success if any child is in `failed` or `blocked` state — always-on invariant (Wave 1 of Drop 4a removed the `RequireChildrenComplete` policy bit; the rule is unconditional)."*
    - `## Paths and Packages (Drop-1 Target)` (line 274-276) — section heading itself is now stale. Wave 1 landed `paths` / `packages` first-class. Rewrite section heading as `## Paths and Packages` (drop the parenthetical), and rewrite body: *"Wave 1 of Drop 4a landed `paths []string` and `packages []string` as first-class fields on every action item (`internal/domain/action_item.go`). Planners set them at creation; dispatcher's lock manager (Wave 2) reads `packages` for package-level locks and `paths` for file-level locks. Builders restrict edits to declared `paths`. Per-package compile collisions are blocked at `in_progress` promotion via runtime `blocked_by` insertion when a sibling holds the same package lock."* Also expand to mention `files`, `start_commit`, `end_commit` per Wave 1 deliverables list — sibling first-class fields with their domain semantics. Cross-reference `WIKI.md § "Atomic Drop Granularity"` (which already references `paths` / `packages` first-class).
    - `## Auth and Leases` (line 278-283) — currently says *"Auth auto-revoke on terminal state is a Drop-1 item; until then, the orchestrator manually revokes stale sessions."* Drop 1 landed. Verify auto-revoke is in fact wired (Wave-3 of Drop 4a does NOT include this per REVISION_BRIEF "Out-of-scope" — auth auto-revoke on terminal state is **Drop 4b** scope). So edit the bullet to: *"Auth auto-revoke on terminal state lands in Drop 4b. Pre-Drop-4b, orchestrators (and STEWARD post-merge) manually revoke stale sessions via `till.auth_request operation=revoke`."* Add a new bullet immediately after about Wave-3's orch-self-approval landing: *"Orchestrators approve their own non-orch subagent auth requests scoped within their lease subtree (Wave 3 of Drop 4a). Cross-orch and orch-spawning-orch approvals still route through the dev TUI."*
    - `## Coordination Surfaces` § "Subagents" + "Orchestrator (this session)" (line 285-298) — verify still accurate. Wave-3 auth flow does not change which subagents may call which Tillsyn surfaces. **No edit needed** unless review surfaces drift; record verification in worklog.
    - `## Build-QA-Commit Discipline` (line 166-183) — verify still accurate. Wave 0 hooks (`mage format-check`, `mage ci` via `pre-push`) are local-CI gates; the Phase-4-5 / Phase-6-7 split is unchanged by Drop 4a. **No edit unless** drift surfaces from a hooks-paragraph reference. Likely a single one-line addition under "Per-droplet (Phases 4-5)" pointing at `mage install-hooks` for fresh clones: *"(Note: dev runs `mage install-hooks` once per fresh clone to activate the Wave-0 git hooks; the hooks invoke `mage format-check` pre-commit and `mage ci` pre-push.)"*
    - `## Orchestrator-as-Hub Architecture` § "How It Works" + "Agent State Management" (line 237-260) — verify the agent-state contract still holds. Wave 3 added orch-self-approval; the spawn-prompt-vs-description split documented at line 247-258 stays canonical. **No edit unless** drift surfaces.
    - `## Drop Closeout` + `## Git Management (Pre-Cascade)` + `## Post-Merge Branch Cleanup` (line 200-227) — verify Drop 4a did not change the drop-close protocol. Drop 4a is itself a single drop with a single PR + a single Hylla reingest (locked decision L8); per-drop close mechanics unchanged. **No edit unless** drift surfaces.
  - **Project ID + Hylla Baseline sections** (`## Tillsyn Project`, `## Hylla Baseline` lines 141-156) — likely untouched. Verify both still accurate (project ID `a5e87c34-3456-4663-9f32-df1b46929e30` stable; artifact ref stable).
  - **Out-of-scope explicit:** no rewrites of pre-cascade history paragraphs that are accurate as history (e.g. the `## Cascade Plan` sentence "That plan is the source of truth for cascade architecture..."). No retitling of sections. No restructuring. Surgical edits only.
  - **Verification:**
    - `mage ci` green (no Go code touched; CI exit status is smoke test — same as drop_3/3.27).
    - `Grep "Drop 1 of the cascade plan"` and `"arrives in Drop 1"` and `"becomes always-on in Drop 1"` against `main/CLAUDE.md` returns zero hits after edit.
    - `Grep "metadata\.paths"` and `"in metadata.*paths"` against `main/CLAUDE.md` returns zero hits after edit.
    - `Grep "auto-revoke on terminal state is a Drop-1 item"` against `main/CLAUDE.md` returns zero hits after edit.
    - Builder records per-section diff inventory in `main/workflow/drop_4a/BUILDER_WORKLOG.md`: `<section> <hits before> <rewrites> <left-as-is>`.
    - **Per `feedback_md_update_qa.md`:** orchestrator (or builder subagent for systematic sweep) presents inventory + sample edits → dev approves → commit.
- **Blocked by:** Wave 1 final droplet (paths/packages/files/start_commit/end_commit + Project node fields + always-on parent-blocks landed); Wave 2 final droplet (dispatcher package + manual-trigger CLI landed); Wave 3 final droplet (orch-self-approval landed). Concretely: orch synthesizes Wave 1 → Wave 4 mapping at PLAN.md unification time and writes the explicit globally-numbered `4a.X` blockers here.
- **Irreducible:** `true`. Rationale: a single coherent doc sweep across `main/CLAUDE.md` — splitting risks half-swept vocabulary (e.g. "Drop 1 lands `failed`" in one paragraph and "post-Drop-1 `failed` is a real terminal state" in another). One sweep, one self-QA pass, one dev-approval gate. Methodology §2.3's "single coordinated vocabulary sweep across active canonical docs" generalization applies.
- **Notes:**
  - **Orchestrator-driven OR builder-driven:** per `feedback_md_update_qa.md`, in-orch quick MD updates are orch's surface; systematic sweeps spawn a builder subagent. **This droplet is borderline — ~10 surgical edits in one file.** Recommendation: orch-driven inline (no subagent QA twins), with the orchestrator's own self-QA + dev-approval gate as the close. If during execution the edit set grows past ~15 hits, escalate to a builder subagent (mirroring drop_3/3.27 for systematic sweeps). Default: orch-driven.
  - **Pre-cascade dogfood window:** every paragraph that says "pre-cascade" and references the orchestrator manually doing dispatcher-like work survives — not because it's stale, but because Drop 4a delivers the *manual-trigger* dispatcher (locked decision L7), not the automatic one. The dogfood window during which both paths exist is itself canonical. The droplet's job is to add the post-Drop-4a sentence, not delete the pre-cascade sentence.
  - **Description-symbol verification:** before writing post-Drop-4a sentences that name code symbols (`internal/app/dispatcher/`, `till dispatcher run`, `RequireChildrenComplete`, `StateFailed`), the orch (or builder) verifies the symbols exist via `LSP` workspace symbol search. Drift between description and reality goes in the worklog Unknowns.

### W4.2 — `MAIN/WIKI.MD` POST-DISPATCHER COHERENCE CHECK

- **State:** todo
- **Paths:** `main/WIKI.md`
- **Packages:** —
- **Acceptance:**
  - **In-scope sections** (review every; edit only on drift):
    - `## The Tillsyn Model (Node Types)` (line 19-31) — verify still accurate. References `metadata.role` and `metadata.structural_type` as orthogonal axes; `paths` and `packages` are first-class but not mentioned in this section. **No edit unless** drift surfaces.
    - `## Cascade Vocabulary` (line 36-83) — canonical glossary owned by drop_3/3.6. Pointer-only edits; do NOT redefine any structural_type semantics here. Verify subsection `### Adjacent Domain Primitives` (line 72-79) still uses `metadata.persistent` / `metadata.dev_gated`. Wave 1 of Drop 4a does NOT touch these — they stayed on metadata per Drop 3's design. **No edit unless** drift surfaces.
    - `## Drop Decomposition Rules` § "Atomic Drop Granularity" (line 122-130) — currently bullet 3 says *"It has a clear `paths` / `packages` footprint so file- and package-level blocking can work."* This is correct first-class language and Wave 1 confirms it. **No edit needed** — verify only.
    - `## Drop Decomposition Rules` § "Level-1 Drop Sizing + Parallelism" (line 132-142) — references `paths` / `packages` consistently. **No edit needed.**
    - `## Drop Decomposition Rules` § "Ordering: Use `blocked_by`, Not `depends_on`" (line 144-153) — references "the dispatcher adds runtime blockers when file/package locks conflict (Drop 4+)." After Wave 2, the dispatcher exists. Edit to past tense for the dispatcher half: *"Wave 2 of Drop 4a delivered the dispatcher's lock manager; runtime `blocked_by` insertion fires on `in_progress` promotion when sibling locks conflict."* Keep the load-bearing rule of thumb intact.
    - `## Build-QA-Commit Loop (Pre-Cascade)` (line 168-182) — currently says *"Until the cascade dispatcher ships (Drop 4+), the parent orchestrator session runs this loop manually..."* After Drop 4a's *manual-trigger* dispatcher (locked decision L7), this loop runs through either the orchestrator (legacy) or `till dispatcher run --action-item <id>` (new). Edit the lead sentence to: *"Until the gate runner ships in Drop 4b, the parent orchestrator session OR Drop-4a's manual-trigger dispatcher runs this loop. Loop body unchanged; the dispatcher merely automates the spawn + lock + auth-provision steps."* Loop body steps 1-7 stay intact — the steps themselves are unchanged.
    - `## Auth Approval Cascade` (line 230-246) — **canonical landing for the post-Wave-3 retire**. Currently line 242 says *"Pre-fix vs post-fix state. The capability to approve subagent auth from an orch session lands in the auth-approval-cascade drop (PLAN §19.1.6), scheduled between Drop 1.5 and Drop 2. Until that drop ships, orch-side approval may fail with a permission error..."* After Wave 3 of Drop 4a, this paragraph retires. Replace with a single past-tense sentence: *"Wave 3 of Drop 4a landed the orch-self-approves-non-orch-subagent capability. Orch-side approval is the canonical path; cross-orch and orch-spawning-orch still route through the dev TUI."* Drop the entire pre-fix-vs-post-fix paragraph. Verify `Approval scope` numbered list (line 236-240) stays intact — those rules are post-fix canonical.
    - `## Drop Orch Cross-Subtree Exception` (line 248-260) — verify still accurate. References STEWARD's six persistent level_1 parents; this is unchanged by Drop 4a. **No edit needed.**
    - `## Response Shape — Section 0 Semi-Formal Reasoning` (line 261-290) — canonical reference to `SEMI-FORMAL-REASONING.md`; unchanged by Drop 4a. **No edit needed.**
    - `## Drop-End Closeout Checklist` (line 292-317) — references `hylla_ingest`, drop-orch flow, STEWARD post-merge. Wave 1's project-node first-class fields (`hylla_artifact_ref`, etc.) MAY warrant a one-line addition: *"Project-node fields (`hylla_artifact_ref`, `repo_bare_root`, `repo_primary_worktree`, `language`, `build_tool`, `dev_mcp_server_name`) are first-class on `Project` post-Wave-1. Drop-orchs read them via `till.project(operation=get)` rather than `metadata.hylla_artifact_ref`."* Insert under existing `hylla_ingest` step (line 303). Otherwise unchanged.
    - `## Related Files` (line 319-332) — verify all listed files still exist. **No edit unless** Drop 4a deleted any. (Drop 4a is additive — no deletes.)
  - **`metadata.paths` audit (REVISION_BRIEF deliverable 2):** `Grep "metadata\.paths"`, `"metadata\.packages"`, `"metadata\.files"` across `main/WIKI.md`. Each hit gets reviewed; if it references the field as a metadata blob entry, rewrite to first-class. **Spawn-prompt expectation:** zero hits expected (WIKI already uses first-class language per line 128, 138). If zero hits found, droplet's edit set is the §"Auth Approval Cascade" retire (line 242) plus the `Build-QA-Commit Loop` lead-sentence rewrite + the §"Atomic Drop Granularity" verification only.
  - **Verification:**
    - `mage ci` green.
    - `Grep "PLAN §19\.1\.6"` against `main/WIKI.md` returns zero hits after edit.
    - `Grep "Pre-fix vs post-fix"` against `main/WIKI.md` returns zero hits after edit.
    - `Grep "metadata\.paths\|metadata\.packages\|metadata\.files"` returns zero hits after edit.
    - `Grep "Until the cascade dispatcher ships"` against `main/WIKI.md` returns zero hits after edit (replaced by "Until the gate runner ships in Drop 4b").
    - Builder records per-section diff inventory in `BUILDER_WORKLOG.md`.
    - Self-QA + dev-approval gate before commit.
- **Blocked by:** Wave 1 final droplet, Wave 2 final droplet, Wave 3 final droplet. (Same as W4.1.)
- **Irreducible:** `true`. Rationale: same-doc coherence — a partial sweep leaves WIKI sections with mixed pre-Wave-3 and post-Wave-3 auth language, which directly violates the "Single-Canonical-Source Rule" the WIKI itself documents at line 81-83.
- **Notes:**
  - **Orchestrator-driven** by default (small in-scope edit set; pointer-style adjustments). If the §"Auth Approval Cascade" retire surfaces unexpected dependencies (e.g. a cross-reference from another section), escalate to builder subagent.
  - **Coordination with W4.1:** both touch the same conceptual surface (auth-flow + dispatcher landing) but file-disjoint. Run W4.2 *after* W4.1 commits so the cross-doc consistency check (W4.4 audits this) sees the canonical updates already in `main/CLAUDE.md`. Strict ordering enforced via the same Wave-3-blocked-by + transitive dependency on Wave 1 + Wave 2.
  - **Description-symbol verification:** before writing the post-Wave-3 auth sentence, the orch verifies the orch-self-approve handler exists in code via `LSP` workspace symbol search for the relevant App-service method (likely `ApproveAuthRequest` or similar in `internal/app`).

### W4.3 — `MAIN/STEWARD_ORCH_PROMPT.MD` §8.1 S2 FALLBACK SWEEP

- **State:** todo
- **Paths:** `main/STEWARD_ORCH_PROMPT.md`
- **Packages:** —
- **Acceptance:**
  - **In-scope edits:**
    - **§8.1 line 302 — `S2 fallback (if approve is rejected today)`** retires. Currently the paragraph says: *"the orch-approves-subagent capability lands in §19.1.6 fix drop — pre-fix, the system may still gate subagent approval to dev. If the approve call returns a guardrail error, surface to the dev in chat with the request_id; dev approves in TUI; capture the approval and continue. Note the friction in `DROP_N_REFINEMENTS_RAISED` for that cycle so it feeds the §19.1.6 design."* After Wave 3 of Drop 4a, this paragraph is replaced with a single sentence: *"S2 succeeds in Drop-4a-and-later sessions: Wave 3 of Drop 4a landed the orch-self-approves-non-orch-subagent capability. If S2 fails (e.g. STEWARD's lease subtree mismatch), surface the request_id to the dev in chat for manual TUI approval and file a `kind=refinement` node under REFINEMENTS — but the path is by-exception, not by-default."*
    - **§8.1 lead paragraph line 261** — currently says *"Canonical flow (current rule, pre-§19.1.6 fix drop): the dev approves orchestrator auth only. STEWARD provisions AND approves auth for every non-orch subagent it spawns..."* After Wave 3 the parenthetical "pre-§19.1.6 fix drop" retires. Replace with: *"Canonical flow (post-Wave-3 of Drop 4a): the dev approves orchestrator auth only. STEWARD provisions AND approves auth for every non-orch subagent it spawns..."* Body unchanged.
    - **Cross-references to §19.1.6** anywhere in the file are reviewed; if Wave 3 absorbed the §19.1.6 work entirely (per REVISION_BRIEF locked decision L5), references that point to §19.1.6 as a future drop are updated to point to "Wave 3 of Drop 4a" or removed if redundant. `Grep "§19\.1\.6\|19\.1\.6"` against `main/STEWARD_ORCH_PROMPT.md` produces a complete hit list. Each hit reviewed; rewrite or delete as appropriate.
  - **Out of scope:**
    - `§8 Auth Bootstrap` lead paragraph (line 240-258) — STEWARD's own claim flow is unchanged by Drop 4a. **No edit.**
    - `§10 Drop-Close Sequence` — drop-close mechanics unchanged. **No edit unless** drift.
    - All other sections — verification only; no edits unless drift surfaces.
  - **Verification:**
    - `mage ci` green.
    - `Grep "S2 fallback (if approve is rejected today)"` against `main/STEWARD_ORCH_PROMPT.md` returns zero hits.
    - `Grep "pre-§19\.1\.6 fix drop"` returns zero hits.
    - `Grep "§19\.1\.6"` and `"19\.1\.6"` returns either zero hits or only past-tense references (e.g. "Wave 3 of Drop 4a absorbed §19.1.6"). No future-tense references survive.
    - Builder records diff in `BUILDER_WORKLOG.md`.
    - Self-QA + dev-approval before commit.
- **Blocked by:** Wave 3 final droplet (orch-self-approval landed). Does NOT block on Wave 1 / Wave 2 — STEWARD prompt is auth-flow scoped; W4.1 already covers dispatcher + domain-field updates that touch other sections.
- **Irreducible:** `true`. Rationale: §8.1 is a tightly-coupled paragraph block; splitting the lead paragraph from the S2-fallback paragraph leaves mixed pre/post-Wave-3 language inside a single subsection. One sweep.
- **Notes:**
  - **Orchestrator-driven** — small edit set (~3-5 hits, all in §8.1 + cross-references). Inline edit + dev-approval gate. No subagent.
  - **Coordination with W4.1 + W4.2:** W4.3's edits are auth-only and file-disjoint from W4.1 + W4.2; can run after Wave 3 lands without waiting for W4.1 / W4.2 to commit. Lower in droplet ordering only because the orch will likely batch all in-repo MD edits within one work session per `feedback_md_update_qa.md`'s self-QA discipline.
  - **Memory-rule alignment:** `feedback_steward_spawn_drop_orch_flow.md` documents the same S2 dev-fallback that retires here. W4.4 retires the memory file caveat alongside this droplet's STEWARD prompt edit, ensuring memory + canonical stay in lockstep.

### W4.4 — OUTSIDE-REPO AGENT + GLOBAL CLAUDE + MEMORY SWEEP

- **State:** todo
- **Paths (NOT git-tracked; audit-gap accept per drop_3/3.27 finding 5.D.10):**
  - `~/.claude/agents/go-builder-agent.md`
  - `~/.claude/agents/go-planning-agent.md`
  - `~/.claude/agents/go-qa-proof-agent.md`
  - `~/.claude/agents/go-qa-falsification-agent.md`
  - `~/.claude/CLAUDE.md`
  - `~/.claude/projects/-Users-evanschultz-Documents-Code-hylla-tillsyn/memory/feedback_steward_spawn_drop_orch_flow.md`
  - `~/.claude/projects/-Users-evanschultz-Documents-Code-hylla-tillsyn/memory/project_steward_auth_bootstrap.md`
- **Packages:** —
- **Acceptance:**
  - **Audit-gap acceptance** (verbatim from drop_3/3.27 finding 5.D.10): `~/.claude/` edits are recorded in `BUILDER_WORKLOG.md` only. The workflow trades off git-tracked permanence for adopter-skill universality. Future maintainers reading the worklog see what edits landed; reconstruction against future filesystem state requires manual diff. Document this audit-gap acceptance in the droplet description explicitly.
  - **Per agent file (4 files, ~/.claude/agents/go-*.md):**
    - **Auth claim section** — search for any reference to "S2 dev-fallback", "pre-§19.1.6 fix drop", "if approve is rejected today", "dev approves in TUI" inside the auth section. If present, retire the paragraph: replace with "Wave 3 of Drop 4a landed orch-self-approval as canonical. If your spawn prompt fails to provide a `request_id` + `resume_token` (orch-side approval failed), surface to the orchestrator and stop — do NOT proceed to dev-TUI fallback within the agent's own flow." If the file has no such paragraph (verify per agent — the current `go-builder-agent.md` does not appear to carry one; `go-planning-agent.md` likely does given REVISION_BRIEF lists all four), record "no S2-fallback paragraph found, no edit" in the worklog.
    - **`task_id` references** — REVISION_BRIEF Wave 4 deliverable 2 mentions "update `task_id` references where appropriate." `Grep "task_id"` per file. Wave 1 of Drop 4a does NOT rename `task_id` to anything (action-item ID is still `task_id` on the prompt contract per current convention). **No `task_id` rename needed.** Record "no task_id rename per Wave 1 scope" in worklog. (REVISION_BRIEF brief language was anticipatory; Wave-1's actual scope per its plan does not include this rename.)
    - **Cascade Binding section** — verify the kind table (`plan` / `build` / `plan-qa-proof` etc.) is unchanged by Drop 4a. **No edit unless** drift.
    - **Required Prompt Fields section** — verify still accurate. Wave-1's domain-field landing does not change the spawn-prompt contract; the prompt still carries `paths` / `packages` as part of the action-item description (durable), not the prompt (ephemeral). **No edit unless** drift.
  - **`~/.claude/CLAUDE.md` parity sync:**
    - `Grep "S2\|dev-fallback\|§19\.1\.6"` against the file. Per `feedback_bare_root_not_tracked.md`, the file is not git-tracked; edits go into `BUILDER_WORKLOG.md` only. If hits surface that mirror retired language, retire them with the same post-Wave-3 canonical sentence.
    - Verify the global file's auth-flow language matches `main/CLAUDE.md`'s post-W4.1 language (the global file is the canonical for cross-project rules; the project file mirrors). If divergence between project + global is found, route via discussion, not silent edit.
  - **Memory file edits:**
    - `~/.claude/projects/-Users-evanschultz-Documents-Code-hylla-tillsyn/memory/feedback_steward_spawn_drop_orch_flow.md` — drop the "S2 dev-fallback pre-fix" caveat in the bullet line. Replace with a single past-tense sentence: "Wave 3 of Drop 4a landed orch-self-approval; S2 fallback retired."
    - `~/.claude/projects/-Users-evanschultz-Documents-Code-hylla-tillsyn/memory/project_steward_auth_bootstrap.md` — same caveat retire. Wherever "S2 dev-fallback" or "(pre-fix)" appears, retire to past-tense.
  - **Verification:**
    - Per-file diff inventory in `BUILDER_WORKLOG.md`: `<path> <hits before> <rewrites> <left-as-is>`.
    - `mage ci` green (no Go code touched).
    - `Grep "S2 fallback (if approve is rejected today)"` across all 7 outside-repo files returns zero hits.
    - `Grep "pre-§19\.1\.6 fix drop"` across all 7 outside-repo files returns zero hits.
    - **Cross-doc consistency check:** the post-W4.1 + post-W4.2 + post-W4.3 in-repo canonical language for orch-self-approval matches the outside-repo agent prompt + global CLAUDE + memory file language. Worklog records the canonical sentence chosen and confirms each outside-repo doc carries it (verbatim or paraphrased; exact match preferred).
    - Self-QA + dev-approval gate before "commit" (no actual git commit for outside-repo files; the dev-approval is the close).
- **Blocked by:** Wave 3 final droplet; **W4.1** (canonical post-Wave-3 sentence chosen in `main/CLAUDE.md` first; outside-repo docs mirror it).
- **Irreducible:** `true`. Rationale: cross-doc consistency between the four agent files + global CLAUDE + two memory files requires picking one canonical post-Wave-3 sentence and applying it consistently. Splitting per file leaves divergent paraphrases — exactly the failure mode `feedback_md_update_qa.md`'s self-QA gate exists to catch.
- **Notes:**
  - **Builder-driven** by recommendation. The cross-doc consistency check + 7 files + outside-repo audit-gap acceptance is heavy enough that orchestrator-inline editing risks losing track. Spawn a builder subagent (model: opus per `feedback_opus_builders_pre_mvp.md`) with explicit `Edit` / `Write` permission for the 7 paths above and the worklog. Subagent's spawn prompt MUST authorize outside-repo paths since they fall outside the project working dir.
  - **Audit-gap acceptance in the droplet description:** per drop_3/3.27 finding 5.D.10, document the trade-off explicitly so future maintainers know git history is not the audit trail for these files.
  - **No Tillsyn-runtime per-droplet QA spawn** — pre-MVP filesystem-MD mode (per REVISION_BRIEF "Pre-MVP rules"). The QA artifacts are a single self-QA pass + dev approval, not Tillsyn `build-qa-proof` / `build-qa-falsification` action items.
  - **Description-symbol verification:** for each outside-repo file edit, the builder verifies the existing paragraph it's about to retire — if the paragraph differs from REVISION_BRIEF's expected text (e.g. an agent file has been updated since REVISION_BRIEF was written), the builder records the divergence in the worklog Unknowns and routes back to the orch rather than silent-fixing.
  - **Memory rule alignment:** `feedback_bare_root_not_tracked.md` explicitly says global `~/.claude/CLAUDE.md` is not git-tracked; edits stay in worklog. `feedback_md_update_qa.md` requires self-QA + dev-approval gate before any MD edit lands. Both rules apply to this droplet.

## Cross-Wave Blocker Summary

```
Wave 1 final droplet (paths/packages/files/start_commit/end_commit + Project fields + always-on parent-blocks)
                                     ↓
Wave 2 final droplet (dispatcher loop + manual-trigger CLI)
                                     ↓
Wave 3 final droplet (orch-self-approval)
                                     ↓
   ┌─────────────┬────────────┬────────────┐
   ↓             ↓            ↓            ↓
  W4.1         W4.2         W4.3         W4.4
(CLAUDE.md)  (WIKI.md)  (STEWARD)  (outside-repo bundle)
                                     ↑
                        also blocked on W4.1 (cross-doc consistency)
```

W4.1 / W4.2 / W4.3 are file-disjoint and could run in parallel from a same-file-lock perspective. Per `feedback_md_update_qa.md`, the orchestrator does each MD edit inline in a single self-QA + dev-approval cycle, so practical sequencing is W4.1 → W4.2 → W4.3 → W4.4 in one work session. W4.4 hard-blocks on W4.1 because the canonical post-Wave-3 sentence is chosen there first.

## Out Of Scope Confirmations

Explicit non-scope, deferred to future drops or refinements per REVISION_BRIEF "Out-of-scope" + pre-MVP rules:

- **Closeout MD rollups** — `LEDGER.md`, `WIKI_CHANGELOG.md`, `REFINEMENTS.md`, `HYLLA_FEEDBACK.md`, `HYLLA_REFINEMENTS.md` per `feedback_no_closeout_md_pre_dogfood.md`. Wave 4 produces `BUILDER_WORKLOG.md` updates only.
- **Drop 4b scope** — gate execution, commit-agent integration, `git commit` + `git push` automation, Hylla reingest hook, auth auto-revoke on terminal state, git-status-pre-check on action-item creation. Wave 4 docs name these as "Drop 4b" — they do NOT promote any of them to past-tense.
- **Drop 4.5 scope** — TUI overhaul, columns-table retirement, file-viewer pane consuming `files`. Wave 1's `files` first-class field lands in 4a; the TUI consumer is Drop 4.5.
- **`task_id` → `action_item_id` rename** — REVISION_BRIEF mentioned this anticipatorily; Wave 1's actual scope does not include the rename. No `task_id` references touched in Wave 4.
- **Bare-root CLAUDE.md edits** — bare-root file is not git-tracked per `feedback_bare_root_not_tracked.md`. Wave 4 does NOT edit bare-root CLAUDE.md as part of any droplet (the `main/CLAUDE.md` edits in W4.1 do not propagate to bare-root automatically; that is a separate dev-managed sync).
- **No source code edits** — Wave 4 is docs-only. `mage ci` green is a smoke check, not a real test pass on new code.
- **No Hylla calls** — REVISION_BRIEF "Concrete planner spawn contract" forbids Hylla calls in planning; Wave 4 is markdown-sweep where Hylla is Go-only and irrelevant.
- **No subagent QA twins per droplet** — pre-MVP filesystem-MD mode. Each droplet's correctness is a self-QA + dev-approval gate, not Tillsyn QA action items.

## Notes

### Orchestrator-Driven vs Builder-Driven Per Droplet

Per `feedback_md_update_qa.md`: orchestrator does in-orch quick MD updates inline; systematic sweeps spawn a builder subagent (per drop_3/3.27 precedent for cross-file vocabulary sweeps).

| Droplet | Recommendation | Rationale |
| --- | --- | --- |
| W4.1 | Orch-driven (default), escalate to builder if hit count > ~15 | ~10 surgical edits in one file; in-orch self-QA suffices for that scale. |
| W4.2 | Orch-driven | ~3-5 edits (mostly verification-only) — squarely in-orch quick-update territory. |
| W4.3 | Orch-driven | ~3-5 edits, all in §8.1; tightly scoped. |
| W4.4 | Builder-driven | 7 outside-repo files + cross-doc consistency check + audit-gap acceptance — too heavy for in-orch attention. |

Final per-droplet decision lands at execution time per `feedback_md_update_qa.md`.

### Wave-Internal Sequencing

Strict linear: W4.1 → W4.2 → W4.3 → W4.4. The strict linearity is not a same-file lock (files are disjoint) but a self-QA budget rule — one MD edit + dev-approval per session round avoids context-switch errors. Plus W4.4 has a real cross-doc consistency dependency on W4.1 (canonical post-Wave-3 sentence chosen there first).

### Wave-External Sequencing

Wave 4 hard-blocks on Wave 1, Wave 2, AND Wave 3 closure. The orchestrator MUST verify all three waves' final droplets are `complete` before any Wave 4 droplet starts. Concretely: at PLAN.md unification time, the orch translates "Wave 1 final droplet" / "Wave 2 final droplet" / "Wave 3 final droplet" into specific globally-numbered `4a.X` blockers (e.g. `4a.13`, `4a.23`, `4a.28`, etc., depending on how the orch numbers the unified plan).

### Pre-MVP Constraints (Locked)

- No closeout MD rollups (`feedback_no_closeout_md_pre_dogfood.md`).
- No subagent QA twins per droplet — filesystem-MD mode.
- Builders run **opus** when Wave 4 escalates a droplet to builder-driven (`feedback_opus_builders_pre_mvp.md`).
- Single-line commits (`feedback_commit_style.md`) — when a Wave 4 droplet commits, message format: `docs(drop-4a): wave 4 — <droplet-scope-short>` — one line, no body.
- No Hylla calls during Wave 4 planning or execution — Wave 4 touches only non-Go files.

### Ambiguities Routed To Orchestrator At Synthesis Time

- **W4.1's Drop-1-mention edits:** the `## Action-Item Lifecycle` section says "Drop 1 of the cascade plan" and "becomes always-on in Drop 1." Verify against actual code state which Wave-1-of-Drop-4a-droplet landed `failed` as a real terminal state vs Wave 1 of the original cascade plan. The plan numbers may have shifted; orch confirms before W4.1's builder/inline edit.
- **W4.4's `~/.claude/CLAUDE.md` actual hit count:** the planner could not run `Grep` against the file (permission rejected during evidence-gathering). The droplet's hit count is therefore unverified at plan time. Orch records actual hit count in worklog at execution.
- **W4.4's per-agent-file S2-fallback paragraph existence:** REVISION_BRIEF asserts all four go-* agent files carry the paragraph, but evidence-gathering only confirmed `go-builder-agent.md` does NOT carry it (read line 1-180 of the file directly). The other three agent files were not exhaustively read. Builder verifies per file at execution; "no paragraph found, no edit" is a valid worklog outcome.

## Hylla Feedback

N/A — planning touched non-Go files only.
