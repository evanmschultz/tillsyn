# DROP_2 — Build QA Falsification

(durable; append `## Droplet N.M — Round K` per QA attempt; NEVER `git rm`d)

## Droplet 2.1 — Round 1

**Verdict:** pass
**Date:** 2026-05-01
**Reviewer:** go-qa-falsification-agent (subagent), Round 1

### Summary

Adversarial attack on the builder's claim that deleting the entire `templates/` package
(`templates/builtin/default-go.json`, `templates/builtin/default-frontend.json`,
`templates/embed.go`, plus auto-removed parent dirs) is clean and `mage ci` is green.
14 attack vectors run; 0 blocking counterexamples constructed; 0 nits worth blocking on.
Local `mage ci` reproduces the builder's reported result (1263 tests / 19 packages /
all coverage ≥ 70.0% / build of `./cmd/till` succeeds / exit 0).

### Attack Vectors and Findings

#### 1. Hidden importer hunt — exhaustive

- `git grep "evanmschultz/tillsyn/templates" -- '*.go'` → empty (exit 1 from grep with no matches).
- `git grep "templates.ReadFile" -- '*.go'` → empty.
- `git grep "templates.Files" -- '*.go'` → empty.
- `git grep -E '"github\.com/evanmschultz/tillsyn/templates"' -- '*.go'` → empty.

REFUTED — no Go importer exists at HEAD. Builder's claim that the package was
runtime-dead is correct.

#### 2. Build-tag-gated importer (linux/freebsd/windows)

- `git grep -n "templates" -- '*.go'` returned only unrelated hits:
  - `internal/adapters/server/mcpapi/instructions_explainer.go:112` — string literal
    "missing templates, kinds, or drift state" (instructional prose).
  - `internal/adapters/server/mcpapi/instructions_tool.go:333` — string literal
    "default templates" (instructional prose).
  - `internal/app/service.go:147,148,149,179,1873,1883` — `sanitizeStateTemplates` /
    `defaultStateTemplates` / `cfg.StateTemplates` — local state-template machinery
    in the lifecycle-state config domain, unrelated to the deleted `templates/` package.
- None of these files carry a `//go:build` constraint that would exclude darwin
  (verified by inspecting hits — they are unconstrained Go source). No
  build-tag-gated importer of the deleted package exists.

REFUTED — no Go file in the tree references `evanmschultz/tillsyn/templates` under
any build constraint.

#### 3. Reflection / plugin / runtime use

- `git grep -nE 'reflect\.[A-Za-z]+.*templates|plugin.*templates|"templates\.Files"|"templates\.ReadFile"'` → empty.
- The deleted package only exposed `Files embed.FS` and `ReadFile(name string)`.
  Both were verified absent from the tree as Go-symbol references AND as string
  literals. No reflection or plugin loader could resolve them.

REFUTED — no runtime/reflection path exists.

#### 4. `go.mod` / `go.sum` orphans

- `git grep -n "templates" -- 'go.mod' 'go.sum'` → empty.
- `templates/` was an internal sub-package of `github.com/evanmschultz/tillsyn`,
  so by construction it would not appear in `go.mod` / `go.sum`. Confirmed empty.

REFUTED — no module-graph orphans.

#### 5. `mage` target file references

- `git grep -n "templates" -- magefile.go` → empty.
- No mage target references the deleted `templates/` directory.

REFUTED.

#### 6. CI workflow references

- `git grep -n "templates" .github/workflows/` → empty (only `ci.yml` and
  `release.yml` exist; neither references `templates/`).

REFUTED — no CI step expects `templates/` to exist.

#### 7. Documentation lies — README.md references

- `README.md:298` and `README.md:309` reference `templates/builtin/default-go.json`
  and `templates/builtin/default-frontend.json` as live-link MD prose.
- Builder explicitly deferred these per `workflow/drop_2/PLAN.md:394` ("the surviving
  MD references are not load-bearing for Drop 2") and the Round 2 dev decision in
  PLAN.md `:426` ("trivial in-section MD edits ... single-sentence / single-phrase
  fix"). The README hits are inside a multi-bullet template-content block where the
  surrounding paragraphs cite the deleted file by name multiple times — exactly the
  ambiguity flagged in `PLAN_QA_FALSIFICATION.md:126`. A single-phrase fix is
  ill-defined; whole-paragraph rewrite is out of scope. Builder correctly deferred
  to Drop 3's full template overhaul.
- These references are MD prose, not load-bearing for `mage ci` or any runtime path.
  A user following README setup steps today would not get a missing-file error from
  the deleted JSON because nothing at runtime reads them (verified in Vector #1).

REFUTED — the README references are doc/prose drift, scheduled for Drop 3
cleanup. They do not break Droplet 2.1's acceptance criteria (which explicitly
allows MD references to stay until Drop 3 cleanup, per PLAN.md:54).

#### 8. `mage ci` re-run

Reviewer ran `mage ci` from `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/`
in this falsification session. Captured tail:

```
Test summary
  tests: 1263
  passed: 1263
  failed: 0
  skipped: 0
  packages: 19
  pkg passed: 19
  pkg failed: 0
  pkg skipped: 0

[SUCCESS] All tests passed
  1263 tests passed across 19 packages.
...
Minimum package coverage: 70.0%.
[SUCCESS] Coverage threshold met
[SUCCESS] Built till from ./cmd/till
```

Per-package coverage observed:

- `internal/adapters/livewait/localipc` — 79.4%
- `internal/adapters/embeddings/fantasy` — 90.6%
- `internal/platform` — 78.0%
- `internal/buildinfo` — 100.0%
- `internal/adapters/auth/autentauth` — 73.0%
- `internal/config` — 76.8%
- `internal/adapters/storage/sqlite` — 75.1%
- `internal/domain` — 79.2%
- `internal/adapters/server/httpapi` — 88.4%
- `internal/tui/gitdiff` — 85.1%
- `internal/adapters/server/mcpapi` — 72.3%
- `internal/app` — 71.5%
- `cmd/till` — 76.6%
- `internal/adapters/server/common` — 73.0%
- `internal/tui` — 70.0%

Every package ≥ 70.0%. Builder's claim reproduced exactly.

REFUTED — `mage ci` is green at HEAD with the deletion staged.

#### 9. State-flip atomicity

- `git diff workflow/drop_2/PLAN.md` shows exactly one diff hunk: line 48,
  `- **State:** todo` → `+ **State:** done`. No other lines in the Droplet 2.1
  block changed. Paths, Packages, Acceptance, Blocked-by are byte-identical to
  the planner-emitted spec.

REFUTED — atomic state flip; planner intent honored.

#### 10. No collateral damage — `git diff --stat`

```
 templates/builtin/default-frontend.json | 1469 ----------------------------
 templates/builtin/default-go.json       | 1585 -------------------------------
 templates/embed.go                      |   16 -
 workflow/drop_2/PLAN.md                 |    2 +-
 4 files changed, 1 insertion(+), 3071 deletions(-)
```

Plus an untracked `workflow/drop_2/BUILDER_WORKLOG.md` (new, expected).

Exactly the 3 deletions + 1-line PLAN.md state flip + the new BUILDER_WORKLOG.md.
No unexpected file change.

REFUTED.

#### 11. Coverage drop below 70%

Reviewer-run `mage ci` confirms every package ≥ 70.0%. The lowest is
`internal/tui` at exactly 70.0% (not regressed by this droplet — the deletion
removed only an unimported `embed.FS`-only package with no test surface to
reshape package-level coverage). Builder's "1263 tests pass, all coverage ≥ 70%"
claim is reproducible.

REFUTED — no coverage regression.

#### 12. Worklog completeness

- `workflow/drop_2/BUILDER_WORKLOG.md` documents:
  - Outcome.
  - Files touched (deletions, including auto-removed parent dirs).
  - Files touched (state-flips).
  - MD edits under carve-out (none, with rationale and PLAN.md line cite).
  - Mage targets run (`mage ci` — green; tests, packages, coverage minimum,
    build, exit code).
  - Design notes (pre-deletion verification, embed semantics, atomic deletion).
  - `## Hylla Feedback` section present (`N/A — task touched non-Go files only`).

Worklog is complete per the implicit Drop 2 droplet template (no formal schema
yet pre-Drop-2 cascade).

REFUTED.

#### 13. AC3 hit-set verification

Builder claimed `git grep "templates/builtin"` returns only MD hits, with
`CLAUDE.md` having zero Go-tree hits and `workflow/drop_2/PLAN_QA_FALSIFICATION.md`
appearing in the hit-set. Re-verified at HEAD:

- `git grep -n "templates/builtin"` returns only MD hits in:
  - `PLAN.md:1605, 1609, 1623` — top-level cascade plan, MD prose.
  - `README.md:298, 309` — MD prose (deferred per #7 above).
  - `workflow/drop_2/PLAN.md:5, 16, 42, 49, 54, 148, 384, 394, 415, 422` — drop
    plan MD.
  - `workflow/drop_2/PLAN_QA_FALSIFICATION.md:126` — drop QA audit-trail MD.
- `git grep -n "templates/builtin" CLAUDE.md` → empty (zero hits — confirmed).
- `git grep -n "templates/builtin" -- '*.go'` → empty (zero Go-tree hits).
- `git grep -n "templates/builtin" -- '*_test.go'` → empty.

Builder's AC3 reading correct: every surviving hit is MD prose, not a Go-tree
reference. AC3 is satisfied.

REFUTED.

#### 14. Forgotten test fixtures

- `git grep -n "templates/builtin" -- '*_test.go'` → empty.
- No `*_test.go` file loads `templates/builtin/*.json` via `os.ReadFile` or any
  string-path I/O.

REFUTED — no test-fixture loader is broken by the deletion.

### Counterexamples

None constructed. All 14 attack vectors REFUTED.

### Verdict Summary

**PASS.** Builder's claim that Droplet 2.1 (delete `templates/` package outright)
is clean and `mage ci` is green is fully verified. Zero Go importers, zero CI
references, zero test fixtures, zero reflection paths, zero magefile references,
zero `go.mod` orphans. State flip is atomic (one line). `git diff --stat` shows
only the expected 3 deletions + 1-line PLAN.md flip + new BUILDER_WORKLOG.md.
Reviewer-run `mage ci` reproduces builder's reported result (1263 tests / 19
packages / coverage ≥ 70.0% / build green / exit 0). README.md lines 298/309
are MD prose drift correctly deferred to Drop 3 per planner carve-out.

No blocking counterexamples. No nits worth blocking on. Droplet 2.1 is ready
for closeout.

## Droplet 2.2 — Round 1

**Verdict:** pass
**Date:** 2026-05-01
**Reviewer:** go-qa-falsification-agent (subagent), Round 1

### Summary

Adversarial attack on the builder's claim that Droplet 2.2 (`internal/domain/role.go`,
`internal/domain/role_test.go`, plus `ErrInvalidRole` sentinel added to
`internal/domain/errors.go`) is clean and `mage ci` is green. 17 attack vectors run;
0 blocking counterexamples constructed; 0 nits worth blocking on. Local `mage ci`
reproduces 1300 tests / 19 packages / coverage ≥ 70.0% / build green. The Droplet
2.2 contribution is +37 tests over Droplet 2.1's 1263 baseline, and `internal/domain`
package coverage rose 79.2% → 79.4%.

### Builder Spec Deviation — Regex Widening

Builder widened the PLAN.md-specified regex `[a-z-]+` to `[a-z0-9-]+` (added `0-9`
to the character class) to admit `qa-a11y` (digit `1`).

**Verdict on the widening: ACCEPTED, JUSTIFIED.**

PLAN.md `:78` specifies the regex as `[a-z-]+`. PLAN.md `:75` (same droplet) also
requires `RoleQAA11y` to round-trip. These two constraints are internally
inconsistent — `qa-a11y` contains a digit, so `[a-z-]+` cannot capture it.
Builder's widening is the minimum-necessary class change to satisfy both
constraints. Uppercase letters remain excluded, so the case-sensitivity contract
(PLAN.md `:80` "case sensitivity (`Role: Builder` should fail since the regex
captures [a-z-]+)") is preserved — `B` is not in `[a-z0-9-]` either.

The `role.go:46-51` doc comment explicitly documents the widening + rationale.

The `IsValidRole`-on-captured-value check at `role.go:92` guarantees no value
outside the closed 9-element `validRoles` set can pass — values that satisfy
the regex but not the enum (e.g. `"123"`, `"---"`, `"abc-"`, `"foobar"`) all
return `("", ErrInvalidRole)`. The widening therefore admits more strings
through the regex stage but cannot admit any non-canonical role through the
overall parser. The closed-set check is the ground truth; the regex is just
a syntactic prefilter.

### Attack Vectors and Findings

#### 1. Regex over-acceptance — `[a-z0-9-]+` admits more than 9 enum values

Constructed candidate strings that pass the regex but should fall through to
`ErrInvalidRole`:

- `"Role: 123"` → regex captures `"123"` → `IsValidRole("123")` = false → `("", ErrInvalidRole)`. ✓
- `"Role: ---"` → captures `"---"` → false → `ErrInvalidRole`. ✓
- `"Role: abc-"` → captures `"abc-"` → false → `ErrInvalidRole`. ✓
- `"Role: -abc"` → captures `"-abc"` → false → `ErrInvalidRole`. ✓
- `"Role: 1"` → captures `"1"` → false → `ErrInvalidRole`. ✓
- `"Role: a"` → captures `"a"` → false → `ErrInvalidRole`. ✓
- `"Role: foobar"` → captures `"foobar"` → false → `ErrInvalidRole`. ✓ (covered by test `role_test.go:113-118`)

The membership gate at `role.go:92` (`if !IsValidRole(candidate)`) uses
`slices.Contains(validRoles, ...)` against a closed 9-element slice. No string
outside that slice can pass — the regex over-acceptance is harmless. Test
`role_test.go:186-190` (`"Role: -"` → `ErrInvalidRole`) explicitly exercises
the regex-passes-but-enum-rejects path.

REFUTED — regex over-acceptance is contained by the closed-set membership check.

#### 2. Regex under-acceptance — all 9 valid values admitted

Mentally traced `[a-z0-9-]+` against each closed-enum value:

- `builder` ✓ — only lowercase letters
- `qa-proof` ✓ — lowercase + hyphen
- `qa-falsification` ✓ — lowercase + hyphen
- `qa-a11y` ✓ — lowercase + digit + hyphen (this is why widening was needed)
- `qa-visual` ✓
- `design` ✓
- `commit` ✓
- `planner` ✓
- `research` ✓

Each value has a dedicated round-trip test at `role_test.go:132-178`. All 9
land in `RoleX` typed constants on parse. REFUTED.

#### 3. Multi-line "first wins" semantics

Code at `role.go:87` uses `roleDescriptionRegex.FindStringSubmatch(desc)` —
returns the FIRST match (single-match API), not `FindAllStringSubmatch`.
Test `role_test.go:102-106`:

```
desc: "Role: builder\nRole: planner"
want: RoleBuilder
```

Asserts first wins. ✓ REFUTED.

#### 4. Mid-paragraph `Role:` rejection

Regex at `role.go:52` is `(?m)^Role:\s*([a-z0-9-]+)\s*$`. The `(?m)` flag
plus `^` anchor means the match must begin at start-of-string OR after `\n`.
A mid-paragraph occurrence like `"Hello Role: builder"` cannot match because
`Role:` is not at the start of any line.

Test `role_test.go:96-100`:

```
desc: "intro paragraph mentioning Role: builder inline\nbut not anchored"
want: Role("")
wantErr: nil
```

Asserts no match. ✓ REFUTED.

#### 5. Trailing whitespace tolerance

Regex line 52 has `\s*$` after the capture group `([a-z0-9-]+)`. The capture
class does NOT include whitespace, so trailing spaces are consumed by `\s*$`,
not by the capture. Test `role_test.go:108-112`:

```
desc: "Role:  builder  "
want: RoleBuilder
```

The captured value is exactly `"builder"` (no surrounding whitespace), which
matches the typed constant `RoleBuilder = "builder"` byte-for-byte. ✓ REFUTED.

#### 6. Tab vs space whitespace

Go's RE2 `\s` character class matches `[\t\n\f\r ]` (per Go regexp/syntax docs
— RE2 follows Perl's `\s` semantics). So `"Role:\tbuilder"` would match the
regex (tab consumed by `\s*` after `Role:`).

This is NOT explicitly asserted by a test case in `role_test.go`. The PLAN.md
spec at `:80` says "whitespace variants (`Role:  builder  ` → `RoleBuilder`)"
— tab is implied by the `\s*` in the spec regex but not enumerated.

Behavior is correct per Go's `\s` semantics. Missing-test-case nit, NOT a
counterexample. Recommend adding a `"Role:\tbuilder"` test case in a future
hardening pass; not blocking for Droplet 2.2.

REFUTED on correctness; minor coverage nit logged below.

#### 7. CRLF line endings

Go's RE2 `(?m)^` anchors to start-of-string or position after `\n`. It does
NOT specifically anchor on `\r\n` boundaries. However, `\s` matches `\r`, so
in a description with `"\r\nRole: builder\r\n"`:

- `(?m)^` matches at position-0 (start of string) and after the first `\n`.
- After `\n`, the regex sees `Role: builder\r\n`. `Role:` matches. `\s*` greedily
  consumes the space. `[a-z0-9-]+` captures `builder`. `\s*` then consumes
  `\r`. `$` matches before the second `\n` (in multiline mode `$` matches
  before `\n` or at end-of-string).

So CRLF input parses correctly — `\s` swallows the `\r`. Verified by mental
trace; no regression.

REFUTED — CRLF works due to `\r ∈ \s`.

#### 8. Empty captured group

`[a-z0-9-]+` uses `+` (one-or-more). Empty cannot match. Even `"Role: "`
(space then nothing) fails because the capture requires ≥1 character.

REFUTED.

#### 9. Case sensitivity

Regex has no `(?i)` flag. `"Role: Builder"` — `B` (capital) is not in
`[a-z0-9-]`. The class fails on the first char of the would-be capture, so
the overall regex fails. Test `role_test.go:120-124`:

```
desc: "Role: Builder"
want: Role("")
wantErr: nil
```

Asserts no match (no error — because `Role:` line was not recognized as a
"Role: line" at all under the strict regex). ✓ REFUTED.

#### 10. `IsValidRole("")` rejection

`role.go:58-60` uses `slices.Contains(validRoles, ...)` against the 9-element
closed slice. Empty string is not in the slice. Returns false.

Test `role_test.go:29`:

```
{name: "empty string is invalid", role: Role(""), want: false}
```

✓ REFUTED.

#### 11. `NormalizeRole` middle whitespace

`NormalizeRole` (`role.go:64-70`) only does `strings.TrimSpace` + `strings.ToLower`.
Internal whitespace is preserved. So `"qa proof"` (space, not hyphen) normalizes
to `"qa proof"` (still invalid — not in the closed enum, which uses `qa-proof`
with hyphen).

This matches the spec at PLAN.md `:77` ("`NormalizeRole(r Role) Role` lowercases
+ trims; returns empty for empty input"). The spec deliberately does not
collapse internal whitespace — that's not a normalization the closed enum
needs.

Test `role_test.go:48-58` covers trim + lowercase + empty + mixed-case-with-
whitespace + whitespace-only → empty. Internal-space-preserved is not
explicitly tested but is the trivial consequence of using only `TrimSpace`.

REFUTED — behavior matches spec; internal whitespace stays put on purpose.

#### 12. Concurrent regex use

`regexp.MustCompile` returns `*regexp.Regexp`. Per the Go stdlib documentation
(`pkg.go.dev/regexp` — verified via memory of the package contract), `*Regexp`
is safe for concurrent use by multiple goroutines after compilation. The
package-level `var roleDescriptionRegex` is initialized once at package init
and never mutated — read-only after init.

`ParseRoleFromDescription` calls `roleDescriptionRegex.FindStringSubmatch(desc)`
— a read-only operation that allocates per-call match storage internally. No
shared mutable state. Safe under concurrent calls.

REFUTED.

#### 13. `mage ci` re-run

Reviewer ran `mage ci` from `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/`.
Tail capture:

```
Test summary
  tests: 1300
  passed: 1300
  failed: 0
  skipped: 0
  packages: 19
  pkg passed: 19
  pkg failed: 0
  pkg skipped: 0

[SUCCESS] All tests passed
  1300 tests passed across 19 packages.

Minimum package coverage: 70.0%.
[SUCCESS] Coverage threshold met
[SUCCESS] Built till from ./cmd/till
```

`internal/domain` coverage: 79.4% (up from Droplet 2.1's 79.2%, consistent
with adding +37 well-covered tests). All packages ≥ 70.0%. Build green.
Exit 0.

Test-count delta: 1263 (post-Droplet-2.1) → 1300 (post-Droplet-2.2) = +37
new tests in `internal/domain`. `mage test-pkg ./internal/domain` reports
exactly 90 tests in that package alone (matches the test-count expectation
implicit in the spec).

REFUTED — `mage ci` is green at HEAD with Droplet 2.2 staged.

#### 14. `go.mod` / `go.sum` impact

`git diff go.mod` → empty. `git diff go.sum` → empty. New file imports:
`regexp` (stdlib), `slices` (Go 1.21+ stdlib), `strings` (stdlib), `errors`
(stdlib, only in test file). Zero new third-party deps.

REFUTED.

#### 15. Test coverage exercises every code path

Traced `ParseRoleFromDescription` branches (`role.go:86-96`):

- **No-match path** (`match == nil` → `return "", nil`):
  - `role_test.go:84-88` — empty description.
  - `role_test.go:90-94` — non-empty description with no `Role:` line.
  - `role_test.go:96-100` — `Role:` mid-paragraph (regex anchor strict).
  - `role_test.go:120-124` — `"Role: Builder"` (case mismatch on capture class).
- **Match-but-invalid-value path** (`!IsValidRole(candidate)` → `return "", ErrInvalidRole`):
  - `role_test.go:113-118` — `"Role: foobar"`.
  - `role_test.go:186-190` — `"Role: -"` (regex passes, enum rejects).
- **Match-and-valid path** (`return candidate, nil`):
  - `role_test.go:132-178` — all 9 valid roles round-trip.
  - `role_test.go:102-106` — first-wins on multi-Role descs.
  - `role_test.go:108-112` — trailing whitespace.
  - `role_test.go:179-184` — Role line embedded in larger desc.

Every branch in `ParseRoleFromDescription` is exercised. Every branch in
`IsValidRole` (member, non-member, empty) is exercised at `role_test.go:20-30`.
Every branch in `NormalizeRole` (trim, lowercase, empty fast-path,
whitespace-only) is exercised at `role_test.go:53-57`.

REFUTED — full path coverage.

#### 16. Test count and Round 1 baseline

`mage test-pkg ./internal/domain` reports `tests: 90`. This is Round 1; there
is no Round 2 yet for Droplet 2.2. The 90 figure becomes the Round-1 baseline
for any future round to defend.

REFUTED (no Round 2 to break — verified the prior-round baseline).

#### 17. `ErrInvalidRole` sentinel declaration

`internal/domain/errors.go:28`:

```go
ErrInvalidRole = errors.New("invalid role")
```

Declared inside the existing `var (...)` block (lines 6-57), exactly matching
the style of every neighbor sentinel (`ErrInvalidKind`, `ErrInvalidLifecycleState`,
etc.). Declared with `var`, not `:=`, not in a function — package-level
sentinel. `errors.Is(err, ErrInvalidRole)` works for downstream callers.

Test `role_test.go:197` uses `errors.Is(err, tc.wantErr)` — confirms sentinel
is exercised through the `errors.Is` API, not via `==`.

REFUTED — sentinel pattern correct.

### Counterexamples

None constructed. All 17 attack vectors REFUTED.

### Minor Nits (Not Blocking)

- **Vector 6 — tab whitespace not explicitly asserted.** `role_test.go` does
  not include a `"Role:\tbuilder"` case. The `\s*` regex correctly handles
  tabs per Go's RE2 `\s` semantics, but the test suite leaves the assertion
  implicit. Recommend adding a tab-whitespace case in a future hardening pass.
  Not blocking — behavior is correct.
- **Vector 11 — internal-whitespace `NormalizeRole` not explicitly asserted.**
  `NormalizeRole(Role("qa proof"))` returning `Role("qa proof")` (preserving
  internal space) is the deliberate consequence of using only `TrimSpace`.
  Not asserted by a dedicated test case but follows from the implementation
  shape. Not blocking.

### Verdict Summary

**PASS.** Builder's claim that Droplet 2.2 (pure `Role` enum + `ParseRoleFromDescription`
parser + `ErrInvalidRole` sentinel) is clean and `mage ci` is green is fully
verified. Spec deviation (regex widening from `[a-z-]+` to `[a-z0-9-]+`) is
necessary and justified — it resolves an internal contradiction in PLAN.md
(`qa-a11y` requires digits in the capture class) and is contained by the
closed-set `IsValidRole` membership check at `role.go:92`. All 17 attack
vectors REFUTED. Local `mage ci` reproduces 1300 tests / 19 packages /
coverage ≥ 70.0% / build green / exit 0. `internal/domain` package coverage
rose 79.2% → 79.4% with 37 new tests. No new third-party deps. Sentinel
error declaration matches the existing var-block style. `errors.Is` semantics
honored by the test suite.

No blocking counterexamples. Two minor coverage nits logged for future
hardening (tab whitespace explicit assertion, internal-whitespace
`NormalizeRole` explicit assertion). Droplet 2.2 is ready for closeout.

## Droplet 2.3 — Round 1

**Verdict:** pass
**Date:** 2026-05-02
**Reviewer:** go-qa-falsification-agent (subagent), Round 1

### Summary

Adversarial attack on the builder's claim that Droplet 2.3 (`Role` field on
`ActionItem` + `ActionItemInput`, `NewActionItem` validation, and the TUI
schema-coverage gate `readOnly` classification) is clean and `mage ci` is green.
14 attack vectors run; 0 blocking counterexamples constructed; 0 nits worth
blocking on. Local `mage ci` reproduces 1313 tests / 19 packages / coverage
≥ 70.0% / build green / exit 0. `internal/domain` rose to 103 tests
(+13 net: 1 new top-level test function `TestNewActionItemRoleValidation`
plus 12 sub-cases) consistent with the +49 LOC test add.

### Attack Vectors and Findings

#### 1. `NewActionItem` validation order — does Role gate get reached?

Traced `NewActionItem` (`action_item.go:97-219`) line-by-line:

- L131-135: Kind empty / invalid → `ErrInvalidKind` (early return).
- L137-143: Scope normalize / default-from-Kind / invalid-applies-to →
  `ErrInvalidKindAppliesTo`.
- L147-149: Scope-mirrors-Kind invariant → `ErrInvalidKindAppliesTo`.
- **L155-158: Role normalize + validate** (NEW BLOCK). Only reached when Kind
  is non-empty AND valid AND Scope is valid AND Scope mirrors Kind.
- L159-164: LifecycleState defaults / validates.

If a caller passes `Kind: ""` and `Role: "foobar"`, they get `ErrInvalidKind`
— NOT `ErrInvalidRole`. This is the standard "first-invalid-wins" semantics
that every other validator in this function follows; there is no
counterexample where a Role-only error would semantically be expected but
is silently elided. The new `TestNewActionItemRoleValidation` cases
(`domain_test.go:213-231`) all set `Kind: KindBuild` + valid scope so they
reach the Role gate as intended.

REFUTED — validation ordering is correct; Role gate is reachable for every
test case that targets it.

#### 2. Empty-role short-circuit correctness

Pattern at `action_item.go:155-158`:

```go
in.Role = NormalizeRole(in.Role)
if in.Role != "" && !IsValidRole(in.Role) {
    return ActionItem{}, ErrInvalidRole
}
```

- `Role: ""` → `NormalizeRole("")` returns `""` (per `role.go:64-70`
  fast-path) → short-circuits `&&`, no validation, falls through, return
  literal sets `Role: ""` (zero value). ✓
- `Role: "   "` (whitespace only) → `NormalizeRole` `TrimSpace` yields `""`
  → fast-path returns `""` → short-circuits `&&`, no validation, falls
  through, `Role: ""`. ✓ (covered by test case `name: "whitespace"`,
  `domain_test.go:220`).
- `Role: "builder"` → normalize returns `"builder"` (already lowercase) →
  `IsValidRole("builder")` true → no error → proceeds. ✓
- `Role: "foobar"` → normalize returns `"foobar"` → `IsValidRole("foobar")`
  false → `ErrInvalidRole` returned. ✓ (covered by test case `unknown
  rejects`, `domain_test.go:230`).

The short-circuit (`!= "" &&`) is required because `IsValidRole("")` returns
false (`role.go:58-60` uses `slices.Contains` against the 9-element closed
slice; empty string is not a member). Without the short-circuit, every empty
role would `ErrInvalidRole`, contradicting the spec.

REFUTED — short-circuit logic is correct for all four input categories.

#### 3. NormalizeRole-then-validate ordering

Builder's pattern `in.Role = NormalizeRole(in.Role)` BEFORE `IsValidRole`
check. Verified:

- Input `Role("  builder  ")` → `NormalizeRole` `TrimSpace` → `"builder"` →
  `ToLower` → `"builder"` → `IsValidRole("builder")` true → passes.
- Without normalize-first: `IsValidRole("  builder  ")` would internally call
  `slices.Contains` after `TrimSpace` + `ToLower` (per `role.go:58-60`), so
  it would still return true. But then `in.Role` would still equal
  `"  builder  "` (un-normalized) when set on the returned struct,
  violating the round-trip contract.

Builder's normalize-first is necessary to ensure `actionItem.Role` byte-equals
the canonical typed constant when the input had surrounding whitespace.
Test case `name: "builder"` (`domain_test.go:221`) uses already-canonical
`RoleBuilder` so it doesn't directly exercise the whitespace-trim path,
but the test wouldn't expose a regression here either; the normalize-first
ordering is correct by inspection of the implementation. Whitespace-only
case (`"   "` → `""`) does exercise normalize.

REFUTED — normalize precedes validate; whitespace handling is correct.

#### 4. `Role` field set on returned struct

Return literal at `action_item.go:195-218` includes `Role: in.Role` at
line 201, AFTER `in.Role = NormalizeRole(in.Role)` at line 155. So the
returned struct's `Role` is the normalized value, not the raw input.
Verified by reading the struct literal field-by-field.

REFUTED — returned struct correctly carries the normalized Role.

#### 5. TUI schema-gate `readOnly` classification correctness

Read `internal/tui/model_test.go:14797-14830`:

- `editable` map: `Title`, `Description`, `Priority`, `DueAt`, `Labels`,
  `Metadata` (6 fields).
- `readOnly` map: `ID`, `ProjectID`, `ParentID`, `Kind`, `Scope`, `Role`
  (NEW), `LifecycleState`, `ColumnID`, `Position`, plus 11 audit/timestamp
  fields (20 fields total).

Comparison against peers:

- `Kind`, `Scope`, `LifecycleState` are all `readOnly` — these are
  closed-enum structural fields set at creation, mutated only via specific
  domain methods (e.g., `SetLifecycleState`), not via free-form TUI form
  editing.
- `Role` is structurally the same shape: closed-enum, set at creation via
  `NewActionItem`, with no mutation method on `*ActionItem` to change it
  later.

Classifying `Role` as `readOnly` is consistent with the peer pattern.
Future Drop 2.5 (MCP plumbing) and Drop 3+ (template-driven role binding)
may permit role to flow in via creation-time payloads from MCP / templates,
but the TUI editable-vs-readOnly classification is about *interactive form
field editing in the TUI*, not about *which subsystems can set the value at
creation time*. So the classification correctly stays `readOnly` even after
MCP plumbing lands.

REFUTED — schema-gate `readOnly` classification is the correct and
peer-consistent choice.

#### 6. Schema-gate test SHOULD fail without the addition

Read `assertExplicitFieldCoverage` at `model_test.go:14984-15015`:

- Iterates `typ.NumField()` → for every exported field, checks
  `classified[field.Name]`; fails the test with `t.Fatalf` if any exported
  field is unclassified.
- Adding `Role` to `domain.ActionItem` without the `readOnly` map entry would
  immediately fail `TestActionItemSchemaCoverageIsExplicit` with:
  `"ActionItem field \"Role\" is not classified for TUI/schema coverage"`.

This confirms: the schema-gate addition was MANDATORY, not optional. The
builder correctly identified the gate dependency and addressed it in the
same droplet.

REFUTED — schema-gate update is required and present; gate would fail
without it.

#### 7. JSON serialization leak

Verified:

- `domain.ActionItem` has NO `json:"..."` struct tags on any field. Without
  tags, default JSON marshaling uses field names verbatim (`"Role": "..."`).
- Searched for direct marshaling of `domain.ActionItem`:
  `rg "json\.Marshal.*ActionItem|json\.Marshal.*\bitem\b|Marshal\(.*ActionItem|json\.Encoder.*ActionItem"`
  → empty.
- The snapshot subsystem (`internal/app/snapshot.go`) defines a separate
  `SnapshotActionItem` struct with explicit `json:"..."` tags on each
  field — adding a field to `domain.ActionItem` does NOT add a key to
  snapshot JSON output. Snapshot would only carry `Role` if the snapshot
  builder explicitly mapped it across (which is a future drop's concern,
  not Droplet 2.3's).
- The MCP surface (`internal/adapters/server/common/mcp_surface.go`,
  `types.go`) does not directly marshal `domain.ActionItem` either.

REFUTED — no JSON serialization leak from adding `Role` to
`domain.ActionItem`. Downstream consumers explicitly project ActionItem
into their own JSON schemas.

#### 8. Zero-value Role == ""

`Role` is `type Role string` (per `role.go:10`). Go's zero value for `string`
is `""`. So `Role` zero value is `Role("")` which equals `""` for all
comparisons. The empty test case `name: "empty"` (`domain_test.go:219`)
exercises this — `input: ""` → `wantRole: ""`.

REFUTED.

#### 9. Test count drift

Reviewer-run `mage test-pkg ./internal/domain` reports `tests: 103`.

Round-1 baseline arithmetic:

- Droplet 2.2 baseline: 90 tests (per `BUILDER_QA_FALSIFICATION.md:541`,
  `mage test-pkg ./internal/domain` reports `tests: 90`).
- Droplet 2.3 adds: 1 new top-level test `TestNewActionItemRoleValidation`
  (sub-cases registered with `t.Run` in a `for _, tc := range cases`
  loop, with 12 sub-cases enumerated at `domain_test.go:219-230`).
- Go's test runner counts each `t.Run` subtests as a distinct test. So
  `TestNewActionItemRoleValidation` contributes 1 (parent) + 12 (subtests)
  = 13 tests. 90 + 13 = 103. ✓

`mage ci` whole-suite delta: 1300 (post-Droplet-2.2) → 1313 (post-Droplet-2.3)
= +13. Matches per-package delta exactly.

REFUTED — test count is correct and arithmetic is consistent across the
whole-suite and per-package counts.

#### 10. `mage ci` re-run

Reviewer ran `mage ci` from `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/`.
Tail capture:

```
Test summary
  tests: 1313
  passed: 1313
  failed: 0
  skipped: 0
  packages: 19
  pkg passed: 19
  pkg failed: 0
  pkg skipped: 0

[SUCCESS] All tests passed
  1313 tests passed across 19 packages.

Minimum package coverage: 70.0%.
[SUCCESS] Coverage threshold met
[SUCCESS] Built till from ./cmd/till
```

Per-package coverage (lowest five):

- `internal/tui` — 70.0% (unchanged, lowest).
- `internal/app` — 71.5%.
- `internal/adapters/server/mcpapi` — 72.3%.
- `internal/adapters/auth/autentauth` — 73.0%.
- `internal/adapters/server/common` — 73.0%.
- `internal/domain` — 79.4% (matches Droplet 2.2 baseline; 13 new
  test runs go through the `NewActionItem` happy-path / `Role`-error
  paths that were already covered by adjacent tests, so the % stays flat
  even with absolute test count rising).

Build green. Exit 0. Builder's claim reproduced exactly.

REFUTED — `mage ci` is green at HEAD with Droplet 2.3 staged.

#### 11. Hidden compile failures from struct literal usage

Searched for `ActionItem{` and `domain.ActionItem{` literal constructions:

- `internal/domain/action_item.go:106-192` — multiple `ActionItem{}`
  zero-value returns. Adding `Role` keeps these as zero literals, so
  `Role` becomes `""` (zero value for `Role string`). No compile break,
  no semantic change.
- `internal/domain/action_item.go:195` — the named-field construction
  inside `NewActionItem` itself; updated by the builder to include
  `Role: in.Role`.
- `internal/domain/domain_test.go:492` — `[]ActionItem{ {ID, Title,
  LifecycleState} }` partial literal. Go allows partial named-field
  literals; `Role` is omitted and defaults to `""`. No compile break.
- `internal/app/kind_capability.go:792` — `&domain.ActionItem{ProjectID,
  Scope}` partial named-field. Same pattern; `Role` defaults to `""`.

`mage ci` green confirms no struct literal anywhere compile-breaks.

REFUTED — Go's named-field struct literals tolerate adding a new field;
no caller breaks.

#### 12. Doc comment on Role field

Both `ActionItem.Role` (`action_item.go:30-33`) and `ActionItemInput.Role`
(`action_item.go:63-67`) carry inline godoc-style comments documenting:

- The optional / closed-enum semantics.
- The empty-zero-value contract.
- The `ErrInvalidRole` rejection behavior on `ActionItemInput`.

CLAUDE.md "Go Development Rules" line `Go doc comments on every top-level
declaration and method` applies to declarations (functions, types,
top-level vars), not to struct fields. Sibling fields like `ID`,
`ProjectID`, `Kind` carry no inline comments either, so the convention
in this package is "doc comment optional on fields." The builder went
above and beyond by adding doc comments anyway. Compliant.

REFUTED — doc comment present and stylistically appropriate.

#### 13. Forvar absence

`rg -n "tc := tc" internal/domain/domain_test.go` → empty. The new
`TestNewActionItemRoleValidation` table-driven loop at
`domain_test.go:233-254` does NOT contain a `tc := tc` line — and
correctly so for Go 1.22+ where the loop variable is per-iteration by
default. (The project is on Go 1.26 per `CLAUDE.md` "Tech Stack".)

REFUTED — no forvar shadow needed; loop variable is per-iteration by
default in Go 1.22+.

#### 14. Validation error wrapping consistency

Inspected the existing error-return style in `NewActionItem` (every
sentinel returned as `return ActionItem{}, ErrInvalidX` — bare, no
`fmt.Errorf("...: %w", ...)` wrap, no context prefix):

- L106: `return ActionItem{}, ErrInvalidID`
- L109: `return ActionItem{}, ErrInvalidID`
- L112: `return ActionItem{}, ErrInvalidParentID`
- L115: `return ActionItem{}, ErrInvalidColumnID`
- L118: `return ActionItem{}, ErrInvalidTitle`
- L121: `return ActionItem{}, ErrInvalidPosition`
- L128: `return ActionItem{}, ErrInvalidPriority`
- L132, L135: `return ActionItem{}, ErrInvalidKind`
- L142, L148: `return ActionItem{}, ErrInvalidKindAppliesTo`
- **L157: `return ActionItem{}, ErrInvalidRole`** (NEW — bare).
- L163: `return ActionItem{}, ErrInvalidLifecycleState`
- L169: `return ActionItem{}, ErrInvalidActorType`

Builder's `return ActionItem{}, ErrInvalidRole` is byte-for-byte
consistent with the surrounding style. The test uses `err != tc.wantErr`
direct comparison (`domain_test.go:244`), which works because the bare
sentinel is `==` to itself. (`errors.Is` would also work but isn't
needed.) Consistent with sibling validators.

REFUTED — error style is consistent with package convention.

### Counterexamples

None constructed. All 14 attack vectors REFUTED.

### Verdict Summary

**PASS.** Builder's claim that Droplet 2.3 (`Role` field on `ActionItem` +
`ActionItemInput`, `NewActionItem` validation block, schema-gate `readOnly`
classification, +1 `TestNewActionItemRoleValidation` table-driven test
covering 12 sub-cases) is clean and `mage ci` is green is fully verified.
All 14 attack vectors REFUTED. Local `mage ci` reproduces 1313 tests / 19
packages / coverage ≥ 70.0% / build green / exit 0 — exactly the builder's
reported result. `internal/domain` rose 90 → 103 tests (+13: 1 parent +
12 sub-cases via `t.Run`), consistent with the whole-suite delta 1300 → 1313.

Validation order is correct: Role gate sits after Kind / Scope mirror
checks, with empty-role short-circuit (`!= "" && !IsValidRole`)
preventing false rejections of the optional-empty zero value. Whitespace-only
input normalizes to `""` and round-trips as the zero `Role` value. The
return literal at `action_item.go:201` correctly sets `Role: in.Role`
post-normalize. Schema-gate `readOnly` classification is peer-consistent
with `Kind` / `Scope` / `LifecycleState` (closed-enum structural fields
not editable via TUI form input). No JSON serialization leak — snapshot
and MCP surfaces use their own projection structs with explicit tags.
No struct-literal callers break (Go's named-field syntax tolerates the
new field; zero-valued `ActionItem{}` returns continue to work). Error
sentinel style matches sibling `ErrInvalidX` returns (bare, no wrap).
No forvar shadow needed (Go 1.22+ per-iteration loop var). Doc comments
on `Role` fields go above and beyond the package's "fields rarely doc'd"
convention.

No blocking counterexamples. No nits worth blocking on. Droplet 2.3 is
ready for closeout.

### Hylla Feedback

N/A — task touched only Go files I read directly via `Read` and `LSP`,
plus shelled `git diff` / `mage ci` / `mage test-pkg`. No Hylla queries
issued during this falsification round; the modified surfaces
(`internal/domain/action_item.go`, `internal/domain/domain_test.go`,
`internal/tui/model_test.go`) are localized and read in full from the
working tree, with `LSP documentSymbol` used to enumerate the
domain_test.go test functions. No Hylla miss to report.
