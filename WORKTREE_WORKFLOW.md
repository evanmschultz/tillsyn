# Bare Repo And Worktree Workflow

## Recommended Model

Use a true bare control repo at:

- `/Users/evanschultz/Documents/Code/hylla/tillsyn`

Use a normal checked-out integration worktree at:

- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main`

Use additional linked worktrees at operator-chosen paths, typically:

- `/Users/evanschultz/Documents/Code/hylla/tillsyn/.tmp/<branch-slug>`

This matches the local `laslig` pattern:
- the root directory is the Git control repo,
- `main/` is the steady-state operator checkout,
- linked worktrees share one object store and one ref database.

## Layout

### Bare root

The bare root keeps:
- Git control files and directories such as `HEAD`, `config`, `objects/`, `refs/`, `hooks/`, `logs/`, and `worktrees/`.
- local-only operator files such as `AGENT_PROMPTS.md`.
- local utility/output paths such as `.tmp/` and `.artifacts/`.
- the checked-out `main/` worktree.

### `main/`

The `main/` worktree keeps the tracked repository contents:
- source directories,
- repo docs,
- `PLAN.md`,
- `Justfile`,
- `worklogs/`,
- all normal repo-root files.

### Important caveat

At the bare root, `worktrees/` is Git admin state. Do not use `/Users/evanschultz/Documents/Code/hylla/tillsyn/worktrees/<name>` as a checkout path.

Use `.tmp/<name>` or another explicit sibling path instead.

## Migration And Rollback

### Migration shape used here

1. Convert the old root checkout into a bare control repo.
2. Create `main/` as the checked-out `main` branch worktree.
3. Keep `AGENT_PROMPTS.md` at the bare root as local-only material.
4. Move the old direct-root checkout into a local safety backup directory until the user decides it can be removed.

### Rollback

If the bare-root model needs to be undone:

1. Stop creating or modifying additional linked worktrees.
2. Preserve the current bare root and `main/` worktree state.
3. Choose one rollback target:
   - restore the old direct-root checkout from `.pre-bare-root-backup/`, or
   - re-create a normal non-bare checkout elsewhere from the same refs.
4. Only remove the backup directory after the restored layout is validated.

## Branch And Worktree Policy

1. `main/` is the operator and integration worktree.
2. Treat `main/PLAN.md` as the authoritative execution ledger.
3. Branch worktrees should treat their own checked-out `PLAN.md` copy as reference-only unless the user explicitly assigns an integration update.
4. Keep one live branch per linked worktree.
5. Do not check out the same branch in two worktrees.
6. Do not use `--ignore-other-worktrees`.
7. Prefer linked worktree paths under `.tmp/` unless the user specifies another location.
8. Use branch names that encode ownership and purpose, for example:
   - `agent/<topic>`
   - `user/<topic>`
   - `fix/<topic>`
   - `docs/<topic>`

## Merge Policy

### Default

Use merge commits for reviewed multi-commit or long-running worktree branches.

Why:
- preserves branch context,
- keeps agent-lane history visible,
- avoids repeated history rewrites on branches that are already referenced in `PLAN.md` or review notes.

### Rebase

Use rebase only on unpublished or private short-lived branches when:
- refreshing on top of current `main`,
- cleaning up local commit order before review,
- resolving drift before handoff to the integrator.

Do not rebase a branch that:
- is already shared with another agent or human,
- is already referenced in shared run notes,
- is acting as an integration base for another lane.

### Squash

Use squash only for narrow self-contained changes where the intermediate commits add little value, for example:
- typo-only docs,
- tiny operational scaffolding,
- very small cleanup fixes.

Do not squash long-running parallel branches by default.

### Long-running parallel branches

For branches that live longer than one integration cycle:

1. Merge `main` into the branch periodically instead of repeatedly rebasing it.
2. Re-run the branch-local validation after each refresh.
3. If collision risk rises, either:
   - serialize the overlapping work,
   - split one branch so the contested file moves to a follow-up branch,
   - or move the overlap into the integration lane.

## Collision Management

1. Assign each concurrent run an explicit lock scope before work starts.
2. Treat `main/PLAN.md` as single-writer by default.
3. Reserve these shared files for serialized ownership unless the user explicitly assigns a docs or integration lane:
   - `PLAN.md`
   - `AGENTS.md`
   - `WORKTREE_WORKFLOW.md`
   - `README.md`
   - `CONTRIBUTING.md`
   - `.github/workflows/**`
   - `Justfile`
4. Keep these hotspot Go files serialized unless the operator intentionally queues them:
   - `internal/tui/model.go`
   - `internal/app/service.go`
   - `internal/adapters/storage/sqlite/repo.go`
5. If two runs need the same file, stop and choose one:
   - serialize the later run behind the first,
   - refactor so scopes no longer overlap,
   - or move the overlap into the integration lane.
6. If a collision is discovered after both branches already changed the same file:
   - freeze the later-started branch,
   - integrate the earlier branch first,
   - merge `main` into the later branch,
   - re-validate only after the collision is resolved.

## Daily Commands

Create a new linked worktree and branch:

```bash
git worktree add .tmp/agent-example -b agent/example main
```

List all worktrees:

```bash
git worktree list
```

Show the correct hooks path for the current layout:

```bash
git rev-parse --git-path hooks/pre-push
```

Remove a linked worktree after it is fully integrated and the user has approved cleanup:

```bash
git worktree remove .tmp/agent-example
```
