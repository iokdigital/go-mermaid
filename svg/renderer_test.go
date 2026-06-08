package svg_test

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	diagram "github.com/iokdigital/go-mermaid"
	"github.com/iokdigital/go-mermaid/ast"
	"github.com/iokdigital/go-mermaid/svg"
)

func opts() diagram.RenderOptions { return diagram.NewRenderOptions() }

func encode(t *testing.T, d diagram.Diagram) string {
	t.Helper()
	var buf bytes.Buffer
	if err := svg.Encode(&buf, d, opts()); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	return buf.String()
}

func TestEncode_SVGDoctype(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionTB)
	out := encode(t, f)
	if !strings.Contains(out, `<?xml version="1.0" encoding="UTF-8"?>`) {
		t.Error("missing XML declaration")
	}
	if !strings.Contains(out, `xmlns="http://www.w3.org/2000/svg"`) {
		t.Error("missing SVG namespace")
	}
}

func TestEncode_ViewBoxPresent(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionTB)
	out := encode(t, f)
	if !strings.Contains(out, `viewBox="0 0`) {
		t.Error("missing viewBox attribute")
	}
}

func TestEncode_TitleElement(t *testing.T) {
	f := ast.NewFlowchart("My Diagram", ast.DirectionTB)
	out := encode(t, f)
	if !strings.Contains(out, `<title>My Diagram</title>`) {
		t.Errorf("title element missing in:\n%s", out)
	}
}

func TestEncode_TitleXSSEscaped(t *testing.T) {
	f := ast.NewFlowchart(`<script>alert('xss')</script>`, ast.DirectionTB)
	out := encode(t, f)
	if strings.Contains(out, "<script>") {
		t.Error("XSS not escaped in title")
	}
	if !strings.Contains(out, "&lt;script&gt;") {
		t.Error("expected escaped title text")
	}
}

func TestEncode_NodesPresent(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A", Label: "Start", Shape: ast.ShapeRect})
	f.MustAddNode(&ast.FlowNode{ID: "B", Label: "End", Shape: ast.ShapeRect})
	out := encode(t, f)
	if !strings.Contains(out, `id="node-A"`) {
		t.Error("node A group missing")
	}
	if !strings.Contains(out, `id="node-B"`) {
		t.Error("node B group missing")
	}
}

func TestEncode_EdgePath(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A", Shape: ast.ShapeRect})
	f.MustAddNode(&ast.FlowNode{ID: "B", Shape: ast.ShapeRect})
	f.AddEdge(&ast.FlowEdge{From: "A", To: "B", Style: ast.EdgeSolid})
	out := encode(t, f)
	if !strings.Contains(out, `<path d="M`) {
		t.Error("expected path element for edge")
	}
}

func TestEncode_ArrowheadPresent(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A", Shape: ast.ShapeRect})
	f.MustAddNode(&ast.FlowNode{ID: "B", Shape: ast.ShapeRect})
	f.AddEdge(&ast.FlowEdge{From: "A", To: "B", Style: ast.EdgeSolid})
	out := encode(t, f)
	if !strings.Contains(out, "<polygon") {
		t.Error("expected polygon arrowhead element")
	}
}

func TestEncode_NoArrowEdge(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A", Shape: ast.ShapeRect})
	f.MustAddNode(&ast.FlowNode{ID: "B", Shape: ast.ShapeRect})
	f.AddEdge(&ast.FlowEdge{From: "A", To: "B", Style: ast.EdgeNoArrow})
	out := encode(t, f)
	if strings.Contains(out, "<polygon") {
		t.Error("EdgeNoArrow should not produce an arrowhead polygon")
	}
}

func TestEncode_EdgeLabel(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionLR)
	f.MustAddNode(&ast.FlowNode{ID: "A", Shape: ast.ShapeRect})
	f.MustAddNode(&ast.FlowNode{ID: "B", Shape: ast.ShapeRect})
	f.AddEdge(&ast.FlowEdge{From: "A", To: "B", Style: ast.EdgeSolid, Label: "File→fileName"})
	out := encode(t, f)
	if !strings.Contains(out, "File→fileName") {
		t.Error("edge label not found in output")
	}
}

func TestEncode_EdgeLabel_AtMidpoint(t *testing.T) {
	// LR layout: A at rank 0, B at rank 1 → exit x of A < midpoint x < entry x of B.
	f := ast.NewFlowchart("", ast.DirectionLR)
	f.MustAddNode(&ast.FlowNode{ID: "A", Shape: ast.ShapeRect})
	f.MustAddNode(&ast.FlowNode{ID: "B", Shape: ast.ShapeRect})
	f.AddEdge(&ast.FlowEdge{From: "A", To: "B", Style: ast.EdgeSolid, Label: "mid"})

	var buf bytes.Buffer
	if err := svg.Encode(&buf, f, opts()); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	out := buf.String()

	// Extract label x from: <text x="NNN.N" ...>mid</text>
	idx := strings.Index(out, ">mid<")
	if idx < 0 {
		t.Fatal("label 'mid' not found in SVG output")
	}
	// Walk back to find the opening <text tag
	start := strings.LastIndex(out[:idx], "<text ")
	if start < 0 {
		t.Fatal("could not find <text element for label")
	}
	tag := out[start:idx]
	var labelX float64
	if _, err := fmt.Sscanf(extractAttr(tag, "x"), "%f", &labelX); err != nil {
		t.Fatalf("could not parse label x from tag %q: %v", tag, err)
	}

	// In LR layout the edge path runs left→right. Label must be strictly between
	// the exit x of A (its right border) and the entry x of B (its left border).
	// Both nodes are nodeWidth=120 wide. A is at rank 0, B at rank 1.
	// We don't hard-code layout constants — just assert the label is not at either end.
	if !strings.Contains(out, `<path d="M`) {
		t.Fatal("no edge path found; cannot validate midpoint")
	}
	// Extract path d to get the edge x range: d="Mx0,y0 Lx1,y1"
	pIdx := strings.Index(out, `<path d="M`)
	if pIdx < 0 {
		t.Fatal("could not find edge path element")
	}
	pTag := out[pIdx:]
	pEnd := strings.Index(pTag, "/>")
	pTag = pTag[:pEnd]
	d := extractAttr(pTag, "d")
	// d format: "Mx0,y0 Lx1,y1" — parse first M and last L coordinates
	var x0, y0, x1, y1 float64
	if _, err := fmt.Sscanf(d, "M%f,%f L%f,%f", &x0, &y0, &x1, &y1); err != nil {
		t.Fatalf("failed to parse path data %q: %v", d, err)
	}
	_ = y0
	_ = y1

	// The label x must be strictly between x0 (exit) and x1+arrowLen (entry, restored).
	entryX := x1 + 8 // arrowLen=8 was subtracted during shortening
	if labelX <= x0 || labelX >= entryX {
		t.Errorf("label x=%.1f is not between exit x=%.1f and entry x=%.1f", labelX, x0, entryX)
	}
}

// extractAttr extracts the value of an SVG attribute from a tag string.
func extractAttr(tag, attr string) string {
	needle := attr + `="`
	idx := strings.Index(tag, needle)
	if idx < 0 {
		return ""
	}
	rest := tag[idx+len(needle):]
	end := strings.IndexByte(rest, '"')
	if end < 0 {
		return rest
	}
	return rest[:end]
}

func TestEncode_ConfidenceHighFillColor(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A", Shape: ast.ShapeRect, Confidence: 0.95})
	out := encode(t, f)
	if !strings.Contains(out, "#90EE90") {
		t.Errorf("expected high-confidence fill #90EE90 in:\n%s", out)
	}
}

func TestEncode_ConfidenceMediumFillColor(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A", Shape: ast.ShapeRect, Confidence: 0.75})
	out := encode(t, f)
	if !strings.Contains(out, "#FFD700") {
		t.Error("expected medium-confidence fill #FFD700")
	}
}

func TestEncode_ConfidenceLowFillColor(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A", Shape: ast.ShapeRect, Confidence: 0.50})
	out := encode(t, f)
	if !strings.Contains(out, "#FFB6C1") {
		t.Error("expected low-confidence fill #FFB6C1")
	}
}

func TestEncode_NeutralFillColor(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A", Shape: ast.ShapeRect}) // confidence 0.0
	out := encode(t, f)
	if !strings.Contains(out, "#f8fafc") {
		t.Error("expected neutral fill #f8fafc for zero confidence")
	}
}

func TestEncode_NodeURL_WrappedInAnchor(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A", Shape: ast.ShapeRect, URL: "https://example.com"})
	out := encode(t, f)
	if !strings.Contains(out, `href="https://example.com"`) {
		t.Error("expected href attribute for node URL")
	}
	if !strings.Contains(out, `rel="noopener noreferrer"`) {
		t.Error("expected rel=noopener noreferrer on anchor")
	}
}

func TestEncode_NodeURL_JavaScriptRejected(t *testing.T) {
	cases := []string{
		"javascript:alert(1)",
		"JAVASCRIPT:alert(1)",
		"  javascript:alert(1)",
		"data:text/html,<h1>xss</h1>",
		"vbscript:msgbox(1)",
	}
	for _, u := range cases {
		f := ast.NewFlowchart("", ast.DirectionTB)
		f.MustAddNode(&ast.FlowNode{ID: "A", Shape: ast.ShapeRect, URL: u})
		out := encode(t, f)
		if strings.Contains(out, "<a ") {
			t.Errorf("URL %q: expected no anchor element for unsafe URL, got one", u)
		}
		if strings.Contains(out, "javascript") || strings.Contains(out, "vbscript") {
			t.Errorf("URL %q: unsafe scheme leaked into SVG output", u)
		}
	}
}

func TestEncode_SelfLoop(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A", Shape: ast.ShapeRect})
	f.AddEdge(&ast.FlowEdge{From: "A", To: "A", Style: ast.EdgeSolid})
	out := encode(t, f)
	if !strings.Contains(out, "<path") {
		t.Error("expected <path> arc for self-loop")
	}
	if !strings.Contains(out, "⟳") {
		t.Error("expected ⟳ glyph on self-loop")
	}
}

func TestEncode_DottedEdge_StrokeDash(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A", Shape: ast.ShapeRect})
	f.MustAddNode(&ast.FlowNode{ID: "B", Shape: ast.ShapeRect})
	f.AddEdge(&ast.FlowEdge{From: "A", To: "B", Style: ast.EdgeDotted})
	out := encode(t, f)
	if !strings.Contains(out, "stroke-dasharray") {
		t.Error("dotted edge should produce stroke-dasharray")
	}
}

func TestEncode_InvisibleEdge_NotRendered(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A", Shape: ast.ShapeRect})
	f.MustAddNode(&ast.FlowNode{ID: "B", Shape: ast.ShapeRect})
	f.AddEdge(&ast.FlowEdge{From: "A", To: "B", Style: ast.EdgeInvisible})
	out := encode(t, f)
	if strings.Contains(out, `<path d="M`) || strings.Contains(out, "<polygon") {
		t.Error("invisible edge should not render any line or arrowhead")
	}
}

func TestEncode_LRDirection_HasContent(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionLR)
	f.MustAddNode(&ast.FlowNode{ID: "A", Shape: ast.ShapeRect})
	f.MustAddNode(&ast.FlowNode{ID: "B", Shape: ast.ShapeRect})
	f.AddEdge(&ast.FlowEdge{From: "A", To: "B", Style: ast.EdgeSolid})
	out := encode(t, f)
	if !strings.Contains(out, `id="node-A"`) || !strings.Contains(out, `id="node-B"`) {
		t.Error("LR diagram missing nodes")
	}
}

func TestEncode_NonFlowchart_ReturnsFallback(t *testing.T) {
	// Pass a non-flowchart diagram type to verify FallbackFormatError is returned.
	var buf bytes.Buffer
	err := svg.Encode(&buf, &fakeNonFlowchart{}, opts())
	if err == nil {
		t.Fatal("expected error for non-flowchart diagram")
	}
	var fbe *diagram.FallbackFormatError
	if !errors.As(err, &fbe) {
		t.Errorf("expected FallbackFormatError, got %T: %v", err, err)
	}
	if fbe.Fallback != diagram.FormatHTML {
		t.Errorf("expected fallback FormatHTML, got %s", fbe.Fallback)
	}
}

// fakeNonFlowchart satisfies diagram.Diagram but is not *ast.FlowchartDiagram.
type fakeNonFlowchart struct{}

func (f *fakeNonFlowchart) Type() diagram.DiagramType { return diagram.TypeSequence }
func (f *fakeNonFlowchart) Title() string             { return "" }
