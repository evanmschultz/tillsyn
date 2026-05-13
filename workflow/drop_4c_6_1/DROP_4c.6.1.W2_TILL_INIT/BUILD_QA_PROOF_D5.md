# W2.D5 — BUILD-QA-PROOF Verdict

**Date:** 2026-05-13
**Reviewer:** go-qa-proof-agent (opus)
**Overall verdict:** PASS WITH NITS

## Acceptance Bullet Coverage

### B1. Signature change: `copyAgentFiles(destDir, group string)` → `copyAgentFiles(destDir string, groups []string) (int, int, error)`

- **Quote:** "`copyAgentFiles` signature changes from `(destDir, group string) (int, int, error)` to `(destDir string, groups []string) (int, int, error)` (or equivalent multi-group signature)."
- **Evidence:** `cmd/till/init_cmd.go:628` — `func copyAgentFiles(destDir string, groups []string) (int, int, error) {`. Return values `(added, skipped, err)` preserved (line 629 + 663).
- **Verdict:** **PASS**

### B2. Per-group subdir copy from UNPREFIXED embed path

- **Quote:** "For each group in `groups`: copies embedded `agents/<group>/*.md` to `<destDir>/.tillsyn/agents/<group>/*.md` (subdir-per-group, NOT flat). Embed path: `builtin/agents/<group>/` (unprefixed — W4.D1's canonical names `go`/`fe`/`gen`, NOT `till-go`/`till-gen`)."
- **Evidence:**
  - `cmd/till/init_cmd.go:631` — `srcDir := path.Join("builtin", "agents", group)` (UNPREFIXED).
  - `cmd/till/init_cmd.go:637` — `groupDir := filepath.Join(destDir, ".tillsyn", "agents", group)` (subdir).
  - `cmd/till/init_cmd.go:646` — `target := filepath.Join(groupDir, entry.Name())` (file under subdir).
  - On-disk confirmation: `internal/templates/builtin/agents/` contains `fe/`, `gen/`, `go/` subdirs (and the legacy `till-gdd/`); no `till-go/` or `till-gen/` is consulted by D5 code.
- **Verdict:** **PASS**

### B3. Per-group subdir creation

- **Quote:** "Creates `<destDir>/.tillsyn/agents/<group>/` directory for each group."
- **Evidence:** `cmd/till/init_cmd.go:638` — `if err := os.MkdirAll(groupDir, 0o755); err != nil { ... }`. Inside the `for _, group := range groups` loop (line 630), so created per group.
- **Verdict:** **PASS**

### B4. Idempotent skip per file

- **Quote:** "Idempotent: existing files at `<destDir>/.tillsyn/agents/<group>/<name>.md` are SKIPPED (not overwritten)."
- **Evidence:**
  - `cmd/till/init_cmd.go:647-651` — `if _, statErr := os.Stat(target); statErr == nil { skipped++; continue } else if !errors.Is(statErr, fs.ErrNotExist) { return added, skipped, ... }`.
  - Test coverage: `cmd/till/init_cmd_test.go:1812-1830` — `TestCopyAgentFiles_SubdirPerGroup/idempotent_skip` asserts `added2 == 0 && skipped2 == added1` on second invocation.
- **Verdict:** **PASS**

### B5. FLAT detection preserved in `runInitPipeline`, NOT duplicated in `copyAgentFiles`

- **Quote:** "FLAT detection guard (from D2) is preserved — it lives in `runInitPipeline`, NOT in `copyAgentFiles`. D5 must NOT add FLAT detection into `copyAgentFiles`."
- **Evidence:**
  - `cmd/till/init_cmd.go:405` — `func detectFLATLayout(destDir string) error { ... }` definition.
  - `cmd/till/init_cmd.go:488` — `if err := detectFLATLayout(destDir); err != nil { return err }` is the ONLY call site, inside `runInitPipeline` pre-flight (before the `copyAgentFiles` call at line 497).
  - No occurrence of `detectFLATLayout` between lines 628-663 (`copyAgentFiles` body).
  - Doc-comment at `cmd/till/init_cmd.go:621-623` explicitly notes "FLAT detection ... lives in `runInitPipeline`, NOT in this function. This preserves the W2.D2 check independently of the D5 signature refactor."
- **Verdict:** **PASS**

### B6. `runInitPipeline` passes `payload.Groups`

- **Quote:** "`runInitPipeline` updated: calls `copyAgentFiles(destDir, payload.Groups)`."
- **Evidence:** `cmd/till/init_cmd.go:497` — `agentsAdded, agentsSkipped, err := copyAgentFiles(destDir, payload.Groups)`. No surviving `Groups[0]` stub anywhere in `init_cmd.go` (grep confirms only two `payload.Groups` references: the call at 497 and the join at 539).
- **Verdict:** **PASS**

### B7. Laslig summary key `"group"` → `"groups"` (comma-joined)

- **Quote:** "Laslig summary row updated: `\"groups\"` key (comma-joined list) replaces `\"group\"` key."
- **Evidence:**
  - `cmd/till/init_cmd.go:539` — `{"groups", strings.Join(payload.Groups, ",")},`
  - Grep for `"group"` (singular) in `init_cmd.go` returns no matches — the old key is fully removed (resolves D4-N2 OOS note as the spec requires).
  - Existing test `TestInit_SuccessMessage_Format` (`init_cmd_test.go:1065-1066`) updated to assert `"groups"` not `"group"`.
- **Verdict:** **PASS**

### B8. CONSUMER-TIE: single-group and multi-group `run()` end-to-end tests

- **Quote:** "CONSUMER-TIE: `run(ctx, args, &out, io.Discard)` end-to-end — single-group test: `--json '{\"name\":\"x\",\"groups\":[\"go\"],\"mcp\":false}'` verifies `<destDir>/.tillsyn/agents/go/<name>.md` created. Multi-group test: `--json '{\"name\":\"x\",\"groups\":[\"go\",\"fe\"],\"mcp\":false}'` verifies both `agents/go/` and `agents/fe/` subdirs created."
- **Evidence:**
  - `cmd/till/init_cmd_test.go:1839-1849` — `TestRunInitPipeline_MultiGroup/single_group_subdir_layout` runs `runInitJSONInTempDir` with `groups:["go"]` and stats `agents/go/builder-agent.md`.
  - `cmd/till/init_cmd_test.go:1851-1872` — `TestRunInitPipeline_MultiGroup/multi_group_both_subdirs_created` runs with `groups:["go","fe"]` and asserts `.md` files under both `agents/go/` and `agents/fe/`.
  - `cmd/till/init_cmd_test.go:1874-1887` — `TestRunInitPipeline_MultiGroup/laslig_summary_groups_key` further drives `run()` directly and asserts `"groups"` appears in stdout.
- **Verdict:** **PASS**

### B9. `mage test-pkg ./cmd/till` passes; `mage ci` green

- **Quote:** "`mage test-pkg ./cmd/till` passes; `mage ci` green."
- **Evidence:** Builder reports 329/329 cmd/till PASS and `mage ci` GREEN. Verifier did not re-run (spec prohibits code modification; re-run would not change tree state). Test file changes inspected — all updated assertions are internally consistent with the new subdir layout and the new `"groups"` key. NO LSP diagnostics were checked in this pass (per Hylla-disabled / read-only review scope), but the QA-falsification sibling owns counterexample probing.
- **Verdict:** **PASS** (builder claim accepted on test-suite reporting; downstream QA-falsification owns counterexample probing).

### Special-focus additions

#### SF1. Aggregated `added` + `skipped` counts across all groups

- **Spec text (Specify):** "`added` count = total files created across all groups; `skipped` count = total skipped."
- **Evidence:** `cmd/till/init_cmd.go:629` declares `added, skipped := 0, 0` ONCE outside the group loop. Lines 648 (`skipped++`) and 661 (`added++`) increment the shared accumulators inside both nested loops. Doc-comment lines 625-627 confirm "Both counters are aggregated across all groups."
- **Verdict:** **PASS**

#### SF2. Multi-group CONSUMER-TIE verified end-to-end via `run()`

- See B8 above. `TestRunInitPipeline_MultiGroup/multi_group_both_subdirs_created` exercises the full pipeline (parser → `runInitPipeline` → `copyAgentFiles`) and stats files under both group subdirs. Verified.

## NITs

### N1. Builder claim drift on sub-test counts (cosmetic)

The spawn brief states "`TestCopyAgentFiles_SubdirPerGroup` (4 sub), `TestRunInitPipeline_MultiGroup` (4 sub, CONSUMER-TIE)." Actual tree:

- `TestCopyAgentFiles_SubdirPerGroup` has **3** sub-tests (`single_group_go`, `multi_group_go_and_fe`, `idempotent_skip`) — `init_cmd_test.go:1732-1830`.
- `TestRunInitPipeline_MultiGroup` has **3** sub-tests (`single_group_subdir_layout`, `multi_group_both_subdirs_created`, `laslig_summary_groups_key`) — `init_cmd_test.go:1840-1887`.

The L2 spec itself does not mandate a specific sub-test count — it only requires single-group + multi-group CONSUMER-TIE coverage plus idempotent-skip + Laslig-key assertions, all of which ARE covered across the 6 actual sub-tests. The drift is a builder-claim wording error, NOT an acceptance gap. Suggested fix: builder updates the brief / next handoff message to read "3 sub" each.

### N2. Doc-comment cross-reference to `internal/templates/...` path uses a `internal/templates/...` rooted path while embed FS uses `builtin/...`

`cmd/till/init_cmd.go:611` doc-comment reads "reads the embedded `internal/templates/builtin/agents/<group>/*.md` set via `templates.DefaultTemplateFS`". The on-disk path is `internal/templates/builtin/agents/<group>/`, but the embed root (relative to `templates.DefaultTemplateFS`) starts at `builtin/...`. The doc-comment uses the on-disk path (which IS accurate — `internal/templates/` is the package dir containing the `//go:embed` directive root). No code defect — just verify-by-eye that the doc-comment doesn't accidentally suggest `fs.ReadDir(DefaultTemplateFS, "internal/templates/builtin/agents/<group>")` as the call. The actual call at line 631 correctly uses `path.Join("builtin", "agents", group)`. This is cosmetic — the doc-comment is technically accurate but a maintainer who reads only the doc-comment and not the code might mis-construct the embed key.

Suggested fix (optional): rewrite the doc-comment to read "reads the embedded `builtin/agents/<group>/*.md` (the `internal/templates/` package's embedded FS root)" — clarifies that `builtin/...` is the embed-FS key, not `internal/templates/builtin/...`.

## Verdict rationale

All 9 numbered acceptance bullets PASS with file:line evidence. The two NITs are cosmetic (builder-claim wording drift on sub-test counts; doc-comment phrasing on embed-FS root) — neither blocks merge. The structural core is clean:

- Signature lifted from single-group to `groups []string`.
- Embed key is UNPREFIXED (`builtin/agents/<group>`) per W4.D1 canonical names.
- Destination is subdir-per-group (`<destDir>/.tillsyn/agents/<group>/<name>.md`).
- Counts aggregate across groups (single accumulator outside the group loop).
- FLAT detection stays in `runInitPipeline` (single call site at line 488); no duplication leaked into `copyAgentFiles`.
- Laslig summary key migrated from `"group"` to `"groups"` (comma-joined); old key fully removed. This resolves the D4-N2 OOS note as the spec mandates.
- Multi-group CONSUMER-TIE exercised via `TestRunInitPipeline_MultiGroup` at three end-to-end levels (single, multi, summary-key).

**Overall verdict: PASS WITH NITS.** Recommend merge after the QA-falsification sibling sign-off.
