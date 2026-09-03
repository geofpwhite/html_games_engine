package engine

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/geofpwhite/html_games_engine/accounts/gamesession"
	interfaces "github.com/geofpwhite/html_games_engine/interfaces"
	"github.com/geofpwhite/pq"
)

func GameLoop(
	inputChannel *pq.PriorityChannel[interfaces.Input],
	outputChannel *pq.PriorityChannel[string],
	games map[string]interfaces.Game,
	sessions *gamesession.Tracker,
	ctx context.Context,
) {
	lastModified := map[interfaces.Game]time.Time{}
	var mu sync.Mutex
	cleanupFunction := func() {
		ticker := time.NewTicker(20 * time.Minute)
		defer ticker.Stop()
		lastTick := time.Now()
		for interval := range ticker.C {
			mu.Lock()
			for id, game := range games {
				if lastTick.Compare(lastModified[game]) > 0 {
					// close the game
					// hmm
					delete(games, id)
				}
			}
			mu.Unlock()
			lastTick = interval
		}
	}
	go cleanupFunction()
	for {
		userInput, priority, err := inputChannel.PopBlocking(ctx)
		if err != nil {
			break
		}
		mu.Lock() // noop unless we are cleaning up
		gameID := userInput.GameID()
		game, ok := games[gameID]
		if !ok {
			mu.Unlock()
			continue
		}
		userInput.ChangeState(game)
		for _, winner := range game.ConsumeWinners() {
			if winner.GameSessionID != 0 {
				go func(sessionID int32, ctx context.Context) {
					recordCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					if err := sessions.RecordWin(recordCtx, sessionID); err != nil {
						slog.Error("gameLoop: RecordWin failed", "error", err, "sessionID", sessionID)
					}
				}(winner.GameSessionID, ctx)
			}
		}
		lastModified[game] = time.Now()
		if err := outputChannel.Push(gameID, priority); err != nil {
			slog.Error("gameLoop: output channel push failed", "error", err)
			mu.Unlock()
			break
		}
		mu.Unlock()
	}
}
