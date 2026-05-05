# Drop 4c F.7.15 — Builder QA Falsification (Round 1)

**Droplet:** F.7-CORE F.7.15 — Project-metadata toggles `dispatcher_commit_enabled` + `dispatcher_push_enabled`
**Reviewer:** go-qa-falsification-agent (subagent, opus, Drop 4c)
**Date:** 2026-05-04
**Verdict:** PASS

---

# Section 0 — SEMI-FORMAL REASONING

## Proposal

**Premises.** F.7.15 claims: (1) `DispatcherCommitEnabled *bool` + `DispatcherPushEnabled *bool` added to `ProjectMetadata` with TOML/JSON tags + `omitempty`; (2) `IsDispatcherCommitEnabled()` + `IsDispatcherPushEnabled()` accessors with nil-means-FALSE semantics (inverse of `OrchSelfApprovalIsEnabled`); (3) tests cover three states + JSON round-trip + TOML round-trip + SQLite round-trip; (4) no DDL — rides `metadata_json` envelope; (5) `mage ci` green; (6) no commit (REV-13).

**Evidence gathered.**
- `Read internal/domain/project.go:119-203` — both fields + both accessors visible with the documented polarity.
- `Read internal/domain/project_test.go` (full file) — 4 new tests landed: defaults x2, JSON round-trip, TOML round-trip.
- `git diff internal/adapters/storage/sqlite/repo_test.go internal/tui/model_test.go` — SQLite round-trip test (5-case table) + TUI schema-coverage 2-line classification.
- `git status` — files `M`, no commit landed; `git log internal/domain/project.go` confirms HEAD remains `77f4541` (4a.25 commit) for that path.
- `Bash rg metadata_json internal/adapters/storage/sqlite/repo.go` — JSON envelope is the project-row columns at lines 870/893/925/942/955/988. `OrchSelfApprovalEnabled` does not appear in repo.go (no symbol-level path). New fields ride the same envelope.
- `Bash rg pelletier/go-toml/v2 go.mod` — line 66: lib is a direct go.mod entry.
- `mage testPkg ./internal/domain/` — 303/303 pass on combined working tree (F.7.15 + F.7.17.8 + F.7.18.5 all on disk).
- `mage testPkg ./internal/adapters/storage/sqlite` — 93/93 pass on combined tree.
- `mage testPkg ./internal/tui` — 372/372 pass on combined tree.
- `mage testPkg ./internal/app/dispatcher` — 188/188 pass on combined tree (validates F.7.17.8 sibling does not break F.7.15 surface).
- `Read workflow/drop_4c/PLAN.md` L56 — Master PLAN L20 confirmed: "Commit + push gates default OFF via `dispatcher_commit_enabled` + `dispatcher_push_enabled`".
- `Read workflow/drop_4c/F7_CORE_PLAN.md` L886-921, L996-1095 — F.7.15 acceptance criteria + REV-6 (struct-field-add precedent), REV-13 (no-self-commit) confirmed.
- `git show 77f4541 --stat` — 4a.25 also touched `internal/tui/model_test.go` (8 lines), confirming TUI schema-coverage edit is precedent-faithful.

**Falsification plan.** Attack vectors A1–A12 listed in spawn prompt. For each, attempt to construct a counterexample.

## QA Proof

**Premises.** Every attack-vector verdict is grounded in Read/diff/mage evidence above. No verdict relies on remembered behavior.

**Trace or cases.** Per-attack mapping:
- A1 polarity: project.go:182-187 + project.go:198-203 + project_test.go:111-163.
- A2 JSON: project_test.go:172-264 + Go encoding/json semantics for `*bool` + `omitempty`.
- A3 TOML: project_test.go:274-356 + pelletier/go-toml/v2 (go.mod L66).
- A4 SQLite: repo_test.go:4201-4301 + repo.go:870-988 (`metadata_json` column).
- A5 naming: project.go method names match `IsDispatcher{Commit,Push}Enabled()` — note `OrchSelfApprovalIsEnabled()` precedent reverses the verb position.
- A6 TUI: model_test.go:15020-15029.
- A7 forward-compat: accessor methods on `ProjectMetadata` returning `bool`; consumers must call accessor (no exported `*bool` reads forced, but field IS exported — see NIT below).
- A8 mage ci: all four mage testPkg invocations green.
- A9 doc-comment quality: project.go:129-148 cites L20 and explains forward-compat rationale.
- A10 no-commit: git log + git status verified.
- A11 memory rules: no JSON-blob DDL → no migration logic; single-domain → short-context fine.
- A12 4a.25 fidelity: 4a.25 commit stat confirms identical file-touch shape.

**Conclusion.** Evidence supports each verdict; no claim is unsupported.

## QA Falsification

I attacked my own PASS verdict on each vector:
- A2 (JSON): would `omitempty` on `*bool(false)` collapse to omitted? No — `omitempty` on a pointer keys off nil-vs-non-nil, not the dereferenced zero value. The test `both_explicit_false` asserts the substring `"dispatcher_commit_enabled":false` IS present — which is actually a stronger guard than nil-vs-not, but still passes mage testPkg. Confirmed safe.
- A3 (TOML): pelletier/go-toml/v2 honors `omitempty` on `*bool` the same way (nil → omit, `*false` → emit). Test asserts `dispatcher_commit_enabled = false` IS present in TOML output for `*false` → matches Go json semantics on pointer types. Confirmed safe.
- A4 (SQLite): the repo_test exercises both CREATE and UPDATE paths. Could the JSON envelope on read silently default `*nil` to `*false`? The test's `nilCommit/nilPush` boolean assertions explicitly check pointer-shape preservation across CreateProject + GetProject. Confirmed safe.
- A5 (naming): `OrchSelfApprovalIsEnabled()` (verb `Is` AFTER subject) vs `IsDispatcherCommitEnabled()` (verb `Is` BEFORE subject) is asymmetric. This is a NIT, not a counterexample — both names are grammatically valid Go and the existing `Is*` prefix convention is more idiomatic. Recording as NIT, not failure.
- A7 (forward-compat): F.7.13/F.7.14 callers read via accessor → returns bool → boolean check. If a future caller reads `m.DispatcherCommitEnabled` directly (non-nil, dereferences `false`), they get the same answer as the accessor. No semantic divergence. Confirmed safe.
- A8 (combined tree): I ran four mage testPkg invocations on the combined tree containing F.7.15 + F.7.17.8 (binding_resolved.go) + F.7.18.5 modifications. All green. The builder's earlier mage ci is corroborated by my fresh invocation.

No unmitigated counterexample emerged. NITs surfaced (A5 naming inconsistency) are surfaced in body without blocking PASS.

## Convergence

(a) QA Falsification produced no unmitigated counterexample to PASS verdict.
(b) QA Proof confirmed every per-attack verdict is grounded in Read/diff/mage evidence.
(c) Remaining Unknowns: none. NITs (A5 naming asymmetry) are explicit and routed to body. No items deferred.

Verdict converges: **PASS** (with one NIT noted).

---

# 1. Findings

- 1.1 **All 12 attack vectors REFUTED.** Detailed per-vector verdicts in §2.
- 1.2 **mage testPkg green on combined tree.** I independently ran `mage testPkg ./internal/domain/` (303/303), `./internal/adapters/storage/sqlite` (93/93), `./internal/tui` (372/372), `./internal/app/dispatcher` (188/188) — all green on the working tree containing F.7.15 + F.7.17.8 + F.7.18.5 simultaneously. Builder's `mage ci` claim corroborated independently against the live combined state.
- 1.3 **Drop 4a 4a.25 precedent fidelity confirmed.** `git show 77f4541 --stat` shows 4a.25 also touched `internal/tui/model_test.go` (8 lines, identical pattern). Builder's 2-line schema-coverage classification is precedent-faithful, not scope creep.
- 1.4 **REV-13 honored.** `git status -- internal/domain/project.go internal/domain/project_test.go internal/adapters/storage/sqlite/repo_test.go internal/tui/model_test.go` returns 4 `M` lines; `git log` HEAD remains `77f4541` (the 4a.25 commit) — no F.7.15 commit landed. Worklog explicitly says "No commit by builder."
- 1.5 **NIT (non-blocking) — accessor naming asymmetry.** Existing precedent is `OrchSelfApprovalIsEnabled()` (verb `Is` AFTER subject — read as "OrchSelfApproval is enabled"). New methods use `IsDispatcherCommitEnabled()` / `IsDispatcherPushEnabled()` (verb `Is` BEFORE subject — read as "is DispatcherCommit enabled"). Both grammatically valid Go; the new `Is*` prefix is arguably MORE idiomatic for boolean accessors (Effective Go § "Getters" recommends `Is*` for predicate methods). Naming this as a NIT, not a counterexample, because (a) both forms appear in stdlib (`time.Time.IsZero` vs `os.FileInfo.IsDir`), (b) the test names mirror the method names so test discovery works, (c) coercing into `OrchSelfApproval`-style would require renaming the existing accessor too (out of scope for F.7.15). Surfacing for orchestrator routing.

# 2. Counterexamples — per-attack verdicts

- 2.1 **A1 — Polarity correctness.** REFUTED. `project.go:182-187` shows `IsDispatcherCommitEnabled()` returns `false` on nil and `*ptr` otherwise — same shape as `OrchSelfApprovalIsEnabled` but with `false` (not `true`) as the nil branch. `project_test.go:120` asserts `nil_defaults_to_disabled → false`. `mage testPkg ./internal/domain/` green. Polarity inversion vs `OrchSelfApprovalIsEnabled` is documented at `project.go:178-181`. No counterexample.

- 2.2 **A2 — JSON omit-vs-false semantics.** REFUTED. Go `encoding/json` `omitempty` on `*bool` keys off nil-vs-non-nil, NOT the dereferenced zero-value. Test `both_explicit_false` (project_test.go:206-213) asserts `"dispatcher_commit_enabled":false` IS present in marshaled JSON; test `both_nil_omits_both_keys` (project_test.go:187-194) asserts the keys are absent. Round-trip preserves nil-vs-non-nil pointer shape (project_test.go:254-261). No counterexample.

- 2.3 **A3 — TOML omit-vs-false semantics.** REFUTED. pelletier/go-toml/v2 `omitempty` on `*bool` matches Go json: nil → omit, `*false` → emit `dispatcher_commit_enabled = false`. Test `TestProjectMetadataDispatcherTogglesTOMLRoundTrip` (project_test.go:274-356) asserts both states. `mage testPkg ./internal/domain/` green. No counterexample.

- 2.4 **A4 — SQLite round-trip via metadata_json.** REFUTED. `internal/adapters/storage/sqlite/repo.go:870/893/925/942/955/988` shows project rows persist via the `metadata_json TEXT` column; the JSON envelope round-trip already verified by A2 carries through. `TestRepository_PersistsDispatcherCommitAndPushEnabled` (repo_test.go:4201+) covers 5 cases through both CreateProject + UpdateProject paths, including pointer-shape preservation (`nilCommit`/`nilPush` flags at lines 4276-4286). `mage testPkg ./internal/adapters/storage/sqlite` returns 93/93. No counterexample.

- 2.5 **A5 — Accessor naming convention.** REFUTED as a counterexample, surfaced as NIT 1.5. Both `Is*Enabled` and `*IsEnabled` are valid Go predicate-method idioms.

- 2.6 **A6 — TUI schema-coverage symmetry.** REFUTED. `internal/tui/model_test.go:15020-15029` adds `DispatcherCommitEnabled` + `DispatcherPushEnabled` to `projectMetadataInternal` map — correct classification (these are dispatcher-internal toggles, not user-form fields). The doc-comment explicitly cites the `OrchSelfApprovalEnabled` precedent. `mage testPkg ./internal/tui` green (372/372). 4a.25 commit `77f4541` made the identical-shape edit, confirming pattern fidelity. No counterexample.

- 2.7 **A7 — Forward-compat for F.7.13/F.7.14 gates.** REFUTED. Future gate consumers will call `m.IsDispatcherCommitEnabled()` and gate-execute only when true. Field is exported (`DispatcherCommitEnabled`), so a hypothetical caller reading the pointer directly would still observe nil-vs-non-nil and dereference correctly — no semantic divergence vs the accessor path. The accessor is the recommended path (centralizes nil-handling) and the F.7.15 doc-comment at `project.go:194-197` warns that `IsDispatcherPushEnabled` is independent of `IsDispatcherCommitEnabled` (callers compose them — push-without-commit is non-sensical but the toggles do not enforce ordering, gate execution does). No counterexample.

- 2.8 **A8 — `mage ci` green per worklog (combined tree).** REFUTED. I independently ran four mage testPkg invocations on the live combined working tree (F.7.15 + F.7.17.8 dispatcher binding_resolved.go + F.7.18.5 sketch + template edits): domain 303/303, sqlite 93/93, tui 372/372, dispatcher 188/188. Combined tree compiles and tests pass. Builder's earlier full `mage ci` (2492 tests) is corroborated. No counterexample.

- 2.9 **A9 — Doc-comment quality.** REFUTED. `project.go:129-148` (`DispatcherCommitEnabled`) cites Master PLAN.md L20, the W3.2 falsification-attack-3 mitigation rationale, the polarity inversion rationale, and the pre-MVP forward-compat rationale (`reserves the pointer shape if Drop 5+ wants to introduce a third state ... project-default-overrides-template-default merge logic`). `project.go:150-154` (`DispatcherPushEnabled`) is more terse but explicitly delegates the three-state shape commentary upward. `project.go:178-181` (accessor) explains the polarity inversion vs `OrchSelfApprovalIsEnabled`. Doc-comments explain WHY (default-off until Drop 5 dogfood proves safe), not just WHAT. No counterexample.

- 2.10 **A10 — No-commit per REV-13.** REFUTED. `git log internal/domain/project.go` HEAD remains `77f4541` (4a.25). `git status --porcelain` shows the four touched files as `M` (working-tree modifications, not committed). Worklog Acceptance Criteria checkbox: `[x] No commit by builder (orchestrator commits after QA pair returns green)`. Proposed commit message is documented in worklog for the orchestrator to use post-QA. REV-13 honored. No counterexample.

- 2.11 **A11 — Memory rule conflicts.** REFUTED.
  - `feedback_no_migration_logic_pre_mvp.md` — JSON-blob ride means no DDL, no migration. Compliant.
  - `feedback_subagents_short_contexts.md` — single-domain task (`internal/domain` + small SQLite/TUI test deltas), short context. Compliant.
  - `feedback_never_remove_workflow_files.md` — no workflow files removed. Compliant.
  - `feedback_orphan_via_collapse_defer_refinement.md` — N/A (no enum/catalog collapse).
  - `feedback_no_closeout_md_pre_dogfood.md` — only worklog + this falsification MD created; no LEDGER/CLOSEOUT/REFINEMENTS rollups. Compliant.
  No counterexample.

- 2.12 **A12 — Drop 4a 4a.25 precedent fidelity.** REFUTED. Comparing `git show 77f4541 --stat` to F.7.15 working tree:
  - Both touch `internal/domain/project.go` (struct field + accessor method).
  - Both touch `internal/domain/project_test.go` (defaults table + round-trip test).
  - Both touch `internal/tui/model_test.go` (schema-coverage classification, 8 vs 10 lines).
  - 4a.25 also touched `internal/app/auth_requests.go` + `internal/app/auth_requests_test.go` + `app_service_adapter_auth_requests_test.go` + `internal/domain/errors.go` — these are gate-consumer sites (orch-self-approval is consumed in Drop 4a Wave 3 itself). F.7.15 does NOT touch consumer sites because gate consumption is F.7.13/F.7.14's territory (Drop 4d). This asymmetry is correct, not drift — F.7.15 is a pure schema droplet, not a gate-implementation droplet.
  - F.7.15 ADDS one path 4a.25 didn't: `internal/adapters/storage/sqlite/repo_test.go` round-trip test. This is additive, not divergent — 4a.25 lacked an explicit SQLite round-trip and was acceptable then, but adding one for the new toggles is good defensive practice given the doc-comment promises three-state preservation across persistence.
  No counterexample.

# 3. Summary

**Verdict: PASS** (with one non-blocking NIT — accessor naming asymmetry, §1.5).

All 12 attack vectors REFUTED. Combined-tree `mage testPkg` corroborates builder's `mage ci` claim. REV-13 (no self-commit) honored. 4a.25 precedent followed faithfully with one additive enhancement (explicit SQLite round-trip test). Three-state pointer-bool semantics preserved across JSON, TOML, and SQLite encoders. Polarity inversion vs `OrchSelfApprovalEnabled` correctly documented and tested. Forward-compat via accessors for F.7.13/F.7.14 gate consumers established.

NIT 1.5 (accessor naming asymmetry — `IsDispatcherCommitEnabled` vs the existing `OrchSelfApprovalIsEnabled`) is grammatically valid in both forms and the new `Is*` prefix is more idiomatic. Renaming the precedent is out of scope for F.7.15. Orchestrator may choose to surface as a Drop 4c refinement or accept as-is.

## Hylla Feedback

`N/A — directive explicitly forbade Hylla calls for this droplet.` Native-tool path (Read + git diff + mage testPkg) was sufficient given the precedent (`OrchSelfApprovalEnabled` at `internal/domain/project.go:119-145`) was a known-good anchor at a known line range. No fallback paths exercised; no Hylla miss to record.

## TL;DR

- T1 12/12 attack vectors REFUTED; 1 non-blocking NIT (naming asymmetry §1.5).
- T2 Per-attack verdicts grounded in Read + git diff + 4 independent `mage testPkg` runs on the combined working tree.
- T3 PASS — one NIT noted, no rework required.
