package roster

import (
	"strings"
	"testing"

	"botvillage/ingest"
)

func names(sessions ...ingest.LiveSession) map[string]string {
	m := map[string]string{}
	for _, v := range Build(sessions, nil) {
		m[v.SessionID] = v.Name
	}
	return m
}

func TestSameSlugNamesAreDistinctAndStable(t *testing.T) {
	a := ingest.LiveSession{SessionID: "aaaa1111", Slug: "my-app"}
	b := ingest.LiveSession{SessionID: "bbbb2222", Slug: "my-app"}
	lower := ingest.LiveSession{SessionID: "00001111", Slug: "my-app"}

	alone := names(a)[a.SessionID]
	withB := names(a, b)
	if withB[a.SessionID] != alone {
		t.Fatalf("name changed when a sibling joined: %q then %q", alone, withB[a.SessionID])
	}
	if withB[a.SessionID] == withB[b.SessionID] {
		t.Fatalf("same-cwd names collide: %q", withB[a.SessionID])
	}

	both := names(a, lower)
	if both[a.SessionID] != alone {
		t.Fatalf("name changed when a lower-id sibling joined: %q then %q", alone, both[a.SessionID])
	}
	if again := names(a)[a.SessionID]; again != both[a.SessionID] {
		t.Fatalf("name changed when a sibling left: %q then %q", both[a.SessionID], again)
	}

	if names(b, a)[a.SessionID] != withB[a.SessionID] {
		t.Fatal("name depends on scan order")
	}
}

func TestNameSuffixIsHex(t *testing.T) {
	got := Build([]ingest.LiveSession{
		{SessionID: "zzzz-9999", Slug: "my-app"},
		{SessionID: "yyyy-8888", Slug: "my-app"},
	}, nil)
	seen := map[string]bool{}
	for _, v := range got {
		suffix := strings.TrimPrefix(v.Name, "my-app-")
		if suffix == v.Name || len(suffix) != 8 {
			t.Fatalf("name = %q, want my-app-<8 hex>", v.Name)
		}
		for _, r := range suffix {
			if !strings.ContainsRune("0123456789abcdef", r) {
				t.Fatalf("suffix %q is not hex", suffix)
			}
		}
		if seen[v.Name] {
			t.Fatalf("duplicate name %q", v.Name)
		}
		seen[v.Name] = true
	}
}

func TestDistinctSlugsGetDistinctNames(t *testing.T) {
	got := Build([]ingest.LiveSession{
		{SessionID: "aaaa1111", Slug: "my-app"},
		{SessionID: "bbbb2222", Slug: "other-app"},
	}, nil)
	seeds := map[uint64]bool{}
	for _, v := range got {
		if !strings.HasPrefix(v.Name, v.Slug+"-") {
			t.Fatalf("name = %q, want prefix %q", v.Name, v.Slug+"-")
		}
		if v.Seed == 0 || seeds[v.Seed] {
			t.Fatalf("seed %d not unique for %q", v.Seed, v.Name)
		}
		seeds[v.Seed] = true
	}
}

func TestSlugShapedLikeSuffixDoesNotCollide(t *testing.T) {
	got := Build([]ingest.LiveSession{
		{SessionID: "aaaa1111", Slug: "my-app"},
		{SessionID: "bbbb2222", Slug: "my-app-a7f9"},
	}, nil)
	if got[0].Name == got[1].Name {
		t.Fatalf("names collide: %q", got[0].Name)
	}
}
