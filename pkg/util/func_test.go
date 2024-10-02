package util

import (
	"image"
	"testing"

	"github.com/disintegration/imaging"
)

func TestHexStringTransform(t *testing.T) {
	t.Parallel()
	cases := []struct {
		s   string
		i   uint32
		err bool
	}{
		{"0x12345678", 0x12345678, false},       // digits
		{"0X0ABCDEF0", 0x0ABCDEF0, false},       // letters, uppercase
		{"   0x0ABCDEF0   ", 0x0ABCDEF0, false}, // space padded
		{"0x123456", 0x123456, false},           // shorter than 32 bits
		{"0x1ab45de8", 0x1ab45de8, false},       // lowercase, mix of letters & numbers
		{"0x0", 0, false},                       // smallest possible value
		{"0ABCDEF0", 0x0ABCDEF0, false},         // no 0x prefix
		{"0x123456789", 0, true},                // too long
		{"0x0abcdefg", 0, true},                 // invalid characters
		{"   ", 0, false},                       // blank string
		{"0x", 0, true},                         // only prefix
	}

	for _, tc := range cases {
		t.Run(tc.s, func(t *testing.T) {
			t.Parallel()

			if s, err := HexStringTransform(tc.s); (err != nil) != tc.err {
				t.Errorf("err: %v", err)
			} else if !tc.err && s != tc.i {
				t.Errorf("Expected %x; got %x", tc.i, s)
			}
		})
	}
}

func TestDetermineResizing(t *testing.T) {
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
			w, h := DetermineResizing(i)

			i = imaging.Resize(i, w, h, imaging.Lanczos)
			if i.Rect.Max.Y < MaxHeight {
				t.Errorf("Expected new height to be greater or equal to %d. Got %d", MaxHeight, h)
			}
			if i.Rect.Max.X < MaxWidth {
				t.Errorf("Expected new width to be greater or equal to %d. Got %d", MaxWidth, w)
			}
			if i.Rect.Max.X > MaxWidth && i.Rect.Max.Y > MaxHeight {
				t.Errorf("Expected one of X or Y to be equal to MaxWidth or MaxHeight; Got %d and %d", w, h)
			}
		})
	}
}
