package interfaces

type InputType string

/*
Game interface for each game state struct to implement
*/
type Game interface {
	Players() []*Player
	JSON() ClientState
}

/*
Input interface for each game type's user input object to implement
*/
type Input interface {
	GameID() string
	PlayerIndex() int
	ChangeState(g Game)
}

type Player struct {
	PlayerID    string
	GameID      string
	PlayerIndex int
	Username    string
}

/*
ClientState interface for each game type's json-sendable object to implement
*/
type ClientState any
