package render

import (
	"context"

	"github.com/evanmschultz/tillsyn/internal/app/dispatcher"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

// init registers a thin adapter around Render with the dispatcher's
// bundle-render hook seam at package import time. The dispatcher cannot
// import this package directly because render imports dispatcher (for
// Bundle / BindingResolved / domain types); a registration init() inverts
// the dependency direction so the spawn-side seam stays cycle-free.
//
// The adapter handles the `any` ↔ PermissionGrantsLister conversion at
// the seam: BundleRenderFunc accepts `any` because dispatcher must not
// import render to name the interface, and the adapter does a typed
// assertion (with the documented nil-tolerance) before calling Render.
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
	dispatcher.RegisterBundleRenderFunc(adaptRender)
}

// adaptRender bridges dispatcher.BundleRenderFunc's `any` lister
// parameter to the concrete render.PermissionGrantsLister Render
// expects. nil → nil; anything else must satisfy the interface or
// the adapter returns a typed error so callers see a clean failure
// rather than a downstream panic.
//
// The signature MUST stay byte-for-byte compatible with
// dispatcher.BundleRenderFunc; changing one without the other would
// break the registration line above at compile time.
func adaptRender(
	ctx context.Context,
	bundle dispatcher.Bundle,
	item domain.ActionItem,
	project domain.Project,
	binding dispatcher.BindingResolved,
	grantsLister any,
) (string, error) {
	var lister PermissionGrantsLister
	if grantsLister != nil {
		typed, ok := grantsLister.(PermissionGrantsLister)
		if !ok {
			return "", ErrInvalidGrantsLister
		}
		lister = typed
	}
	return Render(ctx, bundle, item, project, binding, lister)
}
