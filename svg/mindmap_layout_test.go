package svg

import (
	"math"
	"testing"

	"github.com/iokdigital/go-mermaid/ast"
)

// TestMindmap_MeasuresSubtrees checks the bottom-up subtree measurement used
// by the centered two-sided layout: a leaf's extent is its node height; a
// parent's extent is max(children total + gaps, own height).
func TestMindmap_MeasuresSubtrees(t *testing.T) {
	// 6 nodes, depths 0-2 (same shape as the external mindmapFixture, which
	// lives in package svg_test and is not visible to this internal test).
	m := ast.NewMindmap("Topics")
	root := ast.NewMindNode("root")
	a := ast.NewMindNode("Alpha")
	a.AddChild(ast.NewMindNode("Alpha one"))
	a.AddChild(ast.NewMindNode("Alpha two"))
	b := ast.NewMindNode("Beta")
	c := ast.NewMindNode("Gamma")
	root.AddChild(a)
	root.AddChild(b)
	root.AddChild(c)
	m.SetRoot(root)
	seen := make(map[*ast.MindNode]bool)
	h := mindSubtreeHeight(root, mindNodeSize, seen)
	if h <= 0 || math.IsNaN(h) || math.IsInf(h, 0) {
		t.Fatalf("expected positive finite height, got %f", h)
	}

	// Leaf: extent == node height.
	leaf := ast.NewMindNode("Alpha one")
	lw, lh := mindNodeSize(leaf.Label)
	if got := mindSubtreeHeight(leaf, mindNodeSize, seen); got != lh {
		t.Errorf("leaf extent = %f, want node height %f (w=%f)", got, lh, lw)
	}

	// Parent with two leaf children: extent = sum + gap (if taller than self).
	parent := ast.NewMindNode("P")
	c1 := ast.NewMindNode("one")
	c2 := ast.NewMindNode("two")
	parent.AddChild(c1)
	parent.AddChild(c2)
	want := mindSubtreeHeight(c1, mindNodeSize, make(map[*ast.MindNode]bool)) +
		mindSubtreeHeight(c2, mindNodeSize, make(map[*ast.MindNode]bool)) + mindLevelGap
	if got := mindSubtreeHeight(parent, mindNodeSize, make(map[*ast.MindNode]bool)); math.Abs(got-want) > 0.001 {
		t.Errorf("parent extent = %f, want %f", got, want)
	}

	// Cycle misuse must not hang: ca -> cb -> ca.
	ca := ast.NewMindNode("A")
	cb := ast.NewMindNode("B")
	ca.AddChild(cb)
	cb.AddChild(ca)
	if got := mindSubtreeHeight(ca, mindNodeSize, make(map[*ast.MindNode]bool)); math.IsInf(got, 0) {
		t.Error("cycle should terminate, got +Inf")
	}
}
