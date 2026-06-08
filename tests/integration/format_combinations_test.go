package integration_test

import (
	"strings"
	"testing"

	diagram "github.com/iokdigital/go-mermaid"
	"github.com/iokdigital/go-mermaid/ast"
	"github.com/iokdigital/go-mermaid/render"
)

func TestFlowchartAllFormats(t *testing.T) {
	r := render.NewRenderer(diagram.NewRenderOptions())
	f := ast.NewFlowchart("Test", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A", Label: "Start"})
	f.MustAddNode(&ast.FlowNode{ID: "B", Label: "End"})
	f.AddEdge(&ast.FlowEdge{From: "A", To: "B"})

	formats := []diagram.OutputFormat{
		diagram.FormatMMD, diagram.FormatSVG, diagram.FormatHTML,
		diagram.FormatPDF, diagram.FormatJSON, diagram.FormatDOT,
		diagram.FormatMarkdown,
	}

	for _, fmt := range formats {
		data, err := r.RenderBytes(f, fmt)
		if err != nil {
			t.Errorf("Flowchart + %v: %v", fmt, err)
			continue
		}
		if len(data) == 0 {
			t.Errorf("Flowchart + %v: empty output", fmt)
		}
	}
}

func TestSequenceAllFormats(t *testing.T) {
	r := render.NewRenderer(diagram.NewRenderOptions())
	s := ast.NewSequence("Test", true)
	s.AddParticipant(ast.Participant{Alias: "A", Label: "Alice"})
	s.AddParticipant(ast.Participant{Alias: "B", Label: "Bob"})
	s.AddMessage(ast.SeqMessage{From: "A", To: "B", Text: "Hello"})

	formats := []diagram.OutputFormat{
		diagram.FormatMMD, diagram.FormatSVG, diagram.FormatHTML,
		diagram.FormatJSON, diagram.FormatMarkdown,
	}

	for _, fmt := range formats {
		data, err := r.RenderBytes(s, fmt)
		if err != nil {
			t.Errorf("Sequence + %v: %v", fmt, err)
			continue
		}
		if len(data) == 0 {
			t.Errorf("Sequence + %v: empty output", fmt)
		}
	}
}

func TestStateAllFormats(t *testing.T) {
	r := render.NewRenderer(diagram.NewRenderOptions())
	s := ast.NewState("Test")
	s.AddState(ast.DiagramState{ID: "A", Label: "State A", Kind: ast.StateStart})
	s.AddState(ast.DiagramState{ID: "B", Label: "State B", Kind: ast.StateEnd})
	s.AddTransition(ast.StateTransition{From: "A", To: "B", Event: "go"})

	formats := []diagram.OutputFormat{
		diagram.FormatMMD, diagram.FormatSVG, diagram.FormatHTML,
		diagram.FormatJSON, diagram.FormatMarkdown,
	}

	for _, fmt := range formats {
		data, err := r.RenderBytes(s, fmt)
		if err != nil {
			t.Errorf("State + %v: %v", fmt, err)
			continue
		}
		if len(data) == 0 {
			t.Errorf("State + %v: empty output", fmt)
		}
	}
}

func TestERAllFormats(t *testing.T) {
	r := render.NewRenderer(diagram.NewRenderOptions())
	e := ast.NewER("Test")
	e.AddEntity(ast.EREntity{Name: "User", Attributes: []ast.ERAttribute{
		{Name: "id", DataType: "int", Keys: []ast.ERKey{ast.KeyPrimary}},
	}})
	e.AddEntity(ast.EREntity{Name: "Order"})
	e.AddRelation(ast.ERRelation{
		From: "User", To: "Order",
		FromCard: ast.CardOneMany, ToCard: ast.CardZeroOne,
		Label: "places",
	})

	formats := []diagram.OutputFormat{
		diagram.FormatMMD, diagram.FormatSVG, diagram.FormatHTML,
		diagram.FormatJSON, diagram.FormatMarkdown,
	}

	for _, fmt := range formats {
		data, err := r.RenderBytes(e, fmt)
		if err != nil {
			t.Errorf("ER + %v: %v", fmt, err)
			continue
		}
		if len(data) == 0 {
			t.Errorf("ER + %v: empty output", fmt)
		}
	}
}

func TestClassAllFormats(t *testing.T) {
	r := render.NewRenderer(diagram.NewRenderOptions())
	c := ast.NewClass("Test")
	c.AddClass(ast.DiagramClass{
		Name: "MyClass",
		Members: []ast.ClassMember{
			{Visibility: ast.VisPublic, Type: "void", Name: "method", IsMethod: true},
		},
	})

	formats := []diagram.OutputFormat{
		diagram.FormatMMD, diagram.FormatSVG, diagram.FormatHTML,
		diagram.FormatJSON, diagram.FormatMarkdown,
	}

	for _, fmt := range formats {
		data, err := r.RenderBytes(c, fmt)
		if err != nil {
			t.Errorf("Class + %v: %v", fmt, err)
			continue
		}
		if len(data) == 0 {
			t.Errorf("Class + %v: empty output", fmt)
		}
	}
}

func TestSVGContainsNamespace(t *testing.T) {
	r := render.NewRenderer(diagram.NewRenderOptions())
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A"})

	data, err := r.RenderBytes(f, diagram.FormatSVG)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	out := string(data)
	if !strings.Contains(out, `xmlns="http://www.w3.org/2000/svg"`) {
		t.Error("SVG should contain namespace")
	}
}

func TestHTMLContainsDoctype(t *testing.T) {
	r := render.NewRenderer(diagram.NewRenderOptions())
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A"})

	data, err := r.RenderBytes(f, diagram.FormatHTML)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	out := string(data)
	if !strings.Contains(out, "<!DOCTYPE html>") {
		t.Error("HTML should contain doctype")
	}
}

func TestPDFMagicBytes(t *testing.T) {
	r := render.NewRenderer(diagram.NewRenderOptions())
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A"})

	data, err := r.RenderBytes(f, diagram.FormatPDF)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	if string(data[:5]) != "%PDF-" {
		t.Errorf("expected PDF magic bytes, got %q", string(data[:min(8, len(data))]))
	}
}

func TestPNGMagicBytes(t *testing.T) {
	r := render.NewRenderer(diagram.NewRenderOptions())
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A"})

	data, err := r.RenderBytes(f, diagram.FormatPNG)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	if data[0] != 0x89 || data[1] != 'P' || data[2] != 'N' || data[3] != 'G' {
		t.Error("expected PNG magic bytes")
	}
}

func TestJSONValid(t *testing.T) {
	r := render.NewRenderer(diagram.NewRenderOptions())
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A"})

	data, err := r.RenderBytes(f, diagram.FormatJSON)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	out := string(data)
	if !strings.Contains(out, "{") || !strings.Contains(out, "}") {
		t.Error("expected JSON braces")
	}
}

func TestDOTValid(t *testing.T) {
	r := render.NewRenderer(diagram.NewRenderOptions())
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A"})
	f.MustAddNode(&ast.FlowNode{ID: "B"})
	f.AddEdge(&ast.FlowEdge{From: "A", To: "B"})

	data, err := r.RenderBytes(f, diagram.FormatDOT)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	out := string(data)
	if !strings.Contains(out, "digraph") {
		t.Error("expected DOT digraph")
	}
}

func TestMarkdownContainsFence(t *testing.T) {
	r := render.NewRenderer(diagram.NewRenderOptions())
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A"})

	md := r.RenderMarkdown(f)
	if !strings.Contains(md, "```mermaid") {
		t.Error("expected mermaid code fence")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestRenderToFileCreatesDir(t *testing.T) {
	r := render.NewRenderer(diagram.NewRenderOptions())
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A"})

	tmp := t.TempDir()
	path := tmp + "/subdir/test.mmd"

	err := r.RenderToFile(path, f, diagram.FormatMMD)
	if err != nil {
		t.Fatalf("RenderToFile failed: %v", err)
	}
}

func TestRenderToMultipleFormats(t *testing.T) {
	r := render.NewRenderer(diagram.NewRenderOptions())
	f := ast.NewFlowchart("Multi", ast.DirectionLR)
	f.MustAddNode(&ast.FlowNode{ID: "A", Label: "Node A"})
	f.MustAddNode(&ast.FlowNode{ID: "B", Label: "Node B"})
	f.AddEdge(&ast.FlowEdge{From: "A", To: "B", Label: "Edge"})

	formats := []diagram.OutputFormat{
		diagram.FormatMMD, diagram.FormatSVG, diagram.FormatHTML,
		diagram.FormatJSON, diagram.FormatMarkdown,
	}

	for _, fmt := range formats {
		data, err := r.RenderBytes(f, fmt)
		if err != nil {
			t.Errorf("%v: %v", fmt, err)
			continue
		}
		if len(data) == 0 {
			t.Errorf("%v: empty", fmt)
		}
	}
}

func TestContentTypes(t *testing.T) {
	r := render.NewRenderer(diagram.NewRenderOptions())

	ct := r.ContentType(diagram.FormatMMD)
	if ct != "text/x-mermaid" {
		t.Errorf("MMD: expected text/x-mermaid, got %s", ct)
	}
	ct = r.ContentType(diagram.FormatSVG)
	if ct != "image/svg+xml" {
		t.Errorf("SVG: expected image/svg+xml, got %s", ct)
	}
	ct = r.ContentType(diagram.FormatPNG)
	if ct != "image/png" {
		t.Errorf("PNG: expected image/png, got %s", ct)
	}
	ct = r.ContentType(diagram.FormatHTML)
	if ct != "text/html; charset=utf-8" {
		t.Errorf("HTML: expected text/html; charset=utf-8, got %s", ct)
	}
	ct = r.ContentType(diagram.FormatPDF)
	if ct != "application/pdf" {
		t.Errorf("PDF: expected application/pdf, got %s", ct)
	}
	ct = r.ContentType(diagram.FormatJSON)
	if ct != "application/json" {
		t.Errorf("JSON: expected application/json, got %s", ct)
	}
	ct = r.ContentType(diagram.FormatDOT)
	if ct != "text/vnd.graphviz" {
		t.Errorf("DOT: expected text/vnd.graphviz, got %s", ct)
	}
	ct = r.ContentType(diagram.FormatMarkdown)
	if ct != "text/markdown; charset=utf-8" {
		t.Errorf("MD: expected text/markdown; charset=utf-8, got %s", ct)
	}
}