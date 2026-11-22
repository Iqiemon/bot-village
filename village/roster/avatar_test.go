package roster

import (
	"bytes"
	"image/png"
	"testing"
)

func TestAvatarIsDeterministic(t *testing.T) {
	if !bytes.Equal(AvatarPNG("abc"), AvatarPNG("abc")) {
		t.Fatal("not stable")
	}
	if bytes.Equal(AvatarPNG("abc"), AvatarPNG("abd")) {
		t.Fatal("collides")
	}
}

func TestAvatarIs32x32PNG(t *testing.T) {
	cfg, err := png.DecodeConfig(bytes.NewReader(AvatarPNG("abc")))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Width != 32 || cfg.Height != 32 {
		t.Fatalf("got %dx%d", cfg.Width, cfg.Height)
	}
}
