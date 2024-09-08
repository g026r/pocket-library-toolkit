package model

import (
	"encoding/binary"
	"fmt"
	"io"
	"io/fs"
	"time"

	"github.com/g026r/pocket-library-editor/pkg/util"
)

const PlaytimeHeader uint32 = 0x01545050

type PlayTime struct {
	Added  uint32
	Played uint32
}

func ReadPlayTimes(fs fs.FS) (map[uint32]PlayTime, error) {
	f, err := util.ReadSeeker(fs, "playtimes.bin")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var header uint32
	if err := binary.Read(f, binary.BigEndian, &header); err != nil {
		return nil, err
	}
	if header != PlaytimeHeader {
		return nil, fmt.Errorf("playtimes.bin: %w", util.ErrUnrecognizedFileFormat)
	}

	var num uint32
	if err := binary.Read(f, binary.LittleEndian, &num); err != nil {
		return nil, err
	}

	playtimes := make(map[uint32]PlayTime, num)
	var sig uint32
	for range num {
		v := PlayTime{}

		if err := binary.Read(f, binary.LittleEndian, &sig); err != nil {
			return nil, err
		}
		if err := binary.Read(f, binary.LittleEndian, &v.Added); err != nil {
			return nil, err
		}
		if err := binary.Read(f, binary.LittleEndian, &v.Played); err != nil {
			return nil, err
		}
		playtimes[sig] = v
	}

	return playtimes, nil
}

func (p PlayTime) WriteTo(w io.Writer) (int64, error) {
	if p.Added != 0 {
		if err := binary.Write(w, binary.LittleEndian, p.Added); err != nil {
			return 0, err
		}
		if err := binary.Write(w, binary.LittleEndian, p.Played); err != nil {
			return 4, err
		}
	} else {
		// Pocket doesn't know about timezones, so we have to manually apply the offset to get the correct-ish time.
		//Might get kind of funny around DST changeovers, but I can't be bothered with anything fancier.
		_, offset := time.Now().Zone()

		// Time.Unix() is an int64 but the pocket uses a 32 bit unsigned int
		// Since we don't have played times for these games letting the zeros overflow into the played time word is
		// a simple enough solution
		if err := binary.Write(w, binary.LittleEndian, uint64(time.Now().Add(time.Second*time.Duration(offset)).Unix())); err != nil {
			return 0, err
		}
	}

	return 8, nil
}
