package svg

import (
	"fmt"
	"io"
	"math"
	"strings"
	"unicode/utf8"

	diagram "github.com/iokdigital/go-mermaid"
	"github.com/iokdigital/go-mermaid/ast"
)

// Mindmap node geometry.
const (
	mindMinW  = 90.0
	mindMaxW  = 320.0
	mindCharW = 7.0
	mindPadX  = 28.0
	mindLineH = 20.0
	mindPadY  = 16.0
)

// mindLevelColors shades nodes by tree depth.
var mindLevelColors = [][2]string{
	{"#bfdbfe", "#1d4ed8"}, // root
	{"#dbeafe", "#3b82f6"},
	{"#eff6ff", "#93c5fd"},
	{"#f8fafc", "#cbd5e1"},
}

type mindLayoutNode struct {
	node  *ast.MindNode
	depth int
	idx   int
}

// encodeMindmap renders a *ast.MindmapDiagram to w as an SVG document.
// The tree is laid out left-to-right (root first) using the generic Sugiyama
// layout. This diverges from Mermaid's centered two-sided mindmap style,
// which is documented here. A nil root renders an empty diagram frame.
func encodeMindmap(w io.Writer, d *ast.MindmapDiagram, opts diagram.RenderOptions) error {
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

	var sb strings.Builder
	if d.Root() == nil {
		writeSVGHeader(&sb, padding*2+100, padding*2+60, opts, d.Title())
		sb.WriteString("</svg>\n")
		_, err := io.WriteString(w, sb.String())
		return err
	}

	// Flatten the tree preorder with unique layout ids. The walk drops nil
	// and repeated nodes so cyclic graphs cannot hang the layout.
	var flat []mindLayoutNode
	seen := make(map[*ast.MindNode]bool)
	var walk func(n *ast.MindNode, depth int)
	walk = func(n *ast.MindNode, depth int) {
		if n == nil || seen[n] {
			return
		}
		seen[n] = true
		flat = append(flat, mindLayoutNode{node: n, depth: depth, idx: len(flat)})
		for _, c := range n.Children {
			walk(c, depth+1)
		}
	}
	walk(d.Root(), 0)

	gnodes := make([]gNode, 0, len(flat))
	gedges := make([]gEdge, 0, len(flat))
	idOf := make(map[*ast.MindNode]string, len(flat))
	for _, fn := range flat {
		id := fmt.Sprintf("m%d", fn.idx)
		idOf[fn.node] = id
		nw, nh := mindNodeSize(fn.node.Label)
		gnodes = append(gnodes, gNode{ID: id, W: nw, H: nh})
	}
	linked := make(map[*ast.MindNode]bool)
	var link func(n *ast.MindNode)
	link = func(n *ast.MindNode) {
		if n == nil || linked[n] {
			return
		}
		linked[n] = true
		for _, c := range n.Children {
			to, ok := idOf[c]
			if !ok {
				continue
			}
			gedges = append(gedges, gEdge{From: idOf[n], To: to})
			link(c)
		}
	}
	link(d.Root())

	lr := runGenericLayout(gnodes, gedges, ast.DirectionLR, opts.Layout, padding, maxW, maxH)

	writeSVGHeader(&sb, lr.width, lr.height, opts, d.Title())

	// Edges first so nodes render on top.
	sb.WriteString(`  <g id="mindmap-edges">` + "\n")
	for _, e := range gedges {
		src, ok1 := lr.nodes[e.From]
		dst, ok2 := lr.nodes[e.To]
		if !ok1 || !ok2 {
			continue
		}
		x1 := src.x + src.w/2
		x2 := dst.x - dst.w/2
		fmt.Fprintf(&sb, `    <path d="M%.1f,%.1f L%.1f,%.1f" fill="none" stroke="#94a3b8" stroke-width="1.5"/>`+"\n",
			x1, src.y, x2, dst.y)
	}
	sb.WriteString("  </g>\n")

	sb.WriteString(`  <g id="mindmap-nodes">` + "\n")
	for _, fn := range flat {
		nl, ok := lr.nodes[idOf[fn.node]]
		if !ok {
			continue
		}
		sb.WriteString("    ")
		sb.WriteString(mindNodeSVG(fn, nl.x, nl.y))
		sb.WriteString("\n")
	}
	sb.WriteString("  </g>\n</svg>\n")

	_, err := io.WriteString(w, sb.String())
	return err
}

// mindNodeSize estimates node dimensions from the label text.
func mindNodeSize(label string) (float64, float64) {
	lines := textLines(label)
	maxLen := 0
	for _, l := range lines {
		if n := utf8.RuneCountInString(l); n > maxLen {
			maxLen = n
		}
	}
	w := float64(maxLen)*mindCharW + mindPadX
	w = math.Min(mindMaxW, math.Max(mindMinW, w))
	return w, float64(len(lines))*mindLineH + mindPadY
}

// mindNodeSVG returns SVG for one mindmap node (shape + label).
func mindNodeSVG(fn mindLayoutNode, cx, cy float64) string {
	w, h := mindNodeSize(fn.node.Label)
	colors := mindLevelColors[len(mindLevelColors)-1]
	if fn.depth < len(mindLevelColors) {
		colors = mindLevelColors[fn.depth]
	}
	shape := fn.node.Shape
	// Only mindmap-native shapes render distinctly; others fall back to the
	// same rounded shape the mmd encoder uses.
	switch shape {
	case ast.ShapeCircle, ast.ShapeRect, ast.ShapeHexagon, ast.ShapeRoundedRect, "":
		// Render natively.
	default:
		shape = ast.ShapeRoundedRect
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, `<g id="mind-%d">`, fn.idx)
	sb.WriteString(shapeElement(cx, cy, w, h, shape, colors[0], colors[1]))
	sb.WriteString(labelElement(cx, cy, textLines(fn.node.Label)))
	sb.WriteString(`</g>`)
	return sb.String()
}
