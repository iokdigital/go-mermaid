// Package html encodes a diagram AST to a standalone HTML page that renders
// the diagram in a browser.
//
// # Rendering engines
//
// By default (HTMLEngine "pele") the page loads Pele
// (https://pele.run), a lightweight Mermaid-compatible renderer, and mounts
// every <pre class="mermaid"> block with pele.mount(). If the Pele CDN script
// fails to load, the page dynamically loads mermaid.js from a fallback CDN
// URL and renders with mermaid.initialize() instead.
//
// HTMLEngine "mermaid" emits the legacy mermaid.js-only page (no Pele, no
// fallback). The mmd source is identical in both modes, so pages work with
// either engine.
//
// # Offline operation
//
// No renderer bundle is embedded. If offline operation is required, point
// CDNOverrideURL (and HTMLFallbackCDNURL) at internally hosted copies.
package html

import (
	"fmt"
	"html/template"
	"io"
	"strings"

	diagram "github.com/iokdigital/go-mermaid"
	"github.com/iokdigital/go-mermaid/mmd"
)

const defaultPeleCDNURL = "https://cdn.jsdelivr.net/npm/pele@1/dist/pele.global.js"
const defaultMermaidCDNURL = "https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.min.js"

var pelePageTmpl = template.Must(template.New("diagram-pele").Parse(`<!DOCTYPE html>
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
  <script>(function () {
    function mountAll() {
      document.querySelectorAll('pre.mermaid').forEach(function (el) {
        try {
          window.pele.mount(el, el.textContent);
        } catch (e) {
          // Pele loaded but rejected this diagram: surface the error text.
          // Unlike a CDN load failure this does not fall back to mermaid.js,
          // because go-mermaid's .mmd output is Pele-validated by the
          // tests/pele differential suite.
          el.textContent = String((e && e.message) || e);
        }
      });
    }
    if (window.pele && window.pele.mount) {
      mountAll();
      return;
    }
    // Pele CDN failed: load mermaid.js as a fallback. Render explicitly via
    // mermaid.run(): startOnLoad hooks DOMContentLoaded, which has typically
    // already fired for a dynamically injected script, so initialize() alone
    // may leave the raw source on screen.
    var s = document.createElement('script');
    s.src = "{{.FallbackCDNURL}}";
    s.onload = function () {
      if (!window.mermaid) {
        return;
      }
      var m = window.mermaid;
      if (typeof m.run === 'function') {
        try {
          m.run({ querySelector: 'pre.mermaid' });
          return;
        } catch (e) {
          /* fall through to initialize */
        }
      }
      m.initialize({startOnLoad: true, theme: 'default'});
    };
    document.head.appendChild(s);
  })();</script>
</body>
</html>
`))

var mermaidPageTmpl = template.Must(template.New("diagram-mermaid").Parse(`<!DOCTYPE html>
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
	MermaidSource string // plain string: html/template HTML-escapes it in the <pre> context
	CDNURL        template.URL
	// FallbackCDNURL is embedded in a JS string context; html/template
	// applies JS-string escaping. Kept as string (not template.URL) so the
	// escaping is not bypassed. Scheme is validated by resolveCDN.
	FallbackCDNURL string
}

// Encode writes a standalone HTML page containing the diagram source in a
// <pre class="mermaid"> block plus the client-side renderer script.
//
// The mmd source is HTML-escaped in the page; renderers read
// element.textContent so they receive the original characters regardless.
// This prevents XSS through diagram titles that contain HTML-special
// characters.
//
// opts.CDNOverrideURL, if non-empty, replaces the default CDN URL for the
// selected engine and must start with "https://" or "http://"; any other
// scheme returns an error. opts.HTMLFallbackCDNURL likewise replaces the
// Pele-mode fallback mermaid.js URL.
func Encode(w io.Writer, d diagram.Diagram, opts diagram.RenderOptions) error {
	engine := opts.HTMLEngine
	if engine == "" {
		engine = diagram.HTMLEnginePele
	}

	var mmdBuf strings.Builder
	if err := mmd.Encode(&mmdBuf, d); err != nil {
		return fmt.Errorf("html: encode mmd source: %w", err)
	}

	switch engine {
	case diagram.HTMLEngineMermaid:
		cdnURL, err := resolveCDN(opts.CDNOverrideURL, defaultMermaidCDNURL)
		if err != nil {
			return err
		}
		data := templateData{
			Title:         d.Title(),
			MermaidSource: mmdBuf.String(),
			CDNURL:        template.URL(cdnURL), //nolint:gosec // scheme validated by resolveCDN
		}
		if err := mermaidPageTmpl.Execute(w, data); err != nil {
			return fmt.Errorf("html: execute template: %w", err)
		}
		return nil
	case diagram.HTMLEnginePele:
		cdnURL, err := resolveCDN(opts.CDNOverrideURL, defaultPeleCDNURL)
		if err != nil {
			return err
		}
		fallbackURL, err := resolveCDN(opts.HTMLFallbackCDNURL, defaultMermaidCDNURL)
		if err != nil {
			return fmt.Errorf("html: fallback CDN: %w", err)
		}
		data := templateData{
			Title:          d.Title(),
			MermaidSource:  mmdBuf.String(),
			CDNURL:         template.URL(cdnURL), //nolint:gosec // scheme validated by resolveCDN
			FallbackCDNURL: fallbackURL,
		}
		if err := pelePageTmpl.Execute(w, data); err != nil {
			return fmt.Errorf("html: execute template: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("html: unsupported HTMLEngine %q (want %q or %q)",
			opts.HTMLEngine, diagram.HTMLEnginePele, diagram.HTMLEngineMermaid)
	}
}

// resolveCDN returns override when non-empty, else def. A non-empty override
// must use https:// or http://.
func resolveCDN(override, def string) (string, error) {
	if override == "" {
		return def, nil
	}
	if !strings.HasPrefix(override, "https://") && !strings.HasPrefix(override, "http://") {
		return "", fmt.Errorf("CDN URL must use https:// or http:// scheme, got %q", override)
	}
	return override, nil
}
