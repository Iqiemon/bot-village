package roster

import (
	"bytes"
	"fmt"
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

func TestAvatarsAreDistinctAcrossVillage(t *testing.T) {
	seen := map[string]string{}
	for i := 0; i < 1000; i++ {
		id := fmt.Sprintf("session-%d", i)
		png := string(AvatarPNG(id))
		if prev, ok := seen[png]; ok {
			t.Fatalf("%s and %s share a face", prev, id)
		}
		seen[png] = id
	}
}
