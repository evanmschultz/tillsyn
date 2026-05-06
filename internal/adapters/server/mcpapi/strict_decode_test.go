package mcpapi

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// strictDecodeTarget is the test fixture struct the helper decodes into.
// It mixes plain string fields with the pointer-sentinel shape introduced
// by Drop 4c.5 A.1, so the test surface exercises both legacy zero-value
// fields and the post-A.1 absent-vs-explicit-null distinction.
type strictDecodeTarget struct {
	Operation   string    `json:"operation"`
	ProjectID   string    `json:"project_id"`
	Title       *string   `json:"title"`
	Description *string   `json:"description"`
	Labels      *[]string `json:"labels"`
}

// TestBindArgumentsStrictAcceptsValidInput proves the strict decoder yields
// the same field values as the legacy BindArguments path on a fully-known
// argument set, including the post-A.1 pointer-sentinel fields.
func TestBindArgumentsStrictAcceptsValidInput(t *testing.T) {
	t.Parallel()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "till.action_item",
			Arguments: map[string]any{
				"operation":   "create",
				"project_id":  "p-1",
				"title":       "hello",
				"description": "details",
				"labels":      []any{"a", "b"},
			},
		},
	}

	var got strictDecodeTarget
	if err := bindArgumentsStrict(req, &got); err != nil {
		t.Fatalf("bindArgumentsStrict() error = %v, want nil", err)
	}
	if got.Operation != "create" || got.ProjectID != "p-1" {
		t.Fatalf("scalar fields = %+v, want operation=create project_id=p-1", got)
	}
	if got.Title == nil || *got.Title != "hello" {
		t.Fatalf("Title pointer = %v, want non-nil pointing at \"hello\"", got.Title)
	}
	if got.Description == nil || *got.Description != "details" {
		t.Fatalf("Description pointer = %v, want non-nil pointing at \"details\"", got.Description)
	}
	if got.Labels == nil || len(*got.Labels) != 2 || (*got.Labels)[0] != "a" || (*got.Labels)[1] != "b" {
		t.Fatalf("Labels = %v, want [a,b]", got.Labels)
	}
}

// TestBindArgumentsStrictPreservesNullPointer proves the strict decoder does
// NOT reject an explicit null on a known pointer-shape field — A.1's wire
// contract relies on null decoding to a typed nil pointer to distinguish
// "preserve" from "explicit clear" for callers that send {"description":
// null} (legacy tolerance) the same way it handles a missing key.
func TestBindArgumentsStrictPreservesNullPointer(t *testing.T) {
	t.Parallel()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "till.action_item",
			Arguments: json.RawMessage(`{
				"operation": "update",
				"description": null,
				"title": null,
				"labels": null
			}`),
		},
	}

	var got strictDecodeTarget
	if err := bindArgumentsStrict(req, &got); err != nil {
		t.Fatalf("bindArgumentsStrict() error = %v, want nil (null on known pointer field is legal)", err)
	}
	if got.Description != nil {
		t.Fatalf("Description = %v, want nil (json null → typed nil pointer)", got.Description)
	}
	if got.Title != nil {
		t.Fatalf("Title = %v, want nil", got.Title)
	}
	if got.Labels != nil {
		t.Fatalf("Labels = %v, want nil", got.Labels)
	}
	if got.Operation != "update" {
		t.Fatalf("Operation = %q, want \"update\"", got.Operation)
	}
}

// TestBindArgumentsStrictRejectsUnknownKey is the central spec-driven case:
// a typo'd key produces a structured error that names both the offending
// field and the tool.
func TestBindArgumentsStrictRejectsUnknownKey(t *testing.T) {
	t.Parallel()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "till.action_item",
			Arguments: map[string]any{
				"operation": "create",
				"descrption": "typo'd description (note missing 'i')",
			},
		},
	}

	var got strictDecodeTarget
	err := bindArgumentsStrict(req, &got)
	if err == nil {
		t.Fatalf("bindArgumentsStrict() error = nil, want unknown-field rejection")
	}
	if !errors.Is(err, errUnknownField) {
		t.Fatalf("errors.Is(err, errUnknownField) = false, want true (err = %v)", err)
	}
	msg := err.Error()
	if !strings.Contains(msg, `"descrption"`) {
		t.Fatalf("error message = %q, want it to name the offending field \"descrption\"", msg)
	}
	if !strings.Contains(msg, `"till.action_item"`) {
		t.Fatalf("error message = %q, want it to name the tool \"till.action_item\"", msg)
	}
}

// TestBindArgumentsStrictMultipleUnknownKeysReportsFirst pins json.Decoder's
// stop-at-first-error semantics. The decoder reports the first unknown key
// it encounters; the rest are not surfaced until the caller fixes the named
// one and retries.
func TestBindArgumentsStrictMultipleUnknownKeysReportsFirst(t *testing.T) {
	t.Parallel()

	// Use json.RawMessage so the key order is deterministic.
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "till.action_item",
			Arguments: json.RawMessage(`{
				"operation": "create",
				"first_unknown": "x",
				"second_unknown": "y"
			}`),
		},
	}

	var got strictDecodeTarget
	err := bindArgumentsStrict(req, &got)
	if err == nil {
		t.Fatalf("bindArgumentsStrict() error = nil, want rejection")
	}
	if !errors.Is(err, errUnknownField) {
		t.Fatalf("errors.Is(err, errUnknownField) = false, want true (err = %v)", err)
	}
	msg := err.Error()
	if !strings.Contains(msg, `"first_unknown"`) {
		t.Fatalf("error message = %q, want it to name first_unknown (decoder stops at first unknown)", msg)
	}
}

// TestBindArgumentsStrictHandlesNilArguments mirrors BindArguments behavior
// when the arguments map is absent: json.Marshal(nil) → "null", which the
// decoder accepts and decodes to the zero value. No error.
func TestBindArgumentsStrictHandlesNilArguments(t *testing.T) {
	t.Parallel()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "till.action_item",
			Arguments: nil,
		},
	}

	var got strictDecodeTarget
	if err := bindArgumentsStrict(req, &got); err != nil {
		t.Fatalf("bindArgumentsStrict(nil-args) error = %v, want nil (legacy parity)", err)
	}
	if got.Operation != "" || got.Title != nil || got.Description != nil || got.Labels != nil {
		t.Fatalf("zero-value decode produced = %+v, want all zero", got)
	}
}

// TestBindArgumentsStrictHandlesEmptyArgumentsMap covers the explicit empty
// object case ({}) — every field stays at its zero value, no error.
func TestBindArgumentsStrictHandlesEmptyArgumentsMap(t *testing.T) {
	t.Parallel()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "till.action_item",
			Arguments: map[string]any{},
		},
	}

	var got strictDecodeTarget
	if err := bindArgumentsStrict(req, &got); err != nil {
		t.Fatalf("bindArgumentsStrict({}) error = %v, want nil", err)
	}
	if got.Operation != "" || got.Title != nil {
		t.Fatalf("zero-value decode = %+v, want all zero", got)
	}
}

// TestBindArgumentsStrictRejectsNonPointerTarget mirrors BindArguments's
// own input-shape guard so callers that misuse the helper get the same
// fail-fast diagnostic.
func TestBindArgumentsStrictRejectsNonPointerTarget(t *testing.T) {
	t.Parallel()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "till.action_item",
			Arguments: map[string]any{"operation": "create"},
		},
	}

	var got strictDecodeTarget
	err := bindArgumentsStrict(req, got)
	if err == nil || !strings.Contains(err.Error(), "non-nil pointer") {
		t.Fatalf("non-pointer target error = %v, want \"... non-nil pointer ...\"", err)
	}
}

// TestBindArgumentsStrictRejectsNilTarget covers the nil-pointer path of
// the input-shape guard.
func TestBindArgumentsStrictRejectsNilTarget(t *testing.T) {
	t.Parallel()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "till.action_item",
			Arguments: map[string]any{},
		},
	}

	err := bindArgumentsStrict(req, nil)
	if err == nil || !strings.Contains(err.Error(), "non-nil pointer") {
		t.Fatalf("nil target error = %v, want \"... non-nil pointer ...\"", err)
	}
}

// TestBindArgumentsStrictRawMessageFastPath proves the fast-path branch
// (req.Params.Arguments is already json.RawMessage) decodes identically to
// the re-marshal path. The fast-path must still reject unknown keys.
func TestBindArgumentsStrictRawMessageFastPath(t *testing.T) {
	t.Parallel()

	t.Run("valid raw message", func(t *testing.T) {
		t.Parallel()
		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "till.action_item",
				Arguments: json.RawMessage(`{"operation":"create","title":"x"}`),
			},
		}
		var got strictDecodeTarget
		if err := bindArgumentsStrict(req, &got); err != nil {
			t.Fatalf("raw-message valid error = %v, want nil", err)
		}
		if got.Operation != "create" || got.Title == nil || *got.Title != "x" {
			t.Fatalf("decoded = %+v, want operation=create title=x", got)
		}
	})

	t.Run("unknown key in raw message", func(t *testing.T) {
		t.Parallel()
		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "till.handoff",
				Arguments: json.RawMessage(`{"operation":"create","not_a_field":"x"}`),
			},
		}
		var got strictDecodeTarget
		err := bindArgumentsStrict(req, &got)
		if err == nil || !errors.Is(err, errUnknownField) {
			t.Fatalf("raw-message unknown-key err = %v, want errUnknownField", err)
		}
		if !strings.Contains(err.Error(), `"not_a_field"`) || !strings.Contains(err.Error(), `"till.handoff"`) {
			t.Fatalf("error message = %q, want field+tool names", err.Error())
		}
	})
}

// TestUnknownFieldNameRecoveryEdgeCases covers the parser branches that
// bindArgumentsStrict reaches when the std lib changes its error format
// or when the helper is reused outside its primary call site.
func TestUnknownFieldNameRecoveryEdgeCases(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		err       error
		wantField string
		wantOK    bool
	}{
		{name: "nil error", err: nil, wantField: "", wantOK: false},
		{name: "non-matching prefix", err: errors.New("json: cannot unmarshal foo"), wantField: "", wantOK: false},
		{
			name:      "stable std lib format",
			err:       errors.New(`json: unknown field "abc"`),
			wantField: "abc",
			wantOK:    true,
		},
		{
			name:      "fallback path on bare-token tail",
			err:       errors.New(`json: unknown field bare_token`),
			wantField: "bare_token",
			wantOK:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotField, gotOK := unknownFieldName(tc.err)
			if gotField != tc.wantField || gotOK != tc.wantOK {
				t.Fatalf("unknownFieldName(%v) = (%q, %v), want (%q, %v)", tc.err, gotField, gotOK, tc.wantField, tc.wantOK)
			}
		})
	}
}
