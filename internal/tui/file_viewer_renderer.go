package tui

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// fileRendererKind classifies how a file will be rendered in the viewer.
type fileRendererKind int

const (
	fileRendererMarkdown fileRendererKind = iota
	fileRendererCode
	fileRendererPlain
)

// classifyExtension maps a lowercase filename extension to a rendering kind and
// an optional chroma lexer name. It uses a pure switch — no regex.
func classifyExtension(lowerExt string) (fileRendererKind, string) {
	switch lowerExt {
	case ".md", ".markdown":
		return fileRendererMarkdown, ""
	case ".go":
		return fileRendererCode, "go"
	case ".js":
		return fileRendererCode, "javascript"
	case ".ts":
		return fileRendererCode, "typescript"
	case ".rs":
		return fileRendererCode, "rust"
	case ".py":
		return fileRendererCode, "python"
	case ".sh":
		return fileRendererCode, "bash"
	case ".toml":
		return fileRendererCode, "toml"
	case ".yaml", ".yml":
		return fileRendererCode, "yaml"
	case ".json":
		return fileRendererCode, "json"
	default:
		return fileRendererPlain, ""
	}
}

// chooseRenderer renders content bytes for the named file using the appropriate
// pipeline. Markdown files are rendered via the shared glamour-backed
// markdownRenderer so styling is consistent with the thread view; code files are
// syntax-highlighted via chroma (dracula style, terminal256 formatter); all
// other files are passed through as-is.
//
// chooseRenderer is pure: it has no side effects and does not mutate any shared
// state. The glamour renderer backing md is safe for concurrent use.
func chooseRenderer(filename string, content []byte, md *markdownRenderer) (string, error) {
	base := filepath.Base(filename)
	ext := strings.ToLower(filepath.Ext(base))
	kind, lexerName := classifyExtension(ext)

	switch kind {
	case fileRendererMarkdown:
		if md == nil {
			return string(content), nil
		}
		rendered := md.render(string(content), 80)
		if rendered == "" {
			return string(content), nil
		}
		return rendered, nil
	case fileRendererCode:
		return renderCodeContent(content, lexerName)
	default:
		return string(content), nil
	}
}

// chooseRendererWithFuncs is the test-injectable variant of chooseRenderer.
// It accepts injected render functions so unit tests can pin glamour and chroma
// output without constructing live renderers — keeping golden fixtures ANSI-free
// and environment-independent.
//
// Production code should use chooseRenderer; tests use this variant.
func chooseRendererWithFuncs(
	filename string,
	content []byte,
	renderMD func(string, int) string,
	renderCode func([]byte, string) (string, error),
) (string, error) {
	base := filepath.Base(filename)
	ext := strings.ToLower(filepath.Ext(base))
	kind, lexerName := classifyExtension(ext)
	switch kind {
	case fileRendererMarkdown:
		if renderMD == nil {
			return string(content), nil
		}
		return renderMD(string(content), 80), nil
	case fileRendererCode:
		if renderCode == nil {
			return string(content), nil
		}
		return renderCode(content, lexerName)
	default:
		return string(content), nil
	}
}

// renderCodeContent syntax-highlights content bytes using chroma's terminal256
// formatter with the dracula style. lexerName is the chroma language identifier
// (e.g. "go", "python"). Falls back to an error on tokenise/format failure.
func renderCodeContent(content []byte, lexerName string) (string, error) {
	lexer := lexers.Get(lexerName)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}
	style := styles.Get("dracula")
	if style == nil {
		style = styles.Fallback
	}
	iterator, err := lexer.Tokenise(nil, string(content))
	if err != nil {
		return "", fmt.Errorf("file viewer: tokenise %s: %w", lexerName, err)
	}
	var buf bytes.Buffer
	if err := formatter.Format(&buf, style, iterator); err != nil {
		return "", fmt.Errorf("file viewer: format %s: %w", lexerName, err)
	}
	return buf.String(), nil
}
