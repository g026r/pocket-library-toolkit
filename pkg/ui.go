package pkg

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/buger/goterm"
	"github.com/pkg/term"

	"github.com/g026r/pocket-library-editor/pkg/model"
	"github.com/g026r/pocket-library-editor/pkg/util"
	"github.com/nexidian/gocliselect"
)

const (
	firstLibraryAddr uint32 = 0x4010
	firstThumbsAddr  uint32 = 0x1000C
)

type Application struct {
	RootDir   fs.FS
	Entries   []model.Entry
	PlayTimes map[uint32]model.PlayTime
	Thumbs    map[util.System]model.Thumbnails
	Config
	Internal map[util.System][]model.Entry // Internal is a map of all known possible entries, grouped by system
}

type Config struct {
	RemoveImages    bool
	AdvancedEditing bool
	ShowAdd         bool
}

func (a *Application) Run() {
	menu := gocliselect.NewMenu("Analogue Pocket Library Tool", false)

	menu.AddItem("Library", "lib")
	menu.AddItem("Thumbnails", "thumb")
	menu.AddItem("Settings", "config")
	menu.AddItem("Save & Quit", "save")
	menu.AddItem("Quit without Saving", "")
	for {
		util.ClearScreen()
		switch menu.Display() {
		case "lib":
			a.libraryMenu()
		case "thumb":
			a.thumbnailMenu()
		case "config":
			a.settingsMenu()
		case "save":
			if err := a.writeFiles(); err != nil {
				log.Fatal(err)
			}
			fallthrough
		default:
			return
		}
	}
}

func (a *Application) libraryMenu() error {
	menu := gocliselect.NewMenu("Edit Library", false)
	if a.ShowAdd {
		menu.AddItem("Add Entry", "add")
	}
	menu.AddItem("Edit Entry", "edit")
	menu.AddItem("Remove Entry", "remove")
	menu.AddItem("Back", "")

	for {
		util.ClearScreen()
		switch menu.Display() {
		case "add":
			a.add()
		case "edit":
			a.edit()
		case "remove":
			a.removeGame()
		default:
			return nil
		}
	}
}

func (a *Application) thumbnailMenu() error {
	menu := gocliselect.NewMenu("Edit Thumbnails", false)
	menu.AddItem("Regenerate Game Thumbnail", "single")
	menu.AddItem("Regenerate User Library", "library")
	//menu.AddItem("Remove Thumbnail", "rm") // TODO: Maybe? Maybe not? Has some
	menu.AddItem("Prune Thumbnails", "prune")
	menu.AddItem("Generate Complete System Thumbnails", "all")
	menu.AddItem("Back", "")

	for {
		util.ClearScreen()
		switch menu.Display() {
		case "single":
			a.regenSingle()
		case "library":
			a.regenerate()
		case "rm":
			//a.removeThumb()
		case "prune":
			a.prune()
		case "all":
			return a.generateAll()
		default:
			return nil
		}
	}
}

func (a *Application) settingsMenu() {
	s := gocliselect.NewMenu("Library Editor Options", false)
	s.AddItem(fmt.Sprintf("[%s] Remove thumbnail when removing game", x(a.RemoveImages)), "rm")
	s.AddItem(fmt.Sprintf("[%s] Show advanced library editing fields (Experimental)", x(a.AdvancedEditing)), "adv")
	s.AddItem(fmt.Sprintf("[%s] Show add library entry (Experimental)", x(a.ShowAdd)), "add")
	s.AddItem("Back", "")

	for {
		util.ClearScreen()
		switch s.Display() {
		case "rm":
			a.RemoveImages = !a.RemoveImages
		case "adv":
			a.AdvancedEditing = !a.AdvancedEditing
		case "add":
			a.ShowAdd = !a.ShowAdd
		default:
			return
		}

		// A hack to allow us to update the menu entries without creating an entirely new menu each time.
		s.MenuItems[0].Text = fmt.Sprintf("[%s] Remove thumbnail when removing game", x(a.RemoveImages))
		s.MenuItems[1].Text = fmt.Sprintf("[%s] Show advanced library editing fields (Experimental)", x(a.AdvancedEditing))
		s.MenuItems[2].Text = fmt.Sprintf("[%s] Show add library entry (Experimental)", x(a.ShowAdd))
	}
}

// x is a simple function that returns "X" is setting is true
// Wouldn't ternary operators be great?
func x(setting bool) string {
	if setting {
		return "X"
	}
	return " "
}

func (a *Application) pagedEntries(title string, f func(i int) error) error {
	clone := slices.Clone(a.Entries) // For cancel
	var start, pos int
	var x string
	for {
		if start >= len(a.Entries) {
			start = max(len(a.Entries)-10, 0) // For delete: flips to the previous page if we clear the last page
		}
		end := min(start+10, len(a.Entries))
		switch x, pos = a.displayEntries(title, pos, start, end); x {
		case "<":
			if newStart := max(0, start-10); newStart == start {
				fmt.Printf("%c", 7) // We're at the first page. Ring the bell
			} else {
				start = newStart
			}
		case ">":
			if newStart := min(start+10, len(a.Entries)-len(a.Entries)%10); start == newStart {
				fmt.Printf("%c", 7) // We're at the last page. Ring the bell
			} else {
				start = newStart
			}
		case "done":
			return nil
		case "":
			a.Entries = clone // Restore the entries to the original copy
			return nil
		default:
			i, err := strconv.Atoi(x)
			if err != nil {
				return err
			}
			if err := f(i); err != nil {
				a.Entries = clone // Restore the original
				return err
			}
		}
	}
}

// displayEntries is a simple function that uses gocliselect to fake multipage menus
func (a *Application) displayEntries(title string, pos, start, end int) (string, int) {
	util.ClearScreen()

	menu := gocliselect.NewMenu(fmt.Sprintf("%s Entry [%d-%d]", title, start+1, end), true)

	for i := start; i < end; i++ {
		menu.AddItem(fmt.Sprintf("%d. %s", i+1, a.Entries[i].Name), strconv.Itoa(i))
	}

	// FIXME: Causes more trouble than it's worth wrt cursor position
	//if start != 0 {
	//	menu.AddItem(fmt.Sprintf("<- %d-%d", max(start-9, 0), start), "<")
	//}
	//if end < len(a.Entries) {
	//	menu.AddItem(fmt.Sprintf("%d-%d ->", end+1, min(end+10, len(a.Entries))), ">")
	//}

	menu.AddItem("Cancel", "")
	menu.AddItem("Done", "done")

	menu.CursorPos = pos

	return menu.Display(), menu.CursorPos
}

func (a *Application) add() error {
	util.ClearScreen()
	entry := model.Entry{}

	// Start with the system menu since this will otherwise clear the screen
	sys := gocliselect.NewMenu("Add New Entry", false)
	sys.AddItem("Game Boy", "GB")
	sys.AddItem("Game Boy Color", "GBC")
	sys.AddItem("Game Boy Advance", "GBA")
	sys.AddItem("Game Gear", "GG")
	sys.AddItem("Sega Master System", "SMS")
	sys.AddItem("Neo Geo Pocket", "NGP")
	sys.AddItem("Neo Geo Pocket Color", "NGPC")
	sys.AddItem("TurboGrafx 16", "PCE")
	sys.AddItem("Atari Lynx", "Lynx")
	system := sys.Display()
	if system == "" { // ESC or Ctrl-C pressed
		return nil
	}
	if s, err := util.Parse(system); err != nil {
		return err
	} else {
		entry.System = s
	}

	util.ClearScreen()
	fmt.Printf("%s\n", goterm.Color(goterm.Bold(
		fmt.Sprintf("New Entry (%s)", entry.System.String()),
	)+":", goterm.CYAN))

	in := bufio.NewScanner(os.Stdin)
	fmt.Print("\n\rName: ")
	in.Scan()
	entry.Name = in.Text()

	fmt.Print("\rCRC32: ")
	in.Scan()
	if h, err := util.HexStringTransform(in.Text()); err != nil {
		return err
	} else {
		entry.Crc32 = h
	}

	fmt.Print("\rSignature: ")
	in.Scan()
	if h, err := util.HexStringTransform(in.Text()); err != nil {
		return err
	} else {
		entry.Sig = h
	}

	fmt.Print("\rMagic Number: ")
	in.Scan()
	if h, err := util.HexStringTransform(in.Text()); err != nil {
		return err
	} else {
		entry.Magic = h
	}

	a.Entries = append(a.Entries, entry)
	slices.SortFunc(a.Entries, model.EntrySort)

	if img, err := model.GenerateThumbnail(a.RootDir, entry.System.ThumbFile(), entry.Crc32); err != nil {
		// Don't care that much
	} else {
		sys := entry.System.ThumbFile()
		t := a.Thumbs[sys]
		t.Images = append(t.Images, img)
		t.Modified = true
		a.Thumbs[sys] = t
	}

	return nil
}

func (a *Application) edit() error {
	return a.pagedEntries("Edit", func(i int) error {
		e, err := editEntry(a.Entries[i], a.AdvancedEditing)
		if err != nil {
			return err
		}
		a.Entries[i] = e
		slices.SortFunc(a.Entries, model.EntrySort)

		sys := e.System.ThumbFile()
		for _, img := range a.Thumbs[sys].Images {
			if img.Crc32 == a.Entries[i].Crc32 {
				// Image already exists in the thumbs.bin. Don't do anything.
				return nil
			}
		}

		thumbs := a.Thumbs[sys]
		t, err := model.GenerateThumbnail(a.RootDir, sys, e.Crc32)
		if err != nil { // We don't consider this a blocker
			fmt.Println(goterm.Color("WARN: Could not parse thumbnail file", goterm.YELLOW))
			time.Sleep(time.Second)
			return nil
		}
		thumbs.Images = append(thumbs.Images, t)
		a.Thumbs[sys] = thumbs

		return nil
	})
}

func editEntry(e model.Entry, advanced bool) (model.Entry, error) {
	clone := e // In case the user cancels
	util.ClearScreen()

	fmt.Printf("%s\n", goterm.Color(goterm.Bold("Edit Entry")+":", goterm.CYAN))
	fmt.Printf("%s\n", goterm.Color("(Return to accept defaults)", goterm.CYAN))

	in := bufio.NewScanner(os.Stdin)
	fmt.Printf("\rName (%s): ", e.Name)
	in.Scan()
	if s := in.Text(); s != "" {
		e.Name = s
	}

	if advanced {
		// TODO: Don't really like this section thanks to gocliselect's bolding. Look into customizing it
		sys := gocliselect.NewMenu("System:", false)
		sys.AddItem("Game Boy", "GB")
		sys.AddItem("Game Boy Color", "GBC")
		sys.AddItem("Game Boy Advance", "GBA")
		sys.AddItem("Game Gear", "GG")
		sys.AddItem("Sega Master System", "SMS")
		sys.AddItem("Neo Geo Pocket", "NGP")
		sys.AddItem("Neo Geo Pocket Color", "NGPC")
		sys.AddItem("TurboGrafx 16", "PCE")
		sys.AddItem("Atari Lynx", "Lynx")
		sys.CursorPos = int(e.System)
		system := sys.Display()
		if system == "" { // ESC or Ctrl-C pressed
			return clone, nil
		}
		if s, err := util.Parse(system); err != nil {
			return clone, err
		} else {
			e.System = s
		}
	}

	fmt.Printf("\rCRC32 (%08x): ", e.Crc32)
	in.Scan()
	if s := in.Text(); s != "" {
		h, err := util.HexStringTransform(s)
		if err != nil {
			return clone, err
		}
		e.Crc32 = h
	}

	if advanced {
		// Just a bit unsafe. Leave it behind the advanced toggle
		fmt.Printf("\rSignature (%08x): ", e.Sig)
		in.Scan()
		if s := in.Text(); s != "" {
			h, err := util.HexStringTransform(s)
			if err != nil {
				return clone, err
			}
			e.Sig = h
		}
		fmt.Printf("\rMagic Number (%08x): ", e.Magic)
		in.Scan()
		if s := in.Text(); s != "" {
			h, err := util.HexStringTransform(s)
			if err != nil {
				return clone, err
			}
			e.Magic = h
		}
	}

	return e, nil
}

func (a *Application) removeGame() error {
	return a.pagedEntries("Delete", func(i int) error {
		rm := a.Entries[i]
		a.Entries = slices.Delete(a.Entries, i, i+1)

		if !a.RemoveImages { // If they don't have this flagged, leave the thumbnails alone
			return nil
		}

		// Delete the thumbnail entry if it exists
		sys := rm.System.ThumbFile()
		t := a.Thumbs[sys]
		for j, img := range t.Images {
			if rm.Crc32 == img.Crc32 {
				t.Images = slices.Delete(t.Images, j, j+1)
				t.Modified = true
			}
		}
		a.Thumbs[sys] = t

		return nil
	})
}

func (a *Application) regenSingle() error {
	return a.pagedEntries("Regenerate Thumbnail", func(i int) error {
		e := a.Entries[i]
		sys := e.System.ThumbFile()
		img, err := model.GenerateThumbnail(a.RootDir, sys, e.Crc32)
		if err != nil {
			return err
		}

		// Thumbnails aren't stored in the same order as entries
		found := false
		for j, old := range a.Thumbs[sys].Images {
			if old.Crc32 == img.Crc32 {
				a.Thumbs[sys].Images[j] = img
				found = true
				break
			}
		}
		if !found { // Shouldn't happen. But append if it does
			t := a.Thumbs[sys]
			t.Images = append(t.Images, img)
		}
		return nil
	})
}

func (a *Application) regenerate() error {
	clone := maps.Clone(a.Thumbs)

	util.ClearScreen()
	fmt.Println(goterm.Bold("Regenerating thumbnails. This may take a while..."))

	clear(a.Thumbs)
	for _, e := range a.Entries {
		sys := e.System.ThumbFile()

		i, err := model.GenerateThumbnail(a.RootDir, sys, e.Crc32)
		if err != nil {
			a.Thumbs = clone
			return err
		}

		t := a.Thumbs[sys]
		t.Modified = true
		t.Images = append(t.Images, i)
		a.Thumbs[sys] = t
	}

	return nil
}

func (a *Application) generateAll() error {
	util.ClearScreen()

	fmt.Println(goterm.Bold("WARNING"))
	fmt.Println("This option will generate full _thumbs.bin files for all images known to the Pocket.")
	fmt.Print("Doing this may affect library performance. Are you sure you wish to proceed? (y/N) ")

	t, _ := term.Open("/dev/tty")

	err := term.RawMode(t)
	if err != nil {
		log.Fatal(err)
	}

	readBytes := make([]byte, 3)
	n, err := t.Read(readBytes)
	if err != nil {
		return err
	}
	fmt.Print(string(readBytes)) // Show what they typed
	_ = t.Restore()
	_ = t.Close()

	if n != 1 || strings.ToLower(string(readBytes[0])) != "y" {
		return nil // Anything other than y cancels
	}

	fmt.Println("\n\nThis is going to take a while. Maybe grab a coffee or something?")

	fmt.Printf("\033[?25l")       // Turn the cursor off
	defer fmt.Printf("\033[?25h") // Show it again

	clone := maps.Clone(a.Thumbs) // If something goes wrong, restore this
	for _, sys := range util.ValidThumbsFiles {
		fmt.Printf("Parsing %s", sys.String())
		de, err := os.ReadDir(fmt.Sprintf("%s/System/Library/Images/%s", a.RootDir, sys.String()))
		if errors.Is(err, os.ErrNotExist) {
			// Not found. Just continue
			continue
		} else if err != nil {
			return err
		}

		thumbs := model.Thumbnails{Modified: true}
		dot := 0
		for i, e := range de {
			for j := dot; j < int(float32(i)/float32(len(de))*100); j++ {
				fmt.Print(".")
				dot++
			}
			if e.IsDir() || len(e.Name()) != 12 /* 8 characters + 4 char extension */ {
				continue
			}

			hash, _, found := strings.Cut(e.Name(), ".")
			if !found { // Not a valid file name
				continue
			}
			b, err := hex.DecodeString(hash)
			if err != nil {
				// Not a valid file name. Skip
				continue
			}
			i, err := model.GenerateThumbnail(a.RootDir, sys, binary.BigEndian.Uint32(b))
			if err != nil {
				a.Thumbs = clone
				log.Fatal(err)
			}

			thumbs.Images = append(thumbs.Images, i)
		}
		a.Thumbs[sys] = thumbs
		fmt.Println("done.")
	}
	return nil
}

// prune removes entries from the thumbnails files that are no longer associated with any library entry
// If you have a very large library or very large thumbnail files, this may take a while.
func (a *Application) prune() error {
	for k, v := range a.Thumbs {
		t := a.Thumbs[k]
		t.Images = slices.DeleteFunc(v.Images, func(image model.Image) bool {
			return !slices.ContainsFunc(a.Entries, func(entry model.Entry) bool {
				return entry.System.ThumbFile() == k && entry.Crc32 == image.Crc32
			})
		})
		a.Thumbs[k] = t
	}
	return nil
}

func (a *Application) writeFiles() error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	l, err := os.Create(fmt.Sprintf("%s/list.bin", wd))
	if err != nil {
		return err
	}
	defer l.Close()

	p, err := os.Create(fmt.Sprintf("%s/playtimes.bin", wd))
	if err != nil {
		return err
	}
	defer p.Close()

	// Prep list.bin
	if err := binary.Write(l, binary.BigEndian, model.LibraryHeader); err != nil {
		return err
	}
	if err := binary.Write(l, binary.LittleEndian, uint32(len(a.Entries))); err != nil {
		return err
	}
	if err := binary.Write(l, binary.LittleEndian, uint32(0x10)); err != nil { // Not sure what this value signifies, but accidentally setting it to 1 caused the system to loop
		return err
	}
	if err := binary.Write(l, binary.LittleEndian, firstLibraryAddr); err != nil { // This seems to be duplicated? I dunno
		return err
	}

	// Prep playtimes.bin
	if err := binary.Write(p, binary.BigEndian, model.PlaytimeHeader); err != nil {
		return err
	}
	if err := binary.Write(p, binary.LittleEndian, uint32(len(a.Entries))); err != nil {
		return err
	}

	// Build the address entries
	slices.SortFunc(a.Entries, model.EntrySort)
	addresses := make([]uint32, firstLibraryAddr/4-4)
	addresses[0] = firstLibraryAddr
	last := firstLibraryAddr
	for i := 1; i < len(a.Entries); i++ {
		addresses[i] = last + uint32(a.Entries[i-1].CalculateLength())
		last = addresses[i]
	}

	if err := binary.Write(l, binary.LittleEndian, addresses); err != nil {
		return err
	}

	for _, e := range a.Entries {
		if _, err := e.WriteTo(l); err != nil {
			return err
		}

		// list.bin & playtimes.bin must be recorded in the same order.
		// So write the playtimes.bin info now as well.
		if err := binary.Write(p, binary.LittleEndian, e.Sig); err != nil {
			return err
		}
		if _, err := a.PlayTimes[e.Sig].WriteTo(p); err != nil {
			return err
		}
	}

	for k, v := range a.Thumbs {
		//if v.Modified {
		t, err := os.Create(fmt.Sprintf("%s/%s_thumbs.bin", wd, strings.ToLower(k.String())))
		if err != nil {
			return err
		}
		defer t.Close() // For the early exits

		if err := binary.Write(t, binary.LittleEndian, model.ThumbnailHeader); err != nil {
			return err
		}
		if err := binary.Write(t, binary.LittleEndian, model.UnknownWord); err != nil {
			return err
		}
		if err := binary.Write(t, binary.LittleEndian, uint32(len(v.Images))); err != nil {
			return err
		}
		addr := firstThumbsAddr
		for i, j := range v.Images {
			if err := binary.Write(t, binary.LittleEndian, j.Crc32); err != nil {
				return err
			}
			if err := binary.Write(t, binary.LittleEndian, addr); err != nil {
				return err
			}
			addr = addr + uint32(len(v.Images[i].Image))
		}
		// write the unused addresses out as 0s
		if _, err := t.Write(make([]byte, int(firstThumbsAddr)-0xC-8*len(v.Images))); err != nil {
			return err
		}
		// write out the images
		for _, j := range v.Images {
			if _, err := t.Write(j.Image); err != nil {
				return err
			}
		}
		t.Close()
		//}
	}

	return nil
}
