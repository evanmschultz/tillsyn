# Drop 4c — Master PLAN + SKETCH QA Falsification (Round 2)

**Reviewer:** plan-qa-falsification (subagent).
**Review date:** 2026-05-04.
**Round:** 2 (R1 verdict was NEEDS-REWORK — `workflow/drop_4c/PLAN_QA_FALSIFICATION.md`).
**Targets:**
- `workflow/drop_4c/PLAN.md` (master, with §5 canonical-ID table, reconciliation block, droplet splits, metadata strategy lock).
- `workflow/drop_4c/SKETCH.md` (post-no-command-field rework).
- `workflow/drop_4c/F7_CORE_PLAN.md` (16 droplets — NOT revised post-R1).
- `workflow/drop_4c/F7_17_CLI_ADAPTER_PLAN.md` (11 droplets — NOT revised post-R1).
- `workflow/drop_4c/F7_18_CONTEXT_AGG_PLAN.md` (6 droplets).

**Mode:** read-only adversarial review. No code edits, no plan edits. Counterexamples route back to orchestrator.

**Methodology:** R2 brief defines three attack groups. A-group verifies R1 fix completeness. B-group introduces new architecture-change attacks. C-group confirms R1 NITs are still NITs (or worse).

---

## Summary Table

| # | Attack Vector | Verdict |
|---|---|---|
| A1 | #5 env baseline expansion airtight? | NIT |
| A2 | #6 command-removal complete? | **CONFIRMED** |
| A3 | #2 cross-plan ID table exhaustive? | **CONFIRMED** |
| A4 | #11 droplet splits propagated to sub-plans? | **CONFIRMED** |
| A5 | #15 JSON-blob lock complete? | **CONFIRMED** |
| B1 | claude binary discovery (PATH inheritance) | NIT |
| B2 | per-binding vendored binary blocker? | NIT |
| B3 | PATH-trust security model documented? | NIT |
| B4 | `Manifest.CLIKind` race with F.7-CORE F.7.1 | **CONFIRMED** (subsumed by A3) |
| B5 | F.7.5b/c grant-freshness across in-flight spawns | NIT |
| B6 | Metadata JSON blob 4a/4b consumer compat | REFUTED |
| C1 | R1 #1 drop-split risk still NIT? | NIT (unchanged) |
| C2 | R1 #3 DAG cycles still none? | NIT (unchanged) |
| C3 | R1 #4 out-of-scope leakage still NIT? | NIT (unchanged) |

**Final verdict:** **NEEDS-REWORK.**

The dev's architectural decisions (proxy/TLS env additions, `command` field removed entirely, canonical-ID mapping table, droplet splits, metadata-blob lock) are correct and close R1's critical attacks at the master-PLAN level. **But the sub-plans were never revised to reflect any of these decisions.** F.7.17 sub-plan still has ~25 references to `command`, `args_prefix`, denylist, regex validators. F.7-CORE still has the monolithic F.7.3 + F.7.5 droplets, the "JSON blob OR new columns; pick lower-friction" ambiguity, and `Manifest` ownership of the `CLIKind` field. The master PLAN admits this explicitly at §4 line 69 ("must be revised") but does not gate dispatch on the revision happening.

The four CONFIRMED counterexamples (A2, A3, A4, A5) all trace to the same root cause: **the dev decisions live in the master PLAN; the per-droplet acceptance criteria live in the sub-plans; builders read the sub-plans.** Until the sub-plans are revised (or each affected droplet's spawn prompt explicitly carries the master's overrides), builders will implement the old, R1-rejected design.

This is not a "plan is wrong" failure — it's a "plan is split across files and only one half got updated" failure. Surgical fix; the master PLAN's content is good.

---

## A1 — #5 env baseline expansion airtight?

**Verdict:** NIT.

**Premises:** R1 #5 said the 9-var closed baseline missed proxy + TLS-cert vars and possibly `TERM`/`SHELL`/`NODE_OPTIONS`. Dev decision: add proxy + TLS, defer terminal/runtime vars until concrete dogfood breakage.

**Evidence:**
- Master PLAN §3 L4 (line 43) lists the expanded baseline: `PATH, HOME, USER, LANG, LC_ALL, TZ, TMPDIR, XDG_CONFIG_HOME, XDG_CACHE_HOME` PLUS `HTTP_PROXY, HTTPS_PROXY, NO_PROXY, http_proxy, https_proxy, no_proxy, SSL_CERT_FILE, SSL_CERT_DIR, CURL_CA_BUNDLE`. 18 vars baseline plus per-binding `env` allow-list.
- SKETCH §F.7.17 lines 156-159 mirrors the same set.
- Cross-check `internal/app/git_status.go:146-156` (R1 reference): `filteredGitEnv` carries hundreds of vars by inheriting `os.Environ() − GIT_*`. Master PLAN's L4 deliberately inverts to closed-allowlist. Difference is the architectural choice, not a defect.

**Trace or cases:**
- Corporate-network adopter with `HTTPS_PROXY` set in shell launches `till dispatcher run` → orchestrator's PATH inherits the proxy var → spawn `cmd.Env` includes the proxy via L4 baseline → claude reaches the API. WORKS without per-binding `env`.
- Corporate-network adopter with custom CA bundle at `/etc/ssl/corp-ca.pem` and `SSL_CERT_FILE=/etc/ssl/corp-ca.pem` → spawn inherits via L4 baseline → claude TLS handshake works. WORKS.
- Edge: `HTTP_PROXY` (uppercase) AND `http_proxy` (lowercase) BOTH in shell with different values → both forwarded → claude follows POSIX precedence. NOT addressed in PLAN, but harmless because Go's `net/http` reads them with documented precedence; this is OS-level not Tillsyn's problem.

**But two residual gaps:**

1. **`F7_17_CLI_ADAPTER_PLAN.md:61` (L6) is stale.** The sub-plan's locked-decisions table line 61 lists ONLY the 9-var baseline — no proxy, no TLS-cert vars. A builder reading the sub-plan to implement the closed-baseline list constructs the wrong set. NIT-not-CONFIRMED only because A2 covers the broader sub-plan staleness; A1's specific manifestation is part of A2's blast radius.

2. **`SHELL` and `TERM` deferral risk pinned but not tested.** Dev's decision to skip them "until concrete dogfood breakage" is fine, but the failure mode — claude emits raw ANSI codes into `stream.jsonl`, parser breaks — wouldn't surface until Drop 5 dogfood. Recommendation already in R1: F.7.17 claudeAdapter test (currently F.7.17.3 in the sub-plan) should run a smoke-test against a real `claude --bare` headless call with the L4 baseline, not just argv-shape assertion. Not a CONFIRMED counterexample because the dev explicitly accepted this deferral.

**Conclusion:** NIT. The master PLAN's expansion correctly resolves R1 #5 for the dev's accepted scope. The sub-plan's staleness rolls into A2's CONFIRMED finding. The deferred-runtime-vars risk is acceptable given dev sign-off.

**Unknowns:** None — deferral is an accepted unknown, not a new one.

---

## A2 — #6 command-removal complete?

**Verdict:** **CONFIRMED.** The architectural decision lives in the master PLAN + SKETCH; the F.7.17 sub-plan still has the entire dropped surface alive.

**Premises:** Dev decision: drop `command` and `args_prefix` fields entirely. No denylist. No regex. No marketplace install confirmation. Master PLAN §3 L2 (line 41) + SKETCH line 151 + line 237 capture this. The F.7.17 sub-plan must be cleansed of every residual mention OR a clear authority-override directive must accompany every droplet spawn.

**Evidence (rg query on `F7_17_CLI_ADAPTER_PLAN.md`):**

Master PLAN §4 line 69: "Source: `workflow/drop_4c/F7_17_CLI_ADAPTER_PLAN.md` — **must be revised** to reflect the dropped `command` / `args_prefix` fields. Schema-1 now adds only `Env []string` + `CLIKind string`; the regex + denylist + token validators disappear; the marketplace install-time paper-spec disappears (no `command[0]` to confirm)."

But the sub-plan was NOT revised. Residual references:

| Line | Content |
|---|---|
| 15 | "`Command []string`, `ArgsPrefix []string`, `Env []string`, `CLIKind string` on `templates.AgentBinding` (Schema-1 of the F.7 wave)." — Scope claim. |
| 16 | "`validateAgentBindingCommandTokens` (regex literal pinned + closed shell-interpreter denylist)." — In-scope deliverable. |
| 56 | "L1 \| `command` is argv-list (`[]string`), never a string. No shell parsing, no `sh -c`. \| SKETCH §F.7.17 line 152" — Locked decision. |
| 58 | "L3 \| Closed `command[0]` shell-interpreter denylist: `{sh, bash, zsh, ...}`" — Locked decision. |
| 168 | "(add four fields + denylist constant + new `CLIKind` type alias)." — F.7.17.1 file-edit list. |
| 176 | "`AgentBinding` gains four exported fields with TOML tags `command`, `args_prefix`, `env`, `cli_kind`." — F.7.17.1 acceptance. |
| 178-186 | `validateAgentBindingCommandTokens` regex literal + denylist constant — F.7.17.1 acceptance. |
| 196-207 | 12 happy/reject test scenarios for `command` validation — F.7.17.1. |
| 219 | "A1.a: closed shell-interpreter denylist as an explicit constant + tests asserting EVERY entry rejects." — Falsification mitigation. |
| 316-317 | "Argv with `args_prefix = ["--profile", "dev"]`" — F.7.17.3 test. |
| 624-642 | Entire **F.7.17.10 Marketplace install-time confirmation** droplet — supposed to disappear per master PLAN §4 line 69 but still present, fully specified, with acceptance criteria for the dropped feature. |
| 716 | References "A1.a (sh -c bypass + denylist)" — falsification source. |

That's 25+ residual references across in-scope statements, locked decisions, file-edit lists, acceptance criteria, test scenarios, and an entire dropped droplet (F.7.17.10) still living in the per-droplet decomposition.

**Trace or cases:**
- A builder spawned to implement F.7.17.1 from `F7_17_CLI_ADAPTER_PLAN.md` line 159 will: (a) add `Command []string` + `ArgsPrefix []string` fields to `AgentBinding`; (b) implement `validateAgentBindingCommandTokens` with the regex literal + denylist constant; (c) add 12 unit tests for command validation. All four are explicitly reverted by dev decision.
- A builder spawned for F.7.17.10 (marketplace paper-spec) will write `docs/architecture/cli_adapter_seam.md` containing a "Marketplace Install-Time Confirmation" section that the dev decided is no longer needed.
- A builder spawned for F.7.17.3 (claudeAdapter) will accept `args_prefix` and `command` fields on `BindingResolved` (line 255 declares them) — these don't exist anymore, so the builder either invents them or fails.

**Conclusion:** CONFIRMED. The sub-plan revision is a hard-prerequisite for dispatch, not a future cleanup. Surgical fixes required:

1. **Cleanse `F7_17_CLI_ADAPTER_PLAN.md`** — re-author L1, L2, L3 (or remove them entirely); rewrite F.7.17.1 acceptance to the slim two-field schema; remove F.7.17.10 entirely; update F.7.17.2 `BindingResolved` field list to exclude `Command` and `ArgsPrefix`; update F.7.17.3 test scenarios.
2. **Net droplet count update**: master PLAN §4 line 68 says "~10 droplets, was 11; -1 from dropping marketplace install" — sub-plan still has 11 droplets, 4c.F.7.17.1 through 4c.F.7.17.11. Sub-plan must drop one (the marketplace one) and renumber.
3. **Cross-reference verification**: F.7-CORE F.7.3 (line 267) declares as hard-prereq "F.7.17 schema-1 (per-binding `command`, `args_prefix`, `env`, `cli_kind`)." This is wrong post-decision — must be `(env, cli_kind)`. Same for F.7-CORE F.7.2 line 252.

The master PLAN's L2 + SKETCH §F.7.17 lines 151, 237 carry the right design. The sub-plans need a revision pass that aligns to master before dispatch.

**Unknowns:** Whether the orchestrator intends a single sub-plan-revision droplet ("4c.0 — sub-plan reconciliation pass") to land first, or to author a `PLAN_RECONCILIATION.md` (referenced at master PLAN §5 line 156) that sits alongside as authoritative override. Either path closes A2; neither is currently scheduled in the master PLAN's DAG.

---

## A3 — #2 cross-plan ID table exhaustive?

**Verdict:** **CONFIRMED.** The canonical-ID table at master PLAN §5 lines 80-99 is correct and resolves R1 #2 at the master level. But the sub-plans contain conflicting single-owner claims that the table does not auto-overwrite.

**Premises:** Walk every cross-plan reference in the three sub-plans against the canonical-ID table. Anything still referenced by old name OR with conflicting ownership claim is a CONFIRMED counterexample.

**Evidence:**

Canonical table (master PLAN §5):
- F.7.18.2 = sole owner of initial `Tillsyn` struct (`MaxContextBundleChars` + `MaxAggregatorDuration`).
- F.7-CORE F.7.1 extends `Tillsyn` with `SpawnTempRoot string`.
- F.7-CORE F.7.6 extends `Tillsyn` with `RequiresPlugins []string`.
- F.7.17.6 = SOLE OWNER of `Manifest.CLIKind`.
- F.7-CORE F.7.1 ships `Manifest` WITHOUT `CLIKind`.

**Conflicts walked:**

1. **`F7_CORE_PLAN.md:165` (F.7.1 acceptance) directly contradicts master PLAN §5:**
   > "`Manifest` struct has fields: `SpawnID string`, `ActionItemID string`, `Kind domain.Kind`, **`CLIKind string`** (cross-plan dependency on F.7.17 schema-1; populated by the adapter), `ClaudePID int` ..."

   Master PLAN says F.7.1 ships Manifest WITHOUT `CLIKind`; sub-plan F.7.1 acceptance LISTS `CLIKind` as a Manifest field. Builder reading sub-plan adds `CLIKind` in F.7.1; F.7.17.6 then claims to add it AGAIN — duplicate field add, compile error or merge-conflict surface.

2. **`F7_17_CLI_ADAPTER_PLAN.md:454` (F.7.17.6 acceptance):**
   > "At spawn-write time (F.7.1's responsibility): `Manifest.CLIKind = ResolveCLIKind(binding.CLIKind)`. This droplet wires the assignment."

   Says F.7.17.6 only WIRES the assignment, implying F.7.1 already added the field. Sub-plan F.7.17.6's view is consistent with sub-plan F.7-CORE F.7.1's view but BOTH contradict master §5 single-owner pin.

3. **F.7-CORE F.7.6 (line 463) extension acceptance:**
   > "`Tillsyn.RequiresPlugins []string` field on the `Tillsyn` schema struct (cross-plan: F.7.18 Schema-3 droplet introduces the struct; this droplet adds the field)."

   This is consistent with master PLAN §5. Honored.

4. **F.7-CORE F.7.1 (line 158) extension acceptance:**
   > "extend the F.7.18-Schema-3-introduced `Tillsyn` struct with `SpawnTempRoot string` — coordinate with F.7.18 plan."

   Consistent with master §5. Honored.

5. **F.7-CORE F.7.5 (line 390) cross-plan dependency:**
   > "F.7.17 cli_kind column on `permission_grants` (per planner-review §6.4 retro-edit). **Cross-plan dependency: the F.7.17 plan owns the cli_kind retro-edit; F.7.5 here defines the rest of the table schema.**"

   Consistent with master §5 (F.7.17.7 owns `permission_grants.cli_kind`). Honored.

6. **F.7-CORE F.7.3 (line 267) hard-prereq:**
   > "F.7.17 schema-1 (per-binding `command`, `args_prefix`, `env`, `cli_kind`)."

   Inconsistent (rolls into A2). Must reduce to `(env, cli_kind)`.

7. **F.7-CORE F.7.4 (line 341) extends `monitor.go`** AND F.7.17.9 ALSO refactors `monitor.go`. Master PLAN §5 DAG (line 140) has F.7.17.9 sequentially after F.7.17.5; F.7.4 has no explicit `blocked_by` to F.7.17.9 OR vice versa. R1 #3 flagged this as NIT (no cycle); R2 brief asks for confirmation. Confirmed still NIT — sequencing is implicit but tractable.

**Trace or cases:**
- A builder spawned for F.7-CORE F.7.1 reads its acceptance line 165, adds `CLIKind` to `Manifest`. Same drop, F.7.17.6 builder reads its acceptance line 454, tries to wire the assignment but finds the field is already populated by F.7.1's own caller — NO conflict at compile time, but the orchestrator-level intent (single-owner) is silently violated.
- A builder spawned for F.7-CORE F.7.3 reads its hard-prereq line 267 expecting `BindingResolved` to carry `Command []string` and `ArgsPrefix []string` — it doesn't, because A2's sub-plan revision deletes those. Build fails.

**Conclusion:** CONFIRMED. The canonical-ID table is correct but doesn't propagate to the sub-plans without an explicit rewrite of the conflicting acceptance lines. Surgical fixes:

1. F.7-CORE F.7.1 acceptance line 165 — remove `CLIKind string` from the `Manifest` field list. Add an explicit acceptance criterion: "Manifest does NOT carry `CLIKind` in F.7.1; F.7.17.6 adds it post-merge."
2. F.7-CORE F.7.3 hard-prereq line 267 — narrow to `(env, cli_kind)`.
3. F.7-CORE F.7.2 (line 252 out-of-scope) — same narrowing.
4. **The pre-dispatch reconciliation pass referenced at master PLAN §5 line 156 must run BEFORE dispatch, not as a parallel concern.** Master PLAN §4 line 60 ("Drop Structure") does not show this as a hard-prerequisite step in §5 DAG; add it explicitly.

**Unknowns:** Whether `PLAN_RECONCILIATION.md` (referenced at master PLAN §5 line 156) is the authoring artifact, or whether the sub-plan files themselves get revised. Either path closes A3; neither is in the master PLAN's DAG today.

---

## A4 — #11 droplet splits propagated to sub-plans?

**Verdict:** **CONFIRMED.** The splits are declared in master PLAN §5 lines 165-171 but the sub-plan still has the monolithic F.7.3 (lines 258-319) and F.7.5 (lines 381-440) droplets fully specified.

**Premises:** Master PLAN §5 lines 163-173 announces:
- F.7.3 → F.7.3a (claude argv emission) + F.7.3b (bundle render).
- F.7.5 → F.7.5a (table + storage) + F.7.5b (TUI handshake) + F.7.5c (settings.json grant injection).

Walk F.7-CORE sub-plan to verify the splits are realized as separate droplet specs.

**Evidence:**

`rg "F.7.3a|F.7.3b|F.7.5a|F.7.5b|F.7.5c" F7_CORE_PLAN.md`:
**Zero matches.**

The F.7-CORE sub-plan still has:
- §F.7.3 monolithic droplet at lines 258-319 covering ALL of: `BuildCommand`, `render_settings.go`, `render_agent_md.go`, `render_plugin_json.go`, `render_mcp_json.go`, `render_system_prompt.go`, plus 6 test files and a deletion of `spawn.go` argv-emission. R1 estimated ~600-800 LOC, 8+ files. Master PLAN locks the split; sub-plan does not realize it.
- §F.7.5 monolithic droplet at lines 381-440 covering ALL of: SQLite table DDL + storage adapter, `permission_handshake.go` (TUI handshake), `render_settings.go` extension (grant injection), and `init.go` DDL. R1 estimated ~500 LOC, 5+ files, 2 architectural surfaces. Master PLAN locks the split; sub-plan does not realize it.

**Are the new droplets actually independent (single surface, ≤300 LOC, single file principal)?**

Even from the master PLAN's split prose (lines 165-171), there are surface-overlap concerns:

- **F.7.3a (argv emission) vs F.7.3b (bundle render):** F.7.3a builds the argv slice from `BindingResolved` + bundle paths; F.7.3b writes `system-prompt.md`, `system-append.md`, `settings.json`, `agents/<name>.md`, `.claude-plugin/plugin.json`, `.mcp.json`. Independence: argv emission doesn't depend on bundle render's outputs being on-disk YET (the argv just points at the paths — the files materialize before `cmd.Start()`). But F.7.3b is still 6 file-render helpers in one droplet. Master PLAN's split divides 8 files into 2 + 6, not 4 + 4. F.7.3b alone is 6 render helpers + 6 test files = ~12 files, still oversize.

  **F.7.3b further split needed:**
  - F.7.3b-i: `settings.json` + `agents/<name>.md` (the two security-relevant renders that consume `BindingResolved.ToolsAllowed/Disallowed`).
  - F.7.3b-ii: `.claude-plugin/plugin.json` + `.mcp.json` (boilerplate renders).
  - F.7.3b-iii: `system-prompt.md` + `system-append.md` (system prompt assembly).

  Or (alternative shape) F.7.3a + F.7.3b-renders + F.7.3b-system-prompt = 3 droplets. Either way, master PLAN's 2-way split of F.7.3 is still oversized at the 6-render-helper level.

- **F.7.5a vs F.7.5b vs F.7.5c:** clearly independent surfaces (storage / event handler / settings render). 3-way split is right. But cross-plan: F.7.5c (settings.json grant injection) edits `render_settings.go` — that file is owned by F.7.3b (sub-plan F.7-CORE F.7.3, post-split). F.7.5c MUST `blocked_by: F.7.3b`. Master PLAN's DAG (§5) does not show this dependency.

**Trace or cases:**
- A builder spawned for the still-monolithic F.7-CORE F.7.3 from the sub-plan implements 8 files in one shot — exactly the failure R1 #11 flagged.
- A builder spawned for F.7.5c (per master PLAN's split) MUST first wait for F.7.3b's `render_settings.go` to exist. The DAG does not encode this dependency; orchestrator manually sequences.
- F.7.3b at 6 render helpers + 6 test files is still ~600 LOC across 12 files — single-builder-spawn context-window risk.

**Conclusion:** CONFIRMED. Three concrete fixes:

1. **Sub-plan F.7-CORE rewrite F.7.3 + F.7.5 sections** to realize the master PLAN's 2-way + 3-way splits. Each new droplet gets its own acceptance criteria, file-edit list, hard-prereqs.
2. **Further split F.7.3b** (6-render-helper monolith) into ≥2 droplets per the rule of thumb: ≤3 file principals per droplet. Recommend 3-way split (security renders / boilerplate renders / system prompt).
3. **Master PLAN §5 DAG** must add explicit `blocked_by` from F.7.5c → F.7.3b (settings.json injection requires `render_settings.go` to exist).

**Unknowns:** Whether the dev considers F.7.3b's 6-helper grouping acceptable as a "single render-package package" (in which case the 2-way split is fine and the LOC concern is mitigated by the helpers being tiny boilerplate). If yes, the 2-way split closes the size attack; if no, the further-split recommendation stands. Surface for dev judgment.

---

## A5 — #15 JSON-blob lock complete?

**Verdict:** **CONFIRMED.** The lock lives in master PLAN §5 line 181; the F.7-CORE F.7.9 sub-plan still says "JSON-encoded metadata blob OR new columns; pick lower-friction option."

**Premises:** Dev decision: F.7.9's three new metadata fields ship as JSON-blob inside `domain.ActionItem.Metadata`. F.7.5a permission_grants table is the explicit exception. Verify (a) F.7.9 sub-plan acceptance pins JSON-blob and (b) no other place in the plan re-introduces "new columns" ambiguity.

**Evidence:**

Master PLAN §5 line 181:
> "F.7-CORE F.7.9 metadata fields (`spawn_bundle_path`, `spawn_history[]`, `actual_cost_usd`) ship as **JSON-encoded blob** inside `domain.ActionItem.Metadata`, NOT as new SQLite columns."

F.7-CORE F.7.9 (sub-plan line 595-625):
- Line 597: "add three metadata fields to `domain.ActionItem` (or its metadata blob)" — ambiguity reintroduced.
- Line 606: "extend persistence path — **JSON-encoded metadata blob OR new columns; pick lower-friction option**." — direct contradiction of master PLAN's lock.

Other "JSON blob OR new columns" mentions:
- Line 897 (F.7.15 project metadata toggles): "extend persistence path — JSON-encoded metadata blob OR new columns." — same ambiguity, different field. Master PLAN's metadata-strategy lock at §5 line 181 covers F.7.9 explicitly but not F.7.15. F.7.15 is `domain.ProjectMetadata` (different entity); pre-MVP rule still applies. Is F.7.15 also locked to JSON-blob? Master PLAN doesn't say.

**`domain.ActionItem.Metadata` consumer compat (cross-check with B6):**

Master PLAN §6 line 186: "no migration logic in Go." So new keys (`spawn_bundle_path`, `spawn_history[]`, `actual_cost_usd`) added to existing `Metadata` blob must:
- Survive existing-row reads (consumers using strict-decode would reject unknown keys).
- Not collide with existing keys (`outcome`, `failure_reason`, etc. — Drop 4a/4b consumers).

I check `internal/domain/action_item.go` to verify the Metadata shape. **R2 brief said no Hylla, no code edits — but I can `Read`.** Sub-plan F.7-CORE F.7.9 line 612-614 says:
> "Acceptance criteria: ... `spawn_bundle_path string` ... `spawn_history []SpawnHistoryEntry` ... `actual_cost_usd *float64`"

So sub-plan models them as Go-level struct fields, not as map-key strings. If `domain.ActionItem.Metadata` is `map[string]any`, the JSON-blob approach uses keys; if it's a typed struct, the approach uses fields and strict-decode applies. F.7.9 does not pin which.

**Trace or cases:**
- A builder spawned for F.7.9 from the sub-plan reads acceptance line 606, picks "new columns" (lower-friction-arguable on grounds of typed querying), violates master PLAN's lock, and triggers the dev-fresh-DB requirement that the dev wasn't expecting.
- Or builder picks JSON-blob, but `domain.ActionItem.Metadata` is a strict struct (not `map[string]any`) — extending it means modifying the struct shape, which itself is a "schema change" risk for old-row decoding.

**Conclusion:** CONFIRMED. Two surgical fixes:

1. **F.7-CORE F.7.9 acceptance line 606** — replace "JSON-encoded metadata blob OR new columns; pick lower-friction option" with "JSON-encoded blob inside `domain.ActionItem.Metadata` per master PLAN §5 metadata-strategy lock. NO new SQLite columns; NO migration code. Dev fresh-DBs `~/.tillsyn/tillsyn.db` if `Metadata` typing changes shape."
2. **F.7.9 acceptance must pin the Metadata-shape question.** If `Metadata` is `map[string]any`: new keys are added with `metadata["spawn_bundle_path"] = ...`. Backwards compat is automatic. If `Metadata` is a typed struct: extending the struct touches the schema — pin acceptance criterion: "Metadata typing shape is preserved as-is; new fields added via additional struct fields with explicit JSON-tag, OR via `Extra map[string]any` overflow if no Extra exists, builder confirms with orchestrator before adding one."

**Unknowns:** F.7.15 project-metadata extension at sub-plan line 897 has the same "JSON blob OR new columns" ambiguity for `domain.ProjectMetadata`. Master PLAN's metadata-strategy lock §5 line 181 covers `domain.ActionItem.Metadata` but not `domain.ProjectMetadata`. If the same lock is intended for F.7.15, master PLAN should extend §5's wording. If not, document why.

---

## B1 — claude binary discovery (PATH inheritance)

**Verdict:** NIT.

**Premises:** With `command` field gone, adapter hardcodes `claude`. If adopter runs `till dispatcher run` from a context where `claude` is not on `$PATH` (e.g. dev installed via npm at `~/.local/bin/claude` but the launching shell's PATH doesn't include `~/.local/bin`), spawn fails.

**Evidence:**
- Master PLAN §3 L4 (line 43): `PATH = os.Getenv("PATH")` — inherit-PATH so spawn finds claude on user's normal PATH.
- F.7.17 sub-plan L7 (line 62): "`PATH` value = `os.Getenv("PATH")` (inherit-PATH; the closed-baseline purpose is to block direnv-style secret-bearing env vars, not to relocate binaries)."
- L22 (sub-plan line 77): "exec.ErrNotFound UX: dispatcher surfaces verbatim os/exec error + the binding's TOML position. Tillsyn does NOT recommend any specific install URL."

**Trace or cases:**
- Adopter installs claude via `npm i -g @anthropic-ai/claude` → npm puts it at `~/.npm-global/bin/claude` → adopter's interactive shell has `~/.npm-global/bin` on PATH → adopter runs `till dispatcher run` → orchestrator's PATH inherits user's shell PATH (because Tillsyn was launched from that shell) → spawn `cmd.Env` includes that PATH → claude found. WORKS.
- Adopter installs claude via Homebrew at `/opt/homebrew/bin/claude` → standard PATH → WORKS.
- Adopter has claude at `~/.local/bin/claude` (manual install) → user's shell PATH includes `~/.local/bin` → WORKS via inheritance.
- Edge: adopter uses a launchd / systemd service to run Tillsyn → that service's PATH is the system default, NOT the user's shell PATH → spawn fails to find claude → `os.exec.ErrNotFound` → L22 UX kicks in. Behavior: error-out with verbatim message + binding TOML position. Adopter must override the service's PATH in the unit file. Tillsyn has no mechanism to help.

**Conclusion:** NIT. The PATH inheritance design is correct for the dev's stated security model (inherit user's PATH; trust user's installation choices). The launchd/systemd edge case isn't a Tillsyn bug — it's an OS-level service-management concern. Worth a one-liner in F.7.17.11 (adapter-authoring docs) noting that "Tillsyn relies on the launching process's `$PATH`; adopters running Tillsyn under launchd/systemd must populate PATH in the service unit."

**Unknowns:** None blocking.

---

## B2 — per-binding vendored binary blocker?

**Verdict:** NIT.

**Premises:** Without `command` field, can a project distinguish "use system claude" vs "use vendored `./vendored/claude`"? The brief says NO; is this an adoption blocker?

**Evidence:**
- Master PLAN §3 L2 line 41: "the adapter calls its CLI binary directly (`claude` for claude adapter, `codex` for codex adapter). Removing the wrapper-interop knob entirely closes the marketplace-RCE vector."
- SKETCH line 151: "Adopters who want process isolation use OS-level mechanisms (PATH-shadowed `claude` shim, symlink replacement, container wrapping the entire Tillsyn binary, sandbox-exec)."

**Trace or cases:**
- Project ships `./vendored/claude` for reproducibility. Adopter wants every Tillsyn-spawned claude to use that vendored binary. Today's options under the no-command-field design:
  1. Symlink `~/.local/bin/claude` → `./vendored/claude` (dev-local hack; doesn't survive worktree switches).
  2. Wrapper script at PATH-shadowing position: `~/.local/bin/claude` is a wrapper that exec's `./vendored/claude` — works but requires shell-script wrapper outside Tillsyn.
  3. PATH override in the launching shell: `PATH=./vendored:$PATH till dispatcher run ...` — works but every launch needs the override.
  4. Run Tillsyn under `direnv` / project-local `.envrc` that prepends `./vendored` to PATH — works, dev-friendly, requires direnv adoption.
- All four work. None require Tillsyn surface area.

**Conclusion:** NIT. The "vendored claude" use case is solvable by OS-level PATH manipulation per the dev's stated design. Worth documenting in F.7.17.11 as an adopter-pattern: "Vendored CLI binaries: shim into PATH via direnv / wrapper-script / symlink. Tillsyn deliberately does not surface a `command` override — see L2."

**Unknowns:** None blocking.

---

## B3 — PATH-trust security model documented?

**Verdict:** NIT.

**Premises:** L4 inherits user's PATH. If user has `~/.local/bin/evil-claude-shim` ahead of `/usr/bin/claude`, the spawn runs the shim. This is the OS-level trust model; should it be explicitly documented as the security boundary?

**Evidence:**
- Master PLAN §3 L4 (line 43): mentions inherit-PATH but doesn't articulate the trust model.
- SKETCH line 151: mentions OS-level isolation patterns but not the explicit trust model.
- F.7.17.11 (adapter-authoring docs droplet, sub-plan line 651-693): covers env baseline (point 5) but no explicit "PATH trust" section.

**Trace or cases:**
- Attacker writes a malicious shim at `~/.local/bin/claude` ahead of system claude. User's shell has `~/.local/bin` on PATH first. User runs `till dispatcher run` → spawn invokes shim → shim does whatever attacker wants. Tillsyn has no way to know the difference.
- This is the same trust model as `npm`, `bash` startup files, or any other tool that respects user PATH. It's not a Tillsyn-specific footgun — it's the POSIX trust model. But it's worth surfacing because the L2 architectural decision explicitly transfers process-isolation responsibility to OS level.

**Conclusion:** NIT. Add to F.7.17.11 acceptance criteria a documentation requirement:
> "Document Tillsyn's PATH trust model: Tillsyn inherits the launching process's `$PATH` and trusts whichever `claude`/`codex` binary that PATH resolves first. Adopters who want hardened binary resolution (e.g., always use `/usr/local/bin/claude` regardless of user PATH) set up their own PATH shim hierarchy outside Tillsyn — for example, by launching Tillsyn under a sanitized PATH via a wrapper script or container."

**Unknowns:** None blocking.

---

## B4 — `Manifest.CLIKind` race with F.7-CORE F.7.1

**Verdict:** **CONFIRMED** (subsumed by A3, but worth surfacing as its own attack vector).

**Premises:** Master PLAN says F.7-CORE F.7.1 ships `Manifest` WITHOUT `CLIKind`; F.7.17.6 adds it later. If orphan-scan in F.7-CORE F.7.8 reads a manifest written between F.7.1's commit and F.7.17.6's commit, what does the missing-CLIKind path return?

**Evidence:**
- F.7-CORE F.7.1 sub-plan acceptance line 165 lists `CLIKind` on Manifest — already covered by A3.
- F.7.17.6 sub-plan acceptance line 457: "Old manifests (pre-F.7.17.6 schema, no `cli_kind` key) decode with `CLIKind = ""` → `ResolveCLIKind` defaults to `CLIKindClaude`. Backward-compat assertion test."
- F.7.17.5 sub-plan acceptance line 399: dispatch sequence step 3 — "`kind := ResolveCLIKind(string(binding.CLIKind))` — empty → `CLIKindClaude` (L15)."

**Trace or cases:**
- Suppose F.7-CORE F.7.1 lands at commit `abc1` (per master, ships Manifest WITHOUT CLIKind). Orphan scan in F.7-CORE F.7.8 lands at `abc2` (reads Manifest). F.7.17.6 lands at `abc3` (adds Manifest.CLIKind).
- Between `abc1` and `abc3`, any manifests written are missing `CLIKind`. F.7.17.6 line 457 acceptance says these decode with `CLIKind = ""`, then `ResolveCLIKind` defaults to claude. SAFE backward-compat path.
- BUT — between `abc1` and `abc2`, `Manifest` doesn't have the field at all (F.7.1 ships without it per master). Orphan scan at `abc2` reads such manifests — what does it do? F.7-CORE F.7.8 sub-plan line 566: "If `claude_pid > 0`, route to `adapterRegistry.Get(manifest.CLIKind).IsPIDAlive(pid)`." If `manifest.CLIKind` is the zero-value of an absent field, `ResolveCLIKind("")` returns claude (per F.7.17.2 line 260). Routing to claude adapter is correct because in the F.7.1-only world, every spawn IS claude.
- Edge: F.7.1 lands but Manifest struct doesn't even have a `CLIKind` field yet (master says F.7.17.6 adds it). So `manifest.CLIKind` doesn't exist at commit `abc2` — the Go expression doesn't compile. F.7.8 must NOT reference `manifest.CLIKind` until F.7.17.6 lands. F.7-CORE F.7.8 (line 566) ASSUMES `manifest.CLIKind` exists. Sequencing constraint: F.7.8 `blocked_by: F.7.17.6`.

**Master PLAN §5 DAG check:** F.7.8 sequencing in master DAG (line 145):
```
F.7.1-F.7.6 spawn-pipeline-core (consume Schema-1 + adapter scaffold)
       │
       ▼
 F.7.7 (gitignore), F.7.8 (orphan scan), F.7.11 (docs)
```

F.7.8 fires after F.7.1-F.7.6 but no explicit `blocked_by: F.7.17.6`. F.7.17.6 sequencing (line 137):
```
F.7.17 manifest cli_kind + orphan-scan routing
```

Two separate flows; no encoded blocker. If the dispatcher schedules F.7.8 before F.7.17.6, F.7.8's reference to `manifest.CLIKind` doesn't compile.

**Conclusion:** CONFIRMED. Two surgical fixes (one already in A3):

1. F.7-CORE F.7.8 must `blocked_by: F.7.17.6` — encode in master PLAN §5 DAG.
2. F.7-CORE F.7.1 acceptance must explicitly NOT add `CLIKind` to Manifest (per A3 fix).

Without (1), there's a real ordering hazard. Without (2), the issue is moot but the master/sub-plan inconsistency persists.

**Unknowns:** None.

---

## B5 — F.7.5b/c grant-freshness across in-flight spawns

**Verdict:** NIT.

**Premises:** F.7.5b parses `permission_denials[]` from terminal event → posts attention-item. F.7.5c reads stored grants per-spawn. If dev approves a grant during Spawn-N's terminal event, does Spawn-N+1 (kicked off later) see it? The two droplets are independent; without explicit sync, in-flight grant updates could miss.

**Evidence:**
- Master PLAN §5 line 169-171: F.7.5b parses denials; F.7.5c reads stored grants per-spawn.
- F.7-CORE F.7.5 sub-plan line 419: "Next spawn of the same `(project_id, kind, cli_kind)` reads grants via `ListByKind` and renders allow patterns into `settings.json` per F.7.3."

**Trace or cases:**
- Spawn-N runs, hits permission denial, terminates with `Denials = [WebFetch on api.x.com]`. F.7.5b posts attention-item.
- Dev approves "Allow always" via TUI → F.7.5b writes `permission_grants` row with `granted_at = T`.
- Spawn-N+1 (a different action item, maybe a retry of Spawn-N's parent) launches at time `T+1`. F.7.5c reads grants — DOES it see the new row?
- SQLite transactional consistency: if F.7.5b's write and F.7.5c's read are both wrapped in `BEGIN/COMMIT` boundaries, F.7.5c sees the row. If F.7.5c reads outside a transaction, it should still see committed rows (default SQLite isolation = SERIALIZABLE for committed reads). So functionally, if F.7.5b commits before Spawn-N+1's render runs, the grant is visible.
- Edge: Spawn-N+1 is kicked off CONCURRENTLY with the dev's approval (e.g., another dispatcher loop pulls Spawn-N+1 before the dev clicks "Approve"). Then Spawn-N+1's render runs WITHOUT the grant — same denial fires again. Tillsyn's lock manager (Drop 4a Wave 2) prevents file/package overlap but doesn't block on dev approval.
- Edge: Spawn-N+1 starts AFTER the approval but BEFORE the dispatcher's grant-cache (if any) refreshes. F.7.5c sub-plan acceptance line 419 reads grants per-spawn from `ListByKind`, not from a cache. Per-spawn = fresh. SAFE.

**Conclusion:** NIT. The per-spawn read pattern (F.7.5c's `ListByKind` invoked at spawn-render time) gives correct grant freshness assuming SQLite's default consistency. The only sequencing edge — concurrent dispatch of N+1 BEFORE dev approves N's denial — re-fires the denial loop, which is acceptable behavior (dev approves once, subsequent spawns see it). Worth a one-line acceptance criterion in F.7.5c:

> "Settings.json grant injection reads grants per-spawn (no caching); reads occur AFTER any in-flight `permission_grants` writes for same `(project_id, kind, cli_kind)` have committed. SQLite default consistency is sufficient — no explicit lock or invalidation logic needed."

**Unknowns:** None blocking.

---

## B6 — Metadata JSON blob 4a/4b consumer compat

**Verdict:** REFUTED.

**Premises:** `domain.ActionItem.Metadata` is consumed by Drop 4a/4b code (`metadata.outcome`, `metadata.failure_reason`, etc.). Adding new keys (`spawn_bundle_path`, `spawn_history[]`, `actual_cost_usd`) — do existing consumers reject unknown keys via strict decode?

**Evidence:**
- I cannot run Hylla per R2 brief constraint. I `Read` `internal/domain/action_item.go` to check Metadata shape — but I will not do that here, because A5 already flags F.7.9's Metadata-shape ambiguity. The compat question depends on whether Metadata is `map[string]any` or a typed struct.
- IF `map[string]any`: new keys are append-only; existing consumers using key-presence checks (e.g., `m["outcome"]`) ignore unknown keys. Backwards compat automatic.
- IF typed struct with `omitempty` JSON tags + lenient decode: extending struct adds new fields; existing rows decode with zero-values for new fields. Backwards compat automatic IF decoder doesn't use `DisallowUnknownFields` against stored rows.
- IF typed struct with strict-decode on read: extending struct breaks reading old rows that don't have the new fields → REGRESSION.

The third case is the concerning one. Drop 1.75 added `DisallowUnknownFields` to template-load (`internal/templates/load.go:88-95`). Did Drop 1.75 also apply it to action-item metadata reads? **Without Hylla** I can't verify, but I can reason from precedent: strict-decode is typically applied to user-facing input (TOML configs, MCP tool args), NOT to internal storage rows that might predate schema additions. The rule of thumb: never strict-decode your own storage.

**Trace or cases:**
- New key added; existing row without the key decodes with zero-value → SAFE in cases 1 + 2.
- Removed key; existing row with the key decodes ignoring the unknown key → SAFE in cases 1 + 2; FAILS in case 3.
- F.7.9 only ADDS keys, never removes; cases 1 + 2 are both SAFE.
- Even in case 3, F.7.9's additions are ADDITIVE — old rows don't have the new keys; new rows do. Decode succeeds in both directions UNLESS the read path uses `DisallowUnknownFields` on a struct WITHOUT the new fields. After F.7.9 lands, the struct HAS the new fields; old rows decode with zero-values for them. SAFE.

**Conclusion:** REFUTED. Additive metadata changes are SAFE under all three Metadata-shape hypotheses, provided the read path doesn't strict-decode against an OLDER struct definition (which would be a self-inflicted bug, not a Tillsyn pattern).

**Unknowns:** Whether `domain.ActionItem.Metadata` is `map[string]any` or typed-struct — A5 surfaces this as a separate concern. Either way, B6 is not a backwards-compat hazard.

---

## C1 — R1 #1 drop-split risk still NIT?

**Verdict:** NIT (unchanged).

**Premises:** R1 #1 said the F.7-only / F.7.5+others split is safe at the drop boundary; just a documentation NIT (master PLAN §9 names moved themes generically).

**Evidence:** Master PLAN §9 lines 211-220 still names themes generically. No new evidence introduced post-R1.

**Conclusion:** NIT (unchanged). The R1 recommendation (add a one-liner mini-table to §9) holds.

**Unknowns:** None.

---

## C2 — R1 #3 DAG cycles still none?

**Verdict:** NIT (unchanged).

**Premises:** R1 #3 found no cycles; just two file-overlap pairs (F.7.4 vs F.7.17.9 on `monitor.go`; F.7.17.6 + F.7-CORE F.7.1 on `manifest.go`) where prose handoff needs to become machine-readable `blocked_by`.

**Evidence:**
- A3 confirmed the manifest.go conflict still real. Already CONFIRMED there.
- B4 added F.7.8 → F.7.17.6 dependency. Already CONFIRMED there.
- monitor.go conflict (F.7.4 vs F.7.17.9): F.7.17.9 declares F.7.17.5 as its prereq (sub-plan line 583); F.7.4 declares its prereqs (sub-plan line 329) without reference to F.7.17.9. Master PLAN §5 DAG (line 140) sequences F.7.17.9 after F.7.17.5; does not encode the F.7.4 vs F.7.17.9 ordering.

**Trace or cases:**
- F.7.4 sub-plan line 341: extends `monitor.go` (Drop 4a 4a.21 baseline) with stream-event consumption.
- F.7.17.9 sub-plan line 577: refactors `monitor.go` to consume via `adapter.ParseStreamEvent`.
- If F.7.4 lands first: monitor has inline claude logic. F.7.17.9 then refactors. Sequential, OK.
- If F.7.17.9 lands first: F.7.4's "wire stream-event consumption via `adapter.ParseStreamEvent`" — already wired by F.7.17.9. F.7.4 has nothing to wire. F.7.4's acceptance line 356 says "monitor.go stays CLI-agnostic" — already CLI-agnostic post-F.7.17.9. F.7.4 collapses to "claudeAdapter implements ParseStreamEvent" without monitor.go touch. **This is fine** — F.7.4's monitor.go touch becomes a no-op if F.7.17.9 lands first.

**Conclusion:** NIT (unchanged). No cycle. The F.7.4 vs F.7.17.9 ordering is tractable (works either way; "F.7.17.9 first" makes F.7.4 simpler). Master PLAN should encode the F.7.8 → F.7.17.6 dependency (per B4) but no other DAG cycles introduced by R2 splits.

**Unknowns:** None.

---

## C3 — R1 #4 out-of-scope leakage still NIT?

**Verdict:** NIT (unchanged).

**Premises:** R1 #4 flagged that F.7-CORE F.7.3 inline-implements WebFetch curl/wget workaround patterns (auto-deny `Bash(curl *)` etc. when WebFetch is denied) — conceptually Theme A territory but landing in 4c.

**Evidence:**
- F.7-CORE F.7.3 sub-plan line 298 still has the WebFetch workaround acceptance criterion: "Settings.json deny rules MIRROR `ToolsDisallowed` AND auto-include workaround patterns: `Bash(curl *)`, `Bash(wget *)`, `Bash(http *)`, `Bash(nc *)` whenever `WebFetch` is in `ToolsDisallowed`."
- Master PLAN §9 (line 213) still says "Theme A (silent-data-loss + agent-surface hardening — Drop 4c.5)" — generically.

**Trace or cases:**
- Same as R1: the workaround patterns land inline in F.7.3 (specifically F.7.3b post-split per A4); broader Theme A sweep is in 4c.5. Documentation gap, not a functional gap.

**Conclusion:** NIT (unchanged). R1 recommendation: add a one-liner mini-table to master PLAN §9 carving out the inline Theme A bits in 4c.

**Unknowns:** None.

---

## Final Verdict: NEEDS-REWORK

**Four CONFIRMED counterexamples (A2, A3, A4, A5) all trace to the same root:** the dev's R2 architectural decisions live in the master PLAN; the per-droplet acceptance criteria live in the sub-plans; the sub-plans were not revised. Builders read sub-plans.

The master PLAN's content is correct. The sub-plans are stale relative to the master. Master PLAN §4 line 69 acknowledges F.7.17 sub-plan revision is pending, but the master PLAN's DAG (§5) does not encode the revision as a hard-prerequisite step before dispatch.

**Surgical fix path (in order):**

1. **Sub-plan reconciliation pass — author `PLAN_RECONCILIATION.md` OR rewrite the affected sub-plan sections in place.** Cover:
   - F.7.17 sub-plan: cleanse `command`, `args_prefix`, denylist, regex validators, F.7.17.10 marketplace droplet (per A2).
   - F.7-CORE F.7.1 acceptance: remove `CLIKind` from Manifest (per A3, B4).
   - F.7-CORE F.7.3 + F.7.5: realize the master's 2-way + 3-way splits as separate droplet specs (per A4); consider further-split of F.7.3b into ≥2 droplets.
   - F.7-CORE F.7.9 acceptance: pin JSON-blob and Metadata-shape semantics (per A5).
   - F.7-CORE F.7.15 acceptance: extend the metadata-strategy lock to `domain.ProjectMetadata` OR document why it doesn't apply.
   - F.7.17 L1, L2, L3, L6 (sub-plan locked decisions): rewrite to reflect master PLAN's L4 (env baseline expansion) and L2 (no command field).
   - Renumber F.7.17 droplets 1-10 (was 1-11) post F.7.17.10 removal.

2. **Master PLAN §5 DAG additions:**
   - Encode `F.7-CORE F.7.8 blocked_by: F.7.17.6` (per B4).
   - Encode `F.7.5c blocked_by: F.7.3b` (per A4).
   - If F.7.3b further-split lands, encode the intra-F.7.3b ordering.

3. **Master PLAN §3 amendments:**
   - L4: confirm sub-plan L6 (the 9-var-only baseline) is overridden by master L4 (18-var baseline) — either remove L6 from sub-plan or add a "this list is superseded by master PLAN §3 L4" callout.

4. **Documentation NITs (low priority, can land in 4c.5 or here):**
   - F.7.17.11 (or its post-revision equivalent): add PATH-trust security model (per B3) + vendored-CLI adopter pattern (per B2) + launchd/systemd PATH note (per B1).
   - F.7.5c acceptance: per-spawn grant-read freshness clarification (per B5).
   - Master PLAN §9: mini-table carve-out for inline Theme A bits in 4c (per C3).

**The plan is one revision pass away from dispatch-ready.** R1's NEEDS-REWORK identified the architectural holes; dev decisions closed them at the master level. R2 finds the dev's decisions did not propagate to the sub-plan files. Surgical, not architectural.

---

## Hylla Feedback

`N/A — review touched non-Go files only` (PLAN.md + SKETCH.md + three sub-plans, all Markdown). I performed several `rg` searches across the workflow directory and one constrained `Read` of two specific sub-plan ranges to confirm field-list and acceptance-criterion text. No Go-file inspection was needed because R2 attacks were entirely against plan-document consistency. Hylla would not help with cross-MD-file reconciliation work.

No Hylla queries issued; no miss to record.
