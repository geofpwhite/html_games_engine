package tictactoe

import interfaces "github.com/geofpwhite/html_games_engine/interfaces"

type moveInput struct {
	gameID      string
	playerIndex int
	team        int
	X           int `json:"x"`
	Y           int `json:"y"`
}

func (mi *moveInput) GameID() string {
	return mi.gameID
}

func (mi *moveInput) PlayerIndex() int {
	return mi.playerIndex
}

func (mi *moveInput) ChangeState(gameObj interfaces.Game) {
	if gState, ok := gameObj.(*ticTacToe); ok {
		gState.move(mi.X, mi.Y, mi.team)
	}
}
