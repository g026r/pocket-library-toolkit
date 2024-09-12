package model

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/g026r/pocket-library-editor/pkg/util"
)

const PlaytimeHeader uint32 = 0x01545050

type PlayTime struct {
	Added  uint32
	Played uint32
}

func ReadPlayTimes(src string) (map[uint32]PlayTime, error) {
	f, err := os.Open(fmt.Sprintf("%s/playtimes.bin", src))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var header uint32
	if err := binary.Read(f, binary.BigEndian, &header); err != nil {
		return nil, err
	}
	if header != PlaytimeHeader {
		return nil, fmt.Errorf("%s: %w", f.Name(), util.ErrUnrecognizedFileFormat)
	}

	var num uint32
	if err := binary.Read(f, binary.LittleEndian, &num); err != nil {
		return nil, err
	}

	playtimes := make(map[uint32]PlayTime, num)
	for range num {
		var k uint32
		v := PlayTime{}

		if err := binary.Read(f, binary.LittleEndian, &k); err != nil {
			return nil, err
		}
		if err = binary.Read(f, binary.LittleEndian, &v.Added); err != nil {
			return nil, err
		}
		if err = binary.Read(f, binary.LittleEndian, &v.Played); err != nil {
			return nil, err
		}
		playtimes[k] = v
	}

	return playtimes, nil
}
