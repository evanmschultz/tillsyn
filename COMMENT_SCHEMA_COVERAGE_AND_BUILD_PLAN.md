# Comment Schema Coverage and Build Plan

Date: 2026-03-02
Scope: comment/event ownership tuple, target-type coverage, migration behavior, and TUI ownership rendering.

## Current Implemented State (Post-Wave 2)

### Comments table (`comments`)
Source: `internal/adapters/storage/sqlite/repo.go` migration path.

Canonical columns:
- `id`
- `project_id`
- `target_type`
- `target_id`
- `body_markdown`
- `actor_id`
- `actor_name`
- `actor_type`
- `created_at`
- `updated_at`

Notes:
- Runtime read/write path uses canonical tuple columns only.
- Legacy `author_name` handling is migration-time conversion only (no runtime dual path).

### Change events table (`change_events`)
Canonical ownership tuple for activity rows:
- `actor_id`
- `actor_name`
- `actor_type`

Task mutation events now persist `actor_name` when mutation actor context provides it.

### Ownership tuple contract
Implemented tuple everywhere in this wave:
- `actor_id` (stable identity token)
- `actor_name` (human label snapshot)
- `actor_type` (`user|agent|system`)

Identity source behavior:
- MCP adapter derives deterministic actor identity with precedence:
  - `actor_id`: explicit `actor_id` > `agent_instance_id` > `agent_name` > `tillsyn-user`
  - `actor_name`: explicit `actor_name` > `agent_name` > `actor_id`
- TUI uses config-backed `identity.actor_id` plus display name.

### Target-type coverage
Comment target coverage remains complete:
- `project`, `branch`, `phase`, `subphase`, `task`, `subtask`, `decision`, `note`

## Migration + Versioning Decisions

1. DB migration is additive/transformative with one canonical runtime path (no legacy shim path).
2. Snapshot ownership-shape change is explicit via version bump:
- `SnapshotVersion = tillsyn.snapshot.v2`
- Import requires exact supported version.

## Verification Evidence

Scoped and full gates passed:
1. `just test-pkg ./internal/domain`
2. `just test-pkg ./internal/app`
3. `just test-pkg ./internal/adapters/storage/sqlite`
4. `just test-pkg ./internal/adapters/server/mcpapi`
5. `just test-pkg ./internal/tui`
6. `just test-pkg ./cmd/till`
7. `just check`
8. `just ci`

Independent remediation review status:
- PASS (`REVIEW-REMEDIATION-PASS2`)
