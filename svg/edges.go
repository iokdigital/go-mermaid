package svg

import (
	"fmt"
	"math"
	"strings"

	"github.com/iokdigital/go-mermaid/ast"
)

// edgeSVG returns SVG markup for a single edge (polyline + arrowhead + label).
// Returns "" for invisible edges or edges whose nodes are not in the layout.
func edgeSVG(e *ast.FlowEdge, lr *layoutResult, dir ast.Direction) string {
	if e.Style == ast.EdgeInvisible {
		return ""
	}
	src, ok1 := lr.nodes[e.From]
	dst, ok2 := lr.nodes[e.To]
	if !ok1 || !ok2 {
		return ""
	}

	isSelfLoop := e.From == e.To
	isBack := lr.reversed[edgeKey{e.From, e.To}]

	strokeColor := "#555555"
	sw := 1.5
	dash := ""
	if e.Style == ast.EdgeDotted || isBack {
		dash = `stroke-dasharray="5,3" `
	}
	if e.Style == ast.EdgeThick {
		sw = 3.0
	}

	if isSelfLoop {
		return selfLoopSVG(src, strokeColor, sw, dash)
	}
	if isBack {
		return backEdgeSVG(e, src, dst, lr, strokeColor, sw)
	}

	// Forward edge: exit/entry border points.
	sx, sy := exitPoint(src, dir)
	tx, ty := entryPoint(dst, dir)

	pts := rankWaypoints(src, dst, sx, sy, tx, ty, dir)

	// Save original (unshortened) pts for label placement before we mutate.
	origPts := make([][2]float64, len(pts))
	copy(origPts, pts)

	// Shorten the final segment to make room for the arrowhead tip.
	var lineEnd [2]float64
	if len(pts) >= 2 {
		last := pts[len(pts)-1]
		prev := pts[len(pts)-2]
		angle := math.Atan2(last[1]-prev[1], last[0]-prev[0])
		lineEnd = [2]float64{
			last[0] - arrowLen*math.Cos(angle),
			last[1] - arrowLen*math.Sin(angle),
		}
		pts[len(pts)-1] = lineEnd
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<path d="%s" fill="none" stroke="%s" stroke-width="%.1f" %s/>`,
		pointsToPath(pts), strokeColor, sw, dash))

	if e.Style != ast.EdgeNoArrow && len(pts) >= 2 {
		last := pts[len(pts)-1]
		prev := pts[len(pts)-2]
		angle := math.Atan2(last[1]-prev[1], last[0]-prev[0])
		// tip is the original (un-shortened) endpoint
		tip := pts[len(pts)-1]
		tip[0] += arrowLen * math.Cos(angle)
		tip[1] += arrowLen * math.Sin(angle)
		sb.WriteString(arrowheadSVG(tip[0], tip[1], angle, strokeColor))
	}

	if e.Label != "" {
		mid := polylineMidpoint(origPts)
		fmt.Fprintf(&sb, `<text x="%.1f" y="%.1f" text-anchor="middle" font-size="11" font-family="%s" fill="#555555">%s</text>`,
			mid[0], mid[1]-6, defaultFontFamily, xmlEscape(e.Label))
	}

	return sb.String()
}

// exitPoint returns the border point where an edge departs from src.
func exitPoint(src *nodeLayout, dir ast.Direction) (float64, float64) {
	switch dir {
	case ast.DirectionLR:
		return src.x + src.w/2, src.y
	case ast.DirectionRL:
		return src.x - src.w/2, src.y
	case ast.DirectionBT:
		return src.x, src.y - src.h/2
	default: // TB
		return src.x, src.y + src.h/2
	}
}

// entryPoint returns the border point where an edge arrives at dst.
func entryPoint(dst *nodeLayout, dir ast.Direction) (float64, float64) {
	switch dir {
	case ast.DirectionLR:
		return dst.x - dst.w/2, dst.y
	case ast.DirectionRL:
		return dst.x + dst.w/2, dst.y
	case ast.DirectionBT:
		return dst.x, dst.y + dst.h/2
	default: // TB
		return dst.x, dst.y - dst.h/2
	}
}

// rankWaypoints builds the polyline point list including rank-boundary waypoints.
func rankWaypoints(src, dst *nodeLayout, sx, sy, tx, ty float64, _ ast.Direction) [][2]float64 {
	pts := [][2]float64{{sx, sy}}
	rankDiff := dst.rank - src.rank
	if rankDiff > 1 {
		for step := 1; step < rankDiff; step++ {
			t := float64(step) / float64(rankDiff)
			pts = append(pts, [2]float64{
				sx + (tx-sx)*t,
				sy + (ty-sy)*t,
			})
		}
	}
	pts = append(pts, [2]float64{tx, ty})
	return pts
}

// arrowheadSVG returns a filled triangle polygon at tip (tx,ty) with approach angle θ.
func arrowheadSVG(tx, ty, angle float64, fill string) string {
	cos, sin := math.Cos(angle), math.Sin(angle)
	bx := tx - arrowLen*cos
	by := ty - arrowLen*sin
	lx := bx + (arrowWidth/2)*sin
	ly := by - (arrowWidth/2)*cos
	rx := bx - (arrowWidth/2)*sin
	ry := by + (arrowWidth/2)*cos
	return fmt.Sprintf(`<polygon points="%.1f,%.1f %.1f,%.1f %.1f,%.1f" fill="%s"/>`,
		tx, ty, lx, ly, rx, ry, fill)
}

// selfLoopSVG renders a self-loop arc above/left of a node.
func selfLoopSVG(n *nodeLayout, stroke string, sw float64, dash string) string {
	x, y := n.x, n.y-n.h/2
	r := 15.0
	return fmt.Sprintf(
		`<path d="M%.1f,%.1f A%.1f,%.1f 0 1,1 %.1f,%.1f" fill="none" stroke="%s" stroke-width="%.1f" %s/>`+
			`<text x="%.1f" y="%.1f" text-anchor="middle" font-size="12" font-family="%s" fill="%s">⟳</text>`,
		x-r, y, r, r, x+r, y, stroke, sw, dash,
		x, y-2*r-4, defaultFontFamily, stroke)
}

// backEdgeSVG renders a back-edge as a dashed polyline routed to the left of the diagram.
func backEdgeSVG(e *ast.FlowEdge, src, dst *nodeLayout, lr *layoutResult, stroke string, sw float64) string {
	// Find leftmost x across all nodes.
	leftmost := src.x - src.w/2
	for _, n := range lr.nodes {
		if n.x-n.w/2 < leftmost {
			leftmost = n.x - n.w/2
		}
	}
	detour := leftmost - 30

	sx := src.x - src.w/2
	sy := src.y
	tx := dst.x - dst.w/2
	ty := dst.y

	var sb strings.Builder
	fmt.Fprintf(&sb,
		`<path d="M%.1f,%.1f L%.1f,%.1f L%.1f,%.1f L%.1f,%.1f" fill="none" stroke="%s" stroke-width="%.1f" stroke-dasharray="5,3"/>`,
		sx, sy, detour, sy, detour, ty, tx+arrowLen, ty, stroke, sw)
	if e.Style != ast.EdgeNoArrow {
		sb.WriteString(arrowheadSVG(tx, ty, 0, stroke))
	}
	return sb.String()
}

// pointsToPath converts a sequence of points to an SVG path d attribute value
// using M for the first point and L for subsequent points.
// Using <path d="M L..."> instead of <polyline> for oksvg PNG rasterizer compatibility.
func pointsToPath(pts [][2]float64) string {
	if len(pts) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, p := range pts {
		if i == 0 {
			fmt.Fprintf(&sb, "M%.1f,%.1f", p[0], p[1])
		} else {
			fmt.Fprintf(&sb, " L%.1f,%.1f", p[0], p[1])
		}
	}
	return sb.String()
}

// polylineMidpoint returns the arc-length midpoint of pts.
func polylineMidpoint(pts [][2]float64) [2]float64 {
	if len(pts) == 0 {
		return [2]float64{}
	}
	if len(pts) == 1 {
		return pts[0]
	}
	total := 0.0
	for i := 1; i < len(pts); i++ {
		dx := pts[i][0] - pts[i-1][0]
		dy := pts[i][1] - pts[i-1][1]
		total += math.Sqrt(dx*dx + dy*dy)
	}
	half := total / 2
	walked := 0.0
	for i := 1; i < len(pts); i++ {
		dx := pts[i][0] - pts[i-1][0]
		dy := pts[i][1] - pts[i-1][1]
		seg := math.Sqrt(dx*dx + dy*dy)
		if walked+seg >= half {
			t := (half - walked) / seg
			return [2]float64{pts[i-1][0] + t*dx, pts[i-1][1] + t*dy}
		}
		walked += seg
	}
	return pts[len(pts)-1]
}
