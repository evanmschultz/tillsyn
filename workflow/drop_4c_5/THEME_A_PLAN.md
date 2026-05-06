# Drop 4c.5 — Theme A Plan (Silent-Data-Loss + Agent-Surface Hardening)

**Author:** Theme A planner.
**Status:** decomposition draft awaiting plan-QA. A.1 state: done.
**Source brief:** `workflow/drop_4c_5/REVISION_BRIEF.md` §3.1.
**Evidence basis:** direct code reads (LSP/Read/Grep) on `main` HEAD `7cd84ec`. No Hylla calls (stale post-Drop-4c-merge).

## 0. Theme Scope Recap

Four atomic droplets close audit-debt items that today let agents corrupt or misroute Tillsyn state without an error surfacing:

| ID  | Title (short)                                       |
|-----|-----------------------------------------------------|
| A.1 | PATCH semantics on `Service.UpdateActionItem`       |
| A.2 | Reject unknown JSON keys at MCP boundary            |
| A.3 | Server-infer / require non-empty `client_type`      |
| A.4 | Require non-empty `metadata.outcome` on `failed`    |

All four are pre-MVP correctness work; together they are mandatory before Drop 5 dogfood (per `REVISION_BRIEF` §9 Q5: A + B mandatory).

## 1. Cross-Cutting Decisions / Tradeoffs

- **Pointer-sentinel pattern is precedent.** `Service.UpdateActionItemInput` already uses pointer-sentinels for `Owner`, `DropNumber`, `Persistent`, `DevGated`, `Paths`, `Packages`, `Files`, `StartCommit`, `EndCommit`. A.1 extends the same pattern to `Title`, `Description`, `Priority`, `DueAt`, `Labels`. This is a backward-compatible struct-shape change for callers that supply zero-values today (they will need to migrate to pointers OR adopt a "preserve empty" sentinel — see A.1 for the chosen shape).
- **Strict-decode is per-tool, not framework-wide.** `BindArguments` (mark3labs) uses bare `json.Unmarshal`. We cannot patch the upstream library; we replace the call with a small `bindArgumentsStrict` helper that raw-marshals → decodes via `json.NewDecoder().DisallowUnknownFields()`. Per-tool because each anonymous-struct shape is the source-of-truth schema.
- **`client_type` is server-stamped, not client-asserted.** Dropping `client_type` from agent-supplied input prevents an agent from impersonating another adapter family. The stamper lives at the adapter seam (MCP handler / CLI cobra command), so cascade-spawned auth requests inherit the CLI's `"cli"` value (Q4 resolution: today the dispatcher provisions auth via the CLI, so adapter-stamping is sufficient; if dispatcher gains a direct `CreateAuthRequest` call later, that path stamps `"cli-cascade"` or similar — out of scope for 4c.5).
- **A.4 is a state-transition guard, not a metadata-shape guard.** `validateMetadataOutcome` in `app_service_adapter_mcp.go` correctly accepts empty (outcome may be unset on a non-terminal item). The new check belongs at the move-state boundary in `Service.MoveActionItem` for `toState == StateFailed`.
- **No migration logic.** Per `feedback_no_migration_logic_pre_mvp.md`, schema/state-vocab changes are dev-deletes-DB. None of A.1-A.4 changes schema.
- **Test bodies are table-driven** (`feedback_orchestrator_no_build.md` cross-ref: builder writes them). Each droplet ships its tests in the same builder spawn.

### Sibling ordering (`blocked_by`)

A.1 and A.2 both edit MCP handler-adjacent files for the action-item update path; they share **no** identical `paths` (A.1 = service + UpdateActionItemInput shape; A.2 = MCP adapters' decoder helper). A.3 touches `auth_requests` + `mcpapi/handler.go` + `cmd/till/main.go`. A.4 touches `service.go` + tests only. A.1 touches the same `service.go` as A.4 — **package-level lock collision** on `internal/app`.

Resolution: A.4 `blocked_by: A.1`. Builder for A.4 picks up A.1's pointer-sentinel + tests already in tree. A.2 and A.3 are independent of A.1 and of each other (different files; A.3 touches `mcpapi/handler.go` but only the auth-request branch lines 187-205, while A.2 touches the same file but at the `BindArguments` call sites). Conservative additional `blocked_by`: **A.2 → A.3 within `mcpapi/handler.go`** (both edit the same file even though the line ranges don't overlap). Locking order: A.1 → A.2 → A.3 → A.4.

## 2. Droplets

### A.1 — Pointer-Sentinel PATCH Semantics on `Service.UpdateActionItem`

**Purpose:** Stop empty-string `Description`/`Title`, zero-value `Priority`/`DueAt`/`Labels` from clobbering stored values when a partial-update caller omits them.

**Files / paths to modify:**

- `internal/app/service.go` — `UpdateActionItemInput` struct (lines 664-763); `Service.UpdateActionItem` body (lines 1201-1388, specifically the `priority` defaulting block 1226-1232 and the `actionItem.UpdateDetails` call at 1230).
- `internal/app/service_test.go` — extend existing `UpdateActionItem` tests; add new table cases for partial-update preservation.
- `internal/adapters/server/common/app_service_adapter_mcp.go` — adapter-side `UpdateActionItem` mapping (search for `app.UpdateActionItemInput{` and adjust to populate the new pointer fields from the request struct).
- `internal/adapters/server/mcpapi/extended_tools.go` — wherever the action-item update tool builds `UpdateActionItemRequest` (currently passes raw strings; switch to JSON-tag-driven `*string` pointers if the wire schema change lands; otherwise keep wire compat by detecting JSON-key-presence in the strict-decoder of A.2 — see Falsification mitigations).
- `internal/tui/` — any TUI call site that constructs `UpdateActionItemInput` directly (audit via grep).

**Packages affected:**

- `github.com/evanmschultz/tillsyn/internal/app`
- `github.com/evanmschultz/tillsyn/internal/adapters/server/common`
- `github.com/evanmschultz/tillsyn/internal/adapters/server/mcpapi`
- `github.com/evanmschultz/tillsyn/internal/tui` (only if call sites exist; builder confirms via grep)

**Acceptance criteria:**

1. `UpdateActionItemInput` exposes `Title *string`, `Description *string`, `Priority *domain.Priority`, `DueAt **time.Time` (or stays `*time.Time` if existing nil-vs-zero test asserts that), `Labels *[]string`. Pointer-sentinel rule: nil = preserve, non-nil = apply (empty deref = explicit clear, mirroring Owner/Paths).
2. `Service.UpdateActionItem` body branches on each pointer; only non-nil pointers reach `actionItem.UpdateDetails`. The existing `UpdateDetails` domain method either gets a sibling `UpdateDetailsPartial(...)` OR the service inlines the per-field assignments (builder picks the smaller diff; prefer adding a domain helper to keep validation centralized).
3. Existing tests pass unchanged — they currently pass concrete values, which under the new shape become `&value` via a small test helper or migrate per-test (builder picks).
4. Three new table-driven test cases:
   - **Description-preservation:** call with `Description = nil`; pre-stored description is unchanged.
   - **Title-preservation:** call with `Title = nil`; pre-stored title is unchanged.
   - **Explicit-clear-description:** call with `Description = pointerTo("")`; stored description becomes empty (caller intent honored).
5. `UpdateDetails`-equivalent path still rejects empty `Title` when the caller explicitly sets `Title = pointerTo("")` — empty title remains an `ErrInvalidTitle` outcome, identical to today's behavior. (This is the asymmetry vs Description: title is required.)
6. `mage test-pkg ./internal/app` and `mage test-pkg ./internal/adapters/server/common` pass with `-race`.
7. `mage ci` clean.

**Test scenarios (builder must add — table entries in `service_test.go` `TestServiceUpdateActionItemPartial...`):**

| name                                | input pointers                              | pre-store value     | expected post-store           |
|-------------------------------------|---------------------------------------------|---------------------|-------------------------------|
| description nil preserves           | Title=&"new", Description=nil               | "old desc"          | "old desc"                    |
| description empty pointer clears    | Title=&"new", Description=&""               | "old desc"          | ""                            |
| description non-empty replaces      | Title=&"new", Description=&"fresh"          | "old desc"          | "fresh"                       |
| title nil preserves                 | Title=nil, Description=&"new"               | "old title"         | "old title"                   |
| title empty pointer rejected        | Title=&"", Description=&"new"               | "old title"         | err = ErrInvalidTitle         |
| labels nil preserves                | Labels=nil                                  | ["a","b"]           | ["a","b"]                     |
| labels empty pointer clears         | Labels=&[]string{}                          | ["a","b"]           | []                            |
| priority nil preserves              | Priority=nil                                | High                | High                          |
| due_at nil preserves                | DueAt=nil                                   | non-nil time        | unchanged                     |

**Blocked by:** none.

**Falsification mitigations (top 3):**

- **Wire-schema breakage.** The MCP `till.action_item operation=update` wire format today probably accepts JSON like `{"description":""}` and means "clear" or "preserve" depending on caller convention. Falsification would attack: "after A.1, agent X that omits `description` gets preservation but agent Y that explicitly sends `""` gets clear — undocumented diff." Mitigation: A.1 builder MUST update the MCP tool description string (`mcp.WithString("description", ...)`) to document "omit to preserve, send empty string to explicitly clear". The MCP adapter at `extended_tools.go` must distinguish JSON-key-absent from JSON-key-present-with-empty — easiest path is to switch the anonymous-struct field to `*string` so json.Unmarshal sets nil for absent vs `pointerTo("")` for present-empty (interacts with A.2 strict decoder; if A.2's strict decoder rejects unknown keys, both A.1 and A.2 must coordinate the wire-shape change in a single PR).
- **Domain helper duplication.** Adding `UpdateDetailsPartial` while `UpdateDetails` still exists creates two near-identical methods. Mitigation: replace `UpdateDetails` with `UpdateDetailsPartial` and adjust the one create-time caller (`UpdateActionItem` call at 1230); production code only calls it from `UpdateActionItem`, so single-call-site refactor is safe. Builder verifies via LSP `references` lookup that no test or sibling production code calls `UpdateDetails` directly.
- **TUI / dispatcher silent breakage.** TUI or dispatcher code paths that construct `UpdateActionItemInput{Title: someString}` (no pointer) would silently fail to compile after the shape change — that's good; compile error surfaces every call site. Builder must touch every callsite. Mitigation: builder runs `mage test-pkg ./internal/tui` AND `mage test-pkg ./internal/app/dispatcher` to confirm.

---

### A.2 — Reject Unknown JSON Keys At MCP Boundary

**Purpose:** Stop schema-drift bugs (typo'd field names, deprecated-then-deleted fields) from landing as silent no-ops. An MCP tool call carrying `{"descrption": "..."}` (typo) today drops it; after A.2 it returns a structured error naming the offending key.

**Files / paths to modify:**

- `internal/adapters/server/mcpapi/handler.go` — 5 `BindArguments` call sites (top-level tool handlers).
- `internal/adapters/server/mcpapi/extended_tools.go` — 11 `BindArguments` call sites.
- `internal/adapters/server/mcpapi/handoff_tools.go` — 5 `BindArguments` call sites.
- New file: `internal/adapters/server/mcpapi/strict_decode.go` — exports `bindArgumentsStrict(req mcp.CallToolRequest, target any) error` helper.
- New file: `internal/adapters/server/mcpapi/strict_decode_test.go` — table-driven tests for the helper.
- Update existing `mcpapi/extended_tools_test.go` — at minimum two new test cases asserting unknown-key rejection at MCP boundary (one tool from each of the three files for sample coverage).

**Packages affected:**

- `github.com/evanmschultz/tillsyn/internal/adapters/server/mcpapi`

**Acceptance criteria:**

1. `bindArgumentsStrict` exists, behaves identically to `BindArguments` for valid inputs, returns a structured error for unknown keys. Error shape: `fmt.Errorf("invalid_request: unknown field %q on tool %q: %w", fieldName, toolName, ErrInvalidRequest)` (or equivalent that includes the offending key by name).
2. Implementation strategy: re-marshal `req.Params.Arguments` to bytes, then `json.NewDecoder(bytes.NewReader(b)).DisallowUnknownFields(); decoder.Decode(target)`. The `json.Decoder` error message already includes the field name, so the helper extracts it via `errors.As` against `*json.SyntaxError` / `*json.UnmarshalTypeError` OR via a regex on the error message (prefer `errors.As`; check `go doc json.Decoder.DisallowUnknownFields` for the exact error type).
3. All 21 production `BindArguments` call sites in the three files swap to `bindArgumentsStrict`. Test files retain `BindArguments` (test fixtures, not production decode).
4. The error returned from `bindArgumentsStrict` flows through `invalidRequestToolResult(err)` exactly as today, so MCP clients see a user-facing `invalid_request: unknown field "descrption" on tool "till.action_item"` style message.
5. Three table-driven test cases per call-site sample (one in each of the three files):
   - **Valid input:** all known keys, decoder succeeds, behavior unchanged.
   - **Unknown key:** one extra key, decoder rejects, error names the key.
   - **Multiple unknown keys:** two extra keys, decoder rejects on the first one (json.Decoder's stop-at-first-error semantics — documented behavior).
6. `mage test-pkg ./internal/adapters/server/mcpapi` passes with `-race`.
7. `mage ci` clean.

**Test scenarios (builder must add):**

| target tool                       | input                                            | expected                          |
|-----------------------------------|--------------------------------------------------|-----------------------------------|
| `till.action_item op=create`      | valid full args                                  | passes through                    |
| `till.action_item op=create`      | adds `"made_up_key":"x"`                         | err names `made_up_key`           |
| `till.action_item op=create`      | typo `"descrption":"..."`                        | err names `descrption`            |
| `till.auth_request op=create`     | adds `"ttl":"8h"` (no such field)                | err names `ttl`                   |
| `till.handoff op=create`          | adds `"target":"..."` (no such field)            | err names `target`                |

**Blocked by:** A.1 (same package compile lock on `internal/adapters/server/mcpapi`; A.1 may have touched `extended_tools.go` for the update-tool wire change).

**Falsification mitigations (top 3):**

- **Test fixtures break.** Dev-side test fixtures for tools may carry stale-but-tolerated keys that the strict decoder now rejects. Mitigation: builder runs `mage test-pkg ./internal/adapters/server/mcpapi` first to surface every fixture that needs cleanup; fixes them in the same droplet.
- **Generic anonymous-struct decoder bypass.** The handler at `mcpapi/handler.go:133` declares an anonymous struct local to the handler. The new helper still needs to know the target type at compile time. Mitigation: helper signature `bindArgumentsStrict(req mcp.CallToolRequest, target any) error` — same as `BindArguments` (target is `any`), strict-decode logic does not need static knowledge of the type. The helper extracts the offending key from the json package's error message; verify via `go doc encoding/json.Decoder.DisallowUnknownFields` — the produced error format is `json: unknown field "fieldname"`, which the helper parses by string-prefix.
- **Backward-compat regression for tolerant clients.** Some tolerant existing client may send a future-deprecated key plus the canonical key, expecting the deprecated key to be ignored. Mitigation: this is a deliberate breaking change; document it in the droplet's CONTRIBUTING.md addition (out of scope for builder — orchestrator notes it for closeout). Pre-MVP no production clients depend on tolerance.

---

### A.3 — Server-Infer / Require Non-Empty `client_type` On Auth-Request Create

**Purpose:** Close the asymmetric validation bug — `app.Service.CreateAuthRequest` accepts empty `client_type` while `autentauth.Service.ApproveAuthRequest` (line 829, `internal/adapters/auth/autentauth/service.go`) rejects it. Each adapter family (mcp-stdio, cli, tui) stamps its own value at the adapter seam; downstream agent input cannot override the adapter-stamped value.

**Files / paths to modify:**

- `internal/app/auth_requests.go` — `Service.CreateAuthRequest` (line 224) returns a typed error when `in.ClientType` is empty after trim. Add `domain.ErrInvalidClientType` to `internal/domain/auth_request.go` if not already exported (autentauth has `autentdomain.ErrInvalidClientType` — reuse OR mirror; builder picks the smaller diff).
- `internal/adapters/server/mcpapi/handler.go` — at line 199, **stamp** `ClientType: "mcp-stdio"` regardless of the agent-supplied `args.ClientType`. The `args.ClientType` field on the anonymous struct stays for backward-decode-compat (until the strict-decoder lands and rejects it; A.2 builder coordinates with A.3 to remove the field if A.3 lands first OR keep it and have the adapter ignore agent-sent value).
- `internal/adapters/server/mcpapi/handler.go` — line 113: drop the `mcp.WithString("client_type", ...)` declaration (clients should not send it). Update tool description to note: "client_type is server-inferred from the adapter family — do not send it."
- `cmd/till/main.go` — at lines 2675, 2689, 3055, replace `strings.TrimSpace(opts.clientType)` with the literal `"cli"` so the CLI cannot set a non-`cli` value. Drop the `--client-type` cobra flag if it exists (audit cobra cmd setup).
- `internal/tui/` — any TUI auth-request creation site stamps `"tui"`. (TUI doesn't directly call `CreateAuthRequest`; it goes through the service adapter — confirmed via grep. The adapter's `ClientType` parameter must be set to `"tui"` at the TUI's call site.)
- `internal/adapters/server/common/app_service_adapter_mcp.go` — line 105 already trims; nothing changes here (ClientType passes through).
- `internal/app/auth_requests_test.go` and friends — add new test cases for empty rejection.
- `internal/adapters/server/mcpapi/handler_test.go` — assert MCP-stdio path stamps `"mcp-stdio"` even when caller sends another value (caller-input-ignored test).

**Packages affected:**

- `github.com/evanmschultz/tillsyn/internal/app`
- `github.com/evanmschultz/tillsyn/internal/domain` (only if a new exported error lands there)
- `github.com/evanmschultz/tillsyn/internal/adapters/server/mcpapi`
- `github.com/evanmschultz/tillsyn/cmd/till`
- `github.com/evanmschultz/tillsyn/internal/tui` (only if call sites exist)

**Acceptance criteria:**

1. `Service.CreateAuthRequest` returns wrapped `ErrInvalidClientType` (or equivalent typed error) when `strings.TrimSpace(in.ClientType) == ""`.
2. The MCP-stdio handler unconditionally stamps `ClientType: "mcp-stdio"`. An MCP client cannot send `client_type: "tui"` and have it stick — the server overrides.
3. The CLI sets `ClientType: "cli"` at every auth-request site in `cmd/till/main.go`. No `--client-type` flag remains.
4. `client_type` is dropped from the MCP `till.auth_request` tool's published parameter schema (`mcp.WithString` declaration removed).
5. Q4 resolution (cascade-spawned auth requests): documented in droplet description that today the dispatcher provisions auth via the CLI path (which now stamps `"cli"`), so cascade subagents inherit `"cli"`. If the dispatcher gains a direct `Service.CreateAuthRequest` call later, that call site must stamp `"cli-cascade"` or similar — out of scope for 4c.5; tracked as a Drop 4d / Drop 5 follow-up.
6. Existing tests that pass `ClientType: "mcp-stdio"` via `app.CreateAuthRequestInput` continue to pass (they go through the service-level path directly, simulating an internal caller; no override happens at that layer).
7. New tests:
   - **Empty client_type rejected at service:** `CreateAuthRequest(in.ClientType = "")` returns `ErrInvalidClientType`.
   - **Whitespace-only rejected:** `in.ClientType = "  "` returns same error after trim.
   - **MCP handler overrides client-supplied client_type:** end-to-end MCP request with `{"client_type":"tui"}` produces an auth request with `ClientType == "mcp-stdio"`.
   - **CLI auth-request creates with client_type=cli:** integration test in `cmd/till/main_test.go` (or unit test on the cobra command) asserts the `app.CreateAuthRequestInput` carries `"cli"` regardless of any environment input.
8. `mage ci` clean.

**Blocked by:** A.2 (same-file `mcpapi/handler.go`, same-package compile lock on `internal/adapters/server/mcpapi`).

**Falsification mitigations (top 3):**

- **Test scaffolding breakage.** Many existing tests pass `ClientType: "mcp-stdio"` directly via `app.CreateAuthRequestInput`. Those tests simulate an internal caller — they should not break because the rejection only triggers on empty. Mitigation: builder grep-audits every `app.CreateAuthRequestInput{` literal and confirms each has a non-empty `ClientType:` line. Production code that constructs the input must explicitly set `"cli"` / `"mcp-stdio"` / `"tui"` (no defaults).
- **`autentauth` adapter's `ensureClient` already rejects empty.** A.3 reduces double-validation but builder must verify the autentauth check still fires for the `Approve` path (the original asymmetric branch) — leave that check in place; A.3 only adds the `Create` path symmetry.
- **CLI flag removal vs documentation.** Dropping `--client-type` from the CLI breaks any dev script that passes it. Mitigation: builder also greps `magefile.go` + `.githooks/` + `Makefile` (none expected) + `CONTRIBUTING.md` for `--client-type` usage and updates docs. Pre-MVP scope: no external automation depends on the flag.

---

### A.4 — Require Non-Empty `metadata.outcome` On Transitions To `failed`

**Purpose:** Close the soft-failure path where an agent moves an action item to `failed` without setting `metadata.outcome`, leaving downstream consumers (orchestrator inbox surface, dispatcher gate evaluator) unable to distinguish failure cause from absent metadata.

**Files / paths to modify:**

- `internal/app/service.go` — `MoveActionItem` (line 1043). After the `toState == StateFailed` branch detection (line 1068), add a guard that requires `actionItem.Metadata.Outcome != ""` (post-trim) when the destination is `StateFailed`. Return wrapped `domain.ErrInvalidMetadataOutcome` (new typed error; or reuse `domain.ErrTransitionBlocked` with a specific reason — builder picks based on existing error vocabulary; new error is preferred for differentiation).
- `internal/domain/errors.go` — declare `ErrInvalidMetadataOutcome` if a new error is introduced.
- `internal/app/service_test.go` — new table-driven test cases for failed-transition outcome enforcement.
- `internal/adapters/server/common/app_service_adapter_mcp.go` — `validateMetadataOutcome` stays unchanged (empty remains valid for non-failed states); add a comment cross-referencing the new service-level invariant for `failed`.

**Packages affected:**

- `github.com/evanmschultz/tillsyn/internal/app`
- `github.com/evanmschultz/tillsyn/internal/domain` (only for the new exported error)

**Acceptance criteria:**

1. `Service.MoveActionItem` returns a wrapped `ErrInvalidMetadataOutcome` (or equivalent) when the caller targets `StateFailed` and `actionItem.Metadata.Outcome` is empty (post-trim).
2. The check is positioned AFTER the existing terminal-state guard (line 1079) but BEFORE the column move (line 1099) — so it cannot race with partial state mutations.
3. A separate test that moves to `StateComplete` with empty `metadata.outcome` does NOT fail (asymmetric — only `failed` requires it; documented in `feedback_qa_before_commit.md` semantics).
4. The dispatcher's existing pattern (`UpdateActionItem` to set metadata BEFORE `MoveActionItem` to flip column) is preserved — agents follow the documented order from `CLAUDE.md` § "Action-Item Lifecycle". The check is a regression net for buggy agents that skip the update step.
5. New tests:
   - **Move to failed without outcome rejected:** create item, leave `metadata.outcome` empty, attempt `MoveActionItem` → fails column. Returns wrapped `ErrInvalidMetadataOutcome`. State is unchanged.
   - **Move to failed with outcome=failure succeeds:** set `metadata.outcome = "failure"`, then move to fails column. Succeeds.
   - **Move to failed with outcome=blocked succeeds:** set `metadata.outcome = "blocked"`. Succeeds.
   - **Move to complete without outcome succeeds:** asymmetry assertion — `complete` does not require outcome.
   - **Whitespace-only outcome rejected:** `outcome = "   "` is treated as empty.
6. `mage test-pkg ./internal/app` passes with `-race`.
7. `mage ci` clean.

**Test scenarios (builder must add):**

| name                                          | metadata.outcome before move | toState   | expected                      |
|-----------------------------------------------|------------------------------|-----------|-------------------------------|
| failed-no-outcome rejected                    | ""                           | failed    | err = ErrInvalidMetadataOutcome |
| failed-whitespace-outcome rejected            | "   "                        | failed    | err = ErrInvalidMetadataOutcome |
| failed-with-failure-outcome succeeds          | "failure"                    | failed    | state = failed                |
| failed-with-blocked-outcome succeeds          | "blocked"                    | failed    | state = failed                |
| failed-with-superseded-outcome succeeds       | "superseded"                 | failed    | state = failed                |
| complete-no-outcome succeeds (asymmetry)      | ""                           | complete  | state = complete              |
| in_progress-no-outcome succeeds               | ""                           | in_progress| state = in_progress         |

**Blocked by:** A.1 (same-package compile lock on `internal/app`; A.1 lands the pointer-sentinel pattern, then A.4 adds the `MoveActionItem` guard cleanly without merge conflicts on `service.go`).

**Falsification mitigations (top 3):**

- **Existing tests that call `MoveActionItem(... → StateFailed)` without setting outcome.** Mitigation: builder greps `MoveActionItem` test calls, audits each one targeting fails column. If any test relied on empty-outcome-still-moves, builder updates it to set `metadata.outcome` first (TDD-correct: that test is now wrong because the production behavior has rightly changed).
- **TUI direct-mutation paths bypass the service.** TUI today goes through the service adapter for state changes. If any internal path constructs the action item and calls `repo.UpdateActionItem` directly, the service-level guard is bypassed. Mitigation: builder confirms via LSP `references` on `Service.MoveActionItem` that all production state-flips funnel through it; if a direct-repo bypass exists, a follow-up refinement is logged (out of scope for A.4 — not a new bug, just an existing structural gap).
- **`outcome = "success"` accidentally allowed on `failed`.** The validation accepts any non-empty outcome at A.4; the existing `validateMetadataOutcome` only constrains the closed enum at the MCP boundary. So an agent could send `outcome = "success"` and move to `failed`, which is semantically nonsense. Mitigation: A.4 builder additionally checks that on `toState == StateFailed`, the outcome is in `{"failure", "blocked", "superseded"}` (NOT `"success"`). One additional table case: `failed-with-success-outcome` rejected with a more specific error message. (Builder can decide whether this strict check belongs in A.4 or a follow-up; recommend including it because the cost is one switch-statement and one test row.)

## 3. Notes / Open Questions Routed To Plan-QA

- **Q-A-1 (route to plan-QA falsification):** A.1's wire-shape change for `description` (omit-vs-empty-string semantics) interacts with A.2's strict decoder. If A.1 relies on `*string` to distinguish absent from empty, A.2's strict decoder must NOT reject `null` JSON values for those fields (it shouldn't — `DisallowUnknownFields` only catches unknown keys, not null values; verify via test). Plan-QA should attack: "what if the JSON wire format historically allows callers to send `{"description": null}` and that round-trips through a typed `*string`?"
- **Q-A-2 (route to plan-QA proof):** A.3's CLI-flag removal scope. If `--client-type` is used by `mage` targets or `.githooks/`, the removal breaks dev workflow. Plan-QA proof verifies builder's grep audit covered those paths.
- **Q-A-3 (route to dev / orchestrator):** A.4's strict-failure-outcome-enum check (rejecting `"success"` on `→failed`). Builder may include or defer; orchestrator decides at planning sign-off.
- **Q-A-4 (route to plan-QA falsification):** A.3's MCP handler stamping line. If multiple MCP transports exist (stdio + future http+SSE), the stamper string must vary per-transport. Today only stdio exists; verify via grep that no other MCP transport adapter constructs `CreateAuthRequestInput` directly.
- **Cross-cutting:** all four droplets are within `internal/app` + `internal/adapters/...` — no `cmd/till` core changes except A.3's CLI literal swap. No magefile changes. No `.github/workflows/` changes.

## 4. Summary Table (Quick Reference)

| Droplet | Files (count)                                          | Packages (compile-lock) | Test-Pkg target              | Blocked By |
|---------|--------------------------------------------------------|-------------------------|------------------------------|------------|
| A.1     | service.go + 4 adapters + tests                        | internal/app + 3 more   | `./internal/app`             | —          |
| A.2     | 3 mcpapi/*.go + new strict_decode.go + 1 test          | mcpapi                  | `./internal/adapters/server/mcpapi` | A.1   |
| A.3     | auth_requests.go + handler.go + cmd/till/main.go + tests | app + mcpapi + cmd/till | `./internal/app`, `./cmd/till` | A.2     |
| A.4     | service.go + errors.go + tests                          | app + domain            | `./internal/app`             | A.1        |

## 5. Verification Gates Per Droplet

Each builder spawn for A.1-A.4 verifies via:

- `mage test-pkg <pkg>` for each affected package, with `-race` (the magefile target sets the flag).
- `mage ci` at end of build before commit.
- No `go test` / `go build` / `go vet` / `mage install` ever (per CLAUDE.md § "Build Verification").
- Builder commits with single-line conventional-commit message ≤72 chars per `feedback_commit_style.md`.
- Builder does NOT push (per `feedback_orchestrator_commits_directly.md` and `REVISION_BRIEF` §6 directive); orchestrator pushes at drop end.

## 6. References

- `workflow/drop_4c_5/REVISION_BRIEF.md` §3.1 + §9 Q4.
- `internal/app/service.go:1201-1388` (`Service.UpdateActionItem` body).
- `internal/app/service.go:664-763` (`UpdateActionItemInput` struct with existing pointer-sentinel precedent).
- `internal/app/service.go:1043-1127` (`MoveActionItem` — A.4 insertion point).
- `internal/domain/action_item.go:509-526` (`UpdateDetails` domain method).
- `internal/domain/action_item.go:551-580` (`SetLifecycleState`).
- `internal/adapters/server/mcpapi/handler.go:131-205` (auth-request handler with `BindArguments` + `ClientType`).
- `internal/adapters/server/mcpapi/extended_tools.go` (11 `BindArguments` sites).
- `internal/adapters/server/mcpapi/handoff_tools.go` (5 `BindArguments` sites).
- `internal/adapters/server/common/app_service_adapter_mcp.go:1151-1169` (`validateMetadataOutcome`).
- `internal/adapters/auth/autentauth/service.go:829` (asymmetric `ensureClient` rejection).
- `internal/app/auth_requests.go:224-276` (`Service.CreateAuthRequest`).
- `internal/domain/auth_request.go:480-533` (`NewAuthRequest` — no ClientType validation).
- `cmd/till/main.go:2675-3055` (CLI auth-request call sites).
