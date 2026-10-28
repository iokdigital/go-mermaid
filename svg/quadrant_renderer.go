package svg

import (
	"fmt"
	"io"
	"math"
	"strings"

	diagram "github.com/iokdigital/go-mermaid"
	"github.com/iokdigital/go-mermaid/ast"
)

// Quadrant chart geometry.
const (
	quadGrid    = 400.0 // square plot area
	quadMarginL = 110.0 // room for y-axis labels
	quadMarginT = 20.0
	quadMarginR = 24.0
	quadMarginB = 56.0 // room for x-axis labels
	quadPointR  = 5.0
)

// encodeQuadrant renders a *ast.QuadrantDiagram to w as an SVG document.
// Quadrant numbering follows Mermaid: 1 = top-right, 2 = top-left,
// 3 = bottom-left, 4 = bottom-right. Point coordinates outside 0..1 are
// clamped to the grid (the mmd encoder rejects them instead).
func encodeQuadrant(w io.Writer, d *ast.QuadrantDiagram, opts diagram.RenderOptions) error {
	padding := float64(opts.SVGPadding)
	if padding <= 0 {
		padding = 40
	}

	gx := padding + quadMarginL // grid left
	gy := padding + quadMarginT // grid top
	natW := gx + quadGrid + quadMarginR + padding
	natH := gy + quadGrid + quadMarginB + padding

	var sb strings.Builder
	writeSVGHeader(&sb, natW, natH, opts, d.Title())

	// Grid + quadrant dividers.
	sb.WriteString(`  <g id="quadrant-grid">` + "\n")
	fmt.Fprintf(&sb, `    <rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="#f8fafc" stroke="#64748b" stroke-width="1.5"/>`+"\n",
		gx, gy, quadGrid, quadGrid)
	midX, midY := gx+quadGrid/2, gy+quadGrid/2
	fmt.Fprintf(&sb, `    <path d="M%.1f,%.1f L%.1f,%.1f" stroke="#94a3b8" stroke-width="1" stroke-dasharray="5,3"/>`+"\n",
		midX, gy, midX, gy+quadGrid)
	fmt.Fprintf(&sb, `    <path d="M%.1f,%.1f L%.1f,%.1f" stroke="#94a3b8" stroke-width="1" stroke-dasharray="5,3"/>`+"\n",
		gx, midY, gx+quadGrid, midY)
	sb.WriteString("  </g>\n")

	// Quadrant labels: 1 = top-right, 2 = top-left, 3 = bottom-left, 4 = bottom-right.
	quads := d.Quadrants()
	quadCX := [4]float64{gx + quadGrid*0.75, gx + quadGrid*0.25, gx + quadGrid*0.25, gx + quadGrid*0.75}
	quadCY := [4]float64{gy + 18, gy + 18, gy + quadGrid - 18, gy + quadGrid - 18}
	sb.WriteString(`  <g id="quadrant-labels">` + "\n")
	for i, label := range quads {
		if label == "" {
			continue
		}
		fmt.Fprintf(&sb, `    <text x="%.1f" y="%.1f" text-anchor="middle" dominant-baseline="central" font-size="12" font-family="%s" fill="#64748b" font-style="italic">%s</text>`+"\n",
			quadCX[i], quadCY[i], defaultFontFamily, xmlEscape(label))
	}
	sb.WriteString("  </g>\n")

	// Axes.
	sb.WriteString(`  <g id="quadrant-axes">` + "\n")
	sb.WriteString(quadrantAxisSVG(gx, gy+quadGrid, gx+quadGrid, gy+quadGrid, d.XAxis(), true))
	sb.WriteString(quadrantAxisSVG(gx, gy, gx, gy+quadGrid, d.YAxis(), false))
	sb.WriteString("  </g>\n")

	// Points.
	sb.WriteString(`  <g id="quadrant-points">` + "\n")
	for _, p := range d.Points() {
		px := gx + clamp01(p.X)*quadGrid
		py := gy + (1-clamp01(p.Y))*quadGrid
		fmt.Fprintf(&sb, `    <circle cx="%.1f" cy="%.1f" r="%.1f" fill="#205ea6" stroke="#ffffff" stroke-width="1.5"/>`+"\n",
			px, py, quadPointR)
		name := p.Name
		if name == "" {
			name = "(unnamed)"
		}
		fmt.Fprintf(&sb, `    <text x="%.1f" y="%.1f" font-size="12" font-family="%s" fill="#1e293b">%s</text>`+"\n",
			px+quadPointR+4, py+4, defaultFontFamily, xmlEscape(name))
	}
	sb.WriteString("  </g>\n</svg>\n")

	_, err := io.WriteString(w, sb.String())
	return err
}

// quadrantAxisSVG renders one axis line with arrowheads and labels.
// horizontal: x-axis below the grid (from left, to right).
// vertical: y-axis left of the grid (from bottom, to top).
func quadrantAxisSVG(x1, y1, x2, y2 float64, labels [2]string, horizontal bool) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, `    <path d="M%.1f,%.1f L%.1f,%.1f" stroke="#475569" stroke-width="1.5"/>`+"\n",
		x1, y1, x2, y2)
	// Arrowheads at both ends.
	sb.WriteString(quadrantArrowSVG(x1, y1, x2, y2, false))
	sb.WriteString(quadrantArrowSVG(x1, y1, x2, y2, true))

	from, to := labels[0], labels[1]
	if horizontal {
		ly := y1 + 20
		if to == "" {
			fmt.Fprintf(&sb, `    <text x="%.1f" y="%.1f" text-anchor="middle" font-size="12" font-family="%s" fill="#334155">%s</text>`+"\n",
				(x1+x2)/2, ly, defaultFontFamily, xmlEscape(from))
		} else {
			fmt.Fprintf(&sb, `    <text x="%.1f" y="%.1f" font-size="12" font-family="%s" fill="#334155">%s</text>`+"\n",
				x1, ly, defaultFontFamily, xmlEscape(from))
			fmt.Fprintf(&sb, `    <text x="%.1f" y="%.1f" text-anchor="end" font-size="12" font-family="%s" fill="#334155">%s</text>`+"\n",
				x2, ly, defaultFontFamily, xmlEscape(to))
		}
		return sb.String()
	}
	lx := x1 - 10
	if to == "" {
		fmt.Fprintf(&sb, `    <text x="%.1f" y="%.1f" text-anchor="end" dominant-baseline="central" font-size="12" font-family="%s" fill="#334155">%s</text>`+"\n",
			lx, (y1+y2)/2, defaultFontFamily, xmlEscape(from))
	} else {
		// y1 is the grid top (high values), y2 the grid bottom (low values):
		// `to` (e.g. High) belongs at the top, `from` (e.g. Low) at the bottom.
		fmt.Fprintf(&sb, `    <text x="%.1f" y="%.1f" text-anchor="end" dominant-baseline="central" font-size="12" font-family="%s" fill="#334155">%s</text>`+"\n",
			lx, y1, defaultFontFamily, xmlEscape(to))
		fmt.Fprintf(&sb, `    <text x="%.1f" y="%.1f" text-anchor="end" dominant-baseline="central" font-size="12" font-family="%s" fill="#334155">%s</text>`+"\n",
			lx, y2, defaultFontFamily, xmlEscape(from))
	}
	return sb.String()
}

// quadrantArrowSVG draws a triangular arrowhead on the segment (x1,y1)-(x2,y2).
// atEnd selects which endpoint gets the head.
func quadrantArrowSVG(x1, y1, x2, y2 float64, atEnd bool) string {
	dx, dy := x2-x1, y2-y1
	len := math.Hypot(dx, dy)
	if len == 0 {
		return ""
	}
	ux, uy := dx/len, dy/len
	px, py := -uy, ux
	size := 7.0

	var tipX, tipY float64
	var dir float64
	if atEnd {
		tipX, tipY, dir = x2, y2, 1
	} else {
		tipX, tipY, dir = x1, y1, -1
	}
	bx, by := tipX-dir*size*ux, tipY-dir*size*uy
	return fmt.Sprintf(`<polygon points="%.1f,%.1f %.1f,%.1f %.1f,%.1f" fill="#475569"/>`+"\n",
		tipX, tipY, bx+px*size*0.45, by+py*size*0.45, bx-px*size*0.45, by-py*size*0.45)
}

// clamp01 constrains v to the 0..1 range, mapping NaN to 0.
func clamp01(v float64) float64 {
	if math.IsNaN(v) {
		return 0
	}
	return math.Min(1, math.Max(0, v))
}
