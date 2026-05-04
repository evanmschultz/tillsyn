# Drop 4c — Sketch Update + PLAN.md §19.10 Marketplace QA Proof

**Targets:**
- `workflow/drop_4c/SKETCH.md` (Theme F — Template ergonomics; Theme G — Post-MVP marketplace evolution)
- `PLAN.md` §19.10 marketplace evolution bullet (lines 1793–1800)

**Reviewer:** proof QA agent (filesystem-MD mode, read-only).
**Date:** 2026-05-03.

**Verdict:** **PASS-WITH-NIT** — all 7 specified checks pass on substance. Two internal-consistency nits surface around droplet-count rollups in `SKETCH.md` post-Theme-F insertion. No code edit, no new primitive, no scope drift.

---

## 1. Internal Consistency — Theme F Droplet Count

### 1.1 Stated rollup vs sub-droplet enumeration (NIT)

The Theme F section header at `SKETCH.md:86` reads:

> ### Theme F — Template ergonomics (~10–12 droplets)

But the F.1–F.6 sub-droplet annotations sum higher:

| Sub-theme | Annotation | Count |
| --------- | ---------- | ----- |
| F.1 — Project-template auto-discovery | `(~3 droplets)` (line 90) | 3 |
| F.2 — Generic + Go + FE builtin separation | `(~4 droplets)` (line 92) | 4 |
| F.3 — `till.template` MCP tool | `(~3 droplets)` (line 99) | 3 |
| F.4 — Marketplace CLI | `(~5 droplets)` (line 107) | 5 |
| F.5 — Extended validation | `(~2 droplets)` (line 115) | 2 |
| F.6 — KindTemplate stub cleanup | `(~1 droplet)` (line 121) | 1 |
| **Total** | | **~18** |

The "~10–12" rollup in the section header is **inconsistent** with the explicit F.1–F.6 sub-counts (~18). Two possible reads:

- Author intent: "~10–12" was meant for the whole Drop 4c BEFORE Theme F was inserted. Theme A (~4) + Theme B (~2) + Theme C (~3) + Theme D (~1–2) + Theme E (TBD) = ~10–13, matching the `Approximate Size` paragraph at `SKETCH.md:152–154`.
- Author intent: "~10–12" is meant to apply to Theme F itself but the F sub-annotations are upper bounds.

Either way, there is **drift**. The sub-droplet sum is the source of truth (concrete enumeration > rough estimate). Recommendation: either update Theme F's header to "(~15–18 droplets)" or trim the F sub-annotations to land within ~10–12. Same fix needed for the `Approximate Size` paragraph at line 154 if Theme F is intended to be in-scope for "~10–12 droplets" overall.

### 1.2 Approximate-Size paragraph stale (NIT)

`SKETCH.md:152–154`:

> ## Approximate Size
> ~10–12 droplets. Smaller than 4b.

This paragraph predates Theme F's insertion. Now that Theme F adds ~18 droplets to whatever Themes A–E contribute (~10–13), the drop is realistically ~25–30 droplets, NOT smaller than 4b. This is downstream of nit 1.1 and shares its fix.

---

## 2. Cross-Reference Accuracy — Cited Code Symbols

Each cited symbol verified via direct `Read` of the project source.

### 2.1 `internal/app/service.go:373` — `loadProjectTemplate()` (PASS)

`internal/app/service.go:373–375`:

```go
func loadProjectTemplate() (templates.Template, bool, error) {
	return templates.Template{}, false, nil
}
```

The doc comment at lines 363–372 confirms the SKETCH.md F.1 claim verbatim:

> Droplet 3.14 fills this in with file-system + embedded TOML resolution. Until then it returns (zero, false, nil), which routes the create path through the empty-catalog branch.

`SKETCH.md:90` states "currently returns `(zero, false, nil)` per Drop 3.14 deferral" — matches the actual code and the doc comment.

### 2.2 `internal/app/kind_capability.go:1002` — `mergeActionItemMetadataWithKindTemplate` (PASS)

`internal/app/kind_capability.go:1002–1004`:

```go
func mergeActionItemMetadataWithKindTemplate(base domain.ActionItemMetadata, _ domain.KindDefinition) (domain.ActionItemMetadata, error) {
	return base, nil
}
```

Doc comment at lines 995–1001:

> the legacy KindTemplate surface was deleted; the new templates v1 schema does not encode action-item metadata defaults on a KindRule, and the merge is now a pass-through. ... a future drop will fold it into the caller.

`SKETCH.md:121` states the function is "a no-op pass-through stub kept 'during the transition.' ... Doc comment confirms: *'a future drop will fold it into the caller.'*" — matches the actual code and doc comment exactly.

### 2.3 `internal/templates/load.go` validation chain (PASS)

`internal/templates/load.go:96–107` invokes the four named validators in order:

```go
if err := validateMapKeys(tpl); err != nil {
    return Template{}, err
}
if err := validateChildRuleKinds(tpl.ChildRules); err != nil {
    return Template{}, err
}
if err := validateChildRuleCycles(tpl.ChildRules); err != nil {
    return Template{}, err
}
if err := validateChildRuleReachability(tpl.ChildRules); err != nil {
    return Template{}, err
}
```

Function declarations:

- `validateMapKeys` — `internal/templates/load.go:161`.
- `validateChildRuleKinds` — `internal/templates/load.go:178`.
- `validateChildRuleCycles` — `internal/templates/load.go:196`.
- `validateChildRuleReachability` — `internal/templates/load.go:272` (no-op extension point, doc comment confirms).

All four functions exist and are called. `SKETCH.md:118` claim that `validateChildRuleReachability` "currently a no-op extension point" matches the doc-comment at `load.go:261–271` and the empty body at `load.go:272–275`.

### 2.4 `internal/app/service.go:716` caller of merge stub (claim only — F.6 fold target)

`SKETCH.md:121` mentions "this stub can fold into its caller (`internal/app/service.go:716`)" — this is a Drop 4c IMPLEMENTATION target rather than a cited current-state fact, so it's a forward-looking claim not strictly subject to current-state verification. Out of scope for this proof pass; if F.6 lands, the wave-plan QA at that time should confirm line 716 is still the caller.

---

## 3. Theme G ↔ §19.10 Parity

Each item enumerated and matched.

| `SKETCH.md` Theme G | `PLAN.md` §19.10 marketplace bullet | Match |
| ------------------- | ----------------------------------- | ----- |
| G.1 — TUI marketplace browser (`SKETCH.md:127`) | "TUI marketplace browser" (`PLAN.md:1794`) | YES |
| G.2 — Vector search (`SKETCH.md:128`) | "Vector search" (`PLAN.md:1795`) | YES |
| G.3 — User contribution flow (`SKETCH.md:129`) | "User contribution flow" (`PLAN.md:1796`) | YES |
| G.4 — Live-runtime validation / dry-cascade simulation (`SKETCH.md:130`) | "Live-runtime validation / dry-cascade simulation" (`PLAN.md:1797`) | YES |
| G.5 — Template inheritance / extends (`SKETCH.md:131`) | "Template inheritance / extends" (`PLAN.md:1798`) | YES |
| G.6 — Template-bound agent prompts (`SKETCH.md:132`) | "Template-bound agent prompts" (`PLAN.md:1799`) | YES |
| G.7 — Versioned template references (`SKETCH.md:133`) | "Versioned template references on Project" (`PLAN.md:1800`) | YES |

**7 of 7 items match in title and order.** Body text is paraphrase-equivalent, not verbatim duplicate — both surfaces preserve the same key technical claims (e.g., G.2 cosine-sim local against cached embeddings; G.7 `tillsyn-templates@v1.4.0/go-cascade` ref shape). Drift-free.

---

## 4. JSON→TOML Migration — `git ls-files` Verification

Command `git ls-files | rg "template.*\.json"` returned `NO_MATCHES`.

No tracked `*template*.json` files exist in the repo. The migration claim (templates are TOML end-to-end, no leftover JSON template files) is verified for tracked files. **PASS.**

Caveat: untracked or non-template files containing "template" + ".json" in their basename were not searched (e.g., `template.json` for an unrelated subsystem). The check as specified — "verify the JSON→TOML migration claim" — passes.

---

## 5. MCP Wire Format — TOML-as-String Sanity Check

`SKETCH.md:104–105` states:

> **Wire-format decision (locked):** TOML in, TOML out. The MCP argument `content` is a string carrying TOML text verbatim. Server parses TOML, validates, persists TOML.

Sanity-check against existing MCP request patterns in `internal/adapters/server/common/mcp_surface.go`:

- `CreateCommentRequest.BodyMarkdown string` (`mcp_surface.go:383`) — markdown text carried as a plain string, server-parsed downstream.
- Comment response shape `BodyMarkdown string \`json:"body_markdown"\`` (`mcp_surface.go:605`).

Pattern precedent: structured-text-as-string is an established MCP transport pattern in this codebase. The `body_markdown` example proves the wire format (JSON-RPC stringified payload, server-side parsing) is already exercised. The proposed `till.template(operation=set, content=<toml-string>)` shape is symmetric to existing surface design.

**PASS** on rationale.

---

## 6. Validation-Already-Built — Symbol Existence

All four validators verified above in §2.3:

- `validateMapKeys` — `internal/templates/load.go:161` — implemented (closed-enum membership check on `Template.Kinds` + `Template.AgentBindings` map keys).
- `validateChildRuleKinds` — `internal/templates/load.go:178` — implemented (closed-enum membership on `WhenParentKind` + `CreateChildKind`).
- `validateChildRuleCycles` — `internal/templates/load.go:196` — implemented (DFS with white/gray/black coloring; `ErrTemplateCycle` on back-edge).
- `validateChildRuleReachability` — `internal/templates/load.go:272` — extension point exists; body returns `nil` per documented Drop 3 design (every member of the closed 12-value Kind enum is reachable from project-creation, so reachability is trivially satisfied). The named function is the hook F.5's reachability-as-orphan-detection extension targets.

`SKETCH.md` F.5 claim "Currently a no-op extension point. Grow into kind-orphan detection" is **factually accurate** against current code.

**PASS.**

---

## 7. Pre-MVP Rule Preservation — Theme F vs Theme G Boundary

### 7.1 Theme F items reviewed against post-MVP-only primitive use

- **F.1 auto-discovery** uses `os.ReadFile` + `templates.Load(io.Reader)` — both already exist. No post-MVP primitive.
- **F.2 builtin separation** is a refactor of `internal/templates/builtin/` + the embed-resolver. Pure file reorganization + agent-binding split. No post-MVP primitive.
- **F.3 `till.template` MCP tool** uses existing MCP-handler wiring (`internal/adapters/server/common/mcp_surface.go` patterns) + `templates.Load` + `templates.Bake`. No post-MVP primitive.
- **F.4 marketplace CLI** uses git-shell-out (`git clone --depth 1`, `git pull`) + filesystem caching. No post-MVP primitive — it's bundled CLI surface, NOT vector search (G.2), NOT TUI browser (G.1), NOT versioned refs (G.7). The `till template list|fetch|show|install|validate` set is plain-text CLI on top of the cache directory.
- **F.5 extended validation** adds three new validator functions (`validateAgentBindingFiles`, `validateRequiredChildRules`, growth of `validateChildRuleReachability`). All extend the existing static-validation chain. No live-runtime simulation (that's G.4). No post-MVP primitive.
- **F.6 stub cleanup** is a fold-the-stub-into-its-caller refactor on existing code. No post-MVP primitive.

### 7.2 Theme G items reviewed against pre-MVP scope creep

- G.1 TUI browser — explicitly "Drop 4.5+ scope; FE/TUI track" — boundary clean.
- G.2 Vector search — needs marketplace-repo CI + embedding-download path + cosine-sim runtime — heavyweight, post-MVP.
- G.3 User contribution flow — GitHub PR review + signed templates — post-MVP governance.
- G.4 Live-runtime / dry-cascade simulation — "Heavier than Theme F.5's static checks; requires dispatcher reusability for simulation mode" — explicitly post-MVP.
- G.5 Template inheritance / extends — schema-level addition + bake-time merge — post-MVP refactor of `templates.Bake`.
- G.6 Template-bound agent prompts — sandboxing + adopter trust model — post-MVP security work.
- G.7 Versioned template references — `template_ref` Project field + update flow — post-MVP migration coordination.

**No item from Theme G is silently pulled forward into Theme F.** The boundary is explicitly stated at `SKETCH.md:123–125`:

> ### Theme G — Post-MVP marketplace evolution (NOT in Drop 4c scope; captured for persistence)
> Documented here so the design is preserved across compactions. **NONE of these land in Drop 4c.**

`PLAN.md:1793` mirrors the same boundary:

> **Marketplace evolution (post-Drop-4c)** *(captured during Drop 4a planning to persist scope across compactions; Drop 4c lands the MVP CLI surface; these items are post-MVP)*.

**PASS.**

---

## 8. Verdict Summary

| Check | Verdict |
| ----- | ------- |
| 1. Internal consistency — Theme F droplet count | **NIT** (drift between header "~10–12" and sub-rollup "~18") |
| 2. Cross-reference accuracy — cited code symbols | PASS |
| 3. Theme G ↔ §19.10 parity | PASS |
| 4. JSON→TOML migration claim | PASS |
| 5. MCP wire format (TOML in / TOML out) | PASS |
| 6. Validation-already-built claims | PASS |
| 7. Pre-MVP rule preservation (F vs G) | PASS |

**Overall: PASS-WITH-NIT.** Substantive claims all verify against current code + cross-doc parity. The only finding is the droplet-count rollup drift in `SKETCH.md` post-Theme-F insertion (sections 1.1 + 1.2). Recommendation: either update the Theme F header + Approximate Size paragraph to reflect ~15–18 droplets for Theme F alone (or ~25–30 total drop), or trim sub-droplet estimates to match the legacy "~10–12" rollup. SKETCH.md is a placeholder doc by its own admission ("**Status:** placeholder — NOT a full plan"), so the nit is low-stakes; full PLAN.md authoring at post-Drop-4b time will refine.

---

## TL;DR

- **T1**: Theme F sub-droplet annotations sum to ~18 but the section header + drop-level rollup say "~10–12" — internal-consistency drift. NIT.
- **T2**: All three cited code symbols (`loadProjectTemplate` at `service.go:373`; `mergeActionItemMetadataWithKindTemplate` at `kind_capability.go:1002`; `templates/load.go` validation chain) verified verbatim. PASS.
- **T3**: All 7 Theme G items match all 7 §19.10 marketplace sub-bullets in title and order. PASS.
- **T4**: `git ls-files | rg "template.*\.json"` returns no matches — JSON→TOML migration is clean for tracked files. PASS.
- **T5**: TOML-as-string MCP arg pattern is symmetric to existing `body_markdown string` precedent in `mcp_surface.go:383+605`. PASS.
- **T6**: All four validators (`validateMapKeys`, `validateChildRuleKinds`, `validateChildRuleCycles`, `validateChildRuleReachability`) exist in `templates/load.go` and are wired into `Load`. PASS.
- **T7**: Theme F items use only existing primitives (filesystem, `templates.Load`, MCP-handler wiring, git-shell-out, static validators). Theme G items all sit in genuinely post-MVP territory. Boundary is explicit and clean. PASS.
- **T8**: Overall verdict PASS-WITH-NIT — all substantive claims verify; only finding is the droplet-count drift in SKETCH.md, which is low-stakes given SKETCH.md's self-declared placeholder status.

---

## Hylla Feedback

`N/A — task touched non-Go files only` for the SKETCH.md and PLAN.md edits themselves. For the cited code symbols, evidence was gathered via direct `Read` on the named files at the cited offsets (faster than Hylla for this one-shot file:line verification), so no Hylla queries were issued. No Hylla miss to report.
