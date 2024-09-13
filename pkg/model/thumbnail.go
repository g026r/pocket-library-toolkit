package model

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"os"
	"strings"

	"github.com/disintegration/imaging"

	"github.com/g026r/pocket-library-editor/pkg/util"
)

const (
	ThumbnailHeader uint32 = 0x41544602
	UnknownWord     uint32 = 0x0000CE1C
	ImageHeader32   uint32 = 0x41504920
)

const (
	maxHeight int = 121
	maxWidth  int = 109
)

type Thumbnails struct {
	Modified bool
	Images   []Image
}
type Image struct {
	address uint32 // address is only used when initially loading the file // TODO: Replace this with an address,crc32 tuple when loading?
	Crc32   uint32
	Image   []byte
}

func LoadThumbnails(dir string) (map[util.System]Thumbnails, error) {
	// Initialize our map
	m := make(map[util.System]Thumbnails)

	for _, k := range util.ValidThumbsFiles { // We're going to modify the values, so only range over the keys
		f, err := os.Open(fmt.Sprintf("%s/%s_thumbs.bin", dir, strings.ToLower(k.String())))
		if errors.Is(err, os.ErrNotExist) {
			continue // It's possible for some systems to not have thumbnails yet. Just continue
		} else if err != nil {
			return nil, err
		}
		defer f.Close() // We will close this manually as well, due to being in a loop. But this is for the early returns.

		var header uint32
		if err := binary.Read(f, binary.LittleEndian, &header); err != nil {
			return nil, err
		}
		if header != ThumbnailHeader {
			return nil, fmt.Errorf("%s: %w", f.Name(), util.ErrUnrecognizedFileFormat)
		}
		if err := binary.Read(f, binary.LittleEndian, &header); err != nil {
			return nil, err
		}
		if header != UnknownWord {
			return nil, fmt.Errorf("%s: %w", f.Name(), util.ErrUnrecognizedFileFormat)
		}

		var num uint32
		if err := binary.Read(f, binary.LittleEndian, &num); err != nil {
			return nil, err
		}

		t := Thumbnails{
			Modified: false,
			Images:   make([]Image, 0),
		}
		if num != 0 { // Only perform these steps if there are images
			// Read all of the image addresses
			for range num {
				img := Image{}
				if err := binary.Read(f, binary.LittleEndian, &img.Crc32); err != nil {
					return nil, err
				}
				if err := binary.Read(f, binary.LittleEndian, &img.address); err != nil {
					return nil, err
				}
				t.Images = append(t.Images, img)
			}

			if _, err := f.Seek(int64(t.Images[0].address), 0); err != nil {
				return nil, err
			}
			// Read each of the individual image entries.
			for i := range t.Images {
				if i+1 < len(t.Images) {
					t.Images[i].Image = make([]byte, t.Images[i+1].address-t.Images[i].address)
				} else {
					fi, _ := f.Stat()
					t.Images[i].Image = make([]byte, fi.Size()-int64(t.Images[i].address))
				}
				if n, err := f.Read(t.Images[i].Image); err != nil || n != len(t.Images[i].Image) {
					return nil, fmt.Errorf("read error: %w", err)
				}
			}
		}
		m[k] = t

		_ = f.Close()
	}

	return m, nil
}

func GenerateThumbnail(dir string, sys util.System, crc32 uint32) (Image, error) {
	sys = sys.ThumbFile() // Just in case I forgot to determine the correct system

	f, err := os.Open(fmt.Sprintf("%s/System/Library/Images/%s/%08x.bin", dir, sys.String(), crc32))
	if err != nil {
		return Image{}, err
	}
	defer f.Close()

	var header uint32
	var height, width uint16
	if err := binary.Read(f, binary.LittleEndian, &header); err != nil {
		return Image{}, err
	}
	if header != ImageHeader32 {
		return Image{}, fmt.Errorf("%s: %w", f.Name(), util.ErrUnrecognizedFileFormat)
	}

	if err := binary.Read(f, binary.LittleEndian, &height); err != nil {
		return Image{}, err
	}
	if err := binary.Read(f, binary.LittleEndian, &width); err != nil {
		return Image{}, err
	}

	img := image.NewNRGBA(image.Rectangle{
		Min: image.Point{},
		Max: image.Point{X: int(width), Y: int(height)},
	})
	bgra := make([]byte, 4)
	for i := 0; i < len(img.Pix); i = i + 4 {
		// BGRA order
		if n, err := f.Read(bgra); err != nil || n != 4 {
			return Image{}, fmt.Errorf("read error (%d): %w", n, err)
		}
		// Pix holds the image's pixels, in R, G, B, A order and big-endian format.
		img.Pix[i] = bgra[2]   // r
		img.Pix[i+1] = bgra[1] // g
		img.Pix[i+2] = bgra[0] // b
		img.Pix[i+3] = bgra[3] // a
	}

	// If the image is too square, we need to resize to the longest of the new dimensions
	// Otherwise, resize the shorter side to the new max dimensions
	newWidth, newHeight := determineResizing(img)
	img = imaging.Resize(img, newWidth, newHeight, imaging.Lanczos)
	img = imaging.CropCenter(img, maxWidth, maxHeight)

	pkt := make([]byte, 0)
	pkt, err = binary.Append(pkt, binary.LittleEndian, ImageHeader32)
	if err != nil {
		return Image{}, err
	}
	pkt, err = binary.Append(pkt, binary.LittleEndian, uint16(img.Rect.Max.Y))
	if err != nil {
		return Image{}, err
	}
	pkt, err = binary.Append(pkt, binary.LittleEndian, uint16(img.Rect.Max.X))
	if err != nil {
		return Image{}, err
	}
	// Turn it back into BGRA order
	for i := 0; i < len(img.Pix); i = i + 4 {
		pkt = append(pkt, img.Pix[i+2], img.Pix[i+1], img.Pix[i], img.Pix[i+3])
	}

	return Image{
		Crc32: crc32,
		Image: pkt,
	}, nil
}

func determineResizing(i *image.NRGBA) (int, int) {
	if float32(i.Rect.Max.X)/float32(i.Rect.Max.Y) < float32(maxWidth)/float32(maxHeight) {
		return maxWidth, 0
	}
	return 0, maxHeight
}
