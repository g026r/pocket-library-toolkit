package models

type Thumbnails struct {
	Modified bool
	Images   []Image
}
type Image struct {
	Crc32 uint32
	Image []byte
}
