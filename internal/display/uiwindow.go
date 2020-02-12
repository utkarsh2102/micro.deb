package display

import (
	"github.com/zyedidia/micro/internal/buffer"
	"github.com/zyedidia/micro/internal/config"
	"github.com/zyedidia/micro/internal/screen"
	"github.com/zyedidia/micro/internal/views"
)

type UIWindow struct {
	root *views.Node
}

func NewUIWindow(n *views.Node) *UIWindow {
	uw := new(UIWindow)
	uw.root = n
	return uw
}

func (w *UIWindow) drawNode(n *views.Node) {
	cs := n.Children()
	dividerStyle := config.DefStyle
	if style, ok := config.Colorscheme["divider"]; ok {
		dividerStyle = style
	}

	for i, c := range cs {
		if c.IsLeaf() && c.Kind == views.STVert {
			if i != len(cs)-1 {
				for h := 0; h < c.H; h++ {
					screen.SetContent(c.X+c.W, c.Y+h, '|', nil, dividerStyle.Reverse(true))
				}
			}
		} else {
			w.drawNode(c)
		}
	}
}

func (w *UIWindow) Display() {
	w.drawNode(w.root)
}

func (w *UIWindow) GetMouseSplitID(vloc buffer.Loc) uint64 {
	var mouseLoc func(*views.Node) uint64
	mouseLoc = func(n *views.Node) uint64 {
		cs := n.Children()
		for i, c := range cs {
			if c.Kind == views.STVert {
				if i != len(cs)-1 {
					if vloc.X == c.X+c.W && vloc.Y >= c.Y && vloc.Y < c.Y+c.H {
						return c.ID()
					}
				}
			} else if c.Kind == views.STHoriz {
				if i != len(cs)-1 {
					if vloc.Y == c.Y+c.H-1 && vloc.X >= c.X && vloc.X < c.X+c.W {
						return c.ID()
					}
				}
			}
		}
		for _, c := range cs {
			m := mouseLoc(c)
			if m != 0 {
				return m
			}
		}
		return 0
	}
	return mouseLoc(w.root)
}
func (w *UIWindow) Resize(width, height int) {}
func (w *UIWindow) SetActive(b bool)         {}
