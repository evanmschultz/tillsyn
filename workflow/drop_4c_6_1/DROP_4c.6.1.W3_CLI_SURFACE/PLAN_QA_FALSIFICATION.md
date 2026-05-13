# PLAN-QA FALSIFICATION — Drop 4c.6.1.W3 (CLI_SURFACE)

**Reviewer:** L2 plan-QA falsification agent
**Round:** 1
**Date:** 2026-05-12
**Inputs reviewed:** `workflow/drop_4c_6_1/DROP_4c.6.1.W3_CLI_SURFACE/PLAN.md` + `_BLOCKERS.toml`; L1 `workflow/drop_4c_6_1/PLAN.md` W3 section + wave graph; `REVISION_BRIEF.md` §2.7–2.10 + §2.17; `SKETCH.md` §10; sibling L2 plans `DROP_4c.6.1.W1_TEMPLATE_RESOLUTION/PLAN.md` + `DROP_4c.6.1.W2_TILL_INIT/PLAN.md`; live code: `internal/app/service.go` (UpdateProjectInput / CreateActionItemInput / CreateActionItem / seedStewardAnchors), `internal/domain/action_item.go` (ActionItem / ActionItemInput / NewActionItem), `internal/domain/project.go` (Project / ProjectMetadata / UpdateDetails / Rename), `internal/domain/workitem.go` (ActionItemMetadata), `internal/app/ports.go`, `internal/app/auto_generate_steward.go` (CreateActionItemInput consumer), `internal/templates/embed.go` (DefaultTemplateFS / LoadDefaultTemplateForLanguage), `cmd/till/project_cli.go` (runProjectCreate / cliMutationContext consumer pattern), `cmd/till/action_item_cli.go` (no existing runActionItemCreate; pattern surface).

---

# Section 0 — SEMI-FORMAL REASONING

## Proposal

**Premises:**
- W3 ships 16 cobra subcommands (15 + bootstrap) plus main.go registration across 7 atomic droplets in a strict serial chain (D1→D7), all sharing the `cmd/till` package compile unit.
- Falsification job: attempt counterexamples against (a) blocker graph, (b) cross-wave contract drift, (c) PLAN vs live-code Go-API mismatch, (d) verbatim-string drift between L1 and L2, (e) acceptance-bullet coverage in test wiring, (f) atomic-droplet sizing, (g) YAGNI on the 7-droplet split.
- PASS verdict requires every attack family attempted, each landing CONFIRMED or REFUTED, no unmitigated counterexample.

**Evidence:**
- Service-layer probes via `Read`: `service.go:600-666` (UpdateProjectInput + UpdateProject), `service.go:709-720` (DeleteProject), `service.go:736-831` (CreateActionItemInput — no BlockedBy field, ColumnID required), `service.go:1035-1204` (CreateActionItem — ColumnID empty → `domain.NewActionItem` rejects with ErrInvalidColumnID).
- Domain probe: `action_item.go:24-277` (ActionItem struct — no top-level BlockedBy field), `:291-365` (NewActionItem — empty ColumnID returns `ErrInvalidColumnID`, empty StructuralType returns `ErrInvalidStructuralType`), `workitem.go:179-200` (ActionItemMetadata.BlockedBy []string is the storage seat for blocker edges).
- Domain probe: `project.go:288-298` (Project.Rename — sets Name + reslugs), `:309-353` (UpdateDetails — also reslugs on every call).
- Domain probe: `project.go:119-155` (ProjectMetadata — no Groups field today; ships from W1.D2).
- Sibling L2 plans: `W1_TEMPLATE_RESOLUTION/PLAN.md` D2 ships `ProjectMetadata.Groups []string` BEFORE W3 fires; `W2_TILL_INIT/PLAN.md` D1 finalizes `allowedInitGroups = ["gen", "go", "fe"]` BEFORE W3 fires.
- L1 PLAN.md line 282 `till project update` flag surface includes `--add-group <name>` and `--remove-group <name>`.
- L1 PLAN.md line 317 R3-NIT6 verbatim string ends with `` `--force` `` (closing backtick, no trailing period).
- Existing canonical CreateActionItem-consumer `auto_generate_steward.go:140-158` resolves default column via `firstColumnForProject(ctx, project.ID)` and passes `ColumnID: column.ID` — the established pattern.
- Existing project-CLI pattern `project_cli.go:134-152` uses `cliMutationContext(ctx, cfg)` before any mutating service call.

**Trace or cases (attacks planned):**
1. Blocker graph: 7-node linear chain D1→D7 — check for cycles + missing edges.
2. _BLOCKERS.toml ↔ PLAN.md drift.
3. Hidden file/package locks: every droplet touches `cmd/till` — verify serial chain is mandatory and complete.
4. Cross-wave contract: D1 TODO fallback for `--add-group/--remove-group` "if Groups absent" vs W1.D2 ships Groups BEFORE W3.
5. Cross-wave contract: D6 hard-coded `go|fe|gen` group names vs W4.D1 ships them.
6. PLAN ↔ live-code drift: D3 says "ColumnID (omit)" vs domain rejects empty ColumnID.
7. PLAN ↔ live-code drift: D3 `--blocked-by` "passed through" vs `CreateActionItemInput` has no top-level BlockedBy field.
8. R3-NIT6 verbatim text drift: L1 PLAN.md ends `` `--force` `` (no trailing period); W3 PLAN AC8 + ContextBlock + RiskNote end `--force.` (period after backtick).
9. PLAN-QA-DISCIPLINE-R1: every AC asserting NEW behavior maps to a test-runner shipping it.
10. PLAN-QA-DISCIPLINE-R2 numeric: 16 commands narrated vs 7 droplets enumerated.
11. YAGNI: any droplet that could merge or be skipped?
12. D2 `till project rename` via UpdateProject correctness (no RenameProject service method).
13. D4/D5 embedded FS API: `DefaultTemplateFS.ReadFile(...)` vs `LoadDefaultTemplateForLanguage(...)`.
14. D6 missing-file reporting path test coverage.
15. D7 `main.go` size estimate (163K LOC claim) + LOC budget feasibility.
16. NIT-class: ContextBlock wording inconsistencies, low-risk drifts.

**Conclusion:** plan to render verdict in Section 0 Convergence after running every attack and either landing it CONFIRMED with reproducible trace + disposition route, or REFUTING with concrete reason.

**Unknowns:** none at proposal time — every attack family has either a concrete probe target in live code, a sibling L2 plan to cross-check, or a verbatim-text comparison.

---

## QA Proof

**Premises:** every attempted attack must cite evidence (file:line / verbatim quote / sibling-plan section); REFUTED claims must name the specific contradicting evidence.

**Evidence:**
- All 16 attacks below carry inline `service.go:N` / `action_item.go:N` / `project.go:N` / L1 PLAN.md:N / W3 PLAN.md:N citations.
- Cross-wave attacks 4 + 5 backed by sibling L2 PLAN.md reads (W1.D2 line 28; W2.D1 line 44).
- Live-code attacks 6 + 7 backed by `auto_generate_steward.go:140-158` canonical pattern + `action_item.go:309-310` rejection paths.

**Trace or cases:** see Findings below — each FF/NIT entry follows the schema {Premise / Evidence / Trace / Conclusion / Disposition}.

**Conclusion:** evidence completeness check PASS — every CONFIRMED finding has reproducible repro pointer; every REFUTED finding has a contradicting-evidence citation.

**Unknowns:** Hylla is OFF for this drop, so live-code probes used `Read` + sandboxed `Bash` only. Bash `grep` denied broadly; falsification reads compensated via direct `Read` offsets. No false-negative risk on the attack vectors enumerated.

---

## QA Falsification (adversarial sweep over my own verdict)

**Premises:** the attacks above must themselves survive counterexample search. Self-attack #1: did I miss an attack family? Self-attack #2: am I overcounting NITs? Self-attack #3: did the FF dispositions route correctly?

**Evidence + counterexamples:**

- Self-attack: "FF1 (`--add-group`/`--remove-group` drop) might be moot because W3 wave-level blocked_by W1 means W1.D2's Groups field IS shipped by W3 dispatch." Counter: the L2 PLAN's D1 RiskNote explicitly hedges "if absent, omit these flags and add a TODO" — which is dead code per the wave dep. The plan needs surgical correction (add the flags to D1's acceptance + KindPayload), not the hedge. CONFIRMED stands.
- Self-attack: "FF2 (ColumnID `omit`) might be reading the wrong path — perhaps service auto-resolves." Counter: `service.go:1148` passes `in.ColumnID` straight through to `domain.NewActionItem`, and `action_item.go:309-310` returns `ErrInvalidColumnID` on empty. The fallback at `service.go:1120-1123` is only for `lifecycleState`, NOT for `ColumnID` itself. CONFIRMED stands.
- Self-attack: "FF3 (verbatim drift) might be cosmetic." Counter: W3 AC8 + R3-NIT6 ContextBlock both call it `MUST include VERBATIM (exact string match, case-sensitive)`. Build-QA WILL check character-exactness per the L2 plan's own contract. The L2 plan internally drifts from L1, so build-QA will either fail-match against L1 (catches W3's drift) or fail-match against W3 (catches builder following L2). Either way the drift is load-bearing. CONFIRMED stands.
- Self-attack: "Am I overcounting? Could FF1 and the `--add-group` framing be the same FF as a missing acceptance bullet?" Resolution: they ARE the same finding — keep as single FF1 with two parts (acceptance + RiskNote hedge).
- Self-attack: "FF4 (--blocked-by routing) — maybe builder will discover and adapt." Counter: planning DEFECT is the same shape as FF2 ColumnID — the plan SHOULDN'T require the builder to discover a missing instruction. Plan must explicitly say "wire via `Metadata.BlockedBy []string` or post-create UpdateActionItem". The plan DOES flag this risk in ContextBlock line 376-378 ("BlockedBy on CreateActionItemInput: check LSP... If absent ... the --blocked-by flag adds a post-create UpdateActionItem call. Verify BEFORE writing."). On closer read: this IS adequate planner guidance (verify + choose path), even though it leaves the path open. REFUTED on second look — flag-as-NIT to make explicit.
- Self-attack: "Are the YAGNI / atomic-sizing attacks worth raising?" Counter: D4 (5 subcommands, ~120 LOC at upper end) and D7 (16 cobra registrations × 8 LOC ≈ 128 LOC) are both at-or-over the 120 LOC ceiling. Plan acknowledges D4 split; D7 split is NOT acknowledged. NIT-level concern.

**Trace:** every attack now either lands or is explicitly refuted with a counter-citation; no speculative "could fail" attacks left dressed up as findings.

**Conclusion:** the falsification verdict is **FAIL** — 3 CONFIRMED FFs and 6 NITs documented below. Counter-attacks above narrow the finding set; no inflated findings remain.

**Unknowns:** none.

---

## Convergence

- (a) QA Falsification produced 3 CONFIRMED counterexamples to the W3 plan's correctness (FF1 missing flag pair, FF2 ColumnID omit, FF3 verbatim drift). Each has a concrete absorb-route below.
- (b) QA Proof confirmed evidence completeness — every CONFIRMED finding has cited file:line + verbatim quotation; every REFUTED concern names a contradicting citation.
- (c) Remaining Unknowns: none.

Verdict: **FAIL — 3 FFs, 6 NITs.** PLAN.md needs a round-2 absorb pass before W3 builders dispatch.

---

# QA Falsification Review

## 1. Findings

### FF (CONFIRMED counterexamples — must be absorbed before W3 builders dispatch)

#### FF1 — W3.D1 silently drops `--add-group` / `--remove-group` flags that L1 PLAN.md requires

- **Premises:** L1 PLAN.md is authoritative on the W3 command surface. W3 L2 plan must cover every flag L1 names as in-scope.
- **Evidence:**
  - L1 PLAN.md line 282 (W3 § Scope → Project lifecycle CLIs → update): `update --project-id <id> [--root-path ...] [--bare-root ...] [--language ...] [--add-group <name>] [--remove-group <name>] [--hylla-artifact-ref ...] [--description ...]`. Explicitly includes both group-mutation flags.
  - W3 PLAN.md line 36 (AC1) enumerates flags: `root-path, bare-root, language, description, hylla-artifact-ref, build-tool, dev-mcp-server-name, owner, icon, color, homepage, tags` — **`--add-group` and `--remove-group` ABSENT.**
  - W3 PLAN.md line 211 (D1 RiskNote): "`--add-group` and `--remove-group` flags are listed in REVISION_BRIEF §2.8 but `ProjectMetadata` has no `Groups []string` field today (check LSP on domain.ProjectMetadata before implementing). If absent, omit these flags and add a TODO comment for the future."
  - W3 wave-level `blocked_by W1` (PLAN.md line 7). Sibling W1 PLAN.md line 28 (D2): "`domain.ProjectMetadata` gains a `Groups []string` field." Sibling W1 PLAN.md line 68: "`domain.ProjectMetadata` carries a `Groups []string` field with JSON tag `\"groups,omitempty\"`."
  - Live-code probe `project.go:119-155` confirms `ProjectMetadata` does NOT have `Groups []string` today — it ships from W1.D2.
- **Trace:** W3 → blocked_by → W1 (wave-level) → W1.D2 ships `Groups`. By the time W3.D1 dispatches, the field exists. The TODO/omit fallback is unreachable code; the L2 plan must instead REQUIRE both flags + verify behavior against the W1-shipped field.
- **Conclusion:** CONFIRMED. The W3.D1 acceptance criteria + KindPayload + ContextBlocks are missing `--add-group` and `--remove-group`. The RiskNote's hedge is dead code given W3's wave dep.
- **Disposition (ABSORB in round 2):**
  - Add to W3.D1 AC1 the two flags: `till project update --project-id <id> --add-group fe` appends `"fe"` to `Metadata.Groups` (dedup); `--remove-group go` removes `"go"`; both are repeated cobra flags. Validation: each group value must be in the W2-confirmed allowed set (`gen`, `go`, `fe`).
  - Add to KindPayload's shape_hint: `addGroups []string; removeGroups []string` in `projectUpdateCommandOptions`.
  - Remove the "If absent, omit these flags" hedge from RiskNotes line 211; replace with "W1.D2 ships `ProjectMetadata.Groups []string` BEFORE W3 dispatches; builder MUST verify via LSP and wire both flags."
  - Add a test row to `TestRunProjectUpdate_*` covering add/remove/conflict (e.g. `--add-group go --remove-group go`).

#### FF2 — W3.D3 instructs builder to OMIT ColumnID, but `domain.NewActionItem` rejects empty ColumnID with `ErrInvalidColumnID`

- **Premises:** `(*Service).CreateActionItem` is the only public path for creating action items; it threads `in.ColumnID` directly into `domain.NewActionItem`.
- **Evidence:**
  - W3 PLAN.md line 369 (D3 ContextBlock `reference`): "CreateActionItemInput fields that the CLI wires: ProjectID, ParentID, Kind, Role, StructuralType, Title, Description, Paths, Packages, Files, StartCommit (omit), EndCommit (omit), **ColumnID (omit)**, Priority (omit), DueAt (omit), Labels (omit), Metadata (pass --metadata-json if supplied), BlockedBy (...)."
  - Live-code `service.go:1131-1162` (CreateActionItem → NewActionItem): `actionItem, err := domain.NewActionItem(domain.ActionItemInput{ ..., ColumnID: in.ColumnID, ... }, s.clock())`.
  - Live-code `action_item.go:309-311`: `if in.ColumnID == "" { return ActionItem{}, ErrInvalidColumnID }`.
  - Canonical pattern at `auto_generate_steward.go:123-147`: every existing in-tree `CreateActionItemInput` caller calls `s.firstColumnForProject(ctx, projectID)` to resolve a default column and passes `ColumnID: column.ID`.
- **Trace:** D3 build per the current plan will land code that calls `svc.CreateActionItem(ctx, CreateActionItemInput{ProjectID, Kind, Title, ..., /* ColumnID empty */})`. Every test case that exercises a successful create will fail with `ErrInvalidColumnID`. Build-QA falsification would surface this — but the planning DEFECT is upstream.
- **Conclusion:** CONFIRMED. The plan's explicit "ColumnID (omit)" instruction is wrong.
- **Disposition (ABSORB in round 2):**
  - Replace D3 ContextBlock line 369's "ColumnID (omit)" with a `constraint severity=critical` block: "`till action_item create` MUST resolve a default ColumnID before calling `(*Service).CreateActionItem`. The canonical pattern is `svc.ListColumns(ctx, projectID, false)` → pick first column → pass `column.ID` as `CreateActionItemInput.ColumnID`. Empty ColumnID rejects with `domain.ErrInvalidColumnID` via `domain.NewActionItem`."
  - Add a RiskNote: "If the CLI accepts a `--column-id` flag (REVISION_BRIEF §2.9 does NOT enumerate it; today's `till action_item` mutation surface has none), default to the project's first column. Builder verifies via LSP whether `(*Service).ListColumns` is the right public API or whether a helper like `firstColumnForProject` needs CLI-side equivalent reimplementation."
  - Add a test row to `TestRunActionItemCreate_*`: success path verifies the action item lands on the project's first column (no `--column-id` flag passed). Optional sub-row: explicit `--column-id` flag if the L2 builder chooses to expose it.

#### FF3 — R3-NIT6 verbatim string drifts between L1 PLAN.md and W3 PLAN.md (extra trailing period)

- **Premises:** W3 PLAN.md AC8 + D6 RiskNote + D6 ContextBlock all declare the `--force` warning text MUST match VERBATIM (case-sensitive, exact string). Build-QA will check this. The verbatim text must therefore match L1 PLAN.md's authoritative quotation.
- **Evidence (verbatim quotes):**
  - L1 PLAN.md line 317 (R3-NIT6 source-of-truth):
    > `Overwrites destination files; any post-bootstrap customization is lost. Use `till agents save --from-project <id>` to push customization back to HOME tier before re-running bootstrap with `--force``
    — ends with closing backtick around `--force`, **NO trailing period.**
  - W3 PLAN.md line 43 (AC8):
    > `Overwrites destination files; any post-bootstrap customization is lost. Use `till agents save --from-project <id>` to push customization back to HOME tier before re-running bootstrap with `--force`.`
    — ends with `--force`. **PERIOD after the closing backtick.**
  - W3 PLAN.md line 568 (D6 AC bullet): identical trailing-period drift.
  - W3 PLAN.md lines 582-587 (D6 ContextBlock `constraint severity=critical`):
    > "Overwrites destination files; any post-bootstrap customization is lost. Use `till agents save --from-project <id>` to push customization back to HOME tier before re-running bootstrap with `--force`."
    — same trailing-period drift.
- **Trace:** Build-QA will check the SOURCE-OF-TRUTH (L1) verbatim text. The L2 plan internally drifts from L1. If the builder copies L2's text into the cobra flag's Long help, build-QA matches L2 (the period is present) but L1 contract check FAILS. If the builder catches the drift and copies L1's text, the L2 plan's own `TestRunAgentsBootstrap_ForceHelpTextContainsWarning` test (per KindPayload line 627) — which the builder will write against the L2 plan's verbatim — would assert the period-form and FAIL against the no-period help string. Either way, the planning artifact is internally inconsistent.
- **Conclusion:** CONFIRMED. The L2 plan must match L1 character-for-character on the warn string.
- **Disposition (ABSORB in round 2):**
  - In W3 PLAN.md, replace every occurrence of the `--force` warning text (AC8 line 43, D6 AC line 568, D6 ContextBlock lines 583-587, D6 RiskNote referencing the verbatim) with the L1-authoritative form ending at the closing backtick (no period after the closing backtick).
  - Update `TestRunAgentsBootstrap_ForceHelpTextContainsWarning`'s shape_hint to specify the exact substring search: `strings.Contains(longHelpText, "Overwrites destination files; any post-bootstrap customization is lost. Use \`till agents save --from-project <id>\` to push customization back to HOME tier before re-running bootstrap with \`--force\`")` — note absence of trailing period in the assertion.

### REFUTED counter-attacks (counterexamples I tried that did NOT land)

- **R1 (attempted FF): D3 `--blocked-by` "passed through" via `CreateActionItemInput` — but no top-level BlockedBy field.** REFUTED: W3 PLAN.md line 372-378 ContextBlock `warning severity=high` already names this risk explicitly: "BlockedBy on CreateActionItemInput: check LSP on app.CreateActionItemInput to confirm whether blocked_by is a direct field or must be set via UpdateActionItem after create. If absent from CreateActionItemInput, the --blocked-by flag adds a post-create UpdateActionItem call. Verify BEFORE writing." Live-code probe confirms `BlockedBy` lives at `ActionItemMetadata.BlockedBy []string` (`workitem.go:195`), reachable via `CreateActionItemInput.Metadata.BlockedBy`. The plan offers the builder a verify+choose-path, which is adequate planning. Flagged as NIT1 below for clarity-of-instruction improvement, not as FF.
- **R2 (attempted FF): `till project rename` via UpdateProject — does it actually re-slug?** REFUTED: `project.go:309-353` (UpdateDetails) unconditionally sets `p.Slug = normalizeSlug(name)` on every call. So `(*Service).UpdateProject` with a new `Name` correctly re-slugs. The W3 plan's instruction at D2 (line 277) — "till project rename MUST call `(*Service).UpdateProject` ... new Name" — is correct.
- **R3 (attempted FF): Cycle in blocker graph.** REFUTED: D1 → D2 → D3 → D4 → D5 → D6 → D7 is strictly linear. No cycle possible.
- **R4 (attempted FF): _BLOCKERS.toml drifts from PLAN.md inline blockers.** REFUTED: PLAN.md inline `Blocked by:` lines (D2 lines 261, D3 line 330, D4 line 409, D5 line 485, D6 line 553, D7 line 642) exactly match `_BLOCKERS.toml` rows. D1's "none within W3" omission matches `_BLOCKERS.toml`'s absence of a D1 row.
- **R5 (attempted FF): D6 hard-coded `[]string{"go", "fe", "gen"}` conflicts with W4.D1 canonical names.** REFUTED: sibling W2.D1 PLAN.md line 44 finalizes `allowedInitGroups = ["gen", "go", "fe"]` (same names without `till-` prefix per W4.D1). W3 is transitively blocked by W4.D1 via W1, so names are confirmed by W3 dispatch time.
- **R6 (attempted FF): D6 missing-file reporting path test coverage.** REFUTED: KindPayload line 625 includes `TestRunAgentsBootstrap_MissingFileReport` explicitly. AC line 565 specifies the behavior. Covered.
- **R7 (attempted FF): D7 cobra registration could blow the 120 LOC budget (16 subcommands × ~8-15 LOC each = 130-240 LOC).** REFUTED-AS-NIT: D7 RiskNote line 659-662 acknowledges main.go size + LSP usage, and the per-cobra-cmd LOC budget is realistic if helper functions are reused (existing patterns suggest 5-8 LOC per registration). Flagged as NIT5 below for D7a/D7b conditional-split coverage.
- **R8 (attempted FF): D5 + D6 could merge.** REFUTED: D5 ContextBlock line 519-522 explicitly justifies the split (D6 has substantial complexity: 2-into-4 fan-out, missing-file reporting, force-warning, orchestrator-managed.md starter; merging would blow LOC budget). Defensible split.
- **R9 (attempted FF): PLAN-QA-DISCIPLINE-R1 violation — some AC has no shipping test.** REFUTED: every AC (AC1 through AC10) maps to a `TestRunXxx*` row in the KindPayload of the droplet that owns the corresponding subcommand. Coverage map verified above in "QA Proof" section.
- **R10 (attempted FF): PLAN-QA-DISCIPLINE-R2 numeric drift — 16 commands narrated vs 7 droplets.** REFUTED: counting is internally consistent. Scope recap (line 24-30) = 1+4+1+5+4+1 = 16. CompletionCriteria (line 167) "All 15 + 1 subcommands". KindPayload sums to 16 subcommands across D1-D6 + main.go registration in D7. Math checks out.

### NIT (cosmetic / clarification / low-risk drifts — first-class per `feedback_nits_are_first_class.md`; default ABSORB unless explicit reason)

#### NIT1 — D3 `--blocked-by` instruction is open-ended; clarify the canonical path

- **Evidence:** W3 PLAN.md line 372-378 ContextBlock says "check LSP ... if absent, the --blocked-by flag adds a post-create UpdateActionItem call." Live-code probe shows `ActionItemMetadata.BlockedBy []string` IS the storage seat (`workitem.go:195`), reachable via `CreateActionItemInput.Metadata.BlockedBy`. The plan's open-ended "verify" language could lead the builder down an unnecessary post-create UpdateActionItem path when a pre-create Metadata injection works.
- **Disposition:** ABSORB — replace the open-ended ContextBlock with: "BlockedBy is wired via `CreateActionItemInput.Metadata.BlockedBy []string` (the `ActionItemMetadata` struct field at `internal/domain/workitem.go:195`). The `--blocked-by` flag accumulates `[]string` values into `opts.blockedBy`; builder merges them into the constructed `domain.ActionItemMetadata.BlockedBy` before calling `(*Service).CreateActionItem`. No post-create UpdateActionItem is needed."

#### NIT2 — D7 RiskNote claims "main.go is 163K LOC" — actual is ~4069 LOC (163KB file size, not 163K LOC)

- **Evidence:** W3 PLAN.md line 659: "`main.go` is 163K LOC — the largest file in cmd/till." Probe via `wc -l cmd/till/main.go` returns 4069 lines.
- **Disposition:** ABSORB — correct line 659 to: "`main.go` is ~4,069 LOC (163KB file size — the largest file in cmd/till). Use LSP `goToDefinition` on existing `projectCmd` / `actionItemCmd` / `rootCmd` to locate insertion points."

#### NIT3 — `CONSUMER-TIE TEST CONTRACT` wording inconsistency

- **Evidence:** W3 PLAN.md lines 90-92 (top-level ContextBlock):
  > `[constraint severity=high]
  CONSUMER-TIE TEST CONTRACT: every run function signature follows the existing pattern:
    func runXxx(ctx context.Context, svc *app.Service, opts xxxOptions, stdout io.Writer) error
  Tests exercise via run(ctx, args, &out, io.Discard). No direct cobra exec in tests.`
  
  The second sentence's `run(ctx, args, &out, io.Discard)` shape (with positional `args []string`) does NOT match the first sentence's `runXxx(ctx, svc, opts, stdout)` shape. Either tests exercise `runXxx` directly (correct per Drop 4c.6 W2 precedent) OR they exercise a top-level `run(args)` that dispatches via cobra (which contradicts "No direct cobra exec in tests").
- **Disposition:** ABSORB — replace the second sentence with: "Tests exercise each `runXxx` directly with a constructed options struct + `bytes.Buffer` stdout + `io.Discard` stderr. No cobra exec; option-struct construction in tests mirrors the cobra flag-binding shape."

#### NIT4 — D7 LOC budget not acknowledged

- **Evidence:** D7 wires 16 subcommands × ~5-15 LOC per cobra registration (cobra `Command` struct + flag bindings + `RunE` closure). Conservative midpoint: 16 × 10 = 160 LOC — over the 120 LOC ceiling. D2, D4, D6 each acknowledge a conditional split (D2a/D2b, D4a/D4b, D6a/D6b). D7 does NOT.
- **Disposition:** ABSORB — add to D7 RiskNotes: "If `main.go` modifications + new `main_test.go` test exceed 120 LOC production, split D7a (project + action_item subcommand registrations) + D7b (template + agents subcommand registrations + smoke test). D7b blocked_by D7a since both touch `main.go`."

#### NIT5 — D2's `cliMutationContext` requirement implicit, not stated

- **Evidence:** Pattern probe `project_cli.go:142`: `ctx = cliMutationContext(ctx, cfg)` is the canonical pre-mutation step on every existing project-mutating CLI. D2's RiskNotes do not mention this — builder might omit it and the mutating subcommands would write UpdatedBy/UpdatedByName/UpdatedType as zero values.
- **Disposition:** ABSORB — add a one-liner D2 ContextBlock `reference`: "Every mutating subcommand (delete/archive/restore/rename) MUST call `cliMutationContext(ctx, cfg)` before any `(*Service)` call — pattern at `project_cli.go:142`. Resulting ctx carries the active CLI identity for UpdatedBy/UpdatedByName/UpdatedType audit fields."

#### NIT6 — D4 + D5 embedded FS API ambiguity could be tightened

- **Evidence:** D4 RiskNote line 429 + D5 RiskNote line 503 both say "use `internal/templates` embedded FS" + "Check via LSP for the function that ... Builder MUST verify the actual function name via LSP before writing." Live-code probe shows `templates.LoadDefaultTemplateForLanguage(lang)` returns parsed `Template`, NOT raw bytes — but `templates.DefaultTemplateFS` (`embed.FS`, line 104 of embed.go) is exported, so `templates.DefaultTemplateFS.ReadFile("builtin/till-<group>.toml")` returns the raw TOML for `till template show --source embedded` / `diff`; agent reads use `DefaultTemplateFS.ReadFile("builtin/agents/<group>/<name>.md")`.
- **Disposition:** ABSORB — tighten D4 + D5 RiskNotes to: "Use `templates.DefaultTemplateFS.ReadFile(path)` for raw-bytes access. Path forms: `builtin/till-<group>.toml` for templates, `builtin/agents/<group>/<name>.md` for agents. `LoadDefaultTemplateForLanguage` returns the parsed Template — not what `show --source embedded` / `diff` need (those need raw bytes for content display + diff)."

---

## 2. Counterexamples

CONFIRMED counterexamples (3): FF1 (silent `--add-group`/`--remove-group` drop), FF2 (ColumnID omit vs domain reject), FF3 (R3-NIT6 verbatim drift).

REFUTED self-attacks (10): documented above. NIT-class concerns (6): documented above.

**Cross-wave concerns surfaced:** FF1 + FF2 are cross-wave / cross-axis attacks (FF1 against the W1.D2 ship contract; FF2 against the domain layer's invariant — not a cross-wave contract per se but a cross-package contract). FF3 is intra-doc verbatim drift but compares against L1.

---

## 3. Summary

**Verdict: FAIL.**

- **3 CONFIRMED FFs (must absorb in round 2):** FF1 missing flag pair, FF2 ColumnID omit, FF3 verbatim drift.
- **6 NITs (default ABSORB):** NIT1 (`--blocked-by` clarification), NIT2 (LOC vs KB factual error), NIT3 (CONSUMER-TIE wording), NIT4 (D7 LOC budget split), NIT5 (`cliMutationContext` reference), NIT6 (embedded FS API tightening).
- **Cross-wave concerns:** FF1 surfaces a cross-wave contract dependency that needs the W3 plan to commit to the post-W1.D2 state. No new cross-wave edge needed (wave-level `blocked_by W1` already covers it).
- **No structural attacks landed:** no cycles, no missing `blocked_by` edges between siblings sharing files/packages, no `_BLOCKERS.toml` vs PLAN.md drift, no YAGNI / over-decomposition findings.

The 7-droplet linear chain + package-lock serialization is structurally sound. The atomicity, the wave-level wiring, the test-shipper contract, and the `_BLOCKERS.toml` mirror are all correct. The defects are in three places: a missing flag pair, a wrong API-usage instruction, and a verbatim-string drift — all surgical edits, no architectural rework.

---

## TL;DR

- **T1:** Verdict FAIL — 3 FFs + 6 NITs require round-2 absorption before W3 dispatch.
- **T2:** FF1: W3.D1 silently drops L1's `--add-group`/`--remove-group` requirement; W1.D2 ships `ProjectMetadata.Groups` BEFORE W3, so the "if absent, omit" hedge is dead. FF2: W3.D3 says "ColumnID (omit)" but `domain.NewActionItem` rejects empty ColumnID — builder must resolve a default column via `ListColumns`. FF3: R3-NIT6 verbatim text drifts (W3 plan adds a trailing period after `` `--force` `` that L1 does not have); the plan's own VERBATIM contract requires character-exactness.
- **T3:** No cycles, no missing blockers, no `_BLOCKERS.toml` drift, no YAGNI. Linear D1→D7 chain on `cmd/till` package compile-lock is structurally sound. NITs cover D7 LOC budget, embedded-FS API clarity, `cliMutationContext` reference, `--blocked-by` routing-path, factual LOC-vs-KB error, and CONSUMER-TIE wording.
