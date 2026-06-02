package hangman

import (
	"database/sql"
	"fmt"
	"log"
	"slices"
	"strings"
	"sync"
	"time"

	IDGenerator "github.com/geofpwhite/html_games_engine/IDGenerator"
	interfaces "github.com/geofpwhite/html_games_engine/interfaces"

	_ "modernc.org/sqlite"
)

const (
	HOST_WINS  = 1
	HOST_LOSES = 2
)

type hangmanClientState struct {
	Players        []string  `json:"players"`
	Turn           int       `json:"turn"`
	Host           int       `json:"host"`
	RevealedWord   string    `json:"revealedWord"`
	GuessesLeft    int       `json:"guessesLeft"`
	LettersGuessed string    `json:"lettersGuessed"`
	NeedNewWord    bool      `json:"needNewWord"`
	Warning        string    `json:"warning"`
	PlayerIndex    int       `json:"playerIndex"` // changes for each connection that the update state object is sent to
	Winner         int       `json:"winner"`
	ChatLogs       []chatLog `json:"chatLogs"`
	Hash           string    `json:"hash"`
	GameID         string    `json:"gameID"`
}

type chatLog struct {
	Message string `json:"message"`
	Sender  string `json:"sender"`
}

/*
struct containing necessary fields for game to run
*/
type hangman struct {
	wordCheck           *sql.DB
	currentWord         string
	revealedWord        string
	guessed             string
	players             []*interfaces.Player
	curHostIndex        int
	turn                int
	guessesLeft         int
	needNewWord         bool
	winner              int
	mut                 *sync.RWMutex
	chatLogs            []chatLog
	consecutiveTimeouts int
	randomlyChosen      bool // boolean for methods to check if they need to act differently because the backend randomly chose a word
	gameID              string
}

func newGameHangman() *hangman {
	wordCheck, _ := sql.Open("sqlite3", "./words.db")
	gState := &hangman{
		wordCheck:    wordCheck,
		currentWord:  "",
		revealedWord: "",
		winner:       -1,
		needNewWord:  true,
		guessesLeft:  6,
		players:      make([]*interfaces.Player, 0),
		mut:          &sync.RWMutex{},
	}
	gameCode := IDGenerator.GenerateID(6)
	gState.gameID = gameCode
	return gState
}

/*
starts a ticker that either times out the current turn and increments it, or resets back to 0 on user input
*/
func (gState *hangman) runTicker(timeoutChannel chan<- string, inputChannel <-chan interfaces.Input, closeGameChannel chan<- string) {
	ticker := time.NewTicker(60 * time.Second)
	gState.consecutiveTimeouts = 0
	defer ticker.Stop()
	// defer close(inputChannel) // this may be bad practice to close from the reader side but

	for {
		select {
		case <-ticker.C:
			log.Println("ticker")

			timeoutChannel <- (*gState).gameID
			gState.consecutiveTimeouts++
			if gState.consecutiveTimeouts >= len(gState.players) {
				closeGameChannel <- gState.gameID
			}

		case x := <-inputChannel:
			log.Println("ticker input channel", x)
			if len((*gState).players) == 0 || x.PlayerIndex() == -1 {
				return
			}
			if x.PlayerIndex() == (*gState).turn {
				ticker.Stop()
				ticker = time.NewTicker(60 * time.Second)
				fmt.Println("ticker reset")
				gState.consecutiveTimeouts = 0
			}
			// ticker = time.NewTicker(1 * time.Second)
		}
	}
}

func (gState *hangman) newPlayer(p interfaces.Player) {
	gState.mut.Lock()
	defer gState.mut.Unlock()
	gState.players = append(gState.players, &p)
	if len(gState.players) == 2 && !gState.needNewWord {
		gState.turn = 1
	}
}

func (gState *hangman) guess(letter rune) bool {
	gState.mut.Lock()
	defer gState.mut.Unlock()
	if gState.needNewWord {
		return false
	}
	if !strings.Contains(gState.guessed, string(letter)) {
		good := false
		gState.guessed += string(letter)
		for i, char := range gState.currentWord {
			if char == letter {
				gState.revealedWord = gState.revealedWord[:i] + string(letter) + gState.revealedWord[i+1:]
				good = true
			}
		}
		x := 0x10p23
		print(x)
		if gState.currentWord == gState.revealedWord {
			gState.needNewWord = true
			gState.turn = (gState.curHostIndex + 2) % len(gState.players)
			gState.curHostIndex = (gState.curHostIndex + 1) % len(gState.players)
			gState.winner = HOST_LOSES
		} else if gState.guessesLeft == 1 && !good {
			gState.needNewWord = true
			gState.turn = (gState.curHostIndex + 2) % len(gState.players)
			gState.winner = HOST_WINS
			gState.curHostIndex = (gState.curHostIndex + 1) % len(gState.players)
		} else if !good {
			gState.guessesLeft--
			gState.turn = (gState.turn + 1) % len(gState.players)
			if gState.turn == gState.curHostIndex && !gState.randomlyChosen {
				gState.turn = (gState.turn + 1) % len(gState.players)
			}
		}
		return true
	}
	return false
}

func (gState *hangman) randomNewWord() {
	gState.mut.Lock()
	defer gState.mut.Unlock()
	x, _ := gState.wordCheck.Query("SELECT word FROM words WHERE LENGTH(word)>5 and word not like '%-%' ORDER BY RANDOM() LIMIT 1;")
	result := ""
	if x.Next() {
		x.Scan(&result)
		if result == "" {
			return
		}
	} else {
		return
	}
	gState.currentWord = result
	gState.revealedWord = ""
	gState.needNewWord = false
	gState.guessed = ""
	gState.guessesLeft = 6
	gState.winner = -1
	gState.randomlyChosen = true
	for range result {
		gState.revealedWord += "_"
	}
	gState.turn = (gState.curHostIndex + 1) % len(gState.players)
}

func (gState *hangman) newWord(word string) {
	gState.mut.Lock()
	defer gState.mut.Unlock()
	abc := "abcdefghijklmnopqrstuvwxyz"
	for _, c := range word {
		if !strings.Contains(abc, string(c)) {
			return
		}
	}
	x, _ := gState.wordCheck.Query("select word from words where word ='" + word + "'")
	result := ""
	if x.Next() {
		x.Scan(&result)
		if result == "" {
			return
		}
	} else {
		return
	}
	gState.currentWord = word
	gState.revealedWord = ""
	gState.needNewWord = false
	gState.guessed = ""
	gState.guessesLeft = 6
	gState.winner = -1
	gState.randomlyChosen = false
	for range word {
		gState.revealedWord += "_"
	}
	gState.turn = (gState.curHostIndex + 1) % len(gState.players)
}

func (gState *hangman) closeGame() {
	gState.mut.Lock()
	defer gState.mut.Unlock()
	// for _, p := range gState.players {
	// 	delete(hashes, p.hash)
	// }
	// delete(gameIDes, gState.gameID)
}

func (gState *hangman) removePlayer(playerIndex int) {
	if len(gState.players) == 0 {
		return
	}
	gState.mut.Lock()
	defer gState.mut.Unlock()
	gState.players = slices.Delete(gState.players, playerIndex, playerIndex+1)
	if len(gState.players) == 0 {
		gState.closeGame()
		return
	}
	gState.turn = gState.turn % len(gState.players)
	gState.curHostIndex = gState.curHostIndex % len(gState.players)
	if gState.needNewWord && gState.curHostIndex != gState.turn {
		gState.turn = gState.curHostIndex
	} else if !gState.needNewWord && gState.curHostIndex == gState.turn {
		gState.turn = (gState.turn + 1) % len(gState.players)
	}
}

func (gState *hangman) handleTickerTimeout() string {
	gState.mut.Lock()
	defer gState.mut.Unlock()
	if gState.needNewWord {
		gState.curHostIndex = (gState.curHostIndex + 1) % len(gState.players)
		gState.turn = gState.curHostIndex
	} else {
		(*gState).turn = ((*gState).turn + 1) % len((*gState).players)
		if (*gState).curHostIndex == (*gState).turn {
			(*gState).turn = ((*gState).turn + 1) % len((*gState).players)
		}
	}
	return gState.gameID
}

func (gState *hangman) changeUsername(playerIndex int, newUsername string) {
	log.Println("change username")
	gState.mut.Lock()
	defer gState.mut.Unlock()
	if slices.IndexFunc(gState.players, func(p *interfaces.Player) bool {
		return p.Username == newUsername
	}) == -1 {
		oldUsername := gState.players[playerIndex].Username
		gState.players[playerIndex].Username = newUsername
		for i, chat := range gState.chatLogs {
			if chat.Sender == oldUsername {
				gState.chatLogs[i].Sender = newUsername
			}
		}
	}
}

func (gState *hangman) chat(message string, playerIndex int) {
	gState.mut.Lock()
	defer gState.mut.Unlock()
	gState.chatLogs = append(gState.chatLogs,
		chatLog{
			Message: message,
			Sender:  gState.players[playerIndex].Username,
		})
}

func (gState *hangman) JSON() interfaces.ClientState {
	gState.mut.RLock()
	defer gState.mut.RUnlock()
	usernames := []string{}
	for _, p := range (*gState).players {
		usernames = append(usernames, p.Username)
	}

	newState := hangmanClientState{
		Players:        usernames,
		Turn:           gState.turn,
		Host:           gState.curHostIndex,
		RevealedWord:   gState.revealedWord,
		GuessesLeft:    gState.guessesLeft,
		LettersGuessed: gState.guessed,
		NeedNewWord:    gState.needNewWord,
		GameID:         gState.gameID,
		Warning:        "",
		Winner:         gState.winner,
		ChatLogs:       gState.chatLogs,
	}
	return newState
}

func (gState *hangman) Players() []*interfaces.Player {
	return gState.players
}

// func (gState *hangman) HandleInput(i Input) {
// 	hi := i.(*hangmanInput)
// 	i.
// }
