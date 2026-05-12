package main

import (
	"context"
	"io"
	"strings"
	"testing"
)

// TestInit_BareInvocation_ReturnsTUIStubError verifies that `till init` (bare,
// no --json) routes through cobra to the D3a-stage stub error from
// runInitTUI. The end-to-end run() invocation exercises the cobra
// registration in main.go — calling cmd.RunE or runInitTUI directly would
// not prove the command is wired into rootCmd. CONSUMER-TIE TEST CONTRACT
// (W2-FF6 ROUND-2) — symmetric to D7.5's W2-FF3 contract.
func TestInit_BareInvocation_ReturnsTUIStubError(t *testing.T) {
	var out strings.Builder
	err := run(context.Background(), []string{"--app", "tillsyn-init", "init"}, &out, io.Discard)
	if err == nil {
		t.Fatalf("run(init) returned nil; expected D3a TUI stub error, got stdout=%q", out.String())
	}
	want := "till init: TUI walk not yet wired (W2.D4)"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("run(init) error = %q; want substring %q", err.Error(), want)
	}
}

// TestInit_JSONInvocation_RoutesToValidParse verifies that `till init
// --json '{...}'` with a well-formed payload routes through cobra to the
// real JSON parser shipped in D3b. A valid payload parses + validates and
// then surfaces the D5-stub error from the file-copy pipeline (which D5
// will wire). CONSUMER-TIE TEST CONTRACT (W2-FF6 ROUND-2).
func TestInit_JSONInvocation_RoutesToValidParse(t *testing.T) {
	var out strings.Builder
	err := run(context.Background(), []string{"--app", "tillsyn-init", "init", "--json", `{"name":"foo","group":"till-go","mcp":false}`}, &out, io.Discard)
	if err == nil {
		t.Fatalf("run(init --json valid) returned nil; expected D5 file-copy stub error, got stdout=%q", out.String())
	}
	want := "till init: file copy not yet wired (W2.D5)"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("run(init --json valid) error = %q; want substring %q", err.Error(), want)
	}
}

// TestInit_JSONParse_TableDriven covers the D3b JSON-payload parser +
// group-validation matrix: valid payload, reserved `till-gdd` group,
// unknown group, malformed JSON, and missing required fields. Each case
// drives `run(...)` end-to-end so the cobra wiring is exercised; failure
// surfaces are matched by substring against the wrapped error returned
// from `runInitJSON`.
func TestInit_JSONParse_TableDriven(t *testing.T) {
	cases := []struct {
		name        string
		payload     string
		wantSubstrs []string
	}{
		{
			name:        "valid_till_go",
			payload:     `{"name":"foo","group":"till-go","mcp":false}`,
			wantSubstrs: []string{"file copy not yet wired (W2.D5)"},
		},
		{
			name:        "valid_till_gen_mcp_true",
			payload:     `{"name":"bar","group":"till-gen","mcp":true}`,
			wantSubstrs: []string{"file copy not yet wired (W2.D5)"},
		},
		{
			name:        "reserved_group_till_gdd",
			payload:     `{"name":"foo","group":"till-gdd","mcp":false}`,
			wantSubstrs: []string{"till-gdd", "reserved"},
		},
		{
			name:        "unknown_group",
			payload:     `{"name":"foo","group":"till-rust","mcp":false}`,
			wantSubstrs: []string{"group must be one of"},
		},
		{
			name:        "malformed_json",
			payload:     `{not json`,
			wantSubstrs: []string{"till init", "json"},
		},
		{
			name:        "missing_name",
			payload:     `{"group":"till-go"}`,
			wantSubstrs: []string{"name", "required"},
		},
		{
			name:        "missing_group",
			payload:     `{"name":"foo"}`,
			wantSubstrs: []string{"group", "required"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out strings.Builder
			err := run(context.Background(), []string{"--app", "tillsyn-init", "init", "--json", tc.payload}, &out, io.Discard)
			if err == nil {
				t.Fatalf("run(init --json %q) returned nil; expected error containing %v, stdout=%q", tc.payload, tc.wantSubstrs, out.String())
			}
			got := err.Error()
			for _, sub := range tc.wantSubstrs {
				if !strings.Contains(got, sub) {
					t.Fatalf("run(init --json %q) error = %q; want substring %q", tc.payload, got, sub)
				}
			}
		})
	}
}
