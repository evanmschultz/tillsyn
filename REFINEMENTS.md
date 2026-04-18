# Tillsyn Refinements

Append-only log of Tillsyn product refinements, refactor candidates, and TUI / CLI / MCP ergonomics issues discovered during day-to-day use. Paired with the perpetual Tillsyn tracking drop `REFINEMENTS.MD` — entries added here also get mirrored as comments on that drop (dual-update rule from `CLAUDE.md`).

Hylla-specific refinements live in `HYLLA_REFINEMENTS.md`.

## Entry Schema

Each entry uses this shape. Newest-first ordering.

```markdown
## <YYYY-MM-DD> — <Drop N> — <One-line title>

### Context
What the session was doing when the friction surfaced. One or two sentences.

### Observation
The concrete issue. Include tool name + exact input and actual output if applicable. Include enough detail that a future drop can reproduce without re-deriving context.

### Proposed fix
Concrete action to take. Scope matters — distinguish "inline safe fix" from "cross-cutting refactor drop." Cite a proposed target drop when known.

### Target drop
Where this refinement should land. E.g. `Drop 1`, `pre-Drop-3 template-overhaul`, `post-Drop-4 TUI polish`, or `parking-lot` if unassigned.

### Tags
Comma-separated. Examples: `tui`, `cli`, `mcp`, `refactor`, `docs`, `coordination`, `auth`, `ergonomics`, `performance`.
```

## Status Lifecycle

- **Pending** — entry logged, not yet triaged. Default state.
- **Scheduled** — triaged into a concrete drop.
- **In Progress** — currently being worked in a drop.
- **Shipped** — fix landed. Entry summarized into the drop's closeout `WIKI_CHANGELOG.md` line; original entry either stays as-is or gets trimmed during MD cleanup subdrop.
- **Rejected** — not doing. Kept for audit-trail continuity with the reason.

Transitions are recorded by appending a dated status note to the entry, not by rewriting history.

---

## 2026-04-14 — Drop 0 — Local git hooks for gofumpt + `mage ci` parity

### Context
Drop 0 closeout surfaced that the 18.3 builder caught gofumpt drift on `internal/adapters/server/common/app_service_adapter_outcome_test.go` (pre-existing on `main`, not introduced by 18.3) only because `mage ci`'s Formatting stage ran `go tool gofumpt -l` and listed the file. No local gate had caught it at commit or push time, so the drift sat on `main` until a later build job tripped over it. CI formatting checks are correctly read-only (`-l` not `-w`) — the gap is upstream of CI, not inside it.

### Observation
Two distinct issues bundled:

1. **No local pre-commit / pre-push hooks.** Dev workflow relies on developer discipline (`mage format .` then `mage ci` before push). Drift can land on `main` when that discipline slips.
2. **`mage format` no-arg ergonomics wart.** `func Format(path string) error` (`magefile.go:200`) requires a positional arg from the mage CLI, so `mage format` fails with "not enough arguments for target \"Format\", expected 1, got 0". The `if path == "" || path == "."` branch in the function body (lines 201-211) handles the whole-tree case but is unreachable via the CLI — the dev has to type `mage format .` (with dot) to trigger it. Dead-code-from-CLI surface.

### Proposed fix
1. Add committed `.githooks/pre-commit` that runs a new `mage format-check` target (public wrapper around the existing private `formatCheck()` at `magefile.go:218-236`). Fails the commit if gofumpt would modify any tracked `.go` file; error message points the dev at `mage format .`.
2. Add committed `.githooks/pre-push` that runs `mage ci` in full. Matches the "Mage Precommit = CI Parity" feedback rule — the dev should see the same verdict locally that GH Actions will return.
3. Add `mage install-hooks` target that runs `git config core.hooksPath .githooks` so the tracked scripts become active for any fresh clone. Idempotent.
4. Fix `mage format` signature: split into `Format()` (no-arg = whole tree via `trackedGoFiles()`) and `FormatPath(path string)` (scoped); or adopt a variadic `Format(paths ...string)` form. Either way, `mage format` with no args should format the whole tree and not error.
5. Hooks must remain bypassable via `--no-verify` per existing discipline (global CLAUDE.md rule: never bypass without explicit dev instruction).
6. QA-proof + QA-falsification required — the hook scripts are the local build gate, can't silently break.

### Target drop
**Drop 1 — first item.** Scheduled directly into `PLAN.md` §19.1 as the first bullet of the Drop 1 work list.

### Status
**Scheduled — Drop 1 item 1** (2026-04-14).

### Tags
`mage`, `git-hooks`, `tooling`, `ci-parity`, `gofumpt`

---

## 2026-04-14 — Drop 0 — TUI esc-back navigation does not step up one level

### Context
Dev was navigating the main-screen tree during Drop 0 Tillsyn dogfooding. Drilled into a drop subtree and hit esc to return to the immediately previous level.

### Observation
From the main screen, once focused down into a subtree, the `{todo | prog | done}` column-state screen does **not** respect navigation history. Pressing esc from the column-state screen returns directly to the top-level project screen instead of stepping up one level to wherever the focus came from.

### Proposed fix
Esc should behave like browser back: pop one level of navigation history on each press, not short-circuit to project root. Implement a nav-history stack on the main screen so esc pops the most recent push, regardless of column-state depth.

### Target drop
Drop 1 or a later dedicated TUI polish drop. Not Drop 0 — out of scope for the current closeout.

### Tags
`tui`, `navigation`, `ergonomics`

---

## 2026-04-14 — Drop 0 — Dotted-address fast-nav across TUI / CLI / MCP

### Context
Drop 0 vocabulary convergence established dotted addresses (`0.1.5.2`, `proj_name-0.1.5.2`) as the human-readable shorthand for drop references, distinct from UUIDs which remain authoritative for mutations. Today, dev ↔ orchestrator cross-reference happens by copy-pasting UUIDs, which is high-friction.

### Observation
No TUI / CLI / MCP surface today understands dotted addresses. Examples of the intended UX:

- **TUI**: dev types `0.1.5.2` or `8.9.3` into a go-to / search field and is focused on that drop.
- **CLI**: `till view tillsyn-8.9.3`, `till comment tillsyn-8.9.3 "looks good"`, `till state tillsyn-8.9.3 done` — all resolve the dotted path to the current UUID and operate on it.
- **MCP**: orchestrator can pass dotted addresses to tool calls for **reads** (`till.action_item(operation=get, address="0.1.5.2")`). Mutations should still require UUID — dotted addresses shift under re-parenting.

Project-name prefix (`tillsyn-`) is unnecessary inside a scope-bound surface (TUI already knows the project; MCP session is project-scoped). Required for cross-project references.

### Proposed fix
1. Add a dotted-address resolver in `internal/domain` (or `internal/app`) that walks the drop tree by position to find the UUID.
2. Wire the resolver into TUI go-to input, CLI positional args, and MCP read operations.
3. Document the mutations-are-UUID-only rule so no agent accidentally relies on a dotted address for a `till.action_item(operation=update)` call.

### Target drop
Post-Drop-3 template overhaul or a dedicated addressing drop. Not Drop 1.

### Tags
`tui`, `cli`, `mcp`, `addressing`, `ergonomics`

---

## 2026-04-14 — Drop 0 — Batch operations on action-item nodes

### Context
Orchestrator + cascade agents frequently perform many small action-item mutations in sequence (create N drops, update M descriptions, move K items to `in_progress`). Every call is a separate MCP round-trip.

### Observation
Post-Drop-4 the cascade dispatcher will be doing hundreds of these per cascade run. One-at-a-time MCP round-trips will become a real latency and rate-limit problem. Pre-cascade, the orchestrator already feels the friction (e.g. creating refinement drops, creating build-task + qa-proof + qa-falsification trios).

### Proposed fix
Batch operations on `till.action_item`:

- `till.action_item(operation=create_batch, items=[...])` — create N items in one call.
- `till.action_item(operation=update_batch, updates=[...])` — apply N updates in one call.
- `till.action_item(operation=move_state_batch, moves=[...])` — bulk lifecycle transitions.
- Configurable limit per call (e.g. 25 items) to bound request size.

Atomicity policy (all-or-nothing vs best-effort with per-item error rows) is a design question — lean toward best-effort with a results array so partial success is observable.

### Target drop
Post-Drop-4 (dispatcher drop) — the cascade makes the cost real. Could be pulled earlier if pre-cascade friction gets noisy.

### Tags
`mcp`, `performance`, `ergonomics`, `refactor`
