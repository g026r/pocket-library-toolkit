package model

import (
	"image"
	"io/fs"
	"testing"

	"github.com/disintegration/imaging"

	"github.com/g026r/pocket-library-editor/pkg/util"
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

func TestLoadThumbnails(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		count int
		err   bool
	}{"tests/count_mismatch": {
		count: 2,
	},
		"tests/invalid_header": {
			err: true,
		},
		"tests/valid": {
			count: 7,
		}}

	for k, v := range cases {
		t.Run(k, func(t *testing.T) {
			t.Parallel()
			fsys, err := fs.Sub(files, k)
			if err != nil {
				t.Fatal(err)
			}
			pt, err := LoadThumbnails(fsys)
			if (err != nil) != v.err {
				t.Error(err)
			}
			if !v.err {
				if len(pt) != 1 {
					t.Errorf("Expected 1 system entries; got %d", len(pt))
				} else if tn, ok := pt[util.NGP]; !ok {
					t.Errorf("Expected NGP entry to be present")
				} else if len(tn.Images) != v.count {
					t.Errorf("Expected %d images; got %d", v.count, len(tn.Images))
				}
			}
		})
	}
}
