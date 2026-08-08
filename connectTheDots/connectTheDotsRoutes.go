package connectthedots

import (
	"context"
	"encoding/json"
	"html/template"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"

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
	r.HandleFunc("GET /connect-the-dots", func(w http.ResponseWriter, _ *http.Request) {
		if err := tmpl.ExecuteTemplate(w, "home_screen_connectTheDots.go.tmpl", nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	r.HandleFunc("GET /connect-the-dots-test", func(w http.ResponseWriter, _ *http.Request) {
		str := "auto" + strings.Repeat(" auto", 14)
		if err := tmpl.ExecuteTemplate(w, "connectTheDots.go.tmpl", map[string]any{
			"Rows": [15][15]int{}, "SizeInt": 8, "GridTemplate": str, "SizeGrid": [7]int{},
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	r.HandleFunc("GET /connect-the-dots/new_game", func(w http.ResponseWriter, _ *http.Request) {
		c4, hash := NewGameConnectTheDots(8)
		var g interfaces.Game = c4
		games[hash] = g
		metrics.GamesPlayed.WithLabelValues("connectthedots").Inc()
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(hash); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	r.HandleFunc("GET /connect-the-dots/reconnect/{gameID}/{playerHash}", func(_ http.ResponseWriter, _ *http.Request) {})

	r.HandleFunc("GET /connect-the-dots/ws/{gameID}", func(w http.ResponseWriter, req *http.Request) {
		gameID := req.PathValue("gameID")
		conn, err := upgrader.Upgrade(w, req, nil)
		if err != nil || gameID == "" {
			panic("/connect-the-dots/ws/:gameID gave an error")
		}
		gameObj := games[gameID]
		game := gameObj.(*connectTheDots)
		playerHash := IDGenerator.GenerateID(10)
		playerHashes[playerHash] = conn
		switch game.playersConnected {
		case 0:
			game.players = append(game.players, &interfaces.Player{PlayerID: playerHash, GameID: gameID, PlayerIndex: 0})
			game.playersConnected++
		case 1:
			game.players = append(game.players, &interfaces.Player{PlayerID: playerHash, GameID: gameID, PlayerIndex: 1})
			game.playersConnected++
		default:
			return
		}

		sessionID, tracked := sessions.Start(req.Context(), req, "connectthedots")
		defer func(ctx context.Context) {
			conn.Close()
			if tracked {
				sessions.End(ctx, sessionID)
			}
		}(req.Context())

		HandleWebSocketConnectTheDots(conn, inputChannel, gameObj.(*connectTheDots), false, playerHash, playerHashes, gameID)
	})

	r.HandleFunc("GET /connect-the-dots/{gameID}", func(w http.ResponseWriter, req *http.Request) {
		gameID := req.PathValue("gameID")
		if gameID == "" {
			http.Error(w, "no game matches that ID", http.StatusNotFound)
		}
		str := "auto" + strings.Repeat(" auto", 14)
		if err := tmpl.ExecuteTemplate(w, "connectTheDots.go.tmpl", map[string]any{
			"Rows": (games[gameID]).(*connectTheDots).field, "SizeInt": 8, "GridTemplate": str, "SizeGrid": [7]int{},
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

func HandleWebSocketConnectTheDots(conn *websocket.Conn,
	inputChannel *pq.PriorityChannel[interfaces.Input],
	game *connectTheDots,
	_ bool,
	hash string,
	_ map[string]*websocket.Conn, gameID string,
) {
	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			return
		}
		playerIndex := slices.IndexFunc(game.players, func(p *interfaces.Player) bool { return p.PlayerID == hash })
		if messageType == websocket.TextMessage {
			pString := string(p)
			switch pString[:2] {
			case "a:":
				coords := [2]int{}
				numStrings := strings.Split(pString[2:], "-")
				numString1, numString2 := numStrings[0], numStrings[1]
				num, _ := strconv.Atoi(numString1)
				coords[0] = num
				num, _ = strconv.Atoi(numString2)
				coords[1] = num

				ctdaei := &connectTheDotsAddEdgeInput{
					team:        playerIndex + 1,
					playerIndex: playerIndex,
					coords:      coords,
					gameID:      gameID,
				}
				if err := inputChannel.Push(ctdaei, ctdaei.Priority()); err != nil {
					slog.Error("connectTheDots: input channel push failed", "error", err)
					return
				}

			default:
				continue
			}
		}
	}
}
