# W2.D6 — BUILD-QA-FALSIFICATION Verdict

**Date:** 2026-05-13
**Reviewer:** go-qa-falsification-agent (opus)
**Overall verdict:** PASS WITH NITS

## Attack Hypotheses Tested

### A1 — HOME-tier garbage propagates as parse error (REFUTED, by design)
- **Hypothesis:** If `~/.tillsyn/templates/go.toml` exists with garbage content, does `readTemplateForGroup` propagate the parse error and break the run, or does it fall through to embedded?
- **Code:** `readTemplateForGroup` at `cmd/till/init_cmd.go:723-739` does `os.ReadFile(homePath)`. If `err == nil` it returns the raw bytes regardless of content. No TOML parse, no validation.
- **Trace:** HOME file exists with `"!!!not valid toml!!!"` → `os.ReadFile` returns non-nil bytes + nil error → function returns those bytes → caller writes them under `[go]\n` header into `template.toml`. No error raised. Downstream consumer (not yet wired) would fail to parse, but that's a future-drop concern.
- **Verdict:** REFUTED. Spec line 308 says "source is HOME tier (...) if present, else embedded binary default" — no validation requirement. The HOME-tier acts as a verbatim override. This is the documented contract.

### A2 — `[go.X]` covers but no top-level `[go]` (REFUTED)
- **Hypothesis:** If existing `template.toml` contains `[go.build]` but no literal `[go]` header, does partial-state check correctly mark "go" as covered?
- **Code:** `cmd/till/init_cmd.go:676`:
  ```go
  if !strings.Contains(content, "["+group+"]") && !strings.Contains(content, "["+group+".") {
  ```
- **Trace:** Content `[go.build]\nfoo = 1\n` → `strings.Contains(content, "[go.")` returns true → `go` NOT added to `missing`. No warning.
- **Verdict:** REFUTED. The disjunction correctly covers the dotted sub-table case.

### A3 — Empty/corrupt `[go]` section (REFUTED, by design)
- **Hypothesis:** File with literal `[go]\n# nothing here\n` but no actual content under the section — does simple string check incorrectly mark it as covered?
- **Code:** Same simple `strings.Contains` check as A2.
- **Trace:** Yes — `strings.Contains("[go]\n# nothing\n", "[go]")` is true → "covered". The check is intentionally a presence heuristic per spec line 320: "Simple string check — not full TOML parse."
- **Verdict:** REFUTED (documented tradeoff). Spec line 325 ContextBlocks `decision` accepts this explicitly: "Simpler and consistent with 'no migration, fail loud' philosophy."

### A4 — `rootOpts.homeDir` flag-override path is not exercised by tests (CONFIRMED — but test-only gap, NIT)
- **Hypothesis:** Tests rely on `t.Setenv("HOME", dir)` + `os.UserHomeDir()` fallback. The `--home` flag override path (`opts.homeDir != ""`) is not directly tested.
- **Code:** `runInitPipeline` at `cmd/till/init_cmd.go:497-503`:
  ```go
  homeDir := strings.TrimSpace(opts.homeDir)
  if homeDir == "" {
      homeDir, err = os.UserHomeDir()
      ...
  }
  ```
- **Trace:** No test passes `--home` flag through the `run()` argv. All four sub-tests rely on the `os.UserHomeDir()` fallback branch. The `opts.homeDir != ""` branch is dead in test coverage.
- **Verdict:** CONFIRMED minor gap. Acceptance line 314 says "homeDir override via `rootOpts.homeDir` for test isolation" — this is asserted in spec but not directly exercised. **NIT** — adding an `--home` cli flag invocation in one sub-test would close it. Not load-bearing because the HOME env fallback path is tested and behaviorally identical.

### A5 — Multi-group aggregation produces non-namespaced TOML (CONFIRMED — design issue, NIT)
- **Hypothesis:** Spec line 301 says "Aggregated TOML content is written". For groups=["go","fe"] the output should produce a parseable TOML file with each group's content scoped under `[go.*]` / `[fe.*]`. Current code prepends `[go]\n` then dumps embedded `till-go.toml` raw — but `till-go.toml` itself contains top-level tables like `[kinds.plan]`.
- **Evidence:** `internal/templates/builtin/till-go.toml` line 66+: `[kinds.plan]`, `[kinds.research]`, etc. (43 top-level section headers, none prefixed with `go.`).
- **Code:** `writeTemplateTOML` at `cmd/till/init_cmd.go:691-707` prepends `[group]\n` then writes the raw embedded content. There is NO header rewrite to namespace `[kinds.plan]` → `[go.kinds.plan]`.
- **Trace:** Final aggregated file for `["go","fe"]` would be:
  ```
  [go]
  # comment
  [kinds.plan]   ← This is a TOP-LEVEL table, NOT inside [go]
  ...
  [kinds.build]
  ...
  
  [fe]
  # comment
  [kinds.plan]   ← TOML parse ERROR: duplicate table [kinds.plan]
  ...
  ```
- **Verdict:** CONFIRMED **design issue**, but NIT-class for this drop because:
  - No multi-group consumer is wired (template resolver hasn't been built yet — future drop tracks PLATFORM-TEMPLATES-R1).
  - The four CONSUMER-TIE tests are all single-group `["go"]`, so the aggregation collision is not exercised.
  - Spec ContextBlocks line 326 says "platform.Paths.TemplatesDir does not exist; construct HOME templates path directly" — flags that downstream is not yet built.
  - Calling `till init --json '{"groups":["go","fe"]}'` today would WRITE a file but if anyone later parses it strictly they'll hit duplicate-table errors.
- **NIT routing:** Worth raising as W2-D6-R1 refinement (or W4.D2 consideration) — either (a) the aggregator must rewrite section headers to namespace under `[<group>.X]`, OR (b) the embedded `till-<group>.toml` files must already be namespaced `[<group>.kinds.plan]` etc. before W4.D2 ships.

### A6 — Embedded fallback path correctness (REFUTED)
- **Hypothesis:** Builder verifies `builtin/till-go.toml`, `builtin/till-fe.toml`, `builtin/till-gen.toml` exist.
- **Evidence:** `ls internal/templates/builtin/` confirms all three TOML files exist with sizes 24.9K / 12.9K / 14.4K respectively.
- **Code:** Line 733: `path.Join("builtin", "till-"+group+".toml")` correctly produces `builtin/till-go.toml` for group=`go`.
- **Verdict:** REFUTED. Live `mage test-func TestWriteTemplateTOML_HOMETierAbsent` passes which exercises the embedded fallback path end-to-end.

### A7 — Idempotency byte-for-byte preservation (REFUTED)
- **Hypothesis:** On re-run with existing file, is the file actually untouched?
- **Code:** Lines 670-685: `os.ReadFile` succeeds → check missing → `return 0, 1, nil`. No `WriteFile` or `Open` for write. mtime preserved; bytes unchanged.
- **Test:** `TestWriteTemplateTOML_Idempotent` reads file before second run, asserts `string(firstData) != string(secondData)` is false → byte-equal.
- **Verdict:** REFUTED.

### A8 — Partial-state warning format (REFUTED)
- **Hypothesis:** Warning string must contain `"WARN:"` prefix + the full destDir path + the missing-list in brackets.
- **Code:** Line 681: `fmt.Fprintf(stdout, "WARN: %s already exists but is missing sections for group(s): %v. Remove it and re-run to regenerate.\n", target, missing)`.
- **Trace:** `target` is `<destDir>/.tillsyn/template.toml`; `missing` is `[]string{"go"}` which `%v`-formats as `[go]`. Matches spec line 302 format.
- **Verdict:** REFUTED.

### A9 — Non-fatal partial-state warning (exit zero) (REFUTED)
- **Hypothesis:** Warning path must return nil error and overall run exits zero.
- **Code:** Line 684: `return 0, 1, nil` (nil error).
- **Test:** `TestWriteTemplateTOML_PartialStateWarning` asserts `run(...)` returns `nil`.
- **Verdict:** REFUTED.

### A10 — Sentinel error vs warning (REFUTED)
- **Hypothesis:** Partial-state path returns a special "warning" sentinel rather than nil.
- **Code:** Returns plain `nil` error. Warning surfaces only via stdout side effect.
- **Verdict:** REFUTED.

### A11 — Empty / nil groups slice no-op (REFUTED)
- **Hypothesis:** What happens if `groups = []string{}` or `nil`?
- **Code:** `validateInitPayload` at line 903-905 rejects `len(p.Groups) == 0` before `runInitPipeline` is reached: `"till init: groups required (must supply at least one group)"`.
- **Verdict:** REFUTED — empty groups can never reach `writeTemplateTOML` through the validated `runInitJSON`/`runInitTUI` callers.

### A12 — Empty homeDir silently skips HOME tier (REFUTED)
- **Hypothesis:** What if `homeDir = ""` reaches `writeTemplateTOML`?
- **Code:** `runInitPipeline` line 497-503 explicitly traps empty `opts.homeDir` and calls `os.UserHomeDir()` fallback (returning the OS-resolved home). If `os.UserHomeDir()` itself fails, `runInitPipeline` returns an error before `writeTemplateTOML` is called.
- **Verdict:** REFUTED — `homeDir` is always non-empty when `writeTemplateTOML` is invoked.

### A13 — `## Hylla Feedback` section in builder report (UNVERIFIABLE)
- **Hypothesis:** Builder report should NOT contain a `## Hylla Feedback` section (Hylla is OFF).
- **Evidence:** `BUILDER_WORKLOG.md` was not updated with a D6 entry at the time of this review — the worklog stops at W2.D4. Cannot verify the absence/presence of a Hylla Feedback section in the builder's D6 report.
- **Verdict:** UNVERIFIABLE. **NIT** — flag for orchestrator: builder may not have updated the worklog before declaring done. Process gap, not a code defect.

### A14 — Empty existing template.toml triggers WARN for every group (CONFIRMED, EDGE NIT)
- **Hypothesis:** If a 0-byte `template.toml` exists, every group is marked missing.
- **Code:** `os.ReadFile` returns `([], nil)` for empty file → `content = ""` → `strings.Contains("", "[X]")` is false for all groups → all groups WARN'd.
- **Trace:** This is technically correct behavior (zero sections = all missing) but might confuse users who accidentally `touch .tillsyn/template.toml`.
- **Verdict:** CONFIRMED edge-case behavior but ACCEPTED — spec line 302 says "missing sections for group(s)" — empty file legitimately has zero sections. Minor NIT.

### A15 — W4.D2 future-rename fragility (DEFERRED, NOT a counterexample today)
- **Hypothesis:** Embedded path `builtin/till-<group>.toml` is hardcoded. Spec RiskNote line 321 says "Builder verifies the exact embed path via `fs.ReadDir` ... after W4.D2 ships. Do not hard-code embed paths before W4.D2 completes."
- **Evidence:** W4.D2 has not shipped; the live filenames `till-go.toml`, `till-fe.toml`, `till-gen.toml` match the hardcoded path today. Tests pass live.
- **Verdict:** DEFERRED. W4.D2 dispatch needs to re-verify and update if rename occurs. Spec already calls this out — not a defect of D6.

## Unmitigated Counterexamples

**None.** All 13 numbered attack hypotheses plus 2 supplementary attacks either REFUTED outright (10 of 15) or CONFIRMED as NIT-class issues that do not break the W2.D6 acceptance contract:

- A4 (homeDir flag-override test coverage gap) — NIT, test-only
- A5 (multi-group aggregation produces non-namespaced TOML) — NIT, no consumer wired
- A13 (`## Hylla Feedback` in builder report) — NIT, process-level, unverifiable from worklog state
- A14 (empty existing file WARN cascade) — NIT, technically correct
- A15 (W4.D2 future-rename fragility) — DEFERRED, spec-acknowledged

`mage test-pkg ./cmd/till` is GREEN at 333/333. All four CONSUMER-TIE sub-tests required by spec line 305 pass.

## NITs

1. **NIT-D6-1 (A4):** Test suite does not directly exercise the `--home` flag override path (`opts.homeDir != ""`). All four tests rely on `t.Setenv("HOME", ...)` + `os.UserHomeDir()` fallback. Adding one sub-test that passes `--home <dir>` through the argv would close the spec gap on "homeDir override via `rootOpts.homeDir` for test isolation."

2. **NIT-D6-2 (A5):** The aggregated `template.toml` for multi-group `["go","fe"]` writes `[go]\n<raw till-go.toml content>\n\n[fe]\n<raw till-fe.toml content>` — but `till-go.toml` contains 43 top-level `[kinds.X]` headers that are NOT namespaced under `[go.]`. A strict TOML parser would either (a) see `[kinds.plan]` as a top-level table that escapes from `[go]`, or (b) hit duplicate-table errors when `[fe]`'s content also defines `[kinds.plan]`. Worth raising as W2-D6-R1 (or W4.D2) refinement — either rewrite headers in the aggregator OR namespace embedded templates at source.

3. **NIT-D6-3 (A13):** `BUILDER_WORKLOG.md` was not updated with a D6 entry at the time of this falsification review. Worklog stops at W2.D4. Cannot independently verify the builder's claim about test counts (333/333) from the worklog. The `mage test-pkg` count was independently verified.

4. **NIT-D6-4 (A14):** A user who accidentally `touch .tillsyn/template.toml` (zero-byte file) would see a WARN listing every selected group as missing on every subsequent `till init` run. Technically correct but surprising. Optional: detect zero-byte file and treat as "absent" for the rewrite path.

## Verdict rationale

**PASS WITH NITS.** No counterexample falsifies the spec acceptance criteria. All four required CONSUMER-TIE sub-tests pass (HOME present / HOME absent / idempotent / partial-state warning). Code correctly implements: HOME-then-embedded fallback (`readTemplateForGroup`), blanket skip on existing file, simple-string partial-state check with documented heuristic tradeoff, exit-zero non-fatal warning path, `rootOpts.homeDir` plumbing via `runInitPipeline`. The four NITs are minor design/test-coverage observations that do not invalidate the droplet's acceptance bar.

The single design issue worth orchestrator attention is **NIT-D6-2**: multi-group aggregation produces a TOML file that is not safely composable for strict downstream parsing. Since no consumer is wired yet, this does not break the W2.D6 acceptance contract today, but it must be resolved before any template-resolver consumer ships (PLATFORM-TEMPLATES-R1 or sibling).
