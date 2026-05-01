# Tillsyn — Task Manager + Project Tracker + Team Collaboration

A task management app with multiple UI surfaces backed by a single local daemon. Native TUI, Electron desktop app, Capacitor mobile app — all sharing one SQLite store on each desktop, syncing across devices and team members through Tailscale. Maintained by Hylla, Inc. **Apache 2.0 open source. Local-first, user-owned: each user has their own Tailscale account, no user data on Hylla servers.** Free for everyone; small per-user fee for mobile sync; small one-time-per-team fee for team collaboration.

---

## What Tillsyn Is

Tillsyn covers what GitHub Issues + Projects (the kanban) does for a repository, but local-first and offline-capable:

- **Issue tracking** — bugs, feature requests, todos
- **Project management** — kanban boards, milestones, lists
- **Progress tracking** — status, completion, time estimates
- **Team communication** — @mentions to assign and notify, threaded comments on tasks

All four wrapped into one tool with one data model, syncing via Tailscale.

**Teams and maintainers dictate their own templates and usage patterns.** Tillsyn doesn't impose a workflow — it provides the primitives (tasks, lists, statuses, fields, mentions). A team running Scrum models sprints and stories; a team running Kanban models columns and WIP limits; a solo user might just have a flat task list. Templates are user-defined and shareable.

## Pricing Model

Tillsyn is **free at the user level**. Anyone can install Tillsyn, use it locally forever, and join any team they're invited to without paying anything. The fee structure is deliberately designed to make adoption frictionless — especially for OSS projects.

| What | Who pays | Why |
|------|----------|-----|
| **Tillsyn (local use)** | Nobody | The whole product is free locally, by design |
| **Mobile sync** | The individual user, for their own devices | They get the value of mobile access to their own machines |
| **Team creation** | The team creator, once per team | They get persistent multi-contributor coordination value |
| **Team participation** | Nobody | Members joining someone else's team never pay |

This is the only model that makes Tillsyn viable as OSS infrastructure. A maintainer pays one team fee for their project. Hundreds of contributors join free, contribute, leave when done. The contributor who shows up for one weekend hackathon never sees a paywall. Charging contributors would kill the entire dynamic — like GitHub charging contributors to push to someone else's repo.

### What This Means in Practice

**Solo user:** installs Tillsyn, uses it locally on one machine. Free forever.

**Solo user with multiple devices:** pays the mobile fee once, gets sync between their own devices (laptop, home server, phone) over their own Tailscale.

**OSS maintainer:** pays one team fee for their project. Contributors install Tillsyn (free), get invited (free), participate (free), leave when done.

**FAANG engineer on three teams:** pays nothing personally. Their company pays for the work teams. The OSS project they contribute to has the maintainer's team fee. Their personal use is free.

**Team-by-team, not org-wide:** there's no "Tillsyn Enterprise" license. A 10,000-employee company has hundreds of teams, each licensed separately by whoever creates it. This naturally maps to how teams form (projects, working groups, squads) rather than imposing a billing structure on top.

### License Types

Two distinct license types, both Ed25519-signed and offline-verifiable:

- **Mobile license** — bound to a user's Tailscale identity (`"subject": "user:alice@example.com"`). Unlocks mobile sync for that user across their own devices.
- **Team license** — bound to a team ID (`"subject": "team:xyz123"`). Unlocks team mesh sync for that team. Members verify it without having any license of their own.

Token payload examples:

```json
{
  "id": "lic_abc",
  "kind": "mobile",
  "subject": "user:alice@example.com",
  "exp": 1893456000
}
```

```json
{
  "id": "lic_def",
  "kind": "team",
  "subject": "team:proj-xyz",
  "exp": 1893456000,
  "issued_to": "stripe_customer_alice"
}
```

### How Members Verify Team License Without Paying

When a member's daemon connects to a team, it receives the team's signed license token as part of the CRDT bootstrap data. The member's daemon:

1. Verifies the signature against the embedded Hylla public key (offline, no Hylla call)
2. Confirms `kind: "team"` and `subject` matches the team ID it's syncing
3. Checks `exp` is in the future
4. Optionally checks the weekly-polled revocation list
5. Unlocks team mesh sync for *that team only*

The member has no personal license. If they leave the team or the team's license expires, their daemon stops syncing that team's CRDT but their personal/local Tillsyn keeps working.

### License Lifecycle

**Activation flow:**

1. User creates a team locally in Tillsyn (free, exists in their local CRDT, no sync yet)
2. Clicks "Activate team" → opens Stripe Checkout with team ID as metadata
3. After payment, server issues team license bound to that team ID
4. Tillsyn picks it up via deep link (or email magic link), embeds in the team's CRDT
5. Team becomes synced; members can be invited

**If a team license lapses:** team CRDT data still exists locally on each member's machine. New changes don't propagate. The team creator gets a renewal nudge. They renew, sync resumes, no data lost.

**Team ownership transfer:** the billing party can change without rotating the license. Admin operation in Tillsyn → notifies Hylla → Stripe customer changes for that team's subscription → same license ID, same team ID, just new billing contact. Members notice nothing.

**OSS sponsorship pattern:** for community-funded projects, the maintainer-of-record pays Stripe; the project's GitHub Sponsors / OpenCollective / corporate sponsor reimburses them. Tillsyn doesn't need to know about this — it's just normal billing on the maintainer's side.

## Surfaces

- **TUI** — Go terminal UI for power users
- **Electron desktop app** — wraps the daemon's UI with native shell integration (system tray, native menus, notifications)
- **Capacitor mobile app** — iOS + Android, ships its own SQLite + tsnet
- **MCP server** — optional facet of the daemon, lets AI agents (Claude, Cursor, etc.) interact with Tillsyn over stdio

(WebUI deferred to v2 — see "Why no WebUI in v1" below.)

All desktop UIs share one SQLite store per machine. Mobile has its own SQLite that syncs with desktop daemons over tailnet.

## Architecture: Persistent Daemon + Thin Clients

**One Tillsyn daemon per desktop, runs as a background service, multiple thin UIs.** Required by the "idiot-proof mobile sync" goal — see rationale below.

```
Desktop machine:
  tillsyn-daemon (Go, persistent background process, auto-starts on login)
    ├── owns SQLite storage (~/.local/share/tillsyn/ or platform equivalent)
    ├── HTTP+WebSocket API on 127.0.0.1:PORT (with auth token)
    ├── runs tsnet for tailnet sync (one identity per machine)
    ├── handles CRDT sync logic (personal hub-and-spoke + per-team meshes)
    └── system tray icon (clickable for status, open UI, quit)
  
  tillsyn-tui     → talks to daemon via localhost HTTP/WS
  tillsyn-electron → wraps the daemon's UI; bundles + manages daemon
  tillsyn-mcp (separate invocation) → thin stdio↔HTTP bridge to daemon

Phone:
  tillsyn-mobile (Capacitor)
    ├── own SQLite
    ├── embedded tsnet
    └── syncs with desktop daemon(s) over tailnet
```

### Why a Persistent Daemon (Not Leader-Election Among UIs)

The "idiot-proof" goal forces this. Mobile sync requires the home machine to be reachable over tailnet *whenever* the phone wants to sync — including at 7am while the user is asleep with no UI open. Something has to be running.

Alternatives considered:

- **Leader election among UIs** — first UI to launch owns SQLite + tsnet, others detect and connect, ownership transfers on exit. Works in principle but has nasty failure modes: ungraceful crashes leave stale lock files, race conditions during simultaneous launches, handoff during active sync drops messages. And critically, when the user closes all UIs, mobile sync dies.
- **Each UI runs its own everything** — multiple devices on the user's tailnet per machine, internal sync conflicts, generally a mess.

A persistent daemon is the only architecture that's actually idiot-proof. It can be nearly invisible: auto-start on login (launchd / systemd user units / Task Scheduler), ~15 MB RAM, system tray icon for awareness.

**One-endpoint UX preserved:** the user types `tillsyn` and the binary auto-starts the daemon if it's not running, registers it for auto-start, then proceeds. They never have to think about launching a separate process.

### Why HTTP on Localhost (Not Stdio or Unix Sockets)

Mobile sync over tailnet is HTTP/WebSocket — that's a hard constraint, since stdio is process-local and can't traverse a network. Given that, the choice is whether local UIs use the same HTTP or a different protocol (Unix sockets, stdio).

Same protocol everywhere wins:

- One API surface to maintain, one set of tests
- TUI/Electron use the same code path mobile uses
- Easy to debug with curl
- 127.0.0.1 binding means no network exposure
- Token auth: daemon writes a generated token to `~/.config/tillsyn/token` (mode 0600); clients read it; other processes on the system can't snoop without filesystem access already at that level
- Performance is irrelevant for a task manager

### MCP via Stdio Still Works

When an AI agent invokes `tillsyn-mcp`, it spawns a short-lived process that:

1. Reads the auth token from `~/.config/tillsyn/token`
2. Connects to the running daemon's HTTP API on localhost
3. Translates MCP tool calls to HTTP requests, responses to MCP messages
4. Exits when the agent closes stdio

User experience: stdio for MCP (single endpoint, no port management), without it being the *primary* protocol. The daemon is the source of truth; MCP is a frontend like the others.

**MCP is free across all use cases** — it's just an alternate access pattern to the same local data. No license required.

### Why No WebUI in v1

Considered and dropped. Reasons:

- **Hylla-hosted WebUI is incompatible with the trust model** — browser loading JS from Hylla while interacting with user-private data is the wrong shape, and browsers' CORS policies block `webui.hylla.com` from reaching `home.tailnet.ts.net` anyway.
- **Daemon-hosted WebUI** works at localhost but cross-device requires system-wide Tailscale (not the embedded-tsnet model). Limits the audience.
- **Native UIs cover every legitimate use case** — TUI for power users / SSH'd-in workflows, Electron for desktop, Capacitor for mobile.

WebUI may return in v2 if user demand emerges. The HTTP architecture leaves the door open at zero cost.

### Daemon as Home Agent

The daemon **is** the home agent. Same role as Ro's home agent: tsnet node, sync hub, owns the canonical state. No separate "home agent" binary for Tillsyn — the daemon serves both local UIs and remote tailnet peers (the user's other devices, plus team members).

### Electron Specifics

Electron wraps the UI and provides:

- Native window, system tray, notifications, native menus
- Daemon lifecycle management (start daemon if not running, stays in tray on close)
- Bundles the daemon binary alongside the Electron app
- Loads the UI from `http://127.0.0.1:PORT` of the bundled daemon

## Stack

- **Daemon:** Go, SQLite, embedded HTTP/WS server, tsnet, optional MCP server mode
- **TUI:** Go, HTTP client to daemon
- **UI codebase:** Astro + Solid (TypeScript) — one codebase for Electron and Capacitor mobile
- **Capacitor plugin:** Swift (iOS) + Kotlin (Android) wrapping tsnet
- **Tunnel (mobile):** tsnet (Go), compiled via gomobile
  - iOS: `.xcframework`
  - Android: `.aar`
- **Sync state:** CRDT (Yjs or Automerge), event log + snapshots
- **License:** Ed25519 signed tokens, offline-verifiable, per-user (mobile) or per-team

## Repo Structure

`tillsyn` monorepo:

```
tillsyn/                           (Apache 2.0, public, owned by Hylla, Inc.)
├── daemon/              Go: SQLite owner + sync hub + tsnet + HTTP/WS + MCP mode
├── core/                Shared Go: data model, CRDT schemas, protocol defs
├── tui/                 Go TUI (HTTP client to daemon)
├── ui/                  Astro + Solid (the UI codebase, used by Electron + mobile)
├── electron/            Electron wrapper; bundles daemon + UI
├── mobile/              Capacitor app: UI + Capacitor + tsnet plugin
├── plugin/              Capacitor plugin (Swift + Kotlin) wrapping tsnet
└── docs/
```

Related repos:

- `license-server` — shared with Ro, separate Apache 2.0 repo (see license-server.md)
- `ro` — sibling product (WebKit browser), separate repo

## Distribution & Licensing

- **License:** Apache 2.0
- **Source:** all client and daemon code public
- **Mobile sync + Team mesh sync:** gated by signed license tokens in *official* builds (build flag stubs out the check in source builds)
- **Build-from-source:** anyone can compile and use any feature without paying — by design
- **Pay-for-convenience:** the fee covers official signed/notarized binaries, App Store / Play Store distribution, ongoing maintenance

Aseprite / Ardour model — open code, paid binaries.

## Tailscale Model: User Owns Everything

**The user is Tailscale's customer, not Hylla.** They have their own Tailscale account (free Personal plan covers most use cases — up to 6 users, unlimited devices).

Tillsyn is an **Integration** in Tailscale's terminology — explicitly defined and permitted in their ToS Section 1.5. Hylla charges users for *official Tillsyn builds*; Tailscale provides the network layer for free (or whatever the user pays Tailscale directly). This keeps Tillsyn firmly on the right side of ToS Section 2.3's reseller prohibition.

### OAuth Scopes Required

For personal sync:

- `devices:core` (read/write) — managing the user's own devices
- `auth_keys` (read/write) — minting auth keys for new devices

For teams (programmatic share invitation):

- Same scopes, *plus* the device-share API endpoint
- **Caveat:** there's a known OAuth limitation — `GET /v2/tailnet/-/devices` returns devices owned by the tailnet but not devices shared *into* the tailnet from other users (Tailscale issue #16911). This affects management API enumeration but not actual connectivity. Tillsyn discovers team peers via tsnet's local peer enumeration, not via the management API, so this works around it.
- **To verify during implementation:** confirm the device-share endpoint accepts OAuth client credentials (not just user-scoped tokens). If it requires user-scoped auth, the team-creation flow may need a one-time interactive Tailscale auth step (the user clicks a link, authorizes Tillsyn to share devices on their behalf, returns to the app). Manageable but worth knowing up front.

### Onboarding (Same as Ro)

1. Premium upgrade screen explains Tailscale requirement
2. User signs into Tailscale or creates an account (Google/Apple/GitHub/Microsoft SSO via Tailscale's login)
3. User generates OAuth client credential, pastes into Tillsyn
4. Tillsyn verifies, displays "✓ Connected to your tailnet"

## Two-Tier Credential Model

Same as Ro. Only the user's primary device holds the long-lived OAuth client credential. Other devices hold only their own tsnet identity, gained via combined-QR pairing.

## Adding Devices

**Combined-QR pairing (recommended):** primary device mints a single-use Tailscale auth key, encodes it together with the user's mobile license token in one QR. New device scans, joins tailnet, stores license, persists tsnet state. ~10 seconds.

**Manual paste (fallback):** for first device or remote machines that can't be physically scanned.

## Personal Sync Architecture (Hub-and-Spoke)

For the user's own devices, all daemons sync through a designated primary daemon — typically the home server or always-on desktop. Mobile is always a spoke. Hub-and-spoke for v1, with the data model forward-compatible for full mesh in a later version.

(Full sync implementation details — CRDT representation, WebSocket protocol, event log, snapshots, catch-up — same as Ro, see ro-project.md for the deep dive.)

## Teams Architecture

Tillsyn teams enable multiple users — each on their own Tailscale account — to share specific datasets while keeping personal data private. **Hylla never sees team data.** **Joining a team is free for the joiner** — only the team creator pays.

### Tailscale Sharing as the Cross-User Primitive

Tailscale supports surgical cross-tailnet access: User A can share a specific machine (node) with User B from another tailnet. The shared node appears in B's tailnet at A's chosen name; B sees nothing else of A's. Sharing respects ACLs of both tailnets. Tags, groups, subnet routing all stripped — only the specific node is reachable.

**Quarantine by default:** shared machines can receive incoming connections from the recipient's tailnet but cannot initiate outgoing connections to it. WebSocket connections handle this fine because connections are bidirectional once established by either side. Mutual sharing (the normal team setup) gives full bidirectional sync.

**Cost:** each unique cross-tailnet share increases device limits by 2 on both accounts. Sharing has zero cost on free Personal plan and actually expands both users' headroom.

### Small Teams (≤ ~10 members) — Full Mesh

Each member shares their daemon with each other member. N² shares scales fine for small teams.

```
Alice creates team in Tillsyn → activates with team license (pays Stripe)
  → generates team ID (UUID)
  → invites Bob: invite contains team ID, Alice's tailnet name, one-time CRDT auth token
  → Tillsyn calls Tailscale API to share Alice's daemon with bob@email

Bob receives invite (free for him)
  → Tillsyn opens Tailscale share-accept flow
  → Bob accepts the share
  → Bob's Tillsyn auto-shares Bob's daemon back to Alice
  → both daemons now have bidirectional reachability
  → team's CRDT bootstrap data flows to Bob, including the team license token
  → Bob's daemon verifies the team license, unlocks team sync for this team

Adding Carol: same flow. Each existing member shares with Carol; Carol shares back.
```

### Larger Teams / Open Source Projects — Designated Team Host

Mesh sharing breaks down beyond ~15-20 people. Two patterns scale further:

**Designated team host (hub-and-spoke at team level):** one member runs a "team-host" daemon — could be the project maintainer's existing server, or a small VPS the team chips in for. All members share their daemons with the host; the host's daemon is shared with all members. Members sync through the host. **Hylla is still not in this picture.** This is the natural pattern for OSS projects: maintainer already has a server, contributors install Tillsyn and connect.

**Read-only contributors:** for projects where most contributors only need to *see* tasks (and a few core people write), only core members need mutual sharing. Read-only contributors get a one-way share of the team-host. Their daemons receive updates but don't push.

### Data Model

- Each user's daemon stores both *personal* and *team* data in its SQLite, namespaced by team ID
- Team data is a separate CRDT document per team
- The team license token lives in the team's CRDT root, replicated to all members
- CRDT operations carry team scope; sync filters apply by team membership
- Each user can be on multiple teams; their daemon syncs each team's data with that team's members
- Personal data stays personal — never crosses team boundaries
- Leaving a team: member's daemon stops syncing that team's CRDT but retains a local snapshot (read-only after leave) unless the user explicitly deletes it

### Membership Management

The team's member list is itself part of the CRDT — replicated across all members' daemons. Adding/removing members is an operation that propagates. Bootstrap: an existing member's daemon includes the current member list in the snapshot it sends a new member during catch-up. Permissions for membership changes are policy-driven (founder / admins can add/remove; regular members cannot).

### Cross-Tailnet Authorization

When a remote daemon connects, Tailscale's local API on the receiving daemon reports the peer's verified identity (their Tailscale account email/SSO). The receiving daemon checks: "is this identity on the team's member list?" If yes, allow sync for that team's CRDT. If no, reject. Zero Trust at the network layer (Tailscale verifies identity), authorization at the data layer (Tillsyn enforces membership).

### Templates and Workflow Customization

**Teams and maintainers dictate their own templates and usage patterns.** Tillsyn provides primitives — tasks, lists, statuses, custom fields, mentions, comments, attachments. It does not impose Scrum vs. Kanban vs. flat-list. Templates are:

- User-defined (any member can create)
- Shareable within a team (CRDT-replicated)
- Optionally exportable as files for cross-team or open distribution
- Versioned and forkable

A solo user runs whatever template they want. A team running Scrum models sprints, stories, points. A team running Kanban models columns and WIP. An OSS project might use templates like "bug report," "feature request," "RFC." Hylla ships a few default templates as starting points; everything else is user space.

### Issue Tracking + Project Kanban + Communication

Tillsyn covers what GitHub Issues + GitHub Projects (the kanban) does, but local-first:

- **Issues** are tasks with status, labels, assignees, comments
- **Projects/kanban** are views over tasks — boards with columns, lists, milestones
- **Tasks can link** to issues, PRs, commits if integrated with a code host (future feature, optional)
- **Assignment** uses @mentions; the assigned user gets a notification entry in their Tillsyn (and optionally APNs/FCM push if mobile is set up)
- **Comments** are threaded per-task, with @mentions for back-and-forth
- **All four — issue tracking, project management, progress tracking, team communication — in one tool with one data model.** No separate apps, no separate sync, no separate access control.

### Team Ownership and Billing Transfer

The team creator is the billing party. They can transfer this:

1. In Tillsyn, current owner: Settings → Team → Transfer billing → enter new owner's email
2. New owner accepts via email link (Stripe-mediated payment method change)
3. Hylla updates the team's Stripe subscription to the new customer
4. Same license ID, same team ID, just new billing contact
5. Members notice nothing

Useful when a maintainer steps away from an OSS project, when a team moves between corporate budget owners, or when paying responsibility shifts.

## Network Model (Client)

Two paths. tsnet is per-app, not a system VPN.

1. **Sync traffic** (personal + team CRDT updates) → tsnet → primary daemon (personal) / mesh peers or team host (team) → fan-out
2. **Hylla license server** — periodic license verification + revocation poll. The only requests touching Hylla infrastructure. Daily/weekly cadence; no real-time chatter.

No external API calls in normal Tillsyn operation. No telemetry beyond opt-in anonymized crash reports.

### DERP fallback

Tailscale uses DERP relay servers when direct WireGuard fails (~5–15% of sessions). Traffic remains E2E encrypted. The user's tailnet uses Tailscale's DERP unless they self-host.

## MCP Integration (AI Agent Tooling)

When run with `--mcp`, the daemon exposes itself as an MCP (Model Context Protocol) server over stdio. Or, more commonly, `tillsyn-mcp` is a separate short-lived binary spawned by the AI agent that bridges stdio to the daemon's localhost HTTP API.

Standard MCP tool definitions: `add_task`, `list_tasks`, `complete_task`, `update_task`, `assign_task`, `comment_on_task`, `query`, etc.

**MCP is free** — same as local Tillsyn use. It accesses whatever data the user already has access to (personal data always; team data for teams they're members of).

## Mobile Background Behavior

Both iOS and Android suspend Tillsyn and its embedded tsnet node when backgrounded. Primary → phone pushes don't work in that state. For "open app, see synced state" this is fine. For background notifications use APNs (iOS) / FCM (Android) as a wake signal, then reconnect.

## Build Notes

- gomobile compile twice: `gomobile bind -target=ios` → `.xcframework`, `gomobile bind -target=android` → `.aar`
- Capacitor plugin scaffolding: `npm init @capacitor/plugin`
- Plugin's Swift class wraps the `.xcframework`; Kotlin class wraps the `.aar`
- Both expose the same JS-facing API so the Solid code is platform-agnostic
- Daemon binary built per-OS (Linux, macOS, Windows initially; BSDs later)
- Electron bundles the daemon binary as a sidecar
- UI codebase serves Electron + Capacitor via different build configs
- Embed the license-verification public key at build time
- Source-buildable variant has the license check stubbed out (build flag)

## ToS-Aligned Messaging

Transparency principle is also legal protection. Suggested copy:

- "Powered by Tailscale" — fine, descriptive use of the wordmark
- "Requires a free Tailscale account" — accurate and unambiguous
- ❌ "Includes Tailscale" — reads as bundled resale, avoid
- ❌ "Built-in private networking" without naming Tailscale — opaque, conflicts with transparency *and* muddies the integration relationship

## License Server Integration

License server is a separate repo (`license-server`, shared with Ro). See `license-server.md` for full details. Tillsyn-specific notes:

- Tillsyn embeds the license-server public key at build time (same key Ro uses)
- Two distinct license kinds for Tillsyn: `mobile` (per-user) and `team` (per-team)
- Team license token lives in the team's CRDT root; replicated to all members; verified offline
- Stripe Checkout flows differ: mobile checkout collects user identity at activation; team checkout collects team ID at activation
- Magic-link emails use Tillsyn branding

## Enterprise Use (No Special Handling Required)

The same architecture and pricing model scales to enterprises with no fundamental changes:

- Enterprises have their own Tailscale Enterprise contract (or Headscale, etc.). Hylla doesn't pay anything to Tailscale for them.
- Within a single corporate tailnet, cross-tailnet *sharing* isn't even needed — every employee's daemon is directly reachable subject to corporate ACLs.
- Cross-org collaboration (contractor working with the enterprise, two companies partnering) uses normal Tailscale sharing.
- Teams are billed individually by whoever creates them. A 10,000-employee company has hundreds of teams, each licensed separately. No "Tillsyn Enterprise" tier needed — same product, scaled by team count.
- Volume pricing for orgs paying for many teams may emerge as a billing wrapper later (e.g., 50% off after N teams in the same Stripe customer account), but the architecture doesn't change.

What an enterprise gets by default:

- Their Tailscale ACLs enforce who can see whom — Tillsyn rides on top
- Identity is whatever they use in Tailscale (typically corporate SSO via Okta / Entra ID / Workspace)
- SCIM provisioning of Tailscale users handles employee onboarding/offboarding — when an employee leaves, their devices vacate the tailnet, and Tillsyn loses access naturally
- Audit trails come from Tailscale's logging plus the team's CRDT log

What Hylla doesn't have to do for any enterprise:

- Host their data
- Manage their network
- Be a SOC 2 audit subject for *their* data (Hylla doesn't have it)
- Build SSO integration for *Tillsyn* (their SSO is Tailscale's problem)
- Compliance work for HIPAA / GDPR / etc. on user data (Hylla doesn't process it)

## One-Line Architecture

TUI / Electron / Capacitor mobile → all share one persistent Tillsyn daemon per desktop (HTTP+WS on localhost; auto-starts on login) → daemon owns SQLite, runs tsnet, optional MCP stdio bridge → primary daemon acts as CRDT sync hub for personal devices (hub-and-spoke v1, mesh later); team CRDTs sync via Tailscale node-sharing across team members' tailnets (mesh for small teams, designated team host for larger / OSS). Hylla license server issues per-user mobile licenses and per-team team licenses (Ed25519 signed); verified offline. Apache 2.0 open source; pay only for official builds; team participation is always free.
