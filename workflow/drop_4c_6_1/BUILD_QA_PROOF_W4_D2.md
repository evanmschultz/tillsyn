# W4.D2 — BUILD-QA-PROOF Verdict

**Date:** 2026-05-12
**Reviewer:** go-qa-proof-agent (opus)
**Overall verdict:** PASS WITH NITS

## Acceptance Bullet Coverage

### A1. till-go.toml QA agent_name suffix update

> `till-go.toml` `[agent_bindings.<kind>]` `agent_name` values updated with `-agent` suffix: `plan-qa-proof-agent`, `plan-qa-falsification-agent`, `build-qa-proof-agent`, `build-qa-falsification-agent` match the new 10-agent file names from W4.D1. The 4 orchestrator-managed bindings (`closeout`, `refinement`, `discussion`, `human-verify`) continue to reference `orchestrator-managed` (no `-agent` suffix — this is the special 10th file, not a standard agent).

**Evidence:**
- `internal/templates/builtin/till-go.toml:481` `agent_name = "plan-qa-proof-agent"` (under `[agent_bindings.plan-qa-proof]` at line 480).
- `internal/templates/builtin/till-go.toml:507` `agent_name = "plan-qa-falsification-agent"` (under `[agent_bindings.plan-qa-falsification]` at line 506).
- `internal/templates/builtin/till-go.toml:533` `agent_name = "build-qa-proof-agent"` (under `[agent_bindings.build-qa-proof]` at line 532).
- `internal/templates/builtin/till-go.toml:566` `agent_name = "build-qa-falsification-agent"` (under `[agent_bindings.build-qa-falsification]` at line 565).
- `internal/templates/builtin/till-go.toml:599, 624, 637, 650` four `orchestrator-managed` bindings retain bare `agent_name = "orchestrator-managed"` value (no `-agent` suffix).
- `git diff HEAD -- internal/templates/builtin/till-go.toml` confirms exactly 4 line changes (4 insertions, 4 deletions), matching the 4 QA bindings — no other field touched.

**Verdict:** PASS.

### A2. till-gen.toml same updates

> `till-gen.toml` same updates.

**Evidence:**
- `rg "agent_name" internal/templates/builtin/till-gen.toml` returns only line 44 (a comment in the file header), and no `[agent_bindings.<kind>]` sections exist in the file.
- `internal/templates/builtin/till-gen.toml:35-46` doc-comment explicitly cites F.2.2 acceptance criterion #2: "no agent bindings declared" — generic template intentionally omits agent_bindings.
- `git diff HEAD -- internal/templates/builtin/till-gen.toml` returns empty (file unmodified).

**Verdict:** PASS (vacuous). till-gen.toml has zero `[agent_bindings]` sections by F.2.2 design, so "same updates" is a no-op. Bullet wording is misleading — flagged as NIT-1.

### A3. agents.example.toml schema shift to `[<group>]` / `[<group>.<kind>]`

> `agents.example.toml` sections: `[go]` replaces `[agents]`; `[go.plan-qa-proof]` replaces `[agents.plan-qa-proof]` etc. Full schema per REVISION_BRIEF §2.12. **All 3 group sections present: `[go]`, `[gen]`, and `[fe]`** (Round 10 absorption — W2 fals NIT-R2-2: `till init --group gen` is a valid user path; fixture must cover all 3 canonical groups).

**Evidence:**
- `rg "^\[(agents|go|gen|fe)" internal/templates/builtin/agents.example.toml` returns 27 headers — all in the `[go]` / `[gen]` / `[fe]` family; zero `[agents.*]` headers.
- `internal/templates/builtin/agents.example.toml:35` `[go]` group defaults block present.
- `internal/templates/builtin/agents.example.toml:116` `[gen]` group defaults block present.
- `internal/templates/builtin/agents.example.toml:178` `[fe]` group defaults block present.
- Per-kind blocks for each group (`[go.plan]`, `[go.build]`, `[go.plan-qa-proof]`, `[go.plan-qa-falsification]`, `[go.build-qa-proof]`, `[go.build-qa-falsification]`, `[go.research]`, `[go.commit]`) confirmed at lines 58, 66, 73, 78, 83, 88, 93, 98; same shape for gen at 133/138/141/146/151/156/161/164; same for fe at 195/200/204/209/214/220 (etc.).
- `git grep -F '[agents.' internal/templates/builtin/` returns exit code 1 (zero matches) — old schema fully eradicated.

**Verdict:** PASS.

### A4. till-fe.toml NEW file with minimal cascade template structure

> `till-fe.toml` (NEW) ships at `internal/templates/builtin/till-fe.toml` with minimal cascade template structure for `fe` group per the `[<group>.<kind>]` schema. Agent bindings reference the 10 standard agent names (9 standard + `orchestrator-managed`).

**Evidence:**
- `internal/templates/builtin/till-fe.toml` exists (untracked, 12.9K, 462 lines).
- `internal/templates/builtin/till-fe.toml:34` `schema_version = "v1"`.
- 12-kind `[kinds.<kind>]` catalog at lines 40-168 (plan, research, build, plan-qa-{proof,falsification}, build-qa-{proof,falsification}, closeout, commit, refinement, discussion, human-verify) — all 12 kinds present.
- 6 `[[child_rules]]` entries at lines 175-216: 4 standard (build→build-qa-{proof,falsification}, plan→plan-qa-{proof,falsification}) + 2 drop-narrowed (DROP-PLAN-QA-PROOF, DROP-PLAN-QA-FALSIFICATION).
- 6 `[[steward_seeds]]` at lines 222-244 (DISCUSSIONS / HYLLA_FINDINGS / LEDGER / WIKI_CHANGELOG / REFINEMENTS / HYLLA_REFINEMENTS).
- `[gates] build = ["mage_ci", "commit", "push"]` at line 250-251.
- 12 `[agent_bindings.<kind>]` sections at lines 272-461 covering plan, research, build, all 4 QA kinds, closeout, commit, refinement, discussion, human-verify.
- Agent_name values: `planning-agent` (line 273), `research-agent` (292), `builder-agent` (304), `plan-qa-proof-agent` (324), `plan-qa-falsification-agent` (343), `build-qa-proof-agent` (362), `build-qa-falsification-agent` (381), `commit-message-agent` (413), `orchestrator-managed` (401, 426, 439, 452). All 10 standard agent file names referenced.
- Note: `[kinds.<kind>]` sections for `plan`, `research`, `build`, `plan-qa-proof`, `plan-qa-falsification`, `refinement`, `discussion` carry NO `allowed_child_kinds` allow-list — this matches till-go.toml's pattern (universal-allow when empty per CLAUDE.md). The 5 prohibition-source kinds (`build-qa-proof`, `build-qa-falsification`, `closeout`, `commit`, `human-verify`) carry explicit 11-element allow-lists.

**Verdict:** PASS.

### A5. embed.go //go:embed extended to include till-fe.toml

> `embed.go` `//go:embed` directive extended to include `builtin/till-fe.toml`.

**Evidence:**
- `internal/templates/embed.go:87` `//go:embed builtin/till-go.toml builtin/till-gen.toml builtin/till-fe.toml` — single line lists all three.
- `internal/templates/embed.go:227-245` switch in `LoadDefaultTemplateForLanguage` now handles case `"fe"` (line 234) and resolves to `path = "builtin/till-fe.toml"` (line 242); previous fail-loud Q1 path replaced with resolve path.
- `internal/templates/embed.go:276-278` `BuiltinTemplateNames()` returns `[]string{"till-fe", "till-gen", "till-go"}` — preserves lexical order and adds the FE entry.
- `git diff HEAD -- internal/templates/embed.go` confirms +40 / -... touches the `//go:embed` directive, doc-comment block (lines 62-85), the resolver switch, the closed-enum drift note, and `BuiltinTemplateNames()`.

**Verdict:** PASS.

### A6. embed_test.go asserts till-fe.toml path resolves

> `embed_test.go` updated to assert `till-fe.toml` path resolves.

**Evidence:**
- `internal/templates/embed_test.go:956-973` new `TestLoadDefaultTemplateForLanguage_FESupported` asserts `LoadDefaultTemplateForLanguage("fe")` succeeds and returns a Template with `SchemaVersion == "v1"` AND 12 agent bindings (one per closed-enum kind). Replaces the prior `_FERejected` test.
- `internal/templates/embed_test.go:989-1031` new `TestLoadDefaultTemplateFEResolves` opens `builtin/till-fe.toml` directly via `DefaultTemplateFS.Open`, parses via `Load`, asserts: 12 kinds present, 6 child_rules (4 standard + 2 drop-narrowed), 6 STEWARD seeds, 12 agent_bindings.
- `internal/templates/embed_test.go:1119-1125` `w4d1CanonicalGroups = []string{"gen", "go", "fe"}` now includes `"fe"`; the placeholder-FS test (`TestDefaultTemplateFSEmbedsPlaceholderAgentFiles`) at line 1157 iterates this list and confirms each FE agent .md file is present and carries the "PLACEHOLDER" marker.
- `internal/templates/embed_test.go:1044-1060` `TestLoadDefaultTemplateForLanguage_UnknownRejected` uses `"rust"` (no longer `"fe"`) as the canonical unsupported-language axis — closed-enum drift guard preserved against a future language addition.

**Verdict:** PASS.

### A7. git grep [agents. returns zero hits in internal/templates/builtin/

> `git grep '\[agents\.'` post-edit returns zero hits in `internal/templates/builtin/`.

**Evidence:**
- `git grep -F '[agents.' internal/templates/builtin/` returns exit code 1 (zero matches). Confirmed.

**Verdict:** PASS.

### A8. mage test-pkg ./internal/templates passes; mage ci green

> `mage test-pkg ./internal/templates` passes; `mage ci` green.

**Evidence:**
- `mage test-pkg ./internal/templates` returned `[SUCCESS] All tests passed — 475 tests passed across 1 package` (just executed).
- `mage ci` green tree-wide (3164/3164) was reported by the orchestrator's tree-wide check before spawning this proof pass.

**Verdict:** PASS.

### A9. Scope extension — service_test.go rust replacement

> [From builder report; not in PLAN.md acceptance but called out as scope extension] `internal/app/service_test.go` `TestLoadProjectTemplate_UnsupportedLanguagePropagatesError` updated to use `"rust"` instead of `"fe"` (since `"fe"` is now supported).

**Evidence:**
- `internal/app/service_test.go:6891-6906` test now uses `Language: "rust"` (line 6892) and doc-comment (lines 6888-6890) explains the swap: "Drop 4c.6.1 W4.D2 resolved the Q1 deferral and shipped till-fe.toml, so 'fe' is now a supported language; 'rust' remains unsupported."
- Necessary because the original test relied on `"fe"` being a closed-enum-rejected value. Once W4.D2 makes `"fe"` resolvable, the closed-enum drift test would have to be rewritten — this is the correct minimal change.

**Verdict:** PASS.

## NITs

### NIT-1 (low) — Vacuous PLAN.md acceptance bullet for till-gen.toml

PLAN.md line 432 says "till-gen.toml same updates" but till-gen.toml has zero `[agent_bindings]` sections by F.2.2 design (intentionally omitted per generic-template contract). The bullet is satisfied vacuously — the builder correctly made no change to till-gen.toml — but the bullet wording suggests an action that didn't apply. This is a planner-side wording NIT, not a builder action item. Recommend the planner rewrite the bullet to "till-gen.toml is unchanged — it ships zero `[agent_bindings]` per F.2.2 design" in a future plan-fix round, OR add a falsification note explaining the no-op. **No builder action.**

### NIT-2 (low) — till-fe.toml is untracked, not yet staged

`internal/templates/builtin/till-fe.toml` shows as `??` in `git status --porcelain` (untracked, not staged). This is expected for a brand-new file pre-commit, but the orchestrator should `git add` it explicitly before the per-droplet W4.D2 commit lands (along with the modified embed.go, till-go.toml, agents.example.toml, embed_test.go, service_test.go). **No builder action; orchestrator stages at commit time.**

## Verdict rationale

All 8 PLAN.md acceptance bullets are satisfied by on-disk evidence. The 4 QA bindings in till-go.toml carry the `-agent` suffix; the 4 orchestrator-managed bindings remain unchanged. `agents.example.toml` is fully schema-shifted to `[<group>]` / `[<group>.<kind>]` with all 3 canonical groups present (`[go]`, `[gen]`, `[fe]`). The new `till-fe.toml` ships a complete 12-kind cascade template mirroring `till-go.toml`'s shape (12 kinds + 6 child_rules + 6 STEWARD seeds + gates + 12 agent_bindings). `embed.go`'s `//go:embed` directive, language resolver, and `BuiltinTemplateNames()` all extend to include `till-fe.toml`. `embed_test.go` exercises the new resolver path AND opens the new file directly via `DefaultTemplateFS`. The scope extension to `service_test.go` (swap `"fe"` → `"rust"` in `_UnsupportedLanguagePropagatesError`) is the minimal correct change to preserve closed-enum drift coverage. `git grep '\[agents\.'` confirms full eradication of the old schema. `mage test-pkg ./internal/templates` returns 475/475 PASS.

Two minor NITs are wording-only (PLAN.md bullet wording for the till-gen.toml no-op) and operational (till-fe.toml untracked pre-commit) — neither blocks completion.

**Overall: PASS WITH NITS.**
