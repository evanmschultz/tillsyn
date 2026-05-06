# DROP_4c.5 — Builder QA Falsification

Append a `## Droplet <ID> — Round K` section per QA falsification round. See `workflow/example/drops/WORKFLOW.md § "Phase 5 — QA"` for what each section should contain.

## Droplet E.1 — Round 1

**Reviewer:** go-qa-falsification-agent
**Date:** 2026-05-05
**Verdict:** PASS

### Attack Inventory

**Attack 1 — `slices.Equal` semantic shift (order-blind → order-strict).** Builder swapped 8 call sites in `locks_file_test.go` (and mirror in `locks_package_test.go`) from sort-then-compare `equalStringSlices` to `slices.Equal`. Audited each call site against current impl behavior:

- `TestFileLockReleaseFreesAllPathsHeldByItem:78` — asserts post-Release reacquire of `["a","b","c"]` returns `["a","b","c"]`. Impl appends in input order → assertion is order-preserving against input-order impl. The pre-swap sort-then-compare accidentally weakened the test (would have passed even if impl returned `["c","b","a"]`). The swap aligns the test to the *documented* contract (input-order preservation, locks_file.go:70-76).
- `TestFileLockAcquirePartialConflictReturnsConflicts:99,113` — `["a"]` and `["a","c"]`; same direction.
- `TestFileLockConcurrentAcquireRaceFree:192` — single-element `[path]`; order is moot.
- `TestFileLockPathsAreOpaque:215`, `TestFileLockEmptyInputsAreNoOps:263`, `TestFileLockZeroValueIsUsable:284,298` — single-element checks; order is moot.

**Verdict:** No call site relied on order-blindness. The swap *strengthens* the assertions to match the documented input-order contract. The deletion of `equalStringSlices` removes a misleading helper that didn't reflect Acquire's real behavior. Aligned, not silently shifted.

**Attack 2 — Duplicate-input cross-probe inconsistency.** New test `TestFileLockManagerAcquireDuplicateInputIdempotent` probes whether the internal `holders` map collapsed correctly:

- Lines 380-393: item-2 calls `Acquire("item-2", ["a","b"])` against the post-`["a","a","b"]`-acquire state.
- Asserts `len(conflicts2) == 2` (line 384) AND each conflict maps to `item-1` (lines 388-393).
- This pins exactly the collapse behavior the attack named — the duplicate "a" did NOT create two "holders" of "a"; the cross-probe sees one conflict per distinct key.
- Lines 397-407 add a Release+reacquire round to confirm no stray holder leak.

**Verdict:** Cross-probe assertion is explicit and tight. Mirrored test in `locks_package_test.go:419-446` does the same. Mitigated.

**Attack 3 — Mirror correctness (path-level vs package-level semantics).** Compared `locks_file.go` and `locks_package.go` byte-by-byte structurally:

- `locks_file.go:78-87` "Duplicate-input semantics" matches `locks_package.go:93-102` paragraph-for-paragraph with `s/path/package/` + `s/itemPaths/itemPackages/` substitutions. Confirmed structural identity.
- The "one Go package = many files" distinction is correctly handled UPSTREAM (walker / conflict detector per `locks_package.go:14-22`); the lock manager itself is just a per-key holder map. Both managers genuinely have identical lock semantics — the cross-axis policy ("if any file in package P is path-locked, treat P as effectively locked too") is documented as living in the walker, not in either lock manager.
- `TestPackageLockIndependentFromFileLock` (locks_package_test.go:316-339) regression-protects the "two independent maps" claim.

**Verdict:** Mirror is structurally correct, not sloppy. Mitigated.

**Attack 4 — Helper-removal blast radius.** Audited via `rg "equalStringSlices" --type=go`:

- `internal/templates/load_test.go` — defines its OWN `equalStringSlices` (lines define it locally; package `templates`, not `dispatcher`).
- `internal/adapters/storage/sqlite/repo.go` — defines its OWN `equalStringSlices` (package `sqlite`, not `dispatcher`).
- `internal/app/dispatcher/` — ZERO residual references after E.1 (`rg "equalStringSlices" internal/app/dispatcher/` returns empty).

The dispatcher's deleted helper was package-local to `dispatcher`. The two surviving definitions in other packages are independent test/prod helpers with their own scope. Deletion did NOT cross package boundaries.

**Verdict:** Blast radius cleanly bounded to `internal/app/dispatcher/`. Mitigated.

**Attack 5 — Doc-comment vs impl drift.** Traced the impl loop (locks_file.go:118-134) against the doc-comment claim (locks_file.go:78-87):

- `Acquire("item-1", ["a", "a", "b"])` on empty manager.
- iter 1 (`path="a"`): `holders["a"]` absent → `taken=false` → fall-through to write branch; sets `holders["a"]="item-1"`, inserts `"a"` into `itemPaths["item-1"]` set, appends `"a"` to `acquired`. State: `acquired=["a"]`.
- iter 2 (`path="a"`): `holders["a"]="item-1"` → `taken=true && holder=="item-1"` → conflict branch SKIPPED (because `holder != actionItemID` is false); fall-through to write branch; re-sets `holders["a"]="item-1"` (no-op write), re-inserts `"a"` into `owned` map (no-op since map key exists), appends `"a"` to `acquired`. State: `acquired=["a","a"]`.
- iter 3 (`path="b"`): fresh acquire; appends. State: `acquired=["a","a","b"]`.

Doc-comment claim: "each occurrence appears in acquired in its original input position" — matches. "holders[path] and itemPaths[id][path] end identical to the de-duplicated case (one entry each)" — matches (map writes are idempotent on identical key+value).

**Verdict:** Doc matches impl exactly. No drift. Mitigated.

**Attack 6 — Empty-input + duplicate-empty-string `["", ""]`.** Per `locks_file.go:24-29` "Path opacity: paths are treated as opaque strings. The manager does NOT normalize or canonicalize them." Empty string is a valid opaque key. Tracing `["", ""]` through the impl: same path as `["a", "a"]` mechanically (key just happens to be ""). `acquired=["", ""]`, holders[""]="item-1", one entry in itemPaths set.

The new tests don't cover this explicit case, but:
- Existing `TestFileLockEmptyInputsAreNoOps` covers the zero-length slice case (`nil` and `[]string{}`), which IS the documented edge case.
- Empty-string-as-path is a degenerate input that callers don't produce (planner-side path normalization upstream); not a documented contract surface.
- The semantics are derivable from "paths are opaque strings"; no behavior gap to exploit.

**Verdict:** Out of E.1's documented surface (the spec defines empty-input edge-case as zero-length, not empty-string-element). Coverage gap is theoretical, not a counterexample. Note as a NIT below if desired, but not a falsification.

**Attack 7 — Race-detector regression.** The new `TestFileLockManagerAcquireDuplicateInputIdempotent` and `TestFileLockManagerAcquirePreservesInputOrder` are single-goroutine tests; they don't add concurrent surface area. Existing `TestFileLockConcurrentAcquireRaceFree` (preserved unchanged) continues to cover concurrent Acquire under `-race`. Builder's mixed-conflict scenario `["b","x","a","y"]` against pre-held `a+b` runs in the SAME goroutine sequentially — no concurrency added.

**Verdict:** No new race surface; existing race coverage preserved. Mitigated.

**Attack 8 — Performance regression on duplicate input.** `Acquire("item-1", ["a"]*1000)` would loop 1000 times under one mutex acquisition, each iter doing two map lookups + writes. O(n) where n = input length. For cascade-droplet usage (planner-declared paths/packages, typically ≤20), this is irrelevant. The doc-comment explicitly says "Acquire does NOT dedupe the input. Callers that want a deduped acquired slice MUST dedupe paths before calling" — pushing the dedupe responsibility upstream is documented.

**Verdict:** No regression introduced; behavior is as-documented and acceptable for the use case. Not a falsification.

### Counterexamples (CONFIRMED)

None. All 8 attacks REFUTED.

### Mitigated Attacks

- A1: Call-site audit of the 8 swap sites confirms all were already order-strict in spirit (literal-position assertions); swap aligns helper to documented contract.
- A2: Cross-probe via item-2 in `TestFileLockManagerAcquireDuplicateInputIdempotent:380-393` explicitly pins the collapse property.
- A3: Mirror compared paragraph-by-paragraph; structurally identical with appropriate vocabulary substitution.
- A4: `rg` confirms zero residual `equalStringSlices` usage in `internal/app/dispatcher/`; the two surviving definitions in `internal/templates/load_test.go` and `internal/adapters/storage/sqlite/repo.go` are independent package-local helpers untouched by E.1.
- A5: Impl-loop trace matches doc-comment claim character-for-character.
- A6: Empty-string-as-path follows directly from documented opacity contract; not a new behavior surface.
- A7: No new concurrent surface; existing `TestFileLockConcurrentAcquireRaceFree` preserved.
- A8: O(n)-on-duplicates documented; dedupe responsibility explicitly pushed to caller.

### Observations (non-blocking)

- **Test-name prefix inconsistency.** Existing tests use `TestFileLock...`; new tests use `TestFileLockManager...`. Builder-worklog noted this as spec-mandated (spawn prompt named the new tests verbatim). Cosmetic, not a defect.
- **`acquired = make([]string, 0, len(paths))` capacity hint** (locks_file.go:117, locks_package.go:132) over-allocates when conflicts exist. Negligible (capacity is a hint, not a length); not a regression.
- **Empty-string-as-path coverage** (Attack 6) is implicit from opacity contract but not explicitly tested. If a future droplet wants to harden the opacity surface, a `["", ""]` test would slot in. Not E.1's scope.

### Conclusion

PASS. Builder E.1 cleanly executed the spec:
1. Doc-comments at `locks_file.go:70-87` + `locks_package.go:85-102` pin Input-order and Duplicate-input semantics with structurally identical wording.
2. Test-helper swap from sort-then-compare `equalStringSlices` to `slices.Equal` strengthens 8 existing assertions to match the documented input-order contract; deleted helper has zero residual users in `internal/app/dispatcher/`.
3. New tests `TestFileLockManagerAcquirePreservesInputOrder` + `TestFileLockManagerAcquireDuplicateInputIdempotent` (and package mirrors) pin both contracts with explicit assertions including the cross-probe that demonstrates internal-state collapse.
4. Doc-comment claims match impl trace exactly.
5. Mirror between `locks_file.go` / `locks_package.go` is paragraph-for-paragraph correct.
6. `mage testPkg ./internal/app/dispatcher` 354/354 PASS per builder worklog.

No counterexamples constructed. No CONFIRMED falsifications. Recommend droplet E.1 closes.

### Hylla Feedback

N/A — droplet edits + reviewed surface are Go files, but the spawn-prompt directive ("NO Hylla calls") routed all evidence through `Read` / `Bash` (`rg`, `git diff`, `git show`). No fallback misses to log under the standard rule because the rule was suspended for this round.

## Droplet F.2.1 — Round 1

**Reviewer:** go-qa-falsification-agent
**Date:** 2026-05-05
**Verdict:** PASS

### Attack Inventory

1. **Body-content drift past header** — NO (mitigated). `git diff -M HEAD -- internal/templates/builtin/` reports `similarity index 98%, rename from internal/templates/builtin/default.toml to internal/templates/builtin/default-go.toml`. Only the first 7 header lines changed (rebadge "Tillsyn default" → "Tillsyn Go default" + 4 inserted cross-reference lines about F.2.1/F.1.3/F.2.2 successors). Body content from line 9 onward is byte-identical.
2. **Embed directive completeness** — NO (mitigated). `internal/templates/embed.go:26` carries the only `//go:embed` directive in the package (`builtin/default-go.toml`). `DefaultTemplateFS` is the package's only `embed.FS`. No other code in `internal/templates/` consumes embedded TOML. The single-directive form correctly extends to F.2.2 via explicit-list addition.
3. **`LoadDefaultTemplate()` callers + `default.toml` literal drift** — NO for build/runtime correctness (mitigated). `rg -l LoadDefaultTemplate --type=go` returns 4 files: `internal/templates/embed.go`, `internal/templates/embed_test.go`, `internal/app/auto_generate_steward.go`, `internal/app/service.go`. Only `auto_generate_steward.go:44` is a live call site; `service.go:425` is a doc-comment reference. Both preserved unchanged. Zero compiled-string-literal hits on `"default.toml"` anywhere in `internal/`. Doc-comment drift exists but is non-load-bearing (see "Out-of-Scope Findings" below).
4. **Test rename CI/hooks/mage references** — NO (mitigated). `rg TestDefaultTemplateLoadsCleanly` shows zero hits in `.githooks/`, `magefile.go`, `.github/`, or any current Go file. Only doc-comment "renamed from" references remain (intentional). Old name appears only in historical workflow MDs (`workflow/drop_3/`, `workflow/drop_4b/`, `workflow/drop_4c/`) which are intentional audit trail. Renamed `TestDefaultTemplateGoLoadsCleanly` is the only live test function and runs in the green 380/380 suite.
5. **Header comment vs `templates.Load` parser tolerance** — NO (mitigated). New header is pure `#`-prefixed comment lines (1-38). TOML allows arbitrary leading comments; the canary test `TestDefaultTemplateGoLoadsCleanly` confirms `Load(default-go.toml)` returns nil error in the green test suite.
6. **F.2.2 future-extension shape** — NO (mitigated). `embed.go:26` uses an EXPLICIT single-file directive (`//go:embed builtin/default-go.toml`), NOT a glob. Per F.2.1 falsification mitigation #2, this is the correct form — F.2.2 extends the directive to a two-file explicit list (`//go:embed builtin/default-go.toml builtin/default-generic.toml`) without picking up stray fixtures. Builder worklog confirms this intent.
7. **Caller-audit gap (more than 2 callers?)** — NO (mitigated). Audit via `rg -l LoadDefaultTemplate --type=go` is exhaustive: 2 production callers (`auto_generate_steward.go:44` live, `service.go:425` doc-comment) + the test file's 4 internal references. No indirect callers via mocks. Worklog's caller count is accurate.
8. **Backward-compat for adopter shadowing** — NO (mitigated). Per pre-MVP rules (no external adopters; per F.2.1 falsification mitigation F3 in THEME_F_PLAN.md line 179), this is the intended scope. The rename's "breaking change" status is implicit-by-context — it does not require an explicit worklog callout because the entire Theme F charter assumes the rename. Documented in the spec.

### Counterexamples (CONFIRMED)

None.

### Out-of-Scope Findings (not F.2.1 rework, route to Theme F follow-up)

- **OS1 — stale `default.toml` doc-comment references in 9 places.** `rg "default\.toml" --type=go` returns hits in: `internal/app/auto_generate_steward.go:35-36`, `internal/app/service.go` (line ~427 doc-comment context), `internal/app/kind_capability.go`, `internal/app/kind_capability_test.go` (×3), `internal/app/kind_capability_catalog_test.go`, `internal/adapters/storage/sqlite/repo.go`, `internal/templates/catalog_test.go:16`, `internal/templates/child_rules_test.go:26`, `internal/templates/nesting_test.go:47`. None compile to a string literal; all are doc-comment prose. Per the spawn prompt's strict file-gating rule, these are out of F.2.1's declared file set (only `internal/templates/embed.go` and `internal/templates/embed_test.go` were declared touch points in `internal/`). Recommend Theme F.2.4 caller-audit droplet sweep these to `default-go.toml` (or to language-aware `default-<lang>.toml` phrasing once F.1.3 lands). NOT a counterexample against F.2.1; not a build/runtime correctness issue.
- **OS2 — `nesting_test.go:47` doc-comment "MUST NOT load default.toml" is now anachronistic.** With `default.toml` deleted and replaced by `default-go.toml`, the comment "this fixture-based test MUST NOT load default.toml" reads strangely. Out of F.2.1 scope. Same Theme F.2.4 sweep target as OS1.
- **OS3 — Worklog mismatch on git operation type.** Worklog (line 28) claims `git mv` was used for the rename, with status output showing `R  default.toml -> default-go.toml`. Actual current `git status --porcelain` shows `A  default-go.toml` + ` D  default.toml` (separate add + delete, NOT a tracked rename). However, `git diff -M HEAD` correctly auto-detects the rename via 98% similarity, so functionally indistinguishable for reviewers using `-M`. Cosmetic worklog inaccuracy; no behavioral consequence.

### Mitigated Attacks (citations)

- A1 mitigated by `git diff -M` 98%-similarity output → body byte-identical past header.
- A2 mitigated by reading `embed.go` directly: only one `embed.FS` declaration in the package.
- A3 mitigated by `rg -l LoadDefaultTemplate --type=go` enumeration: 4 files, no surprises; zero compiled-string `"default.toml"` literals.
- A4 mitigated by `rg TestDefaultTemplateLoadsCleanly` showing zero infra hits + 380/380 mage-pkg pass.
- A5 mitigated by canary test `TestDefaultTemplateGoLoadsCleanly` PASS in green suite.
- A6 mitigated by reading `embed.go:26` directly: explicit-file form, not glob.
- A7 mitigated by exhaustive `rg -l` audit (4 files total, all named in worklog).
- A8 mitigated by the spec's own pre-MVP-no-adopter premise (THEME_F_PLAN.md line 179).

### Conclusion

PASS. F.2.1 holds against all 8 required attack categories within its declared file set (`internal/templates/builtin/default-go.toml`, `internal/templates/builtin/default.toml` deletion, `internal/templates/embed.go`, `internal/templates/embed_test.go`, `workflow/drop_4c_5/THEME_F_PLAN.md`, `workflow/drop_4c_5/BUILDER_WORKLOG.md`). The acceptance criteria from THEME_F_PLAN.md § "Droplet F.2.1" lines 158-166 are satisfied:

1. `default-go.toml` exists with byte-identical body + header rebadge — confirmed via `git diff -M`.
2. `default.toml` no longer exists in tree — confirmed via `git status` (` D` entry).
3. `//go:embed builtin/default-go.toml` directive (explicit-file form, F.2.2-extensible) — confirmed at `embed.go:26`.
4. `LoadDefaultTemplate()` API preserved, opens new path — confirmed at `embed.go:55-62`.
5. Caller audit complete (2 live callers, both unchanged) — confirmed via `rg -l`.
6. `mage testPkg ./internal/templates` 380/380 PASS — reported in worklog, structurally consistent with the test surface readable here.

Three out-of-scope findings (OS1/OS2/OS3) routed to Theme F.2.4 caller-audit droplet rather than F.2.1 rework. None block F.2.1.

### Hylla Feedback

N/A — F.2.1 touched non-Go files (TOML + MD) plus minimal Go embed-package edits. Per CLAUDE.md "Hylla Indexes Only Go Files Today" + the spawn prompt's "NO Hylla calls" directive, all evidence resolved via `Read` / `rg` / `git diff -M` / `git status`. No Hylla query was attempted, so no miss to log.
