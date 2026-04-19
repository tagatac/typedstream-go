package typedstream

import (
	"bytes"
	"fmt"
	"strings"
	"time"
)

func init() {
	// Register struct types.
	nsPointEnc := buildStructEncoding([]byte("_NSPoint"), [][]byte{{'f'}, {'f'}})
	RegisterStructClass(nsPointEnc, func(fields []interface{}) (KnownStruct, error) {
		if len(fields) != 2 {
			return nil, fmt.Errorf("NSPoint: expected 2 fields, got %d", len(fields))
		}
		x, ok1 := fields[0].(float32)
		y, ok2 := fields[1].(float32)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("NSPoint: fields must be float32, got %T, %T", fields[0], fields[1])
		}
		return NSPoint{X: x, Y: y}, nil
	})

	nsSizeEnc := buildStructEncoding([]byte("_NSSize"), [][]byte{{'f'}, {'f'}})
	RegisterStructClass(nsSizeEnc, func(fields []interface{}) (KnownStruct, error) {
		if len(fields) != 2 {
			return nil, fmt.Errorf("NSSize: expected 2 fields, got %d", len(fields))
		}
		w, ok1 := fields[0].(float32)
		h, ok2 := fields[1].(float32)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("NSSize: fields must be float32")
		}
		return NSSize{Width: w, Height: h}, nil
	})

	nsRectEnc := buildStructEncoding([]byte("_NSRect"), [][]byte{nsPointEnc, nsSizeEnc})
	RegisterStructClass(nsRectEnc, func(fields []interface{}) (KnownStruct, error) {
		if len(fields) != 2 {
			return nil, fmt.Errorf("NSRect: expected 2 fields, got %d", len(fields))
		}
		origin, ok1 := fields[0].(NSPoint)
		size, ok2 := fields[1].(NSSize)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("NSRect: fields must be NSPoint and NSSize, got %T, %T", fields[0], fields[1])
		}
		return NSRect{Origin: origin, Size: size}, nil
	})

	// Register archived classes.
	RegisterArchivedClass([]byte("NSObject"), func() ArchivedObject { return &NSObject{} })
	RegisterArchivedClass([]byte("NSData"), func() ArchivedObject { return &NSData{} })
	RegisterArchivedClass([]byte("NSMutableData"), func() ArchivedObject { return &NSMutableData{} })
	RegisterArchivedClass([]byte("NSDate"), func() ArchivedObject { return &NSDate{} })
	RegisterArchivedClass([]byte("NSString"), func() ArchivedObject { return &NSString{} })
	RegisterArchivedClass([]byte("NSMutableString"), func() ArchivedObject { return &NSMutableString{} })
	RegisterArchivedClass([]byte("NSURL"), func() ArchivedObject { return &NSURL{} })
	RegisterArchivedClass([]byte("NSValue"), func() ArchivedObject { return &NSValue{} })
	RegisterArchivedClass([]byte("NSNumber"), func() ArchivedObject { return &NSNumber{} })
	RegisterArchivedClass([]byte("NSArray"), func() ArchivedObject { return &NSArray{} })
	RegisterArchivedClass([]byte("NSMutableArray"), func() ArchivedObject { return &NSMutableArray{} })
	RegisterArchivedClass([]byte("NSSet"), func() ArchivedObject { return &NSSet{} })
	RegisterArchivedClass([]byte("NSMutableSet"), func() ArchivedObject { return &NSMutableSet{} })
	RegisterArchivedClass([]byte("NSDictionary"), func() ArchivedObject { return &NSDictionary{} })
	RegisterArchivedClass([]byte("NSMutableDictionary"), func() ArchivedObject { return &NSMutableDictionary{} })
}

// ---- Struct types ----

// NSPoint is a 2D point with float32 coordinates.
type NSPoint struct {
	X, Y float32
}

func (NSPoint) StructName() []byte      { return []byte("_NSPoint") }
func (NSPoint) FieldEncodings() [][]byte { return [][]byte{{'f'}, {'f'}} }
func (p NSPoint) FormatLines(_ map[uintptr]bool) []string {
	return []string{fmt.Sprintf("{%s, %s}", formatFloat32Coord(p.X), formatFloat32Coord(p.Y))}
}
func (p NSPoint) String() string { return p.FormatLines(nil)[0] }

// NSSize is a 2D size with float32 dimensions.
type NSSize struct {
	Width, Height float32
}

func (NSSize) StructName() []byte      { return []byte("_NSSize") }
func (NSSize) FieldEncodings() [][]byte { return [][]byte{{'f'}, {'f'}} }
func (s NSSize) FormatLines(_ map[uintptr]bool) []string {
	return []string{fmt.Sprintf("{%s, %s}", formatFloat32Coord(s.Width), formatFloat32Coord(s.Height))}
}
func (s NSSize) String() string { return s.FormatLines(nil)[0] }

// NSRect is a 2D rectangle.
type NSRect struct {
	Origin NSPoint
	Size   NSSize
}

func (NSRect) StructName() []byte      { return []byte("_NSRect") }
func (NSRect) FieldEncodings() [][]byte {
	return [][]byte{
		buildStructEncoding([]byte("_NSPoint"), [][]byte{{'f'}, {'f'}}),
		buildStructEncoding([]byte("_NSSize"), [][]byte{{'f'}, {'f'}}),
	}
}
func (r NSRect) FormatLines(_ map[uintptr]bool) []string {
	return []string{fmt.Sprintf("{{%s, %s}, {%s, %s}}",
		formatFloat32Coord(r.Origin.X), formatFloat32Coord(r.Origin.Y),
		formatFloat32Coord(r.Size.Width), formatFloat32Coord(r.Size.Height))}
}
func (r NSRect) String() string { return r.FormatLines(nil)[0] }

func formatFloat32(f float32) string {
	s := fmt.Sprintf("%g", float64(f))
	if !strings.ContainsAny(s, ".e") {
		s += ".0"
	}
	return s
}

// formatFloat32Coord formats a float32 coordinate as Python's NSPoint.__str__:
// integer values as integers, others as full float64 precision.
func formatFloat32Coord(f float32) string {
	f64 := float64(f)
	if f64 == float64(int64(f64)) {
		return fmt.Sprintf("%d", int64(f64))
	}
	return fmt.Sprintf("%g", f64)
}

// ---- Archived classes ----

// NSObject is the root archived class.
type NSObject struct{}

func (o *NSObject) InitFromUnarchiver(_ Unarchiver, class *Class) error {
	if class.Superclass != nil {
		return fmt.Errorf("NSObject: expected no superclass in archive, got %v", class.Superclass)
	}
	if class.Version != 0 {
		return fmt.Errorf("NSObject: unsupported version %d", class.Version)
	}
	return nil
}
func (o *NSObject) AllowsExtraData() bool              { return false }
func (o *NSObject) AddExtraField(_ *TypedGroup) error  { return fmt.Errorf("NSObject: no extra data allowed") }
func (o *NSObject) FormatLines(_ map[uintptr]bool) []string {
	return []string{"<NSObject>"}
}
func (*NSObject) DetectBackreferences() bool { return false }

// NSData holds binary data.
type NSData struct {
	NSObject
	Data []byte
}

func (d *NSData) InitFromUnarchiver(u Unarchiver, class *Class) error {
	if !bytes.Equal(class.Superclass.Name, []byte("NSObject")) {
		return fmt.Errorf("NSData: unexpected superclass %v", class.Superclass)
	}
	if err := d.NSObject.InitFromUnarchiver(u, class.Superclass); err != nil {
		return err
	}
	if class.Version != 0 {
		return fmt.Errorf("NSData: unsupported version %d", class.Version)
	}
	var err error
	d.Data, err = u.DecodeDataObject()
	return err
}
func (d *NSData) FormatLines(_ map[uintptr]bool) []string {
	return []string{"NSData(" + bytesRepr(d.Data) + ")"}
}

// NSMutableData is a mutable variant of NSData.
type NSMutableData struct {
	NSData
}

func (d *NSMutableData) InitFromUnarchiver(u Unarchiver, class *Class) error {
	if !bytes.Equal(class.Superclass.Name, []byte("NSData")) {
		return fmt.Errorf("NSMutableData: unexpected superclass %v", class.Superclass)
	}
	if err := d.NSData.InitFromUnarchiver(u, class.Superclass); err != nil {
		return err
	}
	if class.Version != 0 {
		return fmt.Errorf("NSMutableData: unsupported version %d", class.Version)
	}
	return nil
}

// NSDate represents a date/time value.
// NSDate stores seconds since the absolute reference date: 2001-01-01 00:00:00 UTC.
type NSDate struct {
	NSObject
	AbsoluteOffset float64
}

var nsDateReferenceDate = time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)

func (d *NSDate) InitFromUnarchiver(u Unarchiver, class *Class) error {
	if !bytes.Equal(class.Superclass.Name, []byte("NSObject")) {
		return fmt.Errorf("NSDate: unexpected superclass %v", class.Superclass)
	}
	if err := d.NSObject.InitFromUnarchiver(u, class.Superclass); err != nil {
		return err
	}
	if class.Version != 0 {
		return fmt.Errorf("NSDate: unsupported version %d", class.Version)
	}
	v, err := u.DecodeValueOfType([]byte("d"))
	if err != nil {
		return err
	}
	d.AbsoluteOffset = v.(float64)
	return nil
}

// Time returns the time.Time represented by this NSDate.
func (d *NSDate) Time() time.Time {
	return nsDateReferenceDate.Add(time.Duration(d.AbsoluteOffset * float64(time.Second)))
}

func (d *NSDate) FormatLines(_ map[uintptr]bool) []string {
	return []string{fmt.Sprintf("NSDate(%v)", d.Time())}
}

// NSString holds a UTF-8 string.
type NSString struct {
	NSObject
	Value string
}

func (s *NSString) InitFromUnarchiver(u Unarchiver, class *Class) error {
	if class.Superclass == nil || !bytes.Equal(class.Superclass.Name, []byte("NSObject")) {
		return fmt.Errorf("NSString: unexpected superclass %v", class.Superclass)
	}
	if err := s.NSObject.InitFromUnarchiver(u, class.Superclass); err != nil {
		return err
	}
	if class.Version != 1 {
		return fmt.Errorf("NSString: unsupported version %d", class.Version)
	}
	raw, err := u.DecodeValueOfType([]byte("+"))
	if err != nil {
		return err
	}
	b, _ := raw.([]byte)
	s.Value = string(b)
	return nil
}
func (s *NSString) FormatLines(_ map[uintptr]bool) []string {
	return []string{"NSString(" + pyStrRepr(s.Value) + ")"}
}

// pyStrRepr formats a string as Python's repr() with single quotes.
func pyStrRepr(s string) string {
	var sb strings.Builder
	sb.WriteByte('\'')
	for _, r := range s {
		switch r {
		case '\\':
			sb.WriteString("\\\\")
		case '\'':
			sb.WriteString("\\'")
		case '\n':
			sb.WriteString("\\n")
		case '\r':
			sb.WriteString("\\r")
		case '\t':
			sb.WriteString("\\t")
		default:
			sb.WriteRune(r)
		}
	}
	sb.WriteByte('\'')
	return sb.String()
}
func (s *NSString) AllowsExtraData() bool             { return false }
func (s *NSString) AddExtraField(_ *TypedGroup) error { return fmt.Errorf("NSString: no extra data") }

// NSMutableString is a mutable variant of NSString.
type NSMutableString struct {
	NSString
}

func (s *NSMutableString) InitFromUnarchiver(u Unarchiver, class *Class) error {
	if class.Superclass == nil || !bytes.Equal(class.Superclass.Name, []byte("NSString")) {
		return fmt.Errorf("NSMutableString: unexpected superclass %v", class.Superclass)
	}
	if err := s.NSString.InitFromUnarchiver(u, class.Superclass); err != nil {
		return err
	}
	if class.Version != 1 {
		return fmt.Errorf("NSMutableString: unsupported version %d", class.Version)
	}
	return nil
}

// NSURL represents a URL, optionally relative to another URL.
type NSURL struct {
	NSObject
	RelativeTo *NSURL
	Value      string
}

func (u2 *NSURL) InitFromUnarchiver(u Unarchiver, class *Class) error {
	if class.Superclass == nil || !bytes.Equal(class.Superclass.Name, []byte("NSObject")) {
		return fmt.Errorf("NSURL: unexpected superclass %v", class.Superclass)
	}
	if err := u2.NSObject.InitFromUnarchiver(u, class.Superclass); err != nil {
		return err
	}
	if class.Version != 0 {
		return fmt.Errorf("NSURL: unsupported version %d", class.Version)
	}

	isRelRaw, err := u.DecodeValueOfType([]byte("c"))
	if err != nil {
		return err
	}
	isRel, _ := isRelRaw.(int64)

	switch isRel {
	case 0:
		u2.RelativeTo = nil
	case 1:
		parentRaw, err := u.DecodeValueOfType([]byte("@"))
		if err != nil {
			return err
		}
		if parentRaw != nil {
			parent, ok := parentRaw.(*NSURL)
			if !ok {
				// May be wrapped in GenericArchivedObject with NSURL super_object.
				if g, ok2 := parentRaw.(*GenericArchivedObject); ok2 {
					if ns, ok3 := g.SuperObject.(*NSURL); ok3 {
						u2.RelativeTo = ns
					}
				}
			} else {
				u2.RelativeTo = parent
			}
		}
	default:
		return fmt.Errorf("NSURL: unexpected is_relative value: %d", isRel)
	}

	strRaw, err := u.DecodeValueOfType([]byte("@"))
	if err != nil {
		return err
	}
	var nsStr *NSString
	switch sv := strRaw.(type) {
	case *NSString:
		nsStr = sv
	case *NSMutableString:
		nsStr = &sv.NSString
	case *GenericArchivedObject:
		if ns, ok := sv.SuperObject.(*NSString); ok {
			nsStr = ns
		} else if ns, ok := sv.SuperObject.(*NSMutableString); ok {
			nsStr = &ns.NSString
		}
	}
	if nsStr == nil {
		return fmt.Errorf("NSURL: expected NSString value, got %T", strRaw)
	}
	u2.Value = nsStr.Value
	return nil
}

func (u2 *NSURL) FormatLines(seen map[uintptr]bool) []string {
	if u2.RelativeTo == nil {
		return []string{fmt.Sprintf("NSURL(%q)", u2.Value)}
	}
	return []string{fmt.Sprintf("NSURL(relative_to=%q, %q)", u2.RelativeTo.Value, u2.Value)}
}
func (u2 *NSURL) AllowsExtraData() bool             { return false }
func (u2 *NSURL) AddExtraField(_ *TypedGroup) error { return fmt.Errorf("NSURL: no extra data") }

// NSValue wraps a typed value.
type NSValue struct {
	NSObject
	TypeEncoding []byte
	Value        interface{}
}

func (v *NSValue) InitFromUnarchiver(u Unarchiver, class *Class) error {
	if class.Superclass == nil || !bytes.Equal(class.Superclass.Name, []byte("NSObject")) {
		return fmt.Errorf("NSValue: unexpected superclass %v", class.Superclass)
	}
	if err := v.NSObject.InitFromUnarchiver(u, class.Superclass); err != nil {
		return err
	}
	if class.Version != 0 {
		return fmt.Errorf("NSValue: unsupported version %d", class.Version)
	}
	encRaw, err := u.DecodeValueOfType([]byte("*"))
	if err != nil {
		return err
	}
	v.TypeEncoding, _ = encRaw.([]byte)
	v.Value, err = u.DecodeValueOfType(v.TypeEncoding)
	return err
}
func (v *NSValue) FormatLines(seen map[uintptr]bool) []string {
	lines := FormatValue(v.Value, seen)
	return prefixLines(lines, fmt.Sprintf("NSValue, type %s: ", bytesRepr(v.TypeEncoding)), "")
}
func (*NSValue) DetectBackreferences() bool { return false }

// NSNumber is a subclass of NSValue.
type NSNumber struct {
	NSValue
}

func (n *NSNumber) InitFromUnarchiver(u Unarchiver, class *Class) error {
	if class.Superclass == nil || !bytes.Equal(class.Superclass.Name, []byte("NSValue")) {
		return fmt.Errorf("NSNumber: unexpected superclass %v", class.Superclass)
	}
	if err := n.NSValue.InitFromUnarchiver(u, class.Superclass); err != nil {
		return err
	}
	if class.Version != 0 {
		return fmt.Errorf("NSNumber: unsupported version %d", class.Version)
	}
	return nil
}
func (n *NSNumber) FormatLines(seen map[uintptr]bool) []string {
	lines := FormatValue(n.Value, seen)
	return prefixLines(lines, fmt.Sprintf("NSNumber, type %s: ", bytesRepr(n.TypeEncoding)), "")
}

// NSArray holds an ordered collection of objects.
type NSArray struct {
	NSObject
	Elements []interface{}
}

func (a *NSArray) initElements(u Unarchiver, class *Class) error {
	countRaw, err := u.DecodeValueOfType([]byte("i"))
	if err != nil {
		return err
	}
	count, _ := countRaw.(int64)
	if count < 0 {
		return fmt.Errorf("NSArray: negative element count: %d", count)
	}
	a.Elements = make([]interface{}, int(count))
	for i := range a.Elements {
		a.Elements[i], err = u.DecodeValueOfType([]byte("@"))
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *NSArray) InitFromUnarchiver(u Unarchiver, class *Class) error {
	if class.Superclass == nil || !bytes.Equal(class.Superclass.Name, []byte("NSObject")) {
		return fmt.Errorf("NSArray: unexpected superclass %v", class.Superclass)
	}
	if err := a.NSObject.InitFromUnarchiver(u, class.Superclass); err != nil {
		return err
	}
	if class.Version != 0 {
		return fmt.Errorf("NSArray: unsupported version %d", class.Version)
	}
	return a.initElements(u, class)
}
func (a *NSArray) FormatLines(seen map[uintptr]bool) []string {
	var body []string
	for _, e := range a.Elements {
		body = append(body, FormatValue(e, seen)...)
	}
	return formatHeaderBody(fmt.Sprintf("NSArray, %s", countDesc(len(a.Elements), "element")), body)
}
func (*NSArray) DetectBackreferences() bool { return false }

// NSMutableArray is a mutable NSArray.
type NSMutableArray struct {
	NSArray
}

func (a *NSMutableArray) InitFromUnarchiver(u Unarchiver, class *Class) error {
	if class.Superclass == nil || !bytes.Equal(class.Superclass.Name, []byte("NSArray")) {
		return fmt.Errorf("NSMutableArray: unexpected superclass %v", class.Superclass)
	}
	if err := a.NSArray.InitFromUnarchiver(u, class.Superclass); err != nil {
		return err
	}
	if class.Version != 0 {
		return fmt.Errorf("NSMutableArray: unsupported version %d", class.Version)
	}
	return nil
}

// NSSet holds an unordered collection of objects.
type NSSet struct {
	NSObject
	Elements []interface{}
}

func (s *NSSet) InitFromUnarchiver(u Unarchiver, class *Class) error {
	if class.Superclass == nil || !bytes.Equal(class.Superclass.Name, []byte("NSObject")) {
		return fmt.Errorf("NSSet: unexpected superclass %v", class.Superclass)
	}
	if err := s.NSObject.InitFromUnarchiver(u, class.Superclass); err != nil {
		return err
	}
	if class.Version != 0 {
		return fmt.Errorf("NSSet: unsupported version %d", class.Version)
	}
	countRaw, err := u.DecodeValueOfType([]byte("I"))
	if err != nil {
		return err
	}
	count, _ := countRaw.(int64)
	s.Elements = make([]interface{}, int(count))
	for i := range s.Elements {
		s.Elements[i], err = u.DecodeValueOfType([]byte("@"))
		if err != nil {
			return err
		}
	}
	return nil
}
func (s *NSSet) FormatLines(seen map[uintptr]bool) []string {
	var body []string
	for _, e := range s.Elements {
		body = append(body, FormatValue(e, seen)...)
	}
	return formatHeaderBody(fmt.Sprintf("NSSet, %s", countDesc(len(s.Elements), "element")), body)
}
func (*NSSet) DetectBackreferences() bool { return false }

// NSMutableSet is a mutable NSSet.
type NSMutableSet struct {
	NSSet
}

func (s *NSMutableSet) InitFromUnarchiver(u Unarchiver, class *Class) error {
	if class.Superclass == nil || !bytes.Equal(class.Superclass.Name, []byte("NSSet")) {
		return fmt.Errorf("NSMutableSet: unexpected superclass %v", class.Superclass)
	}
	if err := s.NSSet.InitFromUnarchiver(u, class.Superclass); err != nil {
		return err
	}
	if class.Version != 0 {
		return fmt.Errorf("NSMutableSet: unsupported version %d", class.Version)
	}
	return nil
}

// KeyValue holds a key-value pair from NSDictionary (preserves insertion order).
type KeyValue struct {
	Key, Value interface{}
}

// NSDictionary holds an ordered collection of key-value pairs.
type NSDictionary struct {
	NSObject
	Contents []KeyValue
}

func (d *NSDictionary) initContents(u Unarchiver) error {
	countRaw, err := u.DecodeValueOfType([]byte("i"))
	if err != nil {
		return err
	}
	count, _ := countRaw.(int64)
	if count < 0 {
		return fmt.Errorf("NSDictionary: negative count: %d", count)
	}
	d.Contents = make([]KeyValue, int(count))
	for i := range d.Contents {
		k, err := u.DecodeValueOfType([]byte("@"))
		if err != nil {
			return err
		}
		v, err := u.DecodeValueOfType([]byte("@"))
		if err != nil {
			return err
		}
		d.Contents[i] = KeyValue{Key: k, Value: v}
	}
	return nil
}

func (d *NSDictionary) InitFromUnarchiver(u Unarchiver, class *Class) error {
	if class.Superclass == nil || !bytes.Equal(class.Superclass.Name, []byte("NSObject")) {
		return fmt.Errorf("NSDictionary: unexpected superclass %v", class.Superclass)
	}
	if err := d.NSObject.InitFromUnarchiver(u, class.Superclass); err != nil {
		return err
	}
	if class.Version != 0 {
		return fmt.Errorf("NSDictionary: unsupported version %d", class.Version)
	}
	return d.initContents(u)
}
func (d *NSDictionary) FormatLines(seen map[uintptr]bool) []string {
	var body []string
	for _, kv := range d.Contents {
		keyStr := formatKey(kv.Key)
		lines := FormatValue(kv.Value, seen)
		body = append(body, prefixLines(lines, keyStr+": ", "")...)
	}
	var countDesc string
	switch len(d.Contents) {
	case 0:
		countDesc = "empty"
	case 1:
		countDesc = "1 entry"
	default:
		countDesc = fmt.Sprintf("%d entries", len(d.Contents))
	}
	return formatHeaderBody(fmt.Sprintf("NSDictionary, %s", countDesc), body)
}
func (*NSDictionary) DetectBackreferences() bool { return false }

// countDesc returns "empty", "1 unit", or "N units".
func countDesc(n int, unit string) string {
	switch n {
	case 0:
		return "empty"
	case 1:
		return "1 " + unit
	default:
		return fmt.Sprintf("%d %ss", n, unit)
	}
}

// formatKey formats a dictionary key using the value's repr (first FormatLines line),
// without updating the shared seen map (matching Python's repr(key) approach).
func formatKey(v interface{}) string {
	if f, ok := v.(Formatter); ok {
		lines := f.FormatLines(nil)
		if len(lines) > 0 {
			return lines[0]
		}
	}
	return fmt.Sprintf("%v", v)
}

// NSMutableDictionary is a mutable NSDictionary.
type NSMutableDictionary struct {
	NSDictionary
}

func (d *NSMutableDictionary) InitFromUnarchiver(u Unarchiver, class *Class) error {
	if class.Superclass == nil || !bytes.Equal(class.Superclass.Name, []byte("NSDictionary")) {
		return fmt.Errorf("NSMutableDictionary: unexpected superclass %v", class.Superclass)
	}
	if err := d.NSDictionary.InitFromUnarchiver(u, class.Superclass); err != nil {
		return err
	}
	if class.Version != 0 {
		return fmt.Errorf("NSMutableDictionary: unsupported version %d", class.Version)
	}
	return nil
}

// AllowsExtraData / AddExtraField defaults for most foundation types.
func (o *NSObject) allowsExtra() bool                     { return false }
func (d *NSData) AllowsExtraData() bool                   { return false }
func (d *NSData) AddExtraField(_ *TypedGroup) error        { return fmt.Errorf("NSData: no extra") }
func (d *NSMutableData) AllowsExtraData() bool            { return false }
func (d *NSMutableData) AddExtraField(_ *TypedGroup) error { return fmt.Errorf("NSMutableData: no extra") }
func (d *NSDate) AllowsExtraData() bool                   { return false }
func (d *NSDate) AddExtraField(_ *TypedGroup) error        { return fmt.Errorf("NSDate: no extra") }
func (v *NSValue) AllowsExtraData() bool                  { return false }
func (v *NSValue) AddExtraField(_ *TypedGroup) error       { return fmt.Errorf("NSValue: no extra") }
func (n *NSNumber) AllowsExtraData() bool                 { return false }
func (n *NSNumber) AddExtraField(_ *TypedGroup) error      { return fmt.Errorf("NSNumber: no extra") }
func (a *NSArray) AllowsExtraData() bool                  { return false }
func (a *NSArray) AddExtraField(_ *TypedGroup) error       { return fmt.Errorf("NSArray: no extra") }
func (a *NSMutableArray) AllowsExtraData() bool           { return false }
func (a *NSMutableArray) AddExtraField(_ *TypedGroup) error { return fmt.Errorf("NSMutableArray: no extra") }
func (s *NSSet) AllowsExtraData() bool                    { return false }
func (s *NSSet) AddExtraField(_ *TypedGroup) error         { return fmt.Errorf("NSSet: no extra") }
func (s *NSMutableSet) AllowsExtraData() bool             { return false }
func (s *NSMutableSet) AddExtraField(_ *TypedGroup) error  { return fmt.Errorf("NSMutableSet: no extra") }
func (d *NSDictionary) AllowsExtraData() bool             { return false }
func (d *NSDictionary) AddExtraField(_ *TypedGroup) error  { return fmt.Errorf("NSDictionary: no extra") }
func (d *NSMutableDictionary) AllowsExtraData() bool      { return false }
func (d *NSMutableDictionary) AddExtraField(_ *TypedGroup) error { return fmt.Errorf("NSMutableDictionary: no extra") }

// FormatLines for mutable variants.
func (d *NSMutableData) FormatLines(_ map[uintptr]bool) []string {
	return []string{"NSMutableData(" + bytesRepr(d.Data) + ")"}
}
func (s *NSMutableString) FormatLines(_ map[uintptr]bool) []string {
	return []string{"NSMutableString(" + pyStrRepr(s.Value) + ")"}
}
func (a *NSMutableArray) FormatLines(seen map[uintptr]bool) []string {
	var body []string
	for _, e := range a.Elements {
		body = append(body, FormatValue(e, seen)...)
	}
	return formatHeaderBody(fmt.Sprintf("NSMutableArray, %s", countDesc(len(a.Elements), "element")), body)
}
func (s *NSMutableSet) FormatLines(seen map[uintptr]bool) []string {
	var body []string
	for _, e := range s.Elements {
		body = append(body, FormatValue(e, seen)...)
	}
	return formatHeaderBody(fmt.Sprintf("NSMutableSet, %s", countDesc(len(s.Elements), "element")), body)
}
func (d *NSMutableDictionary) FormatLines(seen map[uintptr]bool) []string {
	var body []string
	for _, kv := range d.Contents {
		keyStr := formatKey(kv.Key)
		lines := FormatValue(kv.Value, seen)
		body = append(body, prefixLines(lines, keyStr+": ", "")...)
	}
	var countDesc string
	switch len(d.Contents) {
	case 0:
		countDesc = "empty"
	case 1:
		countDesc = "1 entry"
	default:
		countDesc = fmt.Sprintf("%d entries", len(d.Contents))
	}
	return formatHeaderBody(fmt.Sprintf("NSMutableDictionary, %s", countDesc), body)
}

// Suppress unused warnings.
var _ = (*NSObject).allowsExtra
