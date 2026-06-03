package svg

import (
	"fmt"
	"io"
	"math"
	"strings"

	diagram "github.com/iokdigital/go-mermaid"
	"github.com/iokdigital/go-mermaid/ast"
)

// Class diagram layout constants.
const (
	classBoxW        = 180.0
	classHeaderH     = 32.0 // name section height (+ annotation row if present)
	classAnnotationH = 16.0
	classMemberH     = 18.0 // height per attribute or method row
	classMinSectionH = 18.0 // minimum height for empty attribute/method section
)

// encodeClass renders a *ast.ClassDiagram to w as an SVG document.
func encodeClass(w io.Writer, d *ast.ClassDiagram, opts diagram.RenderOptions) error {
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

	classes := d.Classes()
	relations := d.Relations()
	notes := d.Notes()

	gnodes := make([]gNode, 0, len(classes))
	for _, c := range classes {
		gnodes = append(gnodes, gNode{
			ID: c.Name,
			W:  classBoxW,
			H:  classBoxHeight(c),
		})
	}
	gedges := make([]gEdge, 0, len(relations))
	for _, r := range relations {
		gedges = append(gedges, gEdge{From: r.From, To: r.To})
	}

	lr := runGenericLayout(gnodes, gedges, ast.DirectionTB, opts.Layout, padding, maxW, maxH)

	// Index notes by class name.
	noteByClass := make(map[string]string, len(notes))
	for _, n := range notes {
		noteByClass[n.Class] = n.Text
	}

	// Collect namespaces → classes for namespace grouping boxes.
	nsMap := make(map[string][]ast.DiagramClass)
	for _, c := range classes {
		if c.Namespace != "" {
			nsMap[c.Namespace] = append(nsMap[c.Namespace], c)
		}
	}

	W := int(math.Ceil(lr.width))
	H := int(math.Ceil(lr.height))

	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	fmt.Fprintf(&sb, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`+"\n", W, H, W, H)

	if d.Title() != "" {
		fmt.Fprintf(&sb, "  <title>%s</title>\n", xmlEscape(d.Title()))
	}

	// Namespace grouping boxes (drawn first, behind everything).
	if len(nsMap) > 0 {
		sb.WriteString(`  <g id="namespaces">` + "\n")
		for ns, nsClasses := range nsMap {
			sb.WriteString("    ")
			sb.WriteString(classNamespaceSVG(ns, nsClasses, lr))
			sb.WriteString("\n")
		}
		sb.WriteString("  </g>\n")
	}

	// Relation lines.
	sb.WriteString(`  <g id="class-relations">` + "\n")
	for _, r := range relations {
		src, ok1 := lr.nodes[r.From]
		dst, ok2 := lr.nodes[r.To]
		if !ok1 || !ok2 {
			continue
		}
		sb.WriteString("    ")
		sb.WriteString(classRelationSVG(src, dst, r))
		sb.WriteString("\n")
	}
	sb.WriteString("  </g>\n")

	// Class boxes.
	sb.WriteString(`  <g id="class-nodes">` + "\n")
	for _, c := range classes {
		nl, ok := lr.nodes[c.Name]
		if !ok {
			continue
		}
		sb.WriteString("    ")
		sb.WriteString(classBoxSVG(c, nl.x, nl.y))
		sb.WriteString("\n")

		if text, ok := noteByClass[c.Name]; ok {
			sb.WriteString("    ")
			sb.WriteString(stateNoteSVG(nl.x+nl.w/2+10, nl.y, text))
			sb.WriteString("\n")
		}
	}
	sb.WriteString("  </g>\n")

	sb.WriteString("</svg>\n")

	_, err := io.WriteString(w, sb.String())
	return err
}

// classBoxHeight computes total height of a class box.
func classBoxHeight(c ast.DiagramClass) float64 {
	h := classHeaderH
	if c.Annotation != "" {
		h += classAnnotationH
	}

	attrs := 0
	methods := 0
	for _, m := range c.Members {
		if m.IsMethod {
			methods++
		} else {
			attrs++
		}
	}
	if attrs == 0 {
		h += classMinSectionH
	} else {
		h += float64(attrs) * classMemberH
	}
	if methods == 0 {
		h += classMinSectionH
	} else {
		h += float64(methods) * classMemberH
	}
	return h
}

// classBoxSVG returns SVG for a complete class box (3-section UML style).
func classBoxSVG(c ast.DiagramClass, cx, cy float64) string {
	h := classBoxHeight(c)
	x := cx - classBoxW/2
	y := cy - h/2

	var sb strings.Builder
	fmt.Fprintf(&sb, `<g id="class-%s">`, xmlEscape(c.Name))

	// Outer border.
	fmt.Fprintf(&sb, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="#f8fafc" stroke="#334155" stroke-width="1.5"/>`,
		x, y, classBoxW, h)

	// Header section.
	headerH := classHeaderH
	if c.Annotation != "" {
		headerH += classAnnotationH
	}
	fmt.Fprintf(&sb, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="#e2e8f0"/>`,
		x, y, classBoxW, headerH)

	curY := y
	if c.Annotation != "" {
		// Annotation line (e.g. <<interface>>).
		fmt.Fprintf(&sb, `<text x="%.1f" y="%.1f" text-anchor="middle" dominant-baseline="central" font-size="10" font-family="%s" fill="#64748b" font-style="italic">%s</text>`,
			cx, curY+classAnnotationH/2, defaultFontFamily, xmlEscape(c.Annotation))
		curY += classAnnotationH
	}
	// Class name (bold).
	fmt.Fprintf(&sb, `<text x="%.1f" y="%.1f" text-anchor="middle" dominant-baseline="central" font-size="%d" font-family="%s" fill="#0f172a" font-weight="bold">%s</text>`,
		cx, curY+classHeaderH/2, defaultFontSize, defaultFontFamily, xmlEscape(c.Name))
	curY += classHeaderH

	// Divider after header.
	fmt.Fprintf(&sb, `<path d="M%.1f,%.1f L%.1f,%.1f" stroke="#334155" stroke-width="1"/>`,
		x, curY, x+classBoxW, curY)

	// Attributes section.
	attrs := make([]ast.ClassMember, 0)
	methods := make([]ast.ClassMember, 0)
	for _, m := range c.Members {
		if m.IsMethod {
			methods = append(methods, m)
		} else {
			attrs = append(attrs, m)
		}
	}

	if len(attrs) == 0 {
		curY += classMinSectionH
	} else {
		for _, m := range attrs {
			fmt.Fprintf(&sb, `<text x="%.1f" y="%.1f" dominant-baseline="central" font-size="11" font-family="%s" fill="#1e293b">%s</text>`,
				x+6, curY+classMemberH/2, defaultFontFamily, xmlEscape(classMemberText(m)))
			curY += classMemberH
		}
	}

	// Divider before methods.
	fmt.Fprintf(&sb, `<path d="M%.1f,%.1f L%.1f,%.1f" stroke="#94a3b8" stroke-width="0.75"/>`,
		x, curY, x+classBoxW, curY)

	// Methods section.
	if len(methods) == 0 {
		curY += classMinSectionH
	} else {
		for _, m := range methods {
			text := classMemberText(m)
			if m.IsAbstract {
				fmt.Fprintf(&sb, `<text x="%.1f" y="%.1f" dominant-baseline="central" font-size="11" font-family="%s" fill="#1e293b" font-style="italic">%s</text>`,
					x+6, curY+classMemberH/2, defaultFontFamily, xmlEscape(text))
			} else {
				fmt.Fprintf(&sb, `<text x="%.1f" y="%.1f" dominant-baseline="central" font-size="11" font-family="%s" fill="#1e293b">%s</text>`,
					x+6, curY+classMemberH/2, defaultFontFamily, xmlEscape(text))
			}
			if m.IsStatic {
				// Underline for static.
				fmt.Fprintf(&sb, `<path d="M%.1f,%.1f L%.1f,%.1f" stroke="#1e293b" stroke-width="0.75"/>`,
					x+6, curY+classMemberH-2, x+classBoxW-6, curY+classMemberH-2)
			}
			curY += classMemberH
		}
	}

	sb.WriteString(`</g>`)
	return sb.String()
}

// classMemberText returns the display string for a class member.
func classMemberText(m ast.ClassMember) string {
	vis := string(m.Visibility)
	name := m.Name
	if m.IsMethod {
		if m.Type != "" {
			return vis + name + "() " + m.Type
		}
		return vis + name + "()"
	}
	if m.Type != "" {
		return vis + m.Type + " " + name
	}
	return vis + name
}

// classRelationSVG draws the line between two classes with the appropriate arrowhead.
func classRelationSVG(src, dst *nodeLayout, r ast.ClassRelation) string {
	if src == nil || dst == nil {
		return ""
	}

	sx, sy := src.x, src.y+src.h/2
	tx, ty := dst.x, dst.y-dst.h/2

	angle := math.Atan2(ty-sy, tx-sx)

	isDashed := r.Kind == ast.RelRealization || r.Kind == ast.RelDependency
	dash := ""
	if isDashed {
		dash = `stroke-dasharray="7,4" `
	}

	// Shorten endpoint to leave room for arrowhead/diamond.
	const headRoom = 14.0
	lineEndX := tx - headRoom*math.Cos(angle)
	lineEndY := ty - headRoom*math.Sin(angle)

	// For source decorators (diamonds), shorten source side too.
	hasSrcDecor := r.Kind == ast.RelComposition || r.Kind == ast.RelAggregation
	srcX, srcY := sx, sy
	if hasSrcDecor {
		srcX = sx + headRoom*math.Cos(angle)
		srcY = sy + headRoom*math.Sin(angle)
	}

	stroke := "#334155"

	var sb strings.Builder
	fmt.Fprintf(&sb, `<path d="M%.1f,%.1f L%.1f,%.1f" fill="none" stroke="%s" stroke-width="1.5" %s/>`,
		srcX, srcY, lineEndX, lineEndY, stroke, dash)

	// Target-end decorators.
	switch r.Kind {
	case ast.RelInheritance, ast.RelRealization:
		// Open triangle (inheritance/realization).
		sb.WriteString(classOpenTriangle(tx, ty, angle, stroke))
	case ast.RelAssociation, ast.RelDependency:
		// Simple filled arrowhead.
		sb.WriteString(arrowheadSVG(tx, ty, angle, stroke))
	case ast.RelLink:
		// No arrowhead.
	}

	// Source-end decorators.
	switch r.Kind {
	case ast.RelComposition:
		sb.WriteString(classDiamond(sx, sy, angle, stroke, true))
	case ast.RelAggregation:
		sb.WriteString(classDiamond(sx, sy, angle, stroke, false))
	}

	// Cardinality labels.
	if r.CardFrom != "" {
		lx := sx + 20*math.Cos(angle) + 12*(-math.Sin(angle))
		ly := sy + 20*math.Sin(angle) + 12*math.Cos(angle)
		fmt.Fprintf(&sb, `<text x="%.1f" y="%.1f" text-anchor="middle" font-size="10" font-family="%s" fill="#334155">%s</text>`,
			lx, ly, defaultFontFamily, xmlEscape(r.CardFrom))
	}
	if r.CardTo != "" {
		lx := tx - 20*math.Cos(angle) + 12*(-math.Sin(angle))
		ly := ty - 20*math.Sin(angle) + 12*math.Cos(angle)
		fmt.Fprintf(&sb, `<text x="%.1f" y="%.1f" text-anchor="middle" font-size="10" font-family="%s" fill="#334155">%s</text>`,
			lx, ly, defaultFontFamily, xmlEscape(r.CardTo))
	}

	// Relation label at midpoint.
	if r.Label != "" {
		midX := (sx + tx) / 2
		midY := (sy + ty) / 2
		fmt.Fprintf(&sb, `<text x="%.1f" y="%.1f" text-anchor="middle" font-size="11" font-family="%s" fill="#334155">%s</text>`,
			midX, midY-8, defaultFontFamily, xmlEscape(r.Label))
	}

	return sb.String()
}

// classOpenTriangle draws an open (hollow) arrowhead — used for Inheritance and Realization.
func classOpenTriangle(tx, ty, angle float64, stroke string) string {
	cos, sin := math.Cos(angle), math.Sin(angle)
	bx := tx - arrowLen*1.5*cos
	by := ty - arrowLen*1.5*sin
	lx := bx + (arrowWidth)*sin
	ly := by - (arrowWidth)*cos
	rx := bx - (arrowWidth)*sin
	ry := by + (arrowWidth)*cos
	return fmt.Sprintf(`<polygon points="%.1f,%.1f %.1f,%.1f %.1f,%.1f" fill="white" stroke="%s" stroke-width="1.5"/>`,
		tx, ty, lx, ly, rx, ry, stroke)
}

// classDiamond draws a diamond decorator at the source end of a relation.
// filled=true → Composition (filled), false → Aggregation (open).
func classDiamond(sx, sy, angle float64, stroke string, filled bool) string {
	cos, sin := math.Cos(angle), math.Sin(angle)
	dlen := 12.0
	dwid := 6.0
	// Diamond points: tip (at sx,sy), left, back, right.
	// tip is toward the source (we come from the center so angle points toward dst,
	// meaning back toward source is -angle direction).
	tipX := sx
	tipY := sy
	backX := tipX - 2*dlen*cos
	backY := tipY - 2*dlen*sin
	leftX := tipX - dlen*cos + dwid*sin
	leftY := tipY - dlen*sin - dwid*cos
	rightX := tipX - dlen*cos - dwid*sin
	rightY := tipY - dlen*sin + dwid*cos

	fill := "white"
	if filled {
		fill = stroke
	}
	return fmt.Sprintf(`<polygon points="%.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f" fill="%s" stroke="%s" stroke-width="1.5"/>`,
		tipX, tipY, leftX, leftY, backX, backY, rightX, rightY, fill, stroke)
}

// classNamespaceSVG draws a dashed grouping rectangle around all classes in a namespace.
func classNamespaceSVG(ns string, classes []ast.DiagramClass, lr *layoutResult) string {
	minX, minY := math.MaxFloat64, math.MaxFloat64
	maxX, maxY := -math.MaxFloat64, -math.MaxFloat64
	for _, c := range classes {
		nl, ok := lr.nodes[c.Name]
		if !ok {
			continue
		}
		if nl.x-nl.w/2 < minX {
			minX = nl.x - nl.w/2
		}
		if nl.y-nl.h/2 < minY {
			minY = nl.y - nl.h/2
		}
		if nl.x+nl.w/2 > maxX {
			maxX = nl.x + nl.w/2
		}
		if nl.y+nl.h/2 > maxY {
			maxY = nl.y + nl.h/2
		}
	}
	if minX == math.MaxFloat64 {
		return ""
	}
	pad := 12.0
	x := minX - pad
	y := minY - pad - 16 // extra top space for label
	w := (maxX - minX) + 2*pad
	h := (maxY - minY) + 2*pad + 16
	return fmt.Sprintf(
		`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="6" fill="none" stroke="#94a3b8" stroke-width="1" stroke-dasharray="6,3"/>`+
			`<text x="%.1f" y="%.1f" font-size="11" font-family="%s" fill="#64748b">namespace %s</text>`,
		x, y, w, h,
		x+6, y+12, defaultFontFamily, xmlEscape(ns))
}
