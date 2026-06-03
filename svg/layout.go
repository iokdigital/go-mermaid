package svg

import (
	"sort"

	diagram "github.com/iokdigital/go-mermaid"
	"github.com/iokdigital/go-mermaid/ast"
)

type nodeLayout struct {
	id   string
	rank int
	pos  int
	x    float64
	y    float64
	w    float64
	h    float64
}

type edgeKey struct{ from, to string }

type layoutResult struct {
	nodes    map[string]*nodeLayout
	reversed map[edgeKey]bool // back-edges (cycle members)
	width    float64
	height   float64
}

// runLayout executes the Sugiyama pipeline on a FlowchartDiagram.
// It converts the diagram to generic node/edge lists (all nodes uniform 120×40)
// and delegates to runGenericLayout.
func runLayout(f *ast.FlowchartDiagram, opts diagram.LayoutOptions, padding, maxW, maxH float64) *layoutResult {
	gnodes := make([]gNode, len(f.Nodes()))
	for i, n := range f.Nodes() {
		gnodes[i] = gNode{ID: n.ID, W: nodeWidth, H: nodeHeight}
	}
	gedges := make([]gEdge, 0, len(f.Edges()))
	for _, e := range f.Edges() {
		if e.Style != ast.EdgeInvisible {
			gedges = append(gedges, gEdge{From: e.From, To: e.To})
		}
	}
	return runGenericLayout(gnodes, gedges, f.Direction(), opts, padding, maxW, maxH)
}

func sortedByBary(ids []string, bary []float64) []string {
	idx := make([]int, len(ids))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool {
		return bary[idx[a]] < bary[idx[b]]
	})
	out := make([]string, len(ids))
	for i, j := range idx {
		out[i] = ids[j]
	}
	return out
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

