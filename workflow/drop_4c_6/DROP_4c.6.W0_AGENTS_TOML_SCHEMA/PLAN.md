# DROP_4c.6.W0 — AGENTS_TOML_SCHEMA

**State:** building
**Blocked by:** —
**Paths (expected):** `internal/config/agents.go` (NEW), `internal/config/agents_test.go` (NEW), `internal/config/frontmatter.go` (NEW), `internal/config/frontmatter_test.go` (NEW), `internal/config/testdata/agents/*.toml` (NEW golden fixtures, one per merge edge case)
**Packages (expected):** `internal/config`
**PLAN.md ref:** `workflow/drop_4c_6/PLAN.md` → 4c.6.W0 sub-plan container row (lines 51-67)
**Workflow:** `workflow/example/drops/WORKFLOW.md`
**Cascade concept:** `AGENT_CASCADE_DESIGN.md`
**Started:** 2026-05-09
**Closed:** —

## Scope

Land `internal/config/agents.go` with the `[agents]` defaults struct + per-kind override merge per `SKETCH.md` § 4 + § 5 + § 26.W0. New types: `AgentRuntime` (effective per-kind config), `AgentsRegistry` (loaded `agents.toml`), `Preset` (the `[agents]` block), per-kind `Override` partial-shape struct. Field-level inheritance: per-kind block overrides only fields it sets; absent fields fall through to `[agents]`. Map fields (`env_set` / `env_from_shell`) merge per-key. List fields (`cli_args` / `tools_allow` / `tools_deny` / `claude_md_addons`) full-replace. `agents.local.toml` deep-merge over the resolved `agents.toml`; `tools_deny` user-override REJECTED with a closed sentinel error citing the TOML line. TOML position-tracking errors via `pelletier/go-toml/v2` (existing dep at `go.mod:30 github.com/pelletier/go-toml/v2`; do NOT add a competing TOML lib per `SKETCH.md` § 26.W0 ContextBlocks — confirmed in use at `internal/config/config.go:13`). Render-time frontmatter `model:` / `tools:` strip helper exposed for W3 to call into (lives in a sibling `frontmatter.go` to keep agents.go focused on schema + merge).

This sub-drop is the runtime-config foundation: every other Drop 4c.6 wave that consumes runtime config (W2 `till init` shipping `agents.example.toml`, W3 frontmatter-strip wiring, W5 template-thinning's `tools` migration) reads the types defined here. Per `SKETCH.md` v2.8.4 § 26.W0, plan-QA verdict on the wave-level Specify is locked; this PLAN.md translates the wave-Specify into atomic droplets, NOT a re-validation of the sketch.

## Planner

### Decomposition shape — five atomic droplets serialized on `internal/config` package compile

Every droplet below either edits `internal/config/agents.go` (D1, D2, D3, D5) or adds a new file inside the same Go package `internal/config` (D4 → `frontmatter.go`). Per `~/.claude/agents/go-planning-agent.md` § "Cascade Design — Atomic Droplets + Parallelization (HARD RULES)" — "Siblings sharing a `packages` entry (even with disjoint paths within that package) → `blocked_by`. Go-package compilation is shared." — all five droplets MUST serialize via `blocked_by` because they share the `internal/config` package compile/test unit. The chain is therefore strict:

```
W0.D1 ──▶ W0.D2 ──▶ W0.D3 ──▶ W0.D4 ──▶ W0.D5
```

D4 (frontmatter strip helper) is file-disjoint from D1-D3-D5 (`frontmatter.go` vs `agents.go`) but package-locked, so it sits in the chain. Placing D4 fourth (between D3 and D5) is arbitrary on file-grounds but keeps the semantics-first ordering: schema (D1) → resolve (D2) → local-merge (D3) → frontmatter helper (D4, no schema-dep) → error envelope (D5, wraps decode + merge errors from D1-D3).

Tests are co-located with each droplet's production file. D1 ships `agents_test.go`; D2/D3/D5 append to `agents_test.go`; D4 ships `frontmatter_test.go`. Golden TOML fixtures for merge edge cases live in `internal/config/testdata/agents/`.

### Droplet 4c.6.W0.D1 — `Preset` + per-kind `Override` structs + `AgentRuntime` + `AgentsRegistry` + TOML decode

- **State:** done
- **Kind:** `build` (atomic droplet; `Irreducible: true`)
- **Paths:** `internal/config/agents.go` (NEW), `internal/config/agents_test.go` (NEW), `internal/config/testdata/agents/baseline.toml` (NEW — golden fixture exercising `[agents]` defaults + one per-kind block).
- **Packages:** `internal/config`.
- **Acceptance:**
  - `internal/config/agents.go` defines four exported types matching `SKETCH.md` § 4.1 + § 4.2: `Preset` (the `[agents]` defaults block — fields `Client`, `Model`, `Effort`, `MaxTries`, `MaxBudgetUSD`, `MaxTurns`, `BlockedRetries`, `BlockedRetryCooldown`, `AutoPush`, `EnvSet map[string]string`, `EnvFromShell map[string]string`, `CliArgs []string`, `ToolsAllow []string`, `ToolsDeny []string`, `ClaudeMDAddons []string`); `Override` (partial-shape; every field is a `*T` pointer to distinguish "absent" from "zero value"); `AgentRuntime` (effective per-kind merged result, same fields as `Preset` but resolved); `AgentsRegistry` (loaded `agents.toml` — holds `Preset` + `map[Kind]Override`).
  - `LoadRegistry(path string) (*AgentsRegistry, error)` reads a TOML file via `pelletier/go-toml/v2` with `toml.Unmarshal` against a strict-decode-options builder (`toml.DecodeOptions{DisallowUnknownFields: true}` or equivalent — verify exact API via `go doc github.com/pelletier/go-toml/v2.Decoder` before authoring); returns sentinel error with TOML line number on malformed input.
  - Decoder preserves line numbers in error reporting via `pelletier/go-toml/v2`'s `*toml.DecodeError` (the lib's documented position-aware error type — verify via Context7 / `go doc` before authoring).
  - `Kind` is the closed-12-enum string type per `CLAUDE.md` § Cascade Tree Structure (planner consumer reads `metadata.role`-distinct kinds: `plan` / `build` / `plan-qa-proof` / `plan-qa-falsification` / `build-qa-proof` / `build-qa-falsification` / `research` / `closeout` / `commit` / `refinement` / `discussion` / `human-verify`); reuse the existing kind constant set if `internal/domain` exposes it (verify via Hylla / LSP — if absent, define a local enum in agents.go and TODO-comment a future consolidation).
  - Map fields (`EnvSet`, `EnvFromShell`) decode as `map[string]string`; nil-safe for absent blocks.
  - Test `TestLoadRegistry_Baseline` loads `testdata/agents/baseline.toml` (containing the §4.1 example `[agents]` block + a single `[agents.build]` override); asserts every `Preset` field decoded with the expected value AND the `Override.ToolsAllow` pointer is non-nil with the expected slice.
  - Test `TestLoadRegistry_MalformedTOML` feeds a fixture with a syntax error; asserts the returned error is a typed `*toml.DecodeError` (or wrapped via `errors.As`) AND the error message includes the TOML line number.
  - `mage test-pkg ./internal/config` passes; `mage test-func ./internal/config TestLoadRegistry_Baseline` and `TestLoadRegistry_MalformedTOML` pass individually.
- **Blocked by:** —
- **Specify (droplet-scope; inherits `SKETCH.md` § 26.W0):**
  - **Objective:** Define the schema types + TOML decode wiring for `agents.toml` so D2 (resolve), D3 (local merge), and D5 (error envelope) have concrete types to work over. Single file (`agents.go`) + co-located tests + one golden fixture. The decode path proves `pelletier/go-toml/v2` line-number preservation works for the schema we're shipping; without that proof, D5's error envelope has no line numbers to wrap.
  - **AcceptanceCriteria:** see Acceptance bullets above.
  - **ValidationPlan:** `mage test-pkg ./internal/config`; `mage test-func ./internal/config TestLoadRegistry_Baseline`; `mage test-func ./internal/config TestLoadRegistry_MalformedTOML`; `mage ci` green.
  - **RiskNotes:**
    - `pelletier/go-toml/v2` line-number-preservation API has changed across versions; the dep is already in `go.mod` at the project's pinned version — verify the actual `*toml.DecodeError` shape via `go doc` before writing assertions. If the lib emits position via a different type, adapt the test to the live API.
    - `Override` partial-shape via `*T` pointers vs. sentinel zero values: chose pointer-based per Go idiom for "absent vs zero distinction"; alternative (sentinel zero values) breaks the "absent fields fall through to `[agents]` defaults" semantics required by §4.2.1. Document the choice in a doc-comment on `Override`.
    - `Kind` enum reuse vs local copy: if `internal/domain` already exports the closed-12-enum, prefer reuse to avoid drift; if not exported (Hylla/LSP confirms absence), define locally and add a refinement entry. Builder verifies via `LSP` workspace-symbol query before authoring.
    - Field naming convention: TOML keys use `snake_case` (`max_budget_usd`, `env_set`); Go fields use `PascalCase` with `toml:"snake_case"` struct tags. Standard pattern; no risk beyond consistency.
  - **ContextBlocks:**
    - `reference` (normal): `pelletier/go-toml/v2` already in use at `internal/config/config.go:13` — same import is canonical; do NOT add a competing TOML lib per §26.W0 ContextBlocks.
    - `decision` (normal): `Override` uses pointer fields for absent-vs-zero discrimination per §4.2.1 inheritance contract.
    - `constraint` (high): every `Preset` field MUST be present in the `Override` shape as `*T` — D2's resolver depends on this 1-1 correspondence to merge.
    - `reference` (normal): `SKETCH.md` § 4.1 schema; § 4.2 per-kind blocks; § 4.2.1-4.2.3 inheritance rules.
    - `warning` (high): if `Kind` is defined locally rather than reused from `internal/domain`, add a `// TODO(refinement): consolidate with internal/domain Kind constants once cascade dispatcher matures` doc-comment so this isn't lost.
  - **KindPayload:** `{"changes":[{"file":"internal/config/agents.go","symbol":"Preset, Override, AgentRuntime, AgentsRegistry, Kind, LoadRegistry","action":"add","shape_hint":"new file; types per SKETCH §4.1; LoadRegistry uses pelletier/go-toml/v2 with DisallowUnknownFields strict mode; returns position-aware *toml.DecodeError"},{"file":"internal/config/agents_test.go","symbol":"TestLoadRegistry_Baseline, TestLoadRegistry_MalformedTOML","action":"add","shape_hint":"table-driven; baseline fixture exercises §4.1 example + one per-kind override; malformed fixture exercises line-number preservation via errors.As(*toml.DecodeError)"},{"file":"internal/config/testdata/agents/baseline.toml","symbol":"golden fixture","action":"add","shape_hint":"§4.1 [agents] block + [agents.build] override with tools_allow"}]}`

### Droplet 4c.6.W0.D2 — `Resolve(registry, kind)` inheritance merge engine

- **State:** done
- **Kind:** `build` (atomic droplet; `Irreducible: true`)
- **Paths:** `internal/config/agents.go` (MODIFY — append `Resolve` function + helpers), `internal/config/agents_test.go` (MODIFY — append merge-table tests), `internal/config/testdata/agents/inheritance_*.toml` (NEW — one golden fixture per merge edge case: `inheritance_full_inherit.toml`, `inheritance_partial_override.toml`, `inheritance_map_merge.toml`, `inheritance_list_replace.toml`).
- **Packages:** `internal/config`.
- **Acceptance:**
  - `Resolve(registry *AgentsRegistry, kind Kind) (AgentRuntime, error)` returns the merged effective per-kind config: starts from `registry.Preset`, applies `registry.Overrides[kind]` field-by-field, returns the resulting `AgentRuntime`.
  - Per-field semantics per `SKETCH.md` § 4.2.1 + § 4.2.3: scalar/string/numeric/bool fields → if `Override.<field>` is non-nil, use override value; else use Preset value.
  - Per-key map merge per § 4.2.2: `EnvSet` and `EnvFromShell` merge with override keys winning, default keys absent in override surviving (NOT full-replace).
  - List full-replace per § 4.2.3: `CliArgs`, `ToolsAllow`, `ToolsDeny`, `ClaudeMDAddons` — if override slice is non-nil (even if empty), use override; else use Preset.
  - Empty-list-vs-nil: `Override.ToolsDeny = &[]string{}` (explicit empty list) MUST replace Preset's non-empty list (per "absent vs zero" semantics in D1's pointer model). Test exercises this edge case.
  - `Resolve` for an absent kind (no `[agents.<kind>]` block in TOML, so `Overrides[kind]` is the zero `Override`) returns `Preset` verbatim — pure inheritance.
  - Test `TestResolve_FullInherit` loads `inheritance_full_inherit.toml` (Preset only, no per-kind block); asserts `Resolve(reg, KindBuild)` returns `Preset` field-for-field.
  - Test `TestResolve_PartialOverride` loads `inheritance_partial_override.toml` (Preset + `[agents.build]` overriding only `MaxBudgetUSD`); asserts every other field falls through to Preset, `MaxBudgetUSD` reflects the override.
  - Test `TestResolve_MapMerge` loads `inheritance_map_merge.toml` (Preset `EnvSet = { A = "1" }`, override `EnvSet = { B = "2" }`); asserts resolved `EnvSet = { A = "1", B = "2" }`.
  - Test `TestResolve_ListReplace` loads `inheritance_list_replace.toml` (Preset `ToolsAllow = ["Read", "Bash"]`, override `ToolsAllow = ["Read"]`); asserts resolved `ToolsAllow = ["Read"]` (full replace, not merge).
  - Test `TestResolve_ExplicitEmptyList` constructs an `Override` in code with `ToolsDeny = &[]string{}` over a Preset with `ToolsDeny = ["rm"]`; asserts resolved `ToolsDeny = []string{}` (explicit empty wins).
  - `mage test-pkg ./internal/config` passes; coverage of merge edge cases is golden-fixture-based per `SKETCH.md` § 26.W0 AcceptanceCriteria bullet "Override merge implements field-level + map-level + list-replace semantics per §5; covered by golden-fixture tests."
- **Blocked by:** 4c.6.W0.D1 (D2 consumes the `Preset` / `Override` / `AgentRuntime` / `Kind` types defined by D1 AND edits the same `agents.go` file AND shares the `internal/config` package compile/test unit; serial chain on the package compile is the binding constraint per planning-agent HARD RULES).
- **Specify (droplet-scope; inherits `SKETCH.md` § 26.W0):**
  - **Objective:** Implement the field-level inheritance merge engine that turns a loaded `AgentsRegistry` into per-kind `AgentRuntime` values. Foundational for D3 (which deep-merges `agents.local.toml` over D2's resolved values) and W3's runtime config consumer (which spawns agents via `Resolve(reg, kind)`).
  - **AcceptanceCriteria:** see Acceptance bullets above.
  - **ValidationPlan:** `mage test-pkg ./internal/config`; `mage test-func ./internal/config TestResolve_FullInherit`; same for `TestResolve_PartialOverride`, `TestResolve_MapMerge`, `TestResolve_ListReplace`, `TestResolve_ExplicitEmptyList`; `mage ci` green.
  - **RiskNotes:**
    - Empty-list-vs-nil discrimination is the trickiest semantic edge: `nil` slice on `Override.ToolsAllow` means "inherit"; `&[]string{}` means "explicit empty". The pointer-to-slice idiom carries the discrimination — verify the merge function doesn't conflate them.
    - Map-merge order: per-kind override keys win over Preset keys; documented in §4.2.2. Make this explicit in the merge helper's doc-comment.
    - Recursive merges: `Preset` and `Override` are flat; no nested struct merge needed. If a future field is itself a struct, the merge model needs revisiting — out of scope for D2.
    - Sorted-key iteration for determinism: when iterating `EnvSet` / `EnvFromShell` keys for any test assertion or error-message construction, sort keys to keep output stable across map iteration order.
  - **ContextBlocks:**
    - `constraint` (high): `Override.<scalar-field>` semantics: nil pointer = inherit, non-nil = use override (even if zero value).
    - `constraint` (high): map fields merge per-key (override wins), NOT full-replace.
    - `constraint` (high): list fields full-replace if override non-nil; explicit empty `&[]string{}` replaces non-empty Preset.
    - `reference` (normal): `SKETCH.md` § 4.2.1 (field-level inheritance) + § 4.2.2 (map per-key merge) + § 4.2.3 (list full-replace).
    - `decision` (normal): pointer-to-slice discrimination for empty-vs-absent — chosen at D1; D2 honors it.
  - **KindPayload:** `{"changes":[{"file":"internal/config/agents.go","symbol":"Resolve, mergeMaps, replaceList","action":"add","shape_hint":"Resolve(registry, kind) AgentRuntime walks Preset fields and applies override pointers; mergeMaps merges EnvSet/EnvFromShell per-key; replaceList handles list full-replace + empty-vs-nil discrimination"},{"file":"internal/config/agents_test.go","symbol":"TestResolve_FullInherit, TestResolve_PartialOverride, TestResolve_MapMerge, TestResolve_ListReplace, TestResolve_ExplicitEmptyList","action":"add","shape_hint":"table-driven over golden fixtures + one in-code-constructed Override for the explicit-empty-list edge"},{"file":"internal/config/testdata/agents/inheritance_full_inherit.toml","symbol":"golden fixture","action":"add","shape_hint":"Preset only, no per-kind blocks"},{"file":"internal/config/testdata/agents/inheritance_partial_override.toml","symbol":"golden fixture","action":"add","shape_hint":"Preset + [agents.build] overriding only max_budget_usd"},{"file":"internal/config/testdata/agents/inheritance_map_merge.toml","symbol":"golden fixture","action":"add","shape_hint":"Preset env_set + override env_set with disjoint keys"},{"file":"internal/config/testdata/agents/inheritance_list_replace.toml","symbol":"golden fixture","action":"add","shape_hint":"Preset tools_allow + override tools_allow with shorter list"}]}`

### Droplet 4c.6.W0.D3 — `Merge(local, project)` deep-merge for `agents.local.toml` + `tools_deny` rejection

- **State:** todo
- **Kind:** `build` (atomic droplet; `Irreducible: true`)
- **Paths:** `internal/config/agents.go` (MODIFY — append `MergeLocal` function + `ErrToolsDenyNotOverridable` sentinel), `internal/config/agents_test.go` (MODIFY — append local-merge tests), `internal/config/testdata/agents/local_*.toml` (NEW — `local_override_model.toml`, `local_tools_deny_rejected.toml`, `local_partial_block.toml`).
- **Packages:** `internal/config`.
- **Acceptance:**
  - `MergeLocal(project *AgentsRegistry, local *AgentsRegistry) (*AgentsRegistry, error)` returns a new `AgentsRegistry` with `local`'s fields deep-merged OVER `project`'s. Field-level deep-merge per `SKETCH.md` § 5: top-level fields → per-field replacement (local wins if present); `EnvSet` / `EnvFromShell` → per-key merge; lists `CliArgs` / `ToolsAllow` / `ClaudeMDAddons` → full-replace.
  - `tools_deny` set in `local` (any per-kind block OR the `[agents]` defaults) returns `ErrToolsDenyNotOverridable` — closed sentinel error identifying the offending kind. Per § 4.3.1 + § 5: "`tools_deny` is the safety floor; setting it in `.local.toml` fails loud." TOML-position wrapping is added by D5's envelope; D3 raises the bare sentinel.
  - The bare sentinel's message reads `"tools_deny is not user-overridable; remove the field"` (no file/line/block prefix at the D3 boundary). D5 wraps the sentinel into `*ConfigError` so the user-facing message renders as `"agents.local.toml [agents.<kind>]:<line>: tools_deny is not user-overridable; remove the field"` (or `[agents]:<line>` for the defaults block).
  - `MergeLocal(project, nil)` returns `project` unchanged (nil local = no local file present; valid).
  - `MergeLocal(nil, local)` returns an error — local without project is invalid (per § 3.3 "`agents.toml` is required").
  - Deep-merge preserves `project`'s per-kind overrides where `local` doesn't override them; keys present in `local` win at the field level (NOT block-level — within a `[agents.build]` block, local's `Model` overrides project's, but project's `MaxBudgetUSD` survives if local doesn't set it).
  - Test `TestMergeLocal_OverrideModel` loads `local_override_model.toml`; asserts resolved `[agents.build].Model` reflects local, other build fields fall through.
  - Test `TestMergeLocal_ToolsDenyRejected` loads `local_tools_deny_rejected.toml`; asserts `errors.Is(err, ErrToolsDenyNotOverridable)` succeeds against the sentinel returned by `MergeLocal`. (Position-wrapping at the envelope layer is asserted separately by D5's `TestMergeLocal_ToolsDenyPositionWrapped`; this bullet covers only the sentinel-rejection contract D3 owns.)
  - Test `TestMergeLocal_NilLocal` constructs a project registry and calls `MergeLocal(project, nil)`; asserts returned registry is equivalent to `project` (deep-equal via `reflect.DeepEqual` or per-field assertion).
  - Test `TestMergeLocal_PartialBlock` loads `local_partial_block.toml` (local only sets `[agents.build].Model`); asserts project's other fields in `[agents.build]` survive.
  - `mage test-pkg ./internal/config` passes.
- **Blocked by:** 4c.6.W0.D2 (D3 calls `Resolve` internally for both project and local? — no, D3 operates at the registry layer NOT the resolved-runtime layer; but D3 still depends on D2 because D3's tests need D2's `Resolve` to verify the merged registry resolves correctly. Plus same-package serial chain.).
- **Specify (droplet-scope; inherits `SKETCH.md` § 26.W0):**
  - **Objective:** Implement the deep-merge engine for `agents.local.toml` over the project's `agents.toml`. Adds the `tools_deny` safety floor: local CANNOT relax denials. The `MergeLocal` function operates BEFORE D2's `Resolve` (i.e., merges the two registries; D2's resolver then operates over the merged result). This ordering is load-bearing: per-kind overrides in `local` must merge into project's per-kind overrides BEFORE the kind-level resolution happens, otherwise `local` partial blocks would full-replace project blocks instead of field-merging.
  - **AcceptanceCriteria:** see Acceptance bullets above.
  - **ValidationPlan:** `mage test-pkg ./internal/config`; `mage test-func ./internal/config TestMergeLocal_OverrideModel`; same for `TestMergeLocal_ToolsDenyRejected`, `TestMergeLocal_NilLocal`, `TestMergeLocal_PartialBlock`; `mage ci` green.
  - **RiskNotes:**
    - Order matters: `MergeLocal(project, local)` runs BEFORE `Resolve(merged, kind)`. Document in `MergeLocal`'s doc-comment with a usage example.
    - `tools_deny` rejection MUST surface a position-aware error to the user — without TOML-line context, the user gets a hostile "your file is broken somewhere" message. D3 raises the bare sentinel; D5's envelope adds the file/block/line wrapping. D3's tests assert sentinel-rejection only; D5's tests assert the wrapping.
    - `tools_deny` in the `[agents]` defaults block of local (not just per-kind blocks) must also be rejected — `SKETCH.md` § 4.3.1 says "MUST NOT override," not "MUST NOT override per-kind." Test the defaults-block case explicitly.
    - Map-merge semantics in D3 mirror D2's per-key merge for `EnvSet` / `EnvFromShell`. Reuse D2's `mergeMaps` helper to avoid drift.
  - **ContextBlocks:**
    - `constraint` (critical): `tools_deny` is NEVER user-overridable; setting it in `.local.toml` fails loud via the closed sentinel `ErrToolsDenyNotOverridable` (D3) wrapped into a position-aware `*ConfigError` envelope (D5). Safety floor per `SKETCH.md` § 4.3.1.
    - `constraint` (high): `MergeLocal` operates at registry-level BEFORE `Resolve` (kind-level); ordering is load-bearing.
    - `constraint` (high): per-kind blocks in local field-merge (not block-replace) over project.
    - `reference` (normal): `SKETCH.md` § 4.3 + § 4.3.1 + § 5.
    - `decision` (normal): closed sentinel error `ErrToolsDenyNotOverridable` per `CLAUDE.md` § "Errors" wrapping discipline.
  - **KindPayload:** `{"changes":[{"file":"internal/config/agents.go","symbol":"MergeLocal, ErrToolsDenyNotOverridable","action":"add","shape_hint":"MergeLocal(project, local) deep-merges local registry over project; rejects local tools_deny by returning bare ErrToolsDenyNotOverridable sentinel (D5 wraps with file/line/block); reuses D2's mergeMaps helper"},{"file":"internal/config/agents_test.go","symbol":"TestMergeLocal_OverrideModel, TestMergeLocal_ToolsDenyRejected, TestMergeLocal_NilLocal, TestMergeLocal_PartialBlock","action":"add","shape_hint":"table-driven; tools_deny test asserts errors.Is(err, ErrToolsDenyNotOverridable); line-number wrapping asserted separately by D5's TestMergeLocal_ToolsDenyPositionWrapped"},{"file":"internal/config/testdata/agents/local_override_model.toml","symbol":"golden fixture","action":"add","shape_hint":"local sets [agents.build].model only"},{"file":"internal/config/testdata/agents/local_tools_deny_rejected.toml","symbol":"golden fixture","action":"add","shape_hint":"local sets [agents.build].tools_deny — must reject"},{"file":"internal/config/testdata/agents/local_partial_block.toml","symbol":"golden fixture","action":"add","shape_hint":"local sets [agents.build].model; project has [agents.build] with several other fields"}]}`

### Droplet 4c.6.W0.D4 — Frontmatter `model:` / `tools:` strip helper

- **State:** todo
- **Kind:** `build` (atomic droplet; `Irreducible: true`)
- **Paths:** `internal/config/frontmatter.go` (NEW), `internal/config/frontmatter_test.go` (NEW). Sibling files inside the `internal/config` package; NO overlap with `agents.go` / `agents_test.go`.
- **Packages:** `internal/config`.
- **Acceptance:**
  - `StripFrontmatterKeys(frontmatter string, stripModel bool, stripTools bool) (string, error)` is a pure function — no I/O, no global state. Takes a YAML frontmatter STRING (the content between the leading `---` and trailing `---` of an agent .md file's frontmatter section, NOT including the delimiters), returns the same string with `model:` removed when `stripModel=true` and/or `tools:` (plus `allowedTools:` / `disallowedTools:` per `SKETCH.md` § 15) removed when `stripTools=true`.
  - Implementation chooses the smallest YAML-aware approach: parse YAML to a node tree, drop keys, re-emit. If the frontmatter is invalid YAML, return an error with the parser's message.
  - Preserves field order for fields NOT being stripped (deterministic output).
  - Idempotent: calling `StripFrontmatterKeys` twice with the same args returns the same string.
  - When BOTH flags are false, returns the input string verbatim (no parse, no re-emit) — the frontmatter survives untouched.
  - `model:` strip removes ONLY the `model:` top-level key; nested keys named `model:` (e.g., inside an arbitrary mapping) are NOT stripped — only top-level YAML keys.
  - `tools:` strip removes the top-level `tools:` key AND `allowedTools:` AND `disallowedTools:` per `SKETCH.md` § 15 ("No `model:`, no `tools:`, no `allowedTools` / `disallowedTools`").
  - Test `TestStripFrontmatterKeys_StripModel` feeds `"name: foo\ndescription: bar\nmodel: claude-sonnet-4-6\n"` with `stripModel=true, stripTools=false`; asserts output is `"name: foo\ndescription: bar\n"`.
  - Test `TestStripFrontmatterKeys_StripTools` feeds frontmatter with `tools:`, `allowedTools:`, `disallowedTools:` keys; asserts all three removed when `stripTools=true`.
  - Test `TestStripFrontmatterKeys_BothFalse` feeds any frontmatter with `stripModel=false, stripTools=false`; asserts output equals input verbatim (NO parse cycle, NO whitespace normalization).
  - Test `TestStripFrontmatterKeys_PreservesOtherFields` asserts `name:` and `description:` survive when other fields are stripped.
  - Test `TestStripFrontmatterKeys_InvalidYAML` feeds malformed YAML; asserts error returned + error message includes parse-position info.
  - Test `TestStripFrontmatterKeys_Idempotent` calls the function twice with the same args; asserts output equals.
  - `mage test-pkg ./internal/config` passes; `mage test-func ./internal/config TestStripFrontmatterKeys_StripModel` and the other 5 tests pass individually.
- **Blocked by:** 4c.6.W0.D3 (same-package serial chain — `internal/config` shares one compile unit per planning-agent HARD RULES, even though `frontmatter.go` is file-disjoint from `agents.go`).
- **Specify (droplet-scope; inherits `SKETCH.md` § 26.W0):**
  - **Objective:** Ship the pure helper function W3 will call into at render time to strip `model:` / `tools:` from agent .md frontmatter when `agents.toml` has the corresponding keys set. Lives in `frontmatter.go` (sibling to `agents.go`) so the schema layer (D1-D3, D5) and the render-helper layer (D4) are file-isolated within the same package — minimizes merge friction if either layer evolves independently.
  - **AcceptanceCriteria:** see Acceptance bullets above.
  - **ValidationPlan:** `mage test-pkg ./internal/config`; per-test `mage test-func` invocations as listed; `mage ci` green.
  - **RiskNotes:**
    - YAML lib choice: `SKETCH.md` § 26.W0 RiskNotes says "Frontmatter YAML lib choice: pick smallest dep." Survey existing deps via `go.mod` first — if `gopkg.in/yaml.v3` is already imported transitively (verify via `go list -m all` or Hylla), reuse it. If not, pick `gopkg.in/yaml.v3` (stable, widely used, supports node-tree manipulation needed for order-preserving strip). Avoid `goccy/go-yaml` and `kubernetes-sigs/yaml` for dep-weight reasons. Builder verifies before authoring.
    - Order preservation: naive `map[string]interface{}` decode loses YAML key order. Use `yaml.Node` API for order-preserving manipulation. Documented in builder spec.
    - Top-level-only key stripping: `model:` nested in a mapping value (e.g., `metadata:\n  model: ...`) MUST NOT be stripped. Test exercises this.
    - Both-false short-circuit: skip parsing entirely when both flags are false to preserve original whitespace / comments byte-for-byte. Tests exercise this idempotency.
    - W3 wires this helper at `render.go:assembleAgentFileBody` (per `SKETCH.md` § 26.W3); D4 ships the helper but does NOT wire it. Wiring is W3's responsibility per cascade-design parallelization.
  - **ContextBlocks:**
    - `constraint` (high): pure function — no I/O, no global state. Render-time callable from any goroutine.
    - `constraint` (high): top-level YAML keys only; nested `model:` / `tools:` keys survive.
    - `decision` (normal): YAML lib `gopkg.in/yaml.v3` (subject to dep-survey at build time).
    - `reference` (normal): `SKETCH.md` § 4.4 (frontmatter strip behavior) + § 15 (frontmatter narrowed to `name` + `description`).
    - `warning` (normal): both-false short-circuit preserves whitespace byte-for-byte; do NOT parse-and-reemit when both flags are false.
  - **KindPayload:** `{"changes":[{"file":"internal/config/frontmatter.go","symbol":"StripFrontmatterKeys","action":"add","shape_hint":"new file; pure function (frontmatter string, stripModel bool, stripTools bool) (string, error); uses yaml.Node tree for order-preserving strip; both-false short-circuit returns input verbatim"},{"file":"internal/config/frontmatter_test.go","symbol":"TestStripFrontmatterKeys_StripModel, TestStripFrontmatterKeys_StripTools, TestStripFrontmatterKeys_BothFalse, TestStripFrontmatterKeys_PreservesOtherFields, TestStripFrontmatterKeys_InvalidYAML, TestStripFrontmatterKeys_Idempotent","action":"add","shape_hint":"table-driven where natural; explicit cases for both-false short-circuit + idempotency"}]}`

### Droplet 4c.6.W0.D5 — Position-tracking error envelope

- **State:** todo
- **Kind:** `build` (atomic droplet; `Irreducible: true`)
- **Paths:** `internal/config/agents.go` (MODIFY — append `ConfigError` envelope type + `WrapWithPosition` helper + update `LoadRegistry` / `MergeLocal` to wrap raw `*toml.DecodeError` returns into the envelope), `internal/config/agents_test.go` (MODIFY — append envelope tests).
- **Packages:** `internal/config`.
- **Acceptance:**
  - `ConfigError` is a typed error struct with fields `File string`, `Block string` (e.g., `"[agents.build]"` or `"[agents]"`), `Line int`, `Cause error`. Its `Error()` method formats: `"<file> <block>:<line>: <cause>"` (e.g., `"agents.toml [agents.build]:42: tools_deny is not user-overridable; remove the field"`).
  - `errors.Unwrap(*ConfigError)` returns `Cause` so `errors.Is` / `errors.As` work transitively against sentinels (e.g., `errors.Is(err, ErrToolsDenyNotOverridable)` succeeds when the envelope wraps that sentinel).
  - `WrapWithPosition(err error, file string, decodeErr *toml.DecodeError) error` constructs a `*ConfigError` with `File`, `Block` (extracted from `decodeErr.Key()` per `pelletier/go-toml/v2` API — verify exact method name via `go doc` before authoring), `Line` (from `decodeErr.Position()` per same API verification), and `Cause = err`.
  - `LoadRegistry` (D1) wraps every raw `*toml.DecodeError` it receives into `*ConfigError` with `File = path`. Update D1's `TestLoadRegistry_MalformedTOML` expectation to assert the returned error is `*ConfigError` AND `errors.As(err, &configErr)` extracts the position; the existing assertion that the error message contains the TOML line number survives because `*ConfigError.Error()` includes the line.
  - `MergeLocal` (D3) wraps `ErrToolsDenyNotOverridable` into `*ConfigError` with `File = "agents.local.toml"` (or whatever path `MergeLocal`'s caller passes — D5 adds an optional `LocalPath string` field on the registry or threads the path through `MergeLocal`'s signature; pick whichever fits the existing API surface — builder decides at implementation time).
  - Test `TestConfigError_FormatsCorrectly` constructs a `*ConfigError` with known fields; asserts `Error()` returns the expected `"<file> <block>:<line>: <cause>"` shape.
  - Test `TestConfigError_UnwrapPreservesSentinel` constructs a `*ConfigError{Cause: ErrToolsDenyNotOverridable}`; asserts `errors.Is(err, ErrToolsDenyNotOverridable)` returns true.
  - Test `TestLoadRegistry_PositionWrapped` (REVISES D1's TestLoadRegistry_MalformedTOML expectation) loads a malformed fixture; asserts `errors.As(err, &configErr)` succeeds AND `configErr.Line > 0`.
  - Test `TestMergeLocal_ToolsDenyPositionWrapped` (REVISES D3's TestMergeLocal_ToolsDenyRejected expectation) loads `local_tools_deny_rejected.toml`; asserts `errors.As(err, &configErr)` succeeds AND `configErr.File == "agents.local.toml"` AND `errors.Is(err, ErrToolsDenyNotOverridable)`.
  - `mage test-pkg ./internal/config` passes; `mage ci` green.
- **Blocked by:** 4c.6.W0.D4 (same-package serial chain — D5 modifies `agents.go` AND revises tests authored by D1+D3, so D5 lands LAST in the chain to consolidate position-tracking across all decode + merge errors).
- **Specify (droplet-scope; inherits `SKETCH.md` § 26.W0):**
  - **Objective:** Wrap every position-aware error (TOML decode failures from D1, `tools_deny` rejection from D3) into a unified `ConfigError` envelope so callers get a single error type to inspect (`errors.As`) AND a uniform format string. Closes `SKETCH.md` § 26.W0 AcceptanceCriteria bullet "TOML loader produces TOML-line-aware errors for missing required fields, malformed env-var names, duplicate keys" — by giving downstream consumers (W3's render layer, future MCP boundary) a single envelope to format.
  - **AcceptanceCriteria:** see Acceptance bullets above.
  - **ValidationPlan:** `mage test-pkg ./internal/config`; per-test `mage test-func` for the four new tests; re-run D1's `TestLoadRegistry_MalformedTOML` and D3's `TestMergeLocal_ToolsDenyRejected` (revised assertions); `mage ci` green.
  - **RiskNotes:**
    - `pelletier/go-toml/v2` position API (`DecodeError.Position()` / `.Key()`) — verify via `go doc github.com/pelletier/go-toml/v2.DecodeError` before authoring; the lib has minor API churn across versions and the project's pinned version may or may not expose what D5 needs. If the API doesn't expose `.Key()` directly, reconstruct the block string from `DecodeError`'s string method or its line number alone.
    - `MergeLocal` doesn't currently parse TOML (it operates over already-decoded `AgentsRegistry` structs); the line number for `tools_deny` rejection comes from D3's load step, NOT from `MergeLocal`'s own logic. D5 either threads the source-line through `AgentsRegistry` (e.g., a `tools_deny_line int` field) OR `MergeLocal`'s caller re-decodes the file to extract the line. The simpler approach: `LoadRegistry` records line numbers for sensitive fields on the decoded struct (e.g., a `linePositions map[string]int`) and `MergeLocal` reads them. Builder picks the simpler approach at implementation time.
    - `errors.Unwrap` chain depth: keep `ConfigError` as a single-level wrapper; don't chain envelopes. Future composition (validator findings wrapping multiple envelopes) is W0.5's problem, not W0's.
    - Format-string stability: the format `"<file> <block>:<line>: <cause>"` is a public surface (downstream tests will match against it). Document the format in a doc-comment on `ConfigError.Error()`; avoid changing it lightly.
  - **ContextBlocks:**
    - `constraint` (critical): `*ConfigError` MUST implement `Unwrap() error` returning `Cause` so `errors.Is` / `errors.As` work transitively against sentinels.
    - `constraint` (high): every error returned from `LoadRegistry` / `MergeLocal` is `*ConfigError` (or wrapped one); raw `*toml.DecodeError` MUST NOT escape the `internal/config` package.
    - `decision` (normal): single-level wrapper (no envelope-chaining).
    - `reference` (normal): `pelletier/go-toml/v2` `DecodeError` API — verify via `go doc` before authoring per `CLAUDE.md` § Tool Discipline.
    - `warning` (normal): D5 revises D1+D3 test assertions; tests must be updated in the same droplet to keep CI green at boundary.
  - **KindPayload:** `{"changes":[{"file":"internal/config/agents.go","symbol":"ConfigError, WrapWithPosition","action":"add","shape_hint":"struct ConfigError{File, Block string; Line int; Cause error}; Error() formats <file> <block>:<line>: <cause>; Unwrap() returns Cause; WrapWithPosition wraps DecodeError into ConfigError"},{"file":"internal/config/agents.go","symbol":"LoadRegistry, MergeLocal","action":"modify","shape_hint":"wrap raw *toml.DecodeError + ErrToolsDenyNotOverridable returns into *ConfigError; LoadRegistry threads source-line for sensitive fields onto AgentsRegistry"},{"file":"internal/config/agents_test.go","symbol":"TestConfigError_FormatsCorrectly, TestConfigError_UnwrapPreservesSentinel, TestLoadRegistry_PositionWrapped, TestMergeLocal_ToolsDenyPositionWrapped","action":"add","shape_hint":"new tests for envelope; revises D1+D3 prior-test assertions to expect *ConfigError shape"}]}`

## Notes

### Decomposition rationale

- **Five droplets, not three or seven.** Three would conflate distinct schema-layer concerns (decode + resolve + local-merge in one droplet would exceed atomic-sizing budget; ~250+ LOC + tests). Seven would over-decompose: e.g., splitting `MergeLocal` from its `tools_deny` rejection makes the rejection test a weird floating addendum without a home. Five hits the sweet spot per `~/.claude/agents/go-planning-agent.md` § "Atomic droplet sizing (till-go template values)" — each droplet 80-120 LOC + tests + ideally one production file.

- **D4 (frontmatter) is sibling-file, NOT sibling-package.** Sketch §5 + §15 confirm the strip helper is a render-time pure function with no schema-layer dependency; placing it in `frontmatter.go` keeps `agents.go` focused on schema + merge. But the `internal/config` package is shared, so D4 still chains in the package-locked serial sequence per planning-agent HARD RULES. The file separation matters for future maintainability (W3's render-layer consumer reaches for `frontmatter.go` cleanly), not for parallelism.

- **D5 (error envelope) lands LAST.** D5 revises D1's malformed-TOML test assertion and D3's tools-deny test assertion to expect the envelope shape. Sequencing D5 after D3-D4 means D1+D3 land their tests with a simpler assertion (raw error type), then D5 upgrades them. This keeps each prior droplet's CI passable on its own boundary — if D5 were sequenced earlier, D3 would have to anticipate the envelope, which violates the "smallest concrete design" principle. The cost is one assertion-revision per touched test file; the benefit is each prior droplet ships a self-consistent vertical slice.

- **All five droplets serialize on `internal/config`.** Per planning-agent HARD RULES: package-shared siblings MUST have explicit `blocked_by`. The chain is strict (`D1 → D2 → D3 → D4 → D5`). NO parallelism within W0. The compensating factor: each droplet is genuinely small (~80-120 LOC), so the serial chain completes in 5 builder spawns, not 5 days of work.

### Out of scope for W0

- **Wiring `LoadRegistry` into the application.** No call site reads `agents.toml` yet — that's W2's `till init` (which writes the file) and W3's render-layer consumer (which reads the file). W0 ships the loader + types; downstream waves wire them.
- **`agents.example.toml` content.** W1 ships the example file with sane defaults per `SKETCH.md` § 4.2. W0 only ships the schema types that example file decodes into.
- **Embedded agent .md frontmatter shape (3-tier resolution).** W1 ships the placeholder agent .md files; W3 ships the resolution priority. W0 only ships the strip helper W3 calls into.
- **MCP boundary error formatting.** W11 (Drop 4c.7) wires the runtime fail-loud at the MCP boundary. W0's `*ConfigError` envelope is the substrate; W11 consumes it.

### Cross-references

- **`SKETCH.md` § 26.W0** — wave-Specify source of truth.
- **`SKETCH.md` § 4** — schema definition.
- **`SKETCH.md` § 4.2.1, § 4.2.2, § 4.2.3** — inheritance semantics.
- **`SKETCH.md` § 4.3, § 4.3.1** — `agents.local.toml` override + `tools_deny` floor.
- **`SKETCH.md` § 4.4** — frontmatter `model:` / `tools:` strip.
- **`SKETCH.md` § 5** — override semantics summary.
- **`SKETCH.md` § 15** — frontmatter narrowed shape.
- **`workflow/drop_4c_6/PLAN.md`** lines 51-67 — L1 W0 sub-plan container row.
- **`~/.claude/agents/go-planning-agent.md`** § "Cascade Design — Atomic Droplets + Parallelization (HARD RULES)" — sizing + parallelization.
- **`~/.claude/agents/go-planning-agent.md`** § "Tillsyn-Flavored Specify Pass" — droplet Specify shape.
- **`workflow/example/drops/WORKFLOW.md`** § "`_BLOCKERS.toml` — Sibling Blocker Ledger" — companion `_BLOCKERS.toml` schema.
