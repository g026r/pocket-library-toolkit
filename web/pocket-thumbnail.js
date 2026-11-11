"use strict";

/**
 * Pure client-side thumbnail parser for Analogue Pocket *_thumbs.bin files.
 * Based on specifications from pocket-library-toolkit/docs/thumbs.md
 */

class PocketThumbnail {
    constructor() {
        this.ThumbnailHeader = 0x41544602;
        this.ThumbnailUnknown = 0x0000CE1C;
        this.ImageHeader32 = 0x41504920;
        this.ImageHeader16 = 0x41504910;
    }

    /**
     * Map system ID to corresponding thumbnail filename
     * - GB (0), GBC (1) -> gb_thumbs.bin
     * - GBA (2) -> gba_thumbs.bin
     * - GG (3), SMS (4) -> gg_thumbs.bin
     * - NGP (5), NGPC (6) -> ngp_thumbs.bin
     * - PCE (7) -> pce_thumbs.bin
     * - Lynx (8) -> lynx_thumbs.bin
     * @param {number} systemId 
     * @returns {string} filename
     */
    getThumbFilename(systemId) {
        switch (Number(systemId)) {
            case 0:
            case 1:
                return "gb_thumbs.bin";
            case 2:
                return "gba_thumbs.bin";
            case 3:
            case 4:
                return "gg_thumbs.bin";
            case 5:
            case 6:
                return "ngp_thumbs.bin";
            case 7:
                return "pce_thumbs.bin";
            case 8:
                return "lynx_thumbs.bin";
            default:
                return "gb_thumbs.bin";
        }
    }

    /**
     * Parse *_thumbs.bin file buffer and return a Map of crc32 -> dataUrl (or image object)
     * @param {ArrayBuffer} buffer 
     * @returns {Map<number, Object>} - Map of crc32 to thumbnail info
     */
    parseThumbsBin(buffer) {
        if (!buffer || buffer.byteLength < 12) {
            throw new Error("Invalid thumbs.bin: buffer too small");
        }

        const view = new DataView(buffer);
        
        // Validate header
        const header = view.getUint32(0, true); // Little Endian
        if (header !== this.ThumbnailHeader) {
            throw new Error(`Invalid thumbs.bin header: expected 0x${this.ThumbnailHeader.toString(16)}, got 0x${header.toString(16)}`);
        }

        const unknown = view.getUint32(4, true);
        if (unknown !== this.ThumbnailUnknown) {
            // Warn or throw? Spec says consistent across all checked files.
            console.warn(`Unrecognized thumbs.bin unknown field: 0x${unknown.toString(16)}`);
        }

        const num = view.getUint32(8, true); // Little Endian
        
        const thumbsMap = new Map();
        let offset = 12;

        // Read entry mappings (tuples of crc32 + address)
        const tuples = [];
        for (let i = 0; i < num; i++) {
            if (offset + 8 > buffer.byteLength) break;
            const crc32 = view.getUint32(offset, true);
            const address = view.getUint32(offset + 4, true);
            tuples.push({ crc32, address });
            offset += 8;
        }

        // Read each image entry
        for (const tuple of tuples) {
            if (tuple.address === 0 || tuple.address >= buffer.byteLength) continue;
            
            try {
                const imgView = new DataView(buffer, tuple.address);
                const imgHeader = imgView.getUint32(0, true);
                
                if (imgHeader !== this.ImageHeader32) {
                    // Skip or handle 16-bit / unknown image header
                    continue;
                }

                const height = imgView.getUint16(4, true);
                const width = imgView.getUint16(6, true);
                
                const pixelDataOffset = tuple.address + 8;
                const pixelDataSize = 4 * height * width;
                
                if (pixelDataOffset + pixelDataSize > buffer.byteLength) {
                    continue;
                }

                const bgraBytes = new Uint8Array(buffer, pixelDataOffset, pixelDataSize);
                
                let dataUrl = null;
                if (typeof document !== 'undefined') {
                    dataUrl = this.bgraToDataURL(bgraBytes, width, height);
                }

                thumbsMap.set(tuple.crc32, {
                    crc32: tuple.crc32,
                    width,
                    height,
                    bgraBytes,
                    dataUrl
                });
            } catch (err) {
                console.error(`Error parsing thumbnail at address 0x${tuple.address.toString(16)}:`, err);
            }
        }

        return thumbsMap;
    }

    /**
     * Convert BGRA pixel bytes to PNG Data URL using an offscreen/temporary canvas
     * @param {Uint8Array} bgraBytes 
     * @param {number} width 
     * @param {number} height 
     * @returns {string} dataUrl
     */
    bgraToDataURL(bgraBytes, width, height) {
        const canvas = document.createElement('canvas');
        // Because they're rotated 90 degrees, width = height & height = width
        canvas.width = height;
        canvas.height = width;
        const ctx = canvas.getContext('2d');
        const imgData = ctx.createImageData(height, width);  
        const data = imgData.data;

        // rotates entire image 90 degrees & changes Pocket's BGRA image data to more conventional RGBA
        const total = width * height;
        for (let p = 0; p < total; p++) {
            const row = Math.floor(p / width);                   // source row
            const col = p % width;                               // source column
            const src = p * 4;                                   // index start of BGRA pixel data in source
            const dst = (col * height + (height - 1 - row)) * 4; // index start of RGBA pixel data in destination

            data[dst]     = bgraBytes[src + 2]; // R
            data[dst + 1] = bgraBytes[src + 1]; // G
            data[dst + 2] = bgraBytes[src];     // B
            data[dst + 3] = bgraBytes[src + 3]; // A
        }

        ctx.putImageData(imgData, 0, 0);
        return canvas.toDataURL();
    }
}

// Global instance for browser use
if (typeof window !== 'undefined') {
    window.PocketThumbnail = PocketThumbnail;
}

// Export for Node.js testing
module.exports = { PocketThumbnail };
