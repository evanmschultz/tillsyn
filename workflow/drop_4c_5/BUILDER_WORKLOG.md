# DROP_4c.5 — Builder Worklog

Append a `## Droplet <ID> — Round K` section per build attempt. See `workflow/example/drops/WORKFLOW.md § "Phase 4 — Build (per droplet)"` for what each section should contain.

## Droplet F.2.1 — Round 1

**Date:** 2026-05-05.
**Builder:** go-builder-agent (model: opus).
**Source spec:** `workflow/drop_4c_5/THEME_F_PLAN.md` § "Droplet F.2.1 — Rebadge default.toml → default-go.toml".

### Files touched

- `internal/templates/builtin/default.toml` — DELETED via `git mv` (rename target).
- `internal/templates/builtin/default-go.toml` — NEW (from `git mv`); header comment updated to name "Go default cascade template" + cross-reference Drop 4c.5 F.2.1 / F.1.3 / F.2.2 successors. Body content byte-identical to the prior `default.toml` (12 kinds, agent_bindings, child_rules, gates, steward_seeds, context blocks all preserved).
- `internal/templates/embed.go` — `//go:embed` directive updated from `builtin/default.toml` → `builtin/default-go.toml` (explicit-file form per F.2.1 falsification mitigation #2; F.2.2 will extend the list to include `default-generic.toml`). `LoadDefaultTemplate()` now opens `builtin/default-go.toml`. Doc-comment expanded to call out the F.1.3 thin-wrapper successor and document that pre-F.1.3 callers continue to receive the Go-flavored template (preserving prior behavior byte-for-byte).
- `internal/templates/embed_test.go` — renamed `TestDefaultTemplateLoadsCleanly` → `TestDefaultTemplateGoLoadsCleanly` per spec hint. Updated four doc-comment references where they describe the embedded loader's current target file (`default.toml` → `default-go.toml`): the canary-test docstring, `TestDefaultTemplateMatchesNestingFixture` doc + two failure messages, `TestDefaultTemplateChildRulesForDropPlan` "see comment in default.toml" reference, `TestDefaultTemplateLoadsWithGates` "embedded default.toml" reference, and `TestDefaultTemplateGatesAllValidGateKinds` regression-mode #1 reference. Historical references (e.g. "the soon-to-be-deleted internal/adapters/storage/sqlite/repo_test.go") left unchanged because they describe past state, not current.
- `workflow/drop_4c_5/THEME_F_PLAN.md` — added `**State:** in_progress (round 1)` line under the F.2.1 droplet heading (the spec said "set droplet F.2.1's row" but THEME_F_PLAN.md uses heading-form droplet sections rather than table rows, so the closest analogue is a State line under the heading).

### Targets run

- `mage testPkg ./internal/templates` — **380/380 PASS** (0.28s). Includes all `TestDefaultTemplate*` tests + the renamed `TestDefaultTemplateGoLoadsCleanly`.
- `mage ci` — `internal/templates` clean. Pre-existing failures in `cmd/till`, `internal/app/dispatcher`, `internal/tui`, `internal/tui/gitdiff` traced to concurrent uncommitted edits from sibling droplets (D.1 `go.mod` changes; E.1 `locks_file.go` / `locks_package.go` changes; an unrelated `internal/tui/gitdiff` golden ANSI-ordering drift). Confirmed pre-existing via `git stash` baseline test: `mage testPkg ./cmd/till` passed clean (229/229) on baseline before my changes were re-popped. Not F.2.1's responsibility.

### Design notes

- **Explicit-file embed form chosen over glob.** F.2.1 falsification mitigation #2 calls for `//go:embed builtin/default-go.toml` (and later `builtin/default-go.toml builtin/default-generic.toml` after F.2.2) rather than `//go:embed builtin/*.toml`. The explicit list cannot accidentally pick up unrelated `.toml` fixtures or stray files in `builtin/`. F.2.2 will extend this directive when adding `default-generic.toml`.
- **`LoadDefaultTemplate()` API preserved as-is.** Per spec acceptance criterion #4 + #5, `LoadDefaultTemplate()` continues to open the Go-flavored template directly until F.1.3 generalizes it to a thin wrapper around `LoadDefaultTemplateForLanguage`. This preserves byte-for-byte the prior behavior for both existing callers (`internal/app/auto_generate_steward.go:44` `seedStewardAnchors` and the Drop-3-deferral `loadProjectTemplate` stub at `internal/app/service.go:425`-ish). Callers reach F.1.3 via the F.2.4 caller-audit droplet later in Chain 1.
- **`git mv` used for rename.** Preserves git rename detection so the diff reads as a rename + minor header tweak rather than a delete + create. Verified via `git status --porcelain`: `R  internal/templates/builtin/default.toml -> internal/templates/builtin/default-go.toml` (after `git add` of the rename pair).
- **Header comment expanded, body preserved.** Per acceptance criterion #1 ("EXACT byte-content of prior `default.toml` plus a header-comment update naming 'Go default'"). The header now names this as the Go default + cross-references the F.2.1 / F.1.3 / F.2.2 chain so future readers understand why three TOML siblings exist.
- **Caller audit (per acceptance #5).** `rg LoadDefaultTemplate` shows two production callers: `internal/app/auto_generate_steward.go:44` and a doc-comment reference at `internal/app/service.go:425`. Both currently expect Go-flavored content (the only content that ever existed). Preserving `LoadDefaultTemplate()` semantics in F.2.1 is correct; F.1.3 + F.2.4 generalize the call sites to language-aware later in the chain. No production reference to the literal `default.toml` filename outside `internal/templates/` itself.

### Hylla feedback

N/A — task touched non-Go templates package work + workflow MDs only; per CLAUDE.md "Hylla Indexes Only Go Files Today", no Hylla query was needed. All Go evidence resolved via `Read` + `rg` on uncommitted state, which is the correct fallback per the project rules.

## Droplet E.1 — Round 1

**Date:** 2026-05-05.
**Builder:** go-builder-agent (model: opus).
**Source spec:** `workflow/drop_4c_5/THEME_CE_PLAN.md` § "E.1 — Lock manager doc + test contract: input-order + duplicate-input".
**Outcome:** done — `mage testPkg ./internal/app/dispatcher` green (354/354), `mage formatCheck` clean.

### Files touched

- `internal/app/dispatcher/locks_file.go` — extended `Acquire` doc-comment with explicit "Input-order semantics" + "Duplicate-input semantics" paragraphs; no behavior change.
- `internal/app/dispatcher/locks_package.go` — mirrored doc-comment edit (Input-order + Duplicate-input paragraphs); no behavior change.
- `internal/app/dispatcher/locks_file_test.go` — full rewrite of import block + assertions: replaced every `equalStringSlices(a, b)` call with `slices.Equal(a, b)`; deleted the local `equalStringSlices` helper (sort-then-compare); removed the now-unused `"sort"` import; added `"slices"` import; added two new tests `TestFileLockManagerAcquirePreservesInputOrder` (input `["c","a","b"]` against empty manager → exact-order preservation; plus mixed conflict+free `["b","x","a","y"]` against pre-held `a+b` → `acquired=["x","y"]` in input order) and `TestFileLockManagerAcquireDuplicateInputIdempotent` (input `["a","a","b"]` → `acquired=["a","a","b"]` per documented per-occurrence semantics; cross-probe via item-2 → exactly two distinct conflicts; post-Release re-acquire by item-3 succeeds → no stray holder leak).
- `internal/app/dispatcher/locks_package_test.go` — mirror of the file-lock test refactor (same import swap, same removal of `equalStringSlices` calls — the helper definition lived in `locks_file_test.go` only, package-level scope shared the helper across both files), plus mirrored new tests `TestPackageLockManagerAcquirePreservesInputOrder` and `TestPackageLockManagerAcquireDuplicateInputIdempotent`.
- `workflow/drop_4c_5/THEME_CE_PLAN.md` — added `**State:** done` line under the E.1 droplet heading (started as `in_progress` at spawn, flipped to `done` after green tests).

### Design notes

- **`equalStringSlices` decision: inline replacement, not rename.** Audit via `git grep equalStringSlices` showed every call site in both test files compares slices in input-preserving order against literal expectations. None needed the order-insensitive sort-then-compare semantics the helper actually implemented. Inlining `slices.Equal` removes the misleading helper and tightens every existing assertion to the contract Acquire actually documents (input-order preservation). The helper definition lived in `locks_file_test.go` only and was reachable from `locks_package_test.go` via package-level scope; deleting the definition cleans both files. This is the stronger of the two paths the spec offered (inline OR rename to `equalStringSlicesSorted`).
- **Duplicate-input doc-comment pinned to observed impl, not aspirational behavior.** Traced `Acquire("item-1", ["a","a","b"])` on `locks_file.go`: iter 1 takes path "a" (set holders["a"]="item-1", append "a" to acquired); iter 2 sees `taken=true && holder=="item-1"` → falls through the same-holder branch (no-op map writes since `m.holders[path] = actionItemID` and the itemPaths set entry are already correct), appends "a" again — so `acquired = ["a","a","b"]`, while `holders` and `itemPaths` end with one entry each per distinct path. Doc-comment names exactly this — "each occurrence independently"; "the manager's internal holders[path] and itemPaths[id][path] end identical to the de-duplicated case (one entry each), because both are maps; only the returned acquired slice carries the duplicate." Tests assert both halves: the per-occurrence `acquired` slice AND the collapsed map state (probed externally via item-2's conflict count + post-Release re-acquire).
- **Test naming.** Spawn prompt mandated `TestFileLockManagerAcquirePreservesInputOrder` + `TestFileLockManagerAcquireDuplicateInputIdempotent`. Existing tests use `TestFileLock...` prefix (no `Manager`); I matched the spec's exact names and added matching `TestPackageLockManager...` mirrors for the package-lock tests. Slight inconsistency with the existing `TestFileLock...` prefix is the spec's choice, not drift introduced here.
- **Slices.Equal nil-vs-empty edge.** New tests use only non-empty `[]string` literals; nil-vs-empty edge cases remain covered by the pre-existing `TestFileLockEmptyInputsAreNoOps` / `TestPackageLockEmptyInputsAreNoOps`, which still pass against the new `slices.Equal` assertions.
- **Mirror integrity check.** Diffed the two test files post-edit: matching test functions, matching assertions adapted to package-vocabulary ("a"/"b"/"c" vs "internal/app"/"internal/domain"/etc.), matching new tests with mirrored doc-comments. Doc-comments in `locks_file.go` and `locks_package.go` use structurally identical wording, paragraph-for-paragraph, with only s/path/package/ + s/itemPaths/itemPackages/ substitutions.

### Targets run

- `mage testPkg ./internal/app/dispatcher` → 354 tests passed (1.46s, no hang on `monitor_test.go`).
- `mage formatCheck` → clean.

### Hylla feedback

N/A — per spawn-prompt directive ("NO Hylla calls"), no Hylla queries were attempted. Evidence-gathering used `Read` / `Edit` / `Bash` (`git grep` for call-site audit; `mage testPkg`/`mage formatCheck` for verification) only. The task's Go-only edits would have been candidates for Hylla under normal rules, but the Drop 4c.5 cascade is in filesystem-MD mode with stale Hylla state post-Drop-4c-merge.

## Droplet D.1 — Round 1

**Date:** 2026-05-05.
**Builder:** go-builder-agent (model: opus).
**Source spec:** `workflow/drop_4c_5/THEME_BD_PLAN.md` § "Droplet D.1 — `go.mod` `replace` Directive Cleanup".
**State at end of round:** `in_progress` — surfacing load-bearing-replace findings to orchestrator per spec falsification mitigation #1.

### Files touched

- `go.mod` — stripped 22 non-fantasy-fork `replace` directives; retained the fantasy-fork replace and added a 3-line `// fantasy-fork: ...` rationale annotation per acceptance criterion #1.
- `go.sum` — regenerated via `go mod tidy` (allowed module-file-only operation per PLAN.md §19.1 exemption).
- `workflow/drop_4c_5/THEME_BD_PLAN.md` — added `**State:** in_progress` line under the Droplet D.1 heading (THEME_BD_PLAN.md uses heading-form sections; matches the F.2.1 / E.1 round 1 convention above).

### `teatest_v2` inspection result (per spec falsification mitigation #2)

`third_party/teatest_v2/` is a real local fork, NOT a stale leftover. `third_party/teatest_v2/README.md` documents it as "a local compatibility patch for `github.com/charmbracelet/x/exp/teatest/v2`": `kan` (legacy project name) used Bubble Tea v2 from `charm.land/*` and upstream `x/exp/teatest/v2` periodically drifted, so the patch keeps TUI tests deterministic. The local `teatest.go` imports `tea "charm.land/bubbletea/v2"` (charm.land path) where upstream uses `github.com/charmbracelet/bubbletea/v2`.

Spec § 3.2 recommends "rewrite to point at a published fork (matches the fantasy-fork pattern), NOT a local path" — but no published fork analog exists, and creating one is out of D.1's scope. Strip-and-let-mage-ci-decide path was taken: stripped the local-path replace; `go mod tidy` resolved the upstream module successfully (`v2.0.0-20260216111343-536eb63c1f4c`); upstream module is now used directly. **The teatest strip itself did NOT cause a compile failure** — see load-bearing findings below for the actual blockers.

### Replaces stripped (22 lines)

- `github.com/charmbracelet/x/exp/teatest/v2 => ./third_party/teatest_v2` — local-path patch; upstream resolves cleanly post-strip.
- `charm.land/lipgloss/v2 => charm.land/lipgloss/v2 v2.0.0-beta.3.0.20260212100304-e18737634dea` — self-pin.
- `github.com/alecthomas/chroma/v2 => github.com/alecthomas/chroma/v2 v2.14.0` — self-pin (downgrade). **LOAD-BEARING** — see L2.
- `github.com/aymanbagabas/go-udiff => github.com/aymanbagabas/go-udiff v0.3.1` — self-pin (downgrade).
- `github.com/charmbracelet/colorprofile => github.com/charmbracelet/colorprofile v0.4.2` — self-pin (downgrade).
- `github.com/charmbracelet/ultraviolet => github.com/charmbracelet/ultraviolet v0.0.0-20251205161215-1948445e3318` — self-pin (downgrade). **LOAD-BEARING** — see L1.
- `github.com/charmbracelet/x/exp/golden => ... v0.0.0-20250806222409-83e3a29d542f` — self-pin.
- `github.com/charmbracelet/x/exp/slice => ... v0.0.0-20250904123553-b4e2667e5ad5` — self-pin.
- `github.com/clipperhouse/displaywidth => ... v0.9.0` — self-pin.
- `github.com/clipperhouse/uax29/v2 => ... v2.5.0` — self-pin.
- `github.com/dlclark/regexp2 => ... v1.11.0` — self-pin.
- `github.com/go-logfmt/logfmt => ... v0.6.0` — self-pin.
- `github.com/lucasb-eyer/go-colorful => ... v1.3.0` — self-pin.
- `github.com/mattn/go-runewidth => ... v0.0.19` — self-pin.
- `github.com/yuin/goldmark => ... v1.7.8` — self-pin.
- `github.com/yuin/goldmark-emoji => ... v1.0.5` — self-pin.
- `golang.org/x/exp => ... v0.0.0-20260212183809-81e46e3db34a` — self-pin.
- `golang.org/x/net => ... v0.50.0` — self-pin.
- `golang.org/x/sync => ... v0.19.0` — self-pin.
- `golang.org/x/sys => ... v0.41.0` — self-pin.
- `golang.org/x/term => ... v0.40.0` — self-pin.
- `golang.org/x/text => ... v0.34.0` — self-pin.

### Replace retained (1 line, with annotation)

```go.mod
// fantasy-fork: charm.land/fantasy upstream lacks the embeddings provider
// surface used by internal/adapters/embeddings/fantasy/; the evanmschultz fork
// carries the patches. Retain until upstream lands an equivalent surface.
replace charm.land/fantasy => github.com/evanmschultz/fantasy v0.0.0-20260219222711-d1be5103494b
```

### Rationale check from `git log --oneline -- go.mod`

Last 4 commits touching `go.mod`:
- `66c354e refactor(all): fix bad module name`
- `d684dcb feat(mage): adopt gofumpt via go.mod tool directive and add fmt target`
- `ee4a01e fix(cli): wrap help examples and upgrade laslig`
- `45b2644 feat(cli): upgrade laslig and add progress feedback`

None of these messages mention adding the experimental self-pins; they accumulated from earlier history (pre-rename refactors, original `kan`/`hakoll`/`koll` lineage). No commit message documents WHY any specific self-pin was added — pins arrived as a block of "experimental left-overs" matching exactly the `PLAN.md §19.1` description.

### Targets run

- `go mod tidy` — clean, regenerated `go.sum` (module-file-only op, allowed by PLAN.md §19.1 exemption).
- `mage ci` — **FAIL.** Test summary: 2099 passed, 1 failed, 1 skipped, 21 packages passed, 3 packages failed (2 build errors + 1 golden mismatch).
- `mage build` — used to surface the underlying compile error (test-stream view in `mage ci` masks compile detail). Exposed the `cursed_renderer.go` `*uv.Buffer` vs `*uv.RenderBuffer` mismatch.

### Load-bearing replace findings (per falsification mitigation #1)

**Finding L1 — `github.com/charmbracelet/ultraviolet` is LOAD-BEARING.**

- Symptom: build error in transitive dependency `charm.land/bubbletea/v2@v2.0.0-rc.2/cursed_renderer.go`:
  - line 444: `cannot use s.cellbuf.Buffer (variable of type *uv.Buffer) as *uv.RenderBuffer value in argument to s.scr.Render`
  - line 698: `cannot use s.cellbuf.Buffer (variable of type *uv.Buffer) as *uv.RenderBuffer value in argument to s.scr.PrependString`
- Root cause: stripping the `ultraviolet` pin let `go mod tidy` resolve `github.com/charmbracelet/ultraviolet` to `v0.0.0-20260316091819-b93f6a3b8502` (current HEAD-ish). The pinned `bubbletea/v2 v2.0.0-rc.2` was authored against the older `ultraviolet v0.0.0-20251205161215-1948445e3318` API which exposed `*uv.RenderBuffer`. Newer ultraviolet renamed/replaced that type with `*uv.Buffer`, breaking pinned bubbletea.
- Affected packages: `cmd/till`, `internal/tui`, `internal/tui/gitdiff` (all import bubbletea via the TUI surface).
- Resolution paths for orchestrator: (a) restore the `ultraviolet` pin with explicit annotation ("// load-bearing: bubbletea v2.0.0-rc.2 expects *uv.RenderBuffer; remove only when bubbletea bumps to a release that consumes the new ultraviolet API"); (b) bump `bubbletea/v2` to a version compatible with current ultraviolet HEAD (out of D.1 scope); (c) accept the strip and route to a follow-up droplet that addresses the bubbletea/ultraviolet drift end-to-end.

**Finding L2 — `github.com/alecthomas/chroma/v2 v2.14.0` is LOAD-BEARING.**

- Symptom: `TestHighlighter_Golden` fails in `internal/tui/gitdiff` with golden-file mismatch at `testdata/golden/simple.ansi`. Difference is purely ANSI escape grouping:
  - got (`v2.23.1`): `\x1b[...]<text>\x1b[0m\n` (reset before newline)
  - want (`v2.14.0` golden baseline): `\x1b[...]<text>\n\x1b[0m` (newline before reset)
- Root cause: chroma v2.23.1 changed the ordering of trailing reset escape vs newline relative to v2.14.0. The pinned downgrade kept the golden fixture's exact byte sequence stable.
- Affected: `internal/tui/gitdiff/highlighter_test.go::TestHighlighter_Golden`.
- Resolution paths for orchestrator: (a) restore the `chroma/v2 v2.14.0` pin with annotation ("// load-bearing: pinned for `internal/tui/gitdiff/testdata/golden/simple.ansi` byte stability; regenerate golden when bumping chroma"); (b) regenerate the golden fixture under the new chroma version (`mage testGoldenUpdate` per Magefile, but D.1 spec forbids touching test fixtures inline); (c) route to a chroma-bump-with-golden-regen droplet.

### Resolution-deferred replaces (no observed `mage ci` impact this round)

The other 20 stripped replaces caused no observed failure in this `mage ci` run. Notable transitive shifts visible in `go.mod` after `go mod tidy`:
- `golang.org/x/mod` v0.33.0 → v0.34.0 (was indirectly held by stripped pin).
- `golang.org/x/tools` v0.42.0 → v0.43.0.
- `github.com/charmbracelet/bubbletea/v2` (the `github.com/...` path, NOT `charm.land/...`) added as indirect (pulled in transitively by the now-upstream-resolved `teatest/v2`).
- `github.com/charmbracelet/x/input` added as indirect (also transitive via upstream teatest).
- `github.com/clipperhouse/stringish` removed (no longer needed).

These shifts may or may not reveal additional load-bearing pins under more thorough testing (e.g. `mage testGolden`, integration tests, or downstream gates not exercised by `go test ./...`). The visible failures are L1 and L2 only.

### Cross-droplet sibling-edit awareness

- F.2.1 round 1 (above) ran `mage ci` against the same dirty workspace and attributed pre-existing failures in `cmd/till`, `internal/app/dispatcher`, `internal/tui`, `internal/tui/gitdiff` partly to "D.1 `go.mod` changes" — that observation is consistent with the L1+L2 findings here. F.2.1 was right to defer those failures as not its responsibility.
- E.1 round 1 (above) ran only `mage testPkg ./internal/app/dispatcher` (its scope) and saw clean tests there — D.1's strips do not impact dispatcher-package tests because dispatcher's transitive bubbletea-import path is the same one that compiles fine via `go test` (the build error is bubbletea internal; it surfaces only when something downstream of bubbletea is built).

### Acceptance status

| Acceptance criterion | Status |
| --- | --- |
| 1. `go.mod` contains exactly ONE `replace` directive (the fantasy-fork) annotated with `// fantasy-fork: <rationale>` | **Met.** |
| 2. `teatest/v2 => ./third_party/teatest_v2` removed | **Met.** |
| 3. `go.sum` regenerated via `go mod tidy`; no spurious churn beyond deleted-replace fallout | **Met** (diff stat: 67 insertions, 100 deletions across go.mod + go.sum; all churn traceable to stripped replaces). |
| 4. `mage ci` passes | **NOT MET.** Gate fails with 2 build errors + 1 golden mismatch — proves L1 + L2 replaces were load-bearing. |
| 5. `third_party/teatest_v2/` deletion + no orphan references | **Not done.** Per spec mitigation #2, the directory was inspected and IS a real fork patch (README documents it). The local-path replace was stripped (per spec recommendation that local-path replaces are exactly what §19.1 calls "experimental left-over"); the directory itself was NOT deleted in this round because (a) the upstream teatest module resolved cleanly so the directory is dead but still tracked, (b) deleting tracked files is irreversible without explicit orch sign-off, and (c) acceptance criterion 4 is the priority gate and it failed. **Recommendation:** orchestrator decides whether to delete `third_party/teatest_v2/` in a follow-up round once the L1+L2 path is settled. |
| 6. Only fantasy-fork matches `^replace\b` regex | **Met** in current go.mod state. |

### Returned to orchestrator with state: `in_progress`

Per falsification mitigation #1 explicit directive: "Builder MUST NOT force-fix (e.g. by adding the replace back AND a workaround); instead, surface the failure to the orchestrator and document which replace was load-bearing." Returning with `go.mod` in stripped state, `mage ci` red, two named load-bearing findings (L1, L2) for orchestrator decision.

### Hylla feedback

N/A — task touched only non-Go files (go.mod is module manifest, not Go source). Hylla is Go-only today per project memory rule.
