package models

import (
	"bytes"
	"encoding/binary"
	"slices"
	"testing"
	"time"
)

// rawEntry is the binary representation of entry, as copied from a valid list.bin
var rawEntry = []byte{0x1C, 0x00, 0x00, 0x07, 0x6D, 0x8D, 0xE0, 0xFD, 0x3E, 0x1A, 0xCD, 0x79, 0x94, 0x1B, 0x00, 0x00, 0x31, 0x39, 0x34, 0x33, 0x20, 0x4B, 0x61, 0x69, 0x00, 0x45, 0x00, 0xA0}

// cleanEntry is rawEntry but with the last 3 padding bytes replaced by 0s, since Analogue sometimes just has garbage in there
var cleanEntry = []byte{0x1C, 0x00, 0x00, 0x07, 0x6D, 0x8D, 0xE0, 0xFD, 0x3E, 0x1A, 0xCD, 0x79, 0x94, 0x1B, 0x00, 0x00, 0x31, 0x39, 0x34, 0x33, 0x20, 0x4B, 0x61, 0x69, 0x00, 0x00, 0x00, 0x00}

// entry is rawEntry in properly parsed format
var entry = Entry{
	System: PCE,
	Crc32:  0xfde08d6d,
	Sig:    0x79cd1a3e,
	Magic:  0x1b94,
	Name:   "1943 Kai",
}

func TestEntry_ReadFrom(t *testing.T) {
	t.Parallel()

	var e Entry
	_, err := e.ReadFrom(bytes.NewReader(rawEntry))
	if err != nil {
		t.Errorf("%v", err)
	}

	if e.Name != entry.Name {
		t.Errorf("Expected %v; got %v", entry.Name, e.Name)
	}
	if e.Crc32 != entry.Crc32 {
		t.Errorf("Expected 0x%08x; got 0x%08x", entry.Crc32, e.Crc32)
	}
	if e.System != entry.System {
		t.Errorf("Expected 0x%08x; got 0x%08x", entry.System, e.System)
	}
	if e.Magic != entry.Magic {
		t.Errorf("Expected 0x%08x; got 0x%08x", entry.Magic, e.Magic)
	}
	if e.Sig != entry.Sig {
		t.Errorf("Expected 0x%08x; got 0x%08x", entry.Sig, e.Sig)
	}
}

func TestEntry_WriteTo(t *testing.T) {
	t.Parallel()
	w := &bytes.Buffer{}

	if _, err := entry.WriteTo(w); err != nil {
		t.Fatalf("%v", err)
	}
	if b := w.Bytes(); slices.Compare(b, cleanEntry) != 0 {
		t.Errorf("Expected %v; got %v", cleanEntry, b)
	}
}

func TestEntry_CalculateLength(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		len  uint16
	}{
		{entry.Name, 28}, // Real test
		{"", 20},
		{"i", 20},
		{"ii", 20},
		{"iii", 20},
		{"four", 24}, // Passed the word boundary. Need to pad
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e := Entry{Name: tc.name}

			if n := e.CalculateLength(); n != tc.len {
				t.Errorf("%d", n)
			}
		})
	}
}

func TestExtractName(t *testing.T) {
	t.Parallel()
	t.Run(entry.Name, func(t *testing.T) {
		t.Parallel()
		n, err := extractName(rawEntry[16:])
		if err != nil {
			t.Errorf("%v", err)
		}
		if n != entry.Name {
			t.Errorf("Expected %v; got %v", entry.Name, n)
		}
	})

	t.Run("no terminator", func(t *testing.T) {
		t.Parallel()
		b := []byte{0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30}
		_, err := extractName(b)
		if err == nil {
			t.Error("Expected error; got nil")
		}
	})
}

var rawPT = []byte{0x3E, 0x1A, 0xCD, 0x79, 0x7C, 0x65, 0x79, 0x66, 0x24, 0x26, 0x01, 0x08}

var playtime = PlayTime{
	Added:  0x6679657c,
	Played: 0x00012624,
	System: GBA,
	Sig:    0x79CD1A3E,
}

func TestPlayTime_ReadFrom(t *testing.T) {
	b := bytes.Buffer{}
	b.Write(rawPT)

	sut := PlayTime{}
	if n, err := sut.ReadFrom(&b); err != nil {
		t.Fatal(err)
	} else if n != 12 {
		t.Fatalf("Expecred 12 bytes read; read %d", n)
	}

	if sut.Added != playtime.Added {
		t.Errorf("Excpected %08x, got %08x", playtime.Added, sut.Added)
	}
	if sut.Played != playtime.Played {
		t.Errorf("Excpected %08x, got %08x", playtime.Played, sut.Played)
	}
	if sut.Sig != playtime.Sig {
		t.Errorf("Excpected %08x, got %08x", playtime.Sig, sut.Sig)
	}
	if sut.System != playtime.System {
		t.Errorf("Excpected %v, got %v", playtime.System, sut.System)
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
		if _, err := (&PlayTime{System: NGPC, Sig: 0x12345678}).WriteTo(w); err != nil {
			t.Fatalf("%v", err)
		}
		b := w.Bytes()
		sig := binary.LittleEndian.Uint32(b[:4])
		added := binary.LittleEndian.Uint32(b[4:8])
		play := binary.LittleEndian.Uint32(b[8:])
		_, offset := time.Now().Zone()
		// Since it's using time.Now() we're going to give it some wiggle room & say that anything within 10 seconds is fine.
		if n := time.Since(time.Unix(int64(added), 0).Add(-1 * time.Duration(offset) * time.Second)); n > 10*time.Second {
			t.Errorf("Expected within 10 seconds; got %d", n)
		}
		if play != 0x18000000 {
			t.Errorf("Expected %08x; got %08x", 0x18000000, play)
		}
		if sig != 0x12345678 {
			t.Errorf("Expected %08x; got %08x", 0x12345678, sig)
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
