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
	// ConsumeWinners returns the players who won a round since the last call, then
	// clears the pending list. Only ever called from the GameLoop goroutine.
	ConsumeWinners() []*Player
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
	// GameSessionID is the GameSessions row tracking this player's current
	// connection, or 0 if the player isn't logged in / isn't being tracked.
	GameSessionID int32
}

/*
ClientState interface for each game type's json-sendable object to implement
*/
type ClientState any
