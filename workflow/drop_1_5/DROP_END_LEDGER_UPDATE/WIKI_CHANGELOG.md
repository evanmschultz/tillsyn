# DROP_1_5_WIKI_CHANGELOG_ENTRY — drop-in-ready content for STEWARD

**Target**: `main/WIKI_CHANGELOG.md` append after merge. STEWARD writes.
**Tillsyn drop**: `9cbe395b-f6d8-45a9-816b-ebc3aa92c766` (under WIKI_CHANGELOG parent `1150cefe-...`).

---

## Drop 1.5 — TUI Refactor

### New user-facing surfaces

- **`v` keybinding → file viewer mode**. Opens the active plan-item's first Paths entry. Markdown (`.md`, `.markdown`) renders via glamour (shared with thread-view renderer). Code files render via chroma (dracula style, terminal256). Other extensions pass through plain text. Dotfiles refused with "Dotfiles not supported in v1" banner. Files above `max_bytes` (default 1MiB) refused with size banner — file is NOT fully read into memory.
- **`ctrl+d` keybinding → git diff pane**. Runs `git diff` between branch start SHA and HEAD scoped to paths derived from the active plan-item's `TaskMetadata.ResourceRefs`. Partition: `Tags[0]=="path"` or `"file"` → location unchanged; `"package"` → trailing-slash normalized for tree-prefix; else skipped. Chroma dracula highlighting on the patch. Empty ResourceRefs → whole-repo diff (conventional `git diff` behavior).
- **Path picker + file picker**. Modal input components layered on a shared file-picker core. Path picker accepts `path` tag; file picker accepts `file` tag — both write to the active plan-item's `TaskMetadata.ResourceRefs` via `appendResourceRefIfMissing`.

### New config surface

- `[tui.surfaces.file_viewer]` TOML section — `max_bytes` (default 1048576 = 1MiB), `dotfile_banner` (default "Dotfiles not supported in v1").

### New packages

- `internal/tui/gitdiff/` — `Differ` interface + `ExecDiffer` wrapping `git diff`; `Highlighter` interface + chroma-backed implementation. **Note**: lives under `internal/tui/` today; refinement 17 proposes extraction to a frontend-agnostic location for future web/electron adapter reuse.

### Key architectural note

The file-viewer's glamour renderer is shared with the thread-view renderer via pointer equality. `Model.threadMarkdown` changed from value (`markdownRenderer`) to pointer (`*markdownRenderer`) in P2-A; `NewModel` initializes the instance on heap, every surface holding a markdown renderer holds the same pointer. Don't construct a second renderer inside any new surface — pass the `Model.threadMarkdown` pointer through.

### Pattern reinforcement

- Rak-style MD-file coordination at `workflow/<drop>/<task>/` successfully carried the remainder of Drop 1.5 after Tillsyn MCP mutations failed (refinement 12). Pattern: per-task `PLAN.md` / `BUILDER_WORKLOG.md` / `BUILDER_QA_PROOF.md` / `BUILDER_QA_FALSIFICATION.md` / `CLOSEOUT.md`. Subagents edit only files their phase owns.
- Small parallel planner decomposition (≤N planners, each ≤1 surface/package, ≤15 min wall-clock) confirmed by dev as the system-as-designed cascade operating mode — NOT a workaround for subagent-compaction pain. Captured in `feedback_decomp_small_parallel_plans.md`.
