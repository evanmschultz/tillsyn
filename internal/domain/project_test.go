package domain

import (
	"encoding/json"
	"strings"
	"testing"
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
