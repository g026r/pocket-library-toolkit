package models

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"
)

type PlayTime struct {
	Added  uint32
	Played uint32
	System
}

func (p *PlayTime) ReadFrom(r io.Reader) (int64, error) {
	var played uint32
	if err := binary.Read(r, binary.LittleEndian, &p.Added); err != nil {
		return 0, err
	}
	if err := binary.Read(r, binary.LittleEndian, &played); err != nil {
		return 4, err
	}
	p.System = FromPlayedTime(played)
	p.Played = played - p.PlayOffset()

	return 8, nil
}

func (p *PlayTime) WriteTo(w io.Writer) (int64, error) {
	var added, played uint32
	if p.Added != 0 {
		added = p.Added
		played = p.Played
	} else {
		// Pocket doesn't know about timezones, so we have to manually apply the offset to get the correct-ish time.
		//Might get kind of funny around DST changeovers, but I can't be bothered with anything fancier.
		_, offset := time.Now().Zone()
		added = uint32(time.Now().Add(time.Second * time.Duration(offset)).Unix())
	}

	if err := binary.Write(w, binary.LittleEndian, added); err != nil {
		return 0, err
	}
	if err := binary.Write(w, binary.LittleEndian, played+p.PlayOffset()); err != nil {
		return 4, err
	}

	return 8, nil
}

func (p *PlayTime) FormatPlayTime() string {
	s := p.Played % 60
	m := (p.Played % 3600) / 60
	h := p.Played / 3600

	return fmt.Sprintf("%dh %dm %ds", h, m, s)
}
