package app

import "errors"

// ErrNotFound and related errors describe validation and runtime failures.
var (
	ErrNotFound           = errors.New("not found")
	ErrInvalidDeleteMode  = errors.New("invalid delete mode")
	ErrEmbeddingClaimLost = errors.New("embedding claim lost")
	ErrEmbeddingsDisabled = errors.New("embeddings disabled")
)
