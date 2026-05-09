# DROP_4c.6.W0_AGENTS_TOML_SCHEMA — Builder Worklog

Append `## Droplet N.M — Round K` per build attempt. Per `workflow/example/drops/WORKFLOW.md` § "File Lifecycle", this file is durable; never `git rm`d.

## Droplet 4c.6.W0.D1 — Round 1

### Files touched

- `internal/config/agents.go` (NEW; ~165 LOC including doc comments)
- `internal/config/agents_test.go` (NEW; ~150 LOC including 5 tests + 1 helper)
- `internal/config/testdata/agents/baseline.toml` (NEW; canonical golden fixture)
- `internal/config/testdata/agents/malformed.toml` (NEW; unterminated string for *toml.DecodeError assertion)
- `internal/config/testdata/agents/unknown_field.toml` (NEW; strict-decode rejection assertion)
- `internal/config/testdata/agents/preset_only.toml` (NEW; absent-blocks-nil-safe assertion)
- `workflow/drop_4c_6/DROP_4c.6.W0_AGENTS_TOML_SCHEMA/PLAN.md` (state-flip todo→in_progress→done on D1)

### Build-tool targets run

- `mage test-func ./internal/config TestLoadRegistry_Baseline` — RED (build error: undefined `LoadRegistry`/`Preset`/etc., as expected pre-implementation), then GREEN after `agents.go` landed.
- `mage test-func ./internal/config TestLoadRegistry_MalformedTOML` — GREEN.
- `mage test-func ./internal/config TestLoadRegistry_UnknownTopLevelField` — GREEN.
- `mage test-func ./internal/config TestLoadRegistry_FileNotFound` — GREEN.
- `mage test-func ./internal/config TestLoadRegistry_AbsentBlocksNilSafe` — GREEN.
- `mage test-func ./internal/config "TestLoadRegistry_.*"` — 5/5 GREEN.
- `mage test-pkg ./internal/config` (Phase 5 step 1 package gate) — 37/37 GREEN (32 pre-existing + 5 new).
- `mage format` — clean.

### Design notes

1. **`Kind` reuse, not redefinition.** `internal/domain.Kind` already exports the closed 12-value enum (`KindPlan`, `KindBuild`, …). `internal/config` imports `internal/domain` directly — confirmed no reverse import (`git grep` for `tillsyn/internal/config` under `internal/domain/` returns empty). The PLAN.md `RiskNote` about "TODO consolidation" does not apply: the consolidation is already in place because `internal/domain` is the canonical home and `internal/config` consumes it.

2. **Strict decode via `toml.NewDecoder(r).DisallowUnknownFields()`.** Per `go doc github.com/pelletier/go-toml/v2.Decoder` confirmation, this is the lib's documented strict mode. `toml.Unmarshal` (used elsewhere in `config.go`) does NOT support strict mode; a fresh `*toml.Decoder` is required. Used `bytes.NewReader(content)` for the io.Reader (idiomatic, single import line).

3. **Per-kind blocks as typed fields, not `map[string]Override`.** Each of the 12 kinds gets its own typed pointer field on `agentsTOMLBlock` so `DisallowUnknownFields()` rejects typos like `[agents.bulid]` at decode time. `map[string]Override` would silently drop unknown keys, which is a serious user-experience regression for a config file users hand-edit. Cost: adding a 13th kind requires touching this struct in addition to `internal/domain/kind.go` — documented in the doc-comment.

4. **`Override` uses `*T` pointers across the board** to discriminate "absent" from "explicit zero value." Map fields are `*map[string]string` (nil pointer = inherit, non-nil empty map = explicit drop), list fields are `*[]string` (nil = inherit, non-nil empty = explicit drop). Same pattern already used at `internal/app/dispatcher/binding_resolved.go:44` (`BindingOverrides`) — this codebase is comfortable with the idiom.

5. **`addOverride` helper threads `*Override → map[Kind]Override`.** Absent blocks (nil pointer in the decoded struct) do NOT appear as keys in the resulting map; D2's `Resolve` distinguishes "key absent" from "key present with zero Override" (the latter is currently impossible from TOML decode alone, but D3's `MergeLocal` may construct such an `Override` in code).

6. **`AgentRuntime` ships in D1 even though D2 produces it.** Per Acceptance bullet "AgentRuntime (effective per-kind merged result)", the type lives in `agents.go` from D1 onward so D2 is a pure-function add (`Resolve`) without re-introducing the type. Field set mirrors `Preset` exactly.

7. **Doc comments on every exported type and the one exported function.** Match style of existing `internal/config/config.go` (which uses doc comments per the project Go discipline).

### Decisions deferred to later droplets

- **D2 `Resolve`** — merge engine. Stub not landed; no `Resolve` symbol in agents.go yet.
- **D3 `MergeLocal` + `ErrToolsDenyNotOverridable`** — local-file deep-merge.
- **D5 `*ConfigError` envelope** — D1 currently returns raw `*toml.DecodeError` wrapped via `fmt.Errorf %w`. The `errors.As(err, &decodeErr)` test assertion confirms the chain works. D5 will revise `TestLoadRegistry_MalformedTOML` to also assert `*ConfigError` wrapping; pre-D5 the test asserts the simpler `*toml.DecodeError` shape.

### State flip

- `PLAN.md` → Droplet 4c.6.W0.D1 `**State:**` `todo` → `in_progress` (at start of round) → `done` (at end of round).

### Hylla Feedback

- **Query**: `hylla_search_keyword query="Kind enum domain action_item"`, then `hylla_search_keyword query="Kind plan build"`.
- **Missed because**: enrichment still running for `github.com/evanmschultz/tillsyn@main` (post-recent-commits state). Hylla returned `enrichment still running` for both attempts.
- **Worked via**: `Read /Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/domain/kind.go` directly. Located the closed 12-value enum + constants in 1 read.
- **Suggestion**: when enrichment is running, Hylla could return the previous-snapshot results with a freshness warning rather than refusing the query entirely. Falling back to `Read` works but loses the surrounding-context discovery Hylla normally provides.
