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
| closeout | claude-native | haiku | — | built-in Agent tool + gate hook |

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
Reproduced ≥2x on macOS by ta / hylla / tillsyn / sand; commands + on-disk artifacts in the superseded
docs (git history). Vendor sources: Claude Code docs (`permissions` `//abs` + deny>allow; `hooks`
exec-form + no-bundled-node/python; `headless` `--bare`-needs-API-key); codex docs (`exec-policy`
`prefix_rule(forbidden)`; `concepts/sandboxing` writable-roots + `<writable_root>/.git` ro;
`config-reference` `approval_policy`/`[windows] sandbox`); codex issues #16732 (hooks), #24098 (win init).

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
- **G7 — explicit `claude -p` clean-context recipe (HYLLA_SANDBOX_IDEA finding 17).** The tillsyn
  user-config `-p` path emits the documented set, not a vague "`--bare` does it": env
  `CLAUDE_CODE_DISABLE_CLAUDE_MDS` / `_AUTO_MEMORY` / `_GIT_INSTRUCTIONS` + flags
  `--exclude-dynamic-system-prompt-sections` + `--strict-mcp-config` + `--disable-slash-commands` +
  `outputStyle:"default"`.
- **D1 — claude adapter is `-p`-only with NO gate (biggest refactor).** `cli_claude/argv.go` dispatches
  EVERY Anthropic role as `claude --bare … -p "" --permission-mode acceptEdits`. That is the API-key
  headless path (needs `ANTHROPIC_API_KEY`), and `acceptEdits` AUTO-ACCEPTS every edit → currently ZERO
  edit-scope + ZERO git-block on the claude side. The decision routes OAuth roles to the **built-in Agent
  tool** (no `-p` subprocess) gated by the `till gate` hook — a path that **does not exist in the adapter
  today**. The refactor splits the claude channel in two: built-in (OAuth, default) vs `-p --bare`
  (API-key/ollama config tier).
- **Closeout routing override:** tillsyn routes `closeout` to **haiku**, NOT HYLLA_BIN §2's `opus`. Rationale: closeout is mechanical aggregation + commit-message draft (commit-role tier), and the role has never been exercised (0 closeout nodes in-project) — opus is overkill. §6 above reflects haiku; HYLLA_BIN §2 (shared cross-project doc) is left at opus and this is the documented tillsyn delta.
- **D2 — codex sandbox is currently INERT + no git floor.** `cli_codex/argv.go` hardcodes `--sandbox
  workspace-write` with NO `-c approval_policy="never"` (→ sandbox inert in exec; writes anywhere), NO
  hermetic `CODEX_HOME`, NO execpolicy `rules/default.rules`, NO `read-only` for planning/QA, and NO
  `project_doc_max_bytes=0` / `skills.bundled.enabled=false`. The codex git floor + write-confine are
  not built yet; this confirms + quantifies §7's refactor mandate.
