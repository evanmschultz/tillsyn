# DROP_1_5_HYLLA_REFINEMENTS_RAISED — drop-in-ready content for STEWARD

**Target**: `main/HYLLA_REFINEMENTS.md` append after merge. STEWARD writes.
**Tillsyn drop**: `89a6de2c-59cf-4221-a7b4-e65aa753fc1e` (under HYLLA_REFINEMENTS parent `3ad4b367-...`).

---

## Drop 1.5 Hylla-specific refinements

### HR-1. Struct/interface declaration nodes should rank above call-site refs in symbol queries

- **From**: Drop 1.5 HYLLA_FINDINGS Miss 1.
- **Evidence**: `hylla_search` on `"markdownRenderer struct"` returned only call sites, not the declaration node. Builder fell back to `Grep` on `type markdownRenderer`.
- **Severity**: medium — every time a subagent needs to locate a type/interface/alias declaration (a common operation during refactor planning + impact analysis), Hylla forces a fallback.
- **Proposed fix**: add a ranking rule to the Hylla search ingest pipeline — declaration nodes (struct, interface, type alias, named type) rank above call-site references for queries naming the identifier. Alternative: expose a `node_kind` filter on `hylla_search` so `node_kind=declaration` returns only declarations.

### HR-2. Field-selector expressions (`x.Field`) should be indexed as ref nodes

- **From**: Drop 1.5 HYLLA_FINDINGS Miss 2.
- **Evidence**: `hylla_refs_find` on struct field `Model.threadMarkdown` surfaced struct-declaration sites only, not selector reads (`m.threadMarkdown`, `model.threadMarkdown`). Builder fell back to `Grep` for field-rename impact analysis.
- **Severity**: high — field-rename and field-type-change refactors are common and risky (the `markdownRenderer` value→pointer refactor in P2-A touched ~12 selector read sites). Without selector-expression indexing, refactor planning requires non-Hylla fallback every time.
- **Proposed fix**: index selector expressions as ref nodes with `kind=field-read` / `kind=field-write` discriminators. Extend `hylla_refs_find` to accept `include_field_selectors=true` as a param. Alternative: add a dedicated `hylla_field_refs_find` tool that returns all field-touch sites for a `(struct_name, field_name)` pair.

### HR-3. "Uncommitted files invisible" is working-as-designed but needs clearer signaling

- **From**: Drop 1.5 HYLLA_FINDINGS general-pattern note.
- **Evidence**: Every Drop 1.5 build-drop subagent + QA twin hit the "Hylla @main doesn't have my drop's files" case, correctly fell back to `Read` + `LSP` + `Grep` + `git diff`. Not a Hylla bug — documented pattern.
- **Severity**: low — noise-level issue, agents correctly route around it.
- **Proposed fix**: when `hylla_search` / `hylla_node_full` returns nothing for a symbol in a package under active change, response could include a hint like `"note: package <name> has uncommitted changes on branch <branch>; Hylla reflects @main only"`. Drop 2+ with per-commit snapshots would also solve this. Alternative: nothing — document the pattern more prominently in agent definitions (already done in CLAUDE.md §Code Understanding Rules).

### HR-4. No miss-reports for P4-T1 / P4-T2 due to compaction drop

- **From**: Drop 1.5 HYLLA_FINDINGS no-miss section.
- **Evidence**: P4-T1 + P4-T2 subagent closeout comments pre-dated compaction event; their `## Hylla Feedback` sections weren't preserved in session memory and weren't replicated in Rak MDs (those drops predated Rak MD pivot).
- **Severity**: low — process-gap, not a Hylla functional issue.
- **Proposed fix**: for future drops, Rak MD should include a `BUILDER_QA_FALSIFICATION.md` with `## Hylla Feedback` section in every round from drop inception, not only after Rak pivot. Memory capture for subagent Hylla feedback should be durable — orchestrator rolls up at drop-end regardless of compaction state.
