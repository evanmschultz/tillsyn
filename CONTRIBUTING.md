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

## Recommended Pre-Push Hook

Install a local hook so pushes fail fast if `mage ci` fails:

```bash
hook_path="$(git rev-parse --git-path hooks/pre-push)"
cat > "$hook_path" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
mage ci
EOF
chmod +x "$hook_path"
```

Use `git rev-parse --git-path ...` instead of hard-coding `.git/...` so the hook still lands in the right place when the repo uses a separate common git dir.

Install Mage locally with:

```bash
go install github.com/magefile/mage@v1.17.0
```

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
