// Table-driven tests for ConvertClaudeToolName. Every conversion-doc row from
// ~/.claude/codex-mcp-dispatch-tool-conversion.md lines 19-48 is reproduced
// verbatim below. Adding a server to serverToolFormat without adding rows here
// must be caught by reviewer — there is no schema-level guard.
//
// Upstream Codex issues backing the conversion rule: #15437, #15753, #16501,
// #19430, #13476.
package dispatcher

import (
	"errors"
	"strings"
	"testing"
)

func TestConvertClaudeToolName_KnownServers(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		input     string
		wantSrv   string
		wantTool  string
	}{
		// hylla — underscore→dot rewrite. Verbatim conversion-doc rows.
		{
			name:     "hylla_search_vector",
			input:    "mcp__hylla__hylla_search_vector",
			wantSrv:  "hylla",
			wantTool: "hylla.search.vector",
		},
		{
			name:     "hylla_artifact_overview",
			input:    "mcp__hylla__hylla_artifact_overview",
			wantSrv:  "hylla",
			wantTool: "hylla.artifact.overview",
		},
		// ta — tail-only, verbatim.
		{
			name:     "ta_get",
			input:    "mcp__ta__get",
			wantSrv:  "ta",
			wantTool: "get",
		},
		{
			name:     "ta_update",
			input:    "mcp__ta__update",
			wantSrv:  "ta",
			wantTool: "update",
		},
		// gopls — verbatim underscore form preserved.
		{
			name:     "gopls_go_search",
			input:    "mcp__gopls__go_search",
			wantSrv:  "gopls",
			wantTool: "go_search",
		},
		// tillsyn — Claude-Code form already carries the dot; pass-through.
		{
			name:     "tillsyn_till.attention_item",
			input:    "mcp__tillsyn__till.attention_item",
			wantSrv:  "tillsyn",
			wantTool: "till.attention_item",
		},
		{
			name:     "tillsyn_till.action_item",
			input:    "mcp__tillsyn__till.action_item",
			wantSrv:  "tillsyn",
			wantTool: "till.action_item",
		},
		{
			name:     "tillsyn_till.comment",
			input:    "mcp__tillsyn__till.comment",
			wantSrv:  "tillsyn",
			wantTool: "till.comment",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotSrv, gotTool, err := ConvertClaudeToolName(tc.input)
			if err != nil {
				t.Fatalf("ConvertClaudeToolName(%q) returned unexpected error: %v", tc.input, err)
			}
			if gotSrv != tc.wantSrv {
				t.Errorf("server mismatch for %q: got %q want %q", tc.input, gotSrv, tc.wantSrv)
			}
			if gotTool != tc.wantTool {
				t.Errorf("canonical mismatch for %q: got %q want %q", tc.input, gotTool, tc.wantTool)
			}
		})
	}
}

func TestConvertClaudeToolName_UnknownServerSentinel(t *testing.T) {
	t.Parallel()
	srv, tool, err := ConvertClaudeToolName("mcp__unknown__foo")
	if !errors.Is(err, ErrUnknownMCPServer) {
		t.Fatalf("expected ErrUnknownMCPServer, got err=%v", err)
	}
	if srv != "" || tool != "" {
		t.Errorf("expected empty server+tool on error, got (%q, %q)", srv, tool)
	}
	// The wrapped error should mention the offending server segment so the
	// dispatcher's operator-facing logs are debuggable.
	if !strings.Contains(err.Error(), "unknown") {
		t.Errorf("expected wrapped error to mention 'unknown', got %q", err.Error())
	}
}

func TestConvertClaudeToolName_InvalidShape(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		input string
	}{
		{"empty_string", ""},
		{"missing_prefix", "hylla__hylla_search_vector"},
		{"prefix_only", "mcp__"},
		{"prefix_plus_server_only", "mcp__hylla"},
		{"prefix_plus_server_plus_trailing_separator", "mcp__hylla__"},
		{"random_garbage", "definitely-not-an-mcp-tool"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv, tool, err := ConvertClaudeToolName(tc.input)
			if !errors.Is(err, ErrInvalidClaudeToolName) {
				t.Fatalf("expected ErrInvalidClaudeToolName for %q, got err=%v", tc.input, err)
			}
			if srv != "" || tool != "" {
				t.Errorf("expected empty server+tool on error for %q, got (%q, %q)", tc.input, srv, tool)
			}
		})
	}
}

// TestConvertClaudeToolName_TableSurfaceCoverage asserts that every server in
// serverToolFormat is exercised by at least one row in
// TestConvertClaudeToolName_KnownServers above. Catches the "added a new
// server but forgot to add tests" failure mode at table-level.
func TestConvertClaudeToolName_TableSurfaceCoverage(t *testing.T) {
	t.Parallel()
	knownServers := map[string]bool{}
	for srv := range serverToolFormat {
		knownServers[srv] = false
	}
	// Mirror the inputs from TestConvertClaudeToolName_KnownServers.
	covered := []string{
		"mcp__hylla__hylla_search_vector",
		"mcp__ta__get",
		"mcp__gopls__go_search",
		"mcp__tillsyn__till.attention_item",
	}
	for _, in := range covered {
		srv, _, err := ConvertClaudeToolName(in)
		if err != nil {
			t.Fatalf("ConvertClaudeToolName(%q) errored: %v", in, err)
		}
		knownServers[srv] = true
	}
	for srv, covered := range knownServers {
		if !covered {
			t.Errorf("server %q in serverToolFormat lacks a verbatim conversion-doc row in TestConvertClaudeToolName_KnownServers", srv)
		}
	}
}
