# Drop 4c — F.7.17.1 Builder QA Falsification (Round 1)

**Droplet:** `4c.F.7.17.1` — Schema-1: `Env []string` + `CLIKind string` on `templates.AgentBinding` plus `validateAgentBindingEnvNames`.

**Reviewer mode:** Read-only adversarial. No Hylla calls (per spawn prompt). No code edits.

**Builder output reviewed:**

- `internal/templates/schema.go`
- `internal/templates/load.go`
- `internal/templates/agent_binding_test.go`
- `internal/templates/schema_test.go`
- `internal/templates/load_test.go`
- `workflow/drop_4c/4c_F7_17_1_BUILDER_WORKLOG.md`

---

## Verdict summary

**Overall: PASS-WITH-NITS.**

12 spawn-prompt attacks evaluated, plus 1 self-generated attack (A15). All CONFIRMED-counterexample candidates refuted. Three findings classified NIT — none block merge. Builder honored REV-1 strictly: no `Command`, no `ArgsPrefix`, no denylist, no command-token validator. Env + CLIKind ship clean.

| Attack | Verdict |
|---|---|
| A1 — Regex anchor leak | REFUTED |
| A2 — Duplicate detection cross-binding scope | REFUTED |
| A3 — Empty array semantics | NIT |
| A4 — CLIKind empty default | REFUTED |
| A5 — TOML tag bug | REFUTED |
| A6 — Strict-decode regression for new fields | REFUTED |
| A7 — Existing-test regression | REFUTED |
| A8 — Validator chain ordering | REFUTED |
| A9 — Sentinel error wrapping | REFUTED |
| A10 — POSIX leading-underscore permissiveness | NIT |
| A11 — Round-trip nil-vs-empty | NIT |
| A12 — Memory rule conflicts (migration) | REFUTED |
| A15 (self-gen) — Map iteration determinism | NIT |

---

## Per-attack detail

### A1. Regex anchor leak — REFUTED

`internal/templates/load.go:348` declares `var envVarNameRegex = regexp.MustCompile(\`^[A-Za-z][A-Za-z0-9_]*$\`)` at package scope — compiled once, not per-validate. Pattern has explicit `^` start and `$` end anchors AND the call uses `MatchString` (line 387), not `FindString` or substring search. Constructed counterexample `"FOO; rm -rf /"`: contains `;` and space, neither in `[A-Za-z0-9_]` character class — fails the class membership check independent of anchors. Even with no anchors the substring would still fail. Cannot construct a token that bypasses.

### A2. Duplicate detection cross-binding scope — REFUTED

`internal/templates/load.go:377-396` — outer loop `for kind, binding := range tpl.AgentBindings` instantiates `seen := make(map[string]struct{}, len(binding.Env))` INSIDE the loop (line 379). Each binding gets its own `seen` map. `TestLoadAgentBindingDuplicateEnvNamesAcrossBindingsAllowed` (load_test.go:617-633) seeds two bindings each with `env = ["ANTHROPIC_API_KEY"]` and asserts Load succeeds. Cross-binding behavior is correct AND test-pinned.

### A3. Empty array semantics — NIT

Walk validator: outer iteration over `binding.Env` exits cleanly when `len == 0` — no false-positive error. So `env = []` decodes and validates fine. The omitted-`env`-key path is test-pinned (`TestLoadAgentBindingCLIKindOmittedDefaultsToEmpty` line 486-488: asserts `binding.Env != nil` rejected when omitted, so omitted → nil). The non-empty-array path is test-pinned (`TestLoadAgentBindingEnvAndCLIKindHappyPath`).

The literal `env = []` (empty TOML array, distinct from omitted) is NOT test-pinned. Pelletier v2 will decode `env = []` to `[]string{}` (non-nil zero-length), distinct from `nil`. Validator handles both identically (zero iterations either way) so no functional defect. Worth a one-line test for documentation value:

```go
// Suggested: in load_test.go
func TestLoadAgentBindingEnvLiteralEmptyArrayAllowed(t *testing.T) {
    src := `... env = [] ...`
    tpl, err := Load(strings.NewReader(src))
    // assert err == nil and len(binding.Env) == 0
}
```

NIT, not blocking. The validator is correct; the test gap is on observable boundary distinction.

### A4. CLIKind empty default — REFUTED

`schema.go:371` declares `CLIKind string \`toml:"cli_kind"\`` with no validation hook. `validateAgentBindingEnvNames` (the only new validator) doesn't reference CLIKind (load.go:377-397). `AgentBinding.Validate()` (schema.go:393-416) doesn't reference CLIKind. Doc-comment (schema.go:358-371) explicitly states: "Validation against the closed set is performed at adapter-lookup time, NOT at template Load time, so a template authored against a future Tillsyn release that adds new CLIKind values still loads cleanly under an older binary."

`TestLoadAgentBindingCLIKindOmittedDefaultsToEmpty` (load_test.go:467-489) confirms empty resolves without error. The happy-path test (`TestLoadAgentBindingEnvAndCLIKindHappyPath`) sets `cli_kind = "claude"`. Implicit: arbitrary strings are accepted (no `IsValidCLIKind` introduced). Closed-enum lands in F.7.17.6 per plan; F.7.17.1 stays minimal as REV-1 demands.

### A5. TOML tag bug — REFUTED

`schema.go:356` — `Env []string \`toml:"env"\`` (lowercase, no underscore needed for single word). Line 371 — `CLIKind string \`toml:"cli_kind"\`` (snake_case). Both correct. `TestAgentBindingTOMLRoundTrip` (agent_binding_test.go:46-62) marshals + unmarshals a populated binding and asserts `reflect.DeepEqual` — would catch any tag drop or case-fold confusion via the round-trip. Also covered by `TestTemplateTOMLRoundTrip` at the Template level (schema_test.go:53-112) where the binding rides inside a Template.

### A6. Strict-decode regression for new fields — REFUTED

`TestLoadAgentBindingStrictDecodeUnknownFieldStillRejects` (load_test.go:643-662) declares `bogus_field = true` inside `[agent_bindings.build]` and asserts:

1. `errors.Is(err, ErrUnknownTemplateKey)` — sentinel routing.
2. `strings.Contains(err.Error(), "bogus_field")` — UX message names the offender.

Walk pelletier path: load.go:93-100 wires `strictDecoder.DisallowUnknownFields()` → an unknown nested field surfaces as `*toml.StrictMissingError` → wrapped via `errors.AsType` typed extraction at line 96-98 into `ErrUnknownTemplateKey` with the strict error text. Pre-existing strict-decode coverage at the top-level (`TestLoadRejectionTable`'s "unknown top-level key rejected" row, load_test.go:98-106) confirms the strict pass works generally; the new test extends coverage to the nested `[agent_bindings.<kind>]` table specifically. Adding `Env`/`CLIKind` to the struct does not relax strict decode for any other nested key — pelletier compares struct tags against TOML keys; new struct fields just enlarge the accepted set, they don't broaden it.

### A7. Existing-test regression — REFUTED

`fullyPopulatedAgentBinding` (agent_binding_test.go:22-38) gained two field assignments: `Env: []string{"ANTHROPIC_API_KEY", "https_proxy", "HTTP_PROXY"}` and `CLIKind: "claude"`. Walk every consumer:

- `TestAgentBindingTOMLRoundTrip` — exercises `reflect.DeepEqual` on the new shape; expanded coverage, not regression.
- `TestAgentBindingValidate` (lines 68-190) — table-driven; each row mutates a single field and runs `Validate()`. None of the rows touch `Env` or `CLIKind` because `Validate()` doesn't validate those (env validation is at Load time, not on the binding's per-field rules). The "valid fully populated binding passes" row continues to pass because Env values pass the regex AND CLIKind is non-empty (but Validate doesn't check CLIKind).
- `TestAgentBindingDurationStringWireForm` (lines 199-240) — uses inline TOML literal, not `fullyPopulatedAgentBinding`. Unaffected.
- `TestAgentBindingValidateZeroValueRejected` (lines 247-256) — uses zero-value `var b AgentBinding`. Unaffected.

No existing assertion broke. The new field population is additive.

### A8. Validator chain ordering — REFUTED

`internal/templates/load.go:102-122` runs validators in this order:

1. `validateMapKeys` (Kinds + AgentBindings + Gates map-key membership)
2. `validateChildRuleKinds` (child-rule kind references)
3. `validateChildRuleCycles` (DFS cycle check)
4. `validateChildRuleReachability` (no-op today)
5. `validateGateKinds` (gate-kind value membership)
6. `validateAgentBindingEnvNames` (NEW)

Walk predecessors for any that read AgentBinding's *content* (not just map keys):

- `validateMapKeys` reads `tpl.AgentBindings` keys only — line 199-203 — never the binding values themselves.
- `validateChildRuleKinds` / `validateChildRuleCycles` / `validateChildRuleReachability` operate on `tpl.ChildRules`, never on AgentBindings.
- `validateGateKinds` operates on `tpl.Gates` (map[Kind][]GateKind), never on AgentBindings.

None of the predecessors validate AgentBinding field content. Placing `validateAgentBindingEnvNames` last in the chain is correct: kind-membership is asserted first (so a binding with a bogus map key is caught before its env list is even traversed), then env-content validation fires on a vocabulary-clean shape. Counter-ordering attempt: a malformed env name CAN'T sneak past a higher validator because no higher validator examines env content.

### A9. Sentinel error wrapping — REFUTED

`internal/templates/load.go:182` declares `ErrInvalidAgentBindingEnv = fmt.Errorf("%w: env", ErrInvalidAgentBinding)`. The `%w` verb makes `ErrInvalidAgentBindingEnv` wrap `ErrInvalidAgentBinding`. Two-level chain:

- Validator returns `fmt.Errorf("%w: agent_bindings[%q].env entry %q ...", ErrInvalidAgentBindingEnv, kind, entry)` — wraps ErrInvalidAgentBindingEnv (level 1).
- ErrInvalidAgentBindingEnv itself wraps ErrInvalidAgentBinding (level 2).

`errors.Is` walks the wrap chain, so:

- `errors.Is(err, ErrInvalidAgentBindingEnv)` → true (level 1).
- `errors.Is(err, ErrInvalidAgentBinding)` → true (level 2 reachable via Is).

`TestLoadAgentBindingEnvRejectionTable` (load_test.go:505-610) asserts BOTH sentinels for every reject row (lines 597-604):

```go
if tc.wantSentinel != nil && !errors.Is(err, tc.wantSentinel) { ... }
if !errors.Is(err, ErrInvalidAgentBinding) { ... }
```

Wrap direction is correct (specific wraps general, not inverted). Existing handlers routing on `ErrInvalidAgentBinding` continue to match.

### A10. POSIX leading-underscore permissiveness — NIT

The regex `^[A-Za-z][A-Za-z0-9_]*$` rejects leading-underscore names like `_FOO`, `_JAVA_OPTIONS`, `__TEST`. POSIX IEEE Std 1003.1 `Name` definition allows leading underscore: "consisting solely of underscores, digits, and alphabetics ... the first character of a name is not a digit." Per POSIX, `_JAVA_OPTIONS` (a real-world JVM idiom) and `_AWS_REGION` (uncommon but legal) are valid env var names.

The plan spec (locked decision L5 referenced in REVISIONS section) reads "leading letter, trailing alphanumerics + underscore" — the implemented regex matches the spec text. So this is intentional, not a bug. But:

- Schema doc-comment at `schema.go:343-349` says only "uppercase OR lowercase leading letter" — does not call out that POSIX-legal leading-underscore is rejected.
- `load.go:336-348` (envVarNameRegex commentary) cites L5 + falsification round 2 A2.d but does not call out the underscore-leading rejection.

Adopters configuring `_JAVA_OPTIONS` will get an unhelpful regex-mismatch error message without an explanation of WHY the leading underscore was rejected. Optional improvement: either (a) widen the regex to `^[A-Za-z_][A-Za-z0-9_]*$` matching POSIX, or (b) add a one-liner doc-comment note: "Leading underscore is rejected per L5 spec text; POSIX-legal names like `_JAVA_OPTIONS` must be aliased through a baseline-allowed name in the adapter rather than declared in `env`."

NIT — not a defect, but a docs/spec-narrowness gap that will surface as adopter friction.

### A11. Round-trip nil-vs-empty — NIT

`TestAgentBindingTOMLRoundTrip` exercises `Env != nil` + populated. `TestLoadAgentBindingCLIKindOmittedDefaultsToEmpty` exercises omitted (decodes to nil). The literal `env = []` empty-array form is NOT test-pinned. Pelletier v2 distinguishes:

- omitted key → struct field at Go zero value (nil for slice).
- `env = []` → `[]string{}` (non-nil zero-length).
- `env = ["x"]` → `[]string{"x"}` (non-nil populated).

A consumer relying on `binding.Env == nil` to mean "no override declared" vs `len(binding.Env) == 0` would behave differently for the two empty forms. Today no consumer code exists (F.7.17.5+ wires consumers); the validator treats both empty forms identically. No functional defect at this droplet's scope, but worth a regression test before consumers land:

```go
// Suggested for F.7.17.5 dependency-discipline:
// Assert that pelletier-roundtripped `env = []` survives marshal+unmarshal as []string{} (non-nil),
// distinct from omitted (nil), so consumer code can rely on the distinction OR explicitly normalize.
```

NIT — defer assertion to F.7.17.5 when the consumer cares.

### A12. Memory rule conflicts (migration) — REFUTED

Diff is in-process Go struct + Go validator + Go tests. No SQL, no DB migration code, no TOML→SQLite persistence path. Templates load from TOML files at process startup; the new fields ride alongside the existing 11 AgentBinding fields with the same in-memory semantics. Compatible with `feedback_no_migration_logic_pre_mvp.md` (no migration logic) AND with the project's TOML-as-template-wire-format model. Templates aren't persisted to SQLite; they're parsed at startup.

### A15 (self-generated). Map iteration determinism — NIT

`validateAgentBindingEnvNames` iterates `tpl.AgentBindings` (map[domain.Kind]AgentBinding). Go map iteration is randomized. If two different bindings each contain a malformed env entry, which one fires first is non-deterministic — error UX could surface either kind name on different runs.

Tests in `TestLoadAgentBindingEnvRejectionTable` use a single binding (`agent_bindings.build`) per row, so the test never observes the non-determinism. But for production UX consistency, a deterministic error path is friendlier. Optional improvement: sort `tpl.AgentBindings` keys before iteration:

```go
// Suggested:
kinds := make([]domain.Kind, 0, len(tpl.AgentBindings))
for k := range tpl.AgentBindings { kinds = append(kinds, k) }
sort.Slice(kinds, func(i, j int) bool { return string(kinds[i]) < string(kinds[j]) })
for _, kind := range kinds {
    binding := tpl.AgentBindings[kind]
    ...
}
```

Same pattern as `validateMapKeys` would benefit (it's also non-deterministic for the same reason). NIT — not a defect; reproducibility-of-error-messages improvement.

---

## Build verification cross-check

Worklog claims:

- `mage check` — green. 2281 / 2280 / 1 skipped / 0 failed across 21 packages.
- `internal/templates` package coverage: 96.3%.
- `mage testPkg ./internal/templates` — 268 tests pass.

I did not re-run `mage` (read-only review). Builder's claim is consistent with the test files I read — the new tests are well-formed, named, and exercise distinct cases. If QA Proof's verification confirmed the build, I have no counterevidence to flip the verdict.

---

## REV-1 compliance check

REV-1 supersedes the body of `F7_17_CLI_ADAPTER_PLAN.md` and reduces F.7.17.1 to ONLY `Env` + `CLIKind`. Walked the diff for forbidden artifacts:

- `Command []string` field — ABSENT from `AgentBinding`. Refuted attack against REV-1.
- `ArgsPrefix []string` field — ABSENT.
- `shellInterpreterDenylist` constant — ABSENT.
- `validateAgentBindingCommandTokens` validator — ABSENT.
- Per-token argv regex — ABSENT.
- Marketplace-install validation hooks (REV-4) — ABSENT.

Builder honored REVISIONS-FIRST discipline. REV-1 compliance: clean.

---

## Hylla Feedback

N/A — read-only review, no Hylla calls per spawn prompt's hard constraint.

---

## Final verdict

**PASS-WITH-NITS.**

Three NITs, none blocking:

1. **A3** — `env = []` literal not test-pinned (test gap, not defect).
2. **A10** — POSIX-leading-underscore rejection not called out in doc-comment; widening to `^[A-Za-z_][A-Za-z0-9_]*$` could be reconsidered; or annotate the doc-comment to head off adopter confusion.
3. **A11** — pelletier nil-vs-empty distinction (`env = []` vs omitted) deferred to F.7.17.5 consumer-discipline.
4. **A15** — non-deterministic error UX for multi-binding malformations; sort the map keys.

All 12 spawn-prompt attack vectors refuted on substance OR classified as NIT. REV-1 compliance is clean. Builder shipped exactly the F.7.17.1 minimum surface.

Recommend orchestrator merges this droplet and routes the four NITs as F.7.17 follow-up refinements (small enough to fold into a later F.7.17 droplet's diff or into F.7.17.5 consumer scope) rather than a re-spawn.
