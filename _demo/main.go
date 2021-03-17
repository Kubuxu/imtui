package main

import (
	"fmt"
	"log"
	"strconv"

	"github.com/Kubuxu/imtui"
	"github.com/gdamore/tcell/v2"
)

func ui() func(*imtui.Tui) error {
	baseFee := 2.172 // in nFIL
	maxFee := 0.07 * 1e9
	gasLimit := 1000000000.0
	price := fmt.Sprintf("%.0f", maxFee)

	return func(t *imtui.Tui) error {
		defS := tcell.StyleDefault
		l := log.Default()
		_ = l

		row := 0
		t.Label(0, row, "Fee of the message is too low.", defS)
		row++

		t.Label(0, row, fmt.Sprintf("Current Base Fee is: %.1f nFIL", baseFee), defS)
		row++
		t.Label(0, row, fmt.Sprintf("Your configured maximum fee is: %.1f nFIL", maxFee), defS)
		row++
		t.Label(0, row, fmt.Sprintf("Required maximum fee for the message: %.1f nFIL", gasLimit*baseFee), defS)
		row++
		t.Label(0, row, "Press S to use it", defS)

		w := t.Label(0, row, "Current Price: ", defS)

		w += t.EditFieldFiltered(w, row, 14, &price, imtui.FilterDecimal, defS.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack))

		w += t.Label(w, row, " nFIL", defS)

		priceF, err := strconv.ParseFloat(price, 64)
		if err != nil {
			w += t.Label(w, row, " invalid price", defS.Foreground(tcell.ColorMaroon).Bold(true))
		} else if priceF > gasLimit*baseFee {
			w += t.Label(w, row, " OK", defS.Foreground(tcell.ColorDarkGreen).Bold(true))
		} else {
			w += t.Label(w, row, " too low", defS.Foreground(tcell.ColorRed).Bold(true))
		}

		return nil
	}
}

func main() {
	//encoding.Register()
	t, err := imtui.NewTui()
	if err != nil {
		panic(err)
	}

	t.SetScene(ui())

	err = t.Run()

	if err != nil {
		panic(err)
	}
}
