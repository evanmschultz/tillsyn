package domain

import (
	"strings"
	"testing"
	"time"
)

// TestActionItemMetadata_AppendSpawnHistory tests the AppendSpawnHistory
// method on *ActionItemMetadata. Table-driven test verifies: (a) append to
// nil slice yields len 1, (b) two appends with same (SpawnID, BundlePath)
// BOTH retained (no dedup), (c) time fields canonicalized to UTC, (d) string
// fields trimmed.
func TestActionItemMetadata_AppendSpawnHistory(t *testing.T) {
	tests := []struct {
		name        string
		initialMeta ActionItemMetadata
		entry       SpawnHistoryEntry
		wantLen     int
		description string
	}{
		{
			name:        "append to nil slice yields len 1",
			initialMeta: ActionItemMetadata{},
			entry: SpawnHistoryEntry{
				SpawnID:      "spawn-1",
				BundlePath:   "/tmp/spawn-1",
				StartedAt:    time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC),
				TerminatedAt: time.Date(2026, 5, 26, 12, 5, 0, 0, time.UTC),
				Outcome:      "success",
			},
			wantLen:     1,
			description: "initial append to empty metadata",
		},
		{
			name: "two appends with same id/bundle both retained (no dedup)",
			initialMeta: ActionItemMetadata{
				SpawnHistory: []SpawnHistoryEntry{
					{
						SpawnID:      "spawn-retry",
						BundlePath:   "/tmp/retry",
						StartedAt:    time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC),
						TerminatedAt: time.Date(2026, 5, 26, 12, 1, 0, 0, time.UTC),
						Outcome:      "failure",
					},
				},
			},
			entry: SpawnHistoryEntry{
				SpawnID:      "spawn-retry",
				BundlePath:   "/tmp/retry",
				StartedAt:    time.Date(2026, 5, 26, 12, 2, 0, 0, time.UTC),
				TerminatedAt: time.Date(2026, 5, 26, 12, 3, 0, 0, time.UTC),
				Outcome:      "success",
			},
			wantLen:     2,
			description: "both entries retained despite same spawn id/bundle",
		},
		{
			name:        "time fields canonicalized to UTC",
			initialMeta: ActionItemMetadata{},
			entry: SpawnHistoryEntry{
				SpawnID:      "spawn-tz",
				BundlePath:   "/tmp/tz-test",
				StartedAt:    time.Date(2026, 5, 26, 12, 0, 0, 0, time.FixedZone("EST", -5*3600)),
				TerminatedAt: time.Date(2026, 5, 26, 12, 5, 0, 0, time.FixedZone("EST", -5*3600)),
				Outcome:      "success",
			},
			wantLen:     1,
			description: "times canonicalized to UTC",
		},
		{
			name:        "string fields trimmed",
			initialMeta: ActionItemMetadata{},
			entry: SpawnHistoryEntry{
				SpawnID:      "  spawn-trimmed  ",
				BundlePath:   "  /tmp/bundle  ",
				StartedAt:    time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC),
				TerminatedAt: time.Date(2026, 5, 26, 12, 5, 0, 0, time.UTC),
				Outcome:      "  success  ",
			},
			wantLen:     1,
			description: "string fields trimmed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := tt.initialMeta

			// Append entry.
			meta.AppendSpawnHistory(tt.entry)

			// Verify final length.
			if got := len(meta.SpawnHistory); got != tt.wantLen {
				t.Fatalf("want len %d, got %d", tt.wantLen, got)
			}

			// Verify the appended entry.
			appended := meta.SpawnHistory[len(meta.SpawnHistory)-1]

			// Verify string fields trimmed.
			wantSpawnID := strings.TrimSpace(tt.entry.SpawnID)
			if appended.SpawnID != wantSpawnID {
				t.Errorf("SpawnID not trimmed: want %q, got %q", wantSpawnID, appended.SpawnID)
			}

			wantBundlePath := strings.TrimSpace(tt.entry.BundlePath)
			if appended.BundlePath != wantBundlePath {
				t.Errorf("BundlePath not trimmed: want %q, got %q", wantBundlePath, appended.BundlePath)
			}

			wantOutcome := strings.TrimSpace(tt.entry.Outcome)
			if appended.Outcome != wantOutcome {
				t.Errorf("Outcome not trimmed: want %q, got %q", wantOutcome, appended.Outcome)
			}

			// Verify time fields canonicalized to UTC.
			if appended.StartedAt.Location() != time.UTC {
				t.Errorf("StartedAt not UTC: %v", appended.StartedAt.Location())
			}
			if appended.TerminatedAt.Location() != time.UTC {
				t.Errorf("TerminatedAt not UTC: %v", appended.TerminatedAt.Location())
			}
		})
	}
}
