package cli_claude

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/evanmschultz/tillsyn/internal/app/dispatcher"
)

// ErrMalformedStreamLine is returned by parseStreamEvent when the input
// line is not valid JSON or lacks the discriminator `type` field.
// Monitors log the error but do NOT halt the spawn — claude streams may
// (rarely) emit interleaved progress lines the canonical taxonomy doesn't
// cover, and we want forward-compat for new event types.
var ErrMalformedStreamLine = errors.New("cli_claude: malformed stream-json line")

// claudeEventDiscriminator captures the minimal shape needed to dispatch
// one stream-json line into the cross-CLI canonical StreamEvent vocabulary.
// Adapter-private fields (model, usage, session_id, …) are intentionally
// NOT decoded here — they live inside Raw and are extracted by
// extractTerminalReport (or by future forensic tooling) on demand.
type claudeEventDiscriminator struct {
	Type    string          `json:"type"`
	Subtype string          `json:"subtype,omitempty"`
	Message json.RawMessage `json:"message,omitempty"`
}

// claudeAssistantMessage decodes the Message field of an "assistant"
// event so we can pull the first text + tool_use block into the canonical
// StreamEvent.Text / ToolName / ToolInput fields. Per the
// project_drop_4c_spawn_architecture.md §6.2 taxonomy each assistant
// event's content is a list of blocks; we surface the first text block
// and the first tool_use block. Multiple-block events are rare but we
// degrade cleanly (emit only the first match of each kind).
type claudeAssistantMessage struct {
	Content []claudeContentBlock `json:"content"`
}

// claudeContentBlock is the shared shape across the three documented
// content-block flavors (thinking | text | tool_use). All fields are
// optional; the Type discriminator selects which fields are populated.
type claudeContentBlock struct {
	Type     string          `json:"type"`
	Text     string          `json:"text,omitempty"`
	Thinking string          `json:"thinking,omitempty"`
	Name     string          `json:"name,omitempty"`
	Input    json.RawMessage `json:"input,omitempty"`
}

// claudeResultEvent decodes the terminal "result" event's payload into
// the fields ExtractTerminalReport surfaces. Per memory §6.4 the result
// event carries far more (duration_ms, stop_reason, usage, modelUsage,
// fast_mode_state, …); we decode ONLY the fields TerminalReport exposes.
// Forensic tooling can re-decode StreamEvent.Raw for the rest.
type claudeResultEvent struct {
	TotalCostUSD      *float64                 `json:"total_cost_usd,omitempty"`
	PermissionDenials []claudePermissionDenial `json:"permission_denials,omitempty"`
	TerminalReason    string                   `json:"terminal_reason,omitempty"`
	Errors            []string                 `json:"errors,omitempty"`
}

// claudePermissionDenial is one entry in the result event's
// permission_denials[] array. We retain ToolUseID inside the raw input
// blob (not as a separate field) because dispatcher.ToolDenial only
// surfaces ToolName + ToolInput; the tool_use_id is opaque from the
// dispatcher's perspective.
type claudePermissionDenial struct {
	ToolName  string          `json:"tool_name"`
	ToolInput json.RawMessage `json:"tool_input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
}

// parseStreamEvent decodes one JSONL line emitted by `claude
// --output-format stream-json` into the cross-CLI canonical StreamEvent.
// The full raw bytes are retained in Raw so ExtractTerminalReport (and
// forensic post-mortem tooling) can re-decode adapter-private fields
// without re-reading from disk.
//
// Maps the four documented event families per memory §6:
//
//   - {"type":"system","subtype":"init",...}      → Type="system_init"
//   - {"type":"assistant","message":{...}}        → Type="assistant" + extracted text/tool_use
//   - {"type":"user","message":{...}}             → Type="user" (tool_result events)
//   - {"type":"result","subtype":"...",...}       → Type="result" + IsTerminal=true
//
// Unknown event types pass through with Type set to the raw discriminator
// string and IsTerminal = false. Callers that depend on canonical-name
// matching MUST check explicit constants, not assume any string.
func parseStreamEvent(line []byte) (dispatcher.StreamEvent, error) {
	// Copy line so the returned StreamEvent.Raw is detached from the
	// caller's buffer. Stream readers commonly reuse a scanner buffer;
	// retaining a slice header into reused memory is a well-known bug
	// vector. Allocating once per line is cheap relative to JSON decode.
	raw := append(json.RawMessage(nil), line...)

	var disc claudeEventDiscriminator
	if err := json.Unmarshal(line, &disc); err != nil {
		return dispatcher.StreamEvent{Raw: raw}, fmt.Errorf("%w: %v", ErrMalformedStreamLine, err)
	}
	if disc.Type == "" {
		return dispatcher.StreamEvent{Raw: raw}, fmt.Errorf("%w: missing \"type\" field", ErrMalformedStreamLine)
	}

	ev := dispatcher.StreamEvent{
		Raw:     raw,
		Subtype: disc.Subtype,
	}

	switch disc.Type {
	case "system":
		// claude emits a single system event today (subtype=init). We map
		// it to "system_init" for the canonical vocabulary; if claude
		// adds non-init system events later, we route them here too with
		// the subtype carried through.
		if disc.Subtype == "init" {
			ev.Type = "system_init"
		} else {
			// Unknown system subtype — pass through with raw type so the
			// monitor surfaces the gap.
			ev.Type = "system"
		}
	case "assistant":
		ev.Type = "assistant"
		if len(disc.Message) > 0 {
			var msg claudeAssistantMessage
			// Best-effort decode of the assistant message: a malformed
			// inner shape does not invalidate the whole event (the
			// discriminator decoded fine). We just leave Text / ToolName
			// empty and let the caller see the Raw bytes.
			if err := json.Unmarshal(disc.Message, &msg); err == nil {
				populateAssistantBlocks(&ev, msg.Content)
			}
		}
	case "user":
		ev.Type = "user"
		// Tool_result events live under user.message.content[].
		// Surfacing them is not required by F.7.17 (the monitor cares
		// about terminal reports + tool denials surfaced ON the
		// terminal event) so we map type-only and let downstream
		// readers re-decode Raw if they want per-tool detail.
	case "result":
		ev.Type = "result"
		ev.IsTerminal = true
	default:
		// Forward-compat: unknown event types pass through verbatim.
		ev.Type = disc.Type
	}

	return ev, nil
}

// populateAssistantBlocks scans the content blocks of an "assistant"
// message and sets ev.Text from the first text block and ev.ToolName +
// ev.ToolInput from the first tool_use block. Subsequent matches are
// ignored — the canonical StreamEvent surfaces only the first of each
// kind. Forensic tooling can re-decode Raw for the full block list.
func populateAssistantBlocks(ev *dispatcher.StreamEvent, blocks []claudeContentBlock) {
	textSet := false
	toolSet := false
	for _, b := range blocks {
		switch b.Type {
		case "text":
			if !textSet {
				ev.Text = b.Text
				textSet = true
			}
		case "tool_use":
			if !toolSet {
				ev.ToolName = b.Name
				ev.ToolInput = b.Input
				toolSet = true
			}
		}
		if textSet && toolSet {
			return
		}
	}
}

// extractTerminalReport returns the structured TerminalReport derived
// from a terminal StreamEvent. Returns (zero, false) for non-terminal
// events; callers MUST check the bool before consuming the report.
//
// The function re-decodes ev.Raw into a claudeResultEvent — the parser
// retained the raw bytes in StreamEvent.Raw exactly for this purpose.
// Adapter-private result-event fields (duration_ms, num_turns, usage,
// modelUsage, …) are intentionally NOT surfaced; if a future Tillsyn
// drop wants per-spawn token telemetry, it adds fields to TerminalReport
// and threads them through here.
func extractTerminalReport(ev dispatcher.StreamEvent) (dispatcher.TerminalReport, bool) {
	if !ev.IsTerminal {
		return dispatcher.TerminalReport{}, false
	}

	var result claudeResultEvent
	// Best-effort decode: a malformed terminal event still returns
	// (zero, true) so the dispatcher knows the spawn ended, even if the
	// payload was unparseable. The dispatcher's monitor is responsible
	// for logging the parse failure separately.
	if len(ev.Raw) > 0 {
		_ = json.Unmarshal(ev.Raw, &result)
	}

	report := dispatcher.TerminalReport{
		Cost:    result.TotalCostUSD,
		Reason:  result.TerminalReason,
		Errors:  result.Errors,
		Denials: convertDenials(result.PermissionDenials),
	}
	return report, true
}

// convertDenials lifts claude's permission_denials[] payload into the
// cross-CLI dispatcher.ToolDenial slice. Returns nil (not empty) when
// the input is empty so callers comparing against nil semantically work.
func convertDenials(in []claudePermissionDenial) []dispatcher.ToolDenial {
	if len(in) == 0 {
		return nil
	}
	out := make([]dispatcher.ToolDenial, 0, len(in))
	for _, d := range in {
		out = append(out, dispatcher.ToolDenial{
			ToolName:  d.ToolName,
			ToolInput: d.ToolInput,
		})
	}
	return out
}
