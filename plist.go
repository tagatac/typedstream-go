// Package typedstream parses Apple's NSArchiver "typedstream" binary format.
//
// This file implements an old binary property list format originally from NeXTSTEP.
// It is NOT the same as the modern Mac OS X/macOS binary property list format (bplist00).
// It is used by -[NSArchiver encodePropertyList:] / -[NSUnarchiver decodePropertyList].
package typedstream

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"unicode/utf16"
)

// NeXTSTEP 8-bit character set mapping (index = byte value, value = Unicode code point).
// Bytes 0x00 and 0xFF are unassigned (mapped to U+0000 and U+0000).
var nextStepCharMap = buildNextStepCharMap()

func buildNextStepCharMap() [256]rune {
	// The mapping string covers bytes 0x01–0xFD (253 chars, index 0 = byte 0x01).
	mapping := "\x01\x02\x03\x04\x05\x06\x07\x08\t\n\x0b\x0c\r\x0e\x0f" +
		"\x10\x11\x12\x13\x14\x15\x16\x17\x18\x19\x1a\x1b\x1c\x1d\x1e\x1f" +
		" !\"#$%&'()*+,-./" +
		"0123456789:;<=>?" +
		"@ABCDEFGHIJKLMNO" +
		"PQRSTUVWXYZ[\\]^_" +
		"`abcdefghijklmno" +
		"pqrstuvwxyz{|}~\x7f" +
		"\u00a0\u00c0\u00c1\u00c2\u00c3\u00c4\u00c5\u00c7\u00c8\u00c9\u00ca\u00cb\u00cc\u00cd\u00ce\u00cf" +
		"\u00d0\u00d1\u00d2\u00d3\u00d4\u00d5\u00d6\u00d9\u00da\u00db\u00dc\u00dd\u00de\u00b5\u00d7\u00f7" +
		"\u00a9\u00a1\u00a2\u00a3\u2044\u00a5\u0192\u00a7\u00a4\u2018\u2019\u00ab\u2039\u203a\ufb01\ufb02" +
		"\u00ae\u2013\u2020\u2021\u00b7\u00a6\u00b6\u2022\u201a\u201e\u201c\u00bb\u2026\u2030\u00ac\u00bf" +
		"\u00b9\u02cb\u00b4\u02c6\u02dc\u00af\u02d8\u02d9\u00a8\u00b2\u02da\u00b8\u00b3\u02dd\u02db\u02c7" +
		"\u2014\u00b1\u00bc\u00bd\u00be\u00e0\u00e1\u00e2\u00e3\u00e4\u00e5\u00e7\u00e8\u00e9\u00ea\u00eb" +
		"\u00ec\u00c6\u00ed\u00aa\u00ee\u00ef\u00f0\u00f1\u0141\u00d8\u0152\u00ba\u00f2\u00f3\u00f4\u00f5" +
		"\u00f6\u00e6\u00f9\u00fa\u00fb\u0131\u00fc\u00fd\u0142\u00f8\u0153\u00df\u00fe\u00ff"
	// mapping[i] corresponds to byte value i+1.

	var m [256]rune
	for i, r := range []rune(mapping) {
		m[i+1] = r
	}
	return m
}

func plistReadExact(r *bytes.Reader, n int) ([]byte, error) {
	buf := make([]byte, n)
	if _, err := r.Read(buf); err != nil {
		return nil, fmt.Errorf("old plist: failed to read %d bytes: %w", n, err)
	}
	return buf, nil
}

func plistReadUint32LE(r *bytes.Reader) (uint32, error) {
	buf, err := plistReadExact(r, 4)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(buf), nil
}

// deserializeOldBinaryPlistFromReader deserializes from a bytes.Reader (for position tracking).
func deserializeOldBinaryPlistFromReader(r *bytes.Reader) (interface{}, error) {
	typeNum, err := plistReadUint32LE(r)
	if err != nil {
		return nil, err
	}

	switch typeNum {
	case 4, 5, 6:
		// Byte-length-prefixed data or string.
		dataLen, err := plistReadUint32LE(r)
		if err != nil {
			return nil, err
		}
		data, err := plistReadExact(r, int(dataLen))
		if err != nil {
			return nil, err
		}
		// Alignment padding to 4-byte boundary.
		padLen := (4 - int(dataLen)%4) % 4
		if padLen > 0 {
			pad, err := plistReadExact(r, padLen)
			if err != nil {
				return nil, err
			}
			for _, b := range pad {
				if b != 0 {
					return nil, fmt.Errorf("old plist: alignment padding should be zero bytes")
				}
			}
		}

		switch typeNum {
		case 4: // NSData
			return data, nil
		case 5: // NSString in NeXTSTEP 8-bit encoding
			runes := make([]rune, len(data))
			for i, b := range data {
				runes[i] = nextStepCharMap[b]
			}
			return string(runes), nil
		case 6: // NSString in UTF-16 with BOM
			if len(data) < 2 {
				return nil, fmt.Errorf("old plist: UTF-16 string too short")
			}
			var byteOrder binary.ByteOrder
			switch {
			case data[0] == 0xFF && data[1] == 0xFE:
				byteOrder = binary.LittleEndian
				data = data[2:]
			case data[0] == 0xFE && data[1] == 0xFF:
				byteOrder = binary.BigEndian
				data = data[2:]
			default:
				return nil, fmt.Errorf("old plist: UTF-16 missing BOM")
			}
			if len(data)%2 != 0 {
				return nil, fmt.Errorf("old plist: odd-length UTF-16 data")
			}
			u16 := make([]uint16, len(data)/2)
			for i := range u16 {
				u16[i] = byteOrder.Uint16(data[i*2:])
			}
			return string(utf16.Decode(u16)), nil
		}

	case 2, 7:
		// NSArray (2) or NSDictionary (7).
		elemCount, err := plistReadUint32LE(r)
		if err != nil {
			return nil, err
		}
		n := int(elemCount)

		var keys []string
		if typeNum == 7 {
			keys = make([]string, n)
			for i := range keys {
				k, err := deserializeOldBinaryPlistFromReader(r)
				if err != nil {
					return nil, err
				}
				s, ok := k.(string)
				if !ok {
					return nil, fmt.Errorf("old plist: dictionary key must be string, got %T", k)
				}
				keys[i] = s
			}
		}

		valueLengths := make([]int, n)
		for i := range valueLengths {
			l, err := plistReadUint32LE(r)
			if err != nil {
				return nil, err
			}
			valueLengths[i] = int(l)
		}

		values := make([]interface{}, n)
		for i, expectedLen := range valueLengths {
			posBefore := int(r.Size()) - r.Len()
			v, err := deserializeOldBinaryPlistFromReader(r)
			if err != nil {
				return nil, err
			}
			posAfter := int(r.Size()) - r.Len()
			if posAfter-posBefore != expectedLen {
				return nil, fmt.Errorf("old plist: value[%d] expected length %d, got %d", i, expectedLen, posAfter-posBefore)
			}
			values[i] = v
		}

		if typeNum == 2 {
			return values, nil
		}
		// NSDictionary
		m := make(map[string]interface{}, n)
		for i, k := range keys {
			m[k] = values[i]
		}
		return m, nil

	case 8:
		return nil, nil

	default:
		return nil, fmt.Errorf("old plist: unknown type number %d", typeNum)
	}
	return nil, nil // unreachable
}

// deserializeOldBinaryPlist deserializes an old NeXTSTEP binary property list.
// This is NOT the modern bplist00 format.
func deserializeOldBinaryPlist(data []byte) (interface{}, error) {
	r := bytes.NewReader(data)
	v, err := deserializeOldBinaryPlistFromReader(r)
	if err != nil {
		return nil, err
	}
	if r.Len() != 0 {
		return nil, fmt.Errorf("old plist: %d bytes of trailing data", r.Len())
	}
	return v, nil
}
