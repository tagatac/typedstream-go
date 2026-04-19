package typedstream

import (
	"io"
	"reflect"
	"testing"
)

// STRING_TEST_DATA encodes an NSString with value "string value" as a little-endian typedstream.
var stringTestData = []byte(
	"\x04\x0bstreamtyped\x81\xe8\x03\x84\x01@\x84\x84\x84\x08NSString\x01\x84\x84\x08NSObject\x00\x85\x84\x01+\x0cstring value\x86",
)

// collectEvents drains all events from a TypedStreamReader.
func collectEvents(t *testing.T, ts *TypedStreamReader) []interface{} {
	t.Helper()
	var events []interface{}
	for {
		ev, err := ts.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Next() error: %v", err)
		}
		events = append(events, ev)
	}
	return events
}

func TestReadDataStream(t *testing.T) {
	ts, err := NewReaderFromData(stringTestData)
	if err != nil {
		t.Fatalf("NewReaderFromData: %v", err)
	}
	defer ts.Close()

	events := collectEvents(t, ts)

	expected := []interface{}{
		BeginTypedValues{Encodings: [][]byte{[]byte("@")}},
		BeginObject{},
		SingleClass{Name: []byte("NSString"), Version: 1},
		SingleClass{Name: []byte("NSObject"), Version: 0},
		nil, // nil superclass of NSObject
		BeginTypedValues{Encodings: [][]byte{[]byte("+")}},
		[]byte("string value"),
		EndTypedValues{},
		EndObject{},
		EndTypedValues{},
	}

	if len(events) != len(expected) {
		t.Fatalf("got %d events, want %d\nevents: %v", len(events), len(expected), events)
	}
	for i, got := range events {
		want := expected[i]
		if !reflect.DeepEqual(got, want) {
			t.Errorf("event[%d]: got %#v (%T), want %#v (%T)", i, got, got, want, want)
		}
	}
}

func TestReadFileStream(t *testing.T) {
	files := []string{
		"testdata/Emacs.clr",
		"testdata/Empty2D macOS 10.14.gcx",
		"testdata/Empty2D macOS 13.gcx",
	}
	for _, name := range files {
		t.Run(name, func(t *testing.T) {
			ts, err := OpenReader(name)
			if err != nil {
				t.Fatalf("OpenReader(%q): %v", name, err)
			}
			defer ts.Close()
			// Drain all events — just verify no errors.
			for {
				_, err := ts.Next()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatalf("Next() error: %v", err)
				}
			}
		})
	}
}

func TestHeaderParsing(t *testing.T) {
	ts, err := NewReaderFromData(stringTestData)
	if err != nil {
		t.Fatalf("NewReaderFromData: %v", err)
	}
	defer ts.Close()

	if ts.StreamerVersion != 4 {
		t.Errorf("StreamerVersion = %d, want 4", ts.StreamerVersion)
	}
	if ts.SystemVersion != 1000 {
		t.Errorf("SystemVersion = %d, want 1000", ts.SystemVersion)
	}
}
