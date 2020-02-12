package display

import (
	"bytes"
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	luar "layeh.com/gopher-luar"

	runewidth "github.com/mattn/go-runewidth"
	lua "github.com/yuin/gopher-lua"
	"github.com/zyedidia/micro/internal/buffer"
	"github.com/zyedidia/micro/internal/config"
	ulua "github.com/zyedidia/micro/internal/lua"
	"github.com/zyedidia/micro/internal/screen"
	"github.com/zyedidia/micro/internal/util"
)

// StatusLine represents the information line at the bottom
// of each window
// It gives information such as filename, whether the file has been
// modified, filetype, cursor location
type StatusLine struct {
	Info map[string]func(*buffer.Buffer) string

	win *BufWindow
}

var statusInfo = map[string]func(*buffer.Buffer) string{
	"filename": func(b *buffer.Buffer) string {
		if b.Settings["basename"].(bool) {
			return path.Base(b.GetName())
		}
		return b.GetName()
	},
	"line": func(b *buffer.Buffer) string {
		return strconv.Itoa(b.GetActiveCursor().Y + 1)
	},
	"col": func(b *buffer.Buffer) string {
		return strconv.Itoa(b.GetActiveCursor().X + 1)
	},
	"modified": func(b *buffer.Buffer) string {
		if b.Modified() {
			return "+ "
		}
		if b.Type.Readonly {
			return "[ro] "
		}
		return ""
	},
}

func SetStatusInfoFnLua(fn string) {
	luaFn := strings.Split(fn, ".")
	if len(luaFn) <= 1 {
		return
	}
	plName, plFn := luaFn[0], luaFn[1]
	pl := config.FindPlugin(plName)
	if pl == nil {
		return
	}
	statusInfo[fn] = func(b *buffer.Buffer) string {
		if pl == nil || !pl.IsEnabled() {
			return ""
		}
		val, err := pl.Call(plFn, luar.New(ulua.L, b))
		if err == nil {
			if v, ok := val.(lua.LString); !ok {
				screen.TermMessage(plFn, "should return a string")
				return ""
			} else {
				return string(v)
			}
		}
		return ""
	}
}

// NewStatusLine returns a statusline bound to a window
func NewStatusLine(win *BufWindow) *StatusLine {
	s := new(StatusLine)
	s.win = win
	return s
}

// FindOpt finds a given option in the current buffer's settings
func (s *StatusLine) FindOpt(opt string) interface{} {
	if val, ok := s.win.Buf.Settings[opt]; ok {
		return val
	}
	return "null"
}

var formatParser = regexp.MustCompile(`\$\(.+?\)`)

// Display draws the statusline to the screen
func (s *StatusLine) Display() {
	// We'll draw the line at the lowest line in the window
	y := s.win.Height + s.win.Y - 1

	b := s.win.Buf
	// autocomplete suggestions (for the buffer, not for the infowindow)
	if b.HasSuggestions && len(b.Suggestions) > 1 {
		statusLineStyle := config.DefStyle.Reverse(true)
		if style, ok := config.Colorscheme["statusline"]; ok {
			statusLineStyle = style
		}
		keymenuOffset := 0
		if config.GetGlobalOption("keymenu").(bool) {
			keymenuOffset = len(keydisplay)
		}
		x := 0
		for j, sug := range b.Suggestions {
			style := statusLineStyle
			if b.CurSuggestion == j {
				style = style.Reverse(true)
			}
			for _, r := range sug {
				screen.SetContent(x, y-keymenuOffset, r, nil, style)
				x++
				if x >= s.win.Width {
					return
				}
			}
			screen.SetContent(x, y-keymenuOffset, ' ', nil, statusLineStyle)
			x++
			if x >= s.win.Width {
				return
			}
		}

		for x < s.win.Width {
			screen.SetContent(x, y-keymenuOffset, ' ', nil, statusLineStyle)
			x++
		}
		return
	}

	formatter := func(match []byte) []byte {
		name := match[2 : len(match)-1]
		if bytes.HasPrefix(name, []byte("opt")) {
			option := name[4:]
			return []byte(fmt.Sprint(s.FindOpt(string(option))))
		} else if bytes.HasPrefix(name, []byte("bind")) {
			binding := string(name[5:])
			for k, v := range config.Bindings {
				if v == binding {
					return []byte(k)
				}
			}
			return []byte("null")
		} else {
			if fn, ok := statusInfo[string(name)]; ok {
				return []byte(fn(s.win.Buf))
			}
			return []byte{}
		}
	}

	leftText := []byte(s.win.Buf.Settings["statusformatl"].(string))
	leftText = formatParser.ReplaceAllFunc(leftText, formatter)
	rightText := []byte(s.win.Buf.Settings["statusformatr"].(string))
	rightText = formatParser.ReplaceAllFunc(rightText, formatter)

	statusLineStyle := config.DefStyle.Reverse(true)
	if style, ok := config.Colorscheme["statusline"]; ok {
		statusLineStyle = style
	}

	leftLen := util.StringWidth(leftText, utf8.RuneCount(leftText), 1)
	rightLen := util.StringWidth(rightText, utf8.RuneCount(rightText), 1)

	winX := s.win.X
	for x := 0; x < s.win.Width; x++ {
		if x < leftLen {
			r, size := utf8.DecodeRune(leftText)
			leftText = leftText[size:]
			rw := runewidth.RuneWidth(r)
			for j := 0; j < rw; j++ {
				c := r
				if j > 0 {
					c = ' '
					x++
				}
				screen.SetContent(winX+x, y, c, nil, statusLineStyle)
			}
		} else if x >= s.win.Width-rightLen && x < rightLen+s.win.Width-rightLen {
			r, size := utf8.DecodeRune(rightText)
			rightText = rightText[size:]
			rw := runewidth.RuneWidth(r)
			for j := 0; j < rw; j++ {
				c := r
				if j > 0 {
					c = ' '
					x++
				}
				screen.SetContent(winX+x, y, c, nil, statusLineStyle)
			}
		} else {
			screen.SetContent(winX+x, y, ' ', nil, statusLineStyle)
		}
	}
}
