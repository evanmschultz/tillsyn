# Repository Guidelines

This file defines persistent repo-wide instructions for coding agents working in this repository. It is not runtime behavior for `tillsyn`.

Keep branch-specific, worktree-specific, and temporary phase-specific process in:
- `PLAN.md` for the active execution ledger and temporary run guidance.
- `worktrees/AGENTS.md` for linked-worktree and concurrent-branch workflow rules.

You are a senior Go dev. YOU ALWAYS:

- ALWAYS use Context7 for library and API documentation before writing any code.
- ALWAYS re-run Context7 after any test failure or runtime error before making the next edit.
- If Context7 is unavailable (quota, network, outage), record the fallback source before proceeding (for example official docs, `go doc`, or package-local docs).
- For instruction/policy context, use `till.get_instructions` on-demand (missing/stale/ambiguous guidance), not on every step.
- For `till.get_instructions`, keep context bounded by default:
  - set `doc_names` explicitly,
  - use `max_chars_per_doc` on long docs,
  - use `include_markdown=false` for inventory checks and `include_markdown=true` only when full text is needed.
- Treat all project/task details and all thread comment content as markdown-first authoring surfaces.
- In MCP calls, write markdown-formatted content for `description`, `summary`, and `body_markdown` fields.
- Write idiomatic Go doc comments for all top-level declarations and methods in production and test code, and add inline comments for non-obvious behavior blocks (including behavior blocks in `*_test.go`).
- Review `Justfile` at startup and use its recipes as the source of truth for local automation.
- Run tests/checks through `just` recipes only; do not run `go test` directly from the agent.
- Run `just` recipes directly (for example `just ci`) without `GOCACHE=...` or other cache-path env overrides unless the user explicitly asks for an override.
- Do not create workspace-local ad-hoc Go cache directories (for example `.go-cache-*`) during normal test/check execution.
- During normal implementation loops, run `just check` after meaningful increments to catch local regressions early.
- When you touch Go code, finish by running `just ci` unless the user explicitly approves a narrower suite.
- Before asking the user to push or before opening/refreshing a PR, run `just ci` and report results.
- After pushing a change that is meant to fix or validate CI, run `gh run watch --exit-status` on the new GitHub Actions run and do not claim CI passes until the remote run finishes green.
- Prefer `gh` for GitHub-hosted operations whenever `gh` supports the task directly and clearly.
- Use `gh` by default for pull requests, workflow/check inspection, run logs, review actions, repository metadata, and GitHub authentication.
- Use `git` for core local repository operations such as status, diff, add, commit, branch, merge-base inspection, and worktree management, unless the current conversation explicitly requires a `gh`-specific workflow.
- Do not use the GitHub web UI for repository operations when `gh` can perform the same task.
- Use Conventional Commits for all commit messages.
- Format commit messages as `type(scope): short imperative summary` when a scope is useful, otherwise `type: short imperative summary`.
- Write commit messages in lowercase by default; preserve uppercase only for required literals such as `GitHub`, `MCP`, `TUI`, `Codex`, `OpenAI`, code identifiers, or file/path names.
- Keep commit summaries concise, imperative, and without a trailing period.
- Prefer one primary intent per commit.
- Allowed commit types: `feat`, `fix`, `docs`, `test`, `refactor`, `chore`, `perf`, `build`, `ci`.
- Contributors and agents should follow this commit style consistently.
- If you touch `.github/workflows/` or `Justfile`, run both `just check` and `just ci` before handoff.
- Add package-scoped `Justfile` recipes when needed for fast iteration, then still finish with `just ci`.
- Treat runtime logging as a first-class implementation concern:
  - use `github.com/charmbracelet/log` as the canonical logger for application/runtime logs.
  - keep colored/styled console output enabled for local developer ergonomics.
  - in dev mode, write logs to a workspace-local `.tillsyn/log/` directory so logs are easy to inspect during debugging.
  - log meaningful runtime operations and failures (startup paths/config load, persistence/migrations, mutating actions, recoverable/non-recoverable errors).
- During troubleshooting, inspect recent local log files before proposing fixes and include relevant findings in your reasoning.
- Keep error handling idiomatic:
  - wrap errors with `%w`,
  - return errors upward at clean boundaries,
  - log context-rich failures at adapter/runtime edges instead of swallowing errors.
- If dependency updates need network access, ask the user to run `go get` and module update commands in their own shell.
- Never use dependency-fetch bypasses (for example `GOPROXY=direct`, `GOSUMDB=off`, or checksum bypass flags).
- Never delete files or directories without explicit user approval.
- Never run commands outside this repository root: `/Users/evanschultz/Documents/Code/hylla/tillsyn`.
- For runtime/protocol validation in this phase, run MCP-only checks (no HTTP/curl validation probes).
- It is allowed to `just build` and run `./till serve` locally for MCP-side validation.
- Never push to any remote unless the user explicitly requests it in the current conversation.
- Keep the active execution/work log in `PLAN.md`. Use `worklogs/` only when the user explicitly asks for split logs.
- Treat `PLAN.md` as the active source of truth for temporary run state, acceptance checklists, commands run, evidence, and completion state.
- Keep `PLAN.md` single-writer by default:
  - only the orchestrator/integrator updates run completion state there,
  - worker lanes provide handoff notes unless explicitly assigned to update `PLAN.md`.
- When proposing new implementation phases, explicitly review the active backlog and open discussion items in `PLAN.md` first.
- When clarification is needed, ask in two stages:
  - first ask general goal-alignment questions and lock shared objectives,
  - only after that consensus ask specific implementation-detail questions.

## Project Structure

- `cmd/till`: CLI/TUI entrypoint.
- `internal/domain`: core entities and invariants.
- `internal/app`: application services and use-cases (ports-first, hexagonal core).
- `internal/adapters/storage/sqlite`: SQLite persistence adapter.
- `internal/config`: TOML loading, defaults, validation.
- `internal/platform`: OS-specific config/data/db path resolution.
- `internal/tui`: Bubble Tea/Bubbles/Lip Gloss presentation layer.
- `.artifacts/`: generated local outputs (exports, temporary build outputs).
- `PLAN.md`: active roadmap and execution/work log.
- `WORKTREE_WORKFLOW.md`: repository/worktree topology and concurrent-branch workflow guidance.

## Build and Run

- `just run`: run app from source (`go run ./cmd/till`).
- `just build`: build local binary `./till`.
- `just fmt`: format Go files.
- `just check`: cross-platform smoke gate (source verification, format check, tests, build).
- `just test`, `just test-pkg <pkg>`: test entrypoints.
- `just test-golden`, `just test-golden-update`: golden fixture validation/update.
- `just ci`: canonical full gate (source verification, format check, coverage-verified tests, build).

## Worklogs

- Use `PLAN.md` as the live execution ledger.
- Every meaningful checkpoint should capture:
  - current objective/plan,
  - commands/tests run and outcomes,
  - file edits and why,
  - failures/remediation,
  - current status and next step.
- Temporary or wave-specific workflow detail belongs in `PLAN.md`, not in this file.

## Tech Stack

- Go 1.26+
- Bubble Tea v2, Bubbles v2, Lip Gloss v2
- SQLite (`modernc.org/sqlite`, no CGO)
- TOML config (`github.com/pelletier/go-toml/v2`)

## Core Coding Paradigms

- Hexagonal architecture (ports/adapters), interface-first boundaries, dependency inversion.
- Ship small, testable increments; prioritize maintainability and pragmatic MVP progress.
- TDD-first where practical: tests before implementation for new behavior.
- Preserve Go idioms: clear naming, wrapped errors (`fmt.Errorf("...: %w", err)`), import grouping stdlib -> third-party -> local.
- Keep TUI mode transitions explicit and test-covered.

## Testing Guidelines

- Tests are co-located as `*_test.go`.
- Prefer table-driven tests and behavior-oriented assertions.
- Run package-focused loops with `just test-pkg <pkg>` during implementation.
- For substantial TUI changes, update or add tea-driven tests and golden fixtures.
- Coverage below 70% is a hard failure.
- Build/test execution must go through `just` recipes only.
- Do not wrap `just` test commands with custom Go cache env vars by default; use plain `just` invocations.

## Release and Security

- Keep release/Homebrew work in roadmap unless explicitly requested for execution.
- Keep secrets out of config files committed to the repository.
- Prefer environment overrides for machine-local sensitive settings.
