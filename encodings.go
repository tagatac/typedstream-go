package typedstream

import (
	"bytes"
	"fmt"
	"strconv"
)

// endOfEncoding returns the end index (exclusive) of the single encoding starting at start.
func endOfEncoding(enc []byte, start int) (int, error) {
	if start < 0 || start >= len(enc) {
		return 0, fmt.Errorf("start index %d not in range [0, %d)", start, len(enc))
	}

	parenDepth := 0
	i := start
	for i < len(enc) {
		c := enc[i]
		switch {
		case c == '(' || c == '[' || c == '{':
			parenDepth++
			i++
		case parenDepth > 0:
			if c == ')' || c == ']' || c == '}' {
				parenDepth--
			}
			i++
			if parenDepth == 0 {
				return i, nil
			}
		default:
			return i + 1, nil
		}
	}

	if parenDepth > 0 {
		return 0, fmt.Errorf("incomplete encoding, missing %d closing parentheses: %q", parenDepth, enc)
	}
	return 0, fmt.Errorf("incomplete encoding, reached end of string too early: %q", enc)
}

// splitEncodings splits a concatenated type encoding string into individual encodings.
func splitEncodings(enc []byte) ([][]byte, error) {
	var result [][]byte
	start := 0
	for start < len(enc) {
		end, err := endOfEncoding(enc, start)
		if err != nil {
			return nil, err
		}
		result = append(result, enc[start:end])
		start = end
	}
	return result, nil
}

// joinEncodings concatenates a slice of encodings into a single encoding string.
func joinEncodings(encs [][]byte) []byte {
	return bytes.Join(encs, nil)
}

// parseArrayEncoding parses an array encoding like [10i] into (10, "i").
func parseArrayEncoding(enc []byte) (int, []byte, error) {
	if len(enc) < 3 || enc[0] != '[' || enc[len(enc)-1] != ']' {
		return 0, nil, fmt.Errorf("invalid array encoding: %q", enc)
	}

	i := 1
	for i < len(enc)-1 && enc[i] >= '0' && enc[i] <= '9' {
		i++
	}
	lengthStr := enc[1:i]
	elemType := enc[i : len(enc)-1]

	if len(lengthStr) == 0 {
		return 0, nil, fmt.Errorf("missing length in array encoding: %q", enc)
	}
	if len(elemType) == 0 {
		return 0, nil, fmt.Errorf("missing element type in array encoding: %q", enc)
	}

	length, err := strconv.Atoi(string(lengthStr))
	if err != nil {
		return 0, nil, fmt.Errorf("invalid length in array encoding: %q", enc)
	}
	return length, elemType, nil
}

// buildArrayEncoding builds an array encoding like [10i] from length and element type.
func buildArrayEncoding(length int, elemType []byte) ([]byte, error) {
	if length < 0 {
		return nil, fmt.Errorf("array length cannot be negative: %d", length)
	}
	return append(append([]byte("["), []byte(strconv.Itoa(length))...), append(elemType, ']')...), nil
}

// parseStructEncoding parses a struct encoding like {_NSPoint=ff} into (name, fields).
// name is nil for anonymous structs like {ff}.
func parseStructEncoding(enc []byte) ([]byte, [][]byte, error) {
	if len(enc) < 2 || enc[0] != '{' || enc[len(enc)-1] != '}' {
		return nil, nil, fmt.Errorf("invalid struct encoding: %q", enc)
	}

	inner := enc[1 : len(enc)-1]

	// Look for '=' before the first nested '{' to find the name separator.
	var name []byte
	var fieldStr []byte
	innerBracePos := bytes.IndexByte(inner, '{')
	var searchLimit int
	if innerBracePos < 0 {
		searchLimit = len(inner)
	} else {
		searchLimit = innerBracePos
	}
	eqPos := bytes.IndexByte(inner[:searchLimit], '=')
	if eqPos < 0 {
		name = nil
		fieldStr = inner
	} else {
		name = inner[:eqPos]
		fieldStr = inner[eqPos+1:]
	}

	fields, err := splitEncodings(fieldStr)
	if err != nil {
		return nil, nil, fmt.Errorf("error parsing struct fields in %q: %w", enc, err)
	}
	return name, fields, nil
}

// buildStructEncoding builds a struct encoding from a name and field encodings.
// name may be nil for anonymous structs.
func buildStructEncoding(name []byte, fields [][]byte) []byte {
	inner := joinEncodings(fields)
	if name == nil {
		return append(append([]byte("{"), inner...), '}')
	}
	return append(append(append(append([]byte("{"), name...), '='), inner...), '}')
}

// encodingMatchesExpected checks whether actual matches expected,
// accounting for struct names in actual possibly being missing (nil or "?").
func encodingMatchesExpected(actual, expected []byte) bool {
	if bytes.HasPrefix(actual, []byte("{")) && bytes.HasPrefix(expected, []byte("{")) {
		actualName, actualFields, err1 := parseStructEncoding(actual)
		expectedName, expectedFields, err2 := parseStructEncoding(expected)
		if err1 != nil || err2 != nil {
			return false
		}
		nameOK := actualName == nil || bytes.Equal(actualName, []byte("?")) || bytes.Equal(actualName, expectedName)
		return nameOK && allEncodingsMatchExpected(actualFields, expectedFields)
	}
	if bytes.HasPrefix(actual, []byte("[")) && bytes.HasPrefix(expected, []byte("[")) {
		al, ae, err1 := parseArrayEncoding(actual)
		el, ee, err2 := parseArrayEncoding(expected)
		if err1 != nil || err2 != nil {
			return false
		}
		return al == el && encodingMatchesExpected(ae, ee)
	}
	return bytes.Equal(actual, expected)
}

// allEncodingsMatchExpected checks whether all encodings in actual match those in expected.
func allEncodingsMatchExpected(actual, expected [][]byte) bool {
	if len(actual) != len(expected) {
		return false
	}
	for i := range actual {
		if !encodingMatchesExpected(actual[i], expected[i]) {
			return false
		}
	}
	return true
}
