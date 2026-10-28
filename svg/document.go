package svg

import (
	"fmt"
	"math"
	"strings"

	diagram "github.com/iokdigital/go-mermaid"
)

// writeSVGHeader writes the XML declaration and opening <svg> tag for a
// diagram with the given natural dimensions. When the diagram exceeds
// opts.SVGMaxWidth/Height, the width/height attributes are scaled down while
// viewBox keeps the natural coordinate system, so content (including text)
// scales uniformly instead of being clipped.
//
// Only the pie and quadrant renderers use this helper. The older renderers
// keep their own inline headers (without max-size scaling) to avoid
// behaviour churn in existing golden output.
func writeSVGHeader(sb *strings.Builder, w, h float64, opts diagram.RenderOptions, title string) {
	maxW := float64(opts.SVGMaxWidth)
	if maxW <= 0 {
		maxW = 8000
	}
	maxH := float64(opts.SVGMaxHeight)
	if maxH <= 0 {
		maxH = 6000
	}

	W, H := w, h
	if W > maxW || H > maxH {
		s := math.Min(maxW/W, maxH/H)
		W, H = W*s, H*s
	}

	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	fmt.Fprintf(sb, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %.1f %.1f">`+"\n",
		int(math.Ceil(W)), int(math.Ceil(H)), w, h)

	if title != "" {
		fmt.Fprintf(sb, "  <title>%s</title>\n", xmlEscape(title))
	}
}
