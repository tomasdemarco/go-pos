package logger

import (
	"encoding/json"
	"fmt"
)

type LogLevel int

const (
	Debug LogLevel = iota
	Info
	Warn
	Error
	Fatal
)

var logLevelStrings = [...]string{
	Debug: "Debug",
	Info:  "Info",
	Warn:  "Warn",
	Error: "Error",
	Fatal: "Fatal",
}

// String return string
func (l *LogLevel) String() string {
	return logLevelStrings[*l]
}

// EnumIndex return index
func (l *LogLevel) EnumIndex() int {
	return int(*l)
}

// UnmarshalJSON override default unmarshal json
func (l *LogLevel) UnmarshalJSON(b []byte) error {
	var j string
	err := json.Unmarshal(b, &j)
	if err != nil {
		return err
	}

	for i, str := range logLevelStrings {
		if str == j {
			*l = LogLevel(i)
			return nil
		}
	}

	return fmt.Errorf("invalid logLevel: %s", j)
}

func (l *LogLevel) IsValid() bool {
	if int(*l) >= 0 && int(*l) < len(logLevelStrings) {
		value := logLevelStrings[*l]
		if value != "" {
			return true
		}
	}
	return false
}
