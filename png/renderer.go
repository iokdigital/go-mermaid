// Package png rasterizes SVG output from the svg package into PNG images using
// oksvg + rasterx (pure Go, no CGo, no external binaries).
//
// Known limitation: oksvg does not support <text> or <tspan> elements, so PNG
// output contains shapes and edges but no text labels. Use FormatSVG when
// labelled output is required.
package png

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	stdpng "image/png"
	"io"
	"math"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"

	diagram "github.com/iokdigital/go-mermaid"
)

// Encode rasterizes svgData to PNG at the resolution specified in opts.
// Returns ErrPNGSizeLimitExceeded (wrapped in FallbackFormatError) if the
// estimated output size exceeds opts.MaxPNGBytes.
func Encode(svgData []byte, w io.Writer, opts diagram.RenderOptions) error {
	scale := resolveScale(opts)

	icon, err := oksvg.ReadIconStream(bytes.NewReader(svgData), oksvg.WarnErrorMode)
	if err != nil {
		return fmt.Errorf("parse svg for png: %w", err)
	}

	width := int(math.Ceil(icon.ViewBox.W * scale))
	height := int(math.Ceil(icon.ViewBox.H * scale))
	if width <= 0 {
		width = 1
	}
	if height <= 0 {
		height = 1
	}

	if opts.MaxPNGBytes > 0 {
		estimated := int64(width) * int64(height) * 4 // RGBA bytes pre-compression
		if estimated > opts.MaxPNGBytes {
			return &diagram.FallbackFormatError{
				Err:      fmt.Errorf("%w: estimated %d bytes > limit %d", diagram.ErrPNGSizeLimitExceeded, estimated, opts.MaxPNGBytes),
				Fallback: diagram.FormatHTML,
			}
		}
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

	scanner := rasterx.NewScannerGV(width, height, img, img.Bounds())
	rast := rasterx.NewDasher(width, height, scanner)
	icon.SetTarget(0, 0, float64(width), float64(height))
	icon.Draw(rast, 1.0)

	return stdpng.Encode(w, img)
}

func resolveScale(opts diagram.RenderOptions) float64 {
	if opts.Resolution != "" {
		return diagram.ResolutionScale(opts.Resolution)
	}
	if opts.Scale > 0 {
		return opts.Scale
	}
	return diagram.ResolutionScale(diagram.ResolutionScreen)
}
