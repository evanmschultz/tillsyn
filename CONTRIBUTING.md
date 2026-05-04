# Contributing

This project uses a small `mage` surface plus GitHub Actions gates. Run `mage` from the worktree root. Use `mage test-pkg <pkg>` for focused loops and `mage ci` for the full gate.

## Local Workflow

Use this loop while developing:

```bash
mage test-pkg ./cmd/till
```

Before every push (or PR update), run the full gate:

```bash
mage ci
```

`mage` target intent:
- `mage test-pkg <pkg>`: focused test loop (`mage test-pkg ./...` runs the full suite without coverage)
- `mage ci`: canonical full gate (`verify-sources`, `gofmt` check, coverage-enforced tests, `build`)
- `mage build`: local binary build
- `mage run`: run from source
- `mage dev`: run from source against the repo-local `./.tillsyn` dev runtime
- `mage install`: install `till` into `~/.local/bin`
- `mage format`, `mage format-path <path>`, `mage format-check`: gofumpt write (whole tree / scoped) and CI-parity format gate
- `mage install-hooks`: one-time-per-clone activation of the tracked `.githooks/` scripts (`core.hooksPath = .githooks`)
- `mage test-golden`, `mage test-golden-update`: focused TUI golden workflows

## Windows Note (Line Endings)

`gofmt` checks require LF line endings for Go files. This repository includes `.gitattributes` rules to enforce that on checkout.

If a local Windows clone still reports mass `gofmt required for:` failures, fix Git EOL settings and refresh files:

```bash
git config --global core.autocrlf false
git config --global core.eol lf
git add --renormalize .
```

If line endings are still stale after renormalization, re-clone the repository.

## Local Git Hooks

This repo ships two POSIX `sh` hooks at `.githooks/` that gate gofumpt drift on every commit and push. Full test runs (`mage ci`) live in GitHub Actions only — local hook context has env-coupling risks (gitdiff tests collide with bare-root config lock during concurrent push), and CI runs the same `mage ci` in clean Docker with no parent repo. Activation is one-time-per-clone via a tracked mage target:

```bash
mage install-hooks
```

That sets `core.hooksPath = .githooks` for this clone so the tracked scripts run on every `git commit` / `git push`. Sanity check:

```bash
git config --get core.hooksPath
# expected: .githooks
```

The two hooks:

- `.githooks/pre-commit` runs `mage format-check` — catches gofumpt drift before the commit lands. On failure it suggests `mage format` to auto-fix.
- `.githooks/pre-push` runs `mage format-check` — same fast format gate before push. The full `mage ci` (tests + coverage + build) runs in GitHub Actions on every push.

Bypass policy: `git commit --no-verify` and `git push --no-verify` are honored by git natively. Per dev discipline, never bypass without an explicit reason captured in the commit message or PR description.

If you have local hooks in `.git/hooks/`, copy them into `.githooks/` before running `mage install-hooks` — `core.hooksPath` overrides the default lookup, so untracked local hooks would otherwise stop firing.

Ensure `mage` is on your `PATH` so the hooks find it. Install Mage locally with:

```bash
go install github.com/magefile/mage@v1.17.0
```

For `mage -h <target>` help lookup use the canonical camelCase name (e.g. `mage -h format`); the kebab-case forms (`format-check`, `install-hooks`) work for invocation but Mage's help resolver doesn't follow aliases.

## GitHub Actions Model

CI runs:
- `mage ci` on all OSes (`ubuntu-latest`, `macos-latest`, `windows-latest`)
- release snapshot validation after the matrix passes

Concurrency policy:
- `main` branch runs are not canceled in progress.
- Non-main branch runs can cancel superseded in-progress runs.

## Branch Protection Recommendation

Require these checks on `main`:
- `ci (ubuntu-latest)`
- `ci (macos-latest)`
- `ci (windows-latest)`
- `release snapshot check`

## Notes

Local gates greatly reduce CI surprises, but they do not fully replace GitHub-hosted OS/environment validation.

## Dev MCP Server Setup

When hacking on Tillsyn itself, test against the worktree's own built binary — not the version installed on your system. Each worktree registers its own MCP server entry pointing at that worktree's `till` binary, so changes you make in one worktree don't leak into others.

**Naming rule:** MCP server names cannot contain `.`, so `drop/1.5` maps to `drop-1-5`. Use a unique suffix per worktree so binaries from different worktrees can't collide. The canonical pattern is `tillsyn-dev` for the `main/` worktree and `tillsyn-dev-<branch-slug>` for every other worktree.

**Setup (from each worktree):**

```bash
mage build
claude mcp add --scope local <unique-name> -- /absolute/path/to/worktree/till serve-mcp
```

For example, from `main/`:

```bash
mage build
claude mcp add --scope local tillsyn-dev -- /Users/evanschultz/Documents/Code/hylla/tillsyn/main/till serve-mcp
```

And from `drop/1/`:

```bash
mage build
claude mcp add --scope local tillsyn-dev-drop-1 -- /Users/evanschultz/Documents/Code/hylla/tillsyn/drop/1/till serve-mcp
```

**Active registrations in this repo's dev environment:**

- `main` (STEWARD): `tillsyn-dev` → `<repo>/main/till serve-mcp`
- `drop/1` (DROP_1_ORCH): `tillsyn-dev-drop-1` → `<repo>/drop/1/till serve-mcp`
- `drop/1.5` (DROP_1.5_ORCH): `tillsyn-dev-drop-1-5` → `<repo>/drop/1.5/till serve-mcp`

**After every `mage build`** in a given worktree, that worktree's binary updates in place and MCP picks up the change on the next invocation — no re-registration needed. Orchestrators reference their worktree's MCP name (not the generic `tillsyn-dev`) unless they're the STEWARD session launched from `main/`. Confirm with `claude mcp list`.

**When retiring a worktree,** remove its MCP entry with `claude mcp remove <name>`.
