package model

import (
	"bytes"
	"encoding/binary"
	"os"
	"slices"
	"testing"
	"time"
)

var rawPT = []byte{0x7C, 0x65, 0x79, 0x66, 0x24, 0x26, 0x01, 0x08}

//var new = []byte{0x7C, 0x65, 0x79, 0x66, 0x00, 0x00, 0x00, 0x00}

var playtime = PlayTime{
	Added:  0x6679657c,
	Played: 0x08012624,
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
		if _, err := (PlayTime{}).WriteTo(w); err != nil {
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
		if play != 0 {
			t.Errorf("Expected 0; got %d", play)
		}
	})
}

func TestReadPlayTimes(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		count int
		err   bool
	}{
		"testdata/count_mismatch": {
			count: 4,
		},
		"testdata/invalid_header": {
			err: true,
		},
		"testdata/valid": {
			count: 229,
		},
	}

	for k, v := range cases {
		t.Run(k, func(t *testing.T) {
			t.Parallel()
			pt, err := ReadPlayTimes(os.DirFS(k))
			if (err != nil) != v.err {
				t.Error(err)
			} else if len(pt) != v.count {
				t.Errorf("Expected %d entries; got %d", v.count, len(pt))
			}
		})
	}
}
