# PLAN_QA_PROOF — DROP_4c.6.1.W3_CLI_SURFACE (Round 2)

**Verdict:** PASS with 1 NIT (CONSUMER-TIE signature mismatch in D4/D5 KindPayload). All Round-1 FFs (Proof FF1, Fals FF1/FF2/FF3) and Round-1 NITs (Proof NIT1-NIT6, Fals NIT1-NIT6) absorbed or correctly deferred-with-reason. PLAN structure, blocker graph, droplet count, R3-NIT6 verbatim, FF4 smart-default coverage, ColumnID resolution constraint, and CONSUMER-TIE TEST CONTRACT all clean. The remaining NIT is a minor self-inconsistency between the top-level CONSUMER-TIE signature shape and the non-options run-function signatures declared in D4/D5 KindPayload — non-blocking for build dispatch because the builder can resolve it inline.

**Scope:** L2 plan for Wave W3 (CLI Surface) — 15 new CLI subcommands + `till agents bootstrap`, fully serial inside `cmd/till` package. D1→D2→D3→D4→D5→D6→D7. R10 cross-cutting (W1.D2 typed `Groups` field) reflected.

**Mode:** Filesystem-MD-only. Hylla OFF. Evidence from `Read` against `internal/app/service.go`, `internal/domain/*.go`, sibling L2 plans (W1, W2), and L1 PLAN.md.

---

## 0. Verification Checklist Summary

| Check | Result |
|---|---|
| 1. Every claim backed by evidence | PASS |
| 2. Every acceptance bullet testable | PASS |
| 3. Trace covers all acceptance bullets | PASS (D1–D7 maps 1:1 to scope) |
| 4. `blocked_by` graph acyclic + shared-file pairs gated | PASS (linear D1→…→D7 mirrored in `_BLOCKERS.toml`) |
| 5. PLAN-QA-DISCIPLINE-R1 (every NEW behavior AC → test-runner) | PASS |
| 6. PLAN-QA-DISCIPLINE-R2 (narrated count = enumerated count) | PASS (7 narrated, 7 enumerated D1–D7) |
| 7. `_BLOCKERS.toml` mirrors PLAN.md | PASS |
| 8. CONSUMER-TIE contract documented per droplet | PASS at top-level; see NIT-1 for D4/D5 sub-signatures |
| 9. R3-NIT6 `--force` warning VERBATIM (no trailing period after `` `--force` ``) | PASS (all 4 occurrences clean) |
| 10. FF4 smart-default mapping test enumerates all 12 kinds | PASS (line 443) |
| 11. ColumnID resolution constraint (`severity=critical`) | PASS (D3 ContextBlock lines 402-409) |
| 12. `--add-group`/`--remove-group` wired to typed `Metadata.Groups` (R10-D2) | PASS (D1 lines 240, 250-256, 277) |
| 13. R10-D1 canonical groups (`go|fe|gen`, no `till-` prefix) | PASS (D5 ContextBlock line 566) |
| 14. Round-2 absorption explicitly logged for every Round-1 finding | PASS (lines 14-31) |

---

## 1. Findings (FF — block build dispatch)

None.

---

## 2. NITs (low-severity; absorb in next round if cycling, otherwise inline at build time)

### 2.1 [Axis: spec-conformance] [severity: low] D4 + D5 non-options run-function signatures diverge from top-level CONSUMER-TIE contract

**Claim.** Top-level ContextBlock (line 112-116) declares the universal CONSUMER-TIE contract:

```
func runXxx(ctx context.Context, svc *app.Service, cfg config.Config, opts xxxOptions, stdout io.Writer) error
```

D4 + D5 KindPayload declare two distinct shapes:
- `runTemplateSave`/`runAgentsSave` use the 5-param canonical shape `(ctx, svc, cfg, opts, stdout)` (lines 512, 580).
- `runTemplateList`/`runTemplateShow`/`runTemplateDiff`/`runTemplateRestore` (lines 513-516) and `runAgentsList`/`runAgentsShow`/`runAgentsDiff` (lines 581, 583, 585) use positional-param shapes that drop `ctx`, `svc`, `cfg`, and the options struct, e.g. `runTemplateList(homeDir string, stdout io.Writer) error`.

The contract says "every run function signature follows the existing pattern" — without a carve-out for pure file-I/O commands. The plan internally contradicts itself.

**Evidence.**
- `workflow/drop_4c_6_1/DROP_4c.6.1.W3_CLI_SURFACE/PLAN.md:112-116` (universal contract).
- `workflow/drop_4c_6_1/DROP_4c.6.1.W3_CLI_SURFACE/PLAN.md:513-516` (D4 divergent shapes).
- `workflow/drop_4c_6_1/DROP_4c.6.1.W3_CLI_SURFACE/PLAN.md:581, 583, 585` (D5 divergent shapes).

**Trace.** Builder reads the top-level contract, sees the universal signature with `opts` struct, then opens D4 KindPayload and finds `runTemplateList(homeDir string, stdout io.Writer)` — no opts struct, no ctx, no svc, no cfg. The builder must either: (a) follow the top-level contract and add unused `ctx`/`svc`/`cfg`/`opts` params, (b) follow the KindPayload shape and break the universal contract, or (c) introduce options structs for list/show/diff/restore and align with the contract. Neither path is named in the plan.

**Conclusion.** Non-blocking — the builder will pick (c) since the cobra flag-binding shape naturally produces an options struct, and that's the cleanest path. But the plan should explicitly say so (or carve out file-I/O-only commands from the contract).

**Fix hint.** One of:
- **Option A** (preferred — minimal edit): in D4 + D5 KindPayload, normalize every `runXxx` shape_hint to `func runXxx(ctx, svc, cfg, opts, stdout) error` with the options struct carrying `homeDir`/`group`/`agentName`/`source` etc. as struct fields. This keeps the universal CONSUMER-TIE contract honest. Cost: minor shape_hint rewrites; no architectural change.
- **Option B** (carve-out): amend the top-level ContextBlock to add: "Pure file-I/O run functions that take no service dependency may omit `ctx`/`svc`/`cfg` and use positional params for the small inputs (homeDir, group, agentName, source); they still take `stdout io.Writer` last for test injection."

Builder can resolve inline at build time — neither path changes the spec, just the signature shape. Flagged here so build-QA-falsification doesn't surface it as a FF later.

---

## 3. Verified Round-1 Absorptions (audit trail)

### Proof FF1 / Fals FF1 — `--add-group`/`--remove-group` flag pair via typed `Metadata.Groups`

- **Status:** RESOLVED per R10-D2.
- **Evidence:**
  - W3 PLAN.md line 240 (D1 RiskNote): "W1.D2 ships this typed field on `domain.ProjectMetadata` BEFORE W3 dispatches. Builder MUST verify via LSP `goToDefinition` on `domain.ProjectMetadata` that the `Groups []string` field exists before writing. If (unexpectedly) absent, STOP and return to orchestrator rather than adding a TODO stub."
  - W3 PLAN.md line 250-256 (D1 ContextBlock, `severity=high`): the four-step typed-Groups wiring (read existing → add/dedup or remove → preserve all other Metadata fields → pass to UpdateProjectInput).
  - W3 PLAN.md line 277 (D1 KindPayload): `addGroups []string; removeGroups []string` on `projectUpdateCommandOptions`.
  - W1 sibling PLAN.md line 36 + 58-63: "domain.ProjectMetadata gains a `Groups []string` field … D2 acceptance reaffirmed as shipping typed `ProjectMetadata.Groups []string` field; downstream W2/W3 consume the typed field directly."
  - Live-code check `internal/domain/project.go:115-155`: `ProjectMetadata` today lacks `Groups`; W1.D2 ships it. Wave-level `blocked_by W1` enforces ordering.
  - The TODO fallback hedge from Round-1 is REMOVED. Replaced with STOP-and-return-to-orch on unexpected absence.

### Fals FF2 — D3 ColumnID resolution constraint

- **Status:** RESOLVED.
- **Evidence:**
  - W3 PLAN.md line 402-409 (D3 ContextBlock, `severity=critical`): the canonical pattern using `(*Service).ListColumns(ctx, projectID, false)` → first column → `column.ID`. Cites `auto_generate_steward.go:123-158` + `service.go:1870`.
  - W3 PLAN.md line 425-426 (D3 ContextBlock `reference`): "ColumnID (required — resolved from ListColumns)" — replaces Round-1's "ColumnID (omit)".
  - W3 PLAN.md line 388 (D3 RiskNotes): `domain.NewActionItem` rejects empty `ColumnID` with `ErrInvalidColumnID`.
  - Live-code check `internal/domain/action_item.go:309-311`: empty `ColumnID` returns `ErrInvalidColumnID`. Confirmed.
  - Live-code check `internal/app/service.go:1870`: `ListColumns` returns sorted by `Position` ascending. Confirmed.

### Fals FF3 / Proof NIT1 — R3-NIT6 verbatim trailing-period drift

- **Status:** RESOLVED.
- **Evidence:**
  - W3 PLAN.md line 64 (AC8): ends with `` `--force`" `` (closing backtick, then closing quote — NO period inside).
  - W3 PLAN.md line 619 (D6 AC bullet): same form.
  - W3 PLAN.md lines 633-639 (D6 ContextBlock): same form. Line 638 explicitly states: "The string ends with the closing backtick — NO trailing period after the backtick."
  - W3 PLAN.md line 679 (D6 test shape_hint): same form. Notes "no trailing period after closing backtick".
  - L1 PLAN.md line 338 (authoritative source): ends with `` `--force`" `` (closing backtick, closing quote, NO period).
  - All 4 occurrences now match L1 line 338 byte-for-byte.

### Proof NIT2 — D3 smart-default test enumerates all 12 kinds

- **Status:** RESOLVED.
- **Evidence:** W3 PLAN.md line 443 `TestRunActionItemCreate_StructuralTypeSmartDefault` shape_hint enumerates: `plan→segment, refinement→segment, build→droplet, research→droplet, plan-qa-proof→droplet, plan-qa-falsification→droplet, build-qa-proof→droplet, build-qa-falsification→droplet, closeout→droplet, commit→droplet, discussion→droplet, human-verify→droplet; plus explicit-override-valid (confluence), explicit-override-invalid`. All 12 kinds covered.

### Proof NIT3 — D6 fan-out test enumerates all 4 destinations

- **Status:** RESOLVED.
- **Evidence:** W3 PLAN.md line 674 `TestRunAgentsBootstrap_QAFanOut` shape_hint: "verifies all 4 fan-out destination files written with identical-to-source content: `<group>/plan-qa-proof-agent.md`, `<group>/build-qa-proof-agent.md`, `<group>/plan-qa-falsification-agent.md`, `<group>/build-qa-falsification-agent.md`". W3 PLAN.md line 622 (D6 AcceptanceCriteria fan-out bullet) enumerates all 4 destinations.

### Proof NIT4 — pass-through flags test added

- **Status:** RESOLVED.
- **Evidence:** W3 PLAN.md line 445 `TestRunActionItemCreate_PassThroughFlags` shape_hint covers all 7 flags (paths, packages, files, blocked-by, metadata-json, parent-id, role); specifically calls out `--blocked-by` → `Metadata.BlockedBy` and `ColumnID` auto-resolved to first column.

### Proof NIT5 — service.go line citations

- **Status:** DEFERRED-WITH-REASON-CORRECT.
- **Evidence:**
  - Round-2 plan line 24 documents the deferral: "round-1 line citations (625, 669, 689, 709, 1035) are correct — they cite the `func` keyword lines. The proof-agent cited comment lines (624, 668, etc.) as 'actual.' Standard Go practice cites the func signature line."
  - Independent LSP-equivalent Read verification:
    - `service.go:624` = `// UpdateProject updates state for the requested operation.`
    - `service.go:625` = `func (s *Service) UpdateProject(...)`.
    - `service.go:668` = `// ArchiveProject archives one project.`
    - `service.go:669` = `func (s *Service) ArchiveProject(...)`.
    - `service.go:688` (NOT read but inferred) = comment; line 689 = `func (s *Service) RestoreProject(...)`.
    - `service.go:709` = `func (s *Service) DeleteProject(...)` (confirmed from Read of 700-720 block).
    - `service.go:1035` = `func (s *Service) CreateActionItem(...)`.
    - `service.go:1870` = `func (s *Service) ListColumns(...)`.
  - The Round-1 proof-agent's "off-by-one" claim was incorrect; Round-2's deferral reasoning is sound. Standard Go citation practice cites the func keyword line.

### Proof NIT6 / Fals NIT1c — `--role pre-Drop-2` qualifier dropped

- **Status:** RESOLVED.
- **Evidence:** W3 PLAN.md line 389 (D3 RiskNote): "`--role` is optional (closed enum; empty is valid)." No "pre-Drop-2" qualifier. Matches service.go:742-746 doc-comment: "Empty string is permitted; non-empty values must match the closed Role enum" (no temporal qualifier).

### Fals NIT1 — `--blocked-by` routing explicit

- **Status:** RESOLVED.
- **Evidence:** W3 PLAN.md line 417-421 (D3 ContextBlock `reference`): "BlockedBy is wired via `CreateActionItemInput.Metadata.BlockedBy []string` (the `ActionItemMetadata` struct field at `internal/domain/workitem.go:195`, reachable via `CreateActionItemInput.Metadata`). The `--blocked-by` flag accumulates `[]string` values into `opts.blockedBy`; builder merges them into the constructed `domain.ActionItemMetadata.BlockedBy` before calling `(*Service).CreateActionItem`. No post-create UpdateActionItem is needed."
- Live-code check `internal/domain/workitem.go:195`: `BlockedBy []string \`json:"blocked_by"\`` confirmed on `ActionItemMetadata`.

### Fals NIT2 — D7 LOC factual correction

- **Status:** RESOLVED.
- **Evidence:** W3 PLAN.md line 711 (D7 RiskNotes): "`main.go` is ~4,069 LOC (163KB file size — the largest file in cmd/till)." `wc -l cmd/till/main.go` confirms 4069 lines. Matches.

### Fals NIT3 — CONSUMER-TIE wording unified

- **Status:** RESOLVED at top level.
- **Evidence:** W3 PLAN.md lines 112-116: "Tests exercise each `runXxx` directly with a constructed options struct + `bytes.Buffer` stdout + `io.Discard` stderr. No cobra exec in tests; option-struct construction in tests mirrors the cobra flag-binding shape." The Round-1 self-contradictory `run(ctx, args, &out, io.Discard)` formulation is removed.
- **Caveat:** new NIT-1 above flags the D4/D5 sub-signature divergence from this top-level shape — that's a residual seam between the top-level contract and the per-droplet KindPayload, not a re-introduction of the Round-1 wording bug.

### Fals NIT4 — D7 LOC budget conditional split

- **Status:** RESOLVED.
- **Evidence:** W3 PLAN.md line 715 (D7 RiskNotes): "If D7's cobra registrations + test exceed 120 LOC production code, split: D7a covers project + action_item subcommand registrations; D7b covers template + agents subcommand registrations + smoke test. D7b blocked_by D7a since both touch `main.go`."

### Fals NIT5 — D2 `cliMutationContext` explicit reference

- **Status:** RESOLVED.
- **Evidence:** W3 PLAN.md lines 326-331 (D2 ContextBlock `reference`): "Every mutating subcommand (delete/archive/restore/rename) MUST call `cliMutationContext(ctx, cfg)` before any `(*Service)` call — pattern at `project_cli.go:142`. Resulting ctx carries the active CLI identity for UpdatedBy/UpdatedByName/UpdatedType audit fields." Also added to D1 ContextBlock at lines 267-270.

### Fals NIT6 — D4/D5 embedded FS API path forms

- **Status:** RESOLVED.
- **Evidence:**
  - W3 PLAN.md line 480 (D4 RiskNote): "use `templates.DefaultTemplateFS.ReadFile(path)` for raw-bytes access (path form: `builtin/till-<group>.toml`). Do NOT use `templates.LoadDefaultTemplateForLanguage` — that returns a parsed Template struct, not raw bytes".
  - W3 PLAN.md line 493-496 (D4 ContextBlock `reference`): same.
  - W3 PLAN.md line 554 (D5 RiskNote): "use `templates.DefaultTemplateFS.ReadFile("builtin/agents/<group>/<name>.md")` for raw-bytes access".
  - W3 PLAN.md line 565-567 (D5 ContextBlock `reference`): same with canonical-group-name note (no `till-` prefix per R10-D1).

---

## 4. Cross-Planner / Cross-Wave Observations

- **R10-D2 typed Groups field**: W1.D2 ships `ProjectMetadata.Groups []string` BEFORE W3 dispatches. Round-2 PLAN's D1 absorption is consistent with W1's Round-2 PLAN line 36 ("D2 acceptance reaffirmed as shipping typed `ProjectMetadata.Groups []string` field; downstream W2/W3 consume the typed field directly"). The TODO fallback hedge is dead code; correctly removed.
- **R10-D1 canonical group names**: W4.D1 ships the `git mv till-go → go` and `till-gen → gen` renames. W3.D5 ContextBlock line 566 references the post-rename canonical names (`go`, `fe`, `gen` — no `till-` prefix). W3 transitively depends on W4.D1 via W1's wave-level dep. Order is sound.
- **W7.D3 also touches `cmd/till/main.go`**: noted in Round-1 cross-planner observations; still applies. W3 is Wave D (last); W7.D3 is Wave C. W3.D7's `main.go` modifications come AFTER W7.D3's deletion of `till serve` cobra wiring. No collision risk.
- **W2.D1 `allowedInitGroups = ["gen", "go", "fe"]`**: matches W3.D6's hard-coded known-groups list `{"go", "fe", "gen"}`. Cross-wave consistency PASS.

---

## 5. Decision

**Verdict: PASS** (with NIT-1 flagged for inline absorption at build time, OR addressed in a Round-3 absorb pass alongside any falsification findings).

**Round-1 → Round-2 absorption coverage:**
- Round-1 Proof: 1 FF + 6 NITs → ALL absorbed or correctly deferred-with-reason. (Proof NIT5 is the only deferral; reasoning verified correct.)
- Round-1 Falsification: 3 FFs + 6 NITs → ALL absorbed.

**Total Round-2 residue:** 1 NIT (NIT-1, CONSUMER-TIE sub-signature divergence). Non-blocking for build dispatch — builder can resolve inline by introducing options structs for D4/D5's list/show/diff/restore commands, or by carving out the file-I/O-only category in the top-level contract.

**Recommended next step:** orchestrator may either:
- Accept the Round-2 PLAN as-is and dispatch W3 builders (NIT-1 absorbed inline by builder following the universal contract via Option A — adding options structs).
- Run a Round-3 cycle if the parallel L2 plan-QA-falsification surfaces additional findings, absorbing NIT-1 alongside them.

---

## Hylla Feedback

N/A — action item touched non-Go files only (PLAN.md, _BLOCKERS.toml, REVISION_BRIEF.md, SKETCH.md). Per task brief: "Hylla is OFF". All Go-symbol verification went through `Read` against `internal/app/service.go` (UpdateProjectInput/UpdateProject/Archive/Restore/Delete + CreateActionItem + CreateActionItemInput + ListColumns), `internal/domain/action_item.go` (NewActionItem ColumnID + StructuralType invariants), `internal/domain/workitem.go` (ActionItemMetadata.BlockedBy), `internal/domain/project.go` (ProjectMetadata pre-W1.D2 baseline), `internal/domain/structural_type.go` (closed 4-enum), `internal/domain/role.go` (closed 9-enum), `internal/app/auto_generate_steward.go` (canonical ColumnID-resolution pattern at lines 123-158). No Hylla fallback to report.
