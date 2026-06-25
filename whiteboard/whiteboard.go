package whiteboard

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"log"

	"github.com/geofpwhite/html_games_engine/IDGenerator"
	"github.com/geofpwhite/html_games_engine/interfaces"
)

type whiteboard struct {
	img     image.RGBA
	buf     bytes.Buffer
	bytes   []byte
	len     int32
	players []*interfaces.Player
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
		gState.draw(di.x, di.y, di.color)
		if err := png.Encode(&gState.buf, &gState.img); err != nil {
			log.Fatalf("failed to encode image: %v", err)
		}
		gState.bytes = gState.buf.Bytes()
		gState.len = int32(len(gState.bytes))
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

type whiteboardObj struct {
	Length int32  `json:"Length"`
	Png    []byte `json:"Png"`
}

func (wb *whiteboard) JSON() interfaces.ClientState {
	return whiteboardObj{Length: wb.len, Png: wb.bytes}
}

func (wb *whiteboard) draw(x, y int, color color.RGBA) {
	wb.img.Set(x, y, color)
}
