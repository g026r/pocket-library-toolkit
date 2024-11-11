package io

import (
	"embed"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/disintegration/imaging"

	"github.com/g026r/pocket-library-toolkit/pkg/models"
	"github.com/g026r/pocket-library-toolkit/pkg/util"
)

//go:embed resources/*.json
var jsons embed.FS

const (
	ListHeader       uint32 = 0x01464154
	ListUnknown      uint32 = 0x00000010
	PlaytimesHeader  uint32 = 0x01545050
	ThumbnailHeader  uint32 = 0x41544602
	ThumbnailUnknown uint32 = 0x0000CE1C
	ImageHeader32    uint32 = 0x41504920 // The 32bit colour header. (Actually 24bit + 8 bit Alpha)
	ImageHeader16    uint32 = 0x41504910 // The 16bit colour header. Not currently used.

	firstLibraryAddr uint32 = 0x4010
	firstThumbsAddr  uint32 = 0x1000C
)

var ErrUnrecognizedFileFormat = errors.New("not a pocket binary file")
var ErrSixteenBitImage = errors.New("16-bit image files unsupported")

type Config struct {
	RemoveImages    bool `json:"remove_images"`
	AdvancedEditing bool `json:"advanced_editing"`
	ShowAdd         bool `json:"show_add"`
	GenerateNew     bool `json:"generate_new"`
	SaveUnmodified  bool `json:"save_unmodified"`
	Backup          bool `json:"backup"`
	CheckPlaytimes  bool `json:"check_playtimes"`
}

func (c Config) SaveConfig() error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	dir, err := os.Executable()
	if err != nil {
		return err
	}

	dir, f := filepath.Split(dir)
	// FIXME: Hack for when I'm testing changes. Could cause problems if someone renames the executable
	if f == "main" || f == "___main" {
		dir, err = os.Getwd()
		if err != nil {
			return err
		}
	}

	return os.WriteFile(fmt.Sprintf("%s/pocket-toolkit.json", dir), b, 0644)
}

type jsonEntry struct {
	models.System `json:"system"`
	Name          string `json:"name"`
	Crc32         string `json:"crc"`
	Sig           string `json:"signature"`
	Magic         string `json:"magic"` // TODO: Work out all possible mappings for this?
}

func (j jsonEntry) Entry() models.Entry {
	e := models.Entry{
		Name:   j.Name,
		System: j.System,
	}
	e.Sig, _ = util.HexStringTransform(j.Sig)
	e.Magic, _ = util.HexStringTransform(j.Magic)
	e.Crc32, _ = util.HexStringTransform(j.Crc32)
	return e
}

func LoadEntries(root fs.FS) ([]models.Entry, error) {
	pg, err := fs.Sub(root, "System/Played Games")
	if err != nil {
		return nil, err
	}

	f, err := ReadSeekerCloser(pg, "list.bin")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var header, num, unknown uint32
	if err = binary.Read(f, binary.BigEndian, &header); err != nil {
		return nil, err
	}
	if header != ListHeader { // Missing the magic number = not a Pocket library file
		return nil, fmt.Errorf("list.bin: %w", ErrUnrecognizedFileFormat)
	}

	if err = binary.Read(f, binary.LittleEndian, &num); err != nil {
		return nil, err
	}

	// I don't know what this word represents. It's equivalent to 0x00000010 on mine & things fail if the value changes.
	// Maybe Duo related? Keep an eye on this in future firmware releases, at least.
	if err = binary.Read(f, binary.LittleEndian, &unknown); err != nil {
		return nil, err
	}

	// This appears to be the first entry's file address? But why is it there twice?
	if err = binary.Read(f, binary.LittleEndian, &unknown); err != nil {
		return nil, err
	}

	// Parse the library entry locations.
	addresses := make([]uint32, int(num))
	if err = binary.Read(f, binary.LittleEndian, &addresses); err != nil {
		return nil, err
	}

	// Parse each of the library entries. The addresses are supposed to be sequential, but we're not going to trust that.
	entries := make([]models.Entry, int(num))
	for i := range addresses {
		e := models.Entry{}
		if _, err := f.Seek(int64(addresses[i]), io.SeekStart); err != nil {
			return nil, err
		}

		if _, err := e.ReadFrom(f); err != nil {
			return nil, err
		} else {
			entries[i] = e
		}
	}

	// Should already be sorted. But just in case.
	slices.SortFunc(entries, models.EntrySort)
	return entries, nil
}

func LoadPlaytimes(root fs.FS) ([]models.PlayTime, error) {
	pg, err := fs.Sub(root, "System/Played Games")
	if err != nil {
		return nil, err
	}

	f, err := ReadSeekerCloser(pg, "playtimes.bin")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var header uint32
	if err := binary.Read(f, binary.BigEndian, &header); err != nil {
		return nil, err
	}
	if header != PlaytimesHeader {
		return nil, fmt.Errorf("playtimes.bin: %w", ErrUnrecognizedFileFormat)
	}

	var num uint32
	if err := binary.Read(f, binary.LittleEndian, &num); err != nil {
		return nil, err
	}

	playtimes := make([]models.PlayTime, num)
	for i := range num {
		v := models.PlayTime{}

		if _, err := v.ReadFrom(f); err != nil {
			return nil, err
		}
		playtimes[i] = v
	}

	return playtimes, nil
}

func LoadThumbs(root fs.FS) (map[models.System]models.Thumbnails, error) {
	tb, err := fs.Sub(root, "System/Library/Images")
	if err != nil {
		return nil, err
	}

	thumbs := make(map[models.System]models.Thumbnails)
	for _, k := range models.ValidThumbsFiles { // We're going to modify the values, so only range over the keys
		f, err := ReadSeekerCloser(tb, fmt.Sprintf("%s_thumbs.bin", strings.ToLower(k.String())))
		if errors.Is(err, fs.ErrNotExist) {
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
			return nil, fmt.Errorf("%s_thumbs.bin: %w", strings.ToLower(k.String()), ErrUnrecognizedFileFormat)
		}
		if err := binary.Read(f, binary.LittleEndian, &header); err != nil {
			return nil, err
		}
		if header != ThumbnailUnknown {
			return nil, fmt.Errorf("%s_thumbs.bin: %w", strings.ToLower(k.String()), ErrUnrecognizedFileFormat)
		}

		var num uint32
		if err := binary.Read(f, binary.LittleEndian, &num); err != nil {
			return nil, err
		}

		type tuple struct {
			crc32   uint32
			address uint32
		}
		tuples := make([]tuple, num)
		t := models.Thumbnails{Images: make([]models.Image, num)}
		if num != 0 { // Only perform these steps if there are images
			// Read all the image addresses
			for i := range num {
				tu := tuple{}
				if err := binary.Read(f, binary.LittleEndian, &tu.crc32); err != nil {
					return nil, err
				}
				if err := binary.Read(f, binary.LittleEndian, &tu.address); err != nil {
					return nil, err
				}
				tuples[i] = tu
			}

			// Read each of the individual image entries.
			for i := range tuples {
				t.Images[i].Crc32 = tuples[i].crc32

				// Seek to the address. After the first image this *should* be where we are already. But just to be safe.
				if _, err := f.Seek(int64(tuples[i].address), io.SeekStart); err != nil {
					return nil, fmt.Errorf("seek error: %w", err)
				}
				buf := make([]byte, 8)
				if _, err := f.Read(buf); err != nil { // Read both the header & dimensions at once.
					return nil, fmt.Errorf("read error: %w", err)
				}

				//  Calculate the size of the image in bytes: byte-depth * height * width
				var size int
				dim1 := int(binary.LittleEndian.Uint16(buf[4:6]))
				dim2 := int(binary.LittleEndian.Uint16(buf[6:]))
				if binary.LittleEndian.Uint32(buf[:4]) == ImageHeader32 {
					size = 4 * dim1 * dim2
				} else {
					size = 2 * dim1 * dim2 // We can't handle 16-bit original images when generating thumbs. But if you somehow already have a 16-bit thumbnail, sure.
				}
				//  3. Read that many bytes
				img := make([]byte, size)
				if _, err := f.Read(img); err != nil {
					return nil, fmt.Errorf("read error: %w", err)
				}
				//  4. Concat the two slices together
				t.Images[i].Image = slices.Concat(buf, img)
			}
		}
		thumbs[k] = t

		_ = f.Close()
	}

	return thumbs, nil
}

func GenerateThumbnail(dir fs.FS, sys models.System, crc32 uint32) (models.Image, error) {
	sys = sys.ThumbFile() // Just in case I forgot to determine the correct system

	f, err := dir.Open(fmt.Sprintf("System/Library/Images/%s/%08x.bin", strings.ToLower(sys.String()), crc32))
	if err != nil {
		return models.Image{}, err
	}
	defer f.Close()

	var header uint32
	var height, width uint16
	if err := binary.Read(f, binary.LittleEndian, &header); err != nil {
		return models.Image{}, err
	}
	if header == ImageHeader16 {
		// No docs on 16-bit images available. Haven't been able to work out the byte order, so not going to process.
		return models.Image{}, ErrSixteenBitImage
	}
	if header != ImageHeader32 {
		return models.Image{}, fmt.Errorf("%08x.bin: %w", crc32, ErrUnrecognizedFileFormat)
	}

	if err := binary.Read(f, binary.LittleEndian, &height); err != nil {
		return models.Image{}, err
	}
	if err := binary.Read(f, binary.LittleEndian, &width); err != nil {
		return models.Image{}, err
	}

	img := image.NewNRGBA(image.Rectangle{
		Min: image.Point{},
		Max: image.Point{X: int(width), Y: int(height)},
	})
	bgra := make([]byte, 4)
	// It's stored in sequential rows, so Y on the outer loop & X on the inner
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			if _, err := f.Read(bgra); err != nil {
				return models.Image{}, fmt.Errorf("read error: %w", err)
			}
			img.SetNRGBA(x, y, color.NRGBA{B: bgra[0], G: bgra[1], R: bgra[2], A: bgra[3]})
		}
	}

	// If the image is too square, we need to resize to the longest of the new dimensions
	// Otherwise, resize the shorter side to the new max dimensions
	newWidth, newHeight := util.DetermineResizing(img)
	img = imaging.Resize(img, newWidth, newHeight, imaging.Lanczos)
	img = imaging.CropCenter(img, util.MaxWidth, util.MaxHeight)

	pkt := make([]byte, 0)
	pkt, err = binary.Append(pkt, binary.LittleEndian, header)
	if err != nil {
		return models.Image{}, err
	}
	pkt, err = binary.Append(pkt, binary.LittleEndian, uint16(img.Bounds().Dy()))
	if err != nil {
		return models.Image{}, err
	}
	pkt, err = binary.Append(pkt, binary.LittleEndian, uint16(img.Bounds().Dx()))
	if err != nil {
		return models.Image{}, err
	}

	// Turn it back into BGRA order. Don't do .RGBA() as that's alpha-pre-multiplied
	// It's stored in sequential rows, so outer loop is Y & inner is X
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			c := img.NRGBAAt(x, y)
			pkt = append(pkt, c.B, c.G, c.R, c.A)
		}
	}

	return models.Image{
		Crc32: crc32,
		Image: pkt,
	}, nil
}

func LoadConfig() (Config, error) {
	c := Config{ // Sensible defaults
		RemoveImages:    true,
		AdvancedEditing: false,
		ShowAdd:         false,
		GenerateNew:     true,
		SaveUnmodified:  false,
		Backup:          true,
		CheckPlaytimes:  true,
	}
	dir, err := os.Executable()
	if err != nil {
		return c, err
	}

	dir, f := filepath.Split(dir)
	// FIXME: Hack for when I'm testing changes. Could cause problems if someone renames the executable
	if f == "main" || f == "___main" {
		dir, err = os.Getwd()
		if err != nil {
			return c, err
		}
	}

	b, err := os.ReadFile(fmt.Sprintf("%s/pocket-toolkit.json", dir))
	if errors.Is(err, fs.ErrNotExist) {
		return c, nil // Doesn't exist. Use defaults
	} else if err != nil {
		return c, err
	}
	err = json.Unmarshal(b, &c)
	return c, err
}

func LoadInternal() (map[models.System][]models.Entry, error) {
	dir, err := jsons.ReadDir("resources")
	if err != nil {
		return nil, err
	}

	library := make(map[models.System][]models.Entry)
	for _, d := range dir {
		f, err := jsons.ReadFile(fmt.Sprintf("resources/%s", d.Name()))
		if err != nil {
			return nil, err
		}
		var x []jsonEntry
		if err := json.Unmarshal(f, &x); err != nil {
			return nil, err
		}
		sys, err := models.Parse(strings.TrimSuffix(d.Name(), ".json"))
		if err != nil {
			return nil, err
		}

		// Oh, for a native map function
		e := make([]models.Entry, len(x))
		for i := range x {
			e[i] = x[i].Entry()
		}

		slices.SortFunc(e, models.EntrySort)
		library[sys] = e
	}

	return library, nil
}

func SaveLibrary(l io.Writer, p io.Writer, e []models.Entry, tick chan any) error {
	// Prep list.bin
	if err := binary.Write(l, binary.BigEndian, ListHeader); err != nil {
		return err
	}
	if err := binary.Write(l, binary.LittleEndian, uint32(len(e))); err != nil {
		return err
	}
	if err := binary.Write(l, binary.LittleEndian, ListUnknown); err != nil { // Not sure what this value signifies, but accidentally setting it to 1 caused the system to loop
		return err
	}
	if err := binary.Write(l, binary.LittleEndian, firstLibraryAddr); err != nil { // Don't know why the first entry address appears twice
		return err
	}

	// Prep playtimes.bin
	if err := binary.Write(p, binary.BigEndian, PlaytimesHeader); err != nil {
		return err
	}
	if err := binary.Write(p, binary.LittleEndian, uint32(len(e))); err != nil {
		return err
	}

	// Build the address entries
	slices.SortFunc(e, models.EntrySort)
	addresses := make([]uint32, firstLibraryAddr/4-4)
	addresses[0] = firstLibraryAddr
	last := firstLibraryAddr
	for i := 1; i < len(e); i++ {
		addresses[i] = last + uint32(e[i-1].CalculateLength())
		last = addresses[i]
	}

	if err := binary.Write(l, binary.LittleEndian, addresses); err != nil {
		return err
	}

	for _, entry := range e {
		if _, err := entry.WriteTo(l); err != nil {
			return err
		}

		// list.bin & playtimes.bin must be recorded in the same order.
		// So write the playtimes.bin info now as well.
		if _, err := entry.Times.WriteTo(p); err != nil {
			return err
		}
		if tick != nil {
			tick <- true
		}
	}

	return nil
}

func SaveThumbsFile(t io.Writer, img []models.Image, tick chan any) error {
	if err := binary.Write(t, binary.LittleEndian, ThumbnailHeader); err != nil {
		return err
	}
	if err := binary.Write(t, binary.LittleEndian, ThumbnailUnknown); err != nil {
		return err
	}
	if err := binary.Write(t, binary.LittleEndian, uint32(len(img))); err != nil {
		return err
	}
	addr := firstThumbsAddr
	for i, j := range img {
		if err := binary.Write(t, binary.LittleEndian, j.Crc32); err != nil {
			return err
		}
		if err := binary.Write(t, binary.LittleEndian, addr); err != nil {
			return err
		}
		addr = addr + uint32(len(img[i].Image))
	}
	// write the unused addresses out as 0s
	if _, err := t.Write(make([]byte, int(firstThumbsAddr)-0xC-8*len(img))); err != nil {
		return err
	}
	// write out the images
	for _, j := range img {
		// models.Image doesn't have a WriteTo as it's just stored in memory exactly how it was read.
		if _, err := t.Write(j.Image); err != nil {
			return err
		}
		if tick != nil {
			tick <- true
		}
	}

	return nil
}

// SaveInternal saves one system's entries to a json file
// If it finds that it has more than one system, it throws an error.
// Used in magic.go to generate the files nicely.
func SaveInternal(i io.Writer, entries []models.Entry) error {
	j := make([]jsonEntry, 0)
	for i, e := range entries {
		if i != 0 && entries[i].System != entries[i-1].System {
			return fmt.Errorf("multiple systems found")
		}
		j = append(j,
			jsonEntry{
				System: e.System,
				Name:   e.Name,
				Crc32:  fmt.Sprintf("0x%08x", e.Crc32),
				Sig:    fmt.Sprintf("0x%08x", e.Sig),
				Magic:  fmt.Sprintf("0x%04x", e.Magic),
			})
	}

	b, err := json.MarshalIndent(j, "", "  ")
	if err != nil {
		return err
	}

	_, err = i.Write(b)

	return err
}

func ReadSeekerCloser(fs fs.FS, filename string) (io.ReadSeekCloser, error) {
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
		return nil, fmt.Errorf("cannot cast to io.ReadSeekerCloser: %T", fileSys)
	} else {
		return rs, nil
	}
}
