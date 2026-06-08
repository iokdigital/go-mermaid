package ast_test

import (
	"errors"
	"testing"

	diagram "github.com/iokdigital/go-mermaid"
	"github.com/iokdigital/go-mermaid/ast"
)

func TestNewFlowchart(t *testing.T) {
	f := ast.NewFlowchart("Test Diagram", ast.DirectionTB)
	if f.Title() != "Test Diagram" {
		t.Errorf("expected title %q, got %q", "Test Diagram", f.Title())
	}
	if f.Direction() != ast.DirectionTB {
		t.Errorf("expected direction %q, got %q", ast.DirectionTB, f.Direction())
	}
	if f.Type() != diagram.TypeFlowchart {
		t.Errorf("expected type %q, got %q", diagram.TypeFlowchart, f.Type())
	}
}

func TestNewFlowchartDirections(t *testing.T) {
	dirs := []ast.Direction{ast.DirectionTB, ast.DirectionLR, ast.DirectionBT, ast.DirectionRL}
	for _, d := range dirs {
		f := ast.NewFlowchart("", d)
		if f.Direction() != d {
			t.Errorf("expected direction %q, got %q", d, f.Direction())
		}
	}
}

func TestFlowchartAddNode(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A", Label: "Node A"})

	if len(f.Nodes()) != 1 {
		t.Fatalf("expected 1 node, got %d", len(f.Nodes()))
	}
	if f.Nodes()[0].ID != "A" {
		t.Errorf("expected node ID %q, got %q", "A", f.Nodes()[0].ID)
	}
}

func TestFlowchartAddNodeDuplicate(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A", Label: "Node A"})

	_, err := f.AddNode(&ast.FlowNode{ID: "A", Label: "Duplicate"})
	if err == nil {
		t.Fatal("expected error for duplicate node ID")
	}
	if !errors.Is(err, diagram.ErrDuplicateNodeID) {
		t.Errorf("expected ErrDuplicateNodeID, got %v", err)
	}
}

func TestFlowchartMustAddNodePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for duplicate node ID")
		}
	}()

	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A"})
	f.MustAddNode(&ast.FlowNode{ID: "A"})
}

func TestFlowchartAddEdge(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A"})
	f.MustAddNode(&ast.FlowNode{ID: "B"})
	f.AddEdge(&ast.FlowEdge{From: "A", To: "B", Label: "connects"})

	if len(f.Edges()) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(f.Edges()))
	}
	if f.Edges()[0].From != "A" {
		t.Errorf("expected edge from %q, got %q", "A", f.Edges()[0].From)
	}
	if f.Edges()[0].Label != "connects" {
		t.Errorf("expected label %q, got %q", "connects", f.Edges()[0].Label)
	}
}

func TestFlowchartAddEdgeSelfLoop(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A"})
	f.AddEdge(&ast.FlowEdge{From: "A", To: "A"})

	if len(f.Edges()) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(f.Edges()))
	}
}

func TestFlowchartAddSubgraph(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A"})
	f.MustAddNode(&ast.FlowNode{ID: "B"})
	f.AddSubgraph(&ast.Subgraph{ID: "sg1", Label: "Group", Nodes: []string{"A", "B"}})

	if len(f.Subgraphs()) != 1 {
		t.Fatalf("expected 1 subgraph, got %d", len(f.Subgraphs()))
	}
	if f.Subgraphs()[0].Label != "Group" {
		t.Errorf("expected label %q, got %q", "Group", f.Subgraphs()[0].Label)
	}
}

func TestFlowchartNodeShapes(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionTB)

	shapes := []ast.NodeShape{
		ast.ShapeRect, ast.ShapeDiamond, ast.ShapeCircle, ast.ShapeRoundedRect,
		ast.ShapeParallelogram, ast.ShapeHexagon, ast.ShapeStadium, ast.ShapeAsymmetric,
	}

	for _, shape := range shapes {
		node := &ast.FlowNode{ID: string(shape), Shape: shape}
		f.MustAddNode(node)
	}

	if len(f.Nodes()) != len(shapes) {
		t.Fatalf("expected %d nodes, got %d", len(shapes), len(f.Nodes()))
	}
}

func TestFlowchartEdgeStyles(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A"})
	f.MustAddNode(&ast.FlowNode{ID: "B"})

	styles := []ast.EdgeStyle{
		ast.EdgeSolid, ast.EdgeDotted, ast.EdgeThick, ast.EdgeInvisible, ast.EdgeNoArrow,
	}

	for _, style := range styles {
		f.AddEdge(&ast.FlowEdge{From: "A", To: "B", Style: style})
	}

	if len(f.Edges()) != len(styles) {
		t.Fatalf("expected %d edges, got %d", len(styles), len(f.Edges()))
	}
}

func TestFlowchartConfidence(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionTB)
	node := &ast.FlowNode{ID: "A", Confidence: 0.75}
	f.MustAddNode(node)

	if f.Nodes()[0].Confidence != 0.75 {
		t.Errorf("expected confidence 0.75, got %f", f.Nodes()[0].Confidence)
	}
}

func TestFlowchartURL(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionTB)
	node := &ast.FlowNode{ID: "A", URL: "https://example.com"}
	f.MustAddNode(node)

	if f.Nodes()[0].URL != "https://example.com" {
		t.Errorf("expected URL %q, got %q", "https://example.com", f.Nodes()[0].URL)
	}
}

func TestFlowchartMetadata(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionTB)
	node := &ast.FlowNode{
		ID:       "A",
		Metadata: map[string]string{"key": "value", "foo": "bar"},
	}
	f.MustAddNode(node)

	meta := f.Nodes()[0].Metadata
	if meta["key"] != "value" {
		t.Errorf("expected metadata key=value, got %v", meta)
	}
}

func TestNodeShapeSyntax(t *testing.T) {
	cases := []struct {
		shape    ast.NodeShape
		wantOpen string
		wantClose string
	}{
		{ast.ShapeRect, "[", "]"},
		{ast.ShapeDiamond, "{", "}"},
		{ast.ShapeCircle, "((", "))"},
		{ast.ShapeRoundedRect, "(", ")"},
		{ast.ShapeParallelogram, "[/", "/]"},
		{ast.ShapeHexagon, "{{", "}}"},
		{ast.ShapeStadium, "([", "])"},
		{ast.ShapeAsymmetric, ">", "]"},
		{ast.NodeShape("unknown"), "[", "]"},
	}

	for _, tc := range cases {
		open, close := ast.NodeShapeSyntax(tc.shape)
		if open != tc.wantOpen || close != tc.wantClose {
			t.Errorf("NodeShapeSyntax(%v) = %q,%q want %q,%q",
				tc.shape, open, close, tc.wantOpen, tc.wantClose)
		}
	}
}

func TestSanitizeID(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"valid-id", "valid-id"},
		{"ValidID123", "ValidID123"},
		{"node-1", "node-1"},
		{"node with spaces", "node_with_spaces"},
		{"node@special", "node_special"},
		{"node#hash", "node_hash"},
		{"node$var", "node_var"},
		{"node!punct", "node_punct"},
		{"node[brackets]", "node_brackets_"},
		{"", ""},
		{"a b c d e", "a_b_c_d_e"},
	}

	for _, tc := range cases {
		got := ast.SanitizeID(tc.input)
		if got != tc.want {
			t.Errorf("SanitizeID(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestFlowchartEmpty(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionTB)
	if len(f.Nodes()) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(f.Nodes()))
	}
	if len(f.Edges()) != 0 {
		t.Errorf("expected 0 edges, got %d", len(f.Edges()))
	}
	if len(f.Subgraphs()) != 0 {
		t.Errorf("expected 0 subgraphs, got %d", len(f.Subgraphs()))
	}
}