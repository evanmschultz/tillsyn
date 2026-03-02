package domain

import "time"

// ChangeOperation describes a persisted activity operation for a work item.
type ChangeOperation string

// ChangeOperation values used by the local activity ledger.
const (
	ChangeOperationCreate  ChangeOperation = "create"
	ChangeOperationUpdate  ChangeOperation = "update"
	ChangeOperationMove    ChangeOperation = "move"
	ChangeOperationArchive ChangeOperation = "archive"
	ChangeOperationRestore ChangeOperation = "restore"
	ChangeOperationDelete  ChangeOperation = "delete"
)

// ChangeEvent represents a single activity-log entry for a project work item.
type ChangeEvent struct {
	ID         int64
	ProjectID  string
	WorkItemID string
	Operation  ChangeOperation
	ActorID    string
	ActorName  string
	ActorType  ActorType
	Metadata   map[string]string
	OccurredAt time.Time
}

// DependencyRollup summarizes dependency and blocked-state counts for a project.
type DependencyRollup struct {
	ProjectID                 string
	TotalItems                int
	ItemsWithDependencies     int
	DependencyEdges           int
	BlockedItems              int
	BlockedByEdges            int
	UnresolvedDependencyEdges int
}
