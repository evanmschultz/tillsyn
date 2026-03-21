# Repository Guidelines

This file defines instructions for coding agents working in this repository. It is not runtime behavior for `tillsyn`.

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
- In subagent parallel mode (single-branch orchestration), worker lanes may run scoped checks (`just test-pkg ...`), but the integrator must run `just ci` before marking a lane integrated/closed.
- In subagent prompts, explicitly require: Context7 before any code change, Context7 again after any failed test/runtime error, and package-scoped `just test-pkg` checks for touched packages.
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
- For the current auth/runtime remediation run, the active run section at the top of `PLAN.md` is the single source of truth for scope, status, acceptance checklist, commands run, test evidence, open questions, and completion state.
- For the current auth/runtime remediation run, all other planning or validation markdown files are reference-only unless `PLAN.md` explicitly points to them for corroborating evidence.
- For the current auth/runtime remediation run, worker and QA subagents must verify their acceptance criteria against the active run section at the top of `PLAN.md` first and treat mismatches between `PLAN.md` and secondary docs as a blocker that must be surfaced to the orchestrator.
- When proposing new implementation phases, you must explicitly review and discuss the active backlog and open discussion items in `PLAN.md` first; for the current run, treat older deleted collab docs as retired and do not depend on them.
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

## Build and Run

- `just run`: run app from source (`go run ./cmd/till`).
- `just build`: build local binary `./till`.
- `just fmt`: format Go files.
- `just check`: cross-platform smoke gate (source verification, format check, tests, build).
- `just test`, `just test-pkg <pkg>`: test entrypoints.
- `just test-golden`, `just test-golden-update`: golden fixture validation/update.
- `just ci`: canonical full gate (source verification, format check, coverage-verified tests, build).

## Worktrees

- Worktrees are optional but supported.
- If a worktree path is requested by the user, always `cd` into that exact path before editing, testing, or committing.
- Do not hard-code worktree names.
- Do not run completion/cleanup git actions (push, merge, rebase, worktree removal, branch deletion) without explicit user approval in the current conversation.

## Worklogs

- Use `PLAN.md` as the live execution ledger.
- For the current auth/runtime remediation run, treat the active run section at the top of `PLAN.md` as the only active checklist and completion ledger.
- Keep updates step-by-step while work is in progress. At minimum log:
  - current objective/plan,
  - each command/test run and outcome,
  - each file edit and why,
  - each failure and remediation,
  - current status and next step.
- In subagent parallel mode, `PLAN.md` is single-writer:
  - only the orchestrator/integrator updates lock tables, lane status, and completion markers.
  - worker subagents must not directly edit `PLAN.md`; they provide handoff notes for orchestrator ingestion.
- In the current auth/runtime remediation run, subagent handoffs and QA sign-off must map back to explicit checklist items in `PLAN.md` so the orchestrator can close the run from one file.
- Every orchestrator checkpoint update in `PLAN.md` must include command/test evidence:
  - commands run and outcomes,
  - tests/checks run and outcomes,
  - or explicit `test_not_applicable` with rationale for docs-only/process-only steps.

## Temporary Next-Step Directive (Collaborative Remediation Closeout)

- This temporary section is active for the current phase:
  - close remaining collaborative remediation gaps tracked in active collab docs,
  - prioritize dogfooding readiness for real user+agent workflows,
  - keep planning/roadmap state centralized in `PLAN.md`.
- Schema and compatibility policy for this temporary phase:
  - avoid backward-compatibility shims and dual read/write paths unless the user explicitly requests one,
  - when schema changes are required, implement one clean migration path and remove superseded legacy code in the same wave,
  - avoid leaving orphaned transitional code; track any unavoidable deferred cleanup as an explicit backlog item in `PLAN.md`.
- Orchestrator requirements:
  - plan for coexistent parallel subagents with explicit, non-overlapping file-lock scopes,
  - prevent workers from touching the same file concurrently,
  - perform explicit code review on every worker handoff before integration.
- Delivery requirements for this temporary phase:
  - keep docs synchronized as implementation lands (`README.md` and affected planning/testing docs),
  - ensure `just check` and `just ci` both pass before marking work complete,
  - keep `PLAN.md` as the remediation requirement/worklog source and active validation contract,
  - use `PLAN.md` itself for secondary corroborating checklist detail during this run unless the user explicitly asks for a new split worksheet,
  - do not recreate retired collab/runbook markdown unless the user explicitly asks for a new split document.
- Dogfooding requirement:
  - testing docs must support collaborative user+agent validation and clearly call out guardrails, blockers, and recovery workflows.
- Cleanup requirement:
  - after this temporary phase is confirmed complete by the user, explicitly ask how this temporary AGENTS directive should be removed or reduced.
- Locked execution flow for this temporary phase (section-by-section remediation):
  - use explicit `test/fix cycle (collab)` checkpoints:
    1. run one collaborative test step,
    2. log findings/evidence,
    3. if fixes are needed, implement and validate that one fix scope,
    4. commit the validated fix scope before starting the next fix scope.
  - if the working tree already contains uncommitted edits from a prior `test/fix cycle (collab)`, do not start another fix scope until those edits are either committed or explicitly discarded with user approval.
  - execute collaborative testing one section at a time (do not batch all findings for a later fix wave),
  - when a section reveals a failure/gap (bug), immediately log it in `PLAN.md` and the active worksheet, pause forward testing, and run a focused remediation loop for that section:
    1. spawn subagents to inspect code and gather local context,
    2. run Context7 research (and web research when needed) to collect fix options,
    3. present options to user and reach explicit user+agent consensus before implementation,
    4. implement fixes via scoped worker subagents under lock discipline,
    5. run package-scoped checks via `just test-pkg <pkg>` for touched packages,
    6. rerun the same section’s collaborative checks and record fresh evidence,
    7. only proceed to the next section after the current section is re-validated and documented.
  - record user findings with complete and accurate intent preservation; normalize terminology only when needed for technical correctness.

## Parallel/Subagent Mode

- This repository supports parallel subagent execution on a single branch only under lock discipline.
- Roles:
  - orchestrator: decomposition, lane assignment, lock ownership, approval escalation.
  - worker subagent: scoped implementation lane and evidence handoff.
  - integrator: sole patch applier to shared branch and gate owner.
- Lock rules:
  - lane must declare file-glob lock scope before edits.
  - no edits outside lane lock.
  - hotspot files require serialized ownership (`internal/tui/model.go`, `internal/app/service.go`, `internal/adapters/storage/sqlite/repo.go`).
- Approval/permission failure flow:
  - subagent command fails on permission gate.
  - orchestrator surfaces exact failure and approval request.
  - after user approval, orchestrator reruns blocked command or resumes lane.
- Completion policy:
  - no lane is marked complete until integrator verifies acceptance criteria and test evidence.
  - final wave closeout requires successful `just ci`.
  - for collaborative remediation waves, no lane/section may be marked complete until an independent QA subagent reviews both code changes and affected markdown trackers/worksheets and explicitly signs off.
  - no collaborative remediation implementation subagents may be launched until the user explicitly says to proceed (for example: "go ahead").
  - after QA sign-off and passing tests, pause for user-run confirmation; do not mark the collaborative section complete until the user confirms expected behavior in their run.

### Orchestrator Prompt Contract (Required)

- Every worker-lane prompt must include:
  - lane id and single acceptance objective,
  - lock scope (allowed file globs) and explicit out-of-scope paths,
  - concrete acceptance criteria mapped to the current phase/task,
  - architecture constraints for this repo (hexagonal boundaries, allowed dependency directions, and hotspot ownership),
  - testing plan (`just` commands only) and whether lane follows tests-first or a justified TDD exception,
  - explicit worker test scope: package-level `just test-pkg <pkg>` for touched packages (no repo-wide gate in worker lanes),
  - doc/comment expectations for touched Go declarations and non-obvious logic,
  - explicit Context7 checkpoints: before first code edit and after every failed test/runtime error, plus fallback behavior when unavailable,
  - expected handoff format and evidence requirements.
- Worker prompts must explicitly forbid:
  - edits outside lane lock,
  - direct `go test` execution,
  - running repo-wide test gates (`just test`, `just check`, `just ci`) unless the orchestrator explicitly assigns it,
  - architecture-layer violations unless explicitly authorized by the lane objective.

### Worker Handoff Contract (Required)

- Every worker handoff must include:
  - lane id and checkpoint id,
  - files changed and why,
  - commands run and pass/fail outcomes,
  - acceptance criteria checklist with pass/fail per item,
  - architecture-boundary compliance note,
  - doc/comment compliance note for touched Go code,
  - Context7 compliance note (initial consult + any failure-triggered re-consults),
  - unresolved risks/blockers and recommended next step.

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
- During collaborative validation waves, enforce section-by-section progression:
  - do not advance to the next worksheet section until each current-section bug is logged, fixed, verified (package tests + section rerun), and revalidated.
- For each section remediation:
  - run subagent code/context investigation first,
  - run Context7 before edits and again after any failed test/runtime error before the next edit,
  - propose fixes and confirm consensus with user before implementation,
  - run `just test-pkg <pkg>` for each touched package,
  - rerun that section’s manual/transport checks and update worksheet evidence immediately.

## UX Guardrails

- Help bar stays bottom-anchored in normal mode.
- Expanded help is a centered modal overlay (Fang-inspired style).
- Add/edit/info/project/search overlays are centered and do not push board content.
- Support both vim keys and arrow keys.
- Mouse wheel/click behavior must continue to function.
- Keep modal copy concise and avoid redundant field explanations.

## Release and Security

- Keep release/Homebrew work in roadmap unless explicitly requested for execution.
- Keep secrets out of config files committed to the repository.
- Prefer environment overrides for machine-local sensitive settings.
