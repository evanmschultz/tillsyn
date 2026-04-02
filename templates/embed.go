package templates

import (
	"embed"
	"io/fs"
)

// Files exposes the repo-visible builtin template sources embedded into the binary at build time.
//
//go:embed builtin/*.json
var Files embed.FS

// ReadFile loads one embedded template source file.
func ReadFile(name string) ([]byte, error) {
	return fs.ReadFile(Files, name)
}
