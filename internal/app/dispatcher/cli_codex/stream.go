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
//
// Full table-driven verification of codex event shapes lands in D5 once
// fixture data is available.
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
// Known codex event families (partially documented; pending D5 fixture):
//
//   - type="message"   — conversation message (role: "assistant" | "user")
//   - type="function_call" / kind="function_call" — tool invocation
//   - done=true / finish_reason present — terminal / end-of-stream event
//   - type="error"     — error event
//
// This function maps those families to the canonical vocabulary where
// possible; unknown shapes pass through with Type set to the raw type string
// (or empty if no type-like field is found).
func parseStreamEvent(line []byte) (dispatcher.StreamEvent, error) {
	// Always copy line into Raw before any decoding so the returned
	// StreamEvent.Raw is detached from the caller's buffer. Stream readers
	// commonly reuse a scanner buffer; retaining a slice header into reused
	// memory is a well-known bug vector.
	raw := append(json.RawMessage(nil), line...)

	ev := dispatcher.StreamEvent{Raw: raw}

	// Decode into a flexible map for permissive field access. Non-JSON input
	// returns the error sentinel but still provides Raw for the caller.
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(line, &fields); err != nil {
		return ev, fmt.Errorf("%w: %v", ErrMalformedStreamLine, err)
	}

	// Best-effort type normalization. codex event shapes are partially
	// undocumented; we look for common discriminator fields in priority order.
	// The D5 fixture drop will harden this mapping with table-driven tests.

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
	getBool := func(key string) bool {
		raw, ok := fields[key]
		if !ok || len(raw) == 0 {
			return false
		}
		var b bool
		if err := json.Unmarshal(raw, &b); err != nil {
			return false
		}
		return b
	}

	// Check for terminal markers first. codex signals end-of-stream via
	// a `done` boolean field or a `finish_reason` string field. Both are
	// treated as terminal regardless of the `type` field value.
	isDone := getBool("done")
	finishReason := getString("finish_reason")
	if isDone || finishReason != "" {
		ev.IsTerminal = true
		ev.Type = "result"
		if finishReason != "" {
			ev.Subtype = finishReason
		}
		return ev, nil
	}

	// Map `type` or `kind` fields to the canonical vocabulary.
	typeStr := getString("type")
	if typeStr == "" {
		typeStr = getString("kind")
	}
	ev.Type = typeStr

	switch typeStr {
	case "message":
		// codex message event — look for role and content fields.
		role := getString("role")
		switch role {
		case "assistant":
			ev.Type = "assistant"
			// Best-effort text extraction from content field.
			if contentRaw, ok := fields["content"]; ok {
				ev.Text = extractTextFromContent(contentRaw)
			}
		case "user":
			ev.Type = "user"
		default:
			ev.Type = "message"
		}

	case "function_call":
		// codex tool invocation event.
		ev.Type = "assistant"
		ev.ToolName = getString("name")
		if inputRaw, ok := fields["arguments"]; ok {
			ev.ToolInput = inputRaw
		}

	case "error":
		// codex error event — pass through with type="error"; non-terminal
		// because a terminal error event would carry done=true or finish_reason.
		ev.Type = "error"
		ev.Text = getString("message")

	default:
		// Unknown or absent type: pass through verbatim. The dispatcher's
		// monitor logs unknown event types; forward-compat for new codex
		// event families.
	}

	return ev, nil
}

// extractTextFromContent attempts to decode a codex `content` field into
// a plain text string. Content may be a plain string or an array of content
// blocks (as used in some codex event shapes). Returns empty string on any
// decode failure — best-effort only.
func extractTextFromContent(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	// Try plain string first (common in simple codex events).
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	// Try array of content blocks (OpenAI-style structured content).
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text,omitempty"`
	}
	if err := json.Unmarshal(raw, &blocks); err == nil {
		for _, b := range blocks {
			if b.Type == "text" && b.Text != "" {
				return b.Text
			}
		}
	}
	return ""
}

// extractTerminalReport returns the structured TerminalReport derived from a
// terminal StreamEvent. Returns (zero, false) for non-terminal events;
// callers MUST check the bool before consuming the report.
//
// The function re-decodes ev.Raw into a flexible map for permissive field
// access — codex's terminal event shape is partially undocumented. The D5
// fixture drop will harden the field mapping with table-driven tests.
//
// Cost extraction: codex may emit usage/cost in fields like `usage`,
// `total_cost_usd`, or `cost`. We make a best-effort attempt; nil Cost
// means "no cost telemetry found" per F.7.17 L11.
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

	// Finish reason maps to Reason.
	if raw, ok := fields["finish_reason"]; ok {
		var s string
		if json.Unmarshal(raw, &s) == nil {
			report.Reason = s
		}
	}
	if report.Reason == "" {
		// subtype may carry the reason (set by parseStreamEvent above).
		report.Reason = ev.Subtype
	}

	// Cost extraction: codex may use `total_cost_usd` (matching claude's
	// field name) or a nested usage structure. Try flat field first.
	if raw, ok := fields["total_cost_usd"]; ok {
		var cost float64
		if json.Unmarshal(raw, &cost) == nil {
			report.Cost = &cost
		}
	}

	// Errors: codex may emit an `error` string or `errors` array.
	if raw, ok := fields["errors"]; ok {
		var errs []string
		if json.Unmarshal(raw, &errs) == nil && len(errs) > 0 {
			report.Errors = errs
		}
	}
	if len(report.Errors) == 0 {
		if raw, ok := fields["error"]; ok {
			var errStr string
			if json.Unmarshal(raw, &errStr) == nil && errStr != "" {
				report.Errors = []string{errStr}
			}
		}
	}

	// Denials: codex does not currently document a permission_denials field;
	// report.Denials stays nil (no denials). Future drops can extend this
	// once the D5 fixture documents the actual codex terminal event shape.

	return report, true
}
