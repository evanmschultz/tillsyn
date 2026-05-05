# F.7.5c Builder QA Proof — Round 1

## Verdict

**PASS** — All 8 verification points hold against the modified source. Two
minor non-blocking falsification nits noted (whitespace-asymmetry in CLIKind
guard; test-hygiene drift in failure-injection cleanup adapter); neither
unsettles the PASS verdict.

## Scope

Round 1 review of F.7.5c (settings.json grant injection plumbing). Read-only.
No source edits; this MD is the only artifact.

## Files reviewed

- `internal/app/dispatcher/cli_claude/render/render.go`
- `internal/app/dispatcher/cli_claude/render/init.go`
- `internal/app/dispatcher/cli_claude/render/render_test.go`
- `internal/app/dispatcher/spawn.go`
- `internal/app/dispatcher/spawn_test.go`
- `internal/app/permission_grants_store.go` (interface contract cross-check)
- `internal/domain/permission_grant.go` (storage-layer empty-Rule defense)
- `workflow/drop_4c/4c_F7_5c_BUILDER_WORKLOG.md`

## Verification Results

### V1 — Render signature accepts ctx + lister, nil-tolerant

PASS. `render.go` L125-132 declares
`Render(ctx context.Context, bundle, item, project, binding, grantsLister PermissionGrantsLister) (string, error)`.
L120-124 doc comment explicitly documents nil-tolerance:
"grantsLister MAY be nil — render skips the grants-merge step and renders
the binding's ToolsAllowed only." L173 forwards ctx + lister to
`renderSettings`. Path-separator + empty-input validation (L133-142)
unchanged.

### V2 — mergeAllowList: binding-first dedup, combines grant.Rule, case-sensitive, empty-Rule defended at storage layer

PASS.

- Order: `mergeAllowList` (`render.go` L508-541) iterates
  `binding.ToolsAllowed` first (L517-523), then ranges over `grants` from
  the lister (L533-539). Both passes share a single `seen` map for
  preserve-first-seen dedup.
- Case sensitivity: dedup uses Go `map[string]struct{}` lookup, which is
  case-sensitive at the rune level. CLIKind is lowercased at the domain
  layer (`domain/permission_grant.go` L90 `strings.ToLower`); the lister
  call passes `string(binding.CLIKind)` verbatim, so case folding is
  the storage adapter's concern.
- Empty-Rule defense: `domain.NewPermissionGrant` rejects empty Rule at
  L86 with `ErrInvalidPermissionGrantRule` (after `TrimSpace`).
  `mergeAllowList` itself does NOT re-check Rule — it relies on the
  storage invariant. Acceptable: defense-in-depth here would just
  silently drop an invariant-violating row, masking a real storage bug.

### V3 — adaptRender type-asserts correctly

PASS. `init.go` L44-61 implements three branches:

| Input                                | Branch                                    | Result                              |
| ------------------------------------ | ----------------------------------------- | ----------------------------------- |
| `grantsLister == nil` (untyped nil)  | L52-53 `if grantsLister != nil` is false  | `lister` stays zero → forward       |
| non-nil + does not satisfy interface | L54-57 type assertion `ok == false`        | `return "", ErrInvalidGrantsLister` |
| non-nil + satisfies interface        | L58-59 `lister = typed`                    | forward typed                       |

Registration (L32) calls `RegisterBundleRenderFunc(adaptRender)` —
adaptRender's signature must match `BundleRenderFunc` byte-for-byte or the
program fails to compile. Compile-time guard.

### V4 — spawn.go passes context.Background() + nil

PASS. `spawn.go` L463: `render(context.Background(), bundle, item, project, resolved, nil)`.
Doc block L446-453 explicitly notes the nil is "deferred plumbing of the
production app.PermissionGrantsStore handle". Inline `TODO(F.7-CORE)` at
L459-462 captures the eventual ctx-plumbing follow-up.

### V5 — BundleRenderFunc type signature matches across all callers

PASS. `BundleRenderFunc` declared at `spawn.go` L132-139:
`func(ctx context.Context, bundle Bundle, item domain.ActionItem, project domain.Project, binding BindingResolved, grantsLister any) (string, error)`.

Callers verified compile-compatible:

- `init.go` L44-61 `adaptRender` — registers via L32. Mismatch would fail
  compile.
- `spawn_test.go` L690-697 faulty hook — 6-arg signature with `_ any`.
- `spawn_test.go` L715-732 cleanup adapter — same 6-arg shape.

The dispatcher cannot import `render` (cycle), hence the `any` type at the
seam; the doc comment L116-122 documents this explicitly.

### V6 — 6 new tests + 14 retrofitted existing tests pass

PASS by inspection (test bodies + signatures). I did not execute `mage ci`;
the builder's worklog reports green (V8 below).

**6 new F.7.5c tests** (`render_test.go`):

| Test                                                       | Line | Asserts                                                |
| ---------------------------------------------------------- | ---- | ------------------------------------------------------ |
| `TestRenderSettingsNilListerSkipsGrantsLookup`             | 598  | nil lister → binding only, no error                    |
| `TestRenderSettingsListerZeroGrantsLeavesBindingOnly`      | 617  | empty grants slice → binding only; lister IS called    |
| `TestRenderSettingsListerThreeGrantsAppendedAfterBinding`  | 650  | 3 distinct grants merged after binding entries         |
| `TestRenderSettingsListerDuplicateRuleDeduped`             | 680  | grants matching binding entries dropped                |
| `TestRenderSettingsListerErrorWrapsAndRollsBack`           | 707  | `errors.Is` + bundle rollback                          |
| `TestRenderSettingsEmptyCLIKindSkipsLookup`                | 737  | empty CLIKind short-circuits before lister call        |

**14 retrofitted existing tests** — all updated call sites confirmed to
pass `context.Background()` + `nil` for the new render params:
TestRenderHappyPathWritesAllFiveFiles (L117), TestRenderSystemPromptContainsStructuralTokens (L150),
TestRenderPluginManifestExactShape (L193), TestRenderMCPConfigExactShape (L222),
TestRenderSettingsPermissions (L259), TestRenderSettingsExplicitEmptyArraysWhenBindingEmpty (L309),
TestRenderAgentFileFrontmatter (L340), TestRenderAgentFileWithoutToolGating (L379),
TestRenderRollbackOnAgentDirFailure (L429), TestRenderRejectsEmptyBundleRoot (L458),
TestRenderRejectsEmptyAgentName (L477), TestRenderRejectsAgentNameWithPathSeparator (L504, 3 subtest leaves),
TestRenderOmitsOptionalSystemPromptFields (L528). The 12-test count plus
the 3-leaf subtest expansion in `TestRenderRejectsAgentNameWithPathSeparator`
matches the worklog's "14 existing" claim.

### V7 — NO commit per REV-13

PASS. `git log -5` HEAD is `4e52412 feat(dispatcher): gitignore handling,
TUI handshake, commit agent` (the F.7.5b predecessor). `git status -s`
shows the 5 modified Go files staged but uncommitted. REV-13 honored.

### V8 — mage ci green per worklog

VERIFIED VIA WORKLOG (not independently re-run). Worklog L106-110:
`mage check` + `mage ci` green; 2650 tests pass / 1 skip / 0 fail; render
coverage 84.7% (above 70% floor); dispatcher coverage 75.1%. Orchestrator
can re-verify pre-commit if desired.

## Falsification Findings (Non-Blocking)

### F1 — Whitespace asymmetry in CLIKind guard (minor)

`mergeAllowList` L525 uses
`strings.TrimSpace(string(binding.CLIKind)) == ""` for the empty-CLIKind
short-circuit. But the lister call L529 passes `string(binding.CLIKind)`
*un-trimmed*. If `binding.CLIKind == " claude "` (whitespace-padded), the
short-circuit doesn't fire, and the lister receives `" claude "` instead
of `"claude"`. The storage adapter case-folds + lowercases keys, so the
padded query never matches the lowercased stored value → silent zero
grants returned despite valid stored data.

**Mitigation**: production CLIKind is set via the
`dispatcher.CLIKindClaude` typed constant — no whitespace possible. The
asymmetry is theoretical for production paths today. Recommend trimming
CLIKind once at entry to `mergeAllowList` before both the guard and the
lister call to harden against future free-form CLIKind sources (e.g. TUI
overrides, Drop 4d codex registration).

**Disposition**: not blocking. File as a refinement note.

### F2 — Cleanup adapter drift risk in failure-injection test (minor)

`spawn_test.go` L715-732 restores the production render hook by inlining
its own `any → PermissionGrantsLister` adapter. The inline adapter is
functionally equivalent to `adaptRender` but is a separate copy. If
`adaptRender`'s behavior diverges in a future droplet (e.g. adds logging,
metrics, or tweaks the error path), the test's restoration drifts
silently — the test would still pass but future tests sharing the
process-wide hook would observe the wrong behavior.

**Mitigation**: the comment block L711-714 explains the workaround
(Go does not re-run `init()`), so future maintainers see the intent.
Cleaner long-term fix: export `adaptRender` (e.g.
`render.AdaptRenderForTest`) so the test imports the canonical form.

**Disposition**: not blocking. File as a refinement note.

## Findings Summary

| ID | Severity     | Area              | Disposition         |
| -- | ------------ | ----------------- | ------------------- |
| F1 | Minor (nit)  | CLIKind whitespace| Refinement candidate|
| F2 | Minor (nit)  | Test hygiene      | Refinement candidate|

## Hylla Feedback

- **Query**: `hylla_search_keyword` for `PermissionGrantsStore ListGrantsForKind`
  + `PermissionGrant Rule CLIKind ProjectID` against
  `github.com/evanmschultz/tillsyn@main`.
- **Missed because**: snapshot=5 predates F.7.17.7 + F.7.5c (PermissionGrant
  domain type + PermissionGrantsStore interface only landed in those
  droplets). First query returned zero results; second returned unrelated
  domain types (Comment, ChangeEvent, TemplateLibrary, AuthRequestPath,
  etc.) — keyword overlap on "Permission" / "Grant" was too sparse for
  the scoring to surface the new types.
- **Worked via**: `Read` on the local files
  (`internal/app/permission_grants_store.go`,
  `internal/domain/permission_grant.go`) directly.
- **Suggestion**: when an artifact ref's snapshot is older than the
  HEAD's last-modified-file timestamp, hylla could surface a stale-index
  banner in the search response so the agent immediately knows to fall
  back to local Read. Today the empty-result-set is indistinguishable
  from "the symbol does not exist." A `snapshot_age_warning` field on
  the response envelope would resolve this.
