// Package diagram provides a pure Go library for generating and rendering
// Mermaid diagrams in multiple output formats (mmd, svg, png, html, pdf, json, dot).
//
// All rendering methods write to io.Writer so output can target files, HTTP
// responses, in-memory buffers, or any other sink without additional wrappers.
//
// Repository: https://github.com/iokdigital/go-mermaid
// Specification: FRD-017-3 (github.com/iokdigital/alteryx2talend)
package diagram

import (
	"errors"
	"io"
)

// DiagramType identifies the Mermaid diagram variant.
type DiagramType string

const (
	TypeFlowchart DiagramType = "flowchart"
	TypeSequence  DiagramType = "sequence"
	TypeState     DiagramType = "state"
	TypeER        DiagramType = "er"
	TypeClass     DiagramType = "class"
	TypePie       DiagramType = "pie"
	TypeQuadrant  DiagramType = "quadrant"
	TypeGantt     DiagramType = "gantt"
	TypeMindMap   DiagramType = "mindmap"
)

// OutputFormat identifies the rendering target format.
type OutputFormat string

const (
	FormatMMD      OutputFormat = "mmd"
	FormatSVG      OutputFormat = "svg"
	FormatPNG      OutputFormat = "png"
	FormatHTML     OutputFormat = "html"
	FormatPDF      OutputFormat = "pdf"
	FormatJSON     OutputFormat = "json"
	FormatDOT      OutputFormat = "dot"
	FormatMarkdown OutputFormat = "md"
)

// Diagram is the base interface for all diagram types.
type Diagram interface {
	Type()  DiagramType
	Title() string
}

// Renderer converts a Diagram into an output format.
type Renderer interface {
	// RenderTo writes output to any io.Writer (file, buffer, HTTP response, pipe).
	RenderTo(w io.Writer, d Diagram, format OutputFormat) error

	// RenderBytes returns output as []byte.
	RenderBytes(d Diagram, format OutputFormat) ([]byte, error)

	// RenderToFile writes output to a named file, creating parent directories as needed.
	RenderToFile(path string, d Diagram, format OutputFormat) error

	// ContentType returns the MIME type for a given output format.
	ContentType(format OutputFormat) string

	// RenderMarkdown wraps mmd source in a fenced code block for Markdown/Confluence.
	RenderMarkdown(d Diagram) string
}

// Sentinel errors.
var (
	// ErrRendererNotAvailable is returned when a format/type combination is not
	// yet implemented. Inspect FallbackFormatError.FallbackFormat() for an alternative.
	ErrRendererNotAvailable = errors.New("renderer not available for this diagram type and format")

	// ErrPNGSizeLimitExceeded is returned when the estimated PNG output size would
	// exceed RenderOptions.MaxPNGBytes. FallbackFormat() returns FormatHTML.
	ErrPNGSizeLimitExceeded = errors.New("estimated PNG size exceeds configured limit")

	// ErrDuplicateNodeID is returned when a diagram is constructed with two nodes
	// sharing the same ID.
	ErrDuplicateNodeID = errors.New("duplicate node ID")
)

// FallbackFormatError wraps a sentinel error with a suggested fallback format.
// Use errors.As to unpack it at call sites.
//
//	data, err := renderer.RenderBytes(d, diagram.FormatPNG)
//	var ferr *diagram.FallbackFormatError
//	if errors.As(err, &ferr) {
//	    data, err = renderer.RenderBytes(d, ferr.FallbackFormat())
//	}
type FallbackFormatError struct {
	Err      error
	Fallback OutputFormat
}

func (e *FallbackFormatError) Error() string                { return e.Err.Error() }
func (e *FallbackFormatError) Unwrap() error                { return e.Err }
func (e *FallbackFormatError) FallbackFormat() OutputFormat { return e.Fallback }
