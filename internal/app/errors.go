package app

import "errors"

// ErrNotFound and related errors describe validation and runtime failures.
var (
	ErrNotFound           = errors.New("not found")
	ErrInvalidDeleteMode  = errors.New("invalid delete mode")
	ErrEmbeddingClaimLost = errors.New("embedding claim lost")
	ErrEmbeddingsDisabled = errors.New("embeddings disabled")
	// ErrPathsDirty is returned by Service.CreateActionItem when one or more
	// of the paths declared on CreateActionItemInput.Paths show uncommitted
	// changes in the project's RepoPrimaryWorktree at the time of creation.
	// The wrapped error message names every dirty path so the caller can
	// surface them to the dev. Callers detect the rejection via
	// errors.Is(err, ErrPathsDirty). Per droplet 4b.6 acceptance criteria
	// the check is always-on; bypass requires the post-MVP supersede CLI.
	ErrPathsDirty = errors.New("declared paths have uncommitted changes")
	// ErrGitNotFound is returned by the git-status pre-check when the `git`
	// binary is not on PATH for the running process. It is returned
	// distinct from ErrPathsDirty so callers can distinguish "your tree is
	// dirty, fix it" from "your environment is mis-configured, install
	// git". Wraps exec.ErrNotFound so errors.Is(err, exec.ErrNotFound)
	// also holds.
	ErrGitNotFound = errors.New("git binary not found on PATH")
)
