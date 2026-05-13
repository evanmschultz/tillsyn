# W2.D2 — BUILD-QA-FALSIFICATION Verdict

**Date:** 2026-05-13
**Reviewer:** go-qa-falsification-agent (opus)
**Overall verdict:** PASS WITH NITS

## Attack Hypotheses Tested

### H1 — No-op semantics on missing inputs

- **Hypothesis:** `detectFLATLayout` and `detectOldSchemaAgentsTOML` must no-op (return nil) when their target paths do not exist; only stat/open errors other than `ErrNotExist` should be propagated.
- **Test:** Read `cmd/till/init_cmd.go:418-477` for branch structure; cross-check tests `TestRunInitPipeline_FLATDetection/clean_state_no_flat_layout` (no `.tillsyn/agents/` dir present) and `TestRunInitPipeline_OldSchemaDetection/no_dot_agents_section_not_old_schema` (file present but no match — implicitly verifies an absent file would also no-op since the no-match path returns nil).
- **Finding:** `detectFLATLayout` line 431 returns nil on `errors.Is(err, fs.ErrNotExist)`. `detectOldSchemaAgentsTOML` line 471 returns nil on `fs.ErrNotExist`. The `clean_state_no_flat_layout` sub-test passes end-to-end, demonstrating the no-op path is exercised through `run()`. Note: there is no dedicated `agents_toml_absent_noop` test (the absent-file path of `detectOldSchemaAgentsTOML` is exercised transitively by `clean_state_no_flat_layout` since no `agents.toml` is seeded there).
- **Verdict:** REFUTED — no counterexample. Behavior matches the spec ("If file absent, check is a no-op"; FLAT check is a no-op when the directory is absent).

### H2 — Pre-`copyAgentFiles` ordering

- **Hypothesis:** Both detection calls fire BEFORE `copyAgentFiles`, so no partial writes can occur on failure.
- **Test:** Inspect `runInitPipeline` body at `cmd/till/init_cmd.go:492-560`. Detection block sits at lines 501-506. `copyAgentFiles` invocation sits at line 510. `copyAgentsTOML`, `ensureGitignore`, `registerMCPJSON`, `createProjectDBRecord` all follow at 514-529.
- **Finding:** Detection calls precede every file-copy side effect. `os.Getwd()` (line 493) is read-only. The pipeline returns early via `return err` (lines 502, 505) on detection failure, so no `copyAgentFiles` / `copyAgentsTOML` / `ensureGitignore` / `registerMCPJSON` / DB-record writes occur. CONSUMER-TIE tests `flat_layout_present`, `old_schema_first_line`, `old_schema_within_first_20_lines` all assert `err != nil` and rely on the early-return semantics.
- **Verdict:** REFUTED — no counterexample.

### H3 — Exact error string match

- **Hypothesis:** Implementation matches the spec's exact error strings with `<destDir>` placeholder interpolated correctly.
- **Test:** Compare spec lines 120-121 vs implementation lines 426-427 (FLAT) and 466-467 (old-schema). For FLAT, `agentsDir := filepath.Join(destDir, ".tillsyn", "agents")` yields `<destDir>/.tillsyn/agents` (NO trailing slash).
- **Finding (NIT):**
  - Spec FLAT message: `"FLAT agent layout detected at <destDir>/.tillsyn/agents/. Remove it..."` — there is a trailing `/` immediately before the period after `agents`.
  - Impl FLAT message: produces `"FLAT agent layout detected at <destDir>/.tillsyn/agents. Remove it..."` — NO trailing slash before the period. The `filepath.Join` output omits the trailing separator.
  - Old-schema message: spec and impl match exactly (`rm %s` with `tomlPath = filepath.Join(destDir, "agents.toml")` → `rm <destDir>/agents.toml`).
- **Verdict:** PASS WITH NIT (N1, low severity, cosmetic). Tests use `strings.Contains(err.Error(), "FLAT agent layout")` so they do not enforce the trailing-slash detail.

### H4 — `[agents.` prefix exactness

- **Hypothesis:** Prefix detection must match `[agents.build]`, must NOT match `[agents]` (no dot), `[go.build]` (new schema), or non-`[agents.` content.
- **Test:** Inspect `strings.HasPrefix(strings.TrimSpace(sc.Text()), "[agents.")` at line 464. Cross-check with test `no_dot_agents_section_not_old_schema` (line 1303) which seeds `[agents]\n...` and expects no error.
- **Finding:** The match is byte-exact for `[agents.` (length 8 including the dot). `[agents]` fails the prefix check. `[ agents.build]` (whitespace inside brackets) would also fail the prefix check because `TrimSpace` strips only leading whitespace from the WHOLE LINE, not whitespace inside brackets — see N2 below.
- **Verdict:** PASS WITH NIT (N2, low severity). The spec says "trimmed leading whitespace" which the implementation reads literally (line-leading whitespace), so this is spec-consistent. However, TOML permits `[ agents.build ]` with inner whitespace, which would not be detected. The probability of a user writing TOML with inner-bracket whitespace AND being on the legacy schema is low; not worth widening the detection prefix.

### H5 — First-20-lines heuristic edge case

- **Hypothesis:** `[agents.X]` appearing at line 21+ is intentionally missed; documentation must call this out.
- **Test:** Read doc-comment on `detectOldSchemaAgentsTOML` at `cmd/till/init_cmd.go:439-452`. Cross-check test `old_schema_beyond_20_lines_not_detected` (line 1359) which seeds 25 comment lines + `[agents.plan]` at line 26 and expects no error.
- **Finding:** Doc-comment lines 450-452 explicitly document: "The 20-line bound is documented in W2.D2 RiskNotes: a user with a very long comment block (> 20 lines) before the first section header will not be detected. 20 lines is considered a reasonable pragmatic bound." Test `old_schema_beyond_20_lines_not_detected` verifies the heuristic correctly skips beyond-line-20 matches.
- **Verdict:** REFUTED — no counterexample. Heuristic documented + tested.

### H6 — CONSUMER-TIE via `run()` end-to-end

- **Hypothesis:** Tests must drive `run(ctx, args, &out, io.Discard)` end-to-end, not call `runInitPipeline` directly.
- **Test:** Read `cmd/till/init_cmd_test.go:1240-1243, 1260-1263, 1290-1293, 1318-1321, 1346-1349, 1375-1378`. Every assertion goes through `run(context.Background(), []string{"--app", "tillsyn-init", "init", "--json", ...}, &out, io.Discard)`.
- **Finding:** All five sub-tests exercise the full cobra → init command → `runInitJSON` → `runInitPipeline` chain via `run()`. CONSUMER-TIE satisfied.
- **Verdict:** REFUTED — no counterexample.

### H7 — D5 placement decision survives refactor

- **Hypothesis:** Detection lives in `runInitPipeline` (not nested in `copyAgentFiles`) and has no reverse coupling to `copyAgentFiles`, so D5's `copyAgentFiles` rewrite can proceed without re-touching D2's work.
- **Test:** Inspect `detectFLATLayout` and `detectOldSchemaAgentsTOML` bodies at lines 418-477. Verify only `destDir`, stdlib (`os`, `bufio`, `errors`, `fs`, `filepath`, `strings`, `fmt`), and no references to `copyAgentFiles`, `templates.DefaultTemplateFS`, `agentFileInitPerm`, `fsatomic.WriteFile`, or any other init-pipeline helper.
- **Finding:** Both functions are pure stdlib + a single `destDir` argument. No reverse coupling. `runInitPipeline` calls them at lines 501-506, sequentially before line 510 (`copyAgentFiles`). When D5 changes `copyAgentFiles` signature from `(destDir, group string)` to `(destDir string, groups []string)`, the detection calls at 501-506 are unaffected. Decision documented in doc-comment at lines 415-417.
- **Verdict:** REFUTED — no counterexample.

### H8 — Error wrapping / sentinel consistency

- **Hypothesis:** Detection errors should compose cleanly with the rest of `runInitPipeline`'s error wrapping pattern (which uses `"till init: ...: %w"` prefix throughout).
- **Test:** Compare detection error formats vs adjacent errors. Lines 426-427 (FLAT) produce `"FLAT agent layout detected at ..."`. Lines 466-467 (old-schema) produce `"agents.toml uses the old [agents.kind] schema. ..."`. Compare to line 495 (`"till init: resolve cwd: %w"`), 512 (`"till init: copy agent files: %w"`), 516 (`"till init: copy agents.toml: %w"`), 519 (`"till init: ensure .gitignore: %w"`), 524 (`"till init: register .mcp.json: %w"`), 529 (`"till init: create project DB record: %w"`).
- **Finding:** Detection errors do NOT follow the "till init: ..." prefix convention used by every other error in `runInitPipeline`. They are user-facing remediation messages, designed to read as instructions ("FLAT agent layout detected ... Remove it and re-run: ..."). The spec mandates the exact error strings verbatim, so wrapping with "till init: " would violate spec. There is no sentinel for `errors.Is` testing, but spec did not require one — tests use substring matching, which is the documented contract.
- **Verdict:** PASS — spec-consistent. Style inconsistency with surrounding errors is justified by the user-facing remediation-instruction design and the spec's verbatim-string requirement. Not a NIT — the spec's exact-string mandate at lines 120-121 wins over local style consistency.

### H9 — Permission errors (stat/open propagation)

- **Hypothesis:** Non-`ErrNotExist` errors from `os.ReadDir` and `os.Open` must propagate, not be silently treated as no-ops.
- **Test:** Inspect default branch at line 434-435 (`detectFLATLayout`) and line 474-475 (`detectOldSchemaAgentsTOML`). Confirm both return wrapped `%w` errors.
- **Finding:** `detectFLATLayout` default branch wraps as `"till init: stat %q: %w"`. `detectOldSchemaAgentsTOML` default branch wraps as `"till init: open %q: %w"`. Both use `%w` so `errors.Is` chain semantics are preserved. Tests do not exercise EACCES paths directly (running as a privileged-enough user on a temp dir is hard to set up portably), but the code path is correct.
- **Verdict:** REFUTED — no counterexample.

### H10 — YAGNI / over-engineering

- **Hypothesis:** Builder may have added abstractions, helpers, or interface boundaries beyond what the spec requires.
- **Test:** Inspect the diff. Two new functions: `detectFLATLayout` (29 lines) + `detectOldSchemaAgentsTOML` (24 lines). Two call sites inserted into `runInitPipeline`. Five new test sub-tests. No new types, no new files, no interfaces, no helpers.
- **Finding:** Minimal change. Exactly the surface the spec asked for. No YAGNI violation.
- **Verdict:** REFUTED — no counterexample.

## Test verification

- `mage test-pkg ./cmd/till` → 308/308 PASS (matches builder claim).
- `mage test-func ./cmd/till TestRunInitPipeline_FLATDetection` → 3/3 PASS.
- `mage test-func ./cmd/till TestRunInitPipeline_OldSchemaDetection` → 5/5 PASS.

## Unmitigated Counterexamples

None found.

## NITs

### N1 — FLAT error message missing trailing slash before period (cosmetic, low severity)

- **Location:** `cmd/till/init_cmd.go:426-427`.
- **Description:** Spec at PLAN.md line 120 says the error is `"FLAT agent layout detected at <destDir>/.tillsyn/agents/. Remove it..."`. The implementation produces `"FLAT agent layout detected at <destDir>/.tillsyn/agents. Remove it..."` — the trailing `/` before the period is missing because `filepath.Join(destDir, ".tillsyn", "agents")` omits the trailing separator.
- **Severity:** Low. Cosmetic. The remediation instruction `rm -rf <destDir>/.tillsyn/agents` is unchanged (spec doesn't have a trailing slash there either), so users will copy-paste the correct command. Tests use substring `"FLAT agent layout"` only, so they do not enforce this.
- **Recommended action:** Either (a) accept as documented spec drift (test the actual string in a future hardening pass), or (b) change line 427's first `%s` to `%s/` to literally emit `<destDir>/.tillsyn/agents/.` — half-a-character fix in next-round-NIT mode. Recommend (b) since NITs are first-class.

### N2 — `[agents.` prefix detection misses inner-bracket whitespace `[ agents.build ]` (low severity, edge case)

- **Location:** `cmd/till/init_cmd.go:464`.
- **Description:** `strings.TrimSpace(line)` strips line-leading/trailing whitespace, not whitespace inside the brackets. TOML permits `[ agents.build ]` (whitespace inside brackets) as a valid section header. The current detection skips this form.
- **Severity:** Low. Spec at PLAN.md line 130 reads "any line stripped of leading whitespace starts with `[agents.`" — verbatim implementation. Probability of a user (a) hand-formatting TOML with inner-bracket whitespace AND (b) being on the legacy `[agents.kind]` schema is low. Worth surfacing as a NIT in case the dev wants to widen the detection to also try `strings.HasPrefix(line, "[")` + `strings.Contains(line, "agents.")` style.
- **Recommended action:** Document as a known edge case in the doc-comment OR widen detection. Recommend documenting only — widening risks false positives.

## Verdict rationale

W2.D2's two pre-flight checks are correctly placed, correctly bounded, correctly no-op on missing inputs, correctly propagate other I/O errors, correctly free of D5 coupling, and correctly tested via `run()` end-to-end. Five CONSUMER-TIE sub-tests exercise all three spec-mandated scenarios (FLAT present, old-schema present, clean state) plus two extra edge cases (inner 20-line boundary, beyond-20-line boundary). `mage test-pkg ./cmd/till` 308/308 PASS.

Two cosmetic NITs found: N1 (missing trailing slash before period in FLAT error string — emit `<destDir>/.tillsyn/agents/.` to match spec verbatim) and N2 (inner-bracket-whitespace TOML edge case in old-schema detection — document or accept). Both are surface-level, neither blocks D3-D7 dispatch.

**Overall verdict: PASS WITH NITS.** N1 is a recommended next-round-NIT fix; N2 is a documentation NIT only.
