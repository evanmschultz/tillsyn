# Drop 4c — F.7.17.1 Builder QA-Proof Round 1

**Verdict:** PROOF GREEN

**Reviewer:** go-qa-proof-agent
**Round:** 1
**Date:** 2026-05-04

---

## Scope of review

REVISIONS-first: read F7_17_CLI_ADAPTER_PLAN.md REV-1 (lines 685-714) before body. REV-1 strips `Command []string`, `ArgsPrefix []string`, the shell-interpreter denylist, and the `validateAgentBindingCommandTokens` validator. F.7.17.1 ships ONLY `Env []string` + `CLIKind string` + `validateAgentBindingEnvNames`. Every acceptance criterion below is checked against the post-REV-1 contract, not the original body.

Files reviewed (per `git diff --name-only internal/templates/`):

- `internal/templates/schema.go` (+40)
- `internal/templates/load.go` (+82)
- `internal/templates/load_test.go` (+242)
- `internal/templates/agent_binding_test.go` (+17/-5)
- `internal/templates/schema_test.go` (+2)

The other modified files in `git status` (`internal/adapters/storage/sqlite/repo_test.go`, `internal/domain/*`, `internal/tui/model_test.go`, `workflow/drop_4c/SKETCH.md`) are concurrent in-flight work (F.7.10 + adjacent droplets) and not part of this droplet's commit boundary; they were ignored.

---

## Per-criterion evidence

### Criterion 1 — `Env []string` + `CLIKind string` added with TOML tags + doc-comments

**Evidence:**

- `schema.go:356` — `Env []string \`toml:"env"\`` (doc-comment lines 333-355 anchors L4 / L5 / L6 / L8 + REV-1).
- `schema.go:371` — `CLIKind string \`toml:"cli_kind"\`` (doc-comment lines 358-370 anchors L15 closed-enum + back-compat-default-empty contract).

Doc-comments cite locked decisions and REV-1 supersession. Both fields appended to end of `AgentBinding` struct (positionally — keeps the closed schema's existing field order intact). PASS.

### Criterion 2 — `validateAgentBindingEnvNames` wired into Load chain after `validateGateKinds`

**Evidence:**

- `load.go:117-119` — `validateGateKinds(tpl)` call.
- `load.go:120-122` — `validateAgentBindingEnvNames(tpl)` call (immediately after).
- `load.go:36-53` — Load doc-comment validator-order list extended with step 4(f) explicitly documenting the new validator.

Order is correct: env-name validation runs after gate-kind validation, after all child-rule validators, after `validateMapKeys`. PASS.

### Criterion 3 — Regex pinned literally as `regexp.MustCompile(\`^[A-Za-z][A-Za-z0-9_]*$\`).MatchString(token)`

**Evidence:**

- `load.go:348` — `var envVarNameRegex = regexp.MustCompile(\`^[A-Za-z][A-Za-z0-9_]*$\`)`.
- `load.go:387` — call site: `if !envVarNameRegex.MatchString(entry) { ... }`.

Literal-anchor match per A1.c. The pattern uses raw-string literal so backslashes are not at issue; `^` and `$` are explicit anchors inside the pattern; the call uses `MatchString` against a precompiled regex (NOT `regexp.Match` without a compiled value, NOT a substring fallback). PASS.

### Criterion 4 — All 5 test scenarios added covering required cases

**Evidence (load_test.go:421-662):**

- `TestLoadAgentBindingEnvAndCLIKindHappyPath` (line 426) — happy path: `env = ["ANTHROPIC_API_KEY", "https_proxy", "HTTP_PROXY"]` + `cli_kind = "claude"` decode cleanly; ordering preserved; CLIKind populated correctly.
- `TestLoadAgentBindingCLIKindOmittedDefaultsToEmpty` (line 467) — omitted `cli_kind` → empty string sentinel (per L15); omitted `env` → nil slice.
- `TestLoadAgentBindingEnvRejectionTable` (line 502) — table-driven, exhaustive: `KEY=value`, empty entry, `["FOO", "FOO"]` duplicate, `"FOO BAR"`, `"FOO-BAR"`, `"FOO.BAR"`, `"1FOO"` reject; `https_proxy`, `foo_bar`, `HTTP_PROXY`, `FOO123` allow. Each reject asserts BOTH `errors.Is(err, ErrInvalidAgentBindingEnv)` AND `errors.Is(err, ErrInvalidAgentBinding)`.
- `TestLoadAgentBindingDuplicateEnvNamesAcrossBindingsAllowed` (line 633) — per-binding (NOT cross-binding) duplicate-detection scope.
- `TestLoadAgentBindingStrictDecodeUnknownFieldStillRejects` (line 651) — regression on strict decode: bogus field under `[agent_bindings.build]` still surfaces `ErrUnknownTemplateKey`.

All 5 spawn-prompt scenarios mapped onto distinct test functions. PASS.

### Criterion 5 — NO `Command`, `ArgsPrefix`, denylist, or command-token validator in diff

**Evidence:**

- `rg "ArgsPrefix" internal/templates/` returns only doc-comment text in `schema.go:352, 370` referencing REV-1 supersession. NO struct field, NO logic.
- `rg "shellInterpreterDenylist|validateAgentBindingCommandTokens|commandTokenRegex" internal/templates/` returns no matches.
- `rg "Command \[\]string" internal/templates/` returns only the schema.go:352 supersession-doc-comment line, NOT a struct field declaration.
- The `AgentBinding` struct (`schema.go:285-372`) ends at `CLIKind` — no `Command` or `ArgsPrefix` field.

REV-1 supersession honored cleanly. PASS.

### Criterion 6 — Test fixtures cover the required reject + allow cases

Cross-checked against spawn-prompt explicit list:

| Required case                        | Test fixture | load_test.go line |
| ------------------------------------ | ------------ | ----------------- |
| `["KEY=value"]` reject (= sign)      | YES          | 530               |
| `[""]` reject (empty)                | YES          | 537               |
| `["FOO", "FOO"]` reject (duplicate)  | YES          | 543               |
| `["FOO BAR"]` reject (whitespace)    | YES          | 549               |
| `["FOO-BAR"]` reject (hyphen)        | YES          | 555               |
| `["FOO.BAR"]` reject (dot)           | YES          | 561               |
| `["1FOO"]` reject (leading digit)    | YES          | 567               |
| `["https_proxy"]` allow (lowercase)  | YES          | 573               |

All 8 required cases present. Bonus coverage: `foo_bar`, `HTTP_PROXY`, `FOO123` (allows). PASS.

### Criterion 7 — `mage ci` green per worklog

**Evidence:**

- Worklog claim: `mage check` (alias of `mage ci`) — 2281 tests / 2280 passed / 1 skipped / 0 failed across 21 packages.
- Templates package coverage 96.3% (worklog) — well above the 70% gate.
- `mage testPkg ./internal/templates` — 268 tests pass, no race regressions.

QA-Proof scope is read-only; live `mage ci` re-run is not in scope. **Trust on worklog claim** — falsification sibling will independently verify if it chooses. Marking PASS-WITH-CAVEAT documented under Unknowns. **Note:** if the falsification sibling re-runs `mage ci` and finds a regression, that flips this verdict; in this round, no fail evidence exists. PASS (worklog-trust).

### Criterion 8 — Scope only `internal/templates/` + worklog touched

**Evidence:**

- `git diff --name-only internal/templates/` — exactly 5 files, all under `internal/templates/`.
- Worklog `workflow/drop_4c/4c_F7_17_1_BUILDER_WORKLOG.md` is untracked-new (`git status` "??") per acceptance.
- Other modified files visible in `git status` (`spawn.go`, `domain/*`, `tui/model_test.go`, `repo_test.go`, `SKETCH.md`) are NOT touched by this droplet's diff range. They are pre-existing in-flight work for F.7.10 + other droplets.

Scope discipline confirmed. PASS.

---

## Proof certificate (final)

**Premises:**
- F.7.17.1 ships only `Env []string` + `CLIKind string` per REV-1.
- The 5 required test scenarios are present.
- Regex literal pinned with anchors + `MatchString` per A1.c.
- Validator wired in correct chain order.
- Scope strictly templates package.
- `mage ci` green per worklog claim.

**Evidence:**
- `schema.go:356, 371` (fields with doc-comments).
- `load.go:120-122` (validator wiring), `load.go:348` (regex literal), `load.go:377-397` (validator body).
- `load_test.go:426, 467, 502, 633, 651` (5 tests).
- `agent_binding_test.go:35-36` (Env + CLIKind populated).
- `schema_test.go:86-87` (round-trip exercises new fields).
- `git diff --name-only` (scope to 5 files).
- Worklog 4c_F7_17_1_BUILDER_WORKLOG.md — `mage ci` claim.

**Trace or cases:** Each of 8 acceptance criteria walked sequentially with file:line citations. REV-1 supersession verified by absence-of-symbol search across the templates package.

**Conclusion:** PROOF GREEN — all 8 criteria backed by evidence; no missing premises; no contradictions in the diff.

**Unknowns:**
- Live `mage ci` re-run not performed (read-only QA-Proof scope). Worklog claim accepted on trust. If the falsification sibling re-runs `mage ci` and finds a failure, the verdict flips.
- The other in-flight working-tree changes (`internal/domain/*`, `internal/tui/model_test.go`, etc.) are explicitly NOT this droplet's scope; not reviewed. Their interaction with F.7.17.1 (none expected; templates package is leaf) is the orchestrator's coordination concern, not this proof.

---

## Hylla Feedback

N/A — this QA-Proof reviewed Go diff via direct `Read` + `git diff` per WORKFLOW Phase 5 norm; no Hylla queries needed for an isolated leaf-package change. The committed-baseline ingest doesn't include the new code yet (it's pre-commit), so Hylla wouldn't help here regardless.
