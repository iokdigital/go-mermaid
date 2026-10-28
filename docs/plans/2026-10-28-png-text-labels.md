# PNG Text Labels Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** PNG output renders diagram text labels instead of shapes-only images.

**Architecture:** Keep the oksvg+rasterx pass for shapes, then overlay an independent text pass: parse `<text>` elements from our own SVG with `encoding/xml` and draw glyphs with `golang.org/x/image/font` onto the same RGBA image. No layout reflow — positions come straight from the SVG.

**Tech Stack:** `golang.org/x/image/font`, `font/opentype`, `font/sfnt` (already indirect deps via rasterx), embedded Go TTFs (`font/gofont`), `encoding/xml`. Pure Go, no CGo, no new module dependencies expected.

---

## Background (what investigation established)

- `png/renderer.go:6-7` documents the limitation: oksvg cannot process `<text>`/`<tspan>`, so PNGs contain shapes and edges but no labels ("Cannot process svg element text" warnings in test output).
- No rotation exists in any SVG renderer output (verified by grep: zero `transform`/`rotate` attributes), so the text pass only needs horizontal text.
- Every `<text>` element carries numeric `x`/`y`; the only bare `<text>` (no x/y) is `labelElement`'s two-line wrapper whose `<tspan x y>` children always carry coordinates.
- `text-anchor` values in use: `middle`, `end`, `start`, absent (=start). `dominant-baseline`: `central` or absent. `font-weight="bold"` and `font-style="italic"` occur; `font-size` 9–13.
- `golang.org/x/image@v0.30.0/font/` (already in module cache as an indirect dep) contains `opentype`, `sfnt`, and embedded `gofont` TTFs (`goregular`, `gobold`, `goitalic`, `gomedium` as `data.go` byte slices, ~150–185KB each). Promoting `x/image` to a direct dependency adds zero new modules.
- Caveat: Go fonts are NOT metric-compatible with Arial/Helvetica (our SVG `font-family`). Glyph advances will differ slightly from browser rendering; positions stay as laid out, so worst case is minor overflow inside boxes. Liberation Sans (metric-compatible, SIL OFL) is the fallback if the spike shows unacceptable drift — at the cost of vendoring a ~400KB TTF.

## Rejected alternatives

- **Replace oksvg wholesale:** survey pure-Go SVG rasterizers with text support during the spike (time-boxed, 2h); expected outcome is none mature, which is why this plan leads with the overlay approach.
- **Text-as-paths at SVG stage:** rejected — bloats SVG output, breaks text selection/copy, and churns every golden SVG expectation. The overlay keeps `FormatSVG` byte-identical.

---

### Phase 0: Spike (time-box 1–2 days, throwaway code in `/tmp`, do NOT commit)

**Files:**
- Create (throwaway): `/tmp/pngtext/main.go` (outside repo)

**Step 1: Prove glyph drawing works**

Write a throwaway program that loads `gofont/goregular`, parses one fixture SVG (`testdata/golden/mmd/TestFlowchartBasic.mmd` rendered via `render`), extracts `<text>` elements with `encoding/xml`, and draws them with `font.Drawer` onto the oksvg-rasterized image at `x*scale, y*scale`.

Run: `cd /tmp/pngtext && go run . ../../../iok/go-mermaid/testdata/golden/mmd/TestFlowchartBasic.mmd out.png`
Expected: PNG with visible labels roughly inside boxes.

**Step 2: Measure the metric drift**

Render the same diagram in a browser (or via Pele SVG screenshot) and compare label overflow: do labels stay inside node boxes? Try Liberation Sans (download, SIL OFL) as the face and compare.

Decision (record in plan before Phase 1): **Go fonts** (zero new deps, ~600KB for 4 faces) vs **vendored Liberation Sans** (metric-compatible, one new binary file + license attribution).

**Step 3: Check tspan/anchor coverage**

Verify the spike handles: `text-anchor=middle|end|start|absent`, `dominant-baseline=central|absent`, `font-weight=bold`, `font-style=italic`, two-line `<tspan x y>` wrappers, XML entities (`&amp;` — `encoding/xml` unescapes automatically, confirm).

### Task 1: Text extraction

**Files:**
- Create: `png/text.go`
- Test: `png/text_test.go`

**Step 1: Write the failing test**

```go
func TestExtractTexts(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg">
	  <text x="100" y="50" text-anchor="middle" dominant-baseline="central" font-size="13" fill="#1e293b">Hello &amp; bye</text>
	  <text text-anchor="middle" font-size="13"><tspan x="100" y="40">one</tspan><tspan x="100" y="56">two</tspan></text>
	</svg>`
	got := extractTexts([]byte(svg))
	if len(got) != 3 {
		t.Fatalf("expected 3 text runs, got %d", len(got))
	}
	if got[0].Text != "Hello & bye" {
		t.Errorf("entity not unescaped: %q", got[0].Text)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./png/ -run TestExtractTexts -v`
Expected: FAIL with "undefined: extractTexts"

**Step 3: Write minimal implementation**

```go
// textRun is one drawable string at an absolute SVG position.
type textRun struct {
	Text     string
	X, Y     float64
	Size     float64
	Anchor   string // start|middle|end
	Baseline string // central|""
	Fill     string // hex color, "" = default
	Bold     bool
	Italic   bool
}

func extractTexts(svgData []byte) []textRun {
	// encoding/xml struct with Text []xml.CharData + Tspan children;
	// bare <text> content + each <tspan x y>; skip tspan without coords
	// only if parent has x/y (inherit), else drop the run.
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./png/ -run TestExtractTexts -v`
Expected: PASS

**Step 5: Commit**

```bash
git add png/text.go png/text_test.go
git commit -m "feat(png): extract text runs from SVG"
```

### Task 2: Face loading (per spike decision)

**Files:**
- Modify: `png/text.go`
- Test: `png/text_test.go`

**Step 1: Write the failing test**

```go
func TestLoadFace(t *testing.T) {
	f, err := typeface(13, false, false)
	if err != nil {
		t.Fatalf("typeface: %v", err)
	}
	if f == nil {
		t.Fatal("nil face")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./png/ -run TestLoadFace -v`
Expected: FAIL with "undefined: typeface"

**Step 3: Write minimal implementation**

Parse the chosen embedded TTF once (`sync.OnceValues`), `opentype.NewFace` at `size*72/96` DPI-equivalent points. Cache faces by (size, bold, italic) in a small map with mutex.

**Step 4: Run test to verify it passes**

Run: `go test ./png/ -run TestLoadFace -v`
Expected: PASS

**Step 5: Commit**

```bash
git add png/text.go png/text_test.go
git commit -m "feat(png): load rasterizer typefaces"
```

### Task 3: Draw runs onto the rasterized image

**Files:**
- Modify: `png/text.go`, `png/renderer.go`
- Test: `png/text_test.go`, `png/renderer_test.go`

**Step 1: Write the failing test**

```go
func TestDrawTextNonBlank(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 100))
	// white background
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)
	n := drawTexts(img, []textRun{{Text: "Hi", X: 100, Y: 50, Size: 13, Anchor: "middle", Baseline: "central", Fill: "#1e293b"}}, 1.0)
	if n == 0 {
		t.Fatal("expected pixels drawn")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./png/ -run TestDrawTextNonBlank -v`
Expected: FAIL with "undefined: drawTexts"

**Step 3: Write minimal implementation**

`drawTexts(img, runs, scale) int`: for each run, `font.Drawer` with `Dot` at `fixed(x*scale, y*scale)` adjusted for anchor (measure advance via `Drawer.MeasureString`) and `central` baseline (shift by `(ascent-descent)/2` from `Face.Metrics()`); parse hex fill (`#rgb`/`#rrggbb`, default `#1e293b`); return non-white pixel delta or run count. Hook into `Encode` after `icon.Draw`: `drawTexts(img, extractTexts(svgData), scale)` — note `icon.SetTarget` maps viewBox to pixels, so multiply SVG coords by `float64(width)/viewBox.W` (= scale at matching aspect; assert or compute per-axis).

**Step 4: Run test to verify it passes**

Run: `go test ./png/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add png/
git commit -m "feat(png): overlay text labels on rasterized PNG"
```

### Task 4: Threshold coverage per diagram type

**Files:**
- Modify: `png/renderer_test.go`

**Step 1: Write the failing test**

```go
func TestPNGContainsTextPixels(t *testing.T) {
	// Fixture whose shapes alone yield N non-white samples; with text the
	// count must exceed the shapes-only baseline by a margin.
}
```

Implementation note: render the diagram SVG, count non-white pixels of shapes-only PNG by stripping `<text>` elements from the SVG first (test helper), then assert the real PNG exceeds baseline by ≥20%. Pure-Go rasterization is deterministic, so thresholds are stable across platforms. Cover flowchart + one of pie/quadrant/gantt/mindmap each (parametrized table, not 9 copies).

**Step 2–5:** standard: run-fail (helper missing), implement, run-pass, commit `test(png): pin text pixel coverage`.

### Task 5: Docs + cleanup

**Files:**
- Modify: `png/renderer.go` (remove stale limitation comment), `docs/user-guide.md` (PNG section: font choice, metric note), `README.md` if it mentions the limitation.

Steps: edit, `go test ./...`, commit `docs(png): text labels shipped`.

---

## Acceptance criteria

- [ ] `go test ./...` green, `go vet` clean, new code gofmt-clean
- [ ] Every diagram type's PNG shows its labels on visual inspection of one sample each
- [ ] No `Cannot process svg element text` warnings remain in test output
- [ ] SVG output byte-identical (no golden churn)
- [ ] Binary size impact recorded in the PR description (before/after `go build` bytes)
- [ ] Council review (code-reviewer subagent) → Ready to merge → PR to develop
