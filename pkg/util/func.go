package util

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"

	"github.com/g026r/pocket-library-toolkit/pkg/root"
)

const (
	MaxHeight int = 121
	MaxWidth  int = 109
)

// HexStringTransform takes a string, validates that it is a 32 bit hex string, and returns the uint32 representation of it
// The input string may or may not be prefixed with `0x` and any leading or trailing spaces are removed.
// If a blank string is passed, 0 is returned
func HexStringTransform(s string) (uint32, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}

	// take care of the many different ways a user might input this
	s = strings.TrimPrefix(strings.ToLower(s), "0x")
	if s == "" {
		return 0, fmt.Errorf("invalid string provided: %s", s)
	}

	// String should be exactly 32 bits. We can pad it out if too short, but can't handle too long.
	if len(s) > 8 {
		return 0, fmt.Errorf("hex string too long: %s", s)
	} else if len(s) < 8 {
		s = fmt.Sprintf("%08s", s) // binary.BigEndian.Uint32 fails if not padded out to 32 bits
	}

	h, err := hex.DecodeString(s)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint32(h), nil
}

// GetRoot finds the path to the Pocket root dir.
// If an argument was passed, it uses that.
// If an argument wasn't passed, it uses the current directory.
func GetRoot() (*root.Root, error) {
	var d string
	var err error
	switch len(os.Args) {
	case 1:
		if d, err = os.Executable(); err != nil {
			return nil, err
		}
		d = filepath.Dir(d)
	case 2:
		d = os.Args[1]
	default:
	}

	d, err = filepath.Abs(d)
	if err != nil {
		return nil, err
	}

	fi, err := os.Stat(d)
	if err != nil {
		return nil, err
	} else if !fi.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", d)
	}

	return root.OpenRoot(d)
}

func DetermineResizing(i image.Image) (int, int) {
	if float32(i.Bounds().Dx())/float32(i.Bounds().Dy()) < float32(MaxWidth)/float32(MaxHeight) {
		return MaxWidth, 0
	}
	return 0, MaxHeight
}
