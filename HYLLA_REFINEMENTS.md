# Hylla Findings and Refinements

Append-only log of Hylla ergonomics, search-quality, and product issues discovered during Tillsyn work. Hylla is a sibling project — product fixes land in the Hylla project, not in Tillsyn. Entries here propagate into the Hylla project inside Tillsyn (see bare-root `CLAUDE.md` for the post-Slice-0 HYLLA project setup slice owned by the bare-root orchestrator only).

Format: newest-first. Each entry dated and tagged with the slice it was discovered in. Structure per entry:

- **Context** — what the session was doing.
- **Calls + inputs** — exact Hylla tools, parameters, and results.
- **What went wrong** — observed gap or friction.
- **Refinement candidates** — concrete Hylla-side fixes, each independently shippable.
- **Current-usage implications** — what best practice this implies for `HYLLA_WIKI.md`.

---

## 2026-04-14 — Slice 0 — Glamour theme hunt (library-name lookup)

### Context
Slice 0 continuation after compaction. Dev flagged a `glamour` theme update + commit pin (`e799a9f`). Orchestrator tried to locate the markdown renderer file to confirm the theme change landed in code.

### Hylla calls + inputs
1. `hylla_search_keyword` — `query="glamour"`, `artifact_ref="github.com/evanmschultz/tillsyn@main"`, `fields=["content","summary","docstring"]`, `limit=30` → **0 results.**
2. `hylla_search` (hybrid) — `query="markdown rendering glamour theme"`, `search_types=["keyword","vector"]`, `limit=20` → **error** `"field must be summary, content, or docstring"`. Message does not identify whether it is complaining about `field` (singular, vector path) or `fields` (plural, keyword path).
3. `hylla_search_keyword` — `query="markdown render"`, `fields=["content","summary"]`, `limit=30` → 18 hits, **all unrelated** data-carriers (`CreateCommentInput`, `SnapshotComment`, `MarkdownDocument`, `ErrInvalidBodyMarkdown`, etc.).
4. Fallback — `Grep "glamour" -i` against the worktree → returned `internal/tui/markdown_renderer.go` + 7 other files in one call.

### What went wrong
- **Import-name queries miss.** `glamour` is the literal Go import in the file. Keyword search across content / summary / docstring returned zero. Either LLM summaries do not mention third-party package names, or content tokenization is not indexing import idents as discrete tokens.
- **Schema confusion on hybrid search.** `hylla_search` uses `field` (singular). `hylla_search_keyword` uses `fields` (plural). The error does not say which parameter it is validating, nor which tool path triggered it.
- **Semantic-query flood.** `"markdown render"` is the conceptually correct query, but the project has heavy markdown-data surface (comments, docs, snapshots) so the single renderer file gets drowned. No way to bias toward behavior blocks vs data carriers.
- **Net cost.** 3 Hylla calls + 1 schema error vs. 1 Grep call. For library-name lookups, Hylla was strictly worse than Grep.

### Refinement candidates (Hylla-project plan items)
- **Imports facet.** Index third-party imports as a first-class facet so queries by package name (`glamour`, `bubbletea`, `lipgloss`, `cobra`) return files that import it.
- **Qualified-ident tokenization.** Ensure `glamour.NewTermRenderer`, `glamour.WithAutoStyle`, etc. are indexed as individual tokens so keyword search against `content` finds them.
- **Dependency reverse lookup.** New tool or mode: `hylla_imports_find` — "files that import X." Complements the imports facet.
- **Behavior-vs-data-carrier ranking bias.** Rank files whose primary symbols are funcs / methods above files whose primary symbols are structs / interfaces when the query is action-shaped (`"render markdown"`, `"parse config"`, `"dispatch task"`).
- **Schema cleanup.** Unify `field` (vector) vs `fields` (keyword) — or at minimum, error messages that name the parameter (`"field 'fields' must be one of summary|content|docstring"`) and the tool path.
- **Summary contract.** LLM summaries should include, minimum: (a) third-party imports actually used, (b) one-sentence behavior statement. Improves recall on library-name and concept queries.
- **Local-overlay dirty mode.** Hylla is always one slice behind (ingest lags push). A local-overlay mode that reads uncommitted / untracked files and merges them into search results would cut the "Hylla is stale for my current diff" friction.

### Current-usage implications (see `HYLLA_WIKI.md`)
- Query by concept, never by library / import name.
- One Hylla shot per axis; on miss, switch to Grep rather than rephrasing the same call.
- Narrow `fields` per intent (see wiki decision tree).
- Graph-walk after any foothold — never re-keyword a missed search; navigate from any related node instead.
