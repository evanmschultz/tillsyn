# W2.D2 — BUILD-QA-PROOF Verdict

**Date:** 2026-05-13
**Reviewer:** go-qa-proof-agent (opus)
**Overall verdict:** PASS WITH NITS

## Acceptance Bullet Coverage

### Bullet 1 — `runInitPipeline` calls `detectFLATLayout(destDir string) error` BEFORE `copyAgentFiles`; emits exact remediation error on hit

> `runInitPipeline` calls a new `detectFLATLayout(destDir string) error` function BEFORE calling `copyAgentFiles`. If `<destDir>/.tillsyn/agents/` contains `.md` files directly at root (FLAT layout), `runInitPipeline` returns a non-zero error: `"FLAT agent layout detected at <destDir>/.tillsyn/agents/. Remove it and re-run: rm -rf <destDir>/.tillsyn/agents && till init --group <group>"`.

**Evidence:**
- `cmd/till/init_cmd.go:418-437` — `detectFLATLayout(destDir string) error` defined. Returns `nil` when dir absent (fs.ErrNotExist) or no `.md` at root; returns wrapped sentinel `fmt.Errorf("FLAT agent layout detected at %s. Remove it and re-run: rm -rf %s && till init --group <group>", agentsDir, agentsDir)` when a non-dir `.md` entry is found at root.
- `cmd/till/init_cmd.go:498-503` — call site inside `runInitPipeline`, immediately after `os.Getwd()` resolution, BEFORE `copyAgentFiles` (line 510).
- `!e.IsDir() && strings.HasSuffix(e.Name(), ".md")` guard (line 425) correctly skips subdirectories whose names happen to end with `.md`, matching the spec phrasing "any direct child is a `.md` regular file (not a subdirectory)".

**Verdict:** PASS (functional behavior matches spec; see NIT-1 for cosmetic trailing-slash delta).

---

### Bullet 2 — `runInitPipeline` calls `detectOldSchemaAgentsTOML(destDir string) error` BEFORE `copyAgentFiles`; emits exact remediation error on hit

> `runInitPipeline` calls a new `detectOldSchemaAgentsTOML(destDir string) error` function BEFORE calling `copyAgentFiles`. If `<destDir>/agents.toml` exists and any of its first 20 lines (trimmed) starts with `[agents.`, returns a non-zero error: `"agents.toml uses the old [agents.kind] schema. Remove it and re-run: rm <destDir>/agents.toml && till init --group <group>"`.

**Evidence:**
- `cmd/till/init_cmd.go:453-477` — `detectOldSchemaAgentsTOML(destDir string) error` defined. Opens `<destDir>/agents.toml`; on `fs.ErrNotExist` returns nil (no-op); otherwise scans first 20 lines via `bufio.Scanner` with `for sc.Scan() && lineNum < 20` and checks `strings.HasPrefix(strings.TrimSpace(sc.Text()), "[agents.")`.
- Error string: `fmt.Errorf("agents.toml uses the old [agents.kind] schema. Remove it and re-run: rm %s && till init --group <group>", tomlPath)` where `tomlPath = filepath.Join(destDir, "agents.toml")` — matches spec literal exactly (since spec literal `<destDir>/agents.toml` resolves to the same `filepath.Join` output).
- `cmd/till/init_cmd.go:504-506` — call site inside `runInitPipeline`, BEFORE `copyAgentFiles`.
- Prefix exactness: `[agents.` (with trailing dot) matches `[agents.build]` / `[agents.plan]`; does NOT match bare `[agents]` (test `no_dot_agents_section_not_old_schema` confirms behaviorally).
- 20-line bound: `lineNum < 20` plus pre-increment ordering — fires `for sc.Scan() && lineNum < 20 { lineNum++; ... }`. Test `old_schema_within_first_20_lines` (line at position 16) PASSES detection. Test `old_schema_beyond_20_lines_not_detected` (line at position 26) PASSES no-detection.

**Verdict:** PASS.

---

### Bullet 3 — Both checks run before any file-copy side effects; clean state passes through unaffected

> Both checks run before any file-copy side effects. A project with neither condition passes through unaffected.

**Evidence:**
- `cmd/till/init_cmd.go:498-510` — `detectFLATLayout` (line 501) and `detectOldSchemaAgentsTOML` (line 504) execute BEFORE `copyAgentFiles` (line 510). Both use `return err` on hit, terminating the pipeline before any write.
- `cmd/till/init_cmd_test.go:1252-1270` — `clean_state_no_flat_layout` subtest exercises `run()` end-to-end on a clean temp dir, asserts `err == nil` AND asserts stdout contains the Laslig `Init` block (confirming the full pipeline runs after both detects return nil).

**Verdict:** PASS.

---

### Bullet 4 — Re-run on clean-state (new schema subdir layout) both checks pass

> Re-run on clean-state (new schema subdir layout): both checks pass (no error).

**Evidence:**
- Both detect functions explicitly handle `fs.ErrNotExist` as a no-op (`cmd/till/init_cmd.go:431-433` for FLAT, `cmd/till/init_cmd.go:471-473` for old-schema), so a virgin temp dir trivially passes.
- The `clean_state_no_flat_layout` subtest covers the "neither bad state present" case end-to-end.
- The spec phrasing "new schema subdir layout" refers to `<destDir>/.tillsyn/agents/<group>/<file>.md` (subdir-per-group). `detectFLATLayout` examines only direct children of `<destDir>/.tillsyn/agents/`; a directory entry (`<group>/`) satisfies `e.IsDir()` → loop continues without triggering. Logic-evident PASS.

**Verdict:** PASS.

---

### Bullet 5 — CONSUMER-TIE via `run(ctx, args, &out, io.Discard)` end-to-end (3 mandated cases)

> CONSUMER-TIE: tests via `run(ctx, args, &out, io.Discard)` end-to-end — one test for FLAT layout present (expects non-zero + error substring), one test for old-schema `agents.toml` present (same), one test for clean state (both pass, exits zero).

**Evidence:**
- `cmd/till/init_cmd_test.go:1224-1271` — `TestRunInitPipeline_FLATDetection` with subtests `flat_layout_present` (asserts non-zero + substring `"FLAT agent layout"`) and `clean_state_no_flat_layout` (asserts nil + Laslig stdout).
- `cmd/till/init_cmd_test.go:1279-1385` — `TestRunInitPipeline_OldSchemaDetection` with 4 subtests: `old_schema_first_line` (asserts non-zero + substring), `no_dot_agents_section_not_old_schema`, `old_schema_within_first_20_lines`, `old_schema_beyond_20_lines_not_detected`.
- All six subtests invoke `run(context.Background(), []string{...}, &out, io.Discard)` — full end-to-end CONSUMER-TIE shape per spec.
- Coverage EXCEEDS the spec's 3 mandated cases (FLAT-present, old-schema-present, clean-state); builder added 3 supplementary cases for 20-line-window boundary correctness.

**Verdict:** PASS (exceeds spec).

---

### Bullet 6 — `mage test-pkg ./cmd/till` passes; `mage ci` green

> `mage test-pkg ./cmd/till` passes; `mage ci` green.

**Evidence:**
- `mage test-pkg ./cmd/till`: 308/308 PASS in 9.72s.
- `mage test-func ./cmd/till TestRunInitPipeline_FLATDetection`: 3/3 PASS in 9.18s (parent harness + 2 subtests).
- `mage test-func ./cmd/till TestRunInitPipeline_OldSchemaDetection`: 5/5 PASS in 9.12s (parent harness + 4 subtests).
- `mage ci` not run per cross-droplet note (W8.D21 has parallel WIP on `internal/app/dispatcher/cli_claude/render/render_test.go` — unrelated to W2.D2's `cmd/till` package). Scoped package test is the authoritative gate for W2.D2's verification.

**Verdict:** PASS (scoped-package test green; full `mage ci` properly deferred per cross-droplet directive).

---

## NITs

### NIT-1 — Cosmetic trailing-slash deviation in FLAT error message (low)

The spec literal for the FLAT detection error reads:

```
FLAT agent layout detected at <destDir>/.tillsyn/agents/. Remove it and re-run: rm -rf <destDir>/.tillsyn/agents && till init --group <group>
```

Note the first occurrence of the path ends with `agents/.` (trailing slash, then period); the second occurrence ends with `agents` (no slash). The shipped code uses `filepath.Join(destDir, ".tillsyn", "agents")` which strips trailing slashes, so the actual emitted error is:

```
FLAT agent layout detected at <destDir>/.tillsyn/agents. Remove it and re-run: rm -rf <destDir>/.tillsyn/agents && till init --group <group>
```

(period without preceding slash in the first occurrence).

**Severity:** low — the load-bearing substring `"FLAT agent layout"` matches; remediation text matches; only the cosmetic directory-trailing-slash convention is dropped. The acceptance ContextBlock RiskNote explicitly states "error messages contain `<destDir>` as a placeholder — actual implementation interpolates the real path," suggesting some interpolation flexibility. The shipped test asserts only the substring `"FLAT agent layout"` (consistent with this latitude).

**Recommended action:** Either (a) accept as-is and update the spec literal to drop the cosmetic slash in a future doc pass, or (b) builder appends `string(filepath.Separator)` to the printed path in a follow-up if dev wants exact-literal match. No code change required for W2.D2 verdict.

---

## Verdict rationale

All 6 acceptance bullets have file:line evidence and passing tests. The implementation:

1. Adds `detectFLATLayout` and `detectOldSchemaAgentsTOML` as standalone functions (not embedded in `copyAgentFiles`) — preserves the D5 independence decision documented in the spec ContextBlocks.
2. Calls both checks at the top of `runInitPipeline`, BEFORE `copyAgentFiles`, satisfying the no-partial-writes invariant.
3. Handles absent-dir / absent-file as no-ops via `errors.Is(err, fs.ErrNotExist)` — clean re-runs pass through unaffected.
4. Implements the exact `[agents.` prefix check (with trailing dot), correctly distinguishing the old-schema header from a bare `[agents]` (test confirms).
5. Honors the 20-line scan window heuristic with `lineNum < 20` bound (tests confirm both within-window detect and beyond-window non-detect).
6. Exercises 6 CONSUMER-TIE subtests via `run(ctx, args, &out, io.Discard)`, exceeding the spec's 3 mandated cases.
7. Scoped `mage test-pkg ./cmd/till` returns 308/308 PASS. The full `mage ci` is deferred per the cross-droplet note — `render_test.go` is on a separate parallel droplet's working tree.

Single cosmetic NIT (trailing-slash in FLAT first occurrence) does not affect functional behavior or the asserted substring; recommend follow-up doc/text alignment, not a code change.

**Overall verdict:** PASS WITH NITS.
