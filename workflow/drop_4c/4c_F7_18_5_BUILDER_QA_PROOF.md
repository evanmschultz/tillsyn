# Drop 4c F.7.18.5 — BUILDER QA PROOF (Round 1)

**Verdict: PROOF GREEN**

## Round 1

### Premises

- F.7.18.5 acceptance is "seed six `[agent_bindings.<kind>.context]` blocks in `internal/templates/builtin/default.toml` + add tests covering each acceptance bullet."
- REV-4 contract: `build` binding MUST declare `parent_git_diff = true`; the four QA bindings (`build-qa-proof`, `build-qa-falsification`, `plan-qa-proof`, `plan-qa-falsification`) MUST NOT declare it.
- Every seeded binding must carry `parent = true`, `ancestors_by_kind = ["plan"]`, `delivery = "file"`, `max_chars = 50000`, `max_rule_duration = "500ms"`.
- Scope is bounded to (a) `internal/templates/builtin/default.toml`, (b) `internal/templates/embed_test.go`, (c) the worklog MD.
- Builder must NOT commit (REV-13).
- Builder must NOT run `mage install`; build-verification is `mage ci` only.

### Evidence

- `internal/templates/builtin/default.toml` — added `[agent_bindings.<kind>.context]` blocks at lines 399–404 (plan), 446–452 (build), 474–479 (plan-qa-proof), 501–506 (plan-qa-falsification), 535–540 (build-qa-proof), 567–572 (build-qa-falsification).
- `internal/templates/embed_test.go` — eight new test functions starting at line 448.
- `workflow/drop_4c/4c_F7_18_5_BUILDER_WORKLOG.md` — worklog declaring the diff scope and gate results.
- `git status --short` — only `internal/templates/builtin/default.toml` and `internal/templates/embed_test.go` are modified within the F.7.18.5 spawn scope; other dirty files (`internal/adapters/storage/sqlite/repo_test.go`, `internal/domain/project.go`, `internal/domain/project_test.go`, `internal/tui/model_test.go`, `internal/app/dispatcher/binding_resolved.go`, `SKETCH.md`) are pre-existing sibling-droplet work (orch confirmed combined-tree `mage ci` exit 0).
- `rg "parent_git_diff" internal/templates/builtin/default.toml` — exactly one assignment line (`448: parent_git_diff = true`); the four other occurrences are explanatory comments naming the REV-4 prohibition.
- `rg "ContextDeliveryFile|MaxRuleDuration|type ContextRules|type ContextDelivery" internal/templates/schema.go` — all symbols the tests reference exist (constant `ContextDeliveryFile = "file"` at line 637, type `ContextRules` at line 560, field `MaxRuleDuration Duration` at line 622). Tests are not phantom references.

### Trace or cases — F.7.18.5 acceptance bullets vs evidence

#### Bullet 1: REV-4 — `build` has `parent_git_diff = true`, QA bindings DO NOT

| Binding | TOML state | Verdict |
| --- | --- | --- |
| `build` | `default.toml:448` literal `parent_git_diff = true` | PASS |
| `build-qa-proof` | `default.toml:535–540` block — six fields, no `parent_git_diff` line | PASS |
| `build-qa-falsification` | `default.toml:567–572` block — six fields, no `parent_git_diff` line | PASS |
| `plan-qa-proof` | `default.toml:474–479` block — six fields, no `parent_git_diff` line | PASS |
| `plan-qa-falsification` | `default.toml:501–506` block — six fields, no `parent_git_diff` line | PASS |

Independent grep confirms exactly one assignment of `parent_git_diff` in the entire file (line 448, under `[agent_bindings.build.context]`).

#### Bullet 2: Regression-guard test `TestDefaultTemplateQABindingsRejectParentGitDiff` exists + asserts `false` for all four QA kinds

Confirmed at `embed_test.go:479–501`. Iterates `[domain.KindBuildQAProof, domain.KindBuildQAFalsification, domain.KindPlanQAProof, domain.KindPlanQAFalsification]` as named subtests; each fails on `binding.Context.ParentGitDiff == true` with the explicit message `"REV-4 — QA must verify independently"`.

#### Bullet 3: All six bindings declare `ancestors_by_kind = ["plan"]`

Source verification (TOML literals):
- `plan` at line 401 — `ancestors_by_kind = ["plan"]`.
- `build` at line 449 — `ancestors_by_kind = ["plan"]`.
- `plan-qa-proof` at line 476 — `ancestors_by_kind = ["plan"]`.
- `plan-qa-falsification` at line 503 — `ancestors_by_kind = ["plan"]`.
- `build-qa-proof` at line 537 — `ancestors_by_kind = ["plan"]`.
- `build-qa-falsification` at line 569 — `ancestors_by_kind = ["plan"]`.

Test guard: `TestDefaultTemplateContextSeedsAncestorsByKind` (`embed_test.go:507–524`) iterates `contextSeededKinds` and asserts `len(got) == 1 && got[0] == domain.KindPlan`.

#### Bullet 4: All six declare `max_chars = 50000` + `max_rule_duration = "500ms"`

Source verification — every block above contains both literal lines (lines 403–404, 451–452, 478–479, 505–506, 539–540, 571–572).

Test guard: `TestDefaultTemplateContextSeedsCaps` (`embed_test.go:552–574`) asserts `MaxChars == 50000` and `time.Duration(MaxRuleDuration) == 500 * time.Millisecond` per seeded kind.

#### Bullet 5: All six declare `delivery = "file"`

Verified across all six blocks (lines 402, 450, 477, 504, 538, 570). Test guard: `TestDefaultTemplateContextSeedsDelivery` (`embed_test.go:530–546`) iterates the seeded list asserting `Delivery == ContextDeliveryFile`.

#### Bullet 6: All six declare `parent = true`

Verified across all six blocks (lines 400, 447, 475, 502, 536, 568). Test guard: `TestDefaultTemplateContextSeedsParentTrue` (`embed_test.go:581–597`).

#### Bullet 7: Scope — only the three listed files touched within F.7.18.5 spawn scope

Within F.7.18.5's spawn scope, modified files are exactly:
- `internal/templates/builtin/default.toml` (additive — `git diff --stat` shows `1 file changed, 106 insertions(+)`).
- `internal/templates/embed_test.go` (additive test functions + `time` import).
- `workflow/drop_4c/4c_F7_18_5_BUILDER_WORKLOG.md` (new file).

Other dirty files in `git status` are documented in the worklog as pre-existing sibling-droplet state (commits `46e7ce6`, `de41074`, `248416c`, `f6aec8b`, `31700b6`, `e6cd71c`, `16b86cb`, `0cd016b`, `3c281de` are the recent F.7.x landings). The orch's confirmation that `mage ci` exits 0 on the combined working tree means those pre-existing changes do not break anything.

#### Bullet 8: `mage ci` green

Per spawn prompt step 8: orchestrator confirmed exit 0 on combined working tree. Worklog separately records `mage testPkg ./internal/templates/` PASS (355 tests), `mage formatCheck` PASS, `mage build` PASS — all scoped to the package this droplet touched.

#### Bullet 9: NO commit by builder

Verified — `git log --oneline -10` shows no commit added by F.7.18.5 since the most recent prior landing (`46e7ce6 test(dispatcher): add mock adapter fixture for cli adapter contract`); the F.7.18.5 changes are still unstaged in working tree per `git status`.

#### Bullet 10: Eight test scenarios from spawn prompt + builder's two bonus tests

| Test function | Coverage | Verdict |
| --- | --- | --- |
| `TestDefaultTemplateBuildContextSeedsParentGitDiff` | REV-4 positive — build has `parent_git_diff = true` | PASS |
| `TestDefaultTemplateQABindingsRejectParentGitDiff` | REV-4 negative — four QA kinds, named subtests | PASS |
| `TestDefaultTemplateContextSeedsAncestorsByKind` | Six seeded kinds → `["plan"]` | PASS |
| `TestDefaultTemplateContextSeedsDelivery` | Six seeded kinds → `"file"` | PASS |
| `TestDefaultTemplateContextSeedsCaps` | Six seeded kinds → 50000 / 500ms | PASS |
| `TestDefaultTemplateContextSeedsParentTrue` | Six seeded kinds → `parent = true` | PASS |
| `TestDefaultTemplateNonContextSeededKindsHaveZeroContext` | Six non-seeded kinds → zero-value Context (full field-by-field assertion) | PASS — bonus scope-creep guard |
| `TestDefaultTemplatePlanContextHasNoDescendants` | `plan` binding has no `descendants_by_kind` (planner-flexibility cross-check) | PASS — bonus master PLAN L13 A-λ guard |

The two bonus tests align with spawn-prompt expectations: `TestDefaultTemplateNonContextSeededKindsHaveZeroContext` uses `contextSeededKinds` set membership to enumerate the complement (`research`, `closeout`, `commit`, `refinement`, `discussion`, `human-verify`) and asserts a fully zero-value `ContextRules` for each — this is the exact "non-seeded-kinds-have-zero-context" guard requested in the prompt.

### Conclusion

PROOF GREEN. Every acceptance bullet from the spawn prompt — REV-4 positive + negative, the five uniform-field assertions across six bindings, scope discipline, no-commit discipline, `mage ci` confirmation — has direct evidence in the modified TOML or the new test functions. The two bonus tests strictly increase coverage without changing the contract.

### Unknowns

- The orchestrator's `mage ci` exit-0 confirmation is taken on faith per spawn-prompt step 8; this proof does not re-run the gate (read-only role).
- "Pre-existing sibling-droplet work" in `internal/adapters/...`, `internal/domain/project*.go`, `internal/tui/model_test.go`, `internal/app/dispatcher/binding_resolved*.go`, `SKETCH.md` is identified by recent commit log as F.7.x sibling droplet output — not F.7.18.5 work; outside this droplet's scope to verify their correctness.

## Hylla Feedback

`N/A — action item touched non-Go files (TOML, MD) plus a pure additive Go test file with no cross-package symbol search needed.` Spawn prompt explicitly forbids Hylla calls, and the verification surface (TOML literals + closed-file test assertions + git status) was natively covered by `Read`, `Bash` (git/rg), and `git status`/`git diff --stat`. No miss to report.
