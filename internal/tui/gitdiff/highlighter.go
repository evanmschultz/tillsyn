package gitdiff

import (
	"bytes"
	"fmt"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// Highlighter renders a raw unified-diff patch into an ANSI-styled string
// suitable for direct terminal rendering in the TUI diff pane.
//
// The interface is consumer-side: the diff pane in internal/tui binds to
// Highlighter rather than any concrete type, so alternate implementations
// (plain passthrough for NO_COLOR, snapshot replay for tests) can be wired in
// without touching call sites. Implementations must be safe for concurrent use
// by multiple goroutines.
type Highlighter interface {
	// Highlight converts patch text into ANSI-styled output. An empty patch
	// returns an empty string and a nil error without invoking any downstream
	// highlighter. Errors are wrapped with %w so callers can detect
	// highlighter-specific failures via errors.Is/As.
	Highlight(patch string) (string, error)
}

// chromaHighlighter is the default Highlighter implementation backed by
// the chroma v2 syntax-highlighting library.
//
// The lexer, formatter, and style are resolved once at construction time and
// reused for every Highlight call. chroma's lexer, formatter, and style values
// are immutable after lookup and are documented as safe for concurrent reuse,
// which is what makes a single Highlighter instance shareable across
// goroutines in the TUI.
type chromaHighlighter struct {
	lexer     chroma.Lexer
	formatter chroma.Formatter
	style     *chroma.Style
}

// NewChromaHighlighter returns a Highlighter backed by chroma's "diff" lexer,
// the "terminal256" formatter, and the "dracula" style. Each lookup falls back
// to chroma's documented Fallback values when the registry returns nil, so the
// constructor never fails on a well-formed chroma build.
//
// The returned Highlighter is safe for concurrent use by multiple goroutines.
func NewChromaHighlighter() Highlighter {
	lexer := lexers.Get("diff")
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
	return &chromaHighlighter{
		lexer:     lexer,
		formatter: formatter,
		style:     style,
	}
}

// Highlight renders patch through the configured chroma pipeline.
//
// An empty patch is returned verbatim with a nil error; chroma is never
// invoked for the empty case so the hot path of "no changes" stays cheap.
// Tokenise and Format failures are wrapped with %w and surfaced to the
// caller without partial output — the TUI renders a plain-text fallback when
// Highlight returns an error, so leaking a half-styled buffer would be
// actively misleading.
func (h *chromaHighlighter) Highlight(patch string) (string, error) {
	if patch == "" {
		return "", nil
	}
	iterator, err := h.lexer.Tokenise(nil, patch)
	if err != nil {
		return "", fmt.Errorf("gitdiff: tokenise patch: %w", err)
	}
	var buf bytes.Buffer
	if err := h.formatter.Format(&buf, h.style, iterator); err != nil {
		return "", fmt.Errorf("gitdiff: format patch: %w", err)
	}
	return buf.String(), nil
}
