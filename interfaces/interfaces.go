package interfaces

type InputType string

// Priority values for the input/output priority channels. Higher values are
// popped first, so closing inputs/outputs jump ahead of normal game traffic.
const (
	PriorityNormal = 0
	PriorityClose  = 100
)

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
	// Priority reports this input's queue priority (see PriorityNormal/PriorityClose).
	Priority() int
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
