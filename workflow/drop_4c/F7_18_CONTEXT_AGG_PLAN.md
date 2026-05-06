# Drop 4c F.7.18 — Context Aggregator Plan

**Scope:** F.7.18 sub-theme of Drop 4c Theme F.7. Decomposes the declarative context-aggregator surface into six concrete droplets owning Schema-2 (`Context` sub-struct on `AgentBinding`), Schema-3 (top-level `[tillsyn]` globals), the aggregator engine package, the greedy-fit + two-axis wall-clock cap algorithm, the default-template seeds, and the `metadata.spawn_history[]` audit-only doc-comment requirement.

**Authoring date:** 2026-05-04.
**Authoring planner:** F.7.18-context-aggregator (parallel to F.7-CORE planner for F.7.1–F.7.16 and F.7.17 CLI-adapter-seam planner).
**Mode:** Filesystem-MD planning before builder dispatch. No code edits in this plan; builders ship the code.

## Planner-Locked Architectural Decisions (sourced from SKETCH + planner review + falsification rounds)

These are the locked decisions that drive the per-droplet decomposition. Each cites the source so future readers can audit.

- **F.7.18 is FLEXIBLE not REQUIRED.** Adopters who declare `[context]` get bounded pre-staging; adopters who omit get fully agentic exploration. Both first-class. Source: SKETCH.md:180, 206–209; planner review §3.1; A5 verdict REFUTED-WITH-NIT.
- **Closed-struct schema with explicit TOML tags on every field.** Strict-decode (`internal/templates/load.go:88–95` `DisallowUnknownFields()`) automatically rejects unknown keys. Source: SKETCH.md:182; falsification A3.c.
- **NO schema rule against `descendants_by_kind` on `kind=plan`.** Template authors trusted (round-history fix-planners + tree-pruners legitimately use descendants). Source: SKETCH.md:199; planner review P6 amended (rejected the prior draft's schema rule).
- **Greedy-fit (not serial-drop) for bundle-cap algorithm.** Cheap-1 lands, busting-2 skipped with marker, cheap-3 still lands. Source: SKETCH.md:202; planner review P10; falsification A6.b CONFIRMED-NEW (chose greedy as mitigation).
- **Two-axis wall-clock**: per-rule `max_rule_duration` (default 500ms) + per-bundle `max_aggregator_duration` (default 2s). Per-rule cap localizes failures; per-bundle cap remains hard ceiling. Source: SKETCH.md:203–204; falsification A7.b CONFIRMED-NEW.
- **Round-history aggregation DEFERRED entirely.** `metadata.spawn_history[]` is audit-only (cost / denials / terminal_reason for ledger / dashboard). If a use case surfaces post-Drop-5, add `prior_round_*` rules targeting high-signal artifacts (worklog, gate output, QA findings) — NOT raw stream-json events. Source: SKETCH.md:205, 217; falsification A5.a + A5.b (REFUTED-WITH-NIT).
- **Aggregator engine lives in NEW `internal/app/dispatcher/context/` package.** Pure-function `Resolve(ctx, binding, item, repo) (Bundle, error)`. Per-binding scope only — does NOT iterate other bindings. Source: planner review §4.1.
- **Validation split.** Field-shape + closed-enum-membership validators land in `templates.Load` (static, no runtime deps). Aggregator engine assumes pre-validated input. Source: planner review §4.2 + P8.
- **Three-schema-droplet sequencing (locked in SKETCH:238–243).** Schema-1 (F.7.17 planner's territory) → Schema-2 (THIS plan) → Schema-3 (THIS plan). Each ~1/3 the review surface; independent failure domains. Strict-decode coherent because no droplet ships seed TOML referencing fields whose schema droplet hasn't landed yet.
- **Default-template seeds bounded.** Six bindings get `[context]` defaults: `build`, `build-qa-proof`, `build-qa-falsification`, `plan-qa-proof`, `plan-qa-falsification`, `plan`. Source: SKETCH.md:195–199. NO seeds for `commit`, `research`, `closeout`, `refinement`, `discussion`, `human-verify`.

## Hard Prereqs

- **F.7.17 Schema-1 droplet** (per-binding `command`, `args_prefix`, `env`, `cli_kind` fields on `AgentBinding`) MUST land before F.7.18.1 (Schema-2). Same struct, same file, same package — strict ordering required.
- **All Drop 4b core gates merged.** F.7.18 builds on the existing dispatcher package; no Drop 4b-specific dependency, but the package must be in its post-4b state.
- **Drop 4a Wave 1 first-class fields** (`paths []string`, `packages []string`, `start_commit string`, `end_commit string` on `domain.ActionItem`) — confirmed in `internal/domain/action_item.go`. Aggregator's `parent_git_diff` rule depends on `start_commit` / `end_commit` being populated by the planner.
- **Closed 12-kind enum** (`internal/domain/kind.go:18–47`) — Schema-2 validators reference `domain.IsValidKind`.

## Sequencing Diagram (Text DAG)

```
F.7.17 Schema-1 (per-binding command/env/cli_kind — F.7.17 planner)
       │
       ▼
F.7.18.1 — Schema-2: Context sub-struct + validators on AgentBinding
       │
       ▼
F.7.18.2 — Schema-3: [tillsyn] top-level globals + validators
       │
       ├─────────────────────────────────────┐
       ▼                                     ▼
F.7.18.3 — Aggregator engine                F.7.18.6 — spawn_history doc-comment
       (internal/app/dispatcher/context/)   (extends F.7.9 acceptance — co-routed)
       │
       ▼
F.7.18.4 — Greedy-fit cap + two-axis wall-clock
       │
       ▼
F.7.18.5 — Default-template seeds in default.toml
       (cross-plan blocker on F.7.16 — both edit default.toml)
```

Six droplets total. F.7.18.6 (doc-comment) is parallelizable with F.7.18.3 because they touch independent surfaces (engine package vs `internal/domain/action_item.go` field doc).

---

## Per-Droplet Decomposition

### F.7.18.1 — Schema-2: `Context` sub-struct on `AgentBinding` + validators

**Goal:** add a closed `Context ContextRules` sub-struct on `templates.AgentBinding` so adopters can declare `[agent_bindings.<kind>.context]` blocks; bake field-shape + closed-enum-membership validators into `templates.Load` so corrupt blocks fail at load time, not at spawn time.

**Builder model:** opus.

**Hard prereqs:**
- F.7.17 Schema-1 droplet (per-binding `command`, `args_prefix`, `env`, `cli_kind` fields on same struct in same file). Cross-plan `blocked_by`.

**Files to edit/create:**
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/templates/schema.go` — extend `AgentBinding` with `Context ContextRules` field; declare new `ContextRules` struct + `ContextDelivery` closed-enum string type.
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/templates/load.go` — extend the existing validator chain after `validateGateKinds` with new `validateAgentBindingContext` running per binding; declare new sentinel error `ErrInvalidContextRules`.
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/templates/schema_test.go` (or new `context_rules_test.go`) — unit tests for happy-path decode + every reject path.
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/templates/load_test.go` — full-template-load integration tests covering planner-`descendants_by_kind` happy-path (NO schema rule against it).

**Packages locked:** `internal/templates`.

**Acceptance criteria:**

- [ ] New `ContextRules` struct declared in `schema.go` with these fields and explicit TOML tags (per A3.c — strict-decode rejects unknown keys ONLY when every field carries a TOML tag):
  - `Parent bool` — TOML tag `parent`.
  - `ParentGitDiff bool` — TOML tag `parent_git_diff`.
  - `SiblingsByKind []domain.Kind` — TOML tag `siblings_by_kind`.
  - `AncestorsByKind []domain.Kind` — TOML tag `ancestors_by_kind`.
  - `DescendantsByKind []domain.Kind` — TOML tag `descendants_by_kind`.
  - `Delivery ContextDelivery` — TOML tag `delivery`.
  - `MaxChars int` — TOML tag `max_chars`.
  - `MaxRuleDuration Duration` — TOML tag `max_rule_duration`.
- [ ] New `ContextDelivery` closed-enum string type with constants `ContextDeliveryInline = "inline"` and `ContextDeliveryFile = "file"`. Helper `IsValidContextDelivery(ContextDelivery) bool` mirrors the `IsValidGateKind` pattern.
- [ ] `AgentBinding` extended with `Context ContextRules` field, TOML tag `context`. Existing fields (`AgentName`, `Model`, etc.) untouched.
- [ ] New validator `validateAgentBindingContext` invoked from `Load` after `validateGateKinds`. Iterates `tpl.AgentBindings` per binding; for each `binding.Context`:
  - If `Delivery` is empty AND no other context field is set, treat the whole `Context` block as absent (zero-value path — no validation runs). Implements the SKETCH:180 "FLEXIBLE not REQUIRED" framing.
  - If any context field is set, `Delivery` MUST be `"inline"` or `"file"` — reject empty + reject any other value via `IsValidContextDelivery`.
  - If `MaxChars` is set (> 0) it MUST be positive (caught by struct typing — int can't be negative without explicit `< 0` check; spec MUST add the `< 0` reject).
  - If `MaxChars` is zero AND any other context field is set, default-substitution happens at engine-time (NOT at validation-time) — `MaxChars == 0` is legal in the schema and means "use the bundle-global default."
  - If `MaxRuleDuration` is set (`> 0`) it MUST be positive; zero means "use bundle-global default" (engine-time substitution).
  - Every member of `SiblingsByKind`, `AncestorsByKind`, `DescendantsByKind` MUST be a member of the closed 12-kind enum (`domain.IsValidKind`). Empty slices are legal.
- [ ] New sentinel `ErrInvalidContextRules` declared in `load.go`; all `validateAgentBindingContext` errors wrap it via `fmt.Errorf("%w: ...")`.
- [ ] Unit test: TOML payload with `[agent_bindings.build.context] bogus_field = true` (any unknown key) MUST fail load with an error wrapping `ErrUnknownTemplateKey`. Proves closed-struct strict-decode actually fires for the new sub-struct.
- [ ] Unit test: planner binding with `descendants_by_kind = ["build"]` MUST load clean (no schema rule against it — A6.λ test).
- [ ] Unit test: every kind in the closed 12-value enum is accepted in `siblings_by_kind` / `ancestors_by_kind` / `descendants_by_kind`; an unknown kind (`["bogus-kind"]`) is rejected with `ErrUnknownKindReference` (NOT `ErrInvalidContextRules` — kinds use the existing sentinel for consistency).
- [ ] Unit test: `delivery = "stream"` (not in closed two-value enum) is rejected with `ErrInvalidContextRules`.
- [ ] Unit test: `delivery = ""` with all other context fields zero — load passes (absent block).
- [ ] Unit test: round-trip a fully-populated `[context]` block through `templates.Load` and assert every decoded field matches the TOML literal byte-for-byte.

**Test scenarios (happy + edge):**

- Happy path: a bounded-mode build binding with `parent = true`, `parent_git_diff = true`, `ancestors_by_kind = ["plan"]`, `delivery = "file"`, `max_chars = 50000`, `max_rule_duration = "500ms"` loads clean.
- Agentic path: a build binding with NO `[context]` block at all (the FLEXIBLE-not-REQUIRED case) loads clean and `binding.Context` is the zero value.
- Planner-descendants path (A6.λ cross-check): a `kind=plan` binding with `descendants_by_kind = ["build", "plan"]` loads clean — proves NO schema rule against planner-descendants.
- Unknown-key path (A3.c cross-check): `[agent_bindings.build.context] foobar = "yes"` MUST fail with `ErrUnknownTemplateKey`.
- Closed-enum-fail path: `delivery = "stream"` MUST fail with `ErrInvalidContextRules`.
- Closed-enum-fail path: `siblings_by_kind = ["bulid"]` (transposed letters in `build`) MUST fail with `ErrUnknownKindReference`.
- Negative-int path: TOML decoder accepts negative ints into `int`; the validator MUST reject `max_chars = -1` with `ErrInvalidContextRules`.

**Falsification mitigations to bake in:**

- A3.c (TOML-tag requirement): every field carries an explicit TOML tag; the unknown-key rejection unit test proves it fires.
- A6.λ (planner-descendants flexibility): unit test with `descendants_by_kind` on `kind=plan` MUST pass.
- A3.a (Tillsyn struct anchor risk): NOT applicable here — Schema-3 owns the `Tillsyn` struct. F.7.18.1 stays focused on `AgentBinding.Context`.

**Verification gates:** `mage check` + `mage ci` + per-droplet `build-qa-proof` + `build-qa-falsification` twins.

**Out of scope:**
- Top-level `[tillsyn]` table — that's F.7.18.2.
- Aggregator engine — that's F.7.18.3.
- Default-template seed `[context]` blocks — that's F.7.18.5.
- Cross-cap warning (`max_chars` per rule should not exceed `max_context_bundle_chars`) — bake into F.7.18.2 because it requires Schema-3's `MaxContextBundleChars` field to exist.

---

### F.7.18.2 — Schema-3: `[tillsyn]` top-level globals + validators

**Goal:** add a NEW top-level `Tillsyn Tillsyn` field on `templates.Template` carrying `MaxContextBundleChars int` + `MaxAggregatorDuration Duration`. Without this droplet, `[tillsyn]` keys in TOML are rejected by strict-decode (`load.go:88–95`). Validators reject zero or negative values on either field (when set).

**Builder model:** opus.

**Hard prereqs:**
- F.7.18.1 (Schema-2) MUST land first. Same file (`schema.go`), same package, strict ordering per SKETCH:238–243.

**Files to edit/create:**
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/templates/schema.go` — declare new `Tillsyn` struct; add `Tillsyn Tillsyn` field on `Template`.
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/templates/load.go` — extend validator chain with `validateTillsynGlobals`; cross-cap warning hook for `binding.Context.MaxChars > tpl.Tillsyn.MaxContextBundleChars` (warn-only — log via package-level structured logger; doesn't fail load).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/templates/schema_test.go` (or new `tillsyn_globals_test.go`) — unit tests covering happy + reject paths.
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/templates/load_test.go` — full-template-load integration test asserting `[tillsyn]` block parses + validates.

**Packages locked:** `internal/templates`.

**Acceptance criteria:**

- [ ] New `Tillsyn` struct declared in `schema.go` with these fields and explicit TOML tags:
  - `MaxContextBundleChars int` — TOML tag `max_context_bundle_chars`.
  - `MaxAggregatorDuration Duration` — TOML tag `max_aggregator_duration`.
- [ ] `Template` struct extended with new field `Tillsyn Tillsyn` — TOML tag `tillsyn`. Doc-comment cites SKETCH:241 + falsification A3.a as the explicit anchor that this struct is REQUIRED before any `[tillsyn]` TOML keys parse cleanly.
- [ ] New validator `validateTillsynGlobals` invoked from `Load` after `validateAgentBindingContext` (so it runs after F.7.18.1's validator). Rules:
  - `MaxContextBundleChars` MAY be zero (means "engine uses 200_000 default"). If non-zero, MUST be positive (`> 0`); reject negatives with new sentinel `ErrInvalidTillsynGlobals`.
  - `MaxAggregatorDuration` MAY be zero (means "engine uses 2s default"). If non-zero, MUST be positive (`time.Duration(d) > 0`); reject negatives with `ErrInvalidTillsynGlobals`.
- [ ] New sentinel `ErrInvalidTillsynGlobals` declared in `load.go` with godoc citing the SKETCH-locked default-substitution semantics.
- [ ] Cross-cap warning (per planner review §4.3 + SKETCH:216): if `tpl.Tillsyn.MaxContextBundleChars > 0` AND any `binding.Context.MaxChars > tpl.Tillsyn.MaxContextBundleChars`, emit a warn-only structured log line (does NOT fail load — adopters MAY want a single rule consuming the entire bundle, just unusual). Use the project's existing structured logger pattern from `internal/templates`.
- [ ] Unit test (A3.c cross-check, second instance): TOML `[tillsyn] bogus_global = 7` MUST fail load with `ErrUnknownTemplateKey`. Proves closed-struct unknown-key rejection actually fires for the new `Tillsyn` struct.
- [ ] Unit test: a template WITHOUT a `[tillsyn]` block at all loads clean; `tpl.Tillsyn` is the zero value (`MaxContextBundleChars == 0`, `MaxAggregatorDuration == 0`).
- [ ] Unit test: `[tillsyn] max_context_bundle_chars = -1` MUST fail load with `ErrInvalidTillsynGlobals`.
- [ ] Unit test: `[tillsyn] max_aggregator_duration = "-500ms"` MUST fail load with `ErrInvalidTillsynGlobals`.
- [ ] Unit test: warn-only cross-cap path: a binding with `max_chars = 999999` + `[tillsyn] max_context_bundle_chars = 200000` LOADS (does NOT fail) but emits a warn-line — assertion captures the log via the project's existing log-capture test pattern, OR assertion is a doc-comment promise + manual eyeball at QA time (flagged as Q-item if no log-capture pattern exists).

**Test scenarios (happy + edge):**

- Happy path: `[tillsyn] max_context_bundle_chars = 200000` + `max_aggregator_duration = "2s"` loads clean.
- Default-substitution path: no `[tillsyn]` block; template loads clean; engine substitutes defaults at `Resolve`-time.
- Unknown-key path (A3.c): `[tillsyn] foo = "bar"` MUST fail with `ErrUnknownTemplateKey`.
- Negative-int path: `max_context_bundle_chars = -100` MUST fail.
- Negative-duration path: `max_aggregator_duration = "-1s"` MUST fail.
- Cross-cap warn path: rule `max_chars` exceeds bundle cap — load passes, warn-line emitted.

**Falsification mitigations to bake in:**

- A3.a (Tillsyn struct as separate droplet): explicit. F.7.18.2's whole purpose is the `Tillsyn` struct + `Template.Tillsyn` field.
- A3.c (closed-struct unknown-key rejection): unit test proves it.
- A7.b (default-substitution for two-axis timeouts): zero values are legal; engine substitutes defaults.

**Verification gates:** `mage check` + `mage ci` + per-droplet QA twins.

**Out of scope:**
- The default values themselves (200000 chars / 2s) are SUBSTITUTED at engine-time, NOT default-stamped at decode-time. Engine-time substitution lives in F.7.18.4.
- Default-template `[tillsyn]` block in `default.toml` lands in F.7.18.5.

---

### F.7.18.3 — Aggregator engine: `internal/app/dispatcher/context/` package + `Resolve()` skeleton + per-rule renderers + ports

**Goal:** create the new `internal/app/dispatcher/context/` package; declare the `Resolve(ctx, binding, item, repo) (Bundle, error)` pure-function entry point; declare the `ActionItemReader` and `GitDiffReader` ports the engine consumes; implement per-rule renderers for `parent`, `parent_git_diff`, `siblings_by_kind`, `ancestors_by_kind`, `descendants_by_kind`. Per-binding scope only — the engine consults `binding.Context` for the spawning kind and does NOT iterate other bindings in the catalog.

This droplet ships the engine's structural skeleton + per-rule logic. F.7.18.4 layers the cap algorithm + wall-clock timeouts on top. Splitting these two preserves reviewable droplet size: this droplet ~600 LOC of straight Go (rule renderers + ports + tests); F.7.18.4 ~300 LOC of the cap-and-timeout wrapper + heavy tests.

**Builder model:** opus.

**Hard prereqs:**
- F.7.18.1 (Schema-2) MUST land — engine reads `binding.Context.*`.
- F.7.18.2 (Schema-3) MUST land — engine reads `tpl.Tillsyn.*` for default substitution. (Even though F.7.18.4 wraps the cap algorithm, the engine's `Bundle` shape is sized by the resolved global cap, so the Tillsyn struct must exist before this droplet's tests run.)

**Files to edit/create:**
- NEW `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/context/doc.go` — package-level doc-comment citing SKETCH:182–217 + planner review §4.1 + this plan.
- NEW `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/context/bundle.go` — declare `Bundle` value-type (`{RenderedInline string; Files map[string][]byte; Markers []string}`).
- NEW `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/context/ports.go` — declare narrow ports `ActionItemReader` (single method `GetActionItem(ctx, id string) (domain.ActionItem, error)`) + `GitDiffReader` (single method `GetGitDiff(ctx, repoRoot, startCommit, endCommit string) ([]byte, error)`).
- NEW `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/context/resolve.go` — `Resolve(ctx, args ResolveArgs) (Bundle, error)` entry point + per-rule renderer dispatch.
- NEW `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/context/render_parent.go` — `parent` rule renderer.
- NEW `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/context/render_git_diff.go` — `parent_git_diff` rule renderer.
- NEW `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/context/render_kindwalk.go` — shared renderer for `siblings_by_kind` / `ancestors_by_kind` / `descendants_by_kind`.
- NEW `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/context/resolve_test.go` — table-driven tests covering each rule type independently + per-binding scoping.
- NEW `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/context/ports_test.go` — port stub helpers used by other tests.

**Packages locked:** `internal/app/dispatcher/context` (NEW package; no existing-package conflict).

**Acceptance criteria:**

- [ ] New package `internal/app/dispatcher/context/` exists with `doc.go` package-overview comment naming Drop 4c F.7.18, citing SKETCH lines 182–217 and locking the FLEXIBLE-not-REQUIRED contract.
- [ ] `Bundle` struct declared with three exported fields:
  - `RenderedInline string` — concatenated rendered context destined for `system-append.md` when any rule has `delivery = "inline"`.
  - `Files map[string][]byte` — file-mode rendered context destined for `<bundle>/context/<filename>.md` when rule has `delivery = "file"`. Filename convention: `<rule_name>.md` (e.g. `parent.md`, `parent_git_diff.diff`, `ancestors_plan.md`, `siblings_build-qa-proof.md`). Filename is per-rule per-bundle unique; collisions resolved by appending an index suffix (flagged as Q4).
  - `Markers []string` — diagnostic markers (`[truncated: ...]`, `[skipped: ...]`, `[rule X timed out: ...]`) for the `RenderedInline` to embed AND for the audit log.
- [ ] `ActionItemReader` port declared as a one-method interface taking `ctx context.Context, id string` and returning `(domain.ActionItem, error)`. Production binding adapter lives in dispatcher root package (parallel to `service_adapter.go:23-33` pattern).
- [ ] `GitDiffReader` port declared as a one-method interface taking `ctx context.Context, repoRoot string, startCommit string, endCommit string` and returning `([]byte, error)`. Production implementation: shells out to `git -C <repoRoot> diff <start>..<end>` via `os/exec`. The implementation lives inside this droplet's package OR in an adjacent file under the dispatcher root — final location flagged as Q2 (depends on where the dispatcher's existing exec-shellout helpers live).
- [ ] `Resolve(ctx, args ResolveArgs) (Bundle, error)` entry point declared. `ResolveArgs` carries `Binding templates.AgentBinding`, `Item domain.ActionItem`, `Tillsyn templates.Tillsyn` (for default substitution), `ItemReader ActionItemReader`, `DiffReader GitDiffReader`, `RepoRoot string`. Returns the assembled `Bundle`.
- [ ] Per-binding scope: `Resolve` reads ONLY `args.Binding.Context`. It does NOT receive the full `templates.Template` or `templates.KindCatalog`. Acceptance test asserts `Resolve` works correctly when the catalog has 11 other bindings each with weird `[context]` blocks — `Resolve` for `kind=build` ignores them all.
- [ ] Per-rule renderers for `parent`, `parent_git_diff`, `siblings_by_kind`, `ancestors_by_kind`, `descendants_by_kind`. Each renderer:
  - Takes its specific input subset (`ItemReader` for kind-walks; `DiffReader` + commit fields for git-diff).
  - Returns `(rendered []byte, error)`.
  - Truncates internally per `args.Binding.Context.MaxChars` (engine-substitutes default 50000 if zero) WITH a `[truncated to <N> chars; full content at <bundle>/context/<rule>.full]` marker. **Note:** F.7.18.4 layers the bundle-cap + timeouts on top. This droplet's rule renderers handle ONLY per-rule truncation; bundle-cap skipping is F.7.18.4.
  - Writes the full content to `<bundle>/context/<rule>.full` for the agent to `Read` if needed (filename pattern same as Bundle.Files entries; flagged as Q4).
- [ ] Default-substitution rules (engine-time, NOT validation-time):
  - `binding.Context.MaxChars == 0` → use 50000.
  - `binding.Context.MaxRuleDuration == 0` → use 500ms (note: F.7.18.4 actually applies this; this droplet's renderers don't enforce timeouts directly — they accept a `ctx` and respect cancellation, F.7.18.4 wires the timeout `ctx`).
  - `tillsyn.MaxContextBundleChars == 0` → use 200000 (F.7.18.4 applies).
  - `tillsyn.MaxAggregatorDuration == 0` → use 2s (F.7.18.4 applies).
- [ ] Tests cover each rule type independently with stub `ActionItemReader` + stub `GitDiffReader`:
  - `parent` rule: stub reader returns a parent action-item with rich Description; renderer output contains the Description verbatim (within `MaxChars`).
  - `parent_git_diff` rule: stub diff reader returns a 200-line diff; renderer output contains the diff, prefixed with `start_commit..end_commit` summary line.
  - `siblings_by_kind` rule: stub reader returns siblings filtered by `Kind` matching the rule's kind list. Latest-round-only semantics: assert that if siblings have superseded predecessors, ONLY the latest is included. (Latest-round semantic implementation flagged as Q3 — what defines "round" pre-Drop-5.)
  - `ancestors_by_kind` rule: stub reader builds a chain of ancestors; renderer walks UP and captures the FIRST ancestor whose `Kind` matches the rule's first-listed entry (semantics: first-match per SKETCH:188).
  - `descendants_by_kind` rule: stub reader builds a tree; renderer walks DOWN and captures every direct + transitive descendant whose `Kind` matches.
- [ ] Per-binding scoping test (A-α from Section 0 reasoning): build a `templates.KindCatalog` with all 12 bindings, each with an absurd `[context]` block; `Resolve` for `kind=build` returns a `Bundle` whose contents reflect ONLY the build binding's context — siblings / ancestors / descendants from the build perspective, not from any other kind's binding.
- [ ] Engine assumes pre-validated input (per planner review P8). Tests do NOT exercise validator paths — those are F.7.18.1 / F.7.18.2 territory. If a corrupted binding survives template-load (test-only mutation), engine MAY behave undefined; document this in the package doc-comment.

**Test scenarios (happy + edge):**

- Happy path: bounded build binding + populated parent + 50-line git diff + ancestor `kind=plan` + 2 sibling QA twins → `Resolve` returns a `Bundle` with `RenderedInline` empty (delivery=file) and `Files` containing 4 entries.
- Inline-mode path: same binding but `delivery = "inline"` → `Resolve` returns a `Bundle` with `RenderedInline` populated and `Files` empty.
- Empty `[context]` (FLEXIBLE-not-REQUIRED): `binding.Context` is zero-value → `Resolve` returns an empty `Bundle` (zero `RenderedInline`, empty `Files`, empty `Markers`).
- Per-binding-scope falsification cross-check: catalog has `[agent_bindings.commit.context] descendants_by_kind = ["plan"]` (absurd); `Resolve` for `kind=build` does NOT include any descendants from the commit binding's perspective.
- Truncation marker path: rule output > `MaxChars`; `Bundle.Markers` contains the `[truncated to <N> chars; full content at <bundle>/context/<rule>.full]` line; the truncated rendered content + the full file output are both produced.

**Falsification mitigations to bake in:**

- Per-binding scope (Section 0 attack A-α): explicit test.
- Default-substitution semantics (A7.b): documented + tested with zero-value bindings.
- Engine assumes pre-validated input (P8): documented in package doc-comment.

**Verification gates:** `mage check` + `mage ci` + per-droplet QA twins.

**Out of scope:**
- Greedy-fit bundle-cap algorithm — F.7.18.4.
- Per-rule + per-bundle wall-clock timeouts (the engine's `Resolve` accepts `ctx context.Context` and respects cancellation, but the timeout `ctx` is wired in F.7.18.4).
- Spawn pipeline integration (the dispatcher's `BuildSpawnCommand` calling `Resolve` and writing the `Bundle` into `<bundle>/context/` + appending to `system-append.md`) — F.7.18.4 OR a separate F.7-CORE droplet (flagged as Q3 if the spawn pipeline droplet lives in F.7-CORE).
- Default-template seeds — F.7.18.5.
- Round-history aggregation — DEFERRED entirely.

---

### F.7.18.4 — Greedy-fit bundle-cap algorithm + per-rule + per-bundle wall-clock caps

**Goal:** layer the greedy-fit algorithm + two-axis wall-clock timeouts onto the F.7.18.3 engine. Iterate rules in TOML declaration order; each rule independently enforces `max_rule_duration` via `context.WithTimeout` (default 500ms); rules that bust `max_context_bundle_chars` are SKIPPED with marker AND CONTINUE; outer `context.WithTimeout(max_aggregator_duration)` (default 2s) is hard ceiling.

**Builder model:** opus.

**Hard prereqs:**
- F.7.18.3 (engine skeleton + rule renderers) MUST land first.

**Files to edit/create:**
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/context/resolve.go` — extend `Resolve` with the cap-and-timeout wrapper.
- NEW `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/context/cap.go` — greedy-fit cap algorithm helper + marker-emission helper.
- NEW `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/context/timeout.go` — per-rule + per-bundle timeout wrapper.
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/context/resolve_test.go` — extend with greedy-fit + timeout tests.
- NEW `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/context/cap_test.go` — focused greedy-fit unit tests.
- NEW `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/context/timeout_test.go` — focused timeout unit tests.

**Packages locked:** `internal/app/dispatcher/context` — same package as F.7.18.3 so explicit `blocked_by` to F.7.18.3 (sequential, NOT parallel).

**Acceptance criteria:**

- [ ] Rule iteration order is **TOML declaration order**, NOT a hardcoded priority order. Per planner review P9 + P10: declared order is the source of priority. Test asserts that if a binding declares rules in `[parent, parent_git_diff, ancestors_by_kind]` order in the TOML, the engine renders them in that order. **Note on Go map vs slice:** the schema decodes `Context` into a struct whose fields are NOT a slice — declaration order is FIXED by the field order on the `ContextRules` struct (`Parent` → `ParentGitDiff` → `SiblingsByKind` → `AncestorsByKind` → `DescendantsByKind`). This satisfies the "TOML declaration order" requirement because every rule type has a fixed canonical position; adopters cannot reorder rules within a binding. If future iteration needs adopter-controlled reordering, the schema would need a `[[context.rules]]` array-of-tables refactor — flagged as out-of-scope refinement.
- [ ] Greedy-fit cap algorithm: iterate rules in canonical struct-field order; for each rule:
  - Render the rule via F.7.18.3's per-rule renderer.
  - Measure rendered byte count `rule_size`.
  - If `cumulative + rule_size <= cap`, INCLUDE it: `cumulative += rule_size`, append to `Bundle.RenderedInline` (delivery=inline) or `Bundle.Files` (delivery=file).
  - Else, SKIP it: emit marker `[skipped: <rule_name> (would have added <N> chars; bundle remaining = <M>)]` into `Bundle.Markers`. Continue to next rule.
  - Subsequent rules that fit STILL LAND.
- [ ] Per-rule wall-clock timeout: each rule's render runs under `context.WithTimeout(parent_ctx, max_rule_duration)`. If timeout fires, partial rendered output is DISCARDED (NOT partially included), marker `[rule <name> timed out after <duration>; partial output discarded]` emitted, engine moves to next rule.
- [ ] Per-bundle wall-clock timeout: outer `context.WithTimeout(parent_ctx, max_aggregator_duration)` wraps the entire rule iteration. If outer timeout fires before all rules complete, partial bundle is returned with marker `[aggregator timed out after <duration>; rules pending: <list>]`.
- [ ] Outer-cancel propagates to inner: when the per-bundle timeout fires while a per-rule timeout is in-flight, the inner rule's render is cancelled (via the parent ctx); partial-rule output is DISCARDED. Test asserts this propagation behavior.
- [ ] Default-substitution at engine entry (per F.7.18.3 acceptance criteria — actually applied here):
  - `binding.Context.MaxChars == 0` → 50000 (per-rule cap).
  - `binding.Context.MaxRuleDuration == 0` → 500ms.
  - `tillsyn.MaxContextBundleChars == 0` → 200000.
  - `tillsyn.MaxAggregatorDuration == 0` → 2s.
- [ ] Greedy-fit test: cheap-1 (10KB) + busting-2 (220KB after per-rule trunc) + cheap-3 (50KB), bundle cap 200KB → cheap-1 lands, busting-2 SKIPPED with marker, cheap-3 lands. `Bundle.RenderedInline` size ~60KB; `Bundle.Markers` contains exactly one skip marker. (Direct cross-check of A6.b mitigation.)
- [ ] Per-rule timeout test: a stub `GitDiffReader` that blocks 1s on a `parent_git_diff` call; `max_rule_duration = "200ms"`; `max_aggregator_duration = "2s"` → `parent_git_diff` rule produces a `[rule parent_git_diff timed out after 200ms; partial output discarded]` marker, subsequent rules (`ancestors_by_kind`) STILL run.
- [ ] Per-bundle timeout test: 5 rules, each takes 600ms (configured stub readers); `max_rule_duration = "1s"` (so per-rule does NOT fire); `max_aggregator_duration = "2s"` → only ~3 rules complete; `Bundle.Markers` contains `[aggregator timed out after 2s; rules pending: <list of un-completed>]`.
- [ ] Outer-cancel-propagates test: per-bundle 500ms; per-rule 2s; rule blocks 1s → outer fires at 500ms, inner cancellation propagates, partial-rule output DISCARDED, marker emitted.

**Test scenarios (happy + edge):**

- Greedy-fit happy path: all rules fit under cap; no markers emitted.
- Greedy-fit skip path (A6.b cross-check): cheap + busting + cheap → cheap-cheap landed, single skip marker.
- Per-rule timeout edge: slow `parent_git_diff` on a 10000-line diff → 500ms timeout fires, rule discarded, others land.
- Per-bundle timeout edge: pathological tree with 20 deep ancestors → 2s outer fires, partial bundle.
- Combined-edge: per-rule fires for rule N AND per-bundle fires while rule N+5 is running → both markers emitted, no partial-output included.
- Default-substitution path: zero-value bindings + zero-value Tillsyn → engine uses defaults, all caps apply correctly.

**Falsification mitigations to bake in:**

- A6.b (greedy-fit chosen): direct test demonstrating cheap-bust-cheap pattern.
- A7.b (two-axis timeouts chosen): per-rule + per-bundle tests.
- A-η from Section 0 (outer-cancel propagation): explicit propagation test.
- A-ζ from Section 0 (render-then-measure ambiguity): pinned in acceptance criteria.

**Verification gates:** `mage check` + `mage ci` + per-droplet QA twins.

**Out of scope:**
- Spawn pipeline integration — flagged as Q3 (likely an F.7-CORE droplet, NOT this plan).
- Default-template seeds — F.7.18.5.

---

### F.7.18.5 — Default-template seeds in `default.toml`

**Goal:** populate `internal/templates/builtin/default.toml` with `[agent_bindings.<kind>.context]` blocks for the six in-scope bindings + a `[tillsyn]` block with the explicit defaults. Adopters who fork the default get the bounded-mode shape out of the box; adopters writing project-local templates pick their own shape.

**Builder model:** opus.

**Hard prereqs:**
- F.7.18.1 (Schema-2) MUST land — `[context]` blocks would be rejected by strict-decode otherwise.
- F.7.18.2 (Schema-3) MUST land — `[tillsyn]` block would be rejected otherwise.
- F.7.18.4 MUST land — engine must run greedy-fit + timeouts before adopters depend on these defaults.
- **Cross-plan blocker on F.7.16 (default `[gates.build]` expansion).** Both droplets edit `internal/templates/builtin/default.toml`. F.7.16 lives in F.7-CORE plan. Sibling-build rule: same `paths` entry → explicit `blocked_by` required. F.7.18.5 declares `blocked_by: F.7.16.<final-droplet-id>` (assigned by F.7-CORE planner).

**Files to edit/create:**
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/templates/builtin/default.toml` — add `[tillsyn]` block + six `[agent_bindings.<kind>.context]` blocks.
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/templates/embed_test.go` (or equivalent default-template-load test) — assert the default loads cleanly + each binding's `Context` decodes to expected values + `Tillsyn` decodes to expected values.

**Packages locked:** `internal/templates` (the embedded TOML lives in `internal/templates/builtin/`; the load tests run in `internal/templates`).

**Acceptance criteria:**

- [ ] `[tillsyn]` block added at the top of `default.toml` (or just above `[kinds.*]` — locate near the schema_version line for visibility):
  - `max_context_bundle_chars = 200000` (explicit default; matches engine substitution).
  - `max_aggregator_duration = "2s"` (explicit default).
- [ ] `[agent_bindings.build.context]` block:
  - `parent = true`.
  - `parent_git_diff = true`.
  - `ancestors_by_kind = ["plan"]`.
  - `delivery = "file"`.
  - `max_chars = 50000`.
  - `max_rule_duration = "500ms"`.
- [ ] `[agent_bindings.build-qa-proof.context]` block (per REV-4 — see REVISIONS POST-AUTHORING below; QA bindings DO NOT pre-stage git diff):
  - `parent = true`.
  - **NO `parent_git_diff`** — QA verifies independently by running `git diff` itself via Bash + Read tools. Pre-staging the diff would bias QA toward the builder's framing.
  - `siblings_by_kind = []` — empty placeholder; sibling-builder-worklog wiring depends on metadata plumbing not yet shipped (flagged as future refinement).
  - `ancestors_by_kind = ["plan"]`.
  - `delivery = "file"`.
  - `max_chars = 50000`.
  - `max_rule_duration = "500ms"`.
  - **Companion negative-assertion test**: builder MUST add a unit test asserting `tpl.AgentBindings[domain.KindBuildQAProof].Context.ParentGitDiff == false` after loading the default-go template. Catches future regressions where someone re-adds the field.
- [ ] `[agent_bindings.build-qa-falsification.context]` block: identical to `build-qa-proof` per REV-4 — including the **NO `parent_git_diff`** rule + companion negative-assertion test asserting `tpl.AgentBindings[domain.KindBuildQAFalsification].Context.ParentGitDiff == false`.
- [ ] `[agent_bindings.plan-qa-proof.context]` block:
  - `parent = true`.
  - `ancestors_by_kind = ["plan"]`.
  - `delivery = "file"`.
  - `max_chars = 50000`.
  - `max_rule_duration = "500ms"`.
- [ ] `[agent_bindings.plan-qa-falsification.context]` block: identical to `plan-qa-proof`.
- [ ] `[agent_bindings.plan.context]` block (the planner agent runs the `kind=plan` binding):
  - `parent = true`.
  - `ancestors_by_kind = ["plan"]`.
  - `delivery = "file"`.
  - `max_chars = 50000`.
  - `max_rule_duration = "500ms"`.
  - **NO `descendants_by_kind` entry in the default seed.** Default planners walk UP only. Adopters who want a fix-planner or tree-pruner planner declare `descendants_by_kind` themselves in their project template — and the schema (per F.7.18.1) does NOT reject it. (Cross-check of A-λ flexibility framing.)
- [ ] **NO `[context]` block** for `commit`, `research`, `closeout`, `refinement`, `discussion`, `human-verify` bindings. Per SKETCH:194 these bindings stay agentic-mode in the default. Adopters override per project.
- [ ] Default-template-load integration test asserts:
  - `default.toml` loads clean (no validator errors).
  - `tpl.Tillsyn.MaxContextBundleChars == 200000` and `tpl.Tillsyn.MaxAggregatorDuration == 2 * time.Second`.
  - `tpl.AgentBindings[domain.KindBuild].Context.Parent == true` (and equivalents for the other 5 in-scope bindings).
  - `tpl.AgentBindings[domain.KindCommit].Context` is the zero value (no `[context]` block in the default for commit).
- [ ] Comment block above the `[tillsyn]` table cites SKETCH:202–204 + F.7.18.4 as the engine that consumes these defaults; comment block above each `[agent_bindings.<kind>.context]` cites SKETCH:195–199 as the seed source + locks the FLEXIBLE-not-REQUIRED framing ("adopters who fork this template MAY remove these blocks for fully-agentic spawns").

**Test scenarios (happy + edge):**

- Default-load happy path: `templates.LoadDefaultTemplate()` (or whatever the existing helper is named) returns a `Template` whose `Tillsyn` + per-binding `Context` fields match the seeded values byte-for-byte.
- Agentic-mode preservation: `commit` / `research` / `closeout` / `refinement` / `discussion` / `human-verify` bindings have zero-value `Context` — agent receives no pre-staged context.
- Planner flexibility cross-check: `tpl.AgentBindings[domain.KindPlan].Context.DescendantsByKind` is nil (not set in default) but the schema accepts it being set → adopters can add it.
- Cross-plan-conflict cross-check: F.7.18.5 cannot land before F.7.16 — `blocked_by` is enforced by the dispatcher's lock manager. Manual orchestrator check at planning-time: confirm cross-plan blocker is wired.

**Falsification mitigations to bake in:**

- A-α from Section 0 (sibling lock conflict): explicit `blocked_by` declared.
- A-κ from Section 0 (scope creep): seed list bounded to the six SKETCH-named bindings; remaining six get NO seed.
- A-λ from Section 0 (planner-descendants flexibility): default seed has NO `descendants_by_kind` for `kind=plan` AND schema does NOT reject it; adopter customization works.

**Verification gates:** `mage check` + `mage ci` + per-droplet QA twins.

**Out of scope:**
- Adopter-template authoring documentation — F.7-CORE doc droplet.
- `default-go.toml` / `default-fe.toml` split (Theme F.2) — separate planning effort.

---

### F.7.18.6 — `metadata.spawn_history[]` audit-only doc-comment requirement

**Goal:** ensure `metadata.spawn_history[]` (introduced by F.7.9 in the F.7-CORE plan) carries an explicit doc-comment citing its audit-only role + linking to the round-history-deferred decision in F.7.18 commentary. Survives the SKETCH-to-PLAN handoff so future contributors know to add `prior_round_*` rules (NOT raw stream-json round-history) if the use case surfaces post-Drop-5.

**Builder model:** opus.

**Hard prereqs:** none — this droplet edits a doc-comment that lands in F.7.9. The doc-comment text is authored independently of F.7.9's struct/metadata wiring; the requirement here is that F.7.9 ACCEPT this doc-comment as part of its acceptance criteria.

**Coordination note:** F.7.18.6 is best routed as an ACCEPTANCE CRITERION addition on F.7.9 in the F.7-CORE plan (per A5.b mitigation: "Add a one-liner in F.7.9 droplet acceptance criteria"). If the F.7-CORE planner picks it up, F.7.18.6 retires; otherwise it ships as a tiny standalone droplet here. Surfaced as **Q1** below.

**Files to edit/create (if shipped standalone):**
- The file F.7.9 introduces `metadata.spawn_history[]` on (likely `internal/domain/action_item.go` or a new `metadata.go` adjacent file) — exact path determined by F.7-CORE planner's F.7.9 droplet.
- Doc-comment text: a Go block-comment on the field declaration citing this plan's commentary anchor (this droplet's `Goal` text + F.7.18 SKETCH:205, 217).

**Packages locked:** whichever F.7.9 picks (most likely `internal/domain`).

**Acceptance criteria (if shipped standalone — otherwise these become part of F.7.9's acceptance criteria):**

- [ ] `metadata.spawn_history[]` field carries a doc-comment with the exact required content:
  - **Audit-only role** stated explicitly: "this slice is an APPEND-ONLY AUDIT TRAIL of spawn lifecycle events. Consumers are ledger / dashboard renderers, NOT re-prompt aggregators."
  - **Link to F.7.18 round-history-deferred decision** by pointing readers at the workflow/drop_4c F.7.18 commentary OR its post-Drop-5 successor location.
  - **Future-contributor pointer**: "If a use case for raw stream-json round-history surfaces post-Drop-5, add it as a refinement-drop item with dedicated `prior_round_*` rules (`prior_round_worklog`, `prior_round_gate_output`, `prior_round_qa_findings`) targeting high-signal artifacts — NOT raw stream-json events."
- [ ] No code-behavior change. Doc-comment only.
- [ ] If shipped standalone (Q1 = "yes, F.7-CORE planner declined"): file an LSP-verifiable doc-comment + a `mage check` pass.

**Test scenarios:** N/A — doc-comment-only droplet; verification is `mage check` (godoc / lint passes).

**Falsification mitigations to bake in:**

- A5.b mitigation: doc-comment + future-contributor pointer survives the SKETCH-to-PLAN handoff.

**Verification gates:** `mage check` + `mage ci`. (No QA twins needed on a doc-comment droplet — flagged for plan-QA twins to validate; default is "yes, twins still apply because every build droplet gets twins per CLAUDE.md cascade structure.")

**Out of scope:**
- The `metadata.spawn_history[]` struct + appender wiring — F.7.9 territory.
- Round-history aggregator — DEFERRED entirely (F.7.18 architectural decision).

---

## Open Questions for Plan-QA Twins

**Q1 — Should F.7.18.6 (spawn_history doc-comment requirement) ship as a standalone droplet under THIS plan, or should the F.7-CORE planner absorb it into F.7.9's acceptance criteria?**

Default in this plan: **standalone droplet under F.7.18 (this plan)** because the requirement is F.7.18-derived (round-history-deferred decision lives in this sub-theme). F.7-CORE planner may prefer to absorb it into F.7.9 — both options yield the same end-state. Plan-QA twins should pick the lower-coordination-overhead option. If F.7-CORE absorbs, F.7.18.6 retires from this plan and the droplet count drops from 6 to 5.

**Q2 — Where does the production `GitDiffReader` adapter (the actual `git diff` shell-out) live?**

Two candidates:
- Inside the `internal/app/dispatcher/context/` package as a sibling file (`git_diff_reader.go`) — keeps engine-and-adapter co-located.
- In the dispatcher root package as a sibling to `service_adapter.go` — mirrors the existing pattern where the dispatcher package owns adapters wrapping `*os/exec` shell-outs.

Default in this plan: **dispatcher root package adapter**, since `os/exec` shell-outs already live there (gates' mage-shellouts) and the engine package stays free of `os/exec` for testability. Plan-QA twins should validate against the existing dispatcher package conventions.

**Q3 — How does the engine wire into the spawn pipeline (where does `Resolve(...)` get called from + where does `Bundle.Files` get written into `<bundle>/context/`)?**

This is THE seam between F.7.18 and F.7-CORE. Two candidates:
- F.7-CORE owns the spawn-pipeline-calls-aggregator wiring (a new droplet in their plan that depends on F.7.18.4 landing).
- F.7.18 owns it (a 7th droplet in this plan touching `internal/app/dispatcher/spawn.go`).

Default in this plan: **F.7-CORE owns it** because the spawn pipeline is F.7-CORE's primary surface (F.7.1 bundle layout, F.7.3 argv emission, F.7.4 stream parser). F.7.18 ships the engine + its API; F.7-CORE wires it. The `Resolve(...)` API is the contract surface. Plan-QA twins should confirm with the F.7-CORE planner that this seam is owned on their side.

**Q4 — File-naming convention for `Bundle.Files` entries written to `<bundle>/context/<filename>`?**

Default in this plan: per-rule deterministic name (`parent.md`, `parent_git_diff.diff`, `ancestors_<kind>.md`, `siblings_<kind>.md`, `descendants_<kind>.md`). Collision case: `siblings_by_kind = ["build", "plan"]` → two files `siblings_build.md` + `siblings_plan.md`. Truncation full-content sidecar: append `.full` suffix (`parent.md.full`). Plan-QA twins should validate the convention against any existing F.7-CORE bundle-layout conventions (F.7.1's bundle layout decisions may dictate something specific).

---

## References

- `workflow/drop_4c/SKETCH.md` lines 180–217 (F.7.18 spec) + lines 238–243 (three-schema-droplet sequencing).
- `workflow/drop_4c/4c_F7_EXT_PLANNER_REVIEW.md` §3 (flexibility framing), §4 (engine boundary + token budget composition), §7.2 (F.7.18 droplet decomposition seed), P6/P7/P8/P9/P10.
- `workflow/drop_4c/4c_F7_EXT_QA_FALSIFICATION_R2.md` A3 (Tillsyn struct as separate droplet + closed-struct unknown-key rejection), A5 (round-history dangling — REFUTED-WITH-NIT, doc-comment requirement), A6 (greedy-fit chosen), A7 (two-axis timeouts chosen).
- `internal/templates/schema.go` lines 127–199 (`Template` root struct), lines 285–332 (`AgentBinding` struct).
- `internal/templates/load.go` lines 80–95 (strict-decode chain), lines 174–191 (`validateMapKeys`), lines 119–164 (sentinel error patterns).
- `internal/domain/kind.go` lines 18–47 (closed 12-kind enum + `IsValidKind`).
- `internal/templates/builtin/default.toml` lines 377–536 (current `[agent_bindings.<kind>]` shape — F.7.18.5 extends).
- `internal/app/dispatcher/service_adapter.go` (existing reader-adapter pattern — F.7.18.3 mirrors).
- Project memory: `feedback_no_migration_logic_pre_mvp.md`, `feedback_no_closeout_md_pre_dogfood.md`, `feedback_opus_builders_pre_mvp.md`, `feedback_commit_style.md`.

## REVISIONS POST-AUTHORING (2026-05-05) — supersedes affected portions above

The dev approved architectural changes after this sub-plan was authored. **Where this section conflicts with text above, this section wins.** Builders read this section first.

### REV-1 — F.7.17 Schema-1 scope changed; `command` and `args_prefix` GONE

References at lines 26, 34, 68 to "per-binding `command`, `args_prefix`, `env`, `cli_kind`" are SUPERSEDED. F.7.17 Schema-1 (F.7.17.1) now ships ONLY `Env []string` + `CLIKind string` on `AgentBinding`. The `command` and `args_prefix` fields were dropped from the design 2026-05-05 — Tillsyn does NOT surface a binary-path or wrapper override; OS-level isolation is the adopter's responsibility (PATH-shadowed shim, container wrapping the entire Tillsyn binary, etc.).

Cross-plan `blocked_by F.7.17.1` for F.7.18.1 (Schema-2) is unchanged — Schema-2 still depends on Schema-1 landing first because both add fields to `AgentBinding` in the same file.

### REV-2 — L4 closed env baseline expanded

Per master PLAN.md §3 L4: closed env baseline expanded with proxy + TLS-cert vars (`HTTP_PROXY, HTTPS_PROXY, NO_PROXY` + lowercase variants, `SSL_CERT_FILE, SSL_CERT_DIR, CURL_CA_BUNDLE`). F.7.18.3 aggregator engine does NOT touch env (it reads action items + git diffs); no F.7.18 droplet is impacted directly. Documented here for completeness so this sub-plan's REVISIONS section mirrors the master's invariant.

### REV-4 — QA bindings DO NOT pre-stage `parent_git_diff` (independence rule)

The default-template seeds for `build-qa-proof` + `build-qa-falsification` MUST NOT declare `parent_git_diff = true`. QA agents run their own `git diff` via Bash + `Read` tools to verify the builder's claims independently. Pre-staging the diff would bias QA toward the builder's framing — independent verification is load-bearing for cascade-on-itself trustworthiness.

**Builder lens (gets diff pre-staged):** `[agent_bindings.build.context] parent_git_diff = true` — reduces redundant tool calls during implementation.

**QA lens (does NOT get diff pre-staged):** `[agent_bindings.build-qa-proof.context]` and `[agent_bindings.build-qa-falsification.context]` — diff omitted; QA tools (Bash `git diff`, `Read`, `Grep`) provide it on demand.

**Plan-qa-* bindings:** no diff needed regardless (they review planning artifacts, not code).

F.7.18.5 droplet acceptance is updated above (lines ~364-372) to reflect this. Builder-side and QA-side defaults differ deliberately — this is the rule, not a future revision.

### REV-3 — `Tillsyn` struct extension policy confirmed

F.7.18.2 owns the initial `Tillsyn` top-level struct declaration with two fields: `MaxContextBundleChars int` + `MaxAggregatorDuration Duration`. F.7-CORE F.7.1 + F.7.6 extend the struct with their named fields per master PLAN.md §5. F.7.18.2 acceptance criteria MUST include a unit test asserting strict-decode rejects an unknown key on the `Tillsyn` struct (so subsequent extenders inherit unknown-key rejection automatically per pelletier/go-toml v2 semantics).

---

## Hylla Feedback

`N/A — planning touched non-Go files only` (the SKETCH.md, planner-review MD, falsification-R2 MD; cited code symbols verified via direct `Read` on `schema.go`, `load.go`, `kind.go`, `default.toml`, `service_adapter.go`, `spawn.go` rather than Hylla — faster for one-shot file:line checks during plan authoring). No Hylla queries issued, no miss to report.
