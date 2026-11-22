package ingest

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestScanReadsSessionsAndSkipsGarbage(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "sessions/4242.json"),
		`{"pid":4242,"sessionId":"s-1","cwd":"/home/me/My App","status":"busy","updatedAt":1795357608275,"messagingSocketPath":"/tmp/s-1.sock"}`)
	write(t, filepath.Join(dir, "sessions/9.json"), `{oops`)
	got, err := ScanSessions(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 session, got %+v", got)
	}
	s := got[0]
	if s.SessionID != "s-1" || s.Status != "busy" {
		t.Fatalf("%+v", s)
	}
	if s.PID != 4242 || s.SocketPath != "/tmp/s-1.sock" {
		t.Fatalf("%+v", s)
	}
	if s.Slug != "my-app" {
		t.Fatalf("slug = %q", s.Slug)
	}
	if !s.UpdatedAt.Equal(time.UnixMilli(1795357608275)) {
		t.Fatalf("updatedAt = %v", s.UpdatedAt)
	}
}

func TestScanMissingDirIsEmptyNotError(t *testing.T) {
	got, err := ScanSessions(t.TempDir())
	if err != nil || len(got) != 0 {
		t.Fatalf("%v %v", got, err)
	}
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
