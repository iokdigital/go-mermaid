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
