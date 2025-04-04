package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/g026r/pocket-library-toolkit/pkg/io"
	"github.com/g026r/pocket-library-toolkit/pkg/root"
)

// main provides a simple application to fix played times & nothing else.
func main() {
	ex, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	root, err := root.OpenRoot(filepath.Dir(ex))
	if err != nil {
		log.Fatal(err)
	}
	defer root.Close()

	p, err := io.LoadPlaytimes(root.FS())
	if err != nil {
		log.Fatal(err)
	}

	complete := false
	out, err := root.CreateTemp("System/Played Games", "playtimes_*.tmp")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = out.Close()
		if complete { // Overwrite the original with the temp file if successful; delete it if not.
			err = root.Rename(fmt.Sprintf("System/Played Games/%s", filepath.Base(out.Name())), "System/Played Games/playtimes.bin")
		} else {
			err = os.Remove(out.Name())
		}
	}()

	// Write header
	if err := binary.Write(out, binary.BigEndian, io.PlaytimesHeader); err != nil {
		log.Fatal(err)
	}
	if err := binary.Write(out, binary.LittleEndian, uint32(len(p))); err != nil {
		log.Fatal(err)
	}

	// Write entries in the same order as list.bin
	for _, tmp := range p {
		tmp.Played = tmp.Played &^ 0xFF000000 // Fix the time. System prefix will get handled by WriteTo
		if _, err := tmp.WriteTo(out); err != nil {
			log.Fatal(err)
		}
	}

	complete = true
}
