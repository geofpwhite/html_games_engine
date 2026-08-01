package pong

import (
	"context"
	"errors"
	"io"
	"time"

	pongv1 "github.com/geofpwhite/html_games_engine/pong/pongv1"

	"connectrpc.com/connect"
)

// Service implements the generated pongv1connect.PongServiceHandler
// interface: one call to Play per connected player.
type Service struct {
	registry *Registry
}

// Registry exposes the service's game registry so HTTP routes (new_game)
// can pre-create games that a player's stream then joins by ID.
func (s *Service) Registry() *Registry {
	return s.registry
}

func NewService() *Service {
	return &Service{registry: NewRegistry()}
}

// Play holds one player's bidi stream open for the lifetime of a match: an
// input-reading goroutine feeds paddle moves into the shared game, while
// this goroutine pushes a state snapshot down the stream on every physics
// tick, independent of whether the player has sent anything.
func (s *Service) Play(ctx context.Context, stream *connect.BidiStream[pongv1.ClientMessage, pongv1.ServerMessage]) error {
	first, err := stream.Receive()
	if err != nil {
		return err
	}
	join := first.GetJoin()
	if join == nil {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("first message on the stream must be a join request"))
	}

	g, playerIndex, gameID, err := s.registry.Join(join.GetGameId())
	if err != nil {
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	defer s.registry.Leave(gameID, g)

	if err := stream.Send(&pongv1.ServerMessage{
		Payload: &pongv1.ServerMessage_Joined{
			Joined: &pongv1.JoinResponse{GameId: gameID, PlayerIndex: int32(playerIndex)},
		},
	}); err != nil {
		return err
	}

	recvErr := make(chan error, 1)
	go func() {
		for {
			msg, err := stream.Receive()
			if err != nil {
				recvErr <- err
				return
			}
			if input := msg.GetInput(); input != nil {
				g.setDirection(playerIndex, directionFromProto(input.GetDirection()))
			}
		}
	}()

	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-recvErr:
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		case <-ticker.C:
			if err := stream.Send(&pongv1.ServerMessage{
				Payload: &pongv1.ServerMessage_State{State: stateToProto(g.snapshot())},
			}); err != nil {
				return err
			}
		}
	}
}

func directionFromProto(d pongv1.Direction) direction {
	switch d {
	case pongv1.Direction_DIRECTION_UP:
		return dirUp
	case pongv1.Direction_DIRECTION_DOWN:
		return dirDown
	case pongv1.Direction_DIRECTION_STOP, pongv1.Direction_DIRECTION_UNSPECIFIED:
	}
	return dirStop
}

func stateToProto(s State) *pongv1.GameState {
	return &pongv1.GameState{
		BallX:    s.BallX,
		BallY:    s.BallY,
		Paddle1Y: s.Paddle1Y,
		Paddle2Y: s.Paddle2Y,
		Score1:   int32(s.Score1),
		Score2:   int32(s.Score2),
	}
}
