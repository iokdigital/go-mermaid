package svg

import (
	"fmt"
	"testing"

	diagram "github.com/iokdigital/go-mermaid"
	"github.com/iokdigital/go-mermaid/ast"
)

func defaultOpts() diagram.LayoutOptions {
	return diagram.DefaultLayoutOptions()
}

func TestLayout_EmptyDiagram(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionTB)
	lr := runLayout(f, defaultOpts(), 40, 8000, 6000)
	if len(lr.nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(lr.nodes))
	}
	if lr.width < 400 {
		t.Errorf("width should be clamped to min 400, got %.1f", lr.width)
	}
	if lr.height < 300 {
		t.Errorf("height should be clamped to min 300, got %.1f", lr.height)
	}
}

func TestLayout_SingleNode(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A", Label: "Alpha", Shape: ast.ShapeRect})
	lr := runLayout(f, defaultOpts(), 40, 8000, 6000)
	if len(lr.nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(lr.nodes))
	}
	n := lr.nodes["A"]
	if n == nil {
		t.Fatal("node A not found in layout")
	}
	if n.rank != 0 {
		t.Errorf("single node should be at rank 0, got %d", n.rank)
	}
}

func TestLayout_LinearChain_Ranks(t *testing.T) {
	// A → B → C should produce ranks 0, 1, 2
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A", Shape: ast.ShapeRect})
	f.MustAddNode(&ast.FlowNode{ID: "B", Shape: ast.ShapeRect})
	f.MustAddNode(&ast.FlowNode{ID: "C", Shape: ast.ShapeRect})
	f.AddEdge(&ast.FlowEdge{From: "A", To: "B", Style: ast.EdgeSolid})
	f.AddEdge(&ast.FlowEdge{From: "B", To: "C", Style: ast.EdgeSolid})

	lr := runLayout(f, defaultOpts(), 40, 8000, 6000)
	if lr.nodes["A"].rank != 0 {
		t.Errorf("A rank: want 0, got %d", lr.nodes["A"].rank)
	}
	if lr.nodes["B"].rank != 1 {
		t.Errorf("B rank: want 1, got %d", lr.nodes["B"].rank)
	}
	if lr.nodes["C"].rank != 2 {
		t.Errorf("C rank: want 2, got %d", lr.nodes["C"].rank)
	}
}

func TestLayout_LongestPath(t *testing.T) {
	// A → C, B → C: both A and B at rank 0, C at rank 1
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A", Shape: ast.ShapeRect})
	f.MustAddNode(&ast.FlowNode{ID: "B", Shape: ast.ShapeRect})
	f.MustAddNode(&ast.FlowNode{ID: "C", Shape: ast.ShapeRect})
	f.AddEdge(&ast.FlowEdge{From: "A", To: "C", Style: ast.EdgeSolid})
	f.AddEdge(&ast.FlowEdge{From: "B", To: "C", Style: ast.EdgeSolid})

	lr := runLayout(f, defaultOpts(), 40, 8000, 6000)
	if lr.nodes["A"].rank != 0 {
		t.Errorf("A rank: want 0, got %d", lr.nodes["A"].rank)
	}
	if lr.nodes["B"].rank != 0 {
		t.Errorf("B rank: want 0, got %d", lr.nodes["B"].rank)
	}
	if lr.nodes["C"].rank != 1 {
		t.Errorf("C rank: want 1, got %d", lr.nodes["C"].rank)
	}
}

func TestLayout_Diamond_Ranks(t *testing.T) {
	// A → B, A → C, B → D, C → D (diamond)
	f := ast.NewFlowchart("", ast.DirectionTB)
	for _, id := range []string{"A", "B", "C", "D"} {
		f.MustAddNode(&ast.FlowNode{ID: id, Shape: ast.ShapeRect})
	}
	f.AddEdge(&ast.FlowEdge{From: "A", To: "B", Style: ast.EdgeSolid})
	f.AddEdge(&ast.FlowEdge{From: "A", To: "C", Style: ast.EdgeSolid})
	f.AddEdge(&ast.FlowEdge{From: "B", To: "D", Style: ast.EdgeSolid})
	f.AddEdge(&ast.FlowEdge{From: "C", To: "D", Style: ast.EdgeSolid})

	lr := runLayout(f, defaultOpts(), 40, 8000, 6000)
	if lr.nodes["A"].rank != 0 {
		t.Errorf("A rank: want 0, got %d", lr.nodes["A"].rank)
	}
	if lr.nodes["B"].rank != 1 {
		t.Errorf("B rank: want 1, got %d", lr.nodes["B"].rank)
	}
	if lr.nodes["C"].rank != 1 {
		t.Errorf("C rank: want 1, got %d", lr.nodes["C"].rank)
	}
	if lr.nodes["D"].rank != 2 {
		t.Errorf("D rank: want 2, got %d", lr.nodes["D"].rank)
	}
}

func TestLayout_CycleDetected_NodesGetRanks(t *testing.T) {
	// A → B → A is a cycle; layout should still assign ranks to all nodes.
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A", Shape: ast.ShapeRect})
	f.MustAddNode(&ast.FlowNode{ID: "B", Shape: ast.ShapeRect})
	f.AddEdge(&ast.FlowEdge{From: "A", To: "B", Style: ast.EdgeSolid})
	f.AddEdge(&ast.FlowEdge{From: "B", To: "A", Style: ast.EdgeSolid})

	lr := runLayout(f, defaultOpts(), 40, 8000, 6000)
	if len(lr.nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(lr.nodes))
	}
	if len(lr.reversed) == 0 {
		t.Error("expected at least one back-edge to be marked")
	}
	for id, n := range lr.nodes {
		if n.x <= 0 || n.y <= 0 {
			t.Errorf("node %s has non-positive coordinates (%.1f, %.1f)", id, n.x, n.y)
		}
	}
}

func TestLayout_AllNodesHavePositiveCoords(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionLR)
	ids := []string{"X", "Y", "Z"}
	for _, id := range ids {
		f.MustAddNode(&ast.FlowNode{ID: id, Shape: ast.ShapeRect})
	}
	f.AddEdge(&ast.FlowEdge{From: "X", To: "Y", Style: ast.EdgeSolid})
	f.AddEdge(&ast.FlowEdge{From: "Y", To: "Z", Style: ast.EdgeSolid})

	lr := runLayout(f, defaultOpts(), 40, 8000, 6000)
	for _, id := range ids {
		n := lr.nodes[id]
		if n == nil {
			t.Errorf("node %s missing from layout", id)
			continue
		}
		if n.x <= 0 || n.y <= 0 {
			t.Errorf("node %s: expected positive coords, got (%.1f,%.1f)", id, n.x, n.y)
		}
	}
}

func TestLayout_ViewportClamped(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A", Shape: ast.ShapeRect})

	lr := runLayout(f, defaultOpts(), 40, 8000, 6000)
	if lr.width < 400 || lr.width > 8000 {
		t.Errorf("width %.1f out of [400, 8000]", lr.width)
	}
	if lr.height < 300 || lr.height > 6000 {
		t.Errorf("height %.1f out of [300, 6000]", lr.height)
	}
}

func TestLayout_InvisibleEdgesExcluded(t *testing.T) {
	// A ~~~ B: invisible edge should not affect rank assignment (B stays at rank 0)
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A", Shape: ast.ShapeRect})
	f.MustAddNode(&ast.FlowNode{ID: "B", Shape: ast.ShapeRect})
	f.AddEdge(&ast.FlowEdge{From: "A", To: "B", Style: ast.EdgeInvisible})

	lr := runLayout(f, defaultOpts(), 40, 8000, 6000)
	// With invisible edge excluded from layout graph, both A and B are sources.
	// Both should be at rank 0.
	if lr.nodes["A"].rank != 0 || lr.nodes["B"].rank != 0 {
		t.Errorf("invisible edge should not affect rank: A=%d B=%d", lr.nodes["A"].rank, lr.nodes["B"].rank)
	}
}

// BenchmarkLayout50Nodes measures layout time for a 50-node linear chain.
// FRD §10.6 target: < 5 ms.
func BenchmarkLayout50Nodes(b *testing.B) {
	f := makeLinearChain(50)
	opts := defaultOpts()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runLayout(f, opts, 40, 8000, 6000)
	}
}

// BenchmarkLayout200Nodes measures layout time for a 200-node linear chain.
// FRD §10.6 target: < 50 ms.
func BenchmarkLayout200Nodes(b *testing.B) {
	f := makeLinearChain(200)
	opts := defaultOpts()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runLayout(f, opts, 40, 8000, 6000)
	}
}

func makeLinearChain(n int) *ast.FlowchartDiagram {
	f := ast.NewFlowchart("", ast.DirectionTB)
	for i := 0; i < n; i++ {
		f.MustAddNode(&ast.FlowNode{ID: fmt.Sprintf("N%d", i), Shape: ast.ShapeRect})
	}
	for i := 0; i < n-1; i++ {
		f.AddEdge(&ast.FlowEdge{From: fmt.Sprintf("N%d", i), To: fmt.Sprintf("N%d", i+1), Style: ast.EdgeSolid})
	}
	return f
}
