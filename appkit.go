package typedstream

import (
	"fmt"
	"sort"
	"strings"
)

func init() {
	RegisterArchivedClass([]byte("NSBezierPath"), func() ArchivedObject { return &NSBezierPath{} })
	RegisterArchivedClass([]byte("NSClassSwapper"), func() ArchivedObject { return &NSClassSwapper{} })
	RegisterArchivedClass([]byte("NSColor"), func() ArchivedObject { return &NSColor{} })
	RegisterArchivedClass([]byte("NSCustomObject"), func() ArchivedObject { return &NSCustomObject{} })
	RegisterArchivedClass([]byte("NSCustomResource"), func() ArchivedObject { return &NSCustomResource{} })
	RegisterArchivedClass([]byte("NSFont"), func() ArchivedObject { return &NSFont{} })
	RegisterArchivedClass([]byte("NSIBObjectData"), func() ArchivedObject { return &NSIBObjectData{} })
	RegisterArchivedClass([]byte("NSIBHelpConnector"), func() ArchivedObject { return &NSIBHelpConnector{} })
	RegisterArchivedClass([]byte("NSNibConnector"), func() ArchivedObject { return &NSNibConnector{} })
	RegisterArchivedClass([]byte("NSNibControlConnector"), func() ArchivedObject { return &NSNibControlConnector{} })
	RegisterArchivedClass([]byte("NSNibOutletConnector"), func() ArchivedObject { return &NSNibOutletConnector{} })
	RegisterArchivedClass([]byte("NSMenuItem"), func() ArchivedObject { return &NSMenuItem{} })
	RegisterArchivedClass([]byte("NSMenu"), func() ArchivedObject { return &NSMenu{} })
	RegisterArchivedClass([]byte("NSCell"), func() ArchivedObject { return &NSCell{} })
	RegisterArchivedClass([]byte("NSImageCell"), func() ArchivedObject { return &NSImageCell{} })
	RegisterArchivedClass([]byte("NSActionCell"), func() ArchivedObject { return &NSActionCell{} })
	RegisterArchivedClass([]byte("NSButtonImageSource"), func() ArchivedObject { return &NSButtonImageSource{} })
	RegisterArchivedClass([]byte("NSButtonCell"), func() ArchivedObject { return &NSButtonCell{} })
	RegisterArchivedClass([]byte("NSTextFieldCell"), func() ArchivedObject { return &NSTextFieldCell{} })
	RegisterArchivedClass([]byte("NSComboBoxCell"), func() ArchivedObject { return &NSComboBoxCell{} })
	RegisterArchivedClass([]byte("NSTableHeaderCell"), func() ArchivedObject { return &NSTableHeaderCell{} })
	RegisterArchivedClass([]byte("NSResponder"), func() ArchivedObject { return &NSResponder{} })
	RegisterArchivedClass([]byte("NSView"), func() ArchivedObject { return &NSView{} })
	RegisterArchivedClass([]byte("NSControl"), func() ArchivedObject { return &NSControl{} })
}

// objectClassNameAppKit returns the display class name, with special handling for
// NSClassSwapper and NSCustomObject which substitute their own class_name field.
func objectClassNameAppKit(obj interface{}) string {
	switch v := obj.(type) {
	case *NSClassSwapper:
		return v.ClassName
	case *NSCustomObject:
		return v.ClassName
	default:
		return ObjectClassName(obj)
	}
}

// ---- NSBezierPath ----

type NSBezierPathElement struct {
	Op    int64
	Point NSPoint
}

type NSBezierPath struct {
	NSObject
	Elements      []NSBezierPathElement
	WindingRule   int64
	LineCapStyle  int64
	LineJoinStyle int64
	LineWidth     float32
	MiterLimit    float32
	Flatness      float32
	LineDash      *NSBezierPathDash
}

type NSBezierPathDash struct {
	Phase   float32
	Pattern []float32
}

var bezierOpNames = map[int64]string{
	0: "move_to",
	1: "line_to",
	2: "curve_to",
	3: "close_path",
}

var windingRuleNames = map[int64]string{0: "non_zero", 1: "even_odd"}
var lineCapNames = map[int64]string{0: "butt", 1: "round", 2: "square"}
var lineJoinNames = map[int64]string{0: "miter", 1: "round", 2: "bevel"}

func (b *NSBezierPath) InitFromUnarchiver(u Unarchiver, class *Class) error {
	if err := b.NSObject.InitFromUnarchiver(u, class.Superclass); err != nil {
		return err
	}
	if class.Version != 524 {
		return fmt.Errorf("NSBezierPath: unsupported version %d", class.Version)
	}
	countVal, err := u.DecodeValueOfType([]byte("i"))
	if err != nil {
		return err
	}
	count := countVal.(int64)
	b.Elements = make([]NSBezierPathElement, int(count))
	for i := range b.Elements {
		vals, err := u.DecodeValuesOfTypes([]byte("c"), []byte("f"), []byte("f"))
		if err != nil {
			return fmt.Errorf("NSBezierPath element %d: %w", i, err)
		}
		op := vals[0].(int64)
		x, _ := vals[1].(float32)
		y, _ := vals[2].(float32)
		b.Elements[i] = NSBezierPathElement{Op: op, Point: NSPoint{X: x, Y: y}}
	}
	styleVals, err := u.DecodeValuesOfTypes(
		[]byte("i"), []byte("i"), []byte("i"),
		[]byte("f"), []byte("f"), []byte("f"),
		[]byte("i"),
	)
	if err != nil {
		return fmt.Errorf("NSBezierPath style: %w", err)
	}
	b.WindingRule = styleVals[0].(int64)
	b.LineCapStyle = styleVals[1].(int64)
	b.LineJoinStyle = styleVals[2].(int64)
	b.LineWidth, _ = styleVals[3].(float32)
	b.MiterLimit, _ = styleVals[4].(float32)
	b.Flatness, _ = styleVals[5].(float32)
	dashCount := styleVals[6].(int64)
	if dashCount > 0 {
		phaseVal, err := u.DecodeValueOfType([]byte("f"))
		if err != nil {
			return err
		}
		dash := &NSBezierPathDash{Phase: phaseVal.(float32)}
		dash.Pattern = make([]float32, int(dashCount))
		for i := range dash.Pattern {
			v, err := u.DecodeValueOfType([]byte("f"))
			if err != nil {
				return err
			}
			dash.Pattern[i] = v.(float32)
		}
		b.LineDash = dash
	}
	return nil
}
func (b *NSBezierPath) AllowsExtraData() bool               { return false }
func (b *NSBezierPath) AddExtraField(_ *TypedGroup) error   { return nil }
func (b *NSBezierPath) FormatLines(seen map[uintptr]bool) []string {
	var body []string
	body = append(body, "winding rule: "+lookupName(windingRuleNames, b.WindingRule))
	body = append(body, "line cap style: "+lookupName(lineCapNames, b.LineCapStyle))
	body = append(body, "line join style: "+lookupName(lineJoinNames, b.LineJoinStyle))
	body = append(body, "line width: "+formatFloat32(b.LineWidth))
	body = append(body, "miter limit: "+formatFloat32(b.MiterLimit))
	body = append(body, "flatness: "+formatFloat32(b.Flatness))
	if b.LineDash != nil {
		patStrs := make([]string, len(b.LineDash.Pattern))
		for i, p := range b.LineDash.Pattern {
			patStrs[i] = formatFloat32(p)
		}
		body = append(body, fmt.Sprintf("line dash: phase %s, pattern [%s]", formatFloat32(b.LineDash.Phase), strings.Join(patStrs, ", ")))
	}
	if len(b.Elements) > 0 {
		body = append(body, fmt.Sprintf("%d path elements:", len(b.Elements)))
		for _, el := range b.Elements {
			name := lookupName(bezierOpNames, el.Op)
			body = append(body, fmt.Sprintf("\t%s {%s, %s}", name, formatFloat32Coord(el.Point.X), formatFloat32Coord(el.Point.Y)))
		}
	} else {
		body = append(body, "no path elements")
	}
	return formatHeaderBody("NSBezierPath", body)
}

func lookupName(m map[int64]string, v int64) string {
	if s, ok := m[v]; ok {
		return s
	}
	return fmt.Sprintf("%d", v)
}

// ---- NSClassSwapper ----

type NSClassSwapper struct {
	NSObject
	ClassName     string
	TemplateClass *Class
	Template      ArchivedObject
}

func (c *NSClassSwapper) InitFromUnarchiver(u Unarchiver, class *Class) error {
	if err := c.NSObject.InitFromUnarchiver(u, class.Superclass); err != nil {
		return err
	}
	if class.Version != 42 {
		return fmt.Errorf("NSClassSwapper: unsupported version %d", class.Version)
	}
	vals, err := u.DecodeValuesOfTypes([]byte("@"), []byte("#"))
	if err != nil {
		return err
	}
	nsStr, ok := vals[0].(*NSString)
	if !ok || vals[0] == nil {
		return fmt.Errorf("NSClassSwapper: class_name must be *NSString, got %T", vals[0])
	}
	c.ClassName = nsStr.Value
	if vals[1] == nil {
		return fmt.Errorf("NSClassSwapper: template_class is nil")
	}
	c.TemplateClass, ok = vals[1].(*Class)
	if !ok {
		return fmt.Errorf("NSClassSwapper: template_class must be *Class, got %T", vals[1])
	}
	templateObj, templateSuperclass := instantiateArchivedClass(c.TemplateClass)
	c.Template = templateObj
	var knownObj ArchivedObject
	if generic, ok2 := c.Template.(*GenericArchivedObject); ok2 {
		knownObj = generic.SuperObject
	} else {
		knownObj = c.Template
	}
	if knownObj != nil && templateSuperclass != nil {
		if err := knownObj.InitFromUnarchiver(u, templateSuperclass); err != nil {
			return fmt.Errorf("NSClassSwapper template init: %w", err)
		}
	}
	return nil
}
func (c *NSClassSwapper) AllowsExtraData() bool {
	if c.Template != nil {
		return c.Template.AllowsExtraData()
	}
	return false
}
func (c *NSClassSwapper) AddExtraField(f *TypedGroup) error {
	if c.Template != nil {
		return c.Template.AddExtraField(f)
	}
	return nil
}
func (c *NSClassSwapper) FormatLines(seen map[uintptr]bool) []string {
	prefix := fmt.Sprintf("NSClassSwapper, class name %q, template: ", c.ClassName)
	return FormatValueWithPrefix(c.Template, prefix, seen)
}

// ---- NSColor ----

type NSColorKind int64

const (
	NSColorKindCalibratedRGBA NSColorKind = 1
	NSColorKindDeviceRGBA     NSColorKind = 2
	NSColorKindCalibratedWA   NSColorKind = 3
	NSColorKindDeviceWA       NSColorKind = 4
	NSColorKindDeviceCMYKA    NSColorKind = 5
	NSColorKindNamed          NSColorKind = 6
)

func (k NSColorKind) Name() string {
	switch k {
	case NSColorKindCalibratedRGBA:
		return "CALIBRATED_RGBA"
	case NSColorKindDeviceRGBA:
		return "DEVICE_RGBA"
	case NSColorKindCalibratedWA:
		return "CALIBRATED_WA"
	case NSColorKindDeviceWA:
		return "DEVICE_WA"
	case NSColorKindDeviceCMYKA:
		return "DEVICE_CMYKA"
	case NSColorKindNamed:
		return "NAMED"
	default:
		return fmt.Sprintf("%d", int64(k))
	}
}

type NSColorRGBAValue struct{ Red, Green, Blue, Alpha float32 }
type NSColorWAValue struct{ White, Alpha float32 }
type NSColorCMYKAValue struct{ Cyan, Magenta, Yellow, Black, Alpha float32 }
type NSColorNamedValue struct {
	Group string
	Name  string
	Color *NSColor
}

func (v NSColorRGBAValue) String() string {
	return fmt.Sprintf("%s, %s, %s, %s",
		pyF32(v.Red), pyF32(v.Green), pyF32(v.Blue), pyF32(v.Alpha))
}
func (v NSColorWAValue) String() string { return fmt.Sprintf("%s, %s", pyF32(v.White), pyF32(v.Alpha)) }
func (v NSColorCMYKAValue) String() string {
	return fmt.Sprintf("%s, %s, %s, %s, %s",
		pyF32(v.Cyan), pyF32(v.Magenta), pyF32(v.Yellow), pyF32(v.Black), pyF32(v.Alpha))
}

// pyF32 formats a float32 as Python would: full float64 precision, with .0 for whole numbers.
func pyF32(f float32) string {
	s := fmt.Sprintf("%g", float64(f))
	if !strings.ContainsAny(s, ".e") {
		s += ".0"
	}
	return s
}
func (v NSColorNamedValue) String() string {
	return fmt.Sprintf("group %s, name %s, color %s", pyStrRepr(v.Group), pyStrRepr(v.Name), v.Color)
}

type NSColor struct {
	NSObject
	Kind  NSColorKind
	Value interface{} // NSColorRGBAValue | NSColorWAValue | NSColorCMYKAValue | NSColorNamedValue
}

func (c *NSColor) InitFromUnarchiver(u Unarchiver, class *Class) error {
	if err := c.NSObject.InitFromUnarchiver(u, class.Superclass); err != nil {
		return err
	}
	if class.Version != 0 {
		return fmt.Errorf("NSColor: unsupported version %d", class.Version)
	}
	kindVal, err := u.DecodeValueOfType([]byte("c"))
	if err != nil {
		return err
	}
	c.Kind = NSColorKind(kindVal.(int64))
	switch c.Kind {
	case NSColorKindCalibratedRGBA, NSColorKindDeviceRGBA:
		vals, err := u.DecodeValuesOfTypes([]byte("f"), []byte("f"), []byte("f"), []byte("f"))
		if err != nil {
			return fmt.Errorf("NSColor RGBA: %w", err)
		}
		c.Value = NSColorRGBAValue{
			Red:   vals[0].(float32),
			Green: vals[1].(float32),
			Blue:  vals[2].(float32),
			Alpha: vals[3].(float32),
		}
	case NSColorKindCalibratedWA, NSColorKindDeviceWA:
		vals, err := u.DecodeValuesOfTypes([]byte("f"), []byte("f"))
		if err != nil {
			return fmt.Errorf("NSColor WA: %w", err)
		}
		c.Value = NSColorWAValue{White: vals[0].(float32), Alpha: vals[1].(float32)}
	case NSColorKindDeviceCMYKA:
		vals, err := u.DecodeValuesOfTypes([]byte("f"), []byte("f"), []byte("f"), []byte("f"), []byte("f"))
		if err != nil {
			return fmt.Errorf("NSColor CMYKA: %w", err)
		}
		c.Value = NSColorCMYKAValue{
			Cyan:    vals[0].(float32),
			Magenta: vals[1].(float32),
			Yellow:  vals[2].(float32),
			Black:   vals[3].(float32),
			Alpha:   vals[4].(float32),
		}
	case NSColorKindNamed:
		vals, err := u.DecodeValuesOfTypes([]byte("@"), []byte("@"), []byte("@"))
		if err != nil {
			return fmt.Errorf("NSColor named: %w", err)
		}
		groupStr, ok1 := vals[0].(*NSString)
		nameStr, ok2 := vals[1].(*NSString)
		colorObj, ok3 := vals[2].(*NSColor)
		if !ok1 || !ok2 || !ok3 {
			return fmt.Errorf("NSColor named: unexpected types %T, %T, %T", vals[0], vals[1], vals[2])
		}
		c.Value = NSColorNamedValue{Group: groupStr.Value, Name: nameStr.Value, Color: colorObj}
	default:
		return fmt.Errorf("NSColor: unknown kind %d", c.Kind)
	}
	return nil
}
func (c *NSColor) AllowsExtraData() bool               { return false }
func (c *NSColor) AddExtraField(_ *TypedGroup) error   { return nil }
func (c *NSColor) String() string {
	return fmt.Sprintf("<NSColor %s: %s>", c.Kind.Name(), c.Value)
}
func (c *NSColor) FormatLines(_ map[uintptr]bool) []string {
	return []string{c.String()}
}

// ---- NSCustomObject ----

type NSCustomObject struct {
	NSObject
	ClassName string
	Object    interface{}
}

func (c *NSCustomObject) InitFromUnarchiver(u Unarchiver, class *Class) error {
	if err := c.NSObject.InitFromUnarchiver(u, class.Superclass); err != nil {
		return err
	}
	if class.Version != 41 {
		return fmt.Errorf("NSCustomObject: unsupported version %d", class.Version)
	}
	vals, err := u.DecodeValuesOfTypes([]byte("@"), []byte("@"))
	if err != nil {
		return err
	}
	nsStr, ok := vals[0].(*NSString)
	if !ok || vals[0] == nil {
		return fmt.Errorf("NSCustomObject: class_name must be *NSString, got %T", vals[0])
	}
	c.ClassName = nsStr.Value
	c.Object = vals[1]
	return nil
}
func (c *NSCustomObject) AllowsExtraData() bool               { return false }
func (c *NSCustomObject) AddExtraField(_ *TypedGroup) error   { return nil }
func (c *NSCustomObject) FormatLines(seen map[uintptr]bool) []string {
	header := fmt.Sprintf("NSCustomObject, class %s", c.ClassName)
	if c.Object == nil {
		return []string{header}
	}
	return FormatValueWithPrefix(c.Object, header+", object: ", seen)
}

// ---- NSCustomResource ----

type NSCustomResource struct {
	NSObject
	ClassName    string
	ResourceName string
}

func (c *NSCustomResource) InitFromUnarchiver(u Unarchiver, class *Class) error {
	if err := c.NSObject.InitFromUnarchiver(u, class.Superclass); err != nil {
		return err
	}
	if class.Version != 41 {
		return fmt.Errorf("NSCustomResource: unsupported version %d", class.Version)
	}
	vals, err := u.DecodeValuesOfTypes([]byte("@"), []byte("@"))
	if err != nil {
		return err
	}
	cls, ok1 := vals[0].(*NSString)
	res, ok2 := vals[1].(*NSString)
	if !ok1 || !ok2 {
		return fmt.Errorf("NSCustomResource: fields must be *NSString, got %T, %T", vals[0], vals[1])
	}
	c.ClassName = cls.Value
	c.ResourceName = res.Value
	return nil
}
func (c *NSCustomResource) AllowsExtraData() bool               { return false }
func (c *NSCustomResource) AddExtraField(_ *TypedGroup) error   { return nil }
func (c *NSCustomResource) FormatLines(_ map[uintptr]bool) []string {
	return []string{fmt.Sprintf("NSCustomResource(class_name=%q, resource_name=%q)", c.ClassName, c.ResourceName)}
}

// ---- NSFont ----

type NSFont struct {
	NSObject
	Name  string
	Size  float32
	Flags [4]int64
}

func (f *NSFont) InitFromUnarchiver(u Unarchiver, class *Class) error {
	if err := f.NSObject.InitFromUnarchiver(u, class.Superclass); err != nil {
		return err
	}
	if class.Version != 21 && class.Version != 30 {
		return fmt.Errorf("NSFont: unsupported version %d", class.Version)
	}
	nameVal, err := u.DecodePropertyList()
	if err != nil {
		return fmt.Errorf("NSFont name: %w", err)
	}
	name, ok := nameVal.(string)
	if !ok {
		return fmt.Errorf("NSFont: name must be string, got %T", nameVal)
	}
	f.Name = name
	sizeVal, err := u.DecodeValueOfType([]byte("f"))
	if err != nil {
		return fmt.Errorf("NSFont size: %w", err)
	}
	f.Size = sizeVal.(float32)
	for i := 0; i < 4; i++ {
		v, err := u.DecodeValueOfType([]byte("c"))
		if err != nil {
			return fmt.Errorf("NSFont flag %d: %w", i, err)
		}
		f.Flags[i] = v.(int64)
	}
	return nil
}
func (f *NSFont) AllowsExtraData() bool               { return false }
func (f *NSFont) AddExtraField(_ *TypedGroup) error   { return nil }
func (f *NSFont) FormatLines(_ map[uintptr]bool) []string {
	flags := fmt.Sprintf("0x%02x, 0x%02x, 0x%02x, 0x%02x",
		f.Flags[0]&0xff, f.Flags[1]&0xff, f.Flags[2]&0xff, f.Flags[3]&0xff)
	return []string{fmt.Sprintf("NSFont(name=%s, size=%s, flags_unknown=(%s))", pyStrRepr(f.Name), formatFloat32(f.Size), flags)}
}

// ---- NSIBObjectData helper types ----

type ObjToObj struct{ Obj, Val interface{} }
type ObjToOptStr struct {
	Obj  interface{}
	Name *string
}
type ObjToInt64 struct {
	Obj interface{}
	ID  int64
}
type ObjToStr struct {
	Obj interface{}
	Str string
}

// ---- NSIBObjectData ----

type NSIBObjectData struct {
	NSObject
	Root            interface{}
	Parents         []ObjToObj    // child → parent
	Names           []ObjToOptStr // obj → optional name
	UnknownSet      interface{}
	Connections     []interface{}
	UnknownObject   interface{}
	ObjectIDs       []ObjToInt64
	NextObjectID    int64
	SwapperNames    []ObjToStr
	TargetFramework string
}

func (d *NSIBObjectData) InitFromUnarchiver(u Unarchiver, class *Class) error {
	if err := d.NSObject.InitFromUnarchiver(u, class.Superclass); err != nil {
		return err
	}
	if class.Version != 224 {
		return fmt.Errorf("NSIBObjectData: unsupported version %d", class.Version)
	}
	var err error
	d.Root, err = u.DecodeValueOfType([]byte("@"))
	if err != nil {
		return fmt.Errorf("NSIBObjectData root: %w", err)
	}
	parentsCountVal, err := u.DecodeValueOfType([]byte("i"))
	if err != nil {
		return err
	}
	parentsCount := parentsCountVal.(int64)
	d.Parents = make([]ObjToObj, int(parentsCount))
	for i := range d.Parents {
		vals, err := u.DecodeValuesOfTypes([]byte("@"), []byte("@"))
		if err != nil {
			return fmt.Errorf("NSIBObjectData parent %d: %w", i, err)
		}
		d.Parents[i] = ObjToObj{vals[0], vals[1]}
	}
	namesCountVal, err := u.DecodeValueOfType([]byte("i"))
	if err != nil {
		return err
	}
	namesCount := namesCountVal.(int64)
	d.Names = make([]ObjToOptStr, int(namesCount))
	for i := range d.Names {
		vals, err := u.DecodeValuesOfTypes([]byte("@"), []byte("@"))
		if err != nil {
			return fmt.Errorf("NSIBObjectData name %d: %w", i, err)
		}
		var namePtr *string
		if vals[1] != nil {
			if ns, ok := vals[1].(*NSString); ok {
				s := ns.Value
				namePtr = &s
			}
		}
		d.Names[i] = ObjToOptStr{vals[0], namePtr}
	}
	d.UnknownSet, err = u.DecodeValueOfType([]byte("@"))
	if err != nil {
		return fmt.Errorf("NSIBObjectData unknown set: %w", err)
	}
	connObj, err := u.DecodeValueOfType([]byte("@"))
	if err != nil {
		return fmt.Errorf("NSIBObjectData connections: %w", err)
	}
	if connArr, ok := connObj.(*NSArray); ok {
		d.Connections = connArr.Elements
	}
	d.UnknownObject, err = u.DecodeValueOfType([]byte("@"))
	if err != nil {
		return fmt.Errorf("NSIBObjectData unknown object: %w", err)
	}
	oidsCountVal, err := u.DecodeValueOfType([]byte("i"))
	if err != nil {
		return err
	}
	oidsCount := oidsCountVal.(int64)
	d.ObjectIDs = make([]ObjToInt64, int(oidsCount))
	for i := range d.ObjectIDs {
		vals, err := u.DecodeValuesOfTypes([]byte("@"), []byte("i"))
		if err != nil {
			return fmt.Errorf("NSIBObjectData oid %d: %w", i, err)
		}
		d.ObjectIDs[i] = ObjToInt64{vals[0], vals[1].(int64)}
	}
	nextOIDVal, err := u.DecodeValueOfType([]byte("i"))
	if err != nil {
		return err
	}
	d.NextObjectID = nextOIDVal.(int64)
	swapperCountVal, err := u.DecodeValueOfType([]byte("i"))
	if err != nil {
		return err
	}
	swapperCount := swapperCountVal.(int64)
	d.SwapperNames = make([]ObjToStr, int(swapperCount))
	for i := range d.SwapperNames {
		vals, err := u.DecodeValuesOfTypes([]byte("@"), []byte("@"))
		if err != nil {
			return fmt.Errorf("NSIBObjectData swapper %d: %w", i, err)
		}
		nsStr, ok := vals[1].(*NSString)
		if !ok {
			return fmt.Errorf("NSIBObjectData swapper class name must be *NSString")
		}
		d.SwapperNames[i] = ObjToStr{vals[0], nsStr.Value}
	}
	fwVal, err := u.DecodeValueOfType([]byte("@"))
	if err != nil {
		return fmt.Errorf("NSIBObjectData target framework: %w", err)
	}
	fwStr, ok := fwVal.(*NSString)
	if !ok {
		return fmt.Errorf("NSIBObjectData target framework must be *NSString")
	}
	d.TargetFramework = fwStr.Value
	return nil
}
func (d *NSIBObjectData) AllowsExtraData() bool               { return false }
func (d *NSIBObjectData) AddExtraField(_ *TypedGroup) error   { return nil }

func (d *NSIBObjectData) oidRepr(obj interface{}) string {
	for _, e := range d.ObjectIDs {
		if e.Obj == obj {
			return fmt.Sprintf("#%d", e.ID)
		}
	}
	return "<missing OID!>"
}

func (d *NSIBObjectData) objectDesc(obj interface{}) string {
	if obj == nil {
		return "nil"
	}
	desc := objectClassNameAppKit(obj)
	for _, n := range d.Names {
		if n.Obj == obj {
			if n.Name == nil {
				desc += " None"
			} else {
				desc += " " + fmt.Sprintf("%q", *n.Name)
			}
			break
		}
	}
	return fmt.Sprintf("%s (%s)", d.oidRepr(obj), desc)
}

func (d *NSIBObjectData) renderTree(obj interface{}, children map[interface{}][]interface{}, seen map[interface{}]bool) []string {
	var lines []string
	lines = append(lines, d.objectDesc(obj))
	seen[obj] = true
	for _, child := range children[obj] {
		if seen[child] {
			lines = append(lines, "\tWARNING: object appears more than once in tree: "+d.objectDesc(obj))
		} else {
			for _, line := range d.renderTree(child, children, seen) {
				lines = append(lines, "\t"+line)
			}
		}
	}
	return lines
}

func (d *NSIBObjectData) getOID(obj interface{}) int64 {
	for _, e := range d.ObjectIDs {
		if e.Obj == obj {
			return e.ID
		}
	}
	return 0
}

func (d *NSIBObjectData) FormatLines(seen map[uintptr]bool) []string {
	header := fmt.Sprintf("NSIBObjectData, target framework %q", d.TargetFramework)
	var body []string

	// Build children map: parent → [children].
	children := make(map[interface{}][]interface{})
	for _, p := range d.Parents {
		children[p.Val] = append(children[p.Val], p.Obj)
	}
	// Sort children by OID.
	for parent := range children {
		cs := children[parent]
		sort.Slice(cs, func(i, j int) bool {
			return d.getOID(cs[i]) < d.getOID(cs[j])
		})
		children[parent] = cs
	}

	seenInTree := make(map[interface{}]bool)
	treeLines := d.renderTree(d.Root, children, seenInTree)
	body = append(body, prefixLines(treeLines, "object tree: ", "\t")...)

	// Warn about missed parent objects.
	for parent := range children {
		if !seenInTree[parent] {
			body = append(body, "WARNING: one or more parent objects not reachable from root:")
			body = append(body, fmt.Sprintf("\t%s has children:", d.objectDesc(parent)))
			for _, child := range children[parent] {
				body = append(body, fmt.Sprintf("\t\t%s", d.objectDesc(child)))
			}
		}
	}

	// Warn about missed named objects.
	for _, n := range d.Names {
		if !seenInTree[n.Obj] {
			body = append(body, "WARNING: one or more named objects not reachable from root:")
			body = append(body, fmt.Sprintf("\t%s", d.objectDesc(n.Obj)))
		}
	}

	// Connections.
	body = append(body, fmt.Sprintf("%d connections:", len(d.Connections)))
	for _, conn := range d.Connections {
		line := "\t" + d.objectDesc(conn)
		switch c := conn.(type) {
		case *NSIBHelpConnector:
			line += fmt.Sprintf(": %s %q = %q", d.objectDesc(c.Object), c.Key, c.Value)
		case *NSNibControlConnector:
			line += fmt.Sprintf(": %s -> [%s %s]",
				d.objectDesc(c.NSNibConnector.Source),
				d.objectDesc(c.NSNibConnector.Destination),
				c.NSNibConnector.Label)
		case *NSNibOutletConnector:
			line += fmt.Sprintf(": %s.%s = %s",
				d.objectDesc(c.NSNibConnector.Source),
				c.NSNibConnector.Label,
				d.objectDesc(c.NSNibConnector.Destination))
		case *NSNibConnector:
			line += fmt.Sprintf(": %s -> %q -> %s",
				d.objectDesc(c.Source), c.Label, d.objectDesc(c.Destination))
		}
		body = append(body, line)
	}

	// Warn about objects not in tree or connections.
	connSet := make(map[interface{}]bool)
	for _, c := range d.Connections {
		connSet[c] = true
	}
	for _, e := range d.ObjectIDs {
		if !seenInTree[e.Obj] && !connSet[e.Obj] {
			body = append(body, "WARNING: one or more objects not reachable from root or connections:")
			body = append(body, fmt.Sprintf("\t%s", d.objectDesc(e.Obj)))
		}
	}

	// Swapper class names.
	if len(d.SwapperNames) > 0 {
		body = append(body, fmt.Sprintf("%d swapper class names:", len(d.SwapperNames)))
		for _, sn := range d.SwapperNames {
			body = append(body, fmt.Sprintf("\t%s: %q", d.objectDesc(sn.Obj), sn.Str))
		}
	}

	// Objects list.
	body = append(body, fmt.Sprintf("%d objects:", len(d.ObjectIDs)))
	for _, e := range d.ObjectIDs {
		oidDesc := fmt.Sprintf("#%d", e.ID)
		for _, n := range d.Names {
			if n.Obj == e.Obj {
				if n.Name == nil {
					oidDesc += " None"
				} else {
					oidDesc += " " + fmt.Sprintf("%q", *n.Name)
				}
				break
			}
		}
		body = append(body, prefixLines(FormatValue(e.Obj, seen), "\t"+oidDesc+": ", "\t")...)
	}

	body = append(body, fmt.Sprintf("next object ID: #%d", d.NextObjectID))
	body = append(body, FormatValueWithPrefix(d.UnknownSet, "unknown set: ", seen)...)
	body = append(body, FormatValueWithPrefix(d.UnknownObject, "unknown object: ", seen)...)

	return formatHeaderBody(header, body)
}

// ---- NSIBHelpConnector ----

type NSIBHelpConnector struct {
	NSObject
	Object interface{}
	Key    string
	Value  string
}

func (c *NSIBHelpConnector) InitFromUnarchiver(u Unarchiver, class *Class) error {
	if err := c.NSObject.InitFromUnarchiver(u, class.Superclass); err != nil {
		return err
	}
	if class.Version != 17 {
		return fmt.Errorf("NSIBHelpConnector: unsupported version %d", class.Version)
	}
	vals, err := u.DecodeValuesOfTypes([]byte("@"), []byte("@"), []byte("@"))
	if err != nil {
		return err
	}
	c.Object = vals[0]
	key, ok1 := vals[1].(*NSString)
	val, ok2 := vals[2].(*NSString)
	if !ok1 || !ok2 {
		return fmt.Errorf("NSIBHelpConnector: key/value must be *NSString, got %T, %T", vals[1], vals[2])
	}
	c.Key = key.Value
	c.Value = val.Value
	return nil
}
func (c *NSIBHelpConnector) AllowsExtraData() bool               { return false }
func (c *NSIBHelpConnector) AddExtraField(_ *TypedGroup) error   { return nil }
func (c *NSIBHelpConnector) FormatLines(seen map[uintptr]bool) []string {
	header := fmt.Sprintf("NSIBHelpConnector, key %q", c.Key)
	var body []string
	body = append(body, fmt.Sprintf("value: %q", c.Value))
	body = append(body, FormatValueWithPrefix(c.Object, "object: ", seen)...)
	return formatHeaderBody(header, body)
}

// ---- NSNibConnector ----

type NSNibConnector struct {
	NSObject
	Source      interface{}
	Destination interface{}
	Label       string
}

func (c *NSNibConnector) InitFromUnarchiver(u Unarchiver, class *Class) error {
	if err := c.NSObject.InitFromUnarchiver(u, class.Superclass); err != nil {
		return err
	}
	if class.Version != 17 {
		return fmt.Errorf("NSNibConnector: unsupported version %d", class.Version)
	}
	vals, err := u.DecodeValuesOfTypes([]byte("@"), []byte("@"), []byte("@"))
	if err != nil {
		return err
	}
	c.Source = vals[0]
	c.Destination = vals[1]
	lbl, ok := vals[2].(*NSString)
	if !ok {
		return fmt.Errorf("NSNibConnector: label must be *NSString, got %T", vals[2])
	}
	c.Label = lbl.Value
	return nil
}
func (c *NSNibConnector) AllowsExtraData() bool               { return false }
func (c *NSNibConnector) AddExtraField(_ *TypedGroup) error   { return nil }
func (c *NSNibConnector) FormatLines(seen map[uintptr]bool) []string {
	header := fmt.Sprintf("NSNibConnector, label %q", c.Label)
	var body []string
	body = append(body, FormatValueWithPrefix(c.Source, "source: ", seen)...)
	body = append(body, FormatValueWithPrefix(c.Destination, "destination: ", seen)...)
	return formatHeaderBody(header, body)
}

// ---- NSNibControlConnector ----

type NSNibControlConnector struct {
	NSNibConnector
}

func (c *NSNibControlConnector) InitFromUnarchiver(u Unarchiver, class *Class) error {
	if err := c.NSNibConnector.InitFromUnarchiver(u, class.Superclass); err != nil {
		return err
	}
	if class.Version != 207 {
		return fmt.Errorf("NSNibControlConnector: unsupported version %d", class.Version)
	}
	return nil
}
func (c *NSNibControlConnector) AllowsExtraData() bool               { return false }
func (c *NSNibControlConnector) AddExtraField(_ *TypedGroup) error   { return nil }
func (c *NSNibControlConnector) FormatLines(_ map[uintptr]bool) []string {
	return []string{fmt.Sprintf("NSNibControlConnector <%s> -> -[<%s> %s]",
		objectClassNameAppKit(c.Source), objectClassNameAppKit(c.Destination), c.Label)}
}

// ---- NSNibOutletConnector ----

type NSNibOutletConnector struct {
	NSNibConnector
}

func (c *NSNibOutletConnector) InitFromUnarchiver(u Unarchiver, class *Class) error {
	if err := c.NSNibConnector.InitFromUnarchiver(u, class.Superclass); err != nil {
		return err
	}
	if class.Version != 207 {
		return fmt.Errorf("NSNibOutletConnector: unsupported version %d", class.Version)
	}
	return nil
}
func (c *NSNibOutletConnector) AllowsExtraData() bool               { return false }
func (c *NSNibOutletConnector) AddExtraField(_ *TypedGroup) error   { return nil }
func (c *NSNibOutletConnector) FormatLines(_ map[uintptr]bool) []string {
	return []string{fmt.Sprintf("NSNibOutletConnector <%s>.%s = <%s>",
		objectClassNameAppKit(c.Source), c.Label, objectClassNameAppKit(c.Destination))}
}

// ---- NSMenuItem ----

type NSMenuItem struct {
	NSObject
	Menu            interface{}
	Flags           int64
	Title           string
	KeyEquivalent   string
	ModifierFlags   int64
	State           int64
	OnStateImage    interface{}
	OffStateImage   interface{}
	MixedStateImage interface{}
	Action          []byte
	Int2            int64
	Target          interface{}
	Submenu         interface{}
}

var modifierFlagNames = []struct {
	flag int64
	name string
}{
	{1 << 16, "CapsLock"},
	{1 << 17, "Shift"},
	{1 << 18, "Ctrl"},
	{1 << 19, "Alt"},
	{1 << 20, "Cmd"},
	{1 << 21, "(NumPad)"},
	{1 << 22, "(Help)"},
	{1 << 23, "(FKey)"},
}

func formatModifierFlags(flags int64) string {
	if flags == 0 {
		return "(no modifiers)"
	}
	var parts []string
	rem := flags
	for _, entry := range modifierFlagNames {
		if rem&entry.flag != 0 {
			parts = append(parts, entry.name)
			rem &^= entry.flag
		}
	}
	if rem != 0 {
		parts = append(parts, fmt.Sprintf("(%#x)", rem))
	}
	return strings.Join(parts, "+")
}

var controlStateNames = map[int64]string{-1: "mixed", 0: "off", 1: "on"}

func (m *NSMenuItem) InitFromUnarchiver(u Unarchiver, class *Class) error {
	if err := m.NSObject.InitFromUnarchiver(u, class.Superclass); err != nil {
		return err
	}
	if class.Version != 505 && class.Version != 671 {
		return fmt.Errorf("NSMenuItem: unsupported version %d", class.Version)
	}
	menuVal, err := u.DecodeValueOfType([]byte("@"))
	if err != nil {
		return fmt.Errorf("NSMenuItem menu: %w", err)
	}
	m.Menu = menuVal
	vals, err := u.DecodeValuesOfTypes(
		[]byte("i"), []byte("@"), []byte("@"), []byte("I"),
		[]byte("I"), []byte("i"),
		[]byte("@"), []byte("@"), []byte("@"), []byte("@"),
		[]byte(":"), []byte("i"), []byte("@"),
	)
	if err != nil {
		return fmt.Errorf("NSMenuItem values: %w", err)
	}
	m.Flags = vals[0].(int64) & 0xffffffff
	if ns, ok := vals[1].(*NSString); ok {
		m.Title = ns.Value
	}
	if ns, ok := vals[2].(*NSString); ok {
		m.KeyEquivalent = ns.Value
	}
	m.ModifierFlags = vals[3].(int64)
	// vals[4] is int_1 (must be 0x7fffffff — just ignore)
	m.State = vals[5].(int64)
	// vals[6] is obj_1 (must be nil — just ignore)
	m.OnStateImage = vals[7]
	m.OffStateImage = vals[8]
	m.MixedStateImage = vals[9]
	m.Action, _ = vals[10].([]byte)
	m.Int2 = vals[11].(int64)
	// vals[12] is obj_2 (must be nil — just ignore)
	m.Target, err = u.DecodeValueOfType([]byte("@"))
	if err != nil {
		return fmt.Errorf("NSMenuItem target: %w", err)
	}
	m.Submenu, err = u.DecodeValueOfType([]byte("@"))
	if err != nil {
		return fmt.Errorf("NSMenuItem submenu: %w", err)
	}
	return nil
}
func (m *NSMenuItem) AllowsExtraData() bool               { return false }
func (m *NSMenuItem) AddExtraField(_ *TypedGroup) error   { return nil }
func (m *NSMenuItem) FormatLines(seen map[uintptr]bool) []string {
	header := fmt.Sprintf("NSMenuItem %q", m.Title)
	if m.KeyEquivalent != "" {
		header += fmt.Sprintf(" (%s+%q)", formatModifierFlags(m.ModifierFlags), m.KeyEquivalent)
	}
	var body []string
	if menu, ok := m.Menu.(*NSMenu); ok {
		body = append(body, fmt.Sprintf("in menu: <NSMenu %q>", menu.Title))
	}
	if m.Flags != 0 {
		body = append(body, fmt.Sprintf("flags: 0x%08x", m.Flags))
	}
	if state := lookupName(controlStateNames, m.State); state != "off" {
		body = append(body, "initial state: "+state)
	}
	// Only show non-default state images.
	if !isDefaultCheckmark(m.OnStateImage) {
		body = append(body, FormatValueWithPrefix(m.OnStateImage, "on state image: ", seen)...)
	}
	if m.OffStateImage != nil {
		body = append(body, FormatValueWithPrefix(m.OffStateImage, "off state image: ", seen)...)
	}
	if !isDefaultMixedState(m.MixedStateImage) {
		body = append(body, FormatValueWithPrefix(m.MixedStateImage, "mixed state image: ", seen)...)
	}
	if m.Action != nil {
		body = append(body, fmt.Sprintf("action: %s", m.Action))
	}
	if m.Int2 != 0 {
		body = append(body, fmt.Sprintf("unknown int 2: %d", m.Int2))
	}
	if m.Target != nil {
		body = append(body, fmt.Sprintf("target: <%s>", objectClassNameAppKit(m.Target)))
	}
	if m.Submenu != nil {
		body = append(body, FormatValueWithPrefix(m.Submenu, "submenu: ", seen)...)
	}
	return formatHeaderBody(header, body)
}

func isDefaultCheckmark(img interface{}) bool {
	r, ok := img.(*NSCustomResource)
	return ok && r.ClassName == "NSImage" && r.ResourceName == "NSMenuCheckmark"
}

func isDefaultMixedState(img interface{}) bool {
	r, ok := img.(*NSCustomResource)
	return ok && r.ClassName == "NSImage" && r.ResourceName == "NSMenuMixedState"
}

// ---- NSMenu ----

type NSMenu struct {
	NSObject
	Title      string
	Items      []*NSMenuItem
	Identifier string
	HasIdent   bool
}

func (m *NSMenu) InitFromUnarchiver(u Unarchiver, class *Class) error {
	if err := m.NSObject.InitFromUnarchiver(u, class.Superclass); err != nil {
		return err
	}
	if class.Version != 204 {
		return fmt.Errorf("NSMenu: unsupported version %d", class.Version)
	}
	vals, err := u.DecodeValuesOfTypes([]byte("i"), []byte("@"), []byte("@"), []byte("@"))
	if err != nil {
		return fmt.Errorf("NSMenu: %w", err)
	}
	// vals[0] is unknown int (must be 0)
	if ns, ok := vals[1].(*NSString); ok {
		m.Title = ns.Value
	}
	if arr, ok := vals[2].(*NSArray); ok && arr != nil {
		for _, item := range arr.Elements {
			mi, ok := item.(*NSMenuItem)
			if !ok {
				return fmt.Errorf("NSMenu: item must be *NSMenuItem, got %T", item)
			}
			m.Items = append(m.Items, mi)
		}
	}
	if vals[3] != nil {
		if ns, ok := vals[3].(*NSString); ok {
			m.Identifier = ns.Value
			m.HasIdent = true
		}
	}
	return nil
}
func (m *NSMenu) AllowsExtraData() bool               { return false }
func (m *NSMenu) AddExtraField(_ *TypedGroup) error   { return nil }
func (m *NSMenu) FormatLines(seen map[uintptr]bool) []string {
	header := fmt.Sprintf("NSMenu %q", m.Title)
	if m.HasIdent {
		header += fmt.Sprintf(" (%q)", m.Identifier)
	}
	switch len(m.Items) {
	case 0:
		header += ", no items"
	case 1:
		header += ", 1 item"
	default:
		header += fmt.Sprintf(", %d items", len(m.Items))
	}
	var body []string
	for _, item := range m.Items {
		body = append(body, FormatValue(item, seen)...)
	}
	return formatHeaderBody(header, body)
}

// ---- NSCell ----

type NSCell struct {
	NSObject
	Flags        [2]int64
	TitleOrImage interface{}
	Font         interface{}
}

func (c *NSCell) InitFromUnarchiver(u Unarchiver, class *Class) error {
	if err := c.NSObject.InitFromUnarchiver(u, class.Superclass); err != nil {
		return err
	}
	if class.Version != 65 {
		return fmt.Errorf("NSCell: unsupported version %d", class.Version)
	}
	flagVals, err := u.DecodeValuesOfTypes([]byte("i"), []byte("i"))
	if err != nil {
		return fmt.Errorf("NSCell flags: %w", err)
	}
	c.Flags[0] = flagVals[0].(int64) & 0xffffffff
	c.Flags[1] = flagVals[1].(int64) & 0xffffffff
	objVals, err := u.DecodeValuesOfTypes([]byte("@"), []byte("@"), []byte("@"), []byte("@"))
	if err != nil {
		return fmt.Errorf("NSCell objects: %w", err)
	}
	c.TitleOrImage = objVals[0]
	c.Font = objVals[1]
	// objVals[2] and [3] must be nil
	return nil
}
func (c *NSCell) AllowsExtraData() bool               { return false }
func (c *NSCell) AddExtraField(_ *TypedGroup) error   { return nil }
func (c *NSCell) FormatLines(seen map[uintptr]bool) []string {
	var body []string
	body = append(body, fmt.Sprintf("flags: (0x%08x, 0x%08x)", c.Flags[0], c.Flags[1]))
	body = append(body, fmt.Sprintf("title/image: %v", c.TitleOrImage))
	body = append(body, FormatValueWithPrefix(c.Font, "font: ", seen)...)
	return formatHeaderBody("NSCell", body)
}

// ---- NSImageCell ----

type NSImageCell struct {
	NSCell
	ImageAlignment  int64
	ImageScaling    int64
	ImageFrameStyle int64
}

var imageAlignmentNames = map[int64]string{
	0: "center", 1: "top", 2: "top_left", 3: "top_right",
	4: "left", 5: "bottom", 6: "bottom_left", 7: "bottom_right", 8: "right",
}
var imageScalingNames = map[int64]string{
	0: "proportionally_down", 1: "axes_independently",
	2: "none", 3: "proportionally_up_or_down",
}
var imageFrameStyleNames = map[int64]string{
	0: "none", 1: "photo", 2: "gray_bezel", 3: "groove", 4: "button",
}

func (c *NSImageCell) InitFromUnarchiver(u Unarchiver, class *Class) error {
	if err := c.NSCell.InitFromUnarchiver(u, class.Superclass); err != nil {
		return err
	}
	if class.Version != 41 {
		return fmt.Errorf("NSImageCell: unsupported version %d", class.Version)
	}
	vals, err := u.DecodeValuesOfTypes([]byte("i"), []byte("i"), []byte("i"))
	if err != nil {
		return err
	}
	c.ImageAlignment = vals[0].(int64)
	c.ImageScaling = vals[1].(int64)
	c.ImageFrameStyle = vals[2].(int64)
	return nil
}
func (c *NSImageCell) AllowsExtraData() bool               { return false }
func (c *NSImageCell) AddExtraField(_ *TypedGroup) error   { return nil }
func (c *NSImageCell) FormatLines(seen map[uintptr]bool) []string {
	body := c.NSCell.cellBodyLines(seen)
	body = append(body, "image alignment: "+lookupName(imageAlignmentNames, c.ImageAlignment))
	body = append(body, "image scaling: "+lookupName(imageScalingNames, c.ImageScaling))
	body = append(body, "image frame style: "+lookupName(imageFrameStyleNames, c.ImageFrameStyle))
	return formatHeaderBody("NSImageCell", body)
}

// cellBodyLines returns the common NSCell body lines (for use by subclasses).
func (c *NSCell) cellBodyLines(seen map[uintptr]bool) []string {
	var body []string
	body = append(body, fmt.Sprintf("flags: (0x%08x, 0x%08x)", c.Flags[0], c.Flags[1]))
	body = append(body, fmt.Sprintf("title/image: %v", c.TitleOrImage))
	body = append(body, FormatValueWithPrefix(c.Font, "font: ", seen)...)
	return body
}

// ---- NSActionCell ----

type NSActionCell struct {
	NSCell
	Tag         int64
	Action      []byte
	Target      interface{}
	ControlView interface{}
}

func (c *NSActionCell) InitFromUnarchiver(u Unarchiver, class *Class) error {
	if err := c.NSCell.InitFromUnarchiver(u, class.Superclass); err != nil {
		return err
	}
	if class.Version != 17 {
		return fmt.Errorf("NSActionCell: unsupported version %d", class.Version)
	}
	tagVals, err := u.DecodeValuesOfTypes([]byte("i"), []byte(":"))
	if err != nil {
		return err
	}
	c.Tag = tagVals[0].(int64)
	c.Action, _ = tagVals[1].([]byte)
	c.Target, err = u.DecodeValueOfType([]byte("@"))
	if err != nil {
		return err
	}
	c.ControlView, err = u.DecodeValueOfType([]byte("@"))
	if err != nil {
		return err
	}
	return nil
}
func (c *NSActionCell) AllowsExtraData() bool               { return false }
func (c *NSActionCell) AddExtraField(_ *TypedGroup) error   { return nil }
func (c *NSActionCell) FormatLines(seen map[uintptr]bool) []string {
	body := c.NSCell.cellBodyLines(seen)
	body = append(body, c.actionCellBodyLines(seen)...)
	return formatHeaderBody("NSActionCell", body)
}

func (c *NSActionCell) actionCellBodyLines(seen map[uintptr]bool) []string {
	var body []string
	if c.Tag != 0 {
		body = append(body, fmt.Sprintf("tag: %d", c.Tag))
	}
	if c.Action != nil {
		body = append(body, fmt.Sprintf("action: %q", c.Action))
	}
	if c.Target != nil {
		body = append(body, fmt.Sprintf("target: <%s>", objectClassNameAppKit(c.Target)))
	}
	cvDesc := "None"
	if c.ControlView != nil {
		cvDesc = fmt.Sprintf("<%s>", objectClassNameAppKit(c.ControlView))
	}
	body = append(body, "control view: "+cvDesc)
	return body
}

// ---- NSButtonImageSource ----

type NSButtonImageSource struct {
	NSObject
	ResourceName string
}

func (b *NSButtonImageSource) InitFromUnarchiver(u Unarchiver, class *Class) error {
	if err := b.NSObject.InitFromUnarchiver(u, class.Superclass); err != nil {
		return err
	}
	if class.Version != 3 {
		return fmt.Errorf("NSButtonImageSource: unsupported version %d", class.Version)
	}
	val, err := u.DecodeValueOfType([]byte("@"))
	if err != nil {
		return err
	}
	ns, ok := val.(*NSString)
	if !ok {
		return fmt.Errorf("NSButtonImageSource: resource_name must be *NSString, got %T", val)
	}
	b.ResourceName = ns.Value
	return nil
}
func (b *NSButtonImageSource) AllowsExtraData() bool               { return false }
func (b *NSButtonImageSource) AddExtraField(_ *TypedGroup) error   { return nil }
func (b *NSButtonImageSource) FormatLines(_ map[uintptr]bool) []string {
	return []string{fmt.Sprintf("NSButtonImageSource(resource_name=%q)", b.ResourceName)}
}

// ---- NSButtonCell ----

type NSButtonCell struct {
	NSActionCell
	ShortsUnknown [2]int64
	ButtonType    int64
	TypeFlags     int64
	Flags         int64
	KeyEquivalent string
	Image1        interface{}
	Image2OrFont  interface{}
}

var buttonTypeNames = map[int64]string{
	0: "momentary_light", 1: "push_on_push_off", 2: "toggle",
	3: "switch", 4: "radio", 5: "momentary_change", 6: "on_off",
	7: "momentary_push_in", 8: "accelerator", 9: "multi_level_accelerator",
}

func (b *NSButtonCell) InitFromUnarchiver(u Unarchiver, class *Class) error {
	if err := b.NSActionCell.InitFromUnarchiver(u, class.Superclass); err != nil {
		return err
	}
	if class.Version != 63 {
		return fmt.Errorf("NSButtonCell: unsupported version %d", class.Version)
	}
	vals, err := u.DecodeValuesOfTypes(
		[]byte("s"), []byte("s"), []byte("i"), []byte("i"),
		[]byte("@"), []byte("@"), []byte("@"), []byte("@"), []byte("@"),
	)
	if err != nil {
		return err
	}
	b.ShortsUnknown[0] = vals[0].(int64)
	b.ShortsUnknown[1] = vals[1].(int64)
	buttonType := vals[2].(int64)
	b.ButtonType = buttonType & 0xffffff
	b.TypeFlags = buttonType & int64(^uint32(0xffffff))
	b.Flags = vals[3].(int64) & 0xffffffff
	// vals[4] is string_1 (empty/nil)
	if ns, ok := vals[5].(*NSString); ok {
		b.KeyEquivalent = ns.Value
	}
	b.Image1 = vals[6]
	b.Image2OrFont = vals[7]
	// vals[8] is unknown_object (must be nil)
	return nil
}
func (b *NSButtonCell) AllowsExtraData() bool               { return false }
func (b *NSButtonCell) AddExtraField(_ *TypedGroup) error   { return nil }
func (b *NSButtonCell) FormatLines(seen map[uintptr]bool) []string {
	body := b.NSActionCell.NSCell.cellBodyLines(seen)
	body = append(body, b.NSActionCell.actionCellBodyLines(seen)...)
	body = append(body, fmt.Sprintf("unknown shorts: (%d, %d)", b.ShortsUnknown[0], b.ShortsUnknown[1]))
	body = append(body, "button type: "+lookupName(buttonTypeNames, b.ButtonType))
	if b.TypeFlags != 0 {
		body = append(body, fmt.Sprintf("button type flags: 0x%08x", b.TypeFlags))
	}
	body = append(body, fmt.Sprintf("button flags: 0x%08x", b.Flags))
	if b.KeyEquivalent != "" {
		body = append(body, fmt.Sprintf("key equivalent: %q", b.KeyEquivalent))
	}
	if b.Image1 != nil {
		body = append(body, FormatValueWithPrefix(b.Image1, "image 1: ", seen)...)
	}
	if b.Image2OrFont != nil {
		body = append(body, FormatValueWithPrefix(b.Image2OrFont, "image 2 or font: ", seen)...)
	}
	return formatHeaderBody("NSButtonCell", body)
}

// ---- NSTextFieldCell ----

type NSTextFieldCell struct {
	NSActionCell
	DrawsBackground bool
	BackgroundColor interface{}
	TextColor       interface{}
}

func (c *NSTextFieldCell) InitFromUnarchiver(u Unarchiver, class *Class) error {
	if err := c.NSActionCell.InitFromUnarchiver(u, class.Superclass); err != nil {
		return err
	}
	if class.Version != 61 && class.Version != 62 {
		return fmt.Errorf("NSTextFieldCell: unsupported version %d", class.Version)
	}
	vals, err := u.DecodeValuesOfTypes([]byte("c"), []byte("@"), []byte("@"))
	if err != nil {
		return err
	}
	c.DrawsBackground = vals[0].(int64) != 0
	c.BackgroundColor = vals[1]
	c.TextColor = vals[2]
	return nil
}
func (c *NSTextFieldCell) AllowsExtraData() bool               { return false }
func (c *NSTextFieldCell) AddExtraField(_ *TypedGroup) error   { return nil }
func (c *NSTextFieldCell) FormatLines(seen map[uintptr]bool) []string {
	body := c.NSActionCell.NSCell.cellBodyLines(seen)
	body = append(body, c.NSActionCell.actionCellBodyLines(seen)...)
	body = append(body, c.textFieldCellBodyLines(seen)...)
	return formatHeaderBody("NSTextFieldCell", body)
}

func (c *NSTextFieldCell) textFieldCellBodyLines(seen map[uintptr]bool) []string {
	var body []string
	body = append(body, fmt.Sprintf("draws background: %v", c.DrawsBackground))
	if col, ok := c.BackgroundColor.(*NSColor); ok {
		body = append(body, fmt.Sprintf("background color: %s", col))
	}
	if col, ok := c.TextColor.(*NSColor); ok {
		body = append(body, fmt.Sprintf("text color: %s", col))
	}
	return body
}

// ---- NSComboBoxCell ----

type NSComboBoxCell struct {
	NSTextFieldCell
	NumberOfVisibleItems int64
	Values               []interface{}
	ComboBox             interface{}
	ButtonCell           interface{}
	TableView            interface{}
}

func (c *NSComboBoxCell) InitFromUnarchiver(u Unarchiver, class *Class) error {
	if err := c.NSTextFieldCell.InitFromUnarchiver(u, class.Superclass); err != nil {
		return err
	}
	if class.Version != 2 {
		return fmt.Errorf("NSComboBoxCell: unsupported version %d", class.Version)
	}
	vals, err := u.DecodeValuesOfTypes([]byte("i"), []byte("c"), []byte("c"), []byte("c"))
	if err != nil {
		return err
	}
	c.NumberOfVisibleItems = vals[0].(int64)
	// vals[1],[2],[3] are booleans that must be 1,1,0
	arrVal, err := u.DecodeValueOfType([]byte("@"))
	if err != nil {
		return err
	}
	if arr, ok := arrVal.(*NSArray); ok {
		c.Values = arr.Elements
	}
	if _, err := u.DecodeValueOfType([]byte("@")); err != nil { // unknown_object (nil)
		return err
	}
	c.ComboBox, err = u.DecodeValueOfType([]byte("@"))
	if err != nil {
		return err
	}
	c.ButtonCell, err = u.DecodeValueOfType([]byte("@"))
	if err != nil {
		return err
	}
	c.TableView, err = u.DecodeValueOfType([]byte("@"))
	if err != nil {
		return err
	}
	return nil
}
func (c *NSComboBoxCell) AllowsExtraData() bool               { return false }
func (c *NSComboBoxCell) AddExtraField(_ *TypedGroup) error   { return nil }
func (c *NSComboBoxCell) FormatLines(seen map[uintptr]bool) []string {
	body := c.NSTextFieldCell.NSActionCell.NSCell.cellBodyLines(seen)
	body = append(body, c.NSTextFieldCell.NSActionCell.actionCellBodyLines(seen)...)
	body = append(body, c.NSTextFieldCell.textFieldCellBodyLines(seen)...)
	body = append(body, fmt.Sprintf("number of visible items: %d", c.NumberOfVisibleItems))
	if len(c.Values) > 0 {
		if len(c.Values) == 1 {
			body = append(body, "1 value:")
		} else {
			body = append(body, fmt.Sprintf("%d values:", len(c.Values)))
		}
		for _, v := range c.Values {
			for _, line := range FormatValue(v, seen) {
				body = append(body, "\t"+line)
			}
		}
	}
	body = append(body, fmt.Sprintf("combo box: <%s>", objectClassNameAppKit(c.ComboBox)))
	body = append(body, FormatValueWithPrefix(c.ButtonCell, "button cell: ", seen)...)
	body = append(body, FormatValueWithPrefix(c.TableView, "table view: ", seen)...)
	return formatHeaderBody("NSComboBoxCell", body)
}

// ---- NSTableHeaderCell ----

type NSTableHeaderCell struct {
	NSTextFieldCell
}

func (c *NSTableHeaderCell) InitFromUnarchiver(u Unarchiver, class *Class) error {
	if err := c.NSTextFieldCell.InitFromUnarchiver(u, class.Superclass); err != nil {
		return err
	}
	if class.Version != 28 {
		return fmt.Errorf("NSTableHeaderCell: unsupported version %d", class.Version)
	}
	return nil
}
func (c *NSTableHeaderCell) AllowsExtraData() bool               { return false }
func (c *NSTableHeaderCell) AddExtraField(_ *TypedGroup) error   { return nil }
func (c *NSTableHeaderCell) FormatLines(seen map[uintptr]bool) []string {
	body := c.NSTextFieldCell.NSActionCell.NSCell.cellBodyLines(seen)
	body = append(body, c.NSTextFieldCell.NSActionCell.actionCellBodyLines(seen)...)
	body = append(body, c.NSTextFieldCell.textFieldCellBodyLines(seen)...)
	return formatHeaderBody("NSTableHeaderCell", body)
}

// ---- NSResponder ----

type NSResponder struct {
	NSObject
	NextResponder interface{}
}

func (r *NSResponder) InitFromUnarchiver(u Unarchiver, class *Class) error {
	if err := r.NSObject.InitFromUnarchiver(u, class.Superclass); err != nil {
		return err
	}
	if class.Version != 0 {
		return fmt.Errorf("NSResponder: unsupported version %d", class.Version)
	}
	var err error
	r.NextResponder, err = u.DecodeValueOfType([]byte("@"))
	return err
}
func (r *NSResponder) AllowsExtraData() bool               { return false }
func (r *NSResponder) AddExtraField(_ *TypedGroup) error   { return nil }
func (r *NSResponder) FormatLines(_ map[uintptr]bool) []string {
	nrd := "None"
	if r.NextResponder != nil {
		nrd = fmt.Sprintf("<%s>", objectClassNameAppKit(r.NextResponder))
	}
	return []string{fmt.Sprintf("NSResponder(next_responder=%s)", nrd)}
}

// ---- NSView ----

type NSView struct {
	NSResponder
	Flags                  int64
	Subviews               []interface{}
	RegisteredDraggedTypes []string
	Frame                  NSRect
	Bounds                 NSRect
	Superview              interface{}
	ContentView            interface{}
}

func (v *NSView) InitFromUnarchiver(u Unarchiver, class *Class) error {
	if err := v.NSResponder.InitFromUnarchiver(u, class.Superclass); err != nil {
		return err
	}
	if class.Version != 41 {
		return fmt.Errorf("NSView: unsupported version %d", class.Version)
	}
	flagsVal, err := u.DecodeValueOfType([]byte("i"))
	if err != nil {
		return fmt.Errorf("NSView flags: %w", err)
	}
	v.Flags = flagsVal.(int64)

	vals, err := u.DecodeValuesOfTypes(
		[]byte("@"), []byte("@"), []byte("@"), []byte("@"),
		[]byte("f"), []byte("f"), []byte("f"), []byte("f"),
		[]byte("f"), []byte("f"), []byte("f"), []byte("f"),
	)
	if err != nil {
		return fmt.Errorf("NSView objects/frame/bounds: %w", err)
	}
	if arr, ok := vals[0].(*NSArray); ok && arr != nil {
		v.Subviews = arr.Elements
	}
	// vals[1] and vals[2] must be nil
	if vals[3] != nil {
		if set, ok := vals[3].(*NSSet); ok {
			for _, elem := range set.Elements {
				if ns, ok := elem.(*NSString); ok {
					v.RegisteredDraggedTypes = append(v.RegisteredDraggedTypes, ns.Value)
				}
			}
		}
	}
	frameX, _ := vals[4].(float32)
	frameY, _ := vals[5].(float32)
	frameW, _ := vals[6].(float32)
	frameH, _ := vals[7].(float32)
	boundsX, _ := vals[8].(float32)
	boundsY, _ := vals[9].(float32)
	boundsW, _ := vals[10].(float32)
	boundsH, _ := vals[11].(float32)
	v.Frame = NSRect{Origin: NSPoint{X: frameX, Y: frameY}, Size: NSSize{Width: frameW, Height: frameH}}
	v.Bounds = NSRect{Origin: NSPoint{X: boundsX, Y: boundsY}, Size: NSSize{Width: boundsW, Height: boundsH}}

	v.Superview, err = u.DecodeValueOfType([]byte("@"))
	if err != nil {
		return fmt.Errorf("NSView superview: %w", err)
	}
	if _, err := u.DecodeValueOfType([]byte("@")); err != nil { // obj6 (nil)
		return fmt.Errorf("NSView obj6: %w", err)
	}
	v.ContentView, err = u.DecodeValueOfType([]byte("@"))
	if err != nil {
		return fmt.Errorf("NSView content_view: %w", err)
	}
	if _, err := u.DecodeValueOfType([]byte("@")); err != nil { // obj8 (nil)
		return fmt.Errorf("NSView obj8: %w", err)
	}
	return nil
}
func (v *NSView) AllowsExtraData() bool               { return false }
func (v *NSView) AddExtraField(_ *TypedGroup) error   { return nil }
func (v *NSView) FormatLines(seen map[uintptr]bool) []string {
	body := v.viewBodyLines(seen)
	return formatHeaderBody("NSView", body)
}

func (v *NSView) viewBodyLines(seen map[uintptr]bool) []string {
	var body []string
	nrd := "None"
	if v.NextResponder != nil {
		nrd = fmt.Sprintf("<%s>", objectClassNameAppKit(v.NextResponder))
	}
	body = append(body, "next responder: "+nrd)
	body = append(body, fmt.Sprintf("flags: 0x%08x", v.Flags))
	if len(v.Subviews) > 0 {
		if len(v.Subviews) == 1 {
			body = append(body, "1 subview:")
		} else {
			body = append(body, fmt.Sprintf("%d subviews:", len(v.Subviews)))
		}
		for _, sv := range v.Subviews {
			for _, line := range FormatValue(sv, seen) {
				body = append(body, "\t"+line)
			}
		}
	}
	if len(v.RegisteredDraggedTypes) > 0 {
		body = append(body, fmt.Sprintf("%d registered dragged types:", len(v.RegisteredDraggedTypes)))
		for _, t := range v.RegisteredDraggedTypes {
			body = append(body, fmt.Sprintf("\t%q", t))
		}
	}
	body = append(body, fmt.Sprintf("frame: %s", v.Frame))
	body = append(body, fmt.Sprintf("bounds: %s", v.Bounds))
	svd := "None"
	if v.Superview != nil {
		svd = fmt.Sprintf("<%s>", objectClassNameAppKit(v.Superview))
	}
	body = append(body, "superview: "+svd)
	if v.ContentView != nil {
		body = append(body, fmt.Sprintf("content view: <%s>", objectClassNameAppKit(v.ContentView)))
	}
	return body
}

// ---- NSControl ----

type NSControl struct {
	NSView
	Int1  int64
	Bool1 bool
	Cell  interface{}
}

func (c *NSControl) InitFromUnarchiver(u Unarchiver, class *Class) error {
	if err := c.NSView.InitFromUnarchiver(u, class.Superclass); err != nil {
		return err
	}
	if class.Version != 41 {
		return fmt.Errorf("NSControl: unsupported version %d", class.Version)
	}
	vals, err := u.DecodeValuesOfTypes([]byte("i"), []byte("c"), []byte("c"), []byte("@"))
	if err != nil {
		return err
	}
	c.Int1 = vals[0].(int64)
	c.Bool1 = vals[1].(int64) != 0
	// vals[2] is int_3 (must be 0)
	c.Cell = vals[3]
	return nil
}
func (c *NSControl) AllowsExtraData() bool               { return false }
func (c *NSControl) AddExtraField(_ *TypedGroup) error   { return nil }
func (c *NSControl) FormatLines(seen map[uintptr]bool) []string {
	body := c.NSView.viewBodyLines(seen)
	body = append(body, fmt.Sprintf("unknown int 1: %d", c.Int1))
	body = append(body, fmt.Sprintf("unknown boolean 1: %v", c.Bool1))
	body = append(body, FormatValueWithPrefix(c.Cell, "cell: ", seen)...)
	return formatHeaderBody("NSControl", body)
}
