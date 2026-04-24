package app

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// inboxMentionPattern matches role-style @mentions in markdown text.
var inboxMentionPattern = regexp.MustCompile(`(^|[^[:alnum:]_])@([[:alpha:]][[:alnum:]_-]*)`)

// syncCommentInboxAttention materializes routed comment mentions into attention rows.
func (s *Service) syncCommentInboxAttention(ctx context.Context, comment domain.Comment) error {
	if s == nil || s.repo == nil {
		return nil
	}
	level, ok := commentAttentionLevel(comment)
	if !ok {
		return nil
	}
	for _, role := range mentionedInboxRoles(comment.Summary, comment.BodyMarkdown) {
		item, err := domain.NewAttentionItem(domain.AttentionItemInput{
			ID:                 commentMentionAttentionID(comment.ID, role),
			ProjectID:          level.ProjectID,
			BranchID:           level.BranchID,
			ScopeType:          level.ScopeType,
			ScopeID:            level.ScopeID,
			Kind:               domain.AttentionKindMention,
			Summary:            fmt.Sprintf("mention for %s: %s", role, strings.TrimSpace(comment.Summary)),
			BodyMarkdown:       commentMentionAttentionBody(comment, role),
			TargetRole:         role,
			RequiresUserAction: false,
			CreatedByActor:     comment.ActorID,
			CreatedByType:      comment.ActorType,
		}, comment.CreatedAt)
		if err != nil {
			return err
		}
		if err := s.repo.CreateAttentionItem(ctx, item); err != nil {
			return err
		}
	}
	return nil
}

// syncHandoffInboxAttention mirrors one handoff into the routed attention inbox.
func (s *Service) syncHandoffInboxAttention(ctx context.Context, handoff domain.Handoff) error {
	if s == nil || s.repo == nil {
		return nil
	}
	item, err := handoffInboxAttention(handoff)
	if err != nil {
		return err
	}
	return s.repo.UpsertAttentionItem(ctx, item)
}

// commentAttentionLevel maps one comment target to the matching attention scope tuple.
func commentAttentionLevel(comment domain.Comment) (domain.LevelTupleInput, bool) {
	level := domain.LevelTupleInput{
		ProjectID: strings.TrimSpace(comment.ProjectID),
	}
	switch comment.TargetType {
	case domain.CommentTargetTypeProject:
		level.ScopeType = domain.ScopeLevelProject
		level.ScopeID = strings.TrimSpace(comment.TargetID)
	case domain.CommentTargetTypeActionItem:
		level.ScopeType = domain.ScopeLevelActionItem
		level.ScopeID = strings.TrimSpace(comment.TargetID)
	default:
		return domain.LevelTupleInput{}, false
	}
	return level, true
}

// mentionedInboxRoles returns stable, deduped routed roles mentioned in the supplied markdown text.
func mentionedInboxRoles(parts ...string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0)
	for _, part := range parts {
		for _, match := range inboxMentionPattern.FindAllStringSubmatch(part, -1) {
			if len(match) < 3 {
				continue
			}
			role := normalizeInboxRole(match[2])
			if !isInboxRole(role) {
				continue
			}
			if _, ok := seen[role]; ok {
				continue
			}
			seen[role] = struct{}{}
			out = append(out, role)
		}
	}
	slices.Sort(out)
	return out
}

// commentMentionAttentionID returns the stable attention identifier for one mentioned role on one comment.
func commentMentionAttentionID(commentID, role string) string {
	return strings.TrimSpace(commentID) + "::mention::" + normalizeInboxRole(role)
}

// commentMentionAttentionBody renders one markdown-rich inbox row body for a routed comment mention.
func commentMentionAttentionBody(comment domain.Comment, role string) string {
	lines := []string{
		fmt.Sprintf("Mentioned role: `%s`", normalizeInboxRole(role)),
		fmt.Sprintf("Comment by `%s` (`%s`)", firstNonEmptyTrimmed(comment.ActorName, comment.ActorID), comment.ActorType),
		fmt.Sprintf("Target: `%s/%s`", comment.TargetType, comment.TargetID),
		"",
		strings.TrimSpace(comment.BodyMarkdown),
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// handoffInboxAttention builds one routed inbox attention row for a durable handoff.
func handoffInboxAttention(handoff domain.Handoff) (domain.AttentionItem, error) {
	level := handoffAttentionLevel(handoff)
	role := normalizeInboxRole(handoff.TargetRole)
	state := domain.AttentionStateOpen
	requiresUserAction := true
	if role == "" || handoff.IsTerminal() {
		state = domain.AttentionStateResolved
		requiresUserAction = false
	}
	return domain.NewAttentionItem(domain.AttentionItemInput{
		ID:                 handoffInboxAttentionID(handoff.ID),
		ProjectID:          level.ProjectID,
		BranchID:           level.BranchID,
		ScopeType:          level.ScopeType,
		ScopeID:            level.ScopeID,
		State:              state,
		Kind:               domain.AttentionKindHandoff,
		Summary:            handoffInboxSummary(handoff, role),
		BodyMarkdown:       handoffInboxBody(handoff, role),
		TargetRole:         role,
		RequiresUserAction: requiresUserAction,
		CreatedByActor:     firstNonEmptyTrimmed(handoff.UpdatedByActor, handoff.CreatedByActor),
		CreatedByType:      firstNonEmptyActorType(handoff.UpdatedByType, handoff.CreatedByType),
	}, handoff.UpdatedAt)
}

// handoffAttentionLevel resolves the attention scope that should open when one inbox handoff row is selected.
func handoffAttentionLevel(handoff domain.Handoff) domain.LevelTupleInput {
	level := domain.LevelTupleInput{
		ProjectID: strings.TrimSpace(handoff.ProjectID),
		BranchID:  strings.TrimSpace(handoff.BranchID),
		ScopeType: handoff.ScopeType,
		ScopeID:   strings.TrimSpace(handoff.ScopeID),
	}
	if strings.TrimSpace(handoff.TargetScopeID) != "" && handoff.TargetScopeType != "" {
		level.BranchID = firstNonEmptyTrimmed(handoff.TargetBranchID, level.BranchID)
		level.ScopeType = handoff.TargetScopeType
		level.ScopeID = strings.TrimSpace(handoff.TargetScopeID)
		return level
	}
	if strings.TrimSpace(handoff.TargetBranchID) != "" {
		level.BranchID = strings.TrimSpace(handoff.TargetBranchID)
		level.ScopeType = domain.ScopeLevelBranch
		level.ScopeID = level.BranchID
	}
	return level
}

// handoffInboxAttentionID returns the stable mirrored attention identifier for one handoff.
func handoffInboxAttentionID(handoffID string) string {
	return strings.TrimSpace(handoffID) + "::handoff"
}

// handoffInboxSummary renders one concise inbox summary for the routed handoff row.
func handoffInboxSummary(handoff domain.Handoff, role string) string {
	role = normalizeInboxRole(role)
	if role == "" {
		role = "inbox"
	}
	return fmt.Sprintf("handoff for %s: %s", role, strings.TrimSpace(handoff.Summary))
}

// handoffInboxBody renders one markdown-rich inbox body for the routed handoff row.
func handoffInboxBody(handoff domain.Handoff, role string) string {
	lines := []string{
		fmt.Sprintf("Target role: `%s`", firstNonEmptyTrimmed(normalizeInboxRole(role), "-")),
		fmt.Sprintf("Source role: `%s`", firstNonEmptyTrimmed(strings.TrimSpace(handoff.SourceRole), "-")),
		fmt.Sprintf("Status: `%s`", handoff.Status),
	}
	if nextAction := strings.TrimSpace(handoff.NextAction); nextAction != "" {
		lines = append(lines, "", "Next action:", nextAction)
	}
	if len(handoff.MissingEvidence) > 0 {
		lines = append(lines, "", "Missing evidence:")
		for _, item := range handoff.MissingEvidence {
			lines = append(lines, "- "+item)
		}
	}
	if len(handoff.RelatedRefs) > 0 {
		lines = append(lines, "", "Related refs:")
		for _, item := range handoff.RelatedRefs {
			lines = append(lines, "- "+item)
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// normalizeInboxRole canonicalizes inbox-facing role labels and common aliases.
func normalizeInboxRole(raw string) string {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "dev":
		return "builder"
	case "researcher":
		return "research"
	default:
		return strings.TrimSpace(strings.ToLower(raw))
	}
}

// isInboxRole reports whether one normalized role is currently routable through the inbox model.
func isInboxRole(role string) bool {
	switch normalizeInboxRole(role) {
	case "builder", "qa", "orchestrator", "research", "human":
		return true
	default:
		return false
	}
}

// firstNonEmptyActorType returns the first non-empty actor type.
func firstNonEmptyActorType(values ...domain.ActorType) domain.ActorType {
	for _, value := range values {
		if strings.TrimSpace(string(value)) != "" {
			return value
		}
	}
	return domain.ActorTypeUser
}
