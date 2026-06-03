package svg

import (
	"fmt"
	"io"
	"math"
	"strings"

	diagram "github.com/iokdigital/go-mermaid"
	"github.com/iokdigital/go-mermaid/ast"
)

// ER entity box dimensions.
const (
	erEntityW       = 180.0 // fixed width for all entities
	erHeaderH       = 28.0  // entity name row height
	erAttrRowH      = 20.0  // height per attribute row
	erMinH          = erHeaderH + erAttrRowH // at least header + one empty row
	erCardSymbolLen = 16.0                   // space reserved for cardinality symbols at endpoints
)

// encodeER renders a *ast.ERDiagram to w as an SVG document.
// Entities are laid out top-to-bottom using Sugiyama; relationships are drawn as lines with cardinality symbols.
func encodeER(w io.Writer, d *ast.ERDiagram, opts diagram.RenderOptions) error {
	padding := float64(opts.SVGPadding)
	if padding <= 0 {
		padding = 40
	}
	maxW := float64(opts.SVGMaxWidth)
	if maxW <= 0 {
		maxW = 8000
	}
	maxH := float64(opts.SVGMaxHeight)
	if maxH <= 0 {
		maxH = 6000
	}

	entities := d.Entities()
	relations := d.Relations()

	// Build generic layout nodes (variable height based on attribute count).
	gnodes := make([]gNode, 0, len(entities))
	for _, e := range entities {
		gnodes = append(gnodes, gNode{
			ID: e.Name,
			W:  erEntityW,
			H:  erEntityHeight(e),
		})
	}
	gedges := make([]gEdge, 0, len(relations))
	for _, r := range relations {
		gedges = append(gedges, gEdge{From: r.From, To: r.To})
	}

	lr := runGenericLayout(gnodes, gedges, ast.DirectionTB, opts.Layout, padding, maxW, maxH)

	W := int(math.Ceil(lr.width))
	H := int(math.Ceil(lr.height))

	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	fmt.Fprintf(&sb, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`+"\n", W, H, W, H)

	if d.Title() != "" {
		fmt.Fprintf(&sb, "  <title>%s</title>\n", xmlEscape(d.Title()))
	}

	// Relations first so entity boxes render on top.
	sb.WriteString(`  <g id="er-relations">` + "\n")
	for _, r := range relations {
		src, ok1 := lr.nodes[r.From]
		dst, ok2 := lr.nodes[r.To]
		if !ok1 || !ok2 {
			continue
		}
		sb.WriteString("    ")
		sb.WriteString(erRelationSVG(src, dst, r))
		sb.WriteString("\n")
	}
	sb.WriteString("  </g>\n")

	sb.WriteString(`  <g id="er-entities">` + "\n")
	for _, e := range entities {
		nl, ok := lr.nodes[e.Name]
		if !ok {
			continue
		}
		sb.WriteString("    ")
		sb.WriteString(erEntitySVG(e, nl.x, nl.y))
		sb.WriteString("\n")
	}
	sb.WriteString("  </g>\n")

	sb.WriteString("</svg>\n")

	_, err := io.WriteString(w, sb.String())
	return err
}

// erEntityHeight computes the total height of an entity box.
func erEntityHeight(e ast.EREntity) float64 {
	rows := len(e.Attributes)
	if rows == 0 {
		rows = 1
	}
	return erHeaderH + float64(rows)*erAttrRowH
}

// erEntitySVG returns SVG for an ER entity box with header and attribute rows.
func erEntitySVG(e ast.EREntity, cx, cy float64) string {
	h := erEntityHeight(e)
	x := cx - erEntityW/2
	y := cy - h/2

	var sb strings.Builder
	fmt.Fprintf(&sb, `<g id="er-entity-%s">`, xmlEscape(e.Name))

	// Outer border.
	fmt.Fprintf(&sb, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="#f0f9ff" stroke="#0369a1" stroke-width="1.5"/>`,
		x, y, erEntityW, h)

	// Header row with dark background.
	fmt.Fprintf(&sb, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="#0369a1"/>`,
		x, y, erEntityW, erHeaderH)
	fmt.Fprintf(&sb, `<text x="%.1f" y="%.1f" text-anchor="middle" dominant-baseline="central" font-size="%d" font-family="%s" fill="#ffffff" font-weight="bold">%s</text>`,
		cx, y+erHeaderH/2, defaultFontSize, defaultFontFamily, xmlEscape(e.Name))

	// Divider line between header and attributes.
	fmt.Fprintf(&sb, `<path d="M%.1f,%.1f L%.1f,%.1f" stroke="#0369a1" stroke-width="1"/>`,
		x, y+erHeaderH, x+erEntityW, y+erHeaderH)

	// Attribute rows.
	attrs := e.Attributes
	if len(attrs) == 0 {
		// Empty placeholder row.
		rowY := y + erHeaderH + erAttrRowH/2
		fmt.Fprintf(&sb, `<text x="%.1f" y="%.1f" text-anchor="middle" dominant-baseline="central" font-size="11" font-family="%s" fill="#64748b" font-style="italic">%s</text>`,
			cx, rowY, defaultFontFamily, "(no attributes)")
	} else {
		for i, attr := range attrs {
			rowY := y + erHeaderH + float64(i)*erAttrRowH
			// Alternating row background.
			if i%2 == 0 {
				fmt.Fprintf(&sb, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="#e0f2fe"/>`,
					x, rowY, erEntityW, erAttrRowH)
			}
			// Row separator.
			if i > 0 {
				fmt.Fprintf(&sb, `<path d="M%.1f,%.1f L%.1f,%.1f" stroke="#bae6fd" stroke-width="0.5"/>`,
					x, rowY, x+erEntityW, rowY)
			}
			// Key badge.
			keysText := erKeysText(attr.Keys)
			keyW := 0.0
			if keysText != "" {
				keyW = float64(len(keysText))*7 + 6
				fmt.Fprintf(&sb, `<rect x="%.1f" y="%.1f" width="%.1f" height="12" rx="2" fill="#dbeafe" stroke="#2563eb" stroke-width="0.5"/>`,
					x+4, rowY+erAttrRowH/2-6, keyW)
				fmt.Fprintf(&sb, `<text x="%.1f" y="%.1f" dominant-baseline="central" font-size="9" font-family="%s" fill="#1d4ed8" font-weight="bold">%s</text>`,
					x+7, rowY+erAttrRowH/2, defaultFontFamily, xmlEscape(keysText))
			}
			// Attribute text: DataType Name.
			attrLabel := attr.DataType + " " + attr.Name
			attrX := x + 4 + keyW + 4
			fmt.Fprintf(&sb, `<text x="%.1f" y="%.1f" dominant-baseline="central" font-size="11" font-family="%s" fill="#1e293b">%s</text>`,
				attrX, rowY+erAttrRowH/2, defaultFontFamily, xmlEscape(attrLabel))
		}
	}

	sb.WriteString(`</g>`)
	return sb.String()
}

// erKeysText returns abbreviated key badge text.
func erKeysText(keys []ast.ERKey) string {
	if len(keys) == 0 {
		return ""
	}
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = string(k)
	}
	return strings.Join(parts, ",")
}

// erRelationSVG draws a relationship line with cardinality symbols and an optional label.
func erRelationSVG(src, dst *nodeLayout, r ast.ERRelation) string {
	if src == nil || dst == nil {
		return ""
	}

	// Exit/entry points: bottom of source, top of destination.
	sx, sy := src.x, src.y+src.h/2
	tx, ty := dst.x, dst.y-dst.h/2

	angle := math.Atan2(ty-sy, tx-sx)

	// Shorten endpoints to leave room for cardinality symbols.
	srcX := sx + erCardSymbolLen*math.Cos(angle)
	srcY := sy + erCardSymbolLen*math.Sin(angle)
	dstX := tx - erCardSymbolLen*math.Cos(angle)
	dstY := ty - erCardSymbolLen*math.Sin(angle)

	dash := ""
	if !r.Identifying {
		dash = `stroke-dasharray="5,3" `
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, `<path d="M%.1f,%.1f L%.1f,%.1f" fill="none" stroke="#475569" stroke-width="1.5" %s/>`,
		srcX, srcY, dstX, dstY, dash)

	// Cardinality symbols at each end.
	sb.WriteString(erCardSVG(sx, sy, angle, r.FromCard, false))
	sb.WriteString(erCardSVG(tx, ty, angle+math.Pi, r.ToCard, false))

	// Label at midpoint.
	if r.Label != "" {
		midX := (sx + tx) / 2
		midY := (sy + ty) / 2
		fmt.Fprintf(&sb, `<rect x="%.1f" y="%.1f" width="%.1f" height="14" rx="3" fill="white" stroke="none"/>`,
			midX-float64(len(r.Label))*3.5-4, midY-7, float64(len(r.Label))*7+8)
		fmt.Fprintf(&sb, `<text x="%.1f" y="%.1f" text-anchor="middle" dominant-baseline="central" font-size="11" font-family="%s" fill="#334155" font-style="italic">%s</text>`,
			midX, midY, defaultFontFamily, xmlEscape(r.Label))
	}

	return sb.String()
}

// erCardSVG draws cardinality notation at a line endpoint.
// ox,oy is the exact endpoint on the entity border; angle is the direction from
// the midpoint toward this endpoint (so symbols are drawn "inside" the line at distance).
func erCardSVG(ox, oy, angle float64, card ast.Cardinality, _ bool) string {
	// Perpendicular direction for tick marks.
	perpX := -math.Sin(angle)
	perpY := math.Cos(angle)
	tickLen := 6.0

	// d1: distance of first symbol from endpoint.
	// d2: distance of second symbol from endpoint.
	d1 := 8.0
	d2 := 15.0

	// Point at distance d from endpoint along the line (toward center).
	p1x := ox + d1*math.Cos(angle)
	p1y := oy + d1*math.Sin(angle)
	p2x := ox + d2*math.Cos(angle)
	p2y := oy + d2*math.Sin(angle)

	var sb strings.Builder
	stroke := `stroke="#475569" stroke-width="1.5"`

	switch card {
	case ast.CardExactOne: // "||" — two vertical bars
		// Inner bar at d1.
		fmt.Fprintf(&sb, `<path d="M%.1f,%.1f L%.1f,%.1f" fill="none" %s/>`,
			p1x+perpX*tickLen, p1y+perpY*tickLen, p1x-perpX*tickLen, p1y-perpY*tickLen, stroke)
		// Outer bar at d2.
		fmt.Fprintf(&sb, `<path d="M%.1f,%.1f L%.1f,%.1f" fill="none" %s/>`,
			p2x+perpX*tickLen, p2y+perpY*tickLen, p2x-perpX*tickLen, p2y-perpY*tickLen, stroke)

	case ast.CardZeroOne: // "|o" — bar + circle
		// Bar at d1.
		fmt.Fprintf(&sb, `<path d="M%.1f,%.1f L%.1f,%.1f" fill="none" %s/>`,
			p1x+perpX*tickLen, p1y+perpY*tickLen, p1x-perpX*tickLen, p1y-perpY*tickLen, stroke)
		// Circle at d2.
		fmt.Fprintf(&sb, `<ellipse cx="%.1f" cy="%.1f" rx="4" ry="4" fill="white" %s/>`, p2x, p2y, stroke)

	case ast.CardZeroMany: // "}o" — crow's foot + circle
		erCrowFoot(&sb, p1x, p1y, angle, perpX, perpY, tickLen, stroke)
		fmt.Fprintf(&sb, `<ellipse cx="%.1f" cy="%.1f" rx="4" ry="4" fill="white" %s/>`, p2x, p2y, stroke)

	case ast.CardOneMany: // "}|" — crow's foot + bar
		erCrowFoot(&sb, p1x, p1y, angle, perpX, perpY, tickLen, stroke)
		fmt.Fprintf(&sb, `<path d="M%.1f,%.1f L%.1f,%.1f" fill="none" %s/>`,
			p2x+perpX*tickLen, p2y+perpY*tickLen, p2x-perpX*tickLen, p2y-perpY*tickLen, stroke)
	}

	return sb.String()
}

// erCrowFoot draws a crow's foot (3 lines spreading outward) at position px,py.
func erCrowFoot(sb *strings.Builder, px, py, angle, perpX, perpY, tickLen float64, stroke string) {
	// Center line.
	endX := px - 8*math.Cos(angle)
	endY := py - 8*math.Sin(angle)
	fmt.Fprintf(sb, `<path d="M%.1f,%.1f L%.1f,%.1f" fill="none" %s/>`, px, py, endX, endY, stroke)
	// Left branch.
	fmt.Fprintf(sb, `<path d="M%.1f,%.1f L%.1f,%.1f" fill="none" %s/>`,
		px, py, endX+perpX*tickLen, endY+perpY*tickLen, stroke)
	// Right branch.
	fmt.Fprintf(sb, `<path d="M%.1f,%.1f L%.1f,%.1f" fill="none" %s/>`,
		px, py, endX-perpX*tickLen, endY-perpY*tickLen, stroke)
}
