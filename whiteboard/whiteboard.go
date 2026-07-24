package whiteboard

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"

	IDGenerator "github.com/geofpwhite/html_games_engine/IDGenerator"
	"github.com/geofpwhite/html_games_engine/interfaces"
	"github.com/geofpwhite/paint"
)

type whiteboard struct {
	img        image.RGBA
	lastChange drawChange
	players    []*interfaces.Player
	needsFull  bool
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

func (di *drawInput) GameID() string   { return di.gameID }
func (di *drawInput) PlayerIndex() int { return -1 }
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

type clearInput struct {
	gameID string
}

func (ci *clearInput) GameID() string   { return ci.gameID }
func (ci *clearInput) PlayerIndex() int { return -1 }
func (ci *clearInput) ChangeState(gameObj interfaces.Game) {
	if gState, ok := gameObj.(*whiteboard); ok {
		fillWhite(&gState.img)
		gState.needsFull = true
	}
}

type lineInput struct {
	gameID         string
	x1, y1, x2, y2 int
	clr            color.RGBA
	thickness      int
}

func (li *lineInput) GameID() string   { return li.gameID }
func (li *lineInput) PlayerIndex() int { return -1 }
func (li *lineInput) ChangeState(gameObj interfaces.Game) {
	if gState, ok := gameObj.(*whiteboard); ok {
		paint.DrawLine(&gState.img,
			paint.Coords{
				X: li.x1,
				Y: li.y1,
			},
			paint.Coords{
				X: li.x2, Y: li.y2,
			}, li.clr, li.thickness, false)
		gState.needsFull = true
	}
}

type rectInput struct {
	gameID         string
	x1, y1, x2, y2 int
	clr            color.RGBA
	thickness      int
	thetaDeg       float64
}

func (ri *rectInput) GameID() string   { return ri.gameID }
func (ri *rectInput) PlayerIndex() int { return -1 }
func (ri *rectInput) ChangeState(gameObj interfaces.Game) {
	if gState, ok := gameObj.(*whiteboard); ok {
		if ri.thetaDeg != 0 {
			theta := ri.thetaDeg * math.Pi / 180
			paint.DrawRotatedRectangle(
				&gState.img,
				paint.Coords{
					X: ri.x1,
					Y: ri.y1,
				},
				paint.Coords{
					X: ri.x2,
					Y: ri.y2,
				},
				theta,
				ri.clr,
				ri.thickness,
				false)
		} else {
			paint.DrawRectangle(
				&gState.img,
				paint.Coords{
					X: ri.x1, Y: ri.y1,
				},
				paint.Coords{
					X: ri.x2, Y: ri.y2,
				},
				ri.clr, ri.thickness, false)
		}
		gState.needsFull = true
	}
}

type circleInput struct {
	gameID string
	x, y   int
	radius int
	clr    color.RGBA
	filled bool
}

func (ci *circleInput) GameID() string   { return ci.gameID }
func (ci *circleInput) PlayerIndex() int { return -1 }
func (ci *circleInput) ChangeState(gameObj interfaces.Game) {
	if gState, ok := gameObj.(*whiteboard); ok {
		center := paint.Coords{X: ci.x, Y: ci.y}
		if ci.filled {
			paint.DrawFilledCircle(&gState.img, ci.clr, ci.radius, center)
		} else {
			paint.DrawCircle(&gState.img, ci.clr, ci.radius, center)
		}
		gState.needsFull = true
	}
}

func fillWhite(img *image.RGBA) {
	b := img.Bounds()
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			img.SetRGBA(x, y, white)
		}
	}
}

func NewWhiteboard(w, h int) (*whiteboard, string) {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	fillWhite(img)
	wb := whiteboard{img: *img}
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
	if wb.needsFull {
		wb.needsFull = false
		if pngBytes, err := wb.encodedPNG(); err == nil {
			return whiteboardFull{Type: "full", Png: pngBytes}
		}
	}
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
