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

## 2026-05-14 — pre-dogfood — Remove `project.Language` field; templates are project-tier opt-in only

### Context
Dev caught architecture drift while diagnosing why TILLSYN's `till action_item create --kind build` did NOT auto-spawn QA twins. Tracing the load path revealed:

1. `project.Language` is a closed enum (`"" | "go" | "fe"`) added in Drop 4a L4 (`a334f20`).
2. The dispatcher / service-layer template resolver uses `project.Language` to pick an EMBEDDED-default template (`till-go.toml`, `till-fe.toml`, `till-gen.toml`) when no project-tier `template.toml` exists.
3. Dev's stated design (forgotten / drifted): **Tillsyn projects are not language-bound.** Projects can be multi-language, non-coding, or have any other shape. The `Language` field bakes a coding-project assumption into a domain primitive that should be vocabulary-neutral.

### Observation
Three coupled problems:

1. **`project.Language` is a wrong abstraction** for a general-purpose project tracker. Carries an implicit "this is a coding project in language X" assumption.
2. **Embedded-default template fallback in production** is a wrong design. If a project has no template, Tillsyn should do NOTHING — no child_rules, no enforced kinds, no auto-creation. Templates are user-authored OPT-IN at project tier. Embedded templates exist solely as starter content that `till init` can OFFER to copy into the project on first run.
3. **Multi-group projects (`Metadata.Groups`)** are the partial workaround for the wrong-language assumption (multi-group sidesteps the single-Language constraint by per-group resolution). But this preserves the embedded-fallback antipattern and adds its own complexity.

### Proposed fix
1. **Remove `project.Language` field entirely.** Strip from `domain.Project`, `app.UpdateProjectInput` / `CreateProjectInput`, `till project update --language` CLI flag, `till init --language` JSON payload, TUI form, MCP schema. Migrate persisted Project rows by dropping the column.
2. **Remove `project.Metadata.Groups`** OR re-purpose it as a "starter content selector" (which embedded templates to OFFER on `till init`, not a runtime resolver). Open design question.
3. **Make template resolution project-tier-only at runtime.** `loadProjectTemplate` and `loadProjectTemplatesForGroups` should check `<project>/.tillsyn/template.toml` and `<project>/.tillsyn/templates/*.toml` (multi-file aggregation) ONLY. No HOME tier. No embedded fallback. Empty result is valid — Tillsyn just doesn't auto-create children for that project.
4. **`till init` becomes an opt-in template starter.** Optional flag `--starter-template <name>` (or interactive picker) copies one or more embedded templates into the project's `.tillsyn/templates/`. User can edit afterwards. No starter = no template = no auto-create. Pure tracking-only Tillsyn.
5. **Two-and-only-two validators** on aggregated templates: conflict detection (same kind/child_rule ID with different content) + cycle detection (rules that prevent terminal completion). NO structural-type enforcement. NO kind-enum enforcement. NO closed vocabulary checks.

### Target drop
**Pre-dogfood architectural cleanup drop.** Substantial scope — touches domain primitives, migrations, CLI, TUI, multiple service-layer helpers, all template load tests. Likely a dedicated drop. Should ship BEFORE the cascade-dispatcher auto-trigger lands (Drop 4c.7) so the dispatcher's template resolution path is the right shape from day one.

### Tags
`architecture`, `domain`, `templates`, `breaking-change`, `migration`, `cli`, `tui`, `dogfood-blocker`

---

## 2026-05-14 — pre-dogfood — Multi-template aggregation per project tier

### Context
Tied to the `project.Language` removal above. With the language field gone, the question "how do multiple template files at project tier combine" becomes load-bearing.

### Observation
- Today's multi-group path (`loadProjectTemplatesForGroups`) is a primitive iterate-and-merge that mixes embedded fallback into a multi-group walk. It does not implement the dev's design: project tier supports multiple `template.toml` files (e.g. `<project>/.tillsyn/templates/refactor.toml`, `feature.toml`, `bugfix.toml`) and they aggregate by ID merge with conflict + cycle checks only.
- W2.D6 SKIPPED writing project-tier `template.toml` for multi-group projects (per `init_cmd.go:854`) because naive concat of two embedded templates trips the load-time "table plan already exists" error. The right fix is a semantic ID-merge.

### Proposed fix
1. Add `<project>/.tillsyn/templates/*.toml` discovery (glob the directory).
2. Implement `templates.Aggregate([]Template) (Template, error)` that ID-merges kinds, agent_bindings, child_rules. Last-loaded wins on collision OR error on collision — design choice TBD (probably "error on conflict, force the user to resolve").
3. Cycle check across the aggregated graph — heuristic only, ensures no completion deadlock.
4. Update `loadProjectTemplate` to discover-and-aggregate the `templates/` directory in addition to (or instead of) the single `template.toml` legacy file.

### Target drop
Same drop as `project.Language` removal — they share migration surface.

### Tags
`templates`, `aggregation`, `validation`, `dogfood-blocker`

---

## 2026-05-14 — pre-dogfood — User-defined kinds (dynamic enum from template)

### Context
Dev's design: templates define the vocabulary of kinds for that project. A project could declare `refactor-segment`, `feature-drop`, `bugfix-droplet` as its own kinds. The closed 12-value enum at `internal/domain/kind.go` is a stopgap.

### Observation
Today, `domain.Kind` is a closed Go enum. Template `[kinds.<name>]` sections are validated against this enum at load time (unknown kinds reject). User-defined kinds require:

1. `domain.Kind` becomes a free-form `string` type, validated dynamically against the project's loaded template's `kinds` map.
2. Template-load validators stop rejecting unknown kind names — they only validate against the template's own declared set.
3. Cycle detection in child_rules updates to operate on the dynamic kind set.
4. CLI / TUI / MCP surfaces accept any string kind, surface validation errors when the kind isn't in the bound template.
5. `domain.KindAppliesTo` (the scope mapping) becomes per-template metadata rather than a Go-level closed enum.

### Proposed fix
Land AFTER the `project.Language` removal + multi-template aggregation refinements. User-defined kinds is the keystone of the open-vocabulary design but depends on the template plumbing being right first.

### Target drop
**Post-MVP** — closed 12-kind enum works for the immediate dogfood. Dynamic enum is the next architectural layer.

### Tags
`architecture`, `domain`, `templates`, `kinds`, `post-mvp`

---

## 2026-05-14 — pre-dogfood — Toml-driven agent dispatch split (`-p headless` vs orch-signal)

### Context
Dev's design: not every agent in the cascade should be launched directly by Tillsyn's dispatcher. Some agents need to run as the user's Claude Code orchestrator subagents (oauth-billed, June-15-ToS-compliant interactive sub-spawn). The split is **toml-driven**.

### Observation
Today's dispatcher path (Drop 4a Wave 2 + manual-trigger CLI in Wave 2.2) treats every agent identically: spawn a subprocess via `claude --agent ...` (or equivalent). This doesn't accommodate the split.

The intended model:

- **Tillsyn-launched agents** (`agent.dispatch_mode = "headless"` in agents.toml): codex, openrouter, openai-compat, ollama, claude-api-key, claude-oauth-headless. Tillsyn dispatcher spawns them directly. User pays via their own credential setup (`env_from_shell`, `--api-key`, etc.).
- **Orch-signaled agents** (`agent.dispatch_mode = "orch_subagent"`): typically `oauth-claude-subagent`. Tillsyn does NOT launch. Tillsyn pushes a wake-up event to the orchestrator's MCP client via LiveWait (see separate refinement); the orch's `Agent` tool spawns the subagent using the user's Claude OAuth subscription.

### Proposed fix
1. Add `dispatch_mode` field to `AgentBinding` schema. Closed enum: `headless | orch_subagent`. Default to `headless` for backwards compat.
2. Dispatcher routes by `dispatch_mode`. `headless` continues the existing path. `orch_subagent` publishes a `LiveWaitEventOrchSpawnRequested` event with the action_item ID + agent binding.
3. Orch's MCP client subscribes to that event channel via the Channels API (or equivalent push surface) and routes the wake-up into the orch's conversation context as a system reminder asking the orch to spawn the specified agent.
4. The orch confirms (or declines per its own policy) and uses its `Agent` tool to spawn the subagent.

### Target drop
**Pre-MVP-dogfood phase 2**: after the basic dispatcher path works end-to-end (Drop 4c.7 auto-trigger), the split lands as a follow-on.

### Tags
`dispatcher`, `agents`, `architecture`, `tos`, `oauth`, `dogfood-blocker-phase-2`

---

## 2026-05-14 — pre-dogfood — LiveWait → MCP push to orch (replace `/loop` hack)

### Context
Today the orchestrator polls Tillsyn state via `/loop` cadence to learn about attention items, completed action items, approval requests. The LiveWait broker (`internal/app/live_wait.go`) exists for in-process / cross-process Tillsyn-side wake-ups but does NOT push events through the MCP boundary to the orchestrator's conversation.

Per Claude Code's Channels research-preview feature (https://code.claude.com/docs/en/channels.md), MCP servers can be wrapped as channel plugins that push events into a running session. The Tillsyn MCP server should adopt this so the orch wakes immediately on relevant state changes instead of polling.

### Observation
Coupled with the toml-driven dispatcher split above: when Tillsyn determines an `orch_subagent` agent should fire, it publishes `LiveWaitEventOrchSpawnRequested`. The MCP-as-channel surface receives that event and routes a system reminder into the orch's conversation. The orch's tool surface includes a way to "claim" the wake-up (acknowledge + spawn) so duplicate dispatches don't fire.

### Proposed fix
1. Wrap Tillsyn MCP as a Claude Code channel plugin (Channels API).
2. Channel publishes events for: auth_request approval, attention item raised, action_item state change, handoff created, orch_subagent spawn request.
3. Channel events route into the orch's conversation as system reminders or similar.
4. Orch tool surface includes ack/claim for spawn requests.

### Target drop
**Pre-MVP-dogfood phase 2**. Replaces `/loop` polling entirely for orch-side coordination. Depends on the Channels API stability.

### Tags
`mcp`, `live-wait`, `orch-coordination`, `channels`, `wake-up`, `dogfood-blocker-phase-2`

---

## 2026-05-14 — pre-dogfood — Project-local `till mcp` for natural path / auth scoping

### Context
Joint dogfood smoke session surfaced two related issues in `till init`'s `.mcp.json` generation:

1. **Args bug (fixed inline 2026-05-14).** `registerMCPJSON` wrote `{Command: tillBin}` with no Args, so the entry invoked bare `till` which defaults to the TUI, not the MCP stdio server. Manifested as `claude mcp list` showing `tillsyn: ... ✗ Failed to connect` while a separately-registered `tillsyn-dev: main/till mcp` worked. Fixed in this drop; W2.D6 test gap closed.
2. **Project-local-binary design intent (this refinement).** The dev's stated intent for project-tier `till init` is that the `.mcp.json` entry should invoke a PROJECT-LOCAL `till mcp` binary (e.g. `./till` or some `$PROJECT/till` path), not the global `~/.local/bin/till`. Goal: natural path / auth scope limiting so one orch in project A is less likely to reach into project B's state. Orchs can still request global `till` auth explicitly, but the default invocation stays scoped.

### Observation
- Current `registerMCPJSON` (`cmd/till/init_cmd.go`) resolves the binary via `exec.LookPath("till")` then falls back to `~/.local/bin/till`. Neither path is project-local.
- DB resolution today (user-tier `~/.tillsyn/tillsyn.db`) is independent of which binary invokes `till mcp` — so even a project-local binary wouldn't currently get a project-scoped DB. The scoping has to be either (a) build-time embedded into the project-local binary, or (b) flag-driven (`--db <project>/.tillsyn/tillsyn.db`), or (c) cwd-aware resolution that picks a project-tier DB when `.tillsyn/` is found.
- The `tillsyn-dev` MCP registration today already uses `main/till mcp` — but only because the dev registered it that way by hand. Default `till init` output never produces that shape.

### Proposed fix
Plan + ship in a follow-up drop. Three sub-questions to resolve before building:

1. **Where does the project-local `till` binary live?** Options: `./till` (project root), `.tillsyn/bin/till` (hidden project-tier path), `$REPO_ROOT/till` (whatever the project's build pipeline puts there). Likely the answer depends on whether tillsyn is a Go project (mage builds `./till`) or a non-Go project (where does the binary come from?).
2. **DB resolution policy for project-local `till mcp`.** Decide between (a)/(b)/(c) above. Probably (c) cwd-aware: project-local DB when `.tillsyn/tillsyn.db` exists in cwd or any ancestor, else user-tier fallback. This is symmetric with the FLAT-vs-subdir detection pattern.
3. **Auth scoping semantics.** What does "natural path limit" mean concretely? Auth_request paths today already require an explicit `project/<id>` scope — so the limit is more about which projects the orch can SEE / mutate. If the MCP only opens the project-local DB, the orch literally can't list other projects.

### Target drop
**Parking-lot until dogfood backlog priority surfaces.** Not blocking immediate dogfood since global `till mcp` works and the orch-self-restriction (via scope-bounded auth requests) provides logical limiting today. Surface again during the next dogfood-readiness pass.

### Tags
`till-init`, `mcp`, `auth`, `path-scoping`, `dogfood`, `design`

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
Post-Drop-4 the cascade dispatcher will be doing hundreds of these per cascade run. One-at-a-time MCP round-trips will become a real latency and rate-limit problem. Pre-cascade, the orchestrator already feels the friction (e.g. creating refinement drops, creating build-actionItem + qa-proof + qa-falsification trios).

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
