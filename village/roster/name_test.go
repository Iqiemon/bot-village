package roster

import (
	"strings"
	"testing"

	"botvillage/ingest"
)

func TestSameSlugGetsDistinctStableNames(t *testing.T) {
	a := ingest.LiveSession{SessionID: "aaaa1111", Slug: "my-app"}
	b := ingest.LiveSession{SessionID: "bbbb2222", Slug: "my-app"}

	first := Build([]ingest.LiveSession{a, b}, nil)
	second := Build([]ingest.LiveSession{b, a}, nil)

	names := map[string]string{}
	for _, v := range first {
		names[v.SessionID] = v.Name
	}
	if names["aaaa1111"] == names["bbbb2222"] {
		t.Fatalf("names collide: %q", names["aaaa1111"])
	}
	if names["aaaa1111"] != "my-app" {
		t.Fatalf("first name = %q, want my-app", names["aaaa1111"])
	}
	if !strings.HasPrefix(names["bbbb2222"], "my-app-") || len(names["bbbb2222"]) != len("my-app-")+4 {
		t.Fatalf("second name = %q, want my-app-<4 hex>", names["bbbb2222"])
	}
	for _, v := range second {
		if names[v.SessionID] != v.Name {
			t.Fatalf("name changed on repeat: %q then %q", names[v.SessionID], v.Name)
		}
	}
}

func TestDistinctSlugsKeepPlainNames(t *testing.T) {
	got := Build([]ingest.LiveSession{
		{SessionID: "aaaa1111", Slug: "my-app"},
		{SessionID: "bbbb2222", Slug: "other-app"},
	}, nil)
	seeds := map[uint64]bool{}
	for _, v := range got {
		if v.Name != v.Slug {
			t.Fatalf("name = %q, want slug %q", v.Name, v.Slug)
		}
		if v.Seed == 0 || seeds[v.Seed] {
			t.Fatalf("seed %d not unique for %q", v.Seed, v.Name)
		}
		seeds[v.Seed] = true
	}
}

func TestCollisionSuffixIsHex(t *testing.T) {
	got := Build([]ingest.LiveSession{
		{SessionID: "zzzz-9999", Slug: "my-app"},
		{SessionID: "yyyy-8888", Slug: "my-app"},
	}, nil)
	plain, suffixed := 0, 0
	for _, v := range got {
		if v.Name == "my-app" {
			plain++
			continue
		}
		suffixed++
		suffix := strings.TrimPrefix(v.Name, "my-app-")
		if len(suffix) != 4 {
			t.Fatalf("name = %q, want my-app-<4 hex>", v.Name)
		}
		for _, r := range suffix {
			if !strings.ContainsRune("0123456789abcdef", r) {
				t.Fatalf("suffix %q is not hex", suffix)
			}
		}
	}
	if plain != 1 || suffixed != 1 {
		t.Fatalf("plain %d suffixed %d", plain, suffixed)
	}
}
