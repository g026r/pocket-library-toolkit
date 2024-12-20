package models

import (
	"cmp"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"
)

var EntrySort = func(a, b Entry) int {
	if val := cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)); val != 0 {
		return val
	}
	if val := cmp.Compare(a.System, b.System); val != 0 {
		return val
	}
	if val := cmp.Compare(a.Sig, b.Sig); val != 0 {
		return val
	}
	if val := cmp.Compare(a.Crc32, b.Crc32); val != 0 {
		return val
	}
	return cmp.Compare(a.Magic, b.Magic) // Technically system & magic number should be tied together. But you never know.
}

type Thumbnails struct {
	Modified bool
	Images   []Image
}
type Image struct {
	Crc32 uint32
	Image []byte
}

type Entry struct {
	System
	Crc32 uint32
	Sig   uint32
	Magic uint32
	Name  string
	Times PlayTime
}

func (e Entry) FilterValue() string {
	return fmt.Sprintf("%s (%s)", e.Name, e.System)
}

func (e Entry) String() string {
	return e.Name
}

// CalculateLength returns the length in bytes of the library entry
// includes any extra padding needed to get it to a word boundary & can be used to calculate offsets
func (e Entry) CalculateLength() uint16 {
	length := 4 /*length + system*/ + 4 /*crc*/ + 4 /*hash*/ + 4 /*magic*/ + uint16(len(e.Name)) + 1 /*0-terminator*/
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

func (e *Entry) ReadFrom(r io.Reader) (int64, error) {
	var length uint16
	if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
		return 0, err
	}

	if err := binary.Read(r, binary.BigEndian, &e.System); err != nil {
		return 2, err
	}

	if err := binary.Read(r, binary.LittleEndian, &(e.Crc32)); err != nil {
		return 4, nil
	}

	if err := binary.Read(r, binary.LittleEndian, &(e.Sig)); err != nil {
		return 8, nil
	}

	if err := binary.Read(r, binary.LittleEndian, &(e.Magic)); err != nil {
		return 12, nil
	}

	nameBuf := make([]byte, length-4 /*length + system*/ -4 /*crc*/ -4 /*hash*/ -4 /*magic*/)
	if err := binary.Read(r, binary.BigEndian, &nameBuf); err != nil {
		return 16, err
	}
	// nameBuf may contain padding after the 0 terminator
	if nameStr, err := extractName(nameBuf); err != nil {
		return int64(length), err
	} else {
		e.Name = nameStr
	}

	return int64(length), nil
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

type PlayTime struct {
	Added  uint32
	Played uint32
	// A bit redundant, as the entry this is associated with also stores the signature & system. But having these as
	// part of it reduces the number of things we need to pass to methods
	Sig uint32
	System
}

func (p *PlayTime) ReadFrom(r io.Reader) (int64, error) {
	var played uint32
	if err := binary.Read(r, binary.LittleEndian, &p.Sig); err != nil {
		return 0, err
	}
	if err := binary.Read(r, binary.LittleEndian, &p.Added); err != nil {
		return 4, err
	}
	if err := binary.Read(r, binary.LittleEndian, &played); err != nil {
		return 8, err
	}
	p.System = FromPlayedTime(played)
	p.Played = played - p.PlayOffset()

	return 12, nil
}

func (p *PlayTime) WriteTo(w io.Writer) (int64, error) {
	var added, played uint32
	if p.Added != 0 {
		added = p.Added
		played = p.Played
	} else {
		// Pocket doesn't know about timezones, so we have to manually apply the offset to get the correct-ish time.
		// Might get kind of funny around DST changeovers, but I can't be bothered with anything fancier.
		_, offset := time.Now().Zone()
		added = uint32(time.Now().Add(time.Second * time.Duration(offset)).Unix())
	}

	if err := binary.Write(w, binary.LittleEndian, p.Sig); err != nil {
		return 0, err
	}
	if err := binary.Write(w, binary.LittleEndian, added); err != nil {
		return 4, err
	}
	if err := binary.Write(w, binary.LittleEndian, played+p.PlayOffset()); err != nil {
		return 8, err
	}

	return 12, nil
}

func (p *PlayTime) FormatPlayTime() string {
	s := p.Played % 60
	m := (p.Played % 3600) / 60
	h := p.Played / 3600

	return fmt.Sprintf("%dh %dm %ds", h, m, s)
}
