package html_test

import (
	"bytes"
	"strings"
	"testing"

	diagram "github.com/iokdigital/go-mermaid"
	"github.com/iokdigital/go-mermaid/ast"
	htmlenc "github.com/iokdigital/go-mermaid/html"
)

func encode(t *testing.T, d diagram.Diagram, opts diagram.RenderOptions) string {
	t.Helper()
	var buf bytes.Buffer
	if err := htmlenc.Encode(&buf, d, opts); err != nil {
		t.Fatalf("html.Encode: %v", err)
	}
	return buf.String()
}

func TestHTMLOutputUsesCDNScriptTag(t *testing.T) {
	f := ast.NewFlowchart("My Flow", ast.DirectionLR)
	got := encode(t, f, diagram.NewRenderOptions())
	if !strings.Contains(got, "cdn.jsdelivr.net") {
		t.Error("expected CDN script tag with cdn.jsdelivr.net")
	}
	if strings.Contains(got, "var mermaid") || strings.Contains(got, "function mermaid") {
		t.Error("must not contain inline JS bundle")
	}
}

func TestHTMLContainsMermaidPre(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionLR)
	f.MustAddNode(&ast.FlowNode{ID: "A", Label: "Start"})
	got := encode(t, f, diagram.NewRenderOptions())
	if !strings.Contains(got, `<pre class="mermaid">`) {
		t.Errorf("expected <pre class=\"mermaid\"> in output")
	}
	if !strings.Contains(got, "flowchart LR") {
		t.Error("expected mmd source in pre block")
	}
}

func TestHTMLContainsMermaidInit(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionLR)
	got := encode(t, f, diagram.NewRenderOptions())
	if !strings.Contains(got, "mermaid.initialize(") {
		t.Error("expected mermaid.initialize call")
	}
	if !strings.Contains(got, "startOnLoad:true") {
		t.Error("expected startOnLoad:true in initialize call")
	}
}

func TestHTMLTitleEscaped(t *testing.T) {
	f := ast.NewFlowchart("<script>alert('xss')</script>", ast.DirectionLR)
	got := encode(t, f, diagram.NewRenderOptions())
	if strings.Contains(got, "<script>alert") {
		t.Error("raw <script> tag must not appear in <title>")
	}
	if !strings.Contains(got, "&lt;script&gt;") {
		t.Errorf("expected HTML-escaped title, got:\n%s", got)
	}
}

func TestHTMLMermaidSourceInPreBlock(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionLR)
	f.MustAddNode(&ast.FlowNode{ID: "A"})
	f.MustAddNode(&ast.FlowNode{ID: "B"})
	f.AddEdge(&ast.FlowEdge{From: "A", To: "B", Style: ast.EdgeSolid})
	got := encode(t, f, diagram.NewRenderOptions())
	// Mermaid source keywords must survive in the pre block.
	// Note: html/template HTML-escapes the mmd source; mermaid.js reads
	// element.textContent which decodes entities, so diagrams still render correctly.
	if !strings.Contains(got, "flowchart LR") {
		t.Error("expected flowchart LR keyword in pre block")
	}
	if !strings.Contains(got, `<pre class="mermaid">`) {
		t.Error("expected mermaid pre block")
	}
}

func TestHTMLCDNOverride(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionLR)
	opts := diagram.NewRenderOptions()
	opts.CDNOverrideURL = "https://my-internal.example.com/mermaid.min.js"
	got := encode(t, f, opts)
	if !strings.Contains(got, "my-internal.example.com") {
		t.Error("expected custom CDN URL in script src")
	}
	if strings.Contains(got, "cdn.jsdelivr.net") {
		t.Error("default CDN URL must not appear when override is set")
	}
}

func TestHTMLCDNOverrideHTTP(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionLR)
	opts := diagram.NewRenderOptions()
	opts.CDNOverrideURL = "http://internal.corp/mermaid.js"
	got := encode(t, f, opts)
	if !strings.Contains(got, "http://internal.corp/mermaid.js") {
		t.Error("expected http:// CDN override in output")
	}
}

func TestHTMLInvalidCDNSchemeErrors(t *testing.T) {
	cases := []string{
		"ftp://bad.example.com/mermaid.min.js",
		"javascript:alert(1)",
		"//cdn.example.com/mermaid.js",
		"data:text/javascript,",
	}
	for _, url := range cases {
		opts := diagram.NewRenderOptions()
		opts.CDNOverrideURL = url
		var buf bytes.Buffer
		err := htmlenc.Encode(&buf, ast.NewFlowchart("", ast.DirectionLR), opts)
		if err == nil {
			t.Errorf("expected error for invalid CDN scheme %q", url)
		}
	}
}

func TestHTMLAllDiagramTypes(t *testing.T) {
	opts := diagram.NewRenderOptions()
	tests := []struct {
		name string
		d    diagram.Diagram
		want string
	}{
		{"flowchart", ast.NewFlowchart("FC", ast.DirectionLR), "flowchart LR"},
		{"sequence", ast.NewSequence("Seq", false), "sequenceDiagram"},
		{"state", ast.NewState("State"), "stateDiagram-v2"},
		{"er", ast.NewER("ER"), "erDiagram"},
		{"class", ast.NewClass("Class"), "classDiagram"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := encode(t, tt.d, opts)
			if !strings.Contains(got, tt.want) {
				t.Errorf("expected %q in HTML output for %s", tt.want, tt.name)
			}
			if !strings.Contains(got, `<pre class="mermaid">`) {
				t.Errorf("expected <pre class=\"mermaid\"> for %s", tt.name)
			}
		})
	}
}

func TestHTMLValidStructure(t *testing.T) {
	f := ast.NewFlowchart("Test Title", ast.DirectionTB)
	got := encode(t, f, diagram.NewRenderOptions())
	checks := []string{
		"<!DOCTYPE html>",
		`<html lang="en">`,
		`<meta charset="UTF-8">`,
		"<title>Test Title</title>",
		"</html>",
	}
	for _, want := range checks {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in HTML output", want)
		}
	}
}
