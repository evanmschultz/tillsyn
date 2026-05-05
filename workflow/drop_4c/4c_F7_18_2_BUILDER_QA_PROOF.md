# Drop 4c F.7.18.2 — Builder QA Proof Review

**Droplet:** F.7.18.2 — Schema-3: `Tillsyn` top-level struct
**Verdict:** **PROOF GREEN**
**Reviewer:** go-qa-proof-agent (read-only)

REVISIONS-aware: REV-3 of `F7_18_CONTEXT_AGG_PLAN.md` confines F.7.18.2 to the
initial 2-field `Tillsyn` declaration; F.7-CORE F.7.1 + F.7.6 will extend with
`SpawnTempRoot` + `RequiresPlugins` later. Master PLAN.md §5 + L14/L15 cited.

## 1. Findings

- 1.1 **Criterion 1 — `Tillsyn` struct has EXACTLY 2 fields. PASS.**
  - `internal/templates/schema.go` lines 226-239 declare `type Tillsyn struct`
    with two and only two exported fields:
    - line 231: `MaxContextBundleChars int \`toml:"max_context_bundle_chars"\``
    - line 238: `MaxAggregatorDuration Duration \`toml:"max_aggregator_duration"\``
  - No unexported fields, no embedded structs, no additional exported fields.
    REV-3 boundary respected verbatim.
  - Doc comment at lines 210-225 cites master PLAN §5 extension policy + L14/L15
    default-substitution semantics + REV-3 strict-decode-inheritance contract.

- 1.2 **Criterion 2 — `Template.Tillsyn Tillsyn` field with TOML tag
  `tillsyn`. PASS.**
  - `internal/templates/schema.go` line 183: `Tillsyn Tillsyn \`toml:"tillsyn"\``.
  - Doc comment at lines 176-182 explains the load-time strict-decode rationale
    (without the field, any `[tillsyn]` table would be rejected by step 3) and
    cross-references F.7-CORE extenders.

- 1.3 **Criterion 3 — `validateTillsyn` wired into validator chain after
  `validateAgentBindingContext`. PASS.**
  - `internal/templates/load.go` lines 132-138: chain runs
    `validateAgentBindingContext` (L133) → `validateTillsyn` (L136) in that
    order. Order matches the spawn prompt verbatim.
  - Package doc-comment validator-chain enumeration at lines 60-63 declares
    step `h. validateTillsyn` matching the implementation.
  - Function definition at lines 569-579: rejects negative
    `MaxContextBundleChars` (L570-573) and negative `MaxAggregatorDuration`
    (L574-577); zero passes through. Wraps `ErrInvalidTillsynGlobals` via
    `fmt.Errorf("%w: …")`.

- 1.4 **Criterion 4 — All 8 test scenarios exist. PASS.**
  - The spawn prompt's "7 new tests + strict-decode regression doubles as
    scenario 8" framing is satisfied by 7 functions in
    `internal/templates/load_test.go`:

    | # | Scenario | Test | File:line |
    |---|----------|------|-----------|
    | 1 | Happy path | `TestLoadTillsynHappyPath` | load_test.go:669-687 |
    | 2 | Empty `[tillsyn]` table | `TestLoadTillsynEmptyTableDecodes` | load_test.go:693-711 |
    | 3 | Omitted `[tillsyn]` table | `TestLoadTillsynOmittedTableZeroValue` | load_test.go:718-738 |
    | 4 | Reject negative `max_context_bundle_chars` | `TestLoadTillsynRejectsNegativeMaxContextBundleChars` | load_test.go:768-788 |
    | 5 | Reject negative `max_aggregator_duration` | `TestLoadTillsynRejectsNegativeMaxAggregatorDuration` | load_test.go:793-813 |
    | 6 | Allow zero values | `TestLoadTillsynZeroValuesAllowed` | load_test.go:744-763 |
    | 7 | Strict-decode unknown-key | `TestLoadTillsynStrictDecodeUnknownFieldRejected` | load_test.go:821-839 |
    | 8 | Round-trip | `TestTemplateTOMLRoundTrip` (extended) | schema_test.go:109-112 |

  - Round-trip extension at `schema_test.go` L109-112 populates
    `Tillsyn{MaxContextBundleChars: 200000, MaxAggregatorDuration: Duration(2 * time.Second)}`
    and the existing `reflect.DeepEqual` assertion at L129-131 covers the
    new field equality.

- 1.5 **Criterion 5 — Strict-decode unknown-key test asserts `[tillsyn]
  bogus_field = true` fails with `ErrUnknownTemplateKey`. PASS.**
  - `internal/templates/load_test.go` lines 821-839, function
    `TestLoadTillsynStrictDecodeUnknownFieldRejected`:
    - L825-827: TOML payload places `bogus_field = true` UNDER `[tillsyn]`
      alongside a valid `max_context_bundle_chars = 200000` field — proves the
      rejection fires inside the new struct, not from a sibling table.
    - L833-835: `errors.Is(err, ErrUnknownTemplateKey)` must hold.
    - L836-838: error message must contain literal `bogus_field` for UX.
  - REV-3 contract satisfied: future extenders (`SpawnTempRoot`,
    `RequiresPlugins`) inherit unknown-key rejection automatically because
    `load.go` step 3's `DisallowUnknownFields` decoder operates at the struct
    layer — adding a field to `Tillsyn` does not relax the rejection of any
    other key.

- 1.6 **Criterion 6 — NO additional fields beyond the 2 specified
  (REV-3 boundary). PASS.**
  - `schema.go` L226-239 inspected verbatim; struct body is
    `MaxContextBundleChars` + `MaxAggregatorDuration` only.
  - No additional methods on `Tillsyn` beyond what the embedded `Duration`
    type provides through composition.

- 1.7 **Criterion 7 — Scope: only `internal/templates/` files + worklog. PASS.**
  - `git status --porcelain` shows the modified set:
    - `internal/templates/load.go` (validator + sentinel)
    - `internal/templates/load_test.go` (7 new tests)
    - `internal/templates/schema.go` (struct + Template field)
    - `internal/templates/schema_test.go` (round-trip extension)
    - `workflow/drop_4c/4c_F7_18_2_BUILDER_WORKLOG.md` (new)
    - `workflow/drop_4c/SKETCH.md` (modified — out of droplet scope; NIT 2.1)
  - No leakage into `cmd/till`, `internal/app/dispatcher`, `internal/domain`,
    or other packages.

- 1.8 **Criterion 8 — `mage ci` green per worklog. PASS (worklog evidence
  trusted; not re-run).**
  - Worklog `4c_F7_18_2_BUILDER_WORKLOG.md` lines 70-79 reports:
    - tests 2320, passed 2319, failed 0, skipped 1
    - 21 packages, 21 passed
    - `internal/templates` coverage 96.0% (well above the 70% gate)
  - The single skip is identified as `TestStewardIntegrationDropOrchSupersedeRejected`,
    which is consistent with prior drop worklogs as a pre-existing skip.

- 1.9 **Sentinel error placement consistent with rationale. PASS.**
  - `load.go` L221-235 declares `ErrInvalidTillsynGlobals` as a top-level
    `errors.New("invalid tillsyn globals")` — NOT a nested
    `fmt.Errorf("%w: …", ErrInvalidAgentBinding)` like `ErrInvalidContextRules`
    (L219). Doc comment at L231-234 explicitly justifies the placement: the
    `[tillsyn]` table is global, not per-binding. Aligns with the worklog L23
    rationale.

- 1.10 **Validator chain doc-comment kept in sync with implementation. PASS.**
  - `load.go` L60-63 added step `h. validateTillsyn` — kept in stable
    a/b/c/d/e/f/g/h ordering. Matches both runtime call order (L115-138) and
    the worklog L26 claim of step-h numbering.

- 1.11 **Engine-time default values cited consistently. PASS.**
  - `load.go` L552-556 doc-comment claims engine-time defaults of 200000
    chars and 2s duration. These match the round-trip test populated values
    (`schema_test.go` L110-111: `200000` + `Duration(2 * time.Second)`) and the
    happy-path TOML literal (`load_test.go` L674-675: `200000` + `"2s"`). No
    drift between schema doc and tests.

## 2. Missing Evidence

- 2.1 **NIT — `workflow/drop_4c/SKETCH.md` modified outside this droplet's
  declared scope.** `git status` shows ` M workflow/drop_4c/SKETCH.md`. The
  spawn prompt's "deliverables" enumerated only `schema.go`, `load.go`,
  `schema_test.go`, `load_test.go`, and the worklog. SKETCH is presumably a
  shared scratch surface for the F.7.18 plan author; if it was edited as
  part of REV-3 authoring upstream, that's fine — but if F.7.18.2's builder
  modified it, it should appear in the worklog deliverables list. Not a
  blocker; ask the orchestrator whether SKETCH belongs in this droplet's
  diff or another's. No code impact.

- 2.2 **Round-trip test does not assert per-field equality — it relies on
  the existing `reflect.DeepEqual` at `schema_test.go` L129-131.** This is
  consistent with how the rest of the round-trip body works (Gates,
  AgentBindings, etc. are all DeepEqual'd), so the test is correct, but
  a future refactor that loosens the DeepEqual to selective field checks
  could quietly drop the new `Tillsyn` coverage. Not a blocker for this
  droplet — flag for the F.7-CORE F.7.1/F.7.6 extender to be aware that the
  round-trip is the only place struct-tag drift is caught.

## 3. Summary

PROOF GREEN — every spawn-prompt criterion is backed by file:line evidence in
the committed diff. The `Tillsyn` struct has exactly 2 fields per REV-3, the
validator is wired in the right chain position with the right sentinel, all
8 test scenarios exist with the correct assertions, and the strict-decode
regression test proves REV-3's "extenders inherit unknown-key rejection
automatically" contract today.

One non-blocking NIT: `workflow/drop_4c/SKETCH.md` is modified but not
declared in the worklog deliverables — orchestrator should confirm whether
that edit belongs to F.7.18.2 or a sibling droplet.

## Hylla Feedback

N/A — no Hylla queries performed; this review was a closed-scope read of
committed files in `internal/templates/` plus the droplet worklog. The
spawn prompt explicitly instructed "No Hylla calls."

## TL;DR

- T1 PROOF GREEN — every criterion proven by file:line; one non-blocking
  NIT on undeclared `SKETCH.md` edit.
- T2 No missing evidence; one cosmetic NIT on round-trip's reliance on
  package-level DeepEqual.
- T3 PROOF GREEN. Ready for parent `build` close once falsification sibling
  agrees.
