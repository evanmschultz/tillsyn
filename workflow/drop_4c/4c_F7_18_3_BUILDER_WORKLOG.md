# 4c.F.7.18.3 — Context Aggregator Engine — Builder Worklog

## Round 1

### Goal

Land the pure-function context aggregator engine in NEW package
`internal/app/dispatcher/context/`. Engine reads `binding.Context` rules +
the spawning action item, returns a `Bundle` that the F.7-CORE F.7.3b spawn-
pipeline render layer (later droplet) writes into the agent's spawn directory.

### Files created

- `internal/app/dispatcher/context/aggregator.go` (~22K)
  - `Resolve(stdcontext.Context, ResolveArgs) (Bundle, error)` — exported entry point.
  - `ResolveArgs` struct (Binding, Item, ProjectID, BundleCharCap, BundleDuration, Reader, DiffReader).
  - `Bundle` struct (RenderedInline, Files, Markers).
  - `ActionItemReader` + `GitDiffReader` ports.
  - `ErrNilReader` + `ErrNilDiffReader` sentinel errors.
  - Default-substitution constants: `defaultBundleCharCap = 200_000`,
    `defaultRuleCharCap = 50_000`, `defaultBundleDuration = 2s`,
    `defaultRuleDuration = 500ms`.
  - Closed-enum rule names + `allRuleNames` ordered slice (struct-declaration order).
  - Empty-binding fast-path (master PLAN.md L13 agentic mode).
  - Outer `stdcontext.WithTimeout` wrap (per-bundle wall-clock cap).
  - Per-rule `stdcontext.WithTimeout` wrap (per-rule wall-clock cap).
  - Greedy-fit cap algorithm: `cumulative + ruleSize > cap` → skip with marker, continue.
  - Per-rule truncation: full content stashed in `Files["<rule>.full"]`.
  - Empty-content skip: rules returning `nil` content emit no file and no marker.
  - Inline vs file delivery branch (Delivery="" defaults to file mode).
- `internal/app/dispatcher/context/rules.go` (~9.4K)
  - `resolveParent` — renders parent action item block.
  - `resolveParentGitDiff` — captures git diff between parent's start_commit/end_commit; empty commits → empty content (not error).
  - `resolveSiblingsByKind` — same-parent kind filter; latest round only by `CreatedAt` then ID lexicographic; renders in declaration order of accepted kinds; excludes the spawning item itself.
  - `resolveAncestorsByKind` — first-match-halt walk up parent chain; cycle guard + 256-hop cap.
  - `resolveDescendantsByKind` — DFS subtree walk; matches sorted by (CreatedAt, ID); ctx.Err() check inside loop; 4096-node cap.
  - `renderActionItemBlock` — terse markdown rendering (title + kind + paths/packages/commits + description).
  - `kindMatches` helper — closed linear scan (kind lists are template-bounded).
- `internal/app/dispatcher/context/aggregator_test.go` (~24K, 14 tests):
  1. `TestResolveEmptyBindingAgenticMode` — fast path: zero-value ContextRules → empty Bundle, no reader interactions.
  2. `TestResolveHappyPathAllRulesActive` — all 5 rules active; verifies file-mode produces `<rule>.md` for each.
  3. `TestResolvePerRuleTruncation` — oversized rule truncates with marker; full content in `<rule>.full`.
  4. `TestResolveGreedyFitCap` — rule 1 fits, rule 2 busts cap → skipped + marker, rule 3 fits → lands. NOT serial-drop.
  5. `TestResolvePerRuleTimeout` — slow `parent_git_diff` mock → timeout marker, subsequent rules continue.
  6. `TestResolvePerBundleTimeout` — outer wall-clock fires before all rules complete → aggregator timeout marker, partial bundle returned.
  7. `TestResolveDefaultSubstitution` — zero-valued caps + durations pick up engine-time defaults; verifies constant values.
  8. `TestResolveDeclarationOrderStable` — markers appear in `allRuleNames` declaration order regardless of binding slice order.
  9. `TestResolveInlineDelivery` — inline mode concatenates content into `RenderedInline`; nothing in `Files`.
  10. `TestResolveAncestorsByKindHaltOnFirstMatch` — walk halts on FIRST plan ancestor (parent), does NOT continue up.
  11. `TestResolveSiblingsByKindLatestRoundOnly` — multiple siblings with same kind collapse to most-recent by `CreatedAt`.
  12. `TestResolveParentGitDiffEmptyCommitsClean` — parent without start/end commits → empty content, no file written, no error.
  13. `TestResolveNilReaderRejected` — binding requires reader but `Reader == nil` → `ErrNilReader`.
  14. `TestResolveNilDiffReaderRejected` — binding requires diff reader but `DiffReader == nil` → `ErrNilDiffReader`.

### Implementation decisions

- **Package name `context`** vs stdlib collision: imported stdlib as `stdcontext` per Go convention. Callers from outside import as `aggcontext "..."` or similar.
- **Empty-content skip** (post round-1 fix): rules returning `nil, nil` (e.g. parent_git_diff with no commits, ancestor walk with no match) emit no file, no marker — empty content is a legitimate "nothing to render" outcome distinct from skip-via-cap.
- **Tie-breaking for latest-round selection**: `CreatedAt` first, then ID lexicographic. Documented in `resolveSiblingsByKind` doc-comment.
- **Cycle guards**: ancestors walk uses a `visited` set + 256-hop cap; descendants walk uses a `visited` set + 4096-node cap. Both surface as wrapped errors so corrupted SQLite data fails loudly.
- **Order stability**: `allRuleNames` is a static slice, NOT reflection over the `ContextRules` struct — guarantees stable iteration order.
- **Inline vs file delivery**: empty `Delivery` defaults to `templates.ContextDeliveryFile` per F.7.18.1 schema-layer comment "consumer-time default = file."
- **Markers in RenderedInline regardless of delivery**: file-mode bundles still need markers in `system-append.md` so the agent knows which rules were skipped or timed out. Inline-mode appends markers after content.
- **Empty-binding detection**: field-by-field check rather than `reflect.DeepEqual` — TOML decode produces nil slices vs empty slices inconsistently and field-by-field is robust to both.
- **Reader nil checks at evaluateRule dispatch**: ensures we never panic if a future caller forgets to wire a reader for a rule that needs it.

### Verification

- `mage testPkg ./internal/app/dispatcher/context/...` — 14/14 pass.
- `mage ci` — 2408 pass, 0 fail, 1 skip (pre-existing). Coverage on new package: **81.8%** (above 70% gate). Build green.

### Acceptance criteria status

- [x] `Resolve(ctx, args)` exported function with `ResolveArgs` + `Bundle` types.
- [x] `ActionItemReader` + `GitDiffReader` interfaces defined.
- [x] Empty-binding agentic-mode path returns clean empty Bundle.
- [x] Greedy-fit total-bundle cap: skips busting rules, continues to fitting subsequent rules.
- [x] Per-rule + per-bundle wall-clock caps enforce via `stdcontext.WithTimeout`.
- [x] All 8 spec test scenarios pass (delivered 14 — added inline-mode, ancestor-halt, sibling-latest-only, empty-commits, nil-reader, nil-diff-reader for stronger coverage).
- [x] Default substitution for zero-value caps works (50KB / 500ms / 2s / 200KB).
- [x] `mage check` + `mage ci` green.
- [x] Worklog written.
- [x] **NO commit** by builder (per F.7-CORE REV-13).

### Hylla Feedback

N/A — task touched non-Go-baseline-only files (new Go package, no Hylla queries needed; existing types read directly via Read tool).

### Suggested commit message

```
feat(dispatcher): land context aggregator engine F.7.18.3
```

(Single-line, ≤72 chars. Orchestrator commits.)
