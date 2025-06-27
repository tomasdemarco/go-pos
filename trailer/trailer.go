package trailer

type PackFunc func(interface{}) (valueRaw []byte, length int, err error)
type UnpackFunc func(msgRaw []byte) (value interface{}, length int, err error)

func Pack(interface{}) ([]byte, int, error) {
	//not implemented
	return []byte{}, 0, nil
}

func Unpack(msgRaw []byte) (interface{}, int, error) {
	//not implemented
	return nil, 0, nil
}
