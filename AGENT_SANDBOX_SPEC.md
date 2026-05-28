# AGENT_SANDBOX_SPEC — locked consensus (2026-05-25)

> **Canonical, build-from spec** for per-dispatch agent sandboxing/gating across ta / hylla /
> tillsyn / sand. Supersedes the round-1/2 scratch docs (`*_RECS.md`, `*_REBUTTAL.md`,
> `*_SANDBOX_IDEA.md`, `AGENT_SANDBOXING_tillsyn.md`) — those are retained in git history only.
> Companion to `AGENT_DISPATCH.md` (the dispatch/routing model). Every mechanism below was
> reproduced on ≥2 independent macOS runs (Darwin 25, `claude 2.1.150`, `codex-cli 0.133.0`); the
> evidence + reproduction commands live in the superseded docs' git history. Windows/Linux is the
> one open axis (§8).

## 0. Two non-negotiable invariants

1. **GIT IS ORCHESTRATOR-ONLY.** No dispatched agent may ever `git commit/push/add/reset/checkout/
   branch/tag/stash/rebase/merge/restore` — enforced at the harness/OS layer, never by prompt.
2. **EDIT-SCOPE.** An agent may write ONLY the file(s) (claude) or directory (codex) granted at
   dispatch — regardless of what its prompt says. Off-scope writes FAIL; the agent reports the
   contradiction. Reads stay broad (read-confinement is not a goal).

## 1. Implementation rule for sand + tillsyn (LOAD-BEARING)

**sand and tillsyn dispatch + gate logic is GO CODE ONLY — never `bin/*.sh`.** sand is a Go MCP
server; tillsyn is Go. They do NOT shell out to `bin/agent-dispatch.sh` (that bash dispatcher is the
legacy bootstrap sand REPLACES, and is the other repos' interim only). Concretely:

- The **PreToolUse gate hook is a Go subcommand of the project's own binary** — `sand gate` / `till
  gate` — registered in **exec form** `{"type":"command","command":"<binpath>","args":["gate"]}`.
  (Go = zero runtime dep, `filepath` normalizes `\`/`/`, the binary already ships via `mage install`.)
- All per-channel dispatch (claude headless / built-in Agent-tool spec / codex exec) is built by Go.
- The ONLY `.sh` permitted is a **user's own hook** they choose to register — never sand's/tillsyn's
  dispatch or gate.
- sand/tillsyn are **config-driven translators**: users declare chains + limits in TOML; sand/tillsyn
  Go code TRANSLATES (`<project>/.claude/agents/<role>.md` persona + chain TOML → the per-channel
  invocation + the gate/sandbox artifacts) and ENFORCES them. No isolation policy hardcoded in Go.

## 2. Per-channel enforcement (consensus, all reproduced)

| Channel | Used for | git-block | edit-scope | clean context |
|---|---|---|---|---|
| **Claude built-in Agent tool** | OAuth/subscription roles (builder, plan-qa-proof, build-qa-proof, closeout) | Go `gate` PreToolUse hook (`bash_deny`) | Go `gate` hook, **per-FILE**, allowlist by `agent_id`/`agent_type` | ⚠️ **NOT clean** — built-in subagents ALWAYS load `~/.claude/CLAUDE.md` + project CLAUDE.md + memory (`sub-agents.md` "What loads at startup"; only Explore/Plan skip; no setting changes it). Gated on ACTIONS, not context. Persona carries Section 0 + behavior; hook is the security boundary. |
| **`claude -p --bare`** | API-key / ollama tier only (NOT OAuth — `--bare` needs an API key) | `--disallowedTools "Bash(git commit:*)" …` (deny wins) | `--allowedTools "Edit(//abs) Write(//abs) MultiEdit(//abs)"` **per-FILE, `//` double-slash**, + **no bare `Bash`** | `--bare` strips CLAUDE.md/memory/output-style for free |
| **`codex exec`** | planning, plan-qa-falsification, build-qa-falsification (+ dir-scoped editing on mac/Linux) | **execpolicy** `prefix_rule(decision="forbidden")` in hermetic `$CODEX_HOME/rules/default.rules` (NO `--ignore-rules`) | `--sandbox workspace-write -C <dir>` (**per-DIR**) or `read-only`; **`-c approval_policy="never"`** required | hermetic `CODEX_HOME` + `-c project_doc_max_bytes=0` |

**Settled facts (each reproduced ≥2x):**
- claude path rules need the **`//` double-slash absolute form**; single-slash denies everything.
- `claude -p --bare` requires an **API key, not OAuth** → OAuth roles MUST use the built-in Agent tool.
- codex `--sandbox` is **inert in `exec` without `-c approval_policy="never"`**; `-a never` is an invalid flag.
- codex **execpolicy** is the reliable, OS-independent git/command floor. codex's native `.git`-ro is a
  real bonus when `.git` is OUTSIDE the writable set `[workdir/-C, /tmp, $TMPDIR]` (true for real `$HOME`
  project trees with a subdir `-C`) — but is **void for `/tmp`-resident repos** and is writable-roots-
  dependent, so **execpolicy is PRIMARY**, native is defense-in-depth.
- codex PreToolUse hooks are **dead on 0.133.0 exec** — use execpolicy + `-C`, never codex hooks.
- codex MCP servers run **outside** the sandbox → injected `ta`/`hylla` MCP write records even under `read-only`.
- Hook runtime: **Go binary** (Claude Code bundles neither node nor python on Windows; bash is Windows-broken).
- **Shell-write-bypass must be blocked** (HYLLA_BIN §0): the gate denies Bash write-redirection that escapes the edit allowlist — `echo >`/`>>`, `tee`, `sed -i`, `cp`/`mv` into scope, `dd of=`, heredoc-to-file. Edit-scope on `Edit`/`Write` alone is insufficient; a builder can `echo x > off_scope.go` otherwise.
- **git-block must parse past global flags** (HYLLA_BIN §0): `git -C <dir> commit`, `git -c k=v commit`, `git --git-dir=… commit` defeat a flat `"git commit"` prefix match. Normalize past git's global flags before matching the subcommand.
- **codex hermetic adds `-c skills.bundled.enabled=false`** (HYLLA_BIN-verified) alongside `-c project_doc_max_bytes=0` + hermetic `CODEX_HOME` — disables bundled codex skills so the spawn is persona-only.

## 3. The gate contract (one shape, per-backend translation)

```
gate = {
  "edit":          ["//abs/file.go","//abs/file_test.go"],   # per-FILE (claude); [] = read-only role
  "writable_dirs": ["/abs/droplet-dir"],                      # per-DIR (codex)
  "bash_deny":     ["git commit","git push","git add","git reset","git checkout","git branch",
                    "git tag","git stash","git rebase","git merge","git restore",
                    "mage install","go get","go mod"],
  "network":       false
}
```
sand/tillsyn (Go) translate per channel + **validate mismatch**: a codex role given only `edit` (no
`writable_dirs`) → error "codex is dir-level; provide writable_dirs"; a built-in OAuth role dispatched
as a subprocess → refuse "use the Agent tool". Translation:

| gate field → | built-in Agent tool | `claude -p --bare` | codex exec |
|---|---|---|---|
| `edit` | hook allowlist (`<TA_ALLOWLIST>` by `agent_type`, or gates-file by `agent_id`) | `--allowedTools "Edit(//f) Write(//f) MultiEdit(//f)"` | → use `writable_dirs` (`-C`) |
| `writable_dirs` | (n/a) | (n/a) | `-C <dir0>` + `--add-dir <dirN>` |
| `bash_deny` | hook `bash_deny` | `--disallowedTools "Bash(git commit:*)" …` + no bare `Bash` | execpolicy `rules/default.rules` `prefix_rule(forbidden)` |
| `network:false` | (n/a) | (n/a) | `-c sandbox_workspace_write.network_access=false` |

## 4. Per-role tool / disallowed matrix (the per-role grant)

| Role | hylla | context7 | gopls | playwright | WebSearch | ta/till | edit | git |
|---|---|---|---|---|---|---|---|---|
| planning (go/fe) | **read-only** | ✅ | go only | fe only | ✅ | create+update | ❌ none (`edit:[]`) | ❌ |
| plan-qa-proof / -falsification | **read-only** | ✅ | go only | fe only | ✅ | update (verdict) | ❌ none | ❌ |
| builder (go/fe) | read-only | ✅ | go only | fe only | ✅ | update (comment) | ✅ **the 1–2 droplet files only** | ❌ |
| build-qa-proof / -falsification | ❌ **none** | ✅ | go only | fe only | ✅ | update (verdict) | ❌ none | ❌ |
| closeout | ❌ | ✅ | — | — | ✅ | update | ❌ | ❌ (orch commits) |

- **planning + plan-qa get hylla READ-ONLY** (no `hylla.ingest`/`config.refresh`) + context7 + gopls +
  WebSearch + ta create/update, **no source write**. **build-qa gets NO hylla** (just-shipped code isn't
  in the snapshot — relies on `git diff` + gopls/Read). **No role gets git mutation.**
- Enforcement: codex roles → role-conditional **MCP injection** (hylla only for planning/plan-qa; gopls
  for `*-go-*`; playwright for `*-fe-*`; context7 always; `web_search`) + execpolicy (git/command) +
  `read-only` sandbox for non-editing roles. claude roles → persona `tools:` allowlist + the Go gate hook
  (per-file edit for builder; `edit:[]` for qa/closeout) + `bash_deny`.

## 5. Cross-cutting (all four agree)
- **Veracity audit (HARD):** every channel RETURNS the full tool-call trace (claude `--output-format
  json`; codex exec stream; the sand/tillsyn MCP envelope MUST carry it). Every persona emits a
  `## Tools Used` section. The orchestrator audits claims against the stream — self-report ≠ truth.
- **Section 0 / reasoning lives in the PERSONA body**, never the output style (output style is ambient
  and stripped for small models; the persona always loads).
- **Configurable knobs** (sand/tillsyn own, not hardcoded): system-prompt mode (replace|append), the
  context strip-set, the sandbox mode per role, the gate spec.

## 6. Routing + project-owner chain spec (2026-05-25)

Per-role backend / model / effort (FE + Go identical; the canonical chains are
`hylla|ta|valv .claude/agent-chains.sh` and `sand/tillsyn .claude/sand-chains.toml`):

| Role | backend | model | effort / sandbox | channel |
|---|---|---|---|---|
| planning | codex-exec | gpt-5.5 | low, read-only | hermetic codex exec |
| plan-qa-proof | claude-native | opus | — | built-in Agent tool + gate hook |
| plan-qa-falsification | codex-exec | gpt-5.5 | high, read-only | hermetic codex exec |
| builder | claude-native | haiku (sonnet fallback) | — | built-in Agent tool + gate hook |
| build-qa-proof | claude-native | **sonnet** | — | built-in Agent tool + gate hook |
| build-qa-falsification | codex-exec | gpt-5.5 | low, read-only | hermetic codex exec |
| closeout | claude-native | opus | — | built-in Agent tool + gate hook |

- **Proof QA splits by axis: plan = opus, build = sonnet** (build-axis proof is the lower-stakes,
  cost-aware floor). **Falsification splits by effort: plan = high, build = low.** Codex model is
  **`gpt-5.5`**; the dispatcher adds `-c approval_policy="never"` so chain `opts` carry only
  `--sandbox <mode>` + `model_reasoning_effort`.
- **Builders are claude built-in haiku, NOT ollama.** The API-key / `claude -p --bare` ollama tier
  (§2) stays a config-driven flexibility option (needs a ≥~20b tool-capable model; 7b text-emits tool
  calls, unusable), but the owner's chains use claude-native haiku.

## 7. sand/tillsyn architecture (the build target)
sand (Go MCP) consumes: the chain TOML + the `[codex.hermetic]`/`[codex.mcp]` config + the per-role
gate spec, and **generates project-local artifacts** (no global state): the `sand gate` hook
registration in `settings.json`, the hermetic `CODEX_HOME/rules/default.rules`, the `--sandbox`/`-C`/
`approval_policy`/MCP-injection codex argv, and the claude `-p`/built-in invocation. It returns the full
tool-call trace. Same for tillsyn on the tillsyn substrate. The gate contract (§3) is the stable
interface; bin/sh is NOT part of sand/tillsyn (§1).

## 8. Open (needs a non-macOS box)
- Go exec-form hook firing on **native Windows + Linux**; codex Windows sandbox (exists, `[windows]
  sandbox`, but can fail to *initialize*, codex #24098 → **execpolicy is the floor that holds regardless**).
- The `claude -p` API-key tier cost vs ollama-≥20b vs built-in-subscription routing (product choice).

## 9. Evidence basis
Reproduced ≥2x on macOS by ta / hylla / tillsyn / sand 2026-05-25; commands + on-disk artifacts in
the superseded docs (git history). Vendor sources: Claude Code docs (`permissions` `//abs` +
deny>allow; `hooks` exec-form + no-bundled-node/python; `headless` `--bare`-needs-API-key); codex
docs (`exec-policy` `prefix_rule(forbidden)`; `concepts/sandboxing` writable-roots +
`<writable_root>/.git` ro; `config-reference` `approval_policy`/`[windows] sandbox`); codex issues
#16732 (hooks), #24098 (win init).

**Tillsyn-side re-verification 2026-05-27** — see §11 Empirical Verification. F1-F10 reproduce the
§2 settled facts against the current `bin/agent-dispatch.sh` (post-G7 patches at hash
`0927ffc32c56443c`, 670L) + raw direct invocations. Audit captures in
`tillsyn/main/.claude/agent-runs/`. Test plan A-G captured in Tillsyn action_item P2
(`a0a61f89-d5e5-4998-ba81-c1a43d8043e3`).

## 11. Tillsyn-side empirical verification (2026-05-27)

In-place reproduction from `tillsyn/main/` against the actual `bin/agent-dispatch.sh` (post-G7
patches) + raw direct invocations. macOS Darwin 25, claude 2.1.152, codex-cli 0.134.0, ollama
0.24.0 with gpt-oss:20b. All audit captures in `tillsyn/main/.claude/agent-runs/` (gitignored).

### F1 — codex `--sandbox read-only` blocks writes anywhere

Tested via bin/sh dispatch_codex (ta-go-build-qa-falsification persona). Three write probes:
- `/tmp/sandbox-tmp.txt` → rejected (`operation not permitted`)
- cwd-relative `sandbox-cwd.txt` → rejected
- cwd-absolute `/Users/.../tillsyn/main/sandbox-abs.txt` → rejected

Implication: `read-only` is the mechanical floor for read-only roles (planning, plan-qa,
build-qa). Sandbox is enforced at OS layer regardless of execpolicy.

### F2 — codex execpolicy `default.rules` blocks `git commit` at CreateProcess

Tested via bin/sh dispatch_codex. Hermetic `CODEX_HOME/rules/default.rules` containing
`prefix_rule(pattern=["git", "commit"], decision="forbidden")` rejects the command before
process creation. Error: `Rejected("/bin/zsh -lc 'git commit -m smoke' rejected: policy
forbids commands starting with git commit")`.

Implication: execpolicy is PRIMARY git-block (OS-independent, reproducible). Sandbox is
defense-in-depth (also blocks via write-confine for ref/index writes).

### F3 — codex role-conditional MCP injection per spec §4

Tested with `ta-go-build-qa-falsification` role. Agent reported `mcp__ta__` + codex's bundled
`mcp__codex_apps__github` only. NO `mcp__hylla__*`, NO `mcp__plugin_context7_*`, NO
`mcp__plugin_gopls_*` injected — exactly as `bin/agent-dispatch.sh` dispatch_codex lines
442/460/468 enforce (`ROLE != *build-qa*` excludes hylla/context7/gopls injection).

Implication: per-role tool grant per spec §4 enforced concretely via `-c mcp_servers.X={...}`
inline TOML config. NOTE: codex ships `mcp__codex_apps__github` as a bundled app even with
`skills.bundled.enabled=false`; investigate whether to gate this for build-qa.

### F4 — `claude -p` WITHOUT `--bare` auto-loads CLAUDE.md (cannot be bypassed by flags alone)

Tested raw (no bin/sh): `claude -p --model gpt-oss:20b ...` in `tillsyn/main/`. Prompt
asked for one Hard Rule. Model responded `"No human time estimates — use cascade‑shape work
estimates."` — verbatim from `tillsyn/main/CLAUDE.md` Hard Rules section. Input tokens:
**66,364** (tillsyn CLAUDE.md is 41.8K + memory + .claude/agents/ auto-discovery overhead).

Implication: subscription / OAuth Agent dispatch path ALWAYS loads CLAUDE.md and there is no
flag bypass. Only `--bare` (API-key-only path) strips CLAUDE.md auto-discovery.

### F5 — `claude -p --bare` strips CLAUDE.md / memory (~11× token reduction)

Tested raw (no bin/sh): same prompt under `--bare` + G7 env vars + 4 flags in
`tillsyn/main/`. Input tokens: **5,716** (vs 66,364 without --bare). Model gave generic
non-tillsyn-context response.

Implication: `--bare` is THE hermetic switch for `claude -p`. Requires API key (not OAuth) per
claude --help.

### F6 — `claude -p --bare + --allowedTools "Bash(pwd:*)"` ALLOWS pwd

Tested raw. Bash `pwd` succeeded → returned `/Users/.../tillsyn/main`.

### F7 — `claude -p --bare + --disallowedTools "Bash(git commit:*)"` DENIES git commit

Tested raw. Agent attempt to Bash `git commit -m test` → permission denied. JSON envelope
`permission_denials` array captured the attempt with full tool_input.

Implication: per-pattern Bash allow/deny works on `claude -p --bare`. Combined: `Bash(pwd:*)`
allow + `Bash(git commit:*)` deny is a real gate. Spec §3 gate contract translation row 3
verified for the claude -p row.

### F8 — JSON envelope `permission_denials` array is auditable

The `permission_denials[]` array in `claude -p --output-format json` envelope contains every
denied tool attempt with `{tool_name, tool_use_id, tool_input}`. This is the canonical audit
hook for the claude -p path — directly consumable by `bin/agent-audit-toon.py` (P2 audit
script).

### F9 — bin/sh dispatcher matches raw dispatch shape (no munging)

Compared bin/sh dispatch_codex (T1 + T3) against raw codex exec (T5). Same model, same
sandbox, same response shape. Differences are scope-of-test only; bin/sh adds:
- Hermetic CODEX_HOME with symlinked auth files (raw uses HOME .codex)
- execpolicy default.rules (raw lacks)
- Role-conditional MCP injection (raw lacks)
- `.claude/agent-runs/<base>.{out,err,meta.json}` audit persistence

Implication: bin/sh wraps codex with hermetic + audit + MCP injection; no transparent munging.

### F10 — G7 recipe (3 env vars + 4 flags) works alongside --bare in real call

Tested via bin/sh dispatch_ollama (T2) AND raw claude -p --bare (T-E). All 7 G7 elements
co-existed in the argv + env without conflict:
- ENV: CLAUDE_CODE_DISABLE_CLAUDE_MDS=1, CLAUDE_CODE_DISABLE_AUTO_MEMORY=1,
  CLAUDE_CODE_DISABLE_GIT_INSTRUCTIONS=1
- FLAGS: --exclude-dynamic-system-prompt-sections, --strict-mcp-config,
  --disable-slash-commands, --settings '{"outputStyle":"default"}'

Implication: G7 §10 recipe is wiring-compatible with the existing --bare + --mcp-config +
--allowedTools shape. Belt-and-suspenders context-strip beyond what --bare alone provides.

### F11 — codex `--sandbox workspace-write -C <dir>` writable roots = `[workdir, /tmp, $TMPDIR]`

Tested raw (T-H retry, codex header shows `sandbox: workspace-write [workdir, /tmp, $TMPDIR]`):
- `/tmp/codex-test-h/inside.txt` (workdir = -C) → **success**
- `/tmp/other-h.txt` (/tmp root, NOT under -C) → **success**
- `/Users/evanschultz/codex-outside.txt` (HOME root, NOT in any writable root) → **rejected:
  `operation not permitted`**

Implication: codex's per-dir `-C` is **not really per-dir** — `/tmp` + `$TMPDIR` are ALWAYS
writable in workspace-write mode, regardless of `-C`. Sandbox enforces "outside-all-writable-
roots" denial. Per-file edit-scope is IMPOSSIBLE under codex: you can only narrow writable
roots to `[workdir, /tmp, $TMPDIR]`. This is why **per-file editing builders MUST be claude -p
--bare or built-in Agent tool**, never codex.

### F12 — Hylla MCP injection through bin/sh codex works end-to-end

Tested bin/sh dispatch_codex (T-J, ta-go-planning role). Codex stream emitted:
- `mcp: hylla/hylla.search.keyword started`
- `mcp: hylla/hylla.search.keyword (failed)`

The MCP was successfully INJECTED + REACHABLE + INVOKED. The `(failed)` is a data-level error
("Attribute content is not indexed with type term") — meaning the artifact's index schema
mismatch or wrong query field. Infrastructure side: dispatcher's hermetic codex argv with
`-c mcp_servers.hylla={...startup_timeout_sec=15,tools={...}}` works.

Implication: `mcp: <server>/<tool> started|completed|failed` is the canonical codex stream
marker format. `bin/agent-audit-toon.py` (P2) parses these into the same 15-bucket TOON
schema as claude code's `tool_use` entries.

### F13 — `claude -p --bare + --plugin-dir <path>` does NOT register plugin MCPs (CONFIRMED 2026-05-27)

Tested via three-way comparison PL-A vs PL-B vs PL-C, same ollama gpt-oss:20b backend, same
prompt asking for visible MCP namespace prefixes + browser_* tools:

| Config | Input tokens | Plugins visible |
|---|---|---|
| PL-A: `claude -p` NO --bare (no flags) | 74,577 | 5 MCPs (hylla, tillsyn, tillsyn_dev, plugin_context7_context7, plugin_playwright_playwright) + 23 browser_* tools |
| PL-B: `claude -p --bare` NO --plugin-dir | 834 | **ZERO** |
| PL-C: `claude -p --bare + --plugin-dir <playwright>` | 838 | **ZERO** |

**Conclusion**:
1. **`-p` WITHOUT `--bare` is the all-or-nothing path**: plugins + CLAUDE.md + memory ALL
   load. There is NO flag combination that disables CLAUDE.md or plugin auto-sync without
   --bare. ~74K input tokens of unavoidable context bleed in this project.
2. **`-p --bare` strips ALL plugins** (and CLAUDE.md, memory, auto-discovery). Plugin sync
   is one of the things --bare disables per claude --help (`"skip ... plugin sync ..."`).
3. **`--plugin-dir` under --bare does NOT register plugin MCPs**. Per claude --help,
   --plugin-dir "Load a plugin from a directory or .zip for this session only" — but
   empirically, the plugin's MCP servers do NOT become available to the agent. Likely
   --plugin-dir loads plugin slash commands + agents, not MCPs. Or the plugin's MCP needs
   marketplace-style install to register.

**Implications for the spec**:

- For **ollama-headless / API-key claude -p builders**: stuck with `--mcp-config <file>` for
  MCPs (proven in T2). Plugin-shipped MCPs (`mcp__plugin_*`) are NOT reachable. If a builder
  needs Playwright / context7 / gopls-lsp at the MCP layer, they MUST be invoked via:
    - **Codex exec** (where bin/sh dispatcher injects them inline via `-c mcp_servers.*`); OR
    - **Built-in Agent tool** (OAuth, where plugins auto-load).

- For **`-p` without --bare**: gets everything but at high token cost (~74K input bleed) AND
  cannot be context-stripped. Not viable as a hermetic build path.

- For **built-in Agent dispatch (OAuth)**: plugins always loaded; gated by hook + persona
  `tools:` allowlist, NOT by context strip. Spec §2 confirmed.

**Per-channel plugin-MCP reachability table (added 2026-05-27)**:

| Channel | Plugin MCPs (Playwright/context7/gopls) | How |
|---|---|---|
| Built-in Agent tool (OAuth) | ✅ auto-loaded | enabledPlugins in settings.json |
| `claude -p` WITHOUT --bare | ✅ auto-loaded | plugin auto-sync |
| `claude -p --bare` | ✅ if dispatcher translates plugin manifest into `--mcp-config` (F14) | hand-crafted `mcpServers` JSON pointing at the plugin's command+args (e.g. `{"plugin_playwright_playwright":{"command":"npx","args":["@playwright/mcp@latest"]}}`) |
| `codex exec` | ✅ if injected by dispatcher | bin/sh dispatcher's inline `-c mcp_servers.playwright={...}` config injection |

### F14 — Plugin MCPs CAN register under `--bare` via `--mcp-config` (PROVED 2026-05-27)

The "low-priority research" question framed under F13 is **resolved**: plugin MCPs register
cleanly under `claude -p --bare` when the dispatcher hands `--mcp-config` an `mcpServers`
block mirroring the plugin's own `.mcp.json` entry.

Empirical run 2026-05-27 (action_item `b282a5eb-aa65-4621-a92d-8ba2549e7fc3`):

1. Read plugin manifest: `~/.claude/plugins/cache/claude-plugins-official/playwright/unknown/.mcp.json`
   = `{"playwright":{"command":"npx","args":["@playwright/mcp@latest"]}}`.
2. Wrote `/tmp/mcp-config-playwright-pre-t-i.json` with `{"mcpServers":{"plugin_playwright_playwright":{"command":"npx","args":["@playwright/mcp@latest","--headless","--isolated"]}}}`
   (using the `plugin_<name>_<server>` key so the tool namespace matches what plugin auto-loading would emit).
3. Ran `claude -p --bare --output-format stream-json --model gpt-oss:20b` (ollama @ 11434) +
   full G7 recipe (3 env vars + 4 flags) + `--mcp-config <tmpfile>` + `--allowedTools "mcp__plugin_playwright_playwright__browser_navigate ..."`.
4. Init event reported: `mcp_servers:[{name:"plugin_playwright_playwright",status:"connected"}]`
   + `tools[]` array contained all 23 `mcp__plugin_playwright_playwright__browser_*` tools.
5. Tool call FIRED: assistant emitted `tool_use` for `browser_navigate(url=about:blank)`;
   server returned actual snapshot (`page-2026-05-27T18-45-12-852Z.yml` + `await page.goto('about:blank');`
   confirmation). `permission_denials:[]`.

**Implication**: FE roles (and any plugin-needing builder/QA role) CAN route through
`claude -p --bare` (clean-context + per-file edit-scope via `--allowedTools`) IF the
dispatcher reads the plugin manifest at dispatch time and injects it as `--mcp-config`.
This unlocks the API-key / ollama tier for FE work; previously only built-in Agent
(OAuth + plugins auto-loaded but no per-file edit-scope) or codex (with inline injection but
dir-only scope, no per-file gate) could reach Playwright.

**Dispatcher follow-up** (NOT this phase): `bin/agent-dispatch.sh dispatch_ollama` (and any
future `dispatch_claude_p_bare`) should grow optional plugin-manifest injection for FE
roles. Reads `~/.claude/plugins/cache/<marketplace>/<plugin>/<rev>/.mcp.json`, normalizes
each entry to `mcpServers."plugin_<plugin>_<server>"`, writes a temp config, passes via
`--mcp-config`. Per-role allowlist in `agent-chains.sh` declares which plugins each role
needs. Same pattern translates 1:1 to sand's Go MCP port (P3) + tillsyn's Go adapter.

### Q1-Q5 hypothesis verdicts

| Hypothesis | Belief | Result |
|---|---|---|
| Q1: `--bare` can NOT pass MCPs/plugins | unproven | **REFUTED** — `--mcp-config` works under --bare for standard MCPs (T2 ollama smoke: agent reported `mcp__hylla__hylla_graph_list`); F14 2026-05-27 extends this to plugin MCPs too (Playwright registered + tool actually fired under --bare via translated `--mcp-config`) |
| Q2: `-p` WITHOUT `--bare` can ignore tools but NOT CLAUDE.md | claimed | **CONFIRMED** — T-C loaded tillsyn CLAUDE.md (66K input tokens; cited Hard Rule) |
| Q3: codex exec can ignore CLAUDE.md / project context | claimed | **CONFIRMED** — `--ignore-user-config` + `project_doc_max_bytes=0` + hermetic CODEX_HOME succeeded with no project context bleed in T1/T3 |
| Q4: codex exec CAN'T do per-file edit limitation | claimed | **CONFIRMED via spec** — `--sandbox workspace-write -C <dir>` is dir-scoped; no per-file flag exists |
| Q5: claude -p or built-in does edits via per-file `--allowedTools` | claimed | **VERIFIED mechanism** — T-E proved Bash(pwd:*) allow + Bash(git commit:*) deny; same mechanism applies to Edit(//abs/file) |

## 10. Folded from HYLLA_BIN.md (canonical reference, reconciled 2026-05-25)

`HYLLA_BIN.md` (hylla/polyglot-foundation, the single canonical reference; calls this doc "sand's
summary") adds these on top of §0–§9. Order: G-items = proven findings to capture; D-items = drift the
LIVE tillsyn dispatcher code already exhibits against the decision (read `cli_claude/argv.go` +
`cli_codex/argv.go` 2026-05-25).

- **G5 — tier-fallback signal.** On a tier failure the dispatcher emits a structured `CODEX_EXHAUSTED
  role=<role>` signal so the orch re-dispatches the chain's next tier. tillsyn makes this a configurable
  "run-what-you-can" flow (per-role chain ordering in TOML). Currently absent.
- **G6 — trace PERSISTENCE + auto-capture (not just return).** Every dispatch RETURNS the trace AND the
  dispatcher PERSISTS it to `.claude/agent-runs/<run>.{out,err,meta.json}` (gitignored) — HYLLA_BIN §4.
  Per the dev's directive, tillsyn ALSO auto-captures the trace ref into an action_item `metadata` field
  at dispatch-completion, **regardless of which channel/backend ran the agent**, so the veracity audit
  has a durable system-of-record handle (not just chat-window output).
- **G7 — SUPERSEDED 2026-05-27.** The `--bare` recipe is retired across all dispatch paths. See
  §12 for the no-bare architecture. The G7 finding (clean-context recipe via env vars + flags) is
  historical evidence of what the `--bare` path documented; the new direction is no-bare for parity
  with built-in Agent. The CLAUDE_CODE_DISABLE_* env vars + `--exclude-dynamic-system-prompt-sections`
  + `--strict-mcp-config` + `--disable-slash-commands` + the `outputStyle` settings JSON are all
  dropped from the dispatcher. See `EDIT_PATH_SCOPE_GATING_DEFERRED.md` for the deferred design
  around per-dispatch edit-path scope (the safety layer `--bare` couldn't provide either).
- **D1 — claude adapter is `-p`-only with NO gate (biggest refactor).** `cli_claude/argv.go` dispatches
  EVERY Anthropic role as `claude --bare … -p "" --permission-mode acceptEdits`. That is the API-key
  headless path (needs `ANTHROPIC_API_KEY`), and `acceptEdits` AUTO-ACCEPTS every edit → currently ZERO
  edit-scope + ZERO git-block on the claude side. The decision routes OAuth roles to the **built-in Agent
  tool** (no `-p` subprocess) gated by the `till gate` hook — a path that **does not exist in the adapter
  today**. The refactor splits the claude channel in two: built-in (OAuth, default) vs `-p --bare`
  (API-key/ollama config tier).
- **Closeout routing (canonical opus).** Closeout routes to **opus** across all projects, matching
  HYLLA_BIN §2. Earlier tillsyn delta documented haiku — REVERTED 2026-05-27 per dev directive.
  Closeout requires judgment (intent-match drift, follow-up triage, commit-message draft,
  documentation-sync flagging), not just mechanical aggregation; opus universal.
- **D2 — codex sandbox is currently INERT + no git floor.** `cli_codex/argv.go` hardcodes `--sandbox
  workspace-write` with NO `-c approval_policy="never"` (→ sandbox inert in exec; writes anywhere), NO
  hermetic `CODEX_HOME`, NO execpolicy `rules/default.rules`, NO `read-only` for planning/QA, and NO
  `project_doc_max_bytes=0` / `skills.bundled.enabled=false`. The codex git floor + write-confine are
  not built yet; this confirms + quantifies §7's refactor mandate.

## 12. No-bare architecture (LOCKED 2026-05-27)

`--bare` retired across all dispatch paths. Built-in Agent and `claude -p` both
load the same default context (CLAUDE.md, plugins, hooks, MCPs, skills) — parity
between the two channels is the new directive.

### Why no `--bare`

| Driver | Detail |
|---|---|
| OAuth incompatibility | `--bare` requires `ANTHROPIC_API_KEY` (OAuth + keychain are "never read" per claude --help; PL-La1 OAuth attempt empirically failed with `apiKeySource=none`). Forces every dispatch onto API-key billing. |
| Hooks stripped, no re-enable path | `--bare` strips hooks unconditionally. `--setting-sources project` re-loads settings but NOT hooks (PL-G3 empirical: a custom settings.json's PreToolUse Bash hook never fired). The git-block hook (`ta_action_gate.py`) is therefore unreachable on the `--bare` path. |
| Plugin MCPs only re-enableable via custom JSON | `--mcp-config` with a hand-crafted manifest can wire plugin MCPs (F14 verified Playwright), but the full plugin (skills + hooks + commands + sub-agents) does NOT come along. Layered re-construction is fragile. |
| Built-in Agent has no equivalent strip | Built-in Agent always inherits parent context. Using `--bare` on `-p` means `-p` has LESS context than built-in — breaks parity, forces persona authors to write two flavors. |

Net: accept the CLAUDE.md auto-load cost (~30K input tokens per dispatch on
`-p`; free at billing time on ollama; subscription-cost on OAuth) in exchange
for working hooks + plugin/MCP auto-load + per-channel parity.

### Canonical `-p` invocation

Both dispatch paths (ollama via `ANTHROPIC_BASE_URL` + OAuth via keychain)
use the same flag shape:

```
claude -p \
  --model <model> \
  --output-format stream-json \
  --verbose \
  --no-session-persistence \
  --settings <project>/.claude/agents/<persona>/settings.json \
  --append-system-prompt "${PERSONA_BODY}${ANTI_RECURSION}"
```

Ollama path adds env: `ANTHROPIC_BASE_URL=http://localhost:11434
ANTHROPIC_API_KEY=ollama` (ollama accepts any API key value).

OAuth path adds no env vars (OAuth auto-found by claude code via keychain
or `claude setup-token`-issued long-lived API key).

`--output-format stream-json` (with mandatory `--verbose`) emits the full
tool_use event stream — required by `bin/agent-audit-toon.py` for the per-
dispatch audit. The previous `--output-format json` only captured the final
result envelope + permission_denials (sufficient for safety check, lacked
tool stream for audit).

`--append-system-prompt` (not `--system-prompt`) is the parity choice: the
persona body is appended to claude code's default system prompt, matching
the (assumed) APPEND semantics of built-in Agent. If empirical evidence later
shows built-in REPLACES, switch to `--system-prompt`.

### Per-persona settings.json (the surface gate)

Layout: `<project>/.claude/agents/<persona>/settings.json` — subdir alongside
the flat `<persona>.md`. Claude code's agent auto-discovery scans for `*.md`
in `.claude/agents/`; dirs with the same name (modulo extension) coexist
without confusion.

Contents per persona — `permissions.allow` / `permissions.deny` patterns
declarative of the persona's role-appropriate tool surface (Bash patterns,
MCP tools, edit tools). Example pattern shapes:

- `Bash(mage:*)` — allow any mage invocation.
- `Bash(git commit:*)` — deny git commit.
- `mcp__ta__*` — allow all ta MCP tools.
- `Read` / `Edit` / `Write` / `MultiEdit` — bare tool name allow/deny.

### Hook-mediated enforcement — EMPIRICAL CORRECTION 2026-05-27

Smoke testing revealed that claude code's `-p` headless mode WITHOUT `--bare`
does NOT enforce `--settings <file>` `permissions.deny` rules, nor
`--allowedTools` / `--disallowedTools` flags. Only the natively-loaded
user (`~/.claude/settings.json`) + project (`<project>/.claude/settings.json`)
deny rules fire in headless. Smoke diagnostics:

| Mechanism | Enforced in `-p` no-bare? |
|---|---|
| User `~/.claude/settings.json` deny | YES (rm -rf blocked) |
| Project `.claude/settings.json` deny | YES (awk blocked) |
| `--settings <file>` permissions.deny | NO (git commit ran) |
| `--allowedTools "..."` restriction | NO (git commit ran) |
| `--disallowedTools "..."` flag | NO (git commit ran) |

So `--settings <persona-settings.json>` cannot be the per-persona enforcement
layer for `-p`. The hook (`ta_action_gate.py`) becomes the universal
enforcement layer for BOTH paths. The persona settings.json file is the
declarative source of truth; the hook is the enforcement engine.

**Discovery mechanism — how the hook knows which persona is dispatched**:

- **Built-in Agent subagent**: `agent_id` + `agent_type` are present in hook
  input (claude code passes them automatically for subagent tool calls).
  Hook reads `<project>/.claude/agents/<agent_type>/settings.json`.

- **`claude -p` subprocess** (dispatched via `bin/agent-dispatch.sh`): the
  dispatcher exports `TILL_PERSONA=<role>` env var for the subprocess.
  Subprocess's PreToolUse hook inherits the env. Hook reads
  `os.environ["TILL_PERSONA"]` and loads
  `<project>/.claude/agents/<TILL_PERSONA>/settings.json`.

- **Top-level orchestrator session**: neither `agent_id` nor `TILL_PERSONA`
  set. Hook defers to claude code's normal permission flow.

Same hook logic, same settings.json file, same enforcement decisions across
both paths. `--settings <persona-settings.json>` stays in the dispatcher
argv as INFORMATIONAL (future-proof + visible in claude code's init event)
but is not relied on for enforcement.

The dispatcher passes `--settings` argv AND `TILL_PERSONA` env in one
hardcoded path so the persona declaration is impossible to forget. Built-in
Agent dispatch doesn't pass either — claude code handles `agent_type`
natively.

### Hook architecture (`ta_action_gate.py`)

- **Top-level sessions** (orchestrator OR `claude -p` subprocess): no
  `agent_id` in hook input. DEFER to claude code's normal permission flow.
  For `-p`, that flow includes `--settings <persona-settings.json>` applied
  natively.

- **Built-in Agent subagents**: `agent_id` present. Hook reads
  `<project>/.claude/agents/<agent_type>/settings.json` and applies
  `permissions.deny` patterns to the current Bash command, PLUS hardcoded
  baselines:
    - Git mutation verbs (commit/push/add/etc.) — orchestrator is sole
      committer.
    - Raw go verbs (test/build/vet/run/install/fmt/mod/tool/generate/get/work)
      — must use mage.
    - `gofmt` / `gofumpt` — must use mage format.

- **Non-Bash tool surface** (Read / Edit / Write / mcp__*) is restricted by
  the persona's `tools:` frontmatter, which claude code enforces natively
  for built-in Agent before the hook fires.

### Edit-path scope (DEFERRED)

Per-dispatch edit-path scope (e.g. "this ta-fe-builder may only edit
`ui/foo.tsx`, not `ui/bar.tsx`") is NOT enforced today. The persona-level
allow/deny in settings.json declares Edit/Write/MultiEdit blanket-allowed for
builders and blanket-denied for QA — but does not slice per-dispatch.

Design considered in `EDIT_PATH_SCOPE_GATING_DEFERRED.md`. Pickup notes
included there. Solution shape: per-dispatch JSON state file
(`/tmp/till-dispatch-<TOKEN>.json` or future Tillsyn `action_item.paths`),
discovered via env var (`-p`) or transcript scan (built-in Agent). Deferred
because the no-bare + per-persona-settings architecture is the higher-
priority foundation.

### Migration

- `bin/agent-dispatch.sh dispatch_ollama` — `--bare` removed, redundant flags
  removed, G7 env vars removed, `--mcp-config` removed (auto-loads now),
  `--allowedTools/--disallowedTools` removed (moved to per-persona
  settings.json).
- `bin/agent-dispatch.sh dispatch_claude_native` — un-retired. OAuth `-p` path
  with same flag shape as `dispatch_ollama` minus ollama env vars.
- `.claude/hooks/ta_action_gate.py` — `<TA_ALLOWLIST>` parsing replaced with
  per-persona settings.json reading; env-var fallback retired (subprocesses
  now use `--settings` natively).
- Per-persona settings.json files: one per persona under
  `.claude/agents/<persona>/settings.json`. 13 in tillsyn, 13 in ta, 7 each
  in valv + sand (Go-only), 13 in hylla-poly. Byte-identical cp once tillsyn
  is proven.
