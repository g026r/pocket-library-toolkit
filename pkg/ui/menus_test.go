package ui

import (
	"os"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/g026r/pocket-toolkit/pkg/io"
)

func Test_GenerateGameList(t *testing.T) {
	// Simple test to make certain that I remembered to reset everything when building a new list

	// Set up a new list to be replaced
	sut := list.New(make([]list.Item, 76), list.DefaultDelegate{}, 0, 0)
	sut, _ = sut.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}) // Set it into filtering mode
	sut, _ = sut.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}) // Set a filter value
	sut.Select(42)
	if !sut.SettingFilter() || sut.Index() != 42 || sut.Height() != 0 || sut.Width() != 0 || len(sut.Items()) != 76 || sut.FilterValue() != "a" {
		t.Fatal("List is not properly set up")
	}

	title := "This is a test title"
	width := 1337
	height := 2600
	e, err := io.LoadEntries(os.DirFS("../../testdata/valid"))
	if err != nil {
		t.Fatal(err)
	}

	// Generate the new list & compare it to the expected values
	sut = generateGameList(sut, e, title, width, height)
	if len(sut.Items()) != len(e) {
		t.Errorf("Expected %d items; got %d", len(e), len(sut.Items()))
	}
	if sut.Title != title {
		t.Errorf("Expected %s; got %s", title, sut.Title)
	}
	if sut.Width() != width {
		t.Errorf("Expected %d; got %d", width, sut.Width())
	}
	if sut.Height() != height {
		t.Errorf("Expected %d; got %d", height, sut.Height())
	}
	if sut.Index() != 0 {
		t.Errorf("Expected %d; got %d", 0, sut.Index())
	}
	if sut.FilterState() != list.Unfiltered {
		t.Errorf("Expected filter state to be %v. Got %v", list.Unfiltered, sut.FilterState())
	}
	if sut.FilterValue() != "" {
		t.Errorf("Expected filter value to be empty. Got %q", sut.FilterValue())
	}
}
