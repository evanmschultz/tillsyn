# Tillsyn — Project Wiki

Living reference for the Tillsyn project. Captures the **current** best practices, architecture shape, and project state given where the cascade build is right now. Updated whenever a slice lands that changes best practice. Paired with `WIKI_CHANGELOG.md` — entries that ship in a slice mirror as a one-liner into the changelog.

Hylla-specific usage guidance lives in `HYLLA_WIKI.md`. This wiki focuses on Tillsyn itself — the product, the coordination runtime, the cascade, and the dogfood workflow.

## Update Discipline

- Update during the per-slice `SLICE <N> END — LEDGER UPDATE` task, alongside `LEDGER.md` and `WIKI_CHANGELOG.md`.
- Keep sections short and inspectable. If a section grows past ~30 lines, either split it or cut older guidance it's outdated.
- History does NOT live here. History lives in `WIKI_CHANGELOG.md` (one-liners per slice) and in the slice's ledger entry. This wiki is a snapshot of current best practice.
- When a refinement or finding lands that contradicts an entry here, **update the entry in place** — don't append a "2026-04-XX update:" note. The full audit trail is in `REFINEMENTS.md` + `HYLLA_REFINEMENTS.md` + git history.

## Current State (Slice 0)

Slice 0 is the **project reset + docs cleanup** slice. It does not change Go code; it resets the Tillsyn project, cleans up docs, adds `mage install` dev-promoted commit pinning (18.5), and establishes the MD-artifact baseline for later slices. Cascade dispatch does not exist yet — the orchestrator approximates it manually per `CLAUDE.md` §"Cascade Tree Structure".

## Project Invariants

- **Tillsyn is the system of record** for all coordination. No markdown worklogs, no plan items outside Tillsyn.
- **The orchestrator never edits Go code.** Every code change goes through a `go-builder-agent` subagent. Orchestrator may edit markdown docs in `main/` (CLAUDE.md, this wiki, plan docs, refinement files, agent `.md` files).
- **Hylla is primary for committed Go code.** `git diff` covers post-ingest deltas. Context7 + `go doc` + gopls MCP cover external semantics. See `HYLLA_WIKI.md` for Hylla usage patterns.
- **Mage-only build discipline.** Never raw `go build` / `go test` / `go vet`. Always `mage <target>`. `mage ci` before every push.
- **QA before commit.** Both proof and falsification QA pass before any commit lands. No batched commits.
- **Hylla reingest is slice-end only.** Once per slice, inside the `SLICE <N> END — LEDGER UPDATE` task, full enrichment from remote, only after CI green. Subagents never call `hylla_ingest`.

## Cascade Addressing (Slice 0 Convergence)

See `CLAUDE_MINIONS_PLAN.md` §1.4 for the full vocabulary. Summary:

- Project is the root — NOT a slice.
- Top-level slices are `slice_0`, `slice_1`, … zero-indexed.
- Sub-slices are `slice_sub_N` — zero-indexed among slice-kind children only.
- Dotted addresses (`0.1.5.2`, `tillsyn-0.1.5.2`) are **read-only shorthand**. Mutations always use UUIDs.
- Type-slice kinds (post-Slice-3): `plan-slice`, `build-slice`, `qa-slice`, `closeout-slice`, `refinement-slice`, `human-verify-slice`, `discussion-slice`. Pre-Slice-3 they exist as naming conventions + labels; Slice 3 encodes them as template kinds.

## Pre-Cascade Workflow (Orchestrator-as-Hub)

Until Slice 4's dispatcher lands, the parent Claude Code session IS the orchestrator. It plans, routes, delegates, cleans up — never edits Go code.

1. **Plan** — orchestrator (or, at slice-level, a `go-planning-agent` subagent) decomposes the work into plan items with paths/packages/acceptance criteria.
2. **Build** — `go-builder-agent` subagent implements the increment. Auth + lease + Tillsyn credentials in the spawn prompt; durable task content in the plan-item description.
3. **QA** — `go-qa-proof-agent` + `go-qa-falsification-agent` run in parallel, each with fresh context. Both must pass.
4. **Commit** — orchestrator + dev (pre-Slice-11) commit with conventional-commit format.
5. **Push + CI green** — `git push` then `gh run watch --exit-status` until green.
6. **Update Tillsyn** — metadata, completion notes, move to terminal state.
7. **Next task** — no per-task Hylla reingest. Reingest is slice-end only.

## Slice-End Closeout

Every slice ends with a `SLICE <N> END — LEDGER UPDATE` task:

1. All sibling tasks `done`. `git status --porcelain` clean.
2. All commits on remote. CI green (`gh run watch --exit-status`).
3. Aggregate per-subagent `## Hylla Feedback` sections into `HYLLA_FEEDBACK.md`.
4. `hylla_ingest` full enrichment from remote.
5. Append ledger entry to `LEDGER.md`.
6. Append one-liner to `WIKI_CHANGELOG.md`.
7. Update relevant sections of this wiki if anything shipped that changed best practice.

## Related Files

- `CLAUDE.md` — canonical project rules (bare-root + main/ carry the same body).
- `CLAUDE_MINIONS_PLAN.md` — cascade design and slice ordering.
- `LEDGER.md` — per-slice snapshot of cost, node counts, orphan deltas, commit SHAs.
- `WIKI_CHANGELOG.md` — one-liner per slice mirroring what landed.
- `HYLLA_WIKI.md` — Hylla usage best practices (query hygiene, schema gotchas).
- `HYLLA_FEEDBACK.md` — per-slice aggregation of subagent-reported Hylla misses.
- `HYLLA_REFINEMENTS.md` — append-only log of Hylla ergonomics + search-quality refinement candidates.
- `REFINEMENTS.md` — append-only log of Tillsyn product refinements + TUI/CLI/MCP ergonomics issues.
