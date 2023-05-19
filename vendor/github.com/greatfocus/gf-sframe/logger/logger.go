package logger

import (
	"bytes"
	"log"
)

const (
	WARN  = "WARN"
	ERROR = "ERROR"
	FATAL = "FATAL"
	INFO  = "INFO"
)

// Logger interface
type Logger interface {
	Warn(v any)
	Error(v any)
	Fatal(v any)
	Info(v any)
}

// NewLogger create instance of the logger types
func NewLogger(service string) Logger {
	l := &logger{
		warnLogger:  newLevel(service, WARN),
		errorLogger: newLevel(service, ERROR),
		fatalLogger: newLevel(service, FATAL),
		infoLogger:  newLevel(service, INFO),
	}
	return l
}

// logger struct
type logger struct {
	warnLogger  *log.Logger
	errorLogger *log.Logger
	fatalLogger *log.Logger
	infoLogger  *log.Logger
}

func (l *logger) Warn(v any) {
	l.warnLogger.Println(v)
}

func (l *logger) Error(v any) {
	l.errorLogger.Println(v)
}

func (l *logger) Fatal(v any) {
	l.errorLogger.Fatal(v)
}

func (l *logger) Info(v any) {
	l.infoLogger.Println(v)
}

// newLevel creates new log level
func newLevel(service string, level string) *log.Logger {
	var output bytes.Buffer
	var prefix = level + ": " + service + ": "
	var flag = log.Ldate | log.Ltime | log.Lshortfile
	logger := log.New(&output, prefix, flag)
	return logger
}
