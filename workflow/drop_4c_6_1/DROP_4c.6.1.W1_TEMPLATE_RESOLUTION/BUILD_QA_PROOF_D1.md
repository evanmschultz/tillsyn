# W1.D1 — BUILD-QA-PROOF Verdict

**Date:** 2026-05-12
**Reviewer:** go-qa-proof-agent (opus)
**Overall verdict:** PASS WITH NITS

## Acceptance Bullet Coverage

### AC #1 — 4-tier resolution order in `loadProjectTemplate`

> `loadProjectTemplate` has a 4-tier resolution: bare-root → primary-worktree → HOME (`~/.tillsyn/templates/<group>.toml`) → embedded default.

**Evidence:**
- `internal/app/service.go:577-586` — `loadProjectTemplateWithHome` builds the candidate slice in priority order:
  - bareRoot at line 578-580.
  - primaryWorktree at line 581-583.
  - HOME `filepath.Join(homeDir, ".tillsyn", "templates", group+".toml")` at line 584-586.
- `internal/app/service.go:599` — embedded fallback via `templates.LoadDefaultTemplateForLanguage(project.Language)` after the loop.
- `internal/app/service.go:452-473` — doc-comment enumerates the 4 tiers in the same order.
- Sub-test 1 "HOME file exists is used before embedded fallback" (`service_test.go:6612-6631`) asserts HOME wins over embedded.

**Verdict:** PASS

---

### AC #2 — Group derivation and skip-on-empty-Language

> `group` for HOME tier = `strings.TrimSpace(project.Language)` when non-empty; empty Language skips the HOME tier candidate (no path to read).

**Evidence:**
- `internal/app/service.go:543` — `loadProjectTemplate` calls `loadProjectTemplateWithHome(project, homeDir, strings.TrimSpace(project.Language))`.
- `internal/app/service.go:584` — `loadProjectTemplateWithHome` guards `if homeDir != "" && group != ""` before appending the HOME candidate. Empty group is therefore skipped.

**Verdict:** PASS (skip-on-empty-group enforced by code; no direct test for empty-string group — see NIT #1).

---

### AC #3 — First-candidate-wins

> First-candidate-wins: if HOME file exists + parses OK, embedded is not consulted.

**Evidence:**
- `internal/app/service.go:587-595` — loop returns on first successful candidate via `return tpl, true, nil`.
- Sub-test 1 (`service_test.go:6612-6631`) seeds HOME with a distinct marker (`homeMarker = 5555`) and asserts `tpl.Tillsyn.MaxContextBundleChars == homeMarker`, proving the embedded default (zero marker) was NOT loaded.

**Verdict:** PASS

---

### AC #4 — Error propagates on malformed HOME candidate (no fallthrough)

> HOME file exists but `templates.Load` errors → error propagates (same contract as existing tier-1/tier-2 error propagation).

**Evidence:**
- `internal/app/service.go:589-591` — inside the walk loop, `templates.Template{}, false, err` returned immediately on error.
- Sub-test 3 "HOME file malformed error propagates" (`service_test.go:6655-6677`) seeds HOME with `schema_version = "v1"\nunknown_key = "boom"\n`, asserts `err != nil`, `errors.Is(err, templates.ErrUnknownTemplateKey)`, `ok == false`, and `tpl.SchemaVersion == ""`.

**Verdict:** PASS

---

### AC #5 — `os.UserHomeDir()` failure silently skips HOME

> `os.UserHomeDir()` failure → HOME tier silently skipped (consistent with `readUserTierAgent` pattern in render.go:899-900).

**Evidence:**
- `internal/app/service.go:539-542`:
  ```
  homeDir := ""
  if h, err := os.UserHomeDir(); err == nil && strings.TrimSpace(h) != "" {
      homeDir = h
  }
  ```
  `homeDir` stays empty when `UserHomeDir()` returns an error OR whitespace-only string, then the helper's `if homeDir != "" && group != ""` skip-guard at line 584 omits the candidate.

**Verdict:** PASS (skip path enforced by code; no direct test injects empty homeDir or whitespace-only homeDir — see NIT #2).

---

### AC #6 — `loadProjectTemplateWithHome` seam exists with correct signature

> `loadProjectTemplateWithHome(project *domain.Project, homeDir, group string)` is added as a package-private helper that accepts explicit homeDir and group for test injection. `loadProjectTemplate` calls it with `os.UserHomeDir()` result and `strings.TrimSpace(project.Language)`.

**Evidence:**
- `internal/app/service.go:567` — `func loadProjectTemplateWithHome(project *domain.Project, homeDir, group string) (templates.Template, bool, error)`. Package-private (lowercase initial), exact signature from PLAN.md KindPayload.
- `internal/app/service.go:535-544` — `loadProjectTemplate` delegates: resolves `os.UserHomeDir()` + applies `strings.TrimSpace`, then calls `loadProjectTemplateWithHome(project, homeDir, strings.TrimSpace(project.Language))`.
- All four D1 sub-tests call `loadProjectTemplateWithHome` directly with `t.TempDir()` fake homeDir (e.g. `service_test.go:6620`, `6640`, `6663`, `6686`), confirming test-injection seam works.

**Verdict:** PASS

---

### AC #7 — `mage test-pkg ./internal/app` passes with 4 new sub-tests

> `mage test-pkg ./internal/app` passes. New `TestLoadProjectTemplate_HomeTier` covers all four cases listed in KindPayload: (a) HOME file exists; (b) HOME file absent; (c) HOME file malformed; (d) empty-worktree-paths + no HOME file.

**Evidence:**
- `mage test-pkg ./internal/app` — 481/481 PASS (matches builder's claim of 476 pre-existing + 5 new event-stream entries = parent test + 4 children).
- `mage test-func ./internal/app TestLoadProjectTemplate_HomeTier` — 5/5 PASS (parent + 4 sub-tests).
- `mage ci` — 3164/3164 PASS tree-wide, coverage threshold met (`internal/app` at 71.6%, above 70%).
- Sub-tests present and named (`service_test.go:6604-6705`):
  - (a) "HOME file exists is used before embedded fallback" — 6616
  - (b) "HOME file absent falls through to embedded" — 6636
  - (c) "HOME file malformed error propagates" — 6659
  - (d) "empty worktree paths and no HOME file falls through to embedded" — 6682

**Verdict:** PASS

---

## NITs

### NIT #1 [Axis: acceptance-criteria-coverage] [severity: low]

AC #2's empty-group skip path (`group == ""`) is enforced by the code at `service.go:584` but has no direct unit test asserting `loadProjectTemplateWithHome(&project, fakeHome, "")` returns the embedded fallback rather than constructing a malformed HOME path. Coverage is by construction-inspection only.

**Fix hint:** Add a 5th sub-test case to `TestLoadProjectTemplate_HomeTier`: pass a non-empty `fakeHome` with the file seeded at `<fakeHome>/.tillsyn/templates/.toml`, call `loadProjectTemplateWithHome(&project, fakeHome, "")`, and assert that the dot-prefix file is NOT loaded (embedded fallback wins). This proves the empty-group guard is not bypassed and that a malformed `.toml` path is never constructed.

### NIT #2 [Axis: acceptance-criteria-coverage] [severity: low]

AC #5's "`os.UserHomeDir()` failure → HOME tier silently skipped" path is enforced by `service.go:539-542` but no direct test injects empty homeDir into `loadProjectTemplateWithHome`. The current sub-tests always pass a non-empty `t.TempDir()` as homeDir.

**Fix hint:** Add a sub-test case: call `loadProjectTemplateWithHome(&project, "", "go")`, assert (a) no error, (b) `ok == true`, (c) `tpl.SchemaVersion == templates.SchemaVersionV1` (embedded default loaded). This proves the empty-homeDir skip-guard at line 584 is exercised, not just inspected.

### NIT #3 [Axis: spec-conformance] [severity: low]

The unrelated edit to `TestLoadProjectTemplate_UnsupportedLanguagePropagatesError` (`service_test.go:6884-6888`) changes the fixture language from `"fe"` to `"rust"`. This is not listed in D1's KindPayload changes. The edit is correct in substance (W4.D2 shipped `till-fe.toml`, so `"fe"` is no longer an unsupported axis) but is scope-creep relative to D1's stated `changes` array. No code defect; PLAN.md scope discipline note only.

**Fix hint:** Either fold this rationale into a subsequent commit message with explicit "incidental: D1 collateral from W4.D2 shipping till-fe.toml" framing, or accept as in-scope drive-by since refusing the change would leave a known-broken test. Recommendation: accept as-is, since the alternative is a stranded broken test.

---

## Verdict rationale

All 7 Acceptance bullets are satisfied:

- AC #1, #3, #4, #6, #7 are directly proven by both code inspection AND a passing sub-test.
- AC #2 (skip-on-empty-Language) and AC #5 (UserHomeDir failure skip) are enforced by the implementation but rely on construction-inspection rather than direct unit tests. The guard logic is correct, but coverage of the negative paths is by-construction only — captured as NIT #1 and NIT #2.

The implementation matches the PLAN.md KindPayload exactly:
- `loadProjectTemplateWithHome` ships with the prescribed `(project *domain.Project, homeDir, group string)` signature.
- `loadProjectTemplate` thins to a wrapper resolving `os.UserHomeDir()` + delegating.
- Walk order in the helper is bare-root → primary-worktree → HOME → embedded.
- HOME candidate path is `filepath.Join(homeDir, ".tillsyn", "templates", group+".toml")`.
- `writeHomeTemplateFixture` helper is present with `t.Helper()` and correct permission bits (0o755 / 0o644).
- 4 sub-tests cover all four cases enumerated in KindPayload.

Pass-count claim is reproduced: `mage test-pkg ./internal/app` reports 481 (476 prior + 5 new event-stream entries for parent + 4 children), and `mage ci` is tree-wide green at 3164/3164 with coverage threshold met.

Overall verdict: **PASS WITH NITS**. Builder's claim is supported by on-disk evidence. NITs #1 and #2 are coverage gaps (negative-path unit tests for AC #2 and AC #5 are missing) — recommend a follow-up patch or a sibling QA-falsification finding before W1 wave-close. NIT #3 is scope-discipline-only with no code defect.
