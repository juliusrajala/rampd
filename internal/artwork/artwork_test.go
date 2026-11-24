package artwork

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestSquareCoverCropsTheCenter(t *testing.T) {
	for _, bounds := range []image.Rectangle{image.Rect(2, 3, 6, 5), image.Rect(2, 3, 4, 7)} {
		img := image.NewRGBA(bounds)
		x, y := bounds.Min.X, bounds.Min.Y
		x += (bounds.Dx() - 2) / 2
		y += (bounds.Dy() - 2) / 2
		img.Set(x, y, color.RGBA{R: 255, A: 255})
		img.Set(x, y+1, color.RGBA{B: 255, A: 255})
		lines := SquareCover(img, 1)
		if len(lines) != 1 || ansi.StringWidth(lines[0]) != 2 {
			t.Fatalf("wrong dimensions: %q", lines)
		}
		if !strings.HasPrefix(lines[0], "\x1b[38;2;255;0;0;48;2;0;0;255m▀") {
			t.Fatalf("wrong center pixels for %v: %q", bounds, lines[0])
		}
	}
}

func TestSquareCoverSoftensCornersWithoutChangingWidth(t *testing.T) {
	lines := SquareCover(image.NewUniform(color.White), 12)
	if len(lines) != 12 {
		t.Fatalf("expected 12 rows, got %d", len(lines))
	}
	for _, line := range lines {
		if ansi.StringWidth(line) != 24 {
			t.Fatalf("expected a full-width cover row: %q", line)
		}
	}
	top, bottom := lines[0], lines[len(lines)-1]
	if ansi.Strip(top) != "▄"+strings.Repeat("▀", 22)+"▄" ||
		!strings.Contains(bottom, "\x1b[38;2;255;255;255;49m▀") ||
		strings.Contains(lines[1], ";49m") {
		t.Fatal("only the cover corners should expose the terminal background")
	}
}

func TestDecodePNG(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 2))
	var data bytes.Buffer
	if err := png.Encode(&data, img); err != nil {
		t.Fatal(err)
	}
	decoded, err := Decode(data.Bytes())
	if err != nil || decoded.Bounds() != img.Bounds() {
		t.Fatalf("bad decode: %v", err)
	}
	if _, err := Decode([]byte("not an image")); err == nil {
		t.Fatal("accepted invalid artwork")
	}
}
