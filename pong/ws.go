package pong

import (
	"time"

	"github.com/gorilla/websocket"
)

// HandleWebSocketPong bridges a browser websocket connection into the same
// Registry/game logic that the bidi Connect/gRPC service (service.go) uses.
// Browsers can't actually drive that gRPC service: connect-web's
// createConnectTransport only implements unary and server-streaming over
// fetch, and client/bidi streaming isn't implementable from a browser at
// all (gRPC-Web has the same restriction at the protocol level). A plain
// websocket, like every other game in this repo already uses, has no such
// problem.
func HandleWebSocketPong(conn *websocket.Conn, svc *Service, gameID string) {
	defer conn.Close()

	g, playerIndex, _, err := svc.Registry().Join(gameID)
	if err != nil {
		_ = conn.WriteJSON(wsMessage{Type: "error", Message: err.Error()})
		return
	}
	defer svc.Registry().Leave(gameID, g)

	if err := conn.WriteJSON(wsMessage{Type: "joined", PlayerIndex: playerIndex}); err != nil {
		return
	}

	recvErr := make(chan error, 1)
	go func() {
		for {
			var msg wsMessage
			if err := conn.ReadJSON(&msg); err != nil {
				recvErr <- err
				return
			}
			if msg.Type == "input" {
				g.setDirection(playerIndex, directionFromWire(msg.Direction))
			}
		}
	}()

	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()
	for {
		select {
		case <-recvErr:
			return
		case <-ticker.C:
			s := g.snapshot()
			out := wsMessage{
				Type:     "state",
				BallX:    s.BallX,
				BallY:    s.BallY,
				Paddle1Y: s.Paddle1Y,
				Paddle2Y: s.Paddle2Y,
				Score1:   s.Score1,
				Score2:   s.Score2,
			}
			if err := conn.WriteJSON(out); err != nil {
				return
			}
		}
	}
}

// wsMessage is used for both directions: the browser only ever sends
// {type:"input", direction:"up"|"down"|"stop"}, and only ever receives
// "joined"/"state"/"error" messages, so one loosely-typed struct is simpler
// than four narrow ones.
type wsMessage struct {
	Type        string  `json:"type"`
	Direction   string  `json:"direction,omitempty"`
	PlayerIndex int     `json:"playerIndex"`
	BallX       float64 `json:"ballX"`
	BallY       float64 `json:"ballY"`
	Paddle1Y    float64 `json:"paddle1Y"`
	Paddle2Y    float64 `json:"paddle2Y"`
	Score1      int     `json:"score1"`
	Score2      int     `json:"score2"`
	Message     string  `json:"message,omitempty"`
}

func directionFromWire(s string) direction {
	switch s {
	case "up":
		return dirUp
	case "down":
		return dirDown
	default:
		return dirStop
	}
}
