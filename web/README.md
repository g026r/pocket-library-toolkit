# Pocket Library Toolkit Web Application

A lightweight client-side web application for editing Analogue Pocket library entry names &  playtimes in `list.bin` and `playtimes.bin` files.

## Running Locally

To run the web application locally simply download the web directory & double-click or open `index.html` directly in any modern web browser.

## Running Automated Tests

A Node.js test script is included to validate file parsing, sorting, and roundtrip serialization against the sample files in `testdata`:

```bash
node web/pocket-binary.test.js
```
