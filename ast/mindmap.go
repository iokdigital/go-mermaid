package ast

import diagram "github.com/iokdigital/go-mermaid"

// MindmapDiagram represents a Mermaid mindmap: a single-rooted tree rendered
// from indentation. A nil root encodes to a bare `mindmap` block.
type MindmapDiagram struct {
	title string
	root  *MindNode
}

// MindNode is one node in the mindmap tree. Shape uses NodeShape; only
// Circle, Rect, RoundedRect, and Hexagon have Mermaid mindmap syntax —
// other shapes encode as RoundedRect. An empty ID emits the node as a bare
// label line when the label needs no delimiters; otherwise the encoder
// assigns an automatic id. Trees must be acyclic: encoders emit each node
// once and drop repeated references.
type MindNode struct {
	ID       string
	Label    string
	Shape    NodeShape
	Children []*MindNode
}

// NewMindmap creates an empty mindmap with the given title.
func NewMindmap(title string) *MindmapDiagram {
	return &MindmapDiagram{title: title}
}

func (m *MindmapDiagram) Type() diagram.DiagramType { return diagram.TypeMindMap }
func (m *MindmapDiagram) Title() string             { return m.title }
func (m *MindmapDiagram) Root() *MindNode           { return m.root }

// SetRoot sets the tree root.
func (m *MindmapDiagram) SetRoot(n *MindNode) *MindmapDiagram {
	m.root = n
	return m
}

// NewMindNode creates a mindmap node with default (rounded) shape.
func NewMindNode(label string) *MindNode {
	return &MindNode{Label: label}
}

// AddChild appends a non-nil child to the node. Nil children are ignored.
func (n *MindNode) AddChild(c *MindNode) *MindNode {
	if c == nil {
		return n
	}
	n.Children = append(n.Children, c)
	return n
}
