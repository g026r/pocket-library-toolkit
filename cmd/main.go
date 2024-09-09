package main

import (
	"fmt"
	"log"
	"os"

	"github.com/g026r/pocket-library-editor/pkg"
	"github.com/g026r/pocket-library-editor/pkg/model"
)

func main() {
	list_bin := "/Users/g026r/Downloads/working/Played Games/list.bin"
	playtimes_bin := "/Users/g026r/Downloads/working/Played Games/playtimes.bin"

	entries, err := model.ReadEntries(list_bin)
	if err != nil {
		log.Fatal(err)
	}
	playtimes, err := model.ReadPlayTimes(playtimes_bin)
	if err != nil {
		log.Fatal(err)
	}

	if len(playtimes) != len(entries) {
		log.Fatal("ERROR: Entry count mismatch between playtimes.bin & list.bin!")
	}

	app := pkg.Application{Entries: entries, PlayTimes: playtimes}
	for {
		switch app.Main() {
		case "add":
			app.Add()
		case "edit":
			app.Edit()
		case "remove":
			app.Remove()
		case "save":
			if err := model.WriteFiles(list_bin, app.Entries, playtimes); err != nil {
				log.Fatal(err)
			}
			fallthrough
		default:
			fmt.Println()
			os.Exit(0)
		}
	}

	//out, err := os.Create("/Users/g026r/Downloads/temp.bin")
	//defer out.Close()
	//for i := range library {
	//	_, err := f.Seek(int64(library[i].offset), 0)
	//	if err != nil {
	//		log.Fatalf("seek %d: %v", i, err)
	//	}
	//
	//	library[i].entry, err = model.ReadEntry(out)
	//	if err != nil {
	//		log.Fatalf("read %d: %v", i, err)
	//	}
	//	fmt.Println(library[i].entry.Name)
	//
	//	library[i].entry.WriteTo(out)
	//}
}
