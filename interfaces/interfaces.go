package interfaces

type InputType string

/*
interface for each game state struct to implement
*/
type Game interface {
	Players() []*Player
	JSON() ClientState
}

/*
interface for each game type's user input object to implement
*/
type Input interface {
	GameID() string
	PlayerIndex() int
	ChangeState(Game)
}

type Player struct {
	PlayerID    string
	GameID      string
	PlayerIndex int
	Username    string
}

/*
interface for each game type's json-sendable object to implement
*/
type ClientState interface{}
