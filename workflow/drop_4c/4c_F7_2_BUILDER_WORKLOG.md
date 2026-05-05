# Drop 4c F.7.2 — TOML schema widening (tool-gating + sandbox + system-prompt) — Builder Worklog

**Droplet:** F.7-CORE F.7.2
**Builder model:** opus
**Date:** 2026-05-04
**Plan source:** `workflow/drop_4c/F7_CORE_PLAN.md` § F.7.2 + REVISIONS POST-AUTHORING REV-7 + spawn prompt

## Round 1

### Goal

Add tool-gating + sandbox + system-prompt-template fields to `templates.AgentBinding`. Pure schema additions + load-time validators; NO behavior. Settings.json render path is F.7.3b's territory.

### Files edited

- `internal/templates/schema.go`
  - Added 4 fields on `AgentBinding`:
    - `ToolsAllowed []string` (TOML `tools_allowed`).
    - `ToolsDisallowed []string` (TOML `tools_disallowed`).
    - `SystemPromptTemplatePath string` (TOML `system_prompt_template_path`).
    - `Sandbox SandboxRules` (TOML `sandbox`).
  - Added 3 closed sub-structs with explicit TOML tags:
    - `SandboxRules` — wraps `Filesystem` + `Network`.
    - `SandboxFilesystem` — `AllowWrite []string`, `DenyRead []string`.
    - `SandboxNetwork` — `AllowedDomains []string`, `DeniedDomains []string`.
  - Doc-comments cite memory §5 (tool-gating two-layer strategy), memory §4 (sandbox semantics), and Drop 4c F.7.2 plan.

- `internal/templates/load.go`
  - Added `ErrInvalidAgentBindingToolGating` sentinel (wraps `ErrInvalidAgentBinding`).
  - Added `validateAgentBindingToolGating(tpl Template) error` — wired into `Load`'s validator chain after `validateAgentBindingContext` and before `validateTillsyn` per acceptance contract.
  - Added 5 helper functions:
    - `validateToolNameList` — non-empty + within-list-unique on slice.
    - `validateSystemPromptTemplatePath` — empty allowed; rejects absolute, `..` traversal, shell metachars `;` `|` `&` backtick `$`.
    - `validateSandboxAbsolutePathList` — non-empty + clean absolute (`/`-prefix, no `..`, no `//`).
    - `validateSandboxDomainList` — non-empty + no `://` URL scheme. Glob `*` allowed.
    - `pathContainsTraversal` — splits on `/`, returns true on any `..` segment.
  - Added `systemPromptShellMetacharRunes` package var — pinned closed metachar set.
  - Updated `Load` doc comment to include the new step `4.h` for the validator and renumbered the prior `validateTillsyn` step to `4.i`.

- `internal/templates/agent_binding_test.go`
  - Extended `fullyPopulatedAgentBinding()` to populate every new field with non-zero, validator-passing values. The round-trip test inherits the new coverage automatically.

- `internal/templates/schema_test.go`
  - Extended `TestTemplateTOMLRoundTrip`'s populated `AgentBinding` with the 4 new fields so the existing round-trip exercises them symmetrically.

- `internal/templates/load_test.go`
  - Added `TestLoadAgentBindingToolGatingHappyPath` — full set of fields populated; verifies decode + validator clean.
  - Added `TestLoadAgentBindingToolGatingOmittedFields` — back-compat: no new fields → zero-value struct.
  - Added `TestLoadAgentBindingToolGatingRejectionTable` — 19 sub-cases covering every rejection path:
    - Empty entry in `tools_allowed`, `tools_disallowed`, `allow_write`, `allowed_domains`.
    - Duplicate entry in `tools_allowed`, `tools_disallowed`.
    - Each shell metachar (`;`, `|`, `&`, backtick, `$`) in `system_prompt_template_path`.
    - `..` traversal + absolute path in `system_prompt_template_path`.
    - Relative + traversal + double-slash in sandbox `allow_write`.
    - Relative `deny_read`.
    - URL-scheme `http://` / `https://` in `allowed_domains` + `denied_domains`.
  - Added `TestLoadAgentBindingToolGatingAllowsGlobDomain` — `*.npmjs.org`, `*.pypi.org` MUST PASS.
  - Added `TestLoadAgentBindingToolGatingStrictDecodeUnknownFieldRejected` — closed-struct contract: `bogus_tool_field` on `[agent_bindings.build]` fails with `ErrUnknownTemplateKey`.
  - Added `TestLoadAgentBindingToolGatingStrictDecodeUnknownSandboxFieldRejected` — closed-struct contract on the new sub-structs: unknown keys on `[sandbox]`, `[sandbox.filesystem]`, `[sandbox.network]` all fail.
  - Added local `equalStringSlices` helper to keep nil-vs-empty asymmetries explicit at assertion time.

### Verification

- `mage check` — 22 packages, 2380 tests, 1 skipped, 0 failed. Coverage on `internal/templates` = 96.8%.
- `mage ci` — green end-to-end.
- `mage test-pkg ./internal/templates` — 313 tests pass.
- Targeted reruns confirmed:
  - `TestLoadAgentBindingToolGatingHappyPath` — 1 pass.
  - `TestLoadAgentBindingToolGatingRejectionTable` — 21 reported (19 sub-cases + parent + 1 framework counter).
  - `TestLoadAgentBindingToolGatingStrictDecodeUnknownFieldRejected` — 1 pass.

### Acceptance criteria — checklist

- [x] `ToolsAllowed`, `ToolsDisallowed`, `SystemPromptTemplatePath`, `Sandbox` fields added to `AgentBinding` with explicit TOML tags.
- [x] `SandboxRules` + `SandboxFilesystem` + `SandboxNetwork` closed sub-structs with explicit TOML tags.
- [x] `validateAgentBindingToolGating` wired into `templates.Load` validator chain after `validateAgentBindingContext`.
- [x] All 12 declared test scenarios + 7 additional implicit scenarios (one per shell metachar, plus `denied_domains` + `deny_read` symmetry) pass.
- [x] Strict-decode unknown-key test asserts `bogus_tool_field` fails with `ErrUnknownTemplateKey`.
- [x] `mage check` + `mage ci` green.
- [x] Worklog written.

### Acceptance scenarios mapped to tests

| Spec scenario | Test that proves it |
|---|---|
| Happy path: every field populated loads cleanly | `TestLoadAgentBindingToolGatingHappyPath` |
| Empty fields: all omitted = zero-value struct | `TestLoadAgentBindingToolGatingOmittedFields` |
| Reject empty entry in tools_allowed | `TestLoadAgentBindingToolGatingRejectionTable/reject_empty_entry_in_tools_allowed` |
| Reject duplicate tool | `TestLoadAgentBindingToolGatingRejectionTable/reject_duplicate_entry_in_tools_allowed` |
| Reject shell-metachars in path | `TestLoadAgentBindingToolGatingRejectionTable/reject_shell-metachar_*_in_system_prompt_template_path` (5 metachars) |
| Reject traversal in path | `TestLoadAgentBindingToolGatingRejectionTable/reject_traversal_in_system_prompt_template_path` |
| Reject absolute path | `TestLoadAgentBindingToolGatingRejectionTable/reject_absolute_system_prompt_template_path` |
| Reject relative sandbox path | `TestLoadAgentBindingToolGatingRejectionTable/reject_relative_sandbox_allow_write` (+ deny_read) |
| Reject `..` in sandbox path | `TestLoadAgentBindingToolGatingRejectionTable/reject_traversal_in_sandbox_allow_write` |
| Reject URL-scheme domain | `TestLoadAgentBindingToolGatingRejectionTable/reject_URL-scheme_*` (3 cases) |
| Allow glob domain | `TestLoadAgentBindingToolGatingAllowsGlobDomain` |
| Strict-decode unknown key | `TestLoadAgentBindingToolGatingStrictDecodeUnknownFieldRejected` (+ sandbox sub-struct variant) |

### Decisions / non-spec choices

- **Closed metachar set is package-level var, not const slice.** `var systemPromptShellMetacharRunes = []rune{';', '|', '&', '`', '$'}` at file scope. Allows `range` iteration and a docstring; matches Go convention for closed config sets.
- **`pathContainsTraversal` splits on `/` rather than substring-matching `..`.** Reason: `foo..bar` is a legitimate filename containing two literal dots; only a true `..` segment is a traversal. Tests confirm rejection of `../etc/passwd` and `/abs/../etc` while non-traversal substrings pass.
- **`SandboxNetwork` URL-scheme guard uses `://` substring rather than enumerating `http://` / `https://`.** Future `ftp://` / `git://` / etc. are also not what a domain-allow list should accept; one substring catches them all without enumeration churn.
- **Domain-glob regex NOT validated.** Spec allows `*.npmjs.org`. Tightening to `^\*\.[a-z0-9.-]+$` would over-fit and reject legitimate adopter patterns. The downstream sandbox renderer (F.7.3b) is responsible for any pattern semantics.
- **Strict-decode regression coverage extended beyond spec.** Added per-sub-struct unknown-key tests for `[sandbox]`, `[sandbox.filesystem]`, `[sandbox.network]` because the closed-struct guarantee comes from the strict-decoder applying recursively — explicit tests document that property at the new struct boundaries.

### Hylla Feedback

`N/A — task touched non-Go file (worklog MD) plus Go source in a single small package; all symbol resolution went through Read directly. No Hylla queries issued.`
