package components

// Centralized layout constants. Tweak the design by editing these — every
// component that lays out grids or fixed-width columns reads from here so
// the visual tuning lives in one place instead of scattered across files.
const (
	// LastActiveColWidth is the fixed reservation in each session list row
	// for the "- 7h 12m ago (Apr 2 14:23)" column. Sized for the worst
	// case ("- 99d 23h ago (Jan 22 23:59)" ≈ 28 cols).
	LastActiveColWidth = 28

	// MeterContentWidth is the visible width of one usage-meter line
	// (5h or 7d). Each meter is rendered at this width and centered
	// inside the surrounding tab block.
	MeterContentWidth = 16

	// OrgWrapThreshold — org names ≤ this stay on a single line; longer
	// multi-word orgs wrap onto two balanced lines.
	OrgWrapThreshold = 20

	// OrgNameMaxLineWidth caps how wide a single org-name line may be —
	// beyond this the line is truncated with an ellipsis.
	OrgNameMaxLineWidth = 30

	// TabSeparator is the inter-tab whitespace in the menu's org-tab
	// strip (and the meter row beneath it). Apply to both via the
	// orgtabs helpers.
	TabSeparator = "    "

	// MenuVerticalReserve is rows the menu screen reserves around the
	// session list (header + tabs + footer + spacers). Body height =
	// total - this - tab/meter heights.
	MenuVerticalReserve = 5
)
