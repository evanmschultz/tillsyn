package mcpcommon

import "errors"

// ErrInvalidRequest is the sentinel wrapped by adapter errors that should
// surface as MCP error class "invalid" + code "invalid_request" at the
// transport boundary (mapToolError).
var ErrInvalidRequest = errors.New("invalid request")
