package hangman

import (
	"fmt"
	"html/template"
	"net/http"
	"reflect"
	"slices"
	"strconv"

	IDGenerator "github.com/geofpwhite/html_games_engine/IDGenerator"
	interfaces "github.com/geofpwhite/html_games_engine/interfaces"

	"github.com/gorilla/websocket"
)

func Routes(r *http.ServeMux, _ *template.Template, upgrader *websocket.Upgrader,
	games map[string]interfaces.Game, playerHashes map[string]*websocket.Conn, inputChannel chan interfaces.Input,
) {
	r.Handle("GET /hangman_game/", http.StripPrefix("/hangman_game/", http.FileServer(http.Dir("./build_hangman/"))))

	r.HandleFunc("GET /hangman/new_game", func(w http.ResponseWriter, _ *http.Request) {
		gState := newGameHangman()
		var game interfaces.Game = gState
		games[gState.gameID] = game
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
		inputChannel <- &exitGameInput{gameID, playerIndex}
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
			handleWebSocketHangman(conn, inputChannel, games[gameID], true, playerHash, playerHashes)
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
		handleWebSocketHangman(conn, inputChannel, games[gameID], false, "", playerHashes)
	})
}

func handleWebSocketHangman(
	conn *websocket.Conn,
	inputChannel chan interfaces.Input,
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

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			return
		}

		GameID := gState.gameID
		PlayerIndex := slices.IndexFunc(gState.players, func(p *interfaces.Player) bool {
			return p.PlayerID == hash
		})
		if messageType == websocket.TextMessage {
			pString := string(p)
			switch pString[:2] {
			case "g:":
				Guess := pString[2:]
				inp := guessInput{gameID: GameID, playerIndex: PlayerIndex, guess: Guess}
				inputChannel <- &inp
			case "u:":
				Username := pString[2:]
				inp := usernameInput{gameID: GameID, playerIndex: PlayerIndex, username: Username}
				inputChannel <- &inp
			case "w:":
				Word := pString[2:]
				inp := newWordInput{gameID: GameID, playerIndex: PlayerIndex, newWord: Word}
				inputChannel <- &inp
			case "c:":
				Chat := pString[2:]
				inp := chatInput{gameID: GameID, playerIndex: PlayerIndex, message: Chat}
				inputChannel <- &inp
			case "r:":
				inp := randomlyChooseWordInput{gameID: GameID, playerIndex: PlayerIndex}
				inputChannel <- &inp

			default:
				continue
			}
		}
	}
}
