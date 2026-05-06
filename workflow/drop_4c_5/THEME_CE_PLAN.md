# Drop 4c.5 — Theme C + Theme E Planner Output

**Author:** Theme C+E planner subagent (filesystem-MD mode, no Tillsyn runtime).
**Status:** initial decomposition — feeds master synthesis + plan-QA twins. E.6 state: in_progress.
**Source briefs:** `REVISION_BRIEF.md` §3.3 (Theme C) + §3.5 (Theme E); memory `project_drop_3_refinements_raised.md` (R1/R2/R3/R5); `project_drop_4a_refinements_raised.md` (R4/R7/R5/R8/R9/R12); `project_drop_4b_refinements_raised.md` (R1/R2/R3/R4).
**Q3 disposition (REVISION_BRIEF §9):** correctness gaps mandatory; pure doc-only NITs INCLUDED only when they ride along on a correctness droplet (no separate doc-only droplet). C.4 is the one exception — it is intentionally a WIKI-only droplet because its scope is documentation precision.

## Droplets

### C.1 — Extend `assertOwnerStateGateUpdateFields` to Persistent / DevGated

**State:** done

**Source:** memory R1 (Drop 3 NIT 3.21).

**Files:**
- `internal/adapters/server/common/app_service_adapter_mcp.go` (function body + call site at line 845-852, doc-comment 820-829 + 1109-1121).
- `internal/adapters/server/common/app_service_adapter_steward_gate_test.go` (mirror existing Owner / DropNumber tests for Persistent / DevGated).

**Packages:** `internal/adapters/server/common`.

**Acceptance:**
1. `assertOwnerStateGateUpdateFields` signature extended to `(ctx, existing, wantOwner *string, wantDropNumber *int, wantPersistent *bool, wantDevGated *bool) error` (or equivalent struct-input form — builder picks; flag in falsification mitigations).
2. When `existing.Owner == "STEWARD"` and caller is non-steward, ANY non-nil `wantPersistent` / `wantDevGated` whose dereferenced value differs from existing rejects with `ErrAuthorizationDenied` and a sharp message naming the field.
3. Caller (`UpdateActionItem` at line 845) extends pre-fetch trigger to include `in.Persistent != nil || in.DevGated != nil`.
4. New tests: `TestAssertOwnerStateGateUpdateActionItemPersistentMutationAgentRejected` (existing.Persistent=true, agent flips to false → reject); `TestAssertOwnerStateGateUpdateActionItemDevGatedMutationAgentRejected` (parallel for DevGated); `TestAssertOwnerStateGateUpdateActionItemPersistentSameValueAgentSucceeds` (no-op write of same value → allow); steward-principal happy path on both fields.
5. `mage test-pkg ./internal/adapters/server/common` green.

**Test scenarios (table-driven where the existing Owner / DropNumber test pattern uses them):**
- agent flips Persistent true→false on STEWARD-owned: reject.
- agent flips DevGated false→true on STEWARD-owned: reject.
- agent writes same Persistent value on STEWARD-owned (idempotent): allow.
- steward-principal flips Persistent on STEWARD-owned: allow + persists.
- agent flips Persistent on non-STEWARD-owned: allow (gate bypasses non-STEWARD).
- description-only update with `in.Persistent == nil` on STEWARD-owned: allow (pre-fetch trigger respects pointer-sentinel).

**Blocked by:** none (sole touchpoint in `internal/adapters/server/common`).

**Falsification mitigations:**
- F-attack: signature change breaks call sites. Mitigation: only one call site exists (line 850); builder verifies via grep before commit.
- F-attack: pre-fetch trigger expansion adds a fetch on description-only updates that include unrelated `Persistent: nil` literals. Mitigation: trigger condition stays pointer-nil-aware (`in.Persistent != nil || in.DevGated != nil`), so nil-pointer (the dominant case) does not force the fetch.
- F-attack: builder picks struct-input form, breaking the existing Owner / DropNumber test files' direct-call shape. Mitigation: keep positional form; struct only if the builder finds a clear ergonomic gain.

---

### C.2 — Idempotency: `raiseRefinementsGateForgottenAttention` doc vs impl

**State:** in_progress (impl + test landed; verification blocked by unrelated `internal/templates/load.go` compile error in a concurrent lane — orchestrator-routed)

**Source:** memory R2 (Drop 3 NIT 3.22 1.2.1).

**Files:**
- `internal/app/auto_generate_steward.go` (function at line 364 + doc-comment lines 344-363).
- `internal/app/auto_generate_steward_test.go` (add idempotency-on-duplicate test).

**Packages:** `internal/app`.

**Acceptance:**
1. Resolve the doc/impl drift via the **add-the-lookup** path: prepend `s.repo.GetAttentionItem(ctx, attentionID)` (where `attentionID = fmt.Sprintf("refinements-gate-forgotten::%s", gate.ID)`) before constructing the new attention item. If the lookup returns a non-`ErrNotFound` hit, return nil (idempotent no-op). Rationale: the doc-comment promised idempotency since droplet 3.22; rewording to "non-idempotent" instead would weaken the safety-net contract for any future override-auth pathway.
2. Doc-comment 355-358 stays accurate: re-running gate-close yields the same id, lookup hits, helper no-ops.
3. Treat `errors.Is(err, domain.ErrNotFound)` as "no prior attention" (continue with create); other errors bubble up wrapped.
4. New test: `TestRaiseRefinementsGateForgottenAttentionIsIdempotent` — call helper twice on a gate with stragglers; assert one attention created (the second call must not call `CreateAttentionItem`). Use the existing `fakeRepo` (`internal/app/service_test.go:780`) shape.
5. `mage test-pkg ./internal/app` green.

**Test scenarios:**
- First call with stragglers → attention created.
- Second call (same gate, same stragglers) → no second attention create call observed.
- First call with no stragglers → no attention create + no `GetAttentionItem` call (preserve current early-return at line 397).
- `GetAttentionItem` returns non-`ErrNotFound` infra error → bubble up wrapped (gate-close fails closed).

**Blocked by:** none.

**Falsification mitigations:**
- F-attack: builder removes the idempotency claim from the doc-comment (the doc-only path) instead of fixing impl. Mitigation: acceptance-1 explicitly mandates the lookup path; falsification will assert presence of `GetAttentionItem` call in the code, not just the doc.
- F-attack: lookup-then-create races. Mitigation: existing storage-layer terminal-state guard at `service.go:832` already collapses the race; new lookup is a best-effort fast-path, not a critical-section. Doc-comment must call this out so a future reader doesn't assume strict atomicity.
- F-attack: `ErrNotFound` sentinel mismatch. Mitigation: builder verifies the `domain.ErrNotFound` is the correct sentinel via `internal/domain/errors.go` before committing.

---

### C.3 — Tighten `isRefinementsGate` predicate

**Source:** memory R3 (Drop 3 NIT 3.22 1.2.3).

**Files:**
- `internal/app/auto_generate_steward.go` (function at line 331 + doc-comment lines 326-330).
- `internal/app/auto_generate_steward_test.go` (negative tests for any future second STEWARD-owned numbered confluence).

**Packages:** `internal/app`.

**Acceptance:**
1. Add a `Title` shape check using a constant — likely `strings.HasPrefix(item.Title, "DROP_") && strings.Contains(item.Title, "_REFINEMENTS_GATE_BEFORE_DROP_")` — alongside the existing Owner / StructuralType / DropNumber checks. The auto-generator already controls the title shape (line 329 of doc-comment), so the predicate strengthens correctness without breaking production seeds.
2. Doc-comment 326-330 updated to reflect the title shape requirement and explain why (false-positive resilience against future STEWARD-owned numbered confluence kinds).
3. New tests: `TestIsRefinementsGateRejectsForeignSTEWARDConfluence` — STEWARD-owned + Confluence + DropNumber>0 + arbitrary title (e.g. `"DROP_5_MERGE_WINDOW_GATE"`) returns false. `TestIsRefinementsGateAcceptsCanonicalTitle` — full canonical title (matching the auto-generator's format) returns true. Cover both `DROP_4_…_DROP_5` and edge cases like `DROP_10_…_DROP_11`.
4. `mage test-pkg ./internal/app` green.

**Test scenarios:**
- canonical title `DROP_4_REFINEMENTS_GATE_BEFORE_DROP_5` → true.
- foreign STEWARD-owned numbered confluence with title `DROP_5_MERGE_WINDOW_GATE` → false.
- existing happy paths preserved (test the existing call site at `service.go:1120-1121` is still hit on real refinements gates).
- DropNumber=0 → false (existing rule preserved).

**Blocked by:** **C.2** — same package, same function file, both touch `auto_generate_steward.go`. Sequential edits to avoid merge conflicts on the same file. (Note: `internal/app` is a heavily shared package; both C.2 and C.3 also share the package with the planner's general `service.go` work — flagged as the most contended package surface in §Notes.)

**Falsification mitigations:**
- F-attack: title constant lives in two places (auto-generator's create site + new predicate) → drift risk. Mitigation: extract a shared constant or constructor (e.g. `func refinementsGateTitle(dropNumber int) string`) used by both create + predicate. Builder must touch the create site too if extraction lands.
- F-attack: builder picks the alternative ("add explicit `Kind` discriminator once kind catalog is consulted at this layer") from R3, deferring to a future drop. Mitigation: acceptance-1 mandates the title-shape path explicitly because kind catalog consultation is not on the Drop 4c.5 surface.
- F-attack: regex / prefix check accidentally matches valid future kinds. Mitigation: tests cover both positive AND adversarial titles; predicate uses prefix + contains rather than substring-only to reduce overlap surface.

---

### C.4 — WIKI Cross-Subtree Exception kind-choice survey + clarification

**Source:** memory R5 (Drop 3 NIT 3.22).

**Files:**
- `WIKI.md` (Cross-Subtree Exception section, line 250 + 257).

**Packages:** none (markdown-only).

**Acceptance:**
1. Survey STEWARD's actual rollup-kind choices used in production seeds (read `internal/app/auto_generate_steward.go` `seedStewardAnchors` and any related seed paths) and document the precedent in WIKI.md §"Drop Orch Cross-Subtree Exception."
2. Replace the current hedge ("`kind=refinement` for refinements, `kind=discussion` for discussion topics, `kind=closeout` or `kind=plan` for ledger / wiki-changelog / findings rollups as appropriate") with a precision table mapping persistent-parent → preferred child kind, with rationale for each. Example shape:
   - `REFINEMENTS` → `refinement`.
   - `DISCUSSIONS` → `discussion`.
   - `LEDGER` → `closeout`.
   - `WIKI_CHANGELOG` → `closeout`.
   - `HYLLA_FINDINGS` → `research` (read-only investigation findings) or `refinement` (carry-forward items) — clarify which.
   - `HYLLA_REFINEMENTS` → `refinement`.
3. Cross-reference the precedent in the existing CLAUDE.md "Cascade Tree Structure" section if needed (read-only check; no CLAUDE.md edit unless cross-ref drift surfaces).
4. No code changes. Self-QA pass (per memory `feedback_md_update_qa.md`) before commit.

**Test scenarios:** N/A (doc-only). Acceptance is verified via Read + grep of WIKI.md after edit.

**Blocked by:** none. Pure markdown; non-Go; no package collision.

**Falsification mitigations:**
- F-attack: survey of seed paths reveals STEWARD does NOT yet seed under all six persistent parents (only DISCUSSIONS / REFINEMENTS / HYLLA_FINDINGS land in production seeds today). Mitigation: explicitly mark unseeded parents as "TBD when STEWARD lands the seeder for this parent" rather than guessing the kind.
- F-attack: scope creep — survey reveals deeper drift in the WIKI (e.g. `kind=plan` referenced for ledger entries elsewhere). Mitigation: this droplet is scoped to §"Drop Orch Cross-Subtree Exception" only; broader drift gets logged as a Drop 4c.5 refinement memory entry, NOT folded into this droplet.
- F-attack: builder edits CLAUDE.md without checking for cross-references. Mitigation: acceptance-3 explicitly says read-only on CLAUDE.md.

---

### E.1 — Lock manager doc + test contract: input-order + duplicate-input

**State:** done

**Source:** memory R4 (Drop 4a 4a.16 F1+F2) + R7 (Drop 4a 4a.17 mirror).

**Files:**
- `internal/app/dispatcher/locks_file.go` (Acquire doc-comment lines 60-81).
- `internal/app/dispatcher/locks_file_test.go` (replace `equalStringSlices` sort-then-compare helper at line 307; add explicit input-order test + duplicate-input test).
- `internal/app/dispatcher/locks_package.go` (mirror doc-comment edit).
- `internal/app/dispatcher/locks_package_test.go` (mirror test-helper fix + new tests).

**Packages:** `internal/app/dispatcher`.

**Acceptance:**
1. `locks_file_test.go:307` `equalStringSlices` either deleted (replaced inline with `slices.Equal`) OR renamed to `equalStringSlicesSorted` so its order-blindness is named explicitly. New `equalStringSlicesInOrder` helper used for the input-order assertion.
2. New test `TestFileLockManagerAcquirePreservesInputOrder` — input `["c", "a", "b"]` against an empty manager → returned `acquired` is exactly `["c", "a", "b"]` (use `slices.Equal`, not sort-then-compare).
3. New test `TestFileLockManagerAcquireDuplicateInputIdempotent` — input `["a", "a", "b"]` from one actionItemID → returned `acquired = ["a", "a", "b"]` (or document the dedupe choice; current impl emits the duplicate twice because the same-holder branch hits twice). Doc the chosen behavior in Acquire's doc-comment.
4. Acquire doc-comment (60-81) gains a "Duplicate-input semantics" paragraph: "When `paths` contains duplicates of the same string, Acquire treats each occurrence independently. Same-holder occurrences are idempotent successes (each appears in `acquired`); the manager's internal `holders[path]` and `itemPaths[id][path]` end identical to the de-duplicated case."
5. Mirror all changes to `locks_package.go` + `locks_package_test.go` (R7's "1:1 mirror of 4a.16" claim is preserved by mirroring the fix).
6. `mage test-pkg ./internal/app/dispatcher` green.

**Test scenarios:**
- Input `["c", "a", "b"]` empty manager → `acquired = ["c", "a", "b"]` exactly (order-preserving assertion).
- Input `["a", "a", "b"]` empty manager → `acquired` length and order asserted explicitly per documented semantics.
- Input `["a"]` then second call with `["a"]` from same actionItemID → second call's `acquired = ["a"]`, no conflicts.
- Input `["a", "b"]` from item-1, then `["b", "c"]` from item-2 → item-2 gets `acquired = ["c"]`, `conflicts = {"b": "item-1"}` in input-order-preserving acquired slice.
- Same five scenarios mirrored in `locks_package_test.go`.

**Blocked by:** none. Single package, no shared file with other Theme E droplets within this package (E.2 walker.go, E.3 conflict.go, E.4 monitor.go, E.7 gate_mage_test_pkg.go are distinct files in the same package — see §Notes).

**Falsification mitigations:**
- F-attack: builder changes Acquire's behavior to dedupe duplicates (silent semantic shift). Mitigation: acceptance-3 mandates documenting the chosen semantics; tests pin behavior. If the builder prefers dedupe-on-input, that's a behavior change requiring its own droplet.
- F-attack: `slices.Equal` import drift on Go versions. Mitigation: project is Go 1.26+ per CLAUDE.md; `slices.Equal` is stable.
- F-attack: builder fixes `locks_file.go` but forgets `locks_package.go`. Mitigation: acceptance-5 mandates the mirror; falsification grep should find both.

---

### E.2 — Tree walker test rigor: archived-parent + ListColumns error path + blocker-state doc

**State:** done

**Source:** memory R5 (Drop 4a 4a.18 R1/R2/R3).

**Files:**
- `internal/app/dispatcher/walker.go` (doc-comment lines 45-75 — blocker-state phrasing).
- `internal/app/dispatcher/walker_test.go` (add archived-parent test + ListColumns-error probe).

**Packages:** `internal/app/dispatcher`.

**Acceptance:**
1. New test `TestWalkerTreatsArchivedParentAsNotEligible` — child item with `LifecycleState=Todo`, parent in `byID` with non-zero `ArchivedAt` → `isEligible` returns false. (Today the predicate at lines 167-200 doesn't check `ArchivedAt`; ListActionItems is called with `includeArchived=false` so archived parents shouldn't appear, but a defense-in-depth test pins the contract.) If the builder finds the predicate already correct via `includeArchived=false` filtering, the test asserts the filtering instead.
2. New test `TestWalkerListColumnsErrorPropagates` — stub `walkerService.ListColumns` returning an `errors.New("simulated infra failure")` → `Promote` returns error wrapping the simulated error.
3. Doc-comment update on lines 45-75: clarify that BlockedBy resolution treats missing references and non-complete blockers as "not-clear", and that this is conservative-by-design (planner-side bug surfaces as a stalled item, not a wrongly-promoted one). Drift fix only — match existing impl.
4. `mage test-pkg ./internal/app/dispatcher` green.

**Test scenarios:**
- Child with archived parent → not eligible (or not surfaced via `includeArchived=false` — pin which one).
- `ListColumns` error → `Promote` returns wrapped error (not nil, not `ErrPromotionBlocked`).
- Existing happy paths unchanged.

**Blocked by:** none within Theme C+E (different file from E.1). May share package with E.1 / E.3 / E.4 / E.7 — see §Notes for sequencing.

**Falsification mitigations:**
- F-attack: archived-parent path is gated by `ListActionItems(includeArchived=false)` upstream, making the new test unreachable. Mitigation: test stubs the service directly with a `byID` map containing an archived parent — bypasses upstream filtering and pins the predicate's defensive behavior. If the predicate doesn't actually check `ArchivedAt`, the test surfaces that gap as a behavior question routed to falsification.
- F-attack: doc-comment drift on a different concern. Mitigation: scope doc-edit narrowly to BlockedBy phrasing in lines 45-75; falsification rejects unrelated rewording.

---

### E.3 — Conflict detector: assert both file+package overlap entries + path canonicalization doc

**State:** done

**Source:** memory R8 (Drop 4a 4a.20 A14 + A6).

**Files:**
- `internal/app/dispatcher/conflict_test.go` (extend `TestDetectorFindsFileOverlapBetweenSiblings` at line 56-100 to assert BOTH overlap entries, not just the file one).
- `internal/app/dispatcher/conflict.go` (extend doc-comment lines 89-93 with path canonicalization contract).

**Packages:** `internal/app/dispatcher`.

**Acceptance:**
1. `TestDetectorFindsFileOverlapBetweenSiblings` extended: after asserting the file overlap shape (existing lines 81-99), also assert a `SiblingOverlapPackage` entry with `OverlapValue = "internal/app/dispatcher"` exists in the same `overlaps` slice. Use `len(overlaps) == 2` + a paired check.
2. `OverlapValue` doc-comment (lines 89-93) extended: "Path canonicalization is the planner's / walker's responsibility upstream — the detector does no normalization beyond the trim/dedupe `domain.NewActionItem` already applies on create. Two siblings declaring `./a/b.go` and `a/b.go` will NOT register as overlapping; the upstream caller MUST normalize before handing items to the detector."
3. `mage test-pkg ./internal/app/dispatcher` green.

**Test scenarios:**
- Two siblings declaring same path AND same package: both overlap entries surface (file + package).
- (Existing) two siblings sharing only the package: package-only entry.
- (Existing) two siblings sharing only the path: covered by current test? — verify; if not, add.

**Blocked by:** none within Theme C+E (different file from E.1, E.2, E.4, E.7).

**Falsification mitigations:**
- F-attack: `len(overlaps) == 2` rigid assertion breaks if production code starts emitting de-duped overlap entries. Mitigation: assert BOTH `SiblingOverlapFile` AND `SiblingOverlapPackage` exist in the slice via independent loops, not via length. (Mirrors the existing `for i := range overlaps` shape at line 82.)
- F-attack: A13 (concurrent `InsertRuntimeBlockedBy` single-flight) is in scope. Memory routes A13 to Drop 4b daemon-mode planning; not this droplet. Mitigation: builder reads memory and rejects out-of-scope additions.

---

### E.4 — Process monitor: `Track` doc-comment + atomicity edge case + `for-range int` modernization

**State:** done

**Source:** memory R9 (Drop 4a 4a.21 NITs).

**Files:**
- `internal/app/dispatcher/monitor.go` (doc-comment at line 227-234 — add "defer h.Close()" guidance).
- `internal/app/dispatcher/monitor_test.go` (modernize `for i := 0; i < n; i++` at lines 468 + 474).

**Packages:** `internal/app/dispatcher`.

**Acceptance:**
1. `Track` doc-comment 227-234 gains a `Cleanup contract:` paragraph: "Callers MUST `defer h.Close()` immediately after the successful `Track` return. The monitor owns the cmd's lifecycle — Close is the only safe way to release per-handle goroutines and unblock concurrent waiters. Failure to defer Close leaks a goroutine per untracked Handle."
2. Atomicity edge-case doc — `Track` doc-comment also gains: "Move-success / Update-fail atomicity: when the dispatcher subscribes to monitor results and routes through `MoveActionItem(failed)` then `UpdateActionItem(metadata.BlockedReason)`, a partial failure leaves the action item in `failed` state without `BlockedReason` populated. Drop 4b's structured-failure refactor will collapse the two writes into one transactional call; until then, dispatcher subscribers re-fetch and re-attempt the metadata write on transient errors."
3. `monitor_test.go` lines 468 + 474: `for i := 0; i < n; i++` → `for i := range n` (Go 1.26+ rangeint). `mage ci` clean (no gopls hint regression).
4. PLAN.md row 4a.21 reference (line ~300) updated to use `BlockedReason` rather than `failure_reason` if the row is still authoritative — read PLAN.md first to confirm. (Memory says "PLAN.md edit during Drop 4b" but Drop 4b shipped without doing it; verify before editing.)
5. `goleak.VerifyTestMain` and `S2` mage ergonomics doc are EXCLUDED from this droplet (out of scope per Q3 — pure tooling/test-infra changes; route to a separate hygiene droplet if needed).
6. `mage test-pkg ./internal/app/dispatcher` green.

**Test scenarios:** none new — modernization-only at the test-file layer; doc-only at the production layer. Existing tests (`TestMonitorConcurrentTrackHandlesAreIndependent`) still pass and exercise the modernized loop.

**Blocked by:** none within Theme C+E (different file from E.1, E.2, E.3, E.7).

**Falsification mitigations:**
- F-attack: `for i := range n` is Go 1.22+ but project may have older toolchain. Mitigation: CLAUDE.md pins Go 1.26+; safe.
- F-attack: PLAN.md row update touches a doc the dev considers authoritative. Mitigation: acceptance-4 requires reading PLAN.md first; if the row is already gone or already correct, skip the edit.
- F-attack: scope creep — builder addresses `goleak.VerifyTestMain` and S2 too. Mitigation: acceptance-5 explicitly excludes them.

---

### E.5 — `mapToolError` adds `ErrOrchSelfApprovalDisabled` sharp-prefix case

**State:** done

**Source:** memory R12 (Drop 4a 4a.27).

**Files:**
- `internal/adapters/server/mcpapi/handler.go` (`mapToolError` function at line 891-948).
- `internal/adapters/server/mcpapi/handler_test.go` (existing case-(e) integration test at `TestAuthRequestApproveProjectToggleDisabledRejectedIntegration` — tighten assertion to expect `auth_denied:` prefix).

**Packages:** `internal/adapters/server/mcpapi`.

**Acceptance:**
1. New `case errors.Is(err, domain.ErrOrchSelfApprovalDisabled):` branch in `mapToolError` (placed BEFORE the generic `ErrAuthorizationDenied` case so it doesn't get shadowed — note that today it isn't actually wrapped in `ErrAuthorizationDenied`, so order matters less, but defensive ordering is safer for future ledger changes). Returns `Class: "auth", Code: "auth_denied", Text: "auth_denied: orch-self-approval disabled by project toggle"` (or similar — match the existing `auth_denied:` prefix style at line 933).
2. Integration test `TestAuthRequestApproveProjectToggleDisabledRejectedIntegration` (memory R13 confirms this exists post-4a.27 round-2) tightens its assertion: instead of substring-on-text, expect mapping result with `Code: "auth_denied"` and `Text:` starting with `auth_denied:`.
3. `mage test-pkg ./internal/adapters/server/mcpapi` green.

**Test scenarios:**
- `mapToolError(domain.ErrOrchSelfApprovalDisabled)` returns `auth_denied:` prefix.
- `mapToolError(fmt.Errorf("project xyz: %w", domain.ErrOrchSelfApprovalDisabled))` (wrapped form, matching the production error site at `auth_requests.go:443`) returns `auth_denied:` prefix.
- Existing `ErrAuthorizationDenied` mapping unchanged (regression-protect).

**Blocked by:** none. Sole touch in `internal/adapters/server/mcpapi`.

**Falsification mitigations:**
- F-attack: case ordering shadows `ErrOrchSelfApprovalDisabled` if it ever wraps `ErrAuthorizationDenied`. Mitigation: place new case BEFORE the generic auth-denied case; falsification verifies via case-order grep.
- F-attack: error code drift between message text and code field. Mitigation: tests pin both `Code` and `Text` prefix.
- F-attack: existing test at `auth_requests_test.go:1407` asserts `errors.Is(err, ErrAuthorizationDenied)` is false on the toggle-disabled path. New mapping might not change that contract (the wrap doesn't add `ErrAuthorizationDenied` to the chain). Mitigation: builder verifies `auth_requests.go:443` continues to use only `%w` on `ErrOrchSelfApprovalDisabled` (no `errors.Join` with `ErrAuthorizationDenied`).

---

### E.6 — `validateMapKeys` case-fold footgun: post-decode canonicalization

**State:** done

**Source:** memory 4b R1 (Drop 4b 4b.1 F4).

**Files:**
- `internal/templates/load.go` (`validateMapKeys` at line 284-301; chosen approach changes the function to ALSO canonicalize keys post-decode, so consumer-side lookups by `domain.KindBuild` succeed when the TOML carries `[gates.BUILD]`).
- `internal/templates/load_test.go` (add case-fold acceptance + rejection tests).

**Packages:** `internal/templates`.

**Acceptance:**
1. **Chosen fix path: post-decode canonicalization.** Rationale below in §Notes. After `validateMapKeys` confirms each key is valid (existing behavior — `IsValidKind` lowercases internally), rebuild each map (`tpl.Kinds`, `tpl.AgentBindings`, `tpl.Gates`) with canonicalized lowercase keys. The function signature changes from `func validateMapKeys(tpl Template) error` to `func validateMapKeys(tpl *Template) error` so the canonicalization mutation is visible to the caller. Caller at `load.go:125` updates accordingly.
2. New test `TestValidateMapKeysCanonicalizesGatesKeys` — TOML `[gates.BUILD] = ["mage_ci"]` loads successfully AND `tpl.Gates[domain.KindBuild]` returns the gate sequence (not `tpl.Gates[Kind("BUILD")]`).
3. New test `TestValidateMapKeysCanonicalizesKindsKeys` — TOML `[kinds.BUILD]` loads + `tpl.Kinds[domain.KindBuild]` returns the entry.
4. New test `TestValidateMapKeysCanonicalizesAgentBindingsKeys` — same pattern for `[agent_bindings.BUILD]`.
5. New test `TestValidateMapKeysCollidesOnCaseFold` — TOML with BOTH `[gates.BUILD]` AND `[gates.build]` rejects with a clear error naming the collision (post-canonicalization, both keys would map to `Kind("build")`; the two distinct entries surface as a corruption).
6. `mage test-pkg ./internal/templates` green.

**Test scenarios:**
- `[gates.BUILD]` (uppercase) → loads + lookup by `domain.KindBuild` succeeds.
- `[gates.Build]` (titlecase) → same.
- `[gates.bulid]` (typo, NOT a valid kind) → still rejects via existing `IsValidKind` check.
- `[gates.BUILD]` AND `[gates.build]` in same template → rejects with collision error.
- Default template (`internal/templates/builtin/default.toml`) — regression check that existing all-lowercase keys continue to load and lookup correctly.

**Blocked by:** none within Theme C+E (different package from all other E droplets).

**Falsification mitigations:**
- F-attack: alternative fix path (exact-match validation, REJECT case-folded) is more conservative — no post-decode mutation. Mitigation: discussed in §Notes; chose canonicalization because it preserves template-author tolerance + matches `domain.IsValidKind`'s existing case-fold tolerance contract. If plan-QA prefers exact-match, this droplet flips to that path with the same test surface (collision test removed; case-fold rejection test added).
- F-attack: signature change `func validateMapKeys(tpl Template)` → `func validateMapKeys(tpl *Template)` breaks Step 4a's call ordering at `load.go:125`. Mitigation: builder updates the call site in the same droplet; falsification grep confirms only one call site.
- F-attack: collision test is brittle (TOML decoder may already error on duplicate keys). Mitigation: builder verifies pelletier/go-toml/v2 behavior on duplicate-after-canonicalization; if the decoder rejects upstream, drop the collision test or assert it surfaces from the decoder's error path.

---

### E.7 — `gate_mage_test_pkg` test rigor: no-dedup contract + halt-call-count + empty-string element

**State:** done

**Source:** memory 4b R2 (Drop 4b 4b.4 NITs 3.1/3.2/3.3).

**Files:**
- `internal/app/dispatcher/gate_mage_test_pkg_test.go` (add three new tests).

**Packages:** `internal/app/dispatcher`.

**Acceptance:**
1. New test `TestGateMageTestPkgDoesNotDedupePackages` — `Packages = ["foo", "foo"]` → runner observes 2 calls, not 1 (gate-level no-dedup contract). Failed runs halt-on-first-failure as documented (so the second call only happens if the first passes).
2. `TestGateMageTestPkgHonorsContextCancel` (existing at line 333) extended to assert `len(runner.calls) == 1` explicitly (mirrors the pattern at lines 183-184 + 219-220 in the failure tests).
3. New test `TestGateMageTestPkgRejectsEmptyStringPackage` — `Packages = ["", "pkg2"]` → gate-level behavior pinned. Either gate fails-loud naming the empty entry, or it forwards the empty pkg to the runner and the runner's failure surfaces via runErr — pin which one.
4. `mage test-pkg ./internal/app/dispatcher` green.

**Test scenarios:**
- `Packages = ["foo", "foo"]` → 2 runner calls (success path); 1 runner call + halt (first-call-fails path).
- ctx-cancel + first call records ctx.Err() → exactly 1 runner call observed.
- `Packages = ["", "pkg2"]` → behavior pinned via test expectation (likely loud failure, given gate's general fail-loud posture).

**Blocked by:** none within Theme C+E (different file from E.1, E.2, E.3, E.4).

**Falsification mitigations:**
- F-attack: empty-string-package gate behavior is undocumented; builder picks an arbitrary expectation. Mitigation: builder reads `gate_mage_test_pkg.go:108-115` — production passes the empty string straight to mage, which would error. Test pins observable behavior (gate fails with runner-error), then the doc-comment lines 22-29 gain a "Per-package empty-string handling" paragraph to make the contract explicit.
- F-attack: domain-layer empty-string normalization (per memory: "domain-layer responsibility per WAVE_A_PLAN PQA-4") preempts the gate-level test. Mitigation: gate-level test stubs the domain layer (constructs `domain.ActionItem` directly with `Packages = ["", "pkg2"]`) — bypasses the constructor's normalization, exercising gate behavior in isolation.

---

### E.8 — Auth auto-revoke: ScopeType guard + reason-string source decision

**Source:** memory 4b R3 (Drop 4b 4b.5 A2 + A14).

**Files:**
- `internal/app/auth_requests.go` (filter at line 938 — add ScopeType guard).
- `internal/app/auth_requests_test.go` (add scope-type-mismatch test).

**Packages:** `internal/app`.

**Acceptance:**
1. Filter at line 938 extended: `if path.ScopeType != domain.ScopeLevelActionItem || path.ScopeID != actionItemID { continue }`. Belt-and-suspenders defense against a UUID-collision between project-scope session ID and action-item ID (probability ~10^-37, but trivial to add).
2. Reason-string source decision: align the doc that says `terminal_state_cleanup` with the code (already says it at line 941 via `terminalStateCleanupRevokeReason`). Verify the constant lives somewhere readable; add a doc-comment near the constant explaining its grep-friendly choice + its lifecycle role. (Per memory R3 A14: builder's choice is "intentional, documented, more grep-friendly" — keep code, update doc.)
3. New test `TestRevokeActionItemAuthSessionsScopeTypeMismatchSkipped` — auth session whose `path.ScopeID == actionItemID` BUT `path.ScopeType == ScopeLevelProject` → not revoked. Pair with a happy-path test where `path.ScopeType == ScopeLevelActionItem` → revoked.
4. `mage test-pkg ./internal/app` green.

**Test scenarios:**
- ScopeType=actionItem + ScopeID matches → revoked.
- ScopeType=project + ScopeID UUID-collides with actionItemID → NOT revoked (new defensive behavior).
- ScopeType=actionItem + ScopeID does not match → NOT revoked (existing behavior preserved).

**Blocked by:** none. Same package as C.2/C.3 (`internal/app`) but different file (`auth_requests.go` vs `auto_generate_steward.go`) — see §Notes.

**Falsification mitigations:**
- F-attack: `domain.ScopeLevelActionItem` enum value drift — memory says scope vocabulary is `actionItem` (verified at `internal/domain/level.go:16`). Mitigation: builder verifies enum value before reaching for it.
- F-attack: builder picks the alternative — change the reason-string instead. Mitigation: acceptance-2 picks "keep code, update doc" explicitly. Reason: the lifecycle-embedded variant the plan-doc named is hypothetical; the grep-friendly form is already shipped.
- F-attack: scope-type guard breaks legitimate revocations under a yet-unknown path shape. Mitigation: existing tests preserve happy-path coverage; new test only adds the negative case.

---

### E.9 — Git-status pre-check ergonomic NITs: shared `filteredGitEnv` + nil-checker decision

**Source:** memory 4b R4 (Drop 4b 4b.6 NITs 4/5/7).

**Files:**
- New shared package: `internal/utils/gitenv/gitenv.go` (or `internal/platform/gitenv/`; builder picks the project's existing layered convention — there's no `internal/utils/` today per CLAUDE.md project structure, so `internal/platform/gitenv/` is the more natural home).
- `internal/app/git_status.go` (replace local `filteredGitEnv` at lines 146-156 with import from new package).
- `internal/tui/gitdiff/exec_differ_test.go` (replace local `filteredEnv` at lines 152-onwards with import from new package).
- `internal/app/service.go` (defensive nil-check at lines 1015-1019 — pick: `panic("Service.gitStatusChecker not configured")` OR document the invariant explicitly + remove the guard).

**Packages:** `internal/platform/gitenv` (new), `internal/app`, `internal/tui/gitdiff`.

**Acceptance:**
1. New package `internal/platform/gitenv` with one exported function `Filtered() []string` returning `os.Environ()` minus every `GIT_*=...` entry. Doc-comment names BOTH callers (production + test) and explains the GIT_DIR override risk that motivated the filter.
2. `internal/app/git_status.go` imports `internal/platform/gitenv` and replaces the local `filteredGitEnv()` call at line 110 with `gitenv.Filtered()`. Local function deleted.
3. `internal/tui/gitdiff/exec_differ_test.go` imports the same package and replaces local `filteredEnv()` at line 99 with `gitenv.Filtered()`. Local function deleted.
4. `service.go:1015-1019` defensive nil-check: **replace `return nil` with explicit doc-only panic-mode invariant**. Specifically: keep the nil-check + return-nil shape (it's a real codepath in tests where the seam is explicitly nil) BUT update the doc-comment lines 1016-1018 to read: "Defensive: tests may inject a nil seam to bypass the check; production wiring (NewService) always populates the default. Treat nil as 'skip' deliberately, NOT as 'should never happen'." (Memory NIT-5 framed this as "should panic or document"; documenting the test-injection use case is the precise fix.)
5. `mage test-pkg ./internal/app ./internal/tui/gitdiff ./internal/platform/gitenv` green.

**Test scenarios:**
- New `internal/platform/gitenv` package gets a unit test: env contains `GIT_DIR=/foo`, `HOME=/bar` → output drops `GIT_DIR`, retains `HOME`.
- Existing `internal/app/git_status` tests still pass after the import swap.
- Existing `internal/tui/gitdiff/exec_differ_test` tests still pass after the import swap.

**Blocked by:** none. Touches two existing packages but no shared file with other Theme E droplets.

**Falsification mitigations:**
- F-attack: `internal/utils/` vs `internal/platform/` decision is opinionated. Mitigation: CLAUDE.md "Project Structure" lists `internal/platform — OS-specific paths` — `gitenv` fits the platform-isolation theme. Builder reads the project structure section before placing.
- F-attack: pkg-rename ripples beyond this droplet. Mitigation: only two callers touch the helper; falsification grep confirms.
- F-attack: panic-mode for the nil-check is more orthodox. Mitigation: the seam IS deliberately nil-able in `Service` zero-value tests; panicking would break test harnesses. Acceptance-4 picks the document path explicitly. If plan-QA prefers panic-mode, builder routes the test-harness construction through a non-nil default first.

---

## Notes

### Cross-droplet package collision matrix

| Droplet | Package                                       | File                                                  |
| ------- | --------------------------------------------- | ----------------------------------------------------- |
| C.1     | `internal/adapters/server/common`             | `app_service_adapter_mcp.go` + `*_steward_gate_test.go` |
| C.2     | `internal/app`                                | `auto_generate_steward.go` + `*_test.go`              |
| C.3     | `internal/app`                                | `auto_generate_steward.go` + `*_test.go`              |
| C.4     | (none — markdown only)                        | `WIKI.md`                                             |
| E.1     | `internal/app/dispatcher`                     | `locks_file.go`, `locks_package.go` + tests           |
| E.2     | `internal/app/dispatcher`                     | `walker.go` + `walker_test.go`                        |
| E.3     | `internal/app/dispatcher`                     | `conflict.go` + `conflict_test.go`                    |
| E.4     | `internal/app/dispatcher`                     | `monitor.go` + `monitor_test.go`                      |
| E.5     | `internal/adapters/server/mcpapi`             | `handler.go` + `handler_test.go`                      |
| E.6     | `internal/templates`                          | `load.go` + `load_test.go`                            |
| E.7     | `internal/app/dispatcher`                     | `gate_mage_test_pkg_test.go`                          |
| E.8     | `internal/app`                                | `auth_requests.go` + `auth_requests_test.go`          |
| E.9     | `internal/platform/gitenv` (new), `internal/app`, `internal/tui/gitdiff` | `gitenv.go` (new), `git_status.go`, `service.go`, `exec_differ_test.go` |

**Same-package collisions requiring `blocked_by` wiring:**

- **C.2 ↔ C.3** — both edit `internal/app/auto_generate_steward.go`. C.3 is `blocked_by: C.2` (sequential to avoid merge conflict on the same file).
- **E.1 ↔ E.2 ↔ E.3 ↔ E.4 ↔ E.7** — all in `internal/app/dispatcher` package. They edit DIFFERENT files within that package (locks_file/locks_package, walker, conflict, monitor, gate_mage_test_pkg respectively), so per CLAUDE.md "Paths and Packages" §"File- and package-level blocking" they MUST have `blocked_by` between them at the package-lock layer, even though their `paths` don't overlap. Recommended ordering (any total order works; this minimizes test-file thrash): E.1 → E.2 → E.3 → E.4 → E.7. So: E.2 `blocked_by: E.1`, E.3 `blocked_by: E.2`, E.4 `blocked_by: E.3`, E.7 `blocked_by: E.4`.
- **C.2 ↔ C.3 ↔ E.8** — all in `internal/app` package. C.3 already `blocked_by: C.2` (file collision). E.8 `blocked_by: C.3` (package collision; same `internal/app` compile unit).
- **E.9** touches `internal/app/service.go` AND adds `internal/platform/gitenv`. `internal/app` package collision → E.9 `blocked_by: E.8`.

**Final `blocked_by` chain (within Theme C+E):**

```
C.1                         (alone in internal/adapters/server/common)
C.2  →  C.3  →  E.8  →  E.9 (chain through internal/app)
E.1  →  E.2  →  E.3  →  E.4  →  E.7 (chain through internal/app/dispatcher)
E.5                         (alone in internal/adapters/server/mcpapi)
E.6                         (alone in internal/templates)
C.4                         (markdown-only)
```

Master synthesizer should also flag any cross-theme collisions (e.g. Theme A may also touch `internal/app/service.go` → C.2 / C.3 / E.8 / E.9 may need cross-theme blockers).

### E.6 fix-path decision — post-decode canonicalization vs exact-match rejection

Two candidate paths from memory R1:

1. **Exact-match validation** (rejecting `[gates.BUILD]` at load time) — most conservative. Strictly enforces canonical-form keys in TOML. Authors who mistype get a load-time error.
2. **Post-decode canonicalization** (this droplet's chosen path) — accepts `[gates.BUILD]` and silently rewrites the map key to `Kind("build")`. Authors get tolerance; consumer-side lookups always succeed.

**Chosen: post-decode canonicalization.** Rationale:

- `domain.IsValidKind` (`internal/domain/kind.go:50-52`) ALREADY case-folds via `TrimSpace + ToLower`. The validation contract is "case-tolerant"; the storage contract should match.
- Forcing exact-match would diverge the validation surface from the value-validation surface, requiring authors to learn a second case-rule.
- Templates are author-facing config — tolerance is the right ergonomic.
- The collision case (`[gates.BUILD]` AND `[gates.build]` in same template) becomes detectable + rejectable post-canonicalization, restoring the safety property exact-match would give.

If plan-QA prefers exact-match, this droplet flips with the same surface area; only the test set changes.

### Q3 disposition (REVISION_BRIEF §9 doc-only NITs)

Memory entries mixed pure-doc NITs with correctness gaps. Per Q3 lean ("correctness mandatory; doc-only opportunistic"):

- **Folded into correctness droplets:** E.1 (lock manager — doc claim about input-order preserves IS the correctness fix; the test-helper sort-then-compare is functional drift, not pure doc); E.4 (monitor — doc-comment update PLUS test-file modernization); E.5 (mapping table extension is a sharper-prefix correctness improvement); E.8 (scope-type guard correctness + reason-string doc alignment).
- **Excluded from this drop:** `goleak.VerifyTestMain` (test-infra hygiene only); S2 mage ergonomics PLAN.md doc (process MD, route to Theme D or a follow-up if needed); A13 single-flight (Drop 4b daemon-mode planning).
- **C.4** is the one pure-doc droplet retained because §3.3 explicitly carves it out as Theme C scope; it's not opportunistic.

### Falsification-loud surfaces

Plan-QA falsification should attack these specific points:

1. **C.1** — `Persistent` is a bool; pointer-sentinel `*bool` semantics are subtle. Test the agent-flips-same-value path explicitly so falsification can't construct a "wrongly rejected description-only update" counterexample.
2. **C.3** — `isRefinementsGate` title-shape check could collide with future legitimate kinds. Falsification should propose adversarial titles and verify the predicate still rejects them.
3. **E.6** — Plan-QA may prefer exact-match over canonicalization. The droplet is structured to flip cleanly with no scope expansion.
4. **E.9** — `internal/utils/` vs `internal/platform/` placement is project-convention-driven. Confirm via CLAUDE.md "Project Structure" before commit.
5. **All E.* droplets in `internal/app/dispatcher`** — package-lock chain serializes 5 droplets. Master synthesizer should consider parallelizing them via in-progress-build branches per droplet, OR accept the serial cost (~5 sequential builds @ ~10-15 min each = ~1h).

### Out-of-scope items routed away

- **A13 conflict-detector single-flight** (memory R8) → Drop 4b daemon-mode follow-up.
- **R10 4a.22 cleanup A13/A14** — already fixed in-droplet during Drop 4a (memory marks RESOLVED).
- **R11 4a.25 multi-project toggle bypass** — deferred to multi-project subagent flows (post-Drop-5).
- **R6 4a.19 spawn.go NITs** — superseded by Drop 4c F.7 (memory marks SUPERSEDED).
- **R3 4a.15 F2/F3 doc NITs** — fixed in-droplet during 4a.15 round-2 (memory marks INLINE-FIXED).
- **`goleak.VerifyTestMain` + S2 mage doc** (memory R9) — pure tooling hygiene; route to Theme D if dev wants them in Drop 4c.5.
