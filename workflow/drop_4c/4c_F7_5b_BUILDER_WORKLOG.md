# Drop 4c F.7-CORE F.7.5b — Builder Worklog

## Droplet

F.7.5b — TUI handshake (parse permission_denials, post attention items).

PLAN reference: `workflow/drop_4c/PLAN.md` §6.1 (F.7-CORE F.7.5 split policy)
records the F.7.5 split — F.7.5a (permission_grants table) absorbed into
F.7.17.7 already-merged; this droplet (F.7.5b) ships the handshake type;
F.7.5c (settings.json grant injection) is a separate later droplet.

(Round-1 cited "REV-3 of `workflow/drop_4c/F7_CORE_PLAN.md`" — that file has
no REVISIONS section. Round-2 retracts that citation. The F.7.5 split lives
in master `PLAN.md` §6.1.)

Master PLAN.md L10: handshake fires AT TERMINAL EVENT (not real-time
mid-stream). Memory §6.4: `result` event carries `permission_denials[]`. F.7.4
monitor (committed at `37f5a69`) calls `adapter.ExtractTerminalReport`
returning `TerminalReport.Denials []ToolDenial`.

## What landed

Files added (this droplet only):

- `internal/app/dispatcher/handshake.go`
- `internal/app/dispatcher/handshake_test.go`

Production surface (in `package dispatcher`):

- `AttentionItemStore` — minimal `Create` port, dispatcher-local. Production
  binds to the existing `internal/app` attention service; tests inject a fake.
- `PermissionGrantsStore` — empty placeholder interface so the
  `PermissionHandshake` struct can compose with F.7.5c grant-injection wiring
  later without churn. May be nil today.
- `PermissionHandshake` — struct holding `AttentionStore`, `GrantsStore`,
  optional `Now func() time.Time` clock for deterministic tests.
- `PermissionHandshake.PostDenials(ctx, projectID uuid.UUID, kind domain.Kind,
  denials []ToolDenial) ([]uuid.UUID, error)` — empty input is no-op; non-empty
  iterates per denial, marshals `permissionDenialPayload{tool_name, tool_input,
  project_id, kind}` into `BodyMarkdown`, builds an
  `AttentionKindApprovalRequired` / `AttentionStateOpen` /
  `RequiresUserAction=true` item via `domain.NewAttentionItem`, persists via
  the store; failures aggregate via `errors.Join` and never short-circuit the
  loop.

Tests (`handshake_test.go`):

1. `TestPermissionHandshakePostDenialsEmpty` — nil + empty-slice both no-op,
   no error, no Create calls.
2. `TestPermissionHandshakePostDenialsAllSucceed` — 3 denials, 3 IDs, payload
   round-trips (tool_name / tool_input / project_id / kind), each item carries
   the right `Kind`, `State`, `RequiresUserAction`, `Summary`, and the returned
   ID matches the persisted item's ID; IDs are unique.
3. `TestPermissionHandshakePostDenialsContinuesAfterFailure` — 2nd-of-3
   Create fails: aggregated error wraps the failure, exactly 2 surviving IDs
   (1st + 3rd), all 3 Create calls attempted (loop never short-circuits),
   error message includes the failing tool's name.
4. `TestPermissionHandshakePostDenialsAggregatesMultipleFailures` — both
   denials fail: aggregated error wraps both via `errors.Is`, 0 surviving IDs.
5. `TestPermissionHandshakeNowDefault` — `Now` left nil → `time.Now` fallback;
   `CreatedAt` falls inside the `[before, after]` window.

`fakeAttentionStore` is a programmable mock (per-call `(item, err)` results,
defaults to `(item, nil)` when call index exceeds the configured slice). Mutex
on `received` for snapshot reads.

## Design notes / deviations from spec

- **`projectID uuid.UUID` parameter — kept verbatim from spec.** The
  domain layer (`domain.AttentionItem.ProjectID`) is a `string`. The
  handshake takes the typed `uuid.UUID` per spec, calls `.String()` once
  for the domain layer, and surfaces the string twice (as `ScopeID` since
  `ScopeType=Project`, and inside the JSON payload). This matches the F.7.17.7
  `PermissionGrant` precedent: spec-level `uuid.UUID` is a vocabulary marker;
  string IDs are the domain convention.
- **Returns `[]uuid.UUID` per spec.** Generated via `uuid.New()` per denial,
  stored in domain via `.String()`, returned as the `uuid.UUID` slice.
- **Clock injection via `PermissionHandshake.Now`.** Other dispatcher files
  use `time.Now()` directly (e.g. `gate_mage_ci.go:142`), but the attention-
  item domain ctor takes `now time.Time`, so a tiny clock seam keeps tests
  deterministic without inverting the dispatcher's existing pattern. `Now=nil`
  falls through to `time.Now`.
- **`PermissionGrantsStore` shipped as empty interface.** The spec lists it as
  a field but acceptance scopes it out of this droplet ("permission_grants
  shipped in F.7.17.7"). Empty-interface placeholder lets F.7.5c add methods
  without changing the struct shape. May be nil today.
- **Per-denial summary `"Tool permission denied: <ToolName>"`.** Inbox-glance
  surfaceable; full payload rides in `BodyMarkdown` JSON.
- **`AttentionKindApprovalRequired` chosen.** Among the existing six
  `AttentionKind` constants, `approval_required` is the closest semantic match
  for "dev approves / denies tool permission via TUI" — and it composes
  correctly with `BlocksCompletion()` so a pending denial blocks the spawn's
  parent action item from completing.
- **`CreatedByActor="tillsyn-dispatcher"` + `CreatedByType=ActorTypeAgent`.**
  Mirrors the dispatcher's existing actor labeling.

## Wiring follow-up (deferred — not this droplet)

`PermissionHandshake.PostDenials` is the consumer-facing primitive. Calling it
from the dispatcher monitor's terminal-event hook (so `[]ToolDenial` surfaced
from `adapter.ExtractTerminalReport` flows into attention items) is a
follow-up for the dispatcher orchestration layer. It MUST live in a separate
droplet to keep cross-droplet ordering concerns isolated — F.7.5b's hard
constraint "DO NOT wire into monitor.go in this droplet" is honored.

The hook site: in `monitor.go`'s terminal-event path (search for
`ExtractTerminalReport` callers), after the report is decoded, if
`len(report.Denials) > 0` then `handshake.PostDenials(...)` with the spawn's
project/kind/denials. Production wiring also requires:

1. A real `AttentionItemStore` adapter binding `Create` to the existing
   `internal/app` attention service.
2. The dispatcher constructor (e.g. `NewDispatcher` in `dispatcher.go`)
   accepting a `*PermissionHandshake` and storing it for monitor use.

Both are non-trivial cross-droplet edits and are correctly excluded from this
droplet's scope.

## Verification

- `mage testPkg ./internal/app/dispatcher/` — 281 tests pass (was 276 before
  this droplet; 5 new tests added).
- `mage ci` — full gate green:
  - 2624 tests pass, 1 skip (pre-existing
    `TestStewardIntegrationDropOrchSupersedeRejected`, unrelated).
  - Coverage threshold met across all 24 packages; dispatcher at 74.5%.
  - Build succeeds.
- Format check (`gofumpt`) clean (gate ran inside `mage ci`).

## Hard constraints honored

- DO NOT commit. Confirmed — no `git commit` invoked.
- Edits limited to: `internal/app/dispatcher/handshake.go` (NEW),
  `internal/app/dispatcher/handshake_test.go` (NEW), this worklog (NEW).
- No Hylla calls (Hylla skipped per droplet rule; reads via `Read` / `rg`).
- No edits to `monitor.go`, `dispatcher.go`, or any non-listed file.
- No `mage install`. No raw `go build` / `go test` / `go vet`.

## Hylla Feedback

N/A — task touched only Go code in `internal/app/dispatcher/` plus a new
workflow markdown; per droplet rule "NO Hylla calls," no Hylla queries were
issued. (Codebase search for `ToolDenial` and `PermissionGrant` shape used
`rg` directly per the droplet's NO-Hylla constraint, not as a Hylla
fallback.)

---

## Round 2 — Fix-builder (QA-Falsification F1 + F2)

QA-Falsification round-1 raised 2 CONFIRMED issues. Both are fixed in this
round; round-1 commits remain uncommitted on disk per F.7-CORE REV-13.

### F1 — Payload contract gap (CONFIRMED → fixed)

Round-1 payload was `{tool_name, tool_input, project_id, kind}`. F.7.5 parent
acceptance requires `{tool_name, tool_input, kind, cli_kind, action_item_id}`
because two downstream consumers depend on the missing fields:

- **F.7.5c grant injection**: `permission_grants.cli_kind` (F.7.17.7) decides
  which CLI's `settings.json` the approved-always grant lands in. Without
  `cli_kind` in the attention-item payload, F.7.5c can't route the grant —
  cross-CLI grant misuse vector reopens.
- **Deny-flow**: when the dev clicks Deny, the dispatcher's deny-handler must
  move the action item to `failed` with `metadata.failure_reason =
  "permission_denied"`. Without `action_item_id` in the payload, the deny
  handler can't locate the action item.

**Edits:**

- `handshake.go` `permissionDenialPayload` struct gained two fields:
  `CLIKind string` (JSON tag `cli_kind`) and `ActionItemID uuid.UUID` (JSON
  tag `action_item_id`). Field-roles documented inline.
- `handshake.go` `PostDenials` signature extended:
  `PostDenials(ctx, projectID, kind, cliKind string, actionItemID uuid.UUID,
  denials)`. Per-denial loop unchanged; `errors.Join` aggregation unchanged;
  empty-input no-op unchanged. Signature is the only behavioral surface that
  changed — callers now pass cliKind + actionItemID through to the payload.
- `handshake.go` `postOne` private helper got the same two parameters and
  copies them into the payload struct.

**Test edits (5 existing):**

- `TestPermissionHandshakePostDenialsEmpty` — both call sites pass
  `"claude", uuid.New()` for the new args.
- `TestPermissionHandshakePostDenialsAllSucceed` — call site passes
  `"claude", actionItemID` (new local var); per-item assertions extended to
  verify `payload.CLIKind == "claude"` and `payload.ActionItemID ==
  actionItemID`.
- `TestPermissionHandshakePostDenialsContinuesAfterFailure` — call site
  updated for new args.
- `TestPermissionHandshakePostDenialsAggregatesMultipleFailures` — call site
  updated for new args.
- `TestPermissionHandshakeNowDefault` — call site updated for new args.

**Test additions (2 new):**

- `TestPermissionHandshakePostDenialsPayloadIncludesCLIKind` — round-trips
  `cli_kind="codex"` through the BodyMarkdown JSON and asserts the field is
  present under the `cli_kind` wire-format key (defends the JSON tag
  separately from the Go field name).
- `TestPermissionHandshakePostDenialsPayloadIncludesActionItemID` — same
  shape, round-trips a generated `uuid.UUID` and asserts the
  `action_item_id` wire-format key is present.

Both new tests use the existing `fakeAttentionStore` mock to capture
`BodyMarkdown` exactly as round-1's all-success test does — same evidence
mechanism, narrower assertions.

A small test helper `keysOf(map[string]any) []string` was added at the end of
the test file so payload-key-missing failure messages enumerate available
keys deterministically.

### F2 — REV-3 citation drift (CONFIRMED → fixed)

Round-1 doc-comments at `handshake.go:27-30` and the worklog cited "REV-3 of
`workflow/drop_4c/F7_CORE_PLAN.md`." That file exists but has no REVISIONS
section. The F.7.5 split is documented in master `workflow/drop_4c/PLAN.md`
§6.1 (F.7-CORE F.7.5 split policy).

**Edits:**

- `handshake.go` doc-comment block above `AttentionItemStore` rewritten:
  `REVISIONS-first reference: workflow/drop_4c/F7_CORE_PLAN.md REV-3 ...` →
  `PLAN reference: workflow/drop_4c/PLAN.md §6.1 / F.7-CORE F.7.5 split
  policy ...`. Same content, accurate citation.
- This worklog's top-of-file `REVISIONS-first:` line rewritten to point at
  PLAN.md, with a parenthetical retraction noting round-1's bad citation.

### Acceptance-criterion mapping

- [x] `PostDenials` signature extended with `cliKind string, actionItemID uuid.UUID`.
- [x] Per-denial payload struct includes `CLIKind` + `ActionItemID` fields with JSON tags `cli_kind` + `action_item_id`.
- [x] All 5 existing tests pass with new signature.
- [x] 2 new tests assert payload includes `cli_kind` + `action_item_id`.
- [x] Doc-comments + worklog cite PLAN.md (NOT REV-3 of F7_CORE_PLAN.md).
- [x] `mage check` + `mage ci` green (2644/2645 pass; 1 pre-existing skip;
      dispatcher coverage 75.1%).
- [x] **NO commit by the builder.**

### Verification (round 2)

- `mage check` — green. 2644 tests pass, 1 pre-existing skip; 24 packages all
  ≥ 70% coverage; dispatcher at 75.1%; `till` builds cleanly.
- `mage ci` — green. Same 2644/1 figures; format check clean; full coverage
  gate met; build succeeds.

### Hard constraints honored (round 2)

- DO NOT commit — confirmed, no `git commit` invoked.
- Edits limited to: `internal/app/dispatcher/handshake.go`,
  `internal/app/dispatcher/handshake_test.go`, this worklog. No other files
  touched.
- No Hylla calls.
- No `mage install`. No raw `go build` / `go test` / `go vet`.
- Core algorithm unchanged — `PostDenials` per-denial loop, `errors.Join`
  aggregation, AttentionItem creation untouched. Only payload struct +
  signature widened.

### Suggested commit message (orchestrator drives)

```
feat(dispatcher): widen handshake payload with cli_kind + action_item_id
```
