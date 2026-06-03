// Package ast defines the AST types for all Mermaid diagram variants.
// It imports the root diagram package for DiagramType; the root package
// does not import ast, so there is no import cycle.
package ast

import (
	"fmt"
	"strings"

	diagram "github.com/iokdigital/go-mermaid"
)

// FlowchartDiagram represents a Mermaid flowchart.
// Emits: flowchart TB|LR|BT|RL
type FlowchartDiagram struct {
	title     string
	direction Direction
	nodes     []*FlowNode
	nodeIndex map[string]struct{}
	edges     []*FlowEdge
	subgraphs []*Subgraph
}

// Direction is the layout direction of a flowchart.
type Direction string

const (
	DirectionTB Direction = "TB"
	DirectionLR Direction = "LR"
	DirectionBT Direction = "BT"
	DirectionRL Direction = "RL"
)

// NodeShape controls the SVG and mmd shape syntax for a node.
type NodeShape string

const (
	ShapeRect          NodeShape = "rect"       // A[label]
	ShapeDiamond       NodeShape = "diamond"    // A{label}
	ShapeCircle        NodeShape = "circle"     // A((label))
	ShapeRoundedRect   NodeShape = "rounded"    // A(label)
	ShapeParallelogram NodeShape = "parallel"   // A[/label/]
	ShapeHexagon       NodeShape = "hexagon"    // A{{label}}
	ShapeStadium       NodeShape = "stadium"    // A([label])
	ShapeAsymmetric    NodeShape = "asymmetric" // A>label]
)

// EdgeStyle is the Mermaid arrow/line syntax for an edge.
type EdgeStyle string

const (
	EdgeSolid     EdgeStyle = "-->"
	EdgeDotted    EdgeStyle = "-.->"
	EdgeThick     EdgeStyle = "==>"
	EdgeInvisible EdgeStyle = "~~~"
	EdgeNoArrow   EdgeStyle = "---"
)

// FlowNode is a node in a flowchart.
type FlowNode struct {
	ID         string
	Label      string
	Shape      NodeShape
	Confidence float64           // 0.0–1.0; drives fill color in SVG when > 0
	URL        string            // optional; wraps node in <a href> in SVG output
	Metadata   map[string]string // arbitrary KV; exported in JSON
}

// FlowEdge is a directed edge between two nodes.
type FlowEdge struct {
	From  string
	To    string
	Label string
	Style EdgeStyle
}

// Subgraph groups nodes under a labelled box.
type Subgraph struct {
	ID    string
	Label string
	Nodes []string // node IDs belonging to this subgraph
}

// NewFlowchart creates an empty flowchart with the given title and direction.
func NewFlowchart(title string, direction Direction) *FlowchartDiagram {
	return &FlowchartDiagram{
		title:     title,
		direction: direction,
		nodeIndex: make(map[string]struct{}),
	}
}

func (f *FlowchartDiagram) Type() diagram.DiagramType { return diagram.TypeFlowchart }
func (f *FlowchartDiagram) Title() string              { return f.title }
func (f *FlowchartDiagram) Direction() Direction       { return f.direction }
func (f *FlowchartDiagram) Nodes() []*FlowNode         { return f.nodes }
func (f *FlowchartDiagram) Edges() []*FlowEdge         { return f.edges }
func (f *FlowchartDiagram) Subgraphs() []*Subgraph     { return f.subgraphs }

// AddNode appends a node to the flowchart.
// Returns diagram.ErrDuplicateNodeID (wrapped) if a node with the same ID already exists.
func (f *FlowchartDiagram) AddNode(n *FlowNode) (*FlowchartDiagram, error) {
	if _, exists := f.nodeIndex[n.ID]; exists {
		return f, fmt.Errorf("node %q: %w", n.ID, diagram.ErrDuplicateNodeID)
	}
	f.nodeIndex[n.ID] = struct{}{}
	f.nodes = append(f.nodes, n)
	return f, nil
}

// AddEdge appends an edge to the flowchart.
func (f *FlowchartDiagram) AddEdge(e *FlowEdge) *FlowchartDiagram {
	f.edges = append(f.edges, e)
	return f
}

// AddSubgraph appends a subgraph to the flowchart.
func (f *FlowchartDiagram) AddSubgraph(s *Subgraph) *FlowchartDiagram {
	f.subgraphs = append(f.subgraphs, s)
	return f
}

// MustAddNode is AddNode without error return — panics on duplicate ID.
// Useful for test fixtures and small hand-built diagrams.
func (f *FlowchartDiagram) MustAddNode(n *FlowNode) *FlowchartDiagram {
	if _, err := f.AddNode(n); err != nil {
		panic(err)
	}
	return f
}

// NodeShapeSyntax returns the Mermaid open/close bracket pair for a shape.
func NodeShapeSyntax(shape NodeShape) (open, close string) {
	switch shape {
	case ShapeDiamond:
		return "{", "}"
	case ShapeCircle:
		return "((", "))"
	case ShapeRoundedRect:
		return "(", ")"
	case ShapeParallelogram:
		return "[/", "/]"
	case ShapeHexagon:
		return "{{", "}}"
	case ShapeStadium:
		return "([", "])"
	case ShapeAsymmetric:
		return ">", "]"
	default: // ShapeRect and unknown
		return "[", "]"
	}
}

// sanitizeID returns a valid Mermaid node ID. Spaces and special characters
// that would break the mmd syntax are replaced with underscores.
func sanitizeID(id string) string {
	var b strings.Builder
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return b.String()
}

// SanitizeID exposes sanitizeID for use by encoders in other packages.
func SanitizeID(id string) string { return sanitizeID(id) }
