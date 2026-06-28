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
	inputChannel chan interfaces.Input,
) {
	r.HandleFunc("GET /whiteboard", func(w http.ResponseWriter, req *http.Request) {
		if err := tmpl.ExecuteTemplate(w, "home_screen_whiteboard.go.tmpl", nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	r.HandleFunc("GET /whiteboard/new_game", func(w http.ResponseWriter, req *http.Request) {
		wb, hash := NewWhiteboard(1000, 500)
		games[hash] = wb
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(hash); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	r.HandleFunc("GET /whiteboard/ws/{gameID}", func(w http.ResponseWriter, req *http.Request) {
		gameID := req.PathValue("gameID")
		conn, err := upgrader.Upgrade(w, req, nil)
		if err != nil || gameID == "" {
			return
		}
		gameObj, ok := games[gameID]
		if !ok {
			conn.Close() //nolint:errcheck //We don't care right now if there are errors closing the connection. we are discarding it
			return
		}
		wb, ok := gameObj.(*whiteboard)
		if !ok {
			conn.Close() //nolint:errcheck //We don't care right now if there are errors closing the connection. we are discarding it

			return
		}

		playerHash := IDGenerator.GenerateID(10)
		playerHashes[playerHash] = conn

		// Send the full current image before registering the player so the
		// output loop doesn't race with this initial write.
		if pngBytes, err := wb.encodedPNG(); err == nil {
			err2 := conn.WriteJSON(whiteboardFull{Type: "full", Png: pngBytes})
			if err2 != nil {
				conn.Close() //nolint:errcheck //We don't care right now if there are errors closing the connection. we are discarding it
				return
			}
		}

		wb.players = append(wb.players, &interfaces.Player{
			PlayerID:    playerHash,
			GameID:      gameID,
			PlayerIndex: len(wb.players),
		})

		defer conn.Close() //nolint:errcheck //We don't care right now if there are errors closing the connection. we are discarding it

		HandleWebSocketWhiteboard(conn, inputChannel, gameID)
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
	gameID string,
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
				parts := strings.Split(pString[2:], "-")
				if len(parts) < 4 {
					continue
				}
				x, errX := strconv.Atoi(parts[0])
				y, errY := strconv.Atoi(parts[1])
				clr := imgDecode(parts[2])
				radius, errR := strconv.Atoi(parts[3])
				if errX != nil || errY != nil || errR != nil {
					continue
				}

				inputChannel <- &drawInput{
					x:      x,
					y:      y,
					gameID: gameID,
					color:  clr,
					radius: radius,
				}

			default:
				continue
			}
		}
	}
}

func imgDecode(s string) color.RGBA {
	clr := color.RGBA{A: 255}
	for i := 0; i < 6; i += 2 {
		n, err := strconv.ParseUint(s[i:i+2], 16, 8)
		if err != nil {
			continue
		}
		switch i {
		case 0:
			clr.R = uint8(n)
		case 2:
			clr.G = uint8(n)
		case 4:
			clr.B = uint8(n)
		}
	}
	return clr
}
