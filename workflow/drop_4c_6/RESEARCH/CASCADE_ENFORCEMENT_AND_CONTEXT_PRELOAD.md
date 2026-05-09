# CASCADE_ENFORCEMENT_AND_CONTEXT_PRELOAD

**Scope.** Drop 4c.6 needs to know whether the current Tillsyn code (HEAD: `main`, ingest snapshot 5) actually enforces (A) auto-creation of QA-twin children when `kind=plan` / `kind=build` action items are created, (B) per-spawn context preloading from the cascade tree, plus (C) the rename cost for `default-*.toml` → `till-*.toml`, and (D) the implementation cost of wipe-and-restart vs edit-existing for failed-plan re-decomposition. Findings are read-only; no code changes recommended here.

**TL;DR per investigation.** A: child_rules schema + resolver are committed and the embedded `default-go.toml` declares all four required QA-twin rules; **NO production code consumes `Template.ChildRulesFor` — auto-creation of QA twins is NOT enforced today, it is hand-waved by the orchestrator.** B: `internal/app/dispatcher/context/aggregator.go` is fully implemented with rule schema, per-rule resolvers, caps, timeouts, and tests; **NO production code calls `context.Resolve` from the spawn pipeline — every spawn lands a 6-line stub system-prompt.md that omits parent / sibling / ancestor context entirely.** C: 12 source-tree files reference the filenames; the load-bearing site is one `//go:embed` directive plus two switch-case strings — rename cost is small (~30 string edits in code, ~140 in workflow docs which can be sed-batched). D: `domain.ActionItem.Archive` is single-row only and does NOT cascade to children; wipe-and-restart needs a new service method (~50-80 LOC + tests). Edit-existing avoids the new method but pushes salvage logic into the planner agent prompt — no Go cost but a longer planner prompt.

---

## A. `[[child_rules]]` Auto-Creation Enforcement

**Bottom line: NOT ENFORCED.** The schema, resolver, and TOML rules are all committed and pass tests, but no production code path consumes `Template.ChildRulesFor` to actually spawn the QA-twin children. The orchestrator (parent Claude Code session) creates the twins by hand today. Drop 4c.6's prompt drafts cannot rely on automatic twin creation.

### A.1 What's wired

`internal/templates/builtin/default-go.toml` lines 209-269 declare exactly four `[[child_rules]]` entries plus two structural-type-narrowed extras:

- `when_parent_kind = "build"` → `create_child_kind = "build-qa-proof"`, title `BUILD-QA-PROOF`, `blocked_by_parent = true` (lines 209-213).
- `when_parent_kind = "build"` → `create_child_kind = "build-qa-falsification"`, title `BUILD-QA-FALSIFICATION`, `blocked_by_parent = true` (lines 216-220).
- `when_parent_kind = "plan"` → `create_child_kind = "plan-qa-proof"`, title `PLAN-QA-PROOF`, `blocked_by_parent = true` (lines 223-227).
- `when_parent_kind = "plan"` → `create_child_kind = "plan-qa-falsification"`, title `PLAN-QA-FALSIFICATION`, `blocked_by_parent = true` (lines 230-234).
- `when_parent_kind = "plan"` AND `when_parent_structural_type = "drop"` → `create_child_kind = "plan-qa-proof"`, title `DROP-PLAN-QA-PROOF` (lines 256-261).
- `when_parent_kind = "plan"` AND `when_parent_structural_type = "drop"` → `create_child_kind = "plan-qa-falsification"`, title `DROP-PLAN-QA-FALSIFICATION` (lines 264-269).

The "drop-level planner droplet" (third spec item from `PLAN.md § 19.3 line 1635`) is intentionally deferred per the inline comment at lines 236-253 of `default-go.toml`: a `plan→plan` rule trips the load-time cycle validator at `internal/templates/load.go:validateChildRuleCycles`. The two QA twins ship; the recursive planner-droplet does not.

`default-generic.toml` carries the same four universal rules but omits the two `structural_type=drop` entries (per its lines 253-261 comment block).

`internal/templates/child_rules.go:96-123` defines `func (t Template) ChildRulesFor(parent domain.Kind, parentType domain.StructuralType) []ChildRuleResolution`. The resolver:

- iterates `t.ChildRules` in declaration order;
- skips a rule whose `WhenParentKind != parent`;
- skips a rule whose non-empty `WhenParentStructuralType` does not match `parentType` (empty matches every type);
- looks up `t.Kinds[rule.CreateChildKind]` to pull `StructuralType` + `Owner` defaults;
- returns a `[]ChildRuleResolution` sorted by `(StructuralType, Kind)` for determinism.

The resolver has 7 unit tests in `internal/templates/child_rules_test.go` (95+% coverage per the QA proof at `workflow/drop_3/3.11_BUILDER_QA_PROOF.md`) plus three end-to-end tests in `internal/templates/embed_test.go` (`TestDefaultTemplateChildRulesForBuild`, `TestDefaultTemplateChildRulesForPlan`, `TestDefaultTemplateChildRulesForDropPlan`) that verify the embedded TOML produces the expected resolutions.

### A.2 What's NOT wired

`git grep -n ChildRulesFor` returns hits ONLY in `internal/templates/` (declaration + tests). `git grep -n "child_rules\|ChildRules\b" internal/app internal/adapters cmd` returns two hits — both in `internal/app/template_service_test.go:55` and `:61`, which are TOML literal fixtures inside test bodies, not consumer code.

`internal/app/service.go:1032-1201` (`Service.CreateActionItem`) is the single create path for action items. The function:

1. Resolves parent + scope guards (lines 1043-1063).
2. Runs the template-rejection audit hook (lines 1075-1093).
3. Builds + validates the new action item (lines 1095-1162).
4. Runs the git-status pre-check on declared paths (lines 1174-1178).
5. Persists via `s.repo.CreateActionItem(ctx, actionItem)` (line 1180).
6. Enqueues an embedding job (line 1183).
7. **Conditionally** seeds STEWARD drop-end findings — but only when `parentID == "" && actionItem.DropNumber > 0` (lines 1194-1198, calling `seedDropFindingsAndGate` from `internal/app/auto_generate_steward.go:233`).
8. Publishes a project-changed event (line 1199).

There is no `tpl.ChildRulesFor(actionItem.Kind, actionItem.StructuralType)` call. There is no loop over the resolutions. There is no second `CreateActionItem` invocation per resolution. The dispatcher's `walker.go`, `dispatcher.go`, `spawn.go`, and `bundle.go` likewise contain no `ChildRulesFor` reference (`git grep -n ChildRulesFor internal/app/dispatcher/` returns nothing).

`auto_generate_steward.go:seedDropFindingsAndGate` (lines 186-352) is a sibling auto-generator but for a completely different concern: it materializes the 5 STEWARD-owned `level_2` drop-end findings (`DROP_N_LEDGER_ENTRY`, `DROP_N_REFINEMENTS_RAISED`, `DROP_N_HYLLA_FINDINGS`, etc.) plus the `REFINEMENTS_GATE` confluence under the persistent STEWARD anchors. It runs only on `level_1` numbered drops and is unrelated to the per-action-item cascade-twin contract.

### A.3 Drop reference for the missing wiring

Drop 3 droplet 3.20 was the original consumer-binding droplet. `workflow/drop_3/PLAN.md:476` reads: *"Auto-generator wiring in the rule-engine entry path (Unit B's 3.11 `ChildRulesFor` is the spec resolver; this droplet is the consumer that fires `app.CreateActionItem` for each resolution)."* PLAN.md line 489 names the blocked-by chain: `Blocked by: 3.11 ... 3.14 ... 3.19`. The droplet number was reserved but `git grep -n "droplet 3.20\|3\.20.*BUILDER" workflow/drop_3/` does not return a worklog (verified by visual inspection of the workflow tree — only `3.11_BUILDER_*`, `3.14_BUILDER_*`, `3.28_BUILDER_*` worklogs exist). The auto-generator droplet was deferred. Drop 4 (the dispatcher) was supposed to absorb the responsibility, but `internal/app/dispatcher/dispatcher.go` (1,400 LOC) does not call `ChildRulesFor` either — Drop 4a/4b/4c shipped the spawn pipeline, gates, lock manager, and CLI adapter without picking up the auto-generator hook.

### A.4 What Drop 4c.6 needs

Drop 4c.6's planner-agent prompt and SKETCH §11 refer to "the system properly auto triggers the creation of the other required things." That contract is **not delivered today**. Two options for Drop 4c.6:

1. **Land the consumer.** Add a step inside `Service.CreateActionItem` (after the persist at line 1180, before the publish at 1199) that calls `tpl.ChildRulesFor(actionItem.Kind, actionItem.StructuralType)`, iterates the resolutions, and recursively calls `Service.CreateActionItem` for each. Plumbing concerns: the inner calls need to skip the template-rejection audit (parent IS the just-created item, the kind nesting was already validated by the resolver returning the resolution), need to inherit the same actor for audit attribution, and need to handle the `blocked_by` edge wiring (the closed-enum `blocked_by` field already exists on `domain.ActionItem` per Drop 4a Wave 1, so this is a slice append). Estimated LOC: 30-50 in `service.go` + 5-10 in `template_service.go` (template lookup helper) + 60-80 in `service_test.go` (table-driven coverage).
2. **Document hand-creation.** Keep the current behavior. Drop 4c.6's planner-agent prompt explicitly tells the planner agent to also create the QA twins itself via `till.action_item create` MCP calls. Cost: prompt-only.

Recommendation pointer (NOT a decision): option 1 is small; the schema + resolver + tests are already there; the gap is a 30-50 LOC consumer that turns a documented promise into actual behavior. Drop 4c.6's "agents auto get all the context they need" goal is undermined by a parallel contract gap (auto-creation of QA twins) — landing both together makes the cascade actually work.

### A.5 Citations

- Schema + resolver: `internal/templates/child_rules.go:21-58`, `internal/templates/child_rules.go:96-123`.
- Embedded rules: `internal/templates/builtin/default-go.toml:209-269`, `internal/templates/builtin/default-generic.toml:198-250` (universal pair only).
- Tests: `internal/templates/child_rules_test.go:104-296`, `internal/templates/embed_test.go:272-365`.
- Single create path: `internal/app/service.go:1032-1201`.
- Drop 3 deferral evidence: `workflow/drop_3/PLAN.md:476`, `workflow/drop_3/PLAN.md:649`.
- Confirmation no consumer exists: `git grep -n ChildRulesFor internal/app internal/adapters cmd` returns nothing.

---

## B. Context Preloading Per Cascade Kind

**Bottom line: SCHEMA + ENGINE COMMITTED, BUT NOT WIRED.** The `[agent_bindings.<kind>.context]` blocks are populated in `default-go.toml`. The aggregator engine in `internal/app/dispatcher/context/aggregator.go` is fully implemented (rules, caps, timeouts, file vs inline delivery, greedy-fit, markers). But `internal/app/dispatcher/cli_claude/render/render.go:assembleSystemPromptBody` (lines 246-279) — the function that authors the spawn's `system-prompt.md` — does NOT call `context.Resolve`. Every spawn today receives a 6-field stub (`task_id`, `project_id`, `project_dir`, `kind`, `title`, `paths`/`packages`, move-state directive) and nothing else. Planners, builders, QA agents all see the same minimal envelope today.

### B.1 Schema + engine inventory

Per-kind context blocks in `default-go.toml`:

| Kind                       | parent | parent_git_diff | ancestors_by_kind | siblings_by_kind | descendants_by_kind | delivery | max_chars | max_rule_duration | Citation |
| -------------------------- | ------ | --------------- | ----------------- | ---------------- | ------------------- | -------- | --------- | ----------------- | -------- |
| `plan`                     | true   | (omitted)       | `["plan"]`        | (omitted)        | (omitted)           | `file`   | 50,000    | 500ms             | lines 410-415 |
| `build`                    | true   | **true**        | `["plan"]`        | (omitted)        | (omitted)           | `file`   | 50,000    | 500ms             | lines 457-463 |
| `plan-qa-proof`            | true   | (omitted)       | `["plan"]`        | (omitted)        | (omitted)           | `file`   | 50,000    | 500ms             | lines 485-490 |
| `plan-qa-falsification`    | true   | (omitted)       | `["plan"]`        | (omitted)        | (omitted)           | `file`   | 50,000    | 500ms             | lines 512-517 |
| `build-qa-proof`           | true   | (omitted)       | `["plan"]`        | (omitted)        | (omitted)           | `file`   | 50,000    | 500ms             | lines 546-551 |
| `build-qa-falsification`   | true   | (omitted)       | `["plan"]`        | (omitted)        | (omitted)           | `file`   | 50,000    | 500ms             | lines 578-583 |
| `research`                 | (no `[context]` block declared) | | | | | | | | lines 417-428 |
| `closeout` / `commit` / `refinement` / `discussion` / `human-verify` | (no `[context]` block) | | | | | | | | lines 585-653 |

**Notable absences from the default seed (per the F.7.18 REV-4 design committed in `default-go.toml` lines 535-540 + 567-573):** `parent_git_diff` is intentionally NOT set on `build-qa-proof` / `build-qa-falsification` — QA must run its own `git diff` to avoid being biased by the builder's framing. `siblings_by_kind` is **not declared anywhere in the default template**. `descendants_by_kind` is **not declared anywhere in the default template** either, though the schema accepts it (rules.go:176-240).

The aggregator engine `Resolve` function lives at `internal/app/dispatcher/context/aggregator.go:243-453`. Its features:

- 5 rule names: `parent`, `parent_git_diff`, `siblings_by_kind`, `ancestors_by_kind`, `descendants_by_kind` (constants at lines 67-72 + slice at 78-84).
- Empty-binding fast path (line 247): if every rule is disabled, returns empty `Bundle{}` with no reader calls.
- Engine-time defaults (lines 41-58): bundle cap 200,000 chars / rule cap 50,000 / bundle duration 2s / rule duration 500ms.
- Per-rule wall-clock cap via nested `stdcontext.WithTimeout` (line 331).
- Greedy-fit truncation: per-rule overflow truncates with marker + stashes full content at `Files["<rule>.full"]`; bundle overflow skips the rule entirely with a marker (lines 374-401).
- File vs inline delivery: `Files["<rule>.md"]` for `delivery="file"` (default), or `RenderedInline` accumulator for `delivery="inline"` (lines 419-433).
- Markers always land in `RenderedInline` so `system-append.md` always surfaces them (lines 436-449).

Per-rule resolvers in `internal/app/dispatcher/context/rules.go`:

- `resolveParent` (lines 15-28): looks up `item.ParentID` via the reader, renders `renderActionItemBlock(parent)`.
- `resolveParentGitDiff` (lines 34-57): pulls parent's `StartCommit` + `EndCommit`, calls injected `GitDiffReader.Diff`.
- `resolveSiblingsByKind` (lines 67-124): lists same-parent children via `reader.ListSiblings(item.ParentID)`, filters by accepted kinds, picks the LATEST round per kind (most recent `CreatedAt`, lex tie-break on `ID`), renders each survivor.
- `resolveAncestorsByKind` (lines 135-163): walks UP the parent chain, returns the FIRST ancestor matching `acceptedKinds` (halts on first match per F.7.18 spec). 256-hop loop guard.
- `resolveDescendantsByKind` (lines 176-240): walks the cascade subtree depth-first via `reader.ListChildren`, returns every match. 4096-node loop guard.
- `renderActionItemBlock` (lines 261-304): emits `### <title> (<kind>)\nid: <id>\n[paths/packages/start_commit/end_commit]\n\n<description>\n` — terse markdown agents can read inline.

The full engine ships with `internal/app/dispatcher/context/aggregator_test.go` (24 KB of tests), and template-side schema validation is in `internal/templates/load.go` + `internal/templates/context_rules_test.go`. Coverage is high.

### B.2 The wiring gap

`git grep -n "context\.Resolve\|aggcontext\|dispatcher/context" internal/ cmd/` returns exactly two hits:

1. `internal/app/dispatcher/commit_agent.go:97` — a doc-comment about the parallel `GitDiffReader` interface (NOT a call).
2. `internal/app/dispatcher/context/aggregator.go:22` — the package's own doc-comment about the import alias.

There is no production call site. The aggregator's exported entry point is unreachable from the spawn pipeline.

`internal/app/dispatcher/cli_claude/render/render.go:Render` (lines 125-179) is the function that authors the bundle. Step 1 calls `renderSystemPrompt` at line 147, which delegates to `assembleSystemPromptBody` at line 246. The body of `assembleSystemPromptBody` reads (verbatim, lines 246-279):

```go
func assembleSystemPromptBody(item domain.ActionItem, project domain.Project) string {
    var b strings.Builder
    b.WriteString("task_id: ")
    b.WriteString(item.ID)
    b.WriteString("\n")
    b.WriteString("project_id: ")
    b.WriteString(project.ID)
    b.WriteString("\n")
    b.WriteString("project_dir: ")
    b.WriteString(project.RepoPrimaryWorktree)
    b.WriteString("\n")
    b.WriteString("kind: ")
    b.WriteString(string(item.Kind))
    b.WriteString("\n")
    if item.Title != "" {
        b.WriteString("title: ")
        b.WriteString(item.Title)
        b.WriteString("\n")
    }
    if len(item.Paths) > 0 {
        b.WriteString("paths: ")
        b.WriteString(strings.Join(item.Paths, ", "))
        b.WriteString("\n")
    }
    if len(item.Packages) > 0 {
        b.WriteString("packages: ")
        b.WriteString(strings.Join(item.Packages, ", "))
        b.WriteString("\n")
    }
    b.WriteString("move-state directive: ...\n")
    return b.String()
}
```

That is the entire system-prompt body for every cascade kind today. No parent description. No ancestor walk. No sibling QA verdicts. No `parent_git_diff`. The bundle's `<bundle>/context/` directory is never created because the aggregator never runs.

The closest reference to the gap is the doc-comment on `renderSystemPrompt` at lines 217-219: *"The body shape mirrors the F.7.17.5 assemblePrompt body but is owned by render going forward."* F.7.17.5 was the spawn-pipeline droplet that lifted the prompt body out of `spawn.go` into the render layer; F.7.18 was the aggregator design. They were committed separately and never joined.

### B.3 What the spawned agents actually receive today

Every cascade-spawned process today reads `<bundle>/system-prompt.md` containing only the 6 stub fields above. Subsequent context is pulled by the agent itself via `till.action_item(operation=get)` MCP calls and by reading the canonical agent template at the system-installed plugin path. Concretely:

- A `kind=plan` planner spawn sees `task_id`, `project_id`, `project_dir`, `kind=plan`, `title`, optional `paths` / `packages`, move-state directive. **Nothing** about its parent plan (when it exists), nothing about the project's `description`, nothing about sibling QA verdicts on prior rounds.
- A `kind=plan-qa-proof` spawn sees the same 6 fields. **Nothing** about its parent plan it is supposed to review. Nothing about the parent's child plan/build decomposition (siblings of the QA-twin, peer-children of the parent plan).
- A `kind=build-qa-proof` spawn sees the same 6 fields. **Nothing** about its parent build, nothing about the build's `start_commit`/`end_commit` (so no auto-`git diff`), nothing about its sibling `build-qa-falsification` verdict.
- A `kind=build` builder spawn sees the same 6 fields. **Nothing** about its parent plan's design narrative, nothing about the planner's stated acceptance criteria beyond what's in the build's own description.

The agents work today because (1) the orchestrator stuffs the relevant context into the action-item description prose at create time, and (2) the agents themselves call back into Tillsyn via MCP to fetch parent / sibling / ancestor data. Both behaviors are dev-tax — the aggregator was designed to eliminate them.

### B.4 What Drop 4c.6 needs

The wiring is a single function-call insertion plus the data plumbing. Sketch (NOT a recommendation, just the shape):

1. Inside `render.Render` (or inside `dispatcher.BuildSpawnCommand` upstream of Render), call `aggcontext.Resolve(ctx, ResolveArgs{Binding: binding.<context block>, Item: item, ProjectID: project.ID, BundleCharCap: ..., BundleDuration: ..., Reader: <new adapter>, DiffReader: <new adapter>})`.
2. Wire the `Reader` adapter to the SQLite `Repository` (`GetActionItem`, `ListChildren` already exist; add `ListSiblings(parentID)` if missing — `git grep -n "ListSiblings" internal/adapters/storage/sqlite/repo.go` not yet checked, but a wrapper over `ListActionItems` filtered by `ParentID` is trivial).
3. Wire the `DiffReader` adapter to the existing `git diff` shell-out used by `commit_agent.go` (the parallel interface at `commit_agent.go:102-108` is intentionally identical-shape per the doc-comment).
4. Inside `assembleSystemPromptBody`, append `bundle.RenderedInline` (markers + inline content) to the returned body string. Write `bundle.Files` to `<bundle>/context/<filename>` via a new helper.
5. Plumb the binding's `Context` block through `BindingResolved` (pre-existing field per `internal/app/dispatcher/binding_resolved.go` likely; verify).

LOC estimate: 80-150 in `render.go` + render adapter, 30-50 in repo adapter for `ListSiblings`, 50-100 in tests, plus a wiring change in `dispatcher.BuildSpawnCommand`. Total ~200-300 LOC of net-new + test rewrites. The engine is done; what's missing is plumbing.

### B.5 Bonus: per-kind context **as configured** vs **as designed for Drop 4c.6**

The dev's question 2 names a richer cascade-context contract than the default seed declares:

> *"a planner should start with its own prompt, planning qa should start with the planner and all build action items that are what it level 'on its cascade tree branch (kind of like its child stuff)' and build qa should should get all build nodes, plan nodes for its stuff"*

Mapping that to the schema:

- **Planner**: `parent = true` + `ancestors_by_kind = ["plan"]` (so a sub-plan sees its enclosing plan). Default seed already has this. Adding `descendants_by_kind = ["plan", "build"]` would give a fix-up planner the existing decomposition. NOT in default seed today.
- **Plan-QA (proof + falsification)**: needs the parent plan AND the parent's child build/plan/research action items (siblings-of-the-QA-twin, peer-children-of-the-plan). The default seed declares `parent = true` + `ancestors_by_kind = ["plan"]` — that gives it the parent plan but **NOT the peer-build/plan/research children of that plan**. The schema's `siblings_by_kind` rule is the wrong axis here (siblings of the QA-twin are the OTHER QA-twin and the peer-children, but `siblings_by_kind` in rules.go:67-124 picks the LATEST ROUND per matching kind, not "all children of my parent"). The dev's contract needs either a new rule (`peers_by_kind` = all my parent's children matching kind) or a re-purpose of `siblings_by_kind` semantics. **This is a design gap, not just a wiring gap.**
- **Build-QA (proof + falsification)**: needs the parent build AND the grandparent plan AND ideally the sibling QA verdict if posted. Default seed: `parent = true` + `ancestors_by_kind = ["plan"]` covers parent build + grandparent plan walk-up. Sibling QA verdict requires `siblings_by_kind = ["build-qa-proof", "build-qa-falsification"]` (each picks the OTHER twin). Not in default seed today.

So the answer to the dev's question 2 is two-fold: (a) the engine wiring is missing entirely (B.2, B.3); (b) even with wiring, the default `[context]` blocks don't cover the dev's Plan-QA-sees-peer-children contract — a schema rule design gap. Drop 4c.6's TOML will need to extend the default seed (and possibly the schema if `peers_by_kind` is the right name).

### B.6 Citations

- Engine: `internal/app/dispatcher/context/aggregator.go:243-453`.
- Per-rule resolvers: `internal/app/dispatcher/context/rules.go:1-304`.
- Default context seeds: `internal/templates/builtin/default-go.toml:410-583`.
- The 6-line stub: `internal/app/dispatcher/cli_claude/render/render.go:246-279`.
- Confirmation no caller exists: `git grep -n "context\.Resolve\|aggcontext\|dispatcher/context" internal/ cmd/` returns 2 doc-comment hits, zero call sites.
- F.7.18 design intent: `workflow/drop_4c/F7_18_CONTEXT_AGG_PLAN.md` (referenced; not exhaustively read).

---

## C. Rename Audit — `default-*.toml` → `till-*.toml`

**Bottom line: small code cost (~30 lines across 12 source files, of which ~8 are doc-comments and 2 are load-bearing strings), large doc-stale-references cost (~140 workflow MD references, zero top-level repo MD references).** The rename is a 1-2 hour mechanical edit if batched, with one moderately-risky touchpoint (`//go:embed` directive) that warrants a `mage ci` after.

### C.1 Source-tree references (load-bearing)

12 files reference the filenames. Edit list:

1. **`internal/templates/builtin/default-go.toml`** — RENAME the file. The file's own header doc-comments (lines 5-11) reference `default-go.toml` and `default-generic.toml`; the doc-comments need to be updated to `till-go.toml` / `till-gen.toml` (8 hits).
2. **`internal/templates/builtin/default-generic.toml`** — RENAME the file. Header doc-comments at lines 3-40 reference both filenames extensively (~9 hits across cross-references in comments).
3. **`internal/templates/embed.go`** — line 34 has `//go:embed builtin/default-go.toml builtin/default-generic.toml`. Lines 17, 20, 59, 62, 66, 103, 106, 136, 138 are doc-comment + switch-case references. **Critical edits**: lines 136, 138 are the actual switch-case strings consumed by `LoadDefaultTemplateForLanguage(lang)` — these MUST change to `builtin/till-gen.toml` / `builtin/till-go.toml`. The `//go:embed` line MUST change to match. Doc-comment edits are cosmetic but should land for consistency.
4. **`internal/templates/embed.go:178`** (`BuiltinTemplateNames`) returns `[]string{"default-generic", "default-go"}`. **This is a wire-protocol-visible string** — `till.template list_builtin` MCP op surfaces these literally to the dev. Renaming requires changing both the literal here AND any test that asserts the wire shape (see #5).
5. **`internal/templates/embed_test.go`** — 30+ references including assertion strings (e.g. line 96: `f, err := DefaultTemplateFS.Open("builtin/default-generic.toml")`). All must update.
6. **`internal/templates/load.go`** — 3 doc-comment references at lines 255, 592, 735.
7. **`internal/templates/load_test.go`** — references in test strings (count via `git grep`).
8. **`internal/app/service.go`** — lines 383, 384 (doc-comments on `loadProjectTemplate`).
9. **`internal/app/service_test.go`** — references at 6525, 6529, 6534, 6537, 6551, 6552, 6713, 6849 (mostly test fixture path strings; `path := filepath.Join("..", "templates", "builtin", "default-go.toml")` at 6534 is load-bearing).
10. **`internal/app/auto_generate_steward_test.go:18`** — single doc-comment reference.
11. **`internal/adapters/server/common/mcp_surface.go`** — lines 906, 908 (doc-comments).
12. **`internal/adapters/server/mcpapi/extended_tools.go:1867`** — single doc-comment reference.
13. **`.tillsyn/template.toml`** — the project's own bare-root template carries 5 doc-comment references at lines 10, 16, 22, 39, 666 describing drift contract against the embedded default. Cosmetic but should land.

**Test-fixture concerns.** `internal/app/service_test.go:6534` builds a path with `filepath.Join("..", "templates", "builtin", "default-go.toml")`. The rename breaks this path; the test fails until the string is updated. Same risk for any other tests reading the embedded TOML by relative path — `git grep -n "builtin/default-" internal/` should be the canonical pre-rename audit list.

**Wire-protocol concerns.** `BuiltinTemplateNames()` returns the literal names `"default-generic"` + `"default-go"`. The `till.template list_builtin` MCP op surfaces these to the dev TUI, to `till template list-builtin` CLI output, and to `mcp_surface.go` documentation. Renaming changes the wire output. Drop 4c.6 needs to decide: (a) rename the wire output too (cleaner, breaks any external tooling depending on the prior names), or (b) keep `"default-generic"` / `"default-go"` as wire identifiers but change the file names + embed paths (clean separation but doc-comment confusion). Recommendation: option (a) — wire identifiers should match the file names — but this is the dev's call.

### C.2 Workflow-doc references (non-load-bearing)

`git grep -nE "default-go\.toml|default-generic\.toml" workflow/ | wc -l` returns 140 hits. These live in:

- `workflow/drop_4c/` — 4-5 files (planning + builder worklogs).
- `workflow/drop_4c_5/` — 6+ files (the drop where the rebadge from `default.toml` → `default-go.toml` originally landed).
- `workflow/drop_4d/REVISION_BRIEF.md` — 1 file.
- `workflow/drop_4c_6/SKETCH.md` — already references the rename (per the v2.4 changelog at line 7).

These are immutable per-drop archive. Per the project memory rule "Never Remove Workflow Drop Files" (memory key `feedback_never_remove_workflow_files.md`), drop_N MD files do NOT get retroactively edited even when their content goes stale. The rename doesn't require changing them. They will read as historical references to the prior names — this is consistent with how every drop's "rebadged in F.2.1" historical text reads after F.2.1 ships.

### C.3 Top-level repo MDs

`git grep -nE "default-go\.toml|default-generic\.toml" AGENT_CASCADE_DESIGN.md AGENTS.md CLAUDE.md CLI_ADAPTER_AUTHORING.md CONTRIBUTING.md HYLLA_FEEDBACK.md HYLLA_REFINEMENTS.md HYLLA_WIKI.md LEDGER.md PLAN.md README.md REFINEMENTS.md SEMI-FORMAL-REASONING.md SPAWN_PIPELINE.md STEWARD_ORCH_PROMPT.md WIKI.md WIKI_CHANGELOG.md` returns **zero hits**. Top-level living docs do not reference the filenames. Good — no doc churn at the project entry points.

### C.4 Cost summary + recommendation

- **Code edits**: ~30 string changes across 12 files. ~8 are doc-comments (cosmetic but should land for consistency), ~22 are load-bearing strings (file names, embed directive, switch-case, test paths, wire identifiers).
- **Risk**: low. The `//go:embed` directive is the sharpest edge — if the directive lists a file that doesn't exist, the build fails fast and loud at compile time; `mage ci` catches it instantly. The two `LoadDefaultTemplateForLanguage` switch cases are next-sharpest — wrong string there silently routes to the wrong file (or returns ErrLanguageNotSupported); the embed_test.go regression coverage already pins this case (`TestLoadDefaultTemplateForLanguageGo` / `TestLoadDefaultTemplateForLanguageGeneric` per line 878 + 909 references).
- **Wire shape**: `BuiltinTemplateNames()` is the dev-visible identifier. Recommend renaming wire too.
- **Workflow MDs**: leave alone (140 references, all historical, project rule forbids retroactive edits).
- **Estimated time**: 30-60 minutes for a builder agent given a precise edit list. `mage ci` after to confirm.

**Recommendation pointer**: do the rename in Drop 4c.6 alongside the agent-shipping work. The rename is intuitive ("till-" is the project's convention for embedded artifact groups; the dev wants `till-go` / `till-gen` for symmetry with `till-go` / `till-gen` / `till-gdd` agent groups in the same drop), and Drop 4c.6 will be touching `internal/templates/embed.go` anyway when adding embedded agent prompts. Bundling avoids a second drop's `mage ci` cycle. Defer-to-separate-drop only buys risk isolation, not effort reduction.

### C.5 Citations

- Source files: `git grep -lE "default-go\.toml|default-generic\.toml" internal/ cmd/ .tillsyn/` returns 12 paths.
- Embed directive: `internal/templates/embed.go:34`.
- Resolver switch: `internal/templates/embed.go:132-147` (function `LoadDefaultTemplateForLanguage`).
- Wire identifiers: `internal/templates/embed.go:177-179` (function `BuiltinTemplateNames`).
- Top-level MDs clean: `git grep -nE "default-go\.toml|default-generic\.toml" <17 top-level *.md files>` returns 0 hits.
- Workflow MDs: 140 hits across `workflow/drop_4c*` + `workflow/drop_4d/`.

---

## D. Wipe-and-Restart vs Edit-Existing — Implementation Cost

**Bottom line: wipe-and-restart needs ONE new service method on `app.Service` plus its repo adapter (~50-100 LOC + tests); edit-existing needs zero Go and pushes salvage logic into the planner agent prompt (~200-400 words of prompt). Wipe-and-restart is clearer for MVP; edit-existing is the dogfood-mature target. Recommendation: ship wipe-and-restart for MVP (per SKETCH §11.2 NEW); revisit edit-existing once the cascade-on-itself loop runs reliably enough that salvage logic earns its complexity.**

### D.1 Today's archive primitive

`internal/domain/action_item.go:619-624`:

```go
func (t *ActionItem) Archive(now time.Time) {
    ts := now.UTC()
    t.ArchivedAt = &ts
    t.LifecycleState = StateArchived
    t.UpdatedAt = ts
}
```

**Single-row only.** It mutates `t.ArchivedAt` + `t.LifecycleState` + `t.UpdatedAt` on the receiver and nothing else. No call to children, no walk, no SQL cascade. The repo's `UpdateActionItem` persists exactly this one row.

`internal/app/service.go:1773-1791` (`DeleteActionItem` with `mode == DeleteModeArchive`):

```go
case DeleteModeArchive:
    actionItem, err := s.repo.GetActionItem(ctx, actionItemID)
    if err != nil { return err }
    guardScopes, guardErr := s.capabilityScopesForActionItemLineage(ctx, actionItem)
    if guardErr != nil { return guardErr }
    if err := s.enforceMutationGuardAcrossScopes(ctx, actionItem.ProjectID, ..., guardScopes, domain.CapabilityActionArchiveOrCleanup); err != nil { return err }
    actionItem.Archive(s.clock())
    applyMutationActorToActionItem(ctx, &actionItem)
    if err := s.repo.UpdateActionItem(ctx, actionItem); err != nil { return err }
    s.publishActionItemChanged(actionItem.ProjectID)
    return nil
```

Single row. No child traversal. So today, archiving a plan does NOT archive its child build/plan/research/QA-twin items — they remain `todo` / `in_progress` / etc. with a now-archived parent. The parent-child invariant ("parent cannot complete with incomplete children") still applies, but archive is a state outside the lifecycle invariant — the archived parent simply no longer participates in the cascade.

There is no `RecursiveArchive` or `CascadeArchive` method. There IS a `SupersedeActionItem` per `cmd/till/action_item_cli.go:74-120` and the matching service method, but it operates on a single failed item (sets state to `complete` with `metadata.outcome="superseded"` plus a `--reason`). It does NOT archive children either.

### D.2 What wipe-and-restart costs

Sketched implementation:

```go
// Service.WipeChildrenAndRePlan archives every direct child of plan and every
// transitive descendant, then returns control so the caller can spawn a fresh
// planner. Idempotent: re-archiving an already-archived row is a no-op.
func (s *Service) WipeChildrenAndRePlan(ctx context.Context, planID string, reason string) error {
    plan, err := s.repo.GetActionItem(ctx, planID)
    if err != nil { return err }
    if plan.Kind != domain.KindPlan { return ErrNotAPlan }
    // capability + actor guards same shape as DeleteActionItem
    children, err := s.repo.ListActionItems(ctx, plan.ProjectID, false)
    if err != nil { return err }
    // walk subtree from plan.ID; archive each
    descendants := collectSubtree(children, plan.ID)
    for _, d := range descendants {
        if d.ArchivedAt != nil { continue }
        d.Archive(s.clock())
        d.Metadata.WipeReason = reason  // optional audit field
        if err := s.repo.UpdateActionItem(ctx, d); err != nil { return err }
    }
    s.publishActionItemChanged(plan.ProjectID)
    return nil
}
```

Estimated LOC:
- `service.go`: 40-60 LOC.
- `service_test.go`: 80-120 LOC (table-driven: empty subtree, single-level subtree, deep subtree, mixed-state subtree, idempotency, capability rejection, non-plan kind rejection).
- `cmd/till/action_item_cli.go` extension or new MCP op for adopters who need to invoke it manually: 30-50 LOC.
- Total: ~150-230 LOC + tests.

The actual `WipeReason` audit field is optional but cheap; the `Archive` audit trail (UpdatedAt, ChangeOperation = `Archive` per `internal/adapters/storage/sqlite/repo.go:2662-2667`) already records the event per-row.

**Audit trail.** The dev's SKETCH §11.2 explicitly says "Audit trail preserved (archive, never delete)." This matches the `Archive` semantics — rows stay queryable via `IncludeArchived = true`. `git diff` of the SQLite DB across drop boundaries would still show the prior decomposition.

### D.3 What edit-existing costs

Edit-existing pushes the decision logic into the planner-agent prompt. The planner re-reads the failed plan's children via `till.action_item(operation=list, parent_id=<plan>)`, classifies each as "salvageable" / "needs-rework" / "needs-deletion", then issues a mix of:

- `till.action_item(operation=update, ...)` for description/paths/packages amendments on salvageable children.
- `till.action_item(operation=archive, ...)` for children that are no longer reachable.
- `till.action_item(operation=create, ...)` for net-new replacement decompositions.

**Go cost**: zero. All three operations exist today (`UpdateActionItem`, `DeleteActionItem(mode=archive)`, `CreateActionItem`). The planner does the per-child decision work itself.

**Prompt cost**: the planner-agent draft at `workflow/drop_4c_6/PROPOSED_AGENT_DRAFTS/till-go/planning-agent.md` would gain a "fix-up mode" section: when the spawn detects a failed-prior-round (via `parent.metadata.outcome == "failure"` or sibling QA verdict comments), the planner walks existing children and edits/archives/creates rather than starting fresh. Estimated prompt addition: 200-400 words. Plus a heuristic for "when does it make sense to wipe vs edit" — typically wipe when >70% of children need rework, edit when <30%, judge in the middle.

**Risk**: planner agents executing per-child decisions across an existing decomposition is more complex than re-decomposing from scratch. The dogfood failure mode is the planner mis-classifying a child as salvageable when it isn't — Tillsyn's `--paths`/`--packages` lock-graph would surface the conflict at the next builder spawn (the corrupted child would conflict with its replacement), but the time-to-detect is one full QA cycle longer. Wipe-and-restart fails clean; edit-existing fails subtly.

### D.4 Recommendation

Per SKETCH §11.2 ("wipe-and-restart strategy for MVP ... edit-existing remains a future optimization"), the dev's call already favors wipe-and-restart. The Go cost analysis confirms it's small enough to land cleanly: ~150-230 LOC + tests is one droplet of work in Drop 4c.6's terms.

**Concrete recommendation**: ship `Service.WipeChildrenAndRePlan` (or equivalent name) in Drop 4c.6 alongside the planner-agent prompt. The planner agent's prompt should describe wipe-and-restart as the default failed-plan path; edit-existing surfaces in a future drop when (a) the wipe-and-restart loop has logged enough failed-plan cycles to identify which classes of failure DO have salvageable subtrees, and (b) the planner-agent prompt has the heuristics to classify reliably.

### D.5 Citations

- Single-row archive: `internal/domain/action_item.go:619-624`.
- Single-row delete-mode-archive: `internal/app/service.go:1773-1791`.
- Single-item supersede: `cmd/till/action_item_cli.go:74-120`.
- No cascade method: `git grep -n "RecursiveArchive\|CascadeArchive\|WipeChildren\|RePlan\|replan" internal/ cmd/` returns nothing.
- SKETCH §11.2 dev directive: `workflow/drop_4c_6/SKETCH.md:8` (v2.4 changelog).

---

## E. Summary For Drop 4c.6 Action

The four investigations together expose three "schema-committed-but-not-wired" gaps and one filename-rename hygiene opportunity. None of the four require redesigning shipped vocabulary; all four are wiring + small additions:

1. **A — auto-creation gap**: schema + resolver + tests already shipped in Drop 3 droplet 3.11; consumer at the `Service.CreateActionItem` create boundary is missing. ~30-50 LOC + tests to land.
2. **B — context-preload gap**: aggregator engine + per-rule resolvers + per-binding TOML schema all shipped in Drop 4c F.7.18; render-layer wiring at `assembleSystemPromptBody` is missing. Plus a schema gap: the dev's "Plan-QA sees all peer-children of its parent plan" contract isn't covered by today's `siblings_by_kind` semantics — needs either a new rule or re-purposed semantics. ~200-300 LOC + tests + 1 schema rule design decision.
3. **C — rename hygiene**: cheap to bundle into Drop 4c.6 (~30 string edits, low risk).
4. **D — wipe-and-restart**: small new method on `Service` (~150-230 LOC + tests). Edit-existing punted to a future drop.

Drop 4c.6's planner-agent prompt drafts that assume "auto-creation works" and "spawned agents see preloaded context" cannot rely on either today. Either (a) Drop 4c.6 closes both gaps and the prompts work as drafted, or (b) the prompt drafts get hedged language that documents hand-creation + per-spawn MCP fetches as the current reality. Closing the gaps is the smaller cost (combined ~250-400 LOC) and unlocks the prompt drafts as written. Hedging the prompts costs less Go but more prose churn and leaves the cascade dependent on orchestrator hand-creation forever.

Every claim in §A-D cites at least one `path:line` or `git grep` invocation that yields the cited result. The research is read-only; no code or non-`workflow/drop_4c_6/RESEARCH/` MD has been edited in producing it.
