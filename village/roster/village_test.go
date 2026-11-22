package roster

import (
	"os"
	"path/filepath"
	"testing"

	"botvillage/ingest"
)

func TestTitleFromVillageJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "village.json")
	if err := os.WriteFile(path, []byte(`{"sessions":{"abc":{"title":"Architect"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	titles, err := LoadTitles(path)
	if err != nil {
		t.Fatal(err)
	}
	got := Build([]ingest.LiveSession{{SessionID: "abc", Slug: "my-app"}}, titles)
	if len(got) != 1 || got[0].Title != "Architect" {
		t.Fatalf("%+v", got)
	}
}

func TestMissingVillageJSONHasNoTitles(t *testing.T) {
	titles, err := LoadTitles(filepath.Join(t.TempDir(), "absent.json"))
	if err != nil || len(titles) != 0 {
		t.Fatalf("%v %v", titles, err)
	}
}

func TestMalformedVillageJSONIsIgnored(t *testing.T) {
	path := filepath.Join(t.TempDir(), "village.json")
	if err := os.WriteFile(path, []byte(`{oops`), 0o644); err != nil {
		t.Fatal(err)
	}
	titles, err := LoadTitles(path)
	if err != nil {
		t.Fatalf("want nil error, got %v", err)
	}
	if len(titles) != 0 {
		t.Fatalf("want empty map, got %v", titles)
	}
	got := Build([]ingest.LiveSession{{SessionID: "abc", Slug: "my-app"}}, titles)
	if len(got) != 1 || got[0].Title != "" {
		t.Fatalf("%+v", got)
	}
}
