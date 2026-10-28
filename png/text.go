package png

import (
	"bytes"
	"encoding/xml"
	"strconv"
	"strings"
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
// and numeric.
func coordOpt(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
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
