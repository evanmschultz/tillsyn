# TA versioning state and reuse paths for Tillsyn

Investigates whether Tillsyn can DRY-share file-copy / move-up-and-down logic with its sister project `ta` (`/Users/evanschultz/Documents/Code/hylla/ta/main/`) — by importing ta packages, vendoring, extracting a shared third repo, or shelling out to a shipped `ta` binary. Anchors to evidence in the ta tree (paths cited inline) so the recommendation does not drift from the source.

This file extends the earlier pass at `workflow/drop_4c_6/RESEARCH/TA_AND_KARPATHY_REVIEW.md` (which recommended sibling tool + selective vendor of `internal/configmerge`) with a deeper view of ta's release readiness, package-export surface, and the cost of each reuse strategy. The dev's leaning is import-as-packages, but only if ta is stable and the relevant code is exposed (not under `internal/`). The investigation below reaches a different conclusion — the right move today is **selective vendor of two surgical files**, with a path forward to import-as-packages once ta tags `v0.1.0` and exports the targeted packages.

---

## A. ta versioning state

### A.1 Tag and release state

`git -C /Users/evanschultz/Documents/Code/hylla/ta/main tag` returns **zero output** — ta has never tagged a release. The module path in `/Users/evanschultz/Documents/Code/hylla/ta/main/go.mod:1` is `github.com/evanmschultz/ta`, Go 1.26.2. No `v0.1.0`, no pseudo-stable tag, no semver baseline.

### A.2 Activity profile

`git log --oneline --since="6 weeks ago"` reports **151 commits** in the last six weeks. Most-recent commit `35f65e6` is dated 2026-05-04 (three days ago) — the f38d huh-removal closeout. Recent log shows large structural moves still landing: huh removal (f38d series, multiple weeks), batch ops (`9da0632 feat(ops): batch get/update/create/delete`), schema consolidation (`7285563 feat(schema): consolidate cascade + claude_agents + agents_md`), agents two-db schema (`0882139`), nested→flat agent install transform (`8c11c5a`), strict-provenance + `--target-system` (`ffaa932`). These are not bug-fix patches — they are catalog-level vocabulary changes. A consumer of ta packages today would catch a wire-shape break at every f-cycle.

Working tree is mildly dirty (`M .ta/schema.toml`, `D .ta/index.toml`, two untracked files), consistent with continued in-flight refactors.

### A.3 Self-declared status

`/Users/evanschultz/Documents/Code/hylla/ta/main/CLAUDE.md:37-43` declares ta is "pre-MVP-feature-completion. The first tagged release will be `v0.1.0` — there's no 'v1' semantics here, just 'every MVP feature works without known issues'." The phasing path is **dogfood → full CLI refinement → full TUI overhaul** before `v0.1.0` ships. Same file lines 41-52 enumerate **open pre-MVP items** that must close before tagging:

- F23 runtime-fill semantics (currently breaks cascade auto-spawn — schema fragment commented-out pending fix)
- Coverage gate (`cmd/ta` at 67.1%, target ≥70%)
- `claude_hooks` / `claude_skills` / `claude_settings_fragments` schemas not yet defined (blocks the dogfood plan to ship hook installs via `ta init`)
- MCP project-arg gate-keeping (security review pre-MVP)
- TUI expansion (`-t` flag, glamour-rendered preview, vim-style multi-select) — locked direction, post-`v0.1.0`
- magefile uses `gofmt` not `gofumpt` — memory-rule contradiction to resolve

The README front matter (lines 9-22) describes ta neutrally as an MCP server — no "alpha" / "beta" / "experimental" banner. No `CHANGELOG.md`, `RELEASES.md`, `STATUS.md`, `ROADMAP.md`, or `MILESTONES.md` at the project root. The plan-of-record is the body of `docs/PLAN.md` (36k of it) plus the open-items list in `CLAUDE.md`.

### A.4 What this implies for Tillsyn

**ta cannot be a stable Go-import dependency today.** Three concrete blockers:

1. **No tags** — Tillsyn's `go.mod` would have to pin a commit hash (`replace` directive or `go mod tidy` writes a pseudo-version like `v0.0.0-20260504135221-35f65e6`). That works mechanically, but every Tillsyn drop that bumps the pin gets a fresh face-full of ta's internal vocabulary changes.
2. **Active churn at the catalog / vocabulary layer** — `feat(schema)`, `feat(ops)`, `feat(agents)` commits have been landing weekly. Tillsyn's `till init` would import ta types (`initapply.Selections`, `initapply.Policy`, `templates.Kind`) whose JSON-wire shape was last changed 2026-04 (f32 strict-provenance). Pinning to a hash freezes the import; unpinning to follow ta forward forces the dev to re-audit the wire shape every bump.
3. **Open MVP items are load-bearing** — F23 runtime-fill semantics blocks ta's own dogfood schemas (`docs/PLAN.md` open items + `CLAUDE.md` line 44). MCP security review is open. These are not gold-plating items; they are pre-`v0.1.0` blockers.

ta is feature-rich, well-tested, and architecturally clean — `go.mod` shows Go 1.26.2 with a sane dep set, the package layout under `internal/` is thoughtfully bounded — but it is not yet `v0.1.0`-stable. The dev's instinct to consider import-as-packages is correct for the post-`v0.1.0` world; today, ta cannot be that dependency without forcing Tillsyn to absorb ta's release pace.

---

## B. Package surface — what's importable today

### B.1 Top-level non-internal packages

Outside `internal/` and `cmd/`, the exportable package surface in ta is **essentially empty** for Tillsyn's needs:

- `/Users/evanschultz/Documents/Code/hylla/ta/main/embed.go` — top-level embed package (1.3kB) — embeds `examples/` for the binary. Not directly useful as an import target unless Tillsyn wants ta's bundled fragment library.
- `cmd/ta/` — `package main`, imports unusable from outside (Go build rules).
- `cmd/ta/internal/tuitest/` — under `cmd/ta/internal/`, restricted to `cmd/ta/`-tree consumers only.

There are **no top-level public packages** Tillsyn could import for file-copy or merge logic. Every candidate sits under `internal/` or `cmd/ta/`.

### B.2 Targeted packages under `internal/` — assessment

Per Go's `internal/` rule, **none** of these are importable from outside `github.com/evanmschultz/ta` today. Each row below assesses whether moving it OUT of `internal/` would be a small refactor for the ta team (Hylla-org-controlled).

| Package                    | Path (in ta tree)                       | Tillsyn-relevant?                                    | Refactor to expose       |
|----------------------------|------------------------------------------|------------------------------------------------------|--------------------------|
| `internal/configmerge`     | `internal/configmerge/configmerge.go`    | **High** — JSON / TOML deep-merge with Conflicts     | Small — already 1 dep on `pelletier/go-toml/v2` (already in Tillsyn `go.mod`) |
| `internal/fsatomic`        | `internal/fsatomic/fsatomic.go`          | **High** — atomic temp+rename writer (52 lines)      | Trivial — zero internal deps |
| `internal/initapply`       | `internal/initapply/initapply.go`        | **Medium** — Selections/Policy/Report contract       | Hard — imports `internal/backend/md`, `internal/configmerge`, `internal/fsatomic`, `internal/templates` (transitive cascade) |
| `internal/templates`       | `internal/templates/templates.go`        | **Medium** — fragment-library API                    | Hard — imports `internal/schema`, `internal/fsatomic`; bound to `~/.ta/` |
| `internal/backend/{md,toml}` | `internal/backend/md/`, `internal/backend/toml/` | **Low** — record-as-file backends specific to ta's schema model | N/A — Tillsyn doesn't need ta's record model |
| `internal/render`          | `internal/render/`                       | **Low** — ta's CLI render (laslig styling)           | N/A — Tillsyn has its own render layer |
| `internal/index`           | `internal/index/index.go` (14k LOC)      | **Low** — `.ta/index.toml` runtime index             | N/A — Tillsyn-specific |
| `internal/search`          | `internal/search/`                       | **Low** — structured + regex search                  | N/A — Tillsyn-specific |
| `internal/db`              | `internal/db/`                           | **Low** — id resolver                                | N/A — Tillsyn-specific |
| `internal/schema`          | `internal/schema/`                       | **Low** — ta's schema vocabulary                     | N/A — Tillsyn doesn't model agents/configs/docs as ta does |
| `internal/ops`             | `internal/ops/` (40k LOC `ops.go`)       | **Low** — ta's get/list/create/update/delete         | N/A — Tillsyn-specific |
| `internal/mcpsrv`          | `internal/mcpsrv/`                       | **Low** — ta's MCP server                            | N/A — Tillsyn has its own |
| `cmd/ta/init_picker.go`    | `cmd/ta/init_picker.go` (18k LOC)        | **Medium** — bubbletea collapsible multi-select picker | Hard — `package main`, must extract to a new package; couples to ta's `pickedItem` / `pickerLeaf` shape |

**Two surgical files stand out for short-term reuse:**

- `internal/configmerge/configmerge.go` (12kB) — file-doc-comment at lines 1-19: "structured-merge primitives for the three config formats `ta init` writes: JSON, TOML, line-oriented text. New keys / lines from `incoming` are added; matching values are no-ops; differing values reported as Conflicts." Dependencies: stdlib + `pelletier/go-toml/v2` only. Has a co-located test file (7.7kB).
- `internal/fsatomic/fsatomic.go` (52 lines) — atomic temp+rename. Zero non-stdlib deps. Has a co-located test file. Tillsyn already has `os.CreateTemp` + `os.Rename` patterns scattered, but `fsatomic.Write` is the canonical wrapper.

**Two file-shape files that depend on ta's internal cascade and are NOT short-term-reusable:**

- `internal/initapply/initapply.go` (36kB + 36kB tests) — imports four other internal packages (line 38-41: `internal/backend/md`, `internal/configmerge`, `internal/fsatomic`, `internal/templates`). The `Selections / Policy / Report` contract is a clean design pattern, but the implementation is bound to ta's home-library + binary-fragment dual-source model with strict-provenance preflight (line 277-305). Tillsyn's `till init` is similar at the contract level but doesn't have a binary-fragment library, doesn't have the same provenance pinning, doesn't share ta's `Kind` enum (`KindSchema | KindAgent | KindConfig | KindDocsTemplate`). Importing this verbatim would force Tillsyn to adopt all four upstream dependencies and the strict-provenance design.
- `cmd/ta/init_picker.go` (18kB + 11kB tests) — bubbletea picker with collapsible groups, multi-select, filter mode, target-system mode. Implementation is in `package main` so no import path exists. Extraction would require ta to move it to a new top-level `picker/` package, decouple `pickedItem` / `pickerLeaf` from cmd/ta-specific routing, and stabilize the option API (`PickerOption`, `WithPickerHeader`, `WithPickerCollapsed`). Doable but non-trivial.

### B.3 Public package status

Nothing currently public matches Tillsyn's reuse targets. Every candidate above is under `internal/` or in `package main`. Without ta-side refactor, **Option 2 (import as packages) is unavailable today** — full stop, by Go language rule.

---

## C. Versioning strategy options

Each option is assessed on (a) ta-side work required, (b) Tillsyn-side API impact, (c) maintenance burden, (d) risk if ta evolves.

### C.1 Option 1 — Vendor copies into Tillsyn

Copy `configmerge.go` + tests + `fsatomic.go` + tests verbatim into `internal/configmerge/` and `internal/fsatomic/` under Tillsyn. No `replace` directive, no module dependency.

- (a) **ta-side work**: zero. ta keeps its package private.
- (b) **Tillsyn-side**: two new internal packages, each ~20 lines of import surface. Wire callers in `till init` (Drop 4c.6 scope) to `configmerge.NewJSONMerger` / `configmerge.NewTOMLMerger` / `fsatomic.Write`.
- (c) **Maintenance**: when ta updates `configmerge`, Tillsyn must manually re-vendor. ta's commit log shows configmerge has been touched in f24-era commits — drift risk is real but bounded (the `Merger` interface at `configmerge.go:53-55` is small and stable).
- (d) **Risk**: if ta's array-dedupe-keys contract changes, Tillsyn's vendored copy diverges silently until somebody notices. Mitigation: tag the vendor commit in a `VENDOR_SOURCE.md` next to the file (which ta commit hash, what differs).

**Cost summary**: ~250 lines vendored + a 5-line provenance note. Maintenance is low because the surface is small and stable. **DRY violated nominally**, but nominally only — the org controls both repos and re-vendor is a 30-second `cp + git diff` audit.

### C.2 Option 2 — Import ta packages directly

Tillsyn's `go.mod` adds `github.com/evanmschultz/ta v0.X.Y` and imports `github.com/evanmschultz/ta/configmerge` (after ta moves it out of `internal/`).

- (a) **ta-side work**: substantial.
  - Move `internal/configmerge/` to `configmerge/` (top-level). Update doc-comments. Re-run `mage check`.
  - Move `internal/fsatomic/` to `fsatomic/` (top-level). Same drill.
  - **Tag a release**. ta's `CLAUDE.md:37-43` says `v0.1.0` is the first tag, gated on closing every pre-MVP item (F23, coverage, hook schemas, MCP gate, magefile gofumpt). Realistic ETA: weeks to months (151 commits in 6 weeks suggests active churn, not stabilization yet).
  - Decide and document the public-API contract for `configmerge` + `fsatomic`. Today it's an internal-package "stable enough for ta" contract; making it public means semver guarantees on `Conflict` shape, `Merger` interface, `NewJSONMerger` signature.
- (b) **Tillsyn-side**: clean import. `go.mod` gains one dep. Caller code looks like `import "github.com/evanmschultz/ta/configmerge"` — same as Option 1's caller code, modulo the `internal/` vs top-level path.
- (c) **Maintenance**: Tillsyn pins `v0.1.0`, bumps when needed. Standard Go module hygiene.
- (d) **Risk**: low post-tagging — ta's semver policy gates breaking changes. Pre-tagging (today): high — every commit can change the wire shape.

**Cost summary**: Tillsyn-side near-zero, but the gating ta-side work is `v0.1.0` itself, which ta-CLAUDE.md treats as a real milestone with concrete blockers. **Not viable for Drop 4c.6's `till init` timeline.**

### C.3 Option 3 — Extract shared packages into a third repo

Create `github.com/hylla-org/till-fs` (or similar) carrying `configmerge` + `fsatomic`. Both ta and Tillsyn import.

- (a) **ta-side work**: substantial — same as Option 2 (extract from `internal/`, tag, document semver), plus the additional cost of creating and maintaining a third repo. ta-`CLAUDE.md` line 14 lists templates package as "stdlib + internal/schema + internal/fsatomic only" — a strict firewall — so extracting `fsatomic` out of ta entirely (not just out of `internal/`) means ta would need to depend back on `till-fs`. That's a circular-feeling design that isn't actually circular but adds friction.
- (b) **Tillsyn-side**: clean — `import "github.com/hylla-org/till-fs/configmerge"`.
- (c) **Maintenance**: highest of all options. Three repos to track instead of two. Issues / PRs / release cadence for `till-fs` are now their own concern.
- (d) **Risk**: low long-term, high short-term — the org has two devs, neither of whom needs a third repo right now. Extraction makes sense when ≥3 consumers share the code; today there are two (ta, Tillsyn).

**Cost summary**: high upfront, low long-term. **Not justified at the current consumer count.** Park as a "if a third consumer appears" option.

### C.4 Option 4 — Ship ta as a companion binary

Tillsyn install installs both `till` and `ta` binaries. Tillsyn shells out to `ta` for project-init copy and ~/.tillsyn/↔.tillsyn/ moves.

- (a) **ta-side work**: ship a `go install`-able binary at a known import path (already true: `github.com/evanmschultz/ta/cmd/ta`). Stabilize the CLI surface Tillsyn calls (`ta init`, `ta template *`, exit codes, `--json` output shape). The CLI is mostly stable per `docs/PLAN.md` §A but pre-MVP items still touch it (F32 strict-provenance changed wire shape weeks ago).
- (b) **Tillsyn-side**: shell-out path. Tillsyn's `till init` becomes "build the selections payload, write to a temp file, exec `ta init --target <project> --selections-file <tmp> --json`, parse the JSON report". This is a real architectural commitment — the user's machine now has two binaries, two `~/.X/` directories (`~/.tillsyn/` for tillsyn-state, `~/.ta/` for ta's fragment library), and two install paths. UX cost: real but bounded if README directs `till install` → installs both binaries.
- (c) **Maintenance**: medium. ta's CLI flags can drift; Tillsyn's shell-out caller must track them. JSON `--json` output is the contract — if ta keeps its `init_cmd.go`'s `initReport` (line 77-90 in `init_cmd.go`) wire-stable, Tillsyn is safe.
- (d) **Risk**: medium. CLI evolution is gentler than package-API evolution because exit codes + `--json` shape are designed for stability. ta-`docs/PLAN.md` §A.5 already treats the `--json` shape as a wire contract.

**Cost summary**: highest UX cost (two binaries, two state dirs), lowest coupling. Suits a world where ta and Tillsyn are companion tools the user runs explicitly. **The dev's "ship the ta tool when tillsyn install" framing fits here** — but the same outcome can be achieved by README pointer rather than `till install` packaging both binaries. The packaging-as-one-install adds install-script complexity for marginal user benefit.

---

## D. Specific reuse targets — Drop 4c.6 minimum viable scope

Drop 4c.6's `till init` command needs (per the dev's framing):

1. **File copy with conflict-resolution policy** — overwrite / skip-existing / merge.
2. **Atomic write** — temp + rename so partial writes can't corrupt.
3. **Bubbletea picker** — collapsible groups, multi-select, provenance-tagged leaves.
4. **Field-level deep-merge** — `agents.local.toml` over `agents.toml`.

Per-target ta package:

| Target                                | ta package                                  | Importable today? | Refactor cost to expose |
|---------------------------------------|---------------------------------------------|-------------------|--------------------------|
| File copy + conflict policy           | `internal/initapply` (Apply + Policy)       | No                | High — 4 transitive internal deps |
| Atomic write                          | `internal/fsatomic`                         | No                | Trivial — zero internal deps |
| Bubbletea picker                      | `cmd/ta/init_picker.go`                     | No                | High — `package main`, extract to new top-level package |
| Field-level deep-merge                | `internal/configmerge`                      | No                | Small — 1 external dep already in Tillsyn |

**Two short-term-viable targets (vendor copy)**: `fsatomic` (52 lines) and `configmerge` (12kB + tests). Both are surgical, both have small / zero internal-dep footprints, both are stable enough that re-vendor on a ta change is a quick audit.

**Two long-term targets (wait for ta `v0.1.0` + ta-side extraction)**: `initapply` and `init_picker`. Both are richer architectural patterns that Tillsyn could benefit from — but importing them today means dragging in ta's home-library + binary-fragment cascade, or extracting an 18kB `package main` file into a new public package. Neither is justified at Drop 4c.6's scope.

**Bubbletea picker — alternative path**: Tillsyn already uses `charm.land/bubbles/v2` (`go.mod` in tillsyn — verified via prior research). Building a Tillsyn-local picker on top of the bubbles `list` primitive is comparable effort to extracting ta's, with the upside of zero coupling to ta's `pickerLeaf` / `pickedItem` shape. The dev's prior research (`TA_AND_KARPATHY_REVIEW.md`) noted ta's picker has unique features (collapsible groups, target-system mode); whether Tillsyn needs those is a Drop 4c.6 design question, not an evidence question.

---

## E. Recommendation

**Vendor `fsatomic` and `configmerge` from ta into Tillsyn for Drop 4c.6.** Add a `VENDOR_SOURCE.md` next to each vendored file recording the source commit hash + diffs-from-source. Plan to migrate to import-as-packages once ta tags `v0.1.0` and exposes both packages publicly.

### E.1 Why not Option 2 today

Option 2 (import as packages) is the dev's stated preference and is the right long-term answer — but it requires ta-side prerequisites that aren't met today:

- **No tags** — Tillsyn would pin a pseudo-version, which works mechanically but offers no semver protection.
- **Active vocabulary churn** — 151 commits / 6 weeks at the catalog layer. A Tillsyn drop that bumps the pin gets a fresh wire-shape audit each time.
- **`v0.1.0` blockers are real** — F23 runtime-fill, coverage gate, hook schemas, MCP security review, magefile gofumpt. ta's own CLAUDE.md treats these as load-bearing. ETA is weeks to months, not days.
- **Targeted packages are under `internal/`** — Go's `internal/` rule blocks the import outright until ta moves the packages.

### E.2 Why not Option 4 (companion binary)

Option 4 (ship ta as companion) has lowest coupling but highest UX cost — two binaries, two `~/.X/` directories, two install paths. The dev's "ship the ta tool when tillsyn install" framing is appealing but the same outcome works via README pointer ("Tillsyn pairs well with `ta`, install via `go install github.com/evanmschultz/ta/cmd/ta@latest`") at zero install-script complexity. Worth keeping as a "later, when ta hits MVP" possibility, not a Drop 4c.6 strategy.

### E.3 Why vendor specifically these two files

- `fsatomic.go` — 52 lines, zero deps. Trivial to vendor; trivial to keep current. Saves Tillsyn from open-coding `os.CreateTemp` + `os.Rename` + cleanup-on-error in three different call sites.
- `configmerge.go` — 12kB, one external dep already in Tillsyn (`pelletier/go-toml/v2`). The Conflicts surface (file `:35-55`) is exactly what `till init`'s `agents.local.toml` over `agents.toml` merge needs. Co-located 7.7kB test file vendors cleanly too — Tillsyn gets the test coverage as part of the copy.

### E.4 What needs to happen for the long-term path

If the dev wants to re-evaluate Option 2 in a future Tillsyn drop:

1. **Wait for ta `v0.1.0`** — track ta-`CLAUDE.md` open-items list; the gate is "every item closed".
2. **Ask the ta side** to move `configmerge/` and `fsatomic/` out of `internal/` to top-level packages. Per B.2, this is small refactor for both: zero or one external-dep change, no internal-package consumer churn (initapply imports fsatomic but the import path changes are mechanical).
3. **Optional**: revisit `initapply` extraction once ta's home-library / binary-fragment design has stabilized post-MVP. If Tillsyn's project-init grows enough to need Selections/Policy/Report pattern, the right move then is import-not-vendor — but that's post-`v0.1.0` work and post-Drop-4c.6.
4. **Optional**: revisit `init_picker` extraction if Tillsyn's TUI needs grow into collapsible-group + multi-select. Today, building a Tillsyn-local picker on `charm.land/bubbles/v2`'s `list` is comparable effort to extracting ta's — the deciding factor is which features Tillsyn's TUI needs (target-system mode, filter mode, collapsible groups), and that's a Drop-internal design question.

### E.5 Companion-tool framing

The dev's instinct that "ta could be a great companion tool, that the README at least suggests using" is correct and deserves explicit support. Recommend the Tillsyn README gain a "Companion tooling" section pointing at ta as the structured-data-on-MD-and-TOML companion. Zero install-script changes required — just a README pointer. This is independent of the vendor-vs-import decision above.

---

## F. Cost summary table

| Strategy             | ta-side cost          | Tillsyn-side cost     | Maintenance      | Risk if ta evolves     | Available today? |
|----------------------|-----------------------|-----------------------|------------------|------------------------|------------------|
| **Vendor (rec'd)**   | Zero                  | ~250 LOC + 1 note     | Low (re-vendor)  | Low (small surface)    | Yes              |
| Import packages      | High (extract + tag)  | 1 go.mod dep          | Lowest           | Lowest post-tag        | No               |
| Third-repo extract   | Highest               | 1 go.mod dep          | Highest          | Lowest long-term       | No               |
| Companion binary     | Medium (CLI semver)   | Shell-out path        | Medium           | Medium (CLI drift)     | Partially        |

**Recommendation**: vendor for Drop 4c.6. Re-evaluate import-as-packages at ta `v0.1.0`. Add README companion-tool pointer independently.

---

## G. Sources

- ta module declaration: `/Users/evanschultz/Documents/Code/hylla/ta/main/go.mod:1`
- ta tag state: `git -C /Users/evanschultz/Documents/Code/hylla/ta/main tag` (zero output)
- ta commit cadence: `git log --since="6 weeks ago"` returns 151 commits; latest `35f65e6` 2026-05-04
- ta self-declared status: `/Users/evanschultz/Documents/Code/hylla/ta/main/CLAUDE.md:37-52`
- ta open MVP items: same file lines 41-52
- ta package layout: `/Users/evanschultz/Documents/Code/hylla/ta/main/internal/{configmerge,fsatomic,initapply,templates,backend,render,index,db,ops,mcpsrv,schema,search,record,config}/`
- configmerge Merger contract: `/Users/evanschultz/Documents/Code/hylla/ta/main/internal/configmerge/configmerge.go:1-80`
- fsatomic full source: `/Users/evanschultz/Documents/Code/hylla/ta/main/internal/fsatomic/fsatomic.go:1-52`
- initapply contract + transitive deps: `/Users/evanschultz/Documents/Code/hylla/ta/main/internal/initapply/initapply.go:1-49, 38-41, 277-305`
- templates package firewall: `/Users/evanschultz/Documents/Code/hylla/ta/main/internal/templates/templates.go:1-44, 14`
- ta `init_cmd.go` initReport wire shape: `/Users/evanschultz/Documents/Code/hylla/ta/main/cmd/ta/init_cmd.go:77-90`
- Prior research recommendation context: `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/workflow/drop_4c_6/RESEARCH/TA_AND_KARPATHY_REVIEW.md`
