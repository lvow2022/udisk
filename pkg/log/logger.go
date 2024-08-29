package log

import (
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

// BasicLogger is the logger (wrapper for logrus)
type basicLogger struct {
	*logrus.Logger
}

// Level is the log level of logger (wrapper for logrus)
type Level logrus.Level

// Formatter is the formatter of logger (wrapper for logrus)
type Formatter logrus.Formatter

// 全局变量 logger
var Logger *basicLogger

// init 函数用于初始化全局 logger
func init() {
	Logger = &basicLogger{logrus.New()}
	Logger.Out = os.Stdout
	Logger.Level = logrus.DebugLevel
	Logger.Formatter = &logrus.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	}
}

// SetOutput sets the logger output.
func SetOutput(out io.Writer) {
	Logger.Out = out
}

// SetFormatter sets the logger formatter.
func SetFormatter(formatter Formatter) {
	Logger.Formatter = logrus.Formatter(formatter)
}

// SetLevel sets the logger level.
func SetLevel(level Level) {
	Logger.Level = logrus.Level(level)
}

// GetLevel returns the logger level.
func GetLevel() Level {
	return Level(Logger.Level)
}

var (
	PanicLevel = Level(logrus.PanicLevel)
	FatalLevel = Level(logrus.FatalLevel)
	ErrorLevel = Level(logrus.ErrorLevel)
	WarnLevel  = Level(logrus.WarnLevel)
	InfoLevel  = Level(logrus.InfoLevel)
	DebugLevel = Level(logrus.DebugLevel)

	WithError = logrus.WithError
	WithField = logrus.WithField

	Debug   = logrus.Debug
	Print   = logrus.Print
	Info    = logrus.Info
	Warn    = logrus.Warn
	Warning = logrus.Warning
	Error   = logrus.Error
	Panic   = logrus.Panic
	Fatal   = logrus.Fatal

	Debugf   = logrus.Debugf
	Printf   = logrus.Printf
	Infof    = logrus.Infof
	Warnf    = logrus.Warnf
	Warningf = logrus.Warningf
	Errorf   = logrus.Errorf
	Panicf   = logrus.Panicf
	Fatalf   = logrus.Fatalf
)
