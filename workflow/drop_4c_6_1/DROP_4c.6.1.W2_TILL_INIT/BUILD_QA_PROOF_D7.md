# W2.D7 — BUILD-QA-PROOF Verdict

**Date:** 2026-05-13
**Reviewer:** go-qa-proof-agent (opus)
**Overall verdict:** PASS

---

## Acceptance Bullet Coverage

### Bullet 1 — `createProjectDBRecord` calls `svc.CreateProjectWithMetadata(ctx, app.CreateProjectInput{...})` instead of `svc.CreateProject(ctx, name, "")`.

Quote: "`createProjectDBRecord` calls `svc.CreateProjectWithMetadata(ctx, app.CreateProjectInput{...})` instead of `svc.CreateProject(ctx, name, "")`."

Evidence: `cmd/till/init_cmd.go:693-701` — call site is `svc.CreateProjectWithMetadata(ctx, app.CreateProjectInput{ Name: payload.Name, RepoPrimaryWorktree: cwd, RepoBareRoot: detectBareRoot(ctx, cwd), Language: mapGroupsToLanguage(payload.Groups), Metadata: domain.ProjectMetadata{ Groups: payload.Groups, }, })`. Diff confirms removal of the prior `CreateProject(ctx, name, "")` shape. `svc.CreateProjectWithMetadata` matches signature at `internal/app/service.go:311`.

**Verdict: PASS**

### Bullet 2 — Fields populated in `CreateProjectInput` per spec.

Quote: "`Name = payload.Name` … `RepoPrimaryWorktree = cwd` … `RepoBareRoot` = result of `git rev-parse --git-common-dir`; if the command fails, `RepoBareRoot = ""` … `Language` mapped through closed enum … `Metadata.Groups = payload.Groups` — typed field from W1.D2 … NO `KindPayload` JSON stopgap."

Evidence:
- `Name = payload.Name` — `init_cmd.go:694`.
- `RepoPrimaryWorktree = cwd` — `init_cmd.go:695`, where `cwd` is `os.Getwd()` resolved at `init_cmd.go:688-691`.
- `RepoBareRoot = detectBareRoot(ctx, cwd)` — `init_cmd.go:696`. `detectBareRoot` at `init_cmd.go:586-605` implements: `exec.LookPath("git")` first-guard (line 587-590), `exec.CommandContext(ctx, "git", "rev-parse", "--git-common-dir")` (line 591), `filepath.Abs(filepath.Join(cwd, trimmed))` resolution (line 600), graceful empty `""` on every failure path (lines 589, 594, 598, 602).
- `Language = mapGroupsToLanguage(payload.Groups)` — `init_cmd.go:697`. `mapGroupsToLanguage` at `init_cmd.go:616-629`.
- `Metadata: domain.ProjectMetadata{ Groups: payload.Groups }` — `init_cmd.go:698-700`. Direct typed-field assignment. NO `KindPayload` in the literal (verified by reading the full `CreateProjectInput` literal at `init_cmd.go:693-701`; `KindPayload` does not appear).
- `domain.ProjectMetadata.Groups []string` typed field present at `internal/domain/project.go:168` with `json:"groups,omitempty"` tag.

**Verdict: PASS**

### Bullet 3 — Bare-root detection details (`exec.CommandContext`, `filepath.Abs`, non-zero → empty).

Quote: "`exec.CommandContext(ctx, "git", "rev-parse", "--git-common-dir")` run in `cwd`; output is trimmed and resolved to absolute path via `filepath.Abs`. If relative (e.g. `.git`), resolve relative to `cwd`. If the command exits non-zero, `RepoBareRoot = ""`."

Evidence: `init_cmd.go:586-605` implements every required behavior:
- `exec.CommandContext(ctx, "git", "rev-parse", "--git-common-dir").Output()` at line 591.
- `strings.TrimSpace(string(out))` at line 596.
- `filepath.Abs(filepath.Join(cwd, trimmed))` at line 600 — handles both relative (`.git`) and absolute (linked-worktree) outputs correctly because `filepath.Join` of `cwd` + absolute is normalized by `filepath.Abs`.
- Non-zero exit returns `""` at line 594.
- Empty output returns `""` at line 598.
- `filepath.Abs` failure returns `""` at line 602.

NIT(low): `exec.CommandContext` is NOT explicitly invoked with `cwd` as its `Dir` — relies on the process working directory matching `cwd`. Since the call chain is `runInitPipeline` → `createProjectDBRecord(ctx, opts, payload)` → `detectBareRoot(ctx, cwd)` and `cwd` is obtained from the same `os.Getwd()` immediately above the call, this is sound today (no `os.Chdir` between them). Setting `cmd.Dir = cwd` would be defense-in-depth. Tracked as NIT, not FAIL.

**Verdict: PASS**

### Bullet 4 — Idempotency unchanged: skip on duplicate name.

Quote: "Idempotency unchanged: if a project with the same name already exists, skip creation and return `"already exists — skipped"`."

Evidence: `init_cmd.go:678-686` — `svc.ListProjects(ctx, false)` followed by case-insensitive name comparison; on match returns `"already exists — skipped"`. The exact string literal matches the spec.

**Verdict: PASS**

### Bullet 5 — Laslig summary row for `"project DB"` unchanged format.

Quote: "Laslig summary row for `"project DB"` unchanged format."

Evidence: `init_cmd.go:569` writes `{"project DB", dbStatus}` to the `writeCLIKV` call. The label and shape are unchanged from D6.

**Verdict: PASS**

### Bullet 6 — CONSUMER-TIE three `run()` end-to-end tests (git repo / non-git / idempotent).

Quote: "CONSUMER-TIE: `run(ctx, args, &out, io.Discard)` end-to-end — (a) new project in a git repo (verifies `RepoPrimaryWorktree` non-empty), (b) new project NOT in a git repo (verifies graceful empty `RepoBareRoot`), (c) re-run idempotent."

Evidence:
- `TestCreateProjectDBRecord_GitRepoCase` at `cmd/till/init_cmd_test.go:2051-2119`. Drives `run(context.Background(), …)` (line 2067). Asserts `RepoPrimaryWorktree != ""` (line 2101), `filepath.IsAbs(RepoPrimaryWorktree)` (line 2104), `RepoBareRoot != ""` (line 2108), `Language == "go"` (line 2112), `Metadata.Groups == ["go"]` (line 2116). Skips if git is absent (line 2052-2055) — matches the RiskNote about CI bare PATH.
- `TestCreateProjectDBRecord_NonGitDirCase` at `init_cmd_test.go:2125-2180`. Drives `run(…)` at line 2132. Asserts `RepoBareRoot == ""` (line 2165), `RepoPrimaryWorktree != ""` (line 2169), `Language == "fe"` (line 2173), `Metadata.Groups == ["fe"]` (line 2177).
- `TestCreateProjectDBRecord_IdempotentRerun` at `init_cmd_test.go:2187-2211`. Drives `run(…)` twice (lines 2199, 2205). Asserts second run returns nil error and stdout contains `"already exists"` (line 2208).

All three drive `run()` end-to-end with `--app tillsyn-init` for hermetic isolation, `--json` payload mode for headless flow, and read DB back through `sqlite.Open` + `svc.ListProjects` to assert persisted fields. The three CONSUMER-TIE bullets are covered.

**Verdict: PASS**

### Bullet 7 — `mage test-pkg ./cmd/till` green; `mage ci` green.

Quote: "`mage test-pkg ./cmd/till` passes; `mage ci` green."

Evidence (re-verified during this review):
- `mage test-pkg ./cmd/till` → 336 tests passed, 0 failed, 0 skipped, 1 package passed (output captured in this review session at 12.08s).
- `mage ci` → all packages at or above 70% coverage, `cmd/till` at 76.8%, build succeeded.

**Verdict: PASS**

### Special Focus — `Metadata.Groups = payload.Groups` direct typed-field assignment.

Evidence: `init_cmd.go:698-700` — `Metadata: domain.ProjectMetadata{ Groups: payload.Groups }`. The composite literal uses the typed field name `Groups`, not `KindPayload`. `KindPayload` does NOT appear anywhere in the `CreateProjectInput` construction (verified via diff + full read of the literal). Tests at `init_cmd_test.go:2116`, `:2177` read back `found.Metadata.Groups` and assert per-element equality — this is the typed `[]string` field, not a JSON unmarshalling step. Definitive PASS on the constraint that there is NO `KindPayload` JSON stopgap.

**Verdict: PASS**

### Special Focus — `Language` selection-order mapping (gen→"", go→"go", fe→"fe").

Evidence: `init_cmd.go:616-629` `mapGroupsToLanguage`:
- `len(groups) == 0` → `""` (defensive).
- `groups[0] == "go"` → `"go"`.
- `groups[0] == "fe"` → `"fe"`.
- default (including `"gen"`) → `"".

Tests assert: `["go"]` → `"go"` (`init_cmd_test.go:2112`); `["fe"]` → `"fe"` (`:2173`). The "gen → empty" case is documented in the function's doc comment but not asserted by a dedicated test. NOT a FAIL — the spec's selection-order rule is covered by the `"go"` and `"fe"` positive assertions; the default branch is trivially provable by code inspection of the closed switch.

NIT(low): No test exercises `["gen"]` → `Language == ""`. The behavior is provable by inspection (default-case in a closed switch). Adding `TestMapGroupsToLanguage_GenIsEmpty` as a unit-level micro-test would harden coverage of the default branch.

**Verdict: PASS WITH NIT**

### Special Focus — `RepoBareRoot` graceful empty on git absent / non-git dir.

Evidence:
- git-absent path: `init_cmd.go:587-590` — `exec.LookPath("git")` first-guard returns `""` if git is missing.
- non-git dir path: `init_cmd.go:591-594` — `exec.CommandContext(...).Output()` returns an error on non-git dir (`fatal: not a git repository`), which is caught and returns `""`.
- Verified end-to-end by `TestCreateProjectDBRecord_NonGitDirCase` (`init_cmd_test.go:2125-2180`) — `tmp` is created via `t.TempDir()` and is NOT `git init`-ed; the test asserts `RepoBareRoot == ""` (line 2165).

The git-absent case is NOT covered by a runtime test (would require `PATH=""` manipulation), but the early `LookPath` guard is provable by inspection and the non-git-dir test exercises the same downstream "empty result" semantics. The git-absent branch is a defensive no-op covered by the RiskNote.

**Verdict: PASS**

### Special Focus — D6 scope-extension fix: `writeTemplateTOML` marker comment + multi-group skip.

Evidence:
- `templateGroupMarkerPrefix = "# till-init-groups: "` at `init_cmd.go:724`.
- `writeTemplateTOML` at `init_cmd.go:760-834`:
  - Existing-file partial-state check uses BOTH the marker comment (`templateGroupMarkerPresent`) AND legacy `[<group>]` / `[<group>.]` section detection (lines 772-773). Hand-authored files using the old TOML section header form continue to suppress WARN — backward compatible.
  - Fresh file with single group: writes `# till-init-groups: <group>\n` followed by trimmed template content (lines 812-823). NO `[<group>]` TOML section header — the body is written at the top level so `templates.Load` sees `schema_version = "v1"` at the top level (NOT nested under `[go]`).
  - Multi-group (`len(groups) != 1`): early-return `(0, 0, nil)` at line 808-810. The Laslig row for `template.toml` in this case shows `"skipped (already exists)"` because `templateAdded == 0` (no `added > 0` branch fires; the falsey `templateAdded > 0` keeps the default skipped string). NIT(low): the user-facing status string is misleading for the multi-group case ("skipped (already exists)" when the file was deliberately not written). Acceptable per the worklog rationale that the proper merge is deferred to PLATFORM-TEMPLATES-R1.
- `templateGroupMarkerPresent` at `init_cmd.go:840-855`: scanner-based line parser handling `# till-init-groups: go,fe` and `# till-init-groups: go, fe` variants via `strings.TrimSpace(entry)`.

Tests assert the fix:
- `TestWriteTemplateTOML_HOMETierPresent` at `init_cmd_test.go:1915-1968`: now seeds HOME-tier with `# home-tier-sentinel\n` + valid embedded `till-go.toml` content; asserts both `# till-init-groups: go` marker (line 1956) AND `# home-tier-sentinel` are present (line 1960). NO assertion on `[go]` section header — the old form was deliberately removed.
- `TestWriteTemplateTOML_HOMETierAbsent` at `init_cmd_test.go:1973-2000`: asserts `# till-init-groups: go` marker is written from the embedded fallback (line 1993). NO `[go]` section header expected.
- `TestWriteTemplateTOML_PartialStateWarning` at `init_cmd_test.go:2234-2281`: seeds template.toml with `# till-init-groups: gen\n` + valid embedded `till-gen.toml` content; runs with `groups=["go"]` and asserts the WARN line for the missing `go` group fires AND the file is left unchanged.

The fix resolves the parse-error correctly: `schema_version = "v1"` is no longer nested under `[<group>]`, so `templates.Load` (called by `bakeProjectKindCatalog` when `RepoPrimaryWorktree` is set) succeeds. The tests reflect the new behavior — they assert the marker-comment form and explicitly drop the prior `[<group>]` section-header assertions. Multi-group skip path is provable by inspection; not directly exercised by an end-to-end run() test today, but the rationale is documented in both the code doc-comment and the worklog (refinement PLATFORM-TEMPLATES-R1 tracks future proper merge).

NIT(low): No end-to-end CONSUMER-TIE test exercises a multi-group `--json` payload (e.g. `["go","fe"]`) to confirm the skip-without-error behavior end-to-end. Provable by inspection; adding the test would harden coverage.

**Verdict: PASS WITH NIT**

### Special Focus — 3 CONSUMER-TIE end-to-end via `run()`.

Already covered under Bullet 6 above. All three drive `run(ctx, args, &out, io.Discard)`. Each asserts a distinct acceptance criterion: (a) git-repo path populates `RepoPrimaryWorktree`+`RepoBareRoot`+`Language`+`Metadata.Groups`; (b) non-git path produces graceful empty `RepoBareRoot`; (c) re-run is idempotent with `"already exists"` reported in stdout.

**Verdict: PASS**

---

## NITs

- **NIT.1 [Axis: spec-conformance] [severity: low]** `detectBareRoot` does not set `cmd.Dir = cwd` on the `exec.CommandContext` call → `init_cmd.go:591` → relies on process working directory matching `cwd`. Sound today (no `os.Chdir` between `runInitPipeline`'s `os.Getwd()` and `createProjectDBRecord`'s `os.Getwd()`), but setting `Dir` explicitly would be defense-in-depth and matches the spec's intent of "run in `cwd`".
- **NIT.2 [Axis: acceptance-criteria-coverage] [severity: low]** No dedicated test exercises `mapGroupsToLanguage(["gen"])` → `""` → `init_cmd_test.go` has no `TestMapGroupsToLanguage_*` table-driven test → behavior is provable by closed-switch inspection (default branch) but uncovered by an asserting test.
- **NIT.3 [Axis: spec-conformance] [severity: low]** Multi-group `writeTemplateTOML` skip surfaces as `"skipped (already exists)"` in the Laslig row → `init_cmd.go:554-557` (skip vs added is decided purely by `templateAdded > 0`) → the user-facing message is misleading when the file was not written because of multi-group, not because it already existed. A future polish could distinguish multi-group skip from idempotent-rerun skip.
- **NIT.4 [Axis: acceptance-criteria-coverage] [severity: low]** No CONSUMER-TIE `run()` test exercises a multi-group payload (e.g. `groups=["go","fe"]`) end-to-end → would harden coverage of the multi-group `writeTemplateTOML` early-return path against future regressions. Provable by inspection today.

All four NITs are advisory; none block PASS.

---

## Verdict rationale

Every acceptance bullet in W2.D7's Specify block maps to direct file:line evidence:

1. The signature change to `CreateProjectWithMetadata` is concrete at `init_cmd.go:693-701`.
2. All four typed fields (`Name`, `RepoPrimaryWorktree`, `RepoBareRoot`, `Language`) plus `Metadata.Groups` are populated as the spec requires; `KindPayload` is not used.
3. `detectBareRoot` correctly implements the LookPath-first / CommandContext / filepath.Abs / graceful-empty contract.
4. `mapGroupsToLanguage` honors the selection-order rule with `groups[0]` driving the closed switch (NIT5 absorption from plan-QA — documented in the doc-comment).
5. The D6 scope-extension fix is concrete: section header replaced with `# till-init-groups: <group>` marker comment; multi-group writes are skipped to avoid duplicate-table errors in `templates.Load`; D6 tests are updated to assert the marker form.
6. Three CONSUMER-TIE `run()` tests cover the three acceptance scenarios (git-repo, non-git, idempotent re-run).
7. `mage test-pkg ./cmd/till` reports 336 PASS; `mage ci` reports green with `cmd/till` at 76.8% coverage.

Four low-severity NITs surface during deep review — none block PASS. The multi-group skip is correct behavior given the deferred refinement (PLATFORM-TEMPLATES-R1) but could be hardened with an explicit user-facing status string and a CONSUMER-TIE test. The `mapGroupsToLanguage` default branch is provable by inspection but uncovered by an asserting test. The `detectBareRoot` `cmd.Dir` omission is defense-in-depth.

Overall verdict: **PASS**.
