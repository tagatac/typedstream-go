// Command typedstream reads and decodes Apple NSArchiver typedstream files.
//
// Usage:
//
//	typedstream read <file>    — low-level event dump
//	typedstream decode <file>  — high-level decoded output
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	ts "github.com/tagatac/go-typedstream"
)

func main() {
	if len(os.Args) < 3 {
		usage()
		os.Exit(2)
	}
	sub := os.Args[1]
	file := os.Args[2]

	switch sub {
	case "read":
		if err := doRead(file); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "decode":
		if err := doDecode(file); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %q\n", sub)
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: typedstream <read|decode> <file>")
}

func openReader(file string) (*ts.TypedStreamReader, error) {
	if file == "-" {
		return ts.NewReader(os.Stdin)
	}
	return ts.OpenReader(file)
}

func doRead(file string) error {
	r, err := openReader(file)
	if err != nil {
		return err
	}
	defer r.Close()

	byteOrderStr := strings.TrimSuffix(strings.ToLower(fmt.Sprint(r.ByteOrder)), "endian")
	fmt.Printf("streamer version %d, byte order %s, system version %d\n",
		r.StreamerVersion, byteOrderStr, r.SystemVersion)
	fmt.Println()

	indent := 0
	nextObjectNumber := 0
	for {
		ev, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Decrease indent before End* events.
		switch ev.(type) {
		case ts.EndTypedValues, ts.EndObject, ts.EndArray, ts.EndStruct:
			indent--
		}

		rep := strings.Repeat("\t", indent) + formatEvent(ev)

		// Append object number for shared-reference events.
		switch ev.(type) {
		case ts.CString, ts.SingleClass, ts.BeginObject:
			rep += fmt.Sprintf(" (#%d)", nextObjectNumber)
			nextObjectNumber++
		}

		fmt.Println(rep)

		// Increase indent after Begin* events.
		switch ev.(type) {
		case ts.BeginTypedValues, ts.BeginObject, ts.BeginArray, ts.BeginStruct:
			indent++
		}
	}
	return nil
}

func doDecode(file string) error {
	u, err := openUnarchiver(file)
	if err != nil {
		return err
	}
	defer u.Close()

	groups, err := u.DecodeAll()
	if err != nil {
		return err
	}
	for _, group := range groups {
		// Fresh seen map per group, matching Python's per-object context.
		for _, line := range formatTypedGroup(group, nil) {
			fmt.Println(line)
		}
	}
	return nil
}

// formatTypedGroup formats a TypedGroup as Python's pytypedstream decode would.
// Single-element groups: "type b'ENC': VALUE"
// Multi-element groups:  "group:\n\ttype b'ENC': VALUE\n..."
func formatTypedGroup(group *ts.TypedGroup, seen map[uintptr]bool) []string {
	if len(group.Encodings) == 1 {
		prefix := fmt.Sprintf("type %s: ", pyBytesRepr(group.Encodings[0]))
		var v interface{}
		if len(group.Values) > 0 {
			v = group.Values[0]
		}
		return prefixWithIndent(formatDecodeValue(v, seen), prefix, "")
	}
	var body []string
	for i, enc := range group.Encodings {
		var v interface{}
		if i < len(group.Values) {
			v = group.Values[i]
		}
		prefix := fmt.Sprintf("type %s: ", pyBytesRepr(enc))
		body = append(body, prefixWithIndent(formatDecodeValue(v, seen), prefix, "")...)
	}
	result := []string{"group:"}
	for _, line := range body {
		result = append(result, "\t"+line)
	}
	return result
}

// formatDecodeValue formats a value for the decode subcommand output.
func formatDecodeValue(v interface{}, seen map[uintptr]bool) []string {
	if v == nil {
		return []string{"None"}
	}
	switch val := v.(type) {
	case int64:
		return []string{fmt.Sprintf("%d", val)}
	case float32:
		return []string{pyFloat32Str(val)}
	case float64:
		return []string{pyFloatRepr(val)}
	case bool:
		if val {
			return []string{"True"}
		}
		return []string{"False"}
	case []byte:
		return []string{pyBytesRepr(val)}
	default:
		return ts.FormatValue(v, seen)
	}
}

// prefixWithIndent prefixes the first line with first, and subsequent lines with rest.
func prefixWithIndent(lines []string, first, rest string) []string {
	if len(lines) == 0 {
		return []string{first}
	}
	result := make([]string, len(lines))
	result[0] = first + lines[0]
	for i := 1; i < len(lines); i++ {
		result[i] = rest + lines[i]
	}
	return result
}

// pyFloat32Str formats a float32 value to match Python's str(float) output.
func pyFloat32Str(f float32) string {
	s := fmt.Sprintf("%g", float64(f))
	if !strings.ContainsAny(s, ".e") {
		s += ".0"
	}
	return s
}

func openUnarchiver(file string) (ts.Unarchiver, error) {
	if file == "-" {
		return ts.OpenUnarchiverFromReader(os.Stdin)
	}
	return ts.OpenUnarchiver(file)
}

// formatEvent converts a stream event to its string representation,
// matching Python's pytypedstream read output format.
func formatEvent(ev interface{}) string {
	if ev == nil {
		return "None"
	}
	switch v := ev.(type) {
	case ts.BeginTypedValues:
		return fmt.Sprintf("begin typed values (types %s)", pyBytesList(v.Encodings))
	case ts.EndTypedValues:
		return "end typed values"
	case ts.ObjectReference:
		refType := "object"
		switch v.RefType {
		case ts.ObjRefTypeCString:
			refType = "C string"
		case ts.ObjRefTypeClass:
			refType = "class"
		case ts.ObjRefTypeObject:
			refType = "object"
		}
		return fmt.Sprintf("<reference to %s #%d>", refType, v.Number)
	case ts.CString:
		return "C string: " + pyBytesRepr(v.Contents)
	case ts.Atom:
		return "atom: " + pyBytesReprNullable(v.Contents)
	case ts.Selector:
		return "selector: " + pyBytesReprNullable(v.Name)
	case ts.SingleClass:
		name := string(v.Name)
		return fmt.Sprintf("class %s v%d", name, v.Version)
	case ts.BeginObject:
		return "begin literal object"
	case ts.EndObject:
		return "end literal object"
	case ts.ByteArray:
		return fmt.Sprintf("byte array (element type %s): %s",
			pyBytesRepr(v.ElementEncoding), pyBytesRepr(v.Data))
	case ts.BeginArray:
		return fmt.Sprintf("begin array (element type %s, length %d)",
			pyBytesRepr(v.ElementEncoding), v.Length)
	case ts.EndArray:
		return "end array"
	case ts.BeginStruct:
		name := "(no name)"
		if v.Name != nil {
			name = string(v.Name)
		}
		return fmt.Sprintf("begin struct %s (field types %s)", name, pyBytesList(v.FieldEncodings))
	case ts.EndStruct:
		return "end struct"
	case int64:
		return fmt.Sprintf("%d", v)
	case float32:
		return pyFloatRepr(float64(v))
	case float64:
		return pyFloatRepr(v)
	case bool:
		if v {
			return "True"
		}
		return "False"
	case []byte:
		return pyBytesRepr(v)
	default:
		return fmt.Sprintf("%v", ev)
	}
}

// pyBytesRepr formats a byte slice as Python's repr(b'...').
func pyBytesRepr(b []byte) string {
	var sb strings.Builder
	sb.WriteString("b'")
	for _, c := range b {
		switch c {
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
			if c >= 32 && c < 127 {
				sb.WriteByte(c)
			} else {
				fmt.Fprintf(&sb, "\\x%02x", c)
			}
		}
	}
	sb.WriteString("'")
	return sb.String()
}

func pyBytesReprNullable(b []byte) string {
	if b == nil {
		return "None"
	}
	return pyBytesRepr(b)
}

// pyBytesList formats a slice of byte slices as Python's repr([b'@', ...]).
func pyBytesList(encs [][]byte) string {
	parts := make([]string, len(encs))
	for i, enc := range encs {
		parts[i] = pyBytesRepr(enc)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// pyFloatRepr formats a float to match Python's str(float) representation.
func pyFloatRepr(f float64) string {
	s := fmt.Sprintf("%g", f)
	// Python always includes a decimal point for floats.
	if !strings.ContainsAny(s, ".e") {
		s += ".0"
	}
	return s
}
