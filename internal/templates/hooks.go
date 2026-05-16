package templates

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// hookTemplatePath is the embed-relative path to the validate-action-item-paths
// hook template within DefaultTemplateFS.
const hookTemplatePath = "builtin/hooks/validate-action-item-paths.sh.tmpl"

// ComputeHookHash returns the lowercase-hex sha256 of the embedded
// validate-action-item-paths.sh.tmpl content. Used by till init and dispatcher
// preflight to detect drift between the embedded template and the on-disk
// rendered hook script.
//
// The embed.FS type is documented thread-safe for concurrent reads (Go std-lib
// contract); no sync.Once cache is needed. The function returns a fresh
// computation on every call — callers that need a cached value should cache it
// themselves.
//
// The error return is reserved for programmer-error scenarios (e.g. the
// //go:embed directive is mismatched with the on-disk path). In normal operation
// with the file correctly embedded, the error is always nil.
func ComputeHookHash() (string, error) {
	data, err := DefaultTemplateFS.ReadFile(hookTemplatePath)
	if err != nil {
		return "", fmt.Errorf("templates: ComputeHookHash: read embedded %q: %w", hookTemplatePath, err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
