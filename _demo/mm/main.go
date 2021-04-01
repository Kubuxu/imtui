package main

import (
	"log"

	"github.com/Kubuxu/imtui"
	"github.com/gdamore/tcell/v2"
)

func ui() func(*imtui.Tui) error {

	rows := [][]string{
		{"CID", "To", "Value"},
		{"Kuba", "Kubuxu", "FIL"},
		{"Vyzo", "Vyzoooooooooooo", "libp2p"},
		{"Raul", "Lorem ipsum dolor sit amet, consectetur adipiscing elit", "Upper Managment"},
	}

	sel := 0
	scroll := 0

	return func(t *imtui.Tui) error {
		defS := tcell.StyleDefault
		_ = defS
		l := log.Default()
		_ = l

		t.FlexTable(0, 0, 0, &sel, &scroll, rows, []int{1, 3, 1}, true)
		return nil
	}
}

func main() {
	//encoding.Register()
	t, err := imtui.NewTui()
	if err != nil {
		panic(err)
	}

	t.PushScene(ui())

	err = t.Run()

	if err != nil {
		panic(err)
	}
}
