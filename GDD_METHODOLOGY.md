# GDD Methodology — Graph-Driven Development

<!-- TODO populate post-dogfood -->

This document is a **placeholder**. Graph-Driven Development (GDD) is the
companion methodology to Cascade (see `CASCADE_METHODOLOGY.md`): where Cascade
governs how planning, building, and QA decompose down a tree of action items,
GDD governs how committed code is *understood* through a structural + semantic
graph (today: Hylla) so planners, builders, and QA agents read code-truth
rather than retrieval-truth. Substantive content lands **post-Hylla-rev /
post-dogfood** per `project_methodology_docs_tracker.md` and `SKETCH.md` § 14.2
— this stub reserves the slot, documents intent, and lists what the populated
doc must contain. Until then, the operational rules live in
`CLAUDE.md § Code Understanding Rules` and `WIKI.md`; treat those as the
authoritative pre-MVP source.

## Status

- **State:** placeholder. Do not treat the contents below as normative.
- **Populate after:** Hylla revision (next major Hylla release) lands and the
  Tillsyn dogfood drops have generated enough trace data to ground concrete
  claims.
- **Owner:** the dev; agents may not auto-populate this doc.
- **MVP-release blocker:** yes, per `project_methodology_docs_tracker.md`.

## Scope (when populated)

The full doc will cover, at minimum:

- The graph schema GDD assumes (nodes / edges / packages / blocks / values /
  derived edges) and how it differs from raw AST or LSP views.
- Evidence ordering — committed-code → graph → diff → external semantics —
  and why the graph layer earns its place between local symbol lookup and
  full-text search.
- Plan-time, build-time, and QA-time queries, with worked examples grounded
  in real Tillsyn drops.
- How GDD interacts with Cascade's Section 0 5-field certificate (Premises /
  Evidence / Trace or cases / Conclusion / Unknowns) — specifically which
  graph queries back which Evidence claims.
- Failure modes (stale ingest, schema drift, missing summaries, agent
  fall-through to raw `Read`/`Grep`) and the discipline that contains them.
- Benchmarks comparing graph-grounded planning + QA against retrieval-only
  baselines, once dogfood data exists.

## Prior-art research note (per `SKETCH.md` § 14.2.1)

Before populating, survey adjacent or prior work in graph-grounded
development methodologies — code-knowledge-graph systems, graph-RAG over
codebases, semantic-search-augmented coding agents, structural code search
tools. The populated doc must situate GDD against that landscape, name
where GDD overlaps with existing techniques, and name where it diverges.
This section is reserved for that survey; it is intentionally empty
pre-dogfood.

## Non-goals (explicit)

- **Not a Hylla user manual.** Hylla's tool surface is documented elsewhere;
  GDD is the methodology layer above any concrete graph-tool implementation.
- **Not a replacement for Cascade.** Cascade and GDD are companions —
  Cascade governs work-decomposition, GDD governs code-understanding.

<!-- END TODO -->
