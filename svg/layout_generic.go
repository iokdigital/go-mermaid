package svg

import (
	"math"

	diagram "github.com/iokdigital/go-mermaid"
	"github.com/iokdigital/go-mermaid/ast"
)

// gNode is a generic graph node for the Sugiyama layout engine.
type gNode struct {
	ID string
	W  float64
	H  float64
}

// gEdge is a directed edge between two generic nodes.
type gEdge struct {
	From string
	To   string
}

// runGenericLayout executes the Sugiyama pipeline on abstract node/edge sets.
// Each node carries its own W/H, enabling variable-size nodes per rank.
// padding is applied on all four sides; maxW/maxH cap the viewport.
func runGenericLayout(nodes []gNode, edges []gEdge, dir ast.Direction, opts diagram.LayoutOptions, padding, maxW, maxH float64) *layoutResult {
	result := &layoutResult{
		nodes:    make(map[string]*nodeLayout, len(nodes)),
		reversed: make(map[edgeKey]bool),
	}

	if len(nodes) == 0 {
		result.width = math.Max(400, 2*padding)
		result.height = math.Max(300, 2*padding)
		return result
	}

	nodeByID := make(map[string]gNode, len(nodes))
	for _, n := range nodes {
		nodeByID[n.ID] = n
	}

	succs := make(map[string][]string, len(nodes))
	preds := make(map[string][]string, len(nodes))
	for _, n := range nodes {
		succs[n.ID] = nil
		preds[n.ID] = nil
	}
	for _, e := range edges {
		succs[e.From] = append(succs[e.From], e.To)
		preds[e.To] = append(preds[e.To], e.From)
	}

	// Step 1: DFS cycle detection — mark back-edges.
	color := make(map[string]int, len(nodes))
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

	// Build DAG adjacency (no back-edges).
	fwdSuccs := make(map[string][]string, len(nodes))
	fwdPreds := make(map[string][]string, len(nodes))
	for _, n := range nodes {
		fwdSuccs[n.ID] = nil
		fwdPreds[n.ID] = nil
	}
	for _, e := range edges {
		if !result.reversed[edgeKey{e.From, e.To}] {
			fwdSuccs[e.From] = append(fwdSuccs[e.From], e.To)
			fwdPreds[e.To] = append(fwdPreds[e.To], e.From)
		}
	}

	// Step 2: Longest-path rank assignment via Kahn topological sort.
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
	pos := make(map[string]int, len(nodes))
	for _, ids := range byRank {
		for i, id := range ids {
			pos[id] = i
		}
	}

	// Step 3: Barycenter crossing reduction — 2 passes.
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

	// Step 4: Coordinate assignment with per-rank variable dimensions.
	maxPositions := 0
	for _, ids := range byRank {
		if len(ids) > maxPositions {
			maxPositions = len(ids)
		}
	}
	if maxPositions == 0 {
		maxPositions = 1
	}

	// Per-rank max height (for TB row height) and max width (for LR column width).
	rankMaxH := make([]float64, maxRank+1)
	rankMaxW := make([]float64, maxRank+1)
	for r, ids := range byRank {
		for _, id := range ids {
			n := nodeByID[id]
			if n.H > rankMaxH[r] {
				rankMaxH[r] = n.H
			}
			if n.W > rankMaxW[r] {
				rankMaxW[r] = n.W
			}
		}
	}

	// Global max for cross-axis spacing (columns in TB, rows in LR).
	globalMaxW := 0.0
	globalMaxH := 0.0
	for _, n := range nodes {
		if n.W > globalMaxW {
			globalMaxW = n.W
		}
		if n.H > globalMaxH {
			globalMaxH = n.H
		}
	}

	sh := float64(opts.NodeSpacingH)
	sv := float64(opts.NodeSpacingV)
	rs := float64(opts.RankSpacing)

	// Cumulative rank offsets along the primary axis.
	rankCumY := make([]float64, maxRank+1)
	{
		cum := padding
		for r := 0; r <= maxRank; r++ {
			rankCumY[r] = cum
			cum += rankMaxH[r] + rs
		}
	}
	rankCumX := make([]float64, maxRank+1)
	{
		cum := padding
		for r := 0; r <= maxRank; r++ {
			rankCumX[r] = cum
			cum += rankMaxW[r] + rs
		}
	}

	for r, ids := range byRank {
		nNodes := len(ids)
		offset := (float64(maxPositions) - float64(nNodes)) / 2.0
		for p, id := range ids {
			ap := float64(p) + offset
			n := nodeByID[id]
			var cx, cy float64
			switch dir {
			case ast.DirectionLR:
				cx = rankCumX[r] + n.W/2
				cy = padding + ap*(globalMaxH+sv) + globalMaxH/2
			case ast.DirectionRL:
				cx = rankCumX[maxRank-r] + n.W/2
				cy = padding + ap*(globalMaxH+sv) + globalMaxH/2
			case ast.DirectionBT:
				cx = padding + ap*(globalMaxW+sh) + globalMaxW/2
				cy = rankCumY[maxRank-r] + n.H/2
			default: // TB
				cx = padding + ap*(globalMaxW+sh) + globalMaxW/2
				cy = rankCumY[r] + n.H/2
			}
			result.nodes[id] = &nodeLayout{
				id:   id,
				rank: r,
				pos:  p,
				x:    cx,
				y:    cy,
				w:    n.W,
				h:    n.H,
			}
		}
	}

	// Viewport dimensions.
	var contentW, contentH float64
	switch dir {
	case ast.DirectionLR, ast.DirectionRL:
		for r := 0; r <= maxRank; r++ {
			contentW += rankMaxW[r]
		}
		contentW += float64(maxRank) * rs
		contentH = float64(maxPositions)*globalMaxH + float64(maxPositions-1)*sv
	default: // TB, BT
		for r := 0; r <= maxRank; r++ {
			contentH += rankMaxH[r]
		}
		contentH += float64(maxRank) * rs
		contentW = float64(maxPositions)*globalMaxW + float64(maxPositions-1)*sh
	}
	result.width = clamp(contentW+2*padding, 400, maxW)
	result.height = clamp(contentH+2*padding, 300, maxH)

	return result
}
