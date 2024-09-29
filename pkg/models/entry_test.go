package models

import (
	"bytes"
	"slices"
	"testing"
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
