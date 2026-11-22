package server

import (
	"net/http"

	"botvillage/static"
)

// NewMux serves the embedded static files, the roster API and the websocket.
func NewMux(cfg Config) *http.ServeMux {
	mux := http.NewServeMux()
	v := NewVillage(cfg.DataDirs)
	v.routes(mux)
	mux.Handle("/", http.FileServerFS(static.FS))
	if cfg.Events != nil {
		go v.Run(cfg.Events)
	}
	return mux
}
