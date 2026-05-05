# Drop 4c F.7-CORE F.7.13 — Builder Worklog

## Droplet

F.7.13 — Commit gate implementation. Consumes F.7.12's `CommitAgent.GenerateMessage`
to obtain a single-line conventional-commit message, then runs path-scoped
`git add` + `git commit` against the project worktree, and writes the new
HEAD hash from `git rev-parse` into `action_item.EndCommit`. Gated behind
F.7.15's `dispatcher_commit_enabled` project metadata toggle (default OFF).

## What landed

Files added (this droplet only):

- `internal/app/dispatcher/gate_commit.go` (NEW)
- `internal/app/dispatcher/gate_commit_test.go` (NEW)
- `workflow/drop_4c/4c_F7_13_BUILDER_WORKLOG.md` (NEW, this file)

Files edited:

- `internal/templates/schema.go` — extended the closed `GateKind` enum with
  `GateKindCommit GateKind = "commit"` and appended it to
  `validGateKinds`. The file's GateKind doc-comment was also updated to
  reflect that "commit" has landed (Drop 4c F.7.13) and "push" is the
  remaining future-droplet value.
- `internal/templates/schema_test.go` — moved `GateKind("commit")` from
  `invalidCases` to `validCases` and added a comment marking it as Drop 4c
  F.7.13. The "push" assertion stays in `invalidCases` until that droplet
  ships.

Production surface (in `package dispatcher`):

- `CommitGateRunner` struct with four required fields (`CommitAgent`,
  `GitAdd`, `GitCommit`, `GitRevParseHead`).
- `CommitGateRunner.Run(ctx, *item, project, catalog, auth) error` — the
  algorithmic surface. 7-step pipeline documented in the function
  doc-comment.
- `GitAddFunc`, `GitCommitFunc`, `GitRevParseFunc` — three injected test
  seams documented as production-shells-os/exec / tests-inject-stubs.
- `ErrCommitGateDisabled` — exported sentinel; reserved for future call
  sites that want to distinguish disabled-by-toggle. Run itself does NOT
  return this sentinel — toggle-off is a successful no-op (returns nil).
- `ErrCommitGateNoPaths` — empty `item.Paths` (nil OR zero-length) is a
  hard failure rather than a silent no-op so a misconfigured planner
  surface stays visible.
- `ErrCommitGateAddFailed`, `ErrCommitGateCommitFailed`,
  `ErrCommitGateRevParseFailed` — three sentinels wrapping the underlying
  git-shim errors. Both axes (sentinel + cause) reachable via
  `errors.Is`.

Production surface (in `package templates`):

- `GateKindCommit GateKind = "commit"` — the closed-enum vocabulary entry
  for the F.7.13 commit gate. Required so a template authoring
  `[gates.build] = ["mage_ci", "commit"]` passes the load-time
  `validateGateKinds` check.

Tests (`gate_commit_test.go`, 14 test functions / 16 effective scenarios
counting the table-test sub-rows):

The 8 documented spec scenarios:

1. `TestCommitGateRunHappyPath` — toggle on, paths populated, all deps
   succeed → `item.EndCommit` set, all three git shims fire, Run returns
   nil, message + repoPath + paths verbatim.
2. `TestCommitGateRunToggleOff` — `IsDispatcherCommitEnabled()` == false
   (nil pointer, the default) → no-op, no git shim invoked, EndCommit
   unchanged, Run returns nil. CommitAgent intentionally has zero-value
   fields to prove the toggle short-circuits BEFORE touching it.
3. `TestCommitGateRunEmptyPaths` (2-row table) — nil OR zero-length
   `item.Paths` → `ErrCommitGateNoPaths`; no git shim invoked.
4. `TestCommitGateRunCommitAgentFails` — `GenerateMessage` returns synthetic
   error → wrapped error (errors.Is reachable); no git shim invoked.
5. `TestCommitGateRunGitAddFails` — `GitAdd` error wrapped with
   `ErrCommitGateAddFailed`; underlying error also reachable via
   errors.Is; commit + revParse NOT invoked; EndCommit unchanged.
6. `TestCommitGateRunGitCommitFails` — `GitCommit` error wrapped with
   `ErrCommitGateCommitFailed`; revParse NOT invoked; EndCommit unchanged.
7. `TestCommitGateRunGitRevParseFails` — `GitRevParseHead` error wrapped
   with `ErrCommitGateRevParseFailed`; EndCommit unchanged.
8. `TestCommitGateRunEndCommitSetCorrectly` — explicit assertion that the
   rev-parse return value flows verbatim into `item.EndCommit`, even when
   the action item's prior `EndCommit` was non-empty (gets overwritten).

Plus 6 robustness-edge tests (added because the algorithm surface has more
branches than the 8-scenario spec covers):

- `TestCommitGateRunToggleExplicitFalse` — `*false` (vs nil) is treated
  identically to nil; the three-state pointer-bool reserves the shape but
  collapses both forms to "disabled" today.
- `TestCommitGateRunGitRevParseEmpty` — empty (whitespace-stripped) hash
  from `GitRevParseHead` is treated as a rev-parse failure; downstream
  gates use non-empty EndCommit as the "commit happened" signal and an
  empty value would silently poison that read.
- `TestCommitGateRunNilReceiver` — nil `*CommitGateRunner` returns a loud
  error instead of nil-derefing.
- `TestCommitGateRunNilItem` — nil `*domain.ActionItem` returns a loud
  error.
- `TestCommitGateRunNilCommitAgent` — nil `CommitAgent` field surfaces a
  loud error AT execution time (after toggle / paths guards), not at
  construction. No git shim invoked.
- `TestGateKindCommitRegistered` — cross-checks
  `templates.IsValidGateKind(GateKindCommit) == true`. Belt-and-suspenders
  against accidental enum churn that would otherwise let a template
  author bind GateKindCommit only for the gate to silently no-op at
  template-load time.

## Algorithm

The 7-step pipeline `Run` executes:

1. Toggle gate. `project.Metadata.IsDispatcherCommitEnabled()` returns
   false (default, or explicit `*false`) → return nil. No-op success.
2. Paths guard. `len(item.Paths) == 0` → `ErrCommitGateNoPaths`.
3. Generate message via `CommitAgent.GenerateMessage(ctx, *item, project,
   catalog, auth)`. F.7.12 owns its own validation; failures propagate
   wrapped with a "commit gate: " prefix so the gate-name shows up in the
   error chain.
4. `GitAdd(ctx, project.RepoPrimaryWorktree, item.Paths)`. On error, wrap
   with `ErrCommitGateAddFailed`.
5. `GitCommit(ctx, project.RepoPrimaryWorktree, message)`. On error, wrap
   with `ErrCommitGateCommitFailed`.
6. `GitRevParseHead(ctx, project.RepoPrimaryWorktree)`. On error, wrap
   with `ErrCommitGateRevParseFailed`. Empty hash also treated as
   rev-parse failure.
7. Mutate `item.EndCommit = newHash`. Return nil.

Plus three guards before step 1: nil `*CommitGateRunner` receiver, nil
`*domain.ActionItem` argument, nil `CommitAgent` / `GitAdd` / `GitCommit`
/ `GitRevParseHead` field. Each surfaces a loud error rather than nil-
derefing — defense-in-depth, not a documented contract.

## Design notes / deviations from spec

- **`KindCatalog` is in `templates`, not `domain`.** The spec body wrote
  `domain.KindCatalog`; production reality is `templates.KindCatalog`.
  F.7.12's `CommitAgent.GenerateMessage` already takes
  `templates.KindCatalog`, so the gate's signature matches that for
  consistency. No spec-divergent behavior — only the import path differs
  from the prose.
- **`mage check` is not a real target.** The spec hard constraints listed
  "`mage check` + `mage ci`. NEVER `mage install`." Only `mage ci` exists
  in the magefile (other CI-equivalent targets: `mage formatCheck`,
  `mage testPkg`, `mage testFunc`). Verification ran `mage ci` only.
- **`ErrCommitGateDisabled` is reserved, not returned.** The spec listed
  this sentinel alongside the four "wrap underlying" sentinels. The
  algorithm itself returns nil on the toggle-off path (per spec algorithm
  step 1 "gate disabled = noop, not an error"), so `ErrCommitGateDisabled`
  is an exported label for future call sites that want to distinguish
  "disabled-by-toggle" without re-checking the project metadata. No
  current path produces it; the doc-comment marks it explicitly.
- **`git add -- <paths>` separator.** The gate passes `item.Paths`
  verbatim to the injected `GitAdd` shim. Production wiring must use the
  `--` separator to defend against path-as-flag injection (`-h`, `--help`
  in a path); enforcement lives in the adapter, not the gate. The gate's
  doc-comment names this contract for the production wiring follow-up
  droplet.
- **Empty-hash rev-parse treated as failure.** Spec said "git rev-parse
  fails: GitRevParseHead error → wrapped". I extended this: an empty
  string return ALSO triggers `ErrCommitGateRevParseFailed`. Downstream
  gates use non-empty `EndCommit` as the "commit happened" signal —
  silently writing an empty value would poison that read.
- **Closed-enum extension lands in templates package.** The spec mentioned
  registering `commit` in the dispatcher's gate-kind enum and
  "look up the existing pattern." The closed-enum vocabulary lives in
  `internal/templates/schema.go` (`GateKind` + `validGateKinds`); the
  dispatcher's `gates.go` registry is just a private `map[GateKind]gateFunc`
  populated via private `Register` calls. The gateFunc adapter that
  bridges `CommitGateRunner.Run` (catalog/auth/error/EndCommit-mutation)
  to `gateFunc` (`(ctx, item, project) GateResult`) is the natural
  responsibility of the Drop 4c follow-up wiring droplet — same way
  F.7.12's production wiring deferred to F.7.13. F.7.13 ships the API +
  tested skeleton + closed-enum entry; the gateFunc adapter is the next
  droplet's concern.
- **Three-state pointer-bool collapse.** The toggle's pointer-bool design
  reserves three states (nil / false / true) but collapses nil and false
  to "disabled" today. The `TestCommitGateRunToggleOff` and
  `TestCommitGateRunToggleExplicitFalse` tests pin both forms.
- **Zero-length AND nil paths handled.** `len(nil) == 0` and
  `len([]string{}) == 0` are both 0; a single check covers both. The
  table-test in `TestCommitGateRunEmptyPaths` pins both rows explicitly
  rather than relying on the implicit equivalence.
- **Idempotency disclaimer.** The doc-comment names that Run is NOT
  idempotent — a second Run on the same item spawns the message agent
  again, runs `git add` (idempotent on unchanged files), and fails at
  `git commit` with "nothing to commit" wrapped as
  `ErrCommitGateCommitFailed`. Callers must not retry on success.

## Wiring follow-up (deferred — not this droplet)

- **`gateFunc` adapter for `commit` kind.** Production wiring needs an
  adapter shim that translates `gateFunc(ctx, item, project) GateResult`
  to `CommitGateRunner.Run(ctx, *item, project, catalog, auth) error`.
  The shim closes over a constructed `CommitGateRunner` instance + the
  current dispatcher catalog/auth surface, lifts the action-item value
  to a pointer (mutation-aware), translates Run's error into a
  `GateStatusFailed` result with the underlying error wrapped. Lands
  alongside the F.7.16 default-template `[gates.build]` extension.
- **Real `GitAddFunc` / `GitCommitFunc` / `GitRevParseFunc` adapters.**
  Production wiring needs three concrete os/exec-backed implementations
  of the test seams. Each shells `git <subcommand>` with `cmd.Dir =
  project.RepoPrimaryWorktree`, captures stdout/stderr for logging, and
  returns a wrapped exec error on non-zero exit. The `git add` adapter
  MUST use the `--` separator to reject path-as-flag injection.
- **`F.7.16` default-template `[gates.build]` extension.** Master PLAN.md
  L20 + F.7-CORE plan L223 call for `[gates.build] = ["mage_ci", "commit",
  "push"]` once both gates land. F.7.13 enables `commit` in the closed
  enum but does NOT modify `internal/templates/builtin/default.toml` —
  that's F.7.16's responsibility. Adding it here would silently activate
  the gate on every project that loads the default template, which is
  out of scope for this droplet AND would conflict with the off-by-default
  toggle semantics.
- **Permission-grant mediation.** The commit-message-agent runs with a
  narrow tools allow-list (Read + Bash for `git diff` inspection). The
  `git add` / `git commit` / `git rev-parse` shells inside the gate run
  in the dispatcher's process, NOT in the agent's process — they bypass
  the agent's permission mediation entirely. This is by design (the
  gate trusts its own injected functions) but the production wiring
  needs to pin that the dispatcher process has the authority to mutate
  the worktree.

## Verification

- `mage testPkg ./internal/app/dispatcher` — 323 tests pass (was 299
  before this droplet; +24 effective rows from 14 new test functions
  including the empty-paths 2-row table and the F.7.13 cross-check).
- `mage testPkg ./internal/templates` — 378 tests pass (no count change;
  one row moved from invalid-cases to valid-cases inside the existing
  TestGateKindClosedEnum table-test).
- `mage ci` — full gate green:
  - 2672 tests pass, 1 skip (pre-existing
    `TestStewardIntegrationDropOrchSupersedeRejected`, unrelated).
  - Coverage threshold met across all 24 packages; dispatcher at 75.4%
    (down from 75.5% — small dilution from the new file's algorithm
    branches that the test suite doesn't exhaust, e.g. the four
    nil-receiver / nil-field / nil-item / nil-CommitAgent guards each
    have one untested branch where the gate continues despite the
    nil — by construction they're unreachable). Templates at 97.0%.
  - Format check (`gofumpt`) clean (gate ran inside `mage ci`).
  - Build succeeds.

## Hard constraints honored

- DO NOT commit. Confirmed — no `git commit` invoked by me.
- Edits limited to: `internal/app/dispatcher/gate_commit.go` (NEW),
  `internal/app/dispatcher/gate_commit_test.go` (NEW),
  `internal/templates/schema.go` (CLOSED-ENUM EXTENSION),
  `internal/templates/schema_test.go` (TEST UPDATE FOR ENUM CHANGE),
  this worklog (NEW).
- No Hylla calls (per-droplet rule).
- `git add <paths>` is path-scoped — production wiring contract documented
  in `GitAddFunc` doc-comment ("Implementations MUST treat the paths
  slice verbatim — no `-A`, no `.`, no glob expansion").
- No `mage install`. No raw `go build` / `go test` / `go vet`.
- Single-line conventional commit ≤72 chars rule enforced via F.7.12
  (the gate consumes its already-validated message).
- F.7.13 does NOT wire into a real gate consumer in this droplet —
  the gateFunc adapter + production wiring is the next droplet. F.7.13
  ships the API + tested skeleton + closed-enum entry only.

## Hylla Feedback

N/A — task touched only Go code in `internal/app/dispatcher/` plus
`internal/templates/schema.go` + `schema_test.go`, plus a new workflow
markdown; per droplet rule "NO Hylla calls," no Hylla queries were
issued. Codebase searches for `IsDispatcherCommitEnabled`,
`CommitAgent`, `gateFunc`, `gateRunner`, `GateKind`, `validGateKinds`,
`templates.KindCatalog`, `domain.ActionItem` field shape, and
`fakeSpawnBuilder` / `commitCatalog` test helpers used `Read` / `rg`
directly per the droplet's NO-Hylla constraint, not as a Hylla
fallback.
