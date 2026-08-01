package pong

import (
	"encoding/json"
	"html/template"
	"net/http"
)

// Routes registers the plain HTTP pages around the game: a home screen to
// create/join a match, and the play page itself. The actual gameplay runs
// over the bidi Connect/gRPC stream registered separately in engine/server.go.
func Routes(r *http.ServeMux, tmpl *template.Template, svc *Service) {
	r.HandleFunc("GET /pong", func(w http.ResponseWriter, _ *http.Request) {
		if err := tmpl.ExecuteTemplate(w, "home_screen_pong.go.tmpl", nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	r.HandleFunc("GET /pong/new_game", func(w http.ResponseWriter, _ *http.Request) {
		id := svc.Registry().Create()
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	r.HandleFunc("GET /pong/{gameID}", func(w http.ResponseWriter, req *http.Request) {
		gameID := req.PathValue("gameID")
		if gameID == "" {
			http.NotFound(w, req)
			return
		}
		if err := tmpl.ExecuteTemplate(w, "pong.go.tmpl", map[string]any{"GameID": gameID}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}
