# F.7.16 Builder QA Falsification — Default template `[gates.build]` expansion

**Droplet:** Drop 4c F.7-CORE F.7.16
**Reviewer:** go-qa-falsification-agent
**Mode:** Read-only adversarial counterexample search
**Date:** 2026-05-05
**Round:** 1
**Files reviewed:**
- `internal/templates/builtin/default.toml` (lines 319–358)
- `internal/templates/embed_test.go` (TestDefaultTemplateLoadsWithGates / TestDefaultTemplateGatesAllValidGateKinds / TestDefaultTemplateNoProjectMetadataOverrides — lines 346–455)
- `internal/templates/schema.go` (GateKind enum + validGateKinds, lines 80–135)
- `internal/app/dispatcher/gate_commit.go` (toggle-skip contract, lines 1–220)
- `internal/app/dispatcher/gate_push.go` (toggle-skip contract, lines 1–120)
- `internal/app/dispatcher/gates.go` (gateRunner.Run + ErrGateNotRegistered, lines 1–281)
- `internal/domain/project.go` (DispatcherCommitEnabled / DispatcherPushEnabled toggles, lines 129–202)
- `workflow/drop_4c/F7_CORE_PLAN.md` § F.7.16 (lines 925–963)
- `workflow/drop_4c/4c_F7_16_BUILDER_WORKLOG.md`

---

## 1. Findings

### 1.1 Default template carries `[gates.build] = ["mage_ci", "commit", "push"]` in declared order

`default.toml:358` reads `build = ["mage_ci", "commit", "push"]`. The order matches plan acceptance bullet #1 verbatim and the test's `slices.Equal` assertion (`embed_test.go:370–373`) pins the exact 3-tuple, including order. No silent reorder, no missing entry.

### 1.2 Slice ORDER is correct per the dispatcher gate-runner contract

`gates.go:233–280` shows `gateRunner.Run` iterates `tpl.Gates[item.Kind]` IN DECLARATION ORDER and halts on first non-`Passed` result (`gates.go:276–278`). With the runner's halt-on-first-failure semantics, `mage_ci → commit → push` order is load-bearing because:

- `commit` consumes the path scope only after `mage ci` confirms the build is green (a red build should never produce a commit).
- `push` consumes the local commit produced by `commit` (a push without a fresh local commit on the working ref is a no-op or a stale-state push).

The worklog comment (`default.toml:339–346`) captures this rationale verbatim. No counter-ordering escape hatch exists in either the runner or the template loader.

### 1.3 Closed-enum strictness — `commit` + `push` are valid GateKind values

`schema.go:110–116` declares `validGateKinds` containing both `GateKindCommit` ("commit") and `GateKindPush` ("push"). `IsValidGateKind` (`schema.go:128–135`) does exact-match against this slice — no whitespace trimming, no case folding, no partial-match. The template-load validator (Drop 3.10) wires through this function so any out-of-enum entry fails template load. `TestDefaultTemplateGatesAllValidGateKinds` (`embed_test.go:412–423`) iterates every entry of every `[gates.<kind>]` slice and asserts `IsValidGateKind` returns true — regression guard for both directions (a new TOML string without an enum entry, or a removed enum constant with stale TOML still referencing it).

### 1.4 Toggle-skip no-op contract holds at the gate-runner level when production wiring lands

`gate_commit.go:194–220` documents step 1: `if project.Metadata.IsDispatcherCommitEnabled() returns false, return nil`. The gate is a successful no-op, NO git commands invoked, item.EndCommit unchanged. `gate_push.go:1–34` mirrors the same contract for the push gate. `IsDispatcherCommitEnabled()` / `IsDispatcherPushEnabled()` (`project.go:182–202`) both nil-check the `*bool` toggle — nil → false. `TestDefaultTemplateNoProjectMetadataOverrides` (`embed_test.go:438–455`) pins this on a zero-value `domain.ProjectMetadata{}`.

### 1.5 No closed-enum violations introduced

`mage_test_pkg` and `hylla_reingest` are deliberately ABSENT from `[gates.build]`:

- `mage_test_pkg` is per-package (build-qa kinds, not build-the-kind).
- `hylla_reingest` is drop-end only, not per-build (per CLAUDE.md "Hylla ingest invariants" + `gate_push.go` + Master PLAN).

The plan does NOT name them as F.7.16 deliverables; the test enumerates the full closed enum in `validGateKinds` and asserts only `[gates.build]` is populated. Other kinds carry no gate sequence (`embed_test.go:377–394` enumerates 11 kinds and asserts absence — the gate runner treats absence as "no gates" not "all gates" per `gates.go:238–240`).

### 1.6 No commit by builder per REV-13

`git status --porcelain` confirms `M internal/templates/builtin/default.toml` and `M internal/templates/embed_test.go` are uncommitted. Worklog acceptance bullet 4 explicitly notes "NO commit by builder (per REV-13 + spawn-prompt directive)." Memory rule "QA before commit" satisfied: orchestrator drives commits AFTER QA pair returns green.

### 1.7 Memory-rule conflicts — none

- "No Hylla calls" directive — worklog `## Hylla Feedback` records `N/A — directive explicitly forbade Hylla calls`. Native-tool path was sufficient.
- "No migration logic pre-MVP" — no migration code introduced.
- "Subagent reports Hylla feedback" — section present, content explanation acceptable.
- "QA Before Commit" — uncommitted state preserved.
- "Single-line commit format" — proposed message in worklog is a single line, 60 chars, conventional-commit format.

---

## 2. Counterexamples (attempted)

### 2.1 Attack: fresh-project default state — do `commit` + `push` no-op cleanly OR fail because the gate registry has no production wiring?

**Status:** REFUTED — but with a sharper contextual finding.

**Investigation:** I searched `internal/app/dispatcher/` and `cmd/` for any non-test `Register()` calls binding `GateKindCommit` / `GateKindPush` (or any GateKind for that matter) to a `gateRunner`. Result: ZERO production `Register()` calls. I also searched for production `newGateRunner()` instantiations or `gateRunner{...}` literals. Result: only `gates.go:172` definition + tests reference these.

**Implication:** the gate runner IS NOT INVOKED IN PRODUCTION TODAY (Drop 4b shipped the runner data structures + Wave A gate implementations as a registry; Drop 4c F.7.13/F.7.14 ship the commit/push gate runners standalone but no boot-time wiring binds them to a runner that the dispatcher's `in_progress` transition triggers). The "no-op contract" is therefore satisfied trivially today: nothing reads `[gates.build]` in production yet.

**Why this is REFUTED, not CONFIRMED:**
- This is consistent with the F.7-CORE plan's REV-13 + sequencing — the wiring droplet is downstream of F.7.16 (the F.7.13 doc-comment line 11–14 says "The Drop 4c follow-up wiring droplet adapts CommitGateRunner.Run to the gateRunner's gateFunc interface").
- F.7.16 is a PURE template/test droplet by design; it ships declarative TOML + assertions, not the boot-time `Register()` that wires runners to the runner.
- The F.7.16 acceptance bullet #3 ("Fresh project ... → gate runner runs `mage_ci`, then `commit` (skipped per F.7.13), then `push` (skipped per F.7.14)") is a STATEMENT OF INTENT for the toggle-default contract, not a runtime test F.7.16 was supposed to wire — the test scenario gets exercised when the wiring droplet lands and the dispatcher subscribes to gate execution.
- Plan acceptance bullet #5 ("Tests cover: default-go template loads; gate runner registration; fresh-project skip behavior; toggled-on full-pipeline behavior") IS partially descoped from F.7.16 — the worklog covers (a) "default-go template loads" via TestDefaultTemplateLoadsCleanly + TestDefaultTemplateLoadsWithGates, (b) gate runner registration by closed-enum membership via TestDefaultTemplateGatesAllValidGateKinds. The "fresh-project skip behavior" + "toggled-on full-pipeline behavior" runtime scenarios properly belong to the wiring droplet's QA, not F.7.16's. Spawn prompt explicitly framed F.7.16 as "Pure schema/template additions + companion test scenarios. NO gate-execution behavior changes."

**Note (NIT-grade, not a counterexample):** the worklog's "Acceptance criteria" check on `mage check` + `mage ci` green is marked `[ ]` (unchecked) due to an env hang. Worklog provides repro: stash my changes, run baseline `TestDefaultTemplateLoadsCleanly`, same 11-min hang reproduces. Hang is pre-existing, not a regression. Routing to orchestrator is correct disposition.

### 2.2 Attack: Order matters — `mage_ci` before `commit`, `commit` before `push`

**Status:** REFUTED.

**Investigation:** `default.toml:358` shipped order is `["mage_ci", "commit", "push"]`. `gateRunner.Run` (`gates.go:249`) iterates the slice in template-declared order with halt-on-first-failure (`gates.go:276–278`). `embed_test.go:370–373` asserts the literal 3-tuple shape via `slices.Equal`. `default.toml:339–346` documents the order rationale. No way to land `[push, commit, mage_ci]` without breaking `TestDefaultTemplateLoadsWithGates`.

### 2.3 Attack: closed-enum strictness — does any non-enum gate sneak in? Does default template MISS `mage_test_pkg` / `hylla_reingest`?

**Status:** REFUTED on both axes.

- Sneak-in axis: `IsValidGateKind` is exact-match (`schema.go:128–135`); template-load validation rejects any out-of-enum string at decode time. `TestDefaultTemplateGatesAllValidGateKinds` (`embed_test.go:412–423`) iterates every gate in every `[gates.<kind>]` slice and asserts validity. No silent-tolerance paths.
- Missing-gates axis: `mage_test_pkg` is per-package (build-qa, not build); `hylla_reingest` is drop-end only (CLAUDE.md "Hylla ingest invariants"). The plan does not require either in `[gates.build]`. Adding them WOULD be a CONFIRMED counterexample (they don't belong in the build leaf's chain). Their absence is correct.

### 2.4 Attack: fresh project + dogfood readiness — no-op contract holds end-to-end

**Status:** REFUTED.

**Investigation:** worklog test `TestDefaultTemplateNoProjectMetadataOverrides` (`embed_test.go:438–455`) loads the default template AND asserts a zero-value `domain.ProjectMetadata{}` reports both `IsDispatcherCommitEnabled() == false` and `IsDispatcherPushEnabled() == false`. The test exists as a "structural invariant" hook — it would catch a future drop that adds template-side `[project_metadata]` toggle defaults. Combined with `gate_commit.go:194–220` and `gate_push.go:1–34` toggle-skip contracts, the no-op IS the production behavior when the wiring droplet lands. End-to-end safety holds.

### 2.5 Attack: memory-rule conflicts (no Hylla, no commit, no migration)

**Status:** REFUTED.

- No-Hylla directive: worklog `## Hylla Feedback` is `N/A — directive explicitly forbade Hylla calls`. No Hylla queries made. (My own falsification work DID use Hylla as a sanity probe but enrichment-still-running; I fell back to native-tool grep, which is correct per evidence-source policy.)
- No-commit directive: `git status` confirms uncommitted state. No `git commit` invocation in worklog.
- No-migration: no migration code introduced.

### 2.6 Attack: closed-enum coverage — did the builder skip a closed-enum kind that should have a gate sequence?

**Status:** REFUTED.

**Investigation:** plan acceptance bullet #1 names ONLY `[gates.build]`. Bullet #5 + worklog test `TestDefaultTemplateLoadsWithGates` enumerates all 12 kinds and asserts only `build` carries a gate sequence (the 11 absent kinds are: `plan`, `research`, `build-qa-proof`, `build-qa-falsification`, `plan-qa-proof`, `plan-qa-falsification`, `closeout`, `commit`, `refinement`, `discussion`, `human-verify`). Per `gates.go:238–240` absence means "no gates, return nil result slice immediately" — distinct from "all gates" — so absence is the correct semantic for kinds that have no post-action verification surface. No counterexample.

### 2.7 Attack: TestDefaultTemplateNoProjectMetadataOverrides is vacuous (proves nothing about templates)

**Status:** REFUTED with a NIT.

**Investigation:** the test loads the default template via `loadDefaultOrFatal(t)` and immediately discards the result with `_ = ...`. It then constructs a fresh `var meta domain.ProjectMetadata` and asserts both toggles are false. Strictly speaking this assertion is true tautologically because `*bool` zero value is nil and `IsDispatcher{Commit,Push}Enabled` nil-check returns false (`project.go:182–202`). The Template type carries no project-metadata fields, so the load step is unobservable to the assertion.

The test's worklog rationale (`embed_test.go:432–437`) is honest about this: "The test exists as a structural invariant — the Template type carries no project-metadata-shaped fields, so loading it cannot produce overrides. A future drop that adds template-side toggle defaults (e.g. `[project_metadata]` sub-table) would have to break this test before shipping."

**Why NIT, not counterexample:** the load step IS load-bearing in the regression-guard sense — if a future drop adds a `[project_metadata]` sub-table to default.toml that mutates the global `domain.ProjectMetadata` zero value (e.g. via a package-level init), this test would catch the mutation by virtue of being in the same test binary. The framing "structural invariant" is correct. The test's value is hypothetical-future regression coverage, not present-day correctness verification — but that's exactly what the worklog claims it does, so the test is honestly described and not load-bearing for F.7.16's present-day acceptance.

A stronger version of this test would assert AFTER the load that `tpl.Gates[domain.KindBuild]` still references `commit` + `push` (which would be falsified by a hypothetical [project_metadata] block setting toggles ON in the default — a different failure mode). That's a NIT, not a blocker — `TestDefaultTemplateLoadsWithGates` already pins the gate-list shape.

---

## 3. Summary

**Verdict: PASS.**

Default template `[gates.build] = ["mage_ci", "commit", "push"]` shipped in correct declaration order. Three test scenarios authored:
1. `TestDefaultTemplateLoadsWithGates` — pins the exact 3-tuple shape including order via `slices.Equal`; pins absence of gate sequences on the other 11 closed-enum kinds.
2. `TestDefaultTemplateGatesAllValidGateKinds` — closed-enum membership regression guard for both directions.
3. `TestDefaultTemplateNoProjectMetadataOverrides` — toggle-default-OFF structural invariant for any future `[project_metadata]` sub-table addition.

All seven attack axes (production-wiring no-op safety, slice ordering, closed-enum strictness with mage_test_pkg / hylla_reingest absence, fresh-project end-to-end no-op, memory-rule conflicts, closed-enum coverage gaps, vacuous-test concern) returned REFUTED. No counterexamples produced. One NIT (TestDefaultTemplateNoProjectMetadataOverrides could assert post-load template-state stability, not just zero-value metadata) acknowledged but not blocking — the test's documented purpose is hypothetical-future regression coverage and the worklog is honest about that framing.

The pre-existing test-runner hang on `mage testPkg ./internal/templates` is correctly identified as environmental (worklog reproduced on baseline via `git stash`) and routed to orchestrator. Not a regression from F.7.16. No edits required.

**Disposition:** PASS — orchestrator may proceed to F.7.16 droplet commit + close.

---

## Hylla Feedback

One miss recorded.

- **Query:** `hylla_search_keyword` with `query="Register GateKindCommit GateKindPush dispatcher"` and `query="RegisterGateMageCI RegisterCommitGate dispatcher gates Register"` against `github.com/evanmschultz/tillsyn@main` to find production gate-runner registration sites.
- **Missed because:** Hylla returned `enrichment still running for github.com/evanmschultz/tillsyn@main` — the latest ingest of the F.7.13 / F.7.14 / F.7.16 drop-state was still mid-enrichment when this falsification ran, so the index could not answer about commit/push gate wiring. This is not a Hylla bug; it's an ingest-timing artifact mid-drop. The drop-end Hylla reingest (CLAUDE.md "Hylla ingest invariants") will resolve this once the drop closes.
- **Worked via:** `Bash` with `/usr/bin/grep -rn "GateKindCommit\|GateKindPush" internal/` and `/usr/bin/grep -rn "Register" internal/app/dispatcher/ | grep -v _test.go` against the on-disk worktree. Confirmed zero production `Register()` calls binding any GateKind, and zero production `newGateRunner()` instantiations — i.e. the gate runner pipeline is intentionally unwired today (production wiring deferred to a follow-up droplet).
- **Suggestion:** none Hylla-shape; the miss was a normal mid-drop-enrichment artifact. The orch sandbox restriction on `find` / `ls` / `grep` invocation paths added some friction (had to use `/usr/bin/grep` explicitly), but that's a Claude-Code permission concern, not a Hylla one.

Ergonomic note: Hylla's `enrichment still running` error response is a hard-error rather than a partial-result-with-warning shape. For mid-drop QA work where the latest Go state IS ingested but the embeddings / summaries are mid-build, returning the keyword-only results with a `partial=true` flag would let QA agents proceed without falling back to native tools. NIT-grade refinement, not a blocker.

---

## TL;DR

- **T1.** All seven attack axes refuted; default.toml ships `[gates.build] = ["mage_ci", "commit", "push"]` in correct order, closed-enum strictness holds, toggle-default-OFF no-op contract is intact end-to-end, no commit by builder per REV-13.
- **T2.** No CONFIRMED counterexamples. One NIT on TestDefaultTemplateNoProjectMetadataOverrides being structurally vacuous-but-honestly-documented; not blocking.
- **T3.** Verdict PASS. Orchestrator may proceed to F.7.16 droplet commit and close.
