package render_test

import (
	"errors"
	"strings"
	"testing"

	diagram "github.com/iokdigital/go-mermaid"
	"github.com/iokdigital/go-mermaid/ast"
	"github.com/iokdigital/go-mermaid/render"
)

func newRenderer() *render.DefaultRenderer {
	return render.NewRenderer(diagram.NewRenderOptions())
}

func TestRendererMMD(t *testing.T) {
	f := ast.NewFlowchart("Test", ast.DirectionLR)
	f.MustAddNode(&ast.FlowNode{ID: "A"})
	r := newRenderer()
	data, err := r.RenderBytes(f, diagram.FormatMMD)
	if err != nil {
		t.Fatalf("RenderBytes mmd: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty mmd output")
	}
}

func TestRendererMarkdown(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionLR)
	r := newRenderer()
	md := r.RenderMarkdown(f)
	if md == "" {
		t.Error("expected non-empty markdown output")
	}
	// Must be wrapped in mermaid fence
	if md[:11] != "```mermaid\n" {
		t.Errorf("expected mermaid fence, got: %q", md[:min(20, len(md))])
	}
}

func TestRendererSVGProducesOutput(t *testing.T) {
	f := ast.NewFlowchart("Test", ast.DirectionLR)
	f.MustAddNode(&ast.FlowNode{ID: "A", Label: "Start", Shape: ast.ShapeRect})
	f.MustAddNode(&ast.FlowNode{ID: "B", Label: "End", Shape: ast.ShapeRect})
	f.AddEdge(&ast.FlowEdge{From: "A", To: "B", Style: ast.EdgeSolid})
	r := newRenderer()
	data, err := r.RenderBytes(f, diagram.FormatSVG)
	if err != nil {
		t.Fatalf("RenderBytes svg: %v", err)
	}
	out := string(data)
	if !strings.Contains(out, `xmlns="http://www.w3.org/2000/svg"`) {
		t.Error("expected SVG namespace in output")
	}
	if !strings.Contains(out, "<polyline") {
		t.Error("expected polyline edge in SVG output")
	}
}

func TestRendererPNGReturnsUnavailable(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionLR)
	r := newRenderer()
	_, err := r.RenderBytes(f, diagram.FormatPNG)
	if err == nil {
		t.Fatal("expected error for PNG (Phase 4)")
	}
	var ferr *diagram.FallbackFormatError
	if !errors.As(err, &ferr) {
		t.Fatalf("expected FallbackFormatError, got: %T", err)
	}
	if ferr.FallbackFormat() != diagram.FormatHTML {
		t.Errorf("expected HTML fallback, got: %s", ferr.FallbackFormat())
	}
}

func TestRendererHTMLProducesOutput(t *testing.T) {
	f := ast.NewFlowchart("Phase 2", ast.DirectionLR)
	r := newRenderer()
	data, err := r.RenderBytes(f, diagram.FormatHTML)
	if err != nil {
		t.Fatalf("RenderBytes html: %v", err)
	}
	got := string(data)
	if !strings.Contains(got, "<!DOCTYPE html>") {
		t.Error("expected HTML doctype")
	}
	if !strings.Contains(got, `<pre class="mermaid">`) {
		t.Error("expected mermaid pre block")
	}
	if !strings.Contains(got, "cdn.jsdelivr.net") {
		t.Error("expected CDN script tag")
	}
}

func TestRendererInvalidFormatError(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionLR)
	r := newRenderer()
	_, err := r.RenderBytes(f, diagram.OutputFormat("bogus"))
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
	if !errors.Is(err, diagram.ErrInvalidFormat) {
		t.Errorf("expected ErrInvalidFormat, got: %v", err)
	}
}

func TestContentType(t *testing.T) {
	r := newRenderer()
	cases := []struct {
		format diagram.OutputFormat
		want   string
	}{
		{diagram.FormatMMD, "text/x-mermaid"},
		{diagram.FormatSVG, "image/svg+xml"},
		{diagram.FormatPNG, "image/png"},
		{diagram.FormatHTML, "text/html; charset=utf-8"},
		{diagram.FormatPDF, "application/pdf"},
		{diagram.FormatJSON, "application/json"},
		{diagram.FormatDOT, "text/vnd.graphviz"},
		{diagram.FormatMarkdown, "text/markdown; charset=utf-8"},
	}
	for _, tc := range cases {
		got := r.ContentType(tc.format)
		if got != tc.want {
			t.Errorf("ContentType(%s) = %q, want %q", tc.format, got, tc.want)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
