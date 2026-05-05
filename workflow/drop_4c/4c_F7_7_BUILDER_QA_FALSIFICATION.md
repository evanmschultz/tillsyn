# Drop 4c F.7-CORE F.7.7 — Builder QA Falsification

**Droplet:** F.7-CORE F.7.7 — Auto-add `.tillsyn/spawns/` to `.gitignore`
**Builder:** opus 4.7
**QA mode:** Read-only adversarial counterexample search.
**Date:** 2026-05-04
**Verdict:** **PASS WITH FINDINGS** — no CONFIRMED counterexample against the F.7.7 droplet's stated contract; multiple genuine attack surfaces probed and either REFUTED or downgraded to documented design choices / orchestrator-routed refinements. Two findings worth orchestrator visibility (one minor — call-site cross-project hazard surfaced by the package-scope `sync.Once`; one already-known — `mage test-pkg` reporter glitch).

Sibling parallel droplets F.7.5b (`spawn.go` test additions) and F.7.12 (`bundle.go` adapter wiring) were NOT attributed — every counterexample probe is scoped to symbols introduced by F.7.7's diff (`EnsureSpawnsGitignored`, `gitignoreContainsSpawnsEntry`, `writeGitignoreAtomic`, `gitignoreSpawnsEntry`, `gitignoreSpawnsEntryAnchored`, `ensureSpawnsGitignoredOnce`, `ensureSpawnsGitignoredErr`, `ResetEnsureSpawnsGitignoredOnceForTest`, and the `BuildSpawnCommand` block at `spawn.go:392-406`).

## 1. Findings

- 1.1 **F1 — Cross-project `sync.Once` capture (orchestrator-visible refinement, NOT a counterexample against this droplet's claim).** `ensureSpawnsGitignoredOnce` is a package-scope `sync.Once`. The first `BuildSpawnCommand` call wins the once-shot regardless of which project's `RepoPrimaryWorktree` it carries; every later call (even with a different `project.RepoPrimaryWorktree`) inherits the captured `ensureSpawnsGitignoredErr` value and **never re-runs the helper for the second project**. This means a multi-project dispatcher process where project A is os_tmp and project B is project-mode would silently NOT gitignore project B's `.tillsyn/spawns/` because the once-shot already fired for A. The droplet's claim is "once per dispatch session" which today is single-project — so this is NOT a counterexample against the droplet's narrow contract. It IS a real hazard for the multi-project dispatcher landing post-Drop-4d, and the builder's own doc-comment at `spawn.go:179-186` flags exactly this: *"When the dispatcher grows a struct receiver, this lifts onto the receiver to scope the once-shot to one Dispatcher instance — multi-tenant test runners can then exercise multiple Dispatchers without cross-tenant gating interference. Drop 4c F.7.7 ships the package-scope form to keep the diff minimal; the Drop 4d / Drop 5 dispatcher-as-struct refactor moves it."* The risk is acknowledged in code; flag for the Drop 4c refinements list so the multi-project lift doesn't get forgotten.

- 1.2 **F2 — `mage test-pkg internal/app/dispatcher` reports `[PKG FAIL]` with 0 tests / 0 failures / 0 build errors and exit 1, while `mage check` runs all 2642 dispatcher tests green.** Reproduced live during this review. This is a mage-target regression (the test-pkg reporter glitch), not a F.7.7 build-quality issue — the underlying tests pass cleanly under `mage check` / `mage ci`. It does, however, complicate any QA agent that uses `mage test-pkg` as the per-package gate. Surface to the dev / orchestrator: file under Drop 4c refinements (mage tooling).

- 1.3 **F3 — Build verification clean.** `mage ci` exits green with all 24 packages passing, dispatcher coverage at 75.1% (≥ 70% threshold), and `cmd/till` building. No lint, fmt, or vet failure attributable to the F.7.7 diff.

- 1.4 **F4 — Code-quality nits, not counterexamples.**
  - 1.4.1 `EnsureSpawnsGitignored` does NOT validate that `projectRoot` is a directory (only that the trimmed string is non-empty). A path pointing at a file would surface as a less-clear `read .gitignore: ...` or `open ... .tmp: not a directory` error. The behavior is technically correct (any failure is propagated with a wrapped error) but the error message will be less specific than the input-validation guard's `ErrInvalidBundleInput`-with-reason form. Minor; not load-bearing.
  - 1.4.2 `gitignoreContainsSpawnsEntry` allocates an intermediate string (`string(bytes.TrimRight(raw, "\r"))`) per line. Hot path is bounded (called once per process via `sync.Once`) so the allocation cost is negligible — explicitly accepted under YAGNI.
  - 1.4.3 The Bash builder logs read "276/276 pass (was 269/269 pre-edit; +7 new tests)"; my count from the diff is 7 new direct-helper tests in `bundle_test.go` + 2 new integration tests in `spawn_test.go` = +9 tests, not +7. Worklog accounting drift, not a code-quality issue. Minor — do not act.

## 2. Counterexamples

No CONFIRMED counterexamples against the F.7.7 droplet's contract. Every attack below either REFUTED cleanly or surfaced an out-of-scope refinement (Findings 1.1 / 1.2 above) that the dev / orchestrator decides whether to act on.

### Attempted attacks

- 2.1 **Idempotency edges**

  - 2.1.1 **CRLF line endings (Windows `.gitignore`).** `gitignoreContainsSpawnsEntry` does `bytes.TrimRight(raw, "\r")` per split-line BEFORE the `strings.TrimSpace`, which neutralizes both bare `\r` and the `\r\n` split residue. **REFUTED** — exact CRLF idempotency is preserved.
  - 2.1.2 **Partial-prefix lines (`.tillsyn/spawns/foo`).** Match is line-exact after trim — `.tillsyn/spawns/foo` is a different string from `.tillsyn/spawns/`, so the helper appends. This is the documented + correct behavior because git interprets `.tillsyn/spawns/foo` as a more-specific rule that does NOT cover the directory wildcard. **REFUTED** — the helper would correctly append `.tillsyn/spawns/` so the wildcard fires.
  - 2.1.3 **Multiple variants on consecutive lines.** Both `.tillsyn/spawns/` and `/.tillsyn/spawns/` present. First match wins via early-return; no double-append. **REFUTED** — `TestEnsureSpawnsGitignoredRecognizesAnchoredVariant` covers the single-anchored case; both-present is implied by the same code path.
  - 2.1.4 **Whitespace-only lines / blank-line interleaving.** `strings.TrimSpace(...)` of an empty/blank line is the empty string, which equals neither sentinel. No false positive. **REFUTED**.
  - 2.1.5 **Trailing whitespace on the entry line (`.tillsyn/spawns/   `).** `TrimSpace` strips trailing space + tab + CR. Recognized as the canonical entry. **REFUTED**.
  - 2.1.6 **Negation pattern (`!.tillsyn/spawns/`).** Trim leaves `!.tillsyn/spawns/` which equals neither sentinel. The helper appends `.tillsyn/spawns/` AFTER the negation, which (per git ordering) means the unanchored ignore wins because later entries override earlier negations within the same `.gitignore`. **REFUTED for idempotency**; arguable design call — a dev who deliberately wrote `!.tillsyn/spawns/` to UN-ignore the directory would get auto-overridden. Edge case worth a doc-comment line but NOT a counterexample. (The droplet's claim is idempotency on the canonical form, not handling of un-ignore negations.)

- 2.2 **`sync.Once` captured-error semantics**

  - 2.2.1 **First call fails (e.g. permission-denied .gitignore), second call retries.** It does NOT retry — `ensureSpawnsGitignoredErr` is captured once and re-emitted forever. The builder's doc-comment at `spawn.go:189-194` documents this explicitly as a deliberate design choice: *"Callers that want a fresh attempt after a transient failure must restart the process — a deliberate design choice that lets us treat gitignore maintenance as boot-time setup rather than a hot-path retry loop."* Not a counterexample — explicit and intentional.
  - 2.2.2 **Cross-project once-shot capture.** See Finding 1.1 above. NOT a counterexample against this droplet; flagged as a real refinement for the multi-project dispatcher work.
  - 2.2.3 **Concurrent `BuildSpawnCommand` calls racing the once-shot.** `sync.Once.Do` is safe for concurrent use; the inner closure runs exactly once and all racing callers block until it completes. **REFUTED**.
  - 2.2.4 **Captured error contains stale `projectRoot` reference in its message.** The error message bubbles through `fmt.Errorf("dispatcher: ensure spawns gitignored: %w", ensureSpawnsGitignoredErr)`; if Project A failed and Project B's spawn now sees the wrapped Project-A error, the wrapping does not include B's project ID either. Diagnostic clarity issue, not correctness. **REFUTED for correctness**, noted under Finding 1.1 routing.

- 2.3 **Atomic temp-file leak on crash**

  - 2.3.1 **Crash between `OpenFile` and `Write`.** `f.Close()` is best-effort + `os.Remove(tmpPath)` cleans up — but if the process actually crashes (panic/signal), the temp file persists. On the next invocation, `os.OpenFile(tmpPath, O_RDWR|O_CREATE|O_TRUNC, 0o644)` truncates and reuses, so leak is bounded to one stale `.gitignore.tmp` file in the worktree until the next call. **REFUTED for atomicity** (the partial write never appears at the canonical path); note that crash-leftover `.gitignore.tmp` is NOT in `.gitignore` itself, so it would show up as untracked in `git status`. This is a UX paper cut, not a correctness bug. Minor; not promoted to a counterexample because identical semantics exist in `writeManifestAtomic` (the manifest writer's `.tmp` siblings have the same property and were not flagged in earlier QA rounds).
  - 2.3.2 **Crash between `f.Sync()` and `os.Rename(tmpPath, gitignorePath)`.** Atomicity preserved — the canonical `.gitignore` is unchanged from its pre-call state. The temp file persists; same UX paper cut as 2.3.1. **REFUTED**.
  - 2.3.3 **Disk full / `os.Rename` returns ENOSPC.** Error is wrapped + propagated; temp file `os.Remove`'d; canonical file untouched. **REFUTED**.
  - 2.3.4 **`f.Close()` returns an error.** The `f.Sync()` defer-equivalent pattern is correct: Close-error → Remove tmp → return wrapped error. The `_ = f.Close()` after Write/Sync error paths is conventionally Go-idiomatic (close after error doesn't usually report the same error and there's nothing to do with it). **REFUTED**.

- 2.4 **Race on `.gitignore.tmp`**

  - 2.4.1 **Two `EnsureSpawnsGitignored` callers race the same temp path.** The function itself does no internal locking, but the call site is gated by `sync.Once.Do` — concurrent BuildSpawnCommand calls all block until the first one completes the entire write-then-rename. **REFUTED at the call-site**.
  - 2.4.2 **External tool / sibling process holds `.gitignore.tmp` open.** `os.OpenFile(tmpPath, O_RDWR|O_CREATE|O_TRUNC, ...)` succeeds on POSIX even if another process has the file open (truncates underneath them). On Windows the `O_TRUNC` open could fail if the file is locked; behavior would be a wrapped open-error. The package targets POSIX (no Windows build tag), so **REFUTED in scope**. Build comment doesn't restrict OS so a Windows port would surface this; future-port concern, not today's counterexample.
  - 2.4.3 **Direct `EnsureSpawnsGitignored` call (not gated by `sync.Once`) racing the gated call.** Possible if a future caller bypasses the once-shot. Today only `BuildSpawnCommand` calls the function from production code (`grep` confirms — only call site in `internal/`). Tests call it directly but tests use `t.TempDir()` which gives each test a unique dir. **REFUTED for current call graph**; the doc-comment at `bundle.go:494-498` says *"concurrent calls on the same projectRoot race the rename but the write-temp-then-rename pattern guarantees no torn writes"* — exactly the right thing to say.

- 2.5 **`projectRoot` empty edge case**

  - 2.5.1 **Empty string in project mode.** `EnsureSpawnsGitignored("", "project")` returns wrapped `ErrInvalidBundleInput`. Covered by `TestEnsureSpawnsGitignoredRejectsEmptyProjectRootInProjectMode`. **REFUTED**.
  - 2.5.2 **Empty string in os_tmp mode.** Returns nil before the empty-check. Correct — os_tmp doesn't need a worktree. **REFUTED**.
  - 2.5.3 **Whitespace-only `projectRoot` in project mode.** `strings.TrimSpace(projectRoot) == ""` catches this. **REFUTED**.

- 2.6 **`projectRoot` is not a git worktree (no `.git` directory)**

  - 2.6.1 **`projectRoot` is just an arbitrary directory.** `EnsureSpawnsGitignored` does not check for `.git`. It will happily create or modify `<projectRoot>/.gitignore` for any directory. This is by design — `domain.Project.RepoPrimaryWorktree` is the contract surface; the dispatcher trusts the upstream that built the project value. The doc-comment doesn't promise git-aware behavior. **REFUTED for the droplet's contract**; out-of-band concern: a future caller passing a wrong directory could pollute someone's home dir or `/tmp` with a `.gitignore`. Probability is low (every production caller comes from `domain.Project`); explicit guard is a YAGNI premature-defense.
  - 2.6.2 **`projectRoot` is a sub-worktree of a parent git repo, where the dev's actual `.gitignore` lives one level up.** Helper writes `<projectRoot>/.gitignore` regardless. Git's `.gitignore` resolution is hierarchical, so the new file may be redundant or override parent ignores. By design the droplet treats `projectRoot` as the canonical worktree root. **REFUTED for contract**; minor doc-clarity ask: the helper assumes `projectRoot` IS the git worktree root, not an interior subdir. Worth a one-line doc-comment but not a counterexample.

- 2.7 **Atomicity / fsync ordering**

  - 2.7.1 **No parent-directory fsync.** Per `writeManifestAtomic`'s explicit comment at `bundle.go:317-321`, parent-directory fsync is intentionally omitted as best-effort forensic durability. `writeGitignoreAtomic` inherits the same posture without an explicit comment. **REFUTED for the droplet's intent** (matches the manifest writer's contract); a doc-comment cross-reference would help — minor.
  - 2.7.2 **`os.Rename` on a cross-device `.gitignore.tmp` → final.** Both files share the same parent dir (`gitignorePath + ".tmp"`), so the rename is intra-device. **REFUTED**.

- 2.8 **Doctrine / mage / orchestrator-discipline attacks**

  - 2.8.1 **Raw `go test` / `go vet` / `go build` invocation in the diff.** `grep` of the diff shows zero raw `go` toolchain invocations. **REFUTED**.
  - 2.8.2 **`mage install` invocation.** None. **REFUTED**.
  - 2.8.3 **File-scope bypass.** F.7.7 declared edits to `bundle.go` + `bundle_test.go` + `spawn.go` + `spawn_test.go` + the worklog. Diff stays exactly inside that scope. **REFUTED**.
  - 2.8.4 **Package-scope bypass.** All edits inside `internal/app/dispatcher`. **REFUTED**.
  - 2.8.5 **`go vet` violations / errcheck swallowing.** `_ = f.Close()` and `_ = os.Remove(tmpPath)` in error paths are conventional Go (no useful action on close-after-error / remove-best-effort). The Sync+Close+Rename ordering propagates errors via wrapped `%w`. No errors silently swallowed. **REFUTED**.
  - 2.8.6 **`init()` side effects / package-level mutable state.** `ensureSpawnsGitignoredOnce` + `ensureSpawnsGitignoredErr` ARE package-level mutable state. They are documented, gated by `sync.Once`, and have a test-only reset hook. The pattern is identical to existing `RegisterAdapter` / `RegisterBundleRenderFunc` in the same file. **REFUTED for new contract violation**; the cross-project concern is captured in Finding 1.1.

- 2.9 **Cascade-vocabulary / plan-QA attacks (N/A for build-qa-falsification but spot-checked).** F.7.7 is a `build` droplet (leaf), parent F.7-CORE is a `plan`. Sibling parallel droplets F.7.5b + F.7.12 share the same files (`spawn.go`, `bundle.go`) — this WOULD be a paths/packages-overlap attack on the planner if the three droplets had no `blocked_by` between them. The action-item description here explicitly notes "F.7.5b + F.7.12 are parallel siblings — do NOT attribute," which suggests the orchestrator manually serialized them at runtime. The planner-level attack would need the F.7-CORE plan QA to verify; out of scope for THIS droplet's QA. **REFUTED for THIS droplet's scope**; flag for plan-level QA visibility.

## 3. Summary

**Verdict: PASS.**

No CONFIRMED counterexample against F.7.7's contract: `EnsureSpawnsGitignored` correctly implements the four-row behavior matrix (os_tmp short-circuit / missing-file create / present-and-recognized no-op / present-and-missing append), idempotency holds across CRLF, anchored variants, partial-prefix lines, missing-trailing-newline edges, and concurrent re-entry via the `sync.Once` gate. Atomic write semantics match the existing `writeManifestAtomic` pattern (write-temp → fsync → close → rename). Input validation rejects empty `projectRoot` in project mode with a wrapped `ErrInvalidBundleInput`. The `mage ci` gate is green with 75.1% dispatcher coverage.

Two findings worth orchestrator visibility:

- **F1 (Finding 1.1)** — package-scope `sync.Once` will silently no-op gitignore maintenance on a second project's worktree once the dispatcher grows multi-project support. Already acknowledged in builder's own doc-comment as a Drop 4d / Drop 5 lift target. Add to Drop 4c refinements list so it doesn't get forgotten when the dispatcher-as-struct refactor lands.
- **F2 (Finding 1.2)** — `mage test-pkg internal/app/dispatcher` reports a misleading `[PKG FAIL]` exit-1 with `0 tests / 0 failures / 0 build errors`. Underlying tests pass cleanly under `mage check` / `mage ci`. Mage-tooling regression unrelated to F.7.7 — file under Drop 4c refinements (mage subsystem).

No code change required for F.7.7 to be considered complete.

## TL;DR

- T1 PASS — exhaustive idempotency + concurrency + atomicity attack pass turned up no CONFIRMED counterexample against F.7.7's contract; two orchestrator-visible findings (cross-project sync.Once hazard already self-flagged by the builder; `mage test-pkg` reporter glitch unrelated to this droplet) routed for refinements.
- T2 None CONFIRMED — every probed attack (CRLF / anchored variant / missing-newline / partial-prefix / sync.Once captured-error / temp-file crash leak / race / non-git projectRoot / empty projectRoot / cross-device rename / mage doctrine / file-scope) REFUTED via diff inspection + `bundle.go` / `spawn.go` reread + `mage ci` green.
- T3 PASS — `mage ci` green (2642 tests pass, 24 packages, dispatcher coverage 75.1% ≥ 70% threshold), no doctrine violation, no cascade-vocabulary issue inside the droplet's scope.

## Hylla Feedback

- Query: `hylla_search_keyword(query="EnsureSpawnsGitignored", artifact_ref="github.com/evanmschultz/tillsyn@main")`.
- Missed because: enrichment still running on the artifact ref (`enrichment still running for github.com/evanmschultz/tillsyn@main`). Hylla returned a clear error rather than stale data — correct behavior, not a Hylla quality miss.
- Worked via: `git diff HEAD -- internal/app/dispatcher/{bundle,bundle_test,spawn,spawn_test}.go` for the F.7.7 diff; `Read` for unchanged surrounding context; `rg -n` for symbol cross-references.
- Suggestion: Hylla's "enrichment still running" response is the right shape (fail-fast not stale-data); a one-line ETA hint (e.g. "approx N seconds remaining") would let agents decide between waiting and falling back. Today the right call (fall back to disk for an active diff) was obvious because `git status` already shows uncommitted local edits — Hylla can't help with uncommitted work anyway.
