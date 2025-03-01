package logger

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/tomasdemarco/iso8583/message"
	ctx "go-pos/context"
	"log"
	"runtime"
	"strings"
	"time"
)

func (l *Logger) ISOMessage(c *ctx.Context, message *message.Message, service ...string) (err error) {
	if l.Level <= Info {
		mti, err := message.GetField("000")
		if err != nil {
			return err
		}

		fields := make(map[string]string)
		fields["000"] = mti

		for _, field := range message.Bitmap {
			if field != "000" && field != "001" {
				value, err := message.GetField(field)
				if err != nil {
					return err
				}

				fields[field] = value
			}
		}

		jsonAuth, err := json.Marshal(fields)
		if err != nil {
			return err
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("{\"time\":\"%s\"", time.Now().Format("2006-01-02 15:04:05.000")))

		if c != nil {
			if c.Id != uuid.Nil {
				sb.WriteString(fmt.Sprintf(",\"id\":\"%s\"", c.Id))
			}
		}

		if service != nil && len(service) > 0 {
			sb.WriteString(fmt.Sprintf(",\"service\":\"%s\"", service[0]))
		}

		sb.WriteString(fmt.Sprintf(",\"isoMsg\":%s}", string(jsonAuth)))

		log.Printf("%s", sb.String())
	}

	return nil
}

func (l *Logger) Info(c *ctx.Context, logType LogType, i interface{}, service ...string) {
	if l.Level <= Info {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("{\"time\":\"%s\"", time.Now().Format("2006-01-02 15:04:05.000")))

		if c != nil {
			if c.Id != uuid.Nil {
				sb.WriteString(fmt.Sprintf(",\"id\":\"%s\"", c.Id))
			}
		}

		if service != nil && len(service) > 0 {
			sb.WriteString(fmt.Sprintf(",\"service\":\"%s\"", service[0]))
		}

		sb.WriteString(fmt.Sprintf(",\"%s\":\"%s\"}", logType.String(), i))

		log.Printf("%s", sb.String())
	}
}

func (l *Logger) Debug(c *ctx.Context, logType LogType, i interface{}, service ...string) {
	if l.Level == Debug {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("{\"time\":\"%s\"", time.Now().Format("2006-01-02 15:04:05.000")))

		if c != nil {
			if c.Id != uuid.Nil {
				sb.WriteString(fmt.Sprintf(",\"id\":\"%s\"", c.Id))
			}
		}

		if service != nil && len(service) > 0 {
			sb.WriteString(fmt.Sprintf(",\"service\":\"%s\"", service[0]))
		}

		sb.WriteString(fmt.Sprintf(",\"%s\":%s}", logType.String(), i))

		log.Printf("%s", sb.String())
	}
}

func (l *Logger) Error(c *ctx.Context, err error, service ...string) {
	if l.Level <= Error {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("{\"time\":\"%s\"", time.Now().Format("2006-01-02 15:04:05.000")))

		if c != nil {
			if c.Id != uuid.Nil {
				sb.WriteString(fmt.Sprintf(",\"id\":\"%s\"", c.Id))
			}
		}

		if service != nil && len(service) > 0 {
			sb.WriteString(fmt.Sprintf(",\"service\":\"%s\"", service[0]))
		}

		if l.ErrorDetail {
			pc, file, line, _ := runtime.Caller(1)

			value, errMarshal := json.Marshal(fmt.Sprintf("%v - %s[%s:%d]", err, runtime.FuncForPC(pc).Name(), file, line))
			if errMarshal != nil {
				l.Error(c, errMarshal, "logger")
			}

			sb.WriteString(fmt.Sprintf(",\"error\":%s}", string(value)))
		} else {
			sb.WriteString(fmt.Sprintf(",\"error\":\"%v\"}", err))
		}

		log.Printf("%s", sb.String())
	}
}

func (l *Logger) Panic(c *ctx.Context, err error, panic []byte, service ...string) {
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

		if c != nil {
			if c.Id != uuid.Nil {
				sb.WriteString(fmt.Sprintf(",\"id\":\"%s\"", c.Id))
			}
		}

		if service != nil && len(service) > 0 {
			sb.WriteString(fmt.Sprintf(",\"service\":\"%s\"", service[0]))
		}

		sb.WriteString(fmt.Sprintf(",\"panic\":\"%v\",\"stack\":[%s]}", err, sbStack.String()))

		log.Printf("%s", sb.String())
	}
}
