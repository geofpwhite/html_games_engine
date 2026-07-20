package main

import (
	"log/slog"
	"os"

	engine "github.com/geofpwhite/html_games_engine/engine"
	interfaces "github.com/geofpwhite/html_games_engine/interfaces"
	"github.com/gorilla/websocket"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	games := make(map[string]interfaces.Game)
	playerHashes := make(map[string]*websocket.Conn)
	inputChannel := make(chan interfaces.Input)
	outputChannel := make(chan string)
	go engine.Serve(inputChannel, games, playerHashes)
	go engine.OutputLoop(outputChannel, games, playerHashes)
	engine.GameLoop(inputChannel, outputChannel, games)
}
