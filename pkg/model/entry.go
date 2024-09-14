package model

import (
	"cmp"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"slices"

	"github.com/g026r/pocket-library-editor/pkg/util"
)

const LibraryHeader uint32 = 0x01464154

var EntrySort = func(a, b Entry) int {
	return cmp.Compare(a.Name, b.Name)
}

type Entry struct {
	util.System
	Crc32 uint32
	Sig   uint32
	Magic uint32
	Name  string
}

// CalculateLength returns the length in bytes of the library entry
// includes any extra padding needed to get it to a word boundary & can be used to calculate offsets
func (e Entry) CalculateLength() uint16 {
	length := 4 /*length + system*/ + 4 /*crc*/ + 4 /*hash*/ + 4 /*unknown*/ + uint16(len(e.Name)) + 1 /*0-terminator*/
	if extra := length % 4; extra != 0 {
		length = length + 4 - extra // Need to pad it out to a word boundary
	}

	return length
}

func (e Entry) WriteTo(w io.Writer) (n int64, err error) {
	length := e.CalculateLength()

	if err = binary.Write(w, binary.LittleEndian, length); err != nil {
		return
	}
	n = 2

	if err = binary.Write(w, binary.BigEndian, e.System); err != nil {
		return
	}
	n = n + 2

	if err = binary.Write(w, binary.LittleEndian, e.Crc32); err != nil {
		return
	}
	n = n + 4

	if err = binary.Write(w, binary.LittleEndian, e.Sig); err != nil {
		return
	}
	n = n + 4

	if err = binary.Write(w, binary.LittleEndian, e.Magic); err != nil {
		return
	}
	n = n + 4

	// Write the string plus the terminator
	zeroTerm := append([]byte(e.Name), 0x00)
	if extra := len(zeroTerm) % 4; extra != 0 {
		// Need to pad it out if it's not on a word boundary
		zeroTerm = slices.Concat(zeroTerm, make([]byte, 4-extra))
	}
	if err = binary.Write(w, binary.BigEndian, zeroTerm); err != nil {
		return
	}
	n = n + int64(len(zeroTerm))

	return
}

func ReadEntries(fs fs.FS) ([]Entry, error) {
	f, err := util.ReadSeeker(fs, "list.bin")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var header, num, unknown uint32
	if err = binary.Read(f, binary.BigEndian, &header); err != nil {
		return nil, err
	}
	if header != LibraryHeader { // Missing the magic number = not a Pocket library file
		return nil, fmt.Errorf("list.bin: %w", util.ErrUnrecognizedFileFormat)
	}

	if err = binary.Read(f, binary.LittleEndian, &num); err != nil {
		return nil, err
	}

	// TODO: I don't know what this word represents. It's equivalent to 0x00000010 on mine.
	if err = binary.Read(f, binary.LittleEndian, &unknown); err != nil {
		return nil, err
	}

	// TODO: This appears to be the first entry's value? But why?
	var dupe uint32
	if err = binary.Read(f, binary.LittleEndian, &dupe); err != nil {
		return nil, err
	}

	// Parse the library entry locations.
	addresses := make([]uint32, int(num))
	if err = binary.Read(f, binary.LittleEndian, &addresses); err != nil {
		return nil, err
	}

	// Parse each of the library entries. The addresses are supposed to be sequential, but we're not going to trust that.
	entries := make([]Entry, num)
	for i := range addresses {
		if _, err := f.Seek(int64(addresses[i]), io.SeekStart); err != nil {
			return entries, err
		}

		if e, err := readEntry(f); err != nil {
			return entries, err
		} else {
			entries[i] = e
		}
	}

	// Should already be sorted. But just in case.
	slices.SortFunc(entries, EntrySort)
	return entries, nil
}

func readEntry(r io.Reader) (e Entry, err error) {
	var length uint16
	if err = binary.Read(r, binary.LittleEndian, &length); err != nil {
		return
	}

	if err = binary.Read(r, binary.BigEndian, &e.System); err != nil {
		return
	}

	if err = binary.Read(r, binary.LittleEndian, &(e.Crc32)); err != nil {
		return
	}

	if err = binary.Read(r, binary.LittleEndian, &(e.Sig)); err != nil {
		return
	}

	if err = binary.Read(r, binary.LittleEndian, &(e.Magic)); err != nil {
		return
	}

	nameBuf := make([]byte, length-4 /*length + system*/ -4 /*crc*/ -4 /*hash*/ -4 /*unknown*/)
	if err = binary.Read(r, binary.BigEndian, &nameBuf); err != nil {
		return
	}
	// nameBuf may contain padding after the 0 terminator
	if nameStr, err := extractName(nameBuf); err != nil {
		return e, err
	} else {
		e.Name = nameStr
	}

	return
}

// extractName is a simple function that takes a byte array & looks for a zero-terminator. If found, it translates the
// bytes prior to the terminator into a string & returns that. It returns an error if the terminator is not found.
func extractName(src []byte) (string, error) {
	for i := range src {
		if src[i] == 0 {
			return string(src[:i]), nil
		}
	}

	return "", errors.New("could not find 0-terminator")
}
