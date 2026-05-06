# Drop 4c.5 — Theme F Planner Output (Template Ergonomics)

**Author:** go-planning-agent (Theme F).
**Authored:** 2026-05-05.
**Scope:** F.1 (auto-discovery) + F.2 (builtin separation) + F.3 (`till.template` MCP tool) + F.5 (extended validation) + F.6 (KindTemplate stub cleanup).
**Out of scope:** F.4 marketplace CLI (Drop 4d-prime); F.7 spawn pipeline (already shipped Drop 4c).
**Total droplets:** 13.
**Repo HEAD evidence:** `7cd84ec` on `main`. Template surfaces read live (no Hylla; post-Drop-4c-merge).

---

## 1. Premises and Evidence

- **F.1 hook today** — `internal/app/service.go:427` `loadProjectTemplate()` returns `(templates.Template{}, false, nil)` per Drop 3.14 deferral; consumed by `bakeProjectKindCatalog` (`service.go:401`) which itself is called from `CreateProjectWithMetadata` (`service.go:346`).
- **Project surface fields** — `internal/domain/project.go:25-49` defines `HyllaArtifactRef`, `RepoBareRoot`, `RepoPrimaryWorktree`, `Language`. Each can be empty post-Drop-4a (zero values are meaningful — "project not yet bootstrapped"). Closed `Language` enum is `"" | "go" | "fe"` (`isValidProjectLanguage`).
- **Templates package shape** — `internal/templates/` ships `Load(io.Reader) (Template, error)` (`load.go:78`), `LoadDefaultTemplate()` (`embed.go:33`), embedded `builtin/default.toml` via `//go:embed builtin/default.toml`, and validators `validateMapKeys`, `validateChildRuleKinds`, `validateChildRuleCycles`, `validateChildRuleReachability` (no-op today, `load.go:400`), `validateGateKinds`, `validateAgentBindingEnvNames`, `validateAgentBindingContext`, `validateAgentBindingToolGating`, `validateTillsyn`.
- **Closed validation chain order** — Load runs version pre-pass → strict decode → 9 validators in the documented order. F.5 additions must slot in deterministically without breaking existing routing on `errors.Is`.
- **Bake snapshot policy (5.B.14)** — per `bakeProjectKindCatalog` doc-comment, edits to `<project_root>/.tillsyn/template.toml` AFTER project creation are ignored. F.1 inherits this contract: walk-and-bake at create time, no live re-read.
- **MCP tool registration pattern** — `internal/adapters/server/mcpapi/extended_tools.go:1673` `registerKindTools` shows the canonical AddTool shape: `mcp.NewTool` with operation enum + bindArguments + switch-on-operation + `toolResultFromError` mapping + `mcp.NewToolResultJSON` envelope.
- **F.6 stub** — `internal/app/kind_capability.go:1002` `mergeActionItemMetadataWithKindTemplate(base, _)` returns `(base, nil)` unconditionally. Single caller: `service.go:897` inside `CreateActionItem`. Doc-comment promises future fold-in.
- **Cascade vocabulary (closed 12-kind enum)** — every kind reachable from `plan` via existing child_rules: `build`/`build-qa-proof`/`build-qa-falsification`/`plan-qa-proof`/`plan-qa-falsification` directly; `research`/`closeout`/`commit`/`refinement`/`discussion`/`human-verify` reachable as legitimate children of `plan` per `KindRule.AllowedChildKinds` (`builtin/default.toml`). F.5's `validateChildRuleReachability` extension must not contradict this — see §3 Notes.

---

## 2. Per-Droplet Decomposition

### Droplet F.1.1 — Wire `loadProjectTemplate` to embedded fallback

**State:** done (round 1)
**Title:** `4c.5.F.1.1 — wire loadProjectTemplate embedded default fallback`.

**Files / paths:**

- `internal/app/service.go` (replace `loadProjectTemplate` body lines ~427-429).
- `internal/app/service_test.go` (new test cases for fallback path).

**Packages:** `github.com/evanmschultz/tillsyn/internal/app`.

**Acceptance criteria:**

1. `loadProjectTemplate` accepts the project as an argument: `loadProjectTemplate(project *domain.Project) (templates.Template, bool, error)`. Caller `bakeProjectKindCatalog` (`service.go:401`) updated to pass `project` through.
2. When invoked with a project whose `RepoPrimaryWorktree` and `RepoBareRoot` are both empty, the function returns the parsed embedded `default.toml` with `ok=true` (NOT `ok=false`). This is the change vs Drop 3.14: empty paths now mean "use embedded default" rather than "skip template binding entirely."
3. Returned `Template` round-trips `templates.Bake` without panic; `Template.SchemaVersion == "v1"`.
4. New test `TestLoadProjectTemplate_EmbeddedFallback` asserts: empty-path project → `ok=true`, baked catalog non-empty.
5. Existing test that asserts the empty-catalog branch (per Drop 3.12 acceptance) is updated or replaced — flagged in the test scenarios below.
6. `mage test-pkg ./internal/app` passes.

**Test scenarios (table-driven):**

- `empty paths → embedded fallback` — input `domain.Project{}` (zero value); expect `ok=true`, `tpl.SchemaVersion == "v1"`, `len(tpl.Kinds) == 12`.
- `whitespace-only paths → embedded fallback` — explicit `RepoPrimaryWorktree: "   "`; treat trim-empty as empty.
- Snapshot-existing test `TestBakeProjectKindCatalog_EmptyCatalog` (if it exists) — update assertion: catalog is now non-empty for embedded-default projects.

**Blocked by:** F.2.1 (default-go.toml rebadge MUST land first; the embedded resolver in F.1.2 picks per-Language, and an in-flight rename would break this droplet's snapshot test).

**Falsification mitigations:**

- F1: "Returning ok=true silently injects a 12-kind catalog into projects that the dev created expecting empty (legacy Drop 2.8 universal-nesting compat)." — Mitigation: F.1.1 adds a one-line release-note comment in `bakeProjectKindCatalog` doc-comment naming the behavior change; downstream callers (`initializeProjectAllowedKinds` at `service.go:352`) already handle non-empty catalogs since Drop 3.14.
- F2: "Embedded TOML parse failure at runtime is a programmer-error panic disguised as a silent fallback." — Mitigation: parse error from `templates.LoadDefaultTemplate()` propagates as `(zero, false, err)` to surface at project-create boundary; test asserts error path.
- F3: "Empty-path zero-value collision with not-yet-set field." — Document the contract: F.1.1 deliberately collapses the two cases (no path declared / explicitly empty) under "use embedded default." Rationale: pre-MVP, no project ships explicit empty-path-meaning-skip semantics.

---

### Droplet F.1.2 — File-system walk: bare root → primary worktree → embedded

**State:** done (round 1)
**Title:** `4c.5.F.1.2 — loadProjectTemplate walks .tillsyn/template.toml candidates`.

**Files / paths:**

- `internal/app/service.go` (extend F.1.1's `loadProjectTemplate` with walk).
- `internal/app/service_test.go` (file-system fixture tests).

**Packages:** `github.com/evanmschultz/tillsyn/internal/app`.

**Acceptance criteria:**

1. Walk order, in priority sequence:
   1. `<project.RepoBareRoot>/.tillsyn/template.toml` (when `RepoBareRoot != ""` AND file exists).
   2. `<project.RepoPrimaryWorktree>/.tillsyn/template.toml` (when `RepoPrimaryWorktree != ""` AND file exists).
   3. Embedded `default.toml` selected by F.1.3's language-aware resolver.
2. Each candidate that exists is opened via `os.Open` and passed to `templates.Load`. The first candidate whose `templates.Load` returns `nil` error WINS. Subsequent candidates are NOT consulted on success.
3. If a candidate file exists but `templates.Load` returns an error, the error PROPAGATES (does NOT fall through to the next candidate). Rationale: silent fall-through hides typos in dev-authored templates.
4. Empty `RepoBareRoot` skips the first lookup entirely (no `os.Stat` on `"./.tillsyn/template.toml"` — the relative-path footgun).
5. New tests `TestLoadProjectTemplate_BareRootWins`, `TestLoadProjectTemplate_PrimaryWorktreeFallback`, `TestLoadProjectTemplate_BareRootSyntaxErrorPropagates`.
6. `mage test-pkg ./internal/app` passes.

**Test scenarios (table-driven):**

- `bare root template wins` — `t.TempDir()` write `<bare>/.tillsyn/template.toml` with valid v1 + a custom STEWARD seed; primary-worktree fixture present with different content; expect bare-root content wins.
- `bare root absent, primary worktree present` — only primary worktree fixture; loads from there.
- `bare root present but malformed` — invalid TOML at bare root; expect error wrapping `templates.ErrUnknownTemplateKey` or pelletier parse error; primary-worktree fallback NOT consulted.
- `both absent → embedded` — no .tillsyn/ on disk; falls through to F.1.3 resolver.
- `relative path safety` — empty `RepoBareRoot` does NOT trigger `os.Stat(".tillsyn/template.toml")`.

**Blocked by:** F.1.1 (signature change must land first); F.1.3 (embedded resolver target).

**Falsification mitigations:**

- F1: "Walk uses `filepath.Join` which silently treats a relative `RepoBareRoot` as relative-to-CWD." — Mitigation: assert `filepath.IsAbs(project.RepoBareRoot)` before joining; reject non-absolute with wrapped error. Domain layer already validates this on project create (`project.go:340`-ish), but adding the runtime guard catches downstream test fixtures + hand-edited DBs.
- F2: "Position-aware error from `templates.Load` may lose origin (which file)." — Mitigation: wrap with `fmt.Errorf("template at %s: %w", candidatePath, err)` so dev sees the offending path.
- F3: "Symlink-loop on `<bare>/.tillsyn/`." — Mitigation: `os.Stat` (NOT `Lstat`) follows symlinks; document the policy in doc-comment, defer aggressive symlink protection to a future refinement.

---

### Droplet F.1.3 — Language-aware embedded resolver

**State:** done (round 1)
**Title:** `4c.5.F.1.3 — embedded resolver picks default-generic vs default-go by Language`.

**Files / paths:**

- `internal/templates/embed.go` (extend resolver with language switch).
- `internal/templates/embed_test.go` (resolver tests).

**Packages:** `github.com/evanmschultz/tillsyn/internal/templates`.

**Acceptance criteria:**

1. New function `LoadDefaultTemplateForLanguage(lang string) (Template, error)`. Closed-enum lang values: `""` (generic), `"go"` (Go-flavored), `"fe"` (FE-flavored — see Q1 resolution in §3 Notes).
2. `lang == ""` → loads `builtin/default-generic.toml`.
3. `lang == "go"` → loads `builtin/default-go.toml`.
4. `lang == "fe"` → returns error `ErrLanguageNotSupported` (closed sentinel) WITH a clear message `"fe template unavailable; defer until FE adopter materializes"`. Per Q1 resolution: defer FE shipment.
5. Unknown lang → error wrapping `ErrLanguageNotSupported` with the offending value.
6. Existing `LoadDefaultTemplate()` is preserved as a thin wrapper: `return LoadDefaultTemplateForLanguage("")`.
7. `internal/app/service.go` `loadProjectTemplate` calls `LoadDefaultTemplateForLanguage(project.Language)` for the embedded path.
8. New tests: `TestLoadDefaultTemplateForLanguage_Generic`, `_Go`, `_FERejected`, `_UnknownRejected`.
9. `mage test-pkg ./internal/templates` and `./internal/app` pass.

**Test scenarios (table-driven):**

- `lang="" → generic` — assert `tpl.SchemaVersion == "v1"`.
- `lang="go" → go` — assert presence of a Go-specific marker in baked output (e.g. agent_bindings reference `go-builder-agent` per F.2.2's content).
- `lang="fe" → ErrLanguageNotSupported`.
- `lang="rust" → ErrLanguageNotSupported wrapping unknown-lang reason`.

**Blocked by:** F.2.1 (default-go.toml present) AND F.2.2 (default-generic.toml present).

**Falsification mitigations:**

- F1: "FE rejection at the resolver leaves dev-FE-projects unable to create at all." — Mitigation: FE rejection bubbles to project-create boundary as a clear error; the dev sees a precise instruction to author `<project_root>/.tillsyn/template.toml`. Pre-MVP, no FE projects exist; post-MVP path is via F.4 marketplace CLI.
- F2: "Embedded resolver bypasses validation chain when re-using cached parsed `Template`." — Mitigation: each call re-runs `Load` (no caching). Embed.FS read overhead is bounded; per-spawn perf is dominated by the bundle materializer.
- F3: "Closed enum drift between `domain.isValidProjectLanguage` and `LoadDefaultTemplateForLanguage`." — Mitigation: F.1.3 doc-comment cross-references the domain validator; future adopters who extend project Language (e.g. add `"rust"`) will see the resolver fail loud and add the new TOML alongside.

---

### Droplet F.2.1 — Rebadge default.toml → default-go.toml

**State:** done (round 1)
**Title:** `4c.5.F.2.1 — rename default.toml to default-go.toml in builtin/`.

**Files / paths:**

- `internal/templates/builtin/default-go.toml` (NEW; rename of `default.toml`).
- `internal/templates/builtin/default.toml` (DELETE).
- `internal/templates/embed.go` (update embed directive to add the new file).
- `internal/templates/embed_test.go` (rename TestLoadDefaultTemplate → TestLoadDefaultTemplateGo or similar).

**Packages:** `github.com/evanmschultz/tillsyn/internal/templates`.

**Acceptance criteria:**

1. File `internal/templates/builtin/default-go.toml` exists with EXACT byte-content of the prior `default.toml` plus a header-comment update naming "Go default" instead of "default". No agent-binding / kind / child-rule changes vs. the prior content.
2. `internal/templates/builtin/default.toml` no longer exists in tree.
3. `internal/templates/embed.go` now embeds `builtin/default-go.toml` (and `builtin/default-generic.toml` from F.2.2) — Update the `//go:embed` directive to use a glob: `//go:embed builtin/*.toml` OR explicit list of two files.
4. `LoadDefaultTemplate()` (preserved API per F.1.3) reads `builtin/default-go.toml` directly when called by the existing thin-wrapper for backward-compat (callers that need the Go default explicitly).
5. Wait — re-think: per F.1.3 acceptance criterion 6, `LoadDefaultTemplate()` is `LoadDefaultTemplateForLanguage("")` which returns the GENERIC template. The two existing callers of `LoadDefaultTemplate()` (`bakeProjectKindCatalog` indirectly, and `seedStewardAnchors` per `service.go:425` doc-comment) must be audited. **Action:** audit + redirect each existing caller to the language-explicit form before the rename can land cleanly. F.2.1 includes that audit pass.
6. `mage ci` passes.

**Test scenarios (table-driven):**

- `embedded go template byte-identical to prior default.toml minus header` — golden-file diff of pre-rename vs post-rename body content.
- `existing TestLoadDefaultTemplate_<existing variants>` — adapt to read from `default-go.toml` source.
- `seedStewardAnchors uses go default` — assert STEWARD seed materialization still produces 6 anchors as before (regression).

**Blocked by:** none (this lands first in Theme F).

**Falsification mitigations:**

- F1: "Caller audit misses an indirect usage of `LoadDefaultTemplate()`." — Mitigation: full `rg LoadDefaultTemplate` audit included in droplet acceptance; every call site's expected language is documented in the audit comment.
- F2: "go:embed pattern globs accidentally pull in non-template files." — Mitigation: explicit-file list in the directive (`//go:embed builtin/default-go.toml builtin/default-generic.toml`) instead of glob.
- F3: "Backward-compat for adopters who shadow `default.toml` via a worktree-shipped fixture." — Mitigation: pre-MVP, no external adopters; this repo is the only consumer. The rename is documented in CLAUDE.md/wiki updates handled via the closeout (out of scope here).

---

### Droplet F.2.2 — Add default-generic.toml (language-agnostic showcase)

**State:** done (round 1)
**Title:** `4c.5.F.2.2 — add default-generic.toml language-agnostic builtin`.

**Files / paths:**

- `internal/templates/builtin/default-generic.toml` (NEW).
- `internal/templates/embed.go` (already covered in F.2.1).
- `internal/templates/embed_test.go` (TestLoadDefaultTemplateForLanguage_Generic).

**Packages:** `github.com/evanmschultz/tillsyn/internal/templates`.

**Acceptance criteria:**

1. File contains valid v1 schema with the closed 12-kind catalog + the same 4 standard child_rules + the same 6 STEWARD seeds as default-go.toml.
2. Agent bindings differ: agent_name fields reference language-agnostic placeholders OR are entirely OMITTED. Per Q1 resolution (defer FE) the safest content is **omit `[agent_bindings]` entirely** — the dispatcher tolerates absent bindings (verified via existing `templates.Bake` semantics; an absent binding for kind X means "no auto-spawn for kind X").
3. Loads cleanly through `templates.Load` (full validation chain).
4. New test `TestLoadDefaultGenericTemplate` asserts: parses, validates, has 12 kinds + 4 child_rules + 6 STEWARD seeds, `len(tpl.AgentBindings) == 0`.
5. `mage test-pkg ./internal/templates` passes.

**Test scenarios (table-driven):**

- `parse + validate` — file loads via `Load`, no error.
- `kind catalog matches` — same `[kinds.X]` as default-go.toml for kind/child-restriction shape.
- `agent bindings absent` — `len(tpl.AgentBindings) == 0`.
- `child_rules count == 4` — `build → build-qa-proof`, `build → build-qa-falsification`, `plan → plan-qa-proof`, `plan → plan-qa-falsification`. The two drop-narrowed entries (drop-level QA twins) MAY be omitted in generic — drop-level cascade is Tillsyn-cascade-specific. Document in doc-comment.

**Blocked by:** F.2.1 (embed directive must already cover both files).

**Falsification mitigations:**

- F1: "Generic template with no bindings is useless — silently no-ops every dispatch." — Mitigation: this is the contract. Adopters explicitly opt into agent bindings via `<project_root>/.tillsyn/template.toml`. Doc-comment names this loud.
- F2: "Absent agent_bindings table fails some validator." — Mitigation: pre-flight test the file through `Load` BEFORE the droplet is "complete." If a validator rejects, switch to `[agent_bindings] = {}` (empty table) which is structurally legal.
- F3: "Project with `Language=""` calling `LoadDefaultTemplateForLanguage("")` gets a useless template — and dispatcher will then silently no-op." — Mitigation: per Q1 resolution, this is acceptable pre-MVP; dispatcher logs a clear warning when invoked against a kind with no binding (dispatcher behavior unchanged by this droplet — out of scope).

---

### Droplet F.2.3 — Self-host `<project_root>/.tillsyn/template.toml` for tillsyn

**State:** done (round 1)
**Title:** `4c.5.F.2.3 — add self-host template at repo root for dogfood`.

**Files / paths:**

- `.tillsyn/template.toml` (NEW at repo root — directory may need to be created).
- `.gitignore` (verify `.tillsyn/spawns/` ignore unchanged; `.tillsyn/template.toml` is NOT ignored — it ships).
- `CLAUDE.md` or `WIKI.md` (no edit; this is left to closeout).

**Packages:** none (TOML + dotfile only). **No Go test changes required.**

**Acceptance criteria:**

1. `.tillsyn/template.toml` exists at repo root and contains valid v1 schema. Content seed = exact copy of `default-go.toml` with two adjustments:
   - Header comment names this as the tillsyn self-host template.
   - `[tillsyn]` table includes `spawn_temp_root = "os_tmp"` (or "project" — pick after confirming current behavior).
2. `mage ci` passes — file is parsed by NO test today, but adding it under tracked sources MUST not break embed-FS or build.
3. The directory is excluded from existing `.gitignore` `.tillsyn/spawns/` rule (which only globs `spawns/`). Verify by `git status` / `git ls-files .tillsyn/`.
4. Drop closeout MD will surface this as a tracked self-host artifact.

**Test scenarios (table-driven):** none — no Go-level tests. Verification is `mage ci` green + `git ls-files .tillsyn/template.toml` shows the file.

**Blocked by:** F.2.1 (default-go.toml content stable to copy from). **NOT blocked by F.1.x** — the self-host file is read by F.1.2's walk only AFTER F.1.2 lands; landing F.2.3 first means the file sits unused until F.1.x activates it. Acceptable.

**Falsification mitigations:**

- F1: "Self-host file shadows embedded default during tests run from repo root, breaking test isolation." — Mitigation: test fixtures use `t.TempDir()` for project paths; production `loadProjectTemplate` is called with explicit project paths, not CWD-walking. The repo-root file is consumed only when a project's `RepoBareRoot` or `RepoPrimaryWorktree` resolves to the tillsyn repo itself.
- F2: "Hand-authored TOML drifts from default-go.toml over time, surprising adopters." — Mitigation: F.2.3 uses BYTE-IDENTICAL copy; future drift is intentional, drop-tracked.
- F3: "`.tillsyn/spawns/` gitignore catches `.tillsyn/template.toml`." — Verified manually pre-droplet: existing rule is `.tillsyn/spawns/`, NOT `.tillsyn/`. Document in droplet acceptance.

---

### Droplet F.2.4 — Caller audit + tests for language-aware template loading

**State:** done (round 1)
**Title:** `4c.5.F.2.4 — audit LoadDefaultTemplate callers + add cross-package tests`.

**Files / paths:**

- `internal/app/service.go` (caller-redirect from `LoadDefaultTemplate()` to `LoadDefaultTemplateForLanguage(project.Language)` everywhere — STEWARD seed path is the main candidate).
- `internal/app/service_test.go` (tests for STEWARD seed materialization on Go vs Generic).
- `internal/templates/embed_test.go` (cross-test that LoadDefaultTemplate() == LoadDefaultTemplateForLanguage("")).

**Packages:** `github.com/evanmschultz/tillsyn/internal/app`, `github.com/evanmschultz/tillsyn/internal/templates`.

**Acceptance criteria:**

1. Every existing reference to `templates.LoadDefaultTemplate()` is reviewed; redirected to `LoadDefaultTemplateForLanguage` with the explicit language argument. The thin wrapper `LoadDefaultTemplate()` is preserved (callers may still reach for it intentionally) but no production Tillsyn code path retains the un-suffixed call.
2. `seedStewardAnchors` (referenced at `service.go:425`) reads the language-specific template — the 6 STEWARD seeds are sourced from the project's Language-bound template, NOT a hard-coded default.
3. New test `TestSeedStewardAnchors_LanguageAware` asserts: project created with `Language=""` materializes STEWARD seeds from generic; `Language="go"` from go.
4. `mage ci` passes.

**Test scenarios (table-driven):**

- `Language="" → generic STEWARD seeds`.
- `Language="go" → go STEWARD seeds`.
- `LoadDefaultTemplate() returns same as LoadDefaultTemplateForLanguage("")`.

**Blocked by:** F.1.3, F.2.1, F.2.2.

**Falsification mitigations:**

- F1: "STEWARD seeds in default-generic.toml differ from default-go.toml — adopters get unexpected anchor names." — Mitigation: per F.2.2 acceptance, BOTH files ship the same 6 STEWARD seeds. Verified in F.2.2's tests.
- F2: "Test fixtures hardcode `LoadDefaultTemplate()` and start failing silently." — Mitigation: `mage test-pkg` runs across both packages; any test-side caller is caught.
- F3: "Caller audit misses a non-Go consumer (e.g. a script or magefile target)." — Mitigation: search includes magefile.go, scripts/, .githooks/ — no non-Go consumer expected post-Drop-3, but the audit confirms.

---

### Droplet F.5.1 — `validateAgentBindingFiles` (warn-only) + `validateRequiredChildRules`

**State:** done (round 1)
**Title:** `4c.5.F.5.1 — agent-binding-files warn + required-child-rules assert`.

**Files / paths:**

- `internal/templates/load.go` (add two validator functions; wire into the validation chain).
- `internal/templates/load_test.go` (table-driven test cases for both).

**Packages:** `github.com/evanmschultz/tillsyn/internal/templates`.

**Acceptance criteria:**

1. New `validateAgentBindingFiles(tpl Template, logger func(string))` runs after `validateAgentBindingToolGating` and before `validateTillsyn`. For each `AgentBinding.AgentName` referenced, checks `~/.claude/agents/<name>.md` exists. **WARN-ONLY** per Q2 resolution (§3 Notes): non-existence emits a warning via the supplied logger; never returns an error.
2. The logger is plumbed through `Load` via a new optional parameter or via `LoadWithOptions` (preferred — keeps `Load(io.Reader)` compatible). Recommended shape: `LoadOptions struct { WarnLogger func(string) }`. Adopters who want strict-fail for missing agent files can wrap `LoadOptions.WarnLogger` to escalate to error.
3. New `validateRequiredChildRules(tpl Template) error` runs after `validateChildRuleCycles` and before `validateChildRuleReachability`. Asserts that every parent kind in the closed REQUIRED-CHILD-RULES set has the matching child rule entries:
   - `kind=plan` → must have `plan-qa-proof` AND `plan-qa-falsification` child rules.
   - `kind=build` → must have `build-qa-proof` AND `build-qa-falsification` child rules.
4. Missing required child rule returns new sentinel `ErrMissingRequiredChildRule` with a precise message naming the parent kind + the missing child kind.
5. New tests: `TestValidateAgentBindingFiles_WarnOnMissing`, `TestValidateAgentBindingFiles_NoWarnOnPresent`, `TestValidateRequiredChildRules_PlanMissingProofRejected`, `TestValidateRequiredChildRules_BuildMissingFalsificationRejected`.
6. `mage test-pkg ./internal/templates` passes.

**Test scenarios (table-driven):**

- `agent file present (mocked filesystem stub) → no warn`.
- `agent file absent → exactly one warn line per missing binding, log message names binding's kind + agent_name`.
- `plan with both QA child_rules → ok`.
- `plan with only proof child_rule → ErrMissingRequiredChildRule wrapping "plan-qa-falsification"`.
- `build with no QA child_rules → ErrMissingRequiredChildRule named first`.
- `templates without `kind=plan` row in `[kinds]` → no required-child-rules check applies (empty input is no-op).

**Blocked by:** F.5 hooks ride on existing load.go; depends only on Theme F's earlier droplets via the embedded test fixtures (use `default-go.toml` as the green-path baseline). Conceptually F.5.1 has NO Theme F blocked_by since it edits a separate file from F.2.x. **However:** F.2.1 renames `default.toml` → `default-go.toml`, so F.5.1's tests that load the embedded default must use the new path. Mark `blocked_by: F.2.1` to serialize.

**Falsification mitigations:**

- F1: "Filesystem check is non-deterministic — different dev machines see different warn output, breaking test reproducibility." — Mitigation: validator accepts a stat-fn injection (`func(path string) bool`) defaulting to `os.Stat`; tests pass an in-memory stub. Production callers use the default.
- F2: "Required-child-rules assertion fires on templates without the parent kind defined at all (e.g. an extreme adopter strips `kind=plan`)." — Mitigation: F.5.1's check is conditional on the parent kind being present in `[kinds]`. Absent → skip.
- F3: "WarnLogger nil pointer — Load(io.Reader) overload doesn't supply one and the validator panics." — Mitigation: validator nil-checks the logger; nil = silent. Existing `Load(io.Reader)` callers do not see warning output (preserved compat).

---

### Droplet F.5.2 — `validateChildRuleReachability` + `validateKindStructuralCoherence`

**State:** done (round 1)
**Title:** `4c.5.F.5.2 — childrule reachability + kind-structural coherence`.

**Files / paths:**

- `internal/templates/load.go` (replace the no-op `validateChildRuleReachability` body; add new `validateKindStructuralCoherence`).
- `internal/templates/load_test.go` (test cases).

**Packages:** `github.com/evanmschultz/tillsyn/internal/templates`.

**Acceptance criteria:**

1. `validateChildRuleReachability` becomes a real check (per the existing `ErrUnreachableChildRule` sentinel doc-comment hook). Algorithm: starting from `kind=plan` (the root entry point), DFS through `child_rules` graph; assert every kind in the closed 12-value enum is reachable EXCEPT the explicit standalone-kinds set: `closeout`, `commit`, `refinement`, `discussion`, `human-verify`, `research`. Those are spawn-by-orchestrator-or-template, not auto-create-from-plan, so they are explicitly excluded from the reachability requirement.
2. **However:** see §3 Note 4 — the closed 12-kind enum may make this validator vacuously true for the embedded default. The validator's value is for ADOPTER templates that strip child_rules. F.5.2 still installs the validator to catch adopter mistakes.
3. Unreachable kinds outside the standalone set → `ErrUnreachableChildRule` wrapping the offending kind name.
4. New `validateKindStructuralCoherence(tpl Template) error` asserts a light cross-axis check: any `[kinds.X]` whose `structural_type == "drop"` MUST have at least one `[[child_rules]]` entry where `when_parent_kind == X` (a "drop" structural type implies the kind decomposes — orphan-drop catches missing decomposition rules). Missing → new sentinel `ErrIncoherentStructuralType` with kind name + reason.
5. The structural_type=drop coherence check is the THIN cross-axis wedge — full structural_type ↔ kind ↔ role validation is post-MVP.
6. New tests: `TestValidateChildRuleReachability_AllReachable`, `_BuildOrphanedRejected`, `TestValidateKindStructuralCoherence_DropWithoutChildRulesRejected`, `_DropletNoCheck`.
7. `mage test-pkg ./internal/templates` passes.

**Test scenarios (table-driven):**

- `default-go.toml → all reachable (including standalone kinds skipped)`.
- `synthetic template with kind=build but no child_rules referencing build → ErrUnreachableChildRule on build`.
- `synthetic template where kind=plan has structural_type=drop but no child_rules → ErrIncoherentStructuralType`.
- `synthetic template where kind=plan has structural_type=droplet → no coherence error fires`.

**Blocked by:** F.5.1 (shares load.go editing surface — package-level lock).

**Falsification mitigations:**

- F1: "Reachability validator collides with the standalone-kinds policy by silently allowing orphans." — Mitigation: the standalone set is explicit (one literal slice in load.go); the validator skips ONLY those. Tests assert orphans of `build` are rejected.
- F2: "structural_type=drop coherence rule misfires on closed-enum templates that legitimately decompose drops via in-orchestrator code." — Mitigation: per §3 Note 1, the embedded default uses `structural_type=droplet` for every kind today; the new validator is a no-op for the default. It only fires on adopter templates that opt into structural_type=drop.
- F3: "DFS graph from plan root traverses cycles — but cycle check already runs first, so this is a phantom risk." — Mitigation: validator runs AFTER `validateChildRuleCycles`; cycles are statically impossible at this stage.

---

### Droplet F.6.1 — Inline `mergeActionItemMetadataWithKindTemplate` stub

**State:** done (round 1)
**Title:** `4c.5.F.6.1 — fold mergeActionItemMetadataWithKindTemplate into CreateActionItem`.

**Files / paths:**

- `internal/app/service.go` (line 897 — replace call with inline assignment).
- `internal/app/kind_capability.go` (line 1002 — DELETE the stub function).
- `internal/app/kind_capability_test.go` (line 647 — update test reference doc-comment).

**Packages:** `github.com/evanmschultz/tillsyn/internal/app`.

**Acceptance criteria:**

1. `mergeActionItemMetadataWithKindTemplate` function removed from `kind_capability.go`.
2. Single caller at `service.go:897` replaced with direct assignment: `mergedMetadata := in.Metadata` (since the function was a pure pass-through `return base, nil`).
3. The `_, kindDef` argument that was previously passed is no longer required; ensure the surrounding code still uses `kindDef` for downstream logic if needed (audit the immediately-following lines in `CreateActionItem`).
4. Comment in `kind_capability_test.go:647` referencing the deleted function is updated or removed.
5. No behavior change — this is a pure refactor. `mage test-pkg ./internal/app` passes with all existing tests green, no new tests required.
6. `mage ci` passes.

**Test scenarios (table-driven):** none — pure refactor, existing tests cover the call site.

**Blocked by:** none. Independent of all other Theme F droplets (different files).

**Falsification mitigations:**

- F1: "kindDef was the second arg, and the new code drops it — but other code in CreateActionItem may rely on side effects." — Mitigation: function body was `return base, nil` (no side effects). Trivially safe.
- F2: "Future drop wants to re-introduce kind-template metadata merging and now has no named hook." — Mitigation: the doc-comment promised fold-in. If a future drop needs the hook back, re-introduce intentionally with a real implementation. YAGNI today.
- F3: "Test in `kind_capability_test.go` is named after the function — renaming or removing the test breaks discoverability." — Mitigation: test (line 647 area) is for `CompletionChecklist`-related behavior, not for this stub. Doc-comment update only — test name unchanged.

---

### Droplet F.3.1 — `till.template` MCP tool: `get` + `list_builtin` operations

**State:** done (round 1)
**Title:** `4c.5.F.3.1 — till.template MCP get + list_builtin operations`.

**Files / paths:**

- `internal/adapters/server/mcpapi/extended_tools.go` (NEW `registerTemplateTools` function near `registerKindTools`).
- `internal/adapters/server/mcpapi/handler.go` (call `registerTemplateTools` from `NewServer`).
- `internal/adapters/server/common/` (NEW interface `TemplateService` if not present — service layer adapter).
- `internal/app/service.go` or new `internal/app/template_service.go` (service-level methods backing the MCP tool: `GetProjectTemplate(projectID) (Template, error)` and `ListBuiltinTemplates() ([]string, error)`).
- `internal/adapters/server/mcpapi/extended_tools_test.go` (tests for both ops).

**Packages:** `github.com/evanmschultz/tillsyn/internal/adapters/server/mcpapi`, `github.com/evanmschultz/tillsyn/internal/app`, `github.com/evanmschultz/tillsyn/internal/adapters/server/common`.

**Acceptance criteria:**

1. New MCP tool `till.template` registered with operations `get` (current bake state) and `list_builtin` (returns array of builtin template names — `["default-generic", "default-go"]` post-F.2).
2. `operation=get` requires `project_id`; returns the JSON-decoded `KindCatalogJSON` from the project plus the bake-source provenance string (`"<bare-root>"` / `"<primary-worktree>"` / `"embedded-default-go"` / `"embedded-default-generic"`).
3. `operation=list_builtin` is read-only, no project context; returns `{"templates": ["default-generic", "default-go"]}` per Q1 (no FE).
4. Wire format: TOML-OUT for `get` (per spec); JSON-OUT for `list_builtin` (simple list).
5. **TOML-OUT plumbing:** `get` re-marshals the active `Template` via `toml.Marshal` and returns the bytes as a text result (NOT a JSON result). Implementation note: `templates` package may need a thin `MarshalTOML(Template) ([]byte, error)` helper if not present.
6. New tests: `TestTillTemplate_Get_EmbeddedDefault`, `TestTillTemplate_Get_BareRootSourced`, `TestTillTemplate_ListBuiltin`.
7. `mage test-pkg ./internal/adapters/server/mcpapi` and `./internal/app` pass.

**Test scenarios (table-driven):**

- `get for project with embedded-go bake → returns TOML matching default-go.toml + provenance="embedded-default-go"`.
- `get for project with custom bare-root template → returns TOML matching authored content`.
- `get for absent project_id → toolResultFromError with appropriate sentinel`.
- `list_builtin → returns ["default-generic", "default-go"] in stable order`.

**Blocked by:** F.2.1, F.2.2 (builtin template files must exist for list_builtin to be meaningful); F.1.x landed for the bake-source provenance to be accurate.

**Falsification mitigations:**

- F1: "TOML-out marshalling produces non-canonical output (key ordering drifts)." — Mitigation: use `toml.Marshal` from `github.com/pelletier/go-toml/v2`; document key-order as "implementation-defined." Adopters who need canonical output run their own formatter.
- F2: "Get returns the BAKE snapshot, not the live source file — adopters edit `.tillsyn/template.toml` mid-session expecting `get` to reflect it." — Mitigation: per Drop 3 finding 5.B.14, snapshot policy is documented. `till.template get` doc-string names this loud: "returns the bake-time snapshot; live edits require project re-create."
- F3: "list_builtin enumerates internal/templates/builtin/ directly (filesystem walk on embed.FS) and accidentally includes test fixtures." — Mitigation: list_builtin returns a hardcoded slice initialized at package level. No FS walking.

---

### Droplet F.3.2 — `till.template` MCP tool: `validate` operation

**Title:** `4c.5.F.3.2 — till.template MCP validate operation`.

**Files / paths:**

- `internal/adapters/server/mcpapi/extended_tools.go` (extend `registerTemplateTools` with the `validate` op).
- `internal/app/template_service.go` (extend with `ValidateCandidateTemplate(toml []byte) (validationReport, error)`).
- `internal/adapters/server/mcpapi/extended_tools_test.go` (validate tests).

**Packages:** `github.com/evanmschultz/tillsyn/internal/adapters/server/mcpapi`, `github.com/evanmschultz/tillsyn/internal/app`.

**Acceptance criteria:**

1. `operation=validate` accepts `template_toml` (TOML-IN, string-encoded). Server runs `templates.Load(strings.NewReader(args.TemplateTOML))`. Result envelope:
   - On success: `{"valid": true, "warnings": [...]}` where warnings are F.5.1's `validateAgentBindingFiles` warn lines.
   - On failure: `{"valid": false, "error": "<sentinel-name>: <wrapped-message>"}` with the sentinel canonical name (e.g. `"ErrUnknownTemplateKey"`).
2. Validation chain runs the FULL Load() pipeline (steps 1-4 of load.go), no skipping.
3. Validate is purely lexical — does NOT touch project state, does NOT re-bake, does NOT persist.
4. New tests: `TestTillTemplate_Validate_Valid`, `TestTillTemplate_Validate_UnknownKey`, `TestTillTemplate_Validate_BadSchemaVersion`, `TestTillTemplate_Validate_AgentBindingMissingWarn`.
5. `mage test-pkg ./internal/adapters/server/mcpapi` passes.

**Test scenarios (table-driven):**

- `valid default-go.toml content → valid:true, warnings empty`.
- `unknown-key TOML → valid:false, error names ErrUnknownTemplateKey`.
- `schema_version "v0" → valid:false, error names ErrUnsupportedSchemaVersion`.
- `valid template referencing missing agent file → valid:true, warnings non-empty (F.5.1 plumbed through)`.

**Blocked by:** F.3.1 (shares the registerTemplateTools function); F.5.1 (warning plumbing).

**Falsification mitigations:**

- F1: "TOML-IN size unbounded — adopter sends 100MB document, exhausts server memory." — Mitigation: cap input at 1MB per template (matches typical CI YAML caps); reject larger with `invalid_request`. Document the cap in tool description.
- F2: "Sentinel-name string drifts from the actual sentinel and adopters' tooling breaks." — Mitigation: error envelope includes `errors.Is`-friendly sentinel name PLUS the raw wrapped-error string. Adopters can route on either.
- F3: "Warnings pollute valid:true results unexpectedly." — Mitigation: warnings are a separate field; `valid:true` is unconditional on Load returning nil. Doc-string names the contract.

---

### Droplet F.3.3 — `till.template` MCP tool: `set` operation (atomic install + re-bake)

**Title:** `4c.5.F.3.3 — till.template MCP set operation atomic install`.

**Files / paths:**

- `internal/adapters/server/mcpapi/extended_tools.go` (extend `registerTemplateTools` with `set` op).
- `internal/app/template_service.go` (extend with `SetProjectTemplate(projectID, toml []byte, sessionID, sessionSecret) error`).
- `internal/adapters/server/mcpapi/extended_tools_test.go` (set tests).

**Packages:** `github.com/evanmschultz/tillsyn/internal/adapters/server/mcpapi`, `github.com/evanmschultz/tillsyn/internal/app`.

**Acceptance criteria:**

1. `operation=set` requires `project_id`, `template_toml`, `session_id`, `session_secret`. Auth-gated via `authorizeMCPMutation` per existing pattern.
2. Atomicity contract: the operation (a) validates the candidate via `templates.Load`, (b) writes the candidate to `<project.RepoBareRoot>/.tillsyn/template.toml` (or `<project.RepoPrimaryWorktree>/.tillsyn/template.toml` if bare-root is empty), (c) re-bakes the project's `KindCatalogJSON` via `bakeProjectKindCatalog`, (d) persists the project. Steps (b)-(d) MUST be transactional from the dev's perspective: if (c) or (d) fails, (b)'s write is rolled back via a temp-file + rename pattern.
3. **Atomicity strategy** (per §3 Note 3): write to `<dest>.tillsyn-set-<uuid>.tmp` first, then `os.Rename` to `<dest>` AFTER (c) and (d) succeed. If (c) or (d) fail, delete the tmp file. This makes (b) appear-atomic to other readers. Returns wrapped errors at each failure point.
4. On success: returns `{"set": true, "project_id": "...", "bake_source": "...", "bytes_written": N}`.
5. On validation failure: NO write happens, NO re-bake. Returns `{"set": false, "error": "..."}`.
6. New tests: `TestTillTemplate_Set_HappyPath`, `_ValidationFailureNoWrite`, `_AuthRejected`, `_RebakeFailureRollback`.
7. `mage test-pkg ./internal/adapters/server/mcpapi` and `./internal/app` pass.

**Test scenarios (table-driven):**

- `valid template + valid auth → file written, re-bake succeeds, returns set:true`.
- `invalid template TOML → no file write, returns set:false`.
- `valid template + auth rejected → no file write, returns auth error`.
- `re-bake fails (mocked) → tmp file deleted, returns rollback error`.

**Blocked by:** F.3.1, F.3.2 (shares registerTemplateTools); F.1.2 (paths walk semantics); F.2.1 (default-go.toml content stable).

**Falsification mitigations:**

- F1: "Re-bake means existing in-flight action items see a different KindCatalog mid-cascade." — Mitigation: per Drop 3 finding 5.B.14, the bake snapshot policy is "create-time only." `set` re-bakes by replacing the project's `KindCatalogJSON`; in-flight action items already created continue to use the old catalog (via their own metadata). NEW action items created after `set` use the new catalog. Doc-string names this.
- F2: "Write target ambiguity — bare-root vs primary-worktree." — Mitigation: F.3.3 picks the FIRST non-empty of `(RepoBareRoot, RepoPrimaryWorktree)`. If both are empty, `set` returns an error: "project has no checkout — cannot install template; create the project's checkout layout first." Document loud.
- F3: "Concurrent `set` operations race on the temp-file rename." — Mitigation: `os.Rename` is atomic on POSIX. Two concurrent sets serialize on the rename; the second wins. Doc-string names "last-writer-wins; serialize at the dev layer."
- F4: "Rollback fails (e.g. tmp file already gone) and leaves the project in a half-state." — Mitigation: rollback deletion is best-effort with `os.Remove`; `os.IsNotExist` is acceptable. The project's `KindCatalogJSON` is unchanged on rollback because re-bake is performed AFTER the rename — by ordering atomic-rename and re-bake correctly, the rollback target is just the tmp file. **Re-order:** validate → re-bake to a SHADOW catalog → write file via tmp+rename → swap shadow catalog into project + persist. If persist fails: delete file via reverse rename (rename `<dest>` back to `<dest>.tillsyn-set-failed-<uuid>` and surface to dev for manual cleanup). Document the failure-path operation.
- F5: "Re-bake target uses the JUST-written file via F.1.2 walk — but the walk also re-reads the bare-root, which differs." — Mitigation: `set` re-bakes from the EXACT bytes the dev sent (in-memory), not via a fresh walk. This guarantees the post-set bake matches the post-set on-disk file.

---

## 3. Notes

### Note 1 — F.5.2 reachability validator's value vs. closed enum

Per the closed 12-kind enum, every kind is reachable from `plan` IFF the standard 4 child_rules exist plus `research`/`closeout`/`commit`/`refinement`/`discussion`/`human-verify` are explicitly classified as standalone. The validator is **vacuously true** for the embedded `default-go.toml`. Its real value is for adopter templates that strip child_rules — typo-protection. Ship the validator anyway; the alternative (no check) loses ground later when adopter templates proliferate post-MVP.

### Note 2 — F.1 walk-order: bare-root applicability

The dev's `tillsyn` repo uses a bare-root layout (`/Users/.../hylla/tillsyn/` is the bare repo, `main/` is the primary worktree). But not every adopter project uses bare-root. The walk order MUST treat empty `RepoBareRoot` as "skip step 1, start at primary-worktree." Documented in F.1.2 acceptance #4.

### Note 3 — F.3.3 atomicity strategy

The naive sequence is: (1) validate, (2) write file, (3) re-bake, (4) persist. Failure at (3) or (4) leaves the file written but the catalog unupdated — semi-corrupt state. **Mitigation:** invert step ordering — (1) validate, (2) re-bake to a SHADOW catalog (in-memory), (3) write file via tmp+rename, (4) swap shadow catalog + persist project. Failure at (4) means file is on disk but catalog is not committed — surfaced to dev with a "manual cleanup" message rather than auto-rollback (avoids cascading failure paths). This is a deliberate tradeoff: file-on-disk + project-not-persisted is recoverable (re-run set with the same TOML); file-not-written + project-persisted-with-old-catalog is the worse case (silently drifted).

### Note 4 — F.5.2 reachability and standalone kinds

The standalone-kinds set (`closeout`, `commit`, `refinement`, `discussion`, `human-verify`, `research`) is hard-coded in F.5.2's validator. Future drops that introduce new kinds MUST update the standalone set or add child_rules to integrate them. Doc-comment names this loud so the next drop's planner sees the constraint.

### Note 5 — Q1 resolution: defer FE template

Per REVISION_BRIEF §9 Q1: ship `default-generic.toml` + `default-go.toml` only; defer `default-fe.toml` until an FE adopter materializes. F.1.3's resolver explicitly rejects `lang="fe"` with `ErrLanguageNotSupported`. Rationale: pre-MVP, no FE projects exist; F.4 marketplace CLI (Drop 4d-prime) will cover post-MVP FE adopter onboarding via project-authored templates rather than embedded builtins.

### Note 6 — Q2 resolution: warn vs error on missing agent file

Per REVISION_BRIEF §9 Q2: `validateAgentBindingFiles` is **warn-only**. Rationale: `~/.claude/agents/<name>.md` is dev-machine state; template correctness should not depend on a specific machine's filesystem. Adopters who want strict-fail can wrap the warn-logger to escalate. Documented in F.5.1 acceptance #2.

### Note 7 — F.6 fold-in scope

`mergeActionItemMetadataWithKindTemplate` is purely a `return base, nil` pass-through. Inlining is mechanical; no test-rigor concerns. The slot is preserved for a future re-introduction (kind-template-driven action-item metadata defaults) but YAGNI today.

### Note 8 — Theme F internal blocked_by graph (compiled)

```
F.2.1 (rebadge default → default-go) — independent
F.2.2 (add default-generic)           — blocks: F.2.1
F.1.1 (loadProjectTemplate signature) — blocks: F.2.1
F.1.3 (language-aware resolver)       — blocks: F.2.1, F.2.2
F.1.2 (filesystem walk)               — blocks: F.1.1, F.1.3
F.2.3 (self-host .tillsyn/template)   — blocks: F.2.1
F.2.4 (caller audit)                  — blocks: F.1.3, F.2.1, F.2.2
F.5.1 (warn + required-child-rules)   — blocks: F.2.1
F.5.2 (reachability + coherence)      — blocks: F.5.1
F.6.1 (inline KindTemplate stub)      — independent (parallel)
F.3.1 (template MCP get + list)       — blocks: F.2.1, F.2.2, F.1.2
F.3.2 (template MCP validate)         — blocks: F.3.1, F.5.1
F.3.3 (template MCP set atomic)       — blocks: F.3.1, F.3.2, F.1.2
```

### Note 9 — Cross-theme blocked_by considerations

- **No cross-theme blocker today.** Theme A (silent-data-loss) edits `internal/app/service.go` `UpdateActionItem`, which Theme F touches at `loadProjectTemplate` (different function). However, both edit the same file (`service.go`) — file-level lock applies. Sequence Theme A's service.go droplets BEFORE Theme F's F.1.x sequence, OR have orchestrator serialize at the file-lock level.
- **F.6.1 inlining touches `service.go:897` AND `kind_capability.go:1002`** — both edits are surgical and fast; F.6.1 is parallel-safe with everything else IF the orchestrator's file-lock manager handles multi-file atomic edits per droplet correctly.

### Note 10 — Mage targets

- All Go-changing droplets verify via `mage test-pkg <pkg>` for the specific package, then `mage ci` at the theme close.
- F.2.3 (TOML-only) verifies via `mage ci` only — no Go-level test harness exercises the file directly.

### Note 11 — Test fixture naming convention

Per §5 acceptance pattern: tests landed in this theme follow `Test<UnitName>_<Scenario>` shape. Existing test patterns in `internal/templates/load_test.go` use `TestLoad<X>` — F.5.x adheres. F.3.x follows MCPAPI patterns from `extended_tools_test.go`.

### Note 12 — Builder spawn-prompt reminders

Per REVISION_BRIEF §6:

- Builder spawn prompts MUST include "do NOT commit" directive.
- All builders are model: opus.
- Each builder reads any REVISIONS POST-AUTHORING section first if the sub-plan adds one.
- Builders may NOT run `mage install`.

---

## 4. Acceptance Summary (Theme-Level)

Theme F is complete when:

1. All 13 droplets above are individually green (build + QA-proof + QA-falsification).
2. `mage ci` passes on the post-Theme-F state.
3. `<project_root>/.tillsyn/template.toml` exists and matches `default-go.toml` byte-content (per F.2.3).
4. `loadProjectTemplate` walks the documented 3-step priority and returns embedded fallback at the bottom.
5. `till.template` MCP tool offers `get` / `validate` / `set` / `list_builtin` operations.
6. F.5 validators are wired into `templates.Load`'s chain in the documented order.
7. `mergeActionItemMetadataWithKindTemplate` is removed; `CreateActionItem` reads metadata directly.
8. No dev-FE-project regression (vacuously true pre-MVP — no FE projects exist).
9. STEWARD seed materialization remains language-aware and produces 6 anchors per the bound template.
