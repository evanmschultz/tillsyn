# DROP_4A WAVE 1 — DOMAIN-FIELD INFRASTRUCTURE

**State:** planning
**Wave:** 1 of 5 (Wave 0 → **Wave 1** → Wave 2 → Wave 3 → Wave 4)
**Brief:** `workflow/drop_4a/REVISION_BRIEF.md` § "Wave 1 — Domain-field infrastructure"
**Output target:** unified `workflow/drop_4a/PLAN.md` after orch synthesis (orch renumbers `Wave 1.k` → `4a.k` globally).

## Wave Purpose And Sequencing

Wave 1 lands the first-class domain-field infrastructure that Wave 2's dispatcher and Wave 3's auth integration consume. Every droplet follows the canonical Drop 3 surgical pattern (proven across 3.17 / 3.18 / 3.20 / 3.21):

```
domain struct field
   ↓ ActionItemInput / ProjectInput field
   ↓ app.{Create,Update}ActionItemInput field
   ↓ SQL CREATE TABLE column (+ index where cross-row reads matter)
   ↓ MCP request/response field (CreateActionItemRequest, UpdateActionItemRequest)
   ↓ MCP wire schema (extended_tools.go)
   ↓ snapshot field (SnapshotActionItem / SnapshotProject)
   ↓ table-driven domain tests + round-trip tests
```

The wave's load-bearing planning observation is **same-file lock chains across droplets**:

- `internal/domain/action_item.go` — touched by Wave 1.1 (`paths`), Wave 1.2 (`packages`), Wave 1.3 (`files`), Wave 1.4 (`start_commit`), Wave 1.5 (`end_commit`), Wave 1.7 (always-on parent-blocks). Six droplets all editing the `ActionItem` struct + `NewActionItem` validation block at adjacent lines. Hard serialization required.
- `internal/adapters/storage/sqlite/repo.go` — `action_items` `CREATE TABLE` (`:168-197` post-Drop-3) gains 4 new columns from Wave 1.1-1.5. Same-file compile lock identical to Drop 3's 3.3 → 3.18 chain.
- `internal/adapters/server/common/mcp_surface.go` — `CreateActionItemRequest` (`:65-107`) and `UpdateActionItemRequest` (`:110-158`) get new fields from Wave 1.1-1.6.
- `internal/adapters/server/common/app_service_adapter_mcp.go` — `CreateActionItem` (`:633-678`), `UpdateActionItem` (`:692-739`), `MoveActionItemState` (`:774-809`) thread the new values; Wave 1.6 modifies `MoveActionItemState`'s `state`-resolution path.
- `internal/app/snapshot.go` — `SnapshotActionItem` (`:57-90`) gains 5 new fields; `SnapshotProject` (`:33-42`) gains 6 new project fields from Wave 1.8.
- `internal/app/service.go` — `defaultStateTemplates` (`:2118-2125`) seeds canonical column names; Wave 1.9 verifies (no edit if already correct).
- `internal/domain/project.go` — Wave 1.8 lone editor; serializes within itself (project-fields-as-bundle decision below).

The chain `paths → packages → files → start_commit → end_commit → state-on-create → always-on parent-blocks` strictly serializes. Wave 1.8 (project fields) is independent of `action_item.go` work and parallelizes against Wave 1.1; Wave 1.9 (column-title verify) is independent of everything and parallelizes broadly.

**Pre-MVP constraints in force:** no migration logic in Go, no `till migrate` CLI, no `ALTER TABLE`. Every droplet that mutates schema notes "**DB action:** dev DELETEs `~/.tillsyn/tillsyn.db` BEFORE `mage ci`" per `feedback_no_migration_logic_pre_mvp.md`. Builders run **opus**. `mage ci` at unit boundaries; `mage test-pkg <pkg>` per droplet. NEVER `mage install`. NEVER raw `go test` / `go build` / `go vet`.

**Cross-wave consumer notes:**

- Wave 2 lock manager reads `paths` + `packages` to acquire file-/package-level locks at the `in_progress` transition.
- Wave 2 dispatcher reads project fields (`hylla_artifact_ref`, `repo_bare_root`, `repo_primary_worktree`, `language`, `build_tool`, `dev_mcp_server_name`) when constructing the agent-spawn invocation.
- Drop 4b consumes `start_commit` / `end_commit` for commit-agent diff context.
- Drop 4.5's TUI file-viewer pane consumes `files` for reference attachments.
- Wave 1.7 (always-on parent-blocks) directly enforces Drop 1's `failed`-state semantics at every parent close attempt.

## Decomposition — 9 Droplets

| Wave | Title (short)                                                  | Same-file lock with                |
| ---- | -------------------------------------------------------------- | ---------------------------------- |
| 1.1  | `paths []string` first-class on `ActionItem`                   | (chain anchor)                     |
| 1.2  | `packages []string` first-class on `ActionItem`                | 1.1                                |
| 1.3  | `files []string` first-class on `ActionItem`                   | 1.2                                |
| 1.4  | `start_commit string` first-class on `ActionItem`              | 1.3                                |
| 1.5  | `end_commit string` first-class on `ActionItem`                | 1.4                                |
| 1.6  | `state` accepted on MCP create + move                          | 1.5 (mcp_surface + adapter)        |
| 1.7  | Always-on parent-blocks; remove `RequireChildrenComplete` bit  | 1.6 (action_item.go validation)    |
| 1.8  | Project-node first-class fields (6 fields)                     | (independent — runs parallel)      |
| 1.9  | Verify default-column titles use post-Drop-2 vocabulary        | (independent — runs parallel)      |

Total: 9 droplets. Topological serialization within `action_item.go` chain; 1.8 + 1.9 parallel.

### Droplet Decomposition

#### Wave 1.1 — `paths []string` first-class on `ActionItem`

- **State:** todo
- **Paths:**
  - `internal/domain/action_item.go` — extend `ActionItem` struct (`:24-102`) with `Paths []string` after `DevGated bool` (`:81`); extend `ActionItemInput` struct (`:105-161`) with the same field after `DevGated` (`:146`); extend `NewActionItem` (`:176-327`) with normalization + validation block; thread into the returned `ActionItem` literal (`:297-326`).
  - `internal/domain/domain_test.go` — table-driven tests for `Paths`.
  - `internal/domain/errors.go` — new `ErrInvalidPaths` sentinel for whitespace-only / duplicate paths.
  - `internal/app/service.go` — `CreateActionItemInput` struct (`:491-538`) gains `Paths []string`; `UpdateActionItemInput` struct (`:540+`) gains `Paths *[]string` (**pointer-sentinel; locked post-4a.5 per Drop 3.21 precedent — nil preserves, non-nil applies; prevents description-only updates from silently clobbering planner-set Paths**); thread through service methods.
  - `internal/adapters/server/common/mcp_surface.go` — `CreateActionItemRequest` (`:65-107`) gains `Paths []string`; `UpdateActionItemRequest` (`:110-158`) gains `Paths *[]string` (pointer-sentinel matches the service-layer Update shape).
  - `internal/adapters/server/common/app_service_adapter_mcp.go` — thread `Paths` through `CreateActionItem` (`:650-673`) and `UpdateActionItem` (`:717-734`).
  - `internal/adapters/server/mcpapi/extended_tools.go` — `mcp.WithArray("paths", ...)` schema attached to `till.action_item` create + update tool definitions; parse the array into `[]string`.
  - `internal/adapters/server/mcpapi/extended_tools_test.go` — round-trip test for `paths`.
  - `internal/adapters/storage/sqlite/repo.go` — `action_items` `CREATE TABLE` gains a `paths_json TEXT NOT NULL DEFAULT '[]'` column appended after `dev_gated`; `scanActionItem` decodes JSON into `[]string`; INSERT + UPDATE encode `[]string` as JSON via `json.Marshal`. **Storage shape decision:** JSON-encoded text column rather than a side-table. Rationale: `paths` is read whole on every action-item read; queries against individual paths are dispatcher-side in-memory after read; SQLite JSON1 functions are not exercised. Keeps the schema flat. (Builder verifies this against the existing `labels` column pattern — same approach.)
  - `internal/adapters/storage/sqlite/repo_test.go` — write-then-read round-trip: `Paths=["a/b/c.go", "d/e/f.go"]`.
  - `internal/app/snapshot.go` — `SnapshotActionItem` (`:57-90`) gains `Paths []string \`json:"paths,omitempty"\`` after `DevGated` (`:69`); thread through `snapshotActionItemFromDomain` and `(t SnapshotActionItem) toDomain()`.
  - `internal/app/snapshot_test.go` — round-trip with non-empty paths.
- **Packages:** `internal/domain`, `internal/app`, `internal/adapters/server/common`, `internal/adapters/server/mcpapi`, `internal/adapters/storage/sqlite`
- **Acceptance:**
  - `ActionItem.Paths` round-trips through domain → app → SQL → MCP → snapshot.
  - Domain validation: `NewActionItem` trims each entry, deduplicates, REJECTS `[]string{"", "  "}` (whitespace-only / empty entries) with `ErrInvalidPaths`. Empty slice (`Paths == nil` or `len(Paths) == 0`) is valid (zero value).
  - Duplicate after trim → silently deduplicated (matches `Labels` normalization pattern at `:493-509`); does NOT reject. **Design rationale:** path duplicates almost always come from copy-paste in agent prompts; rejecting forces agent retries on benign noise. Dedup matches existing list-field convention.
  - Slash-normalization: forward slashes only (Go convention); backslashes rejected with `ErrInvalidPaths`. Matches `git ls-files` output convention.
  - Path validation in domain: trim-only + dedup. **Path-exists check is NOT enforced at domain layer** — paths often refer to files the build droplet will create. Validation is consumer-side (Wave 2 lock manager confirms paths exist when they need to lock; missing files mean nothing to lock yet).
  - Table-driven tests: empty round-trip; single-path round-trip; multi-path round-trip; whitespace-only rejected; duplicates dedup'd; backslash rejected.
  - SQL: write-then-read round-trip preserves order (JSON arrays are ordered in `encoding/json`; insertion order survives).
  - MCP wire: `till.action_item` schema documents `paths` as `array of strings`.
  - `mage test-pkg ./internal/domain && mage test-pkg ./internal/app && mage test-pkg ./internal/adapters/storage/sqlite && mage test-pkg ./internal/adapters/server/common && mage test-pkg ./internal/adapters/server/mcpapi` green.
  - **DB action:** dev DELETEs `~/.tillsyn/tillsyn.db` BEFORE `mage ci` (schema column added).
- **Blocked by:** Wave 0 (dev hygiene infra — pre-commit gating must land first).
- **Notes:** Anchor of the `action_item.go` chain. All subsequent action-item field droplets serialize behind this one. Wave-2 lock manager directly consumes this field — keep semantics tight.

#### Wave 1.2 — `packages []string` first-class on `ActionItem`

- **State:** todo
- **Paths:** identical surface as Wave 1.1 — `internal/domain/action_item.go`, `internal/domain/domain_test.go`, `internal/domain/errors.go`, `internal/app/service.go`, `internal/adapters/server/common/mcp_surface.go`, `internal/adapters/server/common/app_service_adapter_mcp.go`, `internal/adapters/server/mcpapi/extended_tools.go`, `internal/adapters/server/mcpapi/extended_tools_test.go`, `internal/adapters/storage/sqlite/repo.go`, `internal/adapters/storage/sqlite/repo_test.go`, `internal/app/snapshot.go`, `internal/app/snapshot_test.go`.
- **Packages:** identical to Wave 1.1.
- **Acceptance:**
  - `ActionItem.Packages` first-class field, inserted in struct after `Paths`.
  - SQL column `packages_json TEXT NOT NULL DEFAULT '[]'` after `paths_json`.
  - Domain validation: trim, dedup, reject whitespace-only / empty entries with new `ErrInvalidPackages`. Same shape as `Paths`.
  - **Coverage rule (domain-light):** when `Paths` is non-empty, `Packages` MUST also be non-empty. Empty `Packages` while `Paths` is non-empty REJECTS with `ErrInvalidPackages` (specifically: "packages must cover paths"). When `Paths` is empty, `Packages` may be empty.
  - **Coverage strict-check deferred to Wave 2.** Strict path → package mapping (every file in `Paths` must resolve to an entry in `Packages`) requires gopls-aware path resolution that the domain layer doesn't have. Wave 2's lock manager performs the strict check at runtime when files exist; the domain rule today is the simpler "non-empty Packages when non-empty Paths" invariant.
  - Package format: any non-empty trimmed string. No format enforcement (`internal/domain`, `github.com/foo/bar`, both valid). Rationale: enforcement requires a Go-import-path validator; planner-set values matter more than a syntactic check.
  - Round-trip tests parallel Wave 1.1.
  - **Update pointer-sentinel pattern (locked post-4a.5 per Drop 3.21 precedent):** `UpdateActionItemInput.Packages *[]string` and `UpdateActionItemRequest.Packages *[]string` — nil preserves, non-nil applies. Same rationale as 4a.5: prevents description-only updates from silently clobbering planner-set Packages. Same exposed-helper pattern as 4a.5 (`domain.NormalizeActionItemPackages` for shared create/update validation).
  - `mage test-pkg` for all touched packages green.
  - **DB action:** dev DELETEs `~/.tillsyn/tillsyn.db` BEFORE `mage ci`.
- **Blocked by:** **Wave 1.1** (same-file compile lock on `action_item.go` + `mcp_surface.go` + `app_service_adapter_mcp.go` + `extended_tools.go` + `repo.go` + `snapshot.go`; coverage rule references `Paths` so 1.2 reads 1.1's field).
- **Notes:** Wave-2 lock manager performs package-level locking on this field. Strict path-coverage enforcement lives in Wave 2's lock acquire path, not here.

#### Wave 1.3 — `files []string` first-class on `ActionItem`

- **State:** todo
- **Paths:** same surface as Wave 1.1.
- **Packages:** same as Wave 1.1.
- **Acceptance:**
  - `ActionItem.Files` first-class field, inserted after `Packages`.
  - SQL column `files_json TEXT NOT NULL DEFAULT '[]'` after `packages_json`.
  - Domain validation: trim, dedup, reject whitespace-only / empty entries with new `ErrInvalidFiles`. Same shape as `Paths`/`Packages`.
  - **Disjoint-axis rule (domain-light):** `Files` and `Paths` are NOT cross-checked for overlap or disjointness. Rationale: `Paths` declares lock scope (write intent — what the build droplet may edit). `Files` declares reference attachments (read attention — files the agent should look at). The two axes overlap legitimately when an agent edits a file referenced as a viewer — that's the dominant case for Drop 4.5's TUI file-viewer pane. Forcing disjointness would require agents to choose which axis a file belongs to and would prohibit a read-then-edit workflow.
  - **Path-exists validation deferred to consumer.** Domain layer does NOT call `os.Stat` — paths often refer to soon-to-be-created files. Drop 4.5's file-viewer is the canonical consumer; today's minimal validation is `[]string` shape only.
  - Round-trip tests: empty / single / multi / overlap-with-`Paths` (legitimate) / whitespace-only rejected.
  - **Update pointer-sentinel pattern (locked post-4a.5 per Drop 3.21 precedent):** `UpdateActionItemInput.Files *[]string` and `UpdateActionItemRequest.Files *[]string` — nil preserves, non-nil applies. Same rationale as 4a.5/4a.6.
  - `mage test-pkg` for all touched packages green.
  - **DB action:** dev DELETEs `~/.tillsyn/tillsyn.db` BEFORE `mage ci`.
- **Blocked by:** **Wave 1.2** (same-file compile lock on the same six files).
- **Notes:** Drop 4.5 file-viewer consumer.

#### Wave 1.4 — `start_commit string` first-class on `ActionItem`

- **State:** todo
- **Paths:** same surface as Wave 1.1.
- **Packages:** same as Wave 1.1.
- **Acceptance:**
  - `ActionItem.StartCommit string` first-class field, inserted after `Files`.
  - SQL column `start_commit TEXT NOT NULL DEFAULT ''` after `files_json` (string column, not JSON-encoded — single value, not a list).
  - Domain validation: trim. **Hex-format check NOT enforced at domain layer** — git short-SHAs (7-char), full-SHAs (40-char), and untracked-state empty all need to round-trip. Trim-only matches `Owner` + `BlockedReason` precedent (free-form principal-string + reason-string).
  - **Population timing decision: opaque domain field.** Domain layer holds the field opaquely as a string; it does NOT call `git rev-parse HEAD` or know about git at all. Rationale: domain → git would introduce a new external dependency; matches precedent (`Title`, `Description`, `Owner` are all caller-supplied opaque strings). The CALLER (orchestrator pre-cascade; dispatcher in Wave 2; commit-agent in Drop 4b) supplies `StartCommit` at creation time — typically the current `git rev-parse HEAD` of the bare-root or active worktree.
  - Empty `StartCommit` is valid (zero value — not yet captured / not applicable).
  - Round-trip tests: empty / 40-char SHA / 7-char short-SHA / whitespace-trimmed.
  - **Update pointer-sentinel pattern (locked post-4a.5 per Drop 3.21 precedent):** `UpdateActionItemInput.StartCommit *string` and `UpdateActionItemRequest.StartCommit *string` — nil preserves, non-nil applies. Empty-string explicit-clear distinguishable from "not provided." Prevents description-only updates from silently clobbering dispatcher-set start commit hashes.
  - `mage test-pkg` for all touched packages green.
  - **DB action:** dev DELETEs `~/.tillsyn/tillsyn.db` BEFORE `mage ci`.
- **Blocked by:** **Wave 1.3** (same-file compile lock on the same six files).
- **Notes:** Drop 4b commit-agent consumes this for diff context (`git diff <start_commit>..<end_commit>` baseline). Pre-cascade orch sets it manually when creating action items; Wave 2 dispatcher sets it programmatically at `in_progress` transition.

#### Wave 1.5 — `end_commit string` first-class on `ActionItem`

- **State:** todo
- **Paths:** same surface as Wave 1.1.
- **Packages:** same as Wave 1.1.
- **Acceptance:**
  - `ActionItem.EndCommit string` first-class field, inserted after `StartCommit`.
  - SQL column `end_commit TEXT NOT NULL DEFAULT ''` after `start_commit`.
  - Domain validation: trim only — same shape as `StartCommit`.
  - **Population-timing hook:** `SetLifecycleState` (`internal/domain/action_item.go:387-415`) does NOT auto-populate `EndCommit`. Same opaque-domain reasoning as `StartCommit`. The CALLER (Wave 2 dispatcher) populates `EndCommit` AT the terminal-state transition by either: (a) calling `UpdateActionItem` with the dereferenced commit value before `MoveActionItemState`, or (b) extending `MoveActionItemStateRequest` with an optional `EndCommit string` field that propagates through to a service method. **Decision: option (a) — pure domain; minimal MCP surface change in Wave 1.5.** Wave 2's dispatcher takes care of the call ordering. Rationale: keeps MCP wire surface simple; domain field is just-a-field.
  - Empty `EndCommit` valid until terminal state. The pre-cascade orchestrator and Wave 2 dispatcher are responsible for populating it; domain doesn't enforce non-empty-on-terminal.
  - Round-trip tests parallel `StartCommit`.
  - **Update pointer-sentinel pattern (locked post-4a.5 per Drop 3.21 precedent):** `UpdateActionItemInput.EndCommit *string` and `UpdateActionItemRequest.EndCommit *string` — same shape as `StartCommit` (nil preserves, non-nil applies, empty-string explicit-clear distinguishable).
  - `mage test-pkg` for all touched packages green.
  - **DB action:** dev DELETEs `~/.tillsyn/tillsyn.db` BEFORE `mage ci`.
- **Blocked by:** **Wave 1.4** (same-file compile lock on the same six files).
- **Notes:** Drop 4b commit-agent consumes both ends. Domain stays opaque; population is caller's responsibility.

#### Wave 1.6 — `state` accepted on `till.action_item(operation=create|move)`

- **State:** todo
- **Paths:**
  - `internal/adapters/server/common/mcp_surface.go` — `CreateActionItemRequest` (`:65-107`) gains `State string` (currently has only `ColumnID string` at `:99`); `MoveActionItemRequest` (`:160-166`) gains `State string` alongside existing `ToColumnID string` (`:163`). **Both fields stay** — column_id remains in the DB and on the request shape; `state` is the agent-facing alternative.
  - `internal/adapters/server/common/app_service_adapter_mcp.go`:
    - `CreateActionItem` (`:633-678`): when `in.ColumnID` is empty AND `in.State` is non-empty, resolve `in.State` via `resolveActionItemColumnIDForState` (`:884-895`) BEFORE calling `app.CreateActionItem`. When BOTH are empty → REJECT with `ErrInvalidColumnID` (existing behavior). When BOTH are non-empty → REJECT with a new clear error: "specify exactly one of column_id or state, not both" (prevents silent precedence bugs).
    - `MoveActionItem` (`:750-771`): same pattern — when `in.ToColumnID` empty AND `in.State` non-empty, resolve via `resolveActionItemColumnIDForState` BEFORE the move SQL. Both empty → REJECT. Both non-empty → REJECT.
  - `internal/adapters/server/mcpapi/extended_tools.go` — `till.action_item` create + move tool schemas gain `mcp.WithString("state", ...)` (description: `"Lifecycle state — todo|in_progress|complete|failed. Use either state OR column_id, not both."`); existing `column_id` parameter stays but description gets an "Optional; legacy. Prefer state." note. Parse `state` from raw input into `CreateActionItemRequest.State` / `MoveActionItemRequest.State`.
  - `internal/adapters/server/mcpapi/extended_tools_test.go` — 4 new test cases:
    1. Create with `state="todo"` only → SUCCEEDS, item lands in the column whose name maps to `StateTodo`.
    2. Create with `column_id="<id>"` only → SUCCEEDS (existing behavior preserved).
    3. Create with both → REJECTED (clear error).
    4. Create with neither → REJECTED (existing `ErrInvalidColumnID` path).
    Same 4 cases for `move`.
  - **NOTE:** existing `MoveActionItemState` (`:774-809`) already accepts `state` and resolves it server-side — that path is the pre-existing surface and stays untouched. Wave 1.6 is adding `state` to `CreateActionItem` + `MoveActionItem` (column-only path) so agents have ONE consistent surface across all three.
- **Packages:** `internal/adapters/server/common`, `internal/adapters/server/mcpapi`
- **Acceptance:**
  - All 8 test cases (4 × create + 4 × move) pass.
  - `resolveActionItemColumnIDForState` reused unchanged — no new resolver function. Single source of truth for column-resolution stays intact.
  - Existing column_id callers continue to work (back-compat path preserved).
  - MCP tool description on `till.action_item` create/move clearly documents the "state OR column_id, not both" rule.
  - `mage test-pkg ./internal/adapters/server/common && mage test-pkg ./internal/adapters/server/mcpapi` green.
  - **DB action:** NONE (no schema change — purely adapter-layer plumbing).
- **Blocked by:** **Wave 1.5** (same-file compile lock on `mcp_surface.go` + `app_service_adapter_mcp.go` + `extended_tools.go` + `extended_tools_test.go` against Wave 1.1-1.5's edits).
- **Notes:** L1 architectural decision (REVISION_BRIEF §"Locked architectural decisions"). Columns table retirement deferred to Drop 4.5's TUI overhaul. Today the columns table stays; the agent surface just hides it.

#### Wave 1.7 — Always-on parent-blocks-on-failed-child; remove `RequireChildrenComplete` policy bit

- **State:** todo
- **Paths:**
  - `internal/domain/workitem.go` — DELETE `CompletionPolicy` struct (`:106-109`) entirely OR keep the type with an empty struct body for forward compat (builder picks; if deleted, every reference must update). DELETE `Policy CompletionPolicy` field from `CompletionContract` (`:118`). DELETE the JSON tag `json:"policy"`. DELETE `Policy:` initializer in `MergeCompletionContract` (`:387-390`).
  - `internal/domain/action_item.go` — modify `CompletionCriteriaUnmet` (`:434-448`): unconditionally walk children — REMOVE the `if t.Metadata.CompletionContract.Policy.RequireChildrenComplete {` guard at `:437`; the children-walk runs always. The non-archived non-`StateComplete` child check stays, but now also gates on `StateFailed`: a non-archived child in any state OTHER than `StateComplete` adds an unmet-criterion. **Decision: `failed` children also block parent close** (matches Drop 1's always-on rule semantics — `failed` is a terminal-non-success state, not a closeable state).
  - `internal/domain/domain_test.go` — REMOVE `Policy: CompletionPolicy{RequireChildrenComplete: true}` from test fixtures at `:594, :730`; REMOVE the `if !merged.CompletionContract.Policy.RequireChildrenComplete` assertion at `:778-779`. ADD new test cases that pin the always-on behavior:
    - parent with one `complete` child → `CompletionCriteriaUnmet` returns empty (no blockers).
    - parent with one `in_progress` child → returns `["child item ... is not complete"]`.
    - parent with one `failed` child → returns `["child item ... is not complete"]` (failed blocks parent close).
    - parent with one `archived` child → empty (archived children skip per existing rule).
  - `internal/domain/kind_capability_test.go` — `:35` and `:73` `RequireChildrenComplete` test fixtures removed.
  - `internal/app/snapshot.go` — `CompletionContract` snapshot wiring no longer carries the policy field; remove `Policy:` initializers from any `SnapshotCompletionContract` value construction.
  - `internal/app/snapshot_test.go` — remove `Policy:` test fixtures.
  - `internal/adapters/server/common/capture_test.go` — `:126` `RequireChildrenComplete:` fixture removed.
  - `internal/adapters/server/mcpapi/instructions_explainer.go` and `extended_tools.go` — sweep any tool-description text mentioning "policy.require_children_complete"; replace with "always-on parent-blocks-on-incomplete-child".
  - `templates/builtin/default.toml` (or `internal/templates/builtin/default.toml` post-Drop-3 L4) — sweep every default-template payload that sets `require_children_complete: false` or `:true`. REMOVE those lines entirely; the field no longer exists. Builder runs `Grep "require_children_complete" templates/ internal/templates/` to enumerate; each hit removed.
  - **Disclaimer in builder spawn prompt:** the Paths list above is a **starting-point lower bound, not closed**. Builder MUST run `LSP findReferences` on `RequireChildrenComplete` symbol AND `Grep "require_children_complete"` (JSON tag) across the entire repo BEFORE editing. Every hit gets either removed (production code, tests, JSON) or rewritten (doc text).
  - `internal/adapters/storage/sqlite/repo.go` — `metadata_json` is the storage shape for completion contract; old `policy: { require_children_complete: true }` in stored JSON becomes vestigial. **Pre-MVP rule: dev fresh-DBs.** No migration code.
- **Packages:** `internal/domain`, `internal/app`, `internal/adapters/server/common`, `internal/adapters/server/mcpapi`, `internal/templates` (and any other package surfaced by the symbol/string sweep).
- **Acceptance:**
  - `LSP findReferences` on `RequireChildrenComplete` symbol returns 0 hits.
  - `Grep "require_children_complete"` across `*.go *.toml *.json` returns 0 hits in production code (test-data fixtures and PLAN.md history excluded).
  - All four new test cases (complete-child / in_progress-child / failed-child / archived-child) pass.
  - All existing `domain_test.go` tests adjusted to reflect new always-on behavior continue to pass.
  - **Bypass-via-supersede note documented:** the only way to close a parent with a non-complete child becomes `till action_item supersede <child-id>` (post-MVP CLI per REVISION_BRIEF). Until the supersede CLI exists, the dev fresh-DBs or manually mutates the child to `complete`. Worklog records this decision so QA Falsification doesn't flag it as a stuck state.
  - `mage test-pkg ./internal/domain && mage test-pkg ./internal/app && mage test-pkg ./internal/adapters/server/common && mage test-pkg ./internal/adapters/server/mcpapi` green; `mage ci` green.
  - **DB action:** dev DELETEs `~/.tillsyn/tillsyn.db` BEFORE `mage ci` (legacy `policy.require_children_complete` JSON-stored data becomes vestigial; fresh-DB clears it).
- **Blocked by:** **Wave 1.6** (same-file compile lock on `app_service_adapter_mcp.go`'s test file + `mcp_surface.go` shared with 1.6; same-file compile lock on `internal/domain/action_item.go` shared with 1.1-1.5; same-file compile lock on `domain_test.go` shared with 1.1-1.5).
- **Notes:** L2 architectural decision. Per QA Falsification the bypass-via-supersede CLI is post-MVP — Drop 4a does NOT ship the supersede CLI. The "stuck-parent" failure mode that arises when a `failed` child blocks a parent's close is an explicit pre-MVP cost; dev fresh-DBs is the escape valve until supersede lands. Document in droplet description so QA does not flag.

#### Wave 1.8 — Project-node first-class fields (6 fields, bundled)

- **State:** todo
- **Paths:**
  - `internal/domain/project.go` — `Project` struct (`:11-40`) gains six new fields after `Description string` (`:15`):
    - `HyllaArtifactRef string` — Hylla ingest reference (e.g. `github.com/evanmschultz/tillsyn@main`).
    - `RepoBareRoot string` — absolute path to the bare repo (orchestration root, e.g. `/Users/.../hylla/tillsyn/`).
    - `RepoPrimaryWorktree string` — absolute path to the primary worktree (e.g. `/Users/.../hylla/tillsyn/main/`).
    - `Language string` — closed enum: `"go"` | `"fe"` | `""` (empty allowed for un-typed projects pre-bootstrap).
    - `BuildTool string` — free-form string: `"mage"` | `"npm"` | `"yarn"` | `"pnpm"` | etc. No closed enum (build tools proliferate; YAGNI).
    - `DevMcpServerName string` — name of the `claude mcp add` registration for the dev MCP server (per `CONTRIBUTING.md` §"Dev MCP Server Setup").
  - `NewProject` (`:62-83`) extended to accept a new `ProjectInput` struct (or extended argument list — builder picks; `ProjectInput` struct is more readable per `ActionItemInput` precedent at `:105-161`). Trims each field; validates `Language ∈ {"", "go", "fe"}` with new `ErrInvalidLanguage` sentinel; rejects relative paths in `RepoBareRoot` / `RepoPrimaryWorktree` with new `ErrInvalidRepoPath` (must be absolute — `filepath.IsAbs`).
  - `internal/domain/errors.go` — new `ErrInvalidLanguage`, `ErrInvalidRepoPath` sentinels.
  - `internal/domain/domain_test.go` — table-driven tests for each field: empty round-trip; happy-path round-trip; `Language="invalid"` rejected; relative `RepoBareRoot` rejected; whitespace-only collapses to empty.
  - `Project.UpdateDetails` (`:98-113`) extended to accept the six new fields (no pointer-sentinels — the Project surface is admin-driven, not agent-driven; explicit-empty intent is rare).
  - `internal/adapters/storage/sqlite/repo.go` — `projects` `CREATE TABLE` (search for existing definition; LSP locate) gains six new columns appended at end: `hylla_artifact_ref TEXT NOT NULL DEFAULT ''`, `repo_bare_root TEXT NOT NULL DEFAULT ''`, `repo_primary_worktree TEXT NOT NULL DEFAULT ''`, `language TEXT NOT NULL DEFAULT ''`, `build_tool TEXT NOT NULL DEFAULT ''`, `dev_mcp_server_name TEXT NOT NULL DEFAULT ''`. `CreateProject` (`:807-818`) and project-update SQL (`:820-832`) thread all six. `scanProject` decodes them.
  - `internal/adapters/storage/sqlite/repo_test.go` — round-trip test with all six populated.
  - `internal/app/service.go` — `CreateProjectInput` (or equivalent) gains all six fields; `UpdateProjectInput` gains all six. Thread through `Service.CreateProject` and `Service.UpdateProject`.
  - `internal/adapters/server/common/mcp_surface.go` — `CreateProjectRequest` (`:48-53`) and `UpdateProjectRequest` (`:56-62`) gain all six fields.
  - `internal/adapters/server/common/app_service_adapter.go` — thread all six through `CreateProject` + `UpdateProject` adapters.
  - `internal/adapters/server/mcpapi/extended_tools.go` — `till.project` create + update tool schemas gain `mcp.WithString` for each of the six fields. Parse and pass through.
  - `internal/adapters/server/mcpapi/extended_tools_test.go` — round-trip test for project-create with all six fields populated.
  - `internal/app/snapshot.go` — `SnapshotProject` (`:33-42`) gains all six fields with `,omitempty` JSON tags. Thread through snapshot encode + decode.
  - `internal/app/snapshot_test.go` — round-trip test.
- **Packages:** `internal/domain`, `internal/app`, `internal/adapters/storage/sqlite`, `internal/adapters/server/common`, `internal/adapters/server/mcpapi`
- **Acceptance:**
  - All six fields round-trip through domain → app → SQL → MCP → snapshot.
  - `Language` validation rejects values outside `{"", "go", "fe"}` with `ErrInvalidLanguage`.
  - `RepoBareRoot` and `RepoPrimaryWorktree` validation rejects relative paths with `ErrInvalidRepoPath`. Empty allowed (zero value — pre-bootstrap project).
  - `BuildTool` and `DevMcpServerName` are free-form trim-only.
  - `HyllaArtifactRef` is free-form trim-only (URL-shape parsing not enforced — Hylla resolves the ref at ingest time).
  - **Bundled-droplet rationale documented in builder spawn prompt:** six fields shipped together because all six edit `project.go` + `repo.go` + `mcp_surface.go` + `app_service_adapter.go` + `extended_tools.go` + `snapshot.go`. Splitting into six per-field droplets means six serialized passes over the same six files (same-file compile lock would force serialization regardless). One coherent diff is the smaller-risk path. Methodology §2.3 fits ("a single struct-shape extension rippling through every wire surface" generalizes — see Drop 3 droplet 3.21 precedent).
  - `mage test-pkg` for all touched packages green; `mage ci` green.
  - **DB action:** dev DELETEs `~/.tillsyn/tillsyn.db` BEFORE `mage ci` (schema change on `projects` table).
- **Blocked by:** Wave 0 (dev hygiene). **Independent of Wave 1.1-1.7's `action_item.go` chain — runs in parallel.**
- **Notes:** Wave 2 dispatcher reads ALL six fields when constructing the agent-spawn invocation: `RepoPrimaryWorktree` for `cd`, `Language` for the `{lang}-builder-agent` variant, `BuildTool` for the verification target ("mage ci" vs "npm test"), `HyllaArtifactRef` for the artifact ref the agent reads, `DevMcpServerName` for the worktree-specific MCP registration, `RepoBareRoot` for orchestration-root operations.
- **Irreducible:** `true`. Rationale: 6 fields × 6 wire surfaces = 36 coupled edit sites that all share the project-field vocabulary. Per Drop 3 droplet 3.21 precedent — single struct-shape extension across all wire surfaces, splitting per-field doubles the diff churn without reducing risk.

#### Wave 1.9 — Verify default-column titles use post-Drop-2 vocabulary

- **State:** todo
- **Paths:**
  - `internal/app/service.go` — verify `defaultStateTemplates` (`:2118-2125`) returns `{"To Do", "In Progress", "Complete", "Failed"}`. Pre-read at HEAD (post-Drop-3) shows it already does — confirmed in the planner's evidence pass. This droplet is a cross-check, not a code change.
  - `internal/adapters/storage/sqlite/repo.go` — verify `migrateFailedColumn` (`:745-769`) uses `'Failed'` (it does at HEAD).
  - **Builder runs `Grep "['\"]Done['\"]"` and `Grep "Done\":" ` across `internal/` to surface any stray `"Done"` literal in seeding code.** Any hit found → flip to `"Complete"`. Any hit ALREADY flipped → no edit; this is the cross-check droplet.
- **Packages:** none if no edit; `internal/app` and/or `internal/adapters/storage/sqlite` if any drift found.
- **Acceptance:**
  - **Branch A — no drift found:** worklog records `Grep` results showing zero hits of `"Done"` in production seeding code. Droplet closes as "verified clean — no code change." `mage ci` green (smoke-test only). `BUILDER_WORKLOG.md` records the cross-check evidence.
  - **Branch B — drift found:** every stray `"Done"` literal in seeding code flipped to `"Complete"`. Test fixture data may legitimately use `"Done"` as a column-rename test input (per Drop 2 PLAN.md `:114` precedent — that line is intentional and stays). Builder distinguishes seeding code from test inputs.
  - `mage test-pkg ./internal/app && mage test-pkg ./internal/adapters/storage/sqlite` green; `mage ci` green.
  - **DB action:** NONE (verification only; if Branch B hit a seeding bug, dev fresh-DBs at next run anyway per pre-MVP rule).
- **Blocked by:** Wave 0. **Independent of all other Wave 1 droplets — runs in parallel.**
- **Notes:** Drop 2 already canonicalized the column-title vocabulary (per `BUILDER_QA_PROOF.md` line 702 — `defaultStateTemplates()` already returns `"To Do"`, `"In Progress"`, `"Complete"`, `"Failed"` post-Drop-2). This droplet's primary expected outcome is Branch A — confirmation note. Branch B exists as a defensive guard against any post-Drop-2 / Drop-3 drift the planner's read pass missed. **Note `"To Do"` (with space) vs the brief's `"Todo"`** — the brief says "Todo / In Progress / Complete / Failed / Archived"; the actual seeded display name at HEAD is `"To Do"`, which `normalizeStateID` (`internal/app/service.go:2199`) maps to canonical `"todo"`. The `"To Do"` display-name spelling is intentional UI-friendly text; canonical state-ID stays `todo`. This droplet does NOT change the display-name spelling — `"To Do"` is correct. Worklog records the "To Do vs Todo" disambiguation explicitly so QA Falsification doesn't flag the wording mismatch in REVISION_BRIEF.
- **Irreducible:** `true` (cross-check droplet — splitting it has no granularity to extract).

## Cross-Wave Blocker Wiring

The serialization chain within Wave 1:

```
Wave 0 (dev hygiene)
   ↓
Wave 1.1 (paths)
   ↓
Wave 1.2 (packages)
   ↓
Wave 1.3 (files)
   ↓
Wave 1.4 (start_commit)
   ↓
Wave 1.5 (end_commit)
   ↓
Wave 1.6 (state on create+move)
   ↓
Wave 1.7 (always-on parent-blocks)

   ↓ (separately, parallel from Wave 0)
Wave 1.8 (project fields, parallel)
Wave 1.9 (column-title verify, parallel)
   ↓
   (all of Wave 1 closes; Wave 2 begins)
```

Topological sort (one valid order): `1.1 → 1.2 → 1.3 → 1.4 → 1.5 → 1.6 → 1.7` serially, with `1.8` and `1.9` parallel from any point post-Wave-0.

## Test-Strategy Notes

- Every droplet adds round-trip tests at the layer it touches (domain, SQL, MCP, snapshot). No new integration test is required at the wave boundary; `mage ci` green is the wave-close gate.
- Wave 1.7 introduces the only behavior-change test (always-on parent-blocks). All other droplets are purely additive — new fields, new validation paths, no existing-behavior regression.
- Coverage rule: each droplet's added LOC must keep package coverage ≥ 70% (the `mage ci` hard floor per CLAUDE.md "Build Verification").

## Hylla Feedback

None — Hylla wasn't queried for this wave. Per the brief's constraint ("no Hylla calls; use Read / Grep / Glob / LSP"), all evidence came from direct file reads, `rg`, and `LSP` (gopls-backed). The constraint reflects the reality that Drop 3's significant new code may not be re-ingested; LSP + direct reads are the right tool for current-state verification, and the planner respected that. Future planner spawns that DO query Hylla should retain this section per CLAUDE.md "Cascade Ledger + Hylla Feedback."
