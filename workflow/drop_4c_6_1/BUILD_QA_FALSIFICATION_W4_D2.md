# W4.D2 — BUILD-QA-FALSIFICATION Verdict

**Date:** 2026-05-12
**Reviewer:** go-qa-falsification-agent (opus)
**Overall verdict:** PASS WITH NITS

## Attack Hypotheses Tested

### H1 — till-gen.toml missing same agent_name updates (PLAN.md acceptance: "till-gen.toml same updates")
- **Hypothesis**: PLAN.md line 432 says till-gen.toml needs the same agent_name updates as till-go.toml. Builder did NOT modify till-gen.toml. Counterexample candidate.
- **Test**: Read till-gen.toml lines 343-353 + the file header at lines 22-46. Verified the file INTENTIONALLY OMITS `[agent_bindings]` entirely per F.2.2 acceptance criterion #2; this is documented in the file header and pinned by a regression test (`TestLoadDefaultGenericTemplate` asserts `len(tpl.AgentBindings) == 0`).
- **Finding**: there are NO `agent_name` fields in till-gen.toml to update. The PLAN.md "same updates" requirement is vacuously satisfied.
- **Verdict**: REFUTED. (NIT: PLAN.md author likely under-specified; the acceptance bullet should have said "till-gen.toml needs no changes because it omits agent_bindings by design.")

### H2 — Agent_name values reference non-existent files (contract mismatch)
- **Hypothesis**: Updated agent_name values may not match actual filenames in `internal/templates/builtin/agents/go/`.
- **Test**: Listed `agents/go/` and `agents/fe/` directories; verified all 10 expected files exist in each (plan-qa-proof-agent.md, plan-qa-falsification-agent.md, build-qa-proof-agent.md, build-qa-falsification-agent.md, builder-agent.md, planning-agent.md, research-agent.md, closeout-agent.md, commit-message-agent.md, orchestrator-managed.md). Cross-checked against `git grep -n '^agent_name' internal/templates/builtin/till-go.toml` and `till-fe.toml`.
- **Finding**: every agent_name value in till-go.toml + till-fe.toml resolves to an existing .md file in the corresponding agents/<group>/ subdir. The 4 orchestrator-managed bindings correctly reference `orchestrator-managed` (no `-agent` suffix), matching the special 10th filename.
- **Verdict**: REFUTED.

### H3 — agents.example.toml [fe] section is [go] copy-paste with no FE-specific content
- **Hypothesis**: Builder shipped 3 groups but [fe] may not have FE-specific differentiators.
- **Test**: Read agents.example.toml lines 170-232.
- **Finding**: [fe] has FE-distinguishing content: `[fe.build-qa-proof]` and `[fe.build-qa-falsification]` add `mcp__plugin_playwright_playwright__*` to tools_allow (visual QA for Wails projects); the [go] + [gen] sections do NOT add Playwright. [fe.build] does NOT add anything Go-specific (no `mage` cli_args). The defaults are mostly shared (model = sonnet, etc.), but the Playwright addition is the legitimate FE-specific cut.
- **Verdict**: REFUTED.

### H4 — embed_test.go FEResolves test is shallow (doesn't actually verify FE template structure)
- **Hypothesis**: New `TestLoadDefaultTemplateFEResolves` may check existence only, missing schema verification.
- **Test**: Read embed_test.go FEResolves test body in diff. Asserts: (a) DefaultTemplateFS.Open(till-fe.toml) succeeds, (b) Load() parses cleanly, (c) SchemaVersion == v1, (d) `len(tpl.Kinds) == 12` and every closed-12-kind has a section, (e) `len(tpl.ChildRules) == 6` (4 standard + 2 drop-narrowed), (f) `len(tpl.StewardSeeds) == 6` (DISCUSSIONS / HYLLA_FINDINGS / LEDGER / WIKI_CHANGELOG / REFINEMENTS / HYLLA_REFINEMENTS), (g) `len(tpl.AgentBindings) == 12`.
- **Finding**: test is thorough — opens via embed.FS, parses through full Load() chain, asserts 4 distinct invariants. Not a stub.
- **Verdict**: REFUTED.

### H5 — Cross-group agent_bindings consistency: till-fe.toml may have wrong agent_name values
- **Hypothesis**: till-fe.toml's 12 agent_name values may drift from till-go.toml's pattern.
- **Test**: `git grep --no-index -n '^agent_name' internal/templates/builtin/till-fe.toml` — verified all 12 values: planning-agent, research-agent, builder-agent, plan-qa-proof-agent, plan-qa-falsification-agent, build-qa-proof-agent, build-qa-falsification-agent, orchestrator-managed (4x for closeout/refinement/discussion/human-verify), commit-message-agent. Matches till-go.toml's pattern 1:1.
- **Verdict**: REFUTED.

### H6 — parent_git_diff cross-binding discipline broken (independence-of-QA violated)
- **Hypothesis**: till-fe.toml may incorrectly add `parent_git_diff = true` to QA bindings (would violate REV-4's independence invariant).
- **Test**: Read till-fe.toml context blocks: `agent_bindings.build.context` (line 315) has `parent_git_diff = true`; `agent_bindings.build-qa-proof.context` (line 373) and `agent_bindings.build-qa-falsification.context` (line 392) do NOT have `parent_git_diff`. Mirrors till-go's REV-4 discipline exactly.
- **Verdict**: REFUTED.

### H7 — Old [agents.kind] schema leaked into builtin/
- **Hypothesis**: Schema shift may have missed a section.
- **Test**: `git grep '\[agents\.' internal/templates/builtin/` returns exit code 1 (zero hits). PLAN.md acceptance criterion met.
- **Verdict**: REFUTED.

### H8 — service_test.go change scope creep beyond 1-line fe→rust
- **Hypothesis**: Builder claimed "1-line scope extension" but service_test.go has 130 insertions.
- **Test**: Read diff. The bulk addition (~125 lines, `writeHomeTemplateFixture` helper + `TestLoadProjectTemplate_HomeTier`) is clearly W1.D1 builder's work (HOME-tier extension is W1.D1's scope per PLAN.md line 213-236). The 1-line `fe → rust` change in `TestLoadProjectTemplate_UnsupportedLanguagePropagatesError` plus a 4-line comment update is THIS droplet's contribution.
- **Finding**: not a W4.D2 scope creep — the bulk addition belongs to a parallel droplet (W1.D1) and coexists in the uncommitted Wave B tree per the orchestrator brief ("Tree state: Wave B uncommitted"). W4.D2's actual edit is the 1-line + comment as claimed.
- **Verdict**: REFUTED.

### H9 — BuiltinTemplateNames change broke external consumers
- **Hypothesis**: Production `BuiltinTemplateNames()` now returns 3 entries; callers elsewhere may break.
- **Test**: `git grep -l 'BuiltinTemplateNames'` returned `internal/app/template_service.go:118` (uses production function — auto-updates) and `internal/adapters/mcp_rpc/extended_tools_test.go` (test stub at line 879-887 hardcodes `["till-gen", "till-go"]`).
- **Finding**: production caller (`template_service.go`) calls the function and forwards the result — automatically gets the new 3-entry list. **Test stub at `extended_tools_test.go:884-885` and the matching `TestTillTemplate_ListBuiltin` assertion at line 3818 still hardcode the OLD 2-entry list** `["till-gen", "till-go"]`. The stub's doc comment (line 874) claims it "mirrors templates.BuiltinTemplateNames so tests assert against the same wire vocabulary the production resolver exposes" — that invariant is now violated. mage ci still passes 3164/3164 because the stub returns 2 names and the test asserts 2 names (internally consistent), but the stub now lies about production behavior.
- **Verdict**: NIT — stale test stub drift. mage ci is GREEN; not a hard fail. `internal/adapters/mcp_rpc/extended_tools_test.go` is NOT in W4.D2's declared paths, so this is cross-droplet stale-reference rather than W4.D2 violation. Should be addressed in a follow-up droplet (W3 or W7 territory) or absorbed inline.

### H10 — Stale doc-comment drift in embed.go
- **Hypothesis**: Builder may have missed updating all doc-comment references to FE's now-shipped status.
- **Test**: Read embed.go lines 128-147 (`ErrLanguageNotSupported` doc) and line 221-226 (`LoadDefaultTemplateForLanguage` "Returns (Template{}, err) on:" list).
- **Finding**: TWO stale doc-comment blocks remain:
  1. **embed.go:130-133** — `ErrLanguageNotSupported` doc says: "currently `"fe"` per the Q1 resolution in workflow/drop_4c_5/THEME_F_PLAN.md §3 Note 5". After W4.D2, `"fe"` is no longer a not-yet-shipped value; it's supported. Doc is stale.
  2. **embed.go:141-146** — "Closed-enum drift guard" says: "when a future drop extends `domain.Project.Language` (e.g. landing FE adopter support)..." Stale — FE adopter support landed in THIS drop.
  3. **embed.go:222** — `LoadDefaultTemplateForLanguage` doc says: "Returns (Template{}, err) on: `lang == "fe"` (deferred per Q1)." Self-contradictory with lines 204-208 which now say `"fe"` loads till-fe.toml successfully. CONFIRMED stale.
- **Verdict**: NIT — doc-comment drift. Code is correct; comments contradict the new behavior. The function-body doc-comment changes lines 195-211 + 232-244 + 271-275 were updated, but the surrounding contractual doc-comment text (ErrLanguageNotSupported sentinel + the Returns list at 221-226) was not. Trivial fix; should be inline-absorbed before commit.

### H11 — till-fe.toml drop-narrowed child_rules choice (vs till-gen omission pattern)
- **Hypothesis**: till-fe.toml includes 6 child_rules (4 standard + 2 drop-narrowed DROP-PLAN-QA-*); till-gen omits drop-narrowed entries by design. The choice for till-fe should be justified.
- **Test**: Read till-fe.toml lines 30-32 + 202-216. The comment at line 31-32 says "Drop-narrowed child_rules (DROP-PLAN-QA-PROOF, DROP-PLAN-QA-FALSIFICATION) included to match till-go.toml's drop-level cascade shape."
- **Finding**: defensible design choice — till-fe is a CASCADE-using template (mirrors till-go's shape), unlike till-gen which is intentionally generic and doesn't assume cascade adoption. PLAN.md acceptance doesn't enforce a specific count; "minimal cascade template structure" leaves room. The choice is justified in the file header.
- **Verdict**: REFUTED.

### H12 — till-gen.toml header comment references old "qa-proof-agent" name (stale doc)
- **Hypothesis**: till-gen.toml's header doc comment at lines 32 and 44 still uses old `qa-proof-agent` / `go-builder-agent` naming.
- **Test**: `git grep --no-index -n 'qa-proof-agent' internal/templates/builtin/till-gen.toml` returned 2 hits, both in doc-comment paragraphs framed as "the bare-name convention" and "historical example."
- **Finding**: the references are clearly framed as historical / convention examples, not active bindings. Minor staleness but not contractually wrong. NIT level.
- **Verdict**: NIT (low severity).

## Unmitigated Counterexamples (if any)
None found.

## NITs (if any)

- **NIT 1 — `internal/templates/embed.go` doc-comment drift (3 stale references)**. **Severity**: low. **Locations**:
  - Lines 130-133 (`ErrLanguageNotSupported` doc): "currently `"fe"` per the Q1 resolution" — `"fe"` is no longer not-yet-shipped.
  - Lines 141-146 ("Closed-enum drift guard"): "when a future drop extends `domain.Project.Language` (e.g. landing FE adopter support)" — FE landed in THIS drop.
  - Line 222 (`LoadDefaultTemplateForLanguage` Returns list): "`lang == "fe"` (deferred per Q1)" — self-contradicts lines 204-208 which now say `"fe"` loads till-fe.toml successfully.
  - **Recommended action**: update all three doc-comment paragraphs to reflect `"fe"` as a now-shipped language. Rename the closed-enum-drift-guard example from "FE adopter support" to a hypothetical future language (e.g. `"rust"`). Trivial fix.

- **NIT 2 — `internal/adapters/mcp_rpc/extended_tools_test.go` test stub drift**. **Severity**: low (mage ci GREEN). **Locations**:
  - Line 874 (doc): stub claims to "mirror templates.BuiltinTemplateNames" — invariant now violated.
  - Lines 884-885: hardcoded `["till-gen", "till-go"]` should now be `["till-fe", "till-gen", "till-go"]`.
  - Line 3775 (test doc): "the closed list `["till-gen", "till-go"]`" should now be 3 entries.
  - Lines 3810-3811: `len(templatesRaw) != 2` should now be `!= 3`.
  - Lines 3818: `want := []string{"till-gen", "till-go"}` should now be `[]string{"till-fe", "till-gen", "till-go"}`.
  - **Recommended action**: update the stub + assertion to the 3-entry list. NOT in W4.D2's declared paths (`internal/adapters/mcp_rpc/` is not listed); could legitimately defer to a follow-up droplet, but inline-absorption is cleaner before commit so the test wire-vocabulary doesn't lie. NIT-class per the "NITs are first-class fixes" rule.

- **NIT 3 — `internal/templates/builtin/till-gen.toml` historical-example references stale agent names**. **Severity**: very low. **Location**: lines 32 + 44 reference `qa-proof-agent` / `go-builder-agent` in doc-comment paragraphs framed as "the bare-name convention" / "historical example." Now that the 2-into-4 QA split landed in W4.D1, the historical example is technically stale.
  - **Recommended action**: update the example to reflect the 4-file QA split (`plan-qa-proof-agent` / `plan-qa-falsification-agent` / `build-qa-proof-agent` / `build-qa-falsification-agent`) OR leave as historical example if framed clearly. Trivial; can defer.

## Verdict rationale

W4.D2 ships exactly what the PLAN.md acceptance specifies:

1. till-go.toml: 4 QA agent_name values updated with `-agent` suffix (matches W4.D1's 10-agent file names); 4 orchestrator-managed bindings continue to reference `orchestrator-managed` bare (correct per PLAN.md ContextBlocks warning).
2. till-gen.toml: vacuously satisfies "same updates" because it deliberately omits `[agent_bindings]` per F.2.2 — no agent_name fields exist to update.
3. agents.example.toml: shifted to `[<group>]` + `[<group>.<kind>]` schema; ships all 3 canonical groups (`[go]`, `[gen]`, `[fe]`) per Round 10 absorption.
4. till-fe.toml (NEW): 12 kinds + 6 child_rules (4 standard + 2 drop-narrowed) + 6 STEWARD seeds + `build` gates + 12 agent_bindings (mirrors till-go's REV-4 discipline on parent_git_diff).
5. embed.go: `//go:embed` extended to include till-fe.toml; `LoadDefaultTemplateForLanguage` switch wires `"fe"` → till-fe.toml; `BuiltinTemplateNames` returns 3 entries in stable lexical order.
6. embed_test.go: `FERejected` → `FESupported` (4 invariants asserted); new `FEResolves` canary asserts schema version + 12 kinds + 6 child_rules + 6 STEWARD seeds + 12 bindings.
7. service_test.go: 1-line `fe → rust` extension to `TestLoadProjectTemplate_UnsupportedLanguagePropagatesError` (the 125-line addition is W1.D1's HOME-tier work, NOT W4.D2's).

`git grep '\[agents\.' internal/templates/builtin/` returns zero hits (PLAN.md acceptance met). `mage ci` runs 3164/3164 tests GREEN; minimum coverage threshold (70%) met across all packages. The two new tests (`FESupported`, `FEResolves`) both PASS.

Three NITs flagged: stale doc comments in embed.go (3 paragraphs), stale test-stub drift in extended_tools_test.go (5 locations across two functions), and a minor historical-example staleness in till-gen.toml doc comment. None of these block the droplet — code is correct; doc is drifted in 3 spots; one test stub returns the wrong wire-vocabulary count but the test still passes because it asserts against the stub's own constant.

**Overall verdict: PASS WITH NITS.** All acceptance bullets satisfied; mage ci GREEN; no unmitigated counterexamples. NITs should be inline-absorbed before commit per the "NITs are first-class fixes" rule, OR explicitly deferred to a follow-up droplet with a tracking refinement entry.
