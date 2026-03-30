# `third_party/teatest_v2`

This directory is a local compatibility patch for
`github.com/charmbracelet/x/exp/teatest/v2`.

## Why this exists

`kan` uses Bubble Tea v2 packages from `charm.land/*`. Upstream
`x/exp/teatest/v2` has periodically drifted from those versions, so we pin and
patch a local copy to keep TUI test behavior stable and CI deterministic.

The root module wires this in with:

```go
replace github.com/charmbracelet/x/exp/teatest/v2 => ./third_party/teatest_v2
```

## Maintenance rules

1. Keep this module minimal and focused on compatibility only.
2. If you change `teatest.go` or imports here, run:
   `cd third_party/teatest_v2 && go mod tidy`
3. Commit both `go.mod` and `go.sum` in this directory.

## Common IDE/gopls error

If you see:

`packages.Load error: go: updates to go.mod needed; to update it: go mod tidy`

that error is from this nested module (not usually the repo root). It means
`third_party/teatest_v2/go.mod` and/or `third_party/teatest_v2/go.sum` are out
of sync for imports in `teatest.go`.

Fix:

```bash
cd third_party/teatest_v2
go mod tidy
```

## When to remove this directory

Remove this patch once upstream `github.com/charmbracelet/x/exp/teatest/v2`
works directly with the repo's Bubble Tea/Lip Gloss stack. At that point:

1. Remove the `replace` line from root `go.mod`.
2. Remove `third_party/teatest_v2`.
3. Re-run repository verification (`mage ci`).
