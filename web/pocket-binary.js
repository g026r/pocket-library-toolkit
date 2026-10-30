"use strict";

/**
 * Pure client-side binary parser and serializer for Analogue Pocket library files.
 * Implements PocketBinary class for parsing and serializing list.bin and playtimes.bin files.
 * 
 * Based on specifications from pocket-library-toolkit/docs/list.md and docs/playtimes.md
 * and models.EntrySort from Go implementation.
 */

class PocketBinary {
    constructor() {
        // System definitions matching Go implementation
        this.systems = {
            0: { id: 0, name: "GB", fullName: "Game Boy", thumbFile: 0 },
            1: { id: 1, name: "GBC", fullName: "Game Boy Color", thumbFile: 0 },
            2: { id: 2, name: "GBA", fullName: "Game Boy Advance", thumbFile: 2 },
            3: { id: 3, name: "GG", fullName: "Game Gear", thumbFile: 3 },
            4: { id: 4, name: "SMS", fullName: "Sega Master System", thumbFile: 3 },
            5: { id: 5, name: "NGP", fullName: "Neo Geo Pocket", thumbFile: 5 },
            6: { id: 6, name: "NGPC", fullName: "Neo Geo Pocket Color", thumbFile: 5 },
            7: { id: 7, name: "PCE", fullName: "PC Engine", thumbFile: 7 },
            8: { id: 8, name: "Lynx", fullName: "Atari Lynx", thumbFile: 8 }
        };

        // Constants from Go implementation
        this.ListHeader = 0x01464154;
        this.PlaytimesHeader = 0x01545050;
        this.ListUnknown = 0x00000010;
        this.firstLibraryAddr = 0x00004010;
        
        // Headers for playtimes.bin
        this.ThumbnailHeader = 0x41544602;
        this.ThumbnailUnknown = 0x0000CE1C;
        this.ImageHeader32 = 0x41504920;
        this.ImageHeader16 = 0x41504910;
    }

    /**
     * Parse list.bin file buffer and return library entries
     * @param {ArrayBuffer} buffer - The list.bin file content
     * @returns {Object} - Parsed entries with metadata
     * @throws {Error} - If file format is invalid
     */
    parseListBin(buffer) {
        if (!buffer || buffer.byteLength < 16) {
            throw new Error("Invalid list.bin: buffer too small");
        }

        const view = new DataView(buffer);
        
        // Validate header
        const header = view.getUint32(0, false); // Big Endian
        if (header !== this.ListHeader) {
            throw new Error(`Invalid list.bin header: expected 0x${this.ListHeader.toString(16)}, got 0x${header.toString(16)}`);
        }

        // Read entry count
        const entryCount = view.getUint32(4, true); // Little Endian
        if (entryCount === 0) {
            return { entries: [], entryCount: 0 };
        }

        // Read unknown values (positions 8-12)
        const unknown1 = view.getUint32(8, true);
        const unknown2 = view.getUint32(12, true);
        
        // Parse address table (4096 entries starting at 0x0010)
        const addressTableStart = 0x0010;
        const addressTable = new Array(entryCount);
        
        for (let i = 0; i < entryCount; i++) {
            if (addressTableStart + (i * 4) + 4 > buffer.byteLength) {
                throw new Error("Invalid list.bin: address table exceeds buffer bounds");
            }
            const address = view.getUint32(addressTableStart + (i * 4), true);
            if (address === 0) {
                throw new Error(`Invalid list.bin: zero address found at index ${i} before entryCount ${entryCount}`);
            }
            if (address + 16 > buffer.byteLength) {
                throw new Error(`Invalid list.bin: entry address 0x${address.toString(16)} exceeds buffer bounds`);
            }
            addressTable[i] = address;
        }

        // Parse library entries
        const entries = [];
        for (let i = 0; i < entryCount; i++) {
            const address = addressTable[i];
            if (address === 0) continue; // Skip unused entries
            
            const entry = this.parseLibraryEntry(view, address);
            if (entry) {
                entries.push(entry);
            }
        }

        return { entries, entryCount };
    }

    /**
     * Parse a single library entry from the given address
     * @param {DataView} view - The data view of the buffer
     * @param {number} address - Starting address of the entry
     * @returns {Object} - Parsed entry object
     */
    parseLibraryEntry(view, address) {
        try {
            const nameStart = address;
            
            // Read length (uint16 LE)
            const length = view.getUint16(nameStart, true);
            
            // Read system (uint16 BE)
            const systemBE = view.getUint16(nameStart + 2, false);
            const system = systemBE & 0xFFFF; // Ensure it's a proper uint16
            
            // Read CRC32 (uint32 LE)
            const crc32 = view.getUint32(nameStart + 4, true);
            
            // Read signature (uint32 LE)
            const sig = view.getUint32(nameStart + 8, true);
            
            // Read magic (uint32 LE)
            const magic = view.getUint32(nameStart + 12, true);
            
            // Read name string (starting after 16-byte header)
            // Safety check for view bounds before reading string
            if (nameStart + 16 > view.byteLength) {
                throw new Error("Address out of bounds");
            }
            const nameBuffer = new Uint8Array(view.buffer, nameStart + 16);
            const name = this.extractNullTerminatedString(nameBuffer);
            
            return {
                system,
                crc32,
                sig,
                magic,
                name,
                // Default playtime values (will be populated from playtimes.bin)
                added: 0,
                played: 0
            };
            
        } catch (error) {
            console.error(`Error parsing library entry at address 0x${address.toString(16)}:`, error);
            throw error; // Re-throw to signal parsing failure
        }
    }

    /**
     * Parse playtimes.bin file buffer and return playtime entries
     * @param {ArrayBuffer} buffer - The playtimes.bin file content
     * @returns {Object} - Parsed playtime entries with metadata
     * @throws {Error} - If file format is invalid
     */
    parsePlaytimesBin(buffer) {
        if (!buffer || buffer.byteLength < 8) {
            throw new Error("Invalid playtimes.bin: buffer too small");
        }

        const view = new DataView(buffer);
        
        // Validate header
        const header = view.getUint32(0, false); // Big Endian
        if (header !== this.PlaytimesHeader) {
            throw new Error(`Invalid playtimes.bin header: expected 0x${this.PlaytimesHeader.toString(16)}, got 0x${header.toString(16)}`);
        }

        // Read entry count
        const entryCount = view.getUint32(4, true); // Little Endian
        
        const requiredSize = 8 + (entryCount * 12);
        if (buffer.byteLength < requiredSize) {
            throw new Error(`Invalid playtimes.bin: buffer too small for ${entryCount} entries (expected at least ${requiredSize} bytes, got ${buffer.byteLength})`);
        }
        
        // Parse playtime records
        const playtimes = [];
        for (let i = 0; i < entryCount; i++) {
            const offset = 8 + (i * 12);
            
            // Read signature (uint32 LE)
            const sig = view.getUint32(offset, true);
            
            // Read added timestamp (uint32 LE)
            const added = view.getUint32(offset + 4, true);
            
            // Read played + system (uint32 LE)
            const playedAndSystem = view.getUint32(offset + 8, true);
            
            // Extract system and played time
            const system = this.fromPlayedTime(playedAndSystem);
            const played = playedAndSystem - this.getPlayOffset(system);
            
            playtimes.push({
                sig,
                added,
                played,
                system
            });
        }

        return { playtimes, entryCount };
    }

    /**
     * Extract null-terminated string from buffer
     * @param {Uint8Array} buffer - Byte buffer containing string
     * @param {number} maxLength - Maximum length to read
     * @returns {string} - Extracted string
     */
    extractNullTerminatedString(buffer, maxLength = 256) {
        let length = 0;
        while (length < maxLength && length < buffer.length) {
            if (buffer[length] === 0) {
                break;
            }
            length++;
        }
        
        if (length === 0) {
            return "";
        }
        
        // Convert to string and handle any potential encoding issues
        return new TextDecoder('utf-8', { fatal: false }).decode(buffer.slice(0, length));
    }

    /**
     * Convert playedAndSystem value to system ID
     * @param {number} playedAndSystem - Combined playtime and system value
     * @returns {number} - System ID (0-8)
     */
    fromPlayedTime(playedAndSystem) {
        // Extract system by checking which system offset matches
        for (let system = 8; system >= 0; system--) {
            const offset = system * 0x04000000;
            if (playedAndSystem >= offset && playedAndSystem < offset + 0x03FFFFFF) {
                return system;
            }
        }
        return 8; // Default to Lynx
    }

    /**
     * Get play offset for a system
     * @param {number} system - System ID (0-8)
     * @returns {number} - Offset value
     */
    getPlayOffset(system) {
        return system * 0x04000000;
    }

    /**
     * Format playtime in human readable format
     * @param {number} seconds - Playtime in seconds
     * @returns {string} - Formatted time string
     */
    formatPlayTime(seconds) {
        if (seconds === 0) return "0h 0m 0s";
        
        const h = Math.floor(seconds / 3600);
        const m = Math.floor((seconds % 3600) / 60);
        const s = seconds % 60;
        
        return `${h}h ${m}m ${s}s`;
    }

    /**
     * System badge color mapping
     * @param {number} system - System ID (0-8)
     * @returns {string} - CSS color class
     */
    getSystemColorClass(system) {
        const systemColors = {
            0: "gb",      // Game Boy
            1: "gbc",     // Game Boy Color
            2: "gba",     // Game Boy Advance
            3: "gg",      // Game Gear
            4: "sms",     // Sega Master System
            5: "ngp",     // Neo Geo Pocket
            6: "ngpc",    // Neo Geo Pocket Color
            7: "pce",     // PC Engine
            8: "lynx"     // Atari Lynx
        };
        return systemColors[system] || "unknown";
    }

    /**
     * Get system name
     * @param {number} system - System ID (0-8)
     * @returns {string} - System name
     */
    getSystemName(system) {
        const systemInfo = this.systems[system];
        return systemInfo ? systemInfo.name : "unknown";
    }

    /**
     * Get full system name
     * @param {number} system - System ID (0-8)
     * @returns {string} - Full system name
     */
    getSystemFullName(system) {
        const systemInfo = this.systems[system];
        return systemInfo ? systemInfo.fullName : "Unknown";
    }

    /**
     * Sort entries according to EntrySort specification
     * @param {Array} entries - Array of entry objects
     * @returns {Array} - Sorted entries
     */
    sortEntries(entries) {
        return entries.slice().sort((a, b) => {
            // Case-insensitive name comparison
            const nameA = a.name.toLowerCase();
            const nameB = b.name.toLowerCase();
            if (nameA < nameB) return -1;
            if (nameA > nameB) return 1;
            
            // System comparison
            if (a.system < b.system) return -1;
            if (a.system > b.system) return 1;
            
            // Signature comparison
            if (a.sig < b.sig) return -1;
            if (a.sig > b.sig) return 1;
            
            // CRC32 comparison
            if (a.crc32 < b.crc32) return -1;
            if (a.crc32 > b.crc32) return 1;
            
            // Magic comparison
            if (a.magic < b.magic) return -1;
            if (a.magic > b.magic) return 1;
            
            return 0;
        });
    }

    /**
     * Calculate length of a library entry
     * @param {Object} entry - Entry object with name property
     * @returns {number} - Length in bytes
     */
    calculateEntryLength(entry) {
        let length = 4; // length + system
        length += 4;   // crc32
        length += 4;   // sig
        length += 4;   // magic
        length += entry.name.length;
        length += 1;   // null terminator
        
        // Pad to 4-byte boundary
        if (length % 4 !== 0) {
            length += 4 - (length % 4);
        }
        
        return length;
    }

    /**
     * Serialize library entries to list.bin format
     * @param {Array} entries - Array of entry objects
     * @returns {ArrayBuffer} - Serialized list.bin data
     */
    serializeLibrary(entries) {
        // Sort entries first
        const sortedEntries = this.sortEntries(entries);
        
        // Calculate sizes for address table
        const addressTableSize = 0x4000; // 4096 entries * 4 bytes = 16KiB
        const headerSize = 16; // 4 uint32 values
        let currentAddress = this.firstLibraryAddr;
        
        // Calculate offsets for each entry
        const addresses = new Array(sortedEntries.length);
        addresses[0] = currentAddress;
        
        for (let i = 1; i < sortedEntries.length; i++) {
            const prevEntryLength = this.calculateEntryLength(sortedEntries[i - 1]);
            currentAddress += prevEntryLength;
            addresses[i] = currentAddress;
        }
        
        // Calculate total size needed
        let totalSize = headerSize + addressTableSize;
        for (const entry of sortedEntries) {
            totalSize += this.calculateEntryLength(entry);
        }
        
        // Create buffer
        const buffer = new ArrayBuffer(totalSize);
        const view = new DataView(buffer);
        
        // Write header
        view.setUint32(0, this.ListHeader, false); // Big Endian
        view.setUint32(4, sortedEntries.length, true); // Little Endian
        view.setUint32(8, this.ListUnknown, true);
        view.setUint32(12, this.firstLibraryAddr, true);
        
        // Write address table
        for (let i = 0; i < sortedEntries.length; i++) {
            view.setUint32(headerSize + (i * 4), addresses[i], true);
        }
        
        // Write padding to fill remaining address table entries
        for (let i = sortedEntries.length; i < 4096; i++) {
            const padOffset = headerSize + (i * 4);
            // Ensure padding doesn't overflow if the buffer is small (which shouldn't happen for valid list.bin)
            if (padOffset + 4 <= buffer.byteLength) {
                view.setUint32(padOffset, 0, true);
            }
        }
        
        // Ensure the full address table is written regardless of entry count
        const totalAddressTableSize = 4096 * 4;
        if (headerSize + totalAddressTableSize > buffer.byteLength) {
            // This case should be handled by totalSize calculation, but just in case
            console.warn("Buffer too small for full address table");
        }
        
        // Write entry data
        let currentOffset = headerSize + addressTableSize;
        for (const entry of sortedEntries) {
            const entryAddress = addresses[sortedEntries.indexOf(entry)];
            currentOffset = entryAddress;
            
            // Write entry
            this.writeEntry(view, currentOffset, entry);
            
            currentOffset += this.calculateEntryLength(entry);
        }
        
        return buffer;
    }

    /**
     * Write a single library entry to the buffer
     * @param {DataView} view - Data view to write to
     * @param {number} offset - Starting offset for entry
     * @param {Object} entry - Entry object to write
     */
    writeEntry(view, offset, entry) {
        const nameBytes = new TextEncoder().encode(entry.name);
        const length = this.calculateEntryLength(entry);
        
        // Write header
        view.setUint16(offset, length, true); // Little Endian
        view.setUint16(offset + 2, entry.system, false); // Big Endian
        view.setUint32(offset + 4, entry.crc32, true); // Little Endian
        view.setUint32(offset + 8, entry.sig, true); // Little Endian
        view.setUint32(offset + 12, entry.magic, true); // Little Endian
        
        // Write name with null terminator and padding
        const nameStart = offset + 16;
        let pos = nameStart;
        
        // Write name bytes
        for (let i = 0; i < nameBytes.length; i++) {
            view.setUint8(pos, nameBytes[i]);
            pos++;
        }
        
        // Write null terminator
        view.setUint8(pos, 0);
        pos++;
        
        // Write padding to reach word boundary
        const remaining = length - (pos - offset);
        for (let i = 0; i < remaining; i++) {
            view.setUint8(pos, 0xFF); // Use 0xFF for padding (could be any value)
            pos++;
        }
    }

    /**
     * Serialize playtimes.bin from entries
     * @param {Array} entries - Array of entry objects with playtime data
     * @returns {ArrayBuffer} - Serialized playtimes.bin data
     */
    serializePlaytimes(entries) {
        const sortedEntries = this.sortEntries(entries);
        
        // Calculate total size
        const headerSize = 8; // 2 uint32 values
        const recordSize = 12; // 3 uint32 values
        const totalSize = headerSize + (sortedEntries.length * recordSize);
        
        // Create buffer
        const buffer = new ArrayBuffer(totalSize);
        const view = new DataView(buffer);
        
        // Write header
        view.setUint32(0, this.PlaytimesHeader, false); // Big Endian
        view.setUint32(4, sortedEntries.length, true); // Little Endian
        
        // Write playtime records
        let offset = headerSize;
        for (const entry of sortedEntries) {
            // Write signature
            view.setUint32(offset, entry.sig, true);
            
            // Write added timestamp
            view.setUint32(offset + 4, entry.added, true);
            
            // Write played + system
            const playedAndSystem = entry.played + this.getPlayOffset(entry.system);
            view.setUint32(offset + 8, playedAndSystem, true);
            
            offset += recordSize;
        }
        
        return buffer;
    }

    /**
     * Validate that list.bin and playtimes.bin are synchronized
     * @param {Array} listEntries - Entries from list.bin
     * @param {Array} playtimeEntries - Entries from playtimes.bin
     * @returns {Array} - Validation results
     */
    validateSynchronization(listEntries, playtimeEntries) {
        const results = {
            valid: true,
            errors: [],
            warnings: []
        };

        if (listEntries.length !== playtimeEntries.length) {
            results.valid = false;
            results.errors.push(`Entry count mismatch: list.bin has ${listEntries.length} entries, playtimes.bin has ${playtimeEntries.length} entries`);
        }

        // Check that signatures and systems match
        for (let i = 0; i < Math.min(listEntries.length, playtimeEntries.length); i++) {
            const listEntry = listEntries[i];
            const playtimeEntry = playtimeEntries[i];
            
            if (listEntry.sig !== playtimeEntry.sig || listEntry.system !== playtimeEntry.system) {
                results.valid = false;
                results.errors.push(`Entry ${i}: Signature or system mismatch / order mismatch between list.bin and playtimes.bin`);
            }
        }

        return results;
    }
}

// Global instance for application use
if (typeof window !== 'undefined') {
    window.PocketBinary = PocketBinary;
}

// Export for Node.js testing
module.exports = { PocketBinary };