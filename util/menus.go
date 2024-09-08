package util

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"slices"
	"strconv"

	"github.com/buger/goterm"
	"github.com/inancgumus/screen"
	"github.com/nexidian/gocliselect"

	"github.com/g026r/pocket-library-editor/model"
)

type Application struct {
	Entries   []model.Entry
	PlayTimes map[uint32]model.PlayTime
}

func (a *Application) Main() string {
	clearScreen()
	menu := gocliselect.NewMenu("Analogue Pocket Library Editor")
	menu.AddItem("Add Entry", "add")
	menu.AddItem("Edit Entry", "edit")
	menu.AddItem("Remove Entry", "remove")
	menu.AddItem("Save & Quit", "save")
	menu.AddItem("Quit without Saving", "quit")

	return menu.Display(false)
}

func (a *Application) Add() string {
	clearScreen()
	return ""
}

func (a *Application) Edit() {
	clone := slices.Clone(a.Entries) // For cancel
	start := 0
	for {
		end := min(start+10, len(a.Entries))
		switch x := a.displayEntries("Edit", start, end); x {
		case "prev":
			start = max(0, start-10)
		case "next":
			start = min(start+10, len(a.Entries)-len(a.Entries)%10)
		case "cancel", "":
			a.Entries = clone
			return
		case "done":
			return
		default:
			i, err := strconv.Atoi(x)
			if err != nil {
				log.Fatal(err)
			}
			a.Entries[i] = editEntry(a.Entries[i])
			slices.SortFunc(a.Entries, model.SortFunc)
		}
	}
}

func (a *Application) Remove() {
	clone := slices.Clone(a.Entries) // For cancel, since slices.Delete directly modifies the underlying slice
	start := 0
	for {
		end := min(start+10, len(a.Entries))
		switch x := a.displayEntries("Delete", start, end); x {
		case "prev":
			start = max(0, start-10)
		case "next":
			start = min(start+10, len(a.Entries)-len(a.Entries)%10)
		case "cancel", "":
			a.Entries = clone
			return
		case "done":
			return
		default:
			i, err := strconv.Atoi(x)
			if err != nil {
				log.Fatal(err)
			}
			slices.Delete(a.Entries, i, i+1)
		}
	}
}

func (a *Application) displayEntries(title string, start, end int) string {
	clearScreen()

	menu := gocliselect.NewMenu(fmt.Sprintf("%s Entries [%d-%d]", title, start+1, end))

	for i := start; i < end; i++ {
		menu.AddItem(fmt.Sprintf("%d. %s", i+1, a.Entries[i].Name), strconv.Itoa(i))
	}

	if start != 0 {
		menu.AddItem(fmt.Sprintf("<- %d-%d", start-9, start), "prev")
	}
	if end < len(a.Entries) {
		menu.AddItem(fmt.Sprintf("%d-%d ->", end+1, min(end+10, len(a.Entries))), "next")
	}

	menu.AddItem("Cancel", "cancel")
	menu.AddItem("Done", "done")

	return menu.Display(true)
}

func editEntry(entry model.Entry) model.Entry {
	clearScreen()
	//clone := entry

	fmt.Printf("%s\n", goterm.Color(goterm.Bold("Edit Entry:")+":", goterm.CYAN))
	fmt.Printf("%s\n", goterm.Color("(Return to accept defaults)", goterm.CYAN))

	in := bufio.NewScanner(os.Stdin)
	fmt.Printf("\rName (%s): ", entry.Name)
	in.Scan()
	if s := in.Text(); s != "" {
		entry.Name = s
	}

	fmt.Printf("\rSystem (%s): ", entry.System.String())
	in.Scan()
	if s := in.Text(); s != "" {
		h, err := model.Parse(s)
		if err != nil {
			log.Fatal(err)
		}
		entry.System = h
	}

	fmt.Printf("\rCRC32 (%x): ", entry.Crc32)
	in.Scan()
	if s := in.Text(); s != "" {
		if len(s) > 8 {
			log.Fatal("input too long")
		} else if len(s) < 8 {
			for len(s) < 8 {
				s = "0" + s
			}
		}
		h, err := hex.DecodeString(s)
		if err != nil {
			log.Fatal(err)
		}
		entry.Crc32 = binary.BigEndian.Uint32(h)
	}
	fmt.Printf("\rHash (%x): ", entry.Hash)
	in.Scan()
	if s := in.Text(); s != "" {
		if s := in.Text(); s != "" {
			if len(s) > 8 {
				log.Fatal("input too long")
			} else if len(s) < 8 {
				for len(s) < 8 {
					s = "0" + s
				}
			}
			h, err := hex.DecodeString(s)
			if err != nil {
				log.Fatal(err)
			}
			entry.Hash = binary.BigEndian.Uint32(h)
		}
	}
	fmt.Printf("\rUnknown (%x): ", entry.Unknown)
	in.Scan()
	if s := in.Text(); s != "" {
		if s := in.Text(); s != "" {
			if len(s) > 8 {
				log.Fatal("input too long")
			} else if len(s) < 8 {
				for len(s) < 8 {
					s = "0" + s
				}
			}
			h, err := hex.DecodeString(s)
			if err != nil {
				log.Fatal(err)
			}
			entry.Unknown = binary.BigEndian.Uint32(h)
		}
	}

	return entry
}

// clearScreen clears the screen & moves the cursor back to the top left
func clearScreen() {
	screen.Clear()
	screen.MoveTopLeft()
}
