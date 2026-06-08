package diagram_test

import (
	"testing"

	"github.com/iokdigital/go-mermaid"
)

func TestDiagramTypeValues(t *testing.T) {
	types := map[diagram.DiagramType]string{
		diagram.TypeFlowchart: "flowchart",
		diagram.TypeSequence:  "sequence",
		diagram.TypeState:     "state",
		diagram.TypeER:        "er",
		diagram.TypeClass:     "class",
		diagram.TypePie:       "pie",
		diagram.TypeQuadrant:  "quadrant",
		diagram.TypeGantt:      "gantt",
		diagram.TypeMindMap:    "mindmap",
	}

	for dt, want := range types {
		if string(dt) != want {
			t.Errorf("DiagramType %v expected %q, got %q", dt, want, string(dt))
		}
	}
}

func TestOutputFormatValues(t *testing.T) {
	formats := map[diagram.OutputFormat]string{
		diagram.FormatMMD:      "mmd",
		diagram.FormatSVG:      "svg",
		diagram.FormatPNG:      "png",
		diagram.FormatHTML:     "html",
		diagram.FormatPDF:      "pdf",
		diagram.FormatJSON:     "json",
		diagram.FormatDOT:      "dot",
		diagram.FormatMarkdown: "md",
	}

	for fmt, want := range formats {
		if string(fmt) != want {
			t.Errorf("OutputFormat %v expected %q, got %q", fmt, want, string(fmt))
		}
	}
}

func TestDiagramTypeUnique(t *testing.T) {
	types := []diagram.DiagramType{
		diagram.TypeFlowchart, diagram.TypeSequence, diagram.TypeState,
		diagram.TypeER, diagram.TypeClass, diagram.TypePie, diagram.TypeQuadrant,
		diagram.TypeGantt, diagram.TypeMindMap,
	}

	seen := make(map[diagram.DiagramType]bool)
	for _, dt := range types {
		if seen[dt] {
			t.Errorf("Duplicate DiagramType: %v", dt)
		}
		seen[dt] = true
	}
}

func TestOutputFormatUnique(t *testing.T) {
	formats := []diagram.OutputFormat{
		diagram.FormatMMD, diagram.FormatSVG, diagram.FormatPNG,
		diagram.FormatHTML, diagram.FormatPDF, diagram.FormatJSON,
		diagram.FormatDOT, diagram.FormatMarkdown,
	}

	seen := make(map[diagram.OutputFormat]bool)
	for _, fmt := range formats {
		if seen[fmt] {
			t.Errorf("Duplicate OutputFormat: %v", fmt)
		}
		seen[fmt] = true
	}
}

func TestAllDiagramTypesCovered(t *testing.T) {
	types := []diagram.DiagramType{
		diagram.TypeFlowchart, diagram.TypeSequence, diagram.TypeState,
		diagram.TypeER, diagram.TypeClass, diagram.TypePie, diagram.TypeQuadrant,
		diagram.TypeGantt, diagram.TypeMindMap,
	}

	if len(types) != 9 {
		t.Errorf("expected 9 diagram types, got %d", len(types))
	}
}

func TestAllOutputFormatsCovered(t *testing.T) {
	formats := []diagram.OutputFormat{
		diagram.FormatMMD, diagram.FormatSVG, diagram.FormatPNG,
		diagram.FormatHTML, diagram.FormatPDF, diagram.FormatJSON,
		diagram.FormatDOT, diagram.FormatMarkdown,
	}

	if len(formats) != 8 {
		t.Errorf("expected 8 output formats, got %d", len(formats))
	}
}