// Package render provides the DefaultRenderer, a concrete implementation of the
// diagram.Renderer interface that dispatches to format-specific sub-packages.
//
// # Usage
//
//	import (
//	    diagram "github.com/iokdigital/go-mermaid"
//	    "github.com/iokdigital/go-mermaid/render"
//	    "github.com/iokdigital/go-mermaid/ast"
//	)
//
//	d := ast.NewFlowchart("My Diagram", ast.DirectionLR)
//	d.MustAddNode(&ast.FlowNode{ID: "A", Label: "Start", Shape: ast.ShapeRoundedRect})
//	r := render.NewRenderer(diagram.NewRenderOptions())
//	if err := r.RenderToFile("out.mmd", d, diagram.FormatMMD); err != nil {
//	    log.Fatal(err)
//	}
//
// The root diagram package is import-free; this package is the only one that
// imports both the root and all encoding sub-packages.
package render

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	diagram "github.com/iokdigital/go-mermaid"
	"github.com/iokdigital/go-mermaid/dot"
	"github.com/iokdigital/go-mermaid/json"
	"github.com/iokdigital/go-mermaid/mmd"
)

// DefaultRenderer dispatches rendering to format-specific sub-packages.
// Obtain one via NewRenderer(opts).
type DefaultRenderer struct {
	opts diagram.RenderOptions
}

// NewRenderer returns a DefaultRenderer with the given options.
func NewRenderer(opts diagram.RenderOptions) *DefaultRenderer {
	return &DefaultRenderer{opts: opts}
}

// RenderTo writes the diagram in the requested format to w.
func (r *DefaultRenderer) RenderTo(w io.Writer, d diagram.Diagram, format diagram.OutputFormat) error {
	switch format {
	case diagram.FormatMMD:
		return mmd.Encode(w, d)
	case diagram.FormatDOT:
		return dot.Encode(w, d)
	case diagram.FormatJSON:
		return json.Encode(w, d)
	case diagram.FormatMarkdown:
		return r.renderMarkdown(w, d)
	case diagram.FormatSVG, diagram.FormatPNG, diagram.FormatPDF:
		return &diagram.FallbackFormatError{
			Err:      fmt.Errorf("%w: %s (Phase 3/4)", diagram.ErrRendererNotAvailable, format),
			Fallback: diagram.FormatHTML,
		}
	case diagram.FormatHTML:
		return &diagram.FallbackFormatError{
			Err:      fmt.Errorf("%w: %s (Phase 2)", diagram.ErrRendererNotAvailable, format),
			Fallback: diagram.FormatMMD,
		}
	default:
		return fmt.Errorf("%w: %q", diagram.ErrInvalidFormat, format)
	}
}

// RenderBytes returns the diagram output as []byte.
func (r *DefaultRenderer) RenderBytes(d diagram.Diagram, format diagram.OutputFormat) ([]byte, error) {
	var buf bytes.Buffer
	if err := r.RenderTo(&buf, d, format); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// RenderToFile writes output to path, creating parent directories as needed.
func (r *DefaultRenderer) RenderToFile(path string, d diagram.Diagram, format diagram.OutputFormat) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create parent dirs for %s: %w", path, err)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file %s: %w", path, err)
	}
	defer f.Close()
	return r.RenderTo(f, d, format)
}

// ContentType returns the MIME type for a given output format.
func (r *DefaultRenderer) ContentType(format diagram.OutputFormat) string {
	switch format {
	case diagram.FormatMMD:
		return "text/x-mermaid"
	case diagram.FormatSVG:
		return "image/svg+xml"
	case diagram.FormatPNG:
		return "image/png"
	case diagram.FormatHTML:
		return "text/html; charset=utf-8"
	case diagram.FormatPDF:
		return "application/pdf"
	case diagram.FormatJSON:
		return "application/json"
	case diagram.FormatDOT:
		return "text/vnd.graphviz"
	case diagram.FormatMarkdown:
		return "text/markdown; charset=utf-8"
	default:
		return "application/octet-stream"
	}
}

// RenderMarkdown wraps mmd source in a fenced mermaid code block.
func (r *DefaultRenderer) RenderMarkdown(d diagram.Diagram) string {
	var buf bytes.Buffer
	_ = r.renderMarkdown(&buf, d)
	return buf.String()
}

func (r *DefaultRenderer) renderMarkdown(w io.Writer, d diagram.Diagram) error {
	var mmdBuf bytes.Buffer
	if err := mmd.Encode(&mmdBuf, d); err != nil {
		return err
	}
	_, err := fmt.Fprintf(w, "```mermaid\n%s```\n", mmdBuf.String())
	return err
}
