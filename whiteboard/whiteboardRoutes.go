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

func Routes(
	r *http.ServeMux,
	tmpl *template.Template,
	upgrader *websocket.Upgrader,
	games map[string]interfaces.Game,
	playerHashes map[string]*websocket.Conn,
	inputChannel chan interfaces.Input,
) {
	r.HandleFunc("GET /whiteboard", func(w http.ResponseWriter, _ *http.Request) {
		if err := tmpl.ExecuteTemplate(w, "home_screen_whiteboard.go.tmpl", nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	r.HandleFunc("GET /whiteboard/new_game", func(w http.ResponseWriter, req *http.Request) {
		width := queryInt(req, "w", 1000, 100, 4000)
		height := queryInt(req, "h", 500, 100, 4000)
		wb, hash := NewWhiteboard(width, height)
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

		// Send the full current image before registering the player so the
		// output loop doesn't race with this initial write.
		if pngBytes, err := wb.encodedPNG(); err == nil {
			err2 := conn.WriteJSON(whiteboardFull{Type: "full", Png: pngBytes})
			if err2 != nil {
				conn.Close()
				return
			}
		}

		wb.players = append(wb.players, &interfaces.Player{
			PlayerID:    playerHash,
			GameID:      gameID,
			PlayerIndex: len(wb.players),
		})

		defer conn.Close()

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

func queryInt(req *http.Request, key string, def, minVal, maxVal int) int {
	v, err := strconv.Atoi(req.URL.Query().Get(key))
	if err != nil || v < minVal || v > maxVal {
		return def
	}
	return v
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
		if messageType != websocket.TextMessage {
			continue
		}
		pString := string(p)
		msgType, msgData, ok := strings.Cut(pString, ":")
		if !ok {
			continue
		}

		switch msgType {
		case "d":
			parts := strings.Split(msgData, "-")
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
			inputChannel <- &drawInput{x: x, y: y, gameID: gameID, color: clr, radius: radius}

		case "c":
			inputChannel <- &clearInput{gameID: gameID}

		case "l":
			// l:x1-y1-x2-y2-color-thickness
			parts := strings.Split(msgData, "-")
			if len(parts) < 6 {
				continue
			}
			x1, e1 := strconv.Atoi(parts[0])
			y1, e2 := strconv.Atoi(parts[1])
			x2, e3 := strconv.Atoi(parts[2])
			y2, e4 := strconv.Atoi(parts[3])
			clr := imgDecode(parts[4])
			thickness, e5 := strconv.Atoi(parts[5])
			if e1 != nil || e2 != nil || e3 != nil || e4 != nil || e5 != nil {
				continue
			}
			inputChannel <- &lineInput{gameID: gameID, x1: x1, y1: y1, x2: x2, y2: y2, clr: clr, thickness: thickness * 2}

		case "r":
			// r:x1-y1-x2-y2-color-thickness-theta
			parts := strings.Split(msgData, "-")
			if len(parts) < 7 {
				continue
			}
			x1, e1 := strconv.Atoi(parts[0])
			y1, e2 := strconv.Atoi(parts[1])
			x2, e3 := strconv.Atoi(parts[2])
			y2, e4 := strconv.Atoi(parts[3])
			clr := imgDecode(parts[4])
			thickness, e5 := strconv.Atoi(parts[5])
			theta, e6 := strconv.ParseFloat(parts[6], 64)
			if e1 != nil || e2 != nil || e3 != nil || e4 != nil || e5 != nil || e6 != nil {
				continue
			}
			inputChannel <- &rectInput{gameID: gameID, x1: x1, y1: y1, x2: x2, y2: y2, clr: clr, thickness: thickness * 2, thetaDeg: theta}

		case "ci":
			// ci:x-y-radius-color-filled
			parts := strings.Split(msgData, "-")
			if len(parts) < 5 {
				continue
			}
			x, e1 := strconv.Atoi(parts[0])
			y, e2 := strconv.Atoi(parts[1])
			radius, e3 := strconv.Atoi(parts[2])
			clr := imgDecode(parts[3])
			filled := parts[4] == "1"
			if e1 != nil || e2 != nil || e3 != nil {
				continue
			}
			inputChannel <- &circleInput{gameID: gameID, x: x, y: y, radius: radius, clr: clr, filled: filled}
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
