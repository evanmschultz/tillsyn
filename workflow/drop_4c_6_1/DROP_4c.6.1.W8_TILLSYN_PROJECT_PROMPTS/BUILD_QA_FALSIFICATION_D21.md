# W8.D21 — BUILD-QA-FALSIFICATION Verdict

**Date:** 2026-05-13
**Reviewer:** go-qa-falsification-agent (opus)
**Overall verdict:** PASS

## Attack Hypotheses Tested

### H1 — HOME-tier interference on case 1 ("project_tier_present")

- **Hypothesis:** Case 1 does NOT call `t.Setenv("HOME", ...)`. A dev with
  `~/.tillsyn/agents/go/builder-agent.md` populated could see the user-tier
  resolver fire and shadow the project-tier check, corrupting the assertion.
- **Test:** Read `render.go:676-693` — the resolver is short-circuit:
  project tier is consulted first; user tier ONLY fires `if !found` from
  project tier; embedded ONLY fires `if !found` from user tier.
- **Finding:** Case 1 seeds the project tier (`agentTierProjectFixture` at
  `<tmpdir>/.tillsyn/agents/go/builder-agent.md`). The project-tier read
  hits at `render.go:676`, returns `found=true`, and the user-tier branch
  is never entered. A populated `$HOME/.tillsyn/agents/go/builder-agent.md`
  cannot interfere. Case 1's correctness is independent of HOME state.
- **Verdict:** REFUTED. Hermeticity is preserved structurally by the resolver
  short-circuit, not by env neutralization. The builder's design comment at
  lines 1751-1754 ("Case 1: project tier wins regardless of user tier") is
  correct and load-bearing.

### H2 — Sentinel uniqueness (`SENTINEL_D21_PROJECT_TIER`)

- **Hypothesis:** If `SENTINEL_D21_PROJECT_TIER` appears in the embedded
  default body by accident, case 1's positive assertion + case 2's negative
  assertion both false-positive / false-negative.
- **Test:** `rtk grep -rn "SENTINEL_D21" internal/` returns 3 hits, ALL in
  the new D21 test (lines 1727, 1740, 1783 of render_test.go). Zero hits in
  any embedded fixture, production code, or other test.
- **Finding:** Sentinel is unique to the fixture; no collision risk.
- **Verdict:** REFUTED.

### H3 — `# PLACEHOLDER` negative assertion on case 1

- **Hypothesis:** If the case-1 fixture body happens to contain
  `# PLACEHOLDER` (e.g. via `validatorConformingBodySuffix()`), the
  `wantNotSentinel = "# PLACEHOLDER"` assertion false-positives.
- **Test:** Read fixture body construction at lines 1780-1783:
  ```
  "---\nname: builder-agent\n---\n\n" +
  "## Role\n\n" +
  strings.Repeat("Project-tier D21 smoke-test filler ...", 4) +
  "\nSENTINEL_D21_PROJECT_TIER\n"
  ```
  No `# PLACEHOLDER` literal in the constructed body. The builder
  EXPLICITLY documents (lines 1772-1779) why `validatorConformingBodySuffix()`
  was NOT used: that helper emits `# PLACEHOLDER` and would conflate the
  fixture with the embedded default's marker.
- **Verdict:** REFUTED. The builder anticipated this attack and structured
  the fixture body precisely to avoid it.

### H4 — Partial `t.Parallel()` coverage

- **Hypothesis:** Only case 1 calls `t.Parallel()`; parent + case 2 are
  serial. Spec says "Test uses `t.Parallel()`" — is partial coverage
  sufficient?
- **Test:** Go testing semantics: `t.Setenv` panics if any ancestor `*T`
  has called `t.Parallel`. Case 2 calls `t.Setenv("HOME", t.TempDir())` to
  neutralize the user tier — this REQUIRES the parent to be sequential.
  Builder's pattern (parent sequential, case 1 individually parallel) is
  the standard idiom for "selective subtest parallelism."
- **Finding:** The spec's acceptance criterion "Test uses `t.Parallel()`"
  is satisfied: case 1 is parallel-safe and calls `t.Parallel()`; case 2
  cannot be parallel because of `t.Setenv`. Parent-level `t.Parallel()`
  would make the test panic. Builder's comment block at lines 1709-1713
  documents the trade-off explicitly.
- **Verdict:** REFUTED. NIT only: spec wording was imprecise; builder's
  interpretation is technically correct and well-documented.

### H5 — `binding.AgentName` vs filename

- **Hypothesis:** Builder reports `AgentName=builder-agent` drives the
  rendered filename. Verify this is documented + asserted, not assumed.
- **Test:** Read lines 1797-1800 — explicit comment "binding.AgentName =
  \"builder-agent\" drives the rendered filename to
  `<bundle.Root>/plugin/agents/builder-agent.md` (render.go:616). Omitting
  AgentName would land the file at a path that does not match the
  assertion below." The assertion at line 1821 calls
  `readRenderedAgentFile(t, bundle.Paths.Root, binding.AgentName)` which
  reads `filepath.Join(bundleRoot, "plugin", "agents", agentName+".md")`
  (render_test.go:849). The path is driven by `binding.AgentName`.
- **Verdict:** REFUTED. Both documented and asserted.

### H6 — Strip-then-inject byte-stability

- **Hypothesis:** Strip strips any tool keys; inject re-adds them. If
  fixture frontmatter contained those keys, post-frontmatter body offsets
  could shift.
- **Test:** Read `stripAndInjectAgentFrontmatter` at render.go:721-770.
  The function operates ONLY on the frontmatter section (between the
  two `---\n` delimiters). The `postFrontmatter` section (lines 738) is
  carried through verbatim regardless of strip-inject outcome. The
  fixture's post-frontmatter section (containing `## Role`, filler, and
  `SENTINEL_D21_PROJECT_TIER`) is byte-stable. Additionally, the fixture
  frontmatter has only `name: builder-agent` — no `model`, `allowedTools`,
  or `disallowedTools` keys — so strip is a no-op on this fixture, and
  with nil tool slices, inject is also a no-op.
- **Verdict:** REFUTED. The strip-inject pipeline only mutates the
  frontmatter; post-frontmatter bytes (where the sentinel lives) are
  copied verbatim.

### H7 — Hard-coded path `go/` group resolution

- **Hypothesis:** Does the rendered group resolution actually pick `go`
  from `SystemPromptTemplatePath`?
- **Test:** `resolveAgentGroup` at render.go:871-878 applies `path.Dir`
  to `SystemPromptTemplatePath`. `path.Dir("go/builder-agent.md")` returns
  `"go"`. `readProjectTierAgent(projectWorktree, "go", "builder-agent.md")`
  reads `<worktree>/.tillsyn/agents/go/builder-agent.md` — which is
  exactly where `agentTierProjectFixture` seeds the file (render_test.go:821).
- **Verdict:** REFUTED. Path linkage is intact: binding template path →
  group "go" → seeded file location → resolver read path. Test passes
  in isolation confirms end-to-end correctness.

### H8 — Test position interference

- **Hypothesis:** Builder placed the test BEFORE
  `TestReadProjectTierAgent_SubdirPerGroup`. Does the new test shadow or
  interfere with the existing test?
- **Test:** Ran `mage test-pkg ./internal/app/dispatcher/cli_claude/render`
  — 86/86 tests PASS. No interference, no skipped, no failed.
- **Verdict:** REFUTED.

### H9 — Pre-existing failures

- **Hypothesis:** W2.D2 builder reported seeing "2 pre-existing failures
  on this exact test name." Verify the final test passes in isolation.
- **Test:** Ran
  `mage test-func ./internal/app/dispatcher/cli_claude/render TestRenderProjectTierOverridesEmbeddedDefault`
  — 3 tests (parent + 2 sub-cases) PASS. Git log on render_test.go (last
  5 commits) confirms the test is brand-new in the uncommitted diff.
  The W2.D2 builder's "pre-existing failures" claim was about OTHER tests
  in the file; this test did not exist before the D21 build.
- **Verdict:** REFUTED.

### H10 — Bonus case is actual FALLBACK (not missing-file error)

- **Hypothesis:** Spec says "bonus: second case where project-tier file
  absent → falls through to embedded default." Verify case 2 is the
  fallback behavior, not a missing-file error.
- **Test:** Read lines 1749-1830. Case 2 (`seedProjectTier=false`) sets
  HOME to an empty temp dir (line 1759), does NOT seed any project-tier
  file (lines 1766 `if tc.seedProjectTier` is false → skipped),
  calls `render.Render()` (line 1817), and asserts the rendered body
  contains `# PLACEHOLDER` (the embedded default's marker, line 1739).
  Resolver behavior: project-tier read returns `found=false` (no file,
  `fs.ErrNotExist` mapped to `("", false, nil)` at render.go:899-901);
  user-tier read returns `found=false` (empty HOME); embedded-tier fires
  and returns the `go/builder-agent.md` embedded body, which contains
  `# PLACEHOLDER` (verified at internal/templates/builtin/agents/go/builder-agent.md:6).
  Test PASSED in isolation — confirms the fallback path actually fires.
- **Verdict:** REFUTED. Case 2 is genuine fallback behavior, not a
  surfaced error.

### H11 — W8-SMOKE-R1 scope adherence (unit-only, not e2e)

- **Hypothesis:** Did builder accidentally add anything that's actually
  full end-to-end (deferred to Drop 4c.7)?
- **Test:** `rtk git diff --stat` shows ONLY
  `internal/app/dispatcher/cli_claude/render/render_test.go` is modified
  by D21. The test calls `render.Render(ctx, bundle, item, project,
  binding, nil)` directly with stub bundles/items/projects — no
  dispatcher invocation, no subagent spawn, no full pipeline. Pure
  unit-test on `render.Render()`.
- **Verdict:** REFUTED. Scope is honored.

### H12 — YAGNI: extra fixtures/helpers beyond acceptance

- **Hypothesis:** Did builder add helpers/fixtures beyond what acceptance
  required?
- **Test:** Builder used the existing `agentTierProjectFixture`,
  `readRenderedAgentFile`, `fixtureBundle`, `fixtureProject`,
  `fixtureItem` helpers — no new helpers introduced. The fixture body
  is inline string concatenation. The table struct has 5 fields, all of
  which are used. No deferred-future-use abstraction.
- **Verdict:** REFUTED.

## Unmitigated Counterexamples

None found.

## NITs

### NIT-1 — Doc-block paragraph break missing (cosmetic, severity: low)

- **Location:** render_test.go:1708-1709 — between the strip-then-inject
  note and the "TestRenderProjectTierOverridesEmbeddedDefault does not
  call t.Parallel()" sentence.
- **Observation:** The two paragraphs in the leading doc-comment are
  joined with `//` rather than separated by a blank `//` line. Reads as
  one run-on paragraph in `go doc` output.
- **Recommended action:** Insert blank `//` line between the strip-then-inject
  paragraph (ends "non-empty.") and the t.Parallel paragraph (starts
  "TestRenderProjectTierOverridesEmbeddedDefault does not..."). Cosmetic
  only — does not affect correctness or test behavior.

### NIT-2 — Acceptance-criterion bullet wording vs. selective parallelism (cosmetic, severity: low)

- **Location:** PLAN.md:1002 ("Test uses `t.Parallel()`").
- **Observation:** Spec wording is ambiguous — builder reasonably read
  this as "test exercises t.Parallel where safe" (case 1 only). Strict
  reading "every t.Run sub-case calls t.Parallel" is incompatible with
  case 2's `t.Setenv` requirement. Builder's interpretation is the only
  technically valid one, but the spec wording could mislead a future
  reader.
- **Recommended action:** This is upstream PLAN.md wording, not a
  builder defect. Leave for closeout / future plan-QA pass if relevant.

## Verdict rationale

All 12 attack hypotheses bounced (REFUTED). The builder:

1. Correctly identified the Go testing rule that `t.Setenv` panics under
   a parallel parent and adopted the selective-parallel-subtest idiom
   (case 1 parallel, case 2 sequential). Documented the trade-off
   explicitly.
2. Constructed a unique sentinel (`SENTINEL_D21_PROJECT_TIER`) with zero
   collisions across the codebase.
3. Anticipated the `# PLACEHOLDER` collision attack and structured the
   case-1 fixture body to omit the marker; documented this in a leading
   comment.
4. Relied on the project-tier resolver's structural short-circuit (project
   tier consulted first; user tier only on miss) rather than environment
   neutralization to make case 1 hermetic — a cleaner design than
   t.Setenv-on-every-case.
5. Honored the unit-test scope (no e2e dispatcher invocation; pure
   `render.Render()` exercise).
6. Did not introduce new helpers; reused existing fixtures.
7. Single-file modification, no production code touched.

Test runs verified:
- `mage test-func ./internal/app/dispatcher/cli_claude/render TestRenderProjectTierOverridesEmbeddedDefault`
  → 3/3 PASS (parent + 2 sub-cases).
- `mage test-pkg ./internal/app/dispatcher/cli_claude/render`
  → 86/86 PASS.

Two cosmetic NITs documented but neither blocks completion. Verdict: **PASS**.
