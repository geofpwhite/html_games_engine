package main

import (
	"log/slog"
	"os"

	engine "github.com/geofpwhite/html_games_engine/engine"
	interfaces "github.com/geofpwhite/html_games_engine/interfaces"
	"github.com/geofpwhite/pq"
	"github.com/gorilla/websocket"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	games := make(map[string]interfaces.Game)
	playerHashes := make(map[string]*websocket.Conn)
	inputChannel := pq.NewPriorityChannel[interfaces.Input]()
	outputChannel := pq.NewPriorityChannel[string]()
	go engine.Serve(inputChannel, games, playerHashes)
	go engine.OutputLoop(outputChannel, games, playerHashes)
	engine.GameLoop(inputChannel, outputChannel, games)
}
