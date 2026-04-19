package typedstream

import (
	"fmt"
	"reflect"
	"strings"
)

// Formatter is implemented by types that want a custom multiline string representation.
type Formatter interface {
	FormatLines(seen map[uintptr]bool) []string
}

// BackreferenceDetector is an optional interface for Formatter types.
// If DetectBackreferences returns false, the object is rendered fresh on each
// encounter (no backreference annotation). Circular reference detection via the
// two-state seen map still applies even when false.
// Default (interface not implemented) = true.
type BackreferenceDetector interface {
	DetectBackreferences() bool
}

// prefixLines prepends first to the first line and rest to all subsequent lines.
func prefixLines(lines []string, first, rest string) []string {
	if len(lines) == 0 {
		if first != "" {
			return []string{first}
		}
		return nil
	}
	result := make([]string, len(lines))
	result[0] = first + lines[0]
	for i := 1; i < len(lines); i++ {
		result[i] = rest + lines[i]
	}
	return result
}

// formatHeaderBody renders a header with an optional indented body.
// If body is empty, returns just [header]. Otherwise returns [header+":", "\t"+body[0], ...].
func formatHeaderBody(header string, body []string) []string {
	if len(body) == 0 {
		return []string{header}
	}
	result := make([]string, 1+len(body))
	result[0] = header + ":"
	for i, line := range body {
		result[i+1] = "\t" + line
	}
	return result
}

// ptrID returns a uintptr identity for a value, if it's a pointer or interface.
// Returns 0 for non-pointer values.
func ptrID(v interface{}) uintptr {
	if v == nil {
		return 0
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func:
		return rv.Pointer()
	default:
		return 0
	}
}

func detectBackreferences(v interface{}) bool {
	if bd, ok := v.(BackreferenceDetector); ok {
		return bd.DetectBackreferences()
	}
	return true
}

// HeaderFormatter is an optional interface for types that can cheaply provide
// a one-line header without rendering their full body (used for circular/backreference display).
type HeaderFormatter interface {
	FormatHeader() string
}

// getHeader returns a short one-line description of f for circular/backreference annotations.
func getHeader(f Formatter) string {
	if h, ok := f.(HeaderFormatter); ok {
		return h.FormatHeader()
	}
	return fmt.Sprintf("%T", f)
}

// FormatValue converts any value to a slice of lines for display.
// seen tracks objects already rendered (two-state: false=rendering, true=rendered).
// Pass nil for the initial call; it will be created automatically.
func FormatValue(v interface{}, seen map[uintptr]bool) []string {
	if seen == nil {
		seen = make(map[uintptr]bool)
	}
	return formatValueInternal(v, seen)
}

func formatValueInternal(v interface{}, seen map[uintptr]bool) []string {
	if v == nil {
		return []string{"None"}
	}
	id := ptrID(v)

	if f, ok := v.(Formatter); ok {
		if id != 0 {
			if inSeen, present := seen[id]; present {
				if !inSeen {
					// currently being rendered → circular reference
					header := getHeader(f)
					return []string{header + " (circular reference)"}
				}
				// fully rendered → backreference (only if detect_backreferences=true)
				if detectBackreferences(v) {
					header := getHeader(f)
					return []string{header + " (backreference)"}
				}
				// detect_backreferences=false: remove stale true marker and re-render
				delete(seen, id)
			}
			seen[id] = false // mark as currently rendering
		}

		lines := f.FormatLines(seen)

		if id != 0 {
			if detectBackreferences(v) {
				seen[id] = true // mark as fully rendered
			} else {
				delete(seen, id) // don't persist — allow fresh re-render next time
			}
		}

		return lines
	}

	// Primitive values: format with Python-compatible representations.
	switch val := v.(type) {
	case int64:
		return []string{fmt.Sprintf("%d", val)}
	case float32:
		s := fmt.Sprintf("%g", float64(val))
		if !strings.ContainsAny(s, ".e") {
			s += ".0"
		}
		return []string{s}
	case float64:
		s := fmt.Sprintf("%g", val)
		if !strings.ContainsAny(s, ".e") {
			s += ".0"
		}
		return []string{s}
	case bool:
		if val {
			return []string{"True"}
		}
		return []string{"False"}
	case []byte:
		return []string{bytesRepr(val)}
	default:
		return strings.Split(fmt.Sprintf("%v", v), "\n")
	}
}

// bytesRepr formats a byte slice as Python's repr(b'...').
func bytesRepr(b []byte) string {
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

// FormatValueWithPrefix converts a value to lines, with a prefix on the first line.
// Continuation lines have no additional prefix (matching Python's prefix_lines with rest="").
func FormatValueWithPrefix(v interface{}, prefix string, seen map[uintptr]bool) []string {
	lines := FormatValue(v, seen)
	return prefixLines(lines, prefix, "")
}
