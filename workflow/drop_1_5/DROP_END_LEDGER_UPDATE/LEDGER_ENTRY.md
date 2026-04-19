# DROP_1_5_LEDGER_ENTRY — drop-in-ready content for STEWARD

**Target**: `main/LEDGER.md` append after merge. STEWARD writes.
**Tillsyn drop**: `6b627c5c-7880-4801-8d91-4f61b227532b` (under LEDGER parent `da97aac2-...`).

---

## Drop 1.5 — TUI Refactor (File Picker, Path Picker, File Viewer, Git Diff Surface)

- **Closed**: 2026-04-19
- **Action-item ID**: `b54471d1-...` (drop root; stays `in_progress` pending refinement 12 unblock for manual TUI flip — audit trail lives in `workflow/drop_1_5/`)
- **Branch**: `drop/1.5` → merged to `main` on 2026-04-19
- **Ingest snapshot**: *TO BE FILLED post-merge after `hylla_ingest` run — currently pending dev merge*
- **Cost (this run / lineage-to-date)**: *TO BE FILLED post-merge*
- **Node counts (total / code / tests / packages)**: *TO BE FILLED post-merge*
- **Orphan delta**: *TO BE FILLED post-merge*
- **Refactors**: `internal/tui/model.go` — `threadMarkdown markdownRenderer` → `*markdownRenderer` (heap-stable pointer for shared glamour renderer across thread + file-viewer surfaces).

### Description

Drop 1.5 delivered four user-facing TUI surfaces on `drop/1.5` branch: (1) generic file-picker core + path-picker variant (P3-A), (2) file-tagged specialization layering on the core (P3-B), (3) ctrl+d-triggered git diff pane backed by a new frontend-agnostic `internal/tui/gitdiff/` package (P4-T1 through P4-T4), and (4) v-triggered file viewer mode with glamour-shared markdown rendering + chroma syntax highlighting (P2-A). ResourceRef `Tags[0]` convention extended to `"package"` in P4-T4 for tree-prefix diff scoping. File viewer shares the Model's existing glamour renderer via pointer equality to avoid double-instantiation.

### Commits (10 on `drop/1.5` → `main`)

- `676ab96` docs(claude): adopt section 0 semi-formal reasoning shape
- `9042f3b` docs: update CLAUDE.md
- `ae53344` feat(tui): add file-picker core with path-picker variant [P3-A]
- `e6e0038` feat(tui): add file-picker file-tagged specialization [P3-B]
- `b5c4b5c` feat(tui): add gitdiff package with Differ interface and exec-based implementation [P4-T1]
- `a52b4c4` feat(tui): add chroma-backed highlighter for gitdiff patches [P4-T2]
- `0e22cdf` feat(tui): wire gitdiff Differ+Highlighter into ctrl+d diff mode [P4-T3 R1]
- `60b6fc5` style(tui): gofumpt diff_mode files for ci parity [P4-T3 R2 fixup]
- `e8914fc` feat(tui): resolve diff paths from active task resourcerefs [P4-T4]
- `af2a69c` feat(tui): add file viewer mode on v keybinding [P2-A]

### Notable file additions (18 new, +4098 lines in drop scope)

- `internal/tui/diff_mode.go` (+444) + `_test.go` (+684)
- `internal/tui/file_picker_core.go` (+360) + `_test.go` (+390) + keymap / render / specialization files
- `internal/tui/file_viewer_mode.go` (+284) + `_renderer.go` (+138) + `_test.go` (+310)
- `internal/tui/gitdiff/` new package: `differ.go`, `exec_differ.go`, `highlighter.go` + corresponding `_test.go` files (+873 total)
- `internal/config/` extended with `TUIConfig` / `TUISurfacesConfig` / `FileViewerConfig` structs

### CI

- Final CI run: `24616918555` (macos-latest) — green in 52s for `af2a69c`.
- `internal/tui` coverage: 70.4% (above 70% floor).
- `internal/config` coverage: 76.8%.
- `mage ci`: 1334 tests green, all 20 packages at/above 70% floor.

### Notable IDs

- P3-A build-drop (closed pre-MCP-block): committed `ae53344`.
- P3-B build-drop (closed pre-MCP-block): committed `e6e0038`.
- P4-T1 / P4-T2 build-drops (closed pre-MCP-block): committed `b5c4b5c` / `a52b4c4`.
- P4-T3 build-drop `e103bc94-2946-4abe-a5b0-1f2e125ebc53`: committed `0e22cdf` + `60b6fc5`; parent terminal state hit refinement 13 (parent moved to done while QA children in todo — 30s window).
- P4-T4 build-drop `7aec9fcf-d9f3-482d-80b9-9377785c305a`: committed `e8914fc`; 2 rounds (added `resolvePaths()` method + package-first dedup test in R2).
- P2-A build-drop `8020fd8a-4c29-402d-9eab-da72cf7fcebf`: committed `af2a69c`; single round.

### Unknowns forwarded

- Drop root `b54471d1-...` stays in `in_progress` until refinement 12 (MCP "invalid scope type") is fixed — manual TUI flip by dev or MCP unblock required to close. Audit trail complete in `workflow/drop_1_5/` Rak MDs + this ledger entry.
