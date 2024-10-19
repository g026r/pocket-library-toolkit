package ui

import (
	"slices"
	"testing"

	"github.com/g026r/pocket-toolkit/pkg/io"
	"github.com/g026r/pocket-toolkit/pkg/models"
)

func TestModel_playfix(t *testing.T) {
	t.Parallel()
	var p float64
	sut := Model{
		percent: p,
		entries: []models.Entry{
			{Times: models.PlayTime{Played: 0x0000ABCD}},
			{Times: models.PlayTime{Played: 0x0100ABCD}},
			{Times: models.PlayTime{Played: 0x0400ABCD}},
			{Times: models.PlayTime{Played: 0xFF00ABCD}}},
	}

	msg := sut.playfix()
	switch msg.(type) {
	case updateMsg: // Don't need to do anything
	default:
		t.Errorf("Expected updateMsg got %v", msg)
	}

	// sut = <-sut.updates
	for _, v := range sut.entries {
		if v.Times.Played != 0x0000ABCD { // +v.SystemOffset() {
			t.Errorf("0x%02x Expected 0x0000ABCD; got 0x%08x", v.Times.Sig, v.Times.Played)
		}
	}
}

func TestModel_prune(t *testing.T) {
	t.Parallel()
	var p float64
	sut :=
		Model{
			// updates: make(chan Model, 1),
			entries: []models.Entry{{
				System: models.GB,
				Crc32:  0x12345678, // Present
			}, {
				System: models.GB,
				Crc32:  0xAAAAAAAA, // Present
			}, {
				System: models.GBA,
				Crc32:  0x66666666, // Present but different system
			}, {
				System: models.GB,
				Crc32:  0xFEDCBA09, // Not present
			}},
			thumbnails: map[models.System]models.Thumbnails{
				models.GB:  {Images: []models.Image{{Crc32: 0xAAAAAAAA}, {Crc32: 0x12345678}, {Crc32: 0x66666666}}},
				models.GBA: {Images: []models.Image{{Crc32: 0x66666666}}},
			},
			percent: p,
		}

	msg := sut.prune()
	switch msg.(type) {
	case updateMsg: // Don't need to do anything
	default:
		t.Errorf("Expected updateMsg got %v", msg)
	}

	// sut = <-sut.updates

	if gba := sut.thumbnails[models.GBA]; gba.Modified || len(gba.Images) != 1 {
		t.Errorf("GBA thumbnails should not have been modified {Modified: %t, Images: %d}", gba.Modified, len(gba.Images))
	}
	gb := sut.thumbnails[models.GB]
	if !gb.Modified {
		t.Error("GB thumbnails should be modified")
	}
	if len(gb.Images) != 2 {
		t.Errorf("Expected 2 images; found %d", len(gb.Images))
	}
	for _, x := range []uint32{0xFEDCBA09, 0x66666666} {
		if slices.ContainsFunc(gb.Images, func(image models.Image) bool {
			return image.Crc32 == x
		}) {
			t.Errorf("Image %08x should not be present", x)
		}
	}
	for _, x := range []uint32{0x12345678, 0xAAAAAAAA} {
		if !slices.ContainsFunc(gb.Images, func(image models.Image) bool {
			return image.Crc32 == x
		}) {
			t.Errorf("Image %08x should be present", x)
		}
	}
}

func TestModel_configChange(t *testing.T) {
	// Simple test to make certain that the correct values are being flipped
	t.Parallel()
	config := io.Config{}
	sut := Model{Config: &config}

	// test all false -> true
	m, _ := sut.configChange(cfgShowAdd)
	if !m.ShowAdd || m.AdvancedEditing || m.RemoveImages {
		t.Errorf("Expected ShowAdd to be true: %v", *m.Config)
	}
	*m.Config = io.Config{}
	m, _ = sut.configChange(cfgRmThumbs)
	if !m.RemoveImages || m.AdvancedEditing || m.ShowAdd {
		t.Errorf("Expected RemoveImages to be true: %v", *m.Config)
	}
	*m.Config = io.Config{}
	m, _ = sut.configChange(cfgAdvEdit)
	if !m.AdvancedEditing || m.ShowAdd || m.RemoveImages {
		t.Errorf("Expected AdvancedEditing to be true: %v", *m.Config)
	}

	allTrue := io.Config{
		RemoveImages:    true,
		AdvancedEditing: true,
		ShowAdd:         true,
	}
	// test true -> false
	*sut.Config = allTrue
	m, _ = sut.configChange(cfgShowAdd)
	if m.ShowAdd || !m.AdvancedEditing || !m.RemoveImages {
		t.Errorf("Expected ShowAdd to be false: %v", *m.Config)
	}
	*sut.Config = allTrue
	m, _ = sut.configChange(cfgRmThumbs)
	if m.RemoveImages || !m.AdvancedEditing || !m.ShowAdd {
		t.Errorf("Expected RemoveImages to be false: %v", *m.Config)
	}
	*sut.Config = allTrue
	m, _ = sut.configChange(cfgAdvEdit)
	if m.AdvancedEditing || !m.RemoveImages || !m.ShowAdd {
		t.Errorf("Expected AdvancedEditing to be false: %v", *m.Config)
	}
}
