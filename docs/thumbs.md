# *_thumbs.bin file specification

## Magic Word (4 bytes)

32 bits (Little Endian): `0x41544602`

## Unknown (4 bytes)

32 bits (Little Endian): `0x0000CE1C`

Unsure what this value represents. All other Pocket files have only a 4 byte magic number at the start, but this value
is consistent across all 6 files that I've checked.

// TODO: What happens if I change this?

## Number of Entries (4 bytes)

32 bits (Little Endian) integer

Number of entries this file contains. Mappings beyond this number will be ignored by the system.

## Entry Mappings (64 kiB)

8192 * 64 bits

A mapping of the location in the file where each game's thumbnail is located. Entries beyond the number of entries are
ignored and are normally set to 0.

Unlike [list.bin](./list.md) & [playtimes.bin](./playtimes.md), which are sorted alphabetically, entries in this file
are in the order they were added to the library.

Each entry is defined as follows:

### CRC32 (4 bytes)

32 bits (Little Endian)

The CRC32 of the cartridge's ROM.

### Thumbnail Byte Address (4 bytes)

32 bits (Little Endian)

The location in the file where the thumbnail entry for this game begins. The first value will always be `0x0001000C`.

## Thumbnail Entry

New thumbnails are simply appended to the end of the file & their location then recorded in the entry mappings. Removing
a game from the library via the Pocket's UI does not remove its image from the thumbnails.

See Analogue's [developer docs](https://www.analogue.co/developer/docs/library#image-format) for details on their image
format, which this duplicates.

Two things are noteworthy about these entries:

1. The images appear to have been transformed into a 121 [`0x0079`] x 109 [`0x006D`] image, with the image being
   resized so that the short edge matches the desired dimensions and then long edge cropped equally on both sides. 
2. Thanks to these consistent dimensions, each library thumbnail requires 52764 bytes in the file.