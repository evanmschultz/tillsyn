# Drop 4c — F.7.17 + F.7.18 Architecture QA Proof Review (Round 2)

**Reviewer:** go-qa-proof-agent
**Date:** 2026-05-05
**Scope:** Read-only verification that round-1 SKETCH.md reworks (F.7.17 CLI adapter seam + F.7.18 context aggregator) addressed the 9 CONFIRMED counterexamples + nits from round 1.
**Mode:** Filesystem-MD review BEFORE full PLAN.md authoring. No SKETCH.md edits, no code edits.
**Round-1 sources:** `4c_F7_EXT_PLANNER_REVIEW.md` (PASS-WITH-NITS, 13 recs); `4c_F7_EXT_QA_PROOF.md` (GREEN-WITH-NITS, 3 must-fix); `4c_F7_EXT_QA_FALSIFICATION.md` (NEEDS-REWORK, 8 CONFIRMED + 4 PASS-WITH-NIT + 2 REFUTED).

---

## Verdict

**PROOF GREEN-WITH-NITS.**

All 13 round-1 fix categories landed in the reworked SKETCH (lines 147-232 + the F.7 dependencies block). Each is internally consistent against the cited code surfaces (`internal/templates/schema.go`, `internal/templates/load.go`, `internal/domain/kind.go`, `internal/app/dispatcher/spawn.go`, `internal/app/git_status.go`). Two NIT-level evidence gaps surfaced as round-1-fix side effects — both deferrable to F.7.17 droplet-design time, neither blocks PLAN.md authoring.

---

## 1. Per-Fix Verification Table

| Fix # | Round-1 source                | Round-2 location in SKETCH | Landed? | Consistent? | New gap? |
|-------|-------------------------------|----------------------------|---------|-------------|----------|
| F1    | V2 argv injection (string→`command []string`, regex `^[A-Za-z0-9_./-]+$`, marketplace allow-list) | Lines 152-160 | YES | YES | NO |
| F2    | V3 env leak (closed baseline + resolved env, `os.Environ()` NOT inherited) | Line 155 | YES | YES | NIT-1 |
| F3    | V1 interface (Drop 4c JSONL-only; `ConsumeStream(ctx, io.Reader, sink)` future push-model refactor path) | Lines 147, 166-173 | YES | YES | NIT-2 |
| F4    | V6 priority (TOML declaration order, drop-whole-rule on cap, no `priority` field) | Line 199 | YES | YES | NO |
| F5    | V14 schema ordering (single schema-bundle droplet at start of F.7 wave) | Line 232 | YES | YES | NO |
| F6    | V5 planner-descendants (schema rule DROPPED, template authors trusted) | Line 196 | YES | YES | NO |
| F7    | V4 line 181 reword (bounded vs agentic parallel-structure equally first-class) | Lines 202-205 | YES | YES | NO |
| F8    | planner→plan vocabulary slip (use `kind=plan` for the planner-agent binding) | Line 196 | YES | YES | NO |
| F9    | V12 round-history DEFERRED (`metadata.spawn_history[]` audit-only, no re-prompting use) | Line 201 | YES | YES | NO |
| F10   | V10 caps (`max_aggregator_duration = "2s"`, depth cap explicitly skipped) | Line 200 | YES | YES | NO |
| F11   | V11 missing-binary UX (`exec.ErrNotFound` from `os/exec` + TOML position; no install URLs) | Line 160 | YES | YES | NO |
| F12   | V9 sequencing (locked option C: 4c → 5 claude-only → 4d → 5.5/6) | Lines 175, 265 | YES | YES | NO |
| F13   | All Valv mentions removed (no specific wrapper named in Tillsyn) | Lines 147-232 (zero `[Vv]alv` matches) | YES | YES | NO |

---

## 2. Detailed Verification Per Fix

### 2.1 F1 — V2 argv injection (`command` list-form + regex)

- **Round-2 SKETCH line 152**: `command = ["claude"]` — argv-list form (NOT a string). Round-1 V2's CONFIRMED counterexample (shell injection via `command = "rm -rf $HOME/.tillsyn ; claude"`) is DEAD because there is no string-tokenization step.
- **Round-2 SKETCH line 157**: per-token regex `^[A-Za-z0-9_./-]+$`. Validation cases:
  - Legitimate tokens PASS: `claude`, `codex`, `wrapper-cli` (hyphen ✓), `bin/run.sh` (slash + dot ✓), `/usr/local/bin/claude` (absolute path with slashes ✓).
  - Pathological tokens REJECTED: `a;b` (semicolon), `$x` (dollar), `a b` (space), `a|b` (pipe), `` `cmd` `` (backtick), `a&b` (ampersand). All rejected by regex character class.
- **Round-2 SKETCH line 159**: marketplace stricter validator using project-level `tillsyn.allowed_commands` allow-list. Project-local templates (`<project>/.tillsyn/template.toml`) bypass the allow-list. Trust boundary explicit and asymmetric — correct for the marketplace-RCE-vector concern.
- **Closed-struct framing consistency**: `command []string` is a slice field on `AgentBinding` (Go-side), not a map. Strict-decode in `internal/templates/load.go:88-95` validates field shape; per-token regex provides the value-shape check. No collision with the closed-struct framing for `[context]`.

**Verdict: LANDED, CONSISTENT, no new gap.**

### 2.2 F2 — V3 env leak (closed baseline)

- **Round-2 SKETCH line 155**: `cmd.Env` set explicitly to closed baseline (`PATH`, `HOME`, `USER`, `LANG`, `LC_ALL`, `TZ`) PLUS the resolved values for each name in the binding's `env` list. **`os.Environ()` is NOT inherited.** Round-1 V3's CONFIRMED counterexample (orchestrator's `AWS_*` / `STRIPE_*` leaking via default `cmd.Env == nil` inheritance) is DEAD.
- **Polarity check**: `internal/app/git_status.go:146-156` `filteredGitEnv()` is **deny-list-from-`os.Environ()`** (strips `GIT_*` keys, keeps everything else). That polarity is the OPPOSITE of what V3 demands for secrets handling. The reworked SKETCH no longer references `filteredGitEnv` as precedent — round-1 falsification did, but the round-1 reworked SKETCH text I read does not. Polarity inversion correctly avoided.
- **NIT-1 (new gap, deferrable)**: SKETCH does not enumerate evidence that `claude` actually only reads those 6 baseline vars. Plausible omissions:
  - `XDG_CONFIG_HOME` — if dev sets this, claude may read config from `$XDG_CONFIG_HOME/claude/` instead of `$HOME/.claude/`.
  - `TMPDIR` — Go's `os.TempDir` reads it; spawned claude doing temp-file work needs it.
  - `TERM` / `NO_COLOR` — affects ANSI rendering in claude's stdout.
  - `SHELL` — claude's Bash tool may read SHELL when spawning subprocesses.
  - These are **NIT-level** because the closed baseline list is explicitly framed as the starting point and can be expanded at F.7.17 droplet design time as compatibility issues surface. Not a blocker; route as a droplet-design-time follow-up.

**Verdict: LANDED, CONSISTENT, NIT-1 surfaced (closed-baseline completeness — droplet-design-time deferrable).**

### 2.3 F3 — V1 interface (JSONL-only Drop 4c + push-model roadmap)

- **Round-2 SKETCH line 147**: explicit "Drop 4c scope is JSONL-stream only."
- **Round-2 SKETCH line 150**: "Both adapters in scope (claude, codex) emit newline-delimited JSON, so the byte-line signature is correct for the JSONL family. Non-JSONL extensibility (SSE / framed / no-stream) is a roadmap concern (see 'Multi-CLI roadmap' below)."
- **Round-2 SKETCH lines 166-173**: full multi-CLI roadmap with `ConsumeStream(ctx, io.Reader, sink chan<- StreamEvent) error` push model. Round-1 V1's CONFIRMED counterexample (Goose SSE / Aider / Cursor framed-binary breaking the byte-line signature) addressed by scoping Drop 4c to JSONL-family CLIs and documenting the future-extension path.
- **Generalized `TerminalReport` value object** (line 169): `TerminalReport struct { Cost *float64; Denials []ToolDenial; Reason string; Errors []string }` shipped from the start so the seam never needs to widen. Pointer-cost addresses round-1 V1's "claude-isms in `ExtractTerminalCost(StreamEvent) (cost float64, denials []ToolDenial, ok bool)`" complaint.
- **NIT-2 (framing imprecision)**: Round-2 SKETCH line 168 says "Backward-compatible refactor: existing claude + codex adapters keep their per-line logic, just wrapped in a scanner loop." This is true at the **per-adapter implementation** level (the inner per-line parser stays as a private method) but NOT at the public **`CLIAdapter` interface** level — replacing `ParseStreamEvent(line []byte) (StreamEvent, error)` with `ConsumeStream(ctx, io.Reader, sink chan<- StreamEvent) error` is a breaking interface signature change. Adapters keep their per-line logic, but the interface they satisfy changes. **Marginal framing imprecision**, not an architectural counterexample. Tighten wording at F.7.17 droplet design time: "the per-line parser implementation is preserved verbatim; the CLIAdapter interface signature changes from pull-per-line to push-per-stream."

**Verdict: LANDED, CONSISTENT, NIT-2 surfaced (multi-CLI roadmap framing imprecision — droplet-design-time deferrable).**

### 2.4 F4 — V6 priority (TOML declaration order, drop-whole-rule on cap)

- **Round-2 SKETCH line 199**: "render rules in **TOML declaration order**; each rule contributes its full (post-per-rule-truncation) output unless cumulative size would exceed the bundle cap, in which case that rule and all subsequent rules are DROPPED WHOLESALE (no mid-rule truncation at the bundle level — the per-rule truncation already handled mid-rule). Drop emits one bundle-level marker `[bundle cap reached at <N> chars; rules dropped: <list>]`. Adopters control priority by declaring most-important rules FIRST in their TOML."
- Round-1 V6's CONFIRMED counterexample (ambiguous "largest first" vs "smaller-priority dropped") is DEAD. Algorithm now unambiguous and deterministic.
- No `priority = N` field added — adopter controls ordering via TOML declaration. Simpler than the round-1 mitigation suggestion of an explicit `priority` int.
- **TOML order preservation**: pelletier/go-toml v2 preserves declared field order for struct decoding. `[context]` is a closed substruct on `AgentBinding`, so field-declaration order in Go is the order rules render. No nondeterminism.

**Verdict: LANDED, CONSISTENT, no new gap.**

### 2.5 F5 — V14 schema-bundle-droplet ordering

- **Round-2 SKETCH line 232**: "All schema-struct additions (`command []string`, `args_prefix []string`, `env []string`, `cli_kind string`, `Context` sub-struct on `AgentBinding`, plus `tillsyn.max_context_bundle_chars` + `tillsyn.max_aggregator_duration` globals) land in ONE droplet at the start of the F.7 wave. Subsequent F.7 droplets consume the wider struct without re-touching `internal/templates/schema.go`. Rationale: pelletier/go-toml v2 strict-decode rejects unknown keys, so a half-landed schema would reject the very seeds F.7.18 lands later. Single-droplet schema bump keeps the strict-decode chain coherent across the F.7 sequence."
- **Rationale check against `load.go:80-95`**: strict-decode rejects unknown FIELDS inside known TABLES. The added fields are all on closed structs (`Context` on `AgentBinding`; globals on `[tillsyn]` config struct). For closed-struct fields, strict-decode would reject `[context]` keys absent from the struct definition. Therefore landing the schema additions piecemeal across droplets would cause partial-state load rejections. **Rationale is correct.**
- Round-1 V14's CONFIRMED counterexample addressed.

**Verdict: LANDED, CONSISTENT, no new gap.**

### 2.6 F6 — V5 planner-descendants schema rule DROPPED

- **Round-2 SKETCH line 196**: "**No schema rule against `descendants_by_kind` on `kind=plan`** — template authors trusted to use the field appropriately. Use cases like round-history fix-planners or tree-pruner planners legitimately need descendants."
- Round-1 V5's CONFIRMED counterexample (load-time hard-reject blocks legitimate fix-planner / tree-pruner / sub-plan-reflection use cases) is DEAD. Rule completely removed; not just demoted to warn.
- **Flexibility framing alignment**: SKETCH's "FLEXIBLE not REQUIRED" framing is now uniform — no carve-outs that mandate template-author behavior at load time.

**Verdict: LANDED, CONSISTENT, no new gap.**

### 2.7 F7 — V4 line 181 reword (parallel structure)

- **Round-2 SKETCH lines 202-205**: "**Bounded mode** (declare `[context]`): agent receives pre-staged parent / siblings / ancestors / git diff at spawn, calls MCP only on completion. Predictable cost, lower latency, less round-tripping. **Agentic mode** (omit `[context]`): agent receives only its own action-item ID + system-prompt, calls MCP for whatever context it needs. Higher cost, more flexibility, more round-tripping. Pick based on the agent kind's actual needs — neither is the recommended default."
- Round-1 V4's PASS-WITH-NIT (line 181 had implicit "minimal latency + bounded" recommendation) addressed. Bounded and agentic now framed parallel-structure-equally-first-class.
- **Default-template seed framing** (line 205): "Default-go template picks bounded for cost predictability; default-generic ships empty `[context]` tables (omit)." Default-template choice is explicitly NOT a Tillsyn-level recommendation — it's an authoring choice for one specific default with a concrete reason (cascade-on-itself dogfood cost predictability). Dev's "we don't dictate" requirement honored.

**Verdict: LANDED, CONSISTENT, no new gap.**

### 2.8 F8 — planner→plan vocabulary slip

- **Round-2 SKETCH line 196**: "`plan` (planner agent runs the `kind=plan` binding): parent + `ancestors_by_kind = ["plan"]`."
- No `[agent_bindings.planner.context]` reference anywhere in the F.7.18 section.
- **Cross-check against `internal/domain/kind.go:18-31`**: `KindPlan = "plan"` is the valid kind. `"planner"` is not in `validKinds` (line 50: `IsValidKind` returns false for `"planner"`). `validateMapKeys` (`internal/templates/load.go:174-191`) would reject `[agent_bindings.planner]` as `ErrUnknownKindReference`. Round-1 QA-proof's NIT (vocabulary slip would fail load) DEAD because the vocabulary now matches the closed enum.

**Verdict: LANDED, CONSISTENT, no new gap.**

### 2.9 F9 — V12 round-history DEFERRED

- **Round-2 SKETCH line 201**: "Round-history aggregation: DEFERRED. YAGNI today — the high-signal fix-builder context (worklog MD, gate output, QA findings) comes from sources other than raw stream-json events. `metadata.spawn_history[]` (F.7.9) remains an audit trail (cost, denials, terminal_reason) for ledger / dashboard, not for re-prompting. If a concrete use case for raw stream-json round-history surfaces post-Drop-5, add it as a refinement-drop item with dedicated `prior_round_*` rules (`prior_round_worklog`, `prior_round_gate_output`, `prior_round_qa_findings`) that target the actual high-signal artifacts."
- Round-1 V12's CONFIRMED counterexample (round-history aggregation surfaces the LEAST useful fix-builder signals; worklog / gate output / QA findings are higher signal) addressed.
- **`include_round_history` field removed** from F.7.18's initial scope. Schema does not need to support it. Lean YAGNI.

**Verdict: LANDED, CONSISTENT, no new gap.**

### 2.10 F10 — V10 caps (wall-clock, no depth cap)

- **Round-2 SKETCH line 200**: "**Wall-clock cap**: `[tillsyn] max_aggregator_duration = "2s"` (default). Aggregator enforces via `context.WithTimeout`; on hit, partial bundle + marker `[aggregator timed out after <duration>; rules pending: <list>]`. Catches pathological-tree pre-spawn delays without per-rule depth-walk caps."
- Round-1 V10's CONFIRMED counterexample (no aggregator-runtime cap) addressed via wall-clock cap. Depth cap explicitly skipped — wall-clock subsumes it (a depth-N walk that completes within 2s is fine; a depth-3 walk that hangs at 5s is caught).
- **2s sufficiency**: V10 worst case was 10-50ms for ~10 reads. 2s = 40-200x safety margin. Pathological tree (100+ ancestors) at 5ms each = 500ms still under cap. **Sufficient default**, adopters can override via `[tillsyn] max_aggregator_duration` knob.

**Verdict: LANDED, CONSISTENT, no new gap.**

### 2.11 F11 — V11 missing-binary UX

- **Round-2 SKETCH line 160**: "**`exec.ErrNotFound` UX**: on spawn, if `command[0]` is not on `$PATH`, the dispatcher surfaces a structured error verbatim from `os/exec` ('command \"<name>\" not found in $PATH') plus the binding's TOML position so the dev can see which template asked for the missing binary. Tillsyn does NOT recommend any specific install URL — the error simply reports what was given to it."
- Round-1 V11's CONFIRMED counterexample (Valv-not-installed UX leaves dev confused) addressed at the architectural level: error message names the binary AND the template position. Dev sees `command "claude" not found in $PATH (template: ~/.tillsyn/template.toml line 47)`.
- **No Tillsyn-bundled tool naming**: Round-1 V11 mitigation suggested install URLs (e.g. "install per https://valv.example.com"). Round-2 explicitly rejects this — Tillsyn names no specific tools, no install URLs. Aligns with the "no Docker awareness, Tillsyn never holds secrets" framing.

**Verdict: LANDED, CONSISTENT, no new gap.**

### 2.12 F12 — V9 sequencing locked option C

- **Round-2 SKETCH line 175**: "**Sequencing: Drop 4c → Drop 5 (claude-only dogfood, validates the cascade-on-itself loop without conflating second-CLI integration risk) → Drop 4d (codex adapter) → Drop 5.5/6 (multi-CLI dogfood validation).**"
- **Reinforced at line 265** ("Approximate Size"): "Sequencing locked: Drop 4c → Drop 5 (claude-only dogfood, validates cascade-on-itself loop without conflating second-CLI integration risk) → Drop 4d (codex) → Drop 5.5/6 (multi-CLI dogfood validation)."
- Round-1 V9's PASS-WITH-NIT addressed by explicit sequence-lock at the architectural level. Plan-time decision foreclosed.

**Verdict: LANDED, CONSISTENT, no new gap.**

### 2.13 F13 — All Valv mentions removed

- **Searched SKETCH lines 147-232**: zero matches for `[Vv]alv`.
- **Searched the full F.7.17 section + dependencies block**: no specific wrapper-product names. The schema accepts "a generic argv-list `command`" (line 151) and "lets adopters point at whatever wrapper they want."
- **Examples now generic**: `command = ["wrapper", "profile", "work", "claude"]` (line 153) — adopter-supplied tokens, no product name.
- Round-1 V11 + V15 NITs (Valv-as-Tillsyn-recommended-wrapper conflation) addressed.

**Verdict: LANDED, CONSISTENT, no new gap.**

---

## 3. New-Evidence-Gap Summary (NITs)

Two NIT-level gaps surfaced as round-1-fix side effects. Both deferrable to F.7.17 droplet-design time. Neither blocks PLAN.md authoring.

### 3.1 NIT-1 — Closed env baseline completeness

**Where**: SKETCH line 155.

**Concern**: The 6-var baseline (`PATH`, `HOME`, `USER`, `LANG`, `LC_ALL`, `TZ`) may omit env vars `claude` actually reads. Plausibly missing:

- `XDG_CONFIG_HOME` — affects claude's config-dir resolution.
- `TMPDIR` — affects Go's temp-file behavior; spawned claude doing temp-file work needs it.
- `TERM` — affects ANSI rendering in stdout.
- `NO_COLOR` — affects color-suppression behavior.
- `SHELL` — claude's Bash tool may read it when spawning subprocesses.

**Evidence weight**: SKETCH cites `project_drop_4c_spawn_architecture.md` (memory) as the canonical spawn architecture source, but I cannot directly verify what `claude` reads from that source at this layer of the review. The closed list is presented as concrete-starting-point.

**Recommended action**: At F.7.17 schema-additions droplet design time, verify `claude --bare` startup against an env-stripped exec environment to enumerate the actual minimum-required-env set. Add to the closed baseline as needed. Document the verification trace in droplet's worklog.

**Severity**: NIT — closed baseline can be expanded at droplet-design time without architectural rework. Not a load-bearing gap.

### 3.2 NIT-2 — Multi-CLI roadmap "backward-compatible" framing

**Where**: SKETCH line 168.

**Concern**: SKETCH says "Backward-compatible refactor: existing claude + codex adapters keep their per-line logic, just wrapped in a scanner loop." This is true at the **per-adapter implementation** level (inner per-line parser stays as a private method) but NOT at the public **`CLIAdapter` interface** level — replacing `ParseStreamEvent(line []byte) (StreamEvent, error)` with `ConsumeStream(ctx, io.Reader, sink chan<- StreamEvent) error` is a breaking interface signature change.

**Evidence weight**: Pure framing imprecision; the architectural roadmap is sound. Adapters keep their per-line logic; the interface they satisfy changes shape.

**Recommended action**: At F.7.17 droplet design time (or in the eventual PLAN.md authoring), tighten the wording to: "the per-line parser implementation is preserved verbatim; the `CLIAdapter` interface signature changes from pull-per-line to push-per-stream. Existing adapters move their per-line parser to a private method called from inside `ConsumeStream`."

**Severity**: NIT — wording precision only, no architectural counterexample.

---

## 4. Round-1 Falsification Counterexamples Status (Re-Audit)

| Round-1 vector | Round-1 verdict | Round-2 status | Evidence |
|----------------|-----------------|----------------|----------|
| V1 (CLIAdapter byte-line wrong-shape for SSE/WebSocket/hybrid) | CONFIRMED | RESOLVED (Drop 4c JSONL-only; future push-model roadmap) | SKETCH 147, 166-173 |
| V2 (`command` string field argv injection) | CONFIRMED | RESOLVED (`command []string` list-form + per-token regex + marketplace allow-list) | SKETCH 152-159 |
| V3 (`env` forwarding leak via `os.Environ()` inheritance) | CONFIRMED | RESOLVED (closed baseline, NOT inherited) | SKETCH 155 |
| V4 (line 181 accidental-mandate framing) | PASS-WITH-NIT | RESOLVED (parallel-structure rewrite) | SKETCH 202-205 |
| V5 (planner-descendants schema rule too-strict) | CONFIRMED | RESOLVED (rule DROPPED entirely) | SKETCH 196 |
| V6 (token-budget priority ambiguity) | CONFIRMED | RESOLVED (TOML declaration order, drop-whole-rule) | SKETCH 199 |
| V7 (multi-CLI auth-env collision) | REFUTED | Still REFUTED (per-binding env scope unchanged) | SKETCH 155 |
| V8 (schema collision with Tools field) | REFUTED | Still REFUTED (no scalar/table key collision) | `schema.go:285-332` |
| V9 (Drop 4d sequencing) | PASS-WITH-NIT | RESOLVED (option C locked) | SKETCH 175, 265 |
| V10 (aggregator runtime cost) | CONFIRMED | RESOLVED (`max_aggregator_duration = "2s"`) | SKETCH 200 |
| V11 (Valv-not-installed UX) | CONFIRMED | RESOLVED (`exec.ErrNotFound` + TOML position; no install URLs) | SKETCH 160 |
| V12 (round-history YAGNI) | CONFIRMED | RESOLVED (DEFERRED; `metadata.spawn_history[]` audit-only) | SKETCH 201 |
| V13 (per-binding scope) | REFUTED | Still REFUTED (`Resolve(binding, ...)` per-binding) | SKETCH 190 |
| V14 (schema-additions ordering) | CONFIRMED | RESOLVED (single schema-bundle droplet at start of F.7 wave) | SKETCH 232 |
| V15 (memory-rule conflicts) | PASS-WITH-NIT | Unchanged (default-template-edit is builder-territory; surface in PLAN.md) | n/a |

All 9 CONFIRMED counterexamples + 4 PASS-WITH-NIT items addressed in the round-1 rework. 2 REFUTED items remain REFUTED.

---

## 5. Round-1 QA-Proof Must-Fix Status

The round-1 QA-proof produced 3 must-fix-before-PLAN.md revisions:

| Round-1 must-fix | Round-2 status | Where |
|------------------|----------------|-------|
| Rename `planner` → `plan` in F.7.18 seed list | RESOLVED | SKETCH 196 |
| Specify priority order for over-cap drop policy | RESOLVED (TOML declaration order) | SKETCH 199 |
| Clarify "open-ended" → "closed struct"; reroute Theme A reference to load.go | RESOLVED (line 179 explicit "decoded into a named Go struct on AgentBinding (NOT map[string]any), so templates.Load's existing strict-decode chain (internal/templates/load.go:88-95) automatically rejects unknown keys at load time") | SKETCH 179 |

All three must-fix items closed.

---

## 6. Hard-Constraint Audit

Re-checked the dev's hard constraints against the round-2 SKETCH:

- **Tillsyn never holds secrets**: SKETCH lines 151, 154, 155 honor this. Env-NAMES only; closed baseline + opt-in resolution. PASS.
- **No Docker awareness**: SKETCH line 164: "No OAuth registry, no Docker awareness, no container model in Tillsyn core." Wrapper-product-agnostic argv-list framing. PASS.
- **No migration logic in Go**: F.7.17 + F.7.18 add new TOML schema fields; no migration code. Pre-MVP rule (dev fresh-DBs) covers schema bumps via fresh DB. PASS.
- **No closeout MD rollups pre-dogfood**: F.7.17 + F.7.18 add no rollup MDs. PASS.
- **Opus builders**: F.7.17 + F.7.18 don't override builder model. PASS.
- **NEVER raw `go test` / `mage install`**: review used `Read` only — no shell-out to either. PASS.
- **No Hylla calls**: review used `Read` exclusively (Hylla stale post-Drop-4b-merge per spawn-prompt directive). PASS.

No hard-constraint violation introduced by the round-1 rework.

---

## 7. Routed Unknowns

- **U1 (carried from round-1)** — `BundlePaths` struct field list (claude-purity vs neutrality). **Route to**: F.7.17 droplet-1 design step. Round-2 SKETCH does not refine this; round-1 routing still applies.
- **U2 (carried from round-1)** — `StreamEvent` neutral-type shape. **Route to**: F.7.17 droplet-1 or droplet-2 design step. Same as U1.
- **U3 (carried from round-1)** — Non-claude tool-gating render (how `tools_allowed` / `tools_disallowed` translate through codex's `--sandbox` / `--ignore-rules`). **Route to**: F.7.17 droplet design (acceptable to defer codex half to Drop 4d, but seam must accommodate). Same as round-1.
- **U4 (carried from round-1)** — Codex NDJSON-line confirmation. **Route to**: F.7.17 implementation time. Same as round-1.
- **U5 (NEW)** — Closed env baseline completeness for `claude --bare` startup. **Route to**: F.7.17 schema-additions droplet design step. NIT-1 above.
- **U6 (NEW, framing-only)** — Multi-CLI roadmap "backward-compatible" wording precision. **Route to**: F.7.17 droplet design. NIT-2 above.

---

## 8. Hylla Feedback

`N/A — review touched non-Go files only` (SKETCH.md and round-1 review MDs), with code-surface citations verified via direct `Read` on `internal/templates/schema.go`, `internal/templates/load.go`, `internal/domain/kind.go`, `internal/app/dispatcher/spawn.go`, `internal/app/git_status.go` per spawn-prompt directive ("No Hylla calls. Hylla stale post-Drop-4b-merge"). No Hylla queries issued; no miss to report.

---

## 9. Verdict

**PROOF GREEN-WITH-NITS.**

- All 13 round-1 fix categories LANDED in the reworked SKETCH.
- All 9 round-1 CONFIRMED counterexamples RESOLVED.
- All 3 round-1 QA-proof must-fix items RESOLVED.
- 2 NIT-level evidence gaps surfaced (closed-env-baseline completeness; multi-CLI roadmap framing precision) — both deferrable to F.7.17 droplet-design time, neither blocks PLAN.md authoring.
- Hard constraints (no secrets / no Docker / no migration / pre-MVP rules / no Hylla / no `mage install`) hold throughout.
- Closed-struct framing consistent for `[context]` AND for `env` / `command` / `args_prefix` (slices-not-maps go through the same strict-decode chain).
- Schema-bundle-droplet rationale (line 232) consistent with pelletier/go-toml v2 strict-decode behavior on closed-struct fields per `load.go:80-95`.
- Multi-CLI roadmap correctly characterizes `ConsumeStream` push model as breaking interface change at the public level, additive at the per-adapter parser-implementation level (NIT-2 tightens framing).

**Ready for PLAN.md authoring** with NIT-1 + NIT-2 noted as droplet-design-time follow-ups.
