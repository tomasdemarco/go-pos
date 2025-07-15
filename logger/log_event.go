package logger

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	ctx "github.com/tomasdemarco/go-pos/context"
	"log"
	"runtime"
	"strings"
	"time"
)

func (l *Logger) Info(c ctx.Context, logType LogType, i interface{}) {
	if l.Level <= Info {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("{\"time\":\"%s\"", time.Now().Format("2006-01-02 15:04:05.000")))

		if l.Service != nil {
			sb.WriteString(fmt.Sprintf(",\"service\":\"%s\"", *l.Service))
		}

		if c != nil && c.GetId() != uuid.Nil {
			sb.WriteString(fmt.Sprintf(",\"id\":\"%s\"", c.GetId().String()))
		}

		if logType == IsoMessage {
			sb.WriteString(fmt.Sprintf(",\"%s\":%s", logType.String(), i))
		} else {
			sb.WriteString(fmt.Sprintf(",\"%s\":\"%s\"", logType.String(), i))
		}

		if c != nil {
			sb.WriteString(c.Attributes().String())
		}

		sb.WriteString("}")

		log.Printf("%s", sb.String())
	}
}

func (l *Logger) Debug(c ctx.Context, i interface{}) {
	if l.Level == Debug {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("{\"time\":\"%s\"", time.Now().Format("2006-01-02 15:04:05.000")))

		if l.Service != nil {
			sb.WriteString(fmt.Sprintf(",\"service\":\"%s\"", *l.Service))
		}

		if c != nil && c.GetId() != uuid.Nil {
			sb.WriteString(fmt.Sprintf(",\"id\":\"%s\"", c.GetId().String()))
		}

		sb.WriteString(fmt.Sprintf(",\"debug\":\"%s\"", i))

		if c != nil {
			sb.WriteString(c.Attributes().String())
		}

		sb.WriteString("}")

		log.Printf("%s", sb.String())
	}
}

func (l *Logger) Error(c ctx.Context, err error) {
	if l.Level <= Error {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("{\"time\":\"%s\"", time.Now().Format("2006-01-02 15:04:05.000")))

		if l.Service != nil {
			sb.WriteString(fmt.Sprintf(",\"service\":\"%s\"", *l.Service))
		}

		if c != nil && c.GetId() != uuid.Nil {
			sb.WriteString(fmt.Sprintf(",\"id\":\"%s\"", c.GetId().String()))
		}

		if l.Level == Debug {
			pc, file, line, _ := runtime.Caller(1)

			value, errMarshal := json.Marshal(fmt.Sprintf("%v - %s[%s:%d]", err, runtime.FuncForPC(pc).Name(), file, line))
			if errMarshal != nil {
				l.Error(c, errMarshal)
			}

			sb.WriteString(fmt.Sprintf(",\"error\":%s", string(value)))
		} else {
			sb.WriteString(fmt.Sprintf(",\"error\":\"%v\"", err))
		}

		if c != nil {
			sb.WriteString(c.Attributes().String())
		}

		sb.WriteString("}")

		log.Printf("%s", sb.String())
	}
}

func (l *Logger) Panic(c ctx.Context, err error, panic []byte) {
	if l.Level <= Fatal {
		var sbStack strings.Builder
		stack := strings.Split(strings.Replace(string(panic), "\t", "", -1), "\n")
		for i, line := range stack {
			if line != "" {
				if i == len(stack)-2 {
					sbStack.WriteString(fmt.Sprintf("\"%s\"", line))
				} else {
					sbStack.WriteString(fmt.Sprintf("\"%s\",", line))
				}
			}
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("{\"time\":\"%s\"", time.Now().Format("2006-01-02 15:04:05.000")))

		if l.Service != nil {
			sb.WriteString(fmt.Sprintf(",\"service\":\"%s\"", *l.Service))
		}

		if c != nil && c.GetId() != uuid.Nil {
			sb.WriteString(fmt.Sprintf(",\"id\":\"%s\"", c.GetId().String()))
		}

		sb.WriteString(fmt.Sprintf(",\"panic\":\"%v\",\"stack\":[%s]", err, sbStack.String()))

		if c != nil {
			sb.WriteString(c.Attributes().String())
		}

		sb.WriteString("}")

		log.Printf("%s", sb.String())
	}
}
