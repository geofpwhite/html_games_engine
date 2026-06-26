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
	lastChange drawChange
	players    []*interfaces.Player
}

type drawChange struct {
	x, y   int
	color  color.RGBA
	radius int
}

type drawInput struct {
	gameID string
	x, y   int
	color  color.RGBA
	radius int
}

func (di *drawInput) GameID() string {
	return di.gameID
}

func (di *drawInput) PlayerIndex() int {
	return -1
}

func (di *drawInput) ChangeState(gameObj interfaces.Game) {
	if gState, ok := gameObj.(*whiteboard); ok {
		bounds := gState.img.Bounds()
		r := di.radius
		for dx := -r; dx <= r; dx++ {
			for dy := -r; dy <= r; dy++ {
				if dx*dx+dy*dy <= r*r {
					px, py := di.x+dx, di.y+dy
					if px >= bounds.Min.X && px < bounds.Max.X && py >= bounds.Min.Y && py < bounds.Max.Y {
						gState.img.Set(px, py, di.color)
					}
				}
			}
		}
		gState.lastChange = drawChange{x: di.x, y: di.y, color: di.color, radius: di.radius}
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
	Type   string `json:"type"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	R      uint8  `json:"r"`
	G      uint8  `json:"g"`
	B      uint8  `json:"b"`
	A      uint8  `json:"a"`
	Radius int    `json:"radius"`
}

type whiteboardFull struct {
	Type string `json:"type"`
	Png  []byte `json:"png"`
}

func (wb *whiteboard) JSON() interfaces.ClientState {
	c := wb.lastChange
	return whiteboardDelta{
		Type:   "delta",
		X:      c.x,
		Y:      c.y,
		R:      c.color.R,
		G:      c.color.G,
		B:      c.color.B,
		A:      c.color.A,
		Radius: c.radius,
	}
}

func (wb *whiteboard) encodedPNG() ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, &wb.img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
