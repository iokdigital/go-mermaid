package png

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"image"
	"image/color"
	"math"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/gomedium"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// textRun is one drawable string at an absolute SVG position.
type textRun struct {
	Text     string
	X, Y     float64
	Size     float64
	Anchor   string // start|middle|end
	Baseline string // central|""
	Fill     string // hex color, "" = default
	Bold     bool
	Italic   bool
}

// svgText mirrors a <text> element: its own attributes and direct character
// data plus any child <tspan> elements. encoding/xml unescapes entities in
// character data automatically.
type svgText struct {
	X        string     `xml:"x,attr"`
	Y        string     `xml:"y,attr"`
	Anchor   string     `xml:"text-anchor,attr"`
	Baseline string     `xml:"dominant-baseline,attr"`
	Size     string     `xml:"font-size,attr"`
	Fill     string     `xml:"fill,attr"`
	Weight   string     `xml:"font-weight,attr"`
	Style    string     `xml:"font-style,attr"`
	Body     string     `xml:",chardata"`
	Tspans   []svgTspan `xml:"tspan"`
}

// svgTspan mirrors a <tspan> child of <text>; coordinates may be absent,
// in which case the parent <text> coordinates are inherited.
type svgTspan struct {
	X    string `xml:"x,attr"`
	Y    string `xml:"y,attr"`
	Body string `xml:",chardata"`
}

// extractTexts parses <text> elements at any depth of svgData into drawable
// runs: the bare <text> character data plus one run per child <tspan>.
// Runs inherit style attributes from the parent <text>. Runs with no
// coordinates on either the <tspan> or its parent are dropped.
func extractTexts(svgData []byte) []textRun {
	dec := xml.NewDecoder(bytes.NewReader(svgData))
	var runs []textRun
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "text" {
			continue
		}
		var t svgText
		if err := dec.DecodeElement(&t, &se); err != nil {
			continue
		}
		anchor := t.Anchor
		if anchor == "" {
			anchor = "start"
		}
		style := textStyle{
			size:     parseFloat(t.Size, 13),
			anchor:   anchor,
			baseline: t.Baseline,
			fill:     t.Fill,
			bold:     t.Weight == "bold",
			italic:   t.Style == "italic",
		}
		if body := strings.TrimSpace(t.Body); body != "" {
			if x, y, ok := coords(t.X, t.Y); ok {
				runs = append(runs, textRun{
					Text: body, X: x, Y: y,
					Size: style.size, Anchor: style.anchor, Baseline: style.baseline,
					Fill: style.fill, Bold: style.bold, Italic: style.italic,
				})
			}
		}
		px, py, pok := coords(t.X, t.Y)
		for _, ts := range t.Tspans {
			body := strings.TrimSpace(ts.Body)
			if body == "" {
				continue
			}
			x, y, ok := coords(ts.X, ts.Y)
			if !ok {
				if !pok {
					continue
				}
				x, y = px, py
			}
			runs = append(runs, textRun{
				Text: body, X: x, Y: y,
				Size: style.size, Anchor: style.anchor, Baseline: style.baseline,
				Fill: style.fill, Bold: style.bold, Italic: style.italic,
			})
		}
	}
	return runs
}

// textStyle groups the inheritable style attributes of a <text> element.
type textStyle struct {
	size     float64
	anchor   string
	baseline string
	fill     string
	bold     bool
	italic   bool
}

// coords parses an (x, y) attribute pair, reporting whether both are present
// and numeric.
func coords(x, y string) (float64, float64, bool) {
	xv, ok := coordOpt(x)
	if !ok {
		return 0, 0, false
	}
	yv, ok := coordOpt(y)
	if !ok {
		return 0, 0, false
	}
	return xv, yv, true
}

// coordOpt parses one coordinate attribute, reporting whether it is present
// and a finite number. NaN/Inf parse successfully but are not drawable, so
// they are rejected here rather than wasting a face lookup downstream.
func coordOpt(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false
	}
	return v, true
}

// parseFloat parses s, returning def when s is blank or not numeric.
func parseFloat(s string, def float64) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return def
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return def
	}
	return v
}

// Embedded Go typefaces, parsed once each. Bold+italic maps to Go Medium:
// x/image ships no bold-italic Go face outside the monospace family, and
// medium reads as emphasis without inventing a synthetic oblique.
var (
	parseRegular = sync.OnceValues(func() (*opentype.Font, error) {
		return opentype.Parse(goregular.TTF)
	})
	parseBold = sync.OnceValues(func() (*opentype.Font, error) {
		return opentype.Parse(gobold.TTF)
	})
	parseItalic = sync.OnceValues(func() (*opentype.Font, error) {
		return opentype.Parse(goitalic.TTF)
	})
	parseMedium = sync.OnceValues(func() (*opentype.Font, error) {
		return opentype.Parse(gomedium.TTF)
	})
)

// faceKey identifies a cached rasterizer face. Sizes quantize to 1/64 px so
// near-identical SVG font sizes share one face.
type faceKey struct {
	size64 int64
	bold   bool
	italic bool
}

var (
	faceMu    sync.Mutex
	faceCache = map[faceKey]font.Face{}
)

// maxFaceSize caps the effective font size: SVG font sizes are ours (≤13px,
// scaled ≤4x), so anything larger comes from untrusted direct png.Encode input
// and would waste memory on a giant face.
const maxFaceSize = 1024.0

// typeface returns a screen-pixel face for size (SVG px) and style. Faces
// are cached by quantized size and style; the error is descriptive (bad size
// or corrupt embedded font data) and typeface never panics.
func typeface(size float64, bold, italic bool) (font.Face, error) {
	if math.IsNaN(size) || math.IsInf(size, 0) || size <= 0 {
		return nil, fmt.Errorf("png: invalid font size %v", size)
	}
	size = math.Min(size, maxFaceSize)
	key := faceKey{size64: int64(math.Round(size * 64)), bold: bold, italic: italic}

	faceMu.Lock()
	defer faceMu.Unlock()
	if f, ok := faceCache[key]; ok {
		return f, nil
	}

	parse, name := parseRegular, "goregular"
	switch {
	case bold && italic:
		parse, name = parseMedium, "gomedium"
	case bold:
		parse, name = parseBold, "gobold"
	case italic:
		parse, name = parseItalic, "goitalic"
	}
	ttf, err := parse()
	if err != nil {
		return nil, fmt.Errorf("png: parse %s typeface: %w", name, err)
	}
	// SVG px are CSS px at 96dpi; opentype Size is in 72dpi points.
	face, err := opentype.NewFace(ttf, &opentype.FaceOptions{
		Size:    size * 72.0 / 96.0,
		DPI:     96,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, fmt.Errorf("png: new %s face at size %v: %w", name, size, err)
	}
	faceCache[key] = face
	return face, nil
}

// defaultFill is the SVG text color used when a run carries no fill or an
// unparseable one.
var defaultFill = color.NRGBA{R: 0x1e, G: 0x29, B: 0x3b, A: 0xff}

// parseFill parses a #rgb or #rrggbb fill (case-insensitive), returning
// defaultFill for empty, non-hex, or malformed input.
func parseFill(s string) color.Color {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "#") {
		return defaultFill
	}
	h := strings.ToLower(strings.TrimSpace(s[1:]))
	var r, g, b uint64
	switch len(h) {
	case 3:
		var err error
		if r, err = strconv.ParseUint(strings.Repeat(string(h[0]), 2), 16, 8); err != nil {
			return defaultFill
		}
		if g, err = strconv.ParseUint(strings.Repeat(string(h[1]), 2), 16, 8); err != nil {
			return defaultFill
		}
		if b, err = strconv.ParseUint(strings.Repeat(string(h[2]), 2), 16, 8); err != nil {
			return defaultFill
		}
	case 6:
		var err error
		if r, err = strconv.ParseUint(h[0:2], 16, 8); err != nil {
			return defaultFill
		}
		if g, err = strconv.ParseUint(h[2:4], 16, 8); err != nil {
			return defaultFill
		}
		if b, err = strconv.ParseUint(h[4:6], 16, 8); err != nil {
			return defaultFill
		}
	default:
		return defaultFill
	}
	return color.NRGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 0xff}
}

// drawTexts overlays runs onto img, mapping SVG units to device pixels via
// scale. Each run loads its face at the scaled size (Size*scale) so glyphs
// match the rasterized shapes. Runs whose face fails to load, or that carry
// no text, are skipped without failing the render. Anchors shift x by the
// measured advance (middle: half, end: full); dominant-baseline="central"
// shifts y down by (ascent-descent)/2 so y marks the vertical center,
// otherwise y is the alphabetic baseline. Returns the number of runs drawn.
func drawTexts(img *image.RGBA, runs []textRun, scale float64) int {
	if img == nil || scale <= 0 {
		return 0
	}
	drawn := 0
	for _, run := range runs {
		if run.Text == "" {
			continue
		}
		face, err := typeface(run.Size*scale, run.Bold, run.Italic)
		if err != nil {
			continue
		}
		d := &font.Drawer{
			Dst:  img,
			Src:  image.NewUniform(parseFill(run.Fill)),
			Face: face,
		}
		x := run.X * scale
		switch run.Anchor {
		case "middle":
			x -= float64(d.MeasureString(run.Text)) / 64 / 2
		case "end":
			x -= float64(d.MeasureString(run.Text)) / 64
		}
		y := run.Y * scale
		if run.Baseline == "central" {
			m := face.Metrics()
			y += float64(m.Ascent-m.Descent) / 64 / 2
		}
		d.Dot = fixed.Point26_6{
			X: fixed.I(int(math.Round(x))),
			Y: fixed.I(int(math.Round(y))),
		}
		d.DrawString(run.Text)
		drawn++
	}
	return drawn
}

// cleanStart drops redundant namespace declarations from a copied start
// element: encoding/xml keeps xmlns in Attr while EncodeToken re-emits it
// from Name.Space, which would duplicate the attribute. Prefixed
// declarations are dropped only when the element itself is namespaced,
// matching the SVG inputs this package processes (default namespace only).
func cleanStart(t xml.StartElement) xml.StartElement {
	if t.Name.Space == "" {
		return t
	}
	kept := t.Attr[:0]
	for _, a := range t.Attr {
		if (a.Name.Space == "" && a.Name.Local == "xmlns") || a.Name.Space == "xmlns" {
			continue
		}
		kept = append(kept, a)
	}
	t.Attr = kept
	return t
}

// stripTexts returns a copy of svgData with all <text>...</text> subtrees
// removed, for the shape-only oksvg pass (oksvg warns on every <text> it
// cannot process; positions already come from extractTexts). Removal is
// depth-aware so nested content cannot leak through. If the input holds no
// <text> element, or if re-encoding fails, the input is returned unchanged.
func stripTexts(svgData []byte) []byte {
	dec := xml.NewDecoder(bytes.NewReader(svgData))
	var buf bytes.Buffer
	enc := xml.NewEncoder(&buf)
	depth := 0
	out := 0 // depth of elements copied to buf; only the root declares xmlns
	stripped := false
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch {
			case depth > 0:
				depth++
			case t.Name.Local == "text":
				depth = 1
				stripped = true
			default:
				if out > 0 {
					// Non-root elements inherit the root default
					// namespace; clearing Space avoids a redundant
					// xmlns on every element.
					t.Name.Space = ""
				}
				if err := enc.EncodeToken(cleanStart(t)); err != nil {
					return svgData
				}
				out++
			}
		case xml.EndElement:
			if depth > 0 {
				depth--
			} else {
				// Mirror the matching start tag: non-root starts are
				// written spaceless (see above), the root keeps its
				// namespace. EncodeToken errors on mismatched pairs.
				if out > 1 {
					t.Name.Space = ""
				}
				if err := enc.EncodeToken(t); err != nil {
					return svgData
				}
				out--
			}
		default:
			if depth == 0 {
				if err := enc.EncodeToken(t); err != nil {
					return svgData
				}
			}
		}
	}
	if !stripped {
		return svgData
	}
	if err := enc.Flush(); err != nil {
		return svgData
	}
	return buf.Bytes()
}
