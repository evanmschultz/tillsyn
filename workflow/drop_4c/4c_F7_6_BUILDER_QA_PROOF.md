# Drop 4c F.7.6 — Builder QA Proof Review (Round 1)

**Verdict:** PROOF GREEN-WITH-NITS
**Reviewer:** go-qa-proof-agent (proof-completeness pass)
**Builder artifact:** `workflow/drop_4c/4c_F7_6_BUILDER_WORKLOG.md`
**Falsification sibling:** spawned in parallel (separate file)

## Scope Boundary (F.7.6 vs Sibling F.7.1)

REV-7 mandates Tillsyn-struct extension across multiple droplets. F.7.18.2 declared
the struct (committed as `e62c379`-era work, baseline at HEAD); F.7.1 (parallel,
uncommitted in this working tree) extends with `SpawnTempRoot`; F.7.6 (this
review) extends with `RequiresPlugins`. The working tree carries BOTH
uncommitted droplets' changes side-by-side. The `git status` snapshot at review
start shows the following files modified or new:

| File | F.7.6 contribution | F.7.1 contribution |
|------|--------------------|---------------------|
| `internal/templates/schema.go` | RequiresPlugins field + doc-comments | SpawnTempRoot field |
| `internal/templates/load.go` | validateTillsynRequiresPlugins + sentinel doc-comments | isValidTillsynSpawnTempRoot |
| `internal/templates/load_test.go` | 5 new test funcs (RequiresPlugins) | 3 new test funcs (SpawnTempRoot) |
| `internal/templates/schema_test.go` | none | round-trip test SpawnTempRoot field |
| `internal/app/dispatcher/spawn.go` | pre-flight call site (L235-240) | NewBundle / WriteManifest call sites |
| `internal/app/dispatcher/spawn_test.go` | none | TestBuildSpawnCommandWritesManifestJSON + bundle-root test |
| `internal/app/dispatcher/plugin_preflight.go` (NEW) | entire file | none |
| `internal/app/dispatcher/plugin_preflight_test.go` (NEW) | entire file | none |
| `internal/app/dispatcher/bundle.go` (NEW) | none | entire file |
| `internal/app/dispatcher/bundle_test.go` (NEW) | none | entire file |

The worklog's "Deliverables" section correctly enumerates only the F.7.6
contributions; it does NOT claim bundle.go, bundle_test.go, schema_test.go, or
spawn_test.go. NO contamination found.

## 1. Findings

### 1.1 Tillsyn struct extension (`schema.go`)
**EVIDENCE GREEN.** `internal/templates/schema.go:260-292` declares
`RequiresPlugins []string` with TOML tag `requires_plugins,omitempty` (L292).
Doc-comment at L260-291 cites the F.7.6 contract: bare-name vs scoped shapes,
matcher rules, empty/nil short-circuit, REV-7 history. Type-level Tillsyn
doc-comment at L210-225 already cites F.7.6 (added speculatively per worklog).

### 1.2 Validator wired into `validateTillsyn` (`load.go`)
**EVIDENCE GREEN.** `internal/templates/load.go:823-825` invokes
`validateTillsynRequiresPlugins(tpl.Tillsyn.RequiresPlugins)` from
`validateTillsyn` after the SpawnTempRoot enum check. The validator definition
at L850-883 enforces:

- L853-855: empty entry rejected with `"requires_plugins entry is empty"`.
- L857-860: whitespace via `strings.ContainsAny(entry, " \t\r\n")` — ASCII
  space/tab/CR/LF set per worklog claim.
- L861-864: more-than-one `@` rejected.
- L865-875: empty name before `@` and empty marketplace after `@` both rejected
  via segment slicing on `IndexByte(entry, '@')`.
- L876-879: case-sensitive within-list duplicate rejected.

All paths wrap `ErrInvalidTillsynGlobals`. Sentinel definition at L256-273 cites
F.7.6 acceptance.

### 1.3 Pre-flight package (`plugin_preflight.go`)
**EVIDENCE GREEN.** `internal/app/dispatcher/plugin_preflight.go:128-168`
defines `CheckRequiredPlugins(ctx, lister, required)` with:

- L153-155: empty/nil required → nil short-circuit before lister.
- L156-158: nil lister + non-empty required → `ErrInvalidSpawnInput`-wrapped.
- L159-167: lister-side errors propagate; `ErrMissingRequiredPlugins` wraps
  the formatted missing list with install instructions.

Sentinels at L107 (`ErrMissingRequiredPlugins`), L118 (`ErrClaudeBinaryMissing`),
L126 (`ErrPluginListUnparseable`) — all three present with distinguishable
docstrings.

`ClaudePluginLister` interface at L89-96 (single-method `List(ctx)`).
`execClaudePluginLister` at L258-299 shells out via
`exec.CommandContext(bounded, "claude", "plugin", "list", "--json")` (L268)
with `pluginPreflightTimeout = 5 * time.Second` (L44) wrapped via
`context.WithTimeout` (L265). Error routing:

- L274-280: `exec.ErrNotFound` → `ErrClaudeBinaryMissing`.
- L282-292: ctx cancel routed via `bounded.Err()`; non-zero exit names code +
  stderr tail; other wait failures wrapped raw.

`parseClaudePluginList` at L309-323 handles malformed JSON
(`ErrPluginListUnparseable` wrap at L320) and empty/whitespace stdout
(returns `(nil, nil)` at L311-317). Forward-compat: standard `json.Unmarshal`
with no `DisallowUnknownFields` — unknown JSON keys silently ignored per
encoding/json default. Validated by
`TestParseClaudePluginListForwardCompatUnknownFields` at L357-366 of the test
file.

`defaultClaudePluginLister` singleton at L328:
`var defaultClaudePluginLister ClaudePluginLister = execClaudePluginLister{}`.

`RequiredPluginsForProject` hook at L37: `var RequiredPluginsForProject func(domain.Project) []string`
with nil default. Doc-comment at L16-36 explains seam rationale and concurrency
constraint ("set before first spawn; reassigning under load is unsafe").

### 1.4 Wire-in `BuildSpawnCommand` call site (`spawn.go`)
**EVIDENCE GREEN.** `internal/app/dispatcher/spawn.go:235-240`:

```
if hook := RequiredPluginsForProject; hook != nil {
    required := hook(project)
    if err := CheckRequiredPlugins(context.Background(), defaultClaudePluginLister, required); err != nil {
        return nil, SpawnDescriptor{}, fmt.Errorf("dispatcher: plugin pre-flight for kind %q: %w", item.Kind, err)
    }
}
```

Placement is correct per claim: AFTER `rawBinding.Validate()` (L214-216) and
BEFORE `ResolveBinding(rawBinding)` (L246) and `lookupAdapter` (L248). Hook
gating on `!= nil` matches design choice; nil hook = pre-flight skipped (zero
exec cost). Wrap message preserves `errors.Is` routing through both
`ErrMissingRequiredPlugins` and `ErrClaudeBinaryMissing`.

Doc-comment at L218-234 explains the hook seam rationale + KindCatalog plumbing
deferral.

### 1.5 Test coverage (5 schema tests + 17 pre-flight tests)
**EVIDENCE GREEN-WITH-NITS.** Worklog claims "5 schema tests" — verified by
`git diff internal/templates/load_test.go`:

1. `TestLoadTillsynRequiresPluginsHappyPath` (bare + scoped via single-list test)
2. `TestLoadTillsynRequiresPluginsOmittedZeroValue`
3. `TestLoadTillsynRequiresPluginsEmptySliceAllowed`
4. `TestLoadTillsynRequiresPluginsRejectionTable` (8 sub-cases — `mage testFunc`
   reports 9 passes including parent)
5. `TestLoadTillsynRequiresPluginsCaseSensitiveDistinct`

Worklog claims "14+ pre-flight tests" — file inspection at
`internal/app/dispatcher/plugin_preflight_test.go` shows 17 top-level test
functions (L37, L51, L66, L88, L120, L149, L198, L250, L270, L284, L308, L330,
L357, L372, L400, L410, L425). Worklog table lists 17 entries — claim accurate
modulo the "14+" lower-bound phrasing in the prompt summary. **NIT 1.5.1**:
the prompt's "14+" floor matches the worklog table footer's listed count of
17, but the body text only enumerates the first ~14 categories — not a
correctness defect, just a presentation drift.

All 17 pre-flight tests pass (`mage testPkg ./internal/app/dispatcher` reports
250 total passing). All schema tests pass (`mage testPkg ./internal/templates`
reports 378 total passing).

### 1.6 `mage ci` green
**EVIDENCE GREEN.** Re-ran `mage ci` end-to-end during this review:
- All 23 packages green.
- `internal/templates` coverage: 97.0% (matches worklog).
- `internal/app/dispatcher` coverage: 74.0% (matches worklog).
- All packages above 70.0% gate.
- Build target produces `till` binary cleanly.

### 1.7 No commit by builder per REV-13
**EVIDENCE GREEN.** `git log --oneline -1` shows HEAD at `d3fbb14`
(F.7.17.5 wire-in commit, pre-existing). `git status --short` confirms 7 F.7.6
files (4 modified + 2 new + 1 worklog) plus F.7.1's parallel files all
uncommitted. Builder honored REV-13.

### 1.8 Scope: only F.7.6-attributable files touched
**EVIDENCE GREEN.** Cross-checked every uncommitted file against F.7.6 vs F.7.1
attribution — see Scope Boundary table above. Worklog "Deliverables" section
correctly enumerates F.7.6 work only; sibling files (bundle.go, bundle_test.go,
schema_test.go SpawnTempRoot row, spawn_test.go manifest tests) belong to
F.7.1's parallel droplet and are appropriately omitted.

### 1.9 Whitespace check covers only ASCII whitespace (NIT)
**NIT 1.9.1.** `validateTillsynRequiresPlugins` uses
`strings.ContainsAny(entry, " \t\r\n")` (load.go L857) — this catches ASCII
space/tab/CR/LF only, NOT unicode whitespace (NBSP ` `, IDEOGRAPHIC SPACE
`　`, etc.). The worklog's L25 description matches the implementation
faithfully ("(space/tab/CR/LF)"), so the documented contract is precise.
However, `unicode.IsSpace` would have caught a wider class of inputs without
ergonomic cost.

**Disposition:** ACCEPT AS DOCUMENTED. Plugin identifiers are conventionally
ASCII; unicode whitespace inside a plugin ID would itself be a more severe
authoring footgun than the validator catches. The worklog discloses the
ASCII-only scope, so adopters who hit a unicode-whitespace edge case have a
documented expectation to push back against. Not a gap — file under
"refinement candidate" if unicode plugin names ever surface.

### 1.10 Doc-comment cross-references (NIT)
**NIT 1.10.1.** `internal/templates/load.go:782` (validateTillsyn) doc-comment
correctly cites "F.7.18.2 + F.7-CORE F.7.1 + F.7-CORE F.7.6" — full chain.
`internal/templates/load.go:803-809` REV-7 footer correctly states the
extension policy and the strict-decode invariant. `validateTillsynRequiresPlugins`
doc-comment at load.go L829-849 cites F.7.6 directly. All cross-references
look consistent.

`schema.go:213` Tillsyn type-level doc-comment cites F.7.18.2 + F.7.1 + F.7.6
(via the broader "subsequent F.7-CORE droplets extend it" framing — slightly
elliptical but accurate).

**Disposition:** ACCEPT. No drift found.

## 2. Missing Evidence

### 2.1 Production lister `execClaudePluginLister.List` integration coverage
The worklog explicitly discloses (lines 118-122) that the production lister is
not exercised by an integration test today — invoking real
`claude plugin list --json` from CI is fragile because runners may lack the
binary and the output format is owned by Anthropic. Mock-driven tests cover the
matcher (`CheckRequiredPlugins`), parser (`parseClaudePluginList`), splitter
(`splitPluginEntry`), and wiring assertion
(`TestExecClaudePluginListerProductionWiring` at preflight_test.go L400).

The wiring assertion (L400-404) is the critical guard: it pins
`defaultClaudePluginLister` to the production type so a future refactor cannot
silently swap the singleton for a stub without breaking this test.

**Disposition:** ACCEPT. The deliverable rationale at worklog L118-122
correctly identifies this as a future opt-in skip-test once dogfooding begins
(Drop 5+). The mock-driven coverage is sufficient for the closed contract this
droplet ships.

### 2.2 Hook-gated wire-in observable end-to-end test
The wire-in at spawn.go L235-240 is unit-tested in two places:
1. Hook-receives-Project: `TestRequiredPluginsForProjectHookReceivesProject`
   (preflight_test.go L425-444).
2. Hook-default-nil: `TestRequiredPluginsForProjectHookDefaultIsNil`
   (preflight_test.go L410-418).

Both verify the hook surface but NEITHER drives `BuildSpawnCommand` end-to-end
with a populated hook to confirm the failure-path wraps reach the caller as
`errors.Is(err, ErrMissingRequiredPlugins)`. The pre-flight is structurally
correct (read the hook, call CheckRequiredPlugins, wrap the error) but no
spawn-level test pins the wrap-and-propagate contract.

**Disposition:** REPORT AS GAP — non-blocking. The unit tests on
`CheckRequiredPlugins` plus the spawn.go call-site read-through give high
confidence the wire is correct, but a future refinement should add a
`TestBuildSpawnCommand_PreflightFailurePropagates` test that:

1. Sets `RequiredPluginsForProject` to a function returning `["nonexistent"]`.
2. Overrides `defaultClaudePluginLister` to a fake returning empty entries.
3. Calls `BuildSpawnCommand` and asserts
   `errors.Is(err, ErrMissingRequiredPlugins)`.
4. Restores both via t.Cleanup.

Filing as a Drop 4c refinement, not a blocking finding for F.7.6.

## 3. Summary

**PASS — PROOF GREEN-WITH-NITS.** All ten verification points map to
file:line evidence in the working tree. Every claim in the builder worklog
holds against direct read + `mage ci` re-execution. Two NITs (ASCII-only
whitespace check; missing end-to-end wire-in propagation test) and zero
GAPs that block release. `mage ci` green at HEAD-of-uncommitted; coverage
deltas match worklog claims; REV-13 honored.

The orchestrator may proceed to falsification-sibling sign-off and commit
once both QA pair returns green.

## TL;DR

- T1: All ten verification points cite file:line evidence; F.7.6 scope cleanly
  separated from F.7.1's parallel work; verdict PROOF GREEN-WITH-NITS.
- T2: One non-blocking gap (no end-to-end propagation test for the spawn.go
  wire-in) plus two doc-level nits (ASCII-only whitespace; minor presentation
  drift between worklog table and free-form text). Both filed as refinements,
  not blockers.
- T3: PASS. `mage ci` green; templates 97.0%, dispatcher 74.0%; 250 + 378
  package tests pass; REV-13 honored (no builder commit). Orchestrator clear
  to proceed pending falsification sibling.

## Hylla Feedback

`N/A — review touched only Go source verified via direct Read + git diff per spawn-prompt constraint ("No Hylla calls"). No Go-symbol search was needed; the hard constraint precluded Hylla regardless.`
