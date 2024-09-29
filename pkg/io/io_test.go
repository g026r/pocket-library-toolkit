package io

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/g026r/pocket-toolkit/pkg/models"
)

func TestSaveInternal(t *testing.T) {
	t.Parallel()

	// Test to make certain it doesn't write multiple systems out.
	e := []models.Entry{{System: models.PCE}, {System: models.GB}}
	if err := SaveInternal(nil, e); err == nil {
		t.Errorf("Expected err but got nil")
	}
}

func TestLoadThumbs(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		count int
		err   bool
	}{
		"testdata/count_mismatch": {
			count: 2,
		},
		"testdata/invalid_header": {
			err: true,
		},
		"testdata/valid": {
			count: 7,
		},
	}

	for k, v := range cases {
		t.Run(k, func(t *testing.T) {
			t.Parallel()
			pt, err := LoadThumbs(os.DirFS(fmt.Sprintf("../../%s", k)))
			if (err != nil) != v.err {
				t.Error(err)
			}
			if !v.err {
				if len(pt) != 1 {
					t.Errorf("Expected 1 system entries; got %d", len(pt))
				} else if tn, ok := pt[models.NGP]; !ok {
					t.Errorf("Expected NGP entry to be present")
				} else if len(tn.Images) != v.count {
					t.Errorf("Expected %d images; got %d", v.count, len(tn.Images))
				}

				for _, img := range pt[models.NGP].Images {
					if img.Crc32 == 0 {
						t.Errorf("Expected CRC32 value to be present")
					}
					if len(img.Image) == 0 {
						t.Errorf("Expected image to be greater than 0 bytes")
					}
				}
			}
		})
	}
}

func TestLoadPlaytimes(t *testing.T) {
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
			count: 230,
		},
	}

	for k, v := range cases {
		t.Run(k, func(t *testing.T) {
			t.Parallel()
			pt, err := LoadPlaytimes(os.DirFS(fmt.Sprintf("../../%s", k)))
			if (err != nil) != v.err {
				t.Error(err)
			} else if len(pt) != v.count {
				t.Errorf("Expected %d entries; got %d", v.count, len(pt))
			}
		})
	}
}

func TestSaveThumbsFile(t *testing.T) {
	t.Parallel()
	f, err := os.Open("../../testdata/thumbs.bin")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	thumbs_bin := make([]byte, fi.Size())
	if _, err := f.Read(thumbs_bin); err != nil {
		t.Fatal(err)
	}

	w := &bytes.Buffer{}
	img := []models.Image{
		{Crc32: 0x01234567, Image: []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77}}, // It can be junk data. Just should be divisible by 4.
		{Crc32: 0xfedcba98, Image: []byte{0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99, 0x88}},
	}
	tick := make(chan any, 1)
	defer close(tick)
	go func() {
		for _ = range tick {
		} // Do nothing. We're just trying to keep the program from deadlocking
	}()

	if err := SaveThumbsFile(w, img, tick); err != nil {
		t.Errorf("Expected nil; got %v", err)
	}
	out := w.Bytes()
	if len(out) != len(thumbs_bin) {
		t.Errorf("thumbs.bin length is wrong. Expected %d, got %d", len(thumbs_bin), len(out))
	} else {
		for i := range out {
			if out[i] != thumbs_bin[i] {
				t.Errorf("thumbs.bin does not match expected starting at byte 0x%04x", i)
				break
			}
		}
	}

}

func TestLoadEntries(t *testing.T) {
	t.Parallel()
	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		e, err := LoadEntries(os.DirFS("../../testdata/valid/"))
		if err != nil {
			t.Errorf("Expected nil got %v", err)
		}
		if len(e) != 230 {
			t.Errorf("Expected 299 entries; got %d", len(e))
		}

		// Verify first, last, and a midpoint entry
		sut := e[0]
		if sut.Name != "1943 Kai" {
			t.Errorf("Expected %q, got %q", "1943 Kai", sut.Name)
		}
		if sut.Sig != 0x79cd1a3e {
			t.Errorf("Expected 0x%08x, got 0x%08x", 0x79cd1a3e, sut.Sig)
		}
		if sut.Crc32 != 0xfde08d6d {
			t.Errorf("Expected 0x%08x, got 0x%08x", 0xfde08d6d, sut.Crc32)
		}
		if sut.Magic != 0x1b94 {
			t.Errorf("Expected 0x%04x, got 0x%04x", 0x1b94, sut.Magic)
		}
		if sut.System != models.PCE {
			t.Errorf("Expected 0x%08x, got 0x%08x", models.PCE.String(), sut.System.String())
		}

		sut = e[len(e)-1]
		if sut.Name != "Xevious - Fardraut Saga" {
			t.Errorf("Expected %q, got %q", "Xevious - Fardraut Saga", sut.Name)
		}
		if sut.Sig != 0xd4c0eadc {
			t.Errorf("Expected 0x%08x, got 0x%08x", 0xd4c0eadc, sut.Sig)
		}
		if sut.Crc32 != 0xf8f85eec {
			t.Errorf("Expected 0x%08x, got 0x%08x", 0xf8f85eec, sut.Crc32)
		}
		if sut.Magic != 0x1d1e {
			t.Errorf("Expected 0x%04x, got 0x%04x", 0x1d1e, sut.Magic)
		}
		if sut.System != models.PCE {
			t.Errorf("Expected 0x%08x, got 0x%08x", models.PCE.String(), sut.System.String())
		}

		sut = e[len(e)/2]
		if sut.Name != "Kinetic Connection" {
			t.Errorf("Expected %q, got %q", "Kinetic Connection", sut.Name)
		}
		if sut.Sig != 0x9bb34518 {
			t.Errorf("Expected 0x%08x, got 0x%08x", 0x9bb34518, sut.Sig)
		}
		if sut.Crc32 != 0x4af7f2aa {
			t.Errorf("Expected 0x%08x, got 0x%08x", 0x4af7f2aa, sut.Crc32)
		}
		if sut.Magic != 0x1817 {
			t.Errorf("Expected 0x%04x, got 0x%04x", 0x1817, sut.Magic)
		}
		if sut.System != models.GG {
			t.Errorf("Expected 0x%08x, got 0x%08x", models.GG.String(), sut.System.String())
		}
	})

	t.Run("invalid header", func(t *testing.T) {
		t.Parallel()
		_, err := LoadEntries(os.DirFS("../../testdata/invalid_header/"))
		if err == nil {
			t.Error("Expected err but got nil")
		}
	})

	t.Run("count mismatch", func(t *testing.T) {
		t.Parallel()
		e, err := LoadEntries(os.DirFS("../../testdata/count_mismatch"))
		if err != nil {
			t.Errorf("Expected nil got %v", err)
		}
		if len(e) != 4 {
			t.Errorf("Expected 299 entries; got %d", len(e))
		}
	})
}

func TestSaveLibrary(t *testing.T) {
	t.Parallel()
	// Just going to load in the files & compare them to what we would write out to ensure they're the same.
	// This does mean that list.bin is using a modified file to begin with, since we just use 0s for the filler data after
	// a string terminator
	dir := os.DirFS("../../testdata/valid/")
	e, err := LoadEntries(dir)
	if err != nil {
		t.Fatal(err)
	}
	cmpList, err := dir.Open("System/Played Games/list.bin")
	if err != nil {
		t.Fatal(err)
	}
	fi, err := cmpList.Stat()
	if err != nil {
		t.Fatal(err)
	}
	list_bin := make([]byte, fi.Size())
	if _, err := cmpList.Read(list_bin); err != nil {
		t.Fatal(err)
	}

	p, err := LoadPlaytimes(dir)
	if err != nil {
		t.Fatal(err)
	}
	cmpPlay, err := dir.Open("System/Played Games/playtimes.bin")
	if err != nil {
		t.Fatal(err)
	}
	fi, err = cmpPlay.Stat()
	if err != nil {
		t.Fatal(err)
	}
	playtimes_bin := make([]byte, fi.Size())
	if _, err := cmpPlay.Read(playtimes_bin); err != nil {
		t.Fatal(err)
	}

	tick := make(chan any, 1)
	defer close(tick)
	go func() {
		for _ = range tick {
		} // Do nothing. We're just trying to keep the program from deadlocking
	}()

	list := &bytes.Buffer{}
	play := &bytes.Buffer{}
	err = SaveLibrary(list, e, play, p, tick)
	if err != nil {
		t.Errorf("Expected nil but got %v", err)
	}
	list_bytes := list.Bytes()
	play_bytes := play.Bytes()
	if len(list_bytes) != len(list_bin) {
		t.Errorf("Expected %d bytes but got %d", len(list_bin), len(list_bytes))
	} else {
		for i := range list_bytes {
			if list_bytes[i] != list_bin[i] {
				t.Errorf("list.bin differs starting at byte 0x%04x", i)
				break
			}
		}
	}
	if len(play_bytes) != len(playtimes_bin) {
		t.Errorf("Expected %d bytes but got %d", len(playtimes_bin), len(play_bytes))
	} else {
		for i := range play_bytes {
			if play_bytes[i] != playtimes_bin[i] {
				t.Errorf("playtimes.bin differs starting at byte 0x%04x", i)
				break
			}
		}
	}

}
