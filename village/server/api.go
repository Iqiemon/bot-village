package server

import (
	"encoding/json"
	"net/http"

	"botvillage/roster"
)

func (v *Village) routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/villagers", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(v.state.Snapshot()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	mux.HandleFunc("GET /avatars/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.Write(roster.AvatarPNG(r.PathValue("id")))
	})
	mux.HandleFunc("GET /ws", v.serveWS)
}
