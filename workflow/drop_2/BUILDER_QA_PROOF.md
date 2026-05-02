# DROP_2 — Build QA Proof

(durable; append `## Droplet N.M — Round K` per QA attempt; NEVER `git rm`d)

## Droplet 2.1 — Round 1

**Verdict:** pass
**Date:** 2026-05-01

### Findings

#### F1. Staging matches expected delta exactly. (PASS)

`git status --porcelain` returned exactly the five expected entries and no extras:

- `D  templates/builtin/default-frontend.json`
- `D  templates/builtin/default-go.json`
- `D  templates/embed.go`
- ` M workflow/drop_2/PLAN.md` (state flips)
- `?? workflow/drop_2/BUILDER_WORKLOG.md` (new round artifact)

No stray modifications, no accidental edits to other files, no orphaned `templates/` subentries. Three deletions match AC1.

#### F2. Zero Go importers of the deleted package. (PASS)

`git grep -n "evanmschultz/tillsyn/templates" -- '*.go'` returned no hits. AC2 satisfied: no orphan import paths anywhere in the Go tree.

#### F3. Zero `templates.ReadFile` / `templates.Files` references in Go. (PASS)

`git grep -nE "templates\.(ReadFile|Files)" -- '*.go'` returned no hits. The package's API surface (the `Files embed.FS` var and `ReadFile` helper from the deleted `templates/embed.go`) has no consumers — the package was already runtime-dead before deletion.

#### F4. `templates/` parent dir gone. (PASS)

`ls templates/` returned `"templates/": No such file or directory (os error 2)`. Both `templates/` and `templates/builtin/` were auto-removed when their last child was `git rm`'d. No empty-dir residue.

#### F5. `mage ci` green at HEAD. (PASS)

I re-ran `mage ci` on HEAD (post-deletion, with the deletes staged) and observed:

- `[SUCCESS] Verified tracked sources`
- `[SUCCESS] Listed tracked Go files`
- `[SUCCESS] Checked Go formatting`
- `[SUCCESS] Test stream detected`
- 19 packages, 1263 tests, 0 failures, 0 skipped — every package `[PKG PASS]`
- `Minimum package coverage: 70.0%` met across all 15 reported packages (lowest is `internal/tui` at exactly 70.0%; `internal/buildinfo` at 100.0%)
- `[SUCCESS] Built till from ./cmd/till`

This independently corroborates the builder's claim. AC4 satisfied.

#### F6. Droplet 2.1 state in `PLAN.md` is `done`. (PASS)

`workflow/drop_2/PLAN.md:48` reads `- **State:** done` immediately under the `#### Droplet 2.1 — Delete \`templates/\` package outright` heading at line 46. State-flip executed correctly.

#### F7. `BUILDER_WORKLOG.md` exists with valid Round 1 content. (PASS)

`workflow/drop_2/BUILDER_WORKLOG.md` (new, untracked, 35 lines) contains:

- `## Droplet 2.1 — Round 1` heading
- `**Outcome:** success.`
- Files-touched section enumerating the three `git rm`'d files + the two auto-removed parent dirs + the PLAN.md state flips
- `**MD edits under carve-out:** none` with explicit reasoning citing PLAN.md line 394
- `**Mage targets run:**` section reporting `mage ci` green, 1263 tests, 19 packages, exit 0
- `**Design notes:**` section covering loader-coupling investigation
- `## Hylla Feedback` section explaining N/A status with rationale (deletion-only droplet, single Go file deleted outright, importer search via `git grep` was the right tool)

Structurally sound and substantively accurate.

#### F8. AC3 — `templates/builtin` references are MD-only outside Go tree. (PASS, with one cosmetic nit)

`git grep "templates/builtin"` returned 12 hits across exactly four MD files:

- `PLAN.md:1605, 1609, 1623` — top-level project PLAN (Drop 3 template overhaul context)
- `README.md:298, 309` — README dogfood-template links
- `workflow/drop_2/PLAN.md:5, 16, 42, 49, 54, 148, 384, 394, 415, 422` — droplet plan itself
- `workflow/drop_2/PLAN_QA_FALSIFICATION.md:126` — QA falsification round 1

**Zero `.go` hits.** AC3's hard constraint ("only MD references...not Go-tree references") is satisfied.

**Nit (T6 raised by builder, confirmed):** AC3's literal expected-hit list at PLAN.md:54 names `README.md`, `PLAN.md`, `CLAUDE.md`, and `workflow/drop_2/PLAN.md`. Reality:

- `CLAUDE.md` is in the literal list but produces no hit. (Unsurprising — CLAUDE.md doesn't reference template builtin paths today.)
- `workflow/drop_2/PLAN_QA_FALSIFICATION.md:126` produces a hit but is not in the literal list.

The substantive intent of AC3 ("zero Go-tree hits, all surviving hits are doc/historical-audit MD prose") is fully satisfied. The literal enumeration is stale but non-blocking — Drop 3's full template rewrite will sweep these MD references anyway, per PLAN.md:394 ("the surviving MD references are not load-bearing for Drop 2"). Recommend leaving as-is for Round 1 PASS; STEWARD or Drop 3 planner can fold the literal list into a future PLAN.md update if useful.

#### F9. No Go orphans. (PASS, covered by F5)

`mage ci` includes `[SUCCESS] Built till from ./cmd/till`, which builds the entire dependency graph from the CLI entrypoint. Successful build means no orphan imports / dead references / unresolved symbols anywhere in the reachable Go tree. Per project rules raw `go build ./...` is forbidden; the mage target is the canonical equivalent and it passed.

#### F10. QA-file preservation. (PASS)

`ls workflow/drop_2/` shows all eight prior QA artifacts intact:

- `PLAN_QA_PROOF.md`, `PLAN_QA_PROOF_R2.md`, `PLAN_QA_PROOF_R3.md`
- `PLAN_QA_FALSIFICATION.md`, `PLAN_QA_FALSIFICATION_R2.md`, `PLAN_QA_FALSIFICATION_R3.md`, `PLAN_QA_FALSIFICATION_R4.md`
- (Plus the new `BUILDER_WORKLOG.md` and modified `PLAN.md`.)

No `git rm` of QA files. Memory rule `feedback_never_remove_workflow_files.md` honored.

### Missing Evidence

None. All 10 required proof checks have direct citations. The builder's claims are reproducible end-to-end on HEAD with the staged deletes.

### Verdict Summary

Droplet 2.1's deletion landed cleanly. Three Go-relevant files (`templates/builtin/default-frontend.json`, `templates/builtin/default-go.json`, `templates/embed.go`) were `git rm`'d; both parent directories auto-removed; the `templates/` package no longer exists. Zero Go importers, zero residual API uses, zero `.go`-tree references to `templates/builtin`. `mage ci` reproduces green on HEAD with 1263 tests passing across 19 packages, all coverage ≥ 70.0%, and the `till` binary builds clean — independently confirming the builder's claim. PLAN.md's Droplet 2.1 state is correctly `done`; `BUILDER_WORKLOG.md` carries a structurally-sound Round 1 entry with a valid `## Hylla Feedback` section. The one cosmetic nit (T6: AC3's literal expected-hit list mentions `CLAUDE.md` which never appeared as a hit and omits `workflow/drop_2/PLAN_QA_FALSIFICATION.md` which does) does not affect the substantive intent of AC3 and is correctly deferred to Drop 3's full template rewrite. Ready to commit.

### Hylla Feedback

N/A — task touched non-Go files only (deletions of two JSON files and one trivial Go file with zero importers). Verification used `git grep`, `git status`, `ls`, `Read`, and `mage ci` directly — Hylla would not have added value over `git grep` for the "are there importers?" question. No Hylla queries were attempted, no fallbacks needed.
