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

	// mindLevelGap is the vertical gap between stacked sibling subtrees;
	// mindLevelGapX is the horizontal distance between depth ranks.
	mindLevelGap  = 16.0
	mindLevelGapX = 120.0
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

// mindLayoutTwoSided lays the tree out Mermaid-style: the root horizontally
// centered, level-1 subtrees alternating right/left (first child right),
// each side stacked top-down, and the root's vertical center at the midpoint
// of the taller side. Nodes are positioned at their centers in a
// layoutResult-shaped map (plus width/height) so the edge and node emission
// code is untouched. The seen/idOf guards keep cyclic or repeated nodes from
// hanging or mispositioning the layout.
func mindLayoutTwoSided(flat []mindLayoutNode, root *ast.MindNode, padding float64) *layoutResult {
	result := &layoutResult{
		nodes:    make(map[string]*nodeLayout, len(flat)),
		reversed: make(map[edgeKey]bool),
	}
	if root == nil || len(flat) == 0 {
		result.width = math.Max(400, 2*padding)
		result.height = math.Max(300, 2*padding)
		return result
	}

	idOf := make(map[*ast.MindNode]string, len(flat))
	sizeOf := make(map[*ast.MindNode][2]float64, len(flat))
	for _, fn := range flat {
		idOf[fn.node] = fmt.Sprintf("m%d", fn.idx)
		w, h := mindNodeSize(fn.node.Label)
		sizeOf[fn.node] = [2]float64{w, h}
	}

	// Bottom-up subtree extents (guarded like the flatten walk; children
	// dropped from flat by the walk contribute nothing).
	subH := make(map[*ast.MindNode]float64, len(flat))
	seen := make(map[*ast.MindNode]bool, len(flat))
	var measure func(n *ast.MindNode) float64
	measure = func(n *ast.MindNode) float64 {
		if n == nil || seen[n] {
			return 0
		}
		seen[n] = true
		h := sizeOf[n][1]
		if _, ok := idOf[n]; !ok {
			return 0
		}
		total, count := 0.0, 0
		for _, c := range n.Children {
			if _, ok := idOf[c]; !ok {
				continue
			}
			total += measure(c)
			count++
		}
		if count > 0 {
			total += float64(count-1) * mindLevelGap
		}
		subH[n] = math.Max(h, total)
		return subH[n]
	}
	measure(root)

	// Split root children alternating right/left by original index (0 right).
	var rightKids, leftKids []*ast.MindNode
	for i, c := range root.Children {
		if _, ok := idOf[c]; !ok {
			continue
		}
		if i%2 == 0 {
			rightKids = append(rightKids, c)
		} else {
			leftKids = append(leftKids, c)
		}
	}
	sideHeight := func(kids []*ast.MindNode) float64 {
		total := 0.0
		for i, k := range kids {
			if i > 0 {
				total += mindLevelGap
			}
			total += subH[k]
		}
		return total
	}

	// Canvas: root centered; ranks at fixed depth offsets from center.
	visited := make(map[*ast.MindNode]bool, len(flat))
	maxDepth, maxNodeW, maxNodeH := 0, 0.0, 0.0
	var scan func(n *ast.MindNode, depth int)
	scan = func(n *ast.MindNode, depth int) {
		if n == nil || visited[n] {
			return
		}
		if _, ok := idOf[n]; !ok {
			return
		}
		visited[n] = true
		if depth > maxDepth {
			maxDepth = depth
		}
		if s := sizeOf[n]; s[0] > maxNodeW {
			maxNodeW = s[0]
		}
		if s := sizeOf[n]; s[1] > maxNodeH {
			maxNodeH = s[1]
		}
		for _, c := range n.Children {
			scan(c, depth+1)
		}
	}
	scan(root, 0)

	halfSpread := float64(maxDepth) * mindLevelGapX
	result.width = math.Max(400, 2*padding+2*halfSpread+maxNodeW)
	contentH := math.Max(math.Max(sideHeight(rightKids), sideHeight(leftKids)), maxNodeH)
	result.height = math.Max(300, contentH+2*padding)
	centerX := result.width / 2
	rootCy := padding + contentH/2

	// Place a subtree: node center at its rank, children stacked around it.
	// placed guards cycles (same misuse pattern the flatten walk drops).
	placed := make(map[*ast.MindNode]bool, len(flat))
	var place func(n *ast.MindNode, depth int, side, cy float64)
	place = func(n *ast.MindNode, depth int, side, cy float64) {
		id, ok := idOf[n]
		if !ok || placed[n] {
			return
		}
		placed[n] = true
		s := sizeOf[n]
		result.nodes[id] = &nodeLayout{
			id: id,
			x:  centerX + side*float64(depth)*mindLevelGapX,
			y:  cy,
			w:  s[0],
			h:  s[1],
		}
		var kids []*ast.MindNode
		for _, c := range n.Children {
			if _, ok := idOf[c]; !ok {
				continue
			}
			kids = append(kids, c)
		}
		top := cy - (sideHeight(kids))/2
		for _, c := range kids {
			place(c, depth+1, side, top+subH[c]/2)
			top += subH[c] + mindLevelGap
		}
	}
	// Root at depth 0; each side's top-level children stack around the
	// root's vertical center, mirroring Mermaid's alternation.
	result.nodes[idOf[root]] = &nodeLayout{
		id: idOf[root],
		x:  centerX,
		y:  rootCy,
		w:  sizeOf[root][0],
		h:  sizeOf[root][1],
	}
	placeSide := func(kids []*ast.MindNode, side float64) {
		top := rootCy - sideHeight(kids)/2
		for _, c := range kids {
			place(c, 1, side, top+subH[c]/2)
			top += subH[c] + mindLevelGap
		}
	}
	placeSide(rightKids, 1)
	placeSide(leftKids, -1)

	return result
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

	lr := mindLayoutTwoSided(flat, d.Root(), padding)

	writeSVGHeader(&sb, lr.width, lr.height, opts, d.Title())

	// Edges first so nodes render on top. Mermaid-style horizontal elbows:
	// parent edge-midpoint → horizontal → vertical trunk → horizontal →
	// child (single M/H/V/H path; the trunk's x is the midpoint between the
	// two anchor edges, which spans the center gap for root-to-first-level).
	sb.WriteString(`  <g id="mindmap-edges">` + "\n")
	for _, e := range gedges {
		src, ok1 := lr.nodes[e.From]
		dst, ok2 := lr.nodes[e.To]
		if !ok1 || !ok2 {
			continue
		}
		var x1, x2 float64
		if dst.x >= src.x {
			x1 = src.x + src.w/2
			x2 = dst.x - dst.w/2
		} else {
			x1 = src.x - src.w/2
			x2 = dst.x + dst.w/2
		}
		midx := (x1 + x2) / 2
		fmt.Fprintf(&sb, `    <path d="M%.1f,%.1f H%.1f V%.1f H%.1f" fill="none" stroke="#94a3b8" stroke-width="1.5"/>`+"\n",
			x1, src.y, midx, dst.y, x2)
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

// mindSubtreeHeight returns the vertical extent of the subtree rooted at n:
// a leaf's extent is its node height; a parent's extent is
// max(children total + gaps, own height). The seen map guards cycles (same
// pattern as the flatten walk).
func mindSubtreeHeight(n *ast.MindNode, size func(string) (float64, float64), seen map[*ast.MindNode]bool) float64 {
	if n == nil || seen[n] {
		return 0
	}
	seen[n] = true
	_, h := size(n.Label)
	if len(n.Children) == 0 {
		return h
	}
	total := 0.0
	for _, c := range n.Children {
		total += mindSubtreeHeight(c, size, seen)
	}
	total += float64(len(n.Children)-1) * mindLevelGap
	return math.Max(h, total)
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
