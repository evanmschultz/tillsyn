---
drop: DROP_1_75_KIND_COLLAPSE
phase: plan-qa
role: qa-falsification
round: 4
verdict: FAIL
---

# Plan QA — Falsification (Round 4)

Round-4 bar: re-attack the plan after Round 3 triage commit `8bd81b0`. Hunt
regressions from the triage edits (units 1.3, 1.4, 1.5, 1.6, 1.11, 1.12, 1.15),
re-verify the DAG with the new `1.12 → 1.6` edge, probe F5 5th-bullet
coherence, probe F2 schema-rebuild coordination between units 1.3 and 1.14,
probe unit 1.4 waiver scope creep, and reconfirm file-share blockers.

Prior rounds' blocking counterexamples (F1 serialization, F2 schema-strip)
are addressed by the triage commit and re-verified below.

---

## F1 — Unit 1.6 cannot pass `mage ci` while test files outside its Paths still reference deleted `SetKind` / `project.Kind` readbacks

**Counterexample.** Unit 1.6 deletes (a) the `Kind KindID` field from
`type Project` at `internal/domain/project.go:16`, (b) the `Kind: DefaultProjectKind,`
entry at `:60`, and (c) the `SetKind(kind KindID, ...)` method entirely at
`:79-88`. Its acceptance requires `mage ci` to succeed.

`mage ci` compiles every test package. After 1.6 lands, the following test
sites in packages **not in unit 1.6's Paths** will fail to compile or fail at
runtime:

- `internal/adapters/storage/sqlite/repo_test.go:2368-2381` — builds a
  `domain.NewProject(...)`, calls `project.SetKind("project-template", now)`
  at line 2369, persists, then asserts `loadedProject.Kind != domain.KindID("project-template")`
  at line 2379. Both `SetKind` (method deletion) and `loadedProject.Kind`
  (field deletion) reference symbols unit 1.6 removed.
- `internal/tui/model_test.go:15202` — reads
  `string(svc.lastCreateProject.Kind)`. `lastCreateProject` is an
  `app.CreateProjectInput` (not `domain.Project`), but unit 1.6's Scope
  names `internal/app/service.go` in Paths — whether the `CreateProjectInput.Kind`
  field at `service.go:226` is also deleted is ambiguous in the Scope prose
  (1.6 says "strip project.Kind references" but this is on the input type,
  not the domain type). If the field is deleted, this test breaks; if not,
  the cleanup is incomplete. Either way, 1.13 owns the test site — runs
  after 1.6.
- `internal/adapters/server/mcpapi/extended_tools_test.go:98` —
  `Kind: domain.KindID("go-project")` inside a `domain.Project{...}` struct
  literal. Field removed by 1.6; test in 1.12 Paths (runs after 1.6).
- `internal/domain/domain_test.go` — has `Project{...}` construction paths
  that may include the `Kind` field (1.10 owns test updates; runs after 1.6
  and 1.9, and `internal/domain` package shares compile with 1.6/1.8/1.9).

The common shape is: unit 1.6 removes a surface; the cleanup of test
consumers lives in 1.10 / 1.11 / 1.12 / 1.13; `blocked_by` orders those
downstream of 1.6; between the 1.6 commit and the cleanup-unit commits,
`mage ci` fails.

This is the Round 3 F3 pattern a second time. Round 3's F3 fix was to add
an explicit `mage build` waiver to unit 1.4 and shift the compile-restoration
burden to unit 1.5. Unit 1.6 has no equivalent waiver.

**Evidence.**

- `internal/domain/project.go:16` — `Kind KindID` field declaration.
- `internal/domain/project.go:60` — `Kind: DefaultProjectKind,` struct-literal entry.
- `internal/domain/project.go:79-88` — `SetKind(kind KindID, now time.Time) error` method.
- `internal/adapters/storage/sqlite/repo_test.go:2368-2381` — test stanza using `SetKind` and `loadedProject.Kind`.
- `internal/tui/model_test.go:15199-15207` — test stanza reading `svc.lastCreateProject.Kind`.
- `internal/adapters/server/mcpapi/extended_tools_test.go:98` — `domain.Project{...Kind: domain.KindID(...)}` literal.
- PLAN.md unit 1.6 "Paths" — sqlite/tui tests not listed.
- PLAN.md unit 1.6 "Acceptance" — `mage ci` succeeds.
- PLAN.md unit 1.4 "Acceptance" — the precedent waiver bullet: "`mage build`
  and `mage ci` are waived for this unit only."
- PLAN.md unit 1.12 `Blocked by: 1.3, 1.5, 1.6, 1.7`.
- PLAN.md unit 1.13 `Blocked by: 1.5, 1.6`.
- PLAN.md unit 1.11 `Blocked by: 1.2, 1.5, 1.6`.

**Severity.** Blocking. Unit 1.6 as written cannot satisfy its own
acceptance criteria; the builder will hit a compile failure on `mage ci`
and either be stuck or be forced to improvise Paths not in the unit's
contract. Round 3 triage set the precedent that compile-broken windows
need explicit waivers, not implicit tolerances.

**Mitigation (two clean options, either is fine — dev picks).**

- **Option A — extend unit 1.6 Paths to subsume all direct-consumer test
  sites.** Add `internal/adapters/storage/sqlite/repo_test.go`,
  `internal/tui/model_test.go`, `internal/adapters/server/mcpapi/extended_tools_test.go`,
  `internal/domain/domain_test.go` (any sites that reference
  `project.Kind` / `Project{Kind:` / `SetKind`). Unit 1.6 then also strips
  those test stanzas inline, and `mage ci` genuinely passes at 1.6
  commit. This grows 1.6 considerably but gives it a clean gate.
- **Option B — mirror the 1.4 waiver.** Add a bullet to unit 1.6
  Acceptance: "`mage build` and `mage ci` are waived for this unit.
  Workspace-compile restoration lands in units 1.11/1.12/1.13 which
  collectively update the test surface." Per-unit `mage test-pkg` for
  packages whose test files 1.6 *does* touch still runs. Unit 1.15
  remains the drop-end gate that proves the whole chain restores.

Dev previously preferred option (b) for unit 1.4 ("*option b is the same
pattern as the workitem→actionitem rename*"). Option B is a single-line
edit; option A is a rewrite.

---

## F5 coherence — Bullet for `KindDefinition.Template` / `KindTemplate` classification squares with unit 1.3 baked-empty + unit 1.4 sentinel preservation + unit 1.11 `AutoCreateChildren` guidance

**Attack attempted, no counterexample found.**

The Round 3 F4 fix added a 5th bullet to the Scope-section orphan-via-collapse
classification covering `KindDefinition.Template` / `KindTemplate` /
`KindTemplateChildSpec` / `validateKindTemplateExpansion` / `normalizeKindTemplate`
(domain) + the `ErrInvalidKindTemplate` sentinel, all marked "naturally
unreachable" post-collapse because `kind_catalog` bakes empty
`auto_create_children`.

I attacked this from three angles and confirmed consistency:

1. **Unit 1.4 preserves `ErrInvalidKindTemplate`.** The Acceptance bullet
   says `rg 'ErrInvalidKindTemplate' internal/domain/errors.go` returns 1.
   The sentinel stays because `kind.go:281 / :288 / :296 / :302` in
   `normalizeKindTemplate` format-wrap it. The machinery stays as dead
   code; the sentinel has valid in-file callers. Consistent.

2. **Unit 1.3 bakes empty auto_create_children.** There is no separate
   column for `auto_create_children` — it serializes inside
   `template_json TEXT NOT NULL DEFAULT '{}'` at `repo.go:323`. When unit
   1.3 bakes the two surviving rows with an empty `'{}'` template blob,
   the materialized `KindDefinition.Template.AutoCreateChildren` is an
   empty slice. Consistent with the F5 classification.

3. **Unit 1.11 acceptance handles `AutoCreateChildren` tests.** The
   triage-added bullet in 1.11 acceptance says app-layer test cases that
   assert non-empty `AutoCreateChildren` are "either rewritten to assert
   empty (matching the F5 `KindTemplate` classification...) or deleted if
   purely template-library-coupled."

   `internal/app/kind_capability_test.go` lines `425, 509, 591, 697` all
   construct `KindDefinition` with non-empty `AutoCreateChildren` as
   direct-input test fixtures — these test the `validateKindTemplateExpansion`
   machinery with arbitrary input, not what's loaded from `kind_catalog`.
   The machinery is classified "naturally unreachable post-collapse" but
   the code stays, so direct-input tests can stay too. The 1.11 bullet is
   correctly permissive ("rewritten OR deleted").

   `internal/app/template_library_test.go:144-235, 2188-2200+` also
   construct non-empty — but this entire file is deleted by unit 1.5.

   `internal/domain/kind_capability_test.go:18-24, :52-53` constructs
   non-empty `AutoCreateChildren` and asserts `len == 1`. This file is in
   the `internal/domain` package; unit 1.10's Paths explicitly list only
   `domain_test.go` + `attention_level_test.go` — so
   `kind_capability_test.go` is untouched. That's consistent with the F5
   classification (machinery stays, tests of the machinery stay).

No bullet silently undoes another bullet. **Refuted (no counterexample).**

---

## F2 schema-rebuild — unit 1.3 CREATE TABLE strip coordinates cleanly with unit 1.14 SQLite table-rebuild pattern

**Attack attempted, no counterexample found.**

SQLite does not support `ALTER TABLE ... DROP COLUMN` before version 3.35;
unit 1.14's Scope note "SQLite equivalent: `CREATE TABLE projects_new` + copy
+ drop + rename" is the table-rebuild workaround.

The attack hypothesis: the two paths diverge in end-state because they
touch different DB surfaces. Walk-through refutes it:

- Unit 1.3 operates on the **Go application's schema migration code**
  (`internal/adapters/storage/sqlite/repo.go`'s `migrate()` function). After
  1.3, a fresh DB opened by the binary never materializes `projects.kind`
  — neither the `CREATE TABLE IF NOT EXISTS projects (...)` at `:147-157`
  nor the `ALTER TABLE projects ADD COLUMN kind` at `:588` carries the
  column.
- Unit 1.14 operates on the **dev-run-once** `scripts/drops-rewrite.sql`
  which takes an existing DB (the dev's `~/.tillsyn/tillsyn.db`, which
  *does* currently have `projects.kind`) and rebuilds the `projects`
  table without the column.

The two schemas are independent at different moments:
- Pre-1.14-run dev DB: has `kind`. (Populated by the old ALTER migration
  that 1.3 is deleting.)
- Post-1.14-run dev DB: no `kind`. (Rebuilt by the SQL script.)
- New CI/dev-first-run DB (post-1.3 Go binary): never has `kind`. (Go
  migration never adds it.)

The end-state invariant "`pragma_table_info('projects') WHERE name = 'kind'`
returns 0" holds for both paths, as asserted by both unit 1.3's test
(`TestRepositoryFreshOpenProjectsSchema`, added by 1.3 acceptance bullet)
and unit 1.14's SQL assertion block. No divergence.

Ordering: 1.3 ships first (Go code), 1.14 ships last (SQL script, blocked
on every code unit). Dev runs 1.14 once against the dev DB at drop end.
After drop end, any newly opened DB uses 1.3's Go path. Both yield the
same end-state. **Refuted.**

---

## F3 waiver creep — unit 1.4's `mage build` / `mage ci` waiver leaking into downstream units

**Attack attempted, no counterexample found (modulo F1).**

Unit 1.4's triage-added waiver bullet explicitly says "This unit only" and
names unit 1.5 as carrying the compile-restoration burden. Unit 1.5's
acceptance says "`mage ci` succeeds from `drop/1.75/`. **This unit carries
the workspace-compile-restoration burden** — the 1.4 waiver expects 1.5 to
re-green the workspace via this check."

The waiver is correctly scoped — 1.5 restores compile; 1.6 / 1.7 / 1.8 / 1.9
all inherit a compiled workspace at their start, and each unit's Acceptance
runs a fresh per-unit verification.

**However, F1 reveals a parallel, unmitigated compile-broken window
between 1.6 and 1.11/1.12/1.13.** That is F1, not waiver creep of 1.4.
The 1.4 waiver itself is correctly bounded. **Refuted on 1.4 waiver
specifically; F1 captures the 1.6 parallel.**

---

## F4 DAG acyclicity under new `1.12 → 1.6` edge

**Attack attempted, no counterexample found.**

Rebuilt full adjacency after triage:

```
1.1 → ∅
1.2 → {1.1}
1.3 → {1.1, 1.2}
1.4 → {1.1}
1.5 → {1.1, 1.2, 1.3, 1.4}
1.6 → {1.1, 1.2, 1.3, 1.4, 1.5}
1.7 → {1.1, 1.2, 1.3, 1.5}
1.8 → {1.1, 1.4, 1.6}
1.9 → {1.1, 1.4, 1.6, 1.8}
1.10 → {1.1, 1.4, 1.6, 1.8, 1.9}
1.11 → {1.2, 1.5, 1.6}
1.12 → {1.3, 1.5, 1.6, 1.7}
1.13 → {1.5, 1.6}
1.14 → {1.1..1.13}
1.15 → {1.14}
```

Every edge points from a higher-indexed unit to a lower-indexed one.
Topological: `1.1, 1.2, 1.3, 1.4, 1.5, 1.6, 1.7, 1.8, 1.9, 1.10, 1.11,
1.12, 1.13, 1.14, 1.15`. **Acyclic. Refuted.**

---

## F5 file-share and package-share blockers under fresh walk

**Attack attempted, no counterexample found.**

Re-scanned every unit's `Paths` + `Packages` with the rule "sibling units
sharing a file OR a package MUST have an explicit `blocked_by`":

- `internal/domain` package touched by 1.1, 1.4, 1.6, 1.8, 1.9, 1.10 →
  linear chain `1.1 → 1.4 → 1.6 → 1.8 → 1.9 → 1.10` holds via the
  `blocked_by` graph.
- `internal/app` package touched by 1.1, 1.2, 1.5, 1.6, 1.11 → chain
  holds (`1.1 → 1.2 → 1.5 → 1.6 → 1.11`).
- `internal/adapters/storage/sqlite` package touched by 1.1, 1.3, 1.5,
  1.7, 1.12 → chain holds (`1.1 → 1.3 → 1.5 → 1.7 → 1.12`).
- `internal/adapters/server/mcpapi` touched by 1.1, 1.5, 1.6, 1.12 →
  chain holds (`1.1 → 1.5 → 1.6 → 1.12`). Round 3 triage added the
  `1.12 → 1.6` edge that closes this loop.
- `internal/adapters/server/common` touched by 1.1, 1.5, 1.12 → chain
  (`1.1 → 1.5 → 1.12`).
- `internal/adapters/server/httpapi` touched by 1.1, 1.5, 1.12 → chain.
- `internal/tui` touched by 1.1, 1.5, 1.6, 1.13 → chain (`1.1 → 1.5 →
  1.6 → 1.13`).
- `cmd/till` touched by 1.1, 1.5, 1.6, 1.13 → chain (`1.1 → 1.5 → 1.6
  → 1.13`).

File-level overlaps within a single package are subsumed by the package-level
chain. No unordered sibling pair. **Refuted.**

---

## F6 YAGNI pressure on triage-expanded units

**Attack attempted, no counterexample found.**

- Unit 1.3 (triage expanded to include CREATE TABLE strip + two baked
  INSERTs + `mergeKindAppliesTo` / `kindAppliesToEqual` cleanup). Each
  piece is load-bearing for the collapse: CREATE-TABLE strip closes the
  fresh-DB hole F2 found; baked INSERTs replace `seedDefaultKindCatalog`;
  the two helpers die with the seeder. Not YAGNI.
- Unit 1.4 (triage added a compile-waiver acceptance bullet + narrowed
  error-sentinel check to preserve `ErrInvalidKindTemplate`). Both edits
  are defensive constraints the builder needs to know about. Not YAGNI.
- Unit 1.6 (triage added the `SetKind` deletion + `NewProject`
  struct-literal cleanup + sitewide `Project{Kind:}` audit). These are
  necessary corollaries of dropping the `Kind` field. Not YAGNI.
  (Separate finding F1 attacks scope *under*-coverage, not YAGNI.)
- Unit 1.11 (triage added `AutoCreateChildren` test-case bullet). Load-bearing
  for the F5 5th-bullet classification to actually land in tests. Not
  YAGNI.
- Unit 1.15 (triage added quoted-DDL `rg` guard). Closes F2 via the
  end-state invariant. Not YAGNI.

**Refuted.**

---

## O1 (non-blocking observation) — `Project{[^}]*Kind` regex in unit 1.6 acceptance is same-line only; real struct literals span lines

Unit 1.6 acceptance bullet 1 says:

```
rg 'project\.Kind|projects\.kind|Project\{[^}]*Kind' drop/1.75/ --glob='!drops/**' ...
```

The third alternative `Project\{[^}]*Kind` only matches same-line content
because `.` does not cross newlines without `-U`. Real `domain.Project{...}`
literals in the codebase span multiple lines:

```go
projects := []domain.Project{
    {
        ID:          "p2",
        Name:        "Beta",
        Kind:        domain.KindID("project"),
```

The acceptance rg would not match this pattern. The unit 1.6 Scope prose
says "audit every `Project{...}` struct literal sitewide (including tests)
to remove `Kind:` field references", so the **intent** catches it, but the
automated `rg` check underperforms against the intent.

**Not blocking** because: (a) deleting the `Kind` field from `type Project`
makes any remaining `Kind:` field in a `Project` struct literal a *compile
error* — `mage ci` catches what `rg` misses, (b) `rg 'project\.Kind'`
catches the `Project.Kind` readback form, and (c) the Scope prose is
clear.

**Mitigation.** Swap the regex to one that runs on every `Kind:\s` field
occurrence inside a Go file, or add `-U --multiline-dotall` to the `rg`
invocation. Low-priority editorial.

---

## O2 (non-blocking observation) — unit 1.4 waiver could name the downstream restorer unit explicitly

Unit 1.4 Acceptance bullet says "Per-unit `mage build` gate is deferred to
1.5" but does not cross-reference unit 1.5's mirror bullet ("carries the
workspace-compile-restoration burden"). The two sit in separate units and
only connect via prose. If a future refactor reorders units, the waiver
could become orphaned.

**Not blocking** — both halves exist, just not cross-linked.

**Mitigation.** Add "(see unit 1.5 Acceptance)" to 1.4's waiver bullet.
Low-priority editorial.

---

## Verdict

**FAIL.** One blocking counterexample (F1) — unit 1.6's deletion of
`type Project.Kind` and `SetKind` breaks `mage ci` because test-file
consumers in `internal/adapters/storage/sqlite/repo_test.go:2368-2381`,
`internal/tui/model_test.go:15199-15207`, and
`internal/adapters/server/mcpapi/extended_tools_test.go:98` remain in
1.11 / 1.12 / 1.13 Paths (ordered after 1.6) rather than 1.6's own Paths.

This is the Round 3 F3 pattern recurring in a different unit — the fix
there was an explicit per-unit compile waiver (option b). The same fix
applies here: either extend 1.6's Paths to subsume the direct-consumer
test sites (option A), or mirror the 1.4 waiver on 1.6 and shift the
compile-restoration burden to 1.11 / 1.12 / 1.13 collectively (option B).

Both options are single-edit fixes. Dev picks; option B matches the Round
3 precedent and keeps unit sizes balanced.

The other six hunts (DAG acyclicity, F2 schema coordination, F3 waiver
creep, F5 5th-bullet coherence, file-share blockers, YAGNI) all refuted.
Two non-blocking observations (O1 regex shape, O2 cross-link) are
editorial and do not affect verdict.

Return to planner for unit 1.6 Acceptance refinement, then re-run plan-QA
Round 5.
