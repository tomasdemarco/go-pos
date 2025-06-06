package logger

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	ctx "github.com/tomasdemarco/go-pos/context"
	"github.com/tomasdemarco/iso8583/message"
	"log"
	"runtime"
	"strings"
	"time"
)

func (l *Logger) ISOMessage(c *ctx.RequestContext, message *message.Message, service ...string) (err error) {
	if l.Level <= Info {
		mti, err := message.GetField("000")
		if err != nil {
			return err
		}

		fields := make(map[string]interface{})
		fields["000"] = mti

		for _, fldId := range message.Bitmap {
			if fldId != "000" && fldId != "001" {
				fld, err := message.GetField(fldId)
				if err != nil {
					return err
				}

				//if fld.Subfields != nil {
				//	val, err := message.GetSubfields(fldId)
				//	if err != nil {
				//	}
				//	fields[fldId] = val
				//} else {
				//}
				fields[fldId] = fld
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

func (l *Logger) Info(c *ctx.RequestContext, logType LogType, i interface{}, service ...string) {
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

func (l *Logger) Debug(c *ctx.RequestContext, i interface{}, service ...string) {
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

		sb.WriteString(fmt.Sprintf(",\"debug\":\"%s\"}", i))

		log.Printf("%s", sb.String())
	}
}

func (l *Logger) Error(c *ctx.RequestContext, err error, service ...string) {
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

		if l.Level == Debug {
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

func (l *Logger) Panic(c *ctx.RequestContext, err error, panic []byte, service ...string) {
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
