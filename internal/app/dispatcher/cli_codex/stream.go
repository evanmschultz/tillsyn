package cli_codex

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/evanmschultz/tillsyn/internal/app/dispatcher"
)

// ErrMalformedStreamLine is returned by parseStreamEvent when the input line
// is not valid JSON. Unlike cli_claude's strict discriminator check, the
// codex adapter is PERMISSIVE — it does not require a `type` field to be
// present because the codex JSONL event schema is partially undocumented.
// Unknown or missing fields degrade gracefully: Raw is always populated,
// and the caller receives a non-terminal event with best-effort type
// normalization.
var ErrMalformedStreamLine = errors.New("cli_codex: malformed stream-json line")

// parseStreamEvent decodes one JSONL line emitted by `codex exec --json` into
// the cross-CLI canonical StreamEvent. The full raw bytes are always retained
// in StreamEvent.Raw so ExtractTerminalReport (and forensic post-mortem
// tooling) can re-decode adapter-private fields without re-reading from disk.
//
// Parsing is PERMISSIVE:
//   - Valid JSON that is NOT a JSON object is treated as a non-terminal event
//     with Raw populated and all other fields at zero value.
//   - Valid JSON objects are decoded into a flexible map; type normalization
//     is best-effort based on known codex event field families (see comments
//     inline).
//   - Non-JSON input returns ErrMalformedStreamLine with Raw still populated
//     from the original line bytes.
//
// Known codex event families (verified via D4 fixture capture):
//
//   - type="thread.started"  — session init; non-terminal.
//   - type="turn.started"    — turn begin; non-terminal.
//   - type="item.completed"  — item output; non-terminal. Canonical mapping:
//     item.type="agent_message" → ev.Type="assistant", ev.Subtype="item.completed"
//     item.type="tool_use"|"function_call" → ev.Type="tool_use", ev.Subtype="item.completed"
//     item.type="reasoning" → ev.Type="reasoning", ev.Subtype="item.completed"
//     unknown item.type → ev.Type="item.completed", ev.Subtype=item.type (permissive fallback)
//   - type="turn.completed"  — turn end with usage; TERMINAL.
//   - type="turn.failed"     — turn error; TERMINAL.
//   - type="error"           — mid-stream error signal (precedes turn.failed); non-terminal.
//
// Unknown type values pass through with Type set to the raw type string
// (or empty if no type-like field is found).
func parseStreamEvent(line []byte) (dispatcher.StreamEvent, error) {
	// Always copy line into Raw before any decoding so the returned
	// StreamEvent.Raw is detached from the caller's buffer. Stream readers
	// commonly reuse a scanner buffer; retaining a slice header into reused
	// memory is a well-known bug vector.
	raw := append(json.RawMessage(nil), line...)

	ev := dispatcher.StreamEvent{Raw: raw}

	// Decode into a flexible map for permissive field access. If the unmarshal
	// fails, check whether the input is valid JSON at all (non-object shapes
	// like arrays, numbers, or strings). Valid non-object JSON is returned as a
	// non-terminal event with Raw populated (permissive contract per doc above).
	// Truly non-JSON input returns ErrMalformedStreamLine.
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(line, &fields); err != nil {
		// Check if the input is valid JSON in a non-object form.
		var jsonVal json.RawMessage
		if jsonErr := json.Unmarshal(line, &jsonVal); jsonErr == nil {
			// Valid JSON but not an object — return as non-terminal event with
			// Raw populated and all other fields at zero value.
			return ev, nil
		}
		return ev, fmt.Errorf("%w: %v", ErrMalformedStreamLine, err)
	}

	// Extract raw string helper (avoids repetition below).
	getString := func(key string) string {
		raw, ok := fields[key]
		if !ok || len(raw) == 0 {
			return ""
		}
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return ""
		}
		return s
	}

	// Map `type` or `kind` fields to the canonical vocabulary.
	typeStr := getString("type")
	if typeStr == "" {
		typeStr = getString("kind")
	}
	ev.Type = typeStr

	switch typeStr {
	case "thread.started", "turn.started":
		// Session/turn lifecycle events — non-terminal; pass through with
		// Type set to the raw value. Raw retains thread_id etc. for callers.

	case "item.completed":
		// Item output event. Decode the nested item object and translate
		// item.type into the canonical ev.Type vocabulary so downstream
		// consumers (e.g. commit_agent filtering on ev.Type=="assistant") are
		// CLI-agnostic. ev.Subtype is always "item.completed" for recognised
		// item subkinds, retaining the codex wire-format name for forensics.
		// Unknown item subkinds fall through to the permissive default below.
		if itemRaw, ok := fields["item"]; ok {
			var item struct {
				Type      string          `json:"type"`
				Text      string          `json:"text"`
				Name      string          `json:"name"`      // tool name (tool_use / function_call)
				Arguments json.RawMessage `json:"arguments"` // tool input args (codex wire format)
			}
			if err := json.Unmarshal(itemRaw, &item); err == nil {
				switch item.Type {
				case "agent_message":
					// Assistant prose output — normalise to the canonical
					// vocabulary so commit_agent's ev.Type=="assistant" filter
					// matches codex-routed spawns.
					ev.Type = "assistant"
					ev.Subtype = "item.completed"
					ev.Text = item.Text
				case "tool_use", "function_call":
					// Tool invocation — normalise to "tool_use" vocabulary.
					ev.Type = "tool_use"
					ev.Subtype = "item.completed"
					ev.ToolName = item.Name
					ev.ToolInput = item.Arguments
				case "reasoning":
					// Internal reasoning block — surface as "reasoning" type.
					ev.Type = "reasoning"
					ev.Subtype = "item.completed"
					ev.Text = item.Text
				default:
					// Unknown item subkind — permissive pass-through. ev.Type
					// retains the codex wire-format "item.completed" and Subtype
					// carries the raw item.type value so callers can refine later.
					ev.Subtype = item.Type
					ev.Text = item.Text
				}
			}
		}

	case "turn.completed":
		// Terminal event — successful turn end with token usage.
		// Cost stays nil: codex emits usage counts but no dollar cost field
		// (per F.7.17 L11 nil signals absent, NOT zero).
		ev.IsTerminal = true

	case "turn.failed":
		// Terminal event — turn ended with an error.
		// Extract error.message from the nested error object.
		if errRaw, ok := fields["error"]; ok {
			ev.Text = extractErrorMessage(errRaw)
		}
		ev.IsTerminal = true

	case "error":
		// Mid-stream error signal — non-terminal. A turn.failed event
		// typically follows. Capture the message for monitor logging.
		ev.Text = getString("message")

	default:
		// Unknown or absent type: pass through verbatim. The dispatcher's
		// monitor logs unknown event types; forward-compat for new codex
		// event families.
	}

	return ev, nil
}

// extractErrorMessage decodes a codex `error` field value into a plain text
// string. The value may be a plain string or an object with a `message` key
// (as in turn.failed: {"message":"..."}). Returns empty string on any
// decode failure — best-effort only.
func extractErrorMessage(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	// Try plain string first (defensive; codex uses object form today).
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	// Try object with message field (documented codex error shape).
	var obj struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil {
		return obj.Message
	}
	return ""
}

// extractTerminalReport returns the structured TerminalReport derived from a
// terminal StreamEvent. Returns (zero, false) for non-terminal events;
// callers MUST check the bool before consuming the report.
//
// The function re-decodes ev.Raw into a flexible map for permissive field
// access. Two terminal event shapes are recognised:
//
//   - turn.completed: turn ended successfully with usage telemetry. Cost stays
//     nil (codex emits token counts but no dollar cost field; per F.7.17 L11
//     nil signals absent, NOT zero). Reason = "turn_completed".
//   - turn.failed: turn ended with an error. Errors[0] = error.message.
//     Reason = "turn_failed".
func extractTerminalReport(ev dispatcher.StreamEvent) (dispatcher.TerminalReport, bool) {
	if !ev.IsTerminal {
		return dispatcher.TerminalReport{}, false
	}

	// Best-effort decode: even a malformed terminal event returns (zero, true)
	// so the dispatcher knows the spawn ended.
	var fields map[string]json.RawMessage
	if len(ev.Raw) == 0 || json.Unmarshal(ev.Raw, &fields) != nil {
		return dispatcher.TerminalReport{}, true
	}

	report := dispatcher.TerminalReport{}

	// Determine event type to select extraction path.
	var typeStr string
	if raw, ok := fields["type"]; ok {
		_ = json.Unmarshal(raw, &typeStr)
	}

	switch typeStr {
	case "turn.completed":
		// Successful turn end. Usage telemetry is present but there is no
		// dollar cost field — Cost stays nil per F.7.17 L11.
		report.Reason = "turn_completed"

	case "turn.failed":
		// Turn ended with error. Extract error.message into Errors. Always
		// populate Errors with at least a sentinel string so consumers can
		// distinguish "failed with no message" from "report was lost"
		// (nil Errors would be ambiguous between the two cases).
		report.Reason = "turn_failed"
		msg := ""
		if errRaw, ok := fields["error"]; ok {
			msg = extractErrorMessage(errRaw)
		}
		if msg == "" {
			msg = "codex: turn.failed without error.message"
		}
		report.Errors = []string{msg}

	default:
		// Generic terminal event (IsTerminal was set by some other path).
		// Return a minimal populated report so the dispatcher knows it ended.
	}

	return report, true
}
