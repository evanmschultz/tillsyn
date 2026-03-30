package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/hylla/tillsyn/internal/domain"
)

// EmbeddingGenerator describes one embedding provider used by application search.
type EmbeddingGenerator interface {
	Embed(ctx context.Context, inputs []string) ([][]float32, error)
}

// EmbeddingSearchTargetType identifies the entity family a semantic document resolves to at query time.
type EmbeddingSearchTargetType string

// Supported semantic search target families.
const (
	EmbeddingSearchTargetTypeWorkItem EmbeddingSearchTargetType = "work_item"
	EmbeddingSearchTargetTypeProject  EmbeddingSearchTargetType = "project"
)

// EmbeddingSearchIndex stores and retrieves semantic documents for indexed subjects.
type EmbeddingSearchIndex interface {
	UpsertEmbeddingDocument(context.Context, EmbeddingDocument) error
	DeleteEmbeddingDocument(context.Context, EmbeddingSubjectType, string) error
	SearchEmbeddingDocuments(context.Context, EmbeddingSearchInput) ([]EmbeddingSearchMatch, error)
}

// EmbeddingDocument represents one persisted vectorized subject row.
type EmbeddingDocument struct {
	SubjectType      EmbeddingSubjectType
	SubjectID        string
	ProjectID        string
	SearchTargetType EmbeddingSearchTargetType
	SearchTargetID   string
	Content          string
	ContentHash      string
	Vector           []float32
	UpdatedAt        time.Time
}

// EmbeddingSearchInput represents semantic search query options for vectors.
type EmbeddingSearchInput struct {
	ProjectIDs        []string
	SubjectTypes      []EmbeddingSubjectType
	SearchTargetTypes []EmbeddingSearchTargetType
	Vector            []float32
	Limit             int
}

// EmbeddingSearchMatch represents one semantic similarity result.
type EmbeddingSearchMatch struct {
	SubjectType      EmbeddingSubjectType
	SubjectID        string
	SearchTargetType EmbeddingSearchTargetType
	SearchTargetID   string
	Similarity       float64
	SearchedAt       time.Time
}

// BuildThreadContextSubjectID encodes one comment target into a stable subject identifier.
func BuildThreadContextSubjectID(target domain.CommentTarget) string {
	return strings.Join([]string{
		url.QueryEscape(strings.TrimSpace(target.ProjectID)),
		url.QueryEscape(string(domain.NormalizeCommentTargetType(target.TargetType))),
		url.QueryEscape(strings.TrimSpace(target.TargetID)),
	}, "|")
}

func buildThreadContextSubjectID(target domain.CommentTarget) (string, error) {
	target, err := domain.NormalizeCommentTarget(target)
	if err != nil {
		return "", err
	}
	return BuildThreadContextSubjectID(target), nil
}

// ParseThreadContextSubjectID decodes one stable thread-context identifier back into a comment target.
func ParseThreadContextSubjectID(subjectID string) (domain.CommentTarget, error) {
	parts := strings.Split(strings.TrimSpace(subjectID), "|")
	if len(parts) != 3 {
		return domain.CommentTarget{}, fmt.Errorf("invalid thread context subject id %q: %w", subjectID, domain.ErrInvalidID)
	}
	projectID, err := url.QueryUnescape(parts[0])
	if err != nil {
		return domain.CommentTarget{}, fmt.Errorf("decode thread context project id: %w", err)
	}
	targetType, err := url.QueryUnescape(parts[1])
	if err != nil {
		return domain.CommentTarget{}, fmt.Errorf("decode thread context target type: %w", err)
	}
	targetID, err := url.QueryUnescape(parts[2])
	if err != nil {
		return domain.CommentTarget{}, fmt.Errorf("decode thread context target id: %w", err)
	}
	return domain.NormalizeCommentTarget(domain.CommentTarget{
		ProjectID:  projectID,
		TargetType: domain.CommentTargetType(targetType),
		TargetID:   targetID,
	})
}

func parseThreadContextSubjectID(subjectID string) (domain.CommentTarget, error) {
	return ParseThreadContextSubjectID(subjectID)
}

// EmbeddingSearchTargetForCommentTarget resolves one comment target into the search target family consumed by query ranking.
func EmbeddingSearchTargetForCommentTarget(target domain.CommentTarget) (EmbeddingSearchTargetType, string, error) {
	target, err := domain.NormalizeCommentTarget(target)
	if err != nil {
		return "", "", err
	}
	if target.TargetType == domain.CommentTargetTypeProject {
		return EmbeddingSearchTargetTypeProject, target.ProjectID, nil
	}
	return EmbeddingSearchTargetTypeWorkItem, target.TargetID, nil
}

func commentTargetEmbeddingSearchTarget(target domain.CommentTarget) (EmbeddingSearchTargetType, string, error) {
	return EmbeddingSearchTargetForCommentTarget(target)
}

// buildWorkItemEmbeddingContent produces canonical searchable text for one work item.
func buildWorkItemEmbeddingContent(task domain.Task) string {
	parts := make([]string, 0, 10)
	appendIfPresent := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		parts = append(parts, value)
	}
	appendIfPresent(task.Title)
	appendIfPresent(task.Description)
	if len(task.Labels) > 0 {
		appendIfPresent(strings.Join(task.Labels, ", "))
	}
	appendIfPresent(task.Metadata.Objective)
	appendIfPresent(task.Metadata.AcceptanceCriteria)
	appendIfPresent(task.Metadata.ValidationPlan)
	appendIfPresent(task.Metadata.BlockedReason)
	appendIfPresent(task.Metadata.RiskNotes)
	return strings.Join(parts, "\n")
}

func buildTaskEmbeddingContent(task domain.Task) string {
	return buildWorkItemEmbeddingContent(task)
}

// buildProjectDocumentEmbeddingContent produces canonical searchable text for project descriptive surfaces.
func buildProjectDocumentEmbeddingContent(project domain.Project) string {
	parts := make([]string, 0, 6)
	appendIfPresent := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		parts = append(parts, value)
	}
	appendIfPresent(project.Name)
	appendIfPresent(project.Description)
	if len(project.Metadata.Tags) > 0 {
		appendIfPresent(strings.Join(project.Metadata.Tags, ", "))
	}
	appendIfPresent(project.Metadata.StandardsMarkdown)
	return strings.Join(parts, "\n")
}

// buildThreadContextEmbeddingContent produces canonical searchable text for one threaded comment target.
func buildThreadContextEmbeddingContent(target domain.CommentTarget, targetTitle, targetBody string, comments []domain.Comment) string {
	parts := make([]string, 0, 5+(len(comments)*2))
	appendIfPresent := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		parts = append(parts, value)
	}
	appendIfPresent(string(domain.NormalizeCommentTargetType(target.TargetType)))
	appendIfPresent(targetTitle)
	appendIfPresent(targetBody)
	for _, comment := range comments {
		appendIfPresent(comment.Summary)
		appendIfPresent(comment.BodyMarkdown)
	}
	return strings.Join(parts, "\n")
}

// hashEmbeddingContent computes one deterministic hash for embedding payload changes.
func hashEmbeddingContent(content string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(content)))
	return hex.EncodeToString(sum[:])
}

// normalizeSearchWeights resolves configured lexical/semantic weights to stable defaults.
func normalizeSearchWeights(lexicalWeight, semanticWeight float64) (float64, float64) {
	if lexicalWeight <= 0 && semanticWeight <= 0 {
		return defaultSearchLexicalWeight, defaultSearchSemanticWeight
	}
	if lexicalWeight < 0 {
		lexicalWeight = 0
	}
	if semanticWeight < 0 {
		semanticWeight = 0
	}
	total := lexicalWeight + semanticWeight
	if total <= 0 {
		return defaultSearchLexicalWeight, defaultSearchSemanticWeight
	}
	return lexicalWeight / total, semanticWeight / total
}
