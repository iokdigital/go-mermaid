package png_test

import (
	"bytes"
	"errors"
	"testing"

	diagram "github.com/iokdigital/go-mermaid"
	"github.com/iokdigital/go-mermaid/ast"
	diapng "github.com/iokdigital/go-mermaid/png"
	"github.com/iokdigital/go-mermaid/svg"
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
