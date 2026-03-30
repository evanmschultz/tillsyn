# Repo And Worktree Workflow

## Recommended Model

Use a separate common git dir plus linked worktrees:

1. Keep the current checkout at `/Users/evanschultz/Documents/Code/hylla/tillsyn` as the operator and integration worktree.
2. Move repository metadata into `/.git-common/` and leave `/.git` as a gitfile that points there.
3. Create concurrent branch worktrees under `/Users/evanschultz/Documents/Code/hylla/tillsyn/worktrees/<branch-slug>/`.

This keeps the current working files, `PLAN.md`, worklogs, and operator-facing docs accessible from the cwd while still giving one shared git object store and clean multi-branch worktrees.

## Why This Model

### Preferred: separate common git dir + linked worktrees

Pros:
- preserves the current cwd as a normal working tree,
- uses supported Git behavior (`git init --separate-git-dir`, `git worktree add`),
- keeps rollback straightforward,
- avoids duplicate object databases across concurrent branches,
- makes multi-branch agent work explicit without forcing a new operator home directory.

Tradeoffs:
- the cwd remains the special "main worktree",
- `.git` becomes a gitfile, so any hard-coded `.git/...` path usage must be replaced with `git rev-parse --git-path ...`,
- in this in-place layout, `git worktree list` may still show `/.git-common/` as the main anchor path even though `core.worktree` points back to the cwd,
- shared config stays common unless there is a later need to enable `extensions.worktreeConfig`.

### Rejected for now: true bare repo at the current cwd

Why not:
- a true bare repository has no working tree,
- it would remove the current working files from the cwd,
- it directly conflicts with the requirement to keep worklogs and operator-facing materials accessible from the current directory.

### Deferred: bare central repo plus current cwd as a linked worktree

Why not yet:
- Git supports bare repositories with linked worktrees,
- but converting this existing non-empty checkout into a linked worktree in place is materially riskier than converting to a separate common git dir,
- it adds complexity without solving a current problem that the recommended model leaves unsolved.

This can be revisited later if the team decides the "main worktree" concept is itself a problem.

### Rejected: raw `core.worktree` hand-managed layout

Why not:
- it is easier to misconfigure,
- it creates more foot-guns around repository discovery and config,
- Git explicitly treats `core.worktree` and `core.bare` as settings that need extra care when worktree-specific config is involved.

## Migration Plan

### Target state

1. `/.git` is a gitfile.
2. `/.git-common/` holds the shared git metadata.
3. `/worktrees/` is reserved for linked worktrees.
4. The cwd stays on `main` unless the operator explicitly chooses otherwise.
5. `PLAN.md` remains the single-writer execution ledger in the cwd worktree.

### Migration steps

1. Confirm current state:
   - `git status --short --branch`
   - `git worktree list --porcelain`
   - `git rev-parse --show-toplevel --git-dir --git-common-dir --is-bare-repository`
2. Add ignore/scaffold files:
   - ignore `/.git-common/`,
   - add `/worktrees/.gitignore`,
   - add `/worktrees/AGENTS.md`.
3. Convert the repository metadata in place:
   - `git init --separate-git-dir=.git-common`
4. Verify the conversion:
   - `git rev-parse --git-dir --git-common-dir --is-bare-repository`
   - `git status --short --branch`
   - `git worktree list --porcelain`
   - if `git worktree list` still shows `/.git-common/` first, confirm `git config --get core.worktree` still points at the cwd and treat that list output as a known display quirk of this model
5. Update operator docs that previously assumed `.git/` was a directory.
6. Validate from the cwd with `just check`.
7. Create future branch worktrees with `git worktree add`.

### Rollback steps

If the separate-git-dir layout causes an unexpected local-tooling problem:

1. Ensure no additional linked worktrees are active.
2. Move `/.git-common/` back to `/.git`.
3. Replace the `/.git` gitfile with the restored directory.
4. Re-run:
   - `git rev-parse --git-dir --git-common-dir --is-bare-repository`
   - `git status --short --branch`
5. Remove the now-unused `/.git-common/` ignore rule only after the restored layout is confirmed.

If linked worktrees already exist, remove or detach them first so the rollback is not racing live per-worktree metadata.

## Branch And Worktree Policy

1. The cwd worktree is the operator/integration worktree.
2. Keep `main` checked out in the cwd unless the user explicitly directs otherwise.
3. Create one live branch per linked worktree under `worktrees/<branch-slug>/`.
4. Do not check out the same branch in two worktrees.
5. Do not use `--ignore-other-worktrees`.
6. Treat root docs and execution ledgers as serialized resources:
   - `PLAN.md`
   - `AGENTS.md`
   - `WORKTREE_WORKFLOW.md`
   - `README.md`
   - `CONTRIBUTING.md`
   - `worklogs/**`
7. Default worker-lane behavior:
   - code changes happen in linked worktrees,
   - integration and shared-doc updates happen from the cwd.
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
- avoids repeated history rewrites on branches that may already be referenced in `PLAN.md` or review notes.

### Rebase

Use rebase only on unpublished or private branches when:
- refreshing a short-lived branch on top of current `main`,
- cleaning up local commit order before review,
- resolving drift before the branch is handed back to the integrator.

Do not rebase a branch that:
- is already shared with another agent or human,
- is already referenced in shared run notes,
- is being used as an active integration base for another lane.

### Squash

Use squash only for narrow, self-contained branches where the intermediate commits add little review value, for example:
- typo or copy-only doc fixes,
- one-off operational scaffolding,
- very small cleanup changes.

Do not squash long-running parallel branches by default; it hides the lane history that usually matters during multi-agent integration and rollback.

### Long-running parallel branches

For branches that live longer than one integration cycle:

1. Merge `main` into the branch periodically instead of repeatedly rebasing it.
2. Re-run the branch-local validation after each refresh.
3. When conflict risk rises, either:
   - serialize the overlapping work,
   - or split one branch so the contested file moves to a new follow-up branch.

## Collision Management

1. Assign each concurrent run an explicit lock scope before work starts.
2. Reserve these files for serialized ownership unless the user explicitly assigns a docs or integration lane:
   - `PLAN.md`
   - `AGENTS.md`
   - `WORKTREE_WORKFLOW.md`
   - `README.md`
   - `CONTRIBUTING.md`
   - `.github/workflows/**`
   - `Justfile`
3. Keep hotspot Go files serialized unless the operator intentionally queues them:
   - `internal/tui/model.go`
   - `internal/app/service.go`
   - `internal/adapters/storage/sqlite/repo.go`
4. If two runs need the same file, stop and choose one:
   - serialize the second run behind the first,
   - refactor to extract a new helper so scopes no longer overlap,
   - or move the overlapping change into the integration lane.
5. If a collision is discovered after both branches already changed the same file:
   - freeze the later-started branch,
   - integrate the earlier branch first,
   - merge `main` into the later branch,
   - re-validate only after the collision is resolved.

## Daily Commands

Create a new linked worktree and branch:

```bash
git worktree add worktrees/agent-example -b agent/example main
```

List all worktrees:

```bash
git worktree list
```

Show the correct hooks path for the current worktree layout:

```bash
git rev-parse --git-path hooks/pre-push
```

Remove a linked worktree after it is fully integrated and the user has approved cleanup:

```bash
git worktree remove worktrees/agent-example
```
