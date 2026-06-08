package diagram_test

import (
	"testing"

	"github.com/iokdigital/go-mermaid"
)

func TestResolutionScale(t *testing.T) {
	cases := []struct {
		res  diagram.Resolution
		want float64
	}{
		{diagram.ResolutionWeb, 1.0},
		{diagram.ResolutionScreen, 2.0},
		{diagram.ResolutionScreenHD, 3.0},
		{diagram.ResolutionPrint, 3.125},
		{diagram.ResolutionPrintHQ, 4.167},
		{diagram.Resolution("unknown"), 2.0},
	}

	for _, tc := range cases {
		got := diagram.ResolutionScale(tc.res)
		if got != tc.want {
			t.Errorf("ResolutionScale(%v) = %f, want %f", tc.res, got, tc.want)
		}
	}
}

func TestResolutionDPI(t *testing.T) {
	cases := []struct {
		res  diagram.Resolution
		want int
	}{
		{diagram.ResolutionWeb, 96},
		{diagram.ResolutionScreen, 192},
		{diagram.ResolutionScreenHD, 288},
		{diagram.ResolutionPrint, 300},
		{diagram.ResolutionPrintHQ, 400},
		{diagram.Resolution("unknown"), 192},
	}

	for _, tc := range cases {
		got := diagram.ResolutionDPI(tc.res)
		if got != tc.want {
			t.Errorf("ResolutionDPI(%v) = %d, want %d", tc.res, got, tc.want)
		}
	}
}

func TestNewRenderOptions(t *testing.T) {
	opts := diagram.NewRenderOptions()

	if opts.Resolution != diagram.ResolutionScreen {
		t.Errorf("expected Resolution %q, got %q", diagram.ResolutionScreen, opts.Resolution)
	}

	if opts.MaxPNGBytes != 10_000_000 {
		t.Errorf("expected MaxPNGBytes 10000000, got %d", opts.MaxPNGBytes)
	}

	if opts.SVGPadding != 40 {
		t.Errorf("expected SVGPadding 40, got %d", opts.SVGPadding)
	}

	if opts.SVGMaxWidth != 8000 {
		t.Errorf("expected SVGMaxWidth 8000, got %d", opts.SVGMaxWidth)
	}

	if opts.SVGMaxHeight != 6000 {
		t.Errorf("expected SVGMaxHeight 6000, got %d", opts.SVGMaxHeight)
	}

	if opts.Layout.NodeSpacingH != 60 {
		t.Errorf("expected Layout.NodeSpacingH 60, got %d", opts.Layout.NodeSpacingH)
	}

	if opts.Layout.NodeSpacingV != 40 {
		t.Errorf("expected Layout.NodeSpacingV 40, got %d", opts.Layout.NodeSpacingV)
	}

	if opts.Layout.RankSpacing != 80 {
		t.Errorf("expected Layout.RankSpacing 80, got %d", opts.Layout.RankSpacing)
	}
}

func TestDefaultLayoutOptions(t *testing.T) {
	layout := diagram.DefaultLayoutOptions()

	if layout.NodeSpacingH != 60 {
		t.Errorf("expected NodeSpacingH 60, got %d", layout.NodeSpacingH)
	}
	if layout.NodeSpacingV != 40 {
		t.Errorf("expected NodeSpacingV 40, got %d", layout.NodeSpacingV)
	}
	if layout.RankSpacing != 80 {
		t.Errorf("expected RankSpacing 80, got %d", layout.RankSpacing)
	}
}

func TestRenderOptionsSetters(t *testing.T) {
	opts := diagram.NewRenderOptions()

	opts.Scale = 1.5
	if opts.Scale != 1.5 {
		t.Errorf("expected Scale 1.5, got %f", opts.Scale)
	}

	opts.DPI = 300
	if opts.DPI != 300 {
		t.Errorf("expected DPI 300, got %d", opts.DPI)
	}

	opts.CDNOverrideURL = "https://custom.cdn.com/mermaid.js"
	if opts.CDNOverrideURL != "https://custom.cdn.com/mermaid.js" {
		t.Errorf("expected CDNOverrideURL %q, got %q",
			"https://custom.cdn.com/mermaid.js", opts.CDNOverrideURL)
	}

	opts.SVGPadding = 20
	if opts.SVGPadding != 20 {
		t.Errorf("expected SVGPadding 20, got %d", opts.SVGPadding)
	}

	opts.SVGMaxWidth = 4000
	if opts.SVGMaxWidth != 4000 {
		t.Errorf("expected SVGMaxWidth 4000, got %d", opts.SVGMaxWidth)
	}

	opts.SVGMaxHeight = 3000
	if opts.SVGMaxHeight != 3000 {
		t.Errorf("expected SVGMaxHeight 3000, got %d", opts.SVGMaxHeight)
	}
}

func TestRenderOptionsMaxPNGBytesZero(t *testing.T) {
	opts := diagram.NewRenderOptions()
	opts.MaxPNGBytes = 0

	if opts.MaxPNGBytes != 0 {
		t.Errorf("expected MaxPNGBytes 0, got %d", opts.MaxPNGBytes)
	}
}

func TestLayoutOptionsSetters(t *testing.T) {
	opts := diagram.NewRenderOptions()

	opts.Layout.NodeSpacingH = 100
	if opts.Layout.NodeSpacingH != 100 {
		t.Errorf("expected NodeSpacingH 100, got %d", opts.Layout.NodeSpacingH)
	}

	opts.Layout.NodeSpacingV = 50
	if opts.Layout.NodeSpacingV != 50 {
		t.Errorf("expected NodeSpacingV 50, got %d", opts.Layout.NodeSpacingV)
	}

	opts.Layout.RankSpacing = 120
	if opts.Layout.RankSpacing != 120 {
		t.Errorf("expected RankSpacing 120, got %d", opts.Layout.RankSpacing)
	}
}