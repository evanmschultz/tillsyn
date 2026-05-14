# End-to-End Dogfood Fixes — Tracking

Issues surfaced during joint smoke-testing on the TILLSYN project itself. Dev exercises CLI/TUI; orchestrator (Claude Code) uses only Tillsyn MCP tools.

Status legend: `open` / `in-progress` / `fixed`.

**Status summary (2026-05-14):**
- E2E-1 — open (narrowed to TUI-only after E2E-4 fix verified CLI path works)
- E2E-2 — fixed (MCP/DB reconnect after session restart)
- E2E-3 — fixed-by-workaround (backfill via wired `till project update`; the original miss was due to pre-W2.D7 binary at first `till init`)
- E2E-4 — fixed (orch-direct cobra wiring for `till project update`; other W3 commands still need wiring)
- E2E-5 — open (`till project discover` output missing W2.D7 first-class fields)
- E2E-6 — open (TUI project view missing W2.D7 first-class fields; project-edit form missing/broken too)
- E2E-7 — open (W2.D5 `copyAgentFiles` overwrote substantive `.tillsyn/agents/{go,fe}/*.md` content with placeholders despite claimed idempotent skip; recovered via `git checkout HEAD --`)

---

## E2E-5 — `till project discover` output missing W2.D7 first-class fields

- **Status:** open
- **Surfaced:** 2026-05-14 — dev ran `till project discover --project-id 5d9b530c-...` after the backfill; output shows only name/id/slug/owner/archived. The four W2.D7 fields (`root_path`, `bare_root`, `language`, `groups`) are NOT in the discover summary.
- **Expected:** `till project discover` should include the new first-class fields the way `till project update`'s post-update output does (which DOES show them per W3.D1 absorption D — `writeProjectDetail` extension).
- **Suspected scope:** Discover-specific output path bypasses `writeProjectDetail`. Likely there's a `writeProjectReadiness` or similar helper that pre-dates W3.D1 absorption and never got the field extension.
- **Fix plan:** Locate the discover-output helper; mirror the `writeProjectDetail` row additions from W3.D1 absorption D.

---

## E2E-6 — TUI project view + edit missing W2.D7 first-class fields

- **Status:** open
- **Surfaced:** 2026-05-14 — dev opened TUI, navigated to project view; root path / bare root / language / groups not visible; cannot edit via TUI form.
- **Expected:** TUI project pane should display all W2.D7 fields AND support editing them via the project-edit form (with a path picker for root-path / bare-root and an enum picker for language).
- **Suspected scope:**
  - TUI rendering: `internal/tui/` project pane definition predates W2.D7 / W3.D1 absorption.
  - TUI edit flow: project-edit form either doesn't have rows for these fields, or has them but doesn't call `(*Service).UpdateProject` with the new `UpdateProjectInput` shape.
- **Fix plan:** Audit `internal/tui/` for the project view + edit forms; add display rows for the new fields; wire edit-form inputs into a `UpdateProject` call (mirror what `runProjectUpdate` does CLI-side).

---

## E2E-7 — `copyAgentFiles` overwrites existing agent files instead of skipping

- **Status:** open
- **Surfaced:** 2026-05-14 — dev ran `till init` against TILLSYN repo (which had committed W8 substantive content under `.tillsyn/agents/{go,fe}/*.md`). Output reported `agents copied  added=20 skipped=0` — overwrote all 20 W8 substantive files with embedded placeholder content. W8 substantive content (104-line `builder-agent.md` etc.) was REPLACED with 10-line placeholders.
- **Verified:** `wc -l` on working-tree file: 10 (placeholder). `git show HEAD:<path> | wc -l`: 104 (W8 substantive). Confirmed: working tree was clobbered.
- **Recovery this session:** `git checkout HEAD -- .tillsyn/agents/fe/ .tillsyn/agents/go/` — restored.
- **Expected per W2.D5 L2 spec:** "Idempotent: existing files at `<destDir>/.tillsyn/agents/<group>/<name>.md` are SKIPPED (not overwritten)."
- **Suspected scope:** W2.D5's `copyAgentFiles` does NOT actually check if the destination file exists before writing (or the check is inverted). The "added=20 skipped=0" output proves the skip predicate never fired for any of the 20 files even though every one existed on disk.
- **Fix plan:**
  - Read `copyAgentFiles` in `cmd/till/init_cmd.go` — verify the existence check.
  - Add `os.Stat(destPath); if err == nil { skipped++; continue }` guard before write.
  - Add a regression test: pre-create a file at destination, run copyAgentFiles, assert `skipped >= 1` AND file content unchanged.
  - This is a real data-safety bug; the W2.D5 test suite passed without catching it. Falsification gap.
- **Severity:** HIGH — silent data loss on `till init` re-run in a project that has hand-edited agent files. Dogfood-relevant since W4.D1's placeholder embedded content will replace any project-tier customizations.

---

## E2E-1 — Project `--root-path` / TUI path editing broken

- **Status:** open
- **Surfaced:** 2026-05-14 during initial smoke setup.
- **Symptom (dev report):**
  - `till init` does not accept a `--root-path` option (CLI ignores it).
  - `till project update` via TUI: project picker for path did not come up; typing the path did not save.
- **Expected:**
  - `till init` should let user set `RepoPrimaryWorktree` explicitly (today it auto-sets via `os.Getwd()` per W2.D7).
  - TUI project update should expose path editing with picker + manual entry, persisting via `(*Service).UpdateProject`.
- **Suspected scope:**
  - CLI: W2.D1's `initJSONPayload` shape has `Name`, `Groups`, `MCP` only — no `RepoPrimaryWorktree`. Today path is implicit from cwd.
  - CLI: W3.D1's `till project update --root-path /abs` was shipped + tested — needs verification it actually works against the current main/ project.
  - TUI: project picker for path doesn't surface or doesn't persist. TUI plumbing for project edit may not have been updated to call `(*Service).UpdateProject` with the new `RepoPrimaryWorktree` field.
- **Repro steps:** (to be filled in during smoke test)
- **Root cause:** (TBD)
- **Fix plan:** (TBD — cascade through Tillsyn itself if possible)

---

## E2E-2 — CLI/MCP database-pointer mismatch

- **Status:** open
- **Surfaced:** 2026-05-14 during joint smoke setup.
- **Symptom:**
  - Dev's `till init` reported "project DB created" — wrote to (suspected) `~/.tillsyn/tillsyn.db`.
  - Orchestrator's `mcp__tillsyn__till_project list` and `mcp__tillsyn-dev__till_project list` BOTH return only the stale `TILLSYN-TEST` project (id `95b0c77d-dbe3-4030-8147-caea70462691`, created 2026-05-12). The newly-created TILLSYN project is invisible to MCP.
- **Expected:**
  - CLI writes + MCP reads should converge on the same DB so joint dev/orch work sees a consistent state.
- **Suspected scope:**
  - MCP `claude mcp add tillsyn ...` was registered with a specific `--db` flag pointing to a workspace path (e.g., `.tillsyn/tillsyn.db` inside main/) or older install path.
  - CLI `till init` defaults to `$HOME/.tillsyn/tillsyn.db`.
  - Drift between them = orch and dev see different worlds.
- **Verification commands** (dev side):
  - `claude mcp list | grep tillsyn` — shows MCP server `--db` arg.
  - `ls -la ~/.tillsyn/tillsyn.db /Users/evanschultz/Documents/Code/hylla/tillsyn/main/.tillsyn/tillsyn.db` — confirms which DBs exist.
- **Root cause:** (TBD)
- **Fix plan:** (TBD — either update MCP registration to match CLI default, or `till init --home=<MCP-DB-parent>`. CLAUDE.md "Dev MCP Server" notes each worktree gets a unique MCP entry.)

---

## Joint smoke setup state

- 1. Orchestrator MCP access: ✅ tools loaded; CAN read `till.project` and other ops.
- 2. Project alignment: ✅ RESOLVED after session restart — MCP now sees TILLSYN (id `5d9b530c-b568-4830-9e16-058c957cfc05`).
- 3. Orchestrator auth claim: ⏳ pending E2E-3 + path correctness verification.

---

## E2E-4 — W3.D7 main.go cobra registration not shipped

- **Status:** open
- **Surfaced:** 2026-05-14 — dev ran `till project update --project-id ... --root-path ...` and got `Unknown flag: --project-id`.
- **Symptom:**
  - `till project update` is NOT a registered cobra subcommand.
  - Same likely for `till project delete/archive/restore/rename` (W3.D2 functions exist, cobra wiring missing).
  - Result: cobra falls back to `till project` parent which doesn't accept `--project-id`.
- **Expected:**
  - W3.D7 plan ships `main.go` cobra registration for all W3.D1-D6 commands.
- **Suspected root cause:**
  - W3.D7 was planned but never built in the autonomous run. The W3 serial chain ran D1 → D2 → D3 and stopped (we attempted §5.13 smoke after D3 since smoke passed without D4-D7).
  - W3.D1 + W3.D2 builders shipped function code + tests but did NOT include cobra registration (it was deferred to W3.D7).
  - W3.D3 absorbed its own cobra registration inline (for `actionItemCreateCmd`) — only working CLI surface today besides `init` and pre-existing commands.
- **Working CLI surfaces today:**
  - `till init` (W2.D1-D7, shipped)
  - `till project list` (pre-W3)
  - `till project get` (pre-W3)
  - `till action_item create` (W3.D3 with self-absorbed cobra)
  - `till action_item list/get` (pre-W3)
  - `till dispatcher run --dry-run` (Drop 4a)
- **Broken / missing CLI surfaces:**
  - `till project update` (W3.D1 — code present, cobra missing)
  - `till project delete/archive/restore/rename` (W3.D2 — code present, cobra missing)
  - `till template save/list/show/diff/restore` (W3.D4 — not shipped at all)
  - `till agents save/list/show/diff` (W3.D5 — not shipped at all)
  - `till agents bootstrap` (W3.D6 — not shipped at all)
- **Fix plan:**
  - Quick: orch-direct edit to `main.go` adding cobra wiring for `projectUpdateCmd` (mirror `actionItemCreateCmd` pattern from W3.D3). ~10 lines.
  - Full: ship W3.D7 as planned via cascade (registers ALL W3.D1-D6 commands).

---

## E2E-3 — `till init` not populating W2.D7 first-class fields (suspected pre-W2.D7 binary)

- **Status:** open
- **Surfaced:** 2026-05-14 after MCP reconnect, listing TILLSYN project shows all W2.D7 fields empty.
- **Symptom:**
  - `till init --json '{"name":"TILLSYN","groups":["go","fe"],"mcp":true}'` reported "project DB created."
  - But MCP list shows the project with:
    - `RepoPrimaryWorktree: ""` (W2.D7 should set to `os.Getwd()`)
    - `RepoBareRoot: ""` (W2.D7 should set via `git rev-parse --git-common-dir`)
    - `Language: ""` (W2.D7 should map `groups[0]="go"` → "go")
    - `Metadata` has no `groups` key (W2.D7 should set `Metadata.Groups = ["go","fe"]`)
- **Expected:** W2.D7 (commit `a4f4c25`) populates all four fields on `till init`.
- **Suspected root cause:** Dev's installed `till` binary at `$HOME/.local/bin/till` is OLDER than commit `a4f4c25`. `mage install` not run after W2.D7 landed.
- **Verification commands:**
  - `which till`
  - `ls -la $(which till)` (compare mtime against `git log -1 a4f4c25`)
- **Fix plan:**
  - If pre-W2.D7 binary: `mage install` then re-run till init in a fresh dir OR `till project update --root-path /Users/evanschultz/Documents/Code/hylla/tillsyn/main` to backfill the existing record.
  - If post-W2.D7 binary fails to populate: real bug in W2.D7's `createProjectDBRecord` not caught by tests — investigate.
