package pkg

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"slices"
	"strconv"
	"time"

	"github.com/buger/goterm"

	"github.com/nexidian/gocliselect"

	"github.com/g026r/pocket-library-editor/pkg/io"
	"github.com/g026r/pocket-library-editor/pkg/models"
	"github.com/g026r/pocket-library-editor/pkg/util"
)

var ptReg = regexp.MustCompile("^(\\d+[Hh])?\\s*(\\d+[Mm])?\\s*(\\d+[Ss])?$")

type Application struct {
	RootDir   fs.FS
	Entries   []models.Entry
	PlayTimes map[uint32]models.PlayTime
	Thumbs    map[models.System]models.Thumbnails
	io.Config
	Internal map[models.System][]models.Entry // Internal is a map of all known possible entries, grouped by system
}

func (a *Application) add() error {
	entry := models.Entry{}

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
	system := sys.Display()
	if system == "" { // ESC or Ctrl-C pressed
		return nil
	}
	if s, err := models.Parse(system); err != nil {
		return err
	} else {
		entry.System = s
	}

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
	slices.SortFunc(a.Entries, models.EntrySort)

	if img, err := models.GenerateThumbnail(a.RootDir, entry.System.ThumbFile(), entry.Crc32); err != nil {
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
		p := a.PlayTimes[a.Entries[i].Sig] // Need to get the playtime now in case the signature changes during editing

		e, err := editEntry(a.Entries[i], a.AdvancedEditing)
		if err != nil {
			return err
		}

		p, err = editPlaytime(p)
		if err != nil {
			return err
		}

		a.Entries[i] = e
		a.PlayTimes[e.Sig] = p
		slices.SortFunc(a.Entries, models.EntrySort)

		sys := e.System.ThumbFile()
		for _, img := range a.Thumbs[sys].Images {
			if img.Crc32 == a.Entries[i].Crc32 {
				// Image already exists in the thumbs.bin. Don't do anything.
				return nil
			}
		}

		thumbs := a.Thumbs[sys]
		t, err := models.GenerateThumbnail(a.RootDir, sys, e.Crc32)
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

// editEntry pops up the edit entry dialog for a given models.Entry & returns the editted result.
// It's in here rather than in models.Entry to keep the UI code out of the ui package
func editEntry(e models.Entry, advanced bool) (models.Entry, error) {
	clone := e // In case the user cancels

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
		sys := gocliselect.NewMenu("System:")
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
		if s, err := models.Parse(system); err != nil {
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

func editPlaytime(pt models.PlayTime) (models.PlayTime, error) {
	added := time.Unix(int64(pt.Added), 0)

	in := bufio.NewScanner(os.Stdin)
	fmt.Printf("\rAdded Date (%v): ", added.UTC().Format("2006/01/02 15:04:05"))
	in.Scan()
	if s := in.Text(); s != "" {
		t, err := time.Parse("2006/01/02 15:04:05", s)
		if err != nil {
			return pt, err
		}
		pt.Added = uint32(t.Unix())
	}

	hour := pt.Played / (60 * 60)
	minute := (pt.Played - hour*60*60) / 60
	sec := pt.Played - hour*60*60 - minute*60
	fmt.Printf("\rPlay Time (%dh %dm %ds): ", hour, minute, sec)
	in.Scan()
	if s := in.Text(); s != "" {
		parts := ptReg.FindStringSubmatch(s)
		if len(parts) == 0 {
			return pt, fmt.Errorf("invalid playtime %s", s)
		}
		var newPlay uint32
		for _, play := range parts[1:] {
			t, _ := strconv.Atoi(play[:len(play)-1]) // Can ignore the error here as the regex took care of that
			switch play[len(play)-1:] {
			case "h", "H":
				newPlay = newPlay + uint32(t)*60*60
			case "m", "M":
				newPlay = newPlay + uint32(t)*60
			case "s", "S":
				newPlay = newPlay + uint32(t)
			}
		}
		pt.Played = newPlay
	}

	return pt, nil
}

// TODO: How to deal with this
//func (a *Application) removeThumb() error {
//	clone := maps.Clone(a.Thumbs)
//
//	sys := gocliselect.NewMenu("System:", false)
//	sys.AddItem("Game Boy / Game Boy Color", "GB")
//	sys.AddItem("Game Boy Advance", "GBA")
//	sys.AddItem("Game Gear / Sega Master System", "GG")
//	sys.AddItem("Neo Geo Pocket / Neo Geo Pocket Color", "NGP")
//	sys.AddItem("TurboGrafx 16", "PCE")
//	sys.AddItem("Atari Lynx", "Lynx")
//	system := sys.Display()
//	if system == "" { // ESC or Ctrl-C pressed
//		return nil
//	}
//}
