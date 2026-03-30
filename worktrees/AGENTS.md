# Linked Worktree Guidelines

This file applies to linked worktrees created under `/Users/evanschultz/Documents/Code/hylla/tillsyn/worktrees`.

The repository root worktree at `/Users/evanschultz/Documents/Code/hylla/tillsyn` is the operator and integration worktree. Linked worktrees under this directory are for concurrent branch-specific runs.

Repo-wide instructions from the root `AGENTS.md` still apply. This file adds tighter rules for concurrent branches.

You ALWAYS:

- Confirm the current worktree before editing:
  - `pwd`
  - `git branch --show-current`
  - `git rev-parse --git-common-dir`
- Keep one live branch per worktree path.
- Do not check out the same branch in two worktrees.
- Do not use `--ignore-other-worktrees`.
- Treat `PLAN.md` as single-writer in the root worktree by default. Do not edit it from a linked worktree unless the operator explicitly assigns that lane.
- Treat shared operator docs as serialized files. Do not edit these from a linked worktree unless the lane is explicitly the integration/docs lane:
  - `AGENTS.md`
  - `WORKTREE_WORKFLOW.md`
  - `README.md`
  - `CONTRIBUTING.md`
  - `worklogs/**`
  - `.github/workflows/**`
  - `Justfile`
- Stay inside the assigned lock scope. If a needed change escapes that scope, stop and hand it back to the integrator instead of expanding the lane informally.
- Keep hotspot files serialized unless the operator explicitly assigns ownership:
  - `internal/tui/model.go`
  - `internal/app/service.go`
  - `internal/adapters/storage/sqlite/repo.go`
- Run package-scoped checks only (`just test-pkg <pkg>`) unless the operator explicitly assigns wider validation.
- Do not run repo-wide gates (`just test`, `just check`, `just ci`) from a linked worktree unless the operator explicitly assigns the integration step to that lane.
- If the branch drifts from `main`, prefer merging `main` into the branch over rebasing once the lane is shared or reviewed.
- If another concurrent lane needs the same file, stop and surface the collision immediately. Do not force through an ad hoc overlap.

Required handoff content:

- lane id or branch name,
- files changed and why,
- commands run and pass/fail outcomes,
- acceptance criteria status,
- unresolved risks/blockers,
- recommended next step.
