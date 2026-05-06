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
