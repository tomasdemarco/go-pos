package logger

import (
	"encoding/json"
	"fmt"
)

type LogType int

const (
	Message LogType = iota
	IsoPack
	IsoUnpack
	IsoMessage
	Request
	Response
)

var logTypeStrings = [...]string{
	Message:    "message",
	IsoPack:    "pack",
	IsoUnpack:  "unpack",
	IsoMessage: "isoMsg",
	Request:    "request",
	Response:   "response",
}

// String return string
func (l *LogType) String() string {
	return logTypeStrings[*l]
}

// EnumIndex return index
func (l *LogType) EnumIndex() int {
	return int(*l)
}

// UnmarshalJSON override default unmarshal json
func (l *LogType) UnmarshalJSON(b []byte) error {
	var j string
	err := json.Unmarshal(b, &j)
	if err != nil {
		return err
	}

	for i, str := range logTypeStrings {
		if str == j {
			*l = LogType(i)
			return nil
		}
	}

	return fmt.Errorf("invalid logtype: %s", j)
}

func (l *LogType) IsValid() bool {
	if int(*l) >= 0 && int(*l) < len(logTypeStrings) {
		value := logTypeStrings[*l]
		if value != "" {
			return true
		}
	}
	return false
}
