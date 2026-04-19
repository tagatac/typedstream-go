// Package typedstream parses Apple's NSArchiver "typedstream" binary format.
//
// Typedstreams are produced by NSArchiver and its predecessor NXTypedStream from NeXTSTEP.
// The format is deprecated since macOS 10.13 but still used by some applications
// (e.g. Stickies, Grapher, color picker).
//
// High-level usage:
//
//	obj, err := typedstream.UnarchiveFromFile("file.typedstream")
//	// obj is an ArchivedObject (known type) or *GenericArchivedObject (unknown type).
//
// Low-level event-by-event reading:
//
//	r, err := typedstream.OpenReader("file.typedstream")
//	for {
//	    event, err := r.Next()
//	    if err == io.EOF { break }
//	    ...
//	}
package typedstream

import "io"

// UnarchiveFromFile opens the file at path, decodes the single root object, and closes it.
func UnarchiveFromFile(path string) (interface{}, error) {
	u, err := OpenUnarchiver(path)
	if err != nil {
		return nil, err
	}
	defer u.Close()
	return u.DecodeSingleRoot()
}

// UnarchiveAllFromFile opens the file at path, decodes all root value groups, and closes it.
func UnarchiveAllFromFile(path string) ([]*TypedGroup, error) {
	u, err := OpenUnarchiver(path)
	if err != nil {
		return nil, err
	}
	defer u.Close()
	return u.DecodeAll()
}

// UnarchiveFromReader decodes the single root object from an io.Reader.
func UnarchiveFromReader(r io.Reader) (interface{}, error) {
	u, err := OpenUnarchiverFromReader(r)
	if err != nil {
		return nil, err
	}
	defer u.Close()
	return u.DecodeSingleRoot()
}
