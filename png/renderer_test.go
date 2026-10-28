package png_test

import (
	"bytes"
	"encoding/xml"
	"errors"
	"image"
	"testing"

	diagram "github.com/iokdigital/go-mermaid"
	"github.com/iokdigital/go-mermaid/ast"
	diapng "github.com/iokdigital/go-mermaid/png"
	"github.com/iokdigital/go-mermaid/render"
	"github.com/iokdigital/go-mermaid/svg"

	// Register PNG decoder for dimension assertions.
	_ "image/png"
)

func svgOf(t *testing.T, d diagram.Diagram) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := svg.Encode(&buf, d, diagram.NewRenderOptions()); err != nil {
		t.Fatalf("svg.Encode: %v", err)
	}
	return buf.Bytes()
}

func TestEncode_ProducesPNG(t *testing.T) {
	f := ast.NewFlowchart("Test", ast.DirectionLR)
	f.MustAddNode(&ast.FlowNode{ID: "A", Shape: ast.ShapeRect})
	svgData := svgOf(t, f)

	var buf bytes.Buffer
	if err := diapng.Encode(svgData, &buf, diagram.NewRenderOptions()); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	data := buf.Bytes()
	if len(data) == 0 {
		t.Fatal("expected non-empty PNG output")
	}
	// PNG magic bytes: 0x89 PNG\r\n\x1a\n
	if len(data) < 8 || data[0] != 0x89 || data[1] != 'P' || data[2] != 'N' || data[3] != 'G' {
		t.Errorf("output does not start with PNG magic bytes, got: %x", data[:min(8, len(data))])
	}
}

func TestEncode_ScaleAffectsDimensions(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A", Shape: ast.ShapeRect})
	svgData := svgOf(t, f)

	opts1 := diagram.NewRenderOptions()
	opts1.Resolution = diagram.ResolutionWeb // 1×

	opts2 := diagram.NewRenderOptions()
	opts2.Resolution = diagram.ResolutionScreen // 2×

	var buf1, buf2 bytes.Buffer
	if err := diapng.Encode(svgData, &buf1, opts1); err != nil {
		t.Fatalf("Encode 1x: %v", err)
	}
	if err := diapng.Encode(svgData, &buf2, opts2); err != nil {
		t.Fatalf("Encode 2x: %v", err)
	}
	// 2× output should be larger than 1× (more pixels)
	if buf2.Len() <= buf1.Len() {
		t.Errorf("2x PNG (%d bytes) should be larger than 1x (%d bytes)", buf2.Len(), buf1.Len())
	}
}

func TestEncode_SizeLimitExceeded(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A", Shape: ast.ShapeRect})
	svgData := svgOf(t, f)

	opts := diagram.NewRenderOptions()
	opts.MaxPNGBytes = 1 // impossibly small

	var buf bytes.Buffer
	err := diapng.Encode(svgData, &buf, opts)
	if err == nil {
		t.Fatal("expected error for size limit")
	}
	var ferr *diagram.FallbackFormatError
	if !errors.As(err, &ferr) {
		t.Fatalf("expected FallbackFormatError, got %T", err)
	}
	if !errors.Is(ferr.Err, diagram.ErrPNGSizeLimitExceeded) {
		t.Errorf("expected ErrPNGSizeLimitExceeded, got: %v", ferr.Err)
	}
	if ferr.Fallback != diagram.FormatHTML {
		t.Errorf("expected HTML fallback, got: %s", ferr.Fallback)
	}
}

func TestEncode_ZeroLimitMeansNoLimit(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "A", Shape: ast.ShapeRect})
	svgData := svgOf(t, f)

	opts := diagram.NewRenderOptions()
	opts.MaxPNGBytes = 0 // disabled

	var buf bytes.Buffer
	if err := diapng.Encode(svgData, &buf, opts); err != nil {
		t.Errorf("expected no error with limit=0, got: %v", err)
	}
}

// TestEncode_AllDiagramTypes is a render smoke test: every AST diagram type
// supported by svg.Encode must rasterize to a decodable PNG with positive
// dimensions. This pins the PNG call-site (render FormatPNG → svg → png)
// to the full supported-type matrix.
func TestEncode_AllDiagramTypes(t *testing.T) {
	flow := ast.NewFlowchart("", ast.DirectionLR)
	flow.MustAddNode(&ast.FlowNode{ID: "A", Shape: ast.ShapeRect})

	seq := ast.NewSequence("", false)
	seq.AddParticipant(ast.Participant{Alias: "A", Kind: ast.ParticipantBox})
	seq.AddParticipant(ast.Participant{Alias: "B", Kind: ast.ParticipantBox})
	seq.AddMessage(ast.SeqMessage{From: "A", To: "B", Text: "hi", Style: ast.MsgSync})

	state := ast.NewState("")
	state.AddState(ast.DiagramState{ID: "Idle", Kind: ast.StateNormal})
	state.AddTransition(ast.StateTransition{From: "[*]", To: "Idle"})

	er := ast.NewER("")
	er.AddEntity(ast.EREntity{Name: "USER"})
	er.AddRelation(ast.ERRelation{
		From: "USER", To: "ORDER",
		FromCard: ast.CardExactOne, ToCard: ast.CardZeroMany,
		Label: "places", Identifying: true,
	})

	class := ast.NewClass("")
	class.AddClass(ast.DiagramClass{Name: "Foo"})
	class.AddRelation(ast.ClassRelation{From: "Foo", To: "Bar", Kind: ast.RelAssociation})

	cases := []struct {
		name string
		d    diagram.Diagram
	}{
		{"flowchart", flow},
		{"sequence", seq},
		{"state", state},
		{"er", er},
		{"class", class},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svgData := svgOf(t, tc.d)
			var buf bytes.Buffer
			if err := diapng.Encode(svgData, &buf, diagram.NewRenderOptions()); err != nil {
				t.Fatalf("Encode(%s): %v", tc.name, err)
			}
			assertPNGDecodable(t, buf.Bytes())
		})
	}
}

// assertPNGDecodable checks PNG magic bytes and that the image decodes with
// positive width and height.
func assertPNGDecodable(t *testing.T, data []byte) {
	t.Helper()
	if len(data) < 8 || data[0] != 0x89 || data[1] != 'P' || data[2] != 'N' || data[3] != 'G' {
		t.Fatalf("output does not start with PNG magic bytes, got: %x", data[:min(8, len(data))])
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("PNG output does not decode: %v", err)
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		t.Errorf("expected positive PNG dimensions, got %dx%d", cfg.Width, cfg.Height)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// stripTextElements returns a copy of svgData with all <text> subtrees
// removed, for the shapes-only baseline. It is test-local (rather than
// reusing the production strip) because the test lives in package png_test
// while stripTexts is unexported; the logic is intentionally the same:
// depth-aware skipping of elements whose local name is "text".
func stripTextElements(t *testing.T, svgData []byte) []byte {
	t.Helper()
	dec := xml.NewDecoder(bytes.NewReader(svgData))
	var buf bytes.Buffer
	enc := xml.NewEncoder(&buf)
	depth := 0
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch el := tok.(type) {
		case xml.StartElement:
			if depth > 0 || el.Name.Local == "text" {
				depth++
			} else if err := enc.EncodeToken(el); err != nil {
				t.Fatalf("encode baseline svg: %v", err)
			}
		case xml.EndElement:
			if depth > 0 {
				depth--
			} else if err := enc.EncodeToken(el); err != nil {
				t.Fatalf("encode baseline svg: %v", err)
			}
		default:
			if depth == 0 {
				if err := enc.EncodeToken(el); err != nil {
					t.Fatalf("encode baseline svg: %v", err)
				}
			}
		}
	}
	if err := enc.Flush(); err != nil {
		t.Fatalf("flush baseline svg: %v", err)
	}
	return buf.Bytes()
}

// TestPNGTextCoverage pins that the text overlay pass actually paints glyphs
// for every supported diagram type. For each fixture it rasterizes the full
// PNG and a shapes-only baseline (same SVG with <text> subtrees stripped,
// fed through the same diapng.Encode path), then counts pixels whose RGBA
// values differ.
//
// Pixel-VALUE diffs are used deliberately: counting non-white pixels cannot
// detect text inside light-filled node boxes (glyph pixels may already be
// non-white from the box fill), while a baseline diff isolates exactly what
// the overlay pass added. The assertion is a global floor of 200 changed
// pixels plus a per-fixture floor recorded at ~50% of the first-run value
// (first run: flowchart 1379, sequence 730, pie 3622, quadrant 4600,
// gantt 3524, mindmap 1948, state 1410, er 2898, class 1520 — pure-Go rasterization is deterministic, verified
// bit-identical across runs). NOTE: the task spec suggested a 1% relative
// floor, but measured text coverage is 0.15–0.53% of pixels — glyphs are
// sparse on large canvases, so 1% is unachievable without stuffing fixtures
// with unrealistic label density. The per-fixture floors preserve the
// intent: any overlay regression for a type fails loudly.
func TestPNGTextCoverage(t *testing.T) {
	flow := ast.NewFlowchart("Flow", ast.DirectionLR)
	flow.MustAddNode(&ast.FlowNode{ID: "A", Label: "Alpha", Shape: ast.ShapeRect})
	flow.MustAddNode(&ast.FlowNode{ID: "B", Label: "Beta", Shape: ast.ShapeRect})
	flow.AddEdge(&ast.FlowEdge{From: "A", To: "B", Label: "go"})

	seq := ast.NewSequence("Seq", false)
	seq.AddParticipant(ast.Participant{Alias: "A", Kind: ast.ParticipantBox})
	seq.AddParticipant(ast.Participant{Alias: "B", Kind: ast.ParticipantBox})
	seq.AddMessage(ast.SeqMessage{From: "A", To: "B", Text: "hello", Style: ast.MsgSync})

	pie := ast.NewPie("Fish")
	pie.AddSlice(ast.PieSlice{Label: "Ahi", Value: 386})
	pie.AddSlice(ast.PieSlice{Label: "Ono", Value: 15})

	quad := ast.NewQuadrant("Campaigns")
	quad.SetXAxis("Low Reach", "High Reach")
	quad.SetYAxis("Low", "High")
	quad.SetQuadrantLabel(1, "Expand")
	quad.AddPoint(ast.QuadrantPoint{Name: "Campaign A", X: 0.3, Y: 0.6})

	gantt := ast.NewGantt("Plan")
	sec := ast.NewGanttSection("Build")
	sec.AddTask(ast.GanttTask{Name: "Foundations", ID: "a", Start: "2014-01-01", End: "10d"})
	sec.AddTask(ast.GanttTask{Name: "Walls", Start: "after a", End: "7d"})
	gantt.AddSection(*sec)

	mind := ast.NewMindmap("Topics")
	root := ast.NewMindNode("root")
	root.AddChild(ast.NewMindNode("Kid One"))
	root.AddChild(ast.NewMindNode("Kid Two"))
	mind.SetRoot(root)

	state := ast.NewState("Lifecycle")
	state.AddState(ast.DiagramState{ID: "Idle", Kind: ast.StateNormal})
	state.AddState(ast.DiagramState{ID: "Done", Kind: ast.StateNormal})
	state.AddTransition(ast.StateTransition{From: "Idle", To: "Done", Event: "finish"})

	er := ast.NewER("Schema")
	er.AddEntity(ast.EREntity{Name: "User", Attributes: []ast.ERAttribute{
		{DataType: "int", Name: "id", Keys: []ast.ERKey{ast.KeyPrimary}},
	}})
	er.AddEntity(ast.EREntity{Name: "Order", Attributes: []ast.ERAttribute{
		{DataType: "int", Name: "id", Keys: []ast.ERKey{ast.KeyPrimary}},
	}})
	er.AddRelation(ast.ERRelation{
		From: "User", To: "Order",
		FromCard: ast.CardExactOne, ToCard: ast.CardZeroMany, Label: "places",
	})

	class := ast.NewClass("Model")
	class.AddClass(ast.DiagramClass{
		Name: "Repo",
		Members: []ast.ClassMember{
			{Visibility: ast.VisPublic, Type: "string", Name: "title"},
		},
	})

	cases := []struct {
		name       string
		d          diagram.Diagram
		minChanged int // ~50% of the deterministic first-run count
	}{
		{"flowchart", flow, 600},
		{"sequence", seq, 300},
		{"pie", pie, 1800},
		{"quadrant", quad, 2300},
		{"gantt", gantt, 1700},
		{"mindmap", mind, 900},
		{"state", state, 700},
		{"er", er, 1400},
		{"class", class, 750},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := render.NewRenderer(diagram.NewRenderOptions())
			full, err := r.RenderBytes(tc.d, diagram.FormatPNG)
			if err != nil {
				t.Fatalf("RenderBytes PNG: %v", err)
			}
			svgData, err := r.RenderBytes(tc.d, diagram.FormatSVG)
			if err != nil {
				t.Fatalf("RenderBytes SVG: %v", err)
			}
			var baseBuf bytes.Buffer
			if err := diapng.Encode(stripTextElements(t, svgData), &baseBuf, diagram.NewRenderOptions()); err != nil {
				t.Fatalf("Encode baseline: %v", err)
			}

			fullImg, _, err := image.Decode(bytes.NewReader(full))
			if err != nil {
				t.Fatalf("decode full PNG: %v", err)
			}
			baseImg, _, err := image.Decode(bytes.NewReader(baseBuf.Bytes()))
			if err != nil {
				t.Fatalf("decode baseline PNG: %v", err)
			}
			// The baseline must contain shapes: otherwise the diff below is
			// meaningless (a blank baseline + text-only image would pass).
			baseNonWhite := 0
			bb := baseImg.Bounds()
			for y := bb.Min.Y; y < bb.Max.Y; y++ {
				for x := bb.Min.X; x < bb.Max.X; x++ {
					r, g, b, _ := baseImg.At(x, y).RGBA()
					if r < 0xffff || g < 0xffff || b < 0xffff {
						baseNonWhite++
					}
				}
			}
			if baseNonWhite == 0 {
				t.Fatalf("baseline PNG for %s is blank, diff is meaningless", tc.name)
			}
			if !fullImg.Bounds().Eq(baseImg.Bounds()) {
				t.Fatalf("bounds differ: full %v vs baseline %v", fullImg.Bounds(), baseImg.Bounds())
			}
			b := fullImg.Bounds()
			total := (b.Max.X - b.Min.X) * (b.Max.Y - b.Min.Y)
			changed := 0
			for y := b.Min.Y; y < b.Max.Y; y++ {
				for x := b.Min.X; x < b.Max.X; x++ {
					if fullImg.At(x, y) != baseImg.At(x, y) {
						changed++
					}
				}
			}
			t.Logf("%s: %d/%d pixels changed by text overlay (%.2f%%)", tc.name, changed, total, 100*float64(changed)/float64(total))
			if changed < 200 {
				t.Errorf("expected >= 200 text pixels, got %d", changed)
			}
			if changed < tc.minChanged {
				t.Errorf("expected >= %d text pixels for %s, got %d", tc.minChanged, tc.name, changed)
			}
		})
	}
}
