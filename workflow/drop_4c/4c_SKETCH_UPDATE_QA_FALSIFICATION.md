# DROP 4C — SKETCH UPDATE QA FALSIFICATION REVIEW

**Targets:**

- `workflow/drop_4c/SKETCH.md` — Theme F (template ergonomics, claimed ~10–12 droplets) + Theme G (post-MVP marketplace evolution, captured for persistence).
- `PLAN.md` §19.10 — new "Marketplace evolution (post-Drop-4c)" bullet mirroring Theme G items.

**Reviewer mode:** filesystem-MD (read-only against MD; no code edits).

**Verdict:** **PASS-WITH-NITS — one CONFIRMED arithmetic / framing counterexample on Theme F droplet count, two PASS-WITH-NIT-class observations (F.3 catalog re-bake safety + F.4 git dependency hard-coupling) recommended for explicit handling at full-planning time, remaining attacks REFUTED.**

**Recommended action:** orchestrator amends SKETCH.md Theme F header (line 86) and the §"Approximate Size" line (line 56 / 154) to reflect the true F.1–F.6 sub-bullet sum (~18 droplets within Theme F alone; total drop size ~25–30+ droplets, not ~10–12). The two NITs (catalog re-bake mid-project, git shell-out hard dependency) get explicit Q-rows added to §"Open Questions To Resolve At Full-Planning Time" so they aren't re-litigated during builder dispatch. Theme G ↔ §19.10 parity check passes cleanly.

---

## 1. Methodology

Fifteen attack vectors applied, grouped per the spawn prompt:

- **YAGNI / scope creep:** A1 (F.4 marketplace size), A2 (3-template builtin separation), A3 (F.5 warn-only validation), A4 (G.6 sandboxing soundness).
- **Cross-reference:** A5 (Theme F droplet count math), A6 (Theme G ↔ PLAN §19.10 parity), A7 (`kind_capability.go:1002` no-op stub claim).
- **Logical:** A8 (`till.template(operation=set)` mid-project re-bake safety), A9 (auto-discovery fallback chain ordering), A10 (marketplace fetch git dependency), A11 (G.2 vector embeddings storage shape).
- **YAGNI on Theme G:** A12 (G.4 vs Drop 4b dispatcher overlap), A13 (TOML wire-format lock-in).
- **Pre-MVP rule alignment:** A14 (no-migration-logic vs catalog re-bake), A15 (closeout MD rollups timing vs Theme F landing).
- **Adversarial self-check:** A16 (additional surface raised in Section 0 QA Falsification pass — agent-binding spawn-gate fail mode).

Evidence sourced from `workflow/drop_4c/SKETCH.md` (lines 86–134 for Theme F, lines 123–134 for Theme G, lines 56 + 154 for size claims), `PLAN.md` §19.10 lines 1793–1800, `internal/app/kind_capability.go` lines 990–1004 (line 1002 stub verification), `internal/app/service.go:716` (caller verification), CLAUDE.md memory rules (`feedback_no_migration_logic_pre_mvp.md`, `feedback_no_closeout_md_pre_dogfood.md`, `feedback_opus_builders_pre_mvp.md`).

---

## 2. Counterexamples

### 2.1 CONFIRMED — Theme F droplet count claim "~10–12 droplets" arithmetically contradicts the F.1–F.6 sub-enumeration sum (~18 droplets) (A5)

**The most damaging counterexample.** This is a planning-arithmetic drift that, left unfixed, materially distorts dogfood-readiness sizing for Drop 4c and the Drop 5 readiness gate question (Q5). It is mechanical and unambiguous.

**Evidence:**

- SKETCH.md line 86 (Theme F header): `### Theme F — Template ergonomics (~10–12 droplets)`.
- SKETCH.md sub-bullet sizing as authored:
  - F.1 line 90: `Project-template auto-discovery (~3 droplets)`.
  - F.2 line 92: `Generic + Go + FE builtin separation (~4 droplets)`.
  - F.3 line 99: `till.template MCP tool (~3 droplets)`.
  - F.4 line 107: `Marketplace CLI (~5 droplets)`.
  - F.5 line 115: `Extended validation (~2 droplets)`.
  - F.6 line 121: `Cleanup of legacy KindTemplate stub (~1 droplet)`.
- Sum: 3 + 4 + 3 + 5 + 2 + 1 = **18 droplets within Theme F alone**.
- SKETCH.md §"Approximate Size" line 154 carries the same drift forward: `~10–12 droplets. Smaller than 4b.` But Theme F by itself is already ~18; adding Themes A (~4) + B (~2) + C (~3) + D (~1–2) + E (TBD) yields a drop total in the ~25–30+ droplet range, decidedly NOT smaller than 4b.

**Cases:**

- **Case 1 (sizing-driven readiness gate misread).** Q5 "Drop 5 readiness gate" decision (line 150) hinges on whether Theme A+B can land before Theme C+D — that is a small-fraction-of-12 question. If the real drop is ~28 droplets, the same fraction is a much larger absolute time-to-readiness, and the readiness gate decision shifts.
- **Case 2 (planner-dispatch sizing).** Line 176 says "likely 4 theme planners, one per Theme A–D." Theme F is bigger than A+B+C+D combined (18 vs ~10). Pretending Theme F slots into the "smaller than 4b" envelope means the planner-dispatch sizing in §"Open Tasks Before Full Planning" is wrong — Theme F demands its own multi-planner sub-decomposition (likely one planner per F.1–F.6 subtheme), which the current text does not mention.
- **Case 3 (anti-goal contradiction).** SKETCH.md §"Anti-Goals" line 180: `Not a "fix everything" drop. Drop 4c is bounded by the MVP-feature-complete gate. Items that aren't blocking Drop 5 dogfood readiness stay deferred to a later refinement.` Theme F's marketplace MVP CLI (F.4) is arguably NOT blocking Drop 5 dogfood (no adopter exists yet to consume the marketplace) — so its 5-droplet inclusion against the ~10–12 envelope was always going to spill, and the spillage is what produced the drift.

**Required fix (one of three options — orchestrator picks at full-planning time, dev sign-off):**

- **Option A — Honor the F.1–F.6 sum:** rewrite SKETCH.md line 86 to `~18 droplets`, line 154 to `~25–30 droplets total. Larger than 4b.` Update line 176 to acknowledge Theme F demands its own per-subtheme planner decomposition (F.1 / F.2 / F.3 / F.4 / F.5 / F.6 each get a planner if dispatched in parallel).
- **Option B — Move marketplace MVP (F.4) out of Drop 4c into Drop 5 or Drop 6 (post-dogfood):** keeps the ~10–12 Drop 4c envelope honest, defers the only post-MVP-flavored subtheme. After F.4 removal, F.1+F.2+F.3+F.5+F.6 = 13 droplets, still over the 12 ceiling but within rounding.
- **Option C — Hard split:** keep F.1, F.2 (without FE template — generic + Go only is enough for self-host), F.3, F.5, F.6 in Drop 4c (~10–11 droplets after the FE-template trim); spin Theme F.4 (marketplace MVP CLI) into a dedicated Drop 4d that runs post-Drop-4c-merge, before Drop 5. This matches the "narrow MVP-shaped drops" pattern the cascade has been pushing.

Severity: **planning-blocker** for full-planning time. Without one of these fixes, the planner-dispatch sizing + Q5 readiness gate question are answered against false sizing premises. **Not a builder-blocker** because builders aren't spawning yet — Drop 4c is sketch-only today. CONFIRMED.

---

### 2.2 REFUTED — F.6 "no-op pass-through stub" claim verified against `internal/app/kind_capability.go:1002` (A7)

**Attack:** SKETCH F.6 says `internal/app/kind_capability.go:1002` `mergeActionItemMetadataWithKindTemplate` is "a no-op pass-through stub kept 'during the transition.'" Verify by reading the function body — if it's anything but a one-line `return base, nil`, flag the misclaim.

**Evidence:**

- `internal/app/kind_capability.go:1002` function declaration:
  ```go
  func mergeActionItemMetadataWithKindTemplate(base domain.ActionItemMetadata, _ domain.KindDefinition) (domain.ActionItemMetadata, error) {
      return base, nil
  }
  ```
- The function body is exactly `return base, nil` (line 1003) — verified at the cited line number, not drifted.
- The godoc above (lines 995–1001) says verbatim: *"the merge is now a pass-through. Kept as a named function so call sites continue to compile during the transition; a future drop will fold it into the caller."*
- Caller verification: `/usr/bin/grep -n "mergeActionItemMetadataWithKindTemplate" internal/app/service.go` → `716:	mergedMetadata, err := mergeActionItemMetadataWithKindTemplate(in.Metadata, kindDef)` — also exactly as the sketch claims (line 121 of SKETCH.md says `service.go:716`).

The sketch's claim is precise and mechanically verifiable. Attack does not land.

---

### 2.3 REFUTED — Theme G ↔ PLAN §19.10 parity check passes (A6)

**Attack:** Confirm every G.1–G.7 item in SKETCH appears in PLAN. Confirm wording matches (or paraphrases acceptably). Flag any drift.

**Evidence:**

| SKETCH Theme G item                               | PLAN.md §19.10 line | Wording check                                                   |
| ------------------------------------------------- | ------------------- | --------------------------------------------------------------- |
| G.1 — TUI marketplace browser                     | 1794                | Verbatim match (Drop 4.5+ scope, FE/TUI track, etc.)            |
| G.2 — Vector search                               | 1795                | Verbatim match (embedding storage in marketplace repo)          |
| G.3 — User contribution flow                      | 1796                | Verbatim match (PR, CI validate, INDEX.toml, signed templates)  |
| G.4 — Live-runtime validation / dry-cascade sim   | 1797                | Verbatim match; "Drop 4c's static checks" replaces "Theme F.5"  |
| G.5 — Template inheritance / extends              | 1798                | Verbatim match                                                  |
| G.6 — Template-bound agent prompts                | 1799                | Verbatim match (sandboxing semantics + adopter trust)           |
| G.7 — Versioned template references on Project    | 1800                | Verbatim match (`tillsyn-templates@v1.4.0/go-cascade`)          |

The §19.10 framing paragraph (line 1793) correctly identifies SKETCH F.4 as "the marketplace foundation" landing in Drop 4c, and labels the seven follow-on items as "post-MVP refinements documented for persistence." The G.4 paraphrase ("Drop 4c's static checks" instead of "Theme F.5's static checks") is a context-correct substitution since §19.10 readers don't have Theme F's local sub-numbering. Acceptable paraphrase.

Attack does not land.

---

### 2.4 REFUTED — F.3 wire-format lock-in not actually a lock-in (A13)

**Attack:** Theme F.3 says TOML in, TOML out (MCP arg is TOML-as-string). Future proto/JSON-RPC clients might want native struct shape — TOML-string lock-in too strong?

**Evidence:**

- SKETCH.md line 105: `The MCP argument content is a string carrying TOML text verbatim. Server parses TOML, validates, persists TOML.`
- The TOML-string envelope is a transport choice, not a domain choice. The on-server validation path is `templates.Load(r io.Reader)` (line 90) — a `Reader` interface that accepts any source. Nothing in the MCP wire format prevents adding a parallel `content_struct` field later for clients that want to pass a typed payload.
- Counter-counter-argument: forcing TOML-string at the wire is actually the right call for symmetry with `cat template.toml | till template validate -` (CLI pipe) and for round-tripping marketplace fetches that arrive AS TOML files. Forcing struct-shape at the wire would force every CLI consumer to first parse-then-restructure the TOML they already have — net-negative ergonomics.

The "lock-in" framing assumes the MCP boundary must be the only typed layer. It isn't — the typed layer is already inside `templates.Load`. Attack does not land.

---

### 2.5 REFUTED — F.2 three-template builtin separation isn't premature (A2)

**Attack:** Is shipping THREE templates in the binary the right call, or is one truly-generic + one tillsyn-self-host enough? FE template might be premature if no FE adopter is real yet.

**Evidence:**

- SKETCH.md line 92–97: `default-generic.toml` is language-agnostic; `default-go.toml` is generic + Go bindings; `default-fe.toml` is generic + FE bindings.
- Project memory `feedback_decomp_small_parallel_plans.md` + `feedback_use_typed_agents.md` both document FE-typed agent variants (`fe-builder-agent`, `fe-qa-proof-agent`, `fe-qa-falsification-agent`, `fe-planning-agent`) as already-existing canonical agent types in `~/.claude/agents/`. The cascade architecture (CLAUDE.md "Agent Selection") explicitly dispatches FE variants for FE projects.
- `default-fe.toml` exists to bind those FE agents at template-bake time. Without it, an FE adopter has to author the entire FE binding set from scratch on day 1 — defeating the "adopter onboarding is fast" promise that motivates the marketplace foundation.
- The "no real FE adopter yet" objection cuts the wrong way: shipping `default-fe.toml` alongside MVP is exactly the on-ramp that makes the first FE adopter possible. Removing it forces the first FE adopter to become the de-facto FE template author too.

Attack does not land — F.2's three-template separation is on-ramp infrastructure, not premature generalization.

---

### 2.6 REFUTED — F.5 warn-only `validateAgentBindingFiles` is the right severity (A3 + A16)

**Attack:** `validateAgentBindingFiles` is warn-only and depends on local `~/.claude/agents/<name>.md` existence. What if a project bootstraps the cascade but doesn't have those agents installed yet? Warn-only is permissive but is the warning useful, or noise?

**Adversarial self-check (A16):** Does warn-only bypass the spawn-gate model — i.e., orchestrator spawns an agent-binding that doesn't exist on disk, agent fails opaquely later?

**Evidence:**

- SKETCH.md line 116–117: `Soft warning since the file might not be installed yet on this dev's machine; emit a finding without rejecting.`
- The right model: template validation happens at **template install / MCP `template(operation=validate)`** time (CLI / dev workflow). Agent dispatch happens at **dispatcher binding-resolution** time (post-Drop-4a runtime). Those are separate stages. The dispatcher already needs its own pre-flight check — `~/.claude/agents/<name>.md` either exists at dispatch time or the dispatch fails fast — and that check IS the spawn-gate model.
- Promoting `validateAgentBindingFiles` to a hard error at template-install time would block legitimate workflows: dev clones a fresh tillsyn project on a new machine, agents not yet installed, template install fails — user can't even read the template they just installed. Warn-only correctly defers the existence check to dispatch-time without losing the signal at install-time.
- The "noise vs useful" question: a warn-only finding emitted to the dev's terminal (or surfaced in `till.template(operation=validate)` MCP response) tells the dev *exactly* which agents they need to install. That is high-signal. The signal is consumed at install-time, where the dev is in position to act on it; the failure-fast at dispatch-time is consumed at run-time, where it surfaces to the orchestrator instead.

Both A3 and A16 fail. Warn-only is the right severity.

---

### 2.7 REFUTED — G.6 template-bound agent prompts not "fundamentally unsound" (A4)

**Attack:** G.6 sandboxing semantics + adopter trust. Is this even achievable post-MVP, or fundamentally unsound (running arbitrary remote-template-supplied prompts in your agent context = supply-chain risk)?

**Evidence:**

- SKETCH.md line 132: G.6 explicitly carries the qualifier *"Requires sandboxing semantics + adopter trust."* The sketch is not claiming G.6 is achievable today; it is captured as a known-hard post-MVP item with the open hard-problem flagged inline.
- "Fundamentally unsound" is too strong. Adopter trust models exist for analogous supply-chain surfaces (npm package install scripts, GitHub Actions reusable workflows, Helm charts, Homebrew formulae). All carry supply-chain risk; all ship with documented trust boundaries (signed releases, manifest pinning, CI sandbox semantics, manual review gates). G.6 inheriting those constraints isn't novel risk — it's the same risk applied to a new artifact class.
- G.6 explicitly says "today agent prompt files are global" — meaning the canonical state is global agents under `~/.claude/agents/`. Template-bound prompts are an opt-in escalation, not the default. An adopter who never opts in faces zero new risk.

The attack is a rhetorical objection to capturing G.6 at all. The sketch's "captured for persistence" framing is exactly the right disposition — capture the design, flag the open problem, defer the work. Attack does not land.

---

### 2.8 REFUTED — G.4 still relevant post-Drop-4b (A12)

**Attack:** G.1–G.7 are post-MVP captured for persistence. Are any already obsolete given Drop 4a/4b/4c scope shifts? Specifically: does G.4 "live-runtime validation / dry-cascade simulation" remain relevant if Drop 4b's gate runner already does template-driven dispatch validation at every state transition?

**Evidence:**

- Drop 4b's gate runner validates **at dispatch time**, on real action items, with real state transitions. If a template misconfiguration only manifests at depth ≥ 3 of the cascade (e.g. a rule that orphans `closeout` from `plan` reachability), the gate runner detects it only when the misconfigured cascade actually runs — by which point real builders have already been dispatched against the bad template.
- G.4 dry-cascade simulation is **at validate time**, against synthetic action-item trees, before any real dispatch. The two checks operate at different stages and different inputs. They are complementary, not redundant.
- Concrete example G.4 catches that 4b's runtime gate doesn't: a template defines `plan` with no `child_rules` for `plan-qa-proof`. Static F.5 `validateRequiredChildRules` catches this if it's enumerated. But a more subtle case — a `plan-qa-proof` `child_rules` entry whose target kind is misspelled `plain-qa-proof` — passes F.5 (the entry exists) and only fails when a real plan tries to spawn its required QA child. Dry-cascade simulation walks the full graph and catches the spelling drift before any builder fires.

G.4 retains independent value. Attack does not land.

---

## 3. NIT-Level Observations

### 3.1 NIT — F.3 `till.template(operation=set)` mid-project re-bake safety not addressed (A8)

**Not a counterexample — but a hard-question that deserves an explicit Q-row before full-planning.**

**Observation:**

- SKETCH.md line 102: `till.template(operation=set, project_id=..., content=<toml-string>)` — validate + install + re-bake catalog.
- Re-baking a catalog mid-project changes the cascade vocabulary. If action items already exist with kinds that the new catalog removes, those action items reference a now-missing kind. Behavior unspecified.
- Possible safety preconditions to consider at full-planning:
  - **No-in-flight precondition:** `set` rejects if any action item is `in_progress` against the project. Conservative; matches the no-migration-logic posture.
  - **Kind-superset precondition:** `set` rejects if the new catalog removes any kind currently in use by extant action items in the project. More permissive; allows additive changes mid-flight.
  - **Force-with-supersede:** `set --force` accepts catalog-removal but auto-supersedes affected action items. Most permissive; introduces migration-flavored behavior the no-migration-logic rule explicitly forbids pre-MVP.
- Recommendation: full-planning time picks one (likely the no-in-flight precondition for pre-MVP simplicity); add a Q-row to §"Open Questions To Resolve At Full-Planning Time."

**Severity:** PASS-WITH-NIT. Not a sketch-blocker (sketch is allowed to defer detail), but a should-be-explicit-before-builder-dispatch concern.

### 3.2 NIT — F.4 marketplace fetch via git shell-out hard-couples to git-on-PATH (A10)

**Observation:**

- SKETCH.md line 108: `internal/templates/marketplace.go — git-shell-out wrapper. git clone --depth 1 on first fetch, git pull on update.`
- Tillsyn binary deployed in a CI container or minimal Docker image without `git` on PATH (e.g. distroless) cannot fetch marketplace templates. Hard dependency.
- Two alternatives at full-planning time:
  - **`go-git`** (`github.com/go-git/go-git/v5`) — pure-Go git implementation. Heavier binary footprint (~3–5MB added) but zero external dependency. Trade-off acceptable for portability.
  - **HTTPS-only fetch** — marketplace repo serves a tarball via GitHub's `archive` endpoint; tillsyn binary uses `net/http` to fetch + `archive/tar` to extract. Zero git dependency, smallest binary footprint, but loses commit-log delta UX (line 110: "Show commit-log delta on update") unless paired with the GitHub API for log fetch.
- Recommendation: full-planning time evaluates the binary-footprint vs portability trade-off; add a Q-row.

**Severity:** PASS-WITH-NIT. Sketch is allowed to defer transport-implementation detail.

### 3.3 NIT — F.1 auto-discovery fallback chain ordering merits explicit dev sign-off (A9)

**Observation:**

- SKETCH.md line 90: `<project.RepoBareRoot>/.tillsyn/template.toml` first, fall back to `<project.RepoPrimaryWorktree>/.tillsyn/template.toml`, fall back to embedded `default.toml`.
- Bare-root precedence is unconventional. The bare-root convention in this project (CLAUDE.md "Bare-Root and Worktree Discipline") treats the bare repo as orchestration root, not code-bearing checkout. `.tillsyn/` directories are typically authored in the visible primary-worktree (where `mage`, `git status`, etc. naturally land), not in the bare root.
- However, the SKETCH ordering may be intentional: the bare-root `.tillsyn/template.toml` is the cross-worktree shared template (overrides apply to every worktree off this bare root), while the primary-worktree `.tillsyn/template.toml` is the worktree-local override. Bare-root-first lets a single shared template ship the canonical cascade vocabulary; primary-worktree-first would let lane-specific tweaks accidentally override the shared vocabulary.
- Dev sign-off needed at full-planning: confirm the bare-root-first ordering is intentional and matches how dev intends adopters to organize their `.tillsyn/` files. The reverse ordering (primary-worktree-first) is a defensible alternative.

**Severity:** PASS-WITH-NIT. Sketch is allowed to defer ordering rationale.

### 3.4 NIT — Pre-MVP rule alignment ambiguous on Theme F landing window (A14 + A15)

**Observation:**

- SKETCH.md §"Pre-MVP Rules (carried forward)" line 137–142: lists `No closeout MD rollups` and `Opus builders` as carried forward.
- §"Open Questions" Q1 line 146: explicitly flags `feedback_no_closeout_md_pre_dogfood.md` as a candidate transition during Drop 4c.
- The catalog re-bake under F.3 (`till.template(operation=set)` re-bakes the catalog, persists new TOML) rewrites Project's `KindCatalog` JSON-encoded field. Memory rule `feedback_no_migration_logic_pre_mvp.md` says "no migration code in Go, no till migrate CLI, no one-shot SQL scripts. Dev deletes ~/.tillsyn/tillsyn.db on schema/state-vocab change."
- Question: is rewriting `KindCatalog` JSON in-place "migration logic in disguise" (forbidden) or "normal mutation" (allowed)?
- **Likely answer:** allowed as normal mutation. The no-migration-logic rule targets *schema migrations* (changing column shape, renaming tables, transforming row data across versions) — `KindCatalog` JSON is a single field whose shape is owned by the application; rewriting it is a regular UPDATE, not a migration. The rule's spirit is "don't write Go that migrates v1 data to v2 data automatically; let dev fresh-DB."
- However, A15's harder question lands: if Drop 4c is the moment when "no closeout MD rollups" retires (Q1), then Drop 4c's own closeout becomes the first dogfood-mode closeout. Is Theme F doing too much *during* the pre-MVP→dogfood transition? Maybe Theme F should land before the rule transition (clean pre-MVP closing) or after (clean dogfood opening), not straddle.

**Recommendation:** Q1 in SKETCH §"Open Questions" already captures this. NIT only because the answer affects Theme F sequencing — adding a cross-reference from Q1 to Theme F.3 + the §"Pre-MVP Rules" line would help full-planning resolve it.

**Severity:** PASS-WITH-NIT. Already partly captured in Q1; would benefit from explicit cross-reference.

### 3.5 NIT — G.2 vector embeddings storage shape merits its own object-storage Q (A11)

**Observation:**

- SKETCH.md line 128: G.2 stores `<name>.embedding.json` per template in the marketplace repo.
- Each embedding is typically 1.5–3KB JSON (1536 × float32 / float64 + metadata). With ~30–50 templates and per-tag embeddings, repo size grows multi-MB over time. Acceptable today; unbounded scaling concern for "one embedding per template tag" if tags multiply.
- Alternatives: GitHub Releases attachments, dedicated CDN bucket, or content-addressable storage with manifest. G.2 captures the simple-shipping default; a future Q could revisit if repo size becomes a real-world concern.

**Severity:** PASS-WITH-NIT. G.2 is post-MVP captured-for-persistence; no immediate action required. Worth noting at full-planning time only if Drop 5 dogfood data shows repo-size pressure.

---

## 4. Most Damaging Counterexample

**§2.1 (Theme F droplet count "~10–12" arithmetically contradicts F.1–F.6 sub-enumeration sum ~18).** Mechanical, unambiguous, and load-bearing for the Q5 readiness-gate decision and the planner-dispatch sizing in §"Open Tasks Before Full Planning." All other attacks either REFUTE cleanly or reduce to NITs that the sketch is already allowed to defer to full-planning. The droplet-count drift is the only one that breaks the sketch's own internal accounting.

Fix is single-line-textual at sketch-revision time, three-line-textual at full-planning time. Cost-to-correct is trivial; cost-to-leave-uncorrected is downstream miscalibration of every sizing decision Drop 4c makes.

---

## 5. Verdict Summary

**PASS-WITH-NITS.**

- One CONFIRMED arithmetic counterexample (§2.1) — Theme F droplet-count math drift; revise SKETCH header + size line + planner-dispatch hint.
- Two attacks that produce CONFIRMED-deferred-Q-rows for full-planning (§3.1 catalog re-bake mid-project, §3.2 git shell-out hard dependency).
- Three attacks that produce NIT observations (§3.3 fallback ordering, §3.4 pre-MVP rule alignment cross-ref, §3.5 G.2 storage shape).
- All other attacks REFUTED with evidence (§2.2–§2.8).

**Recommended sketch revision (single-line edits):**

1. SKETCH.md line 86: `### Theme F — Template ergonomics (~18 droplets, biggest theme in this drop)`.
2. SKETCH.md line 154: `~25–30 droplets total. Larger than 4b, driven by Theme F. Most items are 1–3 file edits each (audit-finding fixes are typically narrow). Full planning at post-4b-merge time will refine the count + the Theme E residue list.`
3. SKETCH.md line 176: change `(likely 4 theme planners, one per Theme A–D)` to `(likely 4 theme planners for Themes A–D + 6 sub-theme planners for F.1–F.6, parallel-dispatched per the small-parallel-planners decomp pattern)`.
4. SKETCH.md §"Open Questions To Resolve At Full-Planning Time" — add Q6 (catalog re-bake mid-project safety) + Q7 (git shell-out vs go-git vs HTTPS-tarball) + Q8 (auto-discovery fallback ordering bare-root-first vs primary-first dev sign-off).

**No revisions required to PLAN.md §19.10** — Theme G ↔ §19.10 parity is clean.

---

## TL;DR

- **T1**: Methodology covers 16 attack vectors (15 spawn-prompted + 1 adversarial self-check) over Theme F, Theme G, and PLAN §19.10; evidence drawn from SKETCH.md, PLAN.md, kind_capability.go, service.go, and CLAUDE.md memory rules.
- **T2**: Eight attacks land as REFUTED (F.6 stub claim verified at line 1002 + caller at service.go:716; Theme G ↔ §19.10 parity exact; TOML wire-format envelope is transport not lock-in; three-template separation is on-ramp infra not premature; warn-only F.5 is the right severity at the right stage; G.6 supply-chain risk is captured-and-flagged not unsound; G.4 retains independent value vs Drop 4b runtime gates) — only §2.1 droplet-count math drift is CONFIRMED.
- **T3**: Five NIT-level observations route to full-planning time as explicit Q-rows (catalog re-bake mid-project safety, git shell-out hard dependency, fallback ordering rationale, pre-MVP rule alignment cross-ref, G.2 storage scaling).
- **T4**: Most damaging counterexample is §2.1 — Theme F header claims ~10–12 droplets but F.1–F.6 sums to ~18; fix is single-line-textual but load-bearing for Q5 readiness-gate decision and planner-dispatch sizing.
- **T5**: Verdict PASS-WITH-NITS — orchestrator amends SKETCH.md lines 86, 154, 176 and adds three Q-rows to §"Open Questions"; no revision required to PLAN.md §19.10.

---

## Hylla Feedback

N/A — task touched non-Go files only. The single Go-file verification (`internal/app/kind_capability.go:1002` + caller at `internal/app/service.go:716`, attack A7) was a precise line-number lookup, not a Hylla query workload — direct `Read` + targeted grep was the correct first-tool choice for verifying a known cited line. No Hylla miss to report.
