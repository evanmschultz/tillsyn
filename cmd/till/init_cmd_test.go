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

// TestInit_JSONInvocation_ReturnsJSONStubError verifies that `till init
// --json '{...}'` routes through cobra to the D3a-stage JSON stub error.
// The flag is registered + readable in D3a but the parser body is a STUB —
// any non-empty --json payload returns the W2.D3b not-yet-wired error.
// CONSUMER-TIE TEST CONTRACT (W2-FF6 ROUND-2).
func TestInit_JSONInvocation_ReturnsJSONStubError(t *testing.T) {
	var out strings.Builder
	err := run(context.Background(), []string{"--app", "tillsyn-init", "init", "--json", `{"name":"foo","group":"till-go","mcp":false}`}, &out, io.Discard)
	if err == nil {
		t.Fatalf("run(init --json) returned nil; expected D3a JSON stub error, got stdout=%q", out.String())
	}
	want := "till init: JSON parse not yet wired (W2.D3b)"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("run(init --json) error = %q; want substring %q", err.Error(), want)
	}
}
