// Package pdf produces PDF output by rasterizing SVG to PNG then embedding
// the image in an A4 landscape page using jung-kurt/gofpdf.
//
// Pipeline: SVG → PNG (via png.Encode at print resolution) → PDF.
// Limitation: output is raster-embedded, not vector. The image will pixelate
// when zoomed beyond the render resolution. Use FormatSVG for scalable output.
package pdf

import (
	"bytes"
	"fmt"
	"io"

	gofpdf "github.com/jung-kurt/gofpdf"

	diagram "github.com/iokdigital/go-mermaid"
	diapng "github.com/iokdigital/go-mermaid/png"
)

// Encode rasterizes svgData to PNG at print resolution then embeds it in an
// A4 landscape PDF written to w.
func Encode(svgData []byte, w io.Writer, title string, opts diagram.RenderOptions) error {
	// Rasterize at print resolution for good PDF quality.
	printOpts := opts
	printOpts.Resolution = diagram.ResolutionPrint

	var pngBuf bytes.Buffer
	if err := diapng.Encode(svgData, &pngBuf, printOpts); err != nil {
		return fmt.Errorf("rasterize for pdf: %w", err)
	}

	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.SetTitle(title, false)
	pdf.AddPage()

	// A4 landscape usable area: 277mm × 190mm (with 10mm margins).
	const (
		pageW  = 297.0
		pageH  = 210.0
		margin = 10.0
		maxW   = pageW - 2*margin
		maxH   = pageH - 2*margin
	)

	// Register PNG from buffer.
	imgName := "diagram.png"
	imgOpts := gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}
	pdf.RegisterImageOptionsReader(imgName, imgOpts, &pngBuf)
	info := pdf.GetImageInfo(imgName)
	if info == nil {
		return fmt.Errorf("pdf: failed to register PNG image")
	}

	// Scale to fit within the usable area, preserving aspect ratio.
	imgW := info.Width()
	imgH := info.Height()
	var scale float64
	if imgW/maxW > imgH/maxH {
		scale = maxW / imgW
	} else {
		scale = maxH / imgH
	}
	drawW := imgW * scale
	drawH := imgH * scale

	// Centre on the page.
	x := margin + (maxW-drawW)/2
	y := margin + (maxH-drawH)/2

	pdf.ImageOptions(imgName, x, y, drawW, drawH, false, imgOpts, 0, "")

	return pdf.Output(w)
}
