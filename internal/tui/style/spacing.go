// Package style provides semantic design tokens for the Tillsyn TUI.
// MIGRATION TARGET: github.com/hylla-org/lykta
package style

// Spacing constants follow a 4-step scale suitable for lipgloss
// padding/margin integer arguments. Values are in terminal cell units.
//
// Mapping to stil tokens.css --space-* (4px base):
//
//	SpaceXS = 0  (no spacing / flush layout)
//	SpaceSM = 1  (~0.25rem / 4px equivalent: 1 cell)
//	SpaceMD = 2  (~0.5rem  / 8px equivalent: 2 cells)
//	SpaceLG = 4  (~1rem    / 16px equivalent: 4 cells)
//	SpaceXL = 6  (~1.5rem  / 24px equivalent: 6 cells)

// SpaceXS is zero spacing — flush layout, no padding or margin.
const SpaceXS = 0

// SpaceSM is a single-cell spacing unit — compact insets.
const SpaceSM = 1

// SpaceMD is a two-cell spacing unit — standard insets and gaps.
const SpaceMD = 2

// SpaceLG is a four-cell spacing unit — section-level padding.
const SpaceLG = 4

// SpaceXL is a six-cell spacing unit — large structural gaps.
const SpaceXL = 6
