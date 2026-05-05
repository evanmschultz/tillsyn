# Drop 4b — Plan-QA-Proof Review (Round 1)

**Mode:** filesystem-MD only.
**Reviewer role:** plan-qa-proof.
**Inputs:** `workflow/drop_4b/PLAN.md`, `WAVE_A_PLAN.md`, `WAVE_C_PLAN.md`, `REVISION_BRIEF.md`, `workflow/drop_4a/PLAN.md`.
**HEAD code state verified:** post-Drop-4a-merge (commit `2339b10`).
**Verdict:** **PASS-WITH-NIT** (one substantive code-citation correction + four minor disambiguation NITs).

---

## 1. Verdict Summary

The plan's claims are largely grounded in real code state. Every cited file exists; every cited struct, method, and field exists with the shape the plan claims; the `blocked_by` DAG is acyclic and respects same-package compile locks; L1–L7 alignment is faithful; pre-MVP rules are honored on schema-changing droplets; and Q1–Q8 are genuinely open.

**One substantive code-citation correction** (NIT-1 below): the WAVE_C_PLAN's claim that `cleanup.go:253-256` contains `revokeAuthBundleStub` is correct, **but** the plan's claim that this function's body "is intentionally empty" understates the wiring depth — `revokeAuthBundleStub` is also bound at `cleanup.go:154` inside `newCleanupHook` (the constructor's `revokeAuthBundle: revokeAuthBundleStub` line). 4b.5's builder needs to update **both** the constructor binding **and** the function body, not just delete the function. Without this clarification, a builder reading the plan literally could break the constructor wiring. Plan correctly hints at this in §2.4 acceptance #1 ("the function no longer exists"), but #2 ("`newCleanupHook` signature widens") doesn't explicitly tie the change to the line-154 binding.

**Four NITs** are minor (line-range drift, doc-comment cosmetics, wave-A misclaim of file-disjoint parallelism that 4b.4's own §"Blocked by" already self-corrects, and one disambiguation on `Service.RevokeAuthSession`'s return type).

No droplet has WRONG findings.

---

## 2. Per-Droplet Findings

### 4b.1 — `[GATES]` TABLE SCHEMA + CLOSED-ENUM GATE-KIND PRIMITIVE

| Acceptance area | Status | Evidence |
| --- | --- | --- |
| `Template` struct extension feasibility | **PROVEN** | `internal/templates/schema.go:60-110` defines `Template` with `Kinds`, `ChildRules`, `AgentBindings`, `GateRulesRaw map[string]any` (toml tag `gate_rules`, schema.go:79-85), `StewardSeeds`. Adding `Gates map[domain.Kind][]GateKind` with toml tag `gates` does NOT collide with the reserved `gate_rules` key. |
| `Load` validator slot | **PROVEN** | `internal/templates/load.go:96-107` runs four validators in order: `validateMapKeys`, `validateChildRuleKinds`, `validateChildRuleCycles`, `validateChildRuleReachability`. Plan's "after `validateChildRuleReachability`" insertion is sound. |
| Default template TOML coexistence | **PROVEN** | `internal/templates/builtin/default.toml` (full file read) contains NO existing `[gates]` or `[gates.build]` section. Adding `[gates.build] = ["mage_ci"]` is a clean addition. |
| `validateMapKeys` extension | **PROVEN** | `load.go:161-167` iterates `tpl.Kinds` and `tpl.AgentBindings` map keys; extension to also iterate `tpl.Gates` keys is mechanical. |
| Sentinel error placement | **PROVEN** | `load.go:115-151` declares the `var (...)` sentinel block; `ErrUnknownGateKind` slots in cleanly. |
| Strict-decode round-trip extension | **PROVEN** | `Template.GateRulesRaw map[string]any` already coexists with strict decoder via distinct toml key `gate_rules`. |

**Findings:** none. PROVEN.

### 4b.2 — GATE RUNNER + REGISTRY

| Acceptance area | Status | Evidence |
| --- | --- | --- |
| New file `gates.go` in `internal/app/dispatcher` | **PROVEN** | Package exists post-Drop-4a-merge; new file additions are routine. |
| `gateFunc` type + `gateRunner` struct | **PROVEN** (design-only; no committed prior art to verify against) | The plan defines the type from scratch; no current-code conflict surface. |
| `templates.GateKind` consumption | **PROVEN-CONDITIONAL** | Depends on 4b.1; `blocked_by 4b.1` is correctly declared. |
| Output capture rule (last 100 lines OR 8KB whichever shorter) | **PROVEN** (locked in REVISION_BRIEF Q7) | REVISION_BRIEF Q7 (line 85) confirms "last 100 lines OR last 8KB whichever shorter." |
| Doc-comment failure-routing contract | **PROVEN** | The contract — runner is pure executor, subscriber writes `BlockedReason` — is consistent with `domain.ActionItemMetadata.BlockedReason` (`internal/domain/workitem.go:150`). |

**Findings:** none. PROVEN.

### 4b.3 — `mage_ci` GATE IMPLEMENTATION

| Acceptance area | Status | Evidence |
| --- | --- | --- |
| Empty-`RepoPrimaryWorktree` guard mirroring `dispatcher.go:392` | **PROVEN** | `dispatcher.go:392-395` does `if strings.TrimSpace(project.RepoPrimaryWorktree) == "" { outcome.Reason = "project has empty repo_primary_worktree"; return outcome, nil }`. The mirror pattern is real. |
| `commandRunner` test-seam pattern | **PROVEN** (design choice, no prior collision) | Package-private indirection is idiomatic Go; no current-code conflict. |
| `exec.CommandContext` shape | **PROVEN** | Standard library; no Tillsyn-specific check needed. |
| `cmd.Dir = project.RepoPrimaryWorktree` | **PROVEN** | Same field used by `dispatcher.go` Stage 1 (`dispatcher.go:392`). |

**Findings:** none. PROVEN.

### 4b.4 — `mage_test_pkg` GATE IMPLEMENTATION

| Acceptance area | Status | Evidence |
| --- | --- | --- |
| `mage test-pkg <pkg>` accepts single positional arg | **PROVEN** | `magefile.go:49` declares `func TestPkg(pkg string) error` — single-arg signature confirmed via direct grep. |
| `item.Packages []string` read access | **PROVEN** | Drop 4a.6 landed `Packages` first-class on `ActionItem` (`internal/domain/action_item.go:104` per Drop 4a PLAN). |
| Per-package iteration with halt-on-first-failure | **PROVEN** (design-only, no collision) | Plan's iteration logic is sound. |
| Reuse of `commandRunner` declared in 4b.3 | **PROVEN** | Same-package symbol reuse across files is standard Go. |
| Linear sequencing 4b.3 → 4b.4 | **PROVEN** | WAVE_A_PLAN.md §"REVISED SEQUENCING" (line 269) self-corrects the earlier "parallel" claim — the plan acknowledges its own NIT and the unified `PLAN.md` row 118 already encodes the linear edge. |

**Findings:** none. PROVEN. Wave-A's earlier "parallel after 4b.2" claim (WAVE_A_PLAN.md line 51 + line 53) **is internally retracted** by the §4b.4 "REVISED SEQUENCING" paragraph at line 269 + the corrected diagram at line 290. Worth a NIT for cleanliness (see NIT-3 below) but not a blocker.

### 4b.5 — AUTH AUTO-REVOKE WIRING

| Acceptance area | Status | Evidence |
| --- | --- | --- |
| `revokeAuthBundleStub` location at `cleanup.go:253-256` | **PROVEN** | `internal/app/dispatcher/cleanup.go:253-256` defines `func revokeAuthBundleStub(_ string) error { /* Drop 4c Theme F.7 fills this in. */ return nil }` — exact match. |
| `errors.Join` aggregation at `cleanup.go:218-237` | **PROVEN** | `cleanup.go:218-237` runs the four-step pipeline (`releaseFileLocks`, `releasePackageLocks`, `revokeAuthBundle`, `unsubscribeMonitor`) and aggregates via `errors.Join(errs...)` at line 237 — exact match. |
| `Service.RevokeAuthSession(ctx, sessionID, reason)` signature | **PROVEN-WITH-NIT** | `internal/app/auth_requests.go:860` declares `func (s *Service) RevokeAuthSession(ctx context.Context, sessionID, reason string) (AuthSession, error)`. **NIT-2:** the plan's §2.4 acceptance #4 says "calls `s.RevokeAuthSession(ctx, session.SessionID, reason)`" — correct. But the plan does not name the **return type** `(AuthSession, error)` — builder may forget to discard the first return value. Minor. |
| `AuthSessionFilter` + `ListAuthSessions` + `session.ApprovedPath` API surface | **PROVEN** | `auth_requests.go:27` declares `ListAuthSessions(context.Context, AuthSessionFilter) ([]AuthSession, error)`; line 43-73 declares `AuthSessionFilter` struct including `ApprovedPath` field; line 460 already uses `s.authBackend.ListAuthSessions(ctx, AuthSessionFilter{...})` pattern; line 1004 + 1013 already use `domain.ParseAuthRequestPath` against `session.ApprovedPath`. The plan's resolution mechanism is consistent with extant code. |
| `cleanup.go:135` `newCleanupHook` signature widening | **PROVEN-WITH-NIT** | `cleanup.go:135` declares `func newCleanupHook(fileLocks *fileLockManager, pkgLocks *packageLockManager, monitor monitorUnsubscriber) (*cleanupHook, error)`. **NIT-1 (substantive):** at `cleanup.go:154` the constructor binds `revokeAuthBundle: revokeAuthBundleStub` — this is the line the builder MUST update when the stub is replaced. Plan §2.4 acceptance #1 says "function no longer exists" but doesn't explicitly cite the line-154 binding. A literal-reading builder might delete the function but leave a dangling reference. Recommend the plan call out line 154 as a load-bearing edit. |
| Lease-vs-session revocation cascade (Q1) | **OPEN-Q PROVEN** | Plan §2.4 acceptance #8 explicitly defers verification to builder ("if the autent backend keeps lease + session as separate rows, an explicit `s.authBackend.RevokeLease(...)` call lands in this droplet too"). This is a legitimate open question, not a planning gap. |

**Findings:** NIT-1 (substantive), NIT-2 (cosmetic).

### 4b.6 — GIT-STATUS PRE-CHECK ON `Service.CreateActionItem`

| Acceptance area | Status | Evidence |
| --- | --- | --- |
| `Service.CreateActionItem` insertion point between line 841 and line 907 | **PROVEN** | `internal/app/service.go:813-907` defines `CreateActionItem`. Line 841 is `return domain.ActionItem{}, err` inside the parent-lookup block; line 842 is `if err := s.enforceMutationGuardAcrossScopes(...)`. Line 907 begins `actionItem, err := domain.NewActionItem(...)`. The plan says "AFTER parent lineage validation (line 841) and BEFORE `domain.NewActionItem` (line 907)" — correct. The cleanest insertion is somewhere in the 875-906 range (after `validateKindPayload`, before `NewActionItem`). |
| `paths []string` on `CreateActionItemInput` | **PROVEN** | Drop 4a.5 landed first-class `Paths` (verified via service.go:919 — `Paths: in.Paths` in the `domain.ActionItemInput` literal). |
| `RepoPrimaryWorktree` first-class on `Project` | **PROVEN** | Drop 4a.12 landed; `dispatcher.go:392` reads `project.RepoPrimaryWorktree`. |
| Empty-paths skip rule | **PROVEN** (design-only) | Locked decision L4 + REVISION_BRIEF §3 lock. |
| Per-path invocation (path count <10) | **PROVEN** | REVISION_BRIEF Q6 locked. |
| `GitStatusChecker` interface in `internal/domain` | **PROVEN-CONDITIONAL** | New file in `internal/domain`; no current-code conflict surface. **Sub-question:** placing the helper in `internal/domain` (which historically holds pure domain entities) vs `internal/app` or `internal/adapters/git`. Plan §3.2 explicitly chose `internal/domain` "so it's reusable by future builders without a new adapter package" — debatable but documented. **Falsification sibling will likely attack this; PROOF accepts the documented choice.** |

**Findings:** none. PROVEN.

### 4b.7 — AUTO-PROMOTION SUBSCRIBER + `hylla_reingest` GATE STUB

| Acceptance area | Status | Evidence |
| --- | --- | --- |
| `dispatcher.Start(ctx)` / `Stop(ctx)` stubs at `dispatcher.go:814-822` | **PROVEN** | `dispatcher.go:814-816` is `func (d *dispatcher) Start(_ context.Context) error { return ErrNotImplemented }`. `dispatcher.go:820-822` is `func (d *dispatcher) Stop(_ context.Context) error { return ErrNotImplemented }`. Replacement is mechanical. |
| `LiveWaitEventActionItemChanged` event constant | **PROVEN** | `internal/app/live_wait.go:18-23` declares `LiveWaitEventActionItemChanged LiveWaitEventType = "action_item_changed"` with doc-comment naming Drop 4a Wave 2.2 as the authoring droplet — the cascade dispatcher is the documented consumer. |
| `subscribeBroker(ctx, projectID)` API | **PROVEN** | `internal/app/dispatcher/broker_sub.go:47` declares `func (d *dispatcher) subscribeBroker(ctx context.Context, projectID string) <-chan app.LiveWaitEvent`. Per-project goroutine model already in place. |
| `walker.EligibleForPromotion(ctx, projectID)` | **PROVEN** | `internal/app/dispatcher/walker.go:130` declares `func (w *treeWalker) EligibleForPromotion(ctx context.Context, projectID string) ([]domain.ActionItem, error)`. The signature matches the plan's "walks the project tree on every received event via `d.walker.EligibleForPromotion(ctx, projectID)`." |
| `RunOnce` callable from goroutine | **PROVEN** | `dispatcher.go:328` declares `func (d *dispatcher) RunOnce(ctx context.Context, actionItemID, projectIDOverride string) (DispatchOutcome, error)`. Note plan's §4.4 acceptance #3 says "calls `d.RunOnce(ctx, item.ID, projectID)`" — but the **third arg** in current code is `projectIDOverride string`, not a positional projectID. Builder will pass `item.ProjectID` or empty string. **NIT-4:** plan should be explicit that the third arg is the override (which can be empty for the auto-promotion path since the walker already filters by project). Minor. |
| `Service.publishActionItemChanged` helper | **PROVEN** | `internal/app/service.go:962`, `:1049`, `:1308` all call `s.publishActionItemChanged(actionItem.ProjectID)` — Drop 4a.15 landed this. The subscriber's events are published from these three paths (CreateActionItem, MoveActionItem, UpdateActionItem). |
| `Service.ListProjects` for Option B multi-project subscription | **PROVEN** | `internal/app/service.go:1386` declares `func (s *Service) ListProjects(ctx context.Context, includeArchived bool) ([]domain.Project, error)`. Option B's startup-time enumeration is feasible. |
| `cmd/till/main.go runServe` integration | **PROVEN-WITH-NIT** | `cmd/till/main.go:471-491` defines the `serveCmd` cobra command + `:2583` defines `runServe(ctx, svc, auth, appName, opts)`. Plan §4.4 acceptance #7 says "After `runServe` constructs `*app.Service`" — but **`runServe` does NOT construct `*app.Service`**; it RECEIVES one as a parameter. The actual construction site is upstream in `runFlow`. **NIT-5:** the plan's wiring point is roughly correct (extend `runServe` to construct + start the dispatcher), but the prose is misleading. The dispatcher's construction needs `svc`, `broker`, and the gate registry — `svc` is already in scope; `broker` needs to be plumbed (LiveWaitBroker is currently not visible inside `runServe`'s call site `serveCommandRunner`). Builder will need to either widen `runServe`'s signature or extract the broker from the service. |
| `dispatcher.go:79-110` `Options` struct extension | **PROVEN** | `dispatcher.go:79-84` declares `type Options struct { _ struct{} }` with a placeholder field and a doc-comment explicitly stating "Drop 4b adds gate-runner / commit-agent fields." The plan's extension to add `GateRegistry` + `ProjectIDs` is sanctioned. |
| `BlockedReason` carrier for gate-failure output | **PROVEN** | `internal/domain/workitem.go:150` declares `BlockedReason string` field on `ActionItemMetadata` — confirmed exact line + name. |

**Findings:** NIT-4, NIT-5.

---

## 3. Cross-Droplet Sequencing Verification

### 3.1 DAG Acyclicity

```
4a-merge → 4b.1 → 4b.2 → 4b.3 → 4b.4 → {4b.5, 4b.6} → 4b.7
                                                   ↑
                                                  4b.2 (separate edge)
```

Topological sort:
1. 4b.1
2. 4b.2
3. 4b.3
4. 4b.4
5. 4b.5 + 4b.6 (parallel)
6. 4b.7 (after BOTH 4b.5 AND 4b.2)

**Acyclic. PROVEN.**

### 3.2 Same-Package Compile-Lock Verification

| Package | Droplets touching | Resolution |
| --- | --- | --- |
| `internal/templates` | 4b.1 | Single, no contention. |
| `internal/app/dispatcher` | 4b.2, 4b.3, 4b.4, 4b.5, 4b.7 | 4b.2→4b.3→4b.4 linear; 4b.5 + 4b.7 share `dispatcher.go` (4b.5: `NewDispatcher` revoker; 4b.7: `Start`/`Stop` impls + `Options` extension) — plan correctly serializes 4b.7 after 4b.5 (WAVE_C_PLAN.md §5.3 + DAG). |
| `internal/app` | 4b.5, 4b.6 | Different files (`auth_requests.go` vs `service.go`) — plan parallelizes per Drop-3-3.21 textual-disjointness precedent. PROVEN consistent with prior practice. |
| `internal/domain` | 4b.6 | Single new file + sentinel-error edit. |
| `cmd/till` | 4b.7 | Single. |

**All package-level locks correctly serialized. PROVEN.**

### 3.3 Drop-4a Hard Prerequisite Coverage

Plan correctly identifies Drop 4a prereqs in REVISION_BRIEF §1 (lines 9-19) and WAVE_A_PLAN.md §"Wave depends on" + WAVE_C_PLAN.md §1.2. Specific hard-prereq citations verified:

- 4a.5 (`paths`) → consumed by 4b.6 ✓
- 4a.12 (`RepoPrimaryWorktree`) → consumed by 4b.3, 4b.4, 4b.6 ✓
- 4a.15 (`LiveWaitEventActionItemChanged` + `subscribeBroker`) → consumed by 4b.7 ✓
- 4a.18 (`walker.EligibleForPromotion`) → consumed by 4b.7 ✓
- 4a.19 (spawn stub) → indirectly consumed by 4b.7 via `RunOnce` ✓
- 4a.22 (cleanup hook + `revokeAuthBundleStub`) → consumed by 4b.5 ✓
- 4a.23 (`RunOnce` real implementation) → consumed by 4b.7 ✓
- 4a.24, 4a.26 (auth role enum + audit trail) → consumed by 4b.5 ✓

**All prereqs traceable to landed Drop-4a droplets. PROVEN.**

---

## 4. L1–L7 Alignment Verification

| Locked decision | Where honored | Status |
| --- | --- | --- |
| **L1** — Gates deterministic; no LLM in 4b | 4b.2/4b.3/4b.4/4b.7 all use `exec.Command` + log warnings; no LLM call site | **PROVEN** |
| **L2** — `[gates.<kind>]` closed-enum table, names `mage_ci`/`mage_test_pkg`/`hylla_reingest` | 4b.1 enum + validator + 4b.2 registry consumes | **PROVEN** |
| **L3** — Auth auto-revoke replaces 4a.22 stub | 4b.5 entire scope | **PROVEN** |
| **L4** — Git-status pre-check on `CreateActionItem` | 4b.6 entire scope | **PROVEN** |
| **L5** — Auto-promotion subscriber via `LiveWaitEventActionItemChanged` + `RunOnce` | 4b.7 entire scope | **PROVEN** |
| **L6** — Default `[gates.build] = ["mage_ci"]` only | 4b.1 default.toml addition | **PROVEN** |
| **L7** — Hylla reingest at `closeout`, pre-MVP fallback skip-with-warning | 4b.7 `hylla_reingest` stub logs warning + returns nil | **PROVEN** |

**All locked decisions faithfully encoded across droplets. PROVEN.**

---

## 5. Pre-MVP Rules Coverage

| Rule | Schema-changing droplet | Honored? |
| --- | --- | --- |
| Dev fresh-DBs on schema change | 4b.1 (template `Gates` field embeds in `KindCatalogJSON`) | **PROVEN** — WAVE_A_PLAN.md §"Pre-MVP DB Action" + §4b.1 "DB action" both call out `~/.tillsyn/tillsyn.db` fresh-DB. |
| Dev fresh-DBs on schema change | 4b.6 (no schema change but defense-in-depth) | **PROVEN** — WAVE_C_PLAN.md §3.7 calls out fresh-DB defensively. |
| No migration logic in Go | 4b.1, 4b.5, 4b.6, 4b.7 | **PROVEN** — every droplet's "DB action" or §"Pre-MVP Rules" entry confirms no migration code. |
| No closeout MD rollups | All droplets | **PROVEN** — REVISION_BRIEF §5 + WAVE_A_PLAN §pre-MVP + WAVE_C_PLAN §7. |
| Opus builders | All droplets | **PROVEN** — REVISION_BRIEF §5 + WAVE_C_PLAN §7. |
| Mage-only build/test | All droplets | **PROVEN** — every droplet's Verification section names mage targets. |

**All pre-MVP rules honored. PROVEN.**

---

## 6. Open Questions Q1–Q8 Verification

PLAN.md §10 lists eight open questions. Cross-checking that each is genuinely open and not silently resolved elsewhere:

- **Q1 — `GateInput.Template` YAGNI** — PLAN §10 + WAVE_A_PLAN §PQA-1: surfaced as open. **Genuinely open.** Plan-QA-falsification can attack.
- **Q2 — `commandRunner` relocation** — PLAN §10 + WAVE_A_PLAN §PQA-2: surfaced as a refactor option. **Genuinely open.**
- **Q3 — Output capture "shorter" semantic** — PLAN §10 + REVISION_BRIEF Q7 locks it as "shorter." **Resolved-but-flagged for re-attack.** Acceptable as documented.
- **Q4 — Empty `Packages` silent-success** — PLAN §10 + WAVE_A_PLAN §PQA-4 + 4b.4 §"Empty packages handling": surfaced as L4-aligned but flagged for re-attack. **Genuinely open.**
- **Q5 — Lease-vs-session revocation cascade** — PLAN §10 + WAVE_C_PLAN §8 Q1: explicitly deferred to builder verification. **Genuinely open.**
- **Q6 — Multi-project subscription Option A/B/C** — PLAN §10 + WAVE_C_PLAN §4.5 + §8 Q3: author recommends B; flagged for falsification arbitration. **Genuinely open.**
- **Q7 — Gate-registry shape contract** — PLAN §10 + WAVE_C_PLAN §8 Q5: shape owned by 4b.2; 4b.7 adapts. **Genuinely open** (cross-droplet contract.)
- **Q8 — Drop 4c F.7 spawn stub adequacy** — PLAN §10: author's stance is "stub is acceptable." **Genuinely open.**

**All eight questions are genuinely open. PROVEN.**

---

## 7. Substantive Findings (Summary)

### NIT-1 (substantive code-citation correction): 4b.5 needs to call out `cleanup.go:154` constructor binding

**Where:** WAVE_C_PLAN.md §2.4 acceptance criteria 1+2.
**Issue:** Plan says "`revokeAuthBundleStub` is **deleted**" and "`newCleanupHook` signature widens to accept an `actionItemAuthRevoker`" but does not explicitly cite line 154 (`revokeAuthBundle: revokeAuthBundleStub` inside the `newCleanupHook` constructor body) as a load-bearing edit.
**Risk:** A literal-reading builder could delete the function definition (line 253-256) without updating the constructor binding (line 154), causing compile failure.
**Suggested fix:** Add a third acceptance criterion: "**3a.** `cleanup.go:154` constructor binding `revokeAuthBundle: revokeAuthBundleStub` is replaced with `revokeAuthBundle: <new revoker closure or seam>` so the constructor produces a real-revoke-bound `cleanupHook` by default."
**Severity:** Recommend fix before builder spawn. Low actual risk because compile failure would surface immediately, but fixing it on paper is cheaper than a build-loop iteration.

### NIT-2 (cosmetic): 4b.5 should name `Service.RevokeAuthSession`'s return shape

**Where:** WAVE_C_PLAN.md §2.4 acceptance #4.
**Issue:** Plan says "Calls `s.RevokeAuthSession(ctx, session.SessionID, reason)`" but doesn't name the return type `(AuthSession, error)`. Builder may forget to discard the first return.
**Suggested fix:** Append "(returns `(AuthSession, error)`; discard the session value, propagate the error)" to the bullet.
**Severity:** Minor.

### NIT-3 (cosmetic): WAVE_A_PLAN's wave-internal sequencing diagram contradicts §4b.4 self-correction

**Where:** WAVE_A_PLAN.md lines 49-55 + line 290.
**Issue:** The `## Wave-Internal Sequencing` diagram at line 49-55 shows 4b.3 + 4b.4 as parallel branches under 4b.2; the corrected diagram at line 290 (after §4b.4's "REVISED SEQUENCING" paragraph) correctly serializes them. Both diagrams coexist in the file, with the §4b.4 paragraph explicitly retracting the earlier claim. This is **internally consistent** (the retraction is clear) but cosmetically untidy.
**Suggested fix:** Replace the line-49 diagram with the corrected line-290 version. Keep one diagram, drop the parallelism claim from line 55.
**Severity:** Cosmetic. Unified `PLAN.md` row 118 already encodes the correct linear edge — builder reads from PLAN.md, not WAVE_A_PLAN's sequencing diagram, so no real risk.

### NIT-4 (cosmetic): 4b.7 should name `RunOnce`'s third positional arg as `projectIDOverride`

**Where:** WAVE_C_PLAN.md §4.4 acceptance #3.
**Issue:** Plan says "calls `d.RunOnce(ctx, item.ID, projectID)`" — the actual signature at `dispatcher.go:328` is `RunOnce(ctx, actionItemID, projectIDOverride string)`. The third arg is the **override** semantic, not just a positional projectID. The auto-promotion path can pass empty string (since walker already filtered by project) or `item.ProjectID` (defensive belt-and-suspenders).
**Suggested fix:** Replace "`d.RunOnce(ctx, item.ID, projectID)`" with "`d.RunOnce(ctx, item.ID, \"\")`" (or `item.ProjectID` as a defensive override).
**Severity:** Minor — builder would catch on type-check.

### NIT-5 (cosmetic): 4b.7 wiring point in `runServe` is misnamed

**Where:** WAVE_C_PLAN.md §4.4 acceptance #7.
**Issue:** Plan says "After `runServe` constructs `*app.Service`" — but `runServe(ctx, svc, ...)` at `cmd/till/main.go:2583` **receives** `svc` as a parameter, not constructs it. Construction happens upstream in `runFlow`. The wiring intent is right (start the dispatcher inside `runServe` alongside the HTTP/MCP servers), but the prose is misleading.
**Suggested fix:** Replace "After `runServe` constructs `*app.Service`" with "Inside `runServe` (`cmd/till/main.go:2583`), `svc` is already in scope; extend `runServe` to construct + start the dispatcher before invoking `serveCommandRunner`. **Note:** the LiveWaitBroker is currently not visible inside `runServe`'s call site — builder will need to either widen `runServe`'s signature to plumb the broker through `runFlow`, or extract the broker from the service via a new accessor."
**Severity:** Minor — flagged here so the builder doesn't burn a round confused by the parameter list.

---

## 8. Evidence Bibliography

Every code citation in the plan, verified at HEAD:

| Citation | File:line | Status |
| --- | --- | --- |
| `Template` struct | `internal/templates/schema.go:60-110` | PROVEN |
| `Load` validator chain | `internal/templates/load.go:96-107` | PROVEN |
| `validateMapKeys` | `internal/templates/load.go:161` | PROVEN |
| Sentinel-error block | `internal/templates/load.go:115-151` | PROVEN |
| Default template `[gates]` absence | `internal/templates/builtin/default.toml` (full file) | PROVEN |
| `Options` struct | `internal/app/dispatcher/dispatcher.go:79-84` | PROVEN |
| `Dispatcher` interface | `internal/app/dispatcher/dispatcher.go:86-110` | PROVEN |
| `RunOnce` | `internal/app/dispatcher/dispatcher.go:328` | PROVEN |
| Empty-worktree guard | `internal/app/dispatcher/dispatcher.go:392` | PROVEN |
| `Start` stub | `internal/app/dispatcher/dispatcher.go:814` | PROVEN |
| `Stop` stub | `internal/app/dispatcher/dispatcher.go:820` | PROVEN |
| `cleanupHook` struct | `internal/app/dispatcher/cleanup.go:86-116` | PROVEN |
| `newCleanupHook` constructor | `internal/app/dispatcher/cleanup.go:135` | PROVEN |
| `revokeAuthBundle: revokeAuthBundleStub` binding | `internal/app/dispatcher/cleanup.go:154` | PROVEN (NIT-1 cite) |
| `OnTerminalState` aggregation | `internal/app/dispatcher/cleanup.go:218-237` | PROVEN |
| `revokeAuthBundleStub` body | `internal/app/dispatcher/cleanup.go:253-256` | PROVEN |
| `Service.RevokeAuthSession` | `internal/app/auth_requests.go:860` | PROVEN |
| `ListAuthSessions` interface method | `internal/app/auth_requests.go:27` | PROVEN |
| `AuthSessionFilter` struct | `internal/app/auth_requests.go:43-73` | PROVEN |
| `domain.ParseAuthRequestPath` usage | `internal/app/auth_requests.go:1004, 1013` | PROVEN |
| `Service.CreateActionItem` | `internal/app/service.go:813-907` | PROVEN |
| `Service.publishActionItemChanged` callsites | `internal/app/service.go:962, 1049, 1308` | PROVEN |
| `Service.ListProjects` | `internal/app/service.go:1386` | PROVEN |
| `LiveWaitEventActionItemChanged` constant | `internal/app/live_wait.go:18-23` | PROVEN |
| `dispatcher.subscribeBroker` | `internal/app/dispatcher/broker_sub.go:47` | PROVEN |
| `treeWalker.EligibleForPromotion` | `internal/app/dispatcher/walker.go:130` | PROVEN |
| `BlockedReason` field | `internal/domain/workitem.go:150` | PROVEN |
| `serveCmd` cobra registration | `cmd/till/main.go:471-491` | PROVEN |
| `runServe` | `cmd/till/main.go:2583` | PROVEN |
| `magefile.go TestPkg` | `magefile.go:49` | PROVEN |

Zero WRONG citations. Zero missing surfaces. Five NITs (one substantive, four cosmetic).

---

## 9. Verdict

**PASS-WITH-NIT.**

The plan is grounded in real code state, the DAG is acyclic, same-package locks are correctly serialized, L1–L7 are faithfully encoded, pre-MVP rules are honored, and Q1–Q8 are genuinely open. The five NITs above are all small enough that builders can be trusted to recover from them, but **NIT-1 specifically** should be patched into WAVE_C_PLAN.md before 4b.5's builder spawn so the constructor-binding edit at `cleanup.go:154` is explicit. NITs 2–5 can be patched optimistically or at the orchestrator's discretion.

No need to re-spawn the planner. Patches are mechanical edits to the wave plans. PROOF gate clears for the unified PLAN's structural correctness.

---

## 10. Hylla Feedback

N/A — review touched non-Go files (markdown plans) and Go-source READS only. No Hylla calls per spawn directive (Hylla stale across post-Drop-4a-merge code; gate framework is new code with no committed surface). Used `Read` + `Bash rg` per CLAUDE.md non-Go fallback rule.
