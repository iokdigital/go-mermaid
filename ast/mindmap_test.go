package ast_test

import (
	"testing"

	"github.com/iokdigital/go-mermaid/ast"
)

func TestNewMindmap(t *testing.T) {
	m := ast.NewMindmap("Topics")
	if m.Title() != "Topics" {
		t.Errorf("expected title Topics, got %q", m.Title())
	}
	if m.Type() != "mindmap" {
		t.Errorf("expected type mindmap, got %q", m.Type())
	}
	if m.Root() != nil {
		t.Error("expected nil root")
	}
}

func TestMindmapTree(t *testing.T) {
	m := ast.NewMindmap("")
	root := ast.NewMindNode("root")
	history := ast.NewMindNode("History")
	history.AddChild(ast.NewMindNode("Old legends"))
	root.AddChild(history)
	root.AddChild(ast.NewMindNode("Nature"))
	m.SetRoot(root)

	if m.Root().Label != "root" {
		t.Errorf("unexpected root: %+v", m.Root())
	}
	if len(m.Root().Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(m.Root().Children))
	}
	if len(m.Root().Children[0].Children) != 1 {
		t.Errorf("expected 1 grandchild, got %d", len(m.Root().Children[0].Children))
	}
}

func TestMindNodeShape(t *testing.T) {
	n := ast.NewMindNode("x")
	n.Shape = ast.ShapeCircle
	if n.Shape != ast.ShapeCircle {
		t.Errorf("expected circle shape, got %q", n.Shape)
	}
	n.ID = "custom"
	if n.ID != "custom" {
		t.Errorf("expected custom id, got %q", n.ID)
	}
}
