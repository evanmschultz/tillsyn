# PLAN-QA-FALSIFICATION — W2 TILL_INIT (Round 1)

**Verdict:** FAIL — 3 confirmed FFs (1 cross-wave critical, 1 cross-wave critical, 1 PLAN/_BLOCKERS drift) + 7 NITs.

W2's internal serial-chain shape is sound (single-file `cmd/till/init_cmd.go`, fully serial D1→D7, no sibling-overlap-without-blocker risk, no cycles). The break-the-plan attacks land at the **cross-wave contract** with W1 and W4, plus a PLAN/_BLOCKERS drift bug.

---

## Confirmed Falsifications (FF)

### FF1 — Cross-wave contract mismatch with W4.D1: canonical group names not delivered upstream (CONFIRMED CRITICAL)

**Trace.** W2.D1 acceptance (PLAN.md line 44) renames `allowedInitGroups` to `["gen", "go", "fe"]` and PLAN.md line 41 says these are "canonical group names `go`, `fe`, `gen` confirmed by W4.D1's agent subdir restructure; D1 must not use stale `till-gen`/`till-go` names."

But W4.D1 in L1 PLAN.md lines 327-350 declares its embedded-agent **paths** as:
- `internal/templates/builtin/agents/till-go/...` (KEEPS `till-` prefix — 13 path bullets)
- `internal/templates/builtin/agents/till-gen/...` (KEEPS `till-` prefix — 6 path bullets)
- `internal/templates/builtin/agents/fe/` (NEW — NO prefix — 1 bullet)

W4.D1 acceptance line 353 explicitly says "Final `till-go/` agent set (10 files)" and line 354 says "Final `till-gen/` agent set". **W4.D1 does NOT rename `till-go/` → `go/` or `till-gen/` → `gen/`.** It only adds the new `fe/` group dir and restructures contents within the existing prefixed dirs.

Consequence at W2.D5 build time:
- W2.D5 RiskNote line 228: "The embedded template FS path is `"builtin/agents/<group>/*.md"` — unchanged from today."
- For `group="go"`, that resolves to `builtin/agents/go/*.md` — but W4.D1 leaves the actual embedded path at `builtin/agents/till-go/*.md`.
- `fs.ReadDir` in `copyAgentFiles` will return ENOENT for `builtin/agents/go/`. Every multi-group `till init` will error at the file-copy step.

This is a CROSS-WAVE CONTRACT MISMATCH. The defect lives in W4.D1 (the dir-rename it claims to confirm isn't in its acceptance bullets), but W2's plan assumes the rename and would silently break at D5 build/test time.

**ABSORPTION RECOMMENDATION.** Two viable absorptions; the W2 plan must NOT proceed without one:

1. **Patch W4.D1** to add `git mv` of `till-go/` → `go/` and `till-gen/` → `gen/` to its KindPayload. W4.D1's source-of-truth is SKETCH §2.1 line 41 ("Built-in groups: `go`, `gen`, `fe`") + §2.1 line 42 ("each lives at `agents/<group>/<name>.md`") — both explicitly use the unprefixed names. The W4.D1 plan is the defective doc, not SKETCH. This is the right fix.

2. **Absorb in W2.D1** by using `["till-gen", "till-go", "fe"]` (mixed-prefix) until a later drop normalizes. UGLY, defers the canonical-name decision, but unblocks W2 without a W4.D1 re-plan cycle.

**Recommended:** Option 1 — patch W4.D1, then dispatch W2. Block W2 plan-close on W4.D1 plan being patched first.

Cross-wave attack target: this FF lands on W4.D1's plan-QA falsification surface; W2 inherits the breakage. File the FF in W2's plan-qa-falsification AND flag for cross-wave routing to W4.D1's planner.

---

### FF2 — D7 `Metadata.KindPayload` JSON stopgap ignores W1.D2's typed `ProjectMetadata.Groups` field (CONFIRMED CRITICAL)

**Trace.** W2.D7 acceptance (PLAN.md line 298) and the matching RiskNote (line 318):
> `Metadata.KindPayload` = JSON `{"groups": ["go","fe"]}` ... `ProjectMetadata.Groups []string` does NOT exist as a typed field on `ProjectMetadata` (confirmed by reading `internal/domain/project.go`). Builder stores groups in `Metadata.KindPayload` as JSON `{"groups":[...]}`.

But W1.D2's plan in `workflow/drop_4c_6_1/DROP_4c.6.1.W1_TEMPLATE_RESOLUTION/PLAN.md` ships exactly this typed field. Quoting that plan's KindPayload (D2):

```json
{"file":"internal/domain/project.go","symbol":"ProjectMetadata.Groups","action":"modify",
 "shape_hint":"add Groups []string field with json tag 'groups,omitempty'"}
```

And L1 PLAN.md line 249 says W2 is `Blocked by: 4c.6.1.W1`. So by W2.D7 build-time, `ProjectMetadata.Groups []string` IS in the tree.

Verified against current code: `internal/domain/project.go:119` `ProjectMetadata` struct — today lacks `Groups []string`. W1.D2 adds it. W2.D7 ignores it and re-routes through `KindPayload`.

Consequences:
1. **Shipped-but-not-wired anti-pattern.** W1.D2 ships the typed field; downstream readers (`bakeProjectKindCatalog` multi-group walker per W1.D2) read from `project.Metadata.Groups`. If `till init` writes groups into `KindPayload` JSON instead, the bake walker sees empty `Groups` and the HOME-tier multi-group walk silently degrades to single-group (using `project.Language`). This recreates the Drop 3 droplet 3.20 anti-pattern (schema + resolver shipped, consumer never built).
2. **Round-trip semantics break.** `KindPayload` is `json.RawMessage` — opaque to typed Go code. Future readers of "what groups does this project have?" will have two non-equivalent sources: `Metadata.Groups` (typed) vs `Metadata.KindPayload.groups` (JSON-decoded). One of them will silently win and the other will silently lose.
3. **W2-GROUPS-R1 refinement is the W2 plan's own admission.** PLAN.md line 350 records: "Add typed `Groups []string` to `internal/domain/project.go:ProjectMetadata` and migrate from `KindPayload` JSON stopgap." But W1.D2 ALREADY ships this — the refinement is closed before it opens.

**ABSORPTION RECOMMENDATION.** Patch W2.D7 acceptance + RiskNote to:

1. Drop the RiskNote claim that `ProjectMetadata.Groups` doesn't exist.
2. Drop the `Metadata.KindPayload = {"groups":[...]}` acceptance bullet. Replace with: "`Metadata.Groups = payload.Groups` (typed field from W1.D2)."
3. Drop W2-GROUPS-R1 from the Raised Refinements table (W1.D2 closes it).
4. KindPayload change: instead of `Metadata.KindPayload = {"groups":[...]}`, write `Metadata.Groups = payload.Groups`.
5. Add to D7's `Blocked by` line: confirm W1.D2 is `complete` (already implicit via wave-level `W2 blocked_by W1`, but D7 should explicitly note typed `Groups` field as a prerequisite).

This is the same load-bearing attack axis flagged in the spawn prompt: "JSON stopgap is a planning DEFECT — D7 should specify 'use typed field.'"

---

### FF3 — PLAN.md vs `_BLOCKERS.toml` drift on W2.D1 (CONFIRMED — process violation)

**Trace.** PLAN.md (W2.D1 row) lines 21 + 41 + 65 + 335 declare:
> W2.D1 — Blocked by: 4c.6.1.W4.D1

`_BLOCKERS.toml` content (lines 6-34):
- Contains 6 `[[blockers]]` entries: W2.D2, W2.D3, W2.D4, W2.D5, W2.D6, W2.D7.
- Has NO entry for W2.D1.

Other entries DO carry external blockers (W2.D3's entry includes `"4c.6.1.W5"`, W2.D4's entry includes `"4c.6.1.W5"`). So the file convention is "external blockers ARE in _BLOCKERS.toml." W2.D1's missing entry is therefore inconsistent.

Per dispatch contract, the dispatcher reads `_BLOCKERS.toml` for blocker enforcement. A missing W2.D1 entry would cause W2.D1 to dispatch BEFORE W4.D1 completes — corrupting the cross-wave ordering. PLAN.md is documented as truth, but downstream tooling that consumes `_BLOCKERS.toml` would observe the drift.

**ABSORPTION RECOMMENDATION.** Add to `_BLOCKERS.toml`:

```toml
[[blockers]]
node = "W2.D1"
blocked_by = ["4c.6.1.W4.D1"]
reason = "W2.D1 hardcodes canonical group names confirmed by W4.D1's agent subdir rename (FF1 corollary)"
```

Note: if FF1 is fixed by Option 2 (mixing prefixes), the `blocked_by` here would shift but the entry is still required to keep PLAN.md ↔ TOML alignment.

---

## NITs (first-class fixes per `feedback_nits_are_first_class.md`)

### NIT1 — W2.D3/D4 `blocked_by W5` coarser than necessary (ABSORB inline)

W2.D3 says `blocked_by W2.D2, 4c.6.1.W5` (PLAN.md line 23, lines 122-123). But D3 only needs `picker_multi.go` (W5.D4). D3 doesn't need W5.D5 (header/footer) or W5.D6 (keybindings).

W2.D4 says `blocked_by W2.D3, 4c.6.1.W5` (PLAN.md line 24, lines 164-165). But D4 only needs `confirm.go` (W5.D2). D4 doesn't need W5.D3/D4/D5/D6.

Per W5's PLAN.md line 76-79: "W2's critical path through W5 is D2 → D3 → D4 (three serialized droplets). The parallel droplets D1 and D6 do not affect W2's readiness. Orch should be aware that W2 dispatch does not unblock the moment W5 starts — it unblocks when D4 reaches `done`."

So W5.D4 is the actual gating droplet for W2.D3. W5.D2 is the actual gating droplet for W2.D4.

Practical impact: coarse `blocked_by W5` waits for W5.D5 + W5.D6 even though they're never imported. This costs ~2-3 hours of unnecessary serialization in the cascade. Not a correctness defect, but a parallelism defect.

**ABSORB recommendation.** Tighten the `blocked_by` declarations:
- W2.D3 `blocked_by W2.D2, 4c.6.1.W5.D4` (instead of full W5).
- W2.D4 `blocked_by W2.D3, 4c.6.1.W5.D2` (instead of full W5).

Update both `_BLOCKERS.toml` entries to match. Note: pre-Drop-4b, the dispatcher's external-blocker enforcement is coarse-grained (wave-level), so this NIT is documentation-only until the dispatcher honors droplet-level external blockers. Track as future refinement if not absorbed inline.

### NIT2 — D7 `KindPayload` field in PLAN.md description carries the JSON-stopgap shape (ABSORB inline with FF2)

W2.D7's KindPayload string (PLAN.md line 327) literally encodes:
> "Metadata.KindPayload={groups:[...]}"

Per FF2, this should be `Metadata.Groups = payload.Groups`. The KindPayload spec field needs the same patch as the acceptance bullet. Same fix; called out separately so the builder doesn't miss it.

### NIT3 — D4 JSON-mode `mcp` default polarity ambiguity (ABSORB — pick *bool option)

W2.D4 RiskNote line 186: "JSON mode: `initJSONPayload.MCP` field has a Go zero-value of `false`. If a `--json` payload omits `"mcp"`, MCP defaults to `false` (not true). The REVISION_BRIEF says 'default true if absent' for JSON mode. Builder decides: either (a) use `*bool` for `MCP` and default nil→true, or (b) keep `bool` and document that JSON callers must pass `"mcp":true` explicitly."

Leaving this to "builder decides" is a plan-level ambiguity. REVISION_BRIEF §2.6 line 88 explicitly says "default true if absent" — option (b) violates the brief. Option (a) is the only spec-consistent answer.

**ABSORB recommendation.** Update D4 RiskNote + KindPayload to pin option (a): `MCP *bool` with `nil → true` defaulting. Add JSON-roundtrip test that asserts `--json '{"name":"x","groups":["go"]}'` (no `mcp` key) produces `MCP = true` behavior in the pipeline.

Cross-check: this changes the type of `initJSONPayload.MCP` — touches D1's renaming work since `initJSONPayload` is modified in D1 too. Builder must coordinate D1 + D4 (D1 lands the typed payload shape; D4 lands the MCP-pointer-bool + confirm step). Probably belongs in D1's KindPayload, not D4's, because D1 is the first droplet that touches `initJSONPayload`. Suggest moving the `MCP *bool` change to D1's acceptance and leaving D4 to handle only the TUI step + reading the field.

### NIT4 — D6 multi-group `template.toml` partial-state idempotency (ABSORB)

D6 acceptance (PLAN.md line 252): "Idempotent — skip if `<destDir>/.tillsyn/template.toml` already exists."

But a partial-state project (was previously initialized with `--group go`, now user runs `till init --group go --group fe`) has `template.toml` already existing for `go` group only. The blanket skip means `fe` group's template entries never land. User-visible bug: `till init --group go --group fe` on a previously-go-only project silently fails to integrate the `fe` template.

**ABSORB recommendation.** Either:
- Tighten acceptance to detect missing-group blocks and append them (delete the file-level skip; switch to per-group block skip).
- OR document the limitation explicitly: "Re-running `till init` with NEW groups does NOT update `template.toml`. User must `rm <project>/.tillsyn/template.toml` first" — and add a fail-loud warning if existing `template.toml` lacks selected groups.

Pre-MVP, the explicit-skip-with-warning is simpler. Builder picks. The plan must DECIDE rather than leave the gap silent.

### NIT5 — D7 `Language` mapping doesn't handle FE-priority case (ABSORB)

D7 ContextBlocks line 323: "Language mapping: `"go"` if any group is `"go"`, else `"fe"` if any group is `"fe"`, else `""`. Priority: go > fe > '' (gen). Rationale: Go backend takes primary-language precedence over FE in a multi-group project."

But if a user runs `till init --group fe --group go` (FE first, intentional), the user's selection-order intent is "fe is primary." The plan ignores selection order and applies a fixed go-priority.

For pre-MVP this is fine (FE backend support is the next-drop story). But the rationale "Go backend takes primary-language precedence over FE" is policy, not fact — it should be stated as policy explicitly.

**ABSORB recommendation.** Either:
- Use `payload.Groups[0]` as the language source (selection-order wins). Simple, respects user intent.
- OR keep the fixed priority but document it as policy in the acceptance bullet, not as rationale in ContextBlocks: "Language = `payload.Groups[0]`'s language mapping (go→go, fe→fe, gen→'')."

Selection-order is more honest. Pick option 1.

### NIT6 — `reservedInitGroups` cleanup is "remove or keep" — ambiguous (ABSORB)

D1 RiskNote line 64: "`reservedInitGroups` may be removed entirely or kept empty; builder picks the simpler option."

Leaving this to the builder is fine in principle but the doc-comment on `reservedInitGroups` (init_cmd.go lines 46-52) is load-bearing — it explains the "reserved-but-not-yet-shipped" semantic. If kept empty, the doc-comment becomes a lie (the rationale evaporates). If deleted, code is simpler.

**ABSORB recommendation.** Pin the simpler option in D1's acceptance: "Delete `reservedInitGroups` map and its doc-comment entirely. The reserved-group branch in `validateInitPayload` is also deleted. If a future group needs reservation, the validation can re-introduce it." Same line-count argument as the var deletion.

### NIT7 — D1 acceptance bullet for `initTUIGroupRows` says "all enabled" but doesn't address the `Disabled bool` field

D1 acceptance (PLAN.md line 47): "`initTUIGroupRows` updated to `["gen", "go", "fe"]` (all enabled — `till-gdd` row removed)."

But `initTUIGroupRows` is `[]initTUIGroupRow` where `initTUIGroupRow` has a `Disabled bool` field (init_cmd.go lines 126-129). If all rows are enabled, the `Disabled` field becomes dead — and so do `nextEnabledGroupRow` + `prevEnabledGroupRow` (lines 305-324, which only exist to skip disabled rows).

D3 acceptance (line 153) calls these out as deletions when picker_multi.go takes over. But D1 lands first — between D1 and D3, the `Disabled bool` field is dead but the navigation helpers still reference it.

**ABSORB recommendation.** In D1, either:
- Keep the `Disabled bool` field for the D1→D3 interim (no semantic effect since nothing is disabled).
- OR remove the field + simplify the navigation helpers in D1 (the simpler stub: cursor wraps over all rows).

D1's acceptance should pick one. The current "all enabled" framing leaves the field's fate unspecified. Recommend: keep the field for D1's stub (minimum diff), delete in D3.

---

## REFUTED Attempted Attacks (worked-through)

### REFUTED — Did D2's FLAT-detection placement survive D5's `copyAgentFiles` rewrite?

D2 RiskNote line 105: "D5 refactors `copyAgentFiles` — the FLAT check is placed in `runInitPipeline` (not inside `copyAgentFiles`) so it survives the D5 rewrite independently."

Verified `runInitPipeline` exists at `cmd/till/init_cmd.go:404`. D5 modifies `copyAgentFiles` signature and inner loop, NOT `runInitPipeline`. The pre-check in `runInitPipeline` is independent. CORRECT placement. No falsification.

### REFUTED — Does CreateProjectInput support the 4 fields D7 claims?

D7 KindPayload claims: `RepoPrimaryWorktree`, `RepoBareRoot`, `Language`, `Metadata.KindPayload`.

Verified `internal/app/service.go:311` `CreateProjectWithMetadata` accepts `CreateProjectInput` which has all 4 fields via the `domain.ProjectInput` shape (lines 212-222) + `in.Metadata`. The struct supports the claimed populations. (FF2 says one of them — KindPayload — is wrong-choice, but the struct shape itself supports it.)

### REFUTED — Narrative "7 droplets" vs D1-D7 enumeration

PLAN.md line 16 says "Seven atomic droplets in a strict serial chain."
Table at line 19-27 enumerates W2.D1 through W2.D7 (7 rows).
Section §"Blockers Reference" line 335-343 enumerates W2.D1 through W2.D7 (7 rows).
All three sources agree on 7. PLAN-QA-DISCIPLINE-R2 — clean.

### REFUTED — Could any 2 droplets merge / any 1 skip (YAGNI pressure)?

- D1 (rename) is the foundation; nothing else can land without it. Can't merge with anything.
- D2 (FLAT detection) is independent error-path code; merging with D1 would mix a rename + new-function-add — keep separate.
- D3 (picker_multi) + D4 (confirm) could conceivably merge, but they touch different W5 components (picker_multi.go vs confirm.go) with different W5 readiness gates. Splitting is correct.
- D5 (copyAgentFiles refactor) is the structural core, naturally separate.
- D6 (template.toml write) is a new function; merging with D5 would mix file-copy and template-write — keep separate.
- D7 (CreateProjectWithMetadata upgrade) is a different surface (DB record, not file copy); naturally separate.

Each droplet ships 1-3 changes per the KindPayload shape. Sizing is fine. No merge-eligible pair.

### REFUTED — Cycles in the blocker graph

Internal chain: D1 → D2 → D3 → D4 → D5 → D6 → D7. Linear, acyclic.
External: D1 ← W4.D1; D3 ← W5; D4 ← W5. No back-edges. Acyclic.

### REFUTED — Sibling file-share locks without explicit blocker

All 7 droplets share `cmd/till/init_cmd.go` + `cmd/till/init_cmd_test.go`, AND all 7 share package `cmd/till`. The fully-serial chain enforces single-writer per file/package — no overlap without blocker.

### REFUTED — Acceptance asserting behavior not in any droplet

Walked every L1-PLAN-W2 acceptance bullet (line 240-248):
- Multi-group agent subdir creation → D5 ✓
- JSON payload groups field → D1 ✓
- FLAT-layout detection → D2 ✓
- Old-schema agents.toml detection → D2 ✓
- `template.toml` written → D6 ✓
- `RepoPrimaryWorktree`, `Language`, `Metadata.groups` populated → D7 ✓ (Metadata.groups is FF2's complaint, but the L1-level claim is satisfied)
- TUI MCP prompt with default=yes → D4 ✓
- Re-run idempotency → D5 + D6 ✓
- `mage test-pkg ./cmd/till/...` + `mage ci` green → all droplets ✓

All L1 acceptance is wired to a droplet. No silently-dropped acceptance.

---

## Plan-QA-Discipline Confirmations

- **R1 (every NEW-behavior acceptance bullet has a test-runner droplet that ships it):** All 7 droplets carry `mage test-pkg ./cmd/till` + `mage ci` in their acceptance. D1's tests cover the rename. D2's tests cover FLAT + old-schema detection. D3's tests cover multi-select TUI. D4's tests cover MCP step. D5's tests cover subdir layout. D6's tests cover template.toml write. D7's tests cover project record fields. Every new behavior is testable AT the droplet that introduces it. PASS.

- **R2 (narrative count matches enumeration):** "7 droplets" matches D1-D7 enumeration in PLAN.md tables and `_BLOCKERS.toml` (sans the W2.D1 drift in FF3). PASS modulo FF3.

---

## Summary

**FF count:** 3 confirmed (FF1 cross-wave critical W4.D1, FF2 cross-wave critical W1.D2, FF3 PLAN/_BLOCKERS drift).
**Cross-wave concerns:** 2 (FF1 against W4.D1; FF2 against W1.D2).
**NIT count:** 7 (NIT1 coarser-than-needed blockers; NIT2 D7 KindPayload field; NIT3 MCP polarity; NIT4 template.toml partial-state; NIT5 Language mapping order; NIT6 reservedInitGroups cleanup; NIT7 Disabled bool dead field).

**Recommended next step:** Block W2 plan-close on:
1. W4.D1 plan being patched to include `git mv till-go → go` + `git mv till-gen → gen` (FF1 Option 1). If dev declines Option 1, fall back to FF1 Option 2 and patch W2.D1 to use `["till-gen", "till-go", "fe"]`.
2. W2.D7 acceptance + RiskNote + KindPayload patched to use typed `Metadata.Groups` (FF2).
3. `_BLOCKERS.toml` gaining the W2.D1 entry with `blocked_by = ["4c.6.1.W4.D1"]` (FF3).
4. All 7 NITs absorbed inline into their respective droplets per recommendations above.

All four are absorbable in a single planner re-pass — no need for L1 replan.
