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
	"github.com/geofpwhite/html_games_engine/accounts/gamesession"
	"github.com/geofpwhite/html_games_engine/accounts/store"
	"github.com/geofpwhite/html_games_engine/accounts/store/pgstore"
	connectthedots "github.com/geofpwhite/html_games_engine/connectTheDots"
	interfaces "github.com/geofpwhite/html_games_engine/interfaces"
	"github.com/geofpwhite/html_games_engine/metrics"
	tictactoe "github.com/geofpwhite/html_games_engine/ticTacToe"

	connect4 "github.com/geofpwhite/html_games_engine/connect4"
	hangman "github.com/geofpwhite/html_games_engine/hangman"
	"github.com/geofpwhite/html_games_engine/pong"
	"github.com/geofpwhite/html_games_engine/pong/pongv1/pongv1connect"
	whiteboard "github.com/geofpwhite/html_games_engine/whiteboard"

	"github.com/geofpwhite/pq"
	"github.com/gorilla/websocket"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
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

func Serve(
	inputChannel *pq.PriorityChannel[interfaces.Input],
	games map[string]interfaces.Game,
	playerHashes map[string]*websocket.Conn,
) {
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

	userStore := pgstore.NewStore()
	userCache := rediscache.NewCache()
	sessions := gamesession.New(userStore, userCache)

	hangman.Routes(r, tmpl, &upgrader, games, playerHashes, inputChannel, sessions)
	connect4.Routes(r, tmpl, &upgrader, games, playerHashes, inputChannel, sessions)
	connectthedots.Routes(r, tmpl, &upgrader, games, playerHashes, inputChannel, sessions)
	tictactoe.Routes(r, tmpl, &upgrader, games, playerHashes, inputChannel, sessions)
	whiteboard.Routes(r, tmpl, &upgrader, games, playerHashes, inputChannel, sessions)

	accounts.AccountRoutes(r, tmpl, &upgrader, userStore, userCache)

	pongService := pong.NewService()
	pong.Routes(r, tmpl, &upgrader, pongService)
	pongPath, pongHandler := pongv1connect.NewPongServiceHandler(pongService)
	// Play is a bidi stream, always called over POST (as every gRPC/Connect
	// streaming call is) - registering it with an explicit method avoids an
	// ambiguous-pattern panic against the "GET /" catch-all above.
	r.Handle("POST "+pongPath, pongHandler)
	// Not used by the play page itself (that's plain WS, see pong/ws.go) -
	// this just exposes the generated Connect/protobuf JS client as a
	// reference for anyone building a non-browser gRPC/Connect client.
	r.Handle("GET /pong_static/", http.StripPrefix("/pong_static/", http.FileServer(http.Dir("./pong/web/src/"))))

	r.Handle("GET /metrics", metrics.Handler())
	go metrics.PollUserCounts(context.Background(), userCounts{store: userStore, cache: userCache}, 30*time.Second)

	limiter := newRateLimiter(1, 20)
	// h2c lets gRPC clients speak HTTP/2 to the pong service in cleartext;
	// it falls through to plain HTTP/1.1 (websockets included) for everything else.
	handler := h2c.NewHandler(limiter.middleware(metrics.VisitMiddleware(r)), &http2.Server{})
	srv := &http.Server{
		Addr:              ":8080",
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}
