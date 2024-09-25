package util

import (
	"slices"
	"strings"
	"testing"
)

var allSystems = []System{GB, GBC, GBA, GG, SMS, NGP, NGPC, PCE, Lynx}

func TestSystem_ThumbFile(t *testing.T) {
	t.Parallel()
	for _, s := range allSystems {
		t.Run(s.String(), func(t *testing.T) {
			result := s.ThumbFile()
			if !slices.Contains(ValidThumbsFiles, result) {
				t.Error(s.ThumbFile().String())
			}
		})
	}
}

func TestSystem_String(t *testing.T) {
	// Test to make certain each system's String() results maps back to what we expect for Parse
	t.Parallel()

	for _, tc := range allSystems {
		t.Run(tc.String(), func(t *testing.T) {
			t.Parallel()
			if s, err := Parse(tc.String()); err != nil {
				t.Errorf("err: %v", err)
			} else if s != tc {
				t.Errorf("Expected %d; got %d", tc, s)
			}
		})
	}
	t.Run("unknown", func(t *testing.T) {
		t.Parallel()
		if s := System(9).String(); s != "unknown" {
			t.Errorf("Expected unknown; got %s", s)
		}
	})
}

func TestParse(t *testing.T) {
	// Test invalid systems & lowercase, as valid ones were in the  above test
	t.Parallel()
	cases := []struct {
		s string
		System
		err bool
	}{
		{strings.ToLower(GB.String()), GB, false}, // lowercase
		{"   ", 0, true},     // blank string
		{"unknown", 0, true}, // invalid string
	}

	for _, tc := range cases {
		t.Run(tc.s, func(t *testing.T) {
			t.Parallel()
			if s, err := Parse(tc.s); (err != nil) != tc.err {
				t.Errorf("err: %v", err)
			} else if !tc.err && s != tc.System {
				t.Errorf("Expected %d; got %d", tc.System, s)
			}
		})
	}
}
