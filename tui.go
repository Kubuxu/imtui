package imtui

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"golang.org/x/xerrors"
)

var ErrNormalExit = errors.New("regular exit")

type Tui struct {
	s tcell.Screen

	scene []func(*Tui) error

	CurrentKey   *tcell.EventKey
	forceRefresh bool
}

func NewTui() (*Tui, error) {
	s, err := tcell.NewScreen()
	if err != nil {
		return nil, xerrors.Errorf("new screen: %w", err)
	}
	if err := s.Init(); err != nil {
		return nil, xerrors.Errorf("init screen: %w", err)
	}

	s.SetStyle(tcell.StyleDefault)
	t := &Tui{s: s}
	t.PushScene(func(_ *Tui) error {
		return ErrNormalExit
	})
	return t, nil
}

func (t *Tui) curScene() func(*Tui) error {
	return t.scene[len(t.scene)-1]
}

func (t *Tui) Run() error {
	handleErr := func(err error) error {
		t.s.Fini()
		if xerrors.Is(err, ErrNormalExit) {
			return nil
		}
		return xerrors.Errorf("scene returned an error: %w", err)
	}

	err := t.curScene()(t)
	if err != nil {
		return handleErr(err)
	}

	for {
		switch ev := t.s.PollEvent().(type) {
		case *tcell.EventResize:
			t.s.Clear()
			t.scene[len(t.scene)-1](t)
			t.s.Sync()
		case *tcell.EventKey:
			if ev.Key() == tcell.KeyCtrlC {
				t.s.Fini()
				return nil
			}
			if ev.Key() == tcell.KeyEscape {
				if !t.PopScene() {
					t.s.Fini()
					return nil
				}
			}
			t.s.Clear()
			t.CurrentKey = ev
			err := t.curScene()(t)
			if err != nil {
				return handleErr(err)
			}
			t.CurrentKey = nil
			for t.forceRefresh {
				t.forceRefresh = false
				t.s.Clear()
				err := t.curScene()(t)
				if err != nil {
					return handleErr(err)
				}
			}
			t.s.Show()
		case nil:
			return xerrors.Errorf("tcell exited")
		}

	}
}

func (t *Tui) PushScene(s func(*Tui) error) {
	t.forceRefresh = true
	t.scene = append(t.scene, s)
}

func (t *Tui) PopScene() bool {
	t.forceRefresh = true
	if len(t.scene) != 0 {
		t.scene = t.scene[:len(t.scene)-1]
	}
	return len(t.scene) != 0
}

func (t *Tui) ReplaceScene(s func(*Tui) error) {
	t.PopScene()
	t.PushScene(s)
}

var colorRegex = regexp.MustCompile(`\[:(\w+):\]`)

func (t *Tui) Label(x, y int, text string, style tcell.Style) int {
	return t.LabelMax(x, 0, y, text, style)
}

func (t *Tui) LabelMax(x, xmax, y int, text string, style tcell.Style) int {
	matches := colorRegex.FindAllStringIndex(text, -1)
	if matches == nil {
		return t.emitStr(x, 0, y, style, text)
	}

	lastStyle := style
	curX := x
	lastIndx := 0

	for _, m := range matches {
		curX += t.emitStr(curX, xmax, y, lastStyle, text[lastIndx:m[0]])

		colStr := text[m[0]+2 : m[1]-2]
		lastIndx = m[1]

		if colStr == "default" {
			defFor, _, _ := style.Decompose()
			lastStyle = lastStyle.Foreground(defFor)
		} else if col := tcell.ColorNames[colStr]; col != tcell.ColorDefault {
			lastStyle = lastStyle.Foreground(col)
		} else {
			lastStyle = lastStyle.Foreground(tcell.ColorDarkRed)
		}
	}
	curX += t.emitStr(curX, xmax, y, lastStyle, text[lastIndx:])

	return curX - x
}

func FilterDecimal(r rune) bool {
	return (r >= '0' && r <= '9') || r == '.'
}

func (t *Tui) EditFieldFiltered(x, y, width int, text *string, filter func(rune) bool, style tcell.Style) int {
	if width < 1 {
		width = 1
	}

	curKey := t.CurrentKey
	if curKey != nil {
		newText := *text

		switch curKey.Key() {
		case tcell.KeyRune:
			if r := curKey.Rune(); filter(r) {
				newText = newText + string(curKey.Rune())
			}
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			if len(newText) != 0 {
				newText = newText[:len(newText)-1]
			}
		}

		*text = newText
	}

	str := fmt.Sprintf("%*s", width, *text)
	w := t.emitStr(x, 0, y, style, str)
	return w
}

func (t *Tui) EditField(x, y, width int, text *string, style tcell.Style) int {
	return t.EditFieldFiltered(x, y, width, text, func(rune) bool { return true }, style)
}

func (t *Tui) emitStr(x, xmax, y int, style tcell.Style, str string) int {
	if len(str) == 0 {
		return 0
	}
	if xmax != 0 && x >= xmax {
		return 0
	}

	xinit := x
	for _, c := range str {
		var comb []rune
		w := runewidth.RuneWidth(c)
		if w == 0 {
			comb = []rune{c}
			c = ' '
			w = 1
		}
		t.s.SetContent(x, y, c, comb, style)
		x += w
		if xmax != 0 && x >= xmax {
			break
		}
	}
	return x - xinit
}

func (t *Tui) FlexTable(y int, maxY, maxX int, sel, scroll *int, rows [][]string, flex []int, header bool) int {

	if len(rows) == 0 {
		return 0
	}

	headerOffSet := 0
	if header {
		headerOffSet = 1
	}

	if t.CurrentKey != nil {
		switch t.CurrentKey.Key() {
		case tcell.KeyUp:
			*sel--
		case tcell.KeyDown:
			*sel++
		}
	}

	if *sel < headerOffSet {
		*sel = headerOffSet
	} else if *sel >= len(rows) {
		*sel = len(rows) - 1
	}

	if maxY == 0 {
		_, maxY = t.s.Size()
	}
	if maxX == 0 {
		maxX, _ = t.s.Size()
	}

	if len(flex) != 0 {
		if len(rows[0]) != len(flex) {
			panic("misconfigured flex")
		}
	}
	flexSum := 0
	for i, f := range flex {
		if f == 0 {
			flex[i] = 1
			flexSum += 1
		} else {
			flexSum += f
		}
	}

	xPer := maxX / flexSum
	display := func(y, rowIdx int) {
		row := rows[rowIdx]
		if rowIdx == *sel {
			t.LabelMax(0, 0, y, ">", tcell.StyleDefault)
		}
		x := 1
		for colIdx, content := range row {
			xmax := x + flex[colIdx]*xPer
			c := content
			if colIdx != 0 {
				c = " " + c
			}
			t.LabelMax(x, xmax, y, c, tcell.StyleDefault)
			x = xmax
		}
	}

	curY := y
	if header {
		display(curY, 0)
		curY++
	}
	if len(rows) > 2 {
		if *sel-*scroll > maxY-y-2 && *sel < len(rows)-1 {
			*scroll = y - maxY + *sel + 2
		}
		if *sel-*scroll < 2 && *sel != 1 {
			*scroll = *sel - 2
		}
	}
	for rowIdx := headerOffSet + *scroll; rowIdx < len(rows) && curY < maxY; rowIdx++ {
		display(curY, rowIdx)
		curY++
	}

	return curY - y
}
