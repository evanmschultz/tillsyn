package dispatcher_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/app/dispatcher"
)

func TestPersistRunTrace(t *testing.T) {
	tests := []struct {
		name      string
		worktree  string
		meta      dispatcher.RunTraceMeta
		stdout    []byte
		stderr    []byte
		wantError bool
		check     func(t *testing.T, paths dispatcher.RunTracePaths, worktree string)
	}{
		{
			name:     "happy path: all three files exist with correct contents",
			worktree: "", // will be populated by t.TempDir()
			meta: dispatcher.RunTraceMeta{
				Run:          "20260526-215051-ta-go-builder-e2",
				ActionItemID: "c721e1ee-992f-4b65-a22d-9af47962f30a",
				AgentName:    "E2_BUILDER",
				CLIKind:      "claude",
				Outcome:      "success",
				StartedAt:    time.Date(2026, 5, 26, 21, 50, 51, 0, time.UTC),
				TerminatedAt: time.Date(2026, 5, 26, 21, 51, 0, 0, time.UTC),
			},
			stdout: []byte("dispatch stdout line 1\ndispatch stdout line 2"),
			stderr: []byte("dispatch stderr diagnostic"),
			check: func(t *testing.T, paths dispatcher.RunTracePaths, worktree string) {
				// Verify all three paths are under the expected audit directory.
				auditDir := filepath.Join(worktree, ".claude", "agent-runs")
				if !filepath.HasPrefix(paths.Out, auditDir) {
					t.Errorf("Out path %q not under %q", paths.Out, auditDir)
				}
				if !filepath.HasPrefix(paths.Err, auditDir) {
					t.Errorf("Err path %q not under %q", paths.Err, auditDir)
				}
				if !filepath.HasPrefix(paths.Meta, auditDir) {
					t.Errorf("Meta path %q not under %q", paths.Meta, auditDir)
				}

				// Verify files exist and have expected contents.
				outBytes, err := os.ReadFile(paths.Out)
				if err != nil {
					t.Fatalf("read out file: %v", err)
				}
				if string(outBytes) != "dispatch stdout line 1\ndispatch stdout line 2" {
					t.Errorf("stdout contents mismatch: got %q", outBytes)
				}

				errBytes, err := os.ReadFile(paths.Err)
				if err != nil {
					t.Fatalf("read err file: %v", err)
				}
				if string(errBytes) != "dispatch stderr diagnostic" {
					t.Errorf("stderr contents mismatch: got %q", errBytes)
				}

				// Verify metadata JSON round-trips and fields match.
				metaBytes, err := os.ReadFile(paths.Meta)
				if err != nil {
					t.Fatalf("read meta file: %v", err)
				}

				var recovered dispatcher.RunTraceMeta
				if err := json.Unmarshal(metaBytes, &recovered); err != nil {
					t.Fatalf("unmarshal metadata: %v", err)
				}

				if recovered.Run != "20260526-215051-ta-go-builder-e2" {
					t.Errorf("Run mismatch: got %q", recovered.Run)
				}
				if recovered.ActionItemID != "c721e1ee-992f-4b65-a22d-9af47962f30a" {
					t.Errorf("ActionItemID mismatch: got %q", recovered.ActionItemID)
				}
				if recovered.AgentName != "E2_BUILDER" {
					t.Errorf("AgentName mismatch: got %q", recovered.AgentName)
				}
				if recovered.CLIKind != "claude" {
					t.Errorf("CLIKind mismatch: got %q", recovered.CLIKind)
				}
				if recovered.Outcome != "success" {
					t.Errorf("Outcome mismatch: got %q", recovered.Outcome)
				}
				if !recovered.StartedAt.Equal(time.Date(2026, 5, 26, 21, 50, 51, 0, time.UTC)) {
					t.Errorf("StartedAt mismatch: got %v", recovered.StartedAt)
				}
				if !recovered.TerminatedAt.Equal(time.Date(2026, 5, 26, 21, 51, 0, 0, time.UTC)) {
					t.Errorf("TerminatedAt mismatch: got %v", recovered.TerminatedAt)
				}
			},
		},
		{
			name:     "directory auto-created when absent",
			worktree: "", // will be populated by t.TempDir()
			meta: dispatcher.RunTraceMeta{
				Run:          "20260526-215100-ta-go-qa-proof",
				ActionItemID: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
				AgentName:    "QA_PROOF",
				CLIKind:      "claude",
				Outcome:      "pass",
				StartedAt:    time.Now().UTC(),
				TerminatedAt: time.Now().UTC(),
			},
			stdout: []byte(""),
			stderr: []byte(""),
			check: func(t *testing.T, paths dispatcher.RunTracePaths, worktree string) {
				auditDir := filepath.Join(worktree, ".claude", "agent-runs")
				info, err := os.Stat(auditDir)
				if err != nil {
					t.Fatalf("stat audit dir: %v", err)
				}
				if !info.IsDir() {
					t.Errorf("audit dir is not a directory")
				}
			},
		},
		{
			name:      "empty worktreeRoot returns ErrEmptyWorktreeRoot",
			worktree:  "", // explicitly empty, not t.TempDir()
			meta:      dispatcher.RunTraceMeta{Run: "test-run"},
			wantError: true,
			check: func(t *testing.T, paths dispatcher.RunTracePaths, worktree string) {
				// Paths should be zero-valued on error.
				if paths.Out != "" || paths.Err != "" || paths.Meta != "" {
					t.Errorf("paths should be zero on error: %+v", paths)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worktree := tt.worktree
			if worktree == "" && !tt.wantError {
				worktree = t.TempDir()
			}

			paths, err := dispatcher.PersistRunTrace(worktree, tt.meta, tt.stdout, tt.stderr)
			if tt.wantError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}

			if tt.check != nil {
				tt.check(t, paths, worktree)
			}
		})
	}
}
