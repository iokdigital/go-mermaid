// Package dot encodes FlowchartDiagram ASTs to Graphviz DOT format.
// Only flowchart diagrams are supported; other types return ErrRendererNotAvailable.
package dot

import (
	"fmt"
	"io"
	"strings"

	diagram "github.com/iokdigital/go-mermaid"
	"github.com/iokdigital/go-mermaid/ast"
)

// Encode writes a Graphviz DOT representation of d to w.
// Returns a FallbackFormatError wrapping ErrRendererNotAvailable for non-flowchart types.
func Encode(w io.Writer, d diagram.Diagram) error {
	f, ok := d.(*ast.FlowchartDiagram)
	if !ok {
		return &diagram.FallbackFormatError{
			Err:      fmt.Errorf("dot: %w: %T", diagram.ErrRendererNotAvailable, d),
			Fallback: diagram.FormatMMD,
		}
	}
	return encodeFlowchart(w, f)
}

func encodeFlowchart(w io.Writer, f *ast.FlowchartDiagram) error {
	rankdir := dirToRankdir(f.Direction())
	title := f.Title()

	fmt.Fprintln(w, "digraph {")
	fmt.Fprintf(w, "  rankdir=%s;\n", rankdir)
	if title != "" {
		fmt.Fprintf(w, "  label=%q;\n", title)
		fmt.Fprintln(w, "  labelloc=t;")
	}
	fmt.Fprintln(w, "  node [fontname=Helvetica shape=box style=rounded];")

	for _, sg := range f.Subgraphs() {
		sgID := sanitizeDOTID(sg.ID)
		label := sg.Label
		if label == "" {
			label = sg.ID
		}
		fmt.Fprintf(w, "  subgraph cluster_%s {\n", sgID)
		fmt.Fprintf(w, "    label=%q;\n", label)
		nodeByID := indexNodes(f.Nodes())
		for _, nid := range sg.Nodes {
			if n, ok := nodeByID[nid]; ok {
				writeNode(w, n, "    ")
			}
		}
		fmt.Fprintln(w, "  }")
	}

	// Emit nodes not in any subgraph.
	subgraphMember := make(map[string]bool)
	for _, sg := range f.Subgraphs() {
		for _, id := range sg.Nodes {
			subgraphMember[id] = true
		}
	}
	for _, n := range f.Nodes() {
		if !subgraphMember[n.ID] {
			writeNode(w, n, "  ")
		}
	}

	for _, e := range f.Edges() {
		writeEdge(w, e)
	}

	fmt.Fprintln(w, "}")
	return nil
}

func writeNode(w io.Writer, n *ast.FlowNode, indent string) {
	id := sanitizeDOTID(n.ID)
	label := n.Label
	if label == "" {
		label = n.ID
	}
	shape := dotShape(n.Shape)
	attrs := fmt.Sprintf("label=%q shape=%s", label, shape)
	if n.Confidence > 0 {
		fill := confidenceFill(n.Confidence)
		attrs += fmt.Sprintf(" style=filled fillcolor=%q", fill)
	}
	fmt.Fprintf(w, "%s%s [%s];\n", indent, id, attrs)
}

func writeEdge(w io.Writer, e *ast.FlowEdge) {
	from := sanitizeDOTID(e.From)
	to := sanitizeDOTID(e.To)
	attrs := ""
	if e.Label != "" {
		attrs = fmt.Sprintf(" [label=%q]", e.Label)
	}
	switch e.Style {
	case ast.EdgeInvisible:
		attrs = " [style=invis]"
	case ast.EdgeDotted:
		if e.Label != "" {
			attrs = fmt.Sprintf(" [style=dashed label=%q]", e.Label)
		} else {
			attrs = " [style=dashed]"
		}
	case ast.EdgeThick:
		if e.Label != "" {
			attrs = fmt.Sprintf(" [style=bold label=%q]", e.Label)
		} else {
			attrs = " [style=bold]"
		}
	case ast.EdgeNoArrow:
		if e.Label != "" {
			attrs = fmt.Sprintf(" [arrowhead=none label=%q]", e.Label)
		} else {
			attrs = " [arrowhead=none]"
		}
	}
	fmt.Fprintf(w, "  %s -> %s%s;\n", from, to, attrs)
}

func dirToRankdir(d ast.Direction) string {
	switch d {
	case ast.DirectionLR:
		return "LR"
	case ast.DirectionBT:
		return "BT"
	case ast.DirectionRL:
		return "RL"
	default:
		return "TB"
	}
}

func dotShape(s ast.NodeShape) string {
	switch s {
	case ast.ShapeDiamond:
		return "diamond"
	case ast.ShapeCircle:
		return "circle"
	case ast.ShapeRoundedRect:
		return "box"
	case ast.ShapeParallelogram:
		return "parallelogram"
	case ast.ShapeHexagon:
		return "hexagon"
	case ast.ShapeStadium:
		return "stadium"
	case ast.ShapeAsymmetric:
		return "cds"
	default:
		return "box"
	}
}

func confidenceFill(c float64) string {
	switch {
	case c >= 0.90:
		return "#90EE90"
	case c >= 0.70:
		return "#FFD700"
	default:
		return "#FFB6C1"
	}
}

func sanitizeDOTID(id string) string {
	var b strings.Builder
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return b.String()
}

func indexNodes(nodes []*ast.FlowNode) map[string]*ast.FlowNode {
	m := make(map[string]*ast.FlowNode, len(nodes))
	for _, n := range nodes {
		m[n.ID] = n
	}
	return m
}
