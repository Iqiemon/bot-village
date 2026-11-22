package server

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRootServesEmbeddedIndex(t *testing.T) {
	srv := httptest.NewServer(NewMux(Config{Listen: ":8040"}))
	defer srv.Close()
	res, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("got %d", res.StatusCode)
	}
	body, _ := io.ReadAll(res.Body)
	if !bytes.Contains(body, []byte("Bot Village")) {
		t.Fatalf("index missing title")
	}
}

func TestStaticAssetsServed(t *testing.T) {
	srv := httptest.NewServer(NewMux(Config{Listen: ":8040"}))
	defer srv.Close()
	for _, path := range []string{"/art.js", "/town.js", "/app.js", "/smoke.html"} {
		res, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != 200 {
			t.Errorf("%s: got %d", path, res.StatusCode)
		}
	}
}
