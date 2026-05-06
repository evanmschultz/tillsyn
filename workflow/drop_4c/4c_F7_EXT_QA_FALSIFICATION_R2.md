# Drop 4c F.7.17 + F.7.18 — Architecture Falsification ROUND 2

**Scope:** re-attack on the SKETCH.md rework dated 2026-05-05 that addressed round-1's 9 CONFIRMED counterexamples (V1 / V2 / V3 / V5 / V6 / V10 / V11 / V12 / V14).
**Mode:** read-only adversarial review. 8 attack vectors A1–A8 from the spawn brief.
**Reviewer:** falsification subagent, fresh context.

Numbering matches the spawn-prompt vector list A1–A8 verbatim.

---

## A1 — argv-list form completeness

**Verdict: CONFIRMED-NEW (cluster of three holes the regex+validator pair does not fully close).**

### A1.a — Per-token regex passes `sh -c` shape

**Evidence.** SKETCH §F.7.17 line 157: per-token regex `^[A-Za-z0-9_./-]+$`. Walk through the dangerous shape:

```toml
command = ["sh", "-c", "claude"]
```

- `sh` → matches (lowercase letters only). Passes.
- `-c` → matches (hyphen + lowercase). Passes.
- `claude` → matches. Passes.

The regex stops shell *metachars within a single token*; it does NOT stop the dev (or marketplace template) from declaring `command[0] = "sh"` and `command[1] = "-c"`. Once `sh -c` is the spawned process, the third token is interpreted by the shell — and that third token can also pass the regex while doing harm:

```toml
command = ["sh", "-c", "rm"]   # all three tokens match the regex
```

Result: dispatcher runs `sh -c rm` which executes `rm` (no path/target arg → harmless on its own, but the shape proves the bypass). Worse:

```toml
command = ["sh", "-c", "rm", "-rf", "/Users/dev/.tillsyn"]
```

Every token passes the per-token regex. `sh -c "rm" "-rf" "/Users/dev/.tillsyn"` — `sh -c` only takes the first arg as script, so this specific shape is benign for `rm`, BUT the shape proves that `sh` as command[0] is the trapdoor regardless of the other tokens.

Actually-deadly shape: `sh -c claude` works in legitimate use cases (someone wraps claude in a shell script), but:

```toml
command = ["bash", "-lc", "/Users/evil/init.sh"]
```

If `/Users/evil/init.sh` exists and is dev-controlled, it runs in the orchestrator's worktree as the orchestrator's user. All tokens pass. Path-shape inside the third token (`/Users/evil/init.sh`) is allowed by the regex.

**Why round-1's V2 fix is incomplete.** V2 closed the in-token shell-metachar vector. It did NOT close the **command[0] = shell-interpreter** vector. The marketplace stricter validator (line 159) tries to close this via `tillsyn.allowed_commands` allow-list — but project-local templates BYPASS the allow-list (per line 159: "Project-local templates ... are dev-trusted and bypass the allow-list").

**Mitigation candidates.**

- Add `command[0]` denylist at template-load: `{"sh", "bash", "zsh", "ksh", "dash", "fish", "tcsh", "csh", "ash", "busybox", "env", "exec", "eval", "/bin/sh", "/bin/bash", "/usr/bin/env", "python", "python3", "perl", "ruby", "node"}` — a closed-enum reject list of known shell interpreters and code-runners. Trying to use one of these triggers a structured load-time error pointing the dev at the schema doc.
- OR (weaker): require the marketplace allow-list semantics for ALL templates, not just marketplace-sourced. Force the dev to sign off on every `command[0]` regardless of source.
- Document that project-local-template trust is a real surface — the marketplace shield is not the only attack path; a malicious PR adding `command = ["env", "FOO=bar", "claude"]` to `<project>/.tillsyn/template.toml` lands via PR merge and bypasses the marketplace allow-list.

### A1.b — Allow-list location authority is unspecified

**Evidence.** SKETCH §F.7.17 line 159: `tillsyn.allowed_commands` lives in `<project>/.tillsyn/config.toml`. The brief asks: where is this file authored and by whom?

- `<project>/.tillsyn/config.toml` is in the project worktree.
- The project worktree is git-tracked.
- A malicious marketplace template could ship its own `config.toml` alongside `template.toml` if `till template install` copies the whole `<name>/` subdir, OR a malicious dev / contributor could submit a PR that adds the allow-list entry alongside their `command` change.

**Verification.** SKETCH does NOT specify:

- Whether `till template install` only copies `template.toml` or also touches `config.toml`. (If it touches `config.toml`, the allow-list is self-modifying — exactly the attack the brief flagged.)
- Whether `config.toml` is a separate dev-only file that lives under `~/.tillsyn/` (the global Tillsyn config in `internal/config/config.go:29`) versus `<project>/.tillsyn/config.toml` (a NEW file the SKETCH introduces here for the first time).

The codebase today has only `~/.tillsyn/config.toml` (`internal/config/config.go`). The SKETCH coins `<project>/.tillsyn/config.toml` as a NEW concept for the allow-list, with no spec for who writes it.

**Mitigation.**

- SKETCH MUST specify: `tillsyn.allowed_commands` lives ONLY in `<project>/.tillsyn/config.toml`, this file is NOT modifiable by `till template install` (the install command writes only `template.toml`), and the file is NEVER shipped from the marketplace template. It is dev-authored, hand-edited, and treated as a dev-trust boundary.
- Add a load-time check: if the allow-list file appears in a commit alongside the marketplace `template.toml` install, refuse to install with "allow-list edits must be a separate dev commit, not bundled with template install."
- Confirm `<project>/.tillsyn/config.toml` is gitignored by default OR explicitly committed. Pick one and document.

### A1.c — `MatchString` vs explicit anchors silently differ

**Evidence.** Go's `regexp.Regexp.MatchString` searches for any substring match by default. `MatchString("^[A-Za-z0-9_./-]+$", "rm; ls")` returns true if the implementor forgets that the pattern is the complete regex source — Go's `regexp.MatchString(pattern, s)` requires the pattern itself to carry anchors. SKETCH writes the regex with explicit `^...$`, which is correct. But the SKETCH does NOT specify HOW the regex is invoked.

If a builder implements:

```go
matched, _ := regexp.MatchString("[A-Za-z0-9_./-]+", token)  // no anchors
```

…the validator silently passes `"rm; ls"` because the regex finds a sub-match `rm`. **Confirmed by Go regexp semantics**: `MatchString` returns true if ANY substring matches.

**Why this is a real risk.** Round-1 V2's mitigation is regex text. The implementor in Drop 4c builder could (a) forget anchors, (b) use `regexp.Match` instead of `regexp.MatchString` and forget anchors, (c) compile `regexp.MustCompile("[A-Za-z0-9_./-]+")` and call `.MatchString` (still substring-match without anchors).

**Mitigation.**

- SKETCH should commit to: validator uses `regexp.MustCompile(`^[A-Za-z0-9_./-]+$`).MatchString(token)` literally. Specify the implementation, not just the pattern. This is a small thing but it's exactly the class of bug that defeats security validators in practice.
- Add a unit test fixture in the schema-bundle droplet acceptance criteria: `["rm; ls"]` MUST fail load. `["valid_token"]` MUST pass. `["valid; injected"]` MUST fail. The test catches missing-anchors at builder-time, not in production.

---

## A2 — closed env baseline completeness

**Verdict: CONFIRMED-NEW (cluster: missing TMPDIR/XDG_CONFIG_HOME, ambiguous PATH default, platform silence).**

### A2.a — `XDG_CONFIG_HOME` and `TMPDIR` missing from baseline

**Evidence.** SKETCH §F.7.17 line 155: closed baseline = `PATH, HOME, USER, LANG, LC_ALL, TZ`. The Drop-4c spawn architecture memory (`project_drop_4c_spawn_architecture.md`) verified by 2026-05-04 CLI probes confirms two facts:

1. Claude reads its plugin / agent / settings files from a path resolved by `<user-home>/.claude/` by default. With `--bare` + `--plugin-dir <bundle>` + `--settings <bundle>/plugin/settings.json`, the resolution is bundle-relative, so `XDG_CONFIG_HOME` is NOT directly required for claude itself.
2. BUT: Go's `os.UserConfigDir()` and `os.UserCacheDir()` (which arbitrary spawned processes including subagent Bash invocations may call) read `XDG_CONFIG_HOME` / `XDG_CACHE_HOME` on Linux. With these stripped, processes silently fall back to `$HOME/.config` (POSIX default). Usually-fine-but-subtle.

The bigger gap is **`TMPDIR`**. Go's `os.MkdirTemp("", ...)` (which `internal/app/dispatcher/spawn.go` may use for per-spawn bundles per F.7.1) uses `os.TempDir()`, which on macOS reads `TMPDIR` and falls back to `/tmp`. Many tools (Bash, git) similarly read `TMPDIR`. With the closed baseline, the spawn writes its temp files to `/tmp` even when the dev's environment overrides to `/private/var/folders/.../T/` (macOS-default). Not a security flaw but a behavior-divergence between orchestrator and spawn — the spawn's tempdir is `/tmp` (often mode 1777 globally readable), the orchestrator's is per-user.

`SHELL` matters when a tool reads it for "default editor" (git config `core.editor` falls back to `$EDITOR` then `$VISUAL` then `vi`; `claude` itself doesn't appear to read `SHELL`). Low-impact gap.

`TERM` matters for TTY-aware rendering (`claude` headless probably doesn't care, but `git diff --color` does). Low-impact.

**Mitigation.**

- Expand baseline to include `TMPDIR` (high-impact for write-tempdir parity) and `XDG_CONFIG_HOME` / `XDG_CACHE_HOME` (Linux platforms). Document why each is on the list.
- OR document that closed baseline excludes these by design and any tool needing them must declare via `env = ["TMPDIR"]` per binding. Either is fine; pick one and document.

### A2.b — `PATH` value ambiguity (which PATH?)

**Evidence.** SKETCH says `PATH` is in the baseline but does NOT specify what value is used.

- **Restrictive default** (`PATH=/usr/bin:/bin`) means `claude` (typically installed at `~/.local/bin/claude` via npm/asdf/mise) is NOT on the spawn's PATH unless the dev's PATH brings it. Spawn fails ENOENT.
- **Inherit-PATH** (`PATH=os.Getenv("PATH")`) means any user-local bins on the orch's PATH are visible to the spawn — which DEFEATS the closed-baseline purpose. If the dev has `~/.local/bin` (containing `claude`), `~/work/scripts` (containing dev utilities), and `~/secrets/bin` (containing custom auth helpers) on PATH, the spawn sees all of them.
- **Hybrid** (filter out specific known-bad entries, keep the rest) requires a denylist with no specified content.

The SKETCH does not commit. Round-1 V3 fix added the closed baseline keys but did not pin the values.

**Why this matters operationally.** If `PATH=/usr/bin:/bin`, the spawn fails to find `claude` unless the dev sets `command = ["/full/path/to/claude"]` per binding. That's a fine and documented requirement, but the SKETCH should say so. Right now it's underspecified.

**Mitigation.**

- SKETCH should commit: default `PATH` value is `os.Getenv("PATH")` (full inheritance, with the recognition that PATH itself is a normal env var that orch and spawn share). Justify: PATH typically does not contain secrets; the closed baseline's purpose is to block direnv/AWS/STRIPE-style secret-bearing names, not to relocate binaries.
- OR: default is `/usr/bin:/bin:$HOME/.local/bin:$HOME/bin`, and the dev MUST add a per-binding `path_prefix = ["/opt/homebrew/bin"]` knob to extend. More restrictive but more explicit.
- Either way, document the choice. Today it's unspecified.

### A2.c — Platform silence (Windows / Linux / macOS)

**Evidence.** SKETCH says `PATH, HOME, USER, LANG, LC_ALL, TZ`. Windows uses `USERPROFILE` (not `HOME`), `Path` (not `PATH`, though Go's `os/exec` normalizes the first letter on Windows). The SKETCH does not say "macOS / Linux only."

**Verification.** Tillsyn today is Go cross-compile-able. Drop 4a's dispatcher in `internal/app/dispatcher/` does not have Windows-specific code paths, but neither does it have a "Windows not supported" disclaimer. The closed-baseline list will silently break on Windows users (zero `HOME` / `USER` / `LANG` env vars in a Windows shell — the equivalents are `USERPROFILE` / `USERNAME` / `LANG` is sometimes absent).

**Mitigation.**

- SKETCH should pick: "Drop 4c spawn pipeline targets POSIX (macOS / Linux) only; Windows support deferred to post-MVP refinement drop." OR ship `runtime.GOOS` switching with a separate Windows baseline.
- Lean toward "POSIX only" — the project's mage targets and dev tooling are macOS-first, and the current dev base is one macOS user. Document the constraint; don't paper over it.

### A2.d — Per-binding `env` cannot expand baseline through the `^[A-Z][A-Z0-9_]*$` regex

**Evidence.** Round-1 V3 mitigation added the env-name regex `^[A-Z][A-Z0-9_]*$`. Confirm the conventional names a dev might want to add:

- `XDG_CONFIG_HOME` → matches (uppercase + digits + underscore). OK.
- `XDG_CACHE_HOME` → matches. OK.
- `TMPDIR` → matches. OK.
- `SHELL` → matches. OK.
- `TERM` → matches. OK.
- `HTTP_PROXY` → matches. OK.
- `https_proxy` (lowercase, the conventional cURL form) → DOES NOT match. **REJECTED.** Devs who want to forward `https_proxy` (lowercase, cURL-canonical) cannot. They must rename to `HTTPS_PROXY` and hope the spawn'd binary reads the uppercase form. cURL reads BOTH but checks lowercase first; some tools (older Go binaries) only read the uppercase form.

**Mitigation.** Either:

- Relax the regex to allow lowercase: `^[A-Za-z][A-Za-z0-9_]*$`. POSIX env-var names are case-sensitive; lowercase is legal. Some conventional names (`https_proxy`, `no_proxy`) are conventionally lowercase. The strict-uppercase rule is over-specified.
- OR keep the strict rule and document that lowercase cURL conventions are explicitly excluded; devs use `HTTPS_PROXY` and accept that some tools may not read it.

Pick one. Round-1 V3 did not surface this case.

---

## A3 — schema-bundle-droplet ordering risk

**Verdict: CONFIRMED-NEW (single-point-of-failure framing + missing `[tillsyn]` top-level key).**

### A3.a — `[tillsyn]` top-level table is NEW and not in the bundle list

**Evidence.** SKETCH §F.7.18 line 199: `[tillsyn] max_context_bundle_chars = 200000`. SKETCH §F.7.18 line 200: `[tillsyn] max_aggregator_duration = "2s"`. SKETCH §F.7 line 232: schema-bundle droplet adds `tillsyn.max_context_bundle_chars + tillsyn.max_aggregator_duration globals`.

**Critical**: there is NO `Tillsyn` field on the `Template` struct today (verified via `internal/templates/schema.go:127-199`). The struct's TOML tags are `schema_version`, `kinds`, `child_rules`, `agent_bindings`, `gates`, `gate_rules`, `steward_seeds`. No `tillsyn` key.

Strict-decode (`internal/templates/load.go:88-95`) calls `DisallowUnknownFields()`. Per the loader's doc-comment: "an unrecognized key inside a [kinds.build] row become[s] StrictMissingError" — and the same applies to top-level keys. A `[tillsyn]` table in the TOML, with no matching `Template.Tillsyn` field, is rejected at load.

**This means the schema-bundle droplet MUST add a top-level `Tillsyn` struct + field on `Template` with a `toml:"tillsyn"` tag** to allow the `[tillsyn]` block. The SKETCH lists "tillsyn.max_context_bundle_chars + tillsyn.max_aggregator_duration globals" but does NOT explicitly call out the new top-level struct + Template field. A planner reading the SKETCH could miss that "globals" requires a struct shape, not loose top-level keys.

**Mitigation.**

- Schema-bundle droplet acceptance criteria MUST include: "(a) new `Tillsyn` struct on `internal/templates/schema.go` with two fields `MaxContextBundleChars int` (TOML tag `max_context_bundle_chars`) and `MaxAggregatorDuration Duration` (TOML tag `max_aggregator_duration`); (b) new `Tillsyn Tillsyn` field on `Template` struct with TOML tag `tillsyn`; (c) load-time validators reject zero / negative `max_context_bundle_chars` and zero / negative `max_aggregator_duration`."
- Without this explicit detail, the planner-author of the schema-bundle droplet may write the validators for the per-binding additions and miss the Template-level addition entirely. The bundled-droplet approach hides the heterogeneity.

### A3.b — Single-point-of-failure framing

**Evidence.** Round-1 V14 fix says ALL F.7 schema additions land in ONE droplet at the start of the F.7 wave (SKETCH line 232). This is a deliberate trade-off:

- **Pro**: keeps strict-decode coherent across the F.7 sequence — half-landed schema rejects later seeds.
- **Con**: one droplet bundles ~6 distinct schema changes (`command []string`, `args_prefix []string`, `env []string`, `cli_kind string`, `Context` sub-struct on `AgentBinding`, plus `Tillsyn` top-level struct with two fields). Each comes with its own per-field validator. Build-QA review surface = ~400 lines of struct + ~6 validator functions + ~6 unit-test tables.

**Risk model.** If the schema-bundle droplet's build-QA twin finds even one bug in any of the six additions, the WHOLE F.7 wave is blocked until rework lands clean. Other F.7 droplets cannot start because they consume the wider struct.

**Counter-design.** Three smaller schema droplets:

1. **Schema-1**: per-binding `command`, `args_prefix`, `env`, `cli_kind` (the F.7.17 surface). Land first.
2. **Schema-2**: `Context` sub-struct on `AgentBinding` (the F.7.18 per-binding surface). Land second.
3. **Schema-3**: `Tillsyn` top-level struct with `max_context_bundle_chars` + `max_aggregator_duration` (the F.7.18 globals). Land third.

Each droplet is ~1/3 the review surface. A bug in Schema-2 doesn't block Schema-1 (already landed) or Schema-3 (independent surface). The pelletier strict-decode concern (round-1 V14's rationale) is satisfied as long as the seeds (default-template `[context]` blocks) land AFTER Schema-2, and the `[tillsyn]` block AFTER Schema-3.

**SKETCH does not pick a side with reasoning.** The single-bundle decision was driven by V14's strict-decode argument, but it conflates "schema addition lands as one struct change" with "all additions land in one droplet." The two are separable: each droplet adds ONE struct field at a time, lands cleanly, the next droplet builds on the wider struct. Strict-decode is satisfied so long as the ORDER is preserved.

**Mitigation.**

- Reconsider the single-bundle decision. Either:
  - Stick with single bundle but acknowledge the SPOF risk in SKETCH prose: "schema-bundle droplet is the single largest review surface in F.7; QA should plan for ~2x review time vs typical F.7 droplet."
  - Split into 2-3 sequential schema droplets with explicit "this droplet adds <FIELD>; next droplet may use it" wiring. Strict-decode coherent so long as no droplet ships seed TOML referencing fields that haven't landed yet.

### A3.c — pelletier go-toml v2 strict-decode named-struct-field behavior is correctly understood

**Evidence.** Confirmed via `internal/templates/load.go:80-95` doc-comment: strict-decode `DisallowUnknownFields()` checks struct-tag membership. A NAMED struct field with a TOML tag IS automatically allowed (no separate "known field" registration). The SKETCH's claim (line 179) "decoded into a named Go struct on `AgentBinding` (NOT `map[string]any`), so `templates.Load`'s existing strict-decode chain ... automatically rejects unknown keys at load time. No new validator needed for unknown-key rejection." is **correct**.

But: the SKETCH claim assumes the reader knows that "named struct" means "struct with a TOML tag." A planner who creates the struct without the TOML tag (`type Context struct { Parent bool }` no tags) would silently break decode (pelletier matches by lowercased field name as fallback, which works for `parent` but not for snake_case fields like `parent_git_diff`).

**Mitigation.**

- Schema-bundle droplet acceptance criteria MUST require explicit TOML tags on every new struct field AND a unit test that rejects an unknown-key TOML payload to prove strict-decode actually fires for the new struct.

---

## A4 — multi-CLI roadmap claim that ConsumeStream refactor is additive

**Verdict: CONFIRMED-NEW (the "backward-compatible" claim is wrong-framed).**

### A4.a — Interface-shape change is BREAKING for any consumer of `ParseStreamEvent`

**Evidence.** SKETCH §F.7.17 line 168: future refactor replaces `ParseStreamEvent(line []byte) (StreamEvent, error)` with `ConsumeStream(ctx, io.Reader, sink chan<- StreamEvent) error`. SKETCH calls this "Backward-compatible refactor: existing claude + codex adapters keep their per-line logic, just wrapped in a scanner loop."

This is incorrect framing.

**The breaking part.** `CLIAdapter` is an interface. Adding `ConsumeStream` while keeping `ParseStreamEvent` is additive at the interface level — implementations get a new method to implement, but old method names stay. **However**: any consumer that calls `adapter.ParseStreamEvent(line)` from the dispatcher monitor WAS the contract surface. Replacing `ParseStreamEvent` with `ConsumeStream` (the SKETCH says "replace," not "add alongside") changes the dispatcher's call path:

- Before: `for line in scanner { event, err := adapter.ParseStreamEvent(line); ... }`
- After: `adapter.ConsumeStream(ctx, reader, eventSink); for event in eventSink { ... }`

These are not the same call shape. The dispatcher monitor (currently a planned consumer of `ParseStreamEvent` per F.7.4) MUST be rewritten to use the channel-sink shape. EVERY existing adapter MUST implement `ConsumeStream`. The SKETCH says they "keep their per-line logic, just wrapped in a scanner loop" — which is technically true at the per-line-parser level but obscures that the wrapping itself is a NEW method on the interface.

**Two valid migration stories (SKETCH picks neither).**

1. **Add-then-deprecate**: introduce `ConsumeStream` as additive; keep `ParseStreamEvent` for one drop; migrate the monitor to consume via `ConsumeStream`; deprecate `ParseStreamEvent` in a future drop. Both methods on the interface during the transition. Twice the per-adapter code. Truly backward-compat.
2. **Hard-cut**: rewrite the interface in one drop. All adapters MUST implement `ConsumeStream`; `ParseStreamEvent` is removed; the monitor is rewritten in the same drop. NOT backward-compat — every adapter is touched.

The SKETCH text "Backward-compatible refactor: existing claude + codex adapters keep their per-line logic, just wrapped in a scanner loop" papers over this distinction. A future planner reading this could believe the migration is zero-risk to existing adapters.

**Mitigation.**

- Reword the multi-CLI roadmap text to commit to one migration story:
  - "When the first non-JSONL CLI lands, the `CLIAdapter` interface gains a new `ConsumeStream` method ALONGSIDE `ParseStreamEvent`. Existing JSONL adapters get a default `ConsumeStream` implementation that wraps `bufio.Scanner` + their existing `ParseStreamEvent`. The dispatcher monitor migrates to consume via `ConsumeStream`. After all adapters are migrated, `ParseStreamEvent` is deprecated in a future cleanup drop." (Add-then-deprecate.)
  - OR: "When the first non-JSONL CLI lands, the `CLIAdapter` interface is rewritten: `ParseStreamEvent` removed, `ConsumeStream` added. All adapters AND the dispatcher monitor are refactored in the same drop. This is a coordinated breaking change across the dispatcher subtree." (Hard-cut.)
- Don't say "additive" when the interface contract changes.

### A4.b — `TerminalReport` shape is locked but `ExtractTerminalCost` signature drifts

**Evidence.** SKETCH §F.7.17 line 150 declares the third method: `ExtractTerminalCost(StreamEvent) (TerminalReport, bool)`. Line 169 declares `TerminalReport struct { Cost *float64; Denials []ToolDenial; Reason string; Errors []string }`. The naming is inconsistent: the method is called `ExtractTerminalCost` (cost-focused) but returns a `TerminalReport` (cost + denials + reason + errors). Future contributor reading `ExtractTerminalCost` would expect a single `cost` return; the struct shape signals broader scope.

**Mitigation.**

- Rename to `ExtractTerminalReport(StreamEvent) (TerminalReport, bool)`. Cosmetic but reduces drift.

---

## A5 — round-history "deferred" still has dangling references

**Verdict: REFUTED-WITH-NIT.**

### A5.a — `metadata.spawn_history[]` correctly framed as audit-only

**Evidence.** SKETCH §F.7.9 line 141: `metadata.spawn_history[]` is "append-only audit trail of `{spawn_id, bundle_path, started_at, terminated_at, outcome, total_cost_usd}`." Round-1 V12's concern was that `spawn_history` was being declared as a future-aggregator surface. Re-reading the post-rework F.7.9: the field is described as audit, no aggregator hook is mentioned. The aggregator (F.7.18 line 201) explicitly says: "`metadata.spawn_history[]` (F.7.9) remains an audit trail (cost, denials, terminal_reason) for ledger / dashboard, not for re-prompting."

Cross-check: F.7.9 lists fields that aggregator could read, but does NOT say "aggregator reads spawn_history." F.7.18's "Round-history aggregation: DEFERRED" line is consistent with F.7.9.

**Verdict on the round-1 fix**: the dangling reference IS gone. The role of `spawn_history` is correctly bounded to audit/ledger.

### A5.b — Future-contributor pointer is good but not in PLAN.md

**Evidence.** SKETCH §F.7.18 line 201: "If a concrete use case for raw stream-json round-history surfaces post-Drop-5, add it as a refinement-drop item with dedicated `prior_round_*` rules (`prior_round_worklog`, `prior_round_gate_output`, `prior_round_qa_findings`) that target the actual high-signal artifacts."

This text correctly points future contributors at the high-signal sources rather than re-litigating raw stream-json. **Nit**: the pointer is in SKETCH.md, which becomes scratch when PLAN.md authors. The full plan must surface this guidance in PLAN.md or in the F.7.9 / F.7.18 droplet description so it survives the SKETCH-to-PLAN handoff.

**Mitigation.**

- Add a one-liner in F.7.9 droplet acceptance criteria: "doc-comment on `metadata.spawn_history[]` MUST cite its audit-only role and link to the round-history-deferred decision in F.7.18 commentary." Survives the SKETCH retirement.

---

## A6 — drop-whole-rule cap algorithm corner cases

**Verdict: CONFIRMED-NEW (algorithm is unambiguous but produces poor behavior at the edge case the brief flagged).**

### A6.a — Brief's specific edge case is real and the SKETCH commits to the suboptimal answer

**Evidence.** SKETCH §F.7.18 line 199: "render rules in TOML declaration order; each rule contributes its full output unless cumulative size would exceed the bundle cap, in which case that rule and all subsequent rules are DROPPED WHOLESALE."

Walk the brief's example:
- Rule #1: 10KB (fits). Cumulative = 10KB.
- Rule #2: 220KB post-per-rule-trunc (would push cumulative to 230KB > 200KB cap). DROP rule #2.
- Rule #3: 50KB (would land at cumulative = 60KB, well under cap). **Per the algorithm, ALSO DROPPED.**

The brief flagged this. The SKETCH commits to the suboptimal answer ("all subsequent rules are dropped wholesale"). The dev's reasoning: "Adopters control priority by declaring most-important rules FIRST in their TOML." The implication is that rule #3 (declared after rule #2) is by definition lower priority than rule #2, so dropping it when rule #2 fails is acceptable.

**The flaw in this reasoning.** Adopters who declare the cheap rules first (parent, parent_git_diff — small) and the expensive rules later (deep ancestor walks — potentially large) get the OPPOSITE behavior from what they want: the small high-signal rules land, the large rule busts the cap, and... no further rules render. If the dev had ordered: cheap-1 (10KB) → expensive (busts) → cheap-2 (50KB), they expect cheap-2 to land. The algorithm drops cheap-2.

This is a real footgun. The dev's design rationale "adopters control priority by ordering" assumes monotonically-decreasing priority, which is not how cascade adopters actually think about context. They think "primary context first, supporting context second" — and primary may be small, supporting may be small-too-but-after-something-large.

### A6.b — Greedy interpretation is more useful

**Counter-design.** Greedy-fit algorithm:
- Iterate rules in declaration order.
- For each rule, if `cumulative + rule_size <= cap`, include it (cumulative += rule_size).
- Else, SKIP this rule (don't drop subsequent), continue to next rule.
- Emit a per-rule `[skipped: <rule_name> (would have added <N> chars; bundle remaining = <M>)]` marker for each skip.

This handles the edge case correctly: cheap-1 lands, expensive skipped, cheap-2 lands. Three markers (one per skip, one per land — actually just one for the skip).

**Trade-off.** The SKETCH's serial-drop algorithm has one nice property: **deterministic stop point**. If a rule is "load-bearing" (the dev needs it AND everything after it), serial-drop guarantees no partial bundle. Greedy could land rule #1 + rule #3 but skip rule #2, which produces a context that's missing a critical middle piece. The dev's choice depends on the failure mode they prefer:

- **Serial-drop**: "Drop everything after the bust. Force the dev to re-tune size budgets."
- **Greedy**: "Land what fits; mark what didn't. Dev sees partial context and decides whether to re-tune."

**SKETCH committed to serial-drop without surfacing the trade-off.** The brief asked: "Confirm that's the dev's decision and the algorithm is unambiguous." Answer: the algorithm is unambiguous in the SKETCH text; it's not clear the dev considered greedy as an alternative.

**Mitigation.**

- Either:
  - Defend serial-drop in SKETCH prose: "Serial-drop chosen over greedy because partial-context bundles (rule #1 + rule #3 without rule #2) are harder to debug than 'bundle cap reached at rule #2' — the dev can see exactly where the budget broke and re-tune, rather than guessing why rule #2 disappeared."
  - Switch to greedy: "Greedy-fit over declaration order: each rule is independently considered; rules that don't fit are skipped with markers; subsequent rules continue. Adopters who want strict-priority can reduce later rules' `max_chars` so they don't ever evict the bust-rule."
- Document the choice. Right now it's underspecified rationale.

---

## A7 — wall-clock-cap interaction with file-mode delivery

**Verdict: CONFIRMED-NEW (per-bundle cap pessimizes deep-tree adopters; per-rule alternative not considered).**

### A7.a — 2s per-bundle is too tight for `parent_git_diff` on large repos

**Evidence.** SKETCH §F.7.18 line 200: `[tillsyn] max_aggregator_duration = "2s"` per-bundle. Line 200 says "Aggregator enforces via `context.WithTimeout`; on hit, partial bundle + marker." The cap covers ALL aggregator runtime including:

- N action-item reads via Tillsyn MCP / SQLite (1-5ms each per V10's rough math from round 1 = 10-50ms for 10 reads).
- 1 `git diff` capture (filesystem op via `os/exec`; small change = 100-500ms; large change = 1-5s).
- File writes for delivery=file rules (~1-10ms each).

Realistic best case: 0.5s. Realistic worst case for a build deeply nested under 5 plan-segments with a 10000-line git diff: 3-5s.

**The 2s cap stops the slow case — but it stops it in the WORST way: partial bundle.** Marker `[aggregator timed out after 2s; rules pending: <list>]` means the agent received maybe 60% of the rules. The agent doesn't know which 40% it's missing in semantic terms (only the rule names from the marker). Agentic-mode (no `[context]` table) would have given the agent the choice of which context to fetch; bounded-mode-with-timeout gives it a partial-and-arbitrary slice.

### A7.b — Per-rule timeout is the natural alternative; SKETCH dismissed it

**Evidence.** SKETCH line 200: "Catches pathological-tree pre-spawn delays without per-rule depth-walk caps." This text dismisses per-rule caps in favor of per-bundle.

**Per-rule design.** `[agent_bindings.<kind>.context.limits] max_rule_duration = "500ms"`. Each rule independently enforces its own timeout. A slow `parent_git_diff` times out at 500ms with a per-rule marker; subsequent rules continue. Total bundle could be 6 rules × 500ms = 3s but each rule's failure is localized.

**Trade-off.**
- **Per-bundle**: fixed total cost ceiling (2s); arbitrary rule cuts.
- **Per-rule**: predictable per-rule cost; total bundle scales with rule count.

A blend (`max_rule_duration = "500ms"` + `max_aggregator_duration = "2s"` as a hard ceiling) covers both axes. The SKETCH only specifies the per-bundle cap.

### A7.c — Per-bundle vs per-rule semantic is implicit

**Evidence.** Brief asked: "Is `max_aggregator_duration` per-rule or per-bundle? SKETCH says 'per-bundle' implicitly."

Re-read SKETCH line 200 carefully: "**Wall-clock cap**: `[tillsyn] max_aggregator_duration = "2s"` (default). Aggregator enforces via `context.WithTimeout`; on hit, partial bundle + marker `[aggregator timed out after <duration>; rules pending: <list>]`. Catches pathological-tree pre-spawn delays without per-rule depth-walk caps."

The text "Catches pathological-tree pre-spawn delays without per-rule depth-walk caps" implicitly says "this is per-bundle, not per-rule." But it's not explicit. A reader could believe `max_aggregator_duration = "2s"` applies per rule (especially given `max_chars` is per rule).

**Mitigation.**

- SKETCH should say: "`max_aggregator_duration` is **per-bundle wall-clock**: total time across all rules. Per-rule timeouts NOT supported in initial scope; if a single rule (e.g. `parent_git_diff` on a giant change) routinely consumes the entire 2s budget, the dev should disable that rule for that binding rather than expect per-rule budgeting."
- OR: introduce both `max_rule_duration` (default 500ms per rule) AND `max_aggregator_duration` (default 2s per bundle) for the cleaner two-axis story.

---

## A8 — multi-CLI sequencing implication

**Verdict: REFUTED.**

### A8.a — Drop 4d post-Drop-5 sequencing is consistent with seam-validation goal

**Evidence.** SKETCH §F.7.17 line 175: "Sequencing: Drop 4c → Drop 5 (claude-only dogfood, validates the cascade-on-itself loop without conflating second-CLI integration risk) → Drop 4d (codex adapter) → Drop 5.5/6 (multi-CLI dogfood validation)."

Round-1 V9 fix landed sequencing locked as Option C. The brief's concern: dogfood validates cascade behavior on a single-adapter foundation, then codex lands, then multi-CLI dogfood. If Drop 4d surfaces a seam flaw, the fix lands AFTER Drop 5 dogfood — so dogfood ran on a soon-to-change foundation.

**Counter-argument.** The dev's reasoning per line 175 is "validates the cascade-on-itself loop without conflating second-CLI integration risk." This is correct: dogfood's primary risk surface is the cascade flow (auth, gates, conflict detection), not the LLM adapter. Adapter-specific issues that surface in Drop 4d are per-adapter contract bugs, not cascade-flow bugs. They get fixed in Drop 4d without invalidating Drop 5's findings.

**Seam-correctness pre-Drop-5 via MockAdapter.** Brief asked: "Verify SKETCH says explicitly that Drop 4c can validate the seam via a `MockAdapter` (test fixture, no real second CLI) so seam-correctness has confidence pre-Drop-5."

**Looking at SKETCH.** I don't see an explicit `MockAdapter` callout. The closest is line 230: "F.7.17 (CLI adapter seam) is internal-refactor-only inside Drop 4c — Drop 4c ships only the `claude` adapter. Drop 4d lands `codex`. The seam exists pre-Drop-4d so Drop 4d is purely additive." This says the seam exists, but does not commit to a test fixture proving the seam is multi-adapter-ready.

**Verdict: REFUTED-WITH-NIT.** The sequencing is sound. The seam-correctness story is missing an explicit `MockAdapter` test fixture commitment.

**Mitigation.**

- Add to F.7.17 acceptance criteria: "Drop 4c MUST ship a `internal/app/dispatcher/cli_adapter_test.go` with a `MockAdapter` test fixture that exercises the `CLIAdapter` interface contract WITHOUT touching `claude` or `codex` binaries. The test asserts: (a) `BuildCommand` returns an `*exec.Cmd` with the expected `Path`, `Args`, `Env` shape; (b) `ParseStreamEvent` round-trips a recorded fixture line; (c) `ExtractTerminalReport` correctly populates `TerminalReport` from a recorded terminal-event fixture. Confirms the seam is multi-adapter-ready before Drop 4d adds the second real adapter."

---

## Summary Table

| Vector | Round 1 verdict | Round 2 verdict | Severity |
|---|---|---|---|
| A1 — argv-list completeness | n/a (V2-derived) | **CONFIRMED-NEW** (3 sub-holes: `sh -c` bypass, allow-list location, MatchString anchors) | high |
| A2 — closed env baseline | n/a (V3-derived) | **CONFIRMED-NEW** (4 sub-holes: TMPDIR/XDG missing, PATH ambiguous, platform silent, lowercase env regex) | medium |
| A3 — schema-bundle SPOF | n/a (V14-derived) | **CONFIRMED-NEW** (Tillsyn struct missing from bundle list, SPOF framing, anchor-tag risk) | high |
| A4 — ConsumeStream "additive" | n/a (V1-derived) | **CONFIRMED-NEW** (claim is wrong-framed; rename `ExtractTerminalCost`) | medium |
| A5 — round-history dangling | V12 CONFIRMED | **REFUTED-WITH-NIT** (V12 fix airtight; nit on PLAN.md persistence) | low |
| A6 — drop-whole-rule edge | V6 CONFIRMED | **CONFIRMED-NEW** (algorithm unambiguous but suboptimal; no greedy alternative considered) | medium |
| A7 — wall-clock cap interaction | V10 CONFIRMED | **CONFIRMED-NEW** (2s per-bundle too tight; per-rule alt not surfaced) | medium |
| A8 — sequencing implication | V9 CONFIRMED | **REFUTED-WITH-NIT** (sequencing sound; MockAdapter fixture missing) | low |

**Overall verdict: NEEDS-REWORK.**

**Rationale.** Round 1 closed 9 specific holes; round 2 finds 6 NEW holes that the round-1 fixes either created (A3.a — `Tillsyn` struct field unaccounted for in the bundled droplet) or did not consider (A1.a `sh -c` bypass; A2.a TMPDIR/XDG; A4.a ConsumeStream framing; A6.b greedy alternative; A7.b per-rule timeout). Five of these are medium-or-high severity. Two are nits with clear mitigations.

**Path forward.**

1. **Must-fix before SKETCH lock**:
   - A1.a: add `command[0]` shell-interpreter denylist.
   - A1.b: spec `<project>/.tillsyn/config.toml` ownership + `till template install` non-modification.
   - A2.b: pick default `PATH` value (inherit-PATH or restrictive); document.
   - A2.c: declare POSIX-only or ship platform-switching.
   - A3.a: schema-bundle droplet acceptance criteria MUST list `Tillsyn` struct + `Template.Tillsyn` field explicitly.
   - A4.a: pick add-then-deprecate or hard-cut migration story; reword roadmap.

2. **Should-fix before PLAN.md authoring**:
   - A1.c: spec `regexp.MustCompile + MatchString` exact form.
   - A2.a: decide TMPDIR / XDG inclusion.
   - A2.d: decide lowercase-env regex relaxation.
   - A3.b: decide single-bundle vs split-schema droplet structure.
   - A6: pick serial-drop or greedy with explicit rationale.
   - A7: pick per-bundle vs per-bundle+per-rule timeout structure.

3. **Nice-to-have (NIT)**:
   - A4.b: rename `ExtractTerminalCost` → `ExtractTerminalReport`.
   - A5.b: surface round-history-deferred guidance in PLAN.md, not just SKETCH.
   - A8.a: add `MockAdapter` test fixture to F.7.17 acceptance criteria.

The round-1 rework is partially successful — it closed the original counterexamples — but introduced new attack surface (SPOF schema-bundle, ConsumeStream "additive" claim) and missed adjacent holes (TMPDIR, MatchString anchors, allow-list location, `sh -c` bypass) that should be closed before builders fire on this design.
