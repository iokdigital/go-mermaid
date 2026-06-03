package svg

import (
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/iokdigital/go-mermaid/ast"
)

// textLines splits label into display lines per FRD §6.6.
// ≤20 chars → single line; 21–40 → word-wrap near char 20; >40 → truncate to 37+"…".
func textLines(label string) []string {
	n := utf8.RuneCountInString(label)
	if n > 40 {
		runes := []rune(label)
		return []string{string(runes[:37]) + "…"}
	}
	if n <= 20 {
		return []string{label}
	}
	runes := []rune(label)
	split := 20
	for i := 20; i > 10; i-- {
		if runes[i] == ' ' {
			split = i
			break
		}
	}
	line1 := strings.TrimSpace(string(runes[:split]))
	line2 := strings.TrimSpace(string(runes[split:]))
	return []string{line1, line2}
}

// xmlEscape escapes text for SVG text content and attribute values.
func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}

// shapeElement returns the SVG shape element for a node body.
func shapeElement(cx, cy, w, h float64, shape ast.NodeShape, fill, stroke string) string {
	sw := defaultStrokeWidth
	switch shape {
	case ast.ShapeCircle:
		r := math.Min(w, h) / 2
		return fmt.Sprintf(`<ellipse cx="%.1f" cy="%.1f" rx="%.1f" ry="%.1f" fill="%s" stroke="%s" stroke-width="%.1f"/>`,
			cx, cy, r, r, fill, stroke, sw)
	case ast.ShapeDiamond:
		return fmt.Sprintf(`<polygon points="%.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f" fill="%s" stroke="%s" stroke-width="%.1f"/>`,
			cx, cy-h/2,
			cx+w/2, cy,
			cx, cy+h/2,
			cx-w/2, cy,
			fill, stroke, sw)
	case ast.ShapeRoundedRect:
		return fmt.Sprintf(`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="8" ry="8" fill="%s" stroke="%s" stroke-width="%.1f"/>`,
			cx-w/2, cy-h/2, w, h, fill, stroke, sw)
	case ast.ShapeParallelogram:
		skew := 10.0
		return fmt.Sprintf(`<polygon points="%.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f" fill="%s" stroke="%s" stroke-width="%.1f"/>`,
			cx-w/2+skew, cy-h/2,
			cx+w/2+skew, cy-h/2,
			cx+w/2-skew, cy+h/2,
			cx-w/2-skew, cy+h/2,
			fill, stroke, sw)
	case ast.ShapeHexagon:
		hw := w / 2
		hh := h / 2
		inset := hw * 0.25
		return fmt.Sprintf(`<polygon points="%.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f" fill="%s" stroke="%s" stroke-width="%.1f"/>`,
			cx-hw, cy,
			cx-hw+inset, cy-hh,
			cx+hw-inset, cy-hh,
			cx+hw, cy,
			cx+hw-inset, cy+hh,
			cx-hw+inset, cy+hh,
			fill, stroke, sw)
	case ast.ShapeStadium:
		return fmt.Sprintf(`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="%.1f" ry="%.1f" fill="%s" stroke="%s" stroke-width="%.1f"/>`,
			cx-w/2, cy-h/2, w, h, h/2, h/2, fill, stroke, sw)
	case ast.ShapeAsymmetric:
		notch := 10.0
		return fmt.Sprintf(`<polygon points="%.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f" fill="%s" stroke="%s" stroke-width="%.1f"/>`,
			cx-w/2, cy-h/2,
			cx+w/2-notch, cy-h/2,
			cx+w/2, cy,
			cx+w/2-notch, cy+h/2,
			cx-w/2, cy+h/2,
			fill, stroke, sw)
	default: // ShapeRect and unknown
		return fmt.Sprintf(`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s" stroke="%s" stroke-width="%.1f"/>`,
			cx-w/2, cy-h/2, w, h, fill, stroke, sw)
	}
}

// labelElement returns SVG text element(s) for a node label.
func labelElement(cx, cy float64, lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	fs := defaultFontSize
	ff := defaultFontFamily
	if len(lines) == 1 {
		return fmt.Sprintf(
			`<text x="%.1f" y="%.1f" text-anchor="middle" dominant-baseline="central" font-size="%d" font-family="%s" fill="#1e293b">%s</text>`,
			cx, cy, fs, ff, xmlEscape(lines[0]))
	}
	lineH := float64(fs) * 1.3
	var sb strings.Builder
	fmt.Fprintf(&sb, `<text text-anchor="middle" font-size="%d" font-family="%s" fill="#1e293b">`, fs, ff)
	fmt.Fprintf(&sb, `<tspan x="%.1f" y="%.1f">%s</tspan>`, cx, cy-lineH/2, xmlEscape(lines[0]))
	fmt.Fprintf(&sb, `<tspan x="%.1f" y="%.1f">%s</tspan>`, cx, cy+lineH/2, xmlEscape(lines[1]))
	sb.WriteString(`</text>`)
	return sb.String()
}

// nodeGroupSVG returns SVG for a complete node (shape + label, optionally in <a>).
func nodeGroupSVG(n *ast.FlowNode, x, y float64) string {
	fill, stroke := nodeColors(n.Confidence)
	shape := shapeElement(x, y, nodeWidth, nodeHeight, n.Shape, fill, stroke)
	label := labelElement(x, y, textLines(n.Label))

	var inner strings.Builder
	fmt.Fprintf(&inner, `<g id="node-%s">`, ast.SanitizeID(n.ID))
	inner.WriteString(shape)
	inner.WriteString(label)
	inner.WriteString(`</g>`)

	if safe := safeURL(n.URL); safe != "" {
		var outer strings.Builder
		fmt.Fprintf(&outer, `<a href="%s" target="_blank" rel="noopener noreferrer">`, xmlEscape(safe))
		outer.WriteString(inner.String())
		outer.WriteString(`</a>`)
		return outer.String()
	}
	return inner.String()
}

// safeURL returns u if the scheme is http, https, or relative; returns "" for
// javascript:, data:, vbscript:, and other dangerous schemes.
func safeURL(u string) string {
	if u == "" {
		return ""
	}
	lower := strings.ToLower(strings.TrimSpace(u))
	for _, blocked := range []string{"javascript:", "data:", "vbscript:"} {
		if strings.HasPrefix(lower, blocked) {
			return ""
		}
	}
	return u
}
