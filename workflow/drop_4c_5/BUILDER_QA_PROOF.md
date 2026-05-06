# DROP_4c.5 — Build-QA Proof

Append a `## Droplet <ID> — Round K` section per QA attempt. See `workflow/example/drops/WORKFLOW.md § "Phase 5 — Build QA"`.

## Droplet F.2.1 — Round 1

**Reviewer:** go-qa-proof-agent
**Date:** 2026-05-05
**Verdict:** PASS

### Trace Coverage

1. **Acceptance #1 — byte-identical body content.** COVERED. `git diff --no-color HEAD -- internal/templates/builtin/` reports `similarity index 98%` with rename detection from `default.toml` → `default-go.toml` (total 21 diff lines). The full diff is the header expansion: 1 line removed (`# Tillsyn default cascade template (builtin).`) + 5 lines added/changed (new `Go default` header + Drop 4c.5 cross-reference comment). No body content (kinds, agent_bindings, child_rules, gates, steward_seeds, context blocks) altered.

2. **Acceptance #2 — `default.toml` no longer in tree.** COVERED. `ls internal/templates/builtin/` shows only `default-go.toml`. `git status --porcelain` confirms `D internal/templates/builtin/default.toml` (deletion staged via `git mv`).

3. **Acceptance #3 — embed directive correct.** COVERED. `internal/templates/embed.go:26` carries `//go:embed builtin/default-go.toml` — explicit-file form, NOT a glob (per F.2.1 falsification mitigation #2). Doc-comment at `embed.go:10-22` names the rebadge, the rationale for explicit-file form, and the F.2.2 / F.1.3 successors.

4. **Acceptance #4 + #5 — `LoadDefaultTemplate()` API preserved.** COVERED. `embed.go:55-62` keeps the function signature `LoadDefaultTemplate() (Template, error)` and calls `DefaultTemplateFS.Open("builtin/default-go.toml")`. Doc-comment at `embed.go:29-54` documents pre-F.1.3 contract (reads `default-go.toml` directly), names the F.1.3 thin-wrapper successor, and identifies the two existing callers (`seedStewardAnchors` at `auto_generate_steward.go:44` + the `loadProjectTemplate` deferral stub at `service.go:425`-area).

5. **Test rename done correctly.** COVERED. `embed_test.go:31` defines `TestDefaultTemplateGoLoadsCleanly` (renamed from `TestDefaultTemplateLoadsCleanly` per spec hint). Doc-comment at `embed_test.go:24-30` names the rename and points to F.2.1. Other tests in the file consistently reference `default-go.toml` in their doc-comments and failure messages (verified at lines 208, 294, 316, 319, 349, 406).

6. **`mage testPkg ./internal/templates`.** COVERED. Re-ran independently: `380 tests passed across 1 package` (0.01s). All `TestDefaultTemplate*` variants, including the renamed canary, are green.

7. **Caller audit completeness.** COVERED. `rg LoadDefaultTemplate` shows two production callers:
   - `internal/app/auto_generate_steward.go:44` — `return templates.LoadDefaultTemplate()`. Signature unchanged; call still compiles. Pre-MVP behavior preserved (Go-flavored content, the only content that ever existed).
   - `internal/app/service.go:425` — doc-comment reference only, inside the `loadProjectTemplate` Drop 3.14 deferral stub at `service.go:427-429` (returns `(Template{}, false, nil)`). The stub does NOT itself call `LoadDefaultTemplate`; the doc-comment merely names it as the function `seedStewardAnchors` uses. Unaffected.

   Historical doc-comments at `auto_generate_steward.go:35-36`, `service.go:380`, `kind_capability.go:594`, `kind_capability_test.go:139,141,256`, `auto_generate_steward_test.go:18,29`, `kind_capability_catalog_test.go:15`, `repo.go:311`, `child_rules_test.go:26`, `nesting_test.go:47`, `catalog_test.go:16` still reference `default.toml` literally. Per builder worklog ("Historical references … left unchanged because they describe past state"). These are descriptive prose about prior drops (3.14 / 3.15 / 3.20 / 5.B.8 etc.); none affect runtime behavior. Touching them all would balloon the droplet beyond F.2.1's mechanical-rename scope and is appropriately deferred to F.2.4 (caller audit + cross-package tests).

8. **Worklog completeness.** COVERED. `workflow/drop_4c_5/BUILDER_WORKLOG.md` § "Droplet F.2.1 — Round 1" contains: (a) date + builder + source-spec pointer; (b) Files-touched section detailing each file's change; (c) Targets-run section with specific test counts and timings; (d) Design-notes section explaining the explicit-embed-form choice, API preservation rationale, `git mv` use, header expansion vs body preservation, and the caller audit; (e) Hylla-feedback section with `N/A — task touched non-Go templates package work + workflow MDs only` rationale per CLAUDE.md "Hylla Indexes Only Go Files Today" rule. THEME_F_PLAN.md droplet F.2.1 heading shows `**State:** done (round 1)` at line 146.

### Findings

None. All eight checks landed clean.

### Conclusion

PASS. F.2.1's mechanical rename is byte-identical on body content (similarity index 98% confirmed by git rename detection), the embed directive uses the spec-mandated explicit-file form, `LoadDefaultTemplate()` API is preserved with both production callers continuing to receive byte-identical Go-flavored content, the test rename and doc-comment cleanup are consistent throughout `embed_test.go`, and `mage testPkg ./internal/templates` is green at 380/380. Worklog meets the orchestrator-audit bar with explicit Hylla-feedback rationale.

## Droplet E.1 — Round 1

**Reviewer:** go-qa-proof-agent
**Date:** 2026-05-05
**Verdict:** PASS

### Trace Coverage

**Check 1 — `equalStringSlices` deletion + `slices.Equal` swap.**
- `rg "slices.Equal" internal/app/dispatcher/locks_file_test.go internal/app/dispatcher/locks_package_test.go | wc -l` → **27**. Breakdown: 13 in `locks_file_test.go`, 14 in `locks_package_test.go`. 27 = 18 swapped existing call sites + 9 assertions inside the four new tests (4 in `TestFileLockManagerAcquirePreservesInputOrder` + 3 in `TestFileLockManagerAcquireDuplicateInputIdempotent`, mirrored). Builder's "18 call sites swapped" claim aligns.
- `rg "equalStringSlices" internal/app/dispatcher/` → **zero matches**. Helper definition fully removed; no residual call sites or comment references.
- `slices` import present at `locks_file_test.go:4` and `locks_package_test.go:4`.

**Check 2 — `TestFileLockManagerAcquirePreservesInputOrder` exists.**
- Defined at `locks_file_test.go:309`. Input `["c","a","b"]` against empty manager (L314-315) asserts `slices.Equal(acquired, []string{"c","a","b"})` (L322). Mixed-conflict extension at L330 (input `["b","x","a","y"]` after item-1 holds `a`+`b`) asserts `acquired2 == ["x","y"]` in input order. Spec acceptance #2 met.

**Check 3 — `TestFileLockManagerAcquireDuplicateInputIdempotent` exists.**
- Defined at `locks_file_test.go:355`. Input `["a","a","b"]` (L360) asserts `acquired == ["a","a","b"]` (L370) per documented per-occurrence semantics. Internal-state collapse probed externally via item-2 conflict count (`len(conflicts2) == 2`, L384) + post-Release re-acquire by item-3 (L398-407). Spec acceptance #3 met.

**Check 4 — Acquire doc-comment in `locks_file.go`.**
- "Input-order semantics" paragraph at `locks_file.go:70-76`: names `["c","a","b"]` example, calls out `slices.Equal` not sort-then-compare.
- "Duplicate-input semantics" paragraph at `locks_file.go:78-87`: states "each occurrence independently"; per-occurrence in `acquired`; internal `holders[path]` and `itemPaths[id][path]` "end identical to the de-duplicated case." Acceptance #4 met.

**Check 5 — Mirror in `locks_package.go`.**
- Mirror paragraphs at `locks_package.go:85-91` (Input-order) + `93-102` (Duplicate-input). Substitutions: `path → package`/`pkg`, `itemPaths → itemPackages`. Structural shape identical paragraph-for-paragraph. Mirror tests `TestPackageLockManagerAcquirePreservesInputOrder` (L348) + `TestPackageLockManagerAcquireDuplicateInputIdempotent` (L394) mirror file-side tests with same scenarios. Acceptance #5 met.

**Check 6 — `mage testPkg ./internal/app/dispatcher` green.**
- Re-ran independently: **354 tests passed** (0 failed, 0 skipped). Matches builder's 354/354 claim. Acceptance #6 met.

**Check 7 — Helper-consolidation correctness (semantic-shift audit).**
- `slices.Equal` is order-sensitive (Go stdlib spec). Builder's claim is that every existing call site already used input-order literals. Spot-check of 4 sites:
  - `locks_file_test.go:71→78` — `Acquire(item-2, ["a","b","c"])` against fresh-released manager, expects `acquired == ["a","b","c"]`. Input order = expected order.
  - `locks_file_test.go:95→99` — `Acquire(item-2, ["a","b"])` partial conflict, expects `acquired == ["a"]` (b elided in place). Input position preserved.
  - `locks_file_test.go:109→113` — same-holder retry `Acquire(item-2, ["a","c"])`, expects `acquired == ["a","c"]`. Input order = expected order.
  - `locks_file_test.go:188→192` — recovery acquire `[path]` (single element), expects `[path]`. Trivial.
- All 4 spot-checks confirm input-order literals; the swap from sort-then-compare to `slices.Equal` strengthens the assertions (it now catches a hypothetical future internal-sort regression) without invalidating any existing case. Helper-consolidation is semantically safe.

**Check 8 — Worklog completeness.**
- `BUILDER_WORKLOG.md` § "Droplet E.1 — Round 1" present (L36-66): Date, Builder, Source spec, Outcome, **Files touched** (5 files itemized — 4 Go + plan-row state line), **Design notes** (5 items: equalStringSlices decision, duplicate-input doc rationale, test naming alignment with spec, slices.Equal nil-vs-empty edge, mirror-integrity diff check), **Targets run** (`mage testPkg ./internal/app/dispatcher` 354/354 + `mage formatCheck` clean), **Hylla feedback** (`N/A` per filesystem-MD-mode directive — explicitly justifies the "no Hylla call" choice). Complete.
- `THEME_CE_PLAN.md` § E.1 row update verified at line 141: `**State:** done`.

### Findings

None. All 8 checks pass; no proof gaps.

### Conclusion

PASS. All six declared acceptance criteria satisfied with concrete file:line evidence. The two new tests (`TestFileLockManagerAcquirePreservesInputOrder`, `TestFileLockManagerAcquireDuplicateInputIdempotent`) plus their package-lock mirrors pin both input-order and duplicate-input contracts to the documented Acquire semantics. The `equalStringSlices` → `slices.Equal` swap was audited across 4 sampled call sites and confirmed semantics-preserving — every existing assertion already used input-order literals. Doc-comment paragraphs in `locks_file.go` and `locks_package.go` are paragraph-for-paragraph mirrors with package-vocabulary substitutions. `mage testPkg ./internal/app/dispatcher` re-run confirms 354/354. Worklog complete.

### Hylla Feedback

N/A — Drop 4c.5 cascade runs in filesystem-MD mode per spawn-prompt directive ("NO Hylla calls"). All evidence resolved via Read / Grep / Bash (`rg`, `mage testPkg`). No miss to report.

## Droplet D.1 — Round 1

**Reviewer:** go-qa-proof-agent
**Date:** 2026-05-05
**Verdict:** NEEDS-REWORK (resolved in round 2 via orchestrator decision)

### Summary

Round 1 builder mechanically executed the spec acceptance #1 ("exactly ONE replace directive — the fantasy-fork") and #2 ("strip `teatest/v2 => ./third_party/teatest_v2`"), regenerated `go.sum`, then ran `mage ci` per acceptance #4. The gate failed with two distinct load-bearing-pin failures:

- **L1 — `github.com/charmbracelet/ultraviolet`** — stripping the pin let `go mod tidy` resolve to current HEAD which renamed `*uv.RenderBuffer` → `*uv.Buffer`. The pinned `charm.land/bubbletea/v2 v2.0.0-rc.2` was authored against the old API; `cursed_renderer.go:444,698` no longer compiles. Affected `cmd/till`, `internal/tui`, `internal/tui/gitdiff`.
- **L2 — `github.com/alecthomas/chroma/v2 v2.14.0`** — chroma `v2.23.1` reordered the trailing `\x1b[0m` reset escape vs newline in syntax-highlight output; `internal/tui/gitdiff/testdata/golden/simple.ansi` was authored against `v2.14.0` byte sequence. `TestHighlighter_Golden` failed.

Per spec falsification mitigation #1 ("Builder MUST NOT force-fix … instead, surface the failure to the orchestrator"), round 1 builder correctly returned the action item with state `in_progress` + named load-bearing findings + recommended resolution paths rather than self-deciding the restoration semantics.

### Resolution Path

Orchestrator amended the spec semantics in round 2: spec acceptance #1 ("exactly ONE replace") was over-strict. The correct semantics — confirmed by the spec falsification mitigation #1 framing ("a stray `replace` that points at a missing path silently breaks every downstream build") — are: **strip every EXPERIMENTAL / STALE-PINNING replace; keep the fantasy-fork PLUS any load-bearing replaces required for API compatibility, with explicit `// load-bearing: <reason>` annotations naming the consumer constraint.** Round 2 restored the 3 load-bearing replaces (L1 ultraviolet, L2 chroma/v2, L3 teatest_v2 local fork) with annotations.

### Conclusion

Round-1 builder performance was correct under the spec-as-written: mechanical strip + surface findings + return without force-fix. The over-strict spec acceptance #1 was the actual defect, exposed by the round-1 `mage ci` red gate. NEEDS-REWORK is procedural; the orchestrator's spec amendment + round-2 restoration is the resolution path. No builder error to flag.

## Droplet D.1 — Round 2

**Reviewer:** go-qa-proof-agent
**Date:** 2026-05-05
**Verdict:** PASS

### Trace Coverage

**Check 1 — `go.mod` replace count + composition (orchestrator-amended semantics).**
- `rg "^replace" go.mod` returns exactly 4 lines: `charm.land/fantasy => github.com/evanmschultz/fantasy v0.0.0-20260219222711-d1be5103494b`, `github.com/charmbracelet/x/exp/teatest/v2 => ./third_party/teatest_v2`, `github.com/charmbracelet/ultraviolet => github.com/charmbracelet/ultraviolet v0.0.0-20251205161215-1948445e3318`, `github.com/alecthomas/chroma/v2 => github.com/alecthomas/chroma/v2 v2.14.0`.
- `rg "^replace \(" go.mod` confirms NO block-form `replace ( … )` directive — all 4 are line-form (no hidden 5th replace inside a block).
- Final composition: 1 fantasy-fork + 1 local-path fork (teatest_v2) + 2 published-version pins (ultraviolet + chroma/v2). Matches round-2 amended spec ("1 fantasy-fork + N load-bearing").

**Check 2 — Annotation completeness.** Each non-fantasy replace carries an explanatory comment with a `load-bearing:` token in the leading line:
- `go.mod:10` — `// load-bearing local fork: keeps TUI tests deterministic against charm.land/bubbletea/v2 drift; no published fork analog exists (per third_party/teatest_v2/README.md)` (precedes teatest_v2 replace at L11). Names the consumer (TUI tests via charm.land/bubbletea/v2 import path) AND the constraint (no published fork analog).
- `go.mod:13` — `// load-bearing: bubbletea/v2 v2.0.0-rc.2 expects *uv.RenderBuffer; ultraviolet HEAD provides *uv.Buffer (Drop 4c.5 D.1 finding L1)` (precedes ultraviolet replace at L14). Names the consumer (`bubbletea/v2 v2.0.0-rc.2`) AND the constraint (`*uv.RenderBuffer` API surface) AND back-references finding L1.
- `go.mod:16` — `// load-bearing: ANSI escape grouping in v2.23.1+ breaks internal/tui/gitdiff/testdata/golden/simple.ansi (Drop 4c.5 D.1 finding L2)` (precedes chroma/v2 replace at L17). Names the consumer (`internal/tui/gitdiff/testdata/golden/simple.ansi`) AND the constraint (ANSI escape grouping reorder) AND back-references finding L2.
- `go.mod:5-7` — fantasy-fork carries `// fantasy-fork: …` annotation (3 lines, full rationale + retention condition) per PLAN.md §19.1 line 1555. Per spec acceptance #1 the fantasy-fork uses the `fantasy-fork:` token (NOT `load-bearing:`) — this is correct: the fantasy-fork rationale category is distinct from a load-bearing version pin.

**Check 3 — `teatest/v2 => ./third_party/teatest_v2` retained with annotation.**
- Replace present at `go.mod:11`. Annotation at `go.mod:10` includes the literal phrase "no published fork analog exists" cross-referencing `third_party/teatest_v2/README.md`.
- Directory `third_party/teatest_v2/` confirmed present: contains `go.mod`, `go.sum`, `README.md`, `teatest.go` (4 files, ~14KB total). Round-2 builder explicitly noted "no edits this round" — directory contents preserved from prior tree state. Spec falsification mitigation #2 prerequisite met (real fork patches, NOT a stale leftover — README documents tea import-path patch from `github.com/charmbracelet/bubbletea/v2` → `charm.land/bubbletea/v2`).

**Check 4 — `go.sum` regenerated + consistent.**
- `git status --porcelain` shows `M go.sum` (modified, staged-able). Builder claim (round-2 worklog L210): "regenerated via `go mod tidy` post-restoration." File length: 248 lines, valid `<module> <version>/go.mod h1:<hash>` format on first 5 lines. `git diff go.mod` shows transitive shifts (`golang.org/x/mod` v0.33.0 → v0.34.0, `golang.org/x/tools` v0.42.0 → v0.43.0, `github.com/clipperhouse/stringish` removed) consistent with the 19 stripped experimental pins; chroma promoted from `// indirect` → direct (`go.mod:84` shows `github.com/alecthomas/chroma/v2 v2.23.1` without `// indirect`) — this matches the chroma/v2 require declaration that the pinned replace targets. Independent `go mod tidy` re-run not run (per spawn directive trusting builder claim); no inconsistency observable in committed `go.sum` shape.

**Check 5 — `mage ci` passes.**
- Trusted builder claim per spawn-note directive: 2705 passed / 1 skip / 24 packages / coverage met / build clean. No independent re-run attempted because (a) spawn note explicitly warns A.1 sibling builder is concurrently dirtying the tree (workflow-level git status confirms pointer-sentinel migration not in D.1 scope), (b) builder's round-2 `git stash` round-trip evidence at worklog L276-278 demonstrates the gate is green when D.1's diff is the ONLY uncommitted state. The 1 skip ("`TestStewardIntegrationDropOrchSupersedeRejected`") is pre-existing and tracked under B.1, not D.1.

**Check 6 — Worklog completeness.**
- Round 1 entry at `BUILDER_WORKLOG.md:68-190` includes Date, Builder, Source spec, State-at-end, Files touched, `teatest_v2` inspection result, Replaces stripped (22 lines enumerated), Replace retained (with full annotation), Rationale check from `git log`, Targets run, Load-bearing replace findings (L1+L2 with full root-cause + resolution paths), Resolution-deferred replaces, Cross-droplet sibling-edit awareness, Acceptance status table, Returned-to-orchestrator clause, Hylla feedback.
- Round 2 entry at `BUILDER_WORKLOG.md:192-292` includes Date, Builder, Source spec + directive, Outcome, Orchestrator-amended-semantics block, Files touched, Restoration block (verbatim), Survived strips (19 lines enumerated), Load-bearing rationales (L1+L2+L3 with consumer + constraint), Targets run (5 mage targets), Sibling-droplet stash maneuver, Acceptance status table (round-2 amended), Hylla feedback. Both rounds are complete and meet the orchestrator-audit bar.

**Check 7 — State row in THEME_BD_PLAN.md.**
- `THEME_BD_PLAN.md:135` — `**State:** done` (under Droplet D.1 heading, post-round-2 update from `in_progress`).
- `THEME_BD_PLAN.md:137-139` — round-2 outcome paragraph documenting the orchestrator semantics amendment is inserted under the heading. Cross-references "L1 `ultraviolet`", "L2 `chroma/v2`", "1 fantasy-fork + 3 load-bearing", "every other (19) experimental self-pin remains stripped", "`mage ci` green" — matches the round-2 worklog claims and the on-disk `go.mod` state.

### Findings

None. All 7 checks land clean.

### Conclusion

PASS. Round-2 builder satisfied the orchestrator-amended acceptance bar: `go.mod` carries exactly 4 replace directives (1 fantasy-fork + 3 load-bearing), each non-fantasy entry carries a `// load-bearing:` annotation naming both the consumer constraint AND the back-reference to round-1 findings (L1, L2). The teatest_v2 local fork is retained with annotation per the round-2 spec amendment. `go.sum` regenerated cleanly with traceable transitive shifts. `mage ci` green per builder claim (trust-builder directive in spawn note; A.1 concurrent activity precludes independent re-run). Worklog rounds 1 and 2 are both complete with all required subsections; THEME_BD_PLAN.md row state flipped to `done`. The round-1 NEEDS-REWORK was a procedural artifact of an over-strict spec acceptance #1 — round-1 builder behavior was correct (surface findings, do not force-fix); the round-2 amendment + restoration is the documented resolution path.

### Hylla Feedback

N/A — D.1 round 2 touched only non-Go files (`go.mod`, `go.sum`, `workflow/drop_4c_5/THEME_BD_PLAN.md`, `workflow/drop_4c_5/BUILDER_WORKLOG.md`). Hylla is Go-only today per project memory rule. All evidence resolved via Read / Bash (`rg`, `git diff`, `git status`, `ls`).

## Droplet A.1 — Round 1

**Reviewer:** go-qa-proof-agent
**Date:** 2026-05-05
**Verdict:** PASS

### Trace Coverage

1. **Acceptance #1 — `UpdateActionItemInput` struct shape.** COVERED.
   - `internal/app/service.go:678-708` declares `UpdateActionItemInput` with the spec-mandated pointer-sentinel shape: `Title *string` (684), `Description *string` (689), `Priority *domain.Priority` (694), `DueAt **time.Time` (703), `Labels *[]string` (708).
   - The `**time.Time` choice is structurally consistent with prior `*time.Time` semantics where the inner pointer doubles as a presence sentinel inside the domain entity (worklog rationale + struct doc-comment 695-702 explain the second indirection level): outer-nil = preserve, outer-non-nil-inner-nil = clear, outer-non-nil-inner-non-nil = set. This preserves the existing nil-vs-zero contract on `domain.ActionItem.DueAt` while adding a higher-order presence layer at the input boundary.
   - Doc-comments 664-677 explain the pattern + cross-reference the precedent fields (Owner / DropNumber / Persistent / DevGated / Paths / Packages / Files / StartCommit / EndCommit) that already used pointer-sentinels pre-A.1.

2. **Acceptance #2 — Service body branches on each pointer.** COVERED.
   - `internal/app/service.go:1263-1290` implements the five-pointer preserve-vs-apply chain. Each field (title, description, priority, dueAt, labels) initializes from the existing `actionItem` value, then conditionally overwrites only when the corresponding input pointer is non-nil. The merged values flow into the canonical `actionItem.UpdateDetails(...)` validator at line 1290.
   - Title-empty rejection still surfaces via `domain.UpdateDetails` -> `ErrInvalidTitle` (worklog confirms; service body line 1267-1269 doc-comment cross-references this invariant).
   - No new domain helper -- service composes inline (12 readable lines), keeps `domain.UpdateDetails` validation centralized.

3. **Acceptance #3 — Existing tests still pass.** COVERED via builder-claimed `mage ci` green (2715 passed / 1 pre-existing skip / 24 packages all >= 70% coverage). Trust-builder directive applies; independent re-run not required.

4. **Acceptance #4 — Three new table-driven test cases (description-preservation / title-preservation / explicit-clear).** COVERED with FULL 9-row table.
   - `internal/app/service_test.go:1538-1768` declares `TestUpdateActionItemPartialPATCHSemantics` with exactly 9 cases mirroring the THEME_A_PLAN.md spec table verbatim:
     1. `description nil preserves` (1581-1593)
     2. `description empty pointer clears` (1594-1608)
     3. `description non-empty replaces` (1609-1623)
     4. `title nil preserves` (1624-1637)
     5. `title empty pointer rejected` (1638-1648, asserts `domain.ErrInvalidTitle`)
     6. `labels nil preserves` (1649-1662)
     7. `labels empty pointer clears` (1663-1677)
     8. `priority nil preserves` (1678-1691)
     9. `due_at nil preserves` (1692-1705)
   - Test runner (1708-1767) seeds a fresh repo per case (no leak), asserts post-update Title/Description/Priority/DueAt/Labels equality, and on `expectErr` asserts both the wrapped error AND that the stored item remains unmutated (lines 1716-1724).

5. **Acceptance #5 — Empty title still rejected.** COVERED by row #5 above (`title empty pointer rejected`, `Title: ptrTo("")`, `expectErr: domain.ErrInvalidTitle`). The `errors.Is` check at line 1713 confirms the wrapped-error contract. The post-rejection state assertion (1716-1724) confirms no partial mutation leaked through.

6. **Acceptance #6/#7 — `mage test-pkg ./internal/app` and `./internal/adapters/server/common` pass with `-race`; `mage ci` clean.** COVERED via builder-claimed counts: `internal/app` 387/387, `internal/adapters/server/common` 160/160, `internal/tui` 372/372, `internal/adapters/server/mcpapi` 171/172 (one pre-existing skip), `mage ci` 2715 passed. Mage targets enforce `-race` by default per project rules.

7. **Wire-shape coordination — MCP tool description string.** PARTIAL — surfaced as Unknown.
   - The wire pointer-shape change DID land at the `args` anonymous struct in `internal/adapters/server/mcpapi/extended_tools.go:764-768` (Title/Description/Priority/DueAt = `*string`; Labels = `*[]string`), and the title-required preflight at the handler boundary was correctly removed (1065-1071 doc-comment + service-layer enforcement).
   - However, the published MCP tool description strings at `extended_tools.go:1437` (Title), 1452 (description), 1453 (priority), 1454 (due_at), 1455 (labels) -- and the legacy-alias declarations at 1501-1510 / 1528-1532 -- were NOT updated to document the new "omit to preserve, send empty string to explicitly clear" wire semantics. The `WithString("title", ...)` declaration still reads "Title. Required for operation=create|update" (1437). This is a documentation gap, not a behavioral defect: the runtime contract is correct, only the human-facing tool description text is stale.
   - Worklog § "Unknowns routed back to orchestrator" explicitly surfaces this as an open item recommending fold into D.2 hint sweep, A.2's wire-audit, or a small standalone docs droplet. PARTIAL coverage is acceptable on this specific point -- the spec-mandated falsification mitigation #1 about omit-vs-empty semantics IS implemented at the runtime layer (which is what protects callers from silent data loss); the description-string update is a lower-stakes follow-up the orchestrator can route. PASS verdict honors the runtime correctness; the docs gap is logged as F1.

8. **TUI call sites — pointer-sentinel idioms.** COVERED.
   - `internal/tui/model.go:6116-6127` (`buildCurrentEditActionItemInput`): wraps every field via `&titleVal` / `&descVal` / `&priorityVal` / `&dueAtVal` / `&labelsVal` with the local-var-then-take-address idiom required by `UpdateActionItemInput`'s pointer fields.
   - `internal/tui/model.go:8059-8065` (resource-add metadata-only path): collapses to nil-everything-except-metadata. Doc-comment 8055-8058 documents the preserve semantic.
   - `internal/tui/model.go:8604-8610` (resource-attach metadata-only path): same nil-everything-except-metadata shape.
   - `internal/tui/model.go:11647-11655` (labels-only update): passes `&labelsCopy` for Labels and nils for Title/Description/Priority/DueAt.
   - `internal/tui/model.go:19856-19862` (`parseActionItemEditInput`): wraps every field in pointer-sentinels, mirroring the build-side helper.
   - `internal/tui/thread_mode.go:514-521` (description-only thread update): passes `&description` for Description plus metadata, nils for Title/Priority/DueAt/Labels.
   - `internal/tui/trace.go:233-244` adds the `traceFormControlCharacterGuardPtr` thin wrapper that no-ops on nil and delegates to the value-typed guard otherwise -- preserves trace semantics across the pointer migration.

9. **Worklog completeness + Hylla feedback section.** COVERED.
   - `workflow/drop_4c_5/BUILDER_WORKLOG.md` § "Droplet A.1 — Round 1" (lines 294-371) carries every required subsection: Files touched (production + tests), Targets run, Design notes (with cross-droplet coordination guidance for A.2 / A.4 / B.1 / C.1 builders), Falsification-mitigation status, Hylla feedback (correctly N/A + explained), Unknowns routed back to orchestrator. Section is well above the standard.

10. **Migration safety — no UpdateActionItem callers left passing concrete values.** COVERED.
    - `rg "app\.UpdateActionItemInput\{" --type=go` (production-only) returns: `internal/adapters/server/common/app_service_adapter_mcp.go:897` (correctly uses pointer-sentinels), `internal/tui/model.go:6116, 8059, 8604, 11647, 19856` (all use pointer-sentinels per check #8), `internal/tui/thread_mode.go:514` (pointer-sentinels), `internal/app/dispatcher/conflict.go:319` (only sets Metadata/UpdatedType -- A.1-invisible per worklog § "For unrelated callers" point 6), `internal/app/dispatcher/service_adapter.go:44` (only Metadata/UpdatedType -- same).
    - Both dispatcher sites only ever wrote Metadata pre-A.1, so the new preserve-by-default pointer semantics are strictly correct for them -- no string clobbering risk because no string fields were being set.
    - Test sites (`service_test.go`, `kind_capability_test.go`, the four `app_service_adapter_*_test.go` files, `handler_steward_integration_test.go`, `model_test.go`) all use the new `ptrTo` test helper or inline `&local` per worklog. The test fakeService at `model_test.go` was rewritten to mirror production preserve-vs-apply semantics.

### Findings

- **F1 (informational, not blocking):** MCP tool description strings at `extended_tools.go:1437/1452-1455` and the legacy-alias clones at 1501-1510 / 1528-1532 still describe pre-A.1 wire semantics ("Title. Required for operation=create|update"). Runtime behavior is correct; only the human-facing tool descriptions are stale. Builder explicitly logged this as an Unknown routed back to orchestrator with three reasonable follow-up paths (D.2 hint sweep, A.2 wire-audit, or standalone docs droplet). Recommend the orchestrator pick one before drop close.
- **F2 (informational, not blocking):** Pre-A.1, an MCP `op=update` request that omitted `title` was rejected at the boundary with `invalid_request: required argument "title" not found` (handler-level preflight). Post-A.1, the same request silently preserves the stored title. Worklog § "Unknowns" notes this; per REVISION_BRIEF §6 ("pre-MVP, no production clients depend on tolerance"), the behavior change is acceptable. Flagged here so QA falsification can attack and orchestrator can decide whether to surface in CHANGELOG-equivalent.

### Conclusion

PASS. Droplet A.1 implements pointer-sentinel PATCH semantics on `Service.UpdateActionItem` exactly per spec: 5 pointer-sentinel fields landed (Title / Description / Priority / DueAt / Labels), service body branches cleanly on nil-vs-non-nil, the title-required invariant survives via `domain.UpdateDetails`'s `ErrInvalidTitle`, and the 9-row table-driven test mirrors THEME_A_PLAN.md verbatim including the empty-title rejection row. Wire-shape coordination at the `args` struct is correct; the MCP tool description-string update is a noted Unknown but not a runtime defect. All 16 source files + 2 workflow MDs in the declared file set are present and consistent. Migration safety holds: every production caller of `app.UpdateActionItemInput{...}` either uses pointer-sentinels or only sets Metadata/UpdatedType (dispatcher's two sites are A.1-invisible). Builder-claimed `mage ci` green (2715 passed / 24 packages / coverage met) accepted under the trust-builder directive.

### Hylla Feedback

N/A -- A.1 review touched Go source files but Hylla is stale post-Drop-4c-merge per the spawn-prompt's filesystem-MD-coordination directive (NO Hylla calls). All evidence resolved via Read / Grep / Bash (`rg`). Per project rule "Hylla Indexes Only Go Files Today" the Go-source review would normally favor Hylla; the override is drop-specific, not a Hylla ergonomics signal.
