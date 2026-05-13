# W1 — Plan QA Falsification (Round 1)

**Verdict:** FAIL — 3 CONFIRMED counterexamples (one CRITICAL cross-wave consistency
defect, one HIGH cross-wave-stale-assumption defect, one MEDIUM mergeTemplates
under-specification). Several NITs queued explicit ABSORB / DEFERRED-AS-NIT
dispositions.

The CRITICAL FF (FF1) demonstrates that, as currently planned, W1.D3 + W2.D5
+ W4.D1 together SHIP a silently broken project-tier override path: dispatched
agents will never resolve the `<project>/.tillsyn/agents/<group>/<basename>`
file that `till init --group go` plants on disk, because `resolveAgentGroup`
still returns `till-go` while W2 writes to `go/`. This is exactly the
shipped-but-not-wired anti-pattern Drop 3 droplet 3.20 institutionalized as a
load-bearing attack axis.

Per `NITs-first-class` (memory `feedback_nits_are_first_class.md`), every NIT
carries an ABSORB / DEFERRED-AS-NIT-with-reason disposition. None are
"judgment calls."

---

## Findings

### FF1 (CRITICAL) — Group-name drift between embedded dirs, project tier, and `resolveAgentGroup` default. D3 ships an invariant-violating project-tier lookup.

**Disposition recommendation:** ABSORB into round-2 — add an explicit D3
acceptance bullet that updates `agentBodyDefaultGroup` (and routes the
`till-go`/`till-gen` → `go`/`gen` rename of `resolveAgentGroup`'s embedded-tier
defaults) so the project-tier path the resolver builds MATCHES the path
`till init --group <group>` writes. OR explicitly route as a CROSS-WAVE
UNKNOWN with a routing decision (rename agentBodyDefaultGroup in this drop,
defer to 4c.7, or rename W4.D1 to also rename embedded dirs).

**Concrete trace:**

1. `internal/app/dispatcher/cli_claude/render/render.go:184` defines
   `agentBodyDefaultGroup = "till-go"` (with `till-` prefix).
2. `render.go:189` defines `agentBodyFallbackGroup = "till-gen"`.
3. `render.go:860–867` `resolveAgentGroup(binding)` returns either
   `path.Dir(binding.SystemPromptTemplatePath)` or `agentBodyDefaultGroup`
   (= `"till-go"`).
4. `render.go:646–667` `assembleAgentFileBody` calls
   `group := resolveAgentGroup(binding)` (line 663) and currently passes
   `basename` (not `group`) to `readProjectTierAgent`. D3's plan keeps
   `resolveAgentGroup` unchanged and threads `group` into
   `readProjectTierAgent`.
5. W4.D1's `Paths:` list (L1 PLAN.md lines 327–350) KEEPS embedded directory
   names `till-go/` and `till-gen/` (only deletes orphan files inside them and
   adds a NEW `fe/` dir with NO `till-` prefix).
6. W2.D1 acceptance (W2 PLAN.md line 44) renames `allowedInitGroups` from
   `["till-gen","till-go"]` to `["gen","go","fe"]`. W2.D5 acceptance writes
   project agents to `<destDir>/.tillsyn/agents/<group>/<name>.md` where
   `<group>` ∈ `{gen, go, fe}`.
7. Therefore, after W1.D3 + W2.D5 + W4.D1 all land:
   - `till init --group go` plants
     `<project>/.tillsyn/agents/go/builder-agent.md`.
   - But on dispatch, `assembleAgentFileBody` builds tier-1 path =
     `filepath.Join(project.RepoPrimaryWorktree, ".tillsyn/agents",
     resolveAgentGroup(binding), basename)` = `<project>/.tillsyn/agents/till-go/builder-agent.md`
     (when `binding.SystemPromptTemplatePath` is empty — the W3-FF5 LOCKED
     embedded default branch).
   - **The project tier always misses.** Falls through silently to user/embedded.
8. The whole stated objective of W1.D3 (REVISION_BRIEF §2.2 — "Update tier 1:
   change FLAT to subdir-per-group: `<project>/.tillsyn/agents/<group>/<name>.md`")
   is silently defeated.

**Why W1's `_BLOCKERS.toml` line 18 doesn't save us:** the BLOCKERS comment
says "if W4.D1 renames agentBodyDefaultGroup ('till-go' → 'go'), D3 updates
the constant in the same commit per U1". But (a) W4.D1's L1 PLAN.md acceptance
does NOT rename `agentBodyDefaultGroup` or rename `till-go/till-gen` embedded
dirs (read W4.D1 acceptance bullets at L1 PLAN.md lines 352–360); (b) there
is no U1 referenced anywhere in W1's PLAN.md (no Unknowns section exists);
(c) D3's acceptance bullets 1, 2, and 5 do NOT mention `agentBodyDefaultGroup`
or any equivalent rename. The whole graceful-coupling story is in a TOML
comment but not in any droplet's acceptance contract.

**Why this is structurally exactly the "shipped-but-not-wired" anti-pattern:**
schema/code shipped (project-tier subdir-per-group lookup in `readProjectTierAgent`),
producer shipped (W2.D5 ships to `<project>/.tillsyn/agents/<group>/`), but
**consumer's path** (the dispatched agent reading the project-tier file) never
hits because `<group>` and `<dirname>` diverge by `till-` prefix. Drop 3
droplet 3.20 was the canonical antecedent.

**Acceptance criterion that should be added in round-2:**

> D3 explicitly updates `agentBodyDefaultGroup` from `"till-go"` to `"go"` AND
> `agentBodyFallbackGroup` from `"till-gen"` to `"gen"`, OR W4.D1 is amended
> to rename embedded `till-go/` → `go/` and `till-gen/` → `gen/`. Either way,
> the dispatched agent's project-tier path MUST match
> `<project>/.tillsyn/agents/<group>/<basename>` where `<group>` is the bare
> group name (`go`, `fe`, `gen`) that `till init --group <group>` plants.
> Smoke test: a project with `<project>/.tillsyn/agents/go/builder-agent.md`
> on disk causes `assembleAgentFileBody` to return that body when
> `binding.SystemPromptTemplatePath == ""`.

Note that renaming embedded dirs (`till-go` → `go`) cascades through
`readEmbeddedTierAgent` (render.go:914+) which currently expects
`builtin/agents/<group>/<basename>` paths with `<group>` ∈
{`till-go`,`till-gen`}. Either the rename is W4.D1's job, OR D3 also patches
the cross-group fallback ladder. This is non-trivial scope — flag for dev
disposition (in-scope round-2 fix vs. defer to a follow-up drop with explicit
"users must run `mv .tillsyn/agents/go .tillsyn/agents/till-go`" instruction).

---

### FF2 (HIGH) — W2.D7 and W3.D1 are STALE about `ProjectMetadata.Groups`. W1.D2 ships the typed field; the consumers don't know.

**Disposition recommendation:** ABSORB into round-2 — add an L1-PLAN.md
Round-N changelog entry calling this out AND post advisory cross-wave notes
on both W2.D7 and W3.D1 (RiskNote/ContextBlock additions). W1's own plan
does not need a code change, but the L2 plan-QA pass on W1 must surface
this for orchestrator routing because W1.D2 is the producer of the
contract.

**Concrete trace:**

1. W1.D2 acceptance #1 (W1 PLAN.md line 278–280): "`domain.ProjectMetadata.Groups`
   field exists: `Groups []string` with JSON tag `json:"groups,omitempty"`."
   W1.D2 adds the typed field.
2. W1.D2 RiskNote (line 154–162, also lines 296+) says the W1 plan accounts
   for it: ProjectMetadata callers via LSP findReferences.
3. W2.D7 acceptance #1 (W2 PLAN.md line 296–298): "`Metadata.KindPayload` =
   JSON `{"groups": ["go","fe"]}` (or whichever groups were selected) —
   **used to persist group selection since `ProjectMetadata` has no typed
   `Groups []string` field**." (emphasis added).
4. W2.D7 RiskNote (W2 PLAN.md line 318): "`ProjectMetadata.Groups []string`
   does NOT exist as a typed field on `ProjectMetadata` (confirmed by reading
   `internal/domain/project.go`). Builder stores groups in
   `Metadata.KindPayload` as JSON. Adding a typed `Groups []string` field
   to `ProjectMetadata` is an `internal/domain` change outside W2's declared
   package scope — that refactor is a future drop concern." → STALE; W1.D2
   IS that change, and W1 is upstream of W2 (W2 blocked_by W1).
5. W2.D7 W2-GROUPS-R1 refinement (W2 PLAN.md line 350) says "Add typed
   `Groups []string` to `ProjectMetadata` and migrate from `KindPayload`
   JSON stopgap. Future drop owns `internal/domain`." → ALSO STALE; W1.D2
   does this in THIS drop.
6. W3.D1 RiskNote (W3 PLAN.md line 211): "`--add-group` and `--remove-group`
   flags are listed in REVISION_BRIEF §2.8 but `ProjectMetadata` has no
   `Groups []string` field today (check LSP on `domain.ProjectMetadata`
   before implementing). If absent, omit these flags and add a TODO comment
   for the future." → STALE; W1.D2 ships the field, and W3 is blocked_by
   W2 which is blocked_by W1.

**Why this matters for W1's own plan:** the L2 plan-QA pass is the only
place the downstream-staleness is visible during the planning phase
(plan-down direction). W1.D2 ships the contract; W2/W3 will consume it. If
the orchestrator dispatches W2.D7 / W3.D1 with their stale RiskNotes intact,
the builder will write JSON-stopgap code (W2.D7) and a `--add-group/--remove-group`
TODO comment (W3.D1) when the typed field is in fact available.

**Concrete impact if NOT fixed:** W2.D7 will land `KindPayload` JSON stopgap
even though typed `Groups []string` exists, creating a parallel-storage
schema drift. W3.D1 will leave a TODO and omit `--add-group/--remove-group`
even though they could be wired against the typed field.

**Acceptance criterion that should be added at the L1 plan level (not in W1
itself but recorded in W1's QA verdict for orchestrator routing):**

> Round-N changelog in L1 PLAN.md notes that W1.D2's `ProjectMetadata.Groups`
> ships in W1 (Wave B). Before dispatching W2.D7, update its RiskNote +
> acceptance #1 to populate `Metadata.Groups` (typed slice) directly,
> NOT `Metadata.KindPayload` JSON. Before dispatching W3.D1, update its
> RiskNote to wire `--add-group/--remove-group` against `Metadata.Groups`,
> not leave a TODO. Refinement W2-GROUPS-R1 is REFUTED — W1.D2 closes
> it inside this drop.

---

### FF3 (MEDIUM) — `mergeTemplates` scope is under-specified vs. the actual `templates.Template` struct shape.

**Disposition recommendation:** ABSORB into round-2 — add an explicit
acceptance bullet or RiskNote entry in W1.D2 enumerating what merge does
for EVERY field of `templates.Template`, OR explicitly scope the helper to
"merge `AgentBindings` only, all other fields take the LAST template's
values" with a documented rationale.

**Concrete trace:**

1. `internal/templates/schema.go:150–248` (`Template` struct) has 8 fields:
   `SchemaVersion`, `Kinds map[domain.Kind]KindRule`, `ChildRules []ChildRule`,
   `AgentBindings map[domain.Kind]AgentBinding`,
   `Agents map[domain.Kind]AgentRuntime`,
   `Gates map[domain.Kind][]GateKind`, `GateRulesRaw map[string]any`,
   `Tillsyn Tillsyn` (struct), `StewardSeeds []StewardSeed`.
2. W1.D2 acceptance #2 (W1 PLAN.md line 282–287): "Merges resulting
   `templates.Template` values: later groups win on `AgentBindings` key
   collision." — only `AgentBindings` is named.
3. W1.D2 RiskNote (W1 PLAN.md line 297–304): "`templates.Template` is a
   complex struct with maps (`AgentBindings`, `ChildRules`, etc.). Builder
   must inspect the struct shape... write a package-private
   `mergeTemplates(base, overlay templates.Template) templates.Template`
   that iterates map keys and overlays." — mentions `AgentBindings`,
   `ChildRules` but is silent on `Kinds`, `Agents`, `Gates`,
   `GateRulesRaw`, `Tillsyn`, `StewardSeeds`.
4. W1.D2 acceptance #5 case (c) only tests "collision on same kind key —
   last group wins" — implicitly `AgentBindings` only.

**Concrete counterexample:** consider a multi-group project with two
templates each declaring a different `[tillsyn]` table (e.g. `go.toml`
sets `max_context_bundle_chars = 50000`, `fe.toml` sets `100000`). The
merge helper is under-specified — does last group win on the entire
`Tillsyn` struct? Does it merge field-by-field? Two builder interpretations:

- **Interpretation A (last-template wins on the whole struct):** loses
  `go.toml`'s `requires_plugins` if `fe.toml` declares its own.
- **Interpretation B (field-by-field shallow merge):** has to define what
  "shallow merge" means for zero-valued vs explicitly-zero fields. Pre-MVP
  this matters less, but the spec under-specifies it.

For `ChildRules` (slice): concat? overlay-by-WhenParentKind? drop earlier?
For `StewardSeeds` (slice): concat or overlay? For `Gates`
(`map[domain.Kind][]GateKind`): per-key last-template wins? Per-key
concat? `GateRulesRaw` (`map[string]any`): shallow merge?

**Acceptance criterion that should be added in round-2:**

> W1.D2's `mergeTemplates` doc-comment AND acceptance bullets enumerate
> the per-field merge strategy for ALL 8 fields of `templates.Template`.
> Minimum: state explicitly that fields not exercised by W1's multi-group
> aggregator (`Tillsyn`, `StewardSeeds`, `Gates`, `GateRulesRaw`,
> `ChildRules`, `Kinds`, `Agents`) inherit the LAST-GROUP-WINS rule on the
> whole-field axis pre-MVP, with a refinement raised
> (MERGE-FIELD-AXIS-R1) to revisit if multi-group projects start setting
> these. OR scope the helper to `AgentBindings` only and document that
> other fields are taken from one canonical group (e.g. the first group
> with a non-zero value, or the primary `project.Language` group).

---

## NITs (cosmetic / micro-defects, with explicit dispositions)

### NIT1 — D2 signature contradiction between RiskNotes and AcceptanceCriteria.

**Disposition:** ABSORB into round-2 — pick one signature and use it
consistently across D1's RiskNote, D2's RiskNote, and D2's
AcceptanceCriteria.

**Trace:** W1 PLAN.md line 156–162 documents
`loadProjectTemplatesForGroups(project *domain.Project)` (no `homeDir`
param). Line 282–287 (D2 acceptance #2) documents
`loadProjectTemplatesForGroups(project *domain.Project, homeDir string)`
(WITH `homeDir` param). Line 306–314 (D2 RiskNote) introduces a NEW
symbol `loadProjectTemplateWithHome(project *domain.Project, homeDir
string)` as a helper that D1 ships and D2 calls per-group. Three
different signatures in one droplet — builder will guess.

Reason for ABSORB (not DEFER): the disagreement is in the load-bearing
acceptance bullets, not a comment. Round-2 should pick one.

### NIT2 — D2 acceptance #2 says "calls `loadProjectTemplate` for each group" but D1's `loadProjectTemplate` signature takes `project`, not `(project, group)`. Implicit signature change to D1.

**Disposition:** ABSORB into round-2 — either (a) D1's plan acceptance
explicitly includes a `group` parameter on the per-candidate helper used
in D2 (e.g. add `loadProjectTemplateWithHome(project, homeDir)` as a D1
symbol with the multi-group homeDir override), OR (b) D2 introduces the
group-axis dispatcher itself, NOT by calling D1's `loadProjectTemplate`
per-group.

**Trace:** D1 acceptance #2 (W1 PLAN.md line 230–232) says "`group` for
HOME tier = `strings.TrimSpace(project.Language)` when non-empty" — so
D1's `loadProjectTemplate` derives group from `project.Language` ONLY.
D2 says it "calls `loadProjectTemplate` for each group in `Groups`"
(line 282) — but D1 doesn't accept a `group` parameter, it reads
`project.Language`. To loop per-group D2 has to either (i) mutate
`project.Language` between calls (ugly), (ii) add a `group` parameter
to `loadProjectTemplate` (signature change), or (iii) inline the candidate
walk per-group (duplicates D1's code). The spec doesn't disambiguate.

### NIT3 — `Groups` json tag inconsistency (D2 acceptance: `json:"groups,omitempty"`; ProjectMetadata convention).

**Disposition:** ABSORB into round-2 — confirm `omitempty` works correctly
for `[]string` zero value (nil slice). Acceptance is fine as-is; the
NIT is whether the convention statement (line 152–153) "no TOML tag
needed on ProjectMetadata" is accurate — confirmed via
`internal/domain/project.go:119` (struct uses `json:` and `toml:` tags
on different fields; some have both, some have one). Doc-comment
clarity in builder code is the only thing to ensure.

Reason for ABSORB: micro-clarity for the builder, minimal cost.

### NIT4 — D2 RiskNote "Builder coordinates with D1's author (same builder in serial execution)" assumes serial author identity.

**Disposition:** DEFERRED-AS-NIT — reason: pre-cascade the orchestrator
is the dev manually, so "same author" is operationally true. Post-cascade,
the dispatcher serializes via `blocked_by` (which is correctly wired
W2.D2 blocked_by W1.D1). The risk-note prose is informal but not
load-bearing — `blocked_by` is the structural guarantee, not author
identity. Doesn't change builder behavior.

### NIT5 — D3 RiskNote claim "`render_test.go` is ~64K" is vague; actual line count is 1661 lines.

**Disposition:** DEFERRED-AS-NIT — reason: cosmetic editorial; the RiskNote
is directionally correct (high-churn risk) even if the K-size is rough.
Doesn't change builder behavior; line-count-vs-byte-count is a paraphrase
nit, not a load-bearing claim.

### NIT6 — `_BLOCKERS.toml` references "U1" but there is no Unknowns section in PLAN.md.

**Disposition:** ABSORB into round-2 — either delete the "per U1"
reference in `_BLOCKERS.toml` line 18 OR add an Unknowns section to
PLAN.md that defines U1 (best fit: rename strategy for
`agentBodyDefaultGroup` is unknown — see FF1).

Reason for ABSORB: orphan reference to a non-existent section is a
contract drift; small edit.

### NIT7 — KindPayload (D3) `shape_hint` reads "filepath.Join(worktree, projectAgentsSubdir, group, basename)" — does NOT include the `agentBodyDefaultGroup` rename or any group-vocabulary check.

**Disposition:** ABSORB into round-2 — folded under FF1's resolution
(any acceptance bullet that addresses FF1 also fixes this KindPayload
hint). Not a standalone NIT to disposition separately.

### NIT8 — D2 acceptance "Existing `ProjectMetadata` marshal/unmarshal round-trips unaffected (field is additive)" lacks a concrete callers-don't-break test.

**Disposition:** DEFERRED-AS-NIT — reason: D2 already says builder runs
`LSP findReferences` on `ProjectMetadata` to enumerate callers. The
"additive field" claim is correct for the named-field-literal usage
pattern (verified via Read on `repo_test.go:771` — uses named-field
literal). Adding a dedicated round-trip test is belt-and-suspenders;
`mage test-pkg ./internal/domain` covers the existing round-trip.

### NIT9 — D1 acceptance #2 "empty Language skips the HOME tier candidate" — what about whitespace-only Language?

**Disposition:** DEFERRED-AS-NIT — reason: D1 acceptance #2 already
specifies `strings.TrimSpace(project.Language)` — empty-after-trim is
treated as empty. Concern is REFUTED.

### NIT10 — D3 RiskNote: "package-private; only one call site today" assumes LSP findReferences is run; not a test gate.

**Disposition:** DEFERRED-AS-NIT — reason: the same RiskNote explicitly
instructs the builder to use LSP findReferences before editing. Process
note, not a code defect.

---

## Cross-Wave Consistency Summary

This is the most important section of this round's verdict — three findings
cross wave boundaries:

| FF | Producer wave | Consumer wave(s) | Symptom | Routing |
|----|---------------|------------------|---------|---------|
| FF1 | W1.D3 (resolver tier-1 path) + W4.D1 (embedded dirs) | dispatcher tier-1 lookup at runtime; ALSO W2.D5 ships into `go/`/`fe/` paths | Project tier never hits — silent fallthrough to embedded | MUST round-2: pick rename strategy (D3 patches consts OR W4.D1 renames dirs) |
| FF2 | W1.D2 (Groups typed field) | W2.D7 + W3.D1 | Downstream consumers use stale KindPayload JSON stopgap + leave TODO instead of typed-field wiring | MUST round-2: cross-wave changelog + advisory notes on W2/W3 |
| FF3 | W1.D2 (mergeTemplates) | (internal to W1) | Under-specified merge semantics for non-AgentBindings fields | Round-2: enumerate per-field strategy OR scope to AgentBindings only |

**Pattern observation (for L1 PLAN-QA-DISCIPLINE refinement):** the
"shipped-but-not-wired" anti-pattern (Drop 3 droplet 3.20) is reasserting
itself across wave boundaries when:

1. Producer wave ships a schema field (W1.D2 Groups).
2. Consumer waves PLAN BEFORE producer ships, so their plan-time research
   says "field absent."
3. Round-N plan-QA only attacks the producer wave's plan, not the consumer
   waves' staleness against the new contract.

This is a **plan-down vs build-up gradient** problem: L2 plans are authored
in parallel (W1/W2/W3 sub-planners spawn concurrently) but the L2 of an
upstream wave SHIPS a contract that the downstream L2 plans pre-conclude
won't exist. Routing this as a methodology refinement:

> **PLAN-QA-DISCIPLINE-R3 (proposed):** when an L2 plan SHIPS a new
> typed field, exported function, or constant that downstream L2 plans
> in the same drop will consume, the L2 plan-QA-falsification pass for
> the producer wave MUST also surface advisory notes for every downstream
> consumer L2 plan that pre-concluded the surface was absent. The
> orchestrator's round-2 absorb step propagates these notes to the
> consumer L2 plans before they dispatch.

Recommend that the dev disposition surfaces this as a permanent
methodology refinement in the L1 PLAN.md.

---

## Verdict Rationale

- **Why FAIL (not PASS-WITH-ABSORB):** FF1 is a CRITICAL cross-wave defect
  that, if dispatched as-is, ships a silently broken project-tier override
  path — exactly the shipped-but-not-wired anti-pattern. The fix is
  non-trivial (rename `agentBodyDefaultGroup` AND `agentBodyFallbackGroup`,
  cascade through `readEmbeddedTierAgent` and embedded dir layout, or rename
  in W4.D1). Dev disposition needed.
- **Why not "ABSORB inline":** FF1's resolution may require changing
  W4.D1's scope (rename embedded dirs) which is OUTSIDE W1 — dev must
  decide whether to expand W4.D1 or accept a more invasive D3 patch.
  FF2 requires changes to W2.D7 / W3.D1 RiskNotes — outside W1's L2 plan
  but visible only here.
- **No droplet-internal scope-creep:** the W1 plan internal to its
  declared 3 droplets is largely coherent. FF3 is the only internal
  under-specification; FF1 and FF2 are cross-wave.

---

## Counter-Evidence Considered (REFUTED attacks)

- **REFUTED: "missing blocked_by between D1 and D3"** — D1 and D3 touch
  disjoint packages (`internal/app` vs `internal/app/dispatcher/cli_claude/render`).
  Different Go compile units. No package-lock collision. Both blocked_by
  W4.D1 is correct.
- **REFUTED: "missing blocked_by between D2 and D3"** — same reason as
  above; disjoint packages.
- **REFUTED: "cycles in blocked_by"** — DAG verified:
  W4.D1 → {D1 → D2, D3}. No cycles.
- **REFUTED: "PLAN.md vs _BLOCKERS.toml drift on D2"** — PLAN.md line 270
  says "D2 blocked_by D1"; `_BLOCKERS.toml` line 12 says
  "W1.D2 blocked_by [W1.D1]". Consistent (ID-format differs cosmetically
  only).
- **REFUTED: "YAGNI — D1 and D2 could be merged"** — D2 changes
  `internal/domain` (adds Groups field) AND adds new coordinator in
  `service.go`. D1 only touches `service.go`. Merging would force two
  package edits in one droplet; the split serializes the
  `internal/domain` change after the `internal/app` HOME-tier extension,
  which keeps each droplet under sizing budget and lets the
  HOME-tier change land + test independently.
- **REFUTED: "YAGNI — D3 could be skipped (project tier already
  exists)"** — REVISION_BRIEF §2.2 explicitly requires the FLAT → subdir
  change for project tier; tier-1 currently DOES NOT thread group, only
  threads basename. This is the load-bearing change for the project
  override path. Cannot be skipped.
- **REFUTED: "test fixture churn risk for `render_test.go` is unstated"** —
  D3 RiskNote line 374–376 explicitly flags ALL render tests with fake
  project worktrees must update to subdir-per-group layout. Risk is
  acknowledged.
- **REFUTED: "missing acceptance for `mage ci`"** — Acceptance #6
  (`mage ci` green) covers it at the wave level. D1/D2/D3 each cite
  `mage test-pkg ./<package>`; the wave-level `mage ci` gate is named.
- **REFUTED: "platform.Paths.TemplatesDir gap"** — D1 explicitly decides
  to use `os.UserHomeDir()` directly, not `platform.Paths.TemplatesDir`
  (confirmed not to exist via dev verification). Consistent with
  `readUserTierAgent` pattern. Architectural justification provided.

---

## Hylla Feedback

N/A — this falsification pass operated against MD planning artifacts and
on-disk Go source via Read / LSP-equivalent rg. No Hylla queries were
performed (per spawn directive "Hylla is OFF").
