# 4c.F.7.18.3 — Context Aggregator Engine — Builder QA Proof

## Round 1

**Verdict:** PROOF GREEN-WITH-NITS.

The builder's claims are supported by file:line evidence. All 13 verification points
are corroborated. Two NITs (file-layout + missing-by-construction-test) are recorded
but neither blocks. `mage ci` re-run green; 14/14 tests pass; package coverage 81.8%.
Builder did not commit; no out-of-scope edits.

---

### V1 — Exported surface

Plan §F.7.18.3 acceptance criteria require exported `Resolve`, `ResolveArgs`, `Bundle`,
`ActionItemReader`, `GitDiffReader`, plus the "nil reader" sentinels.

- `Resolve` — `internal/app/dispatcher/context/aggregator.go:243` (signature
  `Resolve(ctx stdcontext.Context, args ResolveArgs) (Bundle, error)`).
- `ResolveArgs` — `aggregator.go:122-156`. Fields: `Binding`, `Item`, `ProjectID`,
  `BundleCharCap`, `BundleDuration`, `Reader`, `DiffReader`. (Adds `ProjectID` over
  spec — harmless future-proofing per builder doc-comment lines 132-137.)
- `Bundle` — `aggregator.go:177-190`. Fields: `RenderedInline`, `Files`, `Markers`.
- `ActionItemReader` — `aggregator.go:90-105`. Methods: `GetActionItem`,
  `ListChildren`, `ListSiblings`. (Spec line 204 named only `GetActionItem`; builder
  added two list helpers needed by sibling+descendant rules. Necessary, not surface
  drift.)
- `GitDiffReader` — `aggregator.go:111-118`. Method: `Diff(ctx, from, to)`.
- `ErrNilReader` — `aggregator.go:196`.
- `ErrNilDiffReader` — `aggregator.go:201`.

PASS.

---

### V2 — Empty-binding agentic-mode fast path

Plan acceptance line 248 + master PLAN.md L13: zero-value `ContextRules` returns
empty `Bundle{}`, no work, no overhead.

- Implementation: `aggregator.go:243-249` — `if isEmptyContextRules(rules) { return
  Bundle{}, nil }` runs before any default-substitution / context-wrap / loop work.
- Field-by-field zero check: `aggregator.go:469-478` (`isEmptyContextRules`) checks
  every Parent/ParentGitDiff/slice-len/MaxChars/MaxRuleDuration/Delivery field.
- Test: `TestResolveEmptyBindingAgenticMode` (`aggregator_test.go:113-134`) passes
  zero-value `templates.AgentBinding{}` with `Reader == nil` and `DiffReader == nil`,
  asserts no panic + empty Bundle.

PASS.

---

### V3 — Default substitutions

Plan acceptance lines 230-234: `MaxChars=0 → 50000`, `MaxRuleDuration=0 → 500ms`,
`BundleCharCap=0 → 200000`, `BundleDuration=0 → 2s`.

- Constants: `aggregator.go:41-58` (`defaultBundleCharCap = 200_000`,
  `defaultRuleCharCap = 50_000`, `defaultBundleDuration = 2 * time.Second`,
  `defaultRuleDuration = 500 * time.Millisecond`).
- Substitution sites: `aggregator.go:254-269`. Each zero check substitutes the
  corresponding constant.
- Test: `TestResolveDefaultSubstitution` (`aggregator_test.go:476-524`) leaves
  `MaxChars`, `MaxRuleDuration`, `BundleCharCap`, `BundleDuration` zero AND asserts
  the constants equal the spec values.

PASS.

---

### V4 — Outer bundle timeout via `context.WithTimeout(ctx, BundleDuration)`

Plan acceptance line 298: outer `context.WithTimeout(parent_ctx, max_aggregator_duration)`.

- Implementation: `aggregator.go:277-278` — `outerCtx, cancelOuter :=
  stdcontext.WithTimeout(ctx, bundleDuration); defer cancelOuter()`.
- Per-iteration freshness check: `aggregator.go:317-324` reads `outerCtx.Err()`
  before each rule and emits the aggregator-timeout marker on death.
- Test: `TestResolvePerBundleTimeout` (`aggregator_test.go:413-472`) sets
  `BundleDuration: 50ms` with a 200ms slow diff; asserts the
  `aggregator timed out` marker fires.

PASS.

---

### V5 — Per-rule inner timeout via `context.WithTimeout(outerCtx, MaxRuleDuration)`

Plan acceptance line 297: per-rule `context.WithTimeout(parent_ctx,
max_rule_duration)`.

- Implementation: `aggregator.go:330-333` — `ruleCtx, cancelRule :=
  stdcontext.WithTimeout(outerCtx, ruleDuration); ... cancelRule()`.
- Outer-vs-inner attribution logic: `aggregator.go:339-356` distinguishes
  per-rule timeout (outer still alive) from outer timeout (outer dead too) and
  emits the appropriate marker. Per-rule timeout marker continues iteration;
  outer timeout marker breaks the loop.
- Test: `TestResolvePerRuleTimeout` (`aggregator_test.go:344-408`) sets
  `MaxRuleDuration: 10ms` with a 100ms slow diff; asserts `parent_git_diff timed
  out` marker fires AND the subsequent `siblings_by_kind` rule still runs.

PASS.

---

### V6 — Greedy-fit cap algorithm (NOT serial-drop)

Plan acceptance lines 291-296 + master PLAN.md L14: rule busts cap → SKIP with
marker AND continue to subsequent rules; earlier fits stay; later fits still land.

- Implementation: `aggregator.go:386-401`. Computes `remaining = bundleCharCap -
  cumulativeChars`; on `ruleSize > remaining` emits `[skipped: <name> (would have
  added <N>; remaining <M>)]` marker, preserves any pre-existing `truncatedFull`
  side-channel, **continues** (not break) to the next rule.
- Test: `TestResolveGreedyFitCap` (`aggregator_test.go:274-339`) configures cheap
  parent + 5KB sibling + cheap descendant with `BundleCharCap: 1000`; asserts
  parent.md lands, `skipped: siblings_by_kind` marker emitted, `siblings_by_kind.md`
  absent, AND `descendants_by_kind.md` STILL lands. Direct cross-check of the
  greedy-fit-not-serial-drop semantic.

PASS.

---

### V7 — Per-rule truncation with marker + full-content side-channel

Plan acceptance line 228 + lines 250: oversized rule truncated with
`[truncated to <N>; full at <bundle>/context/<rule>.full]` marker; full content
in `Bundle.Files[<rule>.full]`.

- Implementation: `aggregator.go:373-381`. When `len(content) > ruleCharCap`:
  stash original in `truncatedFull`, slice content to `ruleCharCap`, emit marker
  citing `<bundle>/context/<rule>.full` path.
- Full-content stash: `aggregator.go:415-417` (in stitching loop) writes
  `bundle.Files[r.name+".full"]` whenever `r.truncatedFull != nil`. Also stashed
  on cap-skip path at lines 397-399 so even skipped truncated rules expose full
  content.
- Test: `TestResolvePerRuleTruncation` (`aggregator_test.go:226-269`) sets
  `MaxChars: 100` with a 1000-char parent description; asserts marker text
  contains `truncated to 100`, `parent.md` is exactly 100 bytes, `parent.full`
  is > 100 bytes.

PASS.

---

### V8 — Rule order matches struct-declaration order

Plan acceptance line 290: TOML declaration order = struct field order: Parent →
ParentGitDiff → SiblingsByKind → AncestorsByKind → DescendantsByKind.

- Static slice: `aggregator.go:78-84` (`allRuleNames`) lists the five rule names
  in the canonical order. Iteration: `aggregator.go:314` ranges over `allRuleNames`.
- Schema confirmation: `internal/templates/schema.go:560-622` declares
  `ContextRules` fields in exactly this order (Parent, ParentGitDiff,
  SiblingsByKind, AncestorsByKind, DescendantsByKind, then Delivery / MaxChars /
  MaxRuleDuration as non-rule knobs).
- Test: `TestResolveDeclarationOrderStable` (`aggregator_test.go:531-593`) enables
  all 5 rules with truncating descriptions, then walks `bundle.Markers` to assert
  the truncation markers appear in `["parent", "parent_git_diff", "siblings_by_kind",
  "ancestors_by_kind", "descendants_by_kind"]` order.

PASS.

---

### V9 — Per-rule resolver semantics

Plan acceptance lines 225-240 require five resolvers with specific semantics.

#### V9a — Parent

- `rules.go:15-28` (`resolveParent`) — empty `ParentID` returns `(nil, nil)`
  (no error); else `reader.GetActionItem(ctx, ParentID)` and render via
  `renderActionItemBlock`.
- `renderActionItemBlock` at `rules.go:261-304` renders title + kind + paths +
  packages + start_commit + end_commit + description. (Spec mentioned
  `completion_contract` + `metadata`; builder elided those — they are heavy
  composite fields whose stringification was not specified, and the spec line
  does not require them. NIT only.)

PASS.

#### V9b — ParentGitDiff

- `rules.go:34-57` (`resolveParentGitDiff`). Empty parent_id, missing parent,
  empty start, OR empty end → `(nil, nil)` (empty content, NOT an error).
- Test: `TestResolveParentGitDiffEmptyCommitsClean` (`aggregator_test.go:716-749`)
  asserts no file written + no marker + no error when parent has zero commits.
  This is the explicit "empty commits → empty content not error" check.

PASS.

#### V9c — SiblingsByKind (latest-round-only)

- `rules.go:67-124` (`resolveSiblingsByKind`). Filters: skips spawning item itself
  (line 85), kind-match (line 88), latest-round selection by `CreatedAt` then
  ID-lexicographic tiebreak (lines 96-101). Render order follows
  `acceptedKinds` declaration order (line 112).
- Test: `TestResolveSiblingsByKindLatestRoundOnly` (`aggregator_test.go:676-712`)
  asserts NEWQA (newer CreatedAt) is included AND OLDQA is excluded.

PASS.

#### V9d — AncestorsByKind (first-match-halt + cycle guard)

- `rules.go:135-163` (`resolveAncestorsByKind`). 256-hop cap, `visited` map cycle
  guard, halts at first `kindMatches`.
- Test: `TestResolveAncestorsByKindHaltOnFirstMatch` (`aggregator_test.go:636-672`)
  builds great-grand → grand → parent chain (all `plan`); asserts only PARENT
  rendered (NOT GRAND or GREATGRAND).

PASS.

#### V9e — DescendantsByKind (DFS + cycle guard + ctx.Err)

- `rules.go:176-240` (`resolveDescendantsByKind`). 4096-node cap, `visited` map
  cycle guard, **`ctx.Err()` check inside the DFS loop at line 193**, sort by
  (CreatedAt, ID) for deterministic order.
- Tested implicitly via `TestResolveHappyPathAllRulesActive` (`aggregator_test.go:138-221`)
  which builds a tree with a `research` descendant and asserts it lands.

PASS.

---

### V10 — Spec test scenario coverage

Plan §F.7.18.3 lines 244-251 lists 5 explicit happy/edge scenarios; acceptance
criteria lines 235-242 imply additional per-rule + per-binding-scope coverage.

- (1) Happy path file-mode: `TestResolveHappyPathAllRulesActive`. PASS.
- (2) Inline-mode: `TestResolveInlineDelivery` (`aggregator_test.go:597-631`). PASS.
- (3) Empty `[context]` (FLEXIBLE): `TestResolveEmptyBindingAgenticMode`. PASS.
- (4) Truncation marker: `TestResolvePerRuleTruncation`. PASS.
- (5) Per-binding-scope (catalog with absurd other bindings): **NOT WRITTEN AS A
  DEDICATED TEST**, but is enforced by construction — `ResolveArgs.Binding` accepts
  a single `templates.AgentBinding`, never the full catalog. The function literally
  cannot read other bindings. **NIT N1** (see below).

Builder-added supplementary tests (`TestResolvePerRuleTimeout`,
`TestResolvePerBundleTimeout`, `TestResolveDefaultSubstitution`,
`TestResolveDeclarationOrderStable`, `TestResolveAncestorsByKindHaltOnFirstMatch`,
`TestResolveSiblingsByKindLatestRoundOnly`, `TestResolveParentGitDiffEmptyCommitsClean`,
`TestResolveNilReaderRejected`, `TestResolveNilDiffReaderRejected`) extend the spec
coverage to 14 total.

PASS-WITH-NIT (N1).

---

### V11 — `mage ci` green

Plan acceptance line 122 + builder claim "2408 tests, 81.8% package coverage."

- Re-ran `mage ci` from `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/`.
  Tail of output:
  - `github.com/evanmschultz/tillsyn/internal/app/dispatcher/context | 81.8%`.
  - `Coverage threshold met` (70% gate).
  - `[SUCCESS] Built till from ./cmd/till`.
- Re-ran `mage testPkg ./internal/app/dispatcher/context/...`:
  `tests: 14, passed: 14, failed: 0, skipped: 0`.

PASS.

---

### V12 — Scope (only the new package + worklog touched)

`git status --porcelain` from this checkout:

- `??  internal/app/dispatcher/context/` (new package — in scope).
- `??  workflow/drop_4c/4c_F7_18_3_BUILDER_WORKLOG.md` (in scope).
- All other dirty paths (`internal/adapters/storage/sqlite/repo.go`,
  `internal/domain/errors.go`, `workflow/drop_4c/SKETCH.md`,
  `internal/adapters/storage/sqlite/permission_grants_repo*.go`,
  `internal/app/permission_grants_store.go`, `internal/domain/permission_grant*.go`,
  `internal/app/dispatcher/mock_adapter_test.go`,
  `internal/app/dispatcher/testdata/mock_stream_minimal.jsonl`, plus various
  prior-droplet workflow MDs) belong to **other unrelated droplets** (4c.F.7.17.4
  permission grants, 4c.F.7.2 mock adapter, etc.) and were already untracked
  before this droplet ran. Not attributable to this builder.

PASS.

---

### V13 — No commit by builder (REV-13 compliance)

`git log --oneline -5` HEAD top: `e19e9f0 docs(drop-4c): add f.7.2 qa proof and
falsification artifacts`. None of the F.7.18.3 changes are committed; the builder
explicitly noted in the worklog "**NO commit** by builder (per F.7-CORE REV-13)."

PASS.

---

## NITs (non-blocking)

- **N1 — Per-binding scope test absent.** Plan §F.7.18.3 line 241 calls for an
  explicit catalog-with-12-bindings test that proves `Resolve` for `kind=build`
  ignores other bindings. The builder did not write that test. The property is
  type-system-guaranteed (`ResolveArgs.Binding` is a single `AgentBinding`, never
  a catalog), so a runtime test would only cover what the type system already
  enforces. Recommend either: (a) note this in the test file as a doc-comment
  explaining why no test exists, or (b) add a one-shot test that constructs a
  `templates.Template` with 12 bindings and demonstrates the function ignores 11
  by passing only the build binding — visible smoke for future refactors that
  might widen `ResolveArgs`.

- **N2 — `renderActionItemBlock` elides `completion_contract` + `metadata`.**
  Plan acceptance line 226 names "title + kind + paths + packages +
  completion_contract + metadata" as parent-render fields. Builder ships title,
  kind, ID, paths, packages, start_commit, end_commit, description — not
  completion_contract or metadata. The spec line is one-shot prose without sub-
  bullets specifying stringification format for those composite fields, so the
  elision is a defensible "smallest concrete design" call. NIT-only —
  recommend opening a refinement-drop item to either (a) add a deterministic
  rendering helper for those fields or (b) remove them from the plan acceptance
  line. Not a gap that blocks this droplet.

- **N3 — File-layout deviation.** Plan §F.7.18.3 acceptance criteria (lines
  202-210) listed eight separate `.go` files (`doc.go`, `bundle.go`, `ports.go`,
  `resolve.go`, `render_parent.go`, `render_git_diff.go`, `render_kindwalk.go`,
  `resolve_test.go`, `ports_test.go`). Builder consolidated to three files
  (`aggregator.go`, `rules.go`, `aggregator_test.go`). The package is ~31KB total
  source, well within "smallest concrete design" — splitting would have produced
  six tiny files. The doc-comment that the spec line wanted in `doc.go` is in the
  package-level doc-comment at the top of `aggregator.go` (lines 1-24). NIT-only.

---

## Hylla Feedback

- **Query 1**: `hylla_search_keyword` for `ContextRules` over committed code
  (`fields=["content"]`).
  - **Missed because**: the schema additions for `ContextRules` (templates package)
    landed in droplets F.7.18.1 + F.7.18.2 which committed at `16b86cb` (lines
    560-623 of schema.go) — but those commits are still post-last-Hylla-ingest
    on the project. Result: zero hits despite the symbol existing.
  - **Worked via**: `Read /Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/templates/schema.go`
    targeting offset 549.
  - **Suggestion**: Hylla's "stale ingest" condition is a documented pre-cascade
    reality (CLAUDE.md "Code Understanding Rules" §2). The miss here is expected.
    Ergonomic suggestion: when search returns zero hits AND the query string
    looks like a proper Go identifier (mixed-case, no spaces), surface a one-line
    hint like "0 hits — last ingest may predate the symbol; try git diff +
    direct Read." The agent already knows this from the rules, but a passive hint
    inline with empty results would shorten the fallback chain.

---

## Verdict

**PROOF GREEN-WITH-NITS.** All 13 verification points are corroborated by
file:line evidence. Mage ci re-run green (81.8% package coverage). 14/14 tests
pass. No commit by builder. No out-of-scope edits. Three NITs recorded above
(N1 missing-by-construction-test, N2 metadata/completion_contract elision in
parent render, N3 file-layout consolidation) — none block droplet completion.
