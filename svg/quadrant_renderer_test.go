package svg_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/iokdigital/go-mermaid/ast"
)

func quadrantFixture() *ast.QuadrantDiagram {
	q := ast.NewQuadrant("Campaigns")
	q.SetXAxis("Low Reach", "High Reach")
	q.SetYAxis("Low Engagement", "High Engagement")
	q.SetQuadrantLabel(1, "Expand")
	q.SetQuadrantLabel(2, "Promote")
	q.SetQuadrantLabel(3, "Re-evaluate")
	q.SetQuadrantLabel(4, "Improve")
	q.AddPoint(ast.QuadrantPoint{Name: "Campaign A", X: 0.3, Y: 0.6})
	q.AddPoint(ast.QuadrantPoint{Name: "Campaign B", X: 0.75, Y: 0.25})
	return q
}

func TestQuadrant_SVGDoctype(t *testing.T) {
	out := encode2(t, quadrantFixture())
	if !strings.Contains(out, `<?xml version="1.0"`) {
		t.Error("missing XML declaration")
	}
	if !strings.Contains(out, `xmlns="http://www.w3.org/2000/svg"`) {
		t.Error("missing SVG namespace")
	}
}

func TestQuadrant_TitleAxesQuadsPoints(t *testing.T) {
	out := encode2(t, quadrantFixture())
	for _, want := range []string{
		"<title>Campaigns</title>",
		"Low Reach", "High Reach", "Low Engagement", "High Engagement",
		"Expand", "Promote", "Re-evaluate", "Improve",
		"Campaign A", "Campaign B",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in SVG output", want)
		}
	}
	if strings.Count(out, "<circle") != 2 {
		t.Errorf("expected 2 point circles, got %d", strings.Count(out, "<circle"))
	}
}

// textAttr returns the named attribute of the <text> element containing label.
func textAttr(t *testing.T, svgOut, label, attr string) float64 {
	t.Helper()
	idx := strings.Index(svgOut, ">"+label+"<")
	if idx < 0 {
		t.Fatalf("label %q not found in SVG output", label)
	}
	key := " " + attr + `="`
	aIdx := strings.LastIndex(svgOut[:idx], key)
	if aIdx < 0 {
		t.Fatalf("no %s attribute before label %q", attr, label)
	}
	rest := svgOut[aIdx+len(key):]
	end := strings.Index(rest, `"`)
	if end < 0 {
		t.Fatalf("unterminated %s attribute before label %q", attr, label)
	}
	var v float64
	if _, err := fmt.Sscanf(rest[:end], "%f", &v); err != nil {
		t.Fatalf("parse %s for label %q: %v", attr, label, err)
	}
	return v
}

func TestQuadrant_AxisOrientation(t *testing.T) {
	out := encode2(t, quadrantFixture())
	// Y axis: high values plot at the top, so "High" must render above "Low".
	highY := textAttr(t, out, "High Engagement", "y")
	lowY := textAttr(t, out, "Low Engagement", "y")
	if highY >= lowY {
		t.Errorf("High label y=%.1f should be above Low label y=%.1f", highY, lowY)
	}
	// X axis: Low on the left, High on the right.
	lowX := textAttr(t, out, "Low Reach", "x")
	highX := textAttr(t, out, "High Reach", "x")
	if lowX >= highX {
		t.Errorf("Low label x=%.1f should be left of High label x=%.1f", lowX, highX)
	}
}

func TestQuadrant_EmptyRendersGrid(t *testing.T) {
	q := ast.NewQuadrant("Empty")
	out := encode2(t, q)
	if !strings.Contains(out, "<svg ") || !strings.Contains(out, "</svg>") {
		t.Error("empty quadrant should still render a valid SVG document")
	}
	if !strings.Contains(out, "<rect") {
		t.Error("empty quadrant should still render the grid")
	}
}

func TestQuadrant_OutOfRangeClamped(t *testing.T) {
	// The mmd encoder rejects out-of-range points; the SVG renderer clamps
	// them to the grid so output stays a valid document.
	q := ast.NewQuadrant("")
	q.AddPoint(ast.QuadrantPoint{Name: "Far", X: 1.5, Y: -0.5})
	out := encode2(t, q)
	if !strings.Contains(out, "Far") {
		t.Error("clamped point should still render")
	}
	if !strings.Contains(out, "<svg ") {
		t.Error("expected valid SVG document")
	}
}

func TestQuadrant_LabelEscaping(t *testing.T) {
	q := ast.NewQuadrant(`Q <&> "title"`)
	q.SetQuadrantLabel(1, `a<b> & "q"`)
	q.AddPoint(ast.QuadrantPoint{Name: `p<l>`, X: 0.5, Y: 0.5})
	out := encode2(t, q)
	if strings.Contains(out, "<b>") || strings.Contains(out, "<l>") {
		t.Error("SVG should have escaped < in labels")
	}
	if !strings.Contains(out, "&amp;") || !strings.Contains(out, "&quot;") {
		t.Errorf("expected escaped entities in output:\n%s", out)
	}
}
