package svg_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	diagram "github.com/iokdigital/go-mermaid"
	"github.com/iokdigital/go-mermaid/ast"
	"github.com/iokdigital/go-mermaid/svg"
)

func ganttFixture() *ast.GanttDiagram {
	g := ast.NewGantt("Plan")
	g.SetDateFormat("YYYY-MM-DD")
	s := ast.NewGanttSection("Build")
	s.AddTask(ast.GanttTask{Name: "Foundations", ID: "a", Start: "2014-01-01", End: "10d"})
	s.AddTask(ast.GanttTask{Name: "Walls", Tags: []ast.GanttTag{ast.TagCrit}, Start: "after a", End: "7d"})
	s.AddTask(ast.GanttTask{Name: "Roof", Tags: []ast.GanttTag{ast.TagDone}, Start: "2014-01-20", End: "2014-01-25"})
	s.AddTask(ast.GanttTask{Name: "Handover", Tags: []ast.GanttTag{ast.TagMilestone}, ID: "m", Start: "2014-01-25", End: "0d"})
	g.AddSection(*s)
	return g
}

func TestGantt_SVGDoctype(t *testing.T) {
	out := encode2(t, ganttFixture())
	if !strings.Contains(out, `<?xml version="1.0"`) {
		t.Error("missing XML declaration")
	}
	if !strings.Contains(out, `xmlns="http://www.w3.org/2000/svg"`) {
		t.Error("missing SVG namespace")
	}
}

func TestGantt_TitleTasksAndSections(t *testing.T) {
	out := encode2(t, ganttFixture())
	for _, want := range []string{
		"<title>Plan</title>",
		"Foundations", "Walls", "Roof", "Handover", "Build",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in SVG output", want)
		}
	}
	// 4 bars + 1 milestone diamond.
	if !strings.Contains(out, "<rect") {
		t.Error("expected rect bars in SVG output")
	}
	if !strings.Contains(out, "<polygon") {
		t.Error("expected milestone diamond in SVG output")
	}
}

func TestGantt_AfterDependencyPositions(t *testing.T) {
	// "Walls" starts when "Foundations" ends: its bar must start at or after
	// the Foundations bar's right edge. Compare bar x/width attributes.
	out := encode2(t, ganttFixture())
	fx, fw := barGeom(t, out, "Foundations")
	wx, _ := barGeom(t, out, "Walls")
	if wx < fx+fw-1 {
		t.Errorf("Walls x=%.1f should start at/after Foundations end x=%.1f", wx, fx+fw)
	}
}

// barGeom returns the x/width of the bar row labelled name by scanning the
// rows group: label text then the following rect.
func barGeom(t *testing.T, svgOut, name string) (x, w float64) {
	t.Helper()
	idx := strings.Index(svgOut, ">"+name+"<")
	if idx < 0 {
		t.Fatalf("label %q not found", name)
	}
	rest := svgOut[idx:]
	rx := strings.Index(rest, "<rect")
	if rx < 0 {
		t.Fatalf("no bar after label %q", name)
	}
	seg := rest[rx:]
	var y, h float64
	for _, attr := range []struct {
		key string
		ptr *float64
	}{{"x=", &x}, {"width=", &w}, {"y=", &y}, {"height=", &h}} {
		a := strings.Index(seg, attr.key+`"`)
		if a < 0 {
			t.Fatalf("no %s in bar after %q", attr.key, name)
		}
		if _, err := fmt.Sscanf(seg[a+len(attr.key)+1:], "%f", attr.ptr); err != nil {
			t.Fatalf("parse %s: %v", attr.key, err)
		}
	}
	return x, w
}

func TestGantt_EmptyRendersPlaceholder(t *testing.T) {
	g := ast.NewGantt("Nothing")
	out := encode2(t, g)
	if !strings.Contains(out, "<svg ") || !strings.Contains(out, "</svg>") {
		t.Error("empty gantt should still render a valid SVG document")
	}
	if !strings.Contains(out, "(no tasks)") {
		t.Error("empty gantt should render a placeholder")
	}
}

func TestGantt_LabelEscaping(t *testing.T) {
	g := ast.NewGantt(`T <&> "title"`)
	g.AddTask(ast.GanttTask{Name: `a<b> & "q"`, Start: "2014-01-01", End: "2d"})
	out := encode2(t, g)
	if strings.Contains(out, "<b>") {
		t.Error("SVG should have escaped < in labels")
	}
	if !strings.Contains(out, "&amp;") || !strings.Contains(out, "&quot;") {
		t.Errorf("expected escaped entities in output:\n%s", out)
	}
}

func TestGantt_InvalidDiagramErrors(t *testing.T) {
	g := ast.NewGantt("")
	g.AddTask(ast.GanttTask{Name: "Bad", Start: "after missing", End: "5d"})
	var buf bytes.Buffer
	if err := svg.Encode(&buf, g, diagram.NewRenderOptions()); err == nil {
		t.Error("expected error for unresolvable gantt")
	}
}
