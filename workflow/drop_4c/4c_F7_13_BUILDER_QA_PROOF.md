# Drop 4c F.7-CORE F.7.13 — Builder QA Proof Review (Round 1)

## Verdict

PASS.

Every item in the verification checklist is supported by direct evidence in
the claimed surface. No spec deviation found. No unclaimed Go code mutated by
this droplet (the modifications visible on `bundle.go` / `bundle_test.go` /
`spawn.go` / `spawn_test.go` are sibling F.7.17.6 work and are explicitly
excluded from attribution per the spawn prompt).

## 1. Findings

- 1.1 **CommitGateRunner shape — 4 required fields, 5 sentinels — confirmed.**
  `internal/app/dispatcher/gate_commit.go` declares `CommitGateRunner` with
  exactly four fields (`CommitAgent *CommitAgent`, `GitAdd GitAddFunc`,
  `GitCommit GitCommitFunc`, `GitRevParseHead GitRevParseFunc`) at
  lines 175 / 180 / 185 / 191. The five sentinels declared as package-level
  vars: `ErrCommitGateDisabled` (line 60), `ErrCommitGateNoPaths` (line 73),
  `ErrCommitGateAddFailed` (line 85), `ErrCommitGateCommitFailed` (line 96),
  `ErrCommitGateRevParseFailed` (line 105). Doc-comments name each
  sentinel's wrap policy (sentinel + underlying via `fmt.Errorf("%w: %w")`)
  so callers can route both axes via `errors.Is`.
- 1.2 **Run algorithm — 7 steps match spec verbatim.** The function body
  (gate_commit.go lines 226–302) executes the 7-step pipeline in order:
  (1) toggle gate via `project.Metadata.IsDispatcherCommitEnabled()`
  (line 242) — returns `nil` (NOT `ErrCommitGateDisabled`) on the
  disabled path per the F.7.15 default-OFF "no-op success" contract;
  (2) paths guard `len(item.Paths) == 0` → `ErrCommitGateNoPaths` (line 249);
  (3) F.7.12 `r.CommitAgent.GenerateMessage(ctx, *item, project, catalog, auth)`
  (line 260) — value-type item argument matches the F.7.12 receiver
  signature in `commit_agent.go` line 226;
  (4) `r.GitAdd(ctx, project.RepoPrimaryWorktree, item.Paths)` (line 270) —
  Paths slice forwarded verbatim, no `-A` / `.` / glob in sight;
  (5) `r.GitCommit(ctx, project.RepoPrimaryWorktree, message)` (line 279) —
  message taken verbatim from F.7.12, no automatic prefix / Signed-off-by;
  (6) `r.GitRevParseHead(ctx, project.RepoPrimaryWorktree)` (line 290),
  empty-string return ALSO trips `ErrCommitGateRevParseFailed` (line 295)
  to prevent silent EndCommit poisoning;
  (7) `item.EndCommit = newHash` (line 300), observable to caller via the
  `*domain.ActionItem` pointer.
- 1.3 **Toggle helper polarity matches.**
  `internal/domain/project.go` lines 173–187 declare
  `IsDispatcherCommitEnabled() bool` returning `false` when the
  `*bool` field is `nil` AND `false` when it is `*false`, returning
  `true` only on `*true`. The gate's branch `if !project.Metadata.IsDispatcherCommitEnabled() { return nil }`
  collapses the nil and `*false` cases identically, matching the
  three-state pointer-bool reservation documented at lines 119–149.
- 1.4 **8 spec scenarios + 6 robustness edges — all mapped to dedicated
  tests.** `gate_commit_test.go` declares 14 test functions:
  (1) `TestCommitGateRunHappyPath` — happy-path scenario, asserts
  `EndCommit` set, all 3 git shims fired exactly once, message + repoPath
  + paths forwarded verbatim;
  (2) `TestCommitGateRunToggleOff` — nil pointer toggle, no-op success;
  (3) `TestCommitGateRunToggleExplicitFalse` — `*false` collapses to
  same no-op as nil;
  (4) `TestCommitGateRunEmptyPaths` (2-row table: nil + zero-length) →
  `ErrCommitGateNoPaths` via `errors.Is`, no git shim invoked;
  (5) `TestCommitGateRunCommitAgentFails` — `errors.Is(err, agentErr)`
  on the wrapped underlying;
  (6) `TestCommitGateRunGitAddFails` — both `errors.Is(err, ErrCommitGateAddFailed)`
  AND `errors.Is(err, addErr)` reachable; commit + revParse NOT fired;
  EndCommit unchanged;
  (7) `TestCommitGateRunGitCommitFails` — both axes reachable, revParse
  NOT fired;
  (8) `TestCommitGateRunGitRevParseFails` — both axes reachable;
  (9) `TestCommitGateRunGitRevParseEmpty` — empty hash trips
  `ErrCommitGateRevParseFailed`;
  (10) `TestCommitGateRunEndCommitSetCorrectly` — overwrites pre-populated
  `EndCommit`, the spec's separately-listed acceptance line;
  (11) `TestCommitGateRunNilReceiver` — defensive guard;
  (12) `TestCommitGateRunNilItem` — defensive guard;
  (13) `TestCommitGateRunNilCommitAgent` — defensive guard, no git shim
  invoked;
  (14) `TestGateKindCommitRegistered` — cross-checks
  `templates.IsValidGateKind(GateKindCommit) == true` and the canonical
  string `"commit"`.
  Spec-scenario coverage is complete; the 6 robustness edges (toggle-
  explicit-false, rev-parse-empty, three nil-defensive, enum cross-check)
  add asymmetric depth on every algorithm branch the 8-row spec leaves
  ambiguous.
- 1.5 **`GateKindCommit` added to the closed templates enum.**
  `internal/templates/schema.go` line 94 declares
  `GateKindCommit GateKind = "commit"` with a doc-comment naming
  F.7.13 + the F.7.16 default-template-extension follow-up; the
  `validGateKinds` slice at lines 100–105 includes `GateKindCommit`
  alongside the three Drop-4b-Wave-A members. `schema_test.go`'s
  `TestGateKindClosedEnum` (per `git diff`) moves `GateKind("commit")`
  from `invalidCases` to `validCases` (now first in the valid list with
  `GateKindMageCI` etc.), keeps `GateKind("push")` in `invalidCases`
  with a comment marking the F.7.13 scope boundary, and updates the
  test's doc-comment to reflect the F.7.13 vocabulary. Result: the
  closed-enum gate registry now accepts `"commit"` at template load
  time. The complementary cross-check inside `dispatcher` package
  (`TestGateKindCommitRegistered`) closes the loop by asserting
  `IsValidGateKind` from a consumer package.
- 1.6 **No commit per REV-13.** `git status --short` of the action-item
  scope shows only modifications and new files in the working tree — no
  HEAD advance attributable to F.7.13. The five claimed paths
  (`gate_commit.go`, `gate_commit_test.go`, `schema.go`, `schema_test.go`,
  `4c_F7_13_BUILDER_WORKLOG.md`) are all in `?? ` / `M ` state, none
  committed. The four other paths showing as modified
  (`internal/app/dispatcher/bundle.go`, `bundle_test.go`, `spawn.go`,
  `spawn_test.go`) are the F.7.17.6 sibling parallel droplet's surface
  per the spawn prompt's explicit "do NOT attribute" note; they are
  excluded from this review.
- 1.7 **`mage ci` green claim accepted.** Worklog asserts 2672 tests
  pass / 1 skip (pre-existing
  `TestStewardIntegrationDropOrchSupersedeRejected`), dispatcher coverage
  75.4 %, templates 97.0 %, format-check + build green. The QA-Proof
  discipline accepts the builder's `mage ci` verdict — re-running mage
  is the falsification sibling's domain. The dispatcher-coverage delta
  (75.5 → 75.4) is attributed to the four nil-defensive guards each
  carrying one provably-unreachable branch and is consistent with the
  added file's branch density.
- 1.8 **`git add` strictly path-scoped — doc-comment carries the literal
  contract.** `GitAddFunc` doc-comment at gate_commit.go lines 107–119
  states `Implementations MUST treat the paths slice verbatim — no -A,
  no ., no glob expansion.` (line 112) — matches the verification
  point's literal quote requirement verbatim. The `Run` body at line 270
  forwards `item.Paths` to the injected shim with no manipulation, no
  flag prepend, no expansion. Production-wiring follow-up (the `--`
  separator + `os/exec` adapter) is correctly deferred to the next
  droplet per the worklog's "Wiring follow-up" section, which is
  exactly the same shape F.7.12 used.

## 2. Missing Evidence

- 2.1 **None.** Every claim in the verification checklist has direct file
  evidence. Hylla is mid-enrichment for `tillsyn@main` so committed-code
  cross-references resolved via direct `Read` of `internal/domain/project.go`
  and `internal/app/dispatcher/commit_agent.go` instead — the substitution
  is acceptable because both files are in-tree on the working branch and
  the symbols I cited are present at the line numbers given. Hylla
  unavailability does not gap this review.

## 3. Summary

PASS. F.7.13 ships the `CommitGateRunner` API surface (4 fields), the 5
sentinels with errors.Is wrap policy, the 7-step Run algorithm matching
the spec verbatim, the path-scoped `git add` contract enforced by
doc-comment + verbatim Paths forwarding, the closed-enum `GateKindCommit`
addition with both producer-side (`schema_test.go` table move) and
consumer-side (`TestGateKindCommitRegistered`) tests, and the toggle-
gated default-OFF semantics matching `IsDispatcherCommitEnabled`'s
nil-or-false collapse. All 8 spec scenarios + 6 robustness edges have
dedicated test functions. No commit was made (REV-13 honored). The
sibling F.7.17.6 modifications visible on `bundle.go` / `spawn.go` are
out of scope per the spawn-prompt attribution rule. Production wiring
(gateFunc adapter + os/exec adapters + default-template extension) is
correctly deferred to subsequent droplets — same pattern F.7.12 used.

## TL;DR

- T1: F.7.13 PASSES proof review — all 8 spec scenarios + 6 robustness
  edges covered, all 5 sentinels and 4 fields present, `git add`
  contract carries the literal verbatim doc-string, GateKindCommit
  added to closed enum on both producer and consumer sides, no commit
  attributable to this droplet.
- T2: No missing evidence; Hylla unavailability did not gap the review.
- T3: PASS.
