# DROP_2 — Plan QA Falsification, Round 2

**Verdict:** fail
**Date:** 2026-05-01

This Round 2 review attacks the revised `workflow/drop_2/PLAN.md`. Round 1 collapsed Unit B into a single atomic droplet (2.7), enumerated cross-package consumers, narrowed acceptance greps to state-machine context, and added `Repository.ListActionItemsByParent` to droplet 2.10. Most Round 1 blockers are addressed in Round 2 — but several blocking counterexamples remain, including one structural compile-break (Repository test fakes) and one same-package race risk that voids Unit-A→Unit-B serialization.

Severity ladder (highest first):
- **F1** — Droplet 2.10 will not compile: `app.Repository` interface gains a method but `internal/app/service_test.go fakeRepo` is not in `Paths:`.
- **F2** — Same-package compile race: 2.7 (Unit B) and 2.5 (Unit A) both edit `internal/adapters/server/common/app_service_adapter_mcp.go` and 2.7's only `blocked_by` is 2.6. The dispatcher (Drop 4+) is free to fire them in parallel.
- **F3** — Droplet 2.7 misses the `extended_tools.go:1339` MCP tool description string `"todo|in_progress|done"` — a state-machine surface that escapes both the file enumeration and the scope-aware grep.
- **F4** — Droplet 2.10 dotted-address resolution is under-specified: `N` / `N.M` / `N.M.K` semantics are not defined (position-based? created-at-ordered? something else?), so the resolver's correctness is unverifiable.
- **F5** — Droplet 2.7 missing `:73` in `internal/domain/kind_capability_test.go` for `RequireChildrenDone` — file IS in scope, but the second reader is not enumerated and would slip if a builder grep'd only the cited lines.

Six nits below — citation drift on file:line cites that don't currently match HEAD; missing files/lines in service_test.go (search/sanitize fixtures); mcp_surface ReindexEmbeddingsResult.Completed callout; `IsValidLifecycleState` exported-vs-unexported casing mistake; `snapshot.go:421` error-message text not enumerated; `capture_test.go:199` debug-message field rename not flagged.

---

## 1. Round 1 Blocker Resolution Summary

| R1 # | R1 Title (paraphrased)                                          | R2 Status              | Notes                                                                                          |
| ---- | --------------------------------------------------------------- | ---------------------- | ---------------------------------------------------------------------------------------------- |
| R1.1 | 2.7 misses cross-package consumers of `StateDone/Progress`       | resolved               | R2 collapsed Unit B into one atomic droplet 2.7 with whole-tree enumeration.                  |
| R1.2 | `ChecklistItem.Done → Complete` misses `action_item.go`/`tui/model.go` | resolved               | R2 enumerates `internal/domain/action_item.go:357` and `internal/tui/model.go:17742`.         |
| R1.3 | 2.9 misses `service.go`, `capture.go`, `config.go` coercion sites | resolved               | All folded into atomic 2.7 with explicit file-line cites.                                     |
| R1.4 | Whole-tree grep regex too broad (catches non-state `"done"` uses) | resolved               | R2 narrowed to scope-aware regexes + per-file checks against state-machine source files.      |
| R1.5 | PLAN.md § 19.2 still names `internal/app/migrations/role_hydration.go` | partially-resolved     | R2 added a Notes paragraph acknowledging the divergence; the parent PLAN.md is not patched.   |
| R1.6 | 2.12 understates Repository data-shape impact                   | resolved (then broken) | R2 added `ListActionItemsByParent` BUT introduced new structural compile-break (see F1).      |
| R1.7 | 2.3 path ambiguity — `action_item_test.go` doesn't exist         | resolved               | R2 commits to `internal/domain/domain_test.go`.                                                |
| R1.8 | 2.10 doesn't confirm `applies_to_json` stays unchanged           | resolved               | R2 adds explicit "untouched fields" line.                                                      |
| R1.9 | `AllowsParentScope` line citation `:225-232` vs `:224-236`       | partially-resolved     | R2 cites the early-return at `:227-229`. Function-span cite still slightly off but acceptable.|
| FAL.1 | Unit B `Paths:` lists omit ≥ 4 production files                  | resolved               | R2 collapse + enumeration covers them.                                                         |
| FAL.2 | 2.7 missing `internal/domain/action_item.go` (same-pkg nit)      | resolved               | Now enumerated.                                                                                |
| FAL.3 | 2.8 missing `internal/tui/thread_mode.go`                        | resolved               | Now enumerated under `internal/tui/`.                                                          |
| FAL.4 | 2.9 missing `capture.go` (second coercion site)                  | resolved               | `canonicalLifecycleState` rewrite at `:296-312` is in 2.7 acceptance.                          |
| FAL.5 | 2.9 missing `internal/app/service.go` defaults                   | resolved               | `defaultStateTemplates()` flip is in 2.7.                                                      |
| FAL.6 | 2.9 missing `attention_capture.go` and `snapshot.go` references  | resolved               | Both enumerated; aggregate counter rename folded in.                                          |
| FAL.7 | 2.9 missing `config.go` and `config_test.go`                     | resolved               | Both in 2.7 with strict-canonical `isKnownLifecycleState`.                                     |
| FAL.8 | 2.9 missing `app_service_adapter_lifecycle_test.go`              | resolved               | Now enumerated; `:716, 721, 912` correctly flagged as free-form not-state.                     |
| FAL.9 | 2.9 missing `extended_tools_test.go`                             | resolved               | Lines `:427, 446, 1114, 2587, 2600` enumerated.                                                |

Ten of eleven Round-1 blockers resolved. R1.5 left as a pointer-nit (acceptable per dev's "patch parent OR document divergence" guidance — R2 chose document). The R1.6 fix introduced a NEW structural blocker — see Finding 2.1 below.

---

## 2. Round 2 Findings

### 2.1 Droplet 2.10 will not compile — `app.Repository` interface gains `ListActionItemsByParent` but `internal/app/service_test.go` fakeRepo is missing from `Paths:`

- **Severity:** blocking
- **Attack vector:** 1 (symbol leak), 6 (Repository contract), 15 (interface implementers).
- **Counterexample evidence (file:line):**
  - `internal/app/ports.go:11-53` — `Repository` interface, the production-side abstraction `Service` (`internal/app/service.go:130 NewService(repo Repository, ...)`) consumes.
  - `internal/app/service_test.go:18 type fakeRepo struct` — implements every method on the `Repository` interface today. Constructed via `newFakeRepo()` and passed to `NewService(...)` in 25+ tests across `internal/app/*_test.go` (per `git grep -n 'newFakeRepo' -- '*.go'`).
  - `internal/app/service_test.go:598 func (f *fakeRepo) GetActionItem(...)` and `:607 func (f *fakeRepo) ListActionItems(...)` — current implementations of the `ActionItem` read methods on the fake.
  - Plan 2.10 `Paths:` line: *"`internal/app/dotted_address.go` (new), `internal/app/dotted_address_test.go` (new), `internal/app/ports.go` (add `ListActionItemsByParent`), `internal/adapters/storage/sqlite/repo.go` (add `ListActionItemsByParent` method on `*Repository`), `internal/adapters/storage/sqlite/repo_test.go`"*. **`internal/app/service_test.go` is not listed.**
  - Adding `ListActionItemsByParent(ctx context.Context, projectID, parentID string) ([]domain.ActionItem, error)` to the `Repository` interface forces every implementer to add it. `*Repository` (sqlite) is in `Paths:` ✓. `fakeRepo` (`internal/app/service_test.go`) is NOT — but it MUST gain the method or `mage test-pkg ./internal/app` fails at compile time with "*fakeRepo does not implement Repository (missing method ListActionItemsByParent)".
- **Why this is blocking:** Plan 2.10 acceptance says *"`mage test-pkg ./internal/app` and `mage test-pkg ./internal/adapters/storage/sqlite` both green"*. Neither will pass without modifying `internal/app/service_test.go`. The acceptance criterion is unachievable with the declared `Paths:`. This is the exact same shape as Round 1's Finding 1 (under-scoped `Paths:` defeats the acceptance grep) but on a different vector — interface contract instead of state literals.
- **Suggested mitigation:** Add `internal/app/service_test.go` to droplet 2.10 `Paths:`. Add an explicit acceptance bullet: *"`fakeRepo` (defined at `service_test.go:18-…`) gains a `ListActionItemsByParent(ctx, projectID, parentID) ([]domain.ActionItem, error)` method that filters its in-memory `tasks` map by `ParentID == parentID && ProjectID == projectID`."* Also verify there are no other `Repository` implementers — `git grep -nE 'func \([a-z]+ \*?[A-Za-z]+\) CreateActionItem\(' -- 'internal/'` shows fakeRepo is the only test fake besides sqlite Repository, but a sweep at build time would confirm.

### 2.2 Same-package compile race — droplets 2.5 (Unit A) and 2.7 (Unit B) both edit `internal/adapters/server/common/app_service_adapter_mcp.go` with no `blocked_by` between them

- **Severity:** blocking
- **Attack vector:** 5 (same-package racing), 4 (`blocked_by` cycle / serialization).
- **Counterexample evidence:**
  - Plan 2.5 `Paths:` includes *"`internal/adapters/server/common/app_service_adapter_mcp.go` (thread `Role` through `CreateActionItem` at `:620` and `UpdateActionItem` at `:661`)"*.
  - Plan 2.7 `Paths:` includes *"`internal/adapters/server/common/app_service_adapter_mcp.go` — rename symbols at `:820, 854, 856`; rewrite `actionItemLifecycleStateForColumnName` at `:849-864` …"*.
  - **Same file, both droplets, both editing.**
  - Plan 2.5 `Blocked by: 2.4`. Plan 2.7 `Blocked by: 2.6`. Plan 2.6 `Blocked by: 2.3`. Plan 2.4 `Blocked by: 2.3`. So after 2.3 lands, 2.4 and 2.6 unblock concurrently. After 2.4 lands, 2.5 unblocks. After 2.6 lands, 2.7 unblocks. **2.5 and 2.7 share no `blocked_by` edge** — under a state-trigger dispatcher (Drop 4+ semantics, but pre-cascade orchestrator can make the same mistake) they may run in parallel.
  - Plan's stated invariant under "Planner" intro (`workflow/drop_2/PLAN.md:32`): *"droplets that share a Go package carry an explicit `Blocked by:` to the prior package-touching droplet — same-package-parallel-edits break each other's compile."* This invariant is enforced WITHIN units but not ACROSS units.
  - Both 2.5 and 2.7 also touch `internal/adapters/server/common/mcp_surface.go` (2.5 adds `Role` field, 2.7 doesn't touch this file directly per its enumeration — but 2.5 + 2.7 both touch `app_service_adapter_mcp.go`, which is enough).
- **Why this is blocking:** If 2.7 lands first, 2.5 still has unrenamed `domain.StateDone`/`StateProgress` references in its target methods (because the file already had them, and `git diff` for 2.5 will show `Role`-threading touching code that 2.7 already renamed) — merge conflict. If 2.5 lands first, 2.7 will rename symbols in code 2.5 just added (e.g., `app.CreateActionItemInput{... Role: ...}` plumbing) — no conflict per se but builder context drift. Worse: under a dispatcher firing them in parallel, the second build's `git pull` rebase will conflict.
- **Suggested mitigation:** Add `2.5` to 2.7's `Blocked by:` (so 2.7's blocked_by becomes `[2.5, 2.6]`, OR more conservatively `2.5` since 2.5 transitively pulls 2.4 and 2.3 in). The "rename → role-promotion ordering" stated in Notes is *symbolic* — the actual constraint is "every Unit-A droplet must finish before Unit B starts." Encode that in `blocked_by`: 2.7 should explicitly block on the LAST Unit-A droplet, and the LAST Unit-A droplet is whichever sibling has no successor in Unit A. Currently {2.4, 2.5, 2.6} are all Unit-A leaves of the dependency graph (2.4 → 2.5; 2.6 has no Unit-A successor). The safe encoding: `2.7 Blocked by: 2.5, 2.6`. (2.5 transitively brings in 2.4 → 2.3; 2.6 brings in 2.3.)

### 2.3 Droplet 2.7 misses `internal/adapters/server/mcpapi/extended_tools.go:1339` MCP tool description string

- **Severity:** blocking
- **Attack vector:** 2 (literal leak), 13 (hidden state-vocabulary surface).
- **Counterexample evidence (file:line):**
  - `internal/adapters/server/mcpapi/extended_tools.go:1339` — `mcp.WithString("state", mcp.Description("Lifecycle state target for operation=move_state (for example: todo|in_progress|done)"))`.
  - The literal `done` inside the description string is part of an MCP tool schema description that LLM agents read. Under strict-canonical, the canonical example must be `"todo|in_progress|complete"` — not `"todo|in_progress|done"`.
  - Plan 2.7 `Paths:` does NOT enumerate `internal/adapters/server/mcpapi/extended_tools.go` (only `extended_tools_test.go`). Plan acceptance grep `git grep -nE 'lifecycle_state.*"done"|lifecycle_state.*"progress"' -- '*.go'` does NOT match because the literal does not contain `lifecycle_state` (the description text is `Lifecycle state target` — capital L, space-separated, no underscore).
  - Plan acceptance grep `git grep -nE '"done"|"progress"|"completed"' <file>` is scoped per-file to *"`internal/domain/workitem.go`, `internal/app/service.go`, `internal/adapters/server/common/app_service_adapter_mcp.go`, `internal/adapters/server/common/capture.go`, `internal/tui/model.go`, `internal/config/config.go`"* — `extended_tools.go` is NOT in this list.
  - The pattern `for example: todo|in_progress|done` is the kind of seed data an LLM caller will COPY from. Leaving `done` in a tool description while every other surface emits `complete` is an external-API contract leak.
- **Why this is blocking:** Plan claim *"Strict-canonical only … No alias tolerance. The rename is a hard cutover — only canonical values (`complete`, `in_progress`) are accepted on every code path"* is contradicted by an MCP tool description that names `done` as the example state. An MCP client following the description will pass `state="done"` and get rejected — strict-canonical is reachable only if the description is also rewritten.
- **Suggested mitigation:** Add `internal/adapters/server/mcpapi/extended_tools.go` to droplet 2.7 `Paths:` AND add an acceptance bullet: *"`extended_tools.go:1339` MCP tool description for `till.action_item(operation=move_state)` reads `(for example: todo|in_progress|complete)` (NOT `done`); same review across every other MCP tool description string mentioning lifecycle states."* Add an additional acceptance grep: `git grep -nE 'todo\|in_progress\|done|todo\|progress\|done' -- 'internal/adapters/server/mcpapi/' returns empty`. (Note the explicit pipe-separator pattern catches help-text and description strings that the existing greps miss.)

### 2.4 Droplet 2.10 dotted-address resolution semantics are under-specified — N.M.K mapping rule is missing

- **Severity:** blocking
- **Attack vector:** 7 (dotted-address ambiguity rules).
- **Counterexample evidence:**
  - Plan 2.10 says: *"`ResolveDottedAddress` accepts these forms: `N` (level-1), `N.M` (level-2), `N.M.K` (level-3) … Returns `ErrDottedAddressAmbiguous` when the path is non-unique (multiple matches at some level)."*
  - But the plan does NOT define what `N` means as a child-index. Possibilities:
    1. **Position-based** — `N` is the action item with `position == N` (or `position == N-1` for 1-indexed). `Repository.ListActionItems` already orders by `(column_id ASC, position ASC)` — but position is column-scoped, not parent-scoped, so two children with same `parent_id` in different columns share the same position. Multiple matches possible.
    2. **Created-at-ordered** — `N` is the Nth child by creation timestamp. But `created_at` is a `time.Time` and two children created in the same microsecond (test fixtures, batch creation, snapshot restore) tie. No tie-breaker defined.
    3. **Insertion-order** — `N` is the Nth child returned by `ListActionItemsByParent` in some implementation-defined order. Plan 2.10 also does NOT specify an ORDER BY clause for `ListActionItemsByParent`.
  - Plan note in `Repository.ListActionItemsByParent` spec: *"returns the list of action items whose `ParentID == parentID` within `projectID`"*. **No ordering specified.** Different SQLite query plans / collation orders could legitimately return rows in different orders on different runs.
  - Plan 2.10's acceptance for ambiguity errors: *"Returns `ErrDottedAddressAmbiguous` when the path is non-unique (multiple matches at some level)"* — but this presumes a notion of "match" that the plan never defines.
- **Why this is blocking:** A builder cannot implement 2.10 correctly without picking a semantic. Two builders given the same plan would write incompatible resolvers. QA cannot verify the resolver because the spec is ambiguous. The resolver call sites in 2.11 (`till action_item get 2.1`) will produce different results based on which interpretation the builder picks, and the dev's expectation (likely some stable-ordered position-or-creation index) is not encoded.
- **Suggested mitigation:** Add an explicit semantic rule to droplet 2.10 acceptance:
  > *"Dotted-address mapping rule: `N` resolves to the Nth child (1-indexed) of the parent in stable order. Stable order is defined as: (1) `position ASC`, then (2) `created_at ASC`, then (3) `id ASC` as final tie-breaker. `ListActionItemsByParent` SQL returns rows with that exact ORDER BY clause. Out-of-range index (`N > len(children)`) returns `ErrDottedAddressNotFound`. Same-position-collision (caller created two siblings with identical position+created_at) returns `ErrDottedAddressAmbiguous`. Test must cover: position-tied children, created_at-tied children (manually fabricated test fixture with identical timestamps), happy-path 1-indexed `N` resolves to expected child."*
  - And: *"`Repository.ListActionItemsByParent` MUST implement that ORDER BY explicitly; SQLite-side test asserts the order on a fixture with mixed positions and timestamps."*

### 2.5 Droplet 2.7 misses `internal/domain/kind_capability_test.go:73` reader for `RequireChildrenDone`

- **Severity:** blocking
- **Attack vector:** 2 (literal/symbol leak), 5 (same-package racing — internal/domain).
- **Counterexample evidence:**
  - `git grep -nE 'RequireChildrenDone' -- 'internal/domain/kind_capability_test.go'` returns:
    - `:35` — `Policy: CompletionPolicy{RequireChildrenDone: true}` (test fixture, plan-cited).
    - `:73` — `if !kind.Template.ActionItemMetadataDefaults.CompletionContract.Policy.RequireChildrenDone {` (assertion reader, NOT plan-cited).
  - Plan 2.7 enumeration for `internal/domain/kind_capability_test.go`: *"rename `RequireChildrenDone:` test fixture at `:35` to `RequireChildrenComplete:`."* Only `:35`.
  - Renaming the field forces compile-break on `:73` if the builder follows the file-line cite literally.
- **Why this is blocking:** It's same-package (`internal/domain`), so `mage test-pkg ./internal/domain` will catch it at compile-time. The file IS in `Paths:`. So a careful builder will fix it. But if the builder treats the file-line cites as authoritative scope and uses targeted edits without compiling, the test would silently break, and acceptance grep `git grep -nE "\bRequireChildrenDone\b" -- '*.go'` returning empty WOULD catch it. So compile + grep both cover it.
- **Why this is still blocking** (downgrade-pending): If both gates catch it, this is technically a citation-completeness nit, not a blocker. **Filed as blocking because** the plan claims *"All file:line cites verified at HEAD via `git grep` for this Round 2 revision"* (`workflow/drop_2/PLAN.md:157`) — but `:73` shows the verification was incomplete. Verification claims that don't match HEAD undermine confidence in the rest of the file:line cites. Multiple other cites also drift (see Finding 2.6 onward) — pattern suggests the verification was selective.
- **Suggested mitigation:** Update plan 2.7 enumeration to `:35, :73`. Re-verify ALL file-line cites in 2.7 with `git grep -n` at HEAD. Append the verified-at-HEAD timestamp + commit SHA to "All file:line cites verified at HEAD via `git grep` for this Round 2 revision" so future readers can audit.

### 2.6 Plan-claim drift — multiple file:line cites in droplet 2.7 don't match HEAD

- **Severity:** nit (each individually) / blocking-pattern (collectively)
- **Attack vector:** 11 (`canonicalLifecycleState` test fallout) + general planner accuracy attack.
- **Counterexample evidence:**
  - `internal/app/service.go:556` — plan claims this is a `domain.StateDone/Progress` reference. Actual at HEAD: `lifecycleState = domain.StateTodo` (unchanged constant). Cite is wrong.
  - `internal/app/service.go:694` — plan claims `domain.StateDone/Progress`. Actual: `restoredState = domain.StateTodo` (unchanged). Cite wrong.
  - `internal/app/service.go:1817` — plan-cited. Actual: `if !ok || state != domain.StateDone {` ✓ correct.
  - `internal/app/service.go:1965-1975` — plan-cited as containing renamable symbol references. Actual span at HEAD has `:1967, 1969` as the only `domain.StateProgress`/`StateDone` references; `:1965, 1971-1973, 1975` are `domain.StateTodo`/`StateFailed`/`StateArchived` (unchanged). The plan-cited range is half-correct.
  - `internal/app/service.go:1873-1881` — plan calls this `defaultStateTemplates()`. Actual: ✓ correct line span.
  - `internal/app/service.go:1922-1955` — plan calls this `normalizeStateID`. Actual: ✓ correct.
  - `internal/app/service.go:1958-1979` — plan calls this `lifecycleStateForColumnID`. Actual: `:1957-1979` is the function span (declaration at `:1958`). Off-by-one cite.
  - `internal/domain/workitem.go:168` — plan calls this `IsValidLifecycleState` (note exported casing). Actual at HEAD: `isValidLifecycleState` (unexported, declaration at `:166`, body `:166-169`). Plan misnames the function and cites the wrong line.
  - `internal/domain/workitem.go:174` — plan calls this `IsTerminalState`. Actual: ✓ correct.
  - `internal/domain/workitem.go:81-85` — plan claims `ChecklistItem.Done bool` at `:81-85`. Actual at HEAD: `ChecklistItem` struct spans `:81-86`, with `Done bool` at `:84`. Slight drift but acceptable.
  - `internal/domain/workitem.go:147-163` — plan calls this `normalize` block. Actual: `normalizeLifecycleState` is at `:148-163`. Slight drift.
  - `internal/domain/workitem.go:89` — plan claims `CompletionPolicy.RequireChildrenDone bool`. Actual: ✓ correct (field declaration is at `:89`).
- **Why this is collectively blocking:** Each individual citation drift is a 1-2 line off-by error a builder would absorb. But the plan's repeated claim *"All file:line cites verified at HEAD via `git grep` for this Round 2 revision"* (`workflow/drop_2/PLAN.md:157`) is contradicted by these examples. A planner asserting verified accuracy that's actually approximate weakens trust in the rest of the cites — including ones the builder would NOT verify (e.g., the `service_test.go` line numbers for state-reference cleanup at `:2467, 3035, 3055, 3065, 3092, 3108, 3186, 3196, 3797, 4573, 4609, 4626, 4693`, which are not all verified in this review).
- **Suggested mitigation:** Replace plan line `workflow/drop_2/PLAN.md:157`'s claim with: *"File:line cites verified at HEAD on 2026-05-01 against commit `<sha>`. Citations marked `~` are approximate (within ±2 lines of the named function). Builder MUST re-verify with `git grep` before mechanical edits."* And actually verify the cites — at minimum, fix `service.go:556` and `service.go:694` (which are wrong) and the `IsValidLifecycleState` capitalization.

### 2.7 Droplet 2.7 missing `internal/app/service_test.go:1561, 1567, 2953` state-literal references

- **Severity:** blocking
- **Attack vector:** 2 (literal leak), 13 (hidden state surface).
- **Counterexample evidence:**
  - Plan 2.7 enumeration for `internal/app/service_test.go` cites: `:2467, 3035, 3055, 3065, 3092, 3108, 3186, 3196, 3797, 4573, 4609, 4626, 4693`.
  - `git grep -nE '"progress"|"done"' -- 'internal/app/service_test.go'` shows ADDITIONAL hits NOT enumerated:
    - `:1561` — `States: []string{"progress"}` (search-state filter input — state-machine surface).
    - `:1567` — `matches[0].StateID != "progress"` (state-machine assertion reader).
    - `:2953` — `if got[0].ID != "progress"` (state-template sanitization assertion).
  - Plan acceptance criterion includes *"full grep sweep required"* — so technically the builder is supposed to find these. But plan also says *"All file:line cites verified at HEAD via `git grep` for this Round 2 revision"*, and these three sites are NOT in the cite list.
- **Why this is blocking:** Combined with Finding 2.6, this is direct evidence the file:line verification claim is incomplete. Plan acceptance greps WILL catch these (`git grep -nE 'lifecycle_state.*"progress"' ...` may not — these are `States: []string{"progress"}` and `StateID != "progress"`, neither matching `lifecycle_state` directly). Specifically: `:1561` is a `[]string{"progress"}` filter parameter, NOT a `lifecycle_state`-tagged literal. Plan grep `git grep -nE 'lifecycle_state.*"done"|lifecycle_state.*"progress"' -- '*.go'` would NOT catch this. Plan grep `git grep -nE '"in-progress"|"doing"' -- 'internal/domain/' 'internal/app/' ...` would NOT catch a bare `"progress"`. **The literal at `:1561` could survive the plan's acceptance gates if the builder doesn't manually grep.**
- **Suggested mitigation:** Add `:1561, 1567, 2953` to plan 2.7's enumeration. Add acceptance grep: `git grep -nE 'States:\s*\[\]string\{[^}]*"(progress|done)"' -- 'internal/'` returns empty. (This catches search-filter literals that the `lifecycle_state.*"X"` grep misses.)

### 2.8 `mcp_surface.ReindexEmbeddingsResult.Completed` field is independent — confirmed, but plan should reference the file:line directly

- **Severity:** nit
- **Attack vector:** 14 (`mcp_surface.Completed` independence claim).
- **Counterexample evidence:**
  - Plan Notes line `B9` (`workflow/drop_2/PLAN.md:325`): *"Confirmed `internal/adapters/server/common/mcp_surface.go:227 Completed bool json:"completed"` is independent of lifecycle state — it's a checklist-item-completed boolean on an MCP response shape, unrelated to `ChecklistItem.Done`."*
  - Actual at HEAD `internal/adapters/server/common/mcp_surface.go:218-229`: the `Completed` field at `:227` lives on `ReindexEmbeddingsResult` (NOT a checklist response — it's an embeddings-reindex result indicating "did the reindex job complete"). The plan's framing "*checklist-item-completed boolean*" is wrong — it's an embeddings-job-completed boolean. The independence conclusion is correct (it has nothing to do with `ChecklistItem` or `LifecycleState`), but the rationale in Notes is misleading.
- **Why nit:** The conclusion (no rename) is right. Rationale gloss is wrong. No code-level impact, but a future reader of Notes would get the wrong mental model.
- **Suggested mitigation:** Replace `B9` text with: *"`mcp_surface.go:227 Completed bool json:"completed"` lives on `ReindexEmbeddingsResult` (embeddings reindex job status). Independent of `ChecklistItem.Done` and of `LifecycleState`. NO rename, NO acceptance criterion in Drop 2 touches this field."*

### 2.9 Droplet 2.7 missing `internal/app/snapshot.go:421` error-message text rewrite

- **Severity:** nit
- **Attack vector:** 2 (literal leak in user-facing error message).
- **Counterexample evidence:**
  - Plan 2.7 enumerates `snapshot.go:419` (the validation switch). Actual `:421` immediately after is: `return fmt.Errorf("tasks[%d].lifecycle_state must be todo|progress|done|failed|archived", i)`.
  - Strict-canonical removes the `progress`/`done` literals from the state-machine domain. The error message lists them as the allowed values — under strict-canonical, the message must list `todo|in_progress|complete|failed|archived`.
  - Plan acceptance grep `git grep -nE 'lifecycle_state.*"done"|lifecycle_state.*"progress"' -- '*.go'` matches: the literal `"tasks[%d].lifecycle_state must be todo|progress|done"` contains both `lifecycle_state` and `"done"` / `"progress"` (string contains the words even though the format is one quoted Go string). So the grep DOES catch this. Builder would fix it via the grep gate.
- **Why nit:** Acceptance gate catches it. But the plan's per-file enumeration cites `:419` only, missing `:421`. A builder reading the plan literally and doing surgical edits at `:419` only — without running the broad grep — could miss it.
- **Suggested mitigation:** Add `:421` to 2.7's `internal/app/snapshot.go` cite list with explicit text rewrite: `"tasks[%d].lifecycle_state must be todo|in_progress|complete|failed|archived"`.

### 2.10 Droplet 2.7 missing `internal/adapters/server/common/capture_test.go:198-202` field-rename impact

- **Severity:** nit
- **Attack vector:** 2 (counter rename leaks into test debug message).
- **Counterexample evidence:**
  - Plan 2.7 enumerates `capture_test.go:111, 136, 268-269, 198` (`WorkOverview.DoneActionItems` rename to `CompleteActionItems`).
  - Actual at HEAD `:199`: `t.Fatalf("WorkOverview counts = %#v, want todo=2 progress=1 done=1 failed=1 archived=1", capture.WorkOverview)`. The format-string contains `progress=1 done=1` debug labels. After 2.7 renames the fields, the format-string labels should also rename to `in_progress=1 complete=1` for consistency. Plan does not flag this.
  - Plan acceptance grep `git grep -nE 'WorkOverview counts.*progress.*done' -- '*.go'` would catch it, but it's not part of the plan's grep set. The narrow-scope `git grep -E '"done"' <file>` for `capture.go` (NOT `capture_test.go`) misses it.
- **Why nit:** The format-string is debug-only. The test still functions if the format-string lies (it just produces confusing failure output). Not a runtime correctness issue.
- **Suggested mitigation:** Add to plan 2.7 acceptance for `capture_test.go`: *"`:199` `t.Fatalf` format-string updated to `\"WorkOverview counts = %#v, want todo=2 in_progress=1 complete=1 failed=1 archived=1\"`."*

### 2.11 Plan acceptance grep doesn't cover slug-prefix in resolver `extended_tools.go` description text

- **Severity:** nit
- **Attack vector:** 8 (mutation rejection completeness for dotted address).
- **Counterexample evidence:**
  - Plan 2.11 says: *"`till.action_item(operation=update, action_item_id="2.1", ...)` returns a 400-class error explaining that mutations require UUIDs."*
  - Plan does not enumerate the action_item tool's `mcp.WithString("action_item_id", ...)` description text at `internal/adapters/server/mcpapi/extended_tools.go:1335`. After 2.11, the description should clarify "UUID for mutations; UUID OR dotted form for `operation=get`". Otherwise an MCP caller reading only the description tries `update 2.1` and gets the rejection error without knowing why.
- **Why nit:** Documentation, not behavior. Tests pass without rewriting the description. But the user-facing contract is now confusing.
- **Suggested mitigation:** Add to plan 2.11: *"`extended_tools.go:1335` description text for `action_item_id` parameter clarifies: `"Action-item identifier. UUID required for mutations (operation=create|update|move|move_state|delete|restore|reparent). UUID or dotted address (e.g. '2.1.3' or 'tillsyn-2.1.3') accepted for operation=get."` (Mirror the same change to the CLI flag help.)*

---

## 3. Acyclic + Same-Package Re-Verification

- **Cycle check (DAG).** Edges: 2.1→ε, 2.2→ε, 2.3→2.2, 2.4→2.3, 2.5→2.4, 2.6→2.3, 2.7→2.6, 2.8→2.7, 2.9→2.8, 2.10→2.9, 2.11→2.10. No cycles. ✓
- **Topological order.** {2.1, 2.2} → {2.3} → {2.4, 2.6} → {2.5 (after 2.4), 2.7 (after 2.6)} → {2.8 (after 2.7)} → {2.9} → {2.10} → {2.11}.
- **Same-package conflict detection.**
  - `internal/domain`: 2.2 → 2.3, 2.7 (transitive via 2.6, 2.3). All serialized. ✓
  - `internal/app`: 2.5 (mcp adapter, in `common` not `app` — actually `common` only), 2.6 (snapshot.go, package app), 2.7 (everywhere), 2.8 (no), 2.9 (snapshot.go doc-comment + repo.go doc-comment, package app), 2.10 (dotted_address.go + ports.go + service_test.go IF added), 2.11 (none directly). Chain serialized through `2.6 → 2.3` and `2.7 → 2.6` and `2.9 → 2.8 → 2.7` and `2.10 → 2.9`. ✓ All serialized.
  - `internal/adapters/storage/sqlite`: 2.4 (schema + scanner), 2.8 (boot-seed payloads), 2.10 (ListActionItemsByParent + doc-comment). Chain: 2.4 → 2.3, 2.8 → 2.7 → ... → 2.4 (transitively), 2.10 → 2.9 → ... → 2.8. ✓ All serialized.
  - **`internal/adapters/server/common`: 2.5 + 2.7 race — see Finding 2.2 above.** This is the only same-package serialization gap.
  - `internal/adapters/server/mcpapi`: 2.5 (`extended_tools.go`), 2.7 (`extended_tools_test.go`), 2.11 (`extended_tools.go`). 2.7 → 2.6, 2.11 → 2.10 → ... → 2.7. So 2.11 is downstream of 2.7. 2.5 and 2.7 — only 2.5 touches the production file `extended_tools.go`; 2.7 only touches `extended_tools_test.go`. Different files, same package — Go compile unit is package-scoped, so editing different files in the same package CAN race. But Plan 2.5's `extended_tools.go` edit is to the `till.action_item` create + update arg parsing (line ~1019, ~1077 area) and 2.7 doesn't touch the production file. So this slot is fine for compilation, but the package-lock invariant is technically loosened. Same-package, different file is OK if neither changes type definitions both files reference. Confirmed safe.
  - `internal/tui`: 2.7 only. ✓
  - `internal/config`: 2.7 only. ✓
  - `cmd/till`: 2.11 only. ✓

**Conclusion:** Same-package serialization is clean across the chain EXCEPT for the 2.5 + 2.7 race in `internal/adapters/server/common/app_service_adapter_mcp.go`. See Finding 2.2.

---

## 4. JSON-Tag Aggregate Counter Sweep

- **Plan claim:** rename `done_tasks → complete_tasks` and `done_items → complete_items`; verify no false-positive renames.
- **Verified hits (all `_tasks` JSON tags in `WorkOverview`):**
  - `:140 total_tasks` — semantic field, NOT state-related, stays.
  - `:141 todo_tasks` — semantic field, stays.
  - `:142 in_progress_tasks` — already canonical, stays.
  - `:143 done_tasks` — RENAME to `complete_tasks` ✓.
  - `:144 failed_tasks` — semantic field, stays.
  - `:145 archived_tasks` — semantic field, stays.
- **Verified hits in `attention_capture.go AttentionWorkOverview`:**
  - `:95 in_progress_items` — already canonical, stays.
  - `:96 done_items` — RENAME to `complete_items` ✓.
- **Other `_items` / `_tasks` JSON tags swept:**
  - `git grep -nE 'json:"[a-z_]+_tasks"' -- '*.go'` shows ONLY the WorkOverview members. No collisions.
  - `git grep -nE 'json:"[a-z_]+_items"' -- '*.go'` shows AttentionWorkOverview's `done_items` plus various `_items` tags on attention/handoff types — none state-related.
- **Conclusion:** No incidental matches. Plan's targeted rename is correct.

---

## 5. RequireChildrenDone JSON-Key Back-Compat

- **Plan claim:** rename `RequireChildrenDone` field + JSON tag `require_children_done` → `RequireChildrenComplete` / `require_children_complete`. No fallback alias.
- **Persisted-snapshot fixture sweep:**
  - `git grep -n 'require_children_done\|require_children_complete' -- '*.go' '*.json' '*.toml'` returns ONLY the Go source declaration at `internal/domain/workitem.go:89`. No JSON / TOML test fixtures contain the legacy key.
  - `git ls-files '*testdata*' '*golden*'` enumerates 11 files (TUI golden text, sample.go, sample.md, ANSI fixtures). None contain JSON state vocabulary.
- **Conclusion:** No persisted-snapshot back-compat surface. Plan's strict-cutover JSON-tag rename is safe. ✓

---

## 6. canonicalLifecycleState Test Rewrite Compatibility

- **Plan claim:** rewrite `canonicalLifecycleState("doing")` test at `capture_test.go:268` to verify rejection.
- **Function signature at HEAD:** `func canonicalLifecycleState(state domain.LifecycleState) domain.LifecycleState` (returns `LifecycleState`, NOT `(LifecycleState, error)`). The current behavior returns the canonical state for known aliases and (per the switch's default branch — let me re-verify) returns the input as-is for unknowns. Under strict-canonical, it should return either an empty `LifecycleState` or the input unchanged for non-canonical inputs.
- **Actual current code at `capture.go:296-312`:**
  ```go
  func canonicalLifecycleState(state domain.LifecycleState) domain.LifecycleState {
      switch normalizeStateLikeID(string(state)) {
      case "todo":
          return domain.StateTodo
      case "progress", "in-progress", "doing":
          return domain.StateProgress
      case "done", "complete", "completed":
          return domain.StateDone
      ...
      }
  }
  ```
- **Plan acceptance:** *"rewrite `canonicalLifecycleState("doing")` test at `:268` to verify rejection (no longer coercion to `StateProgress`)"*. With the function returning `LifecycleState` (no error), "rejection" must mean returning empty (`""`) or the input verbatim. Plan does NOT specify which.
- **Why nit:** Behavior is under-specified at the API level. Caller code at `capture.go:258, 260` does:
  ```go
  case domain.StateProgress:
      overview.InProgressActionItems++
  case domain.StateDone:
      overview.DoneActionItems++
  ```
  If `canonicalLifecycleState("doing")` returns `""` post-rewrite, the switch falls through (no counter increments) — silent under-counting. If it returns `"doing"`, ditto. The strict-canonical claim "fail loud" isn't compatible with a no-error return type unless the caller checks for empty/unchanged.
- **Suggested mitigation:** Add to plan 2.7: *"`canonicalLifecycleState` rewrite: change return type to `(domain.LifecycleState, error)` and propagate `ErrUnknownLifecycleState` on legacy values. Update both call sites at `capture.go:258, 260` to handle the error explicitly (log + continue OR fatal — dev decision)."* OR alternatively: *"Return empty `LifecycleState` on unknown; add an explicit assertion at both call sites that the input was already canonical (panic on empty). Document the contract."*

---

## 7. Resolver Function Test Coverage Gap

- **Plan 2.10 acceptance:** *"Table-driven tests cover: valid `N`, valid `N.M`, valid `N.M.K`, slug-prefixed valid, slug-prefix mismatch, missing path, ambiguous path, malformed inputs (empty, `1.`, `.1`, `1..2`, `abc`, `1.2.3.4.5` deep nesting), UUID input rejected (must use the dotted form OR the caller is expected to skip the resolver)."*
- **Missing test cases:** Position-tied children (Finding 2.4 above), created_at-tied children, depth-1000 (degenerate case — does the resolver bound recursion?), empty parent (level-1) resolves correctly, slug with dashes (e.g., `tillsyn-old-2.1` — does the regex `^([a-z0-9-]+-)?\d+(\.\d+)*$` correctly distinguish slug-vs-numeric prefix?).
- **Specifically attacking the regex `^([a-z0-9-]+-)?\d+(\.\d+)*$`:**
  - Input `1-2.3`: matches `([a-z0-9-]+-)?` greedy on `1-`, then `\d+` on `2`, then `\.\d+` on `.3`. Resolves to slug=`1`, dotted=`2.3`. **Probably wrong** — likely intent was no slug, dotted=`1-2.3`, but `1-2.3` is not a valid dotted form. Test must enforce: `1-2.3` is invalid.
  - Input `tillsyn-2.1.5-foo`: regex requires the entire string to match. `tillsyn-2.1.5-foo` ends in `-foo` so the trailing `(\.\d+)*$` fails. Returns invalid. ✓
  - Input `2`: matches with empty slug. Resolves to level-1 child. ✓
  - Input `0`: matches `\d+`. But `0` as a 1-indexed child position is invalid. Test must reject `0` or treat as unique semantic.
- **Why nit:** Plan's named test cases don't enumerate these edge cases. Builder MAY add them.
- **Suggested mitigation:** Add to plan 2.10 acceptance: *"Resolver regex disambiguation tests: `1-2.3` rejected (ambiguous slug-vs-numeric), `0` rejected (1-indexed, no zeroth child), slug containing digits-and-dashes resolved correctly when distinguishable."*

---

## Verdict Summary

**FAIL.** Five blocking findings (2.1, 2.2, 2.3, 2.4, 2.5+2.7) and six nits (2.6, 2.8, 2.9, 2.10, 2.11, plus 6 and 7 above which are categorically nits).

**Most damaging counterexample:** Finding 2.1 — adding `ListActionItemsByParent` to the `app.Repository` interface in droplet 2.10 forces every implementer (production `*Repository` AND test `*fakeRepo` at `internal/app/service_test.go:18`) to implement the method, but `service_test.go` is not in 2.10's `Paths:`. `mage test-pkg ./internal/app` will fail at compile time. The acceptance criterion is unachievable as scoped. Same shape as Round 1's Finding 1, different vector — interface contract instead of state literals.

**Second-most damaging:** Finding 2.2 — droplets 2.5 and 2.7 both edit `internal/adapters/server/common/app_service_adapter_mcp.go` with no `blocked_by` between them. The unit-boundary serialization invariant ("each unit lands `mage ci` green before the next unit's first droplet starts") is encoded in prose, not in the dependency graph. A dispatcher will fire them in parallel.

**Round 1 → Round 2 progress:** 10 of 11 R1 blockers resolved (R1.5 partially-resolved via Notes acknowledgment). R2 introduces 5 new blockers and 6 nits. The Unit-B-collapse + cross-package enumeration are largely correct — the remaining gaps are at the *interface/contract* layer (R2.1) and the *cross-unit serialization* layer (R2.2), neither of which Round 1 attacked because Round 1 was overwhelmingly focused on Unit B's state-vocab sweep.

**Single fix path:** Address R2.1 (add `service_test.go` to 2.10 `Paths:` + acceptance bullet for `fakeRepo` method addition), R2.2 (add `2.5` to 2.7's `Blocked by:`), R2.3 (add `extended_tools.go` to 2.7 `Paths:` for the description-string fix), R2.4 (define dotted-address `N.M.K` semantics with stable ordering rule), and R2.5 (re-verify file:line cites at HEAD or downgrade the verification claim). Nits R2.6-2.11 + Sections 6-7 may be addressed in the same revision.
