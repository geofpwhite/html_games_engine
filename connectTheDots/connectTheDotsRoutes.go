package connectthedots

import (
	"encoding/json"
	"html/template"
	"net/http"
	"slices"
	"strconv"
	"strings"

	IDGenerator "github.com/geofpwhite/html_games_engine/IDGenerator"
	interfaces "github.com/geofpwhite/html_games_engine/interfaces"

	"github.com/gorilla/websocket"
)

func ConnectTheDotsRoutes(r *http.ServeMux, tmpl *template.Template, upgrader *websocket.Upgrader, games map[string]interfaces.Game, playerHashes map[string]*websocket.Conn, inputChannel chan interfaces.Input) {
	r.HandleFunc("GET /connect-the-dots", func(w http.ResponseWriter, req *http.Request) {
		if err := tmpl.ExecuteTemplate(w, "home_screen_connectTheDots.go.tmpl", nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	r.HandleFunc("GET /connect-the-dots-test", func(w http.ResponseWriter, req *http.Request) {
		str := "auto"
		for i := 0; i < 14; i++ {
			str += " auto"
		}
		if err := tmpl.ExecuteTemplate(w, "connectTheDots.go.tmpl", map[string]any{"Rows": [15][15]int{}, "SizeInt": 8, "GridTemplate": str, "SizeGrid": [7]int{}}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	r.HandleFunc("GET /connect-the-dots/new_game", func(w http.ResponseWriter, req *http.Request) {
		c4, hash := NewGameConnectTheDots(8)
		var g interfaces.Game = c4
		games[hash] = g
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(hash); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	r.HandleFunc("GET /connect-the-dots/reconnect/{gameID}/{playerHash}", func(w http.ResponseWriter, req *http.Request) {})

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
		if game.playersConnected >= 2 {
			return
		} else if game.playersConnected == 1 {
			game.players = append(game.players, &interfaces.Player{PlayerID: playerHash, GameID: gameID, PlayerIndex: 1})
			game.playersConnected++
		} else if game.playersConnected == 0 {
			game.players = append(game.players, &interfaces.Player{PlayerID: playerHash, GameID: gameID, PlayerIndex: 0})
			game.playersConnected++
		}
		defer conn.Close() //nolint:errcheck

		HandleWebSocketConnectTheDots(conn, inputChannel, gameObj.(*connectTheDots), false, playerHash, playerHashes, gameID)
	})

	r.HandleFunc("GET /connect-the-dots/{gameID}", func(w http.ResponseWriter, req *http.Request) {
		gameID := req.PathValue("gameID")
		if gameID == "" {
			http.Error(w, "no game matches that ID", http.StatusNotFound)
		}
		str := "auto"
		for i := 0; i < 14; i++ {
			str += " auto"
		}
		if err := tmpl.ExecuteTemplate(w, "connectTheDots.go.tmpl", map[string]any{"Rows": (games[gameID]).(*connectTheDots).field, "SizeInt": 8, "GridTemplate": str, "SizeGrid": [7]int{}}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

func HandleWebSocketConnectTheDots(conn *websocket.Conn,
	inputChannel chan interfaces.Input,
	game *connectTheDots,
	reconnect bool,
	hash string,
	playerHashes map[string]*websocket.Conn, gameID string,
) {
	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			return
		}
		playerIndex := slices.IndexFunc(game.players, func(p *interfaces.Player) bool { return p.PlayerID == hash })
		switch messageType {
		case websocket.TextMessage:
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
				inputChannel <- ctdaei

			default:
				continue
			}
		}
	}
}
