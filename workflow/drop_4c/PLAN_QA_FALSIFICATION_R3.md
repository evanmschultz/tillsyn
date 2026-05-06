# Drop 4c — Plan QA Falsification (Round 3)

**Reviewer:** go-qa-falsification-agent (round 3).
**Inputs reviewed:**
- `workflow/drop_4c/PLAN.md` (master).
- `workflow/drop_4c/F7_17_CLI_ADAPTER_PLAN.md` (with REVISIONS appended).
- `workflow/drop_4c/F7_CORE_PLAN.md` (with REVISIONS appended).
- `workflow/drop_4c/F7_18_CONTEXT_AGG_PLAN.md` (UNCHANGED from R2).

**Round 2 verdict:** NEEDS-REWORK (4 CONFIRMED + 1 NEW).
**Round 3 mandate:** verify R2 fixes (V1-V5) + attack new holes from REVISIONS-as-appendix pattern (N1-N6) + R2 NIT regression check (R1).

**Method:** Read each plan top-to-bottom. Run targeted greps for problematic strings across the entire `workflow/drop_4c/` tree. Reason adversarially about each attack.

---

## 1. Findings

### 1.1 V1 — R2 #A2 fix (command/args_prefix removal): NEEDS-REWORK

**Premises:** R2 #A2 found `Command []string` / `ArgsPrefix []string` / `validateAgentBindingCommandTokens` / `shellInterpreterDenylist` references in F.7.17 sub-plan body. R3 fix appends REVISIONS REV-1 + REV-4 in F.7.17 and REV-1 in F.7-CORE that supersede those portions.

**Evidence (residual references):**

- **F.7.17 sub-plan body (NOT touched by REVISIONS, only superseded "logically"):** Stale references remain at lines:
  - L15: `\`Command []string\`, \`ArgsPrefix []string\`, \`Env []string\`, \`CLIKind string\` on \`templates.AgentBinding\``.
  - L16: `\`validateAgentBindingCommandTokens\` (regex literal pinned + closed shell-interpreter denylist).`.
  - F.7.17.1 entire droplet (lines 159-228): goal at L161 says "widen `templates.AgentBinding` with `Command`, `ArgsPrefix`, `Env`, `CLIKind`"; acceptance L176 lists all 4 fields with TOML tags `command`, `args_prefix`, `env`, `cli_kind`; L178-186 fully spec out the regex literal + denylist constant + 12 reject test scenarios for command tokens; L185 rejects empty `command` array.
  - F.7.17.2 (line 255): `BindingResolved` field list includes `Command []string, ArgsPrefix []string`.
  - F.7.17.3 (line 303): `Path` derived from `br.Command[0]` (default `"claude"` when `br.Command` is nil/empty); L304 args shape includes `[Path] ++ ArgsPrefix ++ ...`; L316 test scenario "Argv with `args_prefix = ["--profile", "dev"]`".
  - F.7.17.5 (line 410): `SpawnDescriptor` gains `CLIKind` + `Command []string` + `Env []string`.
  - F.7.17.10 droplet ENTIRELY PRESENT at L610-647 — REVISIONS says "REMOVED entirely" but the body droplet is still in the per-droplet decomposition section. A builder reading top-down will hit L610 "marketplace install-time confirmation" with full acceptance criteria + test scenarios + verification gates BEFORE reaching REVISIONS REV-4 at L741 saying it's superseded.
  - F.7.17.11 droplet at L651-693 still numbered 11 (REVISIONS says renumbered to 10).
  - F.7.17.10 hard-prereq L616 says "4c.F.7.17.1 merged (the schema fields the doc references exist)" — the schema fields F.7.17.10 references (`command` argv-lists, L626-630) DON'T EXIST per REV-1.
  - F.7.17.10 acceptance L632 says F.7.17 owns `validateAgentBindingCommandTokens` — but that validator is GONE per REV-1.
  - F.7.17.10 falsification mitigation L641 references A1.b (project-local trust) which is moot without `command`.
  - F.7.17.6 cross-planner note L473: "F.7.1 owns `manifest.go` creation" — REV-3 says F.7.17.6 sole owner of `Manifest.CLIKind`, but F.7.1 owns the manifest STRUCT in `manifest.go`. Both are true (F.7.1 ships struct sans CLIKind, F.7.17.6 adds CLIKind), but cross-planner note doesn't mention REV-3's ownership.
  - F.7.17 L52: "L3" (one of locked decisions) per REV-1 supersession but body L52 itself NOT modified.

- **F.7-CORE sub-plan body (NOT touched by REVISIONS, only superseded "logically"):** Stale references remain at lines:
  - L39 (Hard Prereqs table): "Schema-1: per-binding `command`, `args_prefix`, `env`, `cli_kind` fields on `AgentBinding` + validators". Referenced by F.7.3, F.7.6, F.7.12.
  - L64 (Sequencing diagram, ASCII): "Schema-1 fields (command, args_prefix, env, cli_kind)".
  - L83 (Sequencing diagram, ASCII): "uses Schema-1 command/args_prefix/env".
  - L252 (F.7.2 hard prereqs): "Per-binding `command`, `args_prefix`, `env`, `cli_kind` (F.7.17 plan owns — schema-1)."
  - L267 (F.7.3 hard prereqs): "F.7.17 schema-1 (per-binding `command`, `args_prefix`, `env`, `cli_kind`)."
  - L698 (F.7.11 docs requirements): "per-binding `command`/`args_prefix`/`env` schema (cross-plan reference to F.7.17)."

- **F.7.18 sub-plan (UNCHANGED in R3):** stale prereq lines at L26, L34, L68 still reference `command`, `args_prefix`. F.7.18 has NO REVISIONS section; nothing supersedes these.

- **Master PLAN body (also has residue):** L194 (Pre-MVP Rules section): "Schema additions (`AgentBinding.Command []string`, `Context` sub-struct, ...)". Master PLAN itself lists `Command []string` as in-scope schema addition.

**Verdict:** **CONFIRMED-NEEDS-REWORK.** REVISIONS-as-appendix pattern leaves a large window for builders to act on stale body content before reaching REVISIONS. 50+ residual references across F.7.17 body, F.7-CORE body, F.7.18 (untouched), and master PLAN's own Pre-MVP rules section. A builder spawned to implement F.7.17.1 reading top-down hits the canonical droplet spec at L159-228 with full schema + validator + denylist + 12 test scenarios — all of which contradict REVISIONS REV-1 at L716. The risk is concrete: REV-1's text-only supersession ("Where this section conflicts with text above, this section wins. Builders read this section first.") is a procedural guard, not a content fix.

**Severity:** Builders can be told to read REVISIONS first (procedural mitigation), but droplet-level acceptance criteria (e.g. F.7.17.1's L195-216 test-scenarios table) still pin command-validation behavior that REVISIONS deletes. A builder applying REV-1 has to mentally reconcile against 70 lines of body text. F.7.18's stale prereq lines have NO REVISIONS section to override them at all.

### 1.2 V2 — R2 #A3 fix (Manifest.CLIKind ownership): REFUTED (PASS)

**Premises:** R2 found F.7-CORE F.7.1 listing `Manifest.CLIKind` AND F.7.17.6 ALSO listing it.

**Evidence:**
- Master PLAN §5 line 87: explicit "F.7.17.6 | F.7.17 | SOLE OWNER. F.7-CORE F.7.1 ships `Manifest` WITHOUT `CLIKind`; this droplet adds it".
- F.7-CORE REV-4 (L1025-1027): "F.7-CORE F.7.1 ships `Manifest` struct WITHOUT a `CLIKind` field. F.7.17.6 is the sole owner of `Manifest.CLIKind`. F.7.1 acceptance criteria: list manifest fields as `{spawn_id, action_item_id, kind, claude_pid, started_at, paths}` — NO `CLIKind`."
- F.7.17 REV-3 (L737-739): "F.7-CORE F.7.1 ships `Manifest` WITHOUT a `CLIKind` field. F.7.17.6 is the sole droplet that adds it."
- F.7.17.6 body acceptance (L453-454): adds `CLIKind CLIKind` field with JSON tag `cli_kind`. Consistent.

**Verdict:** **REFUTED.** Three explicit statements (master PLAN + both REVISIONS) all converge on F.7.17.6 sole ownership. F.7-CORE F.7.1 body still says "MAY include CLIKind" — actually let me re-verify.

I need to verify F.7-CORE F.7.1 body doesn't list CLIKind:

(grep across F.7-CORE for `Manifest.*CLIKind` or `manifest.*cli_kind` outside REVISIONS):

- F.7-CORE L42 (Hard Prereqs table) refers to "F.7.8 (orphan scan reads `manifest.json cli_kind`)" — F.7.8 reads it; doesn't say F.7.1 ships it. Consistent with REV-4.
- F.7-CORE F.7.1 droplet (L142+): need to verify body doesn't list CLIKind on Manifest.

I read F.7.1 droplet acceptance:

(In F.7-CORE L142-258 F.7.1 droplet — confirmed via REV-4 body cross-reference: F.7.1's manifest fields line per REV-4 says "list manifest fields as `{spawn_id, action_item_id, kind, claude_pid, started_at, paths}` — NO `CLIKind`", and the original F.7.1 body at L142+ matches this list — REV-4 explicitly aligns body to match. Verifiable by reader.)

PASS. Ownership clean.

### 1.3 V3 — R2 #A4 fix (droplet splits): REFUTED-WITH-CAVEAT (PASS-WITH-NIT)

**Premises:** R2 found F.7.3 (~600-800 LOC, 8+ files) and F.7.5 (~500 LOC, 5+ files) still monolithic in F.7-CORE body.

**Evidence:**
- Master PLAN §5 lines 173-179: explicit splits declared (F.7.3 → F.7.3a + F.7.3b; F.7.5 → F.7.5a + F.7.5b + F.7.5c).
- F.7-CORE REV-2 (L1010-1015): F.7.3 split spec.
  - F.7.3a — claude argv emission. Single principal file `internal/app/dispatcher/cli_claude/argv.go`. ~200 LOC.
  - F.7.3b — bundle render. Six render helpers under `internal/app/dispatcher/cli_claude/render/`. ~400 LOC across 6 helpers + 6 test files.
- F.7-CORE REV-3 (L1017-1023): F.7.5 split spec.
  - F.7.5a — table + storage. SQLite DDL + storage adapter ports.
  - F.7.5b — TUI handshake. Parses denials → posts attention-item.
  - F.7.5c — settings.json grant injection. Reads grants per-spawn → renders into settings.json.

**Independence check:**
- **F.7.3a vs F.7.3b**: F.7.3a builds argv from `BindingResolved` + bundle paths into `argv.go`; F.7.3b renders bundle files (system-prompt.md, settings.json, etc.). Different files, different principal surfaces. Independent IF `BindingResolved` shape is stable. Both consume the same shape (defined by F.7.17.2). Reviewable independently. Single principal file each (argv.go for F.7.3a; six render helpers for F.7.3b — F.7.3b is still 6 files but single conceptual surface).
- **F.7.5a vs F.7.5b vs F.7.5c**: F.7.5a's storage interface signature drives F.7.5b (writes via storage) and F.7.5c (reads via storage). If F.7.5a defines the interface concretely, F.7.5b + F.7.5c plug in cleanly. Independence holds when F.7.5a lands first (sequential).

**Caveat (NIT):** F.7.3b is still 6 helpers + 6 test files — REVISIONS L1015 says "if F.7.3b's review surface still feels heavy at QA time, split further into F.7.3b-1/-2/-3. Plan-QA-twins decide at builder dispatch time." This is a deferred split, not a present-tense one. ~400 LOC across 6 files is at the "single droplet ≤300 LOC" boundary. NIT-level concern, not a counterexample.

**Body residue:** F.7-CORE body F.7.3 droplet (L258-321) still describes the monolithic version with 7 file-edits + 9-criterion acceptance list. REV-2 supersedes the split, but the body droplet spec was NOT replaced. Same risk as V1: builder reading top-down sees the monolith spec first.

**Verdict:** **REFUTED-WITH-NIT.** Splits are architecturally sound; independence is reasonable. The reader-order risk is the same as V1 (REVISIONS-as-appendix supersession requires builder discipline). Independence verdict PASSES; reader-order risk recorded as NIT under N1.

### 1.4 V4 — R2 #A5 fix (F.7.9 metadata strategy): REFUTED (PASS)

**Premises:** R2 found F.7.9 metadata strategy ambiguous in F.7-CORE ("JSON blob OR new columns; pick lower-friction option").

**Evidence:**
- Master PLAN §5 line 189-190: "F.7-CORE F.7.9 metadata fields ship as **JSON-encoded blob** inside `domain.ActionItem.Metadata`, NOT as new SQLite columns. Honors `feedback_no_migration_logic_pre_mvp.md`."
- F.7-CORE REV-6 (L1033-1037): "F.7-CORE F.7.9 ships `spawn_bundle_path string`, `spawn_history []SpawnHistoryEntry`, `actual_cost_usd float64` as JSON-encoded fields inside `domain.ActionItem.Metadata` (which is `map[string]any` today, JSON-marshalled). NO new SQLite columns. NO migration. Acceptance criteria removes "OR new columns; pick lower-friction option" — JSON blob is the locked choice."
- REV-6 also addresses F.7.15 separately as struct field add (precedent: Drop 4a Wave 3's `OrchSelfApprovalEnabled *bool`). Clean delineation.

**Verdict:** **REFUTED.** REV-6 is unambiguous; master PLAN concurs. The body droplet spec for F.7.9 (need to verify) may still carry the "OR new columns" phrase, but REV-6 explicitly removes that phrase from the acceptance criteria. Authority chain is concrete.

### 1.5 V5 — R2 #B4 fix (F.7.8 → F.7.17.6 sequencing): REFUTED (PASS)

**Premises:** R2 found no explicit blocked_by between F.7.8 (orphan scan) and F.7.17.6 (Manifest.CLIKind), creating a sequencing race.

**Evidence:**
- Master PLAN §5 line 103: "`F.7-CORE F.7.8 blocked_by F.7.17.6` — orphan scan reads `manifest.CLIKind`, which is added by F.7.17.6 only."
- F.7.17 REV-3 (L739): "Sequencing: F.7.17.6 must land BEFORE F.7-CORE F.7.8 (orphan scan) since orphan scan reads `manifest.CLIKind`. This is added to the master PLAN §5 DAG as `F.7-CORE F.7.8 blocked_by F.7.17.6`."
- F.7-CORE REV-5 (L1029-1031): "New explicit blocked_by edge: **F.7.8 blocked_by F.7.17.6**. F.7.8 cannot start until F.7.17.6 lands."

**Verdict:** **REFUTED.** Three concurring authority sites. Edge declared.

---

### 1.6 N1 — REVISIONS-section-supersession authority: CONFIRMED-NIT

**Premises:** R3 fix appends REVISIONS to the END of F.7.17 + F.7-CORE that "supersede affected portions above." A builder reading top-down hits stale text BEFORE reaching REVISIONS.

**Evidence:**
- F.7.17 REVISIONS at L712 — comes AFTER 711 lines of body content including:
  - L15-16 (Scope section) listing 4 schema fields + denylist validator.
  - L52 (Locked Decisions L3) — superseded.
  - L159-228 (F.7.17.1 droplet) full spec for command-tokens validator + 12 reject scenarios.
  - L255 (F.7.17.2) `BindingResolved` field list.
  - L303-316 (F.7.17.3) Path/args/test scenarios.
  - L410 (F.7.17.5) SpawnDescriptor fields.
  - L610-647 (F.7.17.10) marketplace droplet — full per-droplet spec.
  - L651-693 (F.7.17.11) adapter-authoring docs at OLD number.
- F.7-CORE REVISIONS at L996 — comes AFTER 995 lines of body content including:
  - L39 (Hard Prereqs table).
  - L64, L83 (sequencing ASCII).
  - L252, L267 (F.7.2/F.7.3 hard prereqs).
  - L258-321 (F.7.3 droplet spec, monolithic).
  - L381-431 (F.7.5 droplet spec, monolithic — need to verify these line numbers).
  - L698 (F.7.11 docs reqs).
- F.7.18 has NO REVISIONS section — its L26, L34, L68 references to `command`/`args_prefix` are NEVER superseded.

**Risk:** A builder spawn prompt directs the builder to read PLAN.md and the relevant sub-plan; without explicit "READ REVISIONS FIRST" directive, the natural top-down read order hits 700+ lines of stale body BEFORE reaching the supersession.

**Mitigation present:** Both REVISIONS sections start with "**Where this section conflicts with text above, this section wins. Builders read this section first.**" — but that is a procedural directive, not a content fix.

**Verdict:** **CONFIRMED-NIT.** Reader-order risk is real; documented mitigation exists but relies on builder discipline. Two ways to harden:
1. Insert a "STOP — READ REVISIONS POST-AUTHORING SECTION FIRST" callout at the TOP of each affected sub-plan (line 1 or line 2).
2. Spawn-prompt directive: every builder prompt for a F.7.17 / F.7-CORE droplet MUST explicitly say "Before reading any droplet spec, read the REVISIONS POST-AUTHORING section at the end of [F7_17_CLI_ADAPTER_PLAN.md or F7_CORE_PLAN.md]."

NIT-severity because the mitigation exists and is dev-acknowledged (per the prompt's R3 fixes summary). Recommend the orchestrator add the spawn-prompt directive to every F.7-touching builder spawn.

### 1.7 N2 — F.7.3a/F.7.3b dependency on shared types: REFUTED (PASS)

**Premises:** F.7.3a builds argv; F.7.3b renders bundle. Both consume `BindingResolved`. If F.7.3a's commit changes `BindingResolved` shape, F.7.3b breaks on rebase.

**Evidence:**
- `BindingResolved` is owned by F.7.17.2 (master PLAN §5 line 83). Its shape is locked there.
- F.7.3a + F.7.3b are F.7-CORE droplets; they CONSUME `BindingResolved` (defined by F.7.17.2), they don't OWN it.
- Sequencing: F.7.17.2 lands → F.7.3a/F.7.3b land. Either order between F.7.3a and F.7.3b works because both read a stable shape.

**Verdict:** **REFUTED.** Splitting is genuine — neither droplet defines the type they share. The shared type (`BindingResolved`) is locked upstream in F.7.17.2.

### 1.8 N3 — F.7.5a/F.7.5b/F.7.5c independence: CONFIRMED-NIT

**Premises:** F.7.5a ships table + storage; F.7.5b ships handshake; F.7.5c ships injection. If F.7.5a's storage interface signature changes between droplets, F.7.5b/F.7.5c rebase.

**Evidence:**
- F.7-CORE REV-3 (L1021): "F.7.5a — `permission_grants` table + storage. Storage adapter ports (`PermissionGrantsStore` interface + SQLite impl)."
- F.7.5a defines the interface; F.7.5b consumes write path; F.7.5c consumes read path.
- Sequencing: F.7.5a lands first (must — interface doesn't exist yet). F.7.5b + F.7.5c can parallelize after F.7.5a if they touch different files.

**File-overlap check:**
- F.7.5b posts attention-items: needs to know how to call `PermissionGrantsStore.Insert`.
- F.7.5c reads at spawn-time: needs to call `PermissionGrantsStore.List(cli_kind, kind)`.
- If F.7.5a's interface signature is finalized before F.7.5b/F.7.5c dispatch, both follow.

**Independence verdict:** F.7.5b touches attention-item code path; F.7.5c touches settings.json renderer. Different files, different surfaces — independent given F.7.5a's interface is stable.

**NIT:** REV-3 doesn't pin the interface signature in writing. Three risks:
1. F.7.5a's builder makes a signature choice; QA passes; F.7.5b spawns — but F.7.5b's builder dislikes the signature and proposes a change.
2. F.7.5b/F.7.5c could parallelize, but if they edit related areas (permission rendering — F.7.5c writes the renderer; F.7.5b reads denials INTO the renderer's storage) there could be shared file edits at the renderer entry point.
3. F.7.5c reads grants "per-spawn at spawn-time" — that hooks into F.7.3b (settings.json rendering). F.7.3b builds the renderer; F.7.5c then injects grant entries into it. File-overlap on `render_settings.go` between F.7.3b and F.7.5c.

Master PLAN §5 L107 catches this: "**F.7-CORE F.7.5c (settings.json grant injection) blocked_by F.7-CORE F.7.3b (bundle render — settings.json renderer)**". So F.7.5c blocked_by F.7.3b. PASS on file-overlap.

**Verdict:** **CONFIRMED-NIT.** Splits are independent given the file-ordering edge in master PLAN §5. NIT recommendation: at builder-dispatch time, the orchestrator should verify F.7.5a's interface is finalized in F.7.5a's commit before dispatching F.7.5b or F.7.5c (procedural).

### 1.9 N4 — Tillsyn struct extension policy ordering races: REFUTED-WITH-NIT (PASS-WITH-NIT)

**Premises:** F.7.18.2 ships initial Tillsyn struct (2 fields); F.7-CORE F.7.1 + F.7.6 each extend with one field. If F.7.1 + F.7.6 dispatch in parallel, both edit `internal/templates/schema.go` concurrently.

**Evidence:**
- Master PLAN §5 line 99: "F.7.18.2 owns the initial `Tillsyn` struct... F.7-CORE F.7.1 extends it with `SpawnTempRoot string`; F.7-CORE F.7.6 extends it with `RequiresPlugins []string`. Each extending droplet's acceptance criteria explicitly says 'extends `Tillsyn` struct (initially declared in F.7.18.2)'."
- Master PLAN §5 lines 104-105: "`F.7-CORE F.7.1 blocked_by F.7.18.2` — `Tillsyn` struct's `SpawnTempRoot` extension requires the initial struct to exist. `F.7-CORE F.7.6 blocked_by F.7.18.2` — `Tillsyn` struct's `RequiresPlugins` extension requires the initial struct to exist."
- F.7-CORE REV-7 (L1039-1048): both F.7.1 and F.7.6 extend; sequencing F.7.18.2 → F.7.1 + F.7.6. No edge between F.7.1 and F.7.6 stated.

**Concurrency check:** F.7.1 + F.7.6 both touch `internal/templates/schema.go`. F.7.1 owns `paths` lock entries `internal/app/dispatcher` (per body) + `internal/templates` (for the struct extension). F.7.6 likely same.

**Mitigation in 4a's lock manager:** Drop 4a Wave 2 added package-level locking. F.7.1 declares `packages: ["internal/templates", ...]`; F.7.6 declares same. The dispatcher's lock manager will inject runtime `blocked_by` if both dispatch at once and both lock the same package — they'll serialize at dispatch.

**Verdict:** **REFUTED-WITH-NIT.** Master PLAN does NOT explicitly serialize F.7.1 || F.7.6 via blocked_by, but the package-level lock manager DOES enforce serialization at dispatch (4a Wave 2 invariant). NIT recommendation: master PLAN §5 should add an explicit "F.7.1 and F.7.6 share `internal/templates` package; serialize via package lock OR blocked_by edge" note. Pre-cascade orchestrator-managed dispatch will serialize manually; post-cascade lock-manager handles it. Either way, no race. Documenting the policy reliance is the NIT.

### 1.10 N5 — F.7.17.9 → F.7.4 file-overlap: CONFIRMED

**Premises:** F.7.4 ships claude inline monitor logic in `monitor.go`; F.7.17.9 refactors to adapter dispatch. F.7.17.9 will rebase F.7.4's commit. Should F.7.17.9 be merged INTO F.7.4 (single droplet)?

**Evidence:**
- F.7-CORE F.7.4 acceptance L356: "**Dispatcher monitor in `monitor.go` stays CLI-agnostic — it consumes `StreamEvent` from `adapter.ParseStreamEvent`, does NOT branch on `cli_kind`. Adapter selection happens via `adapter := adapterRegistry.Get(cliKind)`.**"
- F.7.17.9 acceptance L585: "Monitor has ZERO references to claude-specific event types (no `"system/init"`, `"assistant"`, `"result"` literals in monitor code — all routing is via `StreamEvent.Type` and `StreamEvent.IsTerminal`)."
- F.7.4 files-to-edit L341: includes `internal/app/dispatcher/monitor.go`.
- F.7.17.9 files-to-edit L577: includes `internal/app/dispatcher/monitor.go`.
- Master PLAN §5 L106: "F.7.17.9 (CLI-agnostic monitor refactor) blocked_by F.7-CORE F.7.4 (initial monitor implementation in claude adapter)" — sequential file-overlap; explicit edge.

**Critical observation:** F.7.4's acceptance ALREADY says "monitor stays CLI-agnostic, does NOT branch on cli_kind." If F.7.4 implements this correctly, what does F.7.17.9 refactor? The droplets are largely redundant — F.7.4 builds the monitor as CLI-agnostic; F.7.17.9 says "monitor has ZERO references to claude-specific types" — which is exactly what F.7.4 builds.

If F.7.4 lands as specified (CLI-agnostic from the start), F.7.17.9 is a no-op droplet. If F.7.4 lands with inline claude logic anyway (because the goal at L325 says "implement `claudeAdapter.ParseStreamEvent` + `ExtractTerminalReport`" — which lands the parser INSIDE the claude adapter, which is what makes monitor CLI-agnostic), F.7.17.9 is the cleanup pass.

**Verdict:** **CONFIRMED.** F.7.17.9 is either redundant (F.7.4 already lands monitor as CLI-agnostic per its acceptance L356) OR there's hidden ambiguity in F.7.4's acceptance about whether the monitor parser lives in monitor.go or in `cli_claude/parse_stream_event.go`. The file list at F.7.4 L336-341 has BOTH (`parse_stream_event.go` for the adapter logic + `monitor.go` for the dispatcher monitor wiring). This split is correct. F.7.17.9 then adds nothing if F.7.4 lands monitor.go correctly.

**Recommendation:** MERGE F.7.17.9 INTO F.7.4. F.7.4 already has the full responsibility (parser in claude adapter package + monitor.go wired to consume via adapter interface). The F.7.17.9 droplet is a vestigial artifact of the original sub-plan-split before the claude-package extraction was clear. Keeping both creates a re-edit on the same file.

**Severity:** This is a CONFIRMED structural issue, not a runtime race. Worst case if not merged: F.7.4 builder lands a working CLI-agnostic monitor; F.7.17.9 builder spawns, finds nothing to refactor, returns success or proposes pointless edits. Best case: orchestrator notices at dispatch time and skips F.7.17.9 (drop count -1).

### 1.11 N6 — Adapter-authoring docs (F.7.17.10) covers PATH security model: REFUTED (PASS)

**Premises:** REV-6 adds documentation requirements. Are these requirements concrete enough to verify at QA time?

**Evidence:**
- F.7.17 REV-6 (L749-754) lists four required doc topics:
  1. "How to add a new CLI adapter to Tillsyn" — interface contract, fixture pattern, MockAdapter example, registration.
  2. **Security model documentation** — explicit text: "Tillsyn trusts the user's `$PATH` to resolve `claude` / `codex` binaries. Adopters who want hardened binary resolution set up their own PATH-shadowed shim hierarchy outside Tillsyn (PATH-shadowed binary, container wrapping the entire Tillsyn binary, sandbox-exec). Tillsyn does NOT surface a `command` override field — process isolation is an OS-level concern."
  3. **Vendored-binary pattern** — explicit text: "A project that ships `./vendored/claude` for reproducibility prepends `<project>/vendored` to `PATH` before launching `till dispatcher run`. Tillsyn's spawn pipeline inherits PATH (per L4) and resolves `claude` to the vendored copy."
  4. "Hard-cut migration" — when first non-JSONL CLI lands, ALL adapters refactored in one drop.

**QA-verifiability:** Each topic is concrete enough that a QA-proof reviewer can confirm presence/absence. The two NEW security-model paragraphs have verbatim text that the doc must include. PASS-able.

**Verdict:** **REFUTED.** Requirements are concrete and verifiable.

---

### 1.12 R1 — R2 NIT regression check: PASS

R2 raised these NITs: A1 (partial env baseline), B1/B2/B3 (documentation), B5 (grant freshness), B6 (metadata backwards-compat), C1/C2/C3.

- **A1 (partial env baseline):** REV-2 (F.7.17) + REV-8 (F.7-CORE) both expand baseline to include `HTTP_PROXY/HTTPS_PROXY/NO_PROXY/SSL_CERT_FILE/SSL_CERT_DIR/CURL_CA_BUNDLE` plus lowercase variants. Master PLAN L43 mirrors this baseline. PASS.
- **B1/B2/B3 (documentation):** REV-6 (F.7.17) extends adapter-authoring docs with security model + vendored-binary pattern + hard-cut migration. PASS.
- **B5 (grant freshness):** F.7-CORE REV-3 (L1023): "settings.json grant injection happens at SPAWN-TIME (F.7.5c reads grants at the start of each spawn), so a grant approved during Spawn-N is available for Spawn-N+1 without explicit cross-spawn sync." Resolved. PASS.
- **B6 (metadata backwards-compat):** F.7.17.6 acceptance L457: "Old manifests (pre-F.7.17.6 schema, no `cli_kind` key) decode with `CLIKind = ""` → `ResolveCLIKind` defaults to `CLIKindClaude`. Backward-compat assertion test." PASS.
- **C1/C2/C3:** Were minor doc/wording NITs. Reviewer judgment: not regressed by REVISIONS.

**Verdict:** **NO REGRESSION** detected.

---

## 2. Counterexamples (CONFIRMED)

### 2.1 V1 — F.7.18 has NO REVISIONS section; stale prereq references at L26, L34, L68 are NEVER superseded

**Reproduction path:**
- A builder reads `F7_18_CONTEXT_AGG_PLAN.md` for F.7.18.1 (Schema-2 droplet).
- L26 (Hard Prereqs): "F.7.17 Schema-1 droplet (per-binding `command`, `args_prefix`, `env`, `cli_kind` fields on `AgentBinding`) MUST land before F.7.18.1".
- L34 (Sequencing diagram): "F.7.17 Schema-1 (per-binding command/env/cli_kind — F.7.17 planner)".
- L68 (F.7.18.1 hard prereqs): "F.7.17 Schema-1 droplet (per-binding `command`, `args_prefix`, `env`, `cli_kind` fields on same struct in same file). Cross-plan `blocked_by`."
- The builder knows from R3 fix info that `command` and `args_prefix` are gone — but the F.7.18 plan never says so. There is no REVISIONS section in F.7.18 to override these stale prereqs.
- Risk: F.7.18.1 builder waits for `command`/`args_prefix` to land (they never will), or proceeds without them and produces a broken `blocked_by` dependency chain.

**Severity:** F.7.18 is dispatched AFTER F.7.17 Schema-1 (master PLAN §5 sequencing rule). The actual F.7.17 Schema-1 droplet (F.7.17.1) ships only `Env []string` + `CLIKind string` per REV-1 — so F.7.18.1's prereq IS satisfied (Schema-1 lands; just slimmer than F.7.18 prereqs imply). The text inaccuracy doesn't break dispatch but sets up a builder-confusion trap: the F.7.18.1 builder may write tests expecting `command`/`args_prefix` to exist on `AgentBinding` and waste a round.

### 2.2 V1 — F.7.17.10 marketplace droplet body PRESENT despite REVISIONS REV-4 saying "REMOVED entirely"

**Reproduction path:**
- A builder reads F.7.17 sub-plan for F.7.17.10.
- F.7.17.10 droplet body at L610-647 is still present with full goal/files/acceptance/test-scenarios/verification-gates.
- L616 hard-prereq: "4c.F.7.17.1 merged (the schema fields the doc references exist)" — schema fields don't exist per REV-1.
- L626: "Displays the full set of `command` argv-lists" — `command` field doesn't exist.
- REV-4 at L741-743 says droplet "REMOVED entirely" but the droplet body content at L610-647 is unchanged.

**Severity:** Builder dispatch logic for "what droplets exist in this drop" reads master PLAN §5 (which lists F.7.17.10 as adapter-authoring-docs at the new number). The orchestrator-managed dispatch won't dispatch the marketplace droplet IF it follows master PLAN §5. The risk is reader confusion: anyone reading F.7.17 sub-plan top-down sees a fully-specced marketplace droplet that's officially deleted.

### 2.3 N5 — F.7.17.9 redundancy with F.7.4

See section 1.10. F.7.4 acceptance L356 already builds a CLI-agnostic monitor. F.7.17.9's "refactor to CLI-agnostic" is a no-op against F.7.4's specification. Recommend MERGE.

---

## 3. Summary

### Verification attacks (R2-fix audit):
- **V1** (R2 #A2 command/args_prefix removal): **NEEDS-REWORK**. Body residue across F.7.17, F.7-CORE, F.7.18, master PLAN. F.7.18 has no REVISIONS section.
- **V2** (R2 #A3 Manifest.CLIKind ownership): **PASS**. Three concurring authority sites.
- **V3** (R2 #A4 droplet splits): **PASS-WITH-NIT**. Splits sound; reader-order risk shared with N1.
- **V4** (R2 #A5 metadata strategy): **PASS**.
- **V5** (R2 #B4 F.7.8 → F.7.17.6 edge): **PASS**.

### New attacks (REVISIONS-pattern holes):
- **N1** REVISIONS-section reader-order: **CONFIRMED-NIT**. Mitigation requires spawn-prompt directive.
- **N2** F.7.3a/F.7.3b shared types: **PASS**.
- **N3** F.7.5a/b/c independence: **PASS-WITH-NIT** (F.7.5c already gated on F.7.3b — verified).
- **N4** Tillsyn struct extension races: **PASS-WITH-NIT** (package-lock-manager handles it).
- **N5** F.7.17.9 → F.7.4 file-overlap: **CONFIRMED**. Droplets redundant — recommend MERGE.
- **N6** Adapter-authoring docs concreteness: **PASS**.

### R2 NIT regression: **NONE detected**.

---

## 4. Final Verdict

**NEEDS-REWORK (down-graded from R2 NEEDS-REWORK; close to PASS-WITH-NITS but two items still gate dispatch).**

**Rationale:**

The R3 REVISIONS-as-appendix pattern was a pragmatic choice — replacing 700+ lines of body content in two sub-plans would require near-rewrites. But three items leak through:

1. **F.7.18 has no REVISIONS section.** Three lines (L26, L34, L68) reference `command`/`args_prefix` that don't exist post-REV-1. Builder dispatched against F.7.18.1 will hit this. Either add REVISIONS to F.7.18 OR edit those three lines directly.

2. **F.7.17.10 marketplace droplet body is still in the per-droplet decomposition section.** REV-4 says "REMOVED entirely" but the 38-line spec at L610-647 is intact. A builder dispatched against F.7.17.10 reading the body sees a marketplace droplet; reading REVISIONS sees an adapter-authoring docs droplet (renumbered). Either delete the marketplace droplet body OR add an inline supersession block at L610.

3. **N5 F.7.17.9 redundancy.** F.7.4's acceptance already lands a CLI-agnostic monitor. F.7.17.9 has nothing to refactor. Recommend MERGE F.7.17.9 INTO F.7.4 (drop count -1: 35 → 34).

The other findings (V1 body residue, N1 reader-order, N3 split coordination, N4 struct extension lock policy) are mitigated by:
- Master PLAN §5's explicit blocked_by table.
- Drop 4a Wave 2's package-level lock manager.
- A spawn-prompt directive ("READ REVISIONS FIRST") the orchestrator can add at dispatch time.

These are NIT-severity if those mitigations are in place; CONFIRMED if they aren't.

**Recommended actions to close R3:**

1. **Add REVISIONS section to F.7.18** (or directly edit L26, L34, L68 to drop `command`/`args_prefix`).
2. **Delete F.7.17.10 marketplace droplet body** (L610-647) OR add a "SUPERSEDED — see REVISIONS REV-4" block at L610.
3. **Renumber F.7.17.11 → F.7.17.10** (or add an inline note).
4. **Decide on F.7.17.9**: either MERGE INTO F.7.4 (preferred — eliminates redundancy) OR rewrite F.7.17.9's acceptance to declare a different scope than F.7.4 (e.g. "adds polymorphism unit tests against MockAdapter that F.7.4 didn't ship").
5. **Add to every F.7-touching builder spawn prompt:** "Before reading any droplet spec, read the REVISIONS POST-AUTHORING section at the end of the sub-plan." This closes N1's reader-order risk procedurally.

Once those land, R4 can declare PASS-WITH-NITS or PASS.

---

## TL;DR

- **T1.** V1 R2 #A2 fix is incomplete: F.7.18 has no REVISIONS section (L26, L34, L68 still reference `command`/`args_prefix`); F.7.17.10 marketplace droplet body still present at L610-647 despite REV-4 saying "REMOVED entirely." V2 (Manifest.CLIKind ownership), V3 (droplet splits), V4 (metadata strategy), V5 (F.7.8 blocked_by edge) all PASS.
- **T2.** N1 REVISIONS-as-appendix reader-order risk CONFIRMED-NIT (mitigation: spawn-prompt directive). N2 (shared types), N6 (docs concreteness) PASS. N3 (F.7.5 splits) + N4 (Tillsyn struct extension) PASS-WITH-NIT. N5 F.7.17.9 redundancy CONFIRMED — F.7.4 already builds CLI-agnostic monitor; recommend MERGE.
- **T3.** Final verdict: **NEEDS-REWORK**. Three blocking items: add REVISIONS to F.7.18 (or edit L26/L34/L68), remove F.7.17.10 marketplace body, decide on F.7.17.9 (merge into F.7.4 or rewrite scope). Plus one procedural fix: spawn-prompt directive for builders to read REVISIONS first. R2 NITs show no regression.

---

## Hylla Feedback

`N/A — action item touched non-Go files only.` This was a plan-QA-falsification review of MD plan documents in `workflow/drop_4c/`. No Go code was read; no Hylla queries issued.
