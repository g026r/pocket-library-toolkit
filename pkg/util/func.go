package util

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/inancgumus/screen"
)

// ClearScreen clears the screen & moves the cursor back to the top left
// Used as I had some issues with gocliselect's clearing & repositioning
func ClearScreen() {
	screen.Clear()
	screen.MoveTopLeft()
}

// HexStringTransform takes a string, validates that it is a 32 bit hex string, and returns the uint32 representation of it
// The input string may or may not be prefixed with `0x` and any leading or trailing spaces are removed.
func HexStringTransform(s string) (uint32, error) {
	// take care of the many different ways a user might input this
	s = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(s)), "0x")

	// String should be exactly 32 bits. We can pad it out if too short, but can't handle too long.
	if len(s) > 8 {
		return 0, errors.New("hex string too long")
	} else if len(s) < 8 {
		s = fmt.Sprintf("%08s", s) // binary.BigEndian.Uint32 fails if not padded out to 32 bits
	}

	h, err := hex.DecodeString(s)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint32(h), nil
}

// determineThumbsFile
func DetermineThumbsFile(sys System) System {
	switch sys {
	case GBC:
		return GB
	case SMS:
		return GG
	case NGPC:
		return NGP
	default:
		return sys
	}
}
