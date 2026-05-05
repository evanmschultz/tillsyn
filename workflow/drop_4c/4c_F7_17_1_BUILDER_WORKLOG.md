# Drop 4c — F.7.17.1 Builder Worklog

## Droplet

`4c.F.7.17.1` — Schema-1: per-binding `Env []string` + `CLIKind string` fields on `templates.AgentBinding` plus the `validateAgentBindingEnvNames` Load-time validator.

REVISIONS-first compliance: read REV-1 + REV-2 before reading the body. REV-1 stripped `Command []string`, `ArgsPrefix []string`, the shell-interpreter denylist, and the `validateAgentBindingCommandTokens` validator from this droplet — only `Env` + `CLIKind` plus the env-name validator ship.

## Files edited

- `internal/templates/schema.go` — added `Env []string` (TOML tag `env`) and `CLIKind string` (TOML tag `cli_kind`) at the end of the `AgentBinding` struct, with full doc-comments anchoring them to F.7.17 locked decisions L4 / L5 / L6 / L8 / L15 and REV-1.
- `internal/templates/load.go`:
  - Added `regexp` import.
  - Extended the `Load` doc-comment validator-order list with step 4(f).
  - Wired `validateAgentBindingEnvNames(tpl)` into `Load` after `validateGateKinds`.
  - Added sentinel `ErrInvalidAgentBindingEnv = fmt.Errorf("%w: env", ErrInvalidAgentBinding)` so callers using `errors.Is(err, ErrInvalidAgentBinding)` continue to route correctly.
  - Added `var envVarNameRegex = regexp.MustCompile(\`^[A-Za-z][A-Za-z0-9_]*$\`)` — explicit anchors inside the pattern AND `MatchString` against a compiled regex per L4 / R2 falsification A1.c.
  - Added `validateAgentBindingEnvNames(tpl Template) error` that rejects empty entries, `=`-containing entries (with a precise pre-regex error message), regex-non-matching names, and within-binding duplicates.
- `internal/templates/agent_binding_test.go` — extended `fullyPopulatedAgentBinding` to populate `Env` (`["ANTHROPIC_API_KEY", "https_proxy", "HTTP_PROXY"]` — exercises both uppercase + lowercase per L5 / REV-2) and `CLIKind = "claude"`. Updated round-trip header comment from "11 fields" to "13 fields."
- `internal/templates/schema_test.go` — extended `TestTemplateTOMLRoundTrip`'s populated `AgentBindings` literal with `Env` + `CLIKind` so the round-trip assertion catches a TOML-tag drop on the new fields.
- `internal/templates/load_test.go` — added five new tests:
  - `TestLoadAgentBindingEnvAndCLIKindHappyPath` — TOML stream with `env = ["ANTHROPIC_API_KEY", "https_proxy", "HTTP_PROXY"]` + `cli_kind = "claude"` decodes cleanly; entries preserved verbatim.
  - `TestLoadAgentBindingCLIKindOmittedDefaultsToEmpty` — omitted `cli_kind` leaves the field at empty string (per L15 the empty-string sentinel resolves to "claude" at adapter-lookup time, NOT at Load time); omitted `env` decodes to nil.
  - `TestLoadAgentBindingEnvRejectionTable` — table-driven coverage of every reject case: `KEY=value`, empty entry, duplicate `["FOO", "FOO"]`, whitespace `"FOO BAR"`, hyphen `"FOO-BAR"`, dot `"FOO.BAR"`, leading-digit `"1FOO"`. Plus four happy-path rows verifying lowercase (`https_proxy`, `foo_bar`), uppercase (`HTTP_PROXY`), and trailing-digit (`FOO123`) names pass. Each rejection asserts `errors.Is(err, ErrInvalidAgentBindingEnv)` AND `errors.Is(err, ErrInvalidAgentBinding)` (sentinel-wrap discipline).
  - `TestLoadAgentBindingDuplicateEnvNamesAcrossBindingsAllowed` — duplicate detection is per-binding, NOT cross-binding; two different bindings each declaring `ANTHROPIC_API_KEY` is legal.
  - `TestLoadAgentBindingStrictDecodeUnknownFieldStillRejects` — regression coverage that adding `Env`/`CLIKind` to the struct doesn't relax strict decode for any other key. Bogus `bogus_field = true` inside `[agent_bindings.build]` still surfaces as `ErrUnknownTemplateKey`.

## NOT edited (per REV-1 supersession)

- No `Command []string` field added.
- No `ArgsPrefix []string` field added.
- No `shellInterpreterDenylist` constant added.
- No `validateAgentBindingCommandTokens` validator added.
- No per-token argv regex added.

The diff is strictly the F.7.17.1 minimum surface after REV-1.

## Verification

- `mage check` (alias of `mage ci`) — green. 2281 tests / 2280 passed / 1 skipped (unrelated steward integration test pre-existing skip) / 0 failed across 21 packages.
- `internal/templates` package coverage: 96.3% (well above the 70% gate).
- `mage testPkg ./internal/templates` — 268 tests pass, no race regressions.

## Acceptance criteria — all met

- [x] `AgentBinding.Env []string` added with TOML tag `env` + Go doc-comment.
- [x] `AgentBinding.CLIKind string` added with TOML tag `cli_kind` + Go doc-comment.
- [x] `validateAgentBindingEnvNames` function wired into `templates.Load` validator chain (after `validateGateKinds`, in step 4f of the documented validator order).
- [x] All test scenarios listed in the spawn prompt are real, named tests:
  - Happy path with mixed-case env + cli_kind.
  - Default cli_kind (omitted → empty).
  - Reject `=` in entry.
  - Reject empty entry.
  - Reject duplicate within binding.
  - Reject malformed names (whitespace, hyphen, dot, leading digit).
  - Allow lowercase + trailing digits.
  - Strict-decode unknown-key on AgentBinding (regression).
- [x] `mage check` passes.
- [x] `mage ci` passes (full test suite + race + coverage; same target as `mage check`).
- [x] **NO** Command, ArgsPrefix, denylist, or command-token validator anywhere in the diff.
