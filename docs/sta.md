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

This word determines how to interpret the next 580 bytes. For cartridges, it is as follows:

## System (4 bytes)

32 bits (Little Endian)

For cartridge snapshots, this doesn't match with the other system values we've seen. Instead, add 2 to the values found
in [list.bin](list.md#system-2-bytes). So Game Boy is `0x00000002`, Game Boy Color is `0x00000003`, etc.

For OpenFPGA snapshots this value is still set, but I have been unable to determine precisely what it means.

## Cartridge Signature (4 bytes)

32 bits (Little Endian)

The Pocket-generated cartridge signature. This appears to be what is used to associate the saves with a given cartridge.

## CRC32 (4 bytes)

32 bits (Little Endian)

Doesn't appear to be used as the CRC of a library entry can be changed without this affecting the association. Can
likely
be blank as unrecognized carts can be used to generate save states despite the Pocket not having a CRC32 associated with
it.

## Name (568 bytes)

4544 bits (Big Endian)

Zero-terminated string containing the cartridge name. This is what is used to display the name when browsing memories.
Unused space is filled with 0s.

## Unknown (4 bytes)

32 bits (Little Endian)

Appears to be the initial [snapshot size](#snapshot-size-4-bytes) added to `0x02000000`.

## Snapshot data (variable size)

The actual snapshot of the system.

## Thumbnail (variable size)

See the [developer docs](https://www.analogue.co/developer/docs/library#image-format) for full details of the format.
Image dimensions are 121 x 109.