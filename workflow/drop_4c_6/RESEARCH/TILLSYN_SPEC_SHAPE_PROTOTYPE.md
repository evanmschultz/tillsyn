# Tillsyn-Flavored Spec Shape Prototype

Read-only research deliverable for Drop 4c.6 sketch §11 pending update #1, second pass. Concrete, copy-pasteable examples of how SDD concepts map onto Tillsyn's existing primitives — **without** introducing new schema, new files, or new validation. The dev's framing is load-bearing:

> "**NOT use sdd as it is explained online**, just inspired by and adapted for our actual strengths that come with tillsyn."
>
> "**there IS NOT supposed to be a separate file.** we just use the concept of sdd for the plans... a drop includes specs on what is going into the project, or what is being changed, or whatever, NOT the WHOLE project. the specs aren't supposed to be persisted files as a source of truth, **it is a way of structuring prompts**."
>
> "even small tiny drops need it, it just wouldn't be a project's specs, it would be the specs for the specific thing at hand, even for droplets, **it would be there for the one or two code blocks the builder will do**. it is just a small key-value thing that says what it is to do, some kind of pseudo code maybe, fields needed, so on."

So the spec is **a prompt-shaping convention applied to action-item description + metadata**, sized to the work. No `workflow/specs/<id>.md` file. No project-wide spec. No round-trip code regeneration.

---

## 1. Available Primitives — What We're Building Out Of

Verified against `internal/domain/action_item.go` and `internal/domain/workitem.go` at HEAD. Every example below uses ONLY these fields. **No proposed schema changes anywhere in this document.**

### `ActionItem` (top-level)

| Field | Type | What it carries in spec usage |
|---|---|---|
| `Title` | string (UPPERCASE per project memory) | The spec's headline — what this droplet is. |
| `Description` | string (markdown) | **Primary spec body.** Stable section headings, prose + pseudocode. |
| `Kind` | closed 12-value enum | What work — `build`, `plan`, `research`, etc. |
| `Role` | closed enum / empty | Who does it. `builder`, `qa-proof`, `qa-falsification`, `planner`, `research`. |
| `StructuralType` | closed 4-value enum | Where in the cascade — `drop` / `segment` / `confluence` / `droplet`. |
| `Irreducible` | bool | Tiny-leaf marker; `true` for single-fn-signature / single-SQL / single-template-edit. |
| `Owner` | string | `STEWARD` for persistent rollups; empty for the rest. |
| `DropNumber` | int | `4` for drop_4, `7` for drop_7, etc. |
| `Persistent` / `DevGated` | bool | Anchor / human-verify markers. |
| `Paths` | []string | **Write scope** (lock domain). What this droplet edits. |
| `Packages` | []string | Go-package lock domain — packages-must-cover-paths invariant. |
| `Files` | []string | **Read attention** — reference material the agent should skim, distinct from `Paths`. |
| `StartCommit` / `EndCommit` | string | Opaque git SHAs captured at lifecycle transitions. |
| `Labels` | []string | Free-form tags (lowercased, deduped). Useful as a thin classification axis. |
| `Priority` | enum | `low` / `medium` / `high`. |
| `Metadata` | `ActionItemMetadata` | The structured-spec carrier — see below. |

### `ActionItemMetadata` (structured spec carrier)

These are the load-bearing fields for spec content:

| Field | Type | What it carries in spec usage |
|---|---|---|
| `Objective` | string | The "why" — one paragraph of intent. |
| `ImplementationNotesUser` | string | Dev-authored guidance ("nuance the planner can't see"). |
| `ImplementationNotesAgent` | string | Agent-facing prose — the planner's own notes to the builder. |
| `AcceptanceCriteria` | string | Prose criteria block (separate from the structured `CompletionCriteria` checklist). |
| `DefinitionOfDone` | string | Higher-level "this droplet is done when…" prose. |
| `ValidationPlan` | string | How QA + tests confirm the criteria. Specific mage targets, expected outputs. |
| `RiskNotes` | string | Where this is most likely to break behavior, perf, contracts. |
| `CommandSnippets` | []string | Copy-pasteable shell — `mage check`, `mage test-pkg ./internal/foo`. |
| `ExpectedOutputs` | []string | Strings the builder/QA should see. Compile-error sentinels, log lines, test names. |
| `DecisionLog` | []string | Append-only design-decision rationale. |
| `RelatedItems` | []string | UUIDs / dotted addresses of sibling droplets the builder should know about. |
| `DependsOn` | []string | Logical dependencies — same-shape as `BlockedBy` but pre-runtime intent. |
| `BlockedBy` | []string | The runtime ordering primitive — sibling UUIDs whose completion gates this droplet. |
| `ContextBlocks` | []`ContextBlock` | **Typed key-value-shaped slots.** Each block has `Title`, `Body`, `Type` (note / constraint / decision / reference / warning / runbook), `Importance` (low / normal / high / critical). |
| `ResourceRefs` | []`ResourceRef` | Typed pointer to local files / URLs / docs / tickets / snippets, with `Title` / `Notes` / `Tags`. |
| `KindPayload` | json.RawMessage | **Free-form kind-specific JSON.** This is the "small key-value thing" the dev called out — a planner can stash any structured shape here without a schema. |
| `CompletionContract` | struct | Wraps `StartCriteria` / `CompletionCriteria` / `CompletionChecklist` / `CompletionEvidence` / `CompletionNotes`. |

### `CompletionContract` checklists

`StartCriteria` / `CompletionCriteria` / `CompletionChecklist` are each `[]ChecklistItem` where `ChecklistItem = { ID, Text, Complete bool }`. The lifecycle invariant — `StartCriteriaUnmet` blocks `in_progress` entry, `CompletionCriteriaUnmet` blocks `complete` entry — is the **executable acceptance gate** that SDD wants. This is the closest existing primitive to "spec must be testable."

### `ContextBlocks` and `KindPayload` are the spec's structural workhorses

The dev's "small key-value thing that says what it is to do, some kind of pseudo code maybe, fields needed" maps cleanly onto two existing surfaces:

- **`ContextBlocks`** — typed prose slots. Stable types (`note` / `constraint` / `decision` / `reference` / `warning` / `runbook`) give the planner a vocabulary. Each block carries an importance hint a builder/QA can sort on.
- **`KindPayload`** — free-form JSON. Use for **anything tabular** — pseudocode field-by-field, a tiny per-droplet schema sketch, a property list. Round-trips as `json.RawMessage` so it survives idempotent normalize.

Together these cover the "key-value-friendly" property the dev wants without inventing new fields.

---

## 2. Five Worked Examples

Each example shows the action-item shape end-to-end, then four short slices showing how planner / builder / qa-proof / qa-falsification compose with the spec.

### Example 1 — Tiny Droplet (1-2 code blocks)

**Scenario.** Rename `dispatcher.Service.SpawnRound` to `dispatcher.Service.DispatchRound` to match the verb already used everywhere else in the package. Behavior-preserving. Touches one file, one package.

#### Action-item shape

```toml
[action_item]
Title              = "DROPLET — RENAME DISPATCHER SERVICE.SPAWNROUND TO DISPATCHROUND"
Kind               = "build"
Role               = "builder"
StructuralType     = "droplet"
Irreducible        = true
Owner              = ""
DropNumber         = 7
Persistent         = false
DevGated           = false
Paths              = ["internal/app/dispatcher/service.go"]
Packages           = ["internal/app/dispatcher"]
Files              = [
  "internal/app/dispatcher/service_test.go",
  "internal/app/dispatcher/cli_claude/spawn.go",  # one caller; needed for read-context
]
StartCommit        = "<populated at in_progress>"
EndCommit          = "<populated at terminal>"
Labels             = ["rename", "behavior-preserving"]
Priority           = "low"
```

#### Description body (the primary spec — ~12 lines)

```markdown
## Spec

**Change.** Rename `Service.SpawnRound` → `Service.DispatchRound` (method only;
struct unchanged). The package's external method vocabulary uses `Dispatch*`
everywhere else (`DispatchOnce`, `DispatchPlan`); this rename closes the drift.

**Pseudo-diff.**
- `service.go` — method receiver line + body unchanged; identifier renamed.
- All in-package callers updated (currently exactly two, both in `spawn.go`).

**Out of scope.** No signature change, no return-type change, no logging string
change. No public-API doc updates outside the renamed method's own doc-comment.
```

#### Metadata

```toml
[metadata]
Objective                = "Close the SpawnRound / Dispatch* verb drift in dispatcher.Service."
AcceptanceCriteria       = """
- The identifier `SpawnRound` no longer appears in `internal/app/dispatcher/`.
- All callers compile.
- `mage test-pkg ./internal/app/dispatcher/...` is green.
"""
ValidationPlan           = """
1. mage check
2. mage test-pkg ./internal/app/dispatcher/...
3. grep -R 'SpawnRound' internal/  # must return zero hits in non-test files
"""
CommandSnippets          = [
  "mage check",
  "mage test-pkg ./internal/app/dispatcher/...",
]
ExpectedOutputs          = [
  "ok\tgithub.com/evanmschultz/tillsyn/internal/app/dispatcher",
]
RiskNotes                = "None expected — pure local rename."
```

#### `KindPayload` (optional for tiny droplets — shown for symmetry)

```json
{
  "rename_target": {
    "from": "SpawnRound",
    "to":   "DispatchRound",
    "kind": "method",
    "receiver": "Service",
    "package": "internal/app/dispatcher"
  }
}
```

#### `CompletionContract`

```toml
[completion_contract.start_criteria]
- text = "Worktree clean for `internal/app/dispatcher/service.go`"
  complete = false
- text = "LSP refreshed for current package"
  complete = false

[completion_contract.completion_criteria]
- text = "Identifier renamed in service.go and all in-package callers"
  complete = false
- text = "`mage check` passes"
  complete = false
- text = "`mage test-pkg ./internal/app/dispatcher/...` passes"
  complete = false
```

#### Composition with cascade roles

- **Planner authored.** The planner produced the description body, paths, packages, the three `CommandSnippets`, the `KindPayload` rename block, and the start/completion criteria. Total authoring time: ~2 minutes — most of the spec is convention, not invention.
- **Builder consumed.** Builder reads `KindPayload.rename_target` first (the canonical change shape), confirms `Paths` matches, runs the rename via `LSP` (rename-symbol), then runs the two mage targets in `CommandSnippets` and updates `EndCommit`.
- **QA-proof verified.** Reads description + `AcceptanceCriteria` + `ValidationPlan`, fetches its own `git diff` against `StartCommit`, confirms only the identifier changed (no semantics drift), confirms `ExpectedOutputs` matched, posts verdict.
- **QA-falsification attacked.** Tries to break the assumption: are there reflective lookups by string name? Generated code? Test fixtures referencing `SpawnRound`? `grep -RIn SpawnRound .` (including non-Go) is the natural attack vector the spec didn't bound — falsification surfaces it explicitly.

The spec composes with Section 0 by being cited in **Premises** ("droplet brief: rename SpawnRound → DispatchRound; behavior-preserving") and the four `CommandSnippets` outputs become **Evidence**.

---

### Example 2 — Multi-File Build Droplet

**Scenario.** Thread a new `auto_promote` boolean through three layers — domain → storage → MCP — so the dispatcher can mark a build droplet for automatic move-to-complete on `mage ci` green. Touches 5 files in 3 packages.

#### Action-item shape

```toml
[action_item]
Title              = "DROPLET — THREAD AUTO_PROMOTE BOOL DOMAIN → STORAGE → MCP"
Kind               = "build"
Role               = "builder"
StructuralType     = "droplet"
Irreducible        = false                        # spans 3 packages, but planner still treats as one builder spawn
Owner              = ""
DropNumber         = 7
Persistent         = false
DevGated           = false
Paths              = [
  "internal/domain/action_item.go",
  "internal/adapters/storage/sqlite/action_items.go",
  "internal/adapters/storage/sqlite/migrations/0023_auto_promote.sql",
  "internal/adapters/server/mcpapi/handler.go",
  "internal/adapters/server/mcpapi/extended_tools.go",
]
Packages           = [
  "internal/domain",
  "internal/adapters/storage/sqlite",
  "internal/adapters/server/mcpapi",
]
Files              = [
  "internal/domain/action_item_test.go",
  "internal/adapters/storage/sqlite/action_items_test.go",
  "internal/adapters/server/mcpapi/extended_tools_test.go",
  "internal/app/service.go",                      # READ ONLY — confirms the field flows through unchanged in app layer
]
Labels             = ["new-field", "cross-layer", "additive"]
Priority           = "medium"
```

#### Description body (~50 lines)

```markdown
## Spec

**Change.** Add `auto_promote bool` end-to-end through the action-item write path.
The dispatcher will read this field after `mage ci` green and auto-transition
the droplet to `complete` without orchestrator intervention. Default `false`;
only the dispatcher and explicit MCP callers set `true`.

### Layer-by-layer

#### Domain (`internal/domain/action_item.go`)

- Add `AutoPromote bool` to `ActionItem` struct (placement: after `DevGated` to
  keep the bool flag run together).
- Add `AutoPromote bool` to `ActionItemInput`.
- `NewActionItem` copies it into the constructed value. No validation — bool
  zero-value (false) is the meaningful default. No normalization needed.
- Doc-comment style: match the existing `Persistent` / `DevGated` doc-comments
  (3-line semantics paragraph + zero-value clause + cascade-methodology pointer).

#### Storage (`internal/adapters/storage/sqlite/action_items.go` + new migration)

- Migration `0023_auto_promote.sql` — `ALTER TABLE action_items ADD COLUMN
  auto_promote INTEGER NOT NULL DEFAULT 0;` (SQLite convention: bool as INTEGER
  0/1; matches existing `persistent` / `dev_gated` columns).
- Reader: extend the SELECT and the row-scan to populate `AutoPromote` (boolean
  from INTEGER).
- Writer: extend the INSERT and the UPDATE statement to bind `AutoPromote`.
- No migration tooling — pre-MVP rule: dev deletes ~/.tillsyn/tillsyn.db on
  schema change.

#### MCP (`internal/adapters/server/mcpapi/handler.go` + `extended_tools.go`)

- Extend the `till.action_item(operation=create)` and `(operation=update)` JSON
  schemas to accept optional `auto_promote: bool`.
- Wire field-level mapping create + update.
- The pointer-sentinel pattern used for other fields (e.g. `Paths`) is NOT
  needed — bool defaults to false and `false` is a meaningful explicit value;
  use a `*bool` only if the dev confirms partial-update semantics matter.
  Default: pass-through assignment (matches `Persistent` precedent).

### Pseudo-shapes

```go
// internal/domain/action_item.go
type ActionItem struct {
    // ... existing fields ...
    DevGated    bool
    AutoPromote bool        // NEW
    Paths       []string
    // ...
}
```

```sql
-- internal/adapters/storage/sqlite/migrations/0023_auto_promote.sql
ALTER TABLE action_items ADD COLUMN auto_promote INTEGER NOT NULL DEFAULT 0;
```

### Out of scope

- The dispatcher's CONSUMPTION of `AutoPromote` (i.e. the post-`mage-ci`
  auto-transition logic). That is a separate downstream droplet.
- TUI surface for setting the field — dispatcher writes via MCP for now.
- CLI flag on `till action_item create` — separate droplet.
```

#### Metadata

```toml
[metadata]
Objective = "Add the AutoPromote field along the create/update path so a downstream droplet can wire dispatcher auto-transition on top."
ImplementationNotesAgent = """
Order: domain → storage → MCP. Tests after each layer compile clean before moving to the next layer. The bool defaults to false and downstream serialization treats false as the meaningful zero-value — match the Persistent precedent throughout.
"""
AcceptanceCriteria = """
- `domain.ActionItem.AutoPromote` round-trips through New / normalize.
- Storage round-trip test asserts the bool persists across SQLite save/load.
- MCP create + update accept and round-trip the field.
- All five test files extend with at least one assertion covering the new field.
- `mage ci` green.
"""
ValidationPlan = """
1. mage test-pkg ./internal/domain/...
2. mage test-pkg ./internal/adapters/storage/sqlite/...
3. mage test-pkg ./internal/adapters/server/mcpapi/...
4. mage ci
"""
CommandSnippets = [
  "mage test-pkg ./internal/domain/...",
  "mage test-pkg ./internal/adapters/storage/sqlite/...",
  "mage test-pkg ./internal/adapters/server/mcpapi/...",
  "mage ci",
]
ExpectedOutputs = [
  "ok\tgithub.com/evanmschultz/tillsyn/internal/domain",
  "ok\tgithub.com/evanmschultz/tillsyn/internal/adapters/storage/sqlite",
  "ok\tgithub.com/evanmschultz/tillsyn/internal/adapters/server/mcpapi",
]
RiskNotes = """
- Schema change requires dev to delete ~/.tillsyn/tillsyn.db before `till` reuse — flag this in the closing comment so the dev hits clean state on next run.
- The Persistent + DevGated fields predate the AutoPromote field; if either acquires migration tooling later, the AutoPromote migration will need parity.
"""
DecisionLog = [
  "Bool not *bool: chose pass-through assignment to match Persistent precedent (description #2).",
  "Migration named 0023_auto_promote.sql: next available number per existing migrations dir.",
]
RelatedItems = [
  # The dispatcher consumer droplet, queued blocked_by this one
  "tillsyn-7.4.2",
]
```

#### `ContextBlocks` (used to scope two key constraints)

```toml
[[metadata.context_blocks]]
Title      = "Bool default is meaningful"
Body       = "AutoPromote=false is the explicit safe default for every existing row in the migration. Do NOT add a NULL+coalesce path; the SQLite NOT NULL DEFAULT 0 column matches the Persistent precedent."
Type       = "constraint"
Importance = "high"

[[metadata.context_blocks]]
Title      = "No migration tooling"
Body       = "Pre-MVP rule: dev manually deletes ~/.tillsyn/tillsyn.db on schema change. Do NOT add a `till migrate` command, do NOT write a one-shot SQL fixup, do NOT touch internal/adapters/storage/sqlite/migrations/runner.go."
Type       = "constraint"
Importance = "critical"
```

#### `KindPayload` (the "small key-value thing" — captures the cross-layer field shape compactly)

```json
{
  "field_thread": {
    "name":     "AutoPromote",
    "type":     "bool",
    "default":  false,
    "layers": [
      { "package": "internal/domain", "file": "action_item.go", "kind": "struct_field", "after": "DevGated" },
      { "package": "internal/domain", "file": "action_item.go", "kind": "struct_field", "after": "DevGated", "struct": "ActionItemInput" },
      { "package": "internal/adapters/storage/sqlite", "file": "action_items.go", "kind": "scan_and_bind" },
      { "package": "internal/adapters/storage/sqlite", "file": "migrations/0023_auto_promote.sql", "kind": "schema_alter" },
      { "package": "internal/adapters/server/mcpapi", "file": "handler.go", "kind": "json_schema_field" },
      { "package": "internal/adapters/server/mcpapi", "file": "extended_tools.go", "kind": "create_update_wire" }
    ]
  }
}
```

#### `CompletionContract`

```toml
[completion_contract.start_criteria]
- text = "Worktree clean across all five Paths"
  complete = false

[completion_contract.completion_criteria]
- text = "Domain layer compiles and tests pass"
  complete = false
- text = "Storage layer compiles and tests pass"
  complete = false
- text = "MCP layer compiles and tests pass"
  complete = false
- text = "mage ci green"
  complete = false
- text = "Closing comment notes dev must delete ~/.tillsyn/tillsyn.db"
  complete = false
```

#### Composition with cascade roles

- **Planner authored.** Planner produced the layer-by-layer description (the load-bearing part — sequencing matters), the `KindPayload.field_thread` table (compact cross-layer reference), and the two `ContextBlocks` (the migration-tooling constraint is the most likely place a builder would over-reach). Authoring time: ~10 minutes.
- **Builder consumed.** Reads `KindPayload.field_thread` first as the working checklist, ticks off layers, runs each layer's mage target before moving on. The `ContextBlocks.critical` "No migration tooling" block is the single most important spec constraint; the builder cites it in the Section 0 Proposal pass.
- **QA-proof verified.** Pulls its own diff (REV-4), confirms each row of `KindPayload.field_thread` is reflected in the diff exactly, confirms the migration file matches the stated naming convention, confirms the bool default is `false` end-to-end, confirms test coverage at all three layers.
- **QA-falsification attacked.** Targets the most likely failure modes: (a) is there silent bool-pointer divergence? (the spec said pass-through, did the builder do `*bool` somewhere?), (b) does the SQLite NOT NULL DEFAULT 0 column actually populate for pre-existing rows, or did the builder forget the migration runner needs to apply this on every existing row? (c) Does the JSON schema allow `null` for the bool, which would create a third state?

The spec composes with Section 0 by anchoring **Premises** (cross-layer thread, default-false, no migration tooling) and **Trace** (one row of `KindPayload.field_thread` per case in the trace).

---

### Example 3 — Refactor Droplet

**Scenario.** Split `internal/tui/model.go` (currently ~22k LOC) into substructures grouped by tab. Behavior-preserving. The drop's purpose is purely structural: every public function must keep the same signature and semantics; every keypress must produce the same view.

This is the example where **"what's there now"** matters most. The spec MUST list the invariants that survive.

#### Action-item shape

```toml
[action_item]
Title              = "DROPLET — SPLIT INTERNAL/TUI MODEL.GO BY TAB GROUPING"
Kind               = "build"
Role               = "builder"
StructuralType     = "droplet"
Irreducible        = false
DropNumber         = 7
Paths              = [
  "internal/tui/model.go",
  "internal/tui/model_actions.go",       # NEW — extracted Update() action handling
  "internal/tui/model_view.go",           # NEW — extracted View() rendering
  "internal/tui/model_tabs.go",           # NEW — extracted per-tab dispatch
  "internal/tui/model_keys.go",           # NEW — extracted keybind dispatch
]
Packages           = ["internal/tui"]
Files              = [
  "internal/tui/model_test.go",
  "internal/tui/keys.go",
  "internal/tui/columns.go",
]
Labels             = ["refactor", "behavior-preserving", "no-public-api-change"]
Priority           = "medium"
```

#### Description body (~45 lines, with explicit preserve block)

```markdown
## Spec — Refactor Only

**Change.** Split `model.go` into 5 files grouped by responsibility. No public
API changes. No behavior changes. Every keystroke produces the exact same
output as before the refactor.

### Decomposition target

| New file | Carries |
|---|---|
| `model.go` | Struct definition, `Init()`, top-level `Update()` shell. |
| `model_actions.go` | `Update()` action-handling switch arms (~50 cases today). |
| `model_view.go` | `View()` and per-tab render helpers. |
| `model_tabs.go` | Tab-switch dispatch + per-tab state. |
| `model_keys.go` | Keybind handling (separate from action-handling so kbd → action mapping is one place). |

### MUST-PRESERVE (non-negotiable invariants)

1. **Public surface unchanged.** The exported identifiers from
   `internal/tui` MUST be byte-identical pre- and post-refactor — same names,
   same signatures, same doc-comment text.
2. **Tab order unchanged.** The TUI's tab order is load-bearing for the dev's
   muscle memory; the post-refactor `Init()` must produce tabs in the same
   order.
3. **Golden tests pass without `mage test-golden-update`.** Behavioral parity
   IS the test. If a golden diff appears, the refactor changed behavior — back
   out and re-investigate.
4. **`internal/tui` package import surface unchanged.** No new third-party
   imports; no new internal-package imports; no removed imports unless they
   were already dead.
5. **`model_test.go` import line unchanged.** Tests should not need to know
   the file split happened.

### MAY-CHANGE (degrees of freedom)

- Internal helper visibility — package-private helpers may be moved between
  the new files freely.
- Comment whitespace and section banners.
- Import grouping per file (stdlib / third-party / local — match Go-style
  guide).
- Internal-only types, if their package-private use sites all move with them.

### Out of scope

- Renaming any helper.
- Behavioral fixes for any latent bug — file a separate refinement.
- Removing dead code — file a separate refinement.
```

#### Metadata

```toml
[metadata]
Objective = "Reduce internal/tui/model.go from ~22k LOC to a navigable five-file split. Behavior-preserving."
AcceptanceCriteria = """
- All five new files exist and compile.
- model.go itself drops below 5k LOC.
- mage ci green.
- mage test-golden green WITHOUT update.
- `git diff --stat <StartCommit> -- internal/tui/model.go` shows net deletions matching the split target (~17k LOC moved out of model.go).
- No exported identifier added or removed (verify via `go doc -all ./internal/tui` diff).
"""
ValidationPlan = """
1. mage check
2. mage test-pkg ./internal/tui/...
3. mage test-golden                            # MUST be green without update
4. go doc -all ./internal/tui > /tmp/post.txt  # diff vs baseline captured pre-refactor
"""
CommandSnippets = [
  "mage check",
  "mage test-pkg ./internal/tui/...",
  "mage test-golden",
]
RiskNotes = """
- Highest risk: the internal split inadvertently changes a closure capture or shared state placement, producing a behavior diff visible only in golden tests.
- Second risk: a pointer receiver moves to a value receiver (or vice versa) in the split — Go-vet usually catches this.
- Third risk: the split breaks `gofmt` import grouping; run `mage format` post-split.
"""
```

#### `ContextBlocks` (preserve / may-change blocks promoted to typed slots)

```toml
[[metadata.context_blocks]]
Title      = "Public surface MUST be byte-identical"
Body       = "Every exported identifier — name, signature, doc-comment text — must round-trip unchanged. Use `go doc -all ./internal/tui` pre/post diff as the single audit."
Type       = "constraint"
Importance = "critical"

[[metadata.context_blocks]]
Title      = "Golden tests are the behavior oracle"
Body       = "If `mage test-golden` produces a diff, the refactor changed behavior — STOP and investigate. Do NOT run `mage test-golden-update`."
Type       = "warning"
Importance = "critical"

[[metadata.context_blocks]]
Title      = "Internal helper visibility is malleable"
Body       = "Package-private helpers may be moved between the five new files freely. Type-name visibility may NOT be promoted from package-private to exported (that would change the public surface)."
Type       = "decision"
Importance = "normal"
```

#### `KindPayload` (the file-split shape, compactly)

```json
{
  "split_target": {
    "source_file": "internal/tui/model.go",
    "source_loc_before": 22000,
    "destination_files": [
      { "name": "model.go",          "carries": "struct + Init + top-level Update shell", "loc_target_after": 4500 },
      { "name": "model_actions.go",  "carries": "Update action switch",                    "loc_target_after": 7000 },
      { "name": "model_view.go",     "carries": "View + per-tab render helpers",           "loc_target_after": 5000 },
      { "name": "model_tabs.go",     "carries": "tab dispatch + per-tab state",            "loc_target_after": 3500 },
      { "name": "model_keys.go",     "carries": "keybind dispatch",                        "loc_target_after": 2000 }
    ]
  }
}
```

#### `CompletionContract`

```toml
[completion_contract.start_criteria]
- text = "go doc -all ./internal/tui captured to baseline"
  complete = false
- text = "Worktree clean for internal/tui/"
  complete = false

[completion_contract.completion_criteria]
- text = "Five new files exist and compile"
  complete = false
- text = "go doc -all post-refactor matches baseline byte-for-byte"
  complete = false
- text = "mage test-golden green without update"
  complete = false
- text = "mage ci green"
  complete = false
```

#### Composition with cascade roles

- **Planner authored.** Planner produced the decomposition table, the MUST-PRESERVE / MAY-CHANGE split, and the two critical-importance ContextBlocks. The planner's domain knowledge is mostly in the Preserve block — that's the spec's load-bearing content.
- **Builder consumed.** Builder cites the preserve block in Section 0 Proposal as a hard constraint. Captures `go doc -all ./internal/tui` to a baseline file BEFORE splitting. Splits incrementally, running `mage check` after each file extraction.
- **QA-proof verified.** Confirms `go doc -all` diff is empty. Confirms golden tests green without update. Confirms net LOC delta in `model.go` matches `KindPayload.split_target.destination_files[0].loc_target_after`.
- **QA-falsification attacked.** Tries to construct a behavior diff: (a) is there a `var` block whose initialization order matters and got reordered? (b) Did a method's value-receiver-vs-pointer-receiver flip? (c) Did a `type alias` move and lose its embedding? Falsification looks specifically at cases the preserve block names AND the sneakier cases it doesn't.

Refactor-droplet specs lean heavier on **constraint** ContextBlocks and lighter on `AcceptanceCriteria` (because the criteria are mostly "nothing changed"). The Section 0 spec citation lands in **Premises** as the preserve invariants and in **Unknowns** as "the refactor left semantics intact for every code path the golden tests don't reach."

---

### Example 4 — Research Droplet

**Scenario.** Investigate whether gopls behaves correctly when two worktrees of the same repo are open simultaneously and the dev runs `gopls` in each. Read-only research.

#### Action-item shape

```toml
[action_item]
Title              = "RESEARCH — GOPLS DAEMON BEHAVIOR ACROSS PARALLEL WORKTREES"
Kind               = "research"
Role               = "research"
StructuralType     = "droplet"
Irreducible        = true
DropNumber         = 7
Paths              = []                              # research is read-only
Packages           = []
Files              = [
  "magefile.go",                                     # how mage triggers gopls
  "internal/tui/model.go",                           # where gopls's symbol cache matters most
  "CONTRIBUTING.md",                                 # current dev docs on gopls + worktrees
]
Labels             = ["research", "gopls", "worktree", "dx"]
Priority           = "medium"
```

#### Description body (~25 lines)

```markdown
## Research Question

When the dev has two worktrees of the same repo open simultaneously
(`/main`, `/drop_4c_6`), does gopls correctly serve symbol / reference / rename
queries against each worktree's own files, or does the daemon race between
them — surfacing stale results from whichever worktree it indexed first?

### Sub-questions

1. Does `gopls` use one process across worktrees, or one per workspace root?
2. If one process: how does the LRU eviction policy behave when both workspaces
   exceed the daemon's per-workspace memory cap?
3. If one per root: how does the `gopls.toml`-configured path cache invalidate
   when the dev `cd`s between worktrees in the same shell?
4. What does the `gopls-sync` skill currently do, and does it cover the
   parallel-worktree case or only the post-refactor stale-cache case?

### Evidence order

1. **Hylla** for `internal/tui/`, `magefile.go`, and `CONTRIBUTING.md` — local
   evidence first.
2. **`go doc golang.org/x/tools/gopls`** for stdlib-side daemon contract.
3. **Context7** `golang/tools` (gopls source) for daemon LRU + workspace-root
   handling.
4. **WebSearch** ONLY for daemon-CLI behavior gaps Hylla + Context7 can't
   close.

### Deliverable shape

Single MD at `workflow/drop_4c_6/RESEARCH/GOPLS_PARALLEL_WORKTREES.md`.
Sections: (1) Question, (2) Findings (with file:line + Hylla node + Context7
ref cites), (3) Options / Trade-offs, (4) Unknowns. No decisions; orchestrator
decides downstream.
```

#### Metadata

```toml
[metadata]
Objective = "Compile findings on gopls parallel-worktree behavior so the orchestrator can decide whether the gopls-sync skill needs a parallel-worktree branch."
AcceptanceCriteria = """
- Each of the 4 sub-questions answered or marked Unknown with route.
- Every finding cites at least one source (file:line, Hylla node, or Context7/web URL).
- No decision recommended.
- ## Hylla Feedback section present.
"""
ValidationPlan = """
- The MD lives at workflow/drop_4c_6/RESEARCH/GOPLS_PARALLEL_WORKTREES.md.
- Closing comment ties Conclusion bullets back to evidence rows.
"""
CommandSnippets = []
ExpectedOutputs = [
  "workflow/drop_4c_6/RESEARCH/GOPLS_PARALLEL_WORKTREES.md exists",
]
RiskNotes = "Risk: research drifts into recommending a fix. Discipline: stop at options/trade-offs."
```

#### `KindPayload` (research has lighter payload — the question itself is the payload)

```json
{
  "research_question": "Does gopls behave correctly across parallel worktrees of the same repo?",
  "sub_questions": [
    "one process across worktrees vs one per root?",
    "LRU eviction across workspaces?",
    "path cache invalidation on cd between worktrees?",
    "does gopls-sync skill cover this case?"
  ],
  "deliverable_path": "workflow/drop_4c_6/RESEARCH/GOPLS_PARALLEL_WORKTREES.md",
  "evidence_order": ["hylla", "go-doc", "context7", "web"]
}
```

#### `CompletionContract`

```toml
[completion_contract.completion_criteria]
- text = "Deliverable MD exists at named path"
  complete = false
- text = "Each sub-question answered or marked Unknown"
  complete = false
- text = "All findings cite a source"
  complete = false
- text = "No decisions in the deliverable"
  complete = false
- text = "## Hylla Feedback section present"
  complete = false
```

#### Composition with cascade roles

- **Planner authored.** Planner crafted the research question + sub-questions + deliverable shape. Research specs are short (the question + the deliverable shape do most of the work) but they MUST be precise — a vague question produces a vague finding.
- **Researcher consumed.** Reads `KindPayload.research_question` and `evidence_order`. Walks evidence sources in order. Writes the deliverable MD. Reports `## Hylla Feedback`.
- **(No build-QA twins for research.)** Research has no `build-qa-proof` / `build-qa-falsification` children. The orchestrator reviews the closing comment via thread; if the question wasn't answered, the orchestrator re-spawns or routes.

Research-droplet specs are the **smallest** kind of spec because the answer's structure is the spec. Section 0 composition: Premises = the question; Conclusion = the findings; Unknowns = explicit and routed in the closing comment.

---

### Example 5 — Plan-Level Spec for a Drop with 5-10 Children

**Scenario.** The parent `plan` action item for drop_4c_6 itself. It carries a brief spec. Children inherit by reference rather than re-authoring.

#### Action-item shape

```toml
[action_item]
Title              = "DROP_4C_6 — AGENTS.TOML RUNTIME CONFIG LAYER"
Kind               = "plan"
Role               = "planner"
StructuralType     = "drop"                          # level-1 drop under the project
Irreducible        = false
DropNumber         = 4                                # treated as drop_4c_6 in dotted form
Paths              = []                                # plans don't lock paths; their build children do
Packages           = []
Files              = [
  "workflow/drop_4c_6/SKETCH.md",
  "workflow/drop_4c_6/RESEARCH/SPEC_DRIVEN_REVIEW.md",
  "workflow/drop_4c_6/RESEARCH/TILLSYN_SPEC_SHAPE_PROTOTYPE.md",
]
Labels             = ["drop", "plan", "agents-toml"]
Priority           = "high"
```

#### Description body (~30 lines — brief because the depth lives in children)

```markdown
## Drop-Level Spec

**Scope.** Move user-tunable runtime config (model, endpoint, env handling,
retries, budgets, turn caps) out of templates into `agents.toml` +
`agents.local.toml`. Six builder droplets per the SKETCH §8 table.

### Children (level 2)

The decomposition table from `SKETCH.md` §8 is the canonical child enumeration
for this plan. Each child droplet carries its own per-droplet spec following
the conventions in `RESEARCH/TILLSYN_SPEC_SHAPE_PROTOTYPE.md`. Children:

| ID | Title (UPPERCASE) | Spec depth |
|---|---|---|
| D1 | DROPLET — AGENTS.TOML SCHEMA TYPES + POSITION-TRACKING LOADER | medium |
| D2 | DROPLET — OVERRIDE MERGE + .GITIGNORE ENSURE-CLAUSE | small |
| D3 | DROPLET — TEMPLATE MIGRATION + AGENTS.EXAMPLE.TOML | small |
| D4 | DROPLET — SPAWN PIPELINE INTEGRATION (env + argv) | medium |
| D5 | DROPLET — ERROR SURFACE: TOML POSITION THREADING | small |
| D6 | DROPLET — DOCS: AGENTS_CONFIG.MD + POINTERS | small |

### Cross-droplet invariants (inherited by every child)

1. **Tillsyn never holds secret values.** Only env-var NAMES.
2. **Tillsyn never validates** model name reachability, endpoint validity, key
   correctness.
3. **Schema validation only**: required-field presence, type-correctness,
   env-var-name regex `^[A-Za-z][A-Za-z0-9_]*$`.

### Sequencing

D1 → D2 → D3, D4 (parallel after D2) → D5 → D6. Encoded as `BlockedBy` on each
child action-item creation; planner sets at fan-out time.
```

#### Metadata (the plan-level metadata is thin — children carry the depth)

```toml
[metadata]
Objective = "Land a runtime-config layer that lets adopters point cascade agents at any provider their CLI accepts, without secrets in TOML and without provider validation by Tillsyn."
DefinitionOfDone = """
- All six children complete with their own QA twins green.
- agents.example.toml exists and the Anthropic-direct flow works end-to-end.
- mage ci green on the integration commit.
"""
ValidationPlan = """
The plan-level validation is the union of every child's ValidationPlan. The plan-qa-proof + plan-qa-falsification twins verify decomposition correctness; build-qa twins on each child verify the child's own work.
"""
DecisionLog = [
  "agents.toml is project-shareable; agents.local.toml is gitignored. (SKETCH §1, §4)",
  "env_set carries non-secret values; env_from_shell carries name-only. (SKETCH §3.1)",
  "Field-level deep merge for overrides. (SKETCH §4)",
]
RelatedItems = [
  "tillsyn-4c.6.RESEARCH",                          # this research itself
  "tillsyn-4c (parent drop)",
]
```

#### `ContextBlocks` — the cross-droplet invariants are promoted to typed slots so each child can reference by parent visibility

```toml
[[metadata.context_blocks]]
Title      = "Tillsyn never holds secret values"
Body       = "agents.toml carries env-var NAMES only (env_from_shell). Literal values in env_set are explicitly non-secret runtime config (base_url, region, use_bedrock=1). Reject any child droplet that puts a secret VALUE in agents.toml."
Type       = "constraint"
Importance = "critical"

[[metadata.context_blocks]]
Title      = "Tillsyn never validates provider-side reality"
Body       = "Schema validation is required-field-presence + type-correctness + env-var-name regex. Model reachability, endpoint validity, key correctness are NOT Tillsyn's concern — runtime errors come back from the spawned CLI; surface them with a TOML-line pointer."
Type       = "constraint"
Importance = "critical"
```

#### `KindPayload` (the decomposition shape — children inherit by ID reference)

```json
{
  "decomposition": {
    "child_count": 6,
    "children": [
      { "id": "D1", "blocked_by": [] },
      { "id": "D2", "blocked_by": ["D1"] },
      { "id": "D3", "blocked_by": ["D2"] },
      { "id": "D4", "blocked_by": ["D2"] },
      { "id": "D5", "blocked_by": ["D3", "D4"] },
      { "id": "D6", "blocked_by": ["D5"] }
    ],
    "shared_constraints": [
      "no_secret_values_in_toml",
      "no_provider_side_validation",
      "env_var_name_regex"
    ]
  }
}
```

#### `CompletionContract`

```toml
[completion_contract.start_criteria]
- text = "RESEARCH/SPEC_DRIVEN_REVIEW.md complete"
  complete = false
- text = "RESEARCH/TILLSYN_SPEC_SHAPE_PROTOTYPE.md complete"
  complete = false
- text = "Dev approves SKETCH.md §7 open questions"
  complete = false

[completion_contract.completion_criteria]
- text = "All six children in StateComplete"
  complete = false
- text = "plan-qa-proof + plan-qa-falsification twins green"
  complete = false
- text = "Drop closeout commit + push + ingest sequence green"
  complete = false
```

#### Composition with cascade roles

- **Planner authored.** Plan-level description is brief because the children carry depth. The planner's load-bearing work is (a) the decomposition table (which children, in which order, with which `BlockedBy`) and (b) the cross-droplet invariants in `ContextBlocks` — those are the constraints children inherit by reference rather than copying.
- **Children consume.** Each child droplet's planner cites the parent's `ContextBlocks` ("see parent constraint: no_secret_values_in_toml") rather than re-authoring. This is the inheritance pattern — parents own cross-cutting constraints; children own the per-droplet specifics.
- **plan-qa-proof verified.** Verifies the decomposition is complete (every SKETCH §8 row has a child), `BlockedBy` wiring matches the sequencing prose, the cross-droplet constraints are present in `ContextBlocks`.
- **plan-qa-falsification attacked.** Tries to find a missing child (does the SKETCH carry an implicit work item the planner forgot to bind?), an over-constrained sequencing chain (does D5 really need both D3 AND D4?), an under-specified constraint (does "schema validation only" cover the duplicate-keys case the SKETCH §9 test plan flags?).

Plan-level specs are **the inheritance root**. Section 0 composition: Premises cite the cross-droplet invariants by name; child droplets inherit them by parent-ID reference, keeping per-droplet specs small.

---

## 3. Tillsyn-Flavored Spec Design Principles

Distilled from the five examples:

1. **Co-located with the action item.** The spec is `Description` + `Metadata` on the same row. There is no `workflow/specs/<id>.md` file. There is no project-wide `spec.md`. If the action item is deleted, the spec is deleted with it — it was the prompt structure for that work, nothing more.

2. **Primitives only — no schema changes.** Every field used here exists in `internal/domain/action_item.go` + `internal/domain/workitem.go` at HEAD. Adding new fields would be a Tillsyn drop, not an SDD adoption. Until the dev decides otherwise, the spec format is a CONVENTION laid over existing fields.

3. **Scale with the droplet.** A tiny rename droplet's spec is ~12 lines of description + 5 metadata fields populated. A multi-file build droplet is ~50 lines + most metadata fields populated. A plan-level drop spec is brief because children carry the depth. The spec is never bigger than the work justifies — that's the YAGNI guardrail SDD's "heavy upfront waterfall" anti-pattern violates.

4. **Key-value-friendly via `KindPayload` + `ContextBlocks`.** The dev's "small key-value thing that says what it is to do" maps cleanly onto these two existing fields. `KindPayload` for free-form JSON shapes (rename targets, field-thread tables, decomposition tables); `ContextBlocks` for typed prose constraints / decisions / warnings / runbooks. Together they cover everything SDD calls "structured spec content" without inventing new fields.

5. **Pseudocode-optional, present when clarifying.** The multi-file build example uses Go + SQL pseudo-shapes because the cross-layer thread is easier to read as code than as prose. The refactor example uses a table because the file-split structure is tabular. The tiny-rename example uses a one-line `KindPayload` because that's all the planner needs to communicate. Pseudocode shows up where it earns its place; nowhere else.

6. **Acceptance criteria are mage targets + grep assertions.** Every example's `AcceptanceCriteria` resolves to executable checks: a mage target outputs `ok`, a grep returns zero hits, a doc-comment diff is empty, a golden test is green without update. SDD's "spec must be testable" property is satisfied by Tillsyn's existing `mage` discipline, not by inventing a spec-conformance language.

7. **Inheritance via `ContextBlocks` + parent reference.** Cross-cutting constraints live on the plan-level parent. Children cite by parent-visibility rather than re-authoring, keeping per-droplet specs small. This is the only "spec hierarchy" Tillsyn needs — the existing parent-child nesting plus typed `ContextBlocks` does the work.

8. **Not vanilla SDD.** No `/specify` phase. No `/plan` artifact. No `/tasks` artifact. No round-trip from spec → regenerated code. No spec-as-source-of-truth elevation. The Tillsyn shape borrows SDD's good parts (specs co-located with the work; testable acceptance; structured constraints) and rejects SDD's heavy parts (separate spec files, project-wide specs, code regenerated from specs). The spec is **a way of structuring prompts**, not a contract the dev maintains alongside the code.

---

## 4. What This Document Is Not

- **Not a planner specification.** A planner agent's spec lives in `~/.claude/agents/go-planning-agent.md`. This document describes what shape the planner's OUTPUT (action-item description + metadata) should take when SDD-inspired conventions land.
- **Not a schema proposal.** No new `ActionItem` fields. No new `ActionItemMetadata` fields. No new validation rules. Every example uses fields that exist in HEAD.
- **Not a mandate.** The dev hasn't yet decided whether these conventions land as a planner-prompt addition, a `~/.claude/agents/go-planning-agent.md` rule, or a `WIKI.md` convention page. This document gives the dev concrete enough material to make that decision.
- **Not enforceable today.** Until / unless a future drop adds template `child_rules` validation for description structure, "did the planner follow these conventions?" is honor-system. That mirrors the rest of pre-cascade Tillsyn (`Required Children` rule, role gating, etc. — orchestrator-enforced today, template-enforced post-Drop-3).

---

## 5. Open Questions for the Dev (not decided here)

1. **Where do these conventions land?** Three candidates:
   - `~/.claude/agents/go-planning-agent.md` body (per-spawn prompt content).
   - `WIKI.md` convention section (durable adopter-facing reference).
   - `internal/templates/builtin/default-go.toml` `[agent_bindings.plan].context` extension (per-spawn structural injection).
   - All three are consistent with the "no separate spec file" rule.
2. **Should `ContextBlocks` types extend with a `spec` type?** Existing types are `note / constraint / decision / reference / warning / runbook`. Constraint covers the load-bearing case but a `spec` block-type would let downstream tooling (TUI, dashboards) render spec-grade context distinctly. **NOT proposed in this document** — listed as a future-drop refinement candidate if the convention is adopted at scale.
3. **`KindPayload` JSON schema discoverability.** Every kind that uses `KindPayload` for a stable shape (e.g. `field_thread` for cross-layer build droplets) implicitly defines a schema. The dev may decide later whether per-kind schema validation belongs in the template, in `internal/domain/`, or stays honor-system. **NOT proposed here.**
4. **Plan-level `ContextBlocks` inheritance.** The Example 5 pattern ("children inherit constraints by parent reference") is convention today. A future drop might encode it as a planner-time auto-copy ("on child create, inherit `Importance: critical` ContextBlocks from the nearest plan ancestor"). **NOT proposed here** — listed as a future-drop ergonomic improvement.

---

## Hylla Feedback

N/A — research touched non-Go primitive enumeration only. All evidence sourced from direct `Read` on `internal/domain/action_item.go`, `internal/domain/workitem.go`, `WIKI.md`, and `workflow/drop_4c_6/SKETCH.md` + the prior `RESEARCH/SPEC_DRIVEN_REVIEW.md`. No Hylla queries attempted in this research session because every needed source is a local non-Go-or-near-by-Go file the prompt named explicitly. If a follow-up research pass needs to walk reverse-references on `ActionItemMetadata` consumers, that pass should hit Hylla first and report misses there.
