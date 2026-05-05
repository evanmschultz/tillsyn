# F.7.17.6 Builder QA Proof — Round 1

Droplet: **F.7.17.6 — `Manifest.CLIKind` extension**

Reviewer: `go-qa-proof-agent`

Verdict: **PASS**

---

## 1. Scope

Verify that the builder's claim — adding `CLIKind string` to `ManifestMetadata`
with JSON tag `cli_kind` (no omitempty), populating it from
`resolved.CLIKind` in `BuildSpawnCommand`, and pinning the round-trip through
`UpdateManifestPID` — is fully supported by the on-disk evidence, with no
attribution overlap into sibling parallel droplet F.7.13.

## 2. Evidence Trace

### 2.1 Field added with correct JSON tag, no `omitempty`

`internal/app/dispatcher/bundle.go` line 146:

```go
CLIKind string `json:"cli_kind"`
```

The struct tag is `cli_kind` (snake_case per memory §2 cross-CLI manifest
shape). No `omitempty` — verified literally on the struct line. The
type-level doc-comment block (lines 101–117 of bundle.go) explicitly cites:

- "Drop 4c F.7-CORE F.7.17.6 is the sole owner of manifest.cli_kind"
- "master PLAN §5 \"Tillsyn struct extension policy\""
- "F.7.8's orphan scan (REV-5: blocked_by F.7.17.6)" as the consumer.

Field-level comment (lines 132–146) restates the same triumvirate (REV-4 +
master PLAN §5 + F.7.8 consumer) and explicitly justifies the no-`omitempty`
choice: "the orphan scan sees an explicit empty string for legacy bundles
authored before this field landed, rather than a missing key that decoders
would silently default."

### 2.2 BuildSpawnCommand populates from `resolved.CLIKind`

`internal/app/dispatcher/spawn.go` line 428 (one-line diff):

```go
CLIKind:      string(resolved.CLIKind),
```

Inside the `bundle.WriteManifest(ManifestMetadata{...})` call. The cast is
correct: `dispatcher.CLIKind` is `type CLIKind string` (verified at
`cli_adapter.go` line 29). `resolved` is the `BindingResolved` returned by
`ResolveBinding(rawBinding)` at line 378, which applies the F.7.17 L15
default-to-claude rule for empty CLIKind — confirmed via the spawn_test
table-driven happy path (`bindingKind: ""` → `wantCLIKind: "claude"`).

### 2.3 UpdateManifestPID round-trip preserves CLIKind

`UpdateManifestPID` body (bundle.go lines 412–426) is `ReadManifest →
mutate ClaudePID → writeManifestAtomic` through the same `ManifestMetadata`
struct. Adding `CLIKind` to the struct means it survives the read-mutate-write
cycle without any UpdateManifestPID code change. Pinned by
`TestUpdateManifestPIDPreservesCLIKind` (bundle_test.go lines 759–795):
WriteManifest with `CLIKind: "claude"` → UpdateManifestPID(31415) → ReadManifest
asserts both `ClaudePID == 31415` AND `CLIKind == "claude"`.

### 2.4 Tests cover the 4 scenarios claimed

| Test name                                          | File / line     | Asserts                                                                    |
| -------------------------------------------------- | --------------- | -------------------------------------------------------------------------- |
| `TestBundleWriteManifestPreservesCLIKind`          | bundle_test:653 | Round-trip `"claude"` + on-disk JSON key `cli_kind` literally present.     |
| `TestBundleWriteManifestEmptyCLIKindIsExplicit`    | bundle_test:706 | Empty string round-trips AND JSON key STILL present (no-`omitempty` pin).  |
| `TestUpdateManifestPIDPreservesCLIKind`            | bundle_test:759 | UpdateManifestPID preserves CLIKind verbatim across read-mutate-write.     |
| `TestBuildSpawnCommandPopulatesManifestCLIKind`    | spawn_test:645  | Table: default-empty → "claude", explicit "claude" → "claude". Both decoded struct + raw JSON key checked. |

All four tests follow project conventions: `t.Parallel()` where safe,
`t.Cleanup` for bundle reaping, `errors.Is` predicates where errors are in
scope, named-fields struct literals.

### 2.5 Mage CI green per worklog

Worklog reports `mage ci`: 24/24 packages PASS, 2656/2657 tests PASS (1
unrelated `TestStewardIntegrationDropOrchSupersedeRejected` skip), dispatcher
coverage 75.1% (above 70% gate). The standalone `mage check` failure
(`TestGateKindClosedEnum/invalid_commit` in `internal/templates`) is documented
as pre-existing F.7.13 territory — verified via `git diff HEAD
internal/templates/schema.go` showing schema-level GateKindCommit work +
schema_test.go fixture in F.7.13's `paths`, not F.7.17.6's. Out of scope.

### 2.6 No commit by builder per REV-13

`git log --oneline -5`: HEAD is `0d957b8 feat(dispatcher): inject permission
grants into settings.json render` — unchanged from drop_4c HEAD before
F.7.17.6 started. `git status --porcelain` shows F.7.17.6's four files
modified-but-uncommitted. Compliant.

### 2.7 Sibling F.7.13 attribution boundary

`git status --porcelain` shows F.7.13 territory clean of F.7.17.6 changes:

- F.7.13 files: `internal/templates/schema.go`, `internal/templates/schema_test.go`,
  `internal/app/dispatcher/gate_commit.go` (untracked), `internal/app/dispatcher/gate_commit_test.go` (untracked).
- F.7.17.6 files: `internal/app/dispatcher/bundle.go`, `bundle_test.go`,
  `spawn.go`, `spawn_test.go`.

Zero file-path overlap. Spawn prompt's "do NOT attribute" directive
respected: the F.7.13 territory carries no edits authored by this droplet.

## 3. Findings

None. Every claim in the spawn prompt's verification list is supported by
the code-on-disk evidence above.

## 4. Hylla Feedback

N/A — Hylla artifact `github.com/evanmschultz/tillsyn@main` returned
"enrichment still running" on the `type CLIKind string dispatcher` query;
fell back to `Read` against `cli_adapter.go` (Go file, in-tree) to verify the
type alias. Not a Hylla miss — the artifact is mid-reingest, an expected
state. No ergonomic gripes.

## 5. Convergence Note

QA Proof confirms evidence completeness for all six verification bullets in
the spawn prompt (field+tag without omitempty; doc-comment citations;
spawn.go population from resolved.CLIKind; UpdateManifestPID round-trip;
mage ci green; no commit). QA Falsification (sibling) runs in parallel.

---

## TL;DR

- T1: F.7.17.6 PASS — field added with correct JSON tag (`cli_kind`, no
  omitempty), doc-comment cites REV-4 + master PLAN §5 + F.7.8 consumer.
- T2: `BuildSpawnCommand` populates `CLIKind: string(resolved.CLIKind)` at
  the manifest-write site; UpdateManifestPID round-trip preserves the field
  for free via the existing read-mutate-write pattern.
- T3: 4 new tests pin the contract; mage ci green per worklog (dispatcher
  75.1% coverage); no builder commit per REV-13; F.7.13 territory clean.
