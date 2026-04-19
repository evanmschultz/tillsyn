---
task: P4-T4 — BUILD DIFF INPUT FROM RESOURCEREFS
tillsyn_id: 7aec9fcf-d9f3-482d-80b9-9377785c305a
role: builder (go-builder-agent, sonnet)
state: done
blocked_by: none (P4-T3 already `done` — commit 60b6fc5)
worktree: /Users/evanschultz/Documents/Code/hylla/tillsyn/drop/1.5/
---

# P4-T4 — BUILD DIFF INPUT FROM RESOURCEREFS

## Purpose

Replace the placeholder / empty `paths []string` argument that P4-T3 passed to `Differ.Diff` with the real set derived from the active plan-item's `TaskMetadata.ResourceRefs` (Option D replan — Drop 1 deferred; reuse existing `TaskMetadata.ResourceRefs []ResourceRef` at `internal/domain/workitem.go:153`). Partition ResourceRefs by `Tags[0]`: `"path"` or `"file"` → `Location` unchanged; `"package"` → `Location + "/"` (normalize trailing slash so `git diff` treats as a tree prefix); any other tag → skip. When the plan-item has zero tagged ResourceRefs (or all entries are skipped), fall back to the whole repo (conventional `git diff` behaviour).

**Option D replan rationale:** Pre-replan this drop keyed on Drop 1's `PlanItem.Paths` + `PlanItem.Packages` domain fields. Drop 1 is deferred, so instead we read `TaskMetadata.ResourceRefs` and use the `Tags[0]` convention established by P3-A (`"path"`) and P3-B (`"file"`) plus a `"package"` tag (future picker — not in Drop 1.5 scope; tested here via fixture data only). Migration back to domain fields if Drop 1 ships is mechanical: swap the read source, keep the partition logic.

## Paths (modified)

- `internal/tui/diff_mode.go` (modified — replace placeholder paths with ResourceRef resolution; add `resolveDiffPaths(item *domain.PlanItem) []string` helper reading `item.Metadata.ResourceRefs`).
- `internal/tui/diff_mode_test.go` (modified — add table test for partition cases).
- `internal/tui/model.go` (modified — pass active plan-item into `diffMode.SetItem()` or equivalent before entering modeDiff).

## Packages

- `internal/tui` (modified)
- `internal/domain` (read-only — access `TaskMetadata.ResourceRefs []ResourceRef` at `internal/domain/workitem.go:153`; `ResourceRef` struct at lines 123-133)

## Acceptance Criteria (QA yes/no calls)

- Helper `resolveDiffPaths(item *domain.PlanItem) []string` (lowercase unexported) in `diff_mode.go`:
  - Input: a plan-item with `Metadata.ResourceRefs []ResourceRef`.
  - Partition: for each ref, inspect `ref.Tags[0]` (empty `Tags` → skip). `"path"` or `"file"` → append `ref.Location` unchanged. `"package"` → append `ref.Location` with a trailing `/` normalized via suffix check (don't double-slash). Any other tag → skip.
  - Deduplicate preserving first-occurrence order. If the same `Location` appears as both `path` and `package`, the trailing-slash (package) variant wins — it's the more specific tree-prefix form.
  - Empty input (nil / empty / all-skipped ResourceRefs) returns an empty slice, which Differ.Diff treats as whole-repo.
- `SetItem(item *domain.PlanItem)` method on `*diffMode` (or equivalent) called from Model before entering modeDiff.
- `SetItem` invokes `Differ.Diff(ctx, startSHA, endSHA, resolveDiffPaths(item))`; start/end SHAs come from existing Model context (branch start + HEAD; Drop-1 won't have per-item SHA fields, so use Model-level defaults for now).
- When `Metadata.ResourceRefs` change on the active plan-item mid-session (e.g. after P3-A or P3-B write new entries), diff mode re-computes on next entry (i.e. don't cache DiffResult across sessions). Assert by test.
- Service interface still 44 methods.
- Top-level `Model` field additions: 0 new fields in this build-drop (we reuse `diffMode *diffMode` from P4-T3).
- `item.Metadata.ResourceRefs` are read-only here — do not mutate.
- Errors wrapped with `%w`.

## TDD Test List (minimum)

- `TestResolveDiffPaths_EmptyResourceRefs` — returns empty slice.
- `TestResolveDiffPaths_PathTagsOnly` — `path`-tagged Locations returned unchanged.
- `TestResolveDiffPaths_FileTagsOnly` — `file`-tagged Locations returned unchanged.
- `TestResolveDiffPaths_PackageTagsOnly` — each `package`-tagged Location gets a trailing slash (and pre-existing trailing slash is not doubled).
- `TestResolveDiffPaths_MixedTags` — merged from `path`/`file`/`package` partitions; order preserved from first occurrence across iterations.
- `TestResolveDiffPaths_Dedup` — duplicate Locations within the same tag removed.
- `TestResolveDiffPaths_PackageWinsOverPath` — same Location `"internal/tui"` in both a `path`-tag and a `package`-tag ResourceRef → output contains `"internal/tui/"` exactly once (trailing-slash variant wins).
- `TestResolveDiffPaths_UnknownTagSkipped` — ResourceRef with `Tags[0]` outside `{"path","file","package"}` is silently skipped (not a compile or runtime error).
- `TestResolveDiffPaths_EmptyTagsSkipped` — ResourceRef with zero-length `Tags` is skipped.
- `TestDiffMode_SetItem_PassesResolvedPaths` — fake Differ; assert it receives `resolveDiffPaths(item)` for the last-set item.
- `TestDiffMode_RecomputesOnItemChange` — set item A, enter diff, exit; mutate `item.Metadata.ResourceRefs` externally; enter diff again; assert fresh Diff call.

## Mage Targets

- `mage test-pkg internal/tui` (-race, -count=1).
- `mage ci`.

## Go Idioms (reinforce)

- `resolveDiffPaths` is a pure function — no receiver, takes item, returns slice. Easy to unit-test.
- Preserve slice order from the merge for deterministic diffs: iterate ResourceRefs once, append per tag, dedup keeps first occurrence.
- Use `strings.HasSuffix` + conditional append for package trailing-slash normalization (don't double-slash if user already typed one).
- Doc comments on the new helper + the `SetItem` method.

## Hylla Artifact Ref

`github.com/evanmschultz/tillsyn@main`
