# `ta` And Karpathy-Skills Review For Drop 4c.6 Planning

**Status**: research deliverable. Read-only investigation. No code changes recommended in this MD — output is informational input for the dev's planning decisions.

**Scope**: two external references named in the Drop 4c.6 sketch session — (1) the sister project at `/Users/evanschultz/Documents/Code/hylla/ta/main/`, (2) the GitHub repo `https://github.com/forrestchang/andrej-karpathy-skills`. The dev wants to know whether to import packages from `ta`, whether `ta` should stay a sibling tool that Tillsyn's README points at, and how the karpathy-skills concept might map onto Tillsyn's per-kind agent dispatch.

**Evidence sources**: direct file reads of `/Users/evanschultz/Documents/Code/hylla/ta/main/` (README, CLAUDE.md, go.mod, embed.go, internal package doc-comments, MCP tool surface, schema fragments, examples README); WebSearch for the karpathy-skills repo and Anthropic's official Claude Code skills format. WebFetch was permission-denied so all external evidence is via WebSearch result excerpts.

---

## Part 1 — `ta` Project Review

### 1.1 What Does `ta` Do?

`ta` is a tiny, single-binary Go MCP server (`module github.com/evanmschultz/ta`, `go 1.26.2`) that exposes per-project TOML and Markdown files **as if they were a structured database**. Each project carries one schema at `<project>/.ta/schema.toml` declaring **dbs** (filesystem-rooted record collections) and **types** (typed records inside dbs). Agents read and write records through a uniform MCP tool surface — `get`, `list_sections`, `create`, `update`, `delete`, `move`, `search`, `schema`, `init` — and `ta` validates every mutation against the schema (required fields, type checks, enum closure, regex patterns) before atomic write. The same surface is mirrored as Cobra/Fang CLI subcommands so humans get a styled terminal UI (`laslig` + bubbletea) and agents get JSON via `--json` (project README §"MCP client config", `cmd/ta/main.go` `longDescription`). The on-disk source of truth is human-editable TOML/MD; `ta`'s job is to keep it schema-valid and append the structured-DB ergonomics agents need.

The project bills itself as "a tiny MCP server that lets LLM coding agents read and write TOML and Markdown files as if they were a structured database — with schemas to keep agents honest" (`README.md` line 3).

### 1.2 Architecture Overview

Top-level package layout (`cmd/ta/`, `internal/*`, `embed.go` at root, `examples/` tree shipped as `embed.FS`):

| Package                   | Responsibility                                                                                                                                |
| ------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------- |
| `cmd/ta/`                 | Cobra/Fang CLI entrypoint; bubbletea pickers (`init_picker.go`, `picker.go`), forms (`form.go`), confirms (`confirm.go`), root menu (`menu.go`). 11 subcommands wired in `main.go`. |
| `embed.go` (package `ta`) | Single `//go:embed all:examples` directive. Exposes `EmbeddedExamples() fs.FS` to consumers via DI. The only non-mage Go file at repo root.   |
| `internal/config`         | Schema resolver — reads exactly `<projectPath>/.ta/schema.toml`. No ancestor walk, no home-layer fallback (V2-PLAN §12.11 / §14.2).            |
| `internal/schema`         | Schema model + meta-schema validator. `meta_schema.toml` ships with the package. Enforces field types, db/type/path uniqueness, format vs. extension match, MD heading uniqueness. Includes the F23 `auto_spawn` mechanism (templates spawn child records on parent create with `{parent_id}` / `{index}` interpolation tokens). |
| `internal/backend/{md,toml}` | File-format adapters. `md/` handles YAML-frontmatter-plus-body (`record_per = "file"` + `body_field = "<name>"`) and section-mode (one MD file = many records keyed by heading). `toml/` does the equivalent for TOML files. Both atomic-write via `internal/fsatomic`. |
| `internal/ops`            | Plain-Go endpoints — `Get`, `Create`, `Update`, `Delete`, `Move`, `Search`, schema mutations. The MCP and CLI surfaces both call into `ops`. Cache layer (`cache.go`) memoizes resolved schemas per project. |
| `internal/mcpsrv`         | MCP-protocol glue. Single `Config{Name, Version, ProjectPath}` constructor; tools registered in `tools.go`; handlers thin-wrap `ops`. Uses `github.com/mark3labs/mcp-go v0.48.0`. |
| `internal/templates`      | The `~/.ta/` home library + binary-embedded `examples/` tree. Schema fragments, agents, configs, docs templates. Only `ta init` and `ta template *` touch it; the runtime MCP server doesn't import it. |
| `internal/initapply`      | Multi-category init flow — `Selections{Schemas, Agents, Configs, DocsTemplates}` → conflict policy (`error` / `skip` / `overwrite` / `force`) → atomic apply. Decouples picker/MCP shapes from apply logic. |
| `internal/configmerge`    | Structured merge for JSON/TOML configs (used by initapply for `.claude/settings.json`, `.codex/config.toml`, `.mcp.json`, `.gitignore`).      |
| `internal/index`          | `.ta/index.toml` runtime record-type index (one record id → which type). Rebuildable via `ta index rebuild`.                                  |
| `internal/render`         | `laslig`-styled output formatting. Schema-flow renderer for `ta schema --json=false`. Golden-tested.                                          |
| `internal/db`             | Internal db-resolution helpers (id parsing, db lookup).                                                                                       |
| `internal/fsatomic`       | Atomic file-write primitive — write-temp-then-rename.                                                                                         |
| `internal/search`         | Search engine — exact-match field filters AND-combined plus a Go RE2 regex over string fields, scope-narrowed by id prefix.                   |
| `internal/record`         | Record-id parsing and validation.                                                                                                             |

The dependency graph is strictly hexagonal. `internal/templates` is firewalled from `internal/config/Resolve`, `internal/ops`, and `internal/mcpsrv` per the V2-PLAN §14.2 invariant — runtime consumers never import the template library; `ta init` and `ta template *` are the only callers.

### 1.3 Storage Model — Nested Directories For Agents And Configs

`ta`'s storage model has three distinct layers (`examples/README.md` lines 16-39):

1. **Per-project schema** — `<project>/.ta/schema.toml`. One file. Declares the project's dbs.
2. **Home library** — `~/.ta/` parallel structure: `~/.ta/schema.toml` (aggregated dbs the user has saved across projects), `~/.ta/agents/<group>/<name>.md` (subagent definitions, kind-nested), `~/.ta/configs/<canonical>` (canonical-named config files), `~/.ta/docs-templates/<canonical>` (project-root MD templates).
3. **Binary-embedded defaults** — `examples/` shipped via `embed.FS`: `examples/schemas/cascade.toml`, `examples/schemas/claude_agents.toml`, `examples/agents/<group>/*.md` (today empty, `.keep` sentinel), `examples/configs/*` (empty), `examples/docs-templates/*` (empty).

**Hierarchical groupings**: agents nest one level by user-chosen group. The README is explicit: "`<group>` is whatever the user chose at save time — ta does not infer language; the picker enumerates whatever subdirs exist" (`examples/README.md` lines 23-27). So you'd see `agents/go/builder.md`, `agents/go/qa-proof.md`, `agents/python/researcher.md`, etc., and the picker enumerates `<group>/` subdirs as collapsible MultiSelect groups.

**On-disk record shape for agents** — `examples/schemas/claude_agents.toml` declares the contract:

- Single `claude_agents` db with **two mounts**: `agents/*/*.md` (home library, kind-nested) AND `.claude/agents/*.md` (project install, flat).
- `record_per = "file"` + `body_field = "prompt"`. Each markdown file IS one whole record. YAML frontmatter carries typed fields (`name`, `description`, `tools`, `model`, `color`); the markdown body becomes the `prompt` field.
- The F33 install transform flattens `<group>/<name>.md` to `<group>-<name>.md` when copying home → project (`.claude/agents/go-builder.md` from `agents/go/builder.md`). Required because Claude Code's project-scoped agent loader expects a flat directory.
- The F35 resolver guarantees a 2-segment home id (`<kind>.<name>`) never silently matches the project mount and a 1-segment project id (`<flat-name>`) never silently matches the home mount.

**Cascade schema** — `examples/schemas/cascade.toml` is much heavier: it declares NodeBase + ActionItem base types with `extends`, `Comment` / `ChecklistItem` / `ContextBlock` / `ResourceRef` aliases, the four cascade dbs (`cascade`, `plans`, `discussions`, `project`), and standard repo-file dbs (`readme`, `contributing`, `security`, `agents_md` multi-mount tracking both `AGENTS.md` and `CLAUDE.md`). The role enum lives at `[project.bases.ActionItem.fields.role]` with the same closed set Tillsyn uses (`builder`, `qa-proof`, `qa-falsification`, `qa-a11y`, `qa-visual`, `design`, `commit`, `planner`, `research`). The structural-type enum is also there (`drop`, `segment`, `confluence`, `droplet`). This is `ta`'s declarative way of saying "if you want a Tillsyn-shaped cascade in your project, here's a schema you can install."

The schema declares but does not yet validate `auto_spawn` runtime-fill semantics — the `cascade.droplet` auto-spawn block is commented out pending F23 token expansion (`{now}`, `{state.initial}`, `{parent.<field>}`) per `CLAUDE.md` line 42-44.

### 1.4 Picker UX — Bubbletea Multi-Category Picker

The picker (`cmd/ta/init_picker.go` + `picker.go`) is a collapsible-group multi-select bubbletea TUI. From `picker.go` doc-comment lines 14-78:

- **`pickerGroup`** — one collapsible bucket of leaves (e.g. "Schemas", "Agents", "Configs", "Docs Templates").
- **`pickerLeaf`** — one selectable row with `Display` (text), `Value` (opaque payload), `Selected` (pre-mark hint).
- **`pickedItem`** — `(group_header, value)` pair returned post-submit; callers route by header.
- Keymap: `j/k` navigate, `h/l` collapse/expand a group, `enter` toggle a leaf or expand a header, `space` toggle all visible leaves in a group, `/` filter mode (search-as-you-type), `tab` defaults all groups to collapsed.
- Provenance tagging: each leaf carries `[ta]` (binary) or `[home]` (user library). Picker shows both sources side-by-side; the user mixes-and-matches across categories.

Used by:

- `ta init` — multi-category bootstrap. Picks N schema fragments + M agents + K configs + L docs templates in one pass. Single source of truth for the F38d-2 VHS demos (`cmd/ta/testdata/vhs/picker_*.gif`).
- `ta init --target-system` — bootstrap-the-system flow that also handles `~/.ta/` (home target) vs `<project>/` (project target).
- The single-group reuse pattern (`WithPickerHeader`, `WithPickerCollapsed`, etc.) feeds the same model from db-toggle and MCP-toggle pickers. Reusable component, not a one-off.

The picker's API is bubbletea-direct — `huh` was deliberately removed in F38d (commit `35f65e6 docs: close f38d huh-removal tracker`). The driver is `charm.land/bubbletea/v2` + `charm.land/bubbles/v2`, with `charm.land/lipgloss/v2` for styling. Verification is teatest goldens (`cmd/ta/internal/tuitest`, sha256-pinned) plus VHS recordings (`mage Vhs`).

### 1.5 MCP Surface — Nine Tools

`internal/mcpsrv/server.go` `registerTools()` registers exactly nine tools (`server.go:74-84`):

| Tool            | Purpose                                                                                                                                        |
| --------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| `get`           | Read records by id. Universal `items[]` shape — length 1 = single read, length >1 = batch. Each item: `{id, fields?}`. Misses → `found: false` per item, not a tool-level error. |
| `list_sections` | Enumerate record ids under a scope (id prefix). Defaults to 10; `all=true` widens.                                                              |
| `create`        | Create records. Per-item `{id, type (db-qualified), data, no_spawn?}`. Auto-spawn fires unless suppressed. Duplicate ids reject.                |
| `update`        | PATCH-style overlay. Per-item `{id, data, type?}`. Empty `data` = no-op success. Atomic re-validation post-merge.                               |
| `delete`        | Remove a record by id, or a whole file by file-relpath + `force=true`. Per-item; siblings continue on per-item failure.                         |
| `move`          | Relocate a record. `{src_id, dst_id, copy?, type?, force?}`. Move = src spliced out post-dst-land; copy preserves src.                          |
| `search`        | Structured + regex search. Scope (id prefix) + `match` (field exact-equality AND-combined) + `query` (Go RE2 regex over string fields) + `field` (restrict regex). |
| `schema`        | Inspect or mutate the resolved schema. `action ∈ {get, create, update, delete}`, `kind ∈ {db, type, field, base}`, `name` (dotted address), `data` (kind-specific meta-schema payload). Sugar: `paths_append` / `paths_remove` for incremental db-paths edits. |
| `init`          | The F24 multi-category init flow's MCP wire shape — accepts a `Selections` payload + conflict policy and produces a structured `Report`.        |

Every tool takes `path` (absolute project directory). The MCP server is **one project per process** — the schema resolves from `cfg.ProjectPath` at boot (pre-warmed in `New`); the tool handlers reuse that resolved schema.

### 1.6 Production Readiness — Pre-MVP

`ta` is **explicitly pre-MVP-feature-completion**. Evidence:

- No git tags. `git tag -l` returns empty.
- `CLAUDE.md` line 38-39: "ta is pre-MVP-feature-completion. The first tagged release will be `v0.1.0` — there's no 'v1' semantics here, just 'every MVP feature works without known issues'. Phasing: **dogfood** (minor issues OK if MCP + basic CLI work) → **full CLI refinement** → **full TUI overhaul**."
- Open pre-MVP items at `CLAUDE.md` lines 41-52 include: F23 runtime-fill semantics still gating cascade auto-spawn (commented out), `cmd/ta` coverage at 67.1% (target ≥70%), MCP project-arg gate-keeping pending security review, TUI expansion deferred post-dogfood.
- Recent commit log (last 25): the `f38*` series is huh-removal cleanup (closed), `f37` was batch-shaped MCP tools, `f36` was `ta move`, `f35` was schema consolidation. Velocity is high; landed-feature density is solid; the gap to v0.1.0 is mostly polish + the F23 cascade-completion item.

**Verdict**: `ta` works today for the MCP-CRUD-on-TOML/MD use case. The cascade-orchestration use case (auto-spawn QA twins, drop-tree integrity) is staged in schema but **not runtime-active** until F23 lands. For Tillsyn's purposes, the `internal/templates` + `internal/initapply` + `cmd/ta` picker stack are the most-useful surfaces today; the `cascade` schema is a working reference but not a runnable engine.

### 1.7 Reusability Assessment — Per Package

For each `ta` package, the question is: cleanly importable from Tillsyn, too coupled to `ta`, or duplicative of something Tillsyn already has.

| Package                 | Cleanly importable?       | Duplicative? | Notes                                                                                                                                                 |
| ----------------------- | ------------------------- | ------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| `internal/config`       | **Yes** (small, focused)  | Partial      | Resolves `<proj>/.ta/schema.toml`. Tillsyn's equivalent is `internal/config` resolving `<proj>/tillsyn.toml`. **Different files, same shape**. Reuse via copy-of-pattern, not import. |
| `internal/schema`       | **Possible**              | Partial      | Generic schema validator with meta-schema. Tillsyn's `internal/templates` does template-shape validation but on different schema vocabulary. Could be the basis of `agents.toml`'s validator if Tillsyn adopts `ta`'s dbs/types/fields model — but that's a bigger commitment than Drop 4c.6 is scoped for. |
| `internal/backend/{md,toml}` | **Yes**                   | No           | TOML and MD-with-YAML-frontmatter file backends with atomic write + section-mode + file-record-mode. Tillsyn has its own TOML reader for templates; `ta`'s backends are richer (frontmatter, multi-mount). Reuse if Tillsyn moves to file-as-record for agents. |
| `internal/ops`          | **No**                    | Yes          | Ops layer is tightly bound to `ta`'s nine MCP tools and the `Resolve(projectPath) → resolvedSchema` cache. Not Tillsyn's shape — Tillsyn has its own `cmd/till dispatcher` ops layer.                                          |
| `internal/mcpsrv`       | **No**                    | Yes          | `ta`'s MCP server. Tillsyn has its own `internal/adapters/server/mcpapi` exposing `till_*` tools. No reason to layer one on top of the other.        |
| `internal/templates`    | **Yes** (high value)      | No           | The `~/.ta/` home library + binary-embedded defaults logic with provenance tagging. Tillsyn's drop 4c.5 templates package lives under `internal/templates` and overlaps here, but `ta`'s home-library + binary-defaults pattern is exactly what Drop 4c.6's `agents.toml` (project-tracked) + `agents.local.toml` (gitignored override) wants — though at file-pair granularity, not directory granularity. |
| `internal/initapply`    | **Yes** (medium value)    | Partial      | The Selections-→-Apply contract with conflict policy (error/skip/overwrite/force). Tillsyn currently does `ta init` style by hand in its dispatcher Wave-1/Wave-2 wiring; this could be lifted directly. |
| `internal/configmerge`  | **Yes** (high value)      | No           | Structured merge for JSON+TOML with append-with-dedupe semantics for arrays. Drop 4c.6 §4 "Override semantics" wants exactly this for `agents.local.toml` field-level deep-merge over `agents.toml`. **Strongest reuse candidate.** |
| `internal/fsatomic`     | **Yes** (trivially)       | Partial      | Write-temp-then-rename. Tillsyn likely has its own equivalent; a 30-line copy is fine, no need to import a third-party package boundary. |
| `internal/render`       | **No**                    | Yes          | `laslig` glamour-rendered MD. Tillsyn doesn't render MD to ANSI in MCP responses (it returns markdown). Different concern.                            |
| `internal/index`        | **No**                    | Partial      | The `.ta/index.toml` runtime index. Tillsyn's SQLite store handles this differently.                                                                  |
| `cmd/ta` picker         | **Maybe** (refactor cost) | No           | The bubbletea picker is bound to `ta`'s MCP-toggle/schema-toggle/init-flow callers. Pulling it into Tillsyn would require refactoring it into a consumable package (`internal/tui/picker`) inside `ta` first. Not currently shaped for cross-project import. |
| `embed.go` (root)       | **No** (pattern only)     | No           | The `//go:embed all:examples` pattern is good prior art for shipping `agents.example.toml` in the Tillsyn binary. Copy the pattern, not the file.   |

**Net**: **`internal/configmerge` and `internal/initapply` are the strongest reuse candidates** for Drop 4c.6. `internal/templates` is interesting if Tillsyn wants the home-library + provenance pattern for shipping default `agents.toml` snippets. The MCP/ops/TUI surfaces are too project-shaped to share.

### 1.8 Recommendation — Sibling Tool, README Pointer, Selective Module Imports

**Headline**: `ta` should stay a separate tool. Tillsyn's README should point at `ta` as the recommended way to manage `~/.tillsyn/` user-defined content. **But** Tillsyn should selectively import the two-to-three packages where `ta` has done load-bearing work that Drop 4c.6 needs.

**Why sibling, not absorbed**:

- `ta`'s value proposition is **generic agent-friendly TOML/MD store with schemas**. That's broader than Tillsyn's cascade orchestration. Forcing `ta` into Tillsyn would constrain `ta`'s design.
- `ta` is pre-MVP. Importing pre-MVP code as a hard dependency couples Tillsyn's release cadence to `ta`'s. Bad for both projects.
- Tillsyn already has its own MCP server, its own SQLite store, its own template resolver. Doubling up the MCP/storage layer is expensive churn.
- Users who want personal cross-project agent libraries (the `~/.ta/agents/<group>/<name>.md` model) are arguably best served by `ta` itself. Tillsyn doesn't need to reinvent that surface — `ta init` will (per the F23/F24 plan) install Tillsyn-shaped schemas + cascade-shaped agents directly into a project, doing exactly what the dev wants.

**Why selective import**:

- **`internal/configmerge`** — Drop 4c.6 §4 specifies field-level deep-merge for `agents.local.toml` over `agents.toml`. `ta/internal/configmerge` does exactly this for JSON+TOML with append-with-dedupe array semantics. Importing as `github.com/evanmschultz/ta/internal/configmerge` is impossible (Go disallows internal-package import across modules), so the practical move is **vendor or mirror the package** under Tillsyn's `internal/configmerge/` with attribution. ~ a few hundred lines of code; small, well-tested.
- **`internal/initapply` (pattern-only)** — the Selections + Policy + Report contract is a clean separation of "what does the user want" vs "how do we apply it with conflict resolution". Drop 4c.6 needs a much smaller version (one file pair, not four categories), so don't vendor — just mirror the shape.
- **`embed.FS` pattern** — Tillsyn's binary should ship `agents.example.toml` (per Drop 4c.6 sketch §8 D3) via `//go:embed`. Copy `ta`'s `embed.go` pattern verbatim.

**README pointer language** (proposal, not prescription):

> "Tillsyn's per-cascade-agent runtime config lives at `<project>/agents.toml` (project default, git-tracked) and `<project>/agents.local.toml` (user override, gitignored). For broader management of personal `~/.tillsyn/agents/` libraries — cross-project agent definitions, project bootstrapping with multi-category pickers, and schema-validated TOML/MD as a structured store — see [`ta`](https://github.com/evanmschultz/ta). `ta` is a sibling tool by the same author; install it separately if you want that workflow."

**Trade-offs**:

| Path                                  | Pros                                                                                                                | Cons                                                                                                                  |
| ------------------------------------- | ------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------- |
| Sibling + selective import (recommended) | Tillsyn ships fast, `ta` evolves freely, users can use either or both, code reuse where it actually pays off.       | Two binaries to install. Some duplication if user wants both.                                                         |
| Absorb `ta` into Tillsyn              | One binary. Unified UX. Tillsyn owns the whole `~/.tillsyn/` story.                                                 | Massive scope creep for Drop 4c.6 (was: 1 file pair; becomes: home-library + binary-defaults + 9 MCP tools). Couples release cadence. Distracts from cascade dispatcher work. |
| Hard dependency on `ta`               | Less code in Tillsyn.                                                                                               | Tillsyn pinned to pre-MVP `ta` version. Breaking changes in `ta` break Tillsyn. No internal-package access without forking. |

**Conclusion**: sibling tool + selective vendor of `configmerge` + README pointer is the lowest-friction path. Drop 4c.6 lands fast; `ta` keeps its independence; users who want the broader agent-library workflow get pointed at `ta`.

---

## Part 2 — Karpathy-Skills Review

### 2.1 What Is A Karpathy-Skill?

**Important framing correction**: the repository name is misleading. `forrestchang/andrej-karpathy-skills` is **not** an Anthropic-format Claude Code skill. It is **a single CLAUDE.md file** of behavioral guidelines. Andrej Karpathy did not write it — Forrest Chang (Jiayuan Zhang) authored it on January 27, 2026, the day after Karpathy posted on X identifying common LLM-coding failure modes. Chang encoded Karpathy's observations into a CLAUDE.md.

The repository is a 65-line CLAUDE.md (per [Miraflow's analysis](https://miraflow.ai/blog/karpathy-claude-md-100k-github-stars-ai-coding-2026)) shipped under MIT license with installation as `curl https://raw.githubusercontent.com/forrestchang/andrej-karpathy-skills/main/CLAUDE.md >> CLAUDE.md`. Repo has 105.4k stars, ranking it among the most-starred Claude Code resources in 2026 (per [star-history.com](https://www.star-history.com/forrestchang/andrej-karpathy-skills/)).

**The four principles** ([DeepWiki summary](https://deepwiki.com/forrestchang/andrej-karpathy-skills/3-the-four-principles)):

1. **Think Before Coding** — "Don't assume. Don't hide confusion. Surface tradeoffs." State assumptions explicitly. Ask when ambiguous. Present multiple interpretations. Surface tradeoffs proactively.
2. **Simplicity First** — Minimum code that solves the problem. Nothing speculative. No features beyond what was asked. No abstractions for single-use code.
3. **Surgical Changes** — Touch only what you must. Every changed line traces to the user's request.
4. **Goal-Driven Execution** — "Define success criteria. Loop until verified." Transform imperative tasks into declarative goals: `"Add validation"` → `"Write tests, then make them pass"`; `"Fix the bug"` → `"Reproduce it in a test, then fix"`.

This is **a behavioral prompt**, not a dispatch unit. There is no YAML frontmatter, no `name:` / `description:` field, no trigger condition, no allowed-tools list. It's loaded as auto-context (every CLAUDE.md in the project hierarchy auto-loads on Claude Code session start), not invoked on demand.

### 2.2 Skill Data Shape

**The Karpathy file**: single `CLAUDE.md` at the project root. ~65 lines. Plain Markdown. No YAML frontmatter. No structure beyond `##` section headers for the four principles.

**Anthropic's "Claude Code skills" format** is different and is what the dev's question conflates. Per [Claude Code docs](https://code.claude.com/docs/en/skills) and [Anthropic's skills repo](https://github.com/anthropics/skills):

- Each skill is a **directory**. Conventionally `skills/<skill-name>/`.
- The directory contains a `SKILL.md` file with **YAML frontmatter** between `---` markers and a Markdown body.
- Frontmatter fields (per [skill authoring best practices](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices)):
  - `name` (required, ≤64 chars, lowercase + digits + hyphens only).
  - `description` (required, ≤1024 chars, non-empty — this is what Claude reads to decide WHEN to invoke the skill).
  - `disable-model-invocation` (optional bool — true means human-only, no auto-trigger).
  - `allowed-tools` (optional — restricts the skill's tool surface, e.g. `Read Grep`).
- The skill directory may contain auxiliary files: `scripts/` (executable Python/Bash), `references/` (documentation Claude can load progressively), `assets/` (templates, binary files).
- Body content guideline: **keep `SKILL.md` body under 500 lines** for context efficiency. Larger content is split into separate files in `references/` and progressively disclosed.

So: **a karpathy-skill (the repo) ≠ a Claude Code Agent Skill (the format)**. The repo's name evokes the Anthropic format but the content is plain CLAUDE.md content.

### 2.3 How Skills Are Invoked / Dispatched

**Karpathy CLAUDE.md** — auto-loaded on Claude Code session start. Always-on context. No invocation. No dispatch.

**Anthropic Claude Code skills** — three-stage progressive disclosure ([Anthropic engineering blog](https://www.anthropic.com/engineering/equipping-agents-for-the-real-world-with-agent-skills) + [Lee Han Chung's deep dive](https://leehanchung.github.io/blogs/2025/10/26/claude-skills-deep-dive/)):

1. **Discovery**: at startup, Claude Code reads ONLY each skill's `name` + `description` from `SKILL.md` frontmatter and pre-loads that metadata into the system prompt. This is just enough information for Claude to know when each skill might be relevant.
2. **Activation**: when a user message or task context matches a skill's description, Claude reads the FULL `SKILL.md` body and reasons further. This is the "skill is now in play" stage.
3. **Deep load**: only if the task demands more depth does Claude read supplementary files in `references/`, `scripts/`, or `assets/`.

The trigger mechanism is **semantic match against the description field**. Anthropic's authoring guidance: include both what the skill does AND specific triggering contexts in the description. All "when to use" info goes in the description, not the body. The model picks based on description-vs-task semantic similarity; deterministic triggering (file extension, command keyword) is not the primary path.

### 2.4 Karpathy-Skills vs Claude Code Built-In Skills

The two are different categories:

| Aspect            | `forrestchang/andrej-karpathy-skills`                                         | Anthropic Claude Code skills                                                  |
| ----------------- | ----------------------------------------------------------------------------- | ----------------------------------------------------------------------------- |
| Format            | Single `CLAUDE.md` file, plain Markdown.                                       | Directory with `SKILL.md` (YAML frontmatter + Markdown body) + optional `scripts/` `references/` `assets/`. |
| Loading model     | Auto-load on session start. Always-on.                                         | Progressive disclosure — name+description always loaded; body on match; aux files on demand. |
| Dispatch trigger  | None (always active).                                                          | Semantic match on `description` field.                                        |
| Scope             | Behavioral guidelines (how to think, what to avoid).                           | Task-specific procedures (how to do X).                                       |
| Invocation        | Implicit (context).                                                            | Implicit (auto-pick by description) or explicit (`/skill-name`).              |
| Tool restrictions | None — describes behavior, not tools.                                          | Optional `allowed-tools` frontmatter narrows the skill's tool surface.        |
| Scope in repo     | One file, applies whole project.                                               | One directory per skill, many skills per project / personal / plugin.         |

**Both can coexist**. CLAUDE.md is auto-loaded behavior; SKILL.md is opt-in procedure. The Tillsyn project itself uses both — `~/.claude/CLAUDE.md` + project `CLAUDE.md` for guidelines, `~/.claude/skills/qa-proof/`, `~/.claude/skills/qa-falsification/`, etc., for opt-in procedures. The skill list shown at the top of every Tillsyn session ("qa-falsification-checker", "semi-formal-reasoning", etc.) is the Anthropic SKILL.md format. The Karpathy file would be additive — drop it into project CLAUDE.md to bias all sessions toward the four principles.

### 2.5 Tillsyn-Fit Ideas — How Skills Could Map

The dev asked specifically about three things: (a) drop types, (b) action-item subgroups, (c) composability with the existing 12-kind closed enum.

#### 2.5.1 The Karpathy-File Mapping — Behavioral Bias On Builder Agents

The four-principles content is **pure orchestrator/builder behavioral bias**. It maps cleanly onto Tillsyn by being copied (or curl-installed) into:

- `~/.claude/CLAUDE.md` (global) — biases every session.
- `tillsyn/main/CLAUDE.md` (project) — biases Tillsyn-project sessions.
- `~/.claude/agents/go-builder-agent.md` (subagent definition) — biases the builder specifically.
- `~/.claude/agents/go-planning-agent.md` — biases the planner.

This is **not a Tillsyn data-model change**. It's content + a curl-equivalent. Drop 4c.6 doesn't need a new field for it.

#### 2.5.2 The SKILL.md Mapping — Per-Kind Or Per-Drop-Type Procedure Library

The richer concept is **Anthropic-format SKILL.md as a Tillsyn dispatch primitive**. Three hypothetical mappings, ordered by ambition:

**(A) Skill-as-spec-template (lowest ambition).** Tillsyn's `agents.toml` already has `client = "claude"`. Add an optional `skill = "<skill-name>"` field that gets passed to the Claude Code spawn as `claude --skill <name>` (if/when Claude Code's CLI supports per-spawn skill activation; today skills are session-wide). This gives the dispatcher a per-kind hook to bias an agent toward a specific procedure. **Composes cleanly with the closed enum** — `skill` is an optional metadata field, not a kind.

**(B) Skill-as-drop-type (medium ambition).** A "drop type" sub-categorization at the **drop** level, not at the action-item level. New project metadata field `drop.metadata.drop_type ∈ {"refactor", "greenfield", "research", "bugfix", "docs", ...}`. The drop type is **not** in the closed kind enum — it's an open-ended metadata tag that templates can dispatch on. Templates declare per-drop-type child_rules:

```toml
[template.child_rules.drop.refactor]
required_qa = ["build-qa-proof", "build-qa-falsification", "build-qa-regression"]
allowed_kinds = ["build", "research"]

[template.child_rules.drop.greenfield]
required_qa = ["build-qa-proof", "build-qa-falsification"]
allowed_kinds = ["plan", "build", "research", "discussion"]
```

The dispatcher reads `drop.drop_type` and the template's per-type rules; the kind enum stays closed. **Composes cleanly** — `drop_type` is open metadata; the kind enum is closed; templates encode the bridge.

**(C) Skill-as-second-axis-on-action-items (high ambition, NOT recommended for Drop 4c.6).** A second open enum on every `build` action item: `metadata.skill ∈ {"refactor", "greenfield", "test-only", "doc-only", ...}`. Dispatcher binds `(kind, role, skill) → agent + spec template + gates`. Each (kind, skill) combination potentially picks a different builder agent, a different QA gate, a different commit cadence rule. This **maps closely to the dev's "action item subgroups" question**. But it expands Tillsyn's dispatch matrix from 12 kinds × 9 roles × 4 structural-types to 12 × 9 × 4 × N skills. The combinatorial explosion is not a fatal blocker (most cells are unused), but it adds a fourth axis to a system that's still landing the third (`structural_type` is Drop 3). **Defer until Drop 5+ when cascade dispatcher is dogfooded and the need is concrete.**

#### 2.5.3 Composability With The Closed 12-Kind Enum

The kind enum was deliberately collapsed in Drop 1.75 to twelve values (`plan`, `research`, `build`, `plan-qa-proof`, `plan-qa-falsification`, `build-qa-proof`, `build-qa-falsification`, `closeout`, `commit`, `refinement`, `discussion`, `human-verify`). Closing the enum was load-bearing — the customization vocabulary moved to **template-bound sub-action-items** under specific generic kinds (per project `CLAUDE.md` line 86-90).

The clean composition rule is:

- **Kind stays closed**. New work-shapes do NOT get new kind values.
- **Skill / drop-type / sub-category become open metadata fields** that templates dispatch on.
- **Template `child_rules`** pattern-match on `(kind, role, skill?)` tuples to declare what children to auto-create, which agent to bind, which gates to run.
- The dispatcher looks up `agent_bindings[(kind, role, skill?)]` and falls back to `agent_bindings[(kind, role)]` when skill is unset.

This composes without breaking the enum and without introducing a new axis to the closed-enum vocabulary.

### 2.6 Tensions With Existing Tillsyn Vocabulary

Skill-as-Tillsyn-concept has three concrete tensions:

1. **Skill ≠ kind**. The dev's framing "drop types like refactoring vs building" sounds like new kinds — but `build` already covers both refactoring and greenfield work. The distinguishing factor is **how QA verifies** (refactor needs regression-equivalence proof; greenfield needs requirements-coverage). That's a gate-set difference, not a kind difference. **Resolution**: skill is a metadata tag the gate-runner reads, not a kind.
2. **Skill ≠ role**. A `qa-proof` builder doing refactor work is still `kind=build-qa-proof, role=qa-proof`. The skill biases the agent's **procedure**, not its **lane**. **Resolution**: skill orthogonal to role.
3. **Skill ≠ structural_type**. A `droplet` doing refactor work is still `structural_type=droplet`. **Resolution**: skill orthogonal to structural_type.

The fourth axis (skill) — if added — must be **opt-in, optional, defaultable**. Most action items will not declare a skill; the dispatcher must not require one. The closed-enum invariants of `kind` / `role` / `structural_type` must hold whether skill is set or not.

### 2.7 Recommendation For Drop 4c.6

For Drop 4c.6 specifically (the `agents.toml` runtime-config drop), the clean recommendation is:

- **Don't add a `skill` field to `agents.toml` in Drop 4c.6**. The drop's stated scope (`SKETCH.md` §10 "Out of scope") explicitly excludes prompt-shaping; skill belongs to that bucket.
- **Add a `claude_md_addons = []` optional list field** to `agents.toml` per kind. Each entry is a path to a CLAUDE.md fragment (e.g. `~/.tillsyn/agents-md/karpathy.md`, `agents-md/refactor-bias.md`) that the dispatcher concatenates into the spawn-prompt's CLAUDE.md context. **This is the Karpathy-content path** — no new dispatch axis, just a content-injection field. Cheap, composes with everything.
- **Defer the SKILL.md / drop-type / skill-as-fourth-axis question to a separate post-dogfood drop.** Document the option in Tillsyn's `WIKI.md § Cascade Vocabulary` as a "future extension space" so the closed-enum invariants stay protected and the dev can revisit when skill-vs-role-vs-kind tradeoffs are evidence-based rather than speculative.

Optionally, if `ta` ships claude-skills-aware schemas (per `ta`'s own pre-MVP roadmap referenced at `ta/CLAUDE.md` line 45-47 — `claude_hooks` / `claude_skills` / `claude_settings_fragments` schemas), Tillsyn can later point at `ta` for managing personal skill libraries the same way it points at `ta` for managing personal agent libraries — keeping the cascade-orchestration runtime separate from the personal-content-management workflow.

---

## Sources

### `ta` project — direct file evidence

- `/Users/evanschultz/Documents/Code/hylla/ta/main/README.md`
- `/Users/evanschultz/Documents/Code/hylla/ta/main/CLAUDE.md`
- `/Users/evanschultz/Documents/Code/hylla/ta/main/go.mod`
- `/Users/evanschultz/Documents/Code/hylla/ta/main/embed.go`
- `/Users/evanschultz/Documents/Code/hylla/ta/main/cmd/ta/main.go`
- `/Users/evanschultz/Documents/Code/hylla/ta/main/cmd/ta/init_picker.go`
- `/Users/evanschultz/Documents/Code/hylla/ta/main/internal/mcpsrv/{server.go,doc.go,tools.go}`
- `/Users/evanschultz/Documents/Code/hylla/ta/main/internal/{config,schema,templates,initapply}/doc.go`
- `/Users/evanschultz/Documents/Code/hylla/ta/main/examples/README.md`
- `/Users/evanschultz/Documents/Code/hylla/ta/main/examples/schemas/{cascade.toml,claude_agents.toml}`
- `git log --oneline -25` on `/Users/evanschultz/Documents/Code/hylla/ta/main`

### Karpathy-skills + Anthropic skills — WebSearch result excerpts (WebFetch was permission-denied)

- [`forrestchang/andrej-karpathy-skills`](https://github.com/forrestchang/andrej-karpathy-skills) — the repo itself.
- [The Four Principles | DeepWiki](https://deepwiki.com/forrestchang/andrej-karpathy-skills/3-the-four-principles) — clearest summary of the four sections.
- [Andrej Karpathy's CLAUDE.md: The 65-Line File With 100K GitHub Stars (Miraflow)](https://miraflow.ai/blog/karpathy-claude-md-100k-github-stars-ai-coding-2026) — line-count + star-count evidence.
- [`anthropics/skills`](https://github.com/anthropics/skills) — Anthropic's official skills repo.
- [Extend Claude with skills — Claude Code Docs](https://code.claude.com/docs/en/skills) — SKILL.md format.
- [Agent Skills — Claude API Docs](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/overview) — frontmatter fields, validation rules.
- [Skill authoring best practices — Claude API Docs](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices) — body-line limits, progressive disclosure pattern.
- [Equipping agents for the real world with Agent Skills (Anthropic engineering blog)](https://www.anthropic.com/engineering/equipping-agents-for-the-real-world-with-agent-skills) — three-stage discovery model.
- [Claude Agent Skills: A First Principles Deep Dive (Lee Han Chung)](https://leehanchung.github.io/blogs/2025/10/26/claude-skills-deep-dive/) — runtime trigger mechanism.

### Tillsyn project — direct file evidence

- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/CLAUDE.md`
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/workflow/drop_4c_6/SKETCH.md`
