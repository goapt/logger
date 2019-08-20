package logger

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

var _ ILogger = (*LogrusLogger)(nil)

// LogrusLogger file logger
type LogrusLogger struct {
	*logrus.Logger
	Fingerprint bool
	fields      map[string]interface{}
}

// NewFileLogger providers a file logger based on logrus
func NewLogrusLogger(option func(l *LogrusLogger)) ILogger {
	l := &LogrusLogger{
		Logger: &logrus.Logger{
			Out: os.Stderr,
			Formatter: &logrus.JSONFormatter{
				TimestampFormat: "2006-01-02 15:04:05",
			},
			Hooks: make(logrus.LevelHooks),
		},
		fields: make(map[string]interface{}),
	}
	option(l)
	return l
}

func (l *LogrusLogger) withFinger(format string) IBaseLogger {
	if l.Fingerprint {
		l.fields["fingerprint"] = []string{format}

		return l.Logger.WithFields(l.fields)
	}

	return l.Logger
}

func (l *LogrusLogger) WithFields(fields map[string]interface{}) ILogger {
	return &LogrusLogger{
		Logger:      l.Logger,
		Fingerprint: l.Fingerprint,
		fields:      fields,
	}
}

func (l *LogrusLogger) Data(v interface{}) ILogger {
	return l.WithFields(map[string]interface{}{
		"data": v,
	})
}

func (l *LogrusLogger) AddHook(hook logrus.Hook) {
	l.Logger.AddHook(hook)
}

func (l *LogrusLogger) Debugf(format string, args ...interface{}) {
	l.withFinger(format).Debugf(format, args...)
}

func (l *LogrusLogger) Infof(format string, args ...interface{}) {
	l.withFinger(format).Infof(format, args...)
}

func (l *LogrusLogger) Warnf(format string, args ...interface{}) {
	l.Logger.WithFields(l.fields).Warnf(format, args...)
}

func (l *LogrusLogger) Errorf(format string, args ...interface{}) {
	l.withFinger(format).Errorf(format, args...)
}

func (l *LogrusLogger) Fatalf(format string, args ...interface{}) {
	l.withFinger(format).Fatalf(format, args...)
}

func (l *LogrusLogger) Tracef(format string, args ...interface{}) {
	l.withFinger(format).Tracef(format, args...)
}

func (l *LogrusLogger) Debug(args ...interface{}) {
	l.withFinger(argsFormat(args...)).Debug(args...)
}

func (l *LogrusLogger) Info(args ...interface{}) {
	l.withFinger(argsFormat(args...)).Info(args...)
}

func (l *LogrusLogger) Warn(args ...interface{}) {
	l.withFinger(argsFormat(args...)).Warn(args...)
}

func (l *LogrusLogger) Error(args ...interface{}) {
	l.withFinger(argsFormat(args...)).Error(args...)
}

func (l *LogrusLogger) Fatal(args ...interface{}) {
	l.withFinger(argsFormat(args...)).Fatal(args...)
}

func (l *LogrusLogger) Trace(args ...interface{}) {
	l.withFinger(argsFormat(args...)).Trace(args...)
}

func argsFormat(args ...interface{}) string {
	format := ""
	if len(args) > 0 {
		format = fmt.Sprint(args[0])
	}

	return format
}
