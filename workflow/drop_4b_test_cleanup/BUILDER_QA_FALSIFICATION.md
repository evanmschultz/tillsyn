# DROP_4B_TEST_CLEANUP — Builder QA Falsification

Per-round, per-droplet adversarial review of the builder's output. APPEND new sections; never overwrite prior rounds.

## Droplet 1.1 — Round 1

- **Reviewer:** go-qa-falsification-agent
- **Reviewed at:** 2026-05-18
- **Files reviewed:**
  - `internal/domain/comment.go` (D1.1 production change)
  - `internal/domain/comment_test.go` (D1.1 test additions)
- **Callers analyzed (cross-package surface, via `git grep`):**
  - `internal/adapters/storage/sqlite/repo.go:3095-3098` (read path)
  - `internal/app/snapshot.go:1485` (snapshot restore path)
  - `internal/app/search_embeddings.go:72, :186` (subject-ID composition)
  - `internal/adapters/mcp_rpc/extended_tools_test.go:1002, :5959-5974` (test layer)
  - `internal/adapters/mcp_rpc/extended_tools.go:2243` (MCP tool schema enum — committed pre-D1.2 state at HEAD)
- **Build-tool targets:** Did NOT re-run mage (sibling builders dirty); read evidence directly via Read/Grep.
- **Verdict:** **PASS-WITH-FINDINGS**. Core claim ("alias works, 309 tests green") is supported by direct source inspection. Several design-layer findings worth dev disposition; none are blockers on the D1.1 build.

### Attack-by-Attack Verdicts

#### Attack 1 — Caller Breakage

- **Verdict: NIT (no breakage found).**
- **Evidence trace:**
  - **SQL read path** (`repo.go:3095-3098`): reads raw `target_type` column, normalizes via `NormalizeCommentTargetType`, then `IsValidCommentTargetType` check. For rows written post-D1.1 these will always be in canonical `"action_item"` form (write path normalizes before insert via `NormalizeCommentTarget` in `comment.go:122-137`). Normalize is idempotent on canonical input. No breakage.
  - **MCP test stub** (`extended_tools_test.go:1002`): calls `NormalizeCommentTargetType` on `in.TargetType`. The stub accepts the same alias surface; no break.
  - **`IsValidCommentTargetType` directly** (`comment.go:173-176`): now accepts `"actionItem"` and variants. Pre-D1.1 it rejected them.
  - **No UNIQUE constraint** on `target_type` column alone — discriminator paired with `target_id`. Alias normalization does not affect uniqueness.
- **Sub-finding (POSSIBLE / cross-droplet):** `internal/app/snapshot.go:1485` normalizes on snapshot **restore**. A snapshot file containing a literal `"actionItem"` string will silently mutate to `"action_item"` on restore. This is per the F6 acceptance note in PLAN.md (no pre-MVP migration, dev wipes DB), so accepted — but **snapshot restore is a different surface than DB read** and PLAN.md only cites repo.go:3095. Recommend extending the F6 note to cover snapshot.go:1485 explicitly.
- **Sub-finding (POSSIBLE / cross-droplet):** `internal/app/search_embeddings.go:72-74` composes a **stable subject ID** as `project|target_type|target_id` for embedding lookup. Any pre-D1.1 embedding indexed under a literal `"actionItem"` subject ID will now be queried under `"action_item"` — **subject ID drift** → embedding misses on stale data. Same dev-wipe mitigation applies, but worth documenting in F6 alongside the persistence note.

#### Attack 2 — Round-Trip Semantics

- **Verdict: CONFIRMED (design-level, low severity).**
- **Concrete trace:** MCP caller sends `target_type="actionItem"` →
  1. Domain validation `IsValidCommentTargetType("actionItem")` → `true` (post-D1.1).
  2. Write path stores canonical `"action_item"` in SQLite via `NormalizeCommentTarget`.
  3. Read path returns `"action_item"` (canonical), NOT the input form.
- **Counterexample:** an MCP client that round-trip-compares `request.target_type == response.target_type` will see a mismatch. Same for any client constructing a CRDT-style sync key on `target_type`.
- **Repro path:** any integration test invoking `till.comment create target_type=actionItem ...` then `till.comment list ...` and asserting string equality would fail (no such test exists today; this is a hypothetical client contract).
- **Mitigation options:**
  - (a) Add a doc comment to `NormalizeCommentTargetType` explicitly noting "input form is NOT preserved across persistence; consumers always receive the canonical token."
  - (b) Emit a `slog.Warn` / structured deprecation log when an alias lookup hits (see Attack 6).
  - (c) Surface in API docs / WIKI when comment-route documentation lands.
- **Severity rationale:** dev-MVP environment has zero live clients with hard equality contracts; severity climbs once external consumers exist.

#### Attack 3 — Whitespace + Casing Variants

- **Verdict: REFUTED for tested cases; REFUTED for additional whitespace variants (ASCII + Unicode).**
- **Evidence trace** (`comment.go:154-170`): order is `TrimSpace` → `ToLower` → alias-map lookup → canonical-form scan.
  - `" actionItem "` → trim → `"actionItem"` → lower → `"actionitem"` → alias-map hit → `"action_item"`. ✓
  - `"actionItem\n"` / `"actionItem\t"` → `strings.TrimSpace` handles both (stdlib doc: removes all leading/trailing Unicode whitespace including `\t`, `\n`, `\v`, `\f`, `\r`, ` `, ``, ` `). ✓
  - Unicode non-break space `" actionItem "` → `strings.TrimSpace` strips per `unicode.IsSpace`. ✓
- **Builder test already covers** `"whitespace-padded actionItem"` (`comment_test.go:222`). Good.
- **No counterexample found.**

#### Attack 4 — Alias Map Mutation Risk

- **Verdict: NIT (documentation gap, no functional bug).**
- **Evidence trace:**
  - `commentTargetTypeAliases` declared at `comment.go:144-146` as package-level `var`, unexported.
  - `git grep` confirms only one assignment (the declaration) and one read site (`comment.go:161`). No mutations anywhere.
  - Go map reads ARE safe under concurrent read-only access; no race risk in current state.
- **Counterexample (latent / future-drift):** a future contributor adding `commentTargetTypeAliases["foo"] = bar` at `init()` time or later would re-introduce concurrent-read-write risk. Map is **de-facto immutable, not de-jure documented**.
- **Mitigation:** add a one-line doc comment above the var declaration: `// commentTargetTypeAliases is initialized once at package load and treated as immutable thereafter; never mutate at runtime (concurrent map read/write is a data race).` Builder already has `// commentTargetTypeAliases maps lowercased camelCase alias forms...` (line 139-143) — extend that comment with the immutability note.

#### Attack 5 — Test Rigor

- **Verdict: NIT (three minor gaps).**
- **Evidence trace** (`comment_test.go:212-235`):
  - **`t.Parallel()` missing**: the `TestNormalizeCommentTargetTypeAlias` body does not call `t.Parallel()`, nor do the per-subtest `t.Run(...)` bodies. Domain code is purely functional / no shared state — parallelization is safe and would exercise race-detector on the map read path. Other domain tests in the same file (`TestNewCommentDefaultsAndNormalization` at line 9) also don't call `t.Parallel()` — this is package-wide style, not a D1.1 regression. NIT only.
  - **Direct `IsValidCommentTargetType` boolean assertions**: subtests at line 230 assert `IsValidCommentTargetType(tc.input) == true`, so the boolean path IS covered. ✓ This was a concern in the spawn prompt — REFUTED on inspection.
  - **Deterministic input**: all five subtests use literal strings; no `time.Now()` / `rand`. ✓ Deterministic.
  - **Missing edge cases**:
    - Empty string `""` → currently returns `""` per `comment.go:156-158`. Untested directly but covered by `TestNewCommentValidation` indirectly.
    - Pure-whitespace `"   "` → `TrimSpace` → `""` → returns `""`. Untested.
    - `"action_item"` with internal whitespace `"action_ item"` → would lower-case to `"action_ item"`, alias-map miss, canonical-scan miss → returns the unknown-form literal lowercased. NOT in builder's test set; not a regression but an edge case worth documenting.
- **Mitigation:** none required for D1.1's claim. Optional: add a `t.Parallel()` to the outer function (and subtests) and one additional subtest for `""` / `"   "`.

#### Attack 6 — Backwards-Compat / Deprecation Signaling

- **Verdict: POSSIBLE (dev disposition needed).**
- **Evidence trace:** `validCommentTargetTypes = ["project", "action_item"]` is the post-Drop-1.75 closed enum (`comment.go:22-25`). Adding `"actionItem"` as accepted input technically loosens the enum contract — it normalizes back to a closed-enum value, but the *acceptance surface* is widened.
- **Counterexample / contract drift:**
  - Silent acceptance means there's no signal to migrate. A client passing `"actionItem"` will continue to work indefinitely.
  - Per `feedback_parity_clarity_no_silent_failures.md` (dev directive 2026-05-16), the system **prefers explicit-loud-failure or warning logs over silent transforms**. The current implementation transforms silently — no log, no metric, no warning header.
- **Mitigation options:**
  - (a) Add a `slog.Warn("comment_target_type: deprecated alias %q normalized to %q; please use canonical form", input, canonical)` when an alias-map hit occurs.
  - (b) Emit a one-time structured deprecation metric so dashboards can track alias usage.
  - (c) Add a docstring note that the alias is a compat shim, with a planned removal version.
- **Dev disposition needed:** is this in-scope for D1.1 round 2, or queue as a Drop 4b refinement?

#### Attack 7 — Schema-vs-Runtime Drift Residual

- **Verdict: POSSIBLE (D1.2 scope, not D1.1 scope) — pre-existing drift that D1.1 narrows; D1.2 closes it.**
- **Evidence trace:**
  - **Committed state at HEAD** (`extended_tools.go:2243`): MCP schema enum is `("project", "branch", "phase", "actionItem", "subtask", "decision", "note")` — stale pre-1.75 vocabulary.
  - **D1.2 WIP** (uncommitted, sibling-builder dirty): `("project", "action_item", "actionItem")` — closes the drift.
  - **D1.1's effect**: post-D1.1 + pre-D1.2, runtime now accepts `"actionItem"` AND `"action_item"` AND `"project"`. Schema still advertises `"branch"`, `"phase"`, `"subtask"`, `"decision"`, `"note"` — runtime REJECTS these (negative test at `extended_tools_test.go:5959` confirms). Schema-advertised tokens are a SUPERSET of runtime-accepted.
- **Counterexample:** an MCP client introspecting the schema sees seven valid values; sending any of {`"branch"`, `"phase"`, `"subtask"`, `"decision"`, `"note"`} produces `ErrInvalidTargetType`. This drift PRE-EXISTED D1.1 — D1.1 narrowed the surface (alias acceptance added) without touching the schema. D1.2 closes the drift fully.
- **Attribution:** NOT a D1.1 regression — schema/runtime drift existed before D1.1 landed. D1.1 is the half-of-the-fix that the planner separated for atomic-droplet sizing.
- **Mitigation:** D1.2 handles this. No action required for D1.1 round 2. If D1.2 fails plan-QA-falsification, that drift remains; but it remains as a pre-existing condition, not a D1.1-introduced bug.

### Counterexamples Summary

- **CONFIRMED (1):** Attack 2 — round-trip semantics: input form not preserved (canonical token returned regardless of input). Low severity in MVP-dev environment, design-level concern when external clients exist.
- **POSSIBLE (3):**
  - Attack 1 sub-finding — `snapshot.go:1485` and `search_embeddings.go:72/:186` are additional surfaces of the persistence round-trip drift not cited in PLAN.md's F6 note. Recommend extending the F6 note.
  - Attack 6 — silent transformation conflicts with `feedback_parity_clarity_no_silent_failures.md`. Add a `slog.Warn` on alias hit. Dev disposition needed.
  - Attack 7 — schema-vs-runtime drift on legacy tokens. Pre-existing; D1.2 closes. Not D1.1 scope.
- **NIT (2):**
  - Attack 4 — `commentTargetTypeAliases` is de-facto immutable but not documented as such. Add immutability docstring.
  - Attack 5 — `t.Parallel()` missing; some edge cases (`""`, `"   "`) untested. Package-wide style consistency; not a D1.1 regression.
- **REFUTED (2):**
  - Attack 3 — all listed whitespace/casing variants resolve correctly (verified via stdlib `strings.TrimSpace` semantics + builder test cases).
  - Attack 5 partial — direct `IsValidCommentTargetType` boolean assertions ARE present at `comment_test.go:230`.

### Severity + Mitigation Summary

| Attack | Verdict | Severity | Mitigation | Owner |
|--------|---------|----------|------------|-------|
| 1 caller breakage | NIT | low | None required (no breakage); optionally extend F6 note to cover snapshot/embedding paths | Dev disposition |
| 2 round-trip | CONFIRMED | low (MVP) | Doc comment on `NormalizeCommentTargetType` re: input form not preserved | D1.1 round 2 or future refinement |
| 3 whitespace | REFUTED | n/a | n/a | — |
| 4 map mutation | NIT | low | Add immutability docstring to `commentTargetTypeAliases` var | D1.1 round 2 |
| 5 test rigor | NIT | low | Optional `t.Parallel()` + edge-case subtests | D1.1 round 2 or skip |
| 6 deprecation | POSSIBLE | medium (per dev directive) | `slog.Warn` on alias hit OR explicit dev waiver | Dev disposition |
| 7 schema drift | POSSIBLE | medium | Closed by D1.2 (sibling, in_progress) | D1.2 scope |

### Recommendation

**PASS the D1.1 build claim.** The core mechanic — domain alias normalization — works correctly per the builder's tests and matches the planner's acceptance criteria verbatim. The findings above are all (a) design-level / contract-shape concerns the builder's scope did not include, (b) cross-droplet drift surfaces, or (c) NITs that don't invalidate the build.

**Dev disposition on Attack 6** (silent deprecation vs `slog.Warn`) is the only finding worth a round-2 build pass. Attack 7 is sibling-D1.2 scope and resolves naturally. Attacks 1/4/5 are doc/style polish.

### Hylla Feedback

Per `feedback_hylla_disabled_for_now.md` (2026-05-18 directive): Hylla MCP is OFF; no Hylla queries attempted in this review. Fallback evidence sources: `Read`, `git grep` via Bash, `Glob` indirectly via plan-file pointer mapping. No section required per the 2026-05-18 rule.

## Droplet 1.2 — Round 1

- **Reviewer:** go-qa-falsification-agent
- **Reviewed at:** 2026-05-18
- **Files reviewed:**
  - `internal/adapters/mcp_rpc/extended_tools.go` (D1.2 production change — line 2243)
  - `internal/adapters/mcp_rpc/extended_tools_test.go` (D1.2 added regression test — line 5953-5974; pre-existing legacy-token tests at 3514-3568)
  - `internal/domain/comment.go` (D1.1 sibling state — for round-trip and read-path semantics)
  - `internal/adapters/storage/sqlite/repo.go:3095-3097` (read-path normalization)
  - `internal/adapters/mcp_common/types.go:60-69` (`commentTargetTypeFromScope` — adjacent vocab)
  - `internal/adapters/mcp_common/app_service_adapter_auth_context.go:167-177` (auth-context discriminator)
  - `internal/app/snapshot.go:1485` (snapshot restore path)
- **Cross-codebase callers analyzed (`git grep`):**
  - `target_type` literal string: 24 hits — all checked; none recreate the old `project|branch|phase|actionItem|subtask|decision|note` vocabulary in production code. Two pre-existing test fixtures (`extended_tools_test.go:3533`, `:3553`) still pass legacy tokens — see Attack 2.
  - `CommentTargetType*` symbol: 130+ hits — all use post-1.75 `CommentTargetTypeProject` / `CommentTargetTypeActionItem`. Legacy const names (`CommentTargetTypeBranch`/`Phase`/`Subtask`/`Decision`/`Note`) are GONE from the codebase since Drop 1.75 commit `36ef724`.
  - Pipe-separated description string `project|branch|phase|actionItem|subtask|decision|note`: ONE remaining hit — `mcp_common/app_service_adapter_mcp.go:26` (`WhatTillsynIs` prose describing **scope** vocab, NOT comment target_type). Not in D1.2's scope; orthogonal vocabulary.
- **Build-tool targets:** Did NOT re-run `mage` (sibling builders dirty per Round 1 prompt). Evidence via `Read` + `git grep` + `git diff`.
- **Verdict:** **PASS-WITH-FINDINGS**. D1.2's production diff (schema enum shrink + description-string fix) is correct and matches PLAN.md acceptance bullets verbatim. The added regression test does not actually guard the schema change — see Attack 5 — but the production change itself is sound. Several pre-existing test-fixture drifts surface adjacent to D1.2; only one is properly in-scope.

### Attack-by-Attack Verdicts

#### Attack 1 — Caller breakage from enum shrink

- **Verdict: REFUTED at runtime.**
- **Evidence trace:**
  - `mcp.Enum(...)` in this MCP framework (`github.com/mark3labs/mcp-go`) is **schema metadata only**, not a server-side runtime validator. The handler at `extended_tools.go:2287-2335` reads `args.TargetType` as a raw string and forwards it to `comments.CreateComment` / `comments.ListCommentsByTarget` without enum-membership checking. Validation happens downstream in `domain.NewComment` → `NormalizeCommentTarget` → `IsValidCommentTargetType`.
  - The pre-existing test `TestHandlerExpandedCommentToolsForwardHierarchyTargetTypes` (line 3514) sends `target_type="branch"` and `"phase"` to a stub-backed handler. The stub at line 999-1011 normalizes via `NormalizeCommentTargetType` but records the raw `in.TargetType` for assertions. The test asserts `service.lastCreateCommentReq.TargetType != "branch"` (line 3543) and `!= "phase"` (line 3561) — both pass because the stub records the raw input. **No test breaks from D1.2's schema change.**
  - In real production flow (non-stub), `branch`/`phase` etc. would be rejected at `domain.NewComment` with `ErrInvalidTargetType` — but that has been the behavior since Drop 1.75 (`36ef724`), pre-existing D1.2.
- **No counterexample.** D1.2's schema-enum tightening does not break any current test or production caller.

#### Attack 2 — Description string drift in fixtures/tests

- **Verdict: CONFIRMED (test-fixture drift; not D1.2-introduced but exposed by D1.2).**
- **Concrete counterexample:**
  - `internal/adapters/mcp_rpc/extended_tools_test.go:3514` test function name `TestHandlerExpandedCommentToolsForwardHierarchyTargetTypes` references "hierarchy target types" — a concept that does NOT exist post-Drop-1.75. The 12-kind collapse eliminated branch/phase/subtask as comment target levels; only `project` and `action_item` survive.
  - Lines 3533 (`target_type: "branch"`) and 3553 (`target_type: "phase"`) construct test cases for vocabulary that the **MCP schema now explicitly excludes** as of D1.2. The test passes because the stub-backed handler bypasses enum validation — but the test contract is **semantically incoherent** post-D1.2: it asserts the handler "forwards" target_types that the schema doesn't advertise as valid.
  - Function name says "hierarchy target types" → fictional concept post-1.75.
  - Test body asserts forwarding of `"branch"` and `"phase"` → not advertised in schema enum.
  - Schema description string at line 2243: `"project|action_item|actionItem"` (post-D1.2) explicitly excludes these.
- **Repro:** any reader of this test, comparing the test against the schema declaration at line 2243, finds an inconsistency. No `mage` failure, but documentation/contract drift.
- **Severity:** LOW (NIT-class) — the test does cover the forwarding mechanism, just under a stale narrative. The fix is either (a) rename/rewrite the test to use `"project"` / `"action_item"` / `"actionItem"` (the new valid values) and rename the function to drop "Hierarchy", or (b) explicitly mark this as a backward-compat negative-path test ("handler still accepts arbitrary strings; domain layer is the real gate"). Option (a) preserves the spirit of "forwarding tests"; option (b) preserves the existing test name's literal behavior.
- **Owner:** D1.2 round 2 — narrow scope to renaming/rewriting the existing test, OR Drop 4b refinement if dev wants to defer.
- **Note:** D1.2's `paths` declaration includes `internal/adapters/mcp_rpc/extended_tools_test.go`, so this drift IS within the droplet's edit envelope. The builder did edit this file (added the regression test) but did not refactor the pre-existing legacy-token test. This is a missed-cleanup, not a scope-violation.

#### Attack 3 — Round-trip with D1.1 alias + mixed-case schema asymmetry

- **Verdict: POSSIBLE (design-level NIT, not a runtime bug).**
- **Evidence trace:**
  - **Exact-case round trip** for the three enum values (`project`, `action_item`, `actionItem`): schema accepts → handler forwards raw → domain normalizes → stored. For `actionItem`: stored as `action_item` (canonical). For `action_item` and `project`: stored unchanged. On read: `NormalizeCommentTargetType` is idempotent on canonical inputs; round-trip clean.
  - **Mixed-case asymmetry**: client sends `"ActionItem"` (mixed). Schema enum is `("project", "action_item", "actionItem")` — these are JSON-Schema enum literals, exact-match by spec. But the MCP framework (`mark3labs/mcp-go`) does NOT enforce enum at the server; the handler accepts the raw string. Then D1.1's alias converts `"actionitem"` (lowered) → `"action_item"`. So mixed-case is accepted at **runtime** but not advertised in **schema**.
  - **Implication**: a strict MCP client doing client-side schema validation would reject `"ActionItem"` / `"ACTIONITEM"` before the request is sent. A permissive client (most JS / Python MCP SDKs today) would send it through and have it succeed. Two clients see two contracts.
- **Counterexample:** an MCP introspector tool comparing the declared schema to runtime acceptance will observe that runtime accepts variants the schema does not list. **Schema is a subset of runtime acceptance** — a different shape of drift than R5's original (R5 had schema > runtime).
- **Mitigation options:**
  - (a) Document in the description string: `"project|action_item|actionItem (case-insensitive)"` to telegraph the runtime contract.
  - (b) Add server-side enum validation in the handler (tight schema enforcement) — would break D1.1's alias semantics, so not recommended.
  - (c) Add a docstring on `registerCommentTools` explaining the schema-vs-runtime layered model.
- **Owner:** D1.2 round 2 (option a — one-line description edit) OR future doc refinement. Same severity as D1.1's Attack 6.

#### Attack 4 — Backwards-compat path for stored rows with legacy target_type

- **Verdict: REFUTED for D1.2 scope (pre-existing condition).**
- **Evidence trace:**
  - `repo.go:3095-3097`: read path normalizes via `NormalizeCommentTargetType` then checks `IsValidCommentTargetType`. For a stored row with `target_type='branch'`: normalize lowercases (no alias hit; not in `validCommentTargetTypes`) → returns `"branch"` → `IsValidCommentTargetType("branch")` returns false → read fails with `decode comment target_type "branch": invalid target type`.
  - **Git blame**: this rejection behavior was introduced by Drop 1.75 commit `36ef724` ("collapse action_items.kind to closed 12-value enum") which tightened the domain enum from 7 values to 2. D1.2 only modifies the MCP **schema** declaration — it does NOT modify the domain validation, does NOT modify the read-path normalization, does NOT introduce new rejection cases.
  - **Pre-MVP rule**: per `feedback_no_migration_logic_pre_mvp.md`, dev wipes `~/.tillsyn/tillsyn.db` on schema-vocab changes. The Drop-1.75-era rejection of legacy rows is accepted by that rule.
- **No counterexample attributable to D1.2.** Any latent rejection of stored `branch`/`phase` rows is a Drop-1.75 inheritance, not a D1.2 regression.

#### Attack 5 — Test placement + wrong-layer regression guard

- **Verdict: CONFIRMED (test guards the wrong layer; pre-existing test-helper pattern unused).**
- **Concrete counterexample:**
  - **The added test `TestIsValidCommentTargetTypeLegacyTokensRejected` (`extended_tools_test.go:5953-5974`) exercises `domain.IsValidCommentTargetType` — a domain function** that has been rejecting legacy tokens since Drop 1.75 (`36ef724`). The test would have **passed identically before D1.2's enum-and-description change** because the domain function is independent of the MCP schema string.
  - **Repro**: revert D1.2's production change (restore `mcp.Enum("project", "branch", "phase", "actionItem", "subtask", "decision", "note")` and the old description string). Run `mage test-func ./internal/adapters/mcp_rpc TestIsValidCommentTargetTypeLegacyTokensRejected`. **The test will still pass.** It does not regression-guard D1.2's schema change at all.
  - **The right regression-guard layer**: the test should introspect the MCP schema via the existing `schemaPropertyEnumStrings` helper (`extended_tools_test.go:1172`) and assert the enum list matches `("project", "action_item", "actionItem")` and EXCLUDES the legacy tokens. This pattern is already used elsewhere — `extended_tools_test.go:2861` and `:2888` use `schemaPropertyEnumStrings` against `search_mode` and `sort` enums; `handler_test.go:1949` and `:3075` use it against `auth_request operation` enum.
- **Layer mismatch summary:**

| Layer | What D1.2 changed | What the regression test verifies | Match? |
| --- | --- | --- | --- |
| MCP schema enum string | Shrunk from 7 → 3 tokens | NOT verified | NO |
| MCP schema description string | Changed pipe-separated value list | NOT verified | NO |
| Domain `IsValidCommentTargetType` | Unchanged by D1.2 | Verified | Mismatched (already true since Drop 1.75) |

- **Recommendation:**
  - **D1.2 round 2** (preferred): replace `TestIsValidCommentTargetTypeLegacyTokensRejected` with `TestTillCommentSchemaTargetTypeEnumExcludesLegacy` that uses `schemaPropertyEnumStrings(t, schema, "target_type")` and asserts the slice equals `("project", "action_item", "actionItem")` exactly. Alternatively, keep both: the domain test guards the domain layer (cheap belt-and-suspenders), and add the schema-introspection test as the *primary* regression guard for D1.2's actual change. The builder's note (line 5955-5958) about "lives in mcp_rpc because sibling D1.1 dirty" is consistent with either option.
  - **Test placement (separate NIT)**: the domain assertion test in `mcp_rpc/extended_tools_test.go` reaches across package boundaries to test domain semantics. Per `feedback_md_update_qa.md` and Go conventions, package-internal tests live in the same package as the code under test. PLAN.md explicitly allowed this fallback (D1.1 dirty), so not a scope violation — but worth surfacing to dev for D2/D3 sibling-coordination doctrine: when a builder needs to land a regression test for changes that span domain+adapter layers and the domain test file is sibling-dirty, the **alternative** is to land the schema-introspection test in the adapter package (which is the layer that changed) and let D1.1's package-resident test guard the domain layer.

#### Attack 6 — Description-string contract format

- **Verdict: REFUTED.**
- **Evidence trace:**
  - `git grep` of `mcp.Description("[a-z_]*|` in `extended_tools.go` returns 18 hits using pipe-separated value lists. The convention is established across the file (`scope`, `priority`, `search_mode`, `sort`, `mode`, `scope_type`, `role`, `target_type`, etc.).
  - D1.2's new description `"project|action_item|actionItem"` matches the established convention exactly.
  - The pipe character `|` is JSON-safe (no escape required); it's a freeform documentation convention, not a parsed contract. MCP clients render the description verbatim in tool-list UIs.
- **No counterexample.** Convention is consistent; format is safe.

#### Attack 7 — R5 scope completion (other surfaces)

- **Verdict: REFUTED (R5 fully closed by D1.1 + D1.2).**
- **Evidence trace:**
  - Full `CommentTargetType` symbol audit (`git grep 'CommentTargetType'`): 130+ hits, every production-code reference uses post-1.75 `CommentTargetTypeProject` or `CommentTargetTypeActionItem`. Legacy const names (`CommentTargetTypeBranch`/`Phase`/`Subtask`/`Decision`/`Note`) are absent from the codebase since Drop 1.75 commit `36ef724`.
  - `internal/adapters/mcp_common/types.go:60-69` `commentTargetTypeFromScope`: switches only on `ScopeTypeProject` / `ScopeTypeActionItem`. Other scope types (`ScopeTypeBranch`, `ScopeTypePhase`, `ScopeTypeSubtask`) are NOT mapped (default → `("", false)`). Aligned with post-1.75 vocab.
  - `internal/adapters/mcp_common/app_service_adapter_auth_context.go:170-176` `populateCommentAuthContext`: switches on `target_type` value, branching `project` → project scope, default → action item scope. Both branches valid.
  - `internal/tui/thread_mode.go:60, :137, :275, :357, :384, :467, :604, :611-613` and `internal/tui/model.go:14417-14430`: all use post-1.75 `CommentTargetTypeProject` / `CommentTargetTypeActionItem` only. TUI explicitly collapses every kind to `CommentTargetTypeActionItem` (model.go:14418-14430).
  - `internal/app/snapshot.go:1485`: snapshot import normalizes target type — round-trip clean.
  - `internal/app/embedding_runtime.go`, `internal/app/inbox_attention.go`, `internal/app/service.go`, `internal/app/search_embeddings.go`: all post-1.75 vocab.
- **No other surface still uses the pre-1.75 vocab.** R5 is jointly closed by D1.1 (domain alias for `actionItem`) and D1.2 (MCP schema enum shrink). No GraphQL/REST surface exists; the MCP layer is the sole transport. No drift.
- **One related observation (NOT in R5 scope):** `internal/adapters/mcp_common/types.go:13-34` retains `ScopeTypeBranch`, `ScopeTypePhase`, `ScopeTypeSubtask` constants for **auth/lease scope_type** vocabulary (a different vocabulary from comment target_type). This is intentional pre-Drop-2 retention per `feedback_auth_path_branch_quirk.md`. Not D1.2's scope.

#### Attack 8 — Description NUL/escape characters

- **Verdict: REFUTED.**
- **Evidence:** the description literal `"project|action_item|actionItem"` is pure ASCII (no NUL, no control chars, no UTF-8 special chars). All three values use `[a-z_]` characters only. The `|` separator is JSON-safe and HTML-safe. No escape concerns.

### Counterexamples Summary

- **CONFIRMED (2):**
  - **Attack 2** — `TestHandlerExpandedCommentToolsForwardHierarchyTargetTypes` (`extended_tools_test.go:3514`) uses legacy `branch`/`phase` target_types and references a "hierarchy" concept that doesn't exist post-Drop-1.75. Function name + test cases inconsistent with the new schema enum. LOW severity; test still passes via stub bypass.
  - **Attack 5** — `TestIsValidCommentTargetTypeLegacyTokensRejected` exercises a domain function that has been rejecting legacy tokens since Drop 1.75. It does NOT actually guard D1.2's schema enum change. The right test would use `schemaPropertyEnumStrings` to introspect the MCP schema directly (helper exists at `extended_tools_test.go:1172`, used elsewhere at lines 2861, 2888, and in handler_test.go).
- **POSSIBLE (1):**
  - **Attack 3** — schema-vs-runtime asymmetry: runtime accepts case-insensitive variants (via D1.1's alias) but schema enum lists only exact-case literals. Schema is a subset of runtime acceptance — a different shape than R5's original drift. NIT; consider description-string annotation.
- **REFUTED (5):**
  - **Attack 1** — no runtime breakage; `mcp.Enum` is schema-metadata-only and the framework doesn't server-side-validate.
  - **Attack 4** — read-path rejection of legacy stored rows is a Drop-1.75 inheritance, not a D1.2 regression.
  - **Attack 6** — pipe-separated description convention is established and consistent.
  - **Attack 7** — R5 closed jointly by D1.1 + D1.2; no other surface retains pre-1.75 vocab.
  - **Attack 8** — description string is ASCII-clean.

### Severity + Mitigation Summary

| Attack | Verdict | Severity | Mitigation | Owner |
|--------|---------|----------|------------|-------|
| 1 caller breakage | REFUTED | n/a | — | — |
| 2 test-fixture drift (`branch`/`phase` in `TestHandlerExpandedCommentToolsForwardHierarchyTargetTypes`) | CONFIRMED | low | Rename/rewrite test to use post-1.75 values, OR add comment explaining stub-bypass intent | D1.2 round 2 OR Drop 4b refinement |
| 3 schema-vs-runtime case asymmetry | POSSIBLE | low | Annotate description: `"project|action_item|actionItem (case-insensitive)"` | D1.2 round 2 OR future refinement |
| 4 legacy stored rows | REFUTED | n/a | — (Drop-1.75 inheritance) | — |
| 5 wrong-layer regression test | CONFIRMED | medium | Replace with `schemaPropertyEnumStrings`-based test against `till.comment` schema; OR keep both (domain + schema-introspection) for belt-and-suspenders | D1.2 round 2 |
| 6 description format | REFUTED | n/a | — | — |
| 7 R5 scope completion | REFUTED | n/a | — | — |
| 8 description NUL/escape | REFUTED | n/a | — | — |

### Recommendation

**PASS-WITH-FINDINGS on the D1.2 build claim.** The production diff (schema enum + description string) is correct and matches PLAN.md acceptance bullets verbatim. The two CONFIRMED findings are:

1. **Attack 5 (medium severity)** — the added regression test does not guard the schema change. D1.2 round 2 should replace it with a schema-introspection test using the existing `schemaPropertyEnumStrings` helper. The dev should decide whether to keep the domain-layer test as belt-and-suspenders or replace entirely.
2. **Attack 2 (low severity)** — the pre-existing `TestHandlerExpandedCommentToolsForwardHierarchyTargetTypes` test uses `branch`/`phase` and references a post-1.75-eliminated "hierarchy" concept. Within D1.2's `paths` envelope. Either rewrite to use new vocab or annotate as backward-compat behavior test.

**Attack 3 (low POSSIBLE)** is dev-disposition — same family as D1.1's Attack 6 (silent transformation vs explicit signaling per `feedback_parity_clarity_no_silent_failures.md`). Consider whether to telegraph case-insensitive runtime in the schema description, or defer to a broader API-doc refinement.

Counterexamples count: **2 CONFIRMED**, **1 POSSIBLE**, **5 REFUTED**.

### Hylla Feedback

Per `feedback_hylla_disabled_for_now.md` (2026-05-18 directive): Hylla MCP is OFF; no Hylla queries attempted in this review. Fallback evidence sources: `Read`, `git grep` via Bash, `git diff`, `git log`, `git show`. No section required per the 2026-05-18 rule.

## Droplet 1.3 — Round 1

**Reviewer:** go-qa-falsification-agent
**Builder:** go-builder-agent (D1.3 Round 1)
**Files under review:**
- `internal/app/dispatcher/subscriber_test.go` (modified — removed 3 symbols + 1 import)
- `internal/app/dispatcher/dispatcher_e2e_test.go` (NEW — `TestMain` + goleak + 2 moved e2e tests)

### Attack 1 — goleak false-positive identifiability

**POSSIBLE / by-design limitation (not a builder defect)**

`goleak.VerifyTestMain(m)` body (verified via `~/go/pkg/mod/go.uber.org/goleak@v1.3.0/testmain.go`):

```go
func VerifyTestMain(m TestingM, options ...Option) {
    exitCode := m.Run()
    // ...
    if exitCode == 0 {
        if err := Find(opts); err != nil {
            fmt.Fprintf(_osStderr, "goleak: Errors on successful test run: %v\n", err)
            exitCode = 1
        }
    }
}
```

`Find()` runs ONCE after `m.Run()` returns — at which point all 389 tests have completed and only residual goroutines remain. The error message contains goroutine STACK TRACES (which include the function that spawned them, e.g. `dispatcher.(*Dispatcher).Start.func1`) but does NOT name which test spawned the goroutine. This is a documented goleak limitation, NOT a builder defect.

Mitigation path if a future leak surfaces: dev would switch to per-test `defer goleak.VerifyNone(t)` to bisect which test leaks. The current `TestMain` approach is the canonical goleak idiom and the right starting point — switching to per-test is a future refinement, not a precondition for D1.3 PASS.

**Verdict: POSSIBLE (limitation acknowledged in worklog comment, no action required).**

### Attack 2 — `TestMain` ⊗ `t.Parallel()` interaction

**REFUTED**

`TestAutoDispatchE2EGateFailViaNewDispatcher` (line 154) DOES call `t.Parallel()`. `TestAutoDispatch_NewDispatcherGateWiring` (line 69) does NOT (correctly — the file comment line 67-68 explains: `withFakeCommandRunner` swaps the package-level `defaultCommandRunner` var; parallel execution would race that swap).

The `t.Parallel()` ⊗ `TestMain` interaction is benign here:
- `m.Run()` waits for all tests (sequential + parallel) to complete before returning.
- `goleak.Find()` runs AFTER `m.Run()` returns — i.e., after every goroutine spawned by any test should have been cleaned up.
- Whether a test ran in parallel or sequential is irrelevant: the leak check observes only the FINAL residual set.

The attack's concern (which test leaked is ambiguous when only the LAST finishing test is captured) is the same as Attack 1's limitation — it does NOT change with parallel vs sequential. REFUTED as a builder defect.

**Verdict: REFUTED.**

### Attack 3 — Hidden state leak from the move

**REFUTED**

Read `subscriber_test.go` head + `dispatcher_e2e_test.go` in full. No package-level state was moved or duplicated:
- `stubE2ETemplateResolver` is declared ONCE, in `dispatcher_e2e_test.go:39-41`. `rg -n "stubE2ETemplateResolver|TestAutoDispatchE2E|templates\." internal/app/dispatcher/subscriber_test.go` returned 0 matches.
- No `init()` functions exist in either file.
- No package-level `var` declarations were moved — only the type + its method + two test functions.

`TestMain` is a special test-binary entrypoint, NOT package-level state in the redeclaration sense. There is exactly ONE `TestMain` in `internal/app/dispatcher/` (verified via `rg -ln "func TestMain" internal/app/dispatcher/`). No redeclaration risk.

**Verdict: REFUTED.**

### Attack 4 — R6.1 rename completeness

**NIT (low severity, intra-drop documentation only)**

`rg -n "TestAutoDispatchE2EGatePassViaNewDispatcher"` repo-wide returns 12 matches across 6 files. ALL 12 matches are in **documentation / workflow MDs**, NOT in code:

- `workflow/drop_4b/D5_BUILDER_WORKLOG.md` — prior drop's worklog (historical; correct).
- `workflow/drop_4b_test_cleanup/BUILDER_WORKLOG.md` — this drop's worklog (correct — documents the rename).
- `workflow/drop_4b_test_cleanup/PLAN.md` — this drop's plan (correct — describes R6.1).
- `workflow/drop_4b_test_cleanup/PLAN_QA_FALSIFICATION.md` + `PLAN_QA_PROOF.md` + `REVISION_BRIEF.md` — planning artifacts (correct — references the old name).

Zero code references, zero CI workflow references, zero non-historical references. The old name appears only where the rename event is being documented. This is correct — workflow MDs are an audit trail and should retain the historical name.

`rg -n "TestAutoDispatch_NewDispatcherGateWiring"` repo-wide returns 13 matches across 7 files, with the canonical code declaration at `dispatcher_e2e_test.go:69` (function definition) and `:50` (doc comment). All matches are accounted for.

**Verdict: REFUTED for code; NIT for workflow MDs (intentional historical references — leave as-is).**

### Attack 5 — R7.4 file split completeness

**REFUTED**

`stubE2ETemplateResolver` is a struct with exactly ONE method `GetProjectTemplate`. Per `dispatcher_e2e_test.go:39-48`:

```go
type stubE2ETemplateResolver struct {
    tpl templates.Template
}

func (s *stubE2ETemplateResolver) GetProjectTemplate(_ context.Context, _ string) (templates.Template, error) {
    return s.tpl, nil
}
```

Type + method moved together to the new file. Original location (subscriber_test.go) returns 0 hits on `rg -n "stubE2ETemplateResolver" internal/app/dispatcher/subscriber_test.go`. No method stranded.

**Verdict: REFUTED.**

### Attack 6 — Import drift

**REFUTED**

`subscriber_test.go` post-D1.3 imports (read lines 1-12):

```go
import (
    "context"
    "errors"
    "sync/atomic"
    "testing"
    "time"

    "github.com/evanmschultz/tillsyn/internal/app"
    "github.com/evanmschultz/tillsyn/internal/domain"
)
```

`templates` is removed. `rg -n "templates\." internal/app/dispatcher/subscriber_test.go` returns 0 matches — confirmed no residual reference. `errors` is kept correctly (used by other tests in the file per worklog note + `mage test-pkg` GREEN at 389/389 confirms the package compiles).

**Verdict: REFUTED.**

### Attack 7 — `TestMain` os.Exit invariant

**REFUTED**

`dispatcher_e2e_test.go:31-33`:

```go
func TestMain(m *testing.M) {
    goleak.VerifyTestMain(m)
}
```

No `os.Exit` wrapping. Body is exactly one statement: the `VerifyTestMain` call. Per goleak source, `VerifyTestMain` handles its own `os.Exit` via a deferred `cleanup(exitCode)` (default = `_osExit = os.Exit`). The builder correctly avoided the `os.Exit(goleak.VerifyTestMain(m))` antipattern (which would fail to compile since `VerifyTestMain` returns `void`, not `int`).

**Verdict: REFUTED. Builder caught the gotcha correctly.**

### Attack 8 — Goroutine source coverage (Start without Cleanup-Stop)

**REFUTED**

`rg -n "d\.Start\b" internal/app/dispatcher/*_test.go` returns 18 matches; the dispatcher-test Start call sites are at:
- `dispatcher_e2e_test.go:107` — paired with `t.Cleanup` Stop at line 110-114. ✓
- `dispatcher_test.go:370` — verified manually (not under D1.3 scope but in-package).
- `subscriber_test.go:147, 191, 228, 268, 277, 300, 349, 383, 403, 431` — sampled lines 147 (Cleanup at 150-154) ✓ , 191 (Cleanup at 194-198) ✓ , 228 (Cleanup absent — `TestDispatcherStopCancelsAllSubscribers` calls Stop directly at line 238 as the test assertion) ✓ , 268 (Cleanup at 271-275) ✓.

Note: `TestDispatcherStopCancelsAllSubscribers` (subscriber_test.go:217) does not have `t.Cleanup` Stop — because the test's own body calls Stop as the assertion. That's correct.

`mage test-pkg ./internal/app/dispatcher` re-run produced 389/389 GREEN with no leak report on stderr — empirical confirmation that every Start has either a matching Stop or that the test exits cleanly. The matching is not just visual; goleak ran and approved.

**Verdict: REFUTED.**

### Attack 9 — Test count integrity

**REFUTED**

Pre-D1.3 subscriber_test.go (`git show HEAD:internal/app/dispatcher/subscriber_test.go` → `rg -c "^func Test"`) = **11 test functions**.

Post-D1.3:
- `subscriber_test.go` = 9 test functions (`rg -c "^func Test"`).
- `dispatcher_e2e_test.go` = 3 matches on `^func Test` but ONE is `TestMain` (not a test, it's the binary entrypoint).
- Net test functions = 9 + 2 = **11**.

Δ = 0. Rename ≠ new test. `TestMain` doesn't count as a runnable test in `mage test-pkg`'s 389-tests reporting. The 389 total is dispatcher-package-wide and includes the count after the move; the pre-move count would have been the same since rename and move preserve test identity.

**Verdict: REFUTED.**

### Attack 10 — R6.2 scope-creep guard "didn't fire" claim

**REFUTED**

Re-ran `mage test-pkg ./internal/app/dispatcher` independently in this review:

```
[PKG PASS] github.com/evanmschultz/tillsyn/internal/app/dispatcher (0.01s)
Test summary
  tests: 389
  passed: 389
  failed: 0
  skipped: 0
[SUCCESS] All tests passed
  389 tests passed across 1 package.
```

Exit code 0, no `goleak: Errors on successful test run:` on stderr — empirically confirms zero leaked goroutines from any of the 389 tests with `TestMain` active. Builder claim verified independently.

**Verdict: REFUTED.**

### Summary Table

| # | Attack                                           | Verdict   | Severity | Concrete repro                              |
| - | ------------------------------------------------ | --------- | -------- | ------------------------------------------- |
| 1 | goleak failure-output identifies leak source     | POSSIBLE  | low      | by-design (`goleak.Find` runs after m.Run)  |
| 2 | `TestMain` ⊗ `t.Parallel()` interaction          | REFUTED   | n/a      | —                                           |
| 3 | Hidden state leak from move                      | REFUTED   | n/a      | —                                           |
| 4 | R6.1 rename completeness                         | NIT       | low      | doc-only refs in workflow MDs (intentional) |
| 5 | R7.4 file split completeness                     | REFUTED   | n/a      | —                                           |
| 6 | Import drift                                     | REFUTED   | n/a      | —                                           |
| 7 | `TestMain` os.Exit invariant                     | REFUTED   | n/a      | —                                           |
| 8 | Goroutine source coverage (Start/Stop pairs)     | REFUTED   | n/a      | —                                           |
| 9 | Test count integrity (rename ≠ new test)         | REFUTED   | n/a      | 11 pre = 9+2 post                           |
| 10 | R6.2 scope-creep "didn't fire" claim            | REFUTED   | n/a      | mage test-pkg re-run GREEN 389/389          |

Counterexamples count: **0 CONFIRMED**, **1 POSSIBLE**, **1 NIT**, **8 REFUTED**.

### Recommendation

**PASS on the D1.3 build claim.** No CONFIRMED counterexamples. The two non-PASS verdicts are:

1. **Attack 1 (POSSIBLE)** — goleak's `TestMain` mode cannot identify which specific test leaked a goroutine when one surfaces in the future. This is a documented goleak limitation, not a builder defect, and the canonical usage pattern matches. If future leaks surface and bisection is needed, the dev can downgrade to per-test `goleak.VerifyNone(t)`. No action required for D1.3 PASS.
2. **Attack 4 (NIT)** — the old test name `TestAutoDispatchE2EGatePassViaNewDispatcher` still appears in 6 workflow MD files. ALL references are intentional (worklog audit trail + planning artifacts describing the rename event). Leave as-is; these are historical references in audit-trail documents.

D1.3 ships:
- `TestMain` + `goleak.VerifyTestMain(m)` correctly wired (R6.2).
- E2E tests split into `dispatcher_e2e_test.go` (R7.4 — type + 2 methods + comments + new file header).
- `TestAutoDispatchE2EGateFailViaNewDispatcher` retained; rename of pass-test → `TestAutoDispatch_NewDispatcherGateWiring` correctly applied (R6.1).
- `templates` import correctly removed from `subscriber_test.go`.
- All 389 dispatcher tests GREEN with goleak active.

### Hylla Feedback

Per `feedback_hylla_disabled_for_now.md` (2026-05-18 directive): Hylla MCP is OFF; no Hylla queries attempted. Fallback evidence sources: `Read`, `Bash` (`rg`, `mage test-pkg`, `git show`), `git status`, `git diff`. No `## Hylla Feedback` section content required per the 2026-05-18 rule.

## Droplet 1.2 — Round 2

- **Reviewer:** go-qa-falsification-agent
- **Reviewed at:** 2026-05-18
- **Files reviewed:**
  - `internal/adapters/mcp_rpc/extended_tools_test.go` (round-2 edits: new `TestHandlerCommentToolTargetTypeEnumSchemaGuard` at L5964 + renamed `TestHandlerCommentToolForwardsArbitraryTargetTypesToService` at L3520)
  - `internal/adapters/mcp_rpc/extended_tools.go` (production state at HEAD)
- **Build-tool targets:**
  - `git diff HEAD -- internal/adapters/mcp_rpc/extended_tools.go` (confirmed identical to round-1 enum + description change, 1 line)
  - `mage test-pkg ./internal/adapters/mcp_rpc` (232 tests pass)
  - `mage test-func ./internal/adapters/mcp_rpc TestHandlerCommentToolTargetTypeEnumSchemaGuard` (1/1 pass)
- **Verdict:** **PASS-WITH-NITS**. The two round-2 absorptions are well-executed. Schema-introspection test correctly catches enum reverts; renamed test preserves transport-permissiveness value and documents intent. Three non-blocking NITs raised below; no CONFIRMED counterexamples.

### Round-2 Attacks

**Attack 1 — Schema-guard test brittleness on future legitimate enum extension.** **REFUTED-AS-DESIRABLE.**

The test's exact-cardinality assertion (`len(gotEnum) != len(wantEnum)` at L5995) plus the explicit stale-token exclusion list at L6005 (`branch, phase, subtask, decision, note`) is **intentionally tight**. If a future drop legitimately adds a new accepted target type (e.g., post-team-feature adds `team` or post-cascade-vocab adds `droplet`), this test will fail with a clear message identifying the new token. That forces explicit human acknowledgment of the schema-contract change rather than silent expansion. This matches the established `regression guard` doc-comment pattern at L5958-5963 — the test exists precisely to make schema drift loud. Not brittle; load-bearing.

**Attack 2 — Schema-guard ordering sensitivity (`reflect.DeepEqual` vs `cmp.Diff`).** **REFUTED.**

The test uses `slices.Contains(gotEnum, want)` (extended_tools_test.go:5999) inside a `for want := range wantEnum` loop, NOT `reflect.DeepEqual` or positional indexing. Set-equality semantics. Even if `mcp.Enum(...)` were to reorder values during JSON marshaling (verified via `go doc github.com/mark3labs/mcp-go/mcp.Enum` — docstring makes no ordering guarantee), the test would still pass. The cardinality check at L5995 (`len(gotEnum) != len(wantEnum)`) plus the per-element `slices.Contains` walk is correctly set-tolerant. No false negatives possible from ordering.

**Attack 3 — `schemaPropertyEnumStrings` helper correctness.** **REFUTED.**

Helper definition at extended_tools_test.go:1171-1192. It:
1. Asserts `schema["properties"]` is a map (fatal on missing — L1175-1177)
2. Asserts `propRaw[property]` is a map (fatal on missing — L1178-1181)
3. Type-asserts `enumRaw, _ := propRaw["enum"].([]any)` (graceful — empty slice if missing or wrong type)
4. Returns `[]string` filtered to string-typed items only

**Failure mode check:** If `enum` is missing from the schema, `enumRaw` is `nil` and the helper returns `[]string{}` (length 0). The test would then fail at L5995 (`len(gotEnum) != len(wantEnum)`) with a clear `[]string{}, want exactly [project action_item actionItem]` message. The helper does NOT silently pass-through vacuously — the cardinality guard at L5995 catches the missing-enum case. Sibling usages at L2861, L2888, L1949, L3075 confirm the helper is well-exercised. Verified by red-green: builder's worklog L110 documents reverting the enum and seeing the test fail at L5996. Helper works.

**Attack 4 — Description-string assertion completeness (drift in other directions).** **NIT.**

L6013 asserts `strings.Contains(targetTypeDesc, "project|action_item|actionItem")` (substring match) and L6017 asserts each stale token is absent. This catches:
- Substring removal (description shortened to drop the vocab)
- Stale-token reintroduction
- Wrong vocab list

It does NOT catch:
- Extra trailing text appended to the description (acceptable — descriptions can grow non-destructively)
- Whitespace drift inside the substring (since the literal is exact, any internal whitespace change WOULD be caught — Contains is byte-exact)
- Case drift (`Project|Action_item|...` would fail `Contains`)

**NIT-1:** The substring approach is fine for the schema-vocab contract this test guards. A purist might argue for `strings.HasPrefix(targetTypeDesc, "project|action_item|actionItem")` to also pin position, but description fields typically only contain the vocab list at L2243 (no trailing prose), so prefix-vs-contains is a 0-value distinction. Accept as-is.

**Attack 5 — Renamed test (`TestHandlerCommentToolForwardsArbitraryTargetTypesToService`) actual value.** **REFUTED.**

The renamed test (L3520-3574) sends `target_type=branch` (L3538) and `target_type=phase` (L3558) and asserts:
1. Handler does NOT return `isError` (L3545, L3562) — proves no rejection at transport layer
2. Service stub captures the literal token (`got != "branch"` at L3548; `got != "phase"` at L3566)

**What it catches that the schema-guard test does NOT:** if a future change adds MCP-layer validation (e.g., a switch statement filtering by `IsValidCommentTargetType` before forwarding to the service), this test would BREAK because the unknown tokens would be rejected at the handler. The schema-guard test only inspects the declared JSON-Schema enum — it doesn't exercise the runtime handler path. These tests guard different surfaces:

- `TestHandlerCommentToolTargetTypeEnumSchemaGuard` → declarative schema contract (what `tools/list` exposes)
- `TestHandlerCommentToolForwardsArbitraryTargetTypesToService` → runtime transport semantics (what the handler actually does with non-enum values)

The renamed test's value is **non-zero and distinct**. The doc-comment at L3514-3519 explicitly captures this distinction. The Option C rename was the correct choice over Option A (replace tokens) which would have collapsed the test into a schema-compliance check redundant with the new schema-guard test. Worklog L114-115 documents this reasoning.

**Attack 6 — Renamed test name length (56 chars).** **NIT.**

Exact length: `TestHandlerCommentToolForwardsArbitraryTargetTypesToService` = 60 chars. Comparable existing tests in the same file:
- `TestHandlerExpandedToolBuildsActorTupleFromAuthenticatedSession` (62 chars)
- `TestHandlerExpandedToolRejectsMissingSessionAndGuardedUserTuples` (63 chars)
- `TestHandlerExpandedEmbeddingsToolsExposeMixedSubjectMetadata` (59 chars)
- `TestHandlerActionItemCreateAcceptsStateOrColumnIDExclusively` (60 chars)

The renamed test fits the established convention. Not a novella — the package has at least 4 sibling tests in the 59-63 char range. Accept as-is.

**Attack 7 — Production file integrity (`extended_tools.go` unchanged from round 1).** **REFUTED.**

`git diff HEAD -- internal/adapters/mcp_rpc/extended_tools.go` shows exactly:
```
@@ -2240,7 +2240,7 @@
-    mcp.WithString("target_type", mcp.Description("project|branch|phase|actionItem|subtask|decision|note"), mcp.Enum("project", "branch", "phase", "actionItem", "subtask", "decision", "note")),
+    mcp.WithString("target_type", mcp.Description("project|action_item|actionItem"), mcp.Enum("project", "action_item", "actionItem")),
```
+1 -1, exactly the round-1 enum + description change. Builder's claim that round 2 touched only `extended_tools_test.go` (worklog L116) is confirmed.

**Attack 8 — Test count integrity (232 pre/post round 2).** **REFUTED.**

`mage test-pkg ./internal/adapters/mcp_rpc` returns `tests: 232, passed: 232, failed: 0`. Net delta:
- Round 1 added `TestIsValidCommentTargetTypeLegacyTokensRejected` (+1)
- Round 2 removed it (-1) and added `TestHandlerCommentToolTargetTypeEnumSchemaGuard` (+1)
- Rename `TestHandlerExpandedCommentToolsForwardHierarchyTargetTypes` → `TestHandlerCommentToolForwardsArbitraryTargetTypesToService` (+0)
- Net: pre-round-1 (231) + round-1 (232) + round-2 (-1 + 1 = 0) = 232

Confirmed via `rg -c '^func Test' extended_tools_test.go` = 54 test functions in this file (matches earlier round-1 inventory + round-2 swap-in-place).

**Attack 9 — Schema-guard test isolation under `t.Parallel()`.** **REFUTED.**

The schema-guard test at L5964 calls `t.Parallel()` at L5965. Verified `NewHandler` (handler.go:512-524) constructs a fresh `mcpserver` + fresh `StreamableHTTPServer` per call — no shared global registry. Each test gets its own `httptest.NewServer(handler)` (L5977) bound to a unique port. The `slices.Contains` check at L5999 operates on the locally-scoped `gotEnum` slice. No cross-test mutation surface. `mage test-pkg` passed all 232 tests under default parallelism (Go testing's `-parallel` defaults to GOMAXPROCS), confirming runtime isolation holds.

**Attack 10 — Renamed test's doc-comment clarity.** **NIT.**

Doc comment at L3514-3519 (6 lines):
```
// TestHandlerCommentToolForwardsArbitraryTargetTypesToService verifies that the MCP
// handler layer does NOT validate target_type values; any string passes through
// to the service. Domain validation is downstream. This test intentionally sends
// pre-Drop-1.75 legacy tokens ("branch", "phase") to confirm they transit the
// transport layer untouched — the schema enum excludes them from the JSON-Schema
// contract but the handler itself does not reject unknown values.
```

This is clear and complete:
- Names what's being tested (handler-layer non-validation)
- States the architectural invariant (domain validation is downstream)
- Justifies the legacy-token choice (intentional, demonstrates transport permissiveness)
- Distinguishes from the schema-enum contract (handler vs schema split)

**NIT-2 (minor):** The phrase "transit the transport layer untouched" could read as "transport doesn't mutate the values," when the test actually proves "transport doesn't reject" plus "transport forwards verbatim to service." A pedant might split those, but in context the meaning is unambiguous. Accept as-is.

**NIT-3 (cross-test linkage):** Neither the renamed test's doc-comment NOR the new schema-guard test's doc-comment cross-references the other. A reader landing on `TestHandlerCommentToolForwardsArbitraryTargetTypesToService` would benefit from `// See also TestHandlerCommentToolTargetTypeEnumSchemaGuard for the declarative-schema-contract side of this split.` (and vice versa). Not blocking; dev-disposition.

### Round-2 Verdict Summary

| Attack | Verdict | Notes |
|--------|---------|-------|
| 1. Schema-guard brittleness on legit expansion | REFUTED-AS-DESIRABLE | Tight cardinality is load-bearing |
| 2. Schema-guard ordering sensitivity | REFUTED | `slices.Contains` is set-semantics |
| 3. `schemaPropertyEnumStrings` vacuous-pass | REFUTED | Cardinality guard catches missing enum |
| 4. Description-string drift coverage | NIT-1 | Contains-only is fine; prefix would be 0-value |
| 5. Renamed test obsolescence | REFUTED | Guards distinct runtime surface vs schema-guard |
| 6. Test name length | NIT (cosmetic) | Fits package convention (4 siblings 59-63 chars) |
| 7. Production file integrity | REFUTED | Diff matches round-1 exactly (+1/-1) |
| 8. Test count integrity | REFUTED | 232 confirmed; net delta computes |
| 9. Schema-guard parallel-test isolation | REFUTED | Fresh handler per test; no shared state |
| 10. Renamed-test doc clarity | NIT-2/NIT-3 | Minor clarity polish + cross-ref opportunity |

**Verdict: PASS-WITH-NITS.** Zero CONFIRMED counterexamples. Three minor NITs (NIT-1 description-contain-vs-prefix, NIT-2 transport-untouched phrasing, NIT-3 cross-reference between split tests) are dev-disposition cosmetic polish, not gate-blockers.

The round-2 absorptions cleanly resolve Attack 5 (wrong-layer test → genuine schema-introspection guard with red-green verification) and Attack 2 (stale-vocab test preserved as transport-permissiveness regression test under Option C). The schema-guard test design is correct: cardinality + set-equality + stale-token exclusion + description substring assertion together pin the schema contract without false-positive ordering risk. The renamed test now occupies a defensibly distinct slot in the test surface (runtime handler vs declarative schema).

### Hylla Feedback

Per `feedback_hylla_disabled_for_now.md` (2026-05-18 directive): Hylla MCP is OFF; no Hylla queries attempted. Fallback evidence sources: `Read`, `Bash` (`rg`, `mage test-pkg`, `mage test-func`, `git diff`, `go doc`). No `## Hylla Feedback` section content required per the 2026-05-18 rule.
