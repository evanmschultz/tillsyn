# DROP_N — EXAMPLE (Illustrative Closed Drop)

> **Pedagogical example.** This drop dir shows the shape of a closed cascade
> drop — `PLAN.md`, `BUILDER_WORKLOG.md`, `CLOSEOUT.md`, plus the audit trail
> for one plan-QA round (`PLAN_QA_PROOF.md`). It is a generic illustration,
> not a real project's work. Read the files in phase order (Plan → Plan QA →
> Build Worklog → Closeout) to see how content evolves across a drop.
>
> The scenario is a fictional "add CLI scaffold + CI workflow" drop in a
> generic Go project — simple enough to walk through without domain
> knowledge, structured enough to show droplet granularity, `blocked_by`
> wiring, QA findings, and closeout aggregation.

**State:** done
**Blocked by:** —
**Paths:** `cmd/<PROJECT>/main.go`, `cmd/<PROJECT>/root.go`, `magefile.go`, `.github/workflows/ci.yml`
**Packages:** `github.com/<org>/<PROJECT>/cmd/<PROJECT>` (only package with Go code in this drop)
**PLAN.md ref:** `<PROJECT>/PLAN.md` → `DROP_N_EXAMPLE` row
**Workflow:** `drops/WORKFLOW.md`
**Cascade concept:** `AGENT_CASCADE_DESIGN.md`
**Started:** YYYY-MM-DD
**Closed:** YYYY-MM-DD

## Scope

Scaffold the CLI entry point, the mage build automation file, and the GitHub Actions CI workflow for `<PROJECT>`. No internal packages land in this drop — they follow in later drops. The goal is a repo that builds green, runs one smoke-level CLI command, and has CI enforcing `mage ci` on every PR.

Decomposition: three droplets (N.1, N.2, N.3) in a short chain. Each touches a distinct file set, and N.3 depends on N.2 (CI workflow invokes `mage ci`, which must exist first).

## Planner

Three atomic droplets implementing the scaffold. Dependency DAG:

```
N.1 ──▶ N.2 ──▶ N.3
```

N.1 creates the CLI scaffold (`cmd/<PROJECT>/`). N.2 adds the `magefile.go` with the canonical targets. N.3 adds the CI workflow that invokes `mage ci`.

### Droplet N.1 — Scaffold CLI entry point

- **State:** done
- **Paths:**
  - `cmd/<PROJECT>/main.go` (new — ~30 LOC: `package main` + `main()` calling `fang.Execute(ctx, newRootCmd())`)
  - `cmd/<PROJECT>/root.go` (new — ~100 LOC: `newRootCmd() *cobra.Command`, flag wiring, `RunE` stub)
- **Packages:** `github.com/<org>/<PROJECT>/cmd/<PROJECT>`
- **Acceptance:**
  - `cmd/<PROJECT>/main.go` exists; body is ≤~30 LOC; exactly one function `main`.
  - `cmd/<PROJECT>/root.go` exists; exports `newRootCmd`; `RunE` returns nil for a basic smoke invocation.
  - Compile check deferred to drop-end `mage ci` once droplet N.2 lands the magefile. Raw `go build` is forbidden per `CLAUDE.md` § "Build Verification" rule 2, so no per-droplet compile assertion is possible until the mage target exists.
- **Blocked by:** —

### Droplet N.2 — Add magefile.go with canonical targets

- **State:** done
- **Paths:** `magefile.go` (new — ~80 LOC)
- **Packages:** — (magefile is build-automation, not a compiled package)
- **Acceptance:**
  - `magefile.go` at repo root with targets: `Build`, `Test`, `Format`, `Lint`, `CI`, `Run`.
  - `mage -l` lists all six targets.
  - `mage build` exits 0; `mage test` exits 0 (no test files yet is expected — `[no test files]` banner per package is acceptable); `mage ci` exits 0.
- **Blocked by:** N.1

### Droplet N.3 — Add GitHub Actions CI workflow

- **State:** done
- **Paths:** `.github/workflows/ci.yml` (new — ~40 LOC)
- **Packages:** —
- **Acceptance:**
  - `.github/workflows/ci.yml` triggers on push + PR.
  - Workflow steps: checkout, setup-go, install mage, run `mage ci`.
  - After push, `gh run watch --exit-status` exits green.
- **Blocked by:** N.2

## Notes

- Library choices (cobra + fang for CLI, mage for build automation) are fixed at the project level in `CLAUDE.md`. The planner did not re-litigate them here.
- `internal/*` packages are explicitly out of scope — they land in later drops once the CLI surface stabilizes.
- `fang.WithNotifySignal` wiring deferred until a later drop when there is cancellable work worth signaling. Logged as a refinement at closeout.
- Go version pinned at 1.26+ per `CLAUDE.md`. CI workflow uses `actions/setup-go@v5` with `go-version-file: go.mod` to keep the pin single-sourced.
