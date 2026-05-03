# DROP_3 — UNIT D — ADOPTER BOOTSTRAP + CASCADE VOCABULARY DOC SWEEP

**State:** planning
**Blocked by:** Unit A's WIKI `## Cascade Vocabulary` section being authored (orch wires inter-unit `blocked_by` at synthesis); soft-blocked on Units B + C for the final wrap-up sweep so all rename / symbol changes from those units land before the cleanup pass.
**Paths (expected):** `~/.claude/agents/*.md` (10 files, NOT git-tracked), `~/.claude/skills/go-project-bootstrap/SKILL.md` + `references/template.md` (NOT git-tracked), `~/.claude/skills/fe-project-bootstrap/SKILL.md` + `references/template.md` (NOT git-tracked), `main/CLAUDE.md`, `main/workflow/example/CLAUDE.md`, `main/workflow/example/drops/WORKFLOW.md`, `main/PLAN.md` (sweep only — no scope changes), `main/WIKI.md` (pointer-touchups; canonical glossary owned by Unit A), `main/STEWARD_ORCH_PROMPT.md`, `main/AGENT_CASCADE_DESIGN.md`, `main/AGENTS.md`, `main/README.md`, `main/CONTRIBUTING.md`, `main/SEMI-FORMAL-REASONING.md`, `main/HYLLA_WIKI.md`, `main/tillsyn-project.md`, `main/DROP_1_75_ORCH_PROMPT.md`.
**Packages (expected):** none — Unit D is documentation only. No Go code edits.
**PLAN.md ref:** `main/PLAN.md` § 19.3 — drop 3 — Template Configuration, bullets *"Adopter bootstrap updates (`go-project-bootstrap` + `fe-project-bootstrap` skills + every `CLAUDE.md` template)"* (line 1650) and *"Per-drop wrap-up for cascade vocabulary specifically"* (line 1651).
**Started:** 2026-05-02
**Closed:** —

## Scope

Unit D handles the documentation-propagation tail of Drop 3: every place that currently holds a duplicate or stale cascade-vocabulary description gets either flipped to a pointer at the canonical Unit-A WIKI `## Cascade Vocabulary` section or rewritten to match the new closed `structural_type` enum. Unit D consumes Unit A's vocabulary content; it does not author it.

Six discrete work surfaces:

1. **Agent file frontmatter reminders** — every `~/.claude/agents/*.md` file (10 files: 5 Go variants + 5 FE variants) gets a one-line reminder pointing at the WIKI glossary. Per PLAN.md § 19.3 line 1650: *"Structural classifications (drop | segment | confluence | droplet) live in WIKI glossary — never redefine."* These files are NOT git-tracked in this repo; the edits land on the dev's `~/.claude/agents/` directory directly and the diff is recorded in `BUILDER_WORKLOG.md`.
2. **`go-project-bootstrap` skill update** — `SKILL.md` workflow step (under "Add Go-specific guidance") plus `references/template.md` content are extended so any new Go project bootstrapped post-Drop-3 receives a CLAUDE.md pointer line *"Cascade vocabulary canonical: `WIKI.md` § `Cascade Vocabulary`."* and a WIKI scaffolding seed with the `## Cascade Vocabulary` section pre-filled. NOT git-tracked.
3. **`fe-project-bootstrap` skill update** — same pattern as #2 for the FE variant. NOT git-tracked.
4. **`main/CLAUDE.md` + `workflow/example/CLAUDE.md` cascade-glossary pointer** — both files gain the explicit pointer line; in the case of `workflow/example/CLAUDE.md`, this hardens the generic-template version that adopters copy. Both are git-tracked and land in the Drop 3 PR.
5. **In-repo legacy-vocabulary sweep** — every active doc / agent prompt / slash-command / skill / memory file that still says `action_item` / `action-item` / `action item` / `ActionItem` (where the prose conflates kind with role with structural_type) gets rewritten. Also flag any remaining instances of the retired `"drops all the way down"` framing (Drop 2 closeout did a partial sweep — verify completeness). The sweep is **scoped to active canonical docs only**: historical worklogs (`workflow/drop_0/`, `workflow/drop_1/`, `workflow/drop_2/`), `LEDGER.md` historical entries, `WIKI_CHANGELOG.md`, `HYLLA_FEEDBACK.md` historical content, `REFINEMENTS.md` historical content are explicitly excluded — those are audit trail and never get rewritten.
6. **Per-drop wrap-up final pass** — runs after Units A + B + C have all landed their renames and new symbols. Sweeps any new vocabulary the other units introduced (`structural_type`, the 4 enum values, template-system terms from Unit B, `principal_type: steward` from Unit C) into all the same docs as #5 plus crosswalks `metadata.role` vs `metadata.structural_type` everywhere they previously got conflated.

**Out of scope (explicit):**

- Authoring the canonical WIKI `## Cascade Vocabulary` section content — that is **Unit A's** deliverable. Unit D consumes it.
- The `metadata.structural_type` enum definition + validation — Unit A.
- TOML template schema + `Template.AllowsNesting` + `templates/builtin/default.toml` — Unit B.
- STEWARD `principal_type: steward` + `metadata.owner` field + auth state-lock — Unit C.
- Rewriting historical workflow drop dirs (`workflow/drop_0/`, `workflow/drop_1/`, `workflow/drop_2/`) — audit trail, immutable per memory rule "Never Remove Workflow Drop Files."
- Cleanup of `LEDGER.md`, `WIKI_CHANGELOG.md`, `HYLLA_FEEDBACK.md`, `REFINEMENTS.md` historical entries — audit trail, even if vocabulary changed since they were written.

**Pre-MVP rules in effect (per memory):**

- No `CLOSEOUT.md` for this unit; worklog MDs (this `UNIT_D_PLAN.md`, `UNIT_D_BUILDER_WORKLOG.md`, `UNIT_D_BUILDER_QA_PROOF.md`, `UNIT_D_BUILDER_QA_FALSIFICATION.md`, `UNIT_D_PLAN_QA_PROOF.md`, `UNIT_D_PLAN_QA_FALSIFICATION.md`) DO happen at the orch's discretion.
- Builders run **opus** per `feedback_opus_builders_pre_mvp.md`.
- No `git rm` of any `workflow/drop_3/` file (per memory rule "Never Remove Workflow Drop Files"). Files that don't apply are never created, not stamped-then-deleted.
- The **`~/.claude/agents/` and `~/.claude/skills/` files are outside this repo's git tree.** The builder spawned for any droplet that edits those files needs explicit edit-scope authorization in its spawn prompt. The diff is documented in `UNIT_D_BUILDER_WORKLOG.md`; nothing lands in the Drop 3 PR for those files.
- The MD-cleanup carve-out from Drop 2 (trivially adjacent fixes) does **not** apply here — Unit D's sweep is intentional and scoped, not opportunistic. Each droplet defines exactly which sites it touches and for what reason.
- The **retired "drops all the way down" / "tasks all the way down" framing** must be flagged in the sweep. Replace with the waterfall-metaphor + `structural_type` axis when it appears in active docs.

## Planner

Decomposition into **6 atomic droplets**. Order: 5.D.1 (agent frontmatter sweep — independent) → 5.D.2 + 5.D.3 (parallel skill updates — disjoint files) → 5.D.4 (CLAUDE.md template pointer drops) → 5.D.5 (in-repo legacy-vocab sweep — touches `main/CLAUDE.md`, blocks on 5.D.4) → 5.D.6 (per-drop wrap-up — depends on Units A/B/C all landing).

Acceptance verification target throughout: `mage ci` green at unit boundary (no Go code touched, but `mage ci` runs the markdown-aware checks — none currently exist beyond gofmt-equivalent on Go files, so CI exit status is the smoke test). Per-droplet smoke test: `git diff --check` (no whitespace errors) + manual diff review. **Never `mage install`.**

**Inter-unit dependencies (orch wires `blocked_by` at synthesis):**

- 5.D.1, 5.D.4, 5.D.5: depend on **Unit A's WIKI § `Cascade Vocabulary` section being authored** so the pointer references something real.
- 5.D.1: depends on **Unit A's `structural_type` enum 4-value definition being final** (so the agent frontmatter reminder names exactly `drop | segment | confluence | droplet`).
- 5.D.6: depends on **Units A + B + C all landing in `mage ci` green state**, so the final sweep covers every vocabulary change introduced in the drop.

**Intra-unit blockers:**

- 5.D.5 shares `main/CLAUDE.md` with 5.D.4 → 5.D.5 `blocked_by: 5.D.4`.
- 5.D.5 shares `main/CLAUDE.md` and `main/workflow/example/CLAUDE.md` with 5.D.4 (above) and with 5.D.6 → 5.D.6 `blocked_by: 5.D.5`.
- 5.D.2 + 5.D.3 are disjoint (different skill folders) — can run in parallel.
- 5.D.4 shares no path with 5.D.1, 5.D.2, or 5.D.3 → independent.

---

### Unit D — Adopter Bootstrap + Cascade Vocabulary Doc Sweep

#### Droplet 5.D.1 — Agent file frontmatter cascade-glossary reminder

- **State:** todo
- **Paths:**
  - `~/.claude/agents/go-builder-agent.md` (NOT git-tracked)
  - `~/.claude/agents/go-planning-agent.md` (NOT git-tracked)
  - `~/.claude/agents/go-qa-proof-agent.md` (NOT git-tracked)
  - `~/.claude/agents/go-qa-falsification-agent.md` (NOT git-tracked)
  - `~/.claude/agents/go-research-agent.md` (NOT git-tracked)
  - `~/.claude/agents/fe-builder-agent.md` (NOT git-tracked)
  - `~/.claude/agents/fe-planning-agent.md` (NOT git-tracked)
  - `~/.claude/agents/fe-qa-proof-agent.md` (NOT git-tracked)
  - `~/.claude/agents/fe-qa-falsification-agent.md` (NOT git-tracked)
  - `~/.claude/agents/fe-research-agent.md` (NOT git-tracked)
- **Packages:** none.
- **Acceptance:**
  - Each of the 10 agent files gains a one-line cascade-glossary reminder placed in the **prose body immediately after the YAML frontmatter `---` close marker** (NOT inside the YAML block — the YAML schema uses `name`, `description`, `tools` only and an unknown key would be quietly dropped). Exact text: *"Structural classifications (`drop` | `segment` | `confluence` | `droplet`) live in the project WIKI's `## Cascade Vocabulary` section — never redefine here."*
  - The reminder lives **before** any "You are the …" identity sentence so it reads as agent-wide context, not as identity.
  - Reminder is identical across all 10 files (no per-language drift).
  - Each file's diff: exactly +1 line (the reminder) plus an optional +1 blank-line padding around it; no other changes.
  - Verification: `wc -l` before and after delta is exactly +1 or +2 (with padding) per file.
  - Builder records the 10-file diff in `UNIT_D_BUILDER_WORKLOG.md` since these paths don't land in the Drop 3 PR.
- **Blocked by:** none (intra-unit). Inter-unit: Unit A's `structural_type` enum 4-value list final + Unit A's WIKI `## Cascade Vocabulary` section authored.
- **Notes:**
  - Builder spawn prompt MUST explicitly authorize edits to `~/.claude/agents/*.md` since those paths sit outside the repo working dir. The orch passes the absolute path list verbatim.
  - The reminder text uses the `Structural classifications (drop | segment | confluence | droplet) live in WIKI glossary — never redefine.` phrasing called out in PLAN.md § 19.3 line 1650, lightly formatted with backticks for the enum values.
  - **Open architectural question for orch (not for the builder)**: PLAN.md § 19.3 says *"frontmatter body"* — ambiguous between (a) inside the YAML frontmatter block as a new key, (b) the prose body immediately under the frontmatter, or (c) somewhere within the agent's identity prose. This droplet locks (b) — prose body immediately after the YAML close — because YAML insertion would require schema-validated keys (which don't exist for free-form notes) and identity-prose insertion buries the note.

#### Droplet 5.D.2 — `go-project-bootstrap` skill update (cascade glossary in template + workflow step)

- **State:** todo
- **Paths:**
  - `~/.claude/skills/go-project-bootstrap/SKILL.md` (NOT git-tracked) — append a new workflow bullet
  - `~/.claude/skills/go-project-bootstrap/references/template.md` (NOT git-tracked) — append cascade-glossary pointer line + WIKI-scaffolding seed pointer
- **Packages:** none.
- **Acceptance:**
  - `SKILL.md` workflow section gains a new bullet under "3. Add Go-specific guidance" (or as its own step "3.5 Pre-fill cascade glossary"): *"Pre-fill the project's `WIKI.md` with a `## Cascade Vocabulary` section seeded from the canonical Tillsyn WIKI glossary (boilerplate text owned by Unit A — TODO orchestrator: confirm seed-text owner). Add a CLAUDE.md pointer line: 'Cascade vocabulary canonical: `WIKI.md` § `Cascade Vocabulary`.'"*
  - `references/template.md` gains a new top-level rule under "Start from these rules": *"Cascade vocabulary canonical: `WIKI.md` § `Cascade Vocabulary` — never redefine in CLAUDE.md."*
  - `references/template.md` gains a "Required WIKI scaffolding" subsection naming the `## Cascade Vocabulary` section the bootstrap MUST seed when `WIKI.md` doesn't exist or doesn't have that section yet.
  - Skill workflow's **Completion Bar** gains a new line: *"the project `WIKI.md` includes a `## Cascade Vocabulary` section (pre-filled or pointer to canonical)."*
  - Builder records the diff in `UNIT_D_BUILDER_WORKLOG.md` since these paths don't land in the Drop 3 PR.
- **Blocked by:** none (intra-unit). Inter-unit: Unit A's WIKI `## Cascade Vocabulary` section authored (the boilerplate text the bootstrap seeds is the canonical version Unit A wrote).
- **Notes:**
  - Builder spawn prompt MUST explicitly authorize edits to `~/.claude/skills/go-project-bootstrap/` since the path is outside the repo.
  - **Open architectural question for orch (not for the builder)**: where does the canonical WIKI `## Cascade Vocabulary` boilerplate that the bootstrap seeds *live*? Three options: (i) bootstrap copies-by-value from a canonical text file shipped in `templates/builtin/wiki-cascade-vocabulary.md` (requires the new `templates/` package Drop 3 reintroduces — Unit B's territory); (ii) bootstrap writes a pointer-only `## Cascade Vocabulary` section that just says *"See the Tillsyn project's WIKI.md for canonical definitions"* and lets each adopter project re-author from scratch; (iii) bootstrap embeds the boilerplate inline in `references/template.md`. This droplet's acceptance criteria assume **(iii) inline in `references/template.md`** because it requires no new infrastructure and is the lowest-risk option. Orch confirms before builder fires.

#### Droplet 5.D.3 — `fe-project-bootstrap` skill update (cascade glossary in template + workflow step)

- **State:** todo
- **Paths:**
  - `~/.claude/skills/fe-project-bootstrap/SKILL.md` (NOT git-tracked) — append a new workflow bullet
  - `~/.claude/skills/fe-project-bootstrap/references/template.md` (NOT git-tracked) — append cascade-glossary pointer line + WIKI-scaffolding seed pointer
- **Packages:** none.
- **Acceptance:**
  - Identical pattern to 5.D.2, applied to the FE variant. Workflow bullet placement: under "3. Add FE-specific guidance" or as its own step "3.5 Pre-fill cascade glossary."
  - `references/template.md` gets the same top-level rule, the same "Required WIKI scaffolding" subsection, and the same Completion Bar update.
  - Boilerplate text seeded into FE adopter projects' WIKIs is the **same** content as the Go variant — cascade vocabulary is language-agnostic.
  - Builder records the diff in `UNIT_D_BUILDER_WORKLOG.md`.
- **Blocked by:** none (intra-unit). Inter-unit: same as 5.D.2 (Unit A's WIKI section + boilerplate-source decision).
- **Notes:**
  - Disjoint files from 5.D.2 — can run in parallel with 5.D.2.
  - Builder spawn prompt MUST explicitly authorize edits to `~/.claude/skills/fe-project-bootstrap/`.

#### Droplet 5.D.4 — Cascade-glossary pointer in `main/CLAUDE.md` + `workflow/example/CLAUDE.md`

- **State:** todo
- **Paths:**
  - `main/CLAUDE.md` (git-tracked)
  - `main/workflow/example/CLAUDE.md` (git-tracked) — the generic adopter template
- **Packages:** none.
- **Acceptance:**
  - `main/CLAUDE.md`: in the existing "Cascade Plan" section (which already says *"The cascade … is designed in `PLAN.md`"*), insert a new sibling line: *"Cascade vocabulary canonical: `WIKI.md` § `Cascade Vocabulary` — never redefine here."* The line lands as the second sentence of the section, not as a new section.
  - `main/workflow/example/CLAUDE.md`: in the existing top-level matter (around the "This file lives in the **primary work checkout**" paragraph), add an explicit pointer line: *"Cascade vocabulary canonical: project `WIKI.md` § `Cascade Vocabulary` — every adopter project's CLAUDE.md MUST include this pointer and MUST NOT redefine the structural_type vocabulary locally."* The line lands near the existing reading-order bullet ("Read `WIKI.md` + `PLAN.md` + ...").
  - Both edits are pure additions — no existing content is removed.
  - Both files lint clean (markdown linter — no broken links, consistent code-fence usage).
  - Both files land in the Drop 3 PR (git-tracked).
- **Blocked by:** none (intra-unit). Inter-unit: Unit A's WIKI `## Cascade Vocabulary` section authored (so the pointer resolves).

#### Droplet 5.D.5 — In-repo legacy-vocabulary sweep (active canonical docs only)

- **State:** todo
- **Paths (in-scope active canonical docs):**
  - `main/CLAUDE.md`
  - `main/PLAN.md` — sweep ONLY where text conflates `action_item` with `kind` / `role` / `structural_type` (vast majority of `action_item` mentions are correct usage and stay). Edits are surgical, not blanket. Spec text describing the schema (e.g. PLAN.md § 19.3 line 1629 *"`action_item` is NOT a structural_type value — it is the generic node concept"*) stays as-is — that IS the schema and Unit D doesn't rewrite the schema.
  - `main/WIKI.md` — pointer-only edits; canonical glossary content owned by Unit A.
  - `main/STEWARD_ORCH_PROMPT.md`
  - `main/AGENT_CASCADE_DESIGN.md`
  - `main/AGENTS.md`
  - `main/README.md`
  - `main/CONTRIBUTING.md`
  - `main/SEMI-FORMAL-REASONING.md`
  - `main/HYLLA_WIKI.md`
  - `main/tillsyn-project.md`
  - `main/DROP_1_75_ORCH_PROMPT.md`
  - `main/workflow/example/CLAUDE.md`
  - `main/workflow/example/drops/WORKFLOW.md`
  - `main/workflow/example/drops/_TEMPLATE/**` (any MD files inside the template scaffold)
  - `main/workflow/example/drops/DROP_N_EXAMPLE/**` (any MD files inside the example walkthrough)
  - `~/.claude/agents/*.md` — full pass after 5.D.1's frontmatter reminder is added
  - `~/.claude/skills/*/SKILL.md` + `~/.claude/skills/*/references/*.md` — global skills (semi-formal-reasoning, plan-from-hylla, qa-proof-checker, qa-falsification-checker, tui-golden-review, tui-vhs-review)
  - `~/.claude/CLAUDE.md` — the global Claude working rules
  - `~/.claude/projects/-Users-evanschultz-Documents-Code-hylla-tillsyn/memory/MEMORY.md` — review only, no edits unless a rule has actually been retired
- **Paths (out of scope — explicitly excluded):**
  - `main/workflow/drop_0/**`, `main/workflow/drop_1/**`, `main/workflow/drop_2/**` — historical worklog audit trail.
  - `main/workflow/drop_3/**` (except this very file `UNIT_D_PLAN.md` — that gets touched only by Unit D itself, not by 5.D.5's sweep).
  - `main/LEDGER.md` — historical ledger entries are audit trail.
  - `main/WIKI_CHANGELOG.md` — historical changelog.
  - `main/HYLLA_FEEDBACK.md` — historical feedback rollup.
  - `main/REFINEMENTS.md` — historical refinements list.
  - `main/HYLLA_REFINEMENTS.md` — historical refinements list.
  - All Go source under `internal/` and `cmd/` — Unit D is doc-only.
- **Packages:** none.
- **Acceptance:**
  - Sweep pass 1: `Grep "drops all the way down"` and `Grep "tasks all the way down"` across in-scope paths returns zero hits. Any hits found are rewritten to the waterfall-metaphor wording from Unit A's WIKI glossary (concrete replacement: *"every non-project node is classified by `metadata.structural_type` as one of `drop | segment | confluence | droplet`"*).
  - Sweep pass 2: `Grep "action_item\|action-item\|action item\|ActionItem"` across in-scope paths is reviewed line-by-line. Hits where the prose **conflates kind with role** (e.g. *"create an action_item with kind=qa"* where the intent is the QA *role* not a fictitious `qa` kind) are rewritten to use the correct vocabulary. Hits that are correct schema usage (e.g. *"`action_items` table"*, *"the action_item generic kind"* — accurate post-Drop-1.75) stay as-is.
  - Sweep pass 3: `Grep "slice\|build-task\|plan-task\|qa-check"` across in-scope paths — these older pre-Drop-1.75 vocabulary terms get rewritten where they appear in **active** prose to use the closed 12-kind enum (`build` / `plan` / `build-qa-proof` / `build-qa-falsification` / `plan-qa-proof` / `plan-qa-falsification`). The `~/.claude/agents/*.md` files contain many of these (they were authored pre-Drop-1.75) and are the largest target. The `main/PLAN.md` audit-trail mentions of historical kinds (in change-log entries describing what kinds *used* to be) stay as-is.
  - Sweep pass 4: `metadata.role` vs `metadata.structural_type` crosswalk. Anywhere a doc previously said *"role determines …"* where the truth is *"kind determines … and role only disambiguates QA flavor"*, the doc is rewritten to make the orthogonal-axes design (`kind` × `role` × `structural_type`) explicit.
  - Builder records the per-file diff inventory in `UNIT_D_BUILDER_WORKLOG.md` with one row per file: `<path> <hits> <rewrites> <left-as-is>`.
  - `mage ci` green (no Go code touched; CI exit status is the smoke test).
  - Files outside the repo (`~/.claude/**`) are diffed in the worklog and do NOT land in the Drop 3 PR.
- **Blocked by:** 5.D.4 (shares `main/CLAUDE.md` and `main/workflow/example/CLAUDE.md`). Inter-unit: Unit A's WIKI glossary authored (so rewrite targets reference real text).
- **Notes:**
  - This droplet is intentional and scoped — NOT opportunistic markdown cleanup. The MD-cleanup carve-out from Drop 2 does not apply.
  - Builder spawn prompt MUST explicitly enumerate the in-scope paths and the explicitly-excluded paths so no historical audit trail gets touched.
  - Builder uses `Read` + `Grep` + `Edit` tools only — never `sed` / `awk` / `python` shell parsers (per memory rule "Use Native Tools Not Shell Parsers").
  - **Self-QA the sweep before moving on (per memory rule "Self-QA MD Updates, Ask Before Applying Suggestions")**: builder presents the per-file diff inventory + a sample of rewrites to the orchestrator, who presents to the dev, who approves before commit.

#### Droplet 5.D.6 — Per-drop final wrap-up sweep (after Units A/B/C land)

- **State:** todo
- **Paths:** Same in-scope set as 5.D.5, re-swept for any new vocabulary introduced by Units A + B + C:
  - Unit A: `structural_type`, the 4 enum values (`drop | segment | confluence | droplet`), atomicity rules, plan-QA-falsification new attack vectors.
  - Unit B: TOML template-system terms (`Template.AllowsNesting`, `[child_rules]`, `KindCatalog`, `templates/builtin/default.toml`), agent-binding kind fields (`agent_name` / `model` / `effort` / `tools` / `max_tries` / `max_budget_usd` / `max_turns` / `auto_push` / `commit_agent` / `blocked_retries` / `blocked_retry_cooldown`).
  - Unit C: `principal_type: steward`, `metadata.owner = STEWARD`, auth-level state-lock, template auto-generation of STEWARD level_2 items.
- **Packages:** none.
- **Acceptance:**
  - Same sweep methodology as 5.D.5, but the search terms are the new vocabulary from A/B/C rather than the legacy vocabulary.
  - Verify every doc that references the new vocabulary uses it consistently and points back to the canonical source (Unit A's WIKI section for cascade vocabulary, Unit B's template-system docs for template terms, Unit C's auth-model docs for STEWARD principal_type).
  - Verify the `metadata.role` × `metadata.structural_type` × `kind` orthogonal-axes design is documented coherently in at least `main/CLAUDE.md`, `main/WIKI.md`, `main/PLAN.md`, and `main/workflow/example/CLAUDE.md` (the four primary always-read docs).
  - Verify no new doc inadvertently re-introduces the retired *"drops all the way down"* framing.
  - Builder records the per-file diff inventory in `UNIT_D_BUILDER_WORKLOG.md`.
  - `mage ci` green.
  - Self-QA pass: builder presents inventory + sample to orchestrator → dev approves → commit.
- **Blocked by:** 5.D.5 (shares paths). Inter-unit: Units A + B + C all closed (`mage ci` green at each unit boundary).
- **Notes:**
  - This is the **final docs-only droplet under Drop 3** per PLAN.md § 19.3 line 1651: *"Commit the sweep as a final docs-only droplet under drop 3."*
  - Commit message convention (per memory rule "Single-Line Commits"): `docs(drop-3): cascade vocabulary final sweep` (one line, no body).
  - **Open architectural question for orch (not for the builder)**: timing — does this run **before** the Drop 3 PR merges (so the sweep ships in the Drop 3 PR) or **after** the PR merges (so STEWARD or a follow-up commit handles it)? This droplet's acceptance criteria assume **before** — the sweep ships inside Drop 3. Orch confirms before builder fires.

---

## Notes

### Cross-Unit Dependency Map (For Orchestrator Synthesis)

| Droplet | Depends On (Inter-Unit) | Depends On (Intra-Unit) |
|---|---|---|
| 5.D.1 | Unit A: `structural_type` enum + WIKI § Cascade Vocabulary authored | none |
| 5.D.2 | Unit A: WIKI § Cascade Vocabulary authored + boilerplate-source decision | none |
| 5.D.3 | Unit A: same as 5.D.2 | none |
| 5.D.4 | Unit A: WIKI § Cascade Vocabulary authored | none |
| 5.D.5 | Unit A: WIKI § Cascade Vocabulary authored | 5.D.4 (shares `main/CLAUDE.md` + `main/workflow/example/CLAUDE.md`) |
| 5.D.6 | Units A + B + C all closed (`mage ci` green at unit boundary) | 5.D.5 (shares paths) |

Orchestrator wires the inter-unit `blocked_by` edges at synthesis time when assembling the unified `## Planner` section in `workflow/drop_3/PLAN.md`. The intra-unit edges are already wired above.

### Architectural Questions Returned To Orchestrator

1. **Skill-file frontmatter convention.** Both `go-project-bootstrap/SKILL.md` and `fe-project-bootstrap/SKILL.md` use a YAML frontmatter block with exactly two keys (`name`, `description`). There is no standard "cascade glossary pointer" key. This plan locks 5.D.2 + 5.D.3 to insert the pointer in `references/template.md` (a file already explicitly listed under "Resources" in each SKILL.md) rather than in the YAML — but if the dev wants it in the YAML as a free-form key, that's a one-line change to the acceptance criteria. **Decision needed before Phase 1 builders fire.**
2. **Per-drop final wrap-up timing (5.D.6).** Two options: (a) runs at end of Drop 3 *before* PR merge, sweep lands inside Drop 3 PR. (b) runs at end of Drop 3 *after* PR merge, sweep lands as a follow-up commit on `main` post-merge handled by STEWARD or by a Drop 3.5 mini-drop. This plan locks **(a)**. Confirm before builder fires.
3. **Boilerplate `## Cascade Vocabulary` content owner.** Three options for how the bootstrap skill seeds the section into adopter projects' `WIKI.md`: (i) reference a canonical text file in `templates/builtin/wiki-cascade-vocabulary.md` (requires Unit B coordination — the templates package is Unit B's deliverable). (ii) bootstrap writes a pointer-only stub *"See Tillsyn upstream WIKI"* and adopter projects re-author from scratch (every adopter ends up with potentially-divergent vocabulary — defeats the whole point). (iii) bootstrap embeds the boilerplate **inline in `references/template.md`** so the skill is fully self-contained and adopter WIKIs all start identical. This plan locks **(iii)**. Confirm before 5.D.2 + 5.D.3 builders fire.
4. **Agent frontmatter insertion location.** PLAN.md § 19.3 line 1650 says *"frontmatter body"* — ambiguous. This plan locks **prose body immediately after the YAML close** (option (b) in droplet 5.D.1's notes). Confirm before 5.D.1 builder fires.

### Out-Of-Scope Confirmations

- **No Go code edits.** Unit D is documentation only.
- **No `git rm` of any file in `workflow/drop_3/`** per memory rule "Never Remove Workflow Drop Files."
- **No edits to historical drop dirs** (`workflow/drop_0/`, `workflow/drop_1/`, `workflow/drop_2/`) — audit trail.
- **No edits to historical audit-trail MDs** (`LEDGER.md`, `WIKI_CHANGELOG.md`, `HYLLA_FEEDBACK.md`, `REFINEMENTS.md`, `HYLLA_REFINEMENTS.md`) — audit trail.
- **No subagents for MD work** per memory rule "Self-QA MD Updates, Ask Before Applying Suggestions" — builder runs self-QA inline + presents to orchestrator who presents to dev for approval before commit. (This rule applies to in-session orch-driven MD updates; for Unit D, the builder subagent IS the work-doer, but the self-QA + dev-approval gate still applies before commit.)

### Hylla Feedback

None — Hylla was not used. Unit D is pure markdown / skill / agent-prompt work; Hylla today indexes Go only (per memory rule "Hylla Indexes Only Go Files Today"). All evidence gathered via `Read`, `Grep`, `Glob`, `Bash` on doc files. **N/A for this planning unit — planning touched non-Go files only.**
