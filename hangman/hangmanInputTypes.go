package hangman

import interfaces "github.com/geofpwhite/html_games_engine/interfaces"

type usernameInput struct {
	username    string
	gameID      string
	playerIndex int
}
type newWordInput struct {
	newWord     string
	gameID      string
	playerIndex int
}
type randomlyChooseWordInput struct {
	gameID      string
	playerIndex int
}
type guessInput struct {
	guess       string
	gameID      string
	playerIndex int
}
type chatInput struct {
	message     string
	gameID      string
	playerIndex int
}
type exitGameInput struct {
	gameID      string
	playerIndex int
}

func (ui *usernameInput) GameID() string {
	return ui.gameID
}
func (ui *usernameInput) PlayerIndex() int {
	return ui.playerIndex
}
func (ui *usernameInput) ChangeState(gameObj interfaces.Game) {
	if gState, ok := (gameObj).(*hangman); ok {
		gState.changeUsername(ui.playerIndex, ui.username)
	}
}

func (nwi *newWordInput) GameID() string {
	return nwi.gameID
}
func (nwi *newWordInput) PlayerIndex() int {
	return nwi.playerIndex
}
func (nwi *newWordInput) ChangeState(gameObj interfaces.Game) {
	gState, ok := gameObj.(*hangman)
	if ok && gState.needNewWord && nwi.playerIndex == gState.curHostIndex {
		gState.newWord(nwi.newWord)
	}

}
func (gi *guessInput) GameID() string {
	return gi.gameID
}
func (gi *guessInput) PlayerIndex() int {
	return gi.playerIndex
}

func (gi *guessInput) ChangeState(gameObj interfaces.Game) {
	gState, ok := gameObj.(*hangman)
	if ok && gi.playerIndex == gState.turn {
		gState.guess(rune(gi.guess[0]))
	}
}

func (ci *chatInput) GameID() string {
	return ci.gameID
}
func (ci *chatInput) PlayerIndex() int {
	return ci.playerIndex
}
func (ci *chatInput) ChangeState(gameObj interfaces.Game) {
	if gState, ok := gameObj.(*hangman); ok {
		gState.chat(ci.message, ci.playerIndex)
	}
}

func (rcwi *randomlyChooseWordInput) GameID() string {
	return rcwi.gameID
}
func (rcwi *randomlyChooseWordInput) PlayerIndex() int {
	return rcwi.playerIndex
}
func (rcwi *randomlyChooseWordInput) ChangeState(gameObj interfaces.Game) {
	if gState, ok := gameObj.(*hangman); ok {
		gState.randomNewWord()
	}
}

func (egi *exitGameInput) GameID() string {
	return egi.gameID
}
func (egi *exitGameInput) PlayerIndex() int {
	return egi.playerIndex
}
func (egi *exitGameInput) ChangeState(gameObj interfaces.Game) {
	if gState, ok := gameObj.(*hangman); ok {
		gState.removePlayer(egi.playerIndex)
	}
}

