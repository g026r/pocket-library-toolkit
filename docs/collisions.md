# Games with Known Signature Collisions on the Analogue Pocket

The Analogue Pocket uses only the first 512 bytes of the game's ROM to generate its cartridge signatures. For most
officially licensed games for Nintendo & SNK handhelds, which usually have a unique header at the
start of the ROM, this is sufficient. (Unlicensed games sometimes shared the same header, so they are more likely to
have collisions.)

Sega's standard for the Game Gear & international releases of the Master System listed a number of different possible
locations for the header, none of which are at the start of the file.

Most commonly this header is found roughly 32kiB into the game ROM — likely due to the SG-1000 & Sega Mark III not
having a header & 32kiB being the maximum size of a Sega MyCard. This results in a number of signature collisions, as
the first 512 bytes is not guaranteed to be unique between games. Mostly this results in the Pocket's library
functionality being unable to detect whether a cartridge is using a revised ROM or confusing one region's release with
another's. Though in a few cases it will confuse different games built on the same engine.

Collisions listed below are grouped under the signature that the Pocket generates for them. The games are listed in
alphabetical order, which appears to be the order the Pocket stores its database of signatures in. As such, the first
entry in each grouping is the game that all cartridges with that signature will map to.

Also provided is the actual CRC32 of the cart. Useful if e.g. you have both Shining Force Gaiden & Shining Force Gaiden
II and want different thumbnails for each in your library.

### Game Boy

Though officially licensed Game Boy games generally generate unique signatures, a small number of licensed multi-carts
made use of the MMM01 mapper chip. If you want more details on it you can
read [gbdev's documentation on it.](https://gbdev.io/pandocs/MMM01.html)

For the purpose of the signature generation, the main thing to know about it is that the actual cartridge header is
located 32kiB before the end of the ROM data and the first 512 bytes consists of data for the first game on the
cartridge, including its original ROM header — yes, including the "Nintendo" logo.

* `0x66adc3af`:
    * Exodus: Journey to the Promised Land  `[0x2e5497ef]`
    * Joshua & the Battle of Jericho  `[0xe0cf879b]`
* `0xf0e6a0ab`:
    * GB Genjin  `[0x690227f6]`
    * Mani 4 in 1 - Hudson  `[0x950773ee]`
* `0x5690030f`:
    * Genki Bakuhatsu Gambaruger  `[0x6226d280]`
    * Mani 4 in 1 - Tomy  `[0xc373ac09]`
* `0xc4bdb30c`:
    * Mani 4 in 1 - Irem  `[0xcb48b6d0]`
    * R-Type II _(Japanese release only)_  `[0x6002e291]`
* `0x4b80fe56`:
    * Mani 4 in 1 - Taito  `[0x5bfc3ef5]`
    * Sagaia  `[0xe43da090]`
    * Taito Variety Pack  `[0x6dbaa5e8]`
* `0x4e27d264`:
    * Momotarou Collection 2  `[0x562c8f7f]`
    * Momotarou Dengeki 2  `[0x3c1c5eb4]`

### Game Boy Color

All known collisions occur within unlicensed cartridges.

* `0xacaad162`:
    * 31 in 1  `[0x0eb8ddfb]`
    * 31-in-1 Mighty Mix  `[0x21524051]`
* `0x4c08f8e7`:
    * Action Replay Online  `[0x04c8c858]`
    * GameShark Online  `[0xc37d21d9]`
* `0x79add746`:
    * ATV Racing & Karate Joe  `[0xa07b6e79]`
    * ATV Racing & Karate Joe _(alternate ROM)_  `[0x6908f4af]`
* `0xe3ec02fa`:
    * Rocman X Gold + 4 in 1  `[0x7e1351cf]`
    * Thunder Blast Man  `[0x1a719ead]`

### Game Boy Advance

All known collisions occur within unlicensed cartridges.

* `0xe22df58d`:
    * Action Replay GBX  `[0x5ad72359]` // TODO check which of these is the first
    * Action Replay GBX _(alternate ROM 1)_  `[0x45bb6f4e]`
    * Action Replay GBX _(alternate ROM 2)_  `[0x18ce5322]`
* `0xdbebd573`:
    * GameShark GBA  `[0xd71dbca6]` // TODO check which of these is the first
    * GameShark GBA _(alternate ROM)_  `[0x9ad94c62]`

### Game Gear

You can tell the cart signature was designed with Nintendo's consoles in mind, because the results for Sega consoles
bad. 59 games share a cart signature with another game, or 12.6% of the Game Gear's library.

* `0x72a45806`:
    * Battletoads _(Japan, Europe)_  `[0xcb3cd075]`
    * Battletoads _(USA)_  `[0x817cc0ca]`
* `0xa78c876e`:
    * Bram Stoker's Dracula _(Europe)_  `[0x69ebe5fa]`
    * Bram Stoker's Dracula _(USA)_  `[0xd966ec47]`
* `0x8655042e`:
    * Chase H.Q.  `[0xc8381def]`
    * Taito Chase H.Q.  `[0x7bb81e3d]`
* `0x091bbcce`:
    * Defenders of Oasis  `[0xe2791cc1]`
    * Shadam Crusader: Harukanaru Oukoku  `[0x09f9ed60]`
* `0xf5637371`:
    * Desert Strike  `[0x3e44eca3]`
    * Desert Strike: Return to the Gulf  `[0xf6c400da]`
* `0x4d75f730`:
    * Earthworm Jim _(Europe)_  `[0x691ae339]`
    * Earthworm Jim _(USA)_  `[0x5d3f23a9]`
* `0x7308c2b6`:
    * Ecco the Dolphin II  `[0xba9cef4f]`
    * Ecco: The Tides of Time  `[0xe2f3b203]`
* `0xd2ed7516`:
    * Fatal Fury Special _(Europe)_  `[0xfbd76387]`
    * Fatal Fury Special _(USA)_  `[0x449787e2]`
    * Garou Densetsu Special  `[0x9afb6f33]`
* `0x41117e28`:
    * In the Wake of Vampire  `[0xdab0f265]`
    * Master of Darkness  `[0x07d0eb42]`
    * Vampire: Master of Darkness  `[0x7ec64025]`
* `0xff688096`:
    * Kick & Rush  `[0xfd14ce00]`
    * Tengen World Cup Soccer  `[0xdd6d2e34]`
* `0x63fcc174`:
    * Legend of Illusion Starring Mickey Mouse  `[0xce5ad8b7]`
    * Mickey Mouse Densetsu no Oukoku: Legend of Illusion  `[0xfe12a92f]`
* `0x5482c3e4`:
    * Madou Monogatari III: Kyuukyoku Joou-sama _(Rev 1)_  `[0x568f4825]`
    * Madou Monogatari III: Kyuukyoku Joou-sama  `[0x0a634d79]`
* `0x53a2ec3a`:
    * Nazo Puyo _(Rev 1)_  `[0xd8d11f8d]`
    * Nazo Puyo  `[0xbcce5fd4]`
* `0x29964f70`:
    * NBA Jam _(Japan)_  `[0xa49e9033]`
    * NBA Jam _(USA) (Rev 1)_  `[0x820fa4ab]`
* `0x70b95384`:
    * OutRun Europa _(Europe)_  `[0x01eab89d]`
    * OutRun Europa _(USA)_  `[0xf037ec00]`
* `0xc6018199`:
    * Riddick Bowe Boxing _(Japan)_  `[0xa45fffb7]`
    * Riddick Bowe Boxing _(USA)_  `[0x38d8ec56]`
* `0x318bcf10`:
    * Road Rash _(Europe)_  `[0x176505d4]`
    * Road Rash _(USA)_  `[0x96045f76]`
* `0xe2f9f8f2`:
    * Shining Force Gaiden II: Jashin no Kakusei  `[0x30374681]`
    * Shining Force Gaiden: Final Conflict (Japan)  `[0x6019fe5e]`
    * Shining Force II: The Sword of Hajya (USA)  `[0xa6ca6fa9]`
* `0x98ec775a`:
    * Star Wars _(USA)_  `[0xdb9bc599]`
    * Star Wars _(Europe)_  `[0x0228769c]`
* `0x9f8c4abb`:
    * Super Columns _(Japan)_  `[0x2a100717]`
    * Super Columns _(Europe, USA)_  `[0x8ba43af3]`
* `0x09e7a464`:
    * Super Monaco GP _(Japan, Korea)_  `[0x4f686c4a]`
    * Super Monaco GP _(Brazil, Europe, USA)_  `[0xfcf12547]`
* `0x1ecc7ca8`:
    * Shinobi  `[0x30f1c984]`
    * The GG Shinobi  `[0x83926bd1]`
* `0xa0af1447`:
    * The Jungle Book _(Europe)_  `[0x90100884]`
    * The Jungle Book _(USA)_  `[0x30c09f31]`
* `0x9062b5ec`:
    * The Lion King _(Europe)_  `[0x0cd9c20b]`
    * The Lion King _(USA)_  `[0x9808d7b3]`
* `0xb9dd2fd9`:
    * Tom and Jerry: The Movie _(Brazil, Japan)_  `[0xa1453efa]`
    * Tom and Jerry: The Movie _(Europe, USA)_  `[0x5cd33ff2]`
* `0xaab18e1f`:
    * Virtua Fighter Animation  `[0xd431c452]`
    * Virtua Fighter Mini  `[0xc05657f8]`
* `0xb8d858bd`:
    * X-Terminator _(Europe)_  `[0x0f448220]`
    * X-Terminator _(Japan)_  `[0xe498090d]`
* `0x159af0a7`:
    * Zool: Ninja of the 'Nth' Dimension _(USA)_  `[0xb287c695]`
    * Zool no Yume Bouken  `[0xe35ef7ed]`

### Sega Master System

The Master System/Mark III has a lot more unlicensed Taiwanese & Korean games in the Pocket's internal library. Without
them the total collisions would probably be lower than the Game Gear's. But as it stands, 81 games have signature
collisions, amounting to 15.7% of the system's library.

* `0xa5ba870c`:
    * 3 in 1 - The Best Game Collection (A) `[0x98af0236]`
    * 3 in 1 - The Best Game Collection (B) `[0x6ebfe1c3]`
    * 3 in 1 - The Best Game Collection (C) `[0x81a36a4f]`
    * 3 in 1 - The Best Game Collection (D) `[0x8d2d695d]`
    * 3 in 1 - The Best Game Collection (E) `[0x82c09b57]`
    * 3 in 1 - The Best Game Collection (F) `[0x4088eeb4]`
* `0x92a4aec2`:
    * 8 in 1 - The Best Game Collection (A) `[0xfba94148]`
    * 8 in 1 - The Best Game Collection (B) `[0x8333c86e]`
    * 8 in 1 - The Best Game Collection (C) `[0x00e9809f]`
* `0x94c277d4`:
    * Action Fighter _(Japan, Europe)_ `[0xd91b340d]`
    * Action Fighter _(Taiwan)_ `[0x8418f438]`
    * Action Fighter _(USA, Europe, Brazil) (Rev 1)_ `[0x3658f3e0]`
* `0x35c2093a`:
    * Alex Kidd in Miracle World _(Taiwan)_ `[0x6f8e46cf]` // TODO: Is this the "World" entry? Is this "Alex Kido"?
    * Alex Kidd no Miracle World _(Japan)_ `[0x08c9ec91]`
* `0xb9408fad`:
    * Alex Kidd in Miracle World _(USA, Europe, Brazil) (Rev 1)_ `[0xaed9aac4]`
    * Alex Kidd in Miracle World 2 _(World)_ `[0x7de172ff]` // TODO: Is this the "World" entry? Is this "Alex Kido"?
* `0x915dec44`:
    * Alibaba and 40 Thieves `[0x08bf3de3]`
    * Galaxian `[0x577ec227]`
* `0x449df5b3`:
    * Asterix _(Europe, Brazil) (Rev 1)_ `[0x8c9d5be8]`
    * Asterix _(Europe, Brazil)_ `[0x147e02fa]`
* `0xf9205dc8`:
    * Bubble Bobble _(Europe, Brazil)_ `[0xe843ba7e]`
    * Final Bubble Bobble `[0x3ebb7457]`
* `0x6e461592`:
    * C_So! `[0x0918fba0]`
    * Xyzolog `[0x565c799f]`
* `0x59155da9`:
    * Comical Machine Gun Joe _(Japan)_ `[0x9d549e08]`
    * Comical Machine Gun Joe _(Korea)_ `[0x643f6bfc]`
    * Comical Machine Gun Joe _(Taiwan)_ `[0x84ad5ae4]`
* `0x29a56fe5`:
    * E.I. - Exa Innova `[0xdd74bcf1]`
    * Mopiranger `[0xb49aa6fc]`
    * Ppang Gongjang - Cosmic Bakery `[0x7778e256]`
    * Road Fighter `[0x8034bd27]`
    * Sky Jaguar _(Clover version)_ `[0xe3f260ca]`
    * Sky Jaguar _(Samsung version)_ `[0x5b8e65e4]`
* `0x8dd42a53`:
    * F-16 Fighting Falcon _(Japan)_ `[0x7ce06fce]`
    * F-16 Fighting Falcon _(Taiwan)_ `[0xc4c53226]`
* `0x3a47062:`
    * F-16 Fighter `[0xeaebf323]`
    * F-16 Fighting Falcon _(USA)_ `[0x184c23b7]`
* `0xf88d5e98`:
    * Family Games `[0x7abc70e9]`
    * Parlour Games `[0xe030e66c]`
* `0x49aabe28`:
    * Fantasy Zone _(Taiwan)_ `[0x5fd48352]`
    * Fantasy Zone _(World)_ `[0x65d7e4e0]`
* `0x36377097`:
    * Ghost House `[0xc0f3ce7e]`
    * Yuyryeong-ui Jip - Ghost House `[0x1203afc9]`
* `0xf8f9c3a7`:
    * Great Golf _(Japan)_ `[0x6586bd1f]`
    * Great Golf _(Korea)_ `[0x5def1bf5]`
* `0xda9bc0c0`:
    * Great Soccer _(Japan)_ `[0x2d7fd7ef]`
    * Great Soccer _(Taiwan)_ `[0x84665648]`
* `0x8cf2a757`:
    * Hokuto no Ken _(Japan)_ `[0x24f5fe8c]`
    * Hokuto no Ken _(Taiwan)_ `[0xc4ab363d]`
* `0xd9f2a9b6`:
    * Monica no Castelo do Dragao `[0x01d67c0b]`
    * Wonder Boy in Monster Land `[0x8cbef0c1]`
* `0x9859722b`:
    * Phantasy Star _(Brazil)_ `[0x75971bef]`
    * Phantasy Star _(USA, Europe) (Rev 1)_ `[0x00bef1d7]`
    * Phantasy Star _(USA, Europe)_ `[0xe4a65e79]`
* `0x3f9b5473`:
    * Psycho Fox `[0x97993479]`
    * Sapo Xule vs. Os Invasores do Brejo `[0x9a608327]`
* `0x62537afa`:
    * Rainbow Islands: Story of Bubble Bobble 2 _(Europe)_ `[0xc172a22c]`
    * Rainbow Islands: The Story of Bubble Bobble 2 _(Brazil)_ `[0x00ec173a]`
* `0xd7b8dbd4`:
    * Seishun Scandal `[0xf0ba2bc6]`
    * Ttoriui Moheom `[0x178801d2]`
* `0x515c54f5`:
    * Shinobi _(Japan, Brazil)_ `[0xe1fff1bb]`
    * Shinobi _(USA, Europe, Brazil) (Rev 1)_ `[0x0c6fac4e]`
* `0x34d8f25f`:
    * Slap Shot (Europe) (Rev 1) `[0xd33b296a]`
    * Slap Shot (Europe) `[0xc93bd0e9]`
* `0xa4bd66df`:
    * Space Harrier 3-D _(USA, Europe, Brazil)_ `[0x6bd5c2bf]`
    * Space Harrier 3D _(Japan)_ `[0x156948f9]`
* `0x2ae0acbb`:
    * Speedball _(Europe) (Rev 1)_ `[0x5ccc1a65]`
    * Speedball _(Europe)_ `[0xa57cad18]`
* `0xca555d47`:
    * Spy vs Spy _(Japan)_ `[0xd41b9a08]`
    * Spy vs Spy _(Taiwan)_ `[0x689f58a2]`
* `0x9b5db90b`:
    * Super Arkanoid `[0xc9dd4e5f]`
    * Woody Pop: Shinjinrui no Block Kuzushi `[0x315917d4]`
* `0x81c8bc58`:
    * Turma da Monica em O Resgate `[0x22cca9bb]`
    * Wonder Boy III: The Dragon's Trap _(GOG release)_ `[0x525f4f3d]`
    * Wonder Boy III: The Dragon's Trap _(USA, Europe)_ `[0x679e1676]`
* `0x54352261`:
    * Where in the World is Carmen Sandiego _(Brazil)_ `[0x88aa8ca6]`
    * Where in the World is Carmen Sandiego _(USA)_ `[0x428b1e7c]`
* `0x7ee6cd81`:
    * Xenon 2: Megablast  _(Rev 1)_ `[0xec726c0d]`
    * Xenon 2: Megablast `[0x5c205ee1]`
* `0xa7e33a29`:
    * Zillion _(Europe, Brazil) (Rev 2)_ `[0x7ba54510]`
    * Zillion _(USA, Europe) (Rev 1)_ `[0x5718762c]`

### Neo Geo Pocket / Neo Geo Pocket Color

No known collisions.

### PC Engine / TurboGrafx-16 / SuperGrafx

// TODO: Check

### Atari Lynx

// TODO: Check