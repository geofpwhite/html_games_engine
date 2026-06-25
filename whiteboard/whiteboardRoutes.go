package whiteboard

import (
	"encoding/json"
	"html/template"
	"image/color"
	"net/http"
	"strconv"
	"strings"

	IDGenerator "github.com/geofpwhite/html_games_engine/IDGenerator"
	"github.com/geofpwhite/html_games_engine/interfaces"
	"github.com/gorilla/websocket"
)

func WhiteboardRoutes(
	r *http.ServeMux,
	tmpl *template.Template,
	upgrader *websocket.Upgrader,
	games map[string]interfaces.Game,
	playerHashes map[string]*websocket.Conn,
	inputChannel chan interfaces.Input) {

	r.HandleFunc("GET /whiteboard/new_game", func(w http.ResponseWriter, req *http.Request) {
		wb, hash := NewWhiteboard(800, 600)
		games[hash] = wb
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(struct {
			GameID string `json:"gameID"`
		}{GameID: hash})
	})

	r.HandleFunc("GET /whiteboard/ws/{gameID}", func(w http.ResponseWriter, req *http.Request) {
		gameID := req.PathValue("gameID")
		conn, err := upgrader.Upgrade(w, req, nil)
		if err != nil || gameID == "" {
			return
		}
		gameObj, ok := games[gameID]
		if !ok {
			conn.Close()
			return
		}
		wb, ok := gameObj.(*whiteboard)
		if !ok {
			conn.Close()
			return
		}

		playerHash := IDGenerator.GenerateID(10)
		playerHashes[playerHash] = conn
		wb.players = append(wb.players, &interfaces.Player{
			PlayerID:    playerHash,
			GameID:      gameID,
			PlayerIndex: len(wb.players),
		})

		defer conn.Close()
		HandleWebSocketWhiteboard(conn, inputChannel, wb, false, playerHash, playerHashes, gameID)
	})

	r.HandleFunc("GET /whiteboard/{gameID}", func(w http.ResponseWriter, req *http.Request) {
		gameID := req.PathValue("gameID")
		if gameID == "" {
			return
		}
		if err := tmpl.ExecuteTemplate(w, "whiteboard.go.tmpl", nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

func HandleWebSocketWhiteboard(conn *websocket.Conn,
	inputChannel chan interfaces.Input,
	game *whiteboard,
	reconnect bool,
	hash string,
	playerHashes map[string]*websocket.Conn, gameID string,
) {
	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			return
		}
		switch messageType {
		case websocket.TextMessage:
			pString := string(p)
			switch pString[:2] {
			case "d:":
				coords := [2]int{}
				numStrings := strings.Split(pString[2:], "-")
				numString1, numString2 := numStrings[0], numStrings[1]
				num, _ := strconv.Atoi(numString1)
				coords[0] = num
				num, _ = strconv.Atoi(numString2)
				coords[1] = num
				s := numStrings[2]

				clr := imgDecode(s)

				inputChannel <- &drawInput{
					x:      coords[0],
					y:      coords[1],
					gameID: gameID,
					color:  clr,
				}

			default:
				continue
			}
		}
	}
}

func imgDecode(s string) color.RGBA {
	clr := color.RGBA{0, 0, 0, 255}
	for i := 0; i < 6; i += 2 {
		n, err := strconv.ParseInt(s[i:i+2], 16, 8)
		if err != nil {
			continue
		}
		switch i {
		case 0:
			clr.R = uint8(n)
		case 1:
			clr.G = uint8(n)
		case 2:
			clr.B = uint8(n)
		}
	}
	return clr
}
