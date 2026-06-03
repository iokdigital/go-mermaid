// Package svg renders Mermaid AST diagrams to standalone SVG 1.1.
//
// Supported diagram types: FlowchartDiagram, StateDiagram, ERDiagram, ClassDiagram,
// SequenceDiagram.
//
// The SVG is self-contained (no external refs, no CSS imports) and uses flat fills
// compatible with the oksvg rasterizer used in Phase 4 PNG output.
package svg

import (
	"fmt"
	"io"
	"strings"

	diagram "github.com/iokdigital/go-mermaid"
	"github.com/iokdigital/go-mermaid/ast"
)

// Encode writes d as an SVG document to w.
// Supported types: *ast.FlowchartDiagram, *ast.StateDiagram, *ast.ERDiagram,
// *ast.ClassDiagram, *ast.SequenceDiagram.
// Unrecognised diagram types return FallbackFormatError with Fallback: FormatHTML.
func Encode(w io.Writer, d diagram.Diagram, opts diagram.RenderOptions) error {
	switch v := d.(type) {
	case *ast.StateDiagram:
		return encodeState(w, v, opts)
	case *ast.ERDiagram:
		return encodeER(w, v, opts)
	case *ast.ClassDiagram:
		return encodeClass(w, v, opts)
	case *ast.SequenceDiagram:
		return encodeSequence(w, v, opts)
	}

	fc, ok := d.(*ast.FlowchartDiagram)
	if !ok {
		return &diagram.FallbackFormatError{
			Err:      fmt.Errorf("%w: SVG renderer does not support %T", diagram.ErrRendererNotAvailable, d),
			Fallback: diagram.FormatHTML,
		}
	}

	padding := float64(opts.SVGPadding)
	if padding <= 0 {
		padding = 40
	}
	maxW := float64(opts.SVGMaxWidth)
	if maxW <= 0 {
		maxW = 8000
	}
	maxH := float64(opts.SVGMaxHeight)
	if maxH <= 0 {
		maxH = 6000
	}

	lr := runLayout(fc, opts.Layout, padding, maxW, maxH)

	W := int(lr.width)
	H := int(lr.height)

	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	fmt.Fprintf(&sb, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`+"\n", W, H, W, H)

	if fc.Title() != "" {
		fmt.Fprintf(&sb, "  <title>%s</title>\n", xmlEscape(fc.Title()))
	}

	// Edges drawn first so nodes render on top.
	sb.WriteString(`  <g id="edges">` + "\n")
	for _, e := range fc.Edges() {
		svg := edgeSVG(e, lr, fc.Direction())
		if svg != "" {
			sb.WriteString("    ")
			sb.WriteString(svg)
			sb.WriteString("\n")
		}
	}
	sb.WriteString("  </g>\n")

	sb.WriteString(`  <g id="nodes">` + "\n")
	for _, n := range fc.Nodes() {
		nl, ok := lr.nodes[n.ID]
		if !ok {
			continue
		}
		sb.WriteString("    ")
		sb.WriteString(nodeGroupSVG(n, nl.x, nl.y))
		sb.WriteString("\n")
	}
	sb.WriteString("  </g>\n")

	sb.WriteString("</svg>\n")

	_, err := io.WriteString(w, sb.String())
	return err
}
