# Hylla — Usage Wiki

Living reference for using Hylla during Tillsyn work. Captures the **current** best practices given Hylla's current behavior. Derived from accumulated entries in `HYLLA_REFINEMENTS.md`. No changelog section — history lives in the findings file.

Update this wiki whenever a Hylla refinement lands that changes best practice (e.g. if the imports facet ships, the "skip Hylla for library names" rule below goes away and this wiki gets updated in the same slice).

## Query Hygiene

### Query by concept, not by library
Hylla currently misses import-name / library-name queries. Do not first-try `"glamour"`, `"bubbletea"`, `"cobra"`, `"lipgloss"` as keyword queries. Use a concept query instead: `"terminal markdown rendering"`, `"TUI component model"`, `"CLI command registration"`, `"style primitives"`. If that misses, fall back to Grep.
*Tracked as "imports facet" + "qualified-ident tokenization" in `HYLLA_REFINEMENTS.md`.*

### One Hylla shot per axis, then Grep
For library-name / import-ident lookups: one Hylla query. If it misses, switch to Grep. Do not burn turns rephrasing along a known-weak axis.

### Skip Hylla entirely for
- **Package / import names** (`glamour`, `bubbletea`, `lipgloss`) — use Grep.
- **Non-Go files** (markdown, TOML, YAML, magefile, SQL) — Hylla does not index them. Use Read / Grep / Glob.
- **Files changed since last ingest** — Hylla is stale for those. Use `git diff <snapshot-sha>..HEAD`.
- **Qualified-ident spot-checks where the file is already known** — just Read it.

### Narrow `fields` per intent
- Looking for a contract / interface doc → `fields=["docstring"]`.
- Looking for a behavior block or exact code string → `fields=["content"]`.
- Looking for a broad concept / cross-file theme → `fields=["summary"]`.
- Looking broadly → default (all three). Default returns more noise; use only when exploring.

### Graph-walk after any foothold
Once any Hylla call returns a related node, pivot to `hylla_graph_nav` and `hylla_refs_find` to expand. Re-running keyword search with new terms rarely improves; graph walk almost always does. A partial hit is a better starting point than a fresh keyword query.

### Check snapshot staleness at session start
Confirm the current ingest snapshot against the slice baseline. If the branch has moved since ingest, `git diff <snapshot-sha>..HEAD` to see what Hylla does not know yet. Note any stale files in scratch notes so you do not trust Hylla results for them.

## Search Tool Decision Tree

```
Need to find a Go symbol / package / behavior?
  ├─ Library / import name?         → Grep (Hylla currently misses)
  ├─ Concept / behavior query?      → hylla_search (hybrid); narrow `field` on the vector path
  ├─ Exact symbol name?             → hylla_search_keyword, fields=["content"]
  ├─ Cross-reference / call graph?  → hylla_refs_find + hylla_graph_nav
  ├─ Full node body after a hit?    → hylla_node_full
  ├─ Changed since ingest?          → git diff (Hylla is stale)
  └─ Non-Go file?                   → Read / Grep / Glob (Hylla does not index non-Go)
```

## Hylla Schema Gotchas

- `hylla_search` (hybrid / vector) takes `field` (**singular**). `hylla_search_keyword` takes `fields` (**plural**). Error messages do not currently disambiguate which param they reject — if you see `"field must be summary, content, or docstring"`, check which tool you called and which casing of `field(s)` you passed.
- `artifact_ref` is `github.com/<owner>/<repo>@<ref>` (e.g. `@main`). Always include it explicitly — defaulting is unreliable across sessions.

## Query-Hygiene Checklist for Agents

Every builder / QA / planner spawn prompt should embed these rules:

- **One-shot rule.** If the first Hylla query misses, switch search mode (vector → keyword → graph) **once**, then fall back to LSP / Grep / Read. Do not grind on the same call.
- **Narrow `fields`.** See the intent table above.
- **Graph-walk after foothold.** Any node hit → `hylla_graph_nav` / `hylla_refs_find` before considering the search complete.
- **Log misses.** Every fallback to LSP / Grep / Read gets a `## Hylla Feedback` entry in the agent's closing comment. No miss is still valid signal — write `None — Hylla answered everything needed.`

## Related Files

- `main/HYLLA_REFINEMENTS.md` — append-only log of misses + Hylla-product refinement candidates.
- `main/HYLLA_FEEDBACK.md` — per-slice aggregation of subagent-reported Hylla feedback.
- `main/CLAUDE.md` → "Code Understanding Rules" — canonical tool-order rules (this wiki supplements, does not replace).
