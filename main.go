package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/geofpwhite/html_games_engine/accounts/cache/rediscache"
	"github.com/geofpwhite/html_games_engine/accounts/gamesession"
	"github.com/geofpwhite/html_games_engine/accounts/store/pgstore"
	engine "github.com/geofpwhite/html_games_engine/engine"
	interfaces "github.com/geofpwhite/html_games_engine/interfaces"
	"github.com/geofpwhite/pq"
	"github.com/gorilla/websocket"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	ctx := context.Background()
	games := make(map[string]interfaces.Game)
	playerHashes := make(map[string]*websocket.Conn)
	inputChannel := pq.NewPriorityChannel[interfaces.Input]()
	outputChannel := pq.NewPriorityChannel[string]()

	userStore := pgstore.NewStore()
	userCache := rediscache.NewCache()
	sessions := gamesession.New(userStore, userCache)

	go engine.Serve(inputChannel, games, playerHashes, userStore, userCache, sessions)
	go engine.OutputLoop(outputChannel, games, playerHashes, ctx)
	engine.GameLoop(inputChannel, outputChannel, games, sessions, ctx)
}
