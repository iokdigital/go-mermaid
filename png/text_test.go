package png

import "testing"

func TestExtractTexts(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg">
	  <text x="100" y="50" text-anchor="middle" dominant-baseline="central" font-size="13" fill="#1e293b">Hello &amp; bye</text>
	  <text text-anchor="middle" font-size="13"><tspan x="100" y="40">one</tspan><tspan x="100" y="56">two</tspan></text>
	</svg>`
	got := extractTexts([]byte(svg))
	if len(got) != 3 {
		t.Fatalf("expected 3 text runs, got %d", len(got))
	}
	if got[0].Text != "Hello & bye" {
		t.Errorf("entity not unescaped: %q", got[0].Text)
	}
}
