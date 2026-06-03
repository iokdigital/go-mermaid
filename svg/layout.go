package svg

import (
	"math"
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

// runLayout executes the Sugiyama pipeline on fc and returns positioned nodes.
// padding is applied on all four sides; maxW/maxH cap the viewport.
func runLayout(f *ast.FlowchartDiagram, opts diagram.LayoutOptions, padding, maxW, maxH float64) *layoutResult {
	nodes := f.Nodes()
	edges := f.Edges()

	result := &layoutResult{
		nodes:    make(map[string]*nodeLayout, len(nodes)),
		reversed: make(map[edgeKey]bool),
	}

	if len(nodes) == 0 {
		result.width = math.Max(400, 2*padding)
		result.height = math.Max(300, 2*padding)
		return result
	}

	// Build forward adjacency (excluding invisible edges).
	succs := make(map[string][]string, len(nodes))
	preds := make(map[string][]string, len(nodes))
	for _, n := range nodes {
		succs[n.ID] = nil
		preds[n.ID] = nil
	}
	for _, e := range edges {
		if e.Style == ast.EdgeInvisible {
			continue
		}
		succs[e.From] = append(succs[e.From], e.To)
		preds[e.To] = append(preds[e.To], e.From)
	}

	// Step 1: DFS cycle detection — mark back-edges.
	color := make(map[string]int, len(nodes)) // 0=white, 1=gray, 2=black
	var dfs func(u string)
	dfs = func(u string) {
		color[u] = 1
		for _, v := range succs[u] {
			switch color[v] {
			case 1:
				result.reversed[edgeKey{u, v}] = true
			case 0:
				dfs(v)
			}
		}
		color[u] = 2
	}
	for _, n := range nodes {
		if color[n.ID] == 0 {
			dfs(n.ID)
		}
	}

	// Build DAG adjacency (forward edges only, no back-edges).
	fwdSuccs := make(map[string][]string, len(nodes))
	fwdPreds := make(map[string][]string, len(nodes))
	for _, n := range nodes {
		fwdSuccs[n.ID] = nil
		fwdPreds[n.ID] = nil
	}
	for _, e := range edges {
		if e.Style == ast.EdgeInvisible {
			continue
		}
		if !result.reversed[edgeKey{e.From, e.To}] {
			fwdSuccs[e.From] = append(fwdSuccs[e.From], e.To)
			fwdPreds[e.To] = append(fwdPreds[e.To], e.From)
		}
	}

	// Step 2: Longest-path layer assignment via Kahn topological sort.
	rank := make(map[string]int, len(nodes))
	inDeg := make(map[string]int, len(nodes))
	for _, n := range nodes {
		for _, v := range fwdSuccs[n.ID] {
			inDeg[v]++
		}
	}
	queue := make([]string, 0, len(nodes))
	for _, n := range nodes {
		if inDeg[n.ID] == 0 {
			queue = append(queue, n.ID)
		}
	}
	for len(queue) > 0 {
		u := queue[0]
		queue = queue[1:]
		for _, v := range fwdSuccs[u] {
			if rank[v] < rank[u]+1 {
				rank[v] = rank[u] + 1
			}
			inDeg[v]--
			if inDeg[v] == 0 {
				queue = append(queue, v)
			}
		}
	}

	// Group nodes by rank.
	maxRank := 0
	for _, n := range nodes {
		if rank[n.ID] > maxRank {
			maxRank = rank[n.ID]
		}
	}
	byRank := make([][]string, maxRank+1)
	for _, n := range nodes {
		byRank[rank[n.ID]] = append(byRank[rank[n.ID]], n.ID)
	}

	// Initial positions within each rank (preserve insertion order).
	pos := make(map[string]int, len(nodes))
	for _, ids := range byRank {
		for i, id := range ids {
			pos[id] = i
		}
	}

	// Step 3: Barycenter crossing reduction — 2 passes (forward then backward).
	for pass := 0; pass < 2; pass++ {
		if pass%2 == 0 {
			for r := 1; r <= maxRank; r++ {
				ids := byRank[r]
				bary := make([]float64, len(ids))
				for i, id := range ids {
					sum, cnt := 0.0, 0
					for _, p := range fwdPreds[id] {
						if rank[p] == r-1 {
							sum += float64(pos[p])
							cnt++
						}
					}
					if cnt > 0 {
						bary[i] = sum / float64(cnt)
					} else {
						bary[i] = float64(pos[id])
					}
				}
				sorted := sortedByBary(ids, bary)
				byRank[r] = sorted
				for i, id := range sorted {
					pos[id] = i
				}
			}
		} else {
			for r := maxRank - 1; r >= 0; r-- {
				ids := byRank[r]
				bary := make([]float64, len(ids))
				for i, id := range ids {
					sum, cnt := 0.0, 0
					for _, s := range fwdSuccs[id] {
						if rank[s] == r+1 {
							sum += float64(pos[s])
							cnt++
						}
					}
					if cnt > 0 {
						bary[i] = sum / float64(cnt)
					} else {
						bary[i] = float64(pos[id])
					}
				}
				sorted := sortedByBary(ids, bary)
				byRank[r] = sorted
				for i, id := range sorted {
					pos[id] = i
				}
			}
		}
	}

	// Step 4: Coordinate assignment with rank centering.
	maxWidth := 0
	for _, ids := range byRank {
		if len(ids) > maxWidth {
			maxWidth = len(ids)
		}
	}
	if maxWidth == 0 {
		maxWidth = 1
	}

	nw := nodeWidth
	nh := nodeHeight
	sh := float64(opts.NodeSpacingH)
	sv := float64(opts.NodeSpacingV)
	rs := float64(opts.RankSpacing)
	dir := f.Direction()

	for r, ids := range byRank {
		nNodes := len(ids)
		offset := (float64(maxWidth) - float64(nNodes)) / 2.0
		for p, id := range ids {
			ap := float64(p) + offset
			var cx, cy float64
			switch dir {
			case ast.DirectionLR:
				cx = padding + float64(r)*(nw+rs) + nw/2
				cy = padding + ap*(nh+sv) + nh/2
			case ast.DirectionRL:
				cx = padding + float64(maxRank-r)*(nw+rs) + nw/2
				cy = padding + ap*(nh+sv) + nh/2
			case ast.DirectionBT:
				cx = padding + ap*(nw+sh) + nw/2
				cy = padding + float64(maxRank-r)*(nh+rs) + nh/2
			default: // TB
				cx = padding + ap*(nw+sh) + nw/2
				cy = padding + float64(r)*(nh+rs) + nh/2
			}
			result.nodes[id] = &nodeLayout{
				id:   id,
				rank: r,
				pos:  p,
				x:    cx,
				y:    cy,
				w:    nw,
				h:    nh,
			}
		}
	}

	// Compute viewport dimensions.
	var contentW, contentH float64
	switch dir {
	case ast.DirectionLR, ast.DirectionRL:
		contentW = float64(maxRank+1)*(nw+rs) - rs
		contentH = float64(maxWidth)*(nh+sv) - sv
	default: // TB, BT
		contentW = float64(maxWidth)*(nw+sh) - sh
		contentH = float64(maxRank+1)*(nh+rs) - rs
	}
	result.width = clamp(contentW+2*padding, 400, maxW)
	result.height = clamp(contentH+2*padding, 300, maxH)

	return result
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
