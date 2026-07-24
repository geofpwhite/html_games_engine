package connectthedots

import (
	IDGenerator "github.com/geofpwhite/html_games_engine/IDGenerator"
	interfaces "github.com/geofpwhite/html_games_engine/interfaces"
)

const (
	EMPTY = 0
	BLUE  = 1
	RED   = 2
	POINT = 3
)

type connectTheDots struct {
	size             int     // number of dots in a row
	field            [][]int //
	turn             int
	players          []*interfaces.Player
	playersConnected int
}

type connectTheDotsClientState struct {
	Size      int     `json:"Size"`
	Field     [][]int `json:"Field"`
	Turn      int     `json:"Turn"`
	RedScore  int     `json:"RedScore"`
	BlueScore int     `json:"BlueScore"`
}

func NewGameConnectTheDots(size int) (*connectTheDots, string) {
	field := make([][]int, size+size-1)
	for i := range field {
		field[i] = make([]int, size+size-1)
	}
	for i := range field {
		for j := range field {
			if i%2 == 0 && j%2 == 0 {
				field[i][j] = POINT
			}
		}
	}
	ctd := &connectTheDots{size: size, field: field, turn: BLUE}
	hash := IDGenerator.GenerateID(6)
	return ctd, hash
}

func (ctd *connectTheDots) addEdge(coord [2]int, team int) {
	if team != ctd.turn {
		println("not your turn")
		return
	}
	if coord[0] >= ctd.size*2 ||
		coord[1] >= ctd.size*2 ||
		coord[0] < 0 ||
		coord[1] < 0 { //nolint:gosec // coord is a fixed [2]int; index is a compile-time constant, always in bounds
		println("out of bounds")
		return
	}
	if coord[0]%2 == coord[1]%2 { //nolint:gosec // coord is a fixed [2]int; index is a compile-time constant, always in bounds
		println("not an edge")
		return
	}

	if ctd.field[coord[0]][coord[1]] == EMPTY {
		ctd.field[coord[0]][coord[1]] = team
	}
	affectedCells := [][2]int{}
	appendIfValid := func(valid bool, cell [2]int) {
		if valid {
			affectedCells = append(affectedCells, cell)
		}
	}
	if coord[0]%2 == 0 {
		appendIfValid(coord[0] > 0, [2]int{coord[0] - 1, coord[1]})
		appendIfValid(coord[0] < ctd.size+ctd.size-2, [2]int{coord[0] + 1, coord[1]})
	} else {
		appendIfValid(coord[1] > 0, [2]int{coord[0], coord[1] - 1})
		appendIfValid(coord[1] < ctd.size+ctd.size-2, [2]int{coord[0], coord[1] + 1})
	}
	advanceTurn := true
	for _, coords := range affectedCells {
		if ctd.field[coords[0]][coords[1]] != EMPTY {
			continue
		}

		shouldContinue := false
		edgesToCheck := [4][2]int{
			{coords[0] + 1, coords[1]},
			{coords[0] - 1, coords[1]},
			{coords[0], coords[1] + 1},
			{coords[0], coords[1] - 1},
		}
		for i := range edgesToCheck {
			if ctd.field[edgesToCheck[i][0]][edgesToCheck[i][1]] == EMPTY {
				shouldContinue = true
				break
			}
		}
		if shouldContinue {
			continue
		}
		ctd.field[coords[0]][coords[1]] = team
		advanceTurn = false
	}
	if advanceTurn {
		ctd.turn = (ctd.turn % 2) + 1
	}
}

func (ctd *connectTheDots) JSON() interfaces.ClientState {
	return connectTheDotsClientState{
		Field: ctd.field,
		Size:  ctd.size,
		Turn:  ctd.turn,
	}
}

func (ctd *connectTheDots) Players() []*interfaces.Player {
	return ctd.players
}
