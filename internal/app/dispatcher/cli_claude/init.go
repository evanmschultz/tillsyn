package cli_claude

import (
	"github.com/evanmschultz/tillsyn/internal/app/dispatcher"
)

// init registers the claude adapter with the dispatcher's CLIKind→adapter
// registry at package import time. The dispatcher cannot import cli_claude
// directly because cli_claude imports dispatcher (for BindingResolved /
// BundlePaths / CLIAdapter type definitions); a registration init() inverts
// the dependency direction so the spawn-side registry stays cycle-free.
//
// Production binaries (cmd/till) trigger this init via a side-effect
// import:
//
//	import _ "github.com/evanmschultz/tillsyn/internal/app/dispatcher/cli_claude"
//
// Tests that exercise BuildSpawnCommand against the real claude adapter do
// the same. Test code that prefers a mock adapter calls
// dispatcher.RegisterAdapter directly with its own CLIAdapter implementation
// and does not need to import this package at all.
//
// Drop 4d adds an analogous init() in cli_codex; cmd/till adds one more
// blank import.
func init() {
	dispatcher.RegisterAdapter(dispatcher.CLIKindClaude, New())
}
