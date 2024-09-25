package models

import (
	"fmt"
	"math"
	"strings"
)

var ValidThumbsFiles = []System{GB, GBA, GG, NGP, PCE, Lynx}

type System uint16

const (
	GB System = iota
	GBC
	GBA
	GG
	SMS
	NGP
	NGPC
	PCE
	Lynx
)

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

// ThumbFile maps certain systems that share thumbs.bin with others to the correct system
func (s System) ThumbFile() System {
	switch s {
	case GBC:
		return GB
	case SMS:
		return GG
	case NGPC:
		return NGP
	default:
		return s
	}
}

func (s System) PlayOffset() uint32 {
	return uint32(s) * 0x04000000
}

func Parse(s string) (System, error) {
	switch strings.ToUpper(s) {
	case "GB":
		return GB, nil
	case "GBC":
		return GBC, nil
	case "GBA":
		return GBA, nil
	case "GG":
		return GG, nil
	case "SMS":
		return SMS, nil
	case "NGP":
		return NGP, nil
	case "NGPC":
		return NGPC, nil
	case "PCE":
		return PCE, nil
	case "LYNX":
		return Lynx, nil
	default:
		return math.MaxUint16, fmt.Errorf("unknown system: %s", s)
	}
}
