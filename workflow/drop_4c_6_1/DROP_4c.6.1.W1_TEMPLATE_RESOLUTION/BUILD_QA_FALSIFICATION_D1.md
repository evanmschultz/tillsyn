# W1.D1 — BUILD-QA-FALSIFICATION Verdict

**Date:** 2026-05-12
**Reviewer:** go-qa-falsification-agent (opus)
**Overall verdict:** PASS WITH NITS

## Attack Hypotheses Tested

### H1 — First-candidate-wins violation
- **Hypothesis:** Implementation could read all 4 tiers and merge instead of short-circuiting on first hit.
- **Evidence:** `internal/app/service.go:587-595`. The loop returns on first `ok=true` or first `err`. The HOME candidate is appended to the slice AFTER bareRoot and primaryWorktree (lines 578-586), and the loop iterates in order.
- **Finding:** Structural short-circuit confirmed by code-read; sub-test "HOME file exists is used before embedded fallback" validates HOME wins over embedded (the only competing pair the new sub-tests cover directly).
- **Verdict:** REFUTED — no counterexample.

### H2 — Error propagation
- **Hypothesis:** Malformed HOME file silently falls through to embedded.
- **Evidence:** `loadProjectTemplateCandidate` (service.go:622-639) wraps `templates.Load` errors with the path. Caller at service.go:589-591 returns on error WITHOUT continuing the loop. Sub-test "HOME file malformed error propagates" verifies `errors.Is(err, templates.ErrUnknownTemplateKey)` AND `ok=false` AND zero-value tpl.
- **Verdict:** REFUTED — error propagates correctly; verified by `mage testFunc ./internal/app TestLoadProjectTemplate_HomeTier` 5/5 PASS.

### H3 — HOME-tier path correctness
- **Hypothesis:** Path construction could differ from spec `~/.tillsyn/templates/<group>.toml`.
- **Evidence:** service.go:585: `filepath.Join(homeDir, ".tillsyn", "templates", group+".toml")` — exact match. Test fixture helper at service_test.go:6584-6594 builds the same path layout.
- **Verdict:** REFUTED.

### H4 — Empty / whitespace-only `project.Language`
- **Hypothesis:** Whitespace-only Language could produce malformed path `~/.tillsyn/templates/   .toml` or skip incorrectly.
- **Evidence:** `loadProjectTemplate` (service.go:543) applies `strings.TrimSpace(project.Language)` BEFORE delegating to the seam. Empty post-trim → guard at line 584 (`group != ""`) → HOME skipped. Then embedded fallback at line 599 handles empty Language via `LoadDefaultTemplateForLanguage("")` → till-gen.toml (verified at internal/templates/embed.go:230-231).
- **Caveat:** The inner `loadProjectTemplateWithHome` does NOT trim `group` itself. A direct caller (like D2's future coordinator) passing whitespace-only `"   "` would NOT be guarded. The current public callers all trim before passing. See NIT-3 below.
- **Verdict:** REFUTED for D1's public surface; NIT for seam hardening.

### H5 — `os.UserHomeDir()` failure handling
- **Hypothesis:** Failure could error out instead of skipping silently.
- **Evidence:** service.go:539-542: `homeDir` stays `""` if `os.UserHomeDir()` errors OR returns whitespace-only. Then guard at line 584 (`homeDir != ""`) skips. Matches `readUserTierAgent` pattern (render.go:898-900). Doc-comment at service.go:559-562 explicitly documents the skip behavior.
- **Verdict:** REFUTED.

### H6 — Nil-project guard duplication
- **Hypothesis:** A path could skip the outer guard but the inner one would fire (or vice-versa, panic).
- **Evidence:** Both `loadProjectTemplate` (line 536-538) and `loadProjectTemplateWithHome` (line 568-570) return `(zero, false, nil)` for nil project. Defensive, idempotent, safe regardless of caller.
- **Verdict:** REFUTED.

### H7 — Test coverage of "primary-worktree wins over HOME"
- **Hypothesis:** No test asserts that a project with BOTH a primary-worktree `template.toml` AND a HOME `~/.tillsyn/templates/<group>.toml` returns the primary-worktree content.
- **Evidence:** Searched service_test.go for joint primary+home assertions — none found. The 4 new sub-tests only cover HOME-vs-embedded competition; existing tests `TestLoadProjectTemplate_BareRootWins` + `TestLoadProjectTemplate_PrimaryWorktreeFallback` predate HOME and don't add a HOME fixture.
- **Finding:** Structural property (candidate-list-order) is intact, but no behavioral test enforces it.
- **Verdict:** NIT (coverage gap). See NIT-1.

### H8 — HOME-tier walked when project has Language but no HOME file
- **Hypothesis:** Could fail to fall through to embedded when HOME file is absent.
- **Evidence:** Sub-test "HOME file absent falls through to embedded" passes — embedded SchemaVersion v1 returned, HOME marker absent.
- **Verdict:** REFUTED.

### H9 — D2 seam contract usability
- **Hypothesis:** D2 might not be able to call the seam cleanly per-group.
- **Evidence:** Seam signature `(project *domain.Project, homeDir, group string) (templates.Template, bool, error)` matches D2's planned `loadProjectTemplatesForGroups` signature per PLAN.md line 387-391. D2 will loop `project.Metadata.Groups`, pass each group with the same homeDir, merge via `mergeTemplates`. Seam is appropriate.
- **Caveat (Unknown, not D1's defect):** Per-group calls will re-read bare-root and primary-worktree candidates N times. If a project has a worktree-level `template.toml`, every per-group call short-circuits BEFORE reaching the HOME tier — the HOME-per-group intent is effectively defeated for projects with worktree templates. This is a D2 architectural decision, not a D1 seam bug. Routed to D2 review.
- **Verdict:** REFUTED for D1's seam contract; Unknown deferred to D2.

### H10 — YAGNI / scope creep
- **Hypothesis:** Builder added code beyond what acceptance required.
- **Evidence:** Build modified `TestLoadProjectTemplate_UnsupportedLanguagePropagatesError` to use `"rust"` instead of `"fe"` (service_test.go:6884-6904). This is OUTSIDE D1's declared KindPayload (which only listed adding the new test + new function). However, the change is justified: `internal/templates/builtin/till-fe.toml` now ships (W4.D2 shipped it; `LoadDefaultTemplateForLanguage("fe")` returns success at embed.go:234-242). Without the `fe → rust` rename, the test's premise ("fe is unsupported") is now false. The rename is documented in the test's doc-comment with the W4.D2 cross-reference.
- **Finding:** Necessary correctness fix; documented in-source. KindPayload-vs-final-code drift, but defensible. See NIT-4.
- **Verdict:** REFUTED for YAGNI (every change has purpose); NIT for KindPayload completeness.

### H11 — Hermeticity regression in pre-existing tests (NEW ATTACK)
- **Hypothesis:** Adding the HOME tier inside the public `loadProjectTemplate` wrapper introduces an environment dependency in pre-existing tests that call the wrapper directly. Pre-D1, the wrapper was hermetic (no $HOME read). Post-D1, the wrapper reads real `os.UserHomeDir()`. If the dev's $HOME contains a file at `~/.tillsyn/templates/<lang>.toml`, pre-existing tests can silently change behavior.
- **Evidence:** 8 tests call `loadProjectTemplate` directly (service_test.go lines 6484, 6512, 6724, 6751, 6790, 6831, 6868, 6893). Of those, 4 are vulnerable (the 4 that DON'T have a competing bare-root or primary-worktree fixture):
  - `TestLoadProjectTemplate_EmbeddedFallback` (line 6484): empty paths, asserts embedded loaded.
  - `TestLoadProjectTemplate_BothAbsentEmbedded` (line 6831): real dirs but no template files, asserts embedded loaded.
  - `TestLoadProjectTemplate_RelativePathSafety` (line 6868): empty paths + chdir trap, asserts embedded loaded.
  - `TestLoadProjectTemplate_UnsupportedLanguagePropagatesError` (line 6893): `Language="rust"`, asserts `ErrLanguageNotSupported`.
- **Concrete failing-test repro:**
  ```
  $ mkdir -p ~/.tillsyn/templates
  $ cp internal/templates/builtin/till-go.toml ~/.tillsyn/templates/rust.toml
  $ mage testFunc ./internal/app TestLoadProjectTemplate_UnsupportedLanguagePropagatesError
  → FAILS: err == nil (HOME tier consumed rust.toml; embedded fallback never reached;
    test expects errors.Is(..., ErrLanguageNotSupported))
  ```
  Similar repro for the other three with `~/.tillsyn/templates/go.toml`:
  - `TestLoadProjectTemplate_EmbeddedFallback` ("whitespace-only" sub-test): would load HOME tpl. SchemaVersion still v1 so assertion passes coincidentally, but the test's semantic intent ("returned the EMBEDDED default") silently breaks.
  - `TestLoadProjectTemplate_BothAbsentEmbedded` (line 6841-6843): asserts `tpl.Tillsyn.MaxContextBundleChars == 0`. If HOME `go.toml` has any non-zero value here, test FAILS.
  - `TestLoadProjectTemplate_RelativePathSafety` (line 6875-6876): asserts `tpl.Tillsyn.MaxContextBundleChars != cwdMarker`. If HOME `go.toml` happens to have `max_context_bundle_chars = 9999`, false-positive failure. SchemaVersion check (6878-6880) might silently pass while the test's hermeticity intent is broken.
- **Current machine state:** `/Users/evanschultz/.tillsyn/templates` does NOT exist today, so tests pass. The CI runner's `$HOME` is also clean. The regression is dormant.
- **Severity:** CONFIRMED counterexample but DORMANT. Becomes active the moment the dev (or a downstream user — the whole POINT of the HOME tier feature) creates a `~/.tillsyn/templates/<lang>.toml` file. Each of the 4 vulnerable tests has a concrete failing-input scenario.
- **Verdict:** CONFIRMED — see Counterexamples §1 below.

### H12 — Race / goroutine leak / interface misuse / hidden init state / file-gating bypass / `mage install` / raw go commands
- **Hypothesis:** Standard Go falsification axes.
- **Evidence:** Function is pure (no goroutines, no shared state); no interfaces involved; no `init()` changes; D1's declared `paths` = `service.go` + `service_test.go` exactly matches the diff; no `go test` / `go build` / `mage install` invocations in tests or fixtures.
- **Verdict:** EXHAUSTED, no counterexample found.

---

## Unmitigated Counterexamples

### 1. Hermeticity regression in `loadProjectTemplate` direct-call tests (CONFIRMED, dormant)

**Surface:** `internal/app/service.go:535-544` — `loadProjectTemplate` now calls `os.UserHomeDir()` and reads real $HOME.

**Affected tests** (all in `internal/app/service_test.go`):
- `TestLoadProjectTemplate_EmbeddedFallback` (line 6457).
- `TestLoadProjectTemplate_BothAbsentEmbedded` (line 6823).
- `TestLoadProjectTemplate_RelativePathSafety` (line 6859).
- `TestLoadProjectTemplate_UnsupportedLanguagePropagatesError` (line 6891).

**Repro (canonical):**
```
mkdir -p ~/.tillsyn/templates
cp internal/templates/builtin/till-go.toml ~/.tillsyn/templates/rust.toml
mage testFunc ./internal/app TestLoadProjectTemplate_UnsupportedLanguagePropagatesError
```
Test fails: `err == nil` because HOME tier returned `rust.toml` content before the unsupported-language embedded fallback fires.

**Status:** Dormant on current dev machine + CI (no `~/.tillsyn/templates/` exists). Becomes active when the HOME tier feature is exercised (which is the entire point of W1).

**Recommended fix (route to follow-up build, NOT this droplet's responsibility unless dev wants it inline):**
Migrate the 4 vulnerable tests to call `loadProjectTemplateWithHome(&project, "", strings.TrimSpace(project.Language))` (forcing the HOME tier to skip via empty homeDir) OR `loadProjectTemplateWithHome(&project, t.TempDir(), strings.TrimSpace(project.Language))` (forcing HOME tier to read an empty fake home). The seam exists; the migration is mechanical.

**Why this is a counterexample (not just a NIT):**
- It is a concrete behavioral regression (tests that previously had no $HOME dependency now do).
- It has a reproducible scenario with a single shell command.
- The spec D1 acceptance criterion #1 says "First-candidate-wins semantics preserved" — preserved in the candidate walk, but NOT in the test invariants of pre-existing tests.
- The hermeticity break was NOT flagged in the builder's report or self-QA.

**Why severity is "PASS WITH NITS" instead of "FAIL":**
- All tests pass on current dev + CI today.
- The fix is small (4 line-edits in service_test.go).
- The core D1 deliverable (HOME tier insertion + seam) is correct.
- The dev can address inline in a follow-up build droplet before W1 is closed, or route to a W1-end refinement.

---

## NITs

### NIT-1 — Missing "primary-worktree wins over HOME" coverage test (low)
- **Severity:** low (structural property is intact; test would harden against future regressions).
- **Recommendation:** add a sub-test to `TestLoadProjectTemplate_HomeTier`: set up both a primary-worktree fixture AND a HOME fixture with distinct markers; assert primary-worktree marker wins. Symmetric with `TestLoadProjectTemplate_BareRootWins`.

### NIT-2 — Missing "primary-worktree dir exists but no template file + HOME present → HOME wins" coverage (low)
- **Severity:** low.
- **Recommendation:** add a sub-test where `RepoPrimaryWorktree` is a real dir without `.tillsyn/template.toml`, `homeDir` has the file — assert HOME content returned. Validates the "ENOENT → continue" branch at the worktree tier when HOME is the eventual hit.

### NIT-3 — Inner seam does not defensively trim `group` (low)
- **Severity:** low; today's only callers (loadProjectTemplate wrapper, future D2 coordinator) are required by spec to trim before passing.
- **Recommendation:** add `group = strings.TrimSpace(group)` at the top of `loadProjectTemplateWithHome` to harden the seam against future direct callers. One line; idempotent. Mirrors the public wrapper's defensive behavior.

### NIT-4 — KindPayload-vs-final-code drift on `fe → rust` test rename (low)
- **Severity:** low; change is justified and documented in-source.
- **Recommendation:** future planners explicitly list cross-cutting test-maintenance edits in KindPayload `changes` when an adjacent change makes an existing test obsolete. (Methodology refinement, not a D1 fix.)

### NIT-5 — `homeDir` not trimmed before assignment (low)
- **Severity:** low; `os.UserHomeDir()` never returns padded paths in practice.
- **Recommendation:** `loadProjectTemplate` line 540-542 could assign `homeDir = strings.TrimSpace(h)` instead of un-trimmed `h`. Minor consistency with the documented spec ("home is whitespace-only → skip").

---

## Verdict rationale

D1's CORE deliverable lands cleanly:
- 4-tier walk in priority order: bareRoot → primaryWorktree → HOME → embedded. Verified by code-read at service.go:577-595.
- Package-private testability seam `loadProjectTemplateWithHome` shipped with correct signature for D2 consumption.
- `loadProjectTemplate` thin wrapper delegates with `os.UserHomeDir()` + `strings.TrimSpace(project.Language)`.
- `TestLoadProjectTemplate_HomeTier` 4 sub-tests + helper. 5/5 PASS on `mage testFunc`.
- 481/481 PASS on `mage testPkg ./internal/app`.
- Symbol blast radius small and reviewable.
- First-candidate-wins, error propagation, nil-guard, empty-Language guard, `os.UserHomeDir()` failure skip — all verified by code-read AND by passing tests.

The CONFIRMED counterexample (hermeticity regression in 4 pre-existing tests) is real but DORMANT — every test passes today on dev + CI because no `~/.tillsyn/templates/` directory exists. The fix is mechanical (route 4 tests through the seam with a fake home). The core feature ships correctly; the gap is in pre-existing-test hermeticity.

**Verdict:** PASS WITH NITS. Recommend either:
- (a) Inline fix in a follow-up D1.1 build droplet migrating the 4 vulnerable tests, OR
- (b) Route to a W1-end refinement (file the regression as a known issue and address before drop close).

Decision left to orchestrator.
