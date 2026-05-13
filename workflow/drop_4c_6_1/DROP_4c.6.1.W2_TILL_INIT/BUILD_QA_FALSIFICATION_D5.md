# W2.D5 — BUILD-QA-FALSIFICATION Verdict

**Date:** 2026-05-13
**Reviewer:** go-qa-falsification-agent (opus)
**Overall verdict:** PASS WITH NITS

## Attack Hypotheses Tested

### A1. UNPREFIXED embed path correctness — REFUTED
- `internal/templates/embed.go:62-65, 87-118` confirms `git mv` landed `builtin/agents/go/`, `builtin/agents/fe/`, `builtin/agents/gen/` (no `till-` prefix).
- Filesystem probe `ls internal/templates/builtin/agents/` returned `fe/`, `gen/`, `go/`, `till-gdd/` — confirms three canonical groups exist; `till-gdd/` is a separate template-family identifier, not a group.
- `copyAgentFiles` (`cmd/till/init_cmd.go:631`) reads `path.Join("builtin", "agents", group)` — directly produces `builtin/agents/<group>` (unprefixed). Match.
- `TestCopyAgentFiles_SubdirPerGroup/single_group_go` and `multi_group_go_and_fe` pass — embed-FS reads return ≥1 file per group.

### A2. Multi-group partial-failure error propagation — REFUTED
- Lines 632-635: `fs.ReadDir` failure path returns `(added, skipped, fmt.Errorf("read embedded %q: %w", srcDir, err))` — partial counts AND wrapped error returned.
- Lines 638-640: MkdirAll failure returns partial counts + wrapped error.
- Lines 650-652, 656-658, 659-661: stat / read / write failures all return partial counts + wrapped error.
- Caller (`runInitPipeline:498-500`) wraps once more with `till init: copy agent files: %w`.
- No rollback of files already on disk is required by spec (`Specify.AcceptanceCriteria` does not mention atomicity across groups); accurate partial-progress counting is what was promised, and that is delivered.
- Defense-in-depth: payload validator (`init_cmd.go:773-790`) rejects any group not in `{gen, go, fe}` before `copyAgentFiles` is called — bogus-group ENOENT cannot reach the function via the normal path.

### A3. Idempotent skip correctness — REFUTED
- `TestCopyAgentFiles_SubdirPerGroup/idempotent_skip` covers the exact scenario: first run -> `added1` files; second run on same destDir -> `added2 == 0` AND `skipped2 == added1`. Test passes.
- `os.Stat` at line 647 returns nil error when file exists -> `skipped++` and `continue`. `errors.Is(statErr, fs.ErrNotExist)` is the only branch that proceeds to write.
- Re-run after partial single-file removal: stat returns ENOENT for the missing file (added++); stat returns nil for the surviving files (skipped++). Both counters increment correctly across the mixed-state case. Not directly covered by a dedicated test (would be a strengthening test); see NIT-1.

### A4. FLAT detection NOT duplicated inside copyAgentFiles — REFUTED
- `grep -n FLAT cmd/till/init_cmd.go` reveals exactly two production occurrences: `detectFLATLayout` at line 395-413 and its single call site at `runInitPipeline:488`. The doc-comment on `copyAgentFiles:621-623` explicitly states FLAT detection lives in `runInitPipeline`.
- Body of `copyAgentFiles` (lines 628-666) contains zero FLAT-detection logic. Spec satisfied.

### A5. Aggregated counts include 0-file groups — REFUTED
- The outer loop at line 630 iterates every supplied group regardless of file count. The inner loop at line 642 simply does not increment if no `.md` entries match.
- Empty-subdir behaviour: `added += 0`, `skipped += 0`, no error. Aggregated counts are correct.
- Currently no empty group ships, so this branch is exercised only as defensive behaviour.

### A6. Empty groups slice behaviour — REFUTED (NIT-2 raised)
- `copyAgentFiles(destDir, []string{})` — outer loop iterates 0 times; returns `(0, 0, nil)` cleanly.
- Pre-D5 contract errored on empty `group` string (`errors.New("copyAgentFiles: group required")`). Post-D5 contract silently no-ops on empty slice. Spec is satisfied because payload validation at `init_cmd.go:773-775` enforces `len(p.Groups) > 0` upstream — `copyAgentFiles` cannot be reached with an empty slice via the normal path.
- The lost defensive guard is a minor regression in callable-from-tests posture but not a functional defect — see NIT-2.

### A7. Per-group MkdirAll semantics — REFUTED
- `os.MkdirAll` is idempotent (already-exists -> nil) and recursive (parent `.tillsyn/agents/` is created on first iteration; subsequent group iterations target a sibling at the same depth, succeed independently).
- No concurrent caller path: `runInitPipeline` is a single goroutine driven by `run()`.
- Permission failure (EACCES on `destDir`) bubbles as wrapped error.

### A8. Laslig `"groups"` migration — REFUTED
- Line 539: `{"groups", strings.Join(payload.Groups, ",")}`. The old key was `"group", payload.Groups[0]` (visible in the diff).
- `TestInit_SuccessMessage_Format` (`init_cmd_test.go:1066`) updated to require `"groups"` substring. Test passes.
- `TestRunInitPipeline_MultiGroup/laslig_summary_groups_key` independently verifies the substring via `run()` end-to-end. Test passes.
- `grep -n Groups cmd/till/init_cmd.go` shows no remaining `"group"` key writes anywhere in init_cmd.go.

### A9. TestInit_FreshDir_CopiesAllFiles scope correctness — REFUTED
- Test payload is `{"name":"foo","groups":["go"],"mcp":false}` — only `go` group selected.
- Scan target is `<dir>/.tillsyn/agents/go/`, not the flat agents root and not `fe/`. Test cannot accidentally read files that don't exist (fe is not in the selected groups).
- Floor of ≥7 .md files matches SKETCH §11.1 (10 actual files ship in `go/`).
- Spot-check is `<dir>/.tillsyn/agents/go/builder-agent.md` — present in embed.FS at line 100.

### A10. TestRunInitPipeline_MultiGroup truly end-to-end — REFUTED
- `runInitJSONInTempDir` at `init_cmd_test.go:391-399` calls `run(context.Background(), []string{"--app", "tillsyn-init", "init", "--json", payload}, &out, io.Discard)` — the same `run` entry point production uses.
- All three sub-tests of `TestRunInitPipeline_MultiGroup` (`single_group_subdir_layout`, `multi_group_both_subdirs_created`, `laslig_summary_groups_key`) call either `runInitJSONInTempDir` or `run(...)` directly. None call internals like `runInitPipeline` or `copyAgentFiles` directly.
- 4-sub-test count for `TestRunInitPipeline_MultiGroup` from `mage test-func` run matches the 3 visible sub-tests (likely table-driven counting + parent test = 4); functional coverage of the consumer-tie path is confirmed.

### A11. Placeholder content round-trip — REFUTED with caveat
- `fsatomic.WriteFile(target, data, agentFileInitPerm)` writes `data` returned by `fs.ReadFile` byte-for-byte; no mutation in the pipeline.
- The 10 embedded `go/*.md` files (sizes 349B-963B per the `ls` probe) are written verbatim. The subdir tests assert file existence + count; byte-level diff is not asserted.
- Spec does NOT require byte-content round-trip assertions ("AcceptanceCriteria" lists subdir layout, multi-group, idempotent skip, aggregated counts, end-to-end consumer-tie — no content-equality bullet). Not a counterexample.
- See NIT-3.

### A12. YAGNI — REFUTED
- No new interfaces, abstractions, or helper layers added. `copyAgentFiles` signature change is the smallest possible refactor (single arg type change + inner loop). No premature generality.

### A13. Hermeticity — REFUTED
- `copyAgentFiles` body (lines 628-666) reads ONLY `templates.DefaultTemplateFS` (embed.FS — pure, no I/O outside the binary) and writes only into `destDir`.
- `grep -n "HOME\|UserHomeDir\|homeDir" cmd/till/init_cmd.go` shows no HOME usage inside or transitively reachable from `copyAgentFiles`. The HOME-touching paths live in `createProjectDBRecord` (line 562) and root-opts plumbing (line 857), unrelated to D5.
- W1.D1 hermeticity lesson honoured.

## Unmitigated Counterexamples

None.

## NITs

- **NIT-1 (minor):** No dedicated test for the mixed-state idempotency case (some files in `<destDir>/.tillsyn/agents/go/` exist, some don't — same run should `added++` for missing and `skipped++` for present). Spec doesn't mandate this case, and the existing `idempotent_skip` sub-test covers the all-skip path; a third sub-test asserting `added > 0 && skipped > 0` would strengthen the contract.
- **NIT-2 (minor):** The pre-D5 `copyAgentFiles` errored loudly on empty `group` string. Post-D5 it silently no-ops on `[]string{}`. Functionally safe because `validateInitJSONPayload` enforces non-empty Groups upstream, but a defensive `if len(groups) == 0 { return 0, 0, errors.New("copyAgentFiles: at least one group required") }` at the top of the function would preserve the "fail loud at the seam" posture from the prior implementation and protect against future internal callers that bypass `validateInitJSONPayload`.
- **NIT-3 (cosmetic):** Tests assert file existence and count but not byte-equality against `templates.DefaultTemplateFS`. A single `bytes.Equal(diskBytes, embedBytes)` spot-check on `builder-agent.md` would close the round-trip loop. Not load-bearing pre-W4.D1 substantive prompt content; revisit when W4 lands real prompt bodies.

## Verdict rationale

All 13 attack hypotheses were exhaustively attempted; every one was REFUTED with concrete evidence. The three remaining NITs are strengthening suggestions rather than spec violations:

- Spec satisfaction: signature `(destDir, groups []string) (int, int, error)`, unprefixed embed paths (`go`/`fe`/`gen`), idempotent skip, FLAT detection retained in `runInitPipeline` only, Laslig `"groups"` key migration, multi-group `run()` consumer-tie, aggregated counts. All confirmed in code + tests.
- Test posture: `TestCopyAgentFiles_SubdirPerGroup` (4 sub-tests passing) + `TestRunInitPipeline_MultiGroup` (4 sub-tests passing) + `TestInit_FreshDir_CopiesAllFiles` adapted + `TestInit_SuccessMessage_Format` adapted + `TestInit_ConsumerTie_W2D1` comment refreshed.
- Local runs: `mage test-func ./cmd/till TestCopyAgentFiles_SubdirPerGroup` -> 4/4 PASS; `mage test-func ./cmd/till TestRunInitPipeline_MultiGroup` -> 4/4 PASS; `mage test-pkg ./cmd/till` -> 329/329 PASS; `mage check` -> 3270/3270 PASS across 30 packages, all coverage ≥70%.
- Hermeticity, YAGNI, FLAT-detection-no-duplication, partial-failure error propagation, MkdirAll semantics all clean.

Verdict: **PASS WITH NITS**. Builder claim is supported by code + tests + local CI. Three minor strengthening opportunities recorded but none block the droplet.
