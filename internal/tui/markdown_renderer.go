package tui

import (
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
)

// markdownRenderer renders markdown for terminal views and recreates the renderer when wrap width changes.
type markdownRenderer struct {
	width    int
	renderer *glamour.TermRenderer
}

// render converts markdown input into ANSI-styled terminal text with the requested wrap width.
func (r *markdownRenderer) render(markdown string, width int) string {
	markdown = strings.TrimSpace(markdown)
	if markdown == "" {
		return ""
	}

	wrapWidth := width
	if wrapWidth < 24 {
		wrapWidth = 24
	}

	if r.renderer == nil || r.width != wrapWidth {
		renderer, err := glamour.NewTermRenderer(
			// Use a stable built-in style to avoid terminal environment probing
			// sequences leaking into interactive TUI input streams.
			glamour.WithStandardStyle(styles.DarkStyle),
			glamour.WithWordWrap(wrapWidth),
		)
		if err != nil {
			return markdown
		}
		r.renderer = renderer
		r.width = wrapWidth
	}

	rendered, err := r.renderer.Render(markdown)
	if err != nil {
		return markdown
	}
	return strings.TrimRight(rendered, "\n")
}
