// Package dispatcher — Claude-Code → server-registered MCP tool-name conversion.
//
// Background
//
// Claude-Code exposes MCP tools to its own internal model namespace using the
// flat `mcp__<server>__<tail>` form (e.g. `mcp__hylla__hylla_search_vector`).
// Codex (and the live MCP servers themselves) expect the canonical name the
// server actually registered (e.g. `hylla.search.vector`). Different servers
// register their tools under different conventions:
//
//   - hylla   registers tools with dots:        hylla.search.vector
//   - tillsyn registers tools with dots:        till.action_item
//   - gopls   registers tools with underscores: go_search
//   - ta      registers tools tail-only:        get, update, list_sections
//
// Without correct conversion, Codex's per-tool approval lookup fails and tools
// auto-cancel silently under `--ephemeral` mode. This file is the hardcoded
// two-tier resolution surface (tier 1 = the lookup below; tier 2 = a live
// JSON-RPC `tools/list` probe deferred to a follow-up droplet — see
// canonical reference).
//
// Canonical reference
//
//   ~/.claude/codex-mcp-dispatch-tool-conversion.md (lines 19-48 — the
//   verbatim conversion table reproduced in the var declaration + tests).
//
// Upstream Codex issues that motivated the conversion rule
//
//   - codex/issues/15437  (per-tool approval lookup vs flat namespace)
//   - codex/issues/15753  (silent auto-cancel under --ephemeral)
//   - codex/issues/16501  (tool name canonicalization across servers)
//   - codex/issues/19430  (mcp__<server>__<tail> prefix handling)
//   - codex/issues/13476  (per-server registration convention discovery)
//
// All four servers below are dogfood-critical for the Tillsyn cascade.
// New servers should land in the table (and the test) together; unknown
// servers return ErrUnknownMCPServer rather than silently mis-converting.
package dispatcher

import (
	"errors"
	"fmt"
	"strings"
)

// claudePrefix is the Claude-Code MCP tool-name leading marker. Every
// Claude-Code MCP tool name begins with `mcp__` followed by `<server>__<tail>`.
const claudePrefix = "mcp__"

// ErrUnknownMCPServer is returned by ConvertClaudeToolName when the parsed
// server segment is not in the hardcoded table. Live JSON-RPC tools/list
// probing for unknown servers is deferred — see file head comment.
var ErrUnknownMCPServer = errors.New("dispatcher: unknown MCP server in Claude tool name")

// ErrInvalidClaudeToolName is returned when the input does not have the
// `mcp__<server>__<tail>` shape (e.g. missing prefix, missing tail).
var ErrInvalidClaudeToolName = errors.New("dispatcher: invalid Claude MCP tool name (expected mcp__<server>__<tail>)")

// toolFormat describes how to map the Claude-Code `<tail>` segment of a
// `mcp__<server>__<tail>` tool name to the canonical name the MCP server
// itself registered. Two orthogonal switches cover all four dogfood servers:
//
//   - underscoreToDot: rewrite every `_` in the tail to `.`. Required for
//     hylla, whose Claude form uses underscores but whose server-registered
//     name uses dots (e.g. `hylla_search_vector` → `hylla.search.vector`).
//
//   - tailVerbatim: keep the tail exactly as Claude-Code presents it. Used
//     for tillsyn (which already keeps dots through Claude-Code's namespace),
//     gopls (which registers with underscores), and ta (whose canonical names
//     are tail-only and already underscore-form).
//
// underscoreToDot=true with tailVerbatim=true is contradictory; only one
// switch is set per server.
type toolFormat struct {
	underscoreToDot bool
	tailVerbatim    bool
}

// serverToolFormat is the hardcoded table from
// ~/.claude/codex-mcp-dispatch-tool-conversion.md lines 19-48. Adding a new
// MCP server requires (1) a row here and (2) verbatim test rows in
// tool_name_conversion_test.go covering every tool that server registers.
var serverToolFormat = map[string]toolFormat{
	"tillsyn": {tailVerbatim: true},
	"hylla":   {underscoreToDot: true},
	"ta":      {tailVerbatim: true},
	"gopls":   {tailVerbatim: true},
}

// ConvertClaudeToolName parses a Claude-Code `mcp__<server>__<tail>` MCP tool
// name and returns the server segment plus the canonical tool name as
// registered by that MCP server. Lookup is case-sensitive and table-driven
// per the conversion doc.
//
// Returns ErrInvalidClaudeToolName if the input is missing the `mcp__` prefix
// or the `<server>__<tail>` body. Returns ErrUnknownMCPServer if the server
// segment is not in serverToolFormat.
//
// Examples (every row mirrors the conversion-doc table verbatim):
//
//	mcp__hylla__hylla_search_vector   → ("hylla",   "hylla.search.vector",  nil)
//	mcp__hylla__hylla_artifact_overview → ("hylla", "hylla.artifact.overview", nil)
//	mcp__ta__get                       → ("ta",     "get",                  nil)
//	mcp__ta__update                    → ("ta",     "update",               nil)
//	mcp__gopls__go_search              → ("gopls",  "go_search",            nil)
//	mcp__tillsyn__till.attention_item  → ("tillsyn","till.attention_item",  nil)
//	mcp__unknown__foo                  → ("", "", ErrUnknownMCPServer)
//	(no mcp__ prefix)                  → ("", "", ErrInvalidClaudeToolName)
func ConvertClaudeToolName(claudeForm string) (server, canonical string, err error) {
	if !strings.HasPrefix(claudeForm, claudePrefix) {
		return "", "", fmt.Errorf("%w: %q", ErrInvalidClaudeToolName, claudeForm)
	}
	body := strings.TrimPrefix(claudeForm, claudePrefix)
	// The server / tail split is on the FIRST `__` occurrence in the body —
	// the tail itself may legitimately contain `__` (rare but possible) so we
	// must not SplitN with -1 and assume two parts.
	sepIdx := strings.Index(body, "__")
	if sepIdx < 1 || sepIdx == len(body)-2 {
		return "", "", fmt.Errorf("%w: %q", ErrInvalidClaudeToolName, claudeForm)
	}
	srv := body[:sepIdx]
	tail := body[sepIdx+len("__"):]
	fmtSpec, ok := serverToolFormat[srv]
	if !ok {
		return "", "", fmt.Errorf("%w: server=%q (input=%q)", ErrUnknownMCPServer, srv, claudeForm)
	}
	if fmtSpec.underscoreToDot {
		return srv, strings.ReplaceAll(tail, "_", "."), nil
	}
	// tailVerbatim or no-switch default: pass the tail through unchanged.
	return srv, tail, nil
}
