package tictactoe

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"

	IDGenerator "github.com/geofpwhite/html_games_engine/IDGenerator"
	"github.com/geofpwhite/html_games_engine/accounts/gamesession"
	interfaces "github.com/geofpwhite/html_games_engine/interfaces"
	"github.com/geofpwhite/html_games_engine/metrics"

	"github.com/geofpwhite/pq"
	"github.com/gorilla/websocket"
)

func Routes(r *http.ServeMux, tmpl *template.Template, upgrader *websocket.Upgrader,
	games map[string]interfaces.Game, playerHashes map[string]*websocket.Conn, inputChannel *pq.PriorityChannel[interfaces.Input],
	sessions *gamesession.Tracker,
) {
	r.HandleFunc("GET /tictactoe", func(w http.ResponseWriter, _ *http.Request) {
		if err := tmpl.ExecuteTemplate(w, "home_screen_tictactoe.go.tmpl", nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	r.HandleFunc("GET /tictactoe/new_game", func(w http.ResponseWriter, _ *http.Request) {
		gState, hash := NewGameTicTacToe()
		var game interfaces.Game = gState
		games[hash] = game
		metrics.GamesPlayed.WithLabelValues("tictactoe").Inc()
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(struct {
			GameID string `json:"gameID"`
			Team   int    `json:"team"`
		}{GameID: hash, Team: 1}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	r.HandleFunc("GET /tictactoe/ws/{gameID}", func(w http.ResponseWriter, req *http.Request) {
		gameID := req.PathValue("gameID")
		conn, err := upgrader.Upgrade(w, req, nil)
		if err != nil || gameID == "" {
			return
		}
		sessionID, tracked := sessions.Start(req.Context(), req, "tictactoe")
		handleWebSocketTicTacToe(conn, inputChannel, games[gameID], false, playerHashes, gameID)
		if tracked {
			sessions.End(req.Context(), sessionID)
		}
	})
	r.HandleFunc("GET /tictactoe/reconnect/{playerHash}/{gameID}", func(_ http.ResponseWriter, _ *http.Request) {})
	r.HandleFunc("GET /tictactoe/{gameID}", func(w http.ResponseWriter, req *http.Request) {
		gameID := req.PathValue("gameID")
		if gameID == "" {
			return
		}
		if err := tmpl.ExecuteTemplate(w, "tictactoe.go.tmpl", map[string]any{"Rows": (games[gameID]).(*ticTacToe).field}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

func handleWebSocketTicTacToe(conn *websocket.Conn,
	inputChannel *pq.PriorityChannel[interfaces.Input],
	gameObj interfaces.Game,
	reconnect bool,
	playerHashes map[string]*websocket.Conn,
	gameID string,
) {
	var hash string
	gState, ok := gameObj.(*ticTacToe)
	if !ok {
		return
	}
	var playerIndex int

	if !reconnect {
		if gState.playersSize > 1 {
			return
		}
		playerIndex = gState.playersSize
		hash = IDGenerator.GenerateID(10)
		newPlayer := interfaces.Player{Username: "Player " + strconv.Itoa(playerIndex), PlayerID: hash}
		playerIndex = gState.newPlayer(newPlayer)
		playerHashes[hash] = conn
	}
	defer conn.Close()

	ui := &moveInput{gameID: gameID, playerIndex: playerIndex, team: playerIndex + 1}
	for {
		err := conn.ReadJSON(ui)
		if err != nil {
			slog.Error("Error reading JSON from WebSocket connection:", "error", err)
			return
		}
		fmt.Println(ui)
		if err := inputChannel.Push(ui, ui.Priority()); err != nil {
			slog.Error("Error pushing to inputChannel:", "error", err)
			return
		}
	}
}
