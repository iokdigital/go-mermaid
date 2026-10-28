# Mindmap Centered Two-Sided Layout Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Mindmap SVG renders Mermaid-style: root centered, subtrees alternating right/left, elbow connectors.

**Architecture:** Replace the single-sided Sugiyama pass in `svg/mindmap_renderer.go` with a bespoke two-sided tree layout (measure subtrees bottom-up, place balanced halves left/right of the root, mirror the left half). Keep `mindNodeSize`, `mindNodeSVG`, `shapeElement`, and `labelElement` untouched.

**Tech Stack:** Existing `svg` package only, no new dependencies.

---

## Layout spec (matches Mermaid convention)

- Root node horizontally centered in the viewport.
- Level-1 subtrees alternate right, left, right, … (first child right).
- Each side lays out top-down; the root's vertical center aligns with the midpoint of the taller side.
- Edges are horizontal elbows: parent edge-midpoint → horizontal → vertical trunk → horizontal → child (single `<path>` with `M/H/V/H` segments, no curves).
- Node sizes, colors, and fonts unchanged; `writeSVGHeader` scaling behavior unchanged.

---

### Task 1: Subtree measurement

**Files:**
- Modify: `svg/mindmap_renderer.go`
- Test: `svg/mindmap_renderer_test.go`

**Step 1: Write the failing test**

```go
func TestMindmap_MeasuresSubtrees(t *testing.T) {
	m := mindmapFixture() // 6 nodes, depths 0-2
	h := mindSubtreeHeight(m.Root(), mindNodeSize)
	if h <= 0 {
		t.Fatalf("expected positive height, got %f", h)
	}
	// Leaf = node height; parent = max(children sum + gaps, own height).
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./svg/ -run TestMindmap_MeasuresSubtrees -v`
Expected: FAIL with "undefined: mindSubtreeHeight"

**Step 3: Write minimal implementation**

```go
// mindSubtreeHeight returns the vertical extent of the subtree.
func mindSubtreeHeight(n *ast.MindNode, size func(string) (float64, float64)) float64 {
	_, h := size(n.Label)
	if len(n.Children) == 0 {
		return h
	}
	total := 0.0
	for _, c := range n.Children {
		total += mindSubtreeHeight(c, size)
	}
	total += float64(len(n.Children)-1) * mindLevelGap // const 16.0
	return math.Max(h, total)
}
```

Guard cycles with the existing `seen` pattern (reuse the walk guard).

**Step 4: Run test to verify it passes**

Run: `go test ./svg/ -run TestMindmap_MeasuresSubtrees -v`
Expected: PASS

**Step 5: Commit**

```bash
git add svg/mindmap_renderer.go svg/mindmap_renderer_test.go
git commit -m "feat(mindmap): subtree measurement"
```

### Task 2: Two-sided placement

**Files:**
- Modify: `svg/mindmap_renderer.go`
- Test: `svg/mindmap_renderer_test.go`

**Step 1: Write the failing test**

```go
func TestMindmap_RootCentered(t *testing.T) {
	out := encode2(t, mindmapFixture())
	rx := textAttr(t, out, "root", "x") // helper exists in test file
	// root center ≈ half the SVG width
	w := svgWidth(t, out) // parse width="..." from the header
	if math.Abs((rx)-w/2) > 5 {
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
```

(`textAttr` helper already exists in `svg/mindmap_renderer_test.go`; add a `svgWidth` parser next to it.)

**Step 2: Run tests to verify they fail**

Run: `go test ./svg/ -run 'TestMindmap_RootCentered|TestMindmap_BalancedSides' -v`
Expected: FAIL (single-sided layout today)

**Step 3: Write minimal implementation**

Replace the `runGenericLayout` call with `mindLayoutTwoSided(flat, opts)` returning `map[string]*nodeLayout`-equivalent positions plus width/height:

```go
// 1. size every node via mindNodeSize
// 2. split root children alternating right/left (index 0 -> right)
// 3. stack each side top-down using mindSubtreeHeight; side height = max(children total)
// 4. root cy = gridTop + max(leftH, rightH)/2; root cx = center
// 5. place columns: vertical ranks at fixed depth offsets from center
//    (levelGapX const 120.0); node w/h from step 1
```

Keep the existing `gNode`/`nodeLayout` types: build `nodeLayout{x, y, w, h}` values directly into `lr.nodes`-shaped map so the edge/node emission code below is untouched. Preserve the `seen`/`idOf` guards.

**Step 4: Run tests to verify they pass**

Run: `go test ./svg/ -v`
Expected: PASS (existing edge/shape/label tests must keep passing unmodified — only positions change)

**Step 5: Commit**

```bash
git add svg/mindmap_renderer.go svg/mindmap_renderer_test.go
git commit -m "feat(mindmap): centered two-sided layout"
```

### Task 3: Elbow connectors

**Files:**
- Modify: `svg/mindmap_renderer.go`
- Test: `svg/mindmap_renderer_test.go`

**Step 1: Write the failing test**

```go
func TestMindmap_ElbowEdges(t *testing.T) {
	out := encode2(t, mindmapFixture())
	// elbow paths contain H/V segments; straight diagonals do not.
	if !strings.Contains(out, " H") && !strings.Contains(out, " V") {
		t.Error("expected elbow connectors with H/V segments")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./svg/ -run TestMindmap_ElbowEdges -v`
Expected: FAIL (current edges are straight `M/L` lines)

**Step 3: Write minimal implementation**

Replace the straight edge path with `M x1,y H midx V y2 H x2` where midx is the horizontal midpoint (for same-side parent/child this yields a clean elbow; root-to-first-level spans the center gap). Keep stroke `#94a3b8`, width 1.5.

**Step 4: Run tests to verify they pass**

Run: `go test ./svg/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add svg/mindmap_renderer.go svg/mindmap_renderer_test.go
git commit -m "feat(mindmap): elbow connectors"
```

### Task 4: Docs

Update the `encodeMindmap` doc comment (remove the single-sided divergence note, describe the centered layout + alternation rule) and `docs/user-guide.md` if it mentions the divergence. Run full suite, commit `docs(mindmap): centered layout`.

---

## Acceptance criteria

- [ ] `go test ./...` green, `go vet` clean, new code gofmt-clean
- [ ] All pre-existing mindmap SVG tests pass unmodified (positions may shift, assertions are positional where it matters and got updated deliberately)
- [ ] Visual check: render `TestMindmapBasic` SVG and confirm root-centered, balanced sides
- [ ] Council review → PR to develop
