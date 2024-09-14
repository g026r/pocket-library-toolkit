package pkg

import (
	"slices"
	"testing"

	"github.com/g026r/pocket-library-editor/pkg/model"
	"github.com/g026r/pocket-library-editor/pkg/util"
)

func TestApplication_prune(t *testing.T) {
	t.Parallel()
	sut :=
		Application{
			Entries: []model.Entry{{
				System: util.GB,
				Crc32:  0x12345678, // Present
			}, {
				System: util.GB,
				Crc32:  0xAAAAAAAA, // Present
			}, {
				System: util.GBA,
				Crc32:  0x66666666, // Present but different system
			}, {
				System: util.GB,
				Crc32:  0xFEDCBA09, // Not present
			}},
			Thumbs: map[util.System]model.Thumbnails{
				util.GB:  {Images: []model.Image{{Crc32: 0xAAAAAAAA}, {Crc32: 0x12345678}, {Crc32: 0x66666666}}},
				util.GBA: {Images: []model.Image{{Crc32: 0x66666666}}},
			},
		}

	sut.prune()

	if gba := sut.Thumbs[util.GBA]; gba.Modified || len(gba.Images) != 1 {
		t.Errorf("GBA thumbnails should not have been modified {Modified: %t, Images: %d}", gba.Modified, len(gba.Images))
	}
	gb := sut.Thumbs[util.GB]
	if !gb.Modified {
		t.Error("GB thumbnails should be modified")
	}
	if len(gb.Images) != 2 {
		t.Errorf("Expected 2 images; found %d", len(gb.Images))
	}
	for _, x := range []uint32{0xFEDCBA09, 0x66666666} {
		if slices.ContainsFunc(gb.Images, func(image model.Image) bool {
			return image.Crc32 == x
		}) {
			t.Errorf("Image %08x should not be present", x)
		}
	}
	for _, x := range []uint32{0x12345678, 0xAAAAAAAA} {
		if !slices.ContainsFunc(gb.Images, func(image model.Image) bool {
			return image.Crc32 == x
		}) {
			t.Errorf("Image %08x should be present", x)
		}
	}
}
