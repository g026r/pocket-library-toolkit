package models

import (
	"encoding/binary"
	"io"
	"time"
)

type PlayTime struct {
	Added  uint32
	Played uint32
}

func (p *PlayTime) ReadFrom(r io.Reader) (int64, error) {
	if err := binary.Read(r, binary.LittleEndian, &p.Added); err != nil {
		return 0, err
	}
	if err := binary.Read(r, binary.LittleEndian, &p.Played); err != nil {
		return 4, err
	}

	return 8, nil
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
