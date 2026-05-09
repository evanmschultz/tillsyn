# DROP_4c.6.W0_AGENTS_TOML_SCHEMA — Builder QA Proof

Append `## Droplet N.M — Round K` per QA pass. Per `workflow/example/drops/WORKFLOW.md` § "File Lifecycle", this file is durable; never `git rm`d.

## Droplet 4c.6.W0.D1 — Round 1

### Findings

(none — see Summary)

### Missing Evidence

(none — see Summary)

### Summary

**Verdict: pass** — 0 findings.

Build-QA-Proof axes evaluated:

1. **Diff-vs-spec.** Commit `4f14547` ("feat(config): w0.d1 agents.toml schema and decode") touches exactly the declared paths from PLAN.md droplet D1: `internal/config/agents.go` (NEW, 195 LOC), `internal/config/agents_test.go` (NEW, 184 LOC), `internal/config/testdata/agents/baseline.toml` (NEW), `internal/config/testdata/agents/malformed.toml` (NEW), `internal/config/testdata/agents/preset_only.toml` (NEW), `internal/config/testdata/agents/unknown_field.toml` (NEW), plus the expected `BUILDER_WORKLOG.md` round entry and one-line state-flip on `PLAN.md`. No drive-by edits. The two extra fixtures (`malformed.toml`, `unknown_field.toml`, `preset_only.toml`) beyond the PLAN.md `Paths` line item ("`baseline.toml` (NEW — golden fixture exercising `[agents]` defaults + one per-kind block)") are spec-justified — `TestLoadRegistry_MalformedTOML` and `TestLoadRegistry_AbsentBlocksNilSafe` are explicit acceptance bullets and need fixtures; `TestLoadRegistry_UnknownTopLevelField` is a load-bearing strict-decode proof.

2. **AcceptanceCriteria coverage.** Each PLAN.md L2 acceptance bullet has a verifying test:
   - "`Preset` (the `[agents]` defaults block — fields `Client`, `Model`, `Effort`, `MaxTries`, `MaxBudgetUSD`, `MaxTurns`, `BlockedRetries`, `BlockedRetryCooldown`, `AutoPush`, `EnvSet`, `EnvFromShell`, `CliArgs`, `ToolsAllow`, `ToolsDeny`, `ClaudeMDAddons`)" → all 15 fields present at `internal/config/agents.go:35-51`; `TestLoadRegistry_Baseline` (`agents_test.go:34-84`) asserts every field's value against `baseline.toml`. Field set matches §4.1 schema exactly.
   - "`Override` (partial-shape; every field is a `*T` pointer)" → all 15 fields are pointers at `agents.go:62-78`; pointer-vs-zero distinction verified by `TestLoadRegistry_Baseline:90-99` which asserts `override.ToolsAllow != nil` AND `override.Model == nil` from a single fixture.
   - "`AgentRuntime` (effective per-kind merged result, same fields as `Preset`)" → present at `agents.go:84-100`, field set mirrors `Preset` 1-1.
   - "`AgentsRegistry` (loaded `agents.toml` — holds `Preset` + `map[Kind]Override`)" → present at `agents.go:106-109`.
   - "`LoadRegistry(path string) (*AgentsRegistry, error)` reads a TOML file via `pelletier/go-toml/v2`" → present at `agents.go:153-184`; uses `toml.NewDecoder(bytes.NewReader(content)).DisallowUnknownFields()`. The PLAN.md acceptance offers `toml.Unmarshal` "or equivalent — verify exact API via `go doc github.com/pelletier/go-toml/v2.Decoder`"; worklog Design Note 2 confirms `toml.Unmarshal` doesn't support strict mode so Decoder API is the lib-correct equivalent.
   - "Decoder preserves line numbers in error reporting via `*toml.DecodeError`" → `TestLoadRegistry_MalformedTOML` (`agents_test.go:106-122`) asserts `errors.As(err, &decodeErr)` succeeds AND `decodeErr.Position()` returns row > 0.
   - "Map fields decode as `map[string]string`; nil-safe for absent blocks" → `Preset.EnvSet` and `Preset.EnvFromShell` are bare `map[string]string` (`agents.go:45-46`); `TestLoadRegistry_AbsentBlocksNilSafe` (`agents_test.go:158-171`) confirms `Overrides` map is non-nil and absent kinds yield `(zero, false)` from map lookup, never panic.
   - "`Kind` is the closed-12-enum string type" → `internal/domain.Kind` reused (worklog Design Note 1 confirms `internal/config` imports `internal/domain` directly with no reverse import; `agents.go:24` imports it). All 12 kind values handled in `LoadRegistry` via `addOverride` calls on `agents.go:167-178`.

3. **Constraint preservation.** `internal/config/config.go` last touched in Drop 2 (`fdcaa65`); D1 did NOT modify it (`git log -- internal/config/config.go` shows no `4f14547` entry). No competing TOML libs introduced — `git grep BurntSushi/toml` and `git grep naoina/toml` return empty; the only TOML lib in `go.mod` remains `github.com/pelletier/go-toml/v2 v2.2.4` (`go.mod:34`), already in use at `internal/config/config.go:13`.

4. **Spec-conformance.** TOML decode preserves line numbers via `*toml.DecodeError` — `TestLoadRegistry_MalformedTOML` exercises `errors.As(err, &decodeErr)` and `decodeErr.Position()` against `malformed.toml` (line 6 unterminated string). The strict-decode rejection path is additionally covered by `TestLoadRegistry_UnknownTopLevelField` against `unknown_field.toml`. Schema field set matches §4.1 exactly: `client / model / effort / max_tries / max_budget_usd / max_turns / blocked_retries / blocked_retry_cooldown / auto_push / env_set / env_from_shell / cli_args / tools_allow / tools_deny / claude_md_addons` — 15 fields, present in both `Preset` and `Override`. Per-kind blocks decoded as typed pointer fields (one per kind in the closed-12-enum) at `agents.go:127-141`; this trades extensibility (adding a 13th kind requires editing this struct) for strict-decode rejection of typo'd kind names like `[agents.bulid]` — explicitly a deliberate design trade documented in the doc-comment at `agents.go:120-126` AND in worklog Design Note 3.

5. **Shipped-but-not-wired.** `LoadRegistry` is the proper consumer for the schema in this droplet's vertical slice (the load+decode proof). `AgentRuntime` is shipped without a producer in D1 — but PLAN.md L2 acceptance ("`AgentRuntime` (effective per-kind merged result, same fields as `Preset` but resolved)") and worklog Design Note 6 explicitly justify this: `AgentRuntime` lives in `agents.go` from D1 onward so D2's `Resolve` is a pure-function add. This is documented planner intent, not orphaned code. `Override` is consumed by the test (asserts pointer-vs-zero distinction); `Preset` is consumed by the test and stored on `AgentsRegistry`; `AgentsRegistry` is the return shape of `LoadRegistry`. No fixture is shipped without a consuming test (all four fixtures map to specific tests).

**Gate evidence:**

- `mage test-pkg ./internal/config` — 37/37 GREEN (32 pre-existing + 5 new). Builder report matches.
- `mage test-func ./internal/config TestLoadRegistry_Baseline` — 1/1 GREEN.
- `mage test-func ./internal/config TestLoadRegistry_MalformedTOML` — 1/1 GREEN.
- `mage test-func ./internal/config TestLoadRegistry_UnknownTopLevelField` — 1/1 GREEN.
- `mage test-func ./internal/config TestLoadRegistry_FileNotFound` — 1/1 GREEN.
- `mage test-func ./internal/config TestLoadRegistry_AbsentBlocksNilSafe` — 1/1 GREEN.
- `git grep "pelletier/go-toml/v2"` — single TOML lib confirmed; `git grep "BurntSushi/toml"` and `git grep "naoina/toml"` empty.

**Proof certificate:**

- **Premises** — D1 ships the 4 declared types (`Preset` / `Override` / `AgentRuntime` / `AgentsRegistry`) per §4.1; `LoadRegistry` decodes via `pelletier/go-toml/v2` in strict mode with position-aware errors; pointer-vs-zero discrimination works on `Override`; absent-blocks are nil-safe; `internal/domain.Kind` is reused (no local enum drift).
- **Evidence** — see per-axis citations above.
- **Trace or cases** — read PLAN.md acceptance bullets; cross-checked each against `agents.go` symbols + `agents_test.go` test names + assertions; ran 5 tests + package gate; verified diff scope via `git show --stat 4f14547`; verified TOML-lib non-competition via `git grep`.
- **Conclusion** — PASS. All five Build-QA-Proof axes satisfied; no findings; no missing evidence.
- **Unknowns** — none for D1 scope. D2/D3/D5 will revise some D1 assertions (`TestLoadRegistry_MalformedTOML` upgraded to expect `*ConfigError` envelope shape per D5); that revision is explicitly planned and out of scope for D1's own verdict.

### Hylla Feedback

N/A — verification touched only Go files, but all evidence-gathering used `Read` / `git log` / `git grep` / `mage` (worklog already reported the only Hylla-fallback case for D1). Hylla was not queried during this proof pass because every needed signal (file diff, commit metadata, test results) is post-commit pre-ingest territory where `git` + `mage` are authoritative.
