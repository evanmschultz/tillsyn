// Package style provides semantic design tokens for the Tillsyn TUI.
// MIGRATION TARGET: github.com/hylla-org/lykta
package style

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Semantic color tokens derived from the Hylla stil design system.
// Hex values map directly from stil tokens.css:
//
//	--carl (#dd9f57)            → Accent
//	--text-primary (#1c1e28)    → Primary (light)
//	--text-secondary (#545862)  → Secondary (light)
//	--text-tertiary (#6e7280)   → Muted (light)
//	--bg-chrome (#f9fafd)       → Background (light)
//	--bg-surface (#fcfcfc)      → Surface (light)
//
// Dark-mode equivalents follow the same mapping from tokens.css dark-theme block.
//
// In lipgloss v2, lipgloss.Color is a function returning color.Color; these
// vars are pre-computed color.Color values ready to pass to .Foreground() et al.

// Primary is the high-emphasis text color.
var Primary color.Color = lipgloss.Color("#1c1e28")

// Secondary is the medium-emphasis text color.
var Secondary color.Color = lipgloss.Color("#545862")

// Muted is the low-emphasis / tertiary text color (stil --text-tertiary).
var Muted color.Color = lipgloss.Color("#6e7280")

// Accent is the brand highlight color (stil --carl).
var Accent color.Color = lipgloss.Color("#dd9f57")

// Success is used for positive state indicators.
var Success color.Color = lipgloss.Color("#22863a")

// Warning is used for cautionary state indicators.
var Warning color.Color = lipgloss.Color("#b08800")

// Error is used for error and destructive state indicators.
var Error color.Color = lipgloss.Color("#cb2431")

// Background is the outermost chrome surface (stil --bg-chrome light).
var Background color.Color = lipgloss.Color("#f9fafd")

// Surface is the main content surface (stil --bg-surface light).
var Surface color.Color = lipgloss.Color("#fcfcfc")

// OnSurface is the default text rendered on the Surface background.
var OnSurface color.Color = Primary

// AllColors returns every semantic palette color in definition order.
// This function is the test anchor for the package coverage gate:
// pure-var/const packages with no executable statements cause the magefile
// coverage runner to error with "no coverage rows parsed."
func AllColors() []color.Color {
	return []color.Color{
		Primary,
		Secondary,
		Muted,
		Accent,
		Success,
		Warning,
		Error,
		Background,
		Surface,
		OnSurface,
	}
}
