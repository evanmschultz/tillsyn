package dispatcher

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ErrEmptyWorktreeRoot is returned by PersistRunTrace when worktreeRoot is empty.
var ErrEmptyWorktreeRoot = errors.New("dispatcher: worktreeRoot is empty")

// RunTraceMeta is the JSON envelope payload persisted to <run>.meta.json,
// mirroring the proven agent-dispatch.sh audit block fields.
type RunTraceMeta struct {
	// Run is the run base name, e.g. "20260526-215051-ta-go-builder-e2spawn123".
	Run string `json:"run"`

	// ActionItemID is the domain.ActionItem.ID this dispatch was for.
	ActionItemID string `json:"action_item_id"`

	// AgentName is the canonical agent identifier (e.g. "go-builder-agent").
	AgentName string `json:"agent_name"`

	// CLIKind is the adapter family ("claude" or "codex").
	CLIKind string `json:"cli_kind"`

	// Outcome is the dispatch outcome ("success", "failed", "blocked", etc.).
	Outcome string `json:"outcome"`

	// StartedAt is the wall-clock time the dispatch began.
	StartedAt time.Time `json:"started_at"`

	// TerminatedAt is the wall-clock time the dispatch ended.
	TerminatedAt time.Time `json:"terminated_at"`
}

// RunTracePaths is the set of absolute paths written by PersistRunTrace.
type RunTracePaths struct {
	// Out is the absolute path of the <run>.out file (stdout).
	Out string

	// Err is the absolute path of the <run>.err file (stderr).
	Err string

	// Meta is the absolute path of the <run>.meta.json file (envelope).
	Meta string
}

// PersistRunTrace writes the full dispatch trace to .claude/agent-runs/<run>.{out,err,meta.json},
// mirroring the proven agent-dispatch.sh audit-capture block. The <run> base is derived from
// RunTraceMeta.Run; the directory is created with os.MkdirAll(0o755) if absent.
//
// Returns RunTracePaths with the three absolute paths on success, or an error if:
//   - worktreeRoot is empty (ErrEmptyWorktreeRoot)
//   - directory creation fails
//   - file write fails
//
// The function is idempotent if called with the same <run> base name — subsequent
// calls overwrite the prior files.
func PersistRunTrace(worktreeRoot string, meta RunTraceMeta, stdout, stderr []byte) (RunTracePaths, error) {
	if worktreeRoot == "" {
		return RunTracePaths{}, ErrEmptyWorktreeRoot
	}

	auditDir := filepath.Join(worktreeRoot, ".claude", "agent-runs")
	if err := os.MkdirAll(auditDir, 0o755); err != nil {
		return RunTracePaths{}, fmt.Errorf("dispatcher: create audit dir: %w", err)
	}

	outPath := filepath.Join(auditDir, meta.Run+".out")
	errPath := filepath.Join(auditDir, meta.Run+".err")
	metaPath := filepath.Join(auditDir, meta.Run+".meta.json")

	// Write stdout to <run>.out
	if err := os.WriteFile(outPath, stdout, 0o644); err != nil {
		return RunTracePaths{}, fmt.Errorf("dispatcher: write stdout trace: %w", err)
	}

	// Write stderr to <run>.err
	if err := os.WriteFile(errPath, stderr, 0o644); err != nil {
		return RunTracePaths{}, fmt.Errorf("dispatcher: write stderr trace: %w", err)
	}

	// Marshal and write metadata to <run>.meta.json
	metaBytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return RunTracePaths{}, fmt.Errorf("dispatcher: marshal metadata: %w", err)
	}
	if err := os.WriteFile(metaPath, metaBytes, 0o644); err != nil {
		return RunTracePaths{}, fmt.Errorf("dispatcher: write metadata trace: %w", err)
	}

	return RunTracePaths{
		Out:  outPath,
		Err:  errPath,
		Meta: metaPath,
	}, nil
}
