# DROP_2 — Plan QA Proof, Round 1

**Verdict:** fail
**Date:** 2026-05-01

This review verifies that `workflow/drop_2/PLAN.md` is evidence-complete, internally consistent, and that every claim in droplet acceptance criteria is supported by current code in `main/`. Findings below are ordered by severity (`blocking` first, then `nit`). Each cites code evidence and proposes a fix.

---

## 1. Droplet 2.7 `Paths:` line is missing every cross-package consumer of `StateDone` / `StateProgress` — the rename is a tree-wide compile break, not a same-package edit

**Severity:** blocking

**Evidence.** Droplet 2.7 lists `Paths: internal/domain/workitem.go ..., internal/domain/workitem_test.go and any other internal/domain/*_test.go files`. Renaming the constants `StateDone → StateComplete` and `StateProgress → StateInProgress` in `internal/domain/workitem.go:18-19` breaks compilation of every consumer package. `git grep "StateDone\|StateProgress" -- '*.go'` returns hits across:

- `internal/adapters/server/common/app_service_adapter_mcp.go:820, 854, 856`
- `internal/adapters/server/common/app_service_adapter_lifecycle_test.go:189-190`
- `internal/adapters/server/common/capture.go:258, 260, 302, 304`
- `internal/adapters/server/common/capture_test.go:111, 136, 268`
- `internal/adapters/server/mcpapi/extended_tools_test.go:427, 446`
- `internal/app/attention_capture.go:350, 353, 371`
- `internal/app/attention_capture_test.go:272, 377-378`
- `internal/app/service.go:623, 627, 639, 644, 1817, 1967, 1969`
- `internal/app/service_test.go:3035, 3055, 3065, 3092, 3108, 3186, 3196, 4573, 4609, 4626, 4693, 3797`
- `internal/app/snapshot.go:419`
- `internal/domain/action_item.go:268`

The 2.7 acceptance line `git grep "StateDone\b" returns empty` cannot be satisfied without editing every file above. The same-package serialization rule (`2.7 → 2.8 → 2.9` per Unit B intro) does not capture this — 2.7 leaves `internal/app`, `internal/adapters/server/common`, `internal/adapters/server/mcpapi`, and `internal/domain/action_item.go` un-compilable until 2.9 closes. `mage test-pkg ./internal/domain` in 2.7's acceptance is locally satisfiable but `mage build ./...` is not until 2.9.

**Suggested fix.** Either (a) expand droplet 2.7's `Paths:` to enumerate every consumer file (and rename the droplet "Domain rename + tree-wide consumer fix"), OR (b) split the rename across two droplets — first add new constants `StateComplete = "complete"`/`StateInProgress = "in_progress"` alongside the old `StateDone`/`StateProgress`, sweep every consumer to the new names in subsequent droplets, then delete the old constants in the final unit-B droplet. Option (b) preserves the green-`mage ci`-per-droplet invariant; option (a) accepts that 2.7 is necessarily a single very-large droplet. Either way, the current spec — narrow `Paths:` plus tree-wide grep-empty acceptance — is contradictory.

---

## 2. Droplet 2.7 `ChecklistItem.Done → Complete` field rename misses production consumers in `internal/domain/action_item.go` and `internal/tui/model.go`

**Severity:** blocking

**Evidence.** Droplet 2.7's `Paths:` line scopes only `internal/domain/workitem.go` and `internal/domain/*_test.go` for the field rename. But `git grep "\.Done\b" -- 'internal/'` shows production references at:

- `internal/domain/action_item.go:357` — `if item.Done {` (production code)
- `internal/tui/model.go:17742` — `if !item.Done {` (production code)

Renaming the field `Done → Complete` on `ChecklistItem` (defined at `internal/domain/workitem.go:84`) compile-breaks both files. Test fixture references in `internal/domain/domain_test.go:275, 536` (`Done: true`/`Done: false`), `internal/app/service_test.go:3003, 3038, 4612`, `internal/domain/kind_capability_test.go:19`, and `internal/app/kind_capability_test.go:428` also need updating. None are listed in 2.7's `Paths:`.

The acceptance grep `git grep "ChecklistItem.*Done bool\|\.Done = true\|\.Done = false"` is also poorly constructed: it does not match the dominant usage pattern `Done: true` (struct-literal field syntax uses colon, not equals) nor bare conditionals like `if item.Done`. The grep would falsely report success while leaving real `.Done` references in place.

**Suggested fix.** Add `internal/domain/action_item.go`, `internal/tui/model.go`, `internal/app/service_test.go`, `internal/app/kind_capability_test.go`, `internal/domain/kind_capability_test.go`, `internal/adapters/server/common/capture_test.go` to the `Paths:` line (or re-frame 2.7 per Finding 1's option (b)). Replace the broken grep with a sound one — e.g., `git grep -nE "ChecklistItem\b.*\bDone\b|\.Done([^a-zA-Z]|$)" -- '*.go'` and confirm only the renamed `Complete` form remains.

---

## 3. Droplet 2.9 `Paths:` line is missing every duplicate state-coercion site outside `app_service_adapter_mcp.go`

**Severity:** blocking

**Evidence.** Droplet 2.9 names only `internal/adapters/server/common/app_service_adapter_mcp.go` (rewriting `actionItemLifecycleStateForColumnName` at `:849-864` and `normalizeStateLikeID` at `:866-901`) and `cmd/till/main.go`. But the legacy alias-coercion logic is duplicated across multiple files — the strict-canonical claim breaks at runtime if any of these are missed:

- `internal/app/service.go:1922-1955` — `normalizeStateID()` switch covers `"in-progress"|"progress"|"doing"` → `"progress"` and `"done"|"complete"|"completed"` → `"done"`. This is the **column-name normalizer** used by `defaultStateTemplates()` (`:1873-1881`) and `lifecycleStateForColumnID()` (`:1957-1979`). Live production code path.
- `internal/app/service.go:1873-1881` — `defaultStateTemplates()` seeds new projects with column IDs `"progress"` and `"done"`. Once the rename lands, new projects must seed `"in_progress"` and `"complete"` or there is no column whose ID matches the canonical state.
- `internal/adapters/server/common/capture.go:296-312` — `canonicalLifecycleState()` switch with `"progress"|"in-progress"|"doing"` and `"done"|"complete"|"completed"` aliases. Live production code path used by `CaptureStateService`.
- `internal/config/config.go:218` — `Search.States` default `[]string{"todo", "progress", "done"}`.
- `internal/config/config.go:550` — same default in `defaultUserConfig`.
- `internal/config/config.go:1094` — `isKnownLifecycleState` validator allowlist `"todo", "progress", "done", "failed", "archived"`. After the rename this validator rejects every canonical search-state input.
- `internal/tui/model.go:17945-17967` — `normalizeColumnStateID()` (TUI's own copy of the alias normalizer).
- `internal/tui/model.go:17971-17985` — `lifecycleStateForColumnName()` switch on `"progress"`/`"done"`.
- `internal/tui/model.go:13687-13697` and `:14146-14156` — switches on `actionItem.LifecycleState` literal values.
- `internal/tui/model.go:18012-18029` — `lifecycleStateLabel()` reading `canonicalSearchStateLabels["done"]`/`["progress"]`.
- `internal/tui/model_test.go:685-686, 967-969, 5737, 13549` — test helpers and assertions on legacy literals.

The 2.9 acceptance grep `git grep -E '"done"' -- '*.go'` returning empty across the whole tree is unachievable without sweeping every site above. The `internal/tui/` paths fall under droplet 2.8's `Paths` line (which names `internal/tui/model.go` and `internal/tui/options.go` plus `internal/tui/*_test.go`), but `internal/app/service.go`, `internal/adapters/server/common/capture.go`, `internal/config/config.go`, and `internal/config/config_test.go` are not named in any droplet.

**Suggested fix.** Expand droplet 2.9's `Paths:` to include `internal/app/service.go`, `internal/adapters/server/common/capture.go`, `internal/config/config.go`, `internal/config/config_test.go`, and any other file the whole-tree grep currently catches. Alternatively, split a new droplet 2.9b solely for `internal/app` + `internal/config` coverage. The `defaultStateTemplates()` change at `internal/app/service.go:1877-1878` is particularly load-bearing — it seeds new-project columns by ID; the state→column resolver in `app_service_adapter_mcp.go:805-816` matches columns by *name* (so column IDs can drift), but consumers may also resolve by ID. Verify the data-shape impact before locking the spec.

---

## 4. Droplet 2.9's whole-tree grep regex is too broad — falsely flags non-state uses of `"done"` / `"completed"`

**Severity:** blocking

**Evidence.** Droplet 2.9 acceptance: `git grep -E '"done"' -- '*.go'` and `git grep -E '"completed"' -- '*.go'` ALL return empty across the whole tree. But `git grep -nE '"done"|"completed"' -- '*.go'` shows several non-state uses:

- `cmd/till/embeddings_cli.go:242` — `status = "completed"` (embedding-job status, NOT a lifecycle state).
- `internal/adapters/server/common/app_service_adapter_lifecycle_test.go:716, 721, 912` — `Reason: "done"`, `RevokedReason: "done"` (capability-lease revoke reason — free-form string, NOT a state).
- `internal/adapters/server/common/app_service_adapter_lifecycle_test.go:180` — `State: "done"` (legitimate state, must be renamed).
- `internal/adapters/server/common/mcp_surface.go:227` — field tag `Completed bool ` json:"completed"` (semantic field, distinct from lifecycle state).
- `internal/adapters/server/mcpapi/handler_integration_test.go:380, 405` — `resolution_note: "done"` (resolution note text, NOT a state).
- `internal/adapters/storage/sqlite/repo_test.go:1749` — `Reason: "done"` (capability-lease revoke reason).
- `internal/app/capability_inventory_test.go:50` — `revoked.Revoke("done", ...)` (revoke reason).
- `internal/domain/comment_test.go:95` — `BodyMarkdown: "done"` (comment body text).
- `internal/domain/handoff_test.go:64` — `Summary: "done"` (handoff summary text).

The unconstrained regex `"done"` also matches `:done` keys, error messages, comments, and unrelated string literals. The acceptance criterion as written cannot be satisfied even after a perfect strict-canonical rename — at least nine non-state hits will remain.

**Suggested fix.** Tighten the acceptance grep to either (a) a context-aware regex (e.g., `git grep -nE 'LifecycleState|state.*[:=].*"done"' -- '*.go'` plus targeted call-site checks), OR (b) explicitly enumerate the legitimate non-state hits and assert "the only remaining hits are these N free-form-string uses." Otherwise build-QA will either falsely fail the gate or paper over real misses.

---

## 5. PLAN.md § 19.2 directly conflicts with the orchestrator's "no migration logic in Go code" hard constraint

**Severity:** blocking

**Evidence.** `main/PLAN.md:1602` (the source of truth this drop decomposes from) contains:

> Migration is two droplets: **D1** — pure parser + tests in `internal/domain/role.go` (regex `^Role:\s*([a-z-]+)\s*$` per line, returns first match or empty, rejects unknown values against the closed enum, ~50 LOC + table-driven tests). **D2** — migration runner in `internal/app/migrations/role_hydration.go`: iterate every action_item, call D1 on description, write `metadata.role` if found and currently empty (idempotent re-runs OK), log a warning per row with no parsable Role line, fail-fast on rows with parsable-but-unknown role values.

Drop 2 hard constraints (orchestrator prompt §"Drop 2 Hard Constraints") explicitly forbid this:

> No migration logic in Go code. No `internal/app/migrations/`, no `till migrate` CLI subcommand, no one-shot SQL scripts, no boot-time migration registry. Dev fresh-DBs after schema or state-vocab changes.

`workflow/drop_2/PLAN.md` correctly drops the D2 migration runner and replaces it with "Pre-MVP rule: dev deletes `~/.tillsyn/tillsyn.db`" at Scope item 1. The drop_2 plan honors the constraint. **However**, `main/PLAN.md` § 19.2 still carries the `internal/app/migrations/role_hydration.go` text. Future drops or any reader trusting `main/PLAN.md` § 19.2 as canonical will reintroduce migration logic.

This is not a bug in `workflow/drop_2/PLAN.md` *per se* — the drop plan is correct on the constraint. But the parent PLAN.md disagrees with the executable plan, which is itself a planning gap: a downstream reader who consults § 19.2 (the plan it claims to implement) will see migration logic green-lit.

**Suggested fix.** Either patch `main/PLAN.md:1602` to drop the D2 migration runner reference (in favor of the dev-fresh-DB approach), OR explicitly call out the divergence in `workflow/drop_2/PLAN.md`'s `## Notes` ("PLAN.md § 19.2 D2 supplanted by Pre-MVP rule — no migration runner ships"). The latter is faster and survives at the artifact level. The former is the durable fix. Prefer the latter for Drop 2 closure; queue the former as a follow-up.

---

## 6. Droplet 2.12 understates the data-shape impact of the resolver against the existing `Repository` interface

**Severity:** nit

**Evidence.** Droplet 2.12 says: "where `Repository` is the existing app-layer interface that already exposes the action-item read methods; consumer-side interface, not a new abstraction." The `Repository` interface at `internal/app/ports.go:11-53` exposes `GetActionItem(ctx, id)` and `ListActionItems(ctx, projectID, includeArchived)` — but no list-children-by-parent operation. Resolving `N.M.K` requires walking the parent→child tree level by level. With only `ListActionItems(projectID, ...)`, the resolver must pull every action item in the project and filter by `ParentID` in memory. For projects with thousands of action items this is O(depth × N) per resolve and blocks UI responsiveness.

The plan's "consumer-side interface, not a new abstraction" claim is therefore borderline — the resolver either (a) accepts an O(N)-per-call cost, (b) introduces a new method on `Repository` (e.g. `ListActionItemsByParent(ctx, projectID, parentID)`), or (c) builds a per-call cache. None is mentioned.

**Suggested fix.** Add a single bullet to droplet 2.12's acceptance: "Resolver MAY add `ListActionItemsByParent` to `Repository` if needed for performance; otherwise documents the in-memory-walk approach explicitly. Decide before implementation, justify in commit message." Builder picks the strategy at build time.

---

## 7. Droplet 2.3 path ambiguity — `internal/domain/action_item_test.go` does not exist

**Severity:** nit

**Evidence.** Droplet 2.3's `Paths:` line says "`internal/domain/action_item_test.go` or `internal/domain/domain_test.go` (extend existing `NewActionItem` table-driven tests)". `find internal/domain -name 'action_item_test.go'` returns empty. Only `internal/domain/domain_test.go` exists today.

**Suggested fix.** Drop the disjunction; commit to `internal/domain/domain_test.go` (or, if the builder prefers a clean per-file split, name `internal/domain/action_item_test.go (new)` explicitly).

---

## 8. Droplet 2.10 acceptance counts 12 rows but does not confirm `applies_to_json` stays unchanged

**Severity:** nit

**Evidence.** Droplet 2.10 says: "All 12 `INSERT OR IGNORE INTO kind_catalog` payloads carry `allowed_parent_scopes_json = '[]'`." The current rows at `internal/adapters/storage/sqlite/repo.go:304-375` (verified row-by-row) seed each kind with `applies_to_json = '["<kind>"]'` and `allowed_parent_scopes_json = '["plan"]'` (or `'["build"]'` for the two `build-qa-*` rows). The droplet flips `allowed_parent_scopes_json` to `'[]'` but says nothing about `applies_to_json`. Builder may inadvertently flatten that too.

**Suggested fix.** Add an explicit "untouched fields" line to 2.10 acceptance: "`applies_to_json` and every other column remain unchanged; only `allowed_parent_scopes_json` flips to `'[]'`." Trivial but prevents over-edit.

---

## 9. `AllowsParentScope` line citation is approximate (`:225-232` vs. actual `:224-236`)

**Severity:** nit

**Evidence.** Drop_2 PLAN.md and `main/PLAN.md` § 19.2 both cite `internal/app/kind_capability.go:566` (verified ✓) and `internal/domain/kind.go:225-232` for `AllowsParentScope`. Actual span at HEAD is `:224-236` — `:224` is the doc-comment, `:225` the func declaration, `:227-229` the empty-list early return, `:236` the closing brace. The cited range `:225-232` covers the declaration through the loop body's end-brace but truncates the function body. Not a blocker — close enough that a builder lands the right code — but the exact `:227-229` "early-return on empty" claim is the load-bearing bit and should be cited as such.

**Suggested fix.** Replace the `:225-232` range with `:224-236` (whole function) plus an explicit "the empty-list early return at `:227-229` is the universal-allow mechanism the plan relies on." Same change in droplet 2.10's acceptance line referencing `internal/domain/kind.go:225-232`.

---

## 10. MD-cleanup carve-out is sufficiently bounded

**Severity:** —

**Evidence.** Section "Deferrals to later drops" → "Future refinement drop — MD content cleanup" at `workflow/drop_2/PLAN.md:299` carves out: *"builders MAY make trivial in-section MD edits adjacent to their droplet's `paths` if the change is a single-sentence / single-phrase fix obviously broken by their code change. Build-QA (proof + falsification) MUST verify any MD edits via `git diff` and confirm correctness. Whole-document or whole-section MD sweeps remain out of scope here."* Plus "this carve-out does not establish a precedent."

This is sufficiently narrow — single-sentence/single-phrase scope, explicit Build-QA gate, explicit no-precedent disclaimer. No finding.

---

## 11. No alias-tolerance leaks elsewhere in the plan

**Severity:** —

**Evidence.** Re-read `workflow/drop_2/PLAN.md` for any phrase contradicting strict-canonical. Found none. The plan repeatedly emphasizes: "Strict-canonical only", "input `done`/`progress`/`completed`/`in-progress`/`doing` returns the unknown-state error path (NOT coerced)", "JSON unmarshal accepts ONLY `complete` — `done` keys produce a decode error (no fallback alias)", and Notes line 281 reaffirms "no `UnmarshalJSON` shim". No drift. No finding.

---

## 12. Blocked-by chain is acyclic

**Severity:** —

**Evidence.** Walking the chain from the plan: `2.1 → ε`, `2.2 → ε`, `2.3 → 2.2`, `2.4 → 2.3`, `2.5 → 2.4`, `2.6 → 2.3`, `2.7 → 2.6`, `2.8 → 2.7`, `2.9 → 2.8`, `2.10 → 2.9`, `2.11 → 2.10`, `2.12 → 2.11`, `2.13 → 2.12`. Each edge points to a strictly-earlier droplet. No cycles. No finding.

---

## 13. Same-package serialization correctly enforced — except where Findings 1–3 apply

**Severity:** —

**Evidence.** Same-package pairs that share a Go package and serialize correctly:

- 2.2 + 2.3 share `internal/domain` → 2.3 blocked by 2.2 ✓
- 2.6 (`internal/app`) does not collide with 2.5 (`internal/adapters/server/{common,mcpapi}`) ✓
- 2.10 + 2.11 are in different packages — but 2.11 doc-comment-edits `internal/adapters/storage/sqlite/repo.go:300` (the same file 2.10 edits its boot-seed in). This is a same-file edit serialized 2.10 → 2.11 ✓.
- 2.12 + 2.13 — 2.12 owns `internal/app`, 2.13 owns `internal/adapters/server/{common,mcpapi}` + `cmd/till`. Disjoint ✓.

The Unit-B compile-break problem (Findings 1–3) is the only same-package issue: 2.7 / 2.8 / 2.9 all touch the live state vocabulary across multiple packages, and the plan's `Paths:` lines do not capture the cross-package consumers. That is filed under Findings 1–3, not duplicated here.

---

## 14. Mutation-path enumeration in droplet 2.13 is complete

**Severity:** —

**Evidence.** Droplet 2.13 acceptance says: "mutation operations `create|update|move|move_state|delete|restore|reparent` reject dotted form with a clear error." Cross-checking against `internal/adapters/server/mcpapi/extended_tools.go` for `till.action_item` operations:

- `:1390` — create (alias)
- `:1416` — update (alias)
- `:1438` — move (alias)
- `:1339, :1456` — move_state, delete
- `:1473` — restore
- `:1489` — reparent

Plus the unified operation dispatch on the `till.action_item` tool itself. All seven mutation operations are enumerated. Read-only operations are `get` and `list`. No finding.

---

## 15. Template-loader-coupling investigation is sound

**Severity:** —

**Evidence.** `templates/embed.go` is the only Go file in the `templates/` package (verified via `ls templates/`). Uses `//go:embed builtin/*.json`. `git grep "evanmschultz/tillsyn/templates" -- '*.go'` returns empty across the whole tree — zero importers. Boot-seed of `kind_catalog` is inline SQL at `internal/adapters/storage/sqlite/repo.go:304-375`, NOT template-loaded. `instructions_tool.go` references `default-go` only as instructional prose. The plan's Unit B-zero deletion (entire `templates/` package) is correctly scoped. No finding.

---

## 16. `AllowedParentKinds` zero-callers claim is sound

**Severity:** —

**Evidence.** `git grep "AllowedParentKinds" -- '*.go'` returns:

- `internal/domain/kind.go:94, 99` (the function being deleted)
- `internal/domain/domain_test.go:680, 684, 703-714` (test fixture being deleted)
- `internal/app/snapshot.go:448` (doc-comment to update)
- `internal/adapters/storage/sqlite/repo.go:300` (doc-comment to update)

Zero production callers. Plan claim verified. No finding.

---

## Verdict Summary

**FAIL.** Five blocking findings (1, 2, 3, 4, 5) and three nits (6, 7, 8, 9). The plan honors the strict-canonical state-vocabulary decision in spirit — every coercion site the plan names is correctly framed — but the `Paths:` lines for droplets 2.7 / 2.8 / 2.9 fail to enumerate every cross-package consumer of `StateDone` / `StateProgress` and the `ChecklistItem.Done` field, and the whole-tree grep acceptance criteria are unachievable as written (Findings 1–4). The plan must either expand the `Paths:` lines or re-frame the rename as a multi-phase add-then-delete sequence before builders fire.

The Unit-B-zero template deletion, Unit-C catalog flip + `AllowedParentKinds` deletion, and Unit-D resolver are all soundly scoped — the rot is concentrated in Unit B's strict-canonical sweep.

Finding 5 (PLAN.md § 19.2 still names `internal/app/migrations/role_hydration.go`) is recorded as a blocker against drift — drop_2 PLAN.md is correct on the constraint, but the parent document a builder might consult disagrees. Either patch the parent or annotate the divergence in drop_2's `## Notes`.
