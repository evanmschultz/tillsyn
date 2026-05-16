# Tillsyn Refinements

Append-only log of Tillsyn product refinements, refactor candidates, and TUI / CLI / MCP ergonomics issues discovered during day-to-day use. Paired with the perpetual Tillsyn tracking drop `REFINEMENTS.MD` — entries added here also get mirrored as comments on that drop (dual-update rule from `CLAUDE.md`).

Hylla-specific refinements live in `HYLLA_REFINEMENTS.md`.

## Entry Schema

Each entry uses this shape. Newest-first ordering.

```markdown
## <YYYY-MM-DD> — <Drop N> — <One-line title>

### Context
What the session was doing when the friction surfaced. One or two sentences.

### Observation
The concrete issue. Include tool name + exact input and actual output if applicable. Include enough detail that a future drop can reproduce without re-deriving context.

### Proposed fix
Concrete action to take. Scope matters — distinguish "inline safe fix" from "cross-cutting refactor drop." Cite a proposed target drop when known.

### Target drop
Where this refinement should land. E.g. `Drop 1`, `pre-Drop-3 template-overhaul`, `post-Drop-4 TUI polish`, or `parking-lot` if unassigned.

### Tags
Comma-separated. Examples: `tui`, `cli`, `mcp`, `refactor`, `docs`, `coordination`, `auth`, `ergonomics`, `performance`.
```

## Status Lifecycle

- **Pending** — entry logged, not yet triaged. Default state.
- **Scheduled** — triaged into a concrete drop.
- **In Progress** — currently being worked in a drop.
- **Shipped** — fix landed. Entry summarized into the drop's closeout `WIKI_CHANGELOG.md` line; original entry either stays as-is or gets trimmed during MD cleanup subdrop.
- **Rejected** — not doing. Kept for audit-trail continuity with the reason.

Transitions are recorded by appending a dated status note to the entry, not by rewriting history.

---

## 2026-05-16 — agent-isolation-followup — `wait_timeout` default-on for live-wait MCP ops

### Context
Tonight's post-compaction recovery flow called `till.auth_request operation=create` without `wait_timeout`. The call returned immediately with `state: pending`. Dev approved in TUI + ran `mage install`, then asked "why didn't you get the notification?" Root cause: the orch never subscribed to the live-wait stream because the parameter was omitted.

### Observation
`wait_timeout` is optional on `till.auth_request operation=create`. When omitted, the call short-circuits — no live-wait subscription — and dev approvals don't propagate back to the caller. This is brittle: even the rule's author missed it the first time, because the rule didn't exist when the gap fired.

Other live-wait surfaces likely share the same gap: `till.comment operation=list` (`wait_timeout` for thread updates), `till.attention_item operation=list` (if it has a wait surface). The pattern is "live-wait optional → callers forget → notifications silently lost."

### Proposed fix
Default-on at the MCP server layer: when `wait_timeout` is omitted on a live-wait-capable op, the server fills it with `timeout` (so live-wait matches the approval window). Callers wanting fire-and-forget pass `wait_timeout: "0s"` explicitly.

Document the new default + opt-out path in `README.md` + `CLAUDE.md`. Sweep all live-wait-capable ops (`till.auth_request create`, `till.comment list`, attention-item waits) uniformly.

Dev disposition 2026-05-16: option (c) approved — server-side default + docs as belt-and-suspenders.

### Target drop
Next ergonomics drop — short single-droplet fix in the MCP server layer. High priority — direct dev-visible friction.

### Tags
`mcp`, `auth`, `live-wait`, `ergonomics`, `dev-approved`

---

## 2026-05-16 — W1 D4 follow-up — MCP error-class drift on adapter-returned `invalid_request:` errors

### Context
W1 D4 (`b3e0840`) added an adapter-layer guard in `internal/adapters/mcp_common/app_service_adapter_mcp.go:634-636` returning `fmt.Errorf("invalid_request: repo_primary_worktree is required ...")`. W1 D4 plan-QA-falsification (`aba648c94473e5396`, VERDICT FAIL counterexample A2) caught a real semantic drift: the adapter's error is not a registered MCP sentinel, so `mapToolError` (`internal/adapters/mcp_rpc/handler.go:914`) falls through to `default:` and stamps it as `internal_error: invalid_request: ...` with `error_class: internal, error_code: internal_error`. MCP clients see this categorized as a 5xx-equivalent rather than a 4xx-class validation failure.

### Observation
The "loud invalid_request" intent of `feedback_parity_clarity_no_silent_failures` is partially undermined: the error message contains `invalid_request:` BUT the MCP error-class field says `internal`. Adopters' retry logic / error routing will treat this as transient/server-side rather than "fix your request and don't retry."

Attempted in-line fix (move guard to `internal/adapters/mcp_rpc/extended_tools.go` next to `args.Name == ""`) BROKE 11 tests because:
- Placement before auth → tests expecting auth errors get validation error first.
- `TestProjectMCPFirstClassFieldsRoundTrip/create_without_first-class_fields_round-trips_empty_values` explicitly tests the OLD permissive behavior.
- Coverage dropped in `mcp_rpc` package.

Revert was clean; mage ci green restored.

### Proposed fix (refinement)
Two-step fix in a separate cascade:

1. **Define `mcpcommon.ErrInvalidRequest` sentinel**, wrap the adapter error with `%w`, add an `errors.Is` case in `mapToolError` to route to `Class: "invalid_request" / Code: "invalid_request"`. This preserves adapter-layer behavior + fixes error class for MCP transport.
2. **Update `TestProjectMCPFirstClassFieldsRoundTrip/create_without_first-class_fields_round-trips_empty_values`** to assert the new required-field behavior (or delete the sub-case if "empty round-trip" is no longer a valid contract).
3. **Tighten `TestCreateProjectRepoPrimaryWorktreeRequired`** to assert MCP error CLASS, not just substring (post-sentinel-route).

### Target drop
Post-W1 single-droplet refinement — small, contained, but touches handler error-mapping which warrants its own plan-QA pair.

### Tags
`mcp`, `error-class`, `parity-clarity`, `refinement`, `w1-d4-followup`

---

## 2026-05-16 — plan-QA methodology — three refinements raised by 2.C falsification

### Context
The 2.C Drop 4b plan-QA-falsification pass surfaced load-bearing counterexamples (C1/C2/C3/C5) that the original planner missed. While analyzing those, the falsification agent raised three standing methodology refinements that apply across cascades, not just to this one.

### Observation

**(R1) Shipped-but-not-wired pattern is recurrent.** Drop 3 droplet 3.20 anti-pattern (template-resolver shipped without consumer) and Drop 4b (gate runner + 4 gate impls shipped, zero `Register` / zero call-sites) have the same shape: a runner / registry / adapter type ships without any production code consuming it. Plan-QA-falsification does not currently have a standing attack family for this. Without one, the pattern keeps slipping through closeouts.

**(R2) Planner spec must land on disk before plan-QA dispatch.** Both 2.C and 2.B.1 planner outputs lived ONLY in Tillsyn descriptions during this session. Plan-QA-falsification flagged "I cannot directly read the Tillsyn descriptions" as degrading the review — the agent had to reconstruct the spec from the orchestrator's spawn-prompt summary. Confidence in the FAIL verdict was high anyway because the counterexamples were objective state-machine facts, but the absence of an on-disk spec made the review harder than necessary.

**(R3) Drop closeouts must include "every shipped runner has a consumer call site" checklist item.** Drop 4b shipped with the gate runner + 4 gate impls but no consumer wiring; closeout did not catch this. Generalize: any drop landing a Register-style or Run-style infrastructure piece must verify, before close, that at least one production call site invokes the consumer side.

### Proposed fix

- **R1** — Add a standing plan-QA-falsification attack family: "for every shipped runner / registry / adapter type, verify a `Register` / consumer call site exists in production code." Encode this in `go-qa-falsification-agent.md` (or equivalent template) as a default attack.
- **R2** — Pre-cascade workflow should formalize "planner writes spec to disk at `workflow/<drop>/PLAN.md` (or equivalent) BEFORE plan-QA dispatch." Tillsyn descriptions can supplement but the on-disk artifact is the source of truth plan-QA agents read.
- **R3** — Add to drop-closeout checklist (whether MD-encoded or template-enforced): "every shipped runner / registry / adapter type has at least one production consumer call site." Cascade closeout-agent picks this up automatically.

### Target drop

Methodology drop — not a Go-code drop. Lives in agent templates + WORKFLOW.md + closeout checklist. Pre-cascade-dispatcher era: orchestrator threads the discipline into spawn prompts manually until templates encode it.

### Tags
`methodology`, `plan-qa`, `falsification`, `closeout`, `workflow`, `cascade-methodology`

---

## 2026-05-16 — agent-isolation-followup — Hook `..`-traversal hardening: out-of-scope attack surfaces raised by W2 plan-QA-falsification

### Context
W2 (hook `..`-traversal hardening) plan-QA-falsification raised 3 out-of-scope attack surfaces while reviewing the `..`-traversal fix. These are NOT addressed in W2's atomic build droplet `00adb52e` — they require their own hardening passes.

### Observation

**(a) `%2e%2e` URL-encoded bypass.** `jq -r` decodes JSON `\u` escapes but does NOT URL-decode `%2e%2e`. If any upstream layer (Claude Code's tool argument parsing) URL-decodes paths before passing to the hook, `%2e%2e/etc/passwd` could land as literal `../etc/passwd` and be caught — but if NO upstream layer decodes, then `%2e%2e/etc/passwd` reaches the hook as the literal 19-char string with no `/` before `%2e%2e` → pattern doesn't match → allowed. Real-world risk depends on call-site behavior — likely low but worth verifying.

**(b) Symlink traversal.** A legitimate in-scope path `only/this/dir/link` could be a symlink to `/etc/passwd`. Hook only sees the path string, not the resolved target. Needs either pre-validate symlink resolution (race-y with TOCTOU) or filesystem-level chroot/sandbox.

**(c) `bash -c` / `sh -c` wrapping** — already documented in `TestHookIntegration_KnownBypassCases` as accepted bypass. Wrapping `bash -c 'rm ../etc/passwd'` hides the dangerous primitive from the tokenizer. Same out-of-scope class.

### Proposed fix

(a) URL-encoded `..`: add a single test case to confirm whether `%2e%2e` reaches the hook as literal. If yes, extend `reject_if_dotdot` to decode common URL-encoding patterns OR document as accepted-bypass.

(b) Symlink traversal: add a `realpath`-based resolution step AFTER `normalize_path` to canonicalize symlinks before scope check. Cross-platform consideration: macOS BSD `realpath` differs from GNU coreutils.

(c) `bash -c` wrapping: separate hardening pass; integrate with Claude Code's tool argument inspection layer rather than the hook script.

### Target drop

Post-W2. (a) and (b) are small enough to absorb into a single follow-up. (c) requires upstream coordination with Claude Code SDK.

### Tags
`hook`, `traversal`, `bypass`, `security`, `parking-lot`

---

## 2026-05-16 — agent-isolation-followup — `move_state` rejects metadata in same call for `failed` transitions

### Context
Tonight's stale-orphan cleanup called `till.action_item operation=move_state state=failed` with `metadata: {"outcome": "blocked", "blocked_reason": "..."}` in the same call. Returned `internal_error: invalid metadata outcome for failed transition: metadata.outcome must be one of {failure, blocked, superseded} on transition to failed (got "")`. The metadata sent in the call payload was not seen by the validation check.

Workaround: split into two calls — `operation=update` with `metadata` first, then `operation=move_state` with no metadata. Worked correctly. But the two-call pattern is non-obvious and the error message implies metadata SHOULD be reachable from `move_state`.

### Observation
Either:
- (a) `move_state` should accept `metadata` and merge it pre-validation (transparent fix), OR
- (b) the schema should explicitly state "metadata must be set via prior `operation=update`; `move_state` does not accept metadata."

Today the schema is ambiguous: `metadata` is documented as "Optional action-item metadata object" without specifying which operations honor it. The error message references `metadata.outcome` but the move_state validation reads from stored row state, not the call payload.

### Proposed fix
Option (a) preferred — merge metadata in `move_state` pre-validation. Saves a round trip.

Option (b) acceptable if (a) is hard — update schema docs to clarify the pattern + add a hint to the error message ("call operation=update with metadata first; move_state does not accept metadata").

### Target drop
Next ergonomics drop — `internal/app/action_item.go` move_state logic. Small fix.

### Tags
`mcp`, `action-item`, `state-transition`, `ergonomics`

---

## 2026-05-16 — agent-isolation — cascade rollup of 12 QA-raised refinements (parent `fab45453`)

### Context
The agent-isolation cascade (`fab45453-ec63-4188-a301-3320b20aa976`, "AGENT ISOLATION PROJECT LOCAL HOOK PATH ENFORCEMENT") shipped 6 build droplets (D-A through D-F, commits `7f7dfe2` / `b08bc06` / `f91987b` / `fb0b3ec` / `009a273` / `94d4abf`). Across the 12 build-QA verdicts (6 proof + 6 falsification), QA raised refinements that were tracked in QA completion_notes but not propagated into REFINEMENTS.md at close time. This entry is the rollup — one umbrella covering all 12 so each is independently triagable in future drops.

### Observation

**(R1) D-A normalize_path `..`-traversal rejection** — hook script's `normalize_path` lacked explicit `..` rejection. SHIPPED 2026-05-16 by W2 D1 commit `b9925e3` (`fix(templates): reject .. components in hook path validation`). Logged here for audit-trail continuity. **Status: Shipped.**

**(R2) D-A tab-tokenization NIT** — hook script's Bash command tokenizer splits on `IFS` default (space + tab + newline) but explicit tab handling in test cases is incomplete. Defense-in-depth class; not a real bypass on POSIX commands but worth covering when the tokenizer is hardened next. **Status: Pending.**

**(R3) D-D dispatcher-staging doc clarification in builder-agent body** — the 3 builder-agent.md files (`go`, `fe`, `gen` groups) declare the PreToolUse hook in YAML frontmatter but the body doesn't describe how the dispatcher stages the `TILLSYN_ACTION_ITEM_ID` + `TILLSYN_BIN` env vars at spawn time. Adopters reading the agent file see the hook but not the contract that makes it work. **Status: Pending. Target: Drop 4c.8 W4 embedded-prompt content.**

**(R4) D-D tools allowlist on builder agents** — the 3 builder-agent.md files do NOT declare a `tools:` allowlist in frontmatter. Adopters running these agents inherit the orchestrator's full tool surface. Per Claude Code SDK semantics, declaring `tools: Read, Edit, Write, Bash` enforces the allowlist strictly when the agent is spawned background. **Status: Pending. Target: Drop 4c.8 W4 embedded-prompt content.**

**(R5) D-B user-set hash sentinel collision in `.claude/settings.json`** — `writeClaudeSettings` uses `tillsyn_hook_hash` as the top-level field name to version-pin. If a downstream user adds their own custom field of the same name, the preserve-and-merge logic could clobber or misread it. Low real-world probability (the key is Tillsyn-namespaced enough) but worth either renaming to `__tillsyn_hook_hash__` or documenting collision behavior. **Status: Pending.**

**(R6) D-B `fsatomic` stale-temp on SIGKILL** — `fsatomic.WriteFile` writes to a temp file then renames. If the process gets SIGKILL'd between temp write and rename, the temp file is orphaned in `.claude/hooks/` (or wherever). Cosmetic housekeeping NIT — a fresh `till init` doesn't pick the temp up, and the file is in a hidden dir, but it's still cruft. **Status: Pending. Class: housekeeping NIT.**

**(R7) D-E substring match on script name vs exact path** — dispatcher's `CheckHookArtifacts` uses substring match (`strings.Contains(command, "validate-action-item-paths.sh")`) to find the hook reference in the agent's YAML frontmatter. A future agent that mentions the script name in a doc-string but not as a real `hooks.PreToolUse.command` entry would trigger a false-positive preflight requirement. Dormant pre-team-feature; surfaces when adopters customize agent frontmatter. **Status: Pending.**

**(R8) D-E yaml.v3 strict mode for duplicate-key override** — `yaml.v3` Unmarshal silently accepts duplicate keys at the same level, with the last value winning. A malicious or buggy agent.md with `hooks:` declared twice would have one block silently dropped. Strict mode (`yaml.Decoder.KnownFields(true)` + manual duplicate-detection) would loud-fail. Dormant pre-team-feature; surfaces when adopters share agent files. **Status: Pending.**

**(R9) D-E multi-tier agent lookup beyond project tier** — dispatcher's preflight resolves the agent file by walking ONLY the project tier (`<worktree>/.claude/agents/`). The 3-tier walk pattern (project → user `~/.claude/agents/` → embedded `internal/templates/builtin/agents/`) exists elsewhere in the dispatcher but the hook preflight doesn't use it. Latent until a builtin agent declares the hook — at that point the preflight would never find the agent file in the project tier and would skip (silent-disable surface). **Status: Pending. Class: latent-bug.**

**(R10) D-E 10-line hash-header scan window fragility** — `CheckHookArtifacts` scans the first 10 lines of the hook script for the `# tillsyn-hook-hash:` header. If the template grows a longer shebang preamble or license header, the hash line could push past line 10 and the preflight silently classifies the script as "stale." Widening the window (to ~30 lines or reading until first non-comment line) would be more robust. **Status: Pending.**

**(R11) D-C update-path seam for project mutation** — `ServiceConfig.BootstrapProjectHooks` fires on CREATE only. When a project's `repo_primary_worktree` is later changed via `till.project operation=update`, the hook bootstrap does NOT re-run for the new worktree. The `update` path needs its own seam (or the create-seam needs to be invoked from update on field change). Real-world relevance: any adopter who creates a project with empty `repo_primary_worktree` then sets it later via update will not get auto-bootstrap. **Status: Pending. Related: W1-D4-R2 from W1 falsification (partial-update merge semantic when `repo_primary_worktree` omitted).**

**(R12) D-F newline-prefix bypass in Bash command tokenizer** — same defense-in-depth class as the accepted `eval` / `bash -c` / `sh -c` bypasses. `command="\ngit checkout -- another/dir/bar.go"` slips past the front-of-command keyword match because the tokenizer doesn't strip leading whitespace. Future tokenizer hardening territory. **Status: Pending. Class: defense-in-depth NIT.**

### Proposed fix

Each refinement is independently triagable:
- **R1**: already shipped. No action.
- **R2, R6, R12**: housekeeping / defense-in-depth NIT class. Batch into a future "hook hardening" droplet when there's a real bypass observed in dogfood.
- **R3, R4**: Drop 4c.8 W4 embedded-prompt content. Frontmatter + body edits to all 3 builder-agent.md files.
- **R5**: tiny one-line rename + test. Anytime.
- **R7, R10**: small dispatcher tightening. Single droplet, ~50 LOC + tests.
- **R8**: yaml.v3 strict mode. Cross-cutting (affects any yaml.Unmarshal call); separate sweep.
- **R9**: extend `CheckHookArtifacts` to use the 3-tier walk. Single droplet when a builtin agent declares the hook (currently nothing forces this).
- **R11**: update-path seam. Single droplet under `internal/app/service.go` (`UpdateProject` or wherever the partial-update logic lives). Pairs naturally with W1-D4-R2.

### Target drop

R3 + R4 → Drop 4c.8 W4 embedded-prompt content (already known target).
R5, R7, R10 → next ergonomics droplet under `internal/app/dispatcher` + `cmd/till`.
R11 → pairs with W1-D4-R2 in a project-update partial-merge droplet.
R8 → cross-cutting; parking-lot until a real duplicate-key incident surfaces.
R2, R6, R9, R12 → parking-lot; revisit on real bypass / dogfood incident.

### Tags
`agent-isolation`, `hook`, `dispatcher`, `templates`, `parity-clarity`, `rollup`, `cascade-followup`

---

## 2026-05-15 — E2E-8 — `createAuthRequestLive` baseline-ordering latency-only race (NIT, deferred)

### Context
E2E-8 (`till_auth_request operation=create` honor `wait_timeout`) shipped at HEAD `ed5f29d` (commits `ba58ba7` + `49fa802` + `ed5f29d`). D1 QA-falsification surfaced a latency-only race window in `createAuthRequestLive` at `internal/app/auth_requests.go:297-313`: the live-wait baseline-sequence subscribe happens AFTER `s.authRequests.CreateAuthRequest` returns (correct — the request ID doesn't exist before), but a publish event firing between persist (~line 274) and baseline capture (~line 298) is theoretically lost. Functional correctness is preserved by the post-deadline `GetAuthRequest` re-read; the only consequence is a full `wait_timeout` delay on that rare race.

### Observation
Probability is very low in practice — between persist and baseline-capture is a few hundred microseconds, and no external caller has the request ID before create returns. Sibling `claimAuthRequestLive` captures baseline BEFORE the action, avoiding the window entirely, but for create the field doesn't exist pre-persist.

### Proposed fix
Investigate whether the gateway can return the request ID synchronously BEFORE persist commits (allowing pre-baseline subscribe). If yes, restructure `Service.CreateAuthRequest` to capture baseline immediately after ID mint. If no, accept the latency-only window as designed and document.

### Target drop
Parking-lot — latency-only, functional correctness preserved, no user-visible impact. Revisit only if a real-world race is observed in dogfood.

### Tags
`auth`, `live-wait`, `latency`, `parking-lot`

### 2026-05-15 — Shipped state
E2E-8 entry in `E2E_FIXES.md` flipped to `fixed`. Schema description on `internal/adapters/mcp_rpc/handler.go:136` now enumerates honored ops (`create` + `claim`).

---

## 2026-05-15 — Phase 4.3 — bundle `--language` CLI flag teardown

### Context
Phase 4.2 removed `domain.Project.Language` + every read/write across the codebase. The CLI surface (`cmd/till project init` + `till project update`) still parses `--language` into a local var and discards via `_ = language`; helpers + UX paths remain. Deferred to Phase 4.3 by deliberate scope guard during Phase 4.2.

### Observation
Four surviving artifacts (D7 QA-falsification 1.3/1.4/1.5/1.6), all rooted in the still-declared `--language` flag:

1. **`_ = language` suppression** at `cmd/till/project_cli.go:~204-220` (update) and `~196` (init) — flag parsed, value dropped.
2. **Dead helper `mapGroupsToLanguage`** at `cmd/till/init_cmd.go:654` — zero live callers, still has its unit test at `init_cmd_test.go:2547`. Both can go.
3. **Init/update validation inconsistency** — `runProjectUpdate` rejects unknown values with `"invalid language"`; `runProjectInit` silently accepts any string. Test `TestRunProjectUpdate_LanguageValidation` (`project_cli_test.go:609`) still asserts the rejection.
4. **`writeProjectDetail` empty Language row** at `cmd/till/project_detail.go:~91` — row still renders with empty value via `compactText("")`. Phase 4.3 should drop the row outright.

Also: **mcp_rpc schema-vs-decoder mismatch** (D6 QA-falsification REFINEMENT). `internal/adapters/mcp_rpc/extended_tools.go:446` still declares `mcp.WithString("language", ...)` as a valid `till.project` schema parameter; args struct no longer carries the field; `bindArgumentsStrict` with `DisallowUnknownFields()` will reject `{"language":"go"}` with `invalid_request: unknown field "language"`. Published schema does not match decoder behavior. No production caller hits it today (CLI/TUI bypass MCP RPC), but agent-MCP callers following the schema will break.

### Proposed fix
Single Phase 4.3 droplet: "Retire `--language` CLI/MCP surface."

1. Remove the `--language` flag declaration from `till project init` and `till project update`.
2. Delete `mapGroupsToLanguage` helper + its test.
3. Delete the language local var + `_ = language` suppression lines.
4. Delete the "invalid language" CLI validation guard + its test (`TestRunProjectUpdate_LanguageValidation`).
5. Remove the Language row from `writeProjectDetail`.
6. Remove `mcp.WithString("language", ...)` from `extended_tools.go:446` (schema-side).
7. `mage ci` green at the end.

### Target drop
Phase 4.3.

### Tags
`cli`, `mcp`, `cleanup`, `phase-4.3`

### 2026-05-15 — Shipped + corrigendum
- **Shipped** as Phase 4.3 in 4 droplets: D1 `0d9009a` (mapGroupsToLanguage helper + test) + D3 `90e870d` (mcp_rpc schema param + REQUIRED rejection sub-test) + D4 `536ba3a` (TUI `projectFieldLanguage` form input) + D2 `aecf640` (`--language` flag teardown). `mage ci` GREEN 3280/3280 at chain tail; cmd/till coverage 77.5%.
- **Corrigendum** (plan-QA falsification 2.1): the original Observation/Proposed-fix referred to `cmd/till project init` having a `--language` flag — factually wrong. The flag was only on `till project update` (`cmd/till/main.go:745` pre-fix). Original Observation line referencing `cmd/till/project_detail.go:~91` is also off — the function lives in `cmd/till/project_cli.go` (~line 91 was approximate). Entries kept as-written per REFINEMENTS append-only schema; corrigendum here records the actual surface.

---

## 2026-05-15 — Phase 4.2 close — planner missed 3 of 7 surfaces

### Context
Phase 4.2 (`PHASE 4.2 REMOVE PROJECT LANGUAGE FIELD`) was originally decomposed into 5 droplets by the planner spawn: D1 domain → D2 app → D3 sqlite → D4 mcp_common → D5 mcp_rpc. Actual landed surface required 7 droplets: D1-D7 covering domain, app, mcp_common, tui (planner-miss), sqlite-finalize, mcp_rpc, cmd/till+dispatcher-fixtures (planner-miss + D7 absorbed D8's scope).

### Observation
Three packages were not in the original plan despite holding `project.Language` reads / `app.*ProjectInput.Language` writes that broke `mage build` and/or `mage test`:

1. **`internal/tui`** — 5 compile errors in `model.go` + `thread_mode.go`. Caught only after running `mage build` post-D2.
2. **`cmd/till`** — 5 compile errors in `init_cmd.go` + `project_cli.go`. Caught only after running `mage build` post-D6.
3. **`internal/app/dispatcher` + `cli_claude/render`** — test fixtures `Language: "go"` in `domain.Project{}` literals. Caught only by D4 QA-falsification's full `mage ci` rerun.

Pattern: planner relied on `grep -l Language internal/` against a tree where `domain.Project.Language` itself was still present (Phase 4.2 was BEFORE the field removal). Once the field was gone, every transitive consumer surfaced.

### Proposed fix
Planner spawn-prompt addition for cross-package refactor plans: BEFORE decomposing, planner runs `git grep -n '<symbol>' <broad-paths>` against the CURRENT tree AND simulates the post-removal state by examining each `domain.Project{...}` / `app.*ProjectInput{...}` struct-literal site. The simulation catches consumer breakage that simple grep misses when the field still exists.

Concrete implementation: add a "Cross-package consumer audit" section to `go-planning-agent.md`'s decomposition workflow — before sizing droplets, enumerate every struct-literal site that references the to-be-removed field + every reader of that field. Each site → one droplet OR explicitly bundled into an adjacent droplet.

### Target drop
`~/.claude/agents/go-planning-agent.md` update (no Tillsyn drop needed — agent definition file). Could land as part of methodology refinement or independently.

### Tags
`planner`, `methodology`, `refactor`, `decomp`

### 2026-05-15 — Validated by Phase 4.3
The Phase 4.3 planner spawn ran the cross-package audit recommended here AND the plan-QA pair caught 5 additional NITs before any builder dispatched. Result: 4-droplet decomp shipped clean in one pass (no planner-miss, no reactive add-on droplets). Methodology refinement validated end-to-end. The `go-planning-agent.md` update remains a future durable change to capture the pattern across all Go projects.

---

## 2026-05-15 — Phase 4.2 — `mage testPkg` suppresses compiler error text

### Context
Phase 4.2 builders for mcp_rpc (D6) and tui (D4) both reported "outcome UNKNOWN — `mage test-pkg` reports `build errors: 1` but the gotestout renderer suppresses the actual compiler error messages." Builders could not diagnose the failure without orchestrator-direct help.

### Observation
Running `mage test-pkg ./internal/adapters/mcp_rpc` on a package with one compile error in a test file emits:

```
[PKG FAIL] github.com/evanmschultz/tillsyn/internal/adapters/mcp_rpc (0.00s)
Test summary
  build errors: 1
...
```

But the actual `go test` stderr (which carries the `unknown field Foo in struct literal` text) is filtered out by the gotestout JSON renderer. Builders must either (a) ask the orchestrator to look directly, (b) blind-grep, or (c) fall back to `mage build` (which DOES surface errors, but only for production code, not test files).

`mage build` is informative; `mage test-pkg` is not. Both should be informative.

### Proposed fix
Update the gotestout renderer used by `mage test-pkg` (and probably `mage ci`'s test phase) to forward build-error stderr to the user terminal when `build errors > 0`. Either:

1. Stream `go test -json` build-error events to stderr as plain text alongside the JSON parse.
2. Add a final "build errors" section that dumps the raw `go test` stderr accumulated.

### Target drop
Mage tooling refinement drop (no specific target yet — parking-lot until next dev-tooling pass).

### Tags
`mage`, `tooling`, `diagnostics`, `dx`

---

## 2026-05-15 — Phase 4.2 — `projectFieldLanguage` dead form state until Phase 4.3

### Context
Phase 4.2 D4 (`refactor(tui): drop project.Language reads + struct fields`) preserved `projectFieldLanguage` enum + form-input rendering per Phase 4.3 deferral. The input still allocates + renders, accepts user typing, but submit + pre-population paths no longer read or write it.

### Observation
Edit-form flow: open project, focus moves through fields including the Language input, user types "go", hits enter. Submit silently drops the typed value (no `Language:` field in `app.UpdateProjectInput`); pre-population sets `""` instead of reading from project. Mildly confusing UX: looks like a working field, behaves like `/dev/null`.

### Proposed fix
Phase 4.3 either: (a) remove `projectFieldLanguage` from `projectFormFields` + `renderProjectInput("language", ...)` call site + the iota entry + every form-test assertion, OR (b) re-wire it to a real `metadata.language` storage path if there's a remaining design need. Likely (a) given Phase 4.2's direction.

### Target drop
Phase 4.3 (alongside `--language` CLI flag teardown above).

### Tags
`tui`, `cleanup`, `phase-4.3`

---

## 2026-05-15 — methodology — compile-coupled droplet chains defer `mage ci` to the chain tail

### Context
Phase 4.2 Droplet 1 (`refactor(domain): remove project.Language field`, commit `8f3a418`) committed locally with `mage test-pkg ./internal/domain` GREEN but full-tree `mage ci` RED. The downstream packages (`internal/app`, `cmd/till`, `internal/adapters/*`, `internal/tui`) reference `project.Language` and `ErrInvalidLanguage` which the droplet removed; their fixes live in Droplets 2-5. Builder + sibling QA proof PASSED within the droplet's declared `paths` scope, but QA falsification CAUGHT a hard conflict with CLAUDE.md "Build Verification" rule 1 ("all relevant mage targets pass") + post-build gate ("`mage ci` on fail → build moves to `failed`"). The rule was written for self-contained droplets and doesn't accommodate compile-coupled cross-package refactors where the only way to honor the atomic-droplet sizing rule (1-4 code blocks) is to leave CI red between intermediate droplets.

### Observation
Two competing rules:

1. **Atomic droplet sizing** (per `feedback_plan_down_build_up.md`): each `build` action item is 1-4 code blocks. Reviewable independently. A planner decomposes a multi-package refactor into N small droplets rather than one giant blob.
2. **Build verification** (CLAUDE.md "Build Verification" rule 1): every `build` action item passes `mage ci` before `complete`. Post-build gate runs `mage ci` and fails the build on red.

For compile-coupled refactors (rename / remove / change-signature touching N packages), these rules collide. Honoring atomicity = `mage ci` red between droplets. Honoring `mage ci` per-droplet = squashing into one un-reviewable mega-droplet.

### Proposed fix — Route A formalized

**A multi-droplet chain is one logical refactor that the planner decomposes for review-ability, not for CI-per-droplet.** The chain's invariants:

1. Every droplet in the chain declares its `paths` to the single package it touches.
2. Each droplet's build gate is `mage test-pkg <package>` — the touched package compiles + its tests pass.
3. Every droplet except the FIRST carries `blocked_by` pointing at the prior droplet, enforcing sequential execution.
4. **`mage ci` is the chain gate, NOT the droplet gate.** It runs on the LAST droplet in the chain (or before push, whichever is first).
5. **Push is held until the chain completes.** All chain commits stay local until the chain's last `mage ci` passes. This means intermediate CI-red commits never reach origin.
6. Per-droplet QA proof + falsification still run; falsifier is expected to surface the CI-red intermediate state as an observation — orchestrator routes it via this methodology rather than failing the droplet.

The plan action item parents the chain. Plan-QA twins review the decomposition + chain integrity (no orphan droplets, all `blocked_by` edges wired, last droplet's `mage ci` actually clears the full tree).

### Worked example — Phase 4.2

- Plan `6e41ec19` PHASE 4.2 REMOVE PROJECT LANGUAGE FIELD (drop): chain parent.
- Droplet 1 `7bad55cd` (domain): builds, `mage test-pkg ./internal/domain` green, `mage ci` red (expected — referenced in app + adapters). `complete` per Route A.
- Droplet 2 `<TBD>` (app): builds against domain commit, `mage test-pkg ./internal/app` green, `mage ci` partially red (adapters still reference Language).
- Droplet 3 + 4 `<TBD>` (storage + mcp_common, parallel after 2).
- Droplet 5 `<TBD>` (mcp_rpc): final droplet, `mage ci` GREEN across full tree, push.

If any droplet in the chain fails its `mage test-pkg <package>` gate, the chain pauses; orchestrator decides whether to fix forward (extra droplet) or abandon the chain (revert all commits + redecompose).

### Target drop
This entry IS the methodology — no separate drop. The rule lands here so future Phase 4.x / Phase 5.x / etc. multi-droplet chains have prior art to cite.

### Tags
`methodology`, `cascade`, `build-verification`, `compile-coupled-refactor`, `chain-semantics`

---

## 2026-05-15 — phase-4.2-orphans — Predicted orphans after `project.Language` removal

### Context
Phase 4.2 (PHASE 4.2 REMOVE PROJECT LANGUAGE FIELD, Tillsyn plan `6e41ec19-347e-4acc-835e-f96137c41fbf`) is decomposed by `go-planning-agent` into 5 atomic droplets. The decomposition predicts the following orphans — pre-logged so each becomes its own future plan rather than expanding Phase 4.2's scope.

### Orphans predicted

1. **`mcp.WithString("language", ...)` tool-schema declaration in `internal/adapters/mcp_rpc/extended_tools.go:446`** — the MCP transport still declares `language` as a request key after Phase 4.2 removes the field from `mcpcommon.CreateProjectRequest` / `UpdateProjectRequest` and from the inline args struct. Phase 4.3 retires this declaration (alongside `--language` CLI flag + TUI `projectFieldLanguage`). Until Phase 4.3 lands, callers that pass `language` in JSON have it accepted-then-dropped at the request boundary.

2. **`loadStewardSeedTemplate(project.Language)` at `internal/app/auto_generate_steward.go:116`** — Phase 4.2 Droplet 2 replaces this with `loadStewardSeedTemplate("")` as a temporary stub. The empty-language path selects the generic embedded template for every project. Phase 4.4 retires `templates.LoadDefaultTemplateForLanguage` and migrates STEWARD seed materialization to a project-tier or aggregated-template mechanism. The `""` stub is intentional transitional state.

3. **`templates.LoadDefaultTemplateForLanguage`** in `internal/templates/embed.go` — still called by `loadStewardSeedTemplate` after Phase 4.2. Phase 4.4 retires it entirely after STEWARD seed migration. Until then, the function is the only remaining production consumer of the language→embedded-template mapping.

4. **`embeddedSourceForLanguage` + `templateBakeSourceEmbeddedGeneric` + `templateBakeSourceEmbeddedGo`** in `internal/app/template_service.go` — already dead-code after Phase 4.1's `f3a9df7` commit (the only caller `resolveProjectTemplateWithSource` no longer fires the fallback). Deleted in Phase 4.2 Droplet 2 cleanup (NOT a separate plan).

### Proposed fix
Each orphan is addressed by its already-scheduled phase:

- Orphan 1: Phase 4.3 (CLI / TUI / MCP schema removal).
- Orphan 2 + 3: Phase 4.4 (STEWARD seed migration + `LoadDefaultTemplateForLanguage` retirement).
- Orphan 4: handled inline in Phase 4.2 Droplet 2 — already dead, just deletion.

### Target drop
N/A — this entry is the index of orphans for Phase 4.2; the fixes live in 4.3 + 4.4 above (which are also entries in this file). This entry exists so future readers can grep "orphan" and find the connection between phases.

### Tags
`phase-4.2`, `orphans`, `language-removal`, `tracking`

---

## 2026-05-14 — pre-dogfood — Remove `project.Language` field; templates are project-tier opt-in only

### Context
Dev caught architecture drift while diagnosing why TILLSYN's `till action_item create --kind build` did NOT auto-spawn QA twins. Tracing the load path revealed:

1. `project.Language` is a closed enum (`"" | "go" | "fe"`) added in Drop 4a L4 (`a334f20`).
2. The dispatcher / service-layer template resolver uses `project.Language` to pick an EMBEDDED-default template (`till-go.toml`, `till-fe.toml`, `till-gen.toml`) when no project-tier `template.toml` exists.
3. Dev's stated design (forgotten / drifted): **Tillsyn projects are not language-bound.** Projects can be multi-language, non-coding, or have any other shape. The `Language` field bakes a coding-project assumption into a domain primitive that should be vocabulary-neutral.

### Observation
Three coupled problems:

1. **`project.Language` is a wrong abstraction** for a general-purpose project tracker. Carries an implicit "this is a coding project in language X" assumption.
2. **Embedded-default template fallback in production** is a wrong design. If a project has no template, Tillsyn should do NOTHING — no child_rules, no enforced kinds, no auto-creation. Templates are user-authored OPT-IN at project tier. Embedded templates exist solely as starter content that `till init` can OFFER to copy into the project on first run.
3. **Multi-group projects (`Metadata.Groups`)** are the partial workaround for the wrong-language assumption (multi-group sidesteps the single-Language constraint by per-group resolution). But this preserves the embedded-fallback antipattern and adds its own complexity.

### Proposed fix
1. **Remove `project.Language` field entirely.** Strip from `domain.Project`, `app.UpdateProjectInput` / `CreateProjectInput`, `till project update --language` CLI flag, `till init --language` JSON payload, TUI form, MCP schema. Migrate persisted Project rows by dropping the column.
2. **Remove `project.Metadata.Groups`** OR re-purpose it as a "starter content selector" (which embedded templates to OFFER on `till init`, not a runtime resolver). Open design question.
3. **Make template resolution project-tier-only at runtime.** `loadProjectTemplate` and `loadProjectTemplatesForGroups` should check `<project>/.tillsyn/template.toml` and `<project>/.tillsyn/templates/*.toml` (multi-file aggregation) ONLY. No HOME tier. No embedded fallback. Empty result is valid — Tillsyn just doesn't auto-create children for that project.
4. **`till init` becomes an opt-in template starter.** Optional flag `--starter-template <name>` (or interactive picker) copies one or more embedded templates into the project's `.tillsyn/templates/`. User can edit afterwards. No starter = no template = no auto-create. Pure tracking-only Tillsyn.
5. **Two-and-only-two validators** on aggregated templates: conflict detection (same kind/child_rule ID with different content) + cycle detection (rules that prevent terminal completion). NO structural-type enforcement. NO kind-enum enforcement. NO closed vocabulary checks.

### Target drop
**Pre-dogfood architectural cleanup drop.** Substantial scope — touches domain primitives, migrations, CLI, TUI, multiple service-layer helpers, all template load tests. Likely a dedicated drop. Should ship BEFORE the cascade-dispatcher auto-trigger lands (Drop 4c.7) so the dispatcher's template resolution path is the right shape from day one.

### Tags
`architecture`, `domain`, `templates`, `breaking-change`, `migration`, `cli`, `tui`, `dogfood-blocker`

---

## 2026-05-14 — pre-dogfood — Multi-template aggregation per project tier

### Context
Tied to the `project.Language` removal above. With the language field gone, the question "how do multiple template files at project tier combine" becomes load-bearing.

### Observation
- Today's multi-group path (`loadProjectTemplatesForGroups`) is a primitive iterate-and-merge that mixes embedded fallback into a multi-group walk. It does not implement the dev's design: project tier supports multiple `template.toml` files (e.g. `<project>/.tillsyn/templates/refactor.toml`, `feature.toml`, `bugfix.toml`) and they aggregate by ID merge with conflict + cycle checks only.
- W2.D6 SKIPPED writing project-tier `template.toml` for multi-group projects (per `init_cmd.go:854`) because naive concat of two embedded templates trips the load-time "table plan already exists" error. The right fix is a semantic ID-merge.

### Proposed fix
1. Add `<project>/.tillsyn/templates/*.toml` discovery (glob the directory).
2. Implement `templates.Aggregate([]Template) (Template, error)` that ID-merges kinds, agent_bindings, child_rules. Last-loaded wins on collision OR error on collision — design choice TBD (probably "error on conflict, force the user to resolve").
3. Cycle check across the aggregated graph — heuristic only, ensures no completion deadlock.
4. Update `loadProjectTemplate` to discover-and-aggregate the `templates/` directory in addition to (or instead of) the single `template.toml` legacy file.

### Target drop
Same drop as `project.Language` removal — they share migration surface.

### Tags
`templates`, `aggregation`, `validation`, `dogfood-blocker`

---

## 2026-05-14 — pre-dogfood — User-defined kinds (dynamic enum from template)

### Context
Dev's design: templates define the vocabulary of kinds for that project. A project could declare `refactor-segment`, `feature-drop`, `bugfix-droplet` as its own kinds. The closed 12-value enum at `internal/domain/kind.go` is a stopgap.

### Observation
Today, `domain.Kind` is a closed Go enum. Template `[kinds.<name>]` sections are validated against this enum at load time (unknown kinds reject). User-defined kinds require:

1. `domain.Kind` becomes a free-form `string` type, validated dynamically against the project's loaded template's `kinds` map.
2. Template-load validators stop rejecting unknown kind names — they only validate against the template's own declared set.
3. Cycle detection in child_rules updates to operate on the dynamic kind set.
4. CLI / TUI / MCP surfaces accept any string kind, surface validation errors when the kind isn't in the bound template.
5. `domain.KindAppliesTo` (the scope mapping) becomes per-template metadata rather than a Go-level closed enum.

### Proposed fix
Land AFTER the `project.Language` removal + multi-template aggregation refinements. User-defined kinds is the keystone of the open-vocabulary design but depends on the template plumbing being right first.

### Target drop
**Post-MVP** — closed 12-kind enum works for the immediate dogfood. Dynamic enum is the next architectural layer.

### Tags
`architecture`, `domain`, `templates`, `kinds`, `post-mvp`

---

## 2026-05-14 — pre-dogfood — Toml-driven agent dispatch split (`-p headless` vs orch-signal)

### Context
Dev's design: not every agent in the cascade should be launched directly by Tillsyn's dispatcher. Some agents need to run as the user's Claude Code orchestrator subagents (oauth-billed, June-15-ToS-compliant interactive sub-spawn). The split is **toml-driven**.

### Observation
Today's dispatcher path (Drop 4a Wave 2 + manual-trigger CLI in Wave 2.2) treats every agent identically: spawn a subprocess via `claude --agent ...` (or equivalent). This doesn't accommodate the split.

The intended model:

- **Tillsyn-launched agents** (`agent.dispatch_mode = "headless"` in agents.toml): codex, openrouter, openai-compat, ollama, claude-api-key, claude-oauth-headless. Tillsyn dispatcher spawns them directly. User pays via their own credential setup (`env_from_shell`, `--api-key`, etc.).
- **Orch-signaled agents** (`agent.dispatch_mode = "orch_subagent"`): typically `oauth-claude-subagent`. Tillsyn does NOT launch. Tillsyn pushes a wake-up event to the orchestrator's MCP client via LiveWait (see separate refinement); the orch's `Agent` tool spawns the subagent using the user's Claude OAuth subscription.

### Proposed fix
1. Add `dispatch_mode` field to `AgentBinding` schema. Closed enum: `headless | orch_subagent`. Default to `headless` for backwards compat.
2. Dispatcher routes by `dispatch_mode`. `headless` continues the existing path. `orch_subagent` publishes a `LiveWaitEventOrchSpawnRequested` event with the action_item ID + agent binding.
3. Orch's MCP client subscribes to that event channel via the Channels API (or equivalent push surface) and routes the wake-up into the orch's conversation context as a system reminder asking the orch to spawn the specified agent.
4. The orch confirms (or declines per its own policy) and uses its `Agent` tool to spawn the subagent.

### Target drop
**Pre-MVP-dogfood phase 2**: after the basic dispatcher path works end-to-end (Drop 4c.7 auto-trigger), the split lands as a follow-on.

### Tags
`dispatcher`, `agents`, `architecture`, `tos`, `oauth`, `dogfood-blocker-phase-2`

---

## 2026-05-14 — pre-dogfood — LiveWait → MCP push to orch (replace `/loop` hack)

### Context
Today the orchestrator polls Tillsyn state via `/loop` cadence to learn about attention items, completed action items, approval requests. The LiveWait broker (`internal/app/live_wait.go`) exists for in-process / cross-process Tillsyn-side wake-ups but does NOT push events through the MCP boundary to the orchestrator's conversation.

Per Claude Code's Channels research-preview feature (https://code.claude.com/docs/en/channels.md), MCP servers can be wrapped as channel plugins that push events into a running session. The Tillsyn MCP server should adopt this so the orch wakes immediately on relevant state changes instead of polling.

### Observation
Coupled with the toml-driven dispatcher split above: when Tillsyn determines an `orch_subagent` agent should fire, it publishes `LiveWaitEventOrchSpawnRequested`. The MCP-as-channel surface receives that event and routes a system reminder into the orch's conversation. The orch's tool surface includes a way to "claim" the wake-up (acknowledge + spawn) so duplicate dispatches don't fire.

### Proposed fix
1. Wrap Tillsyn MCP as a Claude Code channel plugin (Channels API).
2. Channel publishes events for: auth_request approval, attention item raised, action_item state change, handoff created, orch_subagent spawn request.
3. Channel events route into the orch's conversation as system reminders or similar.
4. Orch tool surface includes ack/claim for spawn requests.

### Target drop
**Pre-MVP-dogfood phase 2**. Replaces `/loop` polling entirely for orch-side coordination. Depends on the Channels API stability.

### Tags
`mcp`, `live-wait`, `orch-coordination`, `channels`, `wake-up`, `dogfood-blocker-phase-2`

---

## 2026-04-14 — Drop 0 — Local git hooks for gofumpt + `mage ci` parity

### Context
Drop 0 closeout surfaced that the 18.3 builder caught gofumpt drift on `internal/adapters/server/common/app_service_adapter_outcome_test.go` (pre-existing on `main`, not introduced by 18.3) only because `mage ci`'s Formatting stage ran `go tool gofumpt -l` and listed the file. No local gate had caught it at commit or push time, so the drift sat on `main` until a later build job tripped over it. CI formatting checks are correctly read-only (`-l` not `-w`) — the gap is upstream of CI, not inside it.

### Observation
Two distinct issues bundled:

1. **No local pre-commit / pre-push hooks.** Dev workflow relies on developer discipline (`mage format .` then `mage ci` before push). Drift can land on `main` when that discipline slips.
2. **`mage format` no-arg ergonomics wart.** `func Format(path string) error` (`magefile.go:200`) requires a positional arg from the mage CLI, so `mage format` fails with "not enough arguments for target \"Format\", expected 1, got 0". The `if path == "" || path == "."` branch in the function body (lines 201-211) handles the whole-tree case but is unreachable via the CLI — the dev has to type `mage format .` (with dot) to trigger it. Dead-code-from-CLI surface.

### Proposed fix
1. Add committed `.githooks/pre-commit` that runs a new `mage format-check` target (public wrapper around the existing private `formatCheck()` at `magefile.go:218-236`). Fails the commit if gofumpt would modify any tracked `.go` file; error message points the dev at `mage format .`.
2. Add committed `.githooks/pre-push` that runs `mage ci` in full. Matches the "Mage Precommit = CI Parity" feedback rule — the dev should see the same verdict locally that GH Actions will return.
3. Add `mage install-hooks` target that runs `git config core.hooksPath .githooks` so the tracked scripts become active for any fresh clone. Idempotent.
4. Fix `mage format` signature: split into `Format()` (no-arg = whole tree via `trackedGoFiles()`) and `FormatPath(path string)` (scoped); or adopt a variadic `Format(paths ...string)` form. Either way, `mage format` with no args should format the whole tree and not error.
5. Hooks must remain bypassable via `--no-verify` per existing discipline (global CLAUDE.md rule: never bypass without explicit dev instruction).
6. QA-proof + QA-falsification required — the hook scripts are the local build gate, can't silently break.

### Target drop
**Drop 1 — first item.** Scheduled directly into `PLAN.md` §19.1 as the first bullet of the Drop 1 work list.

### Status
**Scheduled — Drop 1 item 1** (2026-04-14).

### Tags
`mage`, `git-hooks`, `tooling`, `ci-parity`, `gofumpt`

---

## 2026-04-14 — Drop 0 — TUI esc-back navigation does not step up one level

### Context
Dev was navigating the main-screen tree during Drop 0 Tillsyn dogfooding. Drilled into a drop subtree and hit esc to return to the immediately previous level.

### Observation
From the main screen, once focused down into a subtree, the `{todo | prog | done}` column-state screen does **not** respect navigation history. Pressing esc from the column-state screen returns directly to the top-level project screen instead of stepping up one level to wherever the focus came from.

### Proposed fix
Esc should behave like browser back: pop one level of navigation history on each press, not short-circuit to project root. Implement a nav-history stack on the main screen so esc pops the most recent push, regardless of column-state depth.

### Target drop
Drop 1 or a later dedicated TUI polish drop. Not Drop 0 — out of scope for the current closeout.

### Tags
`tui`, `navigation`, `ergonomics`

---

## 2026-04-14 — Drop 0 — Dotted-address fast-nav across TUI / CLI / MCP

### Context
Drop 0 vocabulary convergence established dotted addresses (`0.1.5.2`, `proj_name-0.1.5.2`) as the human-readable shorthand for drop references, distinct from UUIDs which remain authoritative for mutations. Today, dev ↔ orchestrator cross-reference happens by copy-pasting UUIDs, which is high-friction.

### Observation
No TUI / CLI / MCP surface today understands dotted addresses. Examples of the intended UX:

- **TUI**: dev types `0.1.5.2` or `8.9.3` into a go-to / search field and is focused on that drop.
- **CLI**: `till view tillsyn-8.9.3`, `till comment tillsyn-8.9.3 "looks good"`, `till state tillsyn-8.9.3 done` — all resolve the dotted path to the current UUID and operate on it.
- **MCP**: orchestrator can pass dotted addresses to tool calls for **reads** (`till.action_item(operation=get, address="0.1.5.2")`). Mutations should still require UUID — dotted addresses shift under re-parenting.

Project-name prefix (`tillsyn-`) is unnecessary inside a scope-bound surface (TUI already knows the project; MCP session is project-scoped). Required for cross-project references.

### Proposed fix
1. Add a dotted-address resolver in `internal/domain` (or `internal/app`) that walks the drop tree by position to find the UUID.
2. Wire the resolver into TUI go-to input, CLI positional args, and MCP read operations.
3. Document the mutations-are-UUID-only rule so no agent accidentally relies on a dotted address for a `till.action_item(operation=update)` call.

### Target drop
Post-Drop-3 template overhaul or a dedicated addressing drop. Not Drop 1.

### Tags
`tui`, `cli`, `mcp`, `addressing`, `ergonomics`

---

## 2026-04-14 — Drop 0 — Batch operations on action-item nodes

### Context
Orchestrator + cascade agents frequently perform many small action-item mutations in sequence (create N drops, update M descriptions, move K items to `in_progress`). Every call is a separate MCP round-trip.

### Observation
Post-Drop-4 the cascade dispatcher will be doing hundreds of these per cascade run. One-at-a-time MCP round-trips will become a real latency and rate-limit problem. Pre-cascade, the orchestrator already feels the friction (e.g. creating refinement drops, creating build-actionItem + qa-proof + qa-falsification trios).

### Proposed fix
Batch operations on `till.action_item`:

- `till.action_item(operation=create_batch, items=[...])` — create N items in one call.
- `till.action_item(operation=update_batch, updates=[...])` — apply N updates in one call.
- `till.action_item(operation=move_state_batch, moves=[...])` — bulk lifecycle transitions.
- Configurable limit per call (e.g. 25 items) to bound request size.

Atomicity policy (all-or-nothing vs best-effort with per-item error rows) is a design question — lean toward best-effort with a results array so partial success is observable.

### Target drop
Post-Drop-4 (dispatcher drop) — the cascade makes the cost real. Could be pulled earlier if pre-cascade friction gets noisy.

### Tags
`mcp`, `performance`, `ergonomics`, `refactor`
