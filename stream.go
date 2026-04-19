package typedstream

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"sync"
)

// Streamer version constants.
const (
	StreamerVersionOldNeXTSTEP = 3
	StreamerVersionCurrent     = 4
)

// System version constants.
const (
	SystemVersionNeXTSTEP082  = 82
	SystemVersionNeXTSTEP083  = 83
	SystemVersionNeXTSTEP090  = 90
	SystemVersionNeXTSTEP0900 = 900
	SystemVersionNeXTSTEP0901 = 901
	SystemVersionNeXTSTEP0905 = 905
	SystemVersionNeXTSTEP0930 = 930
	SystemVersionMacOSX       = 1000
)

// Tag byte constants (signed).
const (
	tagInteger2      = int8(-127)
	tagInteger4      = int8(-126)
	tagFloatingPoint = int8(-125)
	tagNew           = int8(-124)
	tagNil           = int8(-123)
	tagEndOfObject   = int8(-122)
	firstTag         = int8(-128)
	lastTag          = int8(-111)
)

// firstReferenceNumber is the first reference number value.
// It is one higher than lastTag (-111 + 1 = -110).
const firstReferenceNumber = int(-111) + 1

// decodeReferenceNumber converts a reference number as stored in the stream to a zero-based index.
func decodeReferenceNumber(encoded int) int {
	return encoded - firstReferenceNumber
}

func isTag(b int8) bool {
	return b >= firstTag && b <= lastTag
}

// InvalidTypedStreamError is raised when the typedstream data is invalid.
type InvalidTypedStreamError struct {
	Message string
}

func (e *InvalidTypedStreamError) Error() string {
	return e.Message
}

func invalidTSError(format string, args ...interface{}) *InvalidTypedStreamError {
	return &InvalidTypedStreamError{Message: fmt.Sprintf(format, args...)}
}

// ObjRefType describes what a reference refers to.
type ObjRefType int

const (
	ObjRefTypeCString ObjRefType = iota
	ObjRefTypeClass
	ObjRefTypeObject
)

func (t ObjRefType) String() string {
	switch t {
	case ObjRefTypeCString:
		return "C string"
	case ObjRefTypeClass:
		return "class"
	case ObjRefTypeObject:
		return "object"
	default:
		return fmt.Sprintf("ObjRefType(%d)", int(t))
	}
}

// Event types — all implement no interface; callers use type switches on interface{}.

// BeginTypedValues marks the beginning of a typed value group.
type BeginTypedValues struct{ Encodings [][]byte }

func (e BeginTypedValues) String() string {
	return fmt.Sprintf("begin typed values (types %v)", e.Encodings)
}

// EndTypedValues marks the end of a typed value group.
type EndTypedValues struct{}

func (EndTypedValues) String() string { return "end typed values" }

// ObjectReference is a reference to a previously read object/class/C string.
type ObjectReference struct {
	RefType ObjRefType
	Number  int
}

func (e ObjectReference) String() string {
	return fmt.Sprintf("<reference to %s #%d>", e.RefType, e.Number)
}

// Atom is a NeXTSTEP NXAtom (shared/deduplicated C string). Contents is nil for nil atoms.
type Atom struct{ Contents []byte }

func (e Atom) String() string { return fmt.Sprintf("atom: %q", e.Contents) }

// Selector is an Objective-C selector. Name is nil for nil selectors.
type Selector struct{ Name []byte }

func (e Selector) String() string { return fmt.Sprintf("selector: %q", e.Name) }

// CString is a C string stored literally (not as a reference).
type CString struct{ Contents []byte }

func (e CString) String() string { return fmt.Sprintf("C string: %q", e.Contents) }

// SingleClass is one class in a superclass chain stored literally.
type SingleClass struct {
	Name    []byte
	Version int
}

func (e SingleClass) String() string {
	return fmt.Sprintf("class %s v%d", e.Name, e.Version)
}

// BeginObject marks the beginning of a literally stored object.
type BeginObject struct{}

func (BeginObject) String() string { return "begin literal object" }

// EndObject marks the end of a literally stored object.
type EndObject struct{}

func (EndObject) String() string { return "end literal object" }

// ByteArray is an optimized representation for arrays of bytes (signed or unsigned char).
type ByteArray struct {
	ElementEncoding []byte
	Data            []byte
}

func (e ByteArray) String() string {
	return fmt.Sprintf("byte array (element type %q): %q", e.ElementEncoding, e.Data)
}

// BeginArray marks the beginning of a non-byte array.
type BeginArray struct {
	ElementEncoding []byte
	Length          int
}

func (e BeginArray) String() string {
	return fmt.Sprintf("begin array (element type %q, length %d)", e.ElementEncoding, e.Length)
}

// EndArray marks the end of an array.
type EndArray struct{}

func (EndArray) String() string { return "end array" }

// BeginStruct marks the beginning of a struct. Name is nil for anonymous structs.
type BeginStruct struct {
	Name           []byte
	FieldEncodings [][]byte
}

func (e BeginStruct) String() string {
	name := "(no name)"
	if e.Name != nil {
		name = string(e.Name)
	}
	return fmt.Sprintf("begin struct %s (field types %v)", name, e.FieldEncodings)
}

// EndStruct marks the end of a struct.
type EndStruct struct{}

func (EndStruct) String() string { return "end struct" }

// TypedStreamReader reads typedstream data from a byte stream, emitting events.
type TypedStreamReader struct {
	StreamerVersion   int
	ByteOrder         binary.ByteOrder
	SystemVersion     int
	SharedStringTable [][]byte

	r           io.Reader
	closeStream bool

	events    chan interface{} // buffered channel of events
	done      chan struct{}
	closeOnce sync.Once
}

// NewReaderFromData creates a TypedStreamReader from raw bytes.
func NewReaderFromData(data []byte) (*TypedStreamReader, error) {
	return newReader(&bytesReadSeeker{data: data, pos: 0}, false)
}

// OpenReader opens a typedstream file by path.
func OpenReader(filename string) (*TypedStreamReader, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	return newReader(f, true)
}

// NewReader creates a TypedStreamReader from an io.Reader.
func NewReader(r io.Reader) (*TypedStreamReader, error) {
	return newReader(r, false)
}

type bytesReadSeeker struct {
	data []byte
	pos  int
}

func (b *bytesReadSeeker) Read(p []byte) (int, error) {
	if b.pos >= len(b.data) {
		return 0, io.EOF
	}
	n := copy(p, b.data[b.pos:])
	b.pos += n
	return n, nil
}

func newReader(r io.Reader, closeStream bool) (*TypedStreamReader, error) {
	tr := &TypedStreamReader{
		r:           r,
		closeStream: closeStream,
		events:      make(chan interface{}, 64),
		done:        make(chan struct{}),
	}

	if err := tr.readHeader(); err != nil {
		tr.Close()
		return nil, err
	}

	go tr.generateEvents()
	return tr, nil
}

// Close stops event generation and optionally closes the underlying reader.
func (t *TypedStreamReader) Close() error {
	t.closeOnce.Do(func() {
		close(t.done)
		// Drain remaining events so the goroutine can finish.
		go func() {
			for range t.events {
			}
		}()
	})
	if t.closeStream {
		if c, ok := t.r.(io.Closer); ok {
			return c.Close()
		}
	}
	return nil
}

// Next returns the next event from the stream.
// Returns (nil, nil) for a nil event value (nil object/string).
// Returns (nil, io.EOF) at end of stream.
// Returns (nil, err) on error.
func (t *TypedStreamReader) Next() (interface{}, error) {
	ev, ok := <-t.events
	if !ok {
		return nil, io.EOF
	}
	if we, ok := ev.(errWrapper); ok {
		return nil, we.err
	}
	return ev, nil
}

// errWrapper wraps an error for transmission through the events channel.
type errWrapper struct{ err error }

func (t *TypedStreamReader) generateEvents() {
	defer close(t.events)
	t.readAllValues()
}

func (t *TypedStreamReader) send(ev interface{}) bool {
	select {
	case t.events <- ev:
		return true
	case <-t.done:
		return false
	}
}

func (t *TypedStreamReader) sendErr(err error) {
	select {
	case t.events <- errWrapper{err}:
	case <-t.done:
	}
}

func (t *TypedStreamReader) readExact(n int) ([]byte, error) {
	buf := make([]byte, n)
	_, err := io.ReadFull(t.r, buf)
	if err != nil {
		if err == io.ErrUnexpectedEOF {
			return nil, invalidTSError("attempted to read %d bytes but hit EOF", n)
		}
		return nil, invalidTSError("read error: %v", err)
	}
	return buf, nil
}

func (t *TypedStreamReader) readOneByte() (int8, error) {
	buf, err := t.readExact(1)
	if err != nil {
		return 0, err
	}
	return int8(buf[0]), nil
}

// readHeadByte reads a head byte, or returns the provided lookahead if non-nil.
func (t *TypedStreamReader) readHeadByte(head *int8) (int8, error) {
	if head != nil {
		return *head, nil
	}
	return t.readOneByte()
}

// readUnsignedInteger reads a variable-length unsigned integer.
func (t *TypedStreamReader) readUnsignedInteger(head *int8) (uint64, error) {
	h, err := t.readHeadByte(head)
	if err != nil {
		return 0, err
	}
	if !isTag(h) {
		return uint64(uint8(h)), nil
	}
	switch h {
	case tagInteger2:
		buf, err := t.readExact(2)
		if err != nil {
			return 0, err
		}
		return uint64(t.ByteOrder.Uint16(buf)), nil
	case tagInteger4:
		buf, err := t.readExact(4)
		if err != nil {
			return 0, err
		}
		return uint64(t.ByteOrder.Uint32(buf)), nil
	default:
		return 0, invalidTSError("invalid head tag in integer context: %d", h)
	}
}

// readSignedInteger reads a variable-length signed integer.
func (t *TypedStreamReader) readSignedInteger(head *int8) (int64, error) {
	h, err := t.readHeadByte(head)
	if err != nil {
		return 0, err
	}
	if !isTag(h) {
		return int64(h), nil
	}
	switch h {
	case tagInteger2:
		buf, err := t.readExact(2)
		if err != nil {
			return 0, err
		}
		return int64(int16(t.ByteOrder.Uint16(buf))), nil
	case tagInteger4:
		buf, err := t.readExact(4)
		if err != nil {
			return 0, err
		}
		return int64(int32(t.ByteOrder.Uint32(buf))), nil
	default:
		return 0, invalidTSError("invalid head tag in integer context: %d", h)
	}
}

func (t *TypedStreamReader) readHeader() error {
	// First 2 bytes: streamer version and signature length.
	buf, err := t.readExact(2)
	if err != nil {
		return invalidTSError("failed to read header: %v", err)
	}
	t.StreamerVersion = int(buf[0])
	sigLen := int(buf[1])

	if t.StreamerVersion < StreamerVersionOldNeXTSTEP || t.StreamerVersion > StreamerVersionCurrent {
		return invalidTSError("invalid streamer version: %d", t.StreamerVersion)
	}
	if t.StreamerVersion == StreamerVersionOldNeXTSTEP {
		return invalidTSError("old NeXTSTEP streamer version (%d) not supported", t.StreamerVersion)
	}

	const signatureLength = 11
	if sigLen != signatureLength {
		return invalidTSError("signature length must be %d, got %d", signatureLength, sigLen)
	}

	sig, err := t.readExact(signatureLength)
	if err != nil {
		return invalidTSError("failed to read signature: %v", err)
	}

	switch string(sig) {
	case "typedstream":
		t.ByteOrder = binary.BigEndian
	case "streamtyped":
		t.ByteOrder = binary.LittleEndian
	default:
		return invalidTSError("invalid signature: %q", sig)
	}

	sysVer, err := t.readUnsignedInteger(nil)
	if err != nil {
		return invalidTSError("failed to read system version: %v", err)
	}
	t.SystemVersion = int(sysVer)
	return nil
}

func (t *TypedStreamReader) readFloat(head *int8) (float32, error) {
	h, err := t.readHeadByte(head)
	if err != nil {
		return 0, err
	}
	if h == tagFloatingPoint {
		buf, err := t.readExact(4)
		if err != nil {
			return 0, err
		}
		bits := t.ByteOrder.Uint32(buf)
		return math.Float32frombits(bits), nil
	}
	// Otherwise treat as integer.
	i, err := t.readSignedInteger(&h)
	if err != nil {
		return 0, err
	}
	return float32(i), nil
}

func (t *TypedStreamReader) readDouble(head *int8) (float64, error) {
	h, err := t.readHeadByte(head)
	if err != nil {
		return 0, err
	}
	if h == tagFloatingPoint {
		buf, err := t.readExact(8)
		if err != nil {
			return 0, err
		}
		bits := t.ByteOrder.Uint64(buf)
		return math.Float64frombits(bits), nil
	}
	i, err := t.readSignedInteger(&h)
	if err != nil {
		return 0, err
	}
	return float64(i), nil
}

// readUnsharedString reads a string without reference tracking. Returns nil for nil strings.
func (t *TypedStreamReader) readUnsharedString(head *int8) ([]byte, error) {
	h, err := t.readHeadByte(head)
	if err != nil {
		return nil, err
	}
	if h == tagNil {
		return nil, nil
	}
	length, err := t.readUnsignedInteger(&h)
	if err != nil {
		return nil, err
	}
	return t.readExact(int(length))
}

// readSharedString reads a string with reference tracking. Returns nil for nil strings.
func (t *TypedStreamReader) readSharedString(head *int8) ([]byte, error) {
	h, err := t.readHeadByte(head)
	if err != nil {
		return nil, err
	}
	if h == tagNil {
		return nil, nil
	}
	if h == tagNew {
		s, err := t.readUnsharedString(nil)
		if err != nil {
			return nil, err
		}
		if s == nil {
			return nil, invalidTSError("literal shared string cannot contain a nil unshared string")
		}
		t.SharedStringTable = append(t.SharedStringTable, s)
		return s, nil
	}
	// Reference to previous shared string.
	refNum, err := t.readSignedInteger(&h)
	if err != nil {
		return nil, err
	}
	idx := decodeReferenceNumber(int(refNum))
	if idx < 0 || idx >= len(t.SharedStringTable) {
		return nil, invalidTSError("shared string reference %d out of range (table size %d)", idx, len(t.SharedStringTable))
	}
	return t.SharedStringTable[idx], nil
}

func (t *TypedStreamReader) readObjectReference(refType ObjRefType, head *int8) (ObjectReference, error) {
	refNum, err := t.readSignedInteger(head)
	if err != nil {
		return ObjectReference{}, err
	}
	return ObjectReference{RefType: refType, Number: decodeReferenceNumber(int(refNum))}, nil
}

// readCString reads a C string, returned as CString (literal) or ObjectReference (backreference), or nil.
func (t *TypedStreamReader) readCString(head *int8) (interface{}, error) {
	h, err := t.readHeadByte(head)
	if err != nil {
		return nil, err
	}
	if h == tagNil {
		return nil, nil
	}
	if h == tagNew {
		s, err := t.readSharedString(nil)
		if err != nil {
			return nil, err
		}
		if s == nil {
			return nil, invalidTSError("literal C string cannot contain a nil shared string")
		}
		for _, b := range s {
			if b == 0 {
				return nil, invalidTSError("C string cannot contain zero bytes")
			}
		}
		return CString{Contents: s}, nil
	}
	ref, err := t.readObjectReference(ObjRefTypeCString, &h)
	if err != nil {
		return nil, err
	}
	return ref, nil
}

// readClass reads a class chain, sending events to the channel.
func (t *TypedStreamReader) readClass(head *int8) bool {
	h, err := t.readHeadByte(head)
	if err != nil {
		t.sendErr(err)
		return false
	}
	for h == tagNew {
		name, err := t.readSharedString(nil)
		if err != nil {
			t.sendErr(err)
			return false
		}
		if name == nil {
			t.sendErr(invalidTSError("class name cannot be nil"))
			return false
		}
		version, err := t.readSignedInteger(nil)
		if err != nil {
			t.sendErr(err)
			return false
		}
		if !t.send(SingleClass{Name: name, Version: int(version)}) {
			return false
		}
		h, err = t.readOneByte()
		if err != nil {
			t.sendErr(err)
			return false
		}
	}

	if h == tagNil {
		return t.send(nil)
	}
	ref, err := t.readObjectReference(ObjRefTypeClass, &h)
	if err != nil {
		t.sendErr(err)
		return false
	}
	return t.send(ref)
}

// readObject reads an object from the stream, sending events.
func (t *TypedStreamReader) readObject(head *int8) bool {
	h, err := t.readHeadByte(head)
	if err != nil {
		t.sendErr(err)
		return false
	}
	if h == tagNil {
		return t.send(nil)
	}
	if h != tagNew {
		// Backreference.
		ref, err := t.readObjectReference(ObjRefTypeObject, &h)
		if err != nil {
			t.sendErr(err)
			return false
		}
		return t.send(ref)
	}

	if !t.send(BeginObject{}) {
		return false
	}
	if !t.readClass(nil) {
		return false
	}

	// Read typed value groups until end-of-object.
	for {
		nextH, err := t.readOneByte()
		if err != nil {
			t.sendErr(err)
			return false
		}
		if nextH == tagEndOfObject {
			break
		}
		if !t.readTypedValues(&nextH) {
			return false
		}
	}

	return t.send(EndObject{})
}

// readValueWithEncoding reads a single typed value and sends events.
func (t *TypedStreamReader) readValueWithEncoding(enc []byte, head *int8) bool {
	if len(enc) == 0 {
		t.sendErr(invalidTSError("empty type encoding"))
		return false
	}

	switch {
	case len(enc) == 1 && enc[0] == 'B':
		// Boolean: always stored as 1 literal byte, no tag.
		buf, err := t.readExact(1)
		if err != nil {
			t.sendErr(err)
			return false
		}
		switch buf[0] {
		case 0:
			return t.send(false)
		case 1:
			return t.send(true)
		default:
			t.sendErr(invalidTSError("boolean value should be 0 or 1, not %d", buf[0]))
			return false
		}

	case len(enc) == 1 && enc[0] == 'C':
		// Unsigned char: always 1 literal byte.
		buf, err := t.readExact(1)
		if err != nil {
			t.sendErr(err)
			return false
		}
		return t.send(int64(buf[0]))

	case len(enc) == 1 && enc[0] == 'c':
		// Signed char: always 1 literal byte.
		buf, err := t.readExact(1)
		if err != nil {
			t.sendErr(err)
			return false
		}
		return t.send(int64(int8(buf[0])))

	case len(enc) == 1 && (enc[0] == 'S' || enc[0] == 'I' || enc[0] == 'L' || enc[0] == 'Q'):
		u, err := t.readUnsignedInteger(head)
		if err != nil {
			t.sendErr(err)
			return false
		}
		return t.send(int64(u))

	case len(enc) == 1 && (enc[0] == 's' || enc[0] == 'i' || enc[0] == 'l' || enc[0] == 'q'):
		i, err := t.readSignedInteger(head)
		if err != nil {
			t.sendErr(err)
			return false
		}
		return t.send(i)

	case len(enc) == 1 && enc[0] == 'f':
		f, err := t.readFloat(head)
		if err != nil {
			t.sendErr(err)
			return false
		}
		return t.send(f)

	case len(enc) == 1 && enc[0] == 'd':
		d, err := t.readDouble(head)
		if err != nil {
			t.sendErr(err)
			return false
		}
		return t.send(d)

	case len(enc) == 1 && enc[0] == '*':
		cs, err := t.readCString(head)
		if err != nil {
			t.sendErr(err)
			return false
		}
		return t.send(cs)

	case len(enc) == 1 && enc[0] == '%':
		s, err := t.readSharedString(head)
		if err != nil {
			t.sendErr(err)
			return false
		}
		return t.send(Atom{Contents: s})

	case len(enc) == 1 && enc[0] == ':':
		s, err := t.readSharedString(head)
		if err != nil {
			t.sendErr(err)
			return false
		}
		return t.send(Selector{Name: s})

	case len(enc) == 1 && enc[0] == '+':
		s, err := t.readUnsharedString(head)
		if err != nil {
			t.sendErr(err)
			return false
		}
		return t.send(s) // []byte or nil

	case len(enc) == 1 && enc[0] == '#':
		return t.readClass(head)

	case len(enc) == 1 && enc[0] == '@':
		return t.readObject(head)

	case len(enc) == 1 && enc[0] == '!':
		// Ignored field — no data in stream.
		return t.send(nil)

	case enc[0] == '[':
		length, elemEnc, err := parseArrayEncoding(enc)
		if err != nil {
			t.sendErr(err)
			return false
		}
		// Byte arrays are handled as a single ByteArray event.
		if len(elemEnc) == 1 && (elemEnc[0] == 'C' || elemEnc[0] == 'c') {
			data, err := t.readExact(length)
			if err != nil {
				t.sendErr(err)
				return false
			}
			return t.send(ByteArray{ElementEncoding: elemEnc, Data: data})
		}
		if !t.send(BeginArray{ElementEncoding: elemEnc, Length: length}) {
			return false
		}
		for i := 0; i < length; i++ {
			if !t.readValueWithEncoding(elemEnc, nil) {
				return false
			}
		}
		return t.send(EndArray{})

	case enc[0] == '{':
		name, fieldEncs, err := parseStructEncoding(enc)
		if err != nil {
			t.sendErr(err)
			return false
		}
		if !t.send(BeginStruct{Name: name, FieldEncodings: fieldEncs}) {
			return false
		}
		for _, fEnc := range fieldEncs {
			if !t.readValueWithEncoding(fEnc, nil) {
				return false
			}
		}
		return t.send(EndStruct{})

	default:
		t.sendErr(invalidTSError("unknown type encoding: %q", enc))
		return false
	}
}

// readTypedValues reads one typed value group (encoding string + values).
func (t *TypedStreamReader) readTypedValues(head *int8) bool {
	encStr, err := t.readSharedString(head)
	if err != nil {
		t.sendErr(err)
		return false
	}
	if encStr == nil {
		t.sendErr(invalidTSError("nil type encoding string"))
		return false
	}
	if len(encStr) == 0 {
		t.sendErr(invalidTSError("empty type encoding string"))
		return false
	}

	typeEncs, err := splitEncodings(encStr)
	if err != nil {
		t.sendErr(err)
		return false
	}

	if !t.send(BeginTypedValues{Encodings: typeEncs}) {
		return false
	}
	for _, enc := range typeEncs {
		if !t.readValueWithEncoding(enc, nil) {
			return false
		}
	}
	return t.send(EndTypedValues{})
}

func (t *TypedStreamReader) readAllValues() {
	for {
		// Read one byte directly to detect clean EOF.
		var buf [1]byte
		_, err := t.r.Read(buf[:])
		if err == io.EOF {
			return // clean end of stream
		}
		if err != nil {
			t.sendErr(invalidTSError("read error: %v", err))
			return
		}
		h := int8(buf[0])
		if !t.readTypedValues(&h) {
			return
		}
	}
}
