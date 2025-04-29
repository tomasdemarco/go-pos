package header

import (
	"io"
)

type PackFunc func(interface{}) (valueRaw []byte, length int, err error)
type UnpackFunc func(r io.Reader) (value interface{}, length int, err error)

func Pack(interface{}) ([]byte, int, error) {
	//not implemented
	return []byte{}, 0, nil
}

func Unpack(r io.Reader) (interface{}, int, error) {
	//not implemented
	return nil, 0, nil
}
