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
