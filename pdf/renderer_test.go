package pdf_test

import (
	"bytes"
	"testing"

	diagram "github.com/iokdigital/go-mermaid"
	"github.com/iokdigital/go-mermaid/ast"
	diapdf "github.com/iokdigital/go-mermaid/pdf"
	"github.com/iokdigital/go-mermaid/svg"
)

func svgOf(t *testing.T, d diagram.Diagram) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := svg.Encode(&buf, d, diagram.NewRenderOptions()); err != nil {
		t.Fatalf("svg.Encode: %v", err)
	}
	return buf.Bytes()
}

func TestEncode_ProducesPDF(t *testing.T) {
	f := ast.NewFlowchart("My Diagram", ast.DirectionLR)
	f.MustAddNode(&ast.FlowNode{ID: "A", Shape: ast.ShapeRect})
	f.MustAddNode(&ast.FlowNode{ID: "B", Shape: ast.ShapeRect})
	f.AddEdge(&ast.FlowEdge{From: "A", To: "B", Style: ast.EdgeSolid})
	svgData := svgOf(t, f)

	var buf bytes.Buffer
	if err := diapdf.Encode(svgData, &buf, "My Diagram", diagram.NewRenderOptions()); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	data := buf.Bytes()
	if len(data) == 0 {
		t.Fatal("expected non-empty PDF output")
	}
	// PDF magic bytes: %PDF-
	if len(data) < 5 || string(data[:5]) != "%PDF-" {
		t.Errorf("output does not start with %%PDF-, got: %q", data[:min(8, len(data))])
	}
}

func TestEncode_EmptyTitleOK(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A", Shape: ast.ShapeRect})
	svgData := svgOf(t, f)

	var buf bytes.Buffer
	if err := diapdf.Encode(svgData, &buf, "", diagram.NewRenderOptions()); err != nil {
		t.Errorf("expected no error for empty title, got: %v", err)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
