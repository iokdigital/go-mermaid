package diagram

// RenderOptions configures renderer behaviour. Use NewRenderOptions() for defaults.
type RenderOptions struct {
	// Resolution selects the PNG output quality preset. Takes precedence over
	// Scale and DPI when set. See Resolution constants.
	Resolution Resolution

	// Scale is the PNG rasterization scale factor (1.0 = 96 DPI base).
	// Overridden by Resolution if both are set.
	Scale float64

	// DPI is stored in PNG file metadata only; it does not affect pixel dimensions.
	DPI int

	// MaxPNGBytes is the estimated maximum PNG output size in bytes.
	// When the estimate exceeds this value, RenderTo returns ErrPNGSizeLimitExceeded
	// with FallbackFormat() == FormatHTML.
	// Default: 10_000_000 (10 MB). Set to 0 to disable the limit.
	MaxPNGBytes int64

	// CDNOverrideURL replaces the default mermaid.js CDN URL in HTML output.
	// Useful for air-gapped environments pointing at an internally hosted copy.
	// Default: empty (uses https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.min.js).
	CDNOverrideURL string

	// SVGPadding is the whitespace added around the diagram content (px). Default: 40.
	SVGPadding int

	// SVGMaxWidth and SVGMaxHeight cap the computed SVG viewport. Defaults: 8000 × 6000.
	SVGMaxWidth  int
	SVGMaxHeight int

	// Layout fine-tunes the Sugiyama auto-layout algorithm used for flowcharts.
	Layout LayoutOptions
}

// Resolution is a named PNG output quality preset.
type Resolution string

const (
	// ResolutionWeb: 1× scale (~96 DPI) — standard browser or wiki embed.
	ResolutionWeb Resolution = "web"

	// ResolutionScreen: 2× scale (~192 DPI) — retina/HiDPI display (default).
	ResolutionScreen Resolution = "screen"

	// ResolutionScreenHD: 3× scale (~288 DPI) — high-density display, large monitor export.
	ResolutionScreenHD Resolution = "screen-hd"

	// ResolutionPrint: ~3.125× scale (300 DPI) — standard print quality.
	ResolutionPrint Resolution = "print"

	// ResolutionPrintHQ: ~4.167× scale (400 DPI) — high-quality print or press.
	ResolutionPrintHQ Resolution = "print-hq"
)

// ResolutionScale returns the rasterization scale factor for a given Resolution preset.
func ResolutionScale(r Resolution) float64 {
	switch r {
	case ResolutionWeb:
		return 1.0
	case ResolutionScreen:
		return 2.0
	case ResolutionScreenHD:
		return 3.0
	case ResolutionPrint:
		return 3.125
	case ResolutionPrintHQ:
		return 4.167
	default:
		return 2.0
	}
}

// ResolutionDPI returns the DPI metadata value for a given Resolution preset.
func ResolutionDPI(r Resolution) int {
	switch r {
	case ResolutionWeb:
		return 96
	case ResolutionScreen:
		return 192
	case ResolutionScreenHD:
		return 288
	case ResolutionPrint:
		return 300
	case ResolutionPrintHQ:
		return 400
	default:
		return 192
	}
}

// LayoutOptions configures the Sugiyama-style auto-layout algorithm for flowcharts.
type LayoutOptions struct {
	// NodeSpacingH is the horizontal gap between nodes in the same rank (px). Default: 60.
	NodeSpacingH int

	// NodeSpacingV is the vertical gap between nodes in the same rank (px). Default: 40.
	NodeSpacingV int

	// RankSpacing is the gap between rank layers (px). Default: 80.
	RankSpacing int
}

// NewRenderOptions returns a RenderOptions with production-appropriate defaults.
func NewRenderOptions() RenderOptions {
	return RenderOptions{
		Resolution:   ResolutionScreen,
		MaxPNGBytes:  10_000_000,
		SVGPadding:   40,
		SVGMaxWidth:  8000,
		SVGMaxHeight: 6000,
		Layout:       DefaultLayoutOptions(),
	}
}

// DefaultLayoutOptions returns the default Sugiyama layout spacing values.
func DefaultLayoutOptions() LayoutOptions {
	return LayoutOptions{
		NodeSpacingH: 60,
		NodeSpacingV: 40,
		RankSpacing:  80,
	}
}
