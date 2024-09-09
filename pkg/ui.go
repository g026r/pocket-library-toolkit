package pkg

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"slices"
	"strconv"

	"github.com/buger/goterm"
	"github.com/inancgumus/screen"

	"github.com/g026r/pocket-library-editor/pkg/model"
	"github.com/nexidian/gocliselect"
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

func (a *Application) Add() {
	clearScreen()
	entry := model.Entry{}

	// Start with the system menu since this will otherwise clear the screen
	sys := gocliselect.NewMenu("Add New Entry")
	sys.AddItem("Game Boy", "GB")
	sys.AddItem("Game Boy Color", "GBC")
	sys.AddItem("Game Boy Advance", "GBA")
	sys.AddItem("Game Gear", "GG")
	sys.AddItem("Sega Master System", "SMS")
	sys.AddItem("Neo Geo Pocket", "NGP")
	sys.AddItem("Neo Geo Pocket Color", "NGPC")
	sys.AddItem("TurboGrafx 16", "PCE")
	sys.AddItem("Atari Lynx", "Lynx")
	system := sys.Display(false)
	if system == "" { // ESC or Ctrl-C pressed
		return
	}
	if s, err := model.Parse(system); err != nil {
		log.Fatal(err)
	} else {
		entry.System = s
	}

	clearScreen()
	fmt.Printf("%s\n", goterm.Color(goterm.Bold(
		fmt.Sprintf("New Entry (%s)", entry.System.String()),
	)+":", goterm.CYAN))

	in := bufio.NewScanner(os.Stdin)
	fmt.Print("\n\rName: ")
	in.Scan()
	entry.Name = in.Text()

	fmt.Print("\rCRC32: ")
	in.Scan()
	if h, err := hexStringTransform(in.Text()); err != nil {
		log.Fatal(err)
	} else {
		entry.Crc32 = h
	}

	fmt.Print("\rSignature: ")
	in.Scan()
	if h, err := hexStringTransform(in.Text()); err != nil {
		log.Fatal(err)
	} else {
		entry.Sig = h
	}

	fmt.Print("\rUnknown Value (Just leave this blank): ")
	in.Scan()
	if h, err := hexStringTransform(in.Text()); err != nil {
		log.Fatal(err)
	} else {
		entry.Unknown = h
	}

	a.Entries = append(a.Entries, entry)
	slices.SortFunc(a.Entries, model.SortFunc)
}

func (a *Application) Edit() {
	a.pagedEntries("Edit", func(i int) {
		a.Entries[i] = editEntry(a.Entries[i])
		slices.SortFunc(a.Entries, model.SortFunc)
	})
}

func (a *Application) Remove() {
	a.pagedEntries("Delete", func(i int) {
		slices.Delete(a.Entries, i, i+1)
	})
}

func (a *Application) pagedEntries(title string, f func(i int)) {
	clone := slices.Clone(a.Entries) // For cancel, since slices.Delete directly modifies the underlying slice
	start := 0
	for {
		end := min(start+10, len(a.Entries))
		switch x := a.displayEntries(title, start, end); x {
		case "prev":
			if newStart := max(0, start-10); newStart == start {
				fmt.Printf("%c", 7) // We're at the first page. Ring the bell
			} else {
				start = newStart
			}
		case "next":
			if newStart := min(start+10, len(a.Entries)-len(a.Entries)%10); start == newStart {
				fmt.Printf("%c", 7) // We're at the last page. Ring the bell
			} else {
				start = newStart
			}
		case "done":
			return
		case "":
			a.Entries = clone // Restore the entries to the original copy
			return
		default:
			i, err := strconv.Atoi(x)
			if err != nil {
				log.Fatal(err)
			}
			f(i)
		}
	}
}

// displayEntries is a simple function that uses gocliselect to fake multipage menus
func (a *Application) displayEntries(title string, start, end int) string {
	clearScreen()

	menu := gocliselect.NewMenu(fmt.Sprintf("%s Entry [%d-%d]", title, start+1, end))

	for i := start; i < end; i++ {
		menu.AddItem(fmt.Sprintf("%d. %s", i+1, a.Entries[i].Name), strconv.Itoa(i))
	}

	if start != 0 {
		menu.AddItem(fmt.Sprintf("<- %d-%d", max(start-9, 0), start), "prev")
	}
	if end < len(a.Entries) {
		menu.AddItem(fmt.Sprintf("%d-%d ->", end+1, min(end+10, len(a.Entries))), "next")
	}

	menu.AddItem("Cancel", "")
	menu.AddItem("Done", "done")

	return menu.Display(true)
}

func editEntry(entry model.Entry) model.Entry {
	clearScreen()

	fmt.Printf("%s\n", goterm.Color(goterm.Bold("Edit Entry")+":", goterm.CYAN))
	fmt.Printf("%s\n", goterm.Color("(Return to accept defaults)", goterm.CYAN))

	in := bufio.NewScanner(os.Stdin)
	fmt.Printf("\rName (%s): ", entry.Name)
	in.Scan()
	if s := in.Text(); s != "" {
		entry.Name = s
	}

	//fmt.Printf("\rSystem (%s): ", entry.System.String())
	//in.Scan()
	//if s := in.Text(); s != "" {
	//	h, err := model.Parse(s)
	//	if err != nil {
	//		log.Fatal(err)
	//	}
	//	entry.System = h
	//}

	fmt.Printf("\rCRC32 (%08x): ", entry.Crc32)
	in.Scan()
	if s := in.Text(); s != "" {
		h, err := hexStringTransform(s)
		if err != nil {
			log.Fatal(err)
		}
		entry.Crc32 = h
	}
	// TODO: This seems a bit unsafe. Should it be enabled?
	//fmt.Printf("\rSignature (%08x): ", entry.Sig)
	//in.Scan()
	//if s := in.Text(); s != "" {
	//	h, err := hexStringTransform(s)
	//	if err != nil {
	//		log.Fatal(err)
	//	}
	//	entry.Hash = h
	//}
	//fmt.Printf("\rUnknown (%08x): ", entry.Unknown)
	//in.Scan()
	//if s := in.Text(); s != "" {
	//	h, err := hexStringTransform(s)
	//	if err != nil {
	//		log.Fatal(err)
	//	}
	//	entry.Unknown = h
	//}
	//}

	return entry
}

func hexStringTransform(s string) (uint32, error) {
	// String should be exactly 32 bits. We can pad it out if too short, but can't handle too long.
	if len(s) > 8 {
		return 0, errors.New("hex string too long")
	} else if len(s) < 8 {
		s = fmt.Sprintf("%08s", s) // binary.BigEndian.Uint32 fails if not padded out to 32 bits
	}

	h, err := hex.DecodeString(s)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint32(h), nil
}

// clearScreen clears the screen & moves the cursor back to the top left
func clearScreen() {
	screen.Clear()
	screen.MoveTopLeft()
}
