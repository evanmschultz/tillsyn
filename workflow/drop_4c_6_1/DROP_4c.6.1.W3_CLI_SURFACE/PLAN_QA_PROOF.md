# PLAN_QA_PROOF — DROP_4c.6.1.W3_CLI_SURFACE (Round 1)

**Verdict:** FAIL (1 FF, 6 NITs). PLAN structure, blocker graph, droplet count, R3-NIT6 verbatim ContextBlock, and CONSUMER-TIE contract are clean; one cross-planner contract gap (L1 demands `--add-group/--remove-group` flags that no upstream wave ships the schema for) plus 6 NITs (verbatim period, smart-default coverage, fan-out test enumeration, missing-pass-through test coverage, service.go line-number drift, AC8 missing role-required mapping) must be resolved before build dispatch.

**Scope:** L2 plan for Wave W3 (CLI Surface) — 15 new CLI subcommands + `till agents bootstrap`, fully serial inside `cmd/till` package. D1→D2→D3→D4→D5→D6→D7.

**Mode:** Filesystem-MD-only. NO Tillsyn runtime. NO Hylla calls. Evidence from `Read` + `rg` against `internal/domain/`, `internal/app/service.go`, L1 PLAN.md, REVISION_BRIEF.md, W2 L2 PLAN.md.

---

## 0. Verification Checklist Summary

| Check | Result |
|---|---|
| 1. Every claim backed by evidence | PASS |
| 2. Every acceptance bullet testable | PASS (with NIT-4) |
| 3. Trace covers all acceptance bullets | PASS (D1–D7 maps 1:1 to scope) |
| 4. `blocked_by` graph acyclic + shared-file pairs gated | PASS (linear D1→…→D7 mirrored in `_BLOCKERS.toml`) |
| 5. PLAN-QA-DISCIPLINE-R1 (NEW behavior → test-runner blocked_by) | PARTIAL (see NIT-2, NIT-4) |
| 6. PLAN-QA-DISCIPLINE-R2 (narrative count = enumerated count) | PASS (7 narrative droplets, 7 enumerated D1–D7) |
| 7. `_BLOCKERS.toml` mirrors PLAN.md | PASS |
| 8. CONSUMER-TIE contract documented per droplet | PASS (ContextBlock lines 89–92) |
| 9. R3-NIT6 `--force` warning VERBATIM per L1 line 317 | NIT-1 (closing-period drift) |
| 10. FF4 smart-default mapping (plan/refinement→segment; other 10→droplet) | PASS in ContextBlock; NIT-2 on test coverage |

---

## 1. Findings (FF — block build dispatch)

### 1.1 [Axis: shipped-but-not-wired] [severity: high] L1 contract demands `--add-group`/`--remove-group` flags on `till project update` (W3.D1) but no upstream wave ships the schema; W3 L2 plan silently drops them

**Claim.** L1 PLAN.md line 282 declares `till project update` MUST accept `[--add-group <name>] [--remove-group <name>]` flags. W3 L2 PLAN.md AC1 (line 36), the W3.D1 acceptance bullets (lines 198–208), and the W3.D1 `KindPayload` struct shape_hint (line 244) all OMIT these two flags. The only mention is a hedged RiskNote (line 211) that says "if absent [from `ProjectMetadata`], omit these flags and add a TODO comment for the future."

**Evidence.**

- L1 PLAN.md line 282: `update --project-id <id> [--root-path ...] [--bare-root ...] [--language ...] [--add-group <name>] [--remove-group <name>] [--hylla-artifact-ref ...] [--description ...]` — Per §2.8.
- L1 REVISION_BRIEF.md §2.8 line 105: `Flags: --root-path, --bare-root, --language, --add-group <name>, --remove-group <name>, --hylla-artifact-ref, --description, --owner, --homepage, --icon, --color, --tag.`
- W3 L2 PLAN.md line 36 (AC1): no `--add-group`/`--remove-group`.
- W3 L2 PLAN.md line 24 (Scope recap): omits `--add-group`/`--remove-group`.
- W3 L2 PLAN.md line 244 (D1 `projectUpdateCommandOptions` shape_hint): omits `addGroup`/`removeGroup`.
- `internal/domain/project.go:119` (`ProjectMetadata` struct): NO `Groups []string` field. Fields are: `Owner`, `Icon`, `Color`, `Homepage`, `Tags`, `StandardsMarkdown`, `KindPayload`, `CapabilityPolicy`, `OrchSelfApprovalEnabled`, `DispatcherCommitEnabled`, `DispatcherPushEnabled`.
- W2 L2 PLAN.md line 322: "do NOT add `Groups []string` to `internal/domain/project.go` in D7. That is a [separate concern — deferred]."
- W2 L2 PLAN.md line 318: "Adding a typed `Groups []string` field to `ProjectMetadata` is an [issue]."
- W2 L2 PLAN.md line 326: "KindPayload JSON for groups is a stopgap. Future drop should add typed `Groups`."

**Trace.** L1 contract → W2 defers schema addition + stores groups in `Metadata.KindPayload` JSON stopgap → W3 inherits the unshipped schema → W3.D1 silently drops the two flags rather than (a) escalating to dev, (b) implementing via the KindPayload-JSON stopgap pattern W2 uses, or (c) adding a typed `ProjectMetadata.Groups` field as part of W3's scope. Wave D is the LAST wave; if W3 ships without these flags, the L1 acceptance contract for §2.8 is not met.

**Conclusion.** The L1 contract is broken. Either:
- **Option A**: explicitly add `--add-group`/`--remove-group` to W3.D1's acceptance + `KindPayload` shape_hint, with the implementation reading existing groups from `Metadata.KindPayload` JSON (the same stopgap W2 uses), mutating the slice, and re-encoding into `KindPayload`. This keeps the L1 contract intact without modifying `internal/domain/project.go`.
- **Option B**: explicitly DROP `--add-group`/`--remove-group` from the L1 contract via a documented disposition (FF1 or similar) and update REVISION_BRIEF.md §2.8 + L1 PLAN.md line 282 accordingly. Surface as a refinement (e.g., `GROUPS-CLI-R1` — landed when typed `ProjectMetadata.Groups` ships).
- **Option C**: extend W3's paths to include `internal/domain/project.go` + `internal/app/service.go` (`UpdateProjectInput`) and ship the typed field as a W3 add-on. Largest scope expansion; least clean.

**Fix hint.** Option B with explicit FF disposition is the cleanest. The W3 L2 plan should either escalate to dev for FF disposition or implement via KindPayload JSON stopgap (Option A) and document it in D1 RiskNotes.

---

## 2. NITs (must address before build dispatch; absorb inline per nits-are-first-class)

### 2.1 [Axis: spec-conformance] [severity: low] R3-NIT6 verbatim warning has a closing-period drift from L1 PLAN.md line 317

**Claim.** L1 PLAN.md line 317 quotes the R3-NIT6 warning ending with `\`--force\`"` (close-quote, NO period inside). W3 L2 PLAN.md AC8 (line 43), D6 acceptance bullet (line 568), and D6 ContextBlock (line 586) all quote it ending with `\`--force\`."` (close-quote WITH period inside). The task brief explicitly says "VERBATIM from L1 PLAN.md line 317."

**Evidence.**

- L1 PLAN.md line 317: `…before re-running bootstrap with \`--force\`" — documented so users…` (no period inside the quoted block; the trailing dash is L1's prose continuation).
- W3 L2 PLAN.md line 43 (AC8): `…before re-running bootstrap with \`--force\`."`
- W3 L2 PLAN.md line 568 (D6 acceptance bullet): same `…with \`--force\`."`
- W3 L2 PLAN.md line 586 (D6 ContextBlock): same.
- W3 L2 PLAN.md line 627 (D6 `TestRunAgentsBootstrap_ForceHelpTextContainsWarning`): "verifies --force cobra flag help text contains the R3-NIT6 verbatim warning string" — the build-QA test will compare against whichever string the W3 plan declares; if the plan's declared string has a closing period and the test uses contains-match, the test passes either way, but the spec is technically drifted.

**Fix hint.** Either (a) remove the closing period from all three W3 PLAN.md occurrences (lines 43, 568, 586) to match L1 line 317 byte-for-byte, OR (b) leave the period and add a one-line dispositioning note: "Closing period intentional — grammatical close of the help-text sentence. L1 line 317's no-period form is mid-sentence in L1's prose; the standalone help-text form takes a period." Either is fine; the current state is ambiguous.

### 2.2 [Axis: acceptance-criteria-coverage] [severity: medium] D3 smart-default acceptance lists 12 kinds but D3's `TestRunActionItemCreate_StructuralTypeSmartDefault` table-test enumeration only covers 4 (plan, build, refinement, research)

**Claim.** D3 acceptance bullets (lines 337–341) state per-kind smart-default behavior for ALL 12 kinds (`plan`+`refinement` → `segment`; `build`+9 others → `droplet`). D3 `KindPayload` test shape_hint (line 393) only enumerates: `plan→segment, build→droplet, refinement→segment, research→droplet, explicit-override-valid, explicit-override-invalid`. Eight kinds (plan-qa-proof, plan-qa-falsification, build-qa-proof, build-qa-falsification, closeout, commit, discussion, human-verify) are stated as defaulting to `droplet` but have no explicit table-test row.

**Evidence.**

- W3 L2 PLAN.md line 340: "All other 9 kinds (research, plan-qa-proof, plan-qa-falsification, build-qa-proof, build-qa-falsification, closeout, commit, discussion, human-verify) default to `StructuralType=droplet`."
- W3 L2 PLAN.md line 393: `"table test: plan→segment, build→droplet, refinement→segment, research→droplet, explicit-override-valid, explicit-override-invalid"` — 4 kinds + 2 override cases.
- PLAN-QA-DISCIPLINE-R1: every acceptance bullet asserting NEW behavior has matching test-runner.

**Fix hint.** Update D3 `KindPayload` shape_hint to enumerate all 12 kinds in the table test: a single table-driven test that iterates over every member of the closed `domain.Kind` enum and asserts the expected `StructuralType` default. The implementation cost is one extra `for _, kind := range allKinds` loop; the QA coverage benefit is full FF4 mapping coverage. Add explicit shape_hint: `"table test covering ALL 12 kinds × {explicit-override-valid, explicit-override-invalid, default} = 36 rows"` (or similar — builder's call on exact shape).

### 2.3 [Axis: acceptance-criteria-coverage] [severity: low] D6 `TestRunAgentsBootstrap_QAFanOut` shape_hint missing — does not enumerate all 4 destination files (plan-qa-proof, build-qa-proof, plan-qa-falsification, build-qa-falsification)

**Claim.** D6 acceptance (lines 562–563) declares 2-into-4 fan-out applies to BOTH `qa-proof` AND `qa-falsification` → 4 destination files per group. D6 `KindPayload` entry for `TestRunAgentsBootstrap_QAFanOut` (line 622) has NO shape_hint. The acceptance bullet at line 571 says "verify both plan-qa-proof and build-qa-proof written" — only 2 destinations, falsification dropped.

**Evidence.**

- W3 L2 PLAN.md line 562: "go-qa-proof-agent.md in source produces BOTH go/plan-qa-proof-agent.md AND go/build-qa-proof-agent.md at dest with identical content."
- W3 L2 PLAN.md line 563: "2-into-4 fan-out applies equally to qa-falsification."
- W3 L2 PLAN.md line 571: "Tests cover: … 2-into-4 fan-out (verify both plan-qa-proof and build-qa-proof written)" — falsification destinations not mentioned.
- W3 L2 PLAN.md line 622: `{"file": "cmd/till/agents_cli_test.go", "symbol": "TestRunAgentsBootstrap_QAFanOut", "action": "add"}` — no shape_hint.

**Fix hint.** Add shape_hint to D6 `TestRunAgentsBootstrap_QAFanOut`: `"verifies all 4 fan-out destination files written with identical-to-source content: <group>/plan-qa-proof-agent.md, <group>/build-qa-proof-agent.md, <group>/plan-qa-falsification-agent.md, <group>/build-qa-falsification-agent.md"`. Update line 571 to enumerate all 4 destinations.

### 2.4 [Axis: acceptance-criteria-coverage] [severity: low] D3 acceptance lists `--paths`, `--packages`, `--files`, `--blocked-by`, `--metadata-json`, `--parent-id`, `--role` as "accepted and passed through" with no matching test-runner

**Claim.** D3 acceptance bullet line 343: "`--paths`, `--packages`, `--files`, `--blocked-by`, `--metadata-json`, `--parent-id`, `--role` flags are accepted and passed through." No D3 `KindPayload` test entry covers these flags. The two listed tests are `TestRunActionItemCreate_StructuralTypeSmartDefault` (smart-default only) and `TestRunActionItemCreate_RequiredFields` (missing-flag errors only). Pass-through behavior is untested.

**Evidence.**

- W3 L2 PLAN.md line 343: pass-through flags list.
- W3 L2 PLAN.md lines 393–395: only 2 test families listed.
- W3 L2 PLAN.md line 211 RiskNote: `BlockedBy` requires verification — if absent from `CreateActionItemInput`, must call `UpdateActionItem` post-create. Verified absent from `CreateActionItemInput` (`internal/app/service.go:737-831` — confirmed no `BlockedBy` field on the struct); flag pass-through requires the post-create `UpdateActionItem` round-trip, which is non-trivial wiring that deserves a test.

**Fix hint.** Add a third test family to D3 `KindPayload`: `TestRunActionItemCreate_PassThroughFlags` — verifies that each of the 7 pass-through flags (paths, packages, files, blocked-by, metadata-json, parent-id, role) makes it onto the created `domain.ActionItem` (round-trip via `ListActionItems` or direct repo read). Especially important for `--blocked-by` since it requires a post-create `UpdateActionItem` call.

### 2.5 [Axis: spec-conformance] [severity: low] Service.go line numbers cited in ContextBlock + RiskNotes are off by 1 vs verified Read positions

**Claim.** W3 PLAN ContextBlock at lines 95–102 cites: `UpdateProject (line 625)`, `ArchiveProject (line 669)`, `RestoreProject (line 689)`, `DeleteProject (line 709)`, `CreateActionItem (line 1035)`. Verified via Read: actual line numbers are `624`, `668`, `688`, `708`, `1034` — all off by 1. Same for D2 ContextBlock lines 288–292: same +1 drift.

**Evidence.**

- `internal/app/service.go:624`: `func (s *Service) UpdateProject(ctx context.Context, in UpdateProjectInput) (domain.Project, error) {` (W3 PLAN claims 625).
- `internal/app/service.go:668`: `// ArchiveProject archives one project.` (PLAN claims line 669 for the func).
- `internal/app/service.go:669`: `func (s *Service) ArchiveProject(...)`.
- Similar +1 drifts at 689/688, 709/708, 1035/1034.

**Fix hint.** Update PLAN.md ContextBlock + D2 ContextBlock to use the verified line numbers. Low-severity because the symbols exist and LSP `goToDefinition` will resolve them regardless of whether the comment lines are off by one; the drift is cosmetic but easy to fix.

### 2.6 [Axis: spec-conformance] [severity: low] D3 says "`--role` is OPTIONAL (closed enum; empty is valid pre-Drop-2)" but `CreateActionItemInput.Role` semantics actually allow empty post-Drop-2 too

**Claim.** D3 RiskNote line 349 says "`--role` is optional (closed enum; empty is valid). If non-empty, pass as `domain.Role(opts.role)`." The L1 PLAN at line 295 says "`--role` is OPTIONAL (closed enum; empty is valid pre-Drop-2). Builder L2 confirms the closed-enum list via LSP `goToDefinition` on `domain.Role`." The phrasing "pre-Drop-2" implies post-Drop-2 the empty-role policy might tighten — but per `internal/app/service.go:742-746` the role doc-comment says "Empty string is permitted; non-empty values must match the closed Role enum" with no pre/post-Drop-2 qualifier.

**Evidence.**

- `internal/app/service.go:742-746`: `// Role optionally tags the action item with a closed-enum role (e.g. // builder, qa-proof, planner). Empty string is permitted; non-empty // values must match the closed Role enum or domain.NewActionItem returns // ErrInvalidRole.`
- L1 PLAN.md line 295: "`--role` is OPTIONAL (closed enum; empty is valid pre-Drop-2)."

**Fix hint.** Either drop the "pre-Drop-2" qualifier in L1 line 295 + propagate to W3 D3 RiskNote, or surface a tracked refinement (`ROLE-POLICY-R1`) if the dev intends to tighten the policy. Low-severity because today's behavior is "empty is valid" unconditionally and the plan doesn't depend on the qualifier being accurate.

---

## 3. Verified Strengths (no action needed; documented for audit)

- **Service-method existence + signatures.** All 5 service methods cited (UpdateProject, ArchiveProject, RestoreProject, DeleteProject, CreateActionItem) exist with the documented signatures; line numbers cosmetically off by 1 (NIT-5) but symbols verified.
- **No-RenameProject disposition (R4).** Correctly identified — `(*Service).RenameProject` does not exist; W3.D2's `till project rename` correctly routes through `(*Service).UpdateProject` → `domain.Project.UpdateDetails` → `p.Name = name + p.Slug = normalizeSlug(name)` (`internal/domain/project.go:337-338`). This is the canonical rename path.
- **StructuralType closed enum.** Verified at `internal/domain/structural_type.go:18-22` — `drop|segment|confluence|droplet`. Matches the FF4 override-validation list.
- **Role closed enum.** Verified at `internal/domain/role.go:14-23` — 9 values (builder, qa-proof, qa-falsification, qa-a11y, qa-visual, design, commit, planner, research). Matches PLAN ContextBlock claim.
- **`CreateActionItemInput.StructuralType` mandatory.** Verified at `internal/app/service.go:752` — doc-comment says "Empty is REJECTED on create — `domain.NewActionItem` returns `ErrInvalidStructuralType`." Matches PLAN R5 and FF4 smart-default rationale.
- **`BlockedBy` not on `CreateActionItemInput`.** Verified — `CreateActionItemInput` ends at `UpdatedByType` (line 831); `BlockedBy` lives on `ActionItemMetadata` (`internal/domain/workitem.go:195`). D3 RiskNote correctly flags this for builder verification.
- **`blocked_by` graph acyclic.** Linear chain D1→D2→D3→D4→D5→D6→D7. `_BLOCKERS.toml` mirrors PLAN.md verbatim. No cycle, no missing same-file pair.
- **Narrative count = enumerated count.** "7 droplets (D1–D7)" matches 7 explicitly enumerated droplets. PLAN-QA-DISCIPLINE-R2 PASS.
- **CONSUMER-TIE contract documented.** ContextBlock at lines 89–92 cleanly distinguishes `runXxx(ctx, svc, opts, stdout)` per-subcommand handler from top-level `run(ctx, args, &out, io.Discard)` test entrypoint inherited from Drop 4c.6 W2.
- **Atomic-droplet sizing risk flagged.** R1 (D2 ≤120 LOC), R2 (D4 ≤120 LOC), R3 (D6 ≤120 LOC), with explicit split-paths (D2a/D2b, D4a/D4b, D6a/D6b) if exceeded.
- **R3-NIT6 ContextBlock present.** Lines 582–587 mark the warning as `severity=critical` for build-QA enforcement, modulo the closing-period drift (NIT-1).
- **Cross-platform diff hint.** D4 ContextBlock line 451 explicitly warns against `exec` to external `diff` binary — Windows-safe.
- **Test isolation discipline.** D4/D5/D6 all specify `os.MkdirTemp` (or `testing/fstest.MapFS`) for HOME-dir isolation; no real `~/.tillsyn` touched in tests.

---

## 4. Cross-Planner Observations

- **W3 is Wave D (last wave).** Confirmed in L1 PLAN.md line 834. No downstream wave can absorb fixes; FF-1 must be resolved within W3 or via L1 disposition update.
- **W2 deferred `ProjectMetadata.Groups` typed field** (W2 L2 PLAN.md line 322). W3 inherits the gap; this is the proximate cause of FF-1.
- **W7.D3 also touches `cmd/till/main.go`.** L1 PLAN.md line 844 documents W7.D3 → W2 blocker for `cmd/till` package serialization. W3 is also `cmd/till`-package — but W3 is Wave D (after W7.D3 in Wave C), so W3's `main.go` mods (D7) come AFTER W7.D3's deletion of `till serve`. D7 should not encounter `till serve` cobra wiring; verified safe.
- **D6 known-groups list `{"go", "fe", "gen"}`** is consistent with W2's `allowedInitGroups = []string{"gen", "go", "fe"}` (W2 L2 PLAN.md line 54). Cross-wave consistency PASS.
- **D7 main.go registration depends on every prior D1-D6 having shipped its `runXxx` functions.** Linear chain enforces this; no FF here. PLAN-QA-DISCIPLINE-R1 satisfied for D7 (acceptance via `till --help` smoke test in `TestW3CommandsRegistered`).

---

## 5. Decision

**Verdict: FAIL** — 1 FF + 6 NITs. The FF (`--add-group`/`--remove-group` contract gap) is load-bearing because W3 is the last wave; the L1 §2.8 contract cannot be silently broken. Dev disposition needed before W3 builds.

**Next steps:**

1. Dev resolves FF-1 with Option A (KindPayload-JSON stopgap), Option B (drop flags + refinement), or Option C (extend W3 to add typed schema). My recommendation: Option B with `GROUPS-CLI-R1` refinement, since adding/removing groups is a niche workflow and the typed field belongs in a separate `internal/domain/` drop.
2. Absorb NITs 1–6 inline in next round of W3 PLAN.md (single edit pass).
3. Re-run plan-QA proof + falsification pair.

## Hylla Feedback

N/A — action item touched non-Go files only (PLAN.md, REVISION_BRIEF.md, _BLOCKERS.toml). Per task brief: "Hylla is OFF". All Go-symbol verification went through `Read` against `internal/domain/structural_type.go`, `internal/domain/role.go`, `internal/domain/project.go`, `internal/domain/workitem.go`, `internal/app/service.go`. No Hylla fallback to report.
