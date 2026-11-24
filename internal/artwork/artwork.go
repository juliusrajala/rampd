// Package artwork converts album images to terminal half-blocks.
package artwork

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"strings"
)

const maxPixels = 16 << 20

func Decode(data []byte) (image.Image, error) {
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if config.Width <= 0 || config.Height <= 0 || config.Width > maxPixels/config.Height {
		return nil, errors.New("album image is too large")
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	return img, err
}

// SquareCover center-crops an image and renders it with half-blocks. Each
// terminal cell holds two pixels. The result is twice as many columns as rows.
// At display size, one transparent pixel at each corner softens the outline.
func SquareCover(img image.Image, rows int) []string {
	bounds := img.Bounds()
	if bounds.Empty() || rows <= 0 {
		return nil
	}
	side := min(bounds.Dx(), bounds.Dy())
	x0 := bounds.Min.X + (bounds.Dx()-side)/2
	y0 := bounds.Min.Y + (bounds.Dy()-side)/2
	size := rows * 2
	lines := make([]string, 0, rows)
	for y := 0; y < size; y += 2 {
		var line strings.Builder
		for x := 0; x < size; x++ {
			px := x0 + x*side/size
			py := y0 + y*side/size
			bottom := y0 + (y+1)*side/size
			pr, pg, pb, _ := img.At(px, py).RGBA()
			br, bg, bb, _ := img.At(px, bottom).RGBA()
			edge := size >= 6 && (x == 0 || x == size-1)
			switch {
			case edge && y == 0:
				fmt.Fprintf(&line, "\x1b[38;2;%d;%d;%d;49m▄", br>>8, bg>>8, bb>>8)
			case edge && y == size-2:
				fmt.Fprintf(&line, "\x1b[38;2;%d;%d;%d;49m▀", pr>>8, pg>>8, pb>>8)
			default:
				fmt.Fprintf(&line, "\x1b[38;2;%d;%d;%d;48;2;%d;%d;%dm▀",
					pr>>8, pg>>8, pb>>8, br>>8, bg>>8, bb>>8)
			}
		}
		line.WriteString("\x1b[0m")
		lines = append(lines, line.String())
	}
	return lines
}
