# Drop 4c F.7.18.5 — Default-template seeds for `[agent_bindings.<kind>.context]` in default.toml

## Round 1

### Goal

Seed the six in-scope `[agent_bindings.<kind>.context]` blocks into
`internal/templates/builtin/default.toml` per F.7.18.5 acceptance criteria
+ REV-4 contract (QA bindings MUST NOT pre-stage `parent_git_diff`).

### Files edited

- `internal/templates/builtin/default.toml` — added six
  `[agent_bindings.<kind>.context]` blocks under existing `[agent_bindings.<kind>]`
  sections (`plan`, `build`, `plan-qa-proof`, `plan-qa-falsification`,
  `build-qa-proof`, `build-qa-falsification`). Comment header above each block
  cites SKETCH:195-199 + master PLAN.md L13 FLEXIBLE-not-REQUIRED framing.
  REV-4 explicitly cited above every QA block.
- `internal/templates/embed_test.go` — added eight new test functions covering
  the F.7.18.5 acceptance contract. Imported `time` for duration assertions.

### Out of scope (deliberately deferred)

- The `[tillsyn]` table (max_context_bundle_chars / max_aggregator_duration)
  is mentioned in the F.7.18.5 plan but is NOT in this droplet's spawn-prompt
  scope. The spawn prompt narrows to "the six `[agent_bindings.<kind>.context]`
  blocks" — handled here. The `[tillsyn]` block is left for a separate droplet.
- `default-go.toml` / `default-fe.toml` split (Theme F.2) is explicitly out of
  scope per F.7.18.5 plan. Today only `default.toml` exists; this droplet
  edits that file.

### Per-binding seed shapes (committed verbatim)

```toml
[agent_bindings.plan.context]
parent = true
ancestors_by_kind = ["plan"]
delivery = "file"
max_chars = 50000
max_rule_duration = "500ms"

[agent_bindings.build.context]
parent = true
parent_git_diff = true              # ← only the build binding gets this
ancestors_by_kind = ["plan"]
delivery = "file"
max_chars = 50000
max_rule_duration = "500ms"

[agent_bindings.plan-qa-proof.context]
parent = true
ancestors_by_kind = ["plan"]
delivery = "file"
max_chars = 50000
max_rule_duration = "500ms"

[agent_bindings.plan-qa-falsification.context]
parent = true
ancestors_by_kind = ["plan"]
delivery = "file"
max_chars = 50000
max_rule_duration = "500ms"

[agent_bindings.build-qa-proof.context]
parent = true
ancestors_by_kind = ["plan"]
delivery = "file"
max_chars = 50000
max_rule_duration = "500ms"

[agent_bindings.build-qa-falsification.context]
parent = true
ancestors_by_kind = ["plan"]
delivery = "file"
max_chars = 50000
max_rule_duration = "500ms"
```

The other six bindings (`research`, `commit`, `closeout`, `refinement`,
`discussion`, `human-verify`) deliberately carry NO `[context]` block —
fully-agentic mode per master PLAN L13.

### Test coverage added

Eight new test functions in `internal/templates/embed_test.go`:

1. **`TestDefaultTemplateBuildContextSeedsParentGitDiff`** — positive
   assertion: `tpl.AgentBindings[domain.KindBuild].Context.ParentGitDiff == true`.
2. **`TestDefaultTemplateQABindingsRejectParentGitDiff`** — REV-4
   regression guard. Subtests for each of the four QA kinds asserting
   `Context.ParentGitDiff == false`. Catches future regressions where
   someone re-adds the field.
3. **`TestDefaultTemplateContextSeedsAncestorsByKind`** — every
   context-seeded binding declares `ancestors_by_kind = ["plan"]`. Subtests
   per seeded kind.
4. **`TestDefaultTemplateContextSeedsDelivery`** — every context-seeded
   binding declares `delivery = "file"`. Subtests per seeded kind.
5. **`TestDefaultTemplateContextSeedsCaps`** — every context-seeded binding
   declares `max_chars = 50000` and `max_rule_duration = 500ms`. Subtests
   per seeded kind.
6. **`TestDefaultTemplateContextSeedsParentTrue`** — every context-seeded
   binding declares `parent = true`. Subtests per seeded kind.
7. **`TestDefaultTemplateNonContextSeededKindsHaveZeroContext`** — the six
   kinds NOT in the seed list carry a zero-value `ContextRules`. Scope-creep
   guard. Subtests per non-seeded kind.
8. **`TestDefaultTemplatePlanContextHasNoDescendants`** — planner-flexibility
   cross-check: default plan binding has no `descendants_by_kind` (master
   PLAN L13 A-λ).

The companion negative-assertion tests for `build-qa-proof` /
`build-qa-falsification` /  `plan-qa-proof` / `plan-qa-falsification`
`ParentGitDiff == false` (acceptance criteria item from spawn-prompt) are
delivered by `TestDefaultTemplateQABindingsRejectParentGitDiff`'s four
named subtests.

### Verification gates

- **`mage testPkg ./internal/templates/`** — PASS, 355 tests across the
  package, all green.
- **`mage formatCheck`** — PASS, no Go format drift.
- **`mage build`** — PASS, binary builds.
- **`mage ci`** — partial pass. The `internal/templates` package passes
  cleanly; `internal/tui` and `internal/domain` failures observed are
  pre-existing in the working tree from prior unrelated edits to
  `internal/domain/project.go` + `internal/domain/project_test.go` (both
  show as `modified` in `git status` from BEFORE this droplet started).
  My edits are confined to `internal/templates/`. Orchestrator should
  treat the pre-existing failures as out-of-scope for F.7.18.5.

### Worked-through hard constraints

- DID NOT commit. Worklog written; orchestrator drives commits.
- DID NOT use `mage install`.
- DID NOT use raw `go test` / `go build` / `go vet`.
- DID NOT use Hylla calls (per spawn prompt).
- All edits scoped to two listed files + this worklog.
- REV-4 honored: zero `parent_git_diff` lines on any QA binding.
- Negative-assertion tests for the four QA kinds + the six non-seeded kinds.
- TOML structure preserved — only additive `[context]` blocks inserted
  between existing `[agent_bindings.<kind>]` sections.

### Suggested commit message (≤72 chars, conventional)

```
feat(templates): seed [agent_bindings.<kind>.context] in default.toml
```

(Body for the orchestrator if useful: "F.7.18.5 — six bindings seeded;
build alone gets parent_git_diff per REV-4.")

## Hylla Feedback

N/A — task touched only `internal/templates/builtin/default.toml` (TOML),
`internal/templates/embed_test.go` (Go test), and this worklog (MD). The
Go test edit was straightforward additive code with no need for
cross-package symbol search; spawn prompt explicitly forbids Hylla calls.
