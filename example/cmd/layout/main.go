package main

import (
	"github.com/dytlzl/tervi/pkg/tui"
)

func main() {
	err := tui.Run(rootView)
	if err != nil {
		panic(err)
	}
}

func rootView() *tui.View {
	return tui.VStack(
		tui.HStack(
			tui.VStack(
				tui.HStack(
					tui.VStack().Border(),
					tui.VStack().Border(),
				),
				tui.HStack().Border(),
			),
			tui.VStack().Border(),
		),
		tui.HStack().Border(),
	).Border().Title("Layout")
}