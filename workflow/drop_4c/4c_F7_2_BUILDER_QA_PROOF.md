# Drop 4c F.7.2 — Tool-gating + Sandbox + System-prompt-template — QA PROOF

**Reviewed commit:** `f6aec8b` — `feat(templates): add tool-gating + sandbox + sysprompt fields (4c F.7.2)`
**Reviewer:** go-qa-proof-agent (opus)
**Round:** 1
**Verdict:** **PROOF GREEN-WITH-NITS** (workflow violation noted; no schema or test gap)

## Premises (what must hold for PASS)

P1. Four declared fields (`ToolsAllowed`, `ToolsDisallowed`, `SystemPromptTemplatePath`, `Sandbox`) added to `AgentBinding` with explicit TOML tags + doc-comments citing memory §5 + §4.
P2. `SandboxRules`, `SandboxFilesystem`, `SandboxNetwork` are CLOSED Go structs (no `map[string]any`), every field tagged.
P3. `validateAgentBindingToolGating` wired into `Load` chain after `validateAgentBindingContext` and before `validateTillsyn`.
P4. All 12 spawn-prompt scenarios are covered by tests.
P5. Strict-decode regression: an unknown sub-struct key (`[agent_bindings.build.sandbox.filesystem] bogus = true`) MUST fail with `ErrUnknownTemplateKey`.
P6. `pathContainsTraversal` rejects `..` segments.
P7. Metachar set covers `;`, `|`, `&`, backtick (`` ` ``), `$`.
P8. Scope: only `internal/templates/` + worklog touched.
P9. `mage ci` green; templates package coverage well above 70%.

## Evidence per premise

### P1 — Four AgentBinding fields with explicit TOML tags + memory cites

`internal/templates/schema.go`:

- L445 `ToolsAllowed []string \`toml:"tools_allowed"\`` — doc lines 431–445 cite "memory §5 / SKETCH §F.7.2 the two-layer tool-gating strategy uses this field as Layer A".
- L458 `ToolsDisallowed []string \`toml:"tools_disallowed"\`` — doc lines 447–458 cite "memory §5 / SKETCH §F.7.2 this is the AUTHORITATIVE tool-gating layer — the probe-grounded finding is that agents route around `--allowed-tools` … via Bash, so only deny patterns inside settings.json catch the workaround".
- L479 `SystemPromptTemplatePath string \`toml:"system_prompt_template_path"\`` — doc lines 460–479 describe the project-relative `.tillsyn/` contract + reject rules.
- L491 `Sandbox SandboxRules \`toml:"sandbox"\`` — doc lines 481–491 cite "memory §4 the sandbox semantics rely on Claude Code's settings.json `permissions.{allow,deny}` for filesystem AND on out-of-process network gating; this field is the schema seam, NOT the enforcement layer".

P1 PROVEN.

### P2 — Closed sub-structs, no map[string]any

`internal/templates/schema.go`:

- L506–512 `SandboxRules` — exactly two fields: `Filesystem SandboxFilesystem`, `Network SandboxNetwork`. Both tagged.
- L518–526 `SandboxFilesystem` — exactly two fields: `AllowWrite []string \`toml:"allow_write"\``, `DenyRead []string \`toml:"deny_read"\``.
- L532–540 `SandboxNetwork` — exactly two fields: `AllowedDomains []string \`toml:"allowed_domains"\``, `DeniedDomains []string \`toml:"denied_domains"\``.

No `map[string]any` anywhere in the new types. Every field has an explicit TOML tag → strict-decode rejection inherits automatically (re-confirmed in P5).

P2 PROVEN.

### P3 — Validator wired into Load chain in correct order

`internal/templates/load.go` lines 137–148:

```
137:	if err := validateAgentBindingEnvNames(tpl); err != nil { … }
140:	if err := validateAgentBindingContext(tpl); err != nil { … }
143:	if err := validateAgentBindingToolGating(tpl); err != nil { … }
146:	if err := validateTillsyn(tpl); err != nil { … }
```

Order is `…Env → …Context → …ToolGating → …Tillsyn`. Spawn-prompt requirement was "after validateAgentBindingContext, before validateTillsyn" — satisfied exactly.

Doc-comment in `Load` was updated (lines 60–66 reference step `4.h` for the new validator; step `4.i` for `validateTillsyn`).

P3 PROVEN.

### P4 — All 12 spawn-prompt scenarios covered

Cross-referencing the spawn-prompt scenario list against `internal/templates/load_test.go`:

| # | Scenario | Test (file:line) |
|---|---|---|
| 1 | Happy (every field populated) | `TestLoadAgentBindingToolGatingHappyPath` (L821) |
| 2 | Empty / omitted (back-compat) | `TestLoadAgentBindingToolGatingOmittedFields` (L876) |
| 3 | Reject empty entry (tools_allowed) | rejection table L928 — `tools_allowed = [""]` |
| 3b | Reject empty entry (tools_disallowed) | L938 |
| 3c | Reject empty entry (allow_write) | L989 |
| 3d | Reject empty entry (allowed_domains) | L1024 |
| 4 | Reject duplicate (tools_allowed) | L933 — `["Read", "Read"]` |
| 4b | Reject duplicate (tools_disallowed) | L943 — `["WebFetch", "WebFetch"]` |
| 5 | Reject metachar in path | five sub-rows L948–971: `;`, `\|`, `&`, `` ` ``, `$` |
| 6 | Reject `..` traversal in path | L972 — `"../etc/passwd"` |
| 7 | Reject absolute path | L977 — `"/etc/passwd"` |
| 8 | Reject relative sandbox path | L983 (allow_write), L1006 (deny_read) |
| 9 | Reject `..` in sandbox path | L994 — `"/abs/../etc"` (also `//` rejection L1000) |
| 10 | Reject URL-scheme domain | L1012, L1018 (`https://`, `http://`), L1030 (denied_domains) |
| 11 | Allow glob domain | `TestLoadAgentBindingToolGatingAllowsGlobDomain` (L1069) — `*.npmjs.org`, `*.pypi.org` |
| 12 | Strict-decode unknown key | `TestLoadAgentBindingToolGatingStrictDecodeUnknownFieldRejected` (L1097) + sub-struct variant L1124 |

All 12 scenarios PROVEN. The rejection table reports 19 sub-cases (the `strings.ReplaceAll` for `Test/.../...` count from `mage test-pkg` showed 313 tests pass in the templates package).

P4 PROVEN.

### P5 — Strict-decode regression on sub-struct keys

`internal/templates/load_test.go` lines 1124–1190 — `TestLoadAgentBindingToolGatingStrictDecodeUnknownSandboxFieldRejected` is a 3-row table covering:

- L1131–1144 `[agent_bindings.build.sandbox.filesystem] bogus_filesystem_key = true` → `ErrUnknownTemplateKey`.
- L1146–1158 `[agent_bindings.build.sandbox.network] bogus_network_key = true` → `ErrUnknownTemplateKey`.
- L1161–1173 `[agent_bindings.build.sandbox] bogus_sandbox_key = true` → `ErrUnknownTemplateKey`.

This exactly satisfies the spawn-prompt premise (`[agent_bindings.build.sandbox.filesystem] bogus = true MUST fail`) plus the symmetric variants on `[sandbox]` and `[sandbox.network]`. Strict decode inheritance is a property of `pelletier/go-toml/v2`'s `DisallowUnknownFields` recursing into nested struct types — confirmed by these positive tests.

P5 PROVEN.

### P6 — pathContainsTraversal rejects `..`

`internal/templates/load.go` lines 741–748:

```go
func pathContainsTraversal(path string) bool {
    for _, segment := range strings.Split(path, "/") {
        if segment == ".." {
            return true
        }
    }
    return false
}
```

Splits on `/`, exact-match on `..` segment. The split-not-substring choice is correct: `foo..bar` (legitimate filename with two dots) does NOT trigger; `../etc` and `/abs/../etc` DO trigger.

Tests prove both:

- L972 `system_prompt_template_path = "../etc/passwd"` → rejected (PROOF).
- L994 `allow_write = ["/abs/../etc"]` → rejected (PROOF).

The non-traversal `foo..bar` case is implicitly handled — no test exercises it, but the implementation is split-on-`/` so the property is statically true. The function's Go doc-comment (L738) explicitly calls out this property.

P6 PROVEN.

### P7 — Metachar set is `;`, `|`, `&`, backtick, `$`

`internal/templates/load.go` line 587:

```go
var systemPromptShellMetacharRunes = []rune{';', '|', '&', '`', '$'}
```

Exactly five runes, exact match against the spawn-prompt requirement. Each is exercised by a dedicated rejection-table sub-test (L948–971). Doc-comment L580–587 calls out "deliberately conservative — defense-in-depth".

P7 PROVEN.

### P8 — Scope: only `internal/templates/` + worklog

`git diff --stat HEAD~1 HEAD`:

```
 internal/templates/agent_binding_test.go    |  24 +-
 internal/templates/load.go                  | 204 +++++++++++++-
 internal/templates/load_test.go             | 394 ++++++++++++++++++++++++++++
 internal/templates/schema.go                | 110 ++++++++
 internal/templates/schema_test.go           |  16 ++
 workflow/drop_4c/4c_F7_2_BUILDER_WORKLOG.md | 109 ++++++++
```

Six files. Five inside `internal/templates/`, one inside `workflow/drop_4c/`. No leakage into other packages, no docs or config touched. Scope contract honored.

P8 PROVEN.

### P9 — mage ci green; templates 96.8 % coverage

QA reviewer ran `mage test-pkg ./internal/templates`: **313 tests passed, 0 failed, 0 skipped**.
QA reviewer ran `mage ci`: completed end-to-end including coverage gate. `internal/templates` coverage = **96.8 %** (well above the 70 % minimum), build succeeded ("Built till from ./cmd/till").

P9 PROVEN.

## Trace / cases (load-time validator path)

Trace 1 (happy): `Load` → strict decode → `validateMapKeys` → `…ChildRules*` → `validateGateKinds` → `validateAgentBindingEnvNames` → `validateAgentBindingContext` → **`validateAgentBindingToolGating`** (each helper inspects its slice; all clean) → `validateTillsyn` → return tpl. Demonstrated by `TestLoadAgentBindingToolGatingHappyPath`.

Trace 2 (omitted): same chain, the new fields are nil/empty, every helper iterates zero entries, returns nil. Demonstrated by `TestLoadAgentBindingToolGatingOmittedFields`.

Trace 3 (reject empty): inside `validateToolNameList` (load.go L648–662), `entry == ""` short-circuits → wraps `ErrInvalidAgentBindingToolGating`. Both umbrella sentinels (`ErrInvalidAgentBindingToolGating` and `ErrInvalidAgentBinding`) match per `errors.Is` (asserted L1051, L1054).

Trace 4 (sub-struct strict decode): `pelletier/go-toml/v2`'s `DisallowUnknownFields` walks recursively at decode time (load.go step 3, line 111). Unknown key under `[agent_bindings.build.sandbox.*]` surfaces a `*toml.StrictMissingError` → wrapped to `ErrUnknownTemplateKey` at L114.

Trace 5 (metachar): `validateSystemPromptTemplatePath` (load.go L669–688) — checks empty, then `/`-prefix, then `pathContainsTraversal`, then iterates `systemPromptShellMetacharRunes` calling `strings.ContainsRune`. Each metachar fails fast and returns a descriptive error including the offending rune.

All five traces exercised by named tests.

## Conclusion

**PROOF GREEN-WITH-NITS.**

All 9 premises are evidence-backed. The code change is well-scoped, the validator chain wiring is exactly as specified, the closed-struct contract is correctly enforced, and the test surface exhausts the 12 spawn-prompt scenarios plus seven additional implicit ones (per-metachar coverage, deny_read symmetry, deny_domains symmetry, sub-struct unknown-key variants).

The "WITH-NITS" qualifier is a process finding only — not a code or schema gap.

## Findings

### F1 (process / workflow violation) — Builder self-committed

`git log` shows commit `f6aec8b` was authored by `gitdiff-test <gitdiff-test@example.com>` and pushed without orch involvement. Per Drop 4c discipline (and CLAUDE.md "Build-QA-Commit Discipline"), builders MUST NOT commit — they hand the changeset back to the orchestrator, which runs QA, and orch drives the commit only after QA green. The committed-state-at-review-time pattern means QA had no opportunity to gate the commit; if findings had emerged here, the orch would need to revert + replay.

**Severity:** process — does not block the schema correctness. Recommend orch flag this in droplet closeout for the dev's awareness and reinforce in the next builder spawn prompt (the worklog Round 1 narrative does not mention the commit step, suggesting the builder was unaware of the discipline boundary).

**Routing:** orch follow-up — no code fix required for this droplet.

### F2 (NIT, optional) — pathContainsTraversal lacks a "non-traversal substring" unit test

The doc-comment for `pathContainsTraversal` (load.go L738–740) promises that `foo..bar` (a legitimate filename containing two literal dots) will NOT trigger the traversal check. The implementation is statically correct (split on `/` then exact-match `..` segment), but no positive-allow test pins this. A one-line subtest like `{name: "allow filename with embedded dots", path: "foo..bar/file.md", wantValid: true}` would harden the future-refactor surface.

**Severity:** NIT — not a gap; the implementation is provably correct and the negative tests already exercise the segment-split branch.

**Routing:** optional follow-up; defer to a future hardening pass or include in F.7.3b's render-time path tests.

## Missing Evidence

None. Every premise has a direct file:line citation; the validator chain ordering was verified by reading load.go's step 4 sequence; the `mage ci` gate was independently re-run by the reviewer.

## Hylla Feedback

`None — Hylla answered everything needed.` The review touched only Go source in `internal/templates/` plus a worklog MD. All Go symbol resolution went through `Read` directly (the file is small enough to read end-to-end); no Hylla query was needed and no fallback was triggered.

