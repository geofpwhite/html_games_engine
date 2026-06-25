package tictactoe

import (
	IDGenerator "github.com/geofpwhite/html_games_engine/IDGenerator"
	interfaces "github.com/geofpwhite/html_games_engine/interfaces"
)

const X, O = 1, 2

type ticTacToe struct {
	field       [3][3]int
	turn        int
	players     [2]*interfaces.Player
	scores      [2]int
	playersSize int
}

type ticTacToeClientState struct {
	Field  [3][3]int `json:"Field"`
	Turn   int       `json:"Turn"`
	Scores [2]int    `json:"Scores"`
}

func NewGameTicTacToe() (*ticTacToe, string) {
	gState := &ticTacToe{
		field:  [3][3]int{},
		turn:   1,
		scores: [2]int{},
	}
	return gState, IDGenerator.GenerateID(6)
}

// reset will only be called by the move() function, so we shan't Lock the mutex
func (gState *ticTacToe) reset() {
	gState.field = [3][3]int{}
	gState.turn = X
}

func (gState *ticTacToe) move(x, y, team int) {
	if gState.turn != team || gState.field[x][y] != 0 {
		return
	}
	gState.field[x][y] = team
	if gState.scan() {
		gState.reset()
		gState.scores[team]++
	} else {
		gState.turn = (gState.turn % 2) + 1
	}
}

var possibleTicTacToes = [8][3][2]int{
	{
		{0, 0}, {0, 1}, {0, 2},
	}, {
		{1, 0}, {1, 1}, {1, 2},
	}, {
		{2, 0}, {2, 1}, {2, 2},
	}, {
		{0, 0}, {1, 0}, {2, 0},
	}, {
		{0, 1}, {1, 1}, {2, 1},
	}, {
		{0, 2}, {1, 2}, {2, 2},
	}, {
		{0, 0}, {1, 1}, {2, 2},
	}, {
		{0, 2}, {1, 1}, {2, 0},
	},
}

// scan will only be called by move(), so no mut.Lock
func (gState *ticTacToe) scan() bool {
	for _, streak := range possibleTicTacToes {
		if gState.field[streak[0][0]][streak[0][1]] != 0 &&
			gState.field[streak[0][0]][streak[0][1]] == gState.field[streak[1][0]][streak[1][1]] &&
			gState.field[streak[1][0]][streak[1][1]] == gState.field[streak[2][0]][streak[2][1]] {
			return true
		}
	}
	return false
}

func (gState *ticTacToe) JSON() interfaces.ClientState {
	cState := ticTacToeClientState{
		Field: gState.field,
		Turn:  gState.turn,
	}
	return cState
}

func (gState *ticTacToe) Players() []*interfaces.Player {
	players := []*interfaces.Player{}
	if gState.players[0] != nil {
		players = append(players, gState.players[0])
	}
	if gState.players[1] != nil {
		players = append(players, gState.players[1])
	}
	return players
}

func (gState *ticTacToe) newPlayer(p interfaces.Player) int {
	if gState.playersSize < 2 {
		gState.players[gState.playersSize] = &p
		gState.playersSize++
		return gState.playersSize - 1
	}
	return -1
}
