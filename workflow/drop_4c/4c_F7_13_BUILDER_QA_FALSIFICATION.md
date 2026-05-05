# F.7-CORE F.7.13 Commit Gate — Builder QA Falsification

Read-only adversarial review of the F.7.13 builder's claim:
"`CommitGateRunner` skeleton + `GateKindCommit` enum extension landed; gate
behavior tested across 14 test functions; F.7.16 wiring deferred."

Builder worklog: `workflow/drop_4c/4c_F7_13_BUILDER_WORKLOG.md`.
Files attacked: `internal/app/dispatcher/gate_commit.go` (NEW),
`internal/app/dispatcher/gate_commit_test.go` (NEW),
`internal/templates/schema.go` (closed-enum extension),
`internal/templates/schema_test.go` (test-update for enum change).

Parallel sibling F.7.17.6 (modifies `bundle.go`, `bundle_test.go`,
`spawn.go`, `spawn_test.go`) is NOT attributed to this droplet — those
diffs are out-of-scope.

## Section 0 — SEMI-FORMAL REASONING

### Proposal

**Premises.** The builder claims F.7.13 ships an API skeleton with toggle,
paths-guard, message-generation, three git seams, EndCommit mutation, and
a closed-enum vocabulary entry — all tested via injected stubs. Production
adapter wiring (gateFunc adapter, real os/exec impls) deferred to F.7.16.

**Evidence sources used.**
- `Read` of `internal/app/dispatcher/gate_commit.go` (full, 302 lines).
- `Read` of `internal/app/dispatcher/gate_commit_test.go` (full, 603 lines).
- `Read` of `internal/templates/schema.go` lines 40–125 (GateKind enum +
  validGateKinds + IsValidGateKind).
- `Read` of `internal/templates/schema_test.go` (full).
- `Read` of `internal/domain/project.go` lines 120–195
  (ProjectMetadata.IsDispatcherCommitEnabled, value receiver semantics).
- `git diff HEAD -- internal/templates/` for the closed-enum extension
  diff (clean, exactly the documented changes).
- `git status --porcelain` for untracked/modified inventory (worklog
  scope honored — F.7.17.6 sibling files NOT attributed).
- `rg` searches across the dispatcher package for `gateRegistry`,
  `gateFunc`, `Register.*[Gg]ate`, `IsDispatcherCommitEnabled`,
  `DispatcherCommitEnabled`, `GitAddFunc`, `ErrCommitMessageTooLong`,
  `ErrCommitSpawnNoTerminal` — confirmed F.7.13 ships ONLY the closed-enum
  entry + `CommitGateRunner` skeleton, no production registry binding.
- Prompt-listed attack surface checklist worked top-to-bottom.

**Concrete falsification plan.**

1. `git add` verbatim-paths claim — does the production GitAddFunc impl
   shell metachar-safely?
2. Toggle interaction — nil-pointer Metadata path?
3. CommitAgent error-chain — does `errors.Is` reach F.7.12 sentinels
   through the gate's wrap?
4. EndCommit mutation race — what if commit succeeds but rev-parse fails?
5. Closed-enum closure — all four canonical kinds valid?
6. Registry binding — does F.7.13 actually wire `commit` into the
   gateRunner registry, or only declare the kind?
7. Memory rule conflicts (Hylla, mage install, raw go).

### QA Proof

**Premises.** Each attack lands or is REFUTED with code-line evidence,
not memory.

**Evidence-completeness.**
- All file-line citations were `Read`-tool retrieved this turn, not
  recalled.
- Counterexample search exhausted the documented surface: the worklog
  lists 8 spec scenarios + 6 robustness tests + 14 test functions — every
  branch I attempted to break either has a pinning test or is documented
  as deferred.
- Hylla feedback section authored unconditionally per project rule
  (N/A — task touches Go code in a single drop slice; per project
  discipline subagent is allowed Hylla but per worklog convention the
  builder didn't query Hylla and neither did this review — the surface
  was small enough that direct Read + rg sufficed).

**Trace coverage.** 7 attacks attempted, 7 verdicts recorded below
(REFUTED / REFUTED-WITH-NIT / CONFIRMED / N/A). No "I think" reasoning
left dangling.

### QA Falsification

**Self-attack pass.** Did I miss an angle?

- Concurrency: gate value holds no mutable state per doc-comment line
  170; mutation is solely via the `*item` pointer the caller owns. The
  attack would need shared state across Run calls — the design rules it
  out by construction. Not a counterexample.
- Goroutine leaks: Run is fully synchronous, no `go` keyword anywhere in
  the algorithm. Not a counterexample.
- Hidden init/global state: package has no `init()`, no package-level
  `var` beyond the five sentinel errors. Not a counterexample.
- Test-order coupling: every test calls `t.Parallel()` and uses
  `t.TempDir()`; recordingGitFns instances are created per test. Not a
  counterexample.
- Path-as-flag injection (`-h`, `--help` in paths slice): the gate hands
  paths verbatim to the seam; production adapter (deferred to F.7.16)
  must enforce `git add -- <paths>`. The doc-comment at line 109
  ("`git add -- <paths...>` (the `--` separator rejects path-as-flag
  injection)") names this. Within F.7.13's API-skeleton scope this is
  appropriate deferral, not a bug.
- F.7.12 sentinel propagation through the gate's `%w` wrap: the gate
  wraps with `fmt.Errorf("dispatcher: commit gate: generate message: %w",
  err)`. Single-`%w` chain works with `errors.Is`. The test
  (`TestCommitGateRunCommitAgentFails`) asserts via a synthetic
  `agentErr`, not via `ErrCommitMessageTooLong` or `ErrCommitSpawnNoTerminal`.
  Mechanism is structurally identical, but a direct sentinel-propagation
  test would tighten the contract. NIT.

**Convergence verdict.** No unmitigated counterexample. One nit
(sentinel-propagation test rigor). Final verdict: PASS-WITH-NITS.

### Convergence

(a) QA Falsification produced no CONFIRMED counterexample — every
attempted attack either REFUTED (code path is correct), REFUTED with a
NIT (correct but could be tightened), or N/A (out-of-scope deferral that
the worklog flags transparently).

(b) QA Proof confirmed evidence completeness — every claim cites the
specific file + line read this turn.

(c) Remaining Unknowns: zero — the F.7.16 wiring + os/exec adapter
implementations are out-of-scope per the worklog AND per the F.7-CORE
plan. The deferred surface is documented with a path forward (Wiring
follow-up section in the worklog), not silently dropped.

## 1. Findings

### 1.1 Attack 1 — `git add` verbatim-paths enforcement (shell metachars)

REFUTED (deferred surface, properly scoped).

The `GitAddFunc` doc-comment (gate_commit.go lines 107–119) says
"Implementations MUST treat the paths slice verbatim — no `-A`, no `.`,
no glob expansion." Production wiring is deferred to F.7.16 per the
worklog "Wiring follow-up" section. F.7.13 ships the test-seam type +
contract; the os/exec-backed adapter is the next droplet's responsibility.

`rg "GitAddFunc"` shows only the type declaration in `gate_commit.go`
plus the test stub in `gate_commit_test.go`. No production adapter
exists yet — so the shell-metachar attack surface (path-as-flag like
`-h`, `--help`, or paths containing `;`, `$()`, backticks) is genuinely
deferred, not dropped. The doc-comment at line 109 names the `--`
separator requirement explicitly for the wiring follow-up.

Within the API-skeleton scope of F.7.13, the contract is correctly
declared. CONFIRMING this in F.7.16 is the next reviewer's job.

### 1.2 Attack 2 — Toggle interaction with nil-pointer metadata

REFUTED.

The toggle path is `project.Metadata.IsDispatcherCommitEnabled()`.

`Project.Metadata` is declared as a value type (`Metadata ProjectMetadata`
at `internal/domain/project.go:65`), NOT a pointer. So a zero-value
`domain.Project` produces a zero-value `ProjectMetadata` where
`DispatcherCommitEnabled == nil`. `IsDispatcherCommitEnabled` is a value
receiver (`func (m ProjectMetadata)`, project.go:182) that returns false
when the pointer is nil (line 183–184). No nil-deref possible.

The gate then short-circuits at gate_commit.go:242–244 (`return nil`).
Verified by `TestCommitGateRunToggleOff` (line 223): the test
intentionally constructs the project with `commitGateProjectToggleOff()`
which leaves `Metadata.DispatcherCommitEnabled` as nil and the gate's
`CommitAgent` as a zero-value `&CommitAgent{}` that would nil-deref if
touched. The test asserts no git shim fires.

### 1.3 Attack 3 — CommitAgent error wrapping + errors.Is reachability

REFUTED-WITH-NIT.

The gate wraps with `fmt.Errorf("dispatcher: commit gate: generate
message: %w", err)` at gate_commit.go:262. Single-`%w` chain — Go's
`errors.Is` walks it correctly. F.7.12 sentinels (`ErrCommitMessageTooLong`,
`ErrCommitSpawnNoTerminal`) are themselves wrapped with `%w` inside
`commit_agent.go:403,413`, so the chain is:

```
gateErr → "dispatcher: commit gate: generate message: %w" → F.7.12 sentinel
```

`errors.Is(gateErr, ErrCommitMessageTooLong)` walks the chain and
resolves true. Mechanism verified.

NIT: `TestCommitGateRunCommitAgentFails` (line 329) asserts the chain
works via a synthetic `errors.New("synthetic build-spawn failure")`, not
via the specific F.7.12 sentinels. The mechanism is structurally
identical (single `%w` wrap chains regardless of underlying error
identity), but a direct `errors.Is(err, ErrCommitMessageTooLong)`
assertion through the gate would tighten the cross-droplet contract.
Suggested addendum (NOT required for PASS): one extra test row that
configures `CommitAgent` to produce a too-long message and asserts both
`errors.Is(err, ErrCommitMessageTooLong)` and the gate's sentinel-free
text-prefix.

### 1.4 Attack 4 — Item mutation: commit succeeds but rev-parse fails

REFUTED.

If `GitCommit` succeeds and `GitRevParseHead` returns an error or empty
string, the gate returns `ErrCommitGateRevParseFailed`-wrapped at
gate_commit.go:292 or 295 BEFORE the `item.EndCommit = newHash` mutation
at line 300. So `item.EndCommit` is unchanged on rev-parse failure —
verified by `TestCommitGateRunGitRevParseFails` (line 440) and
`TestCommitGateRunGitRevParseEmpty` (line 478) which both check
`item.EndCommit != preEndCommit` is false.

The "commit landed but EndCommit empty" inconsistency I attempted to
construct is real on the underlying git repo — the worktree DOES have a
new commit when GitRevParseHead fails — but the in-memory action item
state correctly reflects the failure (EndCommit unchanged, error
returned). The caller is then responsible for routing the action item
to a manual-recovery state per the doc-comment at gate_commit.go:103–104.

This is a documented contract, not a silent failure. Idempotency is
explicitly disclaimed at gate_commit.go:222–225 — callers MUST NOT retry
on failure-after-commit. Worklog "Design notes / deviations" section
also names this disposition.

### 1.5 Attack 5 — Closed-enum closure: all historic kinds valid

REFUTED.

`internal/templates/schema.go:100–105`:
```go
var validGateKinds = []GateKind{
    GateKindMageCI,
    GateKindMageTestPkg,
    GateKindHyllaReingest,
    GateKindCommit,
}
```

`TestGateKindClosedEnum` (`schema_test.go:157–191`) explicitly tests
every value:

- valid: `GateKindMageCI`, `GateKindMageTestPkg`, `GateKindHyllaReingest`,
  `GateKindCommit`.
- invalid: `"push"` (still future), `""`, `"garbage"`, `"MAGE_CI"`
  (case mismatch), `" mage_ci "` (whitespace), `"mage-ci"` (hyphen).

`TestGateKindCommitRegistered` (`gate_commit_test.go:594`) cross-checks
from the dispatcher package that `templates.IsValidGateKind(
templates.GateKindCommit) == true`. Belt-and-suspenders for the
cross-package contract.

The closed-enum vocabulary is closed and complete. "push" remains
correctly rejected.

### 1.6 Attack 6 — GateKindCommit registered in dispatcher gate registry?

REFUTED (correctly NOT registered in F.7.13).

`rg "Register.*GateKindCommit"` returns nothing — confirming the
dispatcher's `gateRunner` registry does NOT bind `GateKindCommit` to a
`gateFunc` in this droplet. The worklog "Wiring follow-up" section
names this explicitly: the `gateFunc` adapter that translates
`CommitGateRunner.Run(ctx, *item, project, catalog, auth) error` to
`gateFunc(ctx, item, project) GateResult` is F.7.16's responsibility
(needs to lift item to a pointer, close over catalog/auth, translate
error → `GateStatusFailed`).

This is the correct scope split for an "API skeleton" droplet. Adding
the wiring here would silently activate the gate at template-load time
the moment a project sets `dispatcher_commit_enabled = true`, which is
out-of-scope for F.7.13 AND would conflict with F.7.16's planned
default-template extension.

### 1.7 Attack 7 — Memory rule conflicts

REFUTED.

- Hylla: Worklog explicitly states "N/A — task touched only Go code …
  per droplet rule 'NO Hylla calls,' no Hylla queries were issued."
  Compliant with project per-droplet Hylla feedback rule.
- `mage install`: `rg "mage install"` finds zero matches in
  `gate_commit.go` / `gate_commit_test.go` / `schema.go` /
  `schema_test.go`. Worklog's verification section names `mage testPkg`
  and `mage ci` only. Compliant with the "NEVER mage install" rule.
- Raw `go build` / `go test` / `go vet`: zero matches. Compliant.
- Migration logic: F.7.13 is a pure additive enum extension + new file.
  No SQL migration, no `till migrate` CLI surface, no schema-version
  bump in the templates package. Compliant with the no-migration-pre-MVP
  rule (the closed-enum extension does NOT require a schema-version
  bump because `validGateKinds` is parsed but the version field stays
  v1).
- Closed-vocabulary discipline: `GateKindCommit` added to BOTH the
  constant declaration AND the `validGateKinds` slice (single source of
  truth pattern preserved per the doc-comment at schema.go:67–69).

## 2. Counterexamples

None. Every attack REFUTED.

## 3. Summary

PASS-WITH-NITS.

The single nit is a test-rigor suggestion (Attack 1.3): one extra test
row that asserts `errors.Is(gateErr, ErrCommitMessageTooLong)` flows
through the gate's `%w` wrap directly, rather than relying on the
synthetic-agent-error proxy. The mechanism is structurally identical;
the suggestion only tightens cross-droplet contract clarity.

**The gate's API skeleton is correctly scoped, the closed-enum extension
is clean, and the test suite (14 functions / 16 effective scenarios)
covers every branch within F.7.13's scope including the four documented
robustness-edge cases (toggle-explicit-false, rev-parse-empty,
nil-receiver, nil-item). The deferred wiring (F.7.16 gateFunc adapter,
real os/exec adapters) is documented transparently with a clear
hand-off contract.**

Verification: `mage ci` green per worklog (2672 tests pass, 1 unrelated
skip; coverage 75.4% on dispatcher / 97.0% on templates).

## TL;DR

- T1: Seven attacks attempted (verbatim-paths, toggle nil, errors.Is
  chain, item mutation race, enum closure, registry binding, memory rules).
  All seven REFUTED; one minor test-rigor nit on F.7.12 sentinel
  propagation testing.
- T2: Zero counterexamples constructed. The deferred surface (production
  os/exec adapters + gateFunc registry binding) is properly scoped to
  F.7.16 per worklog "Wiring follow-up."
- T3: PASS-WITH-NITS — F.7.13 is mergeable; the optional sentinel-
  propagation test row is a sharpening suggestion, not a blocker.
