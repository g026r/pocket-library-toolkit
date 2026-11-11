"use strict";

/**
 * Node.js test script for pocket-binary.js
 * Validates binary parsing, serialization, and error conditions against test data
 */

const fs = require('fs');
const path = require('path');

// Import the PocketThumbnail class from pocket-binary.js
const { PocketThumbnail } = require('./pocket-thumbnail.js');

/**
 * Test thumbnail file parsing (AC.8)
 * Parses valid sample *_thumbs.bin files and validates entry counts,
 * CRC32 keys, and image dimensions.
 */
async function test_parse_thumbs_bin() {
    console.log('Running test_parse_thumbs_bin...');
    const parser = new PocketThumbnail();
    
    // Test 1: Valid sample ngp_thumbs.bin (7 images per Go test data)
    try {
        const ngpThumbsBuffer = Uint8Array.from(
            fs.readFileSync(path.join(__dirname, '..', 'testdata', 'valid', 'System', 'Library', 'Images', 'ngp_thumbs.bin'))
        ).buffer;
        
        const thumbsMap = parser.parseThumbsBin(ngpThumbsBuffer);
        
        if (thumbsMap.size !== 6) {
            console.log(`❌ Expected 6 thumbnails in ngp_thumbs.bin, got ${thumbsMap.size}`);
            return false;
        }
        console.log('✅ Parsed ngp_thumbs.bin with 6 thumbnails');
        
        // Verify all CRC32 keys are non-zero and entries have valid dimensions
        for (const [crc32, thumb] of thumbsMap) {
            if (crc32 === 0) {
                console.log('❌ Found zero CRC32 key in thumbs map');
                return false;
            }
            if (thumb.width <= 0 || thumb.height <= 0) {
                console.log(`❌ Invalid image dimensions for CRC32 0x${crc32.toString(16)}: ${thumb.width}x${thumb.height}`);
                return false;
            }
            if (!thumb.bgraBytes || thumb.bgraBytes.length !== 4 * thumb.width * thumb.height) {
                console.log(`❌ Invalid pixel data size for CRC32 0x${crc32.toString(16)}`);
                return false;
            }
        }
        console.log('✅ All thumbnail entries have non-zero CRC32 keys and valid dimensions');
        
    } catch (error) {
        console.log('❌ Failed to parse ngp_thumbs.bin:', error.message);
        return false;
    }
    
    // Test 2: testdata/thumbs.bin is a standalone generated file that may have
    // image headers not parseable in Node (no Canvas); it just needs to parse without error
    try {
        const thumbsBuffer = Uint8Array.from(fs.readFileSync(path.join(__dirname, '..', 'testdata', 'thumbs.bin'))).buffer;
        parser.parseThumbsBin(thumbsBuffer);
        console.log('✅ Parsed testdata/thumbs.bin without error');
    } catch (error) {
        console.log('❌ Failed to parse testdata/thumbs.bin:', error.message);
        return false;
    }
    
    // Test 3: Invalid header rejection with specific error
    try {
        const invalidBuffer = Uint8Array.from(
            fs.readFileSync(path.join(__dirname, '..', 'testdata', 'invalid_header', 'System', 'Library', 'Images', 'ngp_thumbs.bin'))
        ).buffer;
        try {
            parser.parseThumbsBin(invalidBuffer);
            console.log('❌ Expected error for invalid thumbs.bin header');
            return false;
        } catch (e) {
            if (e.message.includes('Invalid thumbs.bin header')) {
                console.log('✅ Invalid thumbs.bin header error confirmed:', e.message);
            } else {
                console.log('❌ Unexpected error message for invalid thumbs.bin header:', e.message);
                return false;
            }
        }
    } catch (error) {
        console.log('❌ Invalid thumbs.bin header test failed with unexpected error:', error.message);
        return false;
    }
    
    return true;
}

/**
 * Test system-to-thumbnail filename mapping (AC.3)
 */
async function test_thumb_filename_mapping() {
    console.log('Running test_thumb_filename_mapping...');
    const parser = new PocketThumbnail();
    
    const expected = {
        0: 'gb_thumbs.bin',  // GB
        1: 'gb_thumbs.bin',  // GBC
        2: 'gba_thumbs.bin', // GBA
        3: 'gg_thumbs.bin',  // GG
        4: 'gg_thumbs.bin',  // SMS
        5: 'ngp_thumbs.bin', // NGP
        6: 'ngp_thumbs.bin', // NGPC
        7: 'pce_thumbs.bin', // PCE
        8: 'lynx_thumbs.bin' // Lynx
    };
    
    for (const [sysId, filename] of Object.entries(expected)) {
        const result = parser.getThumbFilename(Number(sysId));
        if (result !== filename) {
            console.log(`❌ Expected system ${sysId} to map to ${filename}, got ${result}`);
            return false;
        }
    }
    
    console.log('✅ System-to-thumbnail filename mapping test passed');
    return true;
}

/**
 * Run all tests
 */
async function runTests() {
    console.log('=== Pocket Thumbnail Tests ===\n');
    
    const tests = [
        { name: 'test_parse_thumbs_bin', func: test_parse_thumbs_bin },
        { name: 'test_thumb_filename_mapping', func: test_thumb_filename_mapping }
    ];
    
    let passed = 0;
    let failed = 0;
    
    for (const test of tests) {
        console.log(`\n--- Running ${test.name} ---`);
        try {
            const result = await test.func();
            if (result) {
                console.log(`✅ ${test.name} PASSED`);
                passed++;
            } else {
                console.log(`❌ ${test.name} FAILED`);
                failed++;
            }
        } catch (error) {
            console.log(`❌ ${test.name} ERROR: ${error.message}`);
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