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

	// CDNOverrideURL replaces the default client-side renderer CDN URL in HTML
	// output. When HTMLEngine is "pele" (default) it replaces the Pele URL;
	// when "mermaid" it replaces the mermaid.js URL (legacy behaviour).
	// Useful for air-gapped environments pointing at an internally hosted copy.
	// Default: empty (uses the engine's default jsdelivr URL below).
	// Must start with "https://" or "http://" when set.
	CDNOverrideURL string

	// HTMLEngine selects the client-side renderer used in HTML output:
	// HTMLEnginePele (default) or HTMLEngineMermaid (legacy mermaid.js-only
	// page, no Pele, no fallback). Empty means HTMLEnginePele.
	HTMLEngine string

	// HTMLFallbackCDNURL replaces the fallback mermaid.js CDN URL loaded by
	// HTML output when the Pele CDN script fails to load. Only used when
	// HTMLEngine is "pele". Default: empty (uses defaultMermaidCDNURL).
	// Must start with "https://" or "http://" when set.
	HTMLFallbackCDNURL string

	// SVGPadding is the whitespace added around the diagram content (px). Default: 40.
	SVGPadding int

	// SVGMaxWidth and SVGMaxHeight cap the computed SVG viewport. Defaults: 8000 × 6000.
	SVGMaxWidth  int
	SVGMaxHeight int

	// Layout fine-tunes the Sugiyama auto-layout algorithm used for flowcharts.
	Layout LayoutOptions
}

// HTML engine selectors for RenderOptions.HTMLEngine.
const (
	// HTMLEnginePele renders HTML output with Pele (default) and falls back
	// to mermaid.js when the Pele CDN script fails to load.
	HTMLEnginePele = "pele"

	// HTMLEngineMermaid renders legacy mermaid.js-only HTML output.
	HTMLEngineMermaid = "mermaid"
)

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
		HTMLEngine:   HTMLEnginePele,
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
