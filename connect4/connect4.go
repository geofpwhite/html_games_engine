package connect4

import (
	"slices"
	"sync"

	IDGenerator "github.com/geofpwhite/html_games_engine/IDGenerator"
	interfaces "github.com/geofpwhite/html_games_engine/interfaces"
)

const (
	EMPTY = 0
	BLUE  = 1
	RED   = 2
)

type connect4InsertInput struct {
	gameID      string
	playerIndex int
	team        int
	column      int
}
type connect4RotateInput struct {
	gameID      string
	playerIndex int
}

func (c4i *connect4InsertInput) ChangeState(gameObj interfaces.Game) {
	c4 := (gameObj).(*connect4)
	c4.Insert(c4i.team, c4i.column)
	y := c4.scanForConnect4()
	if len(y) > 0 {
		c4.Clear()
	}
}

func (c4i *connect4InsertInput) GameID() string {
	return c4i.gameID
}

func (c4i *connect4InsertInput) PlayerIndex() int {
	return c4i.playerIndex
}

func (c4i *connect4RotateInput) ChangeState(gameObj interfaces.Game) {
	c4 := (gameObj).(*connect4)
	c4.Rotate()
	y := c4.scanForConnect4()
	if len(y) > 0 {
		c4.Clear()
	}
}

func (c4i *connect4RotateInput) GameID() string {
	return c4i.gameID
}

func (c4i *connect4RotateInput) PlayerIndex() int {
	return c4i.playerIndex
}

// -------------------------//
type connect4ClientState struct {
	Field [][]int `json:"Field"`
}

// --------------------------------------------------------//
type connect4 struct {
	field            [][]int
	turn             int
	playersConnected int
	players          []*interfaces.Player
	mut              *sync.RWMutex
}

func newGameConnect4() (*connect4, string) {
	c4 := &connect4{
		field: make([][]int, 8),
		turn:  BLUE,
		mut:   &sync.RWMutex{},
	}
	for i := range 8 {
		c4.field[i] = make([]int, 8)
	}
	hash := IDGenerator.GenerateID(6)
	return c4, hash
}

func (c4 *connect4) Players() []*interfaces.Player {
	return c4.players
}

func (c4 *connect4) JSON() interfaces.ClientState {
	c4.mut.RLock()
	defer c4.mut.RUnlock()
	cp := make([][]int, 8)
	copy(cp, c4.field)
	slices.Reverse(cp)
	c4cs := connect4ClientState{Field: cp}
	return c4cs
}

func (c4 *connect4) Clear() {
	c4.mut.Lock()
	defer c4.mut.Unlock()
	c4.field = make([][]int, 8)
	c4.turn = BLUE
	for i := range 8 {
		c4.field[i] = make([]int, 8)
	}
}

func (c4 *connect4) Insert(team, row int) bool {
	c4.mut.Lock()
	defer func() {
		c4.mut.Unlock()
		c4.Fall()
	}()
	if c4.field[7][row] == 0 {
		c4.field[7][row] = team
		c4.turn = (team % 2) + 1
		return true
	}
	return false
}

func (c4 *connect4) Rotate() {
	c4.mut.Lock()
	defer func() {
		c4.mut.Unlock()
		c4.Fall()
	}()

	size := len(c4.field)
	newField := make([][]int, size)

	for i := range newField {
		for j := range newField {
			newField[size-j-1] = append(newField[size-j-1], c4.field[i][j])
		}
	}
	c4.field = newField
}

func (c4 *connect4) Fall() {
	c4.mut.Lock()
	defer c4.mut.Unlock()
	for j := range c4.field {
		for i := range c4.field {
			k := i
			for k > 0 && c4.field[k-1][j] == EMPTY && c4.field[k][j] != EMPTY {
				c4.field[k-1][j] = c4.field[k][j]
				c4.field[k][j] = EMPTY
				k--
			}
		}
	}
}

type queueElement struct {
	team            int
	coordinate      [2]int
	upStreak        int
	rightStreak     int
	rightUpStreak   int
	rightDownStreak int
}

// rotating may cause both players to have
func (c4 *connect4) scanForConnect4() map[queueElement]bool {
	c4.mut.RLock()
	defer c4.mut.RUnlock()
	winners := map[queueElement]bool{}
	coordinateQueue := []queueElement{}
	for i, num := range c4.field[0] {
		if num > 0 {
			coordinateQueue = append(coordinateQueue, queueElement{team: num, coordinate: [2]int{0, i}})
		}
	}
	// start  by checking bottom left , then check each neighbor going to the right, going up, going down, but not going left.

	for len(coordinateQueue) > 0 {
		poppedElement := coordinateQueue[0]
		coordinateQueue = coordinateQueue[1:]
		if poppedElement.upStreak >= 3 || poppedElement.rightStreak >= 3 ||
			poppedElement.rightUpStreak >= 3 || poppedElement.rightDownStreak >= 3 {
			winners[poppedElement] = true
		}
		// check 4 neighbors
		up := [2]int{poppedElement.coordinate[0] + 1, poppedElement.coordinate[1]}
		right := [2]int{poppedElement.coordinate[0], poppedElement.coordinate[1] + 1}
		rightUp := [2]int{poppedElement.coordinate[0] + 1, poppedElement.coordinate[1] + 1}
		rightDown := [2]int{poppedElement.coordinate[0] - 1, poppedElement.coordinate[1] + 1}
		possibleNeighborCoordinates := [][2]int{up, right, rightUp, rightDown}
		for i, coord := range possibleNeighborCoordinates {
			if coord[0] >= 0 && coord[0] < 8 && coord[1] >= 0 && coord[1] < 8 {
				neighbor := queueElement{coordinate: coord, team: c4.field[coord[0]][coord[1]]}
				if neighbor.team == poppedElement.team {
					switch i {
					case 0:
						neighbor.upStreak = poppedElement.upStreak + 1
						neighbor.rightStreak = 0
						neighbor.rightUpStreak = 0
						neighbor.rightDownStreak = 0

					case 1:
						neighbor.rightStreak = poppedElement.rightStreak + 1
						neighbor.rightDownStreak = 0
						neighbor.rightUpStreak = 0
						neighbor.upStreak = 0
					case 2:
						neighbor.rightUpStreak = poppedElement.rightUpStreak + 1
						neighbor.rightDownStreak = 0
						neighbor.rightStreak = 0
						neighbor.upStreak = 0
					case 3:
						neighbor.rightDownStreak = poppedElement.rightDownStreak + 1
						neighbor.rightUpStreak = 0
						neighbor.rightStreak = 0
						neighbor.upStreak = 0
					}
					coordinateQueue = append(coordinateQueue, neighbor)
				} else if neighbor.team != EMPTY {
					coordinateQueue = append(coordinateQueue, neighbor)
				}
			}
		}
	}
	return winners
}
