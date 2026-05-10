# Plan-QA-Proof Round 1 — DROP_4c.6.W2

**Reviewer:** L2 plan-QA-proof (Round 1, 2026-05-09)
**Sub-drop:** `4c.6.W2` — `till init`
**Plan under review:** `workflow/drop_4c_6/DROP_4c.6.W2_TILL_INIT/PLAN.md`
**Sibling ledger:** `workflow/drop_4c_6/DROP_4c.6.W2_TILL_INIT/_BLOCKERS.toml`
**L1 contract:** `workflow/drop_4c_6/PLAN.md` lines 117-133
**Source-of-truth:** `workflow/drop_4c_6/SKETCH.md` § 9 + § 26.W2
**Verdict (single-line preview):** PASS WITH FINDINGS — atomic-decomposition + parallelization + specify-block axes are sound; D3 and D8 carry small acceptance-bullet drift that the builder spawn prompt should sharpen, and one shipped-but-not-wired vector merits an explicit consumer-tie note.

---

## 1. Findings

### Atomic-decomposition axis

- 1.1 [Axis: atomic-decomposition] [severity: medium] D3 combines six concerns in one droplet (init_cmd.go skeleton + JSON struct + dispatch + flag wiring + main.go register + help.go entry + JSON-payload table-test). The planner explicitly flagged a sizing watch ("escalate if combined LOC exceeds ~120") but did NOT pre-decide a split. Concrete LOC estimate: skeleton (~25) + JSON struct + parser + group-validation (~30) + dispatch wrapper (~10) + main.go register-line edit (~3) + help.go entry (~14) + table-driven test (~60-80 across 4 cases) = production code 80-100 LOC + tests 60-80 LOC. This sits at the upper edge of the till-go atom (1-4 code blocks, 80-120 LOC + tests, ideally ONE production file — D3 touches THREE production files: `init_cmd.go`, `main.go`, `help.go`). Three+ production files in one droplet is flagged as a smell ("under-decomposed") in `~/.claude/agents/go-planning-agent.md:143`. → PLAN.md lines 86-107 (D3 bullet) + sizing watch lines 105-107 → either pre-decide the D3a/D3b split now (D3a: skeleton + register + help-entry; D3b: JSON-payload struct + parser + table-test in `init_cmd.go` only) and re-graph blockers (D3b would `Blocked by: D3a`; D7.5 would still `Blocked by: D3a` for the `main.go` lock; D4 would `Blocked by: D3b`), or formalize an in-spawn-prompt LOC-trip-wire instruction so the builder calls a halt-and-escalate at the > 120 LOC threshold rather than after the fact.

- 1.2 [Axis: atomic-decomposition] [severity: low] D7.5 ships ~80 LOC verbatim-port + ~60 LOC test-port (both ports are mechanical lifts of `runInitDevConfig` body and its two test functions per `main.go:2040-2094` + `main_test.go:2906-3011`). Touches three production files (`install_cmd.go` NEW, `main.go` register-line, `help.go` entry — same three-file shape as D3). On its own this sits within the atomic envelope because the port is mechanical, but the same "ideally one production file" smell applies; a dev acknowledgment that `main.go` + `help.go` edits are one-line surface changes (not real production work) belongs in D7.5's RiskNotes/Notes-for-builder so plan-QA falsification doesn't double-attack the smell. → PLAN.md lines 192-215 (D7.5 bullet) → add a one-line note to D7.5's `Notes-for-builder` clarifying that the `main.go` (1 line) + `help.go` (~14 line table entry) edits are register-only surgery; the substantive code lives in `install_cmd.go`.

### Parallelization-graph axis

- 1.3 [Axis: parallelization-graph] [severity: low] _BLOCKERS.toml D7.5 entry omits the cross-file dependency on the **same line in `main.go`** as D8. D7.5 amends `cmd/till/main.go:1904` to add `installCmd` to `rootCmd.AddCommand(...)`; D8 amends the SAME line at `cmd/till/main.go:1904` to remove `initDevConfigCmd`. _BLOCKERS.toml correctly lists `D8.blocked_by = ["4c.6.W2.D3", "4c.6.W2.D7.5"]` (line 57 — `D7.5` is in the list), so the same-line conflict IS serialized. The PLAN.md narrative at line 246 mentions "D8 modifies BOTH `main.go` and `init_cmd.go`-adjacent surfaces" which is technically correct but obscures that the MOST-load-bearing same-line is `1904` (the `rootCmd.AddCommand(...)` call). Cosmetic clarification only — the lock graph is structurally correct. → PLAN.md line 246 + _BLOCKERS.toml line 57 → optional one-liner clarifying that `cmd/till/main.go:1904` is the touched-by-everyone line and the chain D3 → D7.5 → D8 reflects "add init", "add install", "remove init-dev-config" in that order on that single line.

### Specify-block well-formedness axis

- 1.4 [Axis: specify-block-well-formedness] [severity: medium] None of the eight droplet bullets carry an explicit `Objective` / `ValidationPlan` / `RiskNotes` / `ContextBlocks` / `KindPayload` per the planning-agent Specify-pass spec (`~/.claude/agents/go-planning-agent.md:172-192`). Each droplet has Paths + Packages + Acceptance + Blocked-by + Notes-for-builder, which covers 3-4 of the 6 fields semantically (Paths/Packages = `KindPayload.changes` shape; Acceptance = `AcceptanceCriteria`; Notes-for-builder folds `RiskNotes` + `ContextBlocks` informally). But the L1 W2 sub-plan container DID populate the spec at `workflow/drop_4c_6/PLAN.md` lines 122-129 (an L1 Acceptance contract), so the L2 plan's omission may be "L1 covers it" — confirm with orchestrator. The spec is forward-compatible-mandatory once Tillsyn cascade lands typed `ActionItemMetadata` (post-Drop-4c.7); shipping without it now leaves a migration gap. → PLAN.md droplet sections D1-D8 → add a brief Specify-summary block at the top of the L2 PLAN.md (before "## Planner") that names the L2 plan's `Objective` + `ValidationPlan` + `RiskNotes` (the L1 already enumerates `AcceptanceCriteria` and `ContextBlocks`); per-droplet Specify is acceptable to skip because the inherit-by-parent-reference rule in `~/.claude/agents/go-planning-agent.md:198` covers it.

- 1.5 [Axis: specify-block-well-formedness] [severity: low] D5 acceptance bullet for the `.gitignore` ensure ("Uses `configmerge` if the merge logic fits; otherwise hand-written line-presence check is acceptable") leaves the choice between vendored-dep vs. hand-rolled to the builder without naming the criterion. `configmerge` is ~12kB of dep that vendors entire-file/section TOML merging per SKETCH §9.6 — it does NOT obviously fit a single-line `.gitignore` append. A more decisive plan-side direction would prevent the builder from spending plan-QA-falsification cycles defending the choice. → PLAN.md D5 bullet line 136 → name the acceptable case explicitly, e.g.: "Use `configmerge` ONLY if it exposes a public `EnsureLine(file, line string)` helper; otherwise use a hand-written `os.ReadFile` + `bytes.Contains` + `os.WriteFile` (atomically via fsatomic) sequence — the hand-written path is the default."

### Multi-level-decomposition axis

- 1.6 [Axis: multi-level-decomposition] [severity: low] PASS — planner authored ONE level (8 atomic droplets D1-D8 plus the OQ#3-introduced D7.5 = 9 droplets total). No L3 sub-plans authored. Droplet count is justified by the work (vendor 2 packages + 5 sequential init pipeline stages + 1 install-cmd extraction + 1 cleanup), well within the "no cap on children, atom-size is the cap" rule (`~/.claude/agents/go-planning-agent.md:151-155`). No finding; recorded for completeness and to confirm the multi-level-decomposition axis was deliberately covered.

### Shipped-but-not-wired axis

- 1.7 [Axis: shipped-but-not-wired] [severity: medium] D7.5's `runInstall` has a CLEAR consumer (the `installCmd.RunE` D7.5 itself defines), so the per-droplet wiring is intact. BUT the L1 W2 contract bullet "`till init-dev-config` removed; install-time setup folds into `till install`" (`workflow/drop_4c_6/PLAN.md:128`) is the cross-droplet wiring claim — and the chain that fulfills it is **D7.5 (add till install) → D8 (remove init-dev-config)**. D8's `Blocked by: D3, D7.5` correctly enforces ordering, and D8's pre-flight builder note at PLAN.md line 234 explicitly checks `TestRunInstall_*` test presence before deletion. This is well-wired. The only residual risk is **what happens if D7.5 names the new tests differently from `TestRunInstall_CreatesDebugConfig` / `TestRunInstall_UpdatesExistingConfig`** — D8's pre-flight check at line 234 hard-codes those names. PLAN.md D7.5 line 207 also hard-codes those names. Names match, so the wiring is sound — but a planner-side note that the names are a contract between D7.5 and D8 (not internal builder choice) is missing. → PLAN.md D7.5 line 207 → add a one-line "name-contract" note: "These test names are a hard contract — D8's pre-flight check in builder notes hard-codes them. If you rename, update D8 first."

- 1.8 [Axis: shipped-but-not-wired] [severity: medium] D7.5 acceptance criteria do NOT explicitly require an integration assertion that a `till install` invocation actually creates the dev config — it requires the ported test bodies (`TestRunInstall_CreatesDebugConfig` etc.) which DO exercise this end-to-end via `run(context.Background(), []string{"--app", "tillsyn-init", "install"}, &out, io.Discard)`. So the shipped-but-not-wired axis IS satisfied by the ported test (which exercises route → `installCmd` → `runInstall` → file creation). However, the PLAN.md D7.5 acceptance bullet at line 207 calls the new tests "`TestRunInstall_CreatesDebugConfig` and `TestRunInstall_UpdatesExistingConfig`" without naming the per-test content explicitly — it relies on the builder reading `main_test.go:2906-3011` and porting verbatim. If the builder ports only the function names + minimal scaffolding (without the substantive `[]string{"install"}` invocation chain that exercises the `run(...)` route), the consumer-tie is hollow. → PLAN.md D7.5 acceptance line 207 → strengthen to: "Each new test MUST invoke `run(context.Background(), []string{..., "install"}, &out, io.Discard)` (NOT call `runInstall` directly) so the route → cobra-command → RunE chain is exercised end-to-end. Verify wiring via `mage test-func ./cmd/till TestRunInstall_CreatesDebugConfig`."

---

## 2. Missing Evidence

- 2.1 [Axis: specify-block-well-formedness] PLAN.md `## OQ#3 Verification` section (lines 17-35) is a strong evidence block, but it does NOT cite the L1 directive's exact wording at the L1 PLAN.md lines 121-128 verbatim. The verification runs `git grep` and reports findings, but the L1 directive premise is paraphrased ("the L1 directive's premise that ...") rather than quoted. A QA falsifier could attack on "is the planner reading the L1 spec accurately?" If the L1 spec is updated post-decomposition, the paraphrase risks drift. Routing: optional improvement — would add < 5 lines.

- 2.2 [Axis: shipped-but-not-wired] D6's `.mcp.json` schema-verification clause at PLAN.md line 158 says the builder "MUST verify Claude Code's `.mcp.json` shape via Context7" — but does NOT specify what to do if Context7 returns a stale or partial answer. The fallback "read an authoritative live `.mcp.json` from a known-good install (the dev's own machine — escalate to orchestrator for the path)" is correct, but the FAIL mode (silent shape mismatch breaking MCP registration) is the load-bearing risk, not the schema-discovery process. A test acceptance that explicitly verifies the written `.mcp.json` is parseable by Claude Code's actual `.mcp.json` loader (or, since that's not in this repo, by a JSON-schema validator with a hardcoded expected shape) would close the gap. Routing: dev decision — accept the manual verification path as good enough for D6, OR strengthen to a structural-schema test. The current spec is acceptable because `.mcp.json` registration is opt-in and re-runnable.

- 2.3 [Axis: parallelization-graph] D5 cross-package blocker on D1 + D2 is correctly listed in _BLOCKERS.toml (line 33: `blocked_by = ["4c.6.W2.D1", "4c.6.W2.D2", "4c.6.W2.D4"]`) and PLAN.md (line 145). But D5's import order (whether `internal/vendor/fsatomic` AND `internal/vendor/configmerge` BOTH need to compile before D5 starts, OR only ONE of them per usage) is implicit. If D5 only uses `fsatomic` and not `configmerge` (because finding 1.5 above suggests `configmerge` isn't a fit for the `.gitignore` use case), the D5 → D2 blocker becomes vestigial — but harmless (over-serialized; pessimistic graph). Verification: confirm with the dev whether D2 is load-bearing for D5 or whether finding 1.5's recommendation makes D2 a peer-dep purely for the file-copy path. If purely for the file-copy path, `configmerge` is used to merge embedded `agents.example.toml` content with any user-provided overrides — but the spec at PLAN.md D5 line 135 says `copyAgentsTOML(destDir)` "copies embedded `agents.example.toml` → `<destDir>/agents.toml` atomically" — that's a copy, not a merge. So `configmerge` may NOT be load-bearing for D5 at all. Routing: confirm with planner whether D2 is genuinely needed in D5, or only required as a parallel-track NEW package that D6+/D7+/W3+ might consume.

---

## 3. Summary

- **Verdict: PASS WITH FINDINGS** (no high-severity findings; 4 medium + 4 low across the 5 verification axes).
- **Finding count:** 8 total (1 atomic-decomposition medium, 1 atomic-decomposition low, 1 parallelization-graph low, 1 specify-block medium, 1 specify-block low, 1 multi-level-decomposition low/PASS-recorded, 2 shipped-but-not-wired medium). 3 missing-evidence routing notes (§2.1-2.3).
- **D7.5 addition verdict: CORRECT.** The OQ#3 verification block (PLAN.md lines 17-35) confirms via `git grep` that `till install` does NOT exist today (gate runs reproduced same result during this review: zero matches for `runInstall` / `installCmd`; three matches for `runInitDevConfig` all in `main.go`; 14 matches for `init-dev-config` across `main.go`/`help.go`/`main_test.go`). The disposition (add D7.5 BEFORE D8 removes `init-dev-config`) is sound and the resulting blocker graph (D3 → D7.5 + D3 → D8 + D7.5 → D8) is well-formed. The only follow-up is the test-name contract finding (1.7) and the route-not-direct-call finding (1.8), both medium severity.
- **D3 sizing verdict: BORDERLINE; PLAN-MODIFY RECOMMENDED.** D3 touches 3 production files (`init_cmd.go`, `main.go`, `help.go`) which falls outside the "ideally one production file" rule and into the "smell — under-decomposed" category at 3+ files (`go-planning-agent.md:143`). Combined LOC sits at the upper edge of the till-go atom (80-120 LOC + tests). Recommendation: either pre-split into D3a (skeleton + register + help-entry) and D3b (JSON parser + table-test) with `D3b.blocked_by = [D3a]` and `D7.5.blocked_by = [D3a]` (only the main.go-touching part) and `D4.blocked_by = [D3b]`, OR formalize an in-spawn-prompt LOC trip-wire so the builder halts at > 120 LOC. The planner already flagged this watch in PLAN.md lines 105-107; QA's preference is the pre-split because LOC trip-wires fire late and force re-planning under failure pressure. Either path is acceptable; orchestrator chooses.
- **Round 1 close:** PASS-WITH-FINDINGS verdict means W2 sub-plan can advance to W2 builds IF the orchestrator chooses to address findings 1.1 (D3 split or trip-wire), 1.4 (Specify-summary block), 1.5 (D5 `.gitignore` decisiveness), 1.7 (D7.5 test-name contract note), and 1.8 (D7.5 route-vs-direct-call) inline before launching builders. Findings 1.2, 1.3, and 1.6 are advisory only. Missing-evidence items §2.1-2.3 are dev/orchestrator decisions; none block.

---

## 4. Hylla Feedback

`N/A — sub-drop touched non-Go files (PLAN.md, _BLOCKERS.toml, SKETCH.md) and used Read/Grep against committed Go (main.go, help.go, main_test.go, service.go) where Hylla would have been the first-choice tool but the surfaces under review (cmd/till command-registration block + help-table + test names) are simple enough that `git grep -n` was the highest-evidence-yield tool.` This isn't a Hylla miss — it's an "I never queried Hylla" pass; recorded here for completeness because the project Hylla-Feedback rule (`CLAUDE.md` § Code Understanding Rules item 1) requires the section even when Hylla wasn't engaged. No Hylla query issued; no fallback miss to log.

---

## TL;DR

- **T1.** D3 is borderline-oversized (3 production files, 80-120 LOC + tests at the upper edge of the till-go atom); pre-split or LOC trip-wire recommended. D7.5 is right-sized as a mechanical port. Other droplets pass atomic-decomposition cleanly.
- **T2.** Parallelization graph is structurally correct: _BLOCKERS.toml mirrors PLAN.md inline `Blocked by:` bullets, the cmd/till same-file-lock chain is serialized D3 → D4 → D5 → D6 → D7, and the cross-package D5 blocker on D1 + D2 is captured. One cosmetic same-line-clarity note for `cmd/till/main.go:1904` (no graph defect).
- **T3.** Specify-block fields are partially populated — Paths/Packages/Acceptance/Blocked-by carry the structural content, but Objective/ValidationPlan/RiskNotes/ContextBlocks/KindPayload are not labeled per the planning-agent Specify spec; an L2-summary block before the Planner section would close the gap forward-compatibly.
- **T4.** Multi-level decomposition is correct (single level, 9 atomic droplets, no L3).
- **T5.** Shipped-but-not-wired axis: D7.5's consumer-tie via `installCmd` is intact; the test-name contract between D7.5 and D8 needs a one-line note; D7.5 acceptance should require route-exercising tests (not direct `runInstall` calls). Verdict: PASS-WITH-FINDINGS; orchestrator chooses which findings to address inline before W2 builders fire.

---

## Round 2 Verdict

**Reviewer:** L2 plan-QA-proof (Round 2, 2026-05-09)
**Sub-drop:** `4c.6.W2` — `till init`
**Plan under review (round 2):** `workflow/drop_4c_6/DROP_4c.6.W2_TILL_INIT/PLAN.md` (post-W2-FF1/FF2/FF3/FF4/FF5 application)
**Sibling ledger (round 2):** `workflow/drop_4c_6/DROP_4c.6.W2_TILL_INIT/_BLOCKERS.toml` (D2 commented-out; D3 → D3a/D3b split)
**Verdict (single-line preview):** **PASS** — all five round-1 findings RESOLVED with evidence; round-2 attack passes (D3a/D3b sizing, D7.5 verbatim port, D8 pre-flight cross-ref, audit-trail audit, blocker comment-out pattern) produce no counterexample; one round-2 NIT (W2-PF1, low) flagged for builder spawn-prompt clarity.

### Round-1 finding resolution

#### W2-FF1 (D3 atomic-decomposition split): **RESOLVED**

D3 is now D3a + D3b in PLAN.md and _BLOCKERS.toml.

- **D3a** (PLAN.md lines 77-99): touches 3 files but two are surgical edits — `cmd/till/init_cmd.go` (NEW skeleton, ~25 LOC + stub `RunE` + flag wiring), `cmd/till/main.go` (1-line `rootCmd.AddCommand(...)` amendment at line 1904, verified at `cmd/till/main.go:1904` `rootCmd.AddCommand(...)`), `cmd/till/help.go` (~14-line table entry analogous to lines 377-390). The "skeleton + register + help-entry" allocation gives D3a roughly ~40 LOC of authored code + a smoke test — well within atomic-droplet envelope.
- **D3b** (PLAN.md lines 101-119): `init_cmd.go`-only — replaces the D3a JSON-stub with parser + group-validation, adds table-test. ~30-50 LOC of production + ~60-80 LOC of tests. Fits a single droplet cleanly because both pieces are tightly cohesive (parser body + the test that proves the parser works).
- **Blocker chain rewired** (verified in _BLOCKERS.toml):
  - D3a `blocked_by = []` (line 27).
  - D3b `blocked_by = ["4c.6.W2.D3a"]` (line 34) — same-file lock on `init_cmd.go`.
  - D4 `blocked_by = ["4c.6.W2.D3b"]` (line 41) — was `D3` round-1; now correctly D3b (last `init_cmd.go` writer).
  - D7.5 `blocked_by = ["4c.6.W2.D3a"]` (line 67) — was `D3` round-1; now correctly D3a (D3a owns `main.go` + `help.go`; D3b is `init_cmd.go`-only).
  - D8 `blocked_by = ["4c.6.W2.D3a", "4c.6.W2.D7.5"]` (line 75) — was `D3, D7.5` round-1; now correctly D3a + D7.5.
- **Coherence**: D5/D6/D7 unchanged (still chain on `init_cmd.go` lock through D4). No graph cycle introduced; transitive ordering D3a → D3b → D4 → D5 → D6 → D7 preserved.

Evidence: PLAN.md lines 77-119, _BLOCKERS.toml lines 23-75.

#### W2-FF2 (test-name contract D7.5 ↔ D8): **RESOLVED**

D7.5 acceptance now carries an explicit **TEST-NAME CONTRACT** bullet (PLAN.md line 220):

> `**TEST-NAME CONTRACT (W2-FF2 ROUND-2):** the test names \`TestRunInstall_CreatesDebugConfig\` and \`TestRunInstall_UpdatesExistingConfig\` are a HARD CONTRACT between D7.5 and D8 — D8's pre-flight check (this droplet's twin in the chain) hard-codes these exact names when verifying D7.5 has shipped equivalent coverage before deleting the originals. **If you rename either test in D7.5, you MUST update D8's pre-flight bullet at the same time.** Renaming silently breaks D8's "coverage does NOT regress" gate.`

D8 cross-references the contract in its pre-flight bullet (PLAN.md line 249):

> `**Pre-flight check before deletion (W2-FF2 ROUND-2 contract):** D8 builder MUST run \`mage test-pkg ./cmd/till\` against the current state and confirm \`TestRunInstall_CreatesDebugConfig\` + \`TestRunInstall_UpdatesExistingConfig\` are present and passing. **These exact names are a hard contract pinned in D7.5's acceptance** — if they are NOT present (i.e. D7.5 didn't ship those tests under those exact names), STOP and escalate to orchestrator. The names are not internal builder choice; renaming requires updating BOTH D7.5 AND this pre-flight bullet at the same time. Cross-reference: D7.5 acceptance "TEST-NAME CONTRACT (W2-FF2 ROUND-2)".`

_BLOCKERS.toml line 73 also captures the cross-reference: `D8 pre-flight hard-codes TestRunInstall_* test names per W2-FF2 round-2`.

Evidence: PLAN.md lines 220, 249; _BLOCKERS.toml line 73. The contract is bidirectionally pinned and called out as a coordinated-edit invariant.

#### W2-FF3 (CONSUMER-TIE TEST contract — route-exercising not direct call): **RESOLVED**

D7.5 acceptance now carries an explicit **CONSUMER-TIE TEST CONTRACT** bullet (PLAN.md line 221):

> `**CONSUMER-TIE TEST CONTRACT (W2-FF3 ROUND-2):** each new test MUST invoke \`run(context.Background(), []string{"--app", "tillsyn-init", "install"}, &out, io.Discard)\` end-to-end (NOT call \`runInstall(...)\` directly). This exercises the route → \`cobra\` → \`installCmd.RunE\` → \`runInstall\` chain and proves the command is genuinely wired. Calling \`runInstall\` directly would ship a non-wired install command (the cobra registration in \`main.go\` would not be exercised). Verify wiring via \`mage test-func ./cmd/till TestRunInstall_CreatesDebugConfig\`.`

The contract names the EXACT invocation pattern (`run(context.Background(), []string{"--app", "tillsyn-init", "install"}, &out, io.Discard)`) — verified against the existing `TestRunInitDevConfigCreatesDebugConfig` body at `cmd/till/main_test.go:2928` which uses the identical pattern with `"init-dev-config"` as the third argv element. The verbatim port substitutes only the command name, preserving the route-exercising shape. Anti-pattern (direct `runInstall(...)` call) is explicitly prohibited.

Evidence: PLAN.md line 221; cross-checked against `cmd/till/main_test.go:2928, 2988`.

#### W2-FF4 (D2 / configmerge vestigial removal): **RESOLVED**

D2 is removed from PLAN.md and commented-out in _BLOCKERS.toml. Empirical confirmation that `configmerge` is genuinely vestigial:

- **No production references**: `git grep -ln "configmerge"` returns ONLY workflow MD/TOML/SKETCH artifacts (`workflow/drop_4c_6/DROP_4c.6.W2_TILL_INIT/PLAN.md`, `PLAN_QA_FALSIFICATION.md`, `PLAN_QA_PROOF.md`, `_BLOCKERS.toml`, `workflow/drop_4c_6/PLAN.md`, `RESEARCH/SKETCH_QA_FALSIFICATION.md`, `RESEARCH/TA_AND_KARPATHY_REVIEW.md`, `RESEARCH/TA_VERSIONING_AND_REUSE.md`, `SKETCH.md`). ZERO Go-source references. ZERO `internal/config/configmerge/` directory (verified via `ls /Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/config/` — only `agents.go`, `agents_test.go`, `config.go`, `config_test.go`, `frontmatter.go`, `frontmatter_test.go`, `testdata/`).
- **D5's actual use cases hand-rollable**: PLAN.md line 148 (D5 acceptance ensureGitignore bullet) specifies the implementation: `hand-written \`os.ReadFile\` + \`bytes.Contains([]byte("\\nagents.local.toml\\n"))\` + \`os.WriteFile\` atomically via \`fsatomic\`. Handle the trailing-newline corner case (file ends with no \`\\n\` → append \`\\n\` + \`agents.local.toml\\n\`; file ends with \`\\n\` → append \`agents.local.toml\\n\`). NO \`configmerge\` dependency`. All four D5 use cases (`copyAgentFiles` FLAT-copy, `copyAgentsTOML` byte-copy, `ensureGitignore` line-append, project-DB record creation in D7) are file-COPY operations or single-line idempotent appends — none requires section-merging TOML semantics.
- **Audit-trail discipline clean**: every PLAN.md `configmerge` mention (9 lines: 17, 67, 69, 72, 75, 148, 157, 257, 259, 263) is inside an explicit W2-FF4 audit block (REMOVED-droplet header, ROUND-2-update banners, "NO configmerge dependency" explicit negation in D5 acceptance, "D2 removed" in blockers/notes). NONE appears in active acceptance criteria, RiskNotes, or KindPayload as a live dep.
- **`agents.example.toml` is COPIED (not merged)**: D5 acceptance line 147: `\`copyAgentsTOML(destDir)\` copies embedded \`internal/templates/builtin/agents.example.toml\` → \`<destDir>/agents.toml\` atomically.` This is a byte-for-byte copy, not a merge — confirming the W0/W1 spec.
- **`.gitignore` is single-line append**: D5 acceptance line 148 (above) names the exact algorithm, no merge needed.

Evidence: `git grep -ln "configmerge"` + `ls internal/config/` + PLAN.md lines 17, 67-75, 147-148, 257, 259, 263; _BLOCKERS.toml lines 13-21 (D2 commented-out audit block) and line 49 (D5 reason: `W2-FF4 round-2 removed configmerge dep so D2 dropped`).

**Comment-out vs full-removal precedent**: the W2 plan chose to comment-out the D2 entry in _BLOCKERS.toml (lines 13-21) rather than fully delete it. This aligns with the project memory rule "Never Remove Workflow Drop Files" (`feedback_never_remove_workflow_files.md`) — the file remains tracked, the audit trail is preserved, and the surrounding entries (D1/D3a/D3b/D4/D5/D6/D7/D7.5/D8) keep their semantics. The comment block names W2-FF4 explicitly + lists the two reasons (`copyAgentsTOML` is a copy, `ensureGitignore` is a single-line append). Acceptable pattern; no action needed.

#### W2-FF5 (LASLIG TITLE byte-for-byte preservation): **RESOLVED**

D7.5 acceptance now carries an explicit **LASLIG TITLE CONTRACT** bullet (PLAN.md line 222):

> `**LASLIG TITLE CONTRACT (W2-FF5 ROUND-2):** \`runInstall\`'s \`writeCLIKV\` first arg (the table title) is \`"Dev Config"\` — preserved BYTE-FOR-BYTE from the existing \`runInitDevConfig\` body at \`cmd/till/main.go:2089\`. The ported test bodies at \`main_test.go:2936\` and \`main_test.go:2991\` assert \`"Dev Config"\` substring in the output; preserving the title keeps the verbatim port mechanical and decision-free. **DO NOT rename the title to \`"Install"\` or any other string.** If a future drop wants the title to say \`"Install"\` instead, that is a separate user-visible-label-rename droplet — out of scope for D7.5.`

**Empirical verification of the cited line numbers**:

- `cmd/till/main.go:2089` reads `return writeCLIKV(stdout, "Dev Config", [][2]string{` — the title `"Dev Config"` is byte-exact at the line cited. The full block (lines 2089-2093) is the canonical Laslig table to port.
- `cmd/till/main_test.go:2936` reads `for _, want := range []string{"Dev Config", "status", "created dev config", shellEscapePath(paths.ConfigPath), "logging level", "debug"} {` — `"Dev Config"` is the FIRST substring asserted. Test will FAIL if the port renames the title.
- `cmd/till/main_test.go:2991` reads `for _, want := range []string{"Dev Config", "status", "dev config already exists", shellEscapePath(paths.ConfigPath), "logging level", "debug"} {` — same first-position substring. Same failure mode.

Both tests will continue to assert `"Dev Config"` after the verbatim port (per W2-FF2 contract: same body, just `[]string{"install"}` instead of `[]string{"init-dev-config"}` in the `run(...)` argv). Title preservation makes the port mechanical and decision-free, eliminating the round-1 1.5/2.1 ambiguity.

Evidence: PLAN.md line 222; `cmd/till/main.go:2089`; `cmd/till/main_test.go:2936, 2991`.

### Round-2 attack passes

#### Attack: D3b sizing — does the JSON-parser-only droplet still fit one atomic droplet?

D3b (PLAN.md lines 101-119) scope:
- JSON struct definition (~3 LOC).
- `runInitJSON` body — `encoding/json.Unmarshal` + `Group` whitelist validation against `{"till-gen", "till-go"}` (~15-20 LOC including error wrapping).
- Replace D3a's JSON-stub error in `RunE` (~5 LOC change).
- Table-driven test in `init_cmd_test.go` — 4 cases (valid, invalid-group, malformed-JSON, missing-required-fields) + at least one route-exercising case (~60-80 LOC).

Total: ~25-30 LOC production + ~60-80 LOC tests. **Within atomic envelope** (1-4 code blocks; well below the 80-120 LOC ceiling for production code). Single production file (`init_cmd.go`). REFUTED — D3b is not over-decomposed-into-an-atom; it sits comfortably inside the till-go atom.

#### Attack: D7.5 verbatim port — does `runInitDevConfig`'s body fit `runInstall` 1:1?

`cmd/till/main.go:2039-2094` (`runInitDevConfig`) signature: `func runInitDevConfig(stdout io.Writer, opts rootCommandOptions) error`. Body:
1. `paths := platform.DefaultPathsWithOptions(platform.Options{AppName: opts.appName, DevMode: true, HomeDir: opts.homeDir})` (lines 2045-2049) — uses `opts.appName` and `opts.homeDir`.
2. `os.MkdirAll(filepath.Dir(configPath), 0o755)` (line 2055).
3. `os.Stat(configPath) → os.WriteFile(configPath, templateBytes, 0o644)` if not exists (lines 2059-2071).
4. `os.ReadFile(configPath) → ensureLoggingSectionDebug(string(content)) → os.WriteFile(...)` (lines 2074-2083).
5. `writeCLIKV(stdout, "Dev Config", [][2]string{ {"status", msg}, {"config path", shellEscapePath(configPath)}, {"logging level", "debug"} })` (lines 2089-2093).

D7.5's stated port (PLAN.md line 215): `(a) \`platform.DefaultPathsWithOptions(platform.Options{AppName: opts.appName, DevMode: true, HomeDir: opts.homeDir})\` resolution, (b) \`os.MkdirAll\` + create-if-missing of \`<dev-paths>/till.toml\` from \`config.DefaultTemplate()\`, (c) \`ensureLoggingSectionDebug\` rewrite of the logging section to \`debug\`, (d) \`writeCLIKV\` Laslig success message with status / config path / logging level keys.` All five behavioral elements covered.

Cross-references confirmed:
- `shellEscapePath` (`main.go:2096-2108`) — explicit PLAN.md note line 223: `\`TestShellEscapePath\` (\`main_test.go:3105\`) is NOT moved — \`shellEscapePath\` itself stays in \`main.go\` as a shared helper used by \`runInstall\` (which now lives in \`install_cmd.go\` and imports \`shellEscapePath\` from the same package).` Same package → no import gymnastics needed.
- `ensureLoggingSectionDebug` — referenced in D7.5 acceptance (line 215, item c). Stays in `main.go` per the same shared-helper rule. Verified at `main.go:2078` (called from `runInitDevConfig` at line 2078; will be called from `runInstall` in `install_cmd.go` after D7.5).
- `rootCommandOptions` (the `opts` param type) — same package, no import friction.
- `config.DefaultTemplate()` — already imported in `main.go`; same package + import set in `install_cmd.go` per build.

**No subtle integration points missed**: the body is genuinely a verbatim port. The only added surface is the cobra-command builder (`newInstallCommand`) which D7.5 acceptance line 214 explicitly names. REFUTED — the port is mechanical and complete.

#### Attack: D8's pre-flight bullet — does it verify-before-fail or assert-late?

D8 NOTES bullet (PLAN.md line 249): `D8 builder MUST run \`mage test-pkg ./cmd/till\` against the current state and confirm \`TestRunInstall_CreatesDebugConfig\` + \`TestRunInstall_UpdatesExistingConfig\` are present and passing.`

The check fires BEFORE D8 makes any deletion — `mage test-pkg ./cmd/till` runs the existing test suite (which includes both old `TestRunInitDevConfig*` and new `TestRunInstall_*` if D7.5 has shipped). If `TestRunInstall_*` are missing, the suite still passes (they're net-new tests, absence doesn't fail anything), so the test-suite run itself is not the gate. **The `D8 builder MUST run ... and confirm ... are present and passing` is a builder-discipline assertion that requires the builder to grep / list test names BEFORE deleting.** The assertion is "preflight verify the contract names are in test-name list" — and the escalation path is `STOP and escalate to orchestrator` if names are absent.

This is an acceptable pattern for a builder agent: human-readable instruction + explicit STOP-and-escalate clause + cross-reference to the originating contract (W2-FF2). The D8 _BLOCKERS.toml `blocked_by` chain (`["4c.6.W2.D3a", "4c.6.W2.D7.5"]`) prevents D8 from launching until D7.5 lands at all, which is the structural gate; the pre-flight is the secondary safety net.

REFUTED — pre-flight is timed correctly (before any deletion), the contract is bidirectionally pinned, and the structural blocker_by chain is the primary gate.

#### Attack: Audit-trail completeness on removed-D2

Already covered under W2-FF4 above. All 9 PLAN.md `configmerge` mentions are inside W2-FF4-tagged audit blocks; no live acceptance/RiskNotes/KindPayload reference. _BLOCKERS.toml D2 entry is commented-out with full rationale (lines 13-21). REFUTED.

#### Attack: Comment-out pattern vs full-removal precedent

Project memory rule `feedback_never_remove_workflow_files.md`: "Files that don't apply (e.g. _BLOCKERS.toml on single-child dirs) are never created, not stamped-then-deleted." The rule applies at FILE granularity (don't create the file in the first place); it doesn't prescribe how to handle removed ENTRIES inside an otherwise-load-bearing file. Two viable patterns:

1. **Comment-out** (W2 round-2 choice): D2 entry stays in _BLOCKERS.toml as a comment block with W2-FF4 explanation. Audit trail preserved IN THE TOML FILE; round-3 readers see the round-1 → round-2 transition without consulting PLAN.md.
2. **Full-removal in TOML + audit-trail in PLAN.md only**: Delete D2's `[[blockers]]` block entirely; rely on PLAN.md "Droplet 4c.6.W2.D2 — REMOVED" header (lines 67-75) for audit. TOML stays terse.

Both are defensible. Round-2 chose (1), which is slightly more verbose but slightly more robust against PLAN.md edits that might later drift away from the explicit removal banner. The pattern fits the spirit of the never-remove rule (preserve audit trail) without violating it (entries vs files). REFUTED — pattern is acceptable.

WORKFLOW.md precedent check: no analogous "removed droplet" precedent in tracked _BLOCKERS.toml siblings (drop_1_75, drop_4c_6 root, W0, W0.5, W3) — those files weren't modified mid-round to remove a droplet. The W2 round-2 transition is the first time this pattern appears in the project. The choice is therefore precedent-setting; logging it as a refinement candidate (workflow precedent for "removed droplet entries") might be useful, but it's not a finding against the W2 plan itself.

### Round-2 new findings

#### W2-PF1 [severity: low; bucket: spawn-prompt clarity] — D8 NIT 1.6 (round-1 doc-comment phrasing) STILL UNADDRESSED

Round-1 falsification finding 1.6 flagged: D8 acceptance line 222 (round-1) said "update the `TestShellEscapePath` doc-comment at line 3105 to drop the 'init-dev-config' mention" without specifying the exact replacement string. Round-2 PLAN.md line 250 still reads:

> `The \`TestShellEscapePath\` test at \`main_test.go:3105\` does NOT need to be removed — \`shellEscapePath\` is still in \`main.go\` and still used by \`runInstall\` (in \`install_cmd.go\`). D8 only updates the doc-comment to drop the "init-dev-config" mention.`

The phrasing is unchanged; the builder still has to pick between (a) replace `init-dev-config` with `till install`, (b) drop the command reference entirely, or (c) other. This is a low-severity NIT — the test body is fine either way and the `git grep -n init-dev-config cmd/till/` zero-matches assertion at PLAN.md line 240 forces the builder to scrub the mention. Fix: orchestrator pins the exact replacement in D8's spawn-prompt (e.g., `// TestShellEscapePath verifies till install path output is shell-token safe.`) OR lets the builder pick + log the choice in `BUILDER_WORKLOG.md`. Not blocking; flagged for spawn-prompt completeness.

Evidence: PLAN.md line 250 (round-2 carries round-1 phrasing forward); cross-reference round-1 falsification finding 1.6 in `PLAN_QA_FALSIFICATION.md` line 34.

### Round-2 verdict summary

| Round-1 finding | Severity | Resolution |
| --- | --- | --- |
| W2-FF1 (D3 atomic-decomposition split) | medium | **RESOLVED** — D3a/D3b split + blocker chain rewired (D4→D3b, D7.5→D3a, D8→D3a+D7.5). |
| W2-FF2 (test-name contract D7.5 ↔ D8) | medium | **RESOLVED** — D7.5 pins names; D8 pre-flight cross-references; _BLOCKERS.toml line 73 captures. |
| W2-FF3 (CONSUMER-TIE TEST contract) | medium | **RESOLVED** — D7.5 line 221 pins exact `run(context.Background(), []string{"--app", "tillsyn-init", "install"}, &out, io.Discard)` invocation. |
| W2-FF4 (D2 / configmerge vestigial) | medium | **RESOLVED** — D2 removed; `git grep` confirms zero production references; audit-trail clean across PLAN.md + _BLOCKERS.toml. |
| W2-FF5 (LASLIG TITLE byte-for-byte) | low | **RESOLVED** — D7.5 line 222 pins `"Dev Config"` byte-for-byte; verified against `cmd/till/main.go:2089` + `main_test.go:2936, 2991`. |

**Round-2 NEW findings:** W2-PF1 (low) — D8 doc-comment phrasing still ambiguous (round-1 NIT 1.6 carried forward); spawn-prompt clarification, not a plan amendment.

**Verdict: PASS.**

All five round-1 findings RESOLVED with explicit evidence; round-2 attack passes (D3b atomic sizing, D7.5 verbatim port soundness, D8 pre-flight timing, audit-trail completeness, comment-out pattern vs precedent) produce no counterexample. One low-severity round-2 NIT (W2-PF1) is a builder-spawn-prompt clarification that orchestrator can pin inline; it does not block plan advancement.

W2 sub-plan is GREEN to advance to W2 builds.

### Round 2 Hylla Feedback

`N/A — sub-drop touched non-Go files (PLAN.md, _BLOCKERS.toml, SKETCH.md) plus committed Go (cmd/till/main.go, cmd/till/help.go, cmd/till/main_test.go) verified via Read at exact line spans cited in the L2 plan + git grep for symbol-presence checks (configmerge, init_cmd.go, install_cmd.go).` Per project rule (`CLAUDE.md` § Code Understanding Rules): Hylla covers committed Go but the round-2 review's load-bearing checks were exact-line-span verifications (`main.go:2089`, `main_test.go:2936, 2991`) where Read is the highest-evidence-yield tool, AND directory/file-existence checks (`internal/vendor/`, `internal/config/configmerge/`, `cmd/till/init_cmd.go`, `cmd/till/install_cmd.go`) which are filesystem-shape questions Hylla doesn't answer. No Hylla query issued; no fallback miss to log.
