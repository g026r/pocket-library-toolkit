package models

import (
	"image"
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
