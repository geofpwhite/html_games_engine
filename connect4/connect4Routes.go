package connect4

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"

	IDGenerator "github.com/geofpwhite/html_games_engine/IDGenerator"

	interfaces "github.com/geofpwhite/html_games_engine/interfaces"

	"github.com/gorilla/websocket"
)

func Routes(r *http.ServeMux, tmpl *template.Template, upgrader *websocket.Upgrader,
	games map[string]interfaces.Game, playerHashes map[string]*websocket.Conn, inputChannel chan interfaces.Input,
) {
	r.HandleFunc("GET /connect4", func(w http.ResponseWriter, _ *http.Request) {
		if err := tmpl.ExecuteTemplate(w, "home_screen_connect4.go.tmpl", nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	r.HandleFunc("GET /connect4/new_game", func(w http.ResponseWriter, _ *http.Request) {
		c4, hash := newGameConnect4()
		var g interfaces.Game = c4
		games[hash] = g
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(hash)
		if err != nil {
			http.Error(w, "can't send hash", http.StatusExpectationFailed)
		}
	})

	r.HandleFunc("GET /connect4/ws/{gameID}", func(w http.ResponseWriter, req *http.Request) {
		gameID := req.PathValue("gameID")
		conn, err := upgrader.Upgrade(w, req, nil)
		if err != nil || gameID == "" {
			http.Error(w, "error connecting", http.StatusInternalServerError)
			return
		}
		gameObj := games[gameID]
		game := gameObj.(*connect4)
		playerHash := IDGenerator.GenerateID(10)
		playerHashes[playerHash] = conn
		if game.playersConnected >= 2 {
			return
		}
		game.players = append(game.players, &interfaces.Player{PlayerID: playerHash, GameID: gameID, PlayerIndex: game.playersConnected})
		game.playersConnected++
		defer func() {
			conn.Close()
		}()

		for {
			x, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if x == websocket.TextMessage {
				switch string(msg) {
				case "r":
					c4i := connect4RotateInput{gameID: gameID, playerIndex: -1}
					inputChannel <- &c4i
				default:
					msgStrings := strings.Split(string(msg), ",")
					team, _ := strconv.Atoi(msgStrings[0])
					column, _ := strconv.Atoi(msgStrings[1])
					c4i := connect4InsertInput{gameID: gameID, team: team, column: column}
					inputChannel <- &c4i
				}
			} else {
				obj := make(map[string]any)
				err := json.Unmarshal(msg, &obj)
				if err != nil {
					http.Error(w, "can't unmarshall json", http.StatusInternalServerError)
				}
			}
		}
	})

	r.HandleFunc("GET /connect4/{gameID}", func(w http.ResponseWriter, req *http.Request) {
		gameID := req.PathValue("gameID")
		colors := map[string]string{
			"1": "blue",
			"2": "red",
		}
		if gameID == "" {
			return
		}
		game := (games[gameID]).(*connect4)
		slog.Log(req.Context(), slog.LevelInfo, "gameid: "+gameID, "value", fmt.Sprintf("%v", game))
		rows := make([][]string, 8)
		for i := range rows {
			rows[i] = make([]string, 8)
		}

		for i := range game.field {
			for j := range game.field[i] {
				rows[i][j] = strconv.Itoa(game.field[i][j])
			}
		}
		slices.Reverse(rows)
		if err := tmpl.ExecuteTemplate(w, "connect4.go.tmpl", map[string]any{
			"Rows":   rows,
			"Colors": colors,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}
