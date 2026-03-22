package tui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/hylla/tillsyn/internal/app"
	"github.com/hylla/tillsyn/internal/domain"
)

// fakeService represents fake service data used by this package.
type fakeService struct {
	projects                []domain.Project
	columns                 map[string][]domain.Column
	tasks                   map[string][]domain.Task
	lastSearchFilter        app.SearchTasksFilter
	lastCreateTask          app.CreateTaskInput
	createTaskCalls         int
	comments                map[string][]domain.Comment
	lastCreateComment       app.CreateCommentInput
	authRequests            map[string]domain.AuthRequest
	authSessions            []app.AuthSession
	lastAuthRequestFilter   domain.AuthRequestListFilter
	lastAuthSessionFilter   app.AuthSessionFilter
	lastApproveAuthRequest  app.ApproveAuthRequestInput
	lastDenyAuthRequest     app.DenyAuthRequestInput
	lastRevokeAuthSessionID string
	lastRevokeAuthReason    string
	err                     error
	rollups                 map[string]domain.DependencyRollup
	changeEvents            map[string][]domain.ChangeEvent
	changeEventsErr         error
	attentionErrByProject   map[string]error
	attentionItemsByProject map[string][]domain.AttentionItem
	commentCreateErr        error
	commentListErr          error
	commentSeq              int
}

// newFakeService constructs fake service.
func newFakeService(projects []domain.Project, columns []domain.Column, tasks []domain.Task) *fakeService {
	colByProject := map[string][]domain.Column{}
	for _, c := range columns {
		colByProject[c.ProjectID] = append(colByProject[c.ProjectID], c)
	}
	taskByProject := map[string][]domain.Task{}
	for _, t := range tasks {
		taskByProject[t.ProjectID] = append(taskByProject[t.ProjectID], t)
	}
	return &fakeService{
		projects:                projects,
		columns:                 colByProject,
		tasks:                   taskByProject,
		comments:                map[string][]domain.Comment{},
		authRequests:            map[string]domain.AuthRequest{},
		authSessions:            []app.AuthSession{},
		rollups:                 map[string]domain.DependencyRollup{},
		changeEvents:            map[string][]domain.ChangeEvent{},
		attentionErrByProject:   map[string]error{},
		attentionItemsByProject: map[string][]domain.AttentionItem{},
	}
}

// ListProjects lists projects.
func (f *fakeService) ListProjects(context.Context, bool) ([]domain.Project, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := make([]domain.Project, len(f.projects))
	copy(out, f.projects)
	return out, nil
}

// ListColumns lists columns.
func (f *fakeService) ListColumns(_ context.Context, projectID string, includeArchived bool) ([]domain.Column, error) {
	if f.err != nil {
		return nil, f.err
	}
	cols := f.columns[projectID]
	out := make([]domain.Column, 0, len(cols))
	for _, c := range cols {
		if !includeArchived && c.ArchivedAt != nil {
			continue
		}
		out = append(out, c)
	}
	return out, nil
}

// ListTasks lists tasks.
func (f *fakeService) ListTasks(_ context.Context, projectID string, includeArchived bool) ([]domain.Task, error) {
	if f.err != nil {
		return nil, f.err
	}
	tasks := f.tasks[projectID]
	out := make([]domain.Task, 0, len(tasks))
	for _, task := range tasks {
		if !includeArchived && task.ArchivedAt != nil {
			continue
		}
		out = append(out, task)
	}
	return out, nil
}

// CreateComment creates one ownership-attributed comment.
func (f *fakeService) CreateComment(_ context.Context, in app.CreateCommentInput) (domain.Comment, error) {
	if f.commentCreateErr != nil {
		return domain.Comment{}, f.commentCreateErr
	}
	f.lastCreateComment = in
	f.commentSeq++
	comment, err := domain.NewComment(domain.CommentInput{
		ID:           fmt.Sprintf("cm-%d", f.commentSeq),
		ProjectID:    in.ProjectID,
		TargetType:   in.TargetType,
		TargetID:     in.TargetID,
		BodyMarkdown: in.BodyMarkdown,
		ActorID:      in.ActorID,
		ActorName:    in.ActorName,
		ActorType:    in.ActorType,
	}, time.Now().UTC())
	if err != nil {
		return domain.Comment{}, err
	}
	key := commentThreadKey(comment.ProjectID, comment.TargetType, comment.TargetID)
	f.comments[key] = append(f.comments[key], comment)
	return comment, nil
}

// ListCommentsByTarget lists comments for one concrete comment target.
func (f *fakeService) ListCommentsByTarget(_ context.Context, in app.ListCommentsByTargetInput) ([]domain.Comment, error) {
	if f.commentListErr != nil {
		return nil, f.commentListErr
	}
	key := commentThreadKey(in.ProjectID, in.TargetType, in.TargetID)
	out := append([]domain.Comment(nil), f.comments[key]...)
	return out, nil
}

// ListProjectChangeEvents lists persisted activity entries.
func (f *fakeService) ListProjectChangeEvents(_ context.Context, projectID string, limit int) ([]domain.ChangeEvent, error) {
	if f.changeEventsErr != nil {
		return nil, f.changeEventsErr
	}
	events := append([]domain.ChangeEvent(nil), f.changeEvents[projectID]...)
	if limit > 0 && len(events) > limit {
		events = events[:limit]
	}
	return events, nil
}

// ListAttentionItems returns fake attention rows derived from blocked tasks.
func (f *fakeService) ListAttentionItems(_ context.Context, in app.ListAttentionItemsInput) ([]domain.AttentionItem, error) {
	if f.err != nil {
		return nil, f.err
	}
	projectID := strings.TrimSpace(in.Level.ProjectID)
	if projectID == "" {
		return nil, domain.ErrInvalidID
	}
	if err := f.attentionErrByProject[projectID]; err != nil {
		return nil, err
	}
	if rows, ok := f.attentionItemsByProject[projectID]; ok {
		out := make([]domain.AttentionItem, 0, len(rows))
		for _, item := range rows {
			if req, ok := f.authRequests[strings.TrimSpace(item.ID)]; ok && req.State != domain.AuthRequestStatePending {
				continue
			}
			out = append(out, item)
		}
		if in.Limit > 0 && len(out) > in.Limit {
			out = out[:in.Limit]
		}
		return out, nil
	}
	tasks := f.tasks[projectID]
	out := make([]domain.AttentionItem, 0, len(tasks))
	for idx, task := range tasks {
		blockedBy := uniqueTrimmed(task.Metadata.BlockedBy)
		dependsOn := uniqueTrimmed(task.Metadata.DependsOn)
		if len(blockedBy) == 0 && len(dependsOn) == 0 && strings.TrimSpace(task.Metadata.BlockedReason) == "" {
			continue
		}
		summary := strings.TrimSpace(task.Metadata.BlockedReason)
		if summary == "" {
			summary = fmt.Sprintf("blocked task %s", task.Title)
		}
		out = append(out, domain.AttentionItem{
			ID:                 fmt.Sprintf("att-%s-%d", task.ID, idx),
			ProjectID:          projectID,
			ScopeType:          domain.ScopeLevelTask,
			ScopeID:            task.ID,
			State:              domain.AttentionStateOpen,
			Kind:               domain.AttentionKindBlocker,
			Summary:            summary,
			RequiresUserAction: true,
			CreatedAt:          time.Now().UTC(),
		})
	}
	if in.Limit > 0 && len(out) > in.Limit {
		out = out[:in.Limit]
	}
	return out, nil
}

// ListAuthRequests returns fake auth requests filtered by project/state in stable creation order.
func (f *fakeService) ListAuthRequests(_ context.Context, filter domain.AuthRequestListFilter) ([]domain.AuthRequest, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.lastAuthRequestFilter = filter
	out := make([]domain.AuthRequest, 0, len(f.authRequests))
	for _, request := range f.authRequests {
		if projectID := strings.TrimSpace(filter.ProjectID); projectID != "" && strings.TrimSpace(request.ProjectID) != projectID {
			continue
		}
		if state := domain.NormalizeAuthRequestState(filter.State); state != "" && domain.NormalizeAuthRequestState(request.State) != state {
			continue
		}
		out = append(out, request)
	}
	slices.SortFunc(out, func(a, b domain.AuthRequest) int {
		if !a.CreatedAt.Equal(b.CreatedAt) {
			if a.CreatedAt.Before(b.CreatedAt) {
				return -1
			}
			return 1
		}
		return strings.Compare(a.ID, b.ID)
	})
	if filter.Limit > 0 && len(out) > filter.Limit {
		out = out[:filter.Limit]
	}
	return out, nil
}

// GetAuthRequest returns one auth request by id.
func (f *fakeService) GetAuthRequest(_ context.Context, requestID string) (domain.AuthRequest, error) {
	request, ok := f.authRequests[strings.TrimSpace(requestID)]
	if !ok {
		return domain.AuthRequest{}, domain.ErrInvalidID
	}
	return request, nil
}

// ApproveAuthRequest approves one fake auth request and records the request payload.
func (f *fakeService) ApproveAuthRequest(_ context.Context, in app.ApproveAuthRequestInput) (app.ApprovedAuthRequestResult, error) {
	requestID := strings.TrimSpace(in.RequestID)
	request, ok := f.authRequests[requestID]
	if !ok {
		return app.ApprovedAuthRequestResult{}, domain.ErrInvalidID
	}
	f.lastApproveAuthRequest = in
	request.State = domain.AuthRequestStateApproved
	request.ResolvedByActor = strings.TrimSpace(in.ResolvedBy)
	request.ResolvedByType = in.ResolvedType
	request.ResolutionNote = strings.TrimSpace(in.ResolutionNote)
	f.authRequests[requestID] = request
	sessionTTL := in.SessionTTL
	if sessionTTL <= 0 {
		sessionTTL = time.Hour
	}
	f.authSessions = append(f.authSessions, app.AuthSession{
		SessionID:     "session-" + requestID,
		ProjectID:     request.ProjectID,
		AuthRequestID: request.ID,
		ApprovedPath:  firstNonEmptyTrimmed(strings.TrimSpace(in.Path), request.Path),
		PrincipalID:   request.PrincipalID,
		PrincipalType: request.PrincipalType,
		PrincipalName: firstNonEmptyTrimmed(request.PrincipalName, request.PrincipalID),
		ClientID:      request.ClientID,
		ClientType:    request.ClientType,
		ClientName:    firstNonEmptyTrimmed(request.ClientName, request.ClientID),
		IssuedAt:      time.Now().UTC(),
		ExpiresAt:     time.Now().UTC().Add(sessionTTL),
	})
	return app.ApprovedAuthRequestResult{
		Request:       request,
		SessionSecret: "session-secret",
	}, nil
}

// DenyAuthRequest denies one fake auth request and records the request payload.
func (f *fakeService) DenyAuthRequest(_ context.Context, in app.DenyAuthRequestInput) (domain.AuthRequest, error) {
	requestID := strings.TrimSpace(in.RequestID)
	request, ok := f.authRequests[requestID]
	if !ok {
		return domain.AuthRequest{}, domain.ErrInvalidID
	}
	f.lastDenyAuthRequest = in
	request.State = domain.AuthRequestStateDenied
	request.ResolvedByActor = strings.TrimSpace(in.ResolvedBy)
	request.ResolvedByType = in.ResolvedType
	request.ResolutionNote = strings.TrimSpace(in.ResolutionNote)
	f.authRequests[requestID] = request
	return request, nil
}

// ListAuthSessions returns fake session inventory filtered by project/client/principal/state.
func (f *fakeService) ListAuthSessions(_ context.Context, filter app.AuthSessionFilter) ([]app.AuthSession, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.lastAuthSessionFilter = filter
	now := time.Now().UTC()
	out := make([]app.AuthSession, 0, len(f.authSessions))
	for _, session := range f.authSessions {
		if sessionID := strings.TrimSpace(filter.SessionID); sessionID != "" && strings.TrimSpace(session.SessionID) != sessionID {
			continue
		}
		if projectID := strings.TrimSpace(filter.ProjectID); projectID != "" && strings.TrimSpace(session.ProjectID) != projectID {
			continue
		}
		if principalID := strings.TrimSpace(filter.PrincipalID); principalID != "" && strings.TrimSpace(session.PrincipalID) != principalID {
			continue
		}
		if clientID := strings.TrimSpace(filter.ClientID); clientID != "" && strings.TrimSpace(session.ClientID) != clientID {
			continue
		}
		switch strings.TrimSpace(strings.ToLower(filter.State)) {
		case "", "all":
		case "active":
			if session.RevokedAt != nil || !session.ExpiresAt.After(now) {
				continue
			}
		case "revoked":
			if session.RevokedAt == nil {
				continue
			}
		case "expired":
			if session.RevokedAt != nil || session.ExpiresAt.After(now) {
				continue
			}
		default:
			continue
		}
		out = append(out, session)
	}
	slices.SortFunc(out, func(a, b app.AuthSession) int {
		if !a.ExpiresAt.Equal(b.ExpiresAt) {
			if a.ExpiresAt.Before(b.ExpiresAt) {
				return -1
			}
			return 1
		}
		return strings.Compare(a.SessionID, b.SessionID)
	})
	if filter.Limit > 0 && len(out) > filter.Limit {
		out = out[:filter.Limit]
	}
	return out, nil
}

// RevokeAuthSession marks one fake session revoked with a stable reason.
func (f *fakeService) RevokeAuthSession(_ context.Context, sessionID string, reason string) (app.AuthSession, error) {
	sessionID = strings.TrimSpace(sessionID)
	for idx := range f.authSessions {
		if strings.TrimSpace(f.authSessions[idx].SessionID) != sessionID {
			continue
		}
		now := time.Now().UTC()
		f.authSessions[idx].RevokedAt = &now
		f.authSessions[idx].RevocationReason = strings.TrimSpace(reason)
		f.lastRevokeAuthSessionID = sessionID
		f.lastRevokeAuthReason = strings.TrimSpace(reason)
		return f.authSessions[idx], nil
	}
	return app.AuthSession{}, domain.ErrInvalidID
}

// GetProjectDependencyRollup returns project dependency rollup totals.
func (f *fakeService) GetProjectDependencyRollup(_ context.Context, projectID string) (domain.DependencyRollup, error) {
	if f.err != nil {
		return domain.DependencyRollup{}, f.err
	}
	if rollup, ok := f.rollups[projectID]; ok {
		return rollup, nil
	}
	tasks := f.tasks[projectID]
	rollup := domain.DependencyRollup{
		ProjectID:  projectID,
		TotalItems: len(tasks),
	}
	stateByID := map[string]domain.LifecycleState{}
	for _, task := range tasks {
		stateByID[task.ID] = task.LifecycleState
	}
	for _, task := range tasks {
		dependsOn := uniqueTrimmed(task.Metadata.DependsOn)
		blockedBy := uniqueTrimmed(task.Metadata.BlockedBy)
		if len(dependsOn) > 0 {
			rollup.ItemsWithDependencies++
			rollup.DependencyEdges += len(dependsOn)
		}
		if len(blockedBy) > 0 || strings.TrimSpace(task.Metadata.BlockedReason) != "" {
			rollup.BlockedItems++
		}
		rollup.BlockedByEdges += len(blockedBy)
		for _, depID := range dependsOn {
			state, ok := stateByID[depID]
			if !ok || state != domain.StateDone {
				rollup.UnresolvedDependencyEdges++
			}
		}
	}
	return rollup, nil
}

// SearchTaskMatches handles search task matches.
func (f *fakeService) SearchTaskMatches(ctx context.Context, in app.SearchTasksFilter) ([]app.TaskMatch, error) {
	f.lastSearchFilter = in
	f.lastSearchFilter.States = append([]string(nil), in.States...)
	f.lastSearchFilter.Levels = append([]string(nil), in.Levels...)
	f.lastSearchFilter.Kinds = append([]string(nil), in.Kinds...)
	f.lastSearchFilter.LabelsAny = append([]string(nil), in.LabelsAny...)
	f.lastSearchFilter.LabelsAll = append([]string(nil), in.LabelsAll...)
	query := strings.ToLower(strings.TrimSpace(in.Query))
	stateSet := map[string]struct{}{}
	for _, state := range in.States {
		state = strings.ToLower(strings.TrimSpace(state))
		if state == "" {
			continue
		}
		stateSet[state] = struct{}{}
	}
	allowAllStates := len(stateSet) == 0
	out := make([]app.TaskMatch, 0)

	projectIDs := make([]string, 0)
	if in.CrossProject {
		for _, p := range f.projects {
			if !in.IncludeArchived && p.ArchivedAt != nil {
				continue
			}
			projectIDs = append(projectIDs, p.ID)
		}
	} else {
		projectIDs = append(projectIDs, in.ProjectID)
	}

	for _, projectID := range projectIDs {
		project, ok := f.projectByID(projectID)
		if !ok {
			continue
		}
		for _, task := range f.tasks[projectID] {
			stateID := "todo"
			columnName := ""
			for _, c := range f.columns[projectID] {
				if c.ID == task.ColumnID {
					columnName = strings.ToLower(strings.ReplaceAll(c.Name, " ", "-"))
					break
				}
			}
			if columnName != "" {
				switch columnName {
				case "to-do", "todo":
					stateID = "todo"
				case "in-progress", "progress", "doing":
					stateID = "progress"
				default:
					stateID = columnName
				}
			}
			if task.ArchivedAt != nil {
				if !in.IncludeArchived {
					continue
				}
				stateID = "archived"
			}
			if !allowAllStates {
				if _, ok := stateSet[stateID]; !ok {
					continue
				}
			}
			if query != "" {
				matched := strings.Contains(strings.ToLower(task.Title), query) || strings.Contains(strings.ToLower(task.Description), query)
				if !matched {
					for _, label := range task.Labels {
						if strings.Contains(strings.ToLower(label), query) {
							matched = true
							break
						}
					}
				}
				if !matched {
					continue
				}
			}
			out = append(out, app.TaskMatch{
				Project: project,
				Task:    task,
				StateID: stateID,
			})
		}
	}
	return out, nil
}

// CreateProjectWithMetadata creates project with metadata.
func (f *fakeService) CreateProjectWithMetadata(_ context.Context, in app.CreateProjectInput) (domain.Project, error) {
	project, err := domain.NewProject("p-new", in.Name, in.Description, time.Now().UTC())
	if err != nil {
		return domain.Project{}, err
	}
	if err := project.UpdateDetails(project.Name, project.Description, in.Metadata, time.Now().UTC()); err != nil {
		return domain.Project{}, err
	}
	f.projects = append(f.projects, project)
	if _, ok := f.columns[project.ID]; !ok {
		now := time.Now().UTC()
		c1, _ := domain.NewColumn("c-new-1", project.ID, "To Do", 0, 0, now)
		c2, _ := domain.NewColumn("c-new-2", project.ID, "In Progress", 1, 0, now)
		c3, _ := domain.NewColumn("c-new-3", project.ID, "Done", 2, 0, now)
		f.columns[project.ID] = []domain.Column{c1, c2, c3}
	}
	if _, ok := f.tasks[project.ID]; !ok {
		f.tasks[project.ID] = []domain.Task{}
	}
	return project, nil
}

// UpdateProject updates state for the requested operation.
func (f *fakeService) UpdateProject(_ context.Context, in app.UpdateProjectInput) (domain.Project, error) {
	for idx := range f.projects {
		if f.projects[idx].ID != in.ProjectID {
			continue
		}
		if err := f.projects[idx].UpdateDetails(in.Name, in.Description, in.Metadata, time.Now().UTC()); err != nil {
			return domain.Project{}, err
		}
		return f.projects[idx], nil
	}
	return domain.Project{}, app.ErrNotFound
}

// ArchiveProject archives one project.
func (f *fakeService) ArchiveProject(_ context.Context, projectID string) (domain.Project, error) {
	for idx := range f.projects {
		if f.projects[idx].ID != projectID {
			continue
		}
		f.projects[idx].Archive(time.Now().UTC())
		return f.projects[idx], nil
	}
	return domain.Project{}, app.ErrNotFound
}

// RestoreProject restores one project.
func (f *fakeService) RestoreProject(_ context.Context, projectID string) (domain.Project, error) {
	for idx := range f.projects {
		if f.projects[idx].ID != projectID {
			continue
		}
		f.projects[idx].Restore(time.Now().UTC())
		return f.projects[idx], nil
	}
	return domain.Project{}, app.ErrNotFound
}

// DeleteProject hard-deletes one project and associated in-memory fixtures.
func (f *fakeService) DeleteProject(_ context.Context, projectID string) error {
	next := make([]domain.Project, 0, len(f.projects))
	found := false
	for _, project := range f.projects {
		if project.ID == projectID {
			found = true
			continue
		}
		next = append(next, project)
	}
	if !found {
		return app.ErrNotFound
	}
	f.projects = next
	delete(f.columns, projectID)
	delete(f.tasks, projectID)
	return nil
}

// CreateTask creates task.
func (f *fakeService) CreateTask(_ context.Context, in app.CreateTaskInput) (domain.Task, error) {
	f.lastCreateTask = in
	f.createTaskCalls++
	pos := 0
	for _, t := range f.tasks[in.ProjectID] {
		if t.ColumnID == in.ColumnID && t.Position >= pos {
			pos = t.Position + 1
		}
	}
	task, err := domain.NewTask(domain.TaskInput{
		ID:          "t-new",
		ProjectID:   in.ProjectID,
		ParentID:    in.ParentID,
		Kind:        in.Kind,
		Scope:       in.Scope,
		ColumnID:    in.ColumnID,
		Position:    pos,
		Title:       in.Title,
		Description: in.Description,
		Priority:    in.Priority,
		DueAt:       in.DueAt,
		Labels:      in.Labels,
		Metadata:    in.Metadata,
	}, time.Now().UTC())
	if err != nil {
		return domain.Task{}, err
	}
	f.tasks[in.ProjectID] = append(f.tasks[in.ProjectID], task)
	return task, nil
}

// UpdateTask updates state for the requested operation.
func (f *fakeService) UpdateTask(_ context.Context, in app.UpdateTaskInput) (domain.Task, error) {
	for projectID := range f.tasks {
		for idx := range f.tasks[projectID] {
			if f.tasks[projectID][idx].ID != in.TaskID {
				continue
			}
			f.tasks[projectID][idx].Title = strings.TrimSpace(in.Title)
			f.tasks[projectID][idx].Description = strings.TrimSpace(in.Description)
			f.tasks[projectID][idx].Priority = in.Priority
			f.tasks[projectID][idx].DueAt = in.DueAt
			f.tasks[projectID][idx].Labels = in.Labels
			if in.Metadata != nil {
				f.tasks[projectID][idx].Metadata = *in.Metadata
			}
			return f.tasks[projectID][idx], nil
		}
	}
	return domain.Task{}, app.ErrNotFound
}

// MoveTask moves task.
func (f *fakeService) MoveTask(_ context.Context, taskID, toColumnID string, position int) (domain.Task, error) {
	for projectID := range f.tasks {
		for idx := range f.tasks[projectID] {
			if f.tasks[projectID][idx].ID == taskID {
				f.tasks[projectID][idx].ColumnID = toColumnID
				f.tasks[projectID][idx].Position = position
				for _, column := range f.columns[projectID] {
					if column.ID != toColumnID {
						continue
					}
					switch strings.ToLower(strings.TrimSpace(strings.ReplaceAll(column.Name, " ", "-"))) {
					case "to-do", "todo":
						f.tasks[projectID][idx].LifecycleState = domain.StateTodo
					case "in-progress", "progress", "doing":
						f.tasks[projectID][idx].LifecycleState = domain.StateProgress
					case "done", "complete", "completed":
						f.tasks[projectID][idx].LifecycleState = domain.StateDone
					case "archived", "archive":
						f.tasks[projectID][idx].LifecycleState = domain.StateArchived
					default:
						// Keep the prior state when the column name doesn't map to canonical lifecycle values.
					}
					break
				}
				return f.tasks[projectID][idx], nil
			}
		}
	}
	return domain.Task{}, app.ErrNotFound
}

// DeleteTask deletes task.
func (f *fakeService) DeleteTask(_ context.Context, taskID string, mode app.DeleteMode) error {
	for projectID := range f.tasks {
		for idx := range f.tasks[projectID] {
			task := f.tasks[projectID][idx]
			if task.ID != taskID {
				continue
			}
			switch mode {
			case app.DeleteModeArchive:
				now := time.Now().UTC()
				f.tasks[projectID][idx].ArchivedAt = &now
				return nil
			case app.DeleteModeHard:
				f.tasks[projectID] = append(f.tasks[projectID][:idx], f.tasks[projectID][idx+1:]...)
				return nil
			default:
				return app.ErrInvalidDeleteMode
			}
		}
	}
	return app.ErrNotFound
}

// RestoreTask restores task.
func (f *fakeService) RestoreTask(_ context.Context, taskID string) (domain.Task, error) {
	for projectID := range f.tasks {
		for idx := range f.tasks[projectID] {
			if f.tasks[projectID][idx].ID == taskID {
				f.tasks[projectID][idx].ArchivedAt = nil
				return f.tasks[projectID][idx], nil
			}
		}
	}
	return domain.Task{}, app.ErrNotFound
}

// RenameTask renames task.
func (f *fakeService) RenameTask(_ context.Context, taskID, title string) (domain.Task, error) {
	for projectID := range f.tasks {
		for idx := range f.tasks[projectID] {
			if f.tasks[projectID][idx].ID == taskID {
				f.tasks[projectID][idx].Title = strings.TrimSpace(title)
				return f.tasks[projectID][idx], nil
			}
		}
	}
	return domain.Task{}, app.ErrNotFound
}

// projectByID returns project by id.
func (f *fakeService) projectByID(projectID string) (domain.Project, bool) {
	for _, project := range f.projects {
		if project.ID == projectID {
			return project, true
		}
	}
	return domain.Project{}, false
}

// taskByID returns task by id.
func (f *fakeService) taskByID(taskID string) (domain.Task, bool) {
	for projectID := range f.tasks {
		for _, task := range f.tasks[projectID] {
			if task.ID == taskID {
				return task, true
			}
		}
	}
	return domain.Task{}, false
}

// commentThreadKey builds a deterministic key for one comment target.
func commentThreadKey(projectID string, targetType domain.CommentTargetType, targetID string) string {
	return strings.TrimSpace(projectID) + "|" + strings.TrimSpace(string(targetType)) + "|" + strings.TrimSpace(targetID)
}

// TestModelLoadAndNavigation verifies behavior for the covered scenario.
func TestModelLoadAndNavigation(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c1, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	c2, _ := domain.NewColumn("c2", p.ID, "Done", 1, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c1.ID,
		Position:  0,
		Title:     "Ship",
		Priority:  domain.PriorityMedium,
	}, now)

	svc := newFakeService([]domain.Project{p}, []domain.Column{c1, c2}, []domain.Task{task})
	m := loadReadyModel(t, NewModel(
		svc,
		WithTaskFieldConfig(TaskFieldConfig{
			ShowPriority:    true,
			ShowDueDate:     true,
			ShowLabels:      true,
			ShowDescription: true,
		}),
	))

	if len(m.projects) != 1 || len(m.columns) != 2 || len(m.tasks) != 1 {
		t.Fatalf("unexpected loaded model: %#v", m)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyRight})
	if m.selectedColumn != 1 {
		t.Fatalf("expected selectedColumn=1, got %d", m.selectedColumn)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyLeft})
	if m.selectedColumn != 0 {
		t.Fatalf("expected selectedColumn=0, got %d", m.selectedColumn)
	}
}

// TestModelQuickAddMoveArchiveRestoreDelete verifies behavior for the covered scenario.
func TestModelQuickAddMoveArchiveRestoreDelete(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c1, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	c2, _ := domain.NewColumn("c2", p.ID, "Done", 1, 0, now)
	existing, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c1.ID,
		Position:  0,
		Title:     "Existing",
		Priority:  domain.PriorityLow,
	}, now)

	svc := newFakeService([]domain.Project{p}, []domain.Column{c1, c2}, []domain.Task{existing})
	m := loadReadyModel(t, NewModel(
		svc,
		WithTaskFieldConfig(TaskFieldConfig{
			ShowPriority:    true,
			ShowDueDate:     true,
			ShowLabels:      true,
			ShowDescription: true,
		}),
	))

	m = applyMsg(t, m, keyRune('n'))
	m = applyMsg(t, m, keyRune('N'))
	m = applyMsg(t, m, keyRune('e'))
	m = applyMsg(t, m, keyRune('w'))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if len(svc.tasks[p.ID]) != 2 {
		t.Fatalf("expected 2 tasks after quick add, got %d", len(svc.tasks[p.ID]))
	}

	m = applyMsg(t, m, keyRune(']'))
	moved, ok := svc.taskByID("t-new")
	if !ok || moved.ColumnID != c2.ID {
		t.Fatalf("expected created task to move to column %q, got %#v ok=%t", c2.ID, moved, ok)
	}

	m = applyMsg(t, m, keyRune('d'))
	if m.mode != modeConfirmAction {
		t.Fatalf("expected confirm mode for archive, got %v", m.mode)
	}
	m.confirmChoice = 0
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	archived, ok := svc.taskByID("t-new")
	if !ok || archived.ArchivedAt == nil {
		t.Fatalf("expected selected task archived, got %#v ok=%t", archived, ok)
	}
	m = applyMsg(t, m, keyRune('u'))
	restored, ok := svc.taskByID("t-new")
	if !ok || restored.ArchivedAt != nil {
		t.Fatalf("expected selected task restored, got %#v ok=%t", restored, ok)
	}

	m = applyMsg(t, m, keyRune('D'))
	if m.mode != modeConfirmAction {
		t.Fatalf("expected confirm mode for hard delete, got %v", m.mode)
	}
	m = applyMsg(t, m, keyRune('y'))
	if len(svc.tasks[p.ID]) != 1 {
		t.Fatalf("expected hard delete to remove task, got %d tasks", len(svc.tasks[p.ID]))
	}
}

// TestModelCreateTaskFocusesNewTask verifies that create-task reload focuses the created row.
func TestModelCreateTaskFocusesNewTask(t *testing.T) {
	now := time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	existing, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Existing task",
		Priority:  domain.PriorityMedium,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{existing})
	m := loadReadyModel(t, NewModel(svc))

	m = applyMsg(t, m, keyRune('n'))
	for _, r := range []rune("New focus task") {
		m = applyMsg(t, m, keyRune(r))
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})

	if task, ok := m.selectedTaskInCurrentColumn(); !ok || task.ID != "t-new" {
		t.Fatalf("expected focus on created task t-new, got %#v ok=%t", task, ok)
	}
	if m.selectedTask != 1 {
		t.Fatalf("expected selectedTask index to move to new row, got %d", m.selectedTask)
	}
	if svc.createTaskCalls != 1 {
		t.Fatalf("expected create task to be submitted once, got %d", svc.createTaskCalls)
	}
}

// TestModelProjectSwitchAndSearch verifies behavior for the covered scenario.
func TestModelProjectSwitchAndSearch(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p1, _ := domain.NewProject("p1", "A", "", now)
	p2, _ := domain.NewProject("p2", "B", "", now)
	c1, _ := domain.NewColumn("c1", p1.ID, "To Do", 0, 0, now)
	c2, _ := domain.NewColumn("c2", p2.ID, "To Do", 0, 0, now)
	t1, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p1.ID,
		ColumnID:  c1.ID,
		Position:  0,
		Title:     "Alpha task",
		Priority:  domain.PriorityLow,
	}, now)
	t2, _ := domain.NewTask(domain.TaskInput{
		ID:        "t2",
		ProjectID: p2.ID,
		ColumnID:  c2.ID,
		Position:  0,
		Title:     "Beta task",
		Priority:  domain.PriorityLow,
	}, now)

	svc := newFakeService([]domain.Project{p1, p2}, []domain.Column{c1, c2}, []domain.Task{t1, t2})
	m := loadReadyModel(t, NewModel(svc))

	m = applyMsg(t, m, keyRune('p'))
	if m.mode != modeProjectPicker {
		t.Fatalf("expected project picker mode, got %v", m.mode)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyRight})
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.selectedProject != 1 {
		t.Fatalf("expected selectedProject=1 after picker choose, got %d", m.selectedProject)
	}
	if got := m.status; got == "project switched" {
		t.Fatalf("expected project switch to avoid stale board notification, got %q", got)
	}

	m = applyMsg(t, m, keyRune('/'))
	m = applyMsg(t, m, keyRune('B'))
	m = applyMsg(t, m, keyRune('e'))
	m = applyMsg(t, m, keyRune('t'))
	m = applyMsg(t, m, keyRune('a'))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if len(m.tasks) != 1 || !strings.Contains(m.tasks[0].Title, "Beta") {
		t.Fatalf("expected filtered tasks to include only beta, got %#v", m.tasks)
	}
}

// TestModelCrossProjectSearchResultsAndJump verifies behavior for the covered scenario.
func TestModelCrossProjectSearchResultsAndJump(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p1, _ := domain.NewProject("p1", "Inbox", "", now)
	p2, _ := domain.NewProject("p2", "Client", "", now)
	c1, _ := domain.NewColumn("c1", p1.ID, "To Do", 0, 0, now)
	c2, _ := domain.NewColumn("c2", p2.ID, "To Do", 0, 0, now)
	t1, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p1.ID,
		ColumnID:  c1.ID,
		Position:  0,
		Title:     "Local task",
		Priority:  domain.PriorityLow,
	}, now)
	t2, _ := domain.NewTask(domain.TaskInput{
		ID:        "t2",
		ProjectID: p2.ID,
		ColumnID:  c2.ID,
		Position:  0,
		Title:     "Client roadmap",
		Priority:  domain.PriorityLow,
	}, now)

	svc := newFakeService([]domain.Project{p1, p2}, []domain.Column{c1, c2}, []domain.Task{t1, t2})
	m := loadReadyModel(t, NewModel(svc))
	m = applyMsg(t, m, keyRune('/'))
	m.searchCrossProject = true
	m = applyMsg(t, m, keyRune('c'))
	m = applyMsg(t, m, keyRune('l'))
	m = applyMsg(t, m, keyRune('i'))
	m = applyMsg(t, m, keyRune('e'))
	m = applyMsg(t, m, keyRune('n'))
	m = applyMsg(t, m, keyRune('t'))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeSearchResults {
		t.Fatalf("expected search results mode, got %v", m.mode)
	}
	if len(m.searchMatches) == 0 || m.searchMatches[0].Task.ID != "t2" {
		t.Fatalf("expected cross-project match for t2, got %#v", m.searchMatches)
	}
	levels := canonicalSearchLevels(m.searchLevels)
	if len(svc.lastSearchFilter.Levels) != len(levels) {
		t.Fatalf("expected forwarded levels %#v, got %#v", levels, svc.lastSearchFilter.Levels)
	}
	for idx, level := range levels {
		if svc.lastSearchFilter.Levels[idx] != level {
			t.Fatalf("expected forwarded levels %#v, got %#v", levels, svc.lastSearchFilter.Levels)
		}
	}
	if svc.lastSearchFilter.Mode != app.SearchModeHybrid {
		t.Fatalf("expected search mode %q, got %q", app.SearchModeHybrid, svc.lastSearchFilter.Mode)
	}
	if svc.lastSearchFilter.Sort != app.SearchSortRankDesc {
		t.Fatalf("expected search sort %q, got %q", app.SearchSortRankDesc, svc.lastSearchFilter.Sort)
	}
	if svc.lastSearchFilter.Offset != 0 {
		t.Fatalf("expected search offset 0, got %d", svc.lastSearchFilter.Offset)
	}
	if svc.lastSearchFilter.Limit != defaultSearchResultsLimit {
		t.Fatalf("expected search limit %d, got %d", defaultSearchResultsLimit, svc.lastSearchFilter.Limit)
	}
	if len(svc.lastSearchFilter.Kinds) != 0 {
		t.Fatalf("expected no search kinds forwarded by default, got %#v", svc.lastSearchFilter.Kinds)
	}
	if len(svc.lastSearchFilter.LabelsAny) != 0 {
		t.Fatalf("expected no search labels_any forwarded by default, got %#v", svc.lastSearchFilter.LabelsAny)
	}
	if len(svc.lastSearchFilter.LabelsAll) != 0 {
		t.Fatalf("expected no search labels_all forwarded by default, got %#v", svc.lastSearchFilter.LabelsAll)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.selectedProject != 1 {
		t.Fatalf("expected jump to second project, got %d", m.selectedProject)
	}
	if task, ok := m.selectedTaskInCurrentColumn(); !ok || task.ID != "t2" {
		t.Fatalf("expected selected task t2 after jump, got %#v ok=%t", task, ok)
	}
}

// TestBoardStatusTextSuppressesTransientStatuses verifies stale cancel/focus/loading text cannot reserve board footer rows.
func TestBoardStatusTextSuppressesTransientStatuses(t *testing.T) {
	m := NewModel(newFakeService(nil, nil, nil))
	for _, status := range []string{
		"",
		"ready",
		"cancelled",
		"project switched",
		"board focus",
		"loading thread...",
		"due picker cancelled",
		"text selection mode enabled",
		"edit task",
	} {
		m.status = status
		if got := m.boardStatusText(); got != "" {
			t.Fatalf("status %q should be suppressed, got %q", status, got)
		}
	}
	m.status = "task updated"
	if got := m.boardStatusText(); got != "task updated" {
		t.Fatalf("expected meaningful status to remain visible, got %q", got)
	}
}

// TestModelAddAndEditProject verifies behavior for the covered scenario.
func TestModelAddAndEditProject(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	existing, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-existing",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Existing task",
		Priority:  domain.PriorityMedium,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{existing})
	m := loadReadyModel(t, NewModel(svc))

	m = applyMsg(t, m, keyRune('N'))
	if m.mode != modeAddProject {
		t.Fatalf("expected add project mode, got %v", m.mode)
	}
	m = applyMsg(t, m, keyRune('R'))
	m = applyMsg(t, m, keyRune('o'))
	m = applyMsg(t, m, keyRune('a'))
	m = applyMsg(t, m, keyRune('d'))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if len(m.projects) < 2 {
		t.Fatalf("expected project created, got %#v", m.projects)
	}
	if m.selectedProject != len(m.projects)-1 {
		t.Fatalf("expected selection on new project, got %d", m.selectedProject)
	}
	selected := m.projects[m.selectedProject]
	if selected.ID != "p-new" {
		t.Fatalf("expected created project selected, got %q", selected.ID)
	}
	if len(m.tasks) != 0 {
		t.Fatalf("expected fresh project task list, got %#v", m.tasks)
	}
	for _, column := range m.columns {
		if column.ProjectID != selected.ID {
			t.Fatalf("expected columns for project %q, got %#v", selected.ID, m.columns)
		}
	}

	m = applyMsg(t, m, keyRune('M'))
	if m.mode != modeEditProject {
		t.Fatalf("expected edit project mode, got %v", m.mode)
	}
	m.projectFormInputs[0].SetValue("Renamed")
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if got := m.projects[m.selectedProject].Name; got != "Renamed" {
		t.Fatalf("expected project renamed, got %q", got)
	}
}

// TestModelEditProjectRootPathTypingPreservesPrintableKeys verifies focused project text inputs keep printable characters instead of triggering form-level actions.
func TestModelEditProjectRootPathTypingPreservesPrintableKeys(t *testing.T) {
	now := time.Date(2026, 3, 14, 0, 5, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})))

	m = applyMsg(t, m, keyRune('M'))
	if m.mode != modeEditProject {
		t.Fatalf("expected edit project mode, got %v", m.mode)
	}
	m = applyCmd(t, m, m.focusProjectFormField(projectFieldRootPath))
	if m.projectFormFocus != projectFieldRootPath {
		t.Fatalf("expected root_path focus, got %d", m.projectFormFocus)
	}

	before := m.projectFormInputs[projectFieldRootPath].Value()
	m = applyMsg(t, m, keyRune('r'))
	if m.mode != modeEditProject {
		t.Fatalf("expected lowercase r to keep edit project mode, got %v", m.mode)
	}
	if got := m.projectFormInputs[projectFieldRootPath].Value(); got != before+"r" {
		t.Fatalf("expected lowercase r to type into root_path, got %q", got)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: 'R', Text: "R", Mod: tea.ModShift})
	if got := m.projectFormInputs[projectFieldRootPath].Value(); got != before+"rR" {
		t.Fatalf("expected uppercase R to type into root_path, got %q", got)
	}
}

// TestModelCommandPaletteAndQuickActions verifies behavior for the covered scenario.
func TestModelCommandPaletteAndQuickActions(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityLow,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})
	m := loadReadyModel(t, NewModel(svc))

	m = applyMsg(t, m, keyRune(':'))
	if m.mode != modeCommandPalette {
		t.Fatalf("expected command palette mode, got %v", m.mode)
	}
	m = applyMsg(t, m, keyRune('s'))
	m = applyMsg(t, m, keyRune('e'))
	m = applyMsg(t, m, keyRune('a'))
	m = applyMsg(t, m, keyRune('r'))
	m = applyMsg(t, m, keyRune('c'))
	m = applyMsg(t, m, keyRune('h'))
	m = applyMsg(t, m, keyRune('-'))
	m = applyMsg(t, m, keyRune('a'))
	m = applyMsg(t, m, keyRune('l'))
	m = applyMsg(t, m, keyRune('l'))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeSearch {
		t.Fatalf("expected search mode after search-all, got %v", m.mode)
	}
	if !m.searchCrossProject {
		t.Fatal("expected search-all command to enable cross-project scope")
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})

	updated, cmd := m.executeCommandPalette("search-project")
	m = applyResult(t, updated, cmd)
	if m.mode != modeSearch {
		t.Fatalf("expected search mode after search-project, got %v", m.mode)
	}
	if m.searchCrossProject {
		t.Fatal("expected search-project command to disable cross-project scope")
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})

	m = applyMsg(t, m, keyRune('.'))
	if m.mode != modeQuickActions {
		t.Fatalf("expected quick actions mode, got %v", m.mode)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeTaskInfo {
		t.Fatalf("expected quick action enter to open task info, got %v", m.mode)
	}

	m.mode = modeNone
	m = applyMsg(t, m, keyRune(':'))
	m = applyMsg(t, m, keyRune('z'))
	m = applyMsg(t, m, keyRune('z'))
	m = applyMsg(t, m, keyRune('z'))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if !strings.Contains(m.status, "unknown command") {
		t.Fatalf("expected unknown command status, got %q", m.status)
	}
}

// TestModelThreadModeProjectAndPostCommentUsesConfiguredIdentity verifies project-thread rendering and comment ownership attribution.
func TestModelThreadModeProjectAndPostCommentUsesConfiguredIdentity(t *testing.T) {
	now := time.Date(2026, 2, 23, 10, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "# Project Overview\n\n- keep momentum", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
		Metadata: domain.TaskMetadata{
			BlockedReason: "requires global follow-up",
		},
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})

	existing, err := domain.NewComment(domain.CommentInput{
		ID:           "cm-existing",
		ProjectID:    p.ID,
		TargetType:   domain.CommentTargetTypeProject,
		TargetID:     p.ID,
		Summary:      "Initial project summary",
		BodyMarkdown: "Initial **project** thread comment",
		ActorID:      "system-bot",
		ActorName:    "System Bot",
		ActorType:    domain.ActorTypeSystem,
	}, now)
	if err != nil {
		t.Fatalf("NewComment(existing) error = %v", err)
	}
	projectKey := commentThreadKey(p.ID, domain.CommentTargetTypeProject, p.ID)
	svc.comments[projectKey] = append(svc.comments[projectKey], existing)

	m := loadReadyModel(t, NewModel(
		svc,
		WithIdentityConfig(IdentityConfig{
			ActorID:          "lane-user-17",
			DisplayName:      "Lane User",
			DefaultActorType: "agent",
		}),
	))
	m.identityActorID = "lane-user-17"

	updated, cmd := m.executeCommandPalette("thread-project")
	m = applyResult(t, updated, cmd)
	if m.mode != modeThread {
		t.Fatalf("expected thread mode, got %v", m.mode)
	}
	if m.threadTarget.TargetType != domain.CommentTargetTypeProject || m.threadTarget.TargetID != p.ID {
		t.Fatalf("unexpected project thread target %#v", m.threadTarget)
	}
	if m.threadComposerActive {
		t.Fatal("expected thread to open in read-first mode with composer inactive")
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyRight})
	m = applyMsg(t, m, keyRune('i'))
	if !m.threadComposerActive {
		t.Fatal("expected i to activate thread comment composer")
	}
	m.threadInput.SetValue("New _markdown_ project comment")
	m = applyMsg(t, m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})

	comments := svc.comments[projectKey]
	if len(comments) != 2 {
		t.Fatalf("expected 2 project comments after post, got %#v", comments)
	}
	last := comments[len(comments)-1]
	if last.ActorID != "lane-user-17" {
		t.Fatalf("expected configured actor id lane-user-17, got %q", last.ActorID)
	}
	if last.ActorName != "Lane User" {
		t.Fatalf("expected configured actor name Lane User, got %q", last.ActorName)
	}
	if last.ActorType != domain.ActorTypeAgent {
		t.Fatalf("expected configured actor type agent, got %q", last.ActorType)
	}
	if got := strings.TrimSpace(svc.lastCreateComment.ActorID); got != "lane-user-17" {
		t.Fatalf("expected create-comment actor_id lane-user-17, got %q", got)
	}
	if got := strings.TrimSpace(svc.lastCreateComment.ActorName); got != "Lane User" {
		t.Fatalf("expected create-comment actor_name Lane User, got %q", got)
	}
	if got := svc.lastCreateComment.ActorType; got != domain.ActorTypeAgent {
		t.Fatalf("expected create-comment actor_type agent, got %q", got)
	}
	if len(m.threadComments) != 2 {
		t.Fatalf("expected in-memory thread comments to append, got %#v", m.threadComments)
	}
	rendered := stripANSI(fmt.Sprint(m.View().Content))
	if !strings.Contains(rendered, "Project Overview") {
		t.Fatalf("expected markdown-rendered project description, got\n%s", rendered)
	}
	if !strings.Contains(rendered, "Lane User (lane-user-17)") || !strings.Contains(rendered, "type: agent") {
		t.Fatalf("expected ownership metadata in thread view, got\n%s", rendered)
	}
	if !strings.Contains(rendered, "summary: Initial project summary") {
		t.Fatalf("expected comment summary visible in thread view, got\n%s", rendered)
	}
}

// TestModelThreadModeFromTaskInfoAndBack verifies task-info thread shortcut and back navigation.
func TestModelThreadModeFromTaskInfoAndBack(t *testing.T) {
	now := time.Date(2026, 2, 23, 10, 30, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	phase, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-phase",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Kind:      domain.WorkKindPhase,
		Title:     "Phase 1",
		Priority:  domain.PriorityMedium,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{phase})
	m := loadReadyModel(t, NewModel(svc))

	m = applyMsg(t, m, keyRune('i'))
	if m.mode != modeTaskInfo {
		t.Fatalf("expected task info mode, got %v", m.mode)
	}
	m = applyMsg(t, m, keyRune('c'))
	if m.mode != modeThread {
		t.Fatalf("expected task-info shortcut to open thread mode, got %v", m.mode)
	}
	if m.threadTarget.TargetType != domain.CommentTargetTypePhase {
		t.Fatalf("expected phase target type mapping, got %#v", m.threadTarget)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.mode != modeTaskInfo {
		t.Fatalf("expected esc to return to task info from thread mode, got %v", m.mode)
	}
}

// TestModelThreadTabAndShiftTabMoveInOppositeDirections verifies thread panel traversal reverses correctly on shift+tab.
func TestModelThreadTabAndShiftTabMoveInOppositeDirections(t *testing.T) {
	now := time.Date(2026, 3, 13, 19, 35, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{project}, []domain.Column{column}, []domain.Task{task})))

	m = applyMsg(t, m, keyRune('i'))
	m = applyMsg(t, m, keyRune('c'))
	if m.mode != modeThread {
		t.Fatalf("expected thread mode, got %v", m.mode)
	}
	if m.threadPanelFocus != threadPanelComments {
		t.Fatalf("expected thread comments focus on open, got %d", m.threadPanelFocus)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	if m.threadPanelFocus != threadPanelContext {
		t.Fatalf("expected tab to advance to context, got %d", m.threadPanelFocus)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	if m.threadPanelFocus != threadPanelComments {
		t.Fatalf("expected shift+tab to reverse back to comments, got %d", m.threadPanelFocus)
	}

	threadView := stripANSI(fmt.Sprint(m.View().Content))
	if !strings.Contains(threadView, "tab/shift+tab") {
		t.Fatalf("expected thread help to advertise shift+tab, got\n%s", threadView)
	}
}

// TestModelThreadCommentIdentityFallbacks verifies safe identity fallback behavior during comment creation.
func TestModelThreadCommentIdentityFallbacks(t *testing.T) {
	now := time.Date(2026, 2, 23, 11, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityLow,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})
	m := loadReadyModel(t, NewModel(
		svc,
		WithIdentityConfig(IdentityConfig{
			DisplayName:      " ",
			DefaultActorType: "robot",
		}),
	))

	updated, cmd := m.startSelectedWorkItemThread(modeNone)
	m = applyResult(t, updated, cmd)
	if m.mode != modeThread {
		t.Fatalf("expected work-item thread mode, got %v", m.mode)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyRight})
	m = applyMsg(t, m, keyRune('i'))
	m.threadInput.SetValue("fallback check")
	m = applyMsg(t, m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})

	itemKey := commentThreadKey(p.ID, domain.CommentTargetTypeTask, task.ID)
	comments := svc.comments[itemKey]
	if len(comments) != 1 {
		t.Fatalf("expected one posted comment, got %#v", comments)
	}
	if comments[0].ActorID != "tillsyn-user" {
		t.Fatalf("expected fallback actor id tillsyn-user, got %q", comments[0].ActorID)
	}
	if comments[0].ActorName != "tillsyn-user" {
		t.Fatalf("expected fallback actor name tillsyn-user, got %q", comments[0].ActorName)
	}
	if comments[0].ActorType != domain.ActorTypeUser {
		t.Fatalf("expected fallback actor type user, got %q", comments[0].ActorType)
	}
}

// TestModelThreadReadModeRequiresExplicitComposer verifies thread mode starts read-first and requires explicit composer activation to post.
func TestModelThreadReadModeRequiresExplicitComposer(t *testing.T) {
	now := time.Date(2026, 2, 23, 11, 15, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityLow,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})
	m := loadReadyModel(t, NewModel(svc))

	updated, cmd := m.executeCommandPalette("thread-item")
	m = applyResult(t, updated, cmd)
	if m.mode != modeThread {
		t.Fatalf("expected thread mode, got %v", m.mode)
	}
	if m.threadComposerActive {
		t.Fatal("expected read-first thread mode with composer inactive")
	}

	m = applyMsg(t, m, keyRune('i'))
	if m.threadComposerActive {
		t.Fatal("expected i on details panel to leave composer inactive")
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyRight})
	m = applyMsg(t, m, keyRune('i'))
	if !m.threadComposerActive {
		t.Fatal("expected composer active after i from comments panel")
	}
	m.threadInput.SetValue("explicit composer")
	m = applyMsg(t, m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})

	itemKey := commentThreadKey(p.ID, domain.CommentTargetTypeTask, task.ID)
	comments := svc.comments[itemKey]
	if len(comments) != 1 {
		t.Fatalf("expected one posted comment, got %#v", comments)
	}
}

// TestModelThreadComposerAllowsTypingEditRune verifies composer input accepts plain 'e' text while active.
func TestModelThreadComposerAllowsTypingEditRune(t *testing.T) {
	now := time.Date(2026, 2, 23, 11, 20, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityLow,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})
	m := loadReadyModel(t, NewModel(svc))

	updated, cmd := m.executeCommandPalette("thread-item")
	m = applyResult(t, updated, cmd)
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyRight})
	m = applyMsg(t, m, keyRune('i'))
	m = applyMsg(t, m, keyRune('e'))
	m = applyMsg(t, m, keyRune('x'))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})

	itemKey := commentThreadKey(p.ID, domain.CommentTargetTypeTask, task.ID)
	comments := svc.comments[itemKey]
	if len(comments) != 1 {
		t.Fatalf("expected one posted comment, got %#v", comments)
	}
	if comments[0].BodyMarkdown != "ex" {
		t.Fatalf("expected posted comment body %q, got %q", "ex", comments[0].BodyMarkdown)
	}
}

// TestModelThreadReadModeEditShortcutStartsTaskEditForm verifies read-mode details-first flow before task edit.
func TestModelThreadReadModeEditShortcutStartsTaskEditForm(t *testing.T) {
	now := time.Date(2026, 2, 23, 11, 22, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:          "t1",
		ProjectID:   p.ID,
		ColumnID:    c.ID,
		Position:    0,
		Title:       "Task",
		Description: "## Details\n\n- read me",
		Priority:    domain.PriorityLow,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})
	m := loadReadyModel(t, NewModel(svc))

	updated, cmd := m.executeCommandPalette("thread-item")
	m = applyResult(t, updated, cmd)
	if m.mode != modeThread {
		t.Fatalf("expected thread mode, got %v", m.mode)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeEditTask {
		t.Fatalf("expected edit-task mode from thread details panel, got %v", m.mode)
	}
	if strings.TrimSpace(m.editingTaskID) != task.ID {
		t.Fatalf("expected editing task id %q, got %q", task.ID, m.editingTaskID)
	}
	if got := strings.TrimSpace(m.formInputs[taskFieldDescription].Value()); !strings.Contains(got, "Details") {
		t.Fatalf("expected edit form description prefilled from thread target, got %q", got)
	}
}

// TestModelThreadProjectReadModeEditShortcutStartsProjectEditForm verifies project-thread details-first flow before project edit.
func TestModelThreadProjectReadModeEditShortcutStartsProjectEditForm(t *testing.T) {
	now := time.Date(2026, 2, 23, 11, 24, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "## Overview\n\n- read first", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{project}, []domain.Column{column}, nil)))

	updated, cmd := m.executeCommandPalette("thread-project")
	m = applyResult(t, updated, cmd)
	if m.mode != modeThread {
		t.Fatalf("expected project thread mode, got %v", m.mode)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeEditProject {
		t.Fatalf("expected edit-project mode from thread details panel, got %v", m.mode)
	}
	if strings.TrimSpace(m.editingProjectID) != project.ID {
		t.Fatalf("expected editing project id %q, got %q", project.ID, m.editingProjectID)
	}
	if got := strings.TrimSpace(m.projectFormInputs[projectFieldDescription].Value()); !strings.Contains(got, "Overview") {
		t.Fatalf("expected project description prefilled from thread target, got %q", got)
	}
}

// TestModelThreadDetailsPanelEnterStartsTaskEdit verifies enter on the focused details panel opens task edit.
func TestModelThreadDetailsPanelEnterStartsTaskEdit(t *testing.T) {
	now := time.Date(2026, 3, 3, 9, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:          "t1",
		ProjectID:   project.ID,
		ColumnID:    column.ID,
		Position:    0,
		Title:       "Task",
		Description: "initial description",
		Priority:    domain.PriorityMedium,
	}, now)
	svc := newFakeService([]domain.Project{project}, []domain.Column{column}, []domain.Task{task})
	m := loadReadyModel(t, NewModel(svc))

	updated, cmd := m.executeCommandPalette("thread-item")
	m = applyResult(t, updated, cmd)
	if m.mode != modeThread {
		t.Fatalf("expected thread mode, got %v", m.mode)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeEditTask {
		t.Fatalf("expected edit-task mode from thread details panel, got %v", m.mode)
	}
	if got := strings.TrimSpace(m.editingTaskID); got != task.ID {
		t.Fatalf("expected editing task id %q, got %q", task.ID, got)
	}
}

// TestModelThreadDescriptionFallsBackToTargetDetails verifies notification-opened threads use backing entity details when no thread body is provided.
func TestModelThreadDescriptionFallsBackToTargetDetails(t *testing.T) {
	now := time.Date(2026, 3, 3, 9, 15, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "## Project Details\n\n- keep this visible", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{project}, []domain.Column{column}, nil)))

	target, err := domain.NormalizeCommentTarget(domain.CommentTarget{
		ProjectID:  project.ID,
		TargetType: domain.CommentTargetTypeProject,
		TargetID:   project.ID,
	})
	if err != nil {
		t.Fatalf("normalize comment target: %v", err)
	}
	updated, cmd := m.startNotificationThread(target, "project notice", "")
	m = applyResult(t, updated, cmd)

	if got := strings.TrimSpace(m.threadDescriptionMarkdown); !strings.Contains(got, "Project Details") {
		t.Fatalf("expected fallback to project description markdown, got %q", got)
	}
	rendered := stripANSI(fmt.Sprint(m.View().Content))
	if !strings.Contains(rendered, "Project Details") {
		t.Fatalf("expected fallback description rendered in thread view, got\n%s", rendered)
	}
}

// TestModelTaskInfoShowsCommentPreview verifies task info renders recent markdown comments without requiring thread mode.
func TestModelTaskInfoShowsCommentPreview(t *testing.T) {
	now := time.Date(2026, 2, 23, 11, 30, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:          "t1",
		ProjectID:   p.ID,
		ColumnID:    c.ID,
		Position:    0,
		Title:       "Task",
		Description: "## Details\n\n- read me",
		Priority:    domain.PriorityMedium,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})

	comment, err := domain.NewComment(domain.CommentInput{
		ID:           "cm-1",
		ProjectID:    p.ID,
		TargetType:   domain.CommentTargetTypeTask,
		TargetID:     task.ID,
		Summary:      "comment summary preview",
		BodyMarkdown: "**latest** comment",
		ActorID:      "user-1",
		ActorName:    "User One",
		ActorType:    domain.ActorTypeUser,
	}, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("NewComment(comment) error = %v", err)
	}
	itemKey := commentThreadKey(p.ID, domain.CommentTargetTypeTask, task.ID)
	svc.comments[itemKey] = append(svc.comments[itemKey], comment)

	m := loadReadyModel(t, NewModel(svc))
	m = applyMsg(t, m, keyRune('i'))
	if m.mode != modeTaskInfo {
		t.Fatalf("expected task info mode, got %v", m.mode)
	}

	body := stripANSI(m.taskInfoBody.GetContent())
	if !strings.Contains(body, "comments (1)") {
		t.Fatalf("expected comment preview section in task info body, got %q", body)
	}
	if !strings.Contains(body, "latest") {
		t.Fatalf("expected markdown comment content preview in task info body, got %q", body)
	}
	if !strings.Contains(body, "summary: comment summary preview") {
		t.Fatalf("expected explicit comment summary in task info body, got %q", body)
	}
	if !strings.Contains(body, "Details") {
		t.Fatalf("expected markdown details content in task info body, got %q", body)
	}
}

// TestModelTaskInfoShowsFullCommentsList verifies task-info renders the full comments list with ownership metadata.
func TestModelTaskInfoShowsFullCommentsList(t *testing.T) {
	now := time.Date(2026, 3, 4, 8, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:          "t1",
		ProjectID:   p.ID,
		ColumnID:    c.ID,
		Position:    0,
		Title:       "Task",
		Description: "desc",
		Priority:    domain.PriorityMedium,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})
	itemKey := commentThreadKey(p.ID, domain.CommentTargetTypeTask, task.ID)
	for i := 1; i <= 6; i++ {
		comment, err := domain.NewComment(domain.CommentInput{
			ID:           fmt.Sprintf("cm-%d", i),
			ProjectID:    p.ID,
			TargetType:   domain.CommentTargetTypeTask,
			TargetID:     task.ID,
			Summary:      fmt.Sprintf("summary-%d", i),
			BodyMarkdown: fmt.Sprintf("body-%d", i),
			ActorID:      fmt.Sprintf("actor-%d", i),
			ActorName:    fmt.Sprintf("Actor %d", i),
			ActorType:    domain.ActorTypeUser,
		}, now.Add(time.Duration(i)*time.Minute))
		if err != nil {
			t.Fatalf("NewComment(%d) error = %v", i, err)
		}
		svc.comments[itemKey] = append(svc.comments[itemKey], comment)
	}

	m := loadReadyModel(t, NewModel(svc))
	m = applyMsg(t, m, keyRune('i'))
	if m.mode != modeTaskInfo {
		t.Fatalf("expected task info mode, got %v", m.mode)
	}
	body := stripANSI(m.taskInfoBody.GetContent())
	if !strings.Contains(body, "comments (6)") {
		t.Fatalf("expected comments count in task info body, got %q", body)
	}
	if !strings.Contains(body, "summary: summary-1") {
		t.Fatalf("expected oldest comment summary in task info body list, got %q", body)
	}
	if !strings.Contains(body, "id: cm-1") || !strings.Contains(body, "id: cm-6") {
		t.Fatalf("expected comment ids in task info body list, got %q", body)
	}
	if !strings.Contains(body, "[user] Actor 1 (actor-1)") || !strings.Contains(body, "[user] Actor 6 (actor-6)") {
		t.Fatalf("expected owner metadata rows in task info body list, got %q", body)
	}
	if strings.Contains(body, "older comments") {
		t.Fatalf("expected full comments list without preview truncation, got %q", body)
	}
}

// TestModelTaskInfoShowsMarkdownDetailsWhenCardDescriptionsHidden verifies task-info read mode still shows markdown details.
func TestModelTaskInfoShowsMarkdownDetailsWhenCardDescriptionsHidden(t *testing.T) {
	now := time.Date(2026, 2, 23, 11, 35, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:          "t1",
		ProjectID:   p.ID,
		ColumnID:    c.ID,
		Position:    0,
		Title:       "Task",
		Description: "## Hidden Card Description\n\n- still visible in task info",
		Priority:    domain.PriorityMedium,
	}, now)
	m := loadReadyModel(t, NewModel(
		newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task}),
		WithTaskFieldConfig(TaskFieldConfig{
			ShowPriority:    false,
			ShowDueDate:     false,
			ShowLabels:      false,
			ShowDescription: false,
		}),
	))

	m = applyMsg(t, m, keyRune('i'))
	if m.mode != modeTaskInfo {
		t.Fatalf("expected task info mode, got %v", m.mode)
	}
	overlay := stripANSI(fmt.Sprint(m.renderFullPageNodeModeView().Content))
	if !strings.Contains(overlay, "Hidden Card Description") {
		t.Fatalf("expected markdown details visible in task info despite card-description toggle, got %q", overlay)
	}
}

// TestModelTaskInfoShowsStructuredMetadataSections verifies task-info renders objective/acceptance/validation/risk markdown sections.
func TestModelTaskInfoShowsStructuredMetadataSections(t *testing.T) {
	now := time.Date(2026, 3, 3, 10, 15, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:          "t1",
		ProjectID:   p.ID,
		ColumnID:    c.ID,
		Position:    0,
		Title:       "Task",
		Description: "details",
		Priority:    domain.PriorityMedium,
		Metadata: domain.TaskMetadata{
			Objective:          "objective token",
			AcceptanceCriteria: "acceptance token",
			ValidationPlan:     "validation token",
			RiskNotes:          "risk token",
		},
	}, now)
	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})))

	m = applyMsg(t, m, keyRune('i'))
	if m.mode != modeTaskInfo {
		t.Fatalf("expected task info mode, got %v", m.mode)
	}
	body := stripANSI(m.taskInfoBody.GetContent())
	for _, token := range []string{
		"objective",
		"objective token",
		"acceptance_criteria",
		"acceptance token",
		"validation_plan",
		"validation token",
		"risk_notes",
		"risk token",
	} {
		if !strings.Contains(body, token) {
			t.Fatalf("expected task info metadata token %q, got %q", token, body)
		}
	}
}

// TestModelEditTaskMetadataFieldsPrefillAndSubmit verifies edit-task prefill and save behavior for objective/acceptance/validation/risk metadata fields.
func TestModelEditTaskMetadataFieldsPrefillAndSubmit(t *testing.T) {
	now := time.Date(2026, 3, 3, 10, 25, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:          "t1",
		ProjectID:   p.ID,
		ColumnID:    c.ID,
		Position:    0,
		Title:       "Task",
		Description: "details",
		Priority:    domain.PriorityMedium,
		Metadata: domain.TaskMetadata{
			Objective:          "draft objective",
			AcceptanceCriteria: "draft acceptance",
			ValidationPlan:     "draft validation",
			RiskNotes:          "draft risk",
		},
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})
	m := loadReadyModel(t, NewModel(svc))

	m = applyMsg(t, m, keyRune('e'))
	if m.mode != modeEditTask {
		t.Fatalf("expected edit-task mode, got %v", m.mode)
	}
	if got := m.formInputs[taskFieldObjective].Value(); got != "draft objective" {
		t.Fatalf("expected objective prefill %q, got %q", "draft objective", got)
	}
	if got := m.formInputs[taskFieldAcceptanceCriteria].Value(); got != "draft acceptance" {
		t.Fatalf("expected acceptance prefill %q, got %q", "draft acceptance", got)
	}
	if got := m.formInputs[taskFieldValidationPlan].Value(); got != "draft validation" {
		t.Fatalf("expected validation prefill %q, got %q", "draft validation", got)
	}
	if got := m.formInputs[taskFieldRiskNotes].Value(); got != "draft risk" {
		t.Fatalf("expected risk prefill %q, got %q", "draft risk", got)
	}

	m.setTaskFormMarkdownDraft(taskFieldObjective, "updated objective", true)
	m.setTaskFormMarkdownDraft(taskFieldAcceptanceCriteria, "", true)
	m.setTaskFormMarkdownDraft(taskFieldValidationPlan, "updated validation", true)
	m.setTaskFormMarkdownDraft(taskFieldRiskNotes, "updated risk", true)
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})

	updated, ok := svc.taskByID(task.ID)
	if !ok {
		t.Fatalf("expected updated task %q in fake service", task.ID)
	}
	if got := updated.Metadata.Objective; got != "updated objective" {
		t.Fatalf("expected objective %q, got %q", "updated objective", got)
	}
	if got := updated.Metadata.AcceptanceCriteria; got != "" {
		t.Fatalf("expected acceptance criteria cleared, got %q", got)
	}
	if got := updated.Metadata.ValidationPlan; got != "updated validation" {
		t.Fatalf("expected validation plan %q, got %q", "updated validation", got)
	}
	if got := updated.Metadata.RiskNotes; got != "updated risk" {
		t.Fatalf("expected risk notes %q, got %q", "updated risk", got)
	}
}

// TestModelEditTaskMetadataEditorCtrlSSavesTask verifies editor-level ctrl+s persists existing task metadata.
func TestModelEditTaskMetadataEditorCtrlSSavesTask(t *testing.T) {
	now := time.Date(2026, 3, 3, 10, 40, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
		Metadata: domain.TaskMetadata{
			Objective: "draft objective",
		},
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})
	m := loadReadyModel(t, NewModel(svc))

	m = applyMsg(t, m, keyRune('e'))
	m.formFocus = taskFieldObjective
	m = applyMsg(t, m, keyRune('e'))
	if m.mode != modeDescriptionEditor {
		t.Fatalf("expected description editor for objective, got %v", m.mode)
	}
	m.descriptionEditorInput.SetValue("saved from editor")
	m.saveDescriptionEditor()
	m = applyCmd(t, m, m.closeDescriptionEditor(true))

	if m.mode != modeEditTask {
		t.Fatalf("expected edit mode after metadata editor save, got %v", m.mode)
	}
	updated, ok := svc.taskByID(task.ID)
	if !ok {
		t.Fatalf("expected updated task %q in fake service", task.ID)
	}
	if got := updated.Metadata.Objective; got != "saved from editor" {
		t.Fatalf("expected editor ctrl+s to persist objective, got %q", got)
	}
}

// TestModelTaskInfoDetailsViewportScrolls verifies task-info markdown details are bounded and scrollable.
func TestModelTaskInfoDetailsViewportScrolls(t *testing.T) {
	now := time.Date(2026, 3, 3, 12, 45, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	lines := make([]string, 0, 240)
	for i := 0; i < 240; i++ {
		lines = append(lines, fmt.Sprintf("line %03d full page md details", i))
	}
	task, _ := domain.NewTask(domain.TaskInput{
		ID:          "t1",
		ProjectID:   project.ID,
		ColumnID:    column.ID,
		Position:    0,
		Kind:        domain.WorkKindTask,
		Title:       "full page md",
		Description: strings.Join(lines, "\n"),
		Priority:    domain.PriorityMedium,
	}, now)

	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{project}, []domain.Column{column}, []domain.Task{task})))
	m = applyMsg(t, m, tea.WindowSizeMsg{Width: 100, Height: 28})
	m = applyMsg(t, m, keyRune('i'))
	if m.mode != modeTaskInfo {
		t.Fatalf("expected task info mode, got %v", m.mode)
	}

	before := m.taskInfoDetails.YOffset()
	beforeBody := m.taskInfoBody.YOffset()
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyPgDown})
	after := m.taskInfoDetails.YOffset()
	afterBody := m.taskInfoBody.YOffset()
	if after != before {
		t.Fatalf("expected details viewport to stay put on pgdown body scroll, before=%d after=%d", before, after)
	}
	if afterBody <= beforeBody {
		t.Fatalf("expected task-info body viewport to scroll on pgdown, before=%d after=%d", beforeBody, afterBody)
	}

	beforeJ := m.taskInfoBody.YOffset()
	m = applyMsg(t, m, keyRune('j'))
	afterJ := m.taskInfoBody.YOffset()
	if afterJ <= beforeJ {
		t.Fatalf("expected task-info body viewport to scroll on j, before=%d after=%d", beforeJ, afterJ)
	}

	beforeDown := m.taskInfoBody.YOffset()
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	afterDown := m.taskInfoBody.YOffset()
	if afterDown <= beforeDown {
		t.Fatalf("expected task-info body viewport to scroll on down arrow, before=%d after=%d", beforeDown, afterDown)
	}

	beforeK := m.taskInfoBody.YOffset()
	m = applyMsg(t, m, keyRune('k'))
	afterK := m.taskInfoBody.YOffset()
	if afterK >= beforeK {
		t.Fatalf("expected task-info body viewport to scroll up on k, before=%d after=%d", beforeK, afterK)
	}

	beforeUp := m.taskInfoBody.YOffset()
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
	afterUp := m.taskInfoBody.YOffset()
	if afterUp >= beforeUp {
		t.Fatalf("expected task-info body viewport to scroll up on up arrow, before=%d after=%d", beforeUp, afterUp)
	}

	m.taskInfoBody.GotoTop()
	beforeMouse := m.taskInfoDetails.YOffset()
	beforeMouseBody := m.taskInfoBody.YOffset()
	m = applyMsg(t, m, tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	afterMouse := m.taskInfoDetails.YOffset()
	afterMouseBody := m.taskInfoBody.YOffset()
	if afterMouse != beforeMouse {
		t.Fatalf("expected details preview to stay top-aligned on mouse wheel, before=%d after=%d", beforeMouse, afterMouse)
	}
	if afterMouseBody <= beforeMouseBody {
		t.Fatalf("expected task-info body viewport to scroll on mouse wheel, before=%d after=%d", beforeMouseBody, afterMouseBody)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyHome})
	if got := m.taskInfoDetails.YOffset(); got != 0 {
		t.Fatalf("expected home to reset details viewport to top, got y offset=%d", got)
	}
	if got := m.taskInfoBody.YOffset(); got != 0 {
		t.Fatalf("expected home to reset task-info body viewport to top, got y offset=%d", got)
	}
}

// TestCommentTargetTypeForWorkKind verifies work-kind to comment-target mapping coverage.
func TestCommentTargetTypeForWorkKind(t *testing.T) {
	cases := []struct {
		kind   domain.WorkKind
		want   domain.CommentTargetType
		wantOK bool
	}{
		{kind: domain.WorkKindTask, want: domain.CommentTargetTypeTask, wantOK: true},
		{kind: domain.WorkKindSubtask, want: domain.CommentTargetTypeSubtask, wantOK: true},
		{kind: domain.WorkKindPhase, want: domain.CommentTargetTypePhase, wantOK: true},
		{kind: domain.WorkKindDecision, want: domain.CommentTargetTypeDecision, wantOK: true},
		{kind: domain.WorkKindNote, want: domain.CommentTargetTypeNote, wantOK: true},
		{kind: domain.WorkKind("unknown"), want: "", wantOK: false},
	}

	for _, tc := range cases {
		got, ok := commentTargetTypeForWorkKind(tc.kind)
		if ok != tc.wantOK {
			t.Fatalf("commentTargetTypeForWorkKind(%q) ok=%t want=%t", tc.kind, ok, tc.wantOK)
		}
		if got != tc.want {
			t.Fatalf("commentTargetTypeForWorkKind(%q) target=%q want=%q", tc.kind, got, tc.want)
		}
	}
}

// TestModelCommandPaletteFuzzyAbbreviationExecutesNewSubtask verifies fuzzy abbreviations can target commands like new-subtask.
func TestModelCommandPaletteFuzzyAbbreviationExecutesNewSubtask(t *testing.T) {
	now := time.Date(2026, 2, 23, 9, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	parent, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-parent",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Parent",
		Priority:  domain.PriorityMedium,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{parent})
	m := loadReadyModel(t, NewModel(svc))

	m = applyMsg(t, m, keyRune(':'))
	m = applyMsg(t, m, keyRune('n'))
	m = applyMsg(t, m, keyRune('s'))
	if len(m.commandMatches) == 0 {
		t.Fatal("expected fuzzy command matches for abbreviation 'ns'")
	}
	if got := m.commandMatches[0].Command; got != "new-subtask" {
		t.Fatalf("expected top fuzzy command match new-subtask, got %q", got)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeAddTask {
		t.Fatalf("expected add-task mode after executing new-subtask, got %v", m.mode)
	}
	if got := m.taskFormKind; got != domain.WorkKindSubtask {
		t.Fatalf("expected task form kind subtask, got %q", got)
	}
	if got := m.taskFormParentID; got != parent.ID {
		t.Fatalf("expected subtask parent %q, got %q", parent.ID, got)
	}
}

// TestCommandPaletteInitialismScoringPrefersExpectedAbbreviations verifies command initialisms rank intended matches above unrelated commands.
func TestCommandPaletteInitialismScoringPrefersExpectedAbbreviations(t *testing.T) {
	if got := commandPaletteInitialism("new-subtask"); got != "ns" {
		t.Fatalf("commandPaletteInitialism(new-subtask) = %q, want ns", got)
	}
	subtaskScore, ok := scoreCommandPaletteItem("ns", commandPaletteItem{
		Command:     "new-subtask",
		Aliases:     []string{"task-subtask", "ns"},
		Description: "create subtask for selected item",
	})
	if !ok {
		t.Fatal("scoreCommandPaletteItem(new-subtask) = no match, want match")
	}
	authScore, ok := scoreCommandPaletteItem("ns", commandPaletteItem{
		Command:     "auth-access",
		Aliases:     []string{"auths"},
		Description: "review access requests; list active sessions",
	})
	if ok && authScore >= subtaskScore {
		t.Fatalf("expected new-subtask score %d to outrank auth-access score %d for ns", subtaskScore, authScore)
	}
}

// TestModelCommandPaletteHighlightColorApplies verifies highlight-color command updates focused-row styling.
func TestModelCommandPaletteHighlightColorApplies(t *testing.T) {
	now := time.Date(2026, 2, 23, 11, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Styled task",
		Priority:  domain.PriorityMedium,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})
	m := loadReadyModel(t, NewModel(svc))

	updated, cmd := m.executeCommandPalette("highlight-color")
	m = applyResult(t, updated, cmd)
	if m.mode != modeHighlightColor {
		t.Fatalf("expected highlight-color modal mode, got %v", m.mode)
	}
	m.highlightColorInput.SetValue("201")
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})

	if got := m.highlightColor; got != "201" {
		t.Fatalf("expected stored highlight color 201, got %q", got)
	}
	rendered := fmt.Sprint(m.View().Content)
	if !strings.Contains(rendered, "38;5;201") {
		t.Fatalf("expected focused row rendered with ansi color 201, got\n%s", rendered)
	}
}

// TestModelLabelsConfigCommandSave verifies labels-config command flow updates runtime labels and calls persistence callback.
func TestModelLabelsConfigCommandSave(t *testing.T) {
	now := time.Date(2026, 2, 23, 9, 30, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, nil)

	saveCalls := 0
	var savedSlug string
	var savedGlobal []string
	var savedProject []string
	m := loadReadyModel(t, NewModel(
		svc,
		WithLabelConfig(LabelConfig{
			Global:   []string{"bug"},
			Projects: map[string][]string{"inbox": {"till"}},
		}),
		WithSaveLabelsConfigCallback(func(projectSlug string, globalLabels, projectLabels []string) error {
			saveCalls++
			savedSlug = projectSlug
			savedGlobal = append([]string(nil), globalLabels...)
			savedProject = append([]string(nil), projectLabels...)
			return nil
		}),
	))

	updated, cmd := m.executeCommandPalette("labels-config")
	m = applyResult(t, updated, cmd)
	if m.mode != modeLabelsConfig {
		t.Fatalf("expected labels config mode, got %v", m.mode)
	}
	if got := m.labelsConfigSlug; got != "inbox" {
		t.Fatalf("expected labels config slug inbox, got %q", got)
	}

	m.labelsConfigInputs[0].SetValue("Bug, chore, bug")
	m.labelsConfigInputs[1].SetValue("Roadmap, till, roadmap")
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})

	if saveCalls != 1 {
		t.Fatalf("expected one labels save call, got %d", saveCalls)
	}
	if savedSlug != "inbox" {
		t.Fatalf("expected labels save slug inbox, got %q", savedSlug)
	}
	wantGlobal := []string{"bug", "chore"}
	if len(savedGlobal) != len(wantGlobal) {
		t.Fatalf("unexpected saved global labels %#v", savedGlobal)
	}
	for i := range wantGlobal {
		if savedGlobal[i] != wantGlobal[i] {
			t.Fatalf("unexpected saved global label at %d: got %q want %q", i, savedGlobal[i], wantGlobal[i])
		}
	}
	wantProject := []string{"roadmap", "till"}
	if len(savedProject) != len(wantProject) {
		t.Fatalf("unexpected saved project labels %#v", savedProject)
	}
	for i := range wantProject {
		if savedProject[i] != wantProject[i] {
			t.Fatalf("unexpected saved project label at %d: got %q want %q", i, savedProject[i], wantProject[i])
		}
	}
	if m.mode != modeNone {
		t.Fatalf("expected modal closed after save, got %v", m.mode)
	}
	if !strings.Contains(m.status, "labels config saved") {
		t.Fatalf("expected labels save status, got %q", m.status)
	}
	if len(m.allowedLabelGlobal) != len(wantGlobal) {
		t.Fatalf("unexpected in-memory global labels %#v", m.allowedLabelGlobal)
	}
	for i := range wantGlobal {
		if m.allowedLabelGlobal[i] != wantGlobal[i] {
			t.Fatalf("unexpected in-memory global label at %d: got %q want %q", i, m.allowedLabelGlobal[i], wantGlobal[i])
		}
	}
	projectLabels := m.allowedLabelProject["inbox"]
	if len(projectLabels) != len(wantProject) {
		t.Fatalf("unexpected in-memory project labels %#v", m.allowedLabelProject)
	}
	for i := range wantProject {
		if projectLabels[i] != wantProject[i] {
			t.Fatalf("unexpected in-memory project label at %d: got %q want %q", i, projectLabels[i], wantProject[i])
		}
	}
}

// TestModelLabelsConfigCommandSaveScopedBranchPhase verifies branch/phase labels persist through scoped labels-config saves.
func TestModelLabelsConfigCommandSaveScopedBranchPhase(t *testing.T) {
	now := time.Date(2026, 2, 23, 9, 30, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	branch, _ := domain.NewTask(domain.TaskInput{
		ID:        "b1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		ParentID:  "",
		Kind:      domain.WorkKind("branch"),
		Scope:     domain.KindAppliesToBranch,
		Title:     "Branch",
		Priority:  domain.PriorityMedium,
		Labels:    []string{"branch-old"},
	}, now)
	phase, _ := domain.NewTask(domain.TaskInput{
		ID:        "ph1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  1,
		ParentID:  branch.ID,
		Kind:      domain.WorkKindPhase,
		Scope:     domain.KindAppliesToPhase,
		Title:     "Phase",
		Priority:  domain.PriorityMedium,
		Labels:    []string{"phase-old"},
	}, now)
	leaf, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  2,
		ParentID:  phase.ID,
		Kind:      domain.WorkKindTask,
		Scope:     domain.KindAppliesToTask,
		Title:     "Leaf",
		Priority:  domain.PriorityMedium,
	}, now)

	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{branch, phase, leaf})
	m := loadReadyModel(t, NewModel(
		svc,
		WithLabelConfig(LabelConfig{
			Global:   []string{"bug"},
			Projects: map[string][]string{"inbox": {"roadmap"}},
		}),
		WithSaveLabelsConfigCallback(func(_ string, _ []string, _ []string) error { return nil }),
	))
	m.projectionRootTaskID = branch.ID
	m.selectedColumn = 0
	m.selectedTask = 0

	updated, cmd := m.executeCommandPalette("labels-config")
	m = applyResult(t, updated, cmd)
	if m.mode != modeLabelsConfig {
		t.Fatalf("expected labels config mode, got %v", m.mode)
	}
	if len(m.labelsConfigInputs) != 4 {
		t.Fatalf("expected 4 labels config inputs, got %d", len(m.labelsConfigInputs))
	}
	if m.labelsConfigBranchTaskID != branch.ID || m.labelsConfigPhaseTaskID != phase.ID {
		t.Fatalf("expected branch/phase context IDs, got branch=%q phase=%q", m.labelsConfigBranchTaskID, m.labelsConfigPhaseTaskID)
	}
	if got := strings.TrimSpace(m.labelsConfigInputs[2].Value()); got != "branch-old" {
		t.Fatalf("expected branch labels prefill, got %q", got)
	}
	if got := strings.TrimSpace(m.labelsConfigInputs[3].Value()); got != "phase-old" {
		t.Fatalf("expected phase labels prefill, got %q", got)
	}

	m.labelsConfigInputs[2].SetValue("branch-new")
	m.labelsConfigInputs[3].SetValue("phase-new,phase-two")
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})

	findTask := func(taskID string) (domain.Task, bool) {
		for _, task := range svc.tasks[p.ID] {
			if task.ID == taskID {
				return task, true
			}
		}
		return domain.Task{}, false
	}
	updatedBranch, ok := findTask(branch.ID)
	if !ok {
		t.Fatalf("expected branch task %q to exist", branch.ID)
	}
	if len(updatedBranch.Labels) != 1 || updatedBranch.Labels[0] != "branch-new" {
		t.Fatalf("expected branch labels updated, got %#v", updatedBranch.Labels)
	}
	updatedPhase, ok := findTask(phase.ID)
	if !ok {
		t.Fatalf("expected phase task %q to exist", phase.ID)
	}
	if len(updatedPhase.Labels) != 2 || updatedPhase.Labels[0] != "phase-new" || updatedPhase.Labels[1] != "phase-two" {
		t.Fatalf("expected phase labels updated, got %#v", updatedPhase.Labels)
	}
}

// TestModelCommandPaletteProjectLifecycleActions verifies archive/restore/delete project command flows.
func TestModelCommandPaletteProjectLifecycleActions(t *testing.T) {
	now := time.Date(2026, 2, 23, 9, 30, 0, 0, time.UTC)
	p1, _ := domain.NewProject("p1", "Inbox", "", now)
	p2, _ := domain.NewProject("p2", "Roadmap", "", now)
	c1, _ := domain.NewColumn("c1", p1.ID, "To Do", 0, 0, now)
	c2, _ := domain.NewColumn("c2", p2.ID, "To Do", 0, 0, now)
	svc := newFakeService([]domain.Project{p1, p2}, []domain.Column{c1, c2}, nil)
	m := loadReadyModel(t, NewModel(
		svc,
		WithConfirmConfig(ConfirmConfig{
			Archive:    false,
			HardDelete: false,
			Restore:    false,
		}),
	))
	m.showArchivedProjects = true

	updated, cmd := m.executeCommandPalette("toggle-selection-mode")
	m = applyResult(t, updated, cmd)
	if !m.mouseSelectionMode {
		t.Fatal("expected selection mode enabled")
	}

	updated, cmd = m.executeCommandPalette("archive-project")
	m = applyResult(t, updated, cmd)
	if svc.projects[0].ArchivedAt == nil {
		t.Fatalf("expected archived project state, got %#v", svc.projects[0])
	}

	updated, cmd = m.executeCommandPalette("restore-project")
	m = applyResult(t, updated, cmd)
	if svc.projects[0].ArchivedAt != nil {
		t.Fatalf("expected restored project state, got %#v", svc.projects[0])
	}

	updated, cmd = m.executeCommandPalette("delete-project")
	m = applyResult(t, updated, cmd)
	if len(svc.projects) != 1 {
		t.Fatalf("expected one project after delete, got %d", len(svc.projects))
	}
	if svc.projects[0].ID != p2.ID {
		t.Fatalf("expected remaining project %q, got %#v", p2.ID, svc.projects)
	}
}

// TestModelProjectLifecycleConfirmBranches verifies confirm-mode branches for project lifecycle actions.
func TestModelProjectLifecycleConfirmBranches(t *testing.T) {
	now := time.Date(2026, 2, 23, 9, 30, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, nil)
	m := loadReadyModel(t, NewModel(svc))

	updated, cmd := m.archiveCurrentProject(true)
	m = applyResult(t, updated, cmd)
	if m.mode != modeConfirmAction || m.pendingConfirm.Kind != "archive-project" {
		t.Fatalf("expected archive-project confirm mode, got mode=%v confirm=%#v", m.mode, m.pendingConfirm)
	}
	m.mode = modeNone
	m.pendingConfirm = confirmAction{}

	archived, err := svc.ArchiveProject(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("ArchiveProject() setup error = %v", err)
	}
	svc.projects[0] = archived
	m.projects[0] = archived
	m.showArchivedProjects = true

	updated, cmd = m.restoreCurrentProject(true)
	m = applyResult(t, updated, cmd)
	if m.mode != modeConfirmAction || m.pendingConfirm.Kind != "restore-project" {
		t.Fatalf("expected restore-project confirm mode, got mode=%v confirm=%#v", m.mode, m.pendingConfirm)
	}
	m.mode = modeNone

	updated, cmd = m.deleteCurrentProject(true)
	m = applyResult(t, updated, cmd)
	if m.mode != modeConfirmAction || m.pendingConfirm.Kind != "delete-project" {
		t.Fatalf("expected delete-project confirm mode, got mode=%v confirm=%#v", m.mode, m.pendingConfirm)
	}
}

// TestModelProjectLifecycleGuardsAndSelection verifies project lifecycle guard statuses and next-visible selection.
func TestModelProjectLifecycleGuardsAndSelection(t *testing.T) {
	now := time.Date(2026, 2, 23, 9, 30, 0, 0, time.UTC)

	empty := loadReadyModel(t, NewModel(newFakeService(nil, nil, nil)))
	updated, cmd := empty.archiveCurrentProject(true)
	empty = applyResult(t, updated, cmd)
	if empty.status != "no project selected" {
		t.Fatalf("expected no-project archive status, got %q", empty.status)
	}
	updated, cmd = empty.restoreCurrentProject(true)
	empty = applyResult(t, updated, cmd)
	if empty.status != "no project selected" {
		t.Fatalf("expected no-project restore status, got %q", empty.status)
	}
	updated, cmd = empty.deleteCurrentProject(true)
	empty = applyResult(t, updated, cmd)
	if empty.status != "no project selected" {
		t.Fatalf("expected no-project delete status, got %q", empty.status)
	}

	p1, _ := domain.NewProject("p1", "Inbox", "", now)
	p2, _ := domain.NewProject("p2", "Roadmap", "", now)
	p3, _ := domain.NewProject("p3", "Ops", "", now)
	p2.Archive(now.Add(time.Minute))

	c1, _ := domain.NewColumn("c1", p1.ID, "To Do", 0, 0, now)
	c2, _ := domain.NewColumn("c2", p2.ID, "To Do", 0, 0, now)
	c3, _ := domain.NewColumn("c3", p3.ID, "To Do", 0, 0, now)

	svc := newFakeService([]domain.Project{p1, p2, p3}, []domain.Column{c1, c2, c3}, nil)
	m := loadReadyModel(t, NewModel(svc))
	m.projects[1] = p2
	m.showArchivedProjects = false

	updated, cmd = m.restoreCurrentProject(true)
	m = applyResult(t, updated, cmd)
	if m.status != "project is not archived" {
		t.Fatalf("expected non-archived restore guard status, got %q", m.status)
	}
	if m.mode == modeConfirmAction {
		t.Fatalf("expected no confirm mode for non-archived restore, got %v", m.mode)
	}

	updated, cmd = m.archiveCurrentProject(false)
	m = applyResult(t, updated, cmd)
	if got := m.projects[m.selectedProject].ID; got != p3.ID {
		t.Fatalf("expected next visible project %q after archive, got %q", p3.ID, got)
	}
	m.selectedProject = 1
	updated, cmd = m.archiveCurrentProject(true)
	m = applyResult(t, updated, cmd)
	if m.status != "project already archived" {
		t.Fatalf("expected already-archived guard status, got %q", m.status)
	}

	updated, cmd = m.deleteCurrentProject(false)
	m = applyResult(t, updated, cmd)
	if got := m.projects[m.selectedProject].ID; got != p3.ID {
		t.Fatalf("expected delete to skip archived candidates and keep visible project %q, got %q", p3.ID, got)
	}

	updated, cmd = m.applyConfirmedAction(confirmAction{Kind: "mystery"})
	m = applyResult(t, updated, cmd)
	if m.status != "unknown confirm action" {
		t.Fatalf("expected unknown confirm action status, got %q", m.status)
	}
}

// TestModelCommandPaletteBranchLifecycleGuards verifies branch lifecycle commands require a selected branch.
func TestModelCommandPaletteBranchLifecycleGuards(t *testing.T) {
	now := time.Date(2026, 2, 23, 9, 30, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Kind:      domain.WorkKindTask,
		Scope:     domain.KindAppliesToTask,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)

	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})))

	updated, cmd := m.executeCommandPalette("edit-branch")
	m = applyResult(t, updated, cmd)
	if m.status != "select a branch to edit" {
		t.Fatalf("expected edit-branch guard status, got %q", m.status)
	}

	updated, cmd = m.executeCommandPalette("archive-branch")
	m = applyResult(t, updated, cmd)
	if m.status != "select a branch to archive" {
		t.Fatalf("expected archive-branch guard status, got %q", m.status)
	}

	updated, cmd = m.executeCommandPalette("delete-branch")
	m = applyResult(t, updated, cmd)
	if m.status != "select a branch to delete" {
		t.Fatalf("expected delete-branch guard status, got %q", m.status)
	}

	updated, cmd = m.executeCommandPalette("restore-branch")
	m = applyResult(t, updated, cmd)
	if m.status != "select an archived branch to restore" {
		t.Fatalf("expected restore-branch guard status, got %q", m.status)
	}
}

// TestModelCommandPaletteBranchLifecycleActions verifies branch command palette flows.
func TestModelCommandPaletteBranchLifecycleActions(t *testing.T) {
	now := time.Date(2026, 2, 23, 9, 30, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	branch, _ := domain.NewTask(domain.TaskInput{
		ID:        "b1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Kind:      domain.WorkKind("branch"),
		Scope:     domain.KindAppliesToBranch,
		Title:     "Branch",
		Priority:  domain.PriorityMedium,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{branch})
	m := loadReadyModel(t, NewModel(
		svc,
		WithConfirmConfig(ConfirmConfig{
			Archive:    false,
			HardDelete: false,
			Restore:    false,
		}),
	))
	m.showArchived = true

	updated, cmd := m.executeCommandPalette("new-branch")
	m = applyResult(t, updated, cmd)
	if m.mode != modeAddTask || m.taskFormKind != domain.WorkKind("branch") || m.taskFormScope != domain.KindAppliesToBranch {
		t.Fatalf("expected new branch task form defaults, got mode=%v kind=%q scope=%q", m.mode, m.taskFormKind, m.taskFormScope)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})

	updated, cmd = m.executeCommandPalette("edit-branch")
	m = applyResult(t, updated, cmd)
	if m.mode != modeEditTask || strings.TrimSpace(m.editingTaskID) != branch.ID {
		t.Fatalf("expected branch edit form, got mode=%v task=%q", m.mode, m.editingTaskID)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})

	updated, cmd = m.executeCommandPalette("archive-branch")
	m = applyResult(t, updated, cmd)
	if len(svc.tasks[p.ID]) != 1 || svc.tasks[p.ID][0].ArchivedAt == nil {
		t.Fatalf("expected archived branch task, got %#v", svc.tasks[p.ID])
	}

	updated, cmd = m.executeCommandPalette("restore-branch")
	m = applyResult(t, updated, cmd)
	if len(svc.tasks[p.ID]) != 1 || svc.tasks[p.ID][0].ArchivedAt != nil {
		t.Fatalf("expected restored branch task, got %#v", svc.tasks[p.ID])
	}

	updated, cmd = m.executeCommandPalette("delete-branch")
	m = applyResult(t, updated, cmd)
	if len(svc.tasks[p.ID]) != 0 {
		t.Fatalf("expected branch hard delete, got %#v", svc.tasks[p.ID])
	}
}

// TestModelCommandPaletteNewBranchWarnsWhenFocused verifies branch creation is blocked while subtree focus is active.
func TestModelCommandPaletteNewBranchWarnsWhenFocused(t *testing.T) {
	now := time.Date(2026, 2, 23, 9, 35, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	branch, _ := domain.NewTask(domain.TaskInput{
		ID:        "b1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Kind:      domain.WorkKind("branch"),
		Scope:     domain.KindAppliesToBranch,
		Title:     "Branch",
		Priority:  domain.PriorityMedium,
	}, now)
	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{branch})))

	m.focusTaskByID(branch.ID)
	m = applyMsg(t, m, keyRune('f'))
	if m.projectionRootTaskID != branch.ID {
		t.Fatalf("expected focused branch root %q, got %q", branch.ID, m.projectionRootTaskID)
	}

	updated, cmd := m.executeCommandPalette("new-branch")
	m = applyResult(t, updated, cmd)
	if m.mode != modeWarning {
		t.Fatalf("expected warning modal mode, got %v", m.mode)
	}
	if m.status != "clear focus before creating a branch" {
		t.Fatalf("expected focused-branch warning status, got %q", m.status)
	}
	warning := m.renderModeOverlay(lipgloss.Color("62"), lipgloss.Color("241"), lipgloss.Color("239"), lipgloss.NewStyle(), 96)
	if !strings.Contains(warning, "Branch Creation Blocked") {
		t.Fatalf("expected warning title in modal, got %q", warning)
	}
	if !strings.Contains(warning, "project-level items") {
		t.Fatalf("expected warning guidance in modal, got %q", warning)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeNone {
		t.Fatalf("expected enter to dismiss warning modal, got %v", m.mode)
	}
	if m.projectionRootTaskID != branch.ID {
		t.Fatalf("expected subtree focus to stay active after dismissing warning, got %q", m.projectionRootTaskID)
	}
}

// TestModelCommandPalettePhaseCreationDefaultsToProjectLevel verifies new-phase works without branch context.
func TestModelCommandPalettePhaseCreationDefaultsToProjectLevel(t *testing.T) {
	now := time.Date(2026, 2, 23, 9, 45, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Kind:      domain.WorkKindTask,
		Scope:     domain.KindAppliesToTask,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})))

	updated, cmd := m.executeCommandPalette("new-phase")
	m = applyResult(t, updated, cmd)
	if m.mode != modeAddTask {
		t.Fatalf("expected project-level new-phase to open add-task mode, got mode=%v status=%q", m.mode, m.status)
	}
	if m.taskFormParentID != "" || m.taskFormKind != domain.WorkKindPhase || m.taskFormScope != domain.KindAppliesToPhase {
		t.Fatalf("expected project-level phase defaults, got parent=%q kind=%q scope=%q", m.taskFormParentID, m.taskFormKind, m.taskFormScope)
	}
}

// TestModelCommandPaletteTypedNormalizedProjectPhaseCreation verifies the real palette update path opens project-level phase creation for normalized raw input.
func TestModelCommandPaletteTypedNormalizedProjectPhaseCreation(t *testing.T) {
	now := time.Date(2026, 2, 23, 9, 50, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})))

	m = applyMsg(t, m, keyRune(':'))
	for _, r := range "new_phase" {
		m = applyMsg(t, m, keyRune(r))
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeAddTask {
		t.Fatalf("expected typed new_phase to open add-task mode, got mode=%v status=%q", m.mode, m.status)
	}
	if m.taskFormParentID != "" || m.taskFormKind != domain.WorkKindPhase || m.taskFormScope != domain.KindAppliesToPhase {
		t.Fatalf("expected typed new_phase to create project-level phase defaults, got parent=%q kind=%q scope=%q", m.taskFormParentID, m.taskFormKind, m.taskFormScope)
	}
}

// TestModelCommandPalettePhaseCreationActions verifies new-phase derives parentage from the active focus screen only.
func TestModelCommandPalettePhaseCreationActions(t *testing.T) {
	now := time.Date(2026, 2, 23, 10, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Roadmap", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	branch, _ := domain.NewTask(domain.TaskInput{
		ID:        "b1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Kind:      domain.WorkKind("branch"),
		Scope:     domain.KindAppliesToBranch,
		Title:     "Branch",
		Priority:  domain.PriorityMedium,
	}, now)
	phase, _ := domain.NewTask(domain.TaskInput{
		ID:        "ph1",
		ProjectID: p.ID,
		ParentID:  branch.ID,
		ColumnID:  c.ID,
		Position:  1,
		Kind:      domain.WorkKindPhase,
		Scope:     domain.KindAppliesToPhase,
		Title:     "Phase",
		Priority:  domain.PriorityMedium,
	}, now)
	phaseLeaf, _ := domain.NewTask(domain.TaskInput{
		ID:        "ph-empty",
		ProjectID: p.ID,
		ParentID:  branch.ID,
		ColumnID:  c.ID,
		Position:  2,
		Kind:      domain.WorkKindPhase,
		Scope:     domain.KindAppliesToPhase,
		Title:     "Phase Leaf",
		Priority:  domain.PriorityLow,
	}, now)
	branchLeaf, _ := domain.NewTask(domain.TaskInput{
		ID:        "b-empty",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  3,
		Kind:      domain.WorkKind("branch"),
		Scope:     domain.KindAppliesToBranch,
		Title:     "Branch Leaf",
		Priority:  domain.PriorityLow,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{branch, phase, phaseLeaf, branchLeaf})
	m := loadReadyModel(t, NewModel(svc))

	m.focusTaskByID(branch.ID)
	updated, cmd := m.executeCommandPalette("new-phase")
	m = applyResult(t, updated, cmd)
	if m.mode != modeAddTask {
		t.Fatalf("expected add-task mode for new phase, got %v", m.mode)
	}
	if m.taskFormParentID != "" || m.taskFormKind != domain.WorkKindPhase || m.taskFormScope != domain.KindAppliesToPhase {
		t.Fatalf("expected project-level phase defaults when only a child row is selected, got parent=%q kind=%q scope=%q", m.taskFormParentID, m.taskFormKind, m.taskFormScope)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})

	m.focusTaskByID(branchLeaf.ID)
	m = applyMsg(t, m, keyRune('f'))
	if m.projectionRootTaskID != branchLeaf.ID {
		t.Fatalf("expected empty branch focus root %q, got %q", branchLeaf.ID, m.projectionRootTaskID)
	}
	updated, cmd = m.executeCommandPalette("new-phase")
	m = applyResult(t, updated, cmd)
	if m.mode != modeAddTask {
		t.Fatalf("expected add-task mode for focused-empty new phase, got %v", m.mode)
	}
	if m.taskFormParentID != branchLeaf.ID || m.taskFormKind != domain.WorkKindPhase || m.taskFormScope != domain.KindAppliesToPhase {
		t.Fatalf("expected focused-empty-branch phase defaults, got parent=%q kind=%q scope=%q", m.taskFormParentID, m.taskFormKind, m.taskFormScope)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if !m.clearSubtreeFocus() {
		t.Fatal("expected to clear focused subtree after branch-leaf phase test")
	}

	m.focusTaskByID(phase.ID)
	updated, cmd = m.executeCommandPalette("new-phase")
	m = applyResult(t, updated, cmd)
	if m.mode != modeAddTask {
		t.Fatalf("expected add-task mode for project-screen new phase with selected phase row, got mode=%v status=%q", m.mode, m.status)
	}
	if m.taskFormParentID != "" || m.taskFormKind != domain.WorkKindPhase || m.taskFormScope != domain.KindAppliesToPhase {
		t.Fatalf("expected project-level phase defaults when only a phase row is selected, got parent=%q kind=%q scope=%q", m.taskFormParentID, m.taskFormKind, m.taskFormScope)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})

	m.focusTaskByID(branch.ID)
	m = applyMsg(t, m, keyRune('f'))
	if m.projectionRootTaskID != branch.ID {
		t.Fatalf("expected focused branch root %q for nested phase creation, got %q", branch.ID, m.projectionRootTaskID)
	}
	m.focusTaskByID(phase.ID)
	updated, cmd = m.executeCommandPalette("new-phase")
	m = applyResult(t, updated, cmd)
	if m.mode != modeAddTask {
		t.Fatalf("expected add-task mode for branch-screen new phase, got mode=%v status=%q", m.mode, m.status)
	}
	if m.taskFormParentID != branch.ID || m.taskFormKind != domain.WorkKindPhase || m.taskFormScope != domain.KindAppliesToPhase {
		t.Fatalf("expected focused-branch phase defaults even when a child phase row is selected, got parent=%q kind=%q scope=%q", m.taskFormParentID, m.taskFormKind, m.taskFormScope)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})

	m.focusTaskByID(phaseLeaf.ID)
	m = applyMsg(t, m, keyRune('f'))
	if m.projectionRootTaskID != phaseLeaf.ID {
		t.Fatalf("expected empty phase focus root %q, got %q", phaseLeaf.ID, m.projectionRootTaskID)
	}
	updated, cmd = m.executeCommandPalette("new-phase")
	m = applyResult(t, updated, cmd)
	if m.mode != modeAddTask {
		t.Fatalf("expected add-task mode for focused-empty nested new phase, got %v", m.mode)
	}
	if m.taskFormParentID != phaseLeaf.ID || m.taskFormKind != domain.WorkKindPhase || m.taskFormScope != domain.KindAppliesToPhase {
		t.Fatalf("expected focused-empty-phase nested phase defaults, got parent=%q kind=%q scope=%q", m.taskFormParentID, m.taskFormKind, m.taskFormScope)
	}
}

// TestModelCommandPalettePhaseCreationAcceptsNormalizedCommandIDs verifies typed underscore/space variants map to the same command ids.
func TestModelCommandPalettePhaseCreationAcceptsNormalizedCommandIDs(t *testing.T) {
	now := time.Date(2026, 2, 23, 10, 15, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Roadmap", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	branch, _ := domain.NewTask(domain.TaskInput{
		ID:        "b1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Kind:      domain.WorkKind("branch"),
		Scope:     domain.KindAppliesToBranch,
		Title:     "Branch",
		Priority:  domain.PriorityMedium,
	}, now)

	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{branch})))
	m.focusTaskByID(branch.ID)
	m = applyMsg(t, m, keyRune('f'))

	for _, raw := range []string{"new_phase", "new phase"} {
		updated, cmd := m.executeCommandPalette(raw)
		next := applyResult(t, updated, cmd)
		if next.mode != modeAddTask {
			t.Fatalf("raw command %q should open add-task phase form, got mode=%v status=%q", raw, next.mode, next.status)
		}
		if next.taskFormParentID != branch.ID || next.taskFormKind != domain.WorkKindPhase || next.taskFormScope != domain.KindAppliesToPhase {
			t.Fatalf("raw command %q should preserve phase defaults, got parent=%q kind=%q scope=%q", raw, next.taskFormParentID, next.taskFormKind, next.taskFormScope)
		}
	}
}

// TestModelCommandPalettePhaseCreationBlocksTaskFocusedScreens verifies task and subtask screens cannot parent phases.
func TestModelCommandPalettePhaseCreationBlocksTaskFocusedScreens(t *testing.T) {
	now := time.Date(2026, 2, 23, 10, 20, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Roadmap", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Kind:      domain.WorkKindTask,
		Scope:     domain.KindAppliesToTask,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	subtask, _ := domain.NewTask(domain.TaskInput{
		ID:        "st1",
		ProjectID: p.ID,
		ParentID:  task.ID,
		ColumnID:  c.ID,
		Position:  1,
		Kind:      domain.WorkKindSubtask,
		Scope:     domain.KindAppliesToSubtask,
		Title:     "Subtask",
		Priority:  domain.PriorityLow,
	}, now)
	for _, tc := range []struct {
		name        string
		enterScreen func(Model) Model
		wantRootID  string
	}{
		{
			name: "task",
			enterScreen: func(in Model) Model {
				in.focusTaskByID(task.ID)
				return applyMsg(t, in, keyRune('f'))
			},
			wantRootID: task.ID,
		},
		{
			name: "subtask",
			enterScreen: func(in Model) Model {
				in.focusTaskByID(task.ID)
				in = applyMsg(t, in, keyRune('f'))
				return applyMsg(t, in, keyRune('f'))
			},
			wantRootID: subtask.ID,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			current := loadReadyModel(t, NewModel(newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task, subtask})))
			current = tc.enterScreen(current)
			if current.projectionRootTaskID != tc.wantRootID {
				t.Fatalf("expected focused %s root %q, got %q", tc.name, tc.wantRootID, current.projectionRootTaskID)
			}

			updated, cmd := current.executeCommandPalette("new-phase")
			current = applyResult(t, updated, cmd)
			if current.mode != modeWarning {
				t.Fatalf("expected warning modal mode for %s-focused new phase, got %v", tc.name, current.mode)
			}
			if current.status != "phase creation blocked in current focus" {
				t.Fatalf("expected focused-%s warning status, got %q", tc.name, current.status)
			}
			warning := current.renderModeOverlay(lipgloss.Color("62"), lipgloss.Color("241"), lipgloss.Color("239"), lipgloss.NewStyle(), 96)
			if !strings.Contains(warning, "Phase Creation Blocked") {
				t.Fatalf("expected warning title in modal, got %q", warning)
			}
			if !strings.Contains(warning, "project, branch, or phase") || !strings.Contains(warning, "screens.") {
				t.Fatalf("expected warning guidance in modal, got %q", warning)
			}
		})
	}
}

// TestClipboardShortcutHelpers verifies key-detection and input-splice helper behavior for clipboard shortcuts.
func TestClipboardShortcutHelpers(t *testing.T) {
	if !isClipboardCopyKey(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}) {
		t.Fatal("expected ctrl+c to be recognized as copy")
	}
	if !isClipboardPasteKey(tea.KeyPressMsg{Code: 'v', Mod: tea.ModCtrl}) {
		t.Fatal("expected ctrl+v to be recognized as paste")
	}

	merged, next := spliceRunes("abcd", 2, "XYZ")
	if merged != "abXYZcd" || next != 5 {
		t.Fatalf("unexpected splice result merged=%q next=%d", merged, next)
	}

	in := textinput.New()
	in.SetValue("alpha")
	in.SetCursor(2)

	if handled, _ := applyClipboardShortcutToInput(tea.KeyPressMsg{Code: 'x', Text: "x"}, &in); handled {
		t.Fatal("expected non-clipboard key to bypass clipboard helper")
	}
	if handled, status := applyClipboardShortcutToInput(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}, &in); !handled || status == "" {
		t.Fatalf("expected copy shortcut handled with status, handled=%t status=%q", handled, status)
	}
	if handled, status := applyClipboardShortcutToInput(tea.KeyPressMsg{Code: 'v', Mod: tea.ModCtrl}, &in); !handled || status == "" {
		t.Fatalf("expected paste shortcut handled with status, handled=%t status=%q", handled, status)
	}
}

// TestSanitizeFormFieldValueStripsTerminalProbeArtifacts verifies OSC color probe responses are removed from form values.
func TestSanitizeFormFieldValueStripsTerminalProbeArtifacts(t *testing.T) {
	raw := "mcp-visibility-probe-20260302-0502/1e1e/2e2e/1e1e/2e2e]11;rgb:1e1e/1e1e/2e2e"
	got := sanitizeFormFieldValue(raw)
	if got != "mcp-visibility-probe-20260302-0502" {
		t.Fatalf("expected probe artifact stripped, got %q", got)
	}
}

// TestTaskAndProjectFormValuesSanitizeTerminalProbeArtifacts verifies form extraction applies field sanitization.
func TestTaskAndProjectFormValuesSanitizeTerminalProbeArtifacts(t *testing.T) {
	taskInput := textinput.New()
	taskInput.SetValue("task-title]11;rgb:1e1e/1e1e/2e2e")
	projectInput := textinput.New()
	projectInput.SetValue("project-name]11;rgb:1e1e/1e1e/2e2e")

	m := Model{
		formInputs:        []textinput.Model{taskInput},
		projectFormInputs: []textinput.Model{projectInput},
	}
	taskVals := m.taskFormValues()
	if got := taskVals["title"]; got != "task-title" {
		t.Fatalf("expected task title sanitized, got %q", got)
	}
	projectVals := m.projectFormValues()
	if got := projectVals["name"]; got != "project-name" {
		t.Fatalf("expected project name sanitized, got %q", got)
	}
}

// TestScrubTextInputTerminalArtifactsStripsProbeDuringEdit verifies interactive scrub removes leaked probe text before submit.
func TestScrubTextInputTerminalArtifactsStripsProbeDuringEdit(t *testing.T) {
	in := textinput.New()
	in.SetValue("mcp-visibility/1e1e/2e2e/1e1e/2e2e]11;rgb:1e1e/1e1e/2e2e")
	in.SetCursor(len([]rune(in.Value())))

	changed := scrubTextInputTerminalArtifacts(&in)
	if !changed {
		t.Fatal("expected scrubber to report changed value")
	}
	if got := in.Value(); got != "mcp-visibility" {
		t.Fatalf("expected scrubbed input, got %q", got)
	}
}

// TestScrubTextAreaTerminalArtifactsStripsProbeDuringEdit verifies textarea scrub removes leaked probe text.
func TestScrubTextAreaTerminalArtifactsStripsProbeDuringEdit(t *testing.T) {
	ta := textarea.New()
	ta.SetValue("notes]11;rgb:1e1e/1e1e/2e2e\nnext line")

	changed := scrubTextAreaTerminalArtifacts(&ta)
	if !changed {
		t.Fatal("expected textarea scrubber to report changed value")
	}
	if got := ta.Value(); got != "notes\nnext line" {
		t.Fatalf("expected scrubbed textarea, got %q", got)
	}
}

// TestRenderOverviewPanelHeightMatchesRequestedHeight verifies stacked notifications panels do not exceed the requested board height.
func TestRenderOverviewPanelHeightMatchesRequestedHeight(t *testing.T) {
	now := time.Date(2026, 3, 3, 1, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{project}, []domain.Column{column}, []domain.Task{task})))

	panel := m.renderOverviewPanel(project, lipgloss.Color("62"), lipgloss.Color("241"), lipgloss.Color("239"), 44, 14, 0, 0, 0, nil, false)
	if got := lipgloss.Height(panel); got != 14 {
		t.Fatalf("expected overview panel height 14, got %d", got)
	}
}

// TestModelTaskDescriptionEditorFlow verifies task-form description always routes through the markdown editor.
func TestModelTaskDescriptionEditorFlow(t *testing.T) {
	now := time.Date(2026, 3, 3, 9, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{project}, []domain.Column{column}, nil)))

	m = applyMsg(t, m, keyRune('n'))
	if m.mode != modeAddTask {
		t.Fatalf("expected add-task mode, got %v", m.mode)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	if m.formFocus != taskFieldDescription {
		t.Fatalf("expected description field focus, got %d", m.formFocus)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeDescriptionEditor {
		t.Fatalf("expected markdown description editor mode, got %v", m.mode)
	}
	if m.descriptionEditorTarget != descriptionEditorTargetTask {
		t.Fatalf("expected task description editor target, got %v", m.descriptionEditorTarget)
	}

	m.descriptionEditorInput.SetValue("## Task Detail\n\n- line one\n- line two")
	m = applyMsg(t, m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	if m.mode != modeAddTask {
		t.Fatalf("expected return to add-task mode after save, got %v", m.mode)
	}
	if got := m.taskFormDescription; !strings.Contains(got, "Task Detail") {
		t.Fatalf("expected task markdown description persisted, got %q", got)
	}
	if got := m.formInputs[taskFieldDescription].Value(); !strings.Contains(got, "Task Detail") {
		t.Fatalf("expected compact task description summary in form field, got %q", got)
	}
}

// TestModelProjectDescriptionEditorSeedAndCancel verifies project-description editing opens full markdown editor on typed input.
func TestModelProjectDescriptionEditorSeedAndCancel(t *testing.T) {
	now := time.Date(2026, 3, 3, 9, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{project}, []domain.Column{column}, nil)))

	m = applyMsg(t, m, keyRune('N'))
	if m.mode != modeAddProject {
		t.Fatalf("expected add-project mode, got %v", m.mode)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	if m.projectFormFocus != projectFieldDescription {
		t.Fatalf("expected project description field focus, got %d", m.projectFormFocus)
	}

	m = applyMsg(t, m, keyRune('a'))
	if m.mode != modeDescriptionEditor {
		t.Fatalf("expected markdown description editor mode, got %v", m.mode)
	}
	if m.descriptionEditorTarget != descriptionEditorTargetProject {
		t.Fatalf("expected project description editor target, got %v", m.descriptionEditorTarget)
	}
	if got := m.descriptionEditorInput.Value(); got != "a" {
		t.Fatalf("expected seed key to initialize editor content, got %q", got)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.mode != modeAddProject {
		t.Fatalf("expected return to add-project mode after cancel, got %v", m.mode)
	}
	if got := strings.TrimSpace(m.projectFormDescription); got != "" {
		t.Fatalf("expected cancelled editor to keep project description unchanged, got %q", got)
	}
}

// TestModelDescriptionEditorEditModeQuestionMarkInsertsText verifies edit mode captures '?' as text instead of toggling help.
func TestModelDescriptionEditorEditModeQuestionMarkInsertsText(t *testing.T) {
	now := time.Date(2026, 3, 3, 9, 5, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{project}, []domain.Column{column}, nil)))

	m = applyMsg(t, m, keyRune('n'))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeDescriptionEditor {
		t.Fatalf("expected description editor mode, got %v", m.mode)
	}
	if m.descriptionEditorMode != descriptionEditorViewModeEdit {
		t.Fatalf("expected edit submode, got %v", m.descriptionEditorMode)
	}

	m = applyMsg(t, m, keyRune('?'))
	if m.help.ShowAll {
		t.Fatal("expected help overlay to stay closed in description editor edit mode")
	}
	if got := m.descriptionEditorInput.Value(); got != "?" {
		t.Fatalf("expected '?' inserted into editor input, got %q", got)
	}
}

// TestModelDescriptionEditorCtrlUndoRedo verifies ctrl+z / ctrl+shift+z text undo/redo in description editor edit mode.
func TestModelDescriptionEditorCtrlUndoRedo(t *testing.T) {
	now := time.Date(2026, 3, 3, 9, 5, 30, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{project}, []domain.Column{column}, nil)))

	m = applyMsg(t, m, keyRune('n'))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeDescriptionEditor {
		t.Fatalf("expected description editor mode, got %v", m.mode)
	}
	if m.descriptionEditorMode != descriptionEditorViewModeEdit {
		t.Fatalf("expected description editor edit mode, got %v", m.descriptionEditorMode)
	}

	m = applyMsg(t, m, keyRune('a'))
	m = applyMsg(t, m, keyRune('b'))
	m = applyMsg(t, m, keyRune('c'))
	if got := m.descriptionEditorInput.Value(); got != "abc" {
		t.Fatalf("expected editor value %q, got %q", "abc", got)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl})
	if got := m.descriptionEditorInput.Value(); got != "ab" {
		t.Fatalf("expected ctrl+z to undo editor text to %q, got %q", "ab", got)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl | tea.ModShift})
	if got := m.descriptionEditorInput.Value(); got != "abc" {
		t.Fatalf("expected ctrl+shift+z to redo editor text to %q, got %q", "abc", got)
	}
}

// TestModelDescriptionEditorPreviewModeToggleAndScrollSync verifies preview mode toggle, heading text, and synced scroll offsets.
func TestModelDescriptionEditorPreviewModeToggleAndScrollSync(t *testing.T) {
	now := time.Date(2026, 3, 3, 9, 6, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{project}, []domain.Column{column}, nil)))

	m = applyMsg(t, m, keyRune('n'))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeDescriptionEditor {
		t.Fatalf("expected description editor mode, got %v", m.mode)
	}

	lines := make([]string, 0, 80)
	for idx := 0; idx < 80; idx++ {
		lines = append(lines, fmt.Sprintf("line %02d", idx+1))
	}
	m.descriptionEditorInput.SetValue(strings.Join(lines, "\n"))
	m.descriptionEditorInput.SetWidth(24)
	m.descriptionEditorInput.SetHeight(4)
	m.descriptionEditorInput.MoveToBegin()
	m.syncDescriptionPreviewOffsetToEditor()

	for idx := 0; idx < 60; idx++ {
		m.descriptionEditorInput.CursorDown()
	}
	m.syncDescriptionPreviewOffsetToEditor()
	if got, want := m.descriptionPreview.YOffset(), m.descriptionEditorInput.ScrollYOffset(); got != want {
		t.Fatalf("expected preview y offset %d to match editor offset %d", got, want)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	if m.descriptionEditorMode != descriptionEditorViewModePreview {
		t.Fatalf("expected preview submode after tab, got %v", m.descriptionEditorMode)
	}
	if got := m.descriptionPreview.YOffset(); got != 0 {
		t.Fatalf("expected preview submode to open from top, got y offset=%d", got)
	}
	rendered := stripANSI(fmt.Sprint(m.View().Content))
	if !strings.Contains(rendered, "Description Editor") {
		t.Fatalf("expected full-screen description editor header, got %q", rendered)
	}
	if !strings.Contains(rendered, "path:") {
		t.Fatalf("expected path line in description editor header block, got %q", rendered)
	}
	if strings.Contains(rendered, "Preview (Glamour)") {
		t.Fatalf("expected preview heading without glamour suffix, got %q", rendered)
	}

	before := m.descriptionEditorInput.ScrollYOffset()
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyPgDown})
	after := m.descriptionEditorInput.ScrollYOffset()
	if after != before {
		t.Fatalf(
			"expected preview-mode scrolling to leave editor offset unchanged, before=%d after=%d",
			before,
			after,
		)
	}

	m = applyMsg(t, m, keyRune('?'))
	if !m.help.ShowAll {
		t.Fatal("expected help overlay toggle to remain available in preview submode")
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.help.ShowAll {
		t.Fatal("expected esc to close preview-mode help overlay")
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	if m.descriptionEditorMode != descriptionEditorViewModeEdit {
		t.Fatalf("expected return to edit submode after tab, got %v", m.descriptionEditorMode)
	}
}

// TestModelDescriptionEditorPreviewModeScrollsWrappedContent verifies preview mode scroll input works for wrapped markdown.
func TestModelDescriptionEditorPreviewModeScrollsWrappedContent(t *testing.T) {
	now := time.Date(2026, 3, 3, 9, 7, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{project}, []domain.Column{column}, nil)))

	m = applyMsg(t, m, keyRune('n'))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeDescriptionEditor {
		t.Fatalf("expected description editor mode, got %v", m.mode)
	}

	m.descriptionEditorInput.SetValue(strings.Repeat("wrapped preview paragraph ", 600))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	if m.descriptionEditorMode != descriptionEditorViewModePreview {
		t.Fatalf("expected preview submode after tab, got %v", m.descriptionEditorMode)
	}

	beforeKey := m.descriptionPreview.YOffset()
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyPgDown})
	afterKey := m.descriptionPreview.YOffset()
	if afterKey <= beforeKey {
		t.Fatalf("expected preview y offset to increase after pgdown, before=%d after=%d", beforeKey, afterKey)
	}

	beforeMouse := m.descriptionPreview.YOffset()
	m = applyMsg(t, m, tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	afterMouse := m.descriptionPreview.YOffset()
	if afterMouse <= beforeMouse {
		t.Fatalf("expected preview y offset to increase after mouse wheel, before=%d after=%d", beforeMouse, afterMouse)
	}
}

// TestModelTaskInfoDescriptionEditorOpensInPreviewMode verifies task-info opens full-screen details in preview submode.
func TestModelTaskInfoDescriptionEditorOpensInPreviewMode(t *testing.T) {
	now := time.Date(2026, 3, 3, 12, 55, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:          "t1",
		ProjectID:   project.ID,
		ColumnID:    column.ID,
		Position:    0,
		Kind:        domain.WorkKindTask,
		Title:       "Task with details",
		Description: "## full page md\n\n- line one\n- line two",
		Priority:    domain.PriorityMedium,
	}, now)
	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{project}, []domain.Column{column}, []domain.Task{task})))

	m = applyMsg(t, m, keyRune('i'))
	if m.mode != modeTaskInfo {
		t.Fatalf("expected task info mode, got %v", m.mode)
	}
	m.descriptionPreview.SetYOffset(88)
	m = applyMsg(t, m, keyRune('d'))
	if m.mode != modeDescriptionEditor {
		t.Fatalf("expected description editor mode from task info, got %v", m.mode)
	}
	if m.descriptionEditorMode != descriptionEditorViewModePreview {
		t.Fatalf("expected task-info details to open in preview submode, got %v", m.descriptionEditorMode)
	}
	if got := m.descriptionPreview.YOffset(); got != 0 {
		t.Fatalf("expected task-info details preview to open at top, got y offset=%d", got)
	}
	if m.descriptionEditorBack != modeTaskInfo {
		t.Fatalf("expected task-info return mode, got %v", m.descriptionEditorBack)
	}
	if m.descriptionEditorTarget != descriptionEditorTargetThread {
		t.Fatalf("expected thread target for task-info details, got %v", m.descriptionEditorTarget)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	if m.descriptionEditorMode != descriptionEditorViewModeEdit {
		t.Fatalf("expected tab to open edit submode, got %v", m.descriptionEditorMode)
	}
	if got := m.descriptionEditorInput.ScrollYOffset(); got != 0 {
		t.Fatalf("expected task-info details edit mode to open at top, got y offset=%d", got)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.mode != modeTaskInfo {
		t.Fatalf("expected esc from details edit mode to return to task info, got %v", m.mode)
	}
}

// TestModelDescriptionEditorLayoutRespectsNarrowViewport verifies editor layout stays within terminal bounds.
func TestModelDescriptionEditorLayoutRespectsNarrowViewport(t *testing.T) {
	now := time.Date(2026, 3, 3, 9, 8, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{project}, []domain.Column{column}, nil)))

	m = applyMsg(t, m, keyRune('n'))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeDescriptionEditor {
		t.Fatalf("expected description editor mode, got %v", m.mode)
	}
	m = applyMsg(t, m, tea.WindowSizeMsg{Width: 50, Height: 20})

	layout := m.descriptionEditorLayout()
	if wantMax := m.width - 2; layout.layoutWidth > wantMax {
		t.Fatalf("expected layout width <= %d in narrow viewport, got %d", wantMax, layout.layoutWidth)
	}
	if !layout.splitVertically {
		t.Fatalf("expected narrow viewport to stack editor/preview vertically, got splitVertically=%v", layout.splitVertically)
	}

	m.descriptionEditorInput.SetValue(strings.Repeat("full page md ", 1200))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	if m.descriptionEditorMode != descriptionEditorViewModePreview {
		t.Fatalf("expected preview mode after tab, got %v", m.descriptionEditorMode)
	}

	before := m.descriptionPreview.YOffset()
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyPgDown})
	after := m.descriptionPreview.YOffset()
	if after <= before {
		t.Fatalf("expected preview scroll movement in narrow viewport, before=%d after=%d", before, after)
	}
}

// TestModelFullPageNodeViewStaysWithinScreenBounds verifies full-page node screens keep their frame inside the terminal.
func TestModelFullPageNodeViewStaysWithinScreenBounds(t *testing.T) {
	now := time.Date(2026, 3, 13, 11, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:          "t1",
		ProjectID:   project.ID,
		ColumnID:    column.ID,
		Position:    0,
		Title:       "Frame Bounds",
		Description: strings.Repeat("wrapped content line\n", 40),
		Priority:    domain.PriorityMedium,
	}, now)
	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{project}, []domain.Column{column}, []domain.Task{task})))

	m = applyMsg(t, m, tea.WindowSizeMsg{Width: 84, Height: 26})
	m = applyMsg(t, m, keyRune('e'))
	if m.mode != modeEditTask {
		t.Fatalf("expected edit mode, got %v", m.mode)
	}

	rendered := stripANSI(fmt.Sprint(m.View().Content))
	lines := strings.Split(rendered, "\n")
	if len(lines) > m.height {
		t.Fatalf("expected rendered line count <= %d, got %d", m.height, len(lines))
	}
	for idx, line := range lines {
		if lipgloss.Width(line) > m.width {
			t.Fatalf("expected rendered line %d width <= %d, got %d: %q", idx, m.width, lipgloss.Width(line), line)
		}
	}
	if !strings.Contains(rendered, "TILLSYN") {
		t.Fatalf("expected persistent TILLSYN header in full-page node view, got %q", rendered)
	}
	if !strings.Contains(rendered, "└") || !strings.Contains(rendered, "┘") {
		t.Fatalf("expected bottom border corners to remain visible, got %q", rendered)
	}
}

// TestModelInputModeGlobalHelpAndSelectionToggles verifies '?' and selection-toggle keys work inside modal/input screens.
func TestModelInputModeGlobalHelpAndSelectionToggles(t *testing.T) {
	now := time.Date(2026, 2, 23, 9, 30, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{p}, []domain.Column{c}, nil)))

	m = applyMsg(t, m, keyRune('N'))
	if m.mode != modeAddProject {
		t.Fatalf("expected add-project mode, got %v", m.mode)
	}

	m = applyMsg(t, m, keyRune('?'))
	if !m.help.ShowAll {
		t.Fatal("expected help overlay to open from add-project modal")
	}
	if m.mode != modeAddProject {
		t.Fatalf("expected help toggle to preserve add-project mode, got %v", m.mode)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: 'y', Mod: tea.ModCtrl})
	if !m.mouseSelectionMode {
		t.Fatal("expected text-selection mode to toggle while modal is open")
	}

	// Esc should close help first and keep the modal open.
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.help.ShowAll {
		t.Fatal("expected esc to close help overlay before modal")
	}
	if m.mode != modeAddProject {
		t.Fatalf("expected add-project mode to remain active after closing help, got %v", m.mode)
	}
}

// TestModelHelpOverlayModeSpecific verifies expanded help text changes by active screen.
func TestModelHelpOverlayModeSpecific(t *testing.T) {
	now := time.Date(2026, 2, 23, 9, 30, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{p}, []domain.Column{c}, nil)))

	accent := lipgloss.Color("62")
	muted := lipgloss.Color("241")
	dim := lipgloss.Color("239")

	m.mode = modeCommandPalette
	out := m.renderHelpOverlay(accent, muted, dim, lipgloss.NewStyle().Foreground(muted), 96)
	if !strings.Contains(out, "screen: command palette") {
		t.Fatalf("expected command-palette-specific help title, got %q", out)
	}
	if !strings.Contains(out, "type to filter command names and aliases") {
		t.Fatalf("expected command-palette-specific key guidance, got %q", out)
	}
	if strings.Contains(out, "pgup/pgdown") {
		t.Fatalf("expected thread-only text to be absent in command-palette help, got %q", out)
	}

	m.mode = modeDuePicker
	out = m.renderHelpOverlay(accent, muted, dim, lipgloss.NewStyle().Foreground(muted), 96)
	if !strings.Contains(out, "screen: due picker") {
		t.Fatalf("expected due-picker-specific help title, got %q", out)
	}
	if !strings.Contains(out, "tab cycles include-time, date, time, and options list focus") {
		t.Fatalf("expected due-picker-specific help guidance, got %q", out)
	}
}

// TestHelpOverlayScreenTitleAndLinesCoverage verifies each input mode resolves mode-scoped help lines.
func TestHelpOverlayScreenTitleAndLinesCoverage(t *testing.T) {
	now := time.Date(2026, 2, 23, 9, 30, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{p}, []domain.Column{c}, nil)))

	modes := []inputMode{
		modeNone,
		modeAddTask,
		modeSearch,
		modeRenameTask,
		modeEditTask,
		modeDuePicker,
		modeProjectPicker,
		modeTaskInfo,
		modeAddProject,
		modeEditProject,
		modeSearchResults,
		modeCommandPalette,
		modeQuickActions,
		modeConfirmAction,
		modeWarning,
		modeActivityLog,
		modeResourcePicker,
		modeLabelPicker,
		modePathsRoots,
		modeLabelsConfig,
		modeHighlightColor,
		modeBootstrapSettings,
		modeDependencyInspector,
		modeThread,
	}
	for _, mode := range modes {
		m.mode = mode
		title, lines := m.helpOverlayScreenTitleAndLines()
		if strings.TrimSpace(title) == "" {
			t.Fatalf("expected non-empty help title for mode %v", mode)
		}
		if len(lines) == 0 {
			t.Fatalf("expected non-empty help lines for mode %v", mode)
		}
	}

	m.mode = modeResourcePicker
	m.resourcePickerBack = modeAddProject
	title, lines := m.helpOverlayScreenTitleAndLines()
	if title != "path picker" {
		t.Fatalf("expected path-picker help title for root contexts, got %q", title)
	}
	if len(lines) == 0 {
		t.Fatal("expected path-picker help lines")
	}
}

// TestModelProjectIconEmojiSupport verifies project icon rendering and emoji persistence through form submit.
func TestModelProjectIconEmojiSupport(t *testing.T) {
	now := time.Date(2026, 2, 23, 9, 30, 0, 0, time.UTC)
	p1, _ := domain.NewProject("p1", "Inbox", "", now)
	if err := p1.UpdateDetails(p1.Name, p1.Description, domain.ProjectMetadata{Icon: "🚀"}, now); err != nil {
		t.Fatalf("UpdateDetails() setup error = %v", err)
	}
	p2, _ := domain.NewProject("p2", "Roadmap", "", now)
	c1, _ := domain.NewColumn("c1", p1.ID, "To Do", 0, 0, now)
	c2, _ := domain.NewColumn("c2", p2.ID, "To Do", 0, 0, now)

	svc := newFakeService([]domain.Project{p1, p2}, []domain.Column{c1, c2}, nil)
	m := loadReadyModel(t, NewModel(svc))
	viewText := fmt.Sprint(m.View().Content)
	if !strings.Contains(viewText, "🚀 Inbox") {
		t.Fatalf("expected project icon to render in board header/tabs, got %q", viewText)
	}

	m.mode = modeProjectPicker
	m.projectPickerIndex = 0
	accent := lipgloss.Color("62")
	muted := lipgloss.Color("241")
	dim := lipgloss.Color("239")
	picker := m.renderModeOverlay(accent, muted, dim, lipgloss.NewStyle().Foreground(muted), 80)
	if !strings.Contains(picker, "🚀 Inbox") {
		t.Fatalf("expected project picker to render icon label, got %q", picker)
	}

	m = applyMsg(t, m, keyRune('N'))
	if m.mode != modeAddProject {
		t.Fatalf("expected add-project mode, got %v", m.mode)
	}
	m.projectFormInputs[projectFieldName].SetValue("Emoji Project")
	m.projectFormInputs[projectFieldIcon].SetValue("🧠")
	updated, cmd := m.submitInputMode()
	m = applyResult(t, updated, cmd)
	if len(svc.projects) < 3 {
		t.Fatalf("expected created project in fake service, got %#v", svc.projects)
	}
	got := svc.projects[len(svc.projects)-1].Metadata.Icon
	if got != "🧠" {
		t.Fatalf("expected emoji icon to persist on create, got %q", got)
	}
}

// TestModelCommandPaletteWindowedRendering verifies command selection stays visible past the first page.
func TestModelCommandPaletteWindowedRendering(t *testing.T) {
	now := time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, nil)
	m := loadReadyModel(t, NewModel(svc))

	m = applyMsg(t, m, keyRune(':'))
	for i := 0; i < 12; i++ {
		m = applyMsg(t, m, keyRune('j'))
	}
	if m.commandIndex != 12 {
		t.Fatalf("expected commandIndex=12, got %d", m.commandIndex)
	}

	accent := lipgloss.Color("62")
	muted := lipgloss.Color("241")
	dim := lipgloss.Color("239")
	helpStyle := lipgloss.NewStyle().Foreground(muted)
	out := m.renderModeOverlay(accent, muted, dim, helpStyle, 96)
	selected := m.commandMatches[m.commandIndex].Command
	if !strings.Contains(out, "› "+selected) {
		t.Fatalf("expected selected command %q to remain visible in windowed list, got %q", selected, out)
	}
}

// TestModelQuickActionsDisabledOrderingAndBlocking verifies disabled quick actions sort last and cannot execute.
func TestModelQuickActionsDisabledOrderingAndBlocking(t *testing.T) {
	now := time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, nil)
	m := loadReadyModel(t, NewModel(svc))

	m = applyMsg(t, m, keyRune('.'))
	if m.mode != modeQuickActions {
		t.Fatalf("expected quick actions mode, got %v", m.mode)
	}

	accent := lipgloss.Color("62")
	muted := lipgloss.Color("241")
	dim := lipgloss.Color("239")
	helpStyle := lipgloss.NewStyle().Foreground(muted)
	out := m.renderModeOverlay(accent, muted, dim, helpStyle, 96)
	enabledIdx := strings.Index(out, "Activity Log")
	disabledIdx := strings.Index(out, "Task Info (no task selected)")
	if enabledIdx == -1 || disabledIdx == -1 || enabledIdx > disabledIdx {
		t.Fatalf("expected enabled action before disabled entries, got %q", out)
	}

	actions := m.quickActions()
	disabledIdx = -1
	for idx, action := range actions {
		if !action.Enabled {
			disabledIdx = idx
			break
		}
	}
	if disabledIdx < 0 {
		t.Fatal("expected at least one disabled quick action")
	}
	for m.quickActionIndex < disabledIdx {
		m = applyMsg(t, m, keyRune('j'))
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeQuickActions {
		t.Fatalf("expected disabled quick action to stay in quick-actions mode, got %v", m.mode)
	}
	if !strings.Contains(m.status, "unavailable") {
		t.Fatalf("expected unavailable status for disabled quick action, got %q", m.status)
	}
}

// TestModelCommandPaletteReloadConfigAppliesRuntimeSettings verifies behavior for the covered scenario.
func TestModelCommandPaletteReloadConfigAppliesRuntimeSettings(t *testing.T) {
	now := time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
		Metadata: domain.TaskMetadata{
			BlockedReason: "requires global follow-up",
		},
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})

	rootDir := t.TempDir()
	reloadCalls := 0
	m := loadReadyModel(t, NewModel(
		svc,
		WithReloadConfigCallback(func() (RuntimeConfig, error) {
			reloadCalls++
			return RuntimeConfig{
				DefaultDeleteMode: app.DeleteModeHard,
				TaskFields: TaskFieldConfig{
					ShowPriority:    false,
					ShowDueDate:     false,
					ShowLabels:      false,
					ShowDescription: true,
				},
				Search: SearchConfig{
					CrossProject:    true,
					IncludeArchived: true,
					States:          []string{"todo", "archived"},
				},
				Confirm: ConfirmConfig{
					Delete:     false,
					Archive:    false,
					HardDelete: false,
					Restore:    true,
				},
				Board: BoardConfig{
					ShowWIPWarnings: false,
					GroupBy:         "priority",
				},
				UI: UIConfig{
					DueSoonWindows: []time.Duration{2 * time.Hour},
					ShowDueSummary: false,
				},
				Labels: LabelConfig{
					Global:         []string{"bug"},
					Projects:       map[string][]string{"inbox": {"roadmap"}},
					EnforceAllowed: true,
				},
				ProjectRoots: map[string]string{"inbox": rootDir},
				Keys: KeyConfig{
					CommandPalette: ";",
					QuickActions:   ",",
					MultiSelect:    "x",
					ActivityLog:    "v",
					Undo:           "u",
					Redo:           "U",
				},
				Identity: IdentityConfig{
					ActorID:          "runtime-actor-id",
					DisplayName:      "Runtime User",
					DefaultActorType: "agent",
				},
			}, nil
		}),
	))

	updated, cmd := m.executeCommandPalette("reload-config")
	m = applyResult(t, updated, cmd)

	if reloadCalls != 1 {
		t.Fatalf("expected one reload callback invocation, got %d", reloadCalls)
	}
	if m.defaultDeleteMode != app.DeleteModeHard {
		t.Fatalf("expected hard default delete mode, got %q", m.defaultDeleteMode)
	}
	if m.taskFields.ShowPriority || m.taskFields.ShowDueDate || m.taskFields.ShowLabels || !m.taskFields.ShowDescription {
		t.Fatalf("unexpected task fields after reload %#v", m.taskFields)
	}
	if !m.searchCrossProject || !m.searchDefaultCrossProject || !m.searchIncludeArchived || !m.searchDefaultIncludeArchive {
		t.Fatalf("unexpected search scope flags after reload %#v", m)
	}
	if got := strings.Join(m.searchStates, ","); got != "todo,archived" {
		t.Fatalf("unexpected search states after reload %#v", m.searchStates)
	}
	if m.boardGroupBy != "priority" || m.showWIPWarnings {
		t.Fatalf("unexpected board config after reload group=%q wip=%t", m.boardGroupBy, m.showWIPWarnings)
	}
	if m.confirmDelete || m.confirmArchive || m.confirmHardDelete || !m.confirmRestore {
		t.Fatalf("unexpected confirm flags after reload delete=%t archive=%t hard=%t restore=%t", m.confirmDelete, m.confirmArchive, m.confirmHardDelete, m.confirmRestore)
	}
	if len(m.dueSoonWindows) != 1 || m.dueSoonWindows[0] != 2*time.Hour || m.showDueSummary {
		t.Fatalf("unexpected ui config after reload due=%#v summary=%t", m.dueSoonWindows, m.showDueSummary)
	}
	if !m.enforceAllowedLabels || len(m.allowedLabelGlobal) != 1 || m.allowedLabelGlobal[0] != "bug" {
		t.Fatalf("unexpected label config after reload %#v", m.allowedLabelGlobal)
	}
	if got := m.allowedLabelProject["inbox"]; len(got) != 1 || got[0] != "roadmap" {
		t.Fatalf("unexpected per-project labels after reload %#v", m.allowedLabelProject)
	}
	if got := m.projectRoots["inbox"]; got != rootDir {
		t.Fatalf("unexpected project roots after reload %#v", m.projectRoots)
	}
	if got := m.identityActorID; got != "runtime-actor-id" {
		t.Fatalf("expected actor_id runtime-actor-id after reload, got %q", got)
	}
	if got := m.identityDisplayName; got != "Runtime User" {
		t.Fatalf("expected display_name Runtime User after reload, got %q", got)
	}
	if got := m.identityDefaultActorType; got != "agent" {
		t.Fatalf("expected default_actor_type agent after reload, got %q", got)
	}
	if m.status != "config reloaded" {
		t.Fatalf("expected config reloaded status, got %q", m.status)
	}

	m = applyMsg(t, m, keyRune(';'))
	if m.mode != modeCommandPalette {
		t.Fatalf("expected command palette open with reloaded keybinding, got %v", m.mode)
	}
}

// TestModelWithRuntimeConfigAppliesIdentityAtStartup verifies runtime identity config applies during model construction.
func TestModelWithRuntimeConfigAppliesIdentityAtStartup(t *testing.T) {
	m := NewModel(
		newFakeService(nil, nil, nil),
		WithRuntimeConfig(RuntimeConfig{
			Identity: IdentityConfig{
				ActorID:          "runtime-startup-actor",
				DisplayName:      "Runtime Startup",
				DefaultActorType: "agent",
			},
		}),
	)

	if got := m.identityActorID; got != "runtime-startup-actor" {
		t.Fatalf("expected startup actor_id runtime-startup-actor, got %q", got)
	}
	if got := m.identityDisplayName; got != "Runtime Startup" {
		t.Fatalf("expected startup display_name Runtime Startup, got %q", got)
	}
	if got := m.identityDefaultActorType; got != "agent" {
		t.Fatalf("expected startup default_actor_type agent, got %q", got)
	}
}

// TestModelCommandPaletteReloadConfigError verifies behavior for the covered scenario.
func TestModelCommandPaletteReloadConfigError(t *testing.T) {
	now := time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})
	m := loadReadyModel(t, NewModel(svc, WithReloadConfigCallback(func() (RuntimeConfig, error) {
		return RuntimeConfig{}, errors.New("disk read failed")
	})))

	updated, cmd := m.executeCommandPalette("reload-config")
	m = applyResult(t, updated, cmd)
	if !strings.Contains(m.status, "reload config failed") {
		t.Fatalf("expected reload-config error status, got %q", m.status)
	}
}

// TestModelPathsRootsModalSaveAndClear verifies behavior for the covered scenario.
func TestModelPathsRootsModalSaveAndClear(t *testing.T) {
	now := time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})

	rootDir := t.TempDir()
	type saveCall struct {
		slug string
		path string
	}
	saveCalls := make([]saveCall, 0, 2)
	m := loadReadyModel(t, NewModel(
		svc,
		WithSaveProjectRootCallback(func(projectSlug, rootPath string) error {
			saveCalls = append(saveCalls, saveCall{slug: projectSlug, path: rootPath})
			return nil
		}),
	))

	updated, cmd := m.executeCommandPalette("paths-roots")
	m = applyResult(t, updated, cmd)
	if m.mode != modePathsRoots {
		t.Fatalf("expected paths/roots modal mode, got %v", m.mode)
	}
	m.pathsRootInput.SetValue(rootDir)
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})

	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		t.Fatalf("Abs() error = %v", err)
	}
	if len(saveCalls) != 1 || saveCalls[0].slug != "inbox" || saveCalls[0].path != absRoot {
		t.Fatalf("unexpected save callback calls %#v", saveCalls)
	}
	if got := m.projectRoots["inbox"]; got != absRoot {
		t.Fatalf("expected in-memory project root update to %q, got %#v", absRoot, m.projectRoots)
	}
	if m.status != "project root saved" {
		t.Fatalf("expected project root saved status, got %q", m.status)
	}

	updated, cmd = m.executeCommandPalette("paths-roots")
	m = applyResult(t, updated, cmd)
	m.pathsRootInput.SetValue("")
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if len(saveCalls) != 2 || saveCalls[1].slug != "inbox" || saveCalls[1].path != "" {
		t.Fatalf("unexpected clear callback calls %#v", saveCalls)
	}
	if _, ok := m.projectRoots["inbox"]; ok {
		t.Fatalf("expected root mapping cleared, got %#v", m.projectRoots)
	}
	if m.status != "project root cleared" {
		t.Fatalf("expected project root cleared status, got %q", m.status)
	}
}

// TestModelPathsRootsModalValidationAndSaveError verifies behavior for the covered scenario.
func TestModelPathsRootsModalValidationAndSaveError(t *testing.T) {
	now := time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})

	saveCalls := 0
	m := loadReadyModel(t, NewModel(
		svc,
		WithSaveProjectRootCallback(func(projectSlug, rootPath string) error {
			saveCalls++
			return errors.New("write failed")
		}),
	))

	updated, cmd := m.executeCommandPalette("paths-roots")
	m = applyResult(t, updated, cmd)
	m.pathsRootInput.SetValue(filepath.Join(t.TempDir(), "missing"))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modePathsRoots {
		t.Fatalf("expected modal to remain open after validation failure, got %v", m.mode)
	}
	if saveCalls != 0 {
		t.Fatalf("expected no callback invocation on validation failure, got %d", saveCalls)
	}
	if !strings.Contains(m.status, "root path not found") {
		t.Fatalf("expected not-found validation message, got %q", m.status)
	}

	validRoot := t.TempDir()
	m.pathsRootInput.SetValue(validRoot)
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeNone {
		t.Fatalf("expected modal close after callback execution, got %v", m.mode)
	}
	if saveCalls != 1 {
		t.Fatalf("expected one callback invocation after valid submit, got %d", saveCalls)
	}
	if !strings.Contains(m.status, "save root failed") {
		t.Fatalf("expected save error message, got %q", m.status)
	}
}

// TestModelResourcePickerFallsBackToBootstrapRootInTaskInfo verifies project-root lookup falls back to bootstrap roots while task-info remains read-only.
func TestModelResourcePickerFallsBackToBootstrapRootInTaskInfo(t *testing.T) {
	now := time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	root := t.TempDir()

	m := loadReadyModel(t, NewModel(
		newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task}),
		WithSearchRoots([]string{root}),
	))
	if got := m.resourcePickerRootForCurrentProject(); got != root {
		t.Fatalf("expected bootstrap root fallback %q, got %q", root, got)
	}
	m = applyMsg(t, m, keyRune('i'))
	m = applyMsg(t, m, keyRune('r'))
	if m.mode != modeTaskInfo {
		t.Fatalf("expected task info mode to ignore resource attach key, got %v", m.mode)
	}
}

// TestModelMouseWheelAndClick verifies behavior for the covered scenario.
func TestModelMouseWheelAndClick(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c1, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	c2, _ := domain.NewColumn("c2", p.ID, "Done", 1, 0, now)
	t1, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c1.ID,
		Position:  0,
		Title:     "One",
		Priority:  domain.PriorityLow,
	}, now)
	t2, _ := domain.NewTask(domain.TaskInput{
		ID:        "t2",
		ProjectID: p.ID,
		ColumnID:  c1.ID,
		Position:  1,
		Title:     "Two",
		Priority:  domain.PriorityLow,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c1, c2}, []domain.Task{t1, t2})
	m := loadReadyModel(t, NewModel(svc))

	m = applyMsg(t, m, tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	if m.selectedTask != 1 {
		t.Fatalf("expected selectedTask=1 after wheel down, got %d", m.selectedTask)
	}

	clickX := m.columnWidth() + 5
	clickY := m.boardTop() + 2
	m = applyMsg(t, m, tea.MouseClickMsg{X: clickX, Y: clickY, Button: tea.MouseLeft})
	if m.selectedColumn != 1 {
		t.Fatalf("expected selectedColumn=1 after mouse click, got %d", m.selectedColumn)
	}
}

// TestModelBoardHidesSubtasksAndShowsProgress verifies board rows hide subtask cards but show progress metadata.
func TestModelBoardHidesSubtasksAndShowsProgress(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c1, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)

	parent, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-parent",
		ProjectID: p.ID,
		ColumnID:  c1.ID,
		Position:  0,
		Title:     "Parent task",
		Priority:  domain.PriorityMedium,
	}, now)
	childDone, _ := domain.NewTask(domain.TaskInput{
		ID:             "t-child-done",
		ProjectID:      p.ID,
		ColumnID:       c1.ID,
		Position:       1,
		Title:          "Child done",
		Priority:       domain.PriorityLow,
		Kind:           domain.WorkKindSubtask,
		ParentID:       parent.ID,
		LifecycleState: domain.StateDone,
	}, now)
	childTodo, _ := domain.NewTask(domain.TaskInput{
		ID:             "t-child-todo",
		ProjectID:      p.ID,
		ColumnID:       c1.ID,
		Position:       2,
		Title:          "Child todo",
		Priority:       domain.PriorityLow,
		Kind:           domain.WorkKindSubtask,
		ParentID:       parent.ID,
		LifecycleState: domain.StateTodo,
	}, now)

	svc := newFakeService([]domain.Project{p}, []domain.Column{c1}, []domain.Task{parent, childDone, childTodo})
	m := loadReadyModel(t, NewModel(svc))

	view := m.View()
	rendered := fmt.Sprint(view.Content)
	if strings.Contains(rendered, "Child done") || strings.Contains(rendered, "Child todo") {
		t.Fatalf("expected subtasks hidden from board rows, got\n%s", rendered)
	}
	if !strings.Contains(rendered, "[medium|1/2]") {
		t.Fatalf("expected subtask progress metadata in board row, got\n%s", rendered)
	}
}

// TestModelTaskInfoShowsSubtasksAcrossColumns verifies task-info modal subtask visibility independent of parent column.
func TestModelTaskInfoShowsSubtasksAcrossColumns(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	cTodo, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	cProgress, _ := domain.NewColumn("c2", p.ID, "In Progress", 1, 0, now)

	parent, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-parent",
		ProjectID: p.ID,
		ColumnID:  cProgress.ID,
		Position:  0,
		Title:     "Parent task",
		Priority:  domain.PriorityHigh,
	}, now)
	child, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-child",
		ProjectID: p.ID,
		ColumnID:  cTodo.ID,
		Position:  0,
		Title:     "Cross-column subtask",
		Priority:  domain.PriorityLow,
		Kind:      domain.WorkKindSubtask,
		ParentID:  parent.ID,
	}, now)

	svc := newFakeService([]domain.Project{p}, []domain.Column{cTodo, cProgress}, []domain.Task{parent, child})
	m := loadReadyModel(t, NewModel(svc))
	m.focusTaskByID(parent.ID)

	m = applyMsg(t, m, keyRune('i'))
	if m.mode != modeTaskInfo {
		t.Fatalf("expected task info mode, got %v", m.mode)
	}

	view := m.View()
	rendered := fmt.Sprint(view.Content)
	if !strings.Contains(rendered, "Cross-column subtask") {
		t.Fatalf("expected task info modal to include subtasks across columns, got\n%s", rendered)
	}

	editMode := m
	_ = editMode.startTaskForm(&parent)
	editBody, _ := editMode.taskFormBodyLines(72, lipgloss.NewStyle().Foreground(lipgloss.Color("241")), lipgloss.Color("62"))
	editRendered := strings.ToLower(stripANSI(strings.Join(editBody, "\n")))
	if !strings.Contains(editRendered, "subtasks") || !strings.Contains(editRendered, "cross-column subtask") {
		t.Fatalf("expected edit-task modal to include subtasks section with children, got %q", editRendered)
	}
}

// TestModelTaskInfoBackspaceMovesToParent verifies backspace navigates from a child info view to its parent.
func TestModelTaskInfoBackspaceMovesToParent(t *testing.T) {
	now := time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	cTodo, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	cProgress, _ := domain.NewColumn("c2", p.ID, "In Progress", 1, 0, now)

	parent, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-parent",
		ProjectID: p.ID,
		ColumnID:  cProgress.ID,
		Position:  0,
		Title:     "Parent task",
		Priority:  domain.PriorityHigh,
	}, now)
	child, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-child",
		ProjectID: p.ID,
		ColumnID:  cTodo.ID,
		Position:  0,
		Title:     "Nested subtask",
		Priority:  domain.PriorityLow,
		Kind:      domain.WorkKindSubtask,
		ParentID:  parent.ID,
	}, now)

	svc := newFakeService([]domain.Project{p}, []domain.Column{cTodo, cProgress}, []domain.Task{parent, child})
	m := loadReadyModel(t, NewModel(svc))
	if !m.openTaskInfo(child.ID, "task info") {
		t.Fatal("expected child task info to open")
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyBackspace})
	if m.mode != modeTaskInfo {
		t.Fatalf("expected task-info mode after moving to parent, got %v", m.mode)
	}
	if m.taskInfoTaskID != parent.ID {
		t.Fatalf("expected backspace to open parent, got %q", m.taskInfoTaskID)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.mode != modeNone {
		t.Fatalf("expected esc to close task-info from parent, got %v", m.mode)
	}
}

// TestModelTaskInfoEscClosesCurrentView verifies esc closes the current task-info view without subtask drill-in state.
func TestModelTaskInfoEscClosesCurrentView(t *testing.T) {
	now := time.Date(2026, 3, 3, 12, 50, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	grand, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-grand",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Kind:      domain.WorkKindTask,
		Title:     "Grandparent",
		Priority:  domain.PriorityMedium,
	}, now)
	parent, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-parent",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  1,
		Kind:      domain.WorkKindTask,
		ParentID:  grand.ID,
		Title:     "Parent",
		Priority:  domain.PriorityMedium,
	}, now)
	child, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-child",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  2,
		Kind:      domain.WorkKindSubtask,
		ParentID:  parent.ID,
		Title:     "Child",
		Priority:  domain.PriorityLow,
	}, now)

	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{project}, []domain.Column{column}, []domain.Task{grand, parent, child})))
	if !m.openTaskInfo(child.ID, "task info") {
		t.Fatal("expected openTaskInfo(child) to succeed")
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.mode != modeNone {
		t.Fatalf("expected esc to close current task info view, got %v", m.mode)
	}
	if got := strings.TrimSpace(m.taskInfoTaskID); got != "" {
		t.Fatalf("expected closed task info to clear active task id, got %q", got)
	}
}

// TestModelTaskInfoAllowsSubtaskCreation verifies task-info view can start a subtask form for the focused task.
func TestModelTaskInfoAllowsSubtaskCreation(t *testing.T) {
	now := time.Date(2026, 2, 23, 10, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	parent, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-parent",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Parent",
		Priority:  domain.PriorityMedium,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{parent})
	m := loadReadyModel(t, NewModel(svc))
	m.focusTaskByID(parent.ID)

	m = applyMsg(t, m, keyRune('i'))
	if m.mode != modeTaskInfo {
		t.Fatalf("expected task-info mode, got %v", m.mode)
	}
	m = applyMsg(t, m, keyRune('s'))
	if m.mode != modeAddTask {
		t.Fatalf("expected add-task mode after task-info subtask key, got %v", m.mode)
	}
	if got := m.taskFormKind; got != domain.WorkKindSubtask {
		t.Fatalf("expected subtask form kind, got %q", got)
	}
	if got := m.taskFormParentID; got != parent.ID {
		t.Fatalf("expected parent id %q, got %q", parent.ID, got)
	}
}

// TestModelTaskInfoEnterOpensFocusedSubtask verifies enter drills into the highlighted child task.
func TestModelTaskInfoEnterOpensFocusedSubtask(t *testing.T) {
	now := time.Date(2026, 3, 3, 15, 10, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	parent, _ := domain.NewTask(domain.TaskInput{
		ID:        "parent",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Title:     "Parent",
		Priority:  domain.PriorityMedium,
	}, now)
	child, _ := domain.NewTask(domain.TaskInput{
		ID:        "child",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		ParentID:  parent.ID,
		Kind:      domain.WorkKindSubtask,
		Scope:     domain.KindAppliesToSubtask,
		Title:     "Child",
		Priority:  domain.PriorityMedium,
	}, now)
	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{project}, []domain.Column{column}, []domain.Task{parent, child})))
	m.focusTaskByID(parent.ID)

	m = applyMsg(t, m, keyRune('i'))
	if m.mode != modeTaskInfo {
		t.Fatalf("expected task info mode, got %v", m.mode)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeTaskInfo {
		t.Fatalf("expected task info mode after enter, got %v", m.mode)
	}
	if m.taskInfoTaskID != child.ID {
		t.Fatalf("expected enter to open subtask %q, got %q", child.ID, m.taskInfoTaskID)
	}
}

// TestModelTaskInfoMovesCurrentTaskWithBrackets verifies task-info mode supports moving the focused task between columns.
func TestModelTaskInfoMovesCurrentTaskWithBrackets(t *testing.T) {
	now := time.Date(2026, 2, 23, 10, 15, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	cTodo, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	cProgress, _ := domain.NewColumn("c2", p.ID, "In Progress", 1, 0, now)

	parent, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-parent",
		ProjectID: p.ID,
		ColumnID:  cTodo.ID,
		Position:  0,
		Title:     "Parent",
		Priority:  domain.PriorityMedium,
	}, now)
	child, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-child",
		ProjectID: p.ID,
		ColumnID:  cTodo.ID,
		Position:  1,
		Title:     "Child",
		Priority:  domain.PriorityLow,
		Kind:      domain.WorkKindSubtask,
		ParentID:  parent.ID,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{cTodo, cProgress}, []domain.Task{parent, child})
	m := loadReadyModel(t, NewModel(svc))
	if !m.openTaskInfo(child.ID, "task info") {
		t.Fatal("expected child task info to open")
	}

	m = applyMsg(t, m, keyRune(']'))
	moved, ok := svc.taskByID(child.ID)
	if !ok {
		t.Fatalf("expected moved child task in fake service")
	}
	if moved.ColumnID != cProgress.ID {
		t.Fatalf("expected child moved to progress column %q, got %q", cProgress.ID, moved.ColumnID)
	}
	if m.mode != modeTaskInfo {
		t.Fatalf("expected task-info mode retained after move, got %v", m.mode)
	}

	m = applyMsg(t, m, keyRune('['))
	moved, ok = svc.taskByID(child.ID)
	if !ok {
		t.Fatalf("expected moved child task in fake service")
	}
	if moved.ColumnID != cTodo.ID {
		t.Fatalf("expected child moved back to todo column %q, got %q", cTodo.ID, moved.ColumnID)
	}
}

// TestModelTaskInfoSubtaskChecklistToggleCompletion verifies task-info checklist rendering and completion toggling.
func TestModelTaskInfoSubtaskChecklistToggleCompletion(t *testing.T) {
	now := time.Date(2026, 2, 23, 10, 20, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	cTodo, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	cProgress, _ := domain.NewColumn("c2", p.ID, "In Progress", 1, 0, now)
	cDone, _ := domain.NewColumn("c3", p.ID, "Done", 2, 0, now)

	parent, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-parent",
		ProjectID: p.ID,
		ColumnID:  cProgress.ID,
		Position:  0,
		Title:     "Parent",
		Priority:  domain.PriorityMedium,
	}, now)
	child, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-child",
		ProjectID: p.ID,
		ColumnID:  cTodo.ID,
		Position:  0,
		Title:     "Child",
		Priority:  domain.PriorityLow,
		Kind:      domain.WorkKindSubtask,
		ParentID:  parent.ID,
	}, now)

	svc := newFakeService([]domain.Project{p}, []domain.Column{cTodo, cProgress, cDone}, []domain.Task{parent, child})
	m := loadReadyModel(t, NewModel(svc))
	m.focusTaskByID(parent.ID)
	m = applyMsg(t, m, keyRune('i'))

	rendered := stripANSI(fmt.Sprint(m.View().Content))
	if !strings.Contains(rendered, "[ ] Child") {
		t.Fatalf("expected unchecked checklist row for subtask, got\n%s", rendered)
	}
	if !strings.Contains(rendered, "state:To Do") || !strings.Contains(rendered, "complete:no") {
		t.Fatalf("expected subtask state metadata in task-info checklist, got\n%s", rendered)
	}

	m = applyMsg(t, m, keyRune(' '))
	moved, ok := svc.taskByID(child.ID)
	if !ok {
		t.Fatalf("expected moved child task in fake service")
	}
	if moved.ColumnID != cDone.ID {
		t.Fatalf("expected checklist toggle to move child to done column %q, got %q", cDone.ID, moved.ColumnID)
	}

	rendered = stripANSI(fmt.Sprint(m.View().Content))
	if !strings.Contains(rendered, "[x] Child") {
		t.Fatalf("expected checked checklist row after completion toggle, got\n%s", rendered)
	}
	if !strings.Contains(rendered, "state:Done") || !strings.Contains(rendered, "complete:yes") {
		t.Fatalf("expected done state metadata after completion toggle, got\n%s", rendered)
	}

	m = applyMsg(t, m, keyRune(' '))
	moved, ok = svc.taskByID(child.ID)
	if !ok {
		t.Fatalf("expected moved child task in fake service")
	}
	if moved.ColumnID != cProgress.ID {
		t.Fatalf("expected reopening toggle to move child to progress column %q, got %q", cProgress.ID, moved.ColumnID)
	}
}

// TestModelBoardScrollKeepsSelectedRowVisible verifies dynamic list scrolling for long columns.
func TestModelBoardScrollKeepsSelectedRowVisible(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c1, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)

	tasks := make([]domain.Task, 0, 36)
	for i := 0; i < 36; i++ {
		task, _ := domain.NewTask(domain.TaskInput{
			ID:        fmt.Sprintf("t-%02d", i),
			ProjectID: p.ID,
			ColumnID:  c1.ID,
			Position:  i,
			Title:     fmt.Sprintf("Task %02d", i),
			Priority:  domain.PriorityLow,
		}, now)
		tasks = append(tasks, task)
	}

	svc := newFakeService([]domain.Project{p}, []domain.Column{c1}, tasks)
	m := loadReadyModel(t, NewModel(svc))
	for i := 0; i < 30; i++ {
		m = applyMsg(t, m, keyRune('j'))
	}

	view := m.View()
	rendered := fmt.Sprint(view.Content)
	if !strings.Contains(rendered, "Task 30") {
		t.Fatalf("expected selected row to remain visible after scrolling, got\n%s", rendered)
	}
}

// TestModelFocusedAndSelectedStyling verifies focused rows use fuchsia and retain multi-select cues.
func TestModelFocusedAndSelectedStyling(t *testing.T) {
	now := time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Styled task",
		Priority:  domain.PriorityMedium,
	}, now)

	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})
	m := loadReadyModel(t, NewModel(svc))
	m.selectedTaskIDs = map[string]struct{}{task.ID: {}}

	view := m.View()
	rendered := fmt.Sprint(view.Content)
	if !strings.Contains(rendered, "38;5;212") {
		t.Fatalf("expected focused task rendered with fuchsia color, got\n%s", rendered)
	}
	plain := stripANSI(rendered)
	if !strings.Contains(plain, "│* Styled task") {
		t.Fatalf("expected focused+selected marker to preserve selection cue, got\n%s", plain)
	}
}

// TestModelSelectionMarkerOnlyOnTitleLine verifies marker symbols are not repeated on secondary card lines.
func TestModelSelectionMarkerOnlyOnTitleLine(t *testing.T) {
	now := time.Date(2026, 2, 23, 11, 30, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Has metadata line",
		Priority:  domain.PriorityMedium,
	}, now)

	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})
	m := loadReadyModel(t, NewModel(svc))
	m.selectedTaskIDs = map[string]struct{}{task.ID: {}}

	plain := stripANSI(fmt.Sprint(m.View().Content))
	if strings.Count(plain, "│* ") != 1 {
		t.Fatalf("expected exactly one focused+selected marker on title line, got\n%s", plain)
	}
}

// TestModelEscClearsSubtreeFocus verifies esc returns to full-board view from subtree focus.
func TestModelEscClearsSubtreeFocus(t *testing.T) {
	now := time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c1, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	parent, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-parent",
		ProjectID: p.ID,
		ColumnID:  c1.ID,
		Position:  0,
		Title:     "Parent",
		Priority:  domain.PriorityMedium,
	}, now)
	child, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-child",
		ProjectID: p.ID,
		ColumnID:  c1.ID,
		Position:  1,
		Title:     "Child",
		Priority:  domain.PriorityLow,
		Kind:      domain.WorkKindSubtask,
		ParentID:  parent.ID,
	}, now)
	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{p}, []domain.Column{c1}, []domain.Task{parent, child})))
	m = applyMsg(t, m, keyRune('f'))
	if m.projectionRootTaskID == "" {
		t.Fatal("expected subtree focus root to be set")
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.projectionRootTaskID != "" {
		t.Fatalf("expected esc to clear subtree focus, got %q", m.projectionRootTaskID)
	}
	if m.status != "full board view" {
		t.Fatalf("expected full-board status, got %q", m.status)
	}
}

// TestModelQuitKey verifies behavior for the covered scenario.
func TestModelQuitKey(t *testing.T) {
	m := NewModel(newFakeService(nil, nil, nil))
	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	if updated == nil {
		t.Fatal("expected model return value")
	}
	if cmd == nil {
		t.Fatal("expected quit cmd")
	}
}

// TestModelViewStatesAndPrompts verifies behavior for the covered scenario.
func TestModelViewStatesAndPrompts(t *testing.T) {
	m := NewModel(newFakeService(nil, nil, nil))
	v := m.View()
	if v.Content == nil || v.MouseMode != tea.MouseModeCellMotion {
		t.Fatal("expected loading view with mouse enabled")
	}

	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, nil)
	m = loadReadyModel(t, NewModel(svc))
	m.mode = modeAddTask
	m.input = "abc"
	if !strings.Contains(m.modePrompt(), "new task:") {
		t.Fatal("expected add mode prompt")
	}

	m.err = context.DeadlineExceeded
	v = m.View()
	if v.Content == nil {
		t.Fatal("expected error view content")
	}
}

// TestModelNoProjectsKeepsPickerAndCreationFlow verifies first-run picker flow remains usable for creation.
func TestModelNoProjectsKeepsPickerAndCreationFlow(t *testing.T) {
	m := loadReadyModel(t, NewModel(newFakeService(nil, nil, nil)))
	if m.mode != modeProjectPicker {
		t.Fatalf("expected project-picker mode for empty workspace, got %v", m.mode)
	}
	view := m.View()
	rendered := fmt.Sprint(view.Content)
	if !strings.Contains(rendered, "Projects") || !strings.Contains(rendered, "N new project") {
		t.Fatalf("expected project-picker overlay with create action, got\n%s", rendered)
	}

	m = applyMsg(t, m, keyRune('N'))
	if m.mode != modeAddProject {
		t.Fatalf("expected add-project mode after picker new-project action, got %v", m.mode)
	}
}

// TestModelLaunchStartsInProjectPicker verifies first launch opens the project picker before normal mode.
func TestModelLaunchStartsInProjectPicker(t *testing.T) {
	now := time.Date(2026, 2, 23, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, nil)

	initial := NewModel(svc, WithLaunchProjectPicker(true))
	ready := applyMsg(t, applyCmd(t, initial, initial.Init()), tea.WindowSizeMsg{Width: 120, Height: 40})
	if ready.mode != modeProjectPicker {
		t.Fatalf("expected launch picker mode, got %v", ready.mode)
	}
	if ready.projectPickerIndex != 0 {
		t.Fatalf("expected picker index 0 on launch, got %d", ready.projectPickerIndex)
	}
}

// TestModelStartupBootstrapPrecedesLaunchPicker verifies startup bootstrap modal ordering and completion.
func TestModelStartupBootstrapPrecedesLaunchPicker(t *testing.T) {
	now := time.Date(2026, 2, 23, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, nil)

	root := t.TempDir()
	saveCalls := 0
	var saved BootstrapConfig
	initial := NewModel(
		svc,
		WithLaunchProjectPicker(true),
		WithStartupBootstrap(true),
		WithSaveBootstrapConfigCallback(func(cfg BootstrapConfig) error {
			saveCalls++
			saved = cfg
			return nil
		}),
	)
	ready := applyMsg(t, applyCmd(t, initial, initial.Init()), tea.WindowSizeMsg{Width: 120, Height: 40})
	if ready.mode != modeBootstrapSettings {
		t.Fatalf("expected startup bootstrap mode, got %v", ready.mode)
	}
	ready = applyMsg(t, ready, tea.KeyPressMsg{Code: tea.KeyEscape})
	if ready.mode != modeBootstrapSettings {
		t.Fatalf("expected mandatory bootstrap modal to ignore esc, got %v", ready.mode)
	}

	ready.bootstrapDisplayInput.SetValue("Lane User")
	ready.bootstrapActorIndex = bootstrapActorTypeIndex("agent")
	ready.bootstrapRoots = []string{root}
	ready.bootstrapFocus = 2
	ready = applyMsg(t, ready, tea.KeyPressMsg{Code: tea.KeyEnter})
	if saveCalls != 1 {
		t.Fatalf("expected one bootstrap save call, got %d", saveCalls)
	}
	if saved.ActorID != "tillsyn-user" || saved.DisplayName != "Lane User" || saved.DefaultActorType != "user" {
		t.Fatalf("unexpected saved bootstrap config %#v", saved)
	}
	if ready.mode != modeProjectPicker {
		t.Fatalf("expected project picker after bootstrap save, got %v", ready.mode)
	}
}

// TestModelBootstrapSettingsCommandPaletteRootsEditing verifies command-palette bootstrap settings editing and fuzzy root add.
func TestModelBootstrapSettingsCommandPaletteRootsEditing(t *testing.T) {
	now := time.Date(2026, 2, 23, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	rootA := t.TempDir()
	rootB := t.TempDir()
	if err := os.WriteFile(filepath.Join(rootB, "notes.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	saveCalls := 0
	var saved BootstrapConfig
	m := loadReadyModel(t, NewModel(
		newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task}),
		WithIdentityConfig(IdentityConfig{
			DisplayName:      "Lane User",
			DefaultActorType: "user",
		}),
		WithSearchRoots([]string{rootA}),
		WithSaveBootstrapConfigCallback(func(cfg BootstrapConfig) error {
			saveCalls++
			saved = cfg
			return nil
		}),
	))
	m.defaultRootDir = rootB

	updated, cmd := m.executeCommandPalette("bootstrap-settings")
	m = applyResult(t, updated, cmd)
	if m.mode != modeBootstrapSettings {
		t.Fatalf("expected bootstrap settings mode from command palette, got %v", m.mode)
	}
	if got := m.bootstrapDisplayInput.Value(); got != "Lane User" {
		t.Fatalf("expected display name prefill Lane User, got %q", got)
	}
	if len(m.bootstrapRoots) != 1 || m.bootstrapRoots[0] != filepath.Clean(rootA) {
		t.Fatalf("expected root prefill %q, got %#v", filepath.Clean(rootA), m.bootstrapRoots)
	}

	m = applyCmd(t, m, m.focusBootstrapField(1))
	m = applyMsg(t, m, keyRune('d'))
	if len(m.bootstrapRoots) != 0 {
		t.Fatalf("expected root removal from bootstrap modal, got %#v", m.bootstrapRoots)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
	if m.mode != modeResourcePicker {
		t.Fatalf("expected resource picker for bootstrap root browse, got %v", m.mode)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl})
	if m.mode != modeBootstrapSettings {
		t.Fatalf("expected return to bootstrap modal after root attach, got %v", m.mode)
	}
	if len(m.bootstrapRoots) != 1 || m.bootstrapRoots[0] != filepath.Clean(rootB) {
		t.Fatalf("expected root picker add %q, got %#v", filepath.Clean(rootB), m.bootstrapRoots)
	}
	m = applyCmd(t, m, m.focusBootstrapField(1))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	if m.bootstrapFocus != 2 {
		t.Fatalf("expected down arrow on roots to continue focus navigation, got %d", m.bootstrapFocus)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
	if m.bootstrapFocus != 1 {
		t.Fatalf("expected up arrow to return focus to roots, got %d", m.bootstrapFocus)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
	if m.bootstrapFocus != 0 {
		t.Fatalf("expected up arrow on first root row to move focus backward, got %d", m.bootstrapFocus)
	}

	m.bootstrapDisplayInput.SetValue("Lane Agent")
	m.bootstrapActorIndex = bootstrapActorTypeIndex("system")
	m = applyCmd(t, m, m.focusBootstrapField(2))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if saveCalls != 1 {
		t.Fatalf("expected one bootstrap save call, got %d", saveCalls)
	}
	if m.mode != modeNone {
		t.Fatalf("expected bootstrap modal to close after save, got %v", m.mode)
	}
	if m.identityDisplayName != "Lane Agent" || m.identityDefaultActorType != "system" {
		t.Fatalf("expected in-memory bootstrap settings update, got name=%q actor=%q", m.identityDisplayName, m.identityDefaultActorType)
	}
	if len(m.searchRoots) != 1 || m.searchRoots[0] != filepath.Clean(rootB) {
		t.Fatalf("expected in-memory search roots update %q, got %#v", filepath.Clean(rootB), m.searchRoots)
	}
	if saved.ActorID != "tillsyn-user" || saved.DisplayName != "Lane Agent" || saved.DefaultActorType != "system" {
		t.Fatalf("unexpected callback bootstrap payload %#v", saved)
	}
}

// TestModelInputModePaths verifies behavior for the covered scenario.
func TestModelInputModePaths(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityLow,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})
	m := loadReadyModel(t, NewModel(svc))

	m = applyMsg(t, m, keyRune('n'))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyBackspace})
	if m.mode != modeAddTask {
		t.Fatalf("expected add mode, got %v", m.mode)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.mode != modeNone {
		t.Fatalf("expected modeNone after escape, got %v", m.mode)
	}

	m = applyMsg(t, m, keyRune('n'))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter}) // empty submit
	if !strings.Contains(m.status, "title required") {
		t.Fatalf("expected title required status, got %q", m.status)
	}

	m = applyMsg(t, m, keyRune('/'))
	m = applyMsg(t, m, keyRune('T'))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.searchQuery != "T" {
		t.Fatalf("expected search query set, got %q", m.searchQuery)
	}

	m = applyMsg(t, m, keyRune('e'))
	m.input = "Task 2 | expanded details | high | 2026-03-01 | alpha,beta"
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if !strings.Contains(svc.tasks[p.ID][0].Title, "Task 2") {
		t.Fatalf("expected edited title, got %q", svc.tasks[p.ID][0].Title)
	}
	if svc.tasks[p.ID][0].Priority != domain.PriorityHigh || len(svc.tasks[p.ID][0].Labels) != 2 {
		t.Fatalf("expected full-field update, got %#v", svc.tasks[p.ID][0])
	}
}

// TestModelNormalModeExtraBranches verifies behavior for the covered scenario.
func TestModelNormalModeExtraBranches(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p1, _ := domain.NewProject("p1", "A", "", now)
	p2, _ := domain.NewProject("p2", "B", "", now)
	c1, _ := domain.NewColumn("c1", p1.ID, "To Do", 0, 0, now)
	c2, _ := domain.NewColumn("c2", p2.ID, "To Do", 0, 0, now)
	t1, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p1.ID,
		ColumnID:  c1.ID,
		Position:  0,
		Title:     "Alpha",
		Priority:  domain.PriorityLow,
	}, now)
	svc := newFakeService([]domain.Project{p1, p2}, []domain.Column{c1, c2}, []domain.Task{t1})
	m := loadReadyModel(t, NewModel(svc))

	m = applyMsg(t, m, keyRune('t'))
	if !m.showArchived {
		t.Fatal("expected showArchived enabled")
	}
	m = applyMsg(t, m, keyRune('t'))
	if m.showArchived {
		t.Fatal("expected showArchived disabled")
	}

	m = applyMsg(t, m, keyRune('u'))
	if !strings.Contains(m.status, "nothing to restore") {
		t.Fatalf("expected restore status, got %q", m.status)
	}

	m.searchQuery = "x"
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.searchQuery != "" {
		t.Fatalf("expected search cleared, got %q", m.searchQuery)
	}

	m = applyMsg(t, m, keyRune('P'))
	if m.selectedProject != 0 {
		t.Fatalf("expected selection unchanged in picker-open path, got %d", m.selectedProject)
	}
	if m.mode != modeProjectPicker {
		t.Fatalf("expected project picker mode, got %v", m.mode)
	}

	// out of range move left should no-op
	m.mode = modeNone
	m.selectedColumn = 0
	m = applyMsg(t, m, keyRune('['))
	if m.selectedColumn != 0 {
		t.Fatalf("expected no-op move left, got %d", m.selectedColumn)
	}
}

// TestModelBulkMoveKeysUseSelection verifies that bracket move keys apply to the full multi-selection.
func TestModelBulkMoveKeysUseSelection(t *testing.T) {
	now := time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	todo, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	progress, _ := domain.NewColumn("c2", project.ID, "In Progress", 1, 0, now)

	t1, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  todo.ID,
		Position:  0,
		Title:     "Task 1",
		Priority:  domain.PriorityMedium,
	}, now)
	t2, _ := domain.NewTask(domain.TaskInput{
		ID:        "t2",
		ProjectID: project.ID,
		ColumnID:  todo.ID,
		Position:  1,
		Title:     "Task 2",
		Priority:  domain.PriorityMedium,
	}, now)

	svc := newFakeService([]domain.Project{project}, []domain.Column{todo, progress}, []domain.Task{t1, t2})
	m := loadReadyModel(t, NewModel(svc))
	m.selectedTaskIDs = map[string]struct{}{"t1": {}, "t2": {}}

	m = applyMsg(t, m, keyRune(']'))

	moved := map[string]string{}
	for _, task := range svc.tasks[project.ID] {
		moved[task.ID] = task.ColumnID
	}
	if moved["t1"] != progress.ID || moved["t2"] != progress.ID {
		t.Fatalf("expected both selected tasks moved to progress column, got %#v", moved)
	}
}

// TestHelpersCoverage verifies behavior for the covered scenario.
func TestHelpersCoverage(t *testing.T) {
	if clamp(5, 0, 1) != 1 {
		t.Fatal("clamp upper bound failed")
	}
	if clamp(-1, 0, 1) != 0 {
		t.Fatal("clamp lower bound failed")
	}
	if clamp(0, 2, 1) != 2 {
		t.Fatal("clamp invalid range failed")
	}
	if truncate("abc", 0) != "" {
		t.Fatal("truncate max 0 failed")
	}
	if truncate("abc", 1) != "a" {
		t.Fatal("truncate max 1 failed")
	}
	if truncate("abcdef", 3) != "ab…" {
		t.Fatal("truncate ellipsis failed")
	}
	if summarizeLabels([]string{"a", "b", "c"}, 2) != "#a,#b+1" {
		t.Fatalf("unexpected label summary %q", summarizeLabels([]string{"a", "b", "c"}, 2))
	}

	m := Model{}
	if m.modeLabel() != "normal" {
		t.Fatalf("mode label mismatch: %q", m.modeLabel())
	}
	m.mode = modeAddTask
	if !strings.Contains(m.modePrompt(), "new task:") {
		t.Fatal("expected add mode prompt")
	}
	m.mode = modeSearch
	if !strings.Contains(m.modePrompt(), "search query") {
		t.Fatal("expected search mode prompt")
	}
	m.mode = modeRenameTask
	if !strings.Contains(m.modePrompt(), "rename task") {
		t.Fatal("expected rename mode prompt")
	}
	m.mode = modeEditTask
	if !strings.Contains(m.modePrompt(), "edit task:") {
		t.Fatal("expected edit mode prompt")
	}
	m.mode = modeProjectPicker
	if !strings.Contains(m.modePrompt(), "project picker") {
		t.Fatal("expected picker mode prompt")
	}
	m.mode = modeTaskInfo
	if !strings.Contains(m.modePrompt(), "task info") {
		t.Fatal("expected task info mode prompt")
	}
	m.mode = modeAddProject
	if !strings.Contains(m.modePrompt(), "new project") {
		t.Fatal("expected add project mode prompt")
	}
	m.mode = modeEditProject
	if !strings.Contains(m.modePrompt(), "edit project") {
		t.Fatal("expected edit project mode prompt")
	}
	m.mode = modeSearchResults
	if !strings.Contains(m.modePrompt(), "search results") {
		t.Fatal("expected search results mode prompt")
	}
	m.mode = modeCommandPalette
	if !strings.Contains(m.modePrompt(), "command palette") {
		t.Fatal("expected command palette mode prompt")
	}
	m.mode = modeQuickActions
	if !strings.Contains(m.modePrompt(), "quick actions") {
		t.Fatal("expected quick actions mode prompt")
	}
	m.mode = modeThread
	if !strings.Contains(m.modePrompt(), "thread:") {
		t.Fatal("expected thread mode prompt")
	}

	m.columns = []domain.Column{{ID: "c1"}}
	m.width = 10
	if m.columnWidth() < 18 {
		t.Fatal("expected minimum width")
	}
	m.width = 300
	if m.columnWidth() <= 42 {
		t.Fatal("expected width to continue expanding on wide terminals")
	}
}

// TestRenderProjectTabsAndLabels verifies project-tab rendering and label formatting helpers.
func TestRenderProjectTabsAndLabels(t *testing.T) {
	now := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)
	inbox, _ := domain.NewProject("p1", "Inbox", "", now)
	roadmap, _ := domain.NewProject("p2", "Roadmap", "", now)
	roadmap.Metadata.Icon = "+"
	archivedAt := now
	roadmap.ArchivedAt = &archivedAt

	m := Model{
		projects:        []domain.Project{inbox},
		selectedProject: 0,
	}
	if got := stripANSI(m.renderProjectTabs(lipgloss.Color("62"), lipgloss.Color("239"))); got != "" {
		t.Fatalf("expected no tabs for single project, got %q", got)
	}

	m.projects = []domain.Project{inbox, roadmap}
	m.selectedProject = 1
	got := stripANSI(m.renderProjectTabs(lipgloss.Color("62"), lipgloss.Color("239")))
	if !strings.Contains(got, "Inbox") {
		t.Fatalf("expected inactive project tab label, got %q", got)
	}
	if !strings.Contains(got, "[+ Roadmap (archived)]") {
		t.Fatalf("expected selected archived tab label, got %q", got)
	}
}

// TestNoticesPanelSpacingBudget verifies board/panel width budgeting matches measured rendered widths.
func TestNoticesPanelSpacingBudget(t *testing.T) {
	m := Model{
		columns: []domain.Column{
			{ID: "c1"},
			{ID: "c2"},
			{ID: "c3"},
		},
	}

	minBoardWidth := len(m.columns)*renderedBoardColumnWidth(minimumColumnWidth) + max(0, len(m.columns)-1)*boardColumnGapWidth
	minTotalWithPanel := minBoardWidth + noticesPanelGapWidth + renderedNoticesPanelWidth(minimumNoticesPanelWidth)

	if got := m.noticesPanelWidth(minTotalWithPanel - 1); got != 0 {
		t.Fatalf("expected notices panel hidden below clean-fit threshold, got width %d", got)
	}

	panelWidth := m.noticesPanelWidth(minTotalWithPanel)
	if panelWidth != minimumNoticesPanelWidth {
		t.Fatalf("expected minimum notices panel width at threshold, got %d", panelWidth)
	}

	boardWidth := m.boardWidthFor(minTotalWithPanel)
	columnWidth := m.columnWidthFor(boardWidth)
	boardRenderedWidth := len(m.columns)*renderedBoardColumnWidth(columnWidth) + max(0, len(m.columns)-1)*boardColumnGapWidth
	panelRenderedWidth := renderedNoticesPanelWidth(panelWidth)
	totalRenderedWidth := boardRenderedWidth + noticesPanelGapWidth + panelRenderedWidth
	if totalRenderedWidth != minTotalWithPanel {
		t.Fatalf("expected exact rendered-width budget match at threshold, got %d", totalRenderedWidth)
	}
}

// TestTaskEditParsing verifies behavior for the covered scenario.
func TestTaskEditParsing(t *testing.T) {
	now := time.Date(2026, 2, 21, 0, 0, 0, 0, time.UTC)
	current, _ := domain.NewTask(domain.TaskInput{
		ID:          "t1",
		ProjectID:   "p1",
		ColumnID:    "c1",
		Position:    0,
		Title:       "old",
		Description: "desc",
		Priority:    domain.PriorityMedium,
		DueAt:       &now,
		Labels:      []string{"x"},
	}, now)

	input, err := parseTaskEditInput("new | details | high | 2026-03-01 | a,b", current)
	if err != nil {
		t.Fatalf("parseTaskEditInput() error = %v", err)
	}
	if input.Title != "new" || input.Priority != domain.PriorityHigh {
		t.Fatalf("unexpected parsed input %#v", input)
	}
	if input.DueAt == nil || input.DueAt.Format("2006-01-02") != "2026-03-01" {
		t.Fatalf("unexpected parsed due date %#v", input.DueAt)
	}

	_, err = parseTaskEditInput("x | y | urgent | - | -", current)
	if err == nil {
		t.Fatal("expected invalid priority error")
	}
	_, err = parseTaskEditInput("x | y | low | 03/01/2026 | -", current)
	if err == nil {
		t.Fatal("expected invalid date error")
	}

	if !strings.Contains(formatTaskEditInput(current), "old") {
		t.Fatal("expected formatter to include title")
	}
}

// TestProjectPickerMouseAndWheel verifies behavior for the covered scenario.
func TestProjectPickerMouseAndWheel(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p1, _ := domain.NewProject("p1", "A", "", now)
	p2, _ := domain.NewProject("p2", "B", "", now)
	c1, _ := domain.NewColumn("c1", p1.ID, "To Do", 0, 0, now)
	c2, _ := domain.NewColumn("c2", p2.ID, "To Do", 0, 0, now)
	svc := newFakeService([]domain.Project{p1, p2}, []domain.Column{c1, c2}, nil)
	m := loadReadyModel(t, NewModel(svc))

	m = applyMsg(t, m, keyRune('p'))
	m = applyMsg(t, m, tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	if m.projectPickerIndex != 1 {
		t.Fatalf("expected wheel to move picker, got %d", m.projectPickerIndex)
	}
	m = applyMsg(t, m, tea.MouseClickMsg{X: 2, Y: 7, Button: tea.MouseLeft})
	if m.projectPickerIndex != 1 {
		t.Fatalf("expected click to target second project, got %d", m.projectPickerIndex)
	}
}

// TestTaskFieldConfigAffectsRendering verifies behavior for the covered scenario.
func TestTaskFieldConfigAffectsRendering(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	due := now.Add(24 * time.Hour)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:          "t1",
		ProjectID:   p.ID,
		ColumnID:    c.ID,
		Position:    0,
		Title:       "Task",
		Description: "detailed notes",
		Priority:    domain.PriorityHigh,
		DueAt:       &due,
		Labels:      []string{"one", "two", "three"},
	}, now)

	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})
	mDefault := loadReadyModel(t, NewModel(svc))
	meta := mDefault.cardMeta(task)
	if !strings.Contains(meta, "high") || !strings.Contains(meta, "#one,#three+1") {
		t.Fatalf("expected default card meta with priority and labels, got %q", meta)
	}

	mHidden := loadReadyModel(t, NewModel(svc, WithTaskFieldConfig(TaskFieldConfig{
		ShowPriority:    false,
		ShowDueDate:     false,
		ShowLabels:      false,
		ShowDescription: false,
	})))
	if mHidden.cardMeta(task) != "" {
		t.Fatalf("expected empty card meta when all card fields hidden, got %q", mHidden.cardMeta(task))
	}
	details := mHidden.renderTaskDetails(lipgloss.Color("212"), lipgloss.Color("245"), lipgloss.Color("241"))
	if strings.Contains(details, "priority:") || strings.Contains(details, "due:") || strings.Contains(details, "labels:") {
		t.Fatalf("expected details metadata hidden, got %q", details)
	}
	if strings.Contains(details, "detailed notes") {
		t.Fatalf("expected description hidden, got %q", details)
	}
}

// TestDeleteUsesConfiguredDefaultMode verifies behavior for the covered scenario.
func TestDeleteUsesConfiguredDefaultMode(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	t1, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "One",
		Priority:  domain.PriorityLow,
	}, now)
	t2, _ := domain.NewTask(domain.TaskInput{
		ID:        "t2",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  1,
		Title:     "Two",
		Priority:  domain.PriorityLow,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{t1, t2})

	m := loadReadyModel(t, NewModel(svc, WithDefaultDeleteMode(app.DeleteModeHard)))
	m = applyMsg(t, m, keyRune('d'))
	if m.mode != modeConfirmAction {
		t.Fatalf("expected confirm mode for default hard delete, got %v", m.mode)
	}
	m.confirmChoice = 0
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if len(svc.tasks[p.ID]) != 1 {
		t.Fatalf("expected default delete mode hard to remove selected task, got %d", len(svc.tasks[p.ID]))
	}
}

// TestParseDueAndLabelsInput verifies behavior for the covered scenario.
func TestParseDueAndLabelsInput(t *testing.T) {
	now := time.Date(2026, 2, 21, 0, 0, 0, 0, time.UTC)

	gotDue, err := parseDueInput("", &now)
	if err != nil {
		t.Fatalf("parseDueInput empty unexpected error: %v", err)
	}
	if gotDue == nil || !gotDue.Equal(now) {
		t.Fatalf("expected current due date to be preserved, got %#v", gotDue)
	}

	gotDue, err = parseDueInput("-", &now)
	if err != nil {
		t.Fatalf("parseDueInput dash unexpected error: %v", err)
	}
	if gotDue != nil {
		t.Fatalf("expected due date cleared, got %#v", gotDue)
	}

	gotDue, err = parseDueInput("2026-03-01", nil)
	if err != nil {
		t.Fatalf("parseDueInput valid unexpected error: %v", err)
	}
	if gotDue == nil || gotDue.Format("2006-01-02") != "2026-03-01" {
		t.Fatalf("expected parsed due date, got %#v", gotDue)
	}
	gotDue, err = parseDueInput("2026-03-01T15:04", nil)
	if err != nil {
		t.Fatalf("parseDueInput datetime unexpected error: %v", err)
	}
	if gotDue == nil || gotDue.In(time.Local).Hour() != 15 || gotDue.In(time.Local).Minute() != 4 {
		t.Fatalf("expected parsed due datetime, got %#v", gotDue)
	}

	if _, err = parseDueInput("03/01/2026", nil); err == nil {
		t.Fatal("expected parseDueInput invalid format error")
	}

	currentLabels := []string{"one"}
	if got := parseLabelsInput("", currentLabels); len(got) != 1 || got[0] != "one" {
		t.Fatalf("expected current labels preserved, got %#v", got)
	}
	if got := parseLabelsInput("-", currentLabels); got != nil {
		t.Fatalf("expected labels cleared with -, got %#v", got)
	}
	if got := parseLabelsInput("a, b, , c", nil); len(got) != 3 || got[0] != "a" || got[2] != "c" {
		t.Fatalf("expected parsed labels, got %#v", got)
	}

	if got := canonicalSearchStates([]string{}); len(got) != 4 || got[0] != "todo" {
		t.Fatalf("expected state fallback, got %#v", got)
	}
	if got := canonicalSearchStates([]string{"todo", "progress", "todo"}); len(got) != 2 || got[1] != "progress" {
		t.Fatalf("expected deduped state filters, got %#v", got)
	}
	if got := canonicalSearchStates([]string{"unknown"}); len(got) != 4 || got[0] != "todo" {
		t.Fatalf("expected canonical fallback for unknown states, got %#v", got)
	}
}

// TestRenderModeOverlayAndIndexHelpers verifies behavior for the covered scenario.
func TestRenderModeOverlayAndIndexHelpers(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c1, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	c2, _ := domain.NewColumn("c2", p.ID, "Done", 1, 0, now)
	t1, _ := domain.NewTask(domain.TaskInput{
		ID:          "t1",
		ProjectID:   p.ID,
		ColumnID:    c1.ID,
		Position:    0,
		Title:       "First",
		Description: "desc one",
		Priority:    domain.PriorityLow,
	}, now)
	t2, _ := domain.NewTask(domain.TaskInput{
		ID:        "t2",
		ProjectID: p.ID,
		ColumnID:  c1.ID,
		Position:  1,
		Title:     "Second",
		Priority:  domain.PriorityHigh,
	}, now)
	branch, _ := domain.NewTask(domain.TaskInput{
		ID:        "b1",
		ProjectID: p.ID,
		ColumnID:  c1.ID,
		Position:  2,
		Kind:      domain.WorkKind("branch"),
		Scope:     domain.KindAppliesToBranch,
		Title:     "Platform",
		Priority:  domain.PriorityMedium,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c1, c2}, []domain.Task{t1, t2, branch})
	m := loadReadyModel(t, NewModel(svc))

	accent := lipgloss.Color("62")
	muted := lipgloss.Color("241")
	dim := lipgloss.Color("239")
	helpStyle := lipgloss.NewStyle().Foreground(muted)
	assertSectionOrder := func(overlay string, sections []string) {
		t.Helper()
		last := -1
		for _, section := range sections {
			idx := strings.Index(overlay, section)
			if idx < 0 {
				t.Fatalf("expected overlay section %q, got %q", section, overlay)
			}
			if idx <= last {
				t.Fatalf("expected section %q after prior sections, got %q", section, overlay)
			}
			last = idx
		}
	}
	nextNonEmptyLine := func(lines []string, start int) string {
		t.Helper()
		for idx := start + 1; idx < len(lines); idx++ {
			line := strings.TrimSpace(strings.ToLower(stripANSI(lines[idx])))
			if line != "" {
				return line
			}
		}
		return ""
	}
	assertDescriptionDirectlyUnderTitle := func(lines []string, titleToken string) {
		t.Helper()
		titleToken = strings.TrimSpace(strings.ToLower(titleToken))
		for idx := range lines {
			line := strings.TrimSpace(strings.ToLower(stripANSI(lines[idx])))
			if !strings.Contains(line, titleToken) {
				continue
			}
			if next := nextNonEmptyLine(lines, idx); !strings.Contains(next, "description") {
				t.Fatalf("expected description directly under title %q, got %q", titleToken, next)
			}
			return
		}
		t.Fatalf("expected title line containing %q", titleToken)
	}
	assertSubtasksSection := func(body string) {
		t.Helper()
		if !strings.Contains(body, "subtasks") {
			t.Fatalf("expected subtasks section, got %q", body)
		}
		if !strings.Contains(body, "0/0") && !strings.Contains(body, "no subtasks") && !strings.Contains(body, "(none)") {
			t.Fatalf("expected subtasks empty-state hint, got %q", body)
		}
	}
	assertInfoMetadataSplitLines := func(lines []string) {
		t.Helper()
		clean := make([]string, 0, len(lines))
		for _, raw := range lines {
			clean = append(clean, strings.TrimSpace(strings.ToLower(stripANSI(raw))))
		}
		priorityLine := -1
		dueLine := -1
		labelsLine := -1
		for idx, line := range clean {
			switch {
			case strings.HasPrefix(line, "priority:"):
				if priorityLine < 0 {
					priorityLine = idx
				}
			case strings.HasPrefix(line, "due:"):
				if dueLine < 0 {
					dueLine = idx
				}
			case strings.HasPrefix(line, "labels:"):
				if labelsLine < 0 {
					labelsLine = idx
				}
			}
		}
		if priorityLine < 0 || dueLine < 0 || labelsLine < 0 {
			t.Fatalf("expected separate priority/due/labels lines, got %#v", clean)
		}
		if strings.Contains(clean[priorityLine], "due:") || strings.Contains(clean[priorityLine], "labels:") {
			t.Fatalf("expected priority line to contain only priority metadata, got %q", clean[priorityLine])
		}
		if strings.Contains(clean[dueLine], "priority:") || strings.Contains(clean[dueLine], "labels:") {
			t.Fatalf("expected due line to contain only due metadata, got %q", clean[dueLine])
		}
		if strings.Contains(clean[labelsLine], "priority:") || strings.Contains(clean[labelsLine], "due:") {
			t.Fatalf("expected labels line to contain only labels metadata, got %q", clean[labelsLine])
		}
	}

	projectPicker := m
	projectPicker.mode = modeProjectPicker
	projectPicker.projectPickerIndex = 0
	if out := projectPicker.renderModeOverlay(accent, muted, dim, helpStyle, 80); !strings.Contains(out, "Projects") {
		t.Fatalf("expected project picker overlay, got %q", out)
	}

	addMode := m
	_ = addMode.startTaskForm(nil)
	if out := addMode.renderModeOverlay(accent, muted, dim, helpStyle, 80); !strings.Contains(out, "New Task") || !strings.Contains(out, "title:") {
		t.Fatalf("expected add-task overlay with fields, got %q", out)
	}
	if out := addMode.renderModeOverlay(accent, muted, dim, helpStyle, 80); strings.Contains(out, "fields: title") {
		t.Fatalf("expected simplified modal hints without repeated fields legend, got %q", out)
	}

	task, ok := m.selectedTaskInCurrentColumn()
	if !ok {
		t.Fatal("expected selected task")
	}
	editMode := m
	_ = editMode.startTaskForm(&task)
	if out := editMode.renderModeOverlay(accent, muted, dim, helpStyle, 80); !strings.Contains(out, "Edit Task") {
		t.Fatalf("expected edit overlay, got %q", out)
	} else {
		if !strings.Contains(out, "scroll:") {
			t.Fatalf("expected shared node modal scroll indicator in edit overlay, got %q", out)
		}
		if !strings.Contains(out, "mode: edit") || !strings.Contains(strings.ToLower(out), "kind: task") {
			t.Fatalf("expected edit header metadata line, got %q", out)
		}
	}
	boxWidth := taskInfoOverlayBoxWidth(80)
	contentWidth := max(24, boxWidth-8)
	editBody, _ := editMode.taskFormBodyLines(contentWidth, lipgloss.NewStyle().Foreground(muted), accent)
	editBodyText := strings.ToLower(stripANSI(strings.Join(editBody, "\n")))
	assertSectionOrder(editBodyText, []string{"title:", "description:", "subtasks:", "dependencies:", "comments (", "resources:"})
	assertDescriptionDirectlyUnderTitle(editBody, "title:")
	assertSubtasksSection(editBodyText)
	if strings.Contains(editBodyText, "enter or e opens full markdown editor") || strings.Contains(editBodyText, "left/right changes selected row") {
		t.Fatalf("expected edit body to remove simple inline help text, got %q", editBodyText)
	}
	if strings.Contains(editBodyText, "effective labels") || strings.Contains(editBodyText, "inherited labels") {
		t.Fatalf("expected edit body to hide inherited/effective labels block, got %q", editBodyText)
	}
	if strings.Contains(editBodyText, "csv task") {
		t.Fatalf("expected dependency rows to stop advertising inline csv entry, got %q", editBodyText)
	}
	addBranchMode := m
	_ = addBranchMode.startBranchForm(nil)
	if out := addBranchMode.renderModeOverlay(accent, muted, dim, helpStyle, 80); !strings.Contains(out, "New Branch") {
		t.Fatalf("expected branch add overlay title, got %q", out)
	}
	editBranchMode := m
	_ = editBranchMode.startTaskForm(&branch)
	if out := editBranchMode.renderModeOverlay(accent, muted, dim, helpStyle, 80); !strings.Contains(out, "Edit Branch") {
		t.Fatalf("expected branch edit overlay title, got %q", out)
	}

	searchMode := m
	_ = searchMode.startSearchMode()
	if out := searchMode.renderModeOverlay(accent, muted, dim, helpStyle, 80); !strings.Contains(out, "Search") {
		t.Fatalf("expected search overlay, got %q", out)
	}
	searchMode.mode = modeSearchResults
	searchMode.searchMatches = []app.TaskMatch{{Project: p, Task: t1, StateID: "todo"}}
	if out := searchMode.renderModeOverlay(accent, muted, dim, helpStyle, 80); !strings.Contains(out, "Search Results") {
		t.Fatalf("expected search-results overlay, got %q", out)
	}

	renameMode := m
	renameMode.mode = modeRenameTask
	renameMode.input = "rename me"
	if out := renameMode.renderModeOverlay(accent, muted, dim, helpStyle, 80); !strings.Contains(out, "Rename Task") {
		t.Fatalf("expected rename overlay, got %q", out)
	}
	infoMode := m
	infoMode.mode = modeTaskInfo
	if out := stripANSI(fmt.Sprint(infoMode.renderFullPageNodeModeView().Content)); !strings.Contains(out, "Task Info") {
		t.Fatalf("expected task info full-page view, got %q", out)
	} else {
		if !strings.Contains(out, "scroll:") {
			t.Fatalf("expected shared node modal scroll indicator in task info view, got %q", out)
		}
		if !strings.Contains(out, "mode: info") || !strings.Contains(strings.ToLower(out), "kind: task") {
			t.Fatalf("expected info header metadata line, got %q", out)
		}
	}
	infoBody := infoMode.taskInfoBodyLines(task, boxWidth, contentWidth, lipgloss.NewStyle().Foreground(muted))
	infoBodyText := strings.ToLower(stripANSI(strings.Join(infoBody, "\n")))
	assertSectionOrder(infoBodyText, []string{strings.ToLower(task.Title), "description:", "subtasks (", "dependencies:", "comments (", "resources:"})
	assertDescriptionDirectlyUnderTitle(infoBody, strings.ToLower(task.Title))
	assertSubtasksSection(infoBodyText)
	assertInfoMetadataSplitLines(infoBody)
	if strings.Contains(infoBodyText, "effective labels") || strings.Contains(infoBodyText, "inherited labels") {
		t.Fatalf("expected task-info body to hide inherited/effective labels block, got %q", infoBodyText)
	}
	branchInfoMode := m
	if !branchInfoMode.openTaskInfo(branch.ID, "task info") {
		t.Fatal("expected branch task info mode")
	}
	if out := stripANSI(fmt.Sprint(branchInfoMode.renderFullPageNodeModeView().Content)); !strings.Contains(out, "Branch Info") {
		t.Fatalf("expected branch info full-page title, got %q", out)
	}

	projectMode := m
	_ = projectMode.startProjectForm(nil)
	if out := projectMode.renderModeOverlay(accent, muted, dim, helpStyle, 80); !strings.Contains(out, "New Project") {
		t.Fatalf("expected project overlay, got %q", out)
	}

	commandMode := m
	_ = commandMode.startCommandPalette()
	if out := commandMode.renderModeOverlay(accent, muted, dim, helpStyle, 80); !strings.Contains(out, "Command Palette") {
		t.Fatalf("expected command palette overlay, got %q", out)
	}

	actionMode := m
	_ = actionMode.startQuickActions()
	if out := actionMode.renderModeOverlay(accent, muted, dim, helpStyle, 80); !strings.Contains(out, "Quick Actions") {
		t.Fatalf("expected quick actions overlay, got %q", out)
	}

	tasks := m.currentColumnTasks()
	if idx := m.taskIndexAtRow(tasks, 0); idx != 0 {
		t.Fatalf("expected row 0 => task 0, got %d", idx)
	}
	if idx := m.taskIndexAtRow(tasks, 3); idx != 1 {
		t.Fatalf("expected row 3 => task 1, got %d", idx)
	}
	if idx := m.taskIndexAtRow(tasks, 99); idx != 2 {
		t.Fatalf("expected large row => last task, got %d", idx)
	}

	panelWithSelection := m.renderOverviewPanel(p, accent, muted, dim, 30, 24, 0, 0, 0, nil, false)
	if !strings.Contains(panelWithSelection, "Selection") {
		t.Fatalf("expected overview panel selection section, got %q", panelWithSelection)
	}
	noneSelected := m
	noneSelected.selectedColumn = 1
	panelWithoutSelection := noneSelected.renderOverviewPanel(p, accent, muted, dim, 30, 24, 0, 0, 0, nil, false)
	if !strings.Contains(panelWithoutSelection, "no task selected") {
		t.Fatalf("expected overview panel no-selection hint, got %q", panelWithoutSelection)
	}
}

// TestModelFullPageNodeViewShowsHeaderAndBorder verifies full-page info/edit keep the TILLSYN header and node border.
func TestModelFullPageNodeViewShowsHeaderAndBorder(t *testing.T) {
	now := time.Date(2026, 3, 5, 8, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})
	m := loadReadyModel(t, NewModel(svc))

	m = applyMsg(t, m, keyRune('i'))
	if m.mode != modeTaskInfo {
		t.Fatalf("expected task-info mode, got %v", m.mode)
	}
	infoView := stripANSI(fmt.Sprint(m.View().Content))
	if !strings.Contains(infoView, "TILLSYN") || !strings.Contains(infoView, "Task Info") {
		t.Fatalf("expected task-info view to include TILLSYN header + task info title, got\n%s", infoView)
	}
	if !(strings.Contains(infoView, "┌") && strings.Contains(infoView, "┐") && strings.Contains(infoView, "│")) {
		t.Fatalf("expected bordered full-page node surface in task-info view, got\n%s", infoView)
	}

	m = applyMsg(t, m, keyRune('e'))
	if m.mode != modeEditTask {
		t.Fatalf("expected edit-task mode, got %v", m.mode)
	}
	editView := stripANSI(fmt.Sprint(m.View().Content))
	if !strings.Contains(editView, "TILLSYN") || !strings.Contains(editView, "Edit Task") {
		t.Fatalf("expected edit-task view to include TILLSYN header + edit title, got\n%s", editView)
	}
	if !(strings.Contains(editView, "┌") && strings.Contains(editView, "┐") && strings.Contains(editView, "│")) {
		t.Fatalf("expected bordered full-page node surface in edit view, got\n%s", editView)
	}
}

// TestTaskDescriptionPreviewHeightMatchesBetweenInfoAndEdit verifies info/edit use the same description-preview sizing contract.
func TestTaskDescriptionPreviewHeightMatchesBetweenInfoAndEdit(t *testing.T) {
	now := time.Date(2026, 3, 13, 19, 30, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:          "t1",
		ProjectID:   project.ID,
		ColumnID:    column.ID,
		Position:    0,
		Title:       "Task",
		Priority:    domain.PriorityMedium,
		Description: "## Overview\n\n- item one\n- item two\n- a much longer bullet that wraps over multiple lines inside the shared preview surface",
	}, now)
	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{project}, []domain.Column{column}, []domain.Task{task})))

	m = applyMsg(t, m, keyRune('i'))
	if m.mode != modeTaskInfo {
		t.Fatalf("expected task-info mode, got %v", m.mode)
	}
	infoHeight := m.taskInfoDetails.Height()
	if infoHeight <= 0 {
		t.Fatalf("expected positive task-info preview height, got %d", infoHeight)
	}

	m = applyMsg(t, m, keyRune('e'))
	if m.mode != modeEditTask {
		t.Fatalf("expected edit-task mode, got %v", m.mode)
	}

	accent := projectAccentColor(project)
	metrics := m.fullPageSurfaceMetrics(
		accent,
		lipgloss.Color("241"),
		lipgloss.Color("239"),
		taskInfoOverlayBoxWidth(max(0, m.fullPageNodeContentWidth())),
		"Edit Task",
		m.taskFormHeaderMeta(),
		"",
	)
	editPreview := m.taskDescriptionPreviewViewportForContentWidth(m.taskFormDescription, metrics.contentWidth)
	if got := editPreview.Height(); got != infoHeight {
		t.Fatalf("expected edit preview height %d to match info preview height %d", got, infoHeight)
	}
}

// TestFullPageSurfaceMetricsUseBoardMatchedOuterGaps verifies shared full-page surfaces do not reserve extra top/bottom spacer rows.
func TestFullPageSurfaceMetricsUseBoardMatchedOuterGaps(t *testing.T) {
	now := time.Date(2026, 3, 13, 19, 40, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{project}, []domain.Column{column}, []domain.Task{task})))

	accent := projectAccentColor(project)
	metrics := m.fullPageSurfaceMetrics(
		accent,
		lipgloss.Color("241"),
		lipgloss.Color("239"),
		taskInfoOverlayBoxWidth(max(0, m.fullPageNodeContentWidth())),
		"Task Info",
		m.taskInfoHeaderMeta(task),
		"",
	)
	if metrics.topGapY != 0 || metrics.bottomGapY != 0 {
		t.Fatalf("expected board-matched full-page outer gaps (0/0), got top=%d bottom=%d", metrics.topGapY, metrics.bottomGapY)
	}
}

// TestModelFormValidationPaths verifies behavior for the covered scenario.
func TestModelFormValidationPaths(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	existing, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Existing",
		Priority:  domain.PriorityLow,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{existing})
	m := loadReadyModel(t, NewModel(svc))

	// Add mode: invalid priority branch.
	m = applyMsg(t, m, keyRune('n'))
	m.formInputs[0].SetValue("Draft roadmap")
	m.formInputs[2].SetValue("urgent")
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if !strings.Contains(m.status, "priority must be low|medium|high") {
		t.Fatalf("expected invalid priority status, got %q", m.status)
	}

	// Add mode: invalid due date branch.
	m.formInputs[2].SetValue("high")
	m.formInputs[3].SetValue("03/01/2026")
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if !strings.Contains(m.status, "due date must be YYYY-MM-DD") {
		t.Fatalf("expected invalid due status, got %q", m.status)
	}

	// Add mode: success path.
	m.formInputs[3].SetValue("2026-03-01")
	m.formInputs[4].SetValue("planning,till")
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if len(svc.tasks[p.ID]) != 2 {
		t.Fatalf("expected create task success, got %d tasks", len(svc.tasks[p.ID]))
	}

	// Edit mode: invalid priority branch.
	m.selectedTask = 0
	m = applyMsg(t, m, keyRune('e'))
	m.formInputs[2].SetValue("invalid")
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if !strings.Contains(m.status, "priority must be low|medium|high") {
		t.Fatalf("expected invalid edit priority status, got %q", m.status)
	}
}

// TestTaskInfoModeAndPriorityPicker verifies behavior for the covered scenario.
func TestTaskInfoModeAndPriorityPicker(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})
	m := loadReadyModel(t, NewModel(svc))

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeTaskInfo {
		t.Fatalf("expected enter to open task info mode, got %v", m.mode)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.mode != modeNone {
		t.Fatalf("expected esc to close task info mode, got %v", m.mode)
	}

	m = applyMsg(t, m, keyRune('i'))
	if m.mode != modeTaskInfo {
		t.Fatalf("expected task info mode, got %v", m.mode)
	}
	m = applyMsg(t, m, keyRune('e'))
	if m.mode != modeEditTask {
		t.Fatalf("expected edit mode from task info, got %v", m.mode)
	}

	m.formFocus = 2
	before := m.formInputs[2].Value()
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyRight})
	after := m.formInputs[2].Value()
	if before == after {
		t.Fatalf("expected priority picker value to change, still %q", after)
	}
	changed := m.formInputs[2].Value()
	m = applyMsg(t, m, keyRune('x'))
	if m.formInputs[2].Value() != changed {
		t.Fatalf("expected typing ignored on priority picker, got %q", m.formInputs[2].Value())
	}
}

// TestTaskFormDependencyPlaceholdersUseCSVTask verifies dependency placeholders use `csv task`.
func TestTaskFormDependencyPlaceholdersUseCSVTask(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})
	m := loadReadyModel(t, NewModel(svc))

	assertDependencyPlaceholders := func(mode Model, scope string) {
		t.Helper()
		for _, field := range []int{taskFieldDependsOn, taskFieldBlockedBy} {
			placeholder := strings.TrimSpace(strings.ToLower(mode.formInputs[field].Placeholder))
			if placeholder != "csv task" {
				t.Fatalf("expected %s placeholder %d to be %q, got %q", scope, field, "csv task", placeholder)
			}
			if strings.Contains(placeholder, "ids") {
				t.Fatalf("expected %s placeholder %d to avoid ids wording, got %q", scope, field, placeholder)
			}
		}
	}

	addMode := m
	_ = addMode.startTaskForm(nil)
	assertDependencyPlaceholders(addMode, "add-task")

	editMode := m
	_ = editMode.startTaskForm(&task)
	assertDependencyPlaceholders(editMode, "edit-task")
}

// TestModelEditTaskKeyboardSaveAndPickerShortcuts verifies edit-mode save, picker/editor shortcuts, and wrap-around navigation.
func TestModelEditTaskKeyboardSaveAndPickerShortcuts(t *testing.T) {
	now := time.Date(2026, 2, 25, 9, 30, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "task-1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})
	m := loadReadyModel(t, NewModel(svc))

	m = applyMsg(t, m, keyRune('e'))
	if m.mode != modeEditTask {
		t.Fatalf("expected edit-task mode, got %v", m.mode)
	}
	m.formInputs[taskFieldTitle].SetValue("Task Saved")
	m = applyMsg(t, m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	if m.mode != modeNone {
		t.Fatalf("expected ctrl+s to save and close edit form, got %v", m.mode)
	}
	if svc.createTaskCalls != 0 {
		t.Fatalf("expected ctrl+s to avoid subtask-create path, got createTaskCalls=%d", svc.createTaskCalls)
	}
	updated, ok := svc.taskByID(task.ID)
	if !ok {
		t.Fatal("expected updated task in fake service")
	}
	if updated.Title != "Task Saved" {
		t.Fatalf("expected ctrl+s save to persist title, got %q", updated.Title)
	}

	m = applyMsg(t, m, keyRune('e'))
	if m.mode != modeEditTask {
		t.Fatalf("expected re-opened edit-task mode, got %v", m.mode)
	}

	m.formFocus = taskFieldDescription
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeDescriptionEditor {
		t.Fatalf("expected enter on description to open editor, got %v", m.mode)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.mode != modeEditTask {
		t.Fatalf("expected esc to return from description editor to edit mode, got %v", m.mode)
	}

	m.formFocus = taskFieldDescription
	m = applyMsg(t, m, keyRune('e'))
	if m.mode != modeDescriptionEditor {
		t.Fatalf("expected e on description to open editor, got %v", m.mode)
	}
	if got := m.descriptionEditorInput.Value(); got != "" {
		t.Fatalf("expected e-opened description editor to avoid seed rune injection, got %q", got)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.mode != modeEditTask {
		t.Fatalf("expected esc to return from description editor after e, got %v", m.mode)
	}

	m.formFocus = taskFieldLabels
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeLabelPicker {
		t.Fatalf("expected enter on labels to open picker, got %v", m.mode)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.mode != modeEditTask {
		t.Fatalf("expected esc from label picker to return edit mode, got %v", m.mode)
	}

	_ = m.focusTaskFormField(taskFieldLabels)
	m = applyMsg(t, m, keyRune('e'))
	if m.mode != modeLabelPicker {
		t.Fatalf("expected e on labels to open picker, got %v", m.mode)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.mode != modeEditTask {
		t.Fatalf("expected esc from label picker opened by e to return edit mode, got %v", m.mode)
	}

	m.setTaskFormMarkdownDraft(taskFieldObjective, "existing objective", true)
	m.formFocus = taskFieldObjective
	m = applyMsg(t, m, keyRune('e'))
	if m.mode != modeDescriptionEditor {
		t.Fatalf("expected e on objective to open editor, got %v", m.mode)
	}
	if got := m.descriptionEditorInput.Value(); got != "existing objective" {
		t.Fatalf("expected e-opened objective editor to avoid seed rune injection, got %q", got)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.mode != modeEditTask {
		t.Fatalf("expected esc from objective editor to return edit mode, got %v", m.mode)
	}

	_ = m.focusTaskFormField(taskFieldDue)
	beforeDueText := m.formInputs[taskFieldDue].Value()
	m = applyMsg(t, m, keyRune('d'))
	if m.mode != modeEditTask {
		t.Fatalf("expected d in edit due field to keep edit mode, got %v", m.mode)
	}
	if got := m.formInputs[taskFieldDue].Value(); got != beforeDueText {
		t.Fatalf("expected due field to stay modal-only and ignore typed d, got %q", got)
	}

	m.formFocus = taskFieldResources
	m = applyMsg(t, m, tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
	if m.mode != modeEditTask {
		t.Fatalf("expected ctrl+r in edit mode to avoid resource picker shortcut, got %v", m.mode)
	}

	m.formFocus = taskFieldDependsOn
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeDependencyInspector {
		t.Fatalf("expected enter on depends_on to open dependency inspector, got %v", m.mode)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.mode != modeEditTask {
		t.Fatalf("expected esc from dependency inspector to return edit mode, got %v", m.mode)
	}

	_ = m.focusTaskFormField(taskFieldDependsOn)
	beforeDepends := m.formInputs[taskFieldDependsOn].Value()
	m = applyMsg(t, m, keyRune('e'))
	if m.mode != modeDependencyInspector {
		t.Fatalf("expected e in depends_on field to open dependency inspector, got %v", m.mode)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.mode != modeEditTask {
		t.Fatalf("expected esc from dependency inspector to return edit mode, got %v", m.mode)
	}
	if got := m.formInputs[taskFieldDependsOn].Value(); got != beforeDepends {
		t.Fatalf("expected dependency row to remain modal-only, got %q", got)
	}

	m.formFocus = taskFieldTitle
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
	if m.formFocus != taskFieldResources {
		t.Fatalf("expected up at top field to wrap to bottom field, got %d", m.formFocus)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	if m.formFocus != taskFieldTitle {
		t.Fatalf("expected down at bottom field to wrap to top field, got %d", m.formFocus)
	}

	m.formFocus = taskFieldTitle
	beforeTitle := m.formInputs[taskFieldTitle].Value()
	m = applyMsg(t, m, keyRune('k'))
	if m.formFocus != taskFieldTitle {
		t.Fatalf("expected k in title field to remain on title, got %d", m.formFocus)
	}
	if got := m.formInputs[taskFieldTitle].Value(); got != beforeTitle+"k" {
		t.Fatalf("expected k typed into title field, got %q", got)
	}
	beforeTitle = m.formInputs[taskFieldTitle].Value()
	m = applyMsg(t, m, keyRune('j'))
	if m.formFocus != taskFieldTitle {
		t.Fatalf("expected j in title field to remain on title, got %d", m.formFocus)
	}
	if got := m.formInputs[taskFieldTitle].Value(); got != beforeTitle+"j" {
		t.Fatalf("expected j typed into title field, got %q", got)
	}

	m.formFocus = taskFieldSubtasks
	m = applyMsg(t, m, keyRune('e'))
	if m.mode != modeAddTask {
		t.Fatalf("expected e on subtasks section to open subtask form, got %v", m.mode)
	}
	if m.taskFormKind != domain.WorkKindSubtask {
		t.Fatalf("expected subtask kind from subtasks section action, got %q", m.taskFormKind)
	}
}

// TestModelEditTaskFocusScrollUsesRenderedRows verifies edit-mode focus scrolling follows rendered wrapped rows.
func TestModelEditTaskFocusScrollUsesRenderedRows(t *testing.T) {
	now := time.Date(2026, 3, 13, 17, 12, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	longMarkdown := strings.Repeat("wrapped markdown content for viewport scrolling\n", 20)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:          "task-1",
		ProjectID:   p.ID,
		ColumnID:    c.ID,
		Position:    0,
		Title:       "Task",
		Description: longMarkdown,
		Priority:    domain.PriorityMedium,
		Metadata: domain.TaskMetadata{
			BlockedReason:      longMarkdown,
			Objective:          longMarkdown,
			AcceptanceCriteria: longMarkdown,
			ValidationPlan:     longMarkdown,
			RiskNotes:          "final risk note",
		},
	}, now)
	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})))
	m = applyMsg(t, m, tea.WindowSizeMsg{Width: 88, Height: 22})

	m = applyMsg(t, m, keyRune('e'))
	if m.mode != modeEditTask {
		t.Fatalf("expected edit-task mode, got %v", m.mode)
	}
	if got := m.taskInfoBody.YOffset(); got != 0 {
		t.Fatalf("expected edit form to start at top, got y offset=%d", got)
	}

	for m.formFocus != taskFieldRiskNotes {
		m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	}
	if got := m.taskInfoBody.YOffset(); got <= 0 {
		t.Fatalf("expected risk_notes focus to scroll viewport downward, got y offset=%d", got)
	}
}

// TestModelEditTaskSubtaskAndResourceRowSelection verifies edit-mode row selection for subtasks/resources.
func TestModelEditTaskSubtaskAndResourceRowSelection(t *testing.T) {
	now := time.Date(2026, 3, 5, 10, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	parent, _ := domain.NewTask(domain.TaskInput{
		ID:        "task-parent",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Parent",
		Priority:  domain.PriorityMedium,
		Metadata: domain.TaskMetadata{
			ResourceRefs: []domain.ResourceRef{{
				ResourceType: domain.ResourceTypeLocalFile,
				PathMode:     domain.PathModeRelative,
				Location:     "docs/plan.md",
			}},
		},
	}, now)
	subtask, _ := domain.NewTask(domain.TaskInput{
		ID:        "task-child",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  1,
		ParentID:  parent.ID,
		Kind:      domain.WorkKindSubtask,
		Scope:     domain.KindAppliesToSubtask,
		Title:     "Child",
		Priority:  domain.PriorityLow,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{parent, subtask})
	m := loadReadyModel(t, NewModel(svc))
	m.projectRoots = map[string]string{"inbox": "/tmp"}

	m.focusTaskByID(parent.ID)
	m = applyMsg(t, m, keyRune('e'))
	if m.mode != modeEditTask {
		t.Fatalf("expected edit mode, got %v", m.mode)
	}

	m.formFocus = taskFieldSubtasks
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyRight})
	if m.taskFormSubtaskCursor != 1 {
		t.Fatalf("expected subtask cursor on first existing row, got %d", m.taskFormSubtaskCursor)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeEditTask {
		t.Fatalf("expected selected subtask to open in edit mode, got %v", m.mode)
	}
	if m.editingTaskID != subtask.ID {
		t.Fatalf("expected selected subtask %q to open, got %q", subtask.ID, m.editingTaskID)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.mode != modeEditTask {
		t.Fatalf("expected esc to reopen parent edit form, got %v", m.mode)
	}
	if m.editingTaskID != parent.ID {
		t.Fatalf("expected parent edit task %q after esc, got %q", parent.ID, m.editingTaskID)
	}
	if m.taskFormSubtaskCursor != 1 {
		t.Fatalf("expected parent edit to reselect child row, got %d", m.taskFormSubtaskCursor)
	}
	m.formFocus = taskFieldResources
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyRight})
	if m.taskFormResourceCursor != 1 {
		t.Fatalf("expected resource cursor on first existing row, got %d", m.taskFormResourceCursor)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeResourcePicker {
		t.Fatalf("expected enter on selected resource row to open resource picker, got %v", m.mode)
	}
	if m.taskFormResourceEditIndex != 0 {
		t.Fatalf("expected selected resource row to set edit index 0, got %d", m.taskFormResourceEditIndex)
	}
}

// TestModelAddTaskActionRowsRequireSaveFirst verifies save-dependent rows stay explicit in the new-task form.
func TestModelAddTaskActionRowsRequireSaveFirst(t *testing.T) {
	now := time.Date(2026, 3, 17, 17, 5, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{p}, []domain.Column{c}, nil)))

	m = applyMsg(t, m, keyRune('n'))
	if m.mode != modeAddTask {
		t.Fatalf("expected add task mode, got %v", m.mode)
	}

	m.formFocus = taskFieldSubtasks
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeAddTask || !strings.Contains(m.status, "save task first") {
		t.Fatalf("expected save-first subtask gate in add task, got mode=%v status=%q", m.mode, m.status)
	}

	m.formFocus = taskFieldComments
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeAddTask || !strings.Contains(m.status, "save task first") {
		t.Fatalf("expected save-first comments gate in add task, got mode=%v status=%q", m.mode, m.status)
	}

	m.formFocus = taskFieldResources
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeAddTask || !strings.Contains(m.status, "save task first") {
		t.Fatalf("expected save-first resources gate in add task, got mode=%v status=%q", m.mode, m.status)
	}
}

// TestModelEditTaskQuickActionsRespectFocusedResources verifies dot opens contextual quick actions for focused action rows.
func TestModelEditTaskQuickActionsRespectFocusedResources(t *testing.T) {
	now := time.Date(2026, 3, 17, 17, 15, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	root := t.TempDir()
	m := loadReadyModel(t, NewModel(
		newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task}),
		WithSearchRoots([]string{root}),
	))

	m = applyMsg(t, m, keyRune('e'))
	if m.mode != modeEditTask {
		t.Fatalf("expected edit task mode, got %v", m.mode)
	}
	m = applyCmd(t, m, m.focusTaskFormField(taskFieldResources))
	m = applyMsg(t, m, keyRune('.'))
	if m.mode != modeQuickActions {
		t.Fatalf("expected focused quick actions from resources row, got %v", m.mode)
	}
	if title := stripANSI(m.renderModeOverlay(lipgloss.Color("62"), lipgloss.Color("241"), lipgloss.Color("239"), lipgloss.NewStyle(), 80)); !strings.Contains(title, "Quick Actions: Resources") {
		t.Fatalf("expected contextual quick-actions title, got\n%s", title)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeResourcePicker {
		t.Fatalf("expected enter on focused quick action to open resource picker, got %v", m.mode)
	}
}

// TestModelCommentOwnerLabelUsesConfiguredIdentityFallback verifies legacy local-user labels render the configured display name.
func TestModelCommentOwnerLabelUsesConfiguredIdentityFallback(t *testing.T) {
	m := Model{identityActorID: "user-uuid", identityDisplayName: "Evan"}
	comment := domain.Comment{
		ActorID:   "tillsyn-user",
		ActorName: "tillsyn-user",
		ActorType: domain.ActorTypeUser,
	}
	if got := m.commentOwnerLabel(comment); got != "Evan" {
		t.Fatalf("expected legacy local-user label to render configured display name, got %q", got)
	}

	comment = domain.Comment{
		ActorID:   "user-uuid",
		ActorName: "",
		ActorType: domain.ActorTypeUser,
	}
	if got := m.commentOwnerLabel(comment); got != "Evan" {
		t.Fatalf("expected local actor id fallback to render configured display name, got %q", got)
	}
}

// TestModelEditTaskSubtaskSubmitReturnsToParent verifies saving a child edit reopens the parent edit flow.
func TestModelEditTaskSubtaskSubmitReturnsToParent(t *testing.T) {
	now := time.Date(2026, 3, 17, 16, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	parent, _ := domain.NewTask(domain.TaskInput{
		ID:        "task-parent",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Parent",
		Priority:  domain.PriorityMedium,
	}, now)
	child, _ := domain.NewTask(domain.TaskInput{
		ID:        "task-child",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  1,
		ParentID:  parent.ID,
		Kind:      domain.WorkKindSubtask,
		Scope:     domain.KindAppliesToSubtask,
		Title:     "Child",
		Priority:  domain.PriorityLow,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{parent, child})
	m := loadReadyModel(t, NewModel(svc))

	m.focusTaskByID(parent.ID)
	m = applyMsg(t, m, keyRune('e'))
	m.formFocus = taskFieldSubtasks
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyRight})
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.editingTaskID != child.ID {
		t.Fatalf("expected child edit form, got %q", m.editingTaskID)
	}

	m.formInputs[taskFieldTitle].SetValue("Child updated")
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeEditTask {
		t.Fatalf("expected parent edit mode after child save, got %v", m.mode)
	}
	if m.editingTaskID != parent.ID {
		t.Fatalf("expected parent edit task %q after child save, got %q", parent.ID, m.editingTaskID)
	}
	if m.taskFormSubtaskCursor != 1 {
		t.Fatalf("expected parent edit to reselect saved child row, got %d", m.taskFormSubtaskCursor)
	}
	updated, ok := svc.taskByID(child.ID)
	if !ok {
		t.Fatalf("expected updated child task %q in fake service", child.ID)
	}
	if updated.Title != "Child updated" {
		t.Fatalf("expected saved child title %q, got %q", "Child updated", updated.Title)
	}
}

// TestTaskInfoEscFromDirectChildClosesWithoutAncestorJump verifies esc closes direct child task-info without jumping to ancestors.
func TestTaskInfoEscFromDirectChildClosesWithoutAncestorJump(t *testing.T) {
	now := time.Date(2026, 3, 3, 11, 30, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	parent, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-parent",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Kind:      domain.WorkKindTask,
		Title:     "Parent Task",
		Priority:  domain.PriorityMedium,
	}, now)
	child, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-child",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  1,
		ParentID:  parent.ID,
		Kind:      domain.WorkKindSubtask,
		Title:     "Child Subtask",
		Priority:  domain.PriorityLow,
	}, now)
	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{project}, []domain.Column{column}, []domain.Task{parent, child})))

	if !m.openTaskInfo(child.ID, "task info") {
		t.Fatal("expected openTaskInfo(child) to succeed")
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.mode != modeNone {
		t.Fatalf("expected esc to close direct child task info, got %v", m.mode)
	}
	if got := strings.TrimSpace(m.taskInfoTaskID); got != "" {
		t.Fatalf("expected closed task info to clear active task id, got %q", got)
	}
}

// TestTaskFormDuePickerFlow verifies behavior for the covered scenario.
func TestTaskFormDuePickerFlow(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, nil)
	m := loadReadyModel(t, NewModel(svc))

	m = applyMsg(t, m, keyRune('n'))
	if m.mode != modeAddTask {
		t.Fatalf("expected add task mode, got %v", m.mode)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	if m.formFocus != taskFieldDue {
		t.Fatalf("expected due field focus, got %d", m.formFocus)
	}
	dueOverlay := m.renderModeOverlay(lipgloss.Color("62"), lipgloss.Color("241"), lipgloss.Color("239"), lipgloss.NewStyle(), 96)
	if strings.Contains(dueOverlay, "YYYY-MM-DD HH:MM") || strings.Contains(dueOverlay, "RFC3339") || strings.Contains(dueOverlay, "local time") {
		t.Fatalf("expected due field to hide inline datetime format hints, got %q", dueOverlay)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeDuePicker {
		t.Fatalf("expected due picker mode, got %v", m.mode)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeAddTask {
		t.Fatalf("expected return to add task mode, got %v", m.mode)
	}
	if got := strings.TrimSpace(m.formInputs[taskFieldDue].Value()); got != "-" {
		t.Fatalf("expected due field to be '-', got %q", got)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m = applyMsg(t, m, keyRune('j'))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	due := strings.TrimSpace(m.formInputs[taskFieldDue].Value())
	if len(due) != 10 || strings.Count(due, "-") != 2 {
		t.Fatalf("expected YYYY-MM-DD due value, got %q", due)
	}
}

// TestDuePickerTypedInputAndFocusControls verifies typed date/time options and focus transitions.
func TestDuePickerTypedInputAndFocusControls(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{p}, []domain.Column{c}, nil)))

	m = applyMsg(t, m, keyRune('n'))
	if m.mode != modeAddTask {
		t.Fatalf("expected add task mode, got %v", m.mode)
	}
	m.formFocus = taskFieldDue
	m.startDuePicker()
	if m.mode != modeDuePicker {
		t.Fatalf("expected due picker mode, got %v", m.mode)
	}

	m.duePickerIncludeTime = true
	m.duePickerDateInput.SetValue(time.Now().In(time.Local).Format("2006-01-02"))
	m.duePickerTimeInput.SetValue("17:45")
	options := m.duePickerOptions()
	if len(options) == 0 || !strings.Contains(strings.ToLower(options[0].Label), "use typed datetime") {
		t.Fatalf("expected typed datetime option at top, got %#v", options)
	}
	if !strings.HasSuffix(options[0].Value, "17:45") {
		t.Fatalf("expected typed datetime value to include 17:45, got %q", options[0].Value)
	}

	m.duePickerFocus = 2
	_ = m.setDuePickerIncludeTime(false)
	if m.duePickerFocus != 3 {
		t.Fatalf("expected focus to move from time input to list when time disabled, got %d", m.duePickerFocus)
	}
	if got := m.duePickerFocusSlots(); len(got) != 3 {
		t.Fatalf("expected three focus slots without time input, got %#v", got)
	}

	if _, ok := resolveDuePickerDateToken("today", time.Now().In(time.Local)); !ok {
		t.Fatal("expected today token to parse")
	}
	if hour, minute, ok := parseDuePickerTimeToken("5:30pm"); !ok || hour != 17 || minute != 30 {
		t.Fatalf("expected 5:30pm parse to 17:30, got %d:%d ok=%t", hour, minute, ok)
	}
}

// TestTaskFormLabelSuggestions verifies behavior for the covered scenario.
func TestTaskFormLabelSuggestions(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	t1, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Task 1",
		Priority:  domain.PriorityMedium,
		Labels:    []string{"planning", "till"},
	}, now)
	t2, _ := domain.NewTask(domain.TaskInput{
		ID:        "t2",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  1,
		Title:     "Task 2",
		Priority:  domain.PriorityMedium,
		Labels:    []string{"till", "roadmap"},
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{t1, t2})
	m := loadReadyModel(t, NewModel(svc))
	m = applyMsg(t, m, keyRune('n'))
	m = applyCmd(t, m, m.focusTaskFormField(4))
	accent := lipgloss.Color("62")
	muted := lipgloss.Color("241")
	dim := lipgloss.Color("239")
	helpStyle := lipgloss.NewStyle().Foreground(muted)
	out := m.renderModeOverlay(accent, muted, dim, helpStyle, 96)
	if strings.Contains(out, "suggested labels:") {
		t.Fatalf("expected labels field to hide inline suggestion help, got %q", out)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeLabelPicker {
		t.Fatalf("expected labels row enter to open label picker, got %v", m.mode)
	}
	picker := stripANSI(m.renderModeOverlay(accent, muted, dim, helpStyle, 96))
	if !strings.Contains(strings.ToLower(picker), "till") {
		t.Fatalf("expected label picker suggestions to include 'till', got %q", picker)
	}
}

// TestTaskFormLabelPickerDoesNotAcceptInlineTyping verifies labels stay modal-only.
func TestTaskFormLabelPickerDoesNotAcceptInlineTyping(t *testing.T) {
	now := time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
		Labels:    []string{"chore"},
	}, now)
	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})))
	m = applyMsg(t, m, keyRune('n'))
	m = applyCmd(t, m, m.focusTaskFormField(4))
	m = applyMsg(t, m, keyRune('c'))
	m = applyMsg(t, m, keyRune('h'))
	if got := strings.TrimSpace(m.formInputs[4].Value()); got != "" {
		t.Fatalf("expected modal-only labels field to ignore inline typing, got %q", got)
	}
}

// TestProjectFormSavesRootPathOnCreate verifies project-form root path callback wiring.
func TestProjectFormSavesRootPathOnCreate(t *testing.T) {
	now := time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	rootPath := t.TempDir()

	var savedSlug string
	var savedPath string
	m := loadReadyModel(t, NewModel(
		newFakeService([]domain.Project{p}, []domain.Column{c}, nil),
		WithSaveProjectRootCallback(func(projectSlug, root string) error {
			savedSlug = projectSlug
			savedPath = root
			return nil
		}),
	))
	m = applyMsg(t, m, keyRune('N'))
	if m.mode != modeAddProject {
		t.Fatalf("expected add-project mode, got %v", m.mode)
	}
	m.projectFormInputs[0].SetValue("Roadmap")
	m.projectFormInputs[7].SetValue(rootPath)
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if savedSlug == "" {
		t.Fatal("expected project-root callback to capture project slug")
	}
	if savedPath != rootPath {
		t.Fatalf("expected saved root path %q, got %q", rootPath, savedPath)
	}
}

// TestSearchAndCommandPaletteFlow verifies behavior for the covered scenario.
func TestSearchAndCommandPaletteFlow(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Roadmap planning",
		Priority:  domain.PriorityMedium,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})
	m := loadReadyModel(t, NewModel(svc))

	m = applyMsg(t, m, keyRune('/'))
	for _, r := range []rune("road map") {
		m = applyMsg(t, m, keyRune(r))
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab}) // states
	m = applyMsg(t, m, keyRune(' '))                      // toggle todo off
	if m.isSearchStateEnabled("todo") {
		t.Fatalf("expected todo filter disabled, got %#v", m.searchStates)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab}) // levels
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab}) // scope
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab}) // archived
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab}) // apply
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if !m.searchApplied {
		t.Fatalf("expected search applied, got %#v", m)
	}
	if m.searchQuery != "road map" {
		t.Fatalf("expected search query preserved, got %q", m.searchQuery)
	}
	if !m.searchCrossProject || !m.searchIncludeArchived {
		t.Fatalf("expected scope+archived toggled, got cross=%t archived=%t", m.searchCrossProject, m.searchIncludeArchived)
	}

	m = applyMsg(t, m, keyRune(':'))
	for _, r := range []rune("clear-q") {
		m = applyMsg(t, m, keyRune(r))
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.searchQuery != "" {
		t.Fatalf("expected clear-query command to clear text, got %q", m.searchQuery)
	}

	m = applyMsg(t, m, keyRune(':'))
	for _, r := range []rune("reset-f") {
		m = applyMsg(t, m, keyRune(r))
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.searchApplied {
		t.Fatalf("expected reset-filters to clear applied state, got %#v", m)
	}
}

// TestDueHelpers verifies behavior for the covered scenario.
func TestDueHelpers(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	past := now.Add(-2 * time.Hour)
	soon := now.Add(30 * time.Minute)
	later := now.Add(72 * time.Hour)
	archivedAt := now.Add(-10 * time.Minute)

	m := Model{
		dueSoonWindows: []time.Duration{time.Hour},
		tasks: []domain.Task{
			{ID: "t1", DueAt: &past},
			{ID: "t2", DueAt: &soon},
			{ID: "t3", DueAt: &later},
			{ID: "t4", DueAt: &soon, ArchivedAt: &archivedAt},
		},
	}
	overdue, dueSoon := m.dueCounts(now)
	if overdue != 1 || dueSoon != 1 {
		t.Fatalf("expected overdue=1 dueSoon=1, got %d/%d", overdue, dueSoon)
	}

	if got := dueWarning("2020-01-01", now); !strings.Contains(got, "in the past") {
		t.Fatalf("expected due warning for past datetime, got %q", got)
	}
	if got := dueWarning("2099-01-01", now); got != "" {
		t.Fatalf("expected no warning for future datetime, got %q", got)
	}

	if got := formatDueValue(&soon); !strings.Contains(got, soon.In(time.Local).Format("15:04")) {
		t.Fatalf("expected due value with time, got %q", got)
	}
	dateOnly := time.Date(2026, 2, 22, 0, 0, 0, 0, time.Local)
	if got := formatDueValue(&dateOnly); got != dateOnly.Format("2006-01-02") {
		t.Fatalf("expected date-only due format, got %q", got)
	}
}

// TestModelMouseSelectionModeDisablesMouseCapture verifies selection mode disables TUI mouse handling.
func TestModelMouseSelectionModeDisablesMouseCapture(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	t1, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "One",
		Priority:  domain.PriorityMedium,
	}, now)
	t2, _ := domain.NewTask(domain.TaskInput{
		ID:        "t2",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  1,
		Title:     "Two",
		Priority:  domain.PriorityMedium,
	}, now)
	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{t1, t2})))
	m.mouseSelectionMode = true

	before := m.selectedTask
	m = applyMsg(t, m, tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	if m.selectedTask != before {
		t.Fatalf("expected mouse wheel ignored in selection mode, selected=%d before=%d", m.selectedTask, before)
	}
	view := m.View()
	if view.MouseMode != tea.MouseModeNone {
		t.Fatalf("expected mouse mode none in selection mode, got %v", view.MouseMode)
	}

	m.mouseSelectionMode = false
	view = m.View()
	if view.MouseMode != tea.MouseModeCellMotion {
		t.Fatalf("expected cell-motion mouse mode when selection mode disabled, got %v", view.MouseMode)
	}
}

// TestModelMultiSelectBulkMoveUndoRedo verifies behavior for the covered scenario.
func TestModelMultiSelectBulkMoveUndoRedo(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c1, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	c2, _ := domain.NewColumn("c2", p.ID, "Doing", 1, 0, now)
	t1, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c1.ID,
		Position:  0,
		Title:     "One",
		Priority:  domain.PriorityMedium,
	}, now)
	t2, _ := domain.NewTask(domain.TaskInput{
		ID:        "t2",
		ProjectID: p.ID,
		ColumnID:  c1.ID,
		Position:  1,
		Title:     "Two",
		Priority:  domain.PriorityLow,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c1, c2}, []domain.Task{t1, t2})
	m := loadReadyModel(t, NewModel(svc))

	m = applyMsg(t, m, keyRune(' '))
	m = applyMsg(t, m, keyRune('j'))
	m = applyMsg(t, m, keyRune(' '))
	if len(m.selectedTaskIDs) != 2 {
		t.Fatalf("expected 2 selected task ids, got %d", len(m.selectedTaskIDs))
	}

	updated, cmd := m.executeCommandPalette("bulk-move-right")
	m = applyResult(t, updated, cmd)
	if task, ok := svc.taskByID("t1"); !ok || task.ColumnID != c2.ID {
		t.Fatalf("expected t1 moved to %s, got %#v ok=%t", c2.ID, task, ok)
	}
	if task, ok := svc.taskByID("t2"); !ok || task.ColumnID != c2.ID {
		t.Fatalf("expected t2 moved to %s, got %#v ok=%t", c2.ID, task, ok)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl})
	if task, ok := svc.taskByID("t1"); !ok || task.ColumnID != c1.ID {
		t.Fatalf("expected t1 moved back to %s after undo, got %#v ok=%t", c1.ID, task, ok)
	}
	if task, ok := svc.taskByID("t2"); !ok || task.ColumnID != c1.ID {
		t.Fatalf("expected t2 moved back to %s after undo, got %#v ok=%t", c1.ID, task, ok)
	}
	if !strings.Contains(strings.ToLower(m.status), "undo") {
		t.Fatalf("expected undo status message, got %q", m.status)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl | tea.ModShift})
	if task, ok := svc.taskByID("t1"); !ok || task.ColumnID != c2.ID {
		t.Fatalf("expected t1 moved again to %s after redo, got %#v ok=%t", c2.ID, task, ok)
	}
	if task, ok := svc.taskByID("t2"); !ok || task.ColumnID != c2.ID {
		t.Fatalf("expected t2 moved again to %s after redo, got %#v ok=%t", c2.ID, task, ok)
	}
	if !strings.Contains(strings.ToLower(m.status), "redo") {
		t.Fatalf("expected redo status message, got %q", m.status)
	}
}

// TestModelActivityLogOverlay verifies behavior for the covered scenario.
func TestModelActivityLogOverlay(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})
	svc.changeEvents[p.ID] = []domain.ChangeEvent{
		{
			ID:         2,
			ProjectID:  p.ID,
			WorkItemID: task.ID,
			Operation:  domain.ChangeOperationMove,
			Metadata:   map[string]string{},
			OccurredAt: now.Add(2 * time.Minute),
		},
		{
			ID:         1,
			ProjectID:  p.ID,
			WorkItemID: task.ID,
			Operation:  domain.ChangeOperationCreate,
			Metadata:   map[string]string{"title": task.Title},
			OccurredAt: now.Add(time.Minute),
		},
	}
	m := loadReadyModel(t, NewModel(svc))

	m = applyMsg(t, m, keyRune('g'))
	if m.mode != modeActivityLog {
		t.Fatalf("expected activity-log mode, got %v", m.mode)
	}
	out := m.renderModeOverlay(lipgloss.Color("62"), lipgloss.Color("241"), lipgloss.Color("239"), lipgloss.NewStyle(), 96)
	if !strings.Contains(out, "Activity Log") {
		t.Fatalf("expected activity-log title, got %q", out)
	}
	if !strings.Contains(out, "move task") || !strings.Contains(out, "create task") {
		t.Fatalf("expected persisted activity entries, got %q", out)
	}
	if !strings.Contains(out, ":") {
		t.Fatalf("expected timestamp in activity entry, got %q", out)
	}
}

// TestModelActivityLogOverlayLoadFailure verifies graceful degradation when persisted activity fetch fails.
func TestModelActivityLogOverlayLoadFailure(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})
	svc.changeEventsErr = errors.New("load failed")
	m := loadReadyModel(t, NewModel(svc))

	// Create one in-memory entry so the modal still has content after fetch failure.
	m = applyMsg(t, m, keyRune(' '))
	m = applyMsg(t, m, keyRune('g'))
	if m.mode != modeActivityLog {
		t.Fatalf("expected activity-log mode after load failure, got %v", m.mode)
	}
	if !strings.Contains(m.status, "activity log unavailable") {
		t.Fatalf("expected non-fatal activity load status, got %q", m.status)
	}
	out := m.renderModeOverlay(lipgloss.Color("62"), lipgloss.Color("241"), lipgloss.Color("239"), lipgloss.NewStyle(), 96)
	if !strings.Contains(out, "select task") {
		t.Fatalf("expected in-memory fallback entry after activity load failure, got %q", out)
	}
}

// TestModelRecentActivityPanelShowsOwnerPrefix verifies notices activity rows use owner labels.
func TestModelRecentActivityPanelShowsOwnerPrefix(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})
	svc.changeEvents[p.ID] = []domain.ChangeEvent{
		{
			ID:         1,
			ProjectID:  p.ID,
			WorkItemID: task.ID,
			Operation:  domain.ChangeOperationUpdate,
			ActorID:    "agent-live-sync",
			ActorName:  "Live Sync Bot",
			ActorType:  domain.ActorTypeAgent,
			Metadata: map[string]string{
				"title":      task.Title,
				"item_scope": "phase",
			},
			OccurredAt: now.Add(time.Minute),
		},
	}
	m := loadReadyModel(t, NewModel(svc))
	panel := stripANSI(m.renderOverviewPanel(p, lipgloss.Color("62"), lipgloss.Color("241"), lipgloss.Color("239"), 80, 28, 0, 0, 0, nil, false))
	if !strings.Contains(panel, "agent|Live Sync Bot update phase") {
		t.Fatalf("expected owner-prefixed activity row, got %q", panel)
	}
}

// TestModelNoticesActivityDetailAndJump verifies notices focus, event detail modal, and jump-to-node flow.
func TestModelNoticesActivityDetailAndJump(t *testing.T) {
	now := time.Date(2026, 3, 1, 13, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	first, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "First",
		Priority:  domain.PriorityMedium,
	}, now)
	second, _ := domain.NewTask(domain.TaskInput{
		ID:        "t2",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  1,
		Title:     "Second",
		Priority:  domain.PriorityMedium,
	}, now.Add(time.Minute))
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{first, second})
	svc.changeEvents[p.ID] = []domain.ChangeEvent{
		{
			ID:         2,
			ProjectID:  p.ID,
			WorkItemID: second.ID,
			Operation:  domain.ChangeOperationUpdate,
			ActorID:    "user-owner",
			ActorType:  domain.ActorTypeUser,
			Metadata:   map[string]string{"title": second.Title},
			OccurredAt: now.Add(2 * time.Minute),
		},
	}
	m := loadReadyModel(t, NewModel(svc))
	if task, ok := m.selectedTaskInCurrentColumn(); !ok || task.ID != first.ID {
		t.Fatalf("expected first task selected before jump, got %#v ok=%t", task, ok)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	if !m.noticesFocused {
		t.Fatal("expected notices focus after tab")
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeActivityEventInfo {
		t.Fatalf("expected activity event info modal, got %v", m.mode)
	}
	detail := stripANSI(m.renderModeOverlay(lipgloss.Color("62"), lipgloss.Color("241"), lipgloss.Color("239"), lipgloss.NewStyle(), 96))
	if !strings.Contains(detail, "owner: user • user-owner") {
		t.Fatalf("expected owner details in event modal, got %q", detail)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeNone {
		t.Fatalf("expected return to board mode after jump, got %v", m.mode)
	}
	if m.noticesFocused {
		t.Fatal("expected notices focus cleared after jump")
	}
	if task, ok := m.selectedTaskInCurrentColumn(); !ok || task.ID != second.ID {
		t.Fatalf("expected jump to second task node, got %#v ok=%t", task, ok)
	}
}

// TestModelNoticesSectionNavigationAndTaskInfoAction verifies notices section-level traversal and task-info enter actions.
func TestModelNoticesSectionNavigationAndTaskInfoAction(t *testing.T) {
	now := time.Date(2026, 3, 1, 13, 15, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
		Metadata: domain.TaskMetadata{
			BlockedReason: "waiting for review",
		},
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})
	svc.changeEvents[p.ID] = []domain.ChangeEvent{
		{
			ID:         1,
			ProjectID:  p.ID,
			WorkItemID: task.ID,
			Operation:  domain.ChangeOperationUpdate,
			Metadata:   map[string]string{"title": task.Title},
			OccurredAt: now.Add(time.Minute),
		},
	}
	m := loadReadyModel(t, NewModel(svc))

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	if !m.noticesFocused {
		t.Fatal("expected notices focus after tab")
	}
	if m.noticesSection != noticesSectionRecentActivity {
		t.Fatalf("expected default notices section to be recent activity, got %v", m.noticesSection)
	}

	m = applyMsg(t, m, keyRune('k'))
	if m.noticesSection != noticesSectionSelection {
		t.Fatalf("expected up-navigation to move focus to selection section, got %v", m.noticesSection)
	}
	m = applyMsg(t, m, keyRune('k'))
	if m.noticesSection != noticesSectionAttention {
		t.Fatalf("expected second up-navigation to move focus to attention section, got %v", m.noticesSection)
	}
	m = applyMsg(t, m, keyRune('k'))
	if m.noticesSection != noticesSectionWarnings {
		t.Fatalf("expected third up-navigation to move focus to warnings section, got %v", m.noticesSection)
	}

	for i := 0; i < 6 && m.noticesSection != noticesSectionSelection; i++ {
		m = applyMsg(t, m, keyRune('j'))
	}
	if m.noticesSection != noticesSectionSelection {
		t.Fatalf("expected down-navigation to return focus to selection section, got %v", m.noticesSection)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeTaskInfo {
		t.Fatalf("expected enter on selection row to open task info, got %v", m.mode)
	}
	if m.taskInfoTaskID != task.ID {
		t.Fatalf("expected task-info target %q, got %q", task.ID, m.taskInfoTaskID)
	}
}

// TestModelNoticesWarningsAndAttentionRowsOpenTaskInfoWhenAssociated verifies warning/attention row enter actions when scoped to one task.
func TestModelNoticesWarningsAndAttentionRowsOpenTaskInfoWhenAssociated(t *testing.T) {
	now := time.Date(2026, 3, 1, 13, 25, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
		Metadata: domain.TaskMetadata{
			BlockedReason: "blocked on review",
		},
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})
	m := loadReadyModel(t, NewModel(svc))

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	m = applyMsg(t, m, keyRune('k'))
	m = applyMsg(t, m, keyRune('k'))
	if m.noticesSection != noticesSectionAttention {
		t.Fatalf("expected notices focus on attention section, got %v", m.noticesSection)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeTaskInfo {
		t.Fatalf("expected enter on attention row to open task info, got %v", m.mode)
	}
	if m.taskInfoTaskID != task.ID {
		t.Fatalf("expected attention row task-info target %q, got %q", task.ID, m.taskInfoTaskID)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.mode != modeNone {
		t.Fatalf("expected esc to close task info back to board, got %v", m.mode)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	m = applyMsg(t, m, keyRune('k'))
	if m.noticesSection != noticesSectionWarnings {
		t.Fatalf("expected notices focus on warnings section, got %v", m.noticesSection)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeTaskInfo {
		t.Fatalf("expected enter on warning row to open associated task info, got %v", m.mode)
	}
	if m.taskInfoTaskID != task.ID {
		t.Fatalf("expected warning row task-info target %q, got %q", task.ID, m.taskInfoTaskID)
	}
}

// TestModelProjectNotificationsEnterOnNonTaskAttentionRowOpensThread verifies project attention rows without task ids route to thread mode.
func TestModelProjectNotificationsEnterOnNonTaskAttentionRowOpensThread(t *testing.T) {
	now := time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{project}, []domain.Column{column}, []domain.Task{task})))
	m.attentionItems = []domain.AttentionItem{
		{
			ID:                 "att-project-thread",
			ProjectID:          project.ID,
			ScopeType:          domain.ScopeLevelProject,
			ScopeID:            project.ID,
			State:              domain.AttentionStateOpen,
			Kind:               domain.AttentionKindConsensusRequired,
			Summary:            "project-level consensus required",
			BodyMarkdown:       "Need collaborative review before execution.",
			RequiresUserAction: true,
		},
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	m = applyMsg(t, m, keyRune('k'))
	m = applyMsg(t, m, keyRune('k'))
	if m.noticesSection != noticesSectionAttention {
		t.Fatalf("expected notices focus on attention section, got %v", m.noticesSection)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeThread {
		t.Fatalf("expected project attention enter to open thread mode, got %v", m.mode)
	}
	if m.threadTarget.TargetType != domain.CommentTargetTypeProject || m.threadTarget.TargetID != project.ID {
		t.Fatalf("expected project attention thread target %q, got %#v", project.ID, m.threadTarget)
	}
	if !strings.Contains(strings.ToLower(m.threadTitle), "project attention") {
		t.Fatalf("expected thread title to reflect notification scope, got %q", m.threadTitle)
	}
}

// TestModelProjectNotificationsWarningRowsStayScopedAndActionable verifies warning rows remain notification-scoped and open threads when task routing is not applicable.
func TestModelProjectNotificationsWarningRowsStayScopedAndActionable(t *testing.T) {
	now := time.Date(2026, 3, 2, 9, 5, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)

	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{project}, []domain.Column{column}, []domain.Task{task})))
	m.attentionItems = []domain.AttentionItem{
		{
			ID:                 "att-warning-project",
			ProjectID:          project.ID,
			ScopeType:          domain.ScopeLevelProject,
			ScopeID:            project.ID,
			State:              domain.AttentionStateOpen,
			Kind:               domain.AttentionKindConsensusRequired,
			Summary:            "project warning row",
			RequiresUserAction: true,
		},
	}

	sections := m.noticesSectionsForInteraction()
	warningFound := false
	for _, section := range sections {
		if section.ID != noticesSectionWarnings {
			continue
		}
		if len(section.Items) != 1 {
			t.Fatalf("expected one warning row, got %d", len(section.Items))
		}
		row := section.Items[0]
		if row.ScopeType != domain.ScopeLevelProject || row.ScopeID != project.ID {
			t.Fatalf("expected scoped warning row for project %q, got scopeType=%q scopeID=%q", project.ID, row.ScopeType, row.ScopeID)
		}
		warningFound = true
	}
	if !warningFound {
		t.Fatal("expected warnings section in project notifications")
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	m = applyMsg(t, m, keyRune('k'))
	m = applyMsg(t, m, keyRune('k'))
	m = applyMsg(t, m, keyRune('k'))
	if m.noticesSection != noticesSectionWarnings {
		t.Fatalf("expected warnings section focus, got %v", m.noticesSection)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeThread {
		t.Fatalf("expected warning-row enter to open thread mode, got %v", m.mode)
	}
	if m.threadTarget.ProjectID != project.ID || m.threadTarget.TargetType != domain.CommentTargetTypeProject || m.threadTarget.TargetID != project.ID {
		t.Fatalf("expected project warning thread target %q, got %#v", project.ID, m.threadTarget)
	}
}

// TestModelProjectNotificationsActionRequiredSectionFiltersRequiresUserAction verifies Agent/User Action rows only include requires-user-action records.
func TestModelProjectNotificationsActionRequiredSectionFiltersRequiresUserAction(t *testing.T) {
	now := time.Date(2026, 3, 2, 9, 10, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)

	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{project}, []domain.Column{column}, []domain.Task{task})))
	m.attentionItems = []domain.AttentionItem{
		{
			ID:                 "att-no-action",
			ProjectID:          project.ID,
			ScopeType:          domain.ScopeLevelProject,
			ScopeID:            project.ID,
			State:              domain.AttentionStateOpen,
			Kind:               domain.AttentionKindBlocker,
			Summary:            "do not include",
			RequiresUserAction: false,
		},
		{
			ID:                 "att-requires-action",
			ProjectID:          project.ID,
			ScopeType:          domain.ScopeLevelProject,
			ScopeID:            project.ID,
			State:              domain.AttentionStateOpen,
			Kind:               domain.AttentionKindConsensusRequired,
			Summary:            "requires action include",
			RequiresUserAction: true,
		},
	}

	sections := m.noticesSectionsForInteraction()
	attentionFound := false
	for _, section := range sections {
		if section.ID != noticesSectionAttention {
			continue
		}
		if len(section.Items) != 1 {
			t.Fatalf("expected one action-required row, got %d", len(section.Items))
		}
		if section.Items[0].AttentionID != "att-requires-action" {
			t.Fatalf("expected attention id %q, got %q", "att-requires-action", section.Items[0].AttentionID)
		}
		if !strings.Contains(section.Items[0].Label, "requires action include") {
			t.Fatalf("expected requires-user-action row label, got %q", section.Items[0].Label)
		}
		attentionFound = true
	}
	if !attentionFound {
		t.Fatal("expected action-required section in project notifications")
	}
}

// TestModelProjectNotificationsAuthRequestEnterOpensReview verifies auth-request rows open the review modal instead of a generic project thread.
func TestModelProjectNotificationsAuthRequestApproveShortcut(t *testing.T) {
	now := time.Date(2026, 3, 2, 9, 12, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	authRequest, err := domain.NewAuthRequest(domain.AuthRequestInput{
		ID:                  "req-1",
		Path:                domain.AuthRequestPath{ProjectID: project.ID},
		PrincipalID:         "agent-1",
		PrincipalType:       "agent",
		PrincipalName:       "Agent One",
		ClientID:            "till-mcp-stdio",
		ClientType:          "mcp-stdio",
		ClientName:          "Till MCP STDIO",
		RequestedSessionTTL: 8 * time.Hour,
		Reason:              "approval needed",
		RequestedByActor:    "lane-user",
		RequestedByType:     domain.ActorTypeUser,
		Timeout:             30 * time.Minute,
	}, now)
	if err != nil {
		t.Fatalf("NewAuthRequest() error = %v", err)
	}
	svc := newFakeService([]domain.Project{project}, []domain.Column{column}, []domain.Task{task})
	svc.authRequests[authRequest.ID] = authRequest
	svc.attentionItemsByProject[project.ID] = []domain.AttentionItem{
		{
			ID:                 authRequest.ID,
			ProjectID:          project.ID,
			ScopeType:          domain.ScopeLevelProject,
			ScopeID:            project.ID,
			State:              domain.AttentionStateOpen,
			Kind:               domain.AttentionKindConsensusRequired,
			Summary:            "auth request review",
			BodyMarkdown:       "Please review this approval request.",
			RequiresUserAction: true,
			CreatedAt:          now,
		},
	}
	m := loadReadyModel(t, NewModel(svc))

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	m = applyMsg(t, m, keyRune('k'))
	m = applyMsg(t, m, keyRune('k'))
	if m.noticesSection != noticesSectionAttention {
		t.Fatalf("expected attention section focus, got %v", m.noticesSection)
	}
	sections := m.noticesSectionsForInteraction()
	var row noticesPanelItem
	found := false
	for _, section := range sections {
		if section.ID != noticesSectionAttention {
			continue
		}
		if len(section.Items) != 1 {
			t.Fatalf("expected one auth-request row, got %d", len(section.Items))
		}
		row = section.Items[0]
		found = true
	}
	if !found {
		t.Fatal("expected auth-request row in attention section")
	}
	if row.AttentionID != authRequest.ID {
		t.Fatalf("expected attention id %q, got %q", authRequest.ID, row.AttentionID)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeAuthReview {
		t.Fatalf("expected enter to open auth review surface, got %v", m.mode)
	}
	if m.pendingConfirm.Kind != "approve-auth-request" {
		t.Fatalf("expected approve review kind, got %q", m.pendingConfirm.Kind)
	}
	if m.pendingConfirm.AuthRequestID != authRequest.ID {
		t.Fatalf("expected confirm auth request id %q, got %q", authRequest.ID, m.pendingConfirm.AuthRequestID)
	}
	if got := strings.TrimSpace(m.pendingConfirm.AuthRequestPathLabel); got != "Inbox" {
		t.Fatalf("pendingConfirm.AuthRequestPathLabel = %q, want Inbox", got)
	}
	if !strings.Contains(m.pendingConfirm.AuthRequestNote, "approved in Tillsyn") {
		t.Fatalf("expected deterministic approval note, got %q", m.pendingConfirm.AuthRequestNote)
	}
	if !strings.Contains(m.pendingConfirm.AuthRequestNote, "at Inbox") {
		t.Fatalf("expected approval note to use project label, got %q", m.pendingConfirm.AuthRequestNote)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeNone {
		t.Fatalf("expected approve action to close review, got %v", m.mode)
	}
	if got := strings.TrimSpace(svc.lastApproveAuthRequest.RequestID); got != authRequest.ID {
		t.Fatalf("expected approve request id %q, got %q", authRequest.ID, got)
	}
	if got := strings.TrimSpace(svc.lastApproveAuthRequest.ResolutionNote); !strings.Contains(got, "approved in Tillsyn") || !strings.Contains(got, "at Inbox") {
		t.Fatalf("expected approval note to be forwarded with project label, got %q", got)
	}
	if len(m.attentionItems) != 0 {
		t.Fatalf("expected reload to clear pending auth-request row, got %d attention items", len(m.attentionItems))
	}
}

// TestModelProjectNotificationsAuthRequestStaysOutOfGlobalPanel verifies focused-project requests do not duplicate into global notices.
func TestModelProjectNotificationsAuthRequestStaysOutOfGlobalPanel(t *testing.T) {
	now := time.Date(2026, 3, 2, 9, 13, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	other, _ := domain.NewProject("p2", "Elsewhere", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	authRequest, err := domain.NewAuthRequest(domain.AuthRequestInput{
		ID:                  "req-focused",
		Path:                domain.AuthRequestPath{ProjectID: project.ID},
		PrincipalID:         "agent-1",
		PrincipalType:       "agent",
		ClientID:            "till-mcp-stdio",
		ClientType:          "mcp-stdio",
		RequestedSessionTTL: 8 * time.Hour,
		Reason:              "focused review",
		RequestedByActor:    "lane-user",
		RequestedByType:     domain.ActorTypeUser,
		Timeout:             30 * time.Minute,
	}, now)
	if err != nil {
		t.Fatalf("NewAuthRequest() error = %v", err)
	}
	svc := newFakeService([]domain.Project{project, other}, []domain.Column{column}, []domain.Task{task})
	svc.attentionItemsByProject[project.ID] = []domain.AttentionItem{{
		ID:                 authRequest.ID,
		ProjectID:          project.ID,
		ScopeType:          domain.ScopeLevelProject,
		ScopeID:            project.ID,
		State:              domain.AttentionStateOpen,
		Kind:               domain.AttentionKindConsensusRequired,
		Summary:            "auth request review",
		RequiresUserAction: true,
		CreatedAt:          now,
	}}
	svc.authRequests[authRequest.ID] = authRequest

	m := loadReadyModel(t, NewModel(svc))
	if len(m.attentionItems) != 1 {
		t.Fatalf("attentionItems = %d, want 1", len(m.attentionItems))
	}
	if len(m.globalNotices) != 0 {
		t.Fatalf("globalNotices = %d, want 0 for focused-project request", len(m.globalNotices))
	}
}

// TestModelProjectNotificationsAuthRequestApproveForwardsConstraints verifies TUI approval can narrow path and lifetime before submission.
func TestModelProjectNotificationsAuthRequestApproveForwardsConstraints(t *testing.T) {
	now := time.Date(2026, 3, 2, 9, 14, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	requested, _ := domain.NewTask(domain.TaskInput{
		ID:        "requested",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "Requested Branch",
		Priority:  domain.PriorityMedium,
		Kind:      domain.WorkKind("branch"),
		Scope:     domain.KindAppliesToBranch,
	}, now)
	narrowed, _ := domain.NewTask(domain.TaskInput{
		ID:        "narrowed",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  1,
		Title:     "Narrowed Branch",
		Priority:  domain.PriorityMedium,
		Kind:      domain.WorkKind("branch"),
		Scope:     domain.KindAppliesToBranch,
	}, now)
	authRequest, err := domain.NewAuthRequest(domain.AuthRequestInput{
		ID:                  "req-approve-constraints",
		Path:                domain.AuthRequestPath{ProjectID: project.ID, BranchID: "requested"},
		PrincipalID:         "agent-1",
		PrincipalType:       "agent",
		ClientID:            "till-mcp-stdio",
		ClientType:          "mcp-stdio",
		RequestedSessionTTL: 8 * time.Hour,
		Reason:              "needs narrowed scope",
		RequestedByActor:    "lane-user",
		RequestedByType:     domain.ActorTypeUser,
		Timeout:             30 * time.Minute,
	}, now)
	if err != nil {
		t.Fatalf("NewAuthRequest() error = %v", err)
	}
	svc := newFakeService([]domain.Project{project}, []domain.Column{column}, []domain.Task{requested, narrowed})
	svc.authRequests[authRequest.ID] = authRequest
	svc.attentionItemsByProject[project.ID] = []domain.AttentionItem{{
		ID:                 authRequest.ID,
		ProjectID:          project.ID,
		ScopeType:          domain.ScopeLevelProject,
		ScopeID:            project.ID,
		State:              domain.AttentionStateOpen,
		Kind:               domain.AttentionKindConsensusRequired,
		Summary:            "auth request review",
		RequiresUserAction: true,
		CreatedAt:          now,
	}}

	m := loadReadyModel(t, NewModel(svc))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	m = applyMsg(t, m, keyRune('k'))
	m = applyMsg(t, m, keyRune('k'))
	m.noticesFocused = true
	m = applyMsg(t, m, keyRune('a'))
	if m.mode != modeAuthReview {
		t.Fatalf("expected auth review mode, got %v", m.mode)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	m = applyMsg(t, m, keyRune('s'))
	if m.mode != modeAuthScopePicker {
		t.Fatalf("expected auth scope picker mode, got %v", m.mode)
	}
	m.authReviewScopePickerIndex = 2
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeAuthReview {
		t.Fatalf("expected return to auth review after scope pick, got %v", m.mode)
	}
	if got := strings.TrimSpace(m.pendingConfirm.AuthRequestPath); got != "project/p1/branch/narrowed" {
		t.Fatalf("pendingConfirm.AuthRequestPath = %q, want project/p1/branch/narrowed", got)
	}
	m = applyMsg(t, m, keyRune('t'))
	if m.authReviewStage != authReviewStageEditTTL {
		t.Fatalf("expected edit ttl stage, got %d", m.authReviewStage)
	}
	m.confirmAuthTTLInput.SetValue("2h")
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if got := strings.TrimSpace(svc.lastApproveAuthRequest.Path); got != "project/p1/branch/narrowed" {
		t.Fatalf("ApproveAuthRequest() path = %q, want project/p1/branch/narrowed", got)
	}
	if got := svc.lastApproveAuthRequest.SessionTTL; got != 2*time.Hour {
		t.Fatalf("ApproveAuthRequest() ttl = %s, want 2h", got)
	}
}

// TestModelAutoRefreshLoadsExternalAuthRequest verifies externally created auth requests appear without project-switch workarounds.
func TestModelAutoRefreshLoadsExternalAuthRequest(t *testing.T) {
	now := time.Date(2026, 3, 2, 9, 16, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	svc := newFakeService([]domain.Project{project}, []domain.Column{column}, []domain.Task{task})
	m := loadReadyModel(t, NewModel(svc))
	m.autoRefreshInterval = time.Second

	authRequest, err := domain.NewAuthRequest(domain.AuthRequestInput{
		ID:                  "req-refresh",
		Path:                domain.AuthRequestPath{ProjectID: project.ID},
		PrincipalID:         "agent-1",
		PrincipalType:       "agent",
		ClientID:            "till-mcp-stdio",
		ClientType:          "mcp-stdio",
		RequestedSessionTTL: 8 * time.Hour,
		Reason:              "external create",
		RequestedByActor:    "remote-agent",
		RequestedByType:     domain.ActorTypeAgent,
		Timeout:             30 * time.Minute,
	}, now)
	if err != nil {
		t.Fatalf("NewAuthRequest() error = %v", err)
	}
	svc.authRequests[authRequest.ID] = authRequest
	svc.attentionItemsByProject[project.ID] = []domain.AttentionItem{{
		ID:                 authRequest.ID,
		ProjectID:          project.ID,
		ScopeType:          domain.ScopeLevelProject,
		ScopeID:            project.ID,
		State:              domain.AttentionStateOpen,
		Kind:               domain.AttentionKindConsensusRequired,
		Summary:            "external auth request",
		RequiresUserAction: true,
		CreatedAt:          now,
	}}

	var cmd tea.Cmd
	m, cmd = applyAutoRefreshTickMsg(t, m)
	m = applyAutoRefreshLoadResult(t, m, cmd)
	if len(m.attentionItems) != 1 {
		t.Fatalf("attentionItems = %d, want 1 after auto refresh", len(m.attentionItems))
	}
	if got := m.attentionItems[0].ID; got != authRequest.ID {
		t.Fatalf("attentionItems[0].ID = %q, want %q", got, authRequest.ID)
	}
}

// TestModelAuthRequestApproveRejectsInvalidTTL verifies invalid approval TTL input keeps the auth review editor open.
func TestModelAuthRequestApproveRejectsInvalidTTL(t *testing.T) {
	now := time.Date(2026, 3, 2, 9, 17, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	authRequest, err := domain.NewAuthRequest(domain.AuthRequestInput{
		ID:                  "req-invalid-ttl",
		Path:                domain.AuthRequestPath{ProjectID: project.ID},
		PrincipalID:         "agent-1",
		PrincipalType:       "agent",
		ClientID:            "till-mcp-stdio",
		ClientType:          "mcp-stdio",
		RequestedSessionTTL: 8 * time.Hour,
		Reason:              "invalid ttl branch",
		RequestedByActor:    "lane-user",
		RequestedByType:     domain.ActorTypeUser,
		Timeout:             30 * time.Minute,
	}, now)
	if err != nil {
		t.Fatalf("NewAuthRequest() error = %v", err)
	}
	svc := newFakeService([]domain.Project{project}, []domain.Column{column}, []domain.Task{task})
	svc.authRequests[authRequest.ID] = authRequest
	svc.attentionItemsByProject[project.ID] = []domain.AttentionItem{{
		ID:                 authRequest.ID,
		ProjectID:          project.ID,
		ScopeType:          domain.ScopeLevelProject,
		ScopeID:            project.ID,
		State:              domain.AttentionStateOpen,
		Kind:               domain.AttentionKindConsensusRequired,
		Summary:            "auth request review",
		RequiresUserAction: true,
		CreatedAt:          now,
	}}

	m := loadReadyModel(t, NewModel(svc))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	m = applyMsg(t, m, keyRune('k'))
	m = applyMsg(t, m, keyRune('k'))
	m.noticesFocused = true
	m = applyMsg(t, m, keyRune('a'))
	if m.mode != modeAuthReview {
		t.Fatalf("expected auth review mode, got %v", m.mode)
	}
	m = applyMsg(t, m, keyRune('t'))
	if m.authReviewStage != authReviewStageEditTTL {
		t.Fatalf("expected edit ttl stage, got %d", m.authReviewStage)
	}
	m.confirmAuthTTLInput.SetValue("definitely-not-a-duration")
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeAuthReview || m.authReviewStage != authReviewStageEditTTL {
		t.Fatalf("expected invalid ttl to keep auth review ttl editor, got mode=%v stage=%d", m.mode, m.authReviewStage)
	}
	if svc.lastApproveAuthRequest.RequestID != "" {
		t.Fatalf("ApproveAuthRequest() should not have been called, got %#v", svc.lastApproveAuthRequest)
	}
}

// TestModelBeginSelectedAuthRequestDecisionRequiresPendingRequest verifies auth-request shortcuts fail closed when the selected notice cannot resolve a pending request.
func TestModelBeginSelectedAuthRequestDecisionRequiresPendingRequest(t *testing.T) {
	now := time.Date(2026, 3, 2, 9, 18, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	svc := newFakeService([]domain.Project{project}, []domain.Column{column}, []domain.Task{task})
	svc.attentionItemsByProject[project.ID] = []domain.AttentionItem{{
		ID:                 "req-missing",
		ProjectID:          project.ID,
		ScopeType:          domain.ScopeLevelProject,
		ScopeID:            project.ID,
		State:              domain.AttentionStateOpen,
		Kind:               domain.AttentionKindConsensusRequired,
		Summary:            "auth request review",
		RequiresUserAction: true,
		CreatedAt:          now,
	}}

	m := loadReadyModel(t, NewModel(svc))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	m = applyMsg(t, m, keyRune('k'))
	m = applyMsg(t, m, keyRune('k'))
	m.noticesFocused = true

	if _, ok := m.selectedAuthRequestForActiveNotice(); ok {
		t.Fatal("selectedAuthRequestForActiveNotice() = true, want false for missing auth request state")
	}
	next, cmd, ok := m.beginSelectedAuthRequestDecision("approve")
	if ok || cmd != nil {
		t.Fatalf("beginSelectedAuthRequestDecision() = ok=%t cmd=%v, want false nil", ok, cmd)
	}
	if got := next.(Model).mode; got == modeAuthReview {
		t.Fatalf("beginSelectedAuthRequestDecision() opened auth review unexpectedly: %v", got)
	}
}

// TestModelBeginSelectedAuthRequestDecisionDenyUsesEditableNote verifies deny review keeps the note field editable.
func TestModelBeginSelectedAuthRequestDecisionDenyUsesButtonFocus(t *testing.T) {
	now := time.Date(2026, 3, 2, 9, 18, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	authRequest, err := domain.NewAuthRequest(domain.AuthRequestInput{
		ID:                  "req-deny-focus",
		Path:                domain.AuthRequestPath{ProjectID: project.ID},
		PrincipalID:         "agent-1",
		PrincipalType:       "agent",
		ClientID:            "till-mcp-stdio",
		ClientType:          "mcp-stdio",
		RequestedSessionTTL: 8 * time.Hour,
		Reason:              "deny flow",
		RequestedByActor:    "lane-user",
		RequestedByType:     domain.ActorTypeUser,
		Timeout:             30 * time.Minute,
	}, now)
	if err != nil {
		t.Fatalf("NewAuthRequest() error = %v", err)
	}
	svc := newFakeService([]domain.Project{project}, []domain.Column{column}, []domain.Task{task})
	svc.authRequests[authRequest.ID] = authRequest
	svc.attentionItemsByProject[project.ID] = []domain.AttentionItem{{
		ID:                 authRequest.ID,
		ProjectID:          project.ID,
		ScopeType:          domain.ScopeLevelProject,
		ScopeID:            project.ID,
		State:              domain.AttentionStateOpen,
		Kind:               domain.AttentionKindConsensusRequired,
		Summary:            "auth request review",
		RequiresUserAction: true,
		CreatedAt:          now,
	}}

	m := loadReadyModel(t, NewModel(svc))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	m = applyMsg(t, m, keyRune('k'))
	m = applyMsg(t, m, keyRune('k'))
	m.noticesFocused = true
	m = applyMsg(t, m, keyRune('d'))
	if m.mode != modeAuthReview {
		t.Fatalf("expected auth review mode, got %v", m.mode)
	}
	if m.authReviewStage != authReviewStageDeny {
		t.Fatalf("authReviewStage = %d, want deny stage", m.authReviewStage)
	}
	if !m.authConfirmFieldsActive() {
		t.Fatal("authConfirmFieldsActive() = false for deny flow, want true")
	}
	if m.authConfirmScopeFieldsActive() {
		t.Fatal("authConfirmScopeFieldsActive() = true for deny flow, want false")
	}
	if !m.confirmAuthNoteInput.Focused() {
		t.Fatal("expected deny flow to start in the note input")
	}
}

// TestModelSetConfirmFocusHandlesNilAndBounds verifies auth confirm focus clamps invalid values and tolerates nil receivers.
func TestModelSetConfirmFocusHandlesNilAndBounds(t *testing.T) {
	var nilModel *Model
	if cmd := nilModel.setConfirmFocus(confirmFocusAuthPath); cmd != nil {
		t.Fatalf("nil setConfirmFocus() cmd = %#v, want nil", cmd)
	}

	m := NewModel(newFakeService(nil, nil, nil))
	m.pendingConfirm = confirmAction{
		Kind:          "approve-auth-request",
		AuthRequestID: "req-1",
	}
	if cmd := m.setConfirmFocus(-1); cmd != nil {
		t.Fatalf("setConfirmFocus(invalid low) cmd = %#v, want nil", cmd)
	}
	if m.confirmFocus != confirmFocusButtons {
		t.Fatalf("confirmFocus = %d, want buttons after low clamp", m.confirmFocus)
	}
	if cmd := m.setConfirmFocus(confirmFocusAuthPath); cmd == nil {
		t.Fatal("setConfirmFocus(path) cmd = nil, want focus command")
	}
	if m.confirmFocus != confirmFocusAuthPath {
		t.Fatalf("confirmFocus = %d, want auth path focus", m.confirmFocus)
	}
	if cmd := m.setConfirmFocus(confirmFocusAuthTTL); cmd == nil {
		t.Fatal("setConfirmFocus(ttl) cmd = nil, want focus command")
	}
	if m.confirmFocus != confirmFocusAuthTTL {
		t.Fatalf("confirmFocus = %d, want auth ttl focus", m.confirmFocus)
	}
	if cmd := m.setConfirmFocus(confirmFocusAuthNote); cmd == nil {
		t.Fatal("setConfirmFocus(note) cmd = nil, want focus command")
	}
	if m.confirmFocus != confirmFocusAuthNote {
		t.Fatalf("confirmFocus = %d, want auth note focus", m.confirmFocus)
	}
}

// TestAuthRequestResolutionHelpers verifies auth-request note helpers keep deterministic user-visible wording.
func TestAuthRequestResolutionHelpers(t *testing.T) {
	now := time.Date(2026, 3, 2, 9, 18, 0, 0, time.UTC)
	req, err := domain.NewAuthRequest(domain.AuthRequestInput{
		ID:                  "req-helper",
		Path:                domain.AuthRequestPath{ProjectID: "p1", BranchID: "b1"},
		PrincipalID:         "agent-1",
		PrincipalType:       "agent",
		ClientID:            "till-mcp-stdio",
		ClientType:          "mcp-stdio",
		RequestedSessionTTL: 8 * time.Hour,
		Reason:              "helper coverage",
		RequestedByActor:    "lane-user",
		RequestedByType:     domain.ActorTypeUser,
		Timeout:             30 * time.Minute,
	}, now)
	if err != nil {
		t.Fatalf("NewAuthRequest() error = %v", err)
	}
	if got := authRequestResolutionNote(req, "deny"); !strings.Contains(got, "denied in Tillsyn") || !strings.Contains(got, "agent-1") {
		t.Fatalf("authRequestResolutionNote(deny) = %q, want denied note with principal fallback", got)
	}
	if got := authRequestResolutionNote(req, ""); !strings.Contains(got, "resolved in Tillsyn") {
		t.Fatalf("authRequestResolutionNote(default) = %q, want resolved note", got)
	}
	if got := authRequestResolutionNoteWithPathLabel(req, "approve", "Inbox -> branch:b1"); !strings.Contains(got, "approved in Tillsyn") || !strings.Contains(got, "Inbox -> branch:b1") {
		t.Fatalf("authRequestResolutionNoteWithPathLabel() = %q, want user-facing scope label", got)
	}
	if got := firstNonEmptyTrimmed("", "  ", "value", "fallback"); got != "value" {
		t.Fatalf("firstNonEmptyTrimmed() = %q, want value", got)
	}
}

// TestModelAuthConfirmHelpers verifies auth approval confirm helpers validate edits and render field-aware hints.
func TestModelAuthConfirmHelpers(t *testing.T) {
	now := time.Date(2026, 3, 2, 9, 18, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	svc := newFakeService([]domain.Project{project}, []domain.Column{column}, []domain.Task{task})

	m := loadReadyModel(t, NewModel(svc))
	m.pendingConfirm = confirmAction{
		Kind:          "approve-auth-request",
		AuthRequestID: "req-1",
	}
	if !m.authConfirmFieldsActive() {
		t.Fatal("authConfirmFieldsActive() = false, want true for auth approval")
	}
	m.confirmAuthPathInput.SetValue("project/p1/branch/b1")
	m.confirmAuthTTLInput.SetValue("90m")
	m.confirmAuthNoteInput.SetValue("operator note")
	action, err := m.prepareConfirmAction()
	if err != nil {
		t.Fatalf("prepareConfirmAction() error = %v", err)
	}
	if action.AuthRequestPath != "project/p1/branch/b1" || action.AuthRequestTTL != "90m" || action.AuthRequestNote != "operator note" {
		t.Fatalf("prepareConfirmAction() = %#v, want path + ttl + note snapshot", action)
	}
	if got := confirmActionHints(true, true); !strings.Contains(got, "tab move fields") {
		t.Fatalf("confirmActionHints(true) = %q, want auth field guidance", got)
	}
	if got := confirmActionHints(true, true); strings.Contains(got, "left/right choose decision") || strings.Contains(got, "h/l") {
		t.Fatalf("confirmActionHints(true, true) = %q, want auth review typing-safe guidance", got)
	}
	if got := confirmActionHints(false, false); strings.Contains(got, "a approve") {
		t.Fatalf("confirmActionHints(false, false) = %q, want generic confirm guidance", got)
	}
	if got := confirmActionHints(true, false); strings.Contains(got, "a approve") || strings.Contains(got, "h/l") {
		t.Fatalf("confirmActionHints(true, false) = %q, want auth review guidance without confirm hotkeys", got)
	}
	if got := confirmActionHints(true, false); !strings.Contains(got, "tab move fields") {
		t.Fatalf("confirmActionHints(true, false) = %q, want note-field navigation guidance", got)
	}
	if cmd := m.setConfirmFocus(confirmFocusAuthDecision); cmd != nil {
		t.Fatalf("setConfirmFocus(decision) = %v, want nil focus cmd for decision selector", cmd)
	}
	if cmd := m.setConfirmFocus(confirmFocusAuthPath); cmd == nil || !m.confirmAuthPathInput.Focused() {
		t.Fatalf("setConfirmFocus(path) = %v, want focused path input", cmd)
	}
	if cmd := m.setConfirmFocus(confirmFocusAuthTTL); cmd == nil || !m.confirmAuthTTLInput.Focused() {
		t.Fatalf("setConfirmFocus(ttl) = %v, want focused ttl input", cmd)
	}
	if cmd := m.setConfirmFocus(confirmFocusAuthNote); cmd == nil || !m.confirmAuthNoteInput.Focused() {
		t.Fatalf("setConfirmFocus(note) = %v, want focused note input", cmd)
	}
	_ = m.setConfirmFocus(999)
	if m.confirmFocus != confirmFocusButtons || m.confirmAuthPathInput.Focused() || m.confirmAuthTTLInput.Focused() || m.confirmAuthNoteInput.Focused() {
		t.Fatalf("setConfirmFocus(default) left stale focus state: focus=%d path=%t ttl=%t note=%t", m.confirmFocus, m.confirmAuthPathInput.Focused(), m.confirmAuthTTLInput.Focused(), m.confirmAuthNoteInput.Focused())
	}
	m.confirmAuthPathInput.SetValue("not-a-valid-auth-path")
	if _, err := m.prepareConfirmAction(); err == nil {
		t.Fatal("prepareConfirmAction() expected invalid auth path error")
	}
}

// TestModelAuthRequestPathDisplayUsesProjectName verifies auth review labels prefer user-facing project names over raw ids.
func TestModelAuthRequestPathDisplayUsesProjectName(t *testing.T) {
	now := time.Date(2026, 3, 2, 9, 18, 30, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	other, _ := domain.NewProject("p2", "Roadmap", "", now.Add(time.Minute))
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	branch, _ := domain.NewTask(domain.TaskInput{
		ID:        "b1",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "Planning Branch",
		Priority:  domain.PriorityMedium,
		Kind:      domain.WorkKind("branch"),
		Scope:     domain.KindAppliesToBranch,
	}, now)
	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{project, other}, []domain.Column{column}, []domain.Task{branch})))

	if got := strings.TrimSpace(m.authRequestPathDisplay("project/p1/branch/b1")); got != "Inbox -> Planning Branch" {
		t.Fatalf("authRequestPathDisplay() = %q, want Inbox -> Planning Branch", got)
	}
	if got := strings.TrimSpace(m.authRequestPathDisplay("projects/p1,p2")); got != "Inbox, Roadmap" {
		t.Fatalf("authRequestPathDisplay(multi) = %q, want Inbox, Roadmap", got)
	}
	if got := strings.TrimSpace(m.authRequestPathDisplay("global")); got != "All Projects" {
		t.Fatalf("authRequestPathDisplay(global) = %q, want All Projects", got)
	}
}

// TestModelViewRendersAuthReviewDetails verifies the dedicated auth review renders subject, scope, and actions.
func TestModelViewRendersAuthReviewDetails(t *testing.T) {
	now := time.Date(2026, 3, 2, 9, 19, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	svc := newFakeService([]domain.Project{project}, []domain.Column{column}, []domain.Task{task})

	m := loadReadyModel(t, NewModel(svc))
	m.mode = modeAuthReview
	m.authReviewStage = authReviewStageSummary
	m.pendingConfirm = confirmAction{
		Kind:                 "approve-auth-request",
		Label:                "approve auth request",
		AuthRequestID:        "req-1",
		AuthRequestPrincipal: "Review Agent",
		AuthRequestPath:      "project/p1/branch/b1",
		AuthRequestPathLabel: "Inbox -> branch:b1",
		AuthRequestTTL:       "2h",
		AuthRequestDecision:  "approve",
		AuthRequestNote:      "approved in Tillsyn for Review Agent at Inbox -> branch:b1",
	}
	m.confirmAuthPathInput.SetValue("project/p1/branch/b1")
	m.confirmAuthTTLInput.SetValue("2h")
	m.confirmAuthNoteInput.SetValue("approved in Tillsyn for Review Agent at Inbox -> branch:b1")

	rendered := fmt.Sprint(m.View())
	for _, want := range []string{
		"Access Request Review",
		"principal: Review Agent",
		"requested scope: Inbox -> branch:b1",
		"approve now",
		"default decision: approve",
		"[enter] approve and confirm",
		"[s] pick approved scope",
		"path: project/p1/branch/b1",
		"session ttl: 2h",
		"[d] deny with note",
		"project/p1/branch/b1",
		"approved in Tillsyn for Review Agent at Inbox -> branch:b1",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("View() missing %q in auth review:\n%s", want, rendered)
		}
	}
}

// TestModelViewRendersGenericConfirmHints verifies non-auth confirm modals keep generic confirmation hints.
func TestModelViewRendersGenericConfirmHints(t *testing.T) {
	now := time.Date(2026, 3, 2, 9, 19, 30, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	svc := newFakeService([]domain.Project{project}, []domain.Column{column}, []domain.Task{task})

	m := loadReadyModel(t, NewModel(svc))
	m.mode = modeConfirmAction
	m.pendingConfirm = confirmAction{
		Kind:  "delete-task",
		Label: "delete task",
		Task:  task,
	}

	rendered := fmt.Sprint(m.View())
	if !strings.Contains(rendered, "enter apply") {
		t.Fatalf("View() missing generic confirm hint:\n%s", rendered)
	}
	if strings.Contains(rendered, "a approve") || strings.Contains(rendered, "d deny") {
		t.Fatalf("View() unexpectedly rendered auth review hints for generic confirm modal:\n%s", rendered)
	}
}

// TestModelAuthInventoryLoadsProjectScope verifies the auth inventory opens in current-project scope and can enter auth review.
func TestModelAuthInventoryLoadsProjectScope(t *testing.T) {
	now := time.Date(2026, 3, 2, 9, 20, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	other, _ := domain.NewProject("p2", "Roadmap", "", now.Add(time.Minute))
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	request, err := domain.NewAuthRequest(domain.AuthRequestInput{
		ID:                  "req-auth-inventory",
		Path:                domain.AuthRequestPath{ProjectID: project.ID},
		PrincipalID:         "review-agent",
		PrincipalType:       "agent",
		PrincipalName:       "Review Agent",
		ClientID:            "till-mcp-stdio",
		ClientType:          "mcp-stdio",
		ClientName:          "Till MCP STDIO",
		RequestedSessionTTL: 8 * time.Hour,
		Reason:              "inventory review",
		RequestedByActor:    "lane-user",
		RequestedByType:     domain.ActorTypeUser,
		Timeout:             30 * time.Minute,
	}, now)
	if err != nil {
		t.Fatalf("NewAuthRequest() error = %v", err)
	}
	svc := newFakeService([]domain.Project{project, other}, []domain.Column{column}, []domain.Task{task})
	svc.authRequests[request.ID] = request
	svc.authRequests["req-other"] = domain.AuthRequest{
		ID:                  "req-other",
		ProjectID:           other.ID,
		Path:                "project/" + other.ID,
		ScopeType:           domain.ScopeLevelProject,
		ScopeID:             other.ID,
		PrincipalID:         "other-agent",
		ClientID:            "other-client",
		State:               domain.AuthRequestStatePending,
		RequestedSessionTTL: time.Hour,
		CreatedAt:           now.Add(time.Minute),
		ExpiresAt:           now.Add(2 * time.Hour),
	}
	svc.authSessions = append(svc.authSessions,
		app.AuthSession{
			SessionID:     "session-project",
			ProjectID:     project.ID,
			ApprovedPath:  "project/" + project.ID,
			PrincipalID:   "review-agent",
			PrincipalType: "agent",
			PrincipalName: "Review Agent",
			ClientID:      "till-mcp-stdio",
			ClientType:    "mcp-stdio",
			ClientName:    "Till MCP STDIO",
			ExpiresAt:     time.Now().UTC().Add(2 * time.Hour),
		},
		app.AuthSession{
			SessionID:     "session-other",
			ProjectID:     other.ID,
			ApprovedPath:  "project/" + other.ID,
			PrincipalID:   "other-agent",
			PrincipalType: "agent",
			PrincipalName: "Other Agent",
			ClientID:      "other-client",
			ClientType:    "mcp-stdio",
			ClientName:    "Other Client",
			ExpiresAt:     time.Now().UTC().Add(3 * time.Hour),
		},
	)

	m := loadReadyModel(t, NewModel(svc))
	m = applyCmd(t, m, m.startAuthInventory(false))
	if m.mode != modeAuthInventory {
		t.Fatalf("expected auth inventory mode, got %v", m.mode)
	}
	if got := strings.TrimSpace(svc.lastAuthRequestFilter.ProjectID); got != project.ID {
		t.Fatalf("ListAuthRequests() project filter = %q, want %q", got, project.ID)
	}
	if got := strings.TrimSpace(svc.lastAuthSessionFilter.ProjectID); got != project.ID {
		t.Fatalf("ListAuthSessions() project filter = %q, want %q", got, project.ID)
	}
	if got := len(m.authInventoryRequests); got != 1 {
		t.Fatalf("authInventoryRequests = %d, want 1", got)
	}
	if got := len(m.authInventorySessions); got != 1 {
		t.Fatalf("authInventorySessions = %d, want 1", got)
	}

	rendered := stripANSI(fmt.Sprint(m.View()))
	for _, want := range []string{"Auth Inventory", "Inbox", "pending requests", "resolved requests", "active sessions", "[pending] Review Agent", "[active] Review Agent"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("View() missing %q in auth inventory:\n%s", want, rendered)
		}
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeAuthReview {
		t.Fatalf("expected enter to open auth review, got %v", m.mode)
	}
	if !m.pendingConfirm.ReturnToAuthAccess {
		t.Fatal("expected auth review opened from inventory to return to auth inventory")
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.mode != modeAuthInventory {
		t.Fatalf("expected escape to return to auth inventory, got %v", m.mode)
	}
}

// TestModelAuthInventorySplitsPendingAndResolvedRequests verifies the inventory keeps pending requests selectable while still rendering resolved history.
func TestModelAuthInventorySplitsPendingAndResolvedRequests(t *testing.T) {
	now := time.Date(2026, 3, 2, 9, 20, 30, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	pending, err := domain.NewAuthRequest(domain.AuthRequestInput{
		ID:                  "req-pending",
		Path:                domain.AuthRequestPath{ProjectID: project.ID},
		PrincipalID:         "pending-agent",
		PrincipalType:       "agent",
		ClientID:            "till-mcp-stdio",
		ClientType:          "mcp-stdio",
		RequestedSessionTTL: 8 * time.Hour,
		Reason:              "pending review",
		RequestedByActor:    "lane-user",
		RequestedByType:     domain.ActorTypeUser,
		Timeout:             30 * time.Minute,
	}, now)
	if err != nil {
		t.Fatalf("NewAuthRequest(pending) error = %v", err)
	}
	approved, err := domain.NewAuthRequest(domain.AuthRequestInput{
		ID:                  "req-approved",
		Path:                domain.AuthRequestPath{ProjectID: project.ID},
		PrincipalID:         "approved-agent",
		PrincipalType:       "agent",
		ClientID:            "till-mcp-stdio",
		ClientType:          "mcp-stdio",
		RequestedSessionTTL: 8 * time.Hour,
		Reason:              "approved review",
		RequestedByActor:    "lane-user",
		RequestedByType:     domain.ActorTypeUser,
		Timeout:             30 * time.Minute,
	}, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("NewAuthRequest(approved) error = %v", err)
	}
	if err := approved.Approve("approver-1", domain.ActorTypeUser, "approved", "sess-1", "secret-1", now.Add(2*time.Hour), now.Add(time.Minute)); err != nil {
		t.Fatalf("Approve() error = %v", err)
	}

	svc := newFakeService([]domain.Project{project}, []domain.Column{column}, []domain.Task{task})
	svc.authRequests[pending.ID] = pending
	svc.authRequests[approved.ID] = approved

	m := loadReadyModel(t, NewModel(svc))
	m = applyCmd(t, m, m.startAuthInventory(false))
	if got := domain.NormalizeAuthRequestState(svc.lastAuthRequestFilter.State); got != "" {
		t.Fatalf("ListAuthRequests() state filter = %q, want empty for full inventory", got)
	}
	if got := len(m.authInventoryRequests); got != 1 {
		t.Fatalf("authInventoryRequests = %d, want 1 pending request", got)
	}
	if got := m.authInventoryRequests[0].ID; got != pending.ID {
		t.Fatalf("authInventoryRequests[0].ID = %q, want %q", got, pending.ID)
	}
	if got := len(m.authInventoryResolvedRequests); got != 1 {
		t.Fatalf("authInventoryResolvedRequests = %d, want 1 resolved request", got)
	}
	if got := m.authInventoryResolvedRequests[0].ID; got != approved.ID {
		t.Fatalf("authInventoryResolvedRequests[0].ID = %q, want %q", got, approved.ID)
	}
	m.authInventoryMoveSelection(1)
	rendered := stripANSI(fmt.Sprint(m.View()))
	for _, want := range []string{"selected resolved request", "approved-agent", "note:", "approved"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("View() missing %q for selected resolved request:\n%s", want, rendered)
		}
	}
}

// TestModelAuthInventoryApproveReturnsToInventory verifies project-scope review can approve and reopen the inventory list.
func TestModelAuthInventoryApproveReturnsToInventory(t *testing.T) {
	now := time.Date(2026, 3, 2, 9, 20, 45, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	request, err := domain.NewAuthRequest(domain.AuthRequestInput{
		ID:                  "req-approve-return",
		Path:                domain.AuthRequestPath{ProjectID: project.ID},
		PrincipalID:         "review-agent",
		PrincipalType:       "agent",
		PrincipalName:       "Review Agent",
		ClientID:            "till-mcp-stdio",
		ClientType:          "mcp-stdio",
		ClientName:          "Till MCP STDIO",
		RequestedSessionTTL: 8 * time.Hour,
		Reason:              "approve from inventory",
		RequestedByActor:    "lane-user",
		RequestedByType:     domain.ActorTypeUser,
		Timeout:             30 * time.Minute,
	}, now)
	if err != nil {
		t.Fatalf("NewAuthRequest() error = %v", err)
	}

	svc := newFakeService([]domain.Project{project}, []domain.Column{column}, []domain.Task{task})
	svc.authRequests[request.ID] = request

	m := loadReadyModel(t, NewModel(svc))
	m = applyCmd(t, m, m.startAuthInventory(false))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeAuthReview {
		t.Fatalf("expected enter to open auth review, got %v", m.mode)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeAuthInventory {
		t.Fatalf("expected approve to return to auth inventory, got %v", m.mode)
	}
	if got := strings.TrimSpace(svc.lastApproveAuthRequest.RequestID); got != request.ID {
		t.Fatalf("ApproveAuthRequest() request id = %q, want %q", got, request.ID)
	}
	if got := len(m.authInventoryRequests); got != 0 {
		t.Fatalf("authInventoryRequests after approve = %d, want 0", got)
	}
}

// TestModelAuthInventoryDenyReturnsToInventory verifies deny-with-note returns to the inventory after applying the decision.
func TestModelAuthInventoryDenyReturnsToInventory(t *testing.T) {
	now := time.Date(2026, 3, 2, 9, 20, 50, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	request, err := domain.NewAuthRequest(domain.AuthRequestInput{
		ID:                  "req-deny-return",
		Path:                domain.AuthRequestPath{ProjectID: project.ID},
		PrincipalID:         "review-agent",
		PrincipalType:       "agent",
		PrincipalName:       "Review Agent",
		ClientID:            "till-mcp-stdio",
		ClientType:          "mcp-stdio",
		ClientName:          "Till MCP STDIO",
		RequestedSessionTTL: 8 * time.Hour,
		Reason:              "deny from inventory",
		RequestedByActor:    "lane-user",
		RequestedByType:     domain.ActorTypeUser,
		Timeout:             30 * time.Minute,
	}, now)
	if err != nil {
		t.Fatalf("NewAuthRequest() error = %v", err)
	}

	svc := newFakeService([]domain.Project{project}, []domain.Column{column}, []domain.Task{task})
	svc.authRequests[request.ID] = request

	m := loadReadyModel(t, NewModel(svc))
	m = applyCmd(t, m, m.startAuthInventory(false))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeAuthReview {
		t.Fatalf("expected enter to open auth review, got %v", m.mode)
	}
	m = applyMsg(t, m, keyRune('d'))
	m.confirmAuthNoteInput.SetValue("denied from inventory review")
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeAuthInventory {
		t.Fatalf("expected deny to return to auth inventory, got mode=%v stage=%d status=%q", m.mode, m.authReviewStage, m.status)
	}
	if got := strings.TrimSpace(svc.lastDenyAuthRequest.RequestID); got != request.ID {
		t.Fatalf("DenyAuthRequest() request id = %q, want %q", got, request.ID)
	}
	if got := strings.TrimSpace(svc.lastDenyAuthRequest.ResolutionNote); got != "denied from inventory review" {
		t.Fatalf("DenyAuthRequest() note = %q, want denial note", got)
	}
	if got := len(m.authInventoryRequests); got != 0 {
		t.Fatalf("authInventoryRequests after deny = %d, want 0", got)
	}
}

// TestModelAuthReviewDenyUsesSingleConfirm verifies the deny review step keeps
// note entry simple and confirms on a single explicit enter action.
func TestModelAuthReviewDenyUsesSingleConfirm(t *testing.T) {
	now := time.Date(2026, 3, 2, 9, 20, 55, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	request, err := domain.NewAuthRequest(domain.AuthRequestInput{
		ID:                  "req-deny-confirm",
		Path:                domain.AuthRequestPath{ProjectID: project.ID},
		PrincipalID:         "review-agent",
		PrincipalType:       "agent",
		PrincipalName:       "Review Agent",
		ClientID:            "till-mcp-stdio",
		ClientType:          "mcp-stdio",
		ClientName:          "Till MCP STDIO",
		RequestedSessionTTL: 8 * time.Hour,
		Reason:              "deny confirmation review",
		RequestedByActor:    "lane-user",
		RequestedByType:     domain.ActorTypeUser,
		Timeout:             30 * time.Minute,
	}, now)
	if err != nil {
		t.Fatalf("NewAuthRequest() error = %v", err)
	}

	svc := newFakeService([]domain.Project{project}, []domain.Column{column}, []domain.Task{task})
	svc.authRequests[request.ID] = request

	m := loadReadyModel(t, NewModel(svc))
	m = applyCmd(t, m, m.startAuthInventory(false))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeAuthReview {
		t.Fatalf("expected enter to open auth review, got %v", m.mode)
	}
	m = applyMsg(t, m, keyRune('d'))
	m.confirmAuthNoteInput.SetValue("needs more scope review")
	rendered := fmt.Sprint(m.View())
	if !strings.Contains(rendered, "[confirm deny: enter]  [cancel: esc]") {
		t.Fatalf("View() missing deny confirm prompt:\n%s", rendered)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeAuthInventory {
		t.Fatalf("expected deny to return to auth inventory, got %v", m.mode)
	}
	if got := strings.TrimSpace(svc.lastDenyAuthRequest.RequestID); got != request.ID {
		t.Fatalf("DenyAuthRequest() request id = %q, want %q", got, request.ID)
	}
	if got := strings.TrimSpace(svc.lastDenyAuthRequest.ResolutionNote); got != "needs more scope review" {
		t.Fatalf("DenyAuthRequest() note = %q, want denial note", got)
	}
}

// TestModelAuthInventoryCanToggleGlobalAndRevokeSession verifies project/global inventory toggling and TUI revocation.
func TestModelAuthInventoryCanToggleGlobalAndRevokeSession(t *testing.T) {
	now := time.Date(2026, 3, 2, 9, 21, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	other, _ := domain.NewProject("p2", "Roadmap", "", now.Add(time.Minute))
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	svc := newFakeService([]domain.Project{project, other}, []domain.Column{column}, []domain.Task{task})
	svc.authSessions = append(svc.authSessions,
		app.AuthSession{
			SessionID:     "session-project",
			ProjectID:     project.ID,
			ApprovedPath:  "project/" + project.ID,
			PrincipalID:   "review-agent",
			PrincipalType: "agent",
			PrincipalName: "Review Agent",
			ClientID:      "till-mcp-stdio",
			ClientType:    "mcp-stdio",
			ClientName:    "Till MCP STDIO",
			ExpiresAt:     time.Now().UTC().Add(2 * time.Hour),
		},
		app.AuthSession{
			SessionID:     "session-other",
			ProjectID:     other.ID,
			ApprovedPath:  "project/" + other.ID,
			PrincipalID:   "roadmap-agent",
			PrincipalType: "agent",
			PrincipalName: "Roadmap Agent",
			ClientID:      "till-mcp-stdio",
			ClientType:    "mcp-stdio",
			ClientName:    "Till MCP STDIO",
			ExpiresAt:     time.Now().UTC().Add(3 * time.Hour),
		},
	)

	m := loadReadyModel(t, NewModel(svc))
	m = applyCmd(t, m, m.startAuthInventory(false))
	if got := len(m.authInventorySessions); got != 1 {
		t.Fatalf("project authInventorySessions = %d, want 1", got)
	}

	m = applyMsg(t, m, keyRune('g'))
	if !m.authInventoryGlobal {
		t.Fatal("expected auth inventory to toggle to global scope")
	}
	if got := strings.TrimSpace(svc.lastAuthSessionFilter.ProjectID); got != "" {
		t.Fatalf("global ListAuthSessions() project filter = %q, want empty", got)
	}
	if got := len(m.authInventorySessions); got != 2 {
		t.Fatalf("global authInventorySessions = %d, want 2", got)
	}

	m.authInventoryIndex = 0
	m = applyMsg(t, m, keyRune('r'))
	if m.mode != modeAuthSessionRevoke {
		t.Fatalf("expected revoke action to open dedicated revoke review, got %v", m.mode)
	}
	if !m.pendingConfirm.ReturnToAuthAccess {
		t.Fatal("expected revoke confirm to return to auth inventory")
	}
	rendered := fmt.Sprint(m.View())
	for _, want := range []string{"Revoke Active Session", "principal: Review Agent", "[enter] revoke and confirm", "[esc] cancel"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("View() missing %q in revoke review:\n%s", want, rendered)
		}
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeAuthInventory {
		t.Fatalf("expected confirmed revoke to return to auth inventory, got %v", m.mode)
	}
	if got := strings.TrimSpace(svc.lastRevokeAuthSessionID); got != "session-project" {
		t.Fatalf("RevokeAuthSession() session id = %q, want session-project", got)
	}
	if got := strings.TrimSpace(svc.lastRevokeAuthReason); got != "revoked via TUI auth inventory" {
		t.Fatalf("RevokeAuthSession() reason = %q, want TUI auth inventory reason", got)
	}
	if got := len(m.authInventorySessions); got != 1 {
		t.Fatalf("authInventorySessions after revoke = %d, want 1 active session remaining", got)
	}
}

// TestModelAuthInventoryEscapeReloadsBoard verifies exiting auth inventory can flush deferred board reloads after auth mutations.
func TestModelAuthInventoryEscapeReloadsBoard(t *testing.T) {
	now := time.Date(2026, 3, 2, 9, 22, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	initial, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "Initial",
		Priority:  domain.PriorityMedium,
	}, now)
	external, _ := domain.NewTask(domain.TaskInput{
		ID:        "t2",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  1,
		Title:     "External",
		Priority:  domain.PriorityMedium,
	}, now.Add(time.Minute))
	svc := newFakeService([]domain.Project{project}, []domain.Column{column}, []domain.Task{initial})
	m := loadReadyModel(t, NewModel(svc))
	m.mode = modeAuthInventory
	m.authInventoryNeedsReload = true
	svc.tasks[project.ID] = append(svc.tasks[project.ID], external)

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.mode != modeNone {
		t.Fatalf("expected escape to leave auth inventory, got %v", m.mode)
	}
	if m.authInventoryNeedsReload {
		t.Fatal("expected deferred auth inventory reload flag to clear on escape")
	}
	if got := len(m.tasks); got != 2 {
		t.Fatalf("expected board reload to pick up external task, got %d tasks", got)
	}
}

// TestModelActionMsgOpenAuthAccessReloadsInventory verifies auth-access action messages reopen and reload the auth inventory screen.
func TestModelActionMsgOpenAuthAccessReloadsInventory(t *testing.T) {
	now := time.Date(2026, 3, 2, 9, 23, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	request, err := domain.NewAuthRequest(domain.AuthRequestInput{
		ID:                  "req-open-auth-access",
		Path:                domain.AuthRequestPath{ProjectID: project.ID},
		PrincipalID:         "review-agent",
		PrincipalType:       "agent",
		ClientID:            "till-mcp-stdio",
		ClientType:          "mcp-stdio",
		RequestedSessionTTL: 8 * time.Hour,
		Reason:              "reload inventory",
		RequestedByActor:    "lane-user",
		RequestedByType:     domain.ActorTypeUser,
		Timeout:             30 * time.Minute,
	}, now)
	if err != nil {
		t.Fatalf("NewAuthRequest() error = %v", err)
	}
	svc := newFakeService([]domain.Project{project}, []domain.Column{column}, []domain.Task{task})
	svc.authRequests[request.ID] = request

	m := loadReadyModel(t, NewModel(svc))
	updated, cmd := m.Update(actionMsg{status: "auth changed", reload: true, openAuthAccess: true})
	m = applyCmd(t, mustModelValue(t, updated), cmd)
	if m.mode != modeAuthInventory {
		t.Fatalf("expected actionMsg to reopen auth inventory, got %v", m.mode)
	}
	if !m.authInventoryNeedsReload {
		t.Fatal("expected auth inventory to mark deferred board reload after auth change")
	}
	if got := len(m.authInventoryRequests); got != 1 {
		t.Fatalf("authInventoryRequests = %d, want 1", got)
	}
}

// TestModelProjectNotificationsEnterRecoversArchivedTask verifies project-notification Enter can reopen an archived hidden task before falling back to a thread.
func TestModelProjectNotificationsEnterRecoversArchivedTask(t *testing.T) {
	now := time.Date(2026, 3, 2, 9, 15, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	archivedBlocked, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-archived",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "Archived Blocked Task",
		Priority:  domain.PriorityHigh,
		Metadata: domain.TaskMetadata{
			BlockedReason: "requires archived review",
		},
	}, now)
	archivedBlocked.Archive(now.Add(time.Minute))

	m := loadReadyModel(t, NewModel(newFakeService(
		[]domain.Project{project},
		[]domain.Column{column},
		[]domain.Task{archivedBlocked},
	)))

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	m = applyMsg(t, m, keyRune('k'))
	m = applyMsg(t, m, keyRune('k'))
	if m.noticesSection != noticesSectionAttention {
		t.Fatalf("expected attention section focus, got %v", m.noticesSection)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeTaskInfo {
		t.Fatalf("expected archived task notice to recover into task info, got %v", m.mode)
	}
	if m.taskInfoTaskID != archivedBlocked.ID {
		t.Fatalf("expected task info target %q, got %q", archivedBlocked.ID, m.taskInfoTaskID)
	}
	if !m.showArchived {
		t.Fatal("expected archived task recovery to enable archived visibility")
	}
}

// TestModelProjectNotificationsScopedRowsFallbackToProjectThread verifies scoped notices with malformed scope metadata still produce deterministic thread navigation.
func TestModelProjectNotificationsScopedRowsFallbackToProjectThread(t *testing.T) {
	now := time.Date(2026, 3, 2, 9, 20, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)

	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{project}, []domain.Column{column}, []domain.Task{task})))
	m.attentionItems = []domain.AttentionItem{
		{
			ID:                 "att-malformed-scope",
			ProjectID:          project.ID,
			ScopeType:          domain.ScopeLevelTask,
			ScopeID:            "",
			State:              domain.AttentionStateOpen,
			Kind:               domain.AttentionKindConsensusRequired,
			Summary:            "malformed scope metadata",
			RequiresUserAction: true,
		},
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	m = applyMsg(t, m, keyRune('k'))
	m = applyMsg(t, m, keyRune('k'))
	if m.noticesSection != noticesSectionAttention {
		t.Fatalf("expected attention section focus, got %v", m.noticesSection)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeThread {
		t.Fatalf("expected malformed scoped row to open fallback thread mode, got %v", m.mode)
	}
	if m.threadTarget.ProjectID != project.ID || m.threadTarget.TargetType != domain.CommentTargetTypeProject || m.threadTarget.TargetID != project.ID {
		t.Fatalf("expected project-thread fallback target for malformed scope, got %#v", m.threadTarget)
	}
}

// TestModelPanelFocusTraversalIncludesGlobalNotifications verifies board/project/global panel focus traversal.
func TestModelPanelFocusTraversalIncludesGlobalNotifications(t *testing.T) {
	now := time.Date(2026, 3, 1, 13, 28, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})))
	m.globalNotices = []globalNoticesPanelItem{
		{
			StableKey:    globalNoticesStableKey(p.ID, "att-focus", domain.ScopeLevelTask, task.ID, "focus traversal"),
			AttentionID:  "att-focus",
			ProjectID:    p.ID,
			ProjectLabel: p.Name,
			ScopeType:    domain.ScopeLevelTask,
			ScopeID:      task.ID,
			Summary:      "focus traversal",
			TaskID:       task.ID,
		},
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	if !m.noticesFocused || m.noticesPanel != noticesPanelFocusProject {
		t.Fatalf("expected project notifications focus after tab, got noticesFocused=%t panel=%v", m.noticesFocused, m.noticesPanel)
	}

	m = applyMsg(t, m, keyRune('l'))
	if !m.noticesFocused || m.noticesPanel != noticesPanelFocusGlobal {
		t.Fatalf("expected global notifications focus after right, got noticesFocused=%t panel=%v", m.noticesFocused, m.noticesPanel)
	}

	m = applyMsg(t, m, keyRune('h'))
	if !m.noticesFocused || m.noticesPanel != noticesPanelFocusProject {
		t.Fatalf("expected project notifications focus after left, got noticesFocused=%t panel=%v", m.noticesFocused, m.noticesPanel)
	}

	m = applyMsg(t, m, keyRune('h'))
	if m.noticesFocused {
		t.Fatalf("expected board focus after moving left from project notifications, got noticesFocused=%t", m.noticesFocused)
	}

	m = applyMsg(t, m, keyRune('h'))
	if !m.noticesFocused || m.noticesPanel != noticesPanelFocusGlobal {
		t.Fatalf("expected left from board to wrap to global notifications, got noticesFocused=%t panel=%v", m.noticesFocused, m.noticesPanel)
	}

	m = applyMsg(t, m, keyRune('l'))
	if m.noticesFocused || m.selectedColumn != 0 {
		t.Fatalf("expected right from global notifications to wrap back to board, got noticesFocused=%t selectedColumn=%d", m.noticesFocused, m.selectedColumn)
	}
}

// TestModelGlobalNotificationsEnterSwitchesProjectAndOpensTaskInfo verifies global notifications Enter performs deterministic cross-project navigation.
func TestModelGlobalNotificationsEnterSwitchesProjectAndOpensTaskInfo(t *testing.T) {
	now := time.Date(2026, 3, 1, 13, 29, 0, 0, time.UTC)
	p1, _ := domain.NewProject("p1", "Inbox", "", now)
	p2, _ := domain.NewProject("p2", "Roadmap", "", now.Add(time.Minute))
	c1, _ := domain.NewColumn("c1", p1.ID, "To Do", 0, 0, now)
	c2, _ := domain.NewColumn("c2", p2.ID, "To Do", 0, 0, now)
	base, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p1.ID,
		ColumnID:  c1.ID,
		Position:  0,
		Title:     "Base",
		Priority:  domain.PriorityLow,
	}, now)
	blocked, _ := domain.NewTask(domain.TaskInput{
		ID:        "t2",
		ProjectID: p2.ID,
		ColumnID:  c2.ID,
		Position:  0,
		Title:     "Cross Project Blocked",
		Priority:  domain.PriorityHigh,
		Metadata: domain.TaskMetadata{
			BlockedReason: "waiting for external approval",
		},
	}, now.Add(2*time.Minute))

	svc := newFakeService(
		[]domain.Project{p1, p2},
		[]domain.Column{c1, c2},
		[]domain.Task{base, blocked},
	)
	m := loadReadyModel(t, NewModel(svc))
	if m.projects[m.selectedProject].ID != p1.ID {
		t.Fatalf("expected initial project %q, got %q", p1.ID, m.projects[m.selectedProject].ID)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	m = applyMsg(t, m, keyRune('l'))
	if !m.noticesFocused || m.noticesPanel != noticesPanelFocusGlobal {
		t.Fatalf("expected global notifications focus before activation, got noticesFocused=%t panel=%v", m.noticesFocused, m.noticesPanel)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeTaskInfo {
		t.Fatalf("expected global notification enter to open task info, got %v", m.mode)
	}
	if m.taskInfoTaskID != blocked.ID {
		t.Fatalf("expected cross-project task-info target %q, got %q", blocked.ID, m.taskInfoTaskID)
	}
	if m.projects[m.selectedProject].ID != p2.ID {
		t.Fatalf("expected project context to switch to %q, got %q", p2.ID, m.projects[m.selectedProject].ID)
	}
	if m.noticesFocused {
		t.Fatalf("expected notices focus cleared after global notification activation, got noticesFocused=%t", m.noticesFocused)
	}
}

// TestModelAuthReviewCanSwitchDecisionBeforeApply verifies review can branch from approve-default into deny flow.
func TestModelAuthReviewCanSwitchDecisionBeforeApply(t *testing.T) {
	now := time.Date(2026, 3, 2, 9, 29, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	authRequest, err := domain.NewAuthRequest(domain.AuthRequestInput{
		ID:                  "req-switch-decision",
		Path:                domain.AuthRequestPath{ProjectID: project.ID},
		PrincipalID:         "agent-switch",
		PrincipalType:       "agent",
		PrincipalName:       "Agent Switch",
		ClientID:            "till-mcp-stdio",
		ClientType:          "mcp-stdio",
		ClientName:          "Till MCP STDIO",
		RequestedSessionTTL: 8 * time.Hour,
		Reason:              "switch decision test",
		RequestedByActor:    "lane-user",
		RequestedByType:     domain.ActorTypeUser,
		Timeout:             30 * time.Minute,
	}, now)
	if err != nil {
		t.Fatalf("NewAuthRequest() error = %v", err)
	}
	svc := newFakeService([]domain.Project{project}, []domain.Column{column}, []domain.Task{task})
	svc.authRequests[authRequest.ID] = authRequest
	svc.attentionItemsByProject[project.ID] = []domain.AttentionItem{{
		ID:                 authRequest.ID,
		ProjectID:          project.ID,
		ScopeType:          domain.ScopeLevelProject,
		ScopeID:            project.ID,
		State:              domain.AttentionStateOpen,
		Kind:               domain.AttentionKindConsensusRequired,
		Summary:            "auth request review",
		RequiresUserAction: true,
		CreatedAt:          now,
	}}

	m := loadReadyModel(t, NewModel(svc))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	m = applyMsg(t, m, keyRune('k'))
	m = applyMsg(t, m, keyRune('k'))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeAuthReview {
		t.Fatalf("expected auth review mode, got %v", m.mode)
	}
	m = applyMsg(t, m, keyRune('d'))
	if got := m.authReviewStage; got != authReviewStageDeny {
		t.Fatalf("authReviewStage = %d, want deny stage", got)
	}
	m.confirmAuthNoteInput.SetValue("switched to deny after review")
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if got := strings.TrimSpace(svc.lastDenyAuthRequest.RequestID); got != authRequest.ID {
		t.Fatalf("expected deny request id %q, got %q (mode=%v stage=%d status=%q approveID=%q)", authRequest.ID, got, m.mode, m.authReviewStage, m.status, strings.TrimSpace(svc.lastApproveAuthRequest.RequestID))
	}
	if got := strings.TrimSpace(svc.lastDenyAuthRequest.ResolutionNote); got != "switched to deny after review" {
		t.Fatalf("expected switched denial note, got %q", got)
	}
	if got := strings.TrimSpace(svc.lastApproveAuthRequest.RequestID); got != "" {
		t.Fatalf("unexpected approve request id %q after switching to deny", got)
	}
}

// TestModelGlobalNotificationsAuthRequestDenyShortcut verifies global auth-request rows open the deny-note review stage.
func TestModelGlobalNotificationsAuthRequestDenyShortcut(t *testing.T) {
	now := time.Date(2026, 3, 1, 13, 33, 0, 0, time.UTC)
	p1, _ := domain.NewProject("p1", "Inbox", "", now)
	p2, _ := domain.NewProject("p2", "Roadmap", "", now.Add(time.Minute))
	c1, _ := domain.NewColumn("c1", p1.ID, "To Do", 0, 0, now)
	c2, _ := domain.NewColumn("c2", p2.ID, "To Do", 0, 0, now)
	task1, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p1.ID,
		ColumnID:  c1.ID,
		Position:  0,
		Title:     "Inbox Task",
		Priority:  domain.PriorityMedium,
	}, now)
	task2, _ := domain.NewTask(domain.TaskInput{
		ID:        "t2",
		ProjectID: p2.ID,
		ColumnID:  c2.ID,
		Position:  0,
		Title:     "Roadmap Task",
		Priority:  domain.PriorityMedium,
	}, now)
	authRequest, err := domain.NewAuthRequest(domain.AuthRequestInput{
		ID:                  "req-2",
		Path:                domain.AuthRequestPath{ProjectID: p2.ID},
		PrincipalID:         "agent-2",
		PrincipalType:       "agent",
		PrincipalName:       "Agent Two",
		ClientID:            "till-mcp-stdio",
		ClientType:          "mcp-stdio",
		ClientName:          "Till MCP STDIO",
		RequestedSessionTTL: 8 * time.Hour,
		Reason:              "scope denied",
		RequestedByActor:    "lane-user",
		RequestedByType:     domain.ActorTypeUser,
		Timeout:             30 * time.Minute,
	}, now)
	if err != nil {
		t.Fatalf("NewAuthRequest() error = %v", err)
	}
	svc := newFakeService([]domain.Project{p1, p2}, []domain.Column{c1, c2}, []domain.Task{task1, task2})
	svc.authRequests[authRequest.ID] = authRequest
	m := loadReadyModel(t, NewModel(svc))
	m.globalNotices = []globalNoticesPanelItem{
		{
			StableKey:         globalNoticesStableKey(p2.ID, authRequest.ID, domain.ScopeLevelProject, p2.ID, "auth request review"),
			AttentionID:       authRequest.ID,
			ProjectID:         p2.ID,
			ProjectLabel:      p2.Name,
			ScopeType:         domain.ScopeLevelProject,
			ScopeID:           p2.ID,
			Summary:           "auth request review",
			ThreadDescription: "Please review this denial request.",
		},
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	m = applyMsg(t, m, keyRune('l'))
	if !m.noticesFocused || m.noticesPanel != noticesPanelFocusGlobal {
		t.Fatalf("expected global notifications focus, got noticesFocused=%t panel=%v", m.noticesFocused, m.noticesPanel)
	}
	m = applyMsg(t, m, keyRune('d'))
	if m.mode != modeAuthReview {
		t.Fatalf("expected deny shortcut to open auth review, got %v", m.mode)
	}
	if m.pendingConfirm.Kind != "deny-auth-request" {
		t.Fatalf("expected deny confirm kind, got %q", m.pendingConfirm.Kind)
	}
	if m.pendingConfirm.AuthRequestID != authRequest.ID {
		t.Fatalf("expected confirm auth request id %q, got %q", authRequest.ID, m.pendingConfirm.AuthRequestID)
	}
	if got := m.authReviewStage; got != authReviewStageDeny {
		t.Fatalf("authReviewStage = %d, want deny stage", got)
	}
	m.confirmAuthNoteInput.SetValue("denied in global panel for missing scope")
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if got := strings.TrimSpace(svc.lastDenyAuthRequest.RequestID); got != authRequest.ID {
		t.Fatalf("expected deny request id %q, got %q (mode=%v stage=%d status=%q)", authRequest.ID, got, m.mode, m.authReviewStage, m.status)
	}
	if got := strings.TrimSpace(svc.lastDenyAuthRequest.ResolutionNote); got != "denied in global panel for missing scope" {
		t.Fatalf("expected edited denial note to be forwarded, got %q", got)
	}
}

// TestModelGlobalNotificationsEnterOpensAuthReview verifies enter on a global auth-request row opens the auth review surface.
func TestModelGlobalNotificationsEnterOpensAuthReview(t *testing.T) {
	now := time.Date(2026, 3, 2, 9, 25, 0, 0, time.UTC)
	p1, _ := domain.NewProject("p1", "Inbox", "", now)
	p2, _ := domain.NewProject("p2", "Roadmap", "", now.Add(time.Minute))
	c1, _ := domain.NewColumn("c1", p1.ID, "To Do", 0, 0, now)
	c2, _ := domain.NewColumn("c2", p2.ID, "To Do", 0, 0, now)
	task1, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p1.ID,
		ColumnID:  c1.ID,
		Position:  0,
		Title:     "Inbox Task",
		Priority:  domain.PriorityMedium,
	}, now)
	task2, _ := domain.NewTask(domain.TaskInput{
		ID:        "t2",
		ProjectID: p2.ID,
		ColumnID:  c2.ID,
		Position:  0,
		Title:     "Roadmap Task",
		Priority:  domain.PriorityMedium,
	}, now)
	authRequest, err := domain.NewAuthRequest(domain.AuthRequestInput{
		ID:                  "req-global-enter",
		Path:                domain.AuthRequestPath{ProjectID: p2.ID},
		PrincipalID:         "agent-2",
		PrincipalType:       "agent",
		PrincipalName:       "Agent Two",
		ClientID:            "till-mcp-stdio",
		ClientType:          "mcp-stdio",
		ClientName:          "Till MCP STDIO",
		RequestedSessionTTL: 8 * time.Hour,
		Reason:              "global review",
		RequestedByActor:    "lane-user",
		RequestedByType:     domain.ActorTypeUser,
		Timeout:             30 * time.Minute,
	}, now)
	if err != nil {
		t.Fatalf("NewAuthRequest() error = %v", err)
	}
	svc := newFakeService([]domain.Project{p1, p2}, []domain.Column{c1, c2}, []domain.Task{task1, task2})
	svc.authRequests[authRequest.ID] = authRequest
	m := loadReadyModel(t, NewModel(svc))
	m.globalNotices = []globalNoticesPanelItem{{
		StableKey:         globalNoticesStableKey(p2.ID, authRequest.ID, domain.ScopeLevelProject, p2.ID, "auth request review"),
		AttentionID:       authRequest.ID,
		ProjectID:         p2.ID,
		ProjectLabel:      p2.Name,
		ScopeType:         domain.ScopeLevelProject,
		ScopeID:           p2.ID,
		Summary:           "auth request review",
		ThreadDescription: "Please review this request.",
	}}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	m = applyMsg(t, m, keyRune('l'))
	if !m.noticesFocused || m.noticesPanel != noticesPanelFocusGlobal {
		t.Fatalf("expected global notifications focus, got noticesFocused=%t panel=%v", m.noticesFocused, m.noticesPanel)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeAuthReview {
		t.Fatalf("expected enter to open auth review surface, got %v", m.mode)
	}
	if got := strings.TrimSpace(m.pendingConfirm.AuthRequestID); got != authRequest.ID {
		t.Fatalf("pendingConfirm.AuthRequestID = %q, want %q", got, authRequest.ID)
	}
	if got := strings.TrimSpace(m.pendingConfirm.AuthRequestDecision); got != "approve" {
		t.Fatalf("pendingConfirm.AuthRequestDecision = %q, want approve", got)
	}
}

// TestGlobalNoticesPanelItemFromAttentionCarriesStableIdentifiers verifies global-row mapping keeps stable row identifiers.
func TestGlobalNoticesPanelItemFromAttentionCarriesStableIdentifiers(t *testing.T) {
	now := time.Date(2026, 3, 1, 13, 30, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	item := domain.AttentionItem{
		ID:           "att-1",
		ProjectID:    project.ID,
		ScopeType:    domain.ScopeLevelProject,
		ScopeID:      project.ID,
		State:        domain.AttentionStateOpen,
		Kind:         domain.AttentionKindBlocker,
		Summary:      "project-level blocker",
		BodyMarkdown: "## Context\n\nNeeds a scoped response.",
	}

	row := globalNoticesPanelItemFromAttention(project, item)
	if row.StableKey == "" {
		t.Fatal("expected non-empty stable key")
	}
	if row.AttentionID != item.ID {
		t.Fatalf("expected attention id %q, got %q", item.ID, row.AttentionID)
	}
	if row.ScopeType != domain.ScopeLevelProject || row.ScopeID != project.ID {
		t.Fatalf("expected project scope tuple, got scopeType=%q scopeID=%q", row.ScopeType, row.ScopeID)
	}
	if row.StableKey != globalNoticesStableKey(project.ID, item.ID, item.ScopeType, item.ScopeID, item.Summary) {
		t.Fatalf("expected deterministic stable key, got %q", row.StableKey)
	}
	if row.TaskID != "" {
		t.Fatalf("expected project-scoped row without task id, got %q", row.TaskID)
	}
	if row.ThreadDescription != strings.TrimSpace(item.BodyMarkdown) {
		t.Fatalf("expected thread description to carry body markdown, got %q", row.ThreadDescription)
	}
}

// TestModelGlobalNotificationsSelectionReanchorsByStableKey verifies global-row selection survives reload reorder by stable row key.
func TestModelGlobalNotificationsSelectionReanchorsByStableKey(t *testing.T) {
	now := time.Date(2026, 3, 1, 13, 31, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{project}, []domain.Column{column}, []domain.Task{task})))

	noticeA := globalNoticesPanelItem{
		StableKey:    "row-a",
		AttentionID:  "att-a",
		ProjectID:    project.ID,
		ProjectLabel: project.Name,
		ScopeType:    domain.ScopeLevelTask,
		ScopeID:      "t-a",
		Summary:      "A",
		TaskID:       "t-a",
	}
	noticeB := globalNoticesPanelItem{
		StableKey:    "row-b",
		AttentionID:  "att-b",
		ProjectID:    project.ID,
		ProjectLabel: project.Name,
		ScopeType:    domain.ScopeLevelTask,
		ScopeID:      "t-b",
		Summary:      "B",
		TaskID:       "t-b",
	}
	m.globalNotices = []globalNoticesPanelItem{noticeA, noticeB}
	m.globalNoticesIdx = 1

	reloaded := loadedMsg{
		projects:        m.projects,
		selectedProject: m.selectedProject,
		columns:         m.columns,
		tasks:           m.tasks,
		globalNotices: []globalNoticesPanelItem{
			noticeB,
			noticeA,
		},
		rollup: m.dependencyRollup,
	}
	m = applyMsg(t, m, reloaded)
	if m.globalNoticesIdx != 0 {
		t.Fatalf("expected selection re-anchored to row-b at index 0, got %d", m.globalNoticesIdx)
	}
	selected, ok := m.selectedGlobalNoticesItem()
	if !ok || selected.StableKey != "row-b" {
		t.Fatalf("expected stable-key row-b still selected, got %#v ok=%t", selected, ok)
	}
}

// TestModelGlobalNotificationsEnterOnProjectScopedRowOpensThread verifies non-task global rows open scoped comment threads.
func TestModelGlobalNotificationsEnterOnProjectScopedRowOpensThread(t *testing.T) {
	now := time.Date(2026, 3, 1, 13, 32, 0, 0, time.UTC)
	p1, _ := domain.NewProject("p1", "Inbox", "", now)
	p2, _ := domain.NewProject("p2", "Roadmap", "", now.Add(time.Minute))
	c1, _ := domain.NewColumn("c1", p1.ID, "To Do", 0, 0, now)
	c2, _ := domain.NewColumn("c2", p2.ID, "To Do", 0, 0, now)
	t1, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p1.ID,
		ColumnID:  c1.ID,
		Position:  0,
		Title:     "Inbox Task",
		Priority:  domain.PriorityMedium,
	}, now)
	t2, _ := domain.NewTask(domain.TaskInput{
		ID:        "t2",
		ProjectID: p2.ID,
		ColumnID:  c2.ID,
		Position:  0,
		Title:     "Roadmap Task",
		Priority:  domain.PriorityMedium,
	}, now)
	m := loadReadyModel(t, NewModel(newFakeService(
		[]domain.Project{p1, p2},
		[]domain.Column{c1, c2},
		[]domain.Task{t1, t2},
	)))

	m.globalNotices = []globalNoticesPanelItem{
		{
			StableKey:         globalNoticesStableKey(p2.ID, "att-project", domain.ScopeLevelProject, p2.ID, "Project-level action"),
			AttentionID:       "att-project",
			ProjectID:         p2.ID,
			ProjectLabel:      p2.Name,
			ScopeType:         domain.ScopeLevelProject,
			ScopeID:           p2.ID,
			Summary:           "Project-level action",
			ThreadDescription: "## Next Step\n\nCapture the decision in-thread.",
		},
	}
	m.noticesFocused = true
	m.noticesPanel = noticesPanelFocusGlobal
	initialProjectID := m.projects[m.selectedProject].ID

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeThread {
		t.Fatalf("expected project-scoped global notice to open thread mode, got %v", m.mode)
	}
	if got := m.projects[m.selectedProject].ID; got != initialProjectID {
		t.Fatalf("expected current project context to remain %q on direct thread open, got %q", initialProjectID, got)
	}
	if m.threadTarget.ProjectID != p2.ID || m.threadTarget.TargetType != domain.CommentTargetTypeProject || m.threadTarget.TargetID != p2.ID {
		t.Fatalf("expected project-scoped thread target for %q, got %#v", p2.ID, m.threadTarget)
	}
	if !strings.Contains(m.threadDescriptionMarkdown, "Next Step") {
		t.Fatalf("expected thread view to include notification markdown body, got %q", m.threadDescriptionMarkdown)
	}
	if m.noticesFocused {
		t.Fatalf("expected notices focus cleared after thread open, got noticesFocused=%t", m.noticesFocused)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.mode != modeNone {
		t.Fatalf("expected esc to close thread modal opened from global notice, got %v", m.mode)
	}
}

// TestModelProjectNotificationsEnterRecoversFromSearchAndArchivedFilters verifies project-row enter recovery when the target task is hidden by filters.
func TestModelProjectNotificationsEnterRecoversFromSearchAndArchivedFilters(t *testing.T) {
	now := time.Date(2026, 3, 1, 13, 33, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	active, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-active",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "Visible Active Task",
		Priority:  domain.PriorityMedium,
	}, now)
	archivedBlocked, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-archived",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  1,
		Title:     "Archived Blocked Task",
		Priority:  domain.PriorityHigh,
		Metadata: domain.TaskMetadata{
			BlockedReason: "requires archived review",
		},
	}, now.Add(time.Minute))
	archivedBlocked.Archive(now.Add(2 * time.Minute))

	m := loadReadyModel(t, NewModel(newFakeService(
		[]domain.Project{project},
		[]domain.Column{column},
		[]domain.Task{active, archivedBlocked},
	)))
	if len(m.attentionItems) == 0 {
		t.Fatal("expected archived blocked task to appear in project notifications")
	}
	m.searchApplied = true
	m.searchQuery = "visible active"
	m.showArchived = false

	sections := m.noticesSectionsForInteraction()
	targetSection := noticesSectionRecentActivity
	found := false
	for _, section := range sections {
		for idx, item := range section.Items {
			if item.TaskID != archivedBlocked.ID {
				continue
			}
			targetSection = section.ID
			m.setNoticesSelectionIndex(section.ID, idx)
			found = true
			break
		}
		if found {
			break
		}
	}
	if !found {
		t.Fatal("expected archived blocked task row in project notifications")
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	m.noticesSection = targetSection
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})

	if m.mode != modeTaskInfo {
		t.Fatalf("expected hidden project target to recover into task info, got %v", m.mode)
	}
	if m.taskInfoTaskID != archivedBlocked.ID {
		t.Fatalf("expected archived task-info target %q, got %q", archivedBlocked.ID, m.taskInfoTaskID)
	}
	if !m.showArchived {
		t.Fatal("expected global notice activation to enable archived visibility for archived target")
	}
	if m.searchApplied || m.searchQuery != "" {
		t.Fatalf("expected search filters cleared during recovery, got searchApplied=%t query=%q", m.searchApplied, m.searchQuery)
	}
}

// TestModelGlobalNoticesAggregationDegradesOnNonActiveProjectFailures verifies non-active project notice failures do not abort board load.
func TestModelGlobalNoticesAggregationDegradesOnNonActiveProjectFailures(t *testing.T) {
	now := time.Date(2026, 3, 1, 13, 34, 0, 0, time.UTC)
	p1, _ := domain.NewProject("p1", "Inbox", "", now)
	p2, _ := domain.NewProject("p2", "Roadmap", "", now.Add(time.Minute))
	c1, _ := domain.NewColumn("c1", p1.ID, "To Do", 0, 0, now)
	c2, _ := domain.NewColumn("c2", p2.ID, "To Do", 0, 0, now)
	t1, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p1.ID,
		ColumnID:  c1.ID,
		Position:  0,
		Title:     "Inbox Blocked",
		Priority:  domain.PriorityHigh,
		Metadata: domain.TaskMetadata{
			BlockedReason: "active project blocker",
		},
	}, now)
	t2, _ := domain.NewTask(domain.TaskInput{
		ID:        "t2",
		ProjectID: p2.ID,
		ColumnID:  c2.ID,
		Position:  0,
		Title:     "Roadmap Blocked",
		Priority:  domain.PriorityHigh,
		Metadata: domain.TaskMetadata{
			BlockedReason: "non-active project blocker",
		},
	}, now.Add(time.Minute))
	svc := newFakeService(
		[]domain.Project{p1, p2},
		[]domain.Column{c1, c2},
		[]domain.Task{t1, t2},
	)
	svc.attentionErrByProject[p2.ID] = errors.New("attention load failed")

	m := loadReadyModel(t, NewModel(svc))
	if m.err != nil {
		t.Fatalf("expected board load to continue on non-active attention failure, got err=%v", m.err)
	}
	if m.globalNoticesPartialCount != 1 {
		t.Fatalf("expected partial-count=1, got %d", m.globalNoticesPartialCount)
	}
	if got := strings.Join(m.warnings, " | "); !strings.Contains(got, "global notices partial") {
		t.Fatalf("expected warnings to include partial-results signal, got %q", got)
	}

	m = applyMsg(t, m, tea.WindowSizeMsg{Width: 128, Height: 40})
	rendered := stripANSI(fmt.Sprint(m.View().Content))
	if !strings.Contains(rendered, "partial results: 1 project") {
		t.Fatalf("expected visible partial-results signal in global panel, got\n%s", rendered)
	}
}

// TestModelNoticesRecentActivityScrollAndFallbackDetail verifies notices activity scrolling and detail fallback for non-node events.
func TestModelNoticesRecentActivityScrollAndFallbackDetail(t *testing.T) {
	now := time.Date(2026, 3, 1, 13, 30, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})
	svc.changeEvents[p.ID] = []domain.ChangeEvent{
		{
			ID:        6,
			ProjectID: p.ID,
			Operation: domain.ChangeOperationUpdate,
			ActorID:   "user-owner",
			ActorType: domain.ActorTypeUser,
			Metadata: map[string]string{
				"title":      "Queue Event",
				"item_scope": "queued-action",
			},
			OccurredAt: now.Add(6 * time.Minute),
		},
		{
			ID:         5,
			ProjectID:  p.ID,
			WorkItemID: task.ID,
			Operation:  domain.ChangeOperationArchive,
			Metadata:   map[string]string{"title": task.Title},
			OccurredAt: now.Add(5 * time.Minute),
		},
		{
			ID:         4,
			ProjectID:  p.ID,
			WorkItemID: task.ID,
			Operation:  domain.ChangeOperationMove,
			Metadata:   map[string]string{"title": task.Title},
			OccurredAt: now.Add(4 * time.Minute),
		},
		{
			ID:         3,
			ProjectID:  p.ID,
			WorkItemID: task.ID,
			Operation:  domain.ChangeOperationUpdate,
			Metadata:   map[string]string{"title": task.Title},
			OccurredAt: now.Add(3 * time.Minute),
		},
		{
			ID:         2,
			ProjectID:  p.ID,
			WorkItemID: task.ID,
			Operation:  domain.ChangeOperationCreate,
			Metadata:   map[string]string{"title": task.Title},
			OccurredAt: now.Add(2 * time.Minute),
		},
		{
			ID:         1,
			ProjectID:  p.ID,
			WorkItemID: task.ID,
			Operation:  domain.ChangeOperationUpdate,
			Metadata:   map[string]string{"title": task.Title},
			OccurredAt: now.Add(time.Minute),
		},
	}
	m := loadReadyModel(t, NewModel(svc))

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	if m.noticesSection != noticesSectionRecentActivity {
		t.Fatalf("expected recent-activity section after notices focus, got %v", m.noticesSection)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeActivityEventInfo {
		t.Fatalf("expected enter fallback to open activity event detail for non-node event, got %v", m.mode)
	}
	m.mode = modeNone
	m.noticesFocused = true

	for i := 0; i < 4; i++ {
		m = applyMsg(t, m, keyRune('j'))
	}
	if m.noticesActivity != 4 {
		t.Fatalf("expected notices activity cursor to reach older row index 4, got %d", m.noticesActivity)
	}
	panel := stripANSI(m.renderOverviewPanel(p, lipgloss.Color("62"), lipgloss.Color("241"), lipgloss.Color("239"), 80, 40, 0, 0, 0, nil, true))
	if !strings.Contains(panel, "Recent Activity") {
		t.Fatalf("expected recent-activity section to stay visible while scrolled, got %q", panel)
	}
	if !strings.Contains(panel, "↑ more") {
		t.Fatalf("expected scrolled activity window to show overflow marker, got %q", panel)
	}
}

// TestModelActivityEventJumpLoadsArchivedTask verifies jump-to-node can recover archived nodes hidden by board filters.
func TestModelActivityEventJumpLoadsArchivedTask(t *testing.T) {
	now := time.Date(2026, 3, 1, 14, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	active, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Active",
		Priority:  domain.PriorityLow,
	}, now)
	archived, _ := domain.NewTask(domain.TaskInput{
		ID:        "t2",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  1,
		Title:     "Archived",
		Priority:  domain.PriorityLow,
	}, now.Add(time.Minute))
	archivedAt := now.Add(2 * time.Minute)
	archived.Archive(archivedAt)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{active, archived})
	svc.changeEvents[p.ID] = []domain.ChangeEvent{
		{
			ID:         3,
			ProjectID:  p.ID,
			WorkItemID: archived.ID,
			Operation:  domain.ChangeOperationArchive,
			ActorID:    "agent-ops",
			ActorType:  domain.ActorTypeAgent,
			Metadata:   map[string]string{"title": archived.Title},
			OccurredAt: archivedAt,
		},
	}
	m := loadReadyModel(t, NewModel(svc))
	if m.showArchived {
		t.Fatal("expected archived filter hidden by default")
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeActivityEventInfo {
		t.Fatalf("expected activity event info modal, got %v", m.mode)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if !m.showArchived {
		t.Fatal("expected jump flow to enable archived visibility")
	}
	if task, ok := m.selectedTaskInCurrentColumn(); !ok || task.ID != archived.ID {
		t.Fatalf("expected jump to archived node, got %#v ok=%t", task, ok)
	}
	if !strings.Contains(m.status, "jumped to activity node") {
		t.Fatalf("expected success status after archived jump, got %q", m.status)
	}
}

// TestModelActivityEventJumpFocusesNestedNode verifies jump-to-node focuses nested activity targets by scoping to parent.
func TestModelActivityEventJumpFocusesNestedNode(t *testing.T) {
	now := time.Date(2026, 3, 1, 15, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	branch, _ := domain.NewTask(domain.TaskInput{
		ID:        "b1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Branch",
		Priority:  domain.PriorityMedium,
		Scope:     "branch",
		Kind:      "branch",
	}, now)
	child, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ParentID:  branch.ID,
		ColumnID:  c.ID,
		Position:  1,
		Title:     "Nested Task",
		Priority:  domain.PriorityMedium,
	}, now.Add(time.Minute))
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{branch, child})
	svc.changeEvents[p.ID] = []domain.ChangeEvent{
		{
			ID:         1,
			ProjectID:  p.ID,
			WorkItemID: child.ID,
			Operation:  domain.ChangeOperationUpdate,
			ActorID:    "agent-sync",
			ActorType:  domain.ActorTypeAgent,
			Metadata:   map[string]string{"title": child.Title},
			OccurredAt: now.Add(2 * time.Minute),
		},
	}
	m := loadReadyModel(t, NewModel(svc))

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeActivityEventInfo {
		t.Fatalf("expected activity event info modal, got %v", m.mode)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeNone {
		t.Fatalf("expected board mode after jump, got %v", m.mode)
	}
	if got := strings.TrimSpace(m.projectionRootTaskID); got != branch.ID {
		t.Fatalf("expected jump to scope board to parent branch %q, got %q", branch.ID, got)
	}
	task, ok := m.selectedTaskInCurrentColumn()
	if !ok || task.ID != child.ID {
		t.Fatalf("expected nested task focused after jump, got %#v ok=%t", task, ok)
	}
	if !strings.Contains(m.status, "jumped to activity node") {
		t.Fatalf("expected jump success status, got %q", m.status)
	}
}

// TestModelActivityEventMetadataShowsColumnNames verifies move metadata renders column names instead of UUIDs.
func TestModelActivityEventMetadataShowsColumnNames(t *testing.T) {
	now := time.Date(2026, 3, 1, 16, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	todo, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	doing, _ := domain.NewColumn("c2", p.ID, "In Progress", 1, 0, now)
	done, _ := domain.NewColumn("c3", p.ID, "Done", 2, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  done.ID,
		Position:  0,
		Title:     "Done Task",
		Priority:  domain.PriorityMedium,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{todo, doing, done}, []domain.Task{task})
	svc.changeEvents[p.ID] = []domain.ChangeEvent{
		{
			ID:         3,
			ProjectID:  p.ID,
			WorkItemID: task.ID,
			Operation:  domain.ChangeOperationMove,
			ActorID:    "agent-sync",
			ActorType:  domain.ActorTypeAgent,
			Metadata: map[string]string{
				"from_column_id": todo.ID,
				"from_position":  "1",
				"to_column_id":   done.ID,
				"to_position":    "0",
				"title":          task.Title,
			},
			OccurredAt: now.Add(time.Minute),
		},
	}
	m := loadReadyModel(t, NewModel(svc))
	if !m.setPanelFocusIndex(len(m.columns), false) {
		t.Fatal("expected notices panel focus to be available")
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeActivityEventInfo {
		t.Fatalf("expected activity event info modal, got %v", m.mode)
	}
	detail := stripANSI(m.renderModeOverlay(lipgloss.Color("62"), lipgloss.Color("241"), lipgloss.Color("239"), lipgloss.NewStyle(), 96))
	if !strings.Contains(detail, "column: To Do -> Done") {
		t.Fatalf("expected human-readable move columns in metadata, got %q", detail)
	}
	if strings.Contains(detail, "from_column_id") || strings.Contains(detail, "to_column_id") {
		t.Fatalf("expected raw column-id keys hidden from metadata block, got %q", detail)
	}
	if strings.Contains(detail, "to_position") || strings.Contains(detail, "from_position") {
		t.Fatalf("expected raw position keys hidden from metadata block, got %q", detail)
	}
}

// TestModelDisplayActivityOwnerNormalizesActorLabels verifies notices owner labels use canonical actor type and friendly user identity.
func TestModelDisplayActivityOwnerNormalizesActorLabels(t *testing.T) {
	m := Model{identityDisplayName: "EVAN"}

	actorType, owner := m.displayActivityOwner(activityEntry{
		ActorType: domain.ActorTypeUser,
		ActorID:   "tillsyn-user",
	})
	if actorType != domain.ActorTypeUser {
		t.Fatalf("expected user actor type, got %q", actorType)
	}
	if owner != "EVAN" {
		t.Fatalf("expected local identity display name, got %q", owner)
	}

	actorType, owner = m.displayActivityOwner(activityEntry{
		ActorType: domain.ActorType("AGENT"),
		ActorID:   "agent-lane-2",
		ActorName: "Lane Agent",
	})
	if actorType != domain.ActorTypeAgent {
		t.Fatalf("expected normalized agent actor type, got %q", actorType)
	}
	if owner != "Lane Agent" {
		t.Fatalf("expected actor_name precedence, got %q", owner)
	}

	actorType, owner = m.displayActivityOwner(activityEntry{
		ActorType: domain.ActorType("AGENT"),
		ActorID:   "",
	})
	if actorType != domain.ActorTypeAgent {
		t.Fatalf("expected normalized agent actor type, got %q", actorType)
	}
	if owner != "unknown" {
		t.Fatalf("expected unknown owner fallback, got %q", owner)
	}
}

// TestModelActivityEventTargetDetailsFallbackLabels verifies non-task activity targets render human-facing node/path labels.
func TestModelActivityEventTargetDetailsFallbackLabels(t *testing.T) {
	now := time.Date(2026, 3, 1, 16, 30, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	m := Model{
		projects:        []domain.Project{project},
		selectedProject: 0,
	}

	node, path := m.activityEventTargetDetails(activityEntry{Target: "  queued-action  "})
	if node != "queued-action" {
		t.Fatalf("expected trimmed target node label, got %q", node)
	}
	if path != "Inbox -> queued-action" {
		t.Fatalf("expected project-scoped target path, got %q", path)
	}

	node, path = m.activityEventTargetDetails(activityEntry{Target: ""})
	if node != "-" {
		t.Fatalf("expected fallback node label for blank target, got %q", node)
	}
	if path != "Inbox" {
		t.Fatalf("expected project-only path for blank target, got %q", path)
	}

	node, path = (Model{}).activityEventTargetDetails(activityEntry{Target: ""})
	if node != "-" || path != "-" {
		t.Fatalf("expected root fallback labels without project context, got node=%q path=%q", node, path)
	}
}

// TestModelActivityEventInfoPathCollapsesMiddleSegments verifies activity-event path rendering preserves root + focused node in narrow overlays.
func TestModelActivityEventInfoPathCollapsesMiddleSegments(t *testing.T) {
	now := time.Date(2026, 3, 1, 17, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Project Atlas", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	branch, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-branch",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "Branch Foundation",
		Priority:  domain.PriorityMedium,
	}, now)
	phase, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-phase",
		ProjectID: project.ID,
		ParentID:  branch.ID,
		ColumnID:  column.ID,
		Position:  1,
		Title:     "Phase Delivery",
		Priority:  domain.PriorityMedium,
	}, now)
	leaf, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-leaf",
		ProjectID: project.ID,
		ParentID:  phase.ID,
		ColumnID:  column.ID,
		Position:  2,
		Title:     "Focused Leaf",
		Priority:  domain.PriorityMedium,
	}, now)

	m := Model{
		mode:            modeActivityEventInfo,
		projects:        []domain.Project{project},
		selectedProject: 0,
		tasks:           []domain.Task{branch, phase, leaf},
		activityInfoItem: activityEntry{
			WorkItemID: leaf.ID,
			Summary:    "updated node metadata",
			Operation:  domain.ChangeOperationUpdate,
			At:         now,
		},
	}

	rendered := stripANSI(m.renderModeOverlay(
		lipgloss.Color("62"),
		lipgloss.Color("241"),
		lipgloss.Color("239"),
		lipgloss.NewStyle(),
		54,
	))
	if !strings.Contains(rendered, "path: Project Atlas -> ... -> Focused Leaf") {
		t.Fatalf("expected collapsed activity path in narrow overlay, got\n%s", rendered)
	}
}

// TestModelFormatActivityMetadataFriendlyFallbacks verifies metadata rendering keeps useful labels while hiding raw id noise.
func TestModelFormatActivityMetadataFriendlyFallbacks(t *testing.T) {
	m := Model{
		columns: []domain.Column{
			{ID: "c1", Name: "To Do"},
			{ID: "c2", Name: "In Progress"},
		},
	}
	entry := activityEntry{
		Metadata: map[string]string{
			"column_id":      "c2",
			"position":       "0",
			"changed_fields": "title, due_at",
			"from_state":     "todo",
			"to_state":       "archived",
			"notes":          "manual correction",
			"work_item_id":   "task-123",
		},
	}

	lines := m.formatActivityMetadata(entry)
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "column: In Progress") {
		t.Fatalf("expected human-readable column label, got %q", joined)
	}
	if !strings.Contains(joined, "changed fields: title, due_at") {
		t.Fatalf("expected changed-fields summary, got %q", joined)
	}
	if !strings.Contains(joined, "state: To Do -> Archived") {
		t.Fatalf("expected lifecycle-state summary, got %q", joined)
	}
	if !strings.Contains(joined, "notes: manual correction") {
		t.Fatalf("expected non-id metadata key rendered, got %q", joined)
	}
	if strings.Contains(joined, "work_item_id") || strings.Contains(joined, "position") {
		t.Fatalf("expected raw id/position keys filtered, got %q", joined)
	}
}

// TestModelRecentActivityPanelRefreshesFromPersistedEvents verifies notices-panel activity follows persisted change events on auto-refresh.
func TestModelRecentActivityPanelRefreshesFromPersistedEvents(t *testing.T) {
	now := time.Date(2026, 2, 28, 13, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})
	svc.changeEvents[p.ID] = []domain.ChangeEvent{
		{
			ID:         1,
			ProjectID:  p.ID,
			WorkItemID: task.ID,
			Operation:  domain.ChangeOperationCreate,
			Metadata:   map[string]string{"title": task.Title},
			OccurredAt: now.Add(time.Minute),
		},
	}
	m := loadReadyModel(t, NewModel(svc))
	m.autoRefreshInterval = time.Second
	if got := m.activityLog[len(m.activityLog)-1].Summary; got != "create task" {
		t.Fatalf("expected initial recent activity from persisted event, got %q", got)
	}

	svc.changeEvents[p.ID] = append([]domain.ChangeEvent{
		{
			ID:         2,
			ProjectID:  p.ID,
			WorkItemID: task.ID,
			Operation:  domain.ChangeOperationUpdate,
			Metadata:   map[string]string{"title": task.Title},
			OccurredAt: now.Add(2 * time.Minute),
		},
	}, svc.changeEvents[p.ID]...)

	m, cmd := applyAutoRefreshTickMsg(t, m)
	if !m.autoRefreshInFlight {
		t.Fatal("expected auto-refresh load to start in board mode")
	}
	m = applyAutoRefreshLoadResult(t, m, cmd)
	if got := m.activityLog[len(m.activityLog)-1].Summary; got != "update task" {
		t.Fatalf("expected newest persisted event in recent activity after refresh, got %q", got)
	}
	panel := stripANSI(m.renderOverviewPanel(p, lipgloss.Color("62"), lipgloss.Color("241"), lipgloss.Color("239"), 80, 28, 0, 0, 0, nil, false))
	if !strings.Contains(panel, "user|unknown update task") {
		t.Fatalf("expected notices panel to show refreshed update activity, got %q", panel)
	}
}

// TestModelGroupingByPriority verifies behavior for the covered scenario.
func TestModelGroupingByPriority(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 1, now)
	tLow, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Low first by position",
		Priority:  domain.PriorityLow,
	}, now)
	tHigh, _ := domain.NewTask(domain.TaskInput{
		ID:        "t2",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  1,
		Title:     "High later by position",
		Priority:  domain.PriorityHigh,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{tLow, tHigh})
	m := loadReadyModel(t, NewModel(svc, WithBoardConfig(BoardConfig{
		ShowWIPWarnings: true,
		GroupBy:         "priority",
	})))
	colTasks := m.tasksForColumn(c.ID)
	if len(colTasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(colTasks))
	}
	if colTasks[0].Priority != domain.PriorityHigh {
		t.Fatalf("expected high-priority task first under priority grouping, got %#v", colTasks)
	}
	if got := m.groupLabelForTask(colTasks[0]); got != "Priority: High" {
		t.Fatalf("unexpected group label %q", got)
	}
}

// TestWithKeyConfigOverrides verifies behavior for the covered scenario.
func TestWithKeyConfigOverrides(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityLow,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})
	m := loadReadyModel(t, NewModel(svc, WithKeyConfig(KeyConfig{
		CommandPalette: ";",
		QuickActions:   ",",
		MultiSelect:    "x",
		ActivityLog:    "v",
		Undo:           "u",
		Redo:           "U",
	})))

	m = applyMsg(t, m, keyRune('x'))
	if len(m.selectedTaskIDs) != 1 {
		t.Fatalf("expected selection via configured key, got %d", len(m.selectedTaskIDs))
	}
	m = applyMsg(t, m, keyRune('v'))
	if m.mode != modeActivityLog {
		t.Fatalf("expected configured activity-log key to open modal, got %v", m.mode)
	}
}

// TestModelSelectionHelpers verifies selection helper behavior for the covered scenario.
func TestModelSelectionHelpers(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	c1, _ := domain.NewColumn("c1", "p1", "To Do", 0, 0, now)
	c2, _ := domain.NewColumn("c2", "p1", "Doing", 1, 0, now)
	t1, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: "p1",
		ColumnID:  c1.ID,
		Position:  0,
		Title:     "One",
		Priority:  domain.PriorityLow,
	}, now)
	t2, _ := domain.NewTask(domain.TaskInput{
		ID:        "t2",
		ProjectID: "p1",
		ColumnID:  c2.ID,
		Position:  0,
		Title:     "Two",
		Priority:  domain.PriorityMedium,
	}, now)
	m := Model{
		columns:         []domain.Column{c1, c2},
		tasks:           []domain.Task{t1, t2},
		selectedTaskIDs: map[string]struct{}{"t1": {}, "ghost": {}},
	}

	m.retainSelectionForLoadedTasks()
	if m.isTaskSelected("ghost") {
		t.Fatalf("expected stale selection removed, got %#v", m.selectedTaskIDs)
	}
	if !m.isTaskSelected("t1") {
		t.Fatalf("expected t1 still selected")
	}

	if selected := m.toggleTaskSelection("t1"); selected {
		t.Fatalf("expected toggle remove for existing selection")
	}
	if selected := m.toggleTaskSelection("t2"); !selected {
		t.Fatalf("expected toggle add for missing selection")
	}
	if got := m.sortedSelectedTaskIDs(); len(got) != 1 || got[0] != "t2" {
		t.Fatalf("expected deterministic selection order with t2 only, got %#v", got)
	}

	if removed := m.unselectTasks([]string{"t2", "missing"}); removed != 1 {
		t.Fatalf("expected removed=1, got %d", removed)
	}
	m.toggleTaskSelection("t1")
	m.toggleTaskSelection("t2")
	if cleared := m.clearSelection(); cleared != 2 {
		t.Fatalf("expected clearSelection to clear 2 entries, got %d", cleared)
	}
}

// TestModelHistoryGuards verifies undo/redo guard behavior for the covered scenario.
func TestModelHistoryGuards(t *testing.T) {
	m := Model{
		selectedTaskIDs: map[string]struct{}{},
	}

	updated, cmd := m.undoLastMutation()
	mOut, ok := updated.(Model)
	if !ok {
		t.Fatalf("expected Model from undo guard, got %T", updated)
	}
	if cmd != nil {
		t.Fatalf("expected nil cmd when undo stack empty")
	}
	if mOut.status != "nothing to undo" {
		t.Fatalf("expected empty-stack undo status, got %q", mOut.status)
	}

	mOut.undoStack = []historyActionSet{
		{
			Label:    "bulk hard delete",
			Undoable: false,
			Steps: []historyStep{
				{Kind: historyStepHardDelete, TaskID: "t1"},
			},
		},
	}
	updated, _ = mOut.undoLastMutation()
	mOut, ok = updated.(Model)
	if !ok {
		t.Fatalf("expected Model from non-undoable guard, got %T", updated)
	}
	if mOut.status != "last action cannot be undone" {
		t.Fatalf("expected non-undoable guard status, got %q", mOut.status)
	}
	if len(mOut.undoStack) != 0 {
		t.Fatalf("expected undo stack popped for non-undoable action, got %d", len(mOut.undoStack))
	}
	if len(mOut.activityLog) == 0 || !strings.Contains(mOut.activityLog[len(mOut.activityLog)-1].Summary, "undo") {
		t.Fatalf("expected activity log entry for undo-unavailable path, got %#v", mOut.activityLog)
	}

	mOut.redoStack = nil
	updated, cmd = mOut.redoLastMutation()
	mOut, ok = updated.(Model)
	if !ok {
		t.Fatalf("expected Model from redo guard, got %T", updated)
	}
	if cmd != nil {
		t.Fatalf("expected nil cmd when redo stack empty")
	}
	if mOut.status != "nothing to redo" {
		t.Fatalf("expected empty-stack redo status, got %q", mOut.status)
	}
}

// TestModelExecuteHistorySetHardDeleteUndo verifies hard-delete undo guard behavior.
func TestModelExecuteHistorySetHardDeleteUndo(t *testing.T) {
	m := Model{}
	msg := m.executeHistorySet(historyActionSet{
		Label: "hard delete selection",
		Steps: []historyStep{
			{Kind: historyStepHardDelete, TaskID: "t1"},
		},
	}, true)()
	action, ok := msg.(actionMsg)
	if !ok {
		t.Fatalf("expected actionMsg from executeHistorySet, got %T", msg)
	}
	if !strings.Contains(action.status, "hard delete cannot be restored") {
		t.Fatalf("expected hard-delete undo guard message, got %q", action.status)
	}
}

// TestModelMoveStepBuilderAndGroupingHelpers verifies helper behavior for ordering/grouping.
func TestModelMoveStepBuilderAndGroupingHelpers(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	c1, _ := domain.NewColumn("c1", "p1", "To Do", 0, 0, now)
	c2, _ := domain.NewColumn("c2", "p1", "Doing", 1, 0, now)
	t1, _ := domain.NewTask(domain.TaskInput{
		ID:             "t1",
		ProjectID:      "p1",
		ColumnID:       c1.ID,
		Position:       0,
		Title:          "One",
		Priority:       domain.PriorityHigh,
		LifecycleState: domain.StateTodo,
	}, now)
	t2, _ := domain.NewTask(domain.TaskInput{
		ID:             "t2",
		ProjectID:      "p1",
		ColumnID:       c1.ID,
		Position:       1,
		Title:          "Two",
		Priority:       domain.PriorityLow,
		LifecycleState: domain.StateDone,
	}, now)
	t3, _ := domain.NewTask(domain.TaskInput{
		ID:             "t3",
		ProjectID:      "p1",
		ColumnID:       c2.ID,
		Position:       0,
		Title:          "Three",
		Priority:       domain.PriorityMedium,
		LifecycleState: domain.StateProgress,
	}, now)
	m := Model{
		columns: []domain.Column{c1, c2},
		tasks:   []domain.Task{t1, t2, t3},
	}

	if got := m.buildMoveSteps([]string{"t1", "t2"}, 0); got != nil {
		t.Fatalf("expected nil move steps for delta=0, got %#v", got)
	}
	steps := m.buildMoveSteps([]string{"t2", "t1"}, 1)
	if len(steps) != 2 {
		t.Fatalf("expected 2 move steps, got %#v", steps)
	}
	if steps[0].TaskID != "t1" || steps[1].TaskID != "t2" {
		t.Fatalf("expected deterministic board ordering, got %#v", steps)
	}
	if steps[0].ToPosition != 1 || steps[1].ToPosition != 2 {
		t.Fatalf("expected target column append ordering, got %#v", steps)
	}

	if got := normalizeBoardGroupBy(" PRIORITY "); got != "priority" {
		t.Fatalf("unexpected normalizeBoardGroupBy priority result: %q", got)
	}
	if got := normalizeBoardGroupBy("state"); got != "state" {
		t.Fatalf("unexpected normalizeBoardGroupBy state result: %q", got)
	}
	if got := normalizeBoardGroupBy("weird"); got != "none" {
		t.Fatalf("unexpected normalizeBoardGroupBy fallback result: %q", got)
	}

	m.boardGroupBy = "priority"
	if got := m.groupLabelForTask(t1); got != "Priority: High" {
		t.Fatalf("unexpected priority group label: %q", got)
	}
	m.boardGroupBy = "state"
	if got := m.groupLabelForTask(t3); got != "State: In Progress" {
		t.Fatalf("unexpected state group label: %q", got)
	}
	if rank := taskGroupRank(t1, "priority"); rank != 0 {
		t.Fatalf("expected high priority rank=0, got %d", rank)
	}
	if rank := taskGroupRank(t2, "state"); rank != 2 {
		t.Fatalf("expected done state rank=2, got %d", rank)
	}
}

// TestModelActivityAndHistoryBounds verifies capped retention and transition helpers.
func TestModelActivityAndHistoryBounds(t *testing.T) {
	m := Model{}
	for i := 0; i < 205; i++ {
		m.appendActivity(activityEntry{
			At:      time.Date(2026, 2, 21, 12, 0, i%60, 0, time.UTC),
			Summary: "event",
		})
	}
	if len(m.activityLog) != 200 {
		t.Fatalf("expected bounded activity log length=200, got %d", len(m.activityLog))
	}
	if m.activityLog[0].Target != "-" {
		t.Fatalf("expected default activity target fallback, got %#v", m.activityLog[0])
	}

	m.redoStack = []historyActionSet{{ID: 99}}
	for i := 0; i < 105; i++ {
		m.pushUndoHistory(historyActionSet{
			Label:    "step",
			Undoable: true,
			Steps:    []historyStep{{Kind: historyStepMove, TaskID: "t1"}},
		})
	}
	if len(m.undoStack) != 100 {
		t.Fatalf("expected bounded undo stack length=100, got %d", len(m.undoStack))
	}
	if len(m.redoStack) != 0 {
		t.Fatalf("expected redo stack cleared after new push, got %d", len(m.redoStack))
	}

	last := m.undoStack[len(m.undoStack)-1]
	m.applyUndoTransition(last)
	if len(m.redoStack) != 1 {
		t.Fatalf("expected redo stack append after applyUndoTransition, got %d", len(m.redoStack))
	}
	m.applyRedoTransition(last)
	if len(m.undoStack) == 0 {
		t.Fatalf("expected undo stack append after applyRedoTransition")
	}
}

// TestModelModeLabelPromptAndActivityTimestamp verifies mode helper rendering.
func TestModelModeLabelPromptAndActivityTimestamp(t *testing.T) {
	m := Model{mode: modeActivityLog}
	if got := m.modeLabel(); got != "activity" {
		t.Fatalf("unexpected mode label %q", got)
	}
	if got := m.modePrompt(); !strings.Contains(got, "activity log") {
		t.Fatalf("unexpected mode prompt %q", got)
	}
	if got := formatActivityTimestamp(time.Time{}); got != "--:--:--" {
		t.Fatalf("unexpected zero timestamp format %q", got)
	}
	old := time.Date(2024, 1, 10, 5, 4, 0, 0, time.Local)
	if got := formatActivityTimestamp(old); !strings.Contains(got, "01-10") {
		t.Fatalf("expected old-date timestamp to include month/day, got %q", got)
	}
	now := time.Now()
	if got := formatActivityTimestamp(now); !strings.Contains(got, ":") {
		t.Fatalf("expected time-of-day timestamp, got %q", got)
	}
}

// TestResourcePickerHelpers verifies filesystem picker helper behavior.
func TestResourcePickerHelpers(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "README.md"), []byte("ok"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}

	entries, current, err := listResourcePickerEntries(projectDir, projectDir)
	if err != nil {
		t.Fatalf("listResourcePickerEntries root: %v", err)
	}
	if current != projectDir {
		t.Fatalf("expected current dir %q, got %q", projectDir, current)
	}
	if len(entries) < 2 {
		t.Fatalf("expected at least docs+README entries, got %#v", entries)
	}
	if !entries[0].IsDir {
		t.Fatalf("expected directories sorted before files, got %#v", entries)
	}

	// Outside paths should be honored so callers can navigate to parent directories.
	_, current, err = listResourcePickerEntries(projectDir, root)
	if err != nil {
		t.Fatalf("listResourcePickerEntries outside root: %v", err)
	}
	if current != root {
		t.Fatalf("expected outside current dir %q, got %q", root, current)
	}
}

// TestResourceRefHelpers verifies reference normalization and duplicate suppression.
func TestResourceRefHelpers(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "notes.txt")
	if err := os.WriteFile(filePath, []byte("ok"), 0o644); err != nil {
		t.Fatalf("write notes: %v", err)
	}

	ref := buildResourceRef(root, filePath, false)
	if ref.ResourceType != domain.ResourceTypeLocalFile {
		t.Fatalf("expected local_file resource, got %q", ref.ResourceType)
	}
	if ref.PathMode != domain.PathModeRelative {
		t.Fatalf("expected relative path mode under root, got %q", ref.PathMode)
	}
	if ref.BaseAlias != "project_root" {
		t.Fatalf("expected project_root base alias, got %q", ref.BaseAlias)
	}

	refs, added := appendResourceRefIfMissing(nil, ref)
	if !added || len(refs) != 1 {
		t.Fatalf("expected first ref append, got added=%t refs=%#v", added, refs)
	}
	refs, added = appendResourceRefIfMissing(refs, ref)
	if added || len(refs) != 1 {
		t.Fatalf("expected duplicate suppression, got added=%t refs=%#v", added, refs)
	}
}

// TestProjectionAndRollupHelpers verifies subtree projection and summary helper behavior.
func TestProjectionAndRollupHelpers(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	c1, _ := domain.NewColumn("c1", "p1", "To Do", 0, 0, now)
	c2, _ := domain.NewColumn("c2", "p1", "Doing", 1, 0, now)
	phase, _ := domain.NewTask(domain.TaskInput{
		ID:        "phase",
		ProjectID: "p1",
		ColumnID:  c1.ID,
		Position:  0,
		Kind:      domain.WorkKindPhase,
		Title:     "Phase",
		Priority:  domain.PriorityMedium,
	}, now)
	child, _ := domain.NewTask(domain.TaskInput{
		ID:        "child",
		ProjectID: "p1",
		ParentID:  phase.ID,
		ColumnID:  c2.ID,
		Position:  0,
		Title:     "Child",
		Priority:  domain.PriorityLow,
	}, now)
	unrelated, _ := domain.NewTask(domain.TaskInput{
		ID:        "other",
		ProjectID: "p1",
		ColumnID:  c1.ID,
		Position:  1,
		Title:     "Other",
		Priority:  domain.PriorityHigh,
	}, now)

	m := Model{
		columns: []domain.Column{c1, c2},
		tasks:   []domain.Task{phase, child, unrelated},
		dependencyRollup: domain.DependencyRollup{
			TotalItems:                3,
			BlockedItems:              1,
			UnresolvedDependencyEdges: 2,
			DependencyEdges:           4,
		},
	}
	m.projectionRootTaskID = phase.ID
	set := m.projectedTaskSet()
	if len(set) != 1 {
		t.Fatalf("expected projected set with direct children only, got %#v", set)
	}
	if _, ok := set[child.ID]; !ok {
		t.Fatalf("expected child task in projected set, got %#v", set)
	}
	if _, ok := set[unrelated.ID]; ok {
		t.Fatalf("did not expect unrelated task in projected set")
	}

	col1 := m.tasksForColumn(c1.ID)
	if len(col1) != 0 {
		t.Fatalf("expected focused scope column without direct children to be empty, got %#v", col1)
	}
	if breadcrumb := m.projectionBreadcrumb(); breadcrumb != "Phase" {
		t.Fatalf("unexpected projection breadcrumb %q", breadcrumb)
	}
	if summary := m.dependencyRollupSummary(); !strings.Contains(summary, "blocked 1") {
		t.Fatalf("unexpected dependency summary %q", summary)
	}
}

// TestProjectRootLookup verifies strict project-root lookup and non-task browse fallback behavior.
func TestProjectRootLookup(t *testing.T) {
	root := t.TempDir()
	p := domain.Project{ID: "p1", Slug: "inbox", Name: "Inbox"}
	m := Model{
		projects:     []domain.Project{p},
		projectRoots: map[string]string{"inbox": root},
	}
	if got := m.resourcePickerRootForCurrentProject(); got != root {
		t.Fatalf("expected configured project root %q, got %q", root, got)
	}

	m.projectRoots = map[string]string{}
	m.searchRoots = []string{root}
	if got := m.resourcePickerRootForCurrentProject(); got != root {
		t.Fatalf("expected bootstrap/search-root fallback %q when project root missing, got %q", root, got)
	}
	if got := m.resourcePickerBrowseRoot(); got == "" {
		t.Fatal("expected non-empty browse root fallback")
	}
}

// TestLabelInheritanceHelpers verifies label inheritance and picker helper behavior.
func TestLabelInheritanceHelpers(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p := domain.Project{ID: "p1", Slug: "inbox", Name: "Inbox"}
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	phase, _ := domain.NewTask(domain.TaskInput{
		ID:        "phase",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Kind:      domain.WorkKindPhase,
		Position:  0,
		Title:     "Phase",
		Labels:    []string{"PhaseA", "shared"},
		Priority:  domain.PriorityMedium,
	}, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "task",
		ProjectID: p.ID,
		ParentID:  phase.ID,
		ColumnID:  c.ID,
		Position:  1,
		Title:     "Task",
		Labels:    []string{"task-only"},
		Priority:  domain.PriorityLow,
	}, now)
	m := Model{
		projects:            []domain.Project{p},
		columns:             []domain.Column{c},
		tasks:               []domain.Task{phase, task},
		selectedColumn:      0,
		selectedTask:        1,
		allowedLabelGlobal:  []string{"global", "shared"},
		allowedLabelProject: map[string][]string{"inbox": {"project", "shared"}},
	}

	phaseLabels := m.labelsFromPhaseAncestors(task)
	if len(phaseLabels) != 2 || phaseLabels[0] != "phasea" {
		t.Fatalf("unexpected phase ancestor labels %#v", phaseLabels)
	}

	merged := mergeLabelSources(labelInheritanceSources{
		Global:  []string{"Global", "shared"},
		Project: []string{"Project", "shared"},
		Phase:   []string{"PhaseA", "project"},
	})
	if len(merged) != 4 || merged[0] != "global" {
		t.Fatalf("unexpected merged label set %#v", merged)
	}

	m.taskFormParentID = phase.ID
	_ = m.startTaskForm(nil)
	m.taskFormParentID = phase.ID
	items := m.taskFormLabelPickerItems()
	if len(items) == 0 {
		t.Fatalf("expected label picker items from inheritance sources")
	}

	m.formInputs[4].SetValue("alpha")
	m.appendTaskFormLabel("beta")
	m.appendTaskFormLabel("alpha")
	if got := m.formInputs[4].Value(); got != "alpha,beta" {
		t.Fatalf("expected de-duplicated label append, got %q", got)
	}
}

// TestResourcePickerEntrySelectionAndParent verifies picker-selection helper behavior.
func TestResourcePickerEntrySelectionAndParent(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "child"), 0o755); err != nil {
		t.Fatalf("mkdir child: %v", err)
	}
	m := Model{
		resourcePickerRoot:  root,
		resourcePickerDir:   filepath.Join(root, "child"),
		resourcePickerItems: []resourcePickerEntry{{Name: "child", Path: filepath.Join(root, "child"), IsDir: true}},
		resourcePickerIndex: 0,
	}
	entry, ok := m.selectedResourcePickerEntry()
	if !ok || entry.Name != "child" {
		t.Fatalf("expected selected picker entry child, got %#v ok=%t", entry, ok)
	}

	msg := m.openResourcePickerParent()()
	loaded, ok := msg.(resourcePickerLoadedMsg)
	if !ok {
		t.Fatalf("expected resourcePickerLoadedMsg, got %T", msg)
	}
	if loaded.err != nil {
		t.Fatalf("expected parent directory load success, got %v", loaded.err)
	}
	if loaded.current != root {
		t.Fatalf("expected parent load to clamp at root %q, got %q", root, loaded.current)
	}

	m.resourcePickerDir = root
	msg = m.openResourcePickerParent()()
	loaded, ok = msg.(resourcePickerLoadedMsg)
	if !ok {
		t.Fatalf("expected resourcePickerLoadedMsg from root-parent nav, got %T", msg)
	}
	if loaded.err != nil {
		t.Fatalf("expected root-parent directory load success, got %v", loaded.err)
	}
	if loaded.current != filepath.Dir(root) {
		t.Fatalf("expected root parent %q, got %q", filepath.Dir(root), loaded.current)
	}
}

// TestModelResourcePickerAttachFromEdit verifies resource attachment flows from the modal-only edit row.
func TestModelResourcePickerAttachFromEdit(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})

	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "notes.md"), []byte("spec"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	m := loadReadyModel(t, NewModel(svc, WithProjectRoots(map[string]string{"inbox": root})))
	m = applyMsg(t, m, keyRune('e'))
	if m.mode != modeEditTask {
		t.Fatalf("expected edit mode, got %v", m.mode)
	}
	m.formFocus = taskFieldResources
	m.taskFormResourceCursor = 0
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeResourcePicker {
		t.Fatalf("expected resource picker mode from edit-task new-resource row, got %v", m.mode)
	}
	if got := strings.TrimSpace(m.resourcePickerRoot); got != root {
		t.Fatalf("expected resource root %q, got %q", root, got)
	}
	if m.taskFormResourceEditIndex != -1 {
		t.Fatalf("expected new-resource row to keep edit index unset, got %d", m.taskFormResourceEditIndex)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyDown}) // first file after directory entry
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeEditTask {
		t.Fatalf("expected return to edit mode after attach, got %v", m.mode)
	}
	if len(m.taskFormResourceRefs) != 1 {
		t.Fatalf("expected 1 staged resource ref in edit form, got %#v", m.taskFormResourceRefs)
	}
	ref := m.taskFormResourceRefs[0]
	if ref.ResourceType != domain.ResourceTypeLocalFile {
		t.Fatalf("expected local file resource type, got %q", ref.ResourceType)
	}
	if ref.PathMode != domain.PathModeRelative || ref.BaseAlias != "project_root" {
		t.Fatalf("expected project-root relative reference, got %#v", ref)
	}
	if filepath.ToSlash(ref.Location) != "notes.md" {
		t.Fatalf("expected relative location notes.md, got %q", ref.Location)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	updated, ok := svc.taskByID("t1")
	if !ok {
		t.Fatal("expected updated task in fake service")
	}
	if len(updated.Metadata.ResourceRefs) != 1 {
		t.Fatalf("expected 1 attached resource after save, got %#v", updated.Metadata.ResourceRefs)
	}

	cancelModel := loadReadyModel(t, NewModel(svc, WithProjectRoots(map[string]string{"inbox": root})))
	cancelModel = applyMsg(t, cancelModel, keyRune('e'))
	if cancelModel.mode != modeEditTask {
		t.Fatalf("expected edit mode for cancel-path check, got %v", cancelModel.mode)
	}
	cancelModel = applyCmd(t, cancelModel, cancelModel.focusTaskFormField(taskFieldResources))
	if cancelModel.formFocus != taskFieldResources {
		t.Fatalf("expected resources row focus for cancel-path check, got %d", cancelModel.formFocus)
	}
	cancelModel.taskFormResourceCursor = 0
	cancelModel = applyMsg(t, cancelModel, tea.KeyPressMsg{Code: tea.KeyEnter})
	if cancelModel.mode != modeResourcePicker {
		t.Fatalf("expected resource picker mode from reopened edit-task resources section, got %v", cancelModel.mode)
	}
	cancelModel = applyMsg(t, cancelModel, tea.KeyPressMsg{Code: tea.KeyEscape})
	if cancelModel.mode != modeEditTask {
		t.Fatalf("expected esc from resource picker to return to edit mode, got %v", cancelModel.mode)
	}
}

// TestModelResourcePickerFallsBackToBootstrapRoot verifies task attachment uses the bootstrap/search root when no project root is configured.
func TestModelResourcePickerFallsBackToBootstrapRoot(t *testing.T) {
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})
	root := t.TempDir()
	m := loadReadyModel(t, NewModel(
		svc,
		WithSearchRoots([]string{root}),
	))

	m = applyMsg(t, m, keyRune('i'))
	m = applyMsg(t, m, keyRune('e'))
	if m.mode != modeEditTask {
		t.Fatalf("expected edit mode, got %v", m.mode)
	}
	m.formFocus = taskFieldResources
	m = applyMsg(t, m, keyRune('e'))
	if m.mode != modeResourcePicker {
		t.Fatalf("expected resource picker when bootstrap root fallback is available, got %v", m.mode)
	}
	if got := strings.TrimSpace(m.resourcePickerRoot); got != root {
		t.Fatalf("expected bootstrap/search-root fallback %q, got %q", root, got)
	}
}

// TestModelEditTaskCommentsRowOpensThread verifies edit-mode comments management stays accessible from the form.
func TestModelEditTaskCommentsRowOpensThread(t *testing.T) {
	now := time.Date(2026, 3, 13, 11, 15, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{project}, []domain.Column{column}, []domain.Task{task})))

	m = applyMsg(t, m, keyRune('e'))
	if m.mode != modeEditTask {
		t.Fatalf("expected edit mode, got %v", m.mode)
	}

	beforeTitle := m.formInputs[taskFieldTitle].Value()
	m = applyMsg(t, m, keyRune('c'))
	if m.mode != modeEditTask {
		t.Fatalf("expected lowercase c to remain text input in edit mode, got %v", m.mode)
	}
	if got := m.formInputs[taskFieldTitle].Value(); got != beforeTitle+"c" {
		t.Fatalf("expected lowercase c to type into title, got %q", got)
	}

	m = applyMsg(t, m, keyRune('e'))
	if m.mode != modeEditTask {
		t.Fatalf("expected lowercase e to remain text input in edit mode, got %v", m.mode)
	}
	if got := m.formInputs[taskFieldTitle].Value(); got != beforeTitle+"ce" {
		t.Fatalf("expected lowercase e to type into title, got %q", got)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: 'E', Text: "E", Mod: tea.ModShift})
	if m.mode != modeEditTask {
		t.Fatalf("expected uppercase E to remain text input in edit mode, got %v", m.mode)
	}
	if got := m.formInputs[taskFieldTitle].Value(); got != beforeTitle+"ceE" {
		t.Fatalf("expected uppercase E to type into title, got %q", got)
	}

	m = applyCmd(t, m, m.focusTaskFormField(taskFieldComments))
	if m.formFocus != taskFieldComments {
		t.Fatalf("expected comments row focus, got %d", m.formFocus)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeThread {
		t.Fatalf("expected enter on comments row to open thread/comments, got %v", m.mode)
	}
	if m.threadPanelFocus != threadPanelComments {
		t.Fatalf("expected comments row to open thread on comments panel, got %d", m.threadPanelFocus)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.mode != modeEditTask {
		t.Fatalf("expected esc from thread/comments to return to edit mode, got %v", m.mode)
	}
}

// TestModelTaskInfoAndEditHideLabelInheritanceBlocks verifies info/edit hide inherited label blocks while keeping picker flows available.
func TestModelTaskInfoAndEditHideLabelInheritanceBlocks(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	phase, _ := domain.NewTask(domain.TaskInput{
		ID:        "phase-1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Kind:      domain.WorkKindPhase,
		Title:     "Phase",
		Priority:  domain.PriorityMedium,
		Labels:    []string{"phase-label"},
	}, now)
	child, _ := domain.NewTask(domain.TaskInput{
		ID:        "task-1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  1,
		ParentID:  phase.ID,
		Kind:      domain.WorkKindTask,
		Title:     "Child Task",
		Priority:  domain.PriorityMedium,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{phase, child})
	m := loadReadyModel(t, NewModel(svc, WithLabelConfig(LabelConfig{
		Global:   []string{"global-label"},
		Projects: map[string][]string{"inbox": []string{"project-label"}},
	})))

	m = applyMsg(t, m, keyRune('j')) // select child task
	m = applyMsg(t, m, keyRune('i'))
	info := strings.ToLower(stripANSI(m.renderModeOverlay(
		lipgloss.Color("62"),
		lipgloss.Color("241"),
		lipgloss.Color("239"),
		lipgloss.NewStyle(),
		96,
	)))
	if strings.Contains(info, "effective labels") || strings.Contains(info, "inherited labels") {
		t.Fatalf("expected task-info output to hide inherited/effective labels block, got %q", info)
	}

	m = applyMsg(t, m, keyRune('e'))
	if m.mode != modeEditTask {
		t.Fatalf("expected edit task mode, got %v", m.mode)
	}
	m.formFocus = taskFieldLabels
	editBody, _ := m.taskFormBodyLines(80, lipgloss.NewStyle().Foreground(lipgloss.Color("241")), lipgloss.Color("62"))
	edit := strings.ToLower(stripANSI(strings.Join(editBody, "\n")))
	if strings.Contains(edit, "effective labels") || strings.Contains(edit, "inherited labels") {
		t.Fatalf("expected edit-task output to hide inherited/effective labels block, got %q", edit)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeLabelPicker {
		t.Fatalf("expected enter on labels to open label picker, got %v", m.mode)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.mode != modeEditTask {
		t.Fatalf("expected esc to return to edit-task from label picker, got %v", m.mode)
	}

	_ = m.focusTaskFormField(taskFieldLabels)
	m = applyMsg(t, m, keyRune('e'))
	if m.mode != modeLabelPicker {
		t.Fatalf("expected e on labels to open label picker, got %v", m.mode)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.mode != modeEditTask {
		t.Fatalf("expected esc to return to edit-task from label picker opened by e, got %v", m.mode)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeLabelPicker {
		t.Fatalf("expected enter on labels to open label picker, got %v", m.mode)
	}
	m = applyMsg(t, m, keyRune('j'))
	m = applyMsg(t, m, keyRune('j'))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeEditTask {
		t.Fatalf("expected return to edit-task after picker choose, got %v", m.mode)
	}
	if got := m.formInputs[taskFieldLabels].Value(); !strings.Contains(got, "phase-label") {
		t.Fatalf("expected appended phase label in form value, got %q", got)
	}
}

// TestModelProjectionFocusBreadcrumbMode verifies subtree focus mode and breadcrumb switching.
func TestModelProjectionFocusBreadcrumbMode(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	parent, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-parent",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Parent",
		Priority:  domain.PriorityMedium,
	}, now)
	child, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-child",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  1,
		ParentID:  parent.ID,
		Title:     "Child",
		Priority:  domain.PriorityMedium,
	}, now)
	grandchild, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-grandchild",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  2,
		ParentID:  child.ID,
		Title:     "Grandchild",
		Priority:  domain.PriorityLow,
	}, now)
	other, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-other",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  3,
		Title:     "Unrelated",
		Priority:  domain.PriorityLow,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{parent, child, grandchild, other})
	m := loadReadyModel(t, NewModel(svc))

	m.focusTaskByID(parent.ID)
	m = applyMsg(t, m, keyRune('f')) // enter parent scope (child becomes selected)
	m = applyMsg(t, m, keyRune('f')) // enter child scope
	if m.projectionRootTaskID != child.ID {
		t.Fatalf("expected projection root %q, got %q", child.ID, m.projectionRootTaskID)
	}
	tasks := m.currentColumnTasks()
	if len(tasks) != 1 || tasks[0].ID != grandchild.ID {
		t.Fatalf("expected projected child scope to show direct children only, got %#v", tasks)
	}
	if got := m.projectionBreadcrumb(); got != "Parent / Child" {
		t.Fatalf("expected breadcrumb Parent / Child, got %q", got)
	}
	focusedView := stripANSI(fmt.Sprint(m.View().Content))
	if !strings.Contains(focusedView, "path: Inbox -> Parent -> Child") {
		t.Fatalf("expected explicit focus path line in view, got\n%s", focusedView)
	}
	if m.modePrompt() != "" {
		t.Fatalf("expected normal-mode prompt while projected, got %q", m.modePrompt())
	}

	m = applyMsg(t, m, keyRune('F'))
	if m.projectionRootTaskID != "" {
		t.Fatalf("expected focus cleared after F, got %q", m.projectionRootTaskID)
	}
	if len(m.currentColumnTasks()) != 2 {
		t.Fatalf("expected full board tasks after clearing focus, got %d", len(m.currentColumnTasks()))
	}
	fullBoardView := stripANSI(fmt.Sprint(m.View().Content))
	if !strings.Contains(fullBoardView, "path: Inbox") {
		t.Fatalf("expected project path while not focused, got\n%s", fullBoardView)
	}
}

// TestModelBoardPathLineCollapsesMiddleSegments verifies board header path uses middle ellipsis under constrained width.
func TestModelBoardPathLineCollapsesMiddleSegments(t *testing.T) {
	now := time.Date(2026, 2, 21, 13, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Roadmap Hub", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	branch, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-branch",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "Branch Architecture",
		Priority:  domain.PriorityMedium,
	}, now)
	phase, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-phase",
		ProjectID: project.ID,
		ParentID:  branch.ID,
		ColumnID:  column.ID,
		Position:  1,
		Title:     "Phase Discovery",
		Priority:  domain.PriorityMedium,
	}, now)
	leaf, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-leaf",
		ProjectID: project.ID,
		ParentID:  phase.ID,
		ColumnID:  column.ID,
		Position:  2,
		Title:     "Focused Item",
		Priority:  domain.PriorityLow,
	}, now)

	m := loadReadyModel(t, NewModel(newFakeService(
		[]domain.Project{project},
		[]domain.Column{column},
		[]domain.Task{branch, phase, leaf},
	)))
	m.projectionRootTaskID = leaf.ID
	m = applyMsg(t, m, tea.WindowSizeMsg{Width: 44, Height: 24})

	rendered := stripANSI(fmt.Sprint(m.View().Content))
	if !strings.Contains(rendered, "│ TILLSYN │  path: R -> ... -> Focused It") {
		t.Fatalf("expected collapsed board path in narrow layout, got\n%s", rendered)
	}
}

// TestModelFocusSubtreeRendersBoardForHierarchyLevels verifies branch/phase/nested-phase focus keeps project-board columns visible.
func TestModelFocusSubtreeRendersBoardForHierarchyLevels(t *testing.T) {
	now := time.Date(2026, 2, 24, 9, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Roadmap", "", now)
	todo, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	progress, _ := domain.NewColumn("c2", p.ID, "In Progress", 1, 0, now)
	done, _ := domain.NewColumn("c3", p.ID, "Done", 2, 0, now)

	branch, _ := domain.NewTask(domain.TaskInput{
		ID:        "w-branch",
		ProjectID: p.ID,
		ColumnID:  todo.ID,
		Position:  0,
		Title:     "Branch",
		Priority:  domain.PriorityMedium,
		Kind:      domain.WorkKind("branch"),
		Scope:     domain.KindAppliesToBranch,
	}, now)
	phase, _ := domain.NewTask(domain.TaskInput{
		ID:        "w-phase",
		ProjectID: p.ID,
		ParentID:  branch.ID,
		ColumnID:  progress.ID,
		Position:  0,
		Title:     "Phase",
		Priority:  domain.PriorityMedium,
		Kind:      domain.WorkKindPhase,
		Scope:     domain.KindAppliesToPhase,
	}, now)
	nestedPhase, _ := domain.NewTask(domain.TaskInput{
		ID:        "w-nested-phase",
		ProjectID: p.ID,
		ParentID:  phase.ID,
		ColumnID:  done.ID,
		Position:  0,
		Title:     "Nested Phase",
		Priority:  domain.PriorityMedium,
		Kind:      domain.WorkKindPhase,
		Scope:     domain.KindAppliesToPhase,
	}, now)
	leafTask, _ := domain.NewTask(domain.TaskInput{
		ID:        "w-task",
		ProjectID: p.ID,
		ParentID:  nestedPhase.ID,
		ColumnID:  done.ID,
		Position:  1,
		Title:     "Task",
		Priority:  domain.PriorityLow,
		Kind:      domain.WorkKindTask,
		Scope:     domain.KindAppliesToTask,
	}, now)
	unrelated, _ := domain.NewTask(domain.TaskInput{
		ID:        "w-other",
		ProjectID: p.ID,
		ColumnID:  todo.ID,
		Position:  1,
		Title:     "Other",
		Priority:  domain.PriorityLow,
		Kind:      domain.WorkKindTask,
		Scope:     domain.KindAppliesToTask,
	}, now)

	svc := newFakeService(
		[]domain.Project{p},
		[]domain.Column{todo, progress, done},
		[]domain.Task{branch, phase, nestedPhase, leafTask, unrelated},
	)
	m := loadReadyModel(t, NewModel(svc))
	visibleIDs := func(in Model) []string {
		out := make([]string, 0)
		for _, column := range in.columns {
			for _, task := range in.boardTasksForColumn(column.ID) {
				out = append(out, task.ID)
			}
		}
		return out
	}
	assertVisible := func(in Model, expected []string) {
		t.Helper()
		got := strings.Join(visibleIDs(in), ",")
		want := strings.Join(expected, ",")
		if got != want {
			t.Fatalf("unexpected focused board ids\nwant: %s\ngot:  %s", want, got)
		}
		rendered := stripANSI(fmt.Sprint(in.View().Content))
		if !strings.Contains(rendered, "To Do (") || !strings.Contains(rendered, "In Progress (") || !strings.Contains(rendered, "Done (") {
			t.Fatalf("expected project-board columns while focused, got\n%s", rendered)
		}
	}

	assertVisible(m, []string{branch.ID, unrelated.ID})

	m.focusTaskByID(branch.ID)
	m = applyMsg(t, m, keyRune('f'))
	assertVisible(m, []string{phase.ID})

	m = applyMsg(t, m, keyRune('f'))
	assertVisible(m, []string{nestedPhase.ID})

	m = applyMsg(t, m, keyRune('f'))
	assertVisible(m, []string{leafTask.ID})
}

// TestModelFocusTaskScopeShowsSubtasks verifies task-focused scope rendering includes direct subtasks.
func TestModelFocusTaskScopeShowsSubtasks(t *testing.T) {
	now := time.Date(2026, 2, 25, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	todo, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	done, _ := domain.NewColumn("c2", p.ID, "Done", 1, 0, now)
	parent, _ := domain.NewTask(domain.TaskInput{
		ID:        "task-root",
		ProjectID: p.ID,
		ColumnID:  todo.ID,
		Position:  0,
		Title:     "Parent Task",
		Priority:  domain.PriorityMedium,
		Kind:      domain.WorkKindTask,
		Scope:     domain.KindAppliesToTask,
	}, now)
	subA, _ := domain.NewTask(domain.TaskInput{
		ID:        "sub-a",
		ProjectID: p.ID,
		ParentID:  parent.ID,
		ColumnID:  todo.ID,
		Position:  1,
		Title:     "Subtask A",
		Priority:  domain.PriorityLow,
		Kind:      domain.WorkKindSubtask,
		Scope:     domain.KindAppliesToSubtask,
	}, now)
	subB, _ := domain.NewTask(domain.TaskInput{
		ID:        "sub-b",
		ProjectID: p.ID,
		ParentID:  parent.ID,
		ColumnID:  done.ID,
		Position:  0,
		Title:     "Subtask B",
		Priority:  domain.PriorityLow,
		Kind:      domain.WorkKindSubtask,
		Scope:     domain.KindAppliesToSubtask,
	}, now)
	other, _ := domain.NewTask(domain.TaskInput{
		ID:        "task-other",
		ProjectID: p.ID,
		ColumnID:  todo.ID,
		Position:  2,
		Title:     "Other Top Task",
		Priority:  domain.PriorityLow,
		Kind:      domain.WorkKindTask,
		Scope:     domain.KindAppliesToTask,
	}, now)

	m := loadReadyModel(t, NewModel(newFakeService(
		[]domain.Project{p},
		[]domain.Column{todo, done},
		[]domain.Task{parent, subA, subB, other},
	)))

	projectScope := stripANSI(fmt.Sprint(m.View().Content))
	if strings.Contains(projectScope, "Subtask A") || strings.Contains(projectScope, "Subtask B") {
		t.Fatalf("expected project scope board to hide subtasks, got\n%s", projectScope)
	}

	m.focusTaskByID(parent.ID)
	m = applyMsg(t, m, keyRune('f'))
	visible := []string{}
	for _, column := range m.columns {
		for _, task := range m.boardTasksForColumn(column.ID) {
			visible = append(visible, task.ID)
		}
	}
	got := strings.Join(visible, ",")
	if got != "sub-a,sub-b" {
		t.Fatalf("expected task-focused scope to show direct subtasks, got %s", got)
	}
	focused := stripANSI(fmt.Sprint(m.View().Content))
	if !strings.Contains(focused, "path: Inbox -> Parent Task") {
		t.Fatalf("expected focused task path in board view, got\n%s", focused)
	}
}

// TestModelFocusScopeShowsDirectSubtaskChildrenForLegacyParentKinds verifies focus works when direct children are subtasks.
func TestModelFocusScopeShowsDirectSubtaskChildrenForLegacyParentKinds(t *testing.T) {
	now := time.Date(2026, 2, 25, 13, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	todo, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	branch, _ := domain.NewTask(domain.TaskInput{
		ID:        "b1",
		ProjectID: p.ID,
		ColumnID:  todo.ID,
		Position:  0,
		Title:     "Branch Parent",
		Priority:  domain.PriorityMedium,
		Kind:      domain.WorkKind("branch"),
		Scope:     domain.KindAppliesToBranch,
	}, now)
	legacySubtask, _ := domain.NewTask(domain.TaskInput{
		ID:        "st1",
		ProjectID: p.ID,
		ParentID:  branch.ID,
		ColumnID:  todo.ID,
		Position:  1,
		Title:     "Legacy Subtask Child",
		Priority:  domain.PriorityLow,
		Kind:      domain.WorkKindSubtask,
		Scope:     domain.KindAppliesToSubtask,
	}, now)

	m := loadReadyModel(t, NewModel(newFakeService(
		[]domain.Project{p},
		[]domain.Column{todo},
		[]domain.Task{branch, legacySubtask},
	)))
	if got := len(m.currentColumnTasks()); got != 1 {
		t.Fatalf("expected project scope to hide top-level subtask rows, got %d visible", got)
	}

	m.focusTaskByID(branch.ID)
	m = applyMsg(t, m, keyRune('f'))
	if m.projectionRootTaskID != branch.ID {
		t.Fatalf("expected subtree focus root %q, got %q", branch.ID, m.projectionRootTaskID)
	}
	if got := len(m.currentColumnTasks()); got != 1 {
		t.Fatalf("expected focused scope to render direct subtask child, got %d visible", got)
	}
	child, ok := m.selectedTaskInCurrentColumn()
	if !ok || child.ID != legacySubtask.ID {
		t.Fatalf("expected focused child %q, got task=%#v ok=%t", legacySubtask.ID, child, ok)
	}
}

// TestModelNewTaskFormDefaultsFollowFocusedScope verifies add-task defaults follow active focus scope.
func TestModelNewTaskFormDefaultsFollowFocusedScope(t *testing.T) {
	now := time.Date(2026, 2, 25, 14, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Roadmap", "", now)
	todo, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	branch, _ := domain.NewTask(domain.TaskInput{
		ID:        "b1",
		ProjectID: p.ID,
		ColumnID:  todo.ID,
		Position:  0,
		Title:     "Branch",
		Priority:  domain.PriorityMedium,
		Kind:      domain.WorkKind("branch"),
		Scope:     domain.KindAppliesToBranch,
	}, now)
	phase, _ := domain.NewTask(domain.TaskInput{
		ID:        "ph1",
		ProjectID: p.ID,
		ParentID:  branch.ID,
		ColumnID:  todo.ID,
		Position:  1,
		Title:     "Phase",
		Priority:  domain.PriorityMedium,
		Kind:      domain.WorkKindPhase,
		Scope:     domain.KindAppliesToPhase,
	}, now)
	nestedPhase, _ := domain.NewTask(domain.TaskInput{
		ID:        "sph1",
		ProjectID: p.ID,
		ParentID:  phase.ID,
		ColumnID:  todo.ID,
		Position:  2,
		Title:     "Nested Phase",
		Priority:  domain.PriorityMedium,
		Kind:      domain.WorkKindPhase,
		Scope:     domain.KindAppliesToPhase,
	}, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ParentID:  nestedPhase.ID,
		ColumnID:  todo.ID,
		Position:  3,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
		Kind:      domain.WorkKindTask,
		Scope:     domain.KindAppliesToTask,
	}, now)
	subtask, _ := domain.NewTask(domain.TaskInput{
		ID:        "st1",
		ProjectID: p.ID,
		ParentID:  task.ID,
		ColumnID:  todo.ID,
		Position:  4,
		Title:     "Subtask",
		Priority:  domain.PriorityLow,
		Kind:      domain.WorkKindSubtask,
		Scope:     domain.KindAppliesToSubtask,
	}, now)
	m := loadReadyModel(t, NewModel(newFakeService(
		[]domain.Project{p},
		[]domain.Column{todo},
		[]domain.Task{branch, phase, nestedPhase, task, subtask},
	)))

	assertDefaults := func(parentID string, kind domain.WorkKind, scope domain.KindAppliesTo) {
		t.Helper()
		if got := m.taskFormParentID; got != parentID {
			t.Fatalf("task form parent = %q, want %q", got, parentID)
		}
		if got := m.taskFormKind; got != kind {
			t.Fatalf("task form kind = %q, want %q", got, kind)
		}
		if got := m.taskFormScope; got != scope {
			t.Fatalf("task form scope = %q, want %q", got, scope)
		}
	}

	m = applyMsg(t, m, keyRune('n'))
	assertDefaults("", domain.WorkKindTask, domain.KindAppliesToTask)
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})

	m.focusTaskByID(branch.ID)
	m = applyMsg(t, m, keyRune('f'))
	m = applyMsg(t, m, keyRune('n'))
	assertDefaults(branch.ID, domain.WorkKindTask, domain.KindAppliesToTask)
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})

	m = applyMsg(t, m, keyRune('f'))
	m = applyMsg(t, m, keyRune('n'))
	assertDefaults(phase.ID, domain.WorkKindTask, domain.KindAppliesToTask)
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})

	m = applyMsg(t, m, keyRune('f'))
	m = applyMsg(t, m, keyRune('n'))
	assertDefaults(nestedPhase.ID, domain.WorkKindTask, domain.KindAppliesToTask)
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})

	m = applyMsg(t, m, keyRune('f'))
	m = applyMsg(t, m, keyRune('n'))
	assertDefaults(task.ID, domain.WorkKindSubtask, domain.KindAppliesToSubtask)
}

// TestModelCreateTaskFromFocusedScopeUsesScopedParent verifies submitted create-task calls carry focused defaults.
func TestModelCreateTaskFromFocusedScopeUsesScopedParent(t *testing.T) {
	now := time.Date(2026, 2, 25, 14, 30, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Roadmap", "", now)
	todo, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	branch, _ := domain.NewTask(domain.TaskInput{
		ID:        "b1",
		ProjectID: p.ID,
		ColumnID:  todo.ID,
		Position:  0,
		Title:     "Branch",
		Priority:  domain.PriorityMedium,
		Kind:      domain.WorkKind("branch"),
		Scope:     domain.KindAppliesToBranch,
	}, now)
	phase, _ := domain.NewTask(domain.TaskInput{
		ID:        "ph1",
		ProjectID: p.ID,
		ParentID:  branch.ID,
		ColumnID:  todo.ID,
		Position:  1,
		Title:     "Phase",
		Priority:  domain.PriorityMedium,
		Kind:      domain.WorkKindPhase,
		Scope:     domain.KindAppliesToPhase,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{todo}, []domain.Task{branch, phase})
	m := loadReadyModel(t, NewModel(svc))
	m.focusTaskByID(branch.ID)
	m = applyMsg(t, m, keyRune('f'))
	m = applyMsg(t, m, keyRune('n'))
	if m.mode != modeAddTask {
		t.Fatalf("expected add-task mode, got %v", m.mode)
	}
	m.formInputs[taskFieldTitle].SetValue("Focused child")
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if svc.createTaskCalls != 1 {
		t.Fatalf("expected one create-task call, got %d", svc.createTaskCalls)
	}
	if got := svc.lastCreateTask.ParentID; got != branch.ID {
		t.Fatalf("create parent_id = %q, want %q", got, branch.ID)
	}
	if got := svc.lastCreateTask.Kind; got != domain.WorkKindTask {
		t.Fatalf("create kind = %q, want %q", got, domain.WorkKindTask)
	}
	if got := svc.lastCreateTask.Scope; got != domain.KindAppliesToTask {
		t.Fatalf("create scope = %q, want %q", got, domain.KindAppliesToTask)
	}
}

// TestModelViewShowsSubtreeDiscoverabilityHint verifies hierarchy focus guidance in the board info line.
func TestModelViewShowsSubtreeDiscoverabilityHint(t *testing.T) {
	now := time.Date(2026, 2, 24, 11, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Roadmap", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	branch, _ := domain.NewTask(domain.TaskInput{
		ID:        "b1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Branch",
		Priority:  domain.PriorityMedium,
		Kind:      domain.WorkKind("branch"),
		Scope:     domain.KindAppliesToBranch,
	}, now)
	phase, _ := domain.NewTask(domain.TaskInput{
		ID:        "p2",
		ProjectID: p.ID,
		ParentID:  branch.ID,
		ColumnID:  c.ID,
		Position:  1,
		Title:     "Phase",
		Priority:  domain.PriorityMedium,
		Kind:      domain.WorkKindPhase,
		Scope:     domain.KindAppliesToPhase,
	}, now)

	m := loadReadyModel(t, NewModel(newFakeService(
		[]domain.Project{p},
		[]domain.Column{c},
		[]domain.Task{branch, phase},
	)))
	view := stripANSI(fmt.Sprint(m.View().Content))
	if !strings.Contains(view, "children: 1") {
		t.Fatalf("expected direct child count in info line, got\n%s", view)
	}
	if !strings.Contains(view, "f focus subtree") {
		t.Fatalf("expected focus subtree hint in info line, got\n%s", view)
	}

	m = applyMsg(t, m, keyRune('f'))
	focused := stripANSI(fmt.Sprint(m.View().Content))
	if !strings.Contains(focused, "F full board") {
		t.Fatalf("expected full-board hint while focused, got\n%s", focused)
	}
	if !strings.Contains(focused, "path: Roadmap -> Branch") {
		t.Fatalf("expected focused path line while subtree focus is active, got\n%s", focused)
	}
}

// TestModelFocusSubtreeAllowsEmptyScope verifies pressing f on leaf nodes still enters focus mode.
func TestModelFocusSubtreeAllowsEmptyScope(t *testing.T) {
	now := time.Date(2026, 2, 25, 13, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Roadmap", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "leaf-task",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Leaf Task",
		Priority:  domain.PriorityMedium,
		Kind:      domain.WorkKindTask,
		Scope:     domain.KindAppliesToTask,
	}, now)
	m := loadReadyModel(t, NewModel(newFakeService(
		[]domain.Project{p},
		[]domain.Column{c},
		[]domain.Task{task},
	)))
	m = applyMsg(t, m, keyRune('f'))
	if m.projectionRootTaskID != task.ID {
		t.Fatalf("expected focus root %q for empty scope, got %q", task.ID, m.projectionRootTaskID)
	}
	if m.status != "focused subtree" {
		t.Fatalf("expected focused-subtree status, got %q", m.status)
	}
	rendered := stripANSI(fmt.Sprint(m.View().Content))
	if !strings.Contains(rendered, "path: Roadmap -> Leaf Task") {
		t.Fatalf("expected focused path line for empty scope, got\n%s", rendered)
	}
}

// TestModelViewShowsHierarchyMarkers verifies branch/phase markers in card metadata rows.
func TestModelViewShowsHierarchyMarkers(t *testing.T) {
	now := time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Roadmap", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	branch, _ := domain.NewTask(domain.TaskInput{
		ID:        "b1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Branch",
		Priority:  domain.PriorityMedium,
		Kind:      domain.WorkKind("branch"),
		Scope:     domain.KindAppliesToBranch,
	}, now)
	phase, _ := domain.NewTask(domain.TaskInput{
		ID:        "p2",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  1,
		Title:     "Phase",
		Priority:  domain.PriorityMedium,
		Kind:      domain.WorkKindPhase,
		Scope:     domain.KindAppliesToPhase,
	}, now)

	m := loadReadyModel(t, NewModel(newFakeService(
		[]domain.Project{p},
		[]domain.Column{c},
		[]domain.Task{branch, phase},
	)))
	view := stripANSI(fmt.Sprint(m.View().Content))
	if !strings.Contains(view, "[branch|medium]") {
		t.Fatalf("expected branch marker in card metadata, got\n%s", view)
	}
	if !strings.Contains(view, "[phase|medium]") {
		t.Fatalf("expected phase marker in card metadata, got\n%s", view)
	}
}

// TestModelViewShowsNoticesPanel verifies right-side notices panel rendering on wide layouts.
func TestModelViewShowsNoticesPanel(t *testing.T) {
	now := time.Date(2026, 2, 25, 11, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	todo, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	progress, _ := domain.NewColumn("c2", p.ID, "In Progress", 1, 0, now)
	done, _ := domain.NewColumn("c3", p.ID, "Done", 2, 0, now)
	blocked, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-blocked",
		ProjectID: p.ID,
		ColumnID:  todo.ID,
		Position:  0,
		Title:     "Blocked",
		Priority:  domain.PriorityHigh,
		Metadata: domain.TaskMetadata{
			BlockedReason: "waiting on approval",
			DependsOn:     []string{"missing"},
		},
	}, now)
	m := loadReadyModel(t, NewModel(newFakeService(
		[]domain.Project{p},
		[]domain.Column{todo, progress, done},
		[]domain.Task{blocked},
	)))
	// Notices panel rendering now starts at the clean-fit threshold; widen beyond default test viewport.
	m = applyMsg(t, m, tea.WindowSizeMsg{Width: 128, Height: 40})
	rendered := stripANSI(fmt.Sprint(m.View().Content))
	if !strings.Contains(rendered, "Project Notifications") {
		t.Fatalf("expected project notifications panel title, got\n%s", rendered)
	}
	if !strings.Contains(rendered, "Global Notifications") {
		t.Fatalf("expected global notifications panel title, got\n%s", rendered)
	}
	if !strings.Contains(rendered, "Action Required") {
		t.Fatalf("expected notices panel attention section, got\n%s", rendered)
	}
	if !strings.Contains(rendered, "requires user action across") {
		t.Fatalf("expected global notifications subtitle, got\n%s", rendered)
	}
	if strings.Contains(rendered, "Notices\nproject:") || strings.Contains(rendered, "Notices\r\nproject:") {
		t.Fatalf("expected legacy notices fallback block to be absent, got\n%s", rendered)
	}
	if !strings.Contains(rendered, "work items") {
		t.Fatalf("expected blocker warning in notices panel, got\n%s", rendered)
	}
	if !strings.Contains(rendered, "attention items") {
		t.Fatalf("expected attention warning in notices panel, got\n%s", rendered)
	}
}

// TestRenderOverviewPanelOmitsLegacyNoticesFallbackWhenVisible verifies the side panel always renders project/global notifications layout.
func TestRenderOverviewPanelOmitsLegacyNoticesFallbackWhenVisible(t *testing.T) {
	now := time.Date(2026, 3, 2, 8, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{project}, []domain.Column{column}, []domain.Task{task})))

	panel := stripANSI(m.renderOverviewPanel(project, lipgloss.Color("62"), lipgloss.Color("241"), lipgloss.Color("239"), 44, 26, 0, 0, 0, nil, false))
	if !strings.Contains(panel, "Project Notifications") {
		t.Fatalf("expected project notifications panel title, got\n%s", panel)
	}
	if !strings.Contains(panel, "Global Notifications") {
		t.Fatalf("expected global notifications panel title, got\n%s", panel)
	}
	if strings.Contains(panel, "Notices") {
		t.Fatalf("expected legacy single-panel notices title to be absent, got\n%s", panel)
	}
	if strings.Contains(panel, "project: "+projectDisplayName(project)) {
		t.Fatalf("expected legacy project fallback line to be absent, got\n%s", panel)
	}
	if strings.Contains(panel, "path: "+projectDisplayName(project)) {
		t.Fatalf("expected legacy path fallback line to be absent, got\n%s", panel)
	}
}

// TestModelBoardHorizontalSpacingSymmetry verifies equal one-cell outer gutters and inter-panel gaps.
func TestModelBoardHorizontalSpacingSymmetry(t *testing.T) {
	now := time.Date(2026, 2, 27, 18, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	todo, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	progress, _ := domain.NewColumn("c2", p.ID, "In Progress", 1, 0, now)
	done, _ := domain.NewColumn("c3", p.ID, "Done", 2, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  todo.ID,
		Position:  0,
		Title:     "Task 1",
		Priority:  domain.PriorityMedium,
	}, now)

	m := loadReadyModel(t, NewModel(newFakeService(
		[]domain.Project{p},
		[]domain.Column{todo, progress, done},
		[]domain.Task{task},
	)))

	assertSpacing := func(width int) {
		scoped := applyMsg(t, m, tea.WindowSizeMsg{Width: width, Height: 40})
		rendered := stripANSI(fmt.Sprint(scoped.View().Content))
		layoutWidth := max(0, width-2*tuiOuterHorizontalPadding)
		expectedPanels := len(scoped.columns)
		if scoped.noticesPanelWidth(layoutWidth) > 0 {
			expectedPanels++
		}
		lines := strings.Split(rendered, "\n")
		borderLine := ""
		for _, line := range lines {
			if strings.Count(line, "╭") == expectedPanels {
				borderLine = line
				break
			}
		}
		if borderLine == "" {
			t.Fatalf("expected border row with %d panels at width %d, got\n%s", expectedPanels, width, rendered)
		}

		leadingSpaces := len(borderLine) - len(strings.TrimLeft(borderLine, " "))
		trailingSpaces := len(borderLine) - len(strings.TrimRight(borderLine, " "))
		if leadingSpaces != tuiOuterHorizontalPadding {
			t.Fatalf("expected %d leading space, got %d in %q", tuiOuterHorizontalPadding, leadingSpaces, borderLine)
		}
		if trailingSpaces != tuiOuterHorizontalPadding {
			t.Fatalf("expected %d trailing space, got %d in %q", tuiOuterHorizontalPadding, trailingSpaces, borderLine)
		}

		if got := strings.Count(borderLine, "╮ ╭"); got != expectedPanels-1 {
			t.Fatalf("expected %d single-space panel joins, got %d in %q", expectedPanels-1, got, borderLine)
		}
	}

	assertSpacing(96)
	assertSpacing(110)
	assertSpacing(128)
	assertSpacing(144)
}

// TestModelViewShowsAttentionMarkersAndSummary verifies unresolved-attention markers and compact scope totals in board view.
func TestModelViewShowsAttentionMarkersAndSummary(t *testing.T) {
	now := time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	todo, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	progress, _ := domain.NewColumn("c2", p.ID, "In Progress", 1, 0, now)
	done, _ := domain.NewColumn("c3", p.ID, "Done", 2, 0, now)

	doneTask, _ := domain.NewTask(domain.TaskInput{
		ID:             "t-done",
		ProjectID:      p.ID,
		ColumnID:       done.ID,
		Position:       0,
		Title:          "Done Task",
		Priority:       domain.PriorityLow,
		Kind:           domain.WorkKindTask,
		Scope:          domain.KindAppliesToTask,
		LifecycleState: domain.StateDone,
	}, now)
	blockedTask, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-blocked",
		ProjectID: p.ID,
		ColumnID:  todo.ID,
		Position:  0,
		Title:     "Blocked Task",
		Priority:  domain.PriorityHigh,
		Kind:      domain.WorkKindTask,
		Scope:     domain.KindAppliesToTask,
		Metadata: domain.TaskMetadata{
			DependsOn:     []string{"t-missing"},
			BlockedReason: "waiting on partner team",
		},
	}, now)
	waitingTask, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-waiting",
		ProjectID: p.ID,
		ColumnID:  progress.ID,
		Position:  0,
		Title:     "Waiting Task",
		Priority:  domain.PriorityMedium,
		Kind:      domain.WorkKindTask,
		Scope:     domain.KindAppliesToTask,
		Metadata: domain.TaskMetadata{
			BlockedBy: []string{"t-not-found"},
		},
	}, now)

	m := loadReadyModel(t, NewModel(newFakeService(
		[]domain.Project{p},
		[]domain.Column{todo, progress, done},
		[]domain.Task{doneTask, blockedTask, waitingTask},
	)))
	rendered := stripANSI(fmt.Sprint(m.View().Content))
	if strings.Contains(rendered, "attention: 3") {
		t.Fatalf("expected header to stay path-only (no attention token), got\n%s", rendered)
	}
	if strings.Contains(rendered, "attention scope: 2 items • unresolved 3 • blocked 1") {
		t.Fatalf("expected board summary cleanup to remove attention summary line, got\n%s", rendered)
	}
	if !strings.Contains(rendered, "Project Notifications") {
		t.Fatalf("expected project notifications panel to render, got\n%s", rendered)
	}
	if !strings.Contains(rendered, "Blocked Task !2") {
		t.Fatalf("expected row marker for blocked task, got\n%s", rendered)
	}
	if !strings.Contains(rendered, "Waiting Task !1") {
		t.Fatalf("expected row marker for waiting task, got\n%s", rendered)
	}
}

// TestSearchLevelFiltering verifies level-scoped filtering for project, branch, phase, task, and subtask.
func TestSearchLevelFiltering(t *testing.T) {
	now := time.Date(2026, 2, 24, 11, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Hierarchy", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	branch, _ := domain.NewTask(domain.TaskInput{
		ID:        "l-branch",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Branch",
		Priority:  domain.PriorityLow,
		Kind:      domain.WorkKind("branch"),
		Scope:     domain.KindAppliesToBranch,
	}, now)
	phase, _ := domain.NewTask(domain.TaskInput{
		ID:        "l-phase",
		ProjectID: p.ID,
		ParentID:  branch.ID,
		ColumnID:  c.ID,
		Position:  1,
		Title:     "Phase",
		Priority:  domain.PriorityLow,
		Kind:      domain.WorkKindPhase,
		Scope:     domain.KindAppliesToPhase,
	}, now)
	nestedPhase, _ := domain.NewTask(domain.TaskInput{
		ID:        "l-nested-phase",
		ProjectID: p.ID,
		ParentID:  phase.ID,
		ColumnID:  c.ID,
		Position:  2,
		Title:     "Nested Phase",
		Priority:  domain.PriorityLow,
		Kind:      domain.WorkKindPhase,
		Scope:     domain.KindAppliesToPhase,
	}, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "l-task",
		ProjectID: p.ID,
		ParentID:  nestedPhase.ID,
		ColumnID:  c.ID,
		Position:  3,
		Title:     "Task",
		Priority:  domain.PriorityLow,
		Kind:      domain.WorkKindTask,
		Scope:     domain.KindAppliesToTask,
	}, now)
	subtask, _ := domain.NewTask(domain.TaskInput{
		ID:        "l-subtask",
		ProjectID: p.ID,
		ParentID:  task.ID,
		ColumnID:  c.ID,
		Position:  4,
		Title:     "Subtask",
		Priority:  domain.PriorityLow,
		Kind:      domain.WorkKindSubtask,
		Scope:     domain.KindAppliesToSubtask,
	}, now)

	m := Model{
		tasks: []domain.Task{branch, phase, nestedPhase, task, subtask},
	}
	matches := []app.TaskMatch{
		{Project: p, Task: branch, StateID: "todo"},
		{Project: p, Task: phase, StateID: "todo"},
		{Project: p, Task: nestedPhase, StateID: "todo"},
		{Project: p, Task: task, StateID: "todo"},
		{Project: p, Task: subtask, StateID: "todo"},
	}
	ids := func(in []app.TaskMatch) string {
		out := make([]string, 0, len(in))
		for _, match := range in {
			out = append(out, match.Task.ID)
		}
		return strings.Join(out, ",")
	}

	cases := []struct {
		name   string
		levels []string
		want   string
	}{
		{name: "project", levels: []string{"project"}, want: "l-branch,l-phase,l-nested-phase,l-task,l-subtask"},
		{name: "branch", levels: []string{"branch"}, want: "l-branch"},
		{name: "phase", levels: []string{"phase"}, want: "l-phase,l-nested-phase"},
		{name: "task", levels: []string{"task"}, want: "l-task"},
		{name: "subtask", levels: []string{"subtask"}, want: "l-subtask"},
	}
	for _, tc := range cases {
		m.searchLevels = tc.levels
		got := ids(m.filterTaskMatchesBySearchLevels(matches))
		if got != tc.want {
			t.Fatalf("%s level filter mismatch: want %q, got %q", tc.name, tc.want, got)
		}
	}

	ready := loadReadyModel(t, NewModel(newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{branch, phase, nestedPhase, task, subtask})))
	ready = applyMsg(t, ready, keyRune('/'))
	ready = applyMsg(t, ready, tea.KeyPressMsg{Code: tea.KeyTab}) // states
	ready = applyMsg(t, ready, tea.KeyPressMsg{Code: tea.KeyTab}) // levels
	if ready.searchFocus != 2 {
		t.Fatalf("expected levels focus slot in search modal, got %d", ready.searchFocus)
	}
	if !ready.isSearchLevelEnabled("project") {
		t.Fatalf("expected project level enabled by default, got %#v", ready.searchLevels)
	}
	ready = applyMsg(t, ready, keyRune(' '))
	if ready.isSearchLevelEnabled("project") {
		t.Fatalf("expected project level toggle to disable via level-scoped controls, got %#v", ready.searchLevels)
	}
}

// TestModelDependencyRollupAndTaskInfoHints verifies rollup summary and task dependency hints.
func TestModelDependencyRollupAndTaskInfoHints(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	done, _ := domain.NewTask(domain.TaskInput{
		ID:             "t-done",
		ProjectID:      p.ID,
		ColumnID:       c.ID,
		Position:       0,
		Title:          "Finished",
		Priority:       domain.PriorityLow,
		LifecycleState: domain.StateDone,
	}, now)
	blocked, _ := domain.NewTask(domain.TaskInput{
		ID:             "t-blocked",
		ProjectID:      p.ID,
		ColumnID:       c.ID,
		Position:       1,
		Title:          "Blocked",
		Priority:       domain.PriorityHigh,
		LifecycleState: domain.StateTodo,
		Metadata: domain.TaskMetadata{
			DependsOn:     []string{"t-done", "t-missing"},
			BlockedBy:     []string{"t-done"},
			BlockedReason: "waiting on integration",
		},
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{done, blocked})
	m := loadReadyModel(t, NewModel(svc))

	if summary := m.dependencyRollupSummary(); !strings.Contains(summary, "total 2") || !strings.Contains(summary, "blocked 1") || !strings.Contains(summary, "unresolved 1") {
		t.Fatalf("expected dependency rollup summary counts, got %q", summary)
	}

	m = applyMsg(t, m, keyRune('j'))
	m = applyMsg(t, m, keyRune('i'))
	infoBody := stripANSI(m.taskInfoBody.GetContent())
	if !strings.Contains(infoBody, "depends_on: t-done(Finished)") {
		t.Fatalf("expected depends_on hints in task info body, got %q", infoBody)
	}
	if !strings.Contains(infoBody, "blocked_by: t-done(Finished)") {
		t.Fatalf("expected blocked_by hints in task info body, got %q", infoBody)
	}
	if !strings.Contains(infoBody, "blocked_reason: waiting on integration") {
		t.Fatalf("expected blocked_reason hint in task info body, got %q", infoBody)
	}
}

// TestModelDependencyInspectorPinsLinkedRefsAndAppliesEdits verifies dependency inspector linked-row pinning and edit-form apply flow.
func TestModelDependencyInspectorPinsLinkedRefsAndAppliesEdits(t *testing.T) {
	now := time.Date(2026, 2, 23, 9, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	cTodo, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	cProgress, _ := domain.NewColumn("c2", p.ID, "In Progress", 1, 0, now)
	cDone, _ := domain.NewColumn("c3", p.ID, "Done", 2, 0, now)

	owner, _ := domain.NewTask(domain.TaskInput{
		ID:             "t-owner",
		ProjectID:      p.ID,
		ColumnID:       cTodo.ID,
		Position:       0,
		Title:          "Owner",
		Priority:       domain.PriorityMedium,
		LifecycleState: domain.StateTodo,
	}, now)
	depDone, _ := domain.NewTask(domain.TaskInput{
		ID:             "t-done",
		ProjectID:      p.ID,
		ColumnID:       cDone.ID,
		Position:       0,
		Title:          "Done dependency",
		Priority:       domain.PriorityLow,
		LifecycleState: domain.StateDone,
	}, now.Add(time.Minute))
	depArchived, _ := domain.NewTask(domain.TaskInput{
		ID:             "t-archived",
		ProjectID:      p.ID,
		ColumnID:       cDone.ID,
		Position:       1,
		Title:          "Archived dependency",
		Priority:       domain.PriorityLow,
		LifecycleState: domain.StateArchived,
	}, now.Add(2*time.Minute))
	archivedAt := now.Add(-time.Minute)
	depArchived.ArchivedAt = &archivedAt
	blocker, _ := domain.NewTask(domain.TaskInput{
		ID:             "t-blocker",
		ProjectID:      p.ID,
		ColumnID:       cProgress.ID,
		Position:       0,
		Title:          "Active blocker",
		Priority:       domain.PriorityHigh,
		LifecycleState: domain.StateProgress,
	}, now.Add(3*time.Minute))
	candidate, _ := domain.NewTask(domain.TaskInput{
		ID:             "t-candidate",
		ProjectID:      p.ID,
		ColumnID:       cTodo.ID,
		Position:       1,
		Title:          "Candidate dependency",
		Priority:       domain.PriorityLow,
		LifecycleState: domain.StateTodo,
	}, now.Add(4*time.Minute))

	owner.Metadata = domain.TaskMetadata{
		DependsOn: []string{owner.ID, depDone.ID, depArchived.ID},
		BlockedBy: []string{blocker.ID},
	}

	svc := newFakeService(
		[]domain.Project{p},
		[]domain.Column{cTodo, cProgress, cDone},
		[]domain.Task{owner, depDone, depArchived, blocker, candidate},
	)
	m := loadReadyModel(t, NewModel(svc))
	m.searchLevels = []string{"task"}
	m.focusTaskByID(owner.ID)
	m = applyMsg(t, m, keyRune('e'))
	m = applyCmd(t, m, m.focusTaskFormField(taskFieldDependsOn))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m = applyMsg(t, m, m.loadDependencyMatches())
	expectedLevels := canonicalSearchLevels(m.searchDefaultLevels)
	if len(svc.lastSearchFilter.Levels) != len(expectedLevels) {
		t.Fatalf("expected dependency inspector levels %#v, got %#v", expectedLevels, svc.lastSearchFilter.Levels)
	}
	for idx, level := range expectedLevels {
		if svc.lastSearchFilter.Levels[idx] != level {
			t.Fatalf("expected dependency inspector levels %#v, got %#v", expectedLevels, svc.lastSearchFilter.Levels)
		}
	}
	if got := svc.lastSearchFilter.Mode; got != app.SearchModeHybrid {
		t.Fatalf("expected dependency inspector search mode %q, got %q", app.SearchModeHybrid, got)
	}
	if got := svc.lastSearchFilter.Sort; got != app.SearchSortRankDesc {
		t.Fatalf("expected dependency inspector search sort %q, got %q", app.SearchSortRankDesc, got)
	}
	if got := svc.lastSearchFilter.Limit; got != defaultSearchResultsLimit {
		t.Fatalf("expected dependency inspector search limit %d, got %d", defaultSearchResultsLimit, got)
	}
	if got := svc.lastSearchFilter.Offset; got != 0 {
		t.Fatalf("expected dependency inspector search offset 0, got %d", got)
	}
	if m.mode != modeDependencyInspector {
		t.Fatalf("expected dependency inspector mode, got %v", m.mode)
	}
	if len(m.dependencyMatches) < 3 {
		t.Fatalf("expected linked rows loaded, got %d", len(m.dependencyMatches))
	}
	if idx := dependencyCandidateIndexByID(m.dependencyMatches, owner.ID); idx >= 0 {
		t.Fatalf("expected owner task %q to be excluded from dependency candidates", owner.ID)
	}
	if got := m.dependencyMatches[0].Match.Task.ID; got != depDone.ID {
		t.Fatalf("expected first pinned row %q, got %q", depDone.ID, got)
	}
	if got := m.dependencyMatches[1].Match.Task.ID; got != depArchived.ID {
		t.Fatalf("expected second pinned row %q, got %q", depArchived.ID, got)
	}
	if got := m.dependencyMatches[2].Match.Task.ID; got != blocker.ID {
		t.Fatalf("expected third pinned row %q, got %q", blocker.ID, got)
	}

	foundArchived := false
	for _, candidateRow := range m.dependencyMatches {
		if candidateRow.Match.Task.ID == depArchived.ID {
			foundArchived = true
			break
		}
	}
	if !foundArchived {
		t.Fatal("expected linked archived dependency to remain visible in inspector list")
	}

	for i := 0; i < 4; i++ {
		m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	}
	if m.dependencyFocus != 4 {
		t.Fatalf("expected list focus, got %d", m.dependencyFocus)
	}
	addIdx := dependencyCandidateIndexByID(m.dependencyMatches, candidate.ID)
	if addIdx < 0 {
		t.Fatalf("expected candidate row %q", candidate.ID)
	}
	m.dependencyIndex = addIdx
	m = applyMsg(t, m, keyRune('d'))
	m = applyMsg(t, m, keyRune('a'))
	if m.mode != modeEditTask {
		t.Fatalf("expected return to edit-task after apply, got %v", m.mode)
	}
	formDepends := parseTaskRefIDsInput(m.formInputs[taskFieldDependsOn].Value(), nil)
	if !hasDependencyID(formDepends, candidate.ID) {
		t.Fatalf("expected candidate dependency %q staged in form, got %#v", candidate.ID, formDepends)
	}
	if hasDependencyID(formDepends, owner.ID) {
		t.Fatalf("expected self dependency %q stripped from staged form value, got %#v", owner.ID, formDepends)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	ownerAfter, ok := svc.taskByID(owner.ID)
	if !ok {
		t.Fatalf("expected owner task %q after save", owner.ID)
	}
	if !hasDependencyID(ownerAfter.Metadata.DependsOn, candidate.ID) {
		t.Fatalf("expected candidate dependency %q saved, got %#v", candidate.ID, ownerAfter.Metadata.DependsOn)
	}
	if hasDependencyID(ownerAfter.Metadata.DependsOn, owner.ID) {
		t.Fatalf("expected self dependency %q to be stripped on save, got %#v", owner.ID, ownerAfter.Metadata.DependsOn)
	}
}

// TestModelDependencyInspectorEnterFromTaskForm verifies task-form dependency rows open via enter and apply modal-only updates.
func TestModelDependencyInspectorEnterFromTaskForm(t *testing.T) {
	now := time.Date(2026, 2, 23, 11, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	owner, _ := domain.NewTask(domain.TaskInput{
		ID:             "t-owner",
		ProjectID:      p.ID,
		ColumnID:       c.ID,
		Position:       0,
		Title:          "Owner",
		Priority:       domain.PriorityMedium,
		LifecycleState: domain.StateTodo,
	}, now)
	candidate, _ := domain.NewTask(domain.TaskInput{
		ID:             "t-candidate",
		ProjectID:      p.ID,
		ColumnID:       c.ID,
		Position:       1,
		Title:          "Candidate",
		Priority:       domain.PriorityLow,
		LifecycleState: domain.StateTodo,
	}, now.Add(time.Minute))

	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{owner, candidate})))
	m = applyMsg(t, m, keyRune('e'))
	if m.mode != modeEditTask {
		t.Fatalf("expected edit-task mode, got %v", m.mode)
	}
	m = applyCmd(t, m, m.focusTaskFormField(taskFieldDependsOn))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m = applyMsg(t, m, m.loadDependencyMatches())
	if m.mode != modeDependencyInspector {
		t.Fatalf("expected dependency inspector from enter, got %v", m.mode)
	}

	for i := 0; i < 4; i++ {
		m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	}
	addIdx := dependencyCandidateIndexByID(m.dependencyMatches, candidate.ID)
	if addIdx < 0 {
		t.Fatalf("expected candidate row %q", candidate.ID)
	}
	m.dependencyIndex = addIdx
	m = applyMsg(t, m, keyRune('d'))
	m = applyMsg(t, m, keyRune('a'))
	if m.mode != modeEditTask {
		t.Fatalf("expected return to edit-task mode after apply, got %v", m.mode)
	}
	if got := m.formInputs[taskFieldDependsOn].Value(); got != candidate.ID {
		t.Fatalf("expected depends_on CSV %q, got %q", candidate.ID, got)
	}
}

// TestModelDependencyInspectorOverlayRendersMissingLinkedRefs verifies missing linked references stay inspectable in the dependency modal.
func TestModelDependencyInspectorOverlayRendersMissingLinkedRefs(t *testing.T) {
	now := time.Date(2026, 2, 23, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	owner, _ := domain.NewTask(domain.TaskInput{
		ID:             "t-owner",
		ProjectID:      p.ID,
		ColumnID:       c.ID,
		Position:       0,
		Title:          "Owner",
		Priority:       domain.PriorityMedium,
		LifecycleState: domain.StateTodo,
	}, now)
	blocker, _ := domain.NewTask(domain.TaskInput{
		ID:             "t-blocker",
		ProjectID:      p.ID,
		ColumnID:       c.ID,
		Position:       1,
		Title:          "Blocker",
		Priority:       domain.PriorityLow,
		LifecycleState: domain.StateTodo,
	}, now.Add(time.Minute))
	owner.Metadata = domain.TaskMetadata{
		DependsOn: []string{"t-missing"},
		BlockedBy: []string{blocker.ID},
	}

	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{owner, blocker})))
	m = applyMsg(t, m, keyRune('e'))
	m = applyCmd(t, m, m.focusTaskFormField(taskFieldDependsOn))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m = applyMsg(t, m, m.loadDependencyMatches())
	if m.mode != modeDependencyInspector {
		t.Fatalf("expected dependency inspector mode, got %v", m.mode)
	}

	out := stripANSI(m.renderModeOverlay(
		lipgloss.Color("62"),
		lipgloss.Color("241"),
		lipgloss.Color("239"),
		lipgloss.NewStyle(),
		96,
	))
	if !strings.Contains(out, "linked refs are pinned at top") {
		t.Fatalf("expected pinned-linked hint, got %q", out)
	}
	if !strings.Contains(out, "(missing task reference)") {
		t.Fatalf("expected missing reference row, got %q", out)
	}
	if !strings.Contains(out, "state: missing") {
		t.Fatalf("expected missing reference details state, got %q", out)
	}
}

// TestModelDependencyInspectorFormEnterDoesNotJump verifies enter-jump is blocked when opened from task form context.
func TestModelDependencyInspectorFormEnterDoesNotJump(t *testing.T) {
	now := time.Date(2026, 2, 23, 13, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	owner, _ := domain.NewTask(domain.TaskInput{
		ID:             "t-owner",
		ProjectID:      p.ID,
		ColumnID:       c.ID,
		Position:       0,
		Title:          "Owner",
		Priority:       domain.PriorityMedium,
		LifecycleState: domain.StateTodo,
	}, now)
	candidate, _ := domain.NewTask(domain.TaskInput{
		ID:             "t-candidate",
		ProjectID:      p.ID,
		ColumnID:       c.ID,
		Position:       1,
		Title:          "Candidate",
		Priority:       domain.PriorityLow,
		LifecycleState: domain.StateTodo,
	}, now.Add(time.Minute))

	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{owner, candidate})))
	m = applyMsg(t, m, keyRune('e'))
	m = applyCmd(t, m, m.focusTaskFormField(taskFieldDependsOn))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m = applyMsg(t, m, m.loadDependencyMatches())
	if m.mode != modeDependencyInspector {
		t.Fatalf("expected dependency inspector mode, got %v", m.mode)
	}
	for i := 0; i < 4; i++ {
		m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	}
	m.dependencyIndex = dependencyCandidateIndexByID(m.dependencyMatches, candidate.ID)
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeDependencyInspector {
		t.Fatalf("expected dependency inspector to stay open on enter from form mode, got %v", m.mode)
	}
	if !strings.Contains(m.status, "task-info inspector") {
		t.Fatalf("expected non-task-info jump status, got %q", m.status)
	}
}

// TestDependencyStateIDForTask verifies fallback state-id derivation for dependency rows.
func TestDependencyStateIDForTask(t *testing.T) {
	now := time.Date(2026, 2, 23, 14, 0, 0, 0, time.UTC)
	archivedAt := now.Add(-time.Minute)
	taskArchived := domain.Task{LifecycleState: domain.StateDone, ArchivedAt: &archivedAt}
	taskProgress := domain.Task{LifecycleState: domain.StateProgress}
	taskUnknown := domain.Task{LifecycleState: domain.LifecycleState("review")}
	taskEmpty := domain.Task{}

	if got := dependencyStateIDForTask(taskArchived); got != "archived" {
		t.Fatalf("expected archived state id, got %q", got)
	}
	if got := dependencyStateIDForTask(taskProgress); got != "progress" {
		t.Fatalf("expected progress state id, got %q", got)
	}
	if got := dependencyStateIDForTask(taskUnknown); got != "review" {
		t.Fatalf("expected custom normalized state id, got %q", got)
	}
	if got := dependencyStateIDForTask(taskEmpty); got != "todo" {
		t.Fatalf("expected todo fallback state id, got %q", got)
	}
}

// TestModelDependencyInspectorFilterControls verifies dependency inspector control paths for query/filter toggles and cancel flow.
func TestModelDependencyInspectorFilterControls(t *testing.T) {
	now := time.Date(2026, 2, 23, 15, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	owner, _ := domain.NewTask(domain.TaskInput{
		ID:             "t-owner",
		ProjectID:      p.ID,
		ColumnID:       c.ID,
		Position:       0,
		Title:          "Owner",
		Priority:       domain.PriorityMedium,
		LifecycleState: domain.StateTodo,
	}, now)
	candidate, _ := domain.NewTask(domain.TaskInput{
		ID:             "t-candidate",
		ProjectID:      p.ID,
		ColumnID:       c.ID,
		Position:       1,
		Title:          "Candidate",
		Priority:       domain.PriorityLow,
		LifecycleState: domain.StateTodo,
	}, now.Add(time.Minute))
	owner.Metadata = domain.TaskMetadata{DependsOn: []string{candidate.ID}}

	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{owner, candidate})))
	m = applyMsg(t, m, keyRune('e'))
	m = applyCmd(t, m, m.focusTaskFormField(taskFieldDependsOn))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m = applyMsg(t, m, m.loadDependencyMatches())
	if m.mode != modeDependencyInspector {
		t.Fatalf("expected dependency inspector mode, got %v", m.mode)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab}) // states focus
	m = applyMsg(t, m, keyRune('x'))
	if m.dependencyActiveField != taskFieldBlockedBy {
		t.Fatalf("expected active field switch to blocked_by, got %d", m.dependencyActiveField)
	}
	m = applyMsg(t, m, keyRune('k')) // query focus
	for _, r := range []rune("qq") {
		m = applyMsg(t, m, keyRune(r))
	}
	if got := strings.TrimSpace(m.dependencyInput.Value()); got != "qq" {
		t.Fatalf("expected typed dependency query, got %q", got)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl})
	if got := strings.TrimSpace(m.dependencyInput.Value()); got != "" {
		t.Fatalf("expected ctrl+u to clear dependency query, got %q", got)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab}) // states focus
	m = applyMsg(t, m, keyRune(' '))
	if m.isDependencyStateEnabled("todo") {
		t.Fatalf("expected todo state disabled after toggle, got %#v", m.dependencyStates)
	}
	m = applyMsg(t, m, keyRune(' '))
	if !m.isDependencyStateEnabled("todo") {
		t.Fatalf("expected todo state restored for list actions, got %#v", m.dependencyStates)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab}) // scope focus
	beforeScope := m.dependencyCrossProject
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.dependencyCrossProject == beforeScope {
		t.Fatalf("expected scope toggle, got %t", m.dependencyCrossProject)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab}) // archived focus
	beforeArchived := m.dependencyIncludeArchived
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.dependencyIncludeArchived == beforeArchived {
		t.Fatalf("expected archived toggle, got %t", m.dependencyIncludeArchived)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab}) // list focus
	if m.dependencyFocus != 4 {
		t.Fatalf("expected list focus, got %d", m.dependencyFocus)
	}
	if m.dependencyActiveField != taskFieldBlockedBy {
		t.Fatalf("expected blocked_by active field before list toggle, got %d", m.dependencyActiveField)
	}
	m = applyMsg(t, m, keyRune(' '))
	if !hasDependencyID(m.dependencyBlockedBy, candidate.ID) {
		t.Fatalf("expected space-toggle add into active blocked_by field, got %#v", m.dependencyBlockedBy)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
	if m.dependencyCrossProject != m.searchDefaultCrossProject {
		t.Fatalf("expected ctrl+r reset scope to default, got %t", m.dependencyCrossProject)
	}
	if m.dependencyIncludeArchived != m.searchDefaultIncludeArchive {
		t.Fatalf("expected ctrl+r reset archived flag, got %t", m.dependencyIncludeArchived)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.mode != modeEditTask {
		t.Fatalf("expected esc to return edit-task mode, got %v", m.mode)
	}
}

// TestModelDependencyInspectorInputAndListKeyRouting verifies that query input keeps text keys while list actions stay list-scoped.
func TestModelDependencyInspectorInputAndListKeyRouting(t *testing.T) {
	now := time.Date(2026, 2, 23, 16, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	owner, _ := domain.NewTask(domain.TaskInput{
		ID:             "t-owner",
		ProjectID:      p.ID,
		ColumnID:       c.ID,
		Position:       0,
		Title:          "Owner",
		Priority:       domain.PriorityMedium,
		LifecycleState: domain.StateTodo,
	}, now)
	candidate, _ := domain.NewTask(domain.TaskInput{
		ID:             "t-candidate",
		ProjectID:      p.ID,
		ColumnID:       c.ID,
		Position:       1,
		Title:          "Candidate",
		Priority:       domain.PriorityLow,
		LifecycleState: domain.StateTodo,
	}, now.Add(time.Minute))

	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{owner, candidate})))
	m = applyMsg(t, m, keyRune('e'))
	m = applyCmd(t, m, m.focusTaskFormField(taskFieldDependsOn))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m = applyMsg(t, m, m.loadDependencyMatches())
	if m.mode != modeDependencyInspector {
		t.Fatalf("expected dependency inspector mode, got %v", m.mode)
	}
	if m.dependencyFocus != 0 {
		t.Fatalf("expected query focus, got %d", m.dependencyFocus)
	}
	initialField := m.dependencyActiveField

	m = applyMsg(t, m, keyRune('x'))
	m = applyMsg(t, m, keyRune('a'))
	m = applyMsg(t, m, keyRune('d'))
	if got := m.dependencyInput.Value(); got != "xad" {
		t.Fatalf("expected action keys to type in query input, got %q", got)
	}
	if m.dependencyActiveField != initialField {
		t.Fatalf("expected active field unchanged while typing query, got %d", m.dependencyActiveField)
	}
	if len(m.dependencyDependsOn) != 0 || len(m.dependencyBlockedBy) != 0 {
		t.Fatalf("expected no dependency toggles while query focused, got depends=%#v blocked=%#v", m.dependencyDependsOn, m.dependencyBlockedBy)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl})
	if got := m.dependencyInput.Value(); got != "" {
		t.Fatalf("expected ctrl+u to clear query before list assertions, got %q", got)
	}

	m = applyMsg(t, m, keyRune('j'))
	if m.dependencyFocus != 1 {
		t.Fatalf("expected j to move focus from query to states, got %d", m.dependencyFocus)
	}
	m = applyMsg(t, m, keyRune('d'))
	m = applyMsg(t, m, keyRune('b'))
	if len(m.dependencyDependsOn) != 0 || len(m.dependencyBlockedBy) != 0 {
		t.Fatalf("expected d/b to be ignored outside list focus, got depends=%#v blocked=%#v", m.dependencyDependsOn, m.dependencyBlockedBy)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	if m.dependencyFocus != 4 {
		t.Fatalf("expected down navigation to reach list focus, got %d", m.dependencyFocus)
	}
	idx := dependencyCandidateIndexByID(m.dependencyMatches, candidate.ID)
	if idx < 0 {
		t.Fatalf("expected candidate row %q", candidate.ID)
	}
	m.dependencyIndex = idx
	m = applyMsg(t, m, keyRune('d'))
	m = applyMsg(t, m, keyRune('b'))
	if !hasDependencyID(m.dependencyDependsOn, candidate.ID) {
		t.Fatalf("expected list-focused d to toggle depends_on, got %#v", m.dependencyDependsOn)
	}
	if !hasDependencyID(m.dependencyBlockedBy, candidate.ID) {
		t.Fatalf("expected list-focused b to toggle blocked_by, got %#v", m.dependencyBlockedBy)
	}
}

// TestModelSearchFocusNavigationWithJK verifies query typing and focus navigation in the search modal.
func TestModelSearchFocusNavigationWithJK(t *testing.T) {
	now := time.Date(2026, 2, 23, 16, 30, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:             "t1",
		ProjectID:      p.ID,
		ColumnID:       c.ID,
		Position:       0,
		Title:          "Task",
		Priority:       domain.PriorityLow,
		LifecycleState: domain.StateTodo,
	}, now)

	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})))
	m = applyMsg(t, m, keyRune('/'))
	if m.mode != modeSearch {
		t.Fatalf("expected search mode, got %v", m.mode)
	}
	if m.searchFocus != 0 {
		t.Fatalf("expected initial search focus on query, got %d", m.searchFocus)
	}

	m = applyMsg(t, m, keyRune('j'))
	if m.searchFocus != 0 {
		t.Fatalf("expected query focus to remain active while typing j, got %d", m.searchFocus)
	}
	if got := m.searchInput.Value(); got != "j" {
		t.Fatalf("expected j to type in query input, got %q", got)
	}

	m = applyMsg(t, m, keyRune('k'))
	if m.searchFocus != 0 {
		t.Fatalf("expected query focus to remain active while typing k, got %d", m.searchFocus)
	}
	if got := m.searchInput.Value(); got != "jk" {
		t.Fatalf("expected k to type in query input, got %q", got)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	if m.searchFocus != 1 {
		t.Fatalf("expected down arrow to move search focus forward, got %d", m.searchFocus)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
	if m.searchFocus != 0 {
		t.Fatalf("expected up arrow to move search focus backward, got %d", m.searchFocus)
	}
}

// TestModelAutoRefreshTickReloadsExternalMutationsInBoardMode verifies board-mode auto-refresh pulls externally written tasks.
func TestModelAutoRefreshTickReloadsExternalMutationsInBoardMode(t *testing.T) {
	now := time.Date(2026, 2, 28, 9, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	existing, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-existing",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Existing",
		Priority:  domain.PriorityLow,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{existing})
	m := loadReadyModel(t, NewModel(svc))
	m.autoRefreshInterval = time.Second

	external, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-external",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  1,
		Title:     "External",
		Priority:  domain.PriorityMedium,
	}, now.Add(time.Minute))
	svc.tasks[p.ID] = append(svc.tasks[p.ID], external)

	m, cmd := applyAutoRefreshTickMsg(t, m)
	if !m.autoRefreshInFlight {
		t.Fatal("expected auto-refresh load to be in flight after tick in board mode")
	}
	m = applyAutoRefreshLoadResult(t, m, cmd)
	if m.autoRefreshInFlight {
		t.Fatal("expected auto-refresh load to complete")
	}
	if len(m.tasks) != 2 {
		t.Fatalf("expected 2 tasks after external refresh, got %d", len(m.tasks))
	}
	if _, ok := m.taskByID(external.ID); !ok {
		t.Fatalf("expected refreshed model to include %q", external.ID)
	}
	if m.mode != modeNone {
		t.Fatalf("expected board mode after refresh, got %v", m.mode)
	}
}

// TestModelAutoRefreshTickSkipsInputModes verifies auto-refresh defers while text-entry modals are active.
func TestModelAutoRefreshTickSkipsInputModes(t *testing.T) {
	now := time.Date(2026, 2, 28, 10, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	existing, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-existing",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Existing",
		Priority:  domain.PriorityLow,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{existing})
	m := loadReadyModel(t, NewModel(svc))
	m.autoRefreshInterval = time.Second
	m = applyMsg(t, m, keyRune('n'))
	if m.mode != modeAddTask {
		t.Fatalf("expected add-task mode, got %v", m.mode)
	}

	external, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-external",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  1,
		Title:     "External",
		Priority:  domain.PriorityMedium,
	}, now.Add(time.Minute))
	svc.tasks[p.ID] = append(svc.tasks[p.ID], external)

	m, _ = applyAutoRefreshTickMsg(t, m)
	if m.autoRefreshInFlight {
		t.Fatal("expected auto-refresh to defer while add-task modal is open")
	}
	if len(m.tasks) != 1 {
		t.Fatalf("expected in-memory board to remain stale during modal input, got %d tasks", len(m.tasks))
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.mode != modeNone {
		t.Fatalf("expected return to board mode after escape, got %v", m.mode)
	}
	m, cmd := applyAutoRefreshTickMsg(t, m)
	if !m.autoRefreshInFlight {
		t.Fatal("expected auto-refresh to resume after leaving modal input mode")
	}
	m = applyAutoRefreshLoadResult(t, m, cmd)
	if len(m.tasks) != 2 {
		t.Fatalf("expected deferred external task to appear after board-mode refresh, got %d tasks", len(m.tasks))
	}
}

// TestModelAutoRefreshTickPreservesFocusedSubtree verifies focused subtree projections refresh without losing focus.
func TestModelAutoRefreshTickPreservesFocusedSubtree(t *testing.T) {
	now := time.Date(2026, 2, 28, 11, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	parent, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-parent",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Parent",
		Priority:  domain.PriorityMedium,
	}, now)
	child, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-child-existing",
		ProjectID: p.ID,
		ParentID:  parent.ID,
		ColumnID:  c.ID,
		Position:  1,
		Title:     "Child Existing",
		Priority:  domain.PriorityLow,
	}, now.Add(time.Minute))
	other, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-other-existing",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  2,
		Title:     "Other Existing",
		Priority:  domain.PriorityLow,
	}, now.Add(2*time.Minute))
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{parent, child, other})
	m := loadReadyModel(t, NewModel(svc))
	m.autoRefreshInterval = time.Second
	m.focusTaskByID(parent.ID)
	m = applyMsg(t, m, keyRune('f'))
	if m.projectionRootTaskID != parent.ID {
		t.Fatalf("expected subtree focus root %q, got %q", parent.ID, m.projectionRootTaskID)
	}

	childExternal, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-child-external",
		ProjectID: p.ID,
		ParentID:  parent.ID,
		ColumnID:  c.ID,
		Position:  3,
		Title:     "Child External",
		Priority:  domain.PriorityMedium,
	}, now.Add(3*time.Minute))
	unrelatedExternal, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-unrelated-external",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  4,
		Title:     "Unrelated External",
		Priority:  domain.PriorityLow,
	}, now.Add(4*time.Minute))
	svc.tasks[p.ID] = append(svc.tasks[p.ID], childExternal, unrelatedExternal)

	m, cmd := applyAutoRefreshTickMsg(t, m)
	if !m.autoRefreshInFlight {
		t.Fatal("expected focused subtree refresh to start in board mode")
	}
	m = applyAutoRefreshLoadResult(t, m, cmd)
	if m.projectionRootTaskID != parent.ID {
		t.Fatalf("expected subtree focus root to remain %q, got %q", parent.ID, m.projectionRootTaskID)
	}
	if got := taskIDList(m.currentColumnTasks()); got != "t-child-existing,t-child-external" {
		t.Fatalf("expected focused subtree children only, got %q", got)
	}
}

// TestSortTaskSlicePrefersCreationTime verifies oldest-first ordering regardless of move position churn.
func TestSortTaskSlicePrefersCreationTime(t *testing.T) {
	now := time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC)
	older, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-older",
		ProjectID: "p1",
		ColumnID:  "c1",
		Position:  9,
		Title:     "Older",
		Priority:  domain.PriorityLow,
	}, now)
	newer, _ := domain.NewTask(domain.TaskInput{
		ID:        "t-newer",
		ProjectID: "p1",
		ColumnID:  "c1",
		Position:  0,
		Title:     "Newer",
		Priority:  domain.PriorityLow,
	}, now.Add(time.Minute))

	tasks := []domain.Task{newer, older}
	sortTaskSlice(tasks)
	if tasks[0].ID != older.ID {
		t.Fatalf("expected oldest task first, got %#v", tasks)
	}
}

// applyAutoRefreshTickMsg applies one auto-refresh tick and returns the updated model and resulting command.
func applyAutoRefreshTickMsg(t *testing.T, m Model) (Model, tea.Cmd) {
	t.Helper()
	updated, cmd := m.Update(autoRefreshTickMsg{})
	return mustModelValue(t, updated), cmd
}

// applyAutoRefreshLoadResult executes one auto-refresh load command and applies the returned message.
func applyAutoRefreshLoadResult(t *testing.T, m Model, cmd tea.Cmd) Model {
	t.Helper()
	if cmd == nil {
		t.Fatal("expected auto-refresh load command")
	}
	// Do not execute the follow-up command here to avoid spinning the recurring timer in tests.
	msg := cmd()
	updated, _ := m.Update(msg)
	return mustModelValue(t, updated)
}

// taskIDList returns comma-separated IDs for deterministic assertions.
func taskIDList(tasks []domain.Task) string {
	ids := make([]string, 0, len(tasks))
	for _, task := range tasks {
		ids = append(ids, strings.TrimSpace(task.ID))
	}
	return strings.Join(ids, ",")
}

// dependencyCandidateIndexByID finds one dependency-candidate index by task id.
func dependencyCandidateIndexByID(candidates []dependencyCandidate, taskID string) int {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return -1
	}
	for idx, candidate := range candidates {
		if strings.TrimSpace(candidate.Match.Task.ID) == taskID {
			return idx
		}
	}
	return -1
}

// ansiEscapePattern matches terminal color/style escape sequences for text assertions.
var ansiEscapePattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// stripANSI removes ANSI control codes from rendered output.
func stripANSI(in string) string {
	return ansiEscapePattern.ReplaceAllString(in, "")
}

// mustModelValue normalizes tea.Model results back into a concrete Model for test helpers.
func mustModelValue(t *testing.T, updated tea.Model) Model {
	t.Helper()
	switch typed := updated.(type) {
	case Model:
		return typed
	case *Model:
		if typed == nil {
			t.Fatal("expected non-nil *Model")
		}
		return *typed
	default:
		value := reflect.ValueOf(updated)
		if value.IsValid() && value.Kind() == reflect.Pointer && !value.IsNil() {
			if modelValue, ok := value.Elem().Interface().(Model); ok {
				return modelValue
			}
		}
		t.Fatalf("expected Model, got %T", updated)
		return Model{}
	}
}

// applyResult applies a model+command result tuple.
func applyResult(t *testing.T, updated tea.Model, cmd tea.Cmd) Model {
	t.Helper()
	return applyCmd(t, mustModelValue(t, updated), cmd)
}

// loadReadyModel loads data and accepts the initial launch picker when projects already exist.
func loadReadyModel(t *testing.T, m Model) Model {
	t.Helper()
	ready := applyMsg(t, applyCmd(t, m, m.Init()), tea.WindowSizeMsg{Width: 120, Height: 40})
	if ready.mode == modeProjectPicker && len(ready.projects) > 0 {
		ready = applyMsg(t, ready, tea.KeyPressMsg{Code: tea.KeyEnter})
	}
	return ready
}

// applyMsg applies msg.
func applyMsg(t *testing.T, m Model, msg tea.Msg) Model {
	t.Helper()
	updated, cmd := m.Update(msg)
	return applyCmd(t, mustModelValue(t, updated), cmd)
}

// applyCmd applies cmd.
func applyCmd(t *testing.T, m Model, cmd tea.Cmd) Model {
	t.Helper()
	out := m
	currentCmd := cmd
	for i := 0; i < 6 && currentCmd != nil; i++ {
		msgCh := make(chan tea.Msg, 1)
		// Some bubbles commands schedule timer-driven follow-ups such as cursor
		// blinking. Tests only need immediate commands, so bail out when a command
		// does not yield promptly instead of blocking the helper.
		go func(run tea.Cmd) {
			msgCh <- run()
		}(currentCmd)

		var msg tea.Msg
		select {
		case msg = <-msgCh:
		case <-time.After(10 * time.Millisecond):
			return out
		}
		updated, nextCmd := out.Update(msg)
		out = mustModelValue(t, updated)
		currentCmd = nextCmd
	}
	return out
}

// keyRune handles key rune.
func keyRune(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: r, Text: string(r)}
}

// TestNormalizeAttachmentPathWithinRoot verifies root-bound attachment validation behavior.
func TestNormalizeAttachmentPathWithinRoot(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "docs", "readme.md")
	if err := os.MkdirAll(filepath.Dir(inside), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(inside, []byte("ok"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	outsideDir := t.TempDir()
	outside := filepath.Join(outsideDir, "outside.md")
	if err := os.WriteFile(outside, []byte("bad"), 0o644); err != nil {
		t.Fatalf("WriteFile(outside) error = %v", err)
	}

	normalizedInside, err := normalizeAttachmentPathWithinRoot(root, inside)
	if err != nil {
		t.Fatalf("normalizeAttachmentPathWithinRoot(inside) error = %v", err)
	}
	if filepath.Clean(normalizedInside) != filepath.Clean(inside) {
		t.Fatalf("expected normalized inside path %q, got %q", inside, normalizedInside)
	}

	if _, err := normalizeAttachmentPathWithinRoot(root, outside); err == nil {
		t.Fatal("expected outside-root path to be rejected")
	}
	if _, err := normalizeAttachmentPathWithinRoot("", inside); err == nil {
		t.Fatal("expected empty-root attachment normalization to fail")
	}
	rootFile := filepath.Join(t.TempDir(), "root.txt")
	if err := os.WriteFile(rootFile, []byte("root"), 0o644); err != nil {
		t.Fatalf("WriteFile(rootFile) error = %v", err)
	}
	if _, err := normalizeAttachmentPathWithinRoot(rootFile, inside); err == nil {
		t.Fatal("expected non-directory root to be rejected")
	}
}

// TestTaskInfoBodyLinesRenderSystemSection verifies task info exposes structural/system fields intentionally.
func TestTaskInfoBodyLinesRenderSystemSection(t *testing.T) {
	now := time.Date(2026, 3, 13, 9, 30, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:             "t1",
		ProjectID:      p.ID,
		ColumnID:       c.ID,
		Position:       4,
		Kind:           domain.WorkKindTask,
		Scope:          domain.KindAppliesToTask,
		Title:          "Task",
		Priority:       domain.PriorityMedium,
		CreatedByActor: "user-a",
		UpdatedByActor: "agent-b",
		UpdatedByType:  domain.ActorTypeAgent,
	}, now)
	started := now.Add(2 * time.Hour)
	task.StartedAt = &started
	task.UpdatedAt = now.Add(4 * time.Hour)

	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})))
	lines := m.taskInfoBodyLines(task, taskInfoOverlayBoxWidth(max(0, m.fullPageNodeContentWidth())), 72, lipgloss.NewStyle())
	rendered := strings.Join(lines, "\n")

	for _, want := range []string{
		"system:",
		"id: t1",
		"project: p1",
		"parent: -",
		"kind: task",
		"scope: task",
		"state: todo",
		"column: c1",
		"position: 4",
		"created_by: user-a",
		"updated_by: agent-b (agent)",
		"started_at:",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected task info system section to contain %q, got\n%s", want, rendered)
		}
	}
}

// TestTaskInfoBodyLinesRenderSystemSectionUsesReadableActorNames verifies system ownership lines reuse activity display names.
func TestTaskInfoBodyLinesRenderSystemSectionUsesReadableActorNames(t *testing.T) {
	now := time.Date(2026, 3, 17, 18, 34, 10, 0, time.FixedZone("PDT", -7*60*60))
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:             "t1",
		ProjectID:      p.ID,
		ColumnID:       c.ID,
		Position:       3,
		Kind:           domain.WorkKindTask,
		Scope:          domain.KindAppliesToTask,
		Title:          "Task",
		Priority:       domain.PriorityMedium,
		CreatedByActor: "c75a483e-6628-475e-b12d-9ee7a928a9d1",
		UpdatedByActor: "agent-instance-1",
		UpdatedByType:  domain.ActorTypeAgent,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})
	svc.changeEvents[p.ID] = []domain.ChangeEvent{
		{
			ID:         1,
			ProjectID:  p.ID,
			WorkItemID: task.ID,
			Operation:  domain.ChangeOperationCreate,
			ActorID:    "c75a483e-6628-475e-b12d-9ee7a928a9d1",
			ActorName:  "Evan",
			ActorType:  domain.ActorTypeUser,
			Metadata:   map[string]string{"title": task.Title, "item_scope": "task"},
			OccurredAt: now,
		},
		{
			ID:         2,
			ProjectID:  p.ID,
			WorkItemID: task.ID,
			Operation:  domain.ChangeOperationUpdate,
			ActorID:    "agent-instance-1",
			ActorName:  "Codex Orchestrator",
			ActorType:  domain.ActorTypeAgent,
			Metadata:   map[string]string{"title": task.Title, "item_scope": "task"},
			OccurredAt: now.Add(2 * time.Minute),
		},
	}

	m := loadReadyModel(t, NewModel(
		svc,
		WithIdentityConfig(IdentityConfig{
			ActorID:          "c75a483e-6628-475e-b12d-9ee7a928a9d1",
			DisplayName:      "Evan",
			DefaultActorType: "user",
		}),
	))
	lines := m.taskInfoBodyLines(task, taskInfoOverlayBoxWidth(max(0, m.fullPageNodeContentWidth())), 72, lipgloss.NewStyle())
	rendered := strings.Join(lines, "\n")

	for _, want := range []string{
		"created_by: Evan (user)",
		"updated_by: Codex Orchestrator (agent)",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected task info system section to contain %q, got\n%s", want, rendered)
		}
	}
	if strings.Contains(rendered, "created_by: c75a483e-6628-475e-b12d-9ee7a928a9d1") {
		t.Fatalf("expected system section to avoid raw local actor id, got\n%s", rendered)
	}
}

// TestTaskSchemaCoverageIsExplicit verifies every top-level task field is intentionally editable or read-only in the TUI.
func TestTaskSchemaCoverageIsExplicit(t *testing.T) {
	editable := map[string]struct{}{
		"Title":       {},
		"Description": {},
		"Priority":    {},
		"DueAt":       {},
		"Labels":      {},
		"Metadata":    {},
	}
	readOnly := map[string]struct{}{
		"ID":             {},
		"ProjectID":      {},
		"ParentID":       {},
		"Kind":           {},
		"Scope":          {},
		"LifecycleState": {},
		"ColumnID":       {},
		"Position":       {},
		"CreatedByActor": {},
		"CreatedByName":  {},
		"UpdatedByActor": {},
		"UpdatedByName":  {},
		"UpdatedByType":  {},
		"CreatedAt":      {},
		"UpdatedAt":      {},
		"StartedAt":      {},
		"CompletedAt":    {},
		"ArchivedAt":     {},
		"CanceledAt":     {},
	}
	assertExplicitFieldCoverage(t, reflect.TypeOf(domain.Task{}), editable, readOnly, nil)
}

// TestTaskMetadataSchemaCoverageIsExplicit verifies supported vs intentionally unsupported metadata fields stay documented.
func TestTaskMetadataSchemaCoverageIsExplicit(t *testing.T) {
	editable := map[string]struct{}{
		"Objective":          {},
		"AcceptanceCriteria": {},
		"ValidationPlan":     {},
		"BlockedReason":      {},
		"RiskNotes":          {},
		"DependsOn":          {},
		"BlockedBy":          {},
		"ResourceRefs":       {},
	}
	readOnly := map[string]struct{}{
		"CompletionContract": {},
	}
	internal := map[string]struct{}{
		"ImplementationNotesUser":  {},
		"ImplementationNotesAgent": {},
		"DefinitionOfDone":         {},
		"CommandSnippets":          {},
		"ExpectedOutputs":          {},
		"DecisionLog":              {},
		"RelatedItems":             {},
		"TransitionNotes":          {},
		"ContextBlocks":            {},
		"KindPayload":              {},
	}
	assertExplicitFieldCoverage(t, reflect.TypeOf(domain.TaskMetadata{}), editable, readOnly, internal)
}

// TestProjectSchemaCoverageIsExplicit verifies project metadata support remains an intentional contract.
func TestProjectSchemaCoverageIsExplicit(t *testing.T) {
	editable := map[string]struct{}{
		"Name":        {},
		"Description": {},
		"Metadata":    {},
	}
	readOnly := map[string]struct{}{
		"ID":         {},
		"Slug":       {},
		"Kind":       {},
		"CreatedAt":  {},
		"UpdatedAt":  {},
		"ArchivedAt": {},
	}
	assertExplicitFieldCoverage(t, reflect.TypeOf(domain.Project{}), editable, readOnly, nil)

	projectMetadataEditable := map[string]struct{}{
		"Owner":    {},
		"Icon":     {},
		"Color":    {},
		"Homepage": {},
		"Tags":     {},
	}
	projectMetadataInternal := map[string]struct{}{
		"StandardsMarkdown": {},
		"KindPayload":       {},
		"CapabilityPolicy":  {},
	}
	assertExplicitFieldCoverage(t, reflect.TypeOf(domain.ProjectMetadata{}), projectMetadataEditable, nil, projectMetadataInternal)
}

// TestProjectFormBodyLinesRenderSystemSectionWhenEditing verifies project edit surfaces expose structural read-only fields.
func TestProjectFormBodyLinesRenderSystemSectionWhenEditing(t *testing.T) {
	now := time.Date(2026, 3, 13, 10, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	p.Kind = "ops"
	m := loadReadyModel(t, NewModel(newFakeService([]domain.Project{p}, nil, nil)))
	_ = m.startProjectForm(&p)
	lines, _ := m.projectFormBodyLines(72, lipgloss.NewStyle(), lipgloss.Color("62"))
	rendered := strings.Join(lines, "\n")

	for _, want := range []string{
		"system:",
		"id: p1",
		"slug: inbox",
		"kind: ops",
		"created_at:",
		"updated_at:",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected project edit system section to contain %q, got\n%s", want, rendered)
		}
	}
}

// TestFullPageSurfaceMetricsIgnoreGlobalStatusHeight verifies transient global status text cannot shrink shared full-page bodies.
func TestFullPageSurfaceMetricsIgnoreGlobalStatusHeight(t *testing.T) {
	m := NewModel(newFakeService(nil, nil, nil))
	m.width = 120
	m.height = 40
	accent := lipgloss.Color("62")
	muted := lipgloss.Color("241")
	dim := lipgloss.Color("239")

	m.status = ""
	base := m.fullPageSurfaceMetrics(accent, muted, dim, 96, "Edit Task", "kind: task", "")
	m.status = "cancelled"
	cancelled := m.fullPageSurfaceMetrics(accent, muted, dim, 96, "Edit Task", "kind: task", "")
	if base.bodyHeight != cancelled.bodyHeight {
		t.Fatalf("expected shared full-page body height to ignore status text, base=%d cancelled=%d", base.bodyHeight, cancelled.bodyHeight)
	}
}

// TestFullPageSurfaceMetricsShrinkBodyToFitShortTerminal verifies shared full-page surfaces do not force the body back to the default minimum on short terminals.
func TestFullPageSurfaceMetricsShrinkBodyToFitShortTerminal(t *testing.T) {
	m := NewModel(newFakeService(nil, nil, nil))
	m.width = 120
	m.height = 16
	accent := lipgloss.Color("62")
	muted := lipgloss.Color("241")
	dim := lipgloss.Color("239")

	metrics := m.fullPageSurfaceMetrics(accent, muted, dim, 96, "Edit Task", "kind: task", "")
	if metrics.bodyHeight >= taskInfoBodyViewportMinHeight {
		t.Fatalf("expected shared full-page body height to shrink below the default minimum on short terminals, got %d", metrics.bodyHeight)
	}

	body := viewport.New()
	body.SetWidth(metrics.contentWidth)
	body.SetHeight(metrics.bodyHeight)
	body.SetContent(strings.Repeat("line\n", 40))
	surface := renderFullPageSurfaceViewport(accent, muted, metrics.boxWidth, "Edit Task", "kind: task", "", body)
	totalHeight := lipgloss.Height(metrics.headerBlock) +
		metrics.headerGapY +
		metrics.topGapY +
		lipgloss.Height(surface) +
		metrics.bottomGapY +
		lipgloss.Height(metrics.helpLine)
	if totalHeight > m.height {
		t.Fatalf("expected shared full-page chrome to fit terminal, got %d want <= %d", totalHeight, m.height)
	}
}

// assertExplicitFieldCoverage ensures every exported struct field is classified as editable, read-only, or internal.
func assertExplicitFieldCoverage(
	t *testing.T,
	typ reflect.Type,
	editable map[string]struct{},
	readOnly map[string]struct{},
	internal map[string]struct{},
) {
	t.Helper()

	classified := map[string]string{}
	record := func(kind string, fields map[string]struct{}) {
		for field := range fields {
			if existing, ok := classified[field]; ok {
				t.Fatalf("%s field %q already classified as %s", typ.Name(), field, existing)
			}
			classified[field] = kind
		}
	}
	record("editable", editable)
	record("read-only", readOnly)
	record("internal", internal)

	for idx := 0; idx < typ.NumField(); idx++ {
		field := typ.Field(idx)
		if !field.IsExported() {
			continue
		}
		if _, ok := classified[field.Name]; !ok {
			t.Fatalf("%s field %q is not classified for TUI/schema coverage", typ.Name(), field.Name)
		}
	}
}
