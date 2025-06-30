package trailer

import "io"

type PackFunc func(interface{}) (valueRaw []byte, length int, err error)
type UnpackFunc func(io.Reader) (value interface{}, length int, err error)
type GetLengthFunc func() int

func Pack(interface{}) ([]byte, int, error) {
	//not implemented
	return []byte{}, 0, nil
}

func Unpack(io.Reader) (interface{}, int, error) {
	//not implemented
	return nil, 0, nil
}

func GetLength() int {
	//not implemented
	return 0
}
