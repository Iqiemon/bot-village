package server

import (
	"net/http"

	"botvillage/static"
)

// NewMux serves the embedded static files. cfg is unused until later routes
// need it.
func NewMux(cfg Config) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServerFS(static.FS))
	return mux
}
