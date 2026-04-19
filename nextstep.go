package typedstream

import "fmt"

func init() {
	RegisterArchivedClass([]byte("Object"), func() ArchivedObject { return &Object{} })
	RegisterArchivedClass([]byte("List"), func() ArchivedObject { return &List{} })
	RegisterArchivedClass([]byte("HashTable"), func() ArchivedObject { return &HashTable{} })
	RegisterArchivedClass([]byte("StreamTable"), func() ArchivedObject { return &StreamTable{} })
	RegisterArchivedClass([]byte("Storage"), func() ArchivedObject { return &Storage{} })
}

// Object is the root NeXTSTEP archived class.
type Object struct{}

func (o *Object) InitFromUnarchiver(_ *Unarchiver, class *Class) error {
	if class == nil {
		return nil
	}
	if class.Version != 0 {
		return fmt.Errorf("Object: unsupported version %d", class.Version)
	}
	return nil
}
func (o *Object) AllowsExtraData() bool                   { return false }
func (o *Object) AddExtraField(_ *TypedGroup) error       { return nil }
func (o *Object) FormatLines(_ map[uintptr]bool) []string { return []string{"Object"} }

// List is a NeXTSTEP ordered collection of objects.
type List struct {
	Object
	Elements []interface{}
}

func (l *List) InitFromUnarchiver(u *Unarchiver, class *Class) error {
	if err := l.Object.InitFromUnarchiver(u, class.Superclass); err != nil {
		return err
	}
	switch class.Version {
	case 0:
		vals, err := u.DecodeValuesOfTypes([]byte("i"), []byte("i"))
		if err != nil {
			return fmt.Errorf("List v0: %w", err)
		}
		count := vals[1].(int64)
		if count < 0 {
			return fmt.Errorf("List: element count cannot be negative: %d", count)
		}
		if count > 0 {
			arr, err := u.DecodeArray([]byte("@"), int(count))
			if err != nil {
				return fmt.Errorf("List v0: %w", err)
			}
			l.Elements, _ = arr.Elements.([]interface{})
		}
	case 1:
		countVal, err := u.DecodeValueOfType([]byte("i"))
		if err != nil {
			return fmt.Errorf("List v1: %w", err)
		}
		count := countVal.(int64)
		if count < 0 {
			return fmt.Errorf("List: element count cannot be negative: %d", count)
		}
		if count > 0 {
			arr, err := u.DecodeArray([]byte("@"), int(count))
			if err != nil {
				return fmt.Errorf("List v1: %w", err)
			}
			l.Elements, _ = arr.Elements.([]interface{})
		}
	default:
		return fmt.Errorf("List: unsupported version %d", class.Version)
	}
	return nil
}
func (l *List) AllowsExtraData() bool               { return false }
func (l *List) AddExtraField(_ *TypedGroup) error   { return nil }
func (*List) DetectBackreferences() bool             { return false }
func (l *List) FormatLines(seen map[uintptr]bool) []string {
	header := "List, " + nextstepCountDesc(len(l.Elements), "element")
	var body []string
	for _, elem := range l.Elements {
		body = append(body, FormatValue(elem, seen)...)
	}
	return formatHeaderBody(header, body)
}

// HashTableEntry is a single key/value pair in a HashTable.
type HashTableEntry struct{ Key, Value interface{} }

// HashTable is a NeXTSTEP key/value store.
type HashTable struct {
	Object
	KeyTypeEncoding   []byte
	ValueTypeEncoding []byte
	Contents          []HashTableEntry
}

func (h *HashTable) InitFromUnarchiver(u *Unarchiver, class *Class) error {
	if err := h.Object.InitFromUnarchiver(u, class.Superclass); err != nil {
		return err
	}
	var strEnc []byte
	switch class.Version {
	case 0:
		strEnc = []byte("*")
	case 1:
		strEnc = []byte("%")
	default:
		return fmt.Errorf("HashTable: unsupported version %d", class.Version)
	}
	vals, err := u.DecodeValuesOfTypes([]byte("i"), strEnc, strEnc)
	if err != nil {
		return fmt.Errorf("HashTable: %w", err)
	}
	count := vals[0].(int64)
	if count < 0 {
		return fmt.Errorf("HashTable: element count cannot be negative: %d", count)
	}
	h.KeyTypeEncoding, _ = vals[1].([]byte)
	h.ValueTypeEncoding, _ = vals[2].([]byte)
	h.Contents = make([]HashTableEntry, int(count))
	for i := range h.Contents {
		key, err := u.DecodeValueOfType(h.KeyTypeEncoding)
		if err != nil {
			return fmt.Errorf("HashTable key %d: %w", i, err)
		}
		value, err := u.DecodeValueOfType(h.ValueTypeEncoding)
		if err != nil {
			return fmt.Errorf("HashTable value %d: %w", i, err)
		}
		h.Contents[i] = HashTableEntry{key, value}
	}
	return nil
}
func (h *HashTable) AllowsExtraData() bool               { return false }
func (h *HashTable) AddExtraField(_ *TypedGroup) error   { return nil }
func (*HashTable) DetectBackreferences() bool             { return false }
func (h *HashTable) FormatLines(seen map[uintptr]bool) []string {
	n := len(h.Contents)
	var countD string
	switch n {
	case 0:
		countD = "empty"
	case 1:
		countD = "1 entry"
	default:
		countD = fmt.Sprintf("%d entries", n)
	}
	header := fmt.Sprintf("HashTable, key/value types %s/%s, %s",
		bytesRepr(h.KeyTypeEncoding), bytesRepr(h.ValueTypeEncoding), countD)
	var body []string
	for _, entry := range h.Contents {
		prefix := fmt.Sprintf("%v: ", entry.Key)
		body = append(body, FormatValueWithPrefix(entry.Value, prefix, seen)...)
	}
	return formatHeaderBody(header, body)
}

// StreamTable is a HashTable whose values are raw typedstream blobs, unarchived lazily.
type StreamTable struct {
	HashTable
	UnarchivedContents []HashTableEntry
}

func (s *StreamTable) InitFromUnarchiver(u *Unarchiver, class *Class) error {
	if err := s.HashTable.InitFromUnarchiver(u, class.Superclass); err != nil {
		return err
	}
	if class.Version != 1 {
		return fmt.Errorf("StreamTable: unsupported version %d", class.Version)
	}
	if string(s.ValueTypeEncoding) != "!" {
		return fmt.Errorf("StreamTable values must be ignored, not %q", s.ValueTypeEncoding)
	}
	s.UnarchivedContents = make([]HashTableEntry, len(s.Contents))
	for i, entry := range s.Contents {
		key, err := u.DecodeValueOfType(s.KeyTypeEncoding)
		if err != nil {
			return fmt.Errorf("StreamTable re-read key %d: %w", i, err)
		}
		data, err := u.DecodeDataObject()
		if err != nil {
			return fmt.Errorf("StreamTable data object %d: %w", i, err)
		}
		val, err := UnarchiveFromData(data)
		if err != nil {
			return fmt.Errorf("StreamTable unarchive value %d: %w", i, err)
		}
		s.UnarchivedContents[i] = HashTableEntry{key, val}
		_ = entry
	}
	return nil
}
func (s *StreamTable) AllowsExtraData() bool               { return false }
func (s *StreamTable) AddExtraField(_ *TypedGroup) error   { return nil }
func (s *StreamTable) FormatLines(seen map[uintptr]bool) []string {
	n := len(s.UnarchivedContents)
	var countD string
	switch n {
	case 0:
		countD = "empty"
	case 1:
		countD = "1 entry"
	default:
		countD = fmt.Sprintf("%d entries", n)
	}
	header := fmt.Sprintf("StreamTable, %s", countD)
	var body []string
	for _, entry := range s.UnarchivedContents {
		prefix := fmt.Sprintf("%v: ", entry.Key)
		body = append(body, FormatValueWithPrefix(entry.Value, prefix, seen)...)
	}
	return formatHeaderBody(header, body)
}

// Storage is a NeXTSTEP typed array.
type Storage struct {
	Object
	ElementTypeEncoding []byte
	ElementSize         int64
	Elements            []interface{}
}

func (s *Storage) InitFromUnarchiver(u *Unarchiver, class *Class) error {
	if err := s.Object.InitFromUnarchiver(u, class.Superclass); err != nil {
		return err
	}
	switch class.Version {
	case 0:
		vals, err := u.DecodeValuesOfTypes([]byte("*"), []byte("i"), []byte("i"), []byte("i"))
		if err != nil {
			return fmt.Errorf("Storage v0: %w", err)
		}
		s.ElementTypeEncoding, _ = vals[0].([]byte)
		s.ElementSize = vals[1].(int64)
		count := vals[3].(int64)
		if count < 0 {
			return fmt.Errorf("Storage: element count cannot be negative: %d", count)
		}
		arr, err := u.DecodeArray(s.ElementTypeEncoding, int(count))
		if err != nil {
			return fmt.Errorf("Storage v0: %w", err)
		}
		s.Elements, _ = arr.Elements.([]interface{})
	case 1:
		vals, err := u.DecodeValuesOfTypes([]byte("%"), []byte("i"), []byte("i"))
		if err != nil {
			return fmt.Errorf("Storage v1: %w", err)
		}
		s.ElementTypeEncoding, _ = vals[0].([]byte)
		s.ElementSize = vals[1].(int64)
		count := vals[2].(int64)
		if count < 0 {
			return fmt.Errorf("Storage: element count cannot be negative: %d", count)
		}
		if count > 0 {
			arr, err := u.DecodeArray(s.ElementTypeEncoding, int(count))
			if err != nil {
				return fmt.Errorf("Storage v1: %w", err)
			}
			s.Elements, _ = arr.Elements.([]interface{})
		}
	default:
		return fmt.Errorf("Storage: unsupported version %d", class.Version)
	}
	return nil
}
func (s *Storage) AllowsExtraData() bool               { return false }
func (s *Storage) AddExtraField(_ *TypedGroup) error   { return nil }
func (*Storage) DetectBackreferences() bool             { return false }
func (s *Storage) FormatLines(seen map[uintptr]bool) []string {
	n := len(s.Elements)
	var countD string
	switch n {
	case 0:
		countD = "empty"
	case 1:
		countD = "1 element"
	default:
		countD = fmt.Sprintf("%d elements", n)
	}
	header := fmt.Sprintf("Storage, element type %s (%d bytes each), %s",
		bytesRepr(s.ElementTypeEncoding), s.ElementSize, countD)
	var body []string
	for _, elem := range s.Elements {
		body = append(body, FormatValue(elem, seen)...)
	}
	return formatHeaderBody(header, body)
}

// nextstepCountDesc returns "empty", "1 <unit>", or "N <unit>s".
func nextstepCountDesc(n int, unit string) string {
	switch n {
	case 0:
		return "empty"
	case 1:
		return "1 " + unit
	default:
		return fmt.Sprintf("%d %ss", n, unit)
	}
}
