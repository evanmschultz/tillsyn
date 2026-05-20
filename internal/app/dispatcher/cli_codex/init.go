package cli_codex

import "github.com/evanmschultz/tillsyn/internal/app/dispatcher"

// init registers the codex adapter with the dispatcher's CLIKind→adapter
// registry at package import time. The dispatcher cannot import cli_codex
// directly because cli_codex imports dispatcher (for BindingResolved /
// BundlePaths / CLIAdapter type definitions); a registration init() inverts
// the dependency direction so the spawn-side registry stays cycle-free.
//
// Production binaries (cmd/till) trigger this init via a side-effect
// import:
//
//	import _ "github.com/evanmschultz/tillsyn/internal/app/dispatcher/cli_codex"
//
// Tests that exercise BuildSpawnCommand against the real codex adapter do
// the same. Test code that prefers a mock adapter calls
// dispatcher.RegisterAdapter directly with its own CLIAdapter implementation
// and does not need to import this package at all.
//
// Drop 4d D3 adds this file alongside the analogous
// internal/app/dispatcher/cli_claude/init.go so cmd/till's two blank imports
// wire both adapters at process start — CLIKindClaude and CLIKindCodex are
// both available in the registry from the first instruction after main().
//
// Note: cli_codex has no render sub-package (codex does not need the
// permission-handshake settings.json rendering that cli_claude/render
// provides), so there is no additional blank import here.
func init() {
	dispatcher.RegisterAdapter(dispatcher.CLIKindCodex, New())
}
