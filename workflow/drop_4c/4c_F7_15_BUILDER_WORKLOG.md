# Drop 4c F.7.15 — Project-metadata toggles `dispatcher_commit_enabled` + `dispatcher_push_enabled` — Builder Worklog

**Droplet:** F.7-CORE F.7.15
**Builder model:** opus
**Date:** 2026-05-04
**Plan source:** `workflow/drop_4c/F7_CORE_PLAN.md` § F.7.15 + REVISIONS POST-AUTHORING + Master PLAN.md L20 + spawn prompt

## Round 1

### Goal

Add two `*bool` project-metadata toggles to `domain.ProjectMetadata` matching the Drop 4a Wave 4a.25 `OrchSelfApprovalEnabled` precedent (`internal/domain/project.go:119-145`). Pure schema additions + accessor methods; NO gate-execution behavior (Drop 4d's territory). Default polarity is INVERTED relative to `OrchSelfApprovalEnabled` — both new toggles ship default-OFF per Master PLAN.md L20: *"Commit + push gates default OFF via `dispatcher_commit_enabled` + `dispatcher_push_enabled` project metadata pointer-bools. Default template ships gates listed-but-toggle-disabled until dogfood proves them safe."*

### Files edited

- `internal/domain/project.go`
  - Extended `ProjectMetadata` struct (lines 119-160-ish post-edit) with two new pointer-bool fields:
    - `DispatcherCommitEnabled *bool` (TOML `dispatcher_commit_enabled,omitempty`, JSON `dispatcher_commit_enabled,omitempty`).
    - `DispatcherPushEnabled *bool` (TOML `dispatcher_push_enabled,omitempty`, JSON `dispatcher_push_enabled,omitempty`).
  - Doc-comments cite the W3.2 falsification-attack-3 mitigation rationale (preserve absent-vs-explicit-false distinction even when both currently collapse to "disabled" — reserves the pointer shape for future merge-with-template-default semantics).
  - Added two accessor methods next to `OrchSelfApprovalIsEnabled`:
    - `IsDispatcherCommitEnabled() bool` — nil-means-DISABLED (returns false on nil OR `*false`; true ONLY on `*true`).
    - `IsDispatcherPushEnabled() bool` — same default-disabled polarity.
  - Doc-comments cite Master PLAN.md L20 and explicitly call out the polarity inversion vs. `OrchSelfApprovalIsEnabled`.

- `internal/domain/project_test.go`
  - Added `import "github.com/pelletier/go-toml/v2"` for the new TOML round-trip test (lib already in go.mod L66).
  - Added 4 new test functions, each parallel-safe and named to mirror the existing `TestProjectMetadataOrchSelfApproval*` pattern:
    - `TestProjectMetadataIsDispatcherCommitEnabledDefaults` — 3-case table: nil → false, *true → true, *false → false.
    - `TestProjectMetadataIsDispatcherPushEnabledDefaults` — same 3 cases for the push toggle.
    - `TestProjectMetadataDispatcherTogglesJSONRoundTrip` — 4-case table covering both fields together: both nil (omits both keys), commit *true & push nil, both *false, both *true. Each case asserts marshal substrings (must-have / must-omit), `Is*()` helpers post-decode, and pointer-shape preservation (nil stays nil, non-nil stays non-nil).
    - `TestProjectMetadataDispatcherTogglesTOMLRoundTrip` — 3-case table mirroring the JSON test for `pelletier/go-toml/v2`. Verifies `omitempty` semantics on TOML output and round-trip equivalence.

- `internal/adapters/storage/sqlite/repo_test.go`
  - Added `TestRepository_PersistsDispatcherCommitAndPushEnabled` (5-case table) appended at end-of-file.
  - Each case creates a project with the two-field metadata, calls `repo.CreateProject` → `repo.GetProject`, asserts the dereferenced semantics + pointer-shape equivalence, then mutates both fields to *true and exercises `repo.UpdateProject` → `repo.GetProject` to cover the SET path. No DDL touched: rides the existing `metadata_json` JSON-blob column (same path as `OrchSelfApprovalEnabled`).

- `internal/tui/model_test.go` (**out-of-scope-list edit — see "Scope expansion" below**)
  - Extended `projectMetadataInternal` map in `TestProjectSchemaCoverageIsExplicit` (lines 15001-15022) to classify the two new fields. Without this, the schema-coverage guard mechanically fails (`is not classified for TUI/schema coverage`) and `mage ci` cannot turn green. Doc-comment explains the polarity rationale and points back to OrchSelfApprovalEnabled's classification block for symmetry.

### Files NOT edited (and why)

- `internal/adapters/storage/sqlite/repo.go` — no DDL or plumbing change required. `OrchSelfApprovalEnabled` rides the `metadata_json` JSON-blob column on `projects` (lines 868-880 / 922-933), and the new fields slot into the same envelope through `json.Marshal(p.Metadata)` in `CreateProject` + `UpdateProject`. No fresh DB needed. No `rm -f ~/.tillsyn/tillsyn.db` callout required.

### Persistence pattern — JSON-blob (no DDL)

Confirmed via `rg "OrchSelfApprovalEnabled" internal/adapters/storage/sqlite/repo.go` returning zero hits — the field never appears in repo.go because it's part of the metadata-JSON envelope. The new fields inherit the same path mechanically.

### Scope expansion (orchestrator routing)

The directive's "Files to edit/create" list omitted `internal/tui/model_test.go`, but `TestProjectSchemaCoverageIsExplicit` mechanically enforces that EVERY exported `ProjectMetadata` field is classified into one of `editable / readOnly / internal`. Adding fields to `ProjectMetadata` and refusing to classify them means `mage ci` cannot pass — directly violating the acceptance criterion. Per the agent-local "If scope expansion is needed, report it in your closing response for the orchestrator to route" rule, I made the surgical 2-line classification edit and call it out here. The same precedent already exists in the file: `OrchSelfApprovalEnabled` is classified `internal` for identical reasons.

### Verification

```
mage testPkg ./internal/domain/                 — 303/303 pass
mage testPkg ./internal/adapters/storage/sqlite — 93/93 pass
mage ci                                         — green
  - Sources verified
  - Go formatting clean (gofumpt)
  - Test stream: 2492 passed, 1 skipped, 0 failed across 23 packages
  - Coverage threshold met (all packages ≥ 70%; domain at 81.8%, sqlite at 75.6%, tui at 71.0%)
  - Build green (./cmd/till)
```

### Acceptance criteria

- [x] `DispatcherCommitEnabled *bool` + `DispatcherPushEnabled *bool` added to `ProjectMetadata` with TOML + JSON tags + omitempty.
- [x] `IsDispatcherCommitEnabled()` + `IsDispatcherPushEnabled()` accessors with nil-means-FALSE semantics (default-disabled per Master PLAN.md L20).
- [x] All 8 test scenarios pass (3 commit-default + 3 push-default + JSON round-trip + TOML round-trip + SQLite round-trip).
- [x] Persistence pattern matches existing `OrchSelfApprovalEnabled` (JSON-blob in `metadata_json` column; no DDL).
- [x] No new SQLite column → no fresh-DB callout needed.
- [x] `mage ci` green.
- [x] Worklog written.
- [x] No commit by builder (orchestrator commits after QA pair returns green).

### Proposed commit message

```
feat(domain): add dispatcher_commit_enabled + dispatcher_push_enabled toggles
```

(Single-line conventional commit, ≤72 chars.)

## Hylla Feedback

N/A — directive explicitly forbade Hylla calls for this droplet ("NO Hylla calls. Use `Read` / `Grep` / `LSP` directly."). Native-tool path was sufficient given the precedent (`OrchSelfApprovalEnabled`) was a known-good anchor at a known line range.
