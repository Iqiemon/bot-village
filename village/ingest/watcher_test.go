package ingest

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeSession(t *testing.T, dataDir, pid, id string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dataDir, "sessions"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"pid":` + pid + `,"sessionId":"` + id + `","cwd":"/w/app","status":"busy"}`
	if err := os.WriteFile(filepath.Join(dataDir, "sessions", pid+".json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeTranscript(t *testing.T, dataDir, slug, id, body string) string {
	t.Helper()
	dir := filepath.Join(dataDir, "projects", slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, id+".jsonl")
	f, err := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(body); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return p
}

func startWatcher(t *testing.T, dir string) *Watcher {
	t.Helper()
	w := NewWatcher(WatcherConfig{DataDirs: []string{dir}, Interval: 20 * time.Millisecond, BurstWindow: 250 * time.Millisecond})
	w.Start(context.Background())
	t.Cleanup(w.Stop)
	return w
}

func next(t *testing.T, ch <-chan Event) Event {
	t.Helper()
	select {
	case ev := <-ch:
		return ev
	case <-time.After(2 * time.Second):
		t.Fatal("no event")
		return Event{}
	}
}

func collect(t *testing.T, ch <-chan Event, d time.Duration) []Event {
	t.Helper()
	var got []Event
	deadline := time.After(d)
	for {
		select {
		case ev := <-ch:
			got = append(got, ev)
		case <-deadline:
			return got
		}
	}
}

func TestBurstInsideWindowEmitsOneActivity(t *testing.T) {
	dir := t.TempDir()
	writeSession(t, dir, "1", "s-1")
	writeTranscript(t, dir, "app", "s-1", "")
	w := startWatcher(t, dir)

	if ev := next(t, w.Events()); ev.Kind != "spawn" || ev.SessionID != "s-1" {
		t.Fatalf("%+v", ev)
	}

	writeTranscript(t, dir, "app", "s-1",
		`{"type":"user"}`+"\n"+
			`{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Bash"}]}}`+"\n"+
			`{"type":"user"}`+"\n"+
			`{"type":"assistant"}`+"\n"+
			`{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Read"}]}}`+"\n")

	got := collect(t, w.Events(), 600*time.Millisecond)
	if len(got) != 1 || got[0].Kind != "activity" {
		t.Fatalf("%+v", got)
	}
	if got[0].SessionID != "s-1" || got[0].ToolName != "Read" || got[0].Role != "assistant" {
		t.Fatalf("activity should carry the last line: %+v", got[0])
	}
}

func TestSpawnFiresOncePerSession(t *testing.T) {
	dir := t.TempDir()
	writeSession(t, dir, "1", "s-1")
	w := startWatcher(t, dir)

	got := collect(t, w.Events(), 300*time.Millisecond)
	if len(got) != 1 || got[0].Kind != "spawn" || got[0].SessionID != "s-1" {
		t.Fatalf("%+v", got)
	}
}

func TestGoneEmitsForDeletedSession(t *testing.T) {
	dir := t.TempDir()
	writeSession(t, dir, "1", "s-1")
	w := startWatcher(t, dir)
	next(t, w.Events())

	if err := os.Remove(filepath.Join(dir, "sessions", "1.json")); err != nil {
		t.Fatal(err)
	}

	got := collect(t, w.Events(), 300*time.Millisecond)
	if len(got) != 1 || got[0].Kind != "gone" || got[0].SessionID != "s-1" {
		t.Fatalf("%+v", got)
	}
}

func TestTranscriptWriteIsSeenWithoutWaitingForPoll(t *testing.T) {
	dir := t.TempDir()
	writeSession(t, dir, "1", "s-1")
	writeTranscript(t, dir, "app", "s-1", "")
	const pollDisabled = time.Hour
	w := NewWatcher(WatcherConfig{DataDirs: []string{dir}, Interval: pollDisabled})
	w.Start(context.Background())
	t.Cleanup(w.Stop)
	next(t, w.Events())

	writeTranscript(t, dir, "app", "s-1", `{"type":"user"}`+"\n")

	select {
	case ev := <-w.Events():
		if ev.Kind != "activity" {
			t.Fatalf("%+v", ev)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("fsnotify did not trigger a read")
	}
}
