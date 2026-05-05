# 4c F.7.6 — Builder QA Falsification

**Droplet:** F.7.6 — Required system-plugin pre-flight check.
**Reviewer scope:** read-only adversarial pass over builder's F.7.6 output.
**Sibling:** F.7.6 fired in parallel with F.7.1 (separate worklog, separate code surface).
**REVISIONS source of truth:** `workflow/drop_4c/F7_CORE_PLAN.md` REVISIONS POST-AUTHORING (REV-7, REV-13).

---

## Per-Attack Verdicts

### A1 — Hook-nil silent skip footgun

**Premises** Default `RequiredPluginsForProject == nil` makes pre-flight a no-op for every spawn until adopters opt in at boot. The footgun claim: a misconfigured deployment that *should* enforce required plugins silently bypasses the check.

**Evidence**
- `internal/app/dispatcher/plugin_preflight.go:16-37` doc-comment explicitly frames the seam: *"A nil hook (the default) means 'no required plugins' and CheckRequiredPlugins short-circuits before invoking the lister; callers wire the hook at process boot once the data feed is plumbed."*
- `internal/app/dispatcher/spawn.go:235-240` reads the hook via local-copy + nil-check.
- Worklog § "Wire-In Decision" (line 102-116) explicitly documents `KindCatalog` does not yet carry `[tillsyn]` globals, so the hook is a *deliberate seam* awaiting future plumbing.
- F7_CORE_PLAN.md acceptance criteria (line 470): *"`till bootstrap` invokes `CheckRequiredPlugins` once on each `tillsyn bootstrap`. Pre-dispatch hook also invokes it (Attack #7 mitigation)."* The acceptance assumed `till bootstrap` exists; it does not. The wire-in compromise (per-dispatch-only via opt-in hook) is documented but moves the failure surface from "hard-fail at boot" to "silent-skip until opt-in."

**Trace or cases**
- (A) Adopter sets template TOML `requires_plugins = ["context7@official"]` but forgets `dispatcher.RequiredPluginsForProject = …` at boot. Result: every spawn goes through with no pre-flight; if `claude` lacks `context7@official`, the spawn invokes a Claude run that fails downstream (MCP cannot connect to context7). The TOML field is decoration; the runtime contract is silently absent.
- (B) Adopter migrates from a code path that wired the hook to a refactor that drops it. No compile error, no test failure — TOML still validates, dispatcher still runs.

**Conclusion** REFUTED at code-correctness level (the seam is documented, no spec mandates non-nil-by-default). CONFIRMED as a documented design tradeoff with real ergonomics risk. The seam is consistent with builder's "deferred to a future droplet that plumbs `KindCatalog.RequiresPlugins`" plan, but the gap is wider than the worklog's "follow-up droplet" framing implies — there is no compile-time, test-time, or template-load-time signal that `requires_plugins = [...]` requires hook-wiring to enforce.

**Verdict** **NIT (documented hazard).** Not a counterexample against the F.7.6 acceptance criteria as builder's plan revision (no `till bootstrap` available; hook-pivot documented). But the asymmetry between "TOML field declared = enforcement assumed by reader" and "runtime check skipped without hook" merits a follow-up:
- Either an adopter-facing `dispatcher.RuntimeOptions` (or similar boot-validation hook) that asserts "if any project has non-empty `requires_plugins`, `RequiredPluginsForProject` must be non-nil" at first dispatch.
- Or a startup log line at first dispatch citing whether the pre-flight hook is wired.

This is route-able to a refinement.

---

### A2 — `RequiresPlugins` on Tillsyn vs other locations

**Premises** F.7.6's `RequiresPlugins` lives on the `[tillsyn]` global table — applies project-globally. Per-binding scoping (different builder kinds need different plugins) is not supported.

**Evidence**
- `internal/templates/schema.go:260-292` — `RequiresPlugins []string` on `Tillsyn`, doc-comment frames the table as *"top-level dispatcher / aggregator globals"* and *"the single carrier for dispatcher-global knobs"*.
- F7_CORE_PLAN.md REV-7 line 1039-1048 codifies the policy: `Tillsyn` is the carrier for global knobs. `AgentBinding`-level plugin scoping is not in scope.
- Plan acceptance line 463: `[ ] Tillsyn.RequiresPlugins []string` — the plan placed it on `Tillsyn` deliberately.

**Trace or cases**
- (A) Template TOML wants `[agent_bindings.build]` to require `gopls-lsp` but `[agent_bindings.commit]` not to → today, force `gopls-lsp` on the global table → the commit-agent spawn ALSO fails pre-flight when `gopls-lsp` is missing. Possible footgun under restrictive dev environments.

**Conclusion** REFUTED. The plan author chose project-global scope deliberately (REV-7 + line 463). Per-binding scope is not in F.7.6's contract. Future drop can add `AgentBinding.RequiresPlugins` if a real adopter need surfaces; today there is no in-tree consumer for per-binding scoping.

**Verdict** **REFUTED (documented limitation, in-spec).**

---

### A3 — `@` form parsing edge cases

**Premises** Validator must reject malformed shapes: bare `@` form (no name), bare `name@` (no marketplace), multi-`@`.

**Evidence**
- `internal/templates/load.go:850-883` — `validateTillsynRequiresPlugins`:
  - empty entry rejection (line 853-856).
  - whitespace rejection (line 857-860, includes space + tab + CR + LF).
  - `>1 @` rejection (line 861-864).
  - empty-name-before-`@` (line 867-869).
  - empty-marketplace-after-`@` (line 871-873).
  - within-list duplicate (line 876-879).
- `internal/templates/load_test.go:1399-1462` — `TestLoadTillsynRequiresPluginsRejectionTable` — 8 rows cover all 6 reject categories.

**Trace or cases**
- `["@marketplace"]` → "empty name before '@'" — covered.
- `["foo@bar@baz"]` → "contains more than one '@'" — covered.
- `["context7@"]` → "empty marketplace after '@'" — covered.
- `["@"]` (bare `@`) → "empty name before '@'" — falls through to first guard. (Counts of `@` is 1, so multi-`@` rule does not fire. Empty-name-before-`@` correctly fires first.)

**Conclusion** REFUTED.

**Verdict** **REFUTED.**

---

### A4 — Production lister: `claude` binary not on PATH

**Premises** When `claude` is missing from PATH, `execClaudePluginLister.List` must return `ErrClaudeBinaryMissing` (distinguishable from plugin-missing) per F.7.6 acceptance line 469.

**Evidence**
- `internal/app/dispatcher/plugin_preflight.go:274-280`:
  ```go
  if err := cmd.Start(); err != nil {
      if errors.Is(err, exec.ErrNotFound) {
          return nil, fmt.Errorf("%w: install claude per https://docs.claude.com/en/docs/claude-code (underlying: %v)",
              ErrClaudeBinaryMissing, err)
      }
      return nil, fmt.Errorf("dispatcher: start claude plugin list: %w", err)
  }
  ```
- The `errors.Is(err, exec.ErrNotFound)` branch wraps `ErrClaudeBinaryMissing` correctly.
- However, NO TEST exercises this branch. The production `execClaudePluginLister.List` has no unit test that triggers `exec.ErrNotFound` (would require mocking `exec.LookPath` or using a fake `$PATH`).

**Trace or cases**
- (A) Caller has `claude` not on PATH → `cmd.Start()` returns wrapped `exec.ErrNotFound` → branch fires → `ErrClaudeBinaryMissing` returned. Code is correct by inspection.
- (B) Coverage: zero test exercises this code path. Worklog § "Production Lister Coverage Note" (line 118-122) acknowledges the production `List` is not exercised by integration tests.

**Conclusion** Code is correct by inspection. Test coverage gap is documented and ACCEPTED in the worklog ("deferred to Drop 5 dogfood when a real claude install is part of the dev environment"). The deferment is reasonable (CI runner may not have `claude` installed; the binary's output is owned by Anthropic).

**Verdict** **NIT (acknowledged coverage gap).** The error-shape is correct by inspection; the deferment to dogfood is defensible. Recommend a `t.Skip` integration smoke test land in Drop 5+, as the worklog itself proposes.

---

### A5 — Production lister: claude binary returns non-zero exit

**Premises** When `claude plugin list --json` exits non-zero (e.g. claude itself errored), `execClaudePluginLister` must fail loudly with an actionable message — not silently parse stdout-as-empty as "no plugins."

**Evidence**
- `internal/app/dispatcher/plugin_preflight.go:282-292`:
  ```go
  if err := cmd.Wait(); err != nil {
      if cerr := bounded.Err(); cerr != nil {
          return nil, fmt.Errorf("dispatcher: claude plugin list canceled: %w", cerr)
      }
      var exitErr *exec.ExitError
      if errors.As(err, &exitErr) {
          return nil, fmt.Errorf("dispatcher: claude plugin list exited with code %d (stderr: %s)",
              exitErr.ExitCode(), strings.TrimSpace(stderrBuf.String()))
      }
      return nil, fmt.Errorf("dispatcher: claude plugin list wait: %w", err)
  }
  ```
- Three branches: ctx-cancel, `*exec.ExitError`, generic wait failure. All return error before reaching `parseClaudePluginList`.
- The exit-code branch DOES surface stderr tail in the message (line 288).

**Trace or cases**
- (A) `claude` exits 1 with stderr "command not recognized" → `cmd.Wait()` returns `*exec.ExitError` → branch fires → error names exit code + stderr tail. Stdout is NOT parsed. Correct.
- (B) Race: ctx expires between Start and Wait → `bounded.Err()` non-nil → ctx-cancel branch fires before exit-code branch. Correct.
- (C) Coverage: zero test exercises this code path. Same gap as A4.

**Conclusion** Code is correct by inspection. Same coverage-gap caveat as A4. The exit-code error message does NOT wrap a sentinel (`ErrPluginListUnparseable` reserved for parse failure; `ErrClaudeBinaryMissing` for not-found). Adopters who want to programmatically distinguish "exit=1" from "stderr=garbage" cannot via `errors.Is`. This is acceptable — the failure case is "claude itself broken" which is a hard-fail anyway.

**Verdict** **REFUTED.** The control flow is correct; the diagnostic surface is sufficient.

---

### A6 — Mock lister tests cover production wiring?

**Premises** Builder claims (worklog § "Production Lister Coverage Note") production lister NOT exercised. The `TestExecClaudePluginListerProductionWiring` test (`plugin_preflight_test.go:400-404`) only asserts the singleton type — does not invoke production code.

**Evidence**
- `plugin_preflight_test.go:400-404`:
  ```go
  func TestExecClaudePluginListerProductionWiring(t *testing.T) {
      if _, ok := defaultClaudePluginLister.(execClaudePluginLister); !ok {
          t.Fatalf("defaultClaudePluginLister type = %T; want execClaudePluginLister", defaultClaudePluginLister)
      }
  }
  ```
- The assertion is type-only. No exec.Command is invoked.
- Worklog explicitly accepts the deferment.

**Trace or cases**
- (A) A future refactor changes the singleton to a stub → this test catches it. The wiring guard is the most important property; functional behavior is gated by mock-driven `parseClaudePluginList*` tests covering the parse layer.
- (B) Adopter wants to verify "the binary exists on this dev's PATH" before first dispatch. Today, the verification only happens at first spawn. A `mage smoke-claude` target or similar would catch this — out of scope.

**Conclusion** Coverage gap is real and acknowledged. Smoke-test lands in Drop 5+ when dogfood begins.

**Verdict** **NIT (acknowledged coverage gap, deferred to Drop 5).**

---

### A7 — `RequiredPluginsForProject` hook signature rigidity

**Premises** Today's signature `func(domain.Project) []string` accepts only a Project. Future adopters wanting ctx-aware behavior (e.g. read from per-spawn config, refresh on per-call basis) cannot.

**Evidence**
- `internal/app/dispatcher/plugin_preflight.go:37`: `var RequiredPluginsForProject func(domain.Project) []string`.
- `internal/app/dispatcher/spawn.go:235-240`: invoked with `project` only, no `ctx`.
- The caller already has `ctx.Background()` at line 237 — so even if the hook took a context, the call site would supply Background today.

**Trace or cases**
- (A) Future adopter wants `func(ctx context.Context, p domain.Project) []string` for cancellation-aware lookup → breaking change to the hook signature. Adopter would need to rewrite their boot code. Not silent — compile error catches it.
- (B) Today there is no in-tree consumer needing ctx-awareness; the seam is opt-in and easy to swap.

**Conclusion** Future evolution requires a breaking change but the change is loud (compile error on adopters' boot wiring). Within the YAGNI principle, today's signature is the smallest concrete shape.

**Verdict** **REFUTED.** Future-proofing is not a current acceptance gate.

---

### A8 — `pluginIsInstalled` matching contract: scoped requirement, marketplace mismatch

**Premises** Required `["context7@official"]` against installed `{ID: "context7", Marketplace: "third-party"}` — scoped requirement should NOT match.

**Evidence**
- `internal/app/dispatcher/plugin_preflight.go:190-205` (`pluginIsInstalled`):
  ```go
  for _, row := range installed {
      if row.ID == "" { continue }
      if row.ID != name { continue }
      if scoped && row.Marketplace != marketplace { continue }
      return true
  }
  ```
- Scoped path: `scoped == true`, name match required, marketplace match required. Both must pass.
- Bare path: `scoped == false`, name match only, marketplace ignored.

- `plugin_preflight_test.go:149-192` — `TestCheckRequiredPluginsScopedRequirementMatchesScopedInstalled`:
  - "scoped match" — passes.
  - "scoped mismatch on marketplace" — wantErr: true.
  - "scoped requirement, bare-marketplace installed" — wantErr: true.

**Trace or cases**
- All three scoped-vs-installed cases covered by table tests. The matcher behaves as specified in the F.7.6 entry contract.

**Conclusion** REFUTED.

**Verdict** **REFUTED.**

---

### A9 — `ErrMissingRequiredPlugins` error message bounding

**Premises** With 100 missing plugins, `formatMissingPlugins` produces a 100-fragment semicolon-joined string. Bounded? Truncated?

**Evidence**
- `internal/app/dispatcher/plugin_preflight.go:224-230` (`formatMissingPlugins`):
  ```go
  func formatMissingPlugins(missing []string) string {
      parts := make([]string, 0, len(missing))
      for _, entry := range missing {
          parts = append(parts, fmt.Sprintf("%s (run: claude plugin install %s)", entry, entry))
      }
      return strings.Join(parts, "; ")
  }
  ```
- No truncation; no max length. Each missing entry contributes ~30-100 bytes. 100 missing → ~5-10KB error string. 1000 missing → ~50-100KB.

**Trace or cases**
- (A) Adopter declares 100 plugins via auto-generated TOML, none installed → error message ~10KB. Not catastrophic; surfaces in dispatcher logs + propagated to caller.
- (B) Pathological: 10000 plugins → ~1MB string. Memory-only; not heap-pressure relevant in production. Caller may log to stderr at full length.
- (C) `[tillsyn] requires_plugins` in production templates is unlikely to exceed 5-10 entries (the pattern is "the handful of MCP plugins this project needs"). The pathological case is not realistic.

**Conclusion** Code is correct, no DoS surface, no memory hazard. Lack of explicit truncation is acceptable for the realistic input shape.

**Verdict** **REFUTED (NIT-eligible: no truncation but realistic input bounded).**

---

### A10 — JSON forward-compat with type-mismatch on existing keys

**Premises** Builder's `TestParseClaudePluginListForwardCompatUnknownFields` (`plugin_preflight_test.go:357-366`) verifies UNKNOWN keys decode cleanly. But `encoding/json` errors on TYPE MISMATCH for KNOWN keys: a future claude version that changes `installPath` from string to object would fail parse.

**Evidence**
- `ClaudePluginListEntry`:
  ```go
  type ClaudePluginListEntry struct {
      ID          string `json:"id"`
      Marketplace string `json:"marketplace"`
      Version     string `json:"version"`
      InstallPath string `json:"installPath"`
  }
  ```
- `parseClaudePluginList` uses default `json.Unmarshal` — strict on type, lenient on unknown keys.
- Future shape `{"installPath": {"primary": "/x", "fallback": "/y"}}` → decode error → `ErrPluginListUnparseable` returned.

**Trace or cases**
- (A) Anthropic adds `installPath: object` in future claude → all spawns fail pre-flight with `ErrPluginListUnparseable`. Hard-fail visible to dev. Non-silent.
- (B) The matcher only consumes `ID` and `Marketplace`; `InstallPath` and `Version` are diagnostic-only. If `InstallPath` morphed, the matcher would still work IF the parse layer were lenient on it.
- (C) Defensive option: declare `InstallPath any` or `InstallPath json.RawMessage`. Adopted as forward-compat hedge.

**Conclusion** Today's behavior on type-change-of-known-key is loud failure (parse error → ErrPluginListUnparseable). This is defensible — silent ignore would mask a real schema break. The exposure is real but the failure is non-silent. Builder's TestParseClaudePluginListForwardCompatUnknownFields name is technically accurate (it tests UNKNOWN fields only) but a reader could misread it as "all forward-compat" — minor doc/test-name nit.

**Verdict** **NIT (test name precision).** Suggest renaming the test to `TestParseClaudePluginListForwardCompatUnknownKeys` or adding a sibling test that documents type-mismatch-on-known-key as a hard-fail (with rationale: better to fail loud than silent-skip on schema break).

---

### A11 — Concurrent hook reassignment

**Premises** `RequiredPluginsForProject` is a package-level `var` read at line 235 of spawn.go. Builder explicitly documents single-writer-at-boot expectation. If a dev reassigns mid-run, the read at line 235 races with the write.

**Evidence**
- `internal/app/dispatcher/plugin_preflight.go:35-37`:
  ```go
  // Concurrency: the hook is read once per BuildSpawnCommand call. Callers
  // MUST set it before the first spawn; reassigning under load is unsafe.
  var RequiredPluginsForProject func(domain.Project) []string
  ```
- `internal/app/dispatcher/spawn.go:235`: `if hook := RequiredPluginsForProject; hook != nil {` — reads to local variable, then nil-checks. The local-copy pattern means the function value snapshot doesn't change mid-call, but the read of the package var itself is unsynchronized.

**Trace or cases**
- (A) Adopter assigns hook at boot, never reassigns → no race. Documented contract honored.
- (B) Adopter reassigns mid-dispatch from a different goroutine (e.g. config-reload) → `go test -race` would flag this. Worklog acknowledges; constant-pattern is `sync.RWMutex`-protected (see `adaptersMu` precedent in `spawn.go:103-145`).
- (C) Severity assessment: matches the pattern for `defaultClaudePluginLister` (also `var`, also unsynchronized). Two single-writer-at-boot vars in the same package — consistent design.

**Conclusion** Race surface is real but consistent with `defaultClaudePluginLister` and `adaptersMap` (the latter properly RWMutex-protected). Documented contract makes it the adopter's responsibility. Adopters who follow the documented pattern have no race.

**Verdict** **NIT (consistency).** If a future drop hardens this surface, it should harden ALL three (`RequiredPluginsForProject`, `defaultClaudePluginLister`, `adaptersMap` already done) for consistency. The asymmetry is the lint-worthy detail, not the underlying choice.

---

### A12 — `spawn.go` integration: pre-flight ordering + caching

**Premises** Builder's worklog says pre-flight runs "between binding validate and binding resolve." Confirm the ordering is correct (validation runs before bundle materialization, as expected for fail-fast). Performance: per-spawn exec cost.

**Evidence**
- `internal/app/dispatcher/spawn.go:206-240`:
  - line 206: `LookupAgentBinding`.
  - line 214: `rawBinding.Validate()`.
  - line 218-240: pre-flight (NEW).
  - line 246: `ResolveBinding`.
  - line 248: `lookupAdapter`.
  - line 273: `NewBundle`.

- Ordering: lookup → validate → **pre-flight** → resolve → adapter → materialize. Pre-flight runs BEFORE expensive bundle materialization, so a missing plugin fails before any temp dir is created. Correct fail-fast ordering.

- F7_CORE_PLAN.md line 471: *"Pre-dispatch invocation is fast: `claude plugin list --json` typically <50ms; plus per-call result is NOT cached across dispatches (per-dispatch confirmation is the point)."* Plan-level decision: caching is anti-feature (would mask plugin-uninstall between spawns — Attack #7 mitigation).

**Trace or cases**
- (A) Empty `RequiresPlugins` for a spawn → hook returns `[]` or nil → `CheckRequiredPlugins` short-circuits before invoking lister → zero exec cost. Verified by `TestCheckRequiredPluginsEmptyRequiredReturnsNil`.
- (B) Non-empty `RequiresPlugins` → 50ms per spawn. Acceptable per plan.
- (C) Hook is nil → pre-flight skipped entirely. Zero exec cost.

**Conclusion** REFUTED. Ordering is fail-fast correct. Per-dispatch invocation matches the plan's deliberate no-cache choice.

**Verdict** **REFUTED.**

---

### A13 — No-commit per REV-13

**Premises** REV-13: builder MUST NOT run `git commit`. Orchestrator commits post-QA-green.

**Evidence**
- `git status --short`:
  ```
  M internal/app/dispatcher/spawn.go
  M internal/templates/load.go
  M internal/templates/load_test.go
  M internal/templates/schema.go
  ?? internal/app/dispatcher/plugin_preflight.go
  ?? internal/app/dispatcher/plugin_preflight_test.go
  ?? workflow/drop_4c/4c_F7_6_BUILDER_WORKLOG.md
  ```
- No commit landed for F.7.6 (last commit is `f6aec8b feat(templates): add tool-gating + sandbox + sysprompt fields (4c F.7.2)`).
- Worklog line 99: `- [x] **NO commit by builder** per F.7-CORE REV-13.`

**Trace or cases** Builder honored REV-13.

**Conclusion** REFUTED.

**Verdict** **REFUTED.**

---

### A14 — Memory-rule conflicts

**Premises** Audit against memory rules:
- `feedback_no_migration_logic_pre_mvp.md` — no DDL.
- `feedback_subagents_short_contexts.md` — single-package surface.
- `feedback_orphan_via_collapse_defer_refinement.md` — defer dead code via refinement.
- `feedback_decomp_small_parallel_plans.md` — single-surface droplet OK.
- `feedback_no_closeout_md_pre_dogfood.md` — no closeout MDs from builder; just worklog.

**Evidence**
- No SQL changes. No DDL. No `till migrate` CLI.
- Two surfaces touched: `internal/templates` + `internal/app/dispatcher`. Both are single-author droplet scope; not a multi-builder split. F.7.6 surface is reasonable.
- Worklog is the only artifact authored. No CLOSEOUT/LEDGER/WIKI_CHANGELOG/REFINEMENTS — compliant with `feedback_no_closeout_md_pre_dogfood`.
- Builder did not invoke `mage install`. Build verification used `mage ci`. Compliant with the project rule "NEVER run `mage install`."

**Trace or cases** No memory-rule violations detected.

**Conclusion** REFUTED.

**Verdict** **REFUTED.**

---

## Summary Table

| Attack | Verdict |
|--------|---------|
| A1 hook-nil silent skip footgun | **NIT** (documented hazard, route to refinement) |
| A2 Tillsyn vs per-binding scope | REFUTED |
| A3 `@` form parsing edge cases | REFUTED |
| A4 production lister: claude missing | NIT (acknowledged coverage gap) |
| A5 production lister: non-zero exit | REFUTED |
| A6 mock vs production wiring coverage | NIT (acknowledged coverage gap) |
| A7 hook signature rigidity | REFUTED |
| A8 `pluginIsInstalled` matching contract | REFUTED |
| A9 error-message bounding | REFUTED (NIT-eligible) |
| A10 JSON type-mismatch on known keys | NIT (test name precision) |
| A11 concurrent hook reassignment | NIT (consistency w/ `defaultClaudePluginLister`) |
| A12 spawn.go pre-flight ordering + caching | REFUTED |
| A13 no-commit per REV-13 | REFUTED |
| A14 memory-rule conflicts | REFUTED |

---

## Additional Finding (out of attack list)

### F1 — No spawn-test integration coverage of the hook through `BuildSpawnCommand`

**Premises** F.7.6 wires `RequiredPluginsForProject` into `BuildSpawnCommand` at `spawn.go:235-240`. Hook tests (`TestRequiredPluginsForProjectHookDefaultIsNil`, `TestRequiredPluginsForProjectHookReceivesProject`) only test the variable in isolation. NO test asserts `BuildSpawnCommand` itself short-circuits when the hook is nil, nor that it propagates `ErrMissingRequiredPlugins` when the hook returns required plugins missing from the lister.

**Evidence**
- `internal/app/dispatcher/spawn_test.go` — search for `RequiredPluginsForProject` / `CheckRequiredPlugins` returns 0 matches.
- The hook integration through `BuildSpawnCommand` is exercised only at the unit level on `CheckRequiredPlugins` directly.

**Trace or cases**
- A future refactor that breaks the spawn.go integration (e.g. hook called with wrong project, hook called after bundle materialization, hook nil-check inverted) would not be caught by current tests.
- This is the symmetric concern to F.7.1's `TestBuildSpawnCommandWritesManifestJSON` — F.7.1 added end-to-end manifest assertion through `BuildSpawnCommand`. F.7.6 did not add the equivalent end-to-end pre-flight assertion.

**Conclusion** Test coverage gap. Catch-able by:
- `TestBuildSpawnCommandSkipsPreflightWhenHookNil` — set hook to nil, verify no exec call (mock the lister, assert Calls=0).
- `TestBuildSpawnCommandPropagatesMissingPluginError` — set hook to non-nil returning `["missing-plugin"]`, override `defaultClaudePluginLister` with empty fake, assert returned err `errors.Is(_, ErrMissingRequiredPlugins)`.
- `TestBuildSpawnCommandHookReceivesResolvedProject` — set hook to capture project, assert observed project matches the input.

**Verdict** **NIT (test gap, route to refinement).** Same severity tier as A1 — it's a documented hazard plus an unobserved invariant.

---

## Final Overall Verdict

**PASS-WITH-NITS.** No CONFIRMED counterexamples against F.7.6's acceptance contract as plan-revised (no `till bootstrap` available; per-dispatch hook seam is the documented pivot). The five NITs (A1, A4, A6, A10, A11) plus F1 collectively suggest a **single follow-up refinement** worth raising:

> Drop 4c F.7.6 follow-up: harden the pre-flight hook surface — (a) add boot-time validation that `RequiredPluginsForProject` is non-nil when any project's `requires_plugins` is non-empty (A1); (b) add `BuildSpawnCommand` integration tests for the pre-flight path (F1); (c) add an opt-in production-lister smoke test gated on `claude` being on PATH (A4 + A6); (d) consider RWMutex-protecting `RequiredPluginsForProject` for symmetry with `adaptersMap` (A11); (e) rename `TestParseClaudePluginListForwardCompatUnknownFields` to `…UnknownKeys` and add a sibling test pinning type-mismatch-on-known-key as a hard-fail (A10).

The droplet's core behavior — schema validator, matcher, parser, error sentinels, REV-7-compliant struct extension, REV-13 no-commit — is correct by inspection and well-tested at the unit level.

---

## Hylla Feedback

`N/A — task touched non-Go reference docs (workflow MDs, plan MDs) only for cross-checking; Go evidence was directed by the spawn prompt's explicit file paths.`

The spawn prompt enumerated every file the builder touched. Direct `Read` on each file plus a couple of `grep` calls satisfied every evidence need. No Hylla queries issued; no miss to record.
