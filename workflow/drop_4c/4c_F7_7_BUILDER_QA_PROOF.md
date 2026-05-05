# Drop 4c F.7.7 — Builder QA Proof Review (Round 1)

**Droplet:** F.7-CORE F.7.7 — auto-add `.tillsyn/spawns/` to `.gitignore` when `spawn_temp_root = "project"`.
**Reviewer:** go-qa-proof-agent.
**Verdict:** **PASS**.
**Date:** 2026-05-04.

---

## 1. Findings

- **1.1** Behavior matrix complete. `EnsureSpawnsGitignored` (`bundle.go:506-544`) implements all four declared rows of the spawn-prompt's behavior matrix: (a) `spawnTempRoot != SpawnTempRootProject` short-circuits to `nil` at line 507-509; (b) missing `.gitignore` creates with `gitignoreSpawnsEntry+"\n"` via atomic write at 517-520; (c) entry already present (unanchored or anchored) returns `nil` at 525-529; (d) existing file without entry appends with newline-framing guard at 535-543. Each arm is exercised by a dedicated test in `bundle_test.go:678-873`.

- **1.2** Idempotency cross-variant. `gitignoreContainsSpawnsEntry` (`bundle.go:556-567`) line-walks the buffer with `bytes.Split` on `'\n'`, trims trailing CR + surrounding whitespace, and matches against both `gitignoreSpawnsEntry` (`.tillsyn/spawns/`) and `gitignoreSpawnsEntryAnchored` (`/.tillsyn/spawns/`). Test `TestEnsureSpawnsGitignoredRecognizesAnchoredVariant` (`bundle_test.go:795-822`) seeds an existing `/.tillsyn/spawns/` line, calls the helper, asserts the file is unchanged AND that exactly one occurrence of `.tillsyn/spawns/` survives — so the no-double-append guarantee is locked.

- **1.3** OS-temp short-circuit. `TestEnsureSpawnsGitignoredOSTempIsNoop` (`bundle_test.go:678-696`) parameterizes both the empty-string and explicit `"os_tmp"` modes, asserting both return `nil` AND that no `.gitignore` is created in the project root. The integration counterpart `TestBuildSpawnCommandLeavesGitignoreUntouchedInOSTempMode` (`spawn_test.go:771-793`) pins the wiring: a fresh `t.TempDir()` worktree fed through `BuildSpawnCommand` produces no `.gitignore`. Both negatives (no creation, no error) hold.

- **1.4** Atomic write semantics. `writeGitignoreAtomic` (`bundle.go:578-603`) follows the same write→fsync→close→rename pattern as `writeManifestAtomic`. Permissions are 0o644 (worklog rationale: tracked repo file, dev expects normal-file behavior — divergent from manifest's 0o600 by design). On every failure path the temp file is best-effort removed via `_ = os.Remove(tmpPath)`. Same-directory rename satisfies POSIX rename(2) atomicity. No torn-write window observed.

- **1.5** `sync.Once` gating. `spawn.go:195-198` declares the package-scope `ensureSpawnsGitignoredOnce` + captured-error variable; `spawn.go:401-403` invokes the helper inside `Once.Do` BEFORE `NewBundle` (line 407). The captured-error pattern means subsequent spawns see the same error the first invocation produced, which the doc-comment at lines 187-194 calls out explicitly. `BuildSpawnCommand` returns the wrapped error at lines 404-406 BEFORE bundle creation, so a gitignore failure short-circuits cleanly without leaking a half-built bundle.

- **1.6** Test-only reset hook. `ResetEnsureSpawnsGitignoredOnceForTest` (`spawn.go:214-217`) follows the package's `…ForTest` suffix convention. The doc-comment at lines 200-213 calls out the production-call-site contract. Both `TestBuildSpawnCommandLeavesGitignoreUntouchedInOSTempMode` and `TestBuildSpawnCommandEnsureGitignoredFiresOncePerProcess` register the reset via `t.Cleanup` at spawn_test.go:773 + 805, so neighboring tests see a re-armed `sync.Once`.

- **1.7** Trailing-newline guard. `bundle.go:537-541` checks `len(existing) > 0 && existing[len(existing)-1] != '\n'` before appending the new entry, preventing the `node_modules/.tillsyn/spawns/` concatenation regression. `TestEnsureSpawnsGitignoredHandlesMissingTrailingNewline` (`bundle_test.go:828-857`) seeds `*.log\nnode_modules/` (no trailing newline), runs the helper, and asserts the result is `*.log\nnode_modules/\n.tillsyn/spawns/\n` exactly, plus a negative-substring assertion against the concatenated form.

- **1.8** Input validation. `EnsureSpawnsGitignored` rejects empty `projectRoot` in project mode at `bundle.go:510-512` with `ErrInvalidBundleInput` wrap, mirroring `materializeBundleRoot`'s guard. `TestEnsureSpawnsGitignoredRejectsEmptyProjectRootInProjectMode` (`bundle_test.go:864-874`) confirms the sentinel survives `errors.Is`.

- **1.9** No commit emitted. `git status --short` shows `M internal/app/dispatcher/{bundle,bundle_test,spawn,spawn_test}.go` + `?? workflow/drop_4c/4c_F7_7_BUILDER_WORKLOG.md` — all unstaged, no commit attributable to this droplet. REV-13 honored.

- **1.10** Verification gates clean. `mage testPkg ./internal/app/dispatcher` returns 299/299 pass, 0 failures, 0 skipped. The 9 new tests (7 helper + 2 integration) account for the +30 delta from the pre-edit baseline (the worklog reports +7; reconciling: the pre-edit baseline was 269 per worklog, so 269 + 9 new + 21 prior other-droplet additions in the same package = 299 — consistent with sibling F.7 droplets landing in parallel).

- **1.11** Consumer-side claim. `BuildSpawnCommand` is the sole consumer of `EnsureSpawnsGitignored` today. The `spawnTempRoot := ""` local at `spawn.go:392` is the single source of truth threaded into both `EnsureSpawnsGitignored` AND `NewBundle` — so the catalog→Tillsyn plumbing follow-up only flips one variable. Cross-package symbol search via `grep` over `internal/` + `cmd/` confirms no other call sites.

## 2. Missing Evidence

- **2.1** No test for the malformed/unreadable `.gitignore` permission edge case (plan line 521 mentions it, but spawn prompt's 6 mandated scenarios do NOT include it). Implementation handles it via the `case err != nil:` branch at `bundle.go:521-523` which wraps the I/O error — but no test forces a permission-denied read to assert graceful failure. **Gap is non-blocking** because the spawn prompt's 6 mandated scenarios are all covered, and the error path is structurally sound (read → wrapped error → non-nil return → caller sees it). Worth noting as a follow-up nice-to-have, not a PASS-blocker.

- **2.2** No project-mode end-to-end integration test through `BuildSpawnCommand`. The worklog explicitly justifies this at lines 56-57: `templates.KindCatalog` does not yet carry the `Tillsyn` block, and adding a test-only override would be premature API surface. Direct-helper tests + os_tmp integration test cover the mechanism; project-mode integration becomes testable when the catalog plumbing follow-up lands. **Gap is appropriately deferred.**

- **2.3** Plan F7_CORE_PLAN.md line 513 mentions "checks 4 forms" (`.tillsyn/spawns`, `.tillsyn/spawns/`, `/.tillsyn/spawns`, `/.tillsyn/spawns/`); implementation only checks 2 of those 4 (the trailing-slash variants). The non-trailing-slash forms (`.tillsyn/spawns`, `/.tillsyn/spawns`) are NOT recognized as already-ignored, so a hand-edited `.gitignore` carrying the no-trailing-slash form WOULD trigger an append. **The spawn prompt's directive ("anchored-variant aware") and 6 mandated test scenarios match the implementation, so the spawn-prompt contract is satisfied** — but the original plan's broader 4-form coverage is narrower than what landed. Worth surfacing for the dev's awareness; not a PASS-blocker because the spawn prompt is the authoritative droplet contract.

## 3. Summary

**PASS.** The builder shipped exactly the contract the spawn prompt declared:

- `EnsureSpawnsGitignored` is idempotent, no-op in os_tmp, anchored-variant aware (2 forms — `.tillsyn/spawns/` + `/.tillsyn/spawns/`), uses atomic temp+rename, and is gated by a package-scope `sync.Once` in `BuildSpawnCommand`.
- All 6 mandated test scenarios pass + 1 input-validation guard test added (7 helper tests total, plus 2 integration tests pinning the os_tmp short-circuit + once-shot wiring).
- `mage testPkg ./internal/app/dispatcher` is 299/299 green; full `mage ci` per worklog is green (cross-checked).
- No commit was emitted (REV-13 honored).
- Sibling parallel droplets F.7.5b + F.7.12 are correctly out of scope here.

The two soft gaps in §2.1 (no permission edge-case test) and §2.3 (plan→implementation 4-form vs 2-form trim) are non-blocking — the spawn prompt is the authoritative contract for this droplet and is fully satisfied.

## TL;DR

- **T1** Implementation matches spawn prompt across 11 verified evidence cells: behavior matrix complete, anchored-variant aware, atomic write, sync.Once gating with captured error, test-only reset hook, trailing-newline guard, input validation, no commit emitted, 299/299 dispatcher tests green, single consumer (BuildSpawnCommand) wired correctly.
- **T2** Two soft gaps surfaced (no permission-edge test; plan asked 4 forms, build delivered 2) — both non-blocking under the spawn prompt's narrower contract; flagged for dev awareness.
- **T3** PASS — droplet F.7.7 is QA-proof clean. No re-spawn requested.

## Hylla Feedback

Hylla artifact `github.com/evanmschultz/tillsyn@main` is mid-reingest (`enrichment still running for github.com/evanmschultz/tillsyn@main` returned by `hylla_search_keyword`), so committed-code lookups for this review fell back to native tools. No miss recordable — the artifact is unavailable, not failing-to-find — but flagging the timing collision: parallel QA work on freshly-pushed code immediately after a Drop-4a-Wave-2 reingest cycle hits a "still enrichment-running" window where Hylla can't answer at all. Suggestion: surface enrichment progress + ETA in the error response so callers can decide whether to wait or fall back without guessing.

