# Contributing

This project uses local `just` gates plus GitHub Actions gates. Run local gates before push so most failures are caught before CI.

## Local Workflow

Use this loop while developing:

```bash
just check
```

Before every push (or PR update), run the full gate:

```bash
just ci
```

`just` gate intent:
- `just check`: cross-platform smoke gate (`verify-sources`, `fmt-check`, `test`, `build`)
- `just ci`: full gate (`verify-sources`, `fmt-check`, coverage-enforced test run, `build`)

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

Install a local hook so pushes fail fast if `just ci` fails:

```bash
hook_path="$(git rev-parse --git-path hooks/pre-push)"
cat > "$hook_path" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
just ci
EOF
chmod +x "$hook_path"
```

Use `git rev-parse --git-path ...` instead of hard-coding `.git/...` so the hook still lands in the right place when the repo uses a separate common git dir.

## GitHub Actions Model

CI is intentionally split:
- Matrix smoke checks on all OSes (`ubuntu-latest`, `macos-latest`, `windows-latest`) run `just check`.
- Full coverage gate runs once on Linux and executes `just ci`.
- Release snapshot validation runs only after the Linux full gate passes.

Concurrency policy:
- `main` branch runs are not canceled in progress.
- Non-main branch runs can cancel superseded in-progress runs.

## Branch Protection Recommendation

Require these checks on `main`:
- `check (ubuntu-latest)`
- `check (macos-latest)`
- `check (windows-latest)`
- `full gate (ubuntu-latest)`
- `release snapshot check`

## Notes

Local gates greatly reduce CI surprises, but they do not fully replace GitHub-hosted OS/environment validation.
