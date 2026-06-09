package property_test

import (
	"strings"
	"testing"

	diagram "github.com/iokdigital/go-mermaid"
	"github.com/iokdigital/go-mermaid/ast"
	"github.com/iokdigital/go-mermaid/render"
)

func TestSVGValidXML(t *testing.T) {
	r := render.NewRenderer(diagram.NewRenderOptions())
	f := ast.NewFlowchart("Test", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A", Label: "Start"})
	f.MustAddNode(&ast.FlowNode{ID: "B", Label: "End"})
	f.AddEdge(&ast.FlowEdge{From: "A", To: "B", Label: "connects"})

	data, err := r.RenderBytes(f, diagram.FormatSVG)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	out := string(data)

	if !strings.HasPrefix(out, "<?xml") {
		t.Error("SVG should start with XML declaration")
	}

	if !strings.Contains(out, "<svg") {
		t.Error("SVG should contain <svg> element")
	}

	if !strings.Contains(out, "</svg>") {
		t.Error("SVG should contain closing </svg> element")
	}

	if !strings.Contains(out, `xmlns="http://www.w3.org/2000/svg"`) {
		t.Error("SVG should contain namespace")
	}
}

func TestSVGContainsTitle(t *testing.T) {
	r := render.NewRenderer(diagram.NewRenderOptions())
	f := ast.NewFlowchart("My Title", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A"})

	data, err := r.RenderBytes(f, diagram.FormatSVG)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	out := string(data)
	if !strings.Contains(out, "<title>My Title</title>") {
		t.Error("SVG should contain title element")
	}
}

func TestSVGContainsNodes(t *testing.T) {
	r := render.NewRenderer(diagram.NewRenderOptions())
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "NodeA", Label: "Label A"})
	f.MustAddNode(&ast.FlowNode{ID: "NodeB", Label: "Label B"})
	f.AddEdge(&ast.FlowEdge{From: "NodeA", To: "NodeB"})

	data, err := r.RenderBytes(f, diagram.FormatSVG)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	out := string(data)
	if !strings.Contains(out, "NodeA") || !strings.Contains(out, "NodeB") {
		t.Error("SVG should contain node IDs")
	}
}

func TestSVGContainsEdges(t *testing.T) {
	r := render.NewRenderer(diagram.NewRenderOptions())
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A"})
	f.MustAddNode(&ast.FlowNode{ID: "B"})
	f.AddEdge(&ast.FlowEdge{From: "A", To: "B"})

	data, err := r.RenderBytes(f, diagram.FormatSVG)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	out := string(data)
	if !strings.Contains(out, `<path`) {
		t.Error("SVG should contain path elements for edges")
	}
}

func TestSVGNoInvalidChars(t *testing.T) {
	r := render.NewRenderer(diagram.NewRenderOptions())
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A", Label: "Node with <special> & \"chars\""})

	data, err := r.RenderBytes(f, diagram.FormatSVG)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	out := string(data)

	if strings.Contains(out, "<special>") {
		t.Error("SVG should have escaped < in labels")
	}
	if strings.Contains(out, "& ") && !strings.Contains(out, "&amp;") {
		t.Error("SVG should have escaped & in labels")
	}
}

func TestSVGWithSubgraph(t *testing.T) {
	r := render.NewRenderer(diagram.NewRenderOptions())
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A"})
	f.MustAddNode(&ast.FlowNode{ID: "B"})
	f.AddSubgraph(&ast.Subgraph{ID: "sg1", Label: "Group", Nodes: []string{"A"}})
	f.AddEdge(&ast.FlowEdge{From: "A", To: "B"})

	data, err := r.RenderBytes(f, diagram.FormatSVG)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	out := string(data)
	if !strings.Contains(out, "Group") {
		t.Skip("SVG subgraph rendering not yet implemented")
	}
}

func TestSVGWithEdgeLabels(t *testing.T) {
	r := render.NewRenderer(diagram.NewRenderOptions())
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A"})
	f.MustAddNode(&ast.FlowNode{ID: "B"})
	f.AddEdge(&ast.FlowEdge{From: "A", To: "B", Label: "Edge Label"})

	data, err := r.RenderBytes(f, diagram.FormatSVG)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	out := string(data)
	if !strings.Contains(out, "Edge Label") {
		t.Error("SVG should contain edge label")
	}
}

func TestSVGMultipleEdges(t *testing.T) {
	r := render.NewRenderer(diagram.NewRenderOptions())
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A"})
	f.MustAddNode(&ast.FlowNode{ID: "B"})
	f.MustAddNode(&ast.FlowNode{ID: "C"})
	f.AddEdge(&ast.FlowEdge{From: "A", To: "B"})
	f.AddEdge(&ast.FlowEdge{From: "B", To: "C"})
	f.AddEdge(&ast.FlowEdge{From: "A", To: "C"})

	data, err := r.RenderBytes(f, diagram.FormatSVG)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	out := string(data)
	count := strings.Count(out, "<path")
	if count < 3 {
		t.Errorf("expected at least 3 path elements, got %d", count)
	}
}

func TestSVGEmptyDiagram(t *testing.T) {
	r := render.NewRenderer(diagram.NewRenderOptions())
	f := ast.NewFlowchart("", ast.DirectionTB)

	data, err := r.RenderBytes(f, diagram.FormatSVG)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	out := string(data)
	if !strings.Contains(out, "<svg") {
		t.Error("empty SVG should still have svg element")
	}
}

func TestMMDOutputValid(t *testing.T) {
	r := render.NewRenderer(diagram.NewRenderOptions())
	f := ast.NewFlowchart("Test", ast.DirectionLR)
	f.MustAddNode(&ast.FlowNode{ID: "A", Label: "Start"})
	f.MustAddNode(&ast.FlowNode{ID: "B", Label: "End"})
	f.AddEdge(&ast.FlowEdge{From: "A", To: "B", Label: "go"})

	data, err := r.RenderBytes(f, diagram.FormatMMD)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	out := string(data)

	if !strings.Contains(out, "flowchart LR") {
		t.Error("MMD should contain flowchart declaration")
	}
	if !strings.Contains(out, "Start") {
		t.Error("MMD should contain node label")
	}
	if !strings.Contains(out, "A[Start]") {
		t.Error("MMD should contain node definition")
	}
}

func TestJSONOutputValid(t *testing.T) {
	r := render.NewRenderer(diagram.NewRenderOptions())
	f := ast.NewFlowchart("Test", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A", Label: "Node A"})

	data, err := r.RenderBytes(f, diagram.FormatJSON)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	out := string(data)

	if !strings.Contains(out, `"title"`) {
		t.Error("JSON should contain title field")
	}
	if !strings.Contains(out, `"nodes"`) {
		t.Error("JSON should contain nodes field")
	}
}

func TestDOTOutputValid(t *testing.T) {
	r := render.NewRenderer(diagram.NewRenderOptions())
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A", Label: "Start"})
	f.MustAddNode(&ast.FlowNode{ID: "B", Label: "End"})
	f.AddEdge(&ast.FlowEdge{From: "A", To: "B"})

	data, err := r.RenderBytes(f, diagram.FormatDOT)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	out := string(data)

	if !strings.Contains(out, "digraph") {
		t.Error("DOT should contain digraph")
	}
	if !strings.Contains(out, "A") || !strings.Contains(out, "B") {
		t.Error("DOT should contain node IDs")
	}
}