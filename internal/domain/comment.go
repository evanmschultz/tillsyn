package domain

import (
	"slices"
	"strings"
	"time"
)

// CommentTargetType identifies the entity type a comment belongs to. Comments
// now target either a project as a whole or a single action item; scope-level
// distinctions (branch/phase/subtask) were removed alongside the 12-kind
// action-item enum.
type CommentTargetType string

// Comment target type values.
const (
	CommentTargetTypeProject    CommentTargetType = "project"
	CommentTargetTypeActionItem CommentTargetType = "action_item"
)

// validCommentTargetTypes stores supported target-type values.
var validCommentTargetTypes = []CommentTargetType{
	CommentTargetTypeProject,
	CommentTargetTypeActionItem,
}

// CommentTarget identifies a concrete target within a project.
type CommentTarget struct {
	ProjectID  string
	TargetType CommentTargetType
	TargetID   string
}

// Comment stores an ownership-attributed note attached to a target.
type Comment struct {
	ID           string
	ProjectID    string
	TargetType   CommentTargetType
	TargetID     string
	Summary      string
	BodyMarkdown string
	ActorID      string
	ActorName    string
	ActorType    ActorType
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// CommentInput holds input values for comment creation operations.
type CommentInput struct {
	ID           string
	ProjectID    string
	TargetType   CommentTargetType
	TargetID     string
	Summary      string
	BodyMarkdown string
	ActorID      string
	ActorName    string
	ActorType    ActorType
}

// NewComment constructs a normalized comment.
func NewComment(in CommentInput, now time.Time) (Comment, error) {
	in.ID = strings.TrimSpace(in.ID)
	if in.ID == "" {
		return Comment{}, ErrInvalidID
	}

	target, err := NormalizeCommentTarget(CommentTarget{
		ProjectID:  in.ProjectID,
		TargetType: in.TargetType,
		TargetID:   in.TargetID,
	})
	if err != nil {
		return Comment{}, err
	}

	body := strings.TrimSpace(in.BodyMarkdown)
	if body == "" {
		return Comment{}, ErrInvalidBodyMarkdown
	}
	summary := NormalizeCommentSummary(in.Summary, body)
	if summary == "" {
		return Comment{}, ErrInvalidSummary
	}

	actorType := normalizeActorTypeValue(in.ActorType)
	if actorType == "" {
		actorType = ActorTypeUser
	}
	if !isValidActorType(actorType) {
		return Comment{}, ErrInvalidActorType
	}

	actorID := strings.TrimSpace(in.ActorID)
	if actorID == "" {
		actorID = "tillsyn-user"
	}

	actorName := strings.TrimSpace(in.ActorName)
	if actorName == "" {
		actorName = actorID
	}

	timestamp := now.UTC()
	return Comment{
		ID:           in.ID,
		ProjectID:    target.ProjectID,
		TargetType:   target.TargetType,
		TargetID:     target.TargetID,
		Summary:      summary,
		BodyMarkdown: body,
		ActorID:      actorID,
		ActorName:    actorName,
		ActorType:    actorType,
		CreatedAt:    timestamp,
		UpdatedAt:    timestamp,
	}, nil
}

// NormalizeCommentTarget validates and canonicalizes comment target identifiers.
func NormalizeCommentTarget(target CommentTarget) (CommentTarget, error) {
	target.ProjectID = strings.TrimSpace(target.ProjectID)
	target.TargetID = strings.TrimSpace(target.TargetID)
	target.TargetType = NormalizeCommentTargetType(target.TargetType)

	if target.ProjectID == "" {
		return CommentTarget{}, ErrInvalidID
	}
	if target.TargetID == "" {
		return CommentTarget{}, ErrInvalidTargetID
	}
	if !IsValidCommentTargetType(target.TargetType) {
		return CommentTarget{}, ErrInvalidTargetType
	}
	return target, nil
}

// NormalizeCommentTargetType canonicalizes target types to their stored form.
// Inputs are matched case-insensitively against the supported set and
// returned in their canonical form; unknown values are returned lowercased so
// callers can still detect invalid inputs.
func NormalizeCommentTargetType(targetType CommentTargetType) CommentTargetType {
	lowered := strings.TrimSpace(strings.ToLower(string(targetType)))
	if lowered == "" {
		return ""
	}
	for _, candidate := range validCommentTargetTypes {
		if strings.ToLower(string(candidate)) == lowered {
			return candidate
		}
	}
	return CommentTargetType(lowered)
}

// IsValidCommentTargetType reports whether the target type is supported.
func IsValidCommentTargetType(targetType CommentTargetType) bool {
	targetType = NormalizeCommentTargetType(targetType)
	return slices.Contains(validCommentTargetTypes, targetType)
}

// NormalizeCommentSummary trims the explicit summary and falls back to body markdown.
func NormalizeCommentSummary(summary, bodyMarkdown string) string {
	summary = strings.TrimSpace(summary)
	if summary != "" {
		return summary
	}
	return firstNonEmptyMarkdownLine(bodyMarkdown)
}

// firstNonEmptyMarkdownLine returns the first non-empty markdown line from body text.
func firstNonEmptyMarkdownLine(bodyMarkdown string) string {
	for _, line := range strings.Split(bodyMarkdown, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}

// normalizeActorTypeValue canonicalizes actor type values without applying defaults.
func normalizeActorTypeValue(actorType ActorType) ActorType {
	return ActorType(strings.TrimSpace(strings.ToLower(string(actorType))))
}
