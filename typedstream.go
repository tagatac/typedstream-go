package typedstream

import "io"

type (
	ReaderDecoder interface {
		Decode(r io.Reader) (string, error)
	}

	readerDecoder struct{}
)

func NewReaderDecoder() ReaderDecoder {
	return &readerDecoder{}
}

func (rd *readerDecoder) Decode(r io.Reader) (string, error) {
	bytes, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	u := NewUnarchiver(bytes)
	_, err = u.DecodeAll()
	if err != nil {
		return "", err
	}
	return "", nil
}
