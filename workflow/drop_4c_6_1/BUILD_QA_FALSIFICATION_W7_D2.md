# W7.D2 — BUILD-QA-FALSIFICATION Verdict

**Date:** 2026-05-12
**Reviewer:** go-qa-falsification-agent (opus)
**Overall verdict:** PASS WITH NITS

## Attack Hypotheses Tested

### H1 — Inventory completeness (extraction miss?)

**Hypothesis:** A file or symbol classified non-http-residue in `W7_INVENTORY.md` is still inside `internal/adapters/server/`.

**Test:** `git ls-files internal/adapters/server/**/*.go` and read `internal/adapters/server/server.go` top-level decls.

**Finding:** Files remaining under `internal/adapters/server/`:
- `server.go` — `defaultShutdownTimeout`, `NewHandler`, `Run`, `writeHealthStatus` (all classified http-residue in inventory §1.1).
- `httpapi/handler.go`, `httpapi/handler_test.go`, `httpapi/handler_integration_test.go` (all classified http-residue in inventory §1.3).

Every extracted symbol per inventory:
- `Config`, `Dependencies`, `NormalizeConfig`, `NormalizeEndpoint`, `defaultBindAddress` → `mcp_common/server_config.go` ✓
- `RunStdio` → `mcp_stdio/stdio.go` ✓
- All of `server/common/*.go` → `mcp_common/` (19 files renamed) ✓
- All of `server/mcpapi/*.go` → `mcp_rpc/` (13 files renamed) ✓

**Verdict:** REFUTED. Zero extraction misses.

### H2 — Consumer-map completeness

**Hypothesis:** A consumer in `W7_INVENTORY.md` §2 was missed during import rewriting.

**Test:** `git grep -nE 'internal/adapters/server'` and classify every hit.

**Finding:** Total hits = 11 (in non-MD files):
| File | Status |
|---|---|
| `cmd/till/main.go:25` (`serveradapter "...server"`) | Correct — server.go still has Run for `till serve` |
| `internal/adapters/server/server.go:16` (`...server/httpapi`) | Correct — self-reference, http-residue |
| `internal/adapters/mcp_rpc/extended_tools_test.go:465` | Test-data string literal (stale path in fixture). NIT. |
| `internal/adapters/mcp_rpc/handler_steward_integration_test.go:184` | `//` doc comment. NIT. |
| `internal/app/auth_requests.go:760` | `//` doc comment. NIT. |
| `internal/app/auto_generate_steward.go:17` | `//` doc comment. NIT. |
| `internal/app/dispatcher/walker.go:257` | `//` doc comment. NIT. |
| `internal/app/service.go:1444` | `//` doc comment. NIT. |
| `internal/domain/errors.go:88` | `//` doc comment. NIT. |
| `internal/templates/builtin/till-gen.toml:82` | TOML `#` comment. NIT. |
| `internal/templates/builtin/till-go.toml:55` | TOML `#` comment. NIT. |
| `internal/templates/embed_test.go:469` | `//` doc comment. NIT. |

Zero live imports of `server/common` or `server/mcpapi` remain.

`cmd/till/main.go` correctly rewrites:
- `serveradapter.Config/Dependencies` → `mcpcommon.Config/Dependencies`
- `servercommon.NewAppServiceAdapter` → `mcpcommon.NewAppServiceAdapter`
- `servercommon.CaptureStateRequest` → `mcpcommon.CaptureStateRequest`
- `serveradapter.RunStdio` → `mcpstdio.RunStdio`
- Keeps `serveradapter.Run` for HTTP `till serve` (deletes in W7.D3).

`cmd/till/main_test.go` performs the same set of rewrites symmetrically.

**Verdict:** REFUTED for live imports. NITs (N3 below) for stale doc-comments/test-fixture strings.

### H3 — Test preservation (auth-mutation tests intact?)

**Hypothesis:** `cmd/till/main_test.go` lost auth-mutation tests.

**Test:** `git diff cmd/till/main_test.go` review — search for removed `func Test*` lines.

**Finding:** The diff is +27 / -28, all symmetric `serveradapter.*`→`mcpcommon.*` and `servercommon.*`→`mcpcommon.*` rewrites. The -1 net delta is from collapsing two imports (`serveradapter` + `servercommon`) into one (`mcpcommon`). Zero `func Test*` definitions deleted. `TestAuthorizeMutation*` family (lines 1300-1500) preserved; only type qualifiers changed.

**Verdict:** REFUTED.

### H4 — `till mcp` + `till capture-state` still work

**Hypothesis:** CLI routes for `till mcp` and/or `till capture-state` are broken.

**Test:** Read `cmd/till/main.go` `runMCP` (line 2682), `runServe` (line 2653), `runCaptureState` (line 2757); wire-tracing.

**Finding:**
- `runMCP` → `mcpCommandRunner(ctx, mcpcommon.Config{...}, mcpcommon.Dependencies{...})` → `mcpstdio.RunStdio(...)` → `mcprpc.ServeStdio(mcprpc.Config{...}, captureState, attention)`. ✓
- `runServe` → `serveCommandRunner(ctx, mcpcommon.Config{...}, ...)` → `serveradapter.Run(...)` → `server.NewHandler(...)` → `mcprpc.NewHandler(...)` + `httpapi.NewHandler(...)`. ✓
- `runCaptureState` → `mcpcommon.NewAppServiceAdapter(svc, authSvc)` → `adapter.CaptureState(ctx, mcpcommon.CaptureStateRequest{...})`. ✓

`mage test-pkg ./cmd/till` returned 281/281 pass — covers all three subcommand wiring tests.

**Verdict:** REFUTED.

### H5 — File-level split of `server.go` integrity

**Hypothesis:** The split of `server.go` is incorrect (e.g., `Config`/`Dependencies`/`NormalizeConfig` ended up in the wrong package, or `Run` doesn't call the new import).

**Test:** Read `internal/adapters/server/server.go` and `internal/adapters/mcp_common/server_config.go`.

**Finding:**
- `mcp_common/server_config.go` contains: `Config` (struct), `Dependencies` (struct), `defaultBindAddress` (const, package-private), `NormalizeConfig` (func, exported), `NormalizeEndpoint` (func, exported). ✓
- `server/server.go` line 23-57 `NewHandler(cfg mcpcommon.Config, deps mcpcommon.Dependencies) (http.Handler, mcpcommon.Config, error)` correctly calls `mcpcommon.NormalizeConfig(cfg)` at line 24. ✓
- `server/server.go` line 60 `Run` calls `NewHandler(cfg, deps)` which transitively calls `mcpcommon.NormalizeConfig`. ✓
- `mcp_stdio/stdio.go` line 23 `RunStdio` calls `mcpcommon.NormalizeConfig(cfg)` directly. ✓
- `mcp_rpc/handler.go` retains its own `normalizeConfig(cfg)` for `mcprpc.Config` (a separate, narrower config type) — verified independently namespaced; not the same function.

**Verdict:** REFUTED.

### H6 — Cyclic-import concern

**Hypothesis:** `mcp_common` imports `mcp_rpc` or `mcp_stdio` (creating a cycle since both import `mcp_common`).

**Test:** `git grep -l 'internal/adapters/(mcp_rpc|mcp_stdio|server)' internal/adapters/mcp_common/`.

**Finding:** Zero hits. `mcp_common` imports only `internal/app`, `internal/domain`, `internal/adapters/auth/autentauth`, `internal/adapters/storage/sqlite` (per its actual file contents) — none of the three sibling adapters.

Import graph (verified):
- `mcp_common` → no sibling adapters. ✓
- `mcp_rpc` → `mcp_common` (one direction). ✓
- `mcp_stdio` → `mcp_common`, `mcp_rpc` (one direction each). ✓
- `server` → `mcp_common`, `mcp_rpc`, `server/httpapi` (all one-direction). ✓

DAG: `cmd/till → {server, mcp_stdio} → mcp_rpc → mcp_common`. No cycle.

**Verdict:** REFUTED.

### H7 — `mcpapi/` internal imports rewritten

**Hypothesis:** A file in the new `mcp_rpc/` still imports `internal/adapters/server/common`.

**Test:** `git grep -n 'internal/adapters' internal/adapters/mcp_rpc/`.

**Finding:** Zero live import hits for `server/common`. Every production file (`auth_context_runtime.go`, `extended_tools.go`, `handler.go`, `handoff_tools.go`, `instructions_explainer.go`) imports `internal/adapters/mcp_common`. Tests retain the alias `servercommon` pointing at the new path (`servercommon "github.com/evanmschultz/tillsyn/internal/adapters/mcp_common"`) — path correct, alias stale-named. NIT.

Two remaining `server/common` mentions inside `mcp_rpc/` are doc-comments (handler_steward_integration_test.go:184) and one test-fixture string (extended_tools_test.go:465). Already counted under H2 / N3.

**Verdict:** REFUTED for live imports. NIT for stale `servercommon` alias name (N1 below).

### H8 — Stale doc-comments still mention old paths

**Hypothesis:** Builder claimed NIT-level doc-comment staleness; some are actually live imports.

**Test:** Per H2 enumeration, classify each remaining hit as live-import or doc-only.

**Finding:** All 9 non-`cmd/till` and non-`server` hits are `//` Go comments or `#` TOML comments or test-fixture data strings. Zero are live imports. Builder's NIT-level classification is correct.

**Verdict:** REFUTED for severity. NIT (N3) for cleanup.

### H9 — Package-private symbol leakage

**Hypothesis:** A symbol that was package-private in `server/common/` (lowercase) got exported (uppercase) during the move to `mcp_common/`.

**Test:** `git show HEAD:internal/adapters/server/server.go` (pre-extraction) vs `internal/adapters/mcp_common/server_config.go` (post). Spot-check.

**Finding:** Two intentional exports for cross-package access:
- `normalizeConfig` → `NormalizeConfig`. Required by `server.NewHandler` and `mcpstdio.RunStdio` both calling it from outside `mcp_common`. Unavoidable without code duplication.
- `normalizeEndpoint` → `NormalizeEndpoint`. Same reason; transitively called by both transports' normalize paths.

`defaultBindAddress` (const, lowercase) stayed package-private — only consumed inside `NormalizeConfig`. ✓
`defaultShutdownTimeout` (const, lowercase) stayed in `server.go` (also package-private). ✓

Across the `common/` → `mcp_common/` mass rename (32 files), I did not spot any other identifier-case changes in the diff. Diff was overwhelmingly `common.X` → `mcpcommon.X` qualifier rewrites with no symbol-case flips.

**Verdict:** REFUTED. The two exports are necessary and minimal.

### H10 — Test signal (count regression?)

**Hypothesis:** Some extracted tests aren't running (file moved but test count dropped).

**Test:** `mage test-pkg ./internal/adapters/mcp_common`, `./internal/adapters/mcp_rpc`, `./internal/adapters/mcp_stdio`, `./cmd/till`.

**Finding:**
- `mcp_common`: 165/165 passing. ✓ (matches builder claim)
- `mcp_rpc`: 226/226 passing. ✓ (matches builder claim)
- `mcp_stdio`: 0 tests / 1 package (skipped — no `*_test.go` files). NIT.
- `cmd/till`: 281/281 passing. ✓ (matches builder claim)

The `mcp_stdio` package has zero tests because its single function `RunStdio` is exercised at the `cmd/till` integration layer through the `mcpCommandRunner` seam. The pre-extraction `serveradapter.RunStdio` was also not directly unit-tested — this is a pre-existing coverage pattern, not a regression. NIT.

**Verdict:** REFUTED for regression. NIT (N2) for missing direct unit test on `mcpstdio.RunStdio`.

### H11 — YAGNI on `mcp_stdio`

**Hypothesis:** The `mcp_stdio` package is a 43-line shim that should have stayed in `mcp_rpc` or `cmd/till`.

**Test:** Read `internal/adapters/mcp_stdio/stdio.go`; cross-reference inventory §1.4 recommendation; check PLAN.md acceptance criteria.

**Finding:**
- File is 43 lines, single function `RunStdio`.
- Inventory §1.4 explicitly recommended keeping `ServeStdio` inside `mcp_rpc/` and dropping the `mcp_stdio` package idea.
- PLAN.md L1 W7.D2 Paths line 653 explicitly names `internal/adapters/mcp_stdio/` as a NEW package; Acceptance line 662 explicitly says "Every file/symbol classified as `stdio-relevant` in W7_INVENTORY.md exists in `internal/adapters/mcp_stdio/`."

Plan-vs-inventory conflict resolved in favor of plan (which is the build contract). The architectural justification (anticipated TILL-SERVE-R1 HTTP transport rebuild via `mcp_rpc/` with stdio cleanly isolated) is documented in the W7.D2 decision contextblock. YAGNI smell present, but plan-compliant.

**Verdict:** REFUTED (plan-compliant). NIT (N4) flag for future review: if TILL-SERVE-R1 never lands, `mcp_stdio` should be absorbed into `mcp_rpc`.

### H12 — W7.D3 blast-radius

**Hypothesis:** W7.D2's "leave http-residue in place" is incompatible with W7.D3's "directory does NOT exist post-deletion."

**Test:** Inventory the remaining `server/` contents (H1) against W7.D3's acceptance.

**Finding:** Remaining in `internal/adapters/server/`:
- `server.go` (Run + NewHandler + writeHealthStatus + defaultShutdownTimeout) — all http-residue.
- `httpapi/` (handler.go + 2 test files) — all http-residue.

W7.D3 can `git rm -r internal/adapters/server/` cleanly. The `serveradapter` import in `cmd/till/main.go:25` is W7.D3's responsibility to remove (its KindPayload explicitly covers `cmd/till/main.go` for "remove cobra serve subcommand registration + runServe function + any remaining internal/adapters/server imports").

**Verdict:** REFUTED. W7.D2 leaves exactly the deletion target W7.D3 expects.

---

## Unmitigated Counterexamples

None found.

---

## NITs

### N1 — Stale `servercommon` alias name in mcp_rpc test files
**Severity:** low (cosmetic)

`internal/adapters/mcp_rpc/handler_test.go`, `handler_integration_test.go`, `handler_steward_integration_test.go` use the alias `servercommon "github.com/evanmschultz/tillsyn/internal/adapters/mcp_common"`. Path is correct (`mcp_common`), but the alias name `servercommon` is stale.

**Recommended action:** Rename alias `servercommon` → `mcpcommon` for consistency with `cmd/till/main.go` and the production code in the same package. One-line change per file. Can be deferred to a doc-comment cleanup pass alongside N3.

### N2 — Zero unit tests on `mcpstdio.RunStdio`
**Severity:** low (pre-existing)

`mcp_stdio` package has no `*_test.go` files. `mcpstdio.RunStdio` is only exercised at runtime via the `cmd/till/main.go` `mcpCommandRunner` seam, which is stubbed in tests. The pre-W7.D2 `serveradapter.RunStdio` had the identical coverage pattern — this is not a regression.

**Recommended action:** Optional add a `mcp_stdio/stdio_test.go` that verifies the `mcpcommon.Config` → `mcprpc.Config` field mapping and the nil-dep / nil-ctx guards. Single test function, ~30 lines. Defer to a follow-up pass if not blocking.

### N3 — Stale `server/common` and `server/mcpapi` doc-comments / TOML comments
**Severity:** low (cosmetic, no functional impact)

8 file locations contain doc-comments or TOML comments that reference the pre-extraction paths:
- `internal/app/auth_requests.go:760`
- `internal/app/auto_generate_steward.go:17`
- `internal/app/dispatcher/walker.go:257`
- `internal/app/service.go:1444`
- `internal/domain/errors.go:88`
- `internal/templates/builtin/till-gen.toml:82`
- `internal/templates/builtin/till-go.toml:55`
- `internal/templates/embed_test.go:469`
- `internal/adapters/mcp_rpc/handler_steward_integration_test.go:184`

Also one test-fixture string literal:
- `internal/adapters/mcp_rpc/extended_tools_test.go:465` — `ValidationPlan: "Run mage test-pkg ./internal/adapters/server/mcpapi and mage ci."`

**Recommended action:** Search-and-replace pass (`server/common` → `mcp_common`, `server/mcpapi` → `mcp_rpc`) across these files. Mechanical; can run as part of W7.D3 closeout or as a separate doc-cleanup droplet. Builder explicitly flagged this in completion notes as NIT-acceptable.

### N4 — `mcp_stdio` is a 43-line shim package
**Severity:** low (YAGNI smell, plan-compliant)

The `mcp_stdio` package contains only one function (`RunStdio`) and one file (`stdio.go`). Inventory §1.4 originally recommended consolidating this into `mcp_rpc/`. PLAN.md L1 explicitly required a separate package; builder correctly followed the plan over the inventory.

**Recommended action:** Track as a future refinement. If TILL-SERVE-R1 lands with a separate HTTP transport, the three-package split (`mcp_common`/`mcp_rpc`/`mcp_stdio`) earns its keep. If TILL-SERVE-R1 never lands, absorb `mcp_stdio.RunStdio` into `mcp_rpc/` as `mcprpc.RunStdio` and delete the package. No action in W7.D2 or W7.D3.

### N5 — `defaultBindAddress` dead-write on stdio path
**Severity:** trivial (pre-existing)

`mcpcommon.NormalizeConfig` always sets `cfg.HTTPBind = defaultBindAddress` when empty, even when called from `mcp_stdio.RunStdio` (stdio never binds an HTTP port). The pre-W7.D2 `normalizeConfig` had the same behavior. Cost: one trimspace + one string-assign on every stdio start. Sub-microsecond.

**Recommended action:** None. Track as a TILL-SERVE-R1 cleanup if Config field-split lands.

---

## Verdict rationale

W7.D2 is a large, mechanical extraction (35 files renamed, ~195 qualifier rewrites, one file split). The builder:
1. Followed the W7_INVENTORY.md classification exactly for every file/symbol.
2. Used `git mv` (verified via `git diff --stat -M` showing rename detection on all 32 moved files plus the new `mcp_common/server_config.go` + `mcp_stdio/stdio.go` additions).
3. Updated all live consumers (verified via `git grep` showing zero live imports of `server/common` or `server/mcpapi`).
4. Preserved all auth-mutation tests in `cmd/till/main_test.go` (verified via diff inspection — symmetric qualifier rewrites, zero `func Test*` deletions).
5. Kept `till mcp`, `till serve`, `till capture-state` working (verified via wire-tracing and `mage test-pkg ./cmd/till` 281/281 pass).
6. Left exactly the http-residue W7.D3 expects to delete.
7. Avoided introducing any package-level cycles (verified via grep).
8. Exported only the two symbols (`NormalizeConfig`, `NormalizeEndpoint`) that require cross-package access; kept everything else private.

`mage test-pkg` verified:
- `./internal/adapters/mcp_common` — 165/165 pass
- `./internal/adapters/mcp_rpc` — 226/226 pass
- `./internal/adapters/mcp_stdio` — 0 tests (skipped, NIT N2)
- `./cmd/till` — 281/281 pass

All 12 attack hypotheses REFUTED. Five NITs flagged (N1-N5), all low-severity, none blocking W7.D3.

**Overall: PASS WITH NITS.** Proceed to W7.D3 deletion.
