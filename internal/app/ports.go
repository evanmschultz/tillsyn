package app

import (
	"context"
	"time"

	"github.com/hylla/tillsyn/internal/domain"
)

// Repository represents repository data used by this package.
type Repository interface {
	CreateProject(context.Context, domain.Project) error
	UpdateProject(context.Context, domain.Project) error
	DeleteProject(context.Context, string) error
	GetProject(context.Context, string) (domain.Project, error)
	ListProjects(context.Context, bool) ([]domain.Project, error)
	SetProjectAllowedKinds(context.Context, string, []domain.KindID) error
	ListProjectAllowedKinds(context.Context, string) ([]domain.KindID, error)

	CreateKindDefinition(context.Context, domain.KindDefinition) error
	UpdateKindDefinition(context.Context, domain.KindDefinition) error
	GetKindDefinition(context.Context, domain.KindID) (domain.KindDefinition, error)
	ListKindDefinitions(context.Context, bool) ([]domain.KindDefinition, error)

	CreateColumn(context.Context, domain.Column) error
	UpdateColumn(context.Context, domain.Column) error
	ListColumns(context.Context, string, bool) ([]domain.Column, error)

	CreateTask(context.Context, domain.Task) error
	UpdateTask(context.Context, domain.Task) error
	GetTask(context.Context, string) (domain.Task, error)
	ListTasks(context.Context, string, bool) ([]domain.Task, error)
	DeleteTask(context.Context, string) error
	CreateComment(context.Context, domain.Comment) error
	ListCommentsByTarget(context.Context, domain.CommentTarget) ([]domain.Comment, error)
	ListProjectChangeEvents(context.Context, string, int) ([]domain.ChangeEvent, error)
	CreateAttentionItem(context.Context, domain.AttentionItem) error
	GetAttentionItem(context.Context, string) (domain.AttentionItem, error)
	ListAttentionItems(context.Context, domain.AttentionListFilter) ([]domain.AttentionItem, error)
	ResolveAttentionItem(context.Context, string, string, domain.ActorType, time.Time) (domain.AttentionItem, error)
	CreateAuthRequest(context.Context, domain.AuthRequest) error
	GetAuthRequest(context.Context, string) (domain.AuthRequest, error)
	ListAuthRequests(context.Context, domain.AuthRequestListFilter) ([]domain.AuthRequest, error)
	UpdateAuthRequest(context.Context, domain.AuthRequest) error

	CreateCapabilityLease(context.Context, domain.CapabilityLease) error
	UpdateCapabilityLease(context.Context, domain.CapabilityLease) error
	GetCapabilityLease(context.Context, string) (domain.CapabilityLease, error)
	ListCapabilityLeasesByScope(context.Context, string, domain.CapabilityScopeType, string) ([]domain.CapabilityLease, error)
	RevokeCapabilityLeasesByScope(context.Context, string, domain.CapabilityScopeType, string, time.Time, string) error
}

// HandoffRepository represents optional durable handoff storage used by this package.
type HandoffRepository interface {
	CreateHandoff(context.Context, domain.Handoff) error
	GetHandoff(context.Context, string) (domain.Handoff, error)
	ListHandoffs(context.Context, domain.HandoffListFilter) ([]domain.Handoff, error)
	UpdateHandoff(context.Context, domain.Handoff) error
}
