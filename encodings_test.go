package typedstream

import (
	"reflect"
	"testing"
)

func TestSplitEncodings(t *testing.T) {
	tests := []struct {
		input    []byte
		expected [][]byte
	}{
		{[]byte("@"), [][]byte{[]byte("@")}},
		{[]byte("if"), [][]byte{[]byte("i"), []byte("f")}},
		{[]byte("{_NSPoint=ff}"), [][]byte{[]byte("{_NSPoint=ff}")}},
		{[]byte("{_NSRect={_NSPoint=ff}{_NSSize=ff}}"), [][]byte{[]byte("{_NSRect={_NSPoint=ff}{_NSSize=ff}}")}},
		{[]byte("[10i]"), [][]byte{[]byte("[10i]")}},
		{[]byte("@i"), [][]byte{[]byte("@"), []byte("i")}},
		{[]byte(""), nil},
	}
	for _, tt := range tests {
		got, err := splitEncodings(tt.input)
		if err != nil {
			t.Errorf("splitEncodings(%q) error: %v", tt.input, err)
			continue
		}
		if !reflect.DeepEqual(got, tt.expected) {
			t.Errorf("splitEncodings(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestParseArrayEncoding(t *testing.T) {
	length, elem, err := parseArrayEncoding([]byte("[10i]"))
	if err != nil || length != 10 || string(elem) != "i" {
		t.Errorf("parseArrayEncoding([10i]) = (%d, %q, %v)", length, elem, err)
	}

	length, elem, err = parseArrayEncoding([]byte("[3{_NSPoint=ff}]"))
	if err != nil || length != 3 || string(elem) != "{_NSPoint=ff}" {
		t.Errorf("parseArrayEncoding([3{_NSPoint=ff}]) = (%d, %q, %v)", length, elem, err)
	}
}

func TestBuildArrayEncoding(t *testing.T) {
	got, err := buildArrayEncoding(10, []byte("i"))
	if err != nil || string(got) != "[10i]" {
		t.Errorf("buildArrayEncoding(10, i) = (%q, %v)", got, err)
	}
}

func TestParseStructEncoding(t *testing.T) {
	tests := []struct {
		input      string
		wantName   string
		wantFields []string
	}{
		{"{_NSPoint=ff}", "_NSPoint", []string{"f", "f"}},
		{"{_NSRect={_NSPoint=ff}{_NSSize=ff}}", "_NSRect", []string{"{_NSPoint=ff}", "{_NSSize=ff}"}},
		{"{ff}", "", []string{"f", "f"}}, // anonymous struct
	}
	for _, tt := range tests {
		name, fields, err := parseStructEncoding([]byte(tt.input))
		if err != nil {
			t.Errorf("parseStructEncoding(%q) error: %v", tt.input, err)
			continue
		}
		gotName := string(name)
		if tt.wantName == "" {
			if name != nil {
				t.Errorf("parseStructEncoding(%q) name = %q, want nil", tt.input, gotName)
			}
		} else if gotName != tt.wantName {
			t.Errorf("parseStructEncoding(%q) name = %q, want %q", tt.input, gotName, tt.wantName)
		}
		if len(fields) != len(tt.wantFields) {
			t.Errorf("parseStructEncoding(%q) fields = %v, want %v", tt.input, fields, tt.wantFields)
			continue
		}
		for i, f := range fields {
			if string(f) != tt.wantFields[i] {
				t.Errorf("parseStructEncoding(%q) field[%d] = %q, want %q", tt.input, i, f, tt.wantFields[i])
			}
		}
	}
}

func TestBuildStructEncoding(t *testing.T) {
	got := buildStructEncoding([]byte("_NSPoint"), [][]byte{[]byte("f"), []byte("f")})
	if string(got) != "{_NSPoint=ff}" {
		t.Errorf("buildStructEncoding(_NSPoint, [f f]) = %q", got)
	}
	got = buildStructEncoding(nil, [][]byte{[]byte("f"), []byte("f")})
	if string(got) != "{ff}" {
		t.Errorf("buildStructEncoding(nil, [f f]) = %q", got)
	}
}

func TestEncodingMatchesExpected(t *testing.T) {
	if !encodingMatchesExpected([]byte("i"), []byte("i")) {
		t.Error("i should match i")
	}
	if encodingMatchesExpected([]byte("i"), []byte("f")) {
		t.Error("i should not match f")
	}
	// Anonymous struct matches named struct with same fields
	if !encodingMatchesExpected([]byte("{ff}"), []byte("{_NSPoint=ff}")) {
		t.Error("anonymous struct should match named struct with same fields")
	}
	if !encodingMatchesExpected([]byte("{_NSPoint=ff}"), []byte("{_NSPoint=ff}")) {
		t.Error("same struct should match")
	}
}
