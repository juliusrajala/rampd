package artwork

import (
	"crypto/sha256"
	"image"
	"image/color"
	"math"
	"math/rand/v2"
)

// SynthwaveCover renders a small, repeatable landscape from the playlist name.
func SynthwaveCover(name string, rows int) []string {
	if rows <= 0 {
		return nil
	}
	return SquareCover(synthwaveScene(name, 2*rows), rows)
}

type sceneColors struct {
	sky, glow, sun, stripes, grid color.RGBA
}

func synthwaveScene(name string, size int) *image.RGBA {
	rng := rand.New(rand.NewChaCha8(sha256.Sum256([]byte(name))))
	palettes := [...]sceneColors{
		{rgb(0x120b31), rgb(0xa83c93), rgb(0xffd478), rgb(0xfd417a), rgb(0xdf4fcf)},
		{rgb(0x101030), rgb(0x4045ad), rgb(0xafffe7), rgb(0x2dcec9), rgb(0x8178ff)},
		{rgb(0x20102f), rgb(0xb73e70), rgb(0xffed9d), rgb(0xf65838), rgb(0xf265a2)},
		{rgb(0x0e0c31), rgb(0x583d9c), rgb(0xffbbf3), rgb(0xbf4cf0), rgb(0x37b7e7)},
	}
	palette := palettes[rng.IntN(len(palettes))]
	horizon := size * (56 + rng.IntN(8)) / 100
	cx := float64(size) * (0.30 + 0.40*rng.Float64())
	radius := float64(size) * (0.22 + 0.05*rng.Float64())
	cy := float64(horizon) - 0.55*radius
	vanish := float64(size) * (0.38 + 0.24*rng.Float64())
	ridge := mountainRidge(rng, size)
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := range size {
		for x := range size {
			if y >= horizon {
				img.SetRGBA(x, y, gridPixel(palette, x, y-horizon, size-horizon, vanish, size))
				continue
			}
			dx, dy := float64(x)+0.5-cx, float64(y)+0.5-cy
			pixel := mix(palette.sky, palette.glow, math.Pow(float64(y)/float64(horizon), 1.8))
			if rng.IntN(28) == 0 && y < horizon/2 {
				pixel = mix(pixel, palette.sun, 0.65)
			}
			if dx*dx+dy*dy <= radius*radius && (dy < 0 || (y+1)%3 != 0) {
				pixel = mix(palette.sun, palette.stripes, (dy+radius)/(2*radius))
			}
			if y >= horizon-ridge[x] {
				pixel = mix(palette.sky, palette.glow, 0.17)
				if y == horizon-ridge[x] {
					pixel = mix(pixel, palette.glow, 0.47)
				}
			}
			img.SetRGBA(x, y, pixel)
		}
	}
	return img
}

func mountainRidge(rng *rand.Rand, size int) []int {
	ridge := make([]int, size)
	for range 5 {
		peak := float64(rng.IntN(size))
		height := float64(size) * (0.05 + 0.13*rng.Float64())
		slope := 0.28 + 0.55*rng.Float64()
		for x := range ridge {
			ridge[x] = max(ridge[x], int(math.Round(height-slope*math.Abs(float64(x)-peak))))
		}
	}
	return ridge
}

func gridPixel(p sceneColors, x, y, depth int, vanish float64, size int) color.RGBA {
	t := float64(y) / float64(max(1, depth-1))
	floor := mix(p.sky, rgb(0x2a1246), t)
	if y == 0 {
		return mix(p.glow, p.grid, 0.65)
	}
	// Rays spread toward the foreground. The horizontal lines get farther apart.
	step := float64(size) * (0.04 + 0.20*t)
	ray := (float64(x) + 0.5 - vanish) / step
	grid := math.Abs(ray-math.Round(ray))*step < 0.35
	for _, row := range []float64{0.17, 0.36, 0.63, 0.97} {
		grid = grid || y == int(math.Round(row*float64(depth-1)))
	}
	if grid {
		return mix(floor, p.grid, 0.42+0.58*t)
	}
	return floor
}

func rgb(hex uint32) color.RGBA {
	return color.RGBA{R: uint8(hex >> 16), G: uint8(hex >> 8), B: uint8(hex), A: 255}
}

func mix(a, b color.RGBA, t float64) color.RGBA {
	blend := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t) }
	return color.RGBA{R: blend(a.R, b.R), G: blend(a.G, b.G), B: blend(a.B, b.B), A: 255}
}
