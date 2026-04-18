---
round: 3
verdict: FAIL
---

# Plan QA Falsification — Round 3

Round 3 attacks the planner's 15-unit decomposition (units 1.1-1.15) under `PLAN.md` §Planner against the Round 2 F1-F11 closure claims and against new counterexamples constructed by unit-by-unit inspection.

Result: two blocking counterexamples (F1, F2) plus five non-blocking quality issues. The plan is close to ship-ready but F1 (missing package-level blocker) and F2 (`projects.kind` hardcoded in CREATE TABLE DDL) must be fixed before Phase 4 starts.

## F1 — Missing package-level blocker between 1.12 and 1.6 (`internal/adapters/server/mcpapi`)

**Counterexample.** Unit 1.12 (adapter + MCP test updates) lists `internal/adapters/server/mcpapi` in its `Packages` line. Unit 1.6 (drop `projects.kind` column) also lists `internal/adapters/server/mcpapi` in its `Packages` line and edits `internal/adapters/server/mcpapi/instructions_explainer.go` to strip the `project.Kind` readback. Unit 1.12 is `Blocked by: 1.3, 1.5, 1.7` — it is NOT blocked by 1.6. Both units are therefore eligible to run in parallel after 1.5+1.7 clear, and both share the mcpapi package compile. The scenario that breaks: 1.6 lands partial edits to `instructions_explainer.go` while 1.12 concurrently runs `mage test-pkg ./internal/adapters/server/mcpapi`, observing a half-migrated file with dangling `project.Kind` references that haven't yet been stripped but with test fixtures that no longer provide a `Kind` value. This is exactly the failure mode `CLAUDE.md` §"Blocker Semantics" targets: *"sibling build-tasks sharing a file in `paths` OR a package in `packages` MUST have an explicit `blocked_by` between them."*

**Evidence.**
- `PLAN.md` unit 1.6 `Packages:` line (plan.md:130): `internal/domain, internal/app, internal/adapters/server/mcpapi, internal/tui, cmd/till`.
- `PLAN.md` unit 1.12 `Packages:` line (plan.md:207): `internal/adapters/storage/sqlite, internal/adapters/server/mcpapi, internal/adapters/server/httpapi, internal/adapters/server/common`.
- `PLAN.md` unit 1.12 `Blocked by:` line (plan.md:208): `1.3, 1.5, 1.7` — missing 1.6.
- `CLAUDE.md` (bare root) §"Blocker Semantics" mandates `blocked_by` between same-package siblings.
- `PLAN.md` note at line 274 explicitly claims the planner section "serializes these five via a linear `blocked_by` chain" for `internal/domain` and `internal/app`, but no analogous claim covers `internal/adapters/server/mcpapi`. The chain is incomplete.

**Severity.** Blocking.

**Mitigation.** Add `1.6` to unit 1.12's `Blocked by:` line. Recommended full line: `Blocked by: 1.3, 1.5, 1.6, 1.7`. While on that line, audit 1.12 against all other units touching its packages — `internal/adapters/server/common` is also in 1.5's packages (already blocked_by 1.5, good); no other cross-package gap found.

## F2 — `projects.kind` column hardcoded in `CREATE TABLE` DDL, not stripped by any Go unit

**Counterexample.** `internal/adapters/storage/sqlite/repo.go:152` hardcodes `kind TEXT NOT NULL DEFAULT 'project'` inside the `CREATE TABLE IF NOT EXISTS projects (...)` block at repo.go:147-157. Unit 1.3's Scope targets the `ALTER TABLE projects ADD COLUMN kind` statement at `:588` and deletes that one line, but says nothing about stripping the column from the CREATE. Unit 1.6 strips Go-level `project.Kind` readers (`type Project`, `SetKind`, MCP handlers, TUI readbacks) but does not touch SQL DDL. Unit 1.14 (`drops-rewrite.sql` rewrite) does `ALTER TABLE projects DROP COLUMN kind` against the dev's existing DB, but that script runs **once**, only against `~/.tillsyn/tillsyn.db`.

The failure scenario: every fresh DB opened after the drop ships — test fixtures in `repo_test.go`, CI integration-test DBs, new-user first-run DBs — executes the `migrate()` function which includes the unchanged CREATE TABLE DDL and re-creates a `projects` table WITH a `kind` column. Unit 1.14's assertion `SELECT COUNT(*) FROM pragma_table_info('projects') WHERE name = 'kind'` returns 0 is violated for every fresh DB in the post-merge world. The drop's stated end-state ("`projects` schema carries no `kind` column") is not reached.

Secondary effect: unit 1.15's `rg` sweep does not catch this because the literal regex `projects\.kind|project\.Kind` does not match `kind TEXT NOT NULL DEFAULT 'project'` inside a quoted DDL string. The sweep has a false negative.

**Evidence.**
- `internal/adapters/storage/sqlite/repo.go:147-157` — CREATE TABLE projects hardcodes the `kind` column.
- `PLAN.md` unit 1.3 Scope (plan.md:97-98) mentions deleting ALTER at `:588` but not the CREATE TABLE column.
- `PLAN.md` unit 1.3 acceptance (plan.md:94-95) checks only the ALTER's absence via `rg "ALTER TABLE projects ADD COLUMN kind"`.
- `PLAN.md` unit 1.6 Paths (plan.md:129) list no file from `internal/adapters/storage/sqlite`.
- `PLAN.md` unit 1.14 acceptance invariant (plan.md:240) holds only for the dev DB after script run, not for fresh CI DBs.
- `PLAN.md` unit 1.15 end-state regex (plan.md:256) `projects\.kind|project\.Kind` does not match quoted-DDL substring.

**Severity.** Blocking. End-state schema contract is violated for every non-dev DB opened after the drop ships. CI tests that introspect `pragma_table_info('projects')` would fail.

**Mitigation.** Extend unit 1.3's Scope and acceptance:
- Scope: add *"delete the `kind TEXT NOT NULL DEFAULT 'project'` column from the `CREATE TABLE IF NOT EXISTS projects` block at `repo.go:152` (inside `migrate()` at `:144`)"*.
- Acceptance: add `rg "kind TEXT.*DEFAULT 'project'" drop/1.75/internal/adapters/storage/sqlite/` returns 0 matches, and add a test `TestRepositoryFreshOpenProjectsSchema` asserting `pragma_table_info('projects')` does not include `kind`.
Alternative: move this strip to unit 1.6 if it needs to land after the `projects.kind` Go refs clear — but the CREATE TABLE column is not referenced by Go code (column presence, not name read), so 1.3 is the natural home.

## F3 — Workspace-wide compile broken between units 1.4 and 1.5 contradicts per-unit verification workflow

**Counterexample.** Unit 1.4 deletes `internal/domain/template_library.go`, `template_reapply.go`, `builtin_template_library.go` and strips template-library error sentinels from `errors.go`. Downstream packages — `internal/app/template_library.go`, `template_contract.go`, `snapshot.go`, `internal/adapters/storage/sqlite/repo.go`, every mcpapi/common file holding template references — still import the deleted domain types. The workspace does not compile between the 1.4 commit and the 1.5 commit. Unit 1.4's Scope text explicitly acknowledges this: *"downstream packages won't compile until 1.5 runs"*.

`drops/WORKFLOW.md` Phase 6 says *"Per-unit verification (during Phase 5, before declaring a unit pass): builder runs `mage build` + `mage test` for the touched packages."* Magefile.go:103 defines `Build` as `go build ./cmd/till` which transitively compiles the entire workspace; it does not accept a package flag. The builder cannot honestly declare unit 1.4 "passes" under this workflow phrasing if `mage build` is the gate, because `mage build` will fail immediately after 1.4's commit.

This is not a code-correctness bug; unit 1.4's own acceptance criteria deliberately drops `mage build` and only requires `mage test-pkg ./internal/domain`. But the drop-level workflow in `WORKFLOW.md` §Phase 6 overrides per-unit acceptance unless explicitly scoped. A QA agent running Phase 5 against unit 1.4 following WORKFLOW.md will produce a failing verdict, blocking the drop indefinitely at Phase 5.

**Evidence.**
- `PLAN.md` unit 1.4 Scope (plan.md:111): *"downstream packages won't compile until 1.5 runs"*.
- `PLAN.md` unit 1.4 acceptance (plan.md:106-109): lists only `mage test-pkg ./internal/domain`, not `mage build`.
- `drops/WORKFLOW.md` §Phase 6 line 152: *"builder runs `mage build` + `mage test` for the touched packages"*.
- `magefile.go:103-106` — `mage Build` is workspace-wide, no per-package variant.

**Severity.** Non-blocking for the drop's correctness — the compile-broken window is internal to a planner-controlled ordering. Non-blocking for the closeout gate because drop-end 1.15 runs workspace `mage ci`. But blocking for the workflow semantics unless the orch explicitly waives `mage build` for 1.4.

**Mitigation.** Two options:
1. Add a one-liner to unit 1.4's Scope stating explicitly that the per-unit `mage build` gate is waived for this unit and that compile restoration is deferred to 1.5's acceptance (which does include workspace `mage ci`). Restate acceptance in 1.5 to carry the full compile-restore burden.
2. Merge units 1.4 and 1.5 into a single compile-atomic unit. Ugly, but eliminates the window. Not recommended given 1.5's already-huge surface.

Option 1 is the minimal fix. Recommended language to append to 1.4's Scope: *"The workspace is compile-broken between this unit's commit and 1.5's commit. Per-unit `mage build` is waived for this unit only; the next package-wide compile gate is 1.5's `mage ci` requirement."*

## F4 — `KindDefinition.Template` / `KindTemplate` machinery not in F5 deferred classification

**Counterexample.** `internal/domain/kind.go:57-63` declares `type KindTemplate struct` with `AutoCreateChildren`, `CompletionChecklist`, `ProjectMetadataDefaults`, `ActionItemMetadataDefaults` fields. `internal/domain/kind.go:73` includes `Template KindTemplate` on `KindDefinition`. `internal/app/kind_capability.go:977-1010` implements `validateKindTemplateExpansion` which recurses through `kind.Template.AutoCreateChildren`. Post-collapse, kind_catalog only carries `project` and `actionItem` rows. If unit 1.3 bakes these rows with empty `auto_create_children: []`, this entire template-expansion path becomes unreachable — orphan code, same shape as the F5 deferred sites.

Neither the original F5 deferral list (`KindAppliesTo`, `WorkKind` non-actionItem, `capabilityScopeTypeForActionItem`, `AuthRequestPathKind`) nor any unit in the plan covers this machinery. It is neither deleted nor explicitly classified as deferred-orphan. Builders will not know whether to touch it.

**Evidence.**
- `internal/domain/kind.go:57-63` — `type KindTemplate struct`.
- `internal/domain/kind.go:73` — `Template KindTemplate` on `KindDefinition`.
- `internal/app/kind_capability.go:977-1010` — `validateKindTemplateExpansion`.
- `internal/app/kind_capability_test.go:425, :509, :591, :697` — tests construct `AutoCreateChildren: []domain.KindTemplateChildSpec{...}`. Unit 1.11 covers this test file but doesn't specify whether these test cases become dead (delete) or stay with baked-empty assertion.
- `PLAN.md` Out-of-scope block (plan.md:48-52) — F5 classification does not include `KindTemplate`.

**Severity.** Non-blocking but needs explicit classification. Builder ambiguity could lead to either spurious deletion breaking tests or spurious retention leaving dead tests asserting non-empty templates.

**Mitigation.** Add a fifth bullet to the F5 deferred classification block in `PLAN.md` §Scope "Out-of-scope":
*"`KindDefinition.Template` / `KindTemplate` / `KindTemplateChildSpec` / `validateKindTemplateExpansion` at `internal/domain/kind.go:57-73` + `internal/app/kind_capability.go:977-1010`: **naturally unreachable** post-collapse — `kind_catalog` bakes empty `auto_create_children` for both surviving rows. Retention intentional; refinement drop deletes."*
And in unit 1.11 Scope, state that `kind_capability_test.go` test cases asserting non-empty `AutoCreateChildren` are rewritten to assert empty or deleted.

## F5 — Unit 1.6 ambiguous on `SetKind` method and `NewProject` default assignment

**Counterexample.** Unit 1.6's Scope (plan.md:129) says *"strip `Kind` field from `type Project` at `:16`, delete the assignment at `:85`"*. Line 85 is inside `SetKind(kind KindID, now time.Time) error` at `project.go:80-88`, which is a method whose entire purpose is to set `p.Kind`. Deleting only `p.Kind = kind` leaves `SetKind(kind KindID, ...)` with an unused parameter but valid Go (compiles). The method becomes dead but not a compile error. Unit 1.6 also does not mention line 60 `Kind: DefaultProjectKind,` inside `NewProject` — which WOULD be a compile error if the `Kind` field is gone (struct-literal field reference).

Builder interpreting literally deletes line 85 only, leaves `SetKind` dead and `NewProject`'s line 60 intact — which produces `./internal/domain/project.go:60: unknown field Kind in struct literal of type Project`. Compile break.

Builder interpreting charitably deletes the whole `SetKind` method + updates `NewProject` + audits every struct-literal use of `Project{Kind: ...}` across the codebase. Different builders will interpret differently.

**Evidence.**
- `internal/domain/project.go:11-21` — `type Project struct { ... Kind KindID ... }`.
- `internal/domain/project.go:55-65` — `NewProject(...)` returns `Project{...Kind: DefaultProjectKind, ...}` at line 60.
- `internal/domain/project.go:80-88` — `SetKind` method.
- `PLAN.md` unit 1.6 Scope (plan.md:129) — only mentions `:16` and `:85`.

**Severity.** Non-blocking if caught by builder common sense or `mage ci` gate; blocking if a literalist builder lands a mid-compile state.

**Mitigation.** Rewrite unit 1.6 Scope bullet for project.go to: *"strip `Kind KindID` field from `type Project` at `:16`, delete the `SetKind` method entirely at `:79-88`, delete the `Kind: DefaultProjectKind,` line from `NewProject`'s struct literal at `:60`, and audit every `Project{...}` struct-literal construction sitewide (including tests) to remove `Kind:` field references"*.

## F6 — Unit 1.4 error-sentinel strip scope ambiguous on `ErrInvalidKindTemplate`

**Counterexample.** Unit 1.4 Paths include `internal/domain/errors.go` with Scope "remove template-library error sentinels". `errors.go` lines 25-33 contain ten `Err*` sentinels matching broadly "template"-related names: `ErrInvalidKindTemplate`, `ErrInvalidTemplateLibrary`, `ErrInvalidTemplateLibraryScope`, `ErrInvalidTemplateStatus`, `ErrInvalidTemplateActorKind`, `ErrInvalidTemplateBinding`, `ErrBuiltinTemplateBootstrapRequired`, `ErrTemplateLibraryNotFound`, `ErrNodeContractForbidden`. Of these, `ErrInvalidKindTemplate` is used by `internal/domain/kind.go:262-275`'s `normalizeKindTemplate`, which is part of the surviving `KindDefinition.Template` machinery (see F4). The rest are exclusively template_libraries-related.

Unit 1.4's acceptance check `grep -c 'ErrTemplate' internal/domain/errors.go returns 0` uses the literal substring `ErrTemplate` — this matches `ErrTemplateLibraryNotFound` but does NOT match `ErrInvalidKindTemplate` (no `ErrTemplate` literal substring) or `ErrInvalidTemplateLibrary` (same — no literal `ErrTemplate`). A builder interpreting the acceptance narrowly deletes only `ErrTemplateLibraryNotFound` and leaves every `ErrInvalidTemplate*` intact — but then unit 1.5's deletion of `internal/app/template_library.go` leaves the sentinels with zero references. A builder interpreting broadly ("template-library error sentinels") deletes the whole `Err*Template*` block and would also delete `ErrInvalidKindTemplate`, breaking `kind.go:normalizeKindTemplate`.

**Evidence.**
- `internal/domain/errors.go:25-33` — sentinel block.
- `internal/domain/kind.go:262-275` — `normalizeKindTemplate` uses `ErrInvalidKindTemplate`.
- `PLAN.md` unit 1.4 Scope (plan.md:111) and acceptance `grep -c 'ErrTemplate'`.

**Severity.** Non-blocking (builder should catch). But the acceptance regex is mis-specified for what the Scope prose says.

**Mitigation.** Rewrite unit 1.4 acceptance to a precise list:
*"Acceptance: `rg -F 'ErrTemplateLibraryNotFound' internal/domain/errors.go` returns 0; `rg 'ErrInvalidTemplate(Library|LibraryScope|Status|ActorKind|Binding)' internal/domain/errors.go` returns 0; `rg 'ErrBuiltinTemplateBootstrapRequired|ErrNodeContractForbidden' internal/domain/errors.go` returns 0; `rg 'ErrInvalidKindTemplate' internal/domain/errors.go` returns 1 (preserved for `kind.go:normalizeKindTemplate`)"*.

## F7 — Unit 1.2 edits `template_library_builtin.go`, which is deleted by unit 1.5

**Counterexample.** Unit 1.2 updates callers of `ensureKindCatalogBootstrapped`. There are 9 call sites in `rg` output, two of which are in `internal/app/template_library_builtin.go:29, :79`. Unit 1.5's Paths list `internal/app/template_library_builtin.go` for deletion. Unit 1.2 therefore edits a file destined for deletion — wasted work. Not a correctness bug but a quality issue, and if the builder runs 1.2 edits first then observes 1.5 deletes the file, it can be confused by the double accounting.

Also — if unit 1.2 removes the call sites (replacing with no-ops) so the file continues to compile for the 1.2→1.5 window, and then 1.5 deletes the file wholesale, the intermediate edits were pure waste.

**Evidence.**
- `rg` output above: `internal/app/template_library_builtin.go:29, :79` call `ensureKindCatalogBootstrapped`.
- `PLAN.md` unit 1.5 Paths (plan.md:116) include `internal/app/template_library_builtin.go`.

**Severity.** Non-blocking, efficiency-only.

**Mitigation.** State explicitly in unit 1.2 Scope: *"Callers inside files destined for deletion by unit 1.5 (`internal/app/template_library_builtin.go`, `template_library.go`, `template_contract.go`, `template_reapply.go`) are intentionally left in place; the file deletions in 1.5 moot them."* Remove those files from 1.2's Paths-consideration.

## Verdict

**FAIL.** Two blocking counterexamples: F1 (missing package-level blocker 1.12→1.6 across `internal/adapters/server/mcpapi`) and F2 (`projects.kind` column hardcoded in CREATE TABLE DDL at `repo.go:152`, stripped by no unit, invalidates end-state invariant for every fresh DB). Five non-blocking quality issues (F3 compile window, F4 orphan template machinery, F5 project.go Scope ambiguity, F6 acceptance regex mis-match, F7 wasted edits) should also be addressed but do not block Phase 4 start.

Recommend Phase 3 Discuss+Cleanup loop to address F1 and F2 before declaring plan accepted. F3-F7 can be folded into the same planner brief for efficiency.
