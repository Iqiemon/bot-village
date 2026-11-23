package demo

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"botvillage/ingest"
)

func TestSeedCreatesSessionsAndAppends(t *testing.T) {
	old := tick
	tick = func() time.Duration { return 20 * time.Millisecond }
	defer func() { tick = old }()

	dir := t.TempDir()
	stop, err := Seed(dir, 3)
	if err != nil {
		t.Fatal(err)
	}
	defer stop()

	s, err := ingest.ScanSessions(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(s) != 3 {
		t.Fatalf("got %d sessions, want 3", len(s))
	}
	var path string
	for _, sess := range s {
		if sess.SessionID == "" || sess.Slug == "" {
			t.Fatalf("session not usable: %+v", sess)
		}
		if !hasTranscript(t, dir, sess.SessionID) {
			t.Fatalf("no transcript for %s", sess.SessionID)
		}
		if path == "" {
			path = transcriptPath(t, dir, sess.SessionID)
		}
	}

	before := size(t, path)
	deadline := time.Now().Add(8 * time.Second)
	for size(t, path) <= before {
		if time.Now().After(deadline) {
			t.Fatal("transcript never grew")
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func TestTickStaysBetweenTwoAndSixSeconds(t *testing.T) {
	for i := 0; i < 50; i++ {
		if d := tick(); d < 2*time.Second || d > 6*time.Second {
			t.Fatalf("tick %v outside 2-6s", d)
		}
	}
}

func size(t *testing.T, path string) int64 {
	t.Helper()
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return fi.Size()
}

func transcriptPath(t *testing.T, dir, id string) string {
	t.Helper()
	m, _ := filepath.Glob(filepath.Join(dir, "projects", "*", id+".jsonl"))
	if len(m) == 0 {
		t.Fatalf("no transcript for %s", id)
	}
	return m[0]
}

func hasTranscript(t *testing.T, dir, id string) bool {
	t.Helper()
	m, _ := filepath.Glob(filepath.Join(dir, "projects", "*", id+".jsonl"))
	if len(m) == 0 {
		return false
	}
	fi, err := os.Stat(m[0])
	return err == nil && fi.Size() > 0
}
