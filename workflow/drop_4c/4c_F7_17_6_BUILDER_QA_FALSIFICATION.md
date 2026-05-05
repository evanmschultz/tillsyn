# Drop 4c — F.7.17.6 Builder QA Falsification

Droplet: **F.7.17.6 — `Manifest.CLIKind` extension**

Round: **R1**

Verdict: **PASS-WITH-NITS**

## Scope under attack

Builder claim: `ManifestMetadata.CLIKind string` field added with `json:"cli_kind"` (no omitempty); `BuildSpawnCommand` populates `CLIKind: string(resolved.CLIKind)`; `UpdateManifestPID` round-trip preserves CLIKind; 3 new bundle tests + 1 new spawn test all green; `mage ci` clean (24/24 packages, 2656/2657 tests).

Adversarial pass — five attack templates from the spawn prompt.

## Attack 1 — JSON tag `cli_kind` no `omitempty`, explicit empty persisted

**Premises**: Empty CLIKind written via `WriteManifest` must persist as `cli_kind: ""` on disk (key present, value empty), and `ReadManifest` must decode the key back to the empty string.

**Evidence**:
- `internal/app/dispatcher/bundle.go:146` — struct tag is `\`json:"cli_kind"\`` with no `,omitempty`. Confirmed by reading the diff hunk.
- `internal/app/dispatcher/bundle_test.go` — `TestBundleWriteManifestEmptyCLIKindIsExplicit` writes `CLIKind: ""`, decodes via `ReadManifest`, asserts `decoded.CLIKind == ""`, AND re-reads the raw JSON via `os.ReadFile` + `json.Unmarshal` into `map[string]any` and asserts `generic["cli_kind"]` is present AND equals `""`.

**Trace**: `json.MarshalIndent` on a struct field with no `omitempty` always emits the key regardless of value. The test exercises BOTH the struct-decoded view (`decoded.CLIKind`) and the raw-wire view (`generic["cli_kind"]`).

**Conclusion**: REFUTED — counterexample not constructible. Wire format pinned.

**NIT (not a counterexample, but doc-comment imprecision)**: The field-level doc-comment at `bundle.go:142–145` says the no-omitempty tag means "the orphan scan sees an explicit empty string for **legacy bundles authored before this field landed**, rather than a missing key that decoders would silently default." On a strict reading this is wrong: a legacy bundle has no `cli_kind` key at all, so on `json.Unmarshal` into `ManifestMetadata` the field defaults to `""` — same as a current-write of an explicit empty CLIKind. From the **struct-decoded** vantage the two are indistinguishable. The no-omitempty contract DOES guarantee that current writes always emit the key (the wire-format property the test pins) — but it does NOT let the orphan scan distinguish "legacy bundle" from "current empty" without reading the raw JSON bytes. Suggested doc-comment correction: drop the legacy-bundle phrasing and re-frame as "current writes always carry an explicit `cli_kind` key on the wire so non-Go forensic tooling sees the field; struct-level decoders see an empty string for both legacy bundles and current empty-CLIKind writes." Builder may pick this up; not blocking.

## Attack 2 — UpdateManifestPID round-trip preserves CLIKind

**Premises**: After `WriteManifest(CLIKind="claude")` then `UpdateManifestPID(123)`, a subsequent `ReadManifest` must return BOTH `CLIKind="claude"` AND `ClaudePID=123`.

**Evidence**:
- `bundle.go:412–426` (`UpdateManifestPID` body): the helper calls `ReadManifest(b.Paths.Root)` to load the full struct, mutates `metadata.ClaudePID = pid`, re-asserts `metadata.BundlePath = b.Paths.Root`, then funnels through `writeManifestAtomic`. Every other field — including the new `CLIKind` — is preserved verbatim through the read-decode-write cycle.
- `bundle_test.go` — `TestUpdateManifestPIDPreservesCLIKind` writes `CLIKind="claude"`, calls `UpdateManifestPID(31415)`, re-reads, asserts both `decoded.ClaudePID == 31415` AND `decoded.CLIKind == "claude"`.
- `bundle_test.go:645–...` — pre-existing `TestUpdateManifestPIDPreservesOtherFields` covers the broader contract; the new test extends specifically to CLIKind.

**Trace**: read-mutate-write through a single `ManifestMetadata` struct preserves all fields by construction. No partial-merge code path exists — the helper rewrites the entire JSON payload from the freshly-decoded struct.

**Conclusion**: REFUTED — round-trip is sound and pinned by a regression test.

## Attack 3 — F.7.8 consumer alignment (string vs typed)

**Premises**: F.7.8's orphan scan (REV-5: `blocked_by F.7.17.6`) reads `Manifest.CLIKind` to route adapter-specific liveness checks. The shape F.7.8 expects must match what F.7.17.6 ships.

**Evidence**:
- `workflow/drop_4c/F7_CORE_PLAN.md:566` — F.7.8 acceptance criterion: `adapterRegistry.Get(manifest.CLIKind).IsPIDAlive(pid)`. The expression takes `manifest.CLIKind` and uses it as a map key into the adapter registry.
- `internal/app/dispatcher/spawn.go:251` — `adaptersMap` is `map[CLIKind]CLIAdapter`. Lookup happens via `lookupAdapter(kind CLIKind)`.
- `bundle.go:146` — `CLIKind string` (note: declared as `string`, NOT `dispatcher.CLIKind` the typed alias).

**Trace / sub-attack**: Manifest holds a plain string. F.7.8 must convert to `dispatcher.CLIKind` (or call the lookup with an explicit cast like `lookupAdapter(CLIKind(manifest.CLIKind))`). This is a one-line cast — not a counterexample. Two design options were available to F.7.17.6:
  1. `CLIKind dispatcher.CLIKind \`json:"cli_kind"\`` — typed; requires F.7.8 to import dispatcher's `CLIKind` alias from the same package (already does).
  2. `CLIKind string \`json:"cli_kind"\`` — plain string; F.7.8 casts at the lookup site.

The builder picked (2). The trade-off is that the manifest schema stays decoupled from the dispatcher's enum, which arguably loosens type-safety at the F.7.8 boundary but matches the cross-CLI manifest "wire shell" framing in `bundle.go:101–110`. The worklog (line 29) explicitly cites `string(resolved.CLIKind)` as a "no-op cast that pins type-safety against silent CLIKind→string drift" — but the field on the **struct** itself is plain `string`, so the type-safety pin only protects the spawn.go write path, not the F.7.8 read path. F.7.8 will need to cast `CLIKind(manifest.CLIKind)` at the registry lookup; that's a 1-token addition, not a counterexample.

**Conclusion**: REFUTED — F.7.8's expected shape (`manifest.CLIKind` usable as a map key after a string→CLIKind cast) is satisfiable. NIT possibility: a future refinement could promote the field to typed `CLIKind` with a custom `UnmarshalJSON` that re-validates the closed enum on read. Out of F.7.17.6 scope per the spawn prompt's "F.7.13 parallel sibling — do NOT attribute templates schema work" carve-out.

## Attack 4 — Spawn population when resolved.CLIKind is empty

**Premises**: At `spawn.go:428`, the manifest write does `CLIKind: string(resolved.CLIKind)`. If `resolved.CLIKind` could ever be empty, the manifest would record `""` and the orphan scan would have no adapter routing key. Per L11/L15 default-to-claude rule, `ResolveBinding` MUST substitute `CLIKindClaude` when rawBinding.CLIKind is empty.

**Evidence**:
- `internal/app/dispatcher/binding_resolved.go:117–131` — `ResolveBinding` body:
  ```
  resolved := BindingResolved{
      ...
      CLIKind: CLIKind(rawBinding.CLIKind),
      ...
  }
  if resolved.CLIKind == "" {
      resolved.CLIKind = CLIKindClaude
  }
  ```
  The substitution fires unconditionally when the raw value is empty, BEFORE the function returns. There is no override-layer code path that can re-empty CLIKind (overrides cover Model/Effort/MaxTries/MaxBudgetUSD/MaxTurns/AutoPush/BlockedRetries/BlockedRetryCooldown only — see the `BindingOverrides` field list at `binding_resolved.go:44–79`).
- `cli_adapter.go:33` — `CLIKindClaude CLIKind = "claude"`. The default value is the non-empty string `"claude"`.
- `spawn.go:378–383` — after `ResolveBinding`, BuildSpawnCommand looks up the adapter via `lookupAdapter(resolved.CLIKind)`. An empty CLIKind would surface as `ErrUnsupportedCLIKind` BEFORE the manifest write at line 424, so even a hypothetical bypass of L15 would short-circuit on adapter lookup rather than producing an empty-CLIKind manifest.
- `spawn_test.go` — `TestBuildSpawnCommandPopulatesManifestCLIKind` table-drives both `bindingKind=""` (default empty resolves to claude) and `bindingKind="claude"` (explicit pass-through). Both assert manifest CLIKind == "claude".

**Trace**: rawBinding.CLIKind="" → ResolveBinding substitutes CLIKindClaude → resolved.CLIKind="claude" → string(resolved.CLIKind)="claude" → manifest writes "claude". L15 rule holds end-to-end.

**Conclusion**: REFUTED — no path to an empty manifest CLIKind exists in the pipeline. The cast `string(resolved.CLIKind)` is correct and the test pins both the default-empty and explicit-claude paths.

## Attack 5 — Memory rule conflicts

**Premises**: Builder must not (a) call Hylla, (b) commit, (c) introduce migration logic.

**Evidence**:
- `4c_F7_17_6_BUILDER_WORKLOG.md:54–56` — explicit "N/A — this droplet touched only Go files inside `internal/app/dispatcher/`, but no Hylla queries were issued. The spawn prompt explicitly forbade Hylla calls."
- `4c_F7_17_6_BUILDER_WORKLOG.md:7,44–45` — "Status: complete (DO NOT COMMIT — per F.7-CORE REV-13 / spawn prompt)" and "NO commit by builder. — Confirmed; nothing staged or committed."
- `git status` (per gitStatus snapshot at the top of this session): `bundle.go`, `bundle_test.go`, `spawn.go`, `spawn_test.go` show as `M` modified but unstaged — consistent with "no commit by builder."
- No SQL files, no `internal/adapters/storage/sqlite/` edits, no `cmd/till migrate` invocation, no schema-version bump in the diff.

**Trace**: Each rule independently satisfied.

**Conclusion**: REFUTED — no memory-rule violations.

## Cascade-vocabulary attacks (post-Drop-3)

Not applicable — F.7.17.6 is a `build` droplet, not a `plan` / `confluence` / `segment`. No structural-type / role contradictions to check on a single build droplet's output.

## §4.4 global L1 plan-QA sweep

Not applicable — this is a build-QA-falsification, not a plan-QA-falsification at L1.

## Counterexamples

None CONFIRMED.

## Summary

**Verdict: PASS-WITH-NITS.**

- All 5 attacks REFUTED.
- 1 NIT on the doc-comment at `bundle.go:142–145` — the "legacy bundle" framing of the no-omitempty rationale is imprecise. A legacy bundle (key absent in JSON) and a current empty-CLIKind write decode to identical `""` values via `json.Unmarshal` into `ManifestMetadata`; the no-omitempty contract is a wire-format guarantee for non-Go forensic tooling, not a struct-decoded distinguisher. Suggested rephrase noted in Attack 1. NIT only — does not block PASS, builder may pick up in a follow-up.

Build verification (per worklog line 33): `mage ci` green, dispatcher package coverage 75.1% (above 70% gate), all 4 new tests pass, no test regressions.

The pre-existing `internal/templates` test issue mentioned in worklog line 34 is out of F.7.17.6's `paths` and unrelated to the field addition. (Note: a current read of `schema_test.go:175–191` shows no `invalid_commit` subtest in the `invalidCases` slice — the worklog's "pre-existing failure" claim may be stale, but this is templates-package territory and not in scope for this droplet's QA.)

## Hylla Feedback

None — this falsification reviewed Go source already on disk under `internal/app/dispatcher/` plus the in-tree plan MD at `workflow/drop_4c/F7_CORE_PLAN.md`. Hylla was queried once for `ResolveBinding` / `CLIKindClaude` symbol context; the result set did not include the dispatcher's binding_resolved.go (Hylla index is stale for files modified mid-drop, which is the documented policy — fall back to `Read` for changed-since-ingest content). Not a Hylla bug; expected behavior for an in-flight drop.

## TL;DR

T1 — Verdict PASS-WITH-NITS; all 5 attacks REFUTED.
T2 — One doc-comment NIT at `bundle.go:142–145` on the no-omitempty rationale framing — does not block.
T3 — F.7.8 consumer alignment is satisfiable via a one-token `CLIKind(manifest.CLIKind)` cast at the registry lookup site; no schema mismatch.
