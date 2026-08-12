package hangman

import (
	"fmt"
	"html/template"
	"net/http"
	"reflect"
	"slices"
	"strconv"

	IDGenerator "github.com/geofpwhite/html_games_engine/IDGenerator"
	"github.com/geofpwhite/html_games_engine/accounts/gamesession"
	interfaces "github.com/geofpwhite/html_games_engine/interfaces"
	"github.com/geofpwhite/html_games_engine/metrics"

	"github.com/geofpwhite/pq"
	"github.com/gorilla/websocket"
)

func Routes(r *http.ServeMux, tmpl *template.Template, upgrader *websocket.Upgrader,
	games map[string]interfaces.Game, playerHashes map[string]*websocket.Conn, inputChannel *pq.PriorityChannel[interfaces.Input],
	sessions *gamesession.Tracker,
) {
	r.Handle("GET /hangman_game/", http.StripPrefix("/hangman_game/", http.FileServer(http.Dir("./build_hangman/"))))

	r.HandleFunc("GET /hangman/grateful.jpeg", func(w http.ResponseWriter, req *http.Request) {
		http.ServeFile(w, req, "grateful.jpeg")
	})

	r.HandleFunc("GET /hangman", func(w http.ResponseWriter, _ *http.Request) {
		if err := tmpl.ExecuteTemplate(w, "home_screen_hangman.go.tmpl", nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	r.HandleFunc("GET /hangman/{gameID}", func(w http.ResponseWriter, req *http.Request) {
		gameID := req.PathValue("gameID")
		if gameID == "" {
			http.NotFound(w, req)
			return
		}
		if err := tmpl.ExecuteTemplate(w, "hangman.go.tmpl", map[string]any{"GameID": gameID}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	r.HandleFunc("GET /hangman/new_game", func(w http.ResponseWriter, _ *http.Request) {
		gState := newGameHangman()
		var game interfaces.Game = gState
		games[gState.gameID] = game
		metrics.GamesPlayed.WithLabelValues("hangman").Inc()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"gameID":%q}`, gState.gameID)
	})

	r.HandleFunc("GET /hangman/get_games", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "0")
	})

	r.HandleFunc("GET /hangman/valid/{playerHash}", func(w http.ResponseWriter, req *http.Request) {
		hash := req.PathValue("playerHash")
		if hash == "" {
			return
		}
		if playerHashes[hash] == nil {
			fmt.Fprint(w, "-1")
			return
		}
		var gameID string
		for i, g := range games {
			if reflect.TypeOf(g) == reflect.TypeFor[*hangman]() {
				for _, p := range (g).(*hangman).Players() {
					if p.PlayerID == hash {
						gameID = i
					}
				}
			}
		}
		fmt.Fprint(w, gameID)
	})

	r.HandleFunc("GET /hangman/exit_game/{playerHash}/{gameID}", func(_ http.ResponseWriter, req *http.Request) {
		playerHash := req.PathValue("playerHash")
		gameID := req.PathValue("gameID")
		_player := playerHashes[playerHash]
		if _player == nil || games[gameID] == nil {
			return
		}
		playerIndex := slices.IndexFunc((games[gameID]).(*hangman).players,
			func(p *interfaces.Player) bool { return p.PlayerID == playerHash })

		delete(playerHashes, playerHash)
		egi := &exitGameInput{gameID, playerIndex}
		if err := inputChannel.Push(egi, egi.Priority()); err != nil {
			fmt.Println(err)
		}
	})

	r.HandleFunc("GET /hangman/reconnect/{playerHash}/{gameID}", func(w http.ResponseWriter, req *http.Request) {
		playerHash := req.PathValue("playerHash")
		gameID := req.PathValue("gameID")
		if playerHash == "" || gameID == "" {
			return
		}
		conn, err := upgrader.Upgrade(w, req, nil)
		if err != nil {
			fmt.Println(err)
			return
		}

		if games[gameID] != nil {
			sessionID, tracked := sessions.Start(req.Context(), req, "hangman")
			handleWebSocketHangman(conn, inputChannel, games[gameID], true, playerHash, playerHashes)
			if tracked {
				sessions.End(req.Context(), sessionID)
			}
		} else {
			if err := conn.WriteJSON(hangmanClientState{Hash: "undefined", Warning: "1"}); err != nil {
				fmt.Println(err)
			}
			conn.Close()
		}
	})

	r.HandleFunc("GET /hangman/ws/{gameID}", func(w http.ResponseWriter, req *http.Request) {
		gameID := req.PathValue("gameID")
		conn, err := upgrader.Upgrade(w, req, nil)
		if err != nil || gameID == "" {
			panic("/hangman/ws/:gameID gave an error")
		}
		fmt.Println(games[gameID])
		fmt.Println(gameID)
		sessionID, tracked := sessions.Start(req.Context(), req, "hangman")
		handleWebSocketHangman(conn, inputChannel, games[gameID], false, "", playerHashes)
		if tracked {
			sessions.End(req.Context(), sessionID)
		}
	})
}

func handleWebSocketHangman( //nolint:funlen // It's gonna get a little messy
	conn *websocket.Conn,
	inputChannel *pq.PriorityChannel[interfaces.Input],
	gameObj interfaces.Game,
	reconnect bool,
	hash string,
	playerHashes map[string]*websocket.Conn,
) {
	gState, ok := gameObj.(*hangman)
	if !ok {
		return
	}
	var playerIndex int
	if reconnect { //nolint:nestif // It's gonna get a little messy
		conn2 := playerHashes[hash]
		if conn2 != nil {
			if err := conn2.Close(); err != nil {
				fmt.Println(err)
			}
			playerIndex = slices.IndexFunc(gState.players, func(p *interfaces.Player) bool { return p.PlayerID == hash })
			if playerIndex == -1 {
				if err := conn.WriteJSON(hangmanClientState{Hash: "undefined", Warning: "2"}); err != nil {
					fmt.Println(err)
				}
				conn.Close()
				return
			}
			playerHashes[hash] = conn
		}
	} else {
		playerIndex = len(gState.players)
		playerHash := IDGenerator.GenerateID(32)
		hash = playerHash
		newPlayer := interfaces.Player{Username: "Player " + strconv.Itoa(playerIndex+1), PlayerID: playerHash}
		gState.newPlayer(newPlayer)

		playerHashes[playerHash] = conn
		usernames := []string{}
		for _, p := range gState.players {
			usernames = append(usernames, p.Username)
		}
		if err := conn.WriteJSON(hangmanClientState{
			Players:        usernames,
			Turn:           gState.turn,
			Host:           gState.curHostIndex,
			RevealedWord:   gState.revealedWord,
			GuessesLeft:    gState.guessesLeft,
			LettersGuessed: gState.guessed,
			NeedNewWord:    gState.needNewWord,
			Warning:        "",
			PlayerIndex:    playerIndex,
			Winner:         gState.winner,
			GameID:         gState.gameID,
			ChatLogs:       gState.chatLogs,
			Hash:           playerHash,
		}); err != nil {
			fmt.Println(err)
		}
	}
	defer conn.Close()
	usernames := []string{}
	for _, p := range gState.players {
		usernames = append(usernames, p.Username)
	}

	currentState := hangmanClientState{
		Players:        usernames,
		Turn:           gState.turn,
		Host:           gState.curHostIndex,
		RevealedWord:   gState.revealedWord,
		GuessesLeft:    gState.guessesLeft,
		LettersGuessed: gState.guessed,
		NeedNewWord:    gState.needNewWord,
		Warning:        "",
		PlayerIndex:    playerIndex,
		Winner:         gState.winner,
		GameID:         gState.gameID,
		ChatLogs:       gState.chatLogs,
	}

	for i, player := range gState.players {
		currentState.PlayerIndex = i
		if err := playerHashes[player.PlayerID].WriteJSON(currentState); err != nil {
			fmt.Println(err)
		}
	}

	// push enqueues inp and reports whether it succeeded, logging on failure.
	// A Push only fails once inputChannel has been closed (server shutdown),
	// so there's no point continuing to read from this connection afterward.
	push := func(inp interfaces.Input) bool {
		if err := inputChannel.Push(inp, inp.Priority()); err != nil {
			fmt.Println(err)
			return false
		}
		return true
	}

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			return
		}

		GameID := gState.gameID
		PlayerIndex := slices.IndexFunc(gState.players, func(p *interfaces.Player) bool {
			return p.PlayerID == hash
		})
		if messageType == websocket.TextMessage { //nolint:nestif // It's gonna get a little messy
			pString := string(p)
			switch pString[:2] {
			case "g:":
				Guess := pString[2:]
				inp := &guessInput{gameID: GameID, playerIndex: PlayerIndex, guess: Guess}
				if !push(inp) {
					return
				}
			case "u:":
				Username := pString[2:]
				inp := &usernameInput{gameID: GameID, playerIndex: PlayerIndex, username: Username}
				if !push(inp) {
					return
				}
			case "w:":
				Word := pString[2:]
				inp := &newWordInput{gameID: GameID, playerIndex: PlayerIndex, newWord: Word}
				if !push(inp) {
					return
				}
			case "c:":
				Chat := pString[2:]
				inp := &chatInput{gameID: GameID, playerIndex: PlayerIndex, message: Chat}
				if !push(inp) {
					return
				}
			case "r:":
				inp := &randomlyChooseWordInput{gameID: GameID, playerIndex: PlayerIndex}
				if !push(inp) {
					return
				}

			default:
				continue
			}
		}
	}
}
