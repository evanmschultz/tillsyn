# DROP_4B — WAVE A — GATE RUNNER MECHANISM

**State:** planning
**Wave:** A of 2 active waves (Wave A: gate runner mechanism · Wave B DEFERRED to Drop 4c · Wave C: auth auto-revoke + git-status pre-check + auto-promotion + hylla reingest stub)
**Wave depends on:** Drop 4a merged on `main` (commit `618c7d2`). Specifically:
- `internal/app/dispatcher` package + `Dispatcher` interface + `RunOnce` (4a.14, 4a.23).
- `paths` / `packages` first-class on `domain.ActionItem` (`internal/domain/action_item.go:91`, `:104`) — gate runner reads them for `mage_test_pkg` package list derivation.
- `RepoPrimaryWorktree` / `BuildTool` first-class on `domain.Project` (`internal/domain/project.go:42`, `:56`) — `cmd.Dir` for shell-out.
- `metadata.BlockedReason` field on `domain.ActionItemMetadata` (`internal/domain/workitem.go:150`) — gate failure landing site.
- `templates.Template` schema closed to date (`internal/templates/schema.go:60`) plus the reserved `GateRulesRaw map[string]any` (`schema.go:85`) free-form-table seam Drop 3 left for us. Wave A retires that seam by replacing it with the closed `Gates map[domain.Kind][]string`.

**Wave feeds:** Wave C (auto-promotion subscriber calls `RunOnce` → gate runner; `hylla_reingest` gate registers via 4b.7 atop the registry built in 4b.2).
**Brief ref:** `workflow/drop_4b/REVISION_BRIEF.md` § 4 Wave A (lines 38–44) + § 3 L1/L2/L6 (lines 26–31).
**Sketch ref:** `workflow/drop_4b/SKETCH.md` § Wave A (lines 50–55).
**Started:** 2026-05-04
**Closed:** —

## Wave Purpose

Land the deterministic post-build gate framework. Drop 4a's `RunOnce` ends at Stage 8 (monitor.Track) — the spawn is fired, the dispatcher returns `ResultSpawned`, and the action item runs to completion under the monitor's watch. Wave A adds the **gate runner**: a new component that fires when an action item transitions to its provisional terminal state (`metadata.outcome == "success"` from a builder agent), reads the project template's `[gates.<kind>]` sequence, executes each named gate in order via an internal registry, and either lets the action item progress to `complete` (all gates green) or transitions it to `failed` with `metadata.BlockedReason` describing the failed gate + last 100 lines (or 8KB, whichever is shorter) of captured output.

Wave A ships the framework + two of three closed-enum gate kinds: `mage_ci` and `mage_test_pkg`. The third (`hylla_reingest`) is implemented by Wave C droplet 4b.7 atop the same registry. Drop 4c registers `commit` + `push` atop the same registry without touching the framework.

NO LLM in any Wave A code path. Commit-agent invocation lands in Drop 4c F.7.

## Wave Architecture

Three new files in `internal/app/dispatcher/`, plus one schema extension in `internal/templates/schema.go` and a default-template update in `internal/templates/builtin/default.toml`.

| File                                              | Responsibility                                              | Lands in droplet |
| ------------------------------------------------- | ----------------------------------------------------------- | ---------------- |
| `internal/templates/schema.go` (EXTEND)           | Closed-enum `GateKind` primitive + `Template.Gates` table   | 4b.1             |
| `internal/templates/load.go` (EXTEND)             | `validateGateKinds` validator hooked into `Load`            | 4b.1             |
| `internal/templates/builtin/default.toml` (EXTEND)| Ship `[gates.build] = ["mage_ci"]` (per L6)                 | 4b.1             |
| `internal/app/dispatcher/gates.go` (NEW)          | `gateRunner` struct + registry + `Run` method               | 4b.2             |
| `internal/app/dispatcher/gates_test.go` (NEW)     | Registry behaviour, halt-on-first-failure, output capture   | 4b.2             |
| `internal/app/dispatcher/gate_mage_ci.go` (NEW)   | `mage_ci` gate `exec.Command` wrapper                       | 4b.3             |
| `internal/app/dispatcher/gate_mage_ci_test.go` (NEW) | Stub `commandRunner` injection; success + failure paths | 4b.3             |
| `internal/app/dispatcher/gate_mage_test_pkg.go` (NEW)| `mage_test_pkg` gate per-package iteration                | 4b.4             |
| `internal/app/dispatcher/gate_mage_test_pkg_test.go` (NEW) | Empty-`packages`, multi-`packages`, mid-failure cases | 4b.4             |

All four droplets edit different files, so no same-file `blocked_by` is required between them. **Package-level lock contention DOES bind 4b.2/4b.3/4b.4 together — they all touch `internal/app/dispatcher`.** Plan-QA falsification will check this; mitigation is explicit cross-droplet `blocked_by` entries.

`internal/templates/schema.go` and `internal/templates/load.go` are co-edited within droplet 4b.1 (one droplet, one author, no parallel build) so the same-file rule is not in play.

## Wave-Internal Sequencing

```
4b.1 (schema + closed enum + default template)
 └─→ 4b.2 (gate runner + registry; depends on GateKind enum)
      ├─→ 4b.3 (mage_ci gate; registers into runner)
      └─→ 4b.4 (mage_test_pkg gate; registers into runner)
```

4b.3 and 4b.4 can run in parallel after 4b.2 — they edit separate files and register distinct gate kinds. Both depend on 4b.2's `gateRunner.Register` API + `gateFunc` type signature being stable.

## Cross-Wave Blockers

- **Wave C 4b.5 (auth auto-revoke)** is independent of Wave A — different file (`cleanup.go`), different concern. No cross-wave edge.
- **Wave C 4b.6 (git-status pre-check)** is independent of Wave A — different file (`internal/app/service.go`), different concern. No cross-wave edge.
- **Wave C 4b.7 (auto-promotion + `hylla_reingest`)** depends on Wave A 4b.2's `gateRunner.Register` API to wire the `hylla_reingest` gate. Surface to Wave C planner: 4b.7 carries `blocked_by 4b.2`. Wave A planner does NOT model this here — it's a Wave C concern surfaced for cross-wave consistency.
- **Drop 4c F.7 (spawn pipeline)** consumes 4b.2's gate runner unchanged — no breaking-API constraints surface here.

## How The Gate Runner Plugs Into Dispatcher (Out-Of-Wave-A Context)

For builder + plan-QA reviewer reference. Wave A does NOT modify `RunOnce`. The gate runner is consumed by:

1. **Wave C 4b.7's auto-promotion subscriber.** When the subscriber observes `LiveWaitEventActionItemChanged` and the changed item carries `metadata.outcome == "success"` AND lifecycle is still `in_progress` AND the kind has a `[gates.<kind>]` entry, the subscriber calls `gateRunner.Run(ctx, item, project, template)`. On success, the subscriber transitions the item to `complete`. On failure, the subscriber transitions to `failed` with the gate-runner-supplied `BlockedReason`.
2. **Drop 4c F.7's redesigned spawn pipeline** — same pattern, post-spawn-completion hook.

Wave A's job: ship the gate runner + two of the three gate implementations + schema extension + default-template wiring. Wave A does NOT ship the subscriber wiring.

## Verification Targets

- Per droplet: `mage test-pkg internal/templates` (4b.1) or `mage test-pkg internal/app/dispatcher` (4b.2/3/4); `mage test-func <pkg> <TestName>` for specific scenarios named below.
- Wave-end: `mage ci` clean. Coverage on `internal/app/dispatcher` ≥ 70% (matches Drop 4a target).
- **Never** `mage install`. **Never** raw `go test`/`go build`/`go vet`/`go run`.

## Pre-MVP DB Action

4b.1 changes `templates.Template` (adds `Gates` field; `GateRulesRaw` reserved seam preserved untouched). **`KindCatalog.Gates` baking is NOT in 4b.1's scope** (per plan-QA-falsification F1 correction, 2026-05-04) — the gate runner reads `Template.Gates` directly via the `Run(ctx, item, project, tpl)` signature in 4b.2. If a future droplet decides the catalog should snapshot gates for performance reasons, it lands separately. **Dev fresh-DBs `~/.tillsyn/tillsyn.db` BEFORE `mage ci`** out of conservatism: existing project rows have `KindCatalogJSON` envelopes baked from a `Template` shape WITHOUT the `Gates` field; JSON unmarshal produces nil `Gates` map → harmless ("no gates per kind"), but fresh-DB matches "no migration logic" rule (memory `feedback_no_migration_logic_pre_mvp.md`).

4b.2/4b.3/4b.4 are pure-Go additions with no schema touch — no fresh-DB needed for those alone.

---

## Droplet Decomposition

### 4b.1 — `[gates]` TABLE SCHEMA + CLOSED-ENUM `GateKind` PRIMITIVE

- **State:** todo
- **Paths:**
  - `internal/templates/schema.go` (EXTEND)
  - `internal/templates/schema_test.go` (EXTEND)
  - `internal/templates/load.go` (EXTEND)
  - `internal/templates/load_test.go` (EXTEND)
  - `internal/templates/builtin/default.toml` (EXTEND)
  - `internal/templates/embed_test.go` (EXTEND — assert default decodes with new Gates table)
- **Packages:** `internal/templates`
- **Acceptance:**
  - New top-level `templates.GateKind string` type declared in `internal/templates/schema.go`. Closed-enum constants: `GateKindMageCI = "mage_ci"`, `GateKindMageTestPkg = "mage_test_pkg"`, `GateKindHyllaReingest = "hylla_reingest"`. Helper `IsValidGateKind(GateKind) bool`. Drop 4c will add `commit` + `push` to this enum; the comment block explicitly names that and references this droplet so the open-enum-pressure attack is documented and mitigated.
  - New field `Gates map[domain.Kind][]GateKind` on `templates.Template` with TOML tag `gates`. Per-kind entry decodes a TOML array of gate-kind strings (e.g. `[gates.build] = ["mage_ci"]` → `tpl.Gates[domain.KindBuild] == []GateKind{GateKindMageCI}`). The reserved-but-untyped `Template.GateRulesRaw map[string]any` (`schema.go:85`) is **retained** for forward-compat — a future closed `[gate_rules]` table for richer gate config (timeouts, retry policy) will land via that field. Wave A's `Gates` table is a SEPARATE TOML key (`gates`, not `gate_rules`), so no collision.
  - New validator `validateGateKinds(tpl Template) error` in `internal/templates/load.go` invoked from `Load` after `validateChildRuleReachability` (preserves existing pass order). Validator asserts: (a) every map KEY in `tpl.Gates` is a member of the closed `domain.Kind` enum (mirrors `validateMapKeys` shape, `load.go:161`); (b) every gate-kind string in every value slice is a member of the closed `GateKind` enum. Returns wrapped `ErrUnknownGateKind` (new sentinel).
  - New sentinel `var ErrUnknownGateKind = errors.New("template references an unknown gate kind")` declared alongside `ErrUnknownKindReference` (`load.go:143`).
  - `internal/templates/builtin/default.toml` adds `[gates]` section after `[[steward_seeds]]` with one entry: `[gates.build]` = `["mage_ci"]`. Per L6, no other kinds get gate sequences in 4b. Drop 4c will expand to `["mage_ci", "commit", "push"]`. Other kinds (`plan-qa-proof`, `build-qa-proof`, `closeout`, etc.) are absent — gate runner treats absence as "no gates" not "all gates" (clarified in 4b.2 doc-comment).
  - Strict-decode round-trip in `TestTemplateTOMLRoundTrip` (`schema_test.go:49`) extends to populate `Gates` and verify it round-trips. Add new test `TestTemplateGatesValidation`: load a TOML with `[gates.build] = ["totally-bogus-gate"]` and assert `errors.Is(err, ErrUnknownGateKind)`. Add `TestTemplateGatesUnknownKindKey`: `[gates.bogus-kind]` rejects.
  - `internal/templates/embed_test.go` extends `TestDefaultTemplateLoads` to assert `tpl.Gates[domain.KindBuild]` equals `[]GateKind{GateKindMageCI}`.
  - **YAGNI watch:** do NOT add per-gate config (timeouts, retry policy, env vars) — closed-enum + array of names is the entire surface. The `GateRulesRaw` reserved seam catches any future need.
- **Test scenarios:**
  - `TestGateKindEnumMembership` — every constant `IsValidGateKind` returns true; arbitrary string returns false.
  - `TestTemplateTOMLRoundTrip` (extended) — populated `Gates` field round-trips.
  - `TestTemplateGatesValidation` — unknown gate-kind string rejects with `ErrUnknownGateKind`.
  - `TestTemplateGatesUnknownKindKey` — bogus map key rejects via existing `validateMapKeys` (extended to cover `Gates`).
  - `TestDefaultTemplateLoadsWithGates` (in `embed_test.go`) — `default.toml` decodes and `tpl.Gates[KindBuild]` is `[]GateKind{GateKindMageCI}`.
  - `TestTemplateGatesEmptyMapDecodes` — TOML without any `[gates.*]` table decodes to `nil` or empty map without error (gate runner handles absence).
- **Falsification attacks anticipated + mitigations:**
  - **A1 — Open-enum pressure:** "Why not accept any string and warn?" Mitigation: locked decision L1 + L6 (REVISION_BRIEF lines 26, 31) — closed enum, Drop 4c expands. Wave A doc-comment cites the locked decisions.
  - **A2 — `GateRulesRaw` collision:** "The reserved `[gate_rules]` table from Drop 3 conflicts with this `[gates]` table." Mitigation: distinct TOML keys (`gates` vs `gate_rules`); strict-decode treats them as separate fields; `GateRulesRaw` stays as the future-config seam. Test `TestTemplateGatesAndGateRulesCoexist` populates both and round-trips.
  - **A3 — Map-key validation gap:** "`validateMapKeys` (load.go:161) only checks `Kinds` and `AgentBindings`, not `Gates`." Mitigation: extend `validateMapKeys` body to iterate `tpl.Gates` keys too; test `TestTemplateGatesUnknownKindKey` exercises.
  - **A4 — Default template missing gates breaks Drop 4a tests:** Drop 4a's `mage test-pkg internal/templates` may have golden assertions on the default template's exact field set. Mitigation: builder runs `mage test-pkg internal/templates` first and updates affected goldens (if any) inline.
  - **A5 — Gate kind value normalization:** TOML-decoded strings might have whitespace. Mitigation: `IsValidGateKind` does exact match on closed constants; planner-side trimming is the template author's job (matches existing `domain.Kind` validation pattern). No silent normalization.
- **DB action:** **Dev fresh-DBs `~/.tillsyn/tillsyn.db` BEFORE `mage ci`** — `KindCatalogJSON` envelope baked into existing project rows does not yet embed `Gates`. Pre-MVP rule (no Go migration logic).
- **Blocked by:** none (Wave A entry droplet). Cross-wave: none — Drop 4a's templates package is stable.
- **Verification:**
  - `mage test-pkg internal/templates` clean.
  - `mage test-func internal/templates TestGateKindEnumMembership`.
  - `mage test-func internal/templates TestTemplateGatesValidation`.
  - `mage test-func internal/templates TestDefaultTemplateLoadsWithGates`.
  - `mage ci` clean (after dev fresh-DBs).
- **LOC delta estimate:** ~120 LOC schema/load extensions, ~80 LOC tests, ~3 lines TOML default. Total ~200 LOC.

---

### 4b.2 — GATE RUNNER + REGISTRY (`internal/app/dispatcher/gates.go`)

- **State:** todo
- **Paths:**
  - `internal/app/dispatcher/gates.go` (NEW)
  - `internal/app/dispatcher/gates_test.go` (NEW)
- **Packages:** `internal/app/dispatcher`
**RETRO-EDITED 2026-05-04 to match shipped API shape (per QA-Proof + QA-Falsification NIT). Original spec had `GateInput`/`GateRunOutcome` struct-bag shape; shipped impl uses simpler 3-arg `gateFunc` + `[]GateResult` per-row return. Both cover the same surface; shipped is per-gate-row visibility for downstream consumers. Key changes preserved with strikethrough below.**

- **Acceptance (shipped):**
  - New `type gateFunc func(ctx context.Context, item domain.ActionItem, project domain.Project) GateResult` in `gates.go`. (Original: `func(ctx, GateInput) GateResult` with `Template` in input — slimmed since gates don't need template back-reference.)
  - New `type gateRunner struct` with private fields: `gates map[templates.GateKind]gateFunc`, `mu sync.RWMutex` (Register write-lock + Run read-lock; defense-in-depth + closes data race documented in 4b.2 round-2 fix-builder).
  - `func newGateRunner() *gateRunner` constructor. Returns runner with empty registry. Registration is explicit per-droplet (`Register(GateKindMageCI, mageCIGate)` in 4b.3, etc.).
  - `func (r *gateRunner) Register(name string, fn gateFunc) error` — accepts a string name (not the closed-enum type) since template-load via 4b.1's `validateGateKinds` is the validation layer; rejects duplicates with `ErrGateAlreadyRegistered`. Validation moved upstream from Register-time to template-load-time per separation-of-concerns.
  - `func (r *gateRunner) Run(ctx context.Context, item domain.ActionItem, project domain.Project, tpl *templates.Template) []GateResult` — main entry. Reads `tpl.Gates[item.Kind]`; iterates in declared order; halts on first failure; returns slice of `GateResult` (one per gate executed including failed one). Empty slice on `tpl == nil` or empty kind sequence. Inter-gate `ctx.Err()` check halts cleanly with `Skipped`-flavored result (4b.2 round-2 addition).
  - `GateResult` struct: `GateName GateKind`, `Status GateStatus` (closed enum: `Passed`/`Failed`/`Skipped`), `Output string` (failure output capture; empty on Passed), `Duration time.Duration`, `Err error`. **Per-gate row** (vs original `GateRunOutcome` single-row) gives downstream consumers visibility into the full gate sequence even when one fails.
  - **Output capture rule (resolves Q7):** last 100 LINES of combined stdout+stderr OR last 8KB whichever is **shorter**, formatted as a single string. Helper `tailOutput(combined []byte) string` lives in `gates.go`. The choice is explicit + tested.
  - **Halt-on-first-failure (resolves Q2):** runner does NOT deduplicate gate kinds — if a template lists `[gates.build] = ["mage_ci", "mage_ci"]` it runs twice. Documented as "no implicit deduplication; template authors are responsible." Test `TestGateRunnerNoDeduplication` confirms.
  - **Empty-gates handling:** when `tpl.Gates[item.Kind]` is nil OR empty slice, `Run` returns `GateRunOutcome{Success: true}` immediately. Documented: "absence of gates means no gates, not 'all gates'." Test `TestGateRunnerEmptyGates`.
  - **Unregistered gate handling:** when a template references a gate kind that the runner's registry has not been populated with (e.g. `hylla_reingest` not yet registered in 4b.7 wiring), `Run` returns `GateRunOutcome{Success: false, FailedGate: kind, Err: ErrGateNotRegistered}`. The wiring layer (Wave C 4b.7) is responsible for completing registration. Test `TestGateRunnerUnregisteredGate`.
  - **Context cancellation:** `Run` propagates `ctx.Done()` to each gate via `GateInput.Ctx`. A canceled run mid-sequence returns `GateRunOutcome{Success: false, Err: ctx.Err()}` immediately after the in-flight gate returns. Test `TestGateRunnerContextCancel`.
  - New sentinel errors: `ErrGateAlreadyRegistered`, `ErrGateNotRegistered`. Located in `gates.go` for cohesion (not lumped into `dispatcher.go`).
  - Doc-comment spans the failure-routing contract: `GateRunOutcome.Success == false` is the signal that the **subscriber** (Wave C 4b.7) MUST transition the action item to `failed` and write `metadata.BlockedReason` formatted as `"gate <FailedGate> failed: <Output>"`. The runner ITSELF does NOT mutate the action item — separation of concerns; the runner is a pure executor.
- **Test scenarios (all in `gates_test.go`):**
  - `TestGateRunnerRegister` — duplicate kind rejects with `ErrGateAlreadyRegistered`; unknown kind rejects via `IsValidGateKind`.
  - `TestGateRunnerRunSuccess` — register a fake gate that returns `Success: true`; assert `GateRunOutcome.Success == true`.
  - `TestGateRunnerRunFailureHalts` — register two fake gates, first fails; assert second never invoked + `FailedGate` names the first.
  - `TestGateRunnerRunOrderDeterministic` — three fake gates in `[gates.build]`; assert invocation order matches TOML order.
  - `TestGateRunnerEmptyGates` — `tpl.Gates[KindBuild]` nil; `Run` returns `Success: true` without invoking anything.
  - `TestGateRunnerUnregisteredGate` — template lists `mage_ci` but runner registry empty; `Run` returns `ErrGateNotRegistered` with `FailedGate == "mage_ci"`.
  - `TestGateRunnerContextCancel` — register a gate that blocks on `ctx.Done()`; cancel mid-run; assert `Run` returns within 100 ms with `ctx.Err()` wrapped.
  - `TestGateRunnerOutputCapture` — gate returns 200 lines of output; assert `GateRunOutcome.Output` is exactly the last 100 lines (or 8KB, whichever shorter; both bounds tested).
  - `TestGateRunnerNoDeduplication` — `[gates.build] = ["mage_ci", "mage_ci"]`; gate registered with hit counter; assert it ran twice.
- **Falsification attacks anticipated + mitigations:**
  - **A1 — Runner mutates action item:** "Why doesn't `Run` write `BlockedReason` directly?" Mitigation: separation of concerns — runner is a pure executor; the subscriber owns transitions. Doc-comment explicit; test asserts `Run` never calls back into a service interface.
  - **A2 — Hidden ordering bug:** "TOML decode order is map iteration, not slice declaration order." Mitigation: `Gates` field is `map[domain.Kind][]GateKind` — the SLICE's order is preserved by go-toml/v2 (verified in 4b.1's round-trip test). Per-kind iteration is over the slice, deterministic.
  - **A3 — Cancellation racy:** "If a gate ignores ctx, the runner hangs." Mitigation: documented; test `TestGateRunnerContextCancel` asserts behaviour for a well-behaved gate. Real gates (`mage_ci`, `mage_test_pkg`) MUST honor ctx via `exec.CommandContext`. Plan-QA falsification will check 4b.3/4b.4 use `CommandContext`.
  - **A4 — `GateInput` over-broad:** "Why pass full `templates.Template`?" Mitigation: gates may need to read peer kind config (e.g. `mage_test_pkg` reads `item.Packages` only — no template needed). Builder may slim to just `Item` + `Project` if `Template` proves unused; surface as a planner-time YAGNI watch.
  - **A5 — Output capture loses important early errors:** "Last 100 lines hides the real cause when the build emits 1000-line stack traces." Mitigation: documented limitation; future improvement (full-output side-table) noted in doc-comment + Drop 4c refinement raised. Acceptable for MVP.
  - **A6 — `GateRunOutcome.Err` vs `Success: false` ambiguity:** Mitigation: doc-comment defines: `Err != nil` is **infrastructure failure** (gate not registered, OS error wrapping `exec.Cmd.Start`); `Success: false && Err == nil` is **gate-determined failure** (gate ran, returned non-zero exit). Subscriber routes both to `failed` lifecycle with distinct `BlockedReason` shapes.
  - **A7 — Concurrent `Run` for different action items:** Mitigation: registry guarded by `sync.RWMutex`; `Run` takes RLock for registry lookup, releases before invoking gateFunc. No shared mutable state across `Run` invocations. Test `TestGateRunnerConcurrentRuns` (parallel `t.Run` with different fake gates).
- **DB action:** none (pure Go, no schema).
- **Blocked by:** 4b.1 (`templates.GateKind` enum + `Template.Gates` field must compile before this droplet imports them). Cross-wave: none.
- **Verification:**
  - `mage test-pkg internal/app/dispatcher` clean.
  - `mage test-func internal/app/dispatcher TestGateRunnerRunFailureHalts`.
  - `mage test-func internal/app/dispatcher TestGateRunnerEmptyGates`.
  - `mage test-func internal/app/dispatcher TestGateRunnerOutputCapture`.
  - `mage test-func internal/app/dispatcher TestGateRunnerContextCancel`.
  - `mage ci` clean.
- **LOC delta estimate:** ~180 LOC `gates.go`, ~280 LOC tests. Total ~460 LOC.

---

### 4b.3 — `mage_ci` GATE IMPLEMENTATION (`internal/app/dispatcher/gate_mage_ci.go`)

- **State:** todo
- **Paths:**
  - `internal/app/dispatcher/gate_mage_ci.go` (NEW)
  - `internal/app/dispatcher/gate_mage_ci_test.go` (NEW)
- **Packages:** `internal/app/dispatcher`
- **Acceptance:**
  - New `func mageCIGate(ctx context.Context, input GateInput) GateResult` in `gate_mage_ci.go`. Signature matches `gateFunc` from 4b.2.
  - Implementation: `cmd := exec.CommandContext(ctx, "mage", "ci")`. `cmd.Dir = input.Project.RepoPrimaryWorktree`. Combined stdout+stderr captured into `bytes.Buffer` (single buffer for output ordering — `cmd.Stdout = buf; cmd.Stderr = buf`). `cmd.Env` inherits `os.Environ()` (no scrubbing — `mage` needs `PATH` + `GOPATH` + `HOME`). Run via `cmd.Run()`.
  - Exit 0 → `GateResult{Success: true, Output: ""}` (empty output on success per "successful gates record nothing" rule).
  - Non-zero exit → `GateResult{Success: false, Output: tailOutput(buf.Bytes()), Err: nil}`. The `Err` is nil because non-zero exit is a **gate-determined failure** (build broke), not infrastructure failure. Per 4b.2 doc-comment.
  - `cmd.Start()` failure (e.g. `mage` binary missing on PATH) → `GateResult{Success: false, Output: "", Err: fmt.Errorf("mage_ci: start: %w", err)}`. Infrastructure failure path — 4b.2 subscriber routes to `failed` with distinct `BlockedReason` shape.
  - **Empty `RepoPrimaryWorktree` guard:** if `input.Project.RepoPrimaryWorktree == ""` return `GateResult{Success: false, Err: fmt.Errorf("mage_ci: project has empty repo_primary_worktree")}` immediately — do NOT shell out from arbitrary cwd. Mirrors `RunOnce` Stage 1's empty-worktree skip (`dispatcher.go:392`).
  - **Test seam:** define `var commandRunner func(ctx context.Context, dir, name string, args ...string) (output string, exitCode int, err error)` package-private indirection. Production wiring binds to `defaultCommandRunner` which does the `exec.CommandContext` shell-out; tests inject a fake. Avoids hitting the actual `mage` binary in CI.
  - **Concurrent-safe (resolves L3 falsification surface):** the gate function does not share mutable state across invocations. `commandRunner` is package-level but called via local function-pointer copy → safe for concurrent `Run` calls from 4b.2.
  - Doc-comment cites L1 (deterministic, no LLM), L2 (closed-enum gate kind), and the test-seam pattern.
- **Test scenarios (`gate_mage_ci_test.go`):**
  - `TestMageCIGateSuccess` — fake `commandRunner` returns exit 0 + 50 lines of output; assert `Success: true`, `Output: ""` (empty on success).
  - `TestMageCIGateFailure` — fake returns exit 1 + 200 lines of output; assert `Success: false`, `Output` is last 100 lines, `Err: nil`.
  - `TestMageCIGateStartError` — fake returns synthetic `os/exec` "executable not found" error; assert `Success: false`, `Err` wraps the underlying error and includes `"mage_ci: start"`.
  - `TestMageCIGateEmptyWorktree` — `input.Project.RepoPrimaryWorktree = ""`; assert immediate `Success: false`, `Err` mentions `repo_primary_worktree`. No `commandRunner` invocation.
  - `TestMageCIGateContextCancel` — fake `commandRunner` honors `ctx.Done()` and returns `ctx.Err()`; gate caller asserts `ctx.Err()` propagates.
  - `TestMageCIGateOutputCaptureBound` — fake returns 12KB of output; assert `Output` is last 8KB (the 8KB cap, since 12KB / ~80 chars-per-line ≈ 150 lines > 100, but byte-cap is shorter at 8KB).
- **Falsification attacks anticipated + mitigations:**
  - **A1 — Hardcoded `mage` binary name on PATH:** "What if dev's PATH doesn't have `mage`?" Mitigation: `cmd.Start()` returns "executable not found"; gate returns `Err`-flavored failure; subscriber surfaces a clear `BlockedReason`. Tested in `TestMageCIGateStartError`.
  - **A2 — Output ordering interleave:** "Combined stdout+stderr buffer interleaves writes from concurrent goroutines inside `mage`." Mitigation: `exec.Cmd` provides one-writer-per-stream serialization at the syscall level; sharing the same `bytes.Buffer` via `cmd.Stdout = buf; cmd.Stderr = buf` is the **idiomatic combined-output pattern** (matches stdlib `exec.Cmd.CombinedOutput` source). Documented; no test (it's stdlib behaviour).
  - **A3 — Long-running CI hangs the runner:** "What if `mage ci` runs forever?" Mitigation: `exec.CommandContext` propagates `ctx.Done()` — the runner's caller (Wave C 4b.7 subscriber) supplies the timeout-bearing ctx. Wave A does not enforce a default timeout; documented as subscriber's responsibility.
  - **A4 — `os.Environ()` leaks dev creds into mage subprocess:** Mitigation: this is the existing builder-spawn behaviour from Drop 4a (`spawn.go`); same principle. The gate runs the project's own `mage ci` — it's expected to inherit the parent env. No new attack surface.
  - **A5 — Working directory escape:** "What if `RepoPrimaryWorktree` is set to `/`?" Mitigation: `domain.Project.Validate` (project.go) already enforces absolute-path constraint at project-creation time; the gate trusts the validated value. No re-validation here (single source of truth).
  - **A6 — Output capture truncation hides exit-code message:** "If `mage ci` prints exit-code summary on the FIRST line and 100 lines of stack trace after, last 100 lines drops the summary." Mitigation: same as 4b.2 A5 — documented limitation, MVP-acceptable, future improvement noted.
- **DB action:** none.
- **Blocked by:** 4b.2 (needs `gateFunc` type, `GateInput`, `GateResult`, `tailOutput` helper). Cross-wave: none.
- **Verification:**
  - `mage test-pkg internal/app/dispatcher` clean.
  - `mage test-func internal/app/dispatcher TestMageCIGateSuccess`.
  - `mage test-func internal/app/dispatcher TestMageCIGateFailure`.
  - `mage test-func internal/app/dispatcher TestMageCIGateEmptyWorktree`.
  - `mage ci` clean.
- **LOC delta estimate:** ~80 LOC `gate_mage_ci.go`, ~150 LOC tests. Total ~230 LOC.

---

### 4b.4 — `mage_test_pkg` GATE IMPLEMENTATION (`internal/app/dispatcher/gate_mage_test_pkg.go`)

- **State:** todo
- **Paths:**
  - `internal/app/dispatcher/gate_mage_test_pkg.go` (NEW)
  - `internal/app/dispatcher/gate_mage_test_pkg_test.go` (NEW)
- **Packages:** `internal/app/dispatcher`
- **Acceptance:**
  - New `func mageTestPkgGate(ctx context.Context, input GateInput) GateResult` in `gate_mage_test_pkg.go`. Signature matches `gateFunc` from 4b.2.
  - Reads `input.Item.Packages []string` (`internal/domain/action_item.go:104`).
  - **Empty packages handling:** if `len(input.Item.Packages) == 0`, return `GateResult{Success: true, Output: ""}` immediately. Doc-comment: "no packages declared = nothing to test = success." This matches L4's optimization rationale (`mage_test_pkg` is for sub-package work; absence of packages means the action item isn't sub-package-scoped).
  - **Per-package iteration:** for each `pkg` in `input.Item.Packages`:
    - `cmd := exec.CommandContext(ctx, "mage", "test-pkg", pkg)`.
    - `cmd.Dir = input.Project.RepoPrimaryWorktree`.
    - Combined stdout+stderr capture into a per-package `bytes.Buffer`.
    - On non-zero exit: halt iteration, return `GateResult{Success: false, Output: tailOutput(buf.Bytes()), Err: nil}`. Output prefixed with `"package " + pkg + " failed:\n"` so the subscriber's `BlockedReason` names the failing package.
    - On `cmd.Start()` failure: halt, return `GateResult{Success: false, Err: fmt.Errorf("mage_test_pkg %q: start: %w", pkg, err)}`.
    - On context cancel mid-iteration: halt, return `GateResult{Success: false, Err: ctx.Err()}`.
  - All packages green → `GateResult{Success: true, Output: ""}`.
  - **Empty `RepoPrimaryWorktree` guard:** identical to 4b.3 — return `Err`-flavored failure immediately without shelling out.
  - **No deduplication of `Packages`:** if `Packages = ["pkgA", "pkgA"]`, runs twice. Matches L1's "template author's responsibility" rule.
  - Test seam: reuses `commandRunner` indirection from 4b.3 (declared in `gate_mage_ci.go`; cross-file usage within same package is fine).
  - Doc-comment cites L4 (sub-package optimization), the kind-bound use case (e.g. `[gates.build-qa-proof] = ["mage_test_pkg"]` runs only the package the QA pass owns), and the explicit non-deduplication rule.
- **Test scenarios (`gate_mage_test_pkg_test.go`):**
  - `TestMageTestPkgGateEmptyPackages` — `Item.Packages == nil`; assert immediate `Success: true` without invoking `commandRunner`.
  - `TestMageTestPkgGateSinglePackageSuccess` — `Packages = ["internal/app"]`; fake returns exit 0; assert `Success: true`, single `commandRunner` invocation with `mage test-pkg internal/app`.
  - `TestMageTestPkgGateMultiPackageAllSuccess` — `Packages = ["internal/app", "internal/domain"]`; both exit 0; assert two invocations in declaration order.
  - `TestMageTestPkgGateFirstFails` — first package exits 1; assert second never invoked, `Output` contains `"package internal/app failed:"` prefix + tail of fake output.
  - `TestMageTestPkgGateMidIterationCancel` — cancel ctx between first and second package; assert second never invoked, `Err == ctx.Err()`.
  - `TestMageTestPkgGateEmptyWorktree` — `RepoPrimaryWorktree == ""`; immediate `Err`-flavored failure; no `commandRunner` invocation.
  - `TestMageTestPkgGateStartError` — fake returns "executable not found"; `Err` wraps with package name.
  - `TestMageTestPkgGateOutputCaptureBound` — single package emits 12KB; `Output` last-8KB-or-100-lines bound applies (mirrors 4b.3).
- **Falsification attacks anticipated + mitigations:**
  - **A1 — `mage test-pkg <pkg>` arg shape unverified:** Mitigation: planner verified via `magefile.go:49` (`func TestPkg(pkg string) error`) — the mage target accepts a single positional arg. Documented in droplet acceptance.
  - **A2 — Per-package serial slowness:** "Why not run all packages in one `mage test` invocation?" Mitigation: per-package gives precise blame on first failure (subscriber's `BlockedReason` names the package); single-invocation would lose that. Tradeoff documented; refinement raised for "concurrent `mage test-pkg` if N > some threshold" as a future optimization.
  - **A3 — Package-name validation:** "What if `Packages` contains shell-injection chars?" Mitigation: `exec.CommandContext` uses execve directly (no shell), so arg vector is safe by construction. `domain.ActionItem.Packages` is normalized at create-time (`internal/domain/action_item.go:235` + neighbours). No re-validation needed.
  - **A4 — Empty-string package in slice:** `Packages = ["", "internal/app"]`. Mitigation: domain normalization drops empty strings; if it slips through, `mage test-pkg ""` would error and the gate reports failure on the empty-string package. Test `TestMageTestPkgGateEmptyStringPackage` (deferred to builder if domain-side guarantee holds — surface to plan-QA).
  - **A5 — Overlap with `mage_ci`:** "If a template runs both, the test suite runs twice (once via `mage ci`, once via `mage test-pkg`)." Mitigation: documented in 4b.2 (resolves Q2) — no implicit deduplication; template author's responsibility. Doc-comment cites this.
  - **A6 — `mage_test_pkg` for `closeout` kind makes no sense:** Mitigation: gate doesn't enforce kind-fit — the template author is responsible for sensible bindings. The closed-enum membership check in 4b.1 + the validator in 4b.1 catch invalid gate names; sensibility is downstream.
- **DB action:** none.
- **Blocked by:** 4b.2 (needs `gateFunc` type + helpers), 4b.3 (needs `commandRunner` package-private indirection declared there). The 4b.3 dependency is **same-package, same-symbol-source** — without 4b.3 declaring `commandRunner`, 4b.4 would need to declare its own duplicate. Concrete edge: 4b.4 `blocked_by 4b.3`. Surface to plan-QA: this means 4b.3 and 4b.4 are NOT parallelizable despite editing different files; adjust wave-internal sequencing accordingly. **REVISED SEQUENCING (supersedes top-of-doc graph):** 4b.1 → 4b.2 → 4b.3 → 4b.4 (linear). The "parallel after 4b.2" claim earlier in this document is wrong; corrected here.
- **Verification:**
  - `mage test-pkg internal/app/dispatcher` clean.
  - `mage test-func internal/app/dispatcher TestMageTestPkgGateEmptyPackages`.
  - `mage test-func internal/app/dispatcher TestMageTestPkgGateMultiPackageAllSuccess`.
  - `mage test-func internal/app/dispatcher TestMageTestPkgGateFirstFails`.
  - `mage ci` clean.
- **LOC delta estimate:** ~110 LOC `gate_mage_test_pkg.go`, ~200 LOC tests. Total ~310 LOC.

---

## Wave-A LOC Delta Summary

- 4b.1: ~200 LOC.
- 4b.2: ~460 LOC.
- 4b.3: ~230 LOC.
- 4b.4: ~310 LOC.
- **Wave A total: ~1200 LOC.**

## Wave-A Cross-Droplet Sequencing (CORRECTED)

```
4b.1 (schema + closed enum + default template)
 └─→ 4b.2 (gate runner + registry)
      └─→ 4b.3 (mage_ci gate + commandRunner indirection)
           └─→ 4b.4 (mage_test_pkg gate; reuses commandRunner)
```

**Linear chain.** No droplet pair is parallelizable: each consumes a symbol/file declared by the prior. This is conservative — Wave A is small (~1200 LOC), serial is fine, parallelism savings would be marginal vs. the cross-droplet review overhead.

## Open Questions Surfaced For Plan-QA

- **PQA-1 — `GateInput` shape:** does the gate need `templates.Template` or just `Item` + `Project`? Builder may slim if `Template` proves unused (4b.3, 4b.4 don't read it). Surface as a YAGNI watch for plan-QA-falsification.
- **PQA-2 — `commandRunner` shared across 4b.3 and 4b.4:** declaring it in `gate_mage_ci.go` and reusing in `gate_mage_test_pkg.go` is unconventional (typically helpers live in a neutral file). Builder may relocate to `internal/app/dispatcher/exec_runner.go` — surface as a refactor option for plan-QA-proof.
- **PQA-3 — Output capture: 100 lines OR 8KB whichever shorter:** is "shorter" the right semantic? Alternative: "longer of 100 lines or 8KB" (more diagnostic info). REVISION_BRIEF Q7 says "shorter" — locked. Plan-QA-falsification should re-attack if there's a counterexample.
- **PQA-4 — Empty-`Packages` semantic for `mage_test_pkg`:** "no packages declared = success" matches L4's optimization rationale, but a planner could plausibly intend `mage_test_pkg` as a regression-net even on action items without explicit packages (would default to all packages). Wave A treats absence as success; if dev wants the alternate semantic, raise as a Drop 4c refinement.

## Hylla Feedback

N/A — planning touched non-Go files only (markdown docs) plus Go-source READS (`schema.go`, `dispatcher.go`, `cleanup.go`, `workitem.go`, `project.go`, `default.toml`, `magefile.go`). Per spawn directive Wave A planner does NOT call Hylla (Hylla stale across post-merge code AND gate framework is new code with no committed surface). Used `Read` / `Bash rg` directly per CLAUDE.md non-Go fallback rule.
