package connectthedots

import interfaces "github.com/geofpwhite/html_games_engine/interfaces"

type connectTheDotsAddEdgeInput struct {
	team        int
	coords      [2]int
	gameID      string
	playerIndex int
}

func (ctdaei *connectTheDotsAddEdgeInput) GameID() string {
	return ctdaei.gameID
}

func (ctdaei *connectTheDotsAddEdgeInput) PlayerIndex() int {
	return ctdaei.playerIndex
}

func (ctdaei *connectTheDotsAddEdgeInput) ChangeState(gameObj interfaces.Game) {
	if gState, ok := gameObj.(*connectTheDots); ok {
		gState.addEdge(ctdaei.coords, ctdaei.team)
	}
}
