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
		//if arg, err = os.Getwd(); err != nil {
		//	// TODO error handling
		//}
		arg = "/Users/g026r/Downloads/working"
	case 2:
		arg = os.Args[1]
	default:
		printUsage()
	}

	app, err := verifyDir(arg)
	if err != nil {
		log.Fatal(err)
	}

	app.Run()
}

func printUsage() {

	os.Exit(2)
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
