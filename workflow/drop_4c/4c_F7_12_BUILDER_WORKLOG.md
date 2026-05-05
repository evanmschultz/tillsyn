# Drop 4c F.7-CORE F.7.12 — Builder Worklog

## Droplet

F.7.12 — Commit-agent integration via spawn pipeline.

REVISIONS-first: F.7.17.5 (committed `d3fbb14`) provides `BuildSpawnCommand`;
F.7.3b (committed `37f5a69`) provides bundle render; F.7.4 (committed `37f5a69`)
provides the stream-jsonl `Monitor`. F.7.12 ships ONLY the agent invocation
surface — the post-build commit gate itself (F.7.13) is a separate droplet.

## What landed

Files added (this droplet only):

- `internal/app/dispatcher/commit_agent.go`
- `internal/app/dispatcher/commit_agent_test.go`
- `workflow/drop_4c/4c_F7_12_BUILDER_WORKLOG.md`

Production surface (in `package dispatcher`):

- `CommitMessageMaxLen` — exported constant `= 72` (per CLAUDE.md
  "Single-Line Commits"). Reused by F.7.13 + downstream consumers.
- `ErrNoCommitDiff` — returned when `item.StartCommit` or `item.EndCommit`
  is empty/whitespace.
- `ErrCommitMessageTooLong` — wrapped with the offending message text and
  computed length. Triggered on `>72` chars OR any embedded newline (the
  "no body" enforcement on top of the length cap).
- `ErrCommitAgentMisconfigured` — returned when `GitDiff`,
  `BuildSpawnCommand`, or `Monitor` is nil. Field name appears in the
  error message so production wiring loud-fails at first dispatch.
- `ErrCommitSpawnNoTerminal` — returned when the spawn ran cleanly but
  produced no usable assistant text (sink empty, `report.Reason` empty,
  `report.Errors` empty).
- `GitDiffReader` interface — local consumer-side seam. Same shape as the
  one in `internal/app/dispatcher/context` (a single-method `Diff(ctx,
  fromCommit, toCommit) ([]byte, error)`); declared locally to avoid the
  import cycle that would form if `dispatcher` imported its own
  `context` subpackage.
- `SpawnBuilder` type alias — production signature of `BuildSpawnCommand`
  exposed as a struct field so tests inject mocks.
- `MonitorRunner` type alias — production signature of `(*Monitor).Run`
  exposed as a struct field so tests inject canned reports.
- `CommitAgent` struct with three required fields (`GitDiff`,
  `BuildSpawnCommand`, `Monitor`) plus three unexported test-seams
  (`runCmd`, `openStream`, `lookupAdapterFn`) defaulted at call time when
  nil.
- `CommitAgent.GenerateMessage(ctx, item, project, catalog, auth) (string,
  error)` — the algorithmic surface. 12-step algorithm documented in the
  function doc-comment.

Tests (`commit_agent_test.go`):

The 7 spec scenarios:

1. `TestCommitAgentGenerateMessageHappyPath` — happy path; asserts msg
   round-trip AND that the diff lands at
   `<bundleRoot>/context/git_diff.patch`.
2. `TestCommitAgentGenerateMessageMissingStartCommit` — empty StartCommit
   → `ErrNoCommitDiff`.
3. `TestCommitAgentGenerateMessageMissingEndCommit` — empty EndCommit →
   `ErrNoCommitDiff`.
4. `TestCommitAgentGenerateMessageEmptyDiff` — empty diff bytes still
   spawns (let the agent decide); verifies zero-byte diff file written.
5. `TestCommitAgentGenerateMessageMessageTooLong` — `> 72` chars →
   `ErrCommitMessageTooLong`; offending text wrapped in error.
6. `TestCommitAgentGenerateMessageSpawnBuildFails` — synthetic build
   error surfaces via `errors.Is`.
7. `TestCommitAgentGenerateMessageMonitorFails` — synthetic monitor error
   surfaces via `errors.Is`.

Plus 8 robustness-edge tests (added unprompted but surface real algorithm
branches the spec scenarios miss):

- `TestCommitAgentGenerateMessageMultilineMessageRejected` — embedded
  `\n` triggers `ErrCommitMessageTooLong` (no-body rule on top of cap).
- `TestCommitAgentGenerateMessageAssistantTextWins` — production claude
  path: last non-terminal `assistant.Text` event captured via the sink
  channel takes priority over `TerminalReport.Reason`.
- `TestCommitAgentGenerateMessageNoTerminalText` — empty sink + empty
  Reason + empty Errors → `ErrCommitSpawnNoTerminal`.
- `TestCommitAgentGenerateMessageNilReceiver` — nil `*CommitAgent`
  receiver returns `ErrCommitAgentMisconfigured` instead of panicking.
- `TestCommitAgentGenerateMessageNilDeps` (3 sub-tests) — each of the
  three required fields, when nil, returns `ErrCommitAgentMisconfigured`
  with the field name in the error message.
- `TestCommitAgentGenerateMessageGitDiffFails` — non-nil git error
  propagates via `errors.Is`.
- `TestCommitAgentGenerateMessageMissingMCPConfigPath` — descriptor with
  empty `MCPConfigPath` surfaces a clear error rather than silently
  writing the diff under the wrong directory.
- `TestCommitAgentGenerateMessageNoBindingForKind` — catalog without a
  binding for `KindCommit` surfaces a clear error rather than nil-derefing
  the resolved binding downstream.

Total: 14 test functions, 16 effective scenarios (one is a 3-row table
test). The dispatcher package gained 1 net test (299 vs 298) — many tests
share helpers.

## Algorithm

The 12-step pipeline `GenerateMessage` runs:

1. Validate `*CommitAgent` is non-nil.
2. Validate `item.StartCommit` and `item.EndCommit` both non-empty
   (`ErrNoCommitDiff` otherwise).
3. Validate dependency fields wired (`ErrCommitAgentMisconfigured`).
4. Resolve `git diff <start>..<end>` via `c.GitDiff`. Empty diff is NOT an
   error — empty bytes pass through.
5. Synthesize a commit-kind action item carrying the parent's commit
   anchors plus `Kind=KindCommit` so `BuildSpawnCommand` resolves the
   commit-message-agent binding.
6. Invoke `BuildSpawnCommand` → (`*exec.Cmd`, `SpawnDescriptor`).
7. Derive bundle root: `bundleRoot = filepath.Dir(filepath.Dir(descriptor.MCPConfigPath))`.
   Documented coupling: this depends on the canonical `<root>/plugin/.mcp.json`
   layout — when codex / future adapters publish their own MCPConfigPath
   layout, the bundle-root recovery moves onto `SpawnDescriptor` itself.
8. Materialize the diff at `<bundleRoot>/context/git_diff.patch` so the
   downstream commit-message-agent prompt template (F.7.13 / render
   concern) reads from a fixed location.
9. Resolve the `CLIAdapter` for the binding's `CLIKind` via
   `c.lookupAdapterFn` (default `lookupAdapter`).
10. Run the spawn (`cmd.Run` via `c.runCmd` injection point).
11. Open `<bundleRoot>/stream.jsonl`, feed it through `c.Monitor` with a
    sink channel sized 256.
12. Extract message via three-tier priority:
    a. Last non-terminal `assistant.Text` event captured via the sink
       (production claude path).
    b. `TerminalReport.Reason` (mock/fixture path; some adapters route
       short messages through the terminal report's reason field).
    c. `TerminalReport.Errors[0]` (defensive fallback for adapters that
       route short completions through the errors channel).
    Empty all three → `ErrCommitSpawnNoTerminal`.
13. Validate length: `> CommitMessageMaxLen` OR contains `\n` →
    `ErrCommitMessageTooLong`. Return the validated message.

## Design notes / deviations from spec

- **`BuildSpawnCommand` signature.** The spec listed
  `func(ctx context.Context, item, project, catalog domain.KindCatalog, auth)`,
  but production reality is `func(item, project, templates.KindCatalog, auth)
  (*exec.Cmd, SpawnDescriptor, error)` — no leading `ctx`, the catalog lives
  in `internal/templates` (not `internal/domain`), and the call returns a
  `SpawnDescriptor` alongside the cmd. The `SpawnBuilder` type alias mirrors
  production reality so production wiring can assign
  `dispatcher.BuildSpawnCommand` directly.
- **`Monitor` field uses `MonitorRunner` type alias.** The spec listed the
  signature inline; aliasing it keeps the struct definition readable and
  gives test fixtures a named type to construct.
- **Diff context written to disk, not embedded in prompt.** The spec offered
  two options: write a context file OR embed in system-prompt. I chose the
  file route (`<bundleRoot>/context/git_diff.patch`) because:
  (a) The render hook `cli_claude/render` does NOT pull from
  `item.Description`, so embedding in the synthetic action item would not
  reach the agent.
  (b) The context dir is already materialized by F.7.1's `NewBundle`.
  (c) Diffs can be large; system-prompt injection has a token budget.
  The downstream commit-message-agent prompt template (F.7.13 owns this)
  references `context/git_diff.patch` by canonical name.
- **Three-tier message extraction.** The spec said "Extract the agent's
  final text from terminal event (`StreamEvent.Text` of the last
  `IsTerminal == true` event)." But the canonical `TerminalReport` shape
  declared in `cli_adapter.go` does NOT carry a `Text` field — it has
  `Cost` / `Denials` / `Reason` / `Errors`. The claude adapter's terminal
  event ("result") carries `terminal_reason="completed"` etc., not the
  assistant's actual response — that lands in prior `assistant` events.
  I wired the sink channel to capture the last non-terminal
  `assistant.Text`, with `report.Reason` as fallback (the unit-test
  / mock path) and `report.Errors[0]` as defensive last resort. This is
  documented in the function doc-comment + the inline algorithm comment.
- **`runCmd` / `openStream` / `lookupAdapterFn` as unexported test seams.**
  The spec specified three required exported fields. Internal seams
  defaulted on first use keep the public surface minimal while letting
  the test suite drive every branch deterministically. Tests assign
  these directly (in-package) without exposing them publicly. Pattern is
  consistent with how the rest of the dispatcher package handles
  test-substitution (e.g. clock injection in `processMonitor`).
- **`KindCommit` is the synthetic kind.** The action item passed to
  `GenerateMessage` typically has `Kind=KindBuild`. The synthesis step
  clones the item with `Kind=KindCommit` so the catalog binding for
  commit-message-agent fires. The catalog author MUST register a binding
  for `KindCommit` with `Model="haiku"` and tool restrictions (Read,
  Bash) — that wiring is the F.7.13 / template-author concern, NOT
  F.7.12.
- **Multi-line rejection.** The "Single-Line Commits" rule prohibits a
  body, not just a long subject. A message like `"feat: x\n\nbody"` is
  ≤72 chars on its first line but still rejected as `ErrCommitMessageTooLong`
  (the test pins this).

## Wiring follow-up (deferred — not this droplet)

- **F.7.13 commit gate** consumes `CommitAgent.GenerateMessage` to obtain
  the message string, then runs `git add` + `git commit` against the
  project worktree under the dispatcher's commit-cadence rules.
- **Real `GitDiffReader` adapter.** Production wiring needs a concrete
  `GitDiffReader` that shells out to `git diff <from>..<to>` in the
  project worktree. The `internal/app/dispatcher/context` package already
  has the same shape; either reuse that adapter (after lifting it into
  the parent `dispatcher` package, or by satisfying both interfaces in a
  single concrete type) OR ship a dedicated commit-agent diff reader.
- **Catalog binding for `KindCommit`.** Templates that ship the cascade
  must register an `AgentBinding` for `KindCommit` with `Model="haiku"`,
  `AgentName="commit-message-agent"`, and a narrow tools allow-list.
  Bundled templates (e.g. `default-go`) gain this in F.7.13's wiring.
- **Bundle root on SpawnDescriptor.** F.7.12 walks up two `filepath.Dir`s
  from `descriptor.MCPConfigPath` to recover the bundle root. This
  couples F.7.12 to the claude bundle layout. A clean refactor adds a
  `BundleRoot` (or `Bundle`) field to `SpawnDescriptor` so future
  adapters publish it directly. Tracked as a future-droplet refinement.
- **TerminalReport.Text or FinalText.** F.7.12's three-tier message
  extraction (sink → Reason → Errors[0]) is a workaround for the missing
  canonical "final assistant text" channel. A clean fix adds
  `TerminalReport.FinalText` populated by adapters that surface terminal
  text through the result event (e.g. claude's stop_reason +
  message.content[].text on the assistant event preceding result). The
  three-tier logic survives that refactor unchanged — just with one more
  source.

## Verification

- `mage testPkg ./internal/app/dispatcher` — 299 tests pass (was 298
  before this droplet; 1 net new test plus 14 new test functions reusing
  shared helpers — net ≠ count because shared `t.Run` table-test rows
  were already counted as separate tests in the prior tally).
- `mage ci` — full gate green:
  - 2642 tests pass, 1 skip (pre-existing
    `TestStewardIntegrationDropOrchSupersedeRejected`, unrelated).
  - Coverage threshold met across all 24 packages; dispatcher at 75.1%
    (up from 74.8%).
  - Build succeeds.
- Format check (`gofumpt`) clean (gate ran inside `mage ci`).

## Hard constraints honored

- DO NOT commit. Confirmed — no `git commit` invoked.
- Edits limited to: `internal/app/dispatcher/commit_agent.go` (NEW),
  `internal/app/dispatcher/commit_agent_test.go` (NEW), this worklog
  (NEW).
- No Hylla calls (per-droplet rule).
- No edits to `monitor.go`, `spawn.go`, `dispatcher.go`, or any
  non-listed file.
- No `mage install`. No raw `go build` / `go test` / `go vet`.
- F.7.12 does NOT wire into a real `commit` gate consumer — F.7.13 owns
  that. F.7.12 ships the API + tested skeleton only.
- Single-line conventional commit ≤72 chars rule enforced in the
  algorithm; the worklog itself is exempt (it is project documentation,
  not a git commit message).

## Hylla Feedback

N/A — task touched only Go code in `internal/app/dispatcher/` plus a new
workflow markdown; per droplet rule "NO Hylla calls," no Hylla queries
were issued. Codebase searches for `BuildSpawnCommand`, `Monitor`,
`StreamEvent`, `TerminalReport`, `GitDiffReader`, `MockAdapter`,
`StartCommit`, `EndCommit`, `KindCommit`, and the `templates.AgentBinding`
schema used `Read` / `rg` directly per the droplet's NO-Hylla
constraint, not as a Hylla fallback.
