package util

import (
	"errors"
	"math"
	"strings"
)

var ErrUnrecognizedFileFormat = errors.New("not a pocket binary file")

var ValidThumbsFiles = []System{GB, GBA, GG, NGP, PCE, Lynx}

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
