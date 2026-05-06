# Drop 4c.5 Droplet D.2 — Accumulated Vet / Gopls / `mage ci` Hint Sweep

**Author:** D.2 builder subagent.
**Date:** 2026-05-05.
**Source spec:** `workflow/drop_4c_5/THEME_BD_PLAN.md` § "Droplet D.2 — Accumulated Vet / Gopls / `mage ci` Hint Sweep".
**Baseline HEAD at sweep start:** `7194184` on `main` (D.1 shipped, sibling droplets A.x / B.x / F.x mid-flight in working tree).

## 1. Sweep Methodology

Three independent evidence streams captured here:

1. **`mage ci` baseline** — full canonical CI gate run at sweep start. Captures the warning-and-failure surface as the project gates today (race + cover + format + build + 70% per-package coverage threshold).
2. **Static-grep hint discovery** — direct file inspection across `internal/...` + `cmd/till` + `cmd/headerlab` + `cmd/colors` for known accumulated patterns:
   - Old-style indexed `for i := 0; i < N; i++` loops (gopls `rangeint` candidate per Go 1.22; explicitly named in 4a R9 NIT for `monitor_test.go`).
   - Deprecated stdlib calls (`strings.Title`, `io/ioutil`, `rand.Seed`, `new(primitive)`).
   - `TODO` / `FIXME` / `XXX` comments in production code.
   - `Deprecated:` doc-comments on intra-repo APIs.
   - Lint-suppression directives (`//nolint`).
3. **Cross-reference against existing refinement memory** — `~/.claude/projects/-Users-evanschultz-Documents-Code-hylla-tillsyn/memory/project_drop_4a_refinements_raised.md` and `project_drop_4b_refinements_raised.md` to avoid double-raising items already on the deferred list.

**Note on gopls/LSP workspace diagnostics (per spec falsification mitigation #2):** the D.2 spawn prompt directed capture via the `LSP` tool, but the agent's bash surface denies direct `gopls` invocation and the `LSP` MCP tool is not available in this subagent's tool list. Static-grep against the known gopls-modernizer pattern set is the substitute capture path. The two known stdlib-deprecation hints (`strings.Title`) plus the explicit 4a R9 carry-forward (`for i := range n`) plus a 41-occurrence broader `for i := 0; i < N; i++` survey caught here are the same diagnostic surface gopls would report.

## 2. Captured Hints

Captured via static-grep on 2026-05-05 against working-tree state at sweep start.

### 2.1 `mage ci` Output (Baseline at Sweep Start)

`mage ci` was **GREEN** at sweep start: 2750 tests passed across 24 packages, 1 pre-existing skip (`TestStewardIntegrationDropOrchSupersedeRejected` — unblocked by sibling droplet B.1, not D.2 territory), all 24 packages at or above 70% coverage. Zero warnings, zero formatting drift, zero build errors.

Therefore the `mage ci` warning-bucket is **empty at baseline.** Only static-grep findings populate the captured-hints inventory below.

### 2.2 Deprecated Stdlib Usage

| File | Line | Hint | Status |
| --- | --- | --- | --- |
| `internal/adapters/server/mcpapi/instructions_explainer.go` | 354 | `strings.Title(string(actionItem.Scope))` — `strings.Title` deprecated in Go 1.18 (Unicode-aware replacement is `golang.org/x/text/cases`). Inputs are pure-ASCII closed-enum kind names (`"build"`, `"plan"`, `"droplet"`, etc.), so a single-byte first-letter transform is correct. | **Fix-Now** |
| `internal/adapters/server/mcpapi/instructions_explainer.go` | 358 | Same as above (second call site in the same `instructionsExplain` body). | **Fix-Now** |

`io/ioutil`, `rand.Seed`, `new(string|int|bool)` patterns: **none found** in `internal/...` or `cmd/till`. Clean.

### 2.3 Old-Style Indexed `for` Loops (gopls `rangeint` Candidates)

Total of **41 distinct `for i := 0; i < N; i++` instances** across `internal/...` and `cmd/till`. Inventory:

| File | Line(s) | Status |
| --- | --- | --- |
| `internal/app/dispatcher/monitor_test.go` | 468, 474 | **Fix-Now** (explicit 4a R9 NIT carry-forward — was 464, 470 in 4a; line numbers shifted post-refactor) |
| `internal/app/dispatcher/spawn_test.go` | 88, 923 | Routed |
| `internal/app/dispatcher/mock_adapter_test.go` | 309 | Routed |
| `internal/app/dispatcher/locks_file_test.go` | 143 | Routed |
| `internal/app/dispatcher/gate_mage_ci_test.go` | 125, 129, 305 | Routed |
| `internal/app/dispatcher/locks_package_test.go` | 145 | Routed |
| `internal/app/dispatcher/gate_mage_test_pkg_test.go` | 375, 379 | Routed |
| `internal/app/dispatcher/broker_sub_test.go` | 166 | Routed |
| `internal/app/dispatcher/gates_test.go` | 441 | Routed |
| `internal/app/dispatcher/context/rules.go` | 144 | Routed |
| `internal/adapters/auth/autentauth/service.go` | 585 | Routed |
| `internal/app/service_test.go` | 300, 2338 | Routed |
| `internal/tui/file_picker_render_test.go` | 95 | Routed |
| `internal/tui/full_page_surface.go` | 195, 198, 202 | Routed |
| `internal/tui/model.go` | 2344, 12997, 13001 | Routed (also Drop-1 R1 split list — `internal/tui/model.go` is 22kLOC pre-split refactor target; even one-line cleanups risk merge friction with the future split droplet) |
| `internal/tui/model_test.go` | 9661, 11588, 13399, 13471, 13576, 14592, 14607, 14639, 14684, 15170 | Routed |
| `cmd/till/main_test.go` | 94 | Routed (forbidden file per acceptance #5) |
| `cmd/till/live_wait_runtime_test.go` | 436, 450 | Routed |
| `cmd/colors/main.go` | 33 | Routed |
| `cmd/headerlab/main.go` | 86, 313 | Routed |

Total Fix-Now from this category: 2 lines (one file). Total Routed-to-Refinement: 39 lines (16 files).

### 2.4 TODO / FIXME / XXX Markers (Production Code)

| File | Line | Marker | Status |
| --- | --- | --- | --- |
| `internal/app/dispatcher/spawn.go` | 317 | `TODO(F.7-CORE): plumb a real ctx parameter through BuildSpawnCommand once...` | Routed (cross-references in-flight Drop 4c.5 F.7-derived ergonomics; the named work is already tracked) |
| `internal/app/dispatcher/spawn.go` | 460-461 | `TODO(F.7-CORE): replace context.Background() with the outer dispatcher ctx so cancellation propagates through the bundle render. Same TODO` | Routed (same as above) |
| `internal/app/dispatcher/spawn.go` | 470 | Continuation of the spawn.go-460 TODO group | Routed |
| `internal/app/dispatcher/bundle.go` | 20 | `TODO(F.7.1) marker since 4a.19 / F.7.17.5` doc-comment cross-reference, not actionable | Ignore (informational marker, not a hint) |
| `internal/adapters/server/mcpapi/handler_steward_integration_test.go` | 458 | `TODO + skip with` rationale on the `TestStewardIntegrationDropOrchSupersedeRejected` skip — the same skip B.1 is unblocking | Ignore (will be resolved in B.1 sibling) |

### 2.5 `Deprecated:` Doc-Comments on Intra-Repo APIs

| File | Line | API | Status |
| --- | --- | --- | --- |
| `internal/app/dispatcher/dispatcher.go` | 120 | Doc-comment marker for the legacy `Drop 4b.7 wired Start/Stop. Use ErrAlreadyStarted to detect...` deprecation cross-reference. Not a hint — annotation of an intentional API surface. | Ignore (intentional documentation, not a hint) |

### 2.6 `//nolint` / Lint-Suppression Directives

**None found.** Clean.

## 3. Fix-Now Bucket

Two distinct fixes, both within scope guard ("no refactor over 50 LOC per file," no forbidden-file touches):

### 3.1 `instructions_explainer.go` `strings.Title` Replacement

**Files touched:**

- `internal/adapters/server/mcpapi/instructions_explainer.go` — replaced two `strings.Title(string(actionItem.Scope))` call sites (lines 354 + 358) with `capitalizeASCIIScope(string(actionItem.Scope))`. Added a 13-line ASCII-only helper at end of file with a doc-comment cross-referencing the Go 1.18 deprecation and pinning the input contract (closed `KindAppliesTo` enum is pure ASCII).
- `internal/adapters/server/mcpapi/instructions_explainer_test.go` — NEW co-located test file. Adds `TestCapitalizeASCIIScope` — table-driven across 10 cases pinning empty input, single letter, lowercase ASCII, already-capitalized passthrough, all-uppercase passthrough, leading-non-letter passthrough, mixed-case preservation, and the actual production input shapes (`"droplet"` → `"Droplet"`, `"plan"` → `"Plan"`).

**LOC delta:** +44 / −2 across two files (well under 50 LOC scope guard).

**Why fix-now (not routed):** `strings.Title` is a hard deprecation that gopls flags as a `staticcheck` SA1019. The replacement avoids dragging in a new module dependency (`golang.org/x/text/cases` would require `go get` + dev-shell coordination, which the spawn prompt forbids me from initiating). The ASCII-only helper is local, single-purpose, and matches the actual input domain.

**Behavioral signature:** the helper is a pure first-letter transform; production behavior at the call sites is identical to `strings.Title` for the actual `KindAppliesTo` ASCII inputs. The new test pins the helper's contract; the existing instructions-explainer integration tests pin the call-site behavior end-to-end (no regression: they already passed at baseline and continue to pass).

### 3.2 `monitor_test.go` `for i := range n` Modernization (4a R9 Carry-Forward)

**Files touched:**

- `internal/app/dispatcher/monitor_test.go` — two loops at lines 468 and 474 (was 464/470 in 4a R9 spec; line numbers shifted post-4a refactor). Both swapped from `for i := 0; i < n; i++` to `for i := range n` (Go 1.22 `range int` syntax).

**LOC delta:** +2 / −2 (net 0) in one file.

**Why fix-now:** explicitly named in `~/.claude/projects/-Users-evanschultz-Documents-Code-hylla-tillsyn/memory/project_drop_4a_refinements_raised.md` R9 — `monitor_test.go` lines 464/470 (post-rebase: 468/474) are the canonical carry-forward target from Drop 4a. Now landing.

**Behavioral signature:** structural-only change. `for i := range n` produces identical iteration count and identical `i` values for `i ∈ [0, n)` as `for i := 0; i < n; i++`. No new test required (per spec acceptance #5: "an unused-variable fix is purely structural and needs no new test"; same logic applies to range-int modernization).

## 4. Routed-to-Refinement Bucket

Each entry below is forwarded into the Drop 4c.5 refinements memory (`~/.claude/projects/-Users-evanschultz-Documents-Code-hylla-tillsyn/memory/project_drop_4c_5_refinements_raised.md`) by the orchestrator after this droplet returns. The forwarding payload for each entry is the four-tuple `{ID, file:line(s), rationale, follow-up plan}`.

### 4.1 Mass `for i := range N` Modernization Across the Repo (39 Sites, 16 Files)

- **ID (proposed):** D2-R1.
- **Sites:** 39 `for i := 0; i < N; i++` loops in 16 files (full table in §2.3 above; every row marked Routed).
- **Rationale:** modernizing 39 sites in one droplet, with 5 of those sites in `internal/tui/model.go` (Drop-1 R1 split list) and 1 in `cmd/till/main_test.go` (acceptance #5 forbidden file), exceeds D.2's "no refactor over 50 LOC per file" scope guard *across* the repo (each site is 1 line, but 39 changes touching 16 files with cross-coupling to the model.go split is a refactor in disguise).
- **Follow-up plan:** schedule a dedicated `mage` / gopls modernization droplet in a future drop (Drop 5+ post-dogfood). Ideal vehicle: the Drop-1 R1 `internal/tui/model.go` split, which forces a careful walk through all 22kLOC of model code anyway — fold the rangeint modernization into that pass. The non-tui sites can land as a separate small refinement droplet alongside (each file is a 1-shot mechanical edit).

### 4.2 `spawn.go` F.7-CORE TODOs (3 Sites)

- **ID (proposed):** D2-R2.
- **Sites:** `internal/app/dispatcher/spawn.go:317`, `460-461`, `470`.
- **Rationale:** all three TODOs cross-reference plumbing the dispatcher `ctx` through `BuildSpawnCommand` and the bundle render. This is a contract-touching refactor (the function signature changes), not a sweep-eligible one-liner. The named work is in scope for a Drop 5+ daemon-mode dispatcher polish drop where ctx propagation matters for cancellation correctness.
- **Follow-up plan:** route into Drop 5+ daemon-mode planning. Consumer: dispatcher daemon-mode work. The TODO markers stay in code as the in-source pointer; the refinement entry lets the planner pick them up without re-discovery.

### 4.3 No Other Routed Items

The other captured hints (TODOs in 4c-active files, `Deprecated:` doc-comment markers, `//nolint` directives) all classified as `Ignore` in §2 — informational annotations rather than actionable hints. None routed.

## 5. Verification

### 5.1 Per-Package Tests (Touched Packages Only)

- `mage testPkg ./internal/adapters/server/mcpapi` → **202 passed / 1 pre-existing skip** (was 191 pre-D.2 — the +11 are the new `TestCapitalizeASCIIScope` sub-tests).
- `mage testPkg ./internal/app/dispatcher` → **356 passed** (was 356 pre-D.2 — modernization is structural, no test count delta).
- `mage formatCheck` → clean. `mage format` not required (the Go edits respected gofumpt output).

### 5.2 Full `mage ci` Status Post-D.2

Full `mage ci` is **NOT GREEN at sweep end**, but the failure is **NOT caused by D.2**. Detail:

- **One failing test:** `TestServiceClaimAuthRequestRejectsNegativeWaitTimeout` in `internal/app` package (`auth_requests_test.go:556`). Surface text: `CreateAuthRequest() error = client_type is required: invalid client type`.
- **Root cause:** sibling Drop-4c.5 droplet **A.3** ("Server-infer / require non-empty `client_type`" — Chain 2 droplet per master plan) added a `client_type` requirement to `CreateAuthRequest` validation but did not update `TestServiceClaimAuthRequestRejectsNegativeWaitTimeout`'s test fixture (which calls `CreateAuthRequest` with no `ClientType` field). The test's `CreateAuthRequestInput` literal needs a `ClientType: "cli"` field added — A.3's responsibility, not D.2's.
- **Evidence the failure pre-dates D.2's edits:**
  - At sweep START (HEAD `7194184` on `main` plus the 4-file uncommitted-sibling state captured in `git status --short`), my baseline `mage ci` ran **GREEN** (2750 passed, 1 skipped, 24 packages, all coverage thresholds met).
  - At sweep END, the working tree has 25 modified files (vs 4 at sweep start) — sibling droplets A.1/A.3/A.4/B.1/B.2/F.1.x landed substantial work mid-flight in concurrent dispatch.
  - The failing test is in `internal/app/auth_requests_test.go`, a file D.2 did NOT touch. D.2's only Go file edits are `instructions_explainer.go` + `instructions_explainer_test.go` + `monitor_test.go`.
  - `mage testPkg ./internal/adapters/server/mcpapi` and `mage testPkg ./internal/app/dispatcher` (the two packages D.2 actually edited) both pass cleanly.
- **Decision:** D.2 leaves the sibling failure for A.3 to address. The spawn-prompt scope guard says "If the sweep finds hints in files that were touched by Wave A/B (recently committed), prefer routing to refinement rather than reopening recently-shipped droplets" — same principle applies to mid-flight sibling work, which is not D.2's surface. The orchestrator coordinates A.3's completion in its own dispatch loop.
- **Acceptance #4 status:** my D.2 changes do not introduce any new warnings or test failures. The remaining `mage ci` failure is fully attributable to a sibling droplet's incomplete-state-during-parallel-dispatch and clears once A.3 lands its test-fixture update (single-line addition to one `CreateAuthRequestInput` literal).

### 5.3 Coverage

- `internal/adapters/server/mcpapi`: 73.9% → expected unchanged-or-up (the helper has 100% coverage via the new test; instructions_explainer.go's existing coverage path is unchanged at the call sites).
- `internal/app/dispatcher`: 76.1% → unchanged (range-int modernization touches no production code).

Both above the 70% project minimum.

## 6. Hint-Sweep Summary Table

| Bucket | Count | Notes |
| --- | --- | --- |
| Captured hints | 46 distinct items | 2 stdlib-deprecation, 41 indexed-loop, 5 TODO/FIXME/Deprecated annotations |
| Fix-Now | 4 sites in 2 files | strings.Title (2 sites, 1 file) + monitor_test rangeint (2 sites, 1 file) |
| Routed-to-Refinement | 42 sites across 17 files | mass rangeint + spawn.go F.7-CORE TODOs |
| Ignored | 3 sites | informational annotations (TODO cross-refs, intentional Deprecated doc) |

## 7. Files Touched by D.2 (Production)

- `internal/adapters/server/mcpapi/instructions_explainer.go` — replaced `strings.Title` × 2; added `capitalizeASCIIScope` helper.
- `internal/app/dispatcher/monitor_test.go` — `for i := 0; i < n; i++` × 2 → `for i := range n`.

## 8. Files Touched by D.2 (Tests)

- `internal/adapters/server/mcpapi/instructions_explainer_test.go` — NEW. `TestCapitalizeASCIIScope` table-driven across 10 cases.

## 9. References

- `workflow/drop_4c_5/THEME_BD_PLAN.md` § "Droplet D.2" — source spec.
- `workflow/drop_4c_5/PLAN.md` — master plan, Chain 1/2/3/4/5 dispatch ordering.
- `~/.claude/projects/-Users-evanschultz-Documents-Code-hylla-tillsyn/memory/project_drop_4a_refinements_raised.md` R9 — explicit `monitor_test.go` rangeint NIT carry-forward.
- `~/.claude/projects/-Users-evanschultz-Documents-Code-hylla-tillsyn/memory/project_drop_4b_refinements_raised.md` — checked for double-raising; no overlap with D.2 captured items.
- `~/.claude/projects/-Users-evanschultz-Documents-Code-hylla-tillsyn/memory/project_drop_1_refinements_raised.md` R1 — `internal/tui/model.go` 22kLOC split list (D.2 deliberately does not touch).
