package svg_test

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/iokdigital/go-mermaid/ast"
)

func mindmapFixture() *ast.MindmapDiagram {
	m := ast.NewMindmap("Topics")
	root := ast.NewMindNode("root")
	a := ast.NewMindNode("Alpha")
	a.AddChild(ast.NewMindNode("Alpha one"))
	a.AddChild(ast.NewMindNode("Alpha two"))
	b := ast.NewMindNode("Beta")
	b.Shape = ast.ShapeCircle
	c := ast.NewMindNode("Gamma")
	c.Shape = ast.ShapeHexagon
	root.AddChild(a)
	root.AddChild(b)
	root.AddChild(c)
	return m.SetRoot(root)
}

func TestMindmap_SVGDoctype(t *testing.T) {
	out := encode2(t, mindmapFixture())
	if !strings.Contains(out, `<?xml version="1.0"`) {
		t.Error("missing XML declaration")
	}
	if !strings.Contains(out, `xmlns="http://www.w3.org/2000/svg"`) {
		t.Error("missing SVG namespace")
	}
}

func TestMindmap_TitleLabelsAndShapes(t *testing.T) {
	out := encode2(t, mindmapFixture())
	for _, want := range []string{
		"<title>Topics</title>",
		"root", "Alpha", "Alpha one", "Beta", "Gamma",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in SVG output", want)
		}
	}
	// Circle node -> ellipse, hex node -> polygon, rect nodes -> rect.
	if !strings.Contains(out, "<ellipse") {
		t.Error("expected ellipse for circle node")
	}
	if !strings.Contains(out, "<polygon") {
		t.Error("expected polygon for hexagon node")
	}
}

// svgWidth parses the width attribute from the SVG header.
func svgWidth(t *testing.T, svgOut string) float64 {
	t.Helper()
	idx := strings.Index(svgOut, `width="`)
	if idx < 0 {
		t.Fatal("no width attribute in SVG output")
	}
	rest := svgOut[idx+len(`width="`):]
	end := strings.Index(rest, `"`)
	if end < 0 {
		t.Fatal("unterminated width attribute")
	}
	var w float64
	if _, err := fmt.Sscanf(rest[:end], "%f", &w); err != nil {
		t.Fatalf("bad width %q: %v", rest[:end], err)
	}
	return w
}

func TestMindmap_RootCentered(t *testing.T) {
	out := encode2(t, mindmapFixture())
	rx := textAttr(t, out, "root", "x") // root center ≈ half the SVG width
	w := svgWidth(t, out)
	if math.Abs(rx-w/2) > 5 {
		t.Errorf("root x=%.1f not centered in width %.0f", rx, w)
	}
}

func TestMindmap_BalancedSides(t *testing.T) {
	out := encode2(t, mindmapFixture())
	// Alpha subtree right of root, Beta subtree left of root (alternation).
	if !(textAttr(t, out, "Alpha", "x") > textAttr(t, out, "root", "x")) {
		t.Error("first child should lay out right of root")
	}
	if !(textAttr(t, out, "Beta", "x") < textAttr(t, out, "root", "x")) {
		t.Error("second child should lay out left of root")
	}
}

func TestMindmap_EdgesConnectTree(t *testing.T) {
	out := encode2(t, mindmapFixture())
	// 6 nodes -> 5 parent-child edges.
	if got := strings.Count(out, "<path"); got < 5 {
		t.Errorf("expected at least 5 edge paths, got %d", got)
	}
}

func TestMindmap_EmptyRendersFrame(t *testing.T) {
	m := ast.NewMindmap("Nothing")
	out := encode2(t, m)
	if !strings.Contains(out, "<svg ") || !strings.Contains(out, "</svg>") {
		t.Error("empty mindmap should still render a valid SVG document")
	}
}

func TestMindmap_LabelEscaping(t *testing.T) {
	m := ast.NewMindmap(`T <&> "title"`)
	root := ast.NewMindNode(`a<b> & "q"`)
	root.AddChild(ast.NewMindNode("child"))
	m.SetRoot(root)
	out := encode2(t, m)
	if strings.Contains(out, "<b>") {
		t.Error("SVG should have escaped < in labels")
	}
	if !strings.Contains(out, "&amp;") || !strings.Contains(out, "&quot;") {
		t.Errorf("expected escaped entities in output:\n%s", out)
	}
}

func TestMindmap_UnsupportedShapeFallsBack(t *testing.T) {
	m := ast.NewMindmap("")
	root := ast.NewMindNode("root")
	odd := ast.NewMindNode("diamond")
	odd.Shape = ast.ShapeDiamond // no mindmap syntax: renders rounded
	root.AddChild(odd)
	m.SetRoot(root)
	out := encode2(t, m)
	if !strings.Contains(out, "diamond") {
		t.Error("expected node label in output")
	}
	if !strings.Contains(out, "<svg ") {
		t.Error("expected valid SVG document")
	}
}

func TestMindmap_CycleTerminates(t *testing.T) {
	m := ast.NewMindmap("")
	a := ast.NewMindNode("A")
	b := ast.NewMindNode("B")
	a.AddChild(b)
	b.AddChild(a) // misuse: must not hang the renderer
	m.SetRoot(a)
	out := encode2(t, m)
	if !strings.Contains(out, "<svg ") {
		t.Error("expected valid SVG document")
	}
}
