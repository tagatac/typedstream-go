# Plan: Convert python-typedstream to Go library

## Context

`python-typedstream` parses Apple's NSArchiver "typedstream" binary format — a deprecated serialization format still used by macOS Stickies, Grapher, and some color-picker files. The Python library provides both a Go-importable library and a CLI (`pytypedstream read/decode`). The goal is a full-functionality Go port that can be imported as a library and run as a CLI tool.

**New repo:** `/Users/tag/Code/typedstream-go`  
**Module path:** `github.com/tagatac/typedstream-go`  
**Go version:** 1.21+  
**Dependencies:** stdlib only

---

## File Structure

```
/Users/tag/Code/typedstream-go/
├── go.mod
├── encodings.go          # type encoding parser
├── stream.go             # low-level binary reader + event types
├── archiving.go          # unarchiver + class/struct registry
├── plist.go              # old binary plist (NeXTSTEP format)
├── repr.go               # multiline pretty printer
├── foundation.go         # NSString, NSArray, NSDictionary, NSPoint, etc.
├── core_graphics.go      # CGPoint, CGRect, CGSize, CGVector
├── appkit.go             # ~40 AppKit archived classes
├── nextstep.go           # Object, List, HashTable, StreamTable, etc.
├── typedstream.go        # package doc + public convenience funcs
├── encodings_test.go
├── stream_test.go
├── archiving_test.go
├── testdata/             # symlinks or copies of tests/data/ from Python repo
│   ├── Emacs.clr
│   ├── Empty2D macOS 10.14.gcx
│   └── Empty2D macOS 13.gcx
└── cmd/
    └── typedstream/
        └── main.go       # CLI: `typedstream read/decode`
```

---

## Key Architecture Decisions

### 1. Event types (`stream.go`)
All stream events are plain Go structs; the event type is `interface{}`. Primitive values (`int64`, `float32`, `float64`, `bool`, `[]byte`) are returned directly as `interface{}`. Callers use type switches.

```go
type BeginTypedValues struct{ Encodings [][]byte }
type EndTypedValues    struct{}
type ObjectReference  struct{ RefType ObjRefType; Number int }
type CString          struct{ Contents []byte }
type Atom             struct{ Contents []byte }  // nil = nil atom
type Selector         struct{ Name []byte }
type SingleClass      struct{ Name []byte; Version int }
type BeginObject      struct{}
type EndObject        struct{}
type ByteArray        struct{ ElementEncoding, Data []byte }
type BeginArray       struct{ ElementEncoding []byte; Length int }
type EndArray         struct{}
type BeginStruct      struct{ Name []byte; FieldEncodings [][]byte }
type EndStruct        struct{}
```

`Next() (interface{}, error)` returns `io.EOF` at end-of-stream and `nil` (typed as `interface{}(nil)`) as a valid event for nil objects/strings.

**Stream reader uses a goroutine + channel** to faithfully translate Python's `yield from` generator recursion. A `done chan struct{}` allows early cancellation on `Close()`.

### 2. `ArchivedObject` interface (`archiving.go`)
```go
type ArchivedObject interface {
    InitFromUnarchiver(u *Unarchiver, class *Class) error
    AllowsExtraData() bool
    AddExtraField(field *TypedGroup) error
    FormatLines(seen map[uintptr]bool) []string
}
```

**Superclass chain:** Unlike Python's metaclass magic, each Go type's `InitFromUnarchiver` explicitly calls the embedded supertype's `InitFromUnarchiver` first (with `class.Superclass`), then reads its own versioned data. Mechanical but explicit.

```go
func (s *NSString) InitFromUnarchiver(u *Unarchiver, class *Class) error {
    if err := s.NSObject.InitFromUnarchiver(u, class.Superclass); err != nil { return err }
    if class.Version != 1 { return fmt.Errorf("NSString: unsupported version %d", class.Version) }
    raw, err := u.DecodeValueOfType([]byte("+"))
    if err != nil { return err }
    s.Value = string(raw.([]byte))
    return nil
}
```

### 3. Class and struct registries
```go
// Archived classes: name -> factory
var archivedClassesByName = map[string]func() ArchivedObject{}

// Structs: full encoding string (e.g. "{_NSPoint=ff}") -> factory
var structFactoriesByEncoding = map[string]func(fields []interface{}) (KnownStruct, error){}
```

Each type file registers via `init()`. The main `typedstream.go` imports all type files (blank imports) to trigger their `init()` calls.

### 4. Decoded value types
`DecodeAnyValue(expectedEncoding []byte) (interface{}, error)` returns:
- `nil` for nil objects/strings/classes
- `bool`, `int64`, `float32`, `float64` for primitives
- `[]byte` for `+` (raw bytes), `*` (C string), `%` (atom), `:` (selector)
- `*Class` for class metadata
- `*Array` (with `Elements interface{}` = `[]byte` or `[]interface{}`) for arrays
- `ArchivedObject` implementors for known classes, `*GenericArchivedObject` for unknown
- `KnownStruct` implementors for known structs, `*GenericStruct` for unknown

### 5. Pretty printing (`repr.go`)
Replaces Python's `contextvars`-based circular-reference tracking with explicit `seen map[uintptr]bool` passed through `FormatLines`. Two maps: `seen` (already rendered) and `rendering` (currently on call stack) to distinguish backreferences from circular references.

```go
type Formatter interface {
    FormatLines(seen map[uintptr]bool) []string
}
func FormatValue(v interface{}, seen map[uintptr]bool) []string
```

### 6. Old binary plist (`plist.go`)
Direct port of `old_binary_plist.py`. Uses an `io.ReadSeeker` for position-tracking. Returns: `nil`, `[]byte`, `string`, `[]interface{}`, `map[string]interface{}`. Includes the full 254-char NeXTSTEP 8-bit character map as a `[256]rune` array.

### 7. Struct types
`NSPoint`, `NSSize`, `NSRect` (float32) and `CGPoint`, `CGSize`, `CGRect`, `CGVector` (float64) implement `KnownStruct`. Each registers its factory in `init()` using `buildStructEncoding`.

### 8. `NSDictionary` ordering
Use `[]KeyValue` (ordered slice) to preserve insertion order, since Go `map` doesn't guarantee order and the archive format preserves it.

### 9. CLI (`cmd/typedstream/main.go`)
Two subcommands using stdlib `os.Args` dispatch (no external flag library needed):
- `typedstream read <file>` — low-level event dump with indentation and `(#N)` object numbers
- `typedstream decode <file>` — high-level decoded output via `FormatValue`

---

## Implementation Order

1. **`encodings.go`** + `encodings_test.go` — no deps, test immediately
2. **`stream.go`** + `stream_test.go` — port `TestReadDataStream`, `TestReadFileStream`
3. **`plist.go`** — no deps on other library files
4. **`repr.go`** — can be stubbed initially
5. **`archiving.go`** + `archiving_test.go` — enables integration tests
6. **`foundation.go`** — enables `TestReadDataUnarchive`, URL tests, file unarchive tests
7. **`core_graphics.go`** — short, same patterns as foundation
8. **`nextstep.go`**
9. **`appkit.go`** — largest (~1200 lines Python), do last
10. **`typedstream.go`** — convenience wrappers
11. **`cmd/typedstream/main.go`** — CLI

---

## Critical Source Files (Python)
- [stream.py](src/typedstream/stream.py) — binary format, tag constants, read methods
- [archiving.py](src/typedstream/archiving.py) — unarchiver dispatch, class instantiation, object table
- [encodings.py](src/typedstream/encodings.py) — type encoding parser
- [types/foundation.py](src/typedstream/types/foundation.py) — Foundation types
- [types/appkit.py](src/typedstream/types/appkit.py) — AppKit types (~1201 lines)
- [types/core_graphics.py](src/typedstream/types/core_graphics.py)
- [types/nextstep.py](src/typedstream/types/nextstep.py)
- [old_binary_plist.py](src/typedstream/old_binary_plist.py)
- [tests/test_typedstream.py](tests/test_typedstream.py) — test cases to port

---

## Verification

- `go test ./...` passes all ported tests
- `go test ./... -run TestReadFile` reads all 3 test data files without error
- `typedstream read tests/data/Emacs.clr` produces same output as Python `pytypedstream read`
- `typedstream decode tests/data/Emacs.clr` produces same output as Python `pytypedstream decode`
- `go vet ./...` and `go build ./...` produce no errors
