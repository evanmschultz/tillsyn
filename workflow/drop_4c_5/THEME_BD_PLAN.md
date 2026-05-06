# Drop 4c.5 — Theme B + Theme D Plan (Dev Escape Hatches + Pre-Cascade Hygiene)

**Author:** Theme-BD planner subagent.
**Date:** 2026-05-05.
**Scope owned:** REVISION_BRIEF §3.2 (Theme B — dev escape hatches, ~2 droplets) and §3.4 (Theme D — pre-cascade hygiene, ~1-2 droplets).
**Out of scope:** Theme A (silent-data-loss), Theme C (STEWARD R1-R3, R5), Theme E (4a/4b residue), Theme F (template ergonomics).

## 1. Droplet Inventory

Four droplets total. Theme B is sequenced — `B.2` reads `failed` items, which is much more useful once `B.1` exists to clear stuck ones. Theme D is independent of B; `D.2` is independent of `D.1` but lands second so the `mage ci` warning sweep sees the post-cleanup state.

- **B.1** — `till action_item supersede` CLI (Service method + adapter wiring + cobra subcommand + tests).
- **B.2** — `till action_item list --state failed` CLI (Service.ListActionItemsByState + cobra subcommand + tests). `blocked_by: B.1`.
- **D.1** — `go.mod` `replace` cleanup (strip every replace except the fantasy-fork retention).
- **D.2** — Accumulated `mage ci` / vet / `gopls` hint sweep (capture, classify, fix). `blocked_by: D.1`.

## 2. Droplet Specs

### Droplet B.1 — Supersede CLI

**Title:** `B.1 — TILL ACTION_ITEM SUPERSEDE CLI`

**State:** done

**Goal:** Land the `till action_item supersede <id> --reason "..."` escape hatch. Marks a `failed` action item with `metadata.outcome = "superseded"` and transitions it `failed → complete`, bypassing the always-on terminal-state guard at `service.go:1079` (`domain.IsTerminalState(fromState) && fromState != toState`). Asserts dev intent at the binary boundary — bypasses parent-blocks-on-incomplete-child as a side effect (the superseded item is now `complete`, so it stops blocking its parent).

**Files / paths to modify:**

- `internal/app/service.go` — add `Service.SupersedeActionItem(ctx, actionItemID, reason string) (domain.ActionItem, error)` (new method).
- `internal/adapters/server/common/app_service_adapter_mcp.go` — add MCP-adapter passthrough method `SupersedeActionItem` so the CLI flow + future MCP tool share the boundary; do NOT yet expose a `till.action_item(operation=supersede)` MCP op (CLI-only escape hatch per REVISION_BRIEF L1/L2 — dev-driven, not agent-driven).
- `cmd/till/action_item_cli.go` — add `runActionItemSupersede(ctx, svc, opts, stdout)` next to `runActionItemMutationGate`. Accept UUID-only input via the existing `ValidateActionItemIDForMutation`.
- `cmd/till/main.go` — add `actionItemSupersedeCmd` cobra subcommand under `actionItemCmd` (mirror `actionItemDeleteCmd` shape at lines 786-790); register `--reason` string flag on the new command; route through `runFlow(ctx, "action_item.supersede")`. Extend the dispatch switch at `main.go:2511-2514` so `action_item.supersede` calls `runActionItemSupersede` (NOT the not-implemented mutation gate).
- `cmd/till/action_item_cli_test.go` — table-driven tests for the new CLI flow (UUID-shaped input, dotted rejected, empty-reason rejected, end-to-end success path + parent-unblocks-on-superseded-child path).
- `internal/app/service_test.go` — table-driven tests for `Service.SupersedeActionItem` (failed→complete success, todo-state rejected, in_progress rejected, complete rejected, missing-action-item rejected, empty-reason rejected, audit-trail recorded in `metadata.outcome` + `metadata.transition_notes` or equivalent existing field).
- `internal/adapters/server/mcpapi/handler_steward_integration_test.go` — un-skip `TestStewardIntegrationDropOrchSupersedeRejected` (line 459-461) and adapt the test to the new SupersedeActionItem path. The skipped test was waiting for exactly this method.

**Packages affected:**

- `github.com/evanmschultz/tillsyn/internal/app` (Service, tests)
- `github.com/evanmschultz/tillsyn/internal/adapters/server/common` (adapter passthrough)
- `github.com/evanmschultz/tillsyn/cmd/till` (CLI wiring + tests)
- `github.com/evanmschultz/tillsyn/internal/adapters/server/mcpapi` (un-skip the integration test that already exists)

**Acceptance criteria (yes/no-verifiable):**

1. `till action_item supersede <UUID> --reason "<text>"` on a `failed` action item transitions it to `complete` with `metadata.outcome = "superseded"` and persists the reason on `metadata.transition_notes` (or an equivalent existing free-text metadata field — builder picks one, documents the choice in the SupersedeActionItem doc-comment, no new field added to `ActionItemMetadata`).
2. `till action_item supersede` on a `todo` / `in_progress` / `complete` / `archived` item rejects with a typed error (`domain.ErrTransitionBlocked` wrapped with a "supersede only applies to failed items" hint).
3. `till action_item supersede` on a dotted address rejects with `app.ErrMutationsRequireUUID` (same gate as other mutations).
4. `till action_item supersede` with empty `--reason` rejects with a clear error before any service call (the supersede CLI's whole point is recording dev intent — empty reason defeats it).
5. After supersede on a `failed` child, the parent action item is no longer blocked by it — `Service.MoveActionItem(parent, completeColumn)` succeeds (assuming all other children are also complete). Verified via `internal/app/service_test.go` integration test that pairs supersede + parent move.
6. `metadata.outcome = "superseded"` is the existing recognized value (`internal/adapters/server/common/app_service_adapter_mcp.go:1163`); supersede does NOT introduce a new outcome value.
7. `mage ci` passes; coverage on the new method ≥ 70% (matches project minimum).
8. The previously-skipped `TestStewardIntegrationDropOrchSupersedeRejected` runs and either passes (if the supersede path correctly rejects drop-orch-scoped callers per finding 5.C.13) or is updated with a new skip rationale tied to a future drop — never silently left skipped under the old rationale.

**Test scenarios (table-driven entries the builder MUST add):**

| Case                                                          | Input                                                | Expected                                                    |
| ------------------------------------------------------------- | ---------------------------------------------------- | ----------------------------------------------------------- |
| supersede failed item with reason                             | UUID, `--reason "rejected by dev"`                   | success; state=complete; outcome=superseded; reason stored  |
| supersede todo item                                           | UUID of todo item, valid reason                      | error wrapping `ErrTransitionBlocked`                       |
| supersede in_progress item                                    | UUID of in_progress item, valid reason               | error wrapping `ErrTransitionBlocked`                       |
| supersede already-complete item                               | UUID of complete item, valid reason                  | error (idempotent reject, not silent no-op)                 |
| supersede archived item                                       | UUID of archived item                                | error                                                       |
| supersede with empty reason                                   | UUID, `--reason ""` (or unset)                       | reject before service call                                  |
| supersede with whitespace-only reason                         | UUID, `--reason "   "`                               | reject (treats whitespace as empty)                         |
| supersede via dotted address                                  | `tillsyn:1.5.2`, valid reason                        | reject with `ErrMutationsRequireUUID`                       |
| supersede missing UUID                                        | UUID that does not exist                             | propagates `app.ErrNotFound` (no panic, no silent success)  |
| parent-unblocks-after-child-supersede                         | failed child + completable siblings                  | `MoveActionItem(parent → complete)` succeeds                |

**Blocked by:** none (Theme B head-of-line).

**Falsification mitigations (top-3 attacks the builder MUST pre-empt):**

1. **Cascading children left dangling.** When superseding a `failed` item that has its OWN children in non-terminal state, the supersede must NOT cascade — superseded does not mean "all my children are now complete." Document explicitly in `SupersedeActionItem` godoc: supersede operates on ONE node; descendants keep their own state. Test: failed item with todo descendant — supersede succeeds (the parent transitions failed→complete, descendants untouched), and a SUBSEQUENT attempt to move the SUPERSEDED item's grandparent to complete is gated normally by the still-todo grandchild via the existing `ensureActionItemCompletionBlockersClear` (`internal/app/mutation_guard.go:23`).
2. **Auth-revoke side effect missed.** `failed`-state items already trigger auth auto-revoke (PLAN.md §19.1 line 1561). If we transition `failed → complete` via supersede and the auth was revoked on the original failed-transition, no double-revoke is triggered. If auth-revoke also fires on `complete`, the implementation must be idempotent OR the supersede path must skip the second revoke. Test: pair supersede with the revoke path; assert no double-revoke + no error.
3. **Capability-guard bypass.** Supersede is a privileged dev-only escape hatch. The Service method MUST still go through `enforceMutationGuardAcrossScopes` for the calling actor — pre-MVP the CLI is single-user dev, but the guard plumbing should be uniform with `MoveActionItem`. Use `domain.CapabilityActionMarkComplete` (from `service.go:1067`) so the existing capability surface covers it; do NOT introduce a new `CapabilityActionSupersede` (YAGNI — pre-MVP, single-user, the guard is structural symmetry only).

---

### Droplet B.2 — Failure Listing CLI

**Title:** `B.2 — TILL ACTION_ITEM LIST --STATE FAILED CLI`

**Goal:** Pre-TUI dev visibility into failed action items. `till action_item list --state failed [--project <slug>]` renders a table of failed items so dev can see what needs supersede / re-dispatch. Table format mirrors existing `auth request list --state pending` output via `writeCLITable` (`cmd/till/cli_render.go:157`).

**Files / paths to modify:**

- `internal/app/service.go` — add `Service.ListActionItemsByState(ctx, projectID string, state domain.LifecycleState, includeArchived bool) ([]domain.ActionItem, error)`. Implementation: call existing `Service.ListActionItems` and filter by `LifecycleState`. No new repo method — the in-memory filter is acceptable pre-MVP at expected scale (<1k items per project).
- `cmd/till/action_item_cli.go` — add `runActionItemList(ctx, svc, opts, stdout)`. Accept `--state` flag (validate against the closed `LifecycleState` set: todo / in_progress / complete / failed / archived). Accept optional `--project <slug>` to scope; require either `--project` or a default-only-project fallback (reject when ambiguous).
- `cmd/till/main.go` — add `actionItemListCmd` cobra subcommand under `actionItemCmd`. Add `actionItemCommandOptions.state` and `actionItemCommandOptions.includeArchived` fields (extend the struct at `main.go:262`). Register `--state` (string, default `"failed"`) + `--project` (string) flags on the list command. Wire through `runFlow(ctx, "action_item.list")` and add case in dispatch switch.
- `cmd/till/action_item_cli_test.go` — table-driven tests for `runActionItemList` (state filter behavior, empty-result rendering, project resolution, invalid state rejected).
- `internal/app/service_test.go` — table-driven tests for `Service.ListActionItemsByState` (per-state filter correctness, archived-flag interaction, project-not-found, empty result).

**Packages affected:**

- `github.com/evanmschultz/tillsyn/internal/app`
- `github.com/evanmschultz/tillsyn/cmd/till`

**Acceptance criteria:**

1. `till action_item list --state failed --project tillsyn` renders a table with columns: dotted-address, UUID, title, kind, role, updated_at. Empty-state message: "No failed action items in project tillsyn."
2. `till action_item list --state <invalid>` rejects with a clear error naming the valid set (todo / in_progress / complete / failed / archived).
3. `till action_item list --state failed` without `--project` AND without exactly one project on the system rejects with `--project` hint (matches the existing dotted-address-without-project pattern at `action_item_cli.go:97-99`).
4. Default `--state` is `"failed"` (the canonical pre-TUI use case); `till action_item list --project tillsyn` (no `--state`) still behaves as `--state failed`.
5. `--include-archived` flag (off by default) extends the filter to also surface archived items; off by default since archived ≠ failed.
6. Output is laslig-table styled in human terminals, JSON-friendly (or at least machine-parseable) when piped — match the existing `writeCLITable` behavior; do NOT add a JSON flag pre-MVP (YAGNI).
7. `mage ci` passes; coverage on the new path ≥ 70%.

**Test scenarios:**

| Case                                                                | Input                                              | Expected                                                       |
| ------------------------------------------------------------------- | -------------------------------------------------- | -------------------------------------------------------------- |
| list failed items in project with two failed + three non-failed     | `--state failed --project tillsyn`                 | table with two rows, ordered by updated_at desc                |
| list failed items in project with zero failed                       | `--state failed --project tillsyn`                 | empty-state row "No failed action items..."                    |
| invalid state                                                       | `--state weird --project tillsyn`                  | error naming valid states                                      |
| no project hint, multiple projects exist                            | `--state failed`                                   | error suggesting `--project`                                   |
| state=todo                                                          | `--state todo --project tillsyn`                   | table of todo items                                            |
| state=in_progress                                                   | `--state in_progress --project tillsyn`            | table of in_progress items                                     |
| state=archived without `--include-archived`                         | `--state archived --project tillsyn`               | table of archived items (state=archived implies include)       |
| `--include-archived` + state=failed                                 | `--state failed --include-archived --project tillsyn` | table of failed items including those that are also archived |
| project slug typo                                                   | `--project tillsynx --state failed`                | error from `GetProjectBySlug`                                  |

**Blocked by:** `B.1` (the listing UX strongly implies a follow-up supersede; landing them together is fine, but B.1 must merge first so the dev workflow is real not aspirational).

**Falsification mitigations:**

1. **Output stability under archived items.** A "failed AND archived" item should appear ONCE not TWICE — the filter must handle the orthogonal axes (lifecycle state + archived flag) without double-counting. Test: seed an item that is both `state=failed` AND `archived_at != nil`, call `ListActionItemsByState(failed, includeArchived=true)`, assert exactly one entry.
2. **Project-resolution drift vs `action_item get`.** The list command must use the SAME project-resolution rule as `runActionItemGet` (slug-prefix shorthand explicitly does NOT apply here — list is project-scoped not item-scoped). Builder: explicitly document this divergence in the cobra `Long:` text. Do NOT accept `tillsyn:<state>` as a slug-prefix shorthand on list (different command shape).
3. **Performance regression at scale.** Pre-MVP scale (~hundreds of items) the in-memory filter is fine. Add a one-line comment in `ListActionItemsByState` explaining the filter is in-memory and the indexed-query refactor is deferred until measurement justifies it. Test does not enforce performance, but the doc-comment is mandatory.

---

### Droplet D.1 — `go.mod` `replace` Directive Cleanup

**Title:** `D.1 — STRIP NON-FANTASY-FORK GO.MOD REPLACES`

**State:** done

**Goal:** PLAN.md §19.1 first bullet — keep ONLY the fantasy-fork `replace` line in `go.mod`. Every other `replace` is upstream-version pinning that has accumulated experimentally and is now risk surface (a stale path can silently break builds).

**Round-2 outcome (orchestrator-amended semantics):** spec acceptance criterion #1 ("exactly ONE replace directive") was over-strict. Round 1 surfaced 2 load-bearing pins (L1 `ultraviolet`, L2 `chroma/v2`) plus 1 load-bearing local fork (`teatest_v2`). Round 2 restored those 3 with explicit `// load-bearing: <reason>` annotations naming the consumer constraint. Final state: 4 `replace` directives — 1 fantasy-fork + 3 load-bearing — every other (19) experimental self-pin remains stripped. `mage ci` green.

**Files / paths to modify:**

- `go.mod` — delete every `replace` directive EXCEPT the fantasy-fork retention. Also delete the matching `teatest/v2` local-path replace at line 5 (`./third_party/teatest_v2`). Decision documented below in §3 Notes.
- `go.sum` — regenerate via `go mod tidy` (the module-file-only operation PLAN.md §19.1 already exempts from the no-raw-`go` rule).
- (Possibly) `third_party/teatest_v2/` — delete the directory if the local-path replace is removed and no other consumer remains. Builder: verify with `rg -n "teatest" -g '!go.sum' -g '!go.mod'` before deletion.
- `magefile.go` — none (mage targets do not reference `replace` directives).

**Packages affected:** module-level (no Go package edits — purely module manifest).

**Reference-only `files` (read but not edited):** `PLAN.md` §19.1 line 1555 (defines the retention rule), `internal/adapters/embeddings/fantasy/generator.go`, `internal/adapters/embeddings/fantasy/generator_test.go` (consumers of the fantasy-fork `replace`).

**Acceptance criteria:**

1. `go.mod` contains exactly ONE `replace` directive: the fantasy-fork (`charm.land/fantasy => github.com/evanmschultz/fantasy v0.0.0-...`). Annotated with `// fantasy-fork: <rationale>` per PLAN.md §19.1.
2. `go.mod` does NOT contain the `teatest/v2 => ./third_party/teatest_v2` local-path replace (per PLAN.md "delete any that point at local filesystem paths left over from experimentation").
3. `go.sum` regenerated via `go mod tidy`; no spurious churn beyond the deleted-replace fallout.
4. `mage ci` passes — this is the gate that proves no downstream builds were silently relying on a stripped pin.
5. If `third_party/teatest_v2/` is deleted, `git status` shows the directory removed with no orphan references in source.
6. The fantasy-fork `replace` line is the only line in `go.mod` matching the regex `^replace\b` after the cleanup.

**Test scenarios:** module-file edits do not directly add unit tests. The acceptance gate is the existing `mage ci` (which runs `go test ./...` across all packages). Builder MUST NOT add a "test that go.mod has only one replace" test — that's tautological self-reference. The `mage ci` green is the only test.

**Blocked by:** none.

**Falsification mitigations:**

1. **Stripping a load-bearing pin breaks downstream silently.** The PLAN.md call-out is explicit: "a stray `replace` that points at a missing path silently breaks every downstream build." The mitigation is `mage ci` clean post-strip — if any package fails to compile or test, the strip removed a real pin not a stale one. Builder MUST NOT force-fix (e.g. by adding the replace back AND a workaround); instead, surface the failure to the orchestrator and document which replace was load-bearing. Pre-build evidence: cross-check each replace line against `git log --oneline -- go.mod` (HEAD `66c354e refactor(all): fix bad module name` is the last go.mod-touching commit; check what reasons each replace was added).
2. **`teatest/v2 => ./third_party/teatest_v2` is a local fork, not a stale pin.** Inspect `third_party/teatest_v2/` contents before deleting. If it carries fork-only patches (the upstream module would not behave the same), keep it as a fork-style replace and document inline; do NOT delete just because PLAN.md says "experimental left-overs." Builder: check `git log -- third_party/teatest_v2/` before assuming it's removable.
3. **`go mod tidy` flips indirect/direct module classification.** Tidy may move some `// indirect` comments. The diff should be reviewed for "did tidy resolve a transitive that was being pinned?" — if yes, the pin was load-bearing for version selection and stripping it changed dependency versions. Builder: include the full `go.sum` diff in the worklog; orchestrator reviews before commit.

---

### Droplet D.2 — Accumulated Vet / Gopls / `mage ci` Hint Sweep

**Title:** `D.2 — SWEEP ACCUMULATED VET + GOPLS HINTS`

**State:** done

**Goal:** REVISION_BRIEF §3.4 second bullet — capture every `mage ci` warning + every gopls/LSP diagnostic accumulated through 4a + 4b + 4c that has NOT been individually triaged. Classify each as (a) fix-now (one-line cleanup), (b) refinement-route-to-future-drop (real issue, but scope creep here), or (c) ignore (false positive / known accepted behavior). Fix the (a) bucket inline; route the (b) bucket to `project_drop_4c_5_refinements_raised.md` (memory) for the orchestrator to roll forward.

**Files / paths to modify:**

- `workflow/drop_4c_5/D2_HINT_SWEEP.md` — NEW. Captures the raw hint inventory + classification table. This file is the dev-readable artifact of the sweep; the actual fixes land in (b) below.
- (TBD per sweep finding) — one or more `internal/...` files for fix-now items. Builder MUST list every touched file in the build worklog; planner cannot enumerate ahead of measurement (the hints are not yet captured).

**Packages affected:** TBD per sweep; bound the scope to "no refactor over 50 LOC per file." Anything beyond that bound routes to refinement.

**Reference-only `files`:** `mage ci` output (regenerated by builder), gopls workspace diagnostics (regenerated by builder via LSP calls), `~/.claude/projects/.../memory/project_drop_4a_refinements_raised.md`, `project_drop_4b_refinements_raised.md` (to avoid double-raising items already captured).

**Acceptance criteria:**

1. `D2_HINT_SWEEP.md` exists at `workflow/drop_4c_5/D2_HINT_SWEEP.md` with three sections: `## Captured Hints`, `## Fix-Now Bucket`, `## Routed-to-Refinement Bucket`. Captured Hints lists every distinct warning from `mage ci` + gopls workspace diagnostics with file:line refs.
2. Every Fix-Now bucket entry has a corresponding inline fix in the same droplet build. After fixes, re-running `mage ci` produces strictly fewer warnings (zero ideally; reduction is mandatory, total elimination is not).
3. Every Routed-to-Refinement entry has a one-line rationale ("scope > 50 LOC", "needs design discussion", "blocked on unrelated drop") and an explicit forwarding note for `project_drop_4c_5_refinements_raised.md` (so the orchestrator carries it forward).
4. `mage ci` passes. No new warnings introduced.
5. No fix touches `cmd/till/main.go` (163KB), `cmd/till/main_test.go` (132KB), or `internal/app/service.go` (98KB) for refactor-style cleanup — those are on the Drop-1 R1 split list (memory `project_drop_1_refinements_raised.md`). One-line cleanups in these files are fine; structural changes are out of scope.
6. Coverage stays at or above the 70% project minimum on every touched package.

**Test scenarios:** sweep-driven; no canned scenarios. Builder MUST add one regression test per fix-now item that has a behavioral signature (e.g. an unused-variable fix is purely structural and needs no new test; a nil-check addition needs a test that triggers the nil path).

**Blocked by:** `D.1` (so the sweep observes the post-cleanup state and does not waste time on warnings that disappear when stale `replace`s drop).

**Falsification mitigations:**

1. **Scope creep into Drop-1 R1 territory.** The 22kLOC `internal/tui/model.go` split is already on the refinement list. D.2 must NOT trigger a partial split. Builder MUST refuse refactor-style suggestions in TUI/main and route them. Acceptance criterion 5 enforces this.
2. **Capture incompleteness — the sweep misses warnings only visible under non-default builds.** `mage ci` covers race + cover + format. The sweep MUST also capture gopls workspace diagnostics (via LSP tool — workspace symbols, references, and diagnostics) for the `internal/...` tree. Builder: include the LSP diagnostic snapshot in `D2_HINT_SWEEP.md` § `## Captured Hints` with explicit "captured via gopls workspace diagnostics on <date>" attribution.
3. **Route-to-refinement becomes a punt.** Every routed item MUST land in `project_drop_4c_5_refinements_raised.md` with enough detail that a future planner can pick it up without re-doing the diagnosis. Acceptance criterion 3 enforces the rationale; a routed entry without a clear reason (e.g. "skip for now") is a falsification finding. Plan-QA falsification will spot-check 1-2 routed entries against the diagnosis depth.

## 3. Notes — Cross-Cutting Design Tradeoffs

### 3.1 Supersede Semantics: Cascade-on-Itself Implications

**Question:** when a `failed` action item with non-terminal descendants is superseded, does the supersede cascade to descendants?

**Decision (B.1):** NO cascade. Supersede operates on EXACTLY the named node. Rationale:

- The dev's mental model when typing `till action_item supersede <UUID>` is "I am clearing THIS node so its parent can move forward." Cascading would silently mutate descendants the dev did not name, which is the opposite of the audit-trail behavior we want for an escape hatch.
- The existing `ensureActionItemCompletionBlockersClear` (`internal/app/mutation_guard.go:23`) handles the parent-chain correctly already: superseding the named node moves it `failed → complete`; the parent-blocks gate then sees the named node's state as `complete` and stops blocking the parent. Descendants of the superseded node are still children of a now-complete-but-historically-failed parent — they keep their own state and the orchestrator decides what to do with them next.
- Cascade would conflate two distinct semantic operations: "clear this failure" (supersede) and "abandon this whole subtree" (a future delete-recursive path that does not exist yet and is OUT OF SCOPE for B.1).

**Falsification follow-up:** plan-QA falsification should hit this with a specific scenario — failed parent + still-running grandchild — and confirm the supersede succeeds without touching the grandchild.

### 3.2 `teatest/v2` Local Replace — Strip or Keep?

**Recommendation (D.1):** STRIP. Inspect `third_party/teatest_v2/` first; if it's a vanilla copy (no fork patches), the replace is dead weight and should go. If it carries fork patches, keep it but rewrite the replace to point at a published fork (matches the fantasy-fork pattern) rather than a local path. Local-path replaces are exactly the "experimental left-over" PLAN.md §19.1 calls out.

### 3.3 B.1 ↔ B.2 Sequencing

`B.1` BEFORE `B.2` because:

- B.2 (failure listing) without B.1 (supersede) gives the dev a list of stuck items they can SEE but cannot CLEAR. That's worse than today (the dev can already query SQLite directly when desperate) — the listing shapes expectations that the next action is a supersede CLI call, so the supersede has to exist.
- B.1 alone is operationally complete; the dev can use UUIDs from SQLite-direct queries until B.2 lands.

### 3.4 `metadata.outcome = "superseded"` Field-Reuse

The MCP adapter ALREADY recognizes `"superseded"` as a valid `metadata.outcome` value (`internal/adapters/server/common/app_service_adapter_mcp.go:1163`). B.1 reuses this — no new outcome value, no new field on `ActionItemMetadata`. The supersede-reason text lands on `metadata.transition_notes` (existing field at `internal/domain/workitem.go` ActionItemMetadata.TransitionNotes — a pre-existing free-form string). Builder confirms field choice; if `transition_notes` is unsuitable for any reason, the alternative is `metadata.completion_contract.completion_notes` (existing). Do NOT add a new `Metadata.SupersedeReason` field.

### 3.5 Out-of-Scope Today

- A `till.action_item(operation=supersede)` MCP tool. Pre-MVP, the CLI is enough; agent-driven supersede via MCP is a future drop (tied to the auth-revoke-on-superseded path which itself depends on Drop 1's `failed` state hardening).
- A "delete recursive" or "abandon subtree" CLI. Distinct from supersede; would need its own design discussion.
- TUI rendering of `failed` items. PLAN.md §19.1 line 1572 is explicit: deferred post-dogfood. B.2 is the pre-TUI substitute.

## 4. Verification Gates

Each droplet's acceptance gate is `mage ci` clean. Builder spawn prompts MUST include:

- Hylla artifact ref `github.com/evanmschultz/tillsyn@main` (planner notes Hylla is stale post-Drop-4c-merge; builder uses LSP / Read / Grep / `git diff` until next reingest).
- Explicit "do NOT commit" directive (per F.7-CORE REV-13).
- Section 0 SEMI-FORMAL REASONING required.
- Tillsyn-flow output style.
- model: opus.

Plan-QA proof + plan-QA falsification fire on this MD before any builder spawns. Build-QA proof + build-QA falsification fire on each droplet's resulting build before commit.

## 5. References

- `workflow/drop_4c_5/REVISION_BRIEF.md` §3.2, §3.4 — scope.
- `PLAN.md` §19.1 lines 1555-1574 — original Drop 1 lifecycle items (B.1 supersede + D.1 go.mod cleanup are direct lifts from this section).
- `internal/app/service.go:1042-1127` — `MoveActionItem` (the path B.1 bypasses).
- `internal/app/service.go:1079` — terminal-state guard B.1 must bypass for `failed → complete`.
- `internal/app/mutation_guard.go:23` — `ensureActionItemCompletionBlockersClear`.
- `internal/adapters/server/common/app_service_adapter_mcp.go:1151-1169` — `validateMetadataOutcome` (already recognizes `"superseded"`).
- `internal/adapters/server/mcpapi/handler_steward_integration_test.go:449-461` — `TestStewardIntegrationDropOrchSupersedeRejected` (skipped, waiting for B.1).
- `cmd/till/action_item_cli.go` + `cmd/till/main.go:718-812` — current `action_item` cobra wiring.
- `cmd/till/cli_render.go:157-188` — `writeCLITable` / `writeCLIKV` (for B.2 rendering).
- `internal/domain/workitem.go:106-119` — `CompletionContract` doc-comment naming the post-MVP supersede CLI.
- `go.mod` — current `replace` block.
