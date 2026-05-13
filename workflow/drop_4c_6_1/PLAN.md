# DROP_4c.6.1 — USER_SURFACE_MULTI_GROUP_FE_BOOTSTRAP

**State:** planning
**Round:** 9
**Blocked by:** none (DROP_4c.6 closed, HEAD `e321d3f` on origin/main)
**Blocks:** DROP_4c.7 (cascade wiring needs this drop's user surface to be dogfood-friendly first)
**Paths (expected):** See per-wave rows below.
**Packages (expected):** `internal/config`, `internal/app`, `internal/app/dispatcher/cli_claude/render`, `internal/templates`, `internal/adapters/mcp_stdio` (NEW), `internal/adapters/mcp_common` (NEW), `internal/adapters/mcp_rpc` (NEW — receives `mcpapi/` per W7 inverted carving), `cmd/till`, `internal/tui/components` (NEW), `internal/tui/style` (NEW), `internal/tui/keybindings` (NEW), `fe/` (NEW — separate go.mod per W6 L2 decision), `CLAUDE.md` (doc-only)
**Workflow:** `workflow/example/drops/WORKFLOW.md`
**Started:** 2026-05-12

## Round 9 Changes

Round 8 plan-QA falsification PASS-WITH-ABSORB: one CONFIRMED single-line grammar defect (R8-FF1 at locked-decisions section) + three NITs DEFERRED-AS-NIT with explicit reasons + one REFUTED self-flagged concern. Round 9 is minimal:

- **R8-FF1**: ABSORB — locked-decisions line ~874 corrected from "~22 prompt files at .tillsyn/agents/{go,fe}/" (grammatically wrong; 22 = total droplets) to "20 prompt files at .tillsyn/agents/{go,fe}/ (10 per group) + .tillsyn/bindings.json + .gitignore re-includes (W8)."
- **R8-NIT1**: DEFERRED-AS-NIT — reason: R2 numeric sub-clause visibility is incremental methodology polish; not load-bearing for L2 spawn this drop. Future round may promote to separate refinement row.
- **R8-NIT2**: DEFERRED-AS-NIT — reason: Round 7 Changes retroactive D19→D21 substitution is in historically-consistent direction; reverting creates transient inconsistency for purity-only reasons.
- **R8-NIT3**: DEFERRED-AS-NIT — reason: line ~73 (Round 3 Changes block) has the same "~22 prompts at .tillsyn/agents/{go,fe}/" grammar as R8-FF1, but is historical Round 3 narrative; preservation discipline applies (per round-2 spawn-brief precedent for Round 6 line 34).
- **R8-FF2** (self-flagged): REFUTED — D8a/D8b/D8c prose vs D8/D9/D10 sequential D-list is cosmetic shorthand; L2 D-list is authoritative.

## Round 8 Changes

Round 7 plan-QA falsification surfaced R7-FF1: narrative claims of "19 prompt-authoring droplets" inconsistent with L2 spawn directive's enumerated D-list (16 batched / 18 literal). Round 8 surgically reconciles via option α (un-batch D8 + D16):

- **R7-FF1**: ABSORB — un-batched D8 (closeout/commit-message/orchestrator-managed → D8a/D8b/D8c) and D16 (same for fe). Total = 20 prompt-authoring droplets (10 × 2 groups). Smoke renamed D19 → D21. Six narrative occurrences updated. R7-NIT1 (D-range inconsistency) folded in.
- Extended **PLAN-QA-DISCIPLINE-R2** with sub-clause: narrative counts must match L2 spawn directive's enumerated D-list (captures the R7-FF1 failure pattern).

Pattern observation: PLAN-QA-DISCIPLINE-R2 was added in round 7 but couldn't self-protect round 7's own absorption (discipline-added-in-round-N applies starting round-N+1). R8's plan-QA falsification should fully apply R2 to round 7's absorption + round 8's surgical edits.

## Round 7 Changes

Round 6 plan-QA falsification surfaced R6-FF1 (adjacent L1 structural claims not swept post-R5-FF1 absorption) + R6-FF2 (smoke test structurally infeasible to host in "LAST prompt droplet" — needs dedicated droplet) + R6-NIT1 (cosmetic table format). Round 7 surgically fixes:

- **R6-FF1**: ABSORB — swept PLAN.md lines 122/793/804 to acknowledge W8 is now a DUAL-WAVE sub-plan (20 prompt droplets Wave A; 1 dedicated smoke-test droplet D21 Wave C transitively, blocked by W1).
- **R6-FF2**: ABSORB — rewrote W8 L2 spawn directive cross-wave note: smoke test is a DEDICATED L2 droplet D19 (paths/packages distinct from prompt-authoring droplets; cannot live in "LAST prompt droplet" per atomic-droplet sizing + path/package lock semantics).
- **R6-NIT1**: ABSORB — fixed PLAN-QA-DISCIPLINE-R1 refinements-table row from 3 cells to 2 cells to match 2-column table schema.
- Added **PLAN-QA-DISCIPLINE-R2** methodology refinement: for every surgical cross-wave or cross-droplet absorption, sweep all L1 structural claims (wave roster, parallelism notes, decomposition-shape table, dependency graph) to verify they still hold post-absorption (captured from R6-FF1 pattern).

## Round 6 Changes

Round 5 plan-QA falsification surfaced R5-FF1 (CRITICAL): W8 smoke-test acceptance bullet (added in R5 to absorb R3-NIT4) exercises subdir-per-group resolver behavior that W1 ships in Wave B; W8 is Wave A with no blocked_by W1. Round 6 surgically fixes:

- **R5-FF1**: ABSORB — W8 smoke-test droplet declares blocked_by 4c.6.1.W1 explicitly. Other 19 W8 prompt droplets stay Wave A unblocked. (See W8 L2 spawn directive cross-wave dependency note.)
- **R5-NIT1**: DEFERRED-AS-NIT — reason: paraphrase substance accurate (low-fidelity but not fabricated); fixing risks more drift than benefit.
- **R5-NIT2**: DEFERRED-AS-NIT — reason: stylistic editorial note; non-blocking; doesn't change builder behavior.

Pattern observation worth capturing for future plan-QA falsification: when an acceptance bullet exercises NEW behavior shipped by ANOTHER wave, the testing droplet MUST `blocked_by` that wave. Future plan-QA falsification should attack this surface explicitly: "for every acceptance bullet that asserts NEW behavior, is the wave that ships that behavior in this droplet's blocked_by?"

## Round 5 Changes

Round 4 absorbed 7 of 9 R3 findings cleanly but left R3-NIT2/NIT3/NIT4 with non-explicit dispositions ("accepted as-is" / "no change"). Per dev directive (memory `feedback_nits_are_first_class.md`), NITs are first-class. Round 5 surgically dispositions the remaining three:

- R3-NIT2: MOOT — W7 fully restructured to 4 droplets; original concern about W7.D1 atomicity no longer applies. The new W7.D1 is pure-read INVENTORY (no extraction), so the NIT2 split suggestion is void.
- R3-NIT3: DEFERRED-AS-NIT — flat-19 W8 shape defensible per falsifier ("either shape is defensible," severity: low); orchestrator preserves it; L2 may split internally into W8.go + W8.fe second-level sub-plans if needed. Tracked as W8-DECOMP-R1.
- R3-NIT4: ABSORB — added W8 acceptance bullet for one-prompt integration smoke through 3-tier resolver. W8 L2 spawn directive updated to require it in the final prompt droplet. Tracked as W8-SMOKE-R1 for full end-to-end deferral.

Process change: future plan-QA + build-QA rounds enumerate every finding (FF AND NIT) as ABSORB or DEFERRED-AS-NIT-with-reason. No "judgment call" / "as-is" / "accepted" language without explicit absorb/defer disposition + reason.

## Round 4 Changes

Round 3 plan-QA: proof PASS, falsification FAIL on 2 FFs (R3-FF1 mcpapi/ extraction miss; R3-FF2 stil baseline.json already has product_extensions.tillsyn) + 8 NITs. Dev disposed 2026-05-12 with inverted W7 carving + bindings merge semantic. This round 4 absorbs:

- R3-FF1: W7 restructured to 4 droplets — D1=Inventory (NEW, pure-read, produces W7_INVENTORY.md), D2=Extract-everything-not-HTTP (per inventory), D3=Delete-residue (belt-and-suspenders), D4=CLAUDE.md (renumbered from old W7.D3)
- R3-FF2: bindings merge semantic = ID-based deep merge; 9-command palette deduped (stil baseline 4: `new-drop`, `complete-drop`, `handoff`, `comment`; Tillsyn-local additions 5: `dispatch`, `plan`, `archive`, `settings`, `help`; original `close` dropped as redundant with stil's `complete-drop`)
- R3-NIT5: W8 L2 directive — 6 prompts have no `~/.claude/agents/` source (closeout, commit-message, orchestrator-managed × 2 groups) flagged as AUTHORED FROM SCRATCH; plan-qa vs build-qa differentiation guidance added
- R3-NIT7: stil tokens path → `src/styles/tokens.css` (not `dist/tokens.css`); updated in W6 scope + L2 directive
- R3-NIT8: W8 paths → working-dir-relative (`.tillsyn/...` not `tillsyn/main/.tillsyn/...`)
- R3-NIT1 (proof): W7.D1 inventory instruction covers all consumers including previously-missed :2653; inventory exhaustiveness is the acceptance gate
- R3-NIT1 (falsification): absorbed by inverted carving — W7.D1 INVENTORY explicitly enumerates ALL consumers via `git grep -n` + LSP findReferences
- R3-NIT2: W7.D1a/D1b split moot — W7 entirely restructured to 4 new droplets
- R3-NIT3: W8 size unchanged — defensible; sub-planner may split at L2
- R3-NIT4: W8 integration smoke deferred to 4c.7 — unchanged
- R3-NIT6: W3 L2 directive updated with `--force` overwrites-customization warning
- Wave graph: W7.D1 (inventory) Wave A; W7.D2 (extract) Wave B; W7.D3 (delete-residue) Wave C + blocked_by W2 for cmd/till package lock; W7.D4 (CLAUDE.md) Wave B. L1 node count: 6 sub-plan containers + 7 direct droplets = 13 L1 nodes.
- KEYBIND-R3 wording: reworded from "move" to "canonicalize additive" — Tillsyn's 5 local commands join stil's existing 4 in a future stil-side drop (not a replacement/move).

## Round 3 Changes

Round 2 plan-QA returned proof PASS (2 cosmetic NITs) + falsification FAIL (1 critical R2-FF1 + 3 NITs). Dev added significant scope (vim keybindings + W8 Tillsyn-project-local prompts + `till agents bootstrap` CLI). All dispositions resolved 2026-05-12; this round 3 absorbs them:

- R2-FF1 (W7.D1 expansion to also extract `common/` → `mcp_common/`): W7.D1 now extracts BOTH `RunStdio → mcp_stdio/` AND `common/ → mcp_common/`. All importers updated: `main.go:81-82`, `:2682`, `:2763-2764`, 12+ test sites in `main_test.go`. W7.D2 deletion target narrowed to the true residue: `httpapi/` + HTTP-specific bits of `server.go` only.
- W8 NEW sub-plan: TILLSYN_PROJECT_AGENT_PROMPTS — ~22 prompts at `tillsyn/main/.tillsyn/agents/{go,fe}/` + `.tillsyn/bindings.json` + `.gitignore` re-includes. Wave A entry, disjoint from all other paths.
- W3 expanded: + `till agents bootstrap` CLI (§2.17) alongside existing `save/list/show/diff`.
- W5 expanded: + vim keybinding dispatcher (`internal/tui/keybindings/` — 4 files, new package, migration target lykta) per §2.14.
- W6 expanded: + vim engine + wails-keys + palette (`fe/frontend/src/lib/vim/` — 4 TS files + Vitest + Playwright) per §2.15; default Wails native menu explicit in `fe/main.go`.
- All R2 NITs absorbed: `--groups` plural drift fixed in acceptance map (5.1/5.2 → `--group go` / `--group go --group fe`); ORCH-MANAGED-R1 added to refinements table; `mage ci-fe` decision made explicit ("target added in W6; scope L2-decided"); Playwright added to W6 L2 directive; CONSUMER-TIE TEST CONTRACT added to W2 L2 directive.
- New refinements added: BOOTSTRAP-R1, KEYBIND-R1, KEYBIND-R2, KEYBIND-R3, BIND-CONSIST-R1, NATIVE-MENU-R1, QA-SPLIT-R1, EMBED-PROMPTS-R1, CASCADE-WIRING-R1.
- L1 shape updated: 6 sub-plan containers (W1, W2, W3, W5, W6, W8) + 6 direct droplets (W0, W4.D1, W4.D2, W7.D1, W7.D2, W7.D3) = 12 L1 nodes.

## Round 2 Changes

Round 1 plan-QA pair (proof + falsification) returned FAIL with 5 critical/high findings + 11 NITs. Dev resolved dispositions on 2026-05-12; this round 2 absorbs them:

- FF1 (till serve / till mcp split): new W7.D1 = Step A (refactor `RunStdio` + helpers to `internal/adapters/mcp_stdio/`); new W7.D2 = Step B (delete HTTP server bits); old W7.D2 (CLAUDE.md) renumbered to W7.D3. W7.D2 blocked_by W7.D1.
- FF2 (FLAT migration): no migration code; builder fails loud with clear error + remediation instructions when FLAT layout detected.
- FF3 (orchestrator-managed.md): kept as 10th file per group (9 standard + 1 special). W4.D1 updated: do NOT delete from till-gen; ADD to till-go (if absent) and fe group.
- FF4 (--structural-type): smart-default per kind (plan/refinement→segment; all other 10 kinds→droplet) + optional override flag. W3 `till action_item create` flag surface updated.
- Proof-FF1 (config decoder): new **W0 — CONFIG_DECODER_MULTI_GROUP** direct droplet (Wave A head). Covers `internal/config/agents.go` struct + Resolve + Merge rewrite + golden tests. W0 added to blocked_by of W4.D2.
- All NITs absorbed inline: sub-plan count fixed (5 not 4); W4.D2 Packages collapsed; 5.13 deferred to 4c.7 explicitly; W4.D1 atomicity accepted with note; `## Hylla Feedback` removed from PLAN.md (belongs in closing comment); `-agent` suffix used consistently; `--group` (singular, repeated) used consistently; agents.toml re-init behavior noted in W2 L2 directive; `~/.claude/agents/` split added to Out-of-scope; `mage ci-fe` target decision noted in W6; `## Planner` heading renamed to `## Per-Wave Plans`.

## Scope

User-surface completion drop. Drop 4c.6 shipped the architectural slab (3-tier agent body resolver, agents.toml schema, embedded scaffolding, till init). Drop 4c.6.1 closes the gaps between "architecturally sound" and "dogfood-ready":

- `internal/config/agents.go` decoder + deep-merge updated for the new `[<group>]` / `[<group>.<kind>]` multi-group schema.
- `till init` becomes multi-group, populates all project record fields, writes `template.toml`, confirms MCP registration. Fails loud on FLAT layout detection.
- User HOME tier added to the bake walker (3-tier → 4-tier template resolution).
- Group-aware agent body resolver updated to subdir-per-group for project tier.
- Full CLI surface for template/agents/project/action_item lifecycle management.
- Agent file set restructured: 10 agents per group (9 standard + `orchestrator-managed.md` special), `go-` orphans deleted, `fe/` group added, schema shifted to `[<group>.<kind>]`.
- TUI components + style system (inline, with `// MIGRATION TARGET: github.com/hylla-org/lykta` markers).
- FE scaffold: Wails v2 + Astro + SolidJS + stil tokens (inline in `fe/`).
- `till serve` deleted entirely (two-step: stdio MCP extracted first, then HTTP server bits deleted).
- CLAUDE.md cascade table corrected.

Source-of-truth scope: `workflow/drop_4c_6_1/REVISION_BRIEF.md` §2 (subsections 2.1–2.20 + 2.12a) + `SKETCH.md` §§1–10.
Locked architectural decisions: `SKETCH.md` §10 (multi-group composable, drop `go-` prefix, 10 files per group, 4 QA files per group, Wails + Astro + Solid + stil tokens from `src/styles/tokens.css`, no till serve, no go.work, `[<group>.<kind>]` schema, methodology docs deferred to 4c.8, FF1/FF2/FF3/FF4 dispositions + R3-FF1 inverted-carving W7 4-droplet sequence + R3-FF2 ID-based deep-merge bindings, vim keybinding dispatcher in W5 + vim engine in W6, W8 Tillsyn-project-local prompts, `till agents bootstrap` CLI in W3, Tillsyn-local bindings.json + .gitignore re-includes in W8).

## Per-Wave Source-of-Truth

REVISION_BRIEF.md §2 subsections are the canonical scope for each wave. SKETCH.md §§1–10 carry architectural decisions. Builders + plan-QA read both files for context, narrow to the per-wave and per-droplet scope declared here.

- W0 → REVISION_BRIEF §2.12a (config decoder multi-group rewrite)
- W1 → REVISION_BRIEF §2.1 (bake walker HOME tier) + §2.2 (group-aware resolver)
- W2 → REVISION_BRIEF §2.3–2.6 (multi-group init + project record + TUI MCP confirm + template.toml write)
- W3 → REVISION_BRIEF §2.7–2.10 + §2.17 (template/agents/project/action_item CLIs + `till agents bootstrap`)
- W4 → REVISION_BRIEF §2.11–2.12 (agent set restructure + schema shift)
- W5 → REVISION_BRIEF §2.14 (TUI components + style system + vim keybinding dispatcher)
- W6 → REVISION_BRIEF §2.15 (FE scaffold + vim engine + wails-keys + palette)
- W7 → REVISION_BRIEF §2.13 + §2.16 (INVERTED CARVING: inventory → extract-everything-not-HTTP → delete-residue → CLAUDE.md update; 4-droplet sequence)
- W8 → REVISION_BRIEF §2.18 + §2.19 + §2.20 (Tillsyn-project-local agent prompts + bindings.json + .gitignore)

## Per-Wave Plans

### Decomposition Shape — L1 Mix of Sub-Plan Containers and Direct Droplets

Per `~/.claude/agents/go-planning-agent.md` § "Multi-level decomposition," this PLAN.md is the L1 plan only. Waves whose work exceeds the atomic-droplet sizing budget on first inspection emit a `kind=plan` sub-plan container; the orchestrator spawns a sub-planner agent against each sub-plan when its `blocked_by` clears. Small independent changes emit `kind=build` droplet rows directly.

| Wave    | L1 Shape               | Reason                                                                                                                                                   |
|---------|------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------|
| W0      | direct droplet         | `internal/config/agents.go` struct + Resolve + Merge rewrite + golden tests; ~3-5 functions, 1-2 files; fits atomic sizing.                             |
| W1      | sub-plan container     | HOME-tier bake walker + group-aware resolver update together span `internal/app/service.go` + `internal/app/dispatcher/cli_claude/render/render.go`; multi-droplet. |
| W2      | sub-plan container     | Multi-group flag + groups-plural payload + subdir-per-group copy + `template.toml` write + `CreateProjectWithMetadata` field population + TUI MCP confirm + FLAT detection + re-run safety tests; ~6-8 droplets. |
| W3      | sub-plan container     | 15 CLI subcommands + `till agents bootstrap` CLI (§2.17) across new + modified files in `cmd/till/`; clearly exceeds atomic sizing.                      |
| W4      | direct droplets (2)    | W4.D1: structural file changes only (agent dirs + embed.go — no semantic Go logic); W4.D2: TOML content updates (non-Go files + embed.go). Each fits one atomic droplet. |
| W5      | sub-plan container     | 7 component files + 3 style files + vim keybinding dispatcher (4 files, new package `internal/tui/keybindings/`); each unit is 1-4 code blocks but together exceed 120 LOC. |
| W6      | sub-plan container     | Wails setup + Astro config + 6 FE pages + Go bindings + stil token integration + vim engine (4 TS files + tests); clearly multi-droplet.                 |
| W7      | direct droplets (4)    | W7.D1: INVENTORY — read every file in `internal/adapters/server/`, classify each as http-residue/stdio-relevant/transport-neutral, produce `workflow/drop_4c_6_1/W7_INVENTORY.md` with consumer map; no code changes. W7.D2: EXTRACT EVERYTHING-NOT-HTTP per inventory (new `mcp_stdio/`, `mcp_common/`, `mcp_rpc/` packages + all importer updates). W7.D3: DELETE HTTP RESIDUE per inventory + run `mage ci` as belt-and-suspenders. W7.D4: CLAUDE.md cascade table corrections (doc-only, renumbered from old W7.D3). |
| W8      | sub-plan container     | ~22 build droplets: 20 prompt-authoring droplets (Wave A) + `.tillsyn/bindings.json` + `.gitignore` re-includes (Wave A, D0) + 1 dedicated smoke-test droplet (Wave C, `blocked_by W1`); prompt files are separate atomic droplets. DUAL-WAVE sub-plan: prompt-authoring droplets touch only `.tillsyn/` files; smoke-test droplet (D21) touches `internal/app/dispatcher/cli_claude/render/render_test.go`. |

---

### Wave W0 — Config Decoder Multi-Group

#### 4c.6.1.W0 — Update `internal/config/agents.go` for multi-group schema

- **State:** todo
- **Kind:** `build` (atomic droplet; `Irreducible: true`)
- **Paths:**
  - `internal/config/agents.go` (MODIFY — struct definitions + Resolve + Merge)
  - `internal/config/agents_test.go` (MODIFY — golden fixture tests for multi-group merge cases)
  - `internal/config/testdata/` (ADD golden fixture TOML files for merge cases, if testdata dir exists; new dir otherwise)
- **Packages:** `internal/config`
- **Acceptance:**
  - `AgentsRegistry` struct (or equivalent top-level type) supports `map[group]GroupConfig` where `GroupConfig` has a default entry + per-kind override entries. The `[<group>]` and `[<group>.<kind>]` TOML sections decode cleanly into this shape.
  - `Resolve(registry, group, kind string)` returns the resolved `Preset` for the given group + kind, applying group-level defaults and per-kind overrides in correct priority order.
  - `Merge(localRegistry, projectRegistry AgentsRegistry) AgentsRegistry` performs deep-merge: for each group key, merge default + per-kind entries from localRegistry on top of projectRegistry (local wins).
  - Golden-fixture tests cover: (a) single-group registry; (b) multi-group registry (go + fe groups present); (c) per-kind override wins over group default; (d) local registry wins over project registry on Merge; (e) missing group falls back to empty preset (no panic).
  - `mage test-pkg ./internal/config` passes; `mage ci` green.
- **Blocked by:** — (Wave A head; no blockers)
- **Specify:**
  - **Objective:** Rewrite the `internal/config/agents.go` decoder to support the new `[<group>]` / `[<group>.<kind>]` multi-group schema shipped in W4.D2's TOML files. The decoder is the contract that all consumers of `agents.toml` go through; it must land before W4.D2 ships the new TOML content so that `mage ci` stays green.
  - **AcceptanceCriteria:** see Acceptance bullets above.
  - **ValidationPlan:** `mage test-pkg ./internal/config`; golden-fixture update via `mage test-golden-update` if applicable; `mage ci`.
  - **RiskNotes:**
    - The existing `AgentsRegistry` shape (today: `map[string]Preset` keyed by kind string, no group dimension) changes SHAPE. All call sites of `Resolve` and `Merge` in `internal/app/service.go` and any other consumer must be updated to pass the new `group` parameter. Builder must locate all call sites via LSP `findReferences` on `Resolve` and `Merge` before writing.
    - Today the decoder expects `[agents.build]` style keys. The new shape expects `[go.build]`. The TOML library's struct-tag decoder may need a custom decoder function or a `map[string]map[string]Preset` intermediate shape. Builder verifies the pelletier/go-toml/v2 decode patterns for nested tables before choosing struct vs map approach.
    - golden-fixture test names for the new multi-group merge cases are **new, not yet in tree** — builder creates them.
  - **ContextBlocks:**
    - `constraint` (critical): W0 MUST land before W4.D2. W4.D2 blocked_by W0 + W4.D1 in the plan graph.
    - `constraint` (high): call sites of `Resolve` in `internal/app` must still compile after W0 restructures the signature; builder updates all call sites in the same droplet.
    - `decision` (normal): `Resolve(registry, group, kind)` — group is a first-class parameter, not derived from kind. Pre-Drop-2 there was no group dimension; W0 adds it.
    - `warning` (high): changing `AgentsRegistry` type shape is a potential broad API break. Use LSP `findReferences` on all exported types + functions in `internal/config/agents.go` before editing.
    - `reference` (normal): REVISION_BRIEF §2.12a + §2.12 example TOML schema; Drop 4c.6 W0 for original struct shape.
  - **KindPayload:** `{"changes":[{"file":"internal/config/agents.go","symbol":"AgentsRegistry + Resolve + Merge","action":"modify","shape_hint":"AgentsRegistry = map[group]GroupConfig; GroupConfig = {Default Preset; Kinds map[kind]Preset}; Resolve(reg, group, kind) returns merged Preset; Merge deep-merges per-group entries"},{"file":"internal/config/agents_test.go","symbol":"TestResolve + TestMerge golden tests","action":"modify","shape_hint":"table-driven; 5 cases per function; golden fixtures in testdata/"},{"file":"internal/config/testdata/","symbol":"golden TOML fixtures","action":"add","shape_hint":"new, not yet in tree — multi-group TOML files for merge test cases"}]}`

---

### Wave W1 — Template Resolution (HOME Tier + Group-Aware Resolver)

#### 4c.6.1.W1 — sub-plan container

- **State:** todo
- **Kind:** `plan` (sub-plan container; spawns its own L2 planner)
- **Directory:** `workflow/drop_4c_6_1/DROP_4c.6.1.W1_TEMPLATE_RESOLUTION/`
- **Paths (expected):** `internal/app/service.go` (bakeProjectKindCatalog HOME-tier extension), `internal/app/dispatcher/cli_claude/render/render.go` (group-aware agent body resolver tier-1 update to subdir-per-group)
- **Packages:** `internal/app`, `internal/app/dispatcher/cli_claude/render`
- **Scope:**
  Add user HOME tier to the 3-tier template resolution chain in `bakeProjectKindCatalog` (currently: bare-root → primary-worktree → embedded). New 4-tier resolution:
    1. `<RepoBareRoot>/.tillsyn/template.toml` — project bare-root override (existing tier 1)
    2. `<RepoPrimaryWorktree>/.tillsyn/template.toml` — project worktree override (existing tier 2)
    3. `~/.tillsyn/templates/<group>.toml` — user HOME override per group (NEW — REVISION_BRIEF §2.1)
    4. Embedded `till-<group>.toml` — binary default (existing tier 4, renumbered from 3)
  For multi-group projects: walk tier 3 for EACH selected group, aggregating bindings + child_rules. HOME tier path: `platform.DefaultPathsWithOptions(opts).TemplatesDir` (or equivalent) + `/<group>.toml`.
  Update group-aware agent body resolver tier-1 (`render.go:assembleAgentFileBody`) from FLAT project lookup (`<project>/.tillsyn/agents/<name>.md`) to subdir-per-group lookup (`<project>/.tillsyn/agents/<group>/<name>.md`), per REVISION_BRIEF §2.2. Multi-group projects search declared groups in order; cross-group fallback to `gen` group remains (SKETCH §2.1 "cross-group fallback to gen group as last-resort").
- **Acceptance (L1 contract; L2 plan refines):**
  - `bakeProjectKindCatalog` walks the 4-tier chain; tier 3 reads `~/.tillsyn/templates/<group>.toml` when it exists; first-candidate-wins semantics preserved.
  - For multi-group projects, tier 3 walks all selected groups and aggregates.
  - `render.go:assembleAgentFileBody` tier-1 changes from flat lookup to `<project>/.tillsyn/agents/<group>/<name>.md` subdir-per-group.
  - Cross-group fallback to `gen` group preserved.
  - `mage test-pkg ./internal/app` passes; `mage test-pkg ./internal/app/dispatcher/cli_claude/render` passes; `mage ci` green.
- **Blocked by:** 4c.6.1.W4.D1 (W1's resolver update needs confirmed subdir-per-group layout from W4.D1 before the lookup path is final)
- **Source-of-truth Specify:** REVISION_BRIEF §2.1–2.2; SKETCH §1 + §2.1–2.2.
- **L2 sub-planner spawn directive:** "Decompose W1 into atomic droplets. Likely shape: D1 extend `bakeProjectKindCatalog` HOME-tier walk for single-group projects (verify `loadProjectTemplate` call chain — `bakeProjectKindCatalog` at service.go:416 calls `loadProjectTemplate` at service.go:529 which today does 3-tier; extend `loadProjectTemplate` with tier 3 or add a new multi-tier coordinator — verify exact call site via LSP); D2 extend for multi-group aggregation (multiple groups → aggregate bindings + child_rules); D3 `render.go:assembleAgentFileBody` tier-1 update from flat to `<group>/` subdir lookup. Wire `blocked_by` between droplets sharing `internal/app/service.go` (D1, D2 — serialize) and between those sharing `render.go` (D3 parallel to D1/D2 since different package). Confirm `platform.DefaultPathsWithOptions` exposes a `TemplatesDir` or equivalent method for `~/.tillsyn/templates/` — verify via LSP `documentSymbol` on `internal/platform/` before authoring D1."

---

### Wave W2 — Till Init Overhaul

#### 4c.6.1.W2 — sub-plan container

- **State:** todo
- **Kind:** `plan` (sub-plan container; spawns its own L2 planner)
- **Directory:** `workflow/drop_4c_6_1/DROP_4c.6.1.W2_TILL_INIT/`
- **Paths (expected):** `cmd/till/init_cmd.go`, `cmd/till/init_cmd_test.go`
- **Packages:** `cmd/till`
- **Scope:**
  Transform `till init` from single-group to multi-group, complete project record population, and add FLAT-layout detection:
  - **Multi-group CLI flag:** Replace `--group <name>` (singular) with `--group <name>` repeated cobra flag (e.g. `--group go --group fe`). Current `initJSONPayload.Group string` becomes `initJSONPayload.Groups []string`. Update `validateInitPayload`, `allowedInitGroups` (rename groups: `go`, `fe`, `gen` without `till-` prefix per W4.D1's restructure).
  - **FLAT layout detection (FF2 disposition):** Before copying agent files, detect whether `<project>/.tillsyn/agents/` contains files directly (FLAT layout from Drop 4c.6 or earlier). If FLAT layout detected, FAIL LOUD with a clear error message: `"FLAT agent layout detected at <project>/.tillsyn/agents/. Remove it and re-run: rm -rf <project>/.tillsyn/agents && till init --group <group>"`. NO migration code. NO silent skip.
  - **agents.toml re-init on schema shift (NIT3 from falsification):** On re-run, if `<destDir>/agents.toml` exists and its first `[agents]` header is detected (old schema), FAIL LOUD: `"agents.toml uses the old [agents.kind] schema. Remove it and re-run: rm <project>/agents.toml && till init --group <group>"`. Idempotent skip applies ONLY to files that are already in the new schema.
  - **Multi-group file copy:** `copyAgentFiles` changes from copying to `<project>/.tillsyn/agents/<name>.md` FLAT to copying to `<project>/.tillsyn/agents/<group>/<name>.md` subdir-per-group for each selected group.
  - **`template.toml` write:** After group selection, write `<project>/.tillsyn/template.toml` aggregated from selected groups. Source: `~/.tillsyn/templates/<group>.toml` per group if exists (HOME tier), else embedded `till-<group>.toml`. Idempotent (skip if exists). Per REVISION_BRIEF §2.4.
  - **`CreateProjectWithMetadata`:** Replace `CreateProject(ctx, name, description)` call in `createProjectDBRecord` with `CreateProjectWithMetadata(ctx, CreateProjectInput{...})` populating `RepoPrimaryWorktree = cwd`, `RepoBareRoot = <detect via git>`, `Language = <primary group>`, `Metadata.groups = [...]`. Per REVISION_BRIEF §2.5.
  - **TUI MCP confirm step:** Add a bubbletea step after group selection that asks y/n for `.mcp.json` registration (default YES). Uses the new `confirm.go` component from W5. Per REVISION_BRIEF §2.6.
  - **TUI multi-group picker:** Replace single-select group picker in `initTUIModel` with multi-select (uses `picker_multi.go` from W5). Per REVISION_BRIEF §2.3.
  - **Re-run safety:** New schema group subdirs are idempotent (existing files at new paths skipped). FLAT and old-schema files fail loud (see above).
- **Acceptance (L1 contract; L2 plan refines):**
  - `till init --group go --group fe` creates `<project>/.tillsyn/agents/go/<name>.md` and `<project>/.tillsyn/agents/fe/<name>.md` subdirs with 10 files each (9 standard + `orchestrator-managed.md`).
  - `till init --json '{"name":"...","groups":["go","fe"],"mcp":true}'` identical behavior (JSON payload uses `groups []string`).
  - FLAT layout detection: running `till init` on a project with existing FLAT `<project>/.tillsyn/agents/*.md` exits with non-zero + clear error message + remediation instructions.
  - Old `[agents.kind]` schema in `agents.toml`: running `till init` on a project with old-schema `agents.toml` exits with non-zero + clear error message + remediation instructions.
  - `<project>/.tillsyn/template.toml` written after init.
  - Project record has `RepoPrimaryWorktree`, `Language`, `Metadata.groups` populated.
  - TUI walk prompts for `.mcp.json` registration with default=yes.
  - Re-run on clean-state project (new schema): added=0, skipped=N for existing files.
  - `mage test-pkg ./cmd/till/...` passes; `mage ci` green.
- **Blocked by:** 4c.6.1.W1 (W2 writes `template.toml` using the HOME-tier path convention defined by W1; also needs W1's resolver group-path shape confirmed before subdir-per-group copy is final), 4c.6.1.W4.D1 (W2 copies agent files from embedded `agents/<group>/` subdirs that W4.D1 creates), 4c.6.1.W5 (W2's TUI uses `confirm.go` and `picker_multi.go` from W5)
- **Source-of-truth Specify:** REVISION_BRIEF §2.3–2.6; SKETCH §2 + §4.1.
- **L2 sub-planner spawn directive:** "Decompose W2 into atomic droplets. Likely shape: D1 update `initJSONPayload` Group→Groups + `validateInitPayload` + `allowedInitGroups` renaming (remove `till-` prefix per W4.D1's new group names: `go`, `fe`, `gen`); D2 FLAT layout detection + agents.toml old-schema detection (fail loud per FF2 disposition — new checks in `copyAgentFiles` and re-init path); D3 multi-select TUI picker for group selection (uses `picker_multi.go` from W5); D4 TUI MCP confirm step (uses `confirm.go` from W5); D5 `copyAgentFiles` refactor to subdir-per-group (for each selected group, copy to `<project>/.tillsyn/agents/<group>/`); D6 `template.toml` write (aggregate from HOME or embedded per group); D7 `createProjectDBRecord` upgrade to `CreateProjectWithMetadata` with bare-root detection via `git rev-parse --git-common-dir`. Wire `blocked_by` among all droplets sharing `cmd/till/init_cmd.go` — serialize D1→D2→D3→D4→D5→D6→D7 (single-file chain). Confirm group-name change: current `allowedInitGroups = ['till-gen', 'till-go']` becomes `['gen', 'go', 'fe']` (verify W4.D1 canonical names before finalizing D1). The TUI component imports (D3/D4) require W5 to have shipped — sub-planner must confirm W5 is `complete` before dispatching D3/D4. Per NIT3 (falsification): old-schema `agents.toml` detection must be in D2 or D5; document the exact header-detection heuristic (check for presence of `[agents.` prefix in first N lines of file). CONSUMER-TIE TEST CONTRACT (R2-NIT3): tests invoke `run(ctx, args, &out, io.Discard)` end-to-end — flow-level assertions, not unit assertions on internal helpers. All D-series droplets sharing `init_cmd.go` follow this pattern."

---

### Wave W3 — CLI Surface

#### 4c.6.1.W3 — sub-plan container

- **State:** todo
- **Kind:** `plan` (sub-plan container; spawns its own L2 planner)
- **Directory:** `workflow/drop_4c_6_1/DROP_4c.6.1.W3_CLI_SURFACE/`
- **Paths (expected):** `cmd/till/template_cli.go` (NEW), `cmd/till/template_cli_test.go` (NEW), `cmd/till/agents_cli.go` (NEW), `cmd/till/agents_cli_test.go` (NEW), `cmd/till/project_cli.go` (MODIFY — add update/delete/archive/restore/rename), `cmd/till/project_cli_test.go` (MODIFY), `cmd/till/action_item_cli.go` (MODIFY — add create), `cmd/till/action_item_cli_test.go` (MODIFY), `cmd/till/main.go` (MODIFY — register new commands)
- **Packages:** `cmd/till`
- **Scope:**
  Wire 15 new CLI subcommands (REVISION_BRIEF §2.7–2.10). All follow the existing CONSUMER-TIE TEST CONTRACT (`run(ctx, args, &out, io.Discard)` end-to-end pattern from Drop 4c.6 W2):

  **Template CLIs (`till template`):**
  - `save --from-project <id> --group <group>` — reads project's `<project>/.tillsyn/template.toml` `[<group>]` block, writes to `~/.tillsyn/templates/<group>.toml`. Idempotent (overwrites with user confirm before overwrite). Per §2.7.
  - `list` — show user's HOME templates + embedded defaults side-by-side. Per §2.7.
  - `show --group <group> --source {home|embedded}` — print TOML content. Per §2.7.
  - `diff --group <group>` — diff HOME tier vs embedded default. Per §2.7.
  - `restore --group <group>` — copy embedded default to HOME with confirm. Per §2.7.

  **Agents CLIs (`till agents`):**
  - `save --from-project <id> --group <group>` — read `<project>/.tillsyn/agents/<group>/*.md`, write to `~/.tillsyn/agents/<group>/*.md`. Per §2.7.
  - `list` — show HOME agents + embedded defaults. Per §2.7.
  - `show --group <group> --agent <name> --source {home|embedded}` — print agent body. Per §2.7.
  - `diff --group <group> --agent <name>` — diff HOME vs embedded. Per §2.7.
  - `bootstrap --from <path> [--to <path>] [--dry-run] [--force]` — map `~/.claude/agents/<group>-<role>-agent.md` → `~/.tillsyn/agents/<group>/<role>-agent.md`. **2-into-4 QA fan-out**: source `<group>-qa-proof-agent.md` seeds BOTH `plan-qa-proof-agent.md` AND `build-qa-proof-agent.md` at destination (same for qa-falsification). Group-agnostic agents (closeout, commit-message if no group prefix) copied to each known group dir. Missing files reported; orchestrator-managed.md starter generated. Per §2.17. QA-SPLIT-R1 tracks proper per-role differentiation in Drop 4c.8.

  **Project lifecycle CLIs (`till project update/delete/archive/restore/rename`):**
  - `update --project-id <id> [--root-path ...] [--bare-root ...] [--language ...] [--add-group <name>] [--remove-group <name>] [--hylla-artifact-ref ...] [--description ...]` — calls `(*Service).UpdateProject`. Per §2.8. Closes D7-R3 + D7-R2.
  - `delete --project-id <id> --confirm` — hard delete. Per §2.10.
  - `archive --project-id <id>` — soft archive. Per §2.10.
  - `restore --project-id <id>` — un-archive. Per §2.10.
  - `rename --project-id <id> --name <new-name>` — rename + reslug. Per §2.10.

  **Action item create CLI (`till action_item create`) — FF4 disposition:**
  - `create --project-id <id> --kind <kind> --title <title> --description <desc> [--paths ...] [--packages ...] [--files ...] [--blocked-by <id>] [--metadata-json ...] [--parent-id <id>] [--structural-type <drop|segment|confluence|droplet>] [--role <role>]`
  - `--structural-type` is OPTIONAL. Smart-default per `--kind`:
    - `plan` → `segment`
    - `refinement` → `segment`
    - `research`, `build`, `plan-qa-proof`, `plan-qa-falsification`, `build-qa-proof`, `build-qa-falsification`, `closeout`, `commit`, `discussion`, `human-verify` → `droplet`
  - When `--structural-type <val>` is passed, validate against closed enum (`drop`/`segment`/`confluence`/`droplet`). Fail loud on invalid value.
  - `--role` is OPTIONAL (closed enum; empty is valid pre-Drop-2). Builder L2 confirms the closed-enum list via LSP `goToDefinition` on `domain.Role`.
  - Help text documents the smart-default mapping.
  - Returns created action item UUID + dotted address.
  - Calls `(*Service).CreateActionItem`. Per §2.9.

  Note: `(*Service).UpdateProject`, `(*Service).ArchiveProject`, `(*Service).RestoreProject`, `(*Service).DeleteProject`, `(*Service).CreateActionItem` all exist (LSP-verified at service.go). CLI wiring only — no new service methods needed for project/action_item CLIs. Template/agents CLIs perform direct file I/O to `~/.tillsyn/templates/` and `~/.tillsyn/agents/` directories.

- **Acceptance (L1 contract; L2 plan refines):**
  - All 5 `till template` subcommands register + execute per acceptance in §2.7.
  - All 4 `till agents` save/list/show/diff subcommands register + execute per acceptance in §2.7.
  - `till agents bootstrap --from ~/.claude/agents --dry-run` prints copy plan without writing.
  - `till agents bootstrap --from ~/.claude/agents` copies agent files to `~/.tillsyn/agents/<group>/` with 2-into-4 QA fan-out, reports missing files, generates orchestrator-managed.md starter.
  - `till project update` updates existing project's metadata fields. Per §2.8.
  - `till project delete/archive/restore/rename` work per §2.10.
  - `till action_item create` creates action item, returns UUID. Per §2.9.
  - `till action_item create --kind plan` defaults to `structural-type=segment` without requiring `--structural-type` flag.
  - `till action_item create --kind build` defaults to `structural-type=droplet`.
  - `till action_item create --structural-type invalid` fails with a clear error + valid-values list.
  - All commands follow CONSUMER-TIE TEST CONTRACT (`run(ctx, args, &out, io.Discard)` end-to-end pattern from Drop 4c.6 W2); `mage test-pkg ./cmd/till/...` passes.
  - `mage ci` green.
- **Blocked by:** 4c.6.1.W2 (W3 adds files to `cmd/till` package; W2 modifies `init_cmd.go` in the same package — must serialize to avoid package-compile race; additionally W3's `till template save/list/show/diff/restore` depends on `~/.tillsyn/templates/` path convention finalized by W1), 4c.6.1.W1 (HOME-tier path contract needed before template/agents CLI file-I/O paths are finalized)
- **Source-of-truth Specify:** REVISION_BRIEF §2.7–2.10; SKETCH §1 + §2.
- **L2 sub-planner spawn directive:** "Decompose W3 into atomic droplets. Suggested batching: D1 `till project update` (calls existing `(*Service).UpdateProject` — no new service method; verify flags map to `UpdateProjectInput` fields via LSP on service.go); D2 `till project delete/archive/restore/rename` (4 subcommands, 1 droplet each or grouped as 2 droplets); D3 `till action_item create` (calls existing `(*Service).CreateActionItem`; implements FF4 smart-default logic for `--structural-type`; verify `CreateActionItemInput.StructuralType` required field + `domain.StructuralType` closed enum values via LSP before writing); D4 `till template save/list/show/diff/restore` (5 subcommands, file-I/O to `~/.tillsyn/templates/<group>.toml` — direct OS-level file ops, NO new service methods); D5 `till agents save/list/show/diff` (4 subcommands, file-I/O to `~/.tillsyn/agents/<group>/`); D6 `till agents bootstrap` (REVISION_BRIEF §2.17 — reads `--from <path>` dir for `<group>-<role>-agent.md` files, maps to `<to>/<group>/<role>-agent.md`; 2-into-4 QA fan-out: source `<group>-qa-proof-agent.md` seeds BOTH `plan-qa-proof-agent.md` AND `build-qa-proof-agent.md`; group-agnostic files copied to each known group dir; missing files reported; orchestrator-managed.md starter generated; `--dry-run` preview mode; tests exercise dry-run + actual copy + fan-out + missing-file reporting via CONSUMER-TIE pattern; **R3-NIT6**: `--force` flag help text + docstring MUST explicitly warn "Overwrites destination files; any post-bootstrap customization is lost. Use `till agents save --from-project <id>` to push customization back to HOME tier before re-running bootstrap with `--force`" — documented so users don't silently lose customization); D7 `cmd/till/main.go` command registration for all new commands. Wire `blocked_by` between any droplets sharing `cmd/till/main.go` (D7 last, after all command files exist) and any sharing `project_cli.go` / `action_item_cli.go` (D1/D2 share project_cli.go — serialize; D3 owns action_item_cli.go — parallel with D1/D2 for package safety, BUT same `cmd/till` package compile — wire `blocked_by` D3 after D1 or use package lock). D4, D5, D6 create NEW or modify existing files but same `cmd/till` package — wire `blocked_by` to prevent concurrent build of same package. All droplets use CONSUMER-TIE TEST CONTRACT (`run(ctx, args, &out, io.Discard)` end-to-end pattern from Drop 4c.6 W2)."

---

### Wave W4 — Agent Set Restructure + Schema Shift

#### 4c.6.1.W4.D1 — Restructure embedded agent dirs (orphan deletion + qa split + fe group + embed.go)

- **State:** todo
- **Kind:** `build` (atomic droplet; `Irreducible: true`)
- **Paths:**
  - `internal/templates/builtin/agents/till-go/go-builder-agent.md` (DELETE — orphan)
  - `internal/templates/builtin/agents/till-go/go-planning-agent.md` (DELETE — orphan)
  - `internal/templates/builtin/agents/till-go/go-qa-falsification-agent.md` (DELETE — orphan)
  - `internal/templates/builtin/agents/till-go/go-qa-proof-agent.md` (DELETE — orphan)
  - `internal/templates/builtin/agents/till-go/go-research-agent.md` (DELETE — orphan)
  - `internal/templates/builtin/agents/till-go/qa-proof-agent.md` (REPLACE CONTENTS → becomes plan-qa-proof-agent.md + source for build-qa-proof-agent.md)
  - `internal/templates/builtin/agents/till-go/qa-falsification-agent.md` (REPLACE CONTENTS → becomes plan-qa-falsification-agent.md + source for build-qa-falsification-agent.md)
  - `internal/templates/builtin/agents/till-go/plan-qa-proof-agent.md` (NEW — split from qa-proof-agent.md)
  - `internal/templates/builtin/agents/till-go/plan-qa-falsification-agent.md` (NEW)
  - `internal/templates/builtin/agents/till-go/build-qa-proof-agent.md` (NEW)
  - `internal/templates/builtin/agents/till-go/build-qa-falsification-agent.md` (NEW)
  - `internal/templates/builtin/agents/till-go/orchestrator-managed.md` (ADD if absent in till-go; verify via Read first — it may exist already in till-gen only)
  - `internal/templates/builtin/agents/till-gen/qa-proof-agent.md` (REPLACE CONTENTS → split)
  - `internal/templates/builtin/agents/till-gen/qa-falsification-agent.md` (REPLACE CONTENTS → split)
  - `internal/templates/builtin/agents/till-gen/plan-qa-proof-agent.md` (NEW)
  - `internal/templates/builtin/agents/till-gen/plan-qa-falsification-agent.md` (NEW)
  - `internal/templates/builtin/agents/till-gen/build-qa-proof-agent.md` (NEW)
  - `internal/templates/builtin/agents/till-gen/build-qa-falsification-agent.md` (NEW)
  - `internal/templates/builtin/agents/till-gen/orchestrator-managed.md` (KEEP — do NOT delete; FF3 disposition)
  - `internal/templates/builtin/agents/fe/` (NEW dir — 10 placeholder files: 9 standard + orchestrator-managed.md)
  - `internal/templates/builtin/embed.go` (MODIFY — update //go:embed list explicitly per-file)
  - `internal/templates/embed_test.go` (MODIFY — FS introspection test updates)
  - `internal/templates/builtin/till-gdd/` (AUDIT-ONLY — till-gdd has 7 agents; verify no action needed per §2.11 scope; do NOT add 4 new qa agents to till-gdd unless REVISION_BRIEF explicitly requires it)
- **Packages:** `internal/templates`
- **Acceptance:**
  - Final `till-go/` agent set (**10 files**): `planning-agent.md`, `builder-agent.md`, `plan-qa-proof-agent.md`, `plan-qa-falsification-agent.md`, `build-qa-proof-agent.md`, `build-qa-falsification-agent.md`, `research-agent.md`, `closeout-agent.md`, `commit-message-agent.md`, `orchestrator-managed.md`. No `go-*` orphan files. No old `qa-proof-agent.md` / `qa-falsification-agent.md` (2-file model).
  - Final `till-gen/` agent set (**10 files**): same 10 names. `orchestrator-managed.md` KEPT (FF3 disposition — do NOT delete).
  - Final `fe/` agent set (**10 placeholder files**): same 10 names. Each file body: `# PLACEHOLDER — substantive FE-stack-agnostic content lands in Drop 4c.8 W4` plus frontmatter `name: <name>`, `description: ...placeholder...`.
  - All new agent placeholder files use `name` + `description` frontmatter ONLY (no `model:`, no `tools:`) per Drop 4c.6 W5.D3 convention.
  - `//go:embed` list in `embed.go` is explicit per-file (NOT `**/*.md` glob) — lists all files for `till-go/`, `till-gen/`, `till-gdd/`, `fe/`. Per Drop 4c.6 F.2.1 falsification-mitigation pattern.
  - `embed_test.go` FS-introspection test updated to assert all expected paths resolve, including `fe/orchestrator-managed.md` and all 4 new qa-agent files per group.
  - `git ls-files internal/templates/builtin/agents/` shows 10+10+7+10=37 total agent files (till-go=10, till-gen=10, till-gdd=7 unchanged, fe=10).
  - `mage test-pkg ./internal/templates` passes; `mage ci` green.
- **Blocked by:** — (Wave A head; no blockers)
- **Specify:**
  - **Objective:** Restructure embedded agent dirs to the 10-agent-per-group standard (plan-qa-proof separate from build-qa-proof, `go-` orphans deleted, `orchestrator-managed.md` kept in till-gen and added to till-go and fe, `fe/` group added), update embed.go to list new files explicitly.
  - **AcceptanceCriteria:** see Acceptance bullets above.
  - **ValidationPlan:** `git ls-files internal/templates/builtin/agents/` count check; `mage test-pkg ./internal/templates`; `mage ci`.
  - **RiskNotes:**
    - `orchestrator-managed.md` in `till-gen/`: KEEP per FF3 disposition. DO NOT delete. This file covers closeout/refinement/discussion/human-verify kind bindings in `till-go.toml`. Deleting it would break those bindings. ORCH-MANAGED-R1 tracks future split into role-specific agents in Drop 4c.8.
    - `orchestrator-managed.md` presence in `till-go/` vs `till-gen/` only: builder must `Read` the `till-go/` directory listing first to determine if `orchestrator-managed.md` already exists there. If it does, keep it. If it does not, add it (copy from till-gen's content as starting point).
    - `till-gdd/` currently has 7 agent files (not 10). REVISION_BRIEF §2.11 mentions "Confirm gen/ group has 10 placeholders" and add `fe/` group but does NOT list `till-gdd` for the 10-agent expansion. Leave `till-gdd` at 7 files unless an explicit split is required; document in this droplet's QA verdict.
    - Deletion of 5 `go-*` orphan files uses `git rm` (not `os.Remove`) to preserve git history.
    - `embed.go` explicit per-file list: the builder MUST verify the existing explicit list pattern in `embed.go` before editing (Drop 4c.6 W1.D1 pattern — read embed.go before writing).
    - Old `qa-proof-agent.md` and `qa-falsification-agent.md` files in `till-go/` and `till-gen/` should be deleted (or their content forked into the 4 new files first). The old 2-file model is superseded. Builder reads the old file content, copies forward to `plan-qa-*-agent.md`, then deletes the old file via `git rm`.
  - **ContextBlocks:**
    - `constraint` (high): explicit per-file embed.go list — never `**/*.md` glob.
    - `constraint` (high): 10 standard agent names (including `orchestrator-managed.md`) must be IDENTICAL across groups (same filenames, different content per group).
    - `constraint` (critical): do NOT delete `orchestrator-managed.md` from till-gen. FF3 disposition is KEEP. Deleting it would break 4 kind bindings in till-go.toml and till-gen.toml.
    - `decision` (normal): Drop `go-` prefix per SKETCH §2.1 — group subdir is the distinguisher, not filename prefix.
    - `warning` (high): `till-go.toml` and `till-gen.toml` agent_name references will still use the old names until W4.D2 updates them. The builder for W4.D1 does NOT touch the TOML files — W4.D2 handles that.
    - `reference` (normal): Drop 4c.6 W1.D1 pattern for explicit embed.go list + placeholder frontmatter convention.
  - **KindPayload:** `{"changes":[{"file":"internal/templates/builtin/agents/till-go/","symbol":"5 go-* orphan files","action":"delete","shape_hint":"git rm; preserves history"},{"file":"internal/templates/builtin/agents/till-go/{plan-qa-proof,plan-qa-falsification,build-qa-proof,build-qa-falsification}-agent.md","symbol":"4 new QA agent files","action":"add","shape_hint":"PLACEHOLDER frontmatter; name+description only; plan-qa files copy from old qa-proof-agent.md"},{"file":"internal/templates/builtin/agents/till-go/orchestrator-managed.md","symbol":"orchestrator-managed.md","action":"add if absent","shape_hint":"copy from till-gen if not present; verify first"},{"file":"internal/templates/builtin/agents/till-gen/","symbol":"old qa files","action":"delete+split","shape_hint":"keep orchestrator-managed.md; split old qa-proof/qa-falsification into 4 new files"},{"file":"internal/templates/builtin/agents/fe/","symbol":"10 new placeholder agent files","action":"add","shape_hint":"new dir; FE-generic placeholder content; same 10 names"},{"file":"internal/templates/embed.go","symbol":"DefaultTemplateFS","action":"modify","shape_hint":"extend //go:embed explicit list with all new + renamed files; remove deleted file entries"},{"file":"internal/templates/embed_test.go","symbol":"FS introspection test","action":"modify","shape_hint":"assert 10-agent paths per group + fe dir resolved"}]}`

#### 4c.6.1.W4.D2 — Schema shift TOML files + agents.example.toml + new till-fe.toml

- **State:** todo
- **Kind:** `build` (atomic droplet; `Irreducible: true`)
- **Paths:**
  - `internal/templates/builtin/till-go.toml` (MODIFY — update `[agent_bindings.<kind>]` agent_name to match new 10-agent names: `plan-qa-proof-agent`, `build-qa-proof-agent`, etc.)
  - `internal/templates/builtin/till-gen.toml` (MODIFY — same agent_name updates)
  - `internal/templates/builtin/agents.example.toml` (MODIFY — shift from `[agents]` top-level + `[agents.plan]` per-kind to `[go]` top-level + `[go.plan-qa-proof]` per REVISION_BRIEF §2.12)
  - `internal/templates/builtin/till-fe.toml` (NEW — `fe` group template TOML)
  - `internal/templates/builtin/embed.go` (MODIFY — add `builtin/till-fe.toml` to //go:embed list)
  - `internal/templates/embed_test.go` (MODIFY — assert `till-fe.toml` path resolves)
- **Packages:** `internal/templates`
- **Acceptance:**
  - `till-go.toml` `[agent_bindings.<kind>]` `agent_name` values updated with `-agent` suffix: `plan-qa-proof-agent`, `plan-qa-falsification-agent`, `build-qa-proof-agent`, `build-qa-falsification-agent` match the new 10-agent file names from W4.D1. The 4 orchestrator-managed bindings (`closeout`, `refinement`, `discussion`, `human-verify`) continue to reference `orchestrator-managed` (no `-agent` suffix — this is the special 10th file, not a standard agent).
  - `till-gen.toml` same updates.
  - `agents.example.toml` sections: `[go]` replaces `[agents]`; `[go.plan-qa-proof]` replaces `[agents.plan-qa-proof]` etc. Full schema per REVISION_BRIEF §2.12. Both `[go]` and `[fe]` group sections present.
  - `till-fe.toml` (NEW) ships at `internal/templates/builtin/till-fe.toml` with minimal cascade template structure for `fe` group per the `[<group>.<kind>]` schema. Agent bindings reference the 10 standard agent names (9 standard + `orchestrator-managed`).
  - `embed.go` `//go:embed` directive extended to include `builtin/till-fe.toml`.
  - `embed_test.go` updated to assert `till-fe.toml` path resolves.
  - `git grep '\[agents\.'` post-edit returns zero hits in `internal/templates/builtin/`.
  - `mage test-pkg ./internal/templates` passes; `mage ci` green.
- **Blocked by:** 4c.6.1.W4.D1 (W4.D2 updates agent_name references that must match the W4.D1 file names; also edits `embed.go` which W4.D1 already modified — must serialize to avoid rebase), 4c.6.1.W0 (decoder contract must be updated before schema shift TOML lands — `mage ci` tests will decode the new TOML using W0's new structs)
- **Specify:**
  - **Objective:** Migrate the shipped TOML files and `agents.example.toml` to the new `[<group>.<kind>]` multi-group schema, update agent_name bindings to match W4.D1's new 10-agent filenames (including `-agent` suffix on the 4 QA agents), and ship `till-fe.toml` placeholder for the `fe` group.
  - **AcceptanceCriteria:** see Acceptance bullets above.
  - **ValidationPlan:** `git grep '\[agents\.'` post-edit returns zero hits; `mage test-pkg ./internal/templates`; `mage ci`.
  - **RiskNotes:**
    - Schema shift from `[agents.plan]` to `[go.plan]` is a BREAKING change for existing `agents.toml` files in user projects. Pre-MVP: acceptable. W2 already handles old-schema detection + fail-loud for re-init.
    - `till-go.toml` has ~380 lines of content. Builder reads the file before editing to confirm which section headers need renaming.
    - `agents.example.toml` currently has `[agents.plan]` sections — all must become `[go.plan]` with a `[go]` top-level defaults block. Builder must verify the new schema matches REVISION_BRIEF §2.12 example exactly.
    - `embed.go` edit in W4.D2 must NOT revert or conflict with W4.D1's changes. Builder reads embed.go post-W4.D1 via `git diff HEAD` or `Read` before writing.
    - The 4 `orchestrator-managed` bindings in `till-go.toml` / `till-gen.toml` continue to reference `orchestrator-managed` (without `-agent` suffix). Do NOT rename these — the file is `orchestrator-managed.md` per FF3 disposition. Only the 4 new QA agents get the `-agent` suffix.
  - **ContextBlocks:**
    - `constraint` (high): `agents.example.toml` schema is user-facing documentation; must exactly match REVISION_BRIEF §2.12 worked example.
    - `constraint` (high): agent_name values in TOML must exactly match the filenames from W4.D1 (e.g. `plan-qa-proof-agent` not `plan-qa-proof`).
    - `decision` (normal): No `agents.` prefix per SKETCH §10 — file name is already self-documenting.
    - `warning` (high): `orchestrator-managed` bindings keep the exact value `orchestrator-managed` (no `-agent` suffix). Only the 9 standard agent bindings use `<name>-agent`.
    - `reference` (normal): REVISION_BRIEF §2.12 example TOML schema is the canonical reference.
  - **KindPayload:** `{"changes":[{"file":"internal/templates/builtin/till-go.toml","symbol":"[agent_bindings.<kind>] agent_name","action":"modify","shape_hint":"update 4 qa-related agent_name bindings to plan-qa-proof-agent, plan-qa-falsification-agent, build-qa-proof-agent, build-qa-falsification-agent (with -agent suffix); keep orchestrator-managed unchanged"},{"file":"internal/templates/builtin/till-gen.toml","symbol":"[agent_bindings.<kind>] agent_name","action":"modify","shape_hint":"same updates as till-go.toml"},{"file":"internal/templates/builtin/agents.example.toml","symbol":"section headers","action":"modify","shape_hint":"[agents] → [go]; [agents.plan] → [go.plan]; add [fe] section"},{"file":"internal/templates/builtin/till-fe.toml","symbol":"fe group template","action":"add","shape_hint":"minimal cascade template with agent_bindings for 10 standard agent names"},{"file":"internal/templates/embed.go","symbol":"DefaultTemplateFS","action":"modify","shape_hint":"extend //go:embed to include builtin/till-fe.toml"}]}`

---

### Wave W5 — TUI Components + Style System

#### 4c.6.1.W5 — sub-plan container

- **State:** todo
- **Kind:** `plan` (sub-plan container; spawns its own L2 planner)
- **Directory:** `workflow/drop_4c_6_1/DROP_4c.6.1.W5_TUI_COMPONENTS/`
- **Paths (expected):**
  - `internal/tui/components/confirm.go` (NEW), `internal/tui/components/confirm_test.go` (NEW)
  - `internal/tui/components/textinput.go` (NEW), `internal/tui/components/textinput_test.go` (NEW)
  - `internal/tui/components/picker_single.go` (NEW), `internal/tui/components/picker_single_test.go` (NEW)
  - `internal/tui/components/picker_multi.go` (NEW), `internal/tui/components/picker_multi_test.go` (NEW)
  - `internal/tui/components/header.go` (NEW), `internal/tui/components/footer.go` (NEW)
  - `internal/tui/components/progress.go` (NEW)
  - `internal/tui/style/palette.go` (NEW)
  - `internal/tui/style/spacing.go` (NEW)
  - `internal/tui/style/typography.go` (NEW)
  - `internal/tui/keybindings/dispatcher.go` (NEW)
  - `internal/tui/keybindings/loader.go` (NEW)
  - `internal/tui/keybindings/modes.go` (NEW)
  - `internal/tui/keybindings/dispatcher_test.go` (NEW)
- **Packages:** `internal/tui/components` (NEW package), `internal/tui/style` (NEW package), `internal/tui/keybindings` (NEW package)
- **Scope:**
  Build the inline TUI component library at `internal/tui/components/`, style system at `internal/tui/style/`, and vim keybinding dispatcher at `internal/tui/keybindings/`. Every file carries `// MIGRATION TARGET: github.com/hylla-org/lykta` doc-comment (EXTRACT-R1 + KEYBIND-R1 tracked). Components use Bubble Tea v2 + Bubbles v2 + Lip Gloss v2 (all existing dependencies in go.mod).

  **Components (REVISION_BRIEF §2.14 + SKETCH §4.2):**
  - `confirm.go` — y/n prompt with default. `type ConfirmModel struct{...}` implementing `tea.Model`. Used by till init MCP confirm (W2) and future destructive-action confirms (W3 template/agents save).
  - `textinput.go` — single-line text input with validation. Wrapper over `bubbles/textinput` with Tillsyn styling + validation hook.
  - `picker_single.go` — styled single-select list. Used by future `till template show --source` choice.
  - `picker_multi.go` — styled multi-select list. Used by till init multi-group picker (W2).
  - `header.go` / `footer.go` — styled chrome bars.
  - `progress.go` — single-step status line.

  **Style system (SKETCH §4.3):**
  - `palette.go` — colors via Lip Gloss styles, semantic names (primary, accent, success, warning, error, muted). Where possible, mirrors `stil` token values (translated from CSS custom properties to Lip Gloss `lipgloss.Color`).
  - `spacing.go` — padding/margin constants.
  - `typography.go` — text style helpers (heading, body, label, code).

  **Vim keybinding dispatcher (REVISION_BRIEF §2.14 — new package `internal/tui/keybindings/`):**
  - `dispatcher.go` — Go-side keybinding dispatcher. Consumes stil's `baseline.json` at `/Users/evanschultz/Documents/Code/hylla/stil/main/src/bindings/baseline.json` AND the Tillsyn-local `<project>/.tillsyn/bindings.json` (§2.19) at startup. Dispatches Bubble Tea key events to handlers per mode (nav / insert / visual / visual-block / command / hint).
  - `loader.go` — JSON-decode the baseline + Tillsyn-local extension files into the dispatcher's binding table. **Merge semantic (R3-FF2 disposition)**: load stil's `product_extensions.tillsyn.commands` first (baseline's 4 commands), then ID-based deep merge with local file's `product_extensions.tillsyn.commands` (local's 5 commands); local wins on collision. Absent local file = baseline-only (4 commands); NOT a fail-loud condition.
  - `modes.go` — mode-state machine (nav / insert / visual / visual-block / command / hint).
  - `dispatcher_test.go` — table-tested per-binding dispatch per mode.
  - All TUI components route key events through this dispatcher (the W2 `runInitTUI` refactor uses vim-style keys consistent with the rest of the TUI).
  - Migration marker: `// MIGRATION TARGET: github.com/hylla-org/lykta` (co-extracts with components + style).

  All component implementations are pure Bubble Tea v2 models (Init/Update/View). Tests use `teatest_v2` per existing patterns in `internal/tui/`.
- **Acceptance (L1 contract; L2 plan refines):**
  - All 7 component files + 3 style files + 4 keybinding files exist and compile.
  - Each file carries `// MIGRATION TARGET: github.com/hylla-org/lykta` at package doc-comment level.
  - `confirm.go` implements `tea.Model`; renders y/n prompt; `Confirmed()` / `Cancelled()` accessors work.
  - `picker_multi.go` implements `tea.Model`; returns `[]string` of selected items; handles Enter (confirm), Space (toggle), Esc (cancel).
  - `dispatcher.go` loads stil baseline.json + Tillsyn-local bindings.json with ID-based deep merge; command palette contains 9 commands when local file present (baseline's 4 + local's 5) or 4 commands when local absent; `Dispatch(keyMsg, mode)` returns registered handler or no-op.
  - `mage test-pkg ./internal/tui/components` passes; `mage test-pkg ./internal/tui/style` passes; `mage test-pkg ./internal/tui/keybindings` passes; `mage ci` green.
- **Blocked by:** — (Wave A head; no blockers)
- **Source-of-truth Specify:** REVISION_BRIEF §2.14; SKETCH §4; SKETCH §10 keybinding dispatcher row.
- **L2 sub-planner spawn directive:** "Decompose W5 into atomic droplets. Likely shape: D1 style system (`palette.go`, `spacing.go`, `typography.go` — new package `internal/tui/style`; no deps on components); D2 `confirm.go` + `progress.go` (simple models; `confirm.go` needed by W2 first); D3 `textinput.go` (wrapper over bubbles/textinput); D4 `picker_single.go` + `picker_multi.go` (related; both use list-selection pattern — 1 droplet each or combined if they share a base type); D5 `header.go` + `footer.go` (styled chrome; simple); D6 vim keybinding dispatcher (`dispatcher.go` + `loader.go` + `modes.go` + `dispatcher_test.go` — new package `internal/tui/keybindings`). Wire `blocked_by` between droplets that share the same Go package compile (all new files create NEW packages `internal/tui/components` and `internal/tui/keybindings` — first droplet creates each package, subsequent droplets add to it; serialize D2→D3→D4→D5 or allow parallel if the package is stable after D2's creation). Style package (D1) and keybinding package (D6) are independent of components and can run in parallel with D2-D5 AND with each other since they are separate packages. The MIGRATION TARGET doc-comment on every file is a hard requirement — plan-QA falsification will attack any file missing it. D6 (keybinding dispatcher): loader implements ID-based deep merge per R3-FF2 disposition — (1) load stil baseline `product_extensions.tillsyn.commands` (4 entries), (2) if `.tillsyn/bindings.json` present: ID-merge local `product_extensions.tillsyn.commands` (5 entries) into baseline; local wins on collision, (3) if absent: baseline-only (4 commands); graceful fallback, NOT fail-loud. Command palette exposes the merged command set to the dispatcher."

---

### Wave W6 — FE Scaffold

#### 4c.6.1.W6 — sub-plan container

- **State:** todo
- **Kind:** `plan` (sub-plan container; spawns its own L2 planner)
- **Directory:** `workflow/drop_4c_6_1/DROP_4c.6.1.W6_FE_SCAFFOLD/`
- **Paths (expected):**
  - `fe/main.go` (NEW — Wails main + Service bindings + DEFAULT NATIVE MENU: Quit/About/Hide/Minimize/etc.)
  - `fe/wails.json` (NEW — Wails config)
  - `fe/go.mod` (NEW — separate module; imports main module via replace directive)
  - `fe/frontend/package.json` (NEW)
  - `fe/frontend/astro.config.mjs` (NEW)
  - `fe/frontend/pnpm-lock.yaml` (NEW)
  - `fe/frontend/public/stil-tokens.css` (NEW — built artifact from stil or symlink)
  - `fe/frontend/src/pages/` (NEW — Astro pages: projects.astro, project-detail.astro, settings.astro)
  - `fe/frontend/src/components/` (NEW — Tillsyn FE components with MIGRATION TARGET comments)
  - `fe/frontend/src/layouts/` (NEW — Astro layout)
  - `fe/frontend/src/lib/` (NEW — client-side helpers + Wails IPC wrappers)
  - `fe/frontend/src/lib/vim/engine.ts` (NEW — vim engine consuming stil baseline.json + Tillsyn-local bindings.json)
  - `fe/frontend/src/lib/vim/types.ts` (NEW — binding/mode/dispatch types)
  - `fe/frontend/src/lib/vim/wails-keys.ts` (NEW — Wails-aware keypress filter for OS-level keys)
  - `fe/frontend/src/lib/vim/palette.ts` (NEW — command palette backed by product_extensions.tillsyn.commands)
  - `fe/app.go` (NEW — Go-side service bindings exposed to Wails IPC)
- **Packages:** `fe` (NEW — separate `fe/go.mod` module importing main module via `replace` directive; standard Wails v2 pattern)
- **Scope:**
  Bootstrap the Wails v2 desktop app per REVISION_BRIEF §2.15 + SKETCH §5. All Tillsyn-specific FE components carry `// MIGRATION TARGET: @hylla/stil-solid` doc-comment (EXTRACT-R2 tracked). Size-adaptive CSS from day 1 (container queries + responsive units per SKETCH §5.6).

  **Go layer (Wails bindings):**
  - `fe/main.go` — Wails `Run()` entry point, `*app.Service` bindings via `wails.Bind()`.
  - Wails IPC exposes: `ListProjects`, `GetProject`, `ListActionItems`, `CreateActionItem`, `RunDispatcher(actionItemID string)`, `GetAgentsConfig`, `GetTemplateConfig`.

  **Frontend (Astro + Solid):**
  - Astro pages: project list, project detail + action item tree, settings panel.
  - SolidJS islands for interactive components: action item create dialog, dispatcher trigger button, spawn output viewer, settings editor.
  - stil tokens consumed from `stil/main/src/styles/tokens.css` (the source-of-truth path). `dist/tokens.css` does NOT exist pre-build per `stil/main/package.json`'s `pnpm build:tokens` step (which produces `dist/tokens.json`, not `dist/tokens.css`). Consuming `src/` directly avoids requiring a stil build pre-step. When stil-solid publishes as a pnpm package, switch to the linked artifact. Per REVISION_BRIEF §2.15 (R3-NIT7 disposition) + SKETCH §10 "Stil tokens consumption path" row.

  **Vim keybinding engine (REVISION_BRIEF §2.15 + SKETCH §10):**
  - `engine.ts` — TS-side vim engine. Consumes stil `baseline.json` AND the Tillsyn-local `.tillsyn/bindings.json` (§2.19) at startup. Dispatches key events through mode state machine. Graceful fallback when bindings.json absent (empty extension table). Migration marker: `// MIGRATION TARGET: github.com/hylla-org/ro-vim`.
  - `types.ts` — TypeScript types for bindings, modes, dispatch handlers.
  - `wails-keys.ts` — Wails-aware keypress filter. Runs at document level inside the WebView. Filters OS-level keys (Cmd+Q quit, Cmd+M minimize, Cmd+W close window, Cmd+H hide) so OS/Wails default menu handles them. Passes everything else to `engine.ts`. Migration marker: `// MIGRATION TARGET: github.com/hylla-org/ro-vim`.
  - `palette.ts` — command palette (`:` command mode) backed by the union of stil baseline's `product_extensions.tillsyn.commands` (4 commands: `new-drop`, `complete-drop`, `handoff`, `comment`) plus local `.tillsyn/bindings.json` additions (5 commands: `dispatch`, `plan`, `archive`, `settings`, `help`). v1 total: 9 commands (R3-FF2 disposition — ID-based deep merge). Migration marker: `// MIGRATION TARGET: github.com/hylla-org/ro-vim`.
  - Vitest unit tests for engine + wails-keys filter.
  - Playwright (via MCP `mcp__plugin_playwright_playwright__*`) for end-to-end keybinding tests in dev mode.

  **Wails native menu (REVISION_BRIEF §2.15):** `fe/main.go` uses the DEFAULT Wails v2 native menu (Quit, About, Hide, Minimize, Window controls). No custom menu items in v1. NATIVE-MENU-R1 tracks future vim-command-dispatch integration with the native menu.

  **CI gate decision (R2-NIT1 resolution — "added" is authoritative):** A separate `mage ci-fe` target IS ADDED to `magefile.go` in W6 covering `fe/frontend` build + Vitest runs. `fe/` is excluded from the main `mage ci` target pre-MVP. Dev runs `mage ci-fe` manually during FE development. The exact scope of what `mage ci-fe` runs (which Vitest suites, whether Playwright runs) is L2-decided. Go tests for `fe/main.go` + `fe/app.go` ARE covered by `mage test-pkg ./fe/...` if the root `mage ci` is extended — L2 planner decides and documents.

  **v1 surfaces (SKETCH §5.3):**
  - Project list page — table with archived filter, create button.
  - Project detail / action item tree — collapsible tree left pane, detail right pane.
  - Action item create dialog — kind picker, paths input, description editor.
  - Dispatcher trigger button — per action item, "Run" → `RunDispatcher` Wails IPC.
  - Spawn output viewer — live tail of subagent output (uses Wails event streaming or polling).
  - Settings panel — view/edit agents.toml, view template.toml, manage groups.
- **Acceptance (L1 contract; L2 plan refines):**
  - `wails dev` in `fe/` launches Tillsyn desktop app showing project list (acceptance criterion 5.10).
  - stil tokens load and brand is consistent (Inter / JetBrains Mono fonts load from stil).
  - Project list page renders with real data from Wails IPC → `(*Service).ListProjects`.
  - Action item create dialog submits via Wails IPC → `(*Service).CreateActionItem`.
  - Every Tillsyn-specific FE component file has `// MIGRATION TARGET: @hylla/stil-solid` doc-comment.
  - All vim engine files (`engine.ts`, `wails-keys.ts`, `palette.ts`) have `// MIGRATION TARGET: github.com/hylla-org/ro-vim` doc-comment.
  - Vim engine implements ID-based deep merge: loads baseline's `product_extensions.tillsyn.commands` (4), merges local's 5 additions; falls back gracefully to baseline-only when `.tillsyn/bindings.json` absent. Command palette: 9 commands when local present, 4 when absent.
  - `wails-keys.ts` blocks OS-level keys (Cmd+Q, Cmd+M, Cmd+W, Cmd+H) from reaching engine.ts.
  - Vitest unit tests for component logic + vim engine pass (run via `mage ci-fe`).
  - Playwright (via MCP) test covers at least one user-flow interaction (e.g. project list navigation).
  - `mage ci-fe` target exists in `magefile.go`; runs at minimum `pnpm run test` + `pnpm run build` in `fe/frontend/`.
  - `mage ci` green (Go tests pass; `fe/` excluded from main `mage ci` per pre-MVP decision).
- **Blocked by:** — (Wave A head; no blockers)
- **Source-of-truth Specify:** REVISION_BRIEF §2.15; SKETCH §5; SKETCH §10 vim keybinding FE row.
- **L2 sub-planner spawn directive:** "Decompose W6 into atomic droplets. CRITICAL pre-planning question: The `fe/go.mod` separate-module approach (standard Wails v2 pattern) is the confirmed decision. L2 planner confirms Wails v2 project layout via Context7 before authoring droplets. `mage ci-fe` target: add a new `magefile.go` target that runs `pnpm run test` + `pnpm run build` in `fe/frontend/`. The target IS added in W6 (R2-NIT1 resolution — not deferred). Exact scope of what runs in `mage ci-fe` (which Vitest suites, whether Playwright runs) is decided at L2. Likely droplet shape: D1 `fe/go.mod` + `fe/main.go` (with DEFAULT Wails native menu) + `fe/wails.json` + Wails `Run()` wiring (Wails bootstrap — the skeleton that `wails dev` needs; separate go.mod with replace directive pointing to `../`); D2 Go service bindings (`fe/app.go` — `ListProjects`, `CreateActionItem`, `RunDispatcher`); D3 Astro + Solid dev setup (`fe/frontend/package.json`, `astro.config.mjs`, stil tokens symlink/copy from `stil/main/src/styles/tokens.css` → `fe/frontend/public/stil-tokens.css`) + `mage ci-fe` target in `magefile.go` (R3-NIT7: src/styles/tokens.css is the correct source path; `dist/tokens.css` does NOT exist pre-build); D4 project list page (Astro page + SolidJS island + Wails IPC call); D5 project detail + action item tree; D6 action item create dialog; D7 dispatcher trigger + spawn output viewer; D8 settings panel; D9 vim engine (`engine.ts` + `types.ts` + `wails-keys.ts` + `palette.ts` + Vitest tests + Playwright test; `palette.ts` implements ID-based deep merge: baseline's 4 commands + local's 5 = 9 total; R3-FF2 disposition). Wire `blocked_by`: D1 first; D2 after D1; D3 parallel to D1/D2 (pure frontend setup); D4-D8 each blocked by D2 + D3; D9 blocked by D3 (needs frontend dev environment established). Every FE component file must have `// MIGRATION TARGET: @hylla/stil-solid` in its JS/TS doc comment. Vim engine files get `// MIGRATION TARGET: github.com/hylla-org/ro-vim`. Playwright (via MCP `mcp__plugin_playwright_playwright__*`): FE QA agents use `browser_snapshot` for semantic checks + `browser_take_screenshot` for visual checks — no dev-side screenshot capture required. Playwright test: navigate to project list page, verify at least one project appears in accessibility tree."

---

### Wave W7 — Cleanup (INVERTED CARVING — 4 droplets)

**Round 4 pattern discipline**: inverted carving. Specify the residue (HTTP-specific bits); extract everything else first. Three consecutive rounds found missed dependencies by enumerating from the deletion side. W7.D1 inverts: classify FIRST, then extract everything-not-HTTP in W7.D2, then delete only the classified residue in W7.D3. `mage ci` green between every step.

#### 4c.6.1.W7.D1 — INVENTORY: audit `internal/adapters/server/`, classify every file/symbol, produce consumer map

- **State:** todo
- **Kind:** `build` (atomic droplet; `Irreducible: true`)
- **Paths:**
  - `internal/adapters/server/` (READ-ONLY — audit all files; NO modifications)
  - `cmd/till/` (READ-ONLY — consumer enumeration via `git grep -n`; NO modifications)
  - `internal/` (READ-ONLY — additional consumer search; NO modifications)
  - `workflow/drop_4c_6_1/W7_INVENTORY.md` (NEW — output artifact; load-bearing for W7.D2 + W7.D3)
- **Packages:** none (no code changes; `mage ci` trivially green)
- **Acceptance:**
  - `workflow/drop_4c_6_1/W7_INVENTORY.md` exists and contains three categorized lists:
    - **`http-residue`**: HTTP server transport / handler / wire-protocol-specific code. Stays until W7.D3 deletion.
    - **`stdio-relevant`**: stdio MCP transport code. Extracts to `internal/adapters/mcp_stdio/` in W7.D2.
    - **`transport-neutral`**: shared scaffolding (Service adapter, auth helpers, MCP RPC handlers like `ServeStdio`, MCP types, `mcpapi/` package). Extracts to `internal/adapters/mcp_common/` (Service adapter + auth) or `internal/adapters/mcp_rpc/` (MCP RPC engine, the current `mcpapi/` package) in W7.D2.
  - Every exported symbol in every file under `internal/adapters/server/` (including `mcpapi/`) is assigned to exactly one category.
  - The consumer map lists EVERY file that imports any `internal/adapters/server/...` package, with file:line citations. Builder uses `git grep -n "internal/adapters/server/" cmd/till/ internal/ *_test.go` exhaustively — NO consumer is left undocumented.
  - `mcpapi/` package is explicitly classified (transport-neutral — it is the MCP RPC tool registry shared by all transports).
  - `mage ci` GREEN — no code touched.
- **Blocked by:** — (Wave A head; no blockers)
- **Specify:**
  - **Objective:** Produce an exhaustive classification of every file and exported symbol in `internal/adapters/server/` before any code changes. Three rounds of plan-QA found a new missed dependency each time because the deletion target was over-specified; the inventory approach inverts the problem — classify completely first, then extract/delete based on classification.
  - **AcceptanceCriteria:** see Acceptance bullets above.
  - **ValidationPlan:** `mage ci` (no code changes; trivially green). Inventory completeness verified by W7.D2 builder reading W7_INVENTORY.md and confirming all target packages exist.
  - **RiskNotes:**
    - `mcpapi/` is 16K+ LOC. Classification may split files: some `mcpapi/` files may be truly HTTP-only; others may be shared. Builder reads each file in `mcpapi/` before classifying.
    - Use `git grep -n "\"internal/adapters/server"` (with escaped quotes) to catch all import variations including aliased imports.
    - `cmd/till/main.go` has historically had import aliases: `serveradapter "internal/adapters/server"`, `servercommon "internal/adapters/server/common"`. Enumerate ALL aliases.
    - LSP `findReferences` on every EXPORTED FUNCTION in `internal/adapters/server/` + `internal/adapters/server/mcpapi/` + `internal/adapters/server/common/` as belt-and-suspenders after `git grep`.
  - **ContextBlocks:**
    - `constraint` (critical): NO CODE CHANGES in W7.D1. Inventory is a pure-read artifact. Builder who edits any Go file in W7.D1 scope has exceeded their mandate — stop and return.
    - `constraint` (critical): Every exported symbol must be classified. If a symbol has BOTH HTTP consumers AND stdio consumers, it belongs in `transport-neutral` (extract) NOT `http-residue` (delete).
    - `decision` (normal): inverted carving per R3-FF1 disposition. Prior 2-step and 3-step approaches missed `common/` then `mcpapi/` by specifying from the deletion side. Inventory eliminates the pattern.
    - `warning` (high): `mcpapi/` imports `common/` (`servercommon`) internally. The classification must account for intra-package dependencies — if `mcpapi/` is classified transport-neutral, any `httpapi/`-only sub-file within mcpapi must be called out explicitly.
    - `reference` (normal): REVISION_BRIEF §2.16 (round 3 updated — inverted carving discipline). R3-FF1 finding: `RunStdio` at server.go:122 calls `mcpapi.ServeStdio`; mcpapi/ is 16K LOC.
  - **KindPayload:** `{"changes":[{"file":"workflow/drop_4c_6_1/W7_INVENTORY.md","symbol":"inventory artifact","action":"add","shape_hint":"three-section MD: http-residue list, stdio-relevant list, transport-neutral list; each entry has file path + exported symbols + classification rationale; consumer map with file:line citations for every importer; new, not yet in tree"}]}`

#### 4c.6.1.W7.D2 — EXTRACT EVERYTHING-NOT-HTTP: per W7.D1 inventory, move all non-residue code to new packages

- **State:** todo
- **Kind:** `build` (atomic droplet; `Irreducible: true`)
- **Paths:**
  - `internal/adapters/mcp_stdio/` (NEW package — receives stdio-relevant files per W7.D1 inventory)
  - `internal/adapters/mcp_common/` (NEW package — receives transport-neutral Service adapter + auth per W7.D1 inventory)
  - `internal/adapters/mcp_rpc/` (NEW package — receives `mcpapi/` MCP RPC tool registry per W7.D1 inventory)
  - `internal/adapters/server/` (MODIFY only — remove extracted symbols; http-residue stays)
  - `cmd/till/main.go` (MODIFY — update ALL importers per consumer map from W7.D1)
  - `cmd/till/main_test.go` (MODIFY — update all test-site imports per consumer map from W7.D1)
  - (any other consumers enumerated in W7_INVENTORY.md)
- **Packages:** `internal/adapters/mcp_stdio` (NEW), `internal/adapters/mcp_common` (NEW), `internal/adapters/mcp_rpc` (NEW), `internal/adapters/server`, `cmd/till`
- **Acceptance:**
  - Every file/symbol classified as `stdio-relevant` in W7_INVENTORY.md exists in `internal/adapters/mcp_stdio/`.
  - Every file/symbol classified as `transport-neutral` in W7_INVENTORY.md exists in `internal/adapters/mcp_common/` or `internal/adapters/mcp_rpc/` per the inventory's package assignment.
  - Every consumer in the W7.D1 consumer map has been updated to import from the new packages.
  - `till mcp` still works post-extraction.
  - `till capture-state` still works post-extraction.
  - Auth-mutation tests in `cmd/till/main_test.go` still pass.
  - `internal/adapters/server/` contains ONLY http-residue after extraction (W7.D3 will delete it).
  - `mage ci` GREEN. No test regression.
- **Blocked by:** 4c.6.1.W7.D1 (consumer map from inventory is the extraction spec; W7.D2 builder reads W7_INVENTORY.md as their primary input)
- **Specify:**
  - **Objective:** Move every file/symbol classified as non-HTTP-residue from `internal/adapters/server/` to purpose-built new packages, updating ALL importers per the W7.D1 inventory's consumer map. After this droplet, `internal/adapters/server/` contains ONLY http-residue — the safe deletion target for W7.D3.
  - **AcceptanceCriteria:** see Acceptance bullets above.
  - **ValidationPlan:** `git grep "internal/adapters/server/"` on each extracted symbol — should show zero hits or only http-residue consumers. `mage test-pkg ./internal/adapters/mcp_stdio`; `mage test-pkg ./internal/adapters/mcp_common`; `mage test-pkg ./internal/adapters/mcp_rpc`; `mage test-pkg ./cmd/till/...`; `mage ci`.
  - **RiskNotes:**
    - Builder's PRIMARY INPUT is `workflow/drop_4c_6_1/W7_INVENTORY.md`. Do NOT guess or re-enumerate — consume the W7.D1 artifact directly.
    - `mcpapi/` (classified as `mcp_rpc/` per inventory) has internal imports of `common/` package (now `mcp_common/`). Update intra-package imports when moving `mcpapi/` → `mcp_rpc/`.
    - `cmd/till/main.go` import aliases must be updated: `serveradapter` → split into appropriate new packages per consumer map; `servercommon` → `mcp_common`.
    - If inventory reveals any HTTP-only content INSIDE a file that is otherwise transport-neutral, builder must split the file (separate HTTP-only functions) before extracting the rest. Document the split in completion notes.
    - This droplet is larger than a standard atomic droplet (multi-file package move). Justified as one droplet because the work is purely mechanical import-path changes per an explicit inventory; no new business logic.
  - **ContextBlocks:**
    - `constraint` (critical): `till mcp` + `till capture-state` + auth-mutation tests MUST pass after W7.D2. If any break, stop and return to orchestrator.
    - `constraint` (critical): do NOT delete `internal/adapters/server/` in W7.D2. That is W7.D3's job. Leave http-residue in place.
    - `constraint` (high): use `git mv` (not copy+delete) where possible to preserve git history on moved files.
    - `decision` (normal): three new packages — `mcp_stdio/` (transport), `mcp_common/` (shared scaffolding), `mcp_rpc/` (RPC tool registry). Future TILL-SERVE-R1 HTTP rebuild plugs into `mcp_rpc/` via a new HTTP transport without re-extraction.
    - `reference` (normal): REVISION_BRIEF §2.16 round-3 inverted carving discipline. W7_INVENTORY.md is the authoritative input.
  - **KindPayload:** `{"changes":[{"file":"internal/adapters/mcp_stdio/","symbol":"stdio-relevant files per inventory","action":"add","shape_hint":"move from internal/adapters/server/; new package; not yet in tree"},{"file":"internal/adapters/mcp_common/","symbol":"transport-neutral service adapter + auth per inventory","action":"add","shape_hint":"move from internal/adapters/server/common/; new package; not yet in tree"},{"file":"internal/adapters/mcp_rpc/","symbol":"mcpapi/ MCP RPC tool registry per inventory","action":"add","shape_hint":"move from internal/adapters/server/mcpapi/; new package; update import path in all moved files + in mcp_stdio/; not yet in tree"},{"file":"internal/adapters/server/","symbol":"extracted symbols removed","action":"modify","shape_hint":"remove extracted code; only http-residue remains"},{"file":"cmd/till/main.go","symbol":"all importers per consumer map","action":"modify","shape_hint":"update ALL serveradapter/servercommon import aliases to new packages per W7_INVENTORY.md consumer map"},{"file":"cmd/till/main_test.go","symbol":"all test-site imports per consumer map","action":"modify","shape_hint":"update ALL server/* import references to new packages per W7_INVENTORY.md"}]}`

#### 4c.6.1.W7.D3 — DELETE HTTP RESIDUE: remove what's left in `internal/adapters/server/` + `till serve` CLI

- **State:** todo
- **Kind:** `build` (atomic droplet; `Irreducible: true`)
- **Paths:**
  - `internal/adapters/server/` (DELETE — http-residue only; everything else extracted by W7.D2)
  - `cmd/till/main.go` (MODIFY — remove `till serve` subcommand registration + `runServe` function + any remaining `internal/adapters/server/` imports)
  - `cmd/till/main_test.go` (MODIFY — remove serve-specific tests ONLY; auth-mutation tests already migrated in W7.D2 and must NOT be removed)
  - `cmd/till/help.go` (MODIFY if present — remove `till serve` from help text)
- **Packages:** `cmd/till`, `internal/adapters/server` (deleted)
- **Acceptance:**
  - `internal/adapters/server/` directory does NOT exist post-deletion (all http-residue gone; all useful content extracted by W7.D2).
  - `git grep "internal/adapters/server"` returns ZERO hits in Go source files — belt-and-suspenders confirmation that extraction in W7.D2 was complete.
  - `till serve` does NOT appear in `till --help` output.
  - `till mcp` STILL WORKS.
  - `till capture-state` STILL WORKS.
  - Auth-mutation tests in `cmd/till/main_test.go` STILL PASS (migrated in W7.D2 — not removed here).
  - `mage ci` GREEN — failure here surfaces any missed extraction in W7.D2 (mandatory belt-and-suspenders check per inverted carving discipline).
- **Blocked by:** 4c.6.1.W7.D2 (extraction must be complete before deletion); 4c.6.1.W2 (`cmd/till` package compile lock — W2 modifies `init_cmd.go`, W7.D3 modifies `main.go`; serialize to avoid package-compile race)
- **Specify:**
  - **Objective:** Delete the HTTP-residue that remains in `internal/adapters/server/` after W7.D2's extraction, and remove the `till serve` CLI subcommand. The post-deletion `mage ci` run is the mandatory belt-and-suspenders check: if it fails, W7.D2 missed an extraction and W7.D3's builder returns to the orchestrator.
  - **AcceptanceCriteria:** see Acceptance bullets above.
  - **ValidationPlan:** `git grep "internal/adapters/server"`; `mage test-pkg ./cmd/till/...`; `mage ci`.
  - **RiskNotes:**
    - Before deleting, builder reads the actual contents of `internal/adapters/server/` POST-W7.D2 (via `Read` + `ls`) to confirm only http-residue remains. If anything non-http is found, stop and return to orchestrator — W7.D2 missed it.
    - `cmd/till/main_test.go`: do NOT remove auth-mutation tests — only serve-specific tests (those testing the `till serve` cobra command and HTTP handler registration).
    - TILL-SERVE-R1 refinement tracks future HTTP/MCP server rebuild from scratch.
  - **ContextBlocks:**
    - `constraint` (critical): `mage ci` failure after W7.D3 deletion = W7.D2 missed an extraction. STOP. Return to orchestrator. Do NOT fight the build failure by patching.
    - `constraint` (critical): do NOT remove auth-mutation tests from main_test.go. Only serve-specific tests.
    - `constraint` (high): use `git rm -r` on the directory to preserve deletion history.
    - `decision` (normal): deletion is permanent pre-MVP; future HTTP/MCP server built from scratch (TILL-SERVE-R1).
    - `reference` (normal): REVISION_BRIEF §2.16 (round-3 inverted carving discipline); TILL-SERVE-R1 refinement.
  - **KindPayload:** `{"changes":[{"file":"internal/adapters/server/","symbol":"http-residue (all remaining contents)","action":"delete","shape_hint":"git rm -r after verifying only http-residue remains; mage ci green = extraction was complete"},{"file":"cmd/till/main.go","symbol":"serve subcommand + runServe + remaining server imports","action":"modify","shape_hint":"remove cobra serve subcommand registration + runServe function + any remaining internal/adapters/server imports"},{"file":"cmd/till/main_test.go","symbol":"serve-specific tests only","action":"modify","shape_hint":"remove ONLY tests for till serve; auth-mutation tests NOT removed"},{"file":"cmd/till/help.go","symbol":"till serve help","action":"modify","shape_hint":"remove till serve reference if present; verify via Read first"}]}`

#### 4c.6.1.W7.D4 — CLAUDE.md cascade table corrections (renumbered from old W7.D3)

- **State:** todo
- **Kind:** `build` (atomic droplet; doc-only — `Irreducible: true`)
- **Paths:** `CLAUDE.md` (MODIFY — cascade table in "Agent Bindings" section and "Claude Code Agents" section)
- **Packages:** none (doc-only)
- **Acceptance:**
  - Cascade table in `CLAUDE.md` § "Agent Bindings" (and § "Claude Code Agents") updated:
    - Drop `go-` prefix from all agent names (e.g. `go-builder-agent` → `builder-agent`, `go-planning-agent` → `planning-agent`).
    - Add 4 separate rows for QA agents: `plan-qa-proof`, `plan-qa-falsification`, `build-qa-proof`, `build-qa-falsification` (currently 2 rows for qa-proof + qa-falsification).
    - Document 10-agent-per-group set in `CLAUDE.md` § "Claude Code Agents" table (9 standard + `orchestrator-managed.md` special).
    - Update any kind → agent binding rows that referenced the old `go-*` prefix names.
  - `mage ci` green (no Go code changed).
- **Blocked by:** 4c.6.1.W4.D1 (CLAUDE.md table must reference confirmed new agent names from W4.D1 — can't update the table to names that don't exist in the embedded agent files yet)
- **Specify:**
  - **Objective:** Keep CLAUDE.md's cascade table accurate to the shipped 10-agent set. Agents table is load-bearing documentation — orchestrators read it to determine which `subagent_type` to use when spawning agents.
  - **AcceptanceCriteria:** see Acceptance bullets above.
  - **ValidationPlan:** doc review; `git grep 'go-builder-agent\|go-planning-agent\|go-qa-proof-agent\|go-qa-falsification-agent'` returns zero hits in CLAUDE.md post-edit.
  - **RiskNotes:**
    - CLAUDE.md appears in MULTIPLE places in the repo (bare root + main/ worktree per `feedback_bare_root_not_tracked.md`). The builder must edit `main/CLAUDE.md` (tracked). The bare-root `CLAUDE.md` requires manual parity edit by the dev (not tracked by git) — note this in the completion notes.
    - Two different tables in CLAUDE.md reference agent names: the `Agent Bindings` table and the `Claude Code Agents` table. Both must be updated.
    - Agent count per group is now 10 (9 standard + 1 special `orchestrator-managed.md`). The table must reflect this, not just 9.
  - **ContextBlocks:**
    - `constraint` (high): CLAUDE.md agent table is the canonical reference orchestrators use for `subagent_type` dispatch.
    - `warning` (normal): bare-root CLAUDE.md is NOT git-tracked; dev must manually apply the same edits there.
    - `reference` (normal): REVISION_BRIEF §2.13 + SKETCH §3.2 carry the model-per-kind assignment that the updated table must reflect.
  - **KindPayload:** `{"changes":[{"file":"CLAUDE.md","symbol":"Claude Code Agents table + Agent Bindings table","action":"modify","shape_hint":"drop go- prefix from all agent names; add 4 QA rows; document 10-agent set (9 standard + orchestrator-managed special)"}]}`

---

### Wave W8 — Tillsyn-Project-Local Agent Prompts + Bindings

#### 4c.6.1.W8 — sub-plan container

- **State:** todo
- **Kind:** `plan` (sub-plan container; spawns its own L2 planner)
- **Directory:** `workflow/drop_4c_6_1/DROP_4c.6.1.W8_TILLSYN_PROJECT_PROMPTS/`
- **Paths (expected):** (all paths are working-dir relative from `tillsyn/main/`)
  - `.tillsyn/agents/go/planning-agent.md` (NEW)
  - `.tillsyn/agents/go/builder-agent.md` (NEW)
  - `.tillsyn/agents/go/plan-qa-proof-agent.md` (NEW)
  - `.tillsyn/agents/go/plan-qa-falsification-agent.md` (NEW)
  - `.tillsyn/agents/go/build-qa-proof-agent.md` (NEW)
  - `.tillsyn/agents/go/build-qa-falsification-agent.md` (NEW)
  - `.tillsyn/agents/go/research-agent.md` (NEW)
  - `.tillsyn/agents/go/closeout-agent.md` (NEW)
  - `.tillsyn/agents/go/commit-message-agent.md` (NEW)
  - `.tillsyn/agents/go/orchestrator-managed.md` (NEW)
  - `.tillsyn/agents/fe/planning-agent.md` (NEW)
  - `.tillsyn/agents/fe/builder-agent.md` (NEW)
  - `.tillsyn/agents/fe/plan-qa-proof-agent.md` (NEW)
  - `.tillsyn/agents/fe/plan-qa-falsification-agent.md` (NEW)
  - `.tillsyn/agents/fe/build-qa-proof-agent.md` (NEW)
  - `.tillsyn/agents/fe/build-qa-falsification-agent.md` (NEW)
  - `.tillsyn/agents/fe/research-agent.md` (NEW)
  - `.tillsyn/agents/fe/closeout-agent.md` (NEW)
  - `.tillsyn/agents/fe/commit-message-agent.md` (NEW)
  - `.tillsyn/agents/fe/orchestrator-managed.md` (NEW)
  - `.tillsyn/bindings.json` (NEW — Tillsyn-local vim bindings extension)
  - `.gitignore` (MODIFY — re-include `.tillsyn/agents/**/*.md` + `.tillsyn/bindings.json`)
- **Packages:** none (all non-Go files — Hylla does not index these; no Go compile unit touched)
- **Scope:**
  Author substantive Tillsyn-aware agent prompts for Tillsyn's own project work. These prompts are the project-local (tier-1) override of the 3-tier resolver — they encode mage discipline, Section 0 reasoning, MD-only workflow mode, plan-down/build-up, atomic-droplet sizing, Hylla usage, CONSUMER-TIE test contract, and QA disciplines specific to the Tillsyn project.

  **Per disposition 7.6: SKIP `gen/` group** — Tillsyn's own work is `go` + `fe` only.

  **Source material per prompt** (L2 sub-planner and builders consult these):
  - `~/.claude/agents/<group>-<role>-agent.md` — dev's system agents (production-grade starting point).
  - `main/CLAUDE.md` — cascade tree, agent bindings, build discipline.
  - `workflow/example/drops/WORKFLOW.md` — methodology.
  - `WIKI.md` — cascade vocabulary, structural_type axis.
  - Memory entries: `feedback_plan_down_build_up.md`, `feedback_decomp_small_parallel_plans.md`, `feedback_subagents_background_default.md`, `feedback_section_0_required.md`, `feedback_hylla_go_code.md`, `feedback_cascade_model_policy.md`, `feedback_use_typed_agents.md`, `feedback_commit_style.md`, `feedback_tool_discipline_native_tools.md`.
  - Drop 4c.6 + Drop 4c.6.1 worklog patterns (CONSUMER-TIE, atomic-droplet sizing, plan-QA asymmetry, single-line conventional commits).

  **Tillsyn-local bindings file** (REVISION_BRIEF §2.19 — R3-FF2 disposition):
  - `.tillsyn/bindings.json` — ID-based deep merge with stil's baseline.json `product_extensions.tillsyn` block. Stil's baseline ALREADY ships 4 commands (`new-drop`, `complete-drop`, `handoff`, `comment`). This file adds 5 NEW commands with unique IDs: `dispatch`, `plan`, `archive`, `settings`, `help`. Original `close` DROPPED (redundant with stil's canonical `complete-drop`). No ID collision between baseline's 4 and local's 5; union = 9 commands.
  - **Merge semantic (R3-FF2 disposition)**: ID-based deep merge. W5 + W6 loaders union baseline's `product_extensions.tillsyn.commands` with local file's `product_extensions.tillsyn.commands` by `id`. Local wins on collision. Absent local file = baseline-only (4 commands). No fail-loud on absent file.
  - Consumed by W5 (TUI keybinding dispatcher) + W6 (FE vim engine) at runtime. Both surfaces handle graceful fallback when this file absent.

  **.gitignore re-includes** (REVISION_BRIEF §2.20):
  - Add `!.tillsyn/agents/` + `!.tillsyn/agents/**/*.md` + `!.tillsyn/bindings.json` alongside existing `!.tillsyn/template.toml` re-include.
  - Runtime state files (`.tillsyn/config.toml`, `.tillsyn/tillsyn.db*`, `.tillsyn/logs/`, `.tillsyn/livewait.secret`) stay ignored.

  **Migration markers per prompt**: each prompt carries a doc-comment note at the top —
  `<!-- Tillsyn-project-local; lifted from ~/.claude/agents/ and adapted for Tillsyn's workflow. Future projects use embedded defaults shipped in Drop 4c.8. -->`.
- **Acceptance (L1 contract; L2 plan refines):**
  - All 20 prompt files exist (10 go + 10 fe) with non-stub bodies (>= 1000 chars each).
  - Each prompt has frontmatter: `name`, `description`, `model` (per cascade-model-policy), `tools` (per role).
  - Body encodes the role's Tillsyn-specific discipline (mage targets, Section 0, plan-down/build-up, etc.).
  - Each prompt passes the post-render validator shape (per Drop 4c.6 W3.D5 validator) — frontmatter complete, body non-empty.
  - No Section 0 leakage in any committed prompt file.
  - `.tillsyn/bindings.json` exists with `schema_version: 1`, `product_extensions.tillsyn.commands` containing 5 entries (the Tillsyn-local ADDITIONS only: `dispatch`, `plan`, `archive`, `settings`, `help`; original `close` absent — redundant with stil's `complete-drop`).
  - `.gitignore` re-includes `.tillsyn/agents/**/*.md` + `.tillsyn/bindings.json` alongside existing `!.tillsyn/template.toml`.
  - `git ls-files .tillsyn/agents/` shows 20 tracked files (working-dir-relative path from `tillsyn/main/`).
  - `git ls-files .tillsyn/bindings.json` shows 1 tracked file.
  - **Integration smoke (R3-NIT4 absorption)**: at least one W8-authored prompt (e.g., `.tillsyn/agents/go/builder-agent.md`) is rendered through `internal/app/dispatcher/cli_claude/render/render.go:assembleAgentFileBody` with project-tier override, producing a body identical to the W8-authored file (NOT the embedded default). This is a unit test, NOT a full dispatch — verifies the 3-tier resolver picks up project-tier prompts correctly. Test lives in W8 L2 (suggested file: `internal/app/dispatcher/cli_claude/render/render_test.go` or a new test in W8's per-prompt-set test suite). (smoke-test droplet blocked_by W1; see L2 spawn directive) [New, not yet in tree — W8 authors it.]
- **Blocked by:** — (Wave A head; no blockers — paths are entirely disjoint from all other waves)
- **Source-of-truth Specify:** REVISION_BRIEF §2.18 (prompts), §2.19 (bindings.json), §2.20 (.gitignore); SKETCH §10 Tillsyn-project-local row.
- **L2 sub-planner spawn directive:** "Decompose W8 into atomic droplets. IMPORTANT ordering: D0 `.gitignore` update + `.tillsyn/bindings.json` authoring FIRST (working-dir-relative paths: `.gitignore` and `.tillsyn/bindings.json` from `tillsyn/main/`). D0 makes the subsequent .md files trackable by git and MUST be committed before any prompt-file droplets so `git ls-files .tillsyn/agents/` confirms tracking. Prompt droplets (one per file, no batching): D1 `go/planning-agent.md`; D2 `go/builder-agent.md`; D3 `go/plan-qa-proof-agent.md`; D4 `go/plan-qa-falsification-agent.md`; D5 `go/build-qa-proof-agent.md`; D6 `go/build-qa-falsification-agent.md`; D7 `go/research-agent.md`; D8 `go/closeout-agent.md`; D9 `go/commit-message-agent.md`; D10 `go/orchestrator-managed.md`; D11–D20 same shape for `fe/` group (D11 `fe/planning-agent.md`; D12 `fe/builder-agent.md`; D13 `fe/plan-qa-proof-agent.md`; D14 `fe/plan-qa-falsification-agent.md`; D15 `fe/build-qa-proof-agent.md`; D16 `fe/build-qa-falsification-agent.md`; D17 `fe/research-agent.md`; D18 `fe/closeout-agent.md`; D19 `fe/commit-message-agent.md`; D20 `fe/orchestrator-managed.md`). Parallelism: D0 first (blocks all D1-D20); D1-D10 (go group) parallel with each other after D0; D11-D20 (fe group) parallel with each other after D0; go group parallel with fe group (disjoint files). Per-droplet QA pair (build-qa-proof + build-qa-falsification) runs after each prompt droplet: Proof verifies >= 1000 chars + frontmatter complete + role discipline encoded; Falsification attacks: can a builder reading this prompt go wrong despite following it?

**R3-NIT5 GUIDANCE (CRITICAL — builder must read before authoring):**

Source material notes:
- `~/.claude/agents/<group>-<role>-agent.md` is the PRIMARY starting point for prompts WHERE THE FILE EXISTS. Copy and adapt, do NOT write from scratch when a source file exists.
- **6 of 20 prompts have NO `~/.claude/agents/` source file**: `go/closeout-agent.md`, `go/commit-message-agent.md`, `go/orchestrator-managed.md`, `fe/closeout-agent.md`, `fe/commit-message-agent.md`, `fe/orchestrator-managed.md`. These 6 MUST be authored FROM SCRATCH citing: project `CLAUDE.md` (cascade tree, orchestrator-managed role semantics, closeout/refinement/discussion/human-verify kind bindings), `workflow/example/drops/WORKFLOW.md` (phase structure, closeout discipline), `WIKI.md` (cascade vocabulary), memory entries for orchestrator role.
- **Plan-QA vs Build-QA MUST be differentiated**: `go-qa-proof-agent.md` in `~/.claude/agents/` is a single file that seeds BOTH `go/plan-qa-proof-agent.md` AND `go/build-qa-proof-agent.md`. These MUST NOT be near-identical copies. Per SKETCH §3.1 (different work, different evidence sources):
  - `plan-qa-proof-agent.md`: verifies PLAN.md droplet decomposition — blocked_by graph correctness, paths/packages declarations, acceptance bullets, surface boundaries. Evidence: PLAN.md + REVISION_BRIEF.md + SKETCH.md.
  - `build-qa-proof-agent.md`: verifies actual code changes against the plan — test pass rates, no scope creep beyond declared paths, evidence for each acceptance bullet. Evidence: Go source + test output + git diff.
  - Same asymmetry applies to `plan-qa-falsification-agent.md` vs `build-qa-falsification-agent.md`.
- QA-SPLIT-R1 tracks proper further differentiation in Drop 4c.8; this drop's 4 QA files must at minimum have visibly different 'Evidence Sources' and 'What To Check' sections.

Model assignments per cascade-model-policy: planning/plan-qa-*/build-qa-* → opus; builder → sonnet; commit-message → haiku; research → opus; closeout → orchestrator-managed. Tools per role: qa-proof/qa-falsification/research/planning → Read, Grep, Glob, Hylla; builder → Read, Edit, Write, Grep, Glob; closeout/orchestrator-managed → same as builder scope (orchestrator-managed kinds use the orchestrator's full toolset).

**R3-NIT4 smoke test requirement (REQUIRED in W8 L2) — DEDICATED D21 DROPLET**: Add a **new dedicated smoke-test droplet** (D21, after D0 bindings + D1-D20 prompts) AFTER the 20 prompt-authoring droplets. Specifically: render `.tillsyn/agents/go/builder-agent.md` (or another W8-authored prompt) through `internal/app/dispatcher/cli_claude/render/render.go:assembleAgentFileBody` with project-tier override active, and assert the rendered body matches the W8-authored file (NOT the embedded default). This is a unit test only — NOT the full end-to-end dispatcher flow (that is deferred to Drop 4c.7 per W8-SMOKE-R1). This test is new, not yet in tree.

**Cross-wave dependency note**: the smoke-test is a DEDICATED L2 droplet (separate from the 20 prompt-authoring droplets) because:
- Prompt-authoring droplets: paths `.tillsyn/agents/<group>/<name>.md`, packages: none, atomicity: file-write-only.
- Smoke-test droplet (D21): paths `internal/app/dispatcher/cli_claude/render/render_test.go` (MODIFY — add test case; or a new test file in that package), packages: `internal/app/dispatcher/cli_claude/render`, atomicity: Go test addition.

Different path + package locks; cannot live in the same droplet per atomic-droplet sizing + dispatcher lock semantics.

Smoke-test droplet `blocked_by`:
- All 20 prompt-authoring droplets (sequencing — smoke needs the prompt files written).
- `4c.6.1.W1` (resolver subdir-per-group shape — smoke exercises `assembleAgentFileBody` per the new resolver).

This makes the smoke-test droplet Wave C transitively (after W1 lands in Wave B + after the prompt droplets in Wave A). The other 20 W8 prompt droplets do NOT require the `blocked_by W1` blocker (they only AUTHOR `.md` files; they don't exercise the resolver).

The graph below is the authoritative blocked_by ordering. `→` = "blocked by."

```
Wave A (all parallel — no cross-blockers within wave):
  4c.6.1.W0      (config decoder — no blockers)
  4c.6.1.W4.D1   (agent set restructure — no blockers)
  4c.6.1.W5      (TUI components + vim keybinding dispatcher — no blockers)
  4c.6.1.W6      (FE scaffold + vim engine — no blockers)
  4c.6.1.W7.D1   (INVENTORY — pure read, no code changes, no blockers)
  4c.6.1.W8      (Tillsyn-project-local prompts + bindings.json + .gitignore — no blockers)

Wave B (after Wave A):
  4c.6.1.W1      → 4c.6.1.W4.D1   (resolver needs confirmed group subdir layout)
  4c.6.1.W4.D2   → 4c.6.1.W4.D1, 4c.6.1.W0   (agent_name bindings match W4.D1; decoder contract from W0)
  4c.6.1.W7.D2   → 4c.6.1.W7.D1   (EXTRACT: per inventory; W7.D1 consumer map is the extraction spec)
  4c.6.1.W7.D4   → 4c.6.1.W4.D1   (CLAUDE.md table cites confirmed agent names from W4.D1)

Wave C (after Wave B):
  4c.6.1.W2      → 4c.6.1.W1, 4c.6.1.W4.D1, 4c.6.1.W5
                   (template.toml write uses W1 HOME-tier path; copies W4.D1 agent subdirs; TUI uses W5 components)
  4c.6.1.W7.D3   → 4c.6.1.W7.D2, 4c.6.1.W2
                   (DELETE: W7.D2 extraction must complete; W2 serializes cmd/till package compile lock)

Wave D (after Wave C):
  4c.6.1.W3      → 4c.6.1.W2, 4c.6.1.W1
                   (cmd/till package serialization with W2; template/agents CLIs use W1 HOME-tier path contract)
```

Wave A (parallel): W0, W4.D1, W5, W6, **W7.D1 (Inventory)**, W8 (Tillsyn-project-local prompts) — 20 prompt-authoring droplets are Wave A; the 21st (smoke-test D21, `blocked_by W1`) lands at Wave C transitively.
Wave B (after Wave A completes): W1, W4.D2, W7.D2 (Extract), W7.D4 (CLAUDE.md) — parallel within wave, respecting internal deps.
Wave C (after Wave B): W2 (blocked by W1 + W4.D1 + W5) + W7.D3 (Delete-residue, blocked by W7.D2 + W2 for cmd/till compile lock) + W8.D21 smoke-test droplet (blocked by W1 + all 20 W8 prompt droplets).
Wave D (after Wave C): W3 (blocked by W2 + W1).

**Acyclicity check (topo-sort):** {W0, W4.D1, W5, W6, W7.D1, W8} → {W1, W4.D2, W7.D2, W7.D4} → {W2, W7.D3} → W3. No cycle confirmed. W8 has no downstream blockers. W7.D3 in Wave C because it depends on both W7.D2 (Wave B) and W2 (Wave C — `cmd/till` package compile serialization).

**Parallelism notes:**
- W0 (config decoder) is a small atomic droplet in Wave A — dispatches concurrently with W4.D1, W5, W6, W7.D1, W8.
- W5 (TUI + vim dispatcher) and W6 (FE + vim engine) are the largest parallel workstreams in Wave A — both can dispatch sub-planners and build concurrently.
- W4.D1 and W7.D1 are atomic droplets in Wave A — W7.D1 is now a pure-read inventory with no code changes.
- W8 (Tillsyn-project-local prompts) is a DUAL-WAVE sub-plan — 20 prompt-authoring droplets touch only `.tillsyn/` files (Wave A, parallel with everything else); 1 smoke-test droplet (D21) touches `internal/app/dispatcher/cli_claude/render/render_test.go` and is `blocked_by 4c.6.1.W1` (Wave C transitively, after W1's Wave B resolver lands). The W8 sub-plan container completion thus spans Wave A→Wave C.
- W7.D1 (Inventory) is Wave A; W7.D2 (Extract) is Wave B (blocked by D1); W7.D3 (Delete-residue) is Wave C (blocked by D2 + W2 for cmd/till package lock); W7.D4 (CLAUDE.md) is Wave B (blocked by W4.D1 for confirmed agent names).
- W7.D3 also blocked by W2 to serialize `cmd/till` package — W7.D3 modifies `cmd/till/main.go` and W2 modifies `cmd/till/init_cmd.go`; different files in the same package require a compile lock.
- W3's `till project update/delete/archive/restore`, `till action_item create`, and `till agents bootstrap` CLIs don't technically depend on W2 for service methods (they all exist). However, they share the `cmd/till` package compile unit with W2's `init_cmd.go` changes, so they serialize via W3 blocked_by W2.

---

## Notes

### Sub-plan vs direct-droplet ratio + L2 spawn cadence

L1 emits **6 sub-plan containers** (W1, W2, W3, W5, W6, W8) and **7 direct droplets** (W0, W4.D1, W4.D2, W7.D1, W7.D2, W7.D3, W7.D4) = 13 L1 nodes. The orchestrator spawns L2 sub-planners against each sub-plan when its `blocked_by` clears:
- W0 + W4.D1 + W5 + W6 + **W7.D1 (Inventory)** + **W8** dispatch immediately at drop start (Wave A — all parallel).
- W1 sub-planner spawns after W4.D1 closes (Wave B).
- W4.D2 + W7.D2 (Extract) + W7.D4 (CLAUDE.md) atomic builds after their Wave B blockers close (W7.D2 after W7.D1; W7.D4 after W4.D1).
- W2 sub-planner spawns after W1 + W5 complete (Wave C). W7.D3 (Delete-residue) spawns after W7.D2 + W2 both close (Wave C).
- W3 sub-planner spawns after W2 closes (Wave D).

L2 sub-planners author their own `workflow/drop_4c_6_1/DROP_4c.6.1.W<X>_<NAME>/PLAN.md` per `workflow/example/drops/WORKFLOW.md` § Sub-Drops.

### Locked architectural decisions (inherited from SKETCH §10 — updated for Round 4)

- Multi-group composable (NOT exclusive). 10 agents per group (9 standard + `orchestrator-managed.md` special).
- Subdir-per-group for agent files: `<project>/.tillsyn/agents/<group>/<name>.md`.
- Drop `go-` prefix; group subdir is the distinguisher.
- Plan-QA vs Build-QA split into 4 separate files per group.
- `orchestrator-managed.md` KEPT as special 10th file per group. FF3 disposition. ORCH-MANAGED-R1 tracks split in Drop 4c.8.
- FLAT layout detection: fail loud, no migration code. FF2 disposition. D7-R6 tracks manual cleanup.
- `--structural-type`: smart-default per kind (plan/refinement→segment; others→droplet); optional override validates against closed enum. FF4 disposition.
- `till serve` deletion: **INVERTED CARVING — 4-droplet sequence** (W7.D1=Inventory; W7.D2=Extract-everything-not-HTTP per inventory; W7.D3=Delete-residue with mandatory `mage ci` belt-and-suspenders check; W7.D4=CLAUDE.md update). R3-FF1 disposition. Pattern discipline: specify the residue from the deletion side, extract everything else first, then delete. TILL-SERVE-R1 tracks rebuild. W7.D3 blocked by W7.D2 AND W2 (cmd/till compile lock).
- `internal/config/agents.go` decoder: multi-group `[<group>]` / `[<group>.<kind>]` deep-merge. Proof-FF1 disposition. Must land before W4.D2.
- TUI components inline `internal/tui/components/` (EXTRACT-R1) + vim keybinding dispatcher `internal/tui/keybindings/` (KEYBIND-R1) — both migration-marker'd for lykta.
- FE inline `fe/` with separate `fe/go.mod` (standard Wails v2 pattern; no go.work) + vim engine `fe/frontend/src/lib/vim/` (KEYBIND-R2) migration-marker'd for ro-vim.
- Wails v2 native menu DEFAULT in `fe/main.go` (Quit/About/Hide/Minimize — no custom items in v1). NATIVE-MENU-R1 tracks future vim-dispatch integration.
- Wails v2 + Astro + Solid + stil tokens. NO `till serve`. Wails IPC for Go↔JS.
- `agents.toml` gets NO HOME tier (per-project runtime config only).
- Schema: `[<group>]` and `[<group>.<kind>]` — no `agents.`/`template.` prefix.
- Methodology docs deferred to Drop 4c.8 (out of scope).
- No go.work (single-repo; `fe/go.mod` uses replace directive).
- `stil-rust` adapter dropped from plans entirely.
- `fe/` excluded from `mage ci` pre-MVP; `mage ci-fe` target IS ADDED in W6 (R2-NIT1 resolution — "added" is authoritative; exact scope L2-decided).
- Vim keybindings: single source-of-truth via stil baseline.json + Tillsyn-local `.tillsyn/bindings.json` extension (per REVISION_BRIEF §2.19 R3-FF2 disposition). **Merge semantic: ID-based deep merge** — baseline's `product_extensions.tillsyn.commands` (4: `new-drop`, `complete-drop`, `handoff`, `comment`) + local's additions (5: `dispatch`, `plan`, `archive`, `settings`, `help`); local wins on collision; absent local = baseline-only (4 commands). Both W5 TUI dispatcher and W6 FE engine implement this merge with graceful fallback. **KEYBIND-R3 rewording**: when stil-solid lands, canonicalize Tillsyn's 5 local commands into stil's `product_extensions.tillsyn` as an ADDITIVE operation (baseline's 4 + local's 5 → all 9 in baseline); the local file then becomes a no-op or is deleted. This is NOT a "move" (the slot is already occupied by 4 commands in baseline).
- Tillsyn-project-local prompts: 20 prompt files at `.tillsyn/agents/{go,fe}/` (10 per group) + `.tillsyn/bindings.json` + `.gitignore` re-includes (W8). Skip `gen/` per disposition 7.6.
- `till agents bootstrap` CLI: folds into W3. 2-into-4 QA fan-out (source 2 files → dest 4 files). QA-SPLIT-R1 tracks proper per-role differentiation in Drop 4c.8.

### 5.13 dogfood end-to-end smoke — explicit deferral

REVISION_BRIEF §5.13 ("SQL-free, manual-edit-free path from `till init` → `till action_item create` → `till dispatcher run --dry-run` → spawn descriptor renders cleanly") is implicitly covered by the union of W1+W2+W3+W4.D1+W0 acceptance bullets. An explicit integration smoke-test droplet (e.g. `cmd/till/dogfood_smoke_test.go`) is deferred to Drop 4c.7 (cascade wiring) where end-to-end dispatcher flow is the primary focus. This is an explicit deferral per NIT3 from proof review. Drop 4c.7 scope should include this as its first acceptance gate.

### W4.D1 atomicity note

W4.D1 has ~25 file operations (deletes + adds + modifies). Per the L1 decomposition shape table: "structural file changes only — no semantic Go logic." No production business logic is changed; the work is mechanical directory restructuring + embed.go list updates + test updates. Justified as a single atomic droplet. If the builder finds the droplet too wide in practice during Wave A execution, the orchestrator may split at that point; no pre-split needed from L1.

### Critical W7 inverted-carving gate (R3-FF1 disposition — replaces R2-FF1 two-step)

**Pattern**: INVENTORY first, then EXTRACT everything-not-HTTP, then DELETE only the classified residue.

- W7.D1 (Wave A): pure read — classifies every file/symbol in `internal/adapters/server/` (including `mcpapi/` 16K LOC) as http-residue / stdio-relevant / transport-neutral. Produces `W7_INVENTORY.md` with consumer map. `mage ci` trivially green (no code changes).
- W7.D2 (Wave B, blocked by W7.D1): reads `W7_INVENTORY.md` as primary input. Extracts everything-not-HTTP to `mcp_stdio/`, `mcp_common/`, `mcp_rpc/` (or whatever packages the inventory specifies). Updates ALL importers per consumer map. `mage ci` GREEN after extraction.
- W7.D3 (Wave C, blocked by W7.D2 + W2): deletes the http-residue. `mage ci` failure = W7.D2 missed an extraction. Builder stops and returns to orchestrator. `mage ci` GREEN = deletion complete.
- W7.D4 (Wave B, blocked by W4.D1): CLAUDE.md cascade table corrections (doc-only).

Round-history note: R1-FF1 found `till mcp` depends on `internal/adapters/server/`; R2-FF1 found `till capture-state` depends on `internal/adapters/server/common/`; R3-FF1 found `RunStdio` depends on `internal/adapters/server/mcpapi/`. Each round patched one gap. Inverted carving eliminates the pattern at the root by exhaustively classifying before any deletion target is set.

### `~/.claude/agents/` system-agent split — explicitly out of scope

Per NIT4 from falsification: the split of `~/.claude/agents/go-qa-proof-agent.md` (single file covering plan-qa-proof + build-qa-proof for the global agent library) into separate per-kind variants is NOT in this drop's scope. This drop's plan-qa/build-qa split is template-internal only (embedded agent scaffolding at `internal/templates/builtin/agents/<group>/`). Pre-cascade orchestrator-spawned agents continue to use the single system files. The split of the global `~/.claude/agents/` files is deferred to Drop 4c.8 when substantive agent prompt content is authored.

### Out-of-scope items

Per REVISION_BRIEF §3 + SKETCH §10 decisions:
- **Cascade wiring** (state-trigger dispatch, gate runner) — Drop 4c.7.
- **Substantive agent prompts** (10×3=30 placeholder files) — Drop 4c.8 W4.
- **`till serve` rebuild** — future drop (TILL-SERVE-R1).
- **Web variant** + **Mobile (Capacitor) wrap** — later still.
- **Methodology docs** (`CASCADE_METHODOLOGY.md`, `AGENTS_CONFIG.md`, `GDD_METHODOLOGY.md`) substantive content — Drop 4c.8.
- **Hylla reset** — orthogonal one-off.
- **Extract TUI components to lykta** — REFINEMENT (post-dogfood).
- **Extract FE components to stil-solid** — REFINEMENT (post-dogfood).
- **`till-gdd` 10-agent expansion** — not mentioned in REVISION_BRIEF §2.11; till-gdd stays at 7 files; W4.D1 L2 builder verifies and documents.
- **Split `~/.claude/agents/go-qa-*.md` system files** — not this drop; template-internal split only. See note above.
- **5.13 integration smoke test** — deferred to Drop 4c.7 (see note above).
- **`mage ci-fe` full CI gate parity with `mage ci`** — target IS added in W6 but its exact scope is L2-decided. Running `mage ci-fe` as part of the main CI pipeline (auto-triggered on every push) is deferred post-MVP.

### Acceptance criteria coverage map

| Acceptance | Coverage |
|---|---|
| 5.1 `till init --group go` dispatcher-ready | W2 |
| 5.2 `till init --group go --group fe` multi-group | W2 |
| 5.3 `till template save/list/show/diff/restore` | W3 |
| 5.4 `till agents save/list/show/diff` | W3 |
| 5.5 `till project update --root-path` | W3 |
| 5.6 `till project delete/archive/restore/rename` | W3 |
| 5.7 `till action_item create` | W3 |
| 5.8 TUI MCP confirm (default yes) | W2 |
| 5.9 10-agent set per group, no orphans, plan-qa split, orchestrator-managed kept | W4.D1 |
| 5.10 `wails dev` works with stil tokens | W6 |
| 5.11 `till serve` removed | W7.D1 + W7.D2 |
| 5.12 `mage ci` green | all waves |
| 5.13 SQL-free dogfood path end-to-end | Deferred to Drop 4c.7 (see note above) |

### Pre-MVP rules carried over from Drop 4c.6

Planner + builder run `model: sonnet`; QA pair runs `model: opus` (per system frontmatter at `~/.claude/agents/go-*.md`). Filesystem-MD mode, no Tillsyn-runtime per-droplet plan items, no closeout MD rollups, single-line conventional commits ≤72 chars, never raw `go test` / `go build` / `go vet` / `mage install`. Builder spawn prompts MUST include "do NOT commit" directive — orch commits after each droplet closes per WORKFLOW.md Phase 4. Section 0 SEMI-FORMAL REASONING in every subagent response, but Section 0 stays in the orchestrator-facing response — never in PLAN.md or QA files.

### Refinements to log at drop end

| ID | Description |
|---|---|
| EXTRACT-R1 | Move `internal/tui/components/` + `internal/tui/style/` to `github.com/hylla-org/lykta` post-dogfood |
| EXTRACT-R2 | Move Tillsyn-generic FE components to `@hylla/stil-solid` post-dogfood |
| TILL-SERVE-R1 | Rebuild HTTP/MCP server from scratch as prereq for web variant + teams feature |
| METHO-R1 | Methodology docs substantive content during/after Drop 4c.8 |
| A1-R1 | Update `stil/README.md` to drop `stil-rust` adapter mention |
| D7-R5 | Dev MCP pollutes workspace `.tillsyn/` runtime files in --dev mode (post-dogfood UX cleanup) |
| D7-R6 | Manual cleanup of FLAT agent dirs in existing projects from dogfood-ramp session (TILLSYN-TEST, TILLSYN, /tmp/tillsyn-dogfood-smoke). Dev runs `rm -rf <project>/.tillsyn/agents && till init`. No migration code shipped. |
| FE-MOBILE-R1 | Capacitor wrap for mobile when needed |
| FE-WEB-R1 | Web variant via proper HTTP server post-till-serve-rebuild |
| ORCH-MANAGED-R1 | Split `orchestrator-managed.md` into role-specific agents (closeout-agent, refinement-agent, discussion-agent, human-verify-agent) during Drop 4c.8 prompt-authoring |
| BOOTSTRAP-R1 | `till agents bootstrap` extends to other non-`~/.claude/agents/` sources post-MVP (e.g. per-org template libraries, marketplace pulls) |
| KEYBIND-R1 | Extract `internal/tui/keybindings/` to `github.com/hylla-org/lykta` when lykta publishes |
| KEYBIND-R2 | Extract `fe/frontend/src/lib/vim/` (engine + wails-keys + palette) to `github.com/hylla-org/ro-vim` when ro-vim publishes |
| KEYBIND-R3 | When stil-solid lands: canonicalize Tillsyn's 5 local commands (`dispatch`, `plan`, `archive`, `settings`, `help`) INTO `stil/main/src/bindings/baseline.json`'s existing `product_extensions.tillsyn` block as an ADDITIVE operation — all 9 commands (baseline's 4 + local's 5) land in baseline; local `.tillsyn/bindings.json` becomes no-op. NOT a "move" — the slot already has 4 commands; this is additive canonicalization. (R3-FF2 disposition) |
| BIND-CONSIST-R1 | Cross-surface keybinding consistency test: same `j` does next-item in BOTH TUI action item list AND desktop FE project list |
| NATIVE-MENU-R1 | Wails native menu integration with vim command dispatch (File→Open Project triggers same handler as `:plan` vim command) — post-4c.7 |
| QA-SPLIT-R1 | Drop 4c.8: author 4 distinct QA prompt files per group replacing the 2-into-4 duplicate seed from `till agents bootstrap` |
| EMBED-PROMPTS-R1 | Drop 4c.8: substantive content for embedded `internal/templates/builtin/agents/<group>/*.md` defaults (30 prompts = 10 × 3 groups); replaces 4c.6.1 placeholders |
| CASCADE-WIRING-R1 | Drop 4c.7: state-trigger autonomous dispatch on `in_progress`, full gate runner, post-build pipeline |
| W8-DECOMP-R1 | W8 sub-plan decomposition shape — L2 sub-planner may split into W8.go + W8.fe second-level sub-plans if the flat-20-droplet shape proves unwieldy at decomposition time. Optional optimization; falsifier verdict (R3-NIT3) states "either shape is defensible." Orchestrator preserves flat-20 at L1; L2 decides. |
| W8-SMOKE-R1 | Integration smoke only verifies ONE prompt's 3-tier resolver pickup (W8 acceptance bullet). Full end-to-end smoke (`till dispatcher run --dry-run`) is deferred to Drop 4c.7 acceptance §5.13 per round-2 dev disposition. |
| PLAN-QA-DISCIPLINE-R1 | Future plan-QA falsification spawn briefs include "for every acceptance bullet asserting NEW behavior, verify the test-runner droplet's blocked_by includes the wave that ships that behavior" as an explicit attack angle (tracked; process refinement) |
| PLAN-QA-DISCIPLINE-R2 | For every surgical cross-wave or cross-droplet absorption in round-N+1 planning, sweep all L1 structural claims (wave roster, parallelism notes, decomposition-shape table, dependency graph) to verify they still hold post-absorption. Captured after R6-FF1 (round-6 added W8 cross-wave dep but didn't update lines 122/793/804). Includes verifying NUMERIC consistency — narrative droplet COUNTS in L1 must match the L2 spawn directive's enumerated D-list. Counts carried forward unverified from prior rounds are a recurring failure pattern (captured from R7-FF1) (tracked; process refinement) |
