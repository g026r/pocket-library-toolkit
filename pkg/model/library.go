package model

import (
	"cmp"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"strings"
)

type System uint16

const (
	GB   System = iota
	GBC  System = iota
	GBA  System = iota
	GG   System = iota
	SMS  System = iota
	NGP  System = iota // TODO: Verify
	NGPC System = iota
	PCE  System = iota
	Lynx System = iota
)

const (
	LibraryHeader uint32 = 0x01464154
)

var SortFunc = func(a, b Entry) int {
	return cmp.Compare(a.Name, b.Name)
}

type Entry struct {
	System
	Crc32   uint32
	Sig     uint32
	Unknown uint32 // TODO: What is this?
	Name    string
}

func (s System) String() string {
	switch s {
	case GB:
		return "GB"
	case GBC:
		return "GBC"
	case GBA:
		return "GBA"
	case GG:
		return "GG"
	case SMS:
		return "SMS"
	case NGP:
		return "NGP"
	case NGPC:
		return "NGPC"
	case PCE:
		return "PCE"
	case Lynx:
		return "Lynx"
	default:
		return "unknown"
	}
}

func Parse(s string) (System, error) {
	switch strings.ReplaceAll(strings.ToLower(s), " ", "") {
	case "gameboy", "dmg", "gb":
		return GB, nil
	case "gameboycolor", "gameboycolour", "gbcolor", "gbcolour", "gbc":
		return GBC, nil
	case "gameboyadvance", "agb", "gba":
		return GBA, nil
	case "gamegear", "segagamegear", "gg":
		return GG, nil
	case "segamastersystem", "mastersystem", "sms":
		return SMS, nil
	case "neogeopocket", "ngp":
		return NGP, nil
	case "neogeo", "neogeopocketcolor", "neogeopocketcolour", "ngpc":
		return NGPC, nil
	case "pcengine", "necpcengine", "necpce", "pce", "turbografx", "turbografx16", "necturbografx", "necturbografx16",
		"turbografix", "turbografix16", "necturbografix", "necturbografix16", "turbographics", "turbographics16",
		"necturbographics", "necturbographics16", "turbographix", "turbographix16", "necturbographix", "necturbographix16",
		"tg16", "supergrafx", "supergrafix", "supergraphics", "supergraphix", "sgfx", "sfx", "sgx", "nec":
		return PCE, nil
	case "atari", "lynx", "atarilynx", "lnx":
		return Lynx, nil
	default:
		return math.MaxUint16, errors.New("unknown system")
	}
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

	if err = binary.Write(w, binary.LittleEndian, e.Unknown); err != nil {
		return
	}
	n = n + 4

	// Write the string plus the terminator
	zeroTerm := append([]byte(e.Name), 0x00)
	if err = binary.Write(w, binary.BigEndian, zeroTerm); err != nil {
		return
	}
	n = n + int64(len(zeroTerm))

	if extra := n % 4; extra != 0 { // If we haven't finished on a word boundary then we need to pad it out
		tmp, err := w.Write(make([]byte, 4-extra)) // it's filler and all 0s anyway, so don't care about endian-ness
		n = n + int64(tmp)
		if err != nil {
			return n, err
		}
	}

	return
}

func ReadEntry(r io.Reader) (e Entry, err error) {
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

	if err = binary.Read(r, binary.LittleEndian, &(e.Unknown)); err != nil {
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
