# Drop 4c F.7.18.5 — Builder QA Falsification

## Round 1

**Scope.** Read-only adversarial review of F.7.18.5 (default-template seeds for
`[agent_bindings.<kind>.context]`). Surfaces: `internal/templates/builtin/default.toml`
(+106 lines), `internal/templates/embed_test.go` (+240 lines), the F.7.18 plan
REVISIONS section (REV-4 specifically), and adjacent F.7.18.x specifications.
No Hylla calls per spawn prompt; LSP not refreshed for this read-only pass.

**Method.** Walk the 13 attack vectors A1–A13. Each gets verdict +
evidence-cited reasoning. Counterexamples reproduced with file:line citations.

---

### A1 — REV-4 contract: any way QA gets diff back?

**Verdict: REFUTED.**

**Attack:** A future template editor accidentally sets
`parent_git_diff = true` on a QA binding — does the test fire?

**Evidence:** `internal/templates/embed_test.go:479-501`
(`TestDefaultTemplateQABindingsRejectParentGitDiff`).

```
qaKinds := []domain.Kind{
    domain.KindBuildQAProof,
    domain.KindBuildQAFalsification,
    domain.KindPlanQAProof,
    domain.KindPlanQAFalsification,
}
for _, kind := range qaKinds {
    t.Run(string(kind), func(t *testing.T) {
        ...
        if binding.Context.ParentGitDiff {
            t.Fatalf("AgentBindings[%q].Context.ParentGitDiff = true; want false (REV-4 ...)")
        }
    })
}
```

**Reproduction.** Manual mutation: insert `parent_git_diff = true` into
`[agent_bindings.build-qa-proof.context]` at default.toml:535-540. The test
above iterates all four QA kinds and asserts `ParentGitDiff == false`; the
mutation flips that bool to `true`, the assertion fires, the subtest fails
under the precise sub-name `build-qa-proof` (or whichever kind the editor
touched). Failure name pinpoints the exact regression.

**Bonus coverage:** the positive companion test
(`TestDefaultTemplateBuildContextSeedsParentGitDiff`,
`embed_test.go:456-467`) asserts the build binding HAS the field — guards
against the symmetric regression "someone deleted `parent_git_diff = true`
from the build binding."

REV-4 contract is enforced by symmetric positive + negative assertions.

---

### A2 — Default seed shape vs F.7.18.4 (engine) defaults

**Verdict: PASS-WITH-NIT.**

**Attack:** Engine defaults per F.7.18 plan: 50KB / 500ms / 2s / 200KB.
Builder's seed: `max_chars = 50000` + `max_rule_duration = "500ms"`. Same
numeric values. Coincidence or coupling? If F.7.18.3 changes default
constants but seeds don't, default template diverges from documented
behavior.

**Evidence:** F.7.18 plan acceptance criteria F.7.18.3:230-234 explicitly
documents the engine-substitution defaults:

```
- binding.Context.MaxChars == 0 → use 50000.
- binding.Context.MaxRuleDuration == 0 → use 500ms.
```

The seed values match the documented engine defaults intentionally, NOT
coincidentally. The plan's intent (lines 354-394) is "explicit defaults; matches
engine substitution." Comment block above `[tillsyn]` in the plan
(lines 354-356) calls this out: "explicit default; matches engine substitution."

**However:** there is no automated test cross-checking that the embedded
constant in the engine package (`internal/app/dispatcher/context/`) matches
the seed in `default.toml`. If F.7.18.3 changes the default to 60_000 chars
without re-seeding `default.toml`, the implicit-default semantics drift.

**Mitigation** (NIT, not blocker): the seeded values are explicit, so they
override the engine default — adopters who load the default template get
the seeded number regardless of engine drift. The drift is semantic
(engine-default-doc vs default-template-seed disagree), not behavioral
(default template's seed wins per resolution rules in F.7.18.3:230-234).
Out-of-scope here because F.7.18.3 hasn't landed yet and the engine package
doesn't exist. **Suggest filing as a refinement** for F.7.18.3 acceptance:
"add a test asserting engine constants `defaultMaxChars` /
`defaultMaxRuleDuration` equal the values seeded in default.toml."

Does NOT block F.7.18.5; routes forward as Unknown / refinement seed.

---

### A3 — `delivery = "file"` rationale

**Verdict: REFUTED.**

**Attack:** All six bindings ship with file mode. Builder didn't justify in
the worklog. Is there a binding where `inline` is more appropriate?

**Evidence:** F.7.18.5 plan acceptance criteria explicitly seeds
`delivery = "file"` for all six in-scope bindings (lines 363, 369, 378,
381, 387). The plan's reasoning (cited in
`schema.go:597-606` ContextRules.Delivery doc-comment): file-mode renders
context to `<bundle>/context/<rule>.md` for on-demand `Read`, while inline
appends to `system-append.md` (consumed at every turn).

**Per-binding analysis:**
- `plan` — planner agent: parent + ancestors-walk could be large; file mode
  preserves system-append.md token budget. Correct.
- `build` — builder: parent + git-diff are large. File mode required.
- `build-qa-proof` / `build-qa-falsification` — QA reads parent + ancestors,
  may not need everything every turn. File mode correct.
- `plan-qa-proof` / `plan-qa-falsification` — same as build-QA.

No binding's spawn loop benefits from inline-by-default — the system-append.md
token cost is cumulative across turns, while file-mode is `Read`-on-demand.
Default to file is the conservative correct choice.

**Comment-block citation:** `default.toml:444-445, 532-534, 564-566`
references "Both modes are first-class" — adopters MAY override per project,
which is the correct flexibility framing.

No counterexample produced.

---

### A4 — `ancestors_by_kind = ["plan"]` semantics

**Verdict: REFUTED.**

**Attack:** First-match. For `kind=plan` binding (planner agent), the parent
IS-A plan in nested-plan trees. Walking from a plan up to the first plan
ancestor returns the immediate parent. Is that the intent, or did adopter
want top-most plan?

**Evidence:** `schema.go:581-585` (`AncestorsByKind` doc-comment):

```
AncestorsByKind walks UP the parent chain and captures the FIRST
ancestor whose Kind matches an entry in this slice. The walk respects
declaration order: ["plan", "build"] returns the nearest plan ancestor
when one exists, falling back to the nearest build ancestor otherwise.
```

First-match-by-declaration-order is the locked semantic.

For a nested `plan → plan → plan`, the planner spawned at the deepest plan
sees its IMMEDIATE plan parent (one level up), NOT the drop-level plan. This
is the intent: each planner-pass needs its enclosing-plan context, not the
whole drop's tree. Top-most-plan would dilute context.

The plan's F.7.18.5 acceptance (lines 382-387) seeds the planner with
`ancestors_by_kind = ["plan"]` AND explicitly cites this is "fix-planners +
tree-pruners are an adopter opt-in" — the default seed is for the simple
case (sub-plan sees enclosing plan), and adopters who want different walks
declare them.

The current rule renderer hasn't shipped (F.7.18.3 territory), so the
behavioral check is paper-only. But the SCHEMA contract (first-match) is
locked in `schema.go:581-585`. Default seed is consistent with locked
contract.

No counterexample.

---

### A5 — `build-qa-proof` + `build-qa-falsification` symmetry

**Verdict: REFUTED.**

**Attack:** Builder shipped them with IDENTICAL `[context]` shapes. Should
they differ in any field? E.g., would falsification benefit from
`descendants_by_kind` to attack child structure?

**Evidence:**
- F.7.18.5 plan acceptance criteria line 373: "identical to `build-qa-proof`
  per REV-4."
- F.7.18 SKETCH (cited in plan) frames build-qa-proof + build-qa-falsification
  as asymmetric ROLES (proof verifies completeness; falsification attacks),
  but they consume the SAME upstream context — both review the build's
  output. Asymmetry is in the agent's prompt + reasoning, not in the
  pre-staged context.
- Falsification with `descendants_by_kind` would be paradoxical at this
  layer: a build droplet's descendants are its build-qa-proof +
  build-qa-falsification self-referentially — pre-staging that would create
  a self-loop in the context graph. The aggregator is per-binding-scope (not
  bidirectional), so a build-qa-falsification looking at its own build's
  build-qa-proof sibling is the SiblingsByKind path, NOT DescendantsByKind.

**Note:** the F.7.18.5 plan flagged
(line 367-368) `siblings_by_kind = []` as a future enrichment for QA
bindings (sibling-builder-worklog wiring) but deferred it because metadata
plumbing isn't ready. This is correctly omitted from the seed; adopters can
add it later without schema changes.

Identical `[context]` for the proof + falsification twin is the right call
— asymmetry lives in the agent prompts, not the context.

No counterexample.

---

### A6 — `plan-qa-*` bindings missing `parent_git_diff` is correct + tested

**Verdict: REFUTED.**

**Attack:** Plan QA reviews planning artifacts, not code. Spec confirms no
diff needed. But verify the test asserts this distinction (not just defaults
to false because the field is absent).

**Evidence:** `embed_test.go:483-488` lists `qaKinds` as ALL FOUR QA kinds:

```
qaKinds := []domain.Kind{
    domain.KindBuildQAProof,
    domain.KindBuildQAFalsification,
    domain.KindPlanQAProof,
    domain.KindPlanQAFalsification,
}
```

The negative-assertion test runs against all four — including the two
plan-QA kinds. The test asserts `Context.ParentGitDiff == false`. Same
assertion fires whether the field is absent (Go zero value) OR explicitly
set to `false` in TOML — both decode to `false`. So a future regression that
adds `parent_git_diff = true` to ANY QA binding (including plan-QA) is
caught.

The plan F.7.18.5 line 374-380 + 380 specifies plan-QA bindings have NO
`parent_git_diff` line. The seeded TOML at `default.toml:474-479,
501-506` confirms absence. Test enforces.

The "asserts the distinction" question reduces to: does the test fire on
the right semantic? Yes — the assertion is `== false`, which is the REV-4
contract regardless of why the field is false (absence or explicit-false).

No counterexample.

---

### A7 — Scope creep into other binding fields

**Verdict: REFUTED.**

**Attack:** Did builder accidentally edit `agent_name`, `model`,
`max_tries`, etc. on existing bindings while adding `[context]` blocks?

**Evidence:** `git diff --stat HEAD -- internal/templates/builtin/default.toml`
shows +106 lines, 0 deletions. The full diff (read above) shows ONLY
additive blocks: every `+` line is either a comment line or a new
`[agent_bindings.<kind>.context]` block. Zero `-` lines. Zero modifications
to flat fields (`agent_name`, `model`, `effort`, `tools`, `max_tries`,
`max_budget_usd`, `max_turns`, `auto_push`, `commit_agent`,
`blocked_retries`, `blocked_retry_cooldown`).

The diff layout is purely insert-after-flat-fields per binding. Existing
bindings are byte-identical pre/post.

No counterexample.

---

### A8 — TOML ordering / structure

**Verdict: REFUTED.**

**Attack:** `[context]` is a sub-table of `[agent_bindings.<kind>]`. Is the
placement consistent with TOML conventions?

**Evidence:** TOML spec requires that all key-value pairs in a parent table
appear BEFORE any sub-table (`[agent_bindings.<kind>.context]`) — once you
open a sub-table, you cannot return to the parent's flat scope without an
explicit re-open of the parent header.

Reading `default.toml:419-452` for the build binding:

```
[agent_bindings.build]                       # line 419
agent_name = "go-builder-agent"              # ... flat fields
...
blocked_retry_cooldown = "30s"               # line 430 — last flat field

# comment block                              # 432-445
[agent_bindings.build.context]               # line 446 — sub-table opens
parent = true
parent_git_diff = true
...
```

All flat fields of `[agent_bindings.build]` precede the sub-table. Pattern
repeated for the other 5 bindings. TOML-correct ordering.

The pelletier/go-toml v2 strict-decode chain (per
`internal/templates/load.go`) would reject a malformed ordering at load
time, and the `TestDefaultTemplateLoadsCleanly` test at
`embed_test.go:28-38` would fail. It passes (per worklog: 355 tests green
in `internal/templates`), so ordering is decoder-acceptable.

No counterexample.

---

### A9 — `embed_test.go` test naming

**Verdict: PASS-WITH-NIT.**

**Attack:** Are the eight test names descriptive enough?

**Evidence:** Reading the eight names:

1. `TestDefaultTemplateBuildContextSeedsParentGitDiff` — clear: positive
   assertion that build seeds the field.
2. `TestDefaultTemplateQABindingsRejectParentGitDiff` — clear: REV-4
   regression guard.
3. `TestDefaultTemplateContextSeedsAncestorsByKind` — clear.
4. `TestDefaultTemplateContextSeedsDelivery` — clear.
5. `TestDefaultTemplateContextSeedsCaps` — clear-ish; "Caps" covers MaxChars
   + MaxRuleDuration. The body documents both, so a reader scanning failures
   sees `Caps/<kind>: MaxChars = ...` which clarifies. Acceptable but could
   split into `TestDefaultTemplateContextSeedsMaxChars` +
   `TestDefaultTemplateContextSeedsMaxRuleDuration` for finer-grained
   regressions. Style nit, not blocker.
6. `TestDefaultTemplateContextSeedsParentTrue` — clear.
7. `TestDefaultTemplateNonContextSeededKindsHaveZeroContext` — slightly
   verbose but precise.
8. `TestDefaultTemplatePlanContextHasNoDescendants` — clear.

Names follow the project's existing `TestDefaultTemplate<Aspect>` convention
(see e.g. `TestDefaultTemplateLoadsCleanly`,
`TestDefaultTemplateCoversAllTwelveKinds`). Style fits.

NIT only: A5 ("Caps") merges two distinct field assertions into one test.
If MaxChars regresses, the failure name reads `Caps/<kind>` and the body
distinguishes; debugger has to read the failure body. Splitting would
trade test count for failure-name precision. Either is defensible.

Does NOT block.

---

### A10 — Forward-compat with F.7.16 (gate-list expansion)

**Verdict: REFUTED.**

**Attack:** F.7.16 will edit `default.toml` `[gates.build]` later. Does
F.7.18.5's edits create merge conflict surface?

**Evidence:** `default.toml` structure (read in evidence-gather):
- `schema_version` at line 36.
- `[kinds.*]` blocks lines 56-186.
- `[[child_rules]]` lines 204-265.
- `[[steward_seeds]]` lines 295-317.
- `[gates]` line 350-351 (`build = ["mage_ci"]`).
- `[agent_bindings.*]` lines 377-642.

F.7.18.5's edits land EXCLUSIVELY in `[agent_bindings.<kind>]` blocks
(lines 387-572 region) — six new sub-tables added.

F.7.16 (gate-list expansion) will edit `[gates]` at line 350-351, expanding
to `["mage_ci", "commit", "push"]` per default.toml:336-339 comment. That
edit is at line 350-351, ~30 lines BEFORE the first `[agent_bindings.*]`
block.

Three-way-merge surface: F.7.18.5 + F.7.16 touch DIFFERENT line ranges with
~30+ lines of context between them. Standard `git merge` resolves
trivially. No content-level conflict because they write distinct keys in
distinct top-level tables.

The plan F.7.18.5 acceptance line 344 explicitly declares:
`blocked_by F.7.16.<final-droplet-id>` — the planner anticipated the
overlap and serialized the droplets via `blocked_by`. The serialization is
defensive; even without it, three-way merge would resolve.

No counterexample.

---

### A11 — `[tillsyn]` block missing

**Verdict: REFUTED (deferral is acceptable).**

**Attack:** Builder noted `[tillsyn]` (max_context_bundle_chars /
max_aggregator_duration) was deferred per spawn prompt. Is this acceptable?

**Evidence:**

1. Worklog `4c_F7_18_5_BUILDER_WORKLOG.md:23-30` explicitly documents the
   deferral and cites the spawn prompt narrowed scope: "the six
   `[agent_bindings.<kind>.context]` blocks."

2. F.7.18 plan REV-3 (lines 522-524) declares the `Tillsyn` struct + its
   initial fields are F.7.18.2's territory (Schema-3); F.7-CORE F.7.1 +
   F.7.6 EXTEND it with their named fields. Multi-droplet ownership of the
   `Tillsyn` struct = staged extensions.

3. Engine-time substitution (per F.7.18.3:230-234): a missing `[tillsyn]`
   block resolves to engine defaults (200KB / 2s) at `Resolve`-time, NOT a
   load failure. So the default template loads cleanly without the block,
   and adopters get the correct runtime defaults.

4. `embed_test.go` does NOT yet test the `[tillsyn]` block. This is
   correct: the `Tillsyn` struct doesn't exist on `Template` yet
   (F.7.18.2 hasn't landed in this droplet's view; per the plan `blocked_by`
   chain F.7.18.5 depends on F.7.18.4 which depends on F.7.18.3 which
   depends on F.7.18.2 — by the time F.7.18.5 lands the struct is there).

The risk: if F.7.18.5 lands in `default.toml` AND F.7.18.2 has landed
introducing the `Tillsyn` struct without seeding `[tillsyn]` defaults in
the default template, adopters get implicit engine defaults rather than
explicit seeded values. This is acceptable: the engine-default substitution
is deterministic, and explicit seeds are a future enhancement (not a
correctness bug).

The plan's F.7.18.5 line 354-356 originally specified seeding the
`[tillsyn]` block as part of THIS droplet's scope. The spawn prompt
narrowed scope to "the six bindings only" — the deferral is by orchestrator
direction, not builder oversight. Acceptable.

**Suggest filing as refinement:** "future droplet seeds explicit
`[tillsyn]` block in default.toml with values matching engine defaults."
Routes as Unknown / refinement seed.

Does NOT block F.7.18.5.

---

### A12 — No-commit per REV-13

**Verdict: REFUTED.**

**Attack:** Verify builder didn't auto-commit.

**Evidence:**
1. `git status --porcelain` from session start (system-loaded `gitStatus`)
   shows the two changed files (`default.toml` + `embed_test.go`) plus
   untracked workflow MDs — no commits added by builder.
2. Worklog line 132: "DID NOT commit. Worklog written; orchestrator drives
   commits."
3. Suggested commit message provided at lines 144-149 of the worklog for
   the orchestrator to use.

REV-13 honored. No counterexample.

---

### A13 — Memory rule conflicts

**Verdict: REFUTED.**

**Attack:** `feedback_no_migration_logic_pre_mvp.md` — TOML edits, no SQLite;
`feedback_subagents_short_contexts.md` — focused single-file work.

**Evidence:**
- `feedback_no_migration_logic_pre_mvp.md`: prohibits Go migration code, SQL
  migration scripts, `till migrate` CLI. F.7.18.5 edits a TOML file +
  test file. No DB schema touched, no SQL emitted, no CLI added. PASS.
- `feedback_subagents_short_contexts.md`: focused single-file work + spawn
  scope. F.7.18.5 narrowed to two files (TOML + Go test). PASS.
- `feedback_opus_builders_pre_mvp.md`: builders run opus. The new context
  block on `[agent_bindings.build]` doesn't touch the `model = "opus"` flat
  field. PASS.
- `feedback_no_closeout_md_pre_dogfood.md`: skip CLOSEOUT/LEDGER MDs.
  F.7.18.5 doesn't write any of those. PASS.
- `feedback_never_remove_workflow_files.md`: never delete workflow files.
  F.7.18.5 ADDS workflow MDs (worklog, QA proof, QA falsification). PASS.

No memory-rule conflicts.

---

## Findings

- F1 (REFUTED) — A1: REV-4 negative-assertion test (`TestDefaultTemplateQABindingsRejectParentGitDiff`) at `embed_test.go:479-501` correctly fires on any future regression that adds `parent_git_diff = true` to any QA binding. Symmetric positive test on build at `embed_test.go:456-467`.

- F2 (NIT) — A2: No automated cross-check between engine-default constants (in F.7.18.3's package, not yet existing) and seeded `default.toml` values. Drift risk if F.7.18.3 lands new defaults. Routes as refinement for F.7.18.3 acceptance criteria, NOT a blocker on F.7.18.5.

- F3 (REFUTED) — A3: `delivery = "file"` is the conservative correct default for all six bindings (system-append.md token cost is cumulative; file-mode is `Read`-on-demand). Adopters override per project per FLEXIBLE-not-REQUIRED framing.

- F4 (REFUTED) — A4: First-match `ancestors_by_kind = ["plan"]` returns immediate plan parent in nested-plan trees, which is the locked semantic per `schema.go:581-585` and the intent per F.7.18.5 plan.

- F5 (REFUTED) — A5: Identical `[context]` for build-qa-proof + build-qa-falsification is correct; asymmetry lives in agent prompts, not pre-staged context. `descendants_by_kind` for falsification would create a self-loop.

- F6 (REFUTED) — A6: All four QA kinds (including plan-QA) covered by the negative-assertion test. Field absent in TOML decodes to Go zero value `false`; assertion fires on either explicit-true or accidentally-added-true mutations.

- F7 (REFUTED) — A7: `git diff` shows zero modifications to flat fields on existing bindings; only additive `[context]` blocks.

- F8 (REFUTED) — A8: TOML sub-table ordering correct (flat fields precede sub-table per spec). `TestDefaultTemplateLoadsCleanly` validates decoder acceptance.

- F9 (NIT) — A9: Test names follow project convention; "Caps" merges MaxChars + MaxRuleDuration into one test. Splitting would improve failure-name precision but reduce test count. Style nit only.

- F10 (REFUTED) — A10: F.7.16 edits `[gates]` (line 350-351) ~30 lines before F.7.18.5's `[agent_bindings.*]` edits (line 387+). Three-way merge resolves trivially; plan declared explicit `blocked_by F.7.16.<id>` for additional safety.

- F11 (REFUTED-WITH-NIT) — A11: `[tillsyn]` block deferral acceptable per spawn-prompt narrowing + engine-default-substitution semantics. Refinement seed: future droplet seeds explicit `[tillsyn]` defaults to make adopter-visible. NOT a blocker.

- F12 (REFUTED) — A12: REV-13 honored — builder did not commit; suggested message provided for orchestrator.

- F13 (REFUTED) — A13: No memory-rule conflicts.

## Counterexamples

- None. No CONFIRMED counterexample produced across any of the 13 attack
  vectors. F2 and F11 routed as refinement seeds for future droplets, not
  blockers. F9 is a stylistic nit only.

## Summary

**Verdict: PASS-WITH-NITS.**

REV-4 contract enforced bidirectionally (positive on build, negative on all
four QA kinds via subtest-per-kind). Schema-aligned field types. Additive-only
diff with no scope creep into flat binding fields. TOML ordering correct.
Forward-compat with F.7.16 satisfied via plan-declared `blocked_by` and
non-overlapping line ranges. `[tillsyn]` deferral acceptable per spawn-prompt
narrowing.

NITs (non-blocking, route to refinements):
- N1 (A2/F2): no engine↔seed default-value cross-check test exists. File as
  refinement against F.7.18.3 acceptance: assert
  `defaultMaxChars` / `defaultMaxRuleDuration` in the engine package equal
  the values seeded in `default.toml`.
- N2 (A9/F9): `TestDefaultTemplateContextSeedsCaps` merges two field
  assertions. Optional refinement: split into MaxChars + MaxRuleDuration
  variants for finer-grained failure naming.
- N3 (A11/F11): explicit `[tillsyn]` block in default.toml deferred. File
  as refinement against the F.7.18.x family: future droplet seeds the
  block with values matching engine defaults so adopter-readable defaults
  exist on the embedded template.

No blocker. F.7.18.5 builder output is a clean PASS-WITH-NITS.

## Hylla Feedback

`N/A — action item touched non-Go files only` for the schema review;
the test-file work was straightforward additive Go with no cross-package
symbol search needed. Spawn prompt also explicitly forbade Hylla calls
("Hard constraints: No Hylla calls"). No queries issued, no miss to report.

## TL;DR

- T1 13 attacks walked. 11 REFUTED, 2 PASS-WITH-NIT (engine↔seed cross-check absent; `[tillsyn]` deferred).
- T2 No counterexample produced.
- T3 Verdict: **PASS-WITH-NITS**. Three refinement seeds (N1/N2/N3) routed forward; F.7.18.5 cleared.
