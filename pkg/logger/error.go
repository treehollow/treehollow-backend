package logger

import (
	"github.com/google/uuid"
	"log"
)

type LogLevel int8

const (
	DEBUG LogLevel = 0
	INFO  LogLevel = 1
	WARN  LogLevel = 2
	ERROR LogLevel = 3
	FATAL LogLevel = 4
)

func (l *LogLevel) ToString() string {
	switch *l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO "
	case WARN:
		return "WARN "
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	}
	return "UNKNOWN"
}

type InternalError struct {
	UUID        string
	Err         error
	InternalMsg string
	DisplayMsg  string
	level       LogLevel
}

func NewSimpleError(internalMsg string, displayMsg string, level LogLevel) *InternalError {
	return newInternalError(nil, internalMsg, displayMsg, level)
}

func NewError(err error, internalMsg string, displayMsg string) *InternalError {
	return newInternalError(err, internalMsg, displayMsg, ERROR)
}

func newInternalError(err error, internalMsg string, displayMsg string, level LogLevel) *InternalError {
	id := ""

	if level > INFO {
		id = uuid.New().String()
		displayMsg = displayMsg + "(ErrorID: " + id + ")"
	}

	return &InternalError{
		UUID:        id,
		Err:         err,
		InternalMsg: internalMsg,
		DisplayMsg:  displayMsg,
		level:       level,
	}
}

func (e *InternalError) Log() {
	if e.level >= WARN {
		log.Printf("[%s](%s): %s; Err=%s\n", e.level.ToString(), e.UUID, e.InternalMsg, e.Err)
	}
}
