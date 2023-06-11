package typedstream

type (
	Unarchiver interface {
		DecodeAll() ([]TypedGroup, error)
	}

	TypedGroup interface {
		Add([]byte, interface{})
		Header() string
		Body() []string
	}

	typedGroup struct {
		members []encodingAndValue
	}

	encodingAndValue struct {
		encoding []byte
		value    interface{}
	}

	unarchiver struct {
		bytes []byte
	}
)

func NewTypedGroup() TypedGroup {
	return &typedGroup{}
}

func (tg *typedGroup) Add(encoding []byte, value interface{}) {
	tg.members = append(tg.members, encodingAndValue{encoding, value})
}

func (tg *typedGroup) Header() string {
	return "group"
}

func (tg *typedGroup) Body() []string {
	return []string{"body"}
}

func NewUnarchiver(bytes []byte) Unarchiver {
	return &unarchiver{bytes: bytes}
}

func (u *unarchiver) DecodeAll() ([]TypedGroup, error) {
	return []TypedGroup{NewTypedGroup()}, nil
}
