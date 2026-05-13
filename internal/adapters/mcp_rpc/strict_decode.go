package mcprpc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// errUnknownField is the package-internal sentinel returned (wrapped) by
// bindArgumentsStrict when the inbound JSON arguments object carries a key
// that the target struct does not declare.
//
// Callers route the error through invalidRequestToolResult, which prepends
// "invalid_request: " to the error string. The surface text the MCP client
// sees ends up as e.g.
//
//	invalid_request: unknown field "descrption" on tool "till.action_item"
//
// Use errors.Is(err, errUnknownField) when programmatic detection is needed
// (currently only the helper's own tests rely on it).
var errUnknownField = errors.New("unknown field")

// jsonUnknownFieldPrefix is the canonical prefix produced by
// encoding/json.Decoder.DisallowUnknownFields when it rejects an extra key.
// It is documented as stable in the Go std library: the decoder returns
// fmt.Errorf("json: unknown field %q", key). bindArgumentsStrict matches on
// this prefix to recover the field name without needing a typed error from
// the json package (no such type is exported by encoding/json today).
const jsonUnknownFieldPrefix = "json: unknown field "

// bindArgumentsStrict decodes req.Params.Arguments into target with strict
// field-set checking: any JSON key in the arguments object that does not map
// to an exported field on target produces a wrapped errUnknownField that names
// both the offending field and the tool, so the MCP client receives a
// structured error rather than a silent drop.
//
// Behavior parity with mark3labs CallToolRequest.BindArguments:
//   - target must be a non-nil pointer; otherwise a "target must be a non-nil
//     pointer" error is returned (matches BindArguments wording).
//   - Fast-path: if Arguments is already a json.RawMessage, decode it directly.
//   - Otherwise re-marshal Arguments to bytes, then decode through a strict
//     json.Decoder.
//
// Null-value handling: DisallowUnknownFields only inspects the field-name
// set, not field values. JSON null on a known *string / **time.Time /
// *[]string / *bool field decodes to a typed nil pointer, exactly as plain
// json.Unmarshal would. The pointer-sentinel pattern adopted by Drop 4c.5
// A.1 is preserved.
//
// Error shape on unknown fields: the helper extracts the offending field
// name from the encoding/json error message (prefix "json: unknown field ")
// and returns
//
//	fmt.Errorf("unknown field %q on tool %q: %w", fieldName, toolName, errUnknownField)
//
// invalidRequestToolResult prepends "invalid_request: " when the error
// reaches the wire, producing the surface text shown above.
func bindArgumentsStrict(req mcp.CallToolRequest, target any) error {
	if target == nil || reflect.ValueOf(target).Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a non-nil pointer")
	}

	toolName := strings.TrimSpace(req.Params.Name)

	var data []byte
	if raw, ok := req.Params.Arguments.(json.RawMessage); ok {
		data = []byte(raw)
	} else {
		marshalled, err := json.Marshal(req.Params.Arguments)
		if err != nil {
			return fmt.Errorf("failed to marshal arguments: %w", err)
		}
		data = marshalled
	}

	// Empty / nil arguments map mirrors the legacy BindArguments path:
	// json.Marshal(nil) → "null", which json.Decoder accepts and decodes
	// to the zero value of *target. Strict mode does not change that.
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(target); err != nil {
		if fieldName, ok := unknownFieldName(err); ok {
			return fmt.Errorf("unknown field %q on tool %q: %w", fieldName, toolName, errUnknownField)
		}
		return err
	}
	return nil
}

// unknownFieldName recovers the offending key from an encoding/json
// DisallowUnknownFields error. Returns ("", false) for any other error.
//
// The error message format is stable across Go versions: the decoder
// constructs it via fmt.Errorf("json: unknown field %q", key), so the
// payload is the prefix above followed by a quoted Go string literal.
// strconv.Unquote handles the escape-aware unquoting; if that fails for
// any reason the helper falls back to a naive trim-quotes path, which is
// still correct for every plain-ASCII key that current MCP tools accept.
func unknownFieldName(err error) (string, bool) {
	if err == nil {
		return "", false
	}
	msg := err.Error()
	if !strings.HasPrefix(msg, jsonUnknownFieldPrefix) {
		return "", false
	}
	tail := strings.TrimPrefix(msg, jsonUnknownFieldPrefix)
	tail = strings.TrimSpace(tail)
	if tail == "" {
		return "", false
	}
	if unquoted, uerr := strconv.Unquote(tail); uerr == nil {
		return unquoted, true
	}
	// Fallback: trim a single layer of surrounding quotes if strconv.Unquote
	// rejects the tail (should be unreachable given the std lib's stable
	// formatting, but defends against future format drift).
	if len(tail) >= 2 && tail[0] == '"' && tail[len(tail)-1] == '"' {
		return tail[1 : len(tail)-1], true
	}
	return tail, true
}
