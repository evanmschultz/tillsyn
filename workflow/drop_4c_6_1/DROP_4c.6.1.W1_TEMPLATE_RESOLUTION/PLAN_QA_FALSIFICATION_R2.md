# W1 — Plan QA Falsification (Round 2)

**Verdict:** PASS-WITH-NIT — 0 CONFIRMED falsifications, 4 NITs queued
(one absorb-or-defer at planner discretion). Round-2 absorbed all three
round-1 FFs cleanly (FF1 group-name drift, FF2 cross-wave staleness, FF3
mergeTemplates under-specification) and addressed every round-1 NIT with
explicit ABSORB / DEFERRED-AS-NIT-with-reason disposition. Cross-wave
contract is now coherent: W2.D7 + W3.D1 round-2 plans confirm typed
`Metadata.Groups = payload.Groups` consumption (W2.D7 line 349; W3.D1
lines 232-234, 250-256). All 8 `templates.Template` fields enumerated in
W1.D2 acceptance #4 + RiskNotes. `agentBodyDefaultGroup` +
`agentBodyFallbackGroup` constant renames absorbed into D3 KindPayload
+ acceptance. No new structural defects introduced by round-2's surgical
edits.

NITs are residual authoring polish and forward-coupling reminders, not
load-bearing dispatch defects.

---

## 1. Findings

### 1.1 Counterexamples attempted

| # | Attack | Result |
|---|--------|--------|
| A1 | Round-2 Changes block contradicts PLAN.md body | REFUTED — every Changes entry maps cleanly to the corresponding D1/D2/D3 acceptance + KindPayload edit; sampled all 18 absorption rows |
| A2 | `loadProjectTemplateWithHome` empty-`homeDir` edge case unspecified | NIT-LEVEL (see 1.3 NIT1 below) — pattern is consistent with `readUserTierAgent` but the seam's empty-string semantics is implicit, not stated |
| A3 | `mergeTemplates` 8-field semantics: zero-value handling for `Tillsyn` struct under-specified | REFUTED with minor NIT (see 1.3 NIT2) — D2 acceptance #4 + RiskNotes name the `Tillsyn` rule explicitly: "whole-struct last-wins if overlay non-zero" + concrete non-zero predicate (`MaxContextBundleChars != 0 || MaxAggregatorDuration != 0 || SpawnTempRoot != ""`). One Tillsyn field missing from the predicate: `RequiresPlugins []string` (added per F.7.6 — see schema.go:332) |
| A4 | D3 rename ordering vs W4.D1 revert risk | REFUTED — `blocked_by W4.D1` is the structural enforcement; pre-cascade orchestrator gates dispatch on W4.D1 `complete` state; revert-after-merge is a process risk handled by `git status` confirmation pre-dispatch (the orchestrator inspects W4.D1's merge state before kicking D3, per CLAUDE.md "Build-QA-Commit Discipline") |
| A5 | W2.D7 / W3.D1 round-2 plans actually consume typed `ProjectMetadata.Groups []string` field | REFUTED — W2.D7 round-2 line 349: `Metadata.Groups = payload.Groups` (verbatim). W3.D1 round-2 line 232-234 + 250-256: `--add-group/--remove-group` mutate `Metadata.Groups []string` directly. Both confirm typed-field consumption — no KindPayload JSON stopgap. The cross-wave contract from W1.D2 → W2.D7 + W3.D1 is wired |
| A6 | Numeric consistency (3 droplets across narrative / D-list / `_BLOCKERS.toml` / summary table) | REFUTED — narrative (lines 51-71) names D1/D2/D3; D-list (lines 309, 361, 466) enumerates D1/D2/D3; summary table (lines 579-583) lists D1/D2/D3; `_BLOCKERS.toml` has 3 `[[blockers]]` entries (lines 5-18) for W1.D1/W1.D2/W1.D3. PASS |
| A7 | `agentBodyDefaultGroup` runtime behavior — other callers using `"till-go"` literal post-rename | PARTIAL — see 1.2 NIT3: render.go:175-176 + render.go:678 DOC-COMMENTS contain stale `till-gen` literal that D3 should sweep (cosmetic but explicit-in-spec is better) |
| A8 | MERGE-FIELD-AXIS-R1 refinement raising | REFUTED — D2 RiskNotes line 166-168 + line 442-445 explicitly raise the refinement with concrete fields enumerated for future revisit |
| A9 | `loadProjectTemplate` 4-tier conceptualization vs existing 3-candidate walk | REFUTED — current `loadProjectTemplate` has (1) bare-root candidate, (2) primary-worktree candidate, then embedded fallback. The "4-tier" framing in acceptance #1 maps: tier-1 = bare-root, tier-2 = primary-worktree, tier-3 = HOME (new), tier-4 = embedded fallback. Internally consistent |
| A10 | D3 `assembleAgentFileBody` already-resolved `group` variable threading | REFUTED — D3 RiskNotes line 515-518 + ContextBlock decision line 533-536 confirm `group` is resolved at render.go:663 via `resolveAgentGroup(binding)` and threaded down; current call site at render.go:666 reads `(project.RepoPrimaryWorktree, basename)` — D3 changes it to `(project.RepoPrimaryWorktree, group, basename)`. Signature change captured in KindPayload line 278 |

### 1.2 Counterexamples CONFIRMED

None. Round-2 absorbed round-1's three FFs cleanly and introduced no new
structural defects. The closest-to-falsification observations are NITs,
not counterexamples.

---

## 1.3 NITs (cosmetic / forward-coupling, with explicit dispositions)

### NIT1 — `loadProjectTemplateWithHome` empty-`homeDir` semantics not stated as acceptance

**Axis:** seam-contract well-formedness.

**Trace:**

- D1 acceptance #5 (PLAN.md line 328-329): "`os.UserHomeDir()` failure
  → HOME tier silently skipped (consistent with `readUserTierAgent`
  pattern in render.go:899-900)."
- D1 acceptance #6 (line 330-333) describes
  `loadProjectTemplateWithHome(project, homeDir, group string)` as the
  testability seam.
- D1 KindPayload (line 256) says "homeDir obtained from
  `os.UserHomeDir()` + skip on error" — at the CALLER (`loadProjectTemplate`).
- The seam's behavior when called DIRECTLY with `homeDir == ""` (e.g.
  D2's `loadProjectTemplatesForGroups` per-group loop) is implicit but
  not stated. Reading `readUserTierAgent` (render.go:898-902) shows the
  canonical pattern checks BOTH error AND `strings.TrimSpace(home) == ""`,
  which D1 acceptance #5 references by pointer.

**Why NIT not FF:** the implicit contract is unambiguous in context
(pattern-mirror to `readUserTierAgent`), and a competent builder
following the explicit reference will derive the right behavior.
A small explicit acceptance bullet ("`loadProjectTemplateWithHome`
called with `strings.TrimSpace(homeDir) == ""` skips the HOME tier
silently, returning the embedded fallback") would close the seam
formally without scope expansion.

**Disposition:** ABSORB into round-3 if a further round is run; OR
let D1 builder absorb via the explicit reference to `readUserTierAgent`
in acceptance #5. Per `feedback_nits_are_first_class.md`, default to
fixing — adding a single acceptance bullet costs nothing and removes
seam ambiguity.

**Fix hint:** add to D1 acceptance: "7. `loadProjectTemplateWithHome`
called with `strings.TrimSpace(homeDir) == ""` skips the HOME tier
internally (mirrors `readUserTierAgent`'s empty-home pattern at
render.go:900). Caller (`loadProjectTemplate` AND D2's
`loadProjectTemplatesForGroups`) may safely pass an empty `homeDir`
without precondition checks."

---

### NIT2 — `Tillsyn` non-zero predicate in `mergeTemplates` omits `RequiresPlugins` field

**Axis:** spec-conformance against current struct shape.

**Trace:**

- W1 PLAN.md RiskNotes line 163-164 + D2 acceptance #4 line 401-403:
  "`Tillsyn`: whole-struct last-wins; overlay `Tillsyn` replaces base
  if overlay is non-zero (`MaxContextBundleChars != 0 ||
  MaxAggregatorDuration != 0 || SpawnTempRoot != ""`)."
- Live `Tillsyn` struct at `internal/templates/schema.go:266-333`
  enumerates FOUR exported fields: `MaxContextBundleChars int`,
  `MaxAggregatorDuration Duration`, `SpawnTempRoot string`, AND
  `RequiresPlugins []string` (lines 300-332, added per F.7.6 / Drop 4c
  REV-7).
- The W1 plan's non-zero predicate names only the first three. A
  template that ONLY sets `RequiresPlugins = ["foo"]` (with the other
  three at zero values) would pass the non-zero predicate as written
  AND be treated as "non-zero overlay" only if the predicate is fixed
  to include `len(RequiresPlugins) > 0` — OR be silently ignored if
  the predicate is taken literally.

**Why NIT not FF:** pre-MVP, no template authored today exercises
multi-group aggregation OR sets `RequiresPlugins` differently per
group. The MERGE-FIELD-AXIS-R1 refinement explicitly covers
"revisit per-field semantics for `Tillsyn`...when multi-group projects
start exercising these fields in dogfood" — so the omission is
INSIDE the refinement's documented scope.

**Disposition:** ABSORB into round-3 if a further round is run — the
fix is one-line. Otherwise let D2 builder catch it via LSP
`documentSymbol` on `Tillsyn` (which D2 RiskNotes line 425-426 already
instructs the builder to do via reading `internal/templates/schema.go`).

**Fix hint:** update D2 acceptance #4 + RiskNotes line 163-164:
"overlay `Tillsyn` replaces base if overlay is non-zero
(`MaxContextBundleChars != 0 || MaxAggregatorDuration != 0 ||
SpawnTempRoot != "" || len(RequiresPlugins) > 0`)" — adds the fourth
field present in the live struct.

---

### NIT3 — D3 KindPayload omits sweep of stale `till-gen` literals in render.go DOC-COMMENTS

**Axis:** specify-block completeness / clean-rename.

**Trace:**

- D3 acceptance #3-4 (PLAN.md lines 484-485) update the CONSTANT
  values: `agentBodyDefaultGroup = "go"`, `agentBodyFallbackGroup =
  "gen"`.
- D3 acceptance #8 (line 498-500) covers test-fixture string literal
  updates.
- D3 acceptance #7 (line 494-497) covers `readEmbeddedTierAgent`'s
  cross-group fallback "from `builtin/agents/gen/<basename>` —
  correct after W4.D1's `git mv till-gen → gen`".
- BUT: `render.go:176` DOC-COMMENT for `agentBodyEmbeddedRoot` reads
  "falls back to `<agentBodyEmbeddedRoot>/till-gen/<basename>` on
  miss." `render.go:678` inline comment reads "Tier 3 — embedded
  tier with cross-group fallback to till-gen." `render.go:918` /
  `render.go:920-921` doc-comment for `readEmbeddedTierAgent` says
  "fallback — `builtin/agents/till-gen/<basename>`" and "If the
  primary group is already till-gen, the fallback is skipped."

**Why NIT not FF:** doc-comment drift doesn't affect runtime
behavior (constants are the load-bearing surface). But after D3
lands, the comments will reference a literal that no longer matches
the runtime value — confusing future readers. Standard clean-rename
hygiene flags this.

**Disposition:** ABSORB into round-3 if a further round is run; OR
add one bullet to D3 acceptance to instruct the builder to also
sweep render.go doc-comments for `till-gen` / `till-go` literals
when updating the constants. Per `feedback_nits_are_first_class.md`,
default to fixing.

**Fix hint:** add to D3 acceptance bullet 8 (or a new bullet 10):
"All doc-comments and inline comments in `render.go` referencing
`till-gen` / `till-go` literals are swept to `gen` / `go` (concretely:
the `agentBodyEmbeddedRoot` doc-comment at render.go:175-176, the
inline comment at render.go:678, the `readEmbeddedTierAgent`
doc-comment at render.go:914-930, and any other occurrences found via
`git grep 'till-g' internal/app/dispatcher/cli_claude/render/render.go`
post-edit). The constant rename + doc-comment sweep is atomic in this
droplet."

---

### NIT4 — Cross-wave producer→consumer flow: orchestrator must update W2.D7 + W3.D1 DESCRIPTIONS before dispatch (NOT just PLAN.md)

**Axis:** dispatch-time-contract / cross-wave-coordination.

**Trace:**

- W1.D2 acceptance #6 (PLAN.md line 408-411): "W2.D7 and W3.D1 MUST
  consume `project.Metadata.Groups` typed field directly...The
  orchestrator updates W2 + W3 PLAN.md before dispatching those
  droplets."
- W2 round-2 PLAN.md line 349: confirms `Metadata.Groups = payload.Groups`.
- W3 round-2 PLAN.md line 232-234, 250-256: confirms `--add-group/--remove-group`
  against `Metadata.Groups`.
- **Both downstream L2 PLAN.md files are already updated.** What the
  orchestrator MAY still need to do is ensure that the DISPATCH-TIME
  spawn prompt to the W2.D7 + W3.D1 builders also points them at the
  typed-field surface (vs. having a builder read a stale build-prompt
  template).

**Why NIT not FF:** the PLAN.md files are the dispatch-time source of
truth pre-cascade; both are correctly updated. This is a
forward-coupling reminder, not a defect in W1's plan.

**Disposition:** DEFERRED-AS-NIT — reason: outside W1's scope.
Routing-only reminder for the orchestrator when W2.D7 + W3.D1
dispatch arrives. No edit to W1 PLAN.md needed.

---

## 2. Counterexamples

None CONFIRMED. Every attack family in the spawn directive landed as
REFUTED or NIT-level.

---

## 3. Cross-Wave Consistency

| Cross-wave dependency | Producer | Consumer | Confirmation |
|----------------------|----------|----------|--------------|
| `ProjectMetadata.Groups []string` | W1.D2 acceptance #1 | W2.D7 acceptance line 349 (`Metadata.Groups = payload.Groups`) | WIRED — typed field consumed directly |
| `ProjectMetadata.Groups []string` | W1.D2 acceptance #1 | W3.D1 acceptance line 232-234, ContextBlock line 250-256 (`--add-group/--remove-group` operate on `Metadata.Groups`) | WIRED — typed field consumed directly |
| `git mv till-go → go`, `till-gen → gen` (embedded FS) | W4.D1 (Wave A) acceptance lines 381, 389 | W1.D3 acceptance #3-4 (constant renames), acceptance #7 (`readEmbeddedTierAgent` reads from `gen/`) | WIRED — D3 `blocked_by W4.D1` |
| Canonical group names `go`/`fe`/`gen` (no `till-` prefix) | W4.D1 + Round 10 locked decision (SKETCH §10) | W1.D1 HOME path segments + W1.D2 multi-group iteration | WIRED — all references use canonical names |
| `loadProjectTemplateWithHome` seam | W1.D1 (ships) | W1.D2 `loadProjectTemplatesForGroups` (consumes per-group) | WIRED — D2 `blocked_by D1`; seam signature pinned in D1 KindPayload + acceptance |

All five producer→consumer contracts are wired across L2 plans.

---

## 4. PLAN-QA-DISCIPLINE-R1 / R2 Adherence

**R1 (acceptance criterion → behavior-shipping droplet)**: PASS. Every
W1 wave-level AC bullet maps to D1, D2, or D3 with the corresponding
mage target.

**R2 (numeric consistency)**: PASS. Three droplets across all
surfaces — narrative, D-list, `_BLOCKERS.toml`, summary table,
"Total: **3 atomic droplets**." line.

---

## 5. NITs-First-Class Adherence

Round-1 raised 4 proof NITs + 10 falsification NITs. Round-2's
absorption block dispositions every one:

- ABSORBED: Proof NIT1, NIT2, NIT3, NIT4; Fals NIT1, NIT2, NIT3, NIT5,
  NIT6, NIT7, NIT8 (11 ABSORBED).
- DEFERRED-AS-NIT-WITH-REASON: Fals NIT4 (informal author-identity
  language; `blocked_by` is the structural guarantee), Fals NIT9
  (`strings.TrimSpace` already handles whitespace-only), Fals NIT10
  (LSP findReferences process note, not code defect).
- RESOLVED pre-round-2 by orchestrator: Proof FF1 = Fals NIT6 ("per
  U1" orphan in `_BLOCKERS.toml`).

Every disposition has an explicit reason. No "judgment call" /
"absorb inline per discretion" hand-waving. Per
`feedback_nits_are_first_class.md`: PASS.

---

## 6. Verdict Rationale

**Why PASS-WITH-NIT (not PASS-CLEAN):** four residual NITs (NIT1
seam-contract empty-homeDir; NIT2 `Tillsyn.RequiresPlugins` missing
from non-zero predicate; NIT3 doc-comment sweep; NIT4 cross-wave
dispatch reminder) are authoring polish, not defects. By
`feedback_nits_are_first_class.md`, default to fixing them; absorb
into round-3 if a further round is run, or let builders catch them
via the named LSP-driven verification steps already in the plan.

**Why NOT FAIL:** zero CONFIRMED counterexamples. The three round-1
FFs are resolved. Cross-wave contract is wired (W2.D7 + W3.D1 round-2
plans confirm typed-field consumption). `_BLOCKERS.toml` mirrors
PLAN.md. The blocked_by graph is acyclic
(W4.D1 → {D1 → D2, D3}). All 8 `templates.Template` fields are
enumerated with explicit per-field merge semantics. The
shipped-but-not-wired anti-pattern is closed: producer (W1.D2 types
Groups), consumer (W2.D7 / W3.D1 read typed Groups), W4.D1 rename
serialization (D3 `blocked_by W4.D1`) — all three legs nailed.

**Why NOT a falsification REFUTATION of the round-1 verdict:** the
round-1 FAIL was correct. Round-2 absorbed, the plan now stands.

---

## 7. Counter-Evidence Considered (REFUTED attacks beyond the spawn list)

- **REFUTED: D2 `loadProjectTemplatesForGroups` may call `loadProjectTemplateWithHome`
  with an unhandled error from one group, masking errors from later groups** —
  D2 acceptance #3 line 386-389 says it "iterates...calls `loadProjectTemplateWithHome`
  per group...Merges resulting `templates.Template` values via `mergeTemplates`."
  Error-aggregation behavior is implicit. **Why REFUTED**: per Go convention,
  the first-error-wins / propagate semantics is the default; D1 acceptance #4
  (line 326-327) says "HOME file exists but `templates.Load` errors → error
  propagates", and `loadProjectTemplatesForGroups` is a thin wrapper.
  Builders default to error propagation absent explicit override.
- **REFUTED: D3 test-fixture-churn might exceed the 120-LOC budget per droplet** —
  D3 RiskNotes line 519-523 acknowledges `render_test.go` is 1661 lines, but
  the EDITS are mostly fixture-string updates (mechanical s/till-go/go/) and
  flat-layout → subdir-per-group writes. The PRODUCTION-code changes in
  `render.go` (one signature change + 2 constant renames + 1 call-site
  update) are well under 120 LOC. Test-code edits are not LOC-budgeted in the
  same way; D3 stays atomic.
- **REFUTED: `bakeProjectKindCatalog` `homeDir` resolution might fail on systems
  without `$HOME`** — D2 KindPayload line 267 says
  "`homeDir` obtained from `os.UserHomeDir()` at `bakeProjectKindCatalog`
  call site." If `os.UserHomeDir()` fails, `homeDir` is the empty string;
  `loadProjectTemplatesForGroups` then passes empty to
  `loadProjectTemplateWithHome`, which (per NIT1 above) needs the
  empty-homeDir skip semantics. The pattern is consistent with
  `readUserTierAgent` (render.go:899-901). NIT1 captures the residual
  spec-tightening.
- **REFUTED: `mergeTemplates`'s `ChildRules` dedup tuple
  `(WhenParentKind, CreateChildKind)` might miss legitimate distinct rules
  for the same parent/child kind pair** — D2 RiskNotes + acceptance #4
  pin the tuple. Pre-MVP no template authors multiple ChildRules with
  identical `(WhenParentKind, CreateChildKind)`. MERGE-FIELD-AXIS-R1 covers
  the revisit if dogfood reveals a case.

---

## 8. Hylla Feedback

N/A — this falsification pass operated against MD planning artifacts and
on-disk Go source via `Read`. No Hylla queries were performed (per spawn
directive "Hylla is OFF").

---

## TL;DR

T1: PASS-WITH-NIT. Round-2 absorbed all three round-1 FFs (group-name
drift, cross-wave staleness, mergeTemplates under-spec) cleanly. Zero
new CONFIRMED counterexamples. Four residual NITs (empty-homeDir seam
contract, `Tillsyn.RequiresPlugins` non-zero predicate gap, doc-comment
sweep in render.go, cross-wave dispatch reminder for W2.D7+W3.D1) are
authoring polish. Cross-wave contract is wired: W2.D7 + W3.D1 round-2
plans confirm typed `Metadata.Groups` consumption; W4.D1 git-mv
serializes D3's constant renames via `blocked_by W4.D1`. Per
`feedback_nits_are_first_class.md`, default-fix the four NITs in
round-3 if it runs; otherwise allow D1/D2/D3 builders to absorb via
their already-explicit LSP-verification instructions.
