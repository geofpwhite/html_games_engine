package engine

import (
	"html/template"
	"log"
	"net/http"
	"time"

	accounts "github.com/geofpwhite/html_games_engine/accounts"
	"github.com/geofpwhite/html_games_engine/accounts/cache/rediscache"
	"github.com/geofpwhite/html_games_engine/accounts/store/pgstore"
	connectthedots "github.com/geofpwhite/html_games_engine/connectTheDots"
	interfaces "github.com/geofpwhite/html_games_engine/interfaces"
	tictactoe "github.com/geofpwhite/html_games_engine/ticTacToe"

	connect4 "github.com/geofpwhite/html_games_engine/connect4"
	hangman "github.com/geofpwhite/html_games_engine/hangman"
	whiteboard "github.com/geofpwhite/html_games_engine/whiteboard"

	"github.com/gorilla/websocket"
)

func mod(a, b int) int {
	return a % b
}

func Serve(inputChannel chan interfaces.Input, games map[string]interfaces.Game, playerHashes map[string]*websocket.Conn) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	r := http.NewServeMux()
	funcMap := template.FuncMap{
		"mod": mod,
	}
	tmpl := template.Must(template.New("").Funcs(funcMap).ParseGlob("templates/*"))
	r.HandleFunc("GET /", func(w http.ResponseWriter, _ *http.Request) {
		if err := tmpl.ExecuteTemplate(w, "home_page.go.tmpl", nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	r.HandleFunc("GET /about", func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, "https://github.com/geofpwhite/html_games_engine", http.StatusMovedPermanently)
	})

	hangman.Routes(r, tmpl, &upgrader, games, playerHashes, inputChannel)
	connect4.Routes(r, tmpl, &upgrader, games, playerHashes, inputChannel)
	connectthedots.Routes(r, tmpl, &upgrader, games, playerHashes, inputChannel)
	tictactoe.Routes(r, tmpl, &upgrader, games, playerHashes, inputChannel)
	whiteboard.Routes(r, tmpl, &upgrader, games, playerHashes, inputChannel)

	accounts.AccountRoutes(r, tmpl, &upgrader, pgstore.NewStore(), rediscache.NewCache())

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}
