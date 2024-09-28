package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/g026r/pocket-toolkit/pkg/io"
)

// Simple application to fix played times & nothing else.

func main() {
	ex, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	root := filepath.Dir(ex)

	entries, err := io.LoadEntries(os.DirFS(root))
	if err != nil {
		log.Fatal(err)
	}
	p, err := io.LoadPlaytimes(os.DirFS(root))
	if err != nil {
		log.Fatal(err)
	}

	complete := false
	out, err := os.CreateTemp(root, "playtimes_*.bin")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = out.Close()
		if complete { // Overwrite the original with the temp file if successful; delete it if not.
			err = os.Rename(out.Name(), fmt.Sprintf("%s/System/Played Games/playtimes.bin", root))
		} else {
			err = os.Remove(out.Name())
		}
	}()

	// Write header
	if err := binary.Write(out, binary.BigEndian, io.PlaytimesHeader); err != nil {
		log.Fatal(err)
	}
	if err := binary.Write(out, binary.LittleEndian, uint32(len(entries))); err != nil {
		log.Fatal(err)
	}

	// Write entries in the same order as list.bin
	for _, e := range entries {
		tmp := p[e.Sig]
		tmp.Played = tmp.Played &^ 0xFF000000 // Fix the time. System prefix will get written by WriteTo
		if err := binary.Write(out, binary.LittleEndian, e.Sig); err != nil {
			log.Fatal(err)
		}
		if _, err := tmp.WriteTo(out); err != nil {
			log.Fatal(err)
		}
	}

	complete = true
}
