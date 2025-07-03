package main

import (
	"fmt"

	"github.com/ofabricio/imtui"
)

func main() {

	tui := imtui.New()

	var one, two bool
	var radio int = -1
	var clicks int
	var toggle bool
	for range tui.Loop() {

		if tui.Button(" Button ") {
			clicks++
		}
		tui.Text(fmt.Sprintf(" Button clicked %v times", clicks))

		tui.Break()
		if tui.Toggle(" Toggle ", &toggle); toggle {
			tui.Text(" Toggled ")
		}

		tui.Break()
		tui.Radio("One ", 0, &radio)
		tui.Radio("Two ", 1, &radio)
		tui.Text(fmt.Sprintf(" Selected: %v ", radio))

		tui.Break()
		tui.Check("One ", &one)
		tui.Check("Two ", &two)
		tui.Text(fmt.Sprintf(" One: %v, Two: %v ", one, two))
	}
}
