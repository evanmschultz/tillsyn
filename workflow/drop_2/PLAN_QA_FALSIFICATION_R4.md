# DROP_2 — Plan QA Falsification, Round 4

**Verdict:** pass
**Date:** 2026-05-01

This Round 4 falsification reviews the orchestrator's surgical patches (R4-1, R4-2) closing Round 3's two falsification blockers, and re-attacks every other Round 3 finding to confirm nothing else has drifted. **Both Round 3 blockers are resolved.** No new Round 4 blockers found. Two non-blocking nits surfaced — one about Notes-section historical-record clarity, one about leading-zero acceptance not being explicitly enumerated in the test list — both downgrade-acceptable.

Severity ladder (highest first): zero blockers; two nits.

---

## 1. Round 3 Blocker Resolution Summary

| R3 # | R3 Title (paraphrased)                                                       | R4 Status | Verification at HEAD                                                                                                                                                                                                                                  |
| ---- | ---------------------------------------------------------------------------- | --------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| R3-§2.1 | `position ASC` conditional in 2.10 ORDER BY is unsafe (column-scoped column) | resolved  | PLAN.md `:287` now says `"deterministically ordered by created_at ASC, id ASC"` exclusively; explicit warning *"NOT to be confused with the existing column-scoped position field ... that field is TUI-arrangement-only and would shift dotted addresses under user drag-reordering. The resolver MUST use created_at ASC, id ASC exclusively."* (R4-1 patch). No `position ASC` alternative remains in the canonical droplet spec. |
| R3-§2.2 | `internal/domain/kind_capability_test.go:19` `Done: false` missed from 2.7 Paths | resolved  | PLAN.md `:163` now reads `"rename Done: false ChecklistItem field literal at :19 to Complete: false; rename RequireChildrenDone: test fixtures at :35 AND :73 to RequireChildrenComplete:"`. R4-2 patch verified at HEAD: `internal/domain/kind_capability_test.go:19` carries `CompletionChecklist: []ChecklistItem{{ID: "c1", Text: "run tests", Done: false}}` (read confirmed). Cite is line-accurate. |
| R3-§2.3 | PROOF Round 3's PASS verdict contradicted by §2.1 + §2.2                     | resolved  | Structurally derivative — closing §2.1 and §2.2 closes §2.3. No fresh PROOF round demanded; the orchestrator's R4 self-edit is the appropriate response.                                                                                              |

Round 3 carryover nits (R2-F7, R2-F9, R2-F11, R3 nit) are explicitly accepted as downgrade-acceptable in PLAN.md `:354-359` (Round 4 revision summary). No new attack surface introduced.

---

## 2. Round 4 Findings

### 2.1 (nit) PLAN.md Notes "Round 3 revision summary" R3-8 description still narrates the unsafe conditional

- **Severity:** nit
- **Attack vector:** 1 (R4-1 verification — adjacent contradiction risk).
- **Counterexample evidence (file:line, verified at HEAD):**
  - PLAN.md `:345` (R3-8 historical entry) reads: *"`ORDER BY created_at ASC, id ASC` (or `position ASC, ...` if a position column exists — quick schema check at build time)."*
  - PLAN.md `:351` (R4-1 entry) reads: *"Removed the `position ASC` conditional from R3-8 ... R4-1 commits to `ORDER BY created_at ASC, id ASC` exclusively."*
  - PLAN.md `:287` (canonical droplet 2.10 spec) reads: *"deterministically ordered by `created_at ASC, id ASC`"* with an explicit warning against confusing the resolver with the column-scoped `position` field.

- **Why this is a nit, not a blocker.** The canonical spec at `:287` is the contract a builder executes. The Notes Round 3 summary at `:345` is a chronological audit trail of what R3-8 looked like before R4-1 patched it. A reader following the Notes section in time order sees `:345` (R3 introduces position-conditional) → `:351` (R4 explicitly removes it). The droplet body always supersedes the Notes section.
- **Suggested mitigation.** Tighten `:345` with an inline supersession marker: *"(superseded by R4-1 — see below)"*. Optional polish, not gating.

---

### 2.2 (nit) Body regex `^\d+(\.\d+)*$` accepts leading zeros; not enumerated in the acceptance test list

- **Severity:** nit
- **Attack vector:** 5 (body regex edge cases).
- **Counterexample evidence:**
  - PLAN.md `:285` body regex `^\d+(\.\d+)*$` matches `"007"` (leading-zero forms accepted by `\d+`). The Go `strconv.Atoi("007")` returns `7, nil`, so the resolver semantically maps leading-zero forms to the same position as the bare digit form. Two distinct strings (`"007"` and `"7"`) map to the same address.
  - PLAN.md `:293` test list enumerates: valid `0`, valid `0.0`, valid `2.5.1`, slug-prefixed valid, slug-prefix mismatch, out-of-range, malformed inputs (empty, `1.`, `.1`, `1..2`, `abc`, leading-dash, deep nesting), UUID input rejected. **Leading-zero (`007`) is neither in valid list nor in malformed list** — semantics is undocumented at the spec level.

- **Why this is a nit.** The behavior is well-defined (`\d+` accepts leading zeros, `Atoi` parses them to canonical int). A test fixture `"007"` would resolve identically to `"7"` and that's the natural reading. The plan's test list could be either:
  1. Accept leading-zero as syntactically valid (test asserts `"007"` resolves identically to `"7"`).
  2. Reject leading-zero with `ErrDottedAddressInvalidSyntax` (test asserts rejection; tighten regex to `^(0|[1-9]\d*)(\.(0|[1-9]\d*))*$`).

  Builder defaults to option 1 (the regex literally allows it). Either resolution is correct; the spec is silent.
- **Suggested mitigation.** Append to PLAN.md `:293` test list: *"leading-zero forms (`007`, `0.07`) accepted as canonical aliases of `7`, `0.7`"*. Or alternately tighten regex if the dev prefers strict canonical addresses. Optional, not gating — both interpretations are sound.

---

### 2.3 (note) Snapshot version contract: 2.7's JSON-tag renames are schema-breaking but no `SnapshotVersion` bump

- **Severity:** documented in plan, downgrade-acceptable per pre-MVP rule
- **Attack vector:** 13 (snapshot version contract).
- **Counterexample evidence (file:line, verified at HEAD):**
  - `internal/app/snapshot.go:16` — `const SnapshotVersion = "tillsyn.snapshot.v5"`.
  - 2.6 explicitly states (PLAN.md `:139`): *"**No `SnapshotVersion` bump required** — field uses `omitempty` and `encoding/json` ignores unknown keys by default. Old v5 snapshots load forward-compatibly."* This is correct for 2.6 (additive `omitempty` field).
  - 2.7 (PLAN.md `:154-229`) does NOT carry an equivalent statement, but 2.7's renames ARE breaking: `ChecklistItem.Done` → `Complete` (tag `"done"` → `"complete"`), `RequireChildrenDone` → `RequireChildrenComplete` (tag `"require_children_done"` → `"require_children_complete"`), `DoneActionItems` → `CompleteActionItems` (tag `"done_tasks"` → `"complete_tasks"`), `DoneItems` → `CompleteItems` (tag `"done_items"` → `"complete_items"`), AND lifecycle-state literal rejection (`"done"` and `"progress"` no longer accepted via strict-canonical).
  - Old `v5` snapshots written before 2.7 carry these legacy keys. After 2.7 lands:
    - `"done": false` keys in `ChecklistItem` are silently dropped (Go `encoding/json` ignores unknown keys by default).
    - `"lifecycle_state": "done"` literals are explicitly REJECTED by `snapshot.go:419-421`'s switch (the error path the plan acceptance grep `lifecycle_state.*"done"` checks for).
    - `"done_tasks"` and `"done_items"` aggregate counters silently dropped.
  - Pre-MVP rule (memory `feedback_no_migration_logic_pre_mvp.md`): dev deletes `~/.tillsyn/tillsyn.db` and any persisted snapshot files between schema/state-vocab changes. PLAN.md `:228` explicitly says *"DB action: DELETE ~/.tillsyn/tillsyn.db BEFORE running mage ci for this droplet"*.

- **Why this is documented-and-accepted, not a finding.** The pre-MVP rule explicitly authorizes schema-breaking changes without version bumps because no production data exists to migrate. The DB-delete instruction in the droplet's DB action covers persisted-DB rows; persisted-snapshot rows are equally dev-deletable. A `v6` bump would be a no-op signal pre-MVP and lands as MVP-prep refinement.
- **Suggested mitigation.** Optional polish: append to droplet 2.7's acceptance criteria a one-liner: *"`SnapshotVersion` stays at `v5` — pre-MVP, no persisted snapshots survive across the rename. MVP-prep refinement drop will bump to `v6`."* Current plan's silence on this point isn't wrong (it's covered by the global pre-MVP rule), just less explicit than it could be. Not gating.

---

## 3. Re-Run of Round 3 Attack Vectors (Round 4 incremental)

| Attack | Status at Round 4 | Evidence |
| --- | --- | --- |
| **Symbol leak hunt — `Done: true|false` field literals** | clean | `git grep -n 'Done: false\|Done: true' -- '*.go'` returns sites in: `capture_test.go:124`, `kind_capability_test.go (app):429`, `service_test.go:3003, 3038, 3040, 4612`, `domain_test.go:275, 421, 422, 425, 428, 536`, `kind_capability_test.go (domain):19`. **Every site is in 2.7's enumerated Paths.** The `:19` site that R3 missed is now covered by R4-2. |
| **Symbol leak hunt — `RequireChildrenDone`** | clean | `git grep -n 'RequireChildrenDone' -- '*.go'` returns sites in: `capture_test.go:126`, `service_test.go:3042, 3095, 4613`, `action_item.go:310`, `domain_test.go:430, 566, 614`, `kind_capability_test.go (domain):35, :73`, `workitem.go:89, 380`. The `:380` site (production merge code, `RequireChildrenDone: normalizedBase.Policy.RequireChildrenDone || normalizedDefaults.Policy.RequireChildrenDone`) is NOT explicitly enumerated by PLAN.md `:160`, but the `:89` field rename forces a compile error at `:380` that the builder MUST fix to make `mage test-pkg ./internal/domain` green. Acceptance grep `git grep -nE "\\bRequireChildrenDone\\b"` returns empty post-rename → `:380` is implicitly covered. |
| **Position-column attack — `Order`/`Rank`/`Sequence` fields on `ActionItem`** | clean | `internal/domain/action_item.go:25-50` has only `Position int` at `:32`. No `Order`, `Rank`, or `Sequence` fields exist on the struct. The plan's column-scoped vs tree-stable position warning correctly identifies the only field that could cause confusion. |
| **Body regex attacks** | mostly clean (Finding 2.2) | `^\d+(\.\d+)*$` correctly rejects empty, `1.`, `.1`, `1..2`, `1.a.2`, leading-dash, non-digit. Accepts `0`, `0.0`, `2.5.1`. Leading-zero (`007`) accepted but not enumerated — see Finding 2.2 above. |
| **Slug-prefix syntax** | clean | `internal/domain/project.go:154-178 normalizeSlug` produces `[a-z0-9-]+` strings; `:` is NOT in the slug character set. `<slug>:<dotted>` is unambiguously parseable. |
| **Bare-CLI ergonomics — no default-project mechanism** | clean | `git grep -nE "DefaultProject\|default_project\|defaultProject\|TILLSYN_PROJECT\|TILL_PROJECT" -- 'cmd/till/' 'internal/config/'` returns only `cmd/till/main_test.go:358 TestRunTUIStartupDoesNotCreateDefaultProject` — the test asserts ABSENCE of a default-project mechanism. Confirms plan's *"bare dotted form without project errors"* rule. |
| **`fakeRepo` extension** | clean | `internal/app/service_test.go:18` confirms `type fakeRepo struct {...}` (bare struct, no embedding). PLAN.md `:281` correctly lists `internal/app/service_test.go (extend fakeRepo to implement the new method)` in 2.10 Paths. |
| **Mage-CI compile-overlap between 2.7 and 2.8** | clean | Both 2.7 and 2.8 touch `internal/domain/kind_capability_test.go` — 2.7 at `:19, :35, :73` (Done/RequireChildrenDone renames), 2.8 at "any test that asserted the old `[\"plan\"]`/`[\"build\"]` defaults" (different lines from 2.7's). Same-file serialization is enforced by 2.8's `Blocked by: 2.7` (PLAN.md `:252`). |
| **Cross-unit blocked-by — DAG cycle check** | clean | Edges: 2.1→ε, 2.2→ε, 2.3→2.2, 2.4→2.3, 2.5→2.4, 2.6→2.3, 2.7→{2.5, 2.6}, 2.8→2.7, 2.9→2.8, 2.10→2.9, 2.11→2.10. Topological sort: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7, 2.8, 2.9, 2.10, 2.11. No cycles. |
| **Carryover nit handling — R2-F11, R2-F7, R2-F9, R3 nit** | clean | Each is debug-output-only or covered by acceptance grep. `capture_test.go:199` format-string label `done=1` will be visually inconsistent with the now-renamed struct field at `:198` (`CompleteActionItems` after 2.7), but the test still functions — the format string is a debug failure-message label, not a struct-tag or compile-time reference. PROOF and FALSIFICATION both downgrade-accepted. |
| **JSON-tag rename completeness — `done`/`progress`** | clean | `git grep -n 'json:"(done\|progress\|done_items\|done_tasks\|require_children_done)"' -- '*.go'` returns four sites: `types.go:143 done_tasks`, `attention_capture.go:96 done_items`, `workitem.go:84 done`, `workitem.go:89 require_children_done`. **All four are in 2.7's enumerated Paths.** Post-2.7 acceptance grep at PLAN.md `:218-219` (`json:"done\|json:"progress\|...\|json:"done_tasks\|json:"done_items"`) returns empty → all four sites caught. |
| **Snapshot version contract** | documented (Finding 2.3) | `SnapshotVersion = "tillsyn.snapshot.v5"` stays at `v5` despite 2.7's breaking changes. Pre-MVP rule authorizes; could be more explicit in plan but not blocking. |

---

## 4. Verdict Summary

**PASS.** Both Round 3 blockers (R3-§2.1 unsafe `position ASC` conditional, R3-§2.2 missed `internal/domain/kind_capability_test.go:19`) are cleanly resolved by R4-1 + R4-2. R4-1's prose at PLAN.md `:287` is unambiguous, with an explicit "NOT to be confused with the existing column-scoped position field" warning that pre-empts the exact builder-conflation attack §2.1 raised. R4-2's `:163` cite is line-accurate against HEAD. No new R4 blockers. Two nits (Finding 2.1 historical-record polish, Finding 2.2 leading-zero test-list completeness) and one documented-and-accepted observation (Finding 2.3 snapshot version) — all downgrade-acceptable.

**Most positive R4 strength.** R4-1's mitigation is structural, not just textual: the prose explicitly forbids `position ASC` AND warns about the column-scoped column AND defers any future per-parent `tree_position` column to a later drop. A builder following the spec literally cannot reintroduce the bug.

**Round 3 → Round 4 progress.** 2 of 2 R3 falsification blockers resolved; PROOF Round 3's PASS verdict (which §2.3 derivative blocked) is now consistent with the actual plan state. Round 4 closes the surgical drift cleanly. **Plan is ready to enter Build phase.**

**Single-fix path (optional polish, not gating).**

- PLAN.md `:345` — append `(superseded by R4-1 — see below)` to the R3-8 historical entry's `position ASC, ...` mention. Tightens audit-trail clarity.
- PLAN.md `:293` — append `leading-zero forms (007, 0.07) accepted as canonical aliases of 7, 0.7` to the test list. Or tighten the regex if the dev prefers strict canonical addresses.
- PLAN.md `:228` — append `SnapshotVersion stays at v5 — pre-MVP, no persisted snapshots survive across the rename. MVP-prep refinement drop will bump to v6.` Makes the pre-MVP coverage explicit at the droplet level rather than relying on the global rule.

These are optional. Build can proceed against the current plan without them.

---

## 5. Verdict Block

```
Verdict: pass
Round 4 FALSIFICATION blockers (new): 0
Round 4 FALSIFICATION nits (new): 2 (Finding 2.1 Notes-section R3-8 historical record, Finding 2.2 body regex leading-zero not enumerated)
Round 4 documented-and-accepted observations: 1 (Finding 2.3 snapshot version v5 stays despite breaking 2.7 changes — pre-MVP rule)
Round 3 FALSIFICATION blockers resolved: 2 of 2 (R3-§2.1 + R3-§2.2)
Round 3 FALSIFICATION carryover nits: 4 of 4 downgrade-accepted (R2-F11, R2-F7, R2-F9, R3 nit)
PROOF Round 3's PASS verdict: now consistent with plan state after R4-1 + R4-2
```
