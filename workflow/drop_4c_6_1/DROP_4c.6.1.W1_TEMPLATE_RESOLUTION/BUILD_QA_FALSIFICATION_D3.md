# W1.D3 — BUILD-QA-FALSIFICATION Verdict

**Date:** 2026-05-12
**Reviewer:** go-qa-falsification-agent (opus)
**Overall verdict:** PASS WITH NITS

---

# Section 0 — SEMI-FORMAL REASONING

## Planner

- **Premises:** D3 ships `readProjectTierAgent(projectWorktree, group, basename)` (new signature) + `assembleAgentFileBody` call-site update + test refactors. Constants `agentBodyDefaultGroup="go"` and `agentBodyFallbackGroup="gen"` already in tree pre-D3 (W4.D1).
- **Evidence:** `git diff internal/app/dispatcher/cli_claude/render/render.go` (+17/-13), `git diff render_test.go` (+136/-18). Production source has 3 references to `readProjectTierAgent` (definition + doc + 1 call site at line 676). Embedded FS embed list confirms `gen/orchestrator-managed.md` exists, `till-gdd/orchestrator-managed.md` is NOT embedded (used in cross-group fallback test).
- **Trace or cases:** 9 attack hypotheses enumerated below.
- **Conclusion:** Reduced-scope D3 — surgical 4-token change in render.go (signature + path), test fixtures threaded through helper. Low blast radius.
- **Unknowns:** None.

## Builder

- **Premises:** Construct counterexamples covering flat-layout miss, subdir-layout hit, group propagation, cross-group fallback, edge cases.
- **Evidence:** Read render.go body for `readProjectTierAgent` / `assembleAgentFileBody` / `resolveAgentGroup` / `validateAgentTemplatePath` / `resolveAgentBasename`; ran `mage test-func TestReadProjectTierAgent_SubdirPerGroup` (3/3 PASS), `mage test-pkg ./internal/app/dispatcher/cli_claude/render` (83/83 PASS), `mage test-pkg ./internal/app` (481/481 PASS); `rg readProjectTierAgent` in package shows single call site; `rg till-go|till-gen` shows ZERO matches in render.go + render_test.go for code/literal contexts but NONZERO matches in doc-comments (5 in render.go, 2 in render_test.go).
- **Trace or cases:** see Attack Hypotheses Tested.
- **Conclusion:** No correctness counterexample found. Two NIT-class issues (stale doc-comments referencing `till-gen` / `till-go`).
- **Unknowns:** None.

## QA Proof

- **Premises:** Every attack family must be tested with concrete evidence (not speculation).
- **Evidence:** All 9 hypotheses listed below; each either CONFIRMED or REFUTED with specific file:line + repro.
- **Trace or cases:** Test execution outputs cited verbatim where load-bearing.
- **Conclusion:** Evidence completeness confirmed.
- **Unknowns:** None.

## QA Falsification

- **Premises:** Did I miss an attack angle? Did I stop exploring too early?
- **Evidence:** 9 attack families covered: flat-miss correctness, subdir-hit correctness, group propagation, fixture exhaustiveness, stale literals, cross-group fallback, YAGNI, signature-change consumer impact, path-construction edge cases. All concurrency / interface-misuse / error-swallowing / leaked-goroutine surfaces inapplicable to a 4-token read-file change. Race / mutex attacks N/A — function has no shared state. Hidden-deps via `init()` — N/A — `init()` only calls `dispatcher.RegisterBundleRenderFunc(adaptRender)`, unrelated to `readProjectTierAgent`.
- **Trace or cases:** Attempted to confirm `path.Dir("go")` semantics (returns `"."` → falls to default); confirmed `validateAgentTemplatePath` rejects absolute / `..` / backslash / empty-segment BEFORE group derivation; confirmed cross-group fallback test fixture is correct (`till-gdd/orchestrator-managed.md` not in embed list → ENOENT → fallback to `gen/orchestrator-managed.md` which IS in embed list).
- **Conclusion:** No additional attack angle found.
- **Unknowns:** None.

## Convergence

- (a) QA Falsification produced no unmitigated counterexample to the PASS WITH NITS verdict.
- (b) QA Proof confirmed evidence completeness across all 9 hypotheses.
- (c) Remaining Unknowns: none; NITs routed via this verdict file (orchestrator decides whether to fold a follow-up sweep into a later wave).

---

## Attack Hypotheses Tested

### H1 — Flat-layout miss correctness

- **Hypothesis:** `readProjectTierAgent` might mis-return hit / error when legacy flat-layout file exists at `<worktree>/.tillsyn/agents/<basename>`.
- **Test:** Read `readProjectTierAgent` (render.go:892-905) — joins `(projectWorktree, projectAgentsSubdir, group, basename)`. With flat file at `<wt>/.tillsyn/agents/builder-agent.md`, the resolver looks at `<wt>/.tillsyn/agents/go/builder-agent.md` instead → `os.ReadFile` returns `fs.ErrNotExist` → function returns `("", false, nil)`. New `TestReadProjectTierAgent_SubdirPerGroup/flat_layout_is_miss` (render_test.go:1700-1736) pins this: seeds flat file, expects embedded PLACEHOLDER marker (tier-3 fired) and absence of `SENTINEL_FLAT_LAYOUT_MUST_NOT_WIN`. `mage test-func TestReadProjectTierAgent_SubdirPerGroup`: 3/3 PASS.
- **Finding:** REFUTED. Flat-layout miss is correct.
- **Verdict:** No counterexample.

### H2 — Subdir-layout hit correctness

- **Hypothesis:** Resolver might miss `<group>/<basename>` when seeded correctly.
- **Test:** `TestReadProjectTierAgent_SubdirPerGroup/subdir_layout_is_hit` seeds `<wt>/.tillsyn/agents/go/builder-agent.md` with `SENTINEL_SUBDIR_LAYOUT_WINS` sentinel, expects it to appear in rendered output. `mage test-func`: PASS. Existing `TestAssembleAgentFileBody_ProjectOverride` (render_test.go:926-962) also exercises subdir hit via `agentTierProjectFixture` (which writes to `<wt>/.tillsyn/agents/<group>/<basename>`).
- **Finding:** REFUTED. Subdir hit works.
- **Verdict:** No counterexample.

### H3 — Group propagation

- **Hypothesis:** `group` resolved in `assembleAgentFileBody` might not match `group` passed to `readProjectTierAgent`. Try a binding with `SystemPromptTemplatePath` having a directory prefix.
- **Test:** Read `assembleAgentFileBody` (render.go:656-703). At line 673 `group := resolveAgentGroup(binding)`. At line 676 same `group` variable is passed to `readProjectTierAgent`. At line 682 same `group` passed to `readUserTierAgent`. At line 689 same `group` passed to `readEmbeddedTierAgent`. SINGLE source of truth for group across all three tiers. `TestRenderValidatorAcceptsAllEmbeddedPlaceholders` at render_test.go:1632-1673 exercises every embed-FS group (gen / go / fe / till-gdd) via `SystemPromptTemplatePath: relPath` where relPath = `"<group>/<basename>"`, with fixture seeded at `<wt>/.tillsyn/agents/<group>/<basename>` — that test passes for all 27+ placeholders.
- **Finding:** REFUTED. Group propagation is consistent.
- **Verdict:** No counterexample.

### H4 — Test fixture exhaustiveness

- **Hypothesis:** 6 existing tests + 1 new might miss path coverage; pre-existing tests might still use flat layout.
- **Test:** `rg agentTierProjectFixture render_test.go` shows 6 call sites in pre-existing tests (lines 942, 1481, 1519, 1547, 1579, 1662), all updated to pass `group` parameter. New `TestReadProjectTierAgent_SubdirPerGroup` adds 2 sub-tests (flat-miss + subdir-hit). No pre-existing test directly references `projectAgentsSubdir` outside the helper. No tests bypass `agentTierProjectFixture` to manually populate `<wt>/.tillsyn/agents/<basename>` (verified via `rg '\.tillsyn/agents' render_test.go` — no flat-layout writes outside the new H1 test). 83/83 render tests pass.
- **Finding:** REFUTED. Fixture coverage is exhaustive within the package.
- **Verdict:** No counterexample.

### H5 — Stale comments / leftover `till-go` / `till-gen` literals

- **Hypothesis:** Builder claimed "2 stale comments fixed" — verify ALL `till-go`/`till-gen` literals are gone.
- **Test:** `rg 'till-go|till-gen' render.go render_test.go` returns NONZERO matches. In render.go:
  - line 145: `the till-go embedded default` (doc-comment for `ErrInvalidAgentTemplatePath`).
  - line 176: `<agentBodyEmbeddedRoot>/till-gen/<basename>` (doc-comment for `agentBodyEmbeddedRoot`).
  - line 601: `cross-group till-gen fallback` (doc-comment in `renderAgentFile`).
  - line 688: `Tier 3 — embedded tier with cross-group fallback to till-gen.` (inline comment in `assembleAgentFileBody`).
  - line 815: `till-go embedded default` (doc-comment in `validateAgentTemplatePath`).
  In render_test.go:
  - line 781: project tier doc `<project.RepoPrimaryWorktree>/.tillsyn/agents/<basename>` — STILL claims FLAT layout in doc-comment for the test family header.
  - line 785: `fallback to till-gen/<basename>` (test family doc).
  - line 925: `<project>/.tillsyn/agents/<basename>` (doc-comment for `TestAssembleAgentFileBody_ProjectOverride`).
  These are doc-comment / inline-comment drift. The `ErrAgentBodyNotFound` doc-comment (lines 97-103) IS updated to `gen`. `agentBodyDefaultGroup` / `agentBodyFallbackGroup` doc-comments (lines 185-198) explain the rename. Net: builder updated the load-bearing comments adjacent to the changed symbols but missed 5 sibling doc-comments in render.go + 3 in render_test.go. Comments do not affect program behavior; 83/83 tests pass.
- **Finding:** CONFIRMED as NIT (no behavioral counterexample; comment drift only).
- **Verdict:** NIT — see NITs section below.

### H6 — Cross-group fallback to `gen`

- **Hypothesis:** AC6/AC7 cross-group fallback might break after subdir change.
- **Test:** `TestAssembleAgentFileBody_CrossGroupFallbackToGen` (render_test.go:980-1006) uses `SystemPromptTemplatePath: "till-gdd/orchestrator-managed.md"`. Verified via embed.go that `till-gdd/orchestrator-managed.md` is NOT in the `//go:embed` directive list (lines 119-125 embed only 7 till-gdd agents, no orchestrator-managed.md), and `gen/orchestrator-managed.md` IS embedded (line 98). So the fallback path is genuinely exercised. Test passes. Also `TestAssembleAgentFileBody_CrossGroupFallbackMissesBothGroups` (render_test.go:1013+) covers double-miss → `ErrAgentBodyNotFound`. `readEmbeddedTierAgent` (render.go:947-975) unchanged in this droplet.
- **Finding:** REFUTED. Cross-group fallback preserved.
- **Verdict:** No counterexample.

### H7 — YAGNI

- **Hypothesis:** Builder added scope beyond reduced D3 (signature + path + tests).
- **Test:** Diff inspection. render.go changes: (1) `ErrAgentBodyNotFound` doc-comment `till-gen` → `gen` (cosmetic, aligns with W4.D1 rename); (2) `userAgentsSubdir` doc-comment expanded to note project tier is now also group-scoped (load-bearing for new readers); (3) `readProjectTierAgent` doc-comment updated to reflect new path shape; (4) `assembleAgentFileBody` call-site (1 line); (5) `readProjectTierAgent` signature + body (2 lines). render_test.go changes: helper signature, helper body, 5 existing fixture calls, fallthrough placeholder test (added group derivation), new `TestReadProjectTierAgent_SubdirPerGroup` (2 sub-tests). All within D3's reduced scope. No new abstractions, no new exported APIs, no premature generalization.
- **Finding:** REFUTED. No YAGNI.
- **Verdict:** No counterexample.

### H8 — Signature-change consumer impact (LSP findReferences)

- **Hypothesis:** `readProjectTierAgent` might have additional callers outside `assembleAgentFileBody`.
- **Test:** `rg readProjectTierAgent` across `internal/` returns 3 hits in render.go: definition (line 892), doc-comment (line 880), single call site (line 676). No other Go source files reference it. Function is package-private (lowercase `read…`). Init.go does not call it (verified). Test file does not call it directly by name (verified — exercises via `Render()` wrapper).
- **Finding:** REFUTED. Single call site.
- **Verdict:** No counterexample.

### H9 — Path-construction edge cases

- **Hypothesis:** Empty `group`, `group` with slashes, `group` with `..` might cause unsafe path construction.
- **Test:** Trace input path: `binding.SystemPromptTemplatePath` is first validated by `validateAgentTemplatePath` (render.go:817-842) BEFORE any group derivation. Rejects:
  1. Absolute paths (leading `/`).
  2. Backslashes anywhere.
  3. Any segment equal to `..`.
  4. Empty intermediate segments (consecutive separators).
  Then `resolveAgentGroup` returns `path.Dir(trimmed)` if non-`.` non-empty, else `agentBodyDefaultGroup = "go"`. `agentBodyDefaultGroup` is a compile-time constant — never empty. So `group` passed to `readProjectTierAgent` is either a validated path-prefix from `SystemPromptTemplatePath` or literal `"go"`. The `path.Dir` derivation cannot produce `..` (rejected upstream) or absolute paths (rejected upstream) or empty (falls through to default). Empty `group` impossible. `group` with `..` impossible. `group` with leading `/` impossible. `group` containing `/` is POSSIBLE (e.g. `SystemPromptTemplatePath = "a/b/c.md"` → `group = "a/b"`) but that's a nested subdir, which is a valid `filepath.Join` shape — not a security issue. `validateAgentBasename` separately catches leaf-traversal at line 847.
- **Finding:** REFUTED. Path-construction is defense-in-depth-protected upstream.
- **Verdict:** No counterexample.

---

## Unmitigated Counterexamples

None found.

---

## NITs

### NIT-1 — Stale `till-gen` / `till-go` references in render.go doc-comments

- **Severity:** Low (cosmetic / cognitive-load only; zero behavioral impact; tests all pass).
- **Locations:**
  - render.go:145 — `validateAgentTemplatePath` doc-comment: `"till-go embedded default"`.
  - render.go:176 — `agentBodyEmbeddedRoot` doc-comment: `"<agentBodyEmbeddedRoot>/till-gen/<basename>"`.
  - render.go:601 — `renderAgentFile` doc-comment: `"cross-group till-gen fallback"`.
  - render.go:688 — `assembleAgentFileBody` inline: `"// Tier 3 — embedded tier with cross-group fallback to till-gen."`.
  - render.go:815 — `validateAgentTemplatePath` doc-comment body: `"routes to the till-go embedded default"`.
- **Recommended action:** Single sweep replacing `till-gen` → `gen` and `till-go` → `go` in render.go doc-comments. Builder addressed the comment directly attached to the symbol the droplet edited (`ErrAgentBodyNotFound` lines 97-103 + `userAgentsSubdir` lines 206-208) but missed siblings. Suggest a follow-up doc-hygiene droplet in W1.D3.x or absorb into W4 (which already owns the rename narrative). NOT a build-blocker — `mage ci` is green.

### NIT-2 — Stale `till-gen` / flat-layout references in render_test.go doc-comments

- **Severity:** Low (cosmetic; zero test impact).
- **Locations:**
  - render_test.go:781 — `// (1) project tier — <project.RepoPrimaryWorktree>/.tillsyn/agents/<basename>` (test family header still describes FLAT layout for project tier).
  - render_test.go:785 — `// fallback to till-gen/<basename> on fs.ErrNotExist.` (test family header).
  - render_test.go:925 — `TestAssembleAgentFileBody_ProjectOverride` doc-comment: `"<project>/.tillsyn/agents/<basename> exists"` (still claims FLAT).
- **Recommended action:** Refresh the test-family banner at render_test.go:778-795 plus the `TestAssembleAgentFileBody_ProjectOverride` doc to use subdir-per-group description. Folded into the same sweep as NIT-1.

---

## Verdict rationale

The reduced-scope D3 builder shipped a surgical 4-token production change (`readProjectTierAgent` signature + path) plus the necessary test-helper signature update and 5 existing-fixture call-site updates. All 83 render tests pass and `mage test-pkg ./internal/app` (481/481) confirms no upstream breakage. The new `TestReadProjectTierAgent_SubdirPerGroup` pins the flat-miss + subdir-hit contract correctly. Cross-group fallback to `gen` works correctly (verified via embed.go directive — `till-gdd/orchestrator-managed.md` is genuinely absent, `gen/orchestrator-managed.md` is genuinely embedded). `readProjectTierAgent` is package-private with a single call site at render.go:676 (verified via `rg`). Path-construction is defense-in-depth-protected by `validateAgentTemplatePath` running BEFORE group derivation. No YAGNI. No concurrency / interface-misuse / error-swallowing surfaces (read-file function, no goroutines, deterministic).

Two NIT-class doc-comment drifts remain (`till-gen` / `till-go` strings in 5 render.go doc-comments + 3 render_test.go doc-comments). Zero behavioral impact; recommend a follow-up sweep folded into a later droplet that owns rename narrative. Per `feedback_nits_are_first_class.md` these are flagged but do not block PASS — they are pure comment hygiene and the build is green.

**Verdict: PASS WITH NITS.**
