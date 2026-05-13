# PLAN-QA FALSIFICATION — Drop 4c.6.1.W3 (CLI_SURFACE) — Round 2

**Reviewer:** L2 plan-QA falsification agent (round 2)
**Date:** 2026-05-12
**Inputs reviewed:**
- `workflow/drop_4c_6_1/DROP_4c.6.1.W3_CLI_SURFACE/PLAN.md` + `_BLOCKERS.toml` (round-2 absorption)
- `workflow/drop_4c_6_1/DROP_4c.6.1.W3_CLI_SURFACE/PLAN_QA_PROOF.md` (round-1 proof FAIL — 1 FF + 6 NITs)
- `workflow/drop_4c_6_1/DROP_4c.6.1.W3_CLI_SURFACE/PLAN_QA_FALSIFICATION.md` (round-1 falsification FAIL — 3 FFs + 6 NITs)
- L1 `workflow/drop_4c_6_1/PLAN.md` W3 section (lines 276–339), W4.D1/W4.D2 sections (lines 342–451)
- L1 `REVISION_BRIEF.md` §2.7–2.10, §2.17; `SKETCH.md` §10
- Sibling round-2 L2 plans: `DROP_4c.6.1.W1_TEMPLATE_RESOLUTION/PLAN.md`, `DROP_4c.6.1.W2_TILL_INIT/PLAN.md`
- Live-code probes via `Read`: `internal/app/service.go` (UpdateProjectInput, ArchiveProject, RestoreProject, DeleteProject, CreateActionItem, CreateActionItemInput, ListColumns); `internal/domain/workitem.go` (ActionItemMetadata.BlockedBy); `internal/domain/structural_type.go`; `internal/domain/role.go`; `internal/domain/kind.go`; `cmd/till/project_cli.go` (cliMutationContext call site); `internal/app/auto_generate_steward.go` (firstColumnForProject canonical-pattern citation); `internal/templates/embed.go` (DefaultTemplateFS); `internal/templates/builtin/` (TOML filename layout post-W4.D2).
- `wc -l cmd/till/main.go` → **4069 lines** (confirms round-2 "~4,069 LOC (163KB)").

**Hylla:** OFF per spawn directive. Live-code verification via `Read` + scoped `Bash`.

---

# Section 0 — SEMI-FORMAL REASONING

## Proposal

**Premises**
- W3 round-2 PLAN absorbed round-1 plan-QA findings (1 FF + 6 NITs from proof; 3 FFs + 6 NITs from falsification). Job: verify each absorption is concrete, byte-correct, and consistent with sibling round-2 plans + L1 + live code; surface any residual or new counterexamples.
- 12 attack vectors enumerated in the spawn directive — every one tried, each landing CONFIRMED, REFUTED with citation, or EXHAUSTED.

**Evidence plan**
- Direct `Read` of W3 round-2 PLAN.md byte-by-byte at AC8 / D6 AC / D6 ContextBlock / D6 KindPayload (vector 3).
- Cross-reference W1 round-2 D2 KindPayload + Acceptance (vector 1).
- Cross-reference W2 round-2 CONSUMER-TIE wording (vector 11).
- Probe `internal/app/auto_generate_steward.go:120-160` for canonical column-resolution pattern (vector 2).
- Enumerate `domain.Kind` constants vs D3 12-kind list (vector 5).
- `wc -l cmd/till/main.go` (vector 6).
- Inspect `internal/templates/builtin/` directory listing for actual filename layout (vector 9).
- Cross-reference W4.D1 + W4.D2 paths against W3 D4/D5 path forms (vectors 9 + 10).
- Spot-check `cliMutationContext` location at `cmd/till/project_cli.go:142` (vector 12).
- Read W4 paths to confirm template / agent file layout (vectors 9 + 10).

**Conclusion plan**
- Single CONFIRMED counterexample: CONSUMER-TIE contract drift between W3 round-2 and L1/W2.
- Two NITs: D3 ContextBlock cites `auto_generate_steward.go:123-158` as canonical for `ListColumns` pattern but that file actually uses a package-private helper; D6 PLAN-QA-DISCIPLINE-R1 gap on "missing source" test for `qa-falsification` family (only `qa-proof` family directly tested in `TestRunAgentsBootstrap_MissingFileReport`'s missing-file flow per the line-622 enumeration).
- 1 EXHAUSTED attack family (D6 fan-out coverage all 4 destinations — verified present).
- 9 REFUTED attacks with concrete citations.

## QA Proof

**Premises** every attempted attack carries a file:line citation or verbatim quote. REFUTED claims name the contradicting evidence.

**Evidence completeness** verified — every CONFIRMED finding has a concrete repro pointer (PLAN line N vs sibling-plan line M / live-code line P); every REFUTED has a citation.

**Conclusion** Proof PASS — evidence supports each verdict; no speculative findings.

**Unknowns** Hylla OFF; live-code probes via `Read`/`Bash` only — no false-negative risk on the surfaces enumerated. Bash `grep` denied broadly, so live-code line-counts and content used direct `Read` offsets + `/usr/bin/grep` explicit path where needed.

## QA Falsification (adversarial sweep over my own verdict)

**Self-attack #1: Is the CONSUMER-TIE drift really an FF or just stylistic?**
Counter: L1 PLAN.md line 286 + line 334 explicitly mandate `run(ctx, args, &out, io.Discard)` end-to-end as the contract carried forward from Drop 4c.6 W2. W2 round-2 PLAN's CONSUMER-TIE bullets (lines 74, 87, 124, 132, 167, 215, 261, 305, 353, etc.) consistently use `run(ctx, args, &out, io.Discard)`. W3 round-2 PLAN line 114-116 has now changed to "tests exercise each `runXxx` directly with a constructed options struct... No cobra exec in tests." This is a substantive contract change — `runXxx`-direct tests skip the cobra flag binding layer that `run(args)` exercises. Build-QA for W3 will compare against W3's own contract, but cross-wave consistency (which the dev mandates per `feedback_use_typed_agents` + `feedback_plan_down_build_up`) is broken. CONFIRMED stands.

**Self-attack #2: Is the `auto_generate_steward.go:123-158` citation actually wrong?**
Counter: Direct read of that file shows line 123 calls `s.firstColumnForProject(ctx, project.ID)` — an `internal/app` package-private helper, NOT `svc.ListColumns(ctx, projectID, false)` as the W3 PLAN line 408 claims. The CLI in `cmd/till` cannot call `firstColumnForProject` directly (unexported). The PLAN correctly directs builder to use `svc.ListColumns`, but mis-cites the canonical pattern as if `auto_generate_steward.go` uses `ListColumns` — it doesn't. Builder reading the cited file as "canonical pattern" will see a different API call. NIT-level: builder will recover via LSP, but the misdirection costs spawn budget. NIT stands.

**Self-attack #3: Did I miss any attack family?**
Reviewed: blocker graph (linear, no cycle); `_BLOCKERS.toml` mirror (verified); R3-NIT6 verbatim (4 occurrences checked at PLAN lines 64, 150-152, 619, 635-637, 679 — all match L1 no-trailing-period form); 12-kind smart-default enumeration; D6 2-into-4 fan-out (4 destinations); D7 LOC split criterion; Metadata.BlockedBy field; cliMutationContext reference; W4.D2 path drift on `builtin/till-<group>.toml`; W4.D1 path drift on `builtin/agents/<group>/<name>.md`; sibling W1.D2 typed Groups shipping; D2 cliMutationContext bullet; service.go func-line citations; structural_type/role/kind enums. All twelve user-spawn vectors checked. No miss.

**Self-attack #4: Could I be overcounting findings?**
The CONSUMER-TIE drift is one finding (not two). The `auto_generate_steward.go` mis-citation is a NIT, not a FF (builder recovers via LSP per ContextBlock line 407-409's correct API call). The "missing source for qa-falsification" attack on D6 is also a NIT — the line-622 enumeration of bootstrap test cases names "missing-file reporting" without specifying which family; line 677 names `TestRunAgentsBootstrap_MissingFileReport` without shape_hint. Acceptable as a NIT.

**Conclusion** Falsification verdict: **FAIL** — 1 CONFIRMED FF (CONSUMER-TIE drift) + 2 NITs.

**Unknowns** none.

## Convergence

- (a) QA Falsification produced 1 CONFIRMED counterexample to W3 round-2's cross-wave correctness (CONSUMER-TIE contract drift). 2 NITs documented with concrete absorb routes.
- (b) QA Proof confirmed evidence completeness — every CONFIRMED finding has cited file:line + verbatim quotation.
- (c) Remaining Unknowns: none.

Verdict: **FAIL — 1 FF, 2 NITs.** PLAN.md needs a round-3 absorb pass before W3 builders dispatch. The structural absorption from round-1 (FF1/FF2/FF3 + 6 NITs) is otherwise byte-correct.

---

# QA Falsification Review

## 1. Findings

### FF (CONFIRMED counterexamples — must be absorbed before W3 builders dispatch)

#### FF1 — CONSUMER-TIE TEST CONTRACT drifts between W3 round-2 and L1 / sibling W2 round-2

- **Premises:** L1 PLAN.md line 286 + line 334 + L1 line 338 spawn directive all explicitly mandate `run(ctx, args, &out, io.Discard)` end-to-end as the CONSUMER-TIE TEST CONTRACT pattern carried forward from Drop 4c.6 W2. W3's CONSUMER-TIE wording must match this contract; sibling W2 round-2 PLAN must also follow it. Any drift breaks cross-wave consistency and the L1-declared contract.
- **Evidence:**
  - L1 PLAN.md line 286: "All follow the existing CONSUMER-TIE TEST CONTRACT (`run(ctx, args, &out, io.Discard)` end-to-end pattern from Drop 4c.6 W2)".
  - L1 PLAN.md line 334: "All commands follow CONSUMER-TIE TEST CONTRACT (`run(ctx, args, &out, io.Discard)` end-to-end pattern from Drop 4c.6 W2); `mage test-pkg ./cmd/till/...` passes."
  - L1 PLAN.md line 338 (W3 spawn directive): "All droplets use CONSUMER-TIE TEST CONTRACT (`run(ctx, args, &out, io.Discard)` end-to-end pattern from Drop 4c.6 W2)."
  - W2 round-2 PLAN.md line 74 (W2.D1 acceptance): "validation behavior tested via `run(ctx, args, &out, io.Discard)` end-to-end".
  - W2 round-2 PLAN.md line 87, 124, 132, 167, 215, 261, 305, 353: consistent `run(ctx, args, &out, io.Discard)` pattern across every W2 droplet.
  - W3 round-2 PLAN.md lines 111-116 (top ContextBlock — CONSUMER-TIE block):
    ```
    [constraint severity=high]
    CONSUMER-TIE TEST CONTRACT: every run function signature follows the existing pattern:
      func runXxx(ctx context.Context, svc *app.Service, cfg config.Config, opts xxxOptions, stdout io.Writer) error
    Tests exercise each runXxx directly with a constructed options struct + bytes.Buffer
    stdout + io.Discard stderr. No cobra exec in tests; option-struct construction in
    tests mirrors the cobra flag-binding shape.
    ```
  - W3 round-2 PLAN.md line 28 (Round 2 Changes preamble): "Fals NIT3 (CONSUMER-TIE wording self-contradiction): ABSORB. Top-level ContextBlock second sentence replaced to eliminate the `run(ctx, args, &out, io.Discard)` / `runXxx` contradiction."
- **Trace:** Round-1 falsification NIT3 surfaced the W3 PLAN's internal self-contradiction (first sentence said `runXxx`; second sentence said `run(ctx, args, &out, io.Discard)` — incompatible). Round-2 absorbed this by killing the `run(args)` sentence and choosing `runXxx` direct. But the L1-contract reading is the inverse: L1 wants `run(ctx, args, &out, io.Discard)` end-to-end (the L1 text uses parens with `args`, not the per-handler `runXxx` form). W2 round-2 honored L1; W3 round-2 deviated. Two waves now disagree on what "CONSUMER-TIE TEST CONTRACT" means; build-QA across the two waves will diverge. Worse, "No cobra exec in tests" skips the cobra flag-binding layer — meaning W3's smoke tests do NOT verify that the user-facing CLI flag wiring actually maps to the right `runXxx` argument; the round-2 PLAN's TestW3CommandsRegistered (line 753) becomes the only flag-binding smoke gate, and it's a `--help` test, not a behavior test.
- **Conclusion:** CONFIRMED. The W3 round-2 PLAN's CONSUMER-TIE wording is now byte-incompatible with L1's contract and W2 round-2's pattern. The dev's `feedback_plan_down_build_up.md` + `feedback_use_typed_agents.md` discipline requires cross-wave contract consistency.
- **Disposition (ABSORB in round 3):**
  - Restore the L1-aligned `run(ctx, args, &out, io.Discard)` end-to-end pattern as the primary CONSUMER-TIE bullet. W3's top-level ContextBlock should read (or equivalent):
    ```
    [constraint severity=high]
    CONSUMER-TIE TEST CONTRACT: every run function signature follows the existing pattern:
      func runXxx(ctx context.Context, svc *app.Service, cfg config.Config, opts xxxOptions, stdout io.Writer) error
    Tests exercise the top-level `run(ctx, args, &out, io.Discard)` end-to-end so the
    cobra flag-binding layer is also covered. Direct-invocation of `runXxx` with a
    constructed options struct is an acceptable SUPPLEMENT for table-driven sub-cases
    (per Drop 4c.6 W2 precedent), but the primary acceptance test for every new
    subcommand MUST exercise the cobra layer via `run(args)`.
    ```
  - Update every D1–D7 acceptance bullet's CONSUMER-TIE supplement to name `run(ctx, args, &out, io.Discard)` as the primary path (matching W2 round-2's wording at PLAN.md line 74 / 87 / etc.).
  - If the dev instead INTENDS W3 to break from L1 and use `runXxx`-direct tests, surface a documented L1 contract amendment first (and update W2 + L1 in lockstep). Otherwise round-3 should reverse the round-2 absorption of FALS NIT3 in favor of the L1-aligned form.

---

## 2. Counterexamples

CONFIRMED counterexamples (1): FF1 (CONSUMER-TIE contract drift from L1 and sibling W2 round-2).

REFUTED self-attacks + previous-round attacks now resolved:

- **R1 (round-1 FF1 `--add-group`/`--remove-group` missing):** RESOLVED. PLAN.md AC1 (line 57), D1 acceptance (lines 232-234), D1 KindPayload (line 277 — `addGroups []string; removeGroups []string` in `projectUpdateCommandOptions`) all include both flags. RiskNote (line 240) drops the round-1 "if absent, omit" hedge and now states "W1.D2 ships this typed field on `domain.ProjectMetadata` BEFORE W3 dispatches. Builder MUST verify via LSP `goToDefinition` on `domain.ProjectMetadata` that the `Groups []string` field exists before writing. If (unexpectedly) absent, STOP and return to orchestrator rather than adding a TODO stub." Cross-checked W1 round-2 D2 KindPayload (line 264 of W1 PLAN): ships `Groups []string` field with JSON tag `groups,omitempty` on `domain.ProjectMetadata`. W1 D2 Acceptance #1 (line 375 of W1 PLAN) confirms the same. Spellings match byte-for-byte.
- **R2 (round-1 FF2 ColumnID omit vs domain reject):** RESOLVED. D3 ContextBlock lines 402-410 now have `constraint severity=critical` block: "ColumnID is REQUIRED by domain.NewActionItem — empty ColumnID rejects with ErrInvalidColumnID. till action_item create MUST resolve a default ColumnID BEFORE calling (*Service).CreateActionItem." The canonical-pattern citation is mildly misleading (see NIT1 below), but the directed API call (`svc.ListColumns(ctx, projectID, false)` → first column) is correct. `service.go:1870` reads: `func (s *Service) ListColumns(ctx context.Context, projectID string, includeArchived bool) ([]domain.Column, error)` — sorted by Position ascending (line 1875-1877). The "first column" rule lands on the lowest-Position column, which matches `auto_generate_steward.go`'s use of `firstColumnForProject`. AC3 (line 59) confirms ColumnID auto-resolved from first column. D3 test `TestRunActionItemCreate_PassThroughFlags` (line 445) covers the column auto-resolution.
- **R3 (round-1 FF3 R3-NIT6 verbatim drift):** RESOLVED. All 5 occurrences now match L1 PLAN.md line 338's no-trailing-period form (string ends with closing backtick on `--force`):
  - PLAN.md line 64 (AC8): no trailing period ✓
  - PLAN.md lines 150-152 (top ContextBlock R3-NIT6 block): no trailing period; explicit "The string ends with the closing backtick — NO trailing period after the backtick." on line 154 ✓
  - PLAN.md line 619 (D6 AC bullet): no trailing period ✓
  - PLAN.md lines 635-637 (D6 ContextBlock): no trailing period; explicit "NO trailing period after the backtick" on line 638 ✓
  - PLAN.md line 679 (D6 KindPayload shape_hint for `TestRunAgentsBootstrap_ForceHelpTextContainsWarning`): no trailing period in the asserted substring; explicit "no trailing period after closing backtick" qualifier ✓
- **R4 (round-1 proof NIT2 smart-default 12-kind coverage):** RESOLVED. D3 KindPayload `TestRunActionItemCreate_StructuralTypeSmartDefault` shape_hint at PLAN.md line 443 enumerates all 12 kinds: plan→segment, refinement→segment, build→droplet, research→droplet, plan-qa-proof→droplet, plan-qa-falsification→droplet, build-qa-proof→droplet, build-qa-falsification→droplet, closeout→droplet, commit→droplet, discussion→droplet, human-verify→droplet. Cross-checked vs `internal/domain/kind.go` enum (12 constants at lines 19-30): exact match. Plus explicit-override-valid (confluence) and explicit-override-invalid (error with valid-values list).
- **R5 (round-1 proof NIT3 D6 fan-out test 4 destinations):** RESOLVED. AC7 (line 63) enumerates all 4 destination files: `plan-qa-proof-agent.md, build-qa-proof-agent.md, plan-qa-falsification-agent.md, build-qa-falsification-agent.md`. D6 AC bullet at line 622 enumerates the same 4 destinations as test-coverage requirement. D6 KindPayload line 674 shape_hint for `TestRunAgentsBootstrap_QAFanOut`: "verifies all 4 fan-out destination files written with identical-to-source content: <group>/plan-qa-proof-agent.md, <group>/build-qa-proof-agent.md, <group>/plan-qa-falsification-agent.md, <group>/build-qa-falsification-agent.md". Byte-exact.
- **R6 (round-1 proof NIT4 pass-through flags untested):** RESOLVED. D3 KindPayload now has 3 test families: `TestRunActionItemCreate_StructuralTypeSmartDefault` (line 443), `TestRunActionItemCreate_RequiredFields` (line 444), `TestRunActionItemCreate_PassThroughFlags` (line 445). The third test verifies paths, packages, files, blocked-by, metadata-json, parent-id, role pass-through plus ColumnID auto-resolution.
- **R7 (round-1 proof NIT5 service.go line numbers):** RESOLVED by deferral with disposition: PLAN cites `func` keyword lines (625/669/689/709/1035), which the absorption confirms via direct `Read`. Verified: lines 625, 669, 689, 709, 1035 all point to the `func (s *Service) ...` declaration line; lines 624/668/688/708/1034 are the preceding `//` doc-comment line. Standard Go practice cites the func signature line. Absorption rationale is sound.
- **R8 (round-1 proof NIT6 / fals NIT1c `pre-Drop-2` qualifier stale):** RESOLVED. D3 RiskNote (line 389) drops the "pre-Drop-2" qualifier. Confirmed via `internal/app/service.go:742-746`: doc-comment says "Empty string is permitted; non-empty values must match the closed Role enum or domain.NewActionItem returns ErrInvalidRole" — unconditional.
- **R9 (round-1 fals NIT1 `--blocked-by` open-ended language):** RESOLVED. D3 ContextBlock at lines 416-421 now explicitly states: "BlockedBy is wired via CreateActionItemInput.Metadata.BlockedBy []string (the ActionItemMetadata struct field at internal/domain/workitem.go:195, reachable via CreateActionItemInput.Metadata). The --blocked-by flag accumulates []string values into opts.blockedBy; builder merges them into the constructed domain.ActionItemMetadata.BlockedBy before calling (*Service).CreateActionItem. No post-create UpdateActionItem is needed." Live-code verified: `internal/domain/workitem.go:195` is `BlockedBy []string` field on `ActionItemMetadata`; `internal/app/service.go:825` is `Metadata domain.ActionItemMetadata` on `CreateActionItemInput`. Direct pre-create injection path confirmed.
- **R10 (round-1 fals NIT2 D7 "163K LOC" factual error):** RESOLVED. PLAN line 711 + line 720: "main.go is ~4,069 LOC (163KB file size)". Verified via `wc -l cmd/till/main.go` → **4069** lines. Byte-exact.
- **R11 (round-1 fals NIT4 D7 LOC budget conditional split not acknowledged):** RESOLVED. D7 RiskNote (line 715) adds: "If D7's cobra registrations + test exceed 120 LOC production code, split: D7a covers project + action_item subcommand registrations; D7b covers template + agents subcommand registrations + smoke test. D7b blocked_by D7a since both touch main.go." Concrete trigger (120 LOC ceiling) + concrete split criterion (project+action_item vs template+agents). Crisp.
- **R12 (round-1 fals NIT5 `cliMutationContext` implicit in D2):** RESOLVED. D2 ContextBlock at PLAN line 327-331 explicitly references the pattern: "Every mutating subcommand (delete/archive/restore/rename) MUST call cliMutationContext(ctx, cfg) before any (*Service) call — pattern at project_cli.go:142." Live-code verified at `cmd/till/project_cli.go:142`: `ctx = cliMutationContext(ctx, cfg)` is the line, called inside `runProjectCreate` between `buildProjectMetadata` and the `svc.CreateProjectWithMetadata` call. Exact line. D1 ContextBlock at PLAN line 268-270 mirrors the same reference.
- **R13 (round-1 fals NIT6 D4/D5 embedded FS API ambiguity):** RESOLVED. D4 RiskNote (line 480) now explicitly specifies "Embedded defaults: use `templates.DefaultTemplateFS.ReadFile(path)` for raw-bytes access (path form: `builtin/till-<group>.toml`). Do NOT use `templates.LoadDefaultTemplateForLanguage` — that returns a parsed Template struct, not raw bytes". D5 RiskNote (line 554) same shape for agents at path form `builtin/agents/<group>/<name>.md`. Cross-checked vs W4.D2 paths: TOML filenames retain `till-` prefix (`till-go.toml`, `till-gen.toml`, NEW `till-fe.toml`) per L1 PLAN.md line 423-426 — matches D4's `builtin/till-<group>.toml`. Cross-checked vs W4.D1 paths: agent dirs lose `till-` prefix post-`git mv` (`agents/go/`, `agents/gen/`, NEW `agents/fe/`) per L1 PLAN.md line 351-352 — matches D5's `builtin/agents/<group>/<name>.md`. `internal/templates/embed.go:104` confirms `DefaultTemplateFS embed.FS` is exported, supporting `ReadFile(string) ([]byte, error)`.

## NIT (cosmetic / clarification / low-risk drifts — first-class per `feedback_nits_are_first_class.md`; default ABSORB unless explicit reason)

### NIT1 — D3 ContextBlock cites `auto_generate_steward.go:123-158` as canonical for the `ListColumns` pattern, but the cited file uses `firstColumnForProject` (a package-private helper), not `ListColumns`

- **Evidence:** PLAN.md lines 406-410:
  ```
  ColumnID is REQUIRED by domain.NewActionItem — empty ColumnID rejects with ErrInvalidColumnID.
  till action_item create MUST resolve a default ColumnID BEFORE calling (*Service).CreateActionItem.
  Canonical pattern (auto_generate_steward.go:123-158 + service.go:1870):
    columns, err := svc.ListColumns(ctx, projectID, false)
    if err != nil { return err }
    ...
  ```
  Direct `Read` of `internal/app/auto_generate_steward.go:120-160` shows line 123 is `column, err := s.firstColumnForProject(ctx, project.ID)` — `firstColumnForProject` is an `internal/app` package-private helper, not `Service.ListColumns`. The CLI in `cmd/till` cannot call `firstColumnForProject` directly (unexported). The PLAN's direction to use `svc.ListColumns` is correct in API choice; the canonical-pattern citation is misleading because it points at a file that uses a different (private) API.
- **Disposition (ABSORB):** Replace the canonical-pattern citation. Either (a) drop the file-citation entirely and keep the `service.go:1870` `ListColumns` reference, OR (b) point to a different file where `Service.ListColumns` is consumed externally by a CLI handler (search `cmd/till/` for existing `svc.ListColumns` consumers via LSP — if none exists, this CLI is the first consumer and the pattern is novel; that's fine, just say so). Suggested wording:
  ```
  ColumnID resolution pattern (NEW — this CLI is the first cmd/till consumer of
  svc.ListColumns; internal/app/auto_generate_steward.go:123 uses a package-private
  helper `firstColumnForProject` that the CLI cannot call):
    columns, err := svc.ListColumns(ctx, projectID, false)
    if err != nil { return fmt.Errorf("list columns: %w", err) }
    if len(columns) == 0 { return fmt.Errorf("project has no columns") }
    columnID := columns[0].ID  // ListColumns sorts by Position ascending (service.go:1875-1877)
  ```

### NIT2 — D6 `TestRunAgentsBootstrap_MissingFileReport` shape_hint missing; current text doesn't specify whether missing-source coverage extends to qa-falsification (vs only qa-proof) or to group-agnostic files

- **Evidence:** PLAN.md line 677:
  ```
  {"file": "cmd/till/agents_cli_test.go", "symbol": "TestRunAgentsBootstrap_MissingFileReport", "action": "add"},
  ```
  No `shape_hint` field. D6 AC at line 622 says "missing-file reporting" without enumerating which file families are tested for missing-from-source. AC line 616 names the bootstrap behavior: "Missing files are reported to stdout (e.g. 'Missing: go-planning-agent.md (not found in source)')." The example is for a planning agent (`go-planning-agent.md`); coverage should extend to all 10 standard agent names per group (planning, builder, plan-qa-proof, build-qa-proof, plan-qa-falsification, build-qa-falsification, research, closeout, commit-message, orchestrator-managed).
- **Disposition (ABSORB):** Add shape_hint to line 677:
  ```
  "shape_hint": "verifies bootstrap reports each missing source-file by name when present in
  expected 10-agent set but absent from --from dir; covers at minimum: missing planning agent
  (e.g. go-planning-agent.md), missing one half of qa-proof pair (e.g. go-qa-proof-agent.md
  absent → both plan-qa-proof and build-qa-proof destinations skipped + reported), missing
  one half of qa-falsification pair (symmetric), missing group-agnostic file (e.g.
  closeout-agent.md absent → all 3 known groups report missing); verifies orchestrator-managed.md
  starter is generated even when source missing (per AC line 617)"
  ```

---

## 3. Summary

**Verdict: FAIL.**

- **1 CONFIRMED FF (must absorb in round 3):** FF1 — CONSUMER-TIE TEST CONTRACT drift between W3 round-2 PLAN and L1 contract / sibling W2 round-2 pattern. W3 now mandates `runXxx`-direct tests with "No cobra exec in tests"; L1 + W2 mandate `run(ctx, args, &out, io.Discard)` end-to-end. The dev's cross-wave consistency discipline forces alignment; either revise W3 to L1's form or amend L1 + W2 in lockstep.
- **2 NITs (default ABSORB):**
  - NIT1: D3 ContextBlock canonical-pattern citation points at `auto_generate_steward.go:123-158`, but that file uses a package-private helper (`firstColumnForProject`), not `ListColumns`. Builder reading the citation gets misleading evidence; PLAN's API direction (`svc.ListColumns`) is correct but the cited "canonical" pattern doesn't match.
  - NIT2: D6 `TestRunAgentsBootstrap_MissingFileReport` shape_hint missing — round-2 absorbed all-4-destinations coverage for the fan-out test, but missing-file reporting doesn't enumerate which file families (planning vs qa-proof half vs qa-falsification half vs group-agnostic).
- **Round-1 findings closed (13 of 13):** All round-1 FFs (3) + proof NITs (6) + falsification NITs (6) are byte-correctly absorbed. R3-NIT6 verbatim text fixed at all 5 PLAN occurrences. `--add-group`/`--remove-group` typed Groups consumption wired. ColumnID resolution constraint added. Smart-default 12-kind coverage enumerated. 4-destination fan-out coverage enumerated. Pass-through flags test family added. Service.go line citations consistent with `func`-line convention. `pre-Drop-2` qualifier dropped. BlockedBy pre-create Metadata injection path documented. D7 LOC corrected to 4,069 + 163KB. D7a/D7b split criterion added. D2 cliMutationContext reference added. D4/D5 embedded FS API tightened with explicit `DefaultTemplateFS.ReadFile(path)` form.
- **Structural attacks all REFUTED:** no cycles, no missing `blocked_by` edges, no `_BLOCKERS.toml` vs PLAN.md drift, no YAGNI / over-decomposition findings, no atomic-sizing violations.

The plan's structural absorption is sound; the residual defect is a single cross-wave contract drift that round-2's NIT3 absorption introduced unintentionally.

---

## TL;DR

- **T1:** Verdict FAIL — 1 FF + 2 NITs require round-3 absorption before W3 dispatch.
- **T2:** FF1: W3 round-2 PLAN's CONSUMER-TIE wording drifted from L1 contract (`run(ctx, args, &out, io.Discard)` end-to-end) + sibling W2 round-2 pattern. W3 now mandates `runXxx`-direct invocation with "No cobra exec in tests" — round-1 fals NIT3 absorption picked the wrong reconciliation. Round-3 should restore L1 form.
- **T3:** Round-1 13/13 findings closed byte-correctly. NITs cover D3's misleading canonical-pattern citation (cites `auto_generate_steward.go:123-158` which uses `firstColumnForProject` private helper, not `ListColumns`) and D6's missing test shape_hint on `TestRunAgentsBootstrap_MissingFileReport` (which file families are covered for missing-from-source).

---

## Hylla Feedback

N/A — action item touched non-Go files only (PLAN.md, REVISION_BRIEF.md, _BLOCKERS.toml, sibling PLAN.md files). Per spawn directive: "Hylla is OFF". All Go-symbol verification went through `Read` against `internal/domain/structural_type.go`, `internal/domain/role.go`, `internal/domain/kind.go`, `internal/domain/workitem.go`, `internal/domain/project.go` (indirectly via W1 PLAN cross-check), `internal/app/service.go`, `internal/app/auto_generate_steward.go`, `cmd/till/project_cli.go`, `internal/templates/embed.go`. No Hylla fallback to report.
