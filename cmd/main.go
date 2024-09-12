package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/g026r/pocket-library-editor/pkg"
	"github.com/g026r/pocket-library-editor/pkg/model"
)

func main() {
	var arg string
	var err error
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

	app, err := verifyDir(arg)
	if err != nil {
		log.Fatal(err)
	}

	app.Run()
}

func printUsage() {
	fmt.Println("Usage: place in the root of your Pocket's SD card & run. Or run & pass it the path to the SD card root as an argument.")
	fmt.Println("Outputs files in the current working directory.")
}

func verifyDir(d string) (pkg.Application, error) {
	d, err := filepath.Abs(d)
	if err != nil {
		return pkg.Application{}, err
	}

	entries, err := model.ReadEntries(fmt.Sprintf("%s/System/Played Games", d))
	if err != nil {
		return pkg.Application{}, err
	}

	playtimes, err := model.ReadPlayTimes(fmt.Sprintf("%s/System/Played Games", d))
	if err != nil {
		return pkg.Application{}, err
	}
	if len(playtimes) != len(entries) {
		return pkg.Application{}, errors.New("entry count mismatch between list.bin & playtimes.bin")
	}

	thumbs, err := model.LoadThumbnails(fmt.Sprintf("%s/System/Library/Images", d))
	if err != nil {
		return pkg.Application{}, err
	}

	return pkg.Application{
		RootDir:   d,
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
