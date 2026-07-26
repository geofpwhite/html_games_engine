package engine

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"time"

	accounts "github.com/geofpwhite/html_games_engine/accounts"
	"github.com/geofpwhite/html_games_engine/accounts/cache"
	"github.com/geofpwhite/html_games_engine/accounts/cache/rediscache"
	"github.com/geofpwhite/html_games_engine/accounts/store"
	"github.com/geofpwhite/html_games_engine/accounts/store/pgstore"
	connectthedots "github.com/geofpwhite/html_games_engine/connectTheDots"
	interfaces "github.com/geofpwhite/html_games_engine/interfaces"
	"github.com/geofpwhite/html_games_engine/metrics"
	tictactoe "github.com/geofpwhite/html_games_engine/ticTacToe"

	connect4 "github.com/geofpwhite/html_games_engine/connect4"
	hangman "github.com/geofpwhite/html_games_engine/hangman"
	whiteboard "github.com/geofpwhite/html_games_engine/whiteboard"

	"github.com/gorilla/websocket"
)

// userCounts adapts the existing store/cache methods to metrics.UserCounts
// without widening either interface just for a periodic gauge refresh.
type userCounts struct {
	store store.Store
	cache cache.Cache
}

func (u userCounts) CountUsers(ctx context.Context) (int, error) {
	usernames, err := u.store.GetUsernames(ctx)
	if err != nil {
		return 0, err
	}
	return len(usernames), nil
}

func (u userCounts) CountOnline() (int, error) {
	online, err := u.cache.ListOnline()
	if err != nil {
		return 0, err
	}
	return len(online), nil
}

func mod(a, b int) int {
	return a % b
}

func Serve(inputChannel chan interfaces.Input, games map[string]interfaces.Game, playerHashes map[string]*websocket.Conn) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	r := http.NewServeMux()
	funcMap := template.FuncMap{
		"mod": mod,
	}
	tmpl := template.Must(template.New("").Funcs(funcMap).ParseGlob("templates/*"))
	r.HandleFunc("GET /", func(w http.ResponseWriter, _ *http.Request) {
		if err := tmpl.ExecuteTemplate(w, "home_page.go.tmpl", nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	r.HandleFunc("GET /about", func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, "https://github.com/geofpwhite/html_games_engine", http.StatusMovedPermanently)
	})
	r.HandleFunc("GET /favicon.png", func(w http.ResponseWriter, req *http.Request) {
		http.ServeFile(w, req, "geofpwhite.us.png")
	})

	hangman.Routes(r, tmpl, &upgrader, games, playerHashes, inputChannel)
	connect4.Routes(r, tmpl, &upgrader, games, playerHashes, inputChannel)
	connectthedots.Routes(r, tmpl, &upgrader, games, playerHashes, inputChannel)
	tictactoe.Routes(r, tmpl, &upgrader, games, playerHashes, inputChannel)
	whiteboard.Routes(r, tmpl, &upgrader, games, playerHashes, inputChannel)

	userStore := pgstore.NewStore()
	userCache := rediscache.NewCache()
	accounts.AccountRoutes(r, tmpl, &upgrader, userStore, userCache)

	r.Handle("GET /metrics", metrics.Handler())
	go metrics.PollUserCounts(context.Background(), userCounts{store: userStore, cache: userCache}, 30*time.Second)

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           metrics.VisitMiddleware(r),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}
