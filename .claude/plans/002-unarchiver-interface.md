# Plan: Make Unarchiver an Interface

## Context

The `Unarchiver` type is currently a concrete struct. The `ArchivedObject` interface requires `InitFromUnarchiver(u *Unarchiver, class *Class) error`, meaning anyone implementing a custom `ArchivedObject` (via `RegisterArchivedClass`) must accept the concrete struct. There is no way to inject a mock — testing a custom `InitFromUnarchiver` implementation requires constructing real typedstream data.

Making `Unarchiver` an interface allows library consumers to inject test doubles into their `ArchivedObject` implementations. The library's own internal code (the struct) continues to work unchanged; only the public surface changes.

The `Reader` field is exported on the current struct but is only accessed inside `archiving.go`. No external callers use it, so it can safely disappear from the interface without a practical breaking change.

## Approach

### 1. `archiving.go` — define interface, rename struct

1. **Add `Unarchiver` interface** above the struct definition. Include all public decode methods and `Close()`:
   ```go
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
   ```

2. **Rename struct `Unarchiver` → `unarchiver`** (unexported). The `Reader` field and unexported fields remain on the struct; they are an implementation detail.

3. **Update all method receivers**: `(u *Unarchiver)` → `(u *unarchiver)` (all private and public methods on the struct).

4. **Update all constructor return types**: Change `*Unarchiver` → `Unarchiver` (the interface):
   - `NewUnarchiverFromData` → `(Unarchiver, error)`
   - `OpenUnarchiver` → `(Unarchiver, error)`
   - `NewUnarchiver` → `Unarchiver`
   - `OpenUnarchiverFromReader` → `(Unarchiver, error)`
   - `OpenUnarchiverFromFile` → `(Unarchiver, error)`

5. **Update `ArchivedObject` interface** signature:
   ```go
   InitFromUnarchiver(u Unarchiver, class *Class) error
   ```

### 2. `foundation.go` — update all `InitFromUnarchiver` signatures

Update every method receiver from `*Unarchiver` to `Unarchiver` (interface):
- `NSObject.InitFromUnarchiver`, `NSData.InitFromUnarchiver`, `NSMutableData.InitFromUnarchiver`, `NSDate.InitFromUnarchiver`, `NSString.InitFromUnarchiver`, `NSMutableString.InitFromUnarchiver`, `NSURL.InitFromUnarchiver`, `NSValue.InitFromUnarchiver`, `NSNumber.InitFromUnarchiver`, `NSArray.InitFromUnarchiver`, `NSArray.initElements`, `NSMutableArray.InitFromUnarchiver`, `NSSet.InitFromUnarchiver`, `NSMutableSet.InitFromUnarchiver`, `NSDictionary.InitFromUnarchiver`, `NSDictionary.initContents`, `NSMutableDictionary.InitFromUnarchiver`

### 3. `appkit.go` — same pattern

Update every `InitFromUnarchiver(u *Unarchiver, ...)` → `InitFromUnarchiver(u Unarchiver, ...)` (26 methods).

### 4. `nextstep.go` — same pattern

Update: `Object.InitFromUnarchiver`, `List.InitFromUnarchiver`, `HashTable.InitFromUnarchiver`, `StreamTable.InitFromUnarchiver`, `Storage.InitFromUnarchiver`.

## Breaking Change Note

Any external code holding a `*typedstream.Unarchiver` variable must drop the `*`. Constructor calls that assigned to `*Unarchiver` variables need to change to `Unarchiver`. Direct access to `u.Reader` from outside the package (none found in this repo) would break.

### 5. Add `mockgen` support

1. **Add dependency**: `go get go.uber.org/mock/mockgen@latest` (adds to `go.mod`/`go.sum`).

2. **Add `//go:generate` directive** in `archiving.go`, just above the `Unarchiver` interface:
   ```go
   //go:generate go run go.uber.org/mock/mockgen -destination=mock/mock_unarchiver.go -package=mock github.com/tagatac/typedstream-go Unarchiver
   ```

3. **Run `go generate`** to produce `mock/mock_unarchiver.go` containing `MockUnarchiver` in `package mock`.

Library consumers can then import `github.com/tagatac/typedstream-go/mock` and use `mock.NewMockUnarchiver(ctrl)` in their tests.

## Critical Files

- [archiving.go](archiving.go) — interface definition, `//go:generate` directive, struct rename, constructor return types, `ArchivedObject` interface
- [foundation.go](foundation.go) — ~17 `InitFromUnarchiver` signatures + `initElements`/`initContents` helpers
- [appkit.go](appkit.go) — ~26 `InitFromUnarchiver` signatures
- [nextstep.go](nextstep.go) — 5 `InitFromUnarchiver` signatures
- [mock/mock_unarchiver.go](mock/mock_unarchiver.go) — generated (new file)
- [cmd/typedstream/main.go](cmd/typedstream/main.go) — uses constructors + `DecodeAll()` + `Close()`; no changes expected

## Breaking Change Note

Any external code holding a `*typedstream.Unarchiver` variable must drop the `*`. Constructor calls that assigned to `*Unarchiver` variables need to change to `Unarchiver`. Direct access to `u.Reader` from outside the package (none found in this repo) would break.

## Verification

```bash
go build ./...
go generate ./...   # regenerates mock/mock_unarchiver.go
go test ./...
```

All four existing tests in `archiving_test.go` use constructors and `DecodeAll()`/`DecodeSingleRoot()` — both are in the interface — so they should pass without modification.
