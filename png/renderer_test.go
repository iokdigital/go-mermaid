package png_test

import (
	"bytes"
	"errors"
	"image"
	"testing"

	diagram "github.com/iokdigital/go-mermaid"
	"github.com/iokdigital/go-mermaid/ast"
	diapng "github.com/iokdigital/go-mermaid/png"
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
