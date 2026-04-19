package typedstream

import (
	"testing"
)

func TestReadDataUnarchive(t *testing.T) {
	root, err := UnarchiveFromData(stringTestData)
	if err != nil {
		t.Fatalf("UnarchiveFromData: %v", err)
	}
	ns, ok := root.(*NSString)
	if !ok {
		t.Fatalf("root is %T, want *NSString", root)
	}
	if ns.Value != "string value" {
		t.Errorf("NSString.Value = %q, want %q", ns.Value, "string value")
	}
}

func TestReadFileUnarchive(t *testing.T) {
	files := []string{
		"testdata/Emacs.clr",
		"testdata/Empty2D macOS 10.14.gcx",
		"testdata/Empty2D macOS 13.gcx",
	}
	for _, name := range files {
		t.Run(name, func(t *testing.T) {
			u, err := OpenUnarchiver(name)
			if err != nil {
				t.Fatalf("OpenUnarchiver(%q): %v", name, err)
			}
			defer u.Close()
			if _, err := u.DecodeAll(); err != nil {
				t.Errorf("DecodeAll: %v", err)
			}
		})
	}
}

func TestUnarchiveNSURLAbsolute(t *testing.T) {
	data := []byte("\x04\x0bstreamtyped\x81\xe8\x03\x84\x01@\x84\x84\x84\x05NSURL\x00\x84\x84\x08NSObject\x00\x85\x84\x01c\x00\x92\x84\x84\x84\x08NSString\x01\x94\x84\x01+\x1ehttps://example.com/index.html\x86\x86")
	root, err := UnarchiveFromData(data)
	if err != nil {
		t.Fatalf("UnarchiveFromData: %v", err)
	}
	url, ok := root.(*NSURL)
	if !ok {
		t.Fatalf("root is %T, want *NSURL", root)
	}
	if url.RelativeTo != nil {
		t.Errorf("RelativeTo = %v, want nil", url.RelativeTo)
	}
	if url.Value != "https://example.com/index.html" {
		t.Errorf("Value = %q, want %q", url.Value, "https://example.com/index.html")
	}
}

func TestUnarchiveNSURLRelative(t *testing.T) {
	data := []byte("\x04\x0bstreamtyped\x81\xe8\x03\x84\x01@\x84\x84\x84\x05NSURL\x00\x84\x84\x08NSObject\x00\x85\x84\x01c\x01\x92\x84\x93\x95\x00\x92\x84\x84\x84\x08NSString\x01\x94\x84\x01+\x14https://example.com/\x86\x86\x92\x84\x97\x97\nindex.html\x86\x86")
	root, err := UnarchiveFromData(data)
	if err != nil {
		t.Fatalf("UnarchiveFromData: %v", err)
	}
	url, ok := root.(*NSURL)
	if !ok {
		t.Fatalf("root is %T, want *NSURL", root)
	}
	base := url.RelativeTo
	if base == nil {
		t.Fatalf("RelativeTo is nil, want *NSURL")
	}
	if base.RelativeTo != nil {
		t.Errorf("base.RelativeTo = %v, want nil", base.RelativeTo)
	}
	if base.Value != "https://example.com/" {
		t.Errorf("base.Value = %q, want %q", base.Value, "https://example.com/")
	}
	if url.Value != "index.html" {
		t.Errorf("url.Value = %q, want %q", url.Value, "index.html")
	}
}
