package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/hylla/tillsyn/internal/domain"
)

// EmbeddingGenerator describes one embedding provider used by application search.
type EmbeddingGenerator interface {
	Embed(ctx context.Context, inputs []string) ([][]float32, error)
}

// TaskSearchIndex stores and retrieves task vectors for semantic retrieval.
type TaskSearchIndex interface {
	UpsertTaskEmbedding(context.Context, TaskEmbeddingDocument) error
	DeleteTaskEmbedding(context.Context, string) error
	SearchTaskEmbeddings(context.Context, TaskEmbeddingSearchInput) ([]TaskEmbeddingMatch, error)
}

// TaskEmbeddingDocument represents one persisted vectorized task document row.
type TaskEmbeddingDocument struct {
	TaskID      string
	ProjectID   string
	Content     string
	ContentHash string
	Vector      []float32
	UpdatedAt   time.Time
}

// TaskEmbeddingSearchInput represents semantic search query options for vectors.
type TaskEmbeddingSearchInput struct {
	ProjectIDs []string
	Vector     []float32
	Limit      int
}

// TaskEmbeddingMatch represents one semantic similarity result.
type TaskEmbeddingMatch struct {
	TaskID     string
	Similarity float64
	SearchedAt time.Time
}

// refreshTaskEmbedding best-effort updates one task embedding document after writes.
func (s *Service) refreshTaskEmbedding(ctx context.Context, task domain.Task) {
	if s == nil || s.embeddingGenerator == nil || s.searchIndex == nil {
		return
	}
	content := buildTaskEmbeddingContent(task)
	if strings.TrimSpace(content) == "" {
		if err := s.searchIndex.DeleteTaskEmbedding(ctx, task.ID); err != nil {
			log.Warn(
				"task embedding refresh drop failed",
				"task_id", task.ID,
				"project_id", task.ProjectID,
				"reason", "empty_content",
				"err", err,
			)
		}
		return
	}
	vectorRows, err := s.embeddingGenerator.Embed(ctx, []string{content})
	if err != nil {
		log.Warn(
			"task embedding refresh skipped: embed failed",
			"task_id", task.ID,
			"project_id", task.ProjectID,
			"content_hash", hashEmbeddingContent(content),
			"err", err,
		)
		return
	}
	if len(vectorRows) == 0 || len(vectorRows[0]) == 0 {
		log.Warn(
			"task embedding refresh skipped: empty embedding vector",
			"task_id", task.ID,
			"project_id", task.ProjectID,
			"vector_rows", len(vectorRows),
		)
		return
	}
	doc := TaskEmbeddingDocument{
		TaskID:      task.ID,
		ProjectID:   task.ProjectID,
		Content:     content,
		ContentHash: hashEmbeddingContent(content),
		Vector:      append([]float32(nil), vectorRows[0]...),
		UpdatedAt:   s.clock().UTC(),
	}
	if err := s.searchIndex.UpsertTaskEmbedding(ctx, doc); err != nil {
		log.Warn(
			"task embedding refresh upsert failed",
			"task_id", task.ID,
			"project_id", task.ProjectID,
			"content_hash", doc.ContentHash,
			"err", err,
		)
	}
}

// dropTaskEmbedding best-effort removes one task embedding row after hard delete.
func (s *Service) dropTaskEmbedding(ctx context.Context, taskID string) {
	if s == nil || s.searchIndex == nil {
		return
	}
	if err := s.searchIndex.DeleteTaskEmbedding(ctx, taskID); err != nil {
		log.Warn("task embedding drop failed", "task_id", taskID, "err", err)
	}
}

// buildTaskEmbeddingContent produces canonical searchable text for one task.
func buildTaskEmbeddingContent(task domain.Task) string {
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
