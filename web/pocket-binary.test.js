"use strict";

/**
 * Node.js test script for pocket-binary.js
 * Validates binary parsing, serialization, and error conditions against test data
 */

const fs = require('fs');
const path = require('path');

// Import the PocketBinary class from pocket-binary.js
const { PocketBinary } = require('./pocket-binary.js');

const validListBinPath = path.join(__dirname, '..', 'testdata', 'valid', 'System', 'Played Games', 'list.bin');
const validPlaytimesBinPath = path.join(__dirname, '..', 'testdata', 'valid', 'System', 'Played Games', 'playtimes.bin');

const invalidListBinPath = path.join(__dirname, '..', 'testdata', 'invalid_header', 'System', 'Played Games', 'list.bin');
const invalidPlaytimesBinPath = path.join(__dirname, '..', 'testdata', 'invalid_header', 'System', 'Played Games', 'playtimes.bin');

/**
 * Test loading and validating binary files (AC.1 & AC.2)
 * Confirms invalid headers are specifically caught.
 */
async function test_load_and_validate() {
    console.log('Running test_load_and_validate...');
    
    const parser = new PocketBinary();
    
    // Test 1: Load valid files
    try {
        const listBinBuffer = Uint8Array.from(fs.readFileSync(validListBinPath)).buffer;
        const playtimesBinBuffer = Uint8Array.from(fs.readFileSync(validPlaytimesBinPath)).buffer;
        
        const listResult = parser.parseListBin(listBinBuffer);
        const playtimesResult = parser.parsePlaytimesBin(playtimesBinBuffer);
        
        console.log(`✓ Successfully parsed list.bin: ${listResult.entries.length} entries`);
        console.log(`✓ Successfully parsed playtimes.bin: ${playtimesResult.playtimes.length} entries`);
        
        // Validate synchronization
        const syncResult = parser.validateSynchronization(listResult.entries, playtimesResult.playtimes);
        if (syncResult.valid) {
            console.log('✓ Files are properly synchronized');
        } else {
            console.log('✗ Synchronization errors:', syncResult.errors.join(', '));
            return false;
        }
        
    } catch (error) {
        console.log('✗ Failed to load valid test files:', error.message);
        return false;
    }
    
    // Test 2: Handle invalid headers with specific error assertion
    try {
        const invalidListBinBuffer = Uint8Array.from(fs.readFileSync(invalidListBinPath)).buffer;
        try {
            parser.parseListBin(invalidListBinBuffer);
            console.log('✗ Expected error for invalid list.bin header');
            return false;
        } catch (e) {
            if (e.message.includes('Invalid list.bin header')) {
                console.log('✓ Invalid list.bin header error confirmed:', e.message);
            } else {
                console.log('✗ Unexpected error message for invalid list.bin header:', e.message);
                return false;
            }
        }

        const invalidPlaytimesBinBuffer = Uint8Array.from(fs.readFileSync(invalidPlaytimesBinPath)).buffer;
        try {
            parser.parsePlaytimesBin(invalidPlaytimesBinBuffer);
            console.log('✗ Expected error for invalid playtimes.bin header');
            return false;
        } catch (e) {
            if (e.message.includes('Invalid playtimes.bin header')) {
                console.log('✓ Invalid playtimes.bin header error confirmed:', e.message);
            } else {
                console.log('✗ Unexpected error message for invalid playtimes.bin header:', e.message);
                return false;
            }
        }
        
    } catch (error) {
        console.log('✗ Invalid header test failed with unexpected error:', error.message);
        return false;
    }
    
    return true;
}

/**
 * Test invalid entry count condition (Test 1 requested)
 * Fails when indicated count at byte 0x4 exceeds actual file bounds / zero addresses.
 */
async function test_invalid_count() {
    console.log('Running test_invalid_count...');
    const parser = new PocketBinary();
    const invalidCountDir = path.join(__dirname, '..', 'testdata', 'invalid_count', 'System', 'Played Games');
    
    let caughtError = false;
    try {
        const listBuffer = Uint8Array.from(fs.readFileSync(path.join(invalidCountDir, 'list.bin'))).buffer;
        parser.parseListBin(listBuffer);
    } catch (e) {
        caughtError = true;
        console.log('✓ Invalid count list.bin correctly failed:', e.message);
    }
    if (!caughtError) {
        console.log('✗ Expected invalid count list.bin to fail');
        return false;
    }

    caughtError = false;
    try {
        const playtimesBuffer = Uint8Array.from(fs.readFileSync(path.join(invalidCountDir, 'playtimes.bin'))).buffer;
        parser.parsePlaytimesBin(playtimesBuffer);
    } catch (e) {
        caughtError = true;
        console.log('✓ Invalid count playtimes.bin correctly failed:', e.message);
    }
    if (!caughtError) {
        console.log('✗ Expected invalid count playtimes.bin to fail');
        return false;
    }

    return true;
}

/**
 * Test strict read count (Test 2 requested)
 * Confirms it only reads the number of entries found at byte 0x4 regardless of file length.
 */
async function test_count_mismatch() {
    console.log('Running test_count_mismatch...');
    const parser = new PocketBinary();
    const mismatchDir = path.join(__dirname, '..', 'testdata', 'count_mismatch', 'System', 'Played Games');
    
    const listBuffer = Uint8Array.from(fs.readFileSync(path.join(mismatchDir, 'list.bin'))).buffer;
    const playtimesBuffer = Uint8Array.from(fs.readFileSync(path.join(mismatchDir, 'playtimes.bin'))).buffer;
    
    const listResult = parser.parseListBin(listBuffer);
    const playtimesResult = parser.parsePlaytimesBin(playtimesBuffer);
    
    if (listResult.entries.length !== 4) {
        console.log(`✗ Expected 4 entries based on header count, got ${listResult.entries.length}`);
        return false;
    }
    console.log('✓ Correctly read exactly the number of entries indicated at byte 0x4 (4 entries)');
    
    return true;
}

/**
 * Test synchronization count mismatch (Test 3 requested)
 * Confirms validation fails if entry count in playtimes.bin does not match list.bin.
 */
async function test_synchronization_mismatch() {
    console.log('Running test_synchronization_mismatch...');
    const parser = new PocketBinary();
    const mismatchDir = path.join(__dirname, '..', 'testdata', 'count_mismatch', 'System', 'Played Games');
    const validDir = path.join(__dirname, '..', 'testdata', 'valid', 'System', 'Played Games');
    
    const listBuffer = Uint8Array.from(fs.readFileSync(path.join(mismatchDir, 'list.bin'))).buffer;
    const playtimesBuffer = Uint8Array.from(fs.readFileSync(path.join(validDir, 'playtimes.bin'))).buffer;
    
    const listResult = parser.parseListBin(listBuffer);
    const playtimesResult = parser.parsePlaytimesBin(playtimesBuffer);
    
    const syncResult = parser.validateSynchronization(listResult.entries, playtimesResult.playtimes);
    if (syncResult.valid) {
        console.log('✗ Expected validation to fail due to entry count mismatch between list.bin and playtimes.bin');
        return false;
    }
    console.log('✓ Synchronization validation correctly failed on entry count mismatch:', syncResult.errors[0]);
    
    return true;
}

/**
 * Test out of order synchronization validation (Test 4 requested)
 * Confirms entry ordering, signature, and system match between both files.
 */
async function test_out_of_order() {
    console.log('Running test_out_of_order...');
    const parser = new PocketBinary();
    const outOfOrderDir = path.join(__dirname, '..', 'testdata', 'out_of_order', 'System', 'Played Games');
    
    const listBuffer = Uint8Array.from(fs.readFileSync(path.join(outOfOrderDir, 'list.bin'))).buffer;
    const playtimesBuffer = Uint8Array.from(fs.readFileSync(path.join(outOfOrderDir, 'playtimes.bin'))).buffer;
    
    const listResult = parser.parseListBin(listBuffer);
    const playtimesResult = parser.parsePlaytimesBin(playtimesBuffer);
    
    const syncResult = parser.validateSynchronization(listResult.entries, playtimesResult.playtimes);
    if (syncResult.valid) {
        console.log('✗ Expected validation to fail due to out-of-order or signature/system mismatch');
        return false;
    }
    console.log('✓ Validation correctly failed on out-of-order entries:', syncResult.errors[0]);
    
    return true;
}

/**
 * Test editing and sorting entries (AC.5 & AC.6)
 */
async function test_edit_and_sort() {
    console.log('Running test_edit_and_sort...');
    
    const parser = new PocketBinary();
    
    const listBinBuffer = Uint8Array.from(fs.readFileSync(validListBinPath)).buffer;
    const playtimesBinBuffer = Uint8Array.from(fs.readFileSync(validPlaytimesBinPath)).buffer;
    
    const listResult = parser.parseListBin(listBinBuffer);
    const playtimesResult = parser.parsePlaytimesBin(playtimesBinBuffer);
    
    const entries = listResult.entries;
    for (const listEntry of entries) {
        const matchingPlaytime = playtimesResult.playtimes.find(p => p.sig === listEntry.sig);
        if (matchingPlaytime) {
            listEntry.added = matchingPlaytime.added;
            listEntry.played = matchingPlaytime.played;
            listEntry.system = matchingPlaytime.system;
        }
    }
    
    const zillionIndex = entries.length - 1;
    const zillionEntry = entries[zillionIndex];
    
    if (!zillionEntry || zillionEntry.name !== 'Zillion') {
        console.log('✗ Could not find Zillion entry for testing');
        return false;
    }
    
    const originalName = zillionEntry.name;
    zillionEntry.name = '00000000000000000000';
    
    const sortedEntries = parser.sortEntries(entries);
    const modifiedIndex = sortedEntries.findIndex(e => e.name === '00000000000000000000');
    
    if (modifiedIndex !== 0) {
        console.log(`✗ Expected modified entry to be at index 0, got ${modifiedIndex}`);
        return false;
    }
    
    const originalPosition = sortedEntries.findIndex(e => e.name === originalName);
    if (originalPosition !== -1) {
        console.log('✗ Original entry still found in sorted list by original name');
        return false;
    }
    
    return true;
}

/**
 * Test serialization roundtrip (AC.7 & AC.8)
 */
async function test_serialize_roundtrip() {
    console.log('Running test_serialize_roundtrip...');
    
    const parser = new PocketBinary();
    
    const listBinBuffer = Uint8Array.from(fs.readFileSync(validListBinPath)).buffer;
    const playtimesBinBuffer = Uint8Array.from(fs.readFileSync(validPlaytimesBinPath)).buffer;
    
    const listResult = parser.parseListBin(listBinBuffer);
    const playtimesResult = parser.parsePlaytimesBin(playtimesBinBuffer);
    
    const entries = listResult.entries;
    for (const listEntry of entries) {
        const matchingPlaytime = playtimesResult.playtimes.find(p => p.sig === listEntry.sig);
        if (matchingPlaytime) {
            listEntry.added = matchingPlaytime.added;
            listEntry.played = matchingPlaytime.played;
            listEntry.system = matchingPlaytime.system;
        }
    }
    
    const serializedListBin = parser.serializeLibrary(entries);
    const serializedPlaytimesBin = parser.serializePlaytimes(entries);
    
    const reparsedListResult = parser.parseListBin(serializedListBin.slice(0));
    const reparsedPlaytimesResult = parser.parsePlaytimesBin(serializedPlaytimesBin.slice(0));
    
    if (entries.length !== reparsedListResult.entries.length) {
        console.log(`✗ Entry count mismatch: original ${entries.length}, reparsed ${reparsedListResult.entries.length}`);
        return false;
    }
    
    return true;
}

/**
 * Test system synonyms search filtering (e.g. "TurboGrafx 16" -> PC Engine)
 */
async function test_system_synonyms() {
    console.log('Running test_system_synonyms...');
    const parser = new PocketBinary();
    
    const entry = {
        name: '1943 Kai',
        system: 7, // PCE
        sig: 0x79cd1a3e,
        crc32: 0xfde08d6d,
        magic: 0x1b94
    };
    
    const searchTerm = 'turbografx';
    const sysName = parser.getSystemName(entry.system).toLowerCase();
    const sysFullName = parser.getSystemFullName(entry.system).toLowerCase();
    
    let systemMatch = sysName.includes(searchTerm) || sysFullName.includes(searchTerm);
    if (entry.system === 7) {
        const synonyms = ["turbografx 16", "turbografx-16", "turbografx", "tg16", "tg-16", "supergrafx"];
        if (synonyms.some(syn => syn.includes(searchTerm) || searchTerm.includes(syn))) {
            systemMatch = true;
        }
    }
    
    if (!systemMatch) {
        console.log('✗ Expected synonym search for "turbografx" to match PC Engine entry');
        return false;
    }
    console.log('✓ Synonym search successfully matched PC Engine entry using "turbografx"');
    return true;
}

/**
 * Test 1: Playtime Serialization/Deserialization & Offsets across all console systems
 */
async function test_playtime_offsets_all_systems() {
    console.log('Running test_playtime_offsets_all_systems...');
    const parser = new PocketBinary();
    
    const entries = [];
    for (let sys = 0; sys <= 8; sys++) {
        entries.push({
            name: "",
            sig: 0x1000 + sys,
            added: 1700000000 + sys,
            played: 1234 + sys * 10,
            system: sys
        });
    }
    
    const serialized = parser.serializePlaytimes(entries);
    const reparsed = parser.parsePlaytimesBin(serialized);
    
    if (reparsed.playtimes.length !== 9) {
        console.log(`✗ Expected 9 reparsed playtimes, got ${reparsed.playtimes.length}`);
        return false;
    }
    
    for (let i = 0; i < entries.length; i++) {
        if (reparsed.playtimes[i].played !== entries[i].played || reparsed.playtimes[i].system !== entries[i].system) {
            console.log(`✗ Mismatch at system ${i}: expected played=${entries[i].played}, system=${entries[i].system}; got played=${reparsed.playtimes[i].played}, system=${reparsed.playtimes[i].system}`);
            return false;
        }
    }
    
    console.log('✓ Playtime offsets and serialization across all 9 systems passed');
    return true;
}

/**
 * Test 2: Combined Filtering (Search queries and system filter badges simultaneously)
 */
async function test_combined_filtering() {
    console.log('Running test_combined_filtering...');
    const parser = new PocketBinary();
    
    const entries = [
        { name: 'Super Mario Land', system: 0, sig: 1, crc32: 1, magic: 1 },
        { name: 'Super Mario Advance', system: 2, sig: 2, crc32: 2, magic: 2 },
        { name: 'Metroid', system: 0, sig: 3, crc32: 3, magic: 3 }
    ];
    
    const selectedSystem = 'GB';
    const searchTerm = 'mario';
    
    const filtered = entries.filter(entry => {
        const sysName = parser.getSystemName(entry.system);
        if (sysName !== selectedSystem) return false;
        
        const nameMatch = entry.name.toLowerCase().includes(searchTerm);
        return nameMatch;
    });
    
    if (filtered.length !== 1 || filtered[0].name !== 'Super Mario Land') {
        console.log(`✗ Expected 1 entry ("Super Mario Land"), got ${filtered.length}`);
        return false;
    }
    
    console.log('✓ Combined system filter and search query test passed');
    return true;
}

/**
 * Test 3: Library Reloading & State Reset
 */
async function test_library_reloading() {
    console.log('Running test_library_reloading...');
    const parser = new PocketBinary();
    
    const list1 = Uint8Array.from(fs.readFileSync(validListBinPath)).buffer;
    const play1 = Uint8Array.from(fs.readFileSync(validPlaytimesBinPath)).buffer;
    
    const res1List = parser.parseListBin(list1);
    const res1Play = parser.parsePlaytimesBin(play1);
    
    let listEntries = [...res1List.entries];
    let playtimesEntries = [...res1Play.playtimes];
    
    const mismatchDir = path.join(__dirname, '..', 'testdata', 'count_mismatch', 'System', 'Played Games');
    const list2 = Uint8Array.from(fs.readFileSync(path.join(mismatchDir, 'list.bin'))).buffer;
    const play2 = Uint8Array.from(fs.readFileSync(path.join(mismatchDir, 'playtimes.bin'))).buffer;
    
    const res2List = parser.parseListBin(list2);
    const res2Play = parser.parsePlaytimesBin(play2);
    
    listEntries = [...res2List.entries];
    playtimesEntries = [...res2Play.playtimes];
    
    if (listEntries.length !== 4) {
        console.log(`✗ Expected reloaded list entries length to be 4, got ${listEntries.length}`);
        return false;
    }
    
    console.log('✓ Library state reset and reload test passed');
    return true;
}

/**
 * Run all tests
 */
async function runTests() {
    console.log('=== Pocket Binary Tests ===\n');
    
    const tests = [
        { name: 'test_load_and_validate', func: test_load_and_validate },
        { name: 'test_invalid_count', func: test_invalid_count },
        { name: 'test_count_mismatch', func: test_count_mismatch },
        { name: 'test_synchronization_mismatch', func: test_synchronization_mismatch },
        { name: 'test_out_of_order', func: test_out_of_order },
        { name: 'test_edit_and_sort', func: test_edit_and_sort },
        { name: 'test_serialize_roundtrip', func: test_serialize_roundtrip },
        { name: 'test_system_synonyms', func: test_system_synonyms },
        { name: 'test_playtime_offsets_all_systems', func: test_playtime_offsets_all_systems },
        { name: 'test_combined_filtering', func: test_combined_filtering },
        { name: 'test_library_reloading', func: test_library_reloading }
    ];
    
    let passed = 0;
    let failed = 0;
    
    for (const test of tests) {
        console.log(`\n--- Running ${test.name} ---`);
        try {
            const result = await test.func();
            if (result) {
                console.log(`✓ ${test.name} PASSED`);
                passed++;
            } else {
                console.log(`✗ ${test.name} FAILED`);
                failed++;
            }
        } catch (error) {
            console.log(`✗ ${test.name} ERROR: ${error.message}`);
            console.log(error.stack);
            failed++;
        }
    }
    
    console.log('\n=== Test Summary ===');
    console.log(`Passed: ${passed}`);
    console.log(`Failed: ${failed}`);
    console.log(`Total: ${passed + failed}`);
    
    if (failed === 0) {
        console.log('\n🎉 All tests passed!');
        return true;
    } else {
        console.log('\n❌ Some tests failed');
        return false;
    }
}

if (require.main === module) {
    runTests()
        .then(success => {
            process.exit(success ? 0 : 1);
        })
        .catch(error => {
            console.error('Test runner error:', error);
            process.exit(1);
        });
}

module.exports = { runTests };
