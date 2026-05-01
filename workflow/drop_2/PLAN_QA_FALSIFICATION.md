# DROP_2 — Plan QA Falsification, Round 1

**Verdict:** fail
**Date:** 2026-05-01

The plan has multiple blocking counterexamples around the strict-canonical state-vocabulary rename (Unit B, droplets 2.7/2.8/2.9). The planner under-scoped the `Paths:` lists and missed at least four whole files plus several call sites that reference the renamed `domain.StateDone` / `domain.StateProgress` symbols and the legacy state literals (`"done"` / `"progress"` / `"completed"` / `"in-progress"` / `"doing"`). Because these symbols are renamed at the Go level (not extended), every dependent file MUST be edited in the same droplet/unit or the package and every transitive importer fail to compile. Without remediation, intermediate per-droplet commits between 2.7 and 2.9 will leave `mage ci` red across `internal/app`, `internal/config`, `internal/adapters/server/common`, `internal/adapters/server/mcpapi`, `internal/tui`. The Drop-2 hard rule "Each unit lands `mage ci` green before the next unit's first droplet starts" is not even reachable as currently scoped.

## 1. Finding — Unit B `Paths:` lists omit at least 4 production files that reference renamed symbols (BLOCKING)

- **Severity:** blocking
- **Attack vector:** 3 (strict-canonical leak), 13 (hidden state-vocabulary surface), and the same-package compile-fallout argument.
- **Counterexample evidence (file:line):**
  - `internal/app/service.go:556,623-694,1817,1965-1975,1948-1968,1877-1880` — `defaultStateTemplates` returns `{ID: "progress", "done"}`; `lifecycleStateForColumnID` switches on `"done"`/`"progress"` and returns `domain.StateDone`/`domain.StateProgress`; `normalizeStateID` legacy alias map; many direct `domain.StateDone`/`domain.StateProgress` comparisons in transition logic.
  - `internal/app/snapshot.go:419,1267` — `case domain.StateTodo, domain.StateProgress, domain.StateDone, domain.StateFailed, domain.StateArchived:` validation switch; `state = domain.StateTodo` fallback. Renamed symbols compile-fail here.
  - `internal/app/attention_capture.go:350,353,356,371` — multiple `domain.StateDone`/`domain.StateProgress` checks. Compile-fail.
  - `internal/adapters/server/common/capture.go:296-312` — `canonicalLifecycleState` is a SECOND coercion site (separate from the one named in 2.9 `app_service_adapter_mcp.go`). It has its own `case "progress", "in-progress", "doing":` / `case "done", "complete", "completed":` aliases. **Plan 2.9 does NOT name this file.**
  - `internal/tui/thread_mode.go:151` — uses `domain.StateDone`. Plan 2.8 paths list `model.go`, `options.go`, and `*_test.go` only. `thread_mode.go` (production, non-test) is missing.
  - `internal/domain/action_item.go:268,275,278,315` — same package as 2.7 but **not listed in 2.7 `Paths:`**. References `StateProgress`/`StateDone`/`StateFailed` directly. The file IS in `internal/domain` so `mage test-pkg ./internal/domain` catches it, but the planner did not declare the path.
  - `internal/adapters/storage/sqlite/repo.go:2819` — uses `domain.StateTodo` only (unchanged), so this one is fine, but the file is NOT in any droplet's `Paths:` and 2.4 (which DOES touch repo.go for the role column) precedes 2.7 by units. Tooling-wise this is OK; the more interesting break is whether QA gates catch the missed paths above.
  - `internal/config/config.go:218,550,1094` — `Search.States` defaults to `["todo", "progress", "done"]`; `isKnownLifecycleState` accepts the legacy values; if strict-canonical, `isKnownLifecycleState("complete")` returns FALSE. Config validation breaks.
  - `internal/config/config_test.go:326,814` — config tests using legacy values.
  - `internal/adapters/server/common/app_service_adapter_lifecycle_test.go:180,716,721,912` — `MoveActionItemStateRequest{State: "done"}` test inputs. Under strict-canonical these inputs are now REJECTED, so these tests must be rewritten. **Same package as 2.9 (`internal/adapters/server/common`) but plan 2.9 only lists `app_service_adapter_mcp.go` and `cmd/till/main.go`.**
  - `internal/adapters/server/mcpapi/extended_tools_test.go:1114,2587,2600` — MCP tests with `"state": "done"`. Plan 2.5 (MCP role) touches this file but plan 2.9 (state-vocabulary sweep) does not list it.
  - `internal/adapters/server/common/capture_test.go:268-269` — `canonicalLifecycleState("doing") != domain.StateProgress` — strict-canonical change requires this test rewrite.
- **Why this is blocking:** Plan 2.7 renames `StateDone`→`StateComplete` and `StateProgress`→`StateInProgress` as Go symbols. Every file that imports `internal/domain` and references those names will fail `go build`. The plan's per-droplet acceptance for 2.7 is `mage test-pkg ./internal/domain` only, which compiles ONLY the domain package — the adjacent packages do not get compiled. Per-droplet QA passes; the moment 2.7 is committed, `mage ci` (the unit-boundary gate after 2.9) lights up red and stays red until every missed file is fixed. If a builder is honest about per-droplet scope and only edits the files listed in `Paths:`, the entire repo is broken between commit-2.7 and commit-2.9, and 2.8/2.9 acceptance criteria reference `git grep` sweeps that are also incomplete.
- **Suggested mitigation:** Either (a) rewrite Unit B as one large droplet that touches every file with a `domain.StateDone`/`StateProgress` reference + every legacy state literal in one commit, OR (b) add explicit `Paths:` entries to 2.7/2.8/2.9 covering: `internal/domain/action_item.go`, `internal/app/service.go`, `internal/app/service_test.go`, `internal/app/snapshot.go`, `internal/app/snapshot_test.go`, `internal/app/attention_capture.go`, `internal/app/attention_capture_test.go`, `internal/adapters/server/common/capture.go`, `internal/adapters/server/common/capture_test.go`, `internal/adapters/server/common/app_service_adapter_lifecycle_test.go`, `internal/adapters/server/mcpapi/extended_tools_test.go`, `internal/tui/thread_mode.go`, `internal/config/config.go`, `internal/config/config_test.go`. Add explicit per-droplet acceptance to require `mage build ./...` (full-tree compile) green even if `mage test-pkg` still scopes the test runner. Best path is (a) — bundle all symbol-rename and string-literal flips into one atomic commit, since the strict-canonical cutover cannot be partial without breaking the whole tree.

## 2. Finding — Plan 2.7 missing path inside its own package (NIT)

- **Severity:** nit (compile gate catches it; planner under-declared scope)
- **Attack vector:** 1 (missing `blocked_by` / scope drift)
- **Counterexample evidence:** Plan 2.7 `Paths:` lists `internal/domain/workitem.go` + `internal/domain/workitem_test.go`. `internal/domain/action_item.go:268,275,278,315` references the renamed `StateProgress`/`StateDone` directly; `internal/domain/domain_test.go:288-442` does too. Plan 2.7 mentions "and any other `internal/domain/*_test.go` files referencing the old constants/field name" — that catches `domain_test.go` but **not** `action_item.go`. The builder will discover this mid-flight.
- **Why nit not blocking:** Same-package, so `mage test-pkg ./internal/domain` will fail until the builder edits `action_item.go`. The compile gate forces the right behavior. But the planner's `Paths:` declaration is wrong, which weakens the file-locking blocker semantics.
- **Suggested mitigation:** Add `internal/domain/action_item.go` to 2.7 `Paths:`.

## 3. Finding — Plan 2.8 missing `internal/tui/thread_mode.go` (BLOCKING for 2.8 acceptance criteria, but compile gate forces fix)

- **Severity:** blocking
- **Attack vector:** 1 + 13
- **Counterexample evidence:** `internal/tui/thread_mode.go:151` uses `domain.StateDone`. Plan 2.8 `Paths:` lists only `model.go`, `options.go`, and `*_test.go`. After 2.7 renames the symbol, `thread_mode.go` does not compile. Plan 2.8 acceptance also greps `internal/tui/` for legacy literals — but the production file is not declared scope.
- **Suggested mitigation:** Add `internal/tui/thread_mode.go` to 2.8 `Paths:`.

## 4. Finding — Plan 2.9 missing `internal/adapters/server/common/capture.go` and `capture_test.go` (BLOCKING)

- **Severity:** blocking
- **Attack vector:** 3 (strict-canonical leak — second coercion site)
- **Counterexample evidence:** `internal/adapters/server/common/capture.go:297-312` defines `canonicalLifecycleState(state domain.LifecycleState) domain.LifecycleState` — an INDEPENDENT coercion site outside `app_service_adapter_mcp.go`. Plan 2.9 says "rewrite `actionItemLifecycleStateForColumnName` at `:849-864` … rewrite `normalizeStateLikeID` at `:866-901` to accept ONLY canonical inputs" but is silent on `canonicalLifecycleState` in `capture.go`. If 2.9 leaves `capture.go` accepting `"doing"`/`"completed"`/`"in-progress"` etc., the strict-canonical claim is false: there's still a path through which legacy values get coerced. The acceptance criterion `git grep -E '"in-progress"' -- '*.go'` returns empty would FAIL.
- **Suggested mitigation:** Add `internal/adapters/server/common/capture.go` and `internal/adapters/server/common/capture_test.go` to 2.9 `Paths:`. Update acceptance to explicitly require strict-canonical normalization in `canonicalLifecycleState`.

## 5. Finding — Plan 2.9 missing `internal/app/service.go` (BLOCKING)

- **Severity:** blocking
- **Attack vector:** 3 + 13
- **Counterexample evidence:** `internal/app/service.go` has THREE distinct state-vocabulary surfaces:
  1. `defaultStateTemplates` (`:1873-1881`) returns `[]StateTemplate{{ID: "todo"}, {ID: "progress"}, {ID: "done"}, {ID: "failed"}}` — this is the BOOT-SEED of state IDs for new boards. Pre-MVP dev fresh-DBs each unit, so this seed is the de-facto authoritative seed for state IDs after Unit B lands. Strict-canonical requires `"in_progress"`/`"complete"` here, not `"progress"`/`"done"`.
  2. `normalizeStateID` (`:1922-1955`) has a switch coercing `"done", "complete", "completed"` → `"done"` and `"in-progress", "progress", "doing"` → `"progress"`. Strict-canonical requires this to either reject legacy or coerce TO canonical.
  3. `lifecycleStateForColumnID` (`:1958-1979`) maps column-name `"progress"` → `domain.StateProgress`, `"done"` → `domain.StateDone`. Same rename problem.
- **Why blocking:** Same package as the broader app layer; without the rename here, every test that exercises `defaultStateTemplates` or column-state mapping continues to write legacy values into the in-memory model, and the strict-canonical claim does not hold. Also, lines 1965-1975 reference `domain.StateProgress`/`StateDone` symbols — straight compile failure once 2.7 commits.
- **Suggested mitigation:** Add `internal/app/service.go` and `internal/app/service_test.go` to 2.9 `Paths:`. Update acceptance to cover `defaultStateTemplates` returning canonical IDs and `normalizeStateID` accepting canonical-only.

## 6. Finding — Plan 2.9 missing `internal/app/attention_capture.go` and snapshot.go state-symbol references (BLOCKING)

- **Severity:** blocking
- **Attack vector:** 3 + 13
- **Counterexample evidence:**
  - `internal/app/attention_capture.go:350,353,356,371` references `domain.StateProgress`/`StateDone`/`StateFailed` directly.
  - `internal/app/snapshot.go:419,1267` references `domain.StateProgress`/`StateDone` etc. (separate from the `Role` field that 2.6 owns).
  - `internal/app/service.go:556,623,625,627,639,644,694,1817` — many transition-rule checks against `domain.StateDone`/`StateProgress`.
  - `internal/app/service_test.go:3035,3055,3065,3092,3108,3186,3196,4573,4582,4609,4626,4636,4660,4693,4724,4733` — 16 test sites.
  - `internal/app/snapshot_test.go:410` — uses `domain.StateFailed` (unchanged) but other tests in the file may use the renamed symbols too.
- **Why blocking:** Symbol rename = full-tree compile break. Same root cause as Finding 1.
- **Suggested mitigation:** Roll into Finding 1's bundled-droplet remediation, OR enumerate every file in 2.9 `Paths:`.

## 7. Finding — Plan 2.9 missing `internal/config/config.go` and `config_test.go` (BLOCKING)

- **Severity:** blocking
- **Attack vector:** 3 + 13
- **Counterexample evidence:**
  - `internal/config/config.go:218` — `Search.States` default `[]string{"todo", "progress", "done"}`.
  - `internal/config/config.go:550` — fallback default same.
  - `internal/config/config.go:1094` — `isKnownLifecycleState` accepts legacy `"todo", "progress", "done", "failed", "archived"`. Under strict-canonical, this should return FALSE for `"complete"`/`"in_progress"` because those aren't in the list. Whichever way you flip it (accept canonical only OR keep legacy), the function is semantically inconsistent with the rest of the strict-canonical world unless updated.
  - `internal/config/config_test.go:326,814` — corresponding tests.
- **Suggested mitigation:** Add config files to 2.9 `Paths:`. Acceptance must require `isKnownLifecycleState` recognizes only canonical state values.

## 8. Finding — Plan 2.9 missing `internal/adapters/server/common/app_service_adapter_lifecycle_test.go` (BLOCKING)

- **Severity:** blocking
- **Attack vector:** 3 + 13
- **Counterexample evidence:** `:180,716,721,912` use `State: "done"` and `RevokedReason: "done"` — the second is a free-form reason field (incidental string), but the State field is genuinely strict-canonical-affected. Test `MoveActionItemStateRequest{State: "done"}` will be rejected by the new normalizer; the test must be rewritten to `State: "complete"`.
- **Suggested mitigation:** Add this file to 2.9 `Paths:`. Walk the file to distinguish state-machine inputs from incidental "done"/"completed" word usage.

## 9. Finding — Plan 2.9 missing `internal/adapters/server/mcpapi/extended_tools_test.go` (BLOCKING)

- **Severity:** blocking
- **Attack vector:** 3
- **Counterexample evidence:** `:1114, 2587, 2600` — `"state": "done"` in MCP test scaffolding. Strict-canonical = these tests fail.
- **Suggested mitigation:** Add to 2.9 `Paths:`.

## 10. Finding — `mcp_surface.go:227` `Completed bool json:"completed"` is a legitimate name collision risk (NIT)

- **Severity:** nit
- **Attack vector:** 4 (`ChecklistItem` JSON decoder edge cases)
- **Counterexample evidence:** `internal/adapters/server/common/mcp_surface.go:227` defines `Completed bool json:"completed"` as an MCP response shape (separate from `ChecklistItem`). Plan 2.7 renames `ChecklistItem.Done bool json:"done"` → `ChecklistItem.Complete bool json:"complete"`. The two now share the same conceptual word "complete" but on different types. Not a strict-canonical break (the `mcp_surface` field is unrelated to `ChecklistItem`), but worth flagging because a future developer may see two `complete`/`completed`-flavored fields and conflate them.
- **Suggested mitigation:** None required for Drop 2. Note in Notes that the `mcp_surface.Completed` field is unrelated.

## 11. Finding — Plan 2.6 does not bump `SnapshotVersion` (NIT — accept-as-is, but document)

- **Severity:** nit
- **Attack vector:** 12
- **Counterexample evidence:** `internal/app/snapshot.go:16` — `const SnapshotVersion = "tillsyn.snapshot.v5"`. Plan 2.6 adds `Role` field with `omitempty` JSON tag to `SnapshotActionItem`. Reading code (`:326-327`) does exact-string equality on `Version`, so old snapshots still load. New snapshots emit the `role` key only when non-empty. Old `v5` readers (none exist outside this binary, since `v999` is the only "bad" reference) ignore unknown JSON fields by default (`encoding/json` does not enforce strict fields).
- **Why nit:** Forward-compatible. No version bump needed pre-MVP.
- **Suggested mitigation:** Add a one-line note to plan 2.6 `Acceptance` explicitly saying "no `SnapshotVersion` bump required; field uses `omitempty` and `encoding/json` ignores unknown keys by default."

## 12. Finding — `ParseRoleFromDescription` has no production caller (NIT — YAGNI on a knife edge)

- **Severity:** nit
- **Attack vector:** 15
- **Counterexample evidence:** Plan explicitly cancels the migration runner ("No `internal/app/migrations/` package. The `ParseRoleFromDescription` helper is a domain helper"). With no migration runner, no MCP path that auto-parses `Role:` lines from incoming description text, and no CLI/TUI binding, the parser has zero production callers. Plan rationale: "exists for callers who want to opportunistically lift the value out of description prose at create time" — but that caller does not exist in any droplet.
- **Why nit:** Drop 3+ may add the caller; or it may not. Pre-MVP drop is OK with one orphan helper if the dev decides it's strategically useful. The plan should be explicit.
- **Suggested mitigation:** Either (a) name a concrete future caller and link to the drop where it lands, or (b) drop the parser from 2.2 and add it back when a caller materializes (true YAGNI). The plan's current "domain helper, exists in case someone wants it" framing is the weakest justification.

## 13. Finding — MD-cleanup carve-out boundary is fuzzy (NIT)

- **Severity:** nit
- **Attack vector:** 9
- **Counterexample evidence:** Plan Notes "Future refinement drop — MD content cleanup": "builders MAY make trivial in-section MD edits adjacent to their droplet's `paths` if the change is a single-sentence / single-phrase fix obviously broken by their code change." Ambiguity: a `templates/builtin/default-go.json` deletion (2.1) breaks any README.md / CLAUDE.md / PLAN.md sentence referring to "templates/builtin/default-go.json". A "single-phrase fix" is ill-defined when the surrounding paragraph cites the deleted file by name multiple times. A builder could reasonably read "this paragraph is wholly broken by my deletion" and rewrite the whole paragraph.
- **Why nit:** Build QA will catch over-rewriting via `git diff`, and the carve-out limits scope to "this drop only." But the boundary still lets one builder over-edit and the next builder under-edit, with no shared rule.
- **Suggested mitigation:** Tighten to: "Delete the broken phrase or replace with `<deleted in Drop 2 — see PLAN.md § 19.3>`. No paraphrasing surrounding sentences. Anything beyond a delete-or-stub is out of scope and routes to a future MD-cleanup refinement drop."

## 14. Finding — Acceptance criteria asymmetry (NIT)

- **Severity:** nit
- **Attack vector:** 10
- **Counterexample evidence:** 2.7 has 8 acceptance bullets; 2.4 has 5; 2.6 has 4; 2.13 has 5. 2.6's 4 bullets are: round-trip, omit-empty, JSON shape, mage green — adequate. 2.4 covers schema + scanner + insert/update + test + pre-MVP rule + mage green; OK. 2.10 covers payload + universal-allow + enforcement-unchanged + pre-MVP + mage; OK. 2.13 covers UUID-and-dotted-get + mutation-rejects + CLI + ambiguous + mage ci; OK. No actual hidden work — but several droplets (2.5, 2.6, 2.13) lean heavily on "mage test-pkg green" without enumerating what the test must exercise.
- **Suggested mitigation:** None required, but consider adding "snapshot file with `role` set to each of 9 valid values round-trips" to 2.6 instead of just "non-empty `Role` value."

## 15. Finding — Pre-MVP DB-deletion risk language is OK but could explicit-link to dev workflow (NIT)

- **Severity:** nit
- **Attack vector:** 14
- **Counterexample evidence:** Plan Pre-MVP rules section: "dev deletes `~/.tillsyn/tillsyn.db` between schema or state-vocab-changing units." This is repeated under 2.4, 2.10. But Unit B-zero (template deletion) and Unit D (dotted-address reads) don't say "no DB deletion needed" — implicit but should be explicit.
- **Suggested mitigation:** Add a "DB action: NONE" line under 2.1, 2.11, 2.12, 2.13 acceptance. And a more prominent warning under 2.4 / 2.7 / 2.10: "DELETE `~/.tillsyn/tillsyn.db` BEFORE running `mage ci` for this droplet." Today the rule is in Scope, not on individual droplet acceptance — it's easy to miss.

## 16. Finding — `blocked_by` graph cycle check (PASS — no cycle)

- **Severity:** information / PASS
- **Attack vector:** 2
- **Counterexample evidence:** Edge set: 2.3→2.2; 2.4→2.3; 2.5→2.4; 2.6→2.3; 2.7→2.6; 2.8→2.7; 2.9→2.8; 2.10→2.9; 2.11→2.10; 2.12→2.11; 2.13→2.12. Topological order: 2.1 (no parent) → 2.2 → {2.3} → {2.4, 2.6} → {2.5} (after 2.4) → 2.7 (after 2.6) → 2.8 → 2.9 → 2.10 → 2.11 → 2.12 → 2.13. Linear post-2.6, no cycles.
- **Conclusion:** No counterexample on this vector.

## 17. Finding — Same-package transitive blocker (PASS for declared scope, FAIL for omitted paths)

- **Severity:** information / PASS for declared edges
- **Attack vector:** 1
- **Counterexample evidence for 2.7 / `internal/domain` chain:** 2.3 (touches `internal/domain/action_item.go`) → 2.7 transitive via 2.6 (cross-unit, different package). Confirmed transitive; 2.7 → 2.6 → 2.3 is acyclic and ordered. For 2.10/2.11 (both touch test fixtures + doc-comments in `internal/domain`): 2.11 `Blocked by: 2.10`. 2.10 `Blocked by: 2.9` (cross-unit). Both serial, no race.
- **Conclusion:** Declared blockers are correct. The bug is in the missing `Paths:` entries, not in the `blocked_by` edges — see Findings 1-9.

## 18. Finding — Unit B-zero atomicity (PASS)

- **Severity:** information / PASS
- **Attack vector:** 11
- **Counterexample attempt:** Could deleting the JSON files separately from `embed.go` leave the build broken? Plan 2.1 explicitly bundles all four paths into one droplet = one git commit. `git rm` of the four paths in one commit ⇒ atomic. The build never sees an intermediate state where `embed.go` references missing files. Confirmed.

## 19. Finding — Hylla index covers Go but not the JSON / MD references in templates/ (PASS)

- **Severity:** information / PASS
- **Attack vector:** 5
- **Counterexample attempt:** `git grep "evanmschultz/tillsyn/templates"` returns empty. `git grep "templates.ReadFile"` returns empty. `git grep "templates.Files"` returns empty. Confirmed zero importers. Plan 2.1 deletion is safe.

## 20. Finding — `AllowedParentKinds` deletion is safe (PASS)

- **Severity:** information / PASS
- **Attack vector:** 6
- **Counterexample attempt:** Searched `AllowedParentKinds` — only `internal/domain/kind.go:99` (definition), `internal/domain/domain_test.go:680-714` (test), `internal/app/snapshot.go:448` (doc-comment), `internal/adapters/storage/sqlite/repo.go:300` (doc-comment). Plan accurate.

## 21. Finding — Mutation-operation enumeration is complete (PASS)

- **Severity:** information / PASS
- **Attack vector:** 8
- **Counterexample attempt:** Searched `extended_tools.go` for action-item operations. Found `get|list|search|create|update|move|move_state|delete|restore|reparent`. Plan 2.13 names all of them. No `archive`/`unarchive` separate operation (delete with `mode=archive` handles archival). Plan accurate.

## 22. Finding — Resolver location decision is sound (PASS)

- **Severity:** information / PASS
- **Attack vector:** 7
- **Counterexample attempt:** Searched for any pure-domain function taking a Repository param — none. The architectural argument for placing the resolver in `internal/app` holds.

## Verdict Summary

**FAIL.** 9 blocking counterexamples (Findings 1, 3, 4, 5, 6, 7, 8, 9; Finding 1 is the umbrella but each named missing file is independently blocking) and 6 nits.

The single most damaging counterexample is **Finding 4** — `internal/adapters/server/common/capture.go:296-312` defines an independent `canonicalLifecycleState` coercion site that the plan completely fails to enumerate. Strict-canonical state vocabulary is one of the two named hard constraints of Drop 2 (per the spawn-prompt's Hard Constraints section), and the plan's grep-sweep acceptance criterion at 2.9 (`git grep -E '"in-progress"' -- '*.go'` returns empty) cannot be satisfied without touching this file. If the planner missed this file, the strict-canonical claim is unreachable as decomposed.

**Single fix path:** Either (a) collapse 2.7+2.8+2.9 into one mega-droplet that touches every Go file with a `domain.StateDone`/`StateProgress` reference or a legacy state literal in one atomic commit (best for the symbol-rename half of the work), OR (b) keep the three-droplet split but rewrite each droplet's `Paths:` to enumerate all 14 missing files listed in Findings 1-9, AND add a per-droplet acceptance line "`mage build ./...` clean (full-tree compile, not just per-package test)" to catch transitive symbol-rename fallout. Option (a) is simpler and matches the strict-canonical "hard cutover" framing.

The plan also has 6 nits (Findings 2, 10, 11, 12, 13, 14, 15) that should be addressed in the same revision pass for a clean R2 review.
