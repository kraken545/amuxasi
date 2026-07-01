package log

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	}
	return "UNKNOWN"
}

type Logger struct {
	mu     sync.Mutex
	file   io.WriteCloser
	level  Level
	appLog []string
}

var global *Logger

func Init(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create log dir: %w", err)
	}
	path := filepath.Join(dir, "amuxasi.log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	global = &Logger{
		file:   f,
		level:  LevelDebug,
		appLog: make([]string, 0, 1000),
	}
	Info("logger initialized: %s", path)
	return nil
}

func Global() *Logger {
	return global
}

func Debug(format string, args ...interface{}) {
	if global != nil {
		global.log(LevelDebug, format, args...)
	}
}

func Info(format string, args ...interface{}) {
	if global != nil {
		global.log(LevelInfo, format, args...)
	}
}

func Warn(format string, args ...interface{}) {
	if global != nil {
		global.log(LevelWarn, format, args...)
	}
}

func Error(format string, args ...interface{}) {
	if global != nil {
		global.log(LevelError, format, args...)
	}
}

func (l *Logger) log(level Level, format string, args ...interface{}) {
	if level < l.level {
		return
	}
	msg := fmt.Sprintf(format, args...)
	ts := time.Now().Format("15:04:05.000")
	line := fmt.Sprintf("[%s] %s: %s", ts, level, msg)

	l.mu.Lock()
	defer l.mu.Unlock()

	l.appLog = append(l.appLog, line)
	if len(l.appLog) > 10000 {
		l.appLog = l.appLog[len(l.appLog)-5000:]
	}

	if l.file != nil {
		fmt.Fprintln(l.file, line)
	}
}

func (l *Logger) Lines() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]string, len(l.appLog))
	copy(out, l.appLog)
	return out
}

func Close() {
	if global != nil && global.file != nil {
		global.file.Close()
	}
}

func AgentLogDir(baseDir, workspaceName string) string {
	dir := filepath.Join(baseDir, workspaceName)
	os.MkdirAll(dir, 0755)
	return dir
}

func WriteAgentLog(dir, agentName, content string) error {
	path := filepath.Join(dir, agentName+".log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(content)
	return err
}
