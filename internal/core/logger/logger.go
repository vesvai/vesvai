package logger

import (
	"fmt"
	"sync"
	"time"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

func ParseLevel(s string) Level {
	switch s {
	case "DEBUG":
		return LevelDebug
	case "INFO":
		return LevelInfo
	case "WARN":
		return LevelWarn
	case "ERROR":
		return LevelError
	default:
		return LevelDebug
	}
}

type Record struct {
	ID        int64     `json:"id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Level     Level     `json:"level"`
	Message   string    `json:"message"`
}

type Handler interface {
	Write(rec Record) error
	Close() error
}

const maxBufferedRecords = 1000

type Logger struct {
	mu       sync.RWMutex
	minLevel Level
	handlers []Handler
	records  []Record
}

func New(minLevel Level, handlers ...Handler) *Logger {
	return &Logger{
		minLevel: minLevel,
		handlers: handlers,
		records:  make([]Record, 0, 64),
	}
}

func (l *Logger) AddHandler(h Handler) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.handlers = append(l.handlers, h)
}

func (l *Logger) log(level Level, msg string) {
	if level < l.minLevel {
		return
	}

	rec := Record{
		Timestamp: time.Now(),
		Level:     level,
		Message:   msg,
	}

	l.mu.Lock()
	l.records = append(l.records, rec)
	if len(l.records) > maxBufferedRecords {
		l.records = append([]Record(nil), l.records[len(l.records)-maxBufferedRecords:]...)
	}
	l.mu.Unlock()

	l.mu.RLock()
	defer l.mu.RUnlock()

	for _, h := range l.handlers {
		_ = h.Write(rec)
	}
}

func (l *Logger) Records() []Record {
	l.mu.RLock()
	defer l.mu.RUnlock()

	out := make([]Record, len(l.records))
	copy(out, l.records)
	return out
}

func (l *Logger) Debug(msg string) { l.log(LevelDebug, msg) }
func (l *Logger) Info(msg string)  { l.log(LevelInfo, msg) }
func (l *Logger) Warn(msg string)  { l.log(LevelWarn, msg) }
func (l *Logger) Error(msg string) { l.log(LevelError, msg) }

func (l *Logger) Fdebug(msg string, args ...any) { l.log(LevelDebug, fmt.Sprintf(msg, args...)) }
func (l *Logger) Finfo(msg string, args ...any)  { l.log(LevelInfo, fmt.Sprintf(msg, args...)) }
func (l *Logger) Fwarn(msg string, args ...any)  { l.log(LevelWarn, fmt.Sprintf(msg, args...)) }
func (l *Logger) Ferror(msg string, args ...any) { l.log(LevelError, fmt.Sprintf(msg, args...)) }

func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, h := range l.handlers {
		_ = h.Close()
	}
	return nil
}
