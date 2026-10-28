package svg

import (
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	diagram "github.com/iokdigital/go-mermaid"
	"github.com/iokdigital/go-mermaid/ast"
)

// Pie chart geometry.
const (
	pieRadius   = 150.0
	pieLegendDx = 24.0 // gap between circle and legend
	pieLegendW  = 260.0
	pieRowH     = 22.0
	pieMinArc   = 8.0 // degrees: smaller slices get legend-only labels
)

// piePalette cycles across slices.
var piePalette = []string{
	"#4e79a7", "#f28e2b", "#e15759", "#76b7b2",
	"#59a14f", "#edc948", "#b07aa1", "#ff9da7",
}

// encodePie renders a *ast.PieDiagram to w as an SVG document.
// Slices are drawn clockwise from the top; non-positive slices appear in the
// legend only. A non-positive total renders a placeholder diagram.
func encodePie(w io.Writer, d *ast.PieDiagram, opts diagram.RenderOptions) error {
	padding := float64(opts.SVGPadding)
	if padding <= 0 {
		padding = 40
	}

	slices := d.Slices()
	total := 0.0
	for _, s := range slices {
		if s.Value > 0 {
			total += s.Value
		}
	}

	cx := padding + pieRadius
	cy := padding + pieRadius
	legendX := cx + pieRadius + pieLegendDx
	legendH := float64(len(slices))*pieRowH + 10
	diam := pieRadius * 2
	contentH := math.Max(diam, legendH)
	natW := legendX + pieLegendW + padding
	natH := padding + contentH + padding

	var sb strings.Builder
	writeSVGHeader(&sb, natW, natH, opts, d.Title())
	if total <= 0 {
		fmt.Fprintf(&sb, `  <g id="pie-empty">`)
		fmt.Fprintf(&sb, `<text x="%.1f" y="%.1f" text-anchor="middle" dominant-baseline="central" font-size="%d" font-family="%s" fill="#64748b" font-style="italic">(no data)</text>`,
			cx, cy, defaultFontSize, defaultFontFamily)
		sb.WriteString("</g>\n</svg>\n")
		_, err := io.WriteString(w, sb.String())
		return err
	}

	// Slices, clockwise from 12 o'clock.
	sb.WriteString(`  <g id="pie-slices">` + "\n")
	angle := -90.0
	for i, s := range slices {
		fill := piePalette[i%len(piePalette)]
		pct := 0.0
		if s.Value > 0 {
			pct = s.Value / total * 100
		}
		sweep := pct / 100 * 360
		if sweep > 0 {
			sb.WriteString("    ")
			sb.WriteString(pieSliceSVG(cx, cy, pieRadius, angle, sweep, fill))
			sb.WriteString("\n")
			// Percentage label on slices large enough to hold it.
			if sweep >= pieMinArc {
				mid := (angle + angle + sweep) / 2
				lx := cx + pieRadius*0.68*math.Cos(mid*math.Pi/180)
				ly := cy + pieRadius*0.68*math.Sin(mid*math.Pi/180)
				fmt.Fprintf(&sb, `    <text x="%.1f" y="%.1f" text-anchor="middle" dominant-baseline="central" font-size="12" font-family="%s" fill="#ffffff">%s</text>`+"\n",
					lx, ly, defaultFontFamily, piePctText(pct))
			}
		}
		angle += sweep
	}
	sb.WriteString("  </g>\n")

	// Legend.
	sb.WriteString(`  <g id="pie-legend">` + "\n")
	ly := padding + 5
	for i, s := range slices {
		fill := piePalette[i%len(piePalette)]
		pct := 0.0
		if s.Value > 0 {
			pct = s.Value / total * 100
		}
		fmt.Fprintf(&sb, `    <rect x="%.1f" y="%.1f" width="12" height="12" rx="2" fill="%s"/>`+"\n",
			legendX, ly, fill)
		fmt.Fprintf(&sb, `    <text x="%.1f" y="%.1f" dominant-baseline="central" font-size="12" font-family="%s" fill="#1e293b">%s</text>`+"\n",
			legendX+18, ly+6, defaultFontFamily,
			xmlEscape(pieLegendText(s.Label, s.Value, pct)))
		ly += pieRowH
	}
	sb.WriteString("  </g>\n</svg>\n")

	_, err := io.WriteString(w, sb.String())
	return err
}

// pieSliceSVG returns a wedge path (or full circle for ~100% slices).
func pieSliceSVG(cx, cy, r, startDeg, sweepDeg float64, fill string) string {
	if sweepDeg >= 359.99 {
		return fmt.Sprintf(`<circle cx="%.1f" cy="%.1f" r="%.1f" fill="%s" stroke="#ffffff" stroke-width="2"/>`,
			cx, cy, r, fill)
	}
	start := startDeg * math.Pi / 180
	end := (startDeg + sweepDeg) * math.Pi / 180
	x1, y1 := cx+r*math.Cos(start), cy+r*math.Sin(start)
	x2, y2 := cx+r*math.Cos(end), cy+r*math.Sin(end)
	large := 0
	if sweepDeg > 180 {
		large = 1
	}
	return fmt.Sprintf(
		`<path d="M%.1f,%.1f L%.1f,%.1f A%.1f,%.1f 0 %d,1 %.1f,%.1f Z" fill="%s" stroke="#ffffff" stroke-width="2"/>`,
		cx, cy, x1, y1, r, r, large, x2, y2, fill)
}

// piePctText formats a slice percentage: integral values without decimals.
func piePctText(pct float64) string {
	if pct == math.Trunc(pct) {
		return strconv.FormatFloat(pct, 'f', 0, 64) + "%"
	}
	return strconv.FormatFloat(pct, 'f', 1, 64) + "%"
}

// pieLegendText formats a legend row: "label — value (pct%)".
func pieLegendText(label string, value, pct float64) string {
	if label == "" {
		label = "(unnamed)"
	}
	return fmt.Sprintf("%s — %s (%s)",
		label,
		strconv.FormatFloat(value, 'f', -1, 64),
		piePctText(pct))
}
