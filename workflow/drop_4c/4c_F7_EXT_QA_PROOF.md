# Drop 4c — F.7.17 + F.7.18 Architecture QA Proof Review

**Reviewer:** go-qa-proof-agent
**Date:** 2026-05-04
**Scope:** Read-only review of `workflow/drop_4c/SKETCH.md` lines 147-205 (F.7.17 CLI adapter seam + F.7.18 context aggregator).
**Lens:** Proof — does committed code surface support the claims; are the claims internally consistent.

## Verdict

**PROOF GREEN-WITH-NITS.**

The two new sub-themes are sound, additive, and integrate cleanly with the committed `internal/templates/` + `internal/domain/` + `internal/app/dispatcher/` surface. Three claims need concrete revision before full PLAN.md authoring, plus a small handful of sketch-level NITs that are appropriate to defer to F.7.17 / F.7.18 droplet design.

---

## 1. Per-Claim Evidence

### 1.1 F.7.17 verbatim Codex CLI consistency — PROVED

SKETCH line 157 lists `codex exec --json`, `--profile <name>`, `--ignore-user-config`, `--ignore-rules`, `--sandbox`, `--ephemeral` (dev verified via `codex --help` + `codex exec --help` 2026-05-04). Three internal-consistency checks pass:

- **`--profile` ↔ `args_prefix` mapping.** F.7.17 line 153's `args_prefix = ["--profile", "tillsyn-build"]` semantics (prepended to argv before adapter-specific args) cleanly support both `valv claude --profile <p>` and `codex --profile <p>` because both treat `--profile` as a top-level flag.
- **`codex exec --json` ↔ `claude --output-format stream-json` analogy.** Both emit line-delimited JSON. The shared `ParseStreamEvent(line []byte)` interface works at the byte-line level; per-CLI event-taxonomy diversity is absorbed inside each adapter's body. **The seam is at the right granularity.**
- **Subcommand asymmetry** (`claude` direct vs `codex exec`) is absorbed by `BuildCommand` returning `*exec.Cmd` — adapters own the full argv shape.

### 1.2 F.7.17 `env` field schema-validation — PROVED, additive

`internal/templates/schema.go:285-332` confirms `AgentBinding` currently has NO `env`, NO `command`, NO `args_prefix`, NO `cli_kind` fields. F.7.17 is purely additive at the struct level. Validation hook point is `AgentBinding.Validate()` at schema.go:353 — already exists, runs at template-load time, and is the right place to add the value-shape check (rejecting strings containing `=`).

### 1.3 F.7.18 schema-validation / nested-table unknown-key claim — NIT (clarification needed)

SKETCH line 185: *"unknown keys rejected (per Theme A 'reject unknown keys at MCP boundary' rule extends here too)."* This conflates two surfaces.

- **Theme A's rule** (SKETCH line 26 + 61) refers to `DisallowUnknownFields` at the **MCP boundary** — JSON-RPC argument decoding inside `internal/adapters/server/`.
- **Templates' actual unknown-key behavior** is set inside `internal/templates/load.go:88-95`: `strictDecoder.DisallowUnknownFields()` runs against TOML, and `*toml.StrictMissingError` is wrapped as `ErrUnknownTemplateKey` at line 91. This DOES recurse into named struct fields automatically.
- **The recursion gotcha** (already documented in load.go:166-191): for `map[K]V` destinations, pelletier/go-toml/v2 accepts arbitrary keys; `validateMapKeys` exists precisely to catch them. So if `[agent_bindings.<kind>.context]` decodes into a NAMED STRUCT field on `AgentBinding`, unknown keys are rejected automatically. If it decodes into `map[string]any`, they are NOT.
- SKETCH line 185 also says the table is *"open-ended"*, which implies `map[string]any` — colliding with the next breath's claim that unknown keys are rejected.

**Recommended fix.** SKETCH should:
- Replace *"open-ended table validated by `templates.Load`"* with *"closed struct with named fields, decoded by `templates.Load`'s strict-decode chain (`DisallowUnknownFields` recurses into named struct fields automatically; see internal/templates/load.go:88-95)."*
- Drop the *"per Theme A 'reject unknown keys at MCP boundary' rule extends here too"* clause — it routes to the wrong layer. The right reference is the existing templates.Load strict-decode behavior.

### 1.4 F.7.18 planner-descendants validation — PROVED

SKETCH line 178: *"Schema validation rejects `descendants_by_kind` on `kind=\"plan\"`."*

- The discriminator is the binding map's TOML key: `[agent_bindings.<kind>]` decodes into `map[domain.Kind]AgentBinding` (schema.go:144). The kind axis IS the map key.
- Validation shape: iterate `tpl.AgentBindings`; for `KindPlan` entries, reject if `binding.Context.DescendantsByKind` is non-empty.
- This is a 5–10-line addition slotted into `templates.Load`'s validator chain, ordered after `validateMapKeys` so the kind axis is already proven valid.

### 1.5 Aggregator package boundary — PROVED

`internal/app/dispatcher/context/` does NOT exist. No conflict with the 27-file `internal/app/dispatcher/` Drop 4a/4b code. The proposed boundary cleanly separates argv assembly (spawn.go) from context-bundle assembly (new package).

### 1.6 Token-budget composition — GAP (priority unspecified)

SKETCH line 180:
- Per-rule `max_chars` default 50000 (50KB).
- Total cap `max_context_bundle_chars = 200000` (200KB).
- Over-cap policy: *"agent gets the largest rules first, smaller-priority rules dropped with marker."*

Two issues:
- **"Largest first" and "smaller-priority" reference different ordering axes** (size vs declared priority). The schema (line 162-171) defines `parent`, `parent_git_diff`, `siblings_by_kind`, `ancestors_by_kind`, `descendants_by_kind`, `delivery`, `max_chars` — no `priority` field.
- The ambiguity is not academic. The seed-default for `build-qa-proof` (line 175) declares 4 rules; with each at the 50KB default, total is exactly 200KB — at cap. A planner overriding `max_chars` higher on any single rule, OR a real git diff approaching its budget, makes over-cap a normal occurrence, not an edge case.

**Recommended fix (pick one):**
- (a) Add an explicit `priority` int field per context rule with a documented default ordering.
- (b) Define an implicit priority by rule type (e.g. parent > parent_git_diff > ancestors > siblings > descendants > round_history) and document it in the schema doc-comment.
- (c) Drop the *"largest rules first"* phrase entirely; specify only TOML declaration order (which pelletier/go-toml/v2 preserves for struct decoding).

### 1.7 Round-history aggregation depends on F.7.9 — PROVED, dependency already noted

SKETCH F.7.9 line 141 lists `metadata.spawn_history[]` as a NEW field. F.7.18 line 179's *"if `metadata.spawn_history[]` non-empty"* depends on it. SKETCH line 205 already calls this out: *"ordering is a planning concern only."* Defensive non-empty check naturally degrades when the field is absent. **No revision needed — dependency-order callout is correct.**

### 1.8 Default-template seed kinds — NIT (vocabulary slip)

SKETCH line 174-178 lists six seed kinds. Cross-checked against `internal/domain/kind.go:19-31` and `internal/templates/builtin/default.toml`:

| SKETCH name              | kind.go            | default.toml binding line | Status |
| ------------------------ | ------------------ | ------------------------- | ------ |
| `build`                  | `KindBuild` ✓      | `[agent_bindings.build]` 403 | OK |
| `build-qa-proof`         | `KindBuildQAProof` ✓ | line 442                | OK |
| `build-qa-falsification` | `KindBuildQAFalsification` ✓ | line 455      | OK |
| `plan-qa-proof`          | `KindPlanQAProof` ✓ | line 416                 | OK |
| `plan-qa-falsification`  | `KindPlanQAFalsification` ✓ | line 429         | OK |
| **`planner`**            | **NOT IN ENUM**    | binding is `[agent_bindings.plan]` line 377 | **SLIP** |

`planner` is the agent ROLE for the `plan` kind, not a kind itself. `domain.IsValidKind("planner")` returns false (kind.go:50-52 normalizes via `TrimSpace + ToLower`; "planner" stays "planner", not in `validKinds`). Writing `[agent_bindings.planner.context]` would be rejected by `validateMapKeys` (load.go:166-191).

**Recommended fix.** Rename `planner` → `plan` in F.7.18's seed-default list (SKETCH line 178). Keep the prose semantics ("planners cannot have descendants pre-staged") since they describe the agent that handles `kind=plan`.

### 1.9 Drop 4d preview / clean adapter seam — PARTIALLY PROVED, sketch-level NITs

Three claude-isms must NOT leak through `BuildCommand` / `ParseStreamEvent`:

- **Argv shape leakage (mitigated by interface).** F.7.3 lists ten claude-specific flags (`--bare`, `--plugin-dir`, `--system-prompt-file`, `--settings`, etc.). `BuildCommand` returns `*exec.Cmd`, owning the full argv shape. Dispatcher passes only `BindingResolved` + `BundlePaths`. **Interface satisfies adapter purity** — but the `BundlePaths` struct shape is not sketched. Without it, reviewers cannot confirm the struct exposes generic locations (system-prompt path, settings path, agent-file path) rather than claude-flag-shaped fields.
- **Event-taxonomy leakage.** F.7.4 lists claude-specific event types (`system/init`, `assistant`, `user`, `result`). The neutral `StreamEvent` return type from `ParseStreamEvent` is not sketched. Codex events differ per dev's probe.
- **Tool-gating render path.** F.7.2 specifies settings.json `permissions` deny rules as authoritative — settings.json is a CLAUDE concept. Codex has `--sandbox` + `--ignore-rules`. SKETCH does not specify how `tools_allowed` / `tools_disallowed` render through a non-claude adapter.

**Recommended fix.** At F.7.17 droplet design time (NOT sketch level — these are detail decisions), specify:
- `BundlePaths` struct field list (claude-neutral file-location names).
- `StreamEvent` struct field list (`Kind`, `Text`, `ToolName`, `ToolInput`, `ToolResult`, `IsTerminal`, `Cost`, `PermissionDenials`, `Errors`, etc. — fields both adapters can populate).
- How each adapter renders `tools_allowed` / `tools_disallowed` to CLI flags / config files.

These are NIT-level because the F.7.17 contract is sound — the seam exists and is well-placed. The detail just needs filling in at droplet-design granularity, and Drop 4d is the natural deadline.

---

## 2. Hard-Constraints Check

### 2.1 Tillsyn-never-holds-secrets — PROVED

SKETCH lines 151, 154, 155 are consistent throughout:
- Line 151: parenthetical *"Tillsyn never holds secrets — env var NAMES only"* — declared up front.
- Line 154: `env = ["TILLSYN_API_KEY", "ANTHROPIC_API_KEY"]` — names only example.
- Line 155: schema enforcement (`env` rejects values containing `=`); explicit *"Tillsyn never holds secrets"* heading; adopters route via Valv-managed Docker / system keychain / direnv.

No drift. The secret-handling story is stable across the entire F.7.17 section.

### 2.2 No Docker awareness in Tillsyn core — PROVED

SKETCH line 156 explicit: *"No OAuth registry, no Docker awareness, no container model in Tillsyn core. These stay external to the binary."* Line 152 frames Valv as an external wrapper Tillsyn points at via `command = "valv claude"`. F.7.10 (line 143) confirms Hylla awareness is removed from the shipped binary. No Docker / container concept leaks into the core binary's interface or schema.

---

## 3. Ancillary Findings (non-blocking)

### 3.1 NIT — F.7.17's `command` field default

SKETCH line 152 says *"default `claude`"* — confirm in droplet design that this is the literal string `"claude"` (relying on `$PATH` lookup) rather than an absolute path. `exec.Command(name, args...)` does `LookPath` for non-absolute names, so `"claude"` is the right shape. Worth a one-line schema doc-comment.

### 3.2 NIT — F.7.17 missing-required-env semantics

SKETCH line 154: *"Fail-loud on missing required-env."* The claim should make explicit at droplet design time that fail-loud happens at SPAWN time (not template-load time) — at template-load time the env vars haven't been resolved yet; the dispatcher checks `os.Getenv` for each named var when it builds the spawn `*exec.Cmd`. If not specified, a planner authoring a TOML with bogus env-var names would not be caught until the first spawn attempt.

### 3.3 NIT — F.7.18 `delivery` enum closure

SKETCH line 169 declares `delivery = "file"` with `"inline"` | `"file"` as the two values. This is a closed two-value enum. Schema validation should reject other strings; that requirement is implicit in the *"validates field types"* clause on line 185 but worth making explicit.

---

## 4. Recommended SKETCH Revisions

Before full PLAN.md authoring, three concrete edits to `workflow/drop_4c/SKETCH.md`:

- **Line 178 (F.7.18 seed defaults):** rename `planner` → `plan`. Vocabulary slip between role and kind axes.
- **Line 180 (F.7.18 token budget):** specify priority ordering (option a, b, or c above). Pick one and document.
- **Line 185 (F.7.18 schema validation):** clarify *"open-ended"* → *"closed struct with named fields"*; replace the Theme A reference with a pointer to `internal/templates/load.go`'s existing strict-decode chain.

The rest of F.7.17 + F.7.18 is sound and additive against the committed surface. Drop 4d's purely-additive codex adapter remains achievable on top of the F.7.17 seam, with the BundlePaths / StreamEvent / tool-gating-render details filled in at F.7.17 droplet design time.

---

## 5. Routed Unknowns

- **U1 — `BundlePaths` struct shape.** Claude-purity vs neutrality cannot be fully verified at sketch level. **Route to:** F.7.17 droplet-1 design step.
- **U2 — `StreamEvent` neutral-type shape.** Same constraint. **Route to:** F.7.17 droplet-1 or droplet-2 design step.
- **U3 — Non-claude tool-gating render.** How `tools_allowed` / `tools_disallowed` translate through codex's `--sandbox` / `--ignore-rules`. **Route to:** F.7.17 droplet design (acceptable to defer the codex half to Drop 4d, but the seam must accommodate it).
- **U4 — Codex NDJSON-line confirmation.** `codex exec --json` is line-delimited per dev's verbatim 2026-05-04 probe. Re-confirm at F.7.17 implementation time before locking the `ParseStreamEvent(line []byte)` signature.

---

## 6. Hylla Feedback

N/A — review touched non-Go (SKETCH.md) and Go-source files. Hylla-via-MCP was not used (per spawn-prompt directive: *"Use `Read` / `Grep` / `LSP` directly. Hylla is stale post-Drop-4b-merge."*). All evidence cited from `Read` of committed files.

---

## TL;DR

- **Verdict: PROOF GREEN-WITH-NITS.**
- **3 must-fix-before-PLAN.md revisions:** (a) rename `planner` → `plan` in F.7.18 seed list; (b) specify priority order for over-cap drop policy; (c) clarify *"open-ended"* and reroute the Theme A reference to load.go.
- **3 sketch-level NITs deferrable to F.7.17 droplet design:** BundlePaths shape, StreamEvent shape, tool-gating render for non-claude adapters.
- **Hard constraints (no secrets / no Docker) hold throughout.**
- **All claims that touch committed code traced cleanly to schema.go / load.go / kind.go / spawn.go / project.go / default.toml.**
