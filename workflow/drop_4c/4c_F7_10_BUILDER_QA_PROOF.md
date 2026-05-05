# Drop 4c F.7.10 — Builder QA Proof

**Droplet:** F.7.10 — Drop hardcoded `hylla_artifact_ref:` line from `spawn.go` prompt body.
**Reviewer:** `go-qa-proof-agent` (read-only).
**Date:** 2026-05-04.
**Verdict:** **PROOF GREEN-WITH-NITS** — all seven acceptance criteria evidenced; two non-blocking observations attached.

---

## Per-Criterion Evidence

### C1 — `assemblePrompt` no longer emits `hylla_artifact_ref:` line

**PASS.** The pre-edit block at `spawn.go` (pre-image) lines 211-213 emitted:

```go
b.WriteString("hylla_artifact_ref: ")
b.WriteString(project.HyllaArtifactRef)
b.WriteString("\n")
```

Post-edit `spawn.go:207-232` — the function body inlined below — contains no `hylla_artifact_ref` literal in any `b.WriteString` call. The only structural `b.WriteString` calls now are: `task_id:` (`spawn.go:209`), `project_id:` (`spawn.go:212`), `project_dir:` (`spawn.go:215`), `kind:` (`spawn.go:218`), optional `title:` (`spawn.go:222`), and `move-state directive:` (`spawn.go:228`). The diff at `git diff HEAD -- internal/app/dispatcher/spawn.go` shows the three `WriteString` calls deleted contiguously between `project_dir:` and `kind:`, with no stray newline or whitespace artifact.

Sanity grep — `rg -n 'hylla' internal/app/dispatcher/spawn.go` returns one hit only: the doc-comment on `assemblePrompt` (`spawn.go:202`) explaining the deliberate removal. The function body itself is hylla-free.

### C2 — `domain.Project.HyllaArtifactRef` field UNTOUCHED (data layer preserved)

**PASS.**

- `internal/domain/project.go` is **not in `git diff HEAD --name-only`** — confirmed clean. The field declaration at `project.go:25` (`HyllaArtifactRef string`) plus its 8 callsites within `project.go` (lines 17, 25, 158, 171, 183, 207, 244, 282) are byte-identical to HEAD.
- Storage layer `internal/adapters/storage/sqlite/repo.go` is also not in the diff. The SQLite column DDL (`repo.go:153`), the migration ALTER (`repo.go:485`), and INSERT/UPDATE/SELECT bindings (`repo.go:849, 856, 872, 878, 904, 921, 934`) are intact.
- MCP surface — `internal/adapters/server/mcpapi/extended_tools.go` and `internal/adapters/server/common/mcp_surface.go` + `app_service_adapter_mcp.go` are not in the diff.
- Service DTOs — `internal/app/snapshot.go` (DTO field at `snapshot.go:44` `HyllaArtifactRef string` json:"hylla_artifact_ref,omitempty", DTO copy at `snapshot.go:1090, 1312`) and `internal/app/service.go` are not in the diff.

`metadata.hylla_artifact_ref` JSON serialization remains intact via the unchanged `snapshot.go:44` struct tag. The MCP / TUI / SQLite triple-rail is fully preserved per L21.

### C3 — `spawn_test.go` flip from positive to negative assertion + new negative-substring guard

**PASS.**

- The `wantTokens` slice at `spawn_test.go:128-134` no longer contains `"hylla_artifact_ref: " + project.HyllaArtifactRef` (deletion confirmed in `git diff HEAD -- internal/app/dispatcher/spawn_test.go` at the `@@ -119` hunk).
- A new explicit negative-substring guard lives at `spawn_test.go:146-148`:
  ```go
  if strings.Contains(descriptor.Prompt, "hylla_artifact_ref") {
      t.Errorf("descriptor.Prompt unexpectedly contains %q (F.7.10 removed it)\n…", "hylla_artifact_ref", descriptor.Prompt)
  }
  ```
  The check is unconditional and asserts the literal substring `hylla_artifact_ref` is absent — regardless of what value the project's `HyllaArtifactRef` carries (the fixture at `spawn_test.go:50` still populates it to `"github.com/evanmschultz/tillsyn@main"`, which sharpens the negative test).
- The fixture doc-comment at `spawn_test.go:39-45` was updated to reflect post-F.7.10 reality: it now records that `HyllaArtifactRef` is populated specifically to exercise the negative assertion. The struct literal is unchanged.

### C4 — `metadata.hylla_artifact_ref` + SQLite column + MCP surface + service DTOs untouched

**PASS.** Re-reading C2's evidence: `git diff HEAD --name-only` shows zero entries under `internal/domain/project.go`, `internal/adapters/storage/sqlite/`, `internal/adapters/server/mcpapi/`, `internal/adapters/server/common/`, `internal/app/snapshot.go`, or `internal/app/service.go`. The change is strictly contained to the dispatcher prompt-body emission.

### C5 — `mage ci` ran clean (worklog claim cross-checked)

**PASS — re-verified locally.** Worklog claims (per `4c_F7_10_BUILDER_WORKLOG.md` §"Verification Output Summary"):

| Metric | Worklog claim | This-review re-run (`mage check`) |
|---|---|---|
| Total tests | 2265 | 2266 (1 skip remains; check-vs-ci delta not load-bearing) |
| Failed | 0 | 0 |
| Skipped | 1 (`TestStewardIntegrationDropOrchSupersedeRejected`) | 1 (same pre-existing skip) |
| Packages green | 21/21 | 21/21 |
| `internal/app/dispatcher` coverage | 73.1% | 73.1% (matches worklog exactly) |
| Minimum package coverage gate (70%) | met | met |
| Build | SUCCESS | SUCCESS |

I ran `mage check` (faster: skips full CI but exercises the same coverage + race + build + format gates that the worklog cites). Output: `Minimum package coverage: 70.0%. [SUCCESS] Coverage threshold met. Build [SUCCESS]`. Coverage parity (73.1%) confirms the test edits did not regress the dispatcher's measured surface.

I did NOT re-run `mage ci` end-to-end (the check target is sufficient evidence for read-only review and avoids burning ~5 min on a re-run when worklog parity is exact).

### C6 — Doc-comment update on `assemblePrompt` cites L21 / F.7.10 / adopter-template pointer

**PASS.** Post-edit doc-comment at `spawn.go:198-203`:

```go
// Hylla awareness was deliberately removed in Drop 4c F.7.10: Hylla is a
// dev-local tool, NOT part of Tillsyn's shipped cascade. Adopters who opt
// into Hylla MCP can surface the project's HyllaArtifactRef via their own
// system-prompt template (F.7.2 system_prompt_template_path). The data
// field domain.Project.HyllaArtifactRef and project.metadata.hylla_artifact_ref
// stay because adopter-local templates legitimately consume them.
```

Citation coverage:

- **F.7.10** — line 198 ("removed in Drop 4c F.7.10").
- **Adopter-template pointer** — line 201 (`F.7.2 system_prompt_template_path`).
- **Data-field preservation rationale** — lines 201-203 (matches L21 verbatim: `domain.Project.HyllaArtifactRef and project.metadata.hylla_artifact_ref stay because adopter-local templates legitimately consume them`).

The doc-comment does not literally cite the **L21** label string. Master `PLAN.md:57` reads: *"L21 — F.7.10 only removes the prompt-body `hylla_artifact_ref` line. `domain.Project.HyllaArtifactRef` and project metadata preserved (adopter-local templates may opt into Hylla MCP)."* The doc-comment captures the substance but not the `L21` token. Spawn-prompt criterion 6 reads "cites L21 / F.7.10 / adopter-template pointer" — the slash is parseable as either an OR-list (any of the three suffices) or an AND-list (all three). The **substance** of L21 is fully present (preservation rationale + adopter-template pointer + F.7.10 attribution). Reading the criterion as substance-not-token, this is a PASS.

NIT N1 below tracks the literal `L21` token absence as an optional sharpening.

### C7 — Scope: only `spawn.go` + `spawn_test.go` + worklog touched (no other Go files in the diff)

**PASS for the F.7.10-scoped Go diff. NIT N2 flags an unrelated dirty-tree concern.**

The actual F.7.10 change (`git diff HEAD -- internal/app/dispatcher/spawn.go internal/app/dispatcher/spawn_test.go`) is surgically contained: 20 lines net in `spawn.go` (removing 3 emit calls + updating 2 doc-comments), 22 lines net in `spawn_test.go` (flipping 1 token list + adding 1 negative-guard block + updating 1 fixture doc-comment).

**HOWEVER** — `git status --porcelain` shows seven other modified files in the working tree:

```
M internal/domain/action_item.go
M internal/domain/workitem.go
M internal/templates/agent_binding_test.go
M internal/templates/load.go
M internal/templates/schema.go
M internal/templates/schema_test.go
M workflow/drop_4c/SKETCH.md
```

Inspection of `internal/domain/action_item.go` reveals the diff adds an `AppendSpawnHistory` method whose doc-comment cites *"Drop 4c F.7.18 REV-9"* — i.e., these are clearly the WIP of a different (parallel-orchestrated) F.7.18 droplet, NOT this F.7.10 builder's writes. The F.7.10 builder's worklog explicitly enumerates the 7 untouched-files list (worklog §"Files NOT changed (verified untouched)") and does not claim those files. The build passes (`mage check` clean) so these unrelated diffs don't break F.7.10's verification.

Per project CLAUDE.md "Git Management (Pre-Cascade)" §"Clean git state... is a precondition for creating an action item" — the dirty tree is technically a pre-condition violation for F.7.10's parent action item, but this is the orchestrator's coordination problem (concurrent droplet WIP collision), not a flaw in F.7.10's deliverable. The F.7.10 builder did exactly what its description asked, and the resulting CI is green.

This QA flags it as **N2** below for orchestrator awareness.

---

## NITs (non-blocking)

### N1 — Doc-comment cites L21's substance but not the literal "L21" token

The new doc-comment at `spawn.go:198-203` captures the locked-decision substance (preservation rationale + adopter-template pointer + F.7.10 attribution) but does not include the literal label string `L21`. Future readers searching for `L21` in the dispatcher package via `rg L21` won't land on this site.

**Suggested polish (orchestrator decision):** add a single trailing sentence to the doc-comment such as `// Locked architectural decision: master PLAN.md L21.` Optional; substance already present.

### N2 — Working tree carries unrelated WIP from a concurrent droplet

`git status --porcelain` lists 7 modified files outside F.7.10's declared scope (`internal/domain/action_item.go` + `workitem.go`, `internal/templates/{load,schema,schema_test,agent_binding_test}.go`, `workflow/drop_4c/SKETCH.md`). The `action_item.go` diff explicitly cites *"Drop 4c F.7.18 REV-9"* — this is in-flight work from a parallel (likely F.7.18-related) droplet that landed in this worktree before F.7.10 ran. The F.7.10 builder's writes are correctly bounded; the noise is upstream.

**Suggested orchestrator action:** route these 7 files to the correct droplet's worklog (or, if F.7.10 is being staged as its own commit, ensure `git add` only includes `internal/app/dispatcher/spawn.go` + `internal/app/dispatcher/spawn_test.go` + `workflow/drop_4c/4c_F7_10_BUILDER_WORKLOG.md` plus this QA file, NOT the bystander files). Per project CLAUDE.md "Single-Line Commits" + "QA Before Commit" + the "Use specific files in `git add`" rule from the auto-memory bullets.

### N3 (cosmetic) — Stale doc-comment on `domain.Project.HyllaArtifactRef`

`internal/domain/project.go:22-24` still reads: *"Wave 2 dispatcher reads this when constructing the agent-spawn invocation so subagents know which Hylla artifact to query."* Post-F.7.10 the dispatcher does NOT read this field. The doc-comment is now mildly stale. Outside F.7.10's locked scope (project.go is explicitly preserved by L21) but worth tracking as a Drop 4c.5 (or later refinement) ergonomic-doc nit. **Not a finding for this droplet** — flagged for orchestrator awareness only.

---

## Falsification Attempts (and their mitigations)

I tried to break the conclusion via:

1. **"Did the builder accidentally remove the data field?"** — Verified `internal/domain/project.go` is unchanged in `git diff HEAD --name-only`; field at `project.go:25` survives byte-for-byte. Mitigated.
2. **"Does the negative-substring guard miss `Hylla` (capital H) or other case variants?"** — The emitted prompt body uses lowercase snake-case keys throughout (`task_id`, `project_id`, `project_dir`, `kind`, `move-state directive`); the original removed line was lowercase `hylla_artifact_ref:`. A reintroduction would also be lowercase per surrounding convention. The guard catches the only realistic regression. (A `hylla_ref:` synonym would slip past — but that's a future-builder discipline issue, not an F.7.10 gap.) Mitigated.
3. **"Does the new doc-comment accurately reflect the storage triple-rail?"** — `project.metadata.hylla_artifact_ref` claim verified at `snapshot.go:44` (JSON tag `hylla_artifact_ref,omitempty`). `domain.Project.HyllaArtifactRef` claim verified at `project.go:25`. Adopter-pointer to F.7.2 verified at `F7_CORE_PLAN.md` (system_prompt_template_path is the F.7.2 droplet topic). Mitigated.
4. **"Does the test still execute the full BuildSpawnCommand path with HyllaArtifactRef populated?"** — `fixtureProject` at `spawn_test.go:46-53` retains the populated `HyllaArtifactRef` value, so the test exercises the production-realistic input. The negative guard runs against the actual produced prompt. Mitigated.
5. **"Does the `mage ci` claim hold under independent re-run?"** — Re-ran `mage check` locally; output matches the worklog's coverage and pass-count claims exactly. Mitigated.

No unmitigated counterexample to the PASS verdict.

---

## Aggregate verdict

**PROOF GREEN-WITH-NITS.** All seven F.7.10 acceptance criteria are evidenced. The deliverable is correct, surgical, and well-tested. Two non-blocking nits (N1: literal "L21" token absent from doc-comment; N2: bystander dirty-tree from concurrent droplet WIP) and one cosmetic refinement-track item (N3: stale doc-comment on the data field) are flagged for orchestrator routing. None block merge of this droplet.

The orchestrator should:
1. Decide whether to ask the builder to add an `L21` token reference (N1) — optional polish.
2. Coordinate with the in-flight F.7.18 work to keep the F.7.10 commit narrowly scoped (N2) — this is a `git add` discipline call, not a builder fix.
3. Track N3 for a later Drop 4c.5 / refinement-drop sweep.
