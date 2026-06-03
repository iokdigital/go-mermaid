package dot_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	diagram "github.com/iokdigital/go-mermaid"
	"github.com/iokdigital/go-mermaid/ast"
	"github.com/iokdigital/go-mermaid/dot"
)

func encode(t *testing.T, d diagram.Diagram) string {
	t.Helper()
	var buf bytes.Buffer
	if err := dot.Encode(&buf, d); err != nil {
		t.Fatalf("dot.Encode: %v", err)
	}
	return buf.String()
}

func TestDOTFlowchartBasic(t *testing.T) {
	f := ast.NewFlowchart("My Graph", ast.DirectionLR)
	f.MustAddNode(&ast.FlowNode{ID: "A", Label: "Start", Shape: ast.ShapeRect})
	f.MustAddNode(&ast.FlowNode{ID: "B", Label: "End", Shape: ast.ShapeRect})
	f.AddEdge(&ast.FlowEdge{From: "A", To: "B", Label: "go", Style: ast.EdgeSolid})

	got := encode(t, f)
	if !strings.HasPrefix(got, "digraph {") {
		t.Errorf("expected digraph open, got: %q", got[:min(50, len(got))])
	}
	if !strings.Contains(got, "rankdir=LR") {
		t.Error("expected rankdir=LR")
	}
	if !strings.Contains(got, `label="My Graph"`) {
		t.Error("expected label attribute")
	}
	if !strings.Contains(got, "A ->") {
		t.Error("expected edge A ->")
	}
}

func TestDOTConfidenceFill(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "hi", Confidence: 0.95})
	f.MustAddNode(&ast.FlowNode{ID: "mid", Confidence: 0.75})
	f.MustAddNode(&ast.FlowNode{ID: "lo", Confidence: 0.50})

	got := encode(t, f)
	if !strings.Contains(got, "#90EE90") {
		t.Error("expected high-confidence fill #90EE90")
	}
	if !strings.Contains(got, "#FFD700") {
		t.Error("expected mid-confidence fill #FFD700")
	}
	if !strings.Contains(got, "#FFB6C1") {
		t.Error("expected low-confidence fill #FFB6C1")
	}
}

func TestDOTNonFlowchartReturnsError(t *testing.T) {
	s := ast.NewSequence("", false)
	var buf bytes.Buffer
	err := dot.Encode(&buf, s)
	if err == nil {
		t.Fatal("expected error for non-flowchart type")
	}
	var ferr *diagram.FallbackFormatError
	if !errors.As(err, &ferr) {
		t.Fatalf("expected FallbackFormatError, got: %T %v", err, err)
	}
	if ferr.FallbackFormat() != diagram.FormatMMD {
		t.Errorf("expected FormatMMD fallback, got: %s", ferr.FallbackFormat())
	}
	if !errors.Is(err, diagram.ErrRendererNotAvailable) {
		t.Error("expected ErrRendererNotAvailable in error chain")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
