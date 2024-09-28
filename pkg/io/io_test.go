package io

import (
	"fmt"
	"os"
	"testing"

	"github.com/g026r/pocket-toolkit/pkg/models"
)

func TestSaveInternal(t *testing.T) {
	t.SkipNow()
	t.Parallel()

	e := []models.Entry{{System: models.PCE}, {System: models.GB}}
	if err := SaveInternal(nil, e); err != nil {

	}
}

func TestReadEntries(t *testing.T) {
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
			pt, err := LoadEntries(os.DirFS(fmt.Sprintf("../../%s", k)))
			if (err != nil) != v.err {
				t.Error(err)
			} else if len(pt) != v.count {
				t.Errorf("Expected %d entries; got %d", v.count, len(pt))
			}
		})
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
			count: 229,
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
