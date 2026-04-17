# Consolidation Decisions Ledger — 2026-04-16

Converged decisions from the `PLAN.md` consolidation pass. Pre-apply ledger — rolls into `PLAN.md` after plan edits land. Dev deletes this file once every downstream fold + rename is committed and confirmed.

---

## 1. File Disposition

### 1.1. Deleted Pre-Session (by Dev)

- `/AGENT_PROMPTS.md` — bare-root stale file; referenced removed `PLAN.md`; used `just` instead of mage; pre-reset "embeddings implementation branch" body.
- `/main/PLAN.md` — outdated pre-reset content.

### 1.2. Fold-and-Delete (dev handles delete; no files removed until convergence confirmed)

| File | Fold Target |
|---|---|
| `main/temp.md` (old chat-transcript version; THIS file replaces it) | Residuals into §1.4 / §2 / §3 / §15 of plan |
| `main/HEADLESS_DISCUSSIONS.md` | §4.1 refine + new §23 + new §24 + new drop 4.5 + §19.10 refinement |
| `main/TOS_COMPLIANCE.md` | New §22 + threaded across refinement drops |
| `main/TOS_DISCUSSIONS.md` | Converged rows → §22; pending Qs → refinement-drop placeholders |
| `main/MINIONS_RESEARCH_2026-04-13.md` | §20 open questions + §21 source links + risk register |

### 1.3. Fold-into-README (dev handles source delete)

- `main/TILLSYN_PURPOSE_AND_INTEGRATION_FRAMING_2026-04-11.md` → `main/README.md` with accuracy verification pass.

### 1.4. Renamed This Pass

- `main/PLAN.md` → `main/PLAN.md` (final step of consolidation).
- `main/SLICE_1_ORCH_PROMPT.md` → `main/DROP_1_ORCH_PROMPT.md` (after cross-ref sweep).

### 1.5. Preserved Living Docs (Untouched This Pass)

`LEDGER.md`, `WIKI.md`, `WIKI_CHANGELOG.md`, `HYLLA_WIKI.md`, `HYLLA_FEEDBACK.md`, `HYLLA_REFINEMENTS.md`, `REFINEMENTS.md`, bare-root + `main/` `CLAUDE.md`, bare-root + `main/` `AGENTS.md`, `main/README.md` (edited for TILLSYN_PURPOSE fold), `main/CONTRIBUTING.md`, `third_party/teatest_v2/README.md`, `main/STEWARD_ORCH_PROMPT.md`.

---

## 2. Converged Decisions (Land in Plan)

### 2.1. Concurrency Cap

N=6 hard-coded for dogfood (updated from prior N=4). Configurability deferred to refinement drop 10. §12 adds the N=6 number + the refinement-drop pointer.

### 2.2. Every-Drop Start + End Review Subdrops (New Structural Rule)

Every drop's tree starts with `DROP N START — PLANNING CONFIRMATION WITH DEV` subdrop and ends with `DROP N END — REVIEW DONE + CORRECT` subdrop. Dev-gated. Absorbs existing `DROP N START — REFINEMENT REVIEW` + `DROP N END — LEDGER UPDATE` as siblings inside the bracketing. Replaces the §19 "per-drop wrap-up" line.

Touches: §2 hierarchy, §3 ASCII, §17.1 prerequisites (creation rule), §19 per-drop shape. New `drop-human-verify` kind registers the dev-gated subdrop shape.

### 2.3. Drop-Node Domain Field Expansion — `files []string`

In addition to drop 1's `paths []string` + `packages []string`, add `files []string` as a first-class domain field. Populated by planner via TUI path picker → file picker. Lands in drop 1 alongside paths/packages. §17.1 + §19.1 updated.

### 2.4. TUI File Viewer with Glamour (Was §24 Scope Expansion)

TUI gains a file-viewer pane that renders drop-attached files via `charmbracelet/glamour` (markdown + syntax-highlighted code). Sibling surfaces: git-diff-per-plan-item against `start_commit`; path picker; file picker. Lands inside drop 4.5 (§2.5 below).

### 2.5. Drop 4.5 — Frontend + TUI Overhaul (New)

Inserted as concurrent-track drop starting alongside drop 5 dogfooding. Depends on drop 1 (failed + domain fields) + drop 4 (dispatcher core). Hosts §24 (file viewer + git diff) + §23 (mention routing UX) + all TUI bindings for drop 2's dotted-address nav (§2.6 below). Requires its own planning subdrop before builder fires. Per dev: starts early to inform TUI direction.

### 2.6. Dotted-Address Fast-Nav — Drop 2

`proj_name-0.1.5.2` shorthand lands in drop 2 alongside the hierarchy refactor (phase → drop rename + infinite nesting). CLI + MCP read paths first; TUI bindings follow inside drop 4.5. §1.4 + §19.2 updated.

### 2.7. §23 Mention Routing — Lineage From Current Plan

§23 text explicitly cites its lineage from the existing CLAUDE_MINIONS_PLAN / cascade design (not fresh-invented). From HEADLESS_DISCUSSIONS §3.1: Tillsyn-defined agents via `claude -p --append-system-prompt`, mention-routing model, inter-orchestrator comms.

### 2.8. Minions + Semi-Formal Full-Benefit Rule

Plan-text explicit rule: cascade design uses Stripe Minions + semi-formal reasoning to the full extent of their benefit — deterministic-agentic-deterministic sandwich, mandatory certificate structure, hypothesis-refinement loop, evidence grounding. Not just cited — structurally enforced by template config, gate placement, and QA agent prompts. §10 (Trust Model) + §11 (Semi-Formal) gain this rule.

### 2.9. REFINEMENTS Tracker + Human-Verify Kind

REFINEMENTS.md tracker + `DROP N START — REFINEMENT REVIEW` first-child (absorbed into §2.2 bracketing) + new `drop-human-verify` kind for dev-gated subdrops. §1.4 + §2 + §3 updated.

### 2.10. `go.mod` Replace-Directive Cleanup — Drop 1 First-Task

Strip every `replace` directive in `go.mod` except the fantasy-fork. Lands as drop 1's first bullet, before lifecycle work. §19.1 updated.

### 2.11. `.bare/` Subdir Retrofit — Standalone, This Pass

Fold bare-repo internals into `.bare/` subdir + root `.git` file pointing at `./.bare`. Pure dev-box reorg, no Go code, not scoped to a drop. Must update every worktree's `.git` pointer to the new location. Executed as step 8 of this pass.

### 2.12. §22 Account Tier / Auth / ToS Posture (New)

Consolidates TOS_COMPLIANCE verbatim-quote appendix + TOS_DISCUSSIONS Q3 + Cross-cutting A convergence: pure-headless dispatch, Max $100/$200 subscription for headless dogfood, training opt-out verified ON, `claude setup-token` auth path.

### 2.13. ToS Threaded Through Refinement Drops

Not a single big-bang §22 fold. Individual bullets on refinement drops:

- Cascade concurrency soft-cap enforcement mechanism (N=6 → configurable).
- API-key path for users without Max subscription.
- OpenAI-compat models via Agent SDK as alternate backend.
- Headless-only-for-Max-plans gating in user-facing compliance doc.
- User-side ToS compliance story in README + CONTRIBUTING.

---

## 3. Deferred Items (Flagged, Not This Pass)

### 3.1. TOS Pending Qs (Q1, Q2, Q4, Q5)

Folded as placeholders into refinement-drop bullets under §22 scope. Convergence in chat at refinement-drop planning time.

### 3.2. Kind-Hierarchy ASCII Contradiction in CLAUDE.md × 2

Both CLAUDE.md copies still show `plan-task` / `qa-check` / `task` as distinct kinds — contradicts the post-drop-2 "only project + drop + metadata.role" target. Defer fix to drop 2 prep (hierarchy refactor naturally touches this).

---

## 4. Cross-Reference Sweep Scope

Every file referencing `PLAN.md` updates to `PLAN.md`. Candidates (confirmed via grep before edits):

- `/CLAUDE.md` (bare-root)
- `/main/CLAUDE.md`
- `/main/STEWARD_ORCH_PROMPT.md`
- `/main/SLICE_1_ORCH_PROMPT.md` (updated then itself renamed)
- `/main/WIKI.md`
- `/main/LEDGER.md`
- `/main/REFINEMENTS.md`
- `/main/HYLLA_REFINEMENTS.md`
- `/main/HYLLA_FEEDBACK.md`
- `/main/HYLLA_WIKI.md`
- `/main/WIKI_CHANGELOG.md`
- `~/.claude/agents/go-*.md`
- `~/.claude/CLAUDE.md` (if it references)
- `/main/AGENTS.md` + `/AGENTS.md` (bare-root)
- `/main/CONTRIBUTING.md`
- `/main/README.md` (after TILLSYN_PURPOSE fold)

---

## 5. Memory File Updates (Prior-Session Carryover)

- Rename `feedback_use_tasks_until_slice_kind_lands.md` → `feedback_use_tasks_until_drop_kind_lands.md` + body pre-drop-2 rewrite.
- Rename `feedback_slice0_orchestrator_description_drift.md` → `feedback_drop0_orchestrator_description_drift.md` + body slice→drop sweep.
- Body slice→drop updates: `feedback_orchestrator_runs_ingest.md`, `project_tillsyn_cascade_vocabulary.md`, `feedback_orch_naming_all_caps_snake.md`.
- `MEMORY.md` index: fix slice→drop references + update renamed file pointers.
- `feedback_no_slice_terminology_anywhere.md` stays (saved last session, authoritative forbidden-forms table).

---

## 6. Execution Order

1. [x] Write this ledger (`main/temp.md`).
2. Fold content into `main/PLAN.md` per §2 decisions.
3. Fold TILLSYN_PURPOSE content into `main/README.md` + accuracy verification pass.
4. Rename `main/PLAN.md` → `main/PLAN.md`.
5. Cross-reference sweep (§4 above).
6. Rename `main/SLICE_1_ORCH_PROMPT.md` → `main/DROP_1_ORCH_PROMPT.md` + update body self-references.
7. Memory file updates (§5 above).
8. `.bare/` subdir retrofit (after git state verified clean).
9. Final QA report to dev.
10. [DEV] Delete folded sources (1.2, 1.3) + this ledger (1.4 — but only after convergence confirmed).

---

## 7. Risks + Unknowns

- **`.bare/` retrofit risk:** existing `main/` worktree has a `.git` file pointing into the bare repo's worktrees dir. After moving the bare repo internals to `.bare/`, every worktree's `.git` pointer must be rewritten to the new path. Get this wrong and the worktree detaches. Verification step before retrofit: inspect `main/.git` + bare-root `worktrees/` layout, plan the exact rewrite, dry-run `git status` inside the worktree after the move to confirm the linkage still resolves.
- **Plan size:** PLAN.md is 1624 lines. Folding 6 source files into it is a large edit surface. Self-QA after each section batch.
- **Cross-ref drift:** grep-based enumeration must be exhaustive — any missed reference leaves a dangling link after rename.
