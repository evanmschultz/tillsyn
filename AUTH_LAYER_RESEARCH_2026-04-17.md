# Auth Layer Research — 2026-04-17

Research deliverable investigating two connected auth-layer bugs surfaced by the DROP_1.5_ORCH bootstrap attempt and the dev's orch-approves-subagent regression report. Read-only research; no code, schema, or Tillsyn mutations performed.

- **Artifact ref**: `github.com/evanmschultz/tillsyn@main`, HEAD commit `66c354e` (`refactor(all): fix bad module name`).
- **Requester**: STEWARD (persistent MD-writing orchestrator).
- **Scope**: domain, app, and adapter layers at `internal/domain/auth_request.go`, `internal/domain/capability.go`, `internal/app/auth_requests.go`, `internal/app/kind_capability.go`, `internal/adapters/auth/autentauth/service.go`, `internal/adapters/server/mcpapi/handler.go`, `internal/adapters/server/common/app_service_adapter_mcp.go`, `cmd/till/main.go`.

---

## 1. Summary

Two distinct but mutually reinforcing defects.

**Bug A — Drop-collapse parity gap**: The auth path grammar at `internal/domain/auth_request.go:ParseAuthRequestPath` (lines 138–186) hardcodes the pair vocabulary `project | branch | phase`; `task`, `subtask`, and `drop` are rejected by the `switch segment` default case. Pre-Drop-2, every level_1 Tillsyn drop is created with `kind='task', scope='task'`, so its path must be written as `project/<id>/branch/<drop-id>` to fit the grammar — the grammar leaks a pre-existing vocabulary out of sync with the cascade plan's drop model.

The lease side is subtler. `CapabilityScopeType` (`internal/domain/capability.go:61–73`) does enumerate all five values (`project|branch|phase|task|subtask`) and `IsValidCapabilityScopeType` accepts all of them. But the app-layer gate `Service.validateCapabilityScopeTuple` at `internal/app/kind_capability.go:377` resolves the concrete task row from `repo.GetTask(scopeID)` and derives the **canonical** capability scope for that task from `task.Scope` via `capabilityScopeTypeForTask` (same file, 409–423). Pre-Drop-2 the drop's `task.Scope == KindAppliesToTask`, so the canonical scope is `task`. A request for `scope_type: branch` on a task-scoped row fails the `if taskScopeType != scopeType` check at line 401 with `ErrInvalidCapabilityScope`. That is the "invalid scope type" runtime error the memory documented for branch/phase/subtask on task rows — it is not a schema rejection, it is a **canonical-scope mismatch between the path vocabulary and the task's stored scope**.

Net runtime behavior observed by DROP_1.5_ORCH: session created at `project/<id>/branch/<drop-id>` (forced by the grammar), lease then rejects `scope_type: branch` (canonical-scope mismatch) AND `scope_type: task` lease auth-denies against a branch-scoped session (the approved path normalizes to `ScopeLevelBranch` per `AuthRequestPath.Normalize` lines 210–274, while the lease wants `ScopeLevelTask` — `CapabilityLease.MatchesScope` at `internal/domain/capability.go:347–361` accepts any scope only when `l.ScopeType == CapabilityScopeProject`, so the mismatch survives lease matching). Drop-scoped auth is therefore non-operational end-to-end pre-Drop-2. Project-scope is the only lane that currently works.

**Bug B — Orch-approves-subagent regression**: Not a code regression in the strict sense. The MCP surface `till.auth_request` tool (`internal/adapters/server/mcpapi/handler.go:101–107`) enumerates operations `create | list | get | claim | cancel | list_sessions | validate_session | check_session_governance | revoke_session`. There is **no `approve` operation on the MCP tool at all**. The only approve path today is the CLI (`cmd/till/main.go:1702–1717`, `till auth request approve`) and the TUI, both of which are dev-human-driven. The app-layer `Service.ApproveAuthRequest` (`internal/app/auth_requests.go:270–300`) and the autent-adapter `Service.ApproveAuthRequest` (`internal/adapters/auth/autentauth/service.go:422–472`) both take a `ResolvedBy` string input with **no caller-identity capability gate** — meaning the restriction is not "orchestrator-lacks-capability," it is "no MCP surface to call approve through." An orchestrator with `CapabilityActionApproveAuthWithinBounds` (granted by default in `DefaultCapabilityActions`, `internal/domain/capability.go:221–280`) has nowhere to spend that capability programmatically.

The dev quote ("that used to work and got broken") most likely refers either to (a) the earlier dogfood flow at `9cd4df8 feat(auth): add dogfood auth request workflow` before the governance hardening, or (b) an expectation that the "acting_session_id" create delegation (commit `d502ea6 feat(auth): enforce bounded delegation policy`, Mar 21 2026) would also imply an approve-delegation pair. It did not — only the create seam was wired to `acting_session_id`. Approval was left exclusively on the dev-human lane.

---

## 2. Auth Path Grammar (Bug A, Request Side)

### 2.1 Parser (`internal/domain/auth_request.go:138–186`)

```
parts := strings.Split(raw, "/")           // must be length >= 2 and even
if parts[0] != "project" … error           // first pair must be "project/<id>"
for idx := 2; idx < len(parts); idx += 2 {
    switch segment {
    case "branch":                         // once only, before any phase
        path.BranchID = value
        seenBranch = true
    case "phase":                          // requires seenBranch; repeatable
        path.PhaseIDs = append(...)
    default:                               // ← any other segment keyword fails
        return AuthRequestPath{}, ErrInvalidAuthRequestPath
    }
}
```

Only four kinds exist in the domain vocabulary: `AuthRequestPathKindGlobal`, `AuthRequestPathKindProjects`, `AuthRequestPathKindProject`, and implicit-leaf levels (`branch`, `phase`) that still normalize into one of those kinds. There is no `AuthRequestPathKind` for `task`, `drop`, or `subtask`; the grammar has no way to parse them; the kind constants at lines 47–49 confirm this:

```go
AuthRequestPathKindGlobal   AuthRequestPathKind = "global"
AuthRequestPathKindProjects AuthRequestPathKind = "projects"
AuthRequestPathKindProject  AuthRequestPathKind = "project"
```

### 2.2 Normalize (same file, lines 210–274)

`AuthRequestPath.Normalize` derives `ScopeType`:

- `len(PhaseIDs) > 0` → `ScopeType = ScopeLevelPhase`
- else `BranchID != ""` → `ScopeType = ScopeLevelBranch`
- else → `ScopeType = ScopeLevelProject`

So even though `ScopeLevel` has five values (`project | branch | phase | task | subtask` at `internal/domain/level.go:13–18`), the path parser can only produce three of them: `project`, `branch`, `phase`. `ScopeLevelTask` and `ScopeLevelSubtask` are never produced by `ParseAuthRequestPath`.

### 2.3 NewAuthRequest Role Check (same file, lines 371–405)

```go
if path.Kind != AuthRequestPathKindProject && principalRole != string(AuthRequestRoleOrchestrator) {
    return AuthRequest{}, ErrInvalidAuthRequestRole
}
```

All paths under `project/...` are `Kind == AuthRequestPathKindProject` (including `branch` and `phase` subpaths), so this check allows non-orch principals only when the path is rooted at a project, which is always true for per-project workflows. This gate is NOT the blocker.

### 2.4 Empirical DROP_1.5 path `project/<id>/branch/<drop-id>`

The pre-Drop-2 memory `feedback_auth_path_branch_quirk.md` records that `/task/<id>` and `/drop/<id>` are rejected; only `/branch/<drop-id>` is accepted. That is a literal consequence of §2.1. The `<drop-id>` here is a pre-Drop-2 level_1 drop, which is a `task` row in the DB — the grammar forces the task id to live in a `branch/...` slot regardless of what it really is.

---

## 3. Lease Validator (Bug A, Mutation Side)

### 3.1 Domain Schema — Accepts All Five

`internal/domain/capability.go:61–73`:

```go
const (
    CapabilityScopeProject CapabilityScopeType = "project"
    CapabilityScopeBranch  CapabilityScopeType = "branch"
    CapabilityScopePhase   CapabilityScopeType = "phase"
    CapabilityScopeTask    CapabilityScopeType = "task"
    CapabilityScopeSubtask CapabilityScopeType = "subtask"
)
```

`IsValidCapabilityScopeType` is a simple set membership check against `validCapabilityScopes`, which holds all five. `domain.NewCapabilityLease` (lines 125–179) validates `scope_type` against that set and accepts any of the five.

### 3.2 App-Layer Canonical-Scope Gate — Collapses to Task Row Scope

`internal/app/kind_capability.go:377–406`:

```go
func (s *Service) validateCapabilityScopeTuple(ctx, projectID, scopeType, scopeID) (string, error) {
    switch scopeType {
    case domain.CapabilityScopeProject:
        … project happy path
    default:
        task, err := s.repo.GetTask(ctx, scopeID)
        …
        if taskScopeType := capabilityScopeTypeForTask(task); taskScopeType != scopeType {
            return "", domain.ErrInvalidCapabilityScope          // ← the "invalid scope type" runtime error
        }
        return scopeID, nil
    }
}
```

`capabilityScopeTypeForTask` (lines 409–423) maps `task.Scope` (a `KindAppliesTo` value) to the canonical capability scope:

```go
case domain.KindAppliesToProject:  return domain.CapabilityScopeProject
case domain.KindAppliesToBranch:   return domain.CapabilityScopeBranch
case domain.KindAppliesToPhase:    return domain.CapabilityScopePhase
case domain.KindAppliesToSubtask:  return domain.CapabilityScopeSubtask
default:                           return domain.CapabilityScopeTask
```

Pre-Drop-2 every drop is created with `scope=task` (`KindAppliesToTask`), so `capabilityScopeTypeForTask` returns `CapabilityScopeTask` for every drop row. A caller requesting `scope_type: branch` with a drop id passes §3.1 schema validation but fails §3.2 canonical-scope validation with `ErrInvalidCapabilityScope`.

### 3.3 MCP Adapter Passes Through

`internal/adapters/server/common/app_service_adapter_mcp.go:1633–1653`:

```go
func (a *AppServiceAdapter) IssueCapabilityLease(ctx, in IssueCapabilityLeaseRequest) (…, error) {
    lease, err := a.service.IssueCapabilityLease(ctx, app.IssueCapabilityLeaseInput{
        ScopeType:                 domain.CapabilityScopeType(strings.TrimSpace(in.ScopeType)),
        …
    })
```

The MCP layer does no pre-validation of `scope_type` — it hands the raw string to `app.IssueCapabilityLease`, which eventually calls `validateCapabilityScopeTuple`. So the "invalid scope type" runtime error originates in the app layer at §3.2, not the adapter.

### 3.4 Session-to-Lease Scope Match — The Second Rejection

Even with `scope_type: task` (passing §3.2 for a drop row), the lease must be authorized against the session's approved path. `CapabilityLease.MatchesScope` at `internal/domain/capability.go:347–361`:

```go
if l.ScopeType == CapabilityScopeProject {
    return true                                    // project-scope leases match anything
}
if l.ScopeType != scopeType {
    return false                                   // otherwise must match exactly
}
```

The session approved path normalizes to `ScopeLevelBranch` (forced by grammar §2), which maps to `CapabilityScopeBranch` via `ScopeLevel.ToCapabilityScopeType` in `internal/domain/level.go`. A `task`-scope lease issue against that session fails the scope match. This is the second rejection stacking on top of §3.2 — both originate in the grammar/scope vocabulary collision.

### 3.5 Net Runtime Behavior for a Pre-Drop-2 Drop

| `path` used on create  | `scope_type` used on issue  | Outcome                                               |
|------------------------|------------------------------|-------------------------------------------------------|
| `project/<id>/branch/<drop-id>` | `branch`                   | FAIL canonical-scope check (task row maps to task)   |
| `project/<id>/branch/<drop-id>` | `task`                     | PASS canonical-scope; FAIL session-path match        |
| `project/<id>/branch/<drop-id>` | `phase`                    | FAIL canonical-scope                                  |
| `project/<id>/branch/<drop-id>` | `subtask`                  | FAIL canonical-scope                                  |
| `project/<id>/branch/<drop-id>` | `project`                  | FAIL canonical-scope (scopeID != projectID at line 383) |
| `project/<id>` (project scope) | `project`                  | PASS — the only working lane                          |

Drop-scoped auth is non-operational end-to-end pre-Drop-2. Dev has to fall back to project-scope auth (STEWARD's baseline pattern).

---

## 4. Orch-Approves-Subagent (Bug B)

### 4.1 MCP Surface Does Not Expose `approve`

`internal/adapters/server/mcpapi/handler.go:101–107`:

```go
mcp.WithString("operation", mcp.Required(), …, mcp.Enum(
    "create", "list", "get", "claim", "cancel",
    "list_sessions", "validate_session", "check_session_governance", "revoke_session",
)),
```

Nine operations. `approve`, `deny`, and `cancel`-as-resolver are not present. An MCP client cannot call approve at all.

### 4.2 App-Layer `ApproveAuthRequest` Has No Caller-Identity Gate

`internal/app/auth_requests.go:270–300`:

```go
func (s *Service) ApproveAuthRequest(ctx context.Context, in ApproveAuthRequestInput) (…, error) {
    …
    ctx, _, _ = withResolvedMutationActor(ctx, in.ResolvedBy, "", in.ResolvedType)
    resolvedBy, resolvedType := resolvedAuthRequestActor(ctx, in.ResolvedBy, in.ResolvedType)
    …
    out, err := s.authRequests.ApproveAuthRequest(ctx, ApproveAuthRequestGatewayInput{
        RequestID:      strings.TrimSpace(in.RequestID),
        ResolvedBy:     resolvedBy,
        ResolvedType:   resolvedType,
        …
    })
    …
}
```

No check that `resolvedBy` has `CapabilityActionApproveAuthWithinBounds`. No check that the resolver's approved path covers the request's path. No session/lease validation. The function trusts its caller. That is correct for the dev-human TUI/CLI path (where the CLI operator is implicitly root-authorized by possession of the local DB) but it means there is no machinery today that could extend to orchestrator-driven approval without adding one.

### 4.3 Autent-Adapter Approve Is Also Identity-Free

`internal/adapters/auth/autentauth/service.go:422–472`:

```go
func (s *Service) ApproveAuthRequest(ctx, in app.ApproveAuthRequestGatewayInput) (app.ApprovedAuthRequestResult, error) {
    …
    req, err := s.GetAuthRequest(ctx, in.RequestID)
    …
    issued, err := s.IssueSession(ctx, IssueSessionInput{…})
    …
    if err := req.Approve(strings.TrimSpace(in.ResolvedBy), in.ResolvedType, …); err != nil { … }
    …
}
```

Same shape — takes `ResolvedBy` at face value, issues the session, marks the request approved.

### 4.4 Capability `ApproveAuthWithinBounds` Exists But Is Orphan

`internal/domain/capability.go:19` defines `CapabilityActionApproveAuthWithinBounds`; `DefaultCapabilityActions` at lines 221–280 grants it to `CapabilityRoleOrchestrator` by default. No caller of `CanPerform(ApproveAuthWithinBounds)` exists in the approval path — I grepped `CanPerform` against the approval codepaths and found it used only for mutation guard on plan-item, handoff, comment, etc. The capability is declared but unused on the approve seam.

### 4.5 Related But Not The Same: `acting_session_id` Delegation On CREATE

Commit `d502ea6 feat(auth): enforce bounded delegation policy` (Mar 21 2026) added `acting_session_id` to `CreateAuthRequest` so an orchestrator can create a child-principal request delegated under its own path (`internal/adapters/server/common/app_service_adapter_mcp.go:440–457`). This is the "bounded delegation" path the dev likely remembers. It is only the CREATE seam — approval still goes to the dev. The path-within-acting-path check (`authRequestPathWithin` at lines 495–547) ensures the child request's path is a subset of the orchestrator's, so a child request could in principle be auto-approvable because its scope is provably within the acting orch's approved scope. But no current code performs that auto-approval.

### 4.6 Git History Does Not Show A Prior Working Orch-Approves Path

Searching commits that touched `ApproveAuthRequest`:

- `ba9339e feat(auth): wire autent session-first dogfood runtime` — introduced the approve seam.
- `9cd4df8 feat(auth): add dogfood auth request workflow` — added the CLI/TUI approve commands.
- `3fdfda2 feat(auth): add claim resume and visible review controls` — added claim + review.
- `6de3798 feat(auth): harden dogfood gatekeeping` — added gate checks.
- `d502ea6 feat(auth): enforce bounded delegation policy` — added `acting_session_id` on CREATE.
- `3d7760b fix(auth): enforce child-only delegated claims` — locked CLAIM (not APPROVE) to child-only.

No commit in this history ever wired an MCP-callable or orchestrator-capability-gated approve. The "used to work" intuition is most likely the `acting_session_id` delegation shape plus the latent `CapabilityActionApproveAuthWithinBounds` intent, which together imply an approve delegation that the code does not actually implement.

---

## 5. Fix Options

Four orthogonal options to address the two bugs. A and B are Bug A (grammar parity); C and D are Bug B (orch approve).

### 5.1 Option A — Extend Grammar to Include `task` and `subtask`

**Scope**: grammar + normalize + app validator.

- Add `AuthRequestPathKindTask`, `AuthRequestPathKindSubtask` or extend the segment switch in `ParseAuthRequestPath` to accept `task` and `subtask` segments.
- Add `ScopeLevelTask` and `ScopeLevelSubtask` production in `Normalize` (the constants already exist at `internal/domain/level.go`).
- Update `authRequestPathWithin` to compare task/subtask chains.
- Update `validateCapabilityScopeTuple` to stop rejecting when the session's normalized scope matches the task's canonical scope.

**Pros**: narrow, domain-local fix that makes the grammar symmetrical with `ScopeLevel` / `CapabilityScopeType`. Does not change the Tillsyn data model. Makes the auth layer honest about what scopes exist.

**Cons**: leaves two vocabularies in place (`branch/phase/task/subtask` path segments AND a canonical scope-tree). Drop 2 collapses every non-project kind to `drop` and puts role on `metadata.role` — the grammar will need another pass shortly to support `/drop/<id>` uniformly. Risk of double churn.

### 5.2 Option B — Wait for Drop 2, Collapse Grammar to `drop`

**Scope**: Drop 2 collapse SQL plus a grammar rewrite to `project/<id>[/drop/<id>]*`.

- Drop 2 (`main/PLAN.md` §19.2) rewrites every non-project `kind` to `drop`, hydrates `metadata.role`, and can also drop the `KindAppliesTo*` enum or collapse it.
- Replace `ParseAuthRequestPath` branch/phase segments with a single repeating `drop/<id>` pair.
- Replace `ScopeLevel` variants with `project | drop` (or keep them as a stratification the DB no longer distinguishes).

**Pros**: unifies the vocabulary in one pass. Aligns auth with the cascade plan's drop model (one tree, infinite nesting). Eliminates the canonical-scope collision at the root.

**Cons**: large blast radius across domain, adapters, TUI, and the CLI. Can't ship before Drop 2 Go ships. Needs MD additions to WIKI.md on grammar change and TUI display.

### 5.3 Option C — Add `approve` Operation To MCP `till.auth_request`

**Scope**: MCP handler + app service approve gate.

- Add `approve` (and by symmetry `deny`) to the MCP enum at `internal/adapters/server/mcpapi/handler.go:102`.
- Add `session_id`, `session_secret`, `auth_context_id`, `lease_token`, `agent_instance_id` session-tuple inputs to the approve handler (same shape as other guarded MCP ops).
- In `app.Service.ApproveAuthRequest`, gate on: (a) session's principal has `CapabilityActionApproveAuthWithinBounds`, (b) session's approved path contains the request's path via `authRequestPathWithin`, (c) request's principal_role is non-orchestrator (never allow orch approving orch — dev is the only lane for peer orchs), (d) request's acting_session_id points to the approver's session (or the same subtree).
- Write an audit row with approver session_id + resolved request_id for every non-dev approval.

**Pros**: implements §19.1.6 (drop 1.6 — Auth Approval Cascade) directly. Re-uses `authRequestPathWithin` which already does subtree containment. Matches the latent capability design in `DefaultCapabilityActions`.

**Cons**: must decide on no-configurability policy up front (§19.1.6 says STEWARD can always approve within its subtree; drop orchs can approve within their branch). MCP test coverage needs to grow by several shapes.

### 5.4 Option D — Auto-Approve Bounded Delegated Create

**Scope**: `Service.CreateAuthRequest` auto-approve when `acting_session_id` is present and the request's path is within the acting session's approved path.

- On CREATE, if `acting_session_id` is set, the request's `path` is within the acting session's `ApprovedPath` via `authRequestPathWithin`, and the acting principal has `CapabilityActionApproveAuthWithinBounds`, then inline-approve the request (issue the session immediately) and return `state: approved` with `request_id` + `resume_token` + `session_secret` in one round-trip.
- Skip the two-step approve flow entirely for the bounded-delegation case.

**Pros**: smallest surface change; no new MCP op. Faster (one round trip). Preserves the current two-step approve path for the dev's TUI-human oversight of orch auth.

**Cons**: conflates CREATE and APPROVE semantics. Hides the approval decision inside a create call — harder to audit. Makes it non-obvious in the TUI which requests were auto-approved vs dev-approved. Race condition risk with other approval paths.

---

## 6. Plan Alignment

Against `main/PLAN.md` §19.1.6 (drop 1.6 — Auth Approval Cascade):

| PLAN directive                                                                                  | Option alignment                     |
|-------------------------------------------------------------------------------------------------|--------------------------------------|
| "Orchs may approve non-orch subagent auth requests scoped within their own subtree."           | Option C (direct). Option D (implicit, weaker audit). |
| "STEWARD cross-subtree exception."                                                              | Requires additional allowlist in Option C / D.       |
| "No configurability this drop."                                                                 | Option C defaults; Option D has no knob.             |
| "Project-scope opt-out switch."                                                                 | Option C adds a project metadata flag.               |
| "Audit trail."                                                                                  | Option C natural fit; Option D has to embed in create result. |
| "MCP-layer test coverage."                                                                      | Option C high coverage; Option D mostly existing CREATE coverage. |
| "Prompt updates (STEWARD / drop-orch prompts)."                                                | Independent — same for both.         |

Against §19.2 (Drop 2 — kind collapse + task/drop rename):

- Option B is effectively Drop 2 carrying an auth-grammar change rider. §19.2 already intends to collapse kinds; expanding that to include grammar collapse is a natural extension.
- Option A conflicts with §19.2 — shipping `task`/`subtask` grammar now just to rewrite them to `drop` later is churn.

Against `main/WIKI.md` "Auth Approval Cascade":

- Dev approves orchestrator auth only; orchs approve non-orch subagent auth scoped to subtree; dev never sees subagent auth for STEWARD-owned or drop-orch-owned subtrees. Option C honors this cleanly. Option D requires the CLI/TUI to filter auto-approved rows to avoid clutter (already implicit: they'd show as `state: approved` on create).

---

## 7. Recommendation

**Bug A**: **Option B** — collapse the grammar during Drop 2. Pre-Drop-2 accept the "branch-quirk" workaround (memory `feedback_auth_path_branch_quirk.md`) and route drop-scoped work through project-scope auth until Drop 2 lands. Trying to patch the grammar now means shipping parser code that Drop 2 will throw away in a month.

**Bug B**: **Option C** — add the `approve` operation to the MCP `till.auth_request` tool, gated on `CapabilityActionApproveAuthWithinBounds` + `authRequestPathWithin`. This implements §19.1.6 directly, reuses the existing path-containment primitive, gives a clean audit row shape, and keeps CREATE and APPROVE semantically distinct. Schedule it as PLAN §19.1.6 indicates — between Drop 1.5 and Drop 2.

**Sequencing**: Drop 1.5 (STEWARD-self) continues on project-scope auth; Drop 1.6 ships Option C; Drop 2 ships Option B. During Drop 1.6, Option D is explicitly considered and rejected for the auditability reasons in §5.4 "Cons."

**Alternative if Option C is too expensive for Drop 1.6**: ship Option D as a stopgap, mark it as transitional in PLAN §19.1.6, and replace it with Option C during Drop 2 when the grammar is also changing. This is strictly worse on audit quality but is strictly smaller in code surface.

---

## 8. Unknowns

- **Q1**: Does the TUI auth-request review surface explicit `actor.capability.approve_auth_within_bounds` decisions today, or does it present the row to the dev unconditionally? If the dev-only gate is in the TUI (not in the service), then Option C must also add a "hide this from dev" predicate so STEWARD-approved requests do not still ask the dev. Not yet verified — requires reading `internal/tui/auth_request_*.go`. Routed to this MD.
- **Q2**: Does `authSessionWithinApprovedPath` (at `internal/adapters/server/common/app_service_adapter_mcp.go:476–547`) cover the subtree-containment semantics PLAN §19.1.6 expects, or does it need a drop-tree variant that walks Tillsyn parent chains? The current implementation compares `ProjectID`, `BranchID`, and `PhaseIDs` directly — it does not traverse Tillsyn's parent_id graph. A drop orch whose approved path is `project/<id>/branch/<drop-1-id>` cannot validate that a leaf `task` under a nested sub-drop is "within" the outer drop via this function alone. Drop 2's grammar collapse will make this worse before it makes it better.
- **Q3**: The dev's "used to work" intuition — did an earlier commit actually have MCP-callable approve before the bounded-delegation refactor, and did it get removed during `d502ea6`? Not found in git log. Confidence is "no such commit exists in current main history" but pre-main / squashed history could hide it. Acceptable unknown for this deliverable.
- **Q4**: For Option C, does the `override_token` path on `till_capability_lease` (for equal-scope orch overlap in STEWARD bootstrap) need a parallel `override_token` on approve for STEWARD cross-subtree exceptions? Or should cross-subtree exceptions be encoded as a separate capability action (e.g. `ApproveAuthAnyProjectSubtree`)? Design decision, not a research finding.
- **Q5**: Does Option B (grammar collapse) need a data migration for already-persisted auth requests whose `path` field is a raw string? `auth_requests.path` is stored as text; old rows would still contain `project/<id>/branch/<id>` paths that a new parser might reject. Needs a migration plan before Drop 2 ships.

---

## 9. Hylla Feedback

- **Query**: `hylla_search` was not attempted during this research session. Bypassed in favor of `Grep` + `Read` because:
  1. Artifact ref `@main` snapshot 5 (commit `66c354e`) is the current ingest baseline per `main/CLAUDE.md` Hylla Baseline, but the investigation needed tight structural navigation across ~8 files that the grep-anchor-plus-read pattern handles more directly than vector search.
  2. The investigation is grammar/switch-statement shaped, not semantic — "find every call to X" and "show me this function body" are keyword/lsp tasks, not vector-embedding tasks.
- **Missed because**: nothing — the research did not hit a dead end that Hylla would have unblocked. This is an "I should have tried Hylla first" note rather than a miss report.
- **Worked via**: `Grep` + `Read`.
- **Suggestion**: add a Hylla skill for "grammar / state machine / switch statement fanout" that specifically surfaces case-branches under a named switch across a package. Current vector search returns surrounding docstrings; this kind of investigation benefits from switch-case enumeration directly. One-liner: `hylla_switch_fanout(symbol=ParseAuthRequestPath, scope=package)` returning the set of case-segment strings and their target actions.

No other Hylla ergonomics gripes this session.
