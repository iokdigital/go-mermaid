package svg_test

import (
	"strings"
	"testing"

	"github.com/iokdigital/go-mermaid/ast"
)

func TestPie_SVGDoctype(t *testing.T) {
	p := ast.NewPie("test")
	p.AddSlice(ast.PieSlice{Label: "A", Value: 1})
	out := encode2(t, p)
	if !strings.Contains(out, `<?xml version="1.0"`) {
		t.Error("missing XML declaration")
	}
	if !strings.Contains(out, "<svg ") {
		t.Error("missing svg element")
	}
	if !strings.Contains(out, `xmlns="http://www.w3.org/2000/svg"`) {
		t.Error("missing SVG namespace")
	}
}

func TestPie_TitleAndSlicesInOutput(t *testing.T) {
	p := ast.NewPie("Fish")
	p.AddSlice(ast.PieSlice{Label: "Ahi", Value: 386})
	p.AddSlice(ast.PieSlice{Label: "Ono", Value: 15})
	out := encode2(t, p)
	if !strings.Contains(out, "<title>Fish</title>") {
		t.Error("title not found in SVG output")
	}
	if !strings.Contains(out, "Ahi") || !strings.Contains(out, "Ono") {
		t.Error("slice labels not found in SVG output")
	}
	// 386/401 = 96.3%, 15/401 = 3.7%.
	if !strings.Contains(out, "96.3%") {
		t.Errorf("expected 96.3%% percentage, got:\n%s", out)
	}
	if !strings.Contains(out, "3.7%") {
		t.Errorf("expected 3.7%% percentage, got:\n%s", out)
	}
}

func TestPie_IntegralPercentNoDecimals(t *testing.T) {
	p := ast.NewPie("")
	p.AddSlice(ast.PieSlice{Label: "A", Value: 3})
	p.AddSlice(ast.PieSlice{Label: "B", Value: 1})
	out := encode2(t, p)
	if !strings.Contains(out, "75%") {
		t.Errorf("expected integral 75%%, got:\n%s", out)
	}
}

func TestPie_SingleSliceFullCircle(t *testing.T) {
	p := ast.NewPie("")
	p.AddSlice(ast.PieSlice{Label: "Only", Value: 10})
	out := encode2(t, p)
	if !strings.Contains(out, "<circle") {
		t.Error("single 100% slice should render a full circle")
	}
	if !strings.Contains(out, "100%") {
		t.Error("expected 100% label")
	}
}

func TestPie_EmptyRendersPlaceholder(t *testing.T) {
	p := ast.NewPie("Nothing")
	out := encode2(t, p)
	if !strings.Contains(out, "<svg ") || !strings.Contains(out, "</svg>") {
		t.Error("empty pie should still render a valid SVG document")
	}
	if !strings.Contains(out, "(no data)") {
		t.Error("empty pie should render a placeholder")
	}
}

func TestPie_ZeroSliceLegendOnly(t *testing.T) {
	p := ast.NewPie("")
	p.AddSlice(ast.PieSlice{Label: "Zero", Value: 0})
	p.AddSlice(ast.PieSlice{Label: "Full", Value: 4})
	out := encode2(t, p)
	if !strings.Contains(out, "Zero") {
		t.Error("zero slice should appear in legend")
	}
	if strings.Count(out, "<path") != 0 {
		t.Error("zero slice must not draw a wedge")
	}
}

func TestPie_LabelEscaping(t *testing.T) {
	p := ast.NewPie("")
	p.AddSlice(ast.PieSlice{Label: `a<b> & "q"`, Value: 1})
	out := encode2(t, p)
	if strings.Contains(out, "<b>") {
		t.Error("SVG should have escaped < in labels")
	}
	if !strings.Contains(out, "&amp;") || !strings.Contains(out, "&quot;") {
		t.Errorf("expected escaped entities in output:\n%s", out)
	}
}

func TestPie_NegativeValuePlaceholder(t *testing.T) {
	// Negative values are rejected by the mmd encoder but the SVG renderer
	// stays total-safe: a non-positive total renders the placeholder.
	p := ast.NewPie("")
	p.AddSlice(ast.PieSlice{Label: "Neg", Value: -5})
	out := encode2(t, p)
	if !strings.Contains(out, "(no data)") {
		t.Error("non-positive total should render the placeholder")
	}
}
