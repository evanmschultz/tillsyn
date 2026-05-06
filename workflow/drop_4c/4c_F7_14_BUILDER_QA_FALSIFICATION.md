# F.7-CORE F.7.14 — Push Gate (QA Falsification Round 1)

Read-only adversarial review of the F.7.14 push gate landed at:

- `internal/app/dispatcher/gate_push.go` (NEW, 237 LOC)
- `internal/app/dispatcher/gate_push_test.go` (NEW, 11 tests)
- `internal/templates/schema.go` (EDITED — `GateKindPush` constant + `validGateKinds` extension)
- `internal/templates/schema_test.go` (EDITED — moved `"push"` from invalid → valid; `"mage-ci"` added as new invalid)

Sibling F.7.8 (`orphan_scan.go`/`_test.go`) is in working tree but explicitly out of scope for this review.

## 1. Findings

### 1.1 A1 — Detached HEAD / branch resolution: REFUTED

**Claim under attack:** detached HEAD (no symbolic ref) routes incorrectly.

**Evidence:**
- `gate_push.go:206-212` — `GitCurrentBranch` non-nil error path wraps with `ErrPushGateBranchMissing` via `fmt.Errorf("%w: %w", ErrPushGateBranchMissing, err)`. Empty-string return path wraps with `fmt.Errorf("%w: empty branch returned", ErrPushGateBranchMissing)`.
- `gate_push.go:60-70` doc comment — "Common underlying causes: detached HEAD (CI checked out a tag or a raw commit hash) ..." — design explicitly anticipates this.
- `gate_push_test.go:246-298` — `TestPushGateRunBranchMissingEmpty` and `TestPushGateRunBranchMissingError` pin both shapes. Both assert `GitPush` does NOT fire (`pushCalls` must be 0).

**Trace:** detached HEAD in production → `git symbolic-ref --short HEAD` exits non-zero with stderr `fatal: ref HEAD is not a symbolic ref` → adapter returns non-nil error (or empty string if adapter swallows the exit code) → both shapes collapse to `ErrPushGateBranchMissing`. Underlying error is reachable via `errors.Is`. `GitPush` never invoked.

**Verdict:** REFUTED. The two-shape collapse is documented, tested, and the underlying error remains inspectable.

### 1.2 A2 — Toggle ordering footgun (push enabled, commit disabled): CONFIRMED-AS-ACCEPTED

**Claim under attack:** dev sets `DispatcherPushEnabled=&true` while leaving `DispatcherCommitEnabled=nil/&false`. Push runs against whatever HEAD already exists → pushes prior unrelated commit (or pushes nothing new but advances remote ref to local stale state).

**Evidence:**
- `internal/domain/project.go:182-203` — `IsDispatcherCommitEnabled` and `IsDispatcherPushEnabled` are independent functions; no cross-toggle ordering enforcement at the domain layer.
- `internal/domain/project.go:194-197` doc comment — "Independent of `IsDispatcherCommitEnabled`: callers compose them explicitly when push depends on commit (push-without-commit is non-sensical, but the toggles do not enforce that ordering — gate execution does)."
- `gate_push.go:152-225` Run algorithm — step 1 reads `IsDispatcherPushEnabled` only. There is NO check that `EndCommit` was populated (which would signal F.7.13 commit gate ran successfully) and NO check that the previous gate in the sequence was the commit gate.
- `gate_commit.go:300` — commit gate writes `item.EndCommit = newHash`. PushGate could read this as a "commit happened" signal but does not.
- `F7_CORE_PLAN.md:38` — "Drop 4b closed (gate runner, lock manager, post-build pipeline) | merged | F.7.13, F.7.14, F.7.16" — gate sequencing is a runner-level concern, currently DEFERRED to the wiring droplet (gate_push.go:7-11 doc comment).
- `F7_CORE_PLAN.md:945-946` (F.7.16 acceptance) — "Fresh project with default template + nil metadata flags → gate runner runs `mage_ci`, then `commit` (skipped per F.7.13), then `push` (skipped per F.7.14)." — F.7.16 documents the intended sequence but neither gate enforces "commit-must-precede-push."

**Trace (counterexample reproduction):**
1. Dev edits `~/.tillsyn/tillsyn.db` → sets `DispatcherPushEnabled=&true`, leaves `DispatcherCommitEnabled=nil`.
2. Build agent reports success on droplet `D` with `Paths=["foo.go"]`.
3. F.7.16 wiring (when it lands) runs `[gates.build]=["mage_ci","commit","push"]`. mage_ci passes. commit gate no-ops (toggle off). push gate runs. `git push origin <branch>` advances remote with WHATEVER HEAD currently points at — which may be a prior commit unrelated to droplet D, or the dev's manual mid-build commit. The remote state diverges from the droplet's intended scope.
4. No `ErrPushGate*` fires, no warning logged, no `metadata.BlockedReason` set. Action item moves to `complete`.

**Counterexample severity:** SOFT. The enforcement gap is real but the project metadata states `DispatcherPushEnabled` is "Default OFF; dev flips this on AFTER `DispatcherCommitEnabled` has proven safe in dogfood" (project.go:151-152). The dev is documented as the responsible party for not creating this footgun. Furthermore:

- F.7.14 spec (F7_CORE_PLAN.md:836-883) does NOT list "push-after-commit ordering enforcement" as an acceptance criterion.
- The gate-runner adapter (deferred to a future droplet, currently mis-cited as REV-13 — see 1.3 below) is the architectural place where ordering would be enforced. Per `feedback_no_migration_logic_pre_mvp.md` spirit, building cross-gate ordering before the runner adapter exists is premature.
- `IsDispatcherPushEnabled` doc comment EXPLICITLY pins this as a runner-level concern, not a per-gate concern.

**Verdict:** CONFIRMED-AS-ACCEPTED. The footgun is real but the contract explicitly punts ordering to the runner adapter, the dev is documented as responsible for staged toggle activation, and the F.7.14 spec does not include ordering enforcement. Recommend a NIT acceptance criterion on the future runner-adapter droplet: "When `[gates.build]` includes both `commit` and `push`, the runner MUST refuse to run `push` if `commit` did not produce a non-empty `item.EndCommit` in the same Run." Tracker raised below in §4.

### 1.3 A6 — Doc-comment drift: gate_push.go cites "F.7-CORE REV-13" three times for the future-wiring droplet, but REV-13 is unrelated: CONFIRMED-NIT

**Claim under attack:** the gate_push.go doc comments name the wrong revisions reference.

**Evidence:**
- `gate_push.go:7-11` — "wiring droplet (F.7-CORE REV-13) adapts both gates to the gateRunner's gateFunc interface uniformly".
- `gate_push.go:165-167` — "the wiring droplet (F.7-CORE REV-13) treats both gates uniformly".
- `gate_push.go:228-229` — "future gateRunner-adapter (F.7-CORE REV-13) can treat both uniformly".
- `F7_CORE_PLAN.md:1103-1107` REV-13 is titled **"Builder spawn prompts MUST explicitly forbid self-commit"** — about builder process discipline, NOT about gate-runner-adapter wiring. No code change.
- `F7_CORE_PLAN.md:1000-1115` — every REV-1 through REV-15 is enumerated; none of them name a gate-runner-adapter wiring droplet. The actual wiring task is listed only in F7_CORE_PLAN.md:135 ("F.7.12 → F.7.15 → (F.7.13 ‖ F.7.14) → F.7.16 form the commit/push integration") and F.7.16 itself (F7_CORE_PLAN.md:925-946) handles the default-template gate-list expansion but does NOT include the runner-adapter shim.
- For comparison `gate_commit.go:11-14` says only "Drop 4c follow-up wiring droplet adapts CommitGateRunner.Run to the gateRunner's gateFunc interface" — no specific REV-# named, which is correct.

**Verdict:** CONFIRMED-NIT. Three doc-comment lines mis-cite REV-13. Either the gate-runner-adapter wiring droplet is unnamed today (in which case `gate_push.go` should match `gate_commit.go`'s style — "future wiring droplet" without a specific REV-# anchor) or the spec needs a new REV entry naming this work. Doc-only fix; no behavior impact. Suggested rewrite: replace each "(F.7-CORE REV-13)" with "(future wiring droplet, see F.7.16 + REV-10)".

### 1.4 A3 — Hardcoded "origin" in gate code: REFUTED

**Claim under attack:** push gate hardcodes `origin` as the remote, breaking adopters whose remote is named differently.

**Evidence:**
- `rg '"origin"' internal/app/dispatcher/gate_push.go` returns no matches.
- `gate_push.go:101` — `GitPushFunc func(ctx, repoPath, branch) error`. The remote is NOT a parameter; the production adapter (not yet built) is responsible for the remote-name choice.
- F.7.14 plan "Out of scope: Multi-remote push (out — `origin` only)" applies to the production adapter, not the gate-runner logic. The gate is remote-agnostic.

**Verdict:** REFUTED. Gate logic does not hardcode any remote. Production adapter wiring is appropriately deferred and explicitly out-of-scope.

### 1.5 A4 — Existing gate-registry tests not updated for new GateKinds: REFUTED-WITH-NIT

**Claim under attack:** F.7.13 + F.7.14 added two new GateKinds; the existing `gates_test.go` registry tests don't exercise them.

**Evidence:**
- `internal/templates/schema_test.go:152-191` — `TestGateKindClosedEnum` updated to include `GateKindCommit` (line 164) AND `GateKindPush` (line 165) in the valid-cases slice. Invalid-cases slice updated (line 181) to use `"mage-ci"` instead of `"push"` (which is now valid).
- `gate_push_test.go:381-394` — `TestGateKindPushRegistered` cross-checks `IsValidGateKind(GateKindPush)` AND asserts the literal string `"push"`.
- `gates_test.go` (the gateRunner-Register tests) does NOT exercise `GateKindPush` / `GateKindCommit` — but this is intentional. The runner adapter for these gates is deferred (see 1.2). Today they have NO Register call; their dispatch surface is Run-on-the-struct, not the closed-enum runner registry. Adding gates_test.go cases against a deferred Register call would be premature.

**Verdict:** REFUTED-WITH-NIT. The closed-enum membership is properly tested. The gate-runner-Register integration is correctly deferred (sibling note 1.2). NIT: the `TestGateKindPushRegistered` name is slightly misleading because there is no `gateRunner.Register(GateKindPush, ...)` call yet — the test only verifies the closed-enum membership. Recommend renaming to `TestGateKindPushIsValidEnumMember` for precision, but this is purely cosmetic.

### 1.6 A5 — Memory-rule conflicts: REFUTED

- **No raw `go` invocations:** worklog reports `mage check` + `mage ci` only (4c_F7_14_BUILDER_WORKLOG.md:64). No `go test` / `go build` / `go vet`.
- **No `mage install`:** worklog explicitly confirms (line 46).
- **No commit by builder:** worklog line 40 + 44 confirm. `git status` shows untracked files, no F.7.14-authored commit. Per REV-13's actual content (forbid builder self-commit), this is honored.
- **No migration logic:** F.7.14 adds no SQL migration; `GateKindPush` is a constant-table extension, validated at TOML-load time. Toggle metadata (F.7.15) ships JSON-encoded blob inside `domain.ProjectMetadata` per REV-6.
- **No Hylla calls:** worklog confirms (line 47).

**Verdict:** REFUTED.

### 1.7 A7 (self-attack) — `ctx` cancellation between branch resolve and push: REFUTED-WITH-NIT

**Claim under attack:** between `GitCurrentBranch` returning successfully and `GitPush` firing, ctx is canceled. The gate doesn't check ctx.Err() and proceeds with a stale ctx.

**Evidence:**
- `gate_push.go:206-220` — Run does NOT check `ctx.Err()` between calls.
- BUT each seam (`GitCurrentBranch`, `GitPush`) takes `ctx` as its first argument. Production adapters that shell `os/exec` via `exec.CommandContext(ctx, ...)` will respect ctx cancellation themselves. The `os/exec` package's `CommandContext` is the canonical Go-idiomatic propagation mechanism (verified pattern in stdlib, see `internal/app/dispatcher/gate_mage_ci.go` for in-tree precedent).

**Verdict:** REFUTED-WITH-NIT. ctx propagation is correct. NIT: an explicit `if err := ctx.Err(); err != nil { return ... }` between steps would surface mid-gate cancellation as a distinct error class rather than letting it surface through whichever seam happens to fire first. Defense-in-depth only; not a counterexample. The CommitGateRunner has the same pattern (no inter-step ctx check), so symmetry holds.

### 1.8 A8 (self-attack) — Branch verbatim with shell-meta characters: REFUTED

**Claim under attack:** what if `GitCurrentBranch` returns a branch like `drop/4c; rm -rf /` or with quotes? The gate forwards verbatim to `GitPush`.

**Evidence:**
- `gate_push.go:214-217` doc comment: "The branch name is taken verbatim from `GitCurrentBranch` — no automatic prefix, no automatic refspec rewriting, no force-push." Forwarding is by design.
- The production `GitPushFunc` is expected to use `os/exec` with argv-list invocation (matching `gate_commit.go:107-119` `GitAddFunc` pattern), which does NOT shell-interpret arguments. `git push origin <branch>` with `branch="drop/4c; rm -rf /"` would have git itself reject the ref name (`fatal: 'drop/4c; rm -rf /' is not a valid ref name`).
- The test `TestPushGateRunHappyPath` uses `wantBranch := "drop/4c"` (line 128) — a real-world slash-bearing branch — and asserts the verbatim flow.

**Verdict:** REFUTED. Verbatim forwarding is documented and the argv-list pattern means the production adapter is not shell-vulnerable. Git's own ref-name validation is the authoritative checker.

### 1.9 A9 (self-attack) — `60s push timeout` acceptance criterion not enforced in gate: NIT

**Claim under attack:** F.7.14 plan acceptance criterion (F7_CORE_PLAN.md:862) states "Push timeout: 60s (network operations need real-world breathing room)". `gate_push.go` does NOT impose any timeout.

**Evidence:**
- `gate_push.go:179-225` — Run accepts `ctx context.Context` and forwards to seams. No `context.WithTimeout(ctx, 60*time.Second)` wrapper.
- Test `TestPushGateRunPushFails` does not exercise timeout behavior.

**Verdict:** NIT. Timeout enforcement is a production-adapter concern (the `GitPushFunc` implementation can wrap `exec.CommandContext` with a 60s `context.WithTimeout`), and the gate signature accepts an injectable seam. Hardcoding the timeout in `Run` would conflict with the seam-injection design — the test fixtures need fast-path returns. Recommend: when the production adapter lands, the 60s timeout MUST be applied inside the `GitPushFunc` adapter. Not a counterexample to F.7.14's gate code.

### 1.10 A10 (self-attack) — Description-vs-code drift: REFUTED

The action-item description (per spawn prompt + F7_CORE_PLAN.md:836-883) names: PushGateRunner + sentinels + `GateKindPush` enum extension. Builder shipped:

- `PushGateRunner` ✓
- `ErrPushGateDisabled`, `ErrPushGateBranchMissing`, `ErrPushGatePushFailed` ✓
- `GateKindPush` constant + `validGateKinds` slice extension ✓
- `GitCurrentBranchFunc` + `GitPushFunc` test seams ✓ (DI-matching the F.7.13 shape)
- 11 unit tests covering the 6 mandated scenarios + 5 defense-in-depth ✓

**Verdict:** REFUTED. No silent re-interpretation of the action item.

## 2. Counterexamples

No CONFIRMED counterexamples that block this droplet. The two CONFIRMED items are:

### 2.1 CONFIRMED-AS-ACCEPTED — Toggle ordering footgun (1.2)

Reproduction is real (push-without-commit pushes prior HEAD), but the contract explicitly defers ordering enforcement to the future runner-adapter droplet, the dev is documented as responsible for staged toggle activation, and F.7.14's acceptance criteria do not include this enforcement. Action: raise as a NIT against the unnamed future runner-adapter droplet — see §4.

### 2.2 CONFIRMED-NIT — REV-13 doc-comment drift (1.3)

Three doc-comment citations in `gate_push.go` name `F.7-CORE REV-13` for the future-wiring droplet, but REV-13 in `F7_CORE_PLAN.md:1103` is "Builder spawn prompts MUST explicitly forbid self-commit" — unrelated. Action: doc-only edit; replace with "future wiring droplet (see F.7.16 + REV-10)" or add a new REV entry naming the runner-adapter work explicitly. No behavior impact.

## 3. Summary

**Verdict: PASS-WITH-NITS.**

- 7 attacks REFUTED (A1 detached HEAD, A3 hardcoded origin, A5 memory rules, A7 ctx cancellation, A8 shell-meta branch, A10 description drift, plus implicit A4 closed-enum membership).
- 2 attacks CONFIRMED but accepted-with-tracker (A2 toggle ordering, A6 REV-13 doc drift).
- 1 attack NIT (A9 60s timeout deferred to production adapter).

Code is correct against F.7.14's spec. The droplet ships clean tests, clean closed-enum extension, clean DI seams, clean two-shape error collapse, and parity with F.7.13's CommitGateRunner shape. The future runner-adapter droplet (deferred) is the right place to land A2's ordering enforcement; A6's doc-comment fix is trivial and can land in any subsequent edit to gate_push.go.

## 4. Recommendations / Follow-Up Tracker

- **Doc-only fix (NIT, can land alongside any future gate_push.go edit):** replace three `(F.7-CORE REV-13)` mentions in `gate_push.go:8`, `gate_push.go:166`, `gate_push.go:229` with `(future wiring droplet, see F.7.16 + REV-10)` or equivalent.
- **Future runner-adapter droplet acceptance criterion:** "When `[gates.build]` includes both `commit` and `push`, the runner MUST refuse to invoke `push` if the preceding `commit` gate did not produce a non-empty `item.EndCommit` in the same Run." Surfaces A2 ordering footgun at gate-execution time without burdening F.7.14 itself.
- **Production adapter:** when `adapters.GitPush` lands, the 60s timeout from F.7.14 acceptance criterion #4 must be applied inside the adapter (`exec.CommandContext` + `context.WithTimeout(ctx, 60*time.Second)`).
- **Test polish (cosmetic):** rename `TestGateKindPushRegistered` → `TestGateKindPushIsValidEnumMember` — the current name implies gateRunner-Register coverage that does not yet exist.

## TL;DR

- **T1:** PushGateRunner + sentinels + closed-enum extension are correct, well-tested, and parity-matched with F.7.13 CommitGateRunner. No blocking counterexamples.
- **T2:** CONFIRMED-but-accepted: A2 push-without-commit toggle footgun (deferred to future runner-adapter, dev-responsibility documented). A6 REV-13 doc-comment drift in three lines of gate_push.go (REV-13 actually names builder-self-commit, not runner-adapter wiring).
- **T3:** Verdict PASS-WITH-NITS. Recommend doc-comment fix + future-droplet acceptance criterion; F.7.14 itself ready to commit.
- **T4:** Follow-up tracker captures four items: REV-13 doc fix, runner-adapter ordering criterion, production-adapter 60s timeout, test rename.

## Hylla Feedback

Recorded misses during this review:

- **Query**: `mcp__hylla__hylla_search_keyword` with queries `IsDispatcherPushEnabled`, `IsDispatcherCommitEnabled DispatcherCommitEnabled`, `CommitGateRunner gateRunner gateFunc`.
- **Missed because**: the symbols are defined in F.7.13 + F.7.14 + F.7.15 work, all of which is in the local working tree but uncommitted (gate_push.go and gate_push_test.go are untracked; gate_commit.go and project.go's toggle additions are committed in earlier F.7.13 / F.7.15 droplets but evidently not yet ingested into the snapshot Hylla read). Snapshot lag on the active branch — expected per CLAUDE.md §"Code Understanding Rules" rule 2.
- **Worked via**: `rg` + `Read` on the working tree directly. Found everything in seconds.
- **Suggestion**: when Hylla returns zero hits for a symbol that the working tree clearly defines, surfacing a "snapshot N covers commit X; symbol may exist in newer commits" hint in the empty-results envelope would short-circuit the fallback path. Today the empty result is identical between "symbol does not exist" and "symbol is post-snapshot".
