# DROP_2 — Build QA Falsification

(durable; append `## Droplet N.M — Round K` per QA attempt; NEVER `git rm`d)

## Droplet 2.1 — Round 1

**Verdict:** pass
**Date:** 2026-05-01
**Reviewer:** go-qa-falsification-agent (subagent), Round 1

### Summary

Adversarial attack on the builder's claim that deleting the entire `templates/` package
(`templates/builtin/default-go.json`, `templates/builtin/default-frontend.json`,
`templates/embed.go`, plus auto-removed parent dirs) is clean and `mage ci` is green.
14 attack vectors run; 0 blocking counterexamples constructed; 0 nits worth blocking on.
Local `mage ci` reproduces the builder's reported result (1263 tests / 19 packages /
all coverage ≥ 70.0% / build of `./cmd/till` succeeds / exit 0).

### Attack Vectors and Findings

#### 1. Hidden importer hunt — exhaustive

- `git grep "evanmschultz/tillsyn/templates" -- '*.go'` → empty (exit 1 from grep with no matches).
- `git grep "templates.ReadFile" -- '*.go'` → empty.
- `git grep "templates.Files" -- '*.go'` → empty.
- `git grep -E '"github\.com/evanmschultz/tillsyn/templates"' -- '*.go'` → empty.

REFUTED — no Go importer exists at HEAD. Builder's claim that the package was
runtime-dead is correct.

#### 2. Build-tag-gated importer (linux/freebsd/windows)

- `git grep -n "templates" -- '*.go'` returned only unrelated hits:
  - `internal/adapters/server/mcpapi/instructions_explainer.go:112` — string literal
    "missing templates, kinds, or drift state" (instructional prose).
  - `internal/adapters/server/mcpapi/instructions_tool.go:333` — string literal
    "default templates" (instructional prose).
  - `internal/app/service.go:147,148,149,179,1873,1883` — `sanitizeStateTemplates` /
    `defaultStateTemplates` / `cfg.StateTemplates` — local state-template machinery
    in the lifecycle-state config domain, unrelated to the deleted `templates/` package.
- None of these files carry a `//go:build` constraint that would exclude darwin
  (verified by inspecting hits — they are unconstrained Go source). No
  build-tag-gated importer of the deleted package exists.

REFUTED — no Go file in the tree references `evanmschultz/tillsyn/templates` under
any build constraint.

#### 3. Reflection / plugin / runtime use

- `git grep -nE 'reflect\.[A-Za-z]+.*templates|plugin.*templates|"templates\.Files"|"templates\.ReadFile"'` → empty.
- The deleted package only exposed `Files embed.FS` and `ReadFile(name string)`.
  Both were verified absent from the tree as Go-symbol references AND as string
  literals. No reflection or plugin loader could resolve them.

REFUTED — no runtime/reflection path exists.

#### 4. `go.mod` / `go.sum` orphans

- `git grep -n "templates" -- 'go.mod' 'go.sum'` → empty.
- `templates/` was an internal sub-package of `github.com/evanmschultz/tillsyn`,
  so by construction it would not appear in `go.mod` / `go.sum`. Confirmed empty.

REFUTED — no module-graph orphans.

#### 5. `mage` target file references

- `git grep -n "templates" -- magefile.go` → empty.
- No mage target references the deleted `templates/` directory.

REFUTED.

#### 6. CI workflow references

- `git grep -n "templates" .github/workflows/` → empty (only `ci.yml` and
  `release.yml` exist; neither references `templates/`).

REFUTED — no CI step expects `templates/` to exist.

#### 7. Documentation lies — README.md references

- `README.md:298` and `README.md:309` reference `templates/builtin/default-go.json`
  and `templates/builtin/default-frontend.json` as live-link MD prose.
- Builder explicitly deferred these per `workflow/drop_2/PLAN.md:394` ("the surviving
  MD references are not load-bearing for Drop 2") and the Round 2 dev decision in
  PLAN.md `:426` ("trivial in-section MD edits ... single-sentence / single-phrase
  fix"). The README hits are inside a multi-bullet template-content block where the
  surrounding paragraphs cite the deleted file by name multiple times — exactly the
  ambiguity flagged in `PLAN_QA_FALSIFICATION.md:126`. A single-phrase fix is
  ill-defined; whole-paragraph rewrite is out of scope. Builder correctly deferred
  to Drop 3's full template overhaul.
- These references are MD prose, not load-bearing for `mage ci` or any runtime path.
  A user following README setup steps today would not get a missing-file error from
  the deleted JSON because nothing at runtime reads them (verified in Vector #1).

REFUTED — the README references are doc/prose drift, scheduled for Drop 3
cleanup. They do not break Droplet 2.1's acceptance criteria (which explicitly
allows MD references to stay until Drop 3 cleanup, per PLAN.md:54).

#### 8. `mage ci` re-run

Reviewer ran `mage ci` from `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/`
in this falsification session. Captured tail:

```
Test summary
  tests: 1263
  passed: 1263
  failed: 0
  skipped: 0
  packages: 19
  pkg passed: 19
  pkg failed: 0
  pkg skipped: 0

[SUCCESS] All tests passed
  1263 tests passed across 19 packages.
...
Minimum package coverage: 70.0%.
[SUCCESS] Coverage threshold met
[SUCCESS] Built till from ./cmd/till
```

Per-package coverage observed:

- `internal/adapters/livewait/localipc` — 79.4%
- `internal/adapters/embeddings/fantasy` — 90.6%
- `internal/platform` — 78.0%
- `internal/buildinfo` — 100.0%
- `internal/adapters/auth/autentauth` — 73.0%
- `internal/config` — 76.8%
- `internal/adapters/storage/sqlite` — 75.1%
- `internal/domain` — 79.2%
- `internal/adapters/server/httpapi` — 88.4%
- `internal/tui/gitdiff` — 85.1%
- `internal/adapters/server/mcpapi` — 72.3%
- `internal/app` — 71.5%
- `cmd/till` — 76.6%
- `internal/adapters/server/common` — 73.0%
- `internal/tui` — 70.0%

Every package ≥ 70.0%. Builder's claim reproduced exactly.

REFUTED — `mage ci` is green at HEAD with the deletion staged.

#### 9. State-flip atomicity

- `git diff workflow/drop_2/PLAN.md` shows exactly one diff hunk: line 48,
  `- **State:** todo` → `+ **State:** done`. No other lines in the Droplet 2.1
  block changed. Paths, Packages, Acceptance, Blocked-by are byte-identical to
  the planner-emitted spec.

REFUTED — atomic state flip; planner intent honored.

#### 10. No collateral damage — `git diff --stat`

```
 templates/builtin/default-frontend.json | 1469 ----------------------------
 templates/builtin/default-go.json       | 1585 -------------------------------
 templates/embed.go                      |   16 -
 workflow/drop_2/PLAN.md                 |    2 +-
 4 files changed, 1 insertion(+), 3071 deletions(-)
```

Plus an untracked `workflow/drop_2/BUILDER_WORKLOG.md` (new, expected).

Exactly the 3 deletions + 1-line PLAN.md state flip + the new BUILDER_WORKLOG.md.
No unexpected file change.

REFUTED.

#### 11. Coverage drop below 70%

Reviewer-run `mage ci` confirms every package ≥ 70.0%. The lowest is
`internal/tui` at exactly 70.0% (not regressed by this droplet — the deletion
removed only an unimported `embed.FS`-only package with no test surface to
reshape package-level coverage). Builder's "1263 tests pass, all coverage ≥ 70%"
claim is reproducible.

REFUTED — no coverage regression.

#### 12. Worklog completeness

- `workflow/drop_2/BUILDER_WORKLOG.md` documents:
  - Outcome.
  - Files touched (deletions, including auto-removed parent dirs).
  - Files touched (state-flips).
  - MD edits under carve-out (none, with rationale and PLAN.md line cite).
  - Mage targets run (`mage ci` — green; tests, packages, coverage minimum,
    build, exit code).
  - Design notes (pre-deletion verification, embed semantics, atomic deletion).
  - `## Hylla Feedback` section present (`N/A — task touched non-Go files only`).

Worklog is complete per the implicit Drop 2 droplet template (no formal schema
yet pre-Drop-2 cascade).

REFUTED.

#### 13. AC3 hit-set verification

Builder claimed `git grep "templates/builtin"` returns only MD hits, with
`CLAUDE.md` having zero Go-tree hits and `workflow/drop_2/PLAN_QA_FALSIFICATION.md`
appearing in the hit-set. Re-verified at HEAD:

- `git grep -n "templates/builtin"` returns only MD hits in:
  - `PLAN.md:1605, 1609, 1623` — top-level cascade plan, MD prose.
  - `README.md:298, 309` — MD prose (deferred per #7 above).
  - `workflow/drop_2/PLAN.md:5, 16, 42, 49, 54, 148, 384, 394, 415, 422` — drop
    plan MD.
  - `workflow/drop_2/PLAN_QA_FALSIFICATION.md:126` — drop QA audit-trail MD.
- `git grep -n "templates/builtin" CLAUDE.md` → empty (zero hits — confirmed).
- `git grep -n "templates/builtin" -- '*.go'` → empty (zero Go-tree hits).
- `git grep -n "templates/builtin" -- '*_test.go'` → empty.

Builder's AC3 reading correct: every surviving hit is MD prose, not a Go-tree
reference. AC3 is satisfied.

REFUTED.

#### 14. Forgotten test fixtures

- `git grep -n "templates/builtin" -- '*_test.go'` → empty.
- No `*_test.go` file loads `templates/builtin/*.json` via `os.ReadFile` or any
  string-path I/O.

REFUTED — no test-fixture loader is broken by the deletion.

### Counterexamples

None constructed. All 14 attack vectors REFUTED.

### Verdict Summary

**PASS.** Builder's claim that Droplet 2.1 (delete `templates/` package outright)
is clean and `mage ci` is green is fully verified. Zero Go importers, zero CI
references, zero test fixtures, zero reflection paths, zero magefile references,
zero `go.mod` orphans. State flip is atomic (one line). `git diff --stat` shows
only the expected 3 deletions + 1-line PLAN.md flip + new BUILDER_WORKLOG.md.
Reviewer-run `mage ci` reproduces builder's reported result (1263 tests / 19
packages / coverage ≥ 70.0% / build green / exit 0). README.md lines 298/309
are MD prose drift correctly deferred to Drop 3 per planner carve-out.

No blocking counterexamples. No nits worth blocking on. Droplet 2.1 is ready
for closeout.
