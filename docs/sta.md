## Magic Word (4 bytes)

32 bits (Little Endian): `0x41505301`

## Snapshot Size (4 bytes)

32 bits (Little Endian)

The size of the snapshot in bytes. Includes the [unknown word](#unknown-4-bytes-1)

## Thumbnail Address (4 bytes)

32 bits (Little Endian)

Indicates the byte address in the file where the thumbnail for the save begins.

## Save Type (4 bytes)

32 bits (Little Endian)

`0x00000000` for OpenFPGA cores.

`0x00000001` for cartridge snapshots.

This word determines how to interpret the next 580 bytes.

## System (4 bytes)

32 bits (Little Endian)

This doesn't match with the other system values we've seen.

Known values for cartridges are:

* `0x00000002`: Game Boy (DMG)
* `0x00000003`: Game Boy Color
* `0x00000004`: Game Boy Advance
* `0x00000005`: Game Gear, Sega Master System
* `0x00000008`: Lynx
* `0x00000009`: PC Engine

It's my assumption that `0x00000006` & `0x00000007` are probably reserved for Neo Geo Pocket & Neo Geo Pocket Color
respectively, but until save state support arrives for that system this remains a guess.

What `0x00000000` and `0x00000001` are reserved for is unknown.

For OpenFPGA snapshots this value is still set, but I have been unable to determine precisely what it means.

## Cartridge Signature (4 bytes)

32 bits (Little Endian)

The Pocket-generated cartridge signature. This appears to be what is used to associate the saves with a given cartridge.

For OpenFPGA cores, this signature appears to be specific to the core used to generate the save but I am uncertain how
it's generated

At this point the next 572 bytes vary depending on whether the snapshot is from a cartridge or an OpenFPGA core.

## Cartridges

### CRC32 (4 bytes)

32 bits (Little Endian)

Doesn't appear to be used as the CRC of a library entry can be changed without this affecting the association. Can
likely be blank as unrecognized carts can be used to generate save states despite the Pocket not having a CRC32
associated with them.

### Name (568 bytes)

4544 bits (Big Endian)

Zero-terminated string containing the cartridge name. This is what is used to display the name when browsing memories.
Unused space is filled with 0s.

It is possible this field is limited to 44 bytes, so as to not collide with the OpenFPGA structure listed below.

## OpenFPGA Cores

### Unused (44 bytes)

352 bits, unused

These bytes are all set to `00`.

### Core author (32 bytes)

256 bits (Big Endian)

Zero-terminated string containing the value of the "author" element as defined in `core.json`

### Core shortname (32 bytes)

256 bits (Big Endian)

Zero-terminated string containing the value of the "shortname" element in `core.json`

### Core version (32 bytes)

256 bits (Big Endian)

Zero-terminated string containing the value of the "version"" element in `core.json`

### ROM name (256 bytes)

2048 bits (Big Endian)

Zero-terminated string containing the name of the ROM file. It's possible this value may be shorter than 256 bytes. If
so, it is at minimum 64 bytes. But at the moment I have no seen any additional data present.

### Platform ID (32 bytes)

256 bits (Big Endian)

Zero-terminated string containing the platform ID from `core.json`

### System Name (160 bytes)

1280 bits (Big Endian)

Zero-terminated string. My assumption is that this is taken from the platform_id.json file, but more confirmation is
necessary.

Length provided above is the maximum, though it is possible other values may be present in the 160 bytes. The minimum
length of this field is 32 bytes.

Beyond this point, file structure is once again the same for cartridge & OpenFPGA snapshots.

## Unknown (4 bytes)

32 bits (Little Endian)

Appears to be the initial [snapshot size](#snapshot-size-4-bytes) added to `0x02000000`.

## Snapshot data (variable size)

The actual snapshot of the system.

## Thumbnail (variable size)

See the [developer docs](https://www.analogue.co/developer/docs/library#image-format) for full details of the format.
Image dimensions are 121 x 109.