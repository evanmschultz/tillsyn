# Drop 4c — F.7.10 Builder QA Falsification

**Droplet:** F.7.10 — Drop hardcoded `hylla_artifact_ref` from `spawn.go` prompt body
**QA model:** opus (falsification, fresh context)
**Date:** 2026-05-04
**Mode:** Filesystem-MD (no Tillsyn action items)
**Stance:** read-only adversarial — try to break the claim

## Falsification Certificate

- **Premises** — F.7.10 must (a) delete the three `b.WriteString` calls that emitted `hylla_artifact_ref: <value>\n` from `assemblePrompt`, (b) leave `domain.Project.HyllaArtifactRef`, the SQLite column, MCP surface, and service DTOs intact, (c) flip the `spawn_test.go` assertion from positive ("MUST contain") to negative ("MUST NOT contain"), (d) keep the change scoped to `internal/app/dispatcher`. Per F7-CORE plan decision #9 (line 24) and master PLAN L21.
- **Evidence** — `git diff HEAD -- internal/app/dispatcher/spawn.go internal/app/dispatcher/spawn_test.go` (the two builder-edited files); `rg "HyllaArtifactRef|hylla_artifact_ref"` across the project (32 matches outside the dispatcher, all preserved); `Read` of `dispatcher.go:150–160`, `dispatcher.go:410–425`, `cmd/till/dispatcher_cli.go`, `cmd/till/dispatcher_cli_test.go`, `internal/adapters/storage/sqlite/repo.go`, `internal/adapters/server/mcpapi/extended_tools.go`, `internal/domain/project.go`, `internal/app/snapshot.go`, `internal/app/service.go`.
- **Trace or cases** — see Findings + per-attack verdicts below.
- **Conclusion** — **PASS-WITH-NITS**. No CONFIRMED counterexample. Two NITs worth a follow-up commit, neither blocks merge.
- **Unknowns** — none.

## 1. Per-Attack Verdicts

### 1.1 A1 — Scope creep (REFUTED)

**Attempt:** ran `rg "HyllaArtifactRef|hylla_artifact_ref" --type go` to enumerate every consumer of the field and confirm none were touched outside `internal/app/dispatcher/`.

**Evidence:** 32 matches outside dispatcher all reachable + intact:

- `internal/domain/project.go:25` — field declaration intact.
- `internal/domain/project.go:158, :171, :183, :207, :244, :282` — `NewProjectFromInput` round-trip, MCP "first-class fields" comment, all preserved.
- `internal/adapters/storage/sqlite/repo.go:153, :485, :849, :856, :872, :878, :904, :921, :934, :2862` — DDL, ALTER, INSERT, UPDATE, SELECT, scan rows — all intact.
- `internal/adapters/server/mcpapi/extended_tools.go:439, :463, :529, :584` — MCP `mcp.WithString("hylla_artifact_ref", ...)` arg + `args.HyllaArtifactRef` plumbing intact (CreateProject + UpdateProject both wired).
- `internal/app/service.go:277, :288, :315, :335, :444, :472` — service-layer `CreateProjectInput` / `UpdateProjectInput` DTOs intact.
- `internal/app/snapshot.go:34, :44, :1090, :1312` — snapshot DTO `json:"hylla_artifact_ref,omitempty"` and round-trip intact.
- `internal/tui/model.go` (no matches — TUI doesn't read it directly).
- `internal/tui/model_test.go:822, :857, :4061, :14983, :14992` — TUI field-coverage tests intact.

**Verdict:** **REFUTED.** Scope is exactly `internal/app/dispatcher` per the locked-package declaration. Builder did not over-reach.

### 1.2 A2 — Doc-comment regression (REFUTED, with sibling-NIT spillage — see §2.2)

**Attempt:** read both touched doc-comments verbatim. Hunt for any phrasing that reintroduces "Tillsyn-internal awareness" of Hylla into the prompt-body contract.

- `SpawnDescriptor.Prompt` (lines 62–66 post-edit): structural-fields list now reads `(task_id, project_dir, move-state directive)` — `hylla_artifact_ref` cleanly excised. No reintroduction.
- `assemblePrompt` (lines 192–206 post-edit): the multi-line note explicitly directs adopters at `system_prompt_template_path` (cross-reference to F.7.2) for opt-in Hylla MCP. Phrasing: *"Adopters who opt into Hylla MCP can surface the project's HyllaArtifactRef via their own system-prompt template (F.7.2 system_prompt_template_path). The data field domain.Project.HyllaArtifactRef and project.metadata.hylla_artifact_ref stay because adopter-local templates legitimately consume them."* — this is exactly the adopter-template-opt-in path described in F7-CORE plan decision #9. Doc-comment is correct.

**Verdict:** **REFUTED for the touched files.** But: see §2.2 for two stale comments in `dispatcher.go` that the builder did NOT touch and that still claim `HyllaArtifactRef` is "prompt structural field" / "spawn-site fields the project carries." Those are NITs raised under §2 below.

### 1.3 A3 — Test brittleness (NIT)

**Attempt:** construct a future-refactor counterexample where the negative guard `strings.Contains(descriptor.Prompt, "hylla_artifact_ref")` fails to catch a re-emission.

**Counterexamples that SLIP THE GUARD:**

- Capitalized: `b.WriteString("Hylla artifact: ")` — the substring `hylla_artifact_ref` is not present; guard returns false; test passes despite the leak.
- CamelCased: `b.WriteString("hyllaArtifactRef: ")` — same; substring absent.
- Short form: `b.WriteString("hylla: ")` or `b.WriteString("artifact_ref: ")` — substring absent.

These are theoretical future-builder-refactor risks, NOT current counterexamples — the current code emits nothing at all. The plan acceptance criterion ("Test asserts the rendered prompt does NOT contain the substring `hylla_artifact_ref`") is met **literally and exactly** by the current guard. A stronger guard (e.g., a `Hylla`-case-insensitive check, or a regex `(?i)hylla[\s_]*artifact[\s_]*ref`) would be more defensive but isn't required by the spec.

**Verdict:** **NIT.** The guard meets the literal acceptance criterion. A defense-in-depth tightening would be a small follow-up; not blocking. (Recorded as §2.1 NIT.)

### 1.4 A4 — Test coverage gap (REFUTED)

**Attempt:** find a path through `assemblePrompt` that was previously asserted but is no longer.

**Pre-edit `assemblePrompt`:** unconditionally wrote `hylla_artifact_ref: ` + `project.HyllaArtifactRef` regardless of whether the value was empty. There is NO branching on `HyllaArtifactRef` — the function is linear.

**Post-edit `assemblePrompt`:** no branching. Linear.

**Pre-edit test:** asserted `"hylla_artifact_ref: " + project.HyllaArtifactRef` in `wantTokens`. With the fixture's populated value (`github.com/evanmschultz/tillsyn@main`), this exercised the populated-value path. There was NO test covering the empty-value path.

**Post-edit test:** asserts `hylla_artifact_ref` substring is absent regardless of fixture value. The fixture remains populated specifically to harden this assertion. There is no path coverage that disappears in the flip.

**Plan acceptance criterion** explicitly lists *"Edge: project with empty `HyllaArtifactRef` → still no substring (unchanged)"* — but pre- and post-edit `assemblePrompt` is linear (no branch on the field), so the empty-fixture test would be a no-op tautology. The builder didn't add it; the plan calls it "unchanged"; correct.

**Verdict:** **REFUTED.** No previously-tested path lost coverage. The only path through `assemblePrompt` (linear, no branching) is exercised by the existing populated-fixture test.

### 1.5 A5 — Adopter-template path silently broken? (REFUTED)

**Attempt:** trace the adopter-opt-in path post-F.7.10. If an adopter wants Hylla MCP awareness, what's the documented + implementable route?

**Path:** F.7.2 (sibling droplet, not yet landed) introduces `system_prompt_template_path` on the agent binding. The adopter authors a system-prompt template that reads `project.HyllaArtifactRef` and emits whatever framing they want into the agent's system prompt. The data is still:

- on the project (`domain.Project.HyllaArtifactRef` — preserved).
- in the SQLite column (`hylla_artifact_ref` — preserved).
- on the MCP surface (`extended_tools.go` — preserved).
- in the snapshot/service DTOs (preserved).

The post-edit doc-comment on `assemblePrompt` explicitly cross-references this path: *"Adopters who opt into Hylla MCP can surface the project's HyllaArtifactRef via their own system-prompt template (F.7.2 system_prompt_template_path)."* No silent break.

**Verdict:** **REFUTED.** The opt-in path is documented inline + the underlying data plumbing is preserved end-to-end.

### 1.6 A6 — `assemblePrompt` consumer regression (REFUTED)

**Attempt:** `rg -n "assemblePrompt"` to find every caller.

**Result:** exactly ONE call site:

```
internal/app/dispatcher/spawn.go:138:    prompt := assemblePrompt(item, project, authBundle)
```

The result is consumed in one place: `descriptor := SpawnDescriptor{ ..., Prompt: prompt, ... }` (line 161) and the `-p` argv slot (line 144). No other caller in `cmd/`, `internal/app/`, `internal/adapters/`, or any test. No caller has ever parsed the result for `hylla_artifact_ref:`.

**Verdict:** **REFUTED.** Single producer, no parsing consumers.

### 1.7 A7 — `SpawnDescriptor.Prompt` field consumers (REFUTED)

**Attempt:** `rg -n "descriptor.Prompt|\.Prompt\b"` to find every read of the field downstream.

**Findings:**

- `internal/app/dispatcher/spawn.go:144` — argv slot `"-p", prompt`. Opaque pass-through.
- `internal/app/dispatcher/spawn.go:161` — `SpawnDescriptor.Prompt: prompt` initialization.
- `internal/app/dispatcher/spawn_test.go:88` — argv slot in test fixture, opaque.
- `internal/app/dispatcher/spawn_test.go:136-147` — substring-in-prompt assertions (the touched test).
- `internal/app/dispatcher/spawn_test.go:364` — zero-value field check.
- `cmd/till/dispatcher_cli.go:202, :254` — JSON wire-form rename to snake_case `prompt` for the CLI dry-run output. Does NOT parse for `hylla_artifact_ref:`.
- `internal/app/dispatcher/dispatcher.go:691` — `Descriptor SpawnDescriptor` struct embedding in a richer outcome struct. No field-level read of `.Prompt`.

The TUI `internal/tui/model.go` matches for `searchInput.Prompt`, `commandInput.Prompt`, etc. are unrelated — those are Bubble Tea text-input prompt strings, NOT `SpawnDescriptor.Prompt`.

**Verdict:** **REFUTED.** No downstream consumer parses `SpawnDescriptor.Prompt` for `hylla_artifact_ref:`. The field is treated as an opaque blob (CLI JSON output) or argv passthrough.

## 2. NITs (Non-Blocking Follow-Ups)

### 2.1 N1 — Negative guard is literal-substring only (recommendation: tighten)

The current `strings.Contains(descriptor.Prompt, "hylla_artifact_ref")` guard catches re-emissions of the exact snake_case literal but slips:

- `Hylla artifact:` (capitalized, space-separated).
- `hyllaArtifactRef:` (camelCase).
- `artifact_ref:` (truncated).

A future builder refactor that "moves Hylla awareness back into the prompt body via a different name" would not be caught by the current test. Recommended tightening (small, ≤3 LOC):

```go
prompt := strings.ToLower(descriptor.Prompt)
for _, banned := range []string{"hylla_artifact_ref", "hyllaartifactref", "hylla artifact"} {
    if strings.Contains(prompt, banned) {
        t.Errorf("...")
    }
}
```

Not blocking — the literal acceptance criterion is met. Surface as a refinement-list entry if F.7.10 closes clean.

### 2.2 N2 — Stale doc-comments in `dispatcher.go` (untouched by builder)

Post-F.7.10, the dispatcher no longer consumes `HyllaArtifactRef` for prompt structural-field assembly. Two doc-comments in `internal/app/dispatcher/dispatcher.go` still claim it does:

- **`dispatcher.go:151-157`** — `projectReader` interface comment: *"...needs the project's RepoPrimaryWorktree (cmd.Dir for the spawn), KindCatalogJSON (resolves AgentBinding for the action item's kind), and HyllaArtifactRef (prompt structural field)."* The "prompt structural field" claim is now false.
- **`dispatcher.go:413-417`** — Stage 1 project resolution comment: *"The project carries the spawn-site fields (RepoPrimaryWorktree, KindCatalogJSON, HyllaArtifactRef) and the conflict-detector / walker need ProjectID..."* — `HyllaArtifactRef` is no longer a "spawn-site field" the dispatcher consumes.

Builder's worklog explicitly listed `dispatcher.go` under "Files NOT changed (verified untouched)." Strict reading of the spec ("**Files to edit/create:** `spawn.go`, `spawn_test.go`") supports the builder's choice — but doc-comment drift in a sibling file documents a non-existent contract. Not in builder's scope per the locked-files list; would need a sibling F.7.X follow-up or a refinement entry.

**Verdict on the builder's choice:** defensible (out of declared scope), but the drift exists. Surface as a refinement-list entry. **Not blocking F.7.10 close.**

## 3. Hylla Feedback

`None — no Hylla queries issued.` Per the orchestrator's spawn-prompt directive (`No Hylla calls`), all code lookups went through `Read` + `rg`. No fallback-from-Hylla path was attempted; no miss to record.

## 4. Summary

**Final verdict:** **PASS-WITH-NITS**.

- All seven attack vectors REFUTED at the CONFIRMED level.
- Two NITs (N1 substring-guard literalism; N2 stale doc-comments in untouched `dispatcher.go`). Neither blocks merge.
- Builder followed the locked spec exactly: deleted the three `b.WriteString` lines, preserved `domain.Project.HyllaArtifactRef` end-to-end, flipped the assertion from positive to negative, kept scope inside `internal/app/dispatcher/`. Doc-comments inside the touched files were updated to match post-edit reality (the worklog flagged this scope-edge clarification honestly).
- `mage ci` passes per the worklog (2264/2265 tests, 21/21 packages, coverage above 70%).

Recommend close + log N1 + N2 as drop-end refinements.

## TL;DR

- **T1** Per-attack verdicts: A1 REFUTED (no scope creep beyond `internal/app/dispatcher`), A2 REFUTED for touched files (with N2 spillage), A3 NIT (literal-substring guard), A4 REFUTED (linear function — no path coverage lost), A5 REFUTED (adopter opt-in path documented + plumbed), A6 REFUTED (single producer, no parsing consumer), A7 REFUTED (no downstream `Prompt`-body parser).
- **T2** N1 (test guard tightening) + N2 (`dispatcher.go:151–157` and `:413–417` stale doc-comments) are non-blocking refinements; surface for drop-end refinement list.
- **T3** Hylla feedback: none — no Hylla queries issued per spawn-prompt directive.
- **T4** Final verdict: **PASS-WITH-NITS**. Recommend close + log N1 + N2 to refinements.
