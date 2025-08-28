package logger

import (
	"fmt"
	"log"
	"os"
)

var (
	debugLogger = log.New(os.Stdout, "\033[32m[D]\033[0m ", log.LstdFlags|log.Lshortfile)
	infoLogger  = log.New(os.Stdout, "\033[32m[I]\033[0m ", log.LstdFlags|log.Lshortfile)
	warnLogger  = log.New(os.Stdout, "\033[33m[W]\033[0m ", log.LstdFlags|log.Lshortfile)
	errorLogger = log.New(os.Stdout, "\033[31m[E]\033[0m ", log.LstdFlags|log.Lshortfile)
	panicLogger = log.New(os.Stdout, "\033[35m[P]\033[0m ", log.LstdFlags|log.Lshortfile)
	fatalLogger = log.New(os.Stdout, "\033[35m[F]\033[0m ", log.LstdFlags|log.Lshortfile)
)

type LevelEnum string

const (
	LevelDebug LevelEnum = "debug"
	LevelInfo  LevelEnum = "info"
	LevelWarn  LevelEnum = "warn"
	LevelError LevelEnum = "error"
	LevelFatal LevelEnum = "fatal"
	LevelPanic LevelEnum = "panic"
)

var LogLevel = LevelError

func (l *LevelEnum) Set(level string) error {
	switch level {
	case "debug":
		*l = LevelDebug
	case "info":
		*l = LevelInfo
	case "warn":
		*l = LevelWarn
	case "error":
		*l = LevelError
	case "fatal":
		*l = LevelFatal
	case "panic":
		*l = LevelPanic
	default:
		*l = LevelError
	}
	return nil
}

func (l *LevelEnum) String() string {
	return string(LogLevel)
}

func (l *LevelEnum) Type() string {
	return "LevelEnum"
}

func levelIsEnabled(level LevelEnum) bool {
	levelMap := map[LevelEnum]int{
		LevelDebug: 0,
		LevelInfo:  1,
		LevelWarn:  2,
		LevelError: 3,
		LevelFatal: 4,
		LevelPanic: 5,
	}
	return levelMap[level] >= levelMap[LogLevel]
}

func Debugf(format string, v ...any) {
	if levelIsEnabled(LevelDebug) {
		debugLogger.Output(2, fmt.Sprintf(format, v...))
	}
}

func Infof(format string, v ...any) {
	if levelIsEnabled(LevelInfo) {
		infoLogger.Output(2, fmt.Sprintf(format, v...))
	}
}

func Warnf(format string, v ...any) {
	if levelIsEnabled(LevelWarn) {
		warnLogger.Output(2, fmt.Sprintf(format, v...))
	}
}

func Errorf(format string, v ...any) {
	if levelIsEnabled(LevelError) {
		errorLogger.Output(2, fmt.Sprintf(format, v...))
	}
}

func Panicf(format string, v ...any) {
	if levelIsEnabled(LevelPanic) {
		panicLogger.Output(2, fmt.Sprintf(format, v...))
	}
}

func Fatalf(format string, v ...any) {
	if levelIsEnabled(LevelFatal) {
		fatalLogger.Output(2, fmt.Sprintf(format, v...))
	}
}

func Debug(v ...any) {
	if levelIsEnabled(LevelDebug) {
		debugLogger.Output(2, fmt.Sprintln(v...))
	}
}

func Info(v ...any) {
	if levelIsEnabled(LevelInfo) {
		infoLogger.Output(2, fmt.Sprintln(v...))
	}
}

func Warn(v ...any) {
	if levelIsEnabled(LevelWarn) {
		warnLogger.Output(2, fmt.Sprintln(v...))
	}
}

func Error(v ...any) {
	if levelIsEnabled(LevelError) {
		errorLogger.Output(2, fmt.Sprintln(v...))
	}
}

func Panic(v ...any) {
	if levelIsEnabled(LevelPanic) {
		panicLogger.Output(2, fmt.Sprintln(v...))
	}
}

func Fatal(v ...any) {
	if levelIsEnabled(LevelFatal) {
		fatalLogger.Output(2, fmt.Sprintln(v...))
	}
}
