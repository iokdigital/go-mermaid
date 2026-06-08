package negative_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	diagram "github.com/iokdigital/go-mermaid"
	"github.com/iokdigital/go-mermaid/ast"
	"github.com/iokdigital/go-mermaid/render"
)

func TestInvalidFormatError(t *testing.T) {
	r := render.NewRenderer(diagram.NewRenderOptions())
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A"})

	_, err := r.RenderBytes(f, diagram.OutputFormat("invalid-format"))
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
	if !errors.Is(err, diagram.ErrInvalidFormat) {
		t.Errorf("expected ErrInvalidFormat, got %v", err)
	}
}

func TestInvalidFormatRenderTo(t *testing.T) {
	r := render.NewRenderer(diagram.NewRenderOptions())
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A"})

	err := r.RenderTo(&errorWriter{}, f, diagram.OutputFormat("bogus"))
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
	if !errors.Is(err, diagram.ErrInvalidFormat) {
		t.Errorf("expected ErrInvalidFormat, got %v", err)
	}
}

type errorWriter struct{}

func (e *errorWriter) Write([]byte) (int, error) {
	return 0, errors.New("write error")
}

func TestDuplicateNodeIDError(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A"})

	_, err := f.AddNode(&ast.FlowNode{ID: "A"})
	if err == nil {
		t.Fatal("expected error for duplicate node ID")
	}
	if !errors.Is(err, diagram.ErrDuplicateNodeID) {
		t.Errorf("expected ErrDuplicateNodeID, got %v", err)
	}
}

func TestMustAddNodePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic")
		}
	}()

	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A"})
	f.MustAddNode(&ast.FlowNode{ID: "A"})
}

func TestEmptyFlowchartRender(t *testing.T) {
	r := render.NewRenderer(diagram.NewRenderOptions())
	f := ast.NewFlowchart("", ast.DirectionTB)

	data, err := r.RenderBytes(f, diagram.FormatSVG)
	if err != nil {
		t.Fatalf("empty flowchart should render: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty SVG output")
	}
}

func TestSelfLoopEdge(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A"})
	f.AddEdge(&ast.FlowEdge{From: "A", To: "A", Label: "self"})

	data, err := render.NewRenderer(diagram.NewRenderOptions()).RenderBytes(f, diagram.FormatSVG)
	if err != nil {
		t.Fatalf("self-loop should render: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty output for self-loop")
	}
}

func TestRenderToFileInvalidPath(t *testing.T) {
	r := render.NewRenderer(diagram.NewRenderOptions())
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A"})

	err := r.RenderToFile("/nonexistent/path/that/cannot/be/created/file.txt", f, diagram.FormatSVG)
	if err == nil {
		t.Fatal("expected error for invalid path")
	}
}

func TestRenderToFileValidPath(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "test.mmd")

	r := render.NewRenderer(diagram.NewRenderOptions())
	f := ast.NewFlowchart("Test", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A", Label: "Node A"})

	err := r.RenderToFile(path, f, diagram.FormatMMD)
	if err != nil {
		t.Fatalf("RenderToFile failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty file")
	}
}

func TestRenderMarkdownEmpty(t *testing.T) {
	r := render.NewRenderer(diagram.NewRenderOptions())
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A"})

	md := r.RenderMarkdown(f)
	if md == "" {
		t.Error("expected non-empty markdown")
	}
	if len(md) < 10 {
		t.Error("markdown too short")
	}
}

func TestContentTypeInvalidFormat(t *testing.T) {
	r := render.NewRenderer(diagram.NewRenderOptions())
	ct := r.ContentType(diagram.OutputFormat("invalid"))
	if ct != "application/octet-stream" {
		t.Errorf("expected application/octet-stream, got %q", ct)
	}
}

func TestFallbackFormatErrorUnwrap(t *testing.T) {
	ferr := &diagram.FallbackFormatError{
		Err:      diagram.ErrPNGSizeLimitExceeded,
		Fallback: diagram.FormatHTML,
	}

	var target *diagram.FallbackFormatError
	if !errors.As(ferr, &target) {
		t.Error("should be able to errors.As FallbackFormatError")
	}
}

func TestSequenceDiagramEmpty(t *testing.T) {
	s := ast.NewSequence("", false)
	data, err := render.NewRenderer(diagram.NewRenderOptions()).RenderBytes(s, diagram.FormatMMD)
	if err != nil {
		t.Fatalf("empty sequence should render: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty output")
	}
}

func TestStateDiagramEmpty(t *testing.T) {
	s := ast.NewState("")
	data, err := render.NewRenderer(diagram.NewRenderOptions()).RenderBytes(s, diagram.FormatMMD)
	if err != nil {
		t.Fatalf("empty state should render: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty output")
	}
}

func TestERDiagramEmpty(t *testing.T) {
	e := ast.NewER("")
	data, err := render.NewRenderer(diagram.NewRenderOptions()).RenderBytes(e, diagram.FormatMMD)
	if err != nil {
		t.Fatalf("empty ER should render: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty output")
	}
}

func TestClassDiagramEmpty(t *testing.T) {
	c := ast.NewClass("")
	data, err := render.NewRenderer(diagram.NewRenderOptions()).RenderBytes(c, diagram.FormatMMD)
	if err != nil {
		t.Fatalf("empty class should render: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty output")
	}
}

func TestFlowchartVeryLongLabel(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionTB)
	longLabel := ""
	for i := 0; i < 1000; i++ {
		longLabel += "x"
	}
	f.MustAddNode(&ast.FlowNode{ID: "A", Label: longLabel})

	data, err := render.NewRenderer(diagram.NewRenderOptions()).RenderBytes(f, diagram.FormatSVG)
	if err != nil {
		t.Fatalf("long label should render: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty output")
	}
}

func TestFlowchartSpecialCharactersInID(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "node-with-special!@#chars"})

	data, err := render.NewRenderer(diagram.NewRenderOptions()).RenderBytes(f, diagram.FormatMMD)
	if err != nil {
		t.Fatalf("special chars in ID should render: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty output")
	}
}

func TestPNGUnsupportedDiagramType(t *testing.T) {
	r := render.NewRenderer(diagram.NewRenderOptions())

	_, err := r.RenderBytes(&fakeDiagram{}, diagram.FormatPNG)
	if err == nil {
		t.Fatal("expected error for PNG on unsupported diagram type")
	}

	var ferr *diagram.FallbackFormatError
	if !errors.As(err, &ferr) {
		t.Fatalf("expected FallbackFormatError, got %T: %v", err, err)
	}
	if ferr.FallbackFormat() != diagram.FormatHTML {
		t.Errorf("expected fallback HTML, got %v", ferr.FallbackFormat())
	}
}

type fakeDiagram struct{}

func (f *fakeDiagram) Type() diagram.DiagramType { return diagram.TypePie }
func (f *fakeDiagram) Title() string             { return "" }

func TestAllEdgeStylesRender(t *testing.T) {
	r := render.NewRenderer(diagram.NewRenderOptions())
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A"})
	f.MustAddNode(&ast.FlowNode{ID: "B"})

	styles := []ast.EdgeStyle{
		ast.EdgeSolid, ast.EdgeDotted, ast.EdgeThick, ast.EdgeInvisible, ast.EdgeNoArrow,
	}

	for _, style := range styles {
		f.AddEdge(&ast.FlowEdge{From: "A", To: "B", Style: style})
	}

	data, err := r.RenderBytes(f, diagram.FormatSVG)
	if err != nil {
		t.Fatalf("edge styles should render: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty output")
	}
}

func TestAllShapesRender(t *testing.T) {
	r := render.NewRenderer(diagram.NewRenderOptions())
	f := ast.NewFlowchart("", ast.DirectionTB)

	shapes := []ast.NodeShape{
		ast.ShapeRect, ast.ShapeDiamond, ast.ShapeCircle, ast.ShapeRoundedRect,
		ast.ShapeParallelogram, ast.ShapeHexagon, ast.ShapeStadium, ast.ShapeAsymmetric,
	}

	for i, shape := range shapes {
		f.MustAddNode(&ast.FlowNode{ID: string(shape), Label: string(shape), Shape: shape})
		if i > 0 {
			f.AddEdge(&ast.FlowEdge{From: string(shapes[i-1]), To: string(shape)})
		}
	}

	data, err := r.RenderBytes(f, diagram.FormatSVG)
	if err != nil {
		t.Fatalf("shapes should render: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty output")
	}
}

func TestSubgraphRender(t *testing.T) {
	f := ast.NewFlowchart("Subgraph Test", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A"})
	f.MustAddNode(&ast.FlowNode{ID: "B"})
	f.AddSubgraph(&ast.Subgraph{ID: "sg1", Label: "Group", Nodes: []string{"A", "B"}})
	f.AddEdge(&ast.FlowEdge{From: "A", To: "B"})

	data, err := render.NewRenderer(diagram.NewRenderOptions()).RenderBytes(f, diagram.FormatSVG)
	if err != nil {
		t.Fatalf("subgraph should render: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty output")
	}
}

func TestMultipleSubgraphs(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A"})
	f.MustAddNode(&ast.FlowNode{ID: "B"})
	f.MustAddNode(&ast.FlowNode{ID: "C"})
	f.AddSubgraph(&ast.Subgraph{ID: "sg1", Label: "Group1", Nodes: []string{"A", "B"}})
	f.AddSubgraph(&ast.Subgraph{ID: "sg2", Label: "Group2", Nodes: []string{"B", "C"}})
	f.AddEdge(&ast.FlowEdge{From: "A", To: "C"})

	data, err := render.NewRenderer(diagram.NewRenderOptions()).RenderBytes(f, diagram.FormatSVG)
	if err != nil {
		t.Fatalf("multiple subgraphs should render: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty output")
	}
}