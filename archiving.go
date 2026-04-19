package typedstream

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// ArchivedObject is implemented by all known archived Objective-C object types.
// Each concrete type's InitFromUnarchiver must call its embedded parent's
// InitFromUnarchiver first (with class.Superclass), then read its own data.
type ArchivedObject interface {
	InitFromUnarchiver(u Unarchiver, class *Class) error
	AllowsExtraData() bool
	AddExtraField(field *TypedGroup) error
	FormatLines(seen map[uintptr]bool) []string
}

// KnownStruct is implemented by known C struct types.
type KnownStruct interface {
	StructName() []byte
	FieldEncodings() [][]byte
	FormatLines(seen map[uintptr]bool) []string
}

// ---- Registries ----

// archivedClassesByName maps archived class name (bytes as string) to a factory function.
var archivedClassesByName = map[string]func() ArchivedObject{}

// structFactoriesByEncoding maps full struct encoding (e.g. "{_NSPoint=ff}") to a factory.
var structFactoriesByEncoding = map[string]func(fields []interface{}) (KnownStruct, error){}

// RegisterArchivedClass registers a factory for a known archived class.
func RegisterArchivedClass(name []byte, factory func() ArchivedObject) {
	archivedClassesByName[string(name)] = factory
}

// RegisterStructClass registers a factory for a known struct type.
func RegisterStructClass(encoding []byte, factory func(fields []interface{}) (KnownStruct, error)) {
	structFactoriesByEncoding[string(encoding)] = factory
}

// ---- Core data types ----

// Class holds information about an archived Objective-C class.
type Class struct {
	Name       []byte
	Version    int
	Superclass *Class
}

func (c *Class) String() string {
	if c == nil {
		return "<nil class>"
	}
	s := fmt.Sprintf("%s v%d", c.Name, c.Version)
	if c.Superclass != nil {
		s += ", extends " + c.Superclass.String()
	}
	return s
}

// TypedGroup holds a group of typed values decoded from the stream.
type TypedGroup struct {
	Encodings [][]byte
	Values    []interface{}
}

func (g *TypedGroup) FormatLines(seen map[uintptr]bool) []string {
	if len(g.Encodings) == 1 {
		var v interface{}
		if len(g.Values) > 0 {
			v = g.Values[0]
		}
		return FormatValueWithPrefix(v, fmt.Sprintf("type %s: ", bytesRepr(g.Encodings[0])), seen)
	}
	var body []string
	for i, enc := range g.Encodings {
		var v interface{}
		if i < len(g.Values) {
			v = g.Values[i]
		}
		body = append(body, FormatValueWithPrefix(v, fmt.Sprintf("type %s: ", bytesRepr(enc)), seen)...)
	}
	return formatHeaderBody("group", body)
}
func (*TypedGroup) DetectBackreferences() bool { return false }

// TypedValue is a single-element TypedGroup (the common case).
type TypedValue struct {
	TypedGroup
}

func NewTypedValue(enc []byte, val interface{}) *TypedValue {
	return &TypedValue{TypedGroup: TypedGroup{Encodings: [][]byte{enc}, Values: []interface{}{val}}}
}

func (v *TypedValue) Encoding() []byte    { return v.Encodings[0] }
func (v *TypedValue) Value() interface{}  { return v.Values[0] }
func (v *TypedValue) FormatLines(seen map[uintptr]bool) []string {
	return FormatValueWithPrefix(v.Values[0], fmt.Sprintf("type %s: ", bytesRepr(v.Encodings[0])), seen)
}

// Array holds a C array (either []byte for byte arrays or []interface{} for others).
type Array struct {
	Elements interface{} // []byte or []interface{}
}

func (a *Array) FormatLines(seen map[uintptr]bool) []string {
	switch elems := a.Elements.(type) {
	case []byte:
		return []string{fmt.Sprintf("array, %d bytes: %s", len(elems), bytesRepr(elems))}
	case []interface{}:
		var body []string
		for _, e := range elems {
			body = append(body, FormatValue(e, seen)...)
		}
		return formatHeaderBody(fmt.Sprintf("array, %d elements", len(elems)), body)
	default:
		return []string{fmt.Sprintf("array: %v", a.Elements)}
	}
}
func (*Array) DetectBackreferences() bool { return false }

// GenericArchivedObject represents an archived object of unknown or partially-known class.
type GenericArchivedObject struct {
	Clazz       *Class
	SuperObject ArchivedObject // nil if no known superclass
	Contents    []*TypedGroup
}

func (g *GenericArchivedObject) InitFromUnarchiver(_ Unarchiver, _ *Class) error { return nil }
func (g *GenericArchivedObject) AllowsExtraData() bool                           { return true }
func (g *GenericArchivedObject) AddExtraField(f *TypedGroup) error {
	g.Contents = append(g.Contents, f)
	return nil
}
func (g *GenericArchivedObject) FormatHeader() string {
	hdr := fmt.Sprintf("object of class %s", g.Clazz)
	if g.SuperObject == nil && len(g.Contents) == 0 {
		hdr += ", no contents"
	}
	return hdr
}

func (g *GenericArchivedObject) FormatLines(seen map[uintptr]bool) []string {
	hdr := g.FormatHeader()
	var body []string
	if g.SuperObject != nil {
		body = append(body, FormatValueWithPrefix(g.SuperObject, "super object: ", seen)...)
	}
	for _, tg := range g.Contents {
		body = append(body, FormatValue(tg, seen)...)
	}
	return formatHeaderBody(hdr, body)
}

// GenericStruct represents a C struct of unknown type.
type GenericStruct struct {
	Name   []byte // nil for anonymous
	Fields []interface{}
}

func (s *GenericStruct) FormatLines(seen map[uintptr]bool) []string {
	name := "(no name)"
	if s.Name != nil {
		name = string(s.Name)
	}
	var body []string
	for _, f := range s.Fields {
		body = append(body, FormatValue(f, seen)...)
	}
	return formatHeaderBody("struct "+name, body)
}

// ---- Object table ----

type objTableEntry struct {
	RefType ObjRefType
	Value   interface{}
}

// ---- Unarchiver ----

//go:generate go run go.uber.org/mock/mockgen -destination=mock_typedstream/mock_unarchiver.go -package=mock_typedstream github.com/tagatac/typedstream-go Unarchiver

// Unarchiver decodes high-level objects from a TypedStreamReader.
type Unarchiver interface {
	Close() error
	DecodeAnyValue(expectedEncoding []byte) (interface{}, error)
	DecodeTypedValues() (*TypedGroup, error)
	DecodeValuesOfTypes(typeEncodings ...[]byte) ([]interface{}, error)
	DecodeValueOfType(typeEncoding []byte) (interface{}, error)
	DecodeArray(elemType []byte, length int) (*Array, error)
	DecodeDataObject() ([]byte, error)
	DecodePropertyList() (interface{}, error)
	DecodeAll() ([]*TypedGroup, error)
	DecodeSingleRoot() (interface{}, error)
}

// unarchiver is the concrete implementation of Unarchiver.
type unarchiver struct {
	Reader            *TypedStreamReader
	closeReader       bool
	sharedObjectTable []objTableEntry
}

// NewUnarchiverFromData creates an Unarchiver from raw typedstream bytes.
func NewUnarchiverFromData(data []byte) (Unarchiver, error) {
	r, err := NewReaderFromData(data)
	if err != nil {
		return nil, err
	}
	return &unarchiver{Reader: r, closeReader: true}, nil
}

// OpenUnarchiver opens a typedstream file for unarchiving.
func OpenUnarchiver(filename string) (Unarchiver, error) {
	r, err := OpenReader(filename)
	if err != nil {
		return nil, err
	}
	return &unarchiver{Reader: r, closeReader: true}, nil
}

// NewUnarchiver creates an Unarchiver from an existing reader.
func NewUnarchiver(r *TypedStreamReader) Unarchiver {
	return &unarchiver{Reader: r}
}

// Close closes the Unarchiver (and its underlying reader if owned).
func (u *unarchiver) Close() error {
	if u.closeReader {
		return u.Reader.Close()
	}
	return nil
}

// OpenUnarchiverFromReader creates an Unarchiver from an io.Reader.
func OpenUnarchiverFromReader(r io.Reader) (Unarchiver, error) {
	tr, err := NewReader(r)
	if err != nil {
		return nil, err
	}
	return &unarchiver{Reader: tr, closeReader: true}, nil
}

// OpenUnarchiverFromFile opens a file for unarchiving (convenience wrapper).
func OpenUnarchiverFromFile(filename string) (Unarchiver, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	tr, err := NewReader(f)
	if err != nil {
		f.Close()
		return nil, err
	}
	tr.closeStream = true
	tr.r = f
	return &unarchiver{Reader: tr, closeReader: true}, nil
}

func (u *unarchiver) lookupReference(ref ObjectReference) (interface{}, error) {
	if ref.Number < 0 || ref.Number >= len(u.sharedObjectTable) {
		return nil, fmt.Errorf("reference #%d out of range (table size %d)", ref.Number, len(u.sharedObjectTable))
	}
	entry := u.sharedObjectTable[ref.Number]
	if entry.RefType != ref.RefType {
		return nil, fmt.Errorf("reference type mismatch: expected %v, got %v", ref.RefType, entry.RefType)
	}
	return entry.Value, nil
}

// DecodeAnyValue decodes the next value from the stream using expectedEncoding as a hint.
func (u *unarchiver) DecodeAnyValue(expectedEncoding []byte) (interface{}, error) {
	ev, err := u.Reader.Next()
	if err != nil {
		return nil, err
	}

	switch v := ev.(type) {
	case nil:
		return nil, nil

	case int64, float32, float64, bool:
		return v, nil

	case []byte: // from "+" encoding
		return v, nil

	case ObjectReference:
		return u.lookupReference(v)

	case CString:
		// Literal C string: add to shared object table and return the bytes.
		u.sharedObjectTable = append(u.sharedObjectTable, objTableEntry{ObjRefTypeCString, v.Contents})
		return v.Contents, nil

	case Atom:
		return v.Contents, nil

	case Selector:
		return v.Name, nil

	case SingleClass:
		return u.readClassChain(v)

	case BeginObject:
		return u.readObject()

	case ByteArray:
		return &Array{Elements: v.Data}, nil

	case BeginArray:
		_, elemEnc, err := parseArrayEncoding(expectedEncoding)
		if err != nil {
			return nil, fmt.Errorf("bad expected encoding for array: %w", err)
		}
		elems := make([]interface{}, v.Length)
		for i := range elems {
			elems[i], err = u.DecodeAnyValue(elemEnc)
			if err != nil {
				return nil, err
			}
		}
		end, err := u.Reader.Next()
		if err != nil {
			return nil, err
		}
		if _, ok := end.(EndArray); !ok {
			return nil, fmt.Errorf("expected EndArray, got %T", end)
		}
		return &Array{Elements: elems}, nil

	case BeginStruct:
		return u.readStruct(v, expectedEncoding)

	default:
		return nil, fmt.Errorf("unexpected event type %T in DecodeAnyValue", ev)
	}
}

func (u *unarchiver) readClassChain(first SingleClass) (*Class, error) {
	singles := []SingleClass{first}
	for {
		ev, err := u.Reader.Next()
		if err != nil {
			return nil, err
		}
		switch next := ev.(type) {
		case SingleClass:
			singles = append(singles, next)
		case nil:
			return u.buildClassChain(singles, nil)
		case ObjectReference:
			ref, err := u.lookupReference(next)
			if err != nil {
				return nil, err
			}
			cls, ok := ref.(*Class)
			if !ok {
				return nil, fmt.Errorf("class reference resolved to non-Class: %T", ref)
			}
			return u.buildClassChain(singles, cls)
		default:
			return nil, fmt.Errorf("unexpected event in class chain: %T", ev)
		}
	}
}

// buildClassChain constructs Class objects from a list of SingleClass events.
// singles are in stream order (subclass first). tail is the already-resolved superclass.
func (u *unarchiver) buildClassChain(singles []SingleClass, tail *Class) (*Class, error) {
	// Build in reverse (tail = root), then assign reference numbers in forward order.
	newClasses := make([]*Class, len(singles))
	next := tail
	for i := len(singles) - 1; i >= 0; i-- {
		next = &Class{Name: singles[i].Name, Version: singles[i].Version, Superclass: next}
		newClasses[i] = next
	}
	for _, cls := range newClasses {
		u.sharedObjectTable = append(u.sharedObjectTable, objTableEntry{ObjRefTypeClass, cls})
	}
	if len(newClasses) == 0 {
		return tail, nil
	}
	return newClasses[0], nil
}

func (u *unarchiver) readObject() (interface{}, error) {
	// Reserve a slot in the table before reading the class (self-referential archives).
	placeholderIdx := len(u.sharedObjectTable)
	u.sharedObjectTable = append(u.sharedObjectTable, objTableEntry{ObjRefTypeObject, nil})

	// Read class chain.
	archClassRaw, err := u.DecodeAnyValue([]byte("#"))
	if err != nil {
		return nil, fmt.Errorf("reading object class: %w", err)
	}
	if archClassRaw == nil {
		// nil class — shouldn't happen inside BeginObject but handle gracefully.
		return nil, nil
	}
	archClass, ok := archClassRaw.(*Class)
	if !ok {
		return nil, fmt.Errorf("object class must be *Class, got %T", archClassRaw)
	}

	// Instantiate the object.
	obj, superclass := instantiateArchivedClass(archClass)
	u.sharedObjectTable[placeholderIdx] = objTableEntry{ObjRefTypeObject, obj}

	// Initialize the known part.
	var knownObj ArchivedObject
	if generic, ok := obj.(*GenericArchivedObject); ok {
		knownObj = generic.SuperObject
	} else {
		knownObj = obj
	}
	if knownObj != nil && superclass != nil {
		if err := knownObj.InitFromUnarchiver(u, superclass); err != nil {
			return nil, fmt.Errorf("InitFromUnarchiver for %s: %w", archClass.Name, err)
		}
	}

	// Read any extra data until EndObject.
	for {
		ev, err := u.Reader.Next()
		if err != nil {
			return nil, err
		}
		if _, ok := ev.(EndObject); ok {
			break
		}
		if !obj.AllowsExtraData() {
			return nil, fmt.Errorf("unexpected extra data after fully-known object %s", archClass.Name)
		}
		tg, err := u.decodeTypedValuesWithLookahead(ev)
		if err != nil {
			return nil, err
		}
		if err := obj.AddExtraField(tg); err != nil {
			return nil, err
		}
	}

	return obj, nil
}

func (u *unarchiver) readStruct(begin BeginStruct, expectedEncoding []byte) (interface{}, error) {
	factory := structFactoriesByEncoding[string(expectedEncoding)]
	_, expectedFieldEncs, _ := parseStructEncoding(expectedEncoding)

	fields := make([]interface{}, len(expectedFieldEncs))
	for i, fEnc := range expectedFieldEncs {
		val, err := u.DecodeAnyValue(fEnc)
		if err != nil {
			return nil, fmt.Errorf("struct field %d: %w", i, err)
		}
		fields[i] = val
	}

	end, err := u.Reader.Next()
	if err != nil {
		return nil, err
	}
	if _, ok := end.(EndStruct); !ok {
		return nil, fmt.Errorf("expected EndStruct, got %T", end)
	}

	if factory != nil {
		return factory(fields)
	}
	return &GenericStruct{Name: begin.Name, Fields: fields}, nil
}

// instantiateArchivedClass finds the best matching registered class for archClass.
// Returns the new object and the Class node it corresponds to (for InitFromUnarchiver).
func instantiateArchivedClass(archClass *Class) (ArchivedObject, *Class) {
	// Walk superclass chain looking for a registered type.
	found := archClass
	for found != nil {
		if factory, ok := archivedClassesByName[string(found.Name)]; ok {
			if found == archClass {
				// Exact match.
				return factory(), found
			}
			// Match at a superclass level — wrap in GenericArchivedObject.
			superObj := factory()
			return &GenericArchivedObject{Clazz: archClass, SuperObject: superObj, Contents: nil}, found
		}
		found = found.Superclass
	}
	// No known class at all.
	return &GenericArchivedObject{Clazz: archClass, SuperObject: nil, Contents: nil}, nil
}

// DecodeTypedValues decodes one typed value group.
// lookahead may be a pre-read BeginTypedValues event or nil (will read from stream).
func (u *unarchiver) decodeTypedValuesWithLookahead(lookahead interface{}) (*TypedGroup, error) {
	var begin BeginTypedValues
	if lookahead != nil {
		btv, ok := lookahead.(BeginTypedValues)
		if !ok {
			return nil, fmt.Errorf("expected BeginTypedValues, got %T", lookahead)
		}
		begin = btv
	} else {
		ev, err := u.Reader.Next()
		if err != nil {
			return nil, err
		}
		var ok bool
		begin, ok = ev.(BeginTypedValues)
		if !ok {
			return nil, fmt.Errorf("expected BeginTypedValues, got %T", ev)
		}
	}

	values := make([]interface{}, len(begin.Encodings))
	for i, enc := range begin.Encodings {
		v, err := u.DecodeAnyValue(enc)
		if err != nil {
			return nil, fmt.Errorf("value %d (type %q): %w", i, enc, err)
		}
		values[i] = v
	}

	end, err := u.Reader.Next()
	if err != nil {
		return nil, err
	}
	if _, ok := end.(EndTypedValues); !ok {
		return nil, fmt.Errorf("expected EndTypedValues, got %T", end)
	}

	if len(begin.Encodings) == 1 {
		return &TypedGroup{Encodings: begin.Encodings, Values: values}, nil
	}
	return &TypedGroup{Encodings: begin.Encodings, Values: values}, nil
}

// DecodeTypedValues decodes the next typed value group from the stream.
func (u *unarchiver) DecodeTypedValues() (*TypedGroup, error) {
	return u.decodeTypedValuesWithLookahead(nil)
}

// DecodeValuesOfTypes decodes a typed value group that must have the given encodings.
func (u *unarchiver) DecodeValuesOfTypes(typeEncodings ...[]byte) ([]interface{}, error) {
	if len(typeEncodings) == 0 {
		return nil, fmt.Errorf("DecodeValuesOfTypes: at least one type encoding required")
	}

	group, err := u.DecodeTypedValues()
	if err != nil {
		return nil, err
	}

	if !allEncodingsMatchExpected(group.Encodings, typeEncodings) {
		return nil, fmt.Errorf("type encoding mismatch: got %v, expected %v", group.Encodings, typeEncodings)
	}

	return group.Values, nil
}

// DecodeValueOfType decodes a single typed value of the given encoding.
func (u *unarchiver) DecodeValueOfType(typeEncoding []byte) (interface{}, error) {
	vals, err := u.DecodeValuesOfTypes(typeEncoding)
	if err != nil {
		return nil, err
	}
	return vals[0], nil
}

// DecodeArray decodes a C array of elemType with given length.
func (u *unarchiver) DecodeArray(elemType []byte, length int) (*Array, error) {
	enc, err := buildArrayEncoding(length, elemType)
	if err != nil {
		return nil, err
	}
	v, err := u.DecodeValueOfType(enc)
	if err != nil {
		return nil, err
	}
	a, ok := v.(*Array)
	if !ok {
		return nil, fmt.Errorf("expected *Array, got %T", v)
	}
	return a, nil
}

// DecodeDataObject decodes an NSData-style (length int32 + byte array).
func (u *unarchiver) DecodeDataObject() ([]byte, error) {
	lenVal, err := u.DecodeValueOfType([]byte("i"))
	if err != nil {
		return nil, err
	}
	length, ok := lenVal.(int64)
	if !ok {
		return nil, fmt.Errorf("data object length must be int64, got %T", lenVal)
	}
	if length < 0 {
		return nil, fmt.Errorf("data object length cannot be negative: %d", length)
	}
	arr, err := u.DecodeArray([]byte("c"), int(length))
	if err != nil {
		return nil, err
	}
	data, ok := arr.Elements.([]byte)
	if !ok {
		return nil, fmt.Errorf("data object array must be []byte, got %T", arr.Elements)
	}
	return data, nil
}

// DecodePropertyList decodes a legacy binary property list.
func (u *unarchiver) DecodePropertyList() (interface{}, error) {
	data, err := u.DecodeDataObject()
	if err != nil {
		return nil, err
	}
	return deserializeOldBinaryPlist(data)
}

// DecodeAll decodes all top-level value groups in the stream.
func (u *unarchiver) DecodeAll() ([]*TypedGroup, error) {
	var groups []*TypedGroup
	for {
		ev, err := u.Reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		tg, err := u.decodeTypedValuesWithLookahead(ev)
		if err != nil {
			return nil, err
		}
		groups = append(groups, tg)
	}
	return groups, nil
}

// UnarchiveFromData creates a fresh Unarchiver from raw bytes and decodes the single root value.
func UnarchiveFromData(data []byte) (interface{}, error) {
	u, err := NewUnarchiverFromData(data)
	if err != nil {
		return nil, err
	}
	defer u.Close()
	return u.DecodeSingleRoot()
}

// DecodeSingleRoot decodes the single root value from the stream.
func (u *unarchiver) DecodeSingleRoot() (interface{}, error) {
	groups, err := u.DecodeAll()
	if err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return nil, fmt.Errorf("archive contains no values")
	}
	if len(groups) > 1 {
		return nil, fmt.Errorf("archive contains %d root values (expected 1)", len(groups))
	}
	if len(groups[0].Values) != 1 {
		return nil, fmt.Errorf("root value group contains %d values (expected 1)", len(groups[0].Values))
	}
	return groups[0].Values[0], nil
}

// ---- Formatting helpers used by type files ----

// FormatObject formats an archived object's header.
func FormatObjectHeader(className string, superclass *Class) string {
	if superclass != nil {
		return fmt.Sprintf("object of class %s, extends %s", className, superclass)
	}
	return fmt.Sprintf("object of class %s", className)
}

// classNameOf returns the display name for any value.
func classNameOf(v interface{}) string {
	if g, ok := v.(*GenericArchivedObject); ok {
		return string(g.Clazz.Name)
	}
	if g, ok := v.(*GenericStruct); ok && g.Name != nil {
		return string(g.Name)
	}
	return fmt.Sprintf("%T", v)
}

// ---- Helper for decoding object references in type init code ----

// CastToType asserts that v is of type T, returning a useful error if not.
// expectedType is a human-readable type name for the error message.
func CastBytes(v interface{}) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	b, ok := v.([]byte)
	if !ok {
		return nil, fmt.Errorf("expected []byte, got %T", v)
	}
	return b, nil
}

// ObjectClassName returns the string class name for display.
func ObjectClassName(obj interface{}) string {
	switch v := obj.(type) {
	case *GenericArchivedObject:
		return string(v.Clazz.Name)
	case *GenericStruct:
		if v.Name != nil {
			return string(v.Name)
		}
		return "(anonymous struct)"
	default:
		return strings.TrimPrefix(fmt.Sprintf("%T", obj), "*typedstream.")
	}
}

// ---- Fix for readStruct: avoid shadowed err variable ----

func init() {
	// Ensure the readStruct method references are validated at init time.
	// (No-op; actual validation happens at compile time.)
}
