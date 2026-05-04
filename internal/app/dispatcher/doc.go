// Package dispatcher implements the cascade dispatcher loop that replaces the
// orchestrator-as-dispatcher prototype with a programmatic state-trigger
// dispatcher composed of a LiveWaitBroker subscriber, file/package lock
// managers, a tree walker, an agent spawner, a sibling-overlap conflict
// detector, a process monitor, and a terminal-state cleanup hook.
//
// Wave 2 of Drop 4a delivers the manual-trigger milestone: this package is
// fired from the till dispatcher run CLI rather than auto-running inside the
// server process. Continuous-mode wiring (Start/Stop loops, post-build gates,
// commit-and-push, drop-end Hylla reingest) lands in Drop 4b.
//
// Package layout (one Wave-2 droplet per file, all in this package so a
// single compile-lock serializes the build):
//
//   - dispatcher.go    — Dispatcher interface, dispatcher impl, NewDispatcher,
//     DispatchOutcome, Result enum, Options, sentinel errors. (Wave 2.1)
//   - broker_sub.go    — LiveWaitBroker subscription + event filter. (Wave 2.2)
//   - locks_file.go    — File-level paths lock manager. (Wave 2.3)
//   - locks_package.go — Package-level packages lock manager. (Wave 2.4)
//   - walker.go        — Tree walker + auto-promotion eligibility. (Wave 2.5)
//   - spawn.go         — Template-binding lookup + agent-spawn invocation. (Wave 2.6)
//   - conflict.go      — Sibling overlap → runtime blocked_by insertion. (Wave 2.7)
//   - monitor.go       — Process tracking + crash detection. (Wave 2.8)
//   - cleanup.go       — Terminal-state cleanup. (Wave 2.9)
//
// The Options struct is intentionally an open struct: later Wave 2 droplets
// (2.6, 2.8, 2.10) and Drop 4b will add fields without breaking existing
// callers. Today Options is empty; expand fields rather than introduce a new
// constructor variant.
//
// YAGNI watch: this package does NOT contain gate-runner, commit-agent, push,
// or reingest fields. Those land in Drop 4b. If a Wave 2 attack flags missing
// post-build gating, surface and reject — the manual-trigger milestone is
// scoped to spawn + monitor + cleanup only.
package dispatcher
