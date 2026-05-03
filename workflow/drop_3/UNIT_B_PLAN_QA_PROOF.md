# DROP 3 — UNIT B PLAN QA PROOF — ROUND 1

**Reviewed file:** `workflow/drop_3/UNIT_B_PLAN.md`
**Reviewer:** `go-qa-proof-agent` (subagent, opus)
**Round:** 1
**Date:** 2026-05-02

## Verdict

**PASS WITH MITIGATION-REQUIRED FINDINGS.** The decomposition is structurally sound, evidence-grounded, and the dependency graph + file-lock audit are correct. Three findings need mitigation before the planner's output is canonical for build, but none reshape the unit boundary.

## 1. Findings

### 1.1 (MITIGATION-REQUIRED) Droplet 3.B.8 call-site enumeration is incomplete

The planner's "Paths" block in 3.B.8 misses real call sites that LSP `findReferences` surfaces today:

- `internal/adapters/server/mcpapi/instructions_explainer.go:241-242` reads `kind.AllowedParentScopes` directly (`kind.AllowedParentScopes` length check + `joinKindScopes` rendering for the "rules" field). Plus `:391, 403, 528, 550` reference `domain.KindDefinition` returns. Total: 11 references in this file the planner did not list.
- `internal/adapters/server/common/app_service_adapter_mcp.go:1188, 1200, 1202, 1214` reference `KindDefinition` (4 hits).
- `internal/adapters/server/mcpapi/extended_tools_test.go:703, 706, 712, 720, 723` reference `KindDefinition` + `AllowedParentScopes` (5 hits).
- `internal/adapters/storage/sqlite/repo.go` references go far beyond the boot-seed: 13 hits at `:1061, 1095, 1130, 1140, 1156, 2940, 2942, 2964, 2966, 2970, 2976, 2982, 2988` — these are the `KindDefinition` row marshal/unmarshal helpers + the `archivedAt` filtering, untouched by the planner's "delete boot-seed" line.
- `internal/app/ports.go:21-24` declares the four `KindDefinition`-returning interface methods on the repo port (`GetKindDefinition`, `CreateKindDefinition`, `UpdateKindDefinition`, `ListKindDefinitions`) — these will continue to exist, but droplet 3.B.8 must explicitly state they survive (the type stays; only `KindTemplate` field deletion + `AllowedParentScopes` field deletion changes its shape).
- `internal/tui/model.go:35, 754, 932, 1341` + `internal/tui/model_test.go:39, 87, 100, 104, 14661` reference `KindDefinition` (9 hits). TUI is technically out-of-scope per Drop 3 (TUI overhaul is Drop 4.5), but if `KindDefinition.Template` and `AllowedParentScopes` fields are removed, these references break compile-time. Builder must either (a) stub the fields as zero-value-only carriers for back-compat, (b) sweep the TUI in this droplet, or (c) explicitly route this break to a follow-up droplet.
- `internal/app/snapshot.go:727, 1092, 1339, 1340` reference `KindDefinition`/`Template` beyond the cited `:94`.

**Mitigation.** Builder for 3.B.8 must run `LSP findReferences` on `domain.KindTemplate`, `domain.KindTemplateChildSpec`, `domain.KindDefinition.AllowedParentScopes` field, `domain.KindDefinition.AllowsParentScope` method BEFORE editing. Treat 3.B.8's "Paths" list as a starting-point, not a closed enumeration. Update droplet acceptance: "LSP `findReferences` on each deleted symbol returns 0 hits" already covers this, but the path list reads as comprehensive when it isn't. Add a one-liner: "Builder verifies LSP `findReferences` exhaustively before editing; the path list above is a starting-point lower bound, not a closed set."

### 1.2 (MITIGATION-REQUIRED) Decision deferred to builder in 3.B.5 import-direction question

3.B.5's "NOTE: import direction" paragraph hands the builder a real architecture decision: `KindCatalog` lives in `internal/templates/`, but `Project.KindCatalog` field needs to reference it. Two resolutions documented (lazy-decode `KindCatalogJSON json.RawMessage` accessor on `Project`; or invert `templates`/`domain` package layering). This is a planning-tier decision, not a builder-tier decision — letting the builder pick mid-flight risks two parallel droplets producing incompatible imports.

**Mitigation.** Orchestrator routes this back to the planner for a definitive recommendation before 3.B.5 enters build. Lazy-decode (option A) is the lower-risk path: `domain` stays import-free of `templates`, the JSON snapshot semantics on `Project.Metadata` are already established (Drop 1.75 set the precedent), and `Project.KindCatalog()` becomes a getter that decodes on demand. Option B (invert layering) is a non-trivial refactor across the whole `internal/domain` import graph.

### 1.3 (NIT) Architectural Decision #2 (`KindCatalog` package location) names `internal/templates/` but PLAN.md & spawn-brief said the new package shape was an open choice

The PLAN.md text at § 19.3 line 1623 says `KindCatalog` is "baked into a value type at project-creation time" — it does not name a package. The spawn brief routed package location to the planner. Planner's choice (`internal/templates/`) is sound, but the rationale ("keeps it adapter-adjacent and avoids `internal/app/` mass coupling") is weak — `internal/templates/` is NOT actually adapter-adjacent (adapters live under `internal/adapters/`, `internal/templates/` is a sibling of `internal/app/`, `internal/domain/`). Cleaner rationale: `internal/templates/` keeps the parser + schema + catalog as a hexagonal core sub-package, distinct from `internal/app/` services that consume it.

**Mitigation.** Cosmetic. Planner reframes the rationale in a follow-up update if convenient; not blocking.

### 1.4 (NIT) `ChildRule` schema field missing one explicit invariant

Per PLAN.md § 19.3 Template Overhaul Scope: "load-time validator catching unreachable rules / cycles / unknown kinds." Plus from 3.B.2 acceptance: "missing schema_version, wrong schema_version, malformed TOML." The four invariants cited in the QA spec are: (i) unreachable rules, (ii) cycles, (iii) unknown kinds, (iv) schema-version mismatch. 3.B.2 covers all four through its sentinel-error declarations (`ErrUnreachableChildRule`, `ErrTemplateCycle`, `ErrUnknownKindReference`, `ErrUnsupportedSchemaVersion`). One implicit invariant is not surfaced: **strict unknown-key rejection at the TOML decoder level** (PLAN.md "strict unknown-key rejection"). 3.B.2 covers this via `DisallowUnknownFields()` on the decoder + `ErrUnknownTemplateKey` sentinel — already present, just buried in the prose. Acceptable.

**Mitigation.** None required. Confirmed all four invariants + the strict-decode invariant are in scope.

### 1.5 (CONFIRMED CORRECT) Existing `KindTemplate` + `KindTemplateChildSpec` + `AllowedParentScopes` + `AllowsParentScope` get rewritten, not extended

LSP confirms:

- `KindTemplate` has 25 references across 8 files.
- `KindTemplateChildSpec` has 4 references across 2 files (3 in `internal/domain/kind.go`, 1 test in `kind_capability_test.go:20`).
- `AllowedParentScopes` field has 15 references across 7 files.
- `AllowsParentScope` method has 4 references: `internal/domain/kind.go:200`, `internal/app/kind_capability.go:566`, `internal/adapters/storage/sqlite/repo_test.go:2563`, `internal/domain/kind_capability_test.go:49`.
- `AppliesToScope` method (sibling on `KindDefinition`) has 3 references: `kind.go:189`, `kind_capability.go:562`, `kind_capability_test.go:46`.

3.B.8's intent ("delete deads + migrate every consumer to new API") is correctly scoped to the rewrite, not extension. PLAN.md alignment is exact.

### 1.6 (CONFIRMED CORRECT) TOML library choice — `github.com/pelletier/go-toml/v2`

`go.mod:66` already lists `github.com/pelletier/go-toml/v2 v2.2.4` as a direct dependency. Project tech stack in CLAUDE.md confirms it. 3.B.2's `DisallowUnknownFields()` API is present in v2 (`(*Decoder).DisallowUnknownFields()`). No fallback needed.

### 1.7 (CONFIRMED CORRECT) `Template.AllowsNesting(parent, child Kind) (bool, reason string)` signature matches PLAN.md § 19.3

PLAN.md line 1623 declares: "single `Template.AllowsNesting(parent, child Kind) (bool, reason string)` function as the one validation truth". 3.B.3 acceptance line: `func (t Template) AllowsNesting(parent, child domain.Kind) (allowed bool, reason string)` — identical signature with named returns. Stable reason format: `"kind %q cannot nest under parent kind %q (rule: %s)"` is reasonable. Empty-template universal-allow fallback aligns with Drop 2.8's empty-`AllowedParentScopes` semantics (verified at `internal/domain/kind.go:202-204`).

### 1.8 (CONFIRMED CORRECT) `[child_rules]` table + 4 load-time invariants

3.B.2 sentinel errors cover all four QA-spec invariants:

1. Unreachable rules → `ErrUnreachableChildRule`.
2. Cycles → `ErrTemplateCycle` (with DFS detection).
3. Unknown kinds → `ErrUnknownKindReference` (asserted against `domain.validKinds` 12-value enum at `kind.go:35-48`).
4. Schema-version mismatch → `ErrUnsupportedSchemaVersion`.

Plus strict-decode rejection via `DisallowUnknownFields()` + `ErrUnknownTemplateKey`. Five total invariants. All four QA-spec invariants are present.

### 1.9 (CONFIRMED CORRECT) Default template `templates/builtin/default.toml` re-introduces the deleted `templates/` package

Filesystem confirms `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/templates` does NOT exist. `git log --all --oneline -- templates/builtin/` confirms commit `29cce29 chore(templates): delete builtin package for drop 3 overhaul` removed it. 3.B.7 explicitly states "repo root, NOT under `internal/`; lives as a sibling of `cmd/`, `internal/`, `magefile.go`" — alignment is exact, the package is reintroduced fresh under the new TOML schema with no JSON legacy.

### 1.10 (CONFIRMED CORRECT) All 11 agent-binding fields enumerated

Cross-referenced PLAN.md lines 1653-1656 against 3.B.1 + 3.B.6 acceptance:

| PLAN.md field             | 3.B.1 / 3.B.6 |
| ------------------------- | ------------- |
| `agent_name`              | `AgentName string`             |
| `model`                   | `Model string`                 |
| `effort`                  | `Effort string`                |
| `tools`                   | `Tools []string`               |
| `max_tries`               | `MaxTries int`                 |
| `max_budget_usd`          | `MaxBudgetUSD float64`         |
| `max_turns`               | `MaxTurns int`                 |
| `auto_push`               | `AutoPush bool`                |
| `commit_agent`            | `CommitAgent string`           |
| `blocked_retries`         | `BlockedRetries int`           |
| `blocked_retry_cooldown`  | `BlockedRetryCooldown time.Duration` |

11/11 covered. `time.Duration` for cooldown is correct (TOML v2 supports duration parsing). `float64` for USD is reasonable for pre-MVP precision tier.

### 1.11 (CONFIRMED CORRECT) Cross-unit dependency on Unit A's `StructuralType` is hard-blocker-surfaced

3.B.1, 3.B.4, 3.B.7 each list "Cross-unit blocked by Unit A's `StructuralType` enum (orch wires)." Open Question #1 routes the placeholder strategy back to the orchestrator (string alias + TODO, OR gate Unit B entirely behind Unit A). Cross-unit blocker is explicit and routed; both options are sound; orchestrator decides.

### 1.12 (CONFIRMED CORRECT) Pre-MVP no-migration honored

Open Question #3 explicitly addresses the SQLite `kind_catalog` boot-seed deletion: "delete the seed INSERTs but **keep the table** so existing schema doesn't break. Dev fresh-DBs after 3.B.8 lands." Aligns with `feedback_no_migration_logic_pre_mvp.md`. No SQL backfill, no `till migrate` subcommand. Drop 3 is consistent with the pre-MVP rule.

## 2. Missing Evidence

### 2.1 (LOW SEVERITY) Cycle-detection algorithm choice not specified

3.B.2 says "DFS to detect cycles" but doesn't specify whether it tracks visited+stack (Tarjan-style, finds all SCCs) or simple visited-set (catches direct + transitive but not multi-rule cycles). For the `[child_rules]` table where each rule is parent→child, simple visited-set DFS is sufficient (no multi-edge cycles possible in this graph shape since each rule is a single directed edge). Builder choice; not a blocker.

### 2.2 (LOW SEVERITY) `Tools []string` validation deferred to Drop 4

3.B.6 explicitly defers tool-name validation to Drop 4 ("not enum"). PLAN.md Drop 4 (`§ 19.4`) does not yet have a "validate tools against MCP/Claude catalog" bullet. This is forward-routed correctly but the receiving end (Drop 4) needs a corresponding bullet. Out-of-scope for Unit B; flag for orch to seed in Drop 4 planning.

### 2.3 (LOW SEVERITY) 3.B.9 attention-item category constant location

3.B.9 says "`internal/domain/attention.go` or `internal/domain/inbox.go` (add `AttentionCategoryTemplateRejection` constant — LSP `workspaceSymbol` for `AttentionCategory` to find exact file)". This is a "builder figures out" instruction. Reasonable for a small constant addition; not a blocker.

## 3. Summary

**Verdict: PASS WITH MITIGATION-REQUIRED FINDINGS.**

The decomposition is correct in shape, dependency wiring, file-lock audit, and PLAN.md alignment. Two mitigation-required findings need orchestrator routing before build can proceed:

1. **3.B.8 path enumeration is a lower bound, not a closed set** — builder must run `LSP findReferences` exhaustively. Add explicit acceptance language.
2. **3.B.5 import-direction decision should be made by the planner, not the builder** — orchestrator routes back to the planner for a definitive recommendation (lazy-decode JSON snapshot is the cheaper path).

Plus three nits / low-severity items that don't block: rationale on package-location decision, cycle-detection algorithm choice (builder pick), Drop 4 forward-route note.

The unit is **ready for build orchestration after the two mitigation items resolve**.

## Verdict Summary

| Check                                                            | Result      |
| ---------------------------------------------------------------- | ----------- |
| Existing `KindTemplate`/`KindTemplateChildSpec`/`AllowedParentScopes`/`AllowsParentScope` rewritten not extended | PASS        |
| TOML library `github.com/pelletier/go-toml/v2`                   | PASS        |
| `Template.AllowsNesting(parent, child Kind) (bool, reason string)` signature | PASS        |
| `[child_rules]` load-time validator covers all 4 invariants      | PASS        |
| Default template re-introduces deleted `templates/` package      | PASS        |
| All 11 agent-binding fields enumerated                           | PASS        |
| Cross-unit dependency on Unit A's `StructuralType` surfaced as blocker | PASS        |
| Pre-MVP no-migration honored                                     | PASS        |
| 3.B.8 call-site enumeration completeness                         | MITIGATE    |
| 3.B.5 import-direction decision routed                           | MITIGATE    |
| Architectural Decision #2 rationale clarity                      | NIT         |
| Cycle-detection algorithm choice                                 | LOW SEVERITY |
| Drop 4 forward-route for `Tools` validation                      | LOW SEVERITY |

**Overall: PASS with two mitigation items routed back to the orchestrator.**

## Hylla Feedback

N/A — review touched non-Go MD planning files plus targeted reads of Go source via Read + LSP. No Hylla queries were issued; LSP `findReferences` was sufficient for full call-site enumeration of the four legacy symbols (`KindTemplate`, `KindTemplateChildSpec`, `AllowedParentScopes` field, `AllowsParentScope` method, `AppliesToScope` method). LSP-on-current-tree is the right tool for this kind of structural review since the planner's whole point is enumerating live call sites in HEAD code, not committed/ingested baseline state. No miss to record.
