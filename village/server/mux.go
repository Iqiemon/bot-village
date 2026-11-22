package server

import (
	"net/http"

	"botvillage/static"
)

// NewMux serves the embedded static files.
func NewMux(cfg Config) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServerFS(static.FS))
	return mux
}
