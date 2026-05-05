package render

import (
	"github.com/evanmschultz/tillsyn/internal/app/dispatcher"
)

// init registers Render with the dispatcher's bundle-render hook seam at
// package import time. The dispatcher cannot import this package directly
// because render imports dispatcher (for Bundle / BindingResolved /
// domain types); a registration init() inverts the dependency direction
// so the spawn-side seam stays cycle-free.
//
// Production binaries (cmd/till) trigger this init via a side-effect
// import:
//
//	import _ "github.com/evanmschultz/tillsyn/internal/app/dispatcher/cli_claude/render"
//
// Tests that exercise BuildSpawnCommand against the real bundle render
// do the same. Test code that prefers a fake render hook calls
// dispatcher.RegisterBundleRenderFunc directly with its own function and
// does not need to import this package at all (last-writer-wins on the
// register call lets a test substitute the hook for fault injection).
func init() {
	dispatcher.RegisterBundleRenderFunc(Render)
}
