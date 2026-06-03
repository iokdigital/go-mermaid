package svg

// Confidence color scheme matching the existing MermaidGenerator constants. FRD §6.8.
const (
	fillHigh      = "#90EE90"
	strokeHigh    = "#2d862d"
	fillMedium    = "#FFD700"
	strokeMedium  = "#b38f00"
	fillLow       = "#FFB6C1"
	strokeLow     = "#c0392b"
	fillNeutral   = "#f8fafc"
	strokeNeutral = "#64748b"

	defaultStrokeWidth = 1.5
	defaultFontSize    = 13
	defaultFontFamily  = "Arial, Helvetica, sans-serif"

	nodeWidth  = 120.0
	nodeHeight = 40.0

	arrowLen   = 8.0
	arrowWidth = 6.0
)

// nodeColors returns fill and stroke hex colors for a confidence value.
// Confidence == 0.0 means "not set" (neutral style).
func nodeColors(confidence float64) (fill, stroke string) {
	switch {
	case confidence >= 0.90:
		return fillHigh, strokeHigh
	case confidence >= 0.70:
		return fillMedium, strokeMedium
	case confidence > 0.0:
		return fillLow, strokeLow
	default:
		return fillNeutral, strokeNeutral
	}
}
