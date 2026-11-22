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
}

// AvatarPNG draws a deterministic 32x32 critter face from the session id.
func AvatarPNG(sessionID string) []byte {
	s := seed(sessionID)
	body := palette[s%uint64(len(palette))]
	accent := palette[(s>>8)%uint64(len(palette))]
	dark := color.NRGBA{0x2b, 0x21, 0x2e, 0xff}

	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	earLift := int((s >> 16) % 4)
	eyesY := 15 + int((s>>20)%2)
	mouthW := 3 + int((s>>24)%3)

	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			dx, dy := x-16, y-17
			inHead := dx*dx+dy*dy <= 100
			inEarL := (x-8)*(x-8)+(y-(10-earLift))*(y-(10-earLift)) <= 16
			inEarR := (x-24)*(x-24)+(y-(10-earLift))*(y-(10-earLift)) <= 16
			switch {
			case inEarL || inEarR:
				img.SetNRGBA(x, y, accent)
			case inHead:
				img.SetNRGBA(x, y, body)
			}
		}
	}
	for _, ex := range []int{11, 19} {
		for y := eyesY; y < eyesY+2; y++ {
			for x := ex; x < ex+2; x++ {
				img.SetNRGBA(x, y, dark)
			}
		}
	}
	for x := 16 - mouthW/2; x <= 16+mouthW/2; x++ {
		img.SetNRGBA(x, 22, dark)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}
