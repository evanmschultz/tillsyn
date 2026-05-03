# DROP 3 — UNIT B PLAN QA FALSIFICATION — ROUND 1

**Reviewer:** `go-qa-falsification-agent`
**Subject:** `workflow/drop_3/UNIT_B_PLAN.md` (Drop 3 Unit B — Template System Overhaul, 9 droplets).
**State:** falsification round 1.
**Verdict:** **FAIL — multiple confirmed counterexamples.** Plan must be revised before builders fire on droplets 3.B.1, 3.B.5, 3.B.7, 3.B.8, 3.B.9.

## Counterexamples

### CE1 — `//go:embed` cannot reach repo-root `templates/builtin/default.toml` from `internal/templates/embed.go`

- **Hit.** Droplet 3.B.7. CONFIRMED, breaks the build.
- **Claim.** "`templates/builtin/default.toml` (new — repo root, NOT under `internal/`; lives as a sibling of `cmd/`, `internal/`, `magefile.go`)" + "`internal/templates/embed.go` (new — Go `//go:embed` directive bringing the TOML file into the binary)".
- **Counterexample.** `go doc embed` (Go stdlib): _"Patterns are interpreted relative to the package directory containing the source file. … Patterns may not contain '.' or '..' or empty path elements, nor may they begin or end with a slash."_ An `internal/templates/embed.go` file cannot embed `templates/builtin/default.toml` because the only relative path that reaches it (`../../templates/builtin/default.toml`) contains `..` segments, which Go rejects at build time. Drop 2.1's deleted `templates/embed.go` (commit 29cce29) sat at repo root precisely for this reason — putting `embed.go` under `internal/templates/` while the TOML stays at repo root is a layout the toolchain refuses to compile.
- **Repro.** No mage repro needed — toolchain spec is in `go doc embed`. A builder following 3.B.7 verbatim will see `pattern ../../templates/builtin/default.toml: invalid pattern syntax` from the Go compiler.
- **Mitigation options.** Either (a) move `embed.go` to repo-root `templates/embed.go` and have `internal/templates/` import it (matches Drop 2.1's deleted layout), or (b) move the TOML under `internal/templates/builtin/default.toml`. Plan must pick one explicitly; "builder picks" is unsafe because option (a) requires Go-package gymnastics (a top-level `templates` package) and option (b) contradicts the planner's "lives as a sibling of `cmd/`, `internal/`, `magefile.go`" assertion.

### CE2 — Droplet 3.B.8 omits multiple call sites of `KindTemplate` + `AllowedParentScopes`

- **Hit.** Droplet 3.B.8. CONFIRMED, the rewrite/delete sweep is incomplete.
- **Claim.** "Every consumer of those types now reads through `KindCatalog`. … LSP `findReferences` on each deleted symbol returns 0 hits."
- **Counterexample.** LSP `findReferences` on `KindTemplate` (`internal/domain/kind.go:105`) returns 25 hits across 8 files; on `AllowedParentScopes` (`internal/domain/kind.go:118`) returns 15 hits across 7 files. The plan's path list misses at least:
  - `internal/adapters/server/mcpapi/instructions_explainer.go:241-242` — reads `kind.AllowedParentScopes` to build human-readable instruction text. Not in 3.B.8's path list.
  - `internal/adapters/storage/sqlite/repo.go:1066, 1100, 2981` — `CreateKindDefinition` / `UpdateKindDefinition` / row-decode JSON-marshal `kind.AllowedParentScopes` into the `allowed_parent_scopes_json` SQLite column. Plan only lists `:286-377` (the boot-seed) — misses every CRUD path.
  - `internal/adapters/storage/sqlite/repo.go:1070, 1104, 2987` — same three CRUD paths for `kind.Template` JSON marshaling into `template_json`.
  - `internal/app/snapshot.go:1098, 1100, 1345, 1347` — `SnapshotKindDefinition` import/export carries `AllowedParentScopes` + `Template`. Plan lists `:94` only.
  - `internal/adapters/server/mcpapi/extended_tools_test.go:712` — test fixture builds `domain.KindDefinition{AllowedParentScopes: ...}`. Not in path list.
  - `cmd/till/main.go:3617, 3619` — CLI output struct populates `AllowedParentScopes` + `Template` for `till kind list/get`. Plan lists `:3042, 3045, 3047, 3049, 3442` only — misses both.
  - `internal/domain/kind_capability_test.go:18-73` — test heavily exercises `KindTemplate`, `KindTemplateChildSpec`, `AllowedParentScopes`. Plan lists `:18, 20, 49` only; in reality the entire test body (24 lines of dependency) needs rewriting.
  - `internal/app/kind_capability.go:751-799` — `mergeActionItemMetadataWithKindTemplate` and `validateKindTemplateExpansion` reach deep into `kind.Template.AutoCreateChildren`, `kind.Template.CompletionChecklist`, `kind.Template.ActionItemMetadataDefaults`. Plan lists `:566, 750-766, 771` — misses the `:775-799` recursive expansion logic that operates on `Template.AutoCreateChildren`.
- **Net.** "LSP `findReferences` returns 0 hits" is the right acceptance criterion, but the path list is too thin for a builder to arrive at zero without doing their own discovery pass. Plan should either list every reference (the planner had LSP available — it's a 2-minute query), or replace the enumerated paths with a directive like "delete via LSP-driven sweep until `findReferences` returns 0 across the workspace."

### CE3 — Droplet 3.B.8 deletes the `kind_catalog` boot-seed but the existing test suite asserts the seeded values

- **Hit.** Droplet 3.B.8. CONFIRMED, breaks two existing test functions.
- **Claim.** Open Question 3: "planner recommends 3.B.8 deletes the seed INSERTs but **keeps the table** so existing schema doesn't break. Dev fresh-DBs after 3.B.8 lands."
- **Counterexample.** Two existing tests pin the boot-seed contents:
  - `internal/adapters/storage/sqlite/repo_test.go:2470-2517` — `TestRepositoryFreshOpen…` opens an in-memory DB and asserts `SELECT id FROM kind_catalog ORDER BY id` returns exactly the 12 seeded IDs (`build`, `build-qa-falsification`, `build-qa-proof`, `closeout`, `commit`, `discussion`, `human-verify`, `plan`, `plan-qa-falsification`, `plan-qa-proof`, `refinement`, `research`).
  - `internal/adapters/storage/sqlite/repo_test.go:2520-2568` — `TestRepositoryFreshOpenKindCatalogUniversalParentAllow` asserts `len(kinds) != 12` is fatal and probes every kind for `AllowsParentScope` under universal-allow.
  - Deleting the seed INSERTs from `repo.go:286-377` makes both tests `t.Fatalf` on a fresh in-memory DB.
- **What 3.B.8's acceptance section says.** It lists `internal/adapters/storage/sqlite/repo_test.go` only at `:2563` (the `AllowsParentScope` call site), not the surrounding test bodies that pin the seed. Per CE2 this is symptomatic — the acceptance criterion is "`mage test-pkg ./internal/adapters/storage/sqlite` passes," but the planner didn't verify that deletion of `:286-377` lets that target pass.
- **Mitigation.** 3.B.8 must explicitly delete or rewrite both `TestRepositoryFreshOpen…` test functions (not just the `AllowsParentScope` line). Drop 3 also needs to decide: do the equivalent assertions move to the new `internal/templates/embed_test.go` (asserting `default.toml` covers all 12 kinds), or do they evaporate entirely?

### CE4 — `KindCatalog` import-direction "open question" is a planning gap, not a builder choice

- **Hit.** Droplet 3.B.5. CONFIRMED as a planning incompleteness; will surface as a build-time circular import if builder picks the first option naively.
- **Claim.** "Two options documented in 3.B.5 acceptance section; builder picks. Either is sound."
- **Counterexample.** Both stated options have problems the planner hasn't resolved:
  - **Option A** ("`Project` struct field is `KindCatalogJSON json.RawMessage` and accessor methods on `Project` decode lazily"). If the accessor lives in `internal/domain/project.go`, the accessor must call into `internal/templates` to decode. That re-introduces `domain → templates` import. The lazy-decode dodge only works if the accessor lives outside `internal/domain` — i.e., on a separate type or in `internal/app` / `internal/templates` itself, never on `Project`. The plan says "accessor methods on `Project`" — that exact wording is the cycle.
  - **Option B** ("invert import direction"). `internal/templates` cannot then import `domain.Kind`, but 3.B.1's schema struct `KindRule.AllowedParentKinds []domain.Kind` requires that import. So inverting just moves the cycle.
  - Real third option (which the plan should bake): `internal/domain/project.go` carries `KindCatalogJSON json.RawMessage` only; decoding is done in `internal/app` or `internal/templates`, never on `Project` directly. That's a clean shape, but it's a different design than what the plan says.
- **Net.** Plan-QA failure: the planner punted a non-trivial import-graph design to the builder. A builder may pick Option A, hit the cycle, and ping back for guidance — wasted spawn cycle. Plan should converge before fire.

### CE5 — `Load`'s strict-unknown-key + schema-version-validate ordering is unspecified, and the natural order produces a confusing error

- **Hit.** Droplet 3.B.2. CONFIRMED, the user-facing error on a v2-doc-read-by-v1-parser is wrong.
- **Claim.** "Returns sentinel errors `ErrUnknownTemplateKey`, `ErrUnsupportedSchemaVersion`, …" + "Load-time validator: builds parent → child kind graph from `[child_rules]`, runs DFS to detect cycles, asserts every referenced `Kind` is in `domain.validKinds`."
- **Counterexample.** `pelletier/go-toml/v2 Decoder.DisallowUnknownFields()` (per `go doc`) returns `StrictMissingError` _during decode_, before any of the planner's load-time validators run. A v2 TOML doc that adds a new top-level field `[new_section]` will be rejected by a v1 parser with `StrictMissingError("unknown key new_section")`, not `ErrUnsupportedSchemaVersion`. The dev-facing error blames an unknown key when the real problem is a forward-version doc — confusing UX and a false trail when debugging.
- **Mitigation.** The plan needs to spec the order: (a) tolerant pre-pass that decodes only `schema_version` (separate `Decoder` instance without `DisallowUnknownFields`), (b) reject if `schema_version` is unknown, (c) only then strict-decode the rest. Or: catch `StrictMissingError`, re-classify if `schema_version` is known to be a future version. Either is fine; "builder picks" without flagging the ordering invites a wrong-error-on-future-doc bug that won't surface until v2 ships.

### CE6 — `GateRule` stub has no spec for where it attaches in the schema or what fields it carries

- **Hit.** Droplets 3.B.1, 3.B.6. CONFIRMED, Drop 4's dispatcher contract is undefined.
- **Claim.** 3.B.1: "`GateRule` struct stub for Drop 4 dispatcher consumption — schema-only, no behavior." `main/PLAN.md` § 19.3 line 1654: "Add gate definitions to kind templates." § 19.4 says dispatcher reads `agent_bindings` + `post_build_gates`.
- **Counterexample.** 3.B.1's acceptance lists the fields of every other struct (`Template`, `KindRule`, `ChildRule`, `AgentBinding`) but says nothing about `GateRule`'s fields. 3.B.6 covers the agent-binding fields but never re-mentions `GateRule`. The schema as planned has no field on `Template` or `KindRule` or `AgentBinding` named `Gates []GateRule` or similar — `GateRule` floats free, attached to nothing. Drop 4's dispatcher will need both:
  1. The shape of `GateRule` (name, mage target, on-failure behavior, blocked-state mapping).
  2. The location on the schema (`KindRule.Gates`? `AgentBinding.PostBuildGates`?).
- **Mitigation.** Either (a) Unit B explicitly defers `GateRule` to Drop 4's dispatcher unit, removing it from 3.B.1 entirely (and PLAN.md line 1654 moves to Drop 4); or (b) 3.B.1 specifies `GateRule`'s fields and where it attaches. Current state — "stub for Drop 4 consumption" — gives Drop 4 nothing to consume.

### CE7 — 3.B.3's 144-row test matrix is encoded inline, not derived from `default.toml`, so 3.B.3 and 3.B.7 can drift

- **Hit.** Droplets 3.B.3 + 3.B.7. CONFIRMED, multi-source-of-truth risk for the rule set.
- **Claim.** 3.B.3: "144-row table-driven test (12×12 Kind cartesian product) covering every combo from the default.toml shipped in 3.B.7. Reference matrix encoded inline in the test file."
- **Counterexample.** 3.B.3 is `blocked_by: 3.B.1, 3.B.2`, which puts it before 3.B.7 (default.toml). The "reference matrix encoded inline in the test file" lives in `internal/templates/nesting_test.go`. 3.B.7's `embed_test.go` independently asserts that "every reverse-hierarchy combo is rejected" against the loaded `default.toml`. There's no single-source-of-truth: a future template tweak that updates `default.toml` (3.B.7's file) needs to remember to update the inline matrix in `nesting_test.go` (3.B.3's file). Drift is the expected failure mode.
- **Mitigation.** Either (a) 3.B.3's test loads `default.toml` via embed and asserts against that as the reference (push 3.B.3 to `blocked_by: 3.B.7`), or (b) 3.B.3 uses a hand-coded `Template` value as test fixture, and 3.B.7's `embed_test.go` separately asserts the loaded `default.toml` round-trips against the same fixture. Current spec invites silent drift — a real plan-QA failure.

### CE8 — MCP/CLI wire surfaces accept `template` + `allowed_parent_scopes` JSON but 3.B.8 doesn't say what those surfaces become

- **Hit.** Droplet 3.B.8. CONFIRMED, public wire contract gap.
- **Claim.** 3.B.8 "rewrites or deletes" the old API. Path list includes `extended_tools.go:1682, 1778` and `mcp_surface.go:248`.
- **Counterexample.** `extended_tools.go:1675-1729` (one of the listed paths) is the inline-arg struct for `till.kind operation=upsert` MCP tool — it accepts `Template domain.KindTemplate` and `AllowedParentScopes []string` from external callers. `extended_tools.go:1738-1828` (legacy `till.upsert_kind_definition` alias) does the same. Plus `mcp_surface.go:248`'s `UpsertKindDefinitionRequest` struct, plus `cmd/till/main.go:3438-3446` (CLI JSON output type) and `cmd/till/main.go:3613-3623` (CLI builder/serializer). These are all **public wire surfaces** — once `KindTemplate` and `AllowedParentScopes` are deleted, MCP callers and `till` CLI users get either compile errors or a silent semantic drift.
- **What 3.B.8 should spec but doesn't.** For each public wire surface: do we (a) replace the type with the new TOML schema's `KindRule` / `Template`, (b) reject with a deprecation error pointing at `<project_root>/.tillsyn/template.toml`, or (c) delete the MCP tool / CLI subcommand entirely? The plan picks none. A builder will either silently break the wire contract or guess — both bad outcomes.
- **Mitigation.** 3.B.8 must explicitly classify each wire surface (`till.kind operation=upsert` MCP, `till.upsert_kind_definition` legacy alias, `till kind` CLI) into one of {migrated, deprecated, deleted} and route the deprecation. Pre-MVP rule says no migration logic, so likely answer is "delete the wire tool and CLI subcommand entirely; the kind catalog is now read-only at runtime, mutated only via TOML at project-create." But that's a design decision and the plan must make it.

## Nits

### N1 — `KindCatalog` runtime-mutability semantics are implicit

3.B.5 says "baked at project-creation time" but never explicitly says "edits to `<project_root>/.tillsyn/template.toml` after project create are ignored until dev fresh-DBs." Pre-MVP rule covers this implicitly, but acceptance criterion should make the spec explicit so future-drop reviewers don't assume hot-reload.

### N2 — Rejection-comment authorship doesn't cover dispatcher-driven auto-create

3.B.9 + Architectural Decision both assert "rejection-comments are write-as-the-rejecting-actor — no Unit C STEWARD principal needed." Fine for human/agent-driven creates, but the dispatcher (Drop 4) auto-creating children via `child_rules` may not have a single attributable actor. 3.B.9 should clarify scope: rejection-comments only on auth-gated creates; dispatcher-internal auto-create rejections route differently (likely a `failed` state on the parent, no comment).

### N3 — `default.toml` "implicit-by-absence vs explicit-`allowed_parent_kinds` exclusion" choice has a footgun

3.B.7's acceptance lets the planner pick either form. If they pick `allowed_parent_kinds` exclusion lists, adding a 13th kind in a future drop silently changes the semantic of every existing rule. Recommend forcing explicit `[child_rules]` deny rows so adding a new kind is an explicit opt-in to existing rules, not an implicit allow.

### N4 — `3.B.6` "convergence note" admits 3.B.6 might be a no-op

The note "if 3.B.1's spawn brief lets the builder land all fields up front, 3.B.6 collapses into a tightening + test pass" is fine as a note, but it means the planner has not decided whether 3.B.1's `AgentBinding` is full or skeletal. The two outcomes have different acceptance criteria. Plan should commit: either 3.B.1's `AgentBinding` is full and 3.B.6 is purely a test/round-trip droplet, or 3.B.1 ships skeletal and 3.B.6 fills it. Builder shouldn't read 3.B.1's output to decide.

## Verdict Summary

**FAIL.** Eight confirmed counterexamples, four nits. Most damaging is **CE1** (`//go:embed` toolchain rejection) — it stops the build outright on droplet 3.B.7. **CE2 + CE3** together mean droplet 3.B.8 is significantly under-specified and will leave dead code or broken tests on the first pass. **CE6** + **CE8** mean Drop 4's dispatcher and the existing wire-API contract have unresolved gaps the plan should close.

Recommended actions before round 2:

1. **CE1**: pick a `//go:embed`-compatible layout in 3.B.7 (move TOML under `internal/templates/builtin/` OR move embed.go to `templates/embed.go`). Update 3.B.7's path list.
2. **CE2**: replace 3.B.8's enumerated path list with an LSP-driven sweep directive, OR run `findReferences` on every deleted symbol and capture all 40+ hits.
3. **CE3**: 3.B.8 must explicitly delete or rewrite `TestRepositoryFreshOpen…` (lines 2470-2517) and `TestRepositoryFreshOpenKindCatalogUniversalParentAllow` (lines 2520-2568) in `internal/adapters/storage/sqlite/repo_test.go`.
4. **CE4**: 3.B.5 picks the import-direction shape (recommend: `Project.KindCatalogJSON json.RawMessage`, decoder in `internal/templates` or `internal/app`).
5. **CE5**: 3.B.2 specifies `Load`'s ordering (tolerant `schema_version` pre-pass before strict decode).
6. **CE6**: 3.B.1 specifies `GateRule`'s fields and attachment point, OR moves it to Drop 4 entirely.
7. **CE7**: 3.B.3's nesting test loads `default.toml` (push to `blocked_by: 3.B.7`), OR 3.B.7's `embed_test.go` cross-validates against 3.B.3's matrix.
8. **CE8**: 3.B.8 classifies each MCP/CLI wire surface (`till.kind operation=upsert`, `till.upsert_kind_definition` legacy, `till kind` CLI output struct) into {migrated, deprecated, deleted}.

## Hylla Feedback

N/A — review touched non-Go files (PLAN.md droplet text + UNIT_B_PLAN.md droplet text) plus targeted Go reads via Read + LSP. Hylla queries unnecessary; LSP `findReferences` covered the cross-reference work for `KindTemplate` / `AllowedParentScopes` / `KindTemplateChildSpec`. Context7 was unable to resolve `pelletier/go-toml/v2` (returned an unrelated `go-gh` library); fallback was `go doc github.com/pelletier/go-toml/v2 Decoder.DisallowUnknownFields` and `go doc embed`, both of which succeeded. No Hylla miss to record.
