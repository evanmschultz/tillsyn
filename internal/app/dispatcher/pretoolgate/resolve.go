package pretoolgate

import (
	"context"
	"encoding/json"
	"os"
	"regexp"
	"strings"
)

// ResolveAllowlist determines the allowlist for a scoped agent by trying three
// sources in precedence order:
//  1. env var TILL_GATE_ALLOWLIST (JSON) — for subprocess dispatches
//  2. parent transcript scan by agent_type — for built-in Agent-tool subagents
//  3. neither → return nil (defer/ungated)
//
// The function returns (*GateSpec, nil) on successful resolution. If no allowlist
// is found or if the lookup fails gracefully (fail-open), it returns (nil, nil).
//
// Errors are NOT returned for parse failures in the env or transcript —
// this implements fail-open: if the spec is malformed, the resolver defers to
// an ungated state. The Go contract is simpler than the Python one: the caller
// (A.1 Event) decides whether to gate based on the non-nil return, not a
// boolean flag.
func ResolveAllowlist(ctx context.Context, agentID, agentType, transcriptPath string) *GateSpec {
	// Source 1: env var TILL_GATE_ALLOWLIST (renamed from Python's TA_GATE_ALLOWLIST).
	if spec := resolveFromEnv(); spec != nil {
		return spec
	}

	// Source 2: parent transcript scan.
	// If agentID is empty and we have no env, we're an ungated caller (orchestrator/main session).
	if agentID == "" {
		return nil
	}

	// Try transcript scan by agent_type.
	if spec := resolveFromTranscript(transcriptPath, agentType); spec != nil {
		return spec
	}

	// Source 3: neither → ungated.
	return nil
}

// resolveFromEnv returns a GateSpec parsed from env var TILL_GATE_ALLOWLIST,
// or nil if the env var is absent, empty, or contains invalid JSON.
func resolveFromEnv() *GateSpec {
	raw, ok := os.LookupEnv("TILL_GATE_ALLOWLIST")
	if !ok || strings.TrimSpace(raw) == "" {
		return nil
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		// Fail open: malformed JSON means no spec from env.
		return nil
	}

	return mapToGateSpec(data)
}

// resolveFromTranscript scans a JSONL transcript file for the most recent
// dispatch of the given agentType that carries a <TA_ALLOWLIST> block,
// returning the parsed GateSpec or nil if no matching dispatch is found.
//
// The function implements last-match-wins semantics: it continues scanning
// the entire file and keeps the most recent match, matching the Python
// behavior where the most recent dispatch's allowlist takes precedence
// (handles concurrent same-role dispatches).
func resolveFromTranscript(transcriptPath, agentType string) *GateSpec {
	if transcriptPath == "" || agentType == "" {
		return nil
	}

	// Cheap check: does the file exist?
	if _, err := os.Stat(transcriptPath); err != nil {
		return nil
	}

	data, err := os.ReadFile(transcriptPath)
	if err != nil {
		return nil
	}

	// Scan line-by-line for JSON events.
	var lastSpec *GateSpec
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Cheap pre-filter: only parse lines that could carry a TA_ALLOWLIST block.
		if !strings.Contains(line, "TA_ALLOWLIST") {
			continue
		}

		// Parse the event JSON.
		var evt map[string]interface{}
		if err := json.Unmarshal([]byte(line), &evt); err != nil {
			continue
		}

		// Navigate to the message.content[] blocks.
		msg, ok := evt["message"].(map[string]interface{})
		if !ok {
			continue
		}

		content, ok := msg["content"].([]interface{})
		if !ok {
			continue
		}

		// Look for a tool_use block with name "Agent" or "Task" and matching subagent_type.
		for _, blkIface := range content {
			blk, ok := blkIface.(map[string]interface{})
			if !ok {
				continue
			}

			if blk["type"] != "tool_use" {
				continue
			}

			toolName, ok := blk["name"].(string)
			if !ok || (toolName != "Agent" && toolName != "Task") {
				continue
			}

			inp, ok := blk["input"].(map[string]interface{})
			if !ok {
				continue
			}

			subagentType, ok := inp["subagent_type"].(string)
			if !ok || subagentType != agentType {
				continue
			}

			prompt, ok := inp["prompt"].(string)
			if !ok {
				continue
			}

			// Extract the <TA_ALLOWLIST> block from the prompt using regex.
			spec := extractAllowlistFromPrompt(prompt)
			if spec != nil {
				lastSpec = spec // Keep scanning; last match wins.
			}
		}
	}

	return lastSpec
}

// extractAllowlistFromPrompt uses regex to find and parse the <TA_ALLOWLIST>
// block in a prompt string, returning a GateSpec or nil.
//
// The regex matches <TA_ALLOWLIST> ... </TA_ALLOWLIST> with DOTALL mode
// (. matches newlines), capturing the JSON inside.
func extractAllowlistFromPrompt(prompt string) *GateSpec {
	// Regex pattern: <TA_ALLOWLIST> followed by optional whitespace, then JSON, then </TA_ALLOWLIST>
	// Using DOTALL mode so . matches newlines.
	pattern := regexp.MustCompile(`(?s)<TA_ALLOWLIST>\s*(\{.*?\})\s*</TA_ALLOWLIST>`)

	matches := pattern.FindStringSubmatch(prompt)
	if len(matches) < 2 {
		return nil
	}

	jsonStr := matches[1]
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil
	}

	return mapToGateSpec(data)
}

// mapToGateSpec converts a map[string]interface{} (from JSON decode) into a
// GateSpec, handling the nil-vs-empty distinction for slices.
//
// The key property: if a key is absent from the JSON, the corresponding field
// in GateSpec is nil; if the key is present with an empty array [], the field
// is a non-nil empty slice. This is the default behavior of Go's json.Unmarshal
// for slice fields (absent → nil, empty array → empty non-nil slice).
func mapToGateSpec(data map[string]interface{}) *GateSpec {
	spec := &GateSpec{}

	// Extract "edit" and "bash_deny" fields, preserving nil vs empty-slice distinction.
	// Go's json.Unmarshal already does this for us: an absent key leaves the field nil,
	// a present empty array [] unmarshals to a non-nil empty slice.

	if editIface, ok := data["edit"]; ok {
		if editList, ok := editIface.([]interface{}); ok {
			// Convert []interface{} to []string.
			edit := make([]string, len(editList))
			for i, v := range editList {
				if s, ok := v.(string); ok {
					edit[i] = s
				}
			}
			spec.Edit = edit
		}
	}

	if bashDenyIface, ok := data["bash_deny"]; ok {
		if bashDenyList, ok := bashDenyIface.([]interface{}); ok {
			// Convert []interface{} to []string.
			bashDeny := make([]string, len(bashDenyList))
			for i, v := range bashDenyList {
				if s, ok := v.(string); ok {
					bashDeny[i] = s
				}
			}
			spec.BashDeny = bashDeny
		}
	}

	return spec
}
