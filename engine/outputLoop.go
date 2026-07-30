package engine

import (
	interfaces "github.com/geofpwhite/html_games_engine/interfaces"
	"github.com/geofpwhite/pq"

	"github.com/gorilla/websocket"
)

func OutputLoop(outputChannel *pq.PriorityChannel[string],
	games map[string]interfaces.Game,
	playerHashes map[string]*websocket.Conn,
) {
	var game interfaces.Game
	var json interfaces.ClientState
	var conn *websocket.Conn
	for {
		gameID, _ := outputChannel.PopBlocking()
		game = games[gameID]
		json = game.JSON()
		for _, p := range game.Players() {
			conn = playerHashes[p.PlayerID]
			if err := conn.WriteJSON(json); err != nil {
				continue
			}
		}
	}
}
