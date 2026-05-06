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

## Droplet E.2 — Round 1

**Reviewer:** go-qa-proof-agent
**Date:** 2026-05-05
**Verdict:** PASS

### Trace Coverage

1. **Acceptance #1 — `TestWalkerTreatsArchivedParentAsNotEligible` exists with archived-parent fixture; pins eligibility behavior.** COVERED. `walker_test.go:250-282` defines the test: parent fixture has `LifecycleState=StateTodo`, `ArchivedAt: &archivedAt` (a `time.Date(2026, 5, 1, ...)` value), and a child with `ParentID="parent-1"` + `LifecycleState=StateTodo`. Assertion at lines 277-281 walks the eligible set and fails if `candidate-1` appears. Domain field `ArchivedAt *time.Time` confirmed at `internal/domain/action_item.go:173` — fixture compiles correctly. The test's doc-comment (lines 228-249) explicitly addresses the "predicate doesn't currently check `ArchivedAt`; the existing `LifecycleState != StateInProgress` gate produces the rejection" reality and pins the observable contract (child not promoted) so a future ArchivedAt-explicit refactor stays passing — exactly the third-path framing the spec acceptance #1 endorsed ("If the predicate already correct via `includeArchived=false` filtering, the test asserts the filtering instead").

2. **Acceptance #2 — `TestWalkerListColumnsErrorPropagates` asserts wrapped-error preservation + `ErrPromotionBlocked`-not-set + `MoveActionItem`-not-called.** COVERED. `walker_test.go:540-568` defines the test. Three independent assertions land:
   - Line 559: `errors.Is(err, infraErr)` — wrapped sentinel preservation.
   - Line 562: `errors.Is(err, ErrPromotionBlocked)` MUST be false — sentinel reservation contract (ErrPromotionBlocked is for service-layer transition blocks only, not infra failures).
   - Line 565: `svc.moveCalls == 0` — `MoveActionItem` never called when `ListColumns` errors.

   All three match the spec acceptance #2 contract verbatim ("`Promote` returns wrapped error preserving `errors.Is(err, infraErr)`, NOT `errors.Is(err, ErrPromotionBlocked)`, AND `MoveActionItem` is never called"). The three-pronged shape is the right discriminator: a future regression where Promote silently maps infra-errors to ErrPromotionBlocked, OR drops the wrapped sentinel, OR calls MoveActionItem before the column-resolve step, all surface as test failures with distinct messages.

3. **Acceptance #3 — Doc-comment lines 45-75 clarifies BlockedBy resolution treats missing references AND non-complete blockers as "not-clear". Drift fix only, matches impl.** COVERED. Verified via `git diff`: `walker.go:49-58` is the only doc-comment touched. Pre-edit (single sentence): "Missing references (deleted siblings, typos) are treated as not-clear and skip the item — this is conservative on purpose: the planner sets BlockedBy and a missing target is a planner-side bug, not a walker-side override." Post-edit (multi-line): names BOTH failure modes explicitly ("a missing reference … AND a reference resolved to a non-StateComplete blocker (StateTodo / StateInProgress / StateFailed / StateArchived)"), restates the conservative-by-design framing ("planner-side bug should surface as a stalled-but-untouched item, not a wrongly-promoted one"), and adds the supersede / archive escape-hatch pointer. Behavior unchanged: `walker.go:185-187` still uses `if blocker.LifecycleState != domain.StateComplete { return false }` — the doc now matches impl. No production code outside the doc-comment touched.

4. **Test infrastructure — `stubWalkerService` extended with `columnsErr` field; injection seam works.** COVERED. `walker_test.go:14-32` (struct definition) carries the `columnsErr error` field. `walker_test.go:39-44` (ListColumns method) returns `(nil, s.columnsErr)` when set, else falls through to `(s.columns, nil)`. Doc-comments on the struct (lines 13-21) and the method (lines 34-38) explicitly explain the seam. Existing tests are unaffected — the new field defaults to nil-zero-value, so `TestWalkerFindsTodoItemWithClearedBlockers`, `TestWalkerPromotesEligibleItem`, etc. still hit the success path. The single-field extension is minimal and idiomatic (the alternative — a parallel `erroringListColumnsStub` mirroring the existing `erroringListItemsStub` — would have been busier; builder's worklog acknowledges and rejects it for good reason).

5. **Test rigor — both new tests have docstrings; both pin observable behavior, not implementation specifics.**
   - `TestWalkerTreatsArchivedParentAsNotEligible` doc-comment (lines 228-249): 22 lines explaining the defense-in-depth framing, the predicate-vs-upstream-filter split, and the future-refactor compatibility argument. Pin is on observable outcome (eligible set does not contain `candidate-1`), not on the internal gate path producing the rejection.
   - `TestWalkerListColumnsErrorPropagates` doc-comment (lines 533-539): 7 lines explaining the sentinel-reservation rationale (ErrPromotionBlocked is for service-layer transition blocks; infra failures stay distinguishable). Three independent assertions match three independent regression vectors as analyzed in §2 above.

6. **Worklog completeness — files-touched / targets-run / design notes / Hylla feedback section.** COVERED. `BUILDER_WORKLOG.md` § "Droplet E.2 — Round 1" (lines 417-458) carries:
   - **Files touched** (lines 425-433): walker.go (doc paragraph 2 rewrite), walker_test.go (`time` import + stub extension + 2 new tests), THEME_CE_PLAN.md state flip, BUILDER_WORKLOG.md self-entry.
   - **Design notes** (lines 435-439): explicit dispositions for spec acceptance #1/#2/#3, rationale for the third-path test design + minimal-stub-extension choice + scoped doc-edit.
   - **Falsification-mitigation status** (lines 441-445): all three F-attacks named in spec line 202-204 explicitly addressed (upstream-filter bypass, doc-drift scope, false-coverage trap).
   - **Sandbox hang note** (lines 447-449): builder reports no `monitor_test.go` hang; `mage test-pkg` ran 1.75s clean.
   - **Targets run** (lines 451-454): `mage test-pkg ./internal/app/dispatcher` 356/356 PASS + `mage formatCheck` clean.
   - **Hylla feedback** (lines 456-458): N/A per spawn-prompt directive.

7. **Builder claim — 356/356 (354 existing + 2 new).** COVERED arithmetically. E.1 round 1 reported 354 existing tests (worklog line 61). E.2 adds exactly 2 new test functions: `TestWalkerTreatsArchivedParentAsNotEligible` (line 250) and `TestWalkerListColumnsErrorPropagates` (line 540). 354 + 2 = 356 — matches the claimed test count.

### Findings

None. The build is tight: minimal scope, accurate doc-fix, well-rationalized test choices, infrastructure extension via single nullable field, and worklog completeness covering every required surface. The "predicate doesn't currently check ArchivedAt" gap is acknowledged in the test's own doc-comment and addressed via observable-outcome pinning rather than tautological assertion — the test catches both the existing LifecycleState gate AND a hypothetical future ArchivedAt-explicit gate, which is exactly what defense-in-depth contracts call for.

### Conclusion

PASS. E.2 lands all three acceptance criteria precisely as scoped. The two new tests pin observable predicate / Promote behavior with three-pronged assertions where the spec named them, the `stubWalkerService` extension is minimal and the seam is documented, and the doc-comment edit is a tight drift fix on paragraph 2 with no behavior change. Builder-claimed `mage test-pkg ./internal/app/dispatcher` 356/356 PASS + `mage formatCheck` clean is consistent with the file diffs (one production file gets a doc-only change; one test file adds 1 import + 1 field on the stub + 2 new test functions). No regressions to existing tests visible from the diff.

### Hylla Feedback

N/A — E.2 review touched Go source files but Hylla is stale post-Drop-4c-merge per the spawn-prompt's filesystem-MD-coordination directive (NO Hylla calls). All evidence resolved via Read / Bash (`rg ArchivedAt` for one domain-field cross-check) / `git diff`. Per project rule "Hylla Indexes Only Go Files Today" the Go-source review would normally favor Hylla; the override is drop-specific, not a Hylla ergonomics signal.

## Droplet F.2.2 — Round 1

**Reviewer:** go-qa-proof-agent
**Date:** 2026-05-05
**Verdict:** PASS

### Trace Coverage

1. **Acceptance #1 — valid v1 schema, 12-kind catalog, 4 child_rules, 6 STEWARD seeds.** COVERED.
   - `internal/templates/builtin/default-generic.toml:56` — `schema_version = "v1"`.
   - 12 `[kinds.<kind>]` sections at lines 75-205: plan, research, build, plan-qa-proof, plan-qa-falsification, build-qa-proof, build-qa-falsification, closeout, commit, refinement, discussion, human-verify.
   - 4 `[[child_rules]]` entries at lines 224-249: build→build-qa-proof, build→build-qa-falsification, plan→plan-qa-proof, plan→plan-qa-falsification. Drop-narrowed entries explicitly omitted; comment block at lines 251-265 names the rationale.
   - 6 `[[steward_seeds]]` entries at lines 284-306: DISCUSSIONS, HYLLA_FINDINGS, LEDGER, WIKI_CHANGELOG, REFINEMENTS, HYLLA_REFINEMENTS.

2. **Acceptance #2 — `[agent_bindings]` table absent; test pins `len == 0`.** COVERED.
   - `default-generic.toml:325-336` — explicit prose-comment block names the omission as a load-bearing contract; no `[agent_bindings]` table or sub-keys present.
   - `embed_test.go:157-159` — `if got := len(tpl.AgentBindings); got != 0 { t.Fatalf(...) }`. Direct regression guard.

3. **Acceptance #3 — file loads through `templates.Load` validator chain.** COVERED. `embed_test.go:79-88` opens via `DefaultTemplateFS.Open("builtin/default-generic.toml")` then calls `Load(f)`. Builder reports `mage testPkg ./internal/templates` 381/381 PASS — every `Load` validator (version pre-pass, strict decode, validateMapKeys, validateChildRuleKinds, validateChildRuleCycles, validateGateKinds, validateAgentBindingEnvNames, validateAgentBindingContext, validateAgentBindingToolGating, validateTillsyn, validateChildRuleReachability) ran in that path and accepted.

4. **Acceptance #4 — `TestLoadDefaultGenericTemplate` exists with all required assertions.** COVERED. `embed_test.go:76-160`:
   - Opens via embed.FS (line 79-83).
   - `Load(f)` round-trip (line 85-88).
   - `SchemaVersion == SchemaVersionV1` (line 90-92).
   - `len(Kinds) == len(allKinds)` (i.e. 12) plus per-kind presence loop (line 95-102).
   - `len(ChildRules) == 4` plus edge-by-edge enumeration over a `wantChildRuleEdges` map; defensive guard rejects any non-empty `WhenParentStructuralType` (line 104-130).
   - `len(StewardSeeds) == 6` plus title-by-title enumeration over a `wantSeedTitles` map (line 132-154).
   - `len(AgentBindings) == 0` (line 157-159).

5. **Acceptance #5 — embed directive uses explicit two-file form.** COVERED. `embed.go:29` reads:

   ```
   //go:embed builtin/default-go.toml builtin/default-generic.toml
   ```

   Two filenames space-separated, NOT a glob (`builtin/*.toml`). Doc-comment at `embed.go:7-17` explicitly names this choice and ties it to F.2.1 falsification mitigation #2 (carried forward to F.2.2): an explicit list cannot accidentally pick up unrelated `.toml` fixtures or stray files in `builtin/`.

6. **Acceptance #6 — `LoadDefaultTemplate()` API unchanged.** COVERED. `embed.go:58-65` keeps the function signature identical to F.2.1's round and still calls `DefaultTemplateFS.Open("builtin/default-go.toml")`. Doc-comment at `embed.go:32-57` notes the F.1.3 successor that will reduce this function to a thin wrapper around `LoadDefaultTemplateForLanguage` but explicitly preserves byte-for-byte behavior pre-F.1.3. The `TestDefaultTemplateGoLoadsCleanly` canary (renamed in F.2.1) still passes per the 381/381 result.

7. **Worklog completeness.** COVERED. `BUILDER_WORKLOG.md` § "Droplet F.2.2 — Round 1" (line 460-491) contains: (a) date + builder + source-spec pointer (line 462-465); (b) Files-touched section detailing the new TOML, the embed directive extension, and the new test (line 467-471); (c) Targets-run section with the 381/381 PASS count + `mage formatCheck` clean (line 473-476); (d) Design-notes section explaining the drop-narrowed omission, the OMIT-vs-empty agent_bindings choice and its falsification linkage (F2 — validator did not reject), the test entry-point choice (direct embed.FS open until F.1.3 lands), the defensive drop-narrowed guard, the STEWARD seed and gate parity rationales, and per-validator clean-pass enumeration (line 478-486); (e) Hylla-feedback section with `N/A — task touched only Go-eligible files in principle ... per spawn-prompt directive "filesystem-MD coordination mode. NO Hylla calls"` (line 488-490). THEME_F_PLAN.md droplet F.2.2 heading shows `**State:** done (round 1)` at line 185.

### Findings

None. All six acceptance criteria + worklog completeness landed clean. The `[agent_bindings]` omission is implemented as full table absence (cleaner showcase contract than an empty table) AND pinned in the test as `len == 0` — the load-bearing regression guard. The drop-narrowed `[[child_rules]]` omission is similarly pinned both in the TOML's prose comment AND as a defensive `WhenParentStructuralType != ""` reject inside the test loop, preventing future drops from silently re-introducing them.

### Hylla Feedback

N/A — F.2.2 review touched Go-eligible files (`embed.go`, `embed_test.go`) plus a new TOML and workflow MDs. Per spawn-prompt directive "filesystem-MD coordination mode. NO Hylla calls" all evidence resolved via Read / git diff (verified via mtime + the worklog manifest of files touched). Per project rule "Hylla Indexes Only Go Files Today" the Go-source review would normally favor Hylla; the override is drop-specific, not a Hylla ergonomics signal.

### Conclusion

PASS. F.2.2 ships the language-agnostic showcase precisely as scoped: the closed 12-kind catalog, the four standard `[[child_rules]]`, the six STEWARD seeds, the `[gates.build]` sequence parity with default-go, and the deliberate `[agent_bindings]`-table omission — every one pinned via direct test assertion. The embed directive uses the spec-mandated explicit two-file form. `LoadDefaultTemplate()` semantics are preserved byte-for-byte (F.1.3 will generalize later). `mage testPkg ./internal/templates` 381/381 PASS = 380 prior + 1 new (`TestLoadDefaultGenericTemplate`) — arithmetic checks against F.2.1's 380-test baseline. Worklog is complete with explicit Hylla-feedback rationale.

## Droplet F.2.3 — Round 1

**Reviewer:** go-qa-proof-agent (filesystem-MD coordination mode — NO Tillsyn / Hylla calls).
**Date:** 2026-05-05.
**Source spec:** `workflow/drop_4c_5/THEME_F_PLAN.md` § "Droplet F.2.3 — Self-host `<project_root>/.tillsyn/template.toml` for tillsyn".
**Builder round under review:** `workflow/drop_4c_5/BUILDER_WORKLOG.md` § "Droplet F.2.3 — Round 1" (lines 492-538).
**Verdict:** PASS.

### Premises

1. P1 — `.tillsyn/template.toml` exists at repo root with valid v1 schema.
2. P2 — Header comment block names this as the tillsyn self-host template (NOT the embedded-builtin headering).
3. P3 — Body content (from `schema_version = "v1"` onward through `[agent_bindings.human-verify]`) is faithful to `internal/templates/builtin/default-go.toml`.
4. P4 — A `[tillsyn]` table with `spawn_temp_root = "os_tmp"` is appended at the bottom.
5. P5 — `"os_tmp"` matches the dispatcher's consumer-time default at `internal/app/dispatcher/bundle.go:246-256` `resolveSpawnTempRoot` (empty → `SpawnTempRootOSTmp`; `"os_tmp"` → `SpawnTempRootOSTmp`; observably equivalent).
6. P6 — `.gitignore` re-include rule is correctly wired: `.tillsyn/*` excludes contents AND `!.tillsyn/template.toml` re-includes the dogfood seed.
7. P7 — Runtime state (`spawns/`, `tillsyn.db`, `tillsyn.db-shm/-wal`, `livewait.secret`, `logs/`, `config.toml`) remains ignored.
8. P8 — `mage ci` is green (2719 pass / 1 pre-existing skip / 24 packages, all ≥ 70% coverage / build clean).
9. P9 — `.tillsyn/template.toml` is tracked-eligible (NOT `git add`-ed yet, but will stage cleanly; not blocked by gitignore).
10. P10 — Worklog round entry is complete (files touched, targets run, design notes, falsification-mitigation status, Hylla feedback).

### Evidence

- E1 (P1, P2): `Read .tillsyn/template.toml` lines 1-50 — header comment block (lines 1-46) names the tillsyn self-host template, body header `schema_version = "v1"` at line 47, `# [kinds]` block at lines 49-51. Total 696 lines.
- E2 (P3): Spot-checked three reference points:
  - Schema-version line: `.tillsyn/template.toml:47` matches `default-go.toml:22` exactly (`schema_version = "v1"`).
  - `[kinds]` block heading structure matches at both files.
  - Tail of body: `.tillsyn/template.toml:653` ends `[agent_bindings.human-verify]` block (matching `default-go.toml:653`: `agent_name = "orchestrator-managed"` ... `blocked_retries = 0`).
  - Line-delta arithmetic: 696 - 653 = +43 lines, accounted for by +8-line header expansion + +33-line `[tillsyn]` block + ~+2 whitespace nudges. No silent body drift.
- E3 (P4): `.tillsyn/template.toml:695-696`:
  ```toml
  [tillsyn]
  spawn_temp_root = "os_tmp"
  ```
  Block-comment rationale at lines 660-693 documents the choice + deferred path to `"project"`.
- E4 (P5): `internal/app/dispatcher/bundle.go:246-256` `resolveSpawnTempRoot`:
  ```go
  switch spawnTempRoot {
  case "", SpawnTempRootOSTmp:
      return SpawnTempRootOSTmp, nil
  ...
  ```
  Empty AND `"os_tmp"` both resolve to `SpawnTempRootOSTmp` — observably equivalent. Schema doc at `internal/templates/schema.go:263-281` documents the same. The explicit pin in the dogfood file makes the dogfood semantics observable on inspection without changing runtime behavior.
- E5 (P6, P7): `.gitignore:18-19`:
  ```
  .tillsyn/*
  !.tillsyn/template.toml
  ```
  `git check-ignore -v .tillsyn/template.toml` returned `.gitignore:19:!.tillsyn/template.toml	.tillsyn/template.toml` — negation rule wins. `git status --porcelain .tillsyn/` returns `?? .tillsyn/` — only the re-included file shows as a candidate. Builder's own `git status --ignored --porcelain` evidence (worklog line 520) shows runtime state files all `!!` ignored.
- E6 (P8): Builder worklog line 518 reports `mage ci` GREEN — 2719 pass / 1 pre-existing skip (`TestStewardIntegrationDropOrchSupersedeRejected` — same skip seen across all earlier rounds, not F.2.3-introduced) / 24/24 packages green / all ≥ 70% coverage. Trust the builder claim per spawn-prompt directive.
- E7 (P9): `git ls-files .tillsyn/template.toml` returns empty (file not yet staged); `git ls-files --others --exclude-standard .tillsyn/` returns `.tillsyn/template.toml` (file is a tracked-eligible candidate). The file is NOT yet `git add`-ed — builder explicitly avoids commit per spawn-prompt rules. Acceptance #3 is "tracked / tracked-eligible" — the latter is satisfied.
- E8 (P10): Builder worklog § "Droplet F.2.3 — Round 1" includes Files touched (4 files), spawn_temp_root choice rationale, Targets run, Design notes (5 bullets), Falsification-mitigation status (F1/F2/F3), Hylla feedback (`N/A — task touched only non-Go files`). Complete per the WORKFLOW Phase 4 contract.

### Trace Coverage

1. **Acceptance #1 (file exists, valid v1 schema, header names tillsyn self-host, body matches default-go.toml):** P1 ∧ P2 ∧ P3 → met by E1 + E2.
2. **Acceptance #2 (`mage ci` green):** P8 → met by E6.
3. **Acceptance #3 (file is tracked / tracked-eligible):** P9 → met by E7. Tracked-eligible (not yet staged); orchestrator stages on commit.
4. **Acceptance #4 (gitignore correctness; `template.toml` not ignored):** P6 ∧ P7 → met by E5. Note: spec mitigation F3 said "existing rule is `.tillsyn/spawns/`" pre-droplet — that was wrong (actual rule was `.tillsyn/`). Builder identified the gap, refactored to the canonical pattern, documented the correction in worklog § ".gitignore" (line 502) and § "Falsification-mitigation status F3" (line 534). Forthright self-correction, not drift.
5. **`spawn_temp_root` matches dispatcher default:** P4 ∧ P5 → met by E3 + E4. Empty and `"os_tmp"` are observably equivalent; explicit pin makes the dogfood policy inspectable.
6. **Worklog completeness:** P10 → met by E8.

### Conclusion

PASS. F.2.3 round 1 satisfies every acceptance criterion with evidence pinned to file content + dispatcher source + git surface state. The two judgment calls — (1) `"os_tmp"` over `"project"` for `spawn_temp_root`, (2) `.gitignore` refactor instead of relying on the (incorrect) spec mitigation F3 — are both well-reasoned, documented in worklog, and tightly scoped. The byte-faithful body copy with intentional header + tail adjustments matches the spec's "BYTE-IDENTICAL copy ... future drift is intentional, drop-tracked" framing exactly.

### Unknowns

- U1 — `.tillsyn/template.toml` is not yet `git add`-ed. Acceptance #3 admits "tracked-eligible" so this is not a finding against F.2.3, but the orchestrator MUST stage the file during the drop's commit step (gitignore won't block, but the file won't appear in the next PR diff unless explicitly staged). Routed in QA summary back to orchestrator.
- U2 — F.2.3's self-host file sits inert until F.1.2 (filesystem walk) ships. This is acknowledged in the spec ("landing F.2.3 first means the file sits unused until F.1.x activates it. Acceptable.") and in the worklog design notes. Not a finding.

### Hylla Feedback

N/A — droplet under review touched only non-Go files (TOML + dotfile + workflow MDs). Hylla is Go-only today per project memory `feedback_hylla_go_only_today.md`. All evidence resolved via `Read` / `Bash` (`git ls-files`, `git status --porcelain`, `git check-ignore -v`) / file content inspection. No Hylla query was attempted, so no miss to log.

---

## Droplet A.4 — Round 1

**Reviewer:** go-qa-proof-agent. **Date:** 2026-05-05. **Verdict:** PASS.

### Premises

1. `ErrInvalidMetadataOutcome` declared in `internal/domain/errors.go` with comprehensive doc.
2. Guard in `Service.MoveActionItem` rejects empty/whitespace/non-enum outcome on `→failed`.
3. Guard placed AFTER terminal-state guard, BEFORE column move (no partial-mutation race).
4. `→complete` does NOT require outcome (asymmetric).
5. Idempotent `failed→failed` carve-out preserves pre-A.4 data.
6. Strict closed-enum `{failure, blocked, superseded}` rejects `success` per master PLAN cross-cutting decision.
7. Two pre-existing tests + one adapter test fixed to populate `Outcome="failure"` before move.
8. New table-driven test added (acceptance #5: 5+ rows; spec lists 7; impl ships 10).
9. R-A.4-1 refinement raised: dispatcher's `applyCrashTransition` / `transitionToFailed` violate "metadata-before-move" order and would fail under the new guard in production.

### Evidence

- `internal/domain/errors.go:61-72` — `ErrInvalidMetadataOutcome` sentinel + 12-line doc-comment covering closed enum, asymmetry, carve-out.
- `internal/app/service.go:1116-1141` — terminal-state guard at 1116; A.4 guard at 1119-1141 with case-insensitive match (`strings.TrimSpace + strings.ToLower`); column move (`actionItem.Move`) at 1159. Wrapped error format `%w: ... (got %q)` preserves raw caller value for debug logs.
- `internal/app/service_test.go:5150-5320` — `TestMoveActionItemFailedTransitionRequiresOutcome` 10-row table. Each rejection row asserts both `errors.Is(err, ErrInvalidMetadataOutcome)` AND post-rejection lifecycle state unchanged via `GetActionItem` re-read (proving guard fires before column write).
- `internal/app/service_test.go:4981` + `:5023` — pre-existing `TestMoveActionItemToFailedUsesMarkFailedCapability` and `TestMoveActionItemToFailedSkipsCompletionCriteria` updated to set `Outcome: "failure"`.
- `internal/adapters/server/common/app_service_adapter_lifecycle_test.go:1006` — adapter test updated to set `Outcome: "failure"`.
- `internal/adapters/server/common/app_service_adapter_mcp.go:1193-1222` — `validateMetadataOutcome` doc-comment extended with A.4 cross-reference (lines 1197-1206); function body unchanged (per acceptance criterion).
- Mage: `mage testPkg ./internal/app` 408/408, `./internal/adapters/server/common` 160/160, `./internal/domain` 303/303, `mage testFunc ./internal/app TestMoveActionItemFailedTransitionRequiresOutcome` 11/11 (counts subtests).

### Trace or cases

1. New `→failed` w/ empty outcome → `outcome == ""` → switch default → `ErrInvalidMetadataOutcome`. Lifecycle stays `in_progress`. **Verified row 1.**
2. Whitespace outcome `"   "` → `TrimSpace` → "" → reject. **Row 2.**
3. `success` on `→failed` → not in `{failure, blocked, superseded}` → reject. **Row 3.** Strict-enum check enforced.
4. Garbage outcome → reject. **Row 4.**
5. `failure` / `blocked` / `superseded` accepted → state flips. **Rows 5-7.**
6. `Failure` (mixed case) → `ToLower` → "failure" → accept. **Row 8.**
7. `→complete` w/ empty outcome → `toState != StateFailed` → guard skipped → succeed. **Row 9.**
8. `→in_progress` → guard skipped. **Row 10.**
9. Idempotent failed→failed: `fromState == StateFailed` → carve-out skips guard → succeed (pre-existing `TestMoveActionItemFromFailedIdempotentAllowed` still passes per builder note).

### Conclusion

PASS. All 7 acceptance criteria met:
- AC#1 (wrapped `ErrInvalidMetadataOutcome` on empty post-trim): met + extended to closed-enum.
- AC#2 (placement after terminal-state guard, before column move): verified at lines 1116→1119→1159.
- AC#3 (`→complete` does not require outcome): pinned by row 9.
- AC#4 (dispatcher pattern preserved): A.4 itself preserves the documented order; R-A.4-1 correctly raises that the dispatcher's CURRENT impl violates it (orchestrator-routed, not a builder defect).
- AC#5 (5+ new tests): 10 rows shipped.
- AC#6 (`mage test-pkg ./internal/app -race`): 408/408 green.
- AC#7 (`mage ci` clean on A.4 surface): builder's `mage ci` block at `formatCheck` is on `internal/adapters/server/mcpapi/extended_tools_test.go` — outside A.4's declared paths and traceable to a sibling droplet. A.4's own packages all pass.

Master PLAN cross-cutting decision (`reject success on →failed`): IMPLEMENTED. Verified at row 3 + service.go:1136 closed switch.

### Findings

- **F1 (minor doc-drift, NOT a defect).** Worklog claims "11-row table-driven test"; actual count is 10 rows. Coverage still vastly exceeds spec's 5-row floor and includes every acceptance row. Builder may correct the worklog count opportunistically; no rebuild required.
- **F2 (informational, R-A.4-1 acknowledged).** Builder correctly raised R-A.4-1: dispatcher's `internal/app/dispatcher/monitor.go:applyCrashTransition` (~351-371) and `dispatcher.go:transitionToFailed` (~639-664) call `MoveActionItem(... → failed)` BEFORE setting `metadata.outcome`. Production runs would now hit `ErrInvalidMetadataOutcome`. The dispatcher tests stub the Service so this is not caught by the test suite. Routed correctly to orchestrator for refinement-list closeout entry; out of A.4's declared paths.

### Missing Evidence

None. Spec, code, tests, and worklog all align.

### Hylla Feedback

N/A — A.4 review touched only Go files but Drop 4c.5 is in filesystem-MD coordination mode and Hylla is stale post-Drop-4c-merge. Per spawn directive ("NO Hylla calls"), no Hylla query attempted; all evidence resolved via `Read` + `rg` on disk + git diff. Project memory `feedback_hylla_go_only_today.md` permits the Go-on-disk fallback for stale-ingest windows; no miss to log.

### TL;DR

- T1 — PASS. Guard at `service.go:1133-1141` correctly placed between terminal-state guard (1116) and column move (1159); strict closed-enum {failure, blocked, superseded} with `TrimSpace + ToLower`; idempotent failed→failed carve-out via `fromState != StateFailed`; asymmetric (complete unaffected).
- T2 — `success`-on-failed rejection (master PLAN cross-cutting decision) implemented and pinned by test row 3.
- T3 — 10-row table covers all 7 spec rows + 3 bonus rows (success rejected, garbage rejected, mixed-case accepted); each rejection row verifies state-unchanged via GetActionItem re-read.
- T4 — Pre-existing tests `TestMoveActionItemToFailedUsesMarkFailedCapability` (4981), `TestMoveActionItemToFailedSkipsCompletionCriteria` (5023), and adapter `TestMoveActionItemStateToFailed` (1006) all correctly updated to set `Outcome: "failure"` before move.
- T5 — Worklog claims "11-row" table; actual count is 10. Doc nit, not a defect.
- T6 — R-A.4-1 correctly raised: dispatcher's crash-recovery paths violate metadata-before-move order; orchestrator-routed for closeout refinements list.

## Droplet A.2 — Round 1

**Reviewer:** go-qa-proof-agent
**Date:** 2026-05-05
**Verdict:** PASS

### Trace Coverage

**Check 1 — Acceptance #1: `bindArgumentsStrict` exists with the documented signature.** COVERED. `internal/adapters/server/mcpapi/strict_decode.go:64` declares `func bindArgumentsStrict(req mcp.CallToolRequest, target any) error` — exact signature the spec mandates and the same shape `mark3labs/mcp-go.CallToolRequest.BindArguments` exposes. Doc-comment block at lines 37-63 names the parity contract (non-nil pointer guard, json.RawMessage fast-path, re-marshal fallback), the null-value preservation contract for A.1's pointer-sentinel fields, and the error shape `unknown field %q on tool %q: %w`.

**Check 2 — Acceptance #2: implementation strategy matches spec.** COVERED. Lines 64-94 of `strict_decode.go` execute the spec's strategy:
- Line 65-67: non-nil pointer guard (mirrors `BindArguments` wording).
- Line 69: trims `req.Params.Name` for the error-surface tool name.
- Lines 72-80: fast-path on `json.RawMessage`; otherwise `json.Marshal(req.Params.Arguments)` re-marshal.
- Lines 85-87: `json.NewDecoder(bytes.NewReader(data))` → `dec.DisallowUnknownFields()` → `dec.Decode(target)`.
- Lines 88-90: on rejection, `unknownFieldName(err)` extracts the offending key from the std-lib's `json: unknown field "<key>"` message via the `jsonUnknownFieldPrefix` constant + `strconv.Unquote`, then wraps as `fmt.Errorf("unknown field %q on tool %q: %w", fieldName, toolName, errUnknownField)`. Defensive fallback path at lines 124-127 handles any future std-lib format drift.

**Check 3 — Acceptance #3: all 21 production `BindArguments` call sites swapped.** COVERED. `rg "BindArguments\(" internal/adapters/server/mcpapi/handler.go internal/adapters/server/mcpapi/handoff_tools.go internal/adapters/server/mcpapi/extended_tools.go` returns ZERO non-strict matches (the only hits are inside `bindArgumentsStrict`'s own doc-comment). `rg "bindArgumentsStrict\(" internal/adapters/server/mcpapi/ -g '!*_test.go'` returns exactly 21 production sites: 5 in `handler.go` (lines 166, 642, 670, 700, 722), 5 in `handoff_tools.go` (57, 111, 133, 169, 201), 11 in `extended_tools.go` (483, 806, 1815, 1892, 1917, 1946, 1965, 1985, 2004, 2025, 2083). Counts match the spec's 5+5+11 = 21 exactly.

**Check 4 — Acceptance #4: error flows through `invalidRequestToolResult` unchanged.** COVERED. `invalidRequestToolResult` defined at `extended_tools.go:2183-2188` returns `mcp.NewToolResultError("invalid_request: " + err.Error())`. Every swap site uses the pattern `if err := bindArgumentsStrict(req, &args); err != nil { return invalidRequestToolResult(err), nil }` — verified by sampling all three files (handler.go:166-168, handoff_tools.go:57-59, extended_tools.go:483-485, 806-808, 2083-2085). Surface text becomes `invalid_request: unknown field "<key>" on tool "<name>"` — single canonical prefix because the helper deliberately omits its own `invalid_request:` prefix to avoid double-stamping (builder's design decision documented in worklog and verified by `TestHandlerExpandedToolRejectsUnknownJSONKeys` assertions).

**Check 5 — Acceptance #5: unknown-key tests across at least 3 tools.** COVERED. `extended_tools_test.go:3556` defines `TestHandlerExpandedToolRejectsUnknownJSONKeys` with three table cases that exercise one tool from each of the three production source files end-to-end via `httptest.NewServer(handler)`:
- `till.project` (extended_tools.go) with `made_up_key: x` — line 3567.
- `till.auth_request` (handler.go) with `ttl: 8h` — line 3580.
- `till.handoff` (handoff_tools.go) with typo'd `tartget: typo` — line 3593.
Each case asserts `isError=true` (3619), surface text starts with `invalid_request:` (3623), contains `unknown field` (3626), names the offending field with quotes (3629), and names the tool with quotes (3632). Spec Test Scenarios rows 2 / 4 / 5 are all covered.

**Check 6 — Acceptance #6: `mage test-pkg ./internal/adapters/server/mcpapi` passes.** COVERED via builder claim. Worklog reports 191/192 (1 pre-existing skip) + `mage ci` 2749 passed across 24 packages, mcpapi at 73.9% coverage. Per spawn directive ("trust 2749 pass claim") not re-executed.

**Check 7 — Schema-vs-struct gap fixes.** COVERED. The spec called out 4 fixes; builder identified 6 (4 reactive + 4 proactive — overlap of 2 between the lists). Verified each `AuthContextID` insertion via direct read:
- `attentionItemMutationArgs` (handler.go:582-606) — `AuthContextID` field at line 602 with explanatory comment crosslinking to A.2 + `withMCPToolAuthRuntime`.
- `handoffMutationArgs` (handoff_tools.go:70-100) — `AuthContextID` at line 96, same pattern.
- `capabilityLeaseMutationArgs` (extended_tools.go:149-176) — comment block at lines 167-169, field follows.
- `till.project` anonymous struct (extended_tools.go:458-481) — `AuthContextID` at line 478 with explanatory comment 475-477.
- `handleActionItemOperation` anonymous struct (extended_tools.go:745-805) — `AuthContextID` at line 801 with comment 797-800.
- `till.comment` anonymous struct (extended_tools.go:2060-2082) — both `Operation` (line 2065, with explanatory comment 2061-2064) AND `AuthContextID` (line 2078, comment 2074-2077). The `Operation` field is declared-only; the handler reads via `req.GetString("operation", "")` at line 2098 (preserves prior behavior).

All six insertions carry rationale comments cross-linking A.2 + `withMCPToolAuthRuntime`. None are dead code by accident — each tool's schema declares the corresponding `mcp.WithString(...)` key, so without the struct-side mirror the strict decoder rejects the tool's own declared key.

**Check 8 — A.1 wire-shape preservation (Q-A-1).** COVERED. `strict_decode_test.go:66` defines `TestBindArgumentsStrictPreservesNullPointer` exercising `{"operation":"update","description":null,"title":null,"labels":null}` against a fixture struct that mixes plain-string and post-A.1 pointer-sentinel fields (`Title *string`, `Description *string`, `Labels *[]string`). Assertions: `bindArgumentsStrict` returns nil error; each pointer field decodes to typed nil; `Operation == "update"` survives. This pins Q-A-1's plan-QA falsification concern — `DisallowUnknownFields` is orthogonal to value-type checking, so null on a known pointer-shape field is accepted exactly as bare `json.Unmarshal` would handle it. Round-trip proof is end-to-end at the helper boundary, which is sufficient since every production swap site goes through this helper.

**Check 9 — Worklog completeness + R-A.2-1 + R-A.2-2 raised.** COVERED.
- Worklog at `BUILDER_WORKLOG.md` § "Droplet A.2 — Round 1" (lines 593-655) contains: date + builder + source-spec pointer; Files-touched (production) with each struct field gap fix line-cited; Files-touched (tests) listing both new test files; Stale-fixture findings paragraph documenting the 4 reactive + 4 proactive symmetry fixes; Targets-run with specific counts (191 passed in mcpapi pkg + 2749 in `mage ci`); Design decisions explaining the single-`invalid_request:` prefix choice, package-internal sentinel rationale, std-lib error-format-prefix matching, per-tool struct contract; Falsification-mitigation status block; Cross-droplet coordination notes for A.1 / A.3 / A.4 / F.3.x; Hylla-feedback `None — Hylla unused` block per spawn directive.
- **R-A.2-1 (schema/struct symmetry doc):** raised in "Unknowns routed back to orchestrator" at line 654 — recommends adding a per-tool checklist item to `CLI_ADAPTER_AUTHORING.md` or new `MCP_TOOL_AUTHORING.md` requiring every `mcp.WithString` schema declaration to have a matching JSON-tagged struct field.
- **R-A.2-2 (`till.comment` Operation declared-not-read):** raised at line 655 — flags that `Operation` is now on the typed struct but the handler still reads via `req.GetString("operation", "")`. Recommends a small follow-up droplet to unify the read-from-typed-struct pattern across all tools.

### Findings

None. All nine checks landed clean. Builder's claim aligns with on-disk evidence at every checkpoint; `mage ci` 2749 passed is consistent with the swap count + struct field additions + new tests.

### Missing Evidence

None. Spec, code, and tests align with the worklog narrative and the surface-text contract verified end-to-end.

### Conclusion

PASS. A.2 ships the spec-mandated `bindArgumentsStrict` helper with documented signature, implements the spec's exact decode strategy (re-marshal → `DisallowUnknownFields` → field-name extraction via stable std-lib prefix), swaps all 21 production call sites with zero residual `BindArguments(` matches in production source, preserves A.1's pointer-sentinel null-handling via a dedicated regression test, and surfaces the 6 schema-vs-struct gaps with line-cited rationale comments. End-to-end test coverage at three tools (one per source file) hits every Acceptance Test Scenarios row the spec listed (typo'd key, unknown field, deep tool-name surface). The design decision to omit the `invalid_request:` prefix in the helper (so `invalidRequestToolResult` adds the single canonical prefix) is correct and verified by the new test's assertion shape.

### Hylla Feedback

N/A — A.2 review touched only Go files but Drop 4c.5 is in filesystem-MD coordination mode and Hylla is stale post-Drop-4c-merge. Per spawn directive ("NO Hylla calls"), no Hylla query attempted; all evidence resolved via `Read` + `rg` on disk + builder worklog cross-reference. Project memory `feedback_hylla_go_only_today.md` permits the Go-on-disk fallback for stale-ingest windows; no miss to log.

### TL;DR

- T1 — PASS. `bindArgumentsStrict` shipped with documented signature `(mcp.CallToolRequest, any) error` at `strict_decode.go:64`; spec-exact decode strategy via `json.NewDecoder + DisallowUnknownFields` with stable-prefix field-name recovery; package-internal `errUnknownField` sentinel for assertion clarity.
- T2 — All 21 production `BindArguments` call sites swapped (5+5+11 = 21 in handler.go + handoff_tools.go + extended_tools.go); zero residual non-strict matches in production code; test files appropriately retain `BindArguments` (none actually do — verified zero residuals total in production paths).
- T3 — Surface error flows through `invalidRequestToolResult` exactly as today; helper deliberately omits its own `invalid_request:` prefix to avoid double-stamping (single-prefix design verified end-to-end by the new `TestHandlerExpandedToolRejectsUnknownJSONKeys` assertions).
- T4 — Three end-to-end tests (one tool per source file) plus eight helper-level unit tests including null-pointer preservation (Q-A-1 mitigation), multiple-unknown-keys-stop-at-first, nil/empty-args parity with `BindArguments`, non-pointer/nil target diagnostics, raw-message fast-path, and `unknownFieldName` parser edge cases.
- T5 — 6 schema-vs-struct gap fixes (`AuthContextID` on attention/handoff/lease/project/action-item/comment, plus `Operation` on comment) all carry rationale comments cross-linking A.2 + `withMCPToolAuthRuntime`. None are accidental dead code; each mirrors a `mcp.WithString` schema declaration that the strict decoder would otherwise reject.
- T6 — A.1 wire-shape preservation pinned by `TestBindArgumentsStrictPreservesNullPointer` (null on pointer-shape fields decodes to typed nil; strict mode does not reject — orthogonal to `DisallowUnknownFields`).
- T7 — Worklog complete with file inventory, target results, design rationale, falsification status, cross-droplet notes; R-A.2-1 (schema/struct symmetry doc invariant) and R-A.2-2 (`till.comment` Operation declared-not-read pattern) both routed for orchestrator's closeout list.

## Droplet E.3 — Round 1

**Reviewer:** go-qa-proof-agent (filesystem-MD mode, opus, 2026-05-05).
**Source spec:** `THEME_CE_PLAN.md` § "E.3 — Conflict detector: assert both file+package overlap entries + path canonicalization doc".
**Builder claim:** done — `mage test-pkg ./internal/app/dispatcher` 356/356 PASS; doc + test extension only; A13 untouched.
**Verdict:** PASS.

### Acceptance verification

1. **A1 — `TestDetectorFindsFileOverlapBetweenSiblings` extended with independent presence loops, NOT length-based.** Verified at `internal/app/dispatcher/conflict_test.go:56-124`. The test now contains TWO independent `for i := range overlaps` presence loops: lines 85-91 select the file overlap into `fileGot`, lines 105-111 select the package overlap into `packageGot`. No `len(overlaps) == 2` or equivalent length assertion appears. Failure mode names the missing kind (`"DetectSiblingOverlap() returned no file overlap"` / `"... no package overlap"`), matching the spec's falsification mitigation #1 verbatim. Comment block at lines 79-84 explicitly documents the design choice ("NOT via len(overlaps) == 2").
2. **A2 — `OverlapValue` doc-comment extended with path canonicalization contract.** Verified at `internal/app/dispatcher/conflict.go:89-99`. The struct-field comment for `OverlapValue` now contains: "Path canonicalization is the planner's / walker's responsibility upstream — the detector does no normalization beyond `domain.NewActionItem`'s trim/dedupe. Two siblings declaring `\"./a/b.go\"` and `\"a/b.go\"` will NOT register as overlapping; the upstream caller MUST normalize before handing items to the detector." Names planner AND walker as upstream owners; uses spec's exact `./a/b.go` / `a/b.go` worked example for grep symmetry.
3. **A3 — `mage test-pkg ./internal/app/dispatcher` green.** Trusted per spawn prompt: builder reports 356/356 PASS (1.67s, race enabled). Worklog corroborates with the `mage test-func ./internal/app/dispatcher TestDetectorFindsFileOverlapBetweenSiblings` (1.32s, race enabled, green) plus `mage format` clean.
4. **A4 — A13 (concurrent `InsertRuntimeBlockedBy` single-flight) NOT touched.** Verified by reading `conflict.go:271-351` (`InsertRuntimeBlockedBy` body): no single-flight wrapper, no `sync.Mutex`/`sync.Map` introduced; the existing comment at lines 286-293 about non-atomic `Update + Attention` coupling is unmodified. Worklog files-touched list (lines 666-669) names only `conflict.go` (doc-comment), `conflict_test.go` (test), `THEME_CE_PLAN.md` (state row), `BUILDER_WORKLOG.md` (this entry) — no `InsertRuntimeBlockedBy` body edits. Falsification mitigation #2 explicitly satisfied.
5. **A5 — Worklog complete.** `BUILDER_WORKLOG.md:657-697` carries the full Round 1 entry: author, source spec, state-at-start (`todo`, blocker E.2 satisfied), state-at-end (`done`), files-touched inventory (4 files), targets-run (3 mage invocations, all green), design notes (5 bullets covering loop shape, variable rename rationale, doc placement choice, worked-example phrasing, no-prod-behavior-change), falsification-mitigation status (mitigations #1 + #2 both green), Hylla feedback section, unknowns section (none).

### Out-of-scope discipline

- **Variable renames `got` → `fileGot` / `want` → `wantFile`** are mechanical disambiguation for the new pair (`packageGot` / `wantPackage`), preserve all existing semantics, and do not alter the file-overlap assertion content (still `OverlapKind: SiblingOverlapFile`, `OverlapValue: "internal/app/dispatcher/walker.go"`, `HasExplicitBlockedBy: false` on `SiblingID: "sibling"`). Scope-bounded.
- **No collateral edits.** `TestDetectorFindsPackageOverlapBetweenSiblings` (lines 128-164) remains untouched; it still uses local `want` scope. `TestDetectorIgnoresNonSiblings` (lines 169-195) untouched. Detector implementation (`DetectSiblingOverlap`, `TieBreakSibling`, `InsertRuntimeBlockedBy`) untouched.

### Certificate

- **Premises:** (P1) test independent-loop shape, no length assertion; (P2) doc-comment extension on OverlapValue names planner/walker; (P3) test-pkg green; (P4) A13 untouched; (P5) worklog complete.
- **Evidence:** `conflict_test.go:79-123` (P1), `conflict.go:89-99` (P2), worklog `mage test-pkg` line + builder claim (P3), `conflict.go:271-351` unchanged + worklog files-touched (P4), `BUILDER_WORKLOG.md:657-697` (P5).
- **Trace:** Read THEME_CE_PLAN.md §E.3 → read BUILDER_WORKLOG.md §E.3 Round 1 → read conflict.go (full file) → read conflict_test.go:1-200 → cross-checked each acceptance bullet against actual file content.
- **Conclusion:** PASS. All five acceptance criteria met; out-of-scope items respected.
- **Unknowns:** None. Builder's worklog "Unknowns routed back to orchestrator" section reads "None"; my own pass found no gaps.

### Hylla feedback

N/A — per spawn prompt directive (filesystem-MD mode, NO Hylla calls).

### TL;DR

- T1 — PASS. Test extension at `conflict_test.go:79-123` uses two independent presence loops (file at 85-91, package at 105-111); no length-based assertion; failure modes name missing kind specifically.
- T2 — Doc-comment extension at `conflict.go:89-99` names planner AND walker as upstream canonicalization owners and uses spec's exact `./a/b.go` / `a/b.go` worked example.
- T3 — `mage test-pkg ./internal/app/dispatcher` 356/356 PASS trusted per spawn prompt; worklog corroborates with `mage test-func` (1.32s, race) + `mage format` clean.
- T4 — A13 (`InsertRuntimeBlockedBy` single-flight) untouched: file inventory in worklog covers only doc + test + workflow MDs; `conflict.go:271-351` body unchanged.
- T5 — Worklog at `BUILDER_WORKLOG.md:657-697` covers all required sections (author, spec, state, files, targets, design notes, falsification status, Hylla feedback, unknowns).

## Droplet F.1.3 — Round 1

**Reviewer:** go-qa-proof-agent (filesystem-MD coordination mode — NO Tillsyn / Hylla calls).
**Date:** 2026-05-05.
**Source spec:** `workflow/drop_4c_5/THEME_F_PLAN.md` § "Droplet F.1.3 — Language-aware embedded resolver" (lines 104-141).
**Builder round under review:** `workflow/drop_4c_5/BUILDER_WORKLOG.md` § "Droplet F.1.3 — Round 1" (lines 699-769).
**Verdict:** PASS.

### Trace Coverage

1. **Acceptance #1 — `LoadDefaultTemplateForLanguage(lang string) (Template, error)` exists with documented signature + `ErrLanguageNotSupported` sentinel.** COVERED.
   - `internal/templates/embed.go:130` — `func LoadDefaultTemplateForLanguage(lang string) (Template, error)`. Doc-comment at lines 96-129 documents the closed enum, drift-guard contract pointing at `internal/domain/project.go` `isValidProjectLanguage`, and the four return-error paths (`fe` deferral, unknown lang, embed-FS open failure, Load chain errors).
   - `internal/templates/embed.go:54` — `var ErrLanguageNotSupported = errors.New("template language not supported")` (exported). Doc-comment at lines 35-53 names the routing contract (`errors.Is` across package boundaries) and the closed-enum drift guard.

2. **Acceptance #2 — `lang == ""` → `default-generic.toml`.** COVERED.
   - `embed.go:133-134` — switch case `""` sets `path = "builtin/default-generic.toml"`.
   - Test pin at `embed_test.go:887-905` `TestLoadDefaultTemplateForLanguage_Generic`: invokes resolver with `""`, asserts `SchemaVersion == SchemaVersionV1` AND the load-bearing `len(AgentBindings) == 0` discriminator (default-go ships 12 bindings; mismatched routing surfaces here).

3. **Acceptance #3 — `lang == "go"` → `default-go.toml`.** COVERED.
   - `embed.go:135-136` — switch case `"go"` sets `path = "builtin/default-go.toml"`.
   - Test pin at `embed_test.go:921-939` `TestLoadDefaultTemplateForLanguage_Go`: invokes resolver with `"go"`, asserts `SchemaVersion == "v1"` AND `len(AgentBindings) == len(allKinds)` (12 — the load-bearing discriminator vs generic).

4. **Acceptance #4 — `lang == "fe"` → wrapped `ErrLanguageNotSupported` per Q1.** COVERED.
   - `embed.go:137-142` — switch case `"fe"` returns `fmt.Errorf("language %q: fe template unavailable; defer until FE adopter materializes: %w", lang, ErrLanguageNotSupported)`. Q1 phrasing matches THEME_F_PLAN.md §3 Note 5.
   - Test pin at `embed_test.go:952-968` `TestLoadDefaultTemplateForLanguage_FERejected`: asserts `err != nil`, `errors.Is(err, ErrLanguageNotSupported)`, message contains literal `"fe"` (`%q`-quoted form), AND zero-value Template return.

5. **Acceptance #5 — Unknown lang → wrapped `ErrLanguageNotSupported` with offending value.** COVERED.
   - `embed.go:143-144` — switch default returns `fmt.Errorf("language %q: outside closed Project.Language enum: %w", lang, ErrLanguageNotSupported)`.
   - Test pin at `embed_test.go:981-997` `TestLoadDefaultTemplateForLanguage_UnknownRejected`: uses canonical `"rust"` fixture; asserts wrapped sentinel via `errors.Is`, message contains `"rust"`, zero-value Template return.

6. **Acceptance #6 — `LoadDefaultTemplate()` preserved as thin wrapper; SEMANTIC SHIFT named loud.** COVERED.
   - `embed.go:92-94` — body is exactly `return LoadDefaultTemplateForLanguage("")`. One-line indirection per spec.
   - Doc-comment at `embed.go:56-91` carries an explicit "SEMANTIC SHIFT (Drop 4c.5 droplet F.1.3)" stamp naming the pre→post behavior change (default-go.toml direct read → generic via wrapper), the affected callers (`seedStewardAnchors` at `internal/app/auto_generate_steward.go:44` + `loadProjectTemplate` Drop-3.14 stub), and the F.2.4 caller-redirect successor. The same-6-STEWARD-seeds-across-both-files rationale for why the materialized output is unchanged today is named at lines 75-81.
   - Cross-test at `embed_test.go:1010-1024` `TestLoadDefaultTemplate_WrapsLanguageEmpty` uses `reflect.DeepEqual(LoadDefaultTemplate(), LoadDefaultTemplateForLanguage(""))` — the strict invariant that pins the wrapper's semantic to its delegated form.

7. **Acceptance #8 — Five new tests landed.** COVERED. Direct file inventory at `embed_test.go`:
   - line 887 `TestLoadDefaultTemplateForLanguage_Generic`.
   - line 921 `TestLoadDefaultTemplateForLanguage_Go`.
   - line 952 `TestLoadDefaultTemplateForLanguage_FERejected`.
   - line 981 `TestLoadDefaultTemplateForLanguage_UnknownRejected`.
   - line 1010 `TestLoadDefaultTemplate_WrapsLanguageEmpty` (the wrapper-equality cross-test). Total: five new tests, all `t.Parallel()`, all asserting acceptance bullets #2–#6.

8. **Acceptance #9 — `mage test-pkg ./internal/templates` passes (386/386).** COVERED.
   - Worklog at `BUILDER_WORKLOG.md:730-733` reports `386 passed / 0 failed / 0 skipped` (initial run + post-format rerun). Arithmetic checks against F.2.2's 381-test baseline: 381 prior + 5 new = 386. Trusted per spawn prompt's "Builder F.1.3 returned green: `mage test-pkg ./internal/templates` 386/386 PASS" verbatim.
   - The full Load() validator chain (version pre-pass, strict decode, validateMapKeys, validateChildRuleKinds, validateChildRuleCycles, validateChildRuleReachability, validateGateKinds, validateAgentBindingEnvNames, validateAgentBindingContext, validateAgentBindingToolGating, validateTillsyn) ran inside both new resolver tests via `Load(f)` and accepted both files.

9. **Test-helper rewire — `loadDefaultOrFatal` + `TestDefaultTemplateGoLoadsCleanly` use `LoadDefaultTemplateForLanguage("go")`.** COVERED.
   - `embed_test.go:32` — `tpl, err := LoadDefaultTemplateForLanguage("go")` (was `LoadDefaultTemplate()`). Doc-comment at lines 21-29 explicitly names the SEMANTIC SHIFT rationale: post-F.1.3 `LoadDefaultTemplate()` returns generic, and the catalog-shape assertions in this file (12 agent bindings, gates, context blocks, STEWARD-owned kinds, opus-builders rule, prohibition-allow-list shape) target the GO template specifically. The rewire is the ONLY way the existing ~14 catalog-shape tests survive the wrapper pivot.
   - `embed_test.go:51-58` — `TestDefaultTemplateGoLoadsCleanly` body invokes `LoadDefaultTemplateForLanguage("go")` directly. Doc-comment at lines 39-47 documents the F.2.1 rename (`TestDefaultTemplateLoadsCleanly` → `TestDefaultTemplateGoLoadsCleanly`) and the F.1.3 rewire to the language-explicit form.
   - Spot-checked downstream tests via `loadDefaultOrFatal`: `TestDefaultTemplateAgentBindingsCoverAllKinds` (line 374), `TestDefaultTemplateBuildersRunOpus` (line 396), `TestDefaultTemplateLoadsWithGates` (line 500), the context-seeded suite (lines 661-873) — all pull through `loadDefaultOrFatal` and thus through `LoadDefaultTemplateForLanguage("go")`. The 386/386 PASS confirms no regression.

10. **Worklog completeness — including documented SEMANTIC SHIFT.** COVERED. `BUILDER_WORKLOG.md` § "Droplet F.1.3 — Round 1" (lines 699-769) contains:
    - Author + opus model + filesystem-MD mode + spec pointer (lines 701-705).
    - Files-touched (production) section (lines 707-714) detailing the new sentinel, the new function with the closed-enum switch, the wrapper rewrite, the import additions, and the doc-comment cross-reference to `domain.Project.Language` + F.2.4 successor.
    - Files-touched (tests) section (lines 716-727) detailing the five new tests + the helper rewire + the `TestDefaultTemplateGoLoadsCleanly` body update.
    - Targets-run section with the 386/386 PASS count + `mage formatCheck` cycle (lines 729-733).
    - Production-caller-status section (lines 735-739) verifying that the SEMANTIC SHIFT does not change the materialized seed set today (same 6 STEWARD seeds across both files) and naming the pre-existing `internal/app` failure as out-of-scope.
    - Design-decisions section (lines 741-748) covering the exported sentinel rationale, the switch-vs-map choice, the `%q` format choice, the thin-wrapper indirection, the SEMANTIC SHIFT doc-comment stamp, and the embed.FS close idiom.
    - Falsification-mitigation status section (lines 750-754) walking F1/F2/F3 from the spec.
    - Cross-droplet coordination section (lines 756-760) naming F.2.4, F.1.1, and F.5.x downstream linkages.
    - Hylla-feedback section (line 762-764) with the per-spawn-prompt "NO Hylla calls" justification.
    - Unknowns-routed-back section (lines 766-769) flagging the wrapper-deprecation question and the pre-existing `internal/app` test failure.
    - THEME_F_PLAN.md droplet F.1.3 heading shows `**State:** done (round 1)` at line 106.

### Findings

None. All ten check items land clean. The closed-enum switch is implemented as the spec's preferred shape (switch over map for distinct error-message phrasing), the SEMANTIC SHIFT is named loud in three places (production doc-comment, helper-rewire doc-comment, builder worklog), and the wrapper-equality cross-test is the strict regression net for any future drift between the two call paths. The five new tests cover the four acceptance-listed scenarios PLUS the spec-mandated wrapper-equality cross-test (#6) — exact match to spec's "5 new tests" tally.

### Hylla Feedback

N/A — F.1.3 review touched only Go-eligible files (`embed.go`, `embed_test.go`) plus workflow MDs. Per spawn-prompt directive "filesystem-MD coordination mode. NO Hylla calls" all evidence resolved via Read on the active worktree files. Per project rule "Hylla Indexes Only Go Files Today" the Go-source review would normally favor Hylla; the override is drop-specific (Hylla stale across the post-Drop-4c-merge state), not a Hylla ergonomics signal.

### Conclusion

PASS. F.1.3 ships the language-aware resolver precisely as scoped: closed-enum switch over `""` / `"go"` / `"fe"` / default; exported `ErrLanguageNotSupported` sentinel for cross-package `errors.Is` routing; thin one-line wrapper preservation that re-routes `LoadDefaultTemplate()` to generic per the SEMANTIC SHIFT; helper + canary-test rewire to keep all existing Go-shape catalog assertions targeting default-go.toml. `mage test-pkg ./internal/templates` 386/386 PASS = 381 prior + 5 new — arithmetic checks against F.2.2's baseline. Worklog is complete with explicit SEMANTIC SHIFT documentation, falsification-mitigation walk, downstream coordination notes, and routed unknowns.

### TL;DR

- T1 — PASS. Resolver function at `embed.go:130` + sentinel at `embed.go:54` match acceptance #1 surface; closed-enum switch at lines 132-145 covers acceptance #2-#5 paths.
- T2 — Wrapper preservation at `embed.go:92-94` (one-line indirection) + SEMANTIC SHIFT doc-comment at lines 56-91 + cross-test at `embed_test.go:1010-1024` (`reflect.DeepEqual`) pin acceptance #6.
- T3 — Five new tests landed at `embed_test.go:887`, 921, 952, 981, 1010; all assertions match spec scenarios.
- T4 — Helper rewire at `embed_test.go:32` (`loadDefaultOrFatal` → `"go"`) + canary at `embed_test.go:51` keep ~14 existing catalog-shape tests targeting default-go.toml; 386/386 PASS confirms no regression.
- T5 — Worklog at `BUILDER_WORKLOG.md:699-769` covers all required sections including the SEMANTIC SHIFT documentation, F.2.4 caller-redirect linkage, and routed unknowns.

## Droplet D.2 — Round 1

**Date:** 2026-05-05.
**Reviewer:** go-qa-proof-agent (filesystem-MD coordination mode).
**Source spec:** `workflow/drop_4c_5/THEME_BD_PLAN.md` § "Droplet D.2 — Accumulated Vet / Gopls / `mage ci` Hint Sweep".
**Verdict:** **PASS.**

### Acceptance Trace

| # | Acceptance criterion | Status | Evidence |
| --- | --- | --- | --- |
| 1 | `D2_HINT_SWEEP.md` exists with `## Captured Hints` + `## Fix-Now Bucket` + `## Routed-to-Refinement Bucket` | **MET** | File present at `workflow/drop_4c_5/D2_HINT_SWEEP.md` (196 lines). § 2 "Captured Hints" + § 3 "Fix-Now Bucket" + § 4 "Routed-to-Refinement Bucket" all present (with deeper subsections); § 1 methodology + § 5 verification + § 6 summary table + § 7-9 file lists/references included. |
| 2 | Each Fix-Now entry maps to an inline fix | **MET** | (a) `instructions_explainer.go:354` + `:358` — verified via `Read`: both call sites now invoke `capitalizeASCIIScope(string(actionItem.Scope))`; new helper defined at lines 660-669; no remaining `strings.Title` in the file. (b) `monitor_test.go:468` + `:474` — verified via `git diff HEAD`: both `for i := 0; i < n; i++` lines now read `for i := range n`. |
| 3 | Routed-to-Refinement entries carry rationales | **MET** | D2-R1 (`D2_HINT_SWEEP.md` § 4.1): scope-creep into Drop-1 R1 (`internal/tui/model.go` 22kLOC split list) + acceptance-#5 forbidden file (`cmd/till/main_test.go`); follow-up plan = fold into Drop-1 R1 split + standalone refinement droplet for non-tui sites. D2-R2 (§ 4.2): contract-touching ctx-propagation refactor exceeds one-liner; follow-up = Drop 5+ daemon-mode dispatcher polish. Both entries name consumers + cost shape. |
| 4 | Reduced warnings post-fix | **MET (trust builder)** | Spawn-prompt directive: "trust builder." `mage testPkg ./internal/adapters/server/mcpapi` 202/202 + `./internal/app/dispatcher` 356/356; `mage formatCheck` clean. No new warnings introduced; sibling-induced `mage ci` failure attributed to A.3 (`client_type` test-fixture omission), not D.2 — D.2 did not touch `internal/app/auth_requests*`. |
| 5 | No fix touches `cmd/till/main.go`, `cmd/till/main_test.go`, or `internal/app/service.go` for refactor-style cleanup | **MET** | `git diff --stat` of D.2-declared files shows ONLY `internal/adapters/server/mcpapi/instructions_explainer.go` (+18/-2) + `internal/app/dispatcher/monitor_test.go` (+2/-2). The 3 forbidden files appear in `git status` only via concurrent-sibling work (A.1 / A.2 / A.4 / B.1 / B.2), not D.2. Sweep table § 2.3 explicitly marks `cmd/till/main_test.go:94` as "Routed (forbidden file per acceptance #5)" + the 5 `internal/tui/model.go` sites (Drop-1 R1 territory) as Routed. |
| 6 | Coverage stays ≥ 70% on touched packages | **MET** | `D2_HINT_SWEEP.md` § 5.3: `internal/adapters/server/mcpapi` 73.9% (helper has 100% coverage via new test); `internal/app/dispatcher` 76.1% (unchanged — range-int modernization touches no production code). Both above 70% project minimum. |
| 7 | Worklog completeness | **MET** | `BUILDER_WORKLOG.md` § "Droplet D.2 — Round 1" (lines 771-852) carries: source spec, files touched (production / tests / workflow MD splits), targets run, sweep findings (Fix-Now + Routed), sibling-induced failure note, falsification-mitigation status, design decisions, cross-droplet coordination, Hylla feedback, unknowns routed back to orchestrator. |

### Premises / Evidence / Trace / Conclusion / Unknowns

- **Premises:** D.2 must produce sweep MD with 3 sections; apply 4 inline Fix-Now hints (2× `strings.Title` swap + 2× `rangeint`); route remaining hints with rationales; not touch 3 forbidden files; preserve coverage ≥ 70%; pass tests on touched packages.
- **Evidence:** `Read` of `D2_HINT_SWEEP.md`, `instructions_explainer.go`, `instructions_explainer_test.go`, `THEME_BD_PLAN.md`, `BUILDER_WORKLOG.md` § D.2; `git diff HEAD -- internal/app/dispatcher/monitor_test.go` confirming line 468/474 swap; `git diff --stat` confirming D.2's two-file scope; `git status --porcelain` confirming forbidden-file edits attribute to sibling droplets, not D.2.
- **Trace:** Acceptance table above maps every criterion to a concrete artifact line or diff hunk. (a) Helper `capitalizeASCIIScope` defined at `instructions_explainer.go:660-669`; both call sites at 354 + 358 confirmed via direct read of the post-edit file (no `strings.Title` substring remains in file body). (b) `monitor_test.go` diff shows the two-line swap; structural-only change preserves iteration semantics. (c) Test file `instructions_explainer_test.go` ships 10 table-driven cases including the production-shape inputs (`"droplet"` → `"Droplet"`, `"plan"` → `"Plan"`) plus edge cases (empty / single letter / passthrough / leading non-letter / mixed case). (d) `D2_HINT_SWEEP.md` § 4 routes 39 indexed-loop sites + 3 spawn.go TODOs with rationale tied to scope guards.
- **Conclusion:** PASS. All 7 acceptance criteria met. Builder followed scope guards (no forbidden-file refactors), captured the full hint surface (46 distinct items), classified each into Fix-Now / Routed / Ignore with rationale, applied the 4 Fix-Now fixes inline, and shipped a regression-pinning unit test for the new ASCII-helper.
- **Unknowns:** None gating verdict. Methodology adaptation noted (static-grep substituted for `LSP` workspace diagnostics because the subagent's tool list lacks `LSP` and direct `gopls` bash is denied) — builder documented the substitution + flagged the surface ergonomic gap; routed back to orchestrator for tool-list refinement. Sibling-induced `mage ci` failure (A.3's `client_type` test-fixture omission) is correctly attributed to A.3, not D.2.

### Falsification Hooks Considered

- **`strings.Title` lingering elsewhere** — full-file read of `instructions_explainer.go` confirms no `strings.Title` substring remains; the `strings` import is still used by the rest of the file (TrimSpace / EqualFold / Join / Contains), so no dead-import.
- **`for i := range n` compile risk** — Go 1.22+ supports `range int`; `mage testPkg ./internal/app/dispatcher` 356/356 PASS confirms the modernization compiles and runs identically.
- **Test stub vs real assertions** — `instructions_explainer_test.go` ships 10 distinct cases with `t.Fatalf` on mismatch; not a stub.
- **Forbidden-file scope creep** — verified via `git diff --stat`: D.2 touches only 2 Go files, neither in the forbidden list.
- **Three-section sweep artifact** — confirmed via direct `Read`; § 3 ("Fix-Now Bucket") and § 4 ("Routed-to-Refinement Bucket") are the spec-named sections; § 2 ("Captured Hints") satisfies the third spec-named section with deeper sub-tables for stdlib + indexed-loop + TODO + Deprecated + nolint inventories.

### Hylla Feedback

N/A — D.2 review touched only Go source + workflow MDs in filesystem-MD mode (Hylla is Go-only today and stale post-Drop-4c-merge; the spawn-prompt directive forbids Hylla calls). No Hylla query was attempted, so no miss to log. The builder's worklog flagged the absent `LSP` MCP tool in the subagent surface as a methodology friction point — surfaced once already; not double-raising here.

### TL;DR

- T1 — PASS. All 7 acceptance criteria met.
- T2 — Sweep artifact at `D2_HINT_SWEEP.md` ships the 3 required sections + methodology + verification + summary table.
- T3 — 4 Fix-Now sites verified inline (`strings.Title` × 2 → `capitalizeASCIIScope` at `instructions_explainer.go:354,358`; rangeint × 2 at `monitor_test.go:468,474` confirmed via `git diff`).
- T4 — Routed entries D2-R1 (39 sites in 16 files) and D2-R2 (3 spawn.go TODOs) carry scope-guard rationales and named follow-up consumers.
- T5 — No forbidden file edited by D.2 (`cmd/till/main.go` / `main_test.go` / `internal/app/service.go` appear in `git status` only via concurrent-sibling droplets).
- T6 — Coverage 73.9% (mcpapi) + 76.1% (dispatcher), both ≥ 70%.
- T7 — Worklog at `BUILDER_WORKLOG.md:771-852` complete.

## Droplet A.3 — Round 1

**Verdict:** PASS.

**Reviewed:** `workflow/drop_4c_5/THEME_A_PLAN.md` § "A.3" + `BUILDER_WORKLOG.md` § "Droplet A.3 — Round 1" (lines 854-952) + `git diff main` for declared files.

### 1. Findings

- 1.1 **Acceptance #1 (service rejects empty client_type wrapped in `ErrInvalidClientType`):** PASS. `internal/domain/errors.go:56` declares `ErrInvalidClientType = errors.New("invalid client type")` with full A.3 doc-comment cross-referencing `autentauth.ensureClient`. `internal/app/auth_requests.go:236-238` adds the trim-empty guard returning `fmt.Errorf("client_type is required: %w", domain.ErrInvalidClientType)` immediately after the `s.authRequests == nil` configuration guard — correct positioning (before `ParseAuthRequestPath` so the lighter check fires first). The `%w` verb correctly wraps the sentinel for `errors.Is` routing.
- 1.2 **Acceptance #2 (MCP-stdio handler stamps `"mcp-stdio"` regardless of agent input + typed field retained):** PASS. `internal/adapters/server/mcpapi/handler.go:212` hard-codes `ClientType: "mcp-stdio"` on the `common.CreateAuthRequestRequest` literal; the prior `args.ClientType` pass-through is gone. The typed `ClientType string` field on the anonymous args struct (line 156, identified via `rg`) is intentionally retained per the inline rationale comments at lines 113-122 — this is the correct transitional shape (post-A.2 strict-decode would otherwise reject `"client_type"` keys from existing senders).
- 1.3 **Acceptance #3 (CLI stamps `"cli"` everywhere; `--client-type` flag removed):** PASS. Three CLI stamp sites converted to literal `"cli"`: `cmd/till/main.go:2727` (autent IssueSessionInput), `:2743` (audit-trail authSessionPayloadJSON), `:3113` (CreateAuthRequestInput). Both `clientType string` struct fields removed from `issueSessionCommandOptions` / `requestCreateCommandOptions`. Both `Flags().StringVar(..., "client-type", ...)` declarations removed; replaced with explanatory comments (lines 1464, 1709). Defaults `clientID: "till-mcp-stdio"` / `clientName: "Till MCP STDIO"` correctly migrated to `till-cli` / `Till CLI` for self-consistency.
- 1.4 **Acceptance #4 (`client_type` removed from MCP `till.auth_request` schema):** PASS. `mcp.WithString("client_type", ...)` declaration at the prior line 113 area is gone (replaced by a multi-line A.3 invariant comment). The new test `TestAuthRequestToolSchemaApproveAcceptsOnlyDocumentedArgs` augmentation at `handler_test.go:2826-2833` asserts `properties["client_type"]` does NOT exist in the published schema — strict negative-existence regression net.
- 1.5 **Acceptance #5 (Q4 resolution documented):** PASS. Worklog "Design notes" § (lines 913-927) explicitly addresses Q4: cascade subagents inherit `"cli"` via dispatcher's CLI path (Drop 4a Wave 3 W3.1 orch-self-approval); explicit `cli-cascade` deferred to Drop 4d / Drop 5. Forward-documentation row in `TestServiceCreateAuthRequestAcceptsNonEmptyClientType` exercises the future vocabulary.
- 1.6 **Acceptance #6 (existing `ClientType: "mcp-stdio"` tests still pass):** PASS. `mage test-pkg ./internal/app` reports 430/430 (worklog line 904); the existing fixture audit found exactly one failure (`TestServiceClaimAuthRequestRejectsNegativeWaitTimeout` at line 547 — D.2's flagged Unknown), fixed in-droplet with one-line `ClientType: "mcp-stdio"` addition. Per A.2 falsification mitigation #1, all other 30+ test fixtures already passed non-empty values.
- 1.7 **Acceptance #7 (new tests):** PASS. Empty rejection: `TestServiceCreateAuthRequestRejectsEmptyClientType` (table-driven over `""`, `" "`, `"\t\n "`) asserts `errors.Is(err, domain.ErrInvalidClientType)`. Whitespace-only: same test, second/third rows. MCP override: `TestHandlerAuthRequestCreateOverridesAgentSuppliedClientType` (table-driven over `tui` / `spoofed-orch` / `""` / omitted-key) asserts `capture.lastCreate.ClientType == "mcp-stdio"`. CLI stamp: `TestRunAuthRequestCreateStampsCLIClientType` reads `repo.GetAuthRequest` directly (the auth-request human-render does not show client_type — worklog line 893 explains this); `TestRunAuthIssueSessionStampsCLIClientType` parses the display KV. Bonus: two flag-rejection tests (`TestRunAuthRequestCreateRejectsClientTypeFlag` + `TestRunAuthIssueSessionRejectsClientTypeFlag`) defend against future re-add drift.
- 1.8 **Acceptance #8 (`mage ci` clean):** PASS. Worklog line 909 reports all gates green: 430/430 internal/app, 208/208 mcpapi, 241/241 cmd/till, 160/160 common, format/build/coverage all clean.
- 1.9 **D.2 Unknown closure:** PASS. `auth_requests_test.go:550` carries the `ClientType: "mcp-stdio"` fix that D.2's worklog (line 822) flagged. The pre-existing failure surfaced by F.1.3 + D.2 sibling rounds is now resolved within A.3. Worklog cross-references this at lines 941 + 951.
- 1.10 **Worklog completeness + Unknowns routing:** PASS. Worklog covers files / verification / design notes / falsification status / cross-droplet coordination / Hylla feedback / Unknowns. Four Unknowns routed: tool-description-prose deferral (legitimately deferred for plan-QA judgment); `till-cli` clientID default rename (documented breaking-default for future release-notes); D.2 flag closure note; `Till MCP STDIO` display-name fixture cosmetic drift. All four are well-formed routing items, not unresolved hazards.

### 2. Missing Evidence

- 2.1 None. Every acceptance criterion is grounded by a concrete file + line citation in either the production diff or the test diff. The retained `ClientType` typed-field on the args struct (line 156) is the only subtle correctness point — verified directly via `rg ClientType internal/adapters/server/mcpapi/handler.go`.

### 3. Summary

PASS. A.3 closes the asymmetric-validation bug correctly, removes the agent-impersonation knob from the MCP schema, hard-stamps `"cli"` at every CLI auth-request site, and ships forward-and-backward regression nets (positive stamp + negative flag-rejection + schema absence). The retained typed `ClientType` struct field is the correct transitional shape against post-A.2 strict-decode; the inline comments document why the asymmetric "schema dropped, struct retained" pattern is intentional. The D.2-flagged fixture failure is closed in-droplet. Tool-description prose is the one judgment-call deferral and is properly routed for plan-QA.

### TL;DR

- T1 — All 10 checks PASS: sentinel + service guard, MCP stamp + struct retention, CLI stamps + flag removal, schema absence, Q4 doc, existing tests preserved, four new tests + two bonus flag-rejection tests, mage ci green, D.2 Unknown closed, worklog complete with 4 routed Unknowns.
- T2 — Zero missing evidence — every acceptance traces to a file + line citation in the diff.
- T3 — Verdict: PASS. No round-2 work needed unless plan-QA flips the deferred tool-description-prose acceptance interpretation.

## Hylla Feedback

N/A — action item touched only Go files plus workflow MDs in filesystem-MD coordination mode (per spawn prompt: "NO Hylla calls"). Hylla is Go-only-today and stale post-Drop-4c-merge; the per-droplet directive forbids calls in any case.
