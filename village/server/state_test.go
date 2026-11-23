package server

import (
	"testing"

	"botvillage/ingest"
)

func TestSnapshotIsStableAndOrdered(t *testing.T) {
	s := NewState(map[string]string{"b": "Baker"})
	s.Apply(ingest.Event{Kind: "spawn", SessionID: "b", Slug: "zzz"})
	s.Apply(ingest.Event{Kind: "spawn", SessionID: "a", Slug: "aaa"})
	s.Apply(ingest.Event{Kind: "spawn", SessionID: "a", Slug: "aaa"})

	got := s.Snapshot()
	if len(got) != 2 {
		t.Fatalf("spawn twice should replace, got %d", len(got))
	}
	if got[0].SessionID != "a" || got[1].SessionID != "b" {
		t.Fatalf("not sorted by name: %+v", got)
	}
	if got[1].Title != "Baker" {
		t.Fatalf("title lost: %+v", got[1])
	}
}

func TestActivityDoesNotChangeRoster(t *testing.T) {
	s := NewState(nil)
	s.Apply(ingest.Event{Kind: "activity", SessionID: "ghost", ToolName: "Bash"})
	if len(s.Snapshot()) != 0 {
		t.Fatal("activity invented a villager")
	}
}
