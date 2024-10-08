# list.bin specification

## Magic Word (4 bytes)

32 bits (Little Endian): `0x54414601`

## Number of Entries (4 bytes)

32 bits (Little Endian) integer

Must match the count found in [playtimes.bin](./playtimes.md) or odd things happen. If the value of this word is more
than 3000 (`0x00000BB8`) then the Pocket will display an empty library.

## Unknown (4 bytes)

32 bits (Little Endian): `0x00000010`.

I've been unable to determine what this does but setting it to `0x00000001` by mistake caused the system to crash loop.

## Unknown Purpose (4 bytes)

32 bits (Little Endian)

The purpose of this entry is unknown to me. On my system, this value always contains the same value as the subsequent 4
bytes. This means it duplicates the address of the first library entry.

// TODO: Set this to something else. See what happens.

## Entry Addresses (16 kiB)

4096 * 32 bits (Little Endian)

No matter how many entries you have, this section will be 16kiB in length —
from byte 0x10 to byte 0x400F (inclusive). Unused entries are normally represented as 0s but any data can be present
here as entries beyond the number specified in bytes 0x4-0x7 will be ignored.

Each entry consists of a single 32bit Little Endian entry representing the byte position in the file where the library
entry resides. As the Pocket simply displays these sequentially they should be sorted in alphabetical order by the
game's title.

As the library entry addresses appears to be a fixed size, it is likely that the first entry will always be `0x0004010`.

## Library Entry

The remainder of the file is taken up with entries for the games contained in it. Each entry follows a standard format:

### Size (2 bytes)

16 bits (Little Endian)

The size of the library entry, including these 2 bytes plus any padding necessary to have the entry terminate on a 32
bit word boundary.

### System (2 bytes)

16 bits (Big Endian)

A magic number indicating the system of the entry. Necessary for it to know not just which thumbs.bin file to check for
the thumbnail, but which mapping to check when clicking in & requesting the library info. Might in fact be treated as
Little Endian by the system, but big endian allowed me to increment the enum by one for each system so I'm saying it's
that.

System mappings are as follows:

* `0x0000`: Game Boy
* `0x0001`: Game Boy Color
* `0x0002`: Game Boy Advance
* `0x0003`: Game Gear
* `0x0004`: Sega Master System / Sega Mark III
* `0x0005`: Neo Geo Pocket
* `0x0006`: Neo Geo Pocket Color
* `0x0007`: PC Engine/TurboGrafx-16/SuperGrafx
* `0x0008`: Atari Lynx

Setting this byte to an unknown value will cause the console to crash loop until power off. (Sorry, folks, there are no
additional consoles hidden in the library code.)

### CRC32 (4 bytes)

32 bits (Little Endian)

You can find Analogue's definition of how this value is
computed [here.](https://www.analogue.co/developer/docs/library#filename-generation) But for our concerns, it's the
CRC32 of the cartridge's ROM file. This is used in conjunction with the system bytes to determine the thumbnail to load.

Unlike the signature, the Pocket never calculates this independently. Rather, it uses the signature to look up the
precalculated CRC32 in an internal database.

For most games, these are the same as the values in the [no-intro datomatic.](https://datomatic.no-intro.org/index.php).
But for Lynx games the CRCs match those of headered .lnx ROM files.

### Cartridge Signature (4 bytes)

32 bits (Little Endian)

The Analogue cartridge signature. Generated by calculating the CRC32 of the first 512 bytes of the cartridge's ROM.
Sufficient for most games but, for games where the header is not located at the start of the file, it is possible to
have multiple carts that collide. See [this file](./collisions.md) for all the signature collisions that I've been
able to determine at this time.

NB: the "first 512 bytes of the ROM" that Pocket reads for TurboGrafx-16 & Lynx games is not the same as the first 512
bytes that you might get from .pce & unheadered .lyx files. It seems likely the Pocket is adding something, possibly via
the adapter, but I've been unable to determine what.

### Magic Number (4 bytes)

32 bits (Little Endian), Though could also be 16 bits plus another 16 bits of padding.

I've been unable to determine what this word is used for. Changing its value didn't stop the library from loading
a game entry or change any of the information displayed.

It appeaars to be a simple sequential mapping of games to an integer, mostly arranged alphabetically in system
order, but not entirely.

e.g. Power Strike II on the Game Gear comes immediately after GG Aleste, likely as its Japanese name is "GG Aleste II."
e.g. The lowest Game Boy game in the sequence that I have a value for is Super R.C. Pro-Am, which is `0x00000013` & has a lower value than many games beginning with 'A'.

### Cartridge Name (variable)

Big Endian zero-terminated string.

// TODO: Unknown whether this can be a blank string.

### Padding (0-7 bytes)

Each library entry must align with a word boundary. If the size of the game's name plus zero-terminator in bits is not
evenly divisible by 32, then up to 7 bytes of padding are added. These can be any data as the system ignores it.

// TODO: Does the final entry also need this padding?
