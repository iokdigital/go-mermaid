package png

import (
	"image"
	"image/color"
	"image/draw"
	"testing"
)

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

func TestLoadFace(t *testing.T) {
	f, err := typeface(13, false, false)
	if err != nil {
		t.Fatalf("typeface: %v", err)
	}
	if f == nil {
		t.Fatal("nil face")
	}
}

func TestDrawTextNonBlank(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 100))
	// white background
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)
	n := drawTexts(img, []textRun{{Text: "Hi", X: 100, Y: 50, Size: 13, Anchor: "middle", Baseline: "central", Fill: "#1e293b"}}, 1.0)
	if n == 0 {
		t.Fatal("expected pixels drawn")
	}
}

// TestDrawTextScale checks that scale maps SVG coords to device pixels:
// the same run drawn at scale=2.0 must land at ~2x the pixel columns of
// scale=1.0.
func TestDrawTextScale(t *testing.T) {
	mkib := func(w, h int) *image.RGBA {
		img := image.NewRGBA(image.Rect(0, 0, w, h))
		draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)
		return img
	}
	run := textRun{Text: "Hi", X: 50, Y: 30, Size: 13, Anchor: "start", Fill: "#1e293b"}
	img1 := mkib(200, 100)
	img2 := mkib(400, 200)
	if n := drawTexts(img1, []textRun{run}, 1.0); n != 1 {
		t.Fatalf("scale=1.0: expected 1 run drawn, got %d", n)
	}
	if n := drawTexts(img2, []textRun{run}, 2.0); n != 1 {
		t.Fatalf("scale=2.0: expected 1 run drawn, got %d", n)
	}
	minX := func(img *image.RGBA) int {
		out := img.Bounds().Max.X
		for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
			for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
				o := img.PixOffset(x, y)
				if img.Pix[o] != 0xff || img.Pix[o+1] != 0xff || img.Pix[o+2] != 0xff {
					if x < out {
						out = x
					}
				}
			}
		}
		return out
	}
	x1, x2 := minX(img1), minX(img2)
	if x1 >= img1.Bounds().Max.X {
		t.Fatal("scale=1.0 drew no non-white pixels")
	}
	if x2 >= img2.Bounds().Max.X {
		t.Fatal("scale=2.0 drew no non-white pixels")
	}
	// Leftmost inked column at 2x should be ~2x the 1x column (±3px for
	// hinting/rounding).
	if d := x2 - 2*x1; d < -3 || d > 3 {
		t.Errorf("scale=2.0 minX=%d, want ~2x scale=1.0 minX=%d (delta %d)", x2, x1, d)
	}
}

// TestAnchorColumns checks that start/middle/end anchors for the same
// string at the same x produce different pixel columns: end < middle <
// start in leftmost inked column.
func TestAnchorColumns(t *testing.T) {
	leftmost := func(anchor string) int {
		img := image.NewRGBA(image.Rect(0, 0, 400, 100))
		draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)
		run := textRun{Text: "Hello", X: 200, Y: 50, Size: 13, Anchor: anchor, Fill: "#1e293b"}
		if n := drawTexts(img, []textRun{run}, 1.0); n != 1 {
			t.Fatalf("anchor %q: expected 1 run drawn, got %d", anchor, n)
		}
		out := img.Bounds().Max.X
		for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
			for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
				o := img.PixOffset(x, y)
				if img.Pix[o] != 0xff || img.Pix[o+1] != 0xff || img.Pix[o+2] != 0xff {
					if x < out {
						out = x
					}
				}
			}
		}
		return out
	}
	s, m, e := leftmost("start"), leftmost("middle"), leftmost("end")
	if !(e < m && m < s) {
		t.Errorf("expected leftmost columns end < middle < start, got end=%d middle=%d start=%d", e, m, s)
	}
}
