package imtui

import (
	"errors"
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"golang.org/x/xerrors"
)

var ErrNormalExit = errors.New("regular exit")

type Tui struct {
	s tcell.Screen

	scene func(*Tui) error

	CurrentKey *tcell.EventKey
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
	return t, nil
}

func (t *Tui) Run() error {
	err := t.scene(t)
	if err != nil {
		t.s.Fini()
		if xerrors.Is(err, ErrNormalExit) {
			return nil
		}
		return xerrors.Errorf("scene returned an error: %w", err)
	}

	for {
		switch ev := t.s.PollEvent().(type) {
		case *tcell.EventResize:
			t.s.Sync()
		case *tcell.EventKey:
			if ev.Key() == tcell.KeyCtrlC || ev.Key() == tcell.KeyEscape {
				t.s.Fini()
				return nil
			}
			t.s.Clear()
			t.CurrentKey = ev
			err := t.scene(t)
			if err != nil {
				t.s.Fini()
				if xerrors.Is(err, ErrNormalExit) {
					return nil
				}
				return xerrors.Errorf("scene returned an error: %w", err)
			}
			t.CurrentKey = nil
			t.s.Show()
		case nil:
			return xerrors.Errorf("tcell exited")
		}
	}
}

func (t *Tui) SetScene(s func(*Tui) error) {
	t.scene = s
}

func (t *Tui) Label(x, y int, text string, style tcell.Style) int {
	return t.emitStr(x, y, style, text)
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
	w := t.emitStr(x, y, style, str)
	return w
}

func (t *Tui) EditField(x, y, width int, text *string, style tcell.Style) int {
	return t.EditFieldFiltered(x, y, width, text, func(rune) bool { return true }, style)
}

func (t *Tui) emitStr(x, y int, style tcell.Style, str string) int {
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
	}
	return x - xinit
}
