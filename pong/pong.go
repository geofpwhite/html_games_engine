// Package pong implements a real-time pong game mode. Unlike the other game
// modes in this repo, which are turn-based and driven by the shared
// input/output priority channels in package engine, pong has a ball that
// keeps moving between player inputs. That needs a server-side tick loop
// pushing state on its own schedule, which is what the bidi gRPC/Connect
// stream in service.go is for.
package pong

import (
	"errors"
	"math"
	"math/rand"
	"sync"
	"time"

	IDGenerator "github.com/geofpwhite/html_games_engine/IDGenerator"
)

const (
	fieldWidth   = 100.0
	fieldHeight  = 100.0
	paddleHeight = 20.0
	paddleDepth  = 2.0
	paddleSpeed  = 60.0 // units per second
	ballSpeed    = 50.0 // units per second

	tickInterval = time.Second / 60
)

// ErrGameNotFound is returned when joining a game ID that doesn't exist.
var ErrGameNotFound = errors.New("pong: game not found")

// ErrGameFull is returned when trying to join a game that already has two players.
var ErrGameFull = errors.New("pong: game already has two players")

type direction int

const (
	dirStop direction = iota
	dirUp
	dirDown
)

// State is a transport-agnostic snapshot of a game, for the service layer to
// translate into wire messages.
type State struct {
	BallX, BallY       float64
	Paddle1Y, Paddle2Y float64
	Score1, Score2     int
}

// game is one running match. All mutable fields are guarded by mu since the
// physics loop, and both players' input goroutines, touch it concurrently.
type game struct {
	mu sync.Mutex

	ballX, ballY   float64
	ballVX, ballVY float64
	paddle1Y       float64
	paddle2Y       float64
	dir1, dir2     direction
	score1, score2 int
	players        int

	stop chan struct{}
}

func newGame() *game {
	g := &game{
		paddle1Y: fieldHeight / 2,
		paddle2Y: fieldHeight / 2,
		stop:     make(chan struct{}),
	}
	g.resetBall(randSign())
	go g.run()
	return g
}

func randSign() float64 {
	if rand.Intn(2) == 0 { //nolint:gosec // not security-sensitive, just picking a serve direction
		return -1
	}
	return 1
}

// resetBall centers the ball and serves it off at a random angle, towards
// player 1's side if towards < 0 or player 2's side otherwise. Caller must
// hold mu, except from newGame where the game isn't shared yet.
func (g *game) resetBall(towards float64) {
	g.ballX = fieldWidth / 2
	g.ballY = fieldHeight / 2
	angle := (rand.Float64()*2 - 1) * math.Pi / 4 //nolint:gosec // just picking a serve angle
	vx := math.Cos(angle) * ballSpeed
	if towards < 0 {
		vx = -vx
	}
	g.ballVX = vx
	g.ballVY = math.Sin(angle) * ballSpeed
}

// tryAddPlayer atomically checks for a free slot and claims it, so two
// concurrent joins on the same game can't both succeed as "player 1".
func (g *game) tryAddPlayer() (idx int, ok bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.players >= 2 {
		return 0, false
	}
	idx = g.players
	g.players++
	return idx, true
}

func (g *game) removePlayer() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.players--
	return g.players
}

// setDirection records the requested paddle direction for the given player (0 or 1).
func (g *game) setDirection(playerIndex int, d direction) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if playerIndex == 0 {
		g.dir1 = d
	} else {
		g.dir2 = d
	}
}

func (g *game) snapshot() State {
	g.mu.Lock()
	defer g.mu.Unlock()
	return State{
		BallX: g.ballX, BallY: g.ballY,
		Paddle1Y: g.paddle1Y, Paddle2Y: g.paddle2Y,
		Score1: g.score1, Score2: g.score2,
	}
}

func (g *game) run() {
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()
	last := time.Now()
	for {
		select {
		case <-g.stop:
			return
		case now := <-ticker.C:
			dt := now.Sub(last).Seconds()
			last = now
			g.step(dt)
		}
	}
}

func (g *game) step(dt float64) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Hold the game at kickoff until both players are present, so an idle
	// game doesn't rack up a score (or move the ball at all) while whoever
	// created it is still waiting for an opponent to open the link.
	if g.players < 2 {
		return
	}

	g.paddle1Y = movePaddle(g.paddle1Y, g.dir1, dt)
	g.paddle2Y = movePaddle(g.paddle2Y, g.dir2, dt)

	g.ballX += g.ballVX * dt
	g.ballY += g.ballVY * dt

	if g.ballY <= 0 || g.ballY >= fieldHeight {
		g.ballVY = -g.ballVY
		g.ballY = clamp(g.ballY, 0, fieldHeight)
	}

	switch {
	case g.ballX <= paddleDepth && g.ballVX < 0:
		if withinPaddle(g.ballY, g.paddle1Y) {
			g.ballVX = -g.ballVX
			g.ballX = paddleDepth
		} else if g.ballX <= 0 {
			g.score2++
			g.resetBall(1)
		}
	case g.ballX >= fieldWidth-paddleDepth && g.ballVX > 0:
		if withinPaddle(g.ballY, g.paddle2Y) {
			g.ballVX = -g.ballVX
			g.ballX = fieldWidth - paddleDepth
		} else if g.ballX >= fieldWidth {
			g.score1++
			g.resetBall(-1)
		}
	}
}

func withinPaddle(ballY, paddleY float64) bool {
	return ballY >= paddleY-paddleHeight/2 && ballY <= paddleY+paddleHeight/2
}

func movePaddle(y float64, d direction, dt float64) float64 {
	switch d {
	case dirUp:
		y -= paddleSpeed * dt
	case dirDown:
		y += paddleSpeed * dt
	case dirStop:
	}
	return clamp(y, paddleHeight/2, fieldHeight-paddleHeight/2)
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// Registry tracks in-progress games by ID.
type Registry struct {
	mu    sync.Mutex
	games map[string]*game
}

func NewRegistry() *Registry {
	return &Registry{games: make(map[string]*game)}
}

// Create allocates a new, empty game and returns its ID, without attaching
// either caller as a player. This is what the HTTP new_game endpoint uses,
// so the game exists by the time either player's stream calls Join with it.
func (reg *Registry) Create() string {
	reg.mu.Lock()
	defer reg.mu.Unlock()

	id := IDGenerator.GenerateID(6)
	for reg.games[id] != nil {
		id = IDGenerator.GenerateID(6)
	}
	reg.games[id] = newGame()
	return id
}

// Join creates a new game (and its ID) if gameID is empty, or attaches the
// caller to an existing game as a player. It returns the game, the caller's
// player index (0 or 1), and the game's ID.
func (reg *Registry) Join(gameID string) (g *game, playerIndex int, id string, err error) {
	if gameID == "" {
		gameID = reg.Create()
	}

	reg.mu.Lock()
	g, ok := reg.games[gameID]
	reg.mu.Unlock()
	if !ok {
		return nil, 0, "", ErrGameNotFound
	}
	idx, ok := g.tryAddPlayer()
	if !ok {
		return nil, 0, "", ErrGameFull
	}
	return g, idx, gameID, nil
}

// Leave detaches a player from the game, stopping and removing it once both
// players are gone.
func (reg *Registry) Leave(gameID string, g *game) {
	if g.removePlayer() > 0 {
		return
	}
	close(g.stop)
	reg.mu.Lock()
	delete(reg.games, gameID)
	reg.mu.Unlock()
}
