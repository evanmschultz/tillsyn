# W2.D6 — BUILD-QA-PROOF Verdict

**Date:** 2026-05-13
**Reviewer:** go-qa-proof-agent (opus)
**Overall verdict:** PASS

## Acceptance Bullet Coverage

| # | Acceptance Bullet (PLAN.md L289-306) | Evidence | Verdict |
|---|---|---|---|
| 1 | `runInitPipeline` calls `writeTemplateTOML(destDir, groups, homeDir) (int, int, error)` after `copyAgentFiles` succeeds | init_cmd.go:507-516 — call sequence: copyAgentFiles (507) → writeTemplateTOML (513) → copyAgentsTOML (518) | PASS |
| 2 | Per-group HOME tier source = `filepath.Join(homeDir, ".tillsyn", "templates", group+".toml")` | init_cmd.go:724 — `homePath := filepath.Join(homeDir, ".tillsyn", "templates", group+".toml")` | PASS |
| 3 | Embedded fallback from `builtin/till-<group>.toml` | init_cmd.go:733-738 — `path.Join("builtin", "till-"+group+".toml")` against `templates.DefaultTemplateFS`; fallback triggered when `os.ReadFile(homePath)` returns `fs.ErrNotExist` | PASS |
| 4 | Aggregated TOML content written to `<destDir>/.tillsyn/template.toml` | init_cmd.go:667 (target) + 714 (`fsatomic.WriteFile(target, []byte(buf.String()), ...)`) | PASS |
| 5 | Blanket skip if `<destDir>/.tillsyn/template.toml` exists (no overwrite) | init_cmd.go:670-685 — `os.ReadFile(target)` success path returns `(0, 1, nil)` without writing | PASS |
| 6 | Partial-state warning printed when existing file is missing `[<group>]` section for selected groups; non-fatal | init_cmd.go:672-684 — iterates groups, builds `missing` slice via `strings.Contains(content, "["+group+"]")` OR `strings.Contains(content, "["+group+".")`; if `len(missing) > 0` prints WARN to stdout and returns nil error | PASS |
| 7 | Partial-state check uses string presence (NOT full TOML parse) | init_cmd.go:676 — `strings.Contains` only; no `toml.Unmarshal`. Documented in doc-comment lines 654-658 with the comment-suppression tradeoff accepted per spec | PASS |
| 8 | `homeDir` derived from `os.UserHomeDir()` unless `rootOpts.homeDir` non-empty | init_cmd.go:497-503 — `homeDir := strings.TrimSpace(opts.homeDir); if homeDir == "" { homeDir, err = os.UserHomeDir() ... }`. Wired into pipeline at 513 | PASS |
| 9 | Laslig summary row added: `"template.toml"` → `"added"` or `"skipped (already exists)"` | init_cmd.go:553-557 (status string derivation) + 565 (`{"template.toml", templateTOMLStatus}` row in `writeCLIKV`) | PASS |
| 10 | CONSUMER-TIE (a) HOME tier present | init_cmd_test.go:1905-1946 `TestWriteTemplateTOML_HOMETierPresent` — drives `run(...)`, seeds `<HOME>/.tillsyn/templates/go.toml` with custom content, asserts `[go]` header + `home-tier-value` present in output + Laslig row present | PASS |
| 11 | CONSUMER-TIE (b) HOME tier absent → embedded fallback | init_cmd_test.go:1951-1977 `TestWriteTemplateTOML_HOMETierAbsent` — no HOME seed; asserts `[go]` header + len >= 100 bytes (substantial embedded content) | PASS |
| 12 | CONSUMER-TIE (c) idempotent re-run | init_cmd_test.go:1983-2019 `TestWriteTemplateTOML_Idempotent` — first run creates file, second run zero-error; first/second `os.ReadFile` bytes equal; stdout contains "skipped" | PASS |
| 13 | CONSUMER-TIE (d) partial-state warning | init_cmd_test.go:2026-2065 `TestWriteTemplateTOML_PartialStateWarning` — pre-creates template.toml with `[gen]` only, runs with `groups=["go"]`, asserts `run()` returns nil + stdout contains "WARN" + mentions "go" + file unchanged | PASS |
| 14 | `mage test-pkg ./cmd/till` passes; `mage ci` green | Builder reported 333/333 cmd/till + 3274/3274 mage ci PASS; reviewer re-ran `mage test-pkg ./cmd/till` → 333/333 PASS confirmed | PASS |
| 15 | `readTemplateForGroup` helper added (HOME-then-embedded) | init_cmd.go:723-739 — signature `func readTemplateForGroup(homeDir, group string) ([]byte, error)`; HOME-first read; embedded fallback on `fs.ErrNotExist`; non-NotExist errors wrapped | PASS |
| 16 | Atomic write via `fsatomic.WriteFile` (no torn writes) | init_cmd.go:714 — `fsatomic.WriteFile(target, []byte(buf.String()), agentFileInitPerm)` | PASS |
| 17 | `.tillsyn` directory created before write | init_cmd.go:710-712 — `os.MkdirAll(filepath.Dir(target), 0o755)` before `fsatomic.WriteFile` | PASS |

All 17 enumerated acceptance points: PASS.

## NITs

None. Implementation matches spec verbatim, including the explicitly-accepted tradeoff that a `[<group>]` substring appearing inside a TOML comment line would suppress the partial-state warning (init_cmd.go:654-658 doc-comment documents this per W2.D6 RiskNotes line 326).

Minor observations (NOT findings, not actionable):

- The `--home` flag override path (line 497 `opts.homeDir` non-empty branch) is exercised indirectly via `t.Setenv("HOME", ...)` which routes through `os.UserHomeDir()` (the fallback branch). No explicit `--home tmp` flag test exists for D6, but the spec phrasing ("override for test isolation") is satisfied by env-based isolation. Not a finding.
- Lines 699-706 prepend `[<group>]\n` header only when content does not start with that header. A file like `# comment\n[go]\n...` would NOT match `strings.HasPrefix(content, "[go]")` and a duplicate `[go]\n` would be emitted. Spec is silent; test fixtures use embedded files that start with `[<group>]` so this path is not exercised. Not a finding for D6; could become a NIT after the schema files in W4.D2 ship.

## Verdict rationale

Every acceptance bullet in PLAN.md L289-306 maps to file:line evidence in `cmd/till/init_cmd.go` and `cmd/till/init_cmd_test.go`. All four CONSUMER-TIE sub-tests drive `run(ctx, args, &out, io.Discard)` end-to-end through the full `runInitJSON` → `runInitPipeline` → `writeTemplateTOML` chain. Builder reported and reviewer confirmed `mage test-pkg ./cmd/till` 333/333 PASS. Falsification pass produced no unmitigated counterexample (nine attacks mitigated or explicitly accepted per spec). Verdict: PASS.
