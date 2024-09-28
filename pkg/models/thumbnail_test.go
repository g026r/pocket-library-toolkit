package models

import (
	"image"
	"os"
	"testing"

	"github.com/disintegration/imaging"
)

func Test_determineResizing(t *testing.T) {
	t.Parallel()
	tc := map[string]struct {
		width          int
		height         int
		resizeByHeight bool
	}{
		"square": {
			width:  175,
			height: 175,
		},
		"very long": {
			width:  175,
			height: 128,
		},
		"very tall": {
			width:  175,
			height: 250,
		},
		"long": {
			width:  175,
			height: 170,
		},
		"tall": {
			width:  175,
			height: 186,
		},
	}

	for name, test := range tc {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			i := image.NewNRGBA(image.Rect(0, 0, test.width, test.height))
			w, h := determineResizing(i)

			i = imaging.Resize(i, w, h, imaging.Lanczos)
			if i.Rect.Max.Y < maxHeight {
				t.Errorf("Expected new height to be greater or equal to %d. Got %d", maxHeight, h)
			}
			if i.Rect.Max.X < maxWidth {
				t.Errorf("Expected new width to be greater or equal to %d. Got %d", maxWidth, w)
			}
		})
	}
}

func TestGenerateThumbnail(t *testing.T) {
	t.Parallel()
	crc := uint32(0xb8a12409)

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		f, err := os.Open("../../testdata/valid/thumbnail_output.bin")
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		fi, err := f.Stat()
		if err != nil {
			t.Fatal(err)
		}
		thumbnail := make([]byte, fi.Size())
		if _, err := f.Read(thumbnail); err != nil {
			t.Fatal(err)
		}

		img, err := GenerateThumbnail(os.DirFS("../../testdata/valid"), NGPC, crc)
		if err != nil {
			t.Fatalf("Expected nil but got %v", err)
		}
		if img.Crc32 != crc {
			t.Errorf("Expected 0x%08x but got %v", crc, img.Crc32)
		}
		if len(img.Image) != len(thumbnail) {
			t.Errorf("Thumbnail length is wrong. Expected %d, got %d", len(thumbnail), len(img.Image))
		} else {
			for i := range thumbnail {
				if thumbnail[i] != img.Image[i] {
					t.Errorf("Thumbnail does not match expected starting at byte %d", i)
					break
				}
			}
		}
	})

	t.Run("invalid header", func(t *testing.T) {
		t.Parallel()
		_, err := GenerateThumbnail(os.DirFS("testdata/invalid_header"), NGPC, crc)
		if err == nil {
			t.Error("Expected err but got nil")
		}
	})
}
