package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"

	"botvillage/ingest"
)

func newTestServer() (*Village, *httptest.Server) {
	v := NewVillage(nil)
	mux := http.NewServeMux()
	v.routes(mux)
	srv := httptest.NewServer(mux)
	return v, srv
}

func get(t *testing.T, url string) (int, http.Header, []byte) {
	t.Helper()
	res, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return res.StatusCode, res.Header, body
}

func TestVillagersEndpointReflectsSpawnAndGone(t *testing.T) {
	v, srv := newTestServer()
	defer srv.Close()

	if code, _, body := get(t, srv.URL+"/api/villagers"); code != 200 || strings.TrimSpace(string(body)) != "[]" {
		t.Fatalf("empty roster: %d %s", code, body)
	}

	v.Apply(ingest.Event{Kind: "spawn", SessionID: "abc", Slug: "my-app"})
	code, hdr, body := get(t, srv.URL+"/api/villagers")
	if code != 200 {
		t.Fatalf("got %d", code)
	}
	if !strings.HasPrefix(hdr.Get("Content-Type"), "application/json") {
		t.Fatalf("content type %q", hdr.Get("Content-Type"))
	}
	var got []struct {
		SessionID string `json:"sessionId"`
		Name      string `json:"name"`
		Seed      uint64 `json:"seed"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].SessionID != "abc" || !strings.HasPrefix(got[0].Name, "my-app-") || got[0].Seed == 0 {
		t.Fatalf("roster after spawn: %s", body)
	}

	v.Apply(ingest.Event{Kind: "gone", SessionID: "abc"})
	if _, _, body := get(t, srv.URL+"/api/villagers"); strings.TrimSpace(string(body)) != "[]" {
		t.Fatalf("roster after gone: %s", body)
	}
}

func TestAvatarRouteIsDeterministicPNG(t *testing.T) {
	_, srv := newTestServer()
	defer srv.Close()

	code1, h1, b1 := get(t, srv.URL+"/avatars/abc")
	code2, _, b2 := get(t, srv.URL+"/avatars/abc")
	if code1 != 200 || code2 != 200 {
		t.Fatalf("codes %d %d", code1, code2)
	}
	if h1.Get("Content-Type") != "image/png" {
		t.Fatalf("content type %q", h1.Get("Content-Type"))
	}
	if !bytes.Equal(b1, b2) {
		t.Fatal("avatar changed between requests")
	}
	if !bytes.HasPrefix(b1, []byte("\x89PNG")) {
		t.Fatal("body is not a PNG")
	}
	if _, _, other := get(t, srv.URL+"/avatars/abd"); bytes.Equal(b1, other) {
		t.Fatal("different sessions share a face")
	}
}

func TestWSReceivesRosterThenActivity(t *testing.T) {
	v, srv := newTestServer()
	defer srv.Close()

	v.Apply(ingest.Event{Kind: "spawn", SessionID: "abc", Slug: "my-app"})
	c, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/ws", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	var first rosterFrame
	if err := c.ReadJSON(&first); err != nil {
		t.Fatal(err)
	}
	if first.Type != "roster" || len(first.Villagers) != 1 || first.Villagers[0].SessionID != "abc" {
		t.Fatalf("first frame %+v", first)
	}

	v.Apply(ingest.Event{Kind: "activity", SessionID: "abc", ToolName: "Bash"})
	var second activityFrame
	if err := c.ReadJSON(&second); err != nil {
		t.Fatal(err)
	}
	if second.Type != "activity" || second.SessionID != "abc" || second.Tool != "Bash" {
		t.Fatalf("second frame %+v", second)
	}

	v.Apply(ingest.Event{Kind: "gone", SessionID: "abc"})
	var third rosterFrame
	if err := c.ReadJSON(&third); err != nil {
		t.Fatal(err)
	}
	if third.Type != "roster" || len(third.Villagers) != 0 {
		t.Fatalf("third frame %+v", third)
	}
}

func TestVillagersPayloadHasNoMetrics(t *testing.T) {
	v, srv := newTestServer()
	defer srv.Close()
	v.Apply(ingest.Event{Kind: "spawn", SessionID: "abc", Slug: "my-app"})

	_, _, body := get(t, srv.URL+"/api/villagers")
	for _, key := range []string{"lines", "bytes", "tokens", "snippet", "size"} {
		if bytes.Contains(bytes.ToLower(body), []byte(key)) {
			t.Fatalf("payload leaks %q: %s", key, body)
		}
	}
}
