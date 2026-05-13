# W2.D7 — BUILD-QA-FALSIFICATION Verdict

**Date:** 2026-05-13
**Reviewer:** go-qa-falsification-agent (opus)
**Overall verdict:** PASS WITH NITS

## Attack Hypotheses Tested

### H1 — `Language` mapping correctness (REFUTED)

- **Attack:** `mapGroupsToLanguage(["gen","go"])` should map to `""` per selection-order policy; verify behavior.
- **Probe:** Read `cmd/till/init_cmd.go:616-629`. Function switches on `groups[0]`: "go" → "go", "fe" → "fe", default (including "gen") → "".
- **Verdict:** REFUTED. `["gen","go"]` → "" matches spec line 348: "selection-order wins; user's first group pick determines primary language." Edge case `len(groups) == 0` returns "" defensively without panic.

### H2 — `RepoBareRoot` non-fatal failure paths (REFUTED)

- **Attack:** Verify all failure paths (git absent, non-git dir, exec error, filepath.Abs error) return `""` (not error).
- **Probe:** Read `cmd/till/init_cmd.go:586-605`. Four exit-empty paths:
  1. `exec.LookPath("git")` error → return ""
  2. `exec.CommandContext(...).Output()` error → return ""
  3. Empty trimmed output → return ""
  4. `filepath.Abs` error → return ""
- **Verdict:** REFUTED. All four failure modes return `""`. Function signature is `string` (no error) so non-fatality is enforced by type.

### H3 — `Metadata.KindPayload` stopgap (REFUTED)

- **Attack:** Verify no `Metadata.KindPayload = {"groups":[...]}` stopgap remains.
- **Probe:** `rtk grep -n "KindPayload" cmd/till/init_cmd.go` → 0 matches. `cmd/till/init_cmd.go:698-700` uses typed `domain.ProjectMetadata{Groups: payload.Groups}`.
- **Verdict:** REFUTED. Typed field only; no stopgap.

### H4 — D6 scope-extension correctness (REFUTED)

- **Attack:** Verify (a) runtime resolver parses single-group template.toml cleanly post-fix; (b) D6 partial-state warning works on either marker OR legacy `[<group>]` header.
- **Probe:**
  - (a) `writeTemplateTOML` no longer prepends `[<group>]` header — writes `# till-init-groups: <group>` comment + raw template body. `templates.Load` validates `schema_version` at top level; the embedded `till-go.toml` ships `schema_version = "v1"` at top level. No nesting → parser succeeds.
  - (b) `cmd/till/init_cmd.go:770-777`:
    ```go
    inMarker := templateGroupMarkerPresent(content, group)
    inSection := strings.Contains(content, "["+group+"]") || strings.Contains(content, "["+group+".")
    if !inMarker && !inSection { missing = append(...) }
    ```
    Both forms suppress WARN. Backward-compat preserved.
- **Verdict:** REFUTED. Section-header removal fixes the templates.Load nesting bug; partial-state check accepts both formats.

### H5 — Multi-group template.toml skip (PARTIAL CONFIRM — NIT only)

- **Attack:** Verify (a) runtime resolution still works via HOME tier + embedded fallback; (b) the skip is logged in Laslig summary.
- **Probe (a):** Multi-group runtime resolution goes through `bakeProjectKindCatalogWithHome` → `loadProjectTemplatesForGroups` (`internal/app/service.go:490-519`) which iterates each group calling `loadProjectTemplateWithHome` per group. Each per-group call walks: `<RepoBareRoot>/.tillsyn/template.toml` → `<RepoPrimaryWorktree>/.tillsyn/template.toml` → `<homeDir>/.tillsyn/templates/<group>.toml` → embedded `builtin/till-<group>.toml`. With D7's multi-group skip, candidates 1+2 don't exist → falls through to HOME tier or embedded. Resolution works.
- **Probe (b):** `writeTemplateTOML` returns `(0, 0, nil)` for multi-group fresh dir. In `runInitPipeline` (line 554-557): `templateTOMLStatus = "skipped (already exists)"` because `templateAdded == 0`. **The Laslig row reports "skipped (already exists)" but the file was NEVER written and does NOT exist** — factually wrong.
- **Verdict:** PARTIAL CONFIRM. Runtime resolution works correctly (REFUTED for (a)), but the Laslig status string is **misleading** for the multi-group fresh-install case (NIT for (b) — see NIT-1 below).

### H6 — Idempotent re-run (REFUTED)

- **Attack:** Re-run with same project name returns "already exists — skipped" cleanly without error.
- **Probe:** `cmd/till/init_cmd.go:678-686` scans existing projects via `svc.ListProjects` before create; returns `"already exists — skipped", nil` on match. `TestCreateProjectDBRecord_IdempotentRerun` runs `run()` twice, asserts second run is nil-error and stdout contains "already exists". Test passes.
- **Verdict:** REFUTED.

### H7 — `git rev-parse --git-common-dir` output handling (CONFIRMED, NIT severity)

- **Attack:** Relative output (`.git`, `../..`) resolved via filepath.Abs; verify resolved path is absolute. Also probe absolute-output case (linked-worktree).
- **Probe:**
  - Relative case: `filepath.Abs(filepath.Join(cwd, ".git"))` → `<cwd>/.git`. Works correctly.
  - **Absolute case (linked worktree):** `git rev-parse --git-common-dir` inside a `git worktree add`-created linked worktree returns an absolute path to the main repo's `.git`. The doc-comment claims "If the output is already absolute (linked-worktree case), filepath.Abs is a no-op." **This is WRONG.** `filepath.Join("/cwd", "/abs/main.git")` produces `/cwd/abs/main.git` (lexical concatenation after cleaning the leading `/`). The function would store a garbage path in `RepoBareRoot`.
- **Counterexample repro (conceptual):**
  ```
  cwd = /tmp/worktree/linked
  git_output = /tmp/repo/.git  (absolute, linked-worktree case)
  filepath.Join("/tmp/worktree/linked", "/tmp/repo/.git")
    = "/tmp/worktree/linked/tmp/repo/.git"  (WRONG)
  ```
- **Downstream impact:** `loadProjectTemplateWithHome` walks `<RepoBareRoot>/.tillsyn/template.toml` first — a bogus bareRoot just fails to find the file and falls through to subsequent candidates. No crash, no error to the user.
- **Verdict:** CONFIRMED. Severity: NIT. Function does not crash but the doc-comment claim is wrong AND `RepoBareRoot` carries a wrong value in linked-worktree-init scenarios. See NIT-2 below.

### H8 — CWD detection failure (REFUTED)

- **Attack:** `os.Getwd()` failure path — what happens?
- **Probe:** `cmd/till/init_cmd.go:688-691`:
  ```go
  cwd, err := os.Getwd()
  if err != nil {
      return "", fmt.Errorf("resolve cwd for project record: %w", err)
  }
  ```
- **Verdict:** REFUTED. Error is wrapped with `%w` and bubbled up. Fatal but explicit — the caller (`runInitPipeline`) surfaces it via `"till init: create project DB record: %w"`.

### H9 — CONSUMER-TIE tests actually consume via `run()` (REFUTED)

- **Attack:** Verify all 3 tests go through `run()` not direct `runInitPipeline` calls.
- **Probe:**
  - `TestCreateProjectDBRecord_GitRepoCase` line 2067: `run(context.Background(), []string{...}, nil, io.Discard)`
  - `TestCreateProjectDBRecord_NonGitDirCase` line 2132: `run(context.Background(), []string{...}, nil, io.Discard)`
  - `TestCreateProjectDBRecord_IdempotentRerun` lines 2199, 2205: two `run(context.Background(), args, ...)` invocations.
- **Verdict:** REFUTED. All three end-to-end via `run()`.

### H10 — DB mock vs real (REFUTED)

- **Attack:** Tests said to "mock DB"; verify no real DB file is created against the dev's `~/.tillsyn`.
- **Probe:** Tests use `t.Setenv("HOME", tmp)` (redirects `~/.tillsyn` to `tmp`) + `--app tillsyn-init` flag. The `platform.DefaultPathsWithOptions(platform.Options{AppName: "tillsyn-init"})` produces a `paths.DBPath` rooted in `tmp/.tillsyn-init/`. Real SQLite, but isolated to `t.TempDir()`.
- **Verdict:** REFUTED. The tests use a real SQLite DB but rooted in t.TempDir — full isolation from dev's real `~/.tillsyn/tillsyn.db`. (NIT: builder's worklog described this as "mocks DB" which is slightly imprecise — they use a real SQLite in tmp, not a mock. Doesn't affect correctness.)

### H11 — Scope bleed (REFUTED with justification)

- **Attack:** D7 modified `writeTemplateTOML` which is D6 territory; verify scope extension is bounded.
- **Probe:** D7's paths include `cmd/till/init_cmd.go` (MODIFY), and `writeTemplateTOML` lives in that file. The functional scope-extension is real (D6's `[<group>]` section header was breaking `templates.Load` when `RepoPrimaryWorktree=cwd` was newly set by D7). The fix:
  1. Removes the section-header prefix in single-group case.
  2. Adds the marker-comment for partial-state detection.
  3. Skips writing in multi-group case (avoids "table plan already exists" merge error).
  4. Backward-compat for legacy `[<group>]` headers preserved in the partial-state check.
  No edits to functions outside `writeTemplateTOML` / `createProjectDBRecord` / their helpers.
- **Verdict:** REFUTED. Scope extension is bounded to a single file. Justification is concrete and load-bearing (the D6 output would have broken D7's call site).

### H12 — YAGNI (REFUTED)

- **Attack:** Did builder add anything beyond spec + scope extension?
- **Probe:** New surfaces:
  - `detectBareRoot` — required by spec line 350.
  - `mapGroupsToLanguage` — required by spec line 348.
  - `templateGroupMarkerPrefix` constant — required for D7 scope-extension fix (replaces nested section header).
  - `templateGroupMarkerPresent` — required for partial-state check on marker form.
  - Backward-compat `[<group>]` legacy parsing — required by attack hypothesis 4(b) (explicitly listed in spawn prompt).
- **Verdict:** REFUTED. No YAGNI additions.

### H13 — Hermeticity (REFUTED)

- **Attack:** D7 uses `os.Getwd()` + `exec.Command("git", ...)`. Are tests using `t.Chdir` or fake CWD?
- **Probe:** Each test calls `t.Chdir(tmp)` + `t.Setenv("HOME", tmp)`. No `t.Parallel()` in `cmd/till/init_cmd_test.go` (`rtk grep` confirmed 0 matches). Test ordering is sequential, no env race.
- **Verdict:** REFUTED. Tests are hermetic.

### H14 — Lazy git detection (stale git version) (REFUTED)

- **Attack:** `exec.LookPath("git")` only checks PATH presence, not version compatibility. If git has a stale version, what happens?
- **Probe:** `git rev-parse --git-common-dir` exists in git since version 2.5 (2015). All practical CI runners have newer git. If a hypothetical pre-2.5 git ran the command, `exec.CommandContext(...).Output()` would return non-nil error → `detectBareRoot` returns "". Non-fatal.
- **Verdict:** REFUTED. Stale git → graceful empty result.

## Unmitigated Counterexamples

None — both CONFIRMED findings are NIT severity (non-fatal, downstream gracefully tolerates).

## NITs

### NIT-1 — Misleading Laslig "template.toml: skipped (already exists)" for multi-group fresh install

- **Location:** `cmd/till/init_cmd.go:554-557` (set by `runInitPipeline`) interacting with the multi-group skip path in `writeTemplateTOML:808-810`.
- **Description:** For a multi-group project on first init (no existing `template.toml`), `writeTemplateTOML` returns `(0, 0, nil)` (skips write). The Laslig summary then reports `template.toml: skipped (already exists)` — factually wrong, since the file does not exist and was not written. The doc-comment at line 798-799 says "return added=0, skipped=0 to the Laslig summary — no entry for this case" which contradicts actual behavior (the Laslig row IS emitted, just with misleading text).
- **Suggested fix:** Either (a) suppress the Laslig row for the `(0, 0, nil)` case, or (b) emit a third status string like `"skipped (multi-group — uses per-group HOME/embedded resolution)"`. Option (b) is more discoverable.
- **Severity:** NIT. User sees confusing output but `till init` still completes successfully and `bakeProjectKindCatalog` resolves correctly via per-group HOME tier + embedded fallback.

### NIT-2 — `detectBareRoot` linked-worktree absolute-output handling

- **Location:** `cmd/till/init_cmd.go:600` (`filepath.Abs(filepath.Join(cwd, trimmed))`).
- **Description:** The doc-comment at line 583-585 claims: "If the output is already absolute (linked-worktree case), filepath.Abs is a no-op." This is wrong. `filepath.Join("/cwd", "/abs/path")` produces `/cwd/abs/path` after cleaning the second arg's leading `/` — concatenation, not preservation. In a linked worktree (`git worktree add`-created path) where `git rev-parse --git-common-dir` typically returns an absolute path to the main repo's `.git`, `detectBareRoot` would store a bogus concatenated path in `RepoBareRoot`.
- **Downstream impact:** `loadProjectTemplateWithHome` walks the bareRoot candidate first; a bogus bareRoot just fails to find the file and falls through to next candidates. No crash, no error surfaced.
- **Suggested fix:** Check `filepath.IsAbs(trimmed)` before joining:
  ```go
  var combined string
  if filepath.IsAbs(trimmed) {
      combined = trimmed
  } else {
      combined = filepath.Join(cwd, trimmed)
  }
  abs, absErr := filepath.Abs(combined)
  ```
- **Severity:** NIT. Bug is real but downstream gracefully degrades. Tests don't exercise the linked-worktree case.

### NIT-3 — Builder worklog imprecision ("mock DB")

- **Location:** `workflow/drop_4c_6_1/DROP_4c.6.1.W2_TILL_INIT/BUILDER_WORKLOG.md` D7 entry, and PLAN.md line 372 RiskNotes.
- **Description:** Spec line 372 says "Tests for `createProjectDBRecord` must mock the DB (following existing test patterns in `init_cmd_test.go`) — no real DB in unit tests." The new tests do not mock the DB — they use a real SQLite DB rooted in `t.TempDir()`. This is functionally correct (isolation via tmp dir + HOME-env override) and matches the pattern used elsewhere in `init_cmd_test.go`, but the spec language "mock the DB" is misleading. Worklog reuses the imprecise language.
- **Severity:** NIT. Doc-language drift only. No functional impact.

## Verdict rationale

The build achieves all four W2.D7 acceptance criteria:

1. **RepoPrimaryWorktree non-empty absolute path** — set from `os.Getwd()`. `TestCreateProjectDBRecord_GitRepoCase` asserts `filepath.IsAbs()` and non-empty. PASS.
2. **RepoBareRoot empty for non-git dirs** — `TestCreateProjectDBRecord_NonGitDirCase` confirms empty. PASS.
3. **Language mapping (selection-order policy)** — go→"go", fe→"fe", anything else→"". Spec NIT5 absorption honored. PASS.
4. **Metadata.Groups typed field** — `domain.ProjectMetadata{Groups: payload.Groups}` used directly. No KindPayload stopgap. PASS.

Acceptance gates: `mage test-func` x3 for the new tests + `mage test-pkg ./cmd/till` (336/336) all GREEN.

The D6 scope-extension is well-justified, well-bounded, and accompanied by an internal-consistency fix (legacy `[<group>]` partial-state parsing for backward compat). The change-set passes the asymmetric attack pass.

Two NIT-severity findings (NIT-1 misleading Laslig status, NIT-2 linked-worktree absolute-output handling). Neither blocks D7 close-out per "NITs are first-class fixes" — recommended for inline absorption in a follow-up commit. NIT-3 is documentation-only and can be deferred to a worklog cleanup.

**Overall verdict: PASS WITH NITS.** Recommend orchestrator dispatch a small fix-up commit (or inline absorption in the next droplet) to address NIT-1 and NIT-2 before W2 closes. The droplet itself satisfies its acceptance contract.
