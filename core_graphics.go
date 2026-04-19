package typedstream

import "fmt"

func init() {
	cgPointEnc := buildStructEncoding([]byte("CGPoint"), [][]byte{{'d'}, {'d'}})
	RegisterStructClass(cgPointEnc, func(fields []interface{}) (KnownStruct, error) {
		if len(fields) != 2 {
			return nil, fmt.Errorf("CGPoint: expected 2 fields, got %d", len(fields))
		}
		x, ok1 := fields[0].(float64)
		y, ok2 := fields[1].(float64)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("CGPoint: fields must be float64, got %T, %T", fields[0], fields[1])
		}
		return CGPoint{X: x, Y: y}, nil
	})

	cgSizeEnc := buildStructEncoding([]byte("CGSize"), [][]byte{{'d'}, {'d'}})
	RegisterStructClass(cgSizeEnc, func(fields []interface{}) (KnownStruct, error) {
		if len(fields) != 2 {
			return nil, fmt.Errorf("CGSize: expected 2 fields, got %d", len(fields))
		}
		w, ok1 := fields[0].(float64)
		h, ok2 := fields[1].(float64)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("CGSize: fields must be float64")
		}
		return CGSize{Width: w, Height: h}, nil
	})

	cgVectorEnc := buildStructEncoding([]byte("CGVector"), [][]byte{{'d'}, {'d'}})
	RegisterStructClass(cgVectorEnc, func(fields []interface{}) (KnownStruct, error) {
		if len(fields) != 2 {
			return nil, fmt.Errorf("CGVector: expected 2 fields, got %d", len(fields))
		}
		dx, ok1 := fields[0].(float64)
		dy, ok2 := fields[1].(float64)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("CGVector: fields must be float64")
		}
		return CGVector{Dx: dx, Dy: dy}, nil
	})

	cgRectEnc := buildStructEncoding([]byte("CGRect"), [][]byte{cgPointEnc, cgSizeEnc})
	RegisterStructClass(cgRectEnc, func(fields []interface{}) (KnownStruct, error) {
		if len(fields) != 2 {
			return nil, fmt.Errorf("CGRect: expected 2 fields, got %d", len(fields))
		}
		origin, ok1 := fields[0].(CGPoint)
		size, ok2 := fields[1].(CGSize)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("CGRect: fields must be CGPoint and CGSize, got %T, %T", fields[0], fields[1])
		}
		return CGRect{Origin: origin, Size: size}, nil
	})
}

// CGPoint is a 2D point with float64 coordinates.
type CGPoint struct{ X, Y float64 }

func (CGPoint) StructName() []byte       { return []byte("CGPoint") }
func (CGPoint) FieldEncodings() [][]byte { return [][]byte{{'d'}, {'d'}} }
func (p CGPoint) FormatLines(_ map[uintptr]bool) []string {
	return []string{fmt.Sprintf("{%s, %s}", formatFloat64(p.X), formatFloat64(p.Y))}
}

// CGSize is a 2D size with float64 dimensions.
type CGSize struct{ Width, Height float64 }

func (CGSize) StructName() []byte       { return []byte("CGSize") }
func (CGSize) FieldEncodings() [][]byte { return [][]byte{{'d'}, {'d'}} }
func (s CGSize) FormatLines(_ map[uintptr]bool) []string {
	return []string{fmt.Sprintf("{%s, %s}", formatFloat64(s.Width), formatFloat64(s.Height))}
}

// CGVector is a 2D vector with float64 components.
type CGVector struct{ Dx, Dy float64 }

func (CGVector) StructName() []byte       { return []byte("CGVector") }
func (CGVector) FieldEncodings() [][]byte { return [][]byte{{'d'}, {'d'}} }
func (v CGVector) FormatLines(_ map[uintptr]bool) []string {
	return []string{fmt.Sprintf("{%s, %s}", formatFloat64(v.Dx), formatFloat64(v.Dy))}
}

// CGRect is a 2D rectangle with float64 fields.
type CGRect struct {
	Origin CGPoint
	Size   CGSize
}

func (CGRect) StructName() []byte { return []byte("CGRect") }
func (CGRect) FieldEncodings() [][]byte {
	return [][]byte{
		buildStructEncoding([]byte("CGPoint"), [][]byte{{'d'}, {'d'}}),
		buildStructEncoding([]byte("CGSize"), [][]byte{{'d'}, {'d'}}),
	}
}
func (r CGRect) FormatLines(_ map[uintptr]bool) []string {
	return []string{fmt.Sprintf("{{%s, %s}, {%s, %s}}",
		formatFloat64(r.Origin.X), formatFloat64(r.Origin.Y),
		formatFloat64(r.Size.Width), formatFloat64(r.Size.Height))}
}

func formatFloat64(f float64) string {
	if float64(int(f)) == f {
		return fmt.Sprintf("%d", int(f))
	}
	return fmt.Sprintf("%g", f)
}
