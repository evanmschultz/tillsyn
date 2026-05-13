# W5.D5 — BUILD-QA-FALSIFICATION Verdict

**Date:** 2026-05-13
**Reviewer:** go-qa-falsification-agent (opus)
**Overall verdict:** PASS

## Attack Hypotheses Tested

### Attack 1 — Passive struct discipline (no Init / Update)

**Hypothesis:** Either Header or Footer has a forbidden `Init()` or `Update()` method.

**Evidence:**
- `internal/tui/components/header.go` line 25 (`NewHeader`), line 35 (`(h Header) WithWidth`), line 45 (`(h Header) View`). No `Init` or `Update` declared.
- `internal/tui/components/footer.go` line 25 (`NewFooter`), line 34 (`(f Footer) WithWidth`), line 42 (`(f Footer) View`). No `Init` or `Update` declared.
- Total methods on `Header`: 2 (`WithWidth`, `View`). Total methods on `Footer`: 2 (`WithWidth`, `View`). Plus 2 free constructors.

**Verdict:** REFUTED. No `Init` or `Update` on either type.

### Attack 2 — Value receiver semantics for WithWidth

**Hypothesis:** `WithWidth` is on a pointer receiver and silently mutates the original instead of returning a copy.

**Evidence:**
- `header.go:35` — `func (h Header) WithWidth(w int) Header` — value receiver. Body assigns to local `h.width`, returns `h`. Caller's original is untouched.
- `footer.go:34` — `func (f Footer) WithWidth(w int) Footer` — value receiver. Same pattern.
- `header_test.go:25-34` (`TestHeaderWithWidth`) and `footer_test.go:23-32` (`TestFooterWithWidth`) explicitly assert `orig.width` is unchanged after `orig.WithWidth(...)`.
- `mage test-pkg ./internal/tui/components` → 57/57 PASS, including the round-trip immutability tests.

**Verdict:** REFUTED. Value-receiver copy semantics confirmed in source AND verified by test.

### Attack 3 — Migration marker placement (all 4 files)

**Hypothesis:** Migration marker is missing or misplaced (must be file-level, before `package components`).

**Evidence:** Marker placement confirmed by direct Read of every file:
- `header.go:1` → `// MIGRATION TARGET: github.com/hylla-org/lykta` (before package decl at line 5).
- `footer.go:1` → identical marker (before package decl at line 5).
- `header_test.go:1` → identical marker (before package decl at line 2).
- `footer_test.go:1` → identical marker (before package decl at line 2).

All 4 files compliant.

**Verdict:** REFUTED. Markers present, file-level, before `package components` in every D5-authored file.

### Attack 4 — Width math underflow / overflow

**Hypothesis:** `gap := h.width - leftW - rightW` underflows to a negative when title+subtitle exceed `h.width`, causing `strings.Repeat(" ", gap)` to panic.

**Evidence:**
- `header.go:55-58` clamps `gap < 0` to `gap = 0` BEFORE calling `strings.Repeat`. `strings.Repeat(" ", 0)` returns `""` safely.
- `TestHeaderView_WidthSmallerThanContent` (`header_test.go:60-66`) constructs `NewHeader("LongTitle", "LongSubtitle", 5)` (combined content ≈ 21 cols vs width=5). Test passes — no panic, title and subtitle both rendered intact.
- I attempted to reason about `int` overflow (`h.width = math.MinInt`): even at `math.MinInt - leftW - rightW`, Go integer subtraction wraps silently to a large positive, but the clamp at line 56 still catches negative values. For the wraparound case, `strings.Repeat` would attempt allocation of MaxInt bytes and panic with "out of memory" — but `h.width` is an `int` parameter the caller controls. `NewHeader("x","y",math.MaxInt)` is the symmetric concern: `gap = MaxInt - leftW - rightW` (positive, very large) would also OOM. Both edge cases are caller-controlled inputs and not load-bearing for the droplet's claim.

**Verdict:** REFUTED for the documented use case (clamp present, test covers width-smaller-than-content). Pathological caller inputs (`math.MaxInt`/`math.MinInt`) are out of scope per droplet spec.

### Attack 5 — Empty / nil hints divergence

**Hypothesis:** `NewFooter(nil, w).View()` and `NewFooter([]string{}, w).View()` behave differently.

**Evidence:**
- `footer.go:43` — `if len(f.hints) == 0 { return "" }`. `len(nil) == 0` and `len([]string{}) == 0` per Go spec, so the guard catches both identically.
- `TestFooterView_EmptyHints` (`footer_test.go:48-54`) tests `NewFooter(nil, 80).View() == ""`.
- `TestFooterView_EmptySlice` (`footer_test.go:58-64`) tests `NewFooter([]string{}, 80).View() == ""`.
- Both pass.

**Verdict:** REFUTED. nil-slice and empty-slice produce identical `""` output; explicitly covered by separate tests.

### Attack 6 — Width=0 edge case

**Hypothesis:** `NewHeader("x","y",0).View()` or `NewFooter([]string{"a"},0).View()` panics or returns garbage.

**Evidence:**
- `header.go:55-58` — with `h.width=0`, `gap = 0 - leftW - rightW = -(leftW+rightW) < 0`, clamped to 0. Render proceeds: `left + "" + right`. Both strings appear, no padding gap.
- `TestHeaderView_ZeroWidth` (`header_test.go:50-56`) constructs `NewHeader("T","S",0)` and asserts `"T"` appears in output. Passes.
- `footer.go:42-52` — `width` is stored but never read in `View()` (footer doesn't use `width` for layout). Hints render unconditionally. So `width=0` is a no-op for Footer rendering.
- The footer doc comment at `footer.go:24` explicitly states "A zero or negative width is accepted and has no effect on the rendered output." Honest doc.

**Verdict:** REFUTED. Zero width handled gracefully for both types.

### Attack 7 — ANSI escape leak under NO_COLOR

**Hypothesis:** `lipgloss.NewStyle().Bold(true).Render(...)` emits raw ANSI codes even when terminal doesn't support color or `NO_COLOR=1`.

**Evidence:**
- lipgloss v2 honors color profile via `lipgloss.DefaultRenderer()` which auto-detects terminal capability. Under `NO_COLOR`, styles strip ANSI codes — this is lipgloss's documented contract, not a Header/Footer concern.
- The droplet's claim is "uses inline lipgloss styles." Color-stripping is lipgloss's responsibility, not D5's. No D5 code does anything that would bypass the renderer.
- This attack family lands on lipgloss (out of scope), not D5. Closest in-scope concern: would a deterministic test ever break under `TERM=dumb`? The tests use `strings.Contains(out, "MyTitle")` which is ANSI-insensitive — the substring still appears regardless of ANSI wrapping.

**Verdict:** REFUTED (out-of-scope: lipgloss owns color profile; D5 uses the public Render API correctly).

### Attack 8 — lipgloss.Width() vs len()

**Hypothesis:** Builder claims `lipgloss.Width()` but actually uses `len()` for ANSI-unaware byte counting.

**Evidence:**
- `header.go:52` — `leftW := lipgloss.Width(left)`.
- `header.go:53` — `rightW := lipgloss.Width(right)`.
- No `len(` invocations on rendered strings anywhere in `header.go` or `footer.go`. (Only `len(f.hints)` at footer.go:43 — a slice length, not a byte count of a styled string.)

**Verdict:** REFUTED. `lipgloss.Width()` confirmed in source.

### Attack 9 — Smoke test rigor

**Hypothesis:** The 11 smoke tests only exercise trivial getters and skip edge cases.

**Evidence:** Inventory by category:

| Category | Header tests | Footer tests |
|---|---|---|
| Constructor / field storage | TestNewHeader | TestNewFooter |
| Immutability (round-trip) | TestHeaderWithWidth | TestFooterWithWidth |
| Happy-path render | TestHeaderView_ContainsTitleAndSubtitle | TestFooterView_ContainsHints, TestFooterView_SingleHint |
| Edge: width=0 | TestHeaderView_ZeroWidth | (covered structurally — width unused) |
| Edge: content > width | TestHeaderView_WidthSmallerThanContent | (n/a) |
| Edge: nil hints | (n/a) | TestFooterView_EmptyHints |
| Edge: empty slice | (n/a) | TestFooterView_EmptySlice |

11 tests, 5 categories of coverage. Beyond getters: round-trip immutability, content > width clamping, nil-vs-empty-slice divergence. Reasonable rigor for a passive render struct.

**Verdict:** REFUTED. Smoke tests cover constructor, immutability, happy path, AND the four named edge cases (zero width, content > width, nil hints, empty slice).

### Attack 10 — Coverage gate ≥70%

**Hypothesis:** Package-wide coverage drops below 70%, or per-file coverage in `header.go` / `footer.go` is below 70% even if package average passes.

**Evidence:**
- `mage test-pkg ./internal/tui/components` → 57/57 PASS. Project rules forbid raw `go test -coverprofile=...` invocations; coverage is enforced via the canonical `mage ci` gate at drop end, not per-droplet.
- Per droplet spec lines 642-645: "AC8 (`mage ci` green) is a DROP-END gate, not a per-D5 gate. D5 runs only `mage test-pkg ./internal/tui/components`."
- Structural inspection: `header.go` has 3 functions; tests exercise the constructor (1 test), `WithWidth` (1 test), and `View` (3 tests covering happy path, zero width, narrow width). Every public function reached. `footer.go` has 3 functions; tests exercise the constructor (1 test), `WithWidth` (1 test), and `View` (4 tests covering happy path, nil, empty, single hint). Every public function reached.
- With every function exercised, line coverage should be near-100% for both files. Branch coverage on `header.go:56-58` (`gap < 0` clamp) is hit by `TestHeaderView_WidthSmallerThanContent`. Branch on `footer.go:43` (`len(f.hints) == 0` guard) is hit by `TestFooterView_EmptyHints` and `TestFooterView_EmptySlice`.

**Verdict:** REFUTED structurally. Drop-end `mage ci` is the authoritative coverage gate; per-droplet evidence (all public funcs exercised, all branches hit) supports ≥70% with high confidence.

### Attack 11 — YAGNI (anything beyond spec)

**Hypothesis:** Builder added unspecified types, functions, helpers, or fields.

**Evidence:** Exhaustive inventory of `header.go` + `footer.go`:
- `header.go`: 1 type (`Header`), 3 fields (`title`, `subtitle`, `width`), 3 funcs (`NewHeader`, `WithWidth`, `View`). Imports: `strings`, `charm.land/lipgloss/v2`.
- `footer.go`: 1 type (`Footer`), 2 fields (`hints`, `width`), 3 funcs (`NewFooter`, `WithWidth`, `View`). Imports: `strings`, `charm.land/lipgloss/v2`.

Spec (PLAN.md lines 595-623) asks for exactly these. No extras. Doc comments are appropriately verbose but not gold-plating.

**Verdict:** REFUTED. Strict YAGNI compliance.

### Attack 12 — Inline styles vs style-package import

**Hypothesis:** Builder claimed "inline styles" but actually imports `internal/tui/style`.

**Evidence:**
- `header.go:7-11` imports: `"strings"`, `"charm.land/lipgloss/v2"`. No `internal/tui/style` import.
- `footer.go:7-11` imports: `"strings"`, `"charm.land/lipgloss/v2"`. No `internal/tui/style` import.
- Styles constructed inline at `header.go:46-47` and `footer.go:46`. Spec at line 607-610 explicitly says inline is "always safe" — builder picked the simpler, dependency-free option.

**Verdict:** REFUTED. No style-package import; styles are inline.

## Unmitigated Counterexamples

None. All 12 attacks REFUTED.

## NITs

### NIT-1 — `width` field unused in Footer.View()

`footer.go:18` declares `width int` and `WithWidth` mutates it, but `View()` (`footer.go:42-53`) never reads it. The doc comment at line 24 honestly admits "A zero or negative width is accepted and has no effect on the rendered output."

**Severity:** Low. Not a bug — matches spec (footer renders hints horizontally, no width-driven layout). The field is reserved for caller composition (e.g. parent may right-align or center the rendered footer using its own width). Could be addressed by:
- (a) leaving as-is (current state — defensible),
- (b) using width for right-padding or alignment in `View()` (would add behavior beyond spec — YAGNI risk),
- (c) removing `width` from Footer entirely (would change the API contract specified in PLAN.md line 619-623 — out of scope).

Recommendation: leave as-is per spec. Note kept for future-Footer-evolution awareness.

### NIT-2 — Foreground color hardcoded as `#6e7280`

Both `header.go:47` (subtitle) and `footer.go:46` (hints) hardcode the muted color `#6e7280`. Spec allowed `internal/tui/style.style.Body` as an option (line 607-610) and the builder picked inline.

**Severity:** Low. Two locations of the same hex literal — minor duplication. If `internal/tui/style` standardizes a "muted" color in a later droplet, these two call sites become candidates for refactor. Not a defect.

### NIT-3 — Smoke test for Footer.WithWidth doesn't exercise width-zero edge

`TestFooterWithWidth` (`footer_test.go:23-32`) sets `WithWidth(100)` but never `WithWidth(0)`. Since `width` is unused in `View()`, this is structurally inert, but if a future commit adds width-based rendering to Footer, the `WithWidth(0)` round-trip is uncovered.

**Severity:** Very low. Defer.

## Verdict rationale

All 12 attack hypotheses REFUTED with concrete file:line evidence and test output. `mage test-pkg ./internal/tui/components` runs 57/57 green. The droplet implements exactly what PLAN.md D5 specifies: two passive render structs, no Bubble Tea machinery, inline lipgloss styles, ANSI-safe width via `lipgloss.Width()`, and 11 smoke tests covering both happy paths and the named edge cases (zero width, content > width, nil hints, empty slice). Migration markers correct in all 4 D5-authored files. No style-package import. No YAGNI bloat. Three NITs raised are all low-severity quality-of-life items, not defects.

**Overall verdict: PASS**
