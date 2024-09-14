package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/g026r/pocket-library-editor/pkg"
	"github.com/g026r/pocket-library-editor/pkg/model"
	"github.com/g026r/pocket-library-editor/pkg/util"
)

//go:embed resources/*.json
var jsons embed.FS

func main() {
	var err error
	var arg string
	switch len(os.Args) {
	case 1:
		if arg, err = os.Executable(); err != nil { // TODO: Would it be better to use cwd instead?
			log.Fatal(err)
		}
	case 2:
		arg = os.Args[1]
	default:
		printUsage()
		os.Exit(2)
	}

	app, err := loadPocketDir(arg)
	if err != nil {
		log.Fatal(err)
	}

	if app.ShowAdd { // Only need to load these for the add UI
		library, err := loadInternal()
		if err != nil {
			log.Fatal(err)
		}
		app.Internal = library
	}

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func printUsage() {
	fmt.Println("Usage: place in the root of your Pocket's SD card & run. Or run & pass it the path to the SD card root as an argument.")
	fmt.Println("Outputs files in the current working directory.")
}

func loadPocketDir(d string) (pkg.Application, error) {
	d, err := filepath.Abs(d)
	if err != nil {
		return pkg.Application{}, err
	}
	fi, err := os.Stat(d)
	if err != nil {
		return pkg.Application{}, err
	} else if !fi.IsDir() {
		return pkg.Application{}, fmt.Errorf("%s is not a directory", d)
	}

	root := os.DirFS(d)

	pg, err := fs.Sub(root, "System/Played Games")
	if err != nil {
		return pkg.Application{}, nil
	}
	entries, err := model.ReadEntries(pg)
	if err != nil {
		return pkg.Application{}, err
	}

	playtimes, err := model.ReadPlayTimes(pg)
	if err != nil {
		return pkg.Application{}, err
	}
	if len(playtimes) != len(entries) {
		return pkg.Application{}, fmt.Errorf("entry count mismatch between list.bin [%d] & playtimes.bin [%d]", len(entries), len(playtimes))
	}

	tb, err := fs.Sub(root, "System/Library/Images")
	if err != nil {
		return pkg.Application{}, nil
	}
	thumbs, err := model.LoadThumbnails(tb)
	if err != nil {
		return pkg.Application{}, err
	}

	return pkg.Application{
		RootDir:   root,
		Entries:   entries,
		PlayTimes: playtimes,
		Thumbs:    thumbs,
		Config: pkg.Config{
			RemoveImages:    true,
			AdvancedEditing: false,
			ShowAdd:         true,
		},
	}, nil
}

func loadInternal() (map[util.System][]model.Entry, error) {
	dir, err := jsons.ReadDir("resources")
	if err != nil {
		return nil, err
	}

	library := make(map[util.System][]model.Entry)
	for _, d := range dir {
		f, err := jsons.ReadFile(fmt.Sprintf("resources/%s", d.Name()))
		if err != nil {
			return nil, err
		}
		var x []jsonEntry
		if err := json.Unmarshal(f, &x); err != nil {
			return nil, err
		}
		sys, err := util.Parse(strings.TrimSuffix(d.Name(), ".json"))
		if err != nil {
			return nil, err
		}

		// Oh, for a native map function
		e := make([]model.Entry, len(x))
		for i := range x {
			e[i] = x[i].Entry()
		}

		slices.SortFunc(e, model.EntrySort)
		library[sys] = e
	}

	return library, nil
}

type jsonEntry struct {
	util.System `json:"system"`
	Crc32       string `json:"crc"`
	Sig         string `json:"sig"`
	Magic       string `json:"magic"` // TODO: Work out all possible mappings for this
	Name        string `json:"name"`
}

func (j jsonEntry) Entry() model.Entry {
	e := model.Entry{
		Name:   j.Name,
		System: j.System,
	}
	e.Sig, _ = util.HexStringTransform(j.Sig)
	e.Magic, _ = util.HexStringTransform(j.Magic)
	e.Crc32, _ = util.HexStringTransform(j.Crc32)
	return e
}
