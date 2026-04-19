# DROP_1_5_HYLLA_FINDINGS — drop-in-ready content for STEWARD

**Target**: `main/HYLLA_FEEDBACK.md` append after merge. STEWARD writes.
**Tillsyn drop**: `0aa63b58-69b6-4aea-9501-ab07dec2b7a7` (under HYLLA_FINDINGS parent `8b1717e1-...`).

---

## Drop 1.5

### Miss 1 — `hylla_search` skipped struct-level type declarations

- **Query**: `hylla_search` on `"markdownRenderer struct"` and `"markdownRenderer"` (P2-A).
- **Missed because**: Hylla returned only call sites for `markdownRenderer` (places where the type was instantiated or passed), never the declaration node itself. The type's struct-declaration node ranked below call-site references in the search output, so the builder couldn't find where the type was defined via Hylla alone.
- **Worked via**: `Grep` on the drop/1.5 worktree for `type markdownRenderer` — resolved in one hit.
- **Suggestion**: index struct/interface/type-alias declaration nodes as top-ranked hits for symbol queries. Call-site references should rank below the declaration itself when both match a query.

### Miss 2 — `hylla_refs_find` didn't surface field-selector read sites

- **Query**: `hylla_refs_find` on field `threadMarkdown` of the `Model` struct (P2-A — needed for `threadMarkdown markdownRenderer` → `*markdownRenderer` refactor impact analysis).
- **Missed because**: Hylla surfaced only struct-field declaration sites, not selector-expression reads of the form `m.threadMarkdown` or `Model.threadMarkdown`. Field-rename impact analysis needs read-site coverage to be safe.
- **Worked via**: `Grep` on `m\.threadMarkdown|Model\.threadMarkdown` across `internal/tui/` — ~12 hits located in seconds.
- **Suggestion**: index selector expressions (`x.Field`) as ref nodes, with a `kind=field-read` or `kind=field-write` discriminator. This unlocks safe field-level refactor analysis via Hylla.

### General pattern — uncommitted files invisible to Hylla

- P4-T3, P4-T4, and P2-A builder subagents + QA twins all hit the expected "uncommitted files invisible to Hylla" pattern because Drop 1.5 content never landed on `main` during the drop's build cycle (Hylla resolves `@main`). All agents worked via `Read` + `LSP` + `Grep` + `git diff` directly per project rule. This is not a Hylla bug — it's the documented drop-cycle pattern where Hylla ingest is drop-end-post-merge only. Flagging for the expected noise-level reporting.

### No-miss reports

- P3-A and P3-B builder subagents reported no Hylla misses (both predated the current drop cycle).
- P4-T1 and P4-T2 pre-compact reports not preserved through compaction; likely-clean given similar scope to P3-A.
