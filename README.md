# pocket-library-toolkit

A rough-and-ready way to edit your Pocket's library.

## General Warning

This product is provided as-is. While the chance that it breaks your Pocket is extremely low, it is very much
experimental software that is doing something Analogue was not expecting users to do. Use at your own risk.

## How to Install

Download the appropriate version for your operating system
from [the latest release](https://github.com/g026r/pocket-library-toolkit/releases/latest) & extract the archive. I
recommend placing it in the root of your Pocket's SD card.

Run the application & select the actions you desire. A basic [user guide](docs/userguide.md) is available.

Once complete, then copy the files the tool generates over to the correct locations under your SD card's System
directory _**making backups of any originals before replacing them**_.

## But Why?

Because I can get remarkably anal about these things.

First off: 95% of Pocket users won't need or even want this.

This software is for the users who are annoyed that their library shows _Famicom Mini 01 - Super Mario Bros._ but also
_Famicom Mini 22: Nazo no Murasame Jou_, that it's _The Lion King_ but _NewZealand Story, The_.

It's for those users who have one of the small number of carts that the Pocket misidentifies & who'd rather it appeared
in their library under the correct name.

It's for the users who suddenly have a game claiming they've played more than 3,000 hours & who want to fix it without
manually editing the binary file themselves.

## Limitations

The library info screen for a given cart is stored in the Pocket's internal memory. Even if your library
now shows "_Sagaia_" instead of "_Mani 4 in 1 - Taito_", clicking into it or loading the cart will still show you the
original info.

Additionally, if you have two different entries with the same cart signature, it's likely that only the playtime for the
first will get updated.

## Troubleshooting

_"My Pocket freezes or starts crash looping when trying to access the new library"_

Something's probably gone wrong with the new library if this happens. From my experience, the solution is to power down
your Pocket (hold the power button until it turns off) and then replace the list.bin and playtimes.bin files on the SD
card with the backup copies you made.

You did make backups, right? If not, just delete the two files but you'll have to rebuild
your library from scratch after.

Simply removing the SD card won't solve this problem and powering down is the only way to restore operation.

## Technical Details

Analogue has not made the file format for list.bin, playtimes.bin, or *_thumbs.bin available in their developer docs.
The following is what I've managed to decipher & reverse engineer from my copies.

* [list.bin](./docs/list.md)
* [playtimes.bin](./docs/playtimes.md)
* [*_thumbs.bin](./docs/thumbs.md)
