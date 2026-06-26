package whiteboard

import (
	"bytes"
	"image"
	"image/color"
	"image/png"

	"github.com/geofpwhite/html_games_engine/IDGenerator"
	"github.com/geofpwhite/html_games_engine/interfaces"
)

type whiteboard struct {
	img        image.RGBA
	lastChange pixelChange
	players    []*interfaces.Player
}

type pixelChange struct {
	x, y  int
	color color.RGBA
}

type drawInput struct {
	gameID string
	x, y   int
	color  color.RGBA
}

func (di *drawInput) GameID() string {
	return di.gameID
}

func (di *drawInput) PlayerIndex() int {
	return -1
}

func (di *drawInput) ChangeState(gameObj interfaces.Game) {
	if gState, ok := gameObj.(*whiteboard); ok {
		gState.img.Set(di.x, di.y, di.color)
		gState.lastChange = pixelChange{x: di.x, y: di.y, color: di.color}
	}
}

func NewWhiteboard(w, h int) (*whiteboard, string) {
	wb := whiteboard{
		img: *image.NewRGBA(image.Rect(0, 0, w, h)),
	}
	return &wb, IDGenerator.GenerateID(6)
}

func (wb *whiteboard) Players() []*interfaces.Player {
	return wb.players
}

type whiteboardDelta struct {
	Type string `json:"type"`
	X    int    `json:"x"`
	Y    int    `json:"y"`
	R    uint8  `json:"r"`
	G    uint8  `json:"g"`
	B    uint8  `json:"b"`
	A    uint8  `json:"a"`
}

type whiteboardFull struct {
	Type string `json:"type"`
	Png  []byte `json:"png"`
}

func (wb *whiteboard) JSON() interfaces.ClientState {
	c := wb.lastChange
	return whiteboardDelta{
		Type: "delta",
		X:    c.x,
		Y:    c.y,
		R:    c.color.R,
		G:    c.color.G,
		B:    c.color.B,
		A:    c.color.A,
	}
}

func (wb *whiteboard) encodedPNG() ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, &wb.img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
