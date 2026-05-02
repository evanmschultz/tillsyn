# DROP_2 — BUILDER_WORKLOG

## Droplet 2.1 — Round 1

**Outcome:** success.

**Files touched (deletions):**

- `templates/builtin/default-frontend.json` (deleted via `git rm`)
- `templates/builtin/default-go.json` (deleted via `git rm`)
- `templates/embed.go` (deleted via `git rm`)
- `templates/builtin/` (auto-removed when last child deleted)
- `templates/` (auto-removed when last child deleted)

**Files touched (state-flips):**

- `workflow/drop_2/PLAN.md` — Droplet 2.1 `**State:** todo` → `**State:** in_progress` at start; flipped to `**State:** done` at end of round.

**MD edits under carve-out:** none. The surviving non-Go references to `templates/builtin/...` (in `README.md`, top-level `PLAN.md`, `workflow/drop_2/PLAN.md`, `workflow/drop_2/PLAN_QA_FALSIFICATION.md`) are doc/historical-audit prose, not load-bearing Go references. Per PLAN.md line 394: "the surviving MD references are not load-bearing for Drop 2." Drop 3 cleanup will rewrite the README/PLAN.md prose alongside the new template system. No trivial-phrase-fix candidates encountered — surrounding sentences are coherent without edits.

**Mage targets run:**

- `mage ci` — green. 1263 tests passed across 19 packages. Minimum package coverage 70.0% met. Build of `./cmd/till` succeeded. Exit code 0.

**Design notes:**

- Pre-deletion verification: `templates/embed.go` is a 17-line file declaring `var Files embed.FS` (`//go:embed builtin/*.json`) and `func ReadFile(name string) ([]byte, error)`. No other Go files in the package.
- `git grep "evanmschultz/tillsyn/templates" -- '*.go'` returned empty before deletion — zero Go importers. Post-deletion confirmation also empty. The package's `embed.FS` was unreferenced runtime-dead code.
- `git rm` of the three files cleanly removed both parent directories (`templates/builtin/`, `templates/`) — no residual empty dirs.
- The deletion is atomic in one staged change; orchestrator commits on return.

## Hylla Feedback

N/A — task touched non-Go files only (deletions). The single Go file touched was `templates/embed.go`, deleted outright. Pre-deletion verification used `Read`, `Bash` (`git grep`, `ls`) on a known small file rather than Hylla — appropriate for a deletion-only droplet where the question is "are there importers?" answered fastest by `git grep`.
