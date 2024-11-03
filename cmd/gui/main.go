package main

import (
	"log"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/g026r/pocket-library-toolkit/pkg/ui"
)

func main() {
	if _, err := tea.NewProgram(ui.NewModel(), tea.WithAltScreen()).Run(); err != nil {
		log.Fatal(err)
	}
}
