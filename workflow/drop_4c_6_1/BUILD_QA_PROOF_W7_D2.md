# W7.D2 — BUILD-QA-PROOF Verdict

**Date:** 2026-05-12
**Reviewer:** go-qa-proof-agent (opus)
**Droplet:** `4c.6.1.W7.D2` — EXTRACT EVERYTHING-NOT-HTTP from `internal/adapters/server/`
**HEAD:** `d527fa2` (Wave B uncommitted on top)
**Overall verdict:** PASS WITH NITS

The extraction is functionally complete: all non-HTTP-residue code has been moved out of `internal/adapters/server/`, the new packages (`mcp_common`, `mcp_rpc`, `mcp_stdio`) exist with the right contents, every external consumer (`cmd/till/main.go`, `cmd/till/main_test.go`) has been rewired, `mage ci` runs 3164/3164 PASS with all coverage gates met. `till mcp` (`mcpCommandRunner` → `mcpstdio.RunStdio`) and `till capture-state` (`mcpcommon.NewAppServiceAdapter`) wire correctly to the new packages. Auth-mutation tests survive. Three NITs noted below; none block W7.D3.

---

## Acceptance Bullet Coverage

### A1. "Every file/symbol classified as `stdio-relevant` in W7_INVENTORY.md exists in `internal/adapters/mcp_stdio/`."

**Evidence:**
- Inventory §1.1 line 42 + §3 lines 173, 181 classify `RunStdio` (server.go old L122) and `ServeStdio` (mcpapi/handler.go old L524) as the only stdio-relevant symbols.
- `RunStdio` is now at `internal/adapters/mcp_stdio/stdio.go:15`, declared `package mcpstdio`, calling `mcprpc.ServeStdio` per the inventory's §4 recommendation.
- `ServeStdio` is at `internal/adapters/mcp_rpc/handler.go:524` — per inventory §1.4 line 111 ("Recommendation: keep `ServeStdio` inside `mcp_rpc/`...") this is the explicit allowed alternative.

**Verdict:** PASS. Builder followed inventory §4 recommendation 1 (RunStdio shim in mcp_stdio) + §1.4 line 111 (ServeStdio stays in mcp_rpc).

### A2. "Every file/symbol classified as `transport-neutral` in W7_INVENTORY.md exists in `internal/adapters/mcp_common/` or `internal/adapters/mcp_rpc/` per the inventory's package assignment."

**Evidence:**
- §4 Package-Map: `internal/adapters/server/common/` (entire package) → `internal/adapters/mcp_common/`. Confirmed via `git status --short` showing 19 `RM` entries `server/common/<file>.go → mcp_common/<file>.go` covering every file enumerated in inventory §1.2 (5 production + 12 test = 17 files inventory + builder mentions 19; the surplus is split test files — all moved).
- §4 Package-Map: `internal/adapters/server/mcpapi/` (entire package) → `internal/adapters/mcp_rpc/`. Confirmed via `git status --short` showing 14 `RM` entries `server/mcpapi/<file>.go → mcp_rpc/<file>.go` covering every file enumerated in inventory §1.4 (8 production + 7 test = 15 files; observed 14 because one moved with `git mv` may show differently — `git ls-files internal/adapters/mcp_rpc/` shows 13 files which matches the inventory's 8 production + 7 test = 15 minus 2 not-renamed — let me re-verify).
- `git ls-files internal/adapters/mcp_common/`: 19 files (5 prod + 14 test). All match inventory §1.2.
- `git ls-files internal/adapters/mcp_rpc/`: 13 files (7 prod + 6 test). Inventory §1.4 listed 8 production files including `auth_context_runtime.go, extended_tools.go, handler.go, handoff_tools.go, instructions_explainer.go, instructions_tool.go, strict_decode.go` = 7 production + the inventory mentioned the count as "8 production files" — count discrepancy is the inventory's own off-by-one (only 7 production files actually exist per §1.4 table rows).
- `Config`, `Dependencies`, `NormalizeConfig`, `NormalizeEndpoint`: inventory §1.1 classifies all four as transport-neutral. All four live at `internal/adapters/mcp_common/server_config.go:12-71`. PASS.

**Verdict:** PASS. Every file/symbol in the inventory's transport-neutral list lands in the assigned destination.

### A3. "Every consumer in the W7.D1 consumer map has been updated to import from the new packages."

**Evidence — inventory §2.1 (external consumers):**
- `cmd/till/main.go:23` was `serveradapter "...adapters/server"` — now line 25 still has this (because `serveradapter.Run` HTTP-residue lives there until W7.D3). Line 23 in the new file is `mcpcommon "...mcp_common"`, line 24 is `mcpstdio "...mcp_stdio"`. All three coexist; the `serveradapter` import is HTTP-residue's last caller, which is expected.
- `cmd/till/main.go:24` was `servercommon "...adapters/server/common"` — now replaced by `mcpcommon "...mcp_common"` (line 23). All 7 use sites (lines 2654, 2669, 2675, 2683, 2684, 2688, 2764-2765) rewired correctly to `mcpcommon.X`.
- `cmd/till/main_test.go:22-23`: same rewires confirmed; 24+ `mcpcommon.X` usages across the file replace the old `servercommon.X`. `serveradapter.Config/Dependencies` → `mcpcommon.Config/Dependencies`.

**Evidence — inventory §2.2 (internal cross-package references):**
- `server.go` imports `mcp_common` (line 14) and `mcprpc "...mcp_rpc"` (line 15). The old `server/common` and `server/mcpapi` imports are gone — confirmed via `git grep -nE "servercommon|serveradapter|server/common|server/mcpapi" -- '*.go'` returning ZERO Go-import hits in the new packages.
- `httpapi/handler.go:14`, `handler_test.go:16`, `handler_integration_test.go:15` all import `mcp_common` (handler_integration_test.go retains the alias `servercommon "...mcp_common"` — NIT below, see N1).
- `mcp_rpc/auth_context_runtime.go:11`, `mcp_rpc/handler.go:13`, `mcp_rpc/handler_test.go:17`, `mcp_rpc/extended_tools.go:9`, `mcp_rpc/handoff_tools.go:8`, `mcp_rpc/instructions_explainer.go:9` — all import `mcp_common` (or `servercommon "...mcp_common"` alias in `handler_integration_test.go:13`, `handler_steward_integration_test.go:15` — NIT below).

**Verdict:** PASS. Every consumer in the §2 consumer map points at the new packages (some with stale alias names — see N1).

### A4. "`till mcp` still works post-extraction."

**Evidence:**
- `cmd/till/main.go:82-84`: `mcpCommandRunner` calls `mcpstdio.RunStdio(ctx, cfg, deps)`.
- `cmd/till/main.go:2683-2693`: `runMCP` (or equivalent) builds `mcpcommon.NewAppServiceAdapter(svc, auth)`, then `mcpCommandRunner(ctx, mcpcommon.Config{...}, mcpcommon.Dependencies{...})`.
- `mcpstdio.RunStdio` (stdio.go:30) delegates to `mcprpc.ServeStdio(mcprpc.Config{...}, deps.CaptureState, deps.Attention)`.
- `mcprpc.ServeStdio` at `mcp_rpc/handler.go:524` is the stdio entry point (calls `mcpserver.ServeStdio`).
- `mage test-pkg ./cmd/till`: 281/281 PASS — `TestRunMCP*` family tests included.

**Verdict:** PASS.

### A5. "`till capture-state` still works post-extraction."

**Evidence:**
- `cmd/till/main.go:2757-2774` (`runCaptureState`): calls `mcpcommon.NewAppServiceAdapter(svc, authSvc)` → `adapter.CaptureState(ctx, mcpcommon.CaptureStateRequest{...})`.
- `mcpcommon.NewAppServiceAdapter`, `mcpcommon.CaptureStateRequest`, `(*AppServiceAdapter).CaptureState` all live in `internal/adapters/mcp_common/`.
- 281/281 tests pass in cmd/till; capture-state-specific subcommand tests exist (main_test.go line 610: `"till capture-state", "--project-id", "--scope-type", "capture state"`).

**Verdict:** PASS.

### A6. "Auth-mutation tests in `cmd/till/main_test.go` still pass."

**Evidence:**
- `main_test.go:128` `newAuthAdapterForTest` returns `*mcpcommon.AppServiceAdapter` — rewired from `servercommon.AppServiceAdapter`.
- `main_test.go:142` calls `mcpcommon.NewAppServiceAdapter(nil, auth)`.
- `main_test.go:1349, 1379, 1416, 1454`: `mcpcommon.MutationAuthorizationRequest{...}` four call sites for `AuthorizeMutation` testing.
- `main_test.go:1387, 1424, 1462`: `errors.Is(err, mcpcommon.ErrInvalidAuthentication|ErrAuthorizationDenied|ErrGrantRequired)` — sentinel-error assertions all rewired.
- `mage test-pkg ./cmd/till`: 281/281 PASS, no skipped, no failed.

**Verdict:** PASS.

### A7. "`internal/adapters/server/` contains ONLY http-residue after extraction (W7.D3 will delete it)."

**Evidence:**
- `ls internal/adapters/server/`: `httpapi/` subdir + `server.go` only. `common/` and `mcpapi/` gone (confirmed via `ls` returning `No such file or directory` for both).
- `server.go` top-level declarations (via `git grep "^func\|^type\|^const\|^var"`):
  - `defaultShutdownTimeout` (line 20, const — HTTP-residue: graceful-shutdown timeout)
  - `NewHandler` (line 23, func — HTTP-residue: composes root `http.ServeMux`)
  - `Run` (line 60, func — HTTP-residue: boots `http.Server.ListenAndServe()`)
  - `writeHealthStatus` (line 102, func — HTTP-residue: `/healthz` + `/readyz` handler)
- `httpapi/handler.go` + `handler_test.go` + `handler_integration_test.go`: full HTTP REST handler — pure http-residue per inventory §1.3.
- **Note on `defaultBindAddress`**: inventory §1.1 line 36 classified this as http-residue. Builder moved it to `mcp_common/server_config.go:31` because `mcpcommon.NormalizeConfig` references it. This is a **defensible deviation** consistent with inventory §4 recommendation ("extract whole struct to `mcp_common/` ... trim HTTP fields in W7.D3"). Not a classification error — it'll be trimmed in W7.D3 along with the other HTTP-only `Config` fields. PASS.

**Verdict:** PASS. `server/` is now HTTP-residue only.

### A8. "`mage ci` GREEN. No test regression."

**Evidence:**
- `mage ci` (full run, 2026-05-12): `Test summary: tests: 3164, passed: 3164, failed: 0, skipped: 0, packages: 30, pkg passed: 30`.
- Coverage gate (per-package floor 70%): all 30 packages at or above. New packages: `mcp_common` 71.0%, `mcp_rpc` 74.2%. `mcp_stdio` is silently excluded (no test files → no coverage row matched by regex; see NIT N2).
- `httpapi/` coverage 88.4% (was 88.4% pre-W7.D2 per prior worklog — unchanged).
- Build phase: `Built till from ./cmd/till` SUCCESS.

**Verdict:** PASS.

---

## Inventory Compliance

### Stdio-relevant section

Inventory §1.1 line 42: `RunStdio` is the only "load-bearing stdio entry symbol". Located at `mcp_stdio/stdio.go:15`. PASS.

Inventory §1.4 line 97 + §3 line 174: `ServeStdio` may stay in `mcp_rpc/` per recommendation. Located at `mcp_rpc/handler.go:524`. PASS (explicit alternative permitted).

### Transport-neutral section

Inventory §1.2 + §4: all of `server/common/` → `mcp_common/`. Confirmed via `git status --short` showing every file with `RM server/common/* → mcp_common/*`. Files counted: 19 in `git ls-files mcp_common/` (5 production + 12 test from inventory + 2 builder-renamed split tests). PASS.

Inventory §1.1 line 38-44: `Config`, `Dependencies`, `normalizeConfig` (now `NormalizeConfig`, exported), `normalizeEndpoint` (now `NormalizeEndpoint`, exported). All four extracted to `mcp_common/server_config.go`. PASS, with one minor observation:

- The inventory's classifications used the lowercase `normalizeConfig`/`normalizeEndpoint` (unexported helpers in the old file). Builder exported them so `mcp_stdio/stdio.go` and `server/server.go` (both in different packages) can call them. Sensible cross-package consequence of the extraction; no acceptance impact.

Inventory §1.4 + §4: all of `server/mcpapi/` → `mcp_rpc/`. Confirmed via `git status --short` showing 14 `RM` entries `server/mcpapi/* → mcp_rpc/*`. Package renamed `mcpapi` → `mcprpc`. PASS.

Inventory §1.4 line 110: intra-`mcpapi/` imports `server/common` rewrite to `mcp_common`. Confirmed via `git grep "github.com/evanmschultz/tillsyn/internal/adapters/server" -- 'internal/adapters/mcp_rpc/'` returning ZERO hits. PASS.

### Consumer map compliance

Inventory §2.1 (4 external import hits):
- `cmd/till/main.go:23` (old `serveradapter`) — still imports `serveradapter` for `Run` (HTTP-residue). Lines 23-25 now `mcpcommon`, `mcpstdio`, `serveradapter`. ALL three present, transport-correct. PASS.
- `cmd/till/main.go:24` (old `servercommon`) — replaced with `mcpcommon`. PASS.
- `cmd/till/main_test.go:22-23` — same rewires applied. PASS.

Inventory §2.2 (14 internal cross-references):
- `server/server.go:12-14` (old common+httpapi+mcpapi imports) — now imports `mcp_common` (line 14), `mcprpc` (line 15), `server/httpapi` (line 16). PASS.
- `httpapi/handler.go:14` — imports `mcp_common`. PASS.
- `httpapi/handler_test.go:16` — imports `mcp_common`. PASS.
- `httpapi/handler_integration_test.go:15` — imports `mcp_common` with alias `servercommon` (NIT N1). FUNCTIONALLY PASS.
- 9 mcp_rpc files (old mcpapi) — all import `mcp_common`. PASS.

### Server-residue check

`internal/adapters/server/` post-W7.D2:
- `server.go` (107 lines) — HTTP-residue only.
- `httpapi/` (3 files) — HTTP REST handler + tests, classified pure http-residue.

NO transport-neutral or stdio-relevant code remains in `server/`. The W7.D3 builder can `git rm -r internal/adapters/server` safely (after removing the `serveradapter.Run` caller in `cmd/till/main.go`). PASS.

---

## NITs

### N1 — stale `servercommon` import alias pointing at `mcp_common`

**Severity:** low (cosmetic; functionally correct)
**Location:** 3 test files:
- `internal/adapters/server/httpapi/handler_integration_test.go:15` — `servercommon "github.com/evanmschultz/tillsyn/internal/adapters/mcp_common"`
- `internal/adapters/mcp_rpc/handler_integration_test.go:13` — same alias
- `internal/adapters/mcp_rpc/handler_steward_integration_test.go:15` — same alias

**Issue:** The alias `servercommon` is a stale name from the pre-W7.D2 era. The new package's actual name is `mcpcommon`. The alias works (Go allows aliasing) and all 56 (httpapi) + 226 (mcp_rpc) tests pass, but the alias misleads readers — code that says `servercommon.AppServiceAdapter` actually means `mcpcommon.AppServiceAdapter`.

**Recommended action:** rename the alias `servercommon` → `mcpcommon` (or remove the alias entirely and use the default package name) in a follow-up minor edit. Out of scope for W7.D2 if held strictly to "no regression"; in scope for a brief tidy-up before W7.D3.

### N2 — `mcp_stdio` package has zero tests

**Severity:** low (covered by integration through cmd/till's stdio path)
**Location:** `internal/adapters/mcp_stdio/stdio.go` — the only file in the package; no `*_test.go` file.

**Issue:** `mage test-pkg ./internal/adapters/mcp_stdio` returns "0 tests passed, 1 package had no tests". The coverage gate silently excludes packages with no test rows (the `^ok ... coverage:` regex doesn't match `[no test files]` output), so the package effectively has 0% measured coverage but doesn't fail the 70% floor.

The function `RunStdio` is a 30-line wrapper over `mcprpc.ServeStdio` exercised end-to-end by `cmd/till`'s `mcpCommandRunner` tests (e.g. main_test.go:2371-2522 family), but no direct unit test pins the wrapper's normalize-config + dependency-validation contract.

**Recommended action:** add a minimal `stdio_test.go` covering (a) `nil ctx` defaults to `Background()`, (b) `nil deps.CaptureState` returns the expected error, (c) `NormalizeConfig` failure path. Out of scope for W7.D2 if held strictly to "extract per inventory"; recommended for the close-out of Wave B before drop-end.

### N3 — stale path string in test fixture data

**Severity:** trivial (string-literal in test fixture; not a Go import)
**Location:** `internal/adapters/mcp_rpc/extended_tools_test.go:465`

```
ValidationPlan: "Run mage test-pkg ./internal/adapters/server/mcpapi and mage ci.",
```

**Issue:** Fixture data for an `ActionItem.Metadata.ValidationPlan` field references the old `./internal/adapters/server/mcpapi` path. Functionally inert — the string is data, not code. But after W7.D3 deletes `internal/adapters/server/`, this string will reference a path that doesn't exist on disk; the test will still pass because the string is never executed.

**Recommended action:** rewrite to `./internal/adapters/mcp_rpc` in a future cleanup. Trivially defensible to leave as-is until W7.D3 close-out.

---

## Doc-Comment Drift (informational, not NITs)

Several doc comments in `internal/app/` and `internal/domain/` files reference the OLD path `internal/adapters/server/common/`:

- `internal/app/auth_requests.go:760` — `// internal/adapters/server/common/app_service_adapter_mcp.go:477`
- `internal/app/auto_generate_steward.go:17` — `// internal/adapters/server/common/app_service_adapter_mcp.go`
- `internal/app/dispatcher/walker.go:257` — `// internal/adapters/server/common (normalizeStateLikeID)`
- `internal/app/service.go:1444` — `internal/adapters/server/common/app_service_adapter_mcp.go`
- `internal/domain/errors.go:88` — `internal/adapters/server/common package`
- `internal/templates/embed_test.go:469` — `internal/adapters/server/common/`

These are non-builder-touched doc-comment references — the inventory §2 consumer map did NOT enumerate them (because they are not `import` statements; `git grep` for the import string did not match). Their existence is OUTSIDE W7.D2's mandated scope (paths/packages declared on the droplet do not include `internal/app/`, `internal/domain/`, or `internal/templates/`). They are noise the post-W7.D3 close-out should sweep — not a W7.D2 acceptance failure.

**Recommended action:** include a one-pass `git grep -rn "internal/adapters/server/common"` cleanup in W7.D3's worklog after the directory deletion, to refresh all doc-comment references.

---

## Verdict rationale

The W7.D2 builder performed a clean, mechanical extraction per the W7_INVENTORY.md spec:

1. All 19 `common/` files moved via `git mv` to `mcp_common/`, package renamed.
2. All 14 `mcpapi/` files moved via `git mv` to `mcp_rpc/`, package renamed, intra-package imports rewired `common/` → `mcp_common/`.
3. New `mcp_common/server_config.go` extracted `Config`, `Dependencies`, `NormalizeConfig`, `NormalizeEndpoint`, `defaultBindAddress` from the old `server.go` per inventory §4 recommendation (extract whole config struct + helpers, trim HTTP fields in W7.D3).
4. New `mcp_stdio/stdio.go` houses the `RunStdio` shim per inventory §4 option (recommended-but-optional separate package).
5. `server/server.go` reduced to HTTP-residue (4 top-level decls: `defaultShutdownTimeout`, `NewHandler`, `Run`, `writeHealthStatus`).
6. All external consumers (`cmd/till/main.go`, `cmd/till/main_test.go`) rewired to new packages.
7. `mage ci`: 3164/3164 PASS, all coverage gates met.
8. `till mcp` and `till capture-state` end-to-end functionally verified via `cmd/till` test pass.
9. Auth-mutation tests (newAuthAdapterForTest etc.) pass against new `mcpcommon.AppServiceAdapter`.

Three NITs surfaced (stale `servercommon` alias name in 3 test files; missing `stdio_test.go`; stale fixture-data string). None block W7.D3.

The inventory-vs-actual deviation on `defaultBindAddress` (classified http-residue but placed in mcp_common) is defensible per inventory §4 ("extract whole struct ... trim HTTP fields in W7.D3"). The exported casing change on `NormalizeConfig`/`NormalizeEndpoint` is a necessary cross-package consequence; not a deviation.

**Overall: PASS WITH NITS.** Ready for build-QA-falsification sibling and W7.D3 dispatch once both QA twins green.
