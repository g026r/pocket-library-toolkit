package main

import (
	"log"

	tea "github.com/charmbracelet/bubbletea"

	model2 "github.com/g026r/pocket-library-editor/pkg/ui"
)

func main() {
	if _, err := tea.NewProgram(model2.NewModel(), tea.WithAltScreen()).Run(); err != nil {
		log.Fatal(err)
	}
}
