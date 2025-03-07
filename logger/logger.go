package logger

import (
	"encoding/json"
	"github.com/google/uuid"
	ctx "github.com/tomasdemarco/go-pos/context"
	"log"
	"time"
)

type Logger struct {
	Level       LogLevel
	ErrorDetail bool
}

func New(level LogLevel, errorDetail bool) *Logger {
	// Logs without flags
	log.SetFlags(0)

	return &Logger{
		level,
		errorDetail,
	}
}

// LogEntry define la estructura del log en formato JSON
type LogEntry struct {
	Time        string    `json:"time,omitempty"`
	Id          uuid.UUID `json:"id,omitempty"`
	RemoteAddr  string    `json:"remoteAddr,omitempty"`
	StatusCode  int       `json:"statusCode,omitempty"`
	ElapsedTime string    `json:"elapsedTime,omitempty"`
}

// CustomLogger es un middleware para loguear en formato JSON
func CustomLogger(c *ctx.Context) {
	//gin.HandlerFunc {
	//	return func(c *ctx.Context){

	start := time.Now()

	// Crea la entrada del log
	entry := LogEntry{
		Time: start.Format("2006-01-02 15:04:05.000"),
		Id:   c.Id,
		//RemoteAddr:  c.RemoteAddr.String(),
		ElapsedTime: time.Since(start).String(),
	}

	logJSON(entry)
}

//}

// logJSON convierte la entrada del log a JSON y la imprime
func logJSON(entry LogEntry) {
	jsonData, err := json.Marshal(entry)
	if err != nil {
		log.Fatalf("failed to marshal JSON: %v", err)
	}
	log.Println(string(jsonData))
}
