package dispatcher

import (
	"context"

	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

// monitorServiceAdapter binds the dispatcher's narrow monitor interface (which
// uses the package-local updateActionItemInput) to *app.Service's wider
// UpdateActionItemInput. The adapter is the one-line forward-declared shape
// referenced by monitor.go's package overview: the dispatcher package cannot
// import internal/app's UpdateActionItemInput without breaking the monitor
// stub-injection contract (the test stub wants a tiny shape; production
// supplies the wider one). Translating at this seam keeps both worlds happy.
//
// The adapter only translates the methods the cleanup hook + crash-handling
// pipeline actually invoke: GetActionItem, ListColumns, MoveActionItem,
// UpdateActionItem. Other Service methods consumed elsewhere in the dispatcher
// (walker, conflict detector) are accessed through their own narrow
// interfaces, which *app.Service satisfies directly without translation.
type monitorServiceAdapter struct {
	svc *app.Service
}

func (a monitorServiceAdapter) GetActionItem(ctx context.Context, actionItemID string) (domain.ActionItem, error) {
	return a.svc.GetActionItem(ctx, actionItemID)
}

func (a monitorServiceAdapter) ListColumns(ctx context.Context, projectID string, includeArchived bool) ([]domain.Column, error) {
	return a.svc.ListColumns(ctx, projectID, includeArchived)
}

func (a monitorServiceAdapter) MoveActionItem(ctx context.Context, actionItemID, toColumnID string, position int) (domain.ActionItem, error) {
	return a.svc.MoveActionItem(ctx, actionItemID, toColumnID, position)
}

// UpdateActionItem translates the dispatcher's narrow updateActionItemInput
// into app.UpdateActionItemInput. Today only Metadata flows through; future
// monitor extensions either widen this translation or extend the local
// shape so the adapter stays the single point of contact.
func (a monitorServiceAdapter) UpdateActionItem(ctx context.Context, in updateActionItemInput) (domain.ActionItem, error) {
	wide := app.UpdateActionItemInput{
		ActionItemID: in.ActionItemID,
		Metadata:     in.Metadata,
		UpdatedType:  domain.ActorTypeSystem,
	}
	return a.svc.UpdateActionItem(ctx, wide)
}
