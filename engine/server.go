package engine

import (
	"html/template"
	"net/http"

	connectthedots "github.com/geofpwhite/html_games_engine/connectTheDots"
	interfaces "github.com/geofpwhite/html_games_engine/interfaces"
	tictactoe "github.com/geofpwhite/html_games_engine/ticTacToe"

	connect4 "github.com/geofpwhite/html_games_engine/connect4"
	hangman "github.com/geofpwhite/html_games_engine/hangman"

	"github.com/gorilla/websocket"
)

func mod(a, b int) int {
	return a % b
}

// nastiest part of the system.
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
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if err := tmpl.ExecuteTemplate(w, "home_page.go.tmpl", nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	r.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if err := tmpl.ExecuteTemplate(w, "home_page.go.tmpl", nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	r.HandleFunc("GET /about", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://github.com/geofpwhite/html_games_engine", http.StatusMovedPermanently)
	})

	hangman.HangmanRoutes(r, &upgrader, games, playerHashes, inputChannel)
	connect4.Connect4Routes(r, &upgrader, games, playerHashes, inputChannel)
	connectthedots.ConnectTheDotsRoutes(r, &upgrader, games, playerHashes, inputChannel)
	tictactoe.TicTacToeRoutes(r, &upgrader, games, playerHashes, inputChannel)
	accounts.AccountRoutes(r, accounts.NewAccountsGamesHandler())
	r.ServeHTTP()
}
