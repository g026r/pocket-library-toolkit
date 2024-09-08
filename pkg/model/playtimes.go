package model

import (
	"encoding/binary"
	"errors"
	"os"
)

const PlaytimeHeader uint32 = 0x01545050

type PlayTime struct {
	Added  uint32
	Played uint32
}

func ReadPlayTimes(src string) (map[uint32]PlayTime, error) {
	f, err := os.Open(src)
	defer func(f *os.File) {
		_ = f.Close()
	}(f)
	if err != nil {
		return nil, err
	}

	var header uint32
	if err := binary.Read(f, binary.BigEndian, &header); err != nil {
		return nil, err
	}
	if header != PlaytimeHeader {
		return nil, errors.New("not a valid Analogue play times file")
	}

	var num uint32
	if err := binary.Read(f, binary.LittleEndian, &num); err != nil {
		return nil, err
	}

	playtimes := make(map[uint32]PlayTime, num)
	for i := uint32(0); i < num; i++ {
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
