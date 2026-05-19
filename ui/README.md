# `ui/` — Wails + SolidJS + Astro Desktop Shell

The `ui/` subtree is the Tillsyn desktop app: a Wails v2 host (`ui/main.go`) that
embeds the Astro+SolidJS frontend at `ui/frontend/dist/` via `//go:embed` and
exposes Go services to JS through in-process method bindings. It is a peer to
`cmd/till` (CLI) and `internal/tui` (Bubble Tea TUI) — all three share the same
`internal/app.Service` against the same `.tillsyn/tillsyn.db`. The FE coexists
with the TUI; long-term it earns replacement on merit, not by decree.

## How to run

Two mage targets wrap the Wails CLI from the repo root:

- `mage ui-dev` — hot-reload Wails+Astro dev loop. Starts `wails dev` inside
  `ui/`, which launches the Astro dev server on `http://localhost:4321`, builds
  the Go host with the `wails` build tag, and opens a native WebView window
  with live-reload wired to both sides. Use this for day-to-day FE development.
- `mage ui-build` — production binary. Runs `wails build` inside `ui/`, which
  emits `ui/build/bin/Tillsyn.app/Contents/MacOS/Tillsyn` on macOS (and
  platform-equivalent paths on Linux/Windows). The binary embeds the built
  Astro output and links against the pinned `github.com/wailsapp/wails/v2`
  module — no external Wails runtime needed at launch time.

Both targets expect the dev machine has `wails` (Wails v2 CLI) and `pnpm` on
`$PATH`; see `REVISION_BRIEF.md` §1 "Hard Prerequisites" for the full
dev-machine prerequisite list. The FE-only CI gate `mage ci-ui` runs Vitest +
`astro build` and is what the QA pair exercises.

## Wiring (in-process Go bindings)

The FE talks to Go through in-process Go bindings, NOT through MCP. Wails'
codegen walks the methods on the `App` struct in `ui/main.go` and emits
JS-side wrapper functions reachable from the browser as
`window.go.main.App.<MethodName>`. The Go side constructs a real
`*app.Service` against the same SQLite DB and config the CLI uses (`config.Load`
+ `sqlite.Open`); IPC calls flow Wails IPC → `App` method → `*app.Service` →
`internal/adapters/storage/sqlite`. There is one binding in this drop —
`App.ListProjects() ([]ProjectDTO, error)` — and it is read-only this drop;
the FE exposes no mutations yet. Write operations (`CreateProject`,
`UpdateProject`, etc.) land in subsequent FE drops once the read path is
proven end-to-end.

For the locked architectural decisions, full acceptance criteria, the
out-of-scope list, and the resolved planner questions that landed this shape,
see REVISION_BRIEF.md (alongside this file at the drop root: `workflow/drop_fe_1_bootstrap/REVISION_BRIEF.md`).
