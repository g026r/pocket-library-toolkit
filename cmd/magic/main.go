package main

import (
	"cmp"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/g026r/pocket-library-editor/pkg/io"
	"github.com/g026r/pocket-library-editor/pkg/model"
	"github.com/g026r/pocket-library-editor/pkg/util"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal("Provide a path to the Pocket SD root")
	}
	dir, err := filepath.Abs(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	entries, err := io.LoadEntries(os.DirFS(dir))
	if err != nil {
		log.Fatal(err)
	}
	internal, err := io.LoadInternal()
	if err != nil {
		log.Fatal(err)
	}

	for _, e := range entries {
		if !slices.ContainsFunc(internal[e.System], func(entry model.Entry) bool {
			return entry.Crc32 == e.Crc32 && entry.Sig == e.Sig && entry.Magic == e.Magic
		}) {
			internal[e.System] = append(internal[e.System], e)
		}
	}

	if err := writeNewFiles(internal); err != nil {
		log.Fatal(err)
	}
}

func writeNewFiles(internal map[util.System][]model.Entry) error {
	for k, v := range internal {
		slices.SortFunc(v, func(a, b model.Entry) int {
			return cmp.Compare(a.Magic, b.Magic) // Sort now before we turn the magic number into a string
		})

		d, err := os.Getwd()
		if err != nil {
			return err
		}
		j, err := os.Create(fmt.Sprintf("%s/%s.json", d, strings.ToLower(k.String())))
		if err := io.SaveInternal(j, v); err != nil {
			_ = j.Close()
			return err
		}
		_ = j.Close()

		m, err := os.Create(fmt.Sprintf("%s/%s.md", d, strings.ToLower(k.String())))
		if err != nil {
			return err
		}
		for _, e := range v {
			if _, err := m.WriteString(fmt.Sprintf("## %s\n\n", e.Name)); err != nil {
				m.Close()
				return err
			}
			if _, err := m.WriteString(fmt.Sprintf("- CRC32: `0x%08x`\n", e.Crc32)); err != nil {
				_ = m.Close()
				return err
			}
			if _, err := m.WriteString(fmt.Sprintf("- Signature: `0x%08x`\n", e.Sig)); err != nil {
				_ = m.Close()
				return err
			}
			if _, err := m.WriteString(fmt.Sprintf("- Magic Number: `0x%04x`\n\n", e.Magic)); err != nil {
				m.Close()
				return err
			}
		}
		_ = m.Close()
	}
	return nil
}
