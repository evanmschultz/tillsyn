# W7.D1 — `internal/adapters/server/` Inventory

**Produced by**: Drop 4c.6.1, Wave W7, Droplet D1 (INVENTORY: pure-read audit).
**Date**: 2026-05-12.
**Branch baseline**: `main` HEAD `9baff90` (`docs: update gitignore`) + uncommitted W4-wave agent-template rename activity unrelated to this droplet.
**`mage ci` status**: pre-existing build error in `internal/tui/style/` (a NEW directory introduced by W4-wave uncommitted work; not within W7 scope and not introduced by D1). D1 wrote zero Go files; D1 makes no contribution to `mage ci` delta.

This artifact is **load-bearing** for W7.D2 (extract-everything-not-HTTP) and W7.D3 (delete HTTP residue). Both downstream builders read this file as their primary input and do NOT re-enumerate.

---

## 0. Classification Taxonomy

Per PLAN.md L1 W7.D1 + REVISION_BRIEF §2.16 (round-3 inverted carving):

- **`http-residue`** — HTTP server transport / handler / wire-protocol-specific code. Stays in `internal/adapters/server/` until W7.D3 deletion.
- **`stdio-relevant`** — stdio MCP transport code. Extracts to `internal/adapters/mcp_stdio/` in W7.D2.
- **`transport-neutral`** — shared scaffolding (Service adapter, auth helpers, MCP RPC tool registry, MCP types). Extracts to `internal/adapters/mcp_common/` (Service adapter + auth + types + capture) **or** `internal/adapters/mcp_rpc/` (the current `mcpapi/` MCP tool-registry engine) per the per-file assignment below.

**Disambiguation rule** (per PLAN.md ContextBlocks constraint): a symbol with BOTH HTTP consumers AND stdio consumers belongs in `transport-neutral` (extract), NOT `http-residue` (delete).

**Package-level vs file-level**: most packages classify cleanly at the package level. The two exceptions are called out explicitly per file inside the relevant package.

---

## 1. File Inventory

### 1.1 Top-level `internal/adapters/server/` (1 file)

#### `internal/adapters/server/server.go` — **MIXED** (split per function)

Package docstring (line 1): *"Package server composes HTTP API and MCP transports into one process handler."* This file composes the three sub-packages into one root mux + provides the stdio entry point. The deletion-vs-extract split is **at the function level inside this file**, not the file as a whole.

| Symbol (line) | Kind | Classification | Notes |
|---|---|---|---|
| `defaultBindAddress` (L18) | const | `http-residue` | HTTP `127.0.0.1:5437` bind default. |
| `defaultShutdownTimeout` (L21) | const | `http-residue` | HTTP server graceful-shutdown timeout. |
| `Config` (L24) | type | **`transport-neutral`** | Used by `serveradapter.Config` in `cmd/till/main.go` for BOTH `till serve` (HTTP) AND `till mcp` (stdio). Fields like `ServerName`, `ServerVersion`, `MCPEndpoint`, `Expose*` are stdio-relevant too. `HTTPBind` + `APIEndpoint` are HTTP-only. W7.D2 may either (a) keep the unified `Config` in `mcp_common/` with HTTP-only fields trimmed during W7.D3, or (b) split into `mcp_stdio.Config` + drop HTTP-only fields entirely. Recommendation: extract whole struct to `mcp_common/` (or `mcp_rpc/`) and trim HTTP fields in W7.D3. |
| `Dependencies` (L37) | type | **`transport-neutral`** | Carries `common.CaptureStateReader` + `common.AttentionService`. Both transports require both. |
| `NewHandler(cfg, deps) (http.Handler, Config, error)` (L43) | func | `http-residue` | Builds the root `http.ServeMux` with `/healthz`, `/readyz`, MCP endpoint, API endpoint. Pure HTTP. |
| `Run(ctx, cfg, deps) error` (L80) | func | `http-residue` | Boots `http.Server.ListenAndServe()` + graceful shutdown. The `till serve` engine. |
| **`RunStdio(ctx, cfg, deps) error`** (L122) | func | **`stdio-relevant`** | The `till mcp` engine. Delegates to `mcpapi.ServeStdio(...)`. **THIS IS THE LOAD-BEARING SYMBOL** the R3-FF1 disposition flagged. |
| `normalizeConfig(cfg) (Config, error)` (L153) | func | **`transport-neutral`** | Used by both `Run` (HTTP) and `RunStdio` (stdio). Trim/default logic for `Config`. |
| `normalizeEndpoint(path, fallback) string` (L177) | func | **`transport-neutral`** | Called by `normalizeConfig`. Used by both transports. |
| `writeHealthStatus(w, r)` (L193) | func | `http-residue` | `/healthz` + `/readyz` HTTP handler. |

**W7.D2 plan for `server.go`**: split into two files in new homes. `RunStdio` + the slice of `normalizeConfig`/`normalizeEndpoint` it requires + the stdio-relevant fields of `Config` + `Dependencies` extract to `internal/adapters/mcp_stdio/` (or live inside `mcp_rpc/` alongside `ServeStdio`). `Run` / `NewHandler` / `writeHealthStatus` / the HTTP-side defaults stay until W7.D3 deletion.

---

### 1.2 `internal/adapters/server/common/` (5 production files, 6 test files)

**Package classification**: **`transport-neutral`** in full. The package docstring (`types.go` line 1) says *"Package common provides transport-agnostic server contracts used by HTTP and MCP adapters."* — explicitly named for cross-transport reuse. W7.D2 destination: `internal/adapters/mcp_common/` (proposed package name `mcpcommon`).

Used by BOTH `httpapi/` AND `mcpapi/`. Also imported by `cmd/till/main.go` + `cmd/till/main_test.go` (consumer map §2 below) — the `till capture-state` subcommand and auth-mutation tests depend directly on this package, which is why R2-FF1 (round 2) discovered the original 2-step refactor was insufficient.

| File | Line count guidance | Exported symbols (top-level) | Notes |
|---|---|---|---|
| `auth.go` | small (~45 lines) | `ErrSessionRequired`, `ErrInvalidAuthentication`, `ErrSessionExpired`, `ErrAuthorizationDenied` (aliases `domain.ErrAuthorizationDenied`), `ErrGrantRequired`; types `MutationAuthorizationRequest`, `MutationAuthorizer` | **All `transport-neutral`.** Auth errors + the `MutationAuthorizer` interface are consumed by both `httpapi/handler.go` and `mcpapi/extended_tools.go` (and by `cmd/till/main_test.go` for direct auth-flow assertions). |
| `capture.go` | ~370 lines | `CaptureStateService`, `NewCaptureStateService(...)`, methods `CaptureState(...)`, `buildAttentionOverview(...)`. Unexported helpers include `normalizeCaptureStateRequest`, `findProjectByID`, `sortColumns`, `sortActionItems`, `buildWorkOverview`, `buildWarningsOverview`, `canonicalLifecycleState`, `buildCommentOverview`, `compareAttentionItems`, `computeStateHash`. | **All `transport-neutral`.** Builds the `CaptureState` payload returned by both `till capture-state` (direct call in `cmd/till/main.go` line 2763–2764) and the MCP `till.capture_state` tool. |
| `types.go` | ~232 lines | `ScopeTypeProject`, `ScopeTypeBranch`, `ScopeTypePhase`, `ScopeTypeActionItem`, `ScopeTypeSubtask`; `SupportedScopeTypes()`; `AttentionStateOpen`, `AttentionStateAcknowledged`, `AttentionStateResolved`; `ErrInvalidCaptureStateRequest`, `ErrUnsupportedScope`, `ErrAttentionUnavailable`, `ErrNotFound`; types `CaptureStateRequest`, `ScopeNode`, `GoalOverview`, `AttentionItem`, `AttentionOverview`, `WorkOverview`, `CommentOverview`, `WarningsOverview`, `ResumeHint`, `CaptureState`, `CaptureStateReader`, `CaptureStateReadModel`, `ListAttentionItemsRequest`, `RaiseAttentionItemRequest`, `ResolveAttentionItemRequest`, `AttentionService` | **All `transport-neutral`.** Wire-format-neutral data types. JSON tags are protocol-neutral (MCP returns same shapes). |
| `app_service_adapter.go` | ~620 lines | `AppServiceAdapter`, `NewAppServiceAdapter(service, auth)`; methods `AuthorizeMutation`, `CaptureState`, `ListAttentionItems`, `RaiseAttentionItem`, `ResolveAttentionItem`. Unexported helpers: `now`, `lookupProject`, `normalizeAttentionListRequest`, `normalizeRaiseAttentionItemRequest`, `normalizeResolveAttentionItemRequest`, `cloneStringMap`, `normalizeScopeTuple`, `normalizeAttentionStateFilter`, `convertCaptureStateSummary`, `buildCommentOverview`, `summarizeCommentOverview`, `isImportantCommentMarkdown`, `buildScopePathFromLevel`, `buildResumeHintsFromFollowUps`, `mapDomainAttentionItems`, `mapDomainAttentionItem`, `computeCaptureSummaryHash`, `mapAppError` | **All `transport-neutral`.** The **central** Service-adapter type. Direct consumers: `cmd/till/main.go` (lines 2653, 2682, 2763) — composes the adapter for `till serve`, `till mcp`, AND `till capture-state` subcommands. Also `cmd/till/main_test.go` (lines 129, 143, 1350, 1380, etc.) for auth-flow tests. Also `httpapi/handler_integration_test.go` and `mcpapi/handler_*_integration_test.go` test fixtures. |
| `app_service_adapter_auth_context.go` | ~320 lines | (only methods on `*AppServiceAdapter` — no top-level exported symbols beyond methods): `normalizeMutationAuthorizationRequest`, `enrichMutationAuthorizationContext`, `populateCreateActionItemAuthContext`, `populateCommentAuthContext`, `populateActionItemAuthContext`, `populateHandoffAuthContext`, `populateAttentionAuthContext`, `populateLeaseAuthContext`, `populateCapabilityScopeAuthContext`, `populateLevelAuthContext`. Unexported helpers: `enforceMutationApprovedPathPolicy`, `mutationActionRequiresGlobalApprovedPath`, `mutationActionRequiresProjectScopedApproval`, `applyResolvedAuthScopeContext`, `populateProjectAuthContext`, `projectIDFromNamespace`, `firstNonEmptyTrimmed`. | **All `transport-neutral`.** Methods on `AppServiceAdapter` for resolving auth-scope context per MCP/HTTP mutation. Used by both transports. |
| `app_service_adapter_mcp.go` | ~2470 lines (largest file in `common/`) | ~100 exported methods on `*AppServiceAdapter` covering: project CRUD (`CreateProject`, `UpdateProject`, `ListProjects`, `GetProjectBySlug`); action-item CRUD/state (`ListActionItems`, `GetActionItem`, `CreateActionItem`, `UpdateActionItem`, `MoveActionItem`, `MoveActionItemState`, `SupersedeActionItem`, `DeleteActionItem`, `RestoreActionItem`, `ReparentActionItem`, `ListChildActionItems`, `SearchActionItems`, `ResolveActionItemID`); embeddings (`GetEmbeddingsStatus`, `ReindexEmbeddings`); change events (`ListProjectChangeEvents`, `GetProjectDependencyRollup`); kinds (`ListKindDefinitions`, `UpsertKindDefinition`, `SetProjectAllowedKinds`, `ListProjectAllowedKinds`); templates (`GetProjectTemplate`, `ListBuiltinTemplates`, `ValidateCandidateTemplate`, `SetProjectTemplate`); capability leases (`ListCapabilityLeases`, `IssueCapabilityLease`, `HeartbeatCapabilityLease`, `RenewCapabilityLease`, `RevokeCapabilityLease`, `RevokeAllCapabilityLeases`); comments (`CreateComment`, `ListCommentsByTarget`); handoffs (`CreateHandoff`, `GetHandoff`, `ListHandoffs`, `UpdateHandoff`); auth-request lifecycle (`CreateAuthRequest`, `ListAuthRequests`, `GetAuthRequest`, `ClaimAuthRequest`, `ApproveAuthRequest`, `CancelAuthRequest`, `ListAuthSessions`, `ValidateAuthSession`, `CheckAuthSessionGovernance`, `RevokeAuthSession`); bootstrap (`GetBootstrapGuide`). | **All `transport-neutral`.** Despite the `_mcp.go` filename suffix, these methods are the canonical app-service surface that BOTH HTTP and MCP transports invoke through `common.*Service` interfaces. The MCP-suffix is historic naming, not a transport restriction. |
| `mcp_surface.go` | ~1060 lines | ~70 exported types (request/result DTOs + service interfaces): `ErrBootstrapRequired`, `ErrGuardrailViolation`; `BootstrapGuide`, `ActorLeaseTuple`; request types `CreateProjectRequest`, `UpdateProjectRequest`, `CreateActionItemRequest`, `UpdateActionItemRequest`, `MoveActionItemRequest`, `MoveActionItemStateRequest`, `SupersedeActionItemRequest`, `DeleteActionItemRequest`, `RestoreActionItemRequest`, `ReparentActionItemRequest`, `SearchActionItemsRequest`, `EmbeddingsStatusRequest`, `ReindexEmbeddingsRequest`, `UpsertKindDefinitionRequest`, `SetProjectAllowedKindsRequest`, `IssueCapabilityLeaseRequest`, `HeartbeatCapabilityLeaseRequest`, `RenewCapabilityLeaseRequest`, `RevokeCapabilityLeaseRequest`, `RevokeAllCapabilityLeasesRequest`, `ListCapabilityLeasesRequest`, `CreateCommentRequest`, `CreateHandoffRequest`, `UpdateHandoffRequest`, `ListHandoffsRequest`, `CreateAuthRequestRequest`, `ListAuthRequestsRequest`, `ClaimAuthRequestRequest`, `CancelAuthRequestRequest`, `ApproveAuthRequestRequest`, `ListAuthSessionsRequest`, `ValidateAuthSessionRequest`, `CheckAuthSessionGovernanceRequest`, `RevokeAuthSessionRequest`, `ListCommentsByTargetRequest`, `GetProjectTemplateRequest`, `ValidateCandidateTemplateRequest`, `SetProjectTemplateRequest`; result types `SearchActionItemMatch`, `SearchActionItemsResult`, `EmbeddingSummary`, `EmbeddingStatusRow`, `EmbeddingsStatusResult`, `ReindexEmbeddingsResult`, `ApproveAuthRequestResult`, `AuthRequestRecord`, `AuthRequestClaimResult`, `AuthSessionRecord`, `AuthSessionGovernanceCheckResult`, `CommentRecord`, `GetProjectTemplateResult`, `ListBuiltinTemplatesResult`, `ValidateCandidateTemplateResult`, `SetProjectTemplateResult`; service interfaces `BootstrapGuideReader`, `ProjectService`, `ActionItemService`, `SearchService`, `EmbeddingsService`, `ChangeFeedService`, `KindCatalogService`, `TemplateService`, `CapabilityLeaseService`, `CommentService`, `HandoffService`, `AuthRequestService`; helper `durationFromSeconds`. | **All `transport-neutral`** despite the file name. Same naming-historicism note as `app_service_adapter_mcp.go`. The `Service` interfaces are the boundary between `common/` (the Service adapter) and `mcpapi/` (the MCP RPC tool registry); they are NOT MCP-specific. |
| **Test files** in `common/` (6 files; all classified along with their production peers): `app_service_adapter_test.go`, `app_service_adapter_auth_context_test.go`, `app_service_adapter_auth_requests_test.go`, `app_service_adapter_auth_test.go`, `app_service_adapter_helpers_test.go`, `app_service_adapter_lifecycle_test.go`, `app_service_adapter_mcp_actor_attribution_test.go`, `app_service_adapter_mcp_guard_test.go`, `app_service_adapter_mcp_helpers_test.go`, `app_service_adapter_outcome_test.go`, `app_service_adapter_steward_gate_test.go`, `capture_test.go` | (test files; no production exports) | All `transport-neutral`. Tests stay co-located with the production code under whichever new package houses the adapter (likely `internal/adapters/mcp_common/`). |

**Package-level recommendation for W7.D2**: extract `internal/adapters/server/common/` to `internal/adapters/mcp_common/` as a whole. No file-level splits required.

---

### 1.3 `internal/adapters/server/httpapi/` (1 production file, 2 test files)

**Package classification**: **`http-residue`** in full. The package docstring is absent but the file `handler.go` defines `Handler`, `ServeHTTP`, MIME-aware JSON decoding, `http.ResponseWriter` error mapping, and exists solely to serve the `/api/v1/...` REST surface composed by `server.go`'s `NewHandler`. No stdio path touches this package.

| File | Exported symbols | Classification |
|---|---|---|
| `handler.go` | `Handler` (type), `APIError` (type), `ErrorEnvelope` (type), `NewHandler(captureState, attention) *Handler`, `(*Handler).ServeHTTP(w, r)`. Unexported: `maxRequestBodyBytes`, `raiseAttentionItemPayload`, `resolveAttentionItemPayload`, `httpMutationGuardArgs`, `handleCaptureState`, `handleListAttentionItems`, `handleRaiseAttentionItem`, `handleResolveAttentionItem`, `authorizeHTTPMutation`, `buildAuthenticatedHTTPActor`, `resolveAttentionItemID`, `normalizePath`, `writeErrorFrom`, `httpErrorMapping`, `mapHTTPError`, `firstNonEmptyString`, `writeMethodNotAllowed`, `writeJSONError`, `writeJSON`, `decodeJSONBody`, `decodeOptionalJSONBody` | **`http-residue`** |
| `handler_integration_test.go` | Test fixtures: `newRealAttentionHandlerForTest`, `issueUserSessionForTest`, `approvedPathAttentionFixture`, `newApprovedPathAttentionFixture`, `firstHTTPProjectColumnIDForTest`, `seedHTTPOrphanKindsForTest`, `createHTTPScopedActionItemChainForTest`, `issueApprovedPathHTTPTestSession`; tests: `TestHandlerAttentionMutationPersistsAuthenticatedAttribution`, `TestHandlerResolveAttentionItemApprovedPath`, `TestHandlerResolveAttentionItemOutOfScopeApprovedPathDenied` | **`http-residue`** |
| `handler_test.go` | Stubs: `stubCaptureStateReader`, `stubAttentionService`, `stubMutationAuthorizer`. Helpers: `decodeBody[T]`, `captureDefaultLoggerOutput`, `decodeErrorEnvelope`. Tests: `TestHandlerCaptureStateSuccess`, `TestHandlerCaptureStateErrorMapping`, `TestHandlerAttentionUnavailable`, `TestHandlerAttentionEndpoints`, `TestHandlerRouteGuards`, `TestHandlerCaptureStateServiceUnavailable`, `TestHandlerAttentionEndpointsUnavailable`, `TestHandlerAttentionJSONValidation`, `TestHandlerAttentionMutationsRequireSession`, `TestHandlerAttentionAgentMutationsRequireGuardTuple`, `TestHandlerRaiseAttentionScopeValidationErrorMapping`, `TestHandlerAttentionListRequiresProjectID`, `TestHandlerResolveAttentionItemMinimalBody`, `TestDecodeJSONBodyBranches`, `TestDecodeOptionalJSONBodyBranches`, `TestWriteErrorFromMappingBranches`, `TestResolveAttentionItemID`, `TestNormalizePath` | **`http-residue`** |

**Package-level recommendation for W7.D2**: do nothing. Entire `httpapi/` package deletes in W7.D3.

---

### 1.4 `internal/adapters/server/mcpapi/` (8 production files, 4 test files; ~16K+ LOC)

**Package classification — at the package level**: **`transport-neutral`**. Despite the `mcpapi/` name, this package is the **MCP RPC tool-registry engine** — it registers MCP tools (`till.auth_request`, `till.capture_state`, etc.) onto an `*mcpserver.MCPServer` and exposes that server via BOTH (a) an HTTP streamable handler (`NewHandler` → `*Handler` with embedded `http.Handler`) AND (b) a stdio entry point (`ServeStdio`). The RPC registry (i.e., the `register*Tools` functions + tool argument structs) is identical for both transports; only the **bind layer** in this package's `handler.go` differs.

W7.D2 destination per PLAN.md Acceptance line 625: `internal/adapters/mcp_rpc/` (proposed package name `mcprpc`).

**File-level detail** — two files have HTTP-only sub-content that should be **called out** but doesn't require sub-file splitting in W7.D1's deliverable; W7.D2 builder decides whether to split:

| File | Line count guidance | Exported symbols | Classification | Notes |
|---|---|---|---|---|
| `auth_context_runtime.go` | ~165 lines | (unexported only): `mcpAuthContextStore`, `storedMCPAuthContext`, `newMCPAuthContextStore`, `Bind`/`Resolve` methods, `newMCPAuthContextID`, `withMCPToolAuthRuntime`, `bindMCPAuthContext`, `resolveMCPMutationAuth`, `resolveMCPActingSessionAuth` | **`transport-neutral`** | Per-MCP-session auth-context store. Used by BOTH HTTP (when `EnableAuthContexts = false`) and stdio (when `EnableAuthContexts = true`). The boolean flag flips at the call site (`NewHandler` sets false; `ServeStdio` sets true) — the runtime itself is shared. |
| `handler.go` | ~1150 lines | `Config` (type), `Handler` (type), `NewServer(cfg, captureState, attention) (*mcpserver.MCPServer, Config, error)`, **`NewHandler(cfg, captureState, attention) (*Handler, error)`**, **`ServeStdio(cfg, captureState, attention) error`**, `(*Handler).ServeHTTP(w, r)`. Unexported: `authRequestCreateResult`, `registerAuthRequestTools`, `normalizeConfig`, `registerCaptureStateTool`, `attentionItemMutationArgs`, `registerAttentionTools`, `registerLegacyAttentionListTool`, `registerLegacyAttentionMutationTools`, `handleAttentionItemMutation`, `toolResultFromError`, `toolErrorMapping`, `mapToolError`, plus a dozen `pick*Service` interface-narrowing helpers | **MIXED — but extract whole**. `NewServer` (line 44) is the canonical MCP-server constructor — **`transport-neutral`**. `NewHandler` (line 509) + `(*Handler).ServeHTTP` (line 534) wrap that server in a streamable HTTP transport — **`http-residue` semantically**, but it is so small (~13 lines wrapping `mcpserver.NewStreamableHTTPServer`) that splitting it inside W7.D2 is wasteful. **`ServeStdio` (line 524) is `stdio-relevant`** — the load-bearing entry point `RunStdio` in `server.go` invokes. **Recommendation**: extract the whole file to `mcp_rpc/`. The HTTP wrapper (`NewHandler`, `(*Handler).ServeHTTP`, the `httpHandler` field on `Handler`, the `Handler` type itself) becomes the **first thing W7.D3 deletes** from `mcp_rpc/`. `ServeStdio` and `NewServer` survive. |
| `extended_tools.go` | ~3700 lines (largest file in server tree) | `mcpSessionAuthArgs`, `mcpMutationGuardArgs`, `capabilityLeaseMutationArgs`. Helpers `authorizeMCPMutation`, `buildAuthenticatedMutationActor`, `firstNonEmptyString`, `buildProjectRootedMutationAuthScope`, `handleCapabilityLeaseMutation`, `registerBootstrapTool`, `registerProjectTools`, `registerActionItemTools`, `registerKindTools`, `templateInputMaxBytes`, `registerTemplateTools`, `registerCapabilityLeaseTools`, `registerLegacyCapabilityLeaseReadTool`, `registerLegacyCapabilityLeaseMutationTools`, `registerCommentTools`, `optionalDurationArg`, `invalidRequestToolResult`, `resolveActionItemIDForRead`, `rejectMutationDottedActionItemID` | **`transport-neutral`** | All `register*Tools(srv *mcpserver.MCPServer, ...)` register against an `*mcpserver.MCPServer` — protocol-neutral inside the MCP server's tool table. The same registrations serve HTTP and stdio. |
| `handoff_tools.go` | ~265 lines | (unexported): `handoffMutationArgs`, `registerHandoffTools`, `registerLegacyHandoffReadTools`, `registerLegacyHandoffMutationTools`, `handleHandoffMutation` | **`transport-neutral`** | Same pattern — tool registrations. |
| `instructions_explainer.go` | ~675 lines | (unexported): `instructionsExplainServices`, `instructionsExplainRequest`, `instructionsExplainResult`, `explainInstructionsScope`, `explainTopicInstructions`, `explainBootstrapTopic`, `explainProjectInstructions`, `explainKindInstructions`, `explainNodeInstructions`, `findProjectByID`, `findKindByID`, `tryFindKindByID`, `listProjectAllowedKinds`, `loadActionItemLineage`, `summarizeActionItemLineage`, `collectNodeScopedRules`, `collectNodeWorkflowContract`, `collectNodeAgentExpectations`, `collectNodeWhyItApplies`, `collectNodeEvidence`, `collectNodeGaps`, `buildProjectWhyItApplies`, `kindContextSuffix`, `fallbackText`, `joinKindScopes`, `capitalizeASCIIScope` | **`transport-neutral`** | Synthesizes `till.get_instructions` response payloads. No HTTP knowledge. |
| `instructions_tool.go` | ~370 lines | (unexported): `instructionsToolMode`, `instructionsToolFocus`, `instructionsToolRequest`, `instructionsToolDoc`, `instructionsToolResolvedScope`, `instructionsToolRelatedTool`, `instructionsToolEvidence`, `instructionsToolExplanation`, `instructionsToolResponse`, `registerInstructionsTool`, `buildInstructionsToolResponse`, `normalizeInstructionsToolModeAndFocus`, `buildInstructionsDocsSummary`, `recommendedInstructionSettings`, `recommendedMDFileGuidance` | **`transport-neutral`** | Tool registration for `till.get_instructions`. |
| `strict_decode.go` | ~135 lines | (unexported): `errUnknownField`, `jsonUnknownFieldPrefix`, `bindArgumentsStrict(req mcp.CallToolRequest, target any) error`, `unknownFieldName` | **`transport-neutral`** | Strict JSON decoding of MCP tool arguments. Used by every `register*Tools`. Protocol-neutral. |
| **Test files** in `mcpapi/`: `extended_tools_test.go`, `handler_integration_test.go`, `handler_steward_integration_test.go`, `handler_test.go`, `instructions_explainer_test.go`, `instructions_tool_test.go`, `strict_decode_test.go` | (tests; no production exports) | **`transport-neutral`** | Tests stay co-located with `mcp_rpc/` after extraction. Note: integration tests use `httptest` to drive the streamable-HTTP `Handler` directly — these become invalid once `NewHandler` deletes in W7.D3 and must be rewritten to drive `NewServer` + stdio instead, OR deleted. **W7.D3 handles the test-deletion decision.** |

**Package-level recommendation for W7.D2**:

1. Extract the entire `internal/adapters/server/mcpapi/` directory to `internal/adapters/mcp_rpc/` (package rename `mcpapi` → `mcprpc`).
2. **No file-level splits required**: even `handler.go`'s embedded HTTP-streamable wrapper extracts cleanly with the rest. W7.D3 then deletes the HTTP-wrapper portion (`NewHandler`, `Handler` type, `(*Handler).ServeHTTP`, `httpHandler` field, the `httpHandler` import + the `mcpserver.NewStreamableHTTPServer` call) plus any tests that drove it.
3. Update intra-package import: `mcpapi/` files import `internal/adapters/server/common/` — when moved, this becomes `internal/adapters/mcp_common/`.

**`stdio-relevant` symbols inside `mcpapi/`**: `ServeStdio` (handler.go:524) is the only function whose signature implies stdio. Everything else is RPC-registry. Inside `mcprpc/` post-extraction, `ServeStdio` could **optionally** be moved to a separate `internal/adapters/mcp_stdio/` package — but PLAN.md Acceptance line 625 explicitly proposes both `mcp_stdio/` AND `mcp_rpc/` as separate package homes, leaving the split to W7.D2 builder judgment. **Recommendation: keep `ServeStdio` inside `mcp_rpc/`** because it is a 7-line shim over `mcpserver.ServeStdio(mcpSrv)` and splitting it requires re-exporting `NewServer` + `Config` + `normalizeConfig` across two packages for one function. The `internal/adapters/mcp_stdio/` package proposed in PLAN.md may end up being just the `RunStdio` shim from `server.go` (item 1.1 above) — a thin caller of `mcprpc.ServeStdio`.

---

## 2. Consumer Map (Exhaustive)

Generated via `git grep -nE "\"github.com/evanmschultz/tillsyn/internal/adapters/server"`. Every importer of any `internal/adapters/server/...` package is enumerated.

### 2.1 External consumers (outside `internal/adapters/server/`)

| File | Line | Import statement | Symbols used | Subcommand / context |
|---|---|---|---|---|
| `cmd/till/main.go` | 23 | `serveradapter "github.com/evanmschultz/tillsyn/internal/adapters/server"` | `serveradapter.Config`, `serveradapter.Dependencies`, `serveradapter.Run`, `serveradapter.RunStdio` | `till serve` (HTTP — line 56–57, 2668–2674) + `till mcp` (stdio — line 81–82, 2683–2687). |
| `cmd/till/main.go` | 24 | `servercommon "github.com/evanmschultz/tillsyn/internal/adapters/server/common"` | `servercommon.NewAppServiceAdapter` (lines 2653, 2682, 2763); `servercommon.CaptureStateRequest` (line 2764) | `till serve`, `till mcp`, AND `till capture-state` (line 2763–2764). |
| `cmd/till/main_test.go` | 22 | `serveradapter "github.com/evanmschultz/tillsyn/internal/adapters/server"` | `serveradapter.Config`, `serveradapter.Dependencies` injected into `serveCommandRunner` / `mcpCommandRunner` fakes (lines 493, 2324–2326, 2372, 2402, 2442, 2474–2476, 2522, 2542–2543, 2578, 2616, 2661, 2714) | `till serve` + `till mcp` CLI tests. |
| `cmd/till/main_test.go` | 23 | `servercommon "github.com/evanmschultz/tillsyn/internal/adapters/server/common"` | `servercommon.AppServiceAdapter` (line 129), `servercommon.NewAppServiceAdapter` (lines 143, 1350), `servercommon.MutationAuthorizationRequest` (lines 1350, 1380, 1417, 1455), `servercommon.ErrInvalidAuthentication` (line 1388), `servercommon.ErrAuthorizationDenied` (line 1425), `servercommon.ErrGrantRequired` (line 1463) | Auth-flow integration tests (`newAuthAdapterForTest`, `TestAuth*`-prefixed tests around lines 1300–1500). |

### 2.2 Internal cross-package references (inside `internal/adapters/server/`)

| File | Line | Import | Notes |
|---|---|---|---|
| `internal/adapters/server/server.go` | 12 | `"github.com/evanmschultz/tillsyn/internal/adapters/server/common"` | Top-level composer reads `common.CaptureStateReader`, `common.AttentionService` from the `Dependencies` struct. |
| `internal/adapters/server/server.go` | 13 | `"github.com/evanmschultz/tillsyn/internal/adapters/server/httpapi"` | Top-level composer mounts `httpapi.NewHandler(...)` at `cfg.APIEndpoint`. **Drops in W7.D3** (HTTP residue). |
| `internal/adapters/server/server.go` | 14 | `"github.com/evanmschultz/tillsyn/internal/adapters/server/mcpapi"` | Top-level composer mounts `mcpapi.NewHandler(...)` and calls `mcpapi.ServeStdio(...)`. **Becomes `mcp_rpc.ServeStdio` after W7.D2.** |
| `internal/adapters/server/httpapi/handler.go` | 14 | `"github.com/evanmschultz/tillsyn/internal/adapters/server/common"` | HTTP transport reads/writes `common.CaptureState`, `common.AttentionService`, `common.MutationAuthorizer`, the `common.Err*` sentinels, `common.ActorLeaseTuple`. **All transport-neutral usages — survive in `mcp_common/` after extraction.** |
| `internal/adapters/server/httpapi/handler_integration_test.go` | 15 | `servercommon "github.com/evanmschultz/tillsyn/internal/adapters/server/common"` | Test fixture. Deletes with the httpapi package. |
| `internal/adapters/server/httpapi/handler_test.go` | 16 | `"github.com/evanmschultz/tillsyn/internal/adapters/server/common"` | Test stubs implementing `common.CaptureStateReader` + `common.AttentionService`. Deletes with the httpapi package. |
| `internal/adapters/server/mcpapi/auth_context_runtime.go` | 11 | `"github.com/evanmschultz/tillsyn/internal/adapters/server/common"` | Reads `common.ActorLeaseTuple` (in the broader file context). **Survives — rewrites to import `mcp_common/`.** |
| `internal/adapters/server/mcpapi/extended_tools.go` | 9 | `"github.com/evanmschultz/tillsyn/internal/adapters/server/common"` | Heavy consumer — every `register*Tools` reads `common.*Service` interfaces. **Survives — rewrites to import `mcp_common/`.** |
| `internal/adapters/server/mcpapi/extended_tools_test.go` | 18 | `"github.com/evanmschultz/tillsyn/internal/adapters/server/common"` | Test fixtures. **Survive — rewrite import.** |
| `internal/adapters/server/mcpapi/handler.go` | 13 | `"github.com/evanmschultz/tillsyn/internal/adapters/server/common"` | Reads `common.CaptureStateReader`, `common.AttentionService`, the `pick*Service` helpers' return types. **Survives — rewrites import.** |
| `internal/adapters/server/mcpapi/handler_integration_test.go` | 13 | `servercommon "github.com/evanmschultz/tillsyn/internal/adapters/server/common"` | Integration test fixture. **Decision in W7.D3**: rewrite to drive `NewServer` + stdio, or delete with the streamable-HTTP wrapper. |
| `internal/adapters/server/mcpapi/handler_steward_integration_test.go` | 15 | `servercommon "github.com/evanmschultz/tillsyn/internal/adapters/server/common"` | Steward integration test. Same W7.D3 decision. |
| `internal/adapters/server/mcpapi/handler_test.go` | 17 | `"github.com/evanmschultz/tillsyn/internal/adapters/server/common"` | Unit tests against MCP handler. Same W7.D3 decision. |
| `internal/adapters/server/mcpapi/handoff_tools.go` | 8 | `"github.com/evanmschultz/tillsyn/internal/adapters/server/common"` | Reads `common.HandoffService`. **Survives — rewrites import.** |
| `internal/adapters/server/mcpapi/instructions_explainer.go` | 9 | `"github.com/evanmschultz/tillsyn/internal/adapters/server/common"` | Reads `common.BootstrapGuide`, `common.ProjectService`, `common.KindCatalogService`, etc. **Survives — rewrites import.** |
| `internal/adapters/server/mcpapi/instructions_tool_test.go` | 8 | `"github.com/evanmschultz/tillsyn/internal/adapters/server/common"` | Test fixture. **Survives — rewrites import.** |

### 2.3 Workflow MDs (incidental string matches; not Go imports)

| File | Line | Note |
|---|---|---|
| `workflow/drop_4c_6_1/PLAN_QA_FALSIFICATION.md` | 92, 482 | Documentation references to the import lines above; not source code. |

### 2.4 No-consumer audit

`git grep -nE "\"github.com/evanmschultz/tillsyn/internal/adapters/server"` returns 20 hits total. All accounted for above:

- 4 in `cmd/till/` (the only external consumers — 2 production import lines + 2 test import lines).
- 14 inside `internal/adapters/server/` (the package's own cross-references).
- 2 in workflow MDs (documentation; not source).

**Conclusion**: the W7.D2 import-rewrite surface is **`cmd/till/main.go` + `cmd/till/main_test.go` + every cross-reference inside the `mcpapi/` package post-extraction**. No surprise consumer hides in any other `internal/*` subpackage. The `git grep` exhaustion satisfies the R3-FF1 belt-and-suspenders requirement; LSP `findReferences` would only re-confirm this list because Go imports are textually exact (no aliasing other than `serveradapter` / `servercommon` which the literal string match catches).

---

## 3. Summary Counts

| Category | Files | Production-file count | Production exported-symbol count (approx) |
|---|---|---|---|
| `http-residue` (full files) | `internal/adapters/server/httpapi/handler.go` + 2 test files | 1 production file | 5 exported (`Handler`, `APIError`, `ErrorEnvelope`, `NewHandler`, `(*Handler).ServeHTTP`) |
| `http-residue` (split inside `server.go`) | parts of `internal/adapters/server/server.go` | (inside mixed file) | 6 (`defaultBindAddress`, `defaultShutdownTimeout`, `NewHandler`, `Run`, `writeHealthStatus`, `(*Handler).ServeHTTP` wrapper inside `mcpapi/handler.go`) |
| `stdio-relevant` (split inside `server.go`) | part of `internal/adapters/server/server.go` | (inside mixed file) | 1 (`RunStdio`) — calls into `mcpapi.ServeStdio` inside the `mcp_rpc/` package. |
| `stdio-relevant` (split inside `mcpapi/handler.go`) | part of `internal/adapters/server/mcpapi/handler.go` | (inside transport-neutral file) | 1 (`ServeStdio`) — recommended to keep inside `mcp_rpc/` rather than split out. |
| `transport-neutral` (whole packages) | `internal/adapters/server/common/` (5 production files + 12 test files); `internal/adapters/server/mcpapi/` (8 production files + 7 test files) | 13 production files | ~150 exported types/funcs/vars in `common/` (largest concentration in `mcp_surface.go` + `app_service_adapter_mcp.go`); `mcpapi/` is largely unexported (`register*Tools` etc.) — only `Config`, `Handler`, `NewServer`, `NewHandler`, `ServeStdio`, `(*Handler).ServeHTTP` are exported. |
| `transport-neutral` (split inside `server.go`) | parts of `internal/adapters/server/server.go` | (inside mixed file) | 4 (`Config`, `Dependencies`, `normalizeConfig`, `normalizeEndpoint`) |

**File totals**:

- `http-residue` files: 1 production file (`httpapi/handler.go`) + 2 test files (`httpapi/handler_test.go`, `httpapi/handler_integration_test.go`). Plus the HTTP-only **portions** of `server.go` and `mcpapi/handler.go` (split inside transport-neutral files).
- `stdio-relevant` files: zero **pure** stdio-only files. Two **portions** of mixed files: `RunStdio` inside `server.go` and `ServeStdio` inside `mcpapi/handler.go`. Both extract into `mcp_rpc/` (or optionally `mcp_stdio/`) per W7.D2 builder judgment.
- `transport-neutral` files: **all of** `internal/adapters/server/common/` (5 production + 12 test = 17 files) + **all of** `internal/adapters/server/mcpapi/` (8 production + 7 test = 15 files) + **portions** of `server.go`.

**Total files under `internal/adapters/server/`**: 36 files (1 top-level `server.go` + 17 in `common/` + 3 in `httpapi/` + 15 in `mcpapi/`).

---

## 4. W7.D2 Package-Map Recommendation

Per PLAN.md Acceptance line 625 (proposed package homes), the inventory yields the following extraction plan for W7.D2:

| Destination package | Source | Contents |
|---|---|---|
| `internal/adapters/mcp_common/` | `internal/adapters/server/common/` (entire package) | The `AppServiceAdapter` + auth helpers + types + capture service. Imports: `internal/app`, `internal/domain`, `internal/adapters/auth/autentauth`. |
| `internal/adapters/mcp_rpc/` | `internal/adapters/server/mcpapi/` (entire package; **HTTP-wrapper portions of `handler.go` survive into `mcp_rpc/` and are deleted in W7.D3**) | The MCP tool-registry engine: `NewServer`, `ServeStdio`, the `register*Tools` set, `instructions_*`, `strict_decode`, `auth_context_runtime`, `handoff_tools`, `extended_tools`. Import path rewrites: `internal/adapters/server/common` → `internal/adapters/mcp_common`. |
| `internal/adapters/mcp_stdio/` | The `RunStdio` shim from `server.go` (optional — could also live in `mcp_rpc/`) | Thin caller of `mcp_rpc.ServeStdio`. PLAN.md mentions this as a possible destination but inventory finds the shim is so small the W7.D2 builder may decide to skip the package and put `RunStdio` inside `mcp_rpc/`. |
| `internal/adapters/server/` (remains) | The HTTP-residue portions of `server.go` (`Run`, `NewHandler`, `writeHealthStatus`, `defaultBindAddress`, `defaultShutdownTimeout`) + the entire `httpapi/` subpackage | Survives W7.D2 only to be deleted by W7.D3. |

`cmd/till/main.go` import rewrites (per consumer map §2.1):

- Line 23: `serveradapter "github.com/evanmschultz/tillsyn/internal/adapters/server"` → split into multiple imports per which symbols `cmd/till/main.go` actually uses (`mcp_rpc.ServeStdio` for `till mcp`; the `till serve` HTTP path goes away in W7.D3).
- Line 24: `servercommon "github.com/evanmschultz/tillsyn/internal/adapters/server/common"` → `mcpcommon "github.com/evanmschultz/tillsyn/internal/adapters/mcp_common"`.

`cmd/till/main_test.go` mirrors the same rewrites (lines 22–23). Auth-flow tests around lines 1300–1500 use `servercommon.*` symbols extensively and rewrite their import.

---

## 5. Risk Annotations for W7.D2

Carried forward from PLAN.md RiskNotes + observed during this audit:

- **`mcpapi/handler.go` is mixed at the function level, not the file level.** The `Handler` type + `NewHandler` + `(*Handler).ServeHTTP` are HTTP-streamable wrappers around `mcpserver.NewStreamableHTTPServer`. They survive the W7.D2 extraction (move into `mcp_rpc/handler.go`), then W7.D3 deletes them along with the `httpHandler` field on `Handler` and the `mcp-go/server` streamable import. The file as a whole stays.
- **Test-file decision for `mcpapi/handler_test.go` + `handler_integration_test.go` + `handler_steward_integration_test.go`**: these tests drive the streamable-HTTP `Handler` directly. W7.D3 cannot keep them as-is. Two options the W7.D3 builder picks between: (a) rewrite to drive `mcp_rpc.NewServer` + an in-memory MCP transport, or (b) delete and rely on the stdio path being exercised by other tests. **W7.D1 does not decide; W7.D3 reads the inventory and chooses.**
- **`server.go` mixed-file split**: W7.D2 builder must extract `RunStdio` (line 122) + the `normalizeConfig` / `normalizeEndpoint` helpers it transitively requires + the stdio-relevant fields of `Config` + the full `Dependencies` struct into the appropriate new package. The HTTP-side functions (`NewHandler`, `Run`, `writeHealthStatus`) and constants (`defaultBindAddress`, `defaultShutdownTimeout`) STAY in `internal/adapters/server/server.go` for W7.D3 deletion.
- **`Config` field-level split**: `Config` in `server.go` has BOTH HTTP-only fields (`HTTPBind`, `APIEndpoint`) AND transport-neutral fields (`ServerName`, `ServerVersion`, `MCPEndpoint`, `Expose*Tools`). W7.D2 builder either (a) keeps the unified struct in `mcp_common/` and trims HTTP-only fields in W7.D3, or (b) splits at extraction time. Recommendation: option (a) — defer field trimming to W7.D3 so the `cmd/till/main.go` rewrite in W7.D2 is mechanically simpler.
- **Intra-`mcpapi/` cross-references**: 8 production files + 7 test files in `mcpapi/` import `internal/adapters/server/common`. Every one rewrites to `internal/adapters/mcp_common` in W7.D2. The consumer map §2.2 enumerates each. **No file in `mcpapi/` imports `httpapi/`** — confirmed via the `git grep` exhaustion in §2.2 (zero `httpapi` matches inside `mcpapi/`). The intra-package dep graph is `mcpapi/ → common/` only.
- **No third-party consumer of `httpapi/`**: the only importer of `httpapi/` in the entire repo is `internal/adapters/server/server.go` line 13. When W7.D3 deletes `server.go`'s HTTP-side functions, the single `httpapi.NewHandler` call at `server.go:68` deletes with it, leaving the `httpapi/` package with zero consumers. Safe to delete the package wholesale.

---

## 6. Acceptance Self-Check (PLAN.md L1 W7.D1)

- [x] `workflow/drop_4c_6_1/W7_INVENTORY.md` exists with three categorized lists (§1 above; §3 summary).
- [x] Every exported symbol in every file under `internal/adapters/server/` (including `mcpapi/`) is assigned to exactly one category. Mixed-file callouts are at the function level inside `server.go` and `mcpapi/handler.go`; every individual function is assigned.
- [x] Consumer map (§2) lists EVERY file that imports any `internal/adapters/server/...` package, with file:line citations. Generated via `git grep -nE "\"github.com/evanmschultz/tillsyn/internal/adapters/server"` — 20 hits, all enumerated.
- [x] `mcpapi/` package is explicitly classified as `transport-neutral` (the MCP RPC tool registry shared by all transports) with sub-file callouts for the HTTP-streamable wrapper portions of `handler.go`.
- [x] `mage ci` status unchanged from pre-D1: pre-existing `internal/tui/style/` build error from concurrent W4-wave uncommitted work; D1 introduced ZERO code changes (only this MD file).

---

## 7. Hylla Feedback

N/A — research touched non-Go files only (read Go for classification, did not query Hylla; PLAN.md L1 explicitly said "Hylla is OFF" for this droplet per filesystem-MD mode + Go-only Hylla today rule).

Tool ergonomics gripe (in scope per agent definition): `Bash` rejected `grep`, `ls -R`, and `find` calls citing the agent allowlist, even though `git grep` and explicit `git`-prefixed commands worked. The allowlist as configured leans more conservative than the agent definition suggests it should be for read-only inspection. Worked around by using `git grep -nE "..."` for everything, which is fine, but `git grep` is technically scoped to tracked files only — could miss untracked staging-in-progress work. Not a blocker for W7.D1 because every file under `internal/adapters/server/` IS git-tracked (verified via `git status --porcelain` showing no untracked files in that subtree).
