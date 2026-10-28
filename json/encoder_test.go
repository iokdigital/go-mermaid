package json_test

import (
	stdjson "encoding/json"
	"bytes"
	"testing"

	diagram "github.com/iokdigital/go-mermaid"
	"github.com/iokdigital/go-mermaid/ast"
	jsonenc "github.com/iokdigital/go-mermaid/json"
)

func encode(t *testing.T, d diagram.Diagram) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jsonenc.Encode(&buf, d); err != nil {
		t.Fatalf("json.Encode: %v", err)
	}
	return buf.Bytes()
}

func TestJSONFlowchartValid(t *testing.T) {
	f := ast.NewFlowchart("Test", ast.DirectionLR)
	f.MustAddNode(&ast.FlowNode{ID: "A", Label: "Alpha", Shape: ast.ShapeRect, Confidence: 0.9})
	f.MustAddNode(&ast.FlowNode{ID: "B", Label: "Beta"})
	f.AddEdge(&ast.FlowEdge{From: "A", To: "B", Style: ast.EdgeSolid})

	raw := encode(t, f)
	if !stdjson.Valid(raw) {
		t.Errorf("invalid JSON: %s", raw)
	}

	var v map[string]any
	if err := stdjson.Unmarshal(raw, &v); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if v["type"] != "flowchart" {
		t.Errorf("expected type=flowchart, got: %v", v["type"])
	}
	if v["title"] != "Test" {
		t.Errorf("expected title=Test, got: %v", v["title"])
	}

	nodes, ok := v["nodes"].([]any)
	if !ok || len(nodes) != 2 {
		t.Errorf("expected 2 nodes, got: %v", v["nodes"])
	}
	edges, ok := v["edges"].([]any)
	if !ok || len(edges) != 1 {
		t.Errorf("expected 1 edge, got: %v", v["edges"])
	}
}

func TestJSONAllDiagramTypes(t *testing.T) {
	tests := []struct {
		name string
		d    diagram.Diagram
		want string
	}{
		{"flowchart", ast.NewFlowchart("FC", ast.DirectionLR), "flowchart"},
		{"sequence", ast.NewSequence("Seq", false), "sequence"},
		{"state", ast.NewState("State"), "state"},
		{"er", ast.NewER("ER"), "er"},
		{"class", ast.NewClass("Class"), "class"},
		{"pie", ast.NewPie("Pie"), "pie"},
		{"quadrant", ast.NewQuadrant("Quad"), "quadrant"},
		{"mindmap", ast.NewMindmap("Mind"), "mindmap"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := encode(t, tt.d)
			if !stdjson.Valid(raw) {
				t.Errorf("invalid JSON for %s: %s", tt.name, raw)
			}
			var v map[string]any
			_ = stdjson.Unmarshal(raw, &v)
			if v["type"] != tt.want {
				t.Errorf("expected type=%s, got %v", tt.want, v["type"])
			}
		})
	}
}

func TestJSONRoundTripAllFields(t *testing.T) {
	f := ast.NewFlowchart("Round Trip", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{
		ID:         "n1",
		Label:      "Node One",
		Shape:      ast.ShapeDiamond,
		Confidence: 0.85,
		URL:        "https://example.com",
		Metadata:   map[string]string{"key": "value"},
	})
	f.AddEdge(&ast.FlowEdge{From: "n1", To: "n1", Label: "loop", Style: ast.EdgeDotted})
	f.AddSubgraph(&ast.Subgraph{ID: "sg", Label: "Group", Nodes: []string{"n1"}})

	raw := encode(t, f)
	var v map[string]any
	if err := stdjson.Unmarshal(raw, &v); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	nodes := v["nodes"].([]any)
	n := nodes[0].(map[string]any)
	if n["url"] != "https://example.com" {
		t.Errorf("expected url field, got: %v", n["url"])
	}
	meta := n["metadata"].(map[string]any)
	if meta["key"] != "value" {
		t.Errorf("expected metadata.key=value, got: %v", meta["key"])
	}
	sgs := v["subgraphs"].([]any)
	if len(sgs) != 1 {
		t.Errorf("expected 1 subgraph, got: %v", len(sgs))
	}
}

func TestJSONPieFields(t *testing.T) {
	p := ast.NewPie("Fish")
	p.AddSlice(ast.PieSlice{Label: "Ahi", Value: 386})
	p.AddSlice(ast.PieSlice{Label: "Ono", Value: 15})

	var v map[string]any
	if err := stdjson.Unmarshal(encode(t, p), &v); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	slices, ok := v["slices"].([]any)
	if !ok || len(slices) != 2 {
		t.Fatalf("expected 2 slices, got: %v", v["slices"])
	}
	s0 := slices[0].(map[string]any)
	if s0["label"] != "Ahi" || s0["value"] != 386.0 {
		t.Errorf("unexpected first slice: %v", s0)
	}
}

func TestJSONQuadrantFields(t *testing.T) {
	q := ast.NewQuadrant("Campaigns")
	q.SetXAxis("Low Reach", "High Reach")
	q.SetYAxis("Low", "High")
	q.SetQuadrantLabel(1, "Expand")
	q.AddPoint(ast.QuadrantPoint{Name: "Campaign A", X: 0.3, Y: 0.6})

	var v map[string]any
	if err := stdjson.Unmarshal(encode(t, q), &v); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if v["title"] != "Campaigns" {
		t.Errorf("expected title, got: %v", v["title"])
	}
	points, ok := v["points"].([]any)
	if !ok || len(points) != 1 {
		t.Fatalf("expected 1 point, got: %v", v["points"])
	}
	pt := points[0].(map[string]any)
	if pt["name"] != "Campaign A" || pt["x"] != 0.3 || pt["y"] != 0.6 {
		t.Errorf("unexpected point: %v", pt)
	}
}

func TestJSONMindmapFields(t *testing.T) {
	m := ast.NewMindmap("Topics")
	root := ast.NewMindNode("root")
	kid := ast.NewMindNode("Kid")
	kid.Shape = ast.ShapeCircle
	root.AddChild(kid)
	m.SetRoot(root)

	var v map[string]any
	if err := stdjson.Unmarshal(encode(t, m), &v); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if v["type"] != "mindmap" {
		t.Errorf("expected type=mindmap, got: %v", v["type"])
	}
	r, ok := v["root"].(map[string]any)
	if !ok {
		t.Fatalf("expected root object, got: %v", v["root"])
	}
	if r["label"] != "root" {
		t.Errorf("expected root label, got: %v", r["label"])
	}
	children, ok := r["children"].([]any)
	if !ok || len(children) != 1 {
		t.Fatalf("expected 1 child, got: %v", r["children"])
	}
	c0 := children[0].(map[string]any)
	if c0["label"] != "Kid" || c0["shape"] != "circle" {
		t.Errorf("unexpected child: %v", c0)
	}
}

func TestJSONMindmapEmpty(t *testing.T) {
	m := ast.NewMindmap("Nothing")

	var v map[string]any
	if err := stdjson.Unmarshal(encode(t, m), &v); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, present := v["root"]; present {
		t.Errorf("expected no root field, got: %v", v["root"])
	}
}

func TestJSONMindmapCycleTerminates(t *testing.T) {
	m := ast.NewMindmap("")
	a := ast.NewMindNode("A")
	b := ast.NewMindNode("B")
	a.AddChild(b)
	b.AddChild(a) // misuse: must not hang the encoder
	m.SetRoot(a)

	raw := encode(t, m)
	if !stdjson.Valid(raw) {
		t.Errorf("invalid JSON: %s", raw)
	}
}
