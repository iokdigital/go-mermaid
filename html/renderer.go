// Package html encodes a diagram AST to a standalone HTML page that uses
// the mermaid.js CDN script to render the diagram in a browser.
//
// # Online CDN mode only
//
// This package emits a <script src="..."> tag pointing at the jsdelivr CDN (or
// a caller-supplied override). No mermaid.js bundle is embedded. If offline
// operation is required the consumer should point CDNOverrideURL at an
// internally hosted copy.
package html

import (
	"fmt"
	"html/template"
	"io"
	"strings"

	diagram "github.com/iokdigital/go-mermaid"
	"github.com/iokdigital/go-mermaid/mmd"
)

const defaultCDNURL = "https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.min.js"

var pageTmpl = template.Must(template.New("diagram").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{.Title}}</title>
  <script src="{{.CDNURL}}"></script>
</head>
<body>
  <pre class="mermaid">
{{.MermaidSource}}  </pre>
  <script>mermaid.initialize({startOnLoad:true, theme:'default'});</script>
</body>
</html>
`))

type templateData struct {
	Title         string
	MermaidSource string     // plain string: html/template HTML-escapes it in the <pre> context
	CDNURL        template.URL
}

// Encode writes a standalone HTML page containing the mermaid.js CDN script
// and the diagram source in a <pre class="mermaid"> block.
//
// The mmd source is HTML-escaped in the page; mermaid.js reads element.textContent
// so it receives the original characters regardless. This prevents XSS through
// diagram titles that contain HTML-special characters.
//
// opts.CDNOverrideURL, if non-empty, replaces the default jsdelivr CDN URL and
// must start with "https://" or "http://"; any other scheme returns an error.
func Encode(w io.Writer, d diagram.Diagram, opts diagram.RenderOptions) error {
	cdnURL, err := resolveCDN(opts.CDNOverrideURL)
	if err != nil {
		return err
	}

	var mmdBuf strings.Builder
	if err := mmd.Encode(&mmdBuf, d); err != nil {
		return fmt.Errorf("html: encode mmd source: %w", err)
	}

	data := templateData{
		Title:         d.Title(),
		MermaidSource: mmdBuf.String(),
		CDNURL:        template.URL(cdnURL), //nolint:gosec // scheme validated by resolveCDN
	}
	if err := pageTmpl.Execute(w, data); err != nil {
		return fmt.Errorf("html: execute template: %w", err)
	}
	return nil
}

// resolveCDN returns the CDN URL to use. If override is empty the default
// jsdelivr URL is returned. A non-empty override must use https:// or http://.
func resolveCDN(override string) (string, error) {
	if override == "" {
		return defaultCDNURL, nil
	}
	if !strings.HasPrefix(override, "https://") && !strings.HasPrefix(override, "http://") {
		return "", fmt.Errorf("CDNOverrideURL must use https:// or http:// scheme, got %q", override)
	}
	return override, nil
}
