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
