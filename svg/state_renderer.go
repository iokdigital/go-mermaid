package svg

import (
	"fmt"
	"io"
	"math"
	"strings"

	diagram "github.com/iokdigital/go-mermaid"
	"github.com/iokdigital/go-mermaid/ast"
)

// Dimensions for state diagram nodes.
const (
	stateNodeW   = 120.0
	stateNodeH   = 40.0
	stateCircleR = 12.0 // start/end pseudo-state radius
	stateForkW   = 80.0 // fork/join horizontal bar width
	stateForkH   = 10.0 // fork/join horizontal bar height
	stateDiamW   = 60.0 // choice diamond width
	stateDiamH   = 40.0 // choice diamond height
)

// encodeState renders a *ast.StateDiagram to w as an SVG document.
// Composite state children are rendered as nested sub-graphs inside the parent box.
// Top-level layout uses Sugiyama TB.
func encodeState(w io.Writer, d *ast.StateDiagram, opts diagram.RenderOptions) error {
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

	states := d.States()
	transitions := d.Transitions()
	notes := d.Notes()

	// Convert states to generic layout nodes.
	gnodes := make([]gNode, 0, len(states))
	for _, s := range states {
		gnodes = append(gnodes, stateGNode(s))
	}
	// Convert transitions to generic edges.
	gedges := make([]gEdge, 0, len(transitions))
	for _, t := range transitions {
		gedges = append(gedges, gEdge{From: t.From, To: t.To})
	}

	lr := runGenericLayout(gnodes, gedges, ast.DirectionTB, opts.Layout, padding, maxW, maxH)

	// Build a note index by state ID.
	noteByState := make(map[string]string, len(notes))
	for _, n := range notes {
		noteByState[n.State] = n.Text
	}

	W := int(math.Ceil(lr.width))
	H := int(math.Ceil(lr.height))

	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	fmt.Fprintf(&sb, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`+"\n", W, H, W, H)

	if d.Title() != "" {
		fmt.Fprintf(&sb, "  <title>%s</title>\n", xmlEscape(d.Title()))
	}

	// Edges first so nodes render on top.
	sb.WriteString(`  <g id="state-edges">` + "\n")
	for _, t := range transitions {
		src, ok1 := lr.nodes[t.From]
		dst, ok2 := lr.nodes[t.To]
		if !ok1 || !ok2 {
			continue
		}
		sb.WriteString("    ")
		sb.WriteString(stateEdgeSVG(src, dst, t.Event))
		sb.WriteString("\n")
	}
	sb.WriteString("  </g>\n")

	sb.WriteString(`  <g id="state-nodes">` + "\n")
	for _, s := range states {
		nl, ok := lr.nodes[s.ID]
		if !ok {
			continue
		}
		sb.WriteString("    ")
		sb.WriteString(stateNodeSVG(s, nl.x, nl.y))
		sb.WriteString("\n")

		// Render note if present.
		if text, ok := noteByState[s.ID]; ok {
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

// stateGNode returns the layout dimensions for a state node by kind.
func stateGNode(s ast.DiagramState) gNode {
	switch s.Kind {
	case ast.StateStart, ast.StateEnd:
		return gNode{ID: s.ID, W: stateCircleR * 2, H: stateCircleR * 2}
	case ast.StateFork, ast.StateJoin:
		return gNode{ID: s.ID, W: stateForkW, H: stateForkH}
	case ast.StateChoice:
		return gNode{ID: s.ID, W: stateDiamW, H: stateDiamH}
	default:
		return gNode{ID: s.ID, W: stateNodeW, H: stateNodeH}
	}
}

// stateNodeSVG returns the SVG markup for a single state node.
func stateNodeSVG(s ast.DiagramState, cx, cy float64) string {
	const fill = "#dbeafe"     // light blue
	const stroke = "#2563eb"   // blue border
	const startFill = "#1e293b" // dark for start
	const endFill = "#1e293b"

	var sb strings.Builder
	fmt.Fprintf(&sb, `<g id="state-%s">`, xmlEscape(s.ID))

	switch s.Kind {
	case ast.StateStart:
		// Filled black circle.
		fmt.Fprintf(&sb, `<ellipse cx="%.1f" cy="%.1f" rx="%.1f" ry="%.1f" fill="%s"/>`,
			cx, cy, stateCircleR, stateCircleR, startFill)

	case ast.StateEnd:
		// Double circle: outer ring + inner filled circle.
		outer := stateCircleR + 4
		fmt.Fprintf(&sb, `<ellipse cx="%.1f" cy="%.1f" rx="%.1f" ry="%.1f" fill="none" stroke="%s" stroke-width="1.5"/>`,
			cx, cy, outer, outer, endFill)
		fmt.Fprintf(&sb, `<ellipse cx="%.1f" cy="%.1f" rx="%.1f" ry="%.1f" fill="%s"/>`,
			cx, cy, stateCircleR, stateCircleR, endFill)

	case ast.StateFork, ast.StateJoin:
		// Filled black horizontal bar.
		fmt.Fprintf(&sb, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s"/>`,
			cx-stateForkW/2, cy-stateForkH/2, stateForkW, stateForkH, startFill)

	case ast.StateChoice:
		// Diamond shape.
		fmt.Fprintf(&sb, `<polygon points="%.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f" fill="%s" stroke="%s" stroke-width="1.5"/>`,
			cx, cy-stateDiamH/2,
			cx+stateDiamW/2, cy,
			cx, cy+stateDiamH/2,
			cx-stateDiamW/2, cy,
			fill, stroke)

	default: // StateNormal
		if len(s.Children) > 0 {
			// Composite state: rounded rect with sub-diagram inside.
			// For now render as a labeled dashed box; nested layout deferred.
			w := stateNodeW * 1.5
			h := stateNodeH * (1 + float64(len(s.Children))*0.8)
			fmt.Fprintf(&sb, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="8" ry="8" fill="%s" stroke="%s" stroke-width="1.5" stroke-dasharray="6,3"/>`,
				cx-w/2, cy-h/2, w, h, fill, stroke)
			// Title bar.
			fmt.Fprintf(&sb, `<text x="%.1f" y="%.1f" text-anchor="middle" dominant-baseline="central" font-size="%d" font-family="%s" fill="#1e293b" font-weight="bold">%s</text>`,
				cx, cy-h/2+10, defaultFontSize, defaultFontFamily, xmlEscape(displayLabel(s)))
			// Child states as small labels inside.
			for i, child := range s.Children {
				chy := cy - h/2 + 30 + float64(i)*20
				fmt.Fprintf(&sb, `<text x="%.1f" y="%.1f" text-anchor="middle" font-size="11" font-family="%s" fill="#334155">%s</text>`,
					cx, chy, defaultFontFamily, xmlEscape(displayLabel(child)))
			}
		} else {
			// Plain state: rounded rect with centered label.
			fmt.Fprintf(&sb, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="8" ry="8" fill="%s" stroke="%s" stroke-width="1.5"/>`,
				cx-stateNodeW/2, cy-stateNodeH/2, stateNodeW, stateNodeH, fill, stroke)
			label := displayLabel(s)
			fmt.Fprintf(&sb, `<text x="%.1f" y="%.1f" text-anchor="middle" dominant-baseline="central" font-size="%d" font-family="%s" fill="#1e293b">%s</text>`,
				cx, cy, defaultFontSize, defaultFontFamily, xmlEscape(label))
		}
	}

	sb.WriteString(`</g>`)
	return sb.String()
}

// displayLabel returns the display text for a state: Label if set, else ID.
func displayLabel(s ast.DiagramState) string {
	if s.Label != "" {
		return s.Label
	}
	return s.ID
}

// stateEdgeSVG draws a directed arrow between two state nodes.
func stateEdgeSVG(src, dst *nodeLayout, event string) string {
	if src == nil || dst == nil {
		return ""
	}

	// Exit bottom of source, enter top of destination.
	sx, sy := src.x, src.y+src.h/2
	tx, ty := dst.x, dst.y-dst.h/2

	// For self-loops render a small arc above the node.
	if src.id == dst.id {
		r := 15.0
		return fmt.Sprintf(
			`<path d="M%.1f,%.1f A%.1f,%.1f 0 1,1 %.1f,%.1f" fill="none" stroke="#555" stroke-width="1.5"/>`,
			sx-r, src.y-src.h/2, r, r, sx+r, src.y-src.h/2)
	}

	angle := math.Atan2(ty-sy, tx-sx)
	lineEndX := tx - arrowLen*math.Cos(angle)
	lineEndY := ty - arrowLen*math.Sin(angle)

	var sb strings.Builder
	fmt.Fprintf(&sb, `<path d="M%.1f,%.1f L%.1f,%.1f" fill="none" stroke="#555555" stroke-width="1.5"/>`,
		sx, sy, lineEndX, lineEndY)
	sb.WriteString(arrowheadSVG(tx, ty, angle, "#555555"))

	if event != "" {
		midX := (sx + tx) / 2
		midY := (sy + ty) / 2
		fmt.Fprintf(&sb, `<text x="%.1f" y="%.1f" text-anchor="middle" font-size="11" font-family="%s" fill="#555555">%s</text>`,
			midX, midY-6, defaultFontFamily, xmlEscape(event))
	}

	return sb.String()
}

// stateNoteSVG renders a note box to the right of a state node.
func stateNoteSVG(x, cy float64, text string) string {
	w := 120.0
	h := 30.0
	return fmt.Sprintf(
		`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="4" fill="#fef9c3" stroke="#ca8a04" stroke-width="1"/>` +
			`<text x="%.1f" y="%.1f" text-anchor="middle" dominant-baseline="central" font-size="11" font-family="%s" fill="#713f12">%s</text>`,
		x, cy-h/2, w, h,
		x+w/2, cy, defaultFontFamily, xmlEscape(text))
}
