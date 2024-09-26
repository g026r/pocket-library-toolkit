package models

import (
	"bytes"
	"encoding/binary"
	"slices"
	"testing"
	"time"
)

var rawPT = []byte{0x7C, 0x65, 0x79, 0x66, 0x24, 0x26, 0x01, 0x08}

var playtime = PlayTime{
	Added:  0x6679657c,
	Played: 0x00012624,
	System: GBA,
}

func TestPlayTime_ReadFrom(t *testing.T) {
	b := bytes.Buffer{}
	b.Write(rawPT)

	sut := PlayTime{}
	if n, err := sut.ReadFrom(&b); err != nil {
		t.Fatal(err)
	} else if n != 8 {
		t.Fatalf("Expecred 8 bytes read; read %d", n)
	}

	if sut.Added != playtime.Added {
		t.Errorf("Excpected %08x, got %08x", playtime.Added, sut.Added)
	}
	if sut.Played != playtime.Played {
		t.Errorf("Excpected %08x, got %08x", playtime.Played, sut.Played)
	}
}

func TestPlayTime_WriteTo(t *testing.T) {
	t.Parallel()

	t.Run("values set", func(t *testing.T) {
		t.Parallel()
		w := &bytes.Buffer{}
		if _, err := playtime.WriteTo(w); err != nil {
			t.Fatalf("%v", err)
		}
		if b := w.Bytes(); slices.Compare(b, rawPT) != 0 {
			t.Errorf("Expected %v; got %v", cleanEntry, b)
		}
	})

	t.Run("values unset", func(t *testing.T) {
		t.Parallel()
		w := &bytes.Buffer{}
		if _, err := (&PlayTime{System: NGPC}).WriteTo(w); err != nil {
			t.Fatalf("%v", err)
		}
		b := w.Bytes()
		added := binary.LittleEndian.Uint32(b[:4])
		play := binary.LittleEndian.Uint32(b[4:])
		_, offset := time.Now().Zone()
		// Since it's using time.Now() we're going to give it some wiggle room & say that anything within 10 seconds is fine.
		if n := time.Since(time.Unix(int64(added), 0).Add(-1 * time.Duration(offset) * time.Second)); n > 10*time.Second {
			t.Errorf("Expected within 10 seconds; got %d", n)
		}
		if play != 0x18000000 {
			t.Errorf("Expected %08x; got %08x", 0x18000000, play)
		}
	})
}

func TestPlayTime_FormatPlayTime(t *testing.T) {
	t.Parallel()

	p := PlayTime{}
	if s := p.FormatPlayTime(); s != "0h 0m 0s" {
		t.Errorf("Expected '0h 0m 0s'; got '%s'", s)
	}

	p.Played = 55
	if s := p.FormatPlayTime(); s != "0h 0m 55s" {
		t.Errorf("Expected '0h 0m 55s'; got '%s'", s)
	}

	p.Played = 147
	if s := p.FormatPlayTime(); s != "0h 2m 27s" {
		t.Errorf("Expected '0h 2m 27s'; got '%s'", s)
	}

	p.Played = 11004
	if s := p.FormatPlayTime(); s != "3h 3m 24s" {
		t.Errorf("Expected '3h 3m 24s'; got '%s'", s)
	}
}
