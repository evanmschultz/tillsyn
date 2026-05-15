# End-to-End Dogfood Fixes — Tracking

Issues surfaced during joint smoke-testing on the TILLSYN project itself. Dev exercises CLI/TUI; orchestrator (Claude Code) uses only Tillsyn MCP tools.

Status legend: `open` / `in-progress` / `fixed`.

**Status summary (2026-05-15):**
- E2E-1 — open (narrowed to TUI-only after E2E-4 fix verified CLI path works)
- E2E-2 — fixed (MCP/DB reconnect after session restart)
- E2E-3 — fixed-by-workaround (backfill via wired `till project update`; the original miss was due to pre-W2.D7 binary at first `till init`)
- E2E-4 — fixed (orch-direct cobra wiring for `till project update`; other W3 commands still need wiring)
- E2E-5 — fixed 2026-05-15 (commit `edfa130`; extended `writeProjectReadiness` with 6 W2.D7 rows mirroring `writeProjectDetail` order)
- E2E-6 — fixed 2026-05-15 (commit `1a932e8`; plan-QA falsification caught the real bug — TUI `startProjectForm` at `model.go:4763` read TUI cache instead of `project.RepoPrimaryWorktree`; fix prefers canonical with cache fallback)
- E2E-7 — fixed (root cause was `detectFLATLayout` returning a `rm -rf .tillsyn/agents/` remediation message that destroyed legitimate subdir content; replaced with surgical `cleanFLATLayout` that auto-removes only root-level `.md` files and preserves `<group>/` subdir content)
- E2E-8 — fixed (commits `ba58ba7` + `49fa802` + `ed5f29d`; threaded `wait_timeout` through handler→adapter→app→`createAuthRequestLive`; added 5 tests covering Approval+Deny+Cancel wake + Pending-timeout + Zero-wait regression + MCP wire-layer integration; schema description on `handler.go:136` updated to enumerate honored ops `create` + `claim`)
- E2E-9 — closed 2026-05-15 as not-a-bug (framing was wrong: the auth+lease system already scopes every MCP op to a specific `project_id` via session_id + session_secret + agent_instance_id + lease_token, all bound at claim time. Two orchs in two projects sharing one global MCP server + global DB cannot step on each other because their tokens are bound to different project scopes. Project-local binary/`.mcp.json` was a non-solution to a non-problem.)
- E2E-10 — fixed 2026-05-15 (commit `8ea3822`; Option A auto-clear guard at `service.go:1337` with 3-subtest regression covering top-level / legitimate-parent / not-found branches)

---

## E2E-5 — `till project discover` output missing W2.D7 first-class fields

- **Status:** fixed 2026-05-15 (commit `edfa130`)
- **Surfaced:** 2026-05-14 — dev ran `till project discover --project-id 5d9b530c-...` after the backfill; output shows only name/id/slug/owner/archived. The four W2.D7 fields (`root_path`, `bare_root`, `language`, `groups`) are NOT in the discover summary.
- **Expected:** `till project discover` should include the new first-class fields the way `till project update`'s post-update output does (which DOES show them per W3.D1 absorption D — `writeProjectDetail` extension).
- **Suspected scope:** Discover-specific output path bypasses `writeProjectDetail`. Likely there's a `writeProjectReadiness` or similar helper that pre-dates W3.D1 absorption and never got the field extension.
- **Fix plan:** Locate the discover-output helper; mirror the `writeProjectDetail` row additions from W3.D1 absorption D.

---

## E2E-6 — TUI project view + edit missing W2.D7 first-class fields

- **Status:** fixed 2026-05-15 (commit `1a932e8`)
- **Resolution note:** Plan-QA falsification caught that the planner's framing was wrong — the W2.D7 fields ARE pre-populated in `startProjectForm` (`model.go:4766-4772`), rendered in `projectFormBodyLines` (`model.go:17417-17421`), and wired into submission (`model.go:11704-11781`). The REAL bug was at `model.go:4763`: `RepoPrimaryWorktree` was being read from `m.projectRoots[slug]` (TUI-local cache) instead of `project.RepoPrimaryWorktree`. For projects backfilled via `till project update --root-path` (per the E2E-3 workaround), the canonical field was populated but the form showed the stale cache value. Fix prefers `project.RepoPrimaryWorktree`, falls back to the cache when canonical is empty.
- **Surfaced:** 2026-05-14 — dev opened TUI, navigated to project view; root path / bare root / language / groups not visible; cannot edit via TUI form.
- **Expected:** TUI project pane should display all W2.D7 fields AND support editing them via the project-edit form (with a path picker for root-path / bare-root and an enum picker for language).
- **Suspected scope:**
  - TUI rendering: `internal/tui/` project pane definition predates W2.D7 / W3.D1 absorption.
  - TUI edit flow: project-edit form either doesn't have rows for these fields, or has them but doesn't call `(*Service).UpdateProject` with the new `UpdateProjectInput` shape.
- **Fix plan:** Audit `internal/tui/` for the project view + edit forms; add display rows for the new fields; wire edit-form inputs into a `UpdateProject` call (mirror what `runProjectUpdate` does CLI-side).

---

## E2E-10 — `till_action_item operation=create` rejects valid top-level create when `parent_id == project_id`

- **Status:** fixed 2026-05-15 (commit `8ea3822`)
- **Resolution:** Chose Option A (auto-clear `parent_id` when it equals `project_id`). Guard inserted at `internal/app/service.go:1333-1339`, BEFORE the existing `if strings.TrimSpace(in.ParentID) != ""` block at line 1340 (placement noted by plan-QA falsification). When the guard fires, `in.ParentID` is set to `""` and the call falls through the project-scope path to top-level item creation. `TestCreateActionItemParentIDEqualsProjectID` adds 3 subtests covering top-level / legitimate-parent / not-found branches. Build-QA falsification flagged one acceptable refinement: no MCP wire-layer test for the `parent==project` case (service-layer coverage sufficient because the MCP adapter is a thin trim-and-forward shim).
- **Surfaced:** 2026-05-14 during the E2E-1+E2E-6 cascade setup; repro'd post-restart on the production `tillsyn` MCP.
- **Symptom:**
  - Caller passes `parent_id` = the project's own UUID (mistakenly treating project as the implicit parent for top-level action items).
  - Server returns `not_found: authorize mutation: not found / not found`. The error message is doubly nested "not found" with no hint that `parent_id` is the wrong shape.
  - When `parent_id` is OMITTED, the same `create` call succeeds (creates a top-level action item under the project). The `ParentID` field comes back empty in the response, confirming top-level placement is the correct shape.
- **Repro:**
  ```
  # FAILS — parent_id set to project_id:
  mcp__tillsyn__till_action_item(
      operation="create",
      project_id="<UUID>",
      parent_id="<UUID-same-as-project_id>",  # <-- THIS
      kind="build", structural_type="drop", state="todo",
      title="...", paths=[...], packages=[...],
      session_id="...", session_secret="...",
      agent_instance_id="...", lease_token="..."
  )
  # -> not_found: authorize mutation: not found / not found

  # SUCCEEDS — parent_id omitted, same call otherwise:
  # ... same args minus parent_id ...
  # -> action_item created with ParentID=""
  ```
- **Suspected scope:** Server-side mutation authorizer likely does `repo.GetActionItem(parent_id)` to check scope authorization. When parent_id matches project_id (which lives in the `projects` table, not `action_items`), the lookup returns no rows. The error is then surfaced as the authorize-mutation failure, not as the more accurate "parent_id must reference an action_item, not a project."
- **Why it matters:**
  - Confusing diagnostic. Cost me ~15 minutes of red-herring debugging (suspected capability_lease layer was missing, then auth_context_id wasn't being passed correctly, etc).
  - Inconsistent with the parent_id schema description which says "Optional parent action-item id for operation=create" — the natural reading is that omitting it means top-level. But callers who think "project IS the parent for top-level items" are punished with a confusing error instead of either an actionable rejection OR auto-treating it as top-level.
- **Fix options:**
  - **A (preferred)**: if `parent_id == project_id`, treat as top-level (auto-clear parent_id internally). This matches the natural caller mental model and removes the foot-gun.
  - **B**: return an explicit error like `invalid_request: parent_id must reference an action_item, not the project; omit parent_id for top-level items under the project`. More restrictive but unambiguous.
- **Severity:** MEDIUM — workaround is trivial once known; but the confusing diagnostic costs every first-time caller 10+ minutes.

---

## E2E-8 — `till_auth_request operation=create` does not honor `wait_timeout` (no push to orch on approval)

- **Status:** fixed 2026-05-15
- **Surfaced:** 2026-05-14 — twice in the same session (both `tillsyn-dev` and `tillsyn` MCP endpoints, post-restart).
- **Symptom:**
  - Orchestrator calls `till_auth_request operation=create` with `wait_timeout: 15m` (and again with `wait_timeout` omitted on a separate try — same outcome).
  - Server returns IMMEDIATELY with `state: "pending"`, a fresh request_id, and a resume_token. Response shape matches "request just created" — NOT "blocked until approved, here's the approved state."
  - Dev approves via TUI. Orchestrator receives no notification — has to wait for the dev to type "approved, go ahead" in chat before claiming. This is the exact `/loop`-hack pattern the LiveWait broker exists to replace.
- **Expected:**
  - Per tool schema (`mcp__tillsyn__till_auth_request` → `wait_timeout: "Optional how long to wait for human approval before returning the current request state, for example 30m"`): the call should block up to `wait_timeout` and return with `state: approved` once the dev approves in the TUI. The `LiveWaitEventAuthRequestResolved` broker channel exists for exactly this purpose.
- **Suspected scope:** Server-side `operation=create` handler does NOT subscribe to `LiveWaitEventAuthRequestResolved` for the new request_id, or doesn't honor `wait_timeout` on the `create` code path. Two ways to verify:
  1. Reproduce: call `till_auth_request operation=get` with `request_id` + `wait_timeout: 5m` BEFORE approval; if `get` also returns `pending` immediately rather than blocking, the broker isn't wired into either operation. If `get` DOES block, the bug is `create`-specific.
  2. Code-read the handler in `internal/adapters/server/mcpapi/` (or wherever `till_auth_request` is dispatched) and trace whether `wait_timeout` is read + whether `LiveWaitBroker.Wait` is called on it.
- **Why it matters:** Without this working, every orchestrator session needs the dev to manually type "approved" in chat after TUI approval. That's the /loop-hack workaround. Tillsyn's own auth flow is the most-used customer of the LiveWait broker; if `wait_timeout` doesn't fire here, the broker isn't earning its keep on this surface.
- **Reproduction:**
  ```
  # 1. Orchestrator session:
  mcp__tillsyn__till_auth_request(
      operation="create",
      path="project/<project-uuid>",
      principal_id="TEST_ORCH",
      principal_name="TEST_ORCH",
      principal_type="agent",
      principal_role="orchestrator",
      client_id="claude-code-orchestrator",
      reason="test wait_timeout",
      requested_ttl="1h",
      wait_timeout="2m"
  )

  # 2. Dev approves in TUI within 30s.

  # 3. EXPECTED: call returns at t=30s with state="approved".
  #    ACTUAL: call returns at t=0s with state="pending". Dev approval at t=30s lands in DB but
  #            doesn't unblock the JSON-RPC reply. Orch sits idle until dev types "approved" in chat.
  ```
- **Fix plan:**
  - Audit `till_auth_request operation=create` handler. Confirm `wait_timeout` is parsed; confirm `LiveWaitBroker.Wait(ctx, LiveWaitEventAuthRequestResolved, requestID, 0)` is called when `wait_timeout > 0`.
  - Add integration test: create request with `wait_timeout: 30s`, spawn goroutine that approves at t=2s, assert call returns at t≈2s with `state: approved` (NOT at t=0s with `state: pending`).
  - Same test for `operation=get` with `wait_timeout` parameter — verify polling consumers can also block on approval.
  - Whichever operations don't honor `wait_timeout`: either fix the handler OR remove `wait_timeout` from the schema for that op so callers stop expecting block behavior.
- **Severity:** HIGH — silently degrades the only documented push-to-orch path in Tillsyn's stack. Forces every cascade dispatch to fall back to chat-relay coordination.

---

## E2E-9 — `.mcp.json` invokes global `till` (CLOSED — not a bug)

- **Status:** closed 2026-05-15 as not-a-bug.
- **Resolution:** The original framing assumed the global `till` binary + global DB created a cross-project leakage risk that needed a project-local binary fix. On review the auth+lease system already provides the scoping: every guarded MCP mutation requires `session_id` + `session_secret` + `agent_instance_id` + `lease_token`, all four bound to a specific project at claim time (`path: project/<uuid>/...`). The server rejects any call whose target resource's `project_id` doesn't match the bound auth scope. So one global binary + one global DB CAN serve multiple projects safely — different orchs hold tokens bound to different project paths and can't reach each other's state. No code change required. Closing this ticket; the original REFINEMENTS.md parking-lot entry can also be removed.

---

## E2E-7 — `till init` clobbered subdir agent content on FLAT-leftover repos

- **Status:** fixed (2026-05-14 same session)
- **Surfaced:** 2026-05-14 — dev ran `till init` against TILLSYN repo (which had committed W8 substantive content under `.tillsyn/agents/{go,fe}/*.md`). Output reported `agents copied  added=20 skipped=0` — overwrote all 20 W8 substantive files with embedded placeholder content. W8 substantive content (104-line `builder-agent.md` etc.) was REPLACED with 10-line placeholders.
- **Verified:** `wc -l` on working-tree file: 10 (placeholder). `git show HEAD:<path> | wc -l`: 104 (W8 substantive). Working tree was clobbered.
- **Recovery this session:** `git checkout HEAD -- .tillsyn/agents/fe/ .tillsyn/agents/go/` — restored.
- **Investigation (fix-session, 2026-05-14):**
  - Wrote two regression tests for `copyAgentFiles` skip-on-exists invariant (one unit + one e2e through `run()`). Both PASS against current source — `copyAgentFiles` correctly skips pre-existing user content. So `copyAgentFiles` is NOT the bug.
  - Git status at session start showed 12 FLAT files staged for deletion at `.tillsyn/agents/*.md` (cleaned up in commit `789a494`). These were present at smoke time.
  - Root cause traced to `detectFLATLayout` at `init_cmd.go:406-425`: on detecting FLAT-layout `.md` files at the root of `.tillsyn/agents/`, it returned the error message `"FLAT agent layout detected ... Remove it and re-run: rm -rf %s && till init"` — instructing the user to delete the entire `.tillsyn/agents/` directory, which destroys legitimate `<group>/` subdir content alongside the flat leftovers.
  - Smoke-time chain: (1) dev had FLAT files from earlier sessions, (2) `till init` failed with the `rm -rf` remediation, (3) dev ran `rm -rf .tillsyn/agents/`, (4) re-ran `till init`, (5) `copyAgentFiles` correctly added embedded placeholders to a now-empty dir (`added=20 skipped=0`). The W8 content was lost in step 3, not step 5.
- **Fix shipped:**
  - Replaced `detectFLATLayout` with `cleanFLATLayout` (`cmd/till/init_cmd.go`): surgically removes only root-level `.md` files from `.tillsyn/agents/`, preserves `<group>/` subdirs. Returns the list of removed basenames.
  - `runInitPipeline` calls `cleanFLATLayout` in the pre-flight slot and adds a Laslig audit row `"removed legacy flat agents"` when the list is non-empty.
  - Regression tests added in `cmd/till/init_cmd_test.go`:
    - `TestCopyAgentFiles_SubdirPerGroup/preserves_user_modified_existing_content` — unit guard for skip-on-exists with non-embedded content (closes the original falsification gap claimed in the smoke note).
    - `TestCopyAgentFiles_SubdirPerGroup/end_to_end_preserves_user_subdir_content` — e2e through `run()` with pre-seeded user content in subdir.
    - `TestRunInitPipeline_FLATCleanup/flat_layout_auto_cleaned` — flat files at root are auto-removed and surfaced in the audit notice.
    - `TestRunInitPipeline_FLATCleanup/flat_plus_subdir_preserves_subdir_content` — exact smoke-time scenario: flat files + user subdir content. Flat removed, subdir preserved byte-for-byte.
    - `TestRunInitPipeline_FLATCleanup/clean_state_no_flat_layout` updated to assert the audit row does NOT appear on clean state.
  - `mage ci` green (405 cmd/till tests pass; coverage 77.6%).
- **Severity:** HIGH — silent data loss. Closed.

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
