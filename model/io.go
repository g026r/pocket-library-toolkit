package model

import (
	"encoding/binary"
	"errors"
	"os"
	"slices"
)

var ErrUnrecognizedFileFormat = errors.New("not a pocket library file")

const firstOffset uint32 = 0x4010

func ReadEntries(src string) ([]Entry, error) {
	f, err := os.Open(src)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var header, num, unknown uint32
	if err = binary.Read(f, binary.BigEndian, &header); err != nil {
		return nil, err
	}
	if header != LibraryHeader { // Missing the magic number = not a Pocket library file
		return nil, ErrUnrecognizedFileFormat
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

	// Parse the library entries. blank entries are marked as 0s.
	// TODO: If we remove an entry, does it replace the extra offsets with 0 or does it just shift everything & change the count?
	offsets := make([]uint32, 0)
	var offset uint32
	offset = 0xFFFFFFFF
	for offset != 0 {
		if err = binary.Read(f, binary.LittleEndian, &offset); err != nil {
			return nil, err
		}
		if offset != 0 {
			offsets = append(offsets, offset)
		}
		if uint32(len(offsets)) > num {
			break
		}
	}
	if offset != 0 { // If we didn't end because of a 0, we have a problem
		return nil, errors.New("entry count mismatch")
	}

	// Parse each of the library entries
	entries := make([]Entry, num)
	for i := range offsets {
		if _, err := f.Seek(int64(offsets[i]), 0); err != nil {
			return entries, err
		}

		if e, err := ReadEntry(f); err != nil {
			return entries, err
		} else {
			entries[i] = e
		}
	}

	// Should already be sorted. But just in case.
	//slices.SortFunc(entries, model.SortFunc)
	return entries, nil
}

func WriteFiles(infile string, entries []Entry, playtimes map[uint32]PlayTime) error {
	//dirStr, err := filepath.Abs(filepath.Dir(infile))
	//if err != nil {
	//	return err
	//}
	//
	//l, err := os.CreateTemp(dirStr, "tmp_")
	l, err := os.Create("/Users/g026r/Downloads/list.bin")
	if err != nil {
		return err
	}
	defer l.Close()

	p, err := os.Create("/Users/g026r/Downloads/playtimes.bin")
	if err != nil {
		return err
	}
	defer p.Close()

	binary.Write(l, binary.BigEndian, LibraryHeader)
	binary.Write(l, binary.LittleEndian, uint32(len(entries)))
	binary.Write(l, binary.LittleEndian, uint32(0x10)) // TODO: I don't know what this value signifies
	binary.Write(l, binary.LittleEndian, firstOffset)  // This seems to be duplicated? I dunno

	binary.Write(p, binary.BigEndian, PlaytimeHeader)
	binary.Write(p, binary.LittleEndian, uint32(len(entries)))

	// Build the offset entries
	slices.SortFunc(entries, SortFunc)
	offsets := make([]uint32, firstOffset/4-4)
	offsets[0] = firstOffset
	last := firstOffset
	for i := 1; i < len(entries); i++ {
		offsets[i] = last + uint32(entries[i-1].CalculateLength())
		last = offsets[i]
	}

	binary.Write(l, binary.LittleEndian, offsets)

	for _, e := range entries {
		e.WriteTo(l)
		binary.Write(p, binary.LittleEndian, e.Hash)
		if t, ok := playtimes[e.Hash]; ok {
			binary.Write(p, binary.LittleEndian, t.First)
			binary.Write(p, binary.LittleEndian, t.Second)
		} else {
			// TODO: What are sane defaults here?
			binary.Write(p, binary.LittleEndian, uint32(0))
			binary.Write(p, binary.LittleEndian, uint32(0))
		}
	}

	return nil
}
