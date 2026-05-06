# Drop 4c Theme F.7 Extension — Planner-Grade Review (F.7.17 + F.7.18)

**Reviewer:** go-planning-agent (read-only).
**Targets:** `workflow/drop_4c/SKETCH.md` lines 147-205 (F.7.17 CLI adapter seam + F.7.18 context aggregator + dependencies block) plus retrofit pressure on F.7.1-F.7.16.
**Date:** 2026-05-04.
**Mode:** Filesystem-MD review BEFORE full PLAN.md authoring. No SKETCH.md edits, no code edits — recommendations only.

**Verdict:** **PASS-WITH-NITS.** Both seams are architecturally sound, fit the existing F.7 design, and carry the dev's flexibility framing intact. Six concrete planning recommendations below should be folded into the eventual F.7 PLAN.md so droplet decomposition lands clean. None of the nits invalidates the SKETCH; all are tractable in PLAN authoring.

---

## 1. F.7.17 Adapter Seam Soundness

### 1.1 Three-method interface — does it absorb Codex without leaking claude-isms?

The proposed shape (SKETCH.md:150):

```
BuildCommand(ctx, BindingResolved, BundlePaths) (*exec.Cmd, error)
ParseStreamEvent(line []byte) (StreamEvent, error)
ExtractTerminalCost(StreamEvent) (cost float64, denials []ToolDenial, ok bool)
```

The three-method split is the right factoring **provided** `StreamEvent` and `BundlePaths` are adapter-neutral types defined in the dispatcher core package. Two risks the SKETCH does not yet pin down:

- **`StreamEvent` shape is implicitly claude-shaped.** Probe data in `project_drop_4c_spawn_architecture.md` §6 enumerates claude's `system/init`, `assistant`, `user`, `result` event vocabulary. Codex's `codex exec --json` event taxonomy (verified via dev's 2026-05-04 probe per SKETCH.md:157) is a different vocabulary. If `StreamEvent` is a Go struct with `Type string; Subtype string; Cost float64; PermissionDenials []ToolDenial; …` covering the claude union, Codex's adapter has to either lossily project codex events into claude shape OR carry both shapes via type-tag union. **Plan recommendation P1**: PLAN.md must spec `StreamEvent` as a *minimal cross-CLI canonical shape* — the small set of fields the dispatcher actually consumes (terminal? cost? denials? final-text?). Anything CLI-specific stays inside the adapter and is not surfaced upward. Same rule that keeps the seam claude-clean keeps it codex-clean.

- **`BundlePaths` may be over-specified for non-claude CLIs.** Bundle layout (SKETCH.md:125 + memory §2) is claude-specific: `plugin/.claude-plugin/plugin.json`, `plugin/agents/<name>.md`, `plugin/.mcp.json`, `plugin/settings.json`, `system-prompt.md`, `system-append.md`, `stream.jsonl`. Codex doesn't have a "plugin" abstraction; it reads `$CODEX_HOME/config.toml` profiles + AGENTS.md files. **Plan recommendation P2**: PLAN.md should make `BundlePaths` a thin handle (`Root string`, `StreamLog string`, `Manifest string`) and let each adapter materialize its CLI-specific subdirs (`plugin/` for claude, `codex_home/` for codex) inside `Root` via `BuildCommand`. The bundle materialization in F.7.1 then has TWO phases: cross-CLI (root, manifest, stream.jsonl) and CLI-specific (delegated to the adapter).

### 1.2 Will Cursor / Goose fit the same seam?

Cursor (CLI-mode) and Goose both stream JSON-Lines events with cost + final-text fields per their public docs. The three-method split is sufficient for any CLI that:

- Accepts a single text prompt + flags via argv.
- Emits stream-format events line by line on stdout.
- Carries terminal cost + denial info in some terminal event.

CLIs that **break** the seam: those that require interactive stdin (no headless mode); those that expect the prompt as a config file rather than a flag; those that don't emit cost. None of the four named candidates (claude, codex, cursor, goose) violate. **Plan recommendation P3**: PLAN.md should call out the three CLI-shape invariants explicitly so future-adapter authors have a "does my CLI fit?" checklist before opening a PR.

---

## 2. `command` / `args_prefix` / `env` Design

### 2.1 Coverage of interop cases

Three TOML fields cover:

- **Valv wrapper**: `command = "valv claude"`. Adopter installs Valv separately. Tillsyn never knows about Docker.
- **OAuth-profile-per-account-via-CLAUDE_CONFIG_DIR**: `env = ["CLAUDE_CONFIG_DIR"]` plus an outer-shell `direnv` / `chamber` / system-keychain helper that exports `CLAUDE_CONFIG_DIR` before `till dispatcher run`. Tillsyn forwards by name.
- **Env-only credential plumbing**: `env = ["TILLSYN_API_KEY", "ANTHROPIC_API_KEY"]`. `os.Getenv` at spawn; fail-loud on missing.
- **`args_prefix` for profile flags**: `args_prefix = ["--profile", "tillsyn-build"]` — slots into both `valv claude` and `codex --profile`.

**Sufficient.** No interop case I can construct requires a fourth field.

### 2.2 Schema validation: env rejecting `KEY=value` shapes

The SKETCH (line 155) says `env` rejects values containing `=`. **Necessary but not sufficient.** Three additional guardrails the PLAN.md should pin down:

- **Reject empty strings.** `env = [""]` should fail at template-load with a position-aware error.
- **Reject duplicate names.** `env = ["FOO", "FOO"]` is almost certainly an authoring mistake; rejecting it costs nothing.
- **Validate the env-var-name shape.** Posix-portable env var names are `[A-Za-z_][A-Za-z0-9_]*`. Rejecting names with whitespace, dashes, dots prevents a class of "I copy-pasted a TOML key here by accident" bugs. **Plan recommendation P4**: bake all three checks into the same `validateAgentBindingEnvNames` function inside `templates.Load`.

### 2.3 Fail-loud-on-missing semantics need pinning

SKETCH.md:154 says `env` is "list of env-var NAMES to forward to the spawned process via `os.Getenv` at spawn time. Fail-loud on missing required-env." The phrase "fail-loud" is under-specified for planning:

- Does missing-env fail at `till dispatcher run` invocation (early, dev sees clear error before any agent fires) or at the moment the adapter calls `BuildCommand` (later, after lock acquisition)?
- Does the failure surface as `metadata.outcome = "failure"` + `metadata.failure_reason = "missing required env: TILLSYN_API_KEY"` on the action item, or does the dispatcher fail before any state change?

**Plan recommendation P5**: PLAN.md should pin the failure surface as **early-pre-lock** (validate env presence in `BuildSpawnCommand` before lock acquisition) so a missing env doesn't poison a lock the action item never used.

---

## 3. F.7.18 Flexibility Framing

### 3.1 Does the SKETCH text correctly convey OPTIONAL not REQUIRED?

**Yes, with one quiet drift.** The SKETCH covers the flexibility framing in three places:

- Line 159: "**This is OPTIONAL — Tillsyn supports the pattern but does not require it.** Templates (or adopters) that want full live MCP querying inside the spawn just leave the `[agent_bindings.<kind>.context]` table absent. Both paths are first-class."
- Lines 181-184: "Why this is FLEXIBLE not REQUIRED" subsection with explicit adopter-choice framing.
- Line 184: "Default-go template picks the bounded path; default-generic template ships with empty `[context]` tables (omit)."

The dev's verbatim quote ("we don't expect, dictate that all agents need to only be able to call mcp to update their own node. In fact planner cant do that obviously. We just want templating to allow for that.") is honored.

**Quiet drift**: line 178 says "Planners CANNOT have descendants pre-staged (they create them); the aggregator does NOT walk a planner's children. Schema validation rejects `descendants_by_kind` on `kind=\"plan\"`." This is correct, but it reads as "planners can't use the aggregator for descendants" when the dev's intent is broader: **planners must be able to operate WITHOUT the aggregator at all.** The schema rejection only covers one specific field. **Plan recommendation P6**: PLAN.md should add an explicit assertion that planner agents never depend on `[context]` being present — the planner system-prompt-template assumes it has live MCP access for descendants regardless of whether ancestors/parent are pre-staged.

### 3.2 Will an adopter reading the SKETCH understand they can omit `[context]` entirely?

The "FLEXIBLE not REQUIRED" subsection (lines 181-184) is explicit. The default-template seeds (lines 173-178) are framed as "just defaults, projects override." An adopter following the schema spec at line 161 — `[agent_bindings.build.context]` "(NEW table, all fields optional)" — reads "all fields optional," and the surrounding prose makes the table itself optional too.

**One additional readability win**: the schema example at lines 162-171 shows a fully-populated `[context]` table. An adopter scanning quickly sees the populated example and might infer `[context]` is the canonical shape. **Plan recommendation P7**: in the eventual PLAN.md, the schema spec should show TWO examples side by side — one with `[context]` populated (bounded mode), one with `[context]` omitted entirely (agentic mode) — and label both as "first-class supported configurations." Cheap, prevents the most common mis-read.

---

## 4. Aggregator Engine Boundaries

### 4.1 Is `internal/app/dispatcher/context/` the right package?

**Yes.** The dispatcher already owns lifecycle (`spawn.go`), gates (`gate_*.go`), locks (`locks_*.go`), monitor (`monitor.go`). Context aggregation is a sibling concern that runs at spawn-prep time, between binding resolution and `BuildCommand`. Putting it in `internal/app/dispatcher/context/` keeps it co-located with the consumer.

**Counter-argument considered and rejected**: putting it in `internal/templates/context/` would tightly couple template loading with runtime data fetching (the aggregator needs an `ActionItemReader` + `GitDiffReader`). Templates package today is pure schema + bake; pulling in runtime ports breaks that cleanliness. **Stay in dispatcher.**

### 4.2 Where does planner-descendants-rule schema validation belong?

The SKETCH (line 185) puts it in `templates.Load` (validates "the planner descendants-rule + the cross-cap on `max_chars`; unknown keys rejected"). This is the right call **with one refinement**:

- **Field-shape validation** (types, unknown-key rejection, env-name-shape, max_chars positive int, `delivery` in `{"inline", "file"}`) belongs in `templates.Load`. Static, no runtime dependencies.
- **Cross-axis / cross-kind rules** (planner cannot have `descendants_by_kind`, `siblings_by_kind` entries reference valid kinds) also belong in `templates.Load` — they're closed-enum membership checks, no runtime dependencies.
- **Aggregator engine** in `internal/app/dispatcher/context/` does NOT re-validate. It assumes the binding has passed `templates.Load`. This is the same contract as `BuildSpawnCommand` defensively re-validating `AgentBinding` only because tests can construct corrupted bindings in-memory; production path is single-validation.

**Plan recommendation P8**: PLAN.md should call out the validation split explicitly so the F.7.18 droplet decomposition has a clear seam — "schema validators land in `templates/load.go` extension; engine in `dispatcher/context/` package; engine assumes pre-validated input."

### 4.3 Token budget composition — per-rule + total cap

The SKETCH covers two layers (line 180):

- **Per-rule `max_chars`**: default 50KB, individual rules truncate with marker.
- **Total cap `[tillsyn] max_context_bundle_chars = 200000`**: aggregator applies largest-rules-first, smaller-priority dropped with marker.

**Composition concern**: a single rule blowing the cap is handled by per-rule truncation. But "largest-rules-first with smaller dropped" begs the question — what does "priority" mean? Two reasonable orderings:

- **By declared rule order** (TOML preserves order). Simple, predictable, dev controls priority by where they put the rule in the TOML.
- **By rule type** (parent > ancestors > siblings > descendants > round_history). Opinionated, hard-codes a priority that may not match every adopter's wants.

The SKETCH does not pin which. **Plan recommendation P9**: PLAN.md should pick **declared rule order** (simpler, gives adopter explicit control, avoids tillsyn-opinionated priority hardcoded into the engine). Document it in the schema doc so adopters know "put your most-important context rule first."

A second composition issue: the "largest-rules-first" phrasing in SKETCH line 180 conflicts with declared-order priority. Re-read: "agent gets the largest rules first, smaller-priority rules dropped with marker." This sounds like "render by character count descending, drop the smallest until under cap" — that's NOT priority, it's greedy fitting. **Plan recommendation P10**: PLAN.md should clarify the algorithm. My recommendation: render in declared order until cumulative size hits the cap; drop subsequent rules with a single bundle-level `[truncated due to total bundle cap]` marker. Simpler than "largest first," gives adopters predictable behavior.

---

## 5. Drop 4d Sequencing

### 5.1 Realistic at ~7-10 droplets?

Possible items for Drop 4d codex adapter:

1. **`CLIKindCodex` constant + adapter scaffold** (1 droplet).
2. **`BuildCommand` for codex** — argv shape, `--profile`, `--ignore-user-config`, `--ignore-rules`, `--sandbox`, `--ephemeral`, prompt routing (1 droplet).
3. **`ParseStreamEvent` for codex `--json` taxonomy** — codex emits different event types than claude (1-2 droplets).
4. **`ExtractTerminalCost` for codex's terminal event** (1 droplet).
5. **Codex bundle materialization** — `$CODEX_HOME/config.toml` synthesis, AGENTS.md rendering (1-2 droplets).
6. **TOML schema test fixtures** — `cli_kind = "codex"` round-trips through `templates.Load` (1 droplet).
7. **Default-template seed for codex** — sample binding so adopter-fork projects can opt in (1 droplet).
8. **Documentation** — adapter authoring guide, codex-specific footguns (1 droplet).
9. **End-to-end smoke test** with a real codex install (1 droplet, conditional on dev's machine setup).

That's 9-11 droplets. **Plan recommendation P11**: 7-10 in the SKETCH is optimistic; budget 9-11 in PLAN.md. The slack absorbs the codex bundle materialization (item 5) being more work than expected.

### 5.2 Hard-prereq chain from Drop 4d → Drop 4c

The SKETCH says (line 204): "F.7.17 (CLI adapter seam) is internal-refactor-only inside Drop 4c — Drop 4c ships only the `claude` adapter. Drop 4d lands `codex`. The seam exists pre-Drop-4d so Drop 4d is purely additive."

For "purely additive" to hold, Drop 4c must:

1. **Land `CLIKind` enum + `CLIAdapter` interface in `internal/app/dispatcher/`.**
2. **Migrate the existing claude spawn pipeline TO the new seam** — meaning the claude path goes through `claude := claudeAdapter{}; claude.BuildCommand(...)` rather than a hardcoded function.
3. **Land the `BindingResolved` type the adapter consumes** — flat struct of resolved binding fields (model, max_budget_usd, max_turns, command, args_prefix, env, system_prompt_path, etc.).
4. **Land the cross-CLI canonical `StreamEvent` and `BundlePaths` types** (per P1, P2 above).

If any of these slip to Drop 4d, the "additive" framing breaks — Drop 4d would have to refactor Drop 4c's claude pipeline first. **Plan recommendation P12**: PLAN.md must list these four items as explicit Drop 4c F.7.17 sub-droplets so the seam exists end-to-end before Drop 4c merges. The current SKETCH lists three bullets (enum, interface, TOML overrides) — items 2-4 above are implied but not enumerated.

### 5.3 What does Drop 4d need from Drop 4c that the SKETCH doesn't yet guarantee?

Three items the SKETCH leaves implicit:

- **`BindingResolved` plumbing.** The adapter receives a fully-resolved binding (after CLI > MCP > TUI > TOML > absent priority cascade per memory §3 / SKETCH F.7.3). If Drop 4c's claude adapter takes shortcuts and reads from the raw `templates.AgentBinding` directly, Drop 4d has to go back and introduce `BindingResolved` first.
- **Per-adapter `manifest.json` schema.** The bundle's `manifest.json` (SKETCH F.7.1) records `claude_pid`, `started_at`, `paths`. Codex spawn would record `codex_pid`. The manifest schema needs a `cli_kind` field so the orphan-scan path (F.7.8) knows which CLI's PID-liveness rules to apply.
- **Per-adapter `permission_denials` shape.** F.7.5's TUI handshake parses `permission_denials[]` from claude's terminal event. Codex may use a different field name. The handshake-write path needs a uniform shape (`ToolDenial{ToolName, ToolInput}`) returned by `ExtractTerminalCost` — which is what the interface signature already specifies. Re-confirm in PLAN.md that the SQLite `permission_grants` table's `rule` column is CLI-agnostic so a grant authored against claude isn't accidentally re-rendered into a codex spawn's settings without translation.

**Plan recommendation P13**: PLAN.md must spec all three contracts explicitly in F.7.17 to make Drop 4d additive.

---

## 6. Cross-Section Consistency With F.7.1-F.7.16

### 6.1 Bundle layout (F.7.1) — claude-shape leaks into the package

`<bundle>/plugin/.claude-plugin/plugin.json` is claude-specific. With the adapter seam, the bundle root needs to be CLI-neutral and CLI-specific subdirs delegated to adapters (per P2). **Retro-edit needed**: F.7.1 in PLAN.md should describe the bundle layout in two phases — cross-CLI shell (`<bundle>/manifest.json`, `<bundle>/system-prompt.md`, `<bundle>/system-append.md`, `<bundle>/stream.jsonl`, `<bundle>/context/`) and CLI-specific subtree (`<bundle>/plugin/...` for claude, `<bundle>/codex_home/...` for codex). The SKETCH today bundles them flat as if claude's layout is the universal layout.

### 6.2 Headless argv (F.7.3) — argv shape moves into the adapter

F.7.3 today (SKETCH line 129) lists the claude argv recipe inline. With the adapter seam, the argv recipe is `claudeAdapter.BuildCommand`'s implementation detail. **Retro-edit needed**: F.7.3 should be re-framed as "the claude adapter's argv recipe" rather than the dispatcher's universal recipe. The dispatcher itself only knows it calls `adapter.BuildCommand(...)`. Functionally identical, but the framing change matters for readers who'll come along after Drop 4d and try to understand "where do I add codex argv?"

### 6.3 Stream-JSON monitor (F.7.4) — parser moves into the adapter

Same retro-edit as F.7.3. F.7.4's per-event-type parsing (system/init, assistant, user, result) is claude-specific; the adapter's `ParseStreamEvent` returns the canonical `StreamEvent` shape. **Retro-edit needed**: F.7.4 should be re-framed as "claude adapter's stream parser implementation; the dispatcher monitor is CLI-agnostic and consumes adapter-returned `StreamEvent` values."

### 6.4 Permission handshake (F.7.5) — handshake protocol stays cross-CLI

F.7.5's SQLite `permission_grants(project_id, kind, rule, granted_by, granted_at)` table is CLI-agnostic. The `rule` column stores the claude-pattern-syntax string today. With codex potentially using different rule syntax, the schema needs a CLI discriminator. **Retro-edit needed**: F.7.5 should add `cli_kind` to `permission_grants` so a grant authored against claude doesn't apply to a codex spawn (or vice versa). This is a minor schema addition but it must land in Drop 4c, not Drop 4d, because Drop 4c is the schema owner.

### 6.5 F.7.10 (drop hylla_artifact_ref) — unaffected by F.7.17/F.7.18

The hylla-removal lands cleanly regardless of adapter seam. No retro-edit needed.

### 6.6 F.7.12-F.7.16 (commit / push gates) — affected through commit-agent binding

The commit-agent (F.7.12) is itself a spawn that goes through the adapter seam. If commit-agent's binding has `cli_kind = "claude"` (default) or `cli_kind = "codex"` later, that's already covered. **No retro-edit needed**, but PLAN.md should explicitly note that commit-agent IS dispatched through the adapter seam — not a hardcoded path.

---

## 7. Plan-Level Gaps For PLAN.md Authoring

### 7.1 F.7.17 droplet decomposition seed

Six concrete droplets PLAN.md should land in order:

1. **`CLIKind` closed-enum + `CLIAdapter` interface scaffold** in `internal/app/dispatcher/`. Includes `BindingResolved`, `BundlePaths`, `StreamEvent`, `ToolDenial` cross-CLI types. Pure types, no behavior. Tests assert the enum membership.
2. **`claudeAdapter` struct implementing `CLIAdapter`** — moves the existing F.7.3 / F.7.4 logic into adapter methods. No behavior change visible to dispatcher callers. Tests assert byte-for-byte argv parity with pre-refactor for a fixed input.
3. **TOML schema widening for `cli_kind` + `command` + `args_prefix` + `env`** in `internal/templates/schema.go`. Includes `validateAgentBindingEnvNames` per P4. Tests cover happy path, env-name shape rejection, duplicate-name rejection, KEY=value rejection, empty-string rejection.
4. **Dispatcher wiring** — `BuildSpawnCommand` looks up `CLIKind` from binding, picks the adapter, calls `adapter.BuildCommand`. Default missing `cli_kind` resolves to `claude` for backward-compat. Tests cover both default and explicit-claude.
5. **`manifest.json` `cli_kind` field + orphan-scan adapter routing** (per §6.1 retro-edit). Tests cover claude-PID liveness check; codex path is empty-stub returning "not yet supported."
6. **`permission_grants` schema gets `cli_kind` column** (per §6.4 retro-edit). SQL migration NOT in pre-MVP rules (memory feedback_no_migration_logic_pre_mvp.md): dev fresh-DBs. Tests cover claude grant insert + read; codex insert path is wired but unused.

### 7.2 F.7.18 droplet decomposition seed

Five concrete droplets:

1. **TOML schema widening for `[agent_bindings.<kind>.context]`** in `internal/templates/schema.go`. Includes `parent`, `parent_git_diff`, `siblings_by_kind`, `ancestors_by_kind`, `descendants_by_kind`, `delivery`, `max_chars`, `include_round_history` (optional). Includes the `[tillsyn] max_context_bundle_chars` global. Includes planner-descendants-rule validation (P6) + cross-cap validation. Tests cover happy path, planner-descendants rejection, unknown-key rejection.
2. **Aggregator engine in `internal/app/dispatcher/context/`** — pure-function `Resolve(binding, item, repo) (Bundle, error)`. Reads parent / siblings / ancestors via existing `ActionItemReader` port; reads git diffs via new `GitDiffReader` port. Tests cover each rule type independently.
3. **Token budget enforcement** — per-rule `max_chars` truncation, total cap with declared-order priority (P9, P10). Tests cover cap-exceeded with truncation marker, single-rule blowing cap.
4. **Default-template seeds** in `internal/templates/builtin/default.toml` — bindings for `build`, `build-qa-proof`, `build-qa-falsification`, `plan-qa-proof`, `plan-qa-falsification`, `plan` (planner). Tests cover each binding loads + validates clean.
5. **Round-history aggregation** — when `include_round_history = true` and `metadata.spawn_history[]` non-empty, aggregator includes prior-round terminal events under `<bundle>/context/round_history/round_<N>.json`. Tests cover empty-history, single-round, multi-round.

### 7.3 Items not explicitly in SKETCH but PLAN.md should add

Three items the SKETCH leaves implicit that I'd surface as named droplets:

- **`BindingResolved` priority-cascade resolver** — the `CLI > MCP > TUI > TOML > absent` cascade (memory §3 / SKETCH F.7.3) currently lives in spawn.go's mind. With the adapter seam, this becomes a named function (`ResolveBinding(rawBinding, overrides) BindingResolved`) consumed by every adapter. One droplet.
- **CLI-agnostic monitor refactor** — F.7.4's monitor today implicitly assumes claude's stream taxonomy. Refactoring into a CLI-agnostic monitor that consumes `adapter.ParseStreamEvent` results is a discrete droplet.
- **Adapter-authoring documentation** — companion to F.7.11. One MD droplet covering "how to add a new CLI adapter to Tillsyn." Lands in same drop as the adapter seam.

---

## 8. Hard-Constraint Audit

Re-read the dev's hard constraints against the SKETCH:

- **"Tillsyn never holds secrets."** SKETCH lines 151, 154, 155 honor this. `env` rejects `KEY=value`. PASS.
- **"No Docker awareness in Tillsyn core."** SKETCH line 156 explicit: "No OAuth registry, no Docker awareness, no container model in Tillsyn core. These stay external to the binary." PASS.
- **"F.7.18 flexibility is load-bearing."** SKETCH lines 159, 181-184 carry the framing. The dev's verbatim quote is honored. PASS, with the quiet-drift refinement P6.
- **"Pre-MVP rules in force."** No migration logic, no closeout MD rollups, opus builders, single-line conventional commits, NEVER raw `go test` / `mage install`. SKETCH does not violate any of these. The `permission_grants` schema addition (P-§6.4) is dev-fresh-DB friendly per pre-MVP rule (no migration logic in Go). PASS.
- **"No Hylla calls."** This review was done via direct `Read` on SKETCH.md, the spawn architecture memory, spawn.go, and schema.go. No Hylla queries issued. PASS.

No hard-constraint violation. No planning blocker.

---

## 9. Summary Of Plan Recommendations

| ID  | Recommendation | Target |
|-----|----------------|--------|
| P1  | Spec `StreamEvent` as minimal cross-CLI canonical shape; CLI-specifics stay inside adapter | F.7.17 |
| P2  | Make `BundlePaths` a thin handle; let each adapter materialize CLI-specific subdirs | F.7.17 |
| P3  | Document the three CLI-shape invariants for future-adapter authors | F.7.17 |
| P4  | `validateAgentBindingEnvNames` covers env-shape, duplicates, empty-string, posix-portable name regex | F.7.17 |
| P5  | Pin missing-env failure surface as early-pre-lock | F.7.17 |
| P6  | Add explicit assertion that planner agents never depend on `[context]` being present | F.7.18 |
| P7  | Schema spec shows TWO examples — populated (bounded mode) AND omitted (agentic mode) | F.7.18 |
| P8  | Validation split: schema validators in `templates.Load`; engine assumes pre-validated input | F.7.18 |
| P9  | Pin priority semantics as declared rule order (not type-based) | F.7.18 |
| P10 | Clarify total-cap algorithm: render in order until cap, drop later with bundle-level marker | F.7.18 |
| P11 | Budget Drop 4d at 9-11 droplets, not 7-10 | Drop 4d |
| P12 | Spec the four Drop-4c-must-land items so Drop 4d is additive | F.7.17 |
| P13 | Spec `BindingResolved` plumbing, `manifest.json cli_kind`, `permission_grants cli_kind` | F.7.17 |

Plus four cross-section retro-edits to F.7.1-F.7.16 (§6.1 bundle layout phasing, §6.2 F.7.3 framing, §6.3 F.7.4 framing, §6.4 F.7.5 schema addition).

---

## 10. Verdict

**PASS-WITH-NITS.** Both F.7.17 and F.7.18 are architecturally sound. The three-method `CLIAdapter` interface is the right factoring; the `command` / `args_prefix` / `env` triad covers all named interop cases; `internal/app/dispatcher/context/` is the right home for the aggregator; the flexibility framing is honored.

The 13 plan recommendations above plus four cross-section retro-edits are the substance the eventual PLAN.md should fold in. None block planning; all sharpen the droplet decomposition. SKETCH.md remains a placeholder per its own self-declaration (line 3) and the nits are exactly the kind of refinement full PLAN.md authoring is supposed to surface.

**No planning blocker. Ready to dispatch parallel planners on Theme F.7 once Drop 4b's auto-promotion subscriber finishes settling.**

---

## Hylla Feedback

`N/A — task touched non-Go files only` (the SKETCH.md and the spawn architecture memory MD; cited code symbols verified via direct `Read` on `spawn.go` and `schema.go` rather than Hylla, faster for one-shot file:line checks at this stage). No Hylla queries issued, no miss to report.
