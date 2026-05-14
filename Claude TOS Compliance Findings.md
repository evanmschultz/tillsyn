# Anthropic Terms of Service Compliance — Findings and Plan

**Purpose:** Document our interpretation of Anthropic's terms as they apply to our product, the architectural choices we have made to stay on the right side of those terms, and the things we have explicitly decided not to do.

**Status:** Pre-launch planning document. Subject to revision as Anthropic's policies evolve and as we receive direct guidance from Anthropic.

---

## 1. The Rules That Apply to Us

Our product runs the official `claude` binary on the user's own machines, using the user's own Anthropic credentials. The relevant Anthropic policies are therefore:

### 1.1 Claude Code Legal and Compliance Page

Source: <https://code.claude.com/docs/en/legal-and-compliance>

The two clauses that matter most to us:

**Authentication and credential use.** The page draws a clear line between OAuth (subscription) and API key authentication:

> OAuth authentication is intended exclusively for purchasers of Claude Free, Pro, Max, Team, and Enterprise subscription plans and is designed to support ordinary use of Claude Code and other native Anthropic applications.
>
> Developers building products or services that interact with Claude's capabilities, including those using the Agent SDK, should use API key authentication through Claude Console or a supported cloud provider. Anthropic does not permit third-party developers to offer Claude.ai login or to route requests through Free, Pro, or Max plan credentials on behalf of their users.

**Acceptable use and "ordinary, individual usage":**

> Claude Code usage is subject to the Anthropic Usage Policy. Advertised usage limits for Pro and Max plans assume ordinary, individual usage of Claude Code and the Agent SDK.

**Upcoming change (June 15, 2026):**

> Starting June 15, 2026, Agent SDK and `claude -p` usage on subscription plans will draw from a new monthly Agent SDK credit, separate from your interactive usage limits.

### 1.2 Anthropic Usage Policy

Source: <https://www.anthropic.com/legal/aup>

### 1.3 Consumer Terms of Service

Source: <https://www.anthropic.com/legal/consumer-terms>

Section 3.7 has, since February 2024, restricted unauthorized automated access tools.

### 1.4 Commercial Terms

Source: <https://www.anthropic.com/legal/commercial-terms>

Applies when users authenticate via API key through Claude Console or a supported cloud provider.

### 1.5 Public Anthropic Enforcement Signals

Two public statements from Anthropic that informed our design:

- Anthropic engineer Thariq Shihipar (January 2026): "Third-party harnesses using Claude subscriptions create problems for users and are prohibited by our Terms of Service. They generate unusual traffic patterns without any of the usual telemetry that the Claude Code harness provides."
- The Register's reporting on the February 2026 ToS clarification (<https://www.theregister.com/2026/02/20/anthropic_clarifies_ban_third_party_claude_access/>): "Using OAuth tokens obtained through Claude Free, Pro, or Max accounts in any other product, tool, or service — including the Agent SDK — is not permitted and constitutes a violation of the Consumer Terms of Service."

The pattern these enforcement actions targeted: tools that extracted OAuth tokens from a user's local Claude Code installation and used those tokens in a third-party API client that impersonated the Claude Code harness.

---

## 2. How We Interpret These Rules

We read Anthropic's rules as having two underlying concerns, in order of how seriously they are enforced:

**Concern 1 — Who is the principal making the API call?** When a third party operates a service that authenticates with a user's Claude credentials and makes API calls on the user's behalf, that third party has materially become the principal even if the OAuth token nominally belongs to the user. This is what "route requests through Free, Pro, or Max plan credentials on behalf of their users" means. This is the bright line.

**Concern 2 — Does the traffic look like ordinary individual usage?** Subscription pricing assumes one human doing one human's worth of work. Architectures that fan out concurrent requests from a single subscription, or that drive Claude Code programmatically in ways the harness wasn't designed for, generate "unusual traffic patterns" even when the protocol details are technically correct.

Our compliance strategy is to be cleanly on the right side of Concern 1 (we are not the principal at any layer) and to architect deliberately around Concern 2 (our traffic shape per credential matches one human working).

---

## 3. What We Are Building

### 3.1 What the Product Is

A Kanban-style Agent Development Environment that helps a developer drive their chosen coding agents (Claude Code, Codex, etc.) more effectively across their projects. The product provides:

- A Kanban board for organizing work as tasks
- A managed terminal experience (built on tmux) for opening and managing real CLI agent sessions
- Inter-device sync so the developer can monitor and respond to their own sessions from desktop or mobile
- Document viewing, localhost preview, and other IDE-adjacent features
- Multi-agent support: the user picks which CLI agents they want to use

### 3.2 Architectural Commitments

These are the design decisions that keep us compliant. They are commitments, not aspirations.

**The `claude` binary runs only on the user's own machines.** Either their desktop or a VPS they themselves manage (rented in their own name, provisioned by them, paying the cloud provider directly). We never run `claude` on infrastructure we operate.

**The user authenticates Claude Code directly on their own machine.** OAuth tokens, API keys, and any other Claude credentials live on the user's machine. Our software never collects, stores, transmits, proxies, or has visibility into those credentials.

**All Anthropic API traffic is directly between the user's machine and Anthropic.** No Claude API request or response ever flows through our servers. From Anthropic's perspective, the user's machine is the sole originator of every call.

**The user picks how they pay Anthropic.** Our product supports three payment modes explicitly, and the user chooses:

1. **OAuth / subscription mode** — the user's Free, Pro, Max, Team, or Enterprise subscription via the standard Claude Code OAuth flow
2. **SDK credit mode** — `-p` / Agent SDK usage on subscription plans, which after June 15, 2026 draws from Anthropic's separate Agent SDK credit
3. **API key mode** — keys provisioned through Claude Console or a supported cloud provider (Bedrock, Vertex), operating under Commercial Terms

We do not auto-select, recommend based on cost, or otherwise nudge the user toward any particular mode. The user decides.

**One orchestrator agent per worktree, not many top-level instances per project.** A developer can have a small number of orchestrators running across a few worktrees in a project (a normal multi-branch development pattern), but each orchestrator is a single Claude Code session with subagents handled through Claude Code's built-in Task tool, not many parallel top-level `claude` processes against one subscription. This keeps the traffic shape per credential aligned with one human doing one human's worth of work.

**Configurable, conservative parallel-agent caps.** The product ships with low default caps on how many Task-tool subagents a given orchestrator will spawn concurrently. Users can raise the cap in their own config, but the default reflects ordinary individual usage.

**MCP and SQLite for agent coordination.** Subagents within an orchestrator coordinate through an MCP server and a local SQLite store on the user's machine. MCP is Anthropic's own protocol for this kind of integration. No coordination happens through our servers.

**Mobile app is a thin client to the user's own desktop or VPS.** The mobile app does not run `claude`, does not hold Claude credentials, and does not call Anthropic. It is a remote display and input device for the terminals and Kanban running on the user's own machine.

### 3.3 The Relay Layer (Sync Between User's Own Devices)

To make multi-device use feasible for users who do not want to set up Tailnet or similar, we operate a relay layer that lets the user's mobile app reach the user's own desktop or VPS over the internet. This relay is structurally similar to Signal, iMessage, Tailscale DERP, or any push-notification service. Specifically:

- **It carries only user-to-user-own-device traffic.** Terminal display streams, Kanban events, MCP messages, and notifications between the user's own devices.
- **Traffic is end-to-end encrypted with keys the relay cannot access.** Each user device generates a keypair, devices exchange public keys at pairing, and our relay routes opaque ciphertext.
- **Store-and-forward is short-lived and encrypted.** If the desktop is briefly unreachable, messages may queue at the relay, but they remain ciphertext we cannot read and are bounded to short retention windows.
- **No Claude credentials ever transit the relay.** OAuth tokens, API keys, and session credentials live on the user's machine; the mobile app sends instructions to "ask your local claude session to do X" rather than holding credentials itself.

The relay handles routing between the user's own systems and our own application's auth (account login, device pairing). It does not touch Anthropic.

### 3.4 Team Features

Team functionality, when added, will coordinate shared *work* — shared Kanban boards, shared task templates, presence, comments, project visibility — not shared *Claude access*. Each team member brings their own Claude credentials, runs agents on their own machines, and pays Anthropic directly under their own account.

---

## 4. What We Are Not Doing

These are explicit decisions, recorded so we hold the line if feature pressure ever pushes against them.

**We are not running `claude` on infrastructure we operate.** No hosted compute service, no "we run agents for you" tier, no shared cloud workers. If we ever offer a hosted compute option, the user will provision and own the underlying machine and we will function only as a connection convenience to that user-owned machine.

**We are not collecting, proxying, or storing the user's Anthropic credentials.** No OAuth token storage, no API key vault, no credential helper on our side. The user authenticates Claude Code directly on their own machine, the same way they would without our product.

**We are not routing Claude API requests through our servers.** Not transparently, not as a "performance optimization," not for caching, not for analytics. The user's machine is the only thing that ever talks to api.anthropic.com.

**We are not spoofing or replacing the Claude Code harness.** We run the actual `claude` binary in actual PTYs/terminals. We do not implement a custom Claude API client that masquerades as Claude Code. We do not use undocumented or hidden flags whose purpose is to bypass harness behavior or rate-limit accounting.

**We are not building cost-optimization logic that picks payment modes for the user.** The user chooses their payment mode. We display what we know about their stated configuration; we do not algorithmically route them toward whichever mode extracts the most value from their subscription.

**We are not offering "one shared Claude subscription for the team" features.** Each team member is responsible for their own Anthropic relationship.

**We are not fanning out many top-level `claude` processes against one credential to multiply throughput.** Concurrency happens through the Task tool inside a single orchestrator, which is the harness's own mechanism for it. The handful of orchestrators a user runs corresponds to the handful of worktrees a developer would normally have open.

**We are not encouraging or designing for usage patterns Anthropic has flagged.** When Anthropic introduces a new accounting mechanism (like the June 15, 2026 Agent SDK credit), we treat that as a signal about how they want the use case metered, and we do not architect to evade it.

---

## 5. Justification Summary

Why we believe this design is compliant:

**On Concern 1 (principal):** The user is unambiguously the principal at every layer where Anthropic is involved. They installed `claude`. They authenticated it. They own the machine it runs on. They pay Anthropic directly under their own account. Our product is a UI layer over their tools running on their machines using their credentials. The closest analogy is what Termius is to SSH, or what GitHub Mobile is to git — a better client for a tool the user already runs themselves.

**On Concern 2 (traffic shape):** A user running our product looks to Anthropic like one developer with a few worktrees open, each driven by a real Claude Code session with normal Task-tool subagent usage. This is structurally identical to a developer who keeps three terminal tabs open in iTerm with `claude` in each. Our default parallel-agent caps are conservative, and we do not architect to evade Anthropic's metering mechanisms.

**On the relay layer specifically:** Our relay carries only user-to-user-own-device traffic, end-to-end encrypted with keys we cannot access. It is structurally identical to consumer messaging relays and remote-access services that have not raised concerns for the underlying services those users access. Critically, Anthropic credentials and API traffic never touch it.

---

## 6. Open Items and Things We Will Watch

**Direct contact with Anthropic before commercial launch.** We will email Anthropic (via the contact-sales link on the Claude Code legal page, and/or developer relations) to describe our architecture and get their read on it. The architecture is novel enough that confirming our interpretation in writing is worth the time.

**The June 15, 2026 Agent SDK credit change.** We will monitor how this lands in practice. If Anthropic clarifies what counts as "ordinary individual usage" for orchestrator-with-subagents patterns, we will adjust defaults accordingly. The product already exposes payment-mode choice to the user, which positions us well for whatever they decide.

**Future ToS updates.** Anthropic has explicitly described their Usage Policy as a living document. We will treat any new clarification — especially around subscription limits, harness behavior, or third-party tooling — as a signal that may require architectural adjustment, and we will not assume that "fine today" means "fine forever."

**Documentation and transparency.** When we launch publicly, we will publish a clear architecture/security page describing exactly what we do and do not do, so users and Anthropic can both verify our claims. Tools that operate transparently are rarely the ones caught up in enforcement waves.

---

## 7. Source Index

- Claude Code Legal and Compliance: <https://code.claude.com/docs/en/legal-and-compliance>
- Anthropic Usage Policy: <https://www.anthropic.com/legal/aup>
- Consumer Terms of Service: <https://www.anthropic.com/legal/consumer-terms>
- Commercial Terms: <https://www.anthropic.com/legal/commercial-terms>
- Claude Code documentation hub: <https://code.claude.com/docs/>
- Logging in to your Claude account: <https://support.claude.com/en/articles/13189465-logging-in-to-your-claude-account>
- Use the Claude Agent SDK with your Claude plan (June 15 change): <https://support.claude.com/en/articles/15036540-use-the-claude-agent-sdk-with-your-claude-plan>
- Anthropic Trust Center: <https://trust.anthropic.com>
- The Register on the February 2026 ToS clarification: <https://www.theregister.com/2026/02/20/anthropic_clarifies_ban_third_party_claude_access/>
- Anthropic contact (sales / policy questions): <https://www.anthropic.com/contact-sales>
- Anthropic support: <https://support.claude.com>
