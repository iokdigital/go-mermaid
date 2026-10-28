package mmd_test

import (
	"bytes"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	diagram "github.com/iokdigital/go-mermaid"
	"github.com/iokdigital/go-mermaid/ast"
	"github.com/iokdigital/go-mermaid/mmd"
)

var update = flag.Bool("update", false, "overwrite golden files with current output")

// goldenPath returns the path to a golden file for the given test name.
func goldenPath(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join("..", "testdata", "golden", "mmd", name+".mmd")
}

// assertGolden compares got against the golden file, or writes the golden file
// when -update is set.
func assertGolden(t *testing.T, got string) {
	t.Helper()
	path := goldenPath(t, t.Name())
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir golden: %v", err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("golden file missing — run with -update to create: %v", err)
	}
	if string(want) != got {
		t.Errorf("mmd output differs from golden file %s\ngot:\n%s\nwant:\n%s",
			path, got, string(want))
	}
}

func encode(t *testing.T, d diagram.Diagram) string {
	t.Helper()
	var buf bytes.Buffer
	if err := mmd.Encode(&buf, d); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	return buf.String()
}

// ─── Flowchart ───────────────────────────────────────────────────────────────

func TestFlowchartBasic(t *testing.T) {
	f := ast.NewFlowchart("Component Mapping", ast.DirectionLR)
	f.MustAddNode(&ast.FlowNode{ID: "A", Label: "Start", Shape: ast.ShapeRoundedRect})
	f.MustAddNode(&ast.FlowNode{ID: "B", Label: "Decision", Shape: ast.ShapeDiamond})
	f.MustAddNode(&ast.FlowNode{ID: "C", Label: "End", Shape: ast.ShapeRoundedRect})
	f.AddEdge(&ast.FlowEdge{From: "A", To: "B", Label: "check", Style: ast.EdgeSolid})
	f.AddEdge(&ast.FlowEdge{From: "B", To: "C", Style: ast.EdgeSolid})
	assertGolden(t, encode(t, f))
}

func TestFlowchartAllShapes(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionTB)
	shapes := []struct {
		id    string
		shape ast.NodeShape
	}{
		{"rect", ast.ShapeRect},
		{"diamond", ast.ShapeDiamond},
		{"circle", ast.ShapeCircle},
		{"rounded", ast.ShapeRoundedRect},
		{"parallel", ast.ShapeParallelogram},
		{"hexagon", ast.ShapeHexagon},
		{"stadium", ast.ShapeStadium},
		{"asymmetric", ast.ShapeAsymmetric},
	}
	for _, s := range shapes {
		f.MustAddNode(&ast.FlowNode{ID: s.id, Label: s.id, Shape: s.shape})
	}
	assertGolden(t, encode(t, f))
}

func TestFlowchartAllEdgeStyles(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionLR)
	for _, id := range []string{"A", "B", "C", "D", "E"} {
		f.MustAddNode(&ast.FlowNode{ID: id, Label: id, Shape: ast.ShapeRect})
	}
	f.AddEdge(&ast.FlowEdge{From: "A", To: "B", Style: ast.EdgeSolid, Label: "solid"})
	f.AddEdge(&ast.FlowEdge{From: "B", To: "C", Style: ast.EdgeDotted, Label: "dotted"})
	f.AddEdge(&ast.FlowEdge{From: "C", To: "D", Style: ast.EdgeThick, Label: "thick"})
	f.AddEdge(&ast.FlowEdge{From: "D", To: "E", Style: ast.EdgeNoArrow, Label: "no-arrow"})
	f.AddEdge(&ast.FlowEdge{From: "E", To: "A", Style: ast.EdgeInvisible}) // invisible: label dropped
	assertGolden(t, encode(t, f))
}

func TestFlowchartSubgraph(t *testing.T) {
	f := ast.NewFlowchart("Pipeline", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "in", Label: "Input", Shape: ast.ShapeRect})
	f.MustAddNode(&ast.FlowNode{ID: "proc", Label: "Process", Shape: ast.ShapeRect})
	f.MustAddNode(&ast.FlowNode{ID: "out", Label: "Output", Shape: ast.ShapeRect})
	f.AddSubgraph(&ast.Subgraph{ID: "sg1", Label: "Inner", Nodes: []string{"proc"}})
	f.AddEdge(&ast.FlowEdge{From: "in", To: "proc", Style: ast.EdgeSolid})
	f.AddEdge(&ast.FlowEdge{From: "proc", To: "out", Style: ast.EdgeSolid})
	assertGolden(t, encode(t, f))
}

func TestFlowchartConfidenceColors(t *testing.T) {
	f := ast.NewFlowchart("Confidence", ast.DirectionLR)
	f.MustAddNode(&ast.FlowNode{ID: "hi", Label: "High", Confidence: 0.95})
	f.MustAddNode(&ast.FlowNode{ID: "mid", Label: "Mid", Confidence: 0.75})
	f.MustAddNode(&ast.FlowNode{ID: "lo", Label: "Low", Confidence: 0.50})
	f.MustAddNode(&ast.FlowNode{ID: "none", Label: "None", Confidence: 0})
	assertGolden(t, encode(t, f))
}

func TestFlowchartEmptyNodesEdges(t *testing.T) {
	f := ast.NewFlowchart("Empty", ast.DirectionTB)
	got := encode(t, f)
	if strings.TrimSpace(got) == "" {
		t.Error("expected non-empty output for empty flowchart")
	}
	if !strings.Contains(got, "flowchart TB") {
		t.Errorf("expected flowchart TB in output, got: %s", got)
	}
}

// ─── Negative tests ───────────────────────────────────────────────────────────

func TestDuplicateNodeIDRejected(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionLR)
	if _, err := f.AddNode(&ast.FlowNode{ID: "A"}); err != nil {
		t.Fatalf("unexpected error adding first node: %v", err)
	}
	_, err := f.AddNode(&ast.FlowNode{ID: "A"})
	if err == nil {
		t.Fatal("expected ErrDuplicateNodeID, got nil")
	}
	if !errors.Is(err, diagram.ErrDuplicateNodeID) {
		t.Errorf("expected ErrDuplicateNodeID, got: %v", err)
	}
}

func TestLabelTruncation(t *testing.T) {
	long := strings.Repeat("x", 41)
	f := ast.NewFlowchart("", ast.DirectionLR)
	f.MustAddNode(&ast.FlowNode{ID: "n", Label: long})
	got := encode(t, f)
	// SVG truncation is in the SVG renderer; mmd must preserve full label.
	// The mmd encoder itself truncates labels > 40 chars with ellipsis.
	if strings.Contains(got, long) {
		t.Error("mmd encoder should truncate labels > 40 chars")
	}
	if !strings.Contains(got, "…") {
		t.Error("mmd encoder should append ellipsis to truncated label")
	}
}

func TestInvalidFormatReturnsError(t *testing.T) {
	var buf bytes.Buffer
	// Pass an unsupported concrete type (not one of the ast diagram types)
	err := mmd.Encode(&buf, unexportedDiagram{})
	if err == nil {
		t.Fatal("expected error encoding unknown diagram type")
	}
	if !errors.Is(err, diagram.ErrInvalidFormat) {
		t.Errorf("expected ErrInvalidFormat, got: %v", err)
	}
}

// unexportedDiagram is a test stub that implements diagram.Diagram but is not
// one of the ast types, triggering the ErrInvalidFormat path.
type unexportedDiagram struct{}

func (unexportedDiagram) Type() diagram.DiagramType { return "unknown" }
func (unexportedDiagram) Title() string              { return "" }

// ─── Sequence ────────────────────────────────────────────────────────────────

func TestSequenceBasic(t *testing.T) {
	s := ast.NewSequence("API Call", false)
	s.AddParticipant(ast.Participant{Alias: "Client", Kind: ast.ParticipantActor})
	s.AddParticipant(ast.Participant{Alias: "Server", Kind: ast.ParticipantBox})
	s.AddMessage(ast.SeqMessage{From: "Client", To: "Server", Text: "GET /api/v1/jobs", Style: ast.MsgSync})
	s.AddMessage(ast.SeqMessage{From: "Server", To: "Client", Text: "200 OK", Style: ast.MsgAsync})
	assertGolden(t, encode(t, s))
}

func TestSequenceAutonumber(t *testing.T) {
	s := ast.NewSequence("", true)
	s.AddParticipant(ast.Participant{Alias: "A", Kind: ast.ParticipantBox})
	s.AddParticipant(ast.Participant{Alias: "B", Kind: ast.ParticipantBox})
	s.AddMessage(ast.SeqMessage{From: "A", To: "B", Text: "hello", Style: ast.MsgSync})
	got := encode(t, s)
	if !strings.Contains(got, "autonumber") {
		t.Error("expected autonumber in output")
	}
}

// ─── State ───────────────────────────────────────────────────────────────────

func TestStateJobLifecycle(t *testing.T) {
	s := ast.NewState("Job Lifecycle")
	s.AddTransition(ast.StateTransition{From: "[*]", To: "Pending"})
	s.AddTransition(ast.StateTransition{From: "Pending", To: "Running", Event: "start"})
	s.AddTransition(ast.StateTransition{From: "Running", To: "Complete", Event: "done"})
	s.AddTransition(ast.StateTransition{From: "Running", To: "Failed", Event: "error"})
	s.AddTransition(ast.StateTransition{From: "Complete", To: "[*]"})
	assertGolden(t, encode(t, s))
}

// ─── ER ──────────────────────────────────────────────────────────────────────

func TestERBasic(t *testing.T) {
	e := ast.NewER("Schema Mapping")
	e.AddEntity(ast.EREntity{
		Name: "User",
		Attributes: []ast.ERAttribute{
			{DataType: "int", Name: "id", Keys: []ast.ERKey{ast.KeyPrimary}},
			{DataType: "string", Name: "email", Keys: []ast.ERKey{ast.KeyUnique}},
		},
	})
	e.AddEntity(ast.EREntity{
		Name: "Order",
		Attributes: []ast.ERAttribute{
			{DataType: "int", Name: "id", Keys: []ast.ERKey{ast.KeyPrimary}},
			{DataType: "int", Name: "userId", Keys: []ast.ERKey{ast.KeyForeign}},
		},
	})
	e.AddRelation(ast.ERRelation{
		From: "User", To: "Order",
		FromCard: ast.CardExactOne, ToCard: ast.CardZeroMany,
		Label: "places", Identifying: true,
	})
	assertGolden(t, encode(t, e))
}

// ─── Pie ─────────────────────────────────────────────────────────────────────

func TestPieBasic(t *testing.T) {
	p := ast.NewPie("Fish caught by the crew")
	p.AddSlice(ast.PieSlice{Label: "ʻAhi", Value: 386})
	p.AddSlice(ast.PieSlice{Label: "Mahimahi", Value: 85})
	p.AddSlice(ast.PieSlice{Label: "Ono", Value: 15})
	assertGolden(t, encode(t, p))
}

func TestPieSpecialChars(t *testing.T) {
	p := ast.NewPie(`Catch "report": 2024 & beyond`)
	p.AddSlice(ast.PieSlice{Label: `a:b | c & "quoted"`, Value: 2.5})
	p.AddSlice(ast.PieSlice{Label: "plain", Value: 7.5})
	assertGolden(t, encode(t, p))
}

func TestPieNegativeValueErrors(t *testing.T) {
	p := ast.NewPie("")
	p.AddSlice(ast.PieSlice{Label: "bad", Value: -3})
	var buf bytes.Buffer
	if err := mmd.Encode(&buf, p); err == nil {
		t.Error("expected error for negative pie slice value")
	}
}

func TestPieEmpty(t *testing.T) {
	p := ast.NewPie("Nothing yet")
	assertGolden(t, encode(t, p))
}

// ─── Quadrant ────────────────────────────────────────────────────────────────

func TestQuadrantBasic(t *testing.T) {
	q := ast.NewQuadrant("Reach and engagement of campaigns")
	q.SetXAxis("Low Reach", "High Reach")
	q.SetYAxis("Low Engagement", "High Engagement")
	q.SetQuadrantLabel(1, "We should expand")
	q.SetQuadrantLabel(2, "Need to promote")
	q.SetQuadrantLabel(3, "Re-evaluate")
	q.SetQuadrantLabel(4, "May be improved")
	q.AddPoint(ast.QuadrantPoint{Name: "Campaign A", X: 0.3, Y: 0.6})
	q.AddPoint(ast.QuadrantPoint{Name: "Campaign B", X: 0.45, Y: 0.23})
	assertGolden(t, encode(t, q))
}

func TestQuadrantSpecialChars(t *testing.T) {
	q := ast.NewQuadrant(`Q1 "review": growth & risk`)
	q.SetXAxis("Cost: low", "Cost: high")
	q.SetQuadrantLabel(1, `Grow "fast" & sustainably`)
	q.AddPoint(ast.QuadrantPoint{Name: `Alpha: "first" & best`, X: 0.8, Y: 0.9})
	assertGolden(t, encode(t, q))
}

func TestQuadrantOutOfRangeErrors(t *testing.T) {
	for _, pt := range []ast.QuadrantPoint{
		{Name: "x-high", X: 1.5, Y: 0.5},
		{Name: "y-neg", X: 0.5, Y: -0.1},
	} {
		q := ast.NewQuadrant("")
		q.AddPoint(pt)
		var buf bytes.Buffer
		if err := mmd.Encode(&buf, q); err == nil {
			t.Errorf("expected error for out-of-range point %+v", pt)
		}
	}
}

func TestQuadrantMinimal(t *testing.T) {
	q := ast.NewQuadrant("")
	q.AddPoint(ast.QuadrantPoint{Name: "A", X: 0.3, Y: 0.6})
	assertGolden(t, encode(t, q))
}

// ─── Mindmap ─────────────────────────────────────────────────────────────────

func mindmapFixture() *ast.MindmapDiagram {
	m := ast.NewMindmap("Hawaiian Islands")
	root := ast.NewMindNode("Hawaiʻi")
	history := ast.NewMindNode("History")
	history.AddChild(ast.NewMindNode("Old legends"))
	voyaging := ast.NewMindNode("Voyaging")
	voyaging.Shape = ast.ShapeCircle
	history.AddChild(voyaging)
	root.AddChild(history)
	nature := ast.NewMindNode("Nature")
	nature.Shape = ast.ShapeRect
	mountains := ast.NewMindNode("Mountains and volcanoes")
	mountains.Shape = ast.ShapeHexagon
	nature.AddChild(mountains)
	nature.AddChild(ast.NewMindNode("Ocean life"))
	root.AddChild(nature)
	return m.SetRoot(root)
}

func TestMindmapBasic(t *testing.T) {
	assertGolden(t, encode(t, mindmapFixture()))
}

func TestMindmapSpecialChars(t *testing.T) {
	m := ast.NewMindmap(`Q1 "review": growth & risk <2026>`)
	root := ast.NewMindNode("Root & branch")
	root.AddChild(ast.NewMindNode(`Say "hi" | a: b`))
	brackets := ast.NewMindNode("a[b] (c)")
	root.AddChild(brackets)
	shaped := ast.NewMindNode(`quoted "shape"`)
	shaped.Shape = ast.ShapeCircle
	shaped.ID = "my id"
	root.AddChild(shaped)
	multiline := ast.NewMindNode("line1\nline2")
	root.AddChild(multiline)
	m.SetRoot(root)
	assertGolden(t, encode(t, m))
}

func TestMindmapMinimal(t *testing.T) {
	m := ast.NewMindmap("")
	m.SetRoot(ast.NewMindNode("root"))
	assertGolden(t, encode(t, m))
}

func TestMindmapEmpty(t *testing.T) {
	m := ast.NewMindmap("Nothing yet")
	assertGolden(t, encode(t, m))
}

func TestMindmapBraces(t *testing.T) {
	m := ast.NewMindmap("")
	root := ast.NewMindNode("a{{b}} and {c}")
	root.AddChild(ast.NewMindNode("plain"))
	m.SetRoot(root)
	assertGolden(t, encode(t, m))
}

func TestMindmapCycleTerminates(t *testing.T) {
	m := ast.NewMindmap("")
	a := ast.NewMindNode("A")
	b := ast.NewMindNode("B")
	a.AddChild(b)
	b.AddChild(a) // misuse: must not hang the encoder
	m.SetRoot(a)
	var buf bytes.Buffer
	if err := mmd.Encode(&buf, m); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected non-empty output for cyclic graph")
	}
}

func TestMindmapNilChildIgnored(t *testing.T) {
	n := ast.NewMindNode("A")
	n.AddChild(nil)
	if len(n.Children) != 0 {
		t.Errorf("expected nil child to be ignored, got %d children", len(n.Children))
	}
}

func TestMindmapWhitespaceLabelPlaceholder(t *testing.T) {
	m := ast.NewMindmap("")
	m.SetRoot(ast.NewMindNode("  \n "))
	var buf bytes.Buffer
	if err := mmd.Encode(&buf, m); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	// "(empty)" itself needs delimiters, so it renders entity-escaped.
	if !strings.Contains(buf.String(), "#40;empty#41;") {
		t.Errorf("expected whitespace label to become (empty), got:\n%s", buf.String())
	}
}

func TestMindmapAutoIDCollisionAvoided(t *testing.T) {
	m := ast.NewMindmap("")
	root := ast.NewMindNode("root [needs id]")
	root.AddChild(ast.NewMindNode("explicit"))
	explicit := ast.NewMindNode("has id")
	explicit.ID = "m1" // collides with the first auto-id
	root.AddChild(explicit)
	m.SetRoot(root)
	var buf bytes.Buffer
	if err := mmd.Encode(&buf, m); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	out := buf.String()
	// The explicit m1 keeps its id; the auto id must skip to m2.
	if strings.Count(out, "m1") != 1 {
		t.Errorf("expected exactly one m1 reference, got:\n%s", out)
	}
	if !strings.Contains(out, "m2") {
		t.Errorf("expected auto-id to skip to m2, got:\n%s", out)
	}
}

// ─── Class ───────────────────────────────────────────────────────────────────

func TestClassBasic(t *testing.T) {
	c := ast.NewClass("Component Hierarchy")
	c.AddClass(ast.DiagramClass{
		Name:       "Converter",
		Annotation: "<<interface>>",
		Members: []ast.ClassMember{
			{Visibility: ast.VisPublic, Type: "string", Name: "Convert", IsMethod: true},
		},
	})
	c.AddClass(ast.DiagramClass{
		Name: "AlteryxConverter",
		Members: []ast.ClassMember{
			{Visibility: ast.VisPublic, Type: "string", Name: "Convert", IsMethod: true},
		},
	})
	c.AddRelation(ast.ClassRelation{
		From: "AlteryxConverter", To: "Converter", Kind: ast.RelRealization,
	})
	assertGolden(t, encode(t, c))
}
