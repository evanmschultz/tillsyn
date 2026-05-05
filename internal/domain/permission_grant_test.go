package domain

import (
	"errors"
	"testing"
	"time"
)

// TestNewPermissionGrantValidInput verifies a well-formed input round-trips
// with the expected normalization and UTC timestamp.
func TestNewPermissionGrantValidInput(t *testing.T) {
	now := time.Date(2026, 5, 4, 10, 30, 0, 0, time.UTC)
	in := PermissionGrantInput{
		ID:        "  grant-1  ",
		ProjectID: "  proj-1  ",
		Kind:      KindBuild,
		Rule:      "  Bash(npm run *)  ",
		CLIKind:   "  Claude  ",
		GrantedBy: "  STEWARD  ",
	}
	got, err := NewPermissionGrant(in, now)
	if err != nil {
		t.Fatalf("NewPermissionGrant() unexpected error: %v", err)
	}
	if got.ID != "grant-1" {
		t.Errorf("ID = %q, want %q", got.ID, "grant-1")
	}
	if got.ProjectID != "proj-1" {
		t.Errorf("ProjectID = %q, want %q", got.ProjectID, "proj-1")
	}
	if got.Kind != KindBuild {
		t.Errorf("Kind = %q, want %q", got.Kind, KindBuild)
	}
	if got.Rule != "Bash(npm run *)" {
		t.Errorf("Rule = %q, want %q", got.Rule, "Bash(npm run *)")
	}
	if got.CLIKind != "claude" {
		t.Errorf("CLIKind = %q, want %q (lowercased)", got.CLIKind, "claude")
	}
	if got.GrantedBy != "STEWARD" {
		t.Errorf("GrantedBy = %q, want %q", got.GrantedBy, "STEWARD")
	}
	if !got.GrantedAt.Equal(now) {
		t.Errorf("GrantedAt = %v, want %v", got.GrantedAt, now)
	}
	if got.GrantedAt.Location() != time.UTC {
		t.Errorf("GrantedAt.Location = %v, want UTC", got.GrantedAt.Location())
	}
}

// TestNewPermissionGrantValidationRejections verifies fail-closed validation
// for every required field.
func TestNewPermissionGrantValidationRejections(t *testing.T) {
	now := time.Date(2026, 5, 4, 10, 30, 0, 0, time.UTC)
	base := PermissionGrantInput{
		ID:        "g-1",
		ProjectID: "p-1",
		Kind:      KindBuild,
		Rule:      "Read(./.zshrc)",
		CLIKind:   "claude",
		GrantedBy: "STEWARD",
	}

	tests := []struct {
		name    string
		mutate  func(in *PermissionGrantInput)
		wantErr error
	}{
		{
			name:    "empty ID",
			mutate:  func(in *PermissionGrantInput) { in.ID = "" },
			wantErr: ErrInvalidID,
		},
		{
			name:    "whitespace ID",
			mutate:  func(in *PermissionGrantInput) { in.ID = "   " },
			wantErr: ErrInvalidID,
		},
		{
			name:    "empty ProjectID",
			mutate:  func(in *PermissionGrantInput) { in.ProjectID = "" },
			wantErr: ErrInvalidID,
		},
		{
			name:    "whitespace ProjectID",
			mutate:  func(in *PermissionGrantInput) { in.ProjectID = "   " },
			wantErr: ErrInvalidID,
		},
		{
			name:    "empty Kind",
			mutate:  func(in *PermissionGrantInput) { in.Kind = "" },
			wantErr: ErrInvalidKind,
		},
		{
			name:    "unknown Kind",
			mutate:  func(in *PermissionGrantInput) { in.Kind = Kind("not-a-real-kind") },
			wantErr: ErrInvalidKind,
		},
		{
			name:    "empty Rule",
			mutate:  func(in *PermissionGrantInput) { in.Rule = "" },
			wantErr: ErrInvalidPermissionGrantRule,
		},
		{
			name:    "whitespace Rule",
			mutate:  func(in *PermissionGrantInput) { in.Rule = "   " },
			wantErr: ErrInvalidPermissionGrantRule,
		},
		{
			name:    "empty CLIKind",
			mutate:  func(in *PermissionGrantInput) { in.CLIKind = "" },
			wantErr: ErrInvalidPermissionGrantCLIKind,
		},
		{
			name:    "whitespace CLIKind",
			mutate:  func(in *PermissionGrantInput) { in.CLIKind = "   " },
			wantErr: ErrInvalidPermissionGrantCLIKind,
		},
		{
			name:    "empty GrantedBy",
			mutate:  func(in *PermissionGrantInput) { in.GrantedBy = "" },
			wantErr: ErrInvalidPermissionGrantGrantedBy,
		},
		{
			name:    "whitespace GrantedBy",
			mutate:  func(in *PermissionGrantInput) { in.GrantedBy = "   " },
			wantErr: ErrInvalidPermissionGrantGrantedBy,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := base
			tt.mutate(&in)
			_, err := NewPermissionGrant(in, now)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewPermissionGrant() err = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// TestNewPermissionGrantUTCNormalization verifies a non-UTC `now` is
// normalized to UTC on the returned record.
func TestNewPermissionGrantUTCNormalization(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("LoadLocation() error = %v", err)
	}
	now := time.Date(2026, 5, 4, 6, 30, 0, 0, loc)
	got, err := NewPermissionGrant(PermissionGrantInput{
		ID:        "g-1",
		ProjectID: "p-1",
		Kind:      KindBuildQAProof,
		Rule:      "Bash(mage *)",
		CLIKind:   "claude",
		GrantedBy: "orch-1",
	}, now)
	if err != nil {
		t.Fatalf("NewPermissionGrant() error = %v", err)
	}
	if got.GrantedAt.Location() != time.UTC {
		t.Errorf("GrantedAt.Location = %v, want UTC", got.GrantedAt.Location())
	}
	// 06:30 EDT == 10:30 UTC; LoadLocation handles DST.
	if !got.GrantedAt.Equal(now) {
		t.Errorf("GrantedAt = %v, want %v (in UTC)", got.GrantedAt, now)
	}
}
