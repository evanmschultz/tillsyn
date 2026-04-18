---
round: 4
verdict: PASS
---

# DROP_1_75_KIND_COLLAPSE — Plan QA Proof, Round 4

Scope: verify the Round 3 triage commit `8bd81b0` ("docs(drop-1-75): plan round 3 triage fixes") applied all 11 dev-approved fixes cleanly, didn't introduce DAG cycles, and preserved Scope-to-unit coverage.

## Triage Fix Verification

Every Round 3 fix was located in the current PLAN.md and cross-checked against the evidence it cites.

### F1 / P2 — Unit 1.12 package-level blocker

- **Finding**: Applied. Line 219 reads `Blocked by: 1.3, 1.5, 1.6, 1.7` (was `1.3, 1.5, 1.7`).
- **Evidence**: `git diff 8bd81b0~1 8bd81b0 -- drops/DROP_1_75_KIND_COLLAPSE/PLAN.md` shows `-Blocked by: 1.3, 1.5, 1.7` → `+Blocked by: 1.3, 1.5, 1.6, 1.7`. Package `internal/adapters/server/mcpapi` is shared between 1.5, 1.6 (via `instructions_explainer.go`) and 1.12 (via `mcpapi` tests) — the 1.6 edge closes the gap.
- **Severity**: —
- **Suggestion**: —

### F2 — Unit 1.3 extended schema strip (CREATE TABLE + ALTER)

- **Finding**: Applied in both Scope.5 (line 25) and Unit 1.3 Scope/Acceptance (lines 93-101). Title updated to "strip projects.kind schema (CREATE + ALTER)" on line 87. New test `TestRepositoryFreshOpenProjectsSchema` declared in acceptance. Quoted-DDL rg guard added to 1.3 acceptance (line 96) and also to 1.15 end-state invariant (line 268).
- **Evidence**: `repo.go:152` confirmed to carry `kind TEXT NOT NULL DEFAULT 'project',` inside the `CREATE TABLE IF NOT EXISTS projects` block at line 147-157. 1.3 Scope prose explicitly names both sites `(a) :588 ALTER` and `(b) :152 CREATE TABLE ... inside migrate() at :144`. Acceptance has both rg guards.
- **Severity**: —
- **Suggestion**: —

### F3 — Unit 1.4 mage waiver + Unit 1.5 compile-restoration burden

- **Finding**: Applied. Line 117 carries the unambiguous waiver: `**mage build and mage ci are waived for this unit only.** ... Builder honors this waiver; QA does not fail the unit on workspace-compile failure.` Line 129 carries the reciprocal declaration on 1.5: `**This unit carries the workspace-compile-restoration burden** — the 1.4 waiver expects 1.5 to re-green the workspace via this check.`
- **Evidence**: Round-4 QA reading WORKFLOW.md Phase 5 would see "builder runs `mage build`/`mage test` per unit." The 1.4 waiver explicitly overrides for that unit only — language is tight enough to prevent a well-intentioned QA from spuriously failing. 1.5 acceptance re-anchors on `mage ci` with the compile-restoration call-out.
- **Severity**: —
- **Suggestion**: —

### F4 — F5-classification 5th bullet (KindTemplate machinery)

- **Finding**: Applied at line 53. New bullet under "Orphan-via-collapse refactor" covers `KindDefinition.Template` / `KindTemplate` / `KindTemplateChildSpec` / `validateKindTemplateExpansion` / `normalizeKindTemplate` / `ErrInvalidKindTemplate`, classified **naturally unreachable** because baked `kind_catalog` rows carry empty `auto_create_children`.
- **Evidence**: Reconciles with Unit 1.4's acceptance (line 116) which intentionally preserves `ErrInvalidKindTemplate`, and with Unit 1.11's acceptance (line 210) which rewrites `AutoCreateChildren`-asserting tests to "assert empty (matching the F5 `KindTemplate` classification — post-collapse `kind_catalog` rows carry empty children)."
- **Severity**: —
- **Suggestion**: —

### F5-finding — Unit 1.6 explicit project.go citations

- **Finding**: Applied at line 25 (Scope.5) and line 137 (Unit 1.6 Paths). Replaces the ambiguous "assignment at `:85`" with explicit `type Project` field at `:16`, `NewProject` struct-literal `Kind: DefaultProjectKind,` at `:60`, `SetKind(kind KindID, ...) error { ... p.Kind = kind; ... }` method deletion at `:79-88`, plus sitewide `Project{...}` audit.
- **Evidence**: Direct read of `internal/domain/project.go`:
  - `:11-21` `type Project struct { ... Kind KindID ... }` at `:16`.
  - `:55-64` `NewProject`'s `return Project{ ... Kind: DefaultProjectKind, ... }` at `:60`.
  - `:79-88` `func (p *Project) SetKind(kind KindID, now time.Time) error { ... p.Kind = kind ... }` spanning `:79-88` with assignment at `:85`.
  All three citations in PLAN.md match the source exactly.
- **Severity**: —
- **Suggestion**: —

### F6 — Unit 1.4 precise error-sentinel checks

- **Finding**: Applied at lines 112-116. Replaces the blunt `grep -c 'ErrTemplate' internal/domain/errors.go` returns 0 (which would false-positive on `ErrInvalidKindTemplate`) with four separate rg checks:
  - `ErrTemplateLibraryNotFound` → 0
  - `ErrInvalidTemplate(Library|LibraryScope|Status|ActorKind|Binding)` → 0
  - `ErrBuiltinTemplateBootstrapRequired|ErrNodeContractForbidden` → 0
  - `ErrInvalidKindTemplate` → 1 (intentionally preserved)
- **Evidence**: The 4-line check block is structurally sound — each regex is precise and the preservation assertion is positive (`returns 1`, not `returns 0`).
- **Severity**: —
- **Suggestion**: —

### F7 — Unit 1.2 skip-sites-destined-for-deletion

- **Finding**: Applied at line 85. New language: `**Intentionally skip** call sites inside files destined for deletion by unit 1.5 (internal/app/template_library_builtin.go:29, :79, template_library.go, template_contract.go, template_reapply.go) — 1.5's wholesale file deletion moots them, so edits here would be pure churn.`
- **Evidence**: Consistent with Unit 1.5's Paths which lists all four named files. No conflict on same edit.
- **Severity**: —
- **Suggestion**: —

### P9 — handler.go line-drift correction

- **Finding**: Applied at line 23 (Scope.4) and line 124 (Unit 1.5 Paths). `handler.go:1045 pickTemplateLibraryService` → `handler.go:1046-1054 pickTemplateLibraryService (per P9 line-drift correction)`.
- **Evidence**: Direct read of `handler.go` at `:1045-1054`: line 1045 is a closing `}` of the previous function, line 1046 is `// pickTemplateLibraryService resolves ...`, line 1046 is followed by the `func pickTemplateLibraryService(...)` declaration with body through `:1054`. Scope.4's range `:1046-1054` matches. (Unit 1.5's inline citation says `at :1045-1050` for the body - a slight discrepancy; not blocking because 1.5's acceptance uses rg-based proof rather than line-number strict equality. See P1 below.)
- **Severity**: —
- **Suggestion**: —

### P10 — Unit 1.11 paths narrowed

- **Finding**: Applied at line 203. `search_embeddings_test.go` and `embedding_runtime_test.go` removed from Paths; line 212 now explicitly notes "Per P10: `search_embeddings_test.go` and `embedding_runtime_test.go` are **not** in Paths."
- **Evidence**: Diff lines 202-203 show the before/after; narrative added in line 212.
- **Severity**: —
- **Suggestion**: —

### P11 — Unit 1.5 `TemplateLibraryRepo` port prose fix

- **Finding**: Applied at line 124 (Unit 1.5 Paths): `internal/app/ports.go (strip the 9 TemplateLibrary* / NodeContractSnapshot* / ProjectTemplateBinding* methods from the unified Repository interface at :24-32 — per P11, there is no standalone TemplateLibraryRepo port; the methods live on Repository)`.
- **Evidence**: Direct read of `ports.go:20-40` confirms unified `Repository` interface carries 9 method signatures `:24-32`: `UpsertTemplateLibrary`, `GetTemplateLibrary`, `ListTemplateLibraries`, `UpsertProjectTemplateBinding`, `GetProjectTemplateBinding`, `DeleteProjectTemplateBinding`, `CreateNodeContractSnapshot`, `UpdateNodeContractSnapshot`, `GetNodeContractSnapshot` — all on a single `type Repository interface { ... }`. No standalone `TemplateLibraryRepo` port exists. PLAN.md prose now matches source.
- **Severity**: —
- **Suggestion**: —

### P15 — Unit 1.15 title

- **Finding**: Applied at line 257. Title renamed `### 1.15 — Drop-end mage ci gate` → `### 1.15 — Drop-end verification (mage ci + push + CI watch)`.
- **Evidence**: Diff confirms the rename. Acceptance body (lines 264-268) lists `mage ci`, `git push + gh run watch --exit-status`, coverage, end-state rg sweep, quoted-DDL guard — all four clauses match the new title.
- **Severity**: —
- **Suggestion**: —

## DAG Re-Verification

Edge list after triage (from lines 66, 80, 92, 108, 126, 139, 152, 165, 178, 193, 205, 219, 232, 244, 262):

| Unit | Blocked by |
|------|------------|
| 1.1 | — |
| 1.2 | 1.1 |
| 1.3 | 1.1, 1.2 |
| 1.4 | 1.1 |
| 1.5 | 1.1, 1.2, 1.3, 1.4 |
| 1.6 | 1.1, 1.2, 1.3, 1.4, 1.5 |
| 1.7 | 1.1, 1.2, 1.3, 1.5 |
| 1.8 | 1.1, 1.4, 1.6 |
| 1.9 | 1.1, 1.4, 1.6, 1.8 |
| 1.10 | 1.1, 1.4, 1.6, 1.8, 1.9 |
| 1.11 | 1.2, 1.5, 1.6 |
| 1.12 | 1.3, 1.5, 1.6, 1.7 |
| 1.13 | 1.5, 1.6 |
| 1.14 | 1.1, 1.2, 1.3, 1.4, 1.5, 1.6, 1.7, 1.8, 1.9, 1.10, 1.11, 1.12, 1.13 |
| 1.15 | 1.14 |

Topological order exists: 1.1 → 1.2 → 1.3 → 1.4 → 1.5 → 1.6 → 1.7 → 1.8 → 1.9 → 1.10 → 1.11 → 1.12 → 1.13 → 1.14 → 1.15 (one valid ordering). No back edges. No cycle. 1.12's new 1.6 dependency is acyclic (1.6 has no path to 1.12).

## Scope-to-Unit Coverage

Scope items 1-8 map to units as follows (unchanged from Round 3, but re-audited with triage edits):

- Scope 1 (kind catalog collapse, delete seeders): 1.2, 1.3.
- Scope 2 (Go identifier rename `WorkKind → Kind`): 1.1.
- Scope 3 (file + type renames): 1.8, 1.9.
- Scope 4 (template_libraries excision): 1.4, 1.5.
- Scope 5 (drop projects.kind column — CREATE + ALTER + Go): 1.3 (both SQL sites), 1.6 (Go sites).
- Scope 6 (drops-rewrite.sql rewrite): 1.14.
- Scope 7 (legacy tasks excision): 1.7.
- Scope 8 (tests + fixtures): 1.10, 1.11, 1.12, 1.13.

All 8 Scope items have at least one owning unit. Triage edits to Scope.4, Scope.5 preserve coverage.

## Findings

### P1 — Unit 1.5 inline citation slightly stale for `pickTemplateLibraryService`

- **Finding**: Unit 1.5 Paths at line 124 still reads `handler.go (delete pickTemplateLibraryService at :1045-1050 + call at :66, :72, :86)`. The function body per direct Read spans `:1046-1054` (the inline comment is at `:1045`). Scope.4 at line 23 was corrected per P9 to `:1046-1054`, but the Unit 1.5 Paths prose wasn't propagated the same way.
- **Evidence**: `handler.go:1045-1054` reads: `1045: }` (closing prior fn), `1046: // pickTemplateLibraryService resolves ...`, `1046 declaration line in doc comment`, function body through `1054: }`. The true range is `:1046-1054` (or `:1045-1054` if you include the preceding blank line). The PLAN.md Unit 1.5 citation `:1045-1050` undershoots the body by 4 lines.
- **Severity**: low (non-blocker). The range is close enough that a builder navigating to `:1045-1050` will land inside or adjacent to the function and can use Go LSP to locate the symbol. Acceptance line 131 uses an rg-based check that doesn't depend on line numbers, so the drop-end gate still works. Scope.4's corrected `:1046-1054` already carries the authoritative citation.
- **Suggestion**: Optional polish: bring Unit 1.5 Paths line 124 into sync with Scope.4 (`:1046-1054`). Not required for Phase 4 build entry since Scope.4's correction is authoritative and the builder will have both citations in view.

### P2 — Minor: F5 bullet #5 + 1.4 `ErrInvalidKindTemplate` preservation = intentional dead code

- **Finding**: The F5 orphan-classification bullet #5 (line 53) classifies `ErrInvalidKindTemplate` as naturally-unreachable post-collapse, but Unit 1.4 acceptance (line 116) explicitly preserves the sentinel. This is internally consistent per the dev's orphan-via-collapse policy ("leave and just orphan... we don't want them actually running"), but worth making explicit for the refinement drop that picks up the cleanup.
- **Evidence**: F5 bullet #5 reads "Retention intentional; refinement drop deletes." Unit 1.4 acceptance reads "intentionally preserved for kind.go:normalizeKindTemplate, which is F5-classified as naturally unreachable but kept until refinement drop." The two citations are consistent — both defer deletion.
- **Severity**: low (non-blocker). This is the documented orphan-via-collapse pattern working as designed. The "dead code" label is semantic, not a bug.
- **Suggestion**: None. Pattern is correct; just noting the audit trail is clean for future archaeology.

## Verdict

**PASS.** All 11 Round 3 triage fixes landed in commit `8bd81b0` and read coherently against each other and against the underlying code evidence in `main/`. The 1.12 blocker addition is acyclic and closes a real package-sharing gap on `internal/adapters/server/mcpapi`. Unit 1.3's extended scope covers both `CREATE TABLE projects` (line 152) and `ALTER TABLE projects ADD COLUMN kind` (line 588) in both Scope and Acceptance, with a fresh-DB assertion test named. Unit 1.4's mage waiver is unambiguous and Unit 1.5 explicitly picks up the compile-restoration burden. F5 orphan-classification bullet #5 is internally consistent with 1.4's `ErrInvalidKindTemplate` preservation and 1.11's `AutoCreateChildren`-empty rewrite guidance. Scope-to-unit coverage preserved; 15-unit DAG clean. Only two low-severity findings: a slightly stale line citation on Unit 1.5 (Scope.4's corrected range is authoritative) and a note that `ErrInvalidKindTemplate` retention is intentional dead code per the orphan policy. Neither blocks Phase 4 build entry. Plan is ready to ship.
