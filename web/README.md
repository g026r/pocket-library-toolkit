# Pocket Library Toolkit Web Application

A lightweight client-side web application for editing Analogue Pocket library entry names &  playtimes in `list.bin` and `playtimes.bin` files.

Supports drag-and-drop of individual files or of the `System` directory.

Does not support editing of thumbnails. Maybe sometime in the future.

NB: if you load  `list.bin` and `playtimes.bin` _before_ loading the *_thumbs.bin files, the edit GUI will automatically initialize. If you want to load *_thumbs.bin files, either load them first or drag & drop the entire `System` directory.

## Running Locally

To run the web application locally simply download the web directory & double-click or open `index.html` directly in any modern web browser.

## Running Automated Tests

A Node.js test script is included to validate file parsing, sorting, and roundtrip serialization against the sample files in `testdata`:

```bash
node web/pocket-binary.test.js
```
