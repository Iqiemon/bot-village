package roster

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
)

var palette = []color.NRGBA{
	{0xf2, 0xb5, 0x8c, 0xff}, {0xe8, 0x8f, 0x9a, 0xff}, {0x9a, 0xd1, 0x8a, 0xff},
	{0x8f, 0xb8, 0xe8, 0xff}, {0xd9, 0xb4, 0xe8, 0xff}, {0xf2, 0xd6, 0x7a, 0xff},
	{0xf2, 0xa0, 0x6e, 0xff}, {0x7f, 0xd4, 0xc4, 0xff}, {0xc9, 0x9c, 0xd6, 0xff},
	{0xb8, 0xd9, 0x78, 0xff}, {0xe8, 0xa8, 0xb8, 0xff}, {0x9a, 0xc4, 0xd4, 0xff},
}

// AvatarPNG draws a deterministic 32x32 critter face from the session id.
func AvatarPNG(sessionID string) []byte {
	r := newSplitmix64(seed(sessionID))
	body := palette[r.n(len(palette))]
	accent := palette[r.n(len(palette))]
	dark := color.NRGBA{0x2b, 0x21, 0x2e, 0xff}
	light := color.NRGBA{0xff, 0xf4, 0xe8, 0xff}

	earLift := r.n(4)
	earR := 3 + r.n(2)
	earInner := r.n(3) == 0
	eyesY := 14 + r.n(3)
	eyeGap := 7 + r.n(3)
	eyeH := 1 + r.n(3)
	eyeW := 1 + r.n(2)
	brow := r.n(3)
	nose := r.n(3)
	mouthW := 2 + r.n(4)
	mouthY := 21 + r.n(3)
	smile := r.n(2) == 1
	cheeks := r.n(4)
	markShape := r.n(6)
	markColor := palette[r.n(len(palette))]
	cap := r.n(3) == 0

	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			dx, dy := x-16, y-17
			inHead := dx*dx+dy*dy <= 100
			inEarL := (x-8)*(x-8)+(y-(10-earLift))*(y-(10-earLift)) <= earR*earR
			inEarR := (x-24)*(x-24)+(y-(10-earLift))*(y-(10-earLift)) <= earR*earR
			switch {
			case inEarL || inEarR:
				img.SetNRGBA(x, y, accent)
			case inHead:
				img.SetNRGBA(x, y, body)
			}
		}
	}

	if earInner {
		for _, cx := range []int{8, 24} {
			img.SetNRGBA(cx, 10-earLift, dark)
			img.SetNRGBA(cx-1, 10-earLift, dark)
		}
	}
	if cap {
		for x := 8; x < 24; x++ {
			dx, dy := x-16, 7-17
			if dx*dx+dy*dy <= 100 {
				img.SetNRGBA(x, 7, accent)
				img.SetNRGBA(x, 8, accent)
			}
		}
	}

	marks := [6][][2]int{
		{{0, 0}},
		{{0, 0}, {1, 0}},
		{{0, 0}, {0, 1}},
		{{-1, 0}, {0, 0}, {1, 0}},
		{{0, 0}, {-1, 1}, {1, 1}},
		{{0, 0}, {0, 1}, {0, 2}},
	}
	for _, p := range marks[markShape] {
		img.SetNRGBA(16+p[0], 11+p[1], markColor)
	}

	for _, ex := range []int{16 - eyeGap, 16 + eyeGap - eyeW + 1} {
		for y := eyesY; y < eyesY+eyeH; y++ {
			for x := ex; x < ex+eyeW; x++ {
				img.SetNRGBA(x, y, dark)
			}
		}
		if brow == 2 {
			img.SetNRGBA(ex, eyesY-1, dark)
			img.SetNRGBA(ex+eyeW-1, eyesY-1, dark)
		}
	}

	if nose != 0 {
		img.SetNRGBA(16, mouthY-2, dark)
		if nose == 2 {
			img.SetNRGBA(15, mouthY-2, dark)
		}
	}

	for x := 16 - mouthW/2; x <= 16+mouthW/2; x++ {
		y := mouthY
		if smile && (x == 16-mouthW/2 || x == 16+mouthW/2) {
			y--
		}
		img.SetNRGBA(x, y, dark)
	}

	for i, p := range []struct{ x, y int }{{10, 19}, {21, 19}} {
		if cheeks&(1<<i) != 0 {
			img.SetNRGBA(p.x, p.y, light)
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

type splitmix64 struct{ s uint64 }

func newSplitmix64(seed uint64) *splitmix64 { return &splitmix64{s: seed} }

func (r *splitmix64) next() uint64 {
	r.s += 0x9e3779b97f4a7c15
	z := r.s
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}

func (r *splitmix64) n(k int) int { return int(r.next() % uint64(k)) }
