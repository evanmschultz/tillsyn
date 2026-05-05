package domain

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

// TestProjectMetadataOrchSelfApprovalIsEnabledDefaults verifies the three-
// state pointer-bool helper for the Drop 4a Wave 3 W3.2 project-metadata
// opt-out toggle: nil → true (default-enabled), *true → true, *false →
// false. The nil-means-enabled rule prevents the JSON zero-value silent-
// disable failure mode (W3.2 falsification attack 3) — a plain bool would
// have made every legacy project's metadata-decode flip the toggle off.
func TestProjectMetadataOrchSelfApprovalIsEnabledDefaults(t *testing.T) {
	t.Parallel()
	trueVal := true
	falseVal := false
	cases := []struct {
		name string
		ptr  *bool
		want bool
	}{
		{"nil_defaults_to_enabled", nil, true},
		{"explicit_true", &trueVal, true},
		{"explicit_false", &falseVal, false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			meta := ProjectMetadata{OrchSelfApprovalEnabled: tc.ptr}
			if got := meta.OrchSelfApprovalIsEnabled(); got != tc.want {
				t.Fatalf("OrchSelfApprovalIsEnabled() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestProjectMetadataOrchSelfApprovalEnabledJSONRoundTrip verifies the
// pointer-bool field round-trips through JSON for all three states without
// collapsing nil into false (the silent-disable failure mode). Drop 4a
// Wave 3 W3.2.
func TestProjectMetadataOrchSelfApprovalEnabledJSONRoundTrip(t *testing.T) {
	t.Parallel()
	trueVal := true
	falseVal := false
	cases := []struct {
		name            string
		input           ProjectMetadata
		wantRawIncludes string // substring expected in marshaled JSON; "" means must NOT include the toggle key
		wantRawExcludes string // substring that MUST NOT appear in marshaled JSON
	}{
		{
			name:            "nil_omits_field",
			input:           ProjectMetadata{},
			wantRawExcludes: "orch_self_approval_enabled",
		},
		{
			name:            "explicit_true_serializes",
			input:           ProjectMetadata{OrchSelfApprovalEnabled: &trueVal},
			wantRawIncludes: `"orch_self_approval_enabled":true`,
		},
		{
			name:            "explicit_false_serializes",
			input:           ProjectMetadata{OrchSelfApprovalEnabled: &falseVal},
			wantRawIncludes: `"orch_self_approval_enabled":false`,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			raw, err := json.Marshal(tc.input)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}
			rawStr := string(raw)
			if tc.wantRawIncludes != "" && !strings.Contains(rawStr, tc.wantRawIncludes) {
				t.Fatalf("marshaled JSON %q missing expected substring %q", rawStr, tc.wantRawIncludes)
			}
			if tc.wantRawExcludes != "" && strings.Contains(rawStr, tc.wantRawExcludes) {
				t.Fatalf("marshaled JSON %q unexpectedly contains %q", rawStr, tc.wantRawExcludes)
			}

			var decoded ProjectMetadata
			if err := json.Unmarshal(raw, &decoded); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}
			gotEnabled := decoded.OrchSelfApprovalIsEnabled()
			wantEnabled := tc.input.OrchSelfApprovalIsEnabled()
			if gotEnabled != wantEnabled {
				t.Fatalf("round-trip OrchSelfApprovalIsEnabled() = %v, want %v", gotEnabled, wantEnabled)
			}
			// Pointer-shape equivalence: nil stays nil, non-nil stays non-nil.
			if (decoded.OrchSelfApprovalEnabled == nil) != (tc.input.OrchSelfApprovalEnabled == nil) {
				t.Fatalf("round-trip pointer nil-ness changed: input nil=%v, decoded nil=%v",
					tc.input.OrchSelfApprovalEnabled == nil, decoded.OrchSelfApprovalEnabled == nil)
			}
		})
	}
}

// TestProjectMetadataIsDispatcherCommitEnabledDefaults verifies the three-
// state pointer-bool helper for the Drop 4c F.7.15 dispatcher-commit gate.
// Polarity is INVERTED relative to OrchSelfApprovalIsEnabled — nil →
// false (default-disabled), *true → true, *false → false. Default-OFF
// per Master PLAN.md L20.
func TestProjectMetadataIsDispatcherCommitEnabledDefaults(t *testing.T) {
	t.Parallel()
	trueVal := true
	falseVal := false
	cases := []struct {
		name string
		ptr  *bool
		want bool
	}{
		{"nil_defaults_to_disabled", nil, false},
		{"explicit_true", &trueVal, true},
		{"explicit_false", &falseVal, false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			meta := ProjectMetadata{DispatcherCommitEnabled: tc.ptr}
			if got := meta.IsDispatcherCommitEnabled(); got != tc.want {
				t.Fatalf("IsDispatcherCommitEnabled() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestProjectMetadataIsDispatcherPushEnabledDefaults verifies the three-
// state pointer-bool helper for the Drop 4c F.7.15 dispatcher-push gate.
// Same default-disabled polarity as DispatcherCommitEnabled. Master
// PLAN.md L20.
func TestProjectMetadataIsDispatcherPushEnabledDefaults(t *testing.T) {
	t.Parallel()
	trueVal := true
	falseVal := false
	cases := []struct {
		name string
		ptr  *bool
		want bool
	}{
		{"nil_defaults_to_disabled", nil, false},
		{"explicit_true", &trueVal, true},
		{"explicit_false", &falseVal, false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			meta := ProjectMetadata{DispatcherPushEnabled: tc.ptr}
			if got := meta.IsDispatcherPushEnabled(); got != tc.want {
				t.Fatalf("IsDispatcherPushEnabled() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestProjectMetadataDispatcherTogglesJSONRoundTrip verifies both new
// pointer-bool fields survive JSON marshal/unmarshal cycles for all
// three states without collapsing nil into false. Drop 4c F.7.15.
//
// Both fields are exercised in a single test because their semantics
// are identical (default-disabled pointer-bools) and isolating them
// would duplicate scaffolding without adding coverage.
func TestProjectMetadataDispatcherTogglesJSONRoundTrip(t *testing.T) {
	t.Parallel()
	trueVal := true
	falseVal := false
	cases := []struct {
		name      string
		input     ProjectMetadata
		mustHave  []string
		mustOmit  []string
		commitOK  bool
		pushOK    bool
		nilCommit bool
		nilPush   bool
	}{
		{
			name:      "both_nil_omits_both_keys",
			input:     ProjectMetadata{},
			mustOmit:  []string{"dispatcher_commit_enabled", "dispatcher_push_enabled"},
			commitOK:  false,
			pushOK:    false,
			nilCommit: true,
			nilPush:   true,
		},
		{
			name:      "commit_true_push_nil",
			input:     ProjectMetadata{DispatcherCommitEnabled: &trueVal},
			mustHave:  []string{`"dispatcher_commit_enabled":true`},
			mustOmit:  []string{"dispatcher_push_enabled"},
			commitOK:  true,
			pushOK:    false,
			nilCommit: false,
			nilPush:   true,
		},
		{
			name:      "both_explicit_false",
			input:     ProjectMetadata{DispatcherCommitEnabled: &falseVal, DispatcherPushEnabled: &falseVal},
			mustHave:  []string{`"dispatcher_commit_enabled":false`, `"dispatcher_push_enabled":false`},
			commitOK:  false,
			pushOK:    false,
			nilCommit: false,
			nilPush:   false,
		},
		{
			name:      "both_explicit_true",
			input:     ProjectMetadata{DispatcherCommitEnabled: &trueVal, DispatcherPushEnabled: &trueVal},
			mustHave:  []string{`"dispatcher_commit_enabled":true`, `"dispatcher_push_enabled":true`},
			commitOK:  true,
			pushOK:    true,
			nilCommit: false,
			nilPush:   false,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			raw, err := json.Marshal(tc.input)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}
			rawStr := string(raw)
			for _, want := range tc.mustHave {
				if !strings.Contains(rawStr, want) {
					t.Fatalf("marshaled JSON %q missing expected substring %q", rawStr, want)
				}
			}
			for _, forbidden := range tc.mustOmit {
				if strings.Contains(rawStr, forbidden) {
					t.Fatalf("marshaled JSON %q unexpectedly contains %q", rawStr, forbidden)
				}
			}

			var decoded ProjectMetadata
			if err := json.Unmarshal(raw, &decoded); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}
			if got := decoded.IsDispatcherCommitEnabled(); got != tc.commitOK {
				t.Fatalf("round-trip IsDispatcherCommitEnabled() = %v, want %v", got, tc.commitOK)
			}
			if got := decoded.IsDispatcherPushEnabled(); got != tc.pushOK {
				t.Fatalf("round-trip IsDispatcherPushEnabled() = %v, want %v", got, tc.pushOK)
			}
			if (decoded.DispatcherCommitEnabled == nil) != tc.nilCommit {
				t.Fatalf("round-trip DispatcherCommitEnabled nil-ness changed: got nil=%v, want nil=%v",
					decoded.DispatcherCommitEnabled == nil, tc.nilCommit)
			}
			if (decoded.DispatcherPushEnabled == nil) != tc.nilPush {
				t.Fatalf("round-trip DispatcherPushEnabled nil-ness changed: got nil=%v, want nil=%v",
					decoded.DispatcherPushEnabled == nil, tc.nilPush)
			}
		})
	}
}

// TestProjectMetadataDispatcherTogglesTOMLRoundTrip verifies both new
// pointer-bool fields survive TOML marshal/unmarshal cycles. Templates
// declare project metadata in TOML form (per CLAUDE.md template-binding
// notes); the fields therefore carry first-class `toml:` tags. Drop 4c
// F.7.15.
//
// pelletier/go-toml/v2 is the project's TOML lib (go.mod line 66). The
// `omitempty` semantic for pointer-bools mirrors the JSON encoder.
func TestProjectMetadataDispatcherTogglesTOMLRoundTrip(t *testing.T) {
	t.Parallel()
	trueVal := true
	falseVal := false
	cases := []struct {
		name      string
		input     ProjectMetadata
		mustHave  []string
		mustOmit  []string
		commitOK  bool
		pushOK    bool
		nilCommit bool
		nilPush   bool
	}{
		{
			name:      "both_nil_omits_both_keys",
			input:     ProjectMetadata{},
			mustOmit:  []string{"dispatcher_commit_enabled", "dispatcher_push_enabled"},
			commitOK:  false,
			pushOK:    false,
			nilCommit: true,
			nilPush:   true,
		},
		{
			name:      "both_explicit_true",
			input:     ProjectMetadata{DispatcherCommitEnabled: &trueVal, DispatcherPushEnabled: &trueVal},
			mustHave:  []string{"dispatcher_commit_enabled = true", "dispatcher_push_enabled = true"},
			commitOK:  true,
			pushOK:    true,
			nilCommit: false,
			nilPush:   false,
		},
		{
			name:      "both_explicit_false",
			input:     ProjectMetadata{DispatcherCommitEnabled: &falseVal, DispatcherPushEnabled: &falseVal},
			mustHave:  []string{"dispatcher_commit_enabled = false", "dispatcher_push_enabled = false"},
			commitOK:  false,
			pushOK:    false,
			nilCommit: false,
			nilPush:   false,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			raw, err := toml.Marshal(tc.input)
			if err != nil {
				t.Fatalf("toml.Marshal() error = %v", err)
			}
			rawStr := string(raw)
			for _, want := range tc.mustHave {
				if !strings.Contains(rawStr, want) {
					t.Fatalf("marshaled TOML %q missing expected substring %q", rawStr, want)
				}
			}
			for _, forbidden := range tc.mustOmit {
				if strings.Contains(rawStr, forbidden) {
					t.Fatalf("marshaled TOML %q unexpectedly contains %q", rawStr, forbidden)
				}
			}

			var decoded ProjectMetadata
			if err := toml.Unmarshal(raw, &decoded); err != nil {
				t.Fatalf("toml.Unmarshal() error = %v", err)
			}
			if got := decoded.IsDispatcherCommitEnabled(); got != tc.commitOK {
				t.Fatalf("round-trip IsDispatcherCommitEnabled() = %v, want %v", got, tc.commitOK)
			}
			if got := decoded.IsDispatcherPushEnabled(); got != tc.pushOK {
				t.Fatalf("round-trip IsDispatcherPushEnabled() = %v, want %v", got, tc.pushOK)
			}
			if (decoded.DispatcherCommitEnabled == nil) != tc.nilCommit {
				t.Fatalf("round-trip DispatcherCommitEnabled nil-ness changed: got nil=%v, want nil=%v",
					decoded.DispatcherCommitEnabled == nil, tc.nilCommit)
			}
			if (decoded.DispatcherPushEnabled == nil) != tc.nilPush {
				t.Fatalf("round-trip DispatcherPushEnabled nil-ness changed: got nil=%v, want nil=%v",
					decoded.DispatcherPushEnabled == nil, tc.nilPush)
			}
		})
	}
}
