package util

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"

	"github.com/inancgumus/screen"
)

var ErrUnrecognizedFileFormat = errors.New("not a pocket binary file")

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

func ReadSeeker(fs fs.FS, filename string) (io.ReadSeekCloser, error) {
	fileSys, err := fs.Open(filename)
	if err != nil {
		return nil, err
	}

	fi, err := fileSys.Stat()
	if err != nil {
		return nil, err
	} else if fi.IsDir() {
		return nil, fmt.Errorf("file is a directory: %s", fi.Name())
	}

	if rs, ok := fileSys.(io.ReadSeekCloser); !ok { // fs.FS is such a half-assed interface
		return nil, fmt.Errorf("cannot cast to io.ReadSeeker: %T", fileSys)
	} else {
		return rs, nil
	}
}
